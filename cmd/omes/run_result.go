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

func newRunResult(ctxErr, runErr error, panicked bool, snapshot loadgen.RunCounts, scenarioState any) runResult {
	if snapshot.Failures == nil {
		snapshot.Failures = []loadgen.IterationFailure{}
	}
	result := runResult{
		Iterations:    snapshot.Iterations,
		Failures:      snapshot.Failures,
		ScenarioState: scenarioState,
	}
	timedOut := errors.Is(runErr, context.DeadlineExceeded)
	if snapshot.Iterations != nil {
		timedOut = snapshot.TimedOut
	}
	switch {
	case panicked:
		result.Outcome = "failed"
	case errors.Is(ctxErr, context.Canceled):
		result.Outcome = "interrupted"
	case errors.Is(ctxErr, context.DeadlineExceeded), timedOut:
		result.Outcome = "timedOut"
	case runErr != nil, snapshot.Iterations != nil && snapshot.Iterations.Failed > 0:
		result.Outcome = "failed"
	default:
		result.Outcome = "passed"
	}
	if runErr != nil {
		result.Error = runErr.Error()
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
