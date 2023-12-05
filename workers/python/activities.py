import asyncio

from google.protobuf.duration_pb2 import Duration
from temporalio import activity


@activity.defn(name="noop")
async def noop_activity():
    return


@activity.defn(name="delay")
async def delay_activity(delay_for: Duration):
    await asyncio.sleep(delay_for.ToSeconds())
