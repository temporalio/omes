from __future__ import annotations

from datetime import timedelta
from typing import cast

import nexusrpc
import nexusrpc.handler
import temporalio.common
from temporalio import nexus
from temporalio.api.common.v1 import Payload

from kitchen_sink import KITCHEN_SINK_SERVICE_NAME, KitchenSinkWorkflow
from protos.kitchen_sink_pb2 import NexusHandlerStart, WorkflowInput


@nexusrpc.service(name=KITCHEN_SINK_SERVICE_NAME)
class KitchenSinkNexusService:
    echo_sync: nexusrpc.Operation[str, str] = nexusrpc.Operation(name="echo-sync")
    echo_async: nexusrpc.Operation[NexusHandlerStart, Payload] = nexusrpc.Operation(
        name="echo-async"
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
        self, ctx: nexus.WorkflowRunOperationContext, input: NexusHandlerStart
    ) -> nexus.WorkflowHandle[Payload]:
        # The handler workflow is just a kitchenSink whose behavior is described by
        # input.workflow_input. Callers are responsible for ensuring initial_actions
        # terminate (typically by ending with ReturnResult) or the handler will wait
        # for signals indefinitely.
        wf_input = input.workflow_input if input.HasField("workflow_input") else WorkflowInput()
        if input.workflow_id:
            policy = temporalio.common.WorkflowIDConflictPolicy(
                cast(int, input.workflow_id_conflict_policy)
            )
            return await ctx.start_workflow(
                KitchenSinkWorkflow.run,
                wf_input,
                id=input.workflow_id,
                id_conflict_policy=policy,
                execution_timeout=timedelta(minutes=60),
            )
        return await ctx.start_workflow(
            KitchenSinkWorkflow.run,
            wf_input,
            id=ctx.request_id,
        )
