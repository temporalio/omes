package harness

import (
	"fmt"

	"go.temporal.io/sdk/contrib/sysinfo"
	sdkworker "go.temporal.io/sdk/worker"
)

const (
	workerProfileEnvVar                   = "OMES_WORKER_PROFILE"
	resourceBasedDefaultWorkerProfile     = "resource-based-default"
	throughputStressBaselineWorkerProfile = "throughput-stress-baseline"
)

type workerProfile struct {
	Options                 sdkworker.Options
	StickyWorkflowCacheSize int
}

func registerWorkerProfile(name string, profile workerProfile) {
	workerProfiles[name] = profile
}

func lookupWorkerProfile(name string) (workerProfile, error) {
	profile, ok := workerProfiles[name]
	if !ok {
		return workerProfile{}, fmt.Errorf("unknown worker profile %q", name)
	}
	return profile, nil
}

var workerProfiles = map[string]workerProfile{}

func init() {
	registerWorkerProfile(resourceBasedDefaultWorkerProfile, workerProfile{
		Options: sdkworker.Options{
			Tuner: mustResourceBasedTuner(0.8, 0.8),
		},
	})
	registerWorkerProfile(throughputStressBaselineWorkerProfile, throughputStressProfile(1))
	// The baseline profile with every limit scaled, e.g. throughput-stress-baseline-x4.
	for _, scale := range throughputStressScales {
		registerWorkerProfile(fmt.Sprintf("%s-x%d", throughputStressBaselineWorkerProfile, scale), throughputStressProfile(scale))
	}
}

var throughputStressScales = []int{2, 4, 8, 16}

func throughputStressProfile(scale int) workerProfile {
	return workerProfile{
		StickyWorkflowCacheSize: 50 * scale,
		Options: sdkworker.Options{
			MaxConcurrentWorkflowTaskExecutionSize:  8 * scale,
			MaxConcurrentActivityExecutionSize:      32 * scale,
			MaxConcurrentLocalActivityExecutionSize: 32 * scale,
			MaxConcurrentWorkflowTaskPollers:        2 * scale,
			MaxConcurrentActivityTaskPollers:        4 * scale,
		},
	}
}

func mustResourceBasedTuner(targetMem, targetCpu float64) sdkworker.WorkerTuner {
	tuner, err := sdkworker.NewResourceBasedTuner(sdkworker.ResourceBasedTunerOptions{
		TargetMem:    targetMem,
		TargetCpu:    targetCpu,
		InfoSupplier: sysinfo.SysInfoProvider(),
	})
	if err != nil {
		panic(err)
	}
	return tuner
}
