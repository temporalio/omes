package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

// exitOnError ends the process on err, choosing how to report it by what kind of
// error it is.
//
// Bad input gets the message alone, on the reasoning in [loadgen.UsageError].
// errors.As unwraps, so a usage error still reads as one after a caller has
// wrapped it with run context; the wrapping is dropped from the output, since
// prefixes like "scenario failed" describe a run that never started. Anything
// else is a genuine failure and keeps the fatal-level treatment.
func exitOnError(logger *zap.SugaredLogger, err error) {
	if err == nil {
		return
	}
	var usage loadgen.UsageError
	if errors.As(err, &usage) {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	logger.Fatal(err)
}

func getRepoDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get source file location")
	}
	cliDir := filepath.Dir(filename) // cli
	cmdDir := filepath.Dir(cliDir)   // cmd
	repoDir := filepath.Dir(cmdDir)  // project root
	return repoDir, nil
}

// generateExecutionID generates a random execution ID to uniquely identify this particular
// execution of a scenario. This ensures no two executions with the same RunID collide.
func generateExecutionID() (string, error) {
	bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
