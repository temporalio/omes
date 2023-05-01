# Omes - a load generator for Temporal

This project is for testing load generation scenarios against Temporal. This is primarily used by the Temporal team to
benchmark features and situations. Backwards compatibility may not be maintained.

## Why the weird name?

Omes (pronounced oh-mess) is the Hebrew word for "load" (עומס).

## Prerequisites

- [Go](https://golang.org/) 1.19+
- [Node](https://nodejs.org) 16+
- [Python](https://www.python.org/) 3.10+
  - [Poetry](https://python-poetry.org/): `poetry install`
(More TBD when we support workers in other languages)

## Installation

There's no need to install anything to use this, it's a self-contained Go project.

## Usage

### Define a scenario

Scenarios are defined using plain Go code. They are located in the [scenarios](./scenarios/) folder.

A scenario must select an `Executor` (currently only `SharedIterationsExecutor` is implemented).
`SharedIterationsExecutor` accepts an `Execute` function that is called concurrently to execute each iteration.

```go
func Execute(ctx context.Context, run *scenario.Run) error {
	return run.ExecuteKitchenSinkWorkflow(ctx, &kitchensink.WorkflowParams{
		Actions: []*kitchensink.Action{{ExecuteActivity: &kitchensink.ExecuteActivityAction{Name: "noop"}}},
	})
}
```

Omes comes with pre-implemented workflows and activities that can be run using any SDK language (see the `run-worker`
and `all-in-one` commands below).
Scenarios are not tied to number of workers, cluster configuration, or the worker SDK language.

Typically scenarios will use the Kitchen Sink workflow that runs [actions](./kitchensink/kitchensink.go) specified by
the client.

> NOTE: the Kitchen Sink workflow is the only workflow implemented at the moment, other workflows will be added as
> needed.

Scenarios must be explicitly registered to be runnable by omes:

```go
func init() {
	scenario.Register(&scenario.Scenario{
		Executor: &executors.SharedIterationsExecutor{
      Execute:     Execute,
      Concurrency: 5, // How many instances of the "Execute" function to run concurrently.
      Iterations:  10, // Total number of iterations of the "Execute" function to run.
      // "Duration" may be specified instead of "Iterations".
    }
	})
}
```

> NOTE: The file name where the `Register` function is called, will be used as the name of the scenario.

#### Scenario Authoring Guidelines

1. Use snake care for scenario file names.
1. Use methods of `*omes.Run` in your `Execute` as much as possible.
1. Add methods to `Run` as needed.

### Run a worker for a specific language SDK

```console
$ go run ./cmd run-worker --scenario workflow_with_single_noop_activity --run-id local-test-run --language go
```

### Run a test scenario

```console
$ go run ./cmd run --scenario workflow_with_single_noop_activity --run-id local-test-run
```

Notes:

- Run ID is used to derive ID prefixes and the task queue name, it should to start a worker on the correct task queue
  and by the cleanup script
- By default the number of iterations or duration is specified in the scenario config, those can be overridden with CLI
  flags
- See help output for available flags

### Cleanup after scenario run

```console
$ go run ./cmd cleanup --scenario workflow_with_single_noop_activity --run-id local-test-run
```

# All-in-one - Start a worker, an optional dev server, and run a scenario

```console
$ go run ./cmd all-in-one --scenario workflow_with_single_noop_activity --language go --start-local-server
```

Notes:

- Cleanup is **not** automatically performed here
- Accepts combined flags for `start-worker` and `run` commands

## Design decisions

### Kitchen Sink Workflow

The idea here was to strike a balance between a generic workflow implementation that would work for 99% of scenarios
while maintaining code simplicity and debuggability.

### Scenario Failure

A scenario can only fail if an `Execute` method returns an error, that means the control is fully in the scenario
authors's hands. For enforcing a timeout for a scenario, use options like workflow execution timeouts or write a
workflow that waits for a signal for a configurable amount of time.

## TODO

- Nicer output that includes resource utilization for the worker (when running all-in-one)
- More lang workers
