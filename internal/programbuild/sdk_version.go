package programbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// detectSDKVersion reads the SDK version from project files.
// For Python: runs `uv tree --quiet --depth 0 --package temporalio`
// For TypeScript: parses package.json for @temporalio/* dependencies
func detectSDKVersion(ctx context.Context, language, projectDir string) (string, error) {
	switch language {
	case "python":
		return detectPythonVersion(ctx, projectDir)
	case "typescript":
		return detectTypescriptVersion(projectDir)
	case "go":
		// Go toolchain handles version from go.mod automatically
		return "", nil
	default:
		return "", fmt.Errorf("unrecognized language %q", language)
	}
}

func detectPythonVersion(ctx context.Context, projectDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "uv", "tree", "--quiet", "--depth", "0", "--package", "temporalio")
	cmd.Dir = projectDir
	out, err := cmd.Output()

	if err != nil {
		return "", fmt.Errorf("failed running uv tree: %w", err)
	}

	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "temporalio v") {
		return outStr[len("temporalio v"):], nil
	}

	return "", fmt.Errorf("version not found in uv tree output: %q", outStr)
}

func detectTypescriptVersion(projectDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("failed reading package.json: %w", err)
	}

	type packageJSON struct {
		Dependencies     map[string]string `json:"dependencies"`
		DevDependencies  map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", fmt.Errorf("failed parsing package.json: %w", err)
	}

	// Search dependency maps in precedence order and only for real SDK packages.
	// Intentionally excludes @temporalio/omes-starter which may be a local
	// file dependency and not a Temporal SDK version selector.
	depMaps := []map[string]string{
		pkg.Dependencies,
		pkg.DevDependencies,
		pkg.PeerDependencies,
	}
	for _, depMap := range depMaps {
		version, ok := findTypeScriptSDKVersionSpec(depMap)
		if !ok {
			continue
		}

		if strings.HasPrefix(version, "file:") {
			return "", fmt.Errorf("detected local TypeScript SDK dependency %q in package.json; pass --version <sdk-typescript-path> explicitly", version)
		}
		return version, nil
	}

	return "", fmt.Errorf("version not found in package.json (expected @temporalio/client|worker|workflow)")
}

func findTypeScriptSDKVersionSpec(depMap map[string]string) (string, bool) {
	if depMap == nil {
		return "", false
	}
	for _, depName := range []string{
		"@temporalio/client",
		"@temporalio/worker",
		"@temporalio/workflow",
		"@temporalio/activity",
		"@temporalio/common",
		"@temporalio/proto",
	} {
		spec, ok := depMap[depName]
		if ok && strings.TrimSpace(spec) != "" {
			return strings.TrimSpace(spec), true
		}
	}
	return "", false
}
