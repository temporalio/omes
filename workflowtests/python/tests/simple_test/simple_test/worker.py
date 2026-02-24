from omes_starter import WorkerConfig
from temporalio.client import Client
from temporalio.worker import Worker

from .workflows import SimpleWorkflow


async def worker_main(config: WorkerConfig) -> Worker:
    """Configure and return the worker."""
    client = await Client.connect(config.server_address, **config.connect_kwargs())
    return Worker(
        client,
        task_queue=config.task_queue,
        workflows=[SimpleWorkflow],
    )
