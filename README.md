# Omes - a load generator for Temporal

## Why the weird name?

Omes (pronounced oh-mess) is the Hebrew word for "load" (עומס).

## Prerequisites

- Go 1.19

(More TBD when we support workers in other languages)

## Installation

There's no need to install anything to use this, it's a self-contained Go project.

## Usage

Loadgen has the following capabilities:

### Run a test scenario

```console
$ go run ./cmd/omes run --scenario WorkflowWithSingleNoopActivity --run-id local-test-run
```

Notes:

- Run ID is used to derive ID prefixes and the task queue name, it should to start a worker on the correct task queue
  and by the cleanup script
- By default the number of iterations or duration is specified in the scenario config, those can be overridden with CLI
  flags
- See help output for avaialble flags

### Run a worker for a specific language SDK (currently only Go)

```console
$ go run ./cmd/omes run-worker --scenario WorkflowWithSingleNoopActivity --run-id local-test-run --language go
```

### Cleanup after scenario run (requires ElasticSearch)

```console
$ go run ./cmd/omes cleanup --scenario WorkflowWithSingleNoopActivity --run-id local-test-run
```

# All-in-one - Start a worker, an optional dev server, and run a scenario

```console
$ go run ./cmd/omes all-in-one --scenario WorkflowWithSingleNoopActivity --language go --start-local-server
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
