package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func buildCliImageCmd() *cobra.Command {
	var b cliImageBuilder
	cmd := &cobra.Command{
		Use:   "build-cli-image",
		Short: "Build a CLI image (local only, no push)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context(), false); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	return cmd
}

func buildPushCliImageCmd() *cobra.Command {
	var b cliImageBuilder
	cmd := &cobra.Command{
		Use:   "build-push-cli-image",
		Short: "Build and push a CLI image",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context(), true); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	return cmd
}

type cliImageBuilder struct {
	baseImageBuilder
}

func (b *cliImageBuilder) addCLIFlags(fs *pflag.FlagSet) {
	b.addBaseCLIFlags(fs)
}

func (b *cliImageBuilder) build(ctx context.Context, allowPush bool) error {
	b.logger = b.loggingOptions.MustCreateLogger()

	// At some point we probably want to replace this with a meaningful version of omes itself
	omesVersion, err := getCurrentCommitSha(ctx)
	if err != nil {
		return err
	}

	// Set tags
	b.tags = append(b.tags, omesVersion+"-cli")
	b.tags = append(b.tags, "cli-"+omesVersion)
	if b.tagAsLatest {
		b.tags = append(b.tags, "cli")
		b.tags = append(b.tags, "cli-latest")
	}
	imageTagsForPublish := b.generateImageTags()

	// Set OCI labels
	err = b.addDefaultLabels(ctx, omesVersion, "Load testing CLI")
	if err != nil {
		return err
	}

	// Build docker command args
	args, err := b.buildDockerArgs("dockerfiles/cli.Dockerfile", allowPush, []string{}, "")
	if err != nil {
		if !allowPush {
			return fmt.Errorf("multi-platform builds require pushing to registry. Use build-push-cli-image command instead")
		}
		return err
	}

	err = b.executeDockerBuild(ctx, args, imageTagsForPublish)
	if err != nil {
		return err
	}

	return b.handleImageSave(ctx, imageTagsForPublish)
}
