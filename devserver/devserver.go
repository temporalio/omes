// Package devserver clones temporalio/temporal at a given git ref, builds
// ./cmd/server, and runs the resulting binary as a subprocess. Surface
// (Stop, FrontendHostPort) mirrors testsuite.DevServer so it can be used as
// a drop-in replacement for tests that need a specific server version.
package devserver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
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
	// OutputDir holds the source clone and built binary across runs. Defaults
	// to ".devserver" at the repository root. A single binary is cached per
	// OutputDir; the .sha marker file records which ref it was built from, so
	// switching Ref triggers a re-clone + rebuild.
	OutputDir string
	// Output receives the server process's combined stdout+stderr. When nil,
	// output is discarded.
	Output io.Writer
	// Logger receives build and lifecycle messages. Defaults to zap.NewNop().
	Logger *zap.SugaredLogger
	// Namespace is registered after the server is up. Defaults to "default".
	Namespace string
	// Persistence selects the persistence backend. Defaults to sqlite.
	Persistence PersistenceOptions
	// DynamicConfigValues are merged into the generated dynamic-config file
	// on top of the test-friendly defaults. Use for opt-in feature gates that
	// the test needs.
	DynamicConfigValues map[string]any
	// ClusterEndpoint overrides the "active" cluster info. Defaults to this
	// server's own frontend. Set this to another server's frontend to join a
	// multi-process cluster (e.g. for mixed-version testing).
	ClusterEndpoint ClusterEndpoint
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
	// Driver defaults to "sqlite".
	Driver string
}

// Ports are the service ports allocated for a running dev server.
type Ports struct {
	Host string

	FrontendGRPC       int
	FrontendMembership int
	FrontendHTTP       int
	HistoryGRPC        int
	HistoryMembership  int
	MatchingGRPC       int
	MatchingMembership int
	WorkerGRPC         int
	WorkerMembership   int
}

func newPorts(host string, ports [portCount]int) Ports {
	return Ports{
		Host:               host,
		FrontendGRPC:       ports[portFrontendGRPC],
		FrontendMembership: ports[portFrontendMembership],
		FrontendHTTP:       ports[portFrontendHTTP],
		HistoryGRPC:        ports[portHistoryGRPC],
		HistoryMembership:  ports[portHistoryMembership],
		MatchingGRPC:       ports[portMatchingGRPC],
		MatchingMembership: ports[portMatchingMembership],
		WorkerGRPC:         ports[portWorkerGRPC],
		WorkerMembership:   ports[portWorkerMembership],
	}
}

// Server is a running Temporal dev server subprocess.
type Server struct {
	frontend string
	ports    Ports
	logger   *zap.SugaredLogger
	workDir  string
	launch   func(context.Context) (context.CancelFunc, chan error, error)
	ready    func(context.Context) error

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped bool
	stopErr error
}

