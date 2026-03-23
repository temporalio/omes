package helloworld

import (
	harness "github.com/temporalio/omes/workers/go/projects/harness"
	"go.temporal.io/sdk/client"
)

func Main() {
	h := harness.New()
	h.RegisterClient(func(opts client.Options, _ harness.ClientConfig) (client.Client, error) {
		return client.Dial(opts)
	})
	h.RegisterWorker(workerMain)
	h.OnExecute(clientMain)
	h.Run()
}
