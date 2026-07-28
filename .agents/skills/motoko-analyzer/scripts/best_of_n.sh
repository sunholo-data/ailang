#!/usr/bin/env bash
# Best-of-N analysis (probe #3): pass@1 vs best-of-N ceiling vs realistic typecheck+run-selector gain,
# from any trials=N rotation. Deterministic, read-only, no GPU. Wraps tools/eval_best_of_n.py.
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../../../.." && pwd)"; cd "$REPO" || exit 1
exec python3 tools/eval_best_of_n.py "$@"
