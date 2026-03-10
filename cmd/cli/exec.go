package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/programbuild"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var r execRunner

	cmd := &cobra.Command{
		Use:   "exec --language <lang> -- <program-args>",
		Short: "Build a project and run it as a subprocess",
		Long: `Build a test project and run the resulting binary as a subprocess.
Arguments after "--" are passed directly to the subprocess.

Signal forwarding and graceful shutdown are handled automatically.

Examples:
  # Start a worker on a specific task queue
  omes exec --language go --project-dir ./projecttests/go/tests/helloworld -- worker --task-queue my-queue

  # Start a project gRPC server on port 8080
  omes exec --language go --project-dir ./projecttests/go/tests/helloworld -- project-server --port 8080

  # Start a worker with process metrics sidecar for CPU/memory monitoring
  omes exec --language go --project-dir ./projecttests/go/tests/helloworld --process-monitor-addr :9091 -- worker --task-queue my-queue`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			r.preRun()
			return r.validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			return r.run(ctx, args)
		},
	}

	r.sdkOpts.AddCLIFlags(cmd.Flags())
	cmd.Flags().Lookup("language").Usage = "Language to use for workflow tests (go only)"
	r.programOpts.AddFlags(cmd.Flags())
	cmd.Flags().AddFlagSet(r.loggingOpts.FlagSet())
	cmd.Flags().StringVar(&r.processMonitorAddr, "process-monitor-addr", "", "Address for process metrics sidecar (e.g. :9091)")

	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("project-dir")

	return cmd
}

type execRunner struct {
	sdkOpts            clioptions.SdkOptions
	programOpts        clioptions.ProgramOptions
	loggingOpts        clioptions.LoggingOptions
	processMonitorAddr string

	logger *zap.SugaredLogger
}

func (r *execRunner) preRun() {
	r.logger = r.loggingOpts.MustCreateLogger()
}

func (r *execRunner) validate() error {
	if r.sdkOpts.Language != clioptions.LangGo {
		return fmt.Errorf("--language must be go for workflow tests (got %q)", r.sdkOpts.Language)
	}
	if r.programOpts.ProgramDir == "" {
		return fmt.Errorf("--project-dir is required")
	}
	return nil
}

func (r *execRunner) run(ctx context.Context, args []string) error {
	builder := &programbuild.ProgramBuilder{
		Language:   r.sdkOpts.Language.String(),
		ProjectDir: r.programOpts.ProgramDir,
		BuildDir:   r.programOpts.BuildDir,
		Logger:     r.logger,
	}

	prog, err := builder.BuildProgram(ctx, r.sdkOpts.Version)
	if err != nil {
		return fmt.Errorf("failed to build program: %w", err)
	}

	cmd, err := programbuild.StartProgramProcess(ctx, prog, args)
	if err != nil {
		return fmt.Errorf("failed to start program: %w", err)
	}

	r.logger.Infof("Started subprocess (PID %d): %v", cmd.Process.Pid, args)

	// Start process metrics sidecar (if requested)
	if r.processMonitorAddr != "" {
		sidecar := clioptions.StartProcessMetricsSidecar(
			r.logger,
			r.processMonitorAddr,
			cmd.Process.Pid,
			r.sdkOpts.Version,
			"",
			r.sdkOpts.Language.String(),
		)
		defer func() {
			if err := sidecar.Shutdown(context.Background()); err != nil {
				r.logger.Warnf("Failed to stop process metrics sidecar: %v", err)
			}
		}()
	}

	err = cmd.Wait()
	if err != nil {
		if cmd.ProcessState != nil {
			if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				return fmt.Errorf("process exited with code %d", status.ExitStatus())
			}
			return fmt.Errorf("process exited with code 1")
		}
		return fmt.Errorf("failed waiting for process: %w", err)
	}

	return nil
}
