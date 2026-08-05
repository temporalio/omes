package versions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

var (
	once   sync.Once
	loaded miseConfig
	loadEr error
)

// Get returns the value for the legacy version key convention. Versions now
// come from mise.toml, whose tool and metadata entries are the source of truth.
func Get(key string) (string, error) {
	once.Do(load)
	if loadEr != nil {
		return "", loadEr
	}
	return loaded.get(key), nil
}

type miseConfig struct {
	Tools map[string]any `toml:"tools"`
	Meta  struct {
		SDK    map[string]string `toml:"sdk"`
		Server struct {
			Ref string `toml:"ref"`
		} `toml:"server"`
	} `toml:"_"`
}

func load() {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		loadEr = fmt.Errorf("versions: cannot locate package source")
		return
	}
	repoDir := filepath.Dir(filepath.Dir(filepath.Dir(here)))
	loaded, loadEr = loadConfig(filepath.Join(repoDir, "mise.toml"))
}

func loadConfig(path string) (miseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return miseConfig{}, fmt.Errorf("versions: read %s: %w", path, err)
	}

	var config miseConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return miseConfig{}, fmt.Errorf("versions: parse %s: %w", path, err)
	}
	return config, nil
}

func (c miseConfig) get(key string) string {
	key = strings.ToLower(key)
	switch {
	case key == "server_version":
		return c.Meta.Server.Ref
	case strings.HasSuffix(key, "_sdk_version"):
		return c.Meta.SDK[strings.TrimSuffix(key, "_sdk_version")]
	case strings.HasSuffix(key, "_version"):
		tool := strings.ReplaceAll(strings.TrimSuffix(key, "_version"), "_", "-")
		if tool == "cargo" {
			tool = "rust"
		}
		return toolVersion(c.Tools[tool])
	default:
		return ""
	}
}

func toolVersion(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case map[string]any:
		version, _ := value["version"].(string)
		return version
	default:
		return ""
	}
}
