package loadgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	sdks = []cmdoptions.Language{
		cmdoptions.LangGo,
		cmdoptions.LangJava,
		cmdoptions.LangPython,
		cmdoptions.LangTypeScript,
		cmdoptions.LangDotNet,
	}
	onlySDK = os.Getenv("SDK")

	testDir, _ = os.Getwd()
	repoDir    = filepath.Dir(testDir)

	testStart             = time.Now().Unix()
	testTimeout           = 30 * time.Second
	workerBuildTimeout    = 1 * time.Minute
	workerShutdownTimeout = 5 * time.Second

	workerMutex     sync.RWMutex
	workerBuildOnce = map[cmdoptions.Language]*sync.Once{
		cmdoptions.LangGo:         new(sync.Once),
		cmdoptions.LangJava:       new(sync.Once),
		cmdoptions.LangPython:     new(sync.Once),
		cmdoptions.LangTypeScript: new(sync.Once),
		cmdoptions.LangDotNet:     new(sync.Once),
	}
	workerBuildErrs    = map[cmdoptions.Language]error{}
	workerCleanupFuncs []func()
)

type testCase struct {
	name                 string
	testInput            *TestInput
	expectedHistory      string
	unsupportedSDKs      []cmdoptions.Language
	expectedErrorMessage string
}

