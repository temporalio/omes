package main

import (
	"context"
	"io"
	"net/http"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestStatusServerFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{runScenarioCmd(), runScenarioWithWorkerCmd()} {
		t.Run(cmd.Name(), func(t *testing.T) {
			require.Equal(t, "", cmd.Flags().Lookup("status-listen-address").DefValue)
			require.NoError(t, cmd.ParseFlags([]string{"--status-listen-address", "127.0.0.1:0"}))
		})
	}
}

func TestStatusServerStatusLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := &scenarioRunner{
		scenarioRunConfig: scenarioRunConfig{statusListenAddress: "127.0.0.1:0"},
		logger:            zap.NewNop().Sugar(),
	}
	require.NoError(t, runner.startStatusServer())
	t.Cleanup(runner.shutdownStatusServer)
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	executor := &statusExecutor{}
	executor.Execute = func(ctx context.Context, _ *loadgen.Run) error {
		executor.state.Store(1)
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	runner.stateMu.Lock()
	runner.state = scenarioState(executor)
	runner.stateMu.Unlock()
	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	httpClient := &http.Client{Transport: transport}
	url := "http://" + runner.status.server.Addr + "/status"
	readStatus := func() string {
		response, err := httpClient.Get(url)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "application/json", response.Header.Get("Content-Type"))
		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		return string(data)
	}

	go func() {
		var runErr error
		func() {
			defer runner.finish(ctx, &runErr)
			runErr = executor.Run(ctx, loadgen.ScenarioInfo{
				Logger:         runner.logger,
				MetricsHandler: client.MetricsNopHandler,
				Configuration: loadgen.RunConfiguration{
					Counter: runner.counter, Iterations: 1, MaxConcurrent: 1,
					DoNotRegisterSearchAttributes: true,
				},
			})
		}()
		done <- runErr
	}()
	<-started
	require.JSONEq(t, `{"outcome":"running","iterations":{"started":1,"succeeded":0,"failed":0},"failures":[],"scenarioState":1}`, readStatus())

	close(release)
	require.NoError(t, <-done)
	require.JSONEq(t, `{"outcome":"passed","iterations":{"started":1,"succeeded":1,"failed":0},"failures":[],"scenarioState":1}`, readStatus())

	runner.shutdownStatusServer()
	_, err := httpClient.Get(url)
	require.ErrorIs(t, err, syscall.ECONNREFUSED)
}

func TestStatusServerListenFailure(t *testing.T) {
	for _, withWorker := range []bool{false, true} {
		name := "scenario"
		if withWorker {
			name = "scenario with worker"
		}
		t.Run(name, func(t *testing.T) {
			runner := newRunner("workflow_with_single_noop_activity")
			runner.logger = zap.NewNop().Sugar()
			runner.statusListenAddress = "127.0.0.1:0"
			require.NoError(t, runner.startStatusServer())
			t.Cleanup(runner.shutdownStatusServer)
			config := scenarioRunConfig{statusListenAddress: runner.status.server.Addr}

			var err error
			if withWorker {
				worker := workerWithScenarioRunner{scenarioRunConfig: config}
				worker.ScenarioID = runner.scenario
				worker.Logger = runner.logger
				err = worker.run(t.Context())
			} else {
				runner := newRunner("workflow_with_single_noop_activity")
				runner.logger = zap.NewNop().Sugar()
				runner.scenarioRunConfig = config
				err = runner.run(t.Context())
			}
			var usage loadgen.UsageError
			require.ErrorAs(t, err, &usage)
			require.ErrorContains(t, err, "--status-listen-address")
			require.ErrorContains(t, err, "address already in use")
		})
	}
}
