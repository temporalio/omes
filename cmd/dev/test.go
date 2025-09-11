package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
	defaultImageTag = "latest"
	testScenario    = "workflow_with_single_noop_activity"
	testIterations  = "5"
)

func testCmd() *cobra.Command {
	var imageTag string

	cmd := &cobra.Command{
		Use:   "test [language] [local|image]",
		Short: "Test worker for specified language locally or as Docker image",
		Long: fmt.Sprintf(`Test worker for specified language locally or as Docker image

Supported languages: %s

Examples:
  dev test go local                          # Test Go worker locally
  dev test go image                          # Test Go worker Docker image (latest tag)
  dev test go image --image-tag v1.2.3       # Test Go worker Docker image with specific tag
  dev test python local                      # Test Python worker locally
  dev test python image --image-tag latest   # Test Python worker Docker image (latest tag)`, strings.Join(supportedLanguages, ", ")),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			language := args[0]
			testType := args[1]

			if !slices.Contains(supportedLanguages, language) {
				return fmt.Errorf("unsupported language: %s", language)
			}

			if testType != "local" && testType != "image" {
				return fmt.Errorf("test type must be 'local' or 'image', got: %s", testType)
			}

			if imageTag != defaultImageTag && testType != "image" {
				return fmt.Errorf("--image-tag flag can only be used with 'image' test type")
			}

			return runTestWorker(cmd.Context(), language, testType, imageTag)
		},
	}

	cmd.Flags().StringVar(&imageTag, "image-tag", defaultImageTag, "Docker image tag")

	return cmd
}

func runTestWorker(ctx context.Context, language, testType, imageTag string) error {
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	sdkVersion, err := getVersion(language + "-sdk")
	if err != nil {
		return err
	}

	fmt.Println("\n===========================================")
	fmt.Println("Testing", language, "worker ("+testType+")")
	fmt.Println("===========================================")

	switch testType {
	case "local":
		if err := checkTool(ctx, language); err != nil {
			return err
		}
		return testWorkerLocally(ctx, rootDir, language, sdkVersion)
	case "image":
		return testWorkerImage(ctx, rootDir, language, sdkVersion, imageTag)
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}

func testWorkerLocally(ctx context.Context, rootDir, language, sdkVersion string) error {
	args := []string{
		"go", "run", "./cmd", "run-scenario-with-worker",
		"--scenario", testScenario,
		"--log-level", "debug",
		"--language", language,
		"--embedded-server",
		"--iterations", testIterations,
		"--version", "v" + sdkVersion,
	}
	return runCommandInDir(ctx, rootDir, args[0], args[1:]...)
}

func testWorkerImage(ctx context.Context, rootDir, language, sdkVersion, imageTag string) error {
	// Start the worker container
	dockerArgs := []string{
		"run", "--rm", "--detach", "-i", "-p", "10233:10233",
		fmt.Sprintf("omes:%s-%s", language, imageTag),
		"--scenario", testScenario,
		"--log-level", "debug",
		"--language", language,
		"--run-id", fmt.Sprintf("test-%s-%s", language, sdkVersion),
		"--embedded-server-address", "0.0.0.0:10233",
	}
	containerOutput, err := runCommandOutput(ctx, "docker", dockerArgs...)
	if err != nil {
		return fmt.Errorf("failed to start worker container: %v\nDocker output: %s", err, containerOutput)
	}

	containerId := strings.TrimSpace(containerOutput)
	if containerId == "" {
		return fmt.Errorf("docker command succeeded but returned empty container ID. Output: %s", containerOutput)
	}
	fmt.Println("Started container:", containerId)

	defer func() {
		fmt.Println("Cleaning up container:", containerId)
		if killErr := runCommand(context.Background(), "docker", "kill", containerId); killErr != nil {
			fmt.Printf("Warning: failed to kill container %s: %v\n", containerId, killErr)
		}
	}()

	// Run the scenario
	scenarioArgs := []string{
		"go", "run", "./cmd", "run-scenario",
		"--scenario", testScenario,
		"--log-level", "debug",
		"--server-address", "127.0.0.1:10233",
		"--run-id", fmt.Sprintf("test-%s-%s", language, sdkVersion),
		"--connect-timeout", "1m",
		"--iterations", testIterations,
	}
	err = runCommandInDir(ctx, rootDir, scenarioArgs[0], scenarioArgs[1:]...)
	if err != nil {
		return fmt.Errorf("scenario execution failed: %v", err)
	}
	return nil
}
