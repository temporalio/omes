from __future__ import annotations

import logging
from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from temporalio.client import Client


@dataclass(frozen=True)
class ProjectRunInfo:
    task_queue: str
    run_id: str
    execution_id: str


@dataclass(frozen=True)
class ProjectInitContext:
    client: Client
    logger: logging.Logger
    run: ProjectRunInfo
    config_json: bytes


@dataclass(frozen=True)
class ProjectExecuteContext:
    client: Client
    logger: logging.Logger
    run: ProjectRunInfo
    iteration: int
    payload: bytes = b""


ProjectExecuteHandler = Callable[[ProjectExecuteContext], Awaitable[None]]
ProjectInitHandler = Callable[[ProjectInitContext], Awaitable[None]]


@dataclass(frozen=True)
class ProjectHandlers:
    execute: ProjectExecuteHandler
    init: ProjectInitHandler | None = None
