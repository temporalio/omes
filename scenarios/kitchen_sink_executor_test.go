package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TODO: move into kitchensink
// TestKitchensinkFeatureParity tests specific kitchensink features for SDKs to ensure parity.
// Tests are organized by action to make it easy to run specific actions across all SDKs.
// Use the "SDK" environment variable to run only a specific SDK, or run all by default.
func TestKitchensinkFeatureParity(t *testing.T) {
	testCases := []struct {
		name            string
		action          *Action
		expectedHistory string
		// SDK support flags - when true, expect execution to fail with non-retryable error
		goUnsupported         bool
		javaUnsupported       bool
		dotnetUnsupported     bool
		pythonUnsupported     bool
		typescriptUnsupported bool
		expectedErrorMessage  string
	}{
		{
			name: "TimerAction",
			action: &Action{
				Variant: &Action_Timer{
					Timer: &TimerAction{
						Milliseconds: 100,
						AwaitableChoice: &AwaitableChoice{
							Condition: &AwaitableChoice_WaitFinish{
								WaitFinish: &emptypb.Empty{},
							},
						},
					},
				},
			},
			expectedHistory: `
WorkflowExecutionStarted
WorkflowTaskScheduled
WorkflowTaskStarted
WorkflowTaskCompleted
TimerStarted
TimerFired
WorkflowTaskScheduled
WorkflowTaskStarted
WorkflowTaskCompleted
WorkflowExecutionCompleted
`,
		},
		{
			name: "ExecActivity: LocalActivity",
			action: &Action{
				Variant: &Action_ExecActivity{
					ExecActivity: &ExecuteActivityAction{
						ActivityType: &ExecuteActivityAction_Noop{
							Noop: &emptypb.Empty{},
						},
						ScheduleToCloseTimeout: durationpb.New(10 * time.Second),
						StartToCloseTimeout:    durationpb.New(5 * time.Second),
						Locality: &ExecuteActivityAction_IsLocal{
							IsLocal: &emptypb.Empty{},
						},
					},
				},
			},
			expectedHistory: `
WorkflowExecutionStarted
WorkflowTaskScheduled
WorkflowTaskStarted
WorkflowTaskCompleted
MarkerRecorded
`,
		},
		{
			name: "UnsupportedAction",
			action: &Action{
				// This will be an invalid/unsupported action - test that it returns proper error
				Variant: nil,
			},
			goUnsupported:         true,
			javaUnsupported:       true,
			dotnetUnsupported:     true,
			pythonUnsupported:     true,
			typescriptUnsupported: true,
			expectedErrorMessage:  "unrecognized action",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdks := []cmdoptions.Language{
				cmdoptions.LangGo,
				cmdoptions.LangJava,
				cmdoptions.LangPython,
				cmdoptions.LangTypeScript,
				cmdoptions.LangDotNet,
			}

			if envSDK := os.Getenv("SDK"); envSDK != "" {
				filteredSDKs := []cmdoptions.Language{}
				for _, sdk := range sdks {
					if string(sdk) == envSDK {
						filteredSDKs = append(filteredSDKs, sdk)
						break
					}
				}
				sdks = filteredSDKs
			}

			for _, sdk := range sdks {
				t.Run(string(sdk), func(t *testing.T) {
					t.Parallel()
					testActionForSDK(t, tc, sdk)
				})
			}
		})
	}
}

