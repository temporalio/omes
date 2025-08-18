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
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const namespace = "default"

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

	defaultActivityTimeout = durationpb.New(5 * time.Second)
	javaMutex              sync.Mutex
)

type testCase struct {
	name                    string
	testInput               *TestInput
	historyMatcher          HistoryMatcher
	expectedUnsupportedErrs map[cmdoptions.Language]string
}

// TestKitchenSink tests specific kitchensink features across SDKs.
// Use the `SDK` environment variable to run only a specific SDK.
func TestKitchenSink(t *testing.T) {
	if os.Getenv("CI") != "" && onlySDK == "" {
		t.Skip("Skipping kitchensink test in CI without specific SDK set")
	}
	env := SetupTestEnvironment(t)

	for _, tc := range []testCase{
		{
			name: "TimerAction",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
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
						}),
				},
			},
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				TimerStarted {"startToFireTimeout":"0.001s"}
				TimerFired
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ExecActivity - Noop",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
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
						}),
				},
			},
			historyMatcher: fullHistoryMatcher(`
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
				WorkflowExecutionCompleted`),
		},
		{
			name: "ExecActivity - Noop (local)",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: &ExecuteActivityAction_Noop{
										Noop: &emptypb.Empty{},
									},
									Locality: &ExecuteActivityAction_IsLocal{
										IsLocal: &emptypb.Empty{},
									},
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						}),
				},
			},
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled	
				WorkflowTaskStarted
				WorkflowTaskCompleted
				MarkerRecorded  			# fields across SDKs vary here
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence - Nested",
			testInput: &TestInput{
				ClientSequence: &ClientSequence{
					ActionSets: []*ClientActionSet{
						{
							Actions: []*ClientAction{
								{
									Variant: &ClientAction_DoSignal{
										DoSignal: &DoSignal{
											Variant: &DoSignal_DoSignalActions_{
												DoSignalActions: &DoSignal_DoSignalActions{
													Variant: &DoSignal_DoSignalActions_DoActions{
														DoActions: SingleActionSet(NewTimerAction(1)),
													},
												},
											},
										},
									},
								},
								{
									Variant: &ClientAction_DoUpdate{
										DoUpdate: &DoUpdate{
											Variant: &DoUpdate_DoActions{
												DoActions: &DoActionsUpdate{
													Variant: &DoActionsUpdate_DoActions{
														DoActions: SingleActionSet(NewTimerAction(1)),
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
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewTimerAction(2000), // timer to keep workflow open long enough for client action
					),
				},
			},
			historyMatcher: partialHistoryMatcher(`
				WorkflowExecutionSignaled
				...
				WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "UnsupportedAction",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{Variant: nil}), // unsupported action
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangGo:         "unrecognized action",
				cmdoptions.LangJava:       "unrecognized action",
				cmdoptions.LangPython:     "unrecognized action",
				cmdoptions.LangTypeScript: "unrecognized action",
				cmdoptions.LangDotNet:     "unrecognized action",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Ensure the workflow completes by appending a return action at the end.
			input := tc.testInput
			input.WorkflowInput.InitialActions = append(input.WorkflowInput.InitialActions, ListActionSet(NewEmptyReturnResultAction())...)

			for _, sdk := range sdks {
				if onlySDK != "" && string(sdk) != onlySDK {
					continue // not using t.Skip as it's too noisy
				}
				t.Run(string(sdk), func(t *testing.T) {
					t.Parallel()
					testForSDK(t, tc, sdk, env)
				})
			}
		})
	}
}

func testForSDK(
	t *testing.T,
	tc testCase,
	sdk cmdoptions.Language,
	env *TestEnvironment,
) {
	// Use mutex to ensure only one Java test runs at a time - a Gradle limitation.
	if sdk == cmdoptions.LangJava {
		javaMutex.Lock()
		defer javaMutex.Unlock()
	}

	executor := &KitchenSinkExecutor{
		TestInput: tc.testInput,
		DefaultConfiguration: RunConfiguration{
			Iterations: 1,
		},
	}

	scenarioInfo := ScenarioInfo{
		ScenarioName:  "kitchenSinkTest",
		RunID:         fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix()),
		Configuration: executor.DefaultConfiguration,
	}

	if expectedErr, expectUnsupported := tc.expectedUnsupportedErrs[sdk]; expectUnsupported {
		testUnsupportedFeature(t, env, executor, scenarioInfo, sdk, expectedErr)
	} else {
		testSupportedFeature(t, env, executor, scenarioInfo, tc, sdk)
	}
}

func testUnsupportedFeature(
	t *testing.T,
	env *TestEnvironment,
	executor *KitchenSinkExecutor,
	scenarioInfo ScenarioInfo,
	sdk cmdoptions.Language,
	expectedErr string,
) {
	testExecutor := &kitchenSinkTestWrapper{
		executor: executor,
		sdk:      sdk,
	}
	execErr := env.RunExecutorTest(t, testExecutor, scenarioInfo, sdk)

	require.Errorf(t, execErr, "SDK %s should fail for unsupported feature", sdk)
	require.NotEmptyf(t, expectedErr, "invalid test case: expectedUnsupportedErrs must be set for SDK %s if the feature is unsupported", sdk)
	require.Containsf(t, strings.ToLower(execErr.Error()), expectedErr, "SDK %s error should contain '%s'", sdk, expectedErr)
}

func testSupportedFeature(
	t *testing.T,
	env *TestEnvironment,
	executor *KitchenSinkExecutor,
	scenarioInfo ScenarioInfo,
	tc testCase,
	sdk cmdoptions.Language,
) {
	testExecutor := &kitchenSinkTestWrapper{
		executor: executor,
		sdk:      sdk,
	}
	execErr := env.RunExecutorTest(t, testExecutor, scenarioInfo, sdk)

	taskQueueName := TaskQueueForRun(scenarioInfo.ScenarioName, scenarioInfo.RunID)
	historyEvents, historyErr := getWorkflowHistory(t, taskQueueName, env.TemporalClient())
	if execErr != nil {
		if len(historyEvents) > 0 {
			t.Logf("History events for debugging:")
			logHistoryEvents(t, parseEvents(historyEvents))
		}
	}

	require.NoError(t, execErr, "executor failed")
	require.NoError(t, historyErr, "failed to get workflow history")
	require.NotNilf(t, tc.historyMatcher, "Test case '%s': historyMatcher must be set", tc.name)
	require.NoErrorf(t, tc.historyMatcher.Match(t, historyEvents), "Test case '%s': history matcher failed", tc.name)
}

type kitchenSinkTestWrapper struct {
	executor *KitchenSinkExecutor
	sdk      cmdoptions.Language
}

func (w *kitchenSinkTestWrapper) Run(ctx context.Context, info ScenarioInfo) error {
	return w.executor.Run(ctx, info)
}

func getWorkflowHistory(t *testing.T, taskQueueName string, temporalClient client.Client) ([]*history.HistoryEvent, error) {
	executions, err := temporalClient.ListWorkflow(t.Context(),
		&workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
			Query:     fmt.Sprintf("TaskQueue = '%s'", taskQueueName),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow executions: %w", err)
	}
	if len(executions.Executions) == 0 {
		return nil, fmt.Errorf("no workflow executions found for task queue %s", taskQueueName)
	}
	if len(executions.Executions) > 1 {
		t.Logf("Warning: found %d workflow executions for task queue %s, using the first one", len(executions.Executions), taskQueueName)
	}

	execution := executions.Executions[0]
	historyIter := temporalClient.GetWorkflowHistory(t.Context(), execution.Execution.WorkflowId, execution.Execution.RunId, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var historyEvents []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			return historyEvents, fmt.Errorf("failed to get next history event: %w", err)
		}
		historyEvents = append(historyEvents, event)
	}
	return historyEvents, nil
}
