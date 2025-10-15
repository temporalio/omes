package loadgen_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	. "github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	. "github.com/temporalio/omes/workers"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const namespace = "default"

var (
	sdks = []clioptions.Language{
		clioptions.LangGo,
		clioptions.LangJava,
		clioptions.LangPython,
		clioptions.LangTypeScript,
		clioptions.LangDotNet,
	}
	onlySDK   = os.Getenv("SDK")
	javaMutex sync.Mutex
)

type testCase struct {
	name                    string
	testInput               *TestInput
	historyMatcher          HistoryMatcher
	expectedUnsupportedErrs map[clioptions.Language]string
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
			historyMatcher: PartialHistoryMatcher(`
				TimerStarted {"startToFireTimeout":"0.001s"}
				TimerFired`),
		},
		{
			name: "ExecActivity/Noop",
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
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"noop"},"scheduleToCloseTimeout":"10s","scheduleToStartTimeout":"9s","startToCloseTimeout":"8s"}
				ActivityTaskStarted
				ActivityTaskCompleted`),
		},
		{
			name: "ExecActivity/Noop (local)",
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
									StartToCloseTimeout: durationpb.New(5 * time.Second),
								},
							},
						}),
				},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				MarkerRecorded  			# fields across SDKs vary here
				WorkflowExecutionCompleted`),
		},
		{
			name: "ExecActivity/ExecChildWorkflow",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecChildWorkflow{
								ExecChildWorkflow: &ExecuteChildWorkflowAction{
									WorkflowId:   "my-child",
									WorkflowType: "kitchenSink",
									Input: []*common.Payload{
										ConvertToPayload(&WorkflowInput{
											InitialActions: ListActionSet(NewTimerAction(1 * time.Millisecond)),
										})},
									AwaitableChoice: &AwaitableChoice{
										Condition: &AwaitableChoice_Abandon{
											Abandon: &emptypb.Empty{},
										},
									},
								},
							},
						}),
				},
			},
			// Note: workflowId is now "my-child-{parent-workflow-id}" due to concurrent execution support
			historyMatcher: PartialHistoryMatcher(`
				StartChildWorkflowExecutionInitiated`),
		},
		{
			name: "ExecActivity/Client/Signal/DoActions",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoSignal{
									DoSignal: &DoSignal{
										Variant: &DoSignal_DoSignalActions_{
											DoSignalActions: &DoSignal_DoSignalActions{
												Variant: &DoSignal_DoSignalActions_DoActions{
													DoActions: SingleActionSet(
														NewSetWorkflowStateAction("status", "done"),
													),
												},
											},
										},
									},
								},
							}),
							DefaultRemoteActivity,
						),
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ExecActivity/Client/Signal/WithStart",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoSignal{
									DoSignal: &DoSignal{
										Variant: &DoSignal_DoSignalActions_{
											DoSignalActions: &DoSignal_DoSignalActions{
												Variant: &DoSignal_DoSignalActions_DoActions{
													DoActions: SingleActionSet(
														NewSetWorkflowStateAction("status", "done"),
													),
												},
											},
										},
										WithStart: true, // This makes it a signal-with-start
									},
								},
							}),
							DefaultRemoteActivity,
						),
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ExecActivity/Client/Signal/Custom",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoSignal{
									DoSignal: &DoSignal{
										Variant: &DoSignal_Custom{
											Custom: &HandlerInvocation{
												Name: "test_signal",
											},
										},
									},
								},
							}),
							DefaultRemoteActivity,
						),
					),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionSignaled {"signalName":"test_signal"}`),
		},
		{
			name: "ExecActivity/Client/Query/ReportState",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoQuery{
									DoQuery: &DoQuery{
										Variant: &DoQuery_ReportState{
											ReportState: &common.Payloads{},
										},
									},
								},
							}),
							DefaultRemoteActivity,
						),
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted`),
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
		},
		{
			name: "ExecActivity/Client/Query/Custom/Failure",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoQuery{
									DoQuery: &DoQuery{
										Variant: &DoQuery_Custom{
											Custom: &HandlerInvocation{
												Name: "nonexistent_query",
											},
										},
										FailureExpected: true, // This query doesn't exist
									},
								},
							}),
							DefaultRemoteActivity,
						),
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted`),
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
		},
		{
			name: "ExecActivity/Client/Update/DoActions",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
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
							}),
							DefaultRemoteActivity,
						),
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				...
				WorkflowExecutionUpdateAccepted {"acceptedRequest":{"input":{"name":"do_actions_update"}}}`),
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
		},
		{
			name: "ExecActivity/Client/Update/Custom/Failure",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoUpdate{
									DoUpdate: &DoUpdate{
										Variant: &DoUpdate_Custom{
											Custom: &HandlerInvocation{
												Name: "nonexistent_update",
											},
										},
										FailureExpected: true, // This update doesn't exist
									},
								},
							}),
							DefaultRemoteActivity,
						),
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted`),
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
		},
		{
			name: "ExecActivity/Client/Update/WithStart",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							ClientActions(&ClientAction{
								Variant: &ClientAction_DoUpdate{
									DoUpdate: &DoUpdate{
										Variant: &DoUpdate_DoActions{
											DoActions: &DoActionsUpdate{
												Variant: &DoActionsUpdate_DoActions{
													DoActions: SingleActionSet(
														NewSetWorkflowStateAction("status", "done"),
													),
												},
											},
										},
										WithStart: true, // This makes it an update-with-start
									},
								},
							}),
							DefaultRemoteActivity,
						),
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ExecActivity/Client/Concurrent",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						ClientActivity(
							&ClientSequence{
								ActionSets: []*ClientActionSet{
									{
										Concurrent: true,
										Actions:    []*ClientAction{},
									},
								},
							},
							DefaultRemoteActivity,
						),
					),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangTypeScript: "concurrent client actions are not supported",
				clioptions.LangDotNet:     "concurrent client actions are not supported",
				clioptions.LangPython:     "concurrent client actions are not supported",
				clioptions.LangJava:       "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted`),
		},
		{
			name: "ExecActivity/Client/Nested",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_NestedActionSet{
								NestedActionSet: &ActionSet{
									Actions: []*Action{
										ClientActivity(
											ClientActions(&ClientAction{
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
											}),
											DefaultRemoteActivity,
										),
										ClientActivity(
											ClientActions(&ClientAction{
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
											}),
											DefaultRemoteActivity,
										),
									},
								},
							},
						}),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "client actions activity is not supported",
			},
			historyMatcher: PartialHistoryMatcher(`
				WorkflowExecutionSignaled
				...
				WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ClientSequence/Signal/DoActions",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
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
				}),
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewTimerAction(2000), // timer to keep workflow open long enough for client action
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ClientSequence/Signal/Custom",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
					Variant: &ClientAction_DoSignal{
						DoSignal: &DoSignal{
							Variant: &DoSignal_Custom{
								Custom: &HandlerInvocation{
									Name: "my-signal",
								},
							},
						},
					},
				}),
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewTimerAction(2000), // timer to keep workflow open long enough for client action
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ClientSequence/Query/ReportState",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
					Variant: &ClientAction_DoQuery{
						DoQuery: &DoQuery{
							Variant: &DoQuery_ReportState{
								ReportState: &common.Payloads{},
							},
						},
					},
				}),
				WorkflowInput: &WorkflowInput{},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence/Query/Custom/Failure",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
					Variant: &ClientAction_DoQuery{
						DoQuery: &DoQuery{
							FailureExpected: true,
							Variant: &DoQuery_Custom{
								Custom: &HandlerInvocation{
									Name: "nonexistent-query",
								},
							},
						},
					},
				}),
				WorkflowInput: &WorkflowInput{},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence/Update/DoActions",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
					Variant: &ClientAction_DoUpdate{
						DoUpdate: &DoUpdate{
							Variant: &DoUpdate_DoActions{
								DoActions: &DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(
											NewTimerAction(1),
											NewSetWorkflowStateAction("status", "done"),
										),
									},
								},
							},
						},
					},
				}),
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ClientSequence/Update/Custom/Failure",
			testInput: &TestInput{
				ClientSequence: ClientActions(&ClientAction{
					Variant: &ClientAction_DoUpdate{
						DoUpdate: &DoUpdate{
							FailureExpected: true,
							Variant: &DoUpdate_Custom{
								Custom: &HandlerInvocation{
									Name: "nonexistent-update",
								},
							},
						},
					},
				}),
				WorkflowInput: &WorkflowInput{},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence/Nested",
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
						NewTimerAction(5000), // timer to keep workflow open long enough for client action
					),
				},
			},
			historyMatcher: PartialHistoryMatcher(`
				WorkflowExecutionSignaled
				...
				WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "WithStartAction/Signal",
			testInput: &TestInput{
				WithStartAction: &WithStartClientAction{
					Variant: &WithStartClientAction_DoSignal{
						DoSignal: &DoSignal{
							WithStart: true,
							Variant: &DoSignal_DoSignalActions_{
								DoSignalActions: &DoSignal_DoSignalActions{
									Variant: &DoSignal_DoSignalActions_DoActions{
										DoActions: SingleActionSet(),
									},
								},
							},
						},
					},
				},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowExecutionSignaled {"signalName":"do_actions_signal"}
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "WithStartAction/Update",
			testInput: &TestInput{
				WithStartAction: &WithStartClientAction{
					Variant: &WithStartClientAction_DoUpdate{
						DoUpdate: &DoUpdate{
							WithStart: true,
							Variant: &DoUpdate_DoActions{
								DoActions: &DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(),
									},
								},
							},
						},
					},
				},
			},
			historyMatcher: FullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionUpdateAccepted
				WorkflowExecutionUpdateCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "UnsupportedAction",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{Variant: nil}), // unsupported action
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangGo:         "unrecognized action",
				clioptions.LangJava:       "unrecognized action",
				clioptions.LangPython:     "unrecognized action",
				clioptions.LangTypeScript: "unrecognized action",
				clioptions.LangDotNet:     "unrecognized action",
			},
		},
		{
			name: "ScheduleOperations",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_NestedActionSet{
								NestedActionSet: &ActionSet{
									Actions: []*Action{
										{
											Variant: &Action_CreateSchedule{
												CreateSchedule: &CreateScheduleAction{
													ScheduleId: "test-schedule",
													Spec: &ScheduleSpec{
														CronExpressions: []string{"* * * * *"},
													},
													Action: &ScheduleAction{
														WorkflowId:   "test-schedule-wf",
														WorkflowType: "kitchenSink",
														Input: []*common.Payload{
															ConvertToPayload(&WorkflowInput{
																InitialActions: ListActionSet(NewTimerAction(1)),
															}),
														},
													},
													Policies: &SchedulePolicies{
														RemainingActions:    1,
														TriggerImmediately: false,
													},
												},
											},
										},
										{
											Variant: &Action_DescribeSchedule{
												DescribeSchedule: &DescribeScheduleAction{
													ScheduleId: "test-schedule",
												},
											},
										},
										{
											Variant: &Action_UpdateSchedule{
												UpdateSchedule: &UpdateScheduleAction{
													ScheduleId: "test-schedule",
													Spec: &ScheduleSpec{
														CronExpressions: []string{"*/5 * * * *"},
													},
												},
											},
										},
										{
											Variant: &Action_DeleteSchedule{
												DeleteSchedule: &DeleteScheduleAction{
													ScheduleId: "test-schedule",
												},
											},
										},
									},
									Concurrent: false,
								},
							},
						}),
				},
			},
			expectedUnsupportedErrs: map[clioptions.Language]string{
				clioptions.LangJava: "schedule operations are not yet implemented",
			},
			historyMatcher: PartialHistoryMatcher(`
				ActivityTaskScheduled {"activityType":{"name":"CreateScheduleActivity"}}
				ActivityTaskCompleted
				...
				ActivityTaskScheduled {"activityType":{"name":"DescribeScheduleActivity"}}
				ActivityTaskCompleted
				...
				ActivityTaskScheduled {"activityType":{"name":"UpdateScheduleActivity"}}
				ActivityTaskCompleted
				...
				ActivityTaskScheduled {"activityType":{"name":"DeleteScheduleActivity"}}
				ActivityTaskCompleted`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Ensure the workflow completes by appending a return action at the end.
			input := tc.testInput
			if input.WorkflowInput == nil {
				input.WorkflowInput = &WorkflowInput{}
			}
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
	sdk clioptions.Language,
	env *TestEnvironment,
) {
	// Use mutex to ensure only one Java test runs at a time/a Gradle limitation.
	if sdk == clioptions.LangJava {
		javaMutex.Lock()
		defer javaMutex.Unlock()
	}

	executor := &KitchenSinkExecutor{
		TestInput: tc.testInput,
	}
	scenarioInfo := ScenarioInfo{
		ScenarioName: "kitchenSinkTest",
		RunID:        fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix()),
		Configuration: RunConfiguration{
			Iterations: 1,
		},
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
	sdk clioptions.Language,
	expectedErr string,
) {
	testExecutor := &kitchenSinkTestWrapper{
		executor: executor,
		sdk:      sdk,
	}
	_, execErr := env.RunExecutorTest(t, testExecutor, scenarioInfo, sdk)

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
	sdk clioptions.Language,
) {
	testExecutor := &kitchenSinkTestWrapper{
		executor: executor,
		sdk:      sdk,
	}
	_, execErr := env.RunExecutorTest(t, testExecutor, scenarioInfo, sdk)

	taskQueueName := TaskQueueForRun(scenarioInfo.RunID)
	historyEvents, historyErr := getWorkflowHistory(t, taskQueueName, env.TemporalClient())
	if execErr != nil {
		if len(historyEvents) > 0 {
			t.Logf("History events for debugging:")
			LogHistoryEvents(t, historyEvents)
		}
	}

	require.NoError(t, execErr, "executor failed")
	require.NoError(t, historyErr, "failed to get workflow history")
	require.NotNilf(t, tc.historyMatcher, "Test case '%s': historyMatcher must be set", tc.name)
	require.NoErrorf(t, tc.historyMatcher.Match(t, historyEvents), "Test case '%s': history matcher failed", tc.name)
}

type kitchenSinkTestWrapper struct {
	executor *KitchenSinkExecutor
	sdk      clioptions.Language
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
