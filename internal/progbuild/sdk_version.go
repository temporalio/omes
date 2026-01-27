package progbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func ResolveSDKVersion(ctx context.Context, language, projectDir string, logger *zap.SugaredLogger) (string, error) {
	version, err := DetectSDKVersion(ctx, language, projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to detect SDK version (use --version to specify): %w", err)
	}
	logger.Infof("Auto-detected SDK version: %s", version)
	return version, nil
}

// DetectSDKVersion reads the SDK version from project files.
// For Python: runs `uv tree --quiet --depth 0 --package temporalio`
// For TypeScript: parses package.json for @temporalio/* dependencies
func DetectSDKVersion(ctx context.Context, language, projectDir string) (string, error) {
	switch language {
	case "python":
		return DetectPythonVersion(ctx, projectDir)
	case "typescript":
		return DetectTypeScriptVersion(projectDir)
	case "go":
		// Go toolchain handles version from go.mod automatically
		return "", nil
	default:
		return "", fmt.Errorf("unrecognized language %q", language)
	}
}

func DetectPythonVersion(ctx context.Context, projectDir string) (string, error) {
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

func DetectTypeScriptVersion(projectDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("failed reading package.json: %w", err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"temporalio:\"") || strings.HasPrefix(line, "\"@temporalio/") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[len(parts)-2], nil
			}
		}
	}
	return "", fmt.Errorf("version not found in package.json")
}
