package scenario

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

var registeredScenarios = make(map[string]*Scenario)

// MustRegister resgister a scenario in the global static registry.
// Panics if registration fails.
// If scenario name is empty, the filename of the caller will be used as the name.
func MustRegister(scenario *Scenario) {
	if scenario.Name == "" {
		_, file, _, ok := runtime.Caller(1)
		if !ok {
			panic("Could not infer caller when registering a nameless scenario")
		}
		scenario.Name = strings.Replace(filepath.Base(file), ".go", "", 1)
	}
	_, found := registeredScenarios[scenario.Name]
	if found {
		panic(fmt.Errorf("duplicate scenario with name: %s", scenario.Name))
	}
	registeredScenarios[scenario.Name] = scenario
}

// Get a scenario by name from the global static registry
func Get(name string) *Scenario {
	return registeredScenarios[name]
}
