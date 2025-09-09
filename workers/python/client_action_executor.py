from typing import Any

from temporalio.api.common.v1 import WorkflowExecution
from temporalio.api.workflowservice.v1 import (
    DescribeWorkflowExecutionRequest,
    DescribeWorkflowExecutionResponse,
)
from temporalio.client import Client, WithStartWorkflowOperation
from temporalio.common import WorkflowIDConflictPolicy
from temporalio.exceptions import ApplicationError

from protos.kitchen_sink_pb2 import (
    ClientAction,
    ClientActionSet,
    ClientSequence,
    DoQuery,
    DoSignal,
    DoUpdate,
)


class ClientActionExecutor:
    def __init__(self, client: Client, workflow_id: str, task_queue: str):
        self.client = client
        self.workflow_id: str = workflow_id
        self.workflow_type: str = "kitchenSink"
        self.workflow_input = None
        self.task_queue: str = task_queue

    async def execute_client_sequence(self, client_seq: ClientSequence):
        for action_set in client_seq.action_sets:
            await self._execute_client_action_set(action_set)

    async def _execute_client_action_set(self, action_set: ClientActionSet):
        if action_set.concurrent:
            raise ApplicationError(
                "concurrent client actions are not supported", non_retryable=True
            )

        for action in action_set.actions:
            await self._execute_client_action(action)

    async def _execute_client_action(self, action: ClientAction):
        if action.HasField("do_signal"):
            await self._execute_signal_action(action.do_signal)
        elif action.HasField("do_update"):
            await self._execute_update_action(action.do_update)
        elif action.HasField("do_query"):
            await self._execute_query_action(action.do_query)
        elif action.HasField("do_self_describe"):
            await self._execute_self_describe_action(action.do_self_describe)
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

    async def _execute_self_describe_action(self, self_describe):
        # Get the current workflow execution details
        try:
            # Use current workflow ID if not specified
            workflow_id = self_describe.workflow_id
            if not workflow_id:
                workflow_id = self.workflow_id
                
            # Create the request object
            request = DescribeWorkflowExecutionRequest(
                namespace=self_describe.namespace,
                execution=WorkflowExecution(
                    workflow_id=workflow_id,
                    run_id=self_describe.run_id,
                ),
            )

            # Call the service method
            resp: DescribeWorkflowExecutionResponse = (
                await self.client.workflow_service.describe_workflow_execution(request)
            )

            # Log the workflow execution details
            print("Workflow Execution Details:")
            print(
                f"  Workflow ID: {resp.workflow_execution_info.execution.workflow_id}"
            )
            print(f"  Run ID: {resp.workflow_execution_info.execution.run_id}")
            print(f"  Type: {resp.workflow_execution_info.type.name}")
            print(f"  Status: {resp.workflow_execution_info.status}")
            print(f"  Start Time: {resp.workflow_execution_info.start_time}")
            if resp.workflow_execution_info.close_time:
                print(f"  Close Time: {resp.workflow_execution_info.close_time}")
            print(f"  History Length: {resp.workflow_execution_info.history_length}")
            print(f"  Task Queue: {resp.workflow_execution_info.task_queue}")

        except Exception as e:
            raise ApplicationError(
                f"Failed to describe workflow execution: {e}", non_retryable=True
            )
