from __future__ import annotations

import json
from datetime import timedelta
from typing import cast

import nexusrpc
import nexusrpc.handler
import temporalio.common
from temporalio import nexus
from temporalio.api.common.v1 import Payload

from kitchen_sink import KITCHEN_SINK_SERVICE_NAME, KitchenSinkWorkflow
from protos.kitchen_sink_pb2 import NexusOperationRequest, WorkflowInput


@nexusrpc.service(name=KITCHEN_SINK_SERVICE_NAME)
class KitchenSinkNexusService:
    execute: nexusrpc.Operation[NexusOperationRequest, Payload] = nexusrpc.Operation(
        name="execute"
    )


@nexusrpc.handler.service_handler(service=KitchenSinkNexusService)
class KitchenSinkNexusServiceHandler:
    @nexus.temporal_operation
    async def execute(
        self,
        ctx: nexus.TemporalStartOperationContext,
        client: nexus.TemporalNexusClient,
        input: NexusOperationRequest,
    ) -> nexus.TemporalOperationResult[Payload]:
        action = input.WhichOneof("action")
        if action == "echo":
            return nexus.TemporalOperationResult.sync(
                Payload(
                    metadata={"encoding": b"json/plain"},
                    data=json.dumps(input.echo).encode(),
                )
            )
        if action == "workflow_action" and input.workflow_action.HasField("start"):
            workflow_action = input.workflow_action
            start = workflow_action.start_options
            workflow_input = (
                start.workflow_input
                if start.HasField("workflow_input")
                else WorkflowInput()
            )
            if workflow_action.workflow_id:
                policy = temporalio.common.WorkflowIDConflictPolicy(
                    cast(int, start.workflow_id_conflict_policy)
                )
                return await client.start_workflow(
                    KitchenSinkWorkflow.run,
                    workflow_input,
                    id=workflow_action.workflow_id,
                    task_queue=start.task_queue or None,
                    id_conflict_policy=policy,
                    execution_timeout=timedelta(minutes=60),
                )
            return await client.start_workflow(
                KitchenSinkWorkflow.run,
                workflow_input,
                id=ctx.request_id,
                task_queue=start.task_queue or None,
                execution_timeout=timedelta(minutes=60),
            )

        raise nexusrpc.HandlerError(
            "Nexus operation request has no supported action set",
            type=nexusrpc.HandlerErrorType.BAD_REQUEST,
        )
