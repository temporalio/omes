from __future__ import annotations

import uuid
from typing import Optional

import nexusrpc
import nexusrpc.handler
from temporalio import nexus, workflow
from temporalio.api.common.v1 import Payload

from protos.kitchen_sink_pb2 import WorkflowInput


KITCHEN_SINK_SERVICE_NAME = "kitchen-sink"


@nexusrpc.service(name=KITCHEN_SINK_SERVICE_NAME)
class KitchenSinkNexusService:
    echo_sync: nexusrpc.Operation[str, str] = nexusrpc.Operation(name="echo-sync")
    echo_async: nexusrpc.Operation[str, str] = nexusrpc.Operation(name="echo-async")
    wait_for_cancel: nexusrpc.Operation[None, None] = nexusrpc.Operation(
        name="wait-for-cancel"
    )
    run_handler_actions: nexusrpc.Operation[WorkflowInput, Payload] = (
        nexusrpc.Operation(name="run-handler-actions")
    )


@nexusrpc.handler.service_handler(service=KitchenSinkNexusService)
class KitchenSinkNexusServiceHandler:
    @nexusrpc.handler.sync_operation
    async def echo_sync(
        self, ctx: nexusrpc.handler.StartOperationContext, input: str
    ) -> str:
        return input

    @nexus.workflow_run_operation
    async def echo_async(
        self, ctx: nexus.WorkflowRunOperationContext, input: str
    ) -> nexus.WorkflowHandle[str]:
        return await ctx.start_workflow(
            EchoWorkflow.run,
            input,
            id=ctx.request_id or str(uuid.uuid4()),
        )

    @nexus.workflow_run_operation
    async def wait_for_cancel(
        self, ctx: nexus.WorkflowRunOperationContext, input: None
    ) -> nexus.WorkflowHandle[None]:
        return await ctx.start_workflow(
            WaitForCancelWorkflow.run,
            id=ctx.request_id or str(uuid.uuid4()),
        )

    @nexus.workflow_run_operation
    async def run_handler_actions(
        self, ctx: nexus.WorkflowRunOperationContext, input: WorkflowInput
    ) -> nexus.WorkflowHandle[Payload]:
        return await ctx.start_workflow(
            "kitchenSink",
            input,
            id=ctx.request_id or str(uuid.uuid4()),
        )


@workflow.defn
class EchoWorkflow:
    @workflow.run
    async def run(self, s: str) -> str:
        return s


@workflow.defn
class WaitForCancelWorkflow:
    @workflow.run
    async def run(self) -> None:
        await workflow.wait_condition(lambda: False)
