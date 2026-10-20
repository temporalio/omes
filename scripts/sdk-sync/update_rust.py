#!/usr/bin/env python3

import argparse
from pathlib import Path
import re


DEFAULT_MANIFEST = Path(__file__).resolve().parents[2] / "workers/rust/Cargo.toml"
SDK_CRATES = ("client", "common", "macros", "sdk", "workflow")
VERSION_PATTERN = re.compile(r"[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?")


def update_manifest(path: Path, version: str) -> None:
    if not VERSION_PATTERN.fullmatch(version):
        raise SystemExit(f"invalid SDK version: {version!r}")

    contents = path.read_text(encoding="utf-8")
    updated = contents
    for crate in SDK_CRATES:
        # Keep table features/default-features and all surrounding formatting.
        # Refuse missing/path/workspace dependencies rather than guessing.
        pattern = (
            rf'^(temporalio-{crate}\s*=\s*(?:\{{[^\n}}]*?version\s*=\s*)?)'
            r'"=[^"]+"'
        )
        updated, count = re.subn(
            pattern, rf'\g<1>"={version}"', updated, flags=re.MULTILINE
        )
        if count != 1:
            raise SystemExit(
                f"expected one pinned temporalio-{crate} dependency in {path}, found {count}"
            )
    if updated != contents:
        path.write_text(updated, encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Update the Rust SDK development pins")
    parser.add_argument("version")
    parser.add_argument("--file", type=Path, default=DEFAULT_MANIFEST)
    args = parser.parse_args()
    update_manifest(args.file, args.version)


if __name__ == "__main__":
    main()
