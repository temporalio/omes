package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
)

type runResult struct {
	Version           int                        `json:"version"`
	Scenario          string                     `json:"scenario"`
	RunID             string                     `json:"runId"`
	ExecutionID       string                     `json:"executionId"`
	RunFamily         string                     `json:"runFamily,omitempty"`
	Configuration     resultConfiguration        `json:"configuration"`
	StartedAt         time.Time                  `json:"startedAt"`
	EndedAt           *time.Time                 `json:"endedAt,omitempty"`
	Duration          string                     `json:"duration,omitempty"`
	Outcome           string                     `json:"outcome"`
	Error             string                     `json:"error,omitempty"`
	Iterations        *loadgen.IterationCounts   `json:"iterations,omitempty"`
	IterationFailures []loadgen.IterationFailure `json:"iterationFailures,omitempty"`
	ScenarioState     json.RawMessage            `json:"scenarioState,omitempty"`

	exportResultFile string
	statistics       *loadgen.RunStatistics
	options          *loadgen.OptionSet
	executor         loadgen.Executor
}

type resultConfiguration struct {
	Iterations                 int               `json:"iterations,omitempty"`
	Duration                   string            `json:"duration,omitempty"`
	MaxConcurrent              int               `json:"maxConcurrent"`
	MaxIterationsPerSecond     *float64          `json:"maxIterationsPerSecond"`
	MaxIterationAttempts       int               `json:"maxIterationAttempts"`
	Timeout                    string            `json:"timeout"`
	ContinueOnIterationFailure bool              `json:"continueOnIterationFailure,omitempty"`
	Options                    map[string]string `json:"options,omitempty"`
}

func resultConfig(configuration loadgen.RunConfiguration) resultConfiguration {
	result := resultConfiguration{
		Iterations:                 configuration.Iterations,
		MaxConcurrent:              configuration.MaxConcurrent,
		MaxIterationAttempts:       configuration.MaxIterationAttempts,
		Timeout:                    configuration.Timeout.String(),
		ContinueOnIterationFailure: configuration.ContinueOnIterationFailure,
	}
	if !math.IsNaN(configuration.MaxIterationsPerSecond) && !math.IsInf(configuration.MaxIterationsPerSecond, 0) {
		result.MaxIterationsPerSecond = &configuration.MaxIterationsPerSecond
	}
	if configuration.Duration != 0 {
		result.Duration = configuration.Duration.String()
	}
	return result
}

func startRunResult(id clioptions.ScenarioID, config scenarioRunConfig) *runResult {
	result := &runResult{
		Version:          1,
		Scenario:         id.Scenario,
		RunID:            id.RunID,
		RunFamily:        id.RunFamily,
		StartedAt:        time.Now().UTC(),
		exportResultFile: config.exportResultFile,
	}
	if config.exportResultFile != "" {
		result.statistics = &loadgen.RunStatistics{}
	}
	return result
}

func (r *runResult) finish(ctx context.Context, runErr error) error {
	if r.exportResultFile == "" || r.executor == nil {
		return runErr
	}
	snapshot := r.statistics.Snapshot()
	timedOut := errors.Is(runErr, context.DeadlineExceeded)
	if snapshot.Iterations != nil {
		timedOut = snapshot.TimedOut
	}
	var outcome string
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		outcome = "interrupted"
	case errors.Is(ctx.Err(), context.DeadlineExceeded), timedOut:
		outcome = "timedOut"
	case runErr != nil, snapshot.Iterations != nil && snapshot.Iterations.Failed > 0:
		outcome = "failed"
	default:
		outcome = "passed"
	}
	return errors.Join(runErr, r.write(outcome, runErr))
}

func (r *runResult) write(outcome string, runErr error) error {
	result := *r
	result.Outcome = outcome
	result.EndedAt = nil
	result.Duration = ""
	if outcome != "running" {
		endedAt := time.Now().UTC()
		result.EndedAt = &endedAt
		result.Duration = endedAt.Sub(result.StartedAt).String()
	}
	snapshot := r.statistics.Snapshot()
	if snapshot.Iterations != nil {
		result.Configuration = resultConfig(snapshot.Configuration)
	}
	result.Iterations = snapshot.Iterations
	result.IterationFailures = snapshot.Failures
	if r.options != nil {
		result.Configuration.Options = make(map[string]string)
		r.options.VisitAll(func(flag *pflag.Flag) { result.Configuration.Options[flag.Name] = flag.Value.String() })
	}
	result.Error = ""
	if runErr != nil {
		result.Error = runErr.Error()
	}
	var snapshotErr error
	if executor, ok := r.executor.(interface{ Snapshot() any }); ok {
		result.ScenarioState, snapshotErr = json.Marshal(executor.Snapshot())
		if snapshotErr != nil {
			snapshotErr = fmt.Errorf("encoding scenario state: %w", snapshotErr)
		}
	}
	data, err := json.MarshalIndent(&result, "", "  ")
	if err != nil {
		return errors.Join(snapshotErr, fmt.Errorf("encoding run result: %w", err))
	}
	if err := writeResultFile(r.exportResultFile, append(data, '\n')); err != nil {
		return errors.Join(snapshotErr, fmt.Errorf("writing result file: %w", err))
	}
	return snapshotErr
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
