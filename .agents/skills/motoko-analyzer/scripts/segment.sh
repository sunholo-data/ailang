#!/usr/bin/env bash
# Gate 1 (OBSERVE) — failure-mode segmentation of the latest rotation. Deterministic,
# read-only, no GPU, no rig lock. Wraps tools/eval_failure_modes.py. Run this FIRST
# every cycle and paste the output before proposing any fix.
#
#   segment.sh [results_dir] [--by-benchmark] [--model-substr X] [--disengage-threshold N]
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$REPO" || exit 1
exec python3 tools/eval_failure_modes.py "$@"
