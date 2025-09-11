package main

import (
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
			return runInstallTools(tools)
		},
	}
}

func runInstallTools(tools []string) error {
	if err := checkMise(); err != nil {
		return err
	}

	versions, err := loadVersions()
	if err != nil {
		return err
	}

	fmt.Printf("Installing %s...\n", strings.Join(tools, ", "))

	for _, tool := range tools {
		var err error
		switch tool {
		case "dotnet":
			err = installDotnet(versions)
		case "go":
			// already installed
		case "java":
			err = installJava(versions)
		case "python":
			err = installPython(versions)
		case "node":
			err = installNode(versions)
		case "npm":
			err = installNpm(versions)
		case "rust":
			err = installRust(versions)
		case "protoc":
			err = installProtoc(versions)
		case "uv":
			err = installUv(versions)
		default:
			err = fmt.Errorf("unsupported tool: %s", tool)
		}

		if err != nil {
			return fmt.Errorf("failed to install %s: %v", tool, err)
		}
	}

	return nil
}

func installDotnet(versions map[string]string) error {
	version := versions["DOTNET_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for dotnet")
	}
	return installViaMise("dotnet", version)
}

func installGo(versions map[string]string) error {
	version := versions["GO_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for go")
	}
	return installViaMise("go", version)
}

func installJava(versions map[string]string) error {
	version := versions["JAVA_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for java")
	}
	return installViaMise("java", version)
}

func installPython(versions map[string]string) error {
	version := versions["PYTHON_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for python")
	}

	uvVersion := versions["UV_VERSION"]
	if uvVersion == "" {
		return fmt.Errorf("version not found for uv")
	}

	if err := installViaMise("python", version); err != nil {
		return err
	}

	if err := installViaMise("uv", uvVersion); err != nil {
		return err
	}

	fmt.Println("Installing poethepoet...")
	if err := runCommand("uv", "tool", "install", "poethepoet"); err != nil {
		return fmt.Errorf("failed to install poethepoet: %v", err)
	}
	fmt.Printf("✅ poethepoet installed successfully!\n")

	return nil
}

func installNode(versions map[string]string) error {
	version := versions["NODE_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for node")
	}
	return installViaMise("node", version)
}

func installRust(versions map[string]string) error {
	version := versions["RUST_TOOLCHAIN"]
	if version == "" {
		return fmt.Errorf("version not found for rust")
	}
	return installViaMise("rust", version)
}

func installProtoc(versions map[string]string) error {
	version := versions["PROTOC_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for protoc")
	}

	if err := installViaMise("protoc", version); err != nil {
		return err
	}

	// Install Go protoc plugin
	fmt.Println("Installing Go protoc plugin...")
	protocGenGoVersion := versions["PROTOC_GEN_GO_VERSION"]
	if protocGenGoVersion == "" {
		return fmt.Errorf("version not found for protoc-gen-go")
	}
	if err := runCommand("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@"+protocGenGoVersion); err != nil {
		return fmt.Errorf("failed to install protoc-gen-go: %v", err)
	}
	fmt.Printf("✅ protoc-gen-go %s installed successfully!\n", protocGenGoVersion)

	return nil
}

func installUv(versions map[string]string) error {
	version := versions["UV_VERSION"]
	if version == "" {
		return fmt.Errorf("version not found for uv")
	}
	return installViaMise("uv", version)
}

func installNpm(versions map[string]string) error {
	// npm comes with node, so install node
	return installNode(versions)
}

func installViaMise(tool, version string) error {
	fmt.Printf("Installing %s %s...\n", tool, version)

	cmd := exec.Command("mise", "use", fmt.Sprintf("%s@%s", tool, version))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mise command failed: %v\nOutput: %s", err, output)
	}

	fmt.Printf("✅ %s %s installed successfully!\n", tool, version)
	return nil
}
