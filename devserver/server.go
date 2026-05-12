// Package devserver runs a Temporal dev server.
//
// Two modes:
//   - When Options.Ref is empty, delegates to
//     go.temporal.io/sdk/testsuite.StartDevServer (Temporal CLI dev-server).
//   - When Options.Ref is set, clones temporalio/temporal at that ref, builds
//     ./cmd/server, and runs the resulting binary as a subprocess.
//
// Server's surface (Stop, FrontendHostPort) mirrors testsuite.DevServer so
// it can be used as a drop-in replacement.
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
	"strings"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

// Options configures a Server.
type Options struct {
	// Ref is the temporalio/temporal git reference to build (sha, tag, branch).
	// When empty, the Temporal CLI dev-server (testsuite.StartDevServer) is
	// used instead of a source build.
	Ref string
	// BindAddress is the host:port the frontend gRPC service should bind to.
	// If empty, "127.0.0.1:<random-port>" is used. Only meaningful when Ref is
	// set; the testsuite path picks its own port.
	BindAddress string
	// PortBase, if non-zero, allocates 9 sequential ports starting here for
	// the temporal services (frontend grpc/membership/http, history,
	// matching, worker). Use it when ports must fit in a SMALLINT —
	// postgres stores cluster_membership.rpc_port that way, and Linux
	// ephemeral ports (32768+) overflow it. Mutually exclusive with an
	// explicit bind-address port.
	PortBase int
	// Namespace is registered after the server is up. Defaults to "default".
	Namespace string
	// Persistence selects the persistence backend. Defaults to sqlite (the
	// only backend omes itself uses). External callers (e.g. temporal/tests)
	// can request postgres etc. Ignored when Ref is empty (testsuite path is
	// always in-memory).
	Persistence PersistenceOptions
	// ClusterEndpoint overrides the "active" cluster info pointer in the
	// generated config. Defaults to this server's own frontend. Set this to
	// another server's frontend to join a multi-process cluster (e.g. for
	// mixed-version testing). Ignored when Ref is empty.
	ClusterEndpoint ClusterEndpoint
	// DynamicConfigValues are appended to the generated dynamic-config file
	// after the test-friendly defaults (cache disables, QPS bumps, version
	// check off). Use for opt-in feature gates that the test needs, e.g.
	// {"nexusoperation.enableStandalone": true}.
	DynamicConfigValues map[string]any
	// SourceDir, when set, overrides the cache-based source clone for the
	// Ref-based build path. Useful when the caller has a local source tree
	// to build from. Ignored when Ref is empty.
	SourceDir string
	// CacheRoot is the directory where source clones and built binaries are
	// cached across runs. Defaults to $XDG_CACHE_HOME/omes/server. Ignored
	// when Ref is empty.
	CacheRoot string
	// WorkDir is the directory for per-run state (config file, sqlite DB,
	// logs). Defaults to an os.MkdirTemp directory that is removed on Stop.
	// Ignored when Ref is empty.
	WorkDir string
	// Stdout, Stderr receive the server process's output. When unset, the
	// source-build path writes to <workDir>/server.log and the CLI path
	// discards. Setting either disables the source-build log file.
	Stdout io.Writer
	Stderr io.Writer
	// Logger receives build and lifecycle messages. Required.
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

// PersistenceOptions selects a server persistence backend for the
// source-build path.
type PersistenceOptions struct {
	// Driver is "sqlite" (default), or a postgres plugin name like
	// "postgres12" / "postgres12_pgx".
	Driver string
	// ConnectAddr, User, Password apply when Driver is not "sqlite".
	ConnectAddr string
	User        string
	Password    string
}

func (p PersistenceOptions) driverOrDefault() string {
	if p.Driver == "" {
		return "sqlite"
	}
	return p.Driver
}

// Server is a running Temporal dev server. It wraps either a subprocess
// started from a source build or a testsuite.DevServer, depending on how it
// was started.
type Server struct {
	frontend string
	logger   *zap.SugaredLogger

	// Source-build path:
	workDir     string
	ownsWorkDir bool
	logPath     string
	cancel      context.CancelFunc
	done        chan error

	// Testsuite-delegate path:
	cliServer *testsuite.DevServer
}

// Start runs a dev server per opts and returns it once the frontend is
// reachable. The caller must invoke Stop to release resources.
func Start(ctx context.Context, opts Options) (*Server, error) {
	if opts.Logger == nil {
		return nil, errors.New("devserver: Logger is required")
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	if opts.Ref == "" && opts.SourceDir == "" {
		return startFromCLI(ctx, opts)
	}
	return startFromSource(ctx, opts)
}

func startFromCLI(ctx context.Context, opts Options) (*Server, error) {
	opts.Logger.Info("Starting embedded dev server via Temporal CLI")
	srv, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			HostPort:  opts.BindAddress,
			Namespace: opts.Namespace,
		},
		LogLevel: "error",
		Stdout:   opts.Stdout,
		Stderr:   opts.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("devserver: start CLI dev-server: %w", err)
	}
	return &Server{
		frontend:  srv.FrontendHostPort(),
		logger:    opts.Logger,
		cliServer: srv,
	}, nil
}

