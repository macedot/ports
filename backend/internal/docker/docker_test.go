package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCollector_EmptyPath(t *testing.T) {
	c, err := NewCollector("")
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatal("expected nil collector for empty path")
	}
}

func TestCollect_BasicContainers(t *testing.T) {
	// Create unix socket pair for testing
	// Use a real HTTP server and connect via unix socket
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle list endpoint first
		if r.URL.Path == "/containers/json" {
			if r.URL.Query().Get("all") != "false" {
				t.Errorf("expected all=false")
			}

			mockResp := []dockerContainer{
				{
					ID:     "abc123def456789",
					Names:  []string{"/my-nginx"},
					Image:  "nginx:latest",
					State:  "running",
					Status: "Up 2 hours",
					HostConfig: struct {
						NetworkMode string `json:"NetworkMode"`
					}{NetworkMode: "bridge"},
					Ports: []dockerPort{
						{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
					},
				},
				{
					ID:     "def456abc789012",
					Names:  []string{"/my-app"},
					Image:  "myapp:v1",
					State:  "running",
					Status: "Up 5 minutes",
					HostConfig: struct {
						NetworkMode string `json:"NetworkMode"`
					}{NetworkMode: "host"},
				},
			}
			payload, _ := json.Marshal(mockResp)
			w.Write(payload)
			return
		}

		// Handle inspect endpoint: /containers/{id}/json
		if strings.HasPrefix(r.URL.Path, "/containers/") && strings.HasSuffix(r.URL.Path, "/json") {
			// Return mock PID based on container ID
			pid := 42
			fmt.Fprintf(w, `{"State":{"Pid":%d}}`, pid)
			return
		}

		t.Errorf("unexpected path: %s", r.URL.Path)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Create collector using the test server URL
	collector := &Collector{
		client: server.Client(),
		host:   server.URL,
	}

	containers, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	// Verify first container
	c1 := containers[0]
	if c1.ID != "abc123def456" {
		t.Errorf("expected ID 'abc123def456', got '%s'", c1.ID)
	}
	if c1.Name != "my-nginx" {
		t.Errorf("expected Name 'my-nginx', got '%s'", c1.Name)
	}
	if c1.Image != "nginx:latest" {
		t.Errorf("expected Image 'nginx:latest', got '%s'", c1.Image)
	}
	if c1.NetworkMode != "bridge" {
		t.Errorf("expected NetworkMode 'bridge', got '%s'", c1.NetworkMode)
	}
	if c1.PID != 42 {
		t.Errorf("expected PID 42, got %d", c1.PID)
	}
	if len(c1.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(c1.Ports))
	}
	if c1.Ports[0].HostPort != 8080 {
		t.Errorf("expected HostPort 8080, got %d", c1.Ports[0].HostPort)
	}
	if c1.Ports[0].ContainerPort != 80 {
		t.Errorf("expected ContainerPort 80, got %d", c1.Ports[0].ContainerPort)
	}

	// Verify second container (host network)
	c2 := containers[1]
	if c2.NetworkMode != "host" {
		t.Errorf("expected NetworkMode 'host', got '%s'", c2.NetworkMode)
	}
	if c2.PID != 42 {
		t.Errorf("expected PID 42, got %d", c2.PID)
	}
	if len(c2.Ports) != 0 {
		t.Errorf("expected 0 ports for host network container, got %d", len(c2.Ports))
	}
}

func TestCollect_Unavailable(t *testing.T) {
	// Try connecting to a non-existent socket
	collector, _ := NewCollector("unix:///nonexistent/docker.sock")
	_, err := collector.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error for unavailable socket")
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123def456789", "abc123def456"},
		{"short", "short"},
		{"", ""},
	}
	for _, tt := range tests {
		got := truncateID(tt.input)
		if got != tt.expected {
			t.Errorf("truncateID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestContainerName(t *testing.T) {
	if got := containerName([]string{"/my-container"}); got != "my-container" {
		t.Errorf("expected 'my-container', got '%s'", got)
	}
	if got := containerName(nil); got != "" {
		t.Errorf("expected empty, got '%s'", got)
	}
}
