package scenarios

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// IncludeStandaloneActivityOperatorCommandsFlag enables standalone activity operator
	// commands in throughput_stress, defaulting to the namespace capability.
	IncludeStandaloneActivityOperatorCommandsFlag = "include-standalone-activity-operator-commands"
	// StandaloneActivityOperatorMaxConcurrentFlag limits concurrent operator tests. Default is 3.
	StandaloneActivityOperatorMaxConcurrentFlag = "standalone-activity-operator-max-concurrent"
	// StandaloneActivityOperatorMaxIterationsPerSecondFlag limits operator test
	// starts per second. Default is 0, meaning no rate cap.
	StandaloneActivityOperatorMaxIterationsPerSecondFlag = "standalone-activity-operator-max-iterations-per-second"
	// StandaloneActivityOperatorDrainTimeoutFlag sets how long running operator tests
	// may finish after the main load ends before they are canceled. Default is 30s.
	StandaloneActivityOperatorDrainTimeoutFlag = "standalone-activity-operator-drain-timeout"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Standalone activity operator lifecycle checks driven directly by the client. " +
			"Iterations rotate through pause/unpause, reset, and update, each waiting for activity completion. " +
			"Rate limits count operator test starts, not individual RPCs. Requires a worker implementing heartbeat.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.ExecutorFunc(runStandaloneActivityOperatorCommands)
		},
	})
}

func runStandaloneActivityOperatorCommands(ctx context.Context, info loadgen.ScenarioInfo) error {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	namespace, err := info.Client.WorkflowService().DescribeNamespace(probeCtx,
		&workflowservice.DescribeNamespaceRequest{Namespace: info.Namespace})
	cancel()
	if err != nil {
		return fmt.Errorf("check standalone activity operator support: %w", err)
	}
	if !namespace.GetNamespaceInfo().GetCapabilities().GetStandaloneActivityOperatorCommands() {
		return fmt.Errorf("namespace %q does not report standalone activity operator command support", info.Namespace)
	}
	return runStandaloneActivityOperatorSequences(ctx, info, nil)
}

func declareStandaloneActivityOperatorOptions(o *loadgen.OptionSet) {
	o.Feature(IncludeStandaloneActivityOperatorCommandsFlag,
		"Run standalone activity operator commands in an independent driver loop.",
		func(c loadgen.Capabilities) bool {
			return c.Namespace.GetStandaloneActivityOperatorCommands()
		})
	o.Int(StandaloneActivityOperatorMaxConcurrentFlag, 3, "Maximum concurrent operator tests.")
	o.Float64(StandaloneActivityOperatorMaxIterationsPerSecondFlag, 0, "Maximum operator test starts per second (0 means unlimited).")
	o.Duration(StandaloneActivityOperatorDrainTimeoutFlag, 30*time.Second, "Time for running operator tests to finish after the main load ends before cancellation.")
}

func (config *tpsConfig) configureStandaloneActivityOperator(info loadgen.ScenarioInfo) error {
	config.IncludeStandaloneActivityOperatorCommands = info.OptionBool(
		IncludeStandaloneActivityOperatorCommandsFlag)
	config.StandaloneActivityOperator = independentLoadConfig{
		MaxConcurrent:          info.OptionInt(StandaloneActivityOperatorMaxConcurrentFlag),
		MaxIterationsPerSecond: info.OptionFloat64(StandaloneActivityOperatorMaxIterationsPerSecondFlag),
		DrainTimeout:           info.OptionDuration(StandaloneActivityOperatorDrainTimeoutFlag),
	}

	if config.IncludeStandaloneActivityOperatorCommands {
		if config.StandaloneActivityOperator.MaxConcurrent <= 0 {
			return fmt.Errorf("%s must be positive", StandaloneActivityOperatorMaxConcurrentFlag)
		}

		rate := config.StandaloneActivityOperator.MaxIterationsPerSecond
		if rate != 0 {
			interval := float64(time.Second) / rate
			if rate < 0 || math.IsNaN(rate) || interval < 1 || interval >= float64(math.MaxInt64) {
				return fmt.Errorf("%s must be 0 (unlimited) or produce a finite ticker interval between 1ns and 292 years", StandaloneActivityOperatorMaxIterationsPerSecondFlag)
			}
		}

		if config.StandaloneActivityOperator.DrainTimeout <= 0 {
			return fmt.Errorf("%s must be positive", StandaloneActivityOperatorDrainTimeoutFlag)
		}
	}

	return nil
}

