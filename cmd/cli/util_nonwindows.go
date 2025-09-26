//go:build !windows

package cli

import (
	"os"
	"syscall"
)

func sendInterrupt(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}
