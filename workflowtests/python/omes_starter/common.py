from dataclasses import dataclass

from temporalio.client import Client


@dataclass
class ExecuteContext:
    """Context passed to client's execute function."""

    iteration: int
    run_id: str
    task_queue: str
    client: Client  # Pre-created Temporal client


@dataclass
class WorkerContext:
    """Context passed to worker's configure function."""

    task_queue: str
    client: Client  # Pre-created Temporal client
    prom_listen_address: str | None
