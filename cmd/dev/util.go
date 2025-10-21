package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	supportedLanguages = []string{
		"dotnet", "go", "java", "python", "typescript",
	}
	supportedTools = []string{
		"dotnet", "go", "java", "node", "protoc", "python", "rust",
	}
	toolDependencies = map[string][]string{
		"python":     {"python3", "uv", "poe"},
		"typescript": {"node"},
		"protoc":     {"protoc-gen-go"},
	}
	toolVersionCommands = map[string][]string{
		"go":            {"go", "version"},
		"java":          {"java", "-version"},
		"python3":       {"python3", "--version"},
		"node":          {"node", "--version"},
		"dotnet":        {"dotnet", "--version"},
		"cargo":         {"cargo", "--version"},
		"protoc":        {"protoc", "--version"},
		"protoc-gen-go": {"protoc-gen-go", "--version"},
		"uv":            {"uv", "--version"},
		"poe":           {"poe", "--version"},
	}
	versionsCache map[string]string
)

// getVersion returns the version for a given tool from versions.env
func getVersion(tool string) (string, error) {
	if versionsCache == nil {
		var err error
		versionsCache, err = loadVersions()
		if err != nil {
			return "", fmt.Errorf("failed to load versions.env: %w", err)
		}
	}

	key := strings.ToLower(strings.ReplaceAll(strings.ToLower(tool), "-", "_")) + "_version"
	version := versionsCache[key]
	if version == "" {
		return "", fmt.Errorf("version not found for %s (var: %s)", tool, key)
	}

	return version, nil
}

// checkCommand verifies that a command is available in PATH
func checkCommand(cmd string) error {
	if _, err := exec.LookPath(cmd); err != nil {
		return fmt.Errorf("%s is not installed. Please install %s first", cmd, cmd)
	}
	return nil
}

// runCommand executes a command and pipes output to stdout/stderr
func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandInDir executes a command in a specific directory
func runCommandInDir(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandOutput executes a command and returns combined stdout and stderr as a string
func runCommandOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// getRepoDir returns the project's root directory
func getRepoDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get source file location")
	}
	sourceDir := filepath.Dir(filename) // cmd/dev
	cmdDir := filepath.Dir(sourceDir)   // cmd
	repoDir := filepath.Dir(cmdDir)     // project root
	return repoDir, nil
}

// getWorkerDir returns the worker directory for the given language
func getWorkerDir(language string) (string, error) {
	repoDir, err := getRepoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoDir, "workers", language), nil
}

// loadVersions parses versions.env file and returns a map of version variables
func loadVersions() (map[string]string, error) {
	repoDir, err := getRepoDir()
	if err != nil {
		return nil, err
	}
	versionsFile := filepath.Join(repoDir, "versions.env")
	if _, err := os.Stat(versionsFile); err == nil {
		return parseVersionsFile(versionsFile)
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

		key := strings.ToLower(strings.TrimSpace(parts[0]))
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

// checkTool checks that the tool and its dependencies are available;
// and prints version and path information.
func checkTool(ctx context.Context, tool string) error {
	var toolsToCheck []string
	if dependencies, hasDeps := toolDependencies[tool]; hasDeps {
		toolsToCheck = dependencies
	} else {
		toolsToCheck = []string{tool}
	}

	for _, tool := range toolsToCheck {
		toolPath, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s is not installed. Please install %s first", tool, tool)
		}

		versionCmd, hasVersion := toolVersionCommands[tool]
		if !hasVersion {
			return fmt.Errorf("no version command defined for tool: %s", tool)
		}

		output, err := runCommandOutput(ctx, versionCmd[0], versionCmd[1:]...)
		if err != nil {
			return fmt.Errorf("failed to get version for %s at %s: %v", tool, toolPath, err)
		}

		actualVersion := strings.TrimSpace(output)

		fmt.Println("using", tool)
		fmt.Println("\tpath:", toolPath)
		fmt.Println("\tversion:", actualVersion)

		expectedVersion, err := getVersion(tool)
		if err == nil && expectedVersion != "" {
			fmt.Println("\texpected version:", expectedVersion)
		}
	}
	return nil
}
