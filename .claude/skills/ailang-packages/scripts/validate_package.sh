#!/usr/bin/env bash
# Resolve, compile, execute native declarations and report authoring evidence.
set -euo pipefail
cd "${1:-.}"
AILANG_BIN="${AILANG_BIN:-ailang}"
"$AILANG_BIN" lock
"$AILANG_BIN" check --package .
"$AILANG_BIN" test --package .
report=$(mktemp)
trap 'rm -f "$report"' EXIT
# Fails explicitly on old binaries; never silently downgrade to compilation only.
"$AILANG_BIN" pkg quality --json . > "$report"
# Run inline source evidence too; package test files were already executed above.
python3 - "$report" "$AILANG_BIN" <<'PYTHON'
import json, subprocess, sys
with open(sys.argv[1]) as stream:
    report = json.load(stream)
for source in report["test_sources"]:
    if not source.endswith("_test.ail"):
        subprocess.run([sys.argv[2], "test", source], check=True)
PYTHON
"$AILANG_BIN" pkg quality --strict .
printf '%s\n' 'Compilation/tests and declaration inventory completed. Review test totals/skips; proof and publication are separate.'
