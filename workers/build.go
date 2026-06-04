package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/clioptions"
	"go.uber.org/zap"
)

type Builder struct {
	DirName     string
	ProjectName string
	SdkOptions  clioptions.SdkOptions
	Logger      *zap.SugaredLogger
	stdout      io.Writer
	stderr      io.Writer
}

func (b *Builder) Build(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	if b.DirName == "" {
		return nil, fmt.Errorf("output directory name required")
	} else if strings.ContainsAny(b.DirName, `/\`) {
		return nil, fmt.Errorf("output directory name is not a full path, it is a single name")
	}

	if b.ProjectName != "" {
		baseDir = ProjectDir(baseDir, b.ProjectName)
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
	case clioptions.LangRuby:
		return b.buildRuby(ctx, baseDir)
	default:
		return nil, fmt.Errorf("unrecognized language %v", b.SdkOptions.Language)
	}
}

func (b *Builder) buildGo(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	goMod := `module github.com/temporalio/omes-worker

go 1.20

require github.com/temporalio/omes v1.0.0
require github.com/temporalio/omes/workers/go v1.0.0
require github.com/temporalio/omes/workers/go/harness/api v0.0.0

replace github.com/temporalio/omes => ../../../
replace github.com/temporalio/omes/workers/go => ../
replace github.com/temporalio/omes/workers/go/harness/api => ../harness/api`

	goMain := `package main

import "github.com/temporalio/omes/workers/go/apps"

func main() {
	apps.Main()
}`

	prog, err := sdkbuild.BuildGoProgram(ctx, sdkbuild.BuildGoProgramOptions{
		BaseDir:        baseDir,
		DirName:        b.DirName,
		Version:        b.SdkOptions.Version,
		Stdout:         b.stdout,
		Stderr:         b.stderr,
		GoModContents:  goMod,
		GoMainContents: goMain,
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
		MainClass:         "io.temporal.omes.apps.Registry",
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
		packageJSON, err := os.ReadFile(filepath.Join(baseDir, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("failed reading package.json: %w", err)
		}

		var pkg struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.Unmarshal(packageJSON, &pkg); err != nil {
			return nil, fmt.Errorf("failed parsing package.json: %w", err)
		}
		// Pick a single temporal dependency, assumption is that the version for
		// other temporal dependency versions will match.
		const temporalTypeScriptSDKPackage = "@temporalio/client"
		version = pkg.Dependencies[temporalTypeScriptSDKPackage]
		if version == "" {
			return nil, fmt.Errorf("version not found in package.json for %s", temporalTypeScriptSDKPackage)
		}
	}

	// Generated TypeScript proto files are source-local and intentionally not
	// checked in. sdkbuild installs and builds a separate prepared package, so
	// its pnpm install does not run workers/typescript scripts; generate those
	// source assets before sdkbuild compiles them through ../ includes.
	cmd := exec.Command("npm", "install")
	cmd.Dir = baseDir
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("npm install in ./workers/typescript failed: %w", err)
	}

	cmd = exec.Command("npm", "run", "proto-gen")
	cmd.Dir = baseDir
	cmd.Stdout = b.stdout
	cmd.Stderr = b.stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("npm run proto-gen in ./workers/typescript failed: %w", err)
	}

	prog, err := sdkbuild.BuildTypeScriptProgram(ctx, sdkbuild.BuildTypeScriptProgramOptions{
		BaseDir:        baseDir,
		Version:        version,
		TSConfigPaths:  map[string][]string{"@temporalio/omes": {filepath.ToSlash(filepath.Join("..", "apps", "registry.ts"))}},
		DirName:        b.DirName,
		ApplyToCommand: nil,
		Includes: []string{
			"../apps/**/*.ts",
			"../harness/*.ts",
			"../harness/api/**/*.ts",
			"../workerlib/**/*.ts",
			"../workerlib/kitchensink/protos/json-module.js",
			"../workerlib/kitchensink/protos/root.js",
		},
		MoreDependencies: map[string]string{
			"@grpc/proto-loader": "^0.8.0",
			"winston":            "^3.11.0",
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
		Configuration:   "Release",
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

func (b *Builder) buildRuby(ctx context.Context, baseDir string) (sdkbuild.Program, error) {
	if strings.ContainsAny(b.SdkOptions.Version, `/\`) {
		if err := b.prepareRubySDKSource(ctx, b.SdkOptions.Version); err != nil {
			return nil, err
		}
	}

	options := sdkbuild.BuildRubyProgramOptions{
		BaseDir:   baseDir,
		SourceDir: baseDir,
		DirName:   b.DirName,
		Version:   b.SdkOptions.Version,
		Stdout:    b.stdout,
		Stderr:    b.stderr,
	}

	prog, err := sdkbuild.BuildRubyProgram(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed preparing: %w", err)
	}
	return prog, nil
}

func (b *Builder) prepareRubySDKSource(ctx context.Context, version string) error {
	sdkPath, err := filepath.Abs(version)
	if err != nil {
		return fmt.Errorf("unable to make sdk path absolute: %w", err)
	}

	gemPath := filepath.Join(sdkPath, "temporalio")
	if _, err := os.Stat(filepath.Join(gemPath, "Rakefile")); err != nil {
		gemPath = sdkPath
	}
	if _, err := os.Stat(filepath.Join(gemPath, "Rakefile")); err != nil {
		return nil
	}

	bundleDir := filepath.Join(gemPath, ".bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return fmt.Errorf("failed creating Ruby SDK .bundle dir: %w", err)
	}
	bundleConfig := "---\nBUNDLE_PATH: \"vendor/bundle\"\n"
	if err := os.WriteFile(filepath.Join(bundleDir, "config"), []byte(bundleConfig), 0644); err != nil {
		return fmt.Errorf("failed writing Ruby SDK .bundle/config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "bundle", "install")
	cmd.Dir = gemPath
	cmd.Stdout = b.stdout
	cmd.Stderr = b.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed installing Ruby SDK dependencies: %w", err)
	}
	return nil
}

func BaseDir(repoDir string, lang clioptions.Language) string {
	return filepath.Join(repoDir, "workers", lang.String())
}

func ProjectDir(baseDir string, projectName string) string {
	projectPath := fmt.Sprintf("projects/tests/%s", projectName)
	return filepath.Join(baseDir, projectPath)
}
