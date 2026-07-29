package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
)

func listScenariosCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-scenarios",
		Short: "List Scenarios",
		Run: func(cmd *cobra.Command, args []string) {
			var descs []string
			for name, scen := range loadgen.GetScenarios() {
				descs = append(descs, describeScenario(name, scen))
			}
			sort.Strings(descs)
			for _, desc := range descs {
				fmt.Println(desc)
			}
		},
	}
}

func describeScenario(name string, scen *loadgen.Scenario) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Scenario: %v\n    Description: %v\n", name, scen.Description)

	if config, ok := scen.GetDefaultConfiguration(); ok {
		config.ApplyDefaults()
		b.WriteString("    Default configuration:\n")
		if config.Iterations != 0 {
			fmt.Fprintf(&b, "        Iterations: %v\n", config.Iterations)
		}
		if config.Duration != 0 {
			fmt.Fprintf(&b, "        Duration: %v\n", config.Duration)
		}
		if config.MaxConcurrent != 0 {
			fmt.Fprintf(&b, "        Max concurrent: %v\n", config.MaxConcurrent)
		}
		if config.Timeout != 0 {
			fmt.Fprintf(&b, "        Timeout: %v\n", config.Timeout)
		}
	}

	if opts := scen.DeclaredOptions(); len(opts) > 0 {
		b.WriteString("    Options (set with --option <name>=<value>):\n")
		for _, o := range opts {
			fmt.Fprintf(&b, "        %s (%s)", o.Name, o.Type)
			switch {
			case o.Required:
				b.WriteString(" [required]")
			case o.Default != "":
				fmt.Fprintf(&b, " [default: %s]", o.Default)
			}
			if o.Usage != "" {
				fmt.Fprintf(&b, "\n            %s", o.Usage)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
