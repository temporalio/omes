package programbuild

import (
	"context"
	"fmt"
	"os"
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
// Workflow tests support only Go projects.
func (b *ProgramBuilder) BuildProgram(ctx context.Context, version string) (sdkbuild.Program, error) {
	absProjectDir, err := filepath.Abs(b.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if b.Language != "go" {
		return nil, fmt.Errorf("unsupported language for workflow tests: %s (only go is supported)", b.Language)
	}

	prog, err := b.buildGo(ctx, absProjectDir, version)
	if err != nil {
		return nil, err
	}
	return prog, nil
}

// buildGo generates a temporary Go module that imports both the test project and shared harness,
// linked via replace directives. sdkbuild handles SDK version pinning via go.mod replace.
func (b *ProgramBuilder) buildGo(ctx context.Context, absProjectDir, version string) (sdkbuild.Program, error) {
	b.Logger.Infof("Building Go SDK (version: %s)", displayVersion(version))

	projectName := filepath.Base(absProjectDir)
	baseDir := filepath.Dir(absProjectDir)
	pkgName := strings.ReplaceAll(projectName, "-", "")

	testModule := fmt.Sprintf("github.com/temporalio/omes/projecttests/go/tests/%s", projectName)

	goModContents := fmt.Sprintf(`module github.com/temporalio/omes/build

go 1.22

require %s v0.0.0
require github.com/temporalio/omes/projecttests/go/harness v0.0.0

replace %s => ../%s
replace github.com/temporalio/omes/projecttests/go/harness => ../../harness
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
