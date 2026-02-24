from omes_starter import WorkerConfig
from temporalio.client import Client
from temporalio.worker import Worker, PollerBehaviorSimpleMaximum

from .workflows import MyWorkflow, my_activity

async def worker_main(config: WorkerConfig) -> Worker:
    """Configure and return the worker."""
    client = await Client.connect(config.server_address, **config.connect_kwargs())

    return Worker(
        client,
        task_queue=config.task_queue,
        workflows=[MyWorkflow],
        activities=[my_activity],
        max_concurrent_activities=15,
        max_concurrent_workflow_tasks=10,
        # Any of these configurations fail:
        workflow_task_poller_behavior=PollerBehaviorSimpleMaximum(maximum=5),
        activity_task_poller_behavior=PollerBehaviorSimpleMaximum(maximum=5),
    )
