package phaseplan

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	plan, err := Parse(`{
		"phases":[
			{"name":"load","duration":"70s","addRate":12000,"workerProfile":null},
			{"name":"drain","duration":"7m","addRate":0,"workerProfile":"high"}
		]
	}`)
	require.NoError(t, err)
	require.Equal(t, 8*time.Minute+10*time.Second, plan.TotalDuration())
	require.Nil(t, plan.Phases[0].WorkerProfile)
	require.Equal(t, "high", *plan.Phases[1].WorkerProfile)
	startAt := time.Date(2099, 1, 2, 3, 4, 5, 0, time.UTC)
	require.Equal(t, startAt.Add(70*time.Second), plan.PhaseStart(startAt, 1))
	require.Equal(t, startAt.Add(8*time.Minute+10*time.Second), plan.PhaseEnd(startAt, 1))
}

func TestParseRejectsInvalidPlans(t *testing.T) {
	validPhase := `{"name":"load","duration":"1s","addRate":1,"workerProfile":null}`
	tests := map[string]string{
		"empty phases":           `{"phases":[]}`,
		"missing add rate":       `{"phases":[{"name":"load","duration":"1s","workerProfile":null}]}`,
		"missing worker profile": `{"phases":[{"name":"load","duration":"1s","addRate":1}]}`,
		"empty worker profile":   `{"phases":[{"name":"load","duration":"1s","addRate":1,"workerProfile":""}]}`,
		"all zero":               `{"phases":[{"name":"drain","duration":"1s","addRate":0,"workerProfile":null}]}`,
		"unknown field":          `{"phases":[` + validPhase + `],"startAt":"2099-01-02T03:04:05Z"}`,
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(value)
			require.Error(t, err)
		})
	}
}

func TestWaitUntil(t *testing.T) {
	started := time.Now()
	require.NoError(t, WaitUntil(context.Background(), started.Add(10*time.Millisecond)))
	require.GreaterOrEqual(t, time.Since(started), 10*time.Millisecond)
	require.NoError(t, WaitUntil(context.Background(), time.Now().Add(-time.Second)))
	require.Error(t, WaitForStart(context.Background(), time.Now().Add(-time.Second)))
}
