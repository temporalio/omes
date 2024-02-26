package sdkbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// BuildPythonProgramOptions are options for BuildPythonProgram.
type BuildPythonProgramOptions struct {
	// Directory that will have a temporary directory created underneath. This
	// should be a Poetry project with a pyproject.toml.
	BaseDir string
	// Required version. If it contains a slash it is assumed to be a path with
	// a single wheel in the dist directory. Otherwise it is a specific version
	// (with leading "v" is trimmed if present).
	Version string
	// Required Poetry dependency name of BaseDir.
	DependencyName string
	// If present, this directory is expected to exist beneath base dir. Otherwise
	// a temporary dir is created.
	DirName string
	// If present, applied to build commands before run. May be called multiple
	// times for a single build.
	ApplyToCommand func(context.Context, *exec.Cmd) error
}

// PythonProgram is a Python-specific implementation of Program.
type PythonProgram struct {
	dir string
}

var _ Program = (*PythonProgram)(nil)

// BuildPythonProgram builds a Python program. If completed successfully, this
// can be stored and re-obtained via PythonProgramFromDir() with the Dir() value
// (but the entire BaseDir must be present too).
func BuildPythonProgram(ctx context.Context, options BuildPythonProgramOptions) (*PythonProgram, error) {
	if options.BaseDir == "" {
		return nil, fmt.Errorf("base dir required")
	} else if options.Version == "" {
		return nil, fmt.Errorf("version required")
	} else if _, err := os.Stat(filepath.Join(options.BaseDir, "pyproject.toml")); err != nil {
		return nil, fmt.Errorf("failed finding pyproject.toml in base dir: %w", err)
	} else if options.DependencyName == "" {
		return nil, fmt.Errorf("dependency name required")
	}

	// Create temp dir if needed that we will remove if creating is unsuccessful
	success := false
	var dir string
	if options.DirName != "" {
		dir = filepath.Join(options.BaseDir, options.DirName)
	} else {
		var err error
		dir, err = os.MkdirTemp(options.BaseDir, "program-")
		if err != nil {
			return nil, fmt.Errorf("failed making temp dir: %w", err)
		}
		defer func() {
			if !success {
				// Intentionally swallow error
				_ = os.RemoveAll(dir)
			}
		}()
	}

	// Use semantic version or path if it's a path
	versionStr := strconv.Quote(strings.TrimPrefix(options.Version, "v"))
	if strings.ContainsAny(options.Version, `/\`) {
		// We expect a dist/ directory with a single whl file present
		wheels, err := filepath.Glob(filepath.Join(options.Version, "dist/*.whl"))
		if err != nil {
			return nil, fmt.Errorf("failed glob wheel lookup: %w", err)
		} else if len(wheels) != 1 {
			return nil, fmt.Errorf("expected single dist wheel, found %v", wheels)
		}
		absWheel, err := filepath.Abs(wheels[0])
		if err != nil {
			return nil, fmt.Errorf("unable to make wheel path absolute: %w", err)
		}
		// There's a strange bug in Poetry or somewhere deeper where, on Windows,
		// the single drive letter has to be capitalized
		if runtime.GOOS == "windows" && absWheel[1] == ':' {
			absWheel = strings.ToUpper(absWheel[:1]) + absWheel[1:]
		}
		versionStr = "{ path = " + strconv.Quote(absWheel) + " }"
	}
	pyProjectTOML := `
[tool.poetry]
name = "python-program-` + filepath.Base(dir) + `"
version = "0.1.0"
description = "Temporal SDK Python Test"
authors = ["Temporal Technologies Inc <sdk@temporal.io>"]

[tool.poetry.dependencies]
python = "^3.8"
temporalio = ` + versionStr + `
` + options.DependencyName + ` = { path = "../" }

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyProjectTOML), 0644); err != nil {
		return nil, fmt.Errorf("failed writing pyproject.toml: %w", err)
	}

	// Install
	cmd := exec.CommandContext(ctx, "poetry", "install", "--no-dev", "--no-root", "-v")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if options.ApplyToCommand != nil {
		if err := options.ApplyToCommand(ctx, cmd); err != nil {
			return nil, err
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed installing: %w", err)
	}

	success = true
	return &PythonProgram{dir}, nil
}

// PythonProgramFromDir recreates the Python program from a Dir() result of a
// BuildPythonProgram(). Note, the base directory of dir when it was built must
// also be present.
func PythonProgramFromDir(dir string) (*PythonProgram, error) {
	// Quick sanity check on the presence of pyproject.toml here _and_ in base
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err != nil {
		return nil, fmt.Errorf("failed finding pyproject.toml in dir: %w", err)
	} else if _, err := os.Stat(filepath.Join(dir, "../pyproject.toml")); err != nil {
		return nil, fmt.Errorf("failed finding pyproject.toml in base dir: %w", err)
	}
	return &PythonProgram{dir}, nil
}

// Dir is the directory to run in.
func (p *PythonProgram) Dir() string { return p.dir }

// NewCommand makes a new Poetry command. The first argument needs to be the
// name of the module.
func (p *PythonProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	args = append([]string{"run", "python", "-m"}, args...)
	cmd := exec.CommandContext(ctx, "poetry", args...)
	cmd.Dir = p.dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd, nil
}
