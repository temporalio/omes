//go:build !windows

package main

import (
	"os"
	"syscall"
)

func sendInterrupt(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}
