package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var installToolDependencies = map[string][]string{
	"node":   {"node", "pnpm"},
	"protoc": {"protoc", "protoc-gen-go"},
	"python": {"python", "uv", "pipx:poethepoet"},
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [tool...]",
		Short: "Install development tools using mise",
		Long: fmt.Sprintf(`Install development tools using mise

Supported tools: %s

Examples:
  dev install                        # All tools (default)
  dev install all                    # All tools
  dev install go                     # Single tool
  dev install go java python         # Multiple tools`, strings.Join(supportedTools, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
				return runInstallAll(cmd.Context())
			}
			return runInstallTools(cmd.Context(), args)
		},
	}
}

func runInstallAll(ctx context.Context) error {
	if err := checkMise(); err != nil {
		return err
	}
	if err := runMise(ctx, "install"); err != nil {
		return err
	}
	for _, tool := range supportedTools {
		if err := setupInstalledTool(ctx, tool); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}
	return nil
}

func runInstallTools(ctx context.Context, tools []string) error {
	logicalTools, err := validateAndDeduplicateTools(tools)
	if err != nil {
		return err
	}
	if err := checkMise(); err != nil {
		return err
	}

	configTools := expandInstallTools(logicalTools)
	if err := runMise(ctx, append([]string{"install"}, configTools...)...); err != nil {
		return err
	}

	for _, tool := range logicalTools {
		if err := setupInstalledTool(ctx, tool); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}
	return nil
}

func validateAndDeduplicateTools(tools []string) ([]string, error) {
	var result []string
	for _, tool := range tools {
		if !slices.Contains(supportedTools, tool) {
			return nil, fmt.Errorf("unsupported tool: %s", tool)
		}
		if !slices.Contains(result, tool) {
			result = append(result, tool)
		}
	}
	return result, nil
}

func expandInstallTools(tools []string) []string {
	var result []string
	for _, tool := range tools {
		dependencies, ok := installToolDependencies[tool]
		if !ok {
			dependencies = []string{tool}
		}
		for _, dependency := range dependencies {
			if !slices.Contains(result, dependency) {
				result = append(result, dependency)
			}
		}
	}
	return result
}

func setupInstalledTool(ctx context.Context, tool string) error {
	switch tool {
	case "dotnet":
		workerDir, err := getTargetDir("dotnet")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "dotnet", "restore")
	case "go":
		workerDir, err := getTargetDir("go")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "go", "mod", "tidy")
	case "java":
		workerDir, err := getTargetDir("java")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "./gradlew", "build", "--dry-run")
	case "python":
		workerDir, err := getTargetDir("python")
		if err != nil {
			return err
		}
		version, err := getVersion("python")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "uv", "sync", "--python", version, "--all-packages", "--all-groups")
	case "node":
		workerDir, err := getTargetDir("typescript")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "npm", "ci")
	case "ruby":
		workerDir, err := getTargetDir("ruby")
		if err != nil {
			return err
		}
		return runMiseExecInDir(ctx, workerDir, "bundle", "install")
	case "buf", "protoc", "rust":
		return nil
	default:
		return fmt.Errorf("unsupported tool: %s", tool)
	}
}
