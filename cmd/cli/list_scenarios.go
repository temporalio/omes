package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
)

func listScenariosCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-scenarios",
		Short: "List Scenarios",
		Run: func(cmd *cobra.Command, args []string) {
			// Collect scenario descriptions, sort, display
			var descs []string
			for name, scen := range loadgen.GetScenarios() {
				var defaultConfigDesc string
				executor := scen.ExecutorFn()
				if iface, _ := executor.(loadgen.HasDefaultConfiguration); iface != nil {
					config := iface.GetDefaultConfiguration()
					config.ApplyDefaults()
					defaultConfigDesc = "\n    Default configuration:"
					if config.Iterations != 0 {
						defaultConfigDesc += fmt.Sprintf("\n        Iterations: %v", config.Iterations)
					}
					if config.Duration != 0 {
						defaultConfigDesc += fmt.Sprintf("\n        Duration: %v", config.Duration)
					}
					if config.MaxConcurrent != 0 {
						defaultConfigDesc += fmt.Sprintf("\n        Max concurrent: %v", config.MaxConcurrent)
					}
					if config.Timeout != 0 {
						defaultConfigDesc += fmt.Sprintf("\n        Timeout: %v", config.Timeout)
					}
				}
				descs = append(descs, fmt.Sprintf("Scenario: %v\n    Description: %v%v\n",
					name, scen.Description, defaultConfigDesc))
			}
			sort.Strings(descs)
			for _, desc := range descs {
				fmt.Println(desc)
			}
		},
	}
}
