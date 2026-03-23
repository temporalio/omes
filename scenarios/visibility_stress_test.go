package scenarios

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

// TestVisibilityStressConfigure tests the configuration/validation logic without
// running any workflows.
func TestVisibilityStressConfigure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		options     map[string]string
		config      loadgen.RunConfiguration
		expectError string
	}{
		{
			name:        "no presets → error",
			options:     map[string]string{},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "at least one of loadPreset or queryPreset must be set",
		},
		{
			name:        "iterations not supported",
			options:     map[string]string{"loadPreset": "light"},
			config:      loadgen.RunConfiguration{Iterations: 10},
			expectError: "does not support --iterations",
		},
		{
			name:        "duration required",
			options:     map[string]string{"loadPreset": "light"},
			config:      loadgen.RunConfiguration{},
			expectError: "requires --duration",
		},
		{
			name:        "unknown loadPreset",
			options:     map[string]string{"loadPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown loadPreset",
		},
		{
			name:        "unknown queryPreset",
			options:     map[string]string{"queryPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown queryPreset",
		},
		{
			name:        "unknown csaPreset",
			options:     map[string]string{"loadPreset": "light", "csaPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown csaPreset",
		},
		{
			name:        "read-only without csaPreset → error",
			options:     map[string]string{"queryPreset": "light"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "csaPreset is required in read-only mode",
		},
		{
			name:        "failPercent + timeoutPercent >= 1.0 → error",
			options:     map[string]string{"loadPreset": "light", "failPercent": "0.6", "timeoutPercent": "0.5"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "failPercent + timeoutPercent must be < 1.0",
		},
		{
			name:        "retention too short → error",
			options:     map[string]string{"loadPreset": "light", "retention": "1h"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "retention must be >= 24h",
		},
		{
			name:    "valid write-only config",
			options: map[string]string{"loadPreset": "light"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:    "valid read-only config",
			options: map[string]string{"queryPreset": "light", "csaPreset": "small"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:    "valid full config with overrides",
			options: map[string]string{"loadPreset": "moderate", "queryPreset": "light", "csaPreset": "heavy", "wfRPS": "50", "deleteRPS": "0"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:    "cleanup mode skips preset validation",
			options: map[string]string{"cleanup": "true"},
			config:  loadgen.RunConfiguration{},
		},
		{
			name:    "loadPreset defaults csaPreset to medium",
			options: map[string]string{"loadPreset": "light"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executor := &visibilityStressExecutor{}
			info := loadgen.ScenarioInfo{
				RunID:           "test-run",
				Configuration:   tc.config,
				ScenarioOptions: tc.options,
				Namespace:       "default",
			}
			err := executor.Configure(info)
			if tc.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCSAParsing tests the CSA name prefix parsing logic.
func TestCSAParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectType  csaType
		expectError bool
	}{
		{"int", "VS_Int_01", csaTypeInt, false},
		{"keyword", "VS_Keyword_01", csaTypeKeyword, false},
		{"bool", "VS_Bool_01", csaTypeBool, false},
		{"double", "VS_Double_01", csaTypeDouble, false},
		{"text", "VS_Text_01", csaTypeText, false},
		{"datetime", "VS_Datetime_01", csaTypeDatetime, false},
		{"unknown prefix", "VS_Foo_01", 0, true},
		{"no prefix", "SomeAttribute", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, err := parseCSAName(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectType, def.Type)
				assert.Equal(t, tc.input, def.Name)
			}
		})
	}
}

// TestBuildWorkflowInput tests the workflow input builder logic.
func TestBuildWorkflowInput(t *testing.T) {
	t.Parallel()

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Load: &vsLoadPreset{
				UpdatesPerWF:   3,
				UpdateDelay:    100 * time.Millisecond,
				FailPercent:    0,
				TimeoutPercent: 0,
			},
			CSADefs: []csaDef{
				{Name: "VS_Int_01", Type: csaTypeInt},
				{Name: "VS_Keyword_01", Type: csaTypeKeyword},
				{Name: "VS_Bool_01", Type: csaTypeBool},
			},
		},
		rng: rand.New(rand.NewSource(42)),
	}

	input := executor.buildWorkflowInput()

	// With updatesPerWF=3 (no fractional part), we should get exactly 3 groups.
	assert.Equal(t, 3, len(input.CSAUpdates))
	assert.Equal(t, 100*time.Millisecond, input.Delay)
	assert.False(t, input.ShouldFail)
	assert.False(t, input.ShouldTimeout)

	// Each group should have 1-3 attributes.
	for _, group := range input.CSAUpdates {
		assert.GreaterOrEqual(t, len(group.Attributes), 1)
		assert.LessOrEqual(t, len(group.Attributes), 3)
	}
}

// TestBuildWorkflowInputFractional tests fractional updatesPerWF.
func TestBuildWorkflowInputFractional(t *testing.T) {
	t.Parallel()

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Load: &vsLoadPreset{
				UpdatesPerWF:   0.5, // 50% get 1 update, 50% get 0
				UpdateDelay:    time.Second,
				FailPercent:    0,
				TimeoutPercent: 0,
			},
			CSADefs: []csaDef{
				{Name: "VS_Int_01", Type: csaTypeInt},
			},
		},
		rng: rand.New(rand.NewSource(42)),
	}

	// Generate many inputs and check the distribution.
	var withUpdates, withoutUpdates int
	for i := 0; i < 1000; i++ {
		input := executor.buildWorkflowInput()
		if len(input.CSAUpdates) > 0 {
			withUpdates++
		} else {
			withoutUpdates++
		}
	}

	// With 0.5, roughly half should have updates. Allow wide margin.
	assert.InDelta(t, 500, withUpdates, 100, "expected ~50%% to have updates")
	assert.InDelta(t, 500, withoutUpdates, 100, "expected ~50%% to have no updates")
}

// TestQueryGeneration tests that generated queries reference only CSAs from the preset.
func TestQueryGeneration(t *testing.T) {
	t.Parallel()

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Query: &vsQueryPreset{
				CountRPS:              1,
				ListRPS:               1,
				ListNoFilterWeight:    1,
				ListOpenWeight:        1,
				ListClosedWeight:      1,
				ListSimpleCSAWeight:   5, // bias toward CSA filters
				ListCompoundCSAWeight: 5,
			},
			CSADefs: []csaDef{
				{Name: "VS_Int_01", Type: csaTypeInt},
				{Name: "VS_Keyword_01", Type: csaTypeKeyword},
			},
		},
		rng: rand.New(rand.NewSource(42)),
	}

	totalWeight := 1 + 1 + 1 + 5 + 5

	// Generate many queries and verify they only reference known CSAs.
	for i := 0; i < 100; i++ {
		filter := executor.generateFilter(false, totalWeight)

		// Should never reference VS_Int_02, VS_Keyword_02, etc. (not in preset).
		assert.NotContains(t, filter, "VS_Int_02")
		assert.NotContains(t, filter, "VS_Keyword_02")
		assert.NotContains(t, filter, "VS_Bool_")
		assert.NotContains(t, filter, "VS_Double_")
		assert.NotContains(t, filter, "VS_Text_")
		assert.NotContains(t, filter, "VS_Datetime_")
	}
}

// TestComputeTimeout tests the execution timeout calculation.
func TestComputeTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5*time.Second, vsComputeTimeout(0))
	assert.Equal(t, 5*time.Second, vsComputeTimeout(1))
	assert.Equal(t, 5*time.Second, vsComputeTimeout(2))
	assert.Equal(t, 6*time.Second, vsComputeTimeout(3))
	assert.Equal(t, 20*time.Second, vsComputeTimeout(10))
}

// TestVisibilityStressWriteOnly runs the full scenario in write-only mode
// against a real dev server. This is the integration test.
func TestVisibilityStressWriteOnly(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(1*time.Minute))

	executor := &visibilityStressExecutor{}
	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("vs-test-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Duration: 5 * time.Second,
		},
		ScenarioOptions: map[string]string{
			"loadPreset":    "light",
			"csaPreset":     "small",
			"wfRPS":         "5",   // low rate for test speed
			"updatesPerWF":  "2",   // few updates per WF
			"deleteRPS":     "0",   // no deletes (keep test simple)
			"failPercent":   "0",   // no failures
			"timeoutPercent": "0",  // no timeouts
		},
	}

	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	// Verify some workflows were created.
	require.Greater(t, executor.totalCreated.Load(), int64(0),
		"Should have created at least one workflow")

	t.Logf("Created %d workflows, errors: %d",
		executor.totalCreated.Load(), executor.totalErrors.Load())
}
