package throughputstress

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// WorkflowParams is the single input for the throughput stress workflow.
type WorkflowParams struct {
	// Number of times we should loop through the steps in the workflow.
	Iterations int `json:"iterations"`
	// If true, skip sleeps. This makes workflow end to end latency more informative.
	SkipSleep bool `json:"skipSleep"`
	// What iteration to start on. If we have continued-as-new, we might be starting at a nonzero number.
	InitialIteration int `json:"initialIteration"`
	// If nonzero, we will continue-as-new after specified iteration.
	ContinueAsNewAfterIterCount int `json:"continueAsNewAfterIterCount"`

	// Set internally and incremented every time the workflow spawns a child.
	ChildrenSpawned int `json:"childrenSpawned"`

	// If set, the workflow will run nexus tests.
	// The endpoint should be created ahead of time.
	NexusEndpoint string `json:"nexusEndpoint"`

	// SleepActivities defines the configuration for sleep activities.
	// If not set, no sleep activities will be run.
	SleepActivities *SleepActivityConfig `json:"sleepActivities"`
}

type WorkflowOutput struct {
	// The total number of children that were spawned across all continued runs of the workflow.
	ChildrenSpawned int `json:"childrenSpawned"`
}

// SleepActivityConfig defines the configuration for sleep activities with flexible distribution support
type SleepActivityConfig struct {
	// Distribution of how many sleep activities to run per iteration. Required.
	Count *loadgen.DistributionField[int64] `json:"count"`

	// Map of groups to their configuration. Required.
	Groups map[string]SleepActivityGroupConfig `json:"groups"`
}

// SleepActivityGroupConfig defines a group configuration.
type SleepActivityGroupConfig struct {
	// Weight for this group, used for sampling. Defaults to 1.
	Weight int `json:"weight"`

	// Distribution for sleep duration within this group. Required.
	SleepDuration *loadgen.DistributionField[time.Duration] `json:"sleepDuration"`

	// Distribution for priority keys within this group. Optional.
	PriorityKeys *loadgen.DistributionField[int64] `json:"priorityKeys"`

	// Distribution for fairness keys within this group. Optional.
	FairnessKeys *loadgen.DistributionField[int64] `json:"fairnessKeys"`

	// Distribution for fairness weight within this group. Optional.
	FairnessWeight *loadgen.DistributionField[float32] `json:"fairnessWeight"`
}

func ParseAndValidateSleepActivityConfig(jsonStr string) (*SleepActivityConfig, error) {
	if jsonStr == "" {
		return nil, nil
	}
	config := &SleepActivityConfig{}
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return nil, fmt.Errorf("failed to parse SleepActivityConfig JSON: %w", err)
	}
	if config.Count == nil {
		return nil, fmt.Errorf("SleepActivityConfig: Count field is required")
	}
	if config.Groups == nil || len(config.Groups) == 0 {
		return nil, fmt.Errorf("SleepActivityConfig: Groups field is required and must not be empty")
	}
	for groupID, groupConfig := range config.Groups {
		if groupConfig.Weight < 0 {
			return nil, fmt.Errorf("SleepActivityGroupConfig: Group '%s' Weight must be non-negative", groupID)
		}
		if groupConfig.SleepDuration == nil {
			return nil, fmt.Errorf("SleepActivityGroupConfig: Group '%s' SleepDuration field is required", groupID)
		}
	}
	return config, nil
}

// Sample generates a list of SleepActivityInput instances based on the SleepActivityConfig.
func (config *SleepActivityConfig) Sample() []*kitchensink.ExecuteActivityAction {
	if config == nil {
		return nil
	}

	count, ok := config.Count.Sample()
	if !ok || count <= 0 {
		return nil
	}
	if len(config.Groups) == 0 {
		return nil
	}

	// Create discrete distribution using indices instead of group IDs.
	i := int64(0)
	groupIDs := make([]string, len(config.Groups))
	indexWeights := make(map[int64]int, len(config.Groups))
	for groupID := range config.Groups {
		groupIDs[i] = groupID
		indexWeights[i] = max(1, config.Groups[groupID].Weight)
		i++
	}
	indexDist := loadgen.NewDiscreteDistribution(indexWeights)

	actions := make([]*kitchensink.ExecuteActivityAction, 0, count)
	for range count {
		groupIndex, _ := indexDist.Sample()
		groupConfig := config.Groups[groupIDs[groupIndex]]
		action := &kitchensink.ExecuteActivityAction{
			Priority: &commonpb.Priority{},
		}

		// Pick SleepDuration.
		if duration, ok := groupConfig.SleepDuration.Sample(); ok {
			action.ActivityType = &kitchensink.ExecuteActivityAction_Delay{
				Delay: durationpb.New(duration),
			}
			action.StartToCloseTimeout = durationpb.New(duration + 5*time.Second)
		}

		// Optional: PriorityKeys
		if groupConfig.PriorityKeys != nil {
			if priorityKey, ok := groupConfig.PriorityKeys.Sample(); ok {
				action.Priority.PriorityKey = int32(priorityKey)
			}
		}

		// Optional: FairnessKeys
		if groupConfig.FairnessKeys != nil {
			if fairnessKey, ok := groupConfig.FairnessKeys.Sample(); ok {
				action.FairnessKey = fmt.Sprintf("%d", fairnessKey)
				action.FairnessWeight = 1.0 // always set default
			}
		}

		// Optional: FairnessWeight
		if groupConfig.FairnessWeight != nil {
			if weight, ok := groupConfig.FairnessWeight.Sample(); ok {
				action.FairnessWeight = weight
			}
		}

		actions = append(actions, action)
	}
	return actions
}
