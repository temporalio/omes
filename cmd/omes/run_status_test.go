package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestRunStatus(t *testing.T) {
	for _, exportFile := range []bool{false, true} {
		name := "without result file"
		if exportFile {
			name = "with result file"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			started := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 1)
			executor := &statusExecutor{}
			executor.Execute = func(ctx context.Context, run *loadgen.Run) error {
				executor.state.Add(1)
				if run.Iteration == 2 {
					close(started)
					select {
					case <-release:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			}
			runner := &scenarioRunner{counter: &loadgen.RunCounter{}, state: scenarioState(executor)}
			if exportFile {
				runner.exportResultFile = filepath.Join(t.TempDir(), "result.json")
			}
			handler := runner.statusHandler()
			readStatus := func() string {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/status", nil))
				require.Equal(t, http.StatusOK, response.Code)
				require.Equal(t, "application/json", response.Header().Get("Content-Type"))
				return response.Body.String()
			}

			go func() {
				var runErr error
				func() {
					defer runner.finish(ctx, &runErr)
					runErr = executor.Run(ctx, loadgen.ScenarioInfo{
						Logger:         zap.NewNop().Sugar(),
						MetricsHandler: client.MetricsNopHandler,
						Configuration: loadgen.RunConfiguration{
							Counter: runner.counter, Iterations: 2, MaxConcurrent: 1,
							DoNotRegisterSearchAttributes: true,
						},
					})
				}()
				done <- runErr
			}()
			<-started
			require.JSONEq(t, `{"outcome":"running","iterations":{"started":2,"succeeded":1,"failed":0},"failures":[],"scenarioState":2}`, readStatus())

			close(release)
			require.NoError(t, <-done)
			final := readStatus()
			require.JSONEq(t, `{"outcome":"passed","iterations":{"started":2,"succeeded":2,"failed":0},"failures":[],"scenarioState":2}`, final)
			if exportFile {
				data, err := os.ReadFile(runner.exportResultFile)
				require.NoError(t, err)
				require.JSONEq(t, string(data), final)
			}

			executor.state.Store(99)
			require.JSONEq(t, final, readStatus())
		})
	}
}

type statusExecutor struct {
	loadgen.GenericExecutor
	state atomic.Int64
}

func (e *statusExecutor) Snapshot() any { return e.state.Load() }
