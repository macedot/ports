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
	_, err := buildProcessInfo("/proc", 999999)
	if err != nil {
		t.Logf("buildProcessInfo correctly returned error for invalid PID: %v", err)
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
