#!/usr/bin/env bash
# nightly-eval.sh — nightly regression-guard smoke run on the local rig.
#
# Runs the smoke + core tiers (every `tier: smoke` / `tier: core` benchmark,
# default-core included) against the accuracy-first local Qwen model
# (opencode-qwen3-5-35b-a3b-mxfp8). Regressions — benchmarks with a passing
# baseline that now fail every trial — alert via Discord; never-passed
# benchmarks are filed to the controlplane inbox as known gaps for the
# gap-finder (no Discord). See eval_baselines gating below.
#
# Reproducibility (M-EVAL-NIGHTLY-REPRO): the eval is built and run from an
# isolated git worktree pinned to committed origin/dev — never the live working
# tree or a stray installed binary — so the binary, benchmarks, and prompt card
# all come from one named commit. The run aborts loudly if it can't produce a
# clean build from committed code.
#
# Called by dev.ailang.nightly-eval.plist at 03:00 daily.
#
# Installation:
#   cp tools/launchd/dev.ailang.nightly-eval.plist ~/Library/LaunchAgents/
#   launchctl load ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist
#
# Manual one-shot:
#   AILANG_NIGHTLY_EVAL_DRY_RUN=1 tools/launchd/nightly-eval.sh

set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO"

LOG=/tmp/ailang-nightly-eval.log
RESULTS_DIR="/tmp/nightly_eval_$(date +%Y%m%d)"
DATE=$(date +%Y-%m-%d)

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG"; }

log "=== nightly eval started (${DATE}) ==="

# Ensure ollama is running (rig-watchdog usually handles this, but be safe)
if ! curl -s --max-time 3 http://localhost:11434/api/version >/dev/null 2>&1; then
    log "ollama not reachable — skipping eval (will retry tomorrow)"
    ailang messages send controlplane "Nightly eval skipped: ollama not reachable at 03:00. Check rig-watchdog." \
        --title "Nightly eval skipped (${DATE})" --from "nightly-eval" 2>/dev/null || true
    exit 0
fi

# ─────────────────────────────────────────────────────────────────────────────
# Reproducible build (M-EVAL-NIGHTLY-REPRO): run from an ISOLATED checkout pinned
# to the committed dev HEAD — never the live working tree or a stray installed
# binary. This guarantees the binary, benchmarks, AND prompts/agent/dialect-traps
# card all come from one known commit, and that we can name that commit in every
# result. Fail LOUDLY (skip with an alert) rather than silently evaluate
# uncommitted or stale code.
BUILD_REF="${AILANG_NIGHTLY_REF:-origin/dev}"
WT="$HOME/.ailang-nightly/worktree"

fail() {  # $*: reason — alert controlplane, then abort (no eval runs)
    log "FATAL: $*"
    ailang messages send controlplane \
        "Nightly eval ABORTED (${DATE}): $*. No eval ran — code provenance could not be guaranteed." \
        --title "Nightly eval aborted (${DATE})" --from "nightly-eval" 2>/dev/null || true
    exit 1
}

log "syncing build worktree to ${BUILD_REF}"
git -C "$REPO" fetch --quiet origin || fail "git fetch origin failed"
TARGET=$(git -C "$REPO" rev-parse "$BUILD_REF") || fail "cannot resolve ${BUILD_REF}"
SHORT=$(git -C "$REPO" rev-parse --short "$TARGET")

# Refresh (or create) the dedicated build worktree at the pinned commit. The
# worktree is a throwaway build dir with no user work, separate from $REPO, so
# this never touches in-progress dev changes. --force overwrites only tracked
# files; gitignored build output (bin/) persists and is rebuilt below.
git -C "$REPO" worktree prune
if git -C "$REPO" worktree list --porcelain | grep -qx "worktree $WT"; then
    git -C "$WT" checkout --quiet --detach --force "$TARGET" || fail "worktree checkout ${SHORT} failed"
else
    mkdir -p "$(dirname "$WT")"
    git -C "$REPO" worktree add --quiet --detach --force "$WT" "$TARGET" || fail "worktree add ${SHORT} failed"
