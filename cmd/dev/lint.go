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

func lintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint [language...]",
		Short: "Lint and test worker code for specified language(s)",
		Long: fmt.Sprintf(`Lint and test worker code for specified language(s)

Supported languages: %s

Examples:
  dev lint                           # All languages (default)
  dev lint all                       # All languages
  dev lint go                        # Single language
  dev lint go java python            # Multiple languages`, strings.Join(supportedLanguages, ", ")),
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
			return runLintWorkers(cmd.Context(), languages)
		},
	}
}

func runLintWorkers(ctx context.Context, languages []string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	fmt.Println("Linting", strings.Join(languages, ", "), "worker(s)...")

	for _, lang := range languages {
		if err := lintWorker(ctx, lang, rootDir); err != nil {
			return fmt.Errorf("failed to lint %s: %v", lang, err)
		}
	}

	return nil
}

func lintWorker(ctx context.Context, language, rootDir string) error {
	fmt.Println("\n===========================================")
	fmt.Println("Linting", language, "worker")
	fmt.Println("===========================================")

	workerDir := filepath.Join(rootDir, "workers", language)
	if _, err := os.Stat(workerDir); os.IsNotExist(err) {
		return fmt.Errorf("worker directory not found: %s", workerDir)
	}

	switch language {
	case "go":
		return lintGoWorker(ctx, workerDir)
	case "java":
		return lintJavaWorker(ctx, workerDir)
	case "python":
		return lintPythonWorker(ctx, workerDir)
	case "typescript":
		return lintTypescriptWorker(ctx, workerDir)
	case "dotnet":
		return lintDotnetWorker(ctx, workerDir)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

func lintGoWorker(ctx context.Context, workerDir string) error {
	if err := validateLanguageTools(ctx, "go"); err != nil {
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

	fmt.Println("✅ Go linting completed successfully!")
	return nil
}

func lintJavaWorker(ctx context.Context, workerDir string) error {
	if err := validateLanguageTools(ctx, "java"); err != nil {
		return err
	}

	fmt.Println("Running Java linting and tests...")
	if err := runCommandInDir(ctx, workerDir, "./gradlew", "check"); err != nil {
		return err
	}

	fmt.Println("✅ Java linting completed successfully!")
	return nil
}

func lintPythonWorker(ctx context.Context, workerDir string) error {
	if err := validateLanguageTools(ctx, "python"); err != nil {
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

	fmt.Println("✅ Python linting completed successfully!")
	return nil
}

func lintTypescriptWorker(ctx context.Context, workerDir string) error {
	if err := validateLanguageTools(ctx, "typescript"); err != nil {
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

	fmt.Println("✅ TypeScript linting completed successfully!")
	return nil
}

func lintDotnetWorker(ctx context.Context, workerDir string) error {
	if err := validateLanguageTools(ctx, "dotnet"); err != nil {
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

	fmt.Println("✅ .NET linting completed successfully!")
	return nil
}
