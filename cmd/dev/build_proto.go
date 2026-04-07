package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func buildProtoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build-proto",
		Short: "Build proto files",
		Long:  "Build the kitchen-sink and harness proto files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runBuildKitchensink(cmd.Context()); err != nil {
				return err
			}
			return runBuildHarnessProto(cmd.Context())
		},
	}
}

func runBuildKitchensink(ctx context.Context) error {
	fmt.Println("Building kitchen-sink proto...")

	if err := checkTool(ctx, "cargo"); err != nil {
		return err
	}
	if err := checkTool(ctx, "protoc"); err != nil {
		return err
	}
	if err := checkTool(ctx, "node"); err != nil {
		return err
	}

	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}

	fmt.Println("Building kitchen-sink-gen...")
	kitchenSinkGenDir := getKitchenSinkGenDir(repoDir)

	// Set PROTOC environment variable to use the correct protoc
	protocPath, _ := exec.LookPath("protoc") // already validated earlier
	os.Setenv("PROTOC", protocPath)
	fmt.Println("Setting PROTOC=" + protocPath)

	if err := runCommandInDir(ctx, kitchenSinkGenDir, "cargo", "build"); err != nil {
		return err
	}

	fmt.Println("Generating TypeScript protos...")
	typescriptWorkerDir := filepath.Join(repoDir, "workers", "typescript")

	if err := runCommandInDir(ctx, typescriptWorkerDir, "npm", "run", "proto-gen"); err != nil {
		return err
	}

	fmt.Println("✅ Kitchen-sink proto build complete!")
	return nil
}

func runBuildHarnessProto(ctx context.Context) error {
	fmt.Println("Building harness proto...")

	if err := checkTool(ctx, "buf"); err != nil {
		return err
	}

	repoDir, err := getRepoDir()
	if err != nil {
		return err
	}

	harnessProtoDir := filepath.Join(repoDir, "workers", "proto")
	if err := runCommandInDir(ctx, harnessProtoDir, "buf", "generate"); err != nil {
		return err
	}

	fmt.Println("✅ Harness proto build complete!")
	return nil
}
