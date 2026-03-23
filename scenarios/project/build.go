package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"go.uber.org/zap"
)

// BuildOptions configures a project build.
type BuildOptions struct {
	// Language for the project.
	Language clioptions.Language
	// ProjectDir is the absolute or relative path to the project directory.
	ProjectDir string
	// BaseDir is the directory where the build output will be created.
	BaseDir string
	// Version is the SDK version to use (empty = auto).
	Version string
	// Logger for build output.
	Logger *zap.SugaredLogger
}

// Build builds a project test binary for the given language.
func Build(ctx context.Context, opts BuildOptions) (sdkbuild.Program, error) {
	switch opts.Language {
	case clioptions.LangGo:
		return buildGo(ctx, opts)
	default:
		return nil, fmt.Errorf("unsupported language for project builds: %s", opts.Language)
	}
}

func buildGo(ctx context.Context, opts BuildOptions) (sdkbuild.Program, error) {
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project dir: %w", err)
	}

	projectName := filepath.Base(absProjectDir)
	pkgName := strings.ReplaceAll(projectName, "-", "")
	testModule := fmt.Sprintf("github.com/temporalio/omes/workers/go/projects/tests/%s", projectName)
	harnessModule := "github.com/temporalio/omes/workers/go/projects/harness"
	apiModule := "github.com/temporalio/omes/workers/go/projects/api"

	absHarness, _ := filepath.Abs(filepath.Join(absProjectDir, "..", "..", "harness"))
	absAPI, _ := filepath.Abs(filepath.Join(absProjectDir, "..", "..", "api"))

	goMod := fmt.Sprintf(`module github.com/temporalio/omes/build

go 1.22

require %s v0.0.0

replace %s => %s
replace %s => %s
replace %s => %s
`, testModule,
		testModule, absProjectDir,
		harnessModule, absHarness,
		apiModule, absAPI)

	goMain := fmt.Sprintf(`package main

import %s "%s"

func main() {
	%s.Main()
}
`, pkgName, testModule, pkgName)

	dirName := fmt.Sprintf("project-build-%s", projectName)
	buildDir := filepath.Join(opts.BaseDir, dirName)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating build dir: %w", err)
	}

	buildOpts := sdkbuild.BuildGoProgramOptions{
		BaseDir:        opts.BaseDir,
		DirName:        dirName,
		Version:        opts.Version,
		GoModContents:  goMod,
		GoMainContents: goMain,
		ApplyToCommand: func(_ context.Context, cmd *exec.Cmd) error {
			if len(cmd.Args) > 1 && cmd.Args[1] == "build" {
				cmd.Args = append(cmd.Args, "-buildvcs=false")
			}
			return nil
		},
	}
	if opts.Logger != nil {
		buildOpts.Stdout = &logWriter{logger: opts.Logger}
		buildOpts.Stderr = &logWriter{logger: opts.Logger}
		opts.Logger.Infof("Building project at %s", opts.BaseDir)
	}

	prog, err := sdkbuild.BuildGoProgram(ctx, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to build project: %w", err)
	}
	return prog, nil
}

type logWriter struct {
	logger *zap.SugaredLogger
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Debug(strings.TrimSpace(string(p)))
	return len(p), nil
}
