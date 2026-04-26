package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"listen-ports/internal/docker"
	"listen-ports/internal/mapper"
	"listen-ports/internal/parser"
)

// Tests for exported filter functions (testable without cache mock)

func TestFilterByProto(t *testing.T) {
	tests := []struct {
		protocol string
		proto    string
		expected bool
	}{
		{"TCP", "tcp", true},
		{"TCP6", "tcp", true},
		{"UDP", "tcp", false},
		{"UDP6", "tcp", false},
		{"TCP", "udp", false},
		{"TCP6", "udp", false},
		{"UDP", "udp", true},
		{"UDP6", "udp", true},
		{"TCP", "both", true},
		{"UDP", "both", true},
		{"TCP6", "both", true},
		{"UDP6", "both", true},
	}

	for _, tt := range tests {
		result := filterByProto(tt.protocol, tt.proto)
		if result != tt.expected {
			t.Errorf("filterByProto(%s, %s) = %v, want %v", tt.protocol, tt.proto, result, tt.expected)
		}
	}
}

func TestFilterByIPVer(t *testing.T) {
	tests := []struct {
		protocol string
		ipver    string
		expected bool
	}{
		{"TCP", "4", true},
		{"UDP", "4", true},
		{"TCP6", "4", false},
		{"UDP6", "4", false},
		{"TCP", "6", false},
		{"UDP", "6", false},
		{"TCP6", "6", true},
		{"UDP6", "6", true},
		{"TCP", "both", true},
		{"UDP", "both", true},
		{"TCP6", "both", true},
		{"UDP6", "both", true},
	}

	for _, tt := range tests {
		result := filterByIPVer(tt.protocol, tt.ipver)
		if result != tt.expected {
			t.Errorf("filterByIPVer(%s, %s) = %v, want %v", tt.protocol, tt.ipver, result, tt.expected)
		}
	}
}

func TestIsValidProto(t *testing.T) {
	tests := []struct {
		proto    string
		expected bool
	}{
		{"tcp", true},
		{"udp", true},
		{"both", true},
		{"invalid", false},
		{"", false},
		{"TCP", false},
		{"UDP", false},
	}

	for _, tt := range tests {
		result := isValidProto(tt.proto)
		if result != tt.expected {
			t.Errorf("isValidProto(%s) = %v, want %v", tt.proto, result, tt.expected)
		}
	}
}

func TestIsValidIPVer(t *testing.T) {
	tests := []struct {
		ipver    string
		expected bool
	}{
		{"4", true},
		{"6", true},
		{"both", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidIPVer(tt.ipver)
		if result != tt.expected {
			t.Errorf("isValidIPVer(%s) = %v, want %v", tt.ipver, result, tt.expected)
		}
	}
}

func TestFilterAndEnrichSockets_TCPFilter(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
		{Protocol: "UDP", LocalAddr: "0.0.0.0", LocalPort: 53, State: "UNCONN", Inode: 3},
		{Protocol: "UDP6", LocalAddr: "::", LocalPort: 53, State: "UNCONN", Inode: 4},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "tcp", "both")

	if len(result) != 2 {
		t.Errorf("expected 2 sockets, got %d", len(result))
	}
	for _, s := range result {
		if s.Protocol != "TCP" && s.Protocol != "TCP6" {
			t.Errorf("expected TCP or TCP6, got %s", s.Protocol)
		}
	}
}

func TestFilterAndEnrichSockets_UDPFilter(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
		{Protocol: "UDP", LocalAddr: "0.0.0.0", LocalPort: 53, State: "UNCONN", Inode: 3},
		{Protocol: "UDP6", LocalAddr: "::", LocalPort: 53, State: "UNCONN", Inode: 4},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "udp", "both")

	if len(result) != 2 {
		t.Errorf("expected 2 sockets, got %d", len(result))
	}
	for _, s := range result {
		if s.Protocol != "UDP" && s.Protocol != "UDP6" {
			t.Errorf("expected UDP or UDP6, got %s", s.Protocol)
		}
	}
}

func TestFilterAndEnrichSockets_IPVer4Filter(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
		{Protocol: "UDP", LocalAddr: "0.0.0.0", LocalPort: 53, State: "UNCONN", Inode: 3},
		{Protocol: "UDP6", LocalAddr: "::", LocalPort: 53, State: "UNCONN", Inode: 4},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "4")

	if len(result) != 2 {
		t.Errorf("expected 2 sockets for ipver=4, got %d", len(result))
	}
	for _, s := range result {
		if s.Protocol == "TCP6" || s.Protocol == "UDP6" {
			t.Errorf("expected IPv4 protocols only, got %s", s.Protocol)
		}
	}
}

