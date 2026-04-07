from __future__ import annotations

import sys
import unittest
from unittest.mock import patch

from temporalio.client import Client
from temporalio.worker import Worker

from harness.client import ClientConfig
from harness.main import App, run
from harness.project import ProjectExecuteContext, ProjectHandlers
from harness.worker import WorkerContext


def unused_worker_factory(_: WorkerContext) -> Worker:
    raise AssertionError("worker factory should not be called by CLI dispatch tests")


async def unused_client_factory(_: ClientConfig) -> Client:
    raise AssertionError("client factory should not be called by CLI dispatch tests")


async def unused_project_execute(_: Client, __: ProjectExecuteContext) -> None:
    raise AssertionError("project handlers should not be called by CLI dispatch tests")


def make_app(*, project: ProjectHandlers | None = None) -> App:
    return App(
        worker=unused_worker_factory,
        client_factory=unused_client_factory,
        project=project,
    )


class HarnessCLITests(unittest.TestCase):
    def test_worker_command_dispatches_to_worker_runner(self) -> None:
        app = make_app()

        with (
            patch("harness.main.run_worker_cli", autospec=True) as run_worker_cli,
            patch.object(sys, "argv", ["main.py", "worker", "--task-queue", "q"]),
        ):
            run(app)

        run_worker_cli.assert_called_once_with(
            app.worker, app.client_factory, ["--task-queue", "q"]
        )

    def test_project_server_requires_handlers(self) -> None:
        app = make_app()

        with patch.object(sys, "argv", ["main.py", "project-server"]):
            with self.assertRaisesRegex(
                SystemExit,
                "Wanted project-server but no project handlers registered for this app",
            ):
                run(app)

    def test_project_server_dispatches_to_project_runner(self) -> None:
        app = make_app(project=ProjectHandlers(execute=unused_project_execute))

        with (
            patch(
                "harness.main.run_project_server_cli", autospec=True
            ) as run_project_server_cli,
            patch.object(sys, "argv", ["main.py", "project-server", "--port", "9000"]),
        ):
            run(app)

        run_project_server_cli.assert_called_once_with(
            app.project, app.client_factory, ["--port", "9000"]
        )

    def test_unknown_command_exits(self) -> None:
        app = make_app()

        with patch.object(sys, "argv", ["main.py", "unexpected"]):
            with self.assertRaisesRegex(
                SystemExit, "Unknown command: \\['unexpected'\\]"
            ):
                run(app)
