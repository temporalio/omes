package kitchensink

import (
	"fmt"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Using human-readable JSON encoding for payloads to aid with debugging.
var jsonPayloadConverter = converter.NewProtoJSONPayloadConverter()

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

func NewTimerAction(t time.Duration) *Action {
	return &Action{
		Variant: &Action_Timer{
			Timer: &TimerAction{
				Milliseconds: uint64(t.Milliseconds()),
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
