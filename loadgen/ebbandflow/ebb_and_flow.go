package ebbandflow

import (
	"time"

	"github.com/temporalio/omes/loadgen"
)

type WorkflowParams struct {
	SleepActivities *loadgen.SleepActivityConfig `json:"sleepActivities"`
}

type WorkflowOutput struct {
	Timings []ActivityTiming `json:"timings"`
}

type ActivityTiming struct {
	FairnessKey       string        `json:"fairnessKey"`
	FairnessWeight    float32       `json:"fairnessWeight"`
	ScheduleToStartMS time.Duration `json:"scheduleToStartMS"`
}
