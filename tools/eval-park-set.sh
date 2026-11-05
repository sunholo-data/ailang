#!/usr/bin/env bash
# eval-park-set.sh — print the PARKED benchmark ids for the local-rig rotation,
# one id per line (the counterpart of eval-signal-set.sh's SATURATED set).
#
# "Parked" = no harness passes it. Across the last N version banks that RAN THIS
# BENCHMARK on the active models, the benchmark has >= MIN_ROWS ailang rows, from >=
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
#   tools/eval-park-set.sh --mode saturated --roll DIR --models CSV [--versions N] [--min-per-model N]
#
# --mode saturated prints the LOCALLY SATURATED set instead: benchmarks where every
# active model has >= --min-per-model valid rows in the benchmark's window and all
# of them passed. The filler drops these (OS_FILLER_SIGNAL_ONLY, 2026-11-05): a
# benchmark every local harness already passes teaches the roadmap nothing. The
# global ELO "Trivial" band (eval-signal-set.sh) is NOT used for this, because it
# blends cloud models: trivial for a frontier model can still fail on the rig.
#
# Each parked id is explained on stderr (rows, models, failure categories) so the
# filler log says WHY a benchmark stopped getting full trials.
#
# FAIL-OPEN CONTRACT (same as eval-signal-set.sh): any data problem prints NOTHING
# and exits 0, so the rotation degrades to "nothing parked" (the old behaviour)
# instead of wedging. Only a usage error exits non-zero.
set -uo pipefail

ROLL=""; MODELS=""; NVER=3; MIN_ROWS=4; MIN_MODELS=2; MODE=park; MIN_PER_MODEL=3
while [ $# -gt 0 ]; do
  case "$1" in
    --roll) ROLL="$2"; shift 2;;
    --models) MODELS="$2"; shift 2;;
    --versions) NVER="$2"; shift 2;;
    --min-rows) MIN_ROWS="$2"; shift 2;;
    --min-models) MIN_MODELS="$2"; shift 2;;
    --mode) MODE="$2"; shift 2;;
    --min-per-model) MIN_PER_MODEL="$2"; shift 2;;
    *) echo "usage: $0 --roll DIR --models CSV [--mode park|saturated] [--versions N] [--min-rows N] [--min-models N] [--min-per-model N]" >&2; exit 2;;
  esac
done
case "$MODE" in park|saturated) ;; *) echo "usage: --mode must be park or saturated" >&2; exit 2;; esac
[ -n "$ROLL" ] && [ -n "$MODELS" ] || { echo "usage: --roll and --models are required" >&2; exit 2; }
[ -d "$ROLL" ] || { echo "eval-park-set: no rotation dir $ROLL — nothing parked" >&2; exit 0; }
command -v python3 >/dev/null || { echo "eval-park-set: python3 missing — nothing parked" >&2; exit 0; }

python3 - "$ROLL" "$MODELS" "$NVER" "$MIN_ROWS" "$MIN_MODELS" "$MODE" "$MIN_PER_MODEL" <<'PYEOF' || true
import collections, glob, json, os, re, sys
roll, models, nver, min_rows, min_models = sys.argv[1], set(filter(None, sys.argv[2].split(","))), int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5])
mode, min_per_model = sys.argv[6], int(sys.argv[7])

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
# The window is per BENCHMARK: its last N banks that actually ran it. A global
# "last N banks" window un-parked cp_60166 on 2026-11-04 with no new evidence:
# two quick releases (v0.57.0, v0.57.1) that had not reached it yet pushed its
# old rows out, leaving 3 < min_rows, so missing data read as "might pass now".
by_bench = collections.defaultdict(list)          # id -> [(ver, [rows])] in version order
for ver, rows in banks:
    per = collections.defaultdict(list)
    for d in rows:
        per[d.get("id")].append(d)
    for bid, brows in per.items():
        by_bench[bid].append((ver, brows))
agg = collections.defaultdict(lambda: {"n": 0, "pass": 0, "models": set(), "per": collections.Counter(), "fail": collections.Counter(), "vers": []})
for bid, vrows in by_bench.items():
    a = agg[bid]
    for ver, brows in vrows[-nver:]:
        a["vers"].append(ver)
        for d in brows:
            a["n"] += 1
            a["models"].add(d["model"])
            a["per"][d["model"]] += 1
            if d.get("stdout_ok"):
                a["pass"] += 1
            else:
                a["fail"][d.get("error_category") or "unknown"] += 1
for bid in sorted(agg):
    a = agg[bid]
    if not bid:
        continue
    if mode == "park":
        if a["pass"] == 0 and a["n"] >= min_rows and len(a["models"]) >= min_models:
            print(bid)
            print("parked %s: 0/%d passes across %d models in %s; failures %s"
                  % (bid, a["n"], len(a["models"]), ",".join(a["vers"]), dict(a["fail"])), file=sys.stderr)
    else:
        # LOCALLY saturated: EVERY active model has >= min_per_model valid rows in
        # the benchmark's own window and every one passed. A model with no rows
        # there means "not measured", never "saturated".
        if a["pass"] == a["n"] and all(a["per"][m] >= min_per_model for m in models):
            print(bid)
            print("saturated %s: %d/%d passes, every model >= %d rows, in %s"
                  % (bid, a["pass"], a["n"], min_per_model, ",".join(a["vers"])), file=sys.stderr)
PYEOF
exit 0
