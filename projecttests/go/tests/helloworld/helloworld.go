package helloworld

import "github.com/temporalio/omes/projecttests/go/harness"

// Main is called by the generated entry point (or cmd/main.go for direct run).
func Main() {
	h := harness.New()
	h.RegisterWorker(workerMain)
	h.OnExecute(clientMain)
	h.Run()
}
