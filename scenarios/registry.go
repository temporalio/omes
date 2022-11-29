package scenarios

import (
	"fmt"
)

var registeredScenarios = []Scenario{}

func registerScenario(scenario Scenario) {
	for _, v := range registeredScenarios {
		if v.Name == scenario.Name {
			panic(fmt.Errorf("duplicate scenario with name: %s", scenario.Name))
		}
	}
	registeredScenarios = append(registeredScenarios, scenario)
}

func GetScenarioByName(name string) (Scenario, error) {
	for _, v := range registeredScenarios {
		if v.Name == name {
			return v, nil
		}
	}
	return Scenario{}, fmt.Errorf("scenario %s not found", name)
}