// TestKitchensink tests specific kitchensink features across SDKs.
// Use the `SDK` environment variable to run only a specific SDK.
func TestKitchensink(t *testing.T) {
	if os.Getenv("CI") != "" && onlySDK == "" {
		t.Skip("Skipping kitchensink test in CI without specific SDK set")
	}

	t.Cleanup(func() {
		workerMutex.Lock()
		defer workerMutex.Unlock()
		for _, cleanupFunc := range workerCleanupFuncs {
			cleanupFunc()
		}
	})

	// Start dev server.
	devServer := startDevServer(t)
	defer devServer.Stop()
	devServerAddr := devServer.FrontendHostPort()

	// Create Temporal client.
	temporalClient := createTemporalClient(t, devServerAddr)
	defer temporalClient.Close()

	for _, tc := range []testCase{
		{
			name: "TimerAction",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{
						{
							Actions: []*Action{
								{
									Variant: &Action_Timer{
										Timer: &TimerAction{
											Milliseconds: 1,
											AwaitableChoice: &AwaitableChoice{
												Condition: &AwaitableChoice_WaitFinish{
													WaitFinish: &emptypb.Empty{},
												},
											},
										},
									},
								},
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
				TimerStarted {"startToFireTimeout":"0.001s"}
				TimerFired
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`,
		},
		{
			name: "ExecActivity - Noop",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{
						{
							Actions: []*Action{
								{
									Variant: &Action_ExecActivity{
										ExecActivity: &ExecuteActivityAction{
											ActivityType: &ExecuteActivityAction_Noop{
												Noop: &emptypb.Empty{},
											},
											ScheduleToCloseTimeout: durationpb.New(10 * time.Second),
											ScheduleToStartTimeout: durationpb.New(9 * time.Second),
											StartToCloseTimeout:    durationpb.New(8 * time.Second),
										},
									},
								},
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
				ActivityTaskScheduled {"activityType":{"name":"noop"},"scheduleToCloseTimeout":"10s","scheduleToStartTimeout":"9s","startToCloseTimeout":"8s"}
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`,
		},
		{
			name: "ExecActivity - Noop (local)",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{
						{
							Actions: []*Action{
								{
									Variant: &Action_ExecActivity{
										ExecActivity: &ExecuteActivityAction{
											ActivityType: &ExecuteActivityAction_Noop{
												Noop: &emptypb.Empty{},
											},
											StartToCloseTimeout: durationpb.New(8 * time.Second),
											Locality: &ExecuteActivityAction_IsLocal{
												IsLocal: &emptypb.Empty{},
											},
										},
									},
								},
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
				MarkerRecorded  			# fields across SDKs vary here
				WorkflowExecutionCompleted`,
		},
		{
			name: "UnsupportedAction",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{
						{
							Actions: []*Action{
								{Variant: nil}, // this will be an unrecognized action
							},
						},
					},
				},
			},
			unsupportedSDKs:      sdks,
			expectedErrorMessage: "unrecognized action",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure the workflow completes by appending a return action at the end.
			input := tc.testInput
			input.WorkflowInput.InitialActions = append(input.WorkflowInput.InitialActions, &ActionSet{
				Actions: []*Action{
					{
						Variant: &Action_ReturnResult{ReturnResult: &ReturnResultAction{ReturnThis: &common.Payload{}}},
					},
				},
			})

			for _, sdk := range sdks {
				if onlySDK != "" && string(sdk) != onlySDK {
					continue // not using t.Skip is it's too noisy
				}
				t.Run(string(sdk), func(t *testing.T) {
					t.Parallel()
					testForSDK(t, tc, sdk, temporalClient, devServerAddr)
				})
			}
		})
	}
}

func testForSDK(
	t *testing.T,
	tc testCase,
	sdk cmdoptions.Language,
	temporalClient client.Client,
	devServerAddr string,
) {
	scenarioID := cmdoptions.ScenarioID{
		Scenario: "kitchenSinkTest",
		RunID:    fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix()),
	}
	taskQueueName := TaskQueueForRun(scenarioID.Scenario, scenarioID.RunID)

	// Determine if this feature is unsupported for this SDK.
	var isUnsupported bool
	var expectedError string
	for _, unsupportedSDK := range tc.unsupportedSDKs {
		if unsupportedSDK == sdk {
			isUnsupported = true
			expectedError = tc.expectedErrorMessage
			break
		}
	}

	// Build worker outside of the test context.
	err := ensureWorkerBuilt(t, sdk)
	require.NoErrorf(t, err, "Failed to build worker for SDK %s", sdk)

	testCtx, testCtxCancel := context.WithTimeout(t.Context(), testTimeout)
	defer testCtxCancel()

	// Start worker.
	workerCtx, workerCtxCancel := context.WithCancel(testCtx)
	workerDone := make(chan error, 1)
	workerLogger := zaptest.NewLogger(t).Sugar().Named(fmt.Sprintf("[%s-worker]", sdk))
	go func() {
		defer close(workerDone)
		baseDir := workers.BaseDir(repoDir, sdk)
		runner := &workers.Runner{
			Builder: workers.Builder{
				DirName:    buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     workerLogger,
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID:               scenarioID,
			ClientOptions: cmdoptions.ClientOptions{
				Address:   devServerAddr,
				Namespace: "default",
			},
		}
		workerDone <- runner.Run(workerCtx, baseDir)
	}()

	// Run executor.
	executor := KitchenSinkExecutor{
		TestInput: tc.testInput,
		DefaultConfiguration: RunConfiguration{
			Iterations:    1,
			MaxConcurrent: 1,
		},
	}
	scenarioInfo := ScenarioInfo{
		ScenarioName:   scenarioID.Scenario,
		RunID:          scenarioID.RunID,
		Logger:         zaptest.NewLogger(t).Sugar().Named("[executor]"),
		MetricsHandler: client.MetricsNopHandler,
		Client:         temporalClient,
		Configuration:  executor.DefaultConfiguration,
		Namespace:      "default",
	}
	err = executor.Run(testCtx, scenarioInfo)

	// Wait for worker to stop; otherwise it keeps logging and cause a data race.
	workerCtxCancel()
	workerErr := <-workerDone
	if workerErr != nil {
		t.Logf("Worker shutdown with error: %v", workerErr)
	}

	if isUnsupported {
		// Feature not supported; check for expected error.
		require.Errorf(t, err, "SDK %s should fail for unsupported feature", sdk)
		if expectedError != "" {
			actualErr := strings.ToLower(err.Error())
			require.Containsf(t, actualErr, expectedError, "SDK %s error should contain '%s'", sdk, tc.expectedErrorMessage)
		}
		return
	}
	require.NoError(t, err)

	// Feature is supported; check the event history.
	historyEvents := getWorkflowHistory(t, testCtx, taskQueueName, temporalClient)
	requireHistoryMatches(t, historyEvents, tc.expectedHistory)
}

func getWorkflowHistory(t *testing.T, ctx context.Context, taskQueueName string, temporalClient client.Client) []*history.HistoryEvent {
	executions, err := temporalClient.ListWorkflow(ctx,
		&workflowservice.ListWorkflowExecutionsRequest{
			Namespace: "default",
			Query:     fmt.Sprintf("TaskQueue = '%s'", taskQueueName),
		})
	require.NoError(t, err, "Failed to list workflow executions")
	require.Len(t, executions.Executions, 1)

	execution := executions.Executions[0]
	historyIter := temporalClient.GetWorkflowHistory(ctx, execution.Execution.WorkflowId, execution.Execution.RunId, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var historyEvents []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		require.NoError(t, err, "Failed to get next history event")
		historyEvents = append(historyEvents, event)
	}
	return historyEvents
}

func ensureWorkerBuilt(t *testing.T, sdk cmdoptions.Language) error {
	once := workerBuildOnce[sdk]
	once.Do(func() {
		baseDir := workers.BaseDir(repoDir, sdk)
		dirName := buildDirName()
		buildDir := filepath.Join(baseDir, dirName)

		workerMutex.Lock()
		workerCleanupFuncs = append(workerCleanupFuncs, func() {
			if err := os.RemoveAll(buildDir); err != nil {
				fmt.Printf("Failed to clean up build dir for %s at %s: %v\n", sdk, buildDir, err)
			}
		})
		workerMutex.Unlock()

		builder := workers.Builder{
			DirName:    dirName,
			SdkOptions: cmdoptions.SdkOptions{Language: sdk},
			Logger:     zaptest.NewLogger(t).Sugar().Named(fmt.Sprintf("[%s-builder]", sdk)),
		}

		ctx, cancel := context.WithTimeout(t.Context(), workerBuildTimeout)
		defer cancel()

		_, err := builder.Build(ctx, baseDir)

		workerMutex.Lock()
		workerBuildErrs[sdk] = err
		workerMutex.Unlock()
	})

	workerMutex.RLock()
	err := workerBuildErrs[sdk]
	workerMutex.RUnlock()

	return err
}

func createTemporalClient(t *testing.T, devServerAddr string) client.Client {
	temporalClient, err := client.Dial(client.Options{
		HostPort:  devServerAddr,
		Namespace: "default",
	})
	require.NoError(t, err, "Failed to create Temporal client")
	return temporalClient
}

func startDevServer(t *testing.T) *testsuite.DevServer {
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: "default",
		},
		LogLevel: "error",
	})
	require.NoError(t, err, "Failed to get dev server")
	return server
}

func buildDirName() string {
	return fmt.Sprintf("omes-temp-%d", testStart)
}
