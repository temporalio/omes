package project

import (
	"context"
	"fmt"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers"
	"go.uber.org/zap"
)

// buildProject builds a project test program for the given language.
func buildProject(ctx context.Context, repoRoot string, p projectScenarioOptions, logger *zap.SugaredLogger) (sdkbuild.Program, error) {
	b := workers.Builder{
		DirName:     fmt.Sprintf("project-build-runner-%s", p.projectName),
		SdkOptions:  p.sdkOpts,
		ProjectName: p.projectName,
		Logger:      logger,
	}

	baseDir := workers.BaseDir(repoRoot, p.sdkOpts.Language)
	switch p.sdkOpts.Language {
	case clioptions.LangPython, clioptions.LangTypeScript:
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
	case clioptions.LangTypeScript:
		return sdkbuild.TypeScriptProgramFromDir(dir)
	default:
		return nil, fmt.Errorf("prebuilt projects not supported for language: %s", lang)
	}
}
