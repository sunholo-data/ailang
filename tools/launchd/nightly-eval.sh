#!/usr/bin/env bash
# nightly-eval.sh — nightly regression-guard smoke run on the local rig.
#
# Runs the canonical smoke tier (every `tier: smoke` benchmark, derived
# dynamically) against the accuracy-first local Qwen model
# (opencode-qwen3-5-35b-a3b-mxfp8). Persistent failures are filed to the
# controlplane inbox for human triage.
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

# Derive smoke benchmarks from tier tags (stays in sync automatically)
SMOKE=$(grep -l 'tier: smoke' benchmarks/*.yml 2>/dev/null \
    | xargs -n1 basename | sed 's/\.yml$//' | sort | paste -sd, -)

if [[ -z "$SMOKE" ]]; then
    log "ERROR: could not derive smoke set — no tier:smoke benchmarks found"
    exit 1
fi

log "smoke tier: $(echo $SMOKE | tr ',' '\n' | wc -l | tr -d ' ') benchmarks"
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
        --models opencode-qwen3-5-35b-a3b-mxfp8 \
        --benchmarks "$SMOKE" \
        --langs ailang \
        --microrag "$mode" \
        --output "$outdir" \
        --parallel 1 \
        --trials 2 >> "$LOG" 2>&1
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
    ailang messages send controlplane \
        "Weekly μRAG A/B (${DATE}): on=${PASS_ON}/17  off=${PASS_OFF}/17. Delta=$(( ${PASS_ON:-0} - ${PASS_OFF:-0} 2>/dev/null || echo "?")) benchmarks. Results: ${RESULTS_DIR}_rag_on vs ${RESULTS_DIR}_rag_off" \
        --title "μRAG A/B result (${DATE})" --from "nightly-eval" 2>/dev/null || true
fi

# Use the rag_on results for regression detection (canonical arm)
RESULTS_AGENT="${RESULTS_DIR}_rag_on/agent"
export RESULTS_AGENT
PASS=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); print(f'{p}/{len(r)}')" 2>/dev/null || echo "?/?")
RATE=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); n=len(r); print(f'{100*p//n if n else 0}%')" 2>/dev/null || echo "?%")

log "regression check result: ${PASS} (${RATE})"

# Identify *persistent* failures: a benchmark whose EVERY trial failed.
# A benchmark that passed at least one of its trials is flaky-but-recovered,
# not a regression, and must NOT generate an alert.
#
# Result files are named  <bench>[_trialN]_<lang>_<model>_<ts>.json
# (trial 1 has no _trialN infix; trial 2 is "_trial2", etc.). Both trials must
# collapse to the same benchmark id, otherwise each trial looks like its own
# single-trial benchmark and any one failing trial trips all(not ok). That bug
# is why flaky 1/2 benchmarks were filed as "failed both trials" regressions.
FAILURES=$(python3 - <<'PY'
import json, glob, os, re
from collections import defaultdict

# Failures in these categories are infra/transport noise (e.g. the executor
# hung mid-generation), NOT language/prompt/stdlib regressions. They surface in
# the overall pass count + suite log; don't escalate them as regression bugs.
INFRA_CATEGORIES = {"api_error", "timeout", "executor_error"}

results_agent = os.environ.get("RESULTS_AGENT", "")

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

# Persistent failure = ≥2 trials seen, every trial failed, and at least one
# failure is a real (non-infra) category. A single passing trial clears it.
persistent = []
for bench, results in trials.items():
    if len(results) >= 2 and all(not ok for ok,_ in results):
        cats = {c for _,c in results}
        if cats - INFRA_CATEGORIES:
            persistent.append(f"{bench} [{','.join(sorted(cats))}]")
print("\n".join(sorted(persistent)))
PY
RESULTS_DIR="$RESULTS_DIR"
)

if [[ -n "$FAILURES" ]]; then
    COUNT=$(echo "$FAILURES" | wc -l | tr -d ' ')
    log "persistent failures (${COUNT}): $FAILURES"

    # File per-failure detail messages to controlplane (human review).
    # NOT to design-doc-creator — triage decides later whether to promote.
    while IFS= read -r failure; do
        BENCH=$(echo "$failure" | cut -d' ' -f1)
        CATS=$(echo "$failure" | grep -oE '\[.*\]' || echo "[unknown]")
        ailang messages send controlplane \
            "Nightly eval regression: benchmark '${BENCH}' failed both trials on ${DATE}.
Error category: ${CATS}
Model: opencode-qwen3-5-35b-a3b-mxfp8 (local, tier:smoke)
Results: ${RESULTS_DIR}
This may indicate a prompt gap, stdlib regression, or model-capability boundary.
If it passed in a previous nightly run, this is a regression. If it has never passed,
it is a known language gap worth documenting." \
            --title "Nightly regression: ${BENCH} (${DATE})" \
            --from "nightly-eval" \
            --type "bug" 2>/dev/null || true
        log "  filed to controlplane: ${BENCH}"
    done <<< "$FAILURES"

    # Broadcast failure summary via Pub/Sub → daemon → Discord (one ping, not N).
    # public-feedback inbox is the EventType that Discord's filter accepts.
    AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac-dev \
    ailang messages send public-feedback \
        "Nightly eval: ${COUNT} persistent failure(s) on ${DATE}.
Benchmarks: $(echo "$FAILURES" | cut -d' ' -f1 | tr '\n' ' ')
Model: opencode-qwen3-5-35b-a3b-mxfp8 | Tier: smoke
Details filed to controlplane inbox." \
        --title "Nightly eval: ${COUNT} failure(s) (${DATE})" \
        --from "nightly-eval" 2>/dev/null || true
else
    log "no persistent failures — all smoke benchmarks passing"
fi

# Broadcast summary to controlplane inbox (local, no Discord ping on success)
ailang messages send controlplane \
    "Nightly eval complete: ${PASS}/${TOTAL} (${RATE}) on ${DATE}.
Build: ${BUILD_VERSION} (committed ${SHORT})
Model: opencode-qwen3-5-35b-a3b-mxfp8 | Tier: smoke | Trials: 2
$([ -n "$FAILURES" ] && echo "Persistent failures: $(echo $FAILURES | tr '\n' ', ')" || echo "All benchmarks passing.")" \
    --title "Nightly eval: ${PASS}/${TOTAL} (${DATE})" \
    --from "nightly-eval" 2>/dev/null || true

log "=== nightly eval done ==="
