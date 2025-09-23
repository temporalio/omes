package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func lintAndFormatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint-and-format [language...]",
		Short: "Lint-and-format worker for specified language(s)",
		Long: fmt.Sprintf(`Lint and format worker for specified language(s)

Supported languages: %s

Examples:
  dev lint-and-format                    # All languages (default)
  dev lint-and-format all                # All languages
  dev lint-and-format go                 # Single language
  dev lint-and-format go java python     # Multiple languages`, strings.Join(supportedLanguages, ", ")),
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
			return runLintAndFormatWorkers(cmd.Context(), languages)
		},
	}
}

func runLintAndFormatWorkers(ctx context.Context, languages []string) error {
	fmt.Println("Linting and formatting", strings.Join(languages, ", "), "worker(s)...")

	for _, lang := range languages {
		if err := lintAndFormatWorker(ctx, lang); err != nil {
			return fmt.Errorf("failed to lint-and-format %s: %v", lang, err)
		}
	}

	return nil
}

func lintAndFormatWorker(ctx context.Context, language string) error {
	fmt.Println("\n===========================================")
	fmt.Println("Linting and formatting", language, "worker")
	fmt.Println("===========================================")

	workerDir, err := getWorkerDir(language)
	if err != nil {
		return err
	}

	switch language {
	case "go":
		return lintAndFormatGoWorker(ctx, workerDir)
	case "java":
		return lintAndFormatJavaWorker(ctx, workerDir)
	case "python":
		return lintAndFormatPythonWorker(ctx, workerDir)
	case "typescript":
		return lintAndFormatTypescriptWorker(ctx, workerDir)
	case "dotnet":
		return lintAndFormatDotnetWorker(ctx, workerDir)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

func lintAndFormatGoWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "go"); err != nil {
		return err
	}

	fmt.Println("Formatting Go worker...")
	if err := runCommandInDir(ctx, workerDir, "go", "fmt", "./..."); err != nil {
		return err
	}

	fmt.Println("Compilig Go worker...")
	if err := runCommandInDir(ctx, workerDir, "go", "build", "./..."); err != nil {
		return err
	}

	fmt.Println("✅ Go lint-and-format completed successfully!")
	return nil
}

func lintAndFormatJavaWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "java"); err != nil {
		return err
	}

	fmt.Println("Formatting Java worker...")
	if err := runCommandInDir(ctx, workerDir, "./gradlew", "spotlessApply"); err != nil {
		return err
	}

	if err := runCommandInDir(ctx, workerDir, "./gradlew", "check"); err != nil {
		return err
	}

	fmt.Println("✅ Java lint-and-format completed successfully!")
	return nil
}

func lintAndFormatPythonWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "python"); err != nil {
		return err
	}

	fmt.Println("Formatting Python worker...")
	if err := runCommandInDir(ctx, workerDir, "poe", "format"); err != nil {
		return err
	}

	fmt.Println("Linting Python worker...")
	if err := runCommandInDir(ctx, workerDir, "poe", "lint"); err != nil {
		return err
	}

	fmt.Println("✅ Python lint-and-format completed successfully!")
	return nil
}

func lintAndFormatTypescriptWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "typescript"); err != nil {
		return err
	}

	fmt.Println("Formatting TypeScript worker...")
	if err := runCommandInDir(ctx, workerDir, "npm", "run", "format"); err != nil {
		return err
	}

	fmt.Println("Linting TypeScript worker...")
	if err := runCommandInDir(ctx, workerDir, "npm", "run", "lint"); err != nil {
		return err
	}

	// Ensure the kitchensink is built first as the worker won't compile without it.
	if err := runBuildKitchensink(ctx); err != nil {
		return err
	}

	fmt.Println("Compiling TypeScript worker...")
	if err := runCommandInDir(ctx, workerDir, "npm", "run", "typecheck"); err != nil {
		return err
	}

	fmt.Println("✅ TypeScript lint-and-format completed successfully!")
	return nil
}

func lintAndFormatDotnetWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "dotnet"); err != nil {
		return err
	}

	fmt.Println("Formatting .NET worker...")
	if err := runCommandInDir(ctx, workerDir, "dotnet", "format"); err != nil {
		return err
	}

	fmt.Println("Compiling .NET worker...")
	if err := runCommandInDir(ctx, workerDir, "dotnet", "build", "--configuration", "Library"); err != nil {
		return err
	}

	fmt.Println("✅ .NET lint-and-format completed successfully!")
	return nil
}
