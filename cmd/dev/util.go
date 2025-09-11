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

// Mappings to versions.env
var versionVarNames = map[string]string{
	// Tool versions
	"go":            "GO_VERSION",
	"java":          "JAVA_VERSION",
	"python":        "PYTHON_VERSION",
	"node":          "NODE_VERSION",
	"npm":           "", // not tracked
	"dotnet":        "DOTNET_VERSION",
	"cargo":         "RUST_TOOLCHAIN",
	"protoc":        "PROTOC_VERSION",
	"protoc-gen-go": "PROTOC_GEN_GO_VERSION",
	"uv":            "UV_VERSION",
	"poe":           "", // not tracked

	// SDK versions
	"go-sdk":         "GO_SDK_VERSION",
	"java-sdk":       "JAVA_SDK_VERSION",
	"python-sdk":     "PYTHON_SDK_VERSION",
	"typescript-sdk": "TYPESCRIPT_SDK_VERSION",
	"dotnet-sdk":     "DOTNET_SDK_VERSION",
}

var toolDependencies = map[string][]string{
	"go":         {"go"},
	"java":       {"java"},
	"python":     {"python", "uv", "poe"},
	"typescript": {"node", "npm"},
	"dotnet":     {"dotnet"},
	"rust":       {"cargo"},
	"protoc":     {"protoc", "protoc-gen-go"},
}

var toolVersionCommands = map[string][]string{
	"go":            {"go", "version"},
	"java":          {"java", "-version"},
	"python":        {"python", "--version"},
	"node":          {"node", "--version"},
	"npm":           {"npm", "--version"},
	"dotnet":        {"dotnet", "--version"},
	"cargo":         {"cargo", "--version"},
	"protoc":        {"protoc", "--version"},
	"protoc-gen-go": {"protoc-gen-go", "--version"},
	"uv":            {"uv", "--version"},
	"poe":           {"poe", "--version"},
}

// getSdkVersion returns the SDK version for a given language
func getSdkVersion(language string, versions map[string]string) string {
	if key, ok := versionVarNames[language+"-sdk"]; ok {
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

// runCommandOutput executes a command and returns its output as a string
func runCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
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

// validateLanguageTools checks that all required tools for a language are available
// and prints version and path information for known tools
func validateLanguageTools(language string) error {
	requiredTools, ok := toolDependencies[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}

	versions, err := loadVersions()
	if err != nil {
		return fmt.Errorf("failed to load versions.env: %v", err)
	}

	for _, tool := range requiredTools {
		toolPath, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s is not installed. Please install %s first", tool, tool)
		}

		versionCmd, hasVersion := toolVersionCommands[tool]
		if !hasVersion {
			return fmt.Errorf("no version command defined for tool: %s", tool)
		}

		output, err := runCommandOutput(versionCmd[0], versionCmd[1:]...)
		if err != nil {
			return fmt.Errorf("failed to get version for %s at %s: %v", tool, toolPath, err)
		}

		actualVersion := strings.TrimSpace(output)

		envVar, ok := versionVarNames[tool]
		if !ok {
			return fmt.Errorf("no version env var defined for tool: %s", tool)
		}

		fmt.Printf("using %s\n", tool)
		fmt.Printf("\tpath: %s\n", toolPath)
		fmt.Printf("\tversion: %s\n", actualVersion)

		if envVar != "" {
			expectedVersion := versions[envVar]
			if expectedVersion == "" {
				return fmt.Errorf("no expected version found for %s (env var: %s)", tool, envVar)
			}
			fmt.Printf("\texpected version: %s\n", expectedVersion)
		}
	}
	return nil
}
