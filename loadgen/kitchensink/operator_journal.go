package kitchensink

import (
	"context"
	"errors"
	"time"

	"github.com/temporalio/omes/loadgen/journal"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	sdkpb "go.temporal.io/api/sdk/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type operatorPhaseKey struct{}
type operatorPhase struct {
	session *journal.Session
	name    string
}

func withOperatorPhase(ctx context.Context, name string) context.Context {
	if p, ok := ctx.Value(operatorPhaseKey{}).(operatorPhase); ok {
		p.name = name
		return context.WithValue(ctx, operatorPhaseKey{}, p)
	}
	return ctx
}

type journalOperatorService struct {
	workflowservicepb.WorkflowServiceClient
	session *journal.Session
}

func (r *journalOperatorService) mutate(ctx context.Context, key string, req, resp proto.Message, safe bool, call func() error) error {
	item, err := r.session.Plan(ctx, "operator/"+key, time.Time{}, func() ([]byte, error) { return proto.Marshal(req) })
	if err != nil {
		return err
	}
	if item.Unresolved {
		return journal.ErrUnresolved
	}
	if item.Done {
		return proto.Unmarshal(item.Observation, resp)
	}
	if item.Calls > 0 && !safe {
		recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := r.session.ObserveStatus(recordCtx, "operator/"+key, nil, true); err != nil {
			return err
		}
		return journal.ErrUnresolved
	}
	if err := proto.Unmarshal(item.Plan, req); err != nil {
		return err
	}
	if err := r.session.Issue(ctx, "operator/"+key); err != nil {
		return err
	}
	if err := call(); err != nil {
		if !safe {
			recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if recordErr := r.session.ObserveStatus(recordCtx, "operator/"+key, nil, true); recordErr != nil {
				return errors.Join(err, recordErr)
			}
		}
		return err
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	return r.session.Observe(ctx, "operator/"+key, raw)
}

func (e *ClientActionsExecutor) startJournaledOperator(ctx context.Context, service workflowservicepb.WorkflowServiceClient, act *ExecuteActivityAction, activityID, activityType string, args []any) (client.ActivityHandle, error) {
	payloads, err := converter.GetDefaultDataConverter().ToPayloads(args...)
	if err != nil {
		return nil, err
	}
	details, err := converter.GetDefaultDataConverter().ToPayload("omes.operator/" + activityID)
	if err != nil {
		return nil, err
	}
	request := &workflowservicepb.StartActivityExecutionRequest{
		Namespace: e.Namespace, ActivityId: activityID, RequestId: activityID,
		ActivityType: &commonpb.ActivityType{Name: activityType}, TaskQueue: &taskqueuepb.TaskQueue{Name: act.TaskQueue}, Input: payloads,
		ScheduleToCloseTimeout: act.ScheduleToCloseTimeout, StartToCloseTimeout: act.StartToCloseTimeout, HeartbeatTimeout: act.HeartbeatTimeout,
		RetryPolicy: act.RetryPolicy, IdReusePolicy: enumspb.ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE, IdConflictPolicy: enumspb.ACTIVITY_ID_CONFLICT_POLICY_FAIL,
		UserMetadata: &sdkpb.UserMetadata{Details: details},
	}
	response := &workflowservicepb.StartActivityExecutionResponse{}
	wrapper := service.(*journalOperatorService)
	err = wrapper.mutate(ctx, "start", request, response, true, func() error {
		result, err := wrapper.WorkflowServiceClient.StartActivityExecution(ctx, request)
		if err == nil {
			proto.Merge(response, result)
			return nil
		}
		var exists *serviceerror.ActivityExecutionAlreadyStarted
		if !errors.As(err, &exists) || exists.RunId == "" {
			return err
		}
		observed, err := wrapper.WorkflowServiceClient.DescribeActivityExecution(ctx, &workflowservicepb.DescribeActivityExecutionRequest{Namespace: request.Namespace, ActivityId: request.ActivityId, RunId: exists.RunId})
		if err != nil {
			return err
		}
		info := observed.GetInfo()
		var intent string
		if info == nil || info.GetActivityType().GetName() != request.ActivityType.Name || info.GetTaskQueue() != request.TaskQueue.GetName() || converter.GetDefaultDataConverter().FromPayload(info.GetUserMetadata().GetDetails(), &intent) != nil || intent != "omes.operator/"+request.ActivityId {
			return journal.ErrConflict
		}
		response.RunId = exists.RunId
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response.RunId == "" {
		return nil, errors.New("journaled activity start returned no run ID")
	}
	return e.Client.GetActivityHandle(client.GetActivityHandleOptions{ActivityID: request.ActivityId, RunID: response.RunId}), nil
}

func recordedOperatorState(ctx context.Context, phase operatorPhase) (*activitypb.ActivityExecutionInfo, bool, error) {
	item, err := phase.session.Plan(ctx, "operator/state/"+phase.name, time.Time{}, func() ([]byte, error) { return []byte(phase.name), nil })
	if err != nil {
		return nil, false, err
	}
	if !item.Done {
		return nil, false, nil
	}
	result := &activitypb.ActivityExecutionInfo{}
	if err := proto.Unmarshal(item.Observation, result); err != nil {
		return nil, false, err
	}
	return result, true, nil
}

func (r *journalOperatorService) PauseActivityExecution(ctx context.Context, req *workflowservicepb.PauseActivityExecutionRequest, opts ...grpc.CallOption) (*workflowservicepb.PauseActivityExecutionResponse, error) {
	out := &workflowservicepb.PauseActivityExecutionResponse{}
	err := r.mutate(ctx, "pause", req, out, false, func() error {
		result, err := r.WorkflowServiceClient.PauseActivityExecution(ctx, req, opts...)
		if err == nil {
			proto.Merge(out, result)
		}
		return err
	})
	return out, err
}
func (r *journalOperatorService) UnpauseActivityExecution(ctx context.Context, req *workflowservicepb.UnpauseActivityExecutionRequest, opts ...grpc.CallOption) (*workflowservicepb.UnpauseActivityExecutionResponse, error) {
	out := &workflowservicepb.UnpauseActivityExecutionResponse{}
	err := r.mutate(ctx, "unpause", req, out, false, func() error {
		result, err := r.WorkflowServiceClient.UnpauseActivityExecution(ctx, req, opts...)
		if err == nil {
			proto.Merge(out, result)
		}
		return err
	})
	return out, err
}
func (r *journalOperatorService) ResetActivityExecution(ctx context.Context, req *workflowservicepb.ResetActivityExecutionRequest, opts ...grpc.CallOption) (*workflowservicepb.ResetActivityExecutionResponse, error) {
	out := &workflowservicepb.ResetActivityExecutionResponse{}
	err := r.mutate(ctx, "reset", req, out, false, func() error {
		result, err := r.WorkflowServiceClient.ResetActivityExecution(ctx, req, opts...)
		if err == nil {
			proto.Merge(out, result)
		}
		return err
	})
	return out, err
}
func (r *journalOperatorService) UpdateActivityExecutionOptions(ctx context.Context, req *workflowservicepb.UpdateActivityExecutionOptionsRequest, opts ...grpc.CallOption) (*workflowservicepb.UpdateActivityExecutionOptionsResponse, error) {
	out := &workflowservicepb.UpdateActivityExecutionOptionsResponse{}
	err := r.mutate(ctx, "update", req, out, false, func() error {
		result, err := r.WorkflowServiceClient.UpdateActivityExecutionOptions(ctx, req, opts...)
		if err == nil {
			proto.Merge(out, result)
		}
		return err
	})
	return out, err
}
func (r *journalOperatorService) TerminateActivityExecution(ctx context.Context, req *workflowservicepb.TerminateActivityExecutionRequest, opts ...grpc.CallOption) (*workflowservicepb.TerminateActivityExecutionResponse, error) {
	out := &workflowservicepb.TerminateActivityExecutionResponse{}
	err := r.mutate(ctx, "cleanup", req, out, false, func() error {
		result, err := r.WorkflowServiceClient.TerminateActivityExecution(ctx, req, opts...)
		if err == nil {
			proto.Merge(out, result)
		}
		return err
	})
	return out, err
}
