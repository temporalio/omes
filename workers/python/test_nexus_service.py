from datetime import timedelta
from unittest import IsolatedAsyncioTestCase
from unittest.mock import AsyncMock

from google.protobuf.empty_pb2 import Empty

from nexus_service import KitchenSinkNexusServiceHandler
from protos.kitchen_sink_pb2 import (
    NexusOperationRequest,
    NexusWorkflowAction,
    NexusWorkflowStartOptions,
)


class KitchenSinkNexusServiceHandlerTests(IsolatedAsyncioTestCase):
    async def test_execute_echo_returns_payload_synchronously(self) -> None:
        result = await KitchenSinkNexusServiceHandler().execute(
            AsyncMock(), AsyncMock(), NexusOperationRequest(echo="hello")
        )

        self.assertEqual(b'"hello"', result.value.data)

    async def test_start_workflow_always_sets_execution_timeout(self) -> None:
        for workflow_id in ("", "workflow-id"):
            with self.subTest(workflow_id=workflow_id):
                context = AsyncMock()
                context.request_id = "request-id"
                client = AsyncMock()

                await KitchenSinkNexusServiceHandler().execute(
                    context,
                    client,
                    NexusOperationRequest(
                        workflow_action=NexusWorkflowAction(
                            workflow_id=workflow_id,
                            start_options=NexusWorkflowStartOptions(),
                            start=Empty(),
                        )
                    ),
                )

                self.assertEqual(
                    timedelta(minutes=60),
                    client.start_workflow.await_args.kwargs.get("execution_timeout"),
                )
