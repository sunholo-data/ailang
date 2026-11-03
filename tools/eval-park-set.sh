#!/usr/bin/env bash
# eval-park-set.sh — print the PARKED benchmark ids for the local-rig rotation,
# one id per line (the counterpart of eval-signal-set.sh's SATURATED set).
#
# "Parked" = no harness passes it. Across the last N version banks that hold rows
# for the active models, the benchmark has >= MIN_ROWS ailang rows, from >=
# MIN_MODELS distinct models, and ZERO passes. Such a benchmark separates nothing
# (every harness fails it), and on the rig it is also the most expensive cell: the
# 2026-10-20..11-03 window spent most of its signal-first GPU time on cp_60166
# (0/10) and cp_60159 (0/4), almost all timeouts. The filler therefore runs a
# parked benchmark as ONE probe trial on ONE model per release instead of
# models x 3 trials; the first passing probe row un-parks it automatically, because
# the rule is "zero passes".
#
# Usage:
#   tools/eval-park-set.sh --roll DIR --models CSV [--versions N] [--min-rows N] [--min-models N]
#
# Each parked id is explained on stderr (rows, models, failure categories) so the
# filler log says WHY a benchmark stopped getting full trials.
#
# FAIL-OPEN CONTRACT (same as eval-signal-set.sh): any data problem prints NOTHING
# and exits 0, so the rotation degrades to "nothing parked" (the old behaviour)
# instead of wedging. Only a usage error exits non-zero.
set -uo pipefail

ROLL=""; MODELS=""; NVER=3; MIN_ROWS=4; MIN_MODELS=2
while [ $# -gt 0 ]; do
  case "$1" in
    --roll) ROLL="$2"; shift 2;;
    --models) MODELS="$2"; shift 2;;
    --versions) NVER="$2"; shift 2;;
    --min-rows) MIN_ROWS="$2"; shift 2;;
    --min-models) MIN_MODELS="$2"; shift 2;;
    *) echo "usage: $0 --roll DIR --models CSV [--versions N] [--min-rows N] [--min-models N]" >&2; exit 2;;
  esac
done
[ -n "$ROLL" ] && [ -n "$MODELS" ] || { echo "usage: --roll and --models are required" >&2; exit 2; }
[ -d "$ROLL" ] || { echo "eval-park-set: no rotation dir $ROLL — nothing parked" >&2; exit 0; }
command -v python3 >/dev/null || { echo "eval-park-set: python3 missing — nothing parked" >&2; exit 0; }

python3 - "$ROLL" "$MODELS" "$NVER" "$MIN_ROWS" "$MIN_MODELS" <<'PYEOF' || true
import collections, glob, json, os, re, sys
roll, models, nver, min_rows, min_models = sys.argv[1], set(filter(None, sys.argv[2].split(","))), int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5])

def vkey(path):
    return tuple(int(x) for x in re.findall(r"\d+", os.path.basename(path)))

banks = []
for vdir in sorted(glob.glob(os.path.join(roll, "v[0-9]*")), key=vkey):
    rows = []
    for path in glob.glob(os.path.join(vdir, "agent", "*_ailang_*.json")):
        try:
            d = json.load(open(path))
        except Exception:
            continue
        if d.get("model") not in models or d.get("lang") != "ailang":
            continue
        v = d.get("validity")
        if v is not None and not v.get("valid"):
            continue  # a crash is not a measurement (same rule as the coverage gate)
        rows.append(d)
    if rows:
        banks.append((os.path.basename(vdir), rows))
recent = banks[-nver:]
agg = collections.defaultdict(lambda: {"n": 0, "pass": 0, "models": set(), "fail": collections.Counter()})
for _, rows in recent:
    for d in rows:
        a = agg[d.get("id")]
        a["n"] += 1
        a["models"].add(d["model"])
        if d.get("stdout_ok"):
            a["pass"] += 1
        else:
            a["fail"][d.get("error_category") or "unknown"] += 1
span = ",".join(v for v, _ in recent)
for bid in sorted(agg):
    a = agg[bid]
    if bid and a["pass"] == 0 and a["n"] >= min_rows and len(a["models"]) >= min_models:
        print(bid)
        print("parked %s: 0/%d passes across %d models in %s; failures %s"
              % (bid, a["n"], len(a["models"]), span, dict(a["fail"])), file=sys.stderr)
PYEOF
exit 0
