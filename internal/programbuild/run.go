package programbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/temporalio/features/sdkbuild"
)

const DefaultShutdownWaitDelay = 15 * time.Second

func StartProgramProcess(ctx context.Context, program sdkbuild.Program, args []string) (*exec.Cmd, error) {
	if program == nil {
		return nil, fmt.Errorf("Program is required")
	}

	cmd, err := program.NewCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create command from program: %w", err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = DefaultShutdownWaitDelay

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}
