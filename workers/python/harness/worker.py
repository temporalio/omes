from __future__ import annotations

import argparse
import asyncio
import logging
import os
import sys
from collections.abc import Callable
from contextlib import suppress
from dataclasses import dataclass
from typing import Any

from pythonjsonlogger import jsonlogger
from temporalio.client import Client
from temporalio.runtime import (
    LoggingConfig,
    PrometheusConfig,
    Runtime,
    TelemetryConfig,
    TelemetryFilter,
)
from temporalio.service import TLSConfig
from temporalio.worker import PollerBehaviorAutoscaling, Worker

_NAME_TO_LEVEL = {
    "PANIC": logging.FATAL,
    "FATAL": logging.FATAL,
    "ERROR": logging.ERROR,
    "WARN": logging.WARNING,
    "INFO": logging.INFO,
    "DEBUG": logging.DEBUG,
    "NOTSET": logging.NOTSET,
}


@dataclass(frozen=True)
class WorkerContext:
    client: Client
    logger: logging.Logger
    task_queue: str
    # Kitchen sink specific field - tells the client action executor
    # whether to error or not for unimplemented actions.
    err_on_unimplemented: bool
    worker_kwargs: dict[str, Any]


WorkerFactory = Callable[[WorkerContext], Worker]


def run_worker_cli(worker_factory: WorkerFactory) -> None:
    interrupt_event = asyncio.Event()
    loop = asyncio.new_event_loop()
    try:
        loop.run_until_complete(_run(worker_factory, interrupt_event))
    except KeyboardInterrupt:
        interrupt_event.set()
        loop.run_until_complete(loop.shutdown_asyncgens())
    finally:
        loop.close()


async def _run(worker_factory: WorkerFactory, interrupt_event: asyncio.Event) -> None:
    # Parse args
    args = _build_parser().parse_args()

    if args.task_queue_suffix_index_start > args.task_queue_suffix_index_end:
        raise ValueError("Task queue suffix start after end")

    logger = _configure_logger(args.log_level, args.log_encoding)
    runtime = _build_runtime(args.prom_listen_address)
    api_key = _build_api_key(args.auth_header)
    tls_config = _build_tls_config(args)
    client = await Client.connect(
        target_host=args.server_address,
        namespace=args.namespace,
        api_key=api_key,
        tls=tls_config,
        runtime=runtime,
    )

    # Collect task queues to run workers for (if there is a suffix end, we run
    # multiple)
    task_queues = _build_task_queues(
        logger,
        args.task_queue,
        args.task_queue_suffix_index_start,
        args.task_queue_suffix_index_end,
    )
    worker_kwargs = _build_worker_kwargs(args)
    workers = [
        worker_factory(
            WorkerContext(
                client=client,
                logger=logger,
                task_queue=task_queue,
                err_on_unimplemented=args.err_on_unimplemented,
                worker_kwargs=worker_kwargs,
            )
        )
        for task_queue in task_queues
    ]
    await _run_workers(workers, interrupt_event)


async def _run_workers(workers: list[Worker], interrupt_event: asyncio.Event) -> None:
    # Start all workers, throwing on first exception
    all_workers_task = asyncio.gather(*[worker.run() for worker in workers])
    interrupt_task = asyncio.create_task(interrupt_event.wait())
    try:
        # Wait for worker fail or interrupt (this will not throw)
        await asyncio.wait(  # type: ignore[type-var]
            [all_workers_task, interrupt_task],
            return_when=asyncio.FIRST_COMPLETED,
        )
        # Shut all workers down (shutdown waits for complete but does not throw)
        await asyncio.gather(*[worker.shutdown() for worker in workers])
        # Now await the original run task in case it threw
        await all_workers_task
    finally:
        interrupt_task.cancel()
        with suppress(asyncio.CancelledError):
            await interrupt_task


