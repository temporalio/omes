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
	common "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap/zaptest"
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

	javaMutex sync.Mutex
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

	// Test environment will handle cleanup

	// Start dev server.
	devServer := startDevServer(t)
	t.Cleanup(func() {
		devServer.Stop()
	})
	devServerAddr := devServer.FrontendHostPort()

	// Create Temporal client.
	temporalClient := createTemporalClient(t, devServerAddr)
	t.Cleanup(func() {
		temporalClient.Close()
	})

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
			name: "ExecActivity - Client - Signal - DoActions",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						},
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ExecActivity - Client - Signal - WithStart",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						},
						NewAwaitWorkflowStateAction("status", "done")),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ExecActivity - Client - Signal - Custom",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						}),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionSignaled {"signalName":"test_signal"}`),
		},
		{
			name: "ExecActivity - Client - Query - ReportState",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
										ClientActions(&ClientAction{
											Variant: &ClientAction_DoQuery{
												DoQuery: &DoQuery{
													Variant: &DoQuery_ReportState{
														ReportState: &common.Payloads{},
													},
												},
											},
										}),
									),
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
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
		},
		{
			name: "ExecActivity - Client - Query - Custom - Failure",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
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
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
		},
		{
			name: "ExecActivity - Client - Update - DoActions",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						},
					),
				},
			},
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionUpdateAccepted {"acceptedRequest":{"input":{"name":"do_actions_update"}}}
				TimerStarted
				TimerFired
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionUpdateCompleted
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
		},
		{
			name: "ExecActivity - Client - Update - Custom - Failure",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
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
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
		},
		{
			name: "ExecActivity - Client - Update - WithStart",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
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
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						},
						NewAwaitWorkflowStateAction("status", "done")),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ExecActivity - Client - Concurrent",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_ExecActivity{
								ExecActivity: &ExecuteActivityAction{
									ActivityType: ClientActivity(
										&ClientSequence{
											ActionSets: []*ClientActionSet{
												{
													Concurrent: true,
													Actions:    []*ClientAction{},
												},
											},
										},
									),
									StartToCloseTimeout: defaultActivityTimeout,
								},
							},
						}),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangTypeScript: "concurrent client actions are not supported",
				cmdoptions.LangDotNet:     "concurrent client actions are not supported",
				cmdoptions.LangPython:     "concurrent client actions are not supported",
				cmdoptions.LangJava:       "client actions are not supported",
			},
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				ActivityTaskScheduled {"activityType":{"name":"client"}}
				ActivityTaskStarted
				ActivityTaskCompleted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ExecActivity - Client - Nested",
			testInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						&Action{
							Variant: &Action_NestedActionSet{
								NestedActionSet: &ActionSet{
									Actions: []*Action{
										{
											Variant: &Action_ExecActivity{
												ExecActivity: &ExecuteActivityAction{
													ActivityType: ClientActivity(
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
													),
													StartToCloseTimeout: defaultActivityTimeout,
												},
											},
										},
										{
											Variant: &Action_ExecActivity{
												ExecActivity: &ExecuteActivityAction{
													ActivityType: ClientActivity(
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
													),
													StartToCloseTimeout: defaultActivityTimeout,
												},
											},
										},
									},
								},
							},
						}),
				},
			},
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "client actions are not supported",
			},
			historyMatcher: partialHistoryMatcher(`
				WorkflowExecutionSignaled
				...
				WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ClientSequence - Signal - DoActions",
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
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ClientSequence - Signal - Custom",
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
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionSignaled`),
		},
		{
			name: "ClientSequence - Query - ReportState",
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
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence - Query - Custom - Failure",
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
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
		},
		{
			name: "ClientSequence - Update - DoActions",
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
			historyMatcher: partialHistoryMatcher(`WorkflowExecutionUpdateCompleted`),
		},
		{
			name: "ClientSequence - Update - Custom - Failure",
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
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
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
			name: "WithStartAction - Signal",
			testInput: &TestInput{
				WithStartAction: &WithStartClientAction{
					Variant: &WithStartClientAction_DoSignal{
						DoSignal: &DoSignal{
							WithStart: true,
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
				},
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			historyMatcher: fullHistoryMatcher(`
				WorkflowExecutionStarted
				WorkflowExecutionSignaled {"signalName":"do_actions_signal"}
				WorkflowTaskScheduled
				WorkflowTaskStarted
				WorkflowTaskCompleted
				WorkflowExecutionCompleted`),
			expectedUnsupportedErrs: map[cmdoptions.Language]string{
				cmdoptions.LangJava: "context deadline exceeded", // BUG! The SetWorkflowStateAction does not work.
			},
		},
		{
			name: "WithStartAction - Update",
			testInput: &TestInput{
				WithStartAction: &WithStartClientAction{
					Variant: &WithStartClientAction_DoUpdate{
						DoUpdate: &DoUpdate{
							WithStart: true,
							Variant: &DoUpdate_DoActions{
								DoActions: &DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(
											NewSetWorkflowStateAction("status", "done"),
										),
									},
								},
							},
						},
					},
				},
				WorkflowInput: &WorkflowInput{
					InitialActions: ListActionSet(
						NewAwaitWorkflowStateAction("status", "done"),
					),
				},
			},
			historyMatcher: fullHistoryMatcher(`
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
					continue // not using t.Skip is it's too noisy
				}
				
				testForSDK(t, tc, sdk, temporalClient, devServerAddr)
			}
		})
	}
}

func testForSDK(
	parentT *testing.T,
	tc testCase,
	sdk cmdoptions.Language,
	temporalClient client.Client,
	devServerAddr string,
) {
	parentT.Run(string(sdk), func(childT *testing.T) {
		childT.Parallel()

		// Use mutex to ensure only one Java test runs at a time - a Gradle limitation
		if sdk == cmdoptions.LangJava {
			javaMutex.Lock()
			defer javaMutex.Unlock()
		}

		// Create test environment that handles all worker/devserver complexity
		env := &TestEnvironment{
			DevServer:      nil, // Will be set by the helper when needed
			TemporalClient: temporalClient,
			Logger:         zaptest.NewLogger(childT).Sugar(),
			RepoDir:        repoDir,
		}

		// Create the executor
		executor := &KitchenSinkExecutor{
			TestInput: tc.testInput,
			DefaultConfiguration: RunConfiguration{
				Iterations:    1,
				MaxConcurrent: 1,
			},
		}

		// Setup scenario info
		scenarioInfo := ScenarioInfo{
			ScenarioName: "kitchenSinkTest",
			RunID:        fmt.Sprintf("%s-%d", childT.Name(), time.Now().Unix()),
			Configuration: executor.DefaultConfiguration,
		}

		// Check if feature is expected to be unsupported
		if expectedErr, expectUnsupported := tc.expectedUnsupportedErrs[sdk]; expectUnsupported {
			// Test the unsupported feature case
			testUnsupportedFeature(childT, env, executor, scenarioInfo, sdk, expectedErr, devServerAddr)
		} else {
			// Test the supported feature case
			testSupportedFeature(childT, env, executor, scenarioInfo, tc, sdk, temporalClient, devServerAddr)
		}
	})
}

// testUnsupportedFeature tests that unsupported features fail with expected errors
func testUnsupportedFeature(t *testing.T, env *TestEnvironment, executor *KitchenSinkExecutor, scenarioInfo ScenarioInfo, sdk cmdoptions.Language, expectedErr, devServerAddr string) {
	// Run test using helper that manages worker/devserver
	testExecutor := &kitchenSinkTestWrapper{
		executor:      executor,
		sdk:           sdk,
		devServerAddr: devServerAddr,
	}

	err := env.runExecutorTestWithSDK(t, testExecutor, scenarioInfo, sdk, devServerAddr)

	// Verify it fails with expected error
	require.Error(t, err, "SDK %s should fail for unsupported feature", sdk)
	require.NotEmpty(t, expectedErr, "invalid test case: expectedUnsupportedErrs must be set for SDK %s if the feature is unsupported", sdk)
	require.Contains(t, strings.ToLower(err.Error()), expectedErr, "SDK %s error should contain '%s'", sdk, expectedErr)
}

// testSupportedFeature tests that supported features work correctly and match expected history
func testSupportedFeature(t *testing.T, env *TestEnvironment, executor *KitchenSinkExecutor, scenarioInfo ScenarioInfo, tc testCase, sdk cmdoptions.Language, temporalClient client.Client, devServerAddr string) {
	// Run test using helper that manages worker/devserver
	testExecutor := &kitchenSinkTestWrapper{
		executor:      executor,
		sdk:           sdk,
		devServerAddr: devServerAddr,
	}

	err := env.runExecutorTestWithSDK(t, testExecutor, scenarioInfo, sdk, devServerAddr)
	
	// Get workflow history for verification
	taskQueueName := TaskQueueForRun(scenarioInfo.ScenarioName, scenarioInfo.RunID)
	historyEvents, historyErr := getWorkflowHistory(t, taskQueueName, temporalClient)

	if err != nil {
		if len(historyEvents) > 0 {
			t.Logf("History events for debugging:")
			logHistoryEvents(t, parseEvents(historyEvents))
		}
		require.NoError(t, err, "executor failed")
	}
	
	require.NoError(t, historyErr, "failed to get workflow history")
	require.NotNil(t, tc.historyMatcher, "Test case '%s': historyMatcher must be set", tc.name)
	require.NoError(t, tc.historyMatcher.Match(t, historyEvents), "Test case '%s': history matcher failed", tc.name)
}

// kitchenSinkTestWrapper wraps the kitchen sink executor for use with the test helper
type kitchenSinkTestWrapper struct {
	executor      *KitchenSinkExecutor
	sdk           cmdoptions.Language
	devServerAddr string
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

// ensureWorkerBuilt is now handled by TestEnvironment.ensureWorkerBuilt

func createTemporalClient(t *testing.T, devServerAddr string) client.Client {
	temporalClient, err := client.Dial(client.Options{
		HostPort:  devServerAddr,
		Namespace: namespace,
	})
	require.NoError(t, err, "Failed to create Temporal client")
	return temporalClient
}

func startDevServer(t *testing.T) *testsuite.DevServer {
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: namespace,
		},
		LogLevel: "error",
	})
	require.NoError(t, err, "Failed to get dev server")
	return server
}
