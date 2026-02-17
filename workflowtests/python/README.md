# Omes Starter Library (Python)

Library for writing workflow load tests with Omes.

## Execution model

Your test package exposes client and worker handlers, then calls `run(...)` once:

```python
from omes_starter import run
from .client import client_main
from .worker import worker_main

run(client=client_main, worker=worker_main)
```

At runtime, Omes invokes the same program with a subcommand:

- `client`: starts an HTTP server used by the runner (`/execute`, `/info`)
- `worker`: starts the Temporal worker process

The worker is managed externally by the orchestrator/runtime. The starter does not expose `/shutdown`.

## Client handler API

`client_main(config: ClientConfig)` is called once per `/execute` request.

`ClientConfig` includes:

- `server_address`
- `namespace`
- `task_queue`
- `run_id`
- `iteration`
- `auth_header` / `tls` / `tls_server_name`

Use `config.connect_kwargs()` to pass auth/TLS options into `Client.connect(...)`.

Core pattern for now is direct client construction per execute call:

```python
from temporalio.client import Client
from omes_starter import ClientConfig

async def client_main(config: ClientConfig) -> None:
    client = await Client.connect(config.server_address, **config.connect_kwargs())
    handle = await client.start_workflow(
        "SimpleWorkflow",
        id=f"wf-{config.run_id}-{config.iteration}",
        task_queue=config.task_queue,
    )
    await handle.result()
```

## Worker handler API

`worker_main(config: WorkerConfig) -> Worker` receives:

- `server_address`
- `namespace`
- `task_queue`
- `prom_listen_address`
- `auth_header` / `tls` / `tls_server_name`
- `runtime` (preconfigured runtime, including Prometheus telemetry when `--prom-listen-address` is set)

Use `config.connect_kwargs()` to pass auth/TLS/runtime into `Client.connect(...)`.

```python
from temporalio.client import Client
from temporalio.worker import Worker
from omes_starter import WorkerConfig
from .workflows import SimpleWorkflow

async def worker_main(config: WorkerConfig) -> Worker:
    client = await Client.connect(config.server_address, **config.connect_kwargs())
    return Worker(
        client,
        task_queue=config.task_queue,
        workflows=[SimpleWorkflow],
    )
```

## CLI arguments

Client mode:

- `--port` (default `8080`)
- `--task-queue` (required)
- `--server-address` (default `localhost:7233`)
- `--namespace` (default `default`)
- `--auth-header`
- `--tls`
- `--tls-server-name`

Worker mode:

- `--task-queue` (required)
- `--server-address` (default `localhost:7233`)
- `--namespace` (default `default`)
- `--auth-header`
- `--tls`
- `--tls-server-name`
- `--prom-listen-address`

## SDK version override

The starter is SDK-version agnostic. `omes exec` / `omes workflow` choose the effective SDK version:

- local path: sdkbuild builds a wheel and injects it via `uv run --override`
- released version: sdkbuild resolves/install that version

User code keeps normal imports:

```python
from temporalio.client import Client
from temporalio.worker import Worker
```

## Usage

```bash
# Run worker only
omes exec --language python --project-dir ./tests/simple_test -- \
  worker --task-queue my-queue --server-address localhost:7233 --namespace default

# Run load test (spawns client; add --spawn-worker to co-manage worker)
omes workflow --language python --project-dir ./tests/simple_test \
  --iterations 100 --task-queue my-queue --server-address localhost:7233 --namespace default
```
