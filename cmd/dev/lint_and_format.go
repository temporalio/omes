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
		Use:   "lint-and-format [target...]",
		Short: "Lint-and-format target(s)",
		Long: fmt.Sprintf(`Lint and format target(s)

Supported targets: %s

Examples:
  dev lint-and-format                    # All targets (default)
  dev lint-and-format all                # All targets
  dev lint-and-format go                 # Single target
  dev lint-and-format go java python     # Multiple targets`, strings.Join(supportedTargets, ", ")),
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
			return runLintAndFormat(cmd.Context(), targets)
		},
	}
}

func runLintAndFormat(ctx context.Context, targets []string) error {
	fmt.Println("Linting and formatting", strings.Join(targets, ", "), "target(s)...")

	for _, target := range targets {
		if err := lintAndFormat(ctx, target); err != nil {
			return fmt.Errorf("failed to lint-and-format %s: %v", target, err)
		}
	}

	return nil
}

func lintAndFormat(ctx context.Context, target string) error {
	fmt.Println("\n===========================================")
	fmt.Printf("Linting and formatting %s\n", target)
	fmt.Println("===========================================")

	targetDir, err := getTargetDir(target)
	if err != nil {
		return err
	}

	switch target {
	case "go":
		return lintAndFormatGoWorker(ctx, targetDir)
	case "java":
		return lintAndFormatJavaWorker(ctx, targetDir)
	case "python":
		return lintAndFormatPythonWorker(ctx, targetDir)
	case "typescript":
		return lintAndFormatTypescriptWorker(ctx, targetDir)
	case "dotnet":
		return lintAndFormatDotnetWorker(ctx, targetDir)
	case "ruby":
		return lintAndFormatRubyWorker(ctx, targetDir)
	case "kitchensink-gen":
		return lintAndFormatRustKitchenSinkGen(ctx)
	default:
		return fmt.Errorf("unsupported target: %s", target)
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

func lintAndFormatRustKitchenSinkGen(ctx context.Context) error {
	if err := checkTool(ctx, "cargo"); err != nil {
		return err
	}

	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}
	kitchenSinkGenDir := getKitchenSinkGenDir(repoDir)

	fmt.Println("Formatting Rust kitchensink-gen...")
	if err := runCommandInDir(ctx, kitchenSinkGenDir, "cargo", "fmt"); err != nil {
		return err
	}

	fmt.Println("✅ Rust kitchensink-gen lint-and-format completed successfully!")
	return nil
}

func lintAndFormatRubyWorker(ctx context.Context, workerDir string) error {
	if err := checkTool(ctx, "ruby"); err != nil {
		return err
	}

	fmt.Println("Formatting Ruby worker...")
	if err := runCommandInDir(ctx, workerDir, "bundle", "exec", "rubocop", "-A"); err != nil {
		return err
	}

	fmt.Println("Linting Ruby worker...")
	if err := runCommandInDir(ctx, workerDir, "bundle", "exec", "rubocop"); err != nil {
		return err
	}

	fmt.Println("✅ Ruby lint-and-format completed successfully!")
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
