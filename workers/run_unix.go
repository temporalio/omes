//go:build !windows

package workers

import (
	"os"
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func osSendInterrupt(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}

func osSendKill(process *os.Process) error {
	// Shut down the process group (including all child processes)
	return syscall.Kill(-process.Pid, syscall.SIGKILL)
}
