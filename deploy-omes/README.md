# OMES Worker & Executor Scripts

Manage OMES workers and scenario executors on a running test cell.

## Prerequisites

A test cell with an OMES deployment. Either:

1. Scaffold with `--omes-enabled` (creates cell, namespace, deployment, and mTLS certs), or
2. Use `omni scaffold environment omes setup` to add an OMES deployment to an existing cell

## Setup

Generate config from a running cluster:

```
fish fetch-config.fish <cell-id>
```

Or edit `config.fish` manually. Key settings:

- `auth_method`: `"api_key"` or `"mtls"`
- `api_gateway`: API gateway endpoint (for `api_key` auth)

For API key auth, create the k8s secret:

```
ct kubectl --context <cell> create secret generic omes-api-key \
    -n omes --from-literal=api-key=<your-api-key>
```

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
