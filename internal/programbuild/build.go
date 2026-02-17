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

// ProgramBuilder builds programs using sdkbuild.
// For Python: user project must have <module>/__main__.py as entry point
// For TypeScript: sdkbuild compiles user project + shared omes-starter source
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

// BuildProgram creates a sdkbuild.Program for the user's project.
// For Python: uses <module>/__main__.py convention, run with prog.NewCommand(ctx, "module", "client", ...)
// For TypeScript: compiles user project with shared source, run with prog.NewCommand(ctx, "tslib/main.js", "client", ...)
// BuildProgram builds the program and returns it along with the resolved SDK version.
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

// buildPython builds via sdkbuild in a temp directory adjacent to the project.
//
// sdkbuild creates a temp environment, installs the resolved temporalio version (or local
// wheel), and installs the test project as editable. Transitive dependencies (including
// omes-starter, if declared by the test project) are resolved by uv during sync.
//
// Version is pre-resolved by BuildProgram before this function is called:
//   - version="1.21.0" → sdkbuild runs: uv add temporalio==1.21.0
//   - version="../sdk-python" → sdkbuild builds a wheel from the local SDK and installs it
func (b *ProgramBuilder) buildPython(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Python SDK (version: %s)", displayVersion(version))

	// Use sdkbuild directly - it creates temp dir, installs SDK, adds user project as editable
	// User project must have src/__main__.py as entry point
	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir: absProjectDir,
		// No DirName - let sdkbuild create temp dir
		Version: version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Python program: %w", err)
	}

	return prog, nil
}

// buildTypeScript builds via sdkbuild in a temp directory. The shared starter source
// is compiled alongside the test project via tsconfig paths and includes.
//
// Version is pre-resolved by BuildProgram (detected from project when --version flag not
// provided). sdkbuild then pins generated @temporalio/* deps to that resolved version:
//   - version="1.3.0" → sdkbuild generates package.json with @temporalio/*: "1.3.0"
//   - version="../sdk-typescript" → sdkbuild pre-builds the local SDK, then generates
//     package.json with file: URLs pointing to local SDK packages
//
// Local SDK paths using pnpm are pre-built before sdkbuild (which assumes npm).
func (b *ProgramBuilder) buildTypeScript(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building TypeScript SDK (version: %s)", displayVersion(version))

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
		BaseDir: baseDir,
		// No DirName - let sdkbuild create temp dir
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

// buildGo creates a temporary Go module that composes the test project and shared starter.
//
// Build graph: generated build module → test module + starter module (via replace directives).
// Both test and starter have their own go.temporal.io/sdk requirement. go mod tidy resolves
// a single effective SDK version across both via Go's MVS (minimum version selection).
//
// SDK version precedence:
//   - version="" → go mod tidy resolves from test/starter go.mod requirements
//   - version="1.31.0" → sdkbuild appends: replace go.temporal.io/sdk => go.temporal.io/sdk v1.31.0
//   - version="../sdk-go" → sdkbuild appends: replace go.temporal.io/sdk => <relative path>
//
// No hard check that starter and test SDK versions match. The effective SDK version is
// whatever the final module graph resolves to (or what --version forces via replace).
func (b *ProgramBuilder) buildGo(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Go SDK (version: %s)", displayVersion(version))

	projectName := filepath.Base(absProjectDir)
	baseDir := filepath.Dir(absProjectDir) // tests/
	pkgName := strings.ReplaceAll(projectName, "-", "")

	testModule := fmt.Sprintf("github.com/temporalio/omes-workflow-tests/workflowtests/go/tests/%s", projectName)

	goModContents := fmt.Sprintf(`module github.com/temporalio/omes-workflow-tests/build

go 1.22

require %s v0.0.0
require github.com/temporalio/omes-workflow-tests/workflowtests/go/starter v0.0.0

replace %s => ../%s
replace github.com/temporalio/omes-workflow-tests/workflowtests/go/starter => ../../starter
`, testModule, testModule, projectName)

	// Generated main.go imports user's package and calls Main()
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
