package main

import (
	"context"
	"fmt"
	"os/exec"
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
			return fmt.Errorf("failed to lint-and-format %s: %w", target, err)
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
	if err := checkMise(); err != nil {
		return err
	}

	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}

	// Lint and format both Go modules: the root omes module (cmd, loadgen,
	// scenarios, …) and the Go worker. golangci-lint is pinned in
	// .config/mise/config.toml; `golangci-lint fmt` formats (gofmt/gci/golines)
	// and `golangci-lint run` lints (and compiles, so a separate build is
	// redundant). We pass explicit package directories rather than ./... so the
	// walk skips vendored node_modules Go sources (e.g. under the TypeScript
	// worker) that live outside the module and break ./... loading.
	for _, mod := range []struct{ name, dir string }{
		{"omes", repoDir},
		{"Go worker", workerDir},
	} {
		paths, err := goPackageDirs(ctx, mod.dir)
		if err != nil {
			return err
		}

		fmt.Printf("Formatting %s...\n", mod.name)
		if err := runCommandInDir(ctx, mod.dir, "mise", goLintArgs("fmt", paths)...); err != nil {
			return err
		}

		fmt.Printf("Linting %s...\n", mod.name)
		if err := runCommandInDir(ctx, mod.dir, "mise", goLintArgs("run", paths)...); err != nil {
			return err
		}
	}

	fmt.Println("✅ Go lint-and-format completed successfully!")
	return nil
}

// goLintArgs builds the args for `mise exec -- golangci-lint <sub> <paths...>`.
func goLintArgs(sub string, paths []string) []string {
	return append([]string{"exec", "--", "golangci-lint", sub}, paths...)
}

// goPackageDirs returns the directories of the Go packages in the module rooted
// at moduleDir (via `go list`), excluding vendored node_modules trees. `go list
// -e` tolerates the load errors those stray sources would otherwise cause.
func goPackageDirs(ctx context.Context, moduleDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-e", "-f", "{{.Dir}}", "./...")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go list in %s: %w\n%s", moduleDir, err, out)
	}

	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.Contains(line, "/node_modules/") {
			continue
		}
		dirs = append(dirs, line)
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no Go packages found in %s", moduleDir)
	}
	return dirs, nil
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

	harnessDir := workerDir + "/harness"

	fmt.Println("Formatting Python harness...")
	if err := runCommandInDir(ctx, harnessDir, "uv", "run", "poe", "format"); err != nil {
		return err
	}

	fmt.Println("Linting Python harness...")
	if err := runCommandInDir(ctx, harnessDir, "uv", "run", "poe", "lint"); err != nil {
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

	fmt.Println("Formatting TypeScript harness...")
	if err := runCommandInDir(
		ctx,
		workerDir,
		"npm",
		"run",
		"-w",
		"@temporalio/omes-project-harness",
		"format",
	); err != nil {
		return err
	}

	fmt.Println("Linting TypeScript harness...")
	if err := runCommandInDir(
		ctx,
		workerDir,
		"npm",
		"run",
		"-w",
		"@temporalio/omes-project-harness",
		"lint",
	); err != nil {
		return err
	}

	fmt.Println("Compiling TypeScript harness...")
	if err := runCommandInDir(
		ctx,
		workerDir,
		"npm",
		"run",
		"-w",
		"@temporalio/omes-project-harness",
		"typecheck",
	); err != nil {
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

	harnessDir := workerDir + "/harness"

	fmt.Println("Formatting Ruby harness...")
	if err := runCommandInDir(ctx, harnessDir, "bundle", "exec", "rubocop", "-A"); err != nil {
		return err
	}

	fmt.Println("Linting Ruby harness...")
	if err := runCommandInDir(ctx, harnessDir, "bundle", "exec", "rubocop"); err != nil {
		return err
	}

	fmt.Println("Type checking Ruby harness...")
	if err := runCommandInDir(ctx, harnessDir, "bundle", "exec", "steep", "check"); err != nil {
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
	if err := runCommandInDir(
		ctx,
		workerDir,
		"dotnet",
		"build",
		"--configuration",
		"Library",
	); err != nil {
		return err
	}

	fmt.Println("✅ .NET lint-and-format completed successfully!")
	return nil
}
