# Omes Starter Library (Go)

Library for writing workflow load tests with Omes.

## Structure

```
workflowtests/go/
  starter/       # Shared starter library (run dispatcher, client/worker setup)
  tests/
    simpletest/  # Example test project
```

## Starter / Test Interop

Each test project is a separate Go module that imports the shared starter via a `replace` directive:

```go
// tests/simpletest/go.mod
require github.com/temporalio/omes-workflow-tests/workflowtests/go/starter v0.0.0
replace github.com/temporalio/omes-workflow-tests/workflowtests/go/starter => ../../starter
```

At build time, `omes exec` or `omes workflow` generates a temporary build module that imports both the test and starter, then compiles a single binary.

## SDK Version Override

Both the test and starter have their own `go.temporal.io/sdk` requirement in go.mod. Go's MVS (minimum version selection) resolves a single effective version across both.

When using `--version`:
- **Empty** (default): `go mod tidy` resolves from test/starter go.mod requirements
- **Semver** (e.g. `--version 1.31.0`): sdkbuild appends `replace go.temporal.io/sdk => go.temporal.io/sdk v1.31.0`
- **Local path** (e.g. `--version ../sdk-go`): sdkbuild appends `replace go.temporal.io/sdk => <relative path>`

There is no hard check that starter and test SDK versions match. The effective SDK is whatever the final module graph resolves to, or what `--version` forces via the replace directive.

## Worker Lifecycle

The worker is managed externally by the orchestrator or runtime environment; the starter itself does not expose lifecycle endpoints.

## Handler API

Go tests expose two functions and wire them once via `starter.Run(...)`:

```go
package mytest

import "github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"

func Main() {
	starter.Run(clientMain, workerMain)
}
```

Client handler:

```go
func clientMain(config *starter.ClientConfig) error
```

`ClientConfig` includes:

- `ConnectionOptions` (`go.temporal.io/sdk/client.Options`)
- `TaskQueue`
- `RunID`
- `Iteration`

Core pattern is direct client creation per execute call in `clientMain`.

Optional helper for sharing a client across execute calls:

```go
var pool = starter.NewClientPool()

func clientMain(config *starter.ClientConfig) error {
	c, err := pool.GetOrDial("default", config.ConnectionOptions)
	if err != nil {
		return err
	}
	// use c...
	return nil
}
```

Using `ClientPool` is optional; direct `client.Dial(...)` remains the default pattern.

Worker handler:

```go
func workerMain(config *starter.WorkerConfig) error
```

`WorkerConfig` includes:

- `ConnectionOptions` (`go.temporal.io/sdk/client.Options`)
- `TaskQueue`
- `PromListenAddress`

When `PromListenAddress` is set (via `--prom-listen-address`), starter wires SDK metrics into `ConnectionOptions` and serves Prometheus metrics at `<addr>/metrics`.

The test code still owns Temporal client construction, so it can configure codecs, data converters, interceptors, TLS/headers, and other SDK-native options as needed.

## Usage

```bash
# Build and run a worker
omes exec --language go --project-dir ./tests/simpletest -- worker --task-queue my-queue

# Run a load test (spawns client, optionally spawns worker)
omes workflow --language go --project-dir ./tests/simpletest --spawn-worker --iterations 100
```

## Docker Images

Workflowtest Docker packaging is split into:

1. A language base image (omes + toolchain + starter library)
2. A thin project overlay image (copies one test project)

Build Go base image:

```bash
docker build -f workflowtests/dockerfiles/workflowtest-go.Dockerfile -t omes-workflowtest-go-base:latest .
```

Build a thin project image:

```bash
docker build -f workflowtests/dockerfiles/workflowtest-project.Dockerfile \
  --build-arg BASE_IMAGE=omes-workflowtest-go-base:latest \
  --build-arg TEST_PROJECT=workflowtests/go/tests/simpletest \
  -t omes-workflowtest-go-simpletest:latest .
```

The same image can run either role by command override:

1. Runner: `omes workflow ...`
2. Worker: `omes exec ... -- worker ...`

## Local Modes

We support two clean local modes:

1. Local setup: worker/runner + Prometheus all on host
2. Docker Compose setup: Temporal dev server + UI + Prometheus in Docker, with worker/runner started explicitly

### Mode A: Local Setup Metrics

Local setup metrics setup and export flow live in:

- `workflowtests/dockerfiles/README.md` under `Local Setup`

### Mode B: Docker Compose Setup + Metrics + UI

Docker Compose setup docs (image builds, compose commands, and metrics flow) live in:

- `workflowtests/dockerfiles/README.md`

Docker Compose setup defaults to a dedicated worker container plus explicit runner invocations.
That keeps worker metrics stable and avoids rebuilding/restarting the worker for each test pass.
