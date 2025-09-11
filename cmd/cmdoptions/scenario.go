package cmdoptions

import (
	"encoding/base32"
	"math/rand"
	"strings"

	"github.com/spf13/pflag"
)

type ScenarioID struct {
	Scenario string
	RunID    string
}

func (r *ScenarioID) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.Scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.RunID, "run-id", shortRand(), "Run ID for this run")
}

func shortRand() string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))
}
