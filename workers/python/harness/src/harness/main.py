from __future__ import annotations

import sys
from dataclasses import dataclass

from harness.client import ClientFactory
from harness.project import ProjectHandlers, run_project_server_cli
from harness.worker import WorkerFactory, run_worker_cli


@dataclass(frozen=True)
class App:
    worker: WorkerFactory
    client_factory: ClientFactory
    project: ProjectHandlers | None = None


def run(app: App) -> None:
    argv = sys.argv[1:]
    # If no arg provided, fallback to existing worker CLI usage.
    # Preserves direct `python main.py` usage.
    if not argv or argv[:1] == ["worker"]:
        worker_argv = argv[1:] if argv[:1] == ["worker"] else argv
        run_worker_cli(app.worker, app.client_factory, worker_argv)
    elif argv[:1] == ["project-server"]:
        if app.project is None:
            raise SystemExit(
                "Wanted project-server but no project handlers registered for this app"
            )
        run_project_server_cli(app.project, app.client_factory, argv[1:])
    else:
        raise SystemExit(
            f"Unknown command: {argv[:1]}. Expected 'worker' or 'project-server'"
        )