func TestFilterAndEnrichSockets_IPVer6Filter(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
		{Protocol: "UDP", LocalAddr: "0.0.0.0", LocalPort: 53, State: "UNCONN", Inode: 3},
		{Protocol: "UDP6", LocalAddr: "::", LocalPort: 53, State: "UNCONN", Inode: 4},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "6")

	if len(result) != 2 {
		t.Errorf("expected 2 sockets for ipver=6, got %d", len(result))
	}
	for _, s := range result {
		if s.Protocol == "TCP" || s.Protocol == "UDP" {
			t.Errorf("expected IPv6 protocols only, got %s", s.Protocol)
		}
	}
}

func TestFilterAndEnrichSockets_CombinedFilters(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
		{Protocol: "UDP", LocalAddr: "0.0.0.0", LocalPort: 53, State: "UNCONN", Inode: 3},
		{Protocol: "UDP6", LocalAddr: "::", LocalPort: 53, State: "UNCONN", Inode: 4},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "tcp", "6")

	if len(result) != 1 {
		t.Errorf("expected 1 socket for proto=tcp&ipver=6, got %d", len(result))
	}
	if result[0].Protocol != "TCP6" {
		t.Errorf("expected TCP6, got %s", result[0].Protocol)
	}
}

func TestFilterAndEnrichSockets_WithProcessInfo(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 999},
	}
	processMap := map[uint64]mapper.ProcessInfo{
		999: {PID: 100, Name: "sshd"},
	}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "4")

	if len(result) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result))
	}
	if result[0].Process != "sshd" {
		t.Errorf("expected process 'sshd', got '%s'", result[0].Process)
	}
}

func TestFilterAndEnrichSockets_NoProcessMatch(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 999},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "4")

	if len(result) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result))
	}
	if result[0].Process != "" {
		t.Errorf("expected empty process, got '%s'", result[0].Process)
	}
}

func TestFilterAndEnrichSockets_EmptyData(t *testing.T) {
	data := []parser.SocketEntry{}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "both")

	if len(result) != 0 {
		t.Errorf("expected 0 sockets, got %d", len(result))
	}
}

func TestFilterAndEnrichSockets_EmptyProcessMap(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN", Inode: 1},
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 22, State: "LISTEN", Inode: 2},
	}
	processMap := map[uint64]mapper.ProcessInfo{}

	result := filterAndEnrichSockets(data, processMap, nil, "both", "both")

	if len(result) != 2 {
		t.Errorf("expected 2 sockets, got %d", len(result))
	}
	for _, s := range result {
		if s.Process != "" {
			t.Errorf("expected empty process names, got '%s'", s.Process)
		}
	}
}

func TestSocketResponse_JSONFields(t *testing.T) {
	resp := SocketResponse{
		Protocol:   "TCP",
		LocalAddr:  "0.0.0.0",
		LocalPort:  22,
		RemoteAddr: "0.0.0.0",
		RemotePort: 0,
		State:      "LISTEN",
		Process:    "sshd",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal SocketResponse: %v", err)
	}

	var unmarshaled SocketResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SocketResponse: %v", err)
	}

	if unmarshaled.Protocol != "TCP" {
		t.Errorf("expected Protocol TCP, got %s", unmarshaled.Protocol)
	}
	if unmarshaled.LocalPort != 22 {
		t.Errorf("expected LocalPort 22, got %d", unmarshaled.LocalPort)
	}
}

func TestSocketsResponse_JSONFields(t *testing.T) {
	resp := SocketsResponse{
		Sockets: []SocketResponse{
			{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 22, State: "LISTEN"},
		},
		UpdatedAt: "2026-04-24T10:00:00Z",
		Count:     1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal SocketsResponse: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal SocketsResponse: %v", err)
	}

	if _, ok := unmarshaled["sockets"]; !ok {
		t.Error("response missing sockets field")
	}
	if _, ok := unmarshaled["updated_at"]; !ok {
		t.Error("response missing updated_at field")
	}
	if _, ok := unmarshaled["count"]; !ok {
		t.Error("response missing count field")
	}
}

// AuthMiddleware tests

func TestAuthMiddleware_EmptyToken_Passthrough(t *testing.T) {
	middleware := AuthMiddleware("")
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("expected next handler to be called with empty token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidBearerToken_200(t *testing.T) {
	expected := "secret-token"
	middleware := AuthMiddleware(expected)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer "+expected)
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingAuthHeader_401(t *testing.T) {
	middleware := AuthMiddleware("some-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got '%s'", resp["error"])
	}
}

func TestAuthMiddleware_WrongBearerToken_401(t *testing.T) {
	middleware := AuthMiddleware("correct-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got '%s'", resp["error"])
	}
}

func TestAuthMiddleware_NonBearerScheme_401(t *testing.T) {
	middleware := AuthMiddleware("any-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got '%s'", resp["error"])
	}
}

