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
	}

	rootCmd.AddCommand(cleanupScenarioCmd())
	rootCmd.AddCommand(execCmd())
	rootCmd.AddCommand(listScenariosCmd())
	rootCmd.AddCommand(prepareWorkerCmd())
	rootCmd.AddCommand(runScenarioCmd())
	rootCmd.AddCommand(runScenarioWithWorkerCmd())
	rootCmd.AddCommand(runWorkerCmd())
	rootCmd.AddCommand(workflowCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
