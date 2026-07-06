from __future__ import annotations

import logging
import os
from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from temporalio.client import Client
from temporalio.runtime import (
    LoggingConfig,
    PrometheusConfig,
    Runtime,
    TelemetryConfig,
    TelemetryFilter,
)
from temporalio.service import TLSConfig


def _build_tls_config(
    *,
    tls: bool,
    tls_cert_path: str,
    tls_key_path: str,
    tls_server_name: str | None = None,
    disable_host_verification: bool = False,
) -> TLSConfig | None:
    if disable_host_verification:
        logging.getLogger(__name__).warning(
            "disable_host_verification is not supported by the Python SDK; ignoring"
        )
    if tls_cert_path:
        if not tls_key_path:
            raise ValueError("Client cert specified, but not client key!")
        with open(tls_cert_path, "rb") as cert_file:
            client_cert = cert_file.read()
        with open(tls_key_path, "rb") as key_file:
            client_key = key_file.read()
        return TLSConfig(
            client_cert=client_cert,
            client_private_key=client_key,
            domain=tls_server_name,
        )
    if tls_key_path:
        raise ValueError("Client key specified, but not client cert!")
    if tls:
        return TLSConfig(domain=tls_server_name)
    return None


def _build_api_key(auth_header: str) -> str | None:
    return auth_header.removeprefix("Bearer ") if auth_header else None


def _build_runtime(prom_listen_address: str | None) -> Runtime:
    prometheus = (
        PrometheusConfig(bind_address=prom_listen_address, durations_as_seconds=True)
        if prom_listen_address
        else None
    )
    return Runtime(
        telemetry=TelemetryConfig(
            metrics=prometheus,
            logging=LoggingConfig(
                filter=TelemetryFilter(
                    core_level=os.getenv("TEMPORAL_CORE_LOG_LEVEL", "INFO"),
                    other_level="WARN",
                )
            ),
        ),
    )


@dataclass(frozen=True)
class ClientConfig:
    target_host: str
    namespace: str
    api_key: str | None
    tls: TLSConfig | None
    runtime: Runtime


ClientFactory = Callable[[ClientConfig], Awaitable[Client]]


async def default_client_factory(config: ClientConfig) -> Client:
    return await Client.connect(
        target_host=config.target_host,
        namespace=config.namespace,
        api_key=config.api_key,
        tls=config.tls,
        runtime=config.runtime,
    )


def build_client_config(
    *,
    server_address: str,
    namespace: str,
    auth_header: str,
    tls: bool,
    tls_cert_path: str,
    tls_key_path: str,
    tls_server_name: str | None = None,
    disable_host_verification: bool = False,
    prom_listen_address: str | None = None,
) -> ClientConfig:
    return ClientConfig(
        target_host=server_address,
        namespace=namespace,
        api_key=_build_api_key(auth_header),
        tls=_build_tls_config(
            tls=tls,
            tls_cert_path=tls_cert_path,
            tls_key_path=tls_key_path,
            tls_server_name=tls_server_name,
            disable_host_verification=disable_host_verification,
        ),
        runtime=_build_runtime(prom_listen_address),
    )
