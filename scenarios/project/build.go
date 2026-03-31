package project

import (
	"context"
	"fmt"
	"os"
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
	// Build output is placed in the parent directory of ProjectDir.
	ProjectDir string
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
	case clioptions.LangDotNet:
		return buildDotNet(ctx, opts)
	default:
		return nil, fmt.Errorf("unsupported language for project builds: %s", opts.Language)
	}
}

func buildGo(ctx context.Context, opts BuildOptions) (sdkbuild.Program, error) {
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project dir: %w", err)
	}

	baseDir := filepath.Dir(absProjectDir)
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
	buildDir := filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating build dir: %w", err)
	}

	buildOpts := sdkbuild.BuildGoProgramOptions{
		BaseDir:        baseDir,
		DirName:        dirName,
		Version:        opts.Version,
		GoModContents:  goMod,
		GoMainContents: goMain,
	}
	if opts.Logger != nil {
		buildOpts.Stdout = &logWriter{logger: opts.Logger}
		buildOpts.Stderr = &logWriter{logger: opts.Logger}
		opts.Logger.Infof("Building project at %s", baseDir)
	}

	prog, err := sdkbuild.BuildGoProgram(ctx, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to build project: %w", err)
	}
	return prog, nil
}

func buildDotNet(ctx context.Context, opts BuildOptions) (sdkbuild.Program, error) {
	absProjectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project dir: %w", err)
	}

	if opts.Logger != nil {
		opts.Logger.Infof("Building .NET project at %s", absProjectDir)
	}

	// Find the project's csproj file (<ProjectName>.csproj)
	projectName := filepath.Base(absProjectDir)
	absCsprojFile := filepath.Join(absProjectDir, projectName+".csproj")
	if _, err := os.Stat(absCsprojFile); err != nil {
		return nil, fmt.Errorf("cannot find %s.csproj in %s: %w", projectName, absProjectDir, err)
	}

	// Use sdkbuild to generate a thin wrapper that calls into the project.
	// The generated program.csproj references the project as a ProjectReference,
	// so the project's own csproj handles all dependencies (Temporalio, harness, etc.).
	// sdkbuild handles SDK version/source builds via TemporalioProjectReference.
	baseDir := filepath.Dir(absProjectDir)
	dirName := fmt.Sprintf("project-build-%s", projectName)
	buildDir := filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating build dir: %w", err)
	}

	buildOpts := sdkbuild.BuildDotNetProgramOptions{
		BaseDir:         baseDir,
		DirName:         dirName,
		Version:         opts.Version,
		ProgramContents: fmt.Sprintf("return await %s.RunAsync(args);", projectName),
		CsprojContents: `<Project Sdk="Microsoft.NET.Sdk">
			<PropertyGroup>
				<OutputType>Exe</OutputType>
				<TargetFramework>net8.0</TargetFramework>
			</PropertyGroup>
			<ItemGroup>
				<ProjectReference Include="` + absCsprojFile + `" />
			</ItemGroup>
		</Project>`,
	}
	if opts.Logger != nil {
		buildOpts.Stdout = &logWriter{logger: opts.Logger}
		buildOpts.Stderr = &logWriter{logger: opts.Logger}
	}

	prog, err := sdkbuild.BuildDotNetProgram(ctx, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to build .NET project: %w", err)
	}
	return prog, nil
}

// LoadPrebuilt loads an already-built project binary from the given directory.
// This is used in Docker containers where the binary was built during image creation.
func LoadPrebuilt(dir string, lang clioptions.Language) (sdkbuild.Program, error) {
	switch lang {
	case clioptions.LangGo:
		return sdkbuild.GoProgramFromDir(dir)
	case clioptions.LangDotNet:
		return sdkbuild.DotNetProgramFromDir(dir)
	default:
		return nil, fmt.Errorf("prebuilt projects not supported for language: %s", lang)
	}
}

type logWriter struct {
	logger *zap.SugaredLogger
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Debug(strings.TrimSpace(string(p)))
	return len(p), nil
}
