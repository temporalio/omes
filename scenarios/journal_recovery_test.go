package scenarios

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/journal"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
)

type scenarioJournal struct {
	mu     sync.Mutex
	states map[string]journal.State
	lose   string
}

func (b *scenarioJournal) Apply(_ context.Context, lane string, c journal.Command) (journal.State, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.states == nil {
		b.states = map[string]journal.State{}
	}
	s, ok := b.states[lane]
	if !ok {
		s = journal.New(lane)
	}
	n, err := s.Apply(c)
	if err != nil {
		return s, err
	}
	b.states[lane] = n
	key := c.Kind
	if c.Key != "" {
		key += "/" + c.Key
	}
	if b.lose == key {
		b.lose = ""
		return journal.State{}, errors.New("lost acknowledged journal response")
	}
	return n.Clone(), nil
}
func TestJournaledThroughputRestoresFrozenOutcome(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	setup := loadgen.ScenarioInfo{Client: env.TemporalClient(), Namespace: "default"}
	require.NoError(t, setup.RegisterDefaultSearchAttributes(t.Context()))
	for _, cut := range []string{"observe/workflow", "commit"} {
		t.Run(cut, func(t *testing.T) {
			backend := &scenarioJournal{lose: cut}
			info := loadgen.ScenarioInfo{
				ScenarioName: "throughput_stress", RunID: "journal-" + cut, ExecutionID: cut, OwningExecutionWorkers: true,
				Configuration: loadgen.RunConfiguration{Iterations: 1, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true, ContinueOnIterationFailure: true},
				Options:       loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{IterFlag: "1", ContinueAsNewAfterIterFlag: "0", NexusEnabledFlag: "false", IncludeStandaloneActivityFlag: "false", SleepTimeFlag: "1ms", VisibilityVerificationTimeoutFlag: "10s"}),
				Recovery:      &loadgen.RecoveryOptions{Client: backend, Epoch: 1, Contract: []byte("fixed run contract"), Streams: map[string][]string{"throughput_stress": {"main/0"}}, GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(2 * time.Minute)},
			}
			generations := 0
			first := newThroughputStressExecutor()
			first.onActionsCreated = func([]*ks.ActionSet) { generations++ }
			_, err := env.RunExecutorTest(t, first, info, clioptions.LangGo)
			require.ErrorIs(t, err, journal.ErrPersistence)
			second := newThroughputStressExecutor()
			second.onActionsCreated = func([]*ks.ActionSet) { generations++ }
			info.Recovery.Epoch = 2
			_, err = env.RunExecutorTest(t, second, info, clioptions.LangGo)
			require.NoError(t, err)
			require.Equal(t, 1, generations)
			require.Equal(t, 1, second.Snapshot().(tpsState).CompletedIterations)
			require.Equal(t, journal.Counts{Allocated: 1, Succeeded: 1}, backend.states["main/0"].Counts)
		})
	}
}

func TestJournaledOperatorSequences(t *testing.T) {
	server := workertest.StartDevServer(t, workertest.WithDynamicConfig(map[string]any{
		"activity.enableStandalone": true, "history.enableStandaloneActivityOperatorCommands": true,
	}))
	env := workertest.SetupTestEnvironment(t, workertest.WithDevServer(server), workertest.WithExecutorTimeout(time.Minute))
	backend := &scenarioJournal{}
	name := "standalone_activity_operator_commands"
	info := loadgen.ScenarioInfo{
		ScenarioName: name, RunID: "journal-operators", ExecutionID: "operators",
		Configuration: loadgen.RunConfiguration{Iterations: 3, MaxConcurrent: 3, DoNotRegisterSearchAttributes: true},
		Options:       loadgen.MustResolveScenarioOptions(name, nil),
		Recovery:      &loadgen.RecoveryOptions{Client: backend, Epoch: 1, Contract: []byte("operator contract"), Streams: map[string][]string{name: {"operators/0", "operators/1", "operators/2"}}, GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(2 * time.Minute)},
	}
	_, err := env.RunExecutorTest(t, loadgen.ExecutorFunc(runStandaloneActivityOperatorCommands), info, clioptions.LangGo)
	require.NoError(t, err)
	total, err := info.Recovery.Totals(t.Context(), name)
	require.NoError(t, err)
	require.Equal(t, journal.Counts{Allocated: 3, Succeeded: 3}, total)
}

func TestJournaledSchedulerReceipts(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	backend := &scenarioJournal{}
	name := "scheduler_stress"
	info := loadgen.ScenarioInfo{
		ScenarioName: name, RunID: "journal-schedules", ExecutionID: "schedules",
		Configuration: loadgen.RunConfiguration{Iterations: 1, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true},
		Options: loadgen.MustResolveScenarioOptions(name, map[string]string{
			ScheduleCreationPerIterationFlag: "1", ScheduleReadsPerCreationFlag: "1", ScheduleUpdatesPerCreationFlag: "1",
			SchedulerDurationPerIterationFlag: "1s", WaitTimeBeforeCleanupFlag: "10ms", OperationIntervalFlag: "1ms", RecoverScheduleOperationsFlag: "true",
		}),
		Recovery: &loadgen.RecoveryOptions{Client: backend, Epoch: 1, Contract: []byte("scheduler contract"), Streams: map[string][]string{name: {"scheduler/0"}}, GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(2 * time.Minute)},
	}
	_, err := env.RunExecutorTest(t, loadgen.GetScenario(name).ExecutorFn(), info, clioptions.LangGo)
	require.NoError(t, err)
	require.Equal(t, journal.Counts{Allocated: 1, Succeeded: 1}, backend.states["scheduler/0"].Counts)
}
