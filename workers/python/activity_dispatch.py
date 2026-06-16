from datetime import timedelta
from typing import Any, Optional

from protos.kitchen_sink_pb2 import ExecuteActivityAction


def timeout_or_none(act: ExecuteActivityAction, field: str) -> Optional[timedelta]:
    if act.HasField(field):
        d = getattr(act, field)
        return timedelta(seconds=d.seconds, microseconds=d.nanos / 1000)
    return None


def activity_name_and_args(act: ExecuteActivityAction) -> tuple[str, list[Any]]:
    """Map an ExecuteActivityAction to its registered activity name and args.

    Shared by the workflow-scheduled path and the standalone-activity path.
    """
    if act.HasField("delay"):
        return "delay", [act.delay]
    elif act.HasField("payload"):
        input_data = bytes(i % 256 for i in range(act.payload.bytes_to_receive))
        return "payload", [input_data, act.payload.bytes_to_return]
    elif act.HasField("client"):
        return "client", [act.client]
    elif act.HasField("retryable_error"):
        return "retryable_error", [act.retryable_error]
    elif act.HasField("timeout"):
        return "timeout", [act.timeout]
    elif act.HasField("heartbeat"):
        return "heartbeat", [act.heartbeat]
    return "noop", []