fi

log "building ailang @ ${SHORT} in worktree"
make -C "$WT" build >>"$LOG" 2>&1 || fail "make build @ ${SHORT} failed"
BIN="$WT/bin/ailang"
[[ -x "$BIN" ]] || fail "built binary missing at $BIN"

# Run from the worktree so benchmarks/, the dialect-traps card, and FindAILANG's
# './bin/ailang' fallback all resolve to the pinned commit; put the pinned binary
# first on PATH so FindAILANG's bare 'ailang' lookup hits it too (not ~/go/bin).
cd "$WT"
export PATH="$WT/bin:$PATH"

BUILD_VERSION=$("$BIN" --version 2>/dev/null | head -1)
case "$BUILD_VERSION" in
    *-dirty|"") fail "built binary not clean (${BUILD_VERSION:-<none>}) — worktree dirty at ${SHORT}" ;;
esac
log "build OK: ${BUILD_VERSION} (commit ${SHORT})"

# Benchmark scope: smoke + core tiers (M-EVAL-LOCAL-OLLAMA tier expansion).
# smoke = the regression guard (model reliably passes); core = the capability
# frontier (harder; the local model fails some it has never passed).
#
# Agent mode REQUIRES an explicit --benchmarks list (it refuses to auto-discover),
# so we derive the union of smoke + explicit-core + default-core by filename stem.
# A "benchmark" is a .yml with an `id:` field; default-core = has id: but no
# `tier:` field (the loader defaults missing tier to "core"). The id: guard
# excludes non-benchmark meta-files such as events.yml (the dashboard
# suite-change log) — which the Go discoverBenchmarks/isBenchmarkMetaFile also
# skip. Without it the eval-suite tries to "run" events.yml, LoadSpec rejects it
# ("missing required field: id"), and it produces zero results — a phantom gap
# that wasted two trial slots in the first smoke+core run.
MODEL="opencode-qwen3-5-35b-a3b-mxfp8"
BENCH_TIERS="smoke,core"   # display label for alerts/log
# Thrash ceiling per benchmark (M-EVAL-OS-LONGITUDINAL). When eval_baselines has
# >=5 passing samples for a (model, benchmark), the eval-suite uses an adaptive
# mean+2σ token threshold; otherwise this fixed value is the hard ceiling. Set
# ABOVE the heaviest legitimate passing trial observed (~3.66M tokens) so a
# slow-but-correct run is never converted into a false thrash-abort, while still
# bounding runaways: graph_bfs hit 6.7M–8.3M tokens on a single trial with no cap,
# adding ~an hour to the night. A 1M cap was A/B-validated 2026-06-11 (bounded an
# 8.3M-token runaway); 4M is the looser nightly ceiling that protects legit passes.
MAX_TOKENS_PER_BENCH=4000000

