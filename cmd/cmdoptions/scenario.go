package cmdoptions

import (
	"encoding/base32"
	"math/rand"
	"strings"

	"github.com/spf13/pflag"
)

type ScenarioID struct {
	scenario string
	runID    string
}

func (r *ScenarioID) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.runID, "run-id", shortRand(), "Run ID for this run")
}

func (r *ScenarioID) Scenario() string {
	return r.scenario
}

func (r *ScenarioID) RunID() string {
	return r.runID
}

func shortRand() string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))
}
