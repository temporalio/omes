//go:build windows

package workers

import (
	"os"
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// Setpgid is not supported on Windows; no-op
}

func osSendInterrupt(process *os.Process) error {
	return process.Kill()
}

func osSendKill(process *os.Process) error {
	return process.Kill()
}
