import asyncio
import os
from datetime import datetime, timedelta

from google.protobuf.duration_pb2 import Duration
from temporalio import activity
from temporalio.client import Client, Schedule, ScheduleActionStartWorkflow, ScheduleSpec, SchedulePolicy, ScheduleBackfill

from client_action_executor import ClientActionExecutor


@activity.defn(name="noop")
async def noop_activity():
    return


@activity.defn(name="delay")
async def delay_activity(delay_for: Duration):
    await asyncio.sleep(delay_for.ToSeconds())


@activity.defn(name="payload")
async def payload_activity(input_data: bytes, bytes_to_return: int) -> bytes:
    return os.urandom(bytes_to_return)


def make_schedule_id_unique(base_schedule_id: str, workflow_execution_id: str) -> str:
    sanitized_workflow_id = workflow_execution_id.replace("/", "-")
    return f"{base_schedule_id}-{sanitized_workflow_id}"


def create_client_activity(client: Client):
    @activity.defn(name="client")
    async def client_activity(client_activity_proto):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id
        executor = ClientActionExecutor(client, workflow_id, activity_info.task_queue)
        await executor.execute_client_sequence(client_activity_proto.client_sequence)

    return client_activity


def create_schedule_activities(client: Client):
    @activity.defn(name="CreateScheduleActivity")
    async def create_schedule_activity(action):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id

        if not action.schedule_id:
            raise ValueError("schedule_id is required")
        if not action.action:
            raise ValueError("action is required")

        task_queue = action.action.task_queue or activity_info.task_queue
        unique_schedule_id = make_schedule_id_unique(action.schedule_id, workflow_id)

        unique_workflow_id = action.action.workflow_id or ""
        if unique_workflow_id:
            unique_workflow_id = make_schedule_id_unique(unique_workflow_id, workflow_id)

        schedule_action = ScheduleActionStartWorkflow(
            action.action.workflow_type or "kitchenSink",
            *action.action.input,
            id=unique_workflow_id,
            task_queue=task_queue,
            execution_timeout=timedelta(seconds=action.action.workflow_execution_timeout.seconds) if action.action.workflow_execution_timeout.seconds else None,
            task_timeout=timedelta(seconds=action.action.workflow_task_timeout.seconds) if action.action.workflow_task_timeout.seconds else None,
        )

        spec = ScheduleSpec()
        if action.spec:
            spec.cron_expressions = list(action.spec.cron_expressions) if action.spec.cron_expressions else []
            if action.spec.jitter and action.spec.jitter.seconds:
                spec.jitter = timedelta(seconds=action.spec.jitter.seconds)

        policy = SchedulePolicy()
        if action.policies:
            if action.policies.catchup_window and action.policies.catchup_window.seconds:
                policy.catchup_window = timedelta(seconds=action.policies.catchup_window.seconds)

        backfills = []
        if action.backfill:
            for bf in action.backfill:
                backfills.append(ScheduleBackfill(
                    start_at=datetime.fromtimestamp(bf.start_timestamp),
                    end_at=datetime.fromtimestamp(bf.end_timestamp),
                ))

        schedule = Schedule(
            action=schedule_action,
            spec=spec,
            policy=policy,
        )

        schedule_options = {
            "trigger_immediately": action.policies.trigger_immediately if action.policies else False,
        }
        if backfills:
            schedule_options["backfills"] = backfills

        await client.create_schedule(
            unique_schedule_id,
            schedule,
            **schedule_options,
        )

    @activity.defn(name="DescribeScheduleActivity")
    async def describe_schedule_activity(action):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id

        if not action.schedule_id:
            raise ValueError("schedule_id is required")

        unique_schedule_id = make_schedule_id_unique(action.schedule_id, workflow_id)
        handle = client.get_schedule_handle(unique_schedule_id)
        return await handle.describe()

    @activity.defn(name="UpdateScheduleActivity")
    async def update_schedule_activity(action):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id

        if not action.schedule_id:
            raise ValueError("schedule_id is required")

        unique_schedule_id = make_schedule_id_unique(action.schedule_id, workflow_id)
        handle = client.get_schedule_handle(unique_schedule_id)

        async def updater(input):
            schedule = input.description.schedule
            if action.spec:
                schedule.spec.cron_expressions = list(action.spec.cron_expressions) if action.spec.cron_expressions else []
                if action.spec.jitter and action.spec.jitter.seconds:
                    schedule.spec.jitter = timedelta(seconds=action.spec.jitter.seconds)
            return schedule

        await handle.update(updater)

    @activity.defn(name="DeleteScheduleActivity")
    async def delete_schedule_activity(action):
        activity_info = activity.info()
        workflow_id = activity_info.workflow_id

        if not action.schedule_id:
            raise ValueError("schedule_id is required")

        unique_schedule_id = make_schedule_id_unique(action.schedule_id, workflow_id)
        handle = client.get_schedule_handle(unique_schedule_id)
        await handle.delete()

    return (
        create_schedule_activity,
        describe_schedule_activity,
        update_schedule_activity,
        delete_schedule_activity,
    )
