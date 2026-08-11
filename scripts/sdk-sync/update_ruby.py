#!/usr/bin/env python3

import argparse
from pathlib import Path
import re


DEFAULT_GEMFILE = Path(__file__).resolve().parents[2] / "workers/ruby/Gemfile"
VERSION_PATTERN = re.compile(r"[0-9A-Za-z][0-9A-Za-z.+-]*")


def update_gemfile(path: Path, version: str) -> None:
    if not VERSION_PATTERN.fullmatch(version):
        raise SystemExit(f"invalid SDK version: {version!r}")

    contents = path.read_text(encoding="utf-8")
    dependency = f"gem 'temporalio', '= {version}'"
    updated, count = re.subn(
        r"^gem\s+(['\"])temporalio\1(?:\s*,.*)?$",
        dependency,
        contents,
        flags=re.MULTILINE,
    )
    if count != 1:
        raise SystemExit(f"expected one top-level temporalio dependency in {path}, found {count}")
    if updated != contents:
        path.write_text(updated, encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Update the Ruby SDK development pin")
    parser.add_argument("version")
    parser.add_argument("--file", type=Path, default=DEFAULT_GEMFILE)
    args = parser.parse_args()
    update_gemfile(args.file, args.version)


if __name__ == "__main__":
    main()
