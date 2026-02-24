package programbuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testPackageJSON struct {
	Dependencies     map[string]string `json:"dependencies,omitempty"`
	DevDependencies  map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
}

func writePackageJSON(t *testing.T, dir string, pkg testPackageJSON) {
	t.Helper()
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("failed to marshal package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
}

func TestDetectTypescriptVersionIgnoresStarterFileDep(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/omes-starter": "file:../..",
			"@temporalio/client":       ">=1.0.0",
		},
	})

	got, err := detectTypescriptVersion(projectDir)
	if err != nil {
		t.Fatalf("detectTypescriptVersion returned error: %v", err)
	}
	if got != ">=1.0.0" {
		t.Fatalf("unexpected version: got %q, want %q", got, ">=1.0.0")
	}
}

func TestDetectTypescriptVersionPrefersDependenciesOverDevDependencies(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/worker": "1.2.3",
		},
		DevDependencies: map[string]string{
			"@temporalio/client": "9.9.9",
		},
	})

	got, err := detectTypescriptVersion(projectDir)
	if err != nil {
		t.Fatalf("detectTypescriptVersion returned error: %v", err)
	}
	if got != "1.2.3" {
		t.Fatalf("unexpected version: got %q, want %q", got, "1.2.3")
	}
}

func TestDetectTypescriptVersionFallsBackToDevDependencies(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/omes-starter": "file:../..",
		},
		DevDependencies: map[string]string{
			"@temporalio/client": "^1.3.0",
		},
	})

	got, err := detectTypescriptVersion(projectDir)
	if err != nil {
		t.Fatalf("detectTypescriptVersion returned error: %v", err)
	}
	if got != "^1.3.0" {
		t.Fatalf("unexpected version: got %q, want %q", got, "^1.3.0")
	}
}

func TestDetectTypescriptVersionFileDependencyReturnsError(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/client": "file:../sdk-typescript/packages/client",
		},
	})

	_, err := detectTypescriptVersion(projectDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pass --version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectTypescriptVersionFileURLDependencyReturnsError(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/client": "file:///tmp/fake-sdk/packages/client",
		},
	})

	_, err := detectTypescriptVersion(projectDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pass --version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectTypescriptVersionMissingSDKDepsReturnsError(t *testing.T) {
	projectDir := t.TempDir()
	writePackageJSON(t, projectDir, testPackageJSON{
		Dependencies: map[string]string{
			"@temporalio/omes-starter": "file:../..",
		},
	})

	_, err := detectTypescriptVersion(projectDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "expected @temporalio/client|worker|workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}
