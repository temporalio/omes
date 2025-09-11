import argparse
import asyncio
import logging
import os
import sys
from typing import List

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
from temporalio.worker import Worker

from activities import (
    create_client_activity,
    delay_activity,
    noop_activity,
    payload_activity,
)
from kitchen_sink import KitchenSinkWorkflow

nameToLevel = {
    "PANIC": logging.FATAL,
    "FATAL": logging.FATAL,
    "ERROR": logging.ERROR,
    "WARN": logging.WARNING,
    "INFO": logging.INFO,
    "DEBUG": logging.DEBUG,
    "NOTSET": logging.NOTSET,
}

interrupt_event = asyncio.Event()


async def run():
    # Parse args
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
        "--max-concurrent-activities", type=int, help="Max concurrent activities"
    )
    parser.add_argument(
        "--max-concurrent-workflow-tasks",
        type=int,
        help="Max concurrent workflow tasks",
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
    parser.add_argument("--tls", action="store_true", help="Enable TLS")
    parser.add_argument(
        "--tls-cert-path", default="", help="Path to client TLS certificate"
    )
    parser.add_argument("--tls-key-path", default="", help="Path to client private key")
    parser.add_argument("--api-key", help="API key for authentication")
    # Prometheus metric arguments
    parser.add_argument("--prom-listen-address", help="Prometheus listen address")
    parser.add_argument(
        "--prom-handler-path", default="/metrics", help="Prometheus handler path"
    )
    args = parser.parse_args()

    if args.task_queue_suffix_index_start > args.task_queue_suffix_index_end:
        raise ValueError("Task queue suffix start after end")

    # Configure TLS
    tls_config = None
    if args.tls_cert_path:
        if not args.tls_key_path:
            raise ValueError("Client cert specified, but not client key!")
        with open(args.tls_cert_path, "rb") as f:
            client_cert = f.read()
        with open(args.tls_key_path, "rb") as f:
            client_key = f.read()
        tls_config = TLSConfig(client_cert=client_cert, client_private_key=client_key)
    elif args.tls_key_path and not args.tls_cert_path:
        raise ValueError("Client key specified, but not client cert!")
    elif args.tls:
        tls_config = TLSConfig()

    # Configure logging
    logger = logging.getLogger()
    logHandler = logging.StreamHandler(stream=sys.stderr)
    if args.log_encoding == "json":
        format_str = "%(message)%(levelname)%(name)%(asctime)"
        formatter = jsonlogger.JsonFormatter(format_str)
        logHandler.setFormatter(formatter)
    logger.addHandler(logHandler)
    logger.setLevel(nameToLevel[args.log_level.upper()])

    prometheus = (
        PrometheusConfig(bind_address=args.prom_listen_address)
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
    client = await Client.connect(
        target_host=args.server_address,
        namespace=args.namespace,
        tls=tls_config,
        runtime=new_runtime,
        api_key=args.api_key,
    )

    # Collect task queues to run workers for (if there is a suffix end, we run
    # multiple)
    task_queues: List[str]
    if args.task_queue_suffix_index_end == 0:
        task_queues = [args.task_queue]
        logger.info("Python worker running for task queue %s" % args.task_queue)
    else:
        task_queues = [
            f"{args.task_queue}-{i}"
            for i in range(
                args.task_queue_suffix_index_start, args.task_queue_suffix_index_end + 1
            )
        ]
        logger.info("Python worker running for %s task queue(s)" % len(task_queues))

    worker_kwargs = {}
    if args.max_concurrent_activity_pollers is not None:
        worker_kwargs[
            "max_concurrent_activity_task_polls"
        ] = args.max_concurrent_activity_pollers
    if args.max_concurrent_workflow_pollers is not None:
        worker_kwargs[
            "max_concurrent_workflow_task_polls"
        ] = args.max_concurrent_workflow_pollers
    if args.max_concurrent_activities is not None:
        worker_kwargs["max_concurrent_activities"] = args.max_concurrent_activities
    if args.max_concurrent_workflow_tasks is not None:
        worker_kwargs[
            "max_concurrent_workflow_tasks"
        ] = args.max_concurrent_workflow_tasks

    # Start all workers, throwing on first exception
    workers = [
        Worker(
            client,
            task_queue=task_queue,
            workflows=[KitchenSinkWorkflow],
            activities=[
                noop_activity,
                delay_activity,
                payload_activity,
                create_client_activity(client),
            ],
            **worker_kwargs,
        )
        for task_queue in task_queues
    ]
    all_workers_task = asyncio.gather(*[worker.run() for worker in workers])

    # Wait for worker fail or interrupt (this will not throw)
    await asyncio.wait(
        [all_workers_task, asyncio.create_task(interrupt_event.wait())],
        return_when=asyncio.FIRST_COMPLETED,
    )

    # Shut all workers down (shutdown waits for complete but does not throw)
    await asyncio.gather(*[worker.shutdown() for worker in workers])

    # Now await the original run task in case it threw
    await all_workers_task


if __name__ == "__main__":
    loop = asyncio.new_event_loop()
    try:
        loop.run_until_complete(run())
    except KeyboardInterrupt:
        interrupt_event.set()
        loop.run_until_complete(loop.shutdown_asyncgens())
