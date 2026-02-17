from dataclasses import dataclass
from typing import Any

try:
    from temporalio.service import TLSConfig
except Exception:
    TLSConfig = None


@dataclass
class ClientConfig:
    """Config passed to client's execute function.

    Attributes:
        server_address: Temporal server host:port
        namespace: Temporal namespace
        task_queue: Task queue for workflows
        run_id: Unique ID for this load test run (from /execute request)
        iteration: Current iteration number (from /execute request)
        auth_header: Optional Authorization header value
        tls: Whether TLS is enabled
        tls_server_name: Optional TLS SNI override
    """

    server_address: str
    namespace: str
    task_queue: str
    run_id: str
    iteration: int
    auth_header: str | None = None
    tls: bool = False
    tls_server_name: str | None = None

    def connect_kwargs(self) -> dict[str, Any]:
        kwargs: dict[str, Any] = {"namespace": self.namespace}
        if self.auth_header:
            kwargs["rpc_metadata"] = {"authorization": self.auth_header}
        if self.tls_server_name:
            kwargs["tls"] = _tls_config_with_server_name(self.tls_server_name)
        elif self.tls:
            kwargs["tls"] = True
        return kwargs


@dataclass
class WorkerConfig:
    """Config passed to worker's configure function.

    Attributes:
        server_address: Temporal server host:port
        namespace: Temporal namespace
        task_queue: Task queue for the worker
        prom_listen_address: Optional Prometheus metrics endpoint address
        auth_header: Optional Authorization header value
        tls: Whether TLS is enabled
        tls_server_name: Optional TLS SNI override
        runtime: Optional Temporal runtime instance for telemetry/metrics
    """

    server_address: str
    namespace: str
    task_queue: str
    prom_listen_address: str | None = None
    auth_header: str | None = None
    tls: bool = False
    tls_server_name: str | None = None
    runtime: Any | None = None

    def connect_kwargs(self) -> dict[str, Any]:
        kwargs: dict[str, Any] = {"namespace": self.namespace}
        if self.auth_header:
            kwargs["rpc_metadata"] = {"authorization": self.auth_header}
        if self.tls_server_name:
            kwargs["tls"] = _tls_config_with_server_name(self.tls_server_name)
        elif self.tls:
            kwargs["tls"] = True
        if self.runtime is not None:
            kwargs["runtime"] = self.runtime
        return kwargs


def _tls_config_with_server_name(server_name: str):
    if TLSConfig is None:
        return True
    for key in ("server_name", "domain"):
        try:
            return TLSConfig(**{key: server_name})
        except TypeError:
            continue
    return True
