package programbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
)

const DefaultShutdownWaitDelay = 15 * time.Second

// BuildRuntimeArgs returns the initial runtime args for a program based on language.
// For Python: returns module name (with dashes replaced by underscores)
// For TypeScript: returns path to compiled entry point
// For Go: returns empty slice (binary takes subcommand directly)
func BuildRuntimeArgs(language clioptions.Language, projectDir string, extraArgs ...string) ([]string, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory: %w", err)
	}
	projectName := filepath.Base(absProjectDir)

	var args []string
	switch language {
	case clioptions.LangPython:
		args = append(args, strings.ReplaceAll(projectName, "-", "_"))
	case clioptions.LangTypeScript:
		args = append(args, fmt.Sprintf("tslib/tests/%s/main.js", projectName))
	case clioptions.LangGo:
		// Nothing to do for Go
	default:
		return nil, fmt.Errorf("unsupported language for runtime args: %s", language)
	}
	args = append(args, extraArgs...)
	return args, nil
}

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
