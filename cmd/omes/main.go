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

	// Execute has already reported the error; printing it again here put a second
	// copy below the usage text, where it read as a second, separate failure.
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
