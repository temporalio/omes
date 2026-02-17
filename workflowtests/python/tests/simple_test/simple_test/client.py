from omes_starter import ClientConfig
from temporalio.client import Client

# Optional ClientPool usage:
# from omes_starter import ClientPool
# _pool = ClientPool()
#
# async def client_main(config: ClientConfig):
#     client = await _pool.get_or_connect(
#         "default",
#         config.server_address,
#         **config.connect_kwargs(),
#     )
#     ...

async def client_main(config: ClientConfig):
    """Called for each iteration - start a workflow and wait for result."""
    client = await Client.connect(config.server_address, **config.connect_kwargs())
    handle = await client.start_workflow(
        "SimpleWorkflow",
        id=f"wf-{config.run_id}-{config.iteration}",
        task_queue=config.task_queue,
    )
    result = await handle.result()
    print(
        f"Workflow result (run_id={config.run_id}, iteration={config.iteration}): {result}",
        flush=True,
    )
