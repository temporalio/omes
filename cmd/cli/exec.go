package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/temporalio/features/sdkbuild"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var language string
	var sdkVersion string
	var buildDir string
	var useExisting bool

	cmd := &cobra.Command{
		Use:   "exec [flags] -- <command...>",
		Short: "Build SDK and run command with the built SDK",
		Long: `Build the SDK (if needed) and run a command with the built SDK.

If no command is provided after --, just builds and exits (useful for Dockerfiles).
With --use-existing, skips build and fails if build doesn't exist.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()

			e := &execRunner{
				language:    language,
				sdkVersion:  sdkVersion,
				buildDir:    buildDir,
				useExisting: useExisting,
				logger:      sugar,
			}
			return e.run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&language, "language", "", "SDK language (python, typescript)")
	cmd.Flags().StringVar(&sdkVersion, "sdk-version", "", "SDK version or local path")
	cmd.Flags().StringVar(&buildDir, "build-dir", "", "Directory for SDK build output")
	cmd.Flags().BoolVar(&useExisting, "use-existing", false, "Use existing build, fail if not found")

	cmd.MarkFlagRequired("language")

	return cmd
}

type execRunner struct {
	language    string
	sdkVersion  string
	buildDir    string
	useExisting bool
	logger      *zap.SugaredLogger
}

func (e *execRunner) run(ctx context.Context, command []string) error {
	// Determine build directory
	buildDir := e.buildDir
	if buildDir == "" {
		if e.sdkVersion == "" {
			return fmt.Errorf("--sdk-version required when --build-dir not specified")
		}
		buildDir = filepath.Join(os.TempDir(), "omes-sdk-cache", hashVersion(e.language, e.sdkVersion))
	}

	// Handle --use-existing
	if e.useExisting {
		if !buildExists(buildDir) {
			return fmt.Errorf("--use-existing specified but no build found at %s", buildDir)
		}
		e.logger.Infof("Using existing build at %s", buildDir)
	} else {
		// Build if needed
		if e.sdkVersion == "" {
			return fmt.Errorf("--sdk-version required for building")
		}
		if !buildExists(buildDir) {
			if err := e.buildSDK(ctx, buildDir); err != nil {
				return fmt.Errorf("failed to build SDK: %w", err)
			}
		} else {
			e.logger.Infof("Using cached build at %s", buildDir)
		}
	}

	// If no command, just exit (build-only mode)
	if len(command) == 0 {
		e.logger.Infof("Build complete at %s", buildDir)
		return nil
	}

	// Run command with built SDK
	return e.runWithOverride(ctx, buildDir, command)
}

func buildExists(buildDir string) bool {
	_, err := os.Stat(buildDir)
	return err == nil
}

func hashVersion(language, version string) string {
	h := sha256.New()
	h.Write([]byte(language))
	h.Write([]byte(version))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (e *execRunner) buildSDK(ctx context.Context, buildDir string) error {
	baseDir := filepath.Dir(buildDir)
	dirName := filepath.Base(buildDir)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	e.logger.Infof("Building %s SDK at %s (version: %s)", e.language, buildDir, e.sdkVersion)

	switch e.language {
	case "python":
		_, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
			BaseDir: baseDir,
			DirName: dirName,
			Version: e.sdkVersion,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		return err

	case "typescript":
		_, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
			BaseDir: baseDir,
			DirName: dirName,
			Version: e.sdkVersion,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		return err

	default:
		return fmt.Errorf("unsupported language: %s", e.language)
	}
}

func (e *execRunner) runWithOverride(ctx context.Context, buildDir string, command []string) error {
	switch e.language {
	case "python":
		return e.runPythonWithOverride(ctx, buildDir, command)
	case "typescript":
		return e.runTypeScriptWithOverride(ctx, buildDir, command)
	default:
		return fmt.Errorf("unsupported language: %s", e.language)
	}
}

func (e *execRunner) runPythonWithOverride(ctx context.Context, buildDir string, command []string) error {
	// Find the wheel file
	wheelPattern := filepath.Join(buildDir, "dist", "temporalio-*.whl")
	wheels, err := filepath.Glob(wheelPattern)
	if err != nil || len(wheels) == 0 {
		return fmt.Errorf("no wheel found at %s", wheelPattern)
	}
	wheelPath := wheels[0]

	// Build uv command with override
	override := fmt.Sprintf("temporalio @ file://%s", wheelPath)
	args := append([]string{"run", "--override", override}, command...)

	e.logger.Infof("Running: uv %v", args)
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (e *execRunner) runTypeScriptWithOverride(ctx context.Context, buildDir string, command []string) error {
	// TypeScript: use NODE_PATH to make Node resolve @temporalio from buildDir's node_modules
	// This allows user code in any directory to use the built SDK version
	nodeModules := filepath.Join(buildDir, "node_modules")

	e.logger.Infof("Running with NODE_PATH=%s: %v", nodeModules, command)
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", nodeModules))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
