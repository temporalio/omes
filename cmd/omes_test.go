package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/omes/scenarios"
)

func TestApplyOverrides(t *testing.T) {
	// override duration
	{
		scenario := scenarios.Scenario{Iterations: 3, Duration: 3 * time.Second}
		a := App{appOptions: appOptions{duration: 5 * time.Second}}
		err := a.applyOverrides(&scenario)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), scenario.Iterations)
		assert.Equal(t, 5*time.Second, scenario.Duration)
	}
	// override iterations
	{
		scenario := scenarios.Scenario{Iterations: 3, Duration: 3 * time.Second}
		a := App{appOptions: appOptions{iterations: 5}}
		err := a.applyOverrides(&scenario)
		assert.NoError(t, err)
		assert.Equal(t, uint32(5), scenario.Iterations)
		assert.Equal(t, time.Duration(0), scenario.Duration)
	}
	// override both
	{
		scenario := scenarios.Scenario{Iterations: 3, Duration: 3 * time.Second}
		a := App{appOptions: appOptions{iterations: 5, duration: 5 * time.Second}}
		err := a.applyOverrides(&scenario)
		assert.ErrorContains(t, err, "invalid options: iterations and duration are mutually exclusive")
	}
	// override none
	{
		scenario := scenarios.Scenario{Iterations: 3, Duration: 3 * time.Second}
		a := App{appOptions: appOptions{}}
		err := a.applyOverrides(&scenario)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), scenario.Iterations)
		assert.Equal(t, 3*time.Second, scenario.Duration)
	}
}
