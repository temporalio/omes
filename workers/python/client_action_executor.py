from typing import Any

from temporalio.client import Client
from temporalio.exceptions import ApplicationError

from protos.kitchen_sink_pb2 import ClientSequence


class ClientActionExecutor:
    def __init__(self, client: Client, workflow_id: str, task_queue: str):
        pass

    async def execute_client_sequence(self, client_seq: ClientSequence):
        raise ApplicationError(
            "client actions activity is not supported", non_retryable=True
        )
