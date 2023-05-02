package main

import (
	"github.com/spf13/cobra"
	_ "github.com/temporalio/omes/scenarios" // Register scenarios (side-effect)
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "omes",
		Short: "A load generator for Temporal",
	}

	rootCmd.AddCommand(cleanupScenarioCmd())
	rootCmd.AddCommand(prepareWorkerCmd())
	rootCmd.AddCommand(runScenarioCmd())
	rootCmd.AddCommand(runScenarioWithWorkerCmd())
	rootCmd.AddCommand(runWorkerCmd())

	rootCmd.Execute()
}
