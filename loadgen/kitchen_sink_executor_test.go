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
	common "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
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

	testStart              = time.Now().Unix()
	testRunTimeout         = 10 * time.Second
	workerBuildTimeout     = 1 * time.Minute
	workerShutdownTimeout  = 5 * time.Second
	defaultActivityTimeout = durationpb.New(5 * time.Second)

	javaMutex       sync.Mutex
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

	// Register cleanup of workers builds after test completion.
	t.Cleanup(func() {
		workerMutex.Lock()
		defer workerMutex.Unlock()
		for _, cleanupFunc := range workerCleanupFuncs {
			cleanupFunc()
		}
	})

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
				t.Run(string(sdk), func(t *testing.T) {
					t.Parallel()

					// Use mutex to ensure only one Java test runs at a time - a Gradle limitation :-(
					// Note re-using workers to ensure their logs are not mixed up across tests.
					if sdk == cmdoptions.LangJava {
						javaMutex.Lock()
						defer javaMutex.Unlock()
					}

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
	logger := zaptest.NewLogger(t).Sugar()
	scenarioID := cmdoptions.ScenarioID{
		Scenario: "kitchenSinkTest",
		RunID:    fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix()),
	}
	taskQueueName := TaskQueueForRun(scenarioID.Scenario, scenarioID.RunID)

	// Build worker outside of the test context.
	executorErr := ensureWorkerBuilt(t, logger, sdk)
	require.NoErrorf(t, executorErr, "Failed to build worker for SDK %s", sdk)

	testCtx, testCtxCancel := context.WithTimeout(t.Context(), testRunTimeout)
	defer testCtxCancel()

	// Start worker.
	workerCtx, workerCtxCancel := context.WithCancel(testCtx)
	workerDone := make(chan error, 1)
	go func() {
		defer close(workerDone)
		baseDir := workers.BaseDir(repoDir, sdk)
		runner := &workers.Runner{
			Builder: workers.Builder{
				DirName:    buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     logger.Named(fmt.Sprintf("[%s-worker]", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID:               scenarioID,
			ClientOptions: cmdoptions.ClientOptions{
				Address:   devServerAddr,
				Namespace: namespace,
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
		Logger:         logger.Named("[executor]"),
		MetricsHandler: client.MetricsNopHandler,
		Client:         temporalClient,
		Configuration:  executor.DefaultConfiguration,
		Namespace:      namespace,
	}
	executorErr = executor.Run(testCtx, scenarioInfo)

	// Wait for worker to stop; otherwise it keeps logging and cause a data race.
	workerCtxCancel()
	workerErr := <-workerDone
	if workerErr != nil {
		t.Logf("Worker shutdown with error: %v", workerErr)
	}

	// If feature is not supported; check for expected error.
	if expectedErr, expectUnsupported := tc.expectedUnsupportedErrs[sdk]; expectUnsupported {
		require.Errorf(t, executorErr,
			"SDK %s should fail for unsupported feature", sdk)
		require.NotEmptyf(t, expectedErr,
			"invalid test case: expectedUnsupportedErrs must be set for SDK %s if the feature is unsupported", sdk)
		require.Containsf(t, strings.ToLower(executorErr.Error()), expectedErr,
			"SDK %s error should contain '%s'", sdk, expectedErr)
		return
	}

	historyEvents, historyErr := getWorkflowHistory(t, taskQueueName, temporalClient)

	if executorErr != nil {
		if len(historyEvents) > 0 {
			t.Logf("History events for debugging: \n")
			logHistoryEvents(t, parseEvents(historyEvents))
		}
		require.NoError(t, executorErr, "executor failed")
	}
	require.NoError(t, historyErr, "failed to get workflow history")
	require.NotNilf(t, tc.historyMatcher, "invalid test case", "Test case '%s': historyMatcher must be set", tc.name)
	require.NoErrorf(t, tc.historyMatcher.Match(t, historyEvents), "Test case '%s': history matcher failed", tc.name)
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

func ensureWorkerBuilt(t *testing.T, logger *zap.SugaredLogger, sdk cmdoptions.Language) error {
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
			Logger:     logger.Named(fmt.Sprintf("[%s-builder]", sdk)),
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

func buildDirName() string {
	return fmt.Sprintf("omes-temp-%d", testStart)
}
