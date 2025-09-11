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

	versions, err := loadVersions()
	if err != nil {
		return err
	}

	fmt.Println("Installing", strings.Join(tools, ", "))

	for _, tool := range tools {
		var err error
		switch tool {
		case "dotnet":
			err = installDotnet(ctx, versions)
		case "go":
			// already installed
		case "java":
			err = installJava(ctx, versions)
		case "python":
			err = installPython(ctx, versions)
		case "node":
			err = installNode(ctx, versions)
		case "npm":
			err = installNpm(ctx, versions)
		case "rust":
			err = installRust(ctx, versions)
		case "protoc":
			err = installProtoc(ctx, versions)
		default:
			err = fmt.Errorf("unsupported tool: %s", tool)
		}

		if err != nil {
			return fmt.Errorf("failed to install %s: %v", tool, err)
		}
	}

	return nil
}

func installDotnet(ctx context.Context, versions map[string]string) error {
	version := versions["DOTNET_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for dotnet")
	}
	return installViaMise(ctx, "dotnet", version)
}

func installGo(ctx context.Context, versions map[string]string) error {
	version := versions["GO_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for go")
	}
	return installViaMise(ctx, "go", version)
}

func installJava(ctx context.Context, versions map[string]string) error {
	version := versions["JAVA_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for java")
	}
	return installViaMise(ctx, "java", version)
}

func installPython(ctx context.Context, versions map[string]string) error {
	version := versions["PYTHON_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for python")
	}

	uvVersion := versions["UV_VERSION"]
	if uvVersion == "" {
		return fmt.Errorf("version not found for uv")
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

	return nil
}

func installNode(ctx context.Context, versions map[string]string) error {
	version := versions["NODE_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for node")
	}
	return installViaMise(ctx, "node", version)
}

func installRust(ctx context.Context, versions map[string]string) error {
	version := versions["RUST_TOOLCHAIN"]
	if version == "" {
		return fmt.Errorf("version not found for rust")
	}
	return installViaMise(ctx, "rust", version)
}

func installProtoc(ctx context.Context, versions map[string]string) error {
	version := versions["PROTOC_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for protoc")
	}

	if err := installViaMise(ctx, "protoc", version); err != nil {
		return err
	}

	// Install Go protoc plugin
	fmt.Println("Installing Go protoc plugin...")
	protocGenGoVersion := versions["PROTOC_GEN_GO_VERSION"]
	if protocGenGoVersion == "" {
		return fmt.Errorf("version not found for protoc-gen-go")
	}
	if err := runCommand(ctx, "go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@"+protocGenGoVersion); err != nil {
		return fmt.Errorf("failed to install protoc-gen-go: %v", err)
	}
	fmt.Println("✅ protoc-gen-go", protocGenGoVersion, "installed successfully!")

	return nil
}

func installUv(ctx context.Context, versions map[string]string) error {
	version := versions["UV_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for uv")
	}
	return installViaMise(ctx, "uv", version)
}

func installNpm(ctx context.Context, versions map[string]string) error {
	// npm comes with node, so install node
	return installNode(ctx, versions)
}

func installViaMise(ctx context.Context, tool, version string) error {
	fmt.Println("Installing", tool, version)

	cmd := exec.CommandContext(ctx, "mise", "use", fmt.Sprintf("%s@%s", tool, version))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mise command failed: %v\nOutput: %s", err, output)
	}

	fmt.Println("✅", tool, version, "installed successfully!")
	return nil
}
