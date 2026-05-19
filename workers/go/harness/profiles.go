package harness

import (
	"fmt"

	"go.temporal.io/sdk/contrib/sysinfo"
	sdkworker "go.temporal.io/sdk/worker"
)

const (
	workerProfileEnvVar               = "OMES_WORKER_PROFILE"
	resourceBasedDefaultWorkerProfile = "resource-based-default"
)

var workerProfiles = map[string]sdkworker.Options{}

func registerWorkerProfile(name string, profile sdkworker.Options) {
	workerProfiles[name] = profile
}

func lookupWorkerProfile(name string) (sdkworker.Options, error) {
	profile, ok := workerProfiles[name]
	if !ok {
		return sdkworker.Options{}, fmt.Errorf("unknown worker profile %q", name)
	}
	return profile, nil
}

func init() {
	tuner, err := sdkworker.NewResourceBasedTuner(sdkworker.ResourceBasedTunerOptions{
		TargetMem:    0.8,
		TargetCpu:    0.8,
		InfoSupplier: sysinfo.SysInfoProvider(),
	})
	if err != nil {
		panic(err)
	}
	registerWorkerProfile(resourceBasedDefaultWorkerProfile, sdkworker.Options{
		Tuner: tuner,
	})
}
