from __future__ import annotations

import asyncio
import unittest
from collections.abc import Awaitable, Callable
from typing import cast
from unittest.mock import AsyncMock, Mock, create_autospec, patch

from temporalio.client import Client
from temporalio.worker import PollerBehaviorAutoscaling, Worker

from harness import worker


class FakeWorker:
    def __init__(self, on_run: Callable[[], Awaitable[None]]) -> None:
        self._on_run = on_run
        self.run_calls = 0
        self.shutdown_calls = 0

    async def run(self) -> None:
        self.run_calls += 1
        await self._on_run()

    async def shutdown(self) -> None:
        self.shutdown_calls += 1


class HarnessWorkerTests(unittest.IsolatedAsyncioTestCase):
    async def test_run_passes_shared_client_and_context_to_each_worker_factory(
        self,
    ) -> None:
        client = create_autospec(Client, instance=True, spec_set=True)
        config = object()
        created_workers = [object(), object()]
        worker_factory = Mock(side_effect=created_workers)
        client_factory = AsyncMock(return_value=client)
        run_workers = AsyncMock()

        with (
            patch.object(
                worker, "build_client_config", autospec=True, return_value=config
            ) as build_client_config,
            patch.object(worker, "_run_workers", new=run_workers),
        ):
            await worker._run(
                worker_factory,
                client_factory,
                asyncio.Event(),
                [
                    "--task-queue",
                    "omes",
                    "--task-queue-suffix-index-start",
                    "1",
                    "--task-queue-suffix-index-end",
                    "2",
                ],
            )

        build_client_config.assert_called_once_with(
            server_address="localhost:7233",
            namespace="default",
            auth_header="",
            tls=False,
            tls_cert_path="",
            tls_key_path="",
            prom_listen_address=None,
        )
        client_factory.assert_awaited_once_with(config)
        run_workers.assert_awaited_once_with(created_workers, unittest.mock.ANY)
        self.assertEqual(worker_factory.call_count, 2)
        first_client, first_context = worker_factory.call_args_list[0].args
        second_client, second_context = worker_factory.call_args_list[1].args
        self.assertIs(first_client, client)
        self.assertIs(second_client, client)
        self.assertEqual(first_context.task_queue, "omes-1")
        self.assertEqual(second_context.task_queue, "omes-2")

    async def test_run_workers_shuts_down_all_workers_when_one_fails(self) -> None:
        async def fail_immediately() -> None:
            raise RuntimeError("boom")

        async def succeed_immediately() -> None:
            return None

        failing_worker = FakeWorker(fail_immediately)
        successful_worker = FakeWorker(succeed_immediately)

        with self.assertRaisesRegex(RuntimeError, "boom"):
            await worker._run_workers(
                [cast(Worker, failing_worker), cast(Worker, successful_worker)],
                asyncio.Event(),
            )

        self.assertEqual(failing_worker.run_calls, 1)
        self.assertEqual(successful_worker.run_calls, 1)
        self.assertEqual(failing_worker.shutdown_calls, 1)
        self.assertEqual(successful_worker.shutdown_calls, 1)
