package scenarios

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap/zaptest"
)

func TestSchedulerStress_Configure(t *testing.T) {
	t.Parallel()

	configure := func(t *testing.T, opts map[string]string) (*schedulerExecutorConfig, error) {
		t.Helper()
		var s SchedulerExecutor
		err := s.Configure(loadgen.ScenarioInfo{
			Options: loadgen.MustResolveScenarioOptions("scheduler_stress", opts),
		})
		return s.config, err
	}

	t.Run("defaults", func(t *testing.T) {
		cfg, err := configure(t, nil)
		require.NoError(t, err)
		require.Equal(t, DefaultScheduleCreationPerIteration, cfg.ScheduleCreationPerIteration)
		require.Equal(t, DefaultScheduleReadsPerCreation, cfg.ScheduleReadsPerCreation)
		require.Equal(t, DefaultScheduleUpdatesPerCreation, cfg.ScheduleUpdatesPerCreation)
		require.Equal(t, DefaultPayloadSize, cfg.PayloadSize)
		require.Equal(t, NoopScheduledWorkflowType, cfg.ScheduledWorkflowType)
		require.True(t, cfg.EnableChasmScheduler)
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		cases := map[string]map[string]string{
			"non-positive creations":      {ScheduleCreationPerIterationFlag: "0"},
			"negative reads":              {ScheduleReadsPerCreationFlag: "-1"},
			"negative updates":            {ScheduleUpdatesPerCreationFlag: "-1"},
			"non-positive payload":        {PayloadSizeFlag: "0"},
			"non-positive iter duration":  {SchedulerDurationPerIterationFlag: "0s"},
			"negative cleanup wait":       {WaitTimeBeforeCleanupFlag: "-1s"},
			"negative operation interval": {OperationIntervalFlag: "-1s"},
		}
		for name, opts := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := configure(t, opts)
				require.Error(t, err)
			})
		}
	})
}

func TestSchedulerStress(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(2*time.Minute))

	scenarioInfo := func(runID string, overrides map[string]string) loadgen.ScenarioInfo {
		opts := map[string]string{
			ScheduleCreationPerIterationFlag:  "2",
			ScheduleReadsPerCreationFlag:      "1",
			ScheduleUpdatesPerCreationFlag:    "1",
			OperationIntervalFlag:             "10ms",
			SchedulerDurationPerIterationFlag: "1s",
			WaitTimeBeforeCleanupFlag:         "1s",
			// Routing is the pipeline's concern; these cases are about run outcome.
			EnableChasmSchedulerFlag: "false",
		}
		maps.Copy(opts, overrides)
		return loadgen.ScenarioInfo{
			RunID:         runID,
			Configuration: loadgen.RunConfiguration{Iterations: 1},
			Options:       loadgen.MustResolveScenarioOptions("scheduler_stress", opts),
		}
	}

	run := func(t *testing.T, info loadgen.ScenarioInfo) error {
		t.Helper()
		scenario := loadgen.GetScenario("scheduler_stress")
		require.NotNil(t, scenario)
		_, err := env.RunExecutorTest(t, scenario.ExecutorFn(), info, clioptions.LangGo)
		return err
	}

	t.Run("healthy run reports success", func(t *testing.T) {
		runID := fmt.Sprintf("sched-ok-%d", time.Now().UnixNano())
		require.NoError(t, run(t, scenarioInfo(runID, nil)))
	})

	t.Run("rejected schedule creation fails the run", func(t *testing.T) {
		// An unparseable cron spec makes the server reject every Create.
		runID := fmt.Sprintf("sched-bad-%d", time.Now().UnixNano())
		err := run(t, scenarioInfo(runID, map[string]string{CronExpressionFlag: "not a cron expression"}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating schedule")
	})

	t.Run("cleanup deletes schedules unless retained", func(t *testing.T) {
		for _, retain := range []bool{false, true} {
			t.Run(fmt.Sprintf("retain=%t", retain), func(t *testing.T) {
				info := scenarioInfo(fmt.Sprintf("sched-retain-%d", time.Now().UnixNano()), map[string]string{
					RetainSchedulesFlag:       strconv.FormatBool(retain),
					WaitTimeBeforeCleanupFlag: "0s",
				})
				schedules := &createdSchedules{ScheduleClient: env.TemporalClient().ScheduleClient()}
				info.Client = scheduleRecordingClient{Client: env.TemporalClient(), schedules: schedules}
				info.Logger = zaptest.NewLogger(t).Sugar()
				start := time.Now()

				require.NoError(t, new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1)))

				require.Len(t, schedules.ids, 2)
				for _, id := range schedules.ids {
					handle := env.TemporalClient().ScheduleClient().GetHandle(t.Context(), id)
					description, err := handle.Describe(t.Context())
					if !retain {
						var notFound *serviceerror.NotFound
						require.ErrorAs(t, err, &notFound)
						continue
					}
					require.NoError(t, err)
					t.Cleanup(func() { _ = handle.Delete(context.Background()) })
					end := description.Schedule.Spec.EndAt
					require.True(t, end.After(start) && end.Before(start.Add(4*time.Second)), "retained schedule must stay finite, EndAt %v", end)
				}
			})
		}
	})

	t.Run("a stopped iteration reads as canceled, not failed", func(t *testing.T) {
		// GenericExecutor keys abandoned-vs-failed off errors.Is(err,
		// context.Canceled), which the operations' own gRPC errors do not satisfy.
		info := scenarioInfo(fmt.Sprintf("sched-stop-%d", time.Now().UnixNano()), nil)
		info.Logger = zaptest.NewLogger(t).Sugar()
		info.Client = env.TemporalClient()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// scheduleRecordingClient exposes the IDs the executor creates, which it generates randomly.
type scheduleRecordingClient struct {
	client.Client
	schedules *createdSchedules
}

func (c scheduleRecordingClient) ScheduleClient() client.ScheduleClient { return c.schedules }

type createdSchedules struct {
	client.ScheduleClient
	mu  sync.Mutex
	ids []string
}

func (s *createdSchedules) Create(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	s.mu.Lock()
	s.ids = append(s.ids, options.ID)
	s.mu.Unlock()
	return s.ScheduleClient.Create(ctx, options)
}
