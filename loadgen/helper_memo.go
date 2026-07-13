package loadgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// MaxMemoBytes caps the per-workflow memo blob to stay below the server's blob-size limit.
const MaxMemoBytes = 40 * 1024 // 40 KB

// MemoConfig configures a probabilistic, distribution-sized memo blob (under the key
// "MemoBlob") attached to workflows throughout the tree: the root workflow, child workflows,
// and continue-as-new runs. Parsed from a JSON string passed via --option (with @file.json
// support).
//
// Semantics: Probability is rolled independently per node. Child workflows are independent
// of each other and of the root. Along a continue-as-new chain the memo is sticky/additive:
// once a run in the chain has a memo, it is carried forward to subsequent runs (refreshed
// when a later roll hits), so it never disappears mid-chain. The carry is applied
// client-side, so behavior is identical across all worker SDK languages regardless of their
// native continue-as-new memo carry-forward defaults.
type MemoConfig struct {
	// Probability (0..1) that a given workflow node carries a memo. Rolled independently
	// per node.
	Probability float64 `json:"probability"`
	// Size is the distribution (in bytes) of the memo blob when present. See DistributionField
	// for the supported distribution types and their JSON format. Required when Probability > 0.
	Size *DistributionField[int64] `json:"size"`
}

// ParseAndValidateMemoConfig parses a MemoConfig from JSON. An empty input indicates that no
// memo config was supplied, and (nil, nil) is returned.
func ParseAndValidateMemoConfig(jsonStr string) (*MemoConfig, error) {
	if jsonStr == "" {
		return nil, nil
	}
	config := &MemoConfig{}
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return nil, fmt.Errorf("failed to parse MemoConfig JSON: %w", err)
	}
	if config.Probability < 0 || config.Probability > 1 {
		return nil, fmt.Errorf("MemoConfig: probability must be in [0, 1], got %v", config.Probability)
	}
	if config.Probability > 0 && config.Size == nil {
		return nil, fmt.Errorf("MemoConfig: size is required when probability > 0")
	}
	return config, nil
}

// SampleMemoBytes rolls Probability and, if selected, returns an incompressible pseudo-random
// blob sized from Size (clamped to [1, MaxMemoBytes]). Returns nil when no memo config is set,
// the roll misses, or the size sample fails. All randomness is drawn from rng.
func (c *MemoConfig) SampleMemoBytes(rng *rand.Rand) []byte {
	if c == nil || c.Size == nil || c.Probability <= 0 {
		return nil
	}
	if rng.Float64() >= c.Probability {
		return nil
	}
	size, ok := c.Size.Sample(rng)
	if !ok {
		return nil
	}
	if size < 1 {
		size = 1
	}
	if size > MaxMemoBytes {
		size = MaxMemoBytes
	}
	blob := make([]byte, size)
	_, _ = rng.Read(blob)
	return blob
}
