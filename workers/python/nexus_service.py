from __future__ import annotations

from datetime import timedelta

import nexusrpc
import nexusrpc.handler
import temporalio.common
from temporalio import nexus

from kitchen_sink import KITCHEN_SINK_SERVICE_NAME, NexusHandlerWorkflow
from protos.kitchen_sink_pb2 import NexusHandlerInput


@nexusrpc.service(name=KITCHEN_SINK_SERVICE_NAME)
class KitchenSinkNexusService:
    echo_sync: nexusrpc.Operation[NexusHandlerInput, str] = nexusrpc.Operation(
        name="echo-sync"
    )
    echo_async: nexusrpc.Operation[NexusHandlerInput, str] = nexusrpc.Operation(
        name="echo-async"
    )


@nexusrpc.handler.service_handler(service=KitchenSinkNexusService)
class KitchenSinkNexusServiceHandler:
    @nexusrpc.handler.sync_operation
    async def echo_sync(
        self, ctx: nexusrpc.handler.StartOperationContext, input: NexusHandlerInput
    ) -> str:
        if len(input.before_actions) > 0:
            raise nexusrpc.HandlerError(
                "before_actions not supported in echo-sync",
                type=nexusrpc.HandlerErrorType.BAD_REQUEST,
            )
        return input.input

    @nexus.workflow_run_operation
    async def echo_async(
        self, ctx: nexus.WorkflowRunOperationContext, input: NexusHandlerInput
    ) -> nexus.WorkflowHandle[str]:
        if input.workflow_id_override:
            return await ctx.start_workflow(
                NexusHandlerWorkflow.run,
                input,
                id=input.workflow_id_override,
                id_conflict_policy=temporalio.common.WorkflowIDConflictPolicy.USE_EXISTING,
                # Use a large timeout so we aren't left with dangling handler workflows if the scenario fails.
                execution_timeout=timedelta(minutes=60),
            )
        return await ctx.start_workflow(
            NexusHandlerWorkflow.run,
            input,
            id=ctx.request_id,
        )
