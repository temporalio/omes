package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workerctl"
	"github.com/temporalio/omes/workers"
	"go.uber.org/zap"
)

// buildProject builds a project test program for the given language.
func buildProject(ctx context.Context, repoRoot string, p projectScenarioOptions, logger *zap.SugaredLogger) (sdkbuild.Program, error) {
	b := workerctl.Builder{
		DirName:    fmt.Sprintf("project-build-runner-%s", p.projectName),
		SdkOptions: p.sdkOpts,
		Logger:     logger,
	}

	return b.Build(ctx, workers.BaseDir(repoRoot, p.sdkOpts.Language))
}

// loadPrebuilt loads an already-built project program from the given directory.
func loadPrebuilt(dir, repoRoot string, lang clioptions.Language) (sdkbuild.Program, error) {
	switch lang {
	case clioptions.LangGo:
		return sdkbuild.GoProgramFromDir(dir)
	case clioptions.LangPython:
		return sdkbuild.PythonProgramFromDir(dir)
	case clioptions.LangJava:
		return sdkbuild.JavaProgramFromDir(dir)
	case clioptions.LangTypeScript:
		return sdkbuild.TypeScriptProgramFromDir(dir)
	case clioptions.LangDotNet:
		return sdkbuild.DotNetProgramFromDir(dir)
	case clioptions.LangRuby:
		return sdkbuild.RubyProgramFromDir(dir, filepath.Join(repoRoot, "workers", "ruby"))
	default:
		return nil, fmt.Errorf("prebuilt projects not supported for language: %s", lang)
	}
}
