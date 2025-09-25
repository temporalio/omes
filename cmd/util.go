package main

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func getRepoDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get source file location")
	}
	cmdDir := filepath.Dir(filename) // cmd
	repoDir := filepath.Dir(cmdDir)  // project root
	return repoDir, nil
}
