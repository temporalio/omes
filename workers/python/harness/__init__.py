"""Python harness entrypoints and extension hooks.

Define an `App` with a worker factory and client factory. Add
`ProjectHandlers` only when you need `project-server` mode.

The worker factory is the main integration point: the harness creates one
Temporal client with the configured `ClientFactory`, then passes that client
and a `WorkerContext` to your factory for each task queue it runs. `run(app)`
dispatches to either `worker` mode or `project-server` mode based on the first
CLI argument.

Worker mode runs one or more Temporal workers:

    from temporalio.client import Client
    from temporalio.worker import Worker
    from harness import App, WorkerContext, default_client_factory, run

    def build_worker(client: Client, context: WorkerContext) -> Worker:
        return Worker(client, task_queue=context.task_queue)

    def app() -> App:
        return App(worker=build_worker, client_factory=default_client_factory)

    if __name__ == "__main__":
        run(app())

Project mode exposes init and execute hooks for standalone project runs. The
project server can perform one optional setup step and then invoke your
execute hook once per iteration.

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

Most apps only need a worker factory plus `default_client_factory`. Reach for
`ProjectHandlers` only when you need project mode, and customize
`ClientFactory` only when the default client setup is not enough.
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
