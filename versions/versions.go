package versions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	once   sync.Once
	loaded map[string]string
	loadEr error
)

// Get returns the value for `key` (case-insensitive). Returns "" if the key
// is absent and an error only when versions.env cannot be located or parsed.
func Get(key string) (string, error) {
	once.Do(load)
	if loadEr != nil {
		return "", loadEr
	}
	return loaded[strings.ToLower(key)], nil
}

func load() {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		loadEr = fmt.Errorf("versions: cannot locate package source")
		return
	}
	repoDir := filepath.Dir(filepath.Dir(here)) // versions/ -> repo root
	path := filepath.Join(repoDir, "versions.env")
	data, err := os.ReadFile(path)
	if err != nil {
		loadEr = fmt.Errorf("versions: read %s: %w", path, err)
		return
	}
	loaded = parse(string(data))
}

func parse(content string) map[string]string {
	out := map[string]string{}
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return out
}
