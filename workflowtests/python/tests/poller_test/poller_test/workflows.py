from datetime import timedelta
from temporalio import workflow, activity

@activity.defn
async def my_activity() -> str:
    return "done"

@workflow.defn
class MyWorkflow:
    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_activity(
            my_activity,
            start_to_close_timeout=timedelta(seconds=60),
        )