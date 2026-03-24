package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var cleanCommands = map[string][]string{
	"go":         {"go", "clean"},
	"java":       {"./gradlew", "clean"},
	"python":     {"uv", "clean"},
	"typescript": {"npm", "run", "clean"},
	"dotnet":     {"dotnet", "clean"},
}

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [target...]",
		Short: "Clean build artifacts for specified target(s)",
		Long: fmt.Sprintf(`Clean build artifacts and temporary directories for one or more targets.

Supported targets: %s

Examples:
  dev clean                          # Clean all targets (default)
  dev clean all                      # Clean all targets
  dev clean go                       # Clean Go only
  dev clean go java python           # Clean multiple targets`, strings.Join(supportedTargets, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			var targets []string
			cleanAll := len(args) == 0 || (len(args) == 1 && args[0] == "all")

			if cleanAll {
				targets = supportedTargets
			} else {
				// Validate targets
				for _, target := range args {
					if !slices.Contains(supportedTargets, target) {
						return fmt.Errorf("unsupported target: %s", target)
					}
				}
				targets = args
			}

			return runClean(cmd.Context(), targets, cleanAll)
		},
	}

	return cmd
}

func cleanLanguage(ctx context.Context, target, repoDir string) error {
	workerDir := filepath.Join(repoDir, "workers", target)
	if _, err := os.Stat(workerDir); err != nil {
		return fmt.Errorf("worker directory does not exist: %s", workerDir)
	}

	fmt.Println("Cleaning", target)

	cleanCmd, ok := cleanCommands[target]
	if !ok {
		return fmt.Errorf("unsupported target: %s", target)
	}
	return runCommandInDir(ctx, workerDir, cleanCmd[0], cleanCmd[1:]...)
}

func runClean(ctx context.Context, targets []string, cleanAll bool) error {
	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}

	if cleanAll {
		fmt.Println("Cleaning all build artifacts and temporary directories...")

		// Remove omes-temp-* directories
		fmt.Println("Removing omes-temp-* directories...")
		workersDir := filepath.Join(repoDir, "workers")
		if err := removeTempDirs(workersDir); err != nil {
			fmt.Printf("Warning: failed to remove temp directories: %v\n", err)
		}
	} else {
		fmt.Println("Cleaning", strings.Join(targets, ", "))
	}

	for _, target := range targets {
		if err := cleanLanguage(ctx, target, repoDir); err != nil {
			if cleanAll {
				fmt.Printf("Warning: failed to clean %s: %v\n", target, err)
			} else {
				return fmt.Errorf("failed to clean %s: %v", target, err)
			}
		}
	}

	fmt.Println("✅ Clean completed")
	return nil
}

func removeTempDirs(workersDir string) error {
	return filepath.Walk(workersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if info.IsDir() && strings.HasPrefix(info.Name(), "omes-temp-") {
			fmt.Println("Removing", path)
			if err := os.RemoveAll(path); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", path, err)
			}
			return filepath.SkipDir
		}
		return nil
	})
}
