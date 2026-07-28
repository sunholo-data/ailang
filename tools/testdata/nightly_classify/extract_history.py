#!/usr/bin/env python3
"""Freeze the July 2026 nightly banks into the replay JSONL fixture.

The 2026-07-23 json_parse record is provenance-annotated because its /tmp files
were already reaped when this sprint ran. Never regenerate the committed replay
fixture after the live banks expire.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

TOOLS = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(TOOLS))

import nightly_classify as classifier  # noqa: E402


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("directories", nargs="+")
    args = parser.parse_args()
    records = []
    for directory in args.directories:
        records.extend(classifier.records_for_dir(directory))
    records.append(
        {
            "arm": "rag_on",
            "bench": "json_parse",
            "cats": ["compile_error"],
            "class": "",
            "date": "2026-07-23",
            "model": "opencode-qwen3-5-35b-a3b-mxfp8",
            "passes": 1,
            "provenance": (
                "design doc V9; source files reaped before sprint execution; "
                "no longer independently verifiable"
            ),
            "trials": 2,
        }
    )
    for record in sorted(records, key=classifier.record_key):
        print(json.dumps(record, sort_keys=True, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
