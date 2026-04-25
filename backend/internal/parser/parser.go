package parser

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SocketEntry struct {
	Protocol   string
	LocalAddr  string
	LocalPort  int
	RemoteAddr string
	RemotePort int
	State      string
	Inode      uint64
}

func parseHexPort(hexPort string) (int, error) {
	v, err := strconv.ParseUint(hexPort, 16, 16)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func parseIPv4Addr(hex string) (string, error) {
	if len(hex) != 8 {
		return "", fmt.Errorf("invalid IPv4 hex length %d", len(hex))
	}
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return "", err
		}
		b[i] = byte(v)
	}
	return fmt.Sprintf("%d.%d.%d.%d", b[3], b[2], b[1], b[0]), nil
}

func parseIPv6Addr(hex string) (string, error) {
	if len(hex) != 32 {
		return "", fmt.Errorf("invalid IPv6 hex length %d", len(hex))
	}
	groups := make([]string, 8)
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(hex[i*8:i*8+8], 16, 32)
		if err != nil {
			return "", err
		}
		// Byte-swap the 32-bit word
		v = ((v>>24)&0xFF) | ((v>>8)&0xFF00) | ((v<<8)&0xFF0000) | ((v<<24)&0xFF000000)
		groups[i*2] = fmt.Sprintf("%x", (v>>16)&0xFFFF)
		groups[i*2+1] = fmt.Sprintf("%x", v&0xFFFF)
	}
	return strings.Join(groups, ":"), nil
}

var tcpStates = map[string]string{
	"0A": "LISTEN",
	"01": "ESTABLISHED",
	"06": "TIME_WAIT",
	"02": "SYN_SENT",
	"03": "SYN_RECV",
	"04": "FIN_WAIT1",
	"05": "FIN_WAIT2",
	"07": "CLOSE",
	"08": "CLOSE_WAIT",
	"09": "LAST_ACK",
	"0B": "CLOSING",
}

func parseAddrPort(addrPort string) (addr string, port int, err error) {
	parts := strings.Split(addrPort, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid addr:port format %q", addrPort)
	}
	addr, err = parseIPv4Addr(parts[0])
	if err != nil {
		return "", 0, err
	}
	port, err = parseHexPort(parts[1])
	if err != nil {
		return "", 0, err
	}
	return addr, port, nil
}

func parseAddrPort6(addrPort string) (addr string, port int, err error) {
	parts := strings.Split(addrPort, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid addr:port format %q", addrPort)
	}
	addr, err = parseIPv6Addr(parts[0])
	if err != nil {
		return "", 0, err
	}
	port, err = parseHexPort(parts[1])
	if err != nil {
		return "", 0, err
	}
	return addr, port, nil
}

func parseInode(inodeStr string) (uint64, error) {
	return strconv.ParseUint(inodeStr, 10, 64)
}

func parseTCPEntry(fields []string) (SocketEntry, error) {
	if len(fields) < 12 {
		return SocketEntry{}, fmt.Errorf("expected at least 12 fields, got %d", len(fields))
	}

	localAddr, localPort, err := parseAddrPort(fields[1])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse local address: %w", err)
	}

	remAddr, remPort, err := parseAddrPort(fields[2])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse remote address: %w", err)
	}

	state, ok := tcpStates[fields[3]]
	if !ok {
		state = "UNKNOWN"
	}

	inode, err := parseInode(fields[9])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse inode: %w", err)
	}

	return SocketEntry{
		Protocol:   "TCP",
		LocalAddr:  localAddr,
		LocalPort:  localPort,
		RemoteAddr: remAddr,
		RemotePort: remPort,
		State:      state,
		Inode:      inode,
	}, nil
}

func parseUDPEntry(fields []string) (SocketEntry, error) {
	if len(fields) < 12 {
		return SocketEntry{}, fmt.Errorf("expected at least 12 fields, got %d", len(fields))
	}

	localAddr, localPort, err := parseAddrPort(fields[1])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse local address: %w", err)
	}

	remAddr, remPort, err := parseAddrPort(fields[2])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse remote address: %w", err)
	}

	inode, err := parseInode(fields[9])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse inode: %w", err)
	}

	return SocketEntry{
		Protocol:   "UDP",
		LocalAddr:  localAddr,
		LocalPort:  localPort,
		RemoteAddr: remAddr,
		RemotePort: remPort,
		State:      "UNCONN",
		Inode:      inode,
	}, nil
}

func parseTCP6Entry(fields []string) (SocketEntry, error) {
	if len(fields) < 12 {
		return SocketEntry{}, fmt.Errorf("expected at least 12 fields, got %d", len(fields))
	}

	localAddr, localPort, err := parseAddrPort6(fields[1])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse local address: %w", err)
	}

	remAddr, remPort, err := parseAddrPort6(fields[2])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse remote address: %w", err)
	}

	state, ok := tcpStates[fields[3]]
	if !ok {
		state = "UNKNOWN"
	}

	inode, err := parseInode(fields[9])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse inode: %w", err)
	}

	return SocketEntry{
		Protocol:   "TCP6",
		LocalAddr:  localAddr,
		LocalPort:  localPort,
		RemoteAddr: remAddr,
		RemotePort: remPort,
		State:      state,
		Inode:      inode,
	}, nil
}

func parseUDP6Entry(fields []string) (SocketEntry, error) {
	if len(fields) < 12 {
		return SocketEntry{}, fmt.Errorf("expected at least 12 fields, got %d", len(fields))
	}

	localAddr, localPort, err := parseAddrPort6(fields[1])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse local address: %w", err)
	}

	remAddr, remPort, err := parseAddrPort6(fields[2])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse remote address: %w", err)
	}

	inode, err := parseInode(fields[9])
	if err != nil {
		return SocketEntry{}, fmt.Errorf("failed to parse inode: %w", err)
	}

	return SocketEntry{
		Protocol:   "UDP6",
		LocalAddr:  localAddr,
		LocalPort:  localPort,
		RemoteAddr: remAddr,
		RemotePort: remPort,
		State:      "UNCONN",
		Inode:      inode,
	}, nil
}

func parseFile(path string, parseEntry func([]string) (SocketEntry, error)) ([]SocketEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SocketEntry{}, nil
		}
		return nil, err
	}

	lines := bytes.Split(data, []byte{'\n'})
	entries := make([]SocketEntry, 0, 1024)

	for i, line := range lines {
		if i == 0 {
			continue
		}
		if len(line) == 0 {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) < 12 {
			continue
		}
		strFields := make([]string, len(fields))
		for j, f := range fields {
			strFields[j] = string(f)
		}
		entry, err := parseEntry(strFields)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func ParseTCP() ([]SocketEntry, error) {
	return parseFile("/proc/net/tcp", parseTCPEntry)
}

func ParseTCP6() ([]SocketEntry, error) {
	return parseFile("/proc/net/tcp6", parseTCP6Entry)
}

func ParseUDP() ([]SocketEntry, error) {
	return parseFile("/proc/net/udp", parseUDPEntry)
}

func ParseUDP6() ([]SocketEntry, error) {
	return parseFile("/proc/net/udp6", parseUDP6Entry)
}