#!/usr/bin/env python3
"""Cold-cache recursive example/std audit for a provenance-pinned AILANG binary.

Usage: python3 tools/audit-effect-rows.py SOURCE_ROOT BINARY OUTPUT_JSON
Run once per matching source tree; retain both reports when comparing status flips.
Unlike audit-examples.sh, this includes hidden/broken fixtures and relaxed std checks.
"""
import argparse
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", type=Path)
    parser.add_argument("binary", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    root, binary = args.root.resolve(), args.binary.resolve()
    paths = sorted(subprocess.check_output(
        ["rg", "--files", "--hidden", "--no-ignore", "examples", "std", "-g", "*.ail"],
        cwd=root, text=True).splitlines())
    env = dict(os.environ, AILANG_NO_CACHE="1", AILANG_STDLIB_PATH=str(root / "std"))

    def check(item):
        path, relaxed = item
        command = [str(binary), "check", "--timeout", "30s"]
        if relaxed:
            command.append("--relax-modules")
        command.append(path)
        record = dict(path=path, relaxed=relaxed, command=command)
        try:
            result = subprocess.run(command, cwd=root, env=env, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=40)
            record.update(exit=result.returncode, stdout=result.stdout, stderr=result.stderr)
        except subprocess.TimeoutExpired:
            record.update(exit=124, category="timeout", stdout="", stderr="external timeout")
        return record

    items = [(path, False) for path in paths]
    items += [(path, True) for path in paths if path.startswith("std/")]
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        records = list(pool.map(check, items))
    with binary.open("rb") as source:
        digest = hashlib.sha256(source.read()).hexdigest()
    report = dict(root=str(root), binary=str(binary), binary_sha256=digest,
                  inventory=paths, records=records)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(f"{len(paths)} files; {len(records)} checks; "
          f"{sum(record['exit'] == 0 for record in records)} passes")
    return int(any(record["exit"] in (124, -9, -11) for record in records))


if __name__ == "__main__":
    raise SystemExit(main())
