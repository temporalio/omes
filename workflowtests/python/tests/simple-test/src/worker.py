from omes_starter import WorkerConfig
from temporalio.worker import Worker

from .workflows import SimpleWorkflow


async def worker_main(config: WorkerConfig) -> Worker:
    """Configure and return the worker."""
    return Worker(
        config.client,
        task_queue=config.task_queue,
        workflows=[SimpleWorkflow],
    )
