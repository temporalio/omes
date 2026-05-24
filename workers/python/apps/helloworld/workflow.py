from __future__ import annotations

from temporalio import workflow


@workflow.defn
class HelloWorldWorkflow:
    @workflow.run
    async def run(self, name: str) -> str:
        return f"Hello {name}"
