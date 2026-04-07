from __future__ import annotations

import unittest
from unittest.mock import AsyncMock, create_autospec, patch

import grpc  # type: ignore[import]
from grpc import aio as grpc_aio
from temporalio.client import Client

from harness import project
from harness.api import api_pb2


class AbortError(Exception):
    def __init__(self, status_code: grpc.StatusCode, details: str) -> None:
        super().__init__(details)
        self.status_code = status_code
        self.details = details


def make_servicer_context():
    context = create_autospec(grpc_aio.ServicerContext, instance=True, spec_set=True)

    async def abort(status_code: grpc.StatusCode, details: str) -> None:
        raise AbortError(status_code, details)

    context.abort.side_effect = abort
    return context


def make_client() -> Client:
    return create_autospec(Client, instance=True, spec_set=True)


def make_connect_options() -> api_pb2.ConnectOptions:
    return api_pb2.ConnectOptions(
        namespace="default",
        server_address="localhost:7233",
        auth_header="Bearer token",
        enable_tls=True,
        tls_cert_path="/tmp/cert.pem",
        tls_key_path="/tmp/key.pem",
        tls_server_name="server.local",
        disable_host_verification=True,
    )


def make_init_request(**overrides: object) -> api_pb2.InitRequest:
    request = api_pb2.InitRequest(
        execution_id="exec-id",
        run_id="run-id",
        task_queue="task-queue",
        connect_options=make_connect_options(),
        config_json=b'{"hello":"world"}',
    )
    for field, value in overrides.items():
        setattr(request, field, value)
    return request


def make_execute_request(**overrides: object) -> api_pb2.ExecuteRequest:
    request = api_pb2.ExecuteRequest(
        iteration=7,
        task_queue="task-queue",
        payload=b"payload",
    )
    for field, value in overrides.items():
        setattr(request, field, value)
    return request


