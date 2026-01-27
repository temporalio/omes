package progbuild

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"go.uber.org/zap"
)

type BuilderBase struct {
	// Directory of the program you would like to build with the SDK
	ProgramDir string
	// Output build directory
	OutputDir string
	// Options to configure the SDK build (i.e. language, version)
	SdkOpts clioptions.SdkOptions

	// Custom stdout/stderr capture
	Stdout io.Writer
	Stderr io.Writer

	Logger *zap.SugaredLogger
}

type IProgramBuilder interface {
	GoProgramOptions() sdkbuild.BuildGoProgramOptions
	PythonProgramOptions() sdkbuild.BuildPythonProgramOptions
	TypescriptProgramOptions() sdkbuild.BuildTypeScriptProgramOptions
}

func (b *BuilderBase) BuildProgram(ctx context.Context, builder IProgramBuilder) (sdkbuild.Program, error) {
	version, err := b.resolveSDKVersion(ctx)
	if err != nil {
		return nil, err
	}
	// Get absolute path to user's project
	absProjectDir, err := filepath.Abs(b.ProgramDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	switch b.SdkOpts.Language {
	case "python":
		return b.buildPython(ctx, absProjectDir, version)
	case "typescript":
		return b.buildTypeScript(ctx, absProjectDir, version)
	case "go":
		return b.buildGo(ctx, absProjectDir, version)
	default:
		return nil, fmt.Errorf("unsupported language: %s", b.SdkOpts.Language.String())
	}
}

func (b *BuilderBase) resolveSDKVersion(ctx context.Context) (string, error) {
	if b.SdkOpts.Version != "" {
		return b.SdkOpts.Version, nil
	}
	version, err := DetectSDKVersion(ctx, b.SdkOpts.Language.String(), b.ProgramDir)
	if err != nil {
		return "", fmt.Errorf("failed to detect SDK version (use --version to specify): %w", err)
	}
	b.Logger.Infof("Auto-detected SDK version: %s", version)
	return version, nil
}

func (b *BuilderBase) buildPython(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Python SDK (version: %s)", version)

	// Use sdkbuild directly - it creates temp dir, installs SDK, adds user project as editable
	// User project must have src/__main__.py as entry point
	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir: absProjectDir,
		DirName: b.OutputDir,
		Version: version,
		Stdout:  b.Stdout,
		Stderr:  b.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Python program: %w", err)
	}

	return prog, nil
}

func (b *BuilderBase) buildTypeScript(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building TypeScript SDK (version: %s)", version)

	// TODO(thomas): If version is a local path, pre-build if needed (handles pnpm projects), sdkbuild still tries to build with npm
	if strings.ContainsAny(version, `/\`) {
		if err := b.preBuildLocalTypeScriptSDK(ctx, version); err != nil {
			return nil, err
		}
	}

	// Build dir will be a temp sibling to project dir (both in tests/)
	// Project: tests/<project-name>/, Build: tests/<temp>/
	baseDir := filepath.Dir(absProjectDir)
	projectName := filepath.Base(absProjectDir)

	// Paths relative to build dir (which is a child of baseDir)
	relProjectDir := "../" + projectName // One level up, into project
	relSharedSrc := "../../src"          // Two levels up, into src

	// Set up TSConfigPaths and Includes
	// Paths point to compiled output for runtime resolution via tsconfig-paths
	// TypeScript compiler finds types via the includes, not the paths
	tsConfigPaths := map[string][]string{
		"@temporalio/omes-starter":   {"./tslib/src/index.js"},
		"@temporalio/omes-starter/*": {"./tslib/src/*"},
	}
	includes := []string{
		relProjectDir + "/**/*.ts",
		relSharedSrc + "/**/*.ts",
	}

	// Add dependencies used by omes-starter
	moreDeps := map[string]string{
		"express":        "^4.18.0",
		"@types/express": "^4.17.0",
		"commander":      "^8.3.0",
	}

	prog, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir:          baseDir,
		DirName:          b.OutputDir,
		Version:          version,
		TSConfigPaths:    tsConfigPaths,
		Includes:         includes,
		MoreDependencies: moreDeps,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build TypeScript program: %w", err)
	}

	return prog, nil
}

func (b *BuilderBase) buildGo(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Go SDK (version: %s)", version)

	// Module path for user's test
	testModule := fmt.Sprintf("github.com/temporalio/omes-workflow-tests/workflowtests/go/tests/%s", b.ProgramDir)

	// go.mod with replace directives for starter and user's test
	goModContents := fmt.Sprintf(`module github.com/temporalio/omes-workflow-tests/build

go 1.22

require %s v0.0.0
require github.com/temporalio/omes-workflow-tests/workflowtests/go/starter v0.0.0

replace %s => ../%s
replace github.com/temporalio/omes-workflow-tests/workflowtests/go/starter => ../../starter
`, testModule, testModule, b.ProgramDir)

	// Generated main.go imports user's package and calls Main()
	goMainContents := fmt.Sprintf(`package main

import %s "%s"

func main() {
	%s.Main()
}
`, b.ProgramDir, testModule, b.ProgramDir)

	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir:        absProjectDir,
		DirName:        b.OutputDir,
		Version:        version,
		GoModContents:  goModContents,
		GoMainContents: goMainContents,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Go program: %w", err)
	}

	return prog, nil
}

// preBuildLocalTypeScriptSDK builds a local TypeScript SDK if it uses pnpm and isn't built yet.
// This is needed because sdkbuild assumes npm, but sdk-typescript uses pnpm.
func (b *BuilderBase) preBuildLocalTypeScriptSDK(ctx context.Context, version string) error {
	sdkPath, err := filepath.Abs(version)
	if err != nil {
		return fmt.Errorf("failed to get absolute SDK path: %w", err)
	}

	buildOutput := filepath.Join(sdkPath, "packages", "client", "lib", "index.js")
	nodeModules := filepath.Join(sdkPath, "node_modules")

	// Check if we need to install dependencies
	needsInstall := false
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		needsInstall = true
	}

	if needsInstall {
		b.Logger.Info("Installing dependencies for local TypeScript SDK...")
		cmd := exec.CommandContext(ctx, "pnpm", "install")
		cmd.Dir = sdkPath
		cmd.Env = append(os.Environ(), "CI=true")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run pnpm install: %w", err)
		}
	}

	// Check if already fully built
	if _, err := os.Stat(buildOutput); err == nil {
		return nil // Already built
	}

	b.Logger.Info("Building local TypeScript SDK...")
	cmd := exec.CommandContext(ctx, "pnpm", "build")
	cmd.Dir = sdkPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run pnpm build: %w", err)
	}

	return nil
}
