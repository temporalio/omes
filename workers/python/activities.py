from temporalio import activity


@activity.defn(name="Noop")
async def noop_activity():
    return
