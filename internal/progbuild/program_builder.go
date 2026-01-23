package progbuild

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
	SDKVersion string
	ProjectDir string // User's project directory
	BuildDir   string // Output build directory (TypeScript only)
	Logger     *zap.SugaredLogger
}

// BuildProgram creates a sdkbuild.Program for the user's project.
// For Python: uses <module>/__main__.py convention, run with prog.NewCommand(ctx, "module", "client", ...)
// For TypeScript: compiles user project with shared source, run with prog.NewCommand(ctx, "tslib/main.js", "client", ...)
func (b *ProgramBuilder) BuildProgram(ctx context.Context) (sdkbuild.Program, error) {
	// Get absolute path to user's project
	absProjectDir, err := filepath.Abs(b.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	switch b.Language {
	case "python":
		return b.buildPython(ctx, absProjectDir)
	case "typescript":
		return b.buildTypeScript(ctx, absProjectDir)
	case "go":
		return b.buildGo(ctx, absProjectDir)
	default:
		return nil, fmt.Errorf("unsupported language: %s", b.Language)
	}
}

func (b *ProgramBuilder) buildPython(ctx context.Context, absProjectDir string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Python SDK (version: %s)", b.SDKVersion)

	// Use sdkbuild directly - it creates temp dir, installs SDK, adds user project as editable
	// User project must have src/__main__.py as entry point
	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir: absProjectDir,
		// No DirName - let sdkbuild create temp dir (no caching for now)
		Version: b.SDKVersion,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Python program: %w", err)
	}

	return prog, nil
}

func (b *ProgramBuilder) buildTypeScript(ctx context.Context, absProjectDir string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building TypeScript SDK (version: %s)", b.SDKVersion)

	// TODO(thomas): If version is a local path, pre-build if needed (handles pnpm projects), sdkbuild still tries to build with npm
	if strings.ContainsAny(b.SDKVersion, `/\`) {
		if err := b.preBuildLocalTypeScriptSDK(ctx); err != nil {
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
		// No DirName - let sdkbuild create temp dir (no caching for now)
		Version:          b.SDKVersion,
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

func (b *ProgramBuilder) buildGo(ctx context.Context, absProjectDir string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Go SDK (version: %s)", b.SDKVersion)

	projectName := filepath.Base(absProjectDir)
	baseDir := filepath.Dir(absProjectDir) // tests/
	pkgName := strings.ReplaceAll(projectName, "-", "")

	// Module path for user's test
	testModule := fmt.Sprintf("github.com/temporalio/omes-workflow-tests/workflowtests/go/tests/%s", projectName)

	// go.mod with replace directives for starter and user's test
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
		Version:        b.SDKVersion,
		GoModContents:  goModContents,
		GoMainContents: goMainContents,
		ApplyToCommand: func(_ context.Context, cmd *exec.Cmd) error {
			// Add -buildvcs=false to avoid VCS stamping issues in temp directories
			for i, arg := range cmd.Args {
				if arg == "build" {
					cmd.Args = append(cmd.Args[:i+1], append([]string{"-buildvcs=false"}, cmd.Args[i+1:]...)...)
					break
				}
			}
			return nil
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build Go program: %w", err)
	}

	return prog, nil
}

// preBuildLocalTypeScriptSDK builds a local TypeScript SDK if it uses pnpm and isn't built yet.
// This is needed because sdkbuild assumes npm, but sdk-typescript uses pnpm.
func (b *ProgramBuilder) preBuildLocalTypeScriptSDK(ctx context.Context) error {
	sdkPath, err := filepath.Abs(b.SDKVersion)
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
