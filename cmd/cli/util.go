package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
)

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

// generateExecutionID generates a random hex string to uniquely identify an execution.
func generateExecutionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
