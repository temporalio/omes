package sdkbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// BuildTypeScriptProgramOptions are options for BuildTypeScriptProgram.
type BuildTypeScriptProgramOptions struct {
	// Directory that will have a temporary directory created underneath.
	BaseDir string
	// Required version. If it contains a slash it is assumed to be a path with a
	// package.json. Otherwise it is a specific version (with leading "v" is
	// trimmed if present).
	Version string
	// Required set of paths to include in tsconfig.json paths for the project.
	// The paths should be relative to one-directory beneath BaseDir.
	TSConfigPaths map[string][]string
	// If present, this directory is expected to exist beneath base dir. Otherwise
	// a temporary dir is created.
	DirName string
	// If present, applied to build commands before run. May be called multiple
	// times for a single build.
	ApplyToCommand func(context.Context, *exec.Cmd) error
	// If present, overrides the default "include" array in tsconfig.json.
	Includes []string
	// If present, overrides the default "exclude" array in tsconfig.json.
	Excludes []string
	// If present, add additional dependencies -> version string to package.json.
	MoreDependencies map[string]string
}

// TypeScriptProgram is a TypeScript-specific implementation of Program.
type TypeScriptProgram struct {
	dir string
}

var _ Program = (*TypeScriptProgram)(nil)

// BuildTypeScriptProgram builds a TypeScript program. If completed
// successfully, this can be stored and re-obtained via
// TypeScriptProgramFromDir() with the Dir() value (but the entire BaseDir must
// be present too).
func BuildTypeScriptProgram(ctx context.Context, options BuildTypeScriptProgramOptions) (*TypeScriptProgram, error) {
	if options.BaseDir == "" {
		return nil, fmt.Errorf("base dir required")
	} else if options.Version == "" {
		return nil, fmt.Errorf("version required")
	} else if len(options.TSConfigPaths) == 0 {
		return nil, fmt.Errorf("at least one tsconfig path required")
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

	// Create package JSON
	var packageJSONDepStr string
	if strings.ContainsAny(options.Version, `/\`) {
		if _, err := os.Stat(filepath.Join(options.Version, "package.json")); err != nil {
			return nil, fmt.Errorf("failed finding package.json in version dir: %w", err)
		}

		// Have to build the local repo
		if st, err := os.Stat(filepath.Join(options.Version, "node_modules")); err != nil || !st.IsDir() {
			// Only install dependencies, avoid triggerring any post install build scripts
			cmd := exec.CommandContext(ctx, "npm", "ci", "--ignore-scripts")
			cmd.Dir = options.Version
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed installing SDK deps: %w", err)
			}

			// Build the SDK, ignore the unused `create` package as a mostly insignificant micro optimisation.
			cmd = exec.CommandContext(ctx, "npm", "run", "build", "--", "--ignore", "@temporalio/create")
			cmd.Dir = options.Version
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed building SDK: %w", err)
			}
		}

		// Create package.json updates
		localPath, err := filepath.Abs(options.Version)
		if err != nil {
			return nil, fmt.Errorf("cannot get absolute path from version path: %w", err)
		}
		pkgs := []string{"activity", "client", "common", "internal-workflow-common",
			"internal-non-workflow-common", "proto", "worker", "workflow"}
		for _, pkg := range pkgs {
			pkgPath := "file:" + filepath.Join(localPath, "packages", pkg)
			packageJSONDepStr += fmt.Sprintf(`"@temporalio/%v": %q,`, pkg, pkgPath)
			packageJSONDepStr += "\n    "
		}
	} else {
		version := strings.TrimPrefix(options.Version, "v")
		pkgs := []string{"activity", "client", "common", "worker", "workflow"}
		for _, pkg := range pkgs {
			packageJSONDepStr += fmt.Sprintf(`    "@temporalio/%v": %q,`, pkg, version) + "\n"
		}
	}
	moreDeps := ""
	for dep, version := range options.MoreDependencies {
		moreDeps += fmt.Sprintf(`    "%v": "%v",`, dep, version) + "\n"
	}

	packageJSON := `{
  "name": "program",
  "private": true,
  "scripts": {
    "build": "tsc --build"
  },
  "dependencies": {
    ` + packageJSONDepStr + `
	` + moreDeps + `
    "commander": "^8.3.0",
    "ms": "^3.0.0-canary.1",
    "proto3-json-serializer": "^1.1.1",
    "uuid": "^8.3.2"
  },
  "devDependencies": {
    "@tsconfig/node16": "^1.0.0",
    "@types/node": "^16.11.59",
    "@types/uuid": "^8.3.4",
    "tsconfig-paths": "^3.12.0",
    "typescript": "^4.4.2"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644); err != nil {
		return nil, fmt.Errorf("failed writing package.json: %w", err)
	}

	// Create tsconfig
	var tsConfigPathStr string
	for name, paths := range options.TSConfigPaths {
		if len(paths) == 0 {
			return nil, fmt.Errorf("harness path slice is empty")
		}
		tsConfigPathStr += fmt.Sprintf("%q: [", name)
		for i, path := range paths {
			if i > 0 {
				tsConfigPathStr += ", "
			}
			tsConfigPathStr += strconv.Quote(path)
		}
		tsConfigPathStr += "],\n      "
	}
	includes := []string{"../features/**/*.ts", "../harness/ts/**/*.ts"}
	if len(options.Includes) > 0 {
		includes = options.Includes
	}
	excludes := []string{"../node_modules", "../harness/go", "../harness/java"}
	if len(options.Excludes) > 0 {
		excludes = options.Excludes
	}
	quotedIncludes := make([]string, len(includes))
	for i, include := range includes {
		quotedIncludes[i] = strconv.Quote(include)
	}
	quotedExcludes := make([]string, len(excludes))
	for i, exclude := range excludes {
		quotedExcludes[i] = strconv.Quote(exclude)
	}
	tsConfig := `{
  "extends": "@tsconfig/node16/tsconfig.json",
  "version": "4.4.2",
  "compilerOptions": {
    "baseUrl": ".",
    "outDir": "./tslib",
    "rootDirs": ["../", "."],
    "paths": {
      ` + tsConfigPathStr + `
      "*": ["node_modules/*", "node_modules/@types/*"]
    },
    "typeRoots": ["node_modules/@types"],
    "module": "commonjs",
    "moduleResolution": "node",
    "sourceMap": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "allowJs": true
  },
  "include": [` + strings.Join(quotedIncludes, ", ") + `],
  "exclude": [` + strings.Join(quotedExcludes, ", ") + `]
}`
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(tsConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed writing tsconfig.json: %w", err)
	}

	// Install
	cmd := exec.CommandContext(ctx, "npm", "install")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if options.ApplyToCommand != nil {
		if err := options.ApplyToCommand(ctx, cmd); err != nil {
			return nil, err
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed installing: %w", err)
	}

	// Compile
	cmd = exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if options.ApplyToCommand != nil {
		if err := options.ApplyToCommand(ctx, cmd); err != nil {
			return nil, err
		}
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed compiling: %w", err)
	}

	success = true
	return &TypeScriptProgram{dir}, nil
}

// TypeScriptProgramFromDir recreates the TypeScript program from a Dir() result
// of a BuildTypeScriptProgram(). Note, the base directory of dir when it was
// built must also be present.
func TypeScriptProgramFromDir(dir string) (*TypeScriptProgram, error) {
	// Quick sanity check on the presence of package.json here
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		return nil, fmt.Errorf("failed finding package.json in dir: %w", err)
	}
	return &TypeScriptProgram{dir}, nil
}

// Dir is the directory to run in.
func (t *TypeScriptProgram) Dir() string { return t.dir }

// NewCommand makes a new Node command. The first argument needs to be the name
// of the script.
func (t *TypeScriptProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	args = append([]string{"-r", "tsconfig-paths/register"}, args...)
	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = t.dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd, nil
}
