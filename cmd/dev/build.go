package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func buildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [language...]",
		Short: "Build worker(s) for specified language(s)",
		Long: fmt.Sprintf(`Build worker(s) for specified language(s)

Supported languages: %s

Examples:
  dev build                          # All languages (default)
  dev build all                      # All languages
  dev build go                       # Single language
  dev build go java python           # Multiple languages
  dev build kitchensink              # Build kitchen-sink proto generator`, strings.Join(supportedLanguages, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && args[0] == "kitchensink" {
				return runBuildKitchensink(cmd.Context())
			}

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
			return runBuildWorkers(cmd.Context(), languages)
		},
	}

	return cmd
}

func runBuildWorkers(ctx context.Context, languages []string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	versions, err := loadVersions()
	if err != nil {
		return err
	}

	fmt.Println("Building", strings.Join(languages, ", "), "worker(s)...")

	for _, lang := range languages {
		if err := buildWorker(ctx, lang, rootDir, versions); err != nil {
			return fmt.Errorf("failed to build %s: %v", lang, err)
		}
	}

	return nil
}

func buildWorker(ctx context.Context, language, rootDir string, versions map[string]string) error {
	fmt.Println("\n===========================================")
	fmt.Println("Building", language, "worker")
	fmt.Println("===========================================")

	if err := validateLanguageTools(ctx, language); err != nil {
		return err
	}

	sdkVersion := getSdkVersion(language, versions)
	fmt.Println("Building", language, "worker (SDK", sdkVersion+")...")

	if err := buildTemporalOmes(ctx, rootDir); err != nil {
		return err
	}

	if err := runWorkerScenario(ctx, rootDir, language, sdkVersion); err != nil {
		return err
	}

	if err := buildWorkerImage(ctx, rootDir, language, sdkVersion); err != nil {
		return err
	}

	fmt.Println("✅", language, "worker build completed successfully!")
	return nil
}

func buildTemporalOmes(ctx context.Context, rootDir string) error {
	fmt.Println("Building temporal-omes...")
	return runCommandInDir(ctx, rootDir, "go", "build", "-o", "temporal-omes", "./cmd")
}

func runWorkerScenario(ctx context.Context, rootDir, language, sdkVersion string) error {
	fmt.Println("Running local scenario with worker...")
	scenario := "workflow_with_single_noop_activity"
	iterations := "5"

	args := []string{
		"./temporal-omes", "run-scenario-with-worker",
		"--scenario", scenario,
		"--log-level", "debug",
		"--language", language,
		"--embedded-server",
		"--iterations", iterations,
		"--version", "v" + sdkVersion,
	}
	return runCommandInDir(ctx, rootDir, args[0], args[1:]...)
}

func buildWorkerImage(ctx context.Context, rootDir, language, sdkVersion string) error {
	fmt.Println("Building worker image...")
	args := []string{
		"./temporal-omes", "build-worker-image",
		"--language", language,
		"--version", "v" + sdkVersion,
		"--tag-as-latest",
	}
	return runCommandInDir(ctx, rootDir, args[0], args[1:]...)
}

func runBuildKitchensink(ctx context.Context) error {
	if err := validateLanguageTools(ctx, "rust"); err != nil {
		return err
	}
	if err := validateLanguageTools(ctx, "protoc"); err != nil {
		return err
	}
	if err := validateLanguageTools(ctx, "typescript"); err != nil {
		return err
	}

	fmt.Println("Building kitchen-sink proto...")

	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	fmt.Println("Building kitchen-sink-gen...")
	kitchenSinkGenDir := filepath.Join(rootDir, "loadgen", "kitchen-sink-gen")

	// Set PROTOC environment variable to use the correct protoc
	protocPath, _ := exec.LookPath("protoc") // already validated earlier
	os.Setenv("PROTOC", protocPath)
	fmt.Println("Setting PROTOC="+protocPath)

	if err := runCommandInDir(ctx, kitchenSinkGenDir, "cargo", "build"); err != nil {
		return err
	}

	fmt.Println("Generating TypeScript protos...")
	typescriptWorkerDir := filepath.Join(rootDir, "workers", "typescript")
	if err := runCommandInDir(ctx, typescriptWorkerDir, "npm", "install"); err != nil {
		return err
	}

	if err := runCommandInDir(ctx, typescriptWorkerDir, "npm", "run", "proto-gen"); err != nil {
		return err
	}

	fmt.Println("✅ Kitchen-sink proto build complete!")
	return nil
}
