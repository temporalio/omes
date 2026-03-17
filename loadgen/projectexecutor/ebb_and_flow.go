package projectexecutor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/projecttests/go/harness/api"
)

type ebbAndFlowConfig struct {
	MinBacklog           int64
	MaxBacklog           int64
	Period               time.Duration
	MaxRate              int64
	ControlInterval      time.Duration
	MaxConsecutiveErrors int
	BacklogLogInterval   time.Duration
}

type ebbAndFlowProjectExecutor struct {
	client             *ProjectHandle
	completedWorkflows int
}

// CompletedWorkflows returns the number of workflows completed during the last Run.
func (e *ebbAndFlowProjectExecutor) CompletedWorkflows() int {
	return e.completedWorkflows
}

func parseEbbAndFlowConfig(info loadgen.ScenarioInfo) (*ebbAndFlowConfig, error) {
	config := &ebbAndFlowConfig{
		MinBacklog:           int64(info.ScenarioOptionInt("min-backlog", 0)),
		MaxBacklog:           int64(info.ScenarioOptionInt("max-backlog", 30)),
		Period:               info.ScenarioOptionDuration("period", 60*time.Second),
		MaxRate:              int64(info.ScenarioOptionInt("max-rate", 1000)),
		ControlInterval:      info.ScenarioOptionDuration("control-interval", 100*time.Millisecond),
		MaxConsecutiveErrors: info.ScenarioOptionInt("max-consecutive-errors", 10),
		BacklogLogInterval:   info.ScenarioOptionDuration("backlog-log-interval", 30*time.Second),
	}

	if config.MinBacklog < 0 {
		return nil, fmt.Errorf("min-backlog must be non-negative, got %d", config.MinBacklog)
	}
	if config.MaxBacklog <= config.MinBacklog {
		return nil, fmt.Errorf("max-backlog must be greater than min-backlog, got max=%d min=%d", config.MaxBacklog, config.MinBacklog)
	}
	if config.Period <= 0 {
		return nil, fmt.Errorf("period must be greater than 0, got %v", config.Period)
	}
	return config, nil
}

// calculateBacklogTarget computes the target backlog at a given elapsed time
// using a sine wave oscillation between minBacklog and maxBacklog.
// Duplicated from scenarios/ebb_and_flow.go (unexported there).
func calculateBacklogTarget(elapsed, period time.Duration, minBacklog, maxBacklog int64) int64 {
	periods := elapsed.Seconds() / period.Seconds()
	osc := (math.Sin(2*math.Pi*(periods-0.25)) + 1.0) / 2
	backlogRange := float64(maxBacklog - minBacklog)
	baseTarget := float64(minBacklog) + osc*backlogRange
	return int64(math.Round(baseTarget))
}

// ebbAndFlowPayload is the per-iteration payload sent to the project.
type ebbAndFlowPayload struct {
	ActivityCount int64 `json:"activity_count"`
}

func (e *ebbAndFlowProjectExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	config, err := parseEbbAndFlowConfig(info)
	if err != nil {
		return fmt.Errorf("invalid ebb-and-flow config: %w", err)
	}
	if info.Configuration.Duration <= 0 {
		return fmt.Errorf("ebb-and-flow executor requires --duration")
	}

	executeTimer := info.MetricsHandler.WithTags(map[string]string{
		"scenario": info.ScenarioName,
	}).Timer("omes_execute_histogram")

	var scheduledActivities atomic.Int64
	var completedActivities atomic.Int64
	var consecutiveErrors int

	errCh := make(chan error, 10000)
	ticker := time.NewTicker(config.ControlInterval)
	defer ticker.Stop()
	backlogTicker := time.NewTicker(config.BacklogLogInterval)
	defer backlogTicker.Stop()

	var workersWG sync.WaitGroup
	var iter int64 = 1
	startTime := time.Now()

	info.Logger.Infof("Starting ebb-and-flow executor: min_backlog=%d, max_backlog=%d, period=%v, duration=%v",
		config.MinBacklog, config.MaxBacklog, config.Period, info.Configuration.Duration)

	var scheduled, completed, backlog, target int64

	for elapsed := time.Duration(0); elapsed < info.Configuration.Duration; elapsed = time.Since(startTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				info.Logger.Errorf("Execute failed: %v", err)
				consecutiveErrors++
				if consecutiveErrors >= config.MaxConsecutiveErrors {
					return fmt.Errorf("got %d consecutive errors, most recent: %w",
						config.MaxConsecutiveErrors, err)
				}
			} else {
				consecutiveErrors = 0
			}
		case <-ticker.C:
			scheduled = scheduledActivities.Load()
			completed = completedActivities.Load()
			backlog = scheduled - completed

			target = calculateBacklogTarget(elapsed, config.Period, config.MinBacklog, config.MaxBacklog)
			activities := target - backlog
			activities = max(activities, 0)
			activities = min(activities, config.MaxRate)

			if activities > 0 {
				payload, marshalErr := json.Marshal(ebbAndFlowPayload{ActivityCount: activities})
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal payload: %w", marshalErr)
				}

				scheduledActivities.Add(activities)
				workersWG.Add(1)
				go func(iter, activities int64, payload []byte) {
					defer workersWG.Done()
					iterStart := time.Now()
					defer func() {
						executeTimer.Record(time.Since(iterStart))
					}()

					_, execErr := e.client.Execute(ctx, &api.ExecuteRequest{
						Iteration: iter,
						Payload:   payload,
					})
					completedActivities.Add(activities)
					errCh <- execErr
				}(iter, activities, payload)
				iter++
			}
		case <-backlogTicker.C:
			info.Logger.Debugf("Backlog: %d, target: %d, scheduled: %d, completed: %d",
				backlog, target, scheduled, completed)
		}
	}

	info.Logger.Info("Duration complete; waiting for in-flight workflows to finish...")
	workersWG.Wait()

	e.completedWorkflows = int(iter - 1)
	info.Logger.Infof("Ebb-and-flow run completed: scheduled=%d, completed=%d, workflows=%d",
		scheduledActivities.Load(), completedActivities.Load(), e.completedWorkflows)
	return nil
}
