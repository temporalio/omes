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

	// Naming the flag that sets each value is what tells a reader this knob is a
	// run flag rather than an --option. Attribute the values too: most scenarios
	// set none of their own, and omes's defaults shown unattributed read as a
	// scenario's intent.
	config, declared := scen.GetDefaultConfiguration()
	config.ApplyDefaults()
	source := "omes defaults, as this scenario sets none"
	if declared {
		source = "this scenario's defaults"
	}
	fmt.Fprintf(&b, "    Run configuration (%s) — applies to every scenario, override with these flags:\n", source)
	// A zero here carries no information — an unset timeout just means unlimited —
	// so only settings with a value are worth printing.
	flag := func(name string, value any, isSet bool) {
		if isSet {
			fmt.Fprintf(&b, "        %s %v\n", name, value)
		}
	}
	flag("--iterations", config.Iterations, config.Iterations != 0)
	flag("--duration", config.Duration, config.Duration != 0)
	flag("--max-concurrent", config.MaxConcurrent, config.MaxConcurrent != 0)
	flag("--max-iteration-attempts", config.MaxIterationAttempts, config.MaxIterationAttempts != 0)
	flag("--max-iterations-per-second", config.MaxIterationsPerSecond, config.MaxIterationsPerSecond != 0)
	flag("--timeout", config.Timeout, config.Timeout != 0)
	b.WriteString("        (run 'omes run-scenario --help' for the full set)\n")

	opts := scen.DeclaredOptions()
	if len(opts) == 0 {
		fmt.Fprintf(&b, "    Scenario options: none — %s accepts no --option values.\n", name)
		return b.String()
	}
	fmt.Fprintf(&b, "    Scenario options — specific to %s, set with --option <name>=<value>:\n", name)
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

	return b.String()
}
