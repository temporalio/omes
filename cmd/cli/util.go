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

// generateExecutionID generates a random execution ID to uniquely identify this particular
// execution of a scenario. This ensures no two executions with the same RunID collide.
func generateExecutionID() (string, error) {
	bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
