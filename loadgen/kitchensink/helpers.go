package kitchensink

import (
	"fmt"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Using human-readable JSON encoding for payloads to aid with debugging.
var jsonPayloadConverter = converter.NewProtoJSONPayloadConverter()

type ActionFactory[T any] func(*T) *Action

func SingleActionSet(actions ...*Action) *ActionSet {
	return &ActionSet{
		Actions: actions,
	}
}

func ListActionSet(actions ...*Action) []*ActionSet {
	return []*ActionSet{
		{
			Actions: actions,
		},
	}
}

func ClientActions(clientActions ...*ClientAction) *ClientSequence {
	return &ClientSequence{
		ActionSets: []*ClientActionSet{
			{
				Actions: clientActions,
			},
		},
	}
}

func ClientActivity(clientSeq *ClientSequence, factory ActionFactory[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Client{
			Client: &ExecuteActivityAction_ClientActivity{
				ClientSequence: clientSeq,
			},
		},
	}
	return factory(activity)
}

func NoOpSingleActivityActionSet() *ActionSet {
	return &ActionSet{
		Actions: []*Action{
			{
				Variant: &Action_ExecActivity{
					ExecActivity: &ExecuteActivityAction{
						ActivityType:        &ExecuteActivityAction_Noop{},
						StartToCloseTimeout: &durationpb.Duration{Seconds: 5},
					},
				},
			},
			{
				Variant: &Action_ReturnResult{
					ReturnResult: &ReturnResultAction{
						ReturnThis: &common.Payload{},
					},
				},
			},
		},
	}
}

func ResourceConsumingActivity(bytesToAllocate uint64, cpuYieldEveryNIters uint32, cpuYieldForMs uint32, runForSeconds int64) *Action {
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: &ExecuteActivityAction{
				ActivityType: &ExecuteActivityAction_Resources{
					Resources: &ExecuteActivityAction_ResourcesActivity{
						BytesToAllocate:          bytesToAllocate,
						CpuYieldEveryNIterations: cpuYieldEveryNIters,
						CpuYieldForMs:            cpuYieldForMs,
						RunFor:                   &durationpb.Duration{Seconds: runForSeconds},
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: runForSeconds * 2},
				RetryPolicy: &common.RetryPolicy{
					MaximumAttempts:    1,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

func NewEmptyReturnResultAction() *Action {
	return &Action{
		Variant: &Action_ReturnResult{
			ReturnResult: &ReturnResultAction{
				ReturnThis: &common.Payload{},
			},
		},
	}
}

func NewTimerAction(t time.Duration) *Action {
	return &Action{
		Variant: &Action_Timer{
			Timer: &TimerAction{
				Milliseconds: uint64(t.Milliseconds()),
			},
		},
	}
}

func PayloadActivity(inSize, outSize int, factory ActionFactory[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Payload{
			Payload: &ExecuteActivityAction_PayloadActivity{
				BytesToReceive: int32(inSize),
				BytesToReturn:  int32(outSize),
			},
		},
	}
	return factory(activity)
}

func GenericActivity(activityType string, factory ActionFactory[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Generic{
			Generic: &ExecuteActivityAction_GenericActivity{
				Type: activityType,
			},
		},
	}
	return factory(activity)
}

func DefaultRemoteActivity(activity *ExecuteActivityAction) *Action {
	activity.StartToCloseTimeout = &durationpb.Duration{Seconds: 60}
	activity.Locality = &ExecuteActivityAction_Remote{
		Remote: &RemoteActivityOptions{},
	}
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: activity,
		},
	}
}

func DefaultLocalActivity(activity *ExecuteActivityAction) *Action {
	activity.StartToCloseTimeout = &durationpb.Duration{Seconds: 60}
	activity.Locality = &ExecuteActivityAction_IsLocal{
		IsLocal: &emptypb.Empty{},
	}
	activity.RetryPolicy = &common.RetryPolicy{
		InitialInterval:    durationpb.New(10 * time.Millisecond),
		MaximumAttempts:    10,
		BackoffCoefficient: 2.0,
	}
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: activity,
		},
	}
}

func NewSetWorkflowStateAction(key, value string) *Action {
	return &Action{
		Variant: &Action_SetWorkflowState{
			SetWorkflowState: &WorkflowState{
				Kvs: map[string]string{key: value},
			},
		},
	}
}

func NewAwaitWorkflowStateAction(key, value string) *Action {
	return &Action{
		Variant: &Action_AwaitWorkflowState{
			AwaitWorkflowState: &AwaitWorkflowState{
				Key:   key,
				Value: value,
			},
		},
	}
}

func NewSignalActionsWithIDs(ids ...int32) []*ClientAction {
	actions := make([]*ClientAction, len(ids))
	for i, id := range ids {
		actions[i] = &ClientAction{
			Variant: &ClientAction_DoSignal{
				DoSignal: &DoSignal{
					Variant: &DoSignal_DoSignalActions_{
						DoSignalActions: &DoSignal_DoSignalActions{
							SignalId: id,
							Variant: &DoSignal_DoSignalActions_DoActions{
								DoActions: SingleActionSet(
									NewSetWorkflowStateAction(fmt.Sprintf("signal_%d", i), "received"),
								),
							},
						},
					},
				},
			},
		}
	}
	return actions
}

func ConvertToPayload(newInput any) *common.Payload {
	payload, err := jsonPayloadConverter.ToPayload(newInput)
	if err != nil {
		// this should never happen; but we don't want to swallow the error
		panic(fmt.Sprintf("failed to convert input %T to payload: %v", newInput, err))
	}
	return payload
}
