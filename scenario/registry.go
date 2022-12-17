package scenario

import (
	"fmt"
)

var registeredScenarios = make(map[string]*Scenario)

// Register a scenario in the global static registry
func Register(scenario *Scenario) {
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