BENCH_LIST=$( {
    grep -lE '^[[:space:]]*tier:[[:space:]]*smoke' benchmarks/*.yml
    grep -lE '^[[:space:]]*tier:[[:space:]]*core'  benchmarks/*.yml
    # default-core: a real benchmark (has id:) with no tier: field
    comm -12 <(grep -lE '^[[:space:]]*id:' benchmarks/*.yml | sort) \
             <(grep -LE '^[[:space:]]*tier:[[:space:]]*[a-z]' benchmarks/*.yml | sort)
  } 2>/dev/null | xargs -n1 basename | sed 's/\.yml$//' | sort -u | paste -sd, - )
TIER_COUNT=$(echo "$BENCH_LIST" | tr ',' '\n' | grep -c .)
if [[ -z "$BENCH_LIST" || "$TIER_COUNT" -eq 0 ]]; then
    log "ERROR: could not derive smoke+core benchmark set from benchmarks/*.yml"
    exit 1
fi

log "tiers: ${BENCH_TIERS} (${TIER_COUNT} benchmarks)"
log "output: $RESULTS_DIR"

# Dry-run mode for testing the plist without spending GPU time
if [[ "${AILANG_NIGHTLY_EVAL_DRY_RUN:-0}" == "1" ]]; then
    log "DRY RUN — exiting without running eval"
    exit 0
fi

# Run the eval in A/B mode: microRAG ON then OFF, same smoke set.
# Results land in separate subdirs so analysis can diff them.
# A/B is weekly (Mondays) to avoid doubling the nightly wall-clock every day.
# On non-Monday nights: microRAG=on only (regression guard, no comparison overhead).
DAY_OF_WEEK=$(date +%u)  # 1=Mon … 7=Sun
RUN_AB=0
[[ "${AILANG_FORCE_AB:-0}" == "1" ]] && RUN_AB=1
[[ "$DAY_OF_WEEK" == "1" ]] && RUN_AB=1

run_eval() {
    local mode="$1" outdir="$2"
    log "running smoke: microrag=${mode} → ${outdir}"
    "$BIN" eval-suite --agent \
        --models "$MODEL" \
        --benchmarks "$BENCH_LIST" \
        --langs ailang \
        --microrag "$mode" \
        --output "$outdir" \
        --parallel 1 \
        --trials 2 \
        --max-tokens-per-bench "$MAX_TOKENS_PER_BENCH" >> "$LOG" 2>&1
}

run_eval "on"  "${RESULTS_DIR}_rag_on"

if [[ "$RUN_AB" == "1" ]]; then
    log "Monday A/B run: also running microrag=off"
    run_eval "off" "${RESULTS_DIR}_rag_off"

    # Compare
    PASS_ON=$(grep -oE '[0-9]+/[0-9]+ passed' "${RESULTS_DIR}_rag_on"/../*.log 2>/dev/null | tail -1 | cut -d/ -f1 || \
              python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_DIR}_rag_on/agent/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); print(p)" 2>/dev/null || echo "?")
    PASS_OFF=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_DIR}_rag_off/agent/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); print(p)" 2>/dev/null || echo "?")
    log "A/B result: microrag_on=${PASS_ON} microrag_off=${PASS_OFF}"
    if [[ "${PASS_ON:-x}" =~ ^[0-9]+$ && "${PASS_OFF:-x}" =~ ^[0-9]+$ ]]; then
        DELTA=$(( PASS_ON - PASS_OFF ))
    else
        DELTA="?"
    fi
    ailang messages send controlplane \
        "Weekly μRAG A/B (${DATE}): on=${PASS_ON}/${TIER_COUNT}  off=${PASS_OFF}/${TIER_COUNT}. Delta=${DELTA} benchmarks. Results: ${RESULTS_DIR}_rag_on vs ${RESULTS_DIR}_rag_off" \
        --title "μRAG A/B result (${DATE})" --from "nightly-eval" 2>/dev/null || true
fi

# Use the rag_on results for regression detection (canonical arm)
RESULTS_AGENT="${RESULTS_DIR}_rag_on/agent"
export RESULTS_AGENT
PASS=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); print(f'{p}/{len(r)}')" 2>/dev/null || echo "?/?")
RATE=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); n=len(r); print(f'{100*p//n if n else 0}%')" 2>/dev/null || echo "?%")

log "regression check result: ${PASS} (${RATE})"

# Find the most recent PRIOR nightly run (rag_on arm) for the regression
# comparison. Glob results are lexically sorted and the date (YYYYMMDD) is in the
# path, so ascending order = chronological — the LAST non-today match is the most
# recent prior run.
PREV_RESULTS=""
for d in /tmp/nightly_eval_*_rag_on/agent; do
    [[ -d "$d" ]] || continue                 # no matches → literal pattern, skip
    [[ "$d" == "$RESULTS_AGENT" ]] && continue # skip today's own run
    PREV_RESULTS="$d"
done
if [[ -n "$PREV_RESULTS" ]]; then
    log "regression baseline: previous run = $PREV_RESULTS"
else
    log "regression baseline: no prior run found — flaky failures will NOT alert tonight"
fi

# Classify *persistent* failures (a benchmark whose EVERY trial failed tonight,
# with >=1 non-infra error category) into REGRESSIONS vs non-alerting GAPS by
# comparing against the PREVIOUS nightly run:
#   - REGRESSION: the benchmark passed ALL its trials in the previous run but
#     fails every trial tonight — a genuine, fresh solid->broken break → Discord.
#   - GAP: it was NOT solid last night (flaky, already-failing, or never-passed)
#     → filed quietly to controlplane, NO Discord. This stops flaky benchmarks
#     (which fail both trials by chance every few nights) from paging — 5 such
#     false alerts fired on 2026-06-12 under the old "ever passed (has baseline)"
#     gate. A benchmark that passed >=1 trial TONIGHT is flaky-but-recovered and
#     is not flagged at all.
#
# Result files are named  <bench>[_trialN]_<lang>_<model>_<ts>.json (trial 1 has
# no _trialN infix). Both trials must collapse to the same benchmark id, else
# each trial looks like its own single-trial benchmark.
CLASSIFIED=$(RESULTS_AGENT="$RESULTS_AGENT" PREV_RESULTS="$PREV_RESULTS" python3 - <<'PY'
import json, glob, os, re
from collections import defaultdict

# Infra/transport noise (executor hung, API error, timeout) — not a
# language/prompt/stdlib regression. Surfaces in the pass count + log; never
# escalated as a bug or gap.
INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}

results_agent = os.environ.get("RESULTS_AGENT", "")
prev_results  = os.environ.get("PREV_RESULTS", "")

# Keep the newest file per (benchmark, trial-slot) so re-runs dedupe while BOTH
# trials are retained. The trial slot is everything before "_<lang>_<model>_<ts>".
latest = {}
for f in glob.glob(f"{results_agent}/*.json"):
    base = os.path.basename(f)
    try: ts = int(base.rsplit("_", 1)[1][:-5])
    except (IndexError, ValueError): continue
    slot = base.rsplit("_", 1)[0]
    if slot not in latest or ts > latest[slot][0]:
        latest[slot] = (ts, f)

# Group trials under their benchmark id by stripping the _trialN infix.
trials = defaultdict(list)
for slot,(ts,f) in latest.items():
    try: d = json.load(open(f))
    except: continue
    bench = re.sub(r"_trial\d+$", "", slot.split("_ailang_")[0])
    ok = d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")
    trials[bench].append((ok, d.get("error_category") or "—"))

# Persistent failure = >=2 trials seen, every trial failed, >=1 real category.
persistent = []
for bench, results in trials.items():
    if len(results) >= 2 and all(not ok for ok,_ in results):
        cats = {c for _,c in results}
        if cats - INFRA_CATEGORIES:
            persistent.append((bench, sorted(cats)))

# A benchmark "was solid" if it passed ALL its trials (>=1 seen) in the previous
# nightly run. Only a solid->broken transition is a genuine regression worth a
# Discord ping. Missing prior run (e.g. /tmp cleared, first night) fails SAFE to
# GAP (no alert) — better to miss one alert than to cry wolf on flaky noise.
def was_solid_in_prev(bench):
    if not prev_results or not os.path.isdir(prev_results):
        return False
    seen = []
    for f in glob.glob(f"{prev_results}/{bench}_*.json"):
        b2 = re.sub(r"_trial\d+$", "", os.path.basename(f).split("_ailang_")[0])
        if b2 != bench:
            continue
        try: d = json.load(open(f))
        except: continue
        seen.append(bool(d.get("compile_ok") and d.get("runtime_ok") and d.get("stdout_ok")))
    return len(seen) >= 1 and all(seen)

for bench, cats in sorted(persistent):
    cls = "REGRESSION" if was_solid_in_prev(bench) else "GAP"
    print(f"{cls}\t{bench}\t[{','.join(cats)}]")
PY
) || true

REGRESSIONS=$(echo "$CLASSIFIED" | awk -F'\t' '$1=="REGRESSION"{print $2" "$3}' | sed '/^[[:space:]]*$/d')
GAPS=$(echo "$CLASSIFIED"        | awk -F'\t' '$1=="GAP"{print $2" "$3}'        | sed '/^[[:space:]]*$/d')

if [[ -n "$REGRESSIONS" ]]; then
    RCOUNT=$(echo "$REGRESSIONS" | wc -l | tr -d ' ')
    log "REGRESSIONS (${RCOUNT}): $(echo "$REGRESSIONS" | tr '\n' '; ')"

    # File per-regression detail to controlplane (human review).
    while IFS= read -r failure; do
        [[ -z "$failure" ]] && continue
        BENCH=$(echo "$failure" | cut -d' ' -f1)
        CATS=$(echo "$failure" | grep -oE '\[.*\]' || echo "[unknown]")
        ailang messages send controlplane \
            "Nightly eval REGRESSION: benchmark '${BENCH}' passed ALL trials in the previous run but failed BOTH trials on ${DATE}.
Error category: ${CATS}
Model: ${MODEL} (local, tiers:${BENCH_TIERS})
Results: ${RESULTS_DIR}  (prev: ${PREV_RESULTS})
It was solid last night, so this is a fresh solid->broken regression — investigate." \
            --title "Nightly regression: ${BENCH} (${DATE})" \
            --from "nightly-eval" \
            --type "bug" 2>/dev/null || true
        log "  filed regression to controlplane: ${BENCH}"
    done <<< "$REGRESSIONS"

    # ONE Discord ping for regressions only (Pub/Sub → daemon → Discord).
    # public-feedback is the EventType Discord's filter accepts.
    AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac-dev \
    ailang messages send public-feedback \
        "Nightly eval: ${RCOUNT} REGRESSION(s) on ${DATE} — benchmarks that previously passed now fail.
Benchmarks: $(echo "$REGRESSIONS" | cut -d' ' -f1 | tr '\n' ' ')
Model: ${MODEL} | Tiers: ${BENCH_TIERS}
Details filed to controlplane inbox." \
        --title "Nightly eval: ${RCOUNT} regression(s) (${DATE})" \
        --from "nightly-eval" 2>/dev/null || true
else
    log "no regressions — no previously-solid benchmark broke tonight"
fi

# Non-regression persistent failures: benchmarks that failed both trials tonight
# but were NOT solid in the previous run (flaky, already-failing, or never-passed).
# Not a fresh break → filed once to controlplane for the gap-finder; NO Discord.
if [[ -n "$GAPS" ]]; then
    GCOUNT=$(echo "$GAPS" | wc -l | tr -d ' ')
    log "non-regression failures (${GCOUNT}, no Discord): $(echo "$GAPS" | tr '\n' '; ')"
    ailang messages send controlplane \
        "Nightly eval: ${GCOUNT} non-regression failure(s) on ${DATE} (flaky / already-failing / never-passed — no alert).
Benchmarks: $(echo "$GAPS" | cut -d' ' -f1 | tr '\n' ' ')
Model: ${MODEL} | Tiers: ${BENCH_TIERS}
Not solid in the previous run → flaky noise or a known capability gap, not a fresh regression. Gap-finder candidates." \
        --title "Nightly eval: ${GCOUNT} non-regression failure(s) (${DATE})" \
        --from "nightly-eval" \
        --type "note" 2>/dev/null || true
fi

# Broadcast overall summary to controlplane inbox (local, no Discord on success).
# PASS already carries the "<passed>/<total>" fraction.
REG_NAMES=$( [[ -n "$REGRESSIONS" ]] && echo "$REGRESSIONS" | cut -d' ' -f1 | tr '\n' ' ' || echo "none" )
GAP_NAMES=$( [[ -n "$GAPS" ]] && echo "$GAPS" | cut -d' ' -f1 | tr '\n' ' ' || echo "none" )
ailang messages send controlplane \
    "Nightly eval complete: ${PASS} (${RATE}) on ${DATE}.
Build: ${BUILD_VERSION} (committed ${SHORT})
Model: ${MODEL} | Tiers: ${BENCH_TIERS} | Trials: 2
Regressions: ${REG_NAMES}| Non-regression failures: ${GAP_NAMES}" \
    --title "Nightly eval: ${PASS} (${DATE})" \
    --from "nightly-eval" 2>/dev/null || true

log "=== nightly eval done ==="
