package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
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
	return cmd
}

type workerImageBuilder struct {
	baseImageBuilder
	sdkOptions clioptions.SdkOptions
	appName    string
}

func (b *workerImageBuilder) addCLIFlags(fs *pflag.FlagSet) {
	b.sdkOptions.AddCLIFlags(fs)
	fs.StringVar(&b.appName, "app", "", "Go app entrypoint to build; empty builds the default Go app")
	b.addBaseCLIFlags(fs)
}

func (b *workerImageBuilder) build(ctx context.Context, allowPush bool) error {
	b.logger = b.loggingOptions.MustCreateLogger()
	lang := b.sdkOptions.Language.String()
	sdkVersion := b.sdkOptions.Version

	// If no version provided, load from versions.env
	if sdkVersion == "" {
		if loadedVersion, err := getVersion(lang + "_sdk"); err == nil {
			sdkVersion = loadedVersion
		} else {
			return fmt.Errorf("no version specified and failed to load from versions.env for %s: %w", lang, err)
		}
	}

	// At some point we probably want to replace this with a meaningful version of omes itself
	omesVersion, err := getCurrentCommitSha(ctx)
	if err != nil {
		return err
	}

	// Setup args, tags, and labels based on version
	isPathVersion := strings.ContainsAny(sdkVersion, `/\`)
	imageTarget := workerImageTargetFor(lang, b.appName)
	buildArgs := append([]string{}, imageTarget.buildArgs...)
	if isPathVersion {
		if len(b.tags) == 0 {
			return fmt.Errorf("at least one tag required for path version")
		} else if s, err := os.Stat(sdkVersion); err != nil {
			return fmt.Errorf("invalid path version: %w", err)
		} else if !s.IsDir() {
			return fmt.Errorf("invalid path version: must be dir")
		} else if !filepath.IsLocal(sdkVersion) {
			return fmt.Errorf("invalid path version: must be beneath this dir")
		}

		// Dockerfile copies entire version path to ./repo
		buildArgs = append(buildArgs, "SDK_VERSION=./repo", "SDK_DIR="+sdkVersion)
	} else {
		buildArgs = append(buildArgs, "SDK_VERSION="+sdkVersion)

		// Add label for version
		b.addLabelIfNotPresent("io.temporal.sdk.version", sdkVersion)

		// Check for valid version
		versionToCheck := sdkVersion
		if !strings.HasPrefix(sdkVersion, "v") {
			versionToCheck = "v" + versionToCheck
		}
		if semver.Canonical(versionToCheck) == "" {
			return fmt.Errorf("expected valid semver")
		}

		// Add tag for lang-fullsemver without leading "v". We are intentionally
		// including semver build metadata.
		langTagComponent := imageTarget.tagLanguage + "-" + strings.TrimPrefix(sdkVersion, "v")
		b.tags = append(b.tags, omesVersion+"-"+langTagComponent)
		b.tags = append(b.tags, imageTarget.tagLanguage+"-"+omesVersion)
		if b.tagAsLatest {
			b.tags = append(b.tags, langTagComponent)
			b.tags = append(b.tags, imageTarget.tagLanguage+"-latest")
		}
	}
	imageTagsForPublish := b.generateImageTags()

	// Set OCI labels
	err = b.addDefaultLabels(ctx, omesVersion, "Load testing for "+imageTarget.tagLanguage)
	if err != nil {
		return err
	}
	b.addLabelIfNotPresent("io.temporal.sdk.name", lang)
	b.addLabelIfNotPresent("io.temporal.sdk.version", sdkVersion)
	if imageTarget.workerAppLabel != "" {
		b.addLabelIfNotPresent("io.temporal.omes.worker_app", imageTarget.workerAppLabel)
	}

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

type workerImageTarget struct {
	buildArgs      []string
	tagLanguage    string
	workerAppLabel string
}

func workerImageTargetFor(lang, appName string) workerImageTarget {
	target := workerImageTarget{tagLanguage: lang}

	switch lang {
	case string(clioptions.LangGo):
		switch appName {
		case "", "worker":
			return target
		default:
			target.buildArgs = append(target.buildArgs, "GO_APP="+appName)
			target.tagLanguage = lang + "-" + appName
			target.workerAppLabel = appName
		}
	}

	return target
}
