package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var cleanCommands = map[string][]string{
	"go":         {"go", "clean"},
	"java":       {"./gradlew", "clean"},
	"python":     {"uv", "clean"},
	"typescript": {"npm", "run", "clean"},
	"dotnet":     {"dotnet", "clean"},
}

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [language...]",
		Short: "Clean build artifacts for specified language(s)",
		Long: fmt.Sprintf(`Clean build artifacts and temporary directories for one or more languages.

Supported languages: %s

Examples:
  dev clean                          # Clean all languages (default)
  dev clean all                      # Clean all languages
  dev clean go                       # Clean Go only
  dev clean go java python           # Clean multiple languages`, strings.Join(supportedLanguages, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			var languages []string
			cleanAll := len(args) == 0 || (len(args) == 1 && args[0] == "all")

			if cleanAll {
				languages = supportedLanguages
			} else {
				// Validate languages
				for _, lang := range args {
					if !slices.Contains(supportedLanguages, lang) {
						return fmt.Errorf("unsupported language: %s", lang)
					}
				}
				languages = args
			}

			return runClean(languages, cleanAll)
		},
	}

	return cmd
}

func cleanLanguage(language, rootDir string) error {
	workerDir := filepath.Join(rootDir, "workers", language)
	if _, err := os.Stat(workerDir); os.IsNotExist(err) {
		fmt.Println("Warning:", language, "worker directory not found, skipping")
		return nil
	}

	fmt.Println("Cleaning", language)

	cleanCmd, ok := cleanCommands[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}
	return runCommandInDir(workerDir, cleanCmd[0], cleanCmd[1:]...)
}

func runClean(languages []string, cleanAll bool) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	if cleanAll {
		fmt.Println("Cleaning all build artifacts and temporary directories...")

		// Remove omes-temp-* directories
		fmt.Println("Removing omes-temp-* directories...")
		workersDir := filepath.Join(rootDir, "workers")
		if err := removeTempDirs(workersDir); err != nil {
			fmt.Printf("Warning: failed to remove temp directories: %v\n", err)
		}
	} else {
		fmt.Println("Cleaning", strings.Join(languages, ", "))
	}

	for _, lang := range languages {
		if err := cleanLanguage(lang, rootDir); err != nil {
			if cleanAll {
				fmt.Printf("Warning: failed to clean %s: %v\n", lang, err)
			} else {
				return fmt.Errorf("failed to clean %s: %v", lang, err)
			}
		}
	}

	fmt.Println("✅ Clean completed")
	return nil
}

func removeTempDirs(workersDir string) error {
	return filepath.Walk(workersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if info.IsDir() && strings.HasPrefix(info.Name(), "omes-temp-") {
			fmt.Println("Removing", path)
			if err := os.RemoveAll(path); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", path, err)
			}
			return filepath.SkipDir
		}
		return nil
	})
}
