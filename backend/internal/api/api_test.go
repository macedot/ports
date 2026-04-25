package api

import (
	"encoding/json"
	"testing"

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

	result := filterAndEnrichSockets(data, processMap, "tcp", "both")

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

	result := filterAndEnrichSockets(data, processMap, "udp", "both")

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

	result := filterAndEnrichSockets(data, processMap, "both", "4")

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

	result := filterAndEnrichSockets(data, processMap, "both", "6")

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

	result := filterAndEnrichSockets(data, processMap, "tcp", "6")

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

	result := filterAndEnrichSockets(data, processMap, "both", "4")

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

	result := filterAndEnrichSockets(data, processMap, "both", "4")

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

	result := filterAndEnrichSockets(data, processMap, "both", "both")

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

	result := filterAndEnrichSockets(data, processMap, "both", "both")

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
