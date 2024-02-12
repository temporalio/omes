// Package sdkbuild provides helpers to build and run projects with SDKs across
// languages and versions.
package sdkbuild

import (
	"context"
	"os/exec"
)

// Program is a built SDK program that can be run.
type Program interface {
	// Dir is the directory the program is in. If created on the fly, usually this
	// temporary directory is deleted after use.
	Dir() string

	// NewCommand creates a new command for the program with given args and with
	// stdio set as the current stdio.
	NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error)
}
