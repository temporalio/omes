package sdkbuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const sdkImport = "go.temporal.io/sdk"

// BuildGoProgramOptions are options for BuildGoProgram.
type BuildGoProgramOptions struct {
	// Directory that will have a temporary directory created underneath
	BaseDir string
	// If not set, not put in go.mod which means go mod tidy will automatically
	// use latest. If set and does not start with a "v", it is assumed to be a
	// path, otherwise it is a specific version.
	Version string
	// The SDK Repository import to use. If unspecified we default to go.temporal.io/sdk
	// If specified version must also be provided
	SDKRepository string
	// Required go.mod contents
	GoModContents string
	// Required main.go contents
	GoMainContents string
	// If present, this directory is expected to exist beneath base dir. Otherwise
	// a temporary dir is created.
	DirName string
	// Optional set of tags to build with
	GoBuildTags []string
	// If present, applied to build commands before run. May be called multiple
	// times for a single build.
	ApplyToCommand func(context.Context, *exec.Cmd) error
}

// GoProgram is a Go-specific implementation of Program.
type GoProgram struct {
	dir string
}

var _ Program = (*GoProgram)(nil)

// BuildGoProgram builds a Go program. If completed successfully, this can be
// stored and re-obtained via GoProgramFromDir() with the Dir() value.
func BuildGoProgram(ctx context.Context, options BuildGoProgramOptions) (*GoProgram, error) {
	if options.BaseDir == "" {
		return nil, fmt.Errorf("base dir required")
	} else if options.GoModContents == "" {
		return nil, fmt.Errorf("go.mod contents required")
	} else if options.GoMainContents == "" {
		return nil, fmt.Errorf("main.go contents required")
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

	// Create go.mod
	goMod := options.GoModContents
	// If a version is specified, overwrite the SDK to use that
	if options.Version != "" || options.SDKRepository != "" {
		// If version does not start with a "v" we assume path unless the SDK repository is provided
		if options.SDKRepository != "" {
			if options.Version == "" {
				return nil, errors.New("Version must be provided alongside SDKRepository")
			}
			goMod += fmt.Sprintf("\nreplace %s => %s %s", sdkImport, options.SDKRepository, options.Version)
		} else if strings.HasPrefix(options.Version, "v") {
			goMod += fmt.Sprintf("\nreplace %s => %s %s", sdkImport, sdkImport, options.Version)
		} else {
			absVersion, err := filepath.Abs(options.Version)
			if err != nil {
				return nil, fmt.Errorf("version does not start with 'v' and cannot get abs dir: %w", err)
			}
			relVersion, err := filepath.Rel(dir, absVersion)
			if err != nil {
				return nil, fmt.Errorf("version does not start with 'v' and unable to relativize: %w", err)
			}
			goMod += fmt.Sprintf("\nreplace %s => %s", sdkImport, filepath.ToSlash(relVersion))
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		return nil, fmt.Errorf("failed writing go.mod: %w", err)
	}

	// Create main.go
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(options.GoMainContents), 0644); err != nil {
		return nil, fmt.Errorf("failed writing main.go: %w", err)
	}

	// Tidy it
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if options.ApplyToCommand != nil {
		if err := options.ApplyToCommand(ctx, cmd); err != nil {
			return nil, err
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed go mod tidy: %w", err)
	}

	// Build it
	exe := "program"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmdArgs := []string{"build", "-o", exe}
	for _, tag := range options.GoBuildTags {
		cmdArgs = append(cmdArgs, "-tags", tag)
	}
	cmd = exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if options.ApplyToCommand != nil {
		if err := options.ApplyToCommand(ctx, cmd); err != nil {
			return nil, err
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed go build: %w", err)
	}

	// All good
	success = true
	return &GoProgram{dir}, nil
}

// GoProgramFromDir recreates the Go program from a Dir() result of a
// BuildGoProgram().
func GoProgramFromDir(dir string) (*GoProgram, error) {
	// Quick sanity check on the presence of go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return nil, fmt.Errorf("failed finding go.mod in dir: %w", err)
	}
	return &GoProgram{dir}, nil
}

// Dir is the directory to run in.
func (g *GoProgram) Dir() string { return g.dir }

// NewCommand makes a new command for the given args.
func (g *GoProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	exe := "./program"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = g.dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd, nil
}
