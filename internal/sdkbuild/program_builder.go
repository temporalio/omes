package sdkbuild

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"go.uber.org/zap"
)

// ProgramBuilder builds programs using sdkbuild.
// The user writes their own main.py/main.ts that calls omes_starter.run(client=..., worker=...).
// This builder just builds the SDK and returns a program that can run the user's entry file.
type ProgramBuilder struct {
	Language   string
	SDKVersion string
	ProjectDir string // User's project directory
	BuildDir   string // Output build directory (cached)
	Logger     *zap.SugaredLogger
}

// BuildProgram creates a sdkbuild.Program for the given entry file.
// The entryFile is the path to the user's main file (e.g., "main.py" or "main.ts").
// The returned program can run the entry file with any subcommand:
//
//	prog.NewCommand(ctx, "client", "--port", "8080", ...)  // runs: python main.py client --port 8080 ...
//	prog.NewCommand(ctx, "worker", "--task-queue", "q")    // runs: python main.py worker --task-queue q
func (b *ProgramBuilder) BuildProgram(ctx context.Context, entryFile string) (sdkbuild.Program, error) {
	// Get absolute path to user's project
	absProjectDir, err := filepath.Abs(b.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Compute cache directory
	// For TypeScript, place build dir adjacent to project for simpler relative paths
	// For other languages, use /tmp/omes-sdk-cache/{language}/{version}/{hash(projectDir)}
	if b.BuildDir == "" {
		projectHash := hashDir(absProjectDir)
		if b.Language == "typescript" {
			// Place build dir as sibling to project: {projectParent}/.omes-build-{hash}
			b.BuildDir = filepath.Join(filepath.Dir(absProjectDir), ".omes-build-"+projectHash)
		} else {
			b.BuildDir = filepath.Join(os.TempDir(), "omes-sdk-cache", b.Language, b.SDKVersion, projectHash)
		}
	} else {
		// Ensure explicit build dir is absolute
		absBuildDir, err := filepath.Abs(b.BuildDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute build dir: %w", err)
		}
		b.BuildDir = absBuildDir
	}

	// Create build directory if needed
	if err := os.MkdirAll(b.BuildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}

	switch b.Language {
	case "python":
		return b.buildPython(ctx, absProjectDir, entryFile)
	case "typescript":
		return b.buildTypeScript(ctx, absProjectDir, entryFile)
	default:
		return nil, fmt.Errorf("unsupported language: %s", b.Language)
	}
}

func (b *ProgramBuilder) buildPython(ctx context.Context, absProjectDir, entryFile string) (sdkbuild.Program, error) {
	// Check if SDK is already built (venv exists)
	venvDir := ".venv"
	venvPath := filepath.Join(b.BuildDir, venvDir)

	if !buildExists(venvPath) {
		b.Logger.Infof("Building %s SDK at %s (version: %s)", b.Language, b.BuildDir, b.SDKVersion)

		// Find omes_starter path
		omesStarterPath := findOmesStarterPath()

		// Generate pyproject.toml for sdkbuild
		var pyprojectContent string
		if omesStarterPath != "" {
			pyprojectContent = fmt.Sprintf(`[project]
name = "omes-wrapper"
version = "0.1.0"
requires-python = ">=3.10"
dependencies = [
    "aiohttp>=3.9.0",
    "omes-starter",
]

[tool.uv.sources]
omes-starter = { path = %q }
`, omesStarterPath)
		} else {
			pyprojectContent = `[project]
name = "omes-wrapper"
version = "0.1.0"
requires-python = ">=3.10"
dependencies = [
    "aiohttp>=3.9.0",
]
`
		}

		pyprojectPath := filepath.Join(b.BuildDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write pyproject.toml: %w", err)
		}

		// Create the venv directory (sdkbuild expects it to exist)
		if err := os.MkdirAll(venvPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create venv directory: %w", err)
		}

		_, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
			BaseDir: b.BuildDir,
			DirName: venvDir,
			Version: b.SDKVersion,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build Python program: %w", err)
		}
	} else {
		b.Logger.Infof("Using cached SDK build at %s", b.BuildDir)
	}

	return &pythonProgram{
		buildDir:   b.BuildDir,
		venvDir:    venvPath,
		projectDir: absProjectDir,
		entryFile:  entryFile,
	}, nil
}

