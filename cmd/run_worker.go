package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

func runWorkerCmd() *cobra.Command {
	var r workerRunner
	cmd := &cobra.Command{
		Use:   "run-worker",
		Short: "Run a worker",
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.run(ctx); err != nil {
				r.Logger.Fatal(err)
			}
		},
	}
	r.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("run-id")
	return cmd
}

type workerRunner struct {
	workers.Runner
	builder workerBuilder
}

func (r *workerRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.builder.addCLIFlags(fs)
	r.ScenarioID.AddCLIFlags(fs)
	fs.BoolVar(&r.RetainTempDir, "retain-temp-dir", false,
		"If set, retain the temp directory created if one wasn't given")
	fs.DurationVar(&r.GracefulShutdownDuration, "graceful-shutdown-duration", 30*time.Second,
		"Time to wait for worker to respond to interrupt before killing it")
	fs.BoolVar(&r.EmbeddedServer, "embedded-server", false, "Set to run in a local embedded server")
	fs.StringVar(&r.EmbeddedServerAddress, "embedded-server-address", "", "Address to bind local embedded server to")
	fs.IntVar(&r.TaskQueueIndexSuffixStart, "task-queue-suffix-index-start", 0, "Inclusive start for task queue suffix range")
	fs.IntVar(&r.TaskQueueIndexSuffixEnd, "task-queue-suffix-index-end", 0, "Inclusive end for task queue suffix range")
	r.ClientOptions.AddCLIFlags(fs)
	r.MetricsOptions.AddCLIFlags(fs, "worker-")
	r.WorkerOptions.AddCLIFlags(fs, "worker-")
}

func (r *workerRunner) preRun() {
	r.builder.preRun()
	r.Runner.Builder = r.builder.Builder
	r.TaskQueueName = loadgen.TaskQueueForRun(r.ScenarioID.RunID)
}

func (r *workerRunner) run(ctx context.Context) error {
	return r.Run(ctx, workers.BaseDir(repoDir(), r.SdkOptions.Language))
}

func withCancelOnInterrupt(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
