package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"golang.org/x/mod/semver"
)

func buildWorkerImageCmd() *cobra.Command {
	var b workerImageBuilder
	cmd := &cobra.Command{
		Use:   "build-worker-image",
		Short: "Build a worker image (local only, no push)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context(), false); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("version")
	return cmd
}

func buildPushWorkerImageCmd() *cobra.Command {
	var b workerImageBuilder
	cmd := &cobra.Command{
		Use:   "build-push-worker-image",
		Short: "Build and push a worker image",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context(), true); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("version")
	return cmd
}

type workerImageBuilder struct {
	baseImageBuilder
	sdkOptions cmdoptions.SdkOptions
}

func (b *workerImageBuilder) addCLIFlags(fs *pflag.FlagSet) {
	b.sdkOptions.AddCLIFlags(fs)
	b.addBaseCLIFlags(fs)
}

func (b *workerImageBuilder) build(ctx context.Context, allowPush bool) error {
	b.logger = b.loggingOptions.MustCreateLogger()
	lang := b.sdkOptions.Language.String()
	version := b.sdkOptions.Version

	// At some point we probably want to replace this with a meaningful version of omes itself
	omesVersion, err := getCurrentCommitSha(ctx)
	if err != nil {
		return err
	}

	// Setup args, tags, and labels based on version
	isPathVersion := strings.ContainsAny(version, `/\`)
	var buildArgs []string
	if isPathVersion {
		if len(b.tags) == 0 {
			return fmt.Errorf("at least one tag required for path version")
		} else if s, err := os.Stat(version); err != nil {
			return fmt.Errorf("invalid path version: %w", err)
		} else if !s.IsDir() {
			return fmt.Errorf("invalid path version: must be dir")
		} else if !filepath.IsLocal(version) {
			return fmt.Errorf("invalid path version: must be beneath this dir")
		}

		// Dockerfile copies entire version path to ./repo
		buildArgs = append(buildArgs, "SDK_VERSION=./repo", "SDK_DIR="+version)
	} else {
		buildArgs = append(buildArgs, "SDK_VERSION="+version)

		// Add label for version
		b.addLabelIfNotPresent("io.temporal.sdk.version", version)

		// Check for valid version
		versionToCheck := version
		if !strings.HasPrefix(version, "v") {
			versionToCheck = "v" + versionToCheck
		}
		if semver.Canonical(versionToCheck) == "" {
			return fmt.Errorf("expected valid semver")
		}

		// Add tag for lang-fullsemver without leading "v". We are intentionally
		// including semver build metadata.
		langTagComponent := lang + "-" + strings.TrimPrefix(version, "v")
		b.tags = append(b.tags, omesVersion+"-"+langTagComponent)
		b.tags = append(b.tags, lang+"-"+omesVersion)
		if b.tagAsLatest {
			b.tags = append(b.tags, langTagComponent)
			b.tags = append(b.tags, lang+"-latest")
		}
	}
	imageTagsForPublish := b.generateImageTags()

	// Set OCI labels
	err = b.addDefaultLabels(ctx, omesVersion, "Load testing for "+lang)
	if err != nil {
		return err
	}
	b.addLabelIfNotPresent("io.temporal.sdk.name", lang)
	b.addLabelIfNotPresent("io.temporal.sdk.version", version)

	// Build docker command args
	args, err := b.buildDockerArgs("dockerfiles/"+lang+".Dockerfile", allowPush, buildArgs)
	if err != nil {
		if !allowPush {
			return fmt.Errorf("multi-platform builds require pushing to registry. Use build-push-worker-image command instead")
		}
		return err
	}

	err = b.executeDockerBuild(ctx, args, imageTagsForPublish)
	if err != nil {
		return err
	}

	return b.handleImageSave(ctx, imageTagsForPublish)
}
