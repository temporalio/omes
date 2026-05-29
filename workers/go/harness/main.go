package harness

import (
	"errors"
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
	argv := os.Args[1:]
	if len(argv) == 0 {
		return errors.New("No command specified. Expected 'worker' or 'project-server'")
	}
	switch argv[0] {
	case "worker":
		if app.Worker == nil && app.LambdaWorker == nil {
			return errors.New("worker or lambda worker factory is required")
		}
		if app.Worker != nil && app.LambdaWorker != nil {
			return errors.New("worker and lambda worker factories are mutually exclusive")
		}
		if app.Worker != nil {
			return dispatchWorkerCLI(app.Worker, app.ClientFactory, argv[1:])
		}
		return dispatchLambdaWorker(app.LambdaWorker)
	case "project-server":
		if app.Project == nil {
			return errors.New(
				"Wanted project-server but no project handlers registered for this app",
			)
		}
		return dispatchProjectCLI(*app.Project, app.ClientFactory, argv[1:])
	default:
		return fmt.Errorf("Unknown command: %q. Expected 'worker' or 'project-server'", argv[:1])
	}
}
