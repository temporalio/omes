# OMES Worker & Executor Scripts

Manage OMES workers and scenario executors on a running test cell.

## Prerequisites

A test cell with OMES enabled must be scaffolded first:

```
omni scaffold environment create \
    --cell-id=<cell> \
    --namespace=<ns> \
    --yaml=v5-aws-dev \
    --temporal-version=<server-version> \
    --agent-version=<server-version> \
    --web-version=<web-version> \
    --go-canary-version=<canary-version> \
    --omes-enabled \
    --omes-run-id=<run-id> \
    --omes-image-tag=<image-tag>
```

This creates the cell, namespace, and initial OMES deployment (running `run-scenario-with-worker`). These scripts then split that into separate worker and executor components.

## Setup

Generate config from a running cluster:

```
fish fetch-config.fish <cell-id>
```

Or edit `config.fish` manually.

## Usage

**Patch the deployment to run workers only:**

```
fish patch-worker.fish [replicas]   # default: 2
```

**Run a scenario executor (creates a Job):**

```
fish run-executor.fish [duration]   # default: 600s
```

**Delete the executor job:**

```
fish delete-executor.fish
```
