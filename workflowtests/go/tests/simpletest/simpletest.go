package simpletest

import "github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"

// Main is called by the generated entry point (or cmd/main.go for direct run).
func Main() {
	starter.Run(clientMain, workerMain)
}
