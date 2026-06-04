package harness

import (
	"os"

	"go.temporal.io/sdk/contrib/aws/lambdaworker"
	sdkworker "go.temporal.io/sdk/worker"
)

// LambdaWorkerFactory configures a Temporal Lambda worker. The AWS Lambda
// worker runtime owns client/worker construction and shutdown.
type LambdaWorkerFactory func(*lambdaworker.Options) error

func runLambdaWorker(factory LambdaWorkerFactory) error {
	if factory == nil {
		return nil
	}
	version := sdkworker.WorkerDeploymentVersion{
		DeploymentName: os.Getenv("TEMPORAL_OMES_DEPLOYMENT_NAME"),
		BuildID:        os.Getenv("TEMPORAL_OMES_BUILD_ID"),
	}
	lambdaworker.RunWorker(version, factory)
	return nil
}
