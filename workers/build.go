package workers

import (
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"go.uber.org/zap"
)

type Builder struct {
	DirName    string
	SdkOptions clioptions.SdkOptions
	Logger     *zap.SugaredLogger
	stdout     io.Writer
	stderr     io.Writer
}

func (b *Builder) Build(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	if b.DirName == "" {
		return nil, fmt.Errorf("output directory name required")
	} else if strings.ContainsAny(b.DirName, `/\`) {
		return nil, fmt.Errorf("output directory name is not a full path, it is a single name")
	}

	buildDir := filepath.Join(baseDir, b.DirName)
	b.Logger.Infof("Building %v program at %v", b.SdkOptions.Language, buildDir)
	if err := os.Mkdir(buildDir, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed creating directory: %w", err)
	}

	if b.stdout == nil {
		b.stdout = &logWriter{logger: b.Logger}
	}
	if b.stderr == nil {
		b.stderr = &logWriter{logger: b.Logger}
	}

	switch b.SdkOptions.Language {
	case clioptions.LangGo:
		return b.buildGo(ctx, baseDir)
	case clioptions.LangPython:
		return b.buildPython(ctx, baseDir)
	case clioptions.LangJava:
		return b.buildJava(ctx, baseDir)
	case clioptions.LangTypeScript:
		return b.buildTypeScript(ctx, baseDir)
	case clioptions.LangDotNet:
		return b.buildDotNet(ctx, baseDir)
	default:
		return nil, fmt.Errorf("unrecognized language %v", b.SdkOptions.Language)
	}
}

func (b *Builder) buildGo(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir: baseDir,
		DirName: b.DirName,
		Version: b.SdkOptions.Version,
		Stdout:  b.stdout,
		Stderr:  b.stderr,
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

func (b *Builder) buildPython(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// If version not provided, try to read it from pyproject.toml
	version := b.SdkOptions.Version
	if version == "" {
		cmd := exec.CommandContext(ctx, "uv", "tree", "--quiet", "--depth", "0", "--package", "temporalio")
		cmd.Dir = baseDir
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed running uv tree: %w", err)
		}
		outStr := strings.TrimSpace(string(out))
		if strings.HasPrefix(outStr, "temporalio v") {
			version = outStr[len("temporalio v"):]
		}
		if version == "" {
			return nil, fmt.Errorf("version not found in uv tree output")
		}
	}

	prog, err := sdkbuild.BuildPythonProgram(ctx, sdkbuild.BuildPythonProgramOptions{
		BaseDir: baseDir,
		DirName: b.DirName,
		Version: version,
		Stdout:  b.stdout,
		Stderr:  b.stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *Builder) buildJava(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	prog, err := sdkbuild.BuildJavaProgram(ctx, sdkbuild.BuildJavaProgramOptions{
		BaseDir:           baseDir,
		DirName:           b.DirName,
		Version:           b.SdkOptions.Version,
		MainClass:         "io.temporal.omes.Main",
		HarnessDependency: "io.temporal:omes:0.1.0",
		Build:             true,
		Stdout:            b.stdout,
		Stderr:            b.stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *Builder) buildTypeScript(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// If version not provided, try to read it from package.json
	version := b.SdkOptions.Version
	if version == "" {
		b, err := os.ReadFile(filepath.Join(baseDir, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("failed reading package.json: %w", err)
		}
		for line := range strings.SplitSeq(string(b), "\n") {
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

	// Prep the TypeScript runner to be built
	cmd := exec.Command("npm", "install")
	cmd.Dir = baseDir
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("npm install in ./workers/typescript failed: %w", err)
	}

	cmd = exec.Command("npm", "run", "build")
	cmd.Dir = baseDir
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("npm run build in ./workers/typescript failed: %w", err)
	}

	prog, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir:        baseDir,
		Version:        version,
		TSConfigPaths:  map[string][]string{"@temporalio/omes": {"src/omes.ts"}},
		DirName:        b.DirName,
		ApplyToCommand: nil,
		Includes:       []string{"../src/**/*.ts", "../src/protos/json-module.js", "../src/protos/root.js"},
		MoreDependencies: map[string]string{
			"winston": "^3.11.0",
		},
		Stdout: b.stdout,
		Stderr: b.stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *Builder) buildDotNet(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	// Get version from dotnet.csproj if not present
	version := b.SdkOptions.Version
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
			return nil, fmt.Errorf("failed writing temp csproj: %w", err)
		}
		omesProjPath = "../Temporalio.Omes.temp.csproj"
	}

	prog, err := sdkbuild.BuildDotNetProgram(ctx, sdkbuild.BuildDotNetProgramOptions{
		BaseDir:         baseDir,
		Version:         version,
		DirName:         b.DirName,
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
		Stdout: b.stdout,
		Stderr: b.stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func BaseDir(repoDir string, lang clioptions.Language) string {
	return filepath.Join(repoDir, "workers", lang.String())
}
