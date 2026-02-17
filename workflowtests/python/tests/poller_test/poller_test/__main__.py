"""Entry point for poller_test workflow load testing.

This file calls omes_starter.run() with the client and worker handlers.
It is invoked by omes with a subcommand:
    python -m simple_test client --port 8080 --task-queue omes-xxx ...
    python -m simple_test worker --task-queue omes-xxx ...
"""

from omes_starter import run
from .client import client_main
from .worker import worker_main

run(client=client_main, worker=worker_main)
