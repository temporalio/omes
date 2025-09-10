package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var supportedLanguages = []string{
	"dotnet", "go", "java", "python", "typescript",
}

var supportedTools = []string{
	"dotnet", "go", "java", "node", "npm", "protoc", "python", "rust",
}

var sdkVersionMap = map[string]string{
	"go":         "GO_SDK_VERSION",
	"java":       "JAVA_SDK_VERSION",
	"python":     "PYTHON_SDK_VERSION",
	"typescript": "TYPESCRIPT_SDK_VERSION",
	"dotnet":     "DOTNET_SDK_VERSION",
}

// getSdkVersion returns the SDK version for a given language
func getSdkVersion(language string, versions map[string]string) string {
	if key, ok := sdkVersionMap[language]; ok {
		return versions[key]
	}
	return "unknown"
}

// checkCommand verifies that a command is available in PATH
func checkCommand(cmd string) error {
	if _, err := exec.LookPath(cmd); err != nil {
		return fmt.Errorf("%s is not installed. Please install %s first", cmd, cmd)
	}
	return nil
}

// runCommand executes a command and pipes output to stdout/stderr
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandInDir executes a command in a specific directory
func runCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getRootDir returns the project root directory (current working directory when running go run ./cmd/dev)
func getRootDir() (string, error) {
	return os.Getwd()
}

// loadVersions parses versions.env file and returns a map of version variables
func loadVersions() (map[string]string, error) {
	// Find versions.env file by walking up the directory tree
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for dir != "/" {
		versionsFile := filepath.Join(dir, "versions.env")
		if _, err := os.Stat(versionsFile); err == nil {
			return parseVersionsFile(versionsFile)
		}
		dir = filepath.Dir(dir)
	}

	return nil, fmt.Errorf("versions.env not found")
}

// parseVersionsFile reads and parses a versions.env file
func parseVersionsFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	versions := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		versions[key] = value
	}

	return versions, scanner.Err()
}

// checkMise verifies that mise is installed
func checkMise() error {
	if _, err := exec.LookPath("mise"); err != nil {
		return fmt.Errorf("mise is not installed. Please install mise first: https://mise.jdx.dev/getting-started.html")
	}
	return nil
}

// validatePrerequisites checks if required tools are available before running operations
func validatePrerequisites(languages []string) error {
	// Check for mise first if any language is specified
	if len(languages) > 0 {
		if _, err := exec.LookPath("mise"); err != nil {
			fmt.Printf("Warning: mise not found. Some operations may require mise for version management.\n")
		}
	}

	// Check language-specific tools
	for _, lang := range languages {
		var requiredTools []string

		switch lang {
		case "go":
			requiredTools = []string{"go"}
		case "java":
			requiredTools = []string{"java"}
		case "python":
			requiredTools = []string{"python", "uv"}
		case "typescript":
			requiredTools = []string{"node", "npm"}
		case "dotnet":
			requiredTools = []string{"dotnet"}
		}

		for _, tool := range requiredTools {
			if _, err := exec.LookPath(tool); err != nil {
				fmt.Printf("Warning: %s not found. %s operations may fail.\n", tool, lang)
			}
		}
	}

	return nil
}

// validateLanguageTools checks that all required tools for a language are available
func validateLanguageTools(language string) error {
	var requiredTools []string

	switch language {
	case "go":
		requiredTools = []string{"go"}
	case "java":
		requiredTools = []string{"java"}
	case "python":
		requiredTools = []string{"python", "uv", "poe"}
	case "typescript":
		requiredTools = []string{"node", "npm"}
	case "dotnet":
		requiredTools = []string{"dotnet"}
	case "rust":
		requiredTools = []string{"cargo"}
	case "protoc":
		requiredTools = []string{"protoc"}
	}

	for _, tool := range requiredTools {
		if err := checkCommand(tool); err != nil {
			return err
		}
	}
	return nil
}