// AuthHandler tests

func TestAuthHandler_POST_CorrectToken_200(t *testing.T) {
	expected := "valid-token"
	handler := AuthHandler(expected)

	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer "+expected)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["valid"] != true {
		t.Errorf("expected valid=true, got %v", resp["valid"])
	}
}

func TestAuthHandler_POST_WrongToken_401(t *testing.T) {
	expected := "correct-token"
	handler := AuthHandler(expected)

	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "invalid token" {
		t.Errorf("expected error 'invalid token', got '%s'", resp["error"])
	}
}

func TestAuthHandler_POST_EmptyExpectedToken_DevMode_200(t *testing.T) {
	handler := AuthHandler("") // empty = dev mode

	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 in dev mode, got %d", w.Code)
	}

	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["valid"] != true {
		t.Errorf("expected valid=true in dev mode, got %v", resp["valid"])
	}
}

func TestAuthHandler_GET_Method_405(t *testing.T) {
	handler := AuthHandler("some-token")

	req := httptest.NewRequest(http.MethodGet, "/api/auth", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestAuthHandler_MissingAuthHeader_401(t *testing.T) {
	handler := AuthHandler("any-token")

	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestFilterAndEnrichSockets_HostNetworkContainerByPID(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 8080, State: "LISTEN", Inode: 999},
	}
	processMap := map[uint64]mapper.ProcessInfo{
		999: {PID: 1234, Name: "nginx"},
	}
	containers := []docker.ContainerInfo{
		{ID: "abc123", Name: "nginx", Image: "nginx:alpine", NetworkMode: "host", PID: 1234},
	}

	result := filterAndEnrichSockets(data, processMap, containers, "both", "both")

	if len(result) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result))
	}
	if result[0].Container == nil || *result[0].Container != "nginx" {
		t.Errorf("expected container 'nginx', got %v", result[0].Container)
	}
	if result[0].CImage == nil || *result[0].CImage != "nginx:alpine" {
		t.Errorf("expected c_image 'nginx:alpine', got %v", result[0].CImage)
	}
	if result[0].CNetwork == nil || *result[0].CNetwork != "host" {
		t.Errorf("expected c_network 'host', got %v", result[0].CNetwork)
	}
}

func TestFilterAndEnrichSockets_BridgeContainerByPort(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 8080, State: "LISTEN", Inode: 1},
		{Protocol: "TCP", LocalAddr: "10.0.0.1", LocalPort: 8080, State: "ESTABLISHED", RemoteAddr: "192.168.1.1", RemotePort: 54321, Inode: 2},
	}
	processMap := map[uint64]mapper.ProcessInfo{}
	containers := []docker.ContainerInfo{
		{ID: "def456", Name: "webapp", Image: "webapp:latest", NetworkMode: "bridge", Ports: []docker.PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		}},
	}

	result := filterAndEnrichSockets(data, processMap, containers, "both", "both")

	if len(result) != 2 {
		t.Fatalf("expected 2 sockets, got %d", len(result))
	}
	for i, s := range result {
		if s.Container == nil || *s.Container != "webapp" {
			t.Errorf("socket %d: expected container 'webapp', got %v", i, s.Container)
		}
	}
}

func TestFilterAndEnrichSockets_NoContainerMatch(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP", LocalAddr: "0.0.0.0", LocalPort: 9999, State: "LISTEN", Inode: 1},
	}
	processMap := map[uint64]mapper.ProcessInfo{}
	containers := []docker.ContainerInfo{
		{ID: "xyz789", Name: "other", Image: "other:latest", NetworkMode: "bridge", Ports: []docker.PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		}},
	}

	result := filterAndEnrichSockets(data, processMap, containers, "both", "both")

	if len(result) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result))
	}
	if result[0].Container != nil {
		t.Errorf("expected nil container, got %v", *result[0].Container)
	}
}

func TestFilterAndEnrichSockets_BridgeContainerIPv6(t *testing.T) {
	data := []parser.SocketEntry{
		{Protocol: "TCP6", LocalAddr: "::", LocalPort: 8080, State: "LISTEN", Inode: 1},
	}
	processMap := map[uint64]mapper.ProcessInfo{}
	containers := []docker.ContainerInfo{
		{ID: "ghi789", Name: "webapp", Image: "webapp:latest", NetworkMode: "bridge", Ports: []docker.PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		}},
	}

	result := filterAndEnrichSockets(data, processMap, containers, "both", "both")

	if len(result) != 1 {
		t.Fatalf("expected 1 socket, got %d", len(result))
	}
	if result[0].Container == nil || *result[0].Container != "webapp" {
		t.Errorf("expected container 'webapp' for IPv6, got %v", result[0].Container)
	}
}
