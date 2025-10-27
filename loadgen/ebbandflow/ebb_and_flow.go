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
	ScheduleToStart time.Duration `json:"d"`
}
