package loadgen

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type startInputsClient struct {
	client.Client
	ids    []string
	inputs []int32
}

func (c *startInputsClient) ExecuteWorkflow(ctx context.Context, opts client.StartWorkflowOptions, _ interface{}, args ...interface{}) (client.WorkflowRun, error) {
	c.ids = append(c.ids, opts.ID)
	c.inputs = append(c.inputs, args[0].(*kitchensink.WorkflowInput).ExpectedSignalCount)
	if len(c.ids) == 1 {
		return nil, serviceerror.NewDeadlineExceeded("start response unavailable")
	}
	return completedOriginalRun{id: opts.ID}, nil
}

type completedOriginalRun struct {
	client.WorkflowRun
	id string
}

func (r completedOriginalRun) GetID() string                          { return r.id }
func (r completedOriginalRun) GetRunID() string                       { return "original-run" }
func (r completedOriginalRun) Get(context.Context, interface{}) error { return nil }

func TestKitchenSinkOriginalStartFreezesRetryConstruction(t *testing.T) {
	cl := &startInputsClient{}
	generation := int32(0)
	executor := KitchenSinkExecutor{
		TestInput: &kitchensink.TestInput{
			WorkflowInput: &kitchensink.WorkflowInput{},
		},
		UpdateWorkflowOptions: func(ctx context.Context, r *Run, o *KitchenSinkWorkflowOptions) error {
			generation++
			o.StartOptions.ID = uuid.NewString()
			o.Params.WorkflowInput.ExpectedSignalCount = generation
			return nil
		},
	}
	executor.RecoverOriginalStart = true

	info := ScenarioInfo{
		Client:         cl,
		Namespace:      "default",
		RunID:          "frozen",
		ExecutionID:    "same",
		Logger:         zap.NewNop().Sugar(),
		MetricsHandler: client.MetricsNopHandler,
		Configuration: RunConfiguration{
			Iterations:                    1,
			MaxConcurrent:                 1,
			MaxIterationAttempts:          2,
			DoNotRegisterSearchAttributes: true,
		},
	}
	require.NoError(t, executor.Run(t.Context(), info))
	require.Len(t, cl.ids, 2)
	require.Equal(t, cl.ids[0], cl.ids[1])
	require.Equal(t, []int32{1, 1}, cl.inputs)
}

func TestKitchenSinkOriginalStartRejectsClientSequence(t *testing.T) {
	for _, configuredByCallback := range []bool{false, true} {
		t.Run(fmt.Sprintf("configured_by_callback=%t", configuredByCallback), func(t *testing.T) {
			cl := &startInputsClient{}
			seq := &kitchensink.ClientSequence{ActionSets: []*kitchensink.ClientActionSet{{}}}
			executor := KitchenSinkExecutor{
				RecoverOriginalStart: true,
				TestInput:            &kitchensink.TestInput{WorkflowInput: &kitchensink.WorkflowInput{}},
			}
			if configuredByCallback {
				executor.UpdateWorkflowOptions = func(_ context.Context, _ *Run, o *KitchenSinkWorkflowOptions) error {
					o.Params.ClientSequence = seq
					return nil
				}
			} else {
				executor.TestInput.ClientSequence = seq
			}
			info := ScenarioInfo{
				Client:         cl,
				Namespace:      "default",
				Logger:         zap.NewNop().Sugar(),
				MetricsHandler: client.MetricsNopHandler,
				Configuration: RunConfiguration{
					Iterations:                    1,
					MaxConcurrent:                 1,
					DoNotRegisterSearchAttributes: true,
				},
			}
			err := executor.Run(t.Context(), info)
			require.ErrorContains(t, err, "original start recovery does not support client sequences")
			require.Empty(t, cl.ids, "invalid recovery must be rejected before starting a workflow")
		})
	}
}
