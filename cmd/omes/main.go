package main

import (
	"os"

	"github.com/spf13/cobra"
	_ "github.com/temporalio/omes/scenarios"         // Register scenarios (side-effect)
	_ "github.com/temporalio/omes/scenarios/project" // Register project scenario
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "omes",
		Short: "A load generator for Temporal",
	}

	rootCmd.AddCommand(cleanupScenarioCmd())
	rootCmd.AddCommand(listScenariosCmd())
	rootCmd.AddCommand(prepareWorkerCmd())
	rootCmd.AddCommand(runScenarioCmd())
	rootCmd.AddCommand(runScenarioWithWorkerCmd())
	rootCmd.AddCommand(runWorkerCmd())

	// Execute prints the error itself. Printing it again here would land a second
	// copy below the usage text, reading as a second, separate failure.
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
