# Omes - a load generator for Temporal

This project is for testing load generation scenarios against Temporal. This is primarily used by the Temporal team to
benchmark features and situations. Backwards compatibility may not be maintained.

## Why the weird name?

Omes (pronounced oh-mess) is the Hebrew word for "load" (עומס).

## Prerequisites

- [Go](https://golang.org/) 1.25+
  - `protoc` + `protoc-gen-go` or [mise](https://mise.jdx.dev/) for [Kitchen Sink Workflow](#kitchen-sink-workflow)
- [Java](https://openjdk.org/) 8+
- TypeScript: [Node](https://nodejs.org) 16+
- Python: [uv](https://docs.astral.sh/uv/)
- [.NET](https://dotnet.microsoft.com/en-us/download)

And if you're running the fuzzer (see below)
- [Rust](https://rustup.rs/)

SDK and tool versions are defined in `versions.env`.

## Architecture

This (simplified) diagram shows the main components of Omes:

```mermaid
flowchart TD
    subgraph "CLI"
        RunWorker["run-worker"]
        RunScenario["run-scenario"]
        RunScenarioWithWorker["run-scenario-with-worker"]
    end

    Workers["Worker(s)"]
    WorkflowsAndActivities["Workflows and Activities"]
    
    RunScenarioWithWorker --> RunWorker
    RunScenarioWithWorker --> RunScenario
    RunWorker --> |"start"| Workers
    RunScenario --> |"start"| Scenario
    Scenario --> |"start"| Executor
    Workers --> |"consume"| WorkflowsAndActivities
    Executor --> |"produce"| WorkflowsAndActivities
```

* **Scenario**: starts an Executor to run a particular load configuration
* **Executor**: produces concurrent executions of workflows and activities requested by the Scenario
* **Workers**: consumes the workflows and activities started by the Executor

### Process Metrics Sidecar

Omes includes a process metrics sidecar - a Go-based HTTP server that monitors CPU and memory usage of the worker process. The sidecar is started by `run-worker` (or `run-scenario-with-worker`) after spawning the worker subprocess.

**Endpoints:**
- `/metrics` - Prometheus-format metrics (`process_cpu_percent`, `process_resident_memory_bytes`)
- `/info` - JSON metadata (`sdk_version`, `build_id`, `language`)

**Configuration flags:**
- `--worker-process-metrics-address` - Address for the sidecar HTTP server (required to enable)
- `--worker-metrics-version-tag` - SDK version to report in `/info` (defaults to `--version`)

**Example:**
```sh
go run ./cmd run-worker --language python --run-id my-run \
    --worker-process-metrics-address :9091 \
    --worker-metrics-version-tag v1.24.0
```

**Metrics export:**

When using the Prometheus instance (`--prom-instance-addr`), sidecar metrics can be exported to parquet files on shutdown. The export includes SDK version, build ID, and language from the `/info` endpoint.

- `--prom-export-worker-metrics` - Path to export metrics parquet file
- `--prom-export-process-job` - Prometheus job name for process metrics (default: `omes-worker-process`)
- `--prom-export-worker-info-address` - Address to fetch `/info` from during export (e.g., `localhost:9091`)

## Usage

### Define a scenario

Scenarios are defined using plain Go code. They are located in the [scenarios](./scenarios/) folder. There are already
multiple defined that can be used.

A scenario must select an `Executor`. The most common is the `KitchenSinkExecutor` which is a wrapper on the
`GenericExecutor` specific for executing the Kitchen Sink workflow. The Kitchen Sink workflow accepts
[actions](./workers/go/kitchensink/kitchen_sink.go) and is implemented in every worker language.

For example, here is [scenarios/workflow_with_single_noop_activity.go](scenarios/workflow_with_single_noop_activity.go):

```go
func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
        Description: "Each iteration executes a single workflow with a noop activity.",
        ExecutorFn: func() loadgen.Executor {
            return loadgen.KitchenSinkExecutor{
                TestInput: &kitchensink.TestInput{
                    WorkflowInput: &kitchensink.WorkflowInput{
                        InitialActions: []*kitchensink.ActionSet{
                            kitchensink.NoOpSingleActivityActionSet(),
                        },
                    },
                },
            }
        },
    })
}
```

> NOTE: The file name where the `Register` function is called, will be used as the name of the scenario.

#### Scenario Authoring Guidelines

1. Use snake case for scenario file names.
1. Use `KitchenSinkExecutor` for most basic scenarios, adding common/generic actions as need, but for unique
   scenarios use `GenericExecutor`.
1. When using `GenericExecutor`, use methods of `*loadgen.Run` in your `Execute` as much as possible.
1. Liberally add helpers to the `loadgen` package that will be useful to other scenario authors.

### Run scenario with worker - Start a worker, an optional dev server, and run a scenario

During local development it's typically easiest to run both the worker and the scenario together.
You can do that like follows. If you want an embedded server rather than one you've already started,
pass `--embedded-server`.

```sh
go run ./cmd run-scenario-with-worker --scenario workflow_with_single_noop_activity --language go
```

Notes:

- Cleanup is **not** automatically performed here
- Accepts combined flags for `run-worker` and `run-scenario` commands

### Run a worker for a specific language SDK

```sh
go run ./cmd run-worker --run-id local-test-run --language go
```

Notes:

- `--embedded-server` can be passed here to start an embedded localhost server
- `--task-queue-suffix-index-start` and `--task-queue-suffix-index-end` represent an inclusive range for running the
  worker on multiple task queues. The process will create a worker for every task queue from `<task-queue>-<start>`
  through `<task-queue>-end`. This only applies to multi-task-queue scenarios.

### Run a test scenario

```sh
go run ./cmd run-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

Notes:

- Run ID is used to derive ID prefixes and the task queue name, it should be used to start a worker on the correct task queue
  and by the cleanup script.
- By default the number of iterations or duration is specified in the scenario config. They can be overridden with CLI
  flags.
- See help output for available flags.

### Cleanup after scenario run

```sh
go run ./cmd cleanup-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

### Running a specific version of the SDK

The `--version` flag can be used to specify a version of the SDK to use, it accepts either
a version number like `v1.24.0` or you can also pass a local path to use a local SDK version.
This is useful while testing unreleased or in-development versions of the SDK.

```sh
go run ./cmd run-scenario-with-worker --scenario workflow_with_single_noop_activity --language go --version /path/to/go-sdk
```

### Building and publishing docker images

For example, to build a go worker image using v1.24.0 of the Temporal Go SDK:

```sh
go run ./cmd/dev build-worker-image --language go --version v1.24.0
```

This will produce an image tagged like `<current git commit hash>-go-v1.24.0`.
If version is not specified, the SDK version specified in `versions.env` will be used.

Publishing images is done via CI, using the `build-push-worker-image` command.
See the GHA workflows for more information.

## Project Tests

Project tests are self-contained test programs that exercise specific SDK features (e.g., Nexus operations)
independently of the standard scenario/executor framework. Each project brings its own SDK version and
test logic, while a shared harness handles client creation, worker lifecycle, and gRPC orchestration.

### Structure

```
workers/<lang>/projects/
  harness/     # Shared harness (gRPC server, client factory, worker management)
  tests/
    HelloWorld/           # Simple example project
    NexusSimpleWorkflow/  # Nexus operation test
```

Each project binary supports two subcommands:
- `worker` — starts a Temporal worker
- `project-server` — starts a gRPC server that accepts `Init` and `Execute` RPCs

The `Init` RPC initializing the load test, providing client connection parameters, load test metadata
(i.e. RunID), and optional project-specific data.

The `Execute` RPC is called for each iteration of the executor running against your project test. This is
principally what drives load.

### Docker

Project Dockerfiles produce images that can run both the project worker and the scenario runner.
The entrypoint script routes commands automatically: `worker` and `project-server` run the project
binary, while `run-scenario` and other commands run the omes CLI.

```sh
# Build
docker build -f dockerfiles/go-project.Dockerfile \
  --build-arg PROJECT_DIR=workers/go/projects/tests/helloworld \
  -t omes-go-project-helloworld .

# Run worker (one container)
docker run --rm -d omes-go-project-helloworld worker \
  --task-queue omes-my-run --server-address host.docker.internal:7233 --namespace default

# Run scenario (another container, same image)
docker run --rm omes-go-project-helloworld run-scenario \
  --scenario project \
  --run-id my-run --iterations 1 \
  --server-address host.docker.internal:7233 --namespace default
```

.NET projects use `dockerfiles/dotnet-project.Dockerfile` with the same pattern.

### Running a scenario from the CLI

The `project` scenario can build from source or use a pre-built binary:

```sh
# Build from source (local development)
go run ./cmd run-scenario --scenario project \
  --option language=go \
  --option project-dir=workers/go/projects/tests/helloworld \
  --run-id my-run --iterations 1

# Use pre-built binary (Docker / CI)
go run ./cmd run-scenario --scenario project \
  --option language=go \
  --option prebuilt-project-dir=/app/prebuilt-project \
  --run-id my-run --iterations 1
```

### Writing a new project test

1. Create a directory under `workers/<lang>/projects/tests/<TestName>/`
2. Use the harness API to register a client factory, optional init logic (this will run as part of the `Init` RPC), a worker, and an execute handler:

```go
h := harness.New()
h.RegisterClient(func(opts client.Options, _ harness.ClientConfig) (client.Client, error) {
    return client.Dial(opts)
})
h.OnInit(func(ctx context.Context, c client.Client, config harness.InitConfig) error {
    // Project-specific setup (e.g., create Nexus endpoints) — only runs on project-server Init
    return nil
})
h.RegisterWorker(workerFunc)
h.OnExecute(executeFunc)
h.Run()
```

3. Add a test function in `scenarios/project/project_test.go`

## Specific Scenarios

### ThroughputStress (Go only)

#### Sleep Activity

The throughput_stress scenario can be configured to run "sleep" activities with different configurations.

The configuration is done via a JSON file, which is passed to the scenario with the
`--option sleep-activity-per-priority-json=@<file>` flag. Example:

```
echo '{"count":{"type":"fixed","value":5},"groups":{"high":{"weight":2,"sleepDuration":{"type":"uniform","min":"2s","max":"4s"}},"low":{"weight":3,"sleepDuration":{"type":"discrete","weights":{"5s":3,"10s":1}}}}}' > sleep.json
go run ./cmd run-scenario-with-worker --scenario throughput_stress --language go --option sleep-activity-json=@sleep.json --run-id default-run-id
```

This runs 5 sleep activities per iteration, where "high" has a weight of 2 and sleeps for a random duration between 2-4s,
and "low" has a weight of 3 and sleeps for either 5s or 10s.

Look at `DistributionField` to learn more about different kinds of distrbutions.

#### Nexus 

The throughput_stress scenario can generate Nexus load if the scenario is started with `--option nexus-endpoint=my-nexus-endpoint`:

   ```
   temporal operator nexus endpoint create \
   --name my-nexus-endpoint \
   --target-namespace default \ # Change if needed
   --target-task-queue throughput_stress:default-run-id
   ```

1. Start the scenario with the given run-id:

  ```
  go run ./cmd run-scenario-with-worker --scenario throughput_stress --language go --option nexus-endpoint=my-nexus-endpoint --run-id default-run-id
  ```

### Fuzzer

The fuzzer scenario makes use of the kitchen sink workflow (see below) to exercise a wide
range of possible actions. Actions are pre-generated by the `kitchen-sink-gen` tool, written in
Rust, and are some combination of actions provided to the workflow as input, and actions to be
run by a client inside the scenario executor.

You can run the fuzzer with new random actions like so:

```sh
go run ./cmd run-scenario-with-worker --scenario fuzzer --iterations 1 --language cs
```

The fuzzer automatically creates a Nexus endpoint and generates Nexus operations. To use an existing endpoint instead:

```sh
go run ./cmd run-scenario-with-worker --scenario fuzzer --iterations 1 --language go --option nexus-endpoint=my-endpoint
```

By default, the scenario will spit out a `last_fuzz_run.proto` binary file containing the generated
actions. To re-run the same set of actions, you can pass in such a file like so:

```sh
go run ./cmd run-scenario-with-worker --scenario fuzzer --iterations 1 --language cs --option input-file=last_fuzz_run.proto
```

Or you can run with a specific seed (seeds are printed at the start of the scenario):

```sh
go run ./cmd run-scenario-with-worker --scenario fuzzer --iterations 1 --language cs --option seed=131962944538087455
```

However, the fuzzer is also sensitive to its configuration, and thus the seed will only produce
the exact same set of actions if the config has also not changed. Thus you should prefer to save
binary files rather than seeds.

Please do collect interesting fuzz cases in the `scenarios/fuzz_cases.yaml` file. This file
currently has seeds, but could also easily reference stored binary files instead.

## Design decisions

### Kitchen Sink Workflow

The Kitchen Sink workflows accepts a DSL generated by the `kitchen-sink-gen` Rust tool, allowing us
to test a wide variety of scenarios without having to imagine all possible edge cases that could
come up in workflows. Input may be saved for regression testing, or hand written for specific cases.

Build by running `go run ./cmd/dev build kitchensink`.
Test by running `go test -v ./loadgen -run TestKitchenSink`.
Prefix with env variable `SDK=<sdk>` to test a specific SDK only.

### Scenario Failure

A scenario can only fail if an `Execute` method returns an error, that means the control is fully in the scenario
authors's hands. For enforcing a timeout for a scenario, use options like workflow execution timeouts or write a
workflow that waits for a signal for a configurable amount of time.

## TODO

- Nicer output that includes resource utilization for the worker (when running all-in-one)
- Ruby worker

## Development

Use the dev command for development tasks:

```sh
go run ./cmd/dev install          # Install tools (default: all)
go run ./cmd/dev lint-and-format  # Lint and format workers (default: all)
go run ./cmd/dev test             # Test workers (default: all)
go run ./cmd/dev build            # Build worker images (default: all)
go run ./cmd/dev clean            # Clean worker artifacts (default: all)
go run ./cmd/dev build-proto      # Build kitchen-sink proto
```

Or target specific languages: `go run ./cmd/dev build go java python`

All versions are defined in `versions.env`.

## Fuzzer trophy case

* Python upsert SA with no initial attributes: [PR](https://github.com/temporalio/sdk-python/pull/440)
* Core cancel-before-start on abandon activities: [PR](https://github.com/temporalio/sdk-core/pull/652)
* Core panic on evicting run with buffered tasks: [PR](https://github.com/temporalio/sdk-core/pull/660)
* Out of order replay for local activity + cancel: [PR](https://github.com/temporalio/sdk-core/issues/803)
