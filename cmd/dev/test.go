package main

import (
	"context"
	"fmt"
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
		Long: fmt.Sprintf(`Test worker for specified language(s) locally

Supported languages: %s

Examples:
  dev test                           # All languages (default)
  dev test all                       # All languages
  dev test go                        # Single language
  dev test go java python            # Multiple languages`, strings.Join(supportedLanguages, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			var languages []string
			if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
				languages = supportedLanguages
			} else {
				for _, lang := range args {
					if !slices.Contains(supportedLanguages, lang) {
						return fmt.Errorf("unsupported language: %s", lang)
					}
				}
				languages = args
			}

			for _, language := range languages {
				if err := runTestWorker(cmd.Context(), language); err != nil {
					return fmt.Errorf("failed to test %s: %v", language, err)
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
	return testWorkerLocally(ctx, repoDir, language, sdkVersion)
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


