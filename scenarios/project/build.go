package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers"
	"go.uber.org/zap"
)

// buildProject builds a project test program for the given language.
func buildProject(ctx context.Context, repoRoot string, p projectScenarioOptions, logger *zap.SugaredLogger) (sdkbuild.Program, error) {
	baseDir := workers.BaseDir(repoRoot, p.sdkOpts.Language)
	projectDir := workers.ProjectDir(baseDir, p.projectName)
	absProjDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project-dir: %w", err)
	}

	dirName := fmt.Sprintf("project-build-runner-%s", p.projectName)
	outputDir := filepath.Join(absProjDir, dirName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating build dir: %w", err)
	}

	b := workers.Builder{
		DirName:     filepath.Base(outputDir),
		SdkOptions:  p.sdkOpts,
		ProjectName: p.projectName,
		Logger:      logger,
	}

	switch p.sdkOpts.Language {
	case clioptions.LangPython:
		return b.Build(ctx, baseDir)
	default:
		return nil, fmt.Errorf("unsupported language for project builds: %s", b.SdkOptions.Language)
	}
}

// loadPrebuilt loads an already-built project program from the given directory.
func loadPrebuilt(dir string, lang clioptions.Language) (sdkbuild.Program, error) {
	switch lang {
	case clioptions.LangPython:
		return sdkbuild.PythonProgramFromDir(dir)
	default:
		return nil, fmt.Errorf("prebuilt projects not supported for language: %s", lang)
	}
}
