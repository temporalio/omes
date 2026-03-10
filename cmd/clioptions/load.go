package clioptions

import (
	"time"

	"github.com/spf13/cobra"
)

type LoadOptions struct {
	Iterations             int
	Duration               time.Duration
	MaxConcurrent          int
	MaxIterationsPerSecond float64
	Timeout                time.Duration
}

func (r *LoadOptions) AddCLIFlags(cmd *cobra.Command) {
	fs := cmd.Flags()
	fs.IntVar(&r.Iterations, "iterations", 0, "Number of iterations to run (cannot be provided with --duration)")
	fs.DurationVar(&r.Duration, "duration", 0, "Duration to run (cannot be provided with --iterations)")
	fs.Float64Var(&r.MaxIterationsPerSecond, "max-iterations-per-second", 0, "Maximum rate at which to start new iterations")
	fs.DurationVar(&r.Timeout, "timeout", 0, "If set, stop after this time and exit nonzero")
	fs.IntVar(&r.MaxConcurrent, "max-concurrent", 10, "Maximum concurrent iterations")
	cmd.MarkFlagsMutuallyExclusive("iterations", "duration")
	cmd.MarkFlagsOneRequired("iterations", "duration")
}
