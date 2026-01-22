from dataclasses import dataclass

from temporalio.client import Client


@dataclass
class ClientConfig:
    """Config passed to client's execute function.

    Attributes:
        client: Pre-created Temporal client connection
        task_queue: Task queue for workflows
        run_id: Unique ID for this load test run (from /execute request)
        iteration: Current iteration number (from /execute request)
    """

    client: Client
    task_queue: str
    run_id: str
    iteration: int


@dataclass
class WorkerConfig:
    """Config passed to worker's configure function.

    Attributes:
        client: Pre-created Temporal client connection
        task_queue: Task queue for the worker
        prom_listen_address: Optional Prometheus metrics endpoint address
    """

    client: Client
    task_queue: str
    prom_listen_address: str | None = None


# Backwards compatibility aliases
ExecuteContext = ClientConfig
WorkerContext = WorkerConfig
