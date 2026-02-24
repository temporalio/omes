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
		Short: "Build and run a program",
		Long: `Build a test project and run the resulting program as a subprocess.

Signal forwarding and graceful shutdown are handled automatically.

Examples:
  omes exec --language python --project-dir ./my-test -- worker --task-queue q
  omes exec --language python --project-dir ./my-test -- client --port 8080
  omes exec --language python --project-dir . --version ../sdk-python -- worker --task-queue q
  omes exec --language python --project-dir ./my-test --process-monitor-addr :9091 -- worker --task-queue q`,
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			return r.run(ctx, args)
		},
	}

	r.sdkOpts.AddCLIFlags(cmd.Flags())
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

	runtimeArgs, err := programbuild.BuildRuntimeArgs(r.sdkOpts.Language, r.programOpts.ProgramDir)
	if err != nil {
		return err
	}
	runtimeArgs = append(runtimeArgs, args...)

	cmd, err := programbuild.StartProgramProcess(ctx, prog, runtimeArgs)
	if err != nil {
		return fmt.Errorf("failed to start program: %w", err)
	}

	r.logger.Infof("Started subprocess (PID %d): %v", cmd.Process.Pid, runtimeArgs)

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