def _build_parser() -> argparse.ArgumentParser:
    parse_bool = (
        lambda value: value
        if isinstance(value, bool)
        else value.lower() in ("true", "1", "yes")
    )
    parser = argparse.ArgumentParser()
    parser.add_argument("-q", "--task-queue", default="omes", help="Task queue to use")
    parser.add_argument(
        "--task-queue-suffix-index-start",
        default=0,
        type=int,
        help="Inclusive start for task queue suffix range",
    )
    parser.add_argument(
        "--task-queue-suffix-index-end",
        default=0,
        type=int,
        help="Inclusive end for task queue suffix range",
    )
    parser.add_argument(
        "--max-concurrent-activity-pollers",
        type=int,
        help="Max concurrent activity pollers",
    )
    parser.add_argument(
        "--max-concurrent-workflow-pollers",
        type=int,
        help="Max concurrent workflow pollers",
    )
    parser.add_argument(
        "--activity-poller-autoscale-max",
        type=int,
        help="Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)",
    )
    parser.add_argument(
        "--workflow-poller-autoscale-max",
        type=int,
        help="Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)",
    )
    parser.add_argument(
        "--max-concurrent-activities", type=int, help="Max concurrent activities"
    )
    parser.add_argument(
        "--max-concurrent-workflow-tasks",
        type=int,
        help="Max concurrent workflow tasks",
    )
    parser.add_argument(
        "--activities-per-second",
        type=float,
        help="Per-worker activity rate limit",
    )
    parser.add_argument(
        "--err-on-unimplemented",
        default=False,
        type=parse_bool,
        help="Error when receiving unimplemented actions (currently only affects concurrent client actions)",
    )
    # Log arguments
    parser.add_argument(
        "--log-level", default="info", help="(debug info warn error panic fatal)"
    )
    parser.add_argument("--log-encoding", default="console", help="(console json)")
    # Client arguments
    parser.add_argument(
        "-n", "--namespace", default="default", help="Namespace to connect to"
    )
    parser.add_argument(
        "-a",
        "--server-address",
        default="localhost:7233",
        help="Address of Temporal server",
    )
    parser.add_argument(
        "--tls",
        type=parse_bool,
        default=False,
        help="Enable TLS (true/false)",
    )
    parser.add_argument(
        "--tls-cert-path", default="", help="Path to client TLS certificate"
    )
    parser.add_argument("--tls-key-path", default="", help="Path to client private key")
    # Prometheus metric arguments
    parser.add_argument("--prom-listen-address", help="Prometheus listen address")
    parser.add_argument(
        "--prom-handler-path", default="/metrics", help="Prometheus handler path"
    )
    parser.add_argument("--auth-header", default="", help="Authorization header value")
    parser.add_argument("--build-id", default="", help="Build ID")
    return parser


def _build_tls_config(args: argparse.Namespace) -> TLSConfig | None:
    if args.tls_cert_path:
        if not args.tls_key_path:
            raise ValueError("Client cert specified, but not client key!")
        with open(args.tls_cert_path, "rb") as cert_file:
            client_cert = cert_file.read()
        with open(args.tls_key_path, "rb") as key_file:
            client_key = key_file.read()
        return TLSConfig(client_cert=client_cert, client_private_key=client_key)
    if args.tls_key_path:
        raise ValueError("Client key specified, but not client cert!")
    if args.tls:
        return TLSConfig()
    return None


def _build_api_key(auth_header: str) -> str | None:
    return auth_header.removeprefix("Bearer ") if auth_header else None


def _configure_logger(log_level: str, log_encoding: str) -> logging.Logger:
    logger = logging.getLogger()
    logger.handlers.clear()
    handler = logging.StreamHandler(stream=sys.stderr)
    if log_encoding == "json":
        format_str = "%(message)%(levelname)%(name)%(asctime)"
        handler.setFormatter(jsonlogger.JsonFormatter(format_str))
    logger.addHandler(handler)
    logger.setLevel(_NAME_TO_LEVEL[log_level.upper()])
    return logger


def _build_runtime(prom_listen_address: str | None) -> Runtime:
    prometheus = (
        PrometheusConfig(bind_address=prom_listen_address, durations_as_seconds=True)
        if prom_listen_address
        else None
    )
    return Runtime(
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


def _build_task_queues(
    logger: logging.Logger, task_queue: str, suffix_start: int, suffix_end: int
) -> list[str]:
    if suffix_end == 0:
        logger.info("Python worker will run on task queue %s", task_queue)
        return [task_queue]
    task_queues = [f"{task_queue}-{i}" for i in range(suffix_start, suffix_end + 1)]
    logger.info("Python worker will run on %s task queue(s)", len(task_queues))
    return task_queues


def _build_worker_kwargs(args: argparse.Namespace) -> dict[str, Any]:
    worker_kwargs: dict[str, Any] = {}
    if args.activity_poller_autoscale_max is not None:
        worker_kwargs["activity_task_poller_behavior"] = PollerBehaviorAutoscaling(
            maximum=args.activity_poller_autoscale_max
        )
    elif args.max_concurrent_activity_pollers is not None:
        worker_kwargs[
            "max_concurrent_activity_task_polls"
        ] = args.max_concurrent_activity_pollers
    if args.workflow_poller_autoscale_max is not None:
        worker_kwargs["workflow_task_poller_behavior"] = PollerBehaviorAutoscaling(
            maximum=args.workflow_poller_autoscale_max
        )
    elif args.max_concurrent_workflow_pollers is not None:
        worker_kwargs[
            "max_concurrent_workflow_task_polls"
        ] = args.max_concurrent_workflow_pollers
    if args.max_concurrent_activities is not None:
        worker_kwargs["max_concurrent_activities"] = args.max_concurrent_activities
    if args.max_concurrent_workflow_tasks is not None:
        worker_kwargs[
            "max_concurrent_workflow_tasks"
        ] = args.max_concurrent_workflow_tasks
    if args.activities_per_second is not None:
        worker_kwargs["max_activities_per_second"] = args.activities_per_second
    return worker_kwargs
