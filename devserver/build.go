package devserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	temporalRepo = "https://github.com/temporalio/temporal.git"
	cloneRetries = 3
)

// buildServer builds ./cmd/server into binaryPath. When clone is true, the
// source is cloned from temporalio/temporal at ref into srcDir (caching the
// result across calls); when clone is false, srcDir is used as-is.
//
// Parallel callers targeting the same ref are serialized via a lockfile in
// the cache dir, so concurrent tests don't race on the clone/build.
func buildServer(ctx context.Context, logger *zap.SugaredLogger, srcDir, ref, binaryPath string, clone bool) error {
	unlock, err := acquireBuildLock(ctx, filepath.Dir(binaryPath))
	if err != nil {
		return err
	}
	defer unlock()

	if clone {
		if _, err := os.Stat(filepath.Join(srcDir, ".git")); err != nil {
			if err := cloneAt(ctx, logger, srcDir, ref); err != nil {
				return err
			}
		} else {
			logger.Infof("Reusing temporal source at %s (ref %s)", srcDir, ref)
		}

		if _, err := os.Stat(binaryPath); err == nil {
			logger.Infof("Reusing cached server binary at %s", binaryPath)
			return nil
		}
	}

	logger.Infof("Building temporal server -> %s", binaryPath)
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
		return fmt.Errorf("create binary dir: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "build",
		"-tags", "disable_grpc_modules",
		"-o", binaryPath,
		"./cmd/server",
	)
	cmd.Dir = srcDir
	// Temporal's go.mod may pin a Go toolchain newer than the one running
	// omes' tests; let Go fetch it on demand instead of refusing with
	// GOTOOLCHAIN=local.
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build server: %w\n%s", err, out)
	}
	return nil
}

// acquireBuildLock blocks on a per-cache-dir lockfile so concurrent processes
// (and concurrent goroutines within one process) take turns at clone+build.
// Returns a release function that must be called once the build is done.
func acquireBuildLock(ctx context.Context, cacheDir string) (func(), error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	lockPath := filepath.Join(cacheDir, ".build.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open build lock: %w", err)
	}
	// flock blocks until acquired. Watch ctx so a cancelled test doesn't
	// deadlock — try once, sleep a bit, repeat.
	for {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() {
				_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				_ = f.Close()
			}, nil
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func cloneAt(ctx context.Context, logger *zap.SugaredLogger, dest, ref string) error {
	// Wipe any partial state from a previous failed attempt.
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clean source dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create source parent: %w", err)
	}

	logger.Infof("Cloning %s into %s", temporalRepo, dest)
	var lastErr error
	for attempt := 1; attempt <= cloneRetries; attempt++ {
		out, err := runCmd(ctx, "", "git", "clone", "--filter=blob:none", temporalRepo, dest)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = fmt.Errorf("git clone (attempt %d): %w\n%s", attempt, err, out)
		_ = os.RemoveAll(dest)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if lastErr != nil {
		return lastErr
	}

	logger.Infof("Checking out %s", ref)
	if out, err := runCmd(ctx, dest, "git", "checkout", ref); err != nil {
		return fmt.Errorf("git checkout %s: %w\n%s", ref, err, out)
	}
	return nil
}

func runCmd(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
