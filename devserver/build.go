package devserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"go.uber.org/zap"
)

const (
	temporalRepo = "https://github.com/temporalio/temporal.git"
	cloneRetries = 3
)

// buildMutexes serializes concurrent in-process callers of buildServer per
// srcDir. buildServer also takes a filesystem lock so package test processes
// sharing the same cache don't race on clone/build.
var buildMutexes sync.Map // map[srcDir]*sync.Mutex

// buildServer ensures temporal's ./cmd/server is built inside srcDir and
// returns the path to the binary. When clone is true, the source is cloned
// from temporalio/temporal at gitRef into srcDir; a .sha marker next to the
// binary records which sha is cached, so the next call with the same gitRef
// skips clone+build. When clone is false, srcDir is used as-is and always
// rebuilt (caller may have uncommitted changes).
func buildServer(
	ctx context.Context,
	logger *zap.SugaredLogger,
	srcDir, gitRef string,
	clone bool,
) (string, error) {
	srcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return "", fmt.Errorf("resolve source dir: %w", err)
	}

	muAny, _ := buildMutexes.LoadOrStore(srcDir, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(srcDir), 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	unlock, err := lockBuildCache(buildCacheLockPath(srcDir))
	if err != nil {
		return "", err
	}
	defer unlock()

	binaryPath := filepath.Join(srcDir, "temporal-server")
	shaFile := filepath.Join(srcDir, ".sha")

	var sha string
	if clone {
		var err error
		sha, err = resolveSha(ctx, gitRef)
		if err != nil {
			return "", err
		}
		if cached, _ := os.ReadFile(shaFile); strings.TrimSpace(string(cached)) == sha {
			if _, err := os.Stat(binaryPath); err == nil {
				logger.Infof("Reusing cached server binary at %s (sha %s)", binaryPath, sha)
				return binaryPath, nil
			}
		}
		// Stale or missing — cloneAt wipes srcDir, taking any stale .sha with it.
		if err := cloneAt(ctx, logger, srcDir, gitRef); err != nil {
			return "", err
		}
	}

	logger.Infof("Building temporal server in %s", srcDir)
	// Invoke temporal's own Makefile so build flags (build tags, CGO_ENABLED)
	// stay in sync with how temporal builds the server itself.
	cmd := exec.CommandContext(ctx, "make", "temporal-server")
	cmd.Dir = srcDir
	// Temporal's go.mod may pin a Go toolchain newer than the one running
	// omes' tests; let Go fetch it on demand instead of refusing with
	// GOTOOLCHAIN=local.
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("make temporal-server: %w\n%s", err, out)
	}
	if clone {
		if err := os.WriteFile(shaFile, []byte(sha), 0644); err != nil {
			return "", fmt.Errorf("write sha marker: %w", err)
		}
	}
	return binaryPath, nil
}

func defaultOutputDir(outputDir string) (string, error) {
	if outputDir != "" {
		return outputDir, nil
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".devserver"), nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

func lockBuildCache(path string) (func(), error) {
	lock := flock.New(path)
	if err := lock.Lock(); err != nil {
		return nil, fmt.Errorf("lock build cache: %w", err)
	}
	return func() {
		_ = lock.Unlock()
	}, nil
}

func buildCacheLockPath(srcDir string) string {
	return filepath.Join(filepath.Dir(srcDir), filepath.Base(srcDir)+".lock")
}

// resolveSha returns the full sha for gitRef. Accepts a 40-hex sha as-is;
// `git ls-remote` only resolves branch/tag refs, so raw shas would 404 there.
func resolveSha(ctx context.Context, gitRef string) (string, error) {
	if looksLikeSha(gitRef) {
		return gitRef, nil
	}
	out, err := runCmd(ctx, "", "git", "ls-remote", "--exit-code", temporalRepo, gitRef)
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s: %w\n%s", gitRef, err, out)
	}
	sha, _, ok := strings.Cut(strings.TrimSpace(string(out)), "\t")
	if !ok || !looksLikeSha(sha) {
		return "", fmt.Errorf("unexpected ls-remote output: %q", string(out))
	}
	return sha, nil
}

func looksLikeSha(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func cloneAt(ctx context.Context, logger *zap.SugaredLogger, dest, gitRef string) error {
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

	logger.Infof("Checking out %s", gitRef)
	if out, err := runCmd(ctx, dest, "git", "checkout", gitRef); err != nil {
		return fmt.Errorf("git checkout %s: %w\n%s", gitRef, err, out)
	}
	return nil
}

func runCmd(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
