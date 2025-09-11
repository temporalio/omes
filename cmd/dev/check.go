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

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [language...]",
		Short: "Check worker for specified language(s)",
		Long: fmt.Sprintf(`Check worker for specified language(s)

Supported languages: %s

Examples:
  dev check                           # All languages (default)
  dev check all                       # All languages
  dev check go                        # Single language
  dev check go java python            # Multiple languages`, strings.Join(supportedLanguages, ", ")),
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
			return runCheckWorkers(cmd.Context(), languages)
		},
	}
}

func runCheckWorkers(ctx context.Context, languages []string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	fmt.Println("Checking", strings.Join(languages, ", "), "worker(s)...")

	for _, lang := range languages {
		if err := checkWorker(ctx, lang, rootDir); err != nil {
			return fmt.Errorf("failed to check %s: %v", lang, err)
		}
	}

	return nil
}

func checkWorker(ctx context.Context, language, rootDir string) error {
	fmt.Println("\n===========================================")
	fmt.Println("Checking", language, "worker")
	fmt.Println("===========================================")

	workerDir := filepath.Join(rootDir, "workers", language)
	if _, err := os.Stat(workerDir); os.IsNotExist(err) {
		return fmt.Errorf("worker directory not found: %s", workerDir)
	}

	switch language {
	case "go":
		return checkGoWorker(ctx, workerDir)
	case "java":
		return checkJavaWorker(ctx, workerDir)
	case "python":
		return checkPythonWorker(ctx, workerDir)
	case "typescript":
		return checkTypescriptWorker(ctx, workerDir)
	case "dotnet":
		return checkDotnetWorker(ctx, workerDir)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

func checkGoWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "go"); err != nil {
		return err
	}

	fmt.Println("Checking Go modules...")
	if err := runCommandInDir(ctx, workerDir, "go", "mod", "tidy"); err != nil {
		return err
	}

	fmt.Println("Applying Go format...")
	if err := runCommandInDir(ctx, workerDir, "go", "fmt", "./..."); err != nil {
		return err
	}

	fmt.Println("Running Go tests...")
	if err := runCommandInDir(ctx, workerDir, "go", "test", "-race", "./..."); err != nil {
		return err
	}

	fmt.Println("Building Go worker...")
	if err := runCommandInDir(ctx, workerDir, "go", "build", "./..."); err != nil {
		return err
	}

	fmt.Println("✅ Go check completed successfully!")
	return nil
}

func checkJavaWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "java"); err != nil {
		return err
	}

	fmt.Println("Running Java linting and tests...")
	if err := runCommandInDir(ctx, workerDir, "./gradlew", "check"); err != nil {
		return err
	}

	fmt.Println("✅ Java check completed successfully!")
	return nil
}

func checkPythonWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "python"); err != nil {
		return err
	}

	fmt.Print("uv version: ")
	if err := runCommand(ctx, "uv", "--version"); err != nil {
		return err
	}

	fmt.Println("Checking Python dependencies...")
	if err := runCommandInDir(ctx, workerDir, "uv", "sync"); err != nil {
		return err
	}

	fmt.Println("Applying Python format...")
	if err := runCommandInDir(ctx, workerDir, "poe", "format"); err != nil {
		return err
	}

	fmt.Println("Running Python linting...")
	if err := runCommandInDir(ctx, workerDir, "poe", "lint"); err != nil {
		return err
	}

	fmt.Println("✅ Python check completed successfully!")
	return nil
}

func checkTypescriptWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "typescript"); err != nil {
		return err
	}

	fmt.Println("Installing TypeScript dependencies...")
	if err := runCommandInDir(ctx, workerDir, "npm", "ci"); err != nil {
		return err
	}

	fmt.Println("Running TypeScript linting...")
	if err := runCommandInDir(ctx, workerDir, "npm", "run", "lint"); err != nil {
		return err
	}

	fmt.Println("✅ TypeScript check completed successfully!")
	return nil
}

func checkDotnetWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "dotnet"); err != nil {
		return err
	}

	fmt.Println("Running .NET formatting check...")
	if err := runCommandInDir(ctx, workerDir, "dotnet", "format", "--verify-no-changes"); err != nil {
		return err
	}

	fmt.Println("Running .NET tests...")
	if err := runCommandInDir(ctx, workerDir, "dotnet", "test"); err != nil {
		return err
	}

	fmt.Println("✅ .NET check completed successfully!")
	return nil
}
