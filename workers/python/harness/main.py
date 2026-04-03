from __future__ import annotations

import sys
from dataclasses import dataclass

from harness.client import ClientFactory
from harness.project import ProjectHandlers
from harness.worker import WorkerFactory, run_worker_cli


@dataclass(frozen=True)
class App:
    worker: WorkerFactory
    client_factory: ClientFactory
    project: ProjectHandlers | None = None


def run(app: App) -> None:
    argv = sys.argv[1:]
    if argv[:1] == ["worker"]:
        run_worker_cli(app.worker, app.client_factory, argv[1:])
        return