func (b *ProgramBuilder) buildTypeScript(ctx context.Context, absProjectDir, entryFile string) (sdkbuild.Program, error) {
	// Check if SDK is already built (node_modules exists)
	nodeModulesPath := filepath.Join(b.BuildDir, "node_modules")

	if !buildExists(nodeModulesPath) {
		b.Logger.Infof("Building %s SDK at %s (version: %s)", b.Language, b.BuildDir, b.SDKVersion)

		// Find omes-starter path
		omesStarterPath := findOmesStarterTSPath()

		// Build using sdkbuild
		baseDir := filepath.Dir(b.BuildDir)
		dirName := filepath.Base(b.BuildDir)

		// Compute relative path from build dir to project dir for includes
		relProjectDir, err := filepath.Rel(b.BuildDir, absProjectDir)
		if err != nil {
			return nil, fmt.Errorf("failed to compute relative path: %w", err)
		}

		// Set up dependencies - add omes-starter as file dependency if found
		moreDeps := map[string]string{}
		if omesStarterPath != "" {
			moreDeps["@temporalio/omes-starter"] = "file:" + omesStarterPath
		}

		// TSConfigPaths - sdkbuild requires at least one entry
		// Don't override @temporalio/omes-starter - let TypeScript resolve it from node_modules
		// Just provide a dummy entry to satisfy sdkbuild's requirement
		tsConfigPaths := map[string][]string{
			"@temporalio/omes-dummy": {"node_modules/@temporalio/client"},
		}

		_, err = sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
			BaseDir:          baseDir,
			DirName:          dirName,
			Version:          b.SDKVersion,
			TSConfigPaths:    tsConfigPaths,
			Includes:         []string{relProjectDir + "/**/*.ts"},
			MoreDependencies: moreDeps,
			Stdout:           os.Stdout,
			Stderr:           os.Stderr,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build TypeScript program: %w", err)
		}
	} else {
		b.Logger.Infof("Using cached SDK build at %s", b.BuildDir)
	}

	return &typescriptProgram{
		buildDir:   b.BuildDir,
		projectDir: absProjectDir,
		entryFile:  entryFile,
	}, nil
}

// findOmesStarterTSPath attempts to locate the omes-starter TypeScript package.
func findOmesStarterTSPath() string {
	candidates := []string{
		"workflowtests/typescript",
		"./workflowtests/typescript",
	}

	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "workflowtests/typescript"),
			filepath.Join(execDir, "../workflowtests/typescript"),
		)
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		// Check if package.json exists
		pkgPath := filepath.Join(absPath, "package.json")
		if _, err := os.Stat(pkgPath); err == nil {
			return absPath
		}
	}

	return ""
}

// pythonProgram runs the user's entry file from the venv created by sdkbuild
type pythonProgram struct {
	buildDir   string
	venvDir    string
	projectDir string
	entryFile  string // user's main.py
}

func (p *pythonProgram) Dir() string {
	return p.buildDir
}

func (p *pythonProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Run the user's entry file with the provided args
	// Use --project to force uv to use our build directory's pyproject.toml,
	// not any pyproject.toml in the user's project directory.
	mainFile := filepath.Join(p.projectDir, p.entryFile)
	allArgs := append([]string{"run", "--project", p.buildDir, "python", mainFile}, args...)
	cmd := exec.CommandContext(ctx, "uv", allArgs...)

	// Run from the venv directory where sdkbuild installed dependencies
	cmd.Dir = p.venvDir

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// typescriptProgram runs the user's compiled entry file
type typescriptProgram struct {
	buildDir   string
	projectDir string
	entryFile  string // user's main.ts (will run compiled .js)
}

func (p *typescriptProgram) Dir() string {
	return p.buildDir
}

func (p *typescriptProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Run the compiled entry file from sdkbuild's tslib output
	// sdkbuild compiles to tslib/ with the common prefix stripped
	// so main.ts compiles to tslib/main.js
	jsFile := strings.TrimSuffix(p.entryFile, ".ts") + ".js"
	mainFile := filepath.Join("tslib", jsFile)

	// Use tsconfig-paths/register to resolve path aliases at runtime
	allArgs := append([]string{"-r", "tsconfig-paths/register", mainFile}, args...)
	cmd := exec.CommandContext(ctx, "node", allArgs...)
	cmd.Dir = p.buildDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// hashDir returns a short hash of the directory path for cache keying
func hashDir(dir string) string {
	h := sha256.Sum256([]byte(dir))
	return fmt.Sprintf("%x", h[:8])
}

// findOmesStarterPath attempts to locate the omes_starter Python package.
// It checks several common locations relative to the omes binary or current directory.
func findOmesStarterPath() string {
	// Try to find the omes_starter package in common locations
	candidates := []string{
		// Relative to current working directory (when running from omes repo)
		"workflowtests/python",
		"./workflowtests/python",
	}

	// Also try relative to the executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "workflowtests/python"),
			filepath.Join(execDir, "../workflowtests/python"),
		)
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		// Check if omes_starter/__init__.py exists
		initPath := filepath.Join(absPath, "omes_starter", "__init__.py")
		if _, err := os.Stat(initPath); err == nil {
			return absPath
		}
	}

	return ""
}

// buildExists checks if a build directory exists.
func buildExists(buildDir string) bool {
	_, err := os.Stat(buildDir)
	return err == nil
}

