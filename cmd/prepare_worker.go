package main

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"go.uber.org/zap"
)

func prepareWorkerCmd() *cobra.Command {
	b := workerBuilder{createDir: true}
	cmd := &cobra.Command{
		Use:   "prepare-worker",
		Short: "Build worker ready to run",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := b.build(cmd.Context()); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("dir-name")
	cmd.MarkFlagRequired("language")
	return cmd
}

type workerBuilder struct {
	logger         *zap.SugaredLogger
	dirName        string
	language       string
	version        string
	loggingOptions cmdoptions.LoggingOptions
	createDir      bool
}

func (b *workerBuilder) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&b.dirName, "dir-name", "", "Directory name for prepared worker")
	fs.StringVar(&b.language, "language", "", "Language to prepare a worker for")
	fs.StringVar(&b.version, "version", "", "Version to prepare a worker for - treated as path if slash present")
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *workerBuilder) build(ctx context.Context) (sdkbuild.Program, error) {
	if b.dirName == "" {
		return nil, fmt.Errorf("directory name required")
	} else if strings.ContainsAny(b.dirName, `/\`) {
		return nil, fmt.Errorf("dir name is not a full path, it is a single name")
	}
	if b.logger == nil {
		b.logger = b.loggingOptions.MustCreateLogger()
	}
	lang, err := normalizeLangName(b.language)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(rootDir(), "workers", lang)
	b.logger.Infof("Building %v program at %v", lang, filepath.Join(baseDir, b.dirName))
	// Check or create dir
	if b.createDir {
		if err := os.Mkdir(filepath.Join(baseDir, b.dirName), 0755); err != nil {
			return nil, fmt.Errorf("failed creating directory: %w", err)
		}
	} else {
		if _, err := os.Stat(filepath.Join(baseDir, b.dirName)); err != nil {
			return nil, fmt.Errorf("failed finding directory: %w", err)
		}
	}
	switch lang {
	case "go":
		return b.buildGo(ctx, baseDir)
	case "python":
		return b.buildPython(ctx, baseDir)
	case "java":
		return b.buildJava(ctx, baseDir)
	case "typescript":
		return b.buildTypeScript(ctx, baseDir)
	case "dotnet":
		return b.buildDotNet(ctx, baseDir)
	default:
		return nil, fmt.Errorf("unrecognized language %v", lang)
	}
}

func (b *workerBuilder) buildGo(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir: baseDir,
		DirName: b.dirName,
		Version: b.version,
		GoModContents: `module github.com/temporalio/omes-worker

go 1.20

require github.com/temporalio/omes v1.0.0
require github.com/temporalio/omes/workers/go v1.0.0

replace github.com/temporalio/omes => ../../../
replace github.com/temporalio/omes/workers/go => ../`,
		GoMainContents: `package main

import "github.com/temporalio/omes/workers/go/worker"

func main() {
	worker.Main()
}`,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *workerBuilder) buildPython(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// If version not provided, try to read it from pyproject.toml
	version := b.version
	if version == "" {
		b, err := os.ReadFile(filepath.Join(baseDir, "pyproject.toml"))
		if err != nil {
			return nil, fmt.Errorf("failed reading pyproject.toml: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "temporalio = ") {
				version = line[strings.Index(line, `"`)+1 : strings.LastIndex(line, `"`)]
				break
			}
		}
		if version == "" {
			return nil, fmt.Errorf("version not found in pyproject.toml")
		}
	}

	// Build
	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir:        baseDir,
		DirName:        b.dirName,
		Version:        version,
		DependencyName: "omes",
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *workerBuilder) buildJava(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	prog, err := sdkbuild.BuildJavaProgram(ctx, sdkbuild.BuildJavaProgramOptions{
		BaseDir:           baseDir,
		DirName:           b.dirName,
		Version:           b.version,
		MainClass:         "io.temporal.omes.Main",
		HarnessDependency: "io.temporal:omes:0.1.0",
		Build:             true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *workerBuilder) buildTypeScript(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// If version not provided, try to read it from package.json
	version := b.version
	if version == "" {
		b, err := os.ReadFile(filepath.Join(baseDir, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("failed reading package.json: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "\"temporalio:\"") || strings.HasPrefix(line, "\"@temporalio/") {
				split := strings.Split(line, "\"")
				version = split[len(split)-2]
				break
			}
		}
		if version == "" {
			return nil, fmt.Errorf("version not found in package.json")
		}
	}
	prog, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir:        baseDir,
		Version:        version,
		TSConfigPaths:  map[string][]string{"@temporalio/omes": {"src/omes.ts"}},
		DirName:        b.dirName,
		ApplyToCommand: nil,
		Includes:       []string{"../src/**/*.ts", "../src/protos/json-module.js", "../src/protos/root.js"},
		MoreDependencies: map[string]string{
			"winston": "^3.11.0",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *workerBuilder) buildDotNet(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// Get version from dotnet.csproj if not present
	version := b.version
	csprojBytes, err := os.ReadFile(filepath.Join(baseDir, "Temporalio.Omes.csproj"))
	if err != nil {
		return nil, fmt.Errorf("failed reading dotnet.csproj: %w", err)
	}
	if version == "" {
		const prefix = `<PackageReference Include="Temporalio" Version="`
		csproj := string(csprojBytes)
		beginIndex := strings.Index(csproj, prefix)
		if beginIndex == -1 {
			return nil, fmt.Errorf("cannot find Temporal dependency in csproj")
		}
		beginIndex += len(prefix)
		length := strings.Index(csproj[beginIndex:], `"`)
		version = csproj[beginIndex : beginIndex+length]
	}
	omesProjPath := "../Temporalio.Omes.csproj"

	// Delete any existing temp csproj
	tempCsProjPath := filepath.Join(baseDir, "Temporalio.Omes.temp.csproj")
	if err := os.Remove(tempCsProjPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed removing temp csproj: %w", err)
	}

	// Prepare replaced csproj if using path-dependency
	if strings.ContainsAny(version, `/\`) {
		// Get absolute path of csproj file
		absCsproj, err := filepath.Abs(filepath.Join(version, "src/Temporalio/Temporalio.csproj"))
		if err != nil {
			return nil, fmt.Errorf("cannot make absolute path from version: %w", err)
		} else if _, err := os.Stat(absCsproj); err != nil {
			return nil, fmt.Errorf("cannot find version path of %v: %w", absCsproj, err)
		}
		depLine := `<ProjectReference Include="` + html.EscapeString(absCsproj) + `" />`

		// Remove whole original package reference tag
		csproj := string(csprojBytes)
		beginIndex := strings.Index(csproj, `<PackageReference Include="Temporalio"`)
		packageRefStr := "</PackageReference>"
		endIndex := strings.Index(csproj[beginIndex:], packageRefStr) + beginIndex
		csproj = csproj[:beginIndex] + depLine + csproj[endIndex+len(packageRefStr):]

		// Write new csproj
		if err := os.WriteFile(tempCsProjPath, []byte(csproj), 0644); err != nil {
			if err != nil {
				return nil, fmt.Errorf("failed writing temp csproj: %w", err)
			}
		}
		omesProjPath = "../Temporalio.Omes.temp.csproj"
	}

	prog, err := sdkbuild.BuildDotNetProgram(ctx, sdkbuild.BuildDotNetProgramOptions{
		BaseDir:         baseDir,
		Version:         version,
		DirName:         b.dirName,
		ProgramContents: "await Temporalio.Omes.App.RunAsync(args);",
		CsprojContents: `<Project Sdk="Microsoft.NET.Sdk">
			<PropertyGroup>
				<OutputType>Exe</OutputType>
				<TargetFramework>net8.0</TargetFramework>
			</PropertyGroup>
			<ItemGroup>
				<ProjectReference Include="` + omesProjPath + `" />
			</ItemGroup>
		</Project>`,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func rootDir() string {
	_, currFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(currFile))
}

func normalizeLangName(lang string) (string, error) {
	switch lang {
	case "go", "python", "java", "typescript", "dotnet":
	case "py":
		lang = "python"
	case "ts":
		lang = "typescript"
	case "cs":
		lang = "dotnet"
	default:
		return "", fmt.Errorf("invalid language %q, must be one of: [go, python, java, typescript, dotnet]", lang)
	}
	return lang, nil
}
