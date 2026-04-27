package devserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	temporalRepo = "https://github.com/temporalio/temporal.git"
	cloneRetries = 3
)

// buildServer builds ./cmd/server into binaryPath. When clone is true, the
// source is cloned from temporalio/temporal at ref into srcDir; a .sha marker
// next to the binary records which sha is cached, so the next call with the
// same ref skips clone+build. When clone is false, srcDir is used as-is and
// always rebuilt (caller may have uncommitted changes).
func buildServer(ctx context.Context, logger *zap.SugaredLogger, srcDir, ref, binaryPath string, clone bool) error {
	var sha, shaPath string
	if clone {
		var err error
		sha, err = resolveSha(ctx, ref)
		if err != nil {
			return err
		}
		shaPath = filepath.Join(filepath.Dir(binaryPath), ".sha")
		if cached, _ := os.ReadFile(shaPath); strings.TrimSpace(string(cached)) == sha {
			if _, err := os.Stat(binaryPath); err == nil {
				logger.Infof("Reusing cached server binary at %s (sha %s)", binaryPath, sha)
				return nil
			}
		}
		// Stale or missing — clear marker before re-cloning so a failed build
		// doesn't leave a misleading .sha behind.
		_ = os.Remove(shaPath)
		if err := cloneAt(ctx, logger, srcDir, ref); err != nil {
			return err
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
	if shaPath != "" {
		if err := os.WriteFile(shaPath, []byte(sha), 0644); err != nil {
			return fmt.Errorf("write sha marker: %w", err)
		}
	}
	return nil
}

// resolveSha returns the full sha for ref via `git ls-remote` against the
// temporal repo.
func resolveSha(ctx context.Context, ref string) (string, error) {
	out, err := runCmd(ctx, "", "git", "ls-remote", "--exit-code", temporalRepo, ref)
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s: %w\n%s", ref, err, out)
	}
	sha, _, ok := strings.Cut(strings.TrimSpace(string(out)), "\t")
	if !ok || sha == "" {
		return "", fmt.Errorf("unexpected ls-remote output: %q", string(out))
	}
	return sha, nil
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
