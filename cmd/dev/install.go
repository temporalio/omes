package main

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

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
			var tools []string
			if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
				tools = supportedTools
			} else {
				for _, tool := range args {
					if !slices.Contains(supportedTools, tool) {
						return fmt.Errorf("unsupported tool: %s", tool)
					}
				}
				tools = args
			}
			return runInstallTools(cmd.Context(), tools)
		},
	}
}

func runInstallTools(ctx context.Context, tools []string) error {
	if err := checkMise(); err != nil {
		return err
	}

	fmt.Println("Installing", strings.Join(tools, ", "))

	for _, tool := range tools {
		var err error
		switch tool {
		case "buf":
			err = installBuf(ctx)
		case "dotnet":
			err = installDotnet(ctx)
		case "go":
			err = installGo(ctx)
		case "java":
			err = installJava(ctx)
		case "python":
			err = installPython(ctx)
		case "node":
			err = installNode(ctx)
		case "ruby":
			err = installRuby(ctx)
		case "rust":
			err = installRust(ctx)
		case "protoc":
			err = installProtoc(ctx)
		default:
			err = fmt.Errorf("unsupported tool: %s", tool)
		}

		if err != nil {
			return fmt.Errorf("failed to install %s: %v", tool, err)
		}
	}

	return nil
}

func installDotnet(ctx context.Context) error {
	version, err := getVersion("dotnet")
	if err != nil {
		return err
	}
	if err := installViaMise(ctx, "dotnet", version); err != nil {
		return err
	}

	workerDir, err := getTargetDir("dotnet")
	if err != nil {
		return err
	}
	fmt.Println("Installing .NET worker dependencies...")
	if err := runCommandInDir(ctx, workerDir, "dotnet", "restore"); err != nil {
		return err
	}
	fmt.Println("✅ .NET worker dependencies installed successfully!")

	return nil
}

func installGo(ctx context.Context) error {
	workerDir, err := getTargetDir("go")
	if err != nil {
		return err
	}
	fmt.Println("Installing Go worker dependencies...")
	if err := runCommandInDir(ctx, workerDir, "go", "mod", "tidy"); err != nil {
		return err
	}
	fmt.Println("✅ Go worker dependencies installed successfully!")

	return nil
}

func installJava(ctx context.Context) error {
	version, err := getVersion("java")
	if err != nil {
		return err
	}
	if err := installViaMise(ctx, "java", version); err != nil {
		return err
	}

	workerDir, err := getTargetDir("java")
	if err != nil {
		return err
	}
	fmt.Println("Installing Java worker dependencies...")
	if err := runCommandInDir(ctx, workerDir, "./gradlew", "build", "--dry-run"); err != nil {
		return err
	}
	fmt.Println("✅ Java worker dependencies installed successfully!")

	return nil
}

func installPython(ctx context.Context) error {
	version, err := getVersion("python")
	if err != nil {
		return err
	}

	uvVersion, err := getVersion("uv")
	if err != nil {
		return err
	}

	if err := installViaMise(ctx, "python", version); err != nil {
		return err
	}

	if err := installViaMise(ctx, "uv", uvVersion); err != nil {
		return err
	}

	fmt.Println("Installing poethepoet...")
	if err := runCommand(ctx, "uv", "tool", "install", "poethepoet"); err != nil {
		return fmt.Errorf("failed to install poethepoet: %v", err)
	}
	fmt.Println("✅ poethepoet installed successfully!")

	workerDir, err := getTargetDir("python")
	if err != nil {
		return err
	}
	fmt.Println("Installing Python worker dependencies...")
	if err := runCommandInDir(
		ctx,
		workerDir,
		"uv",
		"sync",
		"--python", version,
		"--all-packages",
		"--all-groups",
	); err != nil {
		return err
	}
	fmt.Println("✅ Python worker dependencies installed successfully!")

	return nil
}

func installNode(ctx context.Context) error {
	version, err := getVersion("node")
	if err != nil {
		return err
	}
	if err := installViaMise(ctx, "node", version); err != nil {
		return err
	}

	workerDir, err := getTargetDir("typescript")
	if err != nil {
		return err
	}
	fmt.Println("Installing TypeScript worker dependencies...")
	if err := runCommandInDir(ctx, workerDir, "npm", "ci"); err != nil {
		return err
	}
	fmt.Println("✅ TypeScript worker dependencies installed successfully!")

	return nil
}

func installRuby(ctx context.Context) error {
	version, err := getVersion("ruby")
	if err != nil {
		return err
	}
	if err := installViaMise(ctx, "ruby", version); err != nil {
		return err
	}

	targetDir, err := getTargetDir("ruby")
	if err != nil {
		return err
	}
	fmt.Println("Installing Ruby worker dependencies...")
	if err := runCommandInDir(ctx, targetDir, "bundle", "install"); err != nil {
		return err
	}
	fmt.Println("✅ Ruby worker dependencies installed successfully!")

	harnessDir := targetDir + "/harness"
	fmt.Println("Installing Ruby harness dependencies...")
	if err := runCommandInDir(ctx, harnessDir, "bundle", "install"); err != nil {
		return err
	}
	fmt.Println("✅ Ruby harness dependencies installed successfully!")
	return nil
}

func installRust(ctx context.Context) error {
	version, err := getVersion("cargo")
	if err != nil {
		return err
	}
	// Use --profile default to ensure rustfmt is included
	return installViaMise(ctx, "rust", version, "--profile", "default")
}

func installProtoc(ctx context.Context) error {
	version, err := getVersion("protoc")
	if err != nil {
		return err
	}

	if err := installViaMise(ctx, "protoc", version); err != nil {
		return err
	}

	// Install Go protoc plugin
	fmt.Println("Installing Go protoc plugin...")
	protocGenGoVersion, err := getVersion("protoc-gen-go")
	if err != nil {
		return err
	}
	if err := runCommand(ctx, "go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@"+protocGenGoVersion); err != nil {
		return fmt.Errorf("failed to install protoc-gen-go: %v", err)
	}
	fmt.Println("✅ protoc-gen-go", protocGenGoVersion, "installed successfully!")

	return nil
}

func installBuf(ctx context.Context) error {
	fmt.Println("Installing buf...")
	if err := runCommand(ctx, "go", "install", "github.com/bufbuild/buf/cmd/buf@latest"); err != nil {
		return fmt.Errorf("failed to install buf: %v", err)
	}
	fmt.Println("✅ buf installed successfully!")
	return nil
}

func installViaMise(ctx context.Context, tool, version string, extraArgs ...string) error {
	fmt.Println("Installing", tool, version)

	args := []string{"use", fmt.Sprintf("%s@%s", tool, version)}
	args = append(args, extraArgs...)
	cmd := exec.CommandContext(ctx, "mise", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mise command failed: %v\nOutput: %s", err, output)
	}

	fmt.Println("✅", tool, version, "installed successfully!")
	return nil
}
