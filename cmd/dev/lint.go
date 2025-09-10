package main

import (
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
			return runLintWorkers(languages)
		},
	}
}

func runLintWorkers(languages []string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	fmt.Printf("Linting %s worker(s)...\n", strings.Join(languages, ", "))

	for _, lang := range languages {
		if err := lintWorker(lang, rootDir); err != nil {
			return fmt.Errorf("failed to lint %s: %v", lang, err)
		}
	}

	return nil
}

func lintWorker(language, rootDir string) error {
	fmt.Printf("\n===========================================\n")
	fmt.Printf("Linting %s worker\n", language)
	fmt.Printf("===========================================\n")

	workerDir := filepath.Join(rootDir, "workers", language)
	if _, err := os.Stat(workerDir); os.IsNotExist(err) {
		return fmt.Errorf("worker directory not found: %s", workerDir)
	}

	switch language {
	case "go":
		return lintGoWorker(workerDir)
	case "java":
		return lintJavaWorker(workerDir)
	case "python":
		return lintPythonWorker(workerDir)
	case "typescript":
		return lintTypescriptWorker(workerDir)
	case "dotnet":
		return lintDotnetWorker(workerDir)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

func lintGoWorker(workerDir string) error {
	if err := validateLanguageTools("go"); err != nil {
		return err
	}

	fmt.Println("Checking Go modules...")
	if err := runCommandInDir(workerDir, "go", "mod", "tidy"); err != nil {
		return err
	}

	fmt.Println("Applying Go format...")
	if err := runCommandInDir(workerDir, "go", "fmt", "./..."); err != nil {
		return err
	}

	fmt.Println("Running Go tests...")
	if err := runCommandInDir(workerDir, "go", "test", "-race", "./..."); err != nil {
		return err
	}

	fmt.Println("Building Go worker...")
	if err := runCommandInDir(workerDir, "go", "build", "./..."); err != nil {
		return err
	}

	fmt.Println("✅ Go linting completed successfully!")
	return nil
}

func lintJavaWorker(workerDir string) error {
	if err := validateLanguageTools("java"); err != nil {
		return err
	}

	fmt.Println("Running Java linting and tests...")
	if err := runCommandInDir(workerDir, "./gradlew", "check"); err != nil {
		return err
	}

	fmt.Println("✅ Java linting completed successfully!")
	return nil
}

func lintPythonWorker(workerDir string) error {
	if err := validateLanguageTools("python"); err != nil {
		return err
	}

	fmt.Printf("uv version: ")
	if err := runCommand("uv", "--version"); err != nil {
		return err
	}

	fmt.Println("Checking Python dependencies...")
	if err := runCommandInDir(workerDir, "uv", "sync"); err != nil {
		return err
	}

	fmt.Println("Applying Python format...")
	if err := runCommandInDir(workerDir, "poe", "format"); err != nil {
		return err
	}

	fmt.Println("Running Python linting...")
	if err := runCommandInDir(workerDir, "poe", "lint"); err != nil {
		return err
	}

	fmt.Println("✅ Python linting completed successfully!")
	return nil
}

func lintTypescriptWorker(workerDir string) error {
	if err := validateLanguageTools("typescript"); err != nil {
		return err
	}

	fmt.Println("Installing TypeScript dependencies...")
	if err := runCommandInDir(workerDir, "npm", "ci"); err != nil {
		return err
	}

	fmt.Println("Running TypeScript linting...")
	if err := runCommandInDir(workerDir, "npm", "run", "lint"); err != nil {
		return err
	}

	fmt.Println("✅ TypeScript linting completed successfully!")
	return nil
}

func lintDotnetWorker(workerDir string) error {
	if err := validateLanguageTools("dotnet"); err != nil {
		return err
	}

	fmt.Println("Running .NET formatting check...")
	if err := runCommandInDir(workerDir, "dotnet", "format", "--verify-no-changes"); err != nil {
		return err
	}

	fmt.Println("Running .NET tests...")
	if err := runCommandInDir(workerDir, "dotnet", "test"); err != nil {
		return err
	}

	fmt.Println("✅ .NET linting completed successfully!")
	return nil
}
