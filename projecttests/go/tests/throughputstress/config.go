package throughputstress

import "time"

type ThroughputStressConfig struct {
	InternalIterations     int    `json:"internal_iterations"`
	ContinueAsNewAfterIter int    `json:"continue_as_new_after_iter"`
	SleepDuration          string `json:"sleep_duration"`
	IncludeRetryScenarios  bool   `json:"include_retry_scenarios"`
	NexusEndpoint          string `json:"nexus_endpoint"` // TODO: not yet used in workflow logic
	PayloadSizeBytes       int    `json:"payload_size_bytes"`
}

func applyDefaults(cfg *ThroughputStressConfig) {
	if cfg.InternalIterations == 0 {
		cfg.InternalIterations = 10
	}
	if cfg.ContinueAsNewAfterIter == 0 {
		cfg.ContinueAsNewAfterIter = 3
	}
	if cfg.SleepDuration == "" {
		cfg.SleepDuration = "1s"
	}
	if cfg.PayloadSizeBytes == 0 {
		cfg.PayloadSizeBytes = 256
	}
}

func parseSleepDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Second
	}
	return d
}
