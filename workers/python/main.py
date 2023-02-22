import argparse
import asyncio
import logging
import sys
import threading
from urllib.parse import urlparse
from wsgiref.simple_server import make_server

from prometheus_client import make_wsgi_app
from pythonjsonlogger import jsonlogger
from temporalio.client import Client, TLSConfig
from temporalio.runtime import PrometheusConfig, Runtime, TelemetryConfig
from temporalio.worker import Worker

from .activities import noop_activity
from .kitchen_sink import KitchenSinkWorkflow

nameToLevel = {
    "PANIC": logging.FATAL,
    "FATAL": logging.FATAL,
    "ERROR": logging.ERROR,
    "WARN": logging.WARNING,
    "INFO": logging.INFO,
    "DEBUG": logging.DEBUG,
    "NOTSET": logging.NOTSET,
}

interrupt_event = asyncio.Event()


async def run():
    # Parse args
    parser = argparse.ArgumentParser()
    parser.add_argument("-q", "--task-queue", default="omes", help="task queue to use")
    # Log arguments
    parser.add_argument(
        "--log-level", default="info", help="(debug info warn error panic fatal)"
    )
    parser.add_argument("--log-encoding", default="console", help="(console json)")
    # Client arguments
    parser.add_argument(
        "-n", "--namespace", default="default", help="Namespace to connect to"
    )
    parser.add_argument(
        "-a",
        "--server-address",
        default="localhost:7233",
        help="Address of Temporal server",
    )
    parser.add_argument(
        "--tls-cert-path", default="", help="Path to client TLS certificate"
    )
    parser.add_argument("--tls-key-path", default="", help="Path to client private key")
    # Prometheus metric arguments
    parser.add_argument("--prom-listen-address", help="Prometheus listen address")
    parser.add_argument(
        "--prom-handler-path", default="/metrics", help="Prometheus handler path"
    )
    args = parser.parse_args()

    # Configure TLS
    tls_config = None
    if args.tls_cert_path:
        if not args.tls_key_path:
            raise ValueError("Client cert specified, but not client key!")
        with open(args.client_cert_path, "rb") as f:
            client_cert = f.read()
        with open(args.client_key_path, "rb") as f:
            client_key = f.read()
        tls_config = TLSConfig(client_cert=client_cert, client_private_key=client_key)
    elif args.tls_key_path and not args.client_cert_path:
        raise ValueError("Client key specified, but not client cert!")

    # Configure logging
    logger = logging.getLogger()
    logHandler = logging.StreamHandler(stream=sys.stderr)
    if args.log_encoding == "json":
        format_str = "%(message)%(levelname)%(name)%(asctime)"
        formatter = jsonlogger.JsonFormatter(format_str)
        logHandler.setFormatter(formatter)
    logger.addHandler(logHandler)
    logger.setLevel(nameToLevel[args.log_level.upper()])

    # Configure metrics
    prometheus = None
    if args.prom_listen_address:
        prom_addr = urlparse(args.prom_listen_address)
        metrics_app = make_wsgi_app()
        handle_path = args.prom_handler_path

        def prom_app(environ, start_fn):
            if environ["PATH_INFO"] == handle_path:
                return metrics_app(environ, start_fn)

        httpd = make_server(prom_addr.hostname, prom_addr.port, prom_app)
        t = threading.Thread(target=httpd.serve_forever)
        t.daemon = True
        t.start()
        prometheus = PrometheusConfig(bind_address=parser.prom_listen_address)

    new_runtime = Runtime(telemetry=TelemetryConfig(metrics=prometheus))
    client = await Client.connect(
        target_host=args.server_address,
        namespace=args.namespace,
        tls=tls_config,
        runtime=new_runtime,
    )

    # Run a worker for the workflow
    async with Worker(
        client,
        task_queue=args.task_queue,
        workflows=[KitchenSinkWorkflow],
        activities=[noop_activity],
    ):
        # Wait until interrupted
        await interrupt_event.wait()


if __name__ == "__main__":
    loop = asyncio.new_event_loop()
    try:
        loop.run_until_complete(run())
    except KeyboardInterrupt:
        interrupt_event.set()
        loop.run_until_complete(loop.shutdown_asyncgens())
