from omes_starter import OmesWorkerStarter, WorkerContext
from temporalio.worker import Worker

from workflows import SimpleWorkflow

starter = OmesWorkerStarter()


@starter.configure_worker
async def configure(ctx: WorkerContext) -> Worker:
    """Configure and return the worker."""
    return Worker(
        ctx.client,
        task_queue=ctx.task_queue,
        workflows=[SimpleWorkflow],
    )


if __name__ == "__main__":
    starter.run()  # Handles CLI args internally
