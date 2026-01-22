"""Entry point for simple-test workflow load testing.

This file calls omes_starter.run() with the client and worker handlers.
It is invoked by omes with a subcommand:
    python main.py client --port 8080 --task-queue omes-xxx ...
    python main.py worker --task-queue omes-xxx ...
"""

from omes_starter import run
from src.client import client_main
from src.worker import worker_main

run(client=client_main, worker=worker_main)
