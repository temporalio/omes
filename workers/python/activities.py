import asyncio
import os

from google.protobuf.duration_pb2 import Duration
from temporalio import activity
from temporalio.client import Client
from temporalio.exceptions import ApplicationError

from client_action_executor import ClientActionExecutor


@activity.defn(name="noop")
async def noop_activity():
    return


@activity.defn(name="delay")
async def delay_activity(delay_for: Duration):
    await asyncio.sleep(delay_for.ToSeconds())


@activity.defn(name="payload")
async def payload_activity(input_data: bytes, bytes_to_return: int) -> bytes:
    return os.urandom(bytes_to_return)


@activity.defn(name="retryable_error")
async def retryable_error_activity(config):
    """Activity that throws retryable errors for N attempts, then succeeds."""
    info = activity.info()
    if info.attempt <= config.fail_attempts:
        raise ApplicationError("retryable error", "RetryableError", non_retryable=False)


@activity.defn(name="timeout")
async def timeout_activity(config):
    """Activity that runs too long for N attempts (causing timeout), then completes quickly."""
    info = activity.info()
    duration_seconds = info.start_to_close_timeout.total_seconds()
    if info.attempt <= config.fail_attempts:
        # Failure case: run for double StartToCloseTimeout
        duration_seconds *= 2
    else:
        # Success case: run for half StartToCloseTimeout
        duration_seconds /= 2

    # Sleep for failure/success timeout duration.
    # In failure case, this will throw a cancellation error.
    await asyncio.sleep(duration_seconds)


@activity.defn(name="heartbeat")
async def heartbeat_activity(config):
    """Activity that skips heartbeats for N attempts (causing heartbeat timeout), then sends them."""
    info = activity.info()
    should_send_heartbeats = info.attempt > config.fail_attempts

    # Run activity for 2x the heartbeat timeout
    # Ensures we miss enough heartbeat intervals (if not sending heartbeats).
    duration_seconds = info.heartbeat_timeout.total_seconds() * 2

    heartbeat_interval = 1.0  # Send heartbeat every second
    elapsed = 0.0
    while elapsed < duration_seconds:
        sleep_time = min(heartbeat_interval, duration_seconds - elapsed)
        await asyncio.sleep(sleep_time)
        elapsed += sleep_time
        if should_send_heartbeats and elapsed < duration_seconds:
            activity.heartbeat()


def create_client_activity(client: Client, err_on_unimplemented: bool):
    @activity.defn(name="client")
    async def client_activity(client_activity_proto):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id
        executor = ClientActionExecutor(
            client, workflow_id, activity_info.task_queue, err_on_unimplemented
        )
        await executor.execute_client_sequence(client_activity_proto.client_sequence)

    return client_activity
