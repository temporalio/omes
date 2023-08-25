// TODO: Can be de-duped with similar code in features repo
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const errFileCmdFmt = "failed to write to github file: %v"

// Set a GitHub environment value. Only works with values without a linebreak.
func writeGitHubEnv(name string, value string) (retErr error) {
	filepath := os.Getenv("GITHUB_ENV")
	if filepath == "" {
		// Just don't do anything if we're not running in a GH env
		return nil
	}
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		retErr = fmt.Errorf(errFileCmdFmt, err)
		return
	}

	defer func() {
		if err := f.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()

	msg := []byte(fmt.Sprintf("%s=%s\n", name, value))
	if _, err := f.Write(msg); err != nil {
		retErr = fmt.Errorf(errFileCmdFmt, err)
		return
	}
	return
}

func getCurrentCommitSha(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed getting git ref: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
