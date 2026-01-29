"""CLI dispatch for omes_starter.

This module provides the `run()` function that dispatches to either
client or worker mode based on the first command-line argument.
"""

import argparse
import asyncio
import os
import sys
from typing import Awaitable, Callable

from temporalio.client import Client
from temporalio.runtime import (
    LoggingConfig,
    PrometheusConfig,
    Runtime,
    TelemetryConfig,
    TelemetryFilter,
)
from temporalio.worker import Worker

from .common import ClientConfig, WorkerConfig

print()
print()
print("=== CLI.PY LOADED ===")
print()
print()

def run(
    *,
    client: Callable[[ClientConfig], Awaitable[None]],
    worker: Callable[[WorkerConfig], Awaitable[Worker]],
    **client_options,
) -> None:
    """Main entry point - dispatches to client or worker based on first arg.

    User writes a main.py that calls this function:

        from omes_starter import run
        from src.client import client_main
        from src.worker import worker_main

        run(client=client_main, worker=worker_main)

    The program is then invoked with a subcommand:
        python main.py client --port 8080 --server-address localhost:7233 ...
        python main.py worker --task-queue my-queue --server-address localhost:7233 ...

    Args:
        client: Async function called for each /execute request in client mode
        worker: Async function that returns a configured Worker in worker mode
        **client_options: Options passed to Client.connect() (data_converter, interceptors, etc.)
    """
    if len(sys.argv) < 2:
        print("Usage: python main.py <client|worker> [options]", file=sys.stderr)
        print("  client  - Run as HTTP client starter", file=sys.stderr)
        print("  worker  - Run as Temporal worker", file=sys.stderr)
        sys.exit(1)

    mode = sys.argv[1]

    if mode == "client":
        _run_client_mode(client, client_options)
    elif mode == "worker":
        _run_worker_mode(worker, client_options)
    else:
        print(f"Unknown mode: {mode}. Expected 'client' or 'worker'.", file=sys.stderr)
        sys.exit(1)


def _run_client_mode(
    handler: Callable[[ClientConfig], Awaitable[None]],
    client_options: dict,
) -> None:
    """Run in client mode - HTTP server that calls handler for each /execute."""
    from .client import OmesClientStarter

    # Parse args (skip program name and 'client' subcommand)
    parser = argparse.ArgumentParser(prog="main.py client")
    parser.add_argument("--port", type=int, default=8080)
    parser.add_argument("--task-queue", required=True)
    parser.add_argument("--server-address", default="localhost:7233")
    parser.add_argument("--namespace", default="default")
    args = parser.parse_args(sys.argv[2:])

    starter = OmesClientStarter(**client_options)
    starter._execute_fn = handler
    starter._run_with_args(args)


def _run_worker_mode(
    handler: Callable[[WorkerConfig], Awaitable[Worker]],
    client_options: dict,
) -> None:
    """Run in worker mode - create worker and run it."""
    # Parse args (skip program name and 'worker' subcommand)
    parser = argparse.ArgumentParser(prog="main.py worker")
    parser.add_argument("--task-queue", required=True)
    parser.add_argument("--server-address", default="localhost:7233")
    parser.add_argument("--namespace", default="default")
    parser.add_argument("--prom-listen-address")
    args = parser.parse_args(sys.argv[2:])

    print("*************************")
    print("PROM LISTEN ADDRESS", args.prom_listen_address)
    print("*************************")
    prometheus = (
        PrometheusConfig(
            bind_address=args.prom_listen_address, durations_as_seconds=True
        )
        if args.prom_listen_address
        else None
    )

    new_runtime = Runtime(
        telemetry=TelemetryConfig(
            metrics=prometheus,
            logging=LoggingConfig(
                filter=TelemetryFilter(
                    core_level=os.getenv("TEMPORAL_CORE_LOG_LEVEL", "INFO"),
                    other_level="WARN",
                )
            ),
        ),
    )

    async def start():
        temporal_client = await Client.connect(
            args.server_address,
            namespace=args.namespace,
            **client_options,
            runtime=new_runtime,
        )

        config = WorkerConfig(
            task_queue=args.task_queue,
            client=temporal_client,
            prom_listen_address=args.prom_listen_address,
        )

        worker_instance = await handler(config)
        print(f"Worker started on task queue: {args.task_queue}")
        await worker_instance.run()

    asyncio.run(start())
