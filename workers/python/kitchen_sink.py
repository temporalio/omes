from __future__ import annotations

import asyncio
from dataclasses import dataclass
from datetime import timedelta
from typing import Any, Awaitable, Optional, Sequence, Tuple

from temporalio import exceptions, workflow
from temporalio.common import RetryPolicy
from temporalio.workflow import ActivityCancellationType


@dataclass
class WorkflowParams:
    action_set: Optional[ActionSet]
    action_set_signal: str


@dataclass
class ActionSet:
    actions: Sequence[Action]
    concurrent: bool


@dataclass
class Action:
    result: Optional[ResultAction]
    error: Optional[ErrorAction]
    continue_as_new: Optional[ContinueAsNewAction]
    sleep: Optional[SleepAction]
    query_handler: Optional[QueryHandlerAction]
    signal: Optional[SignalAction]
    execute_activity: Optional[ExecuteActivityAction]
    execute_child_workflow: Optional[ExecuteChildWorkflowAction]
    nested_action_set: Optional[ActionSet]


@dataclass
class ResultAction:
    value: Any
    run_id: bool


@dataclass
class ErrorAction:
    message: str
    details: Any
    attempt: bool


@dataclass
class ContinueAsNewAction:
    while_above_zero: int


@dataclass
class SleepAction:
    millis: int


@dataclass
class QueryHandlerAction:
    name: str


@dataclass
class SignalAction:
    name: str


@dataclass
class ExecuteActivityAction:
    name: str
    task_queue: str
    args: Optional[Sequence[Any]]
    count: int
    index_as_arg: bool
    schedule_to_close_timeout_ms: int
    start_to_close_timeout_ms: int
    schedule_to_start_timeout_ms: int
    cancel_after_ms: int
    wait_for_cancellation: bool
    heartbeat_timeout_ms: int
    retry_max_attempts: int
    non_retryable_error_types: Optional[Sequence[str]]


@dataclass
class ExecuteChildWorkflowAction:
    name: Optional[str]
    count: int
    args: Optional[Sequence[Any]]
    params: Optional[WorkflowParams]


def workflow_query(arg: str) -> str:
    return arg


@workflow.defn(name="kitchenSink")
class KitchenSinkWorkflow:
    @workflow.run
    async def run(self, input: Optional[WorkflowParams]) -> Any:
        workflow.logger.info("Started kitchen sink workflow {}".format(input))
        if not input:
            return None
        if input.action_set and input.action_set.actions:
            should_return, return_value = await self.handle_action_set(
                input, input.action_set
            )
            if should_return:
                return return_value

        if input.action_set_signal:
            action_set_queue: asyncio.Queue[ActionSet] = asyncio.Queue()

            def enqueue(item: ActionSet) -> None:
                action_set_queue.put_nowait(item)

            workflow.set_signal_handler(input.action_set_signal, enqueue)
            while True:
                action_set = await action_set_queue.get()
                should_return, return_value = await self.handle_action_set(
                    input, action_set
                )
                if should_return:
                    return return_value

        return return_value

    async def handle_action_set(
        self, input: WorkflowParams, action_set: ActionSet
    ) -> Tuple[bool, Any]:
        should_return = False
        return_value = None
        # If these are non-concurrent, just execute and return if requested
        if not action_set.concurrent:
            for action in action_set.actions:
                should_return, return_value = await self.handle_action(input, action)
                if should_return:
                    return (should_return, return_value)
            return (False, return_value)
        # With a concurrent set, we'll create a task for each, only updating
        # return values if we should return, then awaiting on that or completion
        async def run_child(action: Action) -> None:
            maybe_should_return, maybe_return_value = await self.handle_action(
                input, action
            )
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

    async def handle_action(
        self, input: WorkflowParams, action: Action
    ) -> Tuple[bool, Any]:
        info = workflow.info()
        if action.result:
            if action.result.run_id:
                return (True, info.run_id)
            return (True, action.result.value)
        elif action.error:
            if action.error.attempt:
                raise exceptions.ApplicationError(
                    "attempt {}".format(action.error.attempt)
                )
            raise exceptions.ApplicationError(
                action.error.message, action.error.details
            )
        elif action.continue_as_new:
            if action.continue_as_new.while_above_zero > 0:
                action.continue_as_new.while_above_zero -= 1
                workflow.continue_as_new(input)
        elif action.sleep:
            await asyncio.sleep(action.sleep.millis / 1000)
        elif action.query_handler:
            workflow.set_query_handler(action.query_handler.name, workflow_query)
            return (True, None)
        elif action.signal:
            signal_event = asyncio.Event()

            def signal_handler(arg: Optional[Any] = None) -> None:
                signal_event.set()

            workflow.set_signal_handler(action.signal.name, signal_handler)
            await signal_event.wait()
        elif action.execute_activity:
            execute_activity = action.execute_activity
            count = max(1, execute_activity.count)
            results = await asyncio.gather(
                *[launch_activity(execute_activity) for x in range(count)]
            )
            return (False, results[-1])
        elif action.execute_child_workflow:
            child = action.execute_child_workflow.name or "kitchenSink"
            args = (
                [action.execute_child_workflow.params]
                if action.execute_child_workflow.params
                else action.execute_child_workflow.args or [None]
            )
            count = action.execute_child_workflow.count or 1
            await asyncio.gather(
                *[
                    workflow.execute_child_workflow(child, args=args)
                    for _ in range(count)
                ]
            )
            return (False, None)
        elif action.nested_action_set:
            return await self.handle_action_set(input, action.nested_action_set)
        else:
            raise exceptions.ApplicationError("unrecognized action")

        return (False, None)


def launch_activity(execute_activity: ExecuteActivityAction) -> Awaitable:
    if (
        execute_activity.start_to_close_timeout_ms == 0
        and execute_activity.schedule_to_close_timeout_ms == 0
    ):
        execute_activity.schedule_to_close_timeout_ms = 3 * 60 * 1000

    args = execute_activity.args
    if execute_activity.index_as_arg:
        args = [execute_activity.index_as_arg]

    if args is None:
        args = []

    cancelation_type = ActivityCancellationType.ABANDON
    if execute_activity.wait_for_cancellation:
        cancelation_type = ActivityCancellationType.WAIT_CANCELLATION_COMPLETED

    activity_task: Awaitable = workflow.start_activity(
        activity=execute_activity.name,
        args=args,
        task_queue=execute_activity.task_queue,
        schedule_to_close_timeout=timedelta(
            milliseconds=execute_activity.schedule_to_close_timeout_ms
        ),
        start_to_close_timeout=timedelta(
            milliseconds=execute_activity.start_to_close_timeout_ms
        ),
        schedule_to_start_timeout=timedelta(
            milliseconds=execute_activity.schedule_to_start_timeout_ms
        ),
        heartbeat_timeout=timedelta(milliseconds=execute_activity.heartbeat_timeout_ms),
        retry_policy=RetryPolicy(
            initial_interval=timedelta(milliseconds=1),
            backoff_coefficient=1.01,
            maximum_interval=timedelta(milliseconds=2),
            maximum_attempts=max(1, execute_activity.retry_max_attempts),
            non_retryable_error_types=execute_activity.non_retryable_error_types,
        ),
        cancellation_type=cancelation_type,
    )
    if execute_activity.cancel_after_ms > 0:
        activity_task = asyncio.wait_for(
            activity_task, execute_activity.cancel_after_ms / 1000
        )
    return activity_task
