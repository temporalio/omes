import argparse
import asyncio
import logging
import time
import traceback
from typing import Awaitable, Callable

import temporalio
from aiohttp import web
from temporalio.client import Client

from .common import ClientConfig

logger = logging.getLogger(__name__)


class OmesClientStarter:
    """HTTP server for client lifecycle management.

    Provides endpoints:
    - POST /execute: Run user's execute function for one iteration
    - POST /shutdown: Graceful shutdown with request draining
    - GET /info: SDK metadata
    """

    def __init__(self, **client_options):
        """Initialize the client starter.

        Args:
            **client_options: Options passed to Client.connect()
                              (data_converter, interceptors, etc.)
        """
        print("OMES STARTER CLIENT INIT")
        self._client_options = client_options
        self._execute_fn: Callable[[ClientConfig], Awaitable[None]] | None = None
        self._client: Client | None = None
        self._task_queue: str | None = None
        self._runner: web.AppRunner | None = None
        self._active_requests = 0
        self._active_lock = asyncio.Lock()
        self._shutting_down = False

    def on_execute(
        self, fn: Callable[[ClientConfig], Awaitable[None]]
    ) -> Callable[[ClientConfig], Awaitable[None]]:
        """Decorator to register the execute function.

        Example:
            @starter.on_execute
            async def execute(ctx: ClientConfig):
                handle = await ctx.client.start_workflow(...)
                await handle.result()
        """
        self._execute_fn = fn
        return fn

    async def _handle_execute(self, request: web.Request) -> web.Response:
        """POST /execute - call user's execute function."""
        print("HANDLING EXECUTE")
        if self._shutting_down:
            return web.json_response(
                {"success": False, "error": "Starter is shutting down"}
            )

        async with self._active_lock:
            self._active_requests += 1

        try:
            data = await request.json()
            config = ClientConfig(
                client=self._client,
                task_queue=self._task_queue,
                run_id=data["run_id"],
                iteration=data["iteration"],
            )
            print("CALLIN EXECUTE FN")
            await self._execute_fn(config)
            return web.json_response({"success": True})
        except Exception as e:
            return web.json_response(
                {
                    "success": False,
                    "error": str(e),
                    "traceback": traceback.format_exc(),
                }
            )
        finally:
            async with self._active_lock:
                self._active_requests -= 1

    async def _handle_shutdown(self, request: web.Request) -> web.Response:
        """POST /shutdown - graceful shutdown with drain."""
        data = await request.json() if request.body_exists else {}
        drain_timeout_ms = data.get("drain_timeout_ms", 30000)
        asyncio.create_task(self._shutdown_with_drain(drain_timeout_ms / 1000))
        return web.json_response({"status": "shutting_down"})

    async def _shutdown_with_drain(self, timeout: float):
        """Wait for in-flight requests, then shutdown."""
        self._shutting_down = True
        start = time.time()

        # Wait for active requests to complete
        while self._active_requests > 0 and (time.time() - start) < timeout:
            await asyncio.sleep(0.1)

        if self._active_requests > 0:
            logger.warning(
                f"Forcing shutdown with {self._active_requests} active requests"
            )

        if self._runner:
            await self._runner.cleanup()

    async def _handle_info(self, request: web.Request) -> web.Response:
        """GET /info - SDK metadata."""
        return web.json_response(
            {
                "sdk_language": "python",
                "sdk_version": temporalio.__version__,
                "starter_version": "0.1.0",
            }
        )

    def run(self):
        """Start the HTTP server. Parses CLI args and creates Temporal client."""
        parser = argparse.ArgumentParser()
        parser.add_argument("--port", type=int, default=8080)
        parser.add_argument("--task-queue", required=True)
        parser.add_argument("--server-address", default="localhost:7233")
        parser.add_argument("--namespace", default="default")
        args = parser.parse_args()
        self._run_with_args(args)

    def _run_with_args(self, args):
        """Start the HTTP server with pre-parsed args.

        Used by cli.py's run() function to pass already-parsed arguments.
        """
        self._task_queue = args.task_queue

        app = web.Application()
        app.router.add_post("/execute", self._handle_execute)
        app.router.add_post("/shutdown", self._handle_shutdown)
        app.router.add_get("/info", self._handle_info)

        async def start():
            # Create Temporal client at startup
            self._client = await Client.connect(
                args.server_address,
                namespace=args.namespace,
                **self._client_options,
            )

            self._runner = web.AppRunner(app)
            await self._runner.setup()
            site = web.TCPSite(self._runner, "0.0.0.0", args.port)
            await site.start()
            print(f"Client starter listening on port {args.port}")

            # Keep running until shutdown
            while self._runner._server:
                await asyncio.sleep(1)

        asyncio.run(start())


def run_client(
    handler: Callable[[ClientConfig], Awaitable[None]],
    **client_options,
) -> None:
    """Convenience function to run a client with the given execute handler.

    Note: Prefer using omes_starter.run() with the new subcommand pattern.

    Example:
        # src/client.py
        async def client_main(config: ClientConfig):
            handle = await config.client.start_workflow(...)
            await handle.result()

        # Call directly:
        run_client(client_main)

    Args:
        handler: Async function called for each /execute request
        **client_options: Options passed to Client.connect()
    """
    starter = OmesClientStarter(**client_options)
    starter._execute_fn = handler
    starter.run()
