package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "github.com/temporalio/omes/scenarios" // Register scenarios (side-effect)
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "omes",
		Short: "A load generator for Temporal",
	}

	rootCmd.AddCommand(buildWorkerImageCmd())
	rootCmd.AddCommand(buildPushWorkerImageCmd())
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
