from __future__ import annotations

import asyncio
import logging
import unittest
from unittest.mock import AsyncMock, Mock, create_autospec, patch

from temporalio.client import Client

from harness import worker


class HarnessWorkerTests(unittest.IsolatedAsyncioTestCase):
    async def test_run_passes_shared_client_to_each_worker_factory(self) -> None:
        client = create_autospec(Client, instance=True, spec_set=True)
        config = object()
        logger = logging.getLogger("harness-worker-test")
        created_workers = [object(), object()]
        worker_factory = Mock(side_effect=created_workers)
        client_factory = AsyncMock(return_value=client)
        run_workers = AsyncMock()

        with (
            patch.object(
                worker, "configure_logger", autospec=True, return_value=logger
            ),
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
                    "--err-on-unimplemented",
                    "true",
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
        self.assertIs(first_context.logger, logger)
        self.assertIs(second_context.logger, logger)
        self.assertEqual(first_context.task_queue, "omes-1")
        self.assertEqual(second_context.task_queue, "omes-2")
        self.assertTrue(first_context.err_on_unimplemented)
        self.assertTrue(second_context.err_on_unimplemented)
