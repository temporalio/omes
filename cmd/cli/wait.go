package cli

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var waitAfterRun bool

// waitAfterRunIfRequested is called by the root command's PersistentPostRun.
// It waits indefinitely if the --wait flag is set.
func waitAfterRunIfRequested(_ *cobra.Command) {
	if !waitAfterRun {
		return
	}

	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	sugar := logger.Sugar()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	sugar.Info("Waiting indefinitely")
	<-ctx.Done()
	sugar.Info("Shutting Down")
}

func waitCmd(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().BoolVar(&waitAfterRun, "wait", false, "Wait indefinitely after command completes")
}
