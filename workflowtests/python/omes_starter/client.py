from typing import Awaitable, Callable

from aiohttp import web

from .common import ClientConfig


class OmesClientStarter:
    """HTTP server for client lifecycle management.

    Provides endpoints:
    - POST /execute: Run user's execute function for one iteration
    - GET /info: readiness probe
    """

    def __init__(self):
        """Initialize the client starter.
        """
        self._execute_fn: Callable[[ClientConfig], Awaitable[None]] | None = None
        self._task_queue: str | None = None
        self._server_address: str | None = None
        self._namespace: str | None = None
        self._auth_header: str | None = None
        self._tls = False
        self._tls_server_name: str | None = None

    def on_execute(
        self, fn: Callable[[ClientConfig], Awaitable[None]]
    ) -> Callable[[ClientConfig], Awaitable[None]]:
        """Decorator to register the execute function.

        Example:
            @starter.on_execute
            async def execute(ctx: ClientConfig):
                client = await Client.connect(ctx.server_address, **ctx.connect_kwargs())
                handle = await client.start_workflow(...)
                await handle.result()
        """
        self._execute_fn = fn
        return fn

    async def _handle_execute(self, request: web.Request) -> web.Response:
        """POST /execute - call user's execute function."""
        try:
            data = await request.json()
            config = ClientConfig(
                server_address=self._server_address,
                namespace=self._namespace,
                task_queue=self._task_queue,
                run_id=data["run_id"],
                iteration=data["iteration"],
                auth_header=self._auth_header,
                tls=self._tls,
                tls_server_name=self._tls_server_name,
            )
            await self._execute_fn(config)
            return web.json_response({"success": True})
        except Exception as e:
            return web.json_response(
                {
                    "success": False,
                    "error": str(e),
                }
            )

    async def _handle_info(self, request: web.Request) -> web.Response:
        """GET /info - readiness probe."""
        return web.json_response({})

    def _run_with_args(self, args):
        """Start the HTTP server with pre-parsed args.

        Used by cli.py's run() function to pass already-parsed arguments.
        """
        self._task_queue = args.task_queue
        self._server_address = args.server_address
        self._namespace = args.namespace
        self._auth_header = args.auth_header
        self._tls = args.tls
        self._tls_server_name = args.tls_server_name

        app = web.Application()
        app.router.add_post("/execute", self._handle_execute)
        app.router.add_get("/info", self._handle_info)
        print(f"Client starter listening on port {args.port}")
        web.run_app(app, host="0.0.0.0", port=args.port, handle_signals=False)
