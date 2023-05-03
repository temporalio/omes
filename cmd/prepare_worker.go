package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"go.temporal.io/features/sdkbuild"
	"go.uber.org/zap"
)

func prepareWorkerCmd() *cobra.Command {
	b := workerBuilder{createDir: true}
	cmd := &cobra.Command{
		Use:   "prepare-worker",
		Short: "Build worker ready to run",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := b.build(cmd.Context()); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("dir-name")
	cmd.MarkFlagRequired("language")
	return cmd
}

type workerBuilder struct {
	logger         *zap.SugaredLogger
	dirName        string
	language       string
	version        string
	loggingOptions cmdoptions.LoggingOptions
	createDir      bool
}

func (b *workerBuilder) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&b.dirName, "dir-name", "", "Directory name for prepared worker")
	fs.StringVar(&b.language, "language", "", "Language to prepare a worker for")
	fs.StringVar(&b.version, "version", "", "Version to prepare a worker for - treated as path if slash present")
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *workerBuilder) build(ctx context.Context) (sdkbuild.Program, error) {
	if b.dirName == "" {
		return nil, fmt.Errorf("directory name required")
	} else if strings.ContainsAny(b.dirName, `/\`) {
		return nil, fmt.Errorf("dir name is not a full path, it is a single name")
	}
	if b.logger == nil {
		b.logger = b.loggingOptions.MustCreateLogger()
	}
	lang, err := normalizeLangName(b.language)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(rootDir(), "workers", lang)
	b.logger.Infof("Building %v program at %v", lang, filepath.Join(baseDir, b.dirName))
	// Check or create dir
	if b.createDir {
		if err := os.Mkdir(filepath.Join(baseDir, b.dirName), 0755); err != nil {
			return nil, fmt.Errorf("failed creating directory: %w", err)
		}
	} else {
		if _, err := os.Stat(filepath.Join(baseDir, b.dirName)); err != nil {
			return nil, fmt.Errorf("failed finding directory: %w", err)
		}
	}
	switch lang {
	case "go":
		return b.buildGo(ctx, baseDir)
	case "python":
		return b.buildPython(ctx, baseDir)
	default:
		return nil, fmt.Errorf("unrecognized language %v", lang)
	}
}

func (b *workerBuilder) buildGo(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir: baseDir,
		DirName: b.dirName,
		Version: b.version,
		GoModContents: `module github.com/temporalio/omes-worker

go 1.20

require github.com/temporalio/omes v1.0.0
require github.com/temporalio/omes/workers/go v1.0.0

replace github.com/temporalio/omes => ../../../
replace github.com/temporalio/omes/workers/go => ../`,
		GoMainContents: `package main

import "github.com/temporalio/omes/workers/go/worker"

func main() {
	worker.Main()
}`,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *workerBuilder) buildPython(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// If version not provided, try to read it from pyproject.toml
	version := b.version
	if version == "" {
		b, err := os.ReadFile(filepath.Join(baseDir, "pyproject.toml"))
		if err != nil {
			return nil, fmt.Errorf("failed reading package.json: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "temporalio = ") {
				version = line[strings.Index(line, `"`)+1 : strings.LastIndex(line, `"`)]
				break
			}
		}
		if version == "" {
			return nil, fmt.Errorf("version not found in pyproject.toml")
		}
	}

	// Build
	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir:        baseDir,
		DirName:        b.dirName,
		Version:        version,
		DependencyName: "omes",
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func rootDir() string {
	_, currFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(currFile))
}

func normalizeLangName(lang string) (string, error) {
	switch lang {
	case "go", "python":
	case "py":
		lang = "python"
	default:
		return "", fmt.Errorf("invalid language %q, must be one of: go or python", lang)
	}
	return lang, nil
}
