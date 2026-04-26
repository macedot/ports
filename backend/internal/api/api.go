package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"listen-ports/internal/cache"
	"listen-ports/internal/docker"
	"listen-ports/internal/mapper"
	"listen-ports/internal/parser"
)

// AuthMiddleware validates Bearer token for protected routes.
func AuthMiddleware(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			const bearerPrefix = "Bearer "
			if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			token := authHeader[len(bearerPrefix):]
			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthHandler validates token via POST /api/auth.
func AuthHandler(expectedToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path != "/api/auth" {
			http.NotFound(w, r)
			return
		}

		if expectedToken == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"valid": true}`))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB limit for auth payload

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
			return
		}

		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
			return
		}

		token := authHeader[len(bearerPrefix):]
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"valid": true}`))
	})
}

type Handler struct {
	cache       *cache.Cache
	dockerCache *docker.Cache
	procPath    string
}

func NewHandler(cache *cache.Cache, procPath string, dockerCache *docker.Cache) *Handler {
	return &Handler{cache: cache, procPath: procPath, dockerCache: dockerCache}
}

type SocketResponse struct {
	Protocol   string  `json:"protocol"`
	LocalAddr  string  `json:"local_addr"`
	LocalPort  int     `json:"local_port"`
	RemoteAddr string  `json:"remote_addr"`
	RemotePort int     `json:"remote_port"`
	State      string  `json:"state"`
	Process    string  `json:"process"`
	Container  *string `json:"container,omitempty"`
	CImage     *string `json:"c_image,omitempty"`
	CNetwork   *string `json:"c_network,omitempty"`
}

type SocketsResponse struct {
	Sockets     []SocketResponse `json:"sockets"`
	UpdatedAt   string           `json:"updated_at"`
	Count       int              `json:"count"`
	DockerError string           `json:"docker_error,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/api/sockets" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	proto := r.URL.Query().Get("proto")
	if proto == "" {
		proto = "both"
	}
	ipver := r.URL.Query().Get("ipver")
	if ipver == "" {
		ipver = "both"
	}

	if !isValidProto(proto) || !isValidIPVer(ipver) {
		http.Error(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	data, updatedAt, err := h.cache.Get()
	if err != nil {
		http.Error(w, "Failed to retrieve socket data", http.StatusInternalServerError)
		return
	}

	processMap, err := mapper.BuildProcessMap(h.procPath)
	if err != nil {
		http.Error(w, "Failed to build process map", http.StatusInternalServerError)
		return
	}

	// Get container data (nil if Docker unavailable)
	var containers []docker.ContainerInfo
	var dockerErrStr string
	if h.dockerCache != nil {
		var dockerErr error
		containers, dockerErr = h.dockerCache.Get(r.Context())
		if dockerErr != nil {
			log.Printf("Docker container fetch failed: %v", dockerErr)
			dockerErrStr = dockerErr.Error()
		}
	}

	sockets := filterAndEnrichSockets(data, processMap, containers, proto, ipver)

	response := SocketsResponse{
		Sockets:     sockets,
		UpdatedAt:   updatedAt.Format(time.RFC3339),
		Count:       len(sockets),
		DockerError: dockerErrStr,
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(response); err != nil {
		return
	}
}

func isValidProto(p string) bool {
	return p == "tcp" || p == "udp" || p == "both"
}

func isValidIPVer(v string) bool {
	return v == "4" || v == "6" || v == "both"
}

func filterAndEnrichSockets(data []parser.SocketEntry, processMap map[uint64]mapper.ProcessInfo, containers []docker.ContainerInfo, proto, ipver string) []SocketResponse {
	// Build container PID map (host-network containers)
	containerPIDMap := make(map[int]*docker.ContainerInfo)
	for i := range containers {
		c := &containers[i]
		if c.NetworkMode == "host" && c.PID > 0 {
			containerPIDMap[c.PID] = c
		}
	}

	// Build container port map (bridge-network containers)
	containerPortMap := make(map[portKey]*docker.ContainerInfo)
	for i := range containers {
		c := &containers[i]
		for _, pm := range c.Ports {
			// Map protocol: "tcp" -> "TCP", "udp" -> "UDP"
			protoUpper := strings.ToUpper(pm.Protocol)
			containerPortMap[portKey{port: pm.HostPort, proto: protoUpper}] = c
			// Also insert IPv6 variant for matching TCP6/UDP6 socket entries
			containerPortMap[portKey{port: pm.HostPort, proto: protoUpper + "6"}] = c
		}
	}

	sockets := make([]SocketResponse, 0, len(data))

	for _, entry := range data {
		if !filterByProto(entry.Protocol, proto) {
			continue
		}
		if !filterByIPVer(entry.Protocol, ipver) {
			continue
		}

		processName := ""
		var containerName, cImage, cNetwork *string
		if procInfo, ok := processMap[entry.Inode]; ok {
			if procInfo.Name != "" {
				processName = procInfo.Name
			} else if procInfo.PID > 0 {
				processName = fmt.Sprintf("pid:%d", procInfo.PID)
			}

			// Match container by PID (host-network)
			if cInfo, ok := containerPIDMap[procInfo.PID]; ok {
				name := cInfo.Name
				img := cInfo.Image
				net := cInfo.NetworkMode
				containerName = &name
				cImage = &img
				cNetwork = &net
			}
		}

		// Match container by port binding (bridge-network containers)
		if containerName == nil {
			// Try matching port with exact protocol (TCP, UDP, etc.)
			key := portKey{port: entry.LocalPort, proto: entry.Protocol}
			if cInfo, ok := containerPortMap[key]; ok {
				name := cInfo.Name
				img := cInfo.Image
				net := cInfo.NetworkMode
				containerName = &name
				cImage = &img
				cNetwork = &net
			}
		}

		sockets = append(sockets, SocketResponse{
			Protocol:   entry.Protocol,
			LocalAddr:  entry.LocalAddr,
			LocalPort:  entry.LocalPort,
			RemoteAddr: entry.RemoteAddr,
			RemotePort: entry.RemotePort,
			State:      entry.State,
			Process:    processName,
			Container:  containerName,
			CImage:     cImage,
			CNetwork:   cNetwork,
		})
	}

	return sockets
}

type portKey struct {
	port  int
	proto string // "TCP", "UDP", "TCP6", "UDP6"
}

func filterByProto(protocol, proto string) bool {
	switch proto {
	case "tcp":
		return protocol == "TCP" || protocol == "TCP6"
	case "udp":
		return protocol == "UDP" || protocol == "UDP6"
	case "both":
		return true
	default:
		return false
	}
}

func filterByIPVer(protocol, ipver string) bool {
	switch ipver {
	case "4":
		return protocol == "TCP" || protocol == "UDP"
	case "6":
		return protocol == "TCP6" || protocol == "UDP6"
	case "both":
		return true
	default:
		return false
	}
}

// ContainersHandler returns a handler for GET /api/containers.
func ContainersHandler(dockerCache *docker.Cache) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/containers" {
			http.NotFound(w, r)
			return
		}

		var containers []docker.ContainerInfo
		if dockerCache != nil {
			var err error
			containers, err = dockerCache.Get(r.Context())
			if err != nil {
				http.Error(w, "Failed to retrieve container data", http.StatusInternalServerError)
				return
			}
		}

		if containers == nil {
			containers = []docker.ContainerInfo{}
		}

		response := struct {
			Containers []docker.ContainerInfo `json:"containers"`
		}{
			Containers: containers,
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(response); err != nil {
			return
		}
	})
}
