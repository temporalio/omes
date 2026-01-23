from temporalio import workflow


@workflow.defn
class SimpleWorkflow:
    @workflow.run
    async def run(self) -> str:
        return "Hello from SimpleWorkflow!"
