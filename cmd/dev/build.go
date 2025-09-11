package main

import (
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
				return runBuildKitchensink()
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
			return runBuildWorkers(languages)
		},
	}

	return cmd
}

func runBuildWorkers(languages []string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	versions, err := loadVersions()
	if err != nil {
		return err
	}

	fmt.Printf("Building %s worker(s)...\n", strings.Join(languages, ", "))

	for _, lang := range languages {
		if err := buildWorker(lang, rootDir, versions); err != nil {
			return fmt.Errorf("failed to build %s: %v", lang, err)
		}
	}

	return nil
}

func buildWorker(language, rootDir string, versions map[string]string) error {
	fmt.Printf("\n===========================================\n")
	fmt.Printf("Building %s worker\n", language)
	fmt.Printf("===========================================\n")

	if err := validateLanguageTools(language); err != nil {
		return err
	}

	sdkVersion := getSdkVersion(language, versions)
	fmt.Printf("Building %s worker (SDK %s)...\n", language, sdkVersion)

	if err := buildTemporalOmes(rootDir); err != nil {
		return err
	}

	if err := runWorkerScenario(rootDir, language, sdkVersion); err != nil {
		return err
	}

	if err := buildWorkerImage(rootDir, language, sdkVersion); err != nil {
		return err
	}

	fmt.Printf("✅ %s worker build completed successfully!\n", language)
	return nil
}

func buildTemporalOmes(rootDir string) error {
	fmt.Println("Building temporal-omes...")
	return runCommandInDir(rootDir, "go", "build", "-o", "temporal-omes", "./cmd")
}

func runWorkerScenario(rootDir, language, sdkVersion string) error {
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
	return runCommandInDir(rootDir, args[0], args[1:]...)
}

func buildWorkerImage(rootDir, language, sdkVersion string) error {
	fmt.Println("Building worker image...")
	args := []string{
		"./temporal-omes", "build-worker-image",
		"--language", language,
		"--version", "v" + sdkVersion,
		"--tag-as-latest",
	}
	return runCommandInDir(rootDir, args[0], args[1:]...)
}

func runBuildKitchensink() error {
	if err := validateLanguageTools("rust"); err != nil {
		return err
	}
	if err := validateLanguageTools("protoc"); err != nil {
		return err
	}
	if err := validateLanguageTools("typescript"); err != nil {
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
	fmt.Printf("Setting PROTOC=%s\n", protocPath)

	if err := runCommandInDir(kitchenSinkGenDir, "cargo", "build"); err != nil {
		return err
	}

	fmt.Println("Generating TypeScript protos...")
	typescriptWorkerDir := filepath.Join(rootDir, "workers", "typescript")
	if err := runCommandInDir(typescriptWorkerDir, "npm", "install"); err != nil {
		return err
	}

	if err := runCommandInDir(typescriptWorkerDir, "npm", "run", "proto-gen"); err != nil {
		return err
	}

	fmt.Println("✅ Kitchen-sink proto build complete!")
	return nil
}
