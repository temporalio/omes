import asyncio
import os
from typing import Any, List

from google.protobuf.duration_pb2 import Duration
from temporalio import activity
from temporalio.client import Client

from protos.kitchen_sink_pb2 import (
    ClientAction,
    ClientActionSet,
    ClientSequence,
    DoQuery,
    DoSignal,
    DoUpdate,
)


@activity.defn(name="noop")
async def noop_activity():
    return


@activity.defn(name="delay")
async def delay_activity(delay_for: Duration):
    await asyncio.sleep(delay_for.ToSeconds())


@activity.defn(name="payload")
async def payload_activity(input_data: bytes, bytes_to_return: int) -> bytes:
    return os.urandom(bytes_to_return)


@activity.defn(name="client")
async def client_activity(client_activity_proto):
    client = activity.info().context.get("client")

    executor = ClientActionsExecutor(client)
    await executor.execute_client_sequence(client_activity_proto.client_sequence)


class ClientActionsExecutor:
    def __init__(self, client: Client):
        self.client = client
        self.workflow_id: str = ""
        self.run_id: str | None = None
        self.workflow_type: str = "kitchenSink"
        self.workflow_input = None

    async def execute_client_sequence(self, client_seq: ClientSequence):
        for action_set in client_seq.action_sets:
            await self._execute_client_action_set(action_set)

    async def _execute_client_action_set(self, action_set: ClientActionSet):
        if action_set.concurrent:
            raise ValueError(
                "Concurrent client actions are not supported in Python worker"
            )

        for action in action_set.actions:
            await self._execute_client_action(action)

        if action_set.wait_for_current_run_to_finish_at_end:
            if self.workflow_id and self.run_id:
                handle = self.client.get_workflow_handle(
                    self.workflow_id, run_id=self.run_id
                )
                try:
                    await handle.result()
                except Exception:
                    pass

    async def _execute_client_action(self, action: ClientAction):
        if action.HasField("do_signal"):
            await self._execute_signal_action(action.do_signal)
        elif action.HasField("do_update"):
            await self._execute_update_action(action.do_update)
        elif action.HasField("do_query"):
            await self._execute_query_action(action.do_query)
        elif action.HasField("nested_actions"):
            await self._execute_client_action_set(action.nested_actions)
        else:
            raise ValueError("Client action must have a recognized variant")

    async def _execute_signal_action(self, signal: DoSignal):
        if signal.HasField("do_signal_actions"):
            signal_name = "do_actions_signal"
            signal_args = signal.do_signal_actions
        elif signal.HasField("custom"):
            signal_name = signal.custom.name
            signal_args = list(signal.custom.args) if signal.custom.args else []  # type: ignore
        else:
            raise ValueError("DoSignal must have a recognizable variant")

        if signal.with_start:
            import uuid

            workflow_id = self.workflow_id or str(uuid.uuid4())
            handle = await self.client.start_workflow(
                self.workflow_type,
                self.workflow_input,
                id=workflow_id,
                task_queue="default",
            )
            await handle.signal(signal_name, signal_args)
            self.workflow_id = handle.id
            self.run_id = (
                handle.result_run_id if hasattr(handle, "result_run_id") else None
            )
        else:
            handle = self.client.get_workflow_handle(self.workflow_id)
            await handle.signal(signal_name, signal_args)

    async def _execute_update_action(self, update: DoUpdate):
        if update.HasField("do_actions"):
            update_name = "do_actions_update"
            update_args = update.do_actions
        elif update.HasField("custom"):
            update_name = update.custom.name
            update_args = list(update.custom.args) if update.custom.args else []  # type: ignore
        else:
            raise ValueError("DoUpdate must have a recognizable variant")

        try:
            if update.with_start:
                import uuid

                from temporalio.common import WorkflowIDConflictPolicy

                workflow_id = self.workflow_id or str(uuid.uuid4())
                handle = await self.client.start_workflow(
                    self.workflow_type,
                    self.workflow_input,
                    id=workflow_id,
                    task_queue="default",
                    id_conflict_policy=WorkflowIDConflictPolicy.USE_EXISTING,
                )
                await handle.execute_update(update_name, update_args)
                self.workflow_id = handle.id
                self.run_id = (
                    handle.result_run_id if hasattr(handle, "result_run_id") else None
                )
            else:
                handle = self.client.get_workflow_handle(self.workflow_id)
                await handle.execute_update(update_name, update_args)
        except Exception:
            if not update.failure_expected:
                raise

    async def _execute_query_action(self, query: DoQuery):
        try:
            if query.HasField("report_state"):
                handle = self.client.get_workflow_handle(self.workflow_id)
                await handle.query("report_state")
            elif query.HasField("custom"):
                handle = self.client.get_workflow_handle(self.workflow_id)
                query_args = list(query.custom.args) if query.custom.args else None
                await handle.query(query.custom.name, *query_args if query_args else [])
            else:
                raise ValueError("DoQuery must have a recognizable variant")
        except Exception:
            if not query.failure_expected:
                raise
