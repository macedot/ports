package mapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID  int
	Name string
}

func BuildProcessMap() (map[uint64]ProcessInfo, error) {
	result := make(map[uint64]ProcessInfo)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pidStr := entry.Name()
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		procInfo, err := buildProcessInfo(pid)
		if err != nil {
			continue
		}

		inodes, err := readSocketInodes(pid)
		if err != nil {
			continue
		}

		for _, inode := range inodes {
			result[inode] = *procInfo
		}
	}

	return result, nil
}

func buildProcessInfo(pid int) (*ProcessInfo, error) {
	name, err := readProcessName(pid)
	if err != nil {
		return nil, err
	}
	return &ProcessInfo{PID: pid, Name: name}, nil
}

func readProcessName(pid int) (string, error) {
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")
		data, err = os.ReadFile(commPath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				return parts[1], nil
			}
		}
	}

	commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")
	data, err = os.ReadFile(commPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readSocketInodes(pid int) ([]uint64, error) {
	fdPath := filepath.Join("/proc", strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(fdPath)
	if err != nil {
		return nil, err
	}

	var inodes []uint64
	for _, entry := range entries {
		linkPath := filepath.Join(fdPath, entry.Name())
		target, err := os.Readlink(linkPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}

		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			inodeStr := target[8 : len(target)-1]
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if err != nil {
				continue
			}
			inodes = append(inodes, inode)
		}
	}

	return inodes, nil
}
