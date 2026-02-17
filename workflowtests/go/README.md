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
