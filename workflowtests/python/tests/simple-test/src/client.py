from omes_starter import ClientConfig


async def client_main(config: ClientConfig):
    """Called for each iteration - start a workflow and wait for result."""
    handle = await config.client.start_workflow(
        "SimpleWorkflow",
        id=f"wf-{config.run_id}-{config.iteration}",
        task_queue=config.task_queue,
    )
    await handle.result()
