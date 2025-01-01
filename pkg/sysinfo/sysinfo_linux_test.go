package sysinfo // import "github.com/docker/docker/pkg/sysinfo"

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd/pkg/seccomp"
)

func TestReadProcBool(t *testing.T) {
	tmpDir := t.TempDir()

	procFile := filepath.Join(tmpDir, "read-proc-bool")
	if err := os.WriteFile(procFile, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !readProcBool(procFile) {
		t.Fatal("expected proc bool to be true, got false")
	}

	if err := os.WriteFile(procFile, []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if readProcBool(procFile) {
		t.Fatal("expected proc bool to be false, got true")
	}

	if readProcBool(filepath.Join(tmpDir, "no-exist")) {
		t.Fatal("should be false for non-existent entry")
	}
}

func TestCgroupEnabled(t *testing.T) {
	cgroupDir := t.TempDir()

	if cgroupEnabled(cgroupDir, "test") {
		t.Fatal("cgroupEnabled should be false")
	}

	if err := os.WriteFile(filepath.Join(cgroupDir, "test"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	if !cgroupEnabled(cgroupDir, "test") {
		t.Fatal("cgroupEnabled should be true")
	}
}

// TestNew verifies that sysInfo is initialized with the expected values.
func TestNew(t *testing.T) {
	sysInfo := New()
	if sysInfo == nil {
		t.Fatal("sysInfo should not be nil")
	}
	if expected := seccomp.IsEnabled(); sysInfo.Seccomp != expected {
		t.Errorf("got Seccomp %v, wanted %v", sysInfo.Seccomp, expected)
	}
	if expected := apparmorSupported(); sysInfo.AppArmor != expected {
		t.Errorf("got AppArmor %v, wanted %v", sysInfo.AppArmor, expected)
	}
	if expected := cgroupnsSupported(); sysInfo.CgroupNamespaces != expected {
		t.Errorf("got CgroupNamespaces %v, wanted %v", sysInfo.AppArmor, expected)
	}
}

func TestNumCPU(t *testing.T) {
	if cpuNumbers := NumCPU(); cpuNumbers <= 0 {
		t.Fatal("CPU returned must be greater than zero")
	}
}

func TestIsCpusetListAvailable(t *testing.T) {
	cases := []struct {
		provided  string
		available string
		res       bool
		err       bool
	}{
		{"1", "0-4", true, false},
		{"01,3", "0-4", true, false},
		{"", "0-7", true, false},
		{"1--42", "0-7", false, true},
		{"1-42", "00-1,8,,9", false, true},
		{"1,41-42", "43,45", false, false},
		{"0-3", "", false, false},
	}
	for _, c := range cases {
		r, err := isCpusetListAvailable(c.provided, c.available)
		if (c.err && err == nil) && r != c.res {
			t.Fatalf("Expected pair: %v, %v for %s, %s. Got %v, %v instead", c.res, c.err, c.provided, c.available, (c.err && err == nil), r)
		}
	}
}
