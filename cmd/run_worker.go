package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func runWorkerCmd() *cobra.Command {
	var r workerRunner
	cmd := &cobra.Command{
		Use:   "run-worker",
		Short: "Run a worker",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.run(ctx); err != nil {
				r.logger.Fatal(err)
			}
		},
	}
	r.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("scenario")
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("run-id")
	return cmd
}

type workerRunner struct {
	workerBuilder
	scenario                  string
	runID                     string
	retainTempDir             bool
	gracefulShutdownDuration  time.Duration
	embeddedServer            bool
	embeddedServerAddress     string
	taskQueueIndexSuffixStart int
	taskQueueIndexSuffixEnd   int
	clientOptions             cmdoptions.ClientOptions
	metricsOptions            cmdoptions.MetricsOptions
	workerOptions             cmdoptions.WorkerOptions
	onWorkerStarted           func()
}

func (r *workerRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.workerBuilder.addCLIFlags(fs)
	fs.StringVar(&r.scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.runID, "run-id", "", "Run ID for this run")
	fs.BoolVar(&r.retainTempDir, "retain-temp-dir", false,
		"If set, retain the temp directory created if one wasn't given")
	fs.DurationVar(&r.gracefulShutdownDuration, "graceful-shutdown-duration", 30*time.Second,
		"Time to wait for worker to respond to interrupt before killing it")
	fs.BoolVar(&r.embeddedServer, "embedded-server", false, "Set to run in a local embedded server")
	fs.StringVar(&r.embeddedServerAddress, "embedded-server-address", "", "Address to bind local embedded server to")
	fs.IntVar(&r.taskQueueIndexSuffixStart, "task-queue-suffix-index-start", 0, "Inclusive start for task queue suffix range")
	fs.IntVar(&r.taskQueueIndexSuffixEnd, "task-queue-suffix-index-end", 0, "Inclusive end for task queue suffix range")
	r.clientOptions.AddCLIFlags(fs)
	r.metricsOptions.AddCLIFlags(fs, "worker-")
	r.workerOptions.AddCLIFlags(fs, "worker-")
}

func (r *workerRunner) run(ctx context.Context) error {
	r.logger = r.loggingOptions.MustCreateLogger()
	lang, err := normalizeLangName(r.language)
	if err != nil {
		return err
	}
	scenario := loadgen.GetScenario(r.scenario)
	if scenario == nil {
		return fmt.Errorf("scenario %v not found", r.scenario)
	}
	if r.taskQueueIndexSuffixStart > r.taskQueueIndexSuffixEnd {
		return fmt.Errorf("cannot have task queue suffix start past end")
	}
	if r.runID == "" {
		r.runID = shortRand()
	}
	baseDir := filepath.Join(rootDir(), "workers", lang)

	// Run an embedded server if requested
	if r.embeddedServer || r.embeddedServerAddress != "" {
		// Intentionally don't use context, will stop on defer
		if r.clientOptions.EnableTLS || r.clientOptions.ClientCertPath != "" || r.clientOptions.ClientKeyPath != "" {
			return fmt.Errorf("cannot use TLS with embedded server")
		} else if r.clientOptions.Address != client.DefaultHostPort {
			return fmt.Errorf("cannot supply non-default client address when using embedded server")
		}
		server, err := testsuite.StartDevServer(context.Background(), testsuite.DevServerOptions{
			ClientOptions: &client.Options{
				HostPort:  r.embeddedServerAddress,
				Namespace: r.clientOptions.Namespace,
			},
			LogLevel: "error",
		})
		if err != nil {
			return fmt.Errorf("failed starting embedded server: %w", err)
		}
		r.clientOptions.Address = server.FrontendHostPort()
		r.logger.Infof("Started embedded local server at: %v", r.clientOptions.Address)
		defer func() {
			r.logger.Info("Stopping embedded local server")
			if err := server.Stop(); err != nil {
				r.logger.Warnf("Failed stopping embedded local server: %v", err)
			}
		}()
	}

	// If there is not a prepared dir, we must build a temporary one and perform
	// the prep. Otherwise we reload the command from the directory.
	var prog sdkbuild.Program
	if r.dirName == "" {
		// Create temp dir
		tempDir, err := os.MkdirTemp(baseDir, "omes-temp-")
		if err != nil {
			return fmt.Errorf("failed creating temp dir: %w", err)
		}
		if !r.retainTempDir {
			defer os.RemoveAll(tempDir)
		}
		r.dirName = filepath.Base(tempDir)

		// Build
		if prog, err = r.build(ctx); err != nil {
			return err
		}
	} else {
		loadDir := filepath.Join(baseDir, r.dirName)
		switch lang {
		case "go":
			prog, err = sdkbuild.GoProgramFromDir(loadDir)
		case "python":
			prog, err = sdkbuild.PythonProgramFromDir(loadDir)
		case "java":
			prog, err = sdkbuild.JavaProgramFromDir(loadDir)
		case "typescript":
			prog, err = sdkbuild.TypeScriptProgramFromDir(loadDir)
		default:
			return fmt.Errorf("unrecognized language %v", lang)
		}
		if err != nil {
			return fmt.Errorf("failed preparing: %w", err)
		}
	}

	// Build command args
	var args []string
	if lang == "python" {
		// Python needs module name first
		args = append(args, "main")
	} else if lang == "typescript" {
		// Node also needs module
		args = append(args, "./tslib/omes.js")
	}
	args = append(args, "--task-queue", loadgen.TaskQueueForRun(r.scenario, r.runID))
	if r.taskQueueIndexSuffixEnd > 0 {
		args = append(args, "--task-queue-suffix-index-start", strconv.Itoa(r.taskQueueIndexSuffixStart))
		args = append(args, "--task-queue-suffix-index-end", strconv.Itoa(r.taskQueueIndexSuffixEnd))
	}
	args = append(args, r.clientOptions.ToFlags()...)
	args = append(args, r.metricsOptions.ToFlags()...)
	args = append(args, r.loggingOptions.ToFlags()...)
	args = append(args, r.workerOptions.ToFlags()...)

	// Start the command. Do not use the context so we can send interrupt.
	cmd, err := prog.NewCommand(context.Background(), args...)
	if err != nil {
		return fmt.Errorf("failed creating command: %w", err)
	}
	r.logger.Infof("Starting worker with command: %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}
	if r.onWorkerStarted != nil {
		r.onWorkerStarted()
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
		r.logger.Infof("Sending interrupt to worker, PID: %v", cmd.Process.Pid)
		if err := sendInterrupt(cmd.Process); err != nil {
			return fmt.Errorf("failed to send interrupt to worker: %w", err)
		}
		select {
		case err = <-errCh:
		case <-time.After(r.gracefulShutdownDuration):
			if err = cmd.Process.Kill(); err == nil {
				if err = <-errCh; err == nil {
					err = fmt.Errorf("worker did not shutdown within graceful timeout")
				}
			}
		}
		if err != nil {
			r.logger.Warnf("Worker failed after interrupt: %v", err)
		}
		return nil
	}
}

func shortRand() string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))
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
