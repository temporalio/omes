package loadgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// SleepActivityConfig defines the configuration for sleep activities with flexible distribution support
type SleepActivityConfig struct {
	// Distribution of how many sleep activities to run per iteration. Required.
	Count *DistributionField[int64] `json:"count"`

	// Map of groups to their configuration. Required.
	Groups map[string]SleepActivityGroupConfig `json:"groups"`
}

// SleepActivityGroupConfig defines a group configuration.
type SleepActivityGroupConfig struct {
	// Weight for this group, used for sampling. Defaults to 1.
	Weight int `json:"weight"`

	// Distribution for sleep duration within this group. Required.
	SleepDuration *DistributionField[time.Duration] `json:"sleepDuration"`

	// Distribution for priority keys within this group. Optional.
	PriorityKeys *DistributionField[int64] `json:"priorityKeys"`

	// Distribution for fairness keys within this group. Optional.
	FairnessKeys *DistributionField[int64] `json:"fairnessKeys"`

	// Distribution for fairness weight within this group. Optional.
	FairnessWeight *DistributionField[float32] `json:"fairnessWeight"`
}

func ParseAndValidateSleepActivityConfig(jsonStr string, requireCount, requireSleepDuration bool) (*SleepActivityConfig, error) {
	if jsonStr == "" {
		return nil, nil
	}
	config := &SleepActivityConfig{}
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return nil, fmt.Errorf("failed to parse SleepActivityConfig JSON: %w", err)
	}
	if requireCount && config.Count == nil {
		return nil, fmt.Errorf("SleepActivityConfig: Count field is required")
	}
	if config.Groups == nil || len(config.Groups) == 0 {
		return nil, fmt.Errorf("SleepActivityConfig: Groups field is required and must not be empty")
	}
	for groupID, groupConfig := range config.Groups {
		if groupConfig.Weight < 0 {
			return nil, fmt.Errorf("SleepActivityGroupConfig: Group '%s' Weight must be non-negative", groupID)
		}
		if requireSleepDuration && groupConfig.SleepDuration == nil {
			return nil, fmt.Errorf("SleepActivityGroupConfig: Group '%s' SleepDuration field is required", groupID)
		}
	}
	return config, nil
}

// Sample generates a list of SleepActivityInput instances based on the SleepActivityConfig.
func (config *SleepActivityConfig) Sample(rng *rand.Rand) []*kitchensink.ExecuteActivityAction {
	if config == nil {
		return nil
	}

	count, ok := config.Count.Sample(rng)
	if !ok || count <= 0 {
		return nil
	}
	if len(config.Groups) == 0 {
		return nil
	}

	// Index the groups.
	groupIDs := make([]string, 0, len(config.Groups))
	for groupID := range config.Groups {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Strings(groupIDs)

	// Create discrete distribution of the groups according to their weights.
	indexWeights := make(map[int64]int, len(config.Groups))
	for groupIdx, groupID := range groupIDs {
		indexWeights[int64(groupIdx)] = max(1, config.Groups[groupID].Weight)
	}
	indexDist := NewDiscreteDistribution(indexWeights)

	actions := make([]*kitchensink.ExecuteActivityAction, 0, count)
	for range count {
		groupIdx, _ := indexDist.Sample(rng)
		groupConfig := config.Groups[groupIDs[groupIdx]]
		action := &kitchensink.ExecuteActivityAction{
			Priority: &commonpb.Priority{},
		}

		// Pick SleepDuration.
		if duration, ok := groupConfig.SleepDuration.Sample(rng); ok {
			action.ActivityType = &kitchensink.ExecuteActivityAction_Delay{
				Delay: durationpb.New(duration),
			}
			action.StartToCloseTimeout = durationpb.New(duration + 5*time.Second)
		}

		// Optional: PriorityKeys
		if groupConfig.PriorityKeys != nil {
			if priorityKey, ok := groupConfig.PriorityKeys.Sample(rng); ok {
				action.Priority.PriorityKey = int32(priorityKey)
			}
		}

		// Optional: FairnessKeys
		if groupConfig.FairnessKeys != nil {
			if fairnessKey, ok := groupConfig.FairnessKeys.Sample(rng); ok {
				action.FairnessKey = fmt.Sprintf("%d", fairnessKey)
				action.FairnessWeight = 1.0 // always set default
			}
		}

		// Optional: FairnessWeight
		if groupConfig.FairnessWeight != nil {
			if weight, ok := groupConfig.FairnessWeight.Sample(rng); ok {
				action.FairnessWeight = weight
			}
		}

		actions = append(actions, action)
	}
	return actions
}
