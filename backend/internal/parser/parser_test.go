package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHexPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"port 53 (DNS)", "0035", 53, false},
		{"port 8080", "1F90", 8080, false},
		{"port 80", "0050", 80, false},
		{"port 443", "01BB", 443, false},
		{"port 22 (SSH)", "0016", 22, false},
		{"port 0", "0000", 0, false},
		{"port 65535 (max)", "FFFF", 65535, false},
		{"empty string", "", 0, true},
		{"invalid hex", "GGGG", 0, true},
		{"too long", "12345", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHexPort(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseHexPort(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseHexPort(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseHexPort(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseIPv4Addr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"localhost 127.0.0.1", "0100007F", "127.0.0.1", false},
		{"all zeros 0.0.0.0", "00000000", "0.0.0.0", false},
		{"example addr", "6407A8C0", "192.168.7.100", false},
		{"loopback reversed", "7F000001", "1.0.0.127", false},
		{"port 8080 addr", "C0A80164", "100.1.168.192", false},
		{"empty string", "", "", true},
		{"too short (6 chars)", "010000", "", true},
		{"too long (10 chars)", "0100007F00", "", true},
		{"invalid hex chars", "GGGGGGGG", "", true},
		{"odd length", "0100007", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPv4Addr(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseIPv4Addr(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseIPv4Addr(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseIPv4Addr(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseIPv6Addr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"IPv6 localhost (::1)", "00000000000000000000000001000000", "0:0:0:0:0:0:0:1", false},
		{"IPv6 all zeros (::)", "00000000000000000000000000000000", "0:0:0:0:0:0:0:0", false},
		{"empty string", "", "", true},
		{"too short (30 chars)", "000000000000000000000000010000", "", true},
		{"too long (34 chars)", "0000000000000000000000000100000011", "", true},
		{"invalid hex chars", "GGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPv6Addr(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseIPv6Addr(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseIPv6Addr(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseIPv6Addr(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseInode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
		hasError bool
	}{
		{"simple inode", "12345", 12345, false},
		{"zero inode", "0", 0, false},
		{"large inode", "9999999999", 9999999999, false},
		{"empty string", "", 0, true},
		{"non-numeric", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInode(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseInode(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseInode(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseInode(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// Entry parsing functions require >= 12 fields to match actual /proc/net format
// Fields: sl, local_address, rem_address, st, tx_queue (2 fields), rx_queue (2 fields), tr (2 fields), retrnsmt, uid, timeout, inode

func TestParseTCPEntry(t *testing.T) {
	tests := []struct {
		name          string
		fields        []string
		expectedLocal string
		expectedLPort int
		expectedState string
		hasError      bool
	}{
		{
			name:   "localhost:22 ESTABLISHED",
			fields: []string{"1", "0100007F:0016", "00000000:0000", "01", "0", "0", "0", "0", "0", "0", "0", "12345"},
			expectedLocal: "127.0.0.1",
			expectedLPort: 22,
			expectedState: "ESTABLISHED",
			hasError:      false,
		},
		{
			name:   "0.0.0.0:80 LISTEN",
			fields: []string{"2", "00000000:0050", "00000000:0000", "0A", "0", "0", "0", "0", "0", "0", "0", "54321"},
			expectedLocal: "0.0.0.0",
			expectedLPort: 80,
			expectedState: "LISTEN",
			hasError:      false,
		},
		{
			name:   "0.0.0.0:53 TIME_WAIT",
			fields: []string{"3", "00000000:0035", "00000000:0000", "06", "0", "0", "0", "0", "0", "0", "0", "111"},
			expectedLocal: "0.0.0.0",
			expectedLPort: 53,
			expectedState: "TIME_WAIT",
			hasError:      false,
		},
		{
			name:       "insufficient fields",
			fields:     []string{"1", "0100007F:0016", "00000000:0000", "01"},
			hasError:   true,
		},
		{
			name:   "unknown state maps to UNKNOWN",
			fields: []string{"1", "0100007F:0016", "00000000:0000", "FF", "0", "0", "0", "0", "0", "0", "0", "12345"},
			expectedLocal: "127.0.0.1",
			expectedLPort: 22,
			expectedState: "UNKNOWN",
			hasError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseTCPEntry(tt.fields)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseTCPEntry(%v) expected error, got nil", tt.fields)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTCPEntry(%v) unexpected error: %v", tt.fields, err)
				return
			}
			if entry.LocalAddr != tt.expectedLocal {
				t.Errorf("parseTCPEntry LocalAddr = %q, want %q", entry.LocalAddr, tt.expectedLocal)
			}
			if entry.LocalPort != tt.expectedLPort {
				t.Errorf("parseTCPEntry LocalPort = %d, want %d", entry.LocalPort, tt.expectedLPort)
			}
			if entry.State != tt.expectedState {
				t.Errorf("parseTCPEntry State = %q, want %q", entry.State, tt.expectedState)
			}
			if entry.Protocol != "TCP" {
				t.Errorf("parseTCPEntry Protocol = %q, want %q", entry.Protocol, "TCP")
			}
		})
	}
}

func TestParseUDPEntry(t *testing.T) {
	tests := []struct {
		name          string
		fields        []string
		expectedLocal string
		expectedLPort int
		hasError      bool
	}{
		{
			name:   "DNS server 0.0.0.0:53",
			fields: []string{"1", "00000000:0035", "00000000:0000", "07", "0", "0", "0", "0", "0", "0", "0", "22222"},
			expectedLocal: "0.0.0.0",
			expectedLPort: 53,
			hasError:      false,
		},
		{
			name:   "connected UDP 127.0.0.1:8080",
			fields: []string{"2", "0100007F:1F90", "00000000:0035", "07", "0", "0", "0", "0", "0", "0", "0", "33333"},
			expectedLocal: "127.0.0.1",
			expectedLPort: 8080,
			hasError:      false,
		},
		{
			name:       "insufficient fields",
			fields:     []string{"1", "00000000:0035"},
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseUDPEntry(tt.fields)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseUDPEntry(%v) expected error, got nil", tt.fields)
				}
				return
			}
			if err != nil {
				t.Errorf("parseUDPEntry(%v) unexpected error: %v", tt.fields, err)
				return
			}
			if entry.LocalAddr != tt.expectedLocal {
				t.Errorf("parseUDPEntry LocalAddr = %q, want %q", entry.LocalAddr, tt.expectedLocal)
			}
			if entry.LocalPort != tt.expectedLPort {
				t.Errorf("parseUDPEntry LocalPort = %d, want %d", entry.LocalPort, tt.expectedLPort)
			}
			if entry.State != "UNCONN" {
				t.Errorf("parseUDPEntry State = %q, want %q", entry.State, "UNCONN")
			}
			if entry.Protocol != "UDP" {
				t.Errorf("parseUDPEntry Protocol = %q, want %q", entry.Protocol, "UDP")
			}
		})
	}
}

func TestParseTCP6Entry(t *testing.T) {
	tests := []struct {
		name          string
		fields        []string
		expectedLocal string
		expectedLPort int
		expectedState string
		hasError      bool
	}{
		{
			name:   "IPv6 localhost:22 LISTEN",
			fields: []string{"1", "00000000000000000000000001000000:0016", "00000000000000000000000000000000:0000", "0A", "0", "0", "0", "0", "0", "0", "0", "44444"},
			expectedLocal: "0:0:0:0:0:0:0:1",
			expectedLPort: 22,
			expectedState: "LISTEN",
			hasError:      false,
		},
		{
			name:   "IPv6 all interfaces:8080 LISTEN",
			fields: []string{"2", "00000000000000000000000000000000:1F90", "00000000000000000000000000000000:0000", "0A", "0", "0", "0", "0", "0", "0", "0", "55555"},
			expectedLocal: "0:0:0:0:0:0:0:0",
			expectedLPort: 8080,
			expectedState: "LISTEN",
			hasError:      false,
		},
		{
			name:       "insufficient fields",
			fields:     []string{"1", "00000000000000000000000001000000:0016"},
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseTCP6Entry(tt.fields)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseTCP6Entry(%v) expected error, got nil", tt.fields)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTCP6Entry(%v) unexpected error: %v", tt.fields, err)
				return
			}
			if entry.LocalAddr != tt.expectedLocal {
				t.Errorf("parseTCP6Entry LocalAddr = %q, want %q", entry.LocalAddr, tt.expectedLocal)
			}
			if entry.LocalPort != tt.expectedLPort {
				t.Errorf("parseTCP6Entry LocalPort = %d, want %d", entry.LocalPort, tt.expectedLPort)
			}
			if entry.State != tt.expectedState {
				t.Errorf("parseTCP6Entry State = %q, want %q", entry.State, tt.expectedState)
			}
			if entry.Protocol != "TCP6" {
				t.Errorf("parseTCP6Entry Protocol = %q, want %q", entry.Protocol, "TCP6")
			}
		})
	}
}

func TestParseUDP6Entry(t *testing.T) {
	tests := []struct {
		name          string
		fields        []string
		expectedLocal string
		expectedLPort int
		hasError      bool
	}{
		{
			name:   "IPv6 localhost:53",
			fields: []string{"1", "00000000000000000000000001000000:0035", "00000000000000000000000000000000:0000", "07", "0", "0", "0", "0", "0", "0", "0", "66666"},
			expectedLocal: "0:0:0:0:0:0:0:1",
			expectedLPort: 53,
			hasError:      false,
		},
		{
			name:       "insufficient fields",
			fields:     []string{"1", "00000000000000000000000001000000:0035"},
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseUDP6Entry(tt.fields)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseUDP6Entry(%v) expected error, got nil", tt.fields)
				}
				return
			}
			if err != nil {
				t.Errorf("parseUDP6Entry(%v) unexpected error: %v", tt.fields, err)
				return
			}
			if entry.LocalAddr != tt.expectedLocal {
				t.Errorf("parseUDP6Entry LocalAddr = %q, want %q", entry.LocalAddr, tt.expectedLocal)
			}
			if entry.LocalPort != tt.expectedLPort {
				t.Errorf("parseUDP6Entry LocalPort = %d, want %d", entry.LocalPort, tt.expectedLPort)
			}
			if entry.State != "UNCONN" {
				t.Errorf("parseUDP6Entry State = %q, want %q", entry.State, "UNCONN")
			}
			if entry.Protocol != "UDP6" {
				t.Errorf("parseUDP6Entry Protocol = %q, want %q", entry.Protocol, "UDP6")
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()

	// TCP file format matches actual /proc/net/tcp with space-padded columns
	tcpContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 0100007F:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000   973        0 3585 1 00000000213c086d 100 0 0 10 5                     
   1: 0100007F:B66F 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 1229750 1 000000004f68b45f 100 0 0 10 0                   
   2: 00000000:14EB 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 3570 1 0000000016d8ca46 100 0 0 10 5`

	tcpPath := filepath.Join(tmpDir, "tcp")
	if err := os.WriteFile(tcpPath, []byte(tcpContent), 0644); err != nil {
		t.Fatalf("Failed to write tcp fixture: %v", err)
	}

	t.Run("parse tcp file with fixture", func(t *testing.T) {
		entries, err := parseFile(tcpPath, parseTCPEntry)
		if err != nil {
			t.Fatalf("parseFile() error = %v", err)
		}
		if len(entries) != 3 {
			t.Fatalf("parseFile() returned %d entries, want 3", len(entries))
		}

		// Entry 0: 127.0.0.1:53, state CLOSE
		if entries[0].LocalAddr != "127.0.0.1" {
			t.Errorf("entries[0].LocalAddr = %q, want %q", entries[0].LocalAddr, "127.0.0.1")
		}
		if entries[0].LocalPort != 53 {
			t.Errorf("entries[0].LocalPort = %d, want %d", entries[0].LocalPort, 53)
		}
		if entries[0].State != "CLOSE" {
			t.Errorf("entries[0].State = %q, want %q", entries[0].State, "CLOSE")
		}

		// Entry 1: 127.0.0.1:46703 (B66F hex = 46703), state LISTEN
		if entries[1].LocalAddr != "127.0.0.1" {
			t.Errorf("entries[1].LocalAddr = %q, want %q", entries[1].LocalAddr, "127.0.0.1")
		}
		if entries[1].LocalPort != 46703 {
			t.Errorf("entries[1].LocalPort = %d, want %d", entries[1].LocalPort, 46703)
		}
		if entries[1].State != "LISTEN" {
			t.Errorf("entries[1].State = %q, want %q", entries[1].State, "LISTEN")
		}

		// Entry 2: 0.0.0.0:5355, state LISTEN
		if entries[2].LocalAddr != "0.0.0.0" {
			t.Errorf("entries[2].LocalAddr = %q, want %q", entries[2].LocalAddr, "0.0.0.0")
		}
		if entries[2].LocalPort != 5355 {
			t.Errorf("entries[2].LocalPort = %d, want %d", entries[2].LocalPort, 5355)
		}
	})

	t.Run("non-existent file returns empty", func(t *testing.T) {
		entries, err := parseFile(filepath.Join(tmpDir, "nonexistent"), parseTCPEntry)
		if err != nil {
			t.Fatalf("parseFile() error = %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("parseFile() returned %d entries, want 0", len(entries))
		}
	})

	// UDP file fixture
	udpContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 00000000:0042 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 12298 1 00000000175b78b2 100 0 0 10 0`

	udpPath := filepath.Join(tmpDir, "udp")
	if err := os.WriteFile(udpPath, []byte(udpContent), 0644); err != nil {
		t.Fatalf("Failed to write udp fixture: %v", err)
	}

	t.Run("parse udp file with fixture", func(t *testing.T) {
		entries, err := parseFile(udpPath, parseUDPEntry)
		if err != nil {
			t.Fatalf("parseFile() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("parseFile() returned %d entries, want 1", len(entries))
		}
		if entries[0].LocalPort != 66 {
			t.Errorf("entries[0].LocalPort = %d, want %d", entries[0].LocalPort, 66)
		}
		if entries[0].State != "UNCONN" {
			t.Errorf("entries[0].State = %q, want %q", entries[0].State, "UNCONN")
		}
		if entries[0].Protocol != "UDP" {
			t.Errorf("entries[0].Protocol = %q, want %q", entries[0].Protocol, "UDP")
		}
	})

	// TCP6 file fixture
	tcp6Content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 00000000000000000000000001000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 77777 1 00000000213c086d 100 0 0 10 5`

	tcp6Path := filepath.Join(tmpDir, "tcp6")
	if err := os.WriteFile(tcp6Path, []byte(tcp6Content), 0644); err != nil {
		t.Fatalf("Failed to write tcp6 fixture: %v", err)
	}

	t.Run("parse tcp6 file with fixture", func(t *testing.T) {
		entries, err := parseFile(tcp6Path, parseTCP6Entry)
		if err != nil {
			t.Fatalf("parseFile() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("parseFile() returned %d entries, want 1", len(entries))
		}
		if entries[0].LocalAddr != "0:0:0:0:0:0:0:1" {
			t.Errorf("entries[0].LocalAddr = %q, want %q", entries[0].LocalAddr, "0:0:0:0:0:0:0:1")
		}
		if entries[0].LocalPort != 8080 {
			t.Errorf("entries[0].LocalPort = %d, want %d", entries[0].LocalPort, 8080)
		}
		if entries[0].Protocol != "TCP6" {
			t.Errorf("entries[0].Protocol = %q, want %q", entries[0].Protocol, "TCP6")
		}
	})

	// UDP6 file fixture
	udp6Content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 00000000000000000000000001000000:0050 00000000000000000000000000000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 88888 1 00000000213c086d 100 0 0 10 0`

	udp6Path := filepath.Join(tmpDir, "udp6")
	if err := os.WriteFile(udp6Path, []byte(udp6Content), 0644); err != nil {
		t.Fatalf("Failed to write udp6 fixture: %v", err)
	}

	t.Run("parse udp6 file with fixture", func(t *testing.T) {
		entries, err := parseFile(udp6Path, parseUDP6Entry)
		if err != nil {
			t.Fatalf("parseFile() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("parseFile() returned %d entries, want 1", len(entries))
		}
		if entries[0].LocalAddr != "0:0:0:0:0:0:0:1" {
			t.Errorf("entries[0].LocalAddr = %q, want %q", entries[0].LocalAddr, "0:0:0:0:0:0:0:1")
		}
		if entries[0].LocalPort != 80 {
			t.Errorf("entries[0].LocalPort = %d, want %d", entries[0].LocalPort, 80)
		}
		if entries[0].Protocol != "UDP6" {
			t.Errorf("entries[0].Protocol = %q, want %q", entries[0].Protocol, "UDP6")
		}
	})
}

func TestParseAddrPort(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedAddr string
		expectedPort int
		hasError     bool
	}{
		{"localhost:22", "0100007F:0016", "127.0.0.1", 22, false},
		{"all:80", "00000000:0050", "0.0.0.0", 80, false},
		{"empty addr", ":0050", "", 0, true},
		{"empty port", "0100007F:", "", 0, true},
		{"no colon", "0100007F0016", "", 0, true},
		{"multiple colons", "0100007F:0016:00", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port, err := parseAddrPort(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseAddrPort(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseAddrPort(%q) unexpected error: %v", tt.input, err)
				return
			}
			if addr != tt.expectedAddr {
				t.Errorf("parseAddrPort(%q) addr = %q, want %q", tt.input, addr, tt.expectedAddr)
			}
			if port != tt.expectedPort {
				t.Errorf("parseAddrPort(%q) port = %d, want %d", tt.input, port, tt.expectedPort)
			}
		})
	}
}

func TestParseAddrPort6(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedAddr string
		expectedPort int
		hasError     bool
	}{
		{"localhost:22", "00000000000000000000000001000000:0016", "0:0:0:0:0:0:0:1", 22, false},
		{"all:80", "00000000000000000000000000000000:0050", "0:0:0:0:0:0:0:0", 80, false},
		{"empty addr", ":0050", "", 0, true},
		{"empty port", "00000000000000000000000001000000:", "", 0, true},
		{"no colon", "000000000000000000000000010000000016", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port, err := parseAddrPort6(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseAddrPort6(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseAddrPort6(%q) unexpected error: %v", tt.input, err)
				return
			}
			if addr != tt.expectedAddr {
				t.Errorf("parseAddrPort6(%q) addr = %q, want %q", tt.input, addr, tt.expectedAddr)
			}
			if port != tt.expectedPort {
				t.Errorf("parseAddrPort6(%q) port = %d, want %d", tt.input, port, tt.expectedPort)
			}
		})
	}
}
