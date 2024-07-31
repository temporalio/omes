package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

func buildWorkerImageCmd() *cobra.Command {
	var b workerImageBuilder
	cmd := &cobra.Command{
		Use:   "build-worker-image",
		Short: "Build an image",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context()); err != nil {
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
	logger         *zap.SugaredLogger
	language       string
	version        string
	tagAsLatest    bool
	platform       string
	imageName      string
	dryRun         bool
	saveImage      string
	tags           []string
	labels         []string
	loggingOptions cmdoptions.LoggingOptions
}

func (b *workerImageBuilder) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&b.language, "language", "", "Language to build a worker image for")
	fs.StringVar(&b.version, "version", "",
		"SDK version to build a worker image for - treated as path if slash present, but must be beneath this dir and at least one tag required")
	fs.BoolVar(&b.tagAsLatest, "tag-as-latest", false,
		"If set, tag the image as latest in addition to the omes commit sha tag")
	fs.StringVar(&b.platform, "platform", "", "Platform for use in docker build --platform")
	fs.StringVar(&b.imageName, "image-name", "omes", "Name of the image to build")
	fs.BoolVar(&b.dryRun, "dry-run", false, "If set, just print the commands that would run but do not run them")
	fs.StringSliceVar(&b.tags, "image-tag", nil, "Additional tags to add to the image")
	fs.StringSliceVar(&b.labels, "image-label", nil, "Additional labels to add to the image")
	fs.StringVar(&b.saveImage, "save-image", "", "If set, will run `docker save` on the produced image, saving it to the provided path")
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *workerImageBuilder) build(ctx context.Context) error {
	b.logger = b.loggingOptions.MustCreateLogger()
	lang, err := normalizeLangName(b.language)
	if err != nil {
		return err
	}
	// At some point we probably want to replace this with a meaningful version of omes itself
	omesVersion, err := getCurrentCommitSha(ctx)
	if err != nil {
		return err
	}

	// Setup args, tags, and labels based on version
	isPathVersion := strings.ContainsAny(b.version, `/\`)
	var buildArgs []string
	if isPathVersion {
		if len(b.tags) == 0 {
			return fmt.Errorf("at least one tag required for path version")
		} else if s, err := os.Stat(b.version); err != nil {
			return fmt.Errorf("invalid path version: %w", err)
		} else if !s.IsDir() {
			return fmt.Errorf("invalid path version: must be dir")
		} else if !filepath.IsLocal(b.version) {
			return fmt.Errorf("invalid path version: must be beneath this dir")
		}
		// Dockerfile copies entire version path to ./repo
		buildArgs = append(buildArgs, "SDK_VERSION=./repo", "SDK_DIR="+b.version)
	} else {
		buildArgs = append(buildArgs, "SDK_VERSION="+b.version)
		// Add label for version
		b.addLabelIfNotPresent("io.temporal.sdk.version", b.version)
		// Check for valid version
		versionToCheck := b.version
		if !strings.HasPrefix(b.version, "v") {
			versionToCheck = "v" + versionToCheck
		}
		if semver.Canonical(versionToCheck) == "" {
			return fmt.Errorf("expected valid semver")
		}
		// Add tag for lang-fullsemver without leading "v". We are intentionally
		// including semver build metadata.
		langTagComponent := lang + "-" + strings.TrimPrefix(b.version, "v")
		b.tags = append(b.tags, omesVersion+"-"+langTagComponent)
		if b.tagAsLatest {
			b.tags = append(b.tags, langTagComponent)
			b.tags = append(b.tags, lang+"-latest")
		}
	}

	// Setup additional labels
	gitRef, err := gitRef(ctx, ".git")
	if err != nil {
		return err
	}
	b.addLabelIfNotPresent("org.opencontainers.image.created", time.Now().UTC().Format(time.RFC3339))
	b.addLabelIfNotPresent("org.opencontainers.image.source", "https://github.com/temporalio/omes")
	b.addLabelIfNotPresent("org.opencontainers.image.vendor", "Temporal Technologies Inc.")
	b.addLabelIfNotPresent("org.opencontainers.image.authors", "Temporal SDK team <sdk-team@temporal.io>")
	b.addLabelIfNotPresent("org.opencontainers.image.licenses", "MIT")
	b.addLabelIfNotPresent("org.opencontainers.image.revision", gitRef)
	b.addLabelIfNotPresent("org.opencontainers.image.title", "Load testing for "+lang)
	b.addLabelIfNotPresent("org.opencontainers.image.documentation", "See README at https://github.com/temporalio/omes")
	b.addLabelIfNotPresent("io.temporal.sdk.name", lang)
	b.addLabelIfNotPresent("io.temporal.sdk.version", b.version)
	b.addLabelIfNotPresent("io.temporal.omes.githash", omesVersion)

	// Prepare docker command args
	args := []string{
		"build",
		"--pull",
		"--file", "dockerfiles/" + lang + ".Dockerfile",
	}
	if b.platform != "" {
		args = append(args, "--platform", b.platform, "--build-arg", "PLATFORM="+b.platform)
	}
	var imageTagsForPublish []string
	for _, tag := range b.tags {
		tagVal := fmt.Sprintf("%s:%s", b.imageName, tag)
		args = append(args, "--tag", tagVal)
		imageTagsForPublish = append(imageTagsForPublish, tagVal)
	}
	for _, label := range b.labels {
		args = append(args, "--label", label)
	}
	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}
	args = append(args, rootDir())
	b.logger.Infof("Running: docker %v", strings.Join(args, " "))
	if b.dryRun {
		return nil
	}

	// Write all the produced image tags to an env var so that the GH workflow can later use it
	// to publish them.
	err = writeGitHubEnv("FEATURES_BUILT_IMAGE_TAGS", strings.Join(imageTagsForPublish, ";"))
	if err != nil {
		return fmt.Errorf("writing image tags to github env failed: %s", err)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed building image: %w", err)
	}

	if b.saveImage != "" {
		saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", b.saveImage, imageTagsForPublish[0])
		saveCmd.Stdout = os.Stdout
		saveCmd.Stderr = os.Stderr
		err = saveCmd.Run()
		if err != nil {
			return fmt.Errorf("failed saving image: %w", err)
		}
	}

	return nil
}

func (b *workerImageBuilder) addLabelIfNotPresent(key, value string) {
	for _, label := range b.labels {
		if strings.HasPrefix(label, key+"=") {
			return
		}
	}
	b.labels = append(b.labels, key+"="+value)
}

func gitRef(ctx context.Context, gitDir string) (string, error) {
	cmd := exec.Command("git", "--git-dir", gitDir, "rev-parse", "HEAD")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed getting git ref: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
