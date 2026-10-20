package scenarios

import (
	"errors"
	"fmt"

	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
)

type finiteScheduleState struct {
	Version              int    `json:"version"`
	RunID                string `json:"runId"`
	ExecutionID          string `json:"executionId"`
	Namespace            string `json:"namespace"`
	InternalIterations   int    `json:"internalIterations"`
	RootPercentage       int    `json:"rootPercentage"`
	LastPlannedIteration int    `json:"lastPlannedIteration"`
	VerifiedThrough      int    `json:"verifiedThrough"`
	DisruptionRecovery   bool   `json:"disruptionRecovery,omitempty"`
}

func (state *finiteScheduleState) validate() error {
	if state == nil {
		return nil
	}
	if state.Version != 1 || state.RunID == "" || state.ExecutionID == "" || state.Namespace == "" {
		return errors.New("finite schedule state has unsupported version or incomplete identity")
	}
	if state.InternalIterations <= 0 || state.RootPercentage <= 0 || state.RootPercentage > 100 ||
		state.LastPlannedIteration < 0 || state.VerifiedThrough < 0 || state.VerifiedThrough > state.LastPlannedIteration {
		return errors.New("finite schedule state has invalid profile or progress")
	}
	return nil
}

func (state *finiteScheduleState) clone() *finiteScheduleState {
	if state == nil {
		return nil
	}
	copy := *state
	return &copy
}

func (t *tpsExecutor) prepareFiniteScheduleState(info loadgen.ScenarioInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	percentage := t.config.FiniteScheduleRootPercentage
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("finite schedule root percentage must be between 0 and 100, got %d", percentage)
	}
	state := t.state.FiniteSchedules
	if state == nil && percentage == 0 {
		return nil
	}
	if state != nil {
		if err := state.validate(); err != nil {
			return err
		}
		if state.RunID != info.RunID || state.ExecutionID != info.ExecutionID || state.Namespace != info.Namespace ||
			state.InternalIterations != t.config.InternalIterations || state.RootPercentage != percentage ||
			state.DisruptionRecovery != t.config.FiniteScheduleDisruptionRecovery {
			return errors.New("finite schedule resume identity or profile differs from retained state")
		}
		return nil
	}
	if t.state.CompletedIterations != 0 || info.Configuration.StartFromIteration != 0 {
		return errors.New("cannot enable finite schedules on an existing run without recovery state")
	}
	state = &finiteScheduleState{
		Version: 1, RunID: info.RunID, ExecutionID: info.ExecutionID, Namespace: info.Namespace,
		InternalIterations: t.config.InternalIterations, RootPercentage: percentage,
		DisruptionRecovery: t.config.FiniteScheduleDisruptionRecovery,
	}
	if err := state.validate(); err != nil {
		return err
	}
	t.state.FiniteSchedules = state
	return nil
}

func (t *tpsExecutor) recordFiniteScheduleRoot(iteration int) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	state := t.state.FiniteSchedules
	if state == nil || iteration <= 0 {
		return errors.New("finite schedule root requires prepared state and a positive iteration")
	}
	state.LastPlannedIteration = max(state.LastPlannedIteration, iteration)
	return nil
}

func (t *tpsExecutor) recordVerifiedFiniteScheduleRoot(iteration int) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	state := t.state.FiniteSchedules
	if state == nil || iteration != state.VerifiedThrough+1 || iteration > state.LastPlannedIteration {
		return errors.New("finite schedule verification must advance through planned roots in order")
	}
	state.VerifiedThrough = iteration
	return nil
}

func (state *finiteScheduleState) inputsForRoot(iteration int) []ks.FiniteScheduleInput {
	root := &loadgen.Run{ScenarioInfo: &loadgen.ScenarioInfo{RunID: state.RunID, ExecutionID: state.ExecutionID}, Iteration: iteration}
	id := root.DefaultStartWorkflowOptions().ID
	if !finiteScheduleRootSelected(id, state.RootPercentage) {
		return nil
	}
	inputs := make([]ks.FiniteScheduleInput, state.InternalIterations)
	for index := range inputs {
		inputs[index].ScheduleID = finiteScheduleID(id, (iteration-1)*state.InternalIterations+index)
	}
	return inputs
}
