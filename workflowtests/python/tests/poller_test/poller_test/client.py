from omes_starter import ClientConfig, ClientPool

_pool = ClientPool()


async def client_main(config: ClientConfig):
    """Called for each iteration - start a workflow and wait for result."""
    client = await _pool.get_or_connect(
        "default",
        config.server_address,
        **config.connect_kwargs(),
    )
    handle = await client.start_workflow(
        "MyWorkflow",
        id=f"wf-{config.run_id}-{config.iteration}",
        task_queue=config.task_queue,
    )
    result = await handle.result()
    print(
        f"Workflow result (run_id={config.run_id}, iteration={config.iteration}): {result}",
        flush=True,
    )


# Alternative: create a fresh connection per iteration using the SDK directly.
# Note: Python's Temporal Client has no close() method — cleanup relies on GC,
# which can exhaust file descriptors under sustained load. Prefer the pool above.
#
# from temporalio.client import Client
#
# async def client_main(config: ClientConfig):
#     client = await Client.connect(config.server_address, **config.connect_kwargs())
#     handle = await client.start_workflow(
#         "MyWorkflow",
#         id=f"wf-{config.run_id}-{config.iteration}",
#         task_queue=config.task_queue,
#     )
#     result = await handle.result()
#     print(
#         f"Workflow result (run_id={config.run_id}, iteration={config.iteration}): {result}",
#         flush=True,
#     )
