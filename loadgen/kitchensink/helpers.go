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

type ActionConverter[T any] func(*T) *Action

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

func ClientActivity(clientSequence *ClientSequence) *ExecuteActivityAction_Client {
	return &ExecuteActivityAction_Client{
		Client: &ExecuteActivityAction_ClientActivity{
			ClientSequence: clientSequence,
		},
	}
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

func NewTimerAction(milliseconds uint64) *Action {
	return &Action{
		Variant: &Action_Timer{
			Timer: &TimerAction{
				Milliseconds: milliseconds,
			},
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

func ConvertToPayload(newInput any) *common.Payload {
	payload, err := jsonPayloadConverter.ToPayload(newInput)
	if err != nil {
		// this should never happen; but we don't want to swallow the error
		panic(fmt.Sprintf("failed to convert input %T to payload: %v", newInput, err))
	}
	return payload
}

func PayloadActivity(inSize, outSize int, converter ActionConverter[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Payload{
			Payload: &ExecuteActivityAction_PayloadActivity{
				BytesToReceive: int32(inSize),
				BytesToReturn:  int32(outSize),
			},
		},
	}
	return converter(activity)
}

func NoopActivity(converter ActionConverter[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Noop{},
	}
	return converter(activity)
}

func DelayActivity(delay time.Duration, converter ActionConverter[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Delay{
			Delay: durationpb.New(delay),
		},
	}
	return converter(activity)
}

func GenericActivity(activityType string, converter ActionConverter[ExecuteActivityAction]) *Action {
	activity := &ExecuteActivityAction{
		ActivityType: &ExecuteActivityAction_Generic{
			Generic: &ExecuteActivityAction_GenericActivity{
				Type: activityType,
			},
		},
	}
	return converter(activity)
}

// AsRemoteActivityAction configures an activity as remote and converts it to an Action
func AsRemoteActivityAction(activity *ExecuteActivityAction) *Action {
	activity.StartToCloseTimeout = &durationpb.Duration{Seconds: 60}
	activity.Locality = &ExecuteActivityAction_Remote{
		Remote: &RemoteActivityOptions{},
	}
	activity.RetryPolicy = &common.RetryPolicy{
		MaximumAttempts:    10,
		BackoffCoefficient: 1.0,
	}
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: activity,
		},
	}
}

// AsLocalActivityAction configures an activity as local and converts it to an Action
func AsLocalActivityAction(activity *ExecuteActivityAction) *Action {
	activity.StartToCloseTimeout = &durationpb.Duration{Seconds: 60}
	activity.Locality = &ExecuteActivityAction_IsLocal{
		IsLocal: &emptypb.Empty{},
	}
	activity.RetryPolicy = &common.RetryPolicy{
		InitialInterval: durationpb.New(10 * time.Millisecond),
		MaximumAttempts: 10,
	}
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: activity,
		},
	}
}