// Start clones+builds the requested ref and runs the server, returning once
// the frontend is reachable. The caller must invoke Stop to release resources.
func Start(ctx context.Context, opts Options) (*Server, error) {
	// Validate + apply defaults.
	if opts.Ref == "" && opts.SourceDir == "" {
		return nil, errors.New("devserver: Ref or SourceDir is required")
	}
	if opts.Ref != "" && opts.SourceDir != "" {
		return nil, errors.New("devserver: Ref and SourceDir are mutually exclusive")
	}
	opts.Logger = cmp.Or(opts.Logger, zap.NewNop().Sugar())
	opts.Namespace = cmp.Or(opts.Namespace, "default")

	// Per-run state goes in a fresh tempdir; cleaned up on Stop or failure.
	workDir, err := os.MkdirTemp("", "omes-devserver-")
	if err != nil {
		return nil, fmt.Errorf("devserver: create work dir: %w", err)
	}

	// Roll back partially-acquired resources on any error before the Server
	// struct takes ownership. After `success = true`, Stop() handles teardown.
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(workDir)
		}
	}()

	// Allocate service ports and create configs before the clone+build
	// so misconfiguration surfaces immediately.
	const host = "127.0.0.1"
	ports, err := allocatePorts(host)
	if err != nil {
		return nil, fmt.Errorf("devserver: allocate ports: %w", err)
	}
	serverPorts := newPorts(host, ports)
	frontendAddr := net.JoinHostPort(serverPorts.Host, strconv.Itoa(serverPorts.FrontendGRPC))
	frontendHTTPAddr := net.JoinHostPort(serverPorts.Host, strconv.Itoa(serverPorts.FrontendHTTP))
	dynConfigPath, err := writeDynamicConfig(workDir, frontendHTTPAddr, opts.DynamicConfigValues)
	if err != nil {
		return nil, fmt.Errorf("devserver: write dynamic config: %w", err)
	}
	serverEnv, err := buildServerEnv(opts.Persistence, opts.ClusterEndpoint, dynConfigPath, host, ports)
	if err != nil {
		return nil, fmt.Errorf("devserver: build env: %w", err)
	}

	// Resolve persistent paths (cache + source) and build the binary.
	outputDir, err := defaultOutputDir(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("devserver: resolve output dir: %w", err)
	}
	srcDir := cmp.Or(opts.SourceDir, outputDir)
	binaryPath, err := buildServer(ctx, opts.Logger, srcDir, opts.Ref, opts.SourceDir == "")
	if err != nil {
		return nil, fmt.Errorf("devserver: build: %w", err)
	}

	output := cmp.Or(opts.Output, io.Discard)
	s := &Server{
		frontend: frontendAddr,
		ports:    serverPorts,
		logger:   opts.Logger,
		workDir:  workDir,
	}
	s.launch = func(launchCtx context.Context) (context.CancelFunc, chan error, error) {
		runCtx, runCancel := context.WithCancel(launchCtx)
		cmd := exec.CommandContext(runCtx, binaryPath,
			"--allow-no-auth",
			"start",
		)
		cmd.Env = serverEnv
		cmd.Stdout = output
		cmd.Stderr = output
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = 15 * time.Second

		opts.Logger.Infof("Starting temporal server (ref %s, frontend %s)", opts.Ref, frontendAddr)
		if err := cmd.Start(); err != nil {
			runCancel()
			return nil, nil, err
		}

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		return runCancel, done, nil
	}
	s.ready = func(readyCtx context.Context) error {
		return s.registerNamespace(readyCtx, opts.Namespace)
	}
	if err := s.startProcessLocked(ctx); err != nil {
		return nil, err
	}
	success = true
	return s, nil
}

// FrontendHostPort returns the host:port for the running server's frontend
// gRPC service.
func (s *Server) FrontendHostPort() string {
	return s.frontend
}

// Ports returns the service ports allocated for the running server.
func (s *Server) Ports() Ports {
	return s.ports
}

// Restart gracefully terminates the server process and starts it again with
// the same binary, configuration, ports, output, and work directory.
func (s *Server) Restart(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return errors.New("devserver: cannot restart a stopped server")
	}
	if err := s.stopProcessLocked(); err != nil {
		return fmt.Errorf("devserver: stop for restart: %w", err)
	}
	if err := s.startProcessLocked(ctx); err != nil {
		return fmt.Errorf("devserver: restart: %w", err)
	}
	return nil
}

// Stop signals the server to terminate and waits for it to exit. The per-run
// work directory is removed. Stop is final and safe to call more than once;
// subsequent calls return the same error as the first.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return s.stopErr
	}
	s.stopped = true
	s.stopErr = s.stopProcessLocked()
	_ = os.RemoveAll(s.workDir)
	return s.stopErr
}

func (s *Server) startProcessLocked(ctx context.Context) error {
	cancel, done, err := s.launch(ctx)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}
	s.cancel = cancel
	s.done = done
	if err := s.ready(ctx); err != nil {
		_ = s.stopProcessLocked()
		return fmt.Errorf("wait for frontend: %w", err)
	}
	return nil
}

func (s *Server) stopProcessLocked() error {
	if s.cancel == nil {
		return nil
	}
	s.cancel()
	err := <-s.done
	s.cancel = nil
	s.done = nil
	return classifyStopErr(err)
}

func classifyStopErr(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return classifyExitErr(err)
}

// classifyExitErr treats a clean exit or termination by our own SIGTERM
// (sent by cmd.Cancel) as success, and surfaces anything else — including
// non-zero exit codes that didn't come from a signal, and SIGKILL that
// fires when the server fails to exit within WaitDelay.
func classifyExitErr(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() && ws.Signal() == syscall.SIGTERM {
		return nil
	}
	return err
}

func (s *Server) registerNamespace(ctx context.Context, namespace string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	done := s.done

	go func() {
		select {
		case procErr := <-done:
			done <- procErr
			cancel(fmt.Errorf("process exited before namespace registration completed: %w", procErr))
		case <-ctx.Done():
		}
	}()

	return registerNamespace(ctx, s.frontend, namespace)
}
