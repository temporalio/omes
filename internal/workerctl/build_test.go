package workerctl

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestUsePnpmInsteadOfCorepack(t *testing.T) {
	pnpmDir := t.TempDir()
	pnpmName := "pnpm"
	if runtime.GOOS == "windows" {
		pnpmName += ".cmd"
	}
	pnpmPath := filepath.Join(pnpmDir, pnpmName)
	if err := os.WriteFile(pnpmPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pnpmDir)

	tests := []struct {
		name     string
		corepack string
	}{
		{
			name:     "absolute path",
			corepack: "corepack",
		},
		{
			name:     "Windows command path",
			corepack: "corepack.cmd",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &exec.Cmd{
				Path: filepath.Join(t.TempDir(), test.corepack),
				Args: []string{"corepack", "pnpm", "install", "--frozen-lockfile"},
			}
			if err := usePnpmInsteadOfCorepack(context.Background(), cmd); err != nil {
				t.Fatal(err)
			}

			if cmd.Path != pnpmPath {
				t.Errorf("expected executable %q, got %q", pnpmPath, cmd.Path)
			}
			wantArgs := []string{pnpmPath, "install", "--frozen-lockfile"}
			if !slices.Equal(cmd.Args, wantArgs) {
				t.Errorf("expected arguments %q, got %q", wantArgs, cmd.Args)
			}
		})
	}
}

func TestUsePnpmInsteadOfCorepackAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	cmd := exec.Command("corepack", "pnpm", "install")

	err := usePnpmInsteadOfCorepack(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error when pnpm is not on PATH")
	}
	if !strings.Contains(err.Error(), "pnpm") {
		t.Errorf("expected pnpm lookup error, got %v", err)
	}
}

func TestUsePnpmInsteadOfCorepackIgnoresOtherCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *exec.Cmd
	}{
		{
			name: "unrelated command",
			cmd:  exec.Command("npm", "install"),
		},
		{
			name: "corepack without pnpm",
			cmd:  exec.Command("corepack", "yarn", "install"),
		},
		{
			name: "corepack without arguments",
			cmd:  exec.Command("corepack"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.cmd.Path
			args := slices.Clone(test.cmd.Args)

			if err := usePnpmInsteadOfCorepack(context.Background(), test.cmd); err != nil {
				t.Fatal(err)
			}
			if test.cmd.Path != path {
				t.Errorf("expected executable %q, got %q", path, test.cmd.Path)
			}
			if !slices.Equal(test.cmd.Args, args) {
				t.Errorf("expected arguments %q, got %q", args, test.cmd.Args)
			}
		})
	}
}
