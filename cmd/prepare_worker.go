package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
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
			baseDir := workers.BaseDir(repoDir, b.SdkOptions.Language)
			if _, err := b.Build(cmd.Context(), baseDir); err != nil {
				b.Logger.Fatal(err)
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
	loggingOptions cmdoptions.LoggingOptions
}

func (b *workerBuilder) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&b.DirName, "dir-name", "", "Directory name for prepared worker")
	b.SdkOptions.AddCLIFlags(fs)
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *workerBuilder) preRun() {
	b.Logger = b.loggingOptions.MustCreateLogger()
}
