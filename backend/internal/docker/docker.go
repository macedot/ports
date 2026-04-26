package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// ContainerInfo holds relevant container data
type ContainerInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Image       string        `json:"image"`
	Status      string        `json:"status"`
	State       string        `json:"state"`
	NetworkMode string        `json:"network_mode"`
	PID         int           `json:"pid"`
	Ports       []PortMapping `json:"ports"`
}

// PortMapping represents a host-to-container port mapping
type PortMapping struct {
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// dockerContainer is the raw API response type
type dockerContainer struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Ports   []dockerPort `json:"Ports"`
	HostConfig struct {
		NetworkMode string `json:"NetworkMode"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

type dockerPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort,omitempty"`
	Type        string `json:"Type"`
}

// Collector fetches container data from the Docker Engine API
type Collector struct {
	client *http.Client
	host   string // e.g., "unix:///var/run/docker.sock"
}

// NewCollector creates a Docker collector. socketPath is the DOCKER_HOST value.
// If socketPath is empty, returns nil (Docker monitoring disabled).
func NewCollector(socketPath string) (*Collector, error) {
	if socketPath == "" {
		return nil, nil
	}

	// Parse the socket path
	socketFile := socketPath
	if strings.HasPrefix(socketPath, "unix://") {
		socketFile = strings.TrimPrefix(socketPath, "unix://")
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketFile)
			},
		},
		Timeout: 5 * time.Second,
	}

	return &Collector{
		client: client,
		host:   "http://localhost",
	}, nil
}

// inspectContainerPID fetches the PID for a single container via inspect API.
func (c *Collector) inspectContainerPID(ctx context.Context, containerID string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.host+"/containers/"+containerID+"/json", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create inspect request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("inspect returned status %d for %s", resp.StatusCode, containerID)
	}

	var inspectResp struct {
		State struct {
			Pid int `json:"Pid"`
		} `json:"State"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&inspectResp); err != nil {
		return 0, fmt.Errorf("failed to decode inspect response: %w", err)
	}

	return inspectResp.State.Pid, nil
}

// Collect fetches the list of running containers
func (c *Collector) Collect(ctx context.Context) ([]ContainerInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.host+"/containers/json?all=false", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query docker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api returned status %d", resp.StatusCode)
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to decode docker response: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		info := ContainerInfo{
			ID:          truncateID(c.ID),
			Name:        containerName(c.Names),
			Image:       c.Image,
			Status:      c.Status,
			State:       c.State,
			NetworkMode: c.HostConfig.NetworkMode,
			PID:         0, // PID fetched via inspect below
			Ports:       extractPorts(c),
		}
		result = append(result, info)
	}

	// Fetch PIDs via inspect (not available in /containers/json)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // max 10 concurrent inspect requests

	for i := range result {
		i := i
		g.Go(func() error {
			pid, err := c.inspectContainerPID(gctx, containers[i].ID)
			if err != nil {
				// Non-fatal: container may have exited between list and inspect
				return nil
			}
			result[i].PID = pid
			return nil
		})
	}
	_ = g.Wait() // errors are silently skipped (container may have exited)

	return result, nil
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func extractPorts(c dockerContainer) []PortMapping {
	var ports []PortMapping

	// From the top-level Ports array (published ports for bridge containers)
	seen := make(map[string]bool)
	for _, p := range c.Ports {
		if p.PublicPort > 0 {
			key := fmt.Sprintf("%d/%s", p.PublicPort, p.Type)
			if !seen[key] {
				ports = append(ports, PortMapping{
					HostIP:        p.IP,
					HostPort:      p.PublicPort,
					ContainerPort: p.PrivatePort,
					Protocol:      p.Type,
				})
				seen[key] = true
			}
		}
	}

	// From NetworkSettings.Ports (more detailed binding info)
	for portKey, bindings := range c.NetworkSettings.Ports {
		// portKey format: "8080/tcp"
		parts := strings.SplitN(portKey, "/", 2)
		if len(parts) != 2 {
			continue
		}
		containerPort := 0
		fmt.Sscanf(parts[0], "%d", &containerPort)
		proto := parts[1]

		for _, b := range bindings {
			hostPort := 0
			fmt.Sscanf(b.HostPort, "%d", &hostPort)
			key := fmt.Sprintf("%d/%s", hostPort, proto)
			if !seen[key] && hostPort > 0 {
				ports = append(ports, PortMapping{
					HostIP:        b.HostIP,
					HostPort:      hostPort,
					ContainerPort: containerPort,
					Protocol:      proto,
				})
				seen[key] = true
			}
		}
	}

	return ports
}
