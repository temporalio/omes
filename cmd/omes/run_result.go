package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/temporalio/omes/loadgen"
)

type runResult struct {
	Outcome       string                     `json:"outcome"`
	Error         string                     `json:"error,omitempty"`
	Iterations    *loadgen.IterationCounts   `json:"iterations"`
	Failures      []loadgen.IterationFailure `json:"failures"`
	ScenarioState any                        `json:"scenarioState"`
}

// runEnd describes how scenario execution ended.
type runEnd struct {
	finished bool
	err      error // the scenario's own error
	ctxErr   error // cancellation or deadline, if any
	panicked bool
}

func newRunResult(end runEnd, counts loadgen.RunCounts, scenarioState any) runResult {
	if counts.Failures == nil {
		counts.Failures = []loadgen.IterationFailure{}
	}
	result := runResult{
		Iterations:    counts.Iterations,
		Failures:      counts.Failures,
		ScenarioState: scenarioState,
	}
	timedOut := errors.Is(end.err, context.DeadlineExceeded)
	if counts.Iterations != nil {
		timedOut = counts.TimedOut
	}
	switch {
	case !end.finished:
		result.Outcome = "running"
	case end.panicked:
		result.Outcome = "failed"
	case errors.Is(end.ctxErr, context.Canceled):
		result.Outcome = "interrupted"
	case errors.Is(end.ctxErr, context.DeadlineExceeded), timedOut:
		result.Outcome = "timedOut"
	case end.err != nil, counts.Iterations != nil && counts.Iterations.Failed > 0:
		result.Outcome = "failed"
	default:
		result.Outcome = "passed"
	}
	if end.err != nil {
		result.Error = end.err.Error()
	}
	return result
}

func (r runResult) write(path string) error {
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
	// Write to a temp file in the same directory, then rename so readers never see a partial result.
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
