// TODO: Should be de-duped with similar code in features repo
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"go.uber.org/zap"
)

func publishImageCmd() *cobra.Command {
	var ip imagePublisher
	cmd := &cobra.Command{
		Use:   "push-images",
		Short: "Push docker image(s) to our test repository. Used by CI.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ip.publishImages(); err != nil {
				ip.logger.Fatal(err)
			}
		},
	}
	ip.flags(cmd.Flags())
	return cmd
}

func (ip *imagePublisher) flags(fs *pflag.FlagSet) {
	fs.StringVar(&ip.repoPrefix, "repo-prefix", "temporaliotest", "Prefix for the docker image repository")
	ip.loggingOptions.AddCLIFlags(fs)
}

// imagePublisher stores config for the publish-image command.
type imagePublisher struct {
	logger *zap.SugaredLogger

	repoPrefix     string
	loggingOptions cmdoptions.LoggingOptions
}

func (ip *imagePublisher) publishImages() error {
	ip.logger = ip.loggingOptions.MustCreateLogger()

	tagsFromEnv := strings.Split(os.Getenv("FEATURES_BUILT_IMAGE_TAGS"), ";")
	if len(tagsFromEnv) == 0 || tagsFromEnv[0] == "" {
		return fmt.Errorf("no image tags found in FEATURES_BUILT_IMAGE_TAGS")
	}

	var pushedTags []string
	for _, tag := range tagsFromEnv {
		pushAs := fmt.Sprintf("%s/%s", ip.repoPrefix, tag)
		dockerTag := exec.Command("docker", "tag", tag, pushAs)
		dockerTag.Stdout = os.Stdout
		dockerTag.Stderr = os.Stderr
		err := dockerTag.Run()
		if err != nil {
			return fmt.Errorf("failed to tag docker image: %w", err)
		}
		pushedTags = append(pushedTags, pushAs)
	}

	for _, tag := range pushedTags {
		dockerPush := exec.Command("docker", "push", tag)
		dockerPush.Stdout = os.Stdout
		dockerPush.Stderr = os.Stderr
		err := dockerPush.Run()
		if err != nil {
			return fmt.Errorf("failed to push docker image: %w", err)
		}
	}

	return nil
}
