package mapper

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID     int
	Name    string // short name from status/comm/cmdline-basename
	Command string // full command line from /proc/[pid]/cmdline (args joined with spaces)
	Exe     string // executable path from readlink /proc/[pid]/exe
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

		// Read socket inodes FIRST — critical data, never skip on name failure
		inodes, err := readSocketInodes(procPath, pid)
		if err != nil {
			log.Printf("mapper: skipping PID %d (readSocketInodes failed): %v", pid, err)
			continue
		}
		if len(inodes) == 0 {
			continue
		}

		// Try to get process name — non-fatal if it fails
		name := ""
		procInfo := buildProcessInfo(procPath, pid)
		name = procInfo.Name

		for _, inode := range inodes {
			result[inode] = ProcessInfo{PID: pid, Name: name, Command: procInfo.Command, Exe: procInfo.Exe}
		}
	}

	return result, nil
}

func buildProcessInfo(procPath string, pid int) *ProcessInfo {
	name, _ := readProcessName(procPath, pid)
	command := readCommandLine(procPath, pid)
	exe := readExePath(procPath, pid)
	return &ProcessInfo{PID: pid, Name: name, Command: command, Exe: exe}
}

func readCommandLine(procPath string, pid int) string {
	cmdlinePath := filepath.Join(procPath, strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
}

func readExePath(procPath string, pid int) string {
	exePath := filepath.Join(procPath, strconv.Itoa(pid), "exe")
	target, err := os.Readlink(exePath)
	if err != nil {
		return ""
	}
	return target
}

func readProcessName(procPath string, pid int) (string, error) {
	// Try /proc/[pid]/status
	statusPath := filepath.Join(procPath, strconv.Itoa(pid), "status")
	data, err := os.ReadFile(statusPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name:") {
				parts := strings.SplitN(line, "\t", 2)
				if len(parts) == 2 {
					return parts[1], nil
				}
			}
		}
	}

	// Try /proc/[pid]/comm
	commPath := filepath.Join(procPath, strconv.Itoa(pid), "comm")
	data, err = os.ReadFile(commPath)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Try /proc/[pid]/cmdline (first arg, null-separated)
	cmdlinePath := filepath.Join(procPath, strconv.Itoa(pid), "cmdline")
	data, err = os.ReadFile(cmdlinePath)
	if err == nil && len(data) > 0 {
		// cmdline is null-separated; take first element
		args := strings.Split(string(data), "\x00")
		if len(args) > 0 && args[0] != "" {
			// Extract basename from full path (e.g. "/usr/bin/nginx" → "nginx")
			base := filepath.Base(args[0])
			return base, nil
		}
	}

	return "", fmt.Errorf("no readable name source for pid %d", pid)
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
			if errors.Is(err, os.ErrPermission) {
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
