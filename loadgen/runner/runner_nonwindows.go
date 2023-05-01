//go:build !windows

package runner

import (
	"os"
	"syscall"
)

func sendInterrupt(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}
