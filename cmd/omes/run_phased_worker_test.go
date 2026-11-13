package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkerProfileHelpers(t *testing.T) {
	low := "low"
	require.Equal(t, "Pz", workerProfileName(nil))
	require.Equal(t, "low", workerProfileName(&low))
	require.True(t, sameWorkerProfile(nil, nil))
	require.False(t, sameWorkerProfile(nil, &low))
	require.True(t, sameWorkerProfile(&runningPhasedWorker{profile: "low"}, &low))
	require.False(t, sameWorkerProfile(&runningPhasedWorker{profile: "high"}, &low))
}

func TestWaitForWorkerPhaseWithoutWorker(t *testing.T) {
	started := time.Now()
	require.NoError(t, waitForWorkerPhase(context.Background(), started.Add(10*time.Millisecond), nil))
	require.GreaterOrEqual(t, time.Since(started), 10*time.Millisecond)
}

func TestRunPhasedWorkerCommandFlags(t *testing.T) {
	cmd := runPhasedWorkerCmd()
	require.NotNil(t, cmd.Flags().Lookup("phase-plan-file"))
	require.NotNil(t, cmd.Flags().Lookup("start-at"))
	require.True(t, cmd.Flags().Lookup("worker-profile").Hidden)
	require.NotNil(t, cmd.Flags().Lookup("dir-name"))
	require.NotNil(t, cmd.Flags().Lookup("run-id"))
}
