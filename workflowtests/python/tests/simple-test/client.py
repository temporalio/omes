from omes_starter import ExecuteContext, OmesClientStarter

starter = OmesClientStarter()


@starter.on_execute
async def execute(ctx: ExecuteContext):
    """Called for each iteration - start a workflow and wait for result."""
    handle = await ctx.client.start_workflow(
        "SimpleWorkflow",
        id=f"wf-{ctx.run_id}-{ctx.iteration}",
        task_queue=ctx.task_queue,
    )
    await handle.result()


if __name__ == "__main__":
    starter.run()  # Handles CLI args (--port, --task-queue, etc.)
