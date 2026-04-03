from __future__ import annotations

from temporalio.worker import Worker

from activities import (
    create_client_activity,
    delay_activity,
    heartbeat_activity,
    noop_activity,
    payload_activity,
    retryable_error_activity,
    timeout_activity,
)
from harness import App, WorkerContext
from kitchen_sink import KitchenSinkWorkflow, NexusHandlerWorkflow
from nexus_service import KitchenSinkNexusServiceHandler


def app() -> App:
    return App(worker=build_worker)


def build_worker(context: WorkerContext) -> Worker:
    return Worker(
        context.client,
        task_queue=context.task_queue,
        workflows=[KitchenSinkWorkflow, NexusHandlerWorkflow],
        activities=[
            noop_activity,
            delay_activity,
            payload_activity,
            retryable_error_activity,
            timeout_activity,
            heartbeat_activity,
            create_client_activity(
                context.client,
                context.err_on_unimplemented,
            ),
        ],
        nexus_service_handlers=[KitchenSinkNexusServiceHandler()],
        **context.worker_kwargs,
    )
