from temporalio import activity


@activity.defn(name="noop")
async def noop_activity():
    return
