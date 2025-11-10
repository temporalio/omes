import asyncio
import os

from google.protobuf.duration_pb2 import Duration
from temporalio import activity
from temporalio.client import Client

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


def create_client_activity(client: Client, err_on_unimplemented: bool = False):
    @activity.defn(name="client")
    async def client_activity(client_activity_proto):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id
        executor = ClientActionExecutor(
            client, workflow_id, activity_info.task_queue, err_on_unimplemented
        )
        await executor.execute_client_sequence(client_activity_proto.client_sequence)

    return client_activity
