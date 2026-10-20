package scenarios

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/workflowservice/v1"
	"golang.org/x/sync/errgroup"
)

func (state *finiteScheduleState) rootWorkflowID(iteration int) string {
	return (&loadgen.Run{ScenarioInfo: &loadgen.ScenarioInfo{RunID: state.RunID, ExecutionID: state.ExecutionID}, Iteration: iteration}).DefaultStartWorkflowOptions().ID
}

func (t *tpsExecutor) reconcileFiniteScheduleRoots(ctx context.Context, info loadgen.ScenarioInfo) error {
	t.lock.Lock()
	state := t.state.FiniteSchedules.clone()
	t.lock.Unlock()
	if state == nil {
		return errors.New("finite schedule recovery state is missing")
	}
	prefix := strings.TrimSuffix(state.rootWorkflowID(1), "1")
	maximum := state.LastPlannedIteration
	var token []byte
	seen := make(map[string]bool)
	for {
		page, err := info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: info.Namespace, PageSize: 1000, NextPageToken: token,
			Query: fmt.Sprintf("%s='%s'", loadgen.OmesExecutionIDSearchAttribute, strings.ReplaceAll(state.ExecutionID, "'", "''")),
		})
		if err != nil {
			return fmt.Errorf("reconciling finite schedule roots: %w", err)
		}
		if page == nil {
			return errors.New("finite schedule root inventory returned no response")
		}
		for _, execution := range page.Executions {
			id := execution.GetExecution().GetWorkflowId()
			suffix, ok := strings.CutPrefix(id, prefix)
			if !ok {
				continue
			}
			iteration, err := strconv.Atoi(suffix)
			if err == nil && iteration > 0 && strconv.Itoa(iteration) == suffix {
				maximum = max(maximum, iteration)
			}
		}
		token = page.NextPageToken
		if len(token) == 0 {
			break
		}
		if seen[string(token)] {
			return errors.New("finite schedule root inventory pagination did not advance")
		}
		seen[string(token)] = true
	}
	if maximum > 0 {
		return t.recordFiniteScheduleRoot(maximum)
	}
	return nil
}

func (t *tpsExecutor) verifyFiniteScheduleRoots(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := t.reconcileFiniteScheduleRoots(ctx, info); err != nil {
		return err
	}
	t.lock.Lock()
	state := t.state.FiniteSchedules.clone()
	t.lock.Unlock()
	for first := state.VerifiedThrough + 1; first <= state.LastPlannedIteration; first += 8 {
		last := min(first+7, state.LastPlannedIteration)
		group, groupCtx := errgroup.WithContext(ctx)
		for iteration := first; iteration <= last; iteration++ {
			group.Go(func() error {
				id := state.rootWorkflowID(iteration)
				if err := info.Client.GetWorkflow(groupCtx, id, "").Get(groupCtx, nil); err != nil {
					return fmt.Errorf("settling finite schedule parent %s: %w", id, err)
				}
				for _, input := range state.inputsForRoot(iteration) {
					if err := ks.VerifyFiniteSchedule(groupCtx, info.Client, input, loadgen.TaskQueueForRun(state.RunID)); err != nil {
						return err
					}
				}
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
		for iteration := first; iteration <= last; iteration++ {
			if err := t.recordVerifiedFiniteScheduleRoot(iteration); err != nil {
				return err
			}
		}
	}
	t.lock.Lock()
	t.state.CompletedIterations = max(t.state.CompletedIterations, state.LastPlannedIteration)
	t.lock.Unlock()
	return nil
}
