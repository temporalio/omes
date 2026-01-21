# Omes Starter Library (Python)

Library for writing workflow load tests with Omes.

## Client Starter

The client starter provides an HTTP server that receives iteration requests from Omes. It receives runtime parameters via CLI arguments.

### CLI Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `--port` | Yes | HTTP server port |
| `--task-queue` | Yes | Temporal task queue name |
| `--server-address` | No | Temporal server address (default: localhost:7233) |
| `--namespace` | No | Temporal namespace (default: default) |

**Note:** `run_id` is passed per-request in the `/execute` body, not as a CLI arg.

### Example

```python
from omes_starter import OmesClientStarter, ExecuteContext

starter = OmesClientStarter(
    # Optional client options:
    # data_converter=...,
    # interceptors=[...],
)

@starter.on_execute
async def execute(ctx: ExecuteContext):
    """Called for each iteration."""
    handle = await ctx.client.start_workflow(
        "MyWorkflow",
        id=f"wf-{ctx.run_id}-{ctx.iteration}",
        task_queue=ctx.task_queue,
    )
    await handle.result()

if __name__ == "__main__":
    starter.run()  # Handles CLI args internally
```

Run with:
```bash
python client.py --port 8080 --task-queue my-queue
```

## Worker Starter

The worker starter abstracts CLI arg parsing and client creation.

```python
from omes_starter import OmesWorkerStarter, WorkerContext
from temporalio.worker import Worker
from workflows import MyWorkflow

starter = OmesWorkerStarter(
    # Optional client options:
    # data_converter=...,
    # interceptors=[...],
)

@starter.configure_worker
async def configure(ctx: WorkerContext) -> Worker:
    """Configure and return the worker."""
    return Worker(
        ctx.client,
        task_queue=ctx.task_queue,
        workflows=[MyWorkflow],
    )

if __name__ == "__main__":
    starter.run()  # Handles CLI args internally
```

Worker lifecycle is managed by `omes exec --remote-worker` which spawns this as a subprocess and provides HTTP endpoints for shutdown and metrics.

## SDK Version Override

The Python starter library is SDK-version agnostic - it uses whatever `temporalio` is in the environment. When using `omes exec` or `omes workflow` commands, the SDK version is overridden transparently:

- **For local paths:** sdkbuild builds a wheel from the local SDK, then `uv run --override "temporalio @ file:///path/to/wheel.whl"` injects it
- **For released versions:** sdkbuild fetches/builds the specified version

User code writes normal Python imports:
```python
from temporalio.client import Client
from temporalio.worker import Worker
```

The SDK override is invisible to user code - it happens via `uv run --override` at runtime. User's original `pyproject.toml` is never modified.

## Usage

Users run with their desired SDK version via `omes exec` or `omes workflow`:

```bash
# One command runs everything
omes workflow \
  --language python \
  --sdk-version /path/to/sdk \
  --client-command "python client.py" \
  --worker-command "python worker_main.py" \
  --iterations 100
```
