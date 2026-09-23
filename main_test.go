package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const abiSuccessMarker = "VERIFY_CGO_ABI_SUCCESS"

// TestMain_CGO_ABI_Integration builds the plugin as a dynamic library and loads
// it from a separate process, so the ABI contract is exercised end to end:
// exported symbols, init handshake, envelope exchange, management dispatch,
// buffer freeing and shutdown.
func TestMain_CGO_ABI_Integration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("dynamic library loading is verified on windows-latest in CI")
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("CGO toolchain (gcc) is unavailable; skipping dynamic library build")
	}

	tempDir := t.TempDir()
	libraryPath := filepath.Join(tempDir, "cpa-secret-manager-test.dll")
	statePath := filepath.Join(tempDir, "state", "cache.json")

	build := exec.Command("go", "build", "-buildmode=c-shared", "-o", libraryPath, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=1")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build dynamic library: %v\n%s", err, output)
	}
	if _, err := os.Stat(libraryPath); err != nil {
		t.Fatalf("dynamic library missing after build: %v", err)
	}

	probe := exec.Command("go", "run", ".", libraryPath, statePath)
	probe.Dir = filepath.Join(".", "testdata", "abi")
	probe.Env = append(os.Environ(), "CGO_ENABLED=1")
	output, err := probe.CombinedOutput()
	if err != nil {
		t.Fatalf("ABI probe failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), abiSuccessMarker) {
		t.Fatalf("ABI probe output missing %s:\n%s", abiSuccessMarker, output)
	}

	// Shutdown flushes the state document, so the probe must have created it.
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state document missing after shutdown: %v", err)
	}
}
