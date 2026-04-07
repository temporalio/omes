"""Getting started with the Python Omes harness.

Start by defining an `App`. `App` is the object the harness uses to run your
code: you give it a worker factory, a client factory, and, if you need project
mode, optional `ProjectHandlers`.

The worker factory is your main hook. It receives a `WorkerContext` with the
connected Temporal client, logger, task queue, and parsed worker options, and
it returns the `temporalio.worker.Worker` the harness should run. The client
factory controls how that Temporal client is created; most apps can use
`default_client_factory` as-is.

`run(app)` is the process entrypoint. It starts either `worker` mode or
`project-server` mode based on the first CLI argument.

Worker mode is the normal case for running a Temporal worker:

    from temporalio.worker import Worker
    from harness import App, WorkerContext, default_client_factory, run

    def build_worker(context: WorkerContext) -> Worker:
        return Worker(context.client, task_queue=context.task_queue)

    def app() -> App:
        return App(worker=build_worker, client_factory=default_client_factory)

    if __name__ == "__main__":
        run(app())

Project mode is for standalone project execution. In this mode the harness can
run one optional setup step for a project run, then call your execute hook once
per iteration for that run.

    from harness import (
        App,
        ProjectExecuteContext,
        ProjectHandlers,
        ProjectInitContext,
        default_client_factory,
    )

    async def init_project(client, context: ProjectInitContext) -> None:
        ...

    async def execute_project(client, context: ProjectExecuteContext) -> None:
        ...

    def app() -> App:
        return App(
            worker=build_worker,
            client_factory=default_client_factory,
            project=ProjectHandlers(execute=execute_project, init=init_project),
        )

Use `ProjectHandlers` only when you need project mode. Most apps only need a
worker factory plus `default_client_factory`; reach for `ClientConfig` and
`ClientFactory` only when you need custom client setup.
"""

from .client import ClientConfig, ClientFactory, default_client_factory
from .main import App, run
from .project import (
    ProjectExecuteContext,
    ProjectHandlers,
    ProjectInitContext,
    ProjectRunMetadata,
)
from .worker import WorkerContext

__all__ = [
    "App",
    "ClientConfig",
    "ClientFactory",
    "ProjectExecuteContext",
    "ProjectHandlers",
    "ProjectInitContext",
    "ProjectRunMetadata",
    "WorkerContext",
    "default_client_factory",
    "run",
]
