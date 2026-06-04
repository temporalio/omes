import time

from temporalio.client import Client, WithStartWorkflowOperation
from temporalio.common import RetryPolicy, WorkflowIDConflictPolicy
from temporalio.exceptions import ApplicationError

from activity_dispatch import activity_name_and_args, timeout_or_none
from protos.kitchen_sink_pb2 import (
    ClientAction,
    ClientActionSet,
    ClientSequence,
    DoQuery,
    DoSignal,
    DoStandaloneActivity,
    DoUpdate,
)


class ClientActionExecutor:
    def __init__(
        self,
        client: Client,
        workflow_id: str,
        task_queue: str,
        err_on_unimplemented: bool = False,
    ):
        self.client = client
        self.workflow_id: str = workflow_id
        self.workflow_type: str = "kitchenSink"
        self.workflow_input = None
        self.task_queue: str = task_queue
        self.err_on_unimplemented: bool = err_on_unimplemented

    async def execute_client_sequence(self, client_seq: ClientSequence):
        for action_set in client_seq.action_sets:
            await self._execute_client_action_set(action_set)

    async def _execute_client_action_set(self, action_set: ClientActionSet):
        if action_set.concurrent:
            if self.err_on_unimplemented:
                raise ApplicationError(
                    "concurrent client actions are not supported", non_retryable=True
                )
            # Skip concurrent actions when not erroring on unimplemented
            print("Skipping concurrent client actions (not implemented)")
            return

        for action in action_set.actions:
            await self._execute_client_action(action)

    async def _execute_client_action(self, action: ClientAction):
        if action.HasField("do_signal"):
            await self._execute_signal_action(action.do_signal)
        elif action.HasField("do_update"):
            await self._execute_update_action(action.do_update)
        elif action.HasField("do_query"):
            await self._execute_query_action(action.do_query)
        elif action.HasField("do_describe"):
            handle = self.client.get_workflow_handle(self.workflow_id)
            await handle.describe()
        elif action.HasField("do_standalone_activity"):
            await self._execute_standalone_activity(action.do_standalone_activity)
        elif action.HasField("nested_actions"):
            await self._execute_client_action_set(action.nested_actions)
        elif action.HasField("do_standalone_nexus_operation"):
            raise NotImplementedError("DoStandaloneNexusOperation is not supported")
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
            handle = await self.client.start_workflow(
                self.workflow_type,
                self.workflow_input,
                id=self.workflow_id,
                task_queue=self.task_queue,
                start_signal=signal_name,
                start_signal_args=[signal_args],
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
                start_op: WithStartWorkflowOperation = WithStartWorkflowOperation(
                    workflow=self.workflow_type,
                    args=[self.workflow_input],
                    id=self.workflow_id,
                    task_queue=self.task_queue,
                    id_conflict_policy=WorkflowIDConflictPolicy.USE_EXISTING,
                )

                handle = await self.client.execute_update_with_start_workflow(
                    update_name,
                    update_args,
                    start_workflow_operation=start_op,
                )

                workflow_handle = await start_op.workflow_handle()
            else:
                handle = self.client.get_workflow_handle(self.workflow_id)
                await handle.execute_update(update_name, update_args)
        except Exception:
            if not update.failure_expected:
                raise

    async def _execute_standalone_activity(self, sa: DoStandaloneActivity):
        act = sa.activity
        act_type, args = activity_name_and_args(act)
        await self.client.execute_activity(
            act_type,
            args=args,
            id=f"standalone-{self.workflow_id}-{time.time_ns()}",
            task_queue=act.task_queue,
            schedule_to_close_timeout=timeout_or_none(act, "schedule_to_close_timeout"),
            schedule_to_start_timeout=timeout_or_none(act, "schedule_to_start_timeout"),
            start_to_close_timeout=timeout_or_none(act, "start_to_close_timeout"),
            heartbeat_timeout=timeout_or_none(act, "heartbeat_timeout"),
            retry_policy=RetryPolicy.from_proto(act.retry_policy)
            if act.HasField("retry_policy")
            else None,
        )

    async def _execute_query_action(self, query: DoQuery):
        try:
            if query.HasField("report_state"):
                handle = self.client.get_workflow_handle(self.workflow_id)
                await handle.query("report_state", None)
            elif query.HasField("custom"):
                handle = self.client.get_workflow_handle(self.workflow_id)
                query_args = list(query.custom.args) if query.custom.args else None
                await handle.query(query.custom.name, *query_args if query_args else [])
            else:
                raise ValueError("DoQuery must have a recognizable variant")
        except Exception:
            if not query.failure_expected:
                raise
