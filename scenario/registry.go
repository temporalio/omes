package scenario

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

var registeredScenarios = make(map[string]*Scenario)

// MustRegister registers a scenario in the global static registry.
// Panics if registration fails.
// The file name of the caller is be used as the scenario name.
func MustRegister(scenario *Scenario) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("Could not infer caller when registering a nameless scenario")
	}
	scenarioName := strings.Replace(filepath.Base(file), ".go", "", 1)
	_, found := registeredScenarios[scenarioName]
	if found {
		panic(fmt.Errorf("duplicate scenario with name: %s", scenarioName))
	}
	registeredScenarios[scenarioName] = scenario
}

// Get a scenario by name from the global static registry.
func Get(name string) *Scenario {
	return registeredScenarios[name]
}
