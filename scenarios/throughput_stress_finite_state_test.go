package scenarios

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/converter"
)

func finiteStateExecutor(t *testing.T) (*tpsExecutor, loadgen.ScenarioInfo) {
	t.Helper()
	info := finiteScheduleTestInfo("1")
	info.Namespace = "owned-namespace"
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	executor.config.FiniteScheduleRootPercentage = 100
	require.NoError(t, executor.prepareFiniteScheduleState(info))
	return executor, info
}

func TestFiniteStateRoundTripAndReconstruction(t *testing.T) {
	executor, info := finiteStateExecutor(t)
	require.NoError(t, executor.recordFiniteScheduleRoot(3))
	require.NoError(t, executor.recordVerifiedFiniteScheduleRoot(1))
	dc := converter.GetDefaultDataConverter()
	payload, err := dc.ToPayload(executor.Snapshot())
	require.NoError(t, err)
	resumed := newThroughputStressExecutor()
	require.NoError(t, resumed.LoadState(func(state any) error { return dc.FromPayload(payload, state) }))
	require.NoError(t, resumed.Configure(info))
	resumed.config.FiniteScheduleRootPercentage = 100
	require.NoError(t, resumed.prepareFiniteScheduleState(info))
	require.Equal(t, executor.Snapshot(), resumed.Snapshot())
	state := resumed.Snapshot().(tpsState).FiniteSchedules
	for iteration := 1; iteration <= 3; iteration++ {
		actual := finiteScheduleInputs(t, resumed.createActions(info.NewRun(iteration))[0].Actions)
		require.Equal(t, actual, state.inputsForRoot(iteration), "recovery must reconstruct the exact generated IDs")
	}
	for iteration := 4; iteration <= 10000; iteration++ {
		require.NoError(t, resumed.recordFiniteScheduleRoot(iteration))
	}
	large, err := dc.ToPayload(resumed.Snapshot())
	require.NoError(t, err)
	require.Less(t, len(large.Data), 1024, "heartbeat state must not grow one entry per generated schedule")
}

func TestFiniteStateRejectsIdentityAndProfileChanges(t *testing.T) {
	for _, change := range []string{"run", "execution", "namespace", "iterations", "percentage", "disabled"} {
		t.Run(change, func(t *testing.T) {
			executor, info := finiteStateExecutor(t)
			before := executor.Snapshot()
			switch change {
			case "run":
				info.RunID = "different-run"
			case "execution":
				info.ExecutionID = "different-execution"
			case "namespace":
				info.Namespace = "different-namespace"
			case "iterations":
				executor.config.InternalIterations++
			case "percentage":
				executor.config.FiniteScheduleRootPercentage = 50
			case "disabled":
				executor.config.FiniteScheduleRootPercentage = 0
			}
			require.Error(t, executor.prepareFiniteScheduleState(info))
			require.Equal(t, before, executor.Snapshot(), "failed prepare must not discard retained ownership")
		})
	}
}

func TestFiniteStateReconstructsSelectedRoots(t *testing.T) {
	info := finiteScheduleTestInfo("3")
	info.Namespace = "owned-namespace"
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	executor.config.FiniteScheduleRootPercentage = 50
	require.NoError(t, executor.prepareFiniteScheduleState(info))
	state := executor.Snapshot().(tpsState).FiniteSchedules
	var selected, skipped int
	for iteration := 1; iteration <= 100; iteration++ {
		expected := finiteScheduleInputs(t, executor.createActions(info.NewRun(iteration))[0].Actions)
		require.Equal(t, expected, state.inputsForRoot(iteration))
		if len(expected) == 0 {
			skipped++
		} else {
			selected++
		}
	}
	require.Positive(t, selected)
	require.Positive(t, skipped)
}

func TestFiniteStateProgressGuards(t *testing.T) {
	executor, _ := finiteStateExecutor(t)
	require.Error(t, executor.recordFiniteScheduleRoot(0))
	require.Error(t, executor.recordVerifiedFiniteScheduleRoot(1))
	require.NoError(t, executor.recordFiniteScheduleRoot(3))
	require.NoError(t, executor.recordFiniteScheduleRoot(2))
	require.Equal(t, 3, executor.Snapshot().(tpsState).FiniteSchedules.LastPlannedIteration)
	require.Error(t, executor.recordVerifiedFiniteScheduleRoot(2), "cannot skip an unverified root")
	require.NoError(t, executor.recordVerifiedFiniteScheduleRoot(1))
	require.Error(t, executor.recordVerifiedFiniteScheduleRoot(1), "cannot move backwards")
	require.NoError(t, executor.recordVerifiedFiniteScheduleRoot(2))
}

func TestFiniteStateSnapshotIsolation(t *testing.T) {
	executor, _ := finiteStateExecutor(t)
	before := executor.Snapshot().(tpsState)
	before.FiniteSchedules.Namespace = "changed-copy"
	require.Equal(t, "owned-namespace", executor.Snapshot().(tpsState).FiniteSchedules.Namespace)
	var wait sync.WaitGroup
	errors := make(chan error, 100)
	for iteration := 1; iteration <= 100; iteration++ {
		wait.Go(func() { errors <- executor.recordFiniteScheduleRoot(iteration) })
	}
	for range 100 {
		_, err := converter.GetDefaultDataConverter().ToPayload(executor.Snapshot())
		require.NoError(t, err)
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, 100, executor.Snapshot().(tpsState).FiniteSchedules.LastPlannedIteration)
}

func TestFiniteStateLoadOwnsState(t *testing.T) {
	original, _ := finiteStateExecutor(t)
	before := original.Snapshot()
	var retained *tpsState
	loaded := newThroughputStressExecutor()
	require.NoError(t, loaded.LoadState(func(target any) error {
		retained = target.(*tpsState)
		*retained = before.(tpsState)
		return nil
	}))
	retained.CompletedIterations = 100
	retained.FiniteSchedules.Namespace = "changed-after-load"
	require.Equal(t, "owned-namespace", loaded.Snapshot().(tpsState).FiniteSchedules.Namespace)
	require.Zero(t, loaded.Snapshot().(tpsState).CompletedIterations)
}

func TestFiniteStateInvalidLoadPreservesState(t *testing.T) {
	executor, _ := finiteStateExecutor(t)
	before := executor.Snapshot()
	err := executor.LoadState(func(target any) error {
		state := target.(*tpsState)
		state.CompletedIterations = 99
		state.FiniteSchedules = &finiteScheduleState{Version: 999}
		return nil
	})
	require.Error(t, err)
	require.Equal(t, before, executor.Snapshot())
}

func TestFiniteStateLegacyDefaults(t *testing.T) {
	info := finiteScheduleTestInfo("1")
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.LoadState(func(target any) error {
		target.(*tpsState).CompletedIterations = 5
		return nil
	}))
	require.NoError(t, executor.Configure(info))
	require.NoError(t, executor.prepareFiniteScheduleState(info))
	require.Nil(t, executor.Snapshot().(tpsState).FiniteSchedules)
	executor.config.FiniteScheduleRootPercentage = 100
	require.Error(t, executor.prepareFiniteScheduleState(info), "must not assume earlier roots generated finite schedules")
}
