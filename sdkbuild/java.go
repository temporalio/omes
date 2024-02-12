package sdkbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/otiai10/copy"
)

// BuildJavaProgramOptions are options for BuildJavaProgram.
type BuildJavaProgramOptions struct {
	// Directory that will have a temporary directory created underneath. This
	// should be a gradle project with a build.gradle, a gradlew executable, etc.
	BaseDir string
	// If not set, not put in build.gradle which means gradle will automatically
	// use latest. If set and contains a slash it is assumed to be a path,
	// otherwise it is a specific version (with leading "v" is trimmed if
	// present).
	Version string
	// Required Gradle "implementation" dependency name of BaseDir. This is
	// usually "<group>:<project-name>:<version>" with each value replaced with
	// proper values.
	HarnessDependency string
	// Required fully-qualified class name for main.
	MainClass string
	// If true, performs an eager build. This is often just to prime system-level
	// caches and do extra validation, the build won't be used by NewCommand.
	Build bool
	// If present, this directory is expected to exist beneath base dir. Otherwise
	// a temporary dir is created.
	DirName string
	// If present, applied to build commands before run. May be called multiple
	// times for a single build.
	ApplyToCommand func(context.Context, *exec.Cmd) error
}

// JavaProgram is a Java-specific implementation of Program.
type JavaProgram struct {
	dir string
}

var _ Program = (*JavaProgram)(nil)

// BuildJavaProgram builds a Java program. If completed successfully, this can
// be stored and re-obtained via JavaProgramFromDir() with the Dir() value (but
// the entire BaseDir must be present too).
func BuildJavaProgram(ctx context.Context, options BuildJavaProgramOptions) (*JavaProgram, error) {
	if options.BaseDir == "" {
		return nil, fmt.Errorf("base dir required")
	} else if _, err := os.Stat(filepath.Join(options.BaseDir, "build.gradle")); err != nil {
		return nil, fmt.Errorf("failed finding build.gradle in base dir: %w", err)
	} else if options.HarnessDependency == "" {
		return nil, fmt.Errorf("harness dependency required")
	} else if options.MainClass == "" {
		return nil, fmt.Errorf("main class required")
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
	j := &JavaProgram{dir}

	// If we depend on SDK via path, built it and get the JAR
	isPathDep := strings.ContainsAny(options.Version, `/\`)
	if isPathDep {
		cmd := j.buildGradleCommand(ctx, options.Version, false, options.ApplyToCommand, "jar", "gatherRuntimeDeps")
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed building Java SDK: %w", err)
		}
		// Copy JARs to sdkjars dir
		sdkJarsDir := filepath.Join(dir, "sdkjars")
		if err := copy.Copy(filepath.Join(options.Version, "temporal-sdk/build/libs"), sdkJarsDir); err != nil {
			return nil, fmt.Errorf("failed copying lib JARs: %w", err)
		}
		if err := copy.Copy(filepath.Join(options.Version, "temporal-sdk/build/runtimeDeps"), sdkJarsDir); err != nil {
			return nil, fmt.Errorf("failed copying runtime JARs: %w", err)
		}
	}

	// Create build.gradle and settings.gradle
	temporalSDKDependency := ""
	if isPathDep {
		temporalSDKDependency = "implementation fileTree(dir: 'sdkjars', include: ['*.jar'])"
	} else if options.Version != "" {
		temporalSDKDependency = fmt.Sprintf("implementation 'io.temporal:temporal-sdk:%v'",
			strings.TrimPrefix(options.Version, "v"))
	}
	buildGradle := `
plugins {
    id 'application'
}

repositories {
    maven {
        url "https://oss.sonatype.org/content/repositories/snapshots/"
    }
    mavenCentral()
}

dependencies {
    implementation '` + options.HarnessDependency + `'
    ` + temporalSDKDependency + `
}

application {
    mainClass = '` + options.MainClass + `'
}`
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(buildGradle), 0644); err != nil {
		return nil, fmt.Errorf("failed writing build.gradle: %w", err)
	}
	settingsGradle := fmt.Sprintf("rootProject.name = '%v'", filepath.Base(dir))
	if err := os.WriteFile(filepath.Join(dir, "settings.gradle"), []byte(settingsGradle), 0644); err != nil {
		return nil, fmt.Errorf("failed writing settings.gradle: %w", err)
	}

	// Build if wanted
	if options.Build {
		// This is really only to prime the system-level caches. The build won't be
		// used by run.
		cmd := j.buildGradleCommand(ctx, dir, true,
			options.ApplyToCommand, "--no-daemon", "--include-build", "../", "build")
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	success = true
	return j, nil
}

// JavaProgramFromDir recreates the Java program from a Dir() result of a
// BuildJavaProgram(). Note, the base directory of dir when it was built must
// also be present.
func JavaProgramFromDir(dir string) (*JavaProgram, error) {
	// Quick sanity check on the presence of build.gradle here _and_ in base
	if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err != nil {
		return nil, fmt.Errorf("failed finding build.gradle in dir: %w", err)
	} else if _, err := os.Stat(filepath.Join(dir, "../build.gradle")); err != nil {
		return nil, fmt.Errorf("failed finding build.gradle in base dir: %w", err)
	}
	return &JavaProgram{dir}, nil
}

// Dir is the directory to run in.
func (j *JavaProgram) Dir() string { return j.dir }

// NewCommand makes a new command for the given args.
func (j *JavaProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Since args have to be a string, we disallow quotes
	var argsStr string
	for _, arg := range args {
		if strings.ContainsAny(arg, `"'`) {
			return nil, fmt.Errorf("java argument cannot contain single or double quote")
		}
		if argsStr != "" {
			argsStr += " "
		}
		argsStr += "'" + arg + "'"
	}
	return j.buildGradleCommand(ctx, j.dir, true, nil, "--include-build", "../", "run", "--args", argsStr), nil
}

func (j *JavaProgram) buildGradleCommand(
	ctx context.Context,
	dir string,
	gradleInParentDir bool,
	applyToCommand func(context.Context, *exec.Cmd) error,
	args ...string,
) *exec.Cmd {
	// Prepare exe whether windows or not
	var exe string
	if runtime.GOOS == "windows" {
		exe = "cmd.exe"
		if gradleInParentDir {
			args = append([]string{"/C", "..\\gradlew"}, args...)
		} else {
			args = append([]string{"/C", "gradlew"}, args...)
		}
	} else {
		exe = "/bin/sh"
		if gradleInParentDir {
			args = append([]string{"../gradlew"}, args...)
		} else {
			args = append([]string{"gradlew"}, args...)
		}
	}

	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}
