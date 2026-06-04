package apps

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/workers/go/apps/lambda"
	"github.com/temporalio/omes/workers/go/apps/worker"
	"github.com/temporalio/omes/workers/go/harness"
)

const defaultAppName = "worker"

var registry = map[string]harness.App{
	"lambda": lambda.App,
	"worker": worker.App,
}

// Main selects the registered app before handing off to the harness.
func Main() {
	var appName string

	flags := pflag.NewFlagSet("go-worker-app", pflag.ContinueOnError)
	flags.SetInterspersed(false)
	flags.StringVar(&appName, "app", defaultAppName, "Go worker app")
	if err := flags.Parse(os.Args[1:]); err != nil {
		clioptions.BackupLogger.Fatal(err)
	}

	app, ok := registry[appName]
	if !ok {
		clioptions.BackupLogger.Fatalf("unknown Go worker app %q", appName)
	}

	os.Args = append(os.Args[:1], flags.Args()...)
	if err := harness.Run(app); err != nil {
		clioptions.BackupLogger.Fatal(err)
	}
}
