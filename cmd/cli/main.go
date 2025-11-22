package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "github.com/temporalio/omes/scenarios" // Register scenarios (side-effect)
)

func Main() {
	var rootCmd = &cobra.Command{
		Use:   "omes",
		Short: "A load generator for Temporal",
		Run: func(cmd *cobra.Command, args []string) {
			// Default behavior when no subcommand is specified
			// Just wait if --wait flag is set, otherwise show help
			if !waitAfterRun {
				cmd.Help()
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			waitAfterRunIfRequested(cmd)
		},
	}

	waitCmd(rootCmd)
	rootCmd.AddCommand(cleanupScenarioCmd())
	rootCmd.AddCommand(listScenariosCmd())
	rootCmd.AddCommand(prepareWorkerCmd())
	rootCmd.AddCommand(runScenarioCmd())
	rootCmd.AddCommand(runScenarioWithWorkerCmd())
	rootCmd.AddCommand(runWorkerCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
