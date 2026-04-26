package mapper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID  int
	Name string
}

func BuildProcessMap(procPath string) (map[uint64]ProcessInfo, error) {
	result := make(map[uint64]ProcessInfo)

	entries, err := os.ReadDir(procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", procPath, err)
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

		procInfo, err := buildProcessInfo(procPath, pid)
		if err != nil {
			log.Printf("mapper: skipping PID %d (buildProcessInfo failed): %v", pid, err)
			continue
		}

		inodes, err := readSocketInodes(procPath, pid)
		if err != nil {
			log.Printf("mapper: skipping PID %d (readSocketInodes failed): %v", pid, err)
			continue
		}

		for _, inode := range inodes {
			result[inode] = *procInfo
		}
	}

	return result, nil
}

func buildProcessInfo(procPath string, pid int) (*ProcessInfo, error) {
	name, err := readProcessName(procPath, pid)
	if err != nil {
		return nil, err
	}
	return &ProcessInfo{PID: pid, Name: name}, nil
}

func readProcessName(procPath string, pid int) (string, error) {
	statusPath := filepath.Join(procPath, strconv.Itoa(pid), "status")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		commPath := filepath.Join(procPath, strconv.Itoa(pid), "comm")
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

	commPath := filepath.Join(procPath, strconv.Itoa(pid), "comm")
	data, err = os.ReadFile(commPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readSocketInodes(procPath string, pid int) ([]uint64, error) {
	fdPath := filepath.Join(procPath, strconv.Itoa(pid), "fd")
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
			log.Printf("mapper: readlink %s failed: %v", linkPath, err)
			continue
		}

		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			inodeStr := target[8 : len(target)-1]
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if err != nil {
				log.Printf("mapper: invalid socket inode %q: %v", inodeStr, err)
				continue
			}
			inodes = append(inodes, inode)
		}
	}

	return inodes, nil
}
