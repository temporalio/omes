package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func parseOptionValues(values []string) (map[string]string, error) {
	options := make(map[string]string, len(values))
	for _, v := range values {
		pieces := strings.SplitN(v, "=", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("option does not have '='")
		}
		key, value := pieces[0], pieces[1]

		// If the value starts with '@', read the file and use its contents as the value.
		if strings.HasPrefix(value, "@") {
			filePath := strings.TrimPrefix(value, "@")
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			value = string(data)
		}
		options[key] = value
	}
	return options, nil
}
