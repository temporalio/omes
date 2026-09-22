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
