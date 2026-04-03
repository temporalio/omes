from __future__ import annotations

import argparse
import asyncio
import logging
from collections.abc import Awaitable, Callable, Sequence
from dataclasses import dataclass

import grpc
from grpc import aio as grpc_aio
from temporalio.client import Client

from temporalio.api.enums.v1 import IndexedValueType
from temporalio.api.operatorservice.v1 import AddSearchAttributesRequest

from harness.api import api_pb2, api_pb2_grpc
from harness.client import ClientFactory, build_client_config


@dataclass(frozen=True)
class ProjectRunInfo:
    run_id: str
    execution_id: str


@dataclass(frozen=True)
class ProjectInitInfo:
    logger: logging.Logger
    run: ProjectRunInfo
    task_queue: str
    config_json: bytes


@dataclass(frozen=True)
class ProjectExecuteInfo:
    logger: logging.Logger
    run: ProjectRunInfo
    task_queue: str
    iteration: int
    payload: bytes = b""


ProjectExecuteHandler = Callable[[Client, ProjectExecuteInfo], Awaitable[None]]
ProjectInitHandler = Callable[[Client, ProjectInitInfo], Awaitable[None]]


@dataclass(frozen=True)
class ProjectHandlers:
    execute: ProjectExecuteHandler
    init: ProjectInitHandler | None = None


class ProjectServiceServer(api_pb2_grpc.ProjectServiceServicer):
    def __init__(
        self, handlers: ProjectHandlers, client_factory: ClientFactory
    ) -> None:
        self._handlers = handlers
        self._client_factory = client_factory
        self._client: Client | None = None
        self._run: ProjectRunInfo | None = None
        self._logger = logging.getLogger(__name__)

    async def Init(
        self,
        request: api_pb2.InitRequest,
        context: grpc_aio.ServicerContext,
    ) -> api_pb2.InitResponse:
        if not request.task_queue:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "task_queue required")
        if not request.execution_id:
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "execution_id required"
            )
        if not request.run_id:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "run_id required")

        conn = request.connect_options
        if not conn.server_address:
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "server_address required"
            )
        if not conn.namespace:
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "namespace required"
            )

        config = build_client_config(
            server_address=conn.server_address,
            namespace=conn.namespace,
            auth_header=conn.auth_header,
            tls=conn.enable_tls,
            tls_cert_path=conn.tls_cert_path,
            tls_key_path=conn.tls_key_path,
            tls_server_name=conn.tls_server_name or None,
            disable_host_verification=conn.disable_host_verification,
        )
        try:
            self._client = await self._client_factory(config)
        except Exception as e:
            await context.abort(
                grpc.StatusCode.INTERNAL, f"failed to create client: {e}"
            )

        self._run = ProjectRunInfo(
            run_id=request.run_id,
            execution_id=request.execution_id,
        )

        if self._handlers.init is not None:
            init_info = ProjectInitInfo(
                logger=self._logger,
                run=self._run,
                task_queue=request.task_queue,
                config_json=request.config_json,
            )
            try:
                await self._handlers.init(self._client, init_info)
            except Exception as e:
                await context.abort(
                    grpc.StatusCode.INTERNAL, f"init handler failed: {e}"
                )

        if request.register_search_attributes:
            await self._register_search_attributes(conn.namespace)

        return api_pb2.InitResponse()

    async def Execute(
        self,
        request: api_pb2.ExecuteRequest,
        context: grpc_aio.ServicerContext,
    ) -> api_pb2.ExecuteResponse:
        if self._client is None or self._run is None:
            await context.abort(
                grpc.StatusCode.FAILED_PRECONDITION,
                "Init must be called before Execute",
            )

        exec_info = ProjectExecuteInfo(
            logger=self._logger,
            run=self._run,
            task_queue=request.task_queue,
            iteration=request.iteration,
            payload=request.payload,
        )
        try:
            await self._handlers.execute(self._client, exec_info)
        except Exception as e:
            await context.abort(
                grpc.StatusCode.INTERNAL, f"execute handler failed: {e}"
            )
        return api_pb2.ExecuteResponse()

    async def _register_search_attributes(self, namespace: str) -> None:
        assert self._client is not None
        try:
            await self._client.operator_service.add_search_attributes(
                AddSearchAttributesRequest(
                    namespace=namespace,
                    search_attributes={
                        "KS_Keyword": IndexedValueType.INDEXED_VALUE_TYPE_KEYWORD,
                        "KS_Int": IndexedValueType.INDEXED_VALUE_TYPE_INT,
                        "OmesExecutionID": IndexedValueType.INDEXED_VALUE_TYPE_KEYWORD,
                    },
                ),
            )
        except Exception as e:
            err_str = str(e)
            if "already exists" in err_str:
                return
            if "attributes mapping unavailable" in err_str:
                return
            raise RuntimeError(f"failed to register search attributes: {e}") from e


async def _serve(
    handlers: ProjectHandlers, client_factory: ClientFactory, port: int
) -> None:
    server = grpc_aio.server()
    api_pb2_grpc.add_ProjectServiceServicer_to_server(
        ProjectServiceServer(handlers, client_factory), server
    )
    server.add_insecure_port(f"0.0.0.0:{port}")
    await server.start()
    logging.getLogger(__name__).info("Project server listening on port %d", port)
    await server.wait_for_termination()


def run_project_server_cli(
    handlers: ProjectHandlers,
    client_factory: ClientFactory,
    argv: Sequence[str],
) -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8080, help="gRPC listen port")
    args = parser.parse_args(argv)
    asyncio.run(_serve(handlers, client_factory, args.port))
