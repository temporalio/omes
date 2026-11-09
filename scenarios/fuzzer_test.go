package scenarios

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap/zaptest"
)

// TestFuzzerAcceptsLoadInputFileAlone checks that load-input-file passed without conflicting options reads the
// LoadInput, even though output-file has a non-empty default.
func TestFuzzerAcceptsLoadInputFileAlone(t *testing.T) {
	info := loadgen.ScenarioInfo{
		ScenarioName:   "fuzzer",
		RunID:          "run",
		MetricsHandler: client.MetricsNopHandler,
		Logger:         zaptest.NewLogger(t).Sugar(),
		Options: loadgen.MustResolveScenarioOptions("fuzzer", map[string]string{
			"load-input-file": "testdata/load_input_example.json",
		}),
	}
	exec, ok := loadgen.GetScenario("fuzzer").ExecutorFn().(loadgen.FuzzExecutor)
	require.True(t, ok)
	in, err := exec.InitLoadInput(t.Context(), info)
	require.NoError(t, err)
	require.NotEmpty(t, in.GetInputs())
}

// TestFuzzerRejectsOptionsLoadInputReplaces checks that each option a LoadInput makes meaningless fails the run when
// passed with load-input-file, naming the option.
func TestFuzzerRejectsOptionsLoadInputReplaces(t *testing.T) {
	for name, value := range map[string]string{
		"input-file":       "input.proto",
		"seed":             "7",
		"config":           "{}",
		"output-file":      defaultFuzzOutputFile,
		"kitchen-sink-gen": "kitchen-sink-gen",
	} {
		t.Run(name, func(t *testing.T) {
			info := loadgen.ScenarioInfo{
				ScenarioName:   "fuzzer",
				RunID:          "run",
				MetricsHandler: client.MetricsNopHandler,
				Logger:         zaptest.NewLogger(t).Sugar(),
				Options: loadgen.MustResolveScenarioOptions("fuzzer", map[string]string{
					name:              value,
					"load-input-file": "load_input.json",
				}),
			}
			err := loadgen.GetScenario("fuzzer").ExecutorFn().Run(t.Context(), info)
			require.ErrorContains(t, err, "option "+name+" can't be used with load-input-file")
		})
	}
}
