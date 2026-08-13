#!/usr/bin/env python3

import argparse
from pathlib import Path
import re


DEFAULT_PROPERTIES = Path(__file__).resolve().parents[2] / "workers/java/gradle.properties"
VERSION_PATTERN = re.compile(r"[0-9A-Za-z][0-9A-Za-z.+-]*")


def update_properties(path: Path, version: str) -> None:
    if not VERSION_PATTERN.fullmatch(version):
        raise SystemExit(f"invalid SDK version: {version!r}")

    contents = path.read_text(encoding="utf-8")
    updated, count = re.subn(
        r"^temporalSdkVersion=.*$",
        f"temporalSdkVersion={version}",
        contents,
        flags=re.MULTILINE,
    )
    if count != 1:
        raise SystemExit(f"expected one temporalSdkVersion property in {path}, found {count}")
    if updated != contents:
        path.write_text(updated, encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Update the Java SDK version property")
    parser.add_argument("version")
    parser.add_argument("--file", type=Path, default=DEFAULT_PROPERTIES)
    args = parser.parse_args()
    update_properties(args.file, args.version)


if __name__ == "__main__":
    main()
