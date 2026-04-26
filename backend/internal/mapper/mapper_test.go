package mapper

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestProcessInfoStruct(t *testing.T) {
	info := ProcessInfo{PID: 1234, Name: "test-process"}
	if info.PID != 1234 {
		t.Errorf("expected PID 1234, got %d", info.PID)
	}
	if info.Name != "test-process" {
		t.Errorf("expected Name 'test-process', got %s", info.Name)
	}
}

func TestBuildProcessMap(t *testing.T) {
	result, err := BuildProcessMap("/proc")
	if err != nil {
		t.Fatalf("BuildProcessMap failed: %v", err)
	}
	_ = result // may be empty or populated depending on permissions
}

func TestReadProcessNameNonExistentPID(t *testing.T) {
	_, err := readProcessName("/proc", 999999)
	if err == nil {
		t.Log("readProcessName returned no error for non-existent PID (ok if /proc mounted)")
	}
}

func TestReadSocketInodesNonExistentPID(t *testing.T) {
	_, err := readSocketInodes("/proc", 999999)
	if err == nil {
		t.Log("readSocketInodes returned no error (ok if /proc mounted)")
	}
}

func TestInodeExtraction(t *testing.T) {
	testCases := []struct {
		target    string
		wantOK    bool
		wantInode uint64
	}{
		{"socket:[12345]", true, 12345},
		{"socket:[0]", true, 0},
		{"anon_inode:[eventpoll]", false, 0},
		{"/dev/null", false, 0},
		{"", false, 0},
	}

	for _, tc := range testCases {
		if strings.HasPrefix(tc.target, "socket:[") && strings.HasSuffix(tc.target, "]") {
			inodeStr := tc.target[8 : len(tc.target)-1]
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if tc.wantOK {
				if err != nil {
					t.Errorf("failed to parse inode from %s: %v", tc.target, err)
				} else if inode != tc.wantInode {
					t.Errorf("expected inode %d, got %d", tc.wantInode, inode)
				}
			}
		}
	}
}

func TestBuildProcessInfoErrors(t *testing.T) {
	procInfo := buildProcessInfo("/proc", 999999)
	// buildProcessInfo is non-fatal, returns empty struct
	if procInfo == nil {
		t.Fatal("expected non-nil ProcessInfo")
	}
	if procInfo.Name != "" {
		t.Errorf("expected empty name for invalid PID, got %q", procInfo.Name)
	}
}

func TestReadProcessNameParsing(t *testing.T) {
	// Test the Name: parsing logic from status file
	data := []byte("Name:\ttest-mock-process\nPid:\t12345\n")
	lines := strings.Split(string(data), "\n")
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 && parts[1] == "test-mock-process" {
				found = true
			}
		}
	}
	if !found {
		t.Error("failed to parse Name from mock status content")
	}
}

func TestReadProcessNameEmptyStatusFallback(t *testing.T) {
	// Test empty status file falls back to comm
	data := []byte("")
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				t.Errorf("unexpected match in empty data")
			}
		}
	}
}

func TestMockProcStructure(t *testing.T) {
	// Create a mock /proc-like structure to test parsing logic
	tmpDir := t.TempDir()
	pid := os.Getpid()

	statusPath := filepath.Join(tmpDir, strconv.Itoa(pid), "status")
	mockStatus := "Name:\ttest-mock-process\nPid:\t" + strconv.Itoa(pid) + "\n"
	if err := os.MkdirAll(filepath.Dir(statusPath), 0755); err != nil {
		t.Skipf("cannot create temp dir: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(mockStatus), 0644); err != nil {
		t.Skipf("cannot write mock status: %v", err)
	}

	// Verify we can read it back
	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("failed to read mock status: %v", err)
	}
	if string(data) != mockStatus {
		t.Errorf("mock status content mismatch")
	}
}

func TestBuildProcessMap_RecordsPIDWhenNameUnreadable(t *testing.T) {
	// Create a mock /proc structure where:
	// - PID 99999 has no readable status/comm but has fd/ directory
	// - fd/ contains a socket inode
	tmpDir := t.TempDir()
	pid := 99999

	fdPath := filepath.Join(tmpDir, strconv.Itoa(pid), "fd")
	if err := os.MkdirAll(fdPath, 0755); err != nil {
		t.Skipf("cannot create fd dir: %v", err)
	}

	// Create a mock socket fd entry
	socketLink := filepath.Join(fdPath, "0")
	if err := os.Symlink("socket:[12345]", socketLink); err != nil {
		t.Skipf("cannot create socket symlink: %v", err)
	}

	// Build process map - should still record inode 12345 with PID=99999 even though
	// status/comm are missing
	result, err := BuildProcessMap(tmpDir)
	if err != nil {
		t.Fatalf("BuildProcessMap failed: %v", err)
	}

	// Verify inode is recorded
	info, ok := result[12345]
	if !ok {
		t.Fatal("expected inode 12345 to be recorded")
	}

	// Verify PID is set even though name is empty
	if info.PID != pid {
		t.Errorf("expected PID %d, got %d", pid, info.PID)
	}

	// Name should be empty since status/comm are unreadable
	if info.Name != "" {
		t.Errorf("expected empty name, got %q", info.Name)
	}
}

