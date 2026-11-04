#!/usr/bin/env bash
# rig-model-health.sh — is every active rotation model actually banking rows?
#
# Why: from 2026-11-03 16:47 to 11-04 14:00 the rig ran with motoko silently
# absent. Its pre-flight canary failed on every eval-suite call (the v0.57.0
# effect checker rejected motoko's pre-#1958 test fixtures), eval-suite skipped
# it with a message in /tmp/ailang-os-filler.log, and the run carried on with the
# two other models. Nothing outside that log noticed: the coverage gate just read
# 0/54 forever and the weekly report only summarises rows that exist, so a model
# with none did not appear at all.
#
# A model is DARK for a version bank when it has ZERO valid ailang rows there
# while some other active model has at least --min-peer valid rows (so a fresh
# bank where nobody has run yet is not an alarm).
#
# Usage: tools/rig-model-health.sh --vdir DIR --models CSV [--min-peer N]
#   stdout: one "model valid=N invalid=M" line per active model, then
#           "DARK <model>" for each dark model.
#   exit 0 = all healthy, 3 = at least one DARK model, 2 = usage error,
#   1 = the check itself failed. This is an ALARM, so it does not fail open:
#   a broken check exits 1 and the caller says so.
set -uo pipefail

VDIR=""; MODELS=""; MIN_PEER=3
while [ $# -gt 0 ]; do
  case "$1" in
    --vdir) VDIR="$2"; shift 2;;
    --models) MODELS="$2"; shift 2;;
    --min-peer) MIN_PEER="$2"; shift 2;;
    *) echo "usage: $0 --vdir DIR --models CSV [--min-peer N]" >&2; exit 2;;
  esac
done
[ -n "$VDIR" ] && [ -n "$MODELS" ] || { echo "usage: --vdir and --models are required" >&2; exit 2; }
case "$MIN_PEER" in (''|*[!0-9]*) echo "usage: --min-peer must be a number" >&2; exit 2;; esac

python3 - "$VDIR" "$MODELS" "$MIN_PEER" <<'PYEOF'
import glob, json, os, sys
vdir, models, min_peer = sys.argv[1], [m for m in sys.argv[2].split(",") if m], int(sys.argv[3])
valid = {m: 0 for m in models}
invalid = {m: 0 for m in models}
for path in glob.glob(os.path.join(vdir, "*_ailang_*.json")):
    try:
        d = json.load(open(path))
    except Exception:
        continue  # an unreadable row is not a row (same rule as the coverage gate)
    m = d.get("model")
    if m not in valid or d.get("lang") not in (None, "ailang"):
        continue
    v = d.get("validity")
    if v is not None and not v.get("valid"):
        invalid[m] += 1
    else:
        valid[m] += 1
for m in models:
    print("%s valid=%d invalid=%d" % (m, valid[m], invalid[m]))
peer = max(valid.values()) if valid else 0
dark = [m for m in models if valid[m] == 0 and peer >= min_peer]
for m in dark:
    print("DARK %s" % m)
sys.exit(3 if dark else 0)
PYEOF
rc=$?
case "$rc" in 0|3) exit "$rc";; *) echo "rig-model-health: check failed (python rc=$rc)" >&2; exit 1;; esac