// testActionForSDK runs a single action test case for a specific SDK
func testActionForSDK(t *testing.T, tc struct {
	name            string
	action          *Action
	expectedHistory string
	// SDK support flags - when true, expect execution to fail with non-retryable error
	goUnsupported         bool
	javaUnsupported       bool
	dotnetUnsupported     bool
	pythonUnsupported     bool
	typescriptUnsupported bool
	expectedErrorMessage  string
}, lang cmdoptions.Language) {
	ctx := context.Background()

	// Start embedded dev server.
	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: "default",
		},
		LogLevel: "error",
	})
	require.NoError(t, err, "Failed to start embedded dev server")
	t.Cleanup(func() {
		t.Log("Stopping embedded dev server")
		if err := server.Stop(); err != nil {
			t.Logf("Failed stopping embedded dev server: %v", err)
		}
	})

	temporalAddr := server.FrontendHostPort()
	t.Logf("Started embedded dev server at: %v", temporalAddr)

	// Create Temporal client for test execution
	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalAddr,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to create Temporal client")
	t.Cleanup(func() {
		temporalClient.Close()
	})

	// Build and start worker for the specified SDK using Runner
	runID := fmt.Sprintf("test-%s-%d", lang.String(), time.Now().Unix())
	taskQueue := loadgen.TaskQueueForRun("kitchen_sink", runID)
	worker, err := startTestWorker(ctx, t, lang, runID, temporalAddr)
	require.NoError(t, err, "Failed to start worker for SDK %s", lang.String())
	defer worker.stop()

	// Determine if this feature is unsupported for this SDK
	var isUnsupported bool
	var expectedError string
	switch lang {
	case cmdoptions.LangGo:
		isUnsupported = tc.goUnsupported
	case cmdoptions.LangJava:
		isUnsupported = tc.javaUnsupported
	case cmdoptions.LangDotNet:
		isUnsupported = tc.dotnetUnsupported
	case cmdoptions.LangPython:
		isUnsupported = tc.pythonUnsupported
	case cmdoptions.LangTypeScript:
		isUnsupported = tc.typescriptUnsupported
	default:
		t.Fatalf("Unknown SDK: %s", lang.String())
	}
	if isUnsupported {
		expectedError = tc.expectedErrorMessage
	}

	sdk := lang.String()

	// Create logger
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()

	// Create Temporal client
	temporalClient2, err := client.Dial(client.Options{
		HostPort:  temporalAddr,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to create Temporal client")
	defer temporalClient2.Close()

	// Create test input with the action, followed by a return result to complete the workflow
	testInput := &TestInput{
		WorkflowInput: &WorkflowInput{
			InitialActions: []*ActionSet{
				{
					Actions:    []*Action{tc.action},
					Concurrent: false,
				},
				{
					Actions: []*Action{
						{
							Variant: &Action_ReturnResult{
								ReturnResult: &ReturnResultAction{
									ReturnThis: nil,
								},
							},
						},
					},
					Concurrent: false,
				},
			},
		},
	}

	// Configure the KitchenSinkExecutor - use the same runID as the worker
	executor := loadgen.KitchenSinkExecutor{
		TestInput: testInput,
		PrepareTestInput: func(ctx context.Context, info loadgen.ScenarioInfo, input *TestInput) error {
			// Set SDK-specific state if needed
			for _, actionSet := range input.WorkflowInput.InitialActions {
				for _, action := range actionSet.Actions {
					if setState := action.GetSetWorkflowState(); setState != nil {
						setState.Kvs["sdk_name"] = sdk
					}
				}
			}
			return nil
		},
		UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
			// Override task queue to use our test worker
			options.StartOptions.TaskQueue = taskQueue
			return nil
		},
		DefaultConfiguration: loadgen.RunConfiguration{
			Iterations:    1,
			MaxConcurrent: 1,
			Timeout:       30 * time.Second,
		},
	}

	// Configure scenario info
	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   "kitchen_sink_test",
		RunID:          runID,
		Logger:         logger.Sugar(),
		MetricsHandler: client.MetricsNopHandler,
		Client:         temporalClient2,
		Configuration:  executor.DefaultConfiguration,
		ScenarioOptions: map[string]string{
			"language": sdk,
		},
		Namespace: "default",
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Run the executor
	err = executor.Run(testCtx, scenarioInfo)

	if isUnsupported {
		// Feature not supported - expect failure with specific error message
		require.Error(t, err, "SDK %s should fail for unsupported feature", sdk)
		if expectedError != "" {
			require.Contains(t, err.Error(), expectedError, "SDK %s error should contain '%s'", sdk, tc.expectedErrorMessage)
		}
		t.Logf("SDK %s correctly failed with error: %v", sdk, err)
		return
	}

	// Feature is supported - expect success
	require.NoError(t, err, "SDK %s failed to execute action", sdk)
	t.Logf("SDK %s successfully executed action", sdk)

	historyEvents, err := getWorkflowHistory(ctx, temporalClient2)
	require.NoError(t, err, "Failed to get workflow history for SDK %s", sdk)
	RequireHistoryMatches(t, historyEvents, tc.expectedHistory)
}

// testWorker represents a running worker for testing
type testWorker struct {
	cancel context.CancelFunc
	done   <-chan error
}

// startTestWorker starts a worker using the workers.Runner for the specified SDK
func startTestWorker(ctx context.Context, t *testing.T, lang cmdoptions.Language, runID, temporalAddr string) (*testWorker, error) {
	logger := zap.Must(zap.NewDevelopment()).Sugar()
	
	// Create a scenario ID for this test using flag simulation
	var scenarioID cmdoptions.ScenarioID
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	scenarioID.AddCLIFlags(fs)
	fs.Set("scenario", "kitchen_sink")
	fs.Set("run-id", runID)

	runner := &workers.Runner{
		Builder: workers.Builder{
			SdkOptions: cmdoptions.SdkOptions{
				Language: lang,
			},
			Logger: logger,
		},
		RetainTempDir:            false,
		GracefulShutdownDuration: 5 * time.Second,
		EmbeddedServer:           false, // We already have a server running
		ScenarioID:               scenarioID,
		ClientOptions: cmdoptions.ClientOptions{
			Address:   temporalAddr,
			Namespace: "default",
		},
		TaskQueueIndexSuffixStart: 0,
		TaskQueueIndexSuffixEnd:   0,
	}

	// Start the worker in a separate goroutine
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	
	go func() {
		done <- runner.Run(workerCtx, rootDir())
	}()

	// Give worker time to start up and register with server
	time.Sleep(3 * time.Second)

	return &testWorker{
		cancel: cancel,
		done:   done,
	}, nil
}

// stop stops the worker process
func (w *testWorker) stop() {
	w.cancel()
	// Wait for worker to finish or timeout
	select {
	case <-w.done:
	case <-time.After(5 * time.Second):
		// Worker didn't shut down gracefully in time
	}
}

// getWorkflowHistory gets the workflow history for the most recent execution
func getWorkflowHistory(ctx context.Context, temporalClient client.Client) ([]*history.HistoryEvent, error) {
	// Get the most recent workflow execution
	executions, err := temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: "default",
		Query: fmt.Sprintf("WorkflowType = 'kitchenSink' AND StartTime > '%s'",
			time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing workflows: %w", err)
	}

	if len(executions.Executions) == 0 {
		return nil, fmt.Errorf("no workflow executions found")
	}

	execution := executions.Executions[0]
	testWorkflowID := execution.Execution.WorkflowId
	testRunID := execution.Execution.RunId

	// Get workflow history
	historyIter := temporalClient.GetWorkflowHistory(ctx, testWorkflowID, testRunID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

	var historyEvents []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed getting next history event: %w", err)
		}
		historyEvents = append(historyEvents, event)
	}

	return historyEvents, nil
}

// rootDir returns the root directory of the OMES project
func rootDir() string {
	_, currFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(currFile))
}
