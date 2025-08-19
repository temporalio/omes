package workers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

type Runner struct {
	Builder
	RetainTempDir             bool
	GracefulShutdownDuration  time.Duration
	EmbeddedServer            bool
	EmbeddedServerAddress     string
	TaskQueueName             string
	TaskQueueIndexSuffixStart int
	TaskQueueIndexSuffixEnd   int
	ScenarioID                cmdoptions.ScenarioID
	ClientOptions             cmdoptions.ClientOptions
	MetricsOptions            cmdoptions.MetricsOptions
	WorkerOptions             cmdoptions.WorkerOptions
	LoggingOptions            cmdoptions.LoggingOptions
	OnWorkerStarted           func()
}

func (r *Runner) Run(ctx context.Context, baseDir string) error {
	if r.TaskQueueIndexSuffixStart > r.TaskQueueIndexSuffixEnd {
		return fmt.Errorf("cannot have task queue suffix start past end")
	}
	if r.TaskQueueName == "" {
		return fmt.Errorf("task queue name is required")
	}

	// Run an embedded server if requested
	if r.EmbeddedServer || r.EmbeddedServerAddress != "" {
		// Intentionally don't use context, will stop on defer
		if r.ClientOptions.EnableTLS || r.ClientOptions.ClientCertPath != "" || r.ClientOptions.ClientKeyPath != "" {
			return fmt.Errorf("cannot use TLS with embedded server")
		} else if r.ClientOptions.Address != client.DefaultHostPort {
			return fmt.Errorf("cannot supply non-default client address when using embedded server")
		}
		server, err := testsuite.StartDevServer(context.Background(), testsuite.DevServerOptions{
			ClientOptions: &client.Options{
				HostPort:  r.EmbeddedServerAddress,
				Namespace: r.ClientOptions.Namespace,
			},
			LogLevel: "error",
		})
		if err != nil {
			return fmt.Errorf("failed starting embedded server: %w", err)
		}
		r.ClientOptions.Address = server.FrontendHostPort()
		r.Logger.Infof("Started embedded local server at: %v", r.ClientOptions.Address)
		defer func() {
			r.Logger.Info("Stopping embedded local server")
			if err := server.Stop(); err != nil {
				r.Logger.Warnf("Failed stopping embedded local server: %v", err)
			}
		}()
	}

	// If there is not a prepared dir, we must build a temporary one and perform
	// the prep. Otherwise we reload the command from the directory.
	var prog sdkbuild.Program
	if r.DirName == "" {
		// Create temp dir
		tempDir, err := os.MkdirTemp(baseDir, "omes-temp-")
		if err != nil {
			return fmt.Errorf("failed creating temp dir: %w", err)
		}
		if !r.RetainTempDir {
			defer os.RemoveAll(tempDir)
		}
		r.DirName = filepath.Base(tempDir)

		// Build
		if prog, err = r.Build(ctx, baseDir); err != nil {
			return err
		}
	} else {
		var err error
		loadDir := filepath.Join(baseDir, r.DirName)
		switch r.SdkOptions.Language {
		case cmdoptions.LangGo:
			prog, err = sdkbuild.GoProgramFromDir(loadDir)
		case cmdoptions.LangPython:
			prog, err = sdkbuild.PythonProgramFromDir(loadDir)
		case cmdoptions.LangJava:
			prog, err = sdkbuild.JavaProgramFromDir(loadDir)
		case cmdoptions.LangTypeScript:
			prog, err = sdkbuild.TypeScriptProgramFromDir(loadDir)
		case cmdoptions.LangDotNet:
			prog, err = sdkbuild.DotNetProgramFromDir(loadDir)
		default:
			return fmt.Errorf("unrecognized language %v", r.SdkOptions.Language)
		}
		if err != nil {
			return fmt.Errorf("failed preparing: %w", err)
		}
	}

	// Build command args
	var args []string
	if r.SdkOptions.Language == cmdoptions.LangPython {
		// Python needs module name first
		args = append(args, "main")
	} else if r.SdkOptions.Language == cmdoptions.LangTypeScript {
		// Node also needs module
		args = append(args, "./tslib/omes.js")
	}
	args = append(args, "--task-queue", r.TaskQueueName)
	if r.TaskQueueIndexSuffixEnd > 0 {
		args = append(args, "--task-queue-suffix-index-start", strconv.Itoa(r.TaskQueueIndexSuffixStart))
		args = append(args, "--task-queue-suffix-index-end", strconv.Itoa(r.TaskQueueIndexSuffixEnd))
	}
	args = append(args, r.ClientOptions.ToFlags()...)
	args = append(args, r.MetricsOptions.ToFlags()...)
	args = append(args, r.LoggingOptions.ToFlags()...)
	args = append(args, r.WorkerOptions.ToFlags()...)

	cmd, err := prog.NewCommand(context.Background(), args...)
	if err != nil {
		return fmt.Errorf("failed creating command: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // set process group ID for shutdown

	// Start the command. Do not use the context so we can send interrupt.
	r.Logger.Infof("Starting worker with command: %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}
	if r.OnWorkerStarted != nil {
		r.OnWorkerStarted()
	}

	// Wait until context done or worker done
	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Wait() }()
	select {
	case err := <-errCh:
		if err == nil {
			err = fmt.Errorf("worker completed unexpectedly without error")
		}
		return fmt.Errorf("worker failed: %w", err)
	case <-ctx.Done():
		// Context cancelled, interrupt worker
		r.Logger.Infof("Sending interrupt to worker, PID: %v", cmd.Process.Pid)
		if err = sendInterrupt(cmd.Process); err != nil {
			return fmt.Errorf("failed to send interrupt to worker: %w", err)
		}

		select {
		case err = <-errCh:
			r.Logger.Infof("Worker exited after interrupt: %v", err)
		case <-time.After(r.GracefulShutdownDuration):
			if err = sendKill(cmd.Process); err != nil {
				return fmt.Errorf("failed to send kill to worker: %w", err)
			}
			if err = <-errCh; err == nil {
				err = fmt.Errorf("worker did not shutdown within graceful timeout")
			}
			if err != nil {
				r.Logger.Warnf("Worker exited after kill: %v", err)
			}
		}
		return nil
	}
}

func sendInterrupt(process *os.Process) error {
	if runtime.GOOS == "windows" {
		return process.Kill()
	}
	return process.Signal(syscall.SIGINT)
}

func sendKill(process *os.Process) error {
	// shutting down the process group (ie including all child processes)
	return syscall.Kill(-process.Pid, syscall.SIGKILL)
}
