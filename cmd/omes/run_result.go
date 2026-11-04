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
	EndedAt           time.Time                  `json:"endedAt"`
	Duration          string                     `json:"duration"`
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
	r.EndedAt = time.Now().UTC()
	r.Duration = r.EndedAt.Sub(r.StartedAt).String()
	snapshot := r.statistics.Snapshot()
	if snapshot.Iterations != nil {
		r.Configuration = resultConfig(snapshot.Configuration)
	}
	r.Iterations = snapshot.Iterations
	r.IterationFailures = snapshot.Failures
	if r.options != nil {
		r.Configuration.Options = make(map[string]string)
		r.options.VisitAll(func(flag *pflag.Flag) { r.Configuration.Options[flag.Name] = flag.Value.String() })
	}
	timedOut := errors.Is(runErr, context.DeadlineExceeded)
	if snapshot.Iterations != nil {
		timedOut = snapshot.TimedOut
	}
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		r.Outcome = "interrupted"
	case errors.Is(ctx.Err(), context.DeadlineExceeded), timedOut:
		r.Outcome = "timedOut"
	case runErr != nil, r.Iterations != nil && r.Iterations.Failed > 0:
		r.Outcome = "failed"
	default:
		r.Outcome = "passed"
	}
	if runErr != nil {
		r.Error = runErr.Error()
	}
	var snapshotErr error
	if executor, ok := r.executor.(interface{ Snapshot() any }); ok {
		r.ScenarioState, snapshotErr = json.Marshal(executor.Snapshot())
		if snapshotErr != nil {
			snapshotErr = fmt.Errorf("encoding scenario state: %w", snapshotErr)
		}
	}
	return errors.Join(runErr, snapshotErr, r.write(r.exportResultFile))
}

func (r *runResult) write(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding run result: %w", err)
	}
	if err := writeResultFile(path, append(data, '\n')); err != nil {
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
