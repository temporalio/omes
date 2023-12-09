package main

import (
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/scenariorunner"
)

func runScenarioCmd() *cobra.Command {
	var r scenariorunner.ScenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario",
		Short: "Run scenario",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.Run(ctx); err != nil {
				r.Logger.Fatal(err)
			}
		},
	}
	r.AddCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("scenario")
	cmd.MarkFlagRequired("run-id")
	return cmd
}
