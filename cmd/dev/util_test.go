package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A checked-out submodule passes the check.
func TestCheckProtoSubmodulePassesWhenPopulated(t *testing.T) {
	repoDir := t.TempDir()
	seedAPIUpstream(t, repoDir, "temporal/api/common/v1/message.proto")

	require.NoError(t, checkProtoSubmoduleDir(repoDir))
}

// An empty api_upstream directory, which is what a non-recursive clone leaves behind, fails with the git command
// that fixes it.
func TestCheckProtoSubmoduleFailsWhenEmpty(t *testing.T) {
	repoDir := t.TempDir()
	seedAPIUpstream(t, repoDir)

	err := checkProtoSubmoduleDir(repoDir)
	require.ErrorContains(t, err, "workers/proto/api_upstream")
	require.ErrorContains(t, err, "git submodule update --init --recursive")
}

// A missing api_upstream directory fails the same way as an empty one.
func TestCheckProtoSubmoduleFailsWhenMissing(t *testing.T) {
	err := checkProtoSubmoduleDir(t.TempDir())
	require.ErrorContains(t, err, "git submodule update --init --recursive")
}

// The real repo checkout satisfies the check, so the paths in it stay honest.
func TestCheckProtoSubmoduleAgainstRealRepo(t *testing.T) {
	repoDir, err := getRepoDir()
	require.NoError(t, err)

	if entries, err := os.ReadDir(filepath.Join(repoDir, "workers", "proto", "api_upstream")); err != nil || len(entries) == 0 {
		t.Skip("api_upstream submodule is not checked out")
	}
	require.NoError(t, checkProtoSubmoduleDir(repoDir))
}

// seedAPIUpstream creates workers/proto/api_upstream under repoDir, containing the given files.
func seedAPIUpstream(t *testing.T, repoDir string, files ...string) {
	t.Helper()
	upstreamDir := filepath.Join(repoDir, "workers", "proto", "api_upstream")
	require.NoError(t, os.MkdirAll(upstreamDir, 0o755))
	for _, file := range files {
		path := filepath.Join(upstreamDir, filepath.FromSlash(file))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, nil, 0o644))
	}
}
