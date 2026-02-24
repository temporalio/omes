package programbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"go.uber.org/zap"
)

// ProgramBuilder builds test projects into runnable programs using sdkbuild.
type ProgramBuilder struct {
	Language   string
	ProjectDir string // User's project directory
	BuildDir   string // Output build directory
	Logger     *zap.SugaredLogger
}

func displayVersion(version string) string {
	if version == "" {
		return "<auto>"
	}
	return version
}

// BuildProgram builds the project and returns a runnable program.
// If version is empty, it is auto-detected from project files.
func (b *ProgramBuilder) BuildProgram(ctx context.Context, version string) (sdkbuild.Program, error) {
	resolvedVersion, err := b.resolveSDKVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	absProjectDir, err := filepath.Abs(b.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	var prog sdkbuild.Program
	switch b.Language {
	case "python":
		prog, err = b.buildPython(ctx, absProjectDir, resolvedVersion)
	case "typescript":
		prog, err = b.buildTypeScript(ctx, absProjectDir, resolvedVersion)
	case "go":
		prog, err = b.buildGo(ctx, absProjectDir, resolvedVersion)
	default:
		return nil, fmt.Errorf("unsupported language: %s", b.Language)
	}
	if err != nil {
		return nil, err
	}
	return prog, nil
}

func (b *ProgramBuilder) resolveSDKVersion(ctx context.Context, version string) (string, error) {
	if version != "" {
		return version, nil
	}
	detectedVersion, err := detectSDKVersion(ctx, b.Language, b.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("failed to detect SDK version (use --version to specify): %w", err)
	}

	b.Logger.Infof("Auto-detected SDK version: %s", detectedVersion)
	return detectedVersion, nil
}

func (b *ProgramBuilder) buildPython(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Python SDK (version: %s)", displayVersion(version))

	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir: absProjectDir,
		Version: version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Python program: %w", err)
	}

	return prog, nil
}

// buildTypeScript compiles the test project and shared omes-starter source together.
// The starter is included via tsconfig path aliases (not a real npm dependency).
// Local SDK paths are pre-built with pnpm since sdkbuild assumes npm.
func (b *ProgramBuilder) buildTypeScript(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building TypeScript SDK (version: %s)", displayVersion(version))

	// Pre-build local SDK if needed (sdk-typescript uses pnpm, sdkbuild assumes npm)
	if strings.ContainsAny(version, `/\`) {
		if err := b.preBuildLocalTypeScriptSDK(ctx, version); err != nil {
			return nil, err
		}
	}

	baseDir := filepath.Dir(absProjectDir)
	projectName := filepath.Base(absProjectDir)

	// Paths are relative to the sdkbuild temp dir (a sibling of the project dir under tests/)
	relProjectDir := "../" + projectName
	relSharedSrc := "../../src"

	// TSConfigPaths: runtime alias resolution (node -r tsconfig-paths/register)
	// Includes: TypeScript compiler scope (which .ts files to compile)
	tsConfigPaths := map[string][]string{
		"@temporalio/omes-starter":   {"./tslib/src/index.js"},
		"@temporalio/omes-starter/*": {"./tslib/src/*"},
	}
	includes := []string{
		relProjectDir + "/**/*.ts",
		relSharedSrc + "/**/*.ts",
	}

	// Dependencies required by the shared omes-starter source
	moreDeps := map[string]string{
		"express":        "^4.18.0",
		"@types/express": "^4.17.0",
		"commander":      "^8.3.0",
	}

	prog, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir:          baseDir,
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

// buildGo generates a temporary Go module that imports both the test project and shared starter,
// linked via replace directives. sdkbuild handles SDK version pinning via go.mod replace.
func (b *ProgramBuilder) buildGo(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Go SDK (version: %s)", displayVersion(version))

	projectName := filepath.Base(absProjectDir)
	baseDir := filepath.Dir(absProjectDir)
	pkgName := strings.ReplaceAll(projectName, "-", "")

	testModule := fmt.Sprintf("github.com/temporalio/omes-workflow-tests/workflowtests/go/tests/%s", projectName)

	goModContents := fmt.Sprintf(`module github.com/temporalio/omes-workflow-tests/build

go 1.22

require %s v0.0.0
require github.com/temporalio/omes-workflow-tests/workflowtests/go/starter v0.0.0

replace %s => ../%s
replace github.com/temporalio/omes-workflow-tests/workflowtests/go/starter => ../../starter
`, testModule, testModule, projectName)

	goMainContents := fmt.Sprintf(`package main

import %s "%s"

func main() {
	%s.Main()
}
`, pkgName, testModule, pkgName)

	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir:        baseDir,
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
func (b *ProgramBuilder) preBuildLocalTypeScriptSDK(ctx context.Context, version string) error {
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
