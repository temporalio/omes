package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/clioptions"
	"go.uber.org/zap"
)

type baseImageBuilder struct {
	logger         *zap.SugaredLogger
	tagAsLatest    bool
	platforms      []string
	imageName      string
	repoPrefix     string
	dryRun         bool
	saveImage      string
	tags           []string
	labels         []string
	cacheFrom      []string
	cacheTo        []string
	pushByDigest   bool
	metadataFile   string
	loggingOptions clioptions.LoggingOptions
}

func (b *baseImageBuilder) addBaseCLIFlags(fs *pflag.FlagSet) {
	fs.AddFlagSet(b.loggingOptions.FlagSet())
	fs.BoolVar(&b.tagAsLatest, "tag-as-latest", false,
		"If set, tag the image as latest in addition to the omes commit sha tag")
	fs.StringSliceVar(&b.platforms, "platform", []string{"linux/amd64"}, "Platforms for use in docker build --platform")
	fs.StringVar(&b.imageName, "image-name", "omes", "Name of the image to build")
	fs.StringVar(&b.repoPrefix, "repo-prefix", "", "Repository prefix (e.g., 'temporaliotest'). If empty, no prefix is used.")
	fs.BoolVar(&b.dryRun, "dry-run", false, "If set, just print the commands that would run but do not run them")
	fs.StringSliceVar(&b.tags, "image-tag", nil, "Additional tags to add to the image")
	fs.StringSliceVar(&b.labels, "image-label", nil, "Additional labels to add to the image")
	fs.StringArrayVar(&b.cacheFrom, "cache-from", nil, "Buildx cache source (repeatable)")
	fs.StringArrayVar(&b.cacheTo, "cache-to", nil, "Buildx cache destination (repeatable)")
	fs.BoolVar(&b.pushByDigest, "push-by-digest", false, "Push a single-platform image by digest without publishing tags")
	fs.StringVar(&b.metadataFile, "metadata-file", "", "Write Buildx build metadata to this file")
	fs.StringVar(&b.saveImage, "save-image", "", "If set, will run `docker save` on the produced image, saving it to the provided path")
}

func (b *baseImageBuilder) addLabelIfNotPresent(key, value string) {
	for _, label := range b.labels {
		if strings.HasPrefix(label, key+"=") {
			return
		}
	}
	b.labels = append(b.labels, key+"="+value)
}

func (b *baseImageBuilder) addDefaultLabels(ctx context.Context, omesVersion, title string) error {
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
	b.addLabelIfNotPresent("org.opencontainers.image.title", title)
	b.addLabelIfNotPresent("org.opencontainers.image.documentation", "See README at https://github.com/temporalio/omes")
	b.addLabelIfNotPresent("io.temporal.omes.githash", omesVersion)
	return nil
}

func (b *baseImageBuilder) generateImageTags() []string {
	var imageTagsForPublish []string
	for _, tag := range b.tags {
		tagVal := fmt.Sprintf("%s:%s", b.imageName, tag)
		if b.repoPrefix != "" {
			tagVal = fmt.Sprintf("%s/%s", b.repoPrefix, tagVal)
		}
		imageTagsForPublish = append(imageTagsForPublish, tagVal)
	}
	return imageTagsForPublish
}

func (b *baseImageBuilder) buildDockerArgs(dockerFile string, allowPush bool, buildArgs []string) (dockerArgs []string, err error) {
	dockerArgs = []string{
		"buildx",
		"build",
		"--pull",
		"--file", dockerFile,
		"--platform", strings.Join(qualifyPlatforms(b.platforms), ","),
	}

	// Digest-only exports leave tags for a separate manifest publishing step.
	if b.pushByDigest {
		if !allowPush {
			return nil, fmt.Errorf("push-by-digest requires pushing to registry")
		}
		if len(b.platforms) != 1 {
			return nil, fmt.Errorf("push-by-digest requires exactly one platform")
		}
		if b.saveImage != "" {
			return nil, fmt.Errorf("push-by-digest cannot be combined with save-image")
		}
		repository := b.imageName
		if b.repoPrefix != "" {
			repository = b.repoPrefix + "/" + repository
		}
		dockerArgs = append(dockerArgs, "--output", "type=image,name="+repository+",push-by-digest=true,name-canonical=true,push=true")
	} else if len(b.platforms) > 1 {
		if !allowPush {
			err = fmt.Errorf("multi-platform builds require pushing to registry")
			return
		}
		dockerArgs = append(dockerArgs, "--push")
	} else {
		dockerArgs = append(dockerArgs, "--load")
	}

	for _, cache := range b.cacheFrom {
		dockerArgs = append(dockerArgs, "--cache-from", cache)
	}
	for _, cache := range b.cacheTo {
		dockerArgs = append(dockerArgs, "--cache-to", cache)
	}
	if b.metadataFile != "" {
		dockerArgs = append(dockerArgs, "--metadata-file", b.metadataFile)
	}

	// Final tags are still computed and exported, but digest-only builds must not publish them.
	if !b.pushByDigest {
		for _, tagVal := range b.generateImageTags() {
			dockerArgs = append(dockerArgs, "--tag", tagVal)
		}
	}

	// Add labels
	for _, label := range b.labels {
		dockerArgs = append(dockerArgs, "--label", label)
	}

	// Add build arguments
	for _, arg := range buildArgs {
		dockerArgs = append(dockerArgs, "--build-arg", arg)
	}

	return
}

func (b *baseImageBuilder) executeDockerBuild(ctx context.Context, args []string, imageTagsForPublish []string) error {
	repoDir, err := getRepoDir()
	if err != nil {
		return fmt.Errorf("failed to get root directory: %w", err)
	}
	args = append(args, repoDir)
	b.logger.Infof("Running: docker %v", strings.Join(args, " "))
	if b.dryRun {
		return nil
	}

	err = writeGitHubEnv("BUILT_IMAGE_TAGS", strings.Join(imageTagsForPublish, ";"))
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

	return nil
}

func (b *baseImageBuilder) handleImageSave(ctx context.Context, imageTagsForPublish []string) error {
	if b.saveImage != "" {
		err := writeGitHubEnv("SAVED_IMAGE_TAG", imageTagsForPublish[0])
		if err != nil {
			return fmt.Errorf("writing image tags to github env failed: %s", err)
		}
		if len(b.platforms) > 1 {
			b.logger.Warn("Image saving is not supported for multi-platform builds. Skipping save step.")
		} else {
			saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", b.saveImage, imageTagsForPublish[0])
			saveCmd.Stdout = os.Stdout
			saveCmd.Stderr = os.Stderr
			err = saveCmd.Run()
			if err != nil {
				return fmt.Errorf("failed saving image: %w", err)
			}
		}
	}
	return nil
}

// qualifyPlatforms prefixes bare architectures with the linux OS. Docker resolves the OS of an unqualified platform
// from the host, so "amd64" means linux/amd64 on a CI runner but darwin/amd64 on a Mac, where the build itself
// succeeds and only the final export fails with "operating system is not supported".
func qualifyPlatforms(platforms []string) []string {
	qualified := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		if !strings.Contains(platform, "/") {
			platform = "linux/" + platform
		}
		qualified = append(qualified, platform)
	}
	return qualified
}

func gitRef(ctx context.Context, gitDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "--git-dir", gitDir, "rev-parse", "HEAD")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed getting git ref: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
