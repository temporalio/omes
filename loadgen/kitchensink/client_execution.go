package kitchensink

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// ClientExecution identifies the owning workflow and stable activity invocation.
type ClientExecution struct {
	RunID         string
	UpdateIDScope string
}

func (e *ClientActionsExecutor) targetRunID() string {
	if e.Execution != nil {
		return e.Execution.RunID
	}
	return ""
}

func (e ClientExecution) prepare(seq *ClientSequence) (*ClientSequence, error) {
	if e.RunID == "" || e.UpdateIDScope == "" {
		return nil, errors.New("owning-execution actions require run ID and update ID scope")
	}
	if seq == nil {
		return nil, errors.New("client sequence is required")
	}
	cloned := proto.Clone(seq).(*ClientSequence)
	for i, set := range cloned.ActionSets {
		if err := e.prepareSet(set, fmt.Sprint(i)); err != nil {
			return nil, err
		}
	}
	return cloned, nil
}

func (e ClientExecution) prepareSet(set *ClientActionSet, path string) error {
	if set == nil {
		return errors.New("client action set is required")
	}
	if set.WaitForCurrentRunToFinishAtEnd {
		return errors.New("owning-execution actions cannot rebind after waiting for a run")
	}
	for i, action := range set.Actions {
		position := fmt.Sprintf("%s/%d", path, i)
		switch {
		case action.GetDoSignal() != nil:
			if action.GetDoSignal().WithStart {
				return errors.New("owning-execution actions cannot signal with start")
			}
		case action.GetDoUpdate() != nil:
			update := action.GetDoUpdate()
			if update.WithStart {
				return errors.New("owning-execution actions cannot update with start")
			}
			if update.UpdateId == "" {
				update.UpdateId = uuid.NewSHA1(uuid.NameSpaceOID, []byte(e.UpdateIDScope+"/"+position)).String()
			}
		case action.GetDoQuery() != nil, action.GetDoDescribe() != nil:
		case action.GetNestedActions() != nil:
			if err := e.prepareSet(action.GetNestedActions(), position); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported owning-execution action at %s", position)
		}
	}
	return nil
}
