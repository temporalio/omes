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
                  },
                  withStartAction: {
                    type: "WithStartClientAction",
                    id: 3
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
              WithStartClientAction: {
                oneofs: {
                  variant: {
                    oneof: [
                      "doSignal",
                      "doUpdate"
                    ]
                  }
                },
                fields: {
                  doSignal: {
                    type: "DoSignal",
                    id: 1
                  },
                  doUpdate: {
                    type: "DoUpdate",
                    id: 2
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
                      "nestedActions",
                      "doDescribe",
                      "doStandaloneActivity"
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
                  },
                  doDescribe: {
                    type: "DoDescribe",
                    id: 5
                  },
                  doStandaloneActivity: {
                    type: "DoStandaloneActivity",
                    id: 6
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
                  },
                  withStart: {
                    type: "bool",
                    id: 3
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
                      },
                      signalId: {
                        type: "int32",
                        id: 3
                      }
                    }
                  }
                }
              },
              DoDescribe: {
                fields: {}
              },
              DoStandaloneActivity: {
                fields: {
                  activity: {
                    type: "ExecuteActivityAction",
                    id: 1
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
                  withStart: {
                    type: "bool",
                    id: 3
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
                  },
                  expectedSignalCount: {
                    type: "int32",
                    id: 2
                  },
                  expectedSignalIds: {
                    rule: "repeated",
                    type: "int32",
                    id: 3
                  },
                  receivedSignalIds: {
                    rule: "repeated",
                    type: "int32",
                    id: 4
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
                      "nestedActionSet",
                      "nexusOperation"
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
                  },
                  nexusOperation: {
                    type: "ExecuteNexusOperation",
                    id: 15
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
                      "noop",
                      "resources",
                      "payload",
                      "client",
                      "retryableError",
                      "timeout",
                      "heartbeat"
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
                  resources: {
                    type: "ResourcesActivity",
                    id: 14
                  },
                  payload: {
                    type: "PayloadActivity",
                    id: 18
                  },
                  client: {
                    type: "ClientActivity",
                    id: 19
                  },
                  retryableError: {
                    type: "RetryableErrorActivity",
                    id: 20
                  },
                  timeout: {
                    type: "TimeoutActivity",
                    id: 21
                  },
                  heartbeat: {
                    type: "HeartbeatTimeoutActivity",
                    id: 22
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
                  },
                  priority: {
                    type: "temporal.api.common.v1.Priority",
                    id: 15
                  },
                  fairnessKey: {
                    type: "string",
                    id: 16
                  },
                  fairnessWeight: {
                    type: "float",
                    id: 17
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
                  },
                  ResourcesActivity: {
                    fields: {
                      runFor: {
                        type: "google.protobuf.Duration",
                        id: 1
                      },
                      bytesToAllocate: {
                        type: "uint64",
                        id: 2
                      },
                      cpuYieldEveryNIterations: {
                        type: "uint32",
                        id: 3
                      },
                      cpuYieldForMs: {
                        type: "uint32",
                        id: 4
                      }
                    }
                  },
                  PayloadActivity: {
                    fields: {
                      bytesToReceive: {
                        type: "int32",
                        id: 1
                      },
                      bytesToReturn: {
                        type: "int32",
                        id: 2
                      }
                    }
                  },
                  ClientActivity: {
                    fields: {
                      clientSequence: {
                        type: "ClientSequence",
                        id: 1
                      }
                    }
                  },
                  RetryableErrorActivity: {
                    fields: {
                      failAttempts: {
                        type: "int32",
                        id: 1
                      }
                    }
                  },
                  TimeoutActivity: {
                    fields: {
                      failAttempts: {
                        type: "int32",
                        id: 1
                      },
                      successDuration: {
                        type: "google.protobuf.Duration",
                        id: 2
                      },
                      failureDuration: {
                        type: "google.protobuf.Duration",
                        id: 3
                      }
                    }
                  },
                  HeartbeatTimeoutActivity: {
                    fields: {
                      failAttempts: {
                        type: "int32",
                        id: 1
                      },
                      successDuration: {
                        type: "google.protobuf.Duration",
                        id: 2
                      },
                      failureDuration: {
                        type: "google.protobuf.Duration",
                        id: 3
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
              },
              ExecuteNexusOperation: {
                fields: {
                  endpoint: {
                    type: "string",
                    id: 1
                  },
                  operation: {
                    type: "string",
                    id: 2
                  },
                  input: {
                    type: "string",
                    id: 3
                  },
                  headers: {
                    keyType: "string",
                    type: "string",
                    id: 4
                  },
                  awaitableChoice: {
                    type: "AwaitableChoice",
                    id: 5
                  },
                  expectedOutput: {
                    type: "string",
                    id: 6
                  },
                  beforeActions: {
                    rule: "repeated",
                    type: "ActionSet",
                    id: 7
                  }
                }
              },
              NexusHandlerInput: {
                fields: {
                  input: {
                    type: "string",
                    id: 1
                  },
                  beforeActions: {
                    rule: "repeated",
                    type: "ActionSet",
                    id: 2
                  }
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
                        id: 1
                      },
                      backoffCoefficient: {
                        type: "double",
                        id: 2
                      },
                      maximumInterval: {
                        type: "google.protobuf.Duration",
                        id: 3
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
                      },
                      deploymentSeriesName: {
                        type: "string",
                        id: 4
                      }
                    }
                  },
                  ResetOptions: {
                    oneofs: {
                      target: {
                        oneof: [
                          "firstWorkflowTask",
                          "lastWorkflowTask",
                          "workflowTaskId",
                          "buildId"
                        ]
                      }
                    },
                    fields: {
                      firstWorkflowTask: {
                        type: "google.protobuf.Empty",
                        id: 1
                      },
                      lastWorkflowTask: {
                        type: "google.protobuf.Empty",
                        id: 2
                      },
                      workflowTaskId: {
                        type: "int64",
                        id: 3
                      },
                      buildId: {
                        type: "string",
                        id: 4
                      },
                      resetReapplyType: {
                        type: "temporal.api.enums.v1.ResetReapplyType",
                        id: 10,
                        options: {
                          deprecated: true
                        }
                      },
                      currentRunOnly: {
                        type: "bool",
                        id: 11
                      },
                      resetReapplyExcludeTypes: {
                        rule: "repeated",
                        type: "temporal.api.enums.v1.ResetReapplyExcludeType",
                        id: 12
                      }
                    }
                  },
                  Callback: {
                    oneofs: {
                      variant: {
                        oneof: [
                          "nexus",
                          "internal"
                        ]
                      }
                    },
                    fields: {
                      nexus: {
                        type: "Nexus",
                        id: 2
                      },
                      internal: {
                        type: "Internal",
                        id: 3
                      },
                      links: {
                        rule: "repeated",
                        type: "Link",
                        id: 100
                      }
                    },
                    reserved: [
                      [
                        1,
                        1
                      ]
                    ],
                    nested: {
                      Nexus: {
                        fields: {
                          url: {
                            type: "string",
                            id: 1
                          },
                          header: {
                            keyType: "string",
                            type: "string",
                            id: 2
                          }
                        }
                      },
                      Internal: {
                        fields: {
                          data: {
                            type: "bytes",
                            id: 1
                          }
                        }
                      }
                    }
                  },
                  Link: {
                    oneofs: {
                      variant: {
                        oneof: [
                          "workflowEvent",
                          "batchJob"
                        ]
                      }
                    },
                    fields: {
                      workflowEvent: {
                        type: "WorkflowEvent",
                        id: 1
                      },
                      batchJob: {
                        type: "BatchJob",
                        id: 2
                      }
                    },
                    nested: {
                      WorkflowEvent: {
                        oneofs: {
                          reference: {
                            oneof: [
                              "eventRef",
                              "requestIdRef"
                            ]
                          }
                        },
                        fields: {
                          namespace: {
                            type: "string",
                            id: 1
                          },
                          workflowId: {
                            type: "string",
                            id: 2
                          },
                          runId: {
                            type: "string",
                            id: 3
                          },
                          eventRef: {
                            type: "EventReference",
                            id: 100
                          },
                          requestIdRef: {
                            type: "RequestIdReference",
                            id: 101
                          }
                        },
                        nested: {
                          EventReference: {
                            fields: {
                              eventId: {
                                type: "int64",
                                id: 1
                              },
                              eventType: {
                                type: "temporal.api.enums.v1.EventType",
                                id: 2
                              }
                            }
                          },
                          RequestIdReference: {
                            fields: {
                              requestId: {
                                type: "string",
                                id: 1
                              },
                              eventType: {
                                type: "temporal.api.enums.v1.EventType",
                                id: 2
                              }
                            }
                          }
                        }
                      },
                      BatchJob: {
                        fields: {
                          jobId: {
                            type: "string",
                            id: 1
                          }
                        }
                      }
                    }
                  },
                  Priority: {
                    fields: {
                      priorityKey: {
                        type: "int32",
                        id: 1
                      },
                      fairnessKey: {
                        type: "string",
                        id: 2
                      },
                      fairnessWeight: {
                        type: "float",
                        id: 3
                      }
                    }
                  },
                  WorkerSelector: {
                    oneofs: {
                      selector: {
                        oneof: [
                          "workerInstanceKey"
                        ]
                      }
                    },
                    fields: {
                      workerInstanceKey: {
                        type: "string",
                        id: 1
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
                  java_outer_classname: "NexusProto",
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
                  CallbackState: {
                    values: {
                      CALLBACK_STATE_UNSPECIFIED: 0,
                      CALLBACK_STATE_STANDBY: 1,
                      CALLBACK_STATE_SCHEDULED: 2,
                      CALLBACK_STATE_BACKING_OFF: 3,
                      CALLBACK_STATE_FAILED: 4,
                      CALLBACK_STATE_SUCCEEDED: 5,
                      CALLBACK_STATE_BLOCKED: 6
                    }
                  },
                  PendingNexusOperationState: {
                    values: {
                      PENDING_NEXUS_OPERATION_STATE_UNSPECIFIED: 0,
                      PENDING_NEXUS_OPERATION_STATE_SCHEDULED: 1,
                      PENDING_NEXUS_OPERATION_STATE_BACKING_OFF: 2,
                      PENDING_NEXUS_OPERATION_STATE_STARTED: 3,
                      PENDING_NEXUS_OPERATION_STATE_BLOCKED: 4
                    }
                  },
                  NexusOperationCancellationState: {
                    values: {
                      NEXUS_OPERATION_CANCELLATION_STATE_UNSPECIFIED: 0,
                      NEXUS_OPERATION_CANCELLATION_STATE_SCHEDULED: 1,
                      NEXUS_OPERATION_CANCELLATION_STATE_BACKING_OFF: 2,
                      NEXUS_OPERATION_CANCELLATION_STATE_SUCCEEDED: 3,
                      NEXUS_OPERATION_CANCELLATION_STATE_FAILED: 4,
                      NEXUS_OPERATION_CANCELLATION_STATE_TIMED_OUT: 5,
                      NEXUS_OPERATION_CANCELLATION_STATE_BLOCKED: 6
                    }
                  },
                  WorkflowRuleActionScope: {
                    values: {
                      WORKFLOW_RULE_ACTION_SCOPE_UNSPECIFIED: 0,
                      WORKFLOW_RULE_ACTION_SCOPE_WORKFLOW: 1,
                      WORKFLOW_RULE_ACTION_SCOPE_ACTIVITY: 2
                    }
                  },
                  ApplicationErrorCategory: {
                    values: {
                      APPLICATION_ERROR_CATEGORY_UNSPECIFIED: 0,
                      APPLICATION_ERROR_CATEGORY_BENIGN: 1
                    }
                  },
                  WorkerStatus: {
                    values: {
                      WORKER_STATUS_UNSPECIFIED: 0,
                      WORKER_STATUS_RUNNING: 1,
                      WORKER_STATUS_SHUTTING_DOWN: 2,
                      WORKER_STATUS_SHUTDOWN: 3
                    }
                  },
                  EventType: {
                    values: {
                      EVENT_TYPE_UNSPECIFIED: 0,
                      EVENT_TYPE_WORKFLOW_EXECUTION_STARTED: 1,
                      EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED: 2,
                      EVENT_TYPE_WORKFLOW_EXECUTION_FAILED: 3,
                      EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT: 4,
                      EVENT_TYPE_WORKFLOW_TASK_SCHEDULED: 5,
                      EVENT_TYPE_WORKFLOW_TASK_STARTED: 6,
                      EVENT_TYPE_WORKFLOW_TASK_COMPLETED: 7,
                      EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT: 8,
                      EVENT_TYPE_WORKFLOW_TASK_FAILED: 9,
                      EVENT_TYPE_ACTIVITY_TASK_SCHEDULED: 10,
                      EVENT_TYPE_ACTIVITY_TASK_STARTED: 11,
                      EVENT_TYPE_ACTIVITY_TASK_COMPLETED: 12,
                      EVENT_TYPE_ACTIVITY_TASK_FAILED: 13,
                      EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT: 14,
                      EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED: 15,
                      EVENT_TYPE_ACTIVITY_TASK_CANCELED: 16,
                      EVENT_TYPE_TIMER_STARTED: 17,
                      EVENT_TYPE_TIMER_FIRED: 18,
                      EVENT_TYPE_TIMER_CANCELED: 19,
                      EVENT_TYPE_WORKFLOW_EXECUTION_CANCEL_REQUESTED: 20,
                      EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED: 21,
                      EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED: 22,
                      EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED: 23,
                      EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_CANCEL_REQUESTED: 24,
                      EVENT_TYPE_MARKER_RECORDED: 25,
                      EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED: 26,
                      EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED: 27,
                      EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW: 28,
                      EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED: 29,
                      EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED: 30,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED: 31,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED: 32,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED: 33,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED: 34,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT: 35,
                      EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED: 36,
                      EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED: 37,
                      EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED: 38,
                      EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_SIGNALED: 39,
                      EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES: 40,
                      EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ADMITTED: 47,
                      EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED: 41,
                      EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_REJECTED: 42,
                      EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED: 43,
                      EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED_EXTERNALLY: 44,
                      EVENT_TYPE_ACTIVITY_PROPERTIES_MODIFIED_EXTERNALLY: 45,
                      EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED: 46,
                      EVENT_TYPE_NEXUS_OPERATION_SCHEDULED: 48,
                      EVENT_TYPE_NEXUS_OPERATION_STARTED: 49,
                      EVENT_TYPE_NEXUS_OPERATION_COMPLETED: 50,
                      EVENT_TYPE_NEXUS_OPERATION_FAILED: 51,
                      EVENT_TYPE_NEXUS_OPERATION_CANCELED: 52,
                      EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT: 53,
                      EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED: 54,
                      EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED: 55,
                      EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED: 56,
                      EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_FAILED: 57
                    }
                  },
                  ResetReapplyExcludeType: {
                    valuesOptions: {
                      RESET_REAPPLY_EXCLUDE_TYPE_CANCEL_REQUEST: {
                        deprecated: true
                      }
                    },
                    values: {
                      RESET_REAPPLY_EXCLUDE_TYPE_UNSPECIFIED: 0,
                      RESET_REAPPLY_EXCLUDE_TYPE_SIGNAL: 1,
                      RESET_REAPPLY_EXCLUDE_TYPE_UPDATE: 2,
                      RESET_REAPPLY_EXCLUDE_TYPE_NEXUS: 3,
                      RESET_REAPPLY_EXCLUDE_TYPE_CANCEL_REQUEST: 4
                    }
                  },
                  ResetReapplyType: {
                    values: {
                      RESET_REAPPLY_TYPE_UNSPECIFIED: 0,
                      RESET_REAPPLY_TYPE_SIGNAL: 1,
                      RESET_REAPPLY_TYPE_NONE: 2,
                      RESET_REAPPLY_TYPE_ALL_ELIGIBLE: 3
                    }
                  },
                  ResetType: {
                    values: {
                      RESET_TYPE_UNSPECIFIED: 0,
                      RESET_TYPE_FIRST_WORKFLOW_TASK: 1,
                      RESET_TYPE_LAST_WORKFLOW_TASK: 2
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
                  WorkflowIdConflictPolicy: {
                    values: {
                      WORKFLOW_ID_CONFLICT_POLICY_UNSPECIFIED: 0,
                      WORKFLOW_ID_CONFLICT_POLICY_FAIL: 1,
                      WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING: 2,
                      WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING: 3
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
                      PENDING_ACTIVITY_STATE_CANCEL_REQUESTED: 3,
                      PENDING_ACTIVITY_STATE_PAUSED: 4,
                      PENDING_ACTIVITY_STATE_PAUSE_REQUESTED: 5
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
                  },
                  VersioningBehavior: {
                    values: {
                      VERSIONING_BEHAVIOR_UNSPECIFIED: 0,
                      VERSIONING_BEHAVIOR_PINNED: 1,
                      VERSIONING_BEHAVIOR_AUTO_UPGRADE: 2
                    }
                  },
                  NexusHandlerErrorRetryBehavior: {
                    values: {
                      NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_UNSPECIFIED: 0,
                      NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE: 1,
                      NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE: 2
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
                      },
                      nextRetryDelay: {
                        type: "google.protobuf.Duration",
                        id: 4
                      },
                      category: {
                        type: "temporal.api.enums.v1.ApplicationErrorCategory",
                        id: 5
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
                  NexusOperationFailureInfo: {
                    fields: {
                      scheduledEventId: {
                        type: "int64",
                        id: 1
                      },
                      endpoint: {
                        type: "string",
                        id: 2
                      },
                      service: {
                        type: "string",
                        id: 3
                      },
                      operation: {
                        type: "string",
                        id: 4
                      },
                      operationId: {
                        type: "string",
                        id: 5,
                        options: {
                          deprecated: true
                        }
                      },
                      operationToken: {
                        type: "string",
                        id: 6
                      }
                    }
                  },
                  NexusHandlerFailureInfo: {
                    fields: {
                      type: {
                        type: "string",
                        id: 1
                      },
                      retryBehavior: {
                        type: "temporal.api.enums.v1.NexusHandlerErrorRetryBehavior",
                        id: 2
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
                          "childWorkflowExecutionFailureInfo",
                          "nexusOperationExecutionFailureInfo",
                          "nexusHandlerFailureInfo"
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
                      },
                      nexusOperationExecutionFailureInfo: {
                        type: "NexusOperationFailureInfo",
                        id: 13
                      },
                      nexusHandlerFailureInfo: {
                        type: "NexusHandlerFailureInfo",
                        id: 14
                      }
                    }
                  },
                  MultiOperationExecutionAborted: {
                    fields: {}
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
        nested: {
          FileDescriptorSet: {
            edition: "proto2",
            fields: {
              file: {
                rule: "repeated",
                type: "FileDescriptorProto",
                id: 1
              }
            }
          },
          FileDescriptorProto: {
            edition: "proto2",
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
                id: 10
              },
              weakDependency: {
                rule: "repeated",
                type: "int32",
                id: 11
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
            edition: "proto2",
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
                    id: 1,
                    options: {
                      packed: true
                    }
                  },
                  span: {
                    rule: "repeated",
                    type: "int32",
                    id: 2,
                    options: {
                      packed: true
                    }
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
            edition: "proto2",
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
                    id: 1,
                    options: {
                      packed: true
                    }
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
  }
});

module.exports = $root;