class HarnessProjectTests(unittest.IsolatedAsyncioTestCase):
    async def test_init_rejects_missing_task_queue(self) -> None:
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=AsyncMock()),
            AsyncMock(),
        )

        with self.assertRaises(AbortError) as error:
            await server.Init(
                make_init_request(task_queue=""),
                make_servicer_context(),
            )

        self.assertEqual(error.exception.status_code, grpc.StatusCode.INVALID_ARGUMENT)
        self.assertEqual(error.exception.details, "task_queue required")

    async def test_execute_requires_init(self) -> None:
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=AsyncMock()),
            AsyncMock(),
        )

        with self.assertRaises(AbortError) as error:
            await server.Execute(make_execute_request(), make_servicer_context())

        self.assertEqual(
            error.exception.status_code, grpc.StatusCode.FAILED_PRECONDITION
        )
        self.assertEqual(error.exception.details, "Init must be called before Execute")

    async def test_execute_rejects_missing_task_queue(self) -> None:
        client = make_client()
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=AsyncMock()),
            AsyncMock(return_value=client),
        )
        with patch.object(
            project, "build_client_config", autospec=True, return_value=object()
        ):
            await server.Init(make_init_request(), make_servicer_context())

        with self.assertRaises(AbortError) as error:
            await server.Execute(
                make_execute_request(task_queue=""),
                make_servicer_context(),
            )

        self.assertEqual(error.exception.status_code, grpc.StatusCode.INVALID_ARGUMENT)
        self.assertEqual(error.exception.details, "task_queue required")

    async def test_init_passes_run_metadata_to_handler(self) -> None:
        client = make_client()
        init_handler = AsyncMock()
        client_factory = AsyncMock(return_value=client)
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=AsyncMock(), init=init_handler),
            client_factory,
        )

        config = object()
        with patch.object(
            project, "build_client_config", autospec=True, return_value=config
        ) as build_config:
            response = await server.Init(make_init_request(), make_servicer_context())

        self.assertIsInstance(response, api_pb2.InitResponse)
        build_config.assert_called_once_with(
            server_address="localhost:7233",
            namespace="default",
            auth_header="Bearer token",
            tls=True,
            tls_cert_path="/tmp/cert.pem",
            tls_key_path="/tmp/key.pem",
            tls_server_name="server.local",
            disable_host_verification=True,
        )
        client_factory.assert_awaited_once_with(config)
        init_handler.assert_awaited_once()
        await_args = init_handler.await_args
        if await_args is None:
            self.fail("init handler was not awaited")
        handler_client, init_info = await_args.args
        self.assertIs(handler_client, client)
        self.assertEqual(init_info.run.run_id, "run-id")
        self.assertEqual(init_info.run.execution_id, "exec-id")
        self.assertEqual(init_info.task_queue, "task-queue")
        self.assertEqual(init_info.config_json, b'{"hello":"world"}')

    async def test_execute_passes_iteration_payload_and_run_metadata(self) -> None:
        client = make_client()
        execute_handler = AsyncMock()
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=execute_handler),
            AsyncMock(return_value=client),
        )
        with patch.object(
            project, "build_client_config", autospec=True, return_value=object()
        ):
            await server.Init(make_init_request(), make_servicer_context())

        response = await server.Execute(make_execute_request(), make_servicer_context())

        self.assertIsInstance(response, api_pb2.ExecuteResponse)
        execute_handler.assert_awaited_once()
        await_args = execute_handler.await_args
        if await_args is None:
            self.fail("execute handler was not awaited")
        handler_client, execute_info = await_args.args
        self.assertIs(handler_client, client)
        self.assertEqual(execute_info.iteration, 7)
        self.assertEqual(execute_info.payload, b"payload")
        self.assertEqual(execute_info.task_queue, "task-queue")
        self.assertEqual(execute_info.run.run_id, "run-id")
        self.assertEqual(execute_info.run.execution_id, "exec-id")

    async def test_client_factory_failure_maps_to_internal_error(self) -> None:
        server = project.ProjectServiceServer(
            project.ProjectHandlers(execute=AsyncMock()),
            AsyncMock(side_effect=RuntimeError("boom")),
        )

        with patch.object(
            project, "build_client_config", autospec=True, return_value=object()
        ):
            with self.assertRaises(AbortError) as error:
                await server.Init(make_init_request(), make_servicer_context())

        self.assertEqual(error.exception.status_code, grpc.StatusCode.INTERNAL)
        self.assertEqual(error.exception.details, "failed to create client: boom")

    async def test_init_handler_failure_does_not_leave_server_initialized(self) -> None:
        client = make_client()
        server = project.ProjectServiceServer(
            project.ProjectHandlers(
                execute=AsyncMock(),
                init=AsyncMock(side_effect=RuntimeError("bad init")),
            ),
            AsyncMock(return_value=client),
        )

        with patch.object(
            project, "build_client_config", autospec=True, return_value=object()
        ):
            with self.assertRaises(AbortError) as error:
                await server.Init(make_init_request(), make_servicer_context())

        self.assertEqual(error.exception.status_code, grpc.StatusCode.INTERNAL)
        self.assertEqual(error.exception.details, "init handler failed: bad init")

        with self.assertRaises(AbortError) as execute_error:
            await server.Execute(make_execute_request(), make_servicer_context())
        self.assertEqual(
            execute_error.exception.status_code,
            grpc.StatusCode.FAILED_PRECONDITION,
        )

    async def test_execute_handler_failure_maps_to_internal_error(self) -> None:
        client = make_client()
        server = project.ProjectServiceServer(
            project.ProjectHandlers(
                execute=AsyncMock(side_effect=RuntimeError("bad execute"))
            ),
            AsyncMock(return_value=client),
        )
        with patch.object(
            project, "build_client_config", autospec=True, return_value=object()
        ):
            await server.Init(make_init_request(), make_servicer_context())

        with self.assertRaises(AbortError) as error:
            await server.Execute(make_execute_request(), make_servicer_context())

        self.assertEqual(error.exception.status_code, grpc.StatusCode.INTERNAL)
        self.assertEqual(error.exception.details, "execute handler failed: bad execute")
