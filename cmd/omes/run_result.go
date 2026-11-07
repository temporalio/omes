package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/temporalio/omes/loadgen"
)

type runResult struct {
	Outcome       string                     `json:"outcome"`
	Error         string                     `json:"error,omitempty"`
	Iterations    *loadgen.IterationCounts   `json:"iterations"`
	Failures      []loadgen.IterationFailure `json:"failures"`
	ScenarioState any                        `json:"scenarioState"`
}

func (r *scenarioRunner) runExecutor(ctx context.Context, executor loadgen.Executor, info loadgen.ScenarioInfo) (retErr error) {
	var snapshotState func() any
	if state, ok := executor.(interface{ Snapshot() any }); ok && r.exportResultFile != "" {
		snapshotState = state.Snapshot
		defer func() { r.scenarioState = state.Snapshot() }()
	}
	if r.exportResultFile != "" && r.exportResultInterval > 0 {
		ticker := time.NewTicker(r.exportResultInterval)
		stop := make(chan struct{})
		done := make(chan struct{})
		var snapshotErr error
		defer func() {
			ticker.Stop()
			close(stop)
			// Wait for any in-flight snapshot before the final writer can run.
			<-done
			retErr = errors.Join(retErr, snapshotErr)
		}()
		go func() {
			defer close(done)
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					snapshot := r.snapshotStatistics()
					result := runResult{Outcome: "running", Iterations: snapshot.Iterations, Failures: snapshot.Failures}
					if snapshotState != nil {
						result.ScenarioState = snapshotState()
					}
					if snapshotErr = r.writeResult(result); snapshotErr != nil {
						return
					}
				}
			}
		}()
	}
	return executor.Run(ctx, info)
}

func (r *scenarioRunner) finish(ctx context.Context, runErr *error) {
	// Panics on other goroutines bypass this writer.
	panicValue := recover()
	if panicValue != nil {
		*runErr = fmt.Errorf("panic: %v", panicValue)
		defer func() { panic(panicValue) }()
	}
	if r.exportResultFile == "" {
		return
	}

	snapshot := r.snapshotStatistics()
	result := runResult{
		Iterations:    snapshot.Iterations,
		Failures:      snapshot.Failures,
		ScenarioState: r.scenarioState,
	}
	timedOut := errors.Is(*runErr, context.DeadlineExceeded)
	if snapshot.Iterations != nil {
		timedOut = snapshot.TimedOut
	}
	switch {
	case panicValue != nil:
		result.Outcome = "failed"
	case errors.Is(ctx.Err(), context.Canceled):
		result.Outcome = "interrupted"
	case errors.Is(ctx.Err(), context.DeadlineExceeded), timedOut:
		result.Outcome = "timedOut"
	case *runErr != nil, snapshot.Iterations != nil && snapshot.Iterations.Failed > 0:
		result.Outcome = "failed"
	default:
		result.Outcome = "passed"
	}
	if *runErr != nil {
		result.Error = (*runErr).Error()
	}
	*runErr = errors.Join(*runErr, r.writeResult(result))
}

func (r *scenarioRunner) snapshotStatistics() loadgen.RunStatisticsSnapshot {
	var snapshot loadgen.RunStatisticsSnapshot
	if r.statistics != nil {
		snapshot = r.statistics.Snapshot()
	}
	if snapshot.Failures == nil {
		snapshot.Failures = []loadgen.IterationFailure{}
	}
	return snapshot
}

func (r *scenarioRunner) writeResult(result runResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding run result: %w", err)
	}
	if err := writeResultFile(r.exportResultFile, append(data, '\n')); err != nil {
		return fmt.Errorf("writing result file: %w", err)
	}
	return nil
}

func writeResultFile(path string, data []byte) (retErr error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".omes-result-*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, err)
		}
	}()
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
