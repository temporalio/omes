package loadgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"go.uber.org/zap"
)

// SDKRunner handles SDK building and running commands with SDK overrides.
type SDKRunner struct {
	Language    string
	SDKVersion  string
	BuildDir    string
	UseExisting bool
	Logger      *zap.SugaredLogger
}

// isLocalPath returns true if the version looks like a local path.
func isLocalPath(version string) bool {
	return strings.Contains(version, "/") || strings.Contains(version, "\\")
}

// EnsureBuild ensures the SDK is built (if needed), returning the build directory.
// For released versions, returns empty string (no build needed).
// For local paths, builds and returns the build directory containing the wheel.
func (r *SDKRunner) EnsureBuild(ctx context.Context) (string, error) {
	// For released versions, no build is needed
	if !isLocalPath(r.SDKVersion) {
		r.Logger.Infof("Using released SDK version: %s", r.SDKVersion)
		return "", nil
	}

	// For local paths, we need to build the wheel
	buildDir := r.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join(os.TempDir(), "omes-sdk-cache", hashVersion(r.Language, r.SDKVersion))
	}

	if r.UseExisting {
		if !buildExists(buildDir) {
			return "", fmt.Errorf("--use-existing specified but no build found at %s", buildDir)
		}
		r.Logger.Infof("Using existing build at %s", buildDir)
		return buildDir, nil
	}

	if buildExists(buildDir) {
		r.Logger.Infof("Using cached build at %s", buildDir)
		return buildDir, nil
	}

	if err := r.buildSDK(ctx, buildDir); err != nil {
		return "", fmt.Errorf("failed to build SDK: %w", err)
	}

	return buildDir, nil
}

func (r *SDKRunner) buildSDK(ctx context.Context, buildDir string) error {
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	r.Logger.Infof("Building %s SDK at %s (from local path: %s)", r.Language, buildDir, r.SDKVersion)

	switch r.Language {
	case "python":
		return r.buildPythonWheel(ctx, buildDir)
	case "typescript":
		return r.buildTypeScriptPackages(ctx, buildDir)
	default:
		return fmt.Errorf("unsupported language: %s", r.Language)
	}
}

func (r *SDKRunner) buildPythonWheel(ctx context.Context, buildDir string) error {
	// Build the wheel from the local SDK path
	sdkPath, err := filepath.Abs(r.SDKVersion)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	distDir := filepath.Join(buildDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Use uv to build the wheel
	cmd := exec.CommandContext(ctx, "uv", "build", "--wheel", "--out-dir", distDir, sdkPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build wheel: %w", err)
	}

	return nil
}

func (r *SDKRunner) buildTypeScriptPackages(ctx context.Context, buildDir string) error {
	// For TypeScript, use sdkbuild which handles npm pack and linking
	// We need a minimal package.json to make sdkbuild happy
	baseDir := filepath.Dir(buildDir)
	dirName := filepath.Base(buildDir)

	// Create a minimal package.json in baseDir if it doesn't exist
	packageJSON := filepath.Join(baseDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		minimalPkg := `{"name": "omes-sdk-build", "version": "0.0.0", "dependencies": {}}`
		if err := os.WriteFile(packageJSON, []byte(minimalPkg), 0644); err != nil {
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	_, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir: baseDir,
		DirName: dirName,
		Version: r.SDKVersion,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	return err
}

// BuildCommandWithOverride creates an exec.Cmd with the SDK version injected.
// For released versions, buildDir should be empty.
// For local paths, buildDir should contain the built wheel.
func BuildCommandWithOverride(language, buildDir, sdkVersion, program string, args []string) (*exec.Cmd, error) {
	switch language {
	case "python":
		var uvArgs []string
		if buildDir == "" && sdkVersion == "" {
			// No version specified - use environment's temporalio
			uvArgs = append([]string{"run", program}, args...)
		} else if buildDir == "" {
			// Released version - use version specifier directly
			uvArgs = append([]string{"run", "--with", fmt.Sprintf("temporalio==%s", sdkVersion), program}, args...)
		} else {
			// Local path - use the built wheel
			wheelPattern := filepath.Join(buildDir, "dist", "temporalio-*.whl")
			wheels, err := filepath.Glob(wheelPattern)
			if err != nil || len(wheels) == 0 {
				return nil, fmt.Errorf("no wheel found at %s", wheelPattern)
			}
			uvArgs = append([]string{"run", "--with", wheels[0], program}, args...)
		}
		return exec.Command("uv", uvArgs...), nil

	case "typescript":
		if buildDir == "" {
			// For released versions without a build, just run directly
			// The user's package.json should specify the version
			return exec.Command(program, args...), nil
		}
		nodeModules := filepath.Join(buildDir, "node_modules")
		cmd := exec.Command(program, args...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", nodeModules))
		return cmd, nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// RunCommandWithOverride runs a command with the SDK version injected.
func RunCommandWithOverride(ctx context.Context, logger *zap.SugaredLogger, language, buildDir, sdkVersion string, command []string) error {
	cmd, err := BuildCommandWithOverride(language, buildDir, sdkVersion, command[0], command[1:])
	if err != nil {
		return err
	}

	logger.Infof("Running: %v", cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func hashVersion(language, version string) string {
	h := sha256.New()
	h.Write([]byte(language))
	h.Write([]byte(version))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func buildExists(buildDir string) bool {
	_, err := os.Stat(buildDir)
	return err == nil
}
