package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDockerArgsGitHubActionsCache(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		t.Setenv(buildKitCacheScopeEnv, "")
		builder := baseImageBuilder{platforms: []string{"amd64"}}

		args, err := builder.buildDockerArgs("Dockerfile", false, nil)
		require.NoError(t, err)
		require.Equal(t, []string{
			"buildx", "build", "--pull", "--file", "Dockerfile", "--platform", "amd64", "--load",
		}, args)
	})

	t.Run("enabled", func(t *testing.T) {
		t.Setenv(buildKitCacheScopeEnv, "omes-ci-typescript-linux-amd64")
		builder := baseImageBuilder{platforms: []string{"amd64"}}

		args, err := builder.buildDockerArgs("Dockerfile", false, nil)
		require.NoError(t, err)
		require.Equal(t, []string{
			"buildx", "build", "--pull", "--file", "Dockerfile", "--platform", "amd64",
			"--cache-from", "type=gha,scope=omes-ci-typescript-linux-amd64,timeout=5m,version=2",
			"--cache-to", "type=gha,scope=omes-ci-typescript-linux-amd64,mode=max,ignore-error=true,timeout=5m,version=2",
			"--load",
		}, args)
	})

	t.Run("enabled for multi-platform push", func(t *testing.T) {
		t.Setenv(buildKitCacheScopeEnv, "omes-publish-typescript-linux-amd64-linux-arm64")
		builder := baseImageBuilder{platforms: []string{"linux/amd64", "linux/arm64"}}

		args, err := builder.buildDockerArgs("Dockerfile", true, nil)
		require.NoError(t, err)
		require.Equal(t, []string{
			"buildx", "build", "--pull", "--file", "Dockerfile", "--platform", "linux/amd64,linux/arm64",
			"--cache-from", "type=gha,scope=omes-publish-typescript-linux-amd64-linux-arm64,timeout=5m,version=2",
			"--cache-to", "type=gha,scope=omes-publish-typescript-linux-amd64-linux-arm64,mode=max,ignore-error=true,timeout=5m,version=2",
			"--push",
		}, args)
	})
}
