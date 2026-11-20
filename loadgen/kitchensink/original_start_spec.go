package kitchensink

import (
	"errors"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// OriginalStartSpec preserves identity and payloads without SDK handles.
type OriginalStartSpec struct {
	Version                                                int
	Intent, Namespace, WorkflowID, WorkflowType, TaskQueue string
	Input, Action                                          []byte
}

// PrepareOriginalStart freezes a new request without contacting the server.
func (e *ClientActionsExecutor) PrepareOriginalStart(action *WithStartClientAction) (OriginalStartSpec, error) {
	if !e.RecoverOriginalStart || e.Execution != nil || action.GetDoSignal() != nil {
		return OriginalStartSpec{}, errors.New("unsupported original start configuration")
	}
	if e.originalStart == nil {
		e.freezeStart(action)
	}
	input, err := proto.Marshal(e.WorkflowInput)
	if err != nil {
		return OriginalStartSpec{}, err
	}
	var frozenAction []byte
	if e.originalStart.action != nil {
		frozenAction, err = proto.Marshal(e.originalStart.action)
		if err != nil {
			return OriginalStartSpec{}, err
		}
	}
	return OriginalStartSpec{Version: 1, Intent: e.originalStart.intent, Namespace: e.Namespace, WorkflowID: e.WorkflowOptions.ID, WorkflowType: e.WorkflowType, TaskQueue: e.WorkflowOptions.TaskQueue, Input: input, Action: frozenAction}, nil
}

// RestoreOriginalStart imports recorded intent without allocating new IDs or sending requests.
func (e *ClientActionsExecutor) RestoreOriginalStart(spec OriginalStartSpec) error {
	if spec.Version != 1 || spec.Intent == "" || spec.Namespace != e.Namespace || spec.WorkflowID == "" || spec.WorkflowID != e.WorkflowOptions.ID || spec.TaskQueue == "" || spec.TaskQueue != e.WorkflowOptions.TaskQueue || spec.WorkflowType != e.WorkflowType || e.originalStart != nil || e.Execution != nil {
		return errors.New("invalid original start specification")
	}
	if _, err := uuid.Parse(spec.Intent); err != nil {
		return errors.New("invalid original start intent UUID")
	}
	input := &WorkflowInput{}
	if err := proto.Unmarshal(spec.Input, input); err != nil {
		return err
	}
	var action *WithStartClientAction
	if len(spec.Action) > 0 {
		action = &WithStartClientAction{}
		if err := proto.Unmarshal(spec.Action, action); err != nil {
			return err
		}
		if action.GetDoUpdate() == nil || !action.GetDoUpdate().WithStart || action.GetDoUpdate().UpdateId == "" {
			return errors.New("invalid original update-with-start specification")
		}
	}
	e.RecoverOriginalStart = true
	e.WorkflowInput = input
	e.originalStart = &originalStart{intent: spec.Intent, action: action}
	e.setOriginalStartOptions()
	return nil
}
