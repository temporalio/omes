package loadgen

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// SaturationSample is one snapshot of saturation-related metrics.
type SaturationSample struct {
	CPUPercentRaw  float64
	SlotsUsed      float64
	SlotsAvailable float64
}

// SaturationConfig controls adaptive load behavior.
type SaturationConfig struct {
	Interval         time.Duration // sample + dispatch interval
	EWMAAlpha        float64       // CPU smoothing factor
	TargetCPUPerCore float64       // CPU ceiling per core (percentage)
}

// DefaultSaturationConfig returns sensible defaults for saturation testing.
func DefaultSaturationConfig() SaturationConfig {
	return SaturationConfig{
		Interval:         2 * time.Second,
		EWMAAlpha:        0.3,
		TargetCPUPerCore: 90,
	}
}

// SaturationExecutor drives load by filling available worker slots while
// staying below a CPU ceiling. Each tick it samples metrics from Prometheus
// and spawns work if: slots are available AND CPU is below threshold.
type SaturationExecutor struct {
	Execute func(context.Context, *Run) error
	Sample  func(context.Context) (SaturationSample, error)
	Config  SaturationConfig
}

// prepare validates all configuration and returns ready-to-use configs.
func (s *SaturationExecutor) prepare(info ScenarioInfo) (RunConfiguration, SaturationConfig, error) {
	rc := info.Configuration
	rc.ApplyDefaults()
	if err := rc.Validate(); err != nil {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("invalid scenario: %w", err)
	}
	if s.Execute == nil {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("execute callback is required")
	}
	if s.Sample == nil {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("sample callback is required")
	}
	if rc.Iterations > 0 {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("saturation mode requires --duration, not --iterations")
	}
	if rc.Duration <= 0 {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("saturation mode requires a positive duration")
	}

	sc := s.Config
	defaults := DefaultSaturationConfig()
	if sc.Interval == 0 {
		sc.Interval = defaults.Interval
	}
	if sc.EWMAAlpha == 0 {
		sc.EWMAAlpha = defaults.EWMAAlpha
	}
	if sc.TargetCPUPerCore == 0 {
		sc.TargetCPUPerCore = defaults.TargetCPUPerCore
	}
	if sc.Interval <= 0 {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("interval must be positive")
	}
	if sc.EWMAAlpha <= 0 || sc.EWMAAlpha > 1 {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("EWMA alpha must be in (0,1], got %f", sc.EWMAAlpha)
	}
	if sc.TargetCPUPerCore <= 0 {
		return RunConfiguration{}, SaturationConfig{}, fmt.Errorf("target CPU per core must be positive")
	}

	return rc, sc, nil
}

func (s *SaturationExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	config, satConfig, err := s.prepare(info)
	if err != nil {
		return err
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	if config.Timeout > 0 {
		var cancelTimeout context.CancelFunc
		runCtx, cancelTimeout = context.WithTimeout(runCtx, config.Timeout)
		defer cancelTimeout()
	}
	durationCtx, cancelDuration := context.WithTimeout(runCtx, config.Duration)
	defer cancelDuration()

	executeTimer := info.MetricsHandler.WithTags(map[string]string{
		"scenario": info.ScenarioName,
	}).Timer("omes_execute_histogram")

	var nextIteration int
	if config.StartFromIteration > 0 {
		nextIteration = config.StartFromIteration
	}

	errCh := make(chan error, 1)
	sendErr := func(err error) {
		select {
		case errCh <- err:
		default:
		}
	}

	var workersWG sync.WaitGroup
	var ewmaCPU float64
	var hasEWMA bool
	cpuCeiling := satConfig.TargetCPUPerCore * float64(runtime.NumCPU())

	// Main loop: sample metrics and dispatch work each tick.
	go func() {
		ticker := time.NewTicker(satConfig.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-durationCtx.Done():
				return
			case <-runCtx.Done():
				return
			case <-ticker.C:
				sample, err := s.Sample(runCtx)
				if err != nil {
					if info.Logger != nil {
						info.Logger.Warnf("saturate sample failed: %v", err)
					}
					continue
				}

				ewmaCPU, hasEWMA = nextEWMACPU(ewmaCPU, hasEWMA, sample.CPUPercentRaw, satConfig.EWMAAlpha)

				cpuOK := ewmaCPU < cpuCeiling
				slotsToFill := int(sample.SlotsAvailable)

				if info.Logger != nil {
					info.Logger.Debugf(
						"saturate tick: cpu_raw=%.1f cpu_ewma=%.1f cpu_ceiling=%.1f cpu_ok=%v slots_used=%.0f slots_available=%.0f",
						sample.CPUPercentRaw,
						ewmaCPU,
						cpuCeiling,
						cpuOK,
						sample.SlotsUsed,
						sample.SlotsAvailable,
					)
				}

				if !cpuOK || slotsToFill <= 0 {
					continue
				}

				for i := 0; i < slotsToFill; i++ {
					nextIteration++
					run := info.NewRun(nextIteration)
					workersWG.Add(1)
					go func(run *Run) {
						defer workersWG.Done()

						iterStart := time.Now()
						defer func() {
							executeTimer.Record(time.Since(iterStart))
						}()

						err := s.Execute(runCtx, run)
						if err != nil && config.HandleExecuteError != nil {
							err = config.HandleExecuteError(runCtx, run, err)
						}
						if err != nil {
							sendErr(fmt.Errorf("iteration %d failed: %w", run.Iteration, err))
							return
						}
						if config.OnCompletion != nil {
							config.OnCompletion(runCtx, run)
						}
					}(run)
				}
			}
		}
	}()

	// Wait for duration to expire or an error.
	select {
	case err := <-errCh:
		cancelRun()
		workersWG.Wait()
		return err
	case <-durationCtx.Done():
	}

	// Duration expired — wait for in-flight workers to drain.
	done := make(chan struct{})
	go func() {
		defer close(done)
		workersWG.Wait()
	}()

	for {
		select {
		case err := <-errCh:
			cancelRun()
			<-done
			return err
		case <-done:
			if runCtx.Err() != nil {
				return fmt.Errorf("timed out while waiting for runs to complete: %w", runCtx.Err())
			}
			if info.Logger != nil {
				info.Logger.Info("Saturate run completed")
			}
			return nil
		case <-runCtx.Done():
			cancelRun()
			return fmt.Errorf("timed out while waiting for runs to complete: %w", runCtx.Err())
		}
	}
}

func nextEWMACPU(currentEWMA float64, hasEWMA bool, currentRawCPU float64, alpha float64) (float64, bool) {
	if !hasEWMA {
		return currentRawCPU, true
	}
	return alpha*currentRawCPU + (1-alpha)*currentEWMA, true
}
