package sdkbuild

import (
	"context"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildDotNetProgramOptions are options for BuildDotNetProgram.
type BuildDotNetProgramOptions struct {
	// Directory that will have a temporary directory created underneath.
	BaseDir string
	// Required version. If it contains a slash, it is assumed to be a path to the
	// base of the repo (and will have a src/Temporalio/Temporalio.csproj child).
	// Otherwise it is a NuGet version.
	Version string
	// If present, this directory is expected to exist beneath base dir. Otherwise
	// a temporary dir is created.
	DirName string
	// Required Program.cs content. If not set, no Program.cs is created (so it)
	ProgramContents string
	// Required csproj content. This should not contain a dependency on Temporalio
	// because this adds a package/project reference near the end.
	CsprojContents string
}

// DotNetProgram is a .NET-specific implementation of Program.
type DotNetProgram struct {
	dir string
}

var _ Program = (*DotNetProgram)(nil)

func BuildDotNetProgram(ctx context.Context, options BuildDotNetProgramOptions) (*DotNetProgram, error) {
	if options.BaseDir == "" {
		return nil, fmt.Errorf("base dir required")
	} else if options.Version == "" {
		return nil, fmt.Errorf("version required")
	} else if options.ProgramContents == "" {
		return nil, fmt.Errorf("program contents required")
	} else if options.CsprojContents == "" {
		return nil, fmt.Errorf("csproj contents required")
	}

	// Create temp dir if needed that we will remove if creating is unsuccessful
	success := false
	var dir string
	if options.DirName != "" {
		dir = filepath.Join(options.BaseDir, options.DirName)
	} else {
		var err error
		dir, err = os.MkdirTemp(options.BaseDir, "program-")
		if err != nil {
			return nil, fmt.Errorf("failed making temp dir: %w", err)
		}
		defer func() {
			if !success {
				// Intentionally swallow error
				_ = os.RemoveAll(dir)
			}
		}()
	}

	// Create program.csproj
	var depLine string
	// Slash means it is a path
	if strings.ContainsAny(options.Version, `/\`) {
		// Get absolute path of csproj file
		absCsproj, err := filepath.Abs(filepath.Join(options.Version, "src/Temporalio/Temporalio.csproj"))
		if err != nil {
			return nil, fmt.Errorf("cannot make absolute path from version: %w", err)
		} else if _, err := os.Stat(absCsproj); err != nil {
			return nil, fmt.Errorf("cannot find version path of %v: %w", absCsproj, err)
		}
		depLine = `<ProjectReference Include="` + html.EscapeString(absCsproj) + `" />`
		// Need to build this csproj first
		cmd := exec.CommandContext(ctx, "dotnet", "build", absCsproj)
		cmd.Dir = dir
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed dotnet build of csproj in version: %w", err)
		}
	} else {
		depLine = `<PackageReference Include="Temporalio" Version="` +
			html.EscapeString(strings.TrimPrefix(options.Version, "v")) + `" />`
	}
	// Add the item group for the Temporalio dep just before the ending project tag
	endProjectTag := strings.LastIndex(options.CsprojContents, "</Project>")
	if endProjectTag == -1 {
		return nil, fmt.Errorf("no ending project tag found in csproj contents")
	}
	csproj := options.CsprojContents[:endProjectTag] + "\n  <ItemGroup>\n    " + depLine +
		"\n  </ItemGroup>\n" + options.CsprojContents[endProjectTag:]
	if err := os.WriteFile(filepath.Join(dir, "program.csproj"), []byte(csproj), 0644); err != nil {
		return nil, fmt.Errorf("failed writing program.csproj: %w", err)
	}

	// Create Program.cs
	if err := os.WriteFile(filepath.Join(dir, "Program.cs"), []byte(options.ProgramContents), 0644); err != nil {
		return nil, fmt.Errorf("failed writing Program.cs: %w", err)
	}

	// Build it into build folder
	cmd := exec.CommandContext(ctx, "dotnet", "build", "--output", "build")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed dotnet build: %w", err)
	}

	// All good
	success = true
	return &DotNetProgram{dir}, nil
}

// DotNetProgramFromDir recreates the Go program from a Dir() result of a
// BuildDotNetProgram().
func DotNetProgramFromDir(dir string) (*DotNetProgram, error) {
	// Quick sanity check on the presence of program.csproj
	if _, err := os.Stat(filepath.Join(dir, "program.csproj")); err != nil {
		return nil, fmt.Errorf("failed finding program.csproj in dir: %w", err)
	}
	return &DotNetProgram{dir}, nil
}

// Dir is the directory to run in.
func (d *DotNetProgram) Dir() string { return d.dir }

// NewCommand makes a new command for the given args.
func (d *DotNetProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	exe := "./build/program"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = d.dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd, nil
}
