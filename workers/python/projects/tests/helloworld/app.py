from __future__ import annotations

from temporalio.client import Client
from temporalio.worker import Worker
from workflow import HelloWorldWorkflow

from harness import (
    App,
    ProjectExecuteContext,
    ProjectHandlers,
    WorkerContext,
    default_client_factory,
)


def app() -> App:
    return App(
        worker=build_worker,
        client_factory=default_client_factory,
        project=ProjectHandlers(execute=execute_project),
    )


def build_worker(client: Client, context: WorkerContext) -> Worker:
    return Worker(
        client,
        task_queue=context.task_queue,
        workflows=[HelloWorldWorkflow],
        **context.worker_kwargs,
    )


async def execute_project(client: Client, context: ProjectExecuteContext) -> None:
    handle = await client.start_workflow(
        HelloWorldWorkflow.run,
        "World",
        id=f"{context.run.execution_id}-{context.iteration}",
        task_queue=context.task_queue,
    )
    result = await handle.result()
    print(result)
