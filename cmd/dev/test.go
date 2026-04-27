package main

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
	testScenario   = "workflow_with_single_noop_activity"
	testIterations = "5"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [language...]",
		Short: "Test worker for specified language(s) locally",
		Long: fmt.Sprintf(`Test worker for specified target(s) locally

Supported targets: %s

Examples:
  dev test                           # All targets (default)
  dev test all                       # All targets
  dev test go                        # Single target
  dev test go java python            # Multiple targets`, strings.Join(supportedTargets, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			var targets []string
			if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
				targets = supportedTargets
			} else {
				for _, target := range args {
					if !slices.Contains(supportedTargets, target) {
						return fmt.Errorf("unsupported target: %s", target)
					}
				}
				targets = args
			}

			for _, target := range targets {
				if err := runTestWorker(cmd.Context(), target); err != nil {
					return fmt.Errorf("failed to test %s: %v", target, err)
				}
			}
			return nil
		},
	}

	return cmd
}

func runTestWorker(ctx context.Context, language string) error {
	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}

	// kitchensink-gen cannot be tested as a worker.
	if language == "kitchensink-gen" {
		fmt.Println("Skipping test for generator target:", language)
		return nil
	}

	sdkVersion, err := getVersion(language + "-sdk")
	if err != nil {
		return err
	}

	fmt.Println("\n===========================================")
	fmt.Println("Testing", language, "worker")
	fmt.Println("===========================================")

	if err := checkTool(ctx, language); err != nil {
		return err
	}
	if language == "python" {
		if err := runPythonHarnessTests(ctx, repoDir); err != nil {
			return err
		}
	}
	if language == "dotnet" {
		if err := runDotnetHarnessTests(ctx, repoDir); err != nil {
			return err
		}
	}
	if language == "ruby" {
		if err := runRubyHarnessTests(ctx, repoDir); err != nil {
			return err
		}
	}
	return testWorkerLocally(ctx, repoDir, language, sdkVersion)
}

func runPythonHarnessTests(ctx context.Context, repoDir string) error {
	harnessDir := filepath.Join(repoDir, "workers", "python", "harness")
	fmt.Println("Running Python harness tests...")
	if err := runCommandInDir(ctx, harnessDir, "uv", "run", "poe", "test"); err != nil {
		return fmt.Errorf("failed Python harness tests: %w", err)
	}
	fmt.Println("✅ Python harness tests completed successfully!")
	return nil
}

func runDotnetHarnessTests(ctx context.Context, repoDir string) error {
	harnessTestsProj := filepath.Join(repoDir, "workers", "dotnet", "projects", "harness", "tests", "HarnessTests.csproj")
	fmt.Println("Running .NET harness tests...")
	if err := runCommandInDir(ctx, repoDir, "dotnet", "test", harnessTestsProj); err != nil {
		return fmt.Errorf("failed .NET harness tests: %w", err)
	}
	fmt.Println("✅ .NET harness tests completed successfully!")
	return nil
}

func runRubyHarnessTests(ctx context.Context, repoDir string) error {
	harnessDir := filepath.Join(repoDir, "workers", "ruby", "harness")
	rubyVersion, err := getVersion("ruby")
	if err != nil {
		return err
	}
	if err := checkMise(); err != nil {
		return err
	}
	fmt.Println("Running Ruby harness tests...")
	if err := runCommandInDir(
		ctx,
		harnessDir,
		"mise",
		"exec",
		"ruby@"+rubyVersion,
		"--",
		"bundle",
		"exec",
		"rake",
		"test",
	); err != nil {
		return fmt.Errorf("failed Ruby harness tests: %w", err)
	}
	fmt.Println("✅ Ruby harness tests completed successfully!")
	return nil
}

func testWorkerLocally(ctx context.Context, repoDir, language, sdkVersion string) error {
	args := []string{
		"go", "run", "./cmd", "run-scenario-with-worker",
		"--scenario", testScenario,
		"--log-level", "debug",
		"--language", language,
		"--embedded-server",
		"--iterations", testIterations,
		"--version", "v" + sdkVersion,
	}
	return runCommandInDir(ctx, repoDir, args[0], args[1:]...)
}
