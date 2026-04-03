from __future__ import annotations

from dataclasses import dataclass

from .worker import WorkerFactory, run_worker_cli


@dataclass(frozen=True)
class App:
    worker: WorkerFactory


def run(app: App) -> None:
    run_worker_cli(app.worker)
