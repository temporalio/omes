package ebbandflow

import (
	"encoding/json"

	"github.com/temporalio/omes/projecttests/go/harness"
)

var cfg ebbAndFlowProjectConfig

func Main() {
	h := harness.New()
	h.RegisterWorker(workerMain)
	h.OnInit(func(init *harness.InitConfig) error {
		if len(init.ProjectConfig) > 0 {
			if err := json.Unmarshal(init.ProjectConfig, &cfg); err != nil {
				return err
			}
		}
		applyDefaults(&cfg)
		return nil
	})
	h.OnExecute(clientMain)
	h.Run()
}
