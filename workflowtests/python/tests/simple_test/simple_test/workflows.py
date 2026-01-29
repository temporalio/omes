from temporalio import workflow


@workflow.defn
class SimpleWorkflow:
    @workflow.run
    async def run(self) -> str:
        print("WORKFLOW")
        return "Hello from SimpleWorkflow!"
