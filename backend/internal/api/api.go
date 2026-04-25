package api

import (
	"encoding/json"
	"net/http"
	"time"

	"listen-ports/internal/cache"
	"listen-ports/internal/mapper"
	"listen-ports/internal/parser"
)

type Handler struct {
	cache *cache.Cache
}

func NewHandler(cache *cache.Cache) *Handler {
	return &Handler{cache: cache}
}

type SocketResponse struct {
	Protocol   string `json:"protocol"`
	LocalAddr  string `json:"local_addr"`
	LocalPort  int    `json:"local_port"`
	RemoteAddr string `json:"remote_addr"`
	RemotePort int    `json:"remote_port"`
	State      string `json:"state"`
	Process    string `json:"process"`
}

type SocketsResponse struct {
	Sockets   []SocketResponse `json:"sockets"`
	UpdatedAt string           `json:"updated_at"`
	Count     int              `json:"count"`
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

	processMap, err := mapper.BuildProcessMap()
	if err != nil {
		http.Error(w, "Failed to build process map", http.StatusInternalServerError)
		return
	}

	sockets := filterAndEnrichSockets(data, processMap, proto, ipver)

	response := SocketsResponse{
		Sockets:   sockets,
		UpdatedAt: updatedAt.Format(time.RFC3339),
		Count:     len(sockets),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

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

func filterAndEnrichSockets(data []parser.SocketEntry, processMap map[uint64]mapper.ProcessInfo, proto, ipver string) []SocketResponse {
	sockets := make([]SocketResponse, 0, len(data))

	for _, entry := range data {
		if !filterByProto(entry.Protocol, proto) {
			continue
		}
		if !filterByIPVer(entry.Protocol, ipver) {
			continue
		}

		processName := ""
		if procInfo, ok := processMap[entry.Inode]; ok {
			processName = procInfo.Name
		}

		sockets = append(sockets, SocketResponse{
			Protocol:   entry.Protocol,
			LocalAddr:  entry.LocalAddr,
			LocalPort:  entry.LocalPort,
			RemoteAddr: entry.RemoteAddr,
			RemotePort: entry.RemotePort,
			State:      entry.State,
			Process:    processName,
		})
	}

	return sockets
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