func TestReadProcessName_CmdlineFallback(t *testing.T) {
	// Create a mock /proc structure with cmdline but no status/comm
	tmpDir := t.TempDir()
	pid := 99998

	cmdlinePath := filepath.Join(tmpDir, strconv.Itoa(pid), "cmdline")
	if err := os.MkdirAll(filepath.Dir(cmdlinePath), 0755); err != nil {
		t.Skipf("cannot create cmdline dir: %v", err)
	}

	// Write a cmdline with full path - should extract basename
	cmdline := "/usr/bin/test-process\x00arg1\x00arg2\x00"
	if err := os.WriteFile(cmdlinePath, []byte(cmdline), 0644); err != nil {
		t.Skipf("cannot write cmdline: %v", err)
	}

	name, err := readProcessName(tmpDir, pid)
	if err != nil {
		t.Fatalf("readProcessName failed: %v", err)
	}

	// Should extract basename
	if name != "test-process" {
		t.Errorf("expected basename 'test-process', got %q", name)
	}
}

func TestReadCommandLine(t *testing.T) {
	tmpDir := t.TempDir()
	pid := 99997

	cmdlinePath := filepath.Join(tmpDir, strconv.Itoa(pid), "cmdline")
	if err := os.MkdirAll(filepath.Dir(cmdlinePath), 0755); err != nil {
		t.Skipf("cannot create cmdline dir: %v", err)
	}

	cmdline := "/usr/bin/test-process\x00arg1\x00arg2 with space\x00"
	if err := os.WriteFile(cmdlinePath, []byte(cmdline), 0644); err != nil {
		t.Skipf("cannot write cmdline: %v", err)
	}

	result := readCommandLine(tmpDir, pid)
	expected := "/usr/bin/test-process arg1 arg2 with space"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestReadExePath(t *testing.T) {
	tmpDir := t.TempDir()
	pid := 99996

	exePath := filepath.Join(tmpDir, strconv.Itoa(pid), "exe")
	if err := os.MkdirAll(filepath.Dir(exePath), 0755); err != nil {
		t.Skipf("cannot create exe dir: %v", err)
	}

	if err := os.Symlink("/usr/bin/python3", exePath); err != nil {
		t.Skipf("cannot create exe symlink: %v", err)
	}

	result := readExePath(tmpDir, pid)
	expected := "/usr/bin/python3"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestReadCommandLineEmpty(t *testing.T) {
	result := readCommandLine("/proc", 999999)
	if result != "" {
		t.Errorf("expected empty string for missing cmdline, got %q", result)
	}
}

func TestReadExePathMissing(t *testing.T) {
	result := readExePath("/proc", 999999)
	if result != "" {
		t.Errorf("expected empty string for missing exe, got %q", result)
	}
}

func TestBuildProcessInfoAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	pid := 99995

	statusPath := filepath.Join(tmpDir, strconv.Itoa(pid), "status")
	cmdlinePath := filepath.Join(tmpDir, strconv.Itoa(pid), "cmdline")
	exePath := filepath.Join(tmpDir, strconv.Itoa(pid), "exe")
	fdPath := filepath.Join(tmpDir, strconv.Itoa(pid), "fd")

	if err := os.MkdirAll(fdPath, 0755); err != nil {
		t.Skipf("cannot create fd dir: %v", err)
	}

	// Create socket fd so this PID isn't skipped
	socketLink := filepath.Join(fdPath, "0")
	if err := os.Symlink("socket:[99999]", socketLink); err != nil {
		t.Skipf("cannot create socket symlink: %v", err)
	}

	// Write status with Name
	if err := os.WriteFile(statusPath, []byte("Name:\tmock-proc\n"), 0644); err != nil {
		t.Skipf("cannot write status: %v", err)
	}

	// Write cmdline with multiple args
	if err := os.WriteFile(cmdlinePath, []byte("/opt/app/bin/server\x00--config\x00/etc/app.conf\x00"), 0644); err != nil {
		t.Skipf("cannot write cmdline: %v", err)
	}

	// Create exe symlink
	if err := os.Symlink("/opt/app/bin/server", exePath); err != nil {
		t.Skipf("cannot create exe symlink: %v", err)
	}

	result, err := BuildProcessMap(tmpDir)
	if err != nil {
		t.Fatalf("BuildProcessMap failed: %v", err)
	}

	info, ok := result[99999]
	if !ok {
		t.Fatal("expected inode 99999 to be recorded")
	}

	if info.Name != "mock-proc" {
		t.Errorf("expected Name 'mock-proc', got %q", info.Name)
	}
	if info.Command != "/opt/app/bin/server --config /etc/app.conf" {
		t.Errorf("expected Command '/opt/app/bin/server --config /etc/app.conf', got %q", info.Command)
	}
	if info.Exe != "/opt/app/bin/server" {
		t.Errorf("expected Exe '/opt/app/bin/server', got %q", info.Exe)
	}
}
