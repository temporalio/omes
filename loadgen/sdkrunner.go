package loadgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

// EnsureBuild ensures the SDK is built, returning the build directory.
func (r *SDKRunner) EnsureBuild(ctx context.Context) (string, error) {
	buildDir := r.BuildDir
	if buildDir == "" {
		if r.SDKVersion == "" {
			return "", fmt.Errorf("--sdk-version required when --build-dir not specified")
		}
		buildDir = filepath.Join(os.TempDir(), "omes-sdk-cache", hashVersion(r.Language, r.SDKVersion))
	}

	if r.UseExisting {
		if !buildExists(buildDir) {
			return "", fmt.Errorf("--use-existing specified but no build found at %s", buildDir)
		}
		r.Logger.Infof("Using existing build at %s", buildDir)
		return buildDir, nil
	}

	if r.SDKVersion == "" {
		return "", fmt.Errorf("--sdk-version required for building")
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
	baseDir := filepath.Dir(buildDir)
	dirName := filepath.Base(buildDir)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	r.Logger.Infof("Building %s SDK at %s (version: %s)", r.Language, buildDir, r.SDKVersion)

	switch r.Language {
	case "python":
		_, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
			BaseDir: baseDir,
			DirName: dirName,
			Version: r.SDKVersion,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		return err

	case "typescript":
		_, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
			BaseDir: baseDir,
			DirName: dirName,
			Version: r.SDKVersion,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		return err

	default:
		return fmt.Errorf("unsupported language: %s", r.Language)
	}
}

// BuildCommandWithOverride creates an exec.Cmd with the built SDK injected.
func BuildCommandWithOverride(language, buildDir, program string, args []string) (*exec.Cmd, error) {
	switch language {
	case "python":
		wheelPattern := filepath.Join(buildDir, "dist", "temporalio-*.whl")
		wheels, err := filepath.Glob(wheelPattern)
		if err != nil || len(wheels) == 0 {
			return nil, fmt.Errorf("no wheel found at %s", wheelPattern)
		}
		wheelPath := wheels[0]

		override := fmt.Sprintf("temporalio @ file://%s", wheelPath)
		uvArgs := append([]string{"run", "--override", override, program}, args...)
		return exec.Command("uv", uvArgs...), nil

	case "typescript":
		nodeModules := filepath.Join(buildDir, "node_modules")
		cmd := exec.Command(program, args...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", nodeModules))
		return cmd, nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// RunCommandWithOverride runs a command with the built SDK.
func RunCommandWithOverride(ctx context.Context, logger *zap.SugaredLogger, language, buildDir string, command []string) error {
	cmd, err := BuildCommandWithOverride(language, buildDir, command[0], command[1:])
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
