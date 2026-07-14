package loadgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// PayloadConfig configures activity payload sizes using the shared distribution
// framework. Parsed from a JSON string passed via --option (with @file.json support).
type PayloadConfig struct {
	// Size is the distribution (in bytes) of activity payload sizes. See DistributionField
	// for the supported distribution types and their JSON format.
	Size *DistributionField[int32] `json:"size"`
}

// ParseAndValidatePayloadConfig parses a PayloadConfig from JSON. An empty input indicates
// that no payload config was supplied, and (nil, nil) is returned.
func ParseAndValidatePayloadConfig(jsonStr string) (*PayloadConfig, error) {
	if jsonStr == "" {
		return nil, nil
	}
	config := &PayloadConfig{}
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return nil, fmt.Errorf("failed to parse PayloadConfig JSON: %w", err)
	}
	return config, nil
}

// SamplePayloadSize samples an activity payload size (in bytes) from the configured Size
// distribution. If c is nil or Size is unset, fallback is returned. The result is clamped to
// [0, math.MaxInt32] to match bytes_to_receive etc in the kitchensink proto.
func (c *PayloadConfig) SamplePayloadSize(rng *rand.Rand, fallback int32) int32 {
	if c == nil || c.Size == nil {
		return fallback
	}
	value, ok := c.Size.Sample(rng)
	if !ok {
		return fallback
	}
	if value < 0 {
		value = 0
	}
	return value
}
