package main

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestFoldLoadArgs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args, want []string
	}{
		{"parameters", []string{"run-scenario", "--load", "spike-train", "peak=120/s", "every=5m"}, []string{"run-scenario", "--load", "spike-train,peak=120/s,every=5m"}},
		{"equals", []string{"run-scenario-with-worker", "--load=steady", "peak=120/s"}, []string{"run-scenario-with-worker", "--load=steady,peak=120/s"}},
		{"comma", []string{"run-scenario", "--load", "steady,peak=120/s"}, []string{"run-scenario", "--load", "steady,peak=120/s"}},
		{"quoted", []string{"run-scenario", "--load", "steady peak=120/s"}, []string{"run-scenario", "--load", "steady peak=120/s"}},

		{"flag", []string{"run-scenario", "--load", "steady", "peak=120/s", "--duration=1m", "base=0.4"}, []string{"run-scenario", "--load", "steady,peak=120/s", "--duration=1m", "base=0.4"}},
		{"separator", []string{"run-scenario", "--load=steady", "peak=120/s", "--", "base=0.4", "--load=steady", "peak=2"}, []string{"run-scenario", "--load=steady,peak=120/s", "--", "base=0.4", "--load=steady", "peak=2"}},
		{"word", []string{"run-scenario", "--load", "steady", "word", "peak=120/s"}, []string{"run-scenario", "--load", "steady", "word", "peak=120/s"}},
		{"separator before load", []string{"run-scenario", "--", "--load=steady", "peak=2"}, []string{"run-scenario", "--", "--load=steady", "peak=2"}},
		{"separator as value", []string{"run-scenario", "--load", "--", "peak=2"}, []string{"run-scenario", "--load", "--", "peak=2"}},
		{"missing value", []string{"run-scenario", "--load"}, []string{"run-scenario", "--load"}},

		{"other command", []string{"run-worker", "--load=steady", "peak=120/s"}, []string{"run-worker", "--load=steady", "peak=120/s"}},
		{"command in value", []string{"run-worker", "--option", "run-scenario", "--load=steady", "peak=120/s"}, []string{"run-worker", "--option", "run-scenario", "--load=steady", "peak=120/s"}},
		{"empty", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := slices.Clone(tc.args)
			require.Equal(t, tc.want, foldLoadArgs(args))
			require.Equal(t, tc.args, args)
		})
	}
}

func TestLoadArgsRejectLeftovers(t *testing.T) {
	for _, factory := range []func() *cobra.Command{runScenarioCmd, runScenarioWithWorkerCmd} {
		for _, tc := range []struct {
			name string
			args []string
		}{
			{"flag", []string{"--load", "spike-train", "peak=120/s", "--duration=20m", "every=5m"}},
			{"separator", []string{"--load=spike-train", "peak=120/s", "--", "every=5m"}},
			{"word", []string{"--load", "spike-train", "peak=120/s", "word", "every=5m"}},
		} {
			command := factory()
			t.Run(command.Name()+"/"+tc.name, func(t *testing.T) {
				args := foldLoadArgs(append([]string{command.Name()}, tc.args...))
				require.NoError(t, command.ParseFlags(args[1:]))

				require.NotNil(t, command.Args)
				require.EqualError(t, command.Args(command, command.Flags().Args()), "parameters must directly follow --load <pattern>; e.g. --load spike-train peak=120/s every=5m")
				require.NoError(t, command.Args(command, []string{"word"}))
			})
		}
	}
}

func TestLoadHelp(t *testing.T) {
	command := runScenarioCmd()
	help := command.Flags().Lookup("load").Usage
	for _, pattern := range []string{
		"steady: peak required",
		"ramp-hold-ramp: peak required, up=0.1, down=0.1",
		"sweep: peak required, from=0.25, to=1, steps=4",
		"spike-train: peak required, base=0.4, every=10m, for=1m",
	} {
		require.Contains(t, help, pattern)
	}
}
