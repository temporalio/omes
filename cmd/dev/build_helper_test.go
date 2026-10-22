package main

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// The default --platform value names an OS, so docker does not fall back to the host OS.
func TestPlatformFlagDefaultsToLinux(t *testing.T) {
	var b baseImageBuilder
	b.addBaseCLIFlags(pflag.NewFlagSet("test", pflag.ContinueOnError))

	require.Equal(t, []string{"linux/amd64"}, b.platforms)
}

// A platform given as a bare architecture is qualified with linux, because docker otherwise resolves the OS from
// the host and produces a darwin/amd64 image that fails to export on macOS.
func TestBuildDockerArgsQualifiesBarePlatform(t *testing.T) {
	b := baseImageBuilder{platforms: []string{"amd64", "arm64"}}

	args, err := b.buildDockerArgs("dockerfiles/cli.Dockerfile", true, nil)
	require.NoError(t, err)
	require.Equal(t, "linux/amd64,linux/arm64", flagValue(t, args, "--platform"))
}

// A platform that already names an OS is passed through untouched.
func TestBuildDockerArgsKeepsQualifiedPlatform(t *testing.T) {
	b := baseImageBuilder{platforms: []string{"linux/amd64", "linux/arm/v7", "windows/amd64"}}

	args, err := b.buildDockerArgs("dockerfiles/cli.Dockerfile", true, nil)
	require.NoError(t, err)
	require.Equal(t, "linux/amd64,linux/arm/v7,windows/amd64", flagValue(t, args, "--platform"))
}

func TestBuildDockerArgsPreservesCacheSpecifications(t *testing.T) {
	var b baseImageBuilder
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	b.addBaseCLIFlags(flags)
	require.NoError(t, flags.Parse([]string{
		"--cache-from=type=gha,scope=omes-rust,version=2",
		"--cache-from=type=registry,ref=example/cache",
		"--cache-to=type=gha,scope=omes-rust,version=2,mode=max",
	}))

	args, err := b.buildDockerArgs("dockerfiles/rust.Dockerfile", false, nil)
	require.NoError(t, err)
	require.Equal(t, []string{
		"buildx", "build", "--pull", "--file", "dockerfiles/rust.Dockerfile",
		"--platform", "linux/amd64", "--load",
		"--cache-from", "type=gha,scope=omes-rust,version=2",
		"--cache-from", "type=registry,ref=example/cache",
		"--cache-to", "type=gha,scope=omes-rust,version=2,mode=max",
	}, args)
}

func TestBuildDockerArgsPushByDigest(t *testing.T) {
	for _, prefix := range []string{"", "example.com/team"} {
		t.Run(prefix, func(t *testing.T) {
			var b baseImageBuilder
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			b.addBaseCLIFlags(flags)
			require.NoError(t, flags.Parse([]string{
				"--push-by-digest",
				"--platform=arm64",
				"--repo-prefix=" + prefix,
				"--image-name=worker",
				"--image-tag=release",
				"--metadata-file=metadata.json",
			}))

			args, err := b.buildDockerArgs("Dockerfile", true, nil)
			require.NoError(t, err)
			repository := "worker"
			if prefix != "" {
				repository = prefix + "/" + repository
			}
			require.Equal(t, "type=image,name="+repository+",push-by-digest=true,name-canonical=true,push=true", flagValue(t, args, "--output"))
			require.Equal(t, "metadata.json", flagValue(t, args, "--metadata-file"))
			require.Equal(t, "linux/arm64", flagValue(t, args, "--platform"))
			require.NotContains(t, args, "--tag")
			require.NotContains(t, args, "--load")
			require.NotContains(t, args, "--push")
			require.Equal(t, []string{repository + ":release"}, b.generateImageTags())
		})
	}
}

func TestBuildDockerArgsRejectsInvalidPushByDigest(t *testing.T) {
	for _, tt := range []struct {
		name      string
		allowPush bool
		platforms []string
		saveImage string
		wantError string
	}{
		{"local", false, []string{"amd64"}, "", "requires pushing to registry"},
		{"multiplatform", true, []string{"amd64", "arm64"}, "", "requires exactly one platform"},
		{"no platform", true, nil, "", "requires exactly one platform"},
		{"save", true, []string{"amd64"}, "image.tar", "cannot be combined with save-image"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := baseImageBuilder{pushByDigest: true, platforms: tt.platforms, saveImage: tt.saveImage}
			_, err := b.buildDockerArgs("Dockerfile", tt.allowPush, nil)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestBuildDockerArgsPreservesNormalExports(t *testing.T) {
	for _, tt := range []struct {
		name      string
		platforms []string
		allowPush bool
		export    string
	}{
		{"local", []string{"amd64"}, false, "--load"},
		{"single platform publish", []string{"amd64"}, true, "--load"},
		{"multiplatform publish", []string{"amd64", "arm64"}, true, "--push"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := baseImageBuilder{
				platforms: tt.platforms, imageName: "worker", tags: []string{"release"},
				metadataFile: "metadata.json",
			}
			args, err := b.buildDockerArgs("Dockerfile", tt.allowPush, nil)
			require.NoError(t, err)
			require.Contains(t, args, tt.export)
			require.NotContains(t, args, "--output")
			require.Equal(t, "worker:release", flagValue(t, args, "--tag"))
			require.Equal(t, "metadata.json", flagValue(t, args, "--metadata-file"))
		})
	}
}

// flagValue returns the argument following the named flag.
func flagValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i, arg := range args {
		if arg == flag {
			require.Less(t, i+1, len(args), "%s has no value", flag)
			return args[i+1]
		}
	}
	t.Fatalf("%s not found in %v", flag, args)
	return ""
}
