package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
	project "github.com/temporalio/omes/scenarios/project"
	"github.com/temporalio/omes/workers"
)

func prepareWorkerCmd() *cobra.Command {
	b := workerBuilder{}
	cmd := &cobra.Command{
		Use:   "prepare-worker",
		Short: "Build worker ready to run",
		PreRun: func(cmd *cobra.Command, args []string) {
			b.preRun()
		},
		Run: func(cmd *cobra.Command, args []string) {
			repoDir, err := getRepoDir()
			if err != nil {
				b.Logger.Fatal(fmt.Errorf("failed to get root directory: %w", err))
			}
			if b.ProjectDir != "" {
				absProjectDir, err := filepath.Abs(b.ProjectDir)
				if err != nil {
					b.Logger.Fatal(fmt.Errorf("failed to resolve project dir: %w", err))
				}
				baseDir := filepath.Dir(absProjectDir)
				if _, err := project.Build(cmd.Context(), project.BuildOptions{
					Language:   b.SdkOptions.Language,
					ProjectDir: absProjectDir,
					BaseDir:    baseDir,
					Version:    b.SdkOptions.Version,
					Logger:     b.Logger,
				}); err != nil {
					b.Logger.Fatal(err)
				}
			} else {
				baseDir := workers.BaseDir(repoDir, b.SdkOptions.Language)
				if _, err := b.Build(cmd.Context(), baseDir); err != nil {
					b.Logger.Fatal(err)
				}
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("dir-name")
	cmd.MarkFlagRequired("language")
	return cmd
}

type workerBuilder struct {
	workers.Builder
	loggingOptions clioptions.LoggingOptions
}

func (b *workerBuilder) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&b.DirName, "dir-name", "", "Directory name for prepared worker")
	fs.StringVar(&b.ProjectDir, "project-dir", "", "Path to project directory (builds a project test instead of standard worker)")
	b.SdkOptions.AddCLIFlags(fs)
	fs.AddFlagSet(b.loggingOptions.FlagSet())
}

func (b *workerBuilder) preRun() {
	b.Logger = b.loggingOptions.MustCreateLogger()
}
