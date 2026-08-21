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
    duration = config.success_duration
    if info.attempt <= config.fail_attempts:
        # Failure case: run failure duration (exceeds activity timeout)
        duration = config.failure_duration

    # Sleep for failure/success timeout duration.
    # In failure case, this will throw a cancellation error.
    await delay_activity(duration)


@activity.defn(name="heartbeat")
async def heartbeat_activity(config):
    """Activity that skips heartbeats for N attempts (causing heartbeat timeout), then sends them."""
    info = activity.info()
    should_send_heartbeats = info.attempt > config.fail_attempts
    if not should_send_heartbeats:
        # Failure case: run failure duration (exceeds heartbeat timeout)
        await delay_activity(config.failure_duration)
        return

    duration = (
        config.success_duration.seconds + config.success_duration.nanos / 1_000_000_000
    )
    interval = (
        config.heartbeat_interval.seconds
        + config.heartbeat_interval.nanos / 1_000_000_000
    )
    if interval <= 0:
        await asyncio.sleep(duration)
        activity.heartbeat()
        return

    await run_with_heartbeats(duration, interval)


async def run_with_heartbeats(duration, interval):
    activity.heartbeat()
    remaining = duration
    while remaining > 0:
        sleep_for = min(interval, remaining)
        await asyncio.sleep(sleep_for)
        remaining -= sleep_for
        if remaining > 0:
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
