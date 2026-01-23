package progbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectSDKVersion reads the SDK version from project files.
// For Python: runs `uv tree --quiet --depth 0 --package temporalio`
// For TypeScript: parses package.json for @temporalio/* dependencies
func DetectSDKVersion(ctx context.Context, language, projectDir string) (string, error) {
	switch language {
	case "python":
		return detectPythonVersion(ctx, projectDir)
	case "typescript":
		return detectTypeScriptVersion(projectDir)
	default:
		return "", fmt.Errorf("version detection not supported for language %q", language)
	}
}

func detectPythonVersion(ctx context.Context, projectDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "uv", "tree", "--quiet", "--depth", "0", "--package", "temporalio")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed running uv tree: %w (is uv installed?)", err)
	}
	outStr := strings.TrimSpace(string(out))
	if strings.HasPrefix(outStr, "temporalio v") {
		return outStr[len("temporalio v"):], nil
	}
	return "", fmt.Errorf("version not found in uv tree output: %q", outStr)
}

func detectTypeScriptVersion(projectDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("failed reading package.json: %w", err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		// Look for "@temporalio/client": "1.11.0" or similar
		if strings.HasPrefix(line, "\"@temporalio/") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[len(parts)-2], nil
			}
		}
	}
	return "", fmt.Errorf("@temporalio/* dependency not found in package.json")
}
