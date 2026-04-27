// Package devserver clones temporalio/temporal at a given git ref, builds
// ./cmd/server, and runs the resulting binary as a subprocess. Surface
// (Stop, FrontendHostPort) mirrors testsuite.DevServer so it can be used as
// a drop-in replacement for tests that need a specific server version.
package devserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Options configures a Server. Either Ref or SourceDir must be set.
type Options struct {
	// Ref is the temporalio/temporal git reference to build (sha, tag, branch).
	Ref string
	// SourceDir, when set, overrides the cache-based source clone. Useful when
	// the caller has a local source tree to build from.
	SourceDir string
	// Namespace is registered after the server is up. Defaults to "default".
	Namespace string
	// Persistence selects the persistence backend. Defaults to sqlite.
	Persistence PersistenceOptions
	// ClusterEndpoint overrides the "active" cluster info pointer in the
	// generated config. Defaults to this server's own frontend. Set this to
	// another server's frontend to join a multi-process cluster (e.g. for
	// mixed-version testing).
	ClusterEndpoint ClusterEndpoint
	// DynamicConfigValues are merged into the generated dynamic-config file
	// on top of the test-friendly defaults (cache disables, QPS bumps,
	// version check off). Use for opt-in feature gates that the test needs,
	// e.g. {"nexusoperation.enableStandalone": true}.
	DynamicConfigValues map[string]any
	// OutputDir holds the source clone and built binary across runs. Defaults
	// to ".devserver" in the current directory. A single binary is cached per
	// OutputDir; the .sha marker file records which ref it was built from, so
	// switching Ref triggers a re-clone + rebuild.
	OutputDir string
	// Output receives the server process's combined stdout+stderr. When nil,
	// output is written to <workDir>/server.log inside the per-run temp dir.
	Output io.Writer
	// Logger receives build and lifecycle messages. Defaults to zap.NewNop().
	Logger *zap.SugaredLogger
}

// ClusterEndpoint identifies an active-cluster pointer for the server's
// generated config. Both fields are optional and default to "this server".
type ClusterEndpoint struct {
	// RPCAddress is the active cluster's frontend gRPC host:port.
	RPCAddress string
	// HTTPAddress is the active cluster's frontend HTTP host:port.
	HTTPAddress string
}

// PersistenceOptions selects a server persistence backend.
type PersistenceOptions struct {
	// Driver is "sqlite" (default), or a postgres plugin name like
	// "postgres12" / "postgres12_pgx".
	Driver string
	// ConnectAddr, User, Password apply when Driver is not "sqlite".
	ConnectAddr string
	User        string
	Password    string
}

// Server is a running Temporal dev server subprocess.
type Server struct {
	frontend string
	logger   *zap.SugaredLogger
	workDir  string
	logPath  string
	cancel   context.CancelFunc
	done     chan error
}

// Start clones+builds the requested ref and runs the server, returning once
// the frontend is reachable. The caller must invoke Stop to release resources.
func Start(ctx context.Context, opts Options) (*Server, error) {
	if opts.Ref == "" && opts.SourceDir == "" {
		return nil, errors.New("devserver: Ref or SourceDir is required")
	}
	if opts.Logger == nil {
		opts.Logger = zap.NewNop().Sugar()
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = ".devserver"
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("devserver: create output dir: %w", err)
	}
	binaryPath := filepath.Join(outputDir, "server-bin")
	srcDir := opts.SourceDir
	if srcDir == "" {
		srcDir = filepath.Join(outputDir, "src")
	}

	if err := buildServer(ctx, opts.Logger, srcDir, opts.Ref, binaryPath, opts.SourceDir == ""); err != nil {
		return nil, fmt.Errorf("devserver: build: %w", err)
	}

	workDir, err := os.MkdirTemp("", "omes-devserver-")
	if err != nil {
		return nil, fmt.Errorf("devserver: create work dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }

	const host = "127.0.0.1"
	ports, err := allocatePorts(host)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("devserver: allocate ports: %w", err)
	}
	frontendAddr := net.JoinHostPort(host, strconv.Itoa(ports[portFrontendGRPC]))
	frontendHTTPAddr := net.JoinHostPort(host, strconv.Itoa(ports[portFrontendHTTP]))

	dynConfigPath, err := writeDynamicConfig(workDir, frontendHTTPAddr, opts.DynamicConfigValues)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("devserver: write dynamic config: %w", err)
	}

	output := opts.Output
	var logPath string
	var logFile *os.File
	if output == nil {
		logPath = filepath.Join(workDir, "server.log")
		f, err := os.Create(logPath)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("devserver: create log file: %w", err)
		}
		logFile = f
		output = f
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, binaryPath,
		"--allow-no-auth",
		"start",
	)
	cmd.Env = buildServerEnv(workDir, opts.Persistence, opts.ClusterEndpoint, dynConfigPath, host, ports)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second

	opts.Logger.Infof("Starting temporal server (ref %s, frontend %s, logs %s)",
		opts.Ref, frontendAddr, logPath)
	if err := cmd.Start(); err != nil {
		cancel()
		if logFile != nil {
			_ = logFile.Close()
		}
		cleanup()
		return nil, fmt.Errorf("devserver: start: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		if logFile != nil {
			_ = logFile.Close()
		}
	}()

	s := &Server{
		frontend: frontendAddr,
		logger:   opts.Logger,
		workDir:  workDir,
		logPath:  logPath,
		cancel:   cancel,
		done:     done,
	}

	if err := s.waitForFrontend(ctx); err != nil {
		_ = s.Stop()
		return nil, err
	}
	if err := registerNamespace(ctx, s.frontend, opts.Namespace); err != nil {
		_ = s.Stop()
		return nil, fmt.Errorf("devserver: register namespace %q: %w", opts.Namespace, err)
	}
	return s, nil
}

// FrontendHostPort returns the host:port for the running server's frontend
// gRPC service.
func (s *Server) FrontendHostPort() string {
	return s.frontend
}

// Stop signals the server to terminate and waits for it to exit. The work
// directory is removed if it was allocated by Start.
func (s *Server) Stop() error {
	s.cancel()
	err := <-s.done
	_ = os.RemoveAll(s.workDir)
	// Exit due to our SIGTERM is expected; surface anything weirder.
	if err == nil {
		return nil
	}
	if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
		return nil
	}
	return err
}

func (s *Server) waitForFrontend(ctx context.Context) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		select {
		case err := <-s.done:
			s.done <- err
			return fmt.Errorf("devserver: process exited before frontend was reachable (logs: %s): %v", s.logPath, err)
		default:
		}
		conn, err := net.DialTimeout("tcp", s.frontend, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("devserver: frontend %s not reachable within timeout (logs: %s): %w", s.frontend, s.logPath, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

