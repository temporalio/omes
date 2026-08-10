package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunInstallToolsInstallsExpandedToolsAndUsesMiseExec(t *testing.T) {
	lines, err := withFakeMise(t, "", func() error {
		return runInstallTools(context.Background(), []string{"node", "protoc", "python", "node"})
	})
	require.NoError(t, err)

	require.Equal(t, []string{
		"install node pnpm protoc protoc-gen-go python uv pipx:poethepoet|",
		"exec -- npm ci|false",
		"exec -- uv sync --python 3.10 --all-packages --all-groups|false",
	}, lines)
}

func TestInstallAllUsesBareMiseInstall(t *testing.T) {
	for _, args := range [][]string{nil, {"all"}} {
		t.Run(strings.Join(args, ","), func(t *testing.T) {
			lines, err := withFakeMise(t, "", func() error {
				cmd := installCmd()
				cmd.SetArgs(args)
				return cmd.ExecuteContext(context.Background())
			})
			require.NoError(t, err)
			require.Equal(t, []string{
				"install|",
				"exec -- dotnet restore|false",
				"exec -- go mod tidy|false",
				"exec -- ./gradlew build --dry-run|false",
				"exec -- npm ci|false",
				"exec -- uv sync --python 3.10 --all-packages --all-groups|false",
				"exec -- bundle install|false",
			}, lines)
		})
	}
}

func TestRunInstallToolsRejectsUnsupportedTool(t *testing.T) {
	lines, err := withFakeMise(t, "", func() error {
		return runInstallTools(context.Background(), []string{"not-a-tool"})
	})
	require.Error(t, err)
	require.Empty(t, lines)
}

func TestRunInstallToolsStopsWhenMiseInstallFails(t *testing.T) {
	lines, err := withFakeMise(t, "install", func() error {
		return runInstallTools(context.Background(), []string{"node"})
	})
	require.Error(t, err)
	require.Equal(t, []string{"install node pnpm|"}, lines)
}

func TestRunInstallToolsRunsProjectSetupThroughMiseExec(t *testing.T) {
	lines, err := withFakeMise(t, "", func() error {
		return runInstallTools(context.Background(), []string{"dotnet", "go", "java", "python", "node", "ruby"})
	})
	require.NoError(t, err)

	require.Equal(t, []string{
		"install dotnet go java python uv pipx:poethepoet node pnpm ruby|",
		"exec -- dotnet restore|false",
		"exec -- go mod tidy|false",
		"exec -- ./gradlew build --dry-run|false",
		"exec -- uv sync --python 3.10 --all-packages --all-groups|false",
		"exec -- npm ci|false",
		"exec -- bundle install|false",
	}, lines)
}

func withFakeMise(t *testing.T, failOn string, run func() error) ([]string, error) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "mise.log")
	misePath := filepath.Join(dir, "mise")
	require.NoError(t, os.WriteFile(misePath, []byte(`#!/bin/sh
printf '%s|%s\n' "$*" "${MISE_AUTO_INSTALL-}" >> "$MISE_LOG"
if [ "$1" = "$MISE_FAIL_ON" ]; then
  exit 1
fi
`), 0o755))
	t.Setenv("MISE_LOG", logPath)
	t.Setenv("MISE_FAIL_ON", failOn)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	runErr := run()

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil, runErr
	}
	require.NoError(t, err)
	return strings.Split(strings.TrimSpace(string(data)), "\n"), runErr
}
