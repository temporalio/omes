package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
)

func foldLoadArgs(args []string) []string {
	if len(args) == 0 || (args[0] != "run-scenario" && args[0] != "run-scenario-with-worker") {
		return args
	}
	folded := append([]string(nil), args...)
	for i := 1; i < len(folded); i++ {
		if folded[i] == "--" {
			break
		}
		if folded[i] == "--load" {
			if i+1 == len(folded) || strings.HasPrefix(folded[i+1], "-") {
				continue
			}
			i++
		} else if !strings.HasPrefix(folded[i], "--load=") {
			continue
		}
		end := i + 1
		for end < len(folded) && isLoadParameter(folded[end]) {
			end++
		}
		folded[i] = strings.Join(folded[i:end], ",")
		folded = append(folded[:i+1], folded[end:]...)
	}
	return folded
}

func isLoadParameter(arg string) bool {
	key, _, ok := strings.Cut(arg, "=")
	return ok && key != "" && !strings.HasPrefix(arg, "-")
}

func validateLoadArgs(_ *cobra.Command, args []string) error {
	for _, arg := range args {
		if isLoadParameter(arg) {
			return loadgen.NewUsageError("parameters must directly follow --load <pattern>; e.g. --load spike-train peak=120/s every=5m")
		}
	}
	return nil
}
