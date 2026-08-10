package versions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadMiseConfigProvidesToolAndMetadataVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mise.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
[tools]
go = "1.26.5"
protoc-gen-go = "v1.31.0"
rust = { version = "1.74.0", profile = "default" }

[_.sdk]
go = "1.47.0"

[_.server]
ref = "server-ref"
`), 0o644))

	versions, err := loadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "1.26.5", versions.get("GO_VERSION"))
	require.Equal(t, "v1.31.0", versions.get("PROTOC_GEN_GO_VERSION"))
	require.Equal(t, "1.74.0", versions.get("CARGO_VERSION"))
	require.Equal(t, "1.47.0", versions.get("GO_SDK_VERSION"))
	require.Equal(t, "server-ref", versions.get("SERVER_VERSION"))
	require.Equal(t, "", versions.get("UNKNOWN_VERSION"))
}
