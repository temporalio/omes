import asyncio
from dataclasses import dataclass
from datetime import timedelta
from typing import Any, Awaitable, Dict, Optional, Sequence, Tuple

from temporalio import exceptions, workflow
from temporalio.common import RetryPolicy
from temporalio.workflow import ActivityCancellationType


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
class ExecuteActivity:
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
class Action:
    result: Optional[ResultAction]
    error: Optional[ErrorAction]
    continue_as_new: Optional[ContinueAsNewAction]
    sleep: Optional[SleepAction]
    query_handler: Optional[QueryHandlerAction]
    signal: Optional[SignalAction]
    execute_activity: Optional[ExecuteActivity]


@dataclass
class WorkflowParams:
    actions: list[Action]
    action_signal: str


def workflow_query(arg: str) -> str:
    return arg


@workflow.defn(name="kitchenSink")
class KitchenSinkWorkflow:
    def __init__(self) -> None:
        self._signal_name: str = ""
        self._pending_actions: asyncio.Queue[Action] = asyncio.Queue()
        self._signal_recieved: Dict[str, asyncio.Queue[Any]] = {}

    @workflow.signal(dynamic=True)
    async def handle_signal(self, signal_name: str, *varargs) -> None:
        if signal_name == self._signal_name:
            await self._pending_actions.put(varargs[0])
        else:
            await self._signal_recieved.setdefault(signal_name, asyncio.Queue()).put(
                varargs[0]
            )

    @workflow.run
    async def run(self, input: WorkflowParams) -> Any:
        self._signal_name = input.action_signal
        workflow.logger.info("Started kitchen sink workflow {}".format(input))
        for action in input.actions:
            should_return, return_value = await self.handle_action(input, action)
            if should_return:
                return return_value

        if input.action_signal != "":
            while not self._pending_actions.empty():
                action = await self._pending_actions.get()
                should_return, return_value = await self.handle_action(input, action)
                if should_return:
                    return return_value
                self._pending_actions.task_done()

        return None

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
            queue = self._signal_recieved.setdefault(
                action.signal.name, asyncio.Queue()
            )
            await queue.get()
            queue.task_done()
        elif action.execute_activity:
            execute_activity = action.execute_activity
            count = max(1, execute_activity.count)
            results = await asyncio.gather(
                *[launch_activity(execute_activity) for x in range(count)]
            )
            return (True, results[-1])
        else:
            raise exceptions.ApplicationError("unrecognized action")

        return (False, None)


def launch_activity(execute_activity: ExecuteActivity) -> Awaitable:
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