func startFromSource(ctx context.Context, opts Options) (*Server, error) {
	cacheKey := opts.Ref
	if cacheKey == "" {
		cacheKey = "local"
	}
	cacheRoot, err := resolveCacheRoot(opts.CacheRoot, cacheKey)
	if err != nil {
		return nil, err
	}
	binaryPath := filepath.Join(cacheRoot, "server-bin")
	srcDir := opts.SourceDir
	if srcDir == "" {
		srcDir = filepath.Join(cacheRoot, "src")
	}

	if err := buildServer(ctx, opts.Logger, srcDir, opts.Ref, binaryPath, opts.SourceDir == ""); err != nil {
		return nil, fmt.Errorf("devserver: build: %w", err)
	}

	workDir, owns, err := resolveWorkDir(opts.WorkDir)
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		if owns {
			_ = os.RemoveAll(workDir)
		}
	}

	ports, err := allocatePortSet(opts.BindAddress, opts.PortBase)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("devserver: allocate ports: %w", err)
	}

	dynConfigPath, err := writeDynamicConfig(workDir, opts.DynamicConfigValues, ports)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("devserver: write dynamic config: %w", err)
	}

	var stdout, stderr io.Writer = opts.Stdout, opts.Stderr
	var logPath string
	var logFile *os.File
	if stdout == nil && stderr == nil {
		logPath = filepath.Join(workDir, "server.log")
		f, err := os.Create(logPath)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("devserver: create log file: %w", err)
		}
		logFile = f
		stdout = f
		stderr = f
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, binaryPath,
		"--allow-no-auth",
		"start",
	)
	cmd.Env = buildServerEnv(workDir, opts.Persistence, opts.ClusterEndpoint, dynConfigPath, ports)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second

	opts.Logger.Infof("Starting temporal server (ref %s, frontend %s, logs %s)",
		opts.Ref, ports.frontendAddr(), logPath)
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
		frontend:    ports.frontendAddr(),
		logger:      opts.Logger,
		workDir:     workDir,
		ownsWorkDir: owns,
		logPath:     logPath,
		cancel:      cancel,
		done:        done,
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
	if s.cliServer != nil {
		return s.cliServer.Stop()
	}
	s.cancel()
	err := <-s.done
	if s.ownsWorkDir {
		_ = os.RemoveAll(s.workDir)
	}
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

func resolveCacheRoot(override, ref string) (string, error) {
	base := override
	if base == "" {
		base = os.Getenv("OMES_DEVSERVER_CACHE")
	}
	if base == "" {
		if root := findRepoRoot(); root != "" {
			base = filepath.Join(root, ".devserver")
		} else {
			dir, err := os.UserCacheDir()
			if err != nil {
				return "", fmt.Errorf("resolve cache dir: %w", err)
			}
			base = filepath.Join(dir, "omes", "server")
		}
	}
	return filepath.Join(base, sanitizeRef(ref)), nil
}

// findRepoRoot walks up from the current working directory looking for a
// go.mod (the conventional repo root marker). Returns "" when not inside a
// Go module — e.g. when omes runs as an installed binary outside a checkout.
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func sanitizeRef(ref string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return r.Replace(ref)
}

func resolveWorkDir(override string) (string, bool, error) {
	if override != "" {
		if err := os.MkdirAll(override, 0755); err != nil {
			return "", false, fmt.Errorf("create work dir: %w", err)
		}
		return override, false, nil
	}
	dir, err := os.MkdirTemp("", "omes-devserver-")
	if err != nil {
		return "", false, fmt.Errorf("create work dir: %w", err)
	}
	return dir, true, nil
}
