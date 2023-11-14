from __future__ import annotations

import asyncio
from datetime import timedelta
from typing import Any, Awaitable, Optional

import temporalio.workflow
from temporalio import exceptions, workflow
from temporalio.common import RetryPolicy, RawValue

from protos.kitchen_sink_pb2 import (
    Action,
    ActionSet,
    ActivityCancellationType,
    ExecuteActivityAction,
    WorkflowInput,
)


@workflow.defn(name="kitchenSink")
class KitchenSinkWorkflow:
    action_set_queue: asyncio.Queue[ActionSet] = asyncio.Queue()

    @workflow.signal
    async def do_actions_signal(self, action_set: ActionSet) -> None:
        self.action_set_queue.put_nowait(action_set)

    @workflow.run
    async def run(self, input: Optional[WorkflowInput]) -> Any:
        workflow.logger.info("Started kitchen sink workflow")

        # Run all initial input actions
        if input and input.initial_actions:
            for action_set in input.initial_actions:
                return_value = await self.handle_action_set(action_set)
                if return_value is not None:
                    return return_value

        # Run all actions from signals
        while True:
            action_set = await self.action_set_queue.get()
            return_value = await self.handle_action_set(action_set)
            if return_value is not None:
                return return_value

    async def handle_action_set(self, action_set: ActionSet) -> Optional[Any]:
        return_value = None
        # If these are non-concurrent, just execute and return if requested
        if not action_set.concurrent:
            for action in action_set.actions:
                return_value = await self.handle_action(action)
                if return_value is not None:
                    return return_value
            return return_value

        # With a concurrent set, we'll create a task for each, only updating
        # return values if we should return, then awaiting on that or completion
        async def run_action(action: Action) -> None:
            maybe_return_value = await self.handle_action(action)

            if maybe_return_value is not None:
                nonlocal return_value
                return_value = maybe_return_value

        gather_fut = asyncio.gather(*[run_action(a) for a in action_set.actions])
        should_return_task = asyncio.create_task(
            workflow.wait_condition(lambda: return_value is not None)
        )
        await asyncio.wait(
            [gather_fut, should_return_task], return_when=asyncio.FIRST_COMPLETED
        )  # type: ignore
        return return_value

    async def handle_action(self, action: Action) -> Optional[Any]:
        if action.HasField("return_result"):
            return action.return_result.return_this
        elif action.HasField("return_error"):
            raise exceptions.ApplicationError(action.return_error.failure.message)
        elif action.HasField("continue_as_new"):
            workflow.continue_as_new(action.continue_as_new.arguments)
        elif action.HasField("timer"):
            await asyncio.sleep(action.timer.milliseconds / 1000)
        elif action.HasField("exec_activity"):
            await launch_activity(action.exec_activity)
        elif action.HasField("exec_child_workflow"):
            child_action = action.exec_child_workflow
            child = child_action.workflow_type or "kitchenSink"
            args = [RawValue(i) for i in child_action.input]
            await workflow.execute_child_workflow(child, id=child_action.workflow_id, args=args)
        elif action.HasField("nested_action_set"):
            return await self.handle_action_set(action.nested_action_set)
        else:
            raise exceptions.ApplicationError("unrecognized action: " + str(action))

        return None


def launch_activity(execute_activity: ExecuteActivityAction) -> Awaitable:
    args = execute_activity.arguments

    if args is None:
        args = []

    if execute_activity.HasField("is_local"):
        activity_task = workflow.start_local_activity(
            activity=execute_activity.activity_type,
            args=args,
            schedule_to_close_timeout=timeout_or_none(
                execute_activity, "schedule_to_close_timeout"
            ),
            start_to_close_timeout=timeout_or_none(
                execute_activity, "start_to_close_timeout"
            ),
            schedule_to_start_timeout=timeout_or_none(
                execute_activity, "schedule_to_start_timeout"
            ),
            retry_policy=RetryPolicy.from_proto(execute_activity.retry_policy)
            if execute_activity.HasField("retry_policy")
            else None,
            # TODO: cancel type can be in local
        )
    else:
        activity_task = workflow.start_activity(
            activity=execute_activity.activity_type,
            args=args,
            task_queue=execute_activity.task_queue,
            schedule_to_close_timeout=timeout_or_none(
                execute_activity, "schedule_to_close_timeout"
            ),
            start_to_close_timeout=timeout_or_none(
                execute_activity, "start_to_close_timeout"
            ),
            schedule_to_start_timeout=timeout_or_none(
                execute_activity, "schedule_to_start_timeout"
            ),
            heartbeat_timeout=timeout_or_none(execute_activity, "heartbeat_timeout"),
            retry_policy=RetryPolicy.from_proto(execute_activity.retry_policy)
            if execute_activity.HasField("retry_policy")
            else None,
            cancellation_type=convert_act_cancel_type(
                execute_activity.remote.cancellation_type
            ),
        )
    # TODO: Handle cancels
    return activity_task


# Various proto conversions below ==============================================


def timeout_or_none(
    activity_action: ExecuteActivityAction, timeout_field: str
) -> Optional[timedelta]:
    if activity_action.HasField(timeout_field):
        return timedelta(
            seconds=getattr(activity_action, timeout_field).seconds,
            microseconds=getattr(activity_action, timeout_field).nanos / 1000,
        )
    return None


def convert_act_cancel_type(
    ctype: ActivityCancellationType,
) -> temporalio.workflow.ActivityCancellationType:
    if ctype == ActivityCancellationType.TRY_CANCEL:
        return temporalio.workflow.ActivityCancellationType.TRY_CANCEL
    elif ctype == ActivityCancellationType.WAIT_CANCELLATION_COMPLETED:
        return temporalio.workflow.ActivityCancellationType.WAIT_CANCELLATION_COMPLETED
    elif ctype == ActivityCancellationType.ABANDON:
        return temporalio.workflow.ActivityCancellationType.ABANDON
    else:
        raise NotImplementedError("Unknown cancellation type " + str(ctype))
