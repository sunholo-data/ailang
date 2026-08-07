#!/usr/bin/env python3
"""Freeze live nightly trial directories into compact replay manifests.

Never regenerate a committed manifest after its live /tmp bank expires: these
files preserve the first-party evidence used to design the run-validity gate.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


FIELDS = ("compile_ok", "runtime_ok", "stdout_ok", "error_category", "duration_ms")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    args = parser.parse_args()
    for source in sorted(args.directory.glob("*.json")):
        data = json.loads(source.read_text(encoding="utf-8"))
        record = {"slot": source.name}
        record.update({field: data.get(field) for field in FIELDS})
        print(json.dumps(record, sort_keys=True, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
