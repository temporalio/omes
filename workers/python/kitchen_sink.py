from __future__ import annotations

import asyncio
from typing import Any, Awaitable, Optional, Tuple

from protos.kitchen_sink_pb2 import (
    WorkflowInput,
    Action,
    ActionSet,
    ExecuteActivityAction,
)
from temporalio import exceptions, workflow


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
                should_return, return_value = await self.handle_action_set(action_set)
                if should_return:
                    return return_value

        # Run all actions from signals
        while True:
            action_set = await self.action_set_queue.get()
            should_return, return_value = await self.handle_action_set(action_set)
            if should_return:
                return return_value

    async def handle_action_set(self, action_set: ActionSet) -> Tuple[bool, Any]:
        should_return = False
        return_value = None
        # If these are non-concurrent, just execute and return if requested
        if not action_set.concurrent:
            for action in action_set.actions:
                should_return, return_value = await self.handle_action(action)
                if should_return:
                    return should_return, return_value
            return False, return_value

        # With a concurrent set, we'll create a task for each, only updating
        # return values if we should return, then awaiting on that or completion
        async def run_child(action: Action) -> None:
            maybe_should_return, maybe_return_value = await self.handle_action(action)
            if maybe_should_return:
                nonlocal should_return, return_value
                should_return = maybe_should_return
                return_value = maybe_return_value

        gather_fut = asyncio.gather(*[run_child(a) for a in action_set.actions])
        should_return_task = asyncio.create_task(
            workflow.wait_condition(lambda: should_return)
        )
        await asyncio.wait([gather_fut, should_return_task], return_when=asyncio.FIRST_COMPLETED)  # type: ignore
        return should_return, return_value

    async def handle_action(self, action: Action) -> Tuple[bool, Any]:
        if action.return_result:
            return True, action.return_result.return_this
        elif action.return_error:
            raise exceptions.ApplicationError(action.return_error.failure.message)
        elif action.continue_as_new:
            workflow.continue_as_new(action.continue_as_new.arguments)
        elif action.timer:
            await asyncio.sleep(action.timer.milliseconds / 1000)
        elif action.exec_activity:
            await launch_activity(action.exec_activity)
            return False, None
        elif action.exec_child_workflow:
            child_action = action.exec_child_workflow
            child = child_action.workflow_type or "kitchenSink"
            # TODO: Raw input
            args = child_action.input
            await workflow.execute_child_workflow(child, args=args)
            return False, None
        elif action.nested_action_set:
            return await self.handle_action_set(action.nested_action_set)
        else:
            raise exceptions.ApplicationError("unrecognized action")

        return False, None


def launch_activity(execute_activity: ExecuteActivityAction) -> Awaitable:
    args = execute_activity.arguments

    if args is None:
        args = []

    if execute_activity.is_local:
        activity_task = workflow.start_local_activity(
            activity=execute_activity.activity_type,
            args=args,
            schedule_to_close_timeout=execute_activity.schedule_to_close_timeout,
            start_to_close_timeout=execute_activity.start_to_close_timeout,
            schedule_to_start_timeout=execute_activity.schedule_to_start_timeout,
            retry_policy=execute_activity.retry_policy,
            # TODO: cancel type can be in local
        )
    else:
        activity_task = workflow.start_activity(
            activity=execute_activity.activity_type,
            args=args,
            task_queue=execute_activity.task_queue,
            schedule_to_close_timeout=execute_activity.schedule_to_close_timeout,
            start_to_close_timeout=execute_activity.start_to_close_timeout,
            schedule_to_start_timeout=execute_activity.schedule_to_start_timeout,
            heartbeat_timeout=execute_activity.heartbeat_timeout,
            retry_policy=execute_activity.retry_policy,
            cancellation_type=execute_activity.remote.cancellation_type,
        )
    # TODO: Handle cancels
    return activity_task
