/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars*/
"use strict";

var $protobuf = require("protobufjs/light");

var $root = ($protobuf.roots.__temporal_kitchensink || ($protobuf.roots.__temporal_kitchensink = new $protobuf.Root()))
.addJSON({
  temporal: {
    nested: {
      omes: {
        nested: {
          kitchen_sink: {
            options: {
              go_package: "github.com/temporalio/omes/loadgen/kitchensink",
              java_package: "io.temporal.omes"
            },
            nested: {
              TestInput: {
                fields: {
                  workflowInput: {
                    type: "WorkflowInput",
                    id: 1
                  },
                  clientSequence: {
                    type: "ClientSequence",
                    id: 2
                  }
                }
              },
              ClientSequence: {
                fields: {
                  actionSets: {
                    rule: "repeated",
                    type: "ClientActionSet",
                    id: 1
                  }
                }
              },
              ClientActionSet: {
                fields: {
                  actions: {
                    rule: "repeated",
                    type: "ClientAction",
                    id: 1
                  },
                  concurrent: {
                    type: "bool",
                    id: 2
                  },
                  waitAtEnd: {
                    type: "google.protobuf.Duration",
                    id: 3
                  },
                  waitForCurrentRunToFinishAtEnd: {
                    type: "bool",
                    id: 4
                  }
                }
              },
              ClientAction: {
                oneofs: {
                  variant: {
                    oneof: [
                      "doSignal",
                      "doQuery",
                      "doUpdate",
                      "nestedActions"
                    ]
                  }
                },
                fields: {
                  doSignal: {
                    type: "DoSignal",
                    id: 1
                  },
                  doQuery: {
                    type: "DoQuery",
                    id: 2
                  },
                  doUpdate: {
                    type: "DoUpdate",
                    id: 3
                  },
                  nestedActions: {
                    type: "ClientActionSet",
                    id: 4
                  }
                }
              },
              DoSignal: {
                oneofs: {
                  variant: {
                    oneof: [
                      "doSignalActions",
                      "custom"
                    ]
                  }
                },
                fields: {
                  doSignalActions: {
                    type: "DoSignalActions",
                    id: 1
                  },
                  custom: {
                    type: "HandlerInvocation",
                    id: 2
                  }
                },
                nested: {
                  DoSignalActions: {
                    oneofs: {
                      variant: {
                        oneof: [
                          "doActions",
                          "doActionsInMain"
                        ]
                      }
                    },
                    fields: {
                      doActions: {
                        type: "ActionSet",
                        id: 1
                      },
                      doActionsInMain: {
                        type: "ActionSet",
                        id: 2
                      }
                    }
                  }
                }
              },
              DoQuery: {
                oneofs: {
                  variant: {
                    oneof: [
                      "reportState",
                      "custom"
                    ]
                  }
                },
                fields: {
                  reportState: {
                    type: "temporal.api.common.v1.Payloads",
                    id: 1
                  },
                  custom: {
                    type: "HandlerInvocation",
                    id: 2
                  },
                  failureExpected: {
                    type: "bool",
                    id: 10
                  }
                }
              },
              DoUpdate: {
                oneofs: {
                  variant: {
                    oneof: [
                      "doActions",
                      "custom"
                    ]
                  }
                },
                fields: {
                  doActions: {
                    type: "DoActionsUpdate",
                    id: 1
                  },
                  custom: {
                    type: "HandlerInvocation",
                    id: 2
                  },
                  failureExpected: {
                    type: "bool",
                    id: 10
                  }
                }
              },
              DoActionsUpdate: {
                oneofs: {
                  variant: {
                    oneof: [
                      "doActions",
                      "rejectMe"
                    ]
                  }
                },
                fields: {
                  doActions: {
                    type: "ActionSet",
                    id: 1
                  },
                  rejectMe: {
                    type: "google.protobuf.Empty",
                    id: 2
                  }
                }
              },
              HandlerInvocation: {
                fields: {
                  name: {
                    type: "string",
                    id: 1
                  },
                  args: {
                    rule: "repeated",
                    type: "temporal.api.common.v1.Payload",
                    id: 2
                  }
                }
              },
              WorkflowState: {
                fields: {
                  kvs: {
                    keyType: "string",
                    type: "string",
                    id: 1
                  }
                }
              },
              WorkflowInput: {
                fields: {
                  initialActions: {
                    rule: "repeated",
                    type: "ActionSet",
                    id: 1
                  }
                }
              },
              ActionSet: {
                fields: {
                  actions: {
                    rule: "repeated",
                    type: "Action",
                    id: 1
                  },
                  concurrent: {
                    type: "bool",
                    id: 2
                  }
                }
              },
              Action: {
                oneofs: {
                  variant: {
                    oneof: [
                      "timer",
                      "execActivity",
                      "execChildWorkflow",
                      "awaitWorkflowState",
                      "sendSignal",
                      "cancelWorkflow",
                      "setPatchMarker",
                      "upsertSearchAttributes",
                      "upsertMemo",
                      "setWorkflowState",
                      "returnResult",
                      "returnError",
                      "continueAsNew",
                      "nestedActionSet"
                    ]
                  }
                },
                fields: {
                  timer: {
                    type: "TimerAction",
                    id: 1
                  },
                  execActivity: {
                    type: "ExecuteActivityAction",
                    id: 2
                  },
                  execChildWorkflow: {
                    type: "ExecuteChildWorkflowAction",
                    id: 3
                  },
                  awaitWorkflowState: {
                    type: "AwaitWorkflowState",
                    id: 4
                  },
                  sendSignal: {
                    type: "SendSignalAction",
                    id: 5
                  },
                  cancelWorkflow: {
                    type: "CancelWorkflowAction",
                    id: 6
                  },
                  setPatchMarker: {
                    type: "SetPatchMarkerAction",
                    id: 7
                  },
                  upsertSearchAttributes: {
                    type: "UpsertSearchAttributesAction",
                    id: 8
                  },
                  upsertMemo: {
                    type: "UpsertMemoAction",
                    id: 9
                  },
                  setWorkflowState: {
                    type: "WorkflowState",
                    id: 10
                  },
                  returnResult: {
                    type: "ReturnResultAction",
                    id: 11
                  },
                  returnError: {
                    type: "ReturnErrorAction",
                    id: 12
                  },
                  continueAsNew: {
                    type: "ContinueAsNewAction",
                    id: 13
                  },
                  nestedActionSet: {
                    type: "ActionSet",
                    id: 14
                  }
                }
              },
              AwaitableChoice: {
                oneofs: {
                  condition: {
                    oneof: [
                      "waitFinish",
                      "abandon",
                      "cancelBeforeStarted",
                      "cancelAfterStarted",
                      "cancelAfterCompleted"
                    ]
                  }
                },
                fields: {
                  waitFinish: {
                    type: "google.protobuf.Empty",
                    id: 1
                  },
                  abandon: {
                    type: "google.protobuf.Empty",
                    id: 2
                  },
                  cancelBeforeStarted: {
                    type: "google.protobuf.Empty",
                    id: 3
                  },
                  cancelAfterStarted: {
                    type: "google.protobuf.Empty",
                    id: 4
                  },
                  cancelAfterCompleted: {
                    type: "google.protobuf.Empty",
                    id: 5
                  }
                }
              },
              TimerAction: {
                fields: {
                  milliseconds: {
                    type: "uint64",
                    id: 1
                  },
                  awaitableChoice: {
                    type: "AwaitableChoice",
                    id: 2
                  }
                }
              },
              ExecuteActivityAction: {
                oneofs: {
                  activityType: {
                    oneof: [
                      "generic",
                      "delay",
                      "noop"
                    ]
                  },
                  locality: {
                    oneof: [
                      "isLocal",
                      "remote"
                    ]
                  }
                },
                fields: {
                  generic: {
                    type: "GenericActivity",
                    id: 1
                  },
                  delay: {
                    type: "google.protobuf.Duration",
                    id: 2
                  },
                  noop: {
                    type: "google.protobuf.Empty",
                    id: 3
                  },
                  taskQueue: {
                    type: "string",
                    id: 4
                  },
                  headers: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 5
                  },
                  scheduleToCloseTimeout: {
                    type: "google.protobuf.Duration",
                    id: 6
                  },
                  scheduleToStartTimeout: {
                    type: "google.protobuf.Duration",
                    id: 7
                  },
                  startToCloseTimeout: {
                    type: "google.protobuf.Duration",
                    id: 8
                  },
                  heartbeatTimeout: {
                    type: "google.protobuf.Duration",
                    id: 9
                  },
                  retryPolicy: {
                    type: "temporal.api.common.v1.RetryPolicy",
                    id: 10
                  },
                  isLocal: {
                    type: "google.protobuf.Empty",
                    id: 11
                  },
                  remote: {
                    type: "RemoteActivityOptions",
                    id: 12
                  },
                  awaitableChoice: {
                    type: "AwaitableChoice",
                    id: 13
                  }
                },
                nested: {
                  GenericActivity: {
                    fields: {
                      type: {
                        type: "string",
                        id: 1
                      },
                      "arguments": {
                        rule: "repeated",
                        type: "temporal.api.common.v1.Payload",
                        id: 2
                      }
                    }
                  }
                }
              },
              ExecuteChildWorkflowAction: {
                fields: {
                  namespace: {
                    type: "string",
                    id: 2
                  },
                  workflowId: {
                    type: "string",
                    id: 3
                  },
                  workflowType: {
                    type: "string",
                    id: 4
                  },
                  taskQueue: {
                    type: "string",
                    id: 5
                  },
                  input: {
                    rule: "repeated",
                    type: "temporal.api.common.v1.Payload",
                    id: 6
                  },
                  workflowExecutionTimeout: {
                    type: "google.protobuf.Duration",
                    id: 7
                  },
                  workflowRunTimeout: {
                    type: "google.protobuf.Duration",
                    id: 8
                  },
                  workflowTaskTimeout: {
                    type: "google.protobuf.Duration",
                    id: 9
                  },
                  parentClosePolicy: {
                    type: "ParentClosePolicy",
                    id: 10
                  },
                  workflowIdReusePolicy: {
                    type: "temporal.api.enums.v1.WorkflowIdReusePolicy",
                    id: 12
                  },
                  retryPolicy: {
                    type: "temporal.api.common.v1.RetryPolicy",
                    id: 13
                  },
                  cronSchedule: {
                    type: "string",
                    id: 14
                  },
                  headers: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 15
                  },
                  memo: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 16
                  },
                  searchAttributes: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 17
                  },
                  cancellationType: {
                    type: "temporal.omes.kitchen_sink.ChildWorkflowCancellationType",
                    id: 18
                  },
                  versioningIntent: {
                    type: "VersioningIntent",
                    id: 19
                  },
                  awaitableChoice: {
                    type: "AwaitableChoice",
                    id: 20
                  }
                }
              },
              AwaitWorkflowState: {
                fields: {
                  key: {
                    type: "string",
                    id: 1
                  },
                  value: {
                    type: "string",
                    id: 2
                  }
                }
              },
              SendSignalAction: {
                fields: {
                  workflowId: {
                    type: "string",
                    id: 1
                  },
                  runId: {
                    type: "string",
                    id: 2
                  },
                  signalName: {
                    type: "string",
                    id: 3
                  },
                  args: {
                    rule: "repeated",
                    type: "temporal.api.common.v1.Payload",
                    id: 4
                  },
                  headers: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 5
                  },
                  awaitableChoice: {
                    type: "AwaitableChoice",
                    id: 6
                  }
                }
              },
              CancelWorkflowAction: {
                fields: {
                  workflowId: {
                    type: "string",
                    id: 1
                  },
                  runId: {
                    type: "string",
                    id: 2
                  }
                }
              },
              SetPatchMarkerAction: {
                fields: {
                  patchId: {
                    type: "string",
                    id: 1
                  },
                  deprecated: {
                    type: "bool",
                    id: 2
                  },
                  innerAction: {
                    type: "Action",
                    id: 3
                  }
                }
              },
              UpsertSearchAttributesAction: {
                fields: {
                  searchAttributes: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 1
                  }
                }
              },
              UpsertMemoAction: {
                fields: {
                  upsertedMemo: {
                    type: "temporal.api.common.v1.Memo",
                    id: 1
                  }
                }
              },
              ReturnResultAction: {
                fields: {
                  returnThis: {
                    type: "temporal.api.common.v1.Payload",
                    id: 1
                  }
                }
              },
              ReturnErrorAction: {
                fields: {
                  failure: {
                    type: "temporal.api.failure.v1.Failure",
                    id: 1
                  }
                }
              },
              ContinueAsNewAction: {
                fields: {
                  workflowType: {
                    type: "string",
                    id: 1
                  },
                  taskQueue: {
                    type: "string",
                    id: 2
                  },
                  "arguments": {
                    rule: "repeated",
                    type: "temporal.api.common.v1.Payload",
                    id: 3
                  },
                  workflowRunTimeout: {
                    type: "google.protobuf.Duration",
                    id: 4
                  },
                  workflowTaskTimeout: {
                    type: "google.protobuf.Duration",
                    id: 5
                  },
                  memo: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 6
                  },
                  headers: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 7
                  },
                  searchAttributes: {
                    keyType: "string",
                    type: "temporal.api.common.v1.Payload",
                    id: 8
                  },
                  retryPolicy: {
                    type: "temporal.api.common.v1.RetryPolicy",
                    id: 9
                  },
                  versioningIntent: {
                    type: "VersioningIntent",
                    id: 10
                  }
                }
              },
              RemoteActivityOptions: {
                fields: {
                  cancellationType: {
                    type: "temporal.omes.kitchen_sink.ActivityCancellationType",
                    id: 1
                  },
                  doNotEagerlyExecute: {
                    type: "bool",
                    id: 2
                  },
                  versioningIntent: {
                    type: "VersioningIntent",
                    id: 3
                  }
                }
              },
              ParentClosePolicy: {
                values: {
                  PARENT_CLOSE_POLICY_UNSPECIFIED: 0,
                  PARENT_CLOSE_POLICY_TERMINATE: 1,
                  PARENT_CLOSE_POLICY_ABANDON: 2,
                  PARENT_CLOSE_POLICY_REQUEST_CANCEL: 3
                }
              },
              VersioningIntent: {
                values: {
                  UNSPECIFIED: 0,
                  COMPATIBLE: 1,
                  DEFAULT: 2
                }
              },
              ChildWorkflowCancellationType: {
                values: {
                  CHILD_WF_ABANDON: 0,
                  CHILD_WF_TRY_CANCEL: 1,
                  CHILD_WF_WAIT_CANCELLATION_COMPLETED: 2,
                  CHILD_WF_WAIT_CANCELLATION_REQUESTED: 3
                }
              },
              ActivityCancellationType: {
                values: {
                  TRY_CANCEL: 0,
                  WAIT_CANCELLATION_COMPLETED: 1,
                  ABANDON: 2
                }
              }
            }
          }
        }
      },
      api: {
        nested: {
          common: {
            nested: {
              v1: {
                options: {
                  go_package: "go.temporal.io/api/common/v1;common",
                  java_package: "io.temporal.api.common.v1",
                  java_multiple_files: true,
                  java_outer_classname: "MessageProto",
                  ruby_package: "Temporalio::Api::Common::V1",
                  csharp_namespace: "Temporalio.Api.Common.V1"
                },
                nested: {
                  DataBlob: {
                    fields: {
                      encodingType: {
                        type: "temporal.api.enums.v1.EncodingType",
                        id: 1
                      },
                      data: {
                        type: "bytes",
                        id: 2
                      }
                    }
                  },
                  Payloads: {
                    fields: {
                      payloads: {
                        rule: "repeated",
                        type: "Payload",
                        id: 1
                      }
                    }
                  },
                  Payload: {
                    fields: {
                      metadata: {
                        keyType: "string",
                        type: "bytes",
                        id: 1
                      },
                      data: {
                        type: "bytes",
                        id: 2
                      }
                    }
                  },
                  SearchAttributes: {
                    fields: {
                      indexedFields: {
                        keyType: "string",
                        type: "Payload",
                        id: 1
                      }
                    }
                  },
                  Memo: {
                    fields: {
                      fields: {
                        keyType: "string",
                        type: "Payload",
                        id: 1
                      }
                    }
                  },
                  Header: {
                    fields: {
                      fields: {
                        keyType: "string",
                        type: "Payload",
                        id: 1
                      }
                    }
                  },
                  WorkflowExecution: {
                    fields: {
                      workflowId: {
                        type: "string",
                        id: 1
                      },
                      runId: {
                        type: "string",
                        id: 2
                      }
                    }
                  },
                  WorkflowType: {
                    fields: {
                      name: {
                        type: "string",
                        id: 1
                      }
                    }
                  },
                  ActivityType: {
                    fields: {
                      name: {
                        type: "string",
                        id: 1
                      }
                    }
                  },
                  RetryPolicy: {
                    fields: {
                      initialInterval: {
                        type: "google.protobuf.Duration",
                        id: 1,
                        options: {
                          "(gogoproto.stdduration)": true
                        }
                      },
                      backoffCoefficient: {
                        type: "double",
                        id: 2
                      },
                      maximumInterval: {
                        type: "google.protobuf.Duration",
                        id: 3,
                        options: {
                          "(gogoproto.stdduration)": true
                        }
                      },
                      maximumAttempts: {
                        type: "int32",
                        id: 4
                      },
                      nonRetryableErrorTypes: {
                        rule: "repeated",
                        type: "string",
                        id: 5
                      }
                    }
                  },
                  MeteringMetadata: {
                    fields: {
                      nonfirstLocalActivityExecutionAttempts: {
                        type: "uint32",
                        id: 13
                      }
                    }
                  },
                  WorkerVersionStamp: {
                    fields: {
                      buildId: {
                        type: "string",
                        id: 1
                      },
                      bundleId: {
                        type: "string",
                        id: 2
                      },
                      useVersioning: {
                        type: "bool",
                        id: 3
                      }
                    }
                  },
                  WorkerVersionCapabilities: {
                    fields: {
                      buildId: {
                        type: "string",
                        id: 1
                      },
                      useVersioning: {
                        type: "bool",
                        id: 2
                      }
                    }
                  }
                }
              }
            }
          },
          enums: {
            nested: {
              v1: {
                options: {
                  go_package: "go.temporal.io/api/enums/v1;enums",
                  java_package: "io.temporal.api.enums.v1",
                  java_multiple_files: true,
                  java_outer_classname: "WorkflowProto",
                  ruby_package: "Temporalio::Api::Enums::V1",
                  csharp_namespace: "Temporalio.Api.Enums.V1"
                },
                nested: {
                  EncodingType: {
                    values: {
                      ENCODING_TYPE_UNSPECIFIED: 0,
                      ENCODING_TYPE_PROTO3: 1,
                      ENCODING_TYPE_JSON: 2
                    }
                  },
                  IndexedValueType: {
                    values: {
                      INDEXED_VALUE_TYPE_UNSPECIFIED: 0,
                      INDEXED_VALUE_TYPE_TEXT: 1,
                      INDEXED_VALUE_TYPE_KEYWORD: 2,
                      INDEXED_VALUE_TYPE_INT: 3,
                      INDEXED_VALUE_TYPE_DOUBLE: 4,
                      INDEXED_VALUE_TYPE_BOOL: 5,
                      INDEXED_VALUE_TYPE_DATETIME: 6,
                      INDEXED_VALUE_TYPE_KEYWORD_LIST: 7
                    }
                  },
                  Severity: {
                    values: {
                      SEVERITY_UNSPECIFIED: 0,
                      SEVERITY_HIGH: 1,
                      SEVERITY_MEDIUM: 2,
                      SEVERITY_LOW: 3
                    }
                  },
                  WorkflowIdReusePolicy: {
                    values: {
                      WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED: 0,
                      WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE: 1,
                      WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY: 2,
                      WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE: 3,
                      WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING: 4
                    }
                  },
                  ParentClosePolicy: {
                    values: {
                      PARENT_CLOSE_POLICY_UNSPECIFIED: 0,
                      PARENT_CLOSE_POLICY_TERMINATE: 1,
                      PARENT_CLOSE_POLICY_ABANDON: 2,
                      PARENT_CLOSE_POLICY_REQUEST_CANCEL: 3
                    }
                  },
                  ContinueAsNewInitiator: {
                    values: {
                      CONTINUE_AS_NEW_INITIATOR_UNSPECIFIED: 0,
                      CONTINUE_AS_NEW_INITIATOR_WORKFLOW: 1,
                      CONTINUE_AS_NEW_INITIATOR_RETRY: 2,
                      CONTINUE_AS_NEW_INITIATOR_CRON_SCHEDULE: 3
                    }
                  },
                  WorkflowExecutionStatus: {
                    values: {
                      WORKFLOW_EXECUTION_STATUS_UNSPECIFIED: 0,
                      WORKFLOW_EXECUTION_STATUS_RUNNING: 1,
                      WORKFLOW_EXECUTION_STATUS_COMPLETED: 2,
                      WORKFLOW_EXECUTION_STATUS_FAILED: 3,
                      WORKFLOW_EXECUTION_STATUS_CANCELED: 4,
                      WORKFLOW_EXECUTION_STATUS_TERMINATED: 5,
                      WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW: 6,
                      WORKFLOW_EXECUTION_STATUS_TIMED_OUT: 7
                    }
                  },
                  PendingActivityState: {
                    values: {
                      PENDING_ACTIVITY_STATE_UNSPECIFIED: 0,
                      PENDING_ACTIVITY_STATE_SCHEDULED: 1,
                      PENDING_ACTIVITY_STATE_STARTED: 2,
                      PENDING_ACTIVITY_STATE_CANCEL_REQUESTED: 3
                    }
                  },
                  PendingWorkflowTaskState: {
                    values: {
                      PENDING_WORKFLOW_TASK_STATE_UNSPECIFIED: 0,
                      PENDING_WORKFLOW_TASK_STATE_SCHEDULED: 1,
                      PENDING_WORKFLOW_TASK_STATE_STARTED: 2
                    }
                  },
                  HistoryEventFilterType: {
                    values: {
                      HISTORY_EVENT_FILTER_TYPE_UNSPECIFIED: 0,
                      HISTORY_EVENT_FILTER_TYPE_ALL_EVENT: 1,
                      HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT: 2
                    }
                  },
                  RetryState: {
                    values: {
                      RETRY_STATE_UNSPECIFIED: 0,
                      RETRY_STATE_IN_PROGRESS: 1,
                      RETRY_STATE_NON_RETRYABLE_FAILURE: 2,
                      RETRY_STATE_TIMEOUT: 3,
                      RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED: 4,
                      RETRY_STATE_RETRY_POLICY_NOT_SET: 5,
                      RETRY_STATE_INTERNAL_SERVER_ERROR: 6,
                      RETRY_STATE_CANCEL_REQUESTED: 7
                    }
                  },
                  TimeoutType: {
                    values: {
                      TIMEOUT_TYPE_UNSPECIFIED: 0,
                      TIMEOUT_TYPE_START_TO_CLOSE: 1,
                      TIMEOUT_TYPE_SCHEDULE_TO_START: 2,
                      TIMEOUT_TYPE_SCHEDULE_TO_CLOSE: 3,
                      TIMEOUT_TYPE_HEARTBEAT: 4
                    }
                  }
                }
              }
            }
          },
          failure: {
            nested: {
              v1: {
                options: {
                  go_package: "go.temporal.io/api/failure/v1;failure",
                  java_package: "io.temporal.api.failure.v1",
                  java_multiple_files: true,
                  java_outer_classname: "MessageProto",
                  ruby_package: "Temporalio::Api::Failure::V1",
                  csharp_namespace: "Temporalio.Api.Failure.V1"
                },
                nested: {
                  ApplicationFailureInfo: {
                    fields: {
                      type: {
                        type: "string",
                        id: 1
                      },
                      nonRetryable: {
                        type: "bool",
                        id: 2
                      },
                      details: {
                        type: "temporal.api.common.v1.Payloads",
                        id: 3
                      }
                    }
                  },
                  TimeoutFailureInfo: {
                    fields: {
                      timeoutType: {
                        type: "temporal.api.enums.v1.TimeoutType",
                        id: 1
                      },
                      lastHeartbeatDetails: {
                        type: "temporal.api.common.v1.Payloads",
                        id: 2
                      }
                    }
                  },
                  CanceledFailureInfo: {
                    fields: {
                      details: {
                        type: "temporal.api.common.v1.Payloads",
                        id: 1
                      }
                    }
                  },
                  TerminatedFailureInfo: {
                    fields: {}
                  },
                  ServerFailureInfo: {
                    fields: {
                      nonRetryable: {
                        type: "bool",
                        id: 1
                      }
                    }
                  },
                  ResetWorkflowFailureInfo: {
                    fields: {
                      lastHeartbeatDetails: {
                        type: "temporal.api.common.v1.Payloads",
                        id: 1
                      }
                    }
                  },
                  ActivityFailureInfo: {
                    fields: {
                      scheduledEventId: {
                        type: "int64",
                        id: 1
                      },
                      startedEventId: {
                        type: "int64",
                        id: 2
                      },
                      identity: {
                        type: "string",
                        id: 3
                      },
                      activityType: {
                        type: "temporal.api.common.v1.ActivityType",
                        id: 4
                      },
                      activityId: {
                        type: "string",
                        id: 5
                      },
                      retryState: {
                        type: "temporal.api.enums.v1.RetryState",
                        id: 6
                      }
                    }
                  },
                  ChildWorkflowExecutionFailureInfo: {
                    fields: {
                      namespace: {
                        type: "string",
                        id: 1
                      },
                      workflowExecution: {
                        type: "temporal.api.common.v1.WorkflowExecution",
                        id: 2
                      },
                      workflowType: {
                        type: "temporal.api.common.v1.WorkflowType",
                        id: 3
                      },
                      initiatedEventId: {
                        type: "int64",
                        id: 4
                      },
                      startedEventId: {
                        type: "int64",
                        id: 5
                      },
                      retryState: {
                        type: "temporal.api.enums.v1.RetryState",
                        id: 6
                      }
                    }
                  },
                  Failure: {
                    oneofs: {
                      failureInfo: {
                        oneof: [
                          "applicationFailureInfo",
                          "timeoutFailureInfo",
                          "canceledFailureInfo",
                          "terminatedFailureInfo",
                          "serverFailureInfo",
                          "resetWorkflowFailureInfo",
                          "activityFailureInfo",
                          "childWorkflowExecutionFailureInfo"
                        ]
                      }
                    },
                    fields: {
                      message: {
                        type: "string",
                        id: 1
                      },
                      source: {
                        type: "string",
                        id: 2
                      },
                      stackTrace: {
                        type: "string",
                        id: 3
                      },
                      encodedAttributes: {
                        type: "temporal.api.common.v1.Payload",
                        id: 20
                      },
                      cause: {
                        type: "Failure",
                        id: 4
                      },
                      applicationFailureInfo: {
                        type: "ApplicationFailureInfo",
                        id: 5
                      },
                      timeoutFailureInfo: {
                        type: "TimeoutFailureInfo",
                        id: 6
                      },
                      canceledFailureInfo: {
                        type: "CanceledFailureInfo",
                        id: 7
                      },
                      terminatedFailureInfo: {
                        type: "TerminatedFailureInfo",
                        id: 8
                      },
                      serverFailureInfo: {
                        type: "ServerFailureInfo",
                        id: 9
                      },
                      resetWorkflowFailureInfo: {
                        type: "ResetWorkflowFailureInfo",
                        id: 10
                      },
                      activityFailureInfo: {
                        type: "ActivityFailureInfo",
                        id: 11
                      },
                      childWorkflowExecutionFailureInfo: {
                        type: "ChildWorkflowExecutionFailureInfo",
                        id: 12
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  google: {
    nested: {
      protobuf: {
        options: {
          go_package: "google.golang.org/protobuf/types/descriptorpb",
          java_package: "com.google.protobuf",
          java_outer_classname: "DescriptorProtos",
          csharp_namespace: "Google.Protobuf.Reflection",
          objc_class_prefix: "GPB",
          cc_enable_arenas: true,
          optimize_for: "SPEED"
        },
        nested: {
          FileDescriptorSet: {
            fields: {
              file: {
                rule: "repeated",
                type: "FileDescriptorProto",
                id: 1
              }
            }
          },
          FileDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              "package": {
                type: "string",
                id: 2
              },
              dependency: {
                rule: "repeated",
                type: "string",
                id: 3
              },
              publicDependency: {
                rule: "repeated",
                type: "int32",
                id: 10,
                options: {
                  packed: false
                }
              },
              weakDependency: {
                rule: "repeated",
                type: "int32",
                id: 11,
                options: {
                  packed: false
                }
              },
              messageType: {
                rule: "repeated",
                type: "DescriptorProto",
                id: 4
              },
              enumType: {
                rule: "repeated",
                type: "EnumDescriptorProto",
                id: 5
              },
              service: {
                rule: "repeated",
                type: "ServiceDescriptorProto",
                id: 6
              },
              extension: {
                rule: "repeated",
                type: "FieldDescriptorProto",
                id: 7
              },
              options: {
                type: "FileOptions",
                id: 8
              },
              sourceCodeInfo: {
                type: "SourceCodeInfo",
                id: 9
              },
              syntax: {
                type: "string",
                id: 12
              }
            }
          },
          DescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              field: {
                rule: "repeated",
                type: "FieldDescriptorProto",
                id: 2
              },
              extension: {
                rule: "repeated",
                type: "FieldDescriptorProto",
                id: 6
              },
              nestedType: {
                rule: "repeated",
                type: "DescriptorProto",
                id: 3
              },
              enumType: {
                rule: "repeated",
                type: "EnumDescriptorProto",
                id: 4
              },
              extensionRange: {
                rule: "repeated",
                type: "ExtensionRange",
                id: 5
              },
              oneofDecl: {
                rule: "repeated",
                type: "OneofDescriptorProto",
                id: 8
              },
              options: {
                type: "MessageOptions",
                id: 7
              },
              reservedRange: {
                rule: "repeated",
                type: "ReservedRange",
                id: 9
              },
              reservedName: {
                rule: "repeated",
                type: "string",
                id: 10
              }
            },
            nested: {
              ExtensionRange: {
                fields: {
                  start: {
                    type: "int32",
                    id: 1
                  },
                  end: {
                    type: "int32",
                    id: 2
                  }
                }
              },
              ReservedRange: {
                fields: {
                  start: {
                    type: "int32",
                    id: 1
                  },
                  end: {
                    type: "int32",
                    id: 2
                  }
                }
              }
            }
          },
          FieldDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              number: {
                type: "int32",
                id: 3
              },
              label: {
                type: "Label",
                id: 4
              },
              type: {
                type: "Type",
                id: 5
              },
              typeName: {
                type: "string",
                id: 6
              },
              extendee: {
                type: "string",
                id: 2
              },
              defaultValue: {
                type: "string",
                id: 7
              },
              oneofIndex: {
                type: "int32",
                id: 9
              },
              jsonName: {
                type: "string",
                id: 10
              },
              options: {
                type: "FieldOptions",
                id: 8
              }
            },
            nested: {
              Type: {
                values: {
                  TYPE_DOUBLE: 1,
                  TYPE_FLOAT: 2,
                  TYPE_INT64: 3,
                  TYPE_UINT64: 4,
                  TYPE_INT32: 5,
                  TYPE_FIXED64: 6,
                  TYPE_FIXED32: 7,
                  TYPE_BOOL: 8,
                  TYPE_STRING: 9,
                  TYPE_GROUP: 10,
                  TYPE_MESSAGE: 11,
                  TYPE_BYTES: 12,
                  TYPE_UINT32: 13,
                  TYPE_ENUM: 14,
                  TYPE_SFIXED32: 15,
                  TYPE_SFIXED64: 16,
                  TYPE_SINT32: 17,
                  TYPE_SINT64: 18
                }
              },
              Label: {
                values: {
                  LABEL_OPTIONAL: 1,
                  LABEL_REQUIRED: 2,
                  LABEL_REPEATED: 3
                }
              }
            }
          },
          OneofDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              options: {
                type: "OneofOptions",
                id: 2
              }
            }
          },
          EnumDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              value: {
                rule: "repeated",
                type: "EnumValueDescriptorProto",
                id: 2
              },
              options: {
                type: "EnumOptions",
                id: 3
              }
            }
          },
          EnumValueDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              number: {
                type: "int32",
                id: 2
              },
              options: {
                type: "EnumValueOptions",
                id: 3
              }
            }
          },
          ServiceDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              method: {
                rule: "repeated",
                type: "MethodDescriptorProto",
                id: 2
              },
              options: {
                type: "ServiceOptions",
                id: 3
              }
            }
          },
          MethodDescriptorProto: {
            fields: {
              name: {
                type: "string",
                id: 1
              },
              inputType: {
                type: "string",
                id: 2
              },
              outputType: {
                type: "string",
                id: 3
              },
              options: {
                type: "MethodOptions",
                id: 4
              },
              clientStreaming: {
                type: "bool",
                id: 5
              },
              serverStreaming: {
                type: "bool",
                id: 6
              }
            }
          },
          FileOptions: {
            fields: {
              javaPackage: {
                type: "string",
                id: 1
              },
              javaOuterClassname: {
                type: "string",
                id: 8
              },
              javaMultipleFiles: {
                type: "bool",
                id: 10
              },
              javaGenerateEqualsAndHash: {
                type: "bool",
                id: 20,
                options: {
                  deprecated: true
                }
              },
              javaStringCheckUtf8: {
                type: "bool",
                id: 27
              },
              optimizeFor: {
                type: "OptimizeMode",
                id: 9,
                options: {
                  "default": "SPEED"
                }
              },
              goPackage: {
                type: "string",
                id: 11
              },
              ccGenericServices: {
                type: "bool",
                id: 16
              },
              javaGenericServices: {
                type: "bool",
                id: 17
              },
              pyGenericServices: {
                type: "bool",
                id: 18
              },
              deprecated: {
                type: "bool",
                id: 23
              },
              ccEnableArenas: {
                type: "bool",
                id: 31
              },
              objcClassPrefix: {
                type: "string",
                id: 36
              },
              csharpNamespace: {
                type: "string",
                id: 37
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ],
            reserved: [
              [
                38,
                38
              ]
            ],
            nested: {
              OptimizeMode: {
                values: {
                  SPEED: 1,
                  CODE_SIZE: 2,
                  LITE_RUNTIME: 3
                }
              }
            }
          },
          MessageOptions: {
            fields: {
              messageSetWireFormat: {
                type: "bool",
                id: 1
              },
              noStandardDescriptorAccessor: {
                type: "bool",
                id: 2
              },
              deprecated: {
                type: "bool",
                id: 3
              },
              mapEntry: {
                type: "bool",
                id: 7
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ],
            reserved: [
              [
                8,
                8
              ]
            ]
          },
          FieldOptions: {
            fields: {
              ctype: {
                type: "CType",
                id: 1,
                options: {
                  "default": "STRING"
                }
              },
              packed: {
                type: "bool",
                id: 2
              },
              jstype: {
                type: "JSType",
                id: 6,
                options: {
                  "default": "JS_NORMAL"
                }
              },
              lazy: {
                type: "bool",
                id: 5
              },
              deprecated: {
                type: "bool",
                id: 3
              },
              weak: {
                type: "bool",
                id: 10
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ],
            reserved: [
              [
                4,
                4
              ]
            ],
            nested: {
              CType: {
                values: {
                  STRING: 0,
                  CORD: 1,
                  STRING_PIECE: 2
                }
              },
              JSType: {
                values: {
                  JS_NORMAL: 0,
                  JS_STRING: 1,
                  JS_NUMBER: 2
                }
              }
            }
          },
          OneofOptions: {
            fields: {
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ]
          },
          EnumOptions: {
            fields: {
              allowAlias: {
                type: "bool",
                id: 2
              },
              deprecated: {
                type: "bool",
                id: 3
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ]
          },
          EnumValueOptions: {
            fields: {
              deprecated: {
                type: "bool",
                id: 1
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ]
          },
          ServiceOptions: {
            fields: {
              deprecated: {
                type: "bool",
                id: 33
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ]
          },
          MethodOptions: {
            fields: {
              deprecated: {
                type: "bool",
                id: 33
              },
              uninterpretedOption: {
                rule: "repeated",
                type: "UninterpretedOption",
                id: 999
              }
            },
            extensions: [
              [
                1000,
                536870911
              ]
            ]
          },
          UninterpretedOption: {
            fields: {
              name: {
                rule: "repeated",
                type: "NamePart",
                id: 2
              },
              identifierValue: {
                type: "string",
                id: 3
              },
              positiveIntValue: {
                type: "uint64",
                id: 4
              },
              negativeIntValue: {
                type: "int64",
                id: 5
              },
              doubleValue: {
                type: "double",
                id: 6
              },
              stringValue: {
                type: "bytes",
                id: 7
              },
              aggregateValue: {
                type: "string",
                id: 8
              }
            },
            nested: {
              NamePart: {
                fields: {
                  namePart: {
                    rule: "required",
                    type: "string",
                    id: 1
                  },
                  isExtension: {
                    rule: "required",
                    type: "bool",
                    id: 2
                  }
                }
              }
            }
          },
          SourceCodeInfo: {
            fields: {
              location: {
                rule: "repeated",
                type: "Location",
                id: 1
              }
            },
            nested: {
              Location: {
                fields: {
                  path: {
                    rule: "repeated",
                    type: "int32",
                    id: 1
                  },
                  span: {
                    rule: "repeated",
                    type: "int32",
                    id: 2
                  },
                  leadingComments: {
                    type: "string",
                    id: 3
                  },
                  trailingComments: {
                    type: "string",
                    id: 4
                  },
                  leadingDetachedComments: {
                    rule: "repeated",
                    type: "string",
                    id: 6
                  }
                }
              }
            }
          },
          GeneratedCodeInfo: {
            fields: {
              annotation: {
                rule: "repeated",
                type: "Annotation",
                id: 1
              }
            },
            nested: {
              Annotation: {
                fields: {
                  path: {
                    rule: "repeated",
                    type: "int32",
                    id: 1
                  },
                  sourceFile: {
                    type: "string",
                    id: 2
                  },
                  begin: {
                    type: "int32",
                    id: 3
                  },
                  end: {
                    type: "int32",
                    id: 4
                  }
                }
              }
            }
          },
          Duration: {
            fields: {
              seconds: {
                type: "int64",
                id: 1
              },
              nanos: {
                type: "int32",
                id: 2
              }
            }
          },
          Empty: {
            fields: {}
          }
        }
      }
    }
  },
  gogoproto: {
    options: {
      go_package: "github.com/gogo/protobuf/gogoproto"
    },
    nested: {
      goprotoEnumPrefix: {
        type: "bool",
        id: 62001,
        extend: "google.protobuf.EnumOptions"
      },
      goprotoEnumStringer: {
        type: "bool",
        id: 62021,
        extend: "google.protobuf.EnumOptions"
      },
      enumStringer: {
        type: "bool",
        id: 62022,
        extend: "google.protobuf.EnumOptions"
      },
      enumCustomname: {
        type: "string",
        id: 62023,
        extend: "google.protobuf.EnumOptions"
      },
      enumdecl: {
        type: "bool",
        id: 62024,
        extend: "google.protobuf.EnumOptions"
      },
      enumvalueCustomname: {
        type: "string",
        id: 66001,
        extend: "google.protobuf.EnumValueOptions"
      },
      goprotoGettersAll: {
        type: "bool",
        id: 63001,
        extend: "google.protobuf.FileOptions"
      },
      goprotoEnumPrefixAll: {
        type: "bool",
        id: 63002,
        extend: "google.protobuf.FileOptions"
      },
      goprotoStringerAll: {
        type: "bool",
        id: 63003,
        extend: "google.protobuf.FileOptions"
      },
      verboseEqualAll: {
        type: "bool",
        id: 63004,
        extend: "google.protobuf.FileOptions"
      },
      faceAll: {
        type: "bool",
        id: 63005,
        extend: "google.protobuf.FileOptions"
      },
      gostringAll: {
        type: "bool",
        id: 63006,
        extend: "google.protobuf.FileOptions"
      },
      populateAll: {
        type: "bool",
        id: 63007,
        extend: "google.protobuf.FileOptions"
      },
      stringerAll: {
        type: "bool",
        id: 63008,
        extend: "google.protobuf.FileOptions"
      },
      onlyoneAll: {
        type: "bool",
        id: 63009,
        extend: "google.protobuf.FileOptions"
      },
      equalAll: {
        type: "bool",
        id: 63013,
        extend: "google.protobuf.FileOptions"
      },
      descriptionAll: {
        type: "bool",
        id: 63014,
        extend: "google.protobuf.FileOptions"
      },
      testgenAll: {
        type: "bool",
        id: 63015,
        extend: "google.protobuf.FileOptions"
      },
      benchgenAll: {
        type: "bool",
        id: 63016,
        extend: "google.protobuf.FileOptions"
      },
      marshalerAll: {
        type: "bool",
        id: 63017,
        extend: "google.protobuf.FileOptions"
      },
      unmarshalerAll: {
        type: "bool",
        id: 63018,
        extend: "google.protobuf.FileOptions"
      },
      stableMarshalerAll: {
        type: "bool",
        id: 63019,
        extend: "google.protobuf.FileOptions"
      },
      sizerAll: {
        type: "bool",
        id: 63020,
        extend: "google.protobuf.FileOptions"
      },
      goprotoEnumStringerAll: {
        type: "bool",
        id: 63021,
        extend: "google.protobuf.FileOptions"
      },
      enumStringerAll: {
        type: "bool",
        id: 63022,
        extend: "google.protobuf.FileOptions"
      },
      unsafeMarshalerAll: {
        type: "bool",
        id: 63023,
        extend: "google.protobuf.FileOptions"
      },
      unsafeUnmarshalerAll: {
        type: "bool",
        id: 63024,
        extend: "google.protobuf.FileOptions"
      },
      goprotoExtensionsMapAll: {
        type: "bool",
        id: 63025,
        extend: "google.protobuf.FileOptions"
      },
      goprotoUnrecognizedAll: {
        type: "bool",
        id: 63026,
        extend: "google.protobuf.FileOptions"
      },
      gogoprotoImport: {
        type: "bool",
        id: 63027,
        extend: "google.protobuf.FileOptions"
      },
      protosizerAll: {
        type: "bool",
        id: 63028,
        extend: "google.protobuf.FileOptions"
      },
      compareAll: {
        type: "bool",
        id: 63029,
        extend: "google.protobuf.FileOptions"
      },
      typedeclAll: {
        type: "bool",
        id: 63030,
        extend: "google.protobuf.FileOptions"
      },
      enumdeclAll: {
        type: "bool",
        id: 63031,
        extend: "google.protobuf.FileOptions"
      },
      goprotoRegistration: {
        type: "bool",
        id: 63032,
        extend: "google.protobuf.FileOptions"
      },
      messagenameAll: {
        type: "bool",
        id: 63033,
        extend: "google.protobuf.FileOptions"
      },
      goprotoSizecacheAll: {
        type: "bool",
        id: 63034,
        extend: "google.protobuf.FileOptions"
      },
      goprotoUnkeyedAll: {
        type: "bool",
        id: 63035,
        extend: "google.protobuf.FileOptions"
      },
      goprotoGetters: {
        type: "bool",
        id: 64001,
        extend: "google.protobuf.MessageOptions"
      },
      goprotoStringer: {
        type: "bool",
        id: 64003,
        extend: "google.protobuf.MessageOptions"
      },
      verboseEqual: {
        type: "bool",
        id: 64004,
        extend: "google.protobuf.MessageOptions"
      },
      face: {
        type: "bool",
        id: 64005,
        extend: "google.protobuf.MessageOptions"
      },
      gostring: {
        type: "bool",
        id: 64006,
        extend: "google.protobuf.MessageOptions"
      },
      populate: {
        type: "bool",
        id: 64007,
        extend: "google.protobuf.MessageOptions"
      },
      stringer: {
        type: "bool",
        id: 67008,
        extend: "google.protobuf.MessageOptions"
      },
      onlyone: {
        type: "bool",
        id: 64009,
        extend: "google.protobuf.MessageOptions"
      },
      equal: {
        type: "bool",
        id: 64013,
        extend: "google.protobuf.MessageOptions"
      },
      description: {
        type: "bool",
        id: 64014,
        extend: "google.protobuf.MessageOptions"
      },
      testgen: {
        type: "bool",
        id: 64015,
        extend: "google.protobuf.MessageOptions"
      },
      benchgen: {
        type: "bool",
        id: 64016,
        extend: "google.protobuf.MessageOptions"
      },
      marshaler: {
        type: "bool",
        id: 64017,
        extend: "google.protobuf.MessageOptions"
      },
      unmarshaler: {
        type: "bool",
        id: 64018,
        extend: "google.protobuf.MessageOptions"
      },
      stableMarshaler: {
        type: "bool",
        id: 64019,
        extend: "google.protobuf.MessageOptions"
      },
      sizer: {
        type: "bool",
        id: 64020,
        extend: "google.protobuf.MessageOptions"
      },
      unsafeMarshaler: {
        type: "bool",
        id: 64023,
        extend: "google.protobuf.MessageOptions"
      },
      unsafeUnmarshaler: {
        type: "bool",
        id: 64024,
        extend: "google.protobuf.MessageOptions"
      },
      goprotoExtensionsMap: {
        type: "bool",
        id: 64025,
        extend: "google.protobuf.MessageOptions"
      },
      goprotoUnrecognized: {
        type: "bool",
        id: 64026,
        extend: "google.protobuf.MessageOptions"
      },
      protosizer: {
        type: "bool",
        id: 64028,
        extend: "google.protobuf.MessageOptions"
      },
      compare: {
        type: "bool",
        id: 64029,
        extend: "google.protobuf.MessageOptions"
      },
      typedecl: {
        type: "bool",
        id: 64030,
        extend: "google.protobuf.MessageOptions"
      },
      messagename: {
        type: "bool",
        id: 64033,
        extend: "google.protobuf.MessageOptions"
      },
      goprotoSizecache: {
        type: "bool",
        id: 64034,
        extend: "google.protobuf.MessageOptions"
      },
      goprotoUnkeyed: {
        type: "bool",
        id: 64035,
        extend: "google.protobuf.MessageOptions"
      },
      nullable: {
        type: "bool",
        id: 65001,
        extend: "google.protobuf.FieldOptions"
      },
      embed: {
        type: "bool",
        id: 65002,
        extend: "google.protobuf.FieldOptions"
      },
      customtype: {
        type: "string",
        id: 65003,
        extend: "google.protobuf.FieldOptions"
      },
      customname: {
        type: "string",
        id: 65004,
        extend: "google.protobuf.FieldOptions"
      },
      jsontag: {
        type: "string",
        id: 65005,
        extend: "google.protobuf.FieldOptions"
      },
      moretags: {
        type: "string",
        id: 65006,
        extend: "google.protobuf.FieldOptions"
      },
      casttype: {
        type: "string",
        id: 65007,
        extend: "google.protobuf.FieldOptions"
      },
      castkey: {
        type: "string",
        id: 65008,
        extend: "google.protobuf.FieldOptions"
      },
      castvalue: {
        type: "string",
        id: 65009,
        extend: "google.protobuf.FieldOptions"
      },
      stdtime: {
        type: "bool",
        id: 65010,
        extend: "google.protobuf.FieldOptions"
      },
      stdduration: {
        type: "bool",
        id: 65011,
        extend: "google.protobuf.FieldOptions"
      },
      wktpointer: {
        type: "bool",
        id: 65012,
        extend: "google.protobuf.FieldOptions"
      }
    }
  }
});

module.exports = $root;
