from __future__ import annotations

import argparse
import sys
from collections.abc import Sequence

from apps.helloworld.app import app as helloworld_app
from apps.worker.app import app as worker_app
from harness import App, run

DEFAULT_APP_NAME = "worker"

registry: dict[str, App] = {
    DEFAULT_APP_NAME: worker_app,
    "helloworld": helloworld_app,
}


def main(argv: Sequence[str] | None = None) -> None:
    parser = argparse.ArgumentParser(add_help=False, allow_abbrev=False)
    parser.add_argument("--app", default=DEFAULT_APP_NAME)
    args, remaining = parser.parse_known_args(sys.argv[1:] if argv is None else argv)

    app = registry.get(args.app)
    if app is None:
        raise SystemExit(f"unknown Python worker app {args.app!r}")

    run(app, remaining)


if __name__ == "__main__":
    main()
