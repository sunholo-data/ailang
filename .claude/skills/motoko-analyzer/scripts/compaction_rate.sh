#!/usr/bin/env bash
# Compaction-fire-rate report — the convergence-thrash leading indicator (M-AILANG-SEMANTIC-CONTEXT).
# Deterministic, read-only, no GPU, no rig lock. Companion to segment.sh. Wraps
# tools/eval_compaction_rate.py. fire_rate ~0 = pi-level context hygiene (the A/B target).
#
#   compaction_rate.sh [results_dir] [--by-benchmark] [--model-substr X] [--lang ailang]
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$REPO" || exit 1
exec python3 tools/eval_compaction_rate.py "$@"
