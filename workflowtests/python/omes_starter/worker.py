import argparse
import asyncio
from typing import Awaitable, Callable

from temporalio.client import Client
from temporalio.worker import Worker

from .common import WorkerConfig


class OmesWorkerStarter:
    """Wrapper for worker boilerplate.

    Handles CLI arg parsing and client creation. Lifecycle management
    (HTTP endpoints for shutdown/metrics) is handled by `omes exec --remote-worker`.
    """

    def __init__(self, **client_options):
        """Initialize the worker starter.

        Args:
            **client_options: Options passed to Client.connect()
                              (data_converter, interceptors, etc.)
        """
        self._client_options = client_options
        self._configure_fn: Callable[[WorkerConfig], Awaitable[Worker]] | None = None

    def configure_worker(
        self, fn: Callable[[WorkerConfig], Awaitable[Worker]]
    ) -> Callable[[WorkerConfig], Awaitable[Worker]]:
        """Decorator to register the worker configuration function.

        Example:
            @starter.configure_worker
            async def configure(config: WorkerConfig) -> Worker:
                return Worker(
                    config.client,
                    task_queue=config.task_queue,
                    workflows=[MyWorkflow],
                )
        """
        self._configure_fn = fn
        return fn

    def run(self):
        """Parse CLI args, create client, configure worker, and run."""
        parser = argparse.ArgumentParser()
        parser.add_argument("--task-queue", required=True)
        parser.add_argument("--server-address", default="localhost:7233")
        parser.add_argument("--namespace", default="default")
        parser.add_argument("--prom-listen-address")
        args = parser.parse_args()

        async def start():
            client = await Client.connect(
                args.server_address,
                namespace=args.namespace,
                **self._client_options,
            )

            config = WorkerConfig(
                client=client,
                task_queue=args.task_queue,
                prom_listen_address=args.prom_listen_address,
            )

            worker = await self._configure_fn(config)
            print(f"Worker started on task queue: {args.task_queue}")
            await worker.run()

        asyncio.run(start())


def run_worker(
    handler: Callable[[WorkerConfig], Awaitable[Worker]],
    **client_options,
) -> None:
    """Convenience function to run a worker with the given configuration handler.

    Note: Prefer using omes_starter.run() with the new subcommand pattern.

    Example:
        # src/worker.py
        async def worker_main(config: WorkerConfig) -> Worker:
            return Worker(
                config.client,
                task_queue=config.task_queue,
                workflows=[MyWorkflow],
                activities=[my_activity],
            )

        # Call directly:
        run_worker(worker_main)

    Args:
        handler: Async function that returns a configured Worker
        **client_options: Options passed to Client.connect()
    """
    starter = OmesWorkerStarter(**client_options)
    starter._configure_fn = handler
    starter.run()
