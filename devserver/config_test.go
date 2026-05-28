package devserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

func TestAllocatePorts(t *testing.T) {
	ports, err := allocatePorts("127.0.0.1")
	require.NoError(t, err)
	seen := make(map[int]struct{}, portCount)
	for _, port := range ports {
		require.GreaterOrEqual(t, port, safePortMin)
		require.Less(t, port, 32768)
		_, ok := seen[port]
		require.False(t, ok, "duplicate port %d", port)
		seen[port] = struct{}{}
	}
}

func TestDefaultOutputDirUsesRepoRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644))
	nested := filepath.Join(root, "scenarios", "project")
	require.NoError(t, os.MkdirAll(nested, 0755))

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(nested))
	t.Cleanup(func() { require.NoError(t, os.Chdir(prevDir)) })

	outputDir, err := defaultOutputDir("")
	require.NoError(t, err)
	root, err = filepath.EvalSymlinks(root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, ".devserver"), outputDir)
}

func TestBuildCacheLockPath(t *testing.T) {
	require.Equal(
		t,
		filepath.Join("tmp", ".devserver.lock"),
		buildCacheLockPath(filepath.Join("tmp", ".devserver")),
	)
}

func TestWriteDynamicConfig(t *testing.T) {
	tmp := t.TempDir()
	path, err := writeDynamicConfig(tmp, "127.0.0.1:7243", map[string]any{
		"nexusoperation.enableStandalone": true,
		"my.numeric.setting":              42,
		"my.string.setting":               "hello",
		// Override an existing default to verify caller wins.
		"frontend.persistenceMaxQPS": 999,
	})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// Parse instead of byte-compare so formatting changes (indent, trailing
	// newlines) don't break the test — only the semantic content matters.
	var parsed map[string][]map[string]any
	require.NoError(t, yaml.Unmarshal(data, &parsed))

	require.Len(t, parsed, len(dynamicConfigDefaults("127.0.0.1:7243"))+3)
	for k, want := range map[string]any{
		"frontend.persistenceMaxQPS":      999,
		"nexusoperation.enableStandalone": true,
		"my.numeric.setting":              42,
		"my.string.setting":               "hello",
	} {
		entries, ok := parsed[k]
		require.True(t, ok, "missing key %s", k)
		require.Len(t, entries, 1, "key %s", k)
		require.Equal(t, want, entries[0]["value"], "key %s value", k)
		require.Equal(t, map[string]any{}, entries[0]["constraints"], "key %s constraints", k)
	}
}

func TestIsRetryable(t *testing.T) {
	require.True(t, isRetryable(status.Error(codes.Unavailable, "x")))
	require.True(t, isRetryable(status.Error(codes.DeadlineExceeded, "x")))
	require.True(t, isRetryable(status.Error(codes.ResourceExhausted, "x")))
	require.True(t, isRetryable(status.Error(codes.Internal, "x")))

	require.False(t, isRetryable(status.Error(codes.InvalidArgument, "x")))
	require.False(t, isRetryable(status.Error(codes.PermissionDenied, "x")))
	require.False(t, isRetryable(status.Error(codes.NotFound, "x")))

	// Non-gRPC errors retry.
	require.True(t, isRetryable(errors.New("connection refused")))
}
