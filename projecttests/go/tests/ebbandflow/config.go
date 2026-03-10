package ebbandflow

type ebbAndFlowProjectConfig struct {
	SleepDurationNanos int64 `json:"sleep_duration_nanos"`
}

func applyDefaults(cfg *ebbAndFlowProjectConfig) {
	if cfg.SleepDurationNanos == 0 {
		cfg.SleepDurationNanos = 1_000_000 // 1ms
	}
}
