package clioptions

import (
	"encoding/base32"
	"math/rand"
	"strings"

	"github.com/spf13/pflag"
)

type ScenarioID struct {
	Scenario  string
	RunID     string
	RunFamily string
}

func (r *ScenarioID) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.Scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.RunID, "run-id", shortRand(), "Run ID for this run")
	fs.StringVar(&r.RunFamily, "run-family", "", "Human-readable identifier for grouping related runs")
}

func shortRand() string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))
}