func (t *tpsExecutor) standaloneActivityOperatorLoad(info loadgen.ScenarioInfo) independentLoad {
	operatorInfo := info
	operatorInfo.ScenarioName = "throughput_stress_operators"
	operatorInfo.Logger = info.Logger.With("load_stream", "operators")
	// Use separate scheduling limits and progress tracking for operator tests.
	operatorInfo.Configuration = loadgen.RunConfiguration{
		MaxConcurrent:                 t.config.StandaloneActivityOperator.MaxConcurrent,
		MaxIterationsPerSecond:        t.config.StandaloneActivityOperator.MaxIterationsPerSecond,
		MaxIterationAttempts:          info.Configuration.MaxIterationAttempts,
		ContinueOnIterationFailure:    info.Configuration.ContinueOnIterationFailure,
		DoNotRegisterSearchAttributes: true,
	}
	return independentLoad{
		Name:         "operator",
		DrainTimeout: t.config.StandaloneActivityOperator.DrainTimeout,
		Run: func(ctx context.Context, stopNewIterations <-chan struct{}) error {
			return runStandaloneActivityOperatorSequences(ctx, operatorInfo, stopNewIterations)
		},
	}
}

func runStandaloneActivityOperatorSequences(ctx context.Context, info loadgen.ScenarioInfo, stopNewIterations <-chan struct{}) error {
	var started, completed atomic.Int64
	defer func() {
		info.Logger.Infof("Operator load summary: started=%d completed=%d", started.Load(), completed.Load())
	}()

	// No workflows are created, so this scenario needs no workflow search attributes.
	info.Configuration.DoNotRegisterSearchAttributes = true
	previousOnCompletion := info.Configuration.OnCompletion
	info.Configuration.OnCompletion = func(ctx context.Context, run *loadgen.Run) {
		completed.Add(1)
		command := standaloneActivityOperatorClientAction(run.TaskQueue(), run.Iteration-1).
			GetDoStandaloneActivityOperatorCommands().GetCommandType()
		info.Logger.Infof("Operator sequence completed: iteration=%d command=%s elapsed=%s",
			run.Iteration, command, run.Duration)
		if previousOnCompletion != nil {
			previousOnCompletion(ctx, run)
		}
	}

	executor := loadgen.GenericExecutor{
		StopNewIterations: stopNewIterations,
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			started.Add(1)
			action := standaloneActivityOperatorClientAction(run.TaskQueue(), run.Iteration-1)
			info.Logger.Infof("Operator sequence starting: iteration=%d command=%s task_queue=%s",
				run.Iteration, action.GetDoStandaloneActivityOperatorCommands().GetCommandType(), run.TaskQueue())
			sequence := &ClientSequence{
				ActionSets: []*ClientActionSet{{Actions: []*ClientAction{action}}},
			}
			clientExecutor := ClientActionsExecutor{
				Client:    info.Client,
				Namespace: info.Namespace,
			}
			return clientExecutor.ExecuteClientSequence(ctx, sequence)
		},
	}
	return executor.Run(ctx, info)
}

func standaloneActivityOperatorClientAction(taskQueue string, commandOrdinal int) *ClientAction {
	commandTypes := []DoStandaloneActivityOperatorCommands_CommandType{
		DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
		DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
		DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE,
	}
	commandType := commandTypes[commandOrdinal%len(commandTypes)]
	return &ClientAction{
		Variant: &ClientAction_DoStandaloneActivityOperatorCommands{
			DoStandaloneActivityOperatorCommands: &DoStandaloneActivityOperatorCommands{
				CommandType: commandType,
				Activity: &ExecuteActivityAction{
					ActivityType: &ExecuteActivityAction_Heartbeat{
						Heartbeat: &ExecuteActivityAction_HeartbeatTimeoutActivity{
							SuccessDuration:   durationpb.New(5 * time.Second),
							HeartbeatInterval: durationpb.New(time.Second),
						},
					},
					TaskQueue:              taskQueue,
					ScheduleToCloseTimeout: durationpb.New(2 * time.Minute),
					StartToCloseTimeout:    durationpb.New(15 * time.Second),
					HeartbeatTimeout:       durationpb.New(3 * time.Second),
					RetryPolicy:            &common.RetryPolicy{MaximumAttempts: 2},
				},
			},
		},
	}
}
