#!/usr/bin/env bash
# Compare the extracted legacy mode with the byte-for-byte pre-change heredoc.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
PREVIOUS=""
for DAY in 4 5 6 7 8; do
    TONIGHT="/tmp/nightly_eval_2026072${DAY}_rag_on/agent"
    [[ -d "$TONIGHT" ]] || {
        echo "missing live bank: $TONIGHT" >&2
        exit 1
    }
    LEGACY=$(mktemp)
    EXTRACTED=$(mktemp)
    RESULTS_AGENT="$TONIGHT" PREV_RESULTS="$PREVIOUS" python3 - <<'PY' >"$LEGACY"
import json, glob, os, re
from collections import defaultdict
INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}
results_agent = os.environ.get("RESULTS_AGENT", "")
prev_results = os.environ.get("PREV_RESULTS", "")
latest = {}
for f in glob.glob(f"{results_agent}/*.json"):
    base = os.path.basename(f)
    try: ts = int(base.rsplit("_", 1)[1][:-5])
    except (IndexError, ValueError): continue
    slot = base.rsplit("_", 1)[0]
    if slot not in latest or ts > latest[slot][0]:
        latest[slot] = (ts, f)
trials = defaultdict(list)
for slot, (_, f) in latest.items():
    try:
        with open(f) as handle: d = json.load(handle)
    except Exception: continue
    bench = re.sub(r"_trial\d+$", "", slot.split("_ailang_")[0])
    ok = d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")
    trials[bench].append((ok, d.get("error_category") or "—"))
persistent = []
for bench, results in trials.items():
    if len(results) >= 2 and all(not ok for ok, _ in results):
        cats = {c for _, c in results}
        if cats - INFRA_CATEGORIES:
            persistent.append((bench, sorted(cats)))
def was_solid_in_prev(bench):
    if not prev_results or not os.path.isdir(prev_results):
        return False
    seen = []
    for f in glob.glob(f"{prev_results}/{bench}_*.json"):
        b2 = re.sub(r"_trial\d+$", "", os.path.basename(f).split("_ailang_")[0])
        if b2 != bench: continue
        try:
            with open(f) as handle: d = json.load(handle)
        except Exception: continue
        seen.append(bool(d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")))
    return len(seen) >= 1 and all(seen)
for bench, cats in sorted(persistent):
    cls = "REGRESSION" if was_solid_in_prev(bench) else "GAP"
    print(f"{cls}\t{bench}\t[{','.join(cats)}]")
PY
    python3 "$ROOT/tools/nightly_classify.py" --legacy \
        --tonight "$TONIGHT" ${PREVIOUS:+--previous "$PREVIOUS"} >"$EXTRACTED"
    diff -u "$LEGACY" "$EXTRACTED"
    rm -f "$LEGACY" "$EXTRACTED"
    echo "equivalent: 2026-07-2${DAY}"
    PREVIOUS="$TONIGHT"
done
