package harness

import (
	"fmt"
	"os"
)

type App struct {
	Worker        WorkerFactory
	LambdaWorker  LambdaWorkerFactory
	ClientFactory ClientFactory
	Project       *ProjectHandlers
}

var dispatchWorkerCLI = runWorkerCLI
var dispatchProjectCLI = runProjectServerCLI
var dispatchLambdaWorker = runLambdaWorker

func Run(app App) error {
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" && app.LambdaWorker != nil {
		return dispatchLambdaWorker(app.LambdaWorker)
	}

	argv := os.Args[1:]
	if len(argv) == 0 {
		return fmt.Errorf("No command specified. Expected 'worker' or 'project-server'")
	}
	switch argv[0] {
	case "worker":
		return dispatchWorkerCLI(app.Worker, app.ClientFactory, argv[1:])
	case "project-server":
		if app.Project == nil {
			return fmt.Errorf("Wanted project-server but no project handlers registered for this app")
		}
		return dispatchProjectCLI(*app.Project, app.ClientFactory, argv[1:])
	default:
		return fmt.Errorf("Unknown command: %q. Expected 'worker' or 'project-server'", argv[:1])
	}
}
