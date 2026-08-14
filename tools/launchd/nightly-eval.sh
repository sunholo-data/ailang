#!/usr/bin/env bash
# nightly-eval.sh — nightly regression-guard smoke run on the local rig.
#
# Runs the smoke + core tiers (every `tier: smoke` / `tier: core` benchmark,
# default-core included) against the accuracy-first local Qwen model
# (opencode-qwen3-6-35b-a3b-mxfp8). Regressions — benchmarks with a passing
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

# Load API keys (OPENROUTER_API_KEY, etc.) — the local ollama profile's motoko
# canary pre-flight requires OPENROUTER_API_KEY even for local models (motoko
# routes ALL models via OpenRouter), and the non-login launchd env doesn't have
# it. Without this the canary fails, every local model is skipped, eval-suite
# exits "No models support agent evaluation", and the nightly banks ZERO rows
# (#665). Mirrors os-rotation-filler.sh / mission-control.sh. secrets.env uses
# `export KEY=...`.
# shellcheck source=/dev/null
# shellcheck disable=SC1091
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

LOG=/tmp/ailang-nightly-eval.log
RESULTS_DIR="/tmp/nightly_eval_$(date +%Y%m%d)"
DATE=$(date +%Y-%m-%d)

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG"; }

# Shared rig mutex (M-EVAL-OS-CONTINUOUS-ROTATION): the nightly is the priority
# job — wait for any in-flight rig work, then hold the lock for the whole run so
# the background os-rotation-filler never overlaps it.
# shellcheck source=tools/launchd/rig-lock.sh
# shellcheck disable=SC1091
source "$(dirname "$0")/rig-lock.sh"
rig_lock_acquire wait

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
# qwen3.5 retired from every active lane 2026-08-01 (Mark directive on #484:
# "Remove qwen 3.5 only use qwen 3.6"). os-rotation-filler had already dropped it
# on 2026-06-15; this was the last driver still pinning it. The swap does NOT
# blind the regression guard: opencode-qwen3-6-35b-a3b-mxfp8 already carries 60
# banked baselines (44 with >=5 passing trials) from the OS rotation, vs
# qwen3.5's 61/58 — so adaptive thresholds cover 44 benchmarks from day one and
# the other 16 fall back to the fixed THRASH ceiling below until they accrue 5
# passing samples. qwen3.5 registry entries are deliberately KEPT in models.yml:
# 2,438 banked pass-trials are attributed to them.
MODEL="opencode-qwen3-6-35b-a3b-mxfp8"
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
#
# EACH EXPERIMENT GETS ITS OWN NIGHT (Mark, 2026-07-29). Both A/Bs used to hang
# off one Monday gate, which meant Monday paid for both: the microRAG comparison
# AND the fmt comparison in a single night, each needing its own OFF arm. Two
# full comparisons do not fit one night, so they were competing for the same rig
# hours and neither got the trial count it needs.
#
#   Monday    -> microRAG A/B  (on vs off)
#   Wednesday -> fmt A/B       (ollama_fmt vs ollama)
#
# Every other night: microRAG=on only, as a regression guard with no comparison
# overhead. Both A/Bs pool across weeks — see the fmt block's note on why
# McNemar needs accumulation rather than one big night.
DAY_OF_WEEK=$(date +%u)  # 1=Mon … 7=Sun
RUN_AB_MICRORAG=0
RUN_AB_FMT=0
[[ "$DAY_OF_WEEK" == "1" ]] && RUN_AB_MICRORAG=1
[[ "$DAY_OF_WEEK" == "3" ]] && RUN_AB_FMT=1
# AILANG_FORCE_AB=1 forces BOTH (kept for back-compat); the per-experiment
# overrides below force just one, which is what a manual catch-up run wants.
[[ "${AILANG_FORCE_AB:-0}" == "1" ]] && { RUN_AB_MICRORAG=1; RUN_AB_FMT=1; }
[[ "${AILANG_FORCE_AB_MICRORAG:-0}" == "1" ]] && RUN_AB_MICRORAG=1
[[ "${AILANG_FORCE_AB_FMT:-0}" == "1" ]] && RUN_AB_FMT=1
# Opt OUT of one on its own night (e.g. to give the other the whole rig).
[[ "${AILANG_AB_MICRORAG:-1}" == "0" ]] && RUN_AB_MICRORAG=0
[[ "${AILANG_AB_FMT:-1}" == "0" ]] && RUN_AB_FMT=0

# select_ab_benchmarks <model> <max> -- echo a comma-separated benchmark set chosen
# by CONFIDENCE from the ratings DB, or nothing if selection is unavailable.
#
# WHY THIS EXISTS. Both A/Bs used to pick their benchmarks by hand:
# microRAG took every smoke+core benchmark (tier is a proxy for "cheap", not for
# "informative"), and fmt used a list frozen at selection time. Measured against
# the seeded agent ratings on 2026-07-31, 17 of the 32 rated smoke+core
# benchmarks sit in the Trivial band -- every model passes, so the pair is always
# concordant and contributes exactly ZERO to McNemar. That is over half the
# night's GPU buying no information, and it is the mechanism behind the recurring
# "both arms ~100%, 0 discordant pairs, no p-value" nulls.
#
# `--benchmarks-by-confidence` drops the saturated band and ranks the rest by
# proximity to the field's median rating, which is where discordance -- the thing
# McNemar actually consumes -- is maximised. It only became usable once
# `eval-elo --persist` stopped being a silent no-op (see cmd/ailang/eval_elo.go);
# before that the ratings DB had zero agent rows and this path could not run.
#
# NO SILENT FALLBACK: if selection fails, the caller SKIPS its A/B rather than
# quietly running a stale or saturated set. A null from the wrong benchmarks is
# indistinguishable from a null from a working treatment, and we have already
# spent weeks on that ambiguity.
select_ab_benchmarks() {
    local model="$1" max="$2"
    "$BIN" eval-suite --agent --models "$model" \
        --benchmarks-by-confidence auto --max-benchmarks "$max" \
        --langs ailang --dry-run 2>/dev/null \
        | grep '^Benchmarks:' | sed 's/^Benchmarks:[[:space:]]*//' | tr -d ' '
}

run_eval() {
    local mode="$1" outdir="$2"
    log "running smoke: microrag=${mode} → ${outdir}"
    "$BIN" eval-suite --agent \
        --models "$MODEL" \
        --benchmarks "${RAG_BENCH_LIST:-$BENCH_LIST}" \
        --langs ailang \
        --microrag "$mode" \
        --output "$outdir" \
        --parallel 1 \
        --trials 2 \
        --max-tokens-per-bench "$MAX_TOKENS_PER_BENCH" >> "$LOG" 2>&1
}

# Count passing results in an arm dir. Both arms of an A/B MUST be counted the
# same way — the old PASS_ON had a grep-first path whose `||` fallback could
# never fire (the pipeline's exit status is cut's, which is 0 even when grep
# matched nothing). It therefore yielded an EMPTY string, which `${PASS_ON:-0}`
# turned into a literal 0 and banked as a real 0% measurement. That is exactly
# how the 2026-07-20 row (on_pass=0, delta_pp=-73.8) entered microrag_ab.jsonl.
# Top-level on purpose: BOTH the microRAG (Monday) and fmt (Wednesday) A/B
# blocks call it — defined inside the microRAG block it was undefined on
# fmt-only nights, and under `set -e` the 127 killed the whole run at the
# comparison step (2026-08-05: 8.5h of evals died unbanked at line 477).
count_passes() {
    python3 -c "
import json,glob,sys
fs=glob.glob('$1/agent/*.json')
if not fs: sys.exit(3)
print(sum(1 for f in fs
          for d in [json.load(open(f))]
          if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')))" 2>/dev/null
}

# Top-level for the same reason as count_passes: both A/B blocks log through it.
# Reads eval-paired JSON on stdin, prints a one-line b/c/p summary. The Python
# is a single-quoted shell string, so its own strings MUST be double-quoted and
# never \"-escaped — a \" here reaches Python as a literal backslash and is a
# SyntaxError, which 2>/dev/null converts into an unconditional "parse failed"
# (that exact bug shipped 2026-08-04..13: every A/B night logged "parse failed"
# while the banked JSON was fine). No f-strings: nested same-type quotes are the
# trap this helper exists to avoid.
paired_summary() {
    python3 -c '
import json, sys
d = json.load(sys.stdin)
m = d["mcnemar"]
head = "b=%d c=%d unpaired=%d " % (d["only_on_passed"], d["only_off_passed"], d["unpaired"])
tail = "p=%.4f (%s)" % (m["p_value"], m["method"]) if m["reportable"] else "no p-value: " + m["note"]
print(head + tail)' 2>/dev/null || echo "parse failed"
}

# microRAG arm set: confidence-selected, not the smoke+core tier dump.
    RAG_BENCH_LIST="${AILANG_AB_MICRORAG_BENCHMARKS:-$(select_ab_benchmarks "$MODEL" "${AILANG_AB_MICRORAG_N:-12}")}"
    if [[ -z "$RAG_BENCH_LIST" ]]; then
        log "microRAG A/B SKIPPED: confidence selection returned nothing (seed ratings: go run ./tools/eval-elo --mode agent --persist ~/.ailang/state/observatory.db <results_dir>)"
        ailang messages send controlplane \
            "microRAG A/B skipped (${DATE}): no agent benchmark ratings, so the arm set could not be chosen by headroom. Running the saturated tier set would produce another uninterpretable null." \
            --title "microRAG A/B skipped (${DATE})" --from "nightly-eval" 2>/dev/null || true
        RUN_AB_MICRORAG=0
    fi
    log "microRAG arm set (confidence-selected): ${RAG_BENCH_LIST}"

run_eval "on"  "${RESULTS_DIR}_rag_on"

if [[ "$RUN_AB_MICRORAG" == "1" || "$RUN_AB_FMT" == "1" ]]; then

# --- microRAG A/B (Mondays) -------------------------------------------------
# Guarded WITHOUT re-indenting the ~110-line body: reindentation would bury a
# two-line control-flow change in a whole-block diff. Closing `fi` is marked
# "end microRAG A/B" below.
if [[ "$RUN_AB_MICRORAG" == "1" ]]; then
    log "microRAG A/B night: also running microrag=off"
    run_eval "off" "${RESULTS_DIR}_rag_off"

    # Compare with the shared top-level count_passes (see its comment for why
    # both arms must be counted the same way).
    # `|| true`: under set -e a bare failing substitution kills the run before
    # the non-numeric guard below can turn it into a refuse-to-bank.
    PASS_ON=$(count_passes "${RESULTS_DIR}_rag_on" || true)
    PASS_OFF=$(count_passes "${RESULTS_DIR}_rag_off" || true)
    log "A/B result: microrag_on=${PASS_ON} microrag_off=${PASS_OFF}"
    if [[ "${PASS_ON:-x}" =~ ^[0-9]+$ && "${PASS_OFF:-x}" =~ ^[0-9]+$ ]]; then
        DELTA=$(( PASS_ON - PASS_OFF ))
    else
        DELTA="?"
    fi
    ailang messages send controlplane \
        "Weekly μRAG A/B (${DATE}): on=${PASS_ON}/${TIER_COUNT}  off=${PASS_OFF}/${TIER_COUNT}. Delta=${DELTA} benchmarks. Results: ${RESULTS_DIR}_rag_on vs ${RESULTS_DIR}_rag_off" \
        --title "μRAG A/B result (${DATE})" --from "nightly-eval" 2>/dev/null || true

    # Persist the weekly A/B so deltas ACCUMULATE into a committed, trend-able
    # history. The controlplane message above is ephemeral and /tmp is cleared on
    # reboot, so without this we run the A/B but never learn from it. Weekly
    # cadence ⇒ ~1 commit/week (negligible churn — no extra machine time, just a
    # one-line append). Write to the MAIN repo working tree ($REPO), NOT the
    # detached build worktree we cd'd into. Best-effort: a git failure never
    # touches the eval result.
    # Paired per-benchmark outcomes + McNemar (m-eval-measurement-contract M3).
    # Aggregate rate deltas cannot resolve anything at n=84 (unpaired SE ~6.8pp,
    # so only effects >~13pp reach significance). Pairing on the benchmark
    # cancels the between-benchmark variance and costs no extra GPU time — it is
    # a better reading of runs we already do. Computed in Go, not shell: the old
    # shell arithmetic is what produced the 2026-07-20 artefact.
    PAIRED=$("$BIN" eval-paired --with-pairs=true "${RESULTS_DIR}_rag_on" "${RESULTS_DIR}_rag_off" 2>>"$LOG" || echo "")
    if [[ -n "$PAIRED" ]]; then
        log "paired: $(echo "$PAIRED" | paired_summary)"
    else
        log "paired: eval-paired produced no output (see $LOG)"
    fi

    HIST="$REPO/docs/static/benchmarks/microrag_ab.jsonl"
    ON_TOTAL=$(find "${RESULTS_DIR}_rag_on"/agent -name '*.json' -type f 2>/dev/null | wc -l | tr -d ' ')
    OFF_TOTAL=$(find "${RESULTS_DIR}_rag_off"/agent -name '*.json' -type f 2>/dev/null | wc -l | tr -d ' ')
    REC=$(python3 -c "
import json
on_p,on_t=${PASS_ON:-0},${ON_TOTAL:-0}; off_p,off_t=${PASS_OFF:-0},${OFF_TOTAL:-0}
print(json.dumps({
  'date':'${DATE}','model':'${MODEL}','lang':'ailang','trials':2,
  'on_pass':on_p,'on_total':on_t,'off_pass':off_p,'off_total':off_t,
  'on_rate':round(on_p/on_t,4) if on_t else None,
  'off_rate':round(off_p/off_t,4) if off_t else None,
  'delta_pp':round(100*(on_p/on_t-off_p/off_t),1) if on_t and off_t else None}))" 2>/dev/null)
    # Data integrity: never bank a broken run as a measurement. A zero-pass arm
    # or a zero-file arm is a HARNESS failure, not a result — banking it poisons
    # the trend (see the 2026-07-20 row). Loud skip, no silent fallback.
    AB_VALID=1
    for pair in "ON:${PASS_ON}:${ON_TOTAL}" "OFF:${PASS_OFF}:${OFF_TOTAL}"; do
        arm=${pair%%:*}; rest=${pair#*:}; p=${rest%%:*}; t=${rest#*:}
        if ! [[ "$p" =~ ^[0-9]+$ ]] || ! [[ "$t" =~ ^[0-9]+$ ]] || [[ "$t" -eq 0 ]] || [[ "$p" -eq 0 ]]; then
            log "A/B INVALID: ${arm} arm pass='${p}' total='${t}' — refusing to bank this week"
            AB_VALID=0
        fi
    done

    if [[ "$AB_VALID" == "0" ]]; then
        ailang messages send controlplane \
            "Weekly A/B (${DATE}) NOT banked — an arm returned zero passes or zero result files. This is a harness failure, not a result. Check ${RESULTS_DIR}_rag_on / _rag_off and $LOG" \
            --title "A/B invalid (${DATE})" --from "nightly-eval" 2>/dev/null || true
        REC=""
    fi

    # Fold the paired analysis into the banked row so the test is recomputable
    # from history without re-running the eval.
    if [[ -n "$REC" && -n "$PAIRED" ]]; then
        REC=$(python3 -c "
import json,sys
rec=json.loads(sys.argv[1]); paired=json.loads(sys.argv[2])
rec['pairs']=paired.get('pairs')
rec['only_on_passed']=paired.get('only_on_passed')
rec['only_off_passed']=paired.get('only_off_passed')
rec['unpaired']=paired.get('unpaired')
rec['mcnemar']=paired.get('mcnemar')
print(json.dumps(rec))" "$REC" "$PAIRED" 2>/dev/null || echo "$REC")
    fi

    if [[ -n "$REC" ]]; then
        echo "$REC" >> "$HIST"
        log "μRAG A/B persisted -> docs/static/benchmarks/microrag_ab.jsonl"
        if git -C "$REPO" add "$HIST" 2>>"$LOG" && ! git -C "$REPO" diff --cached --quiet -- "$HIST" 2>/dev/null; then
            git -C "$REPO" commit -q \
                -m "data(microrag): weekly A/B ${DATE} (on=${PASS_ON}/${ON_TOTAL} off=${PASS_OFF}/${OFF_TOTAL})" \
                -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
            if [ "${OS_FILLER_PUSH:-0}" = "1" ]; then
                git -C "$REPO" pull --rebase --autostash --quiet origin dev >>"$LOG" 2>&1 || true
                if git -C "$REPO" push --quiet origin dev >>"$LOG" 2>&1; then
                    log "μRAG A/B history pushed"
                else
                    log "μRAG A/B push failed (committed locally)"
                fi
            else
                log "μRAG A/B history committed locally (push off)"
            fi
        fi
    fi
fi  # end microRAG A/B

    # ------------------------------------------------------------------
    # Wednesday experiment: motoko fmt extension (M-EVAL-FMT-WEAKMODEL-AB).
    #
    # Hypothesis: running `ailang fmt --write` on every .ail write removes
    # syntax drift, so a WEAK local model re-reads canonical code each turn and
    # spirals into compile-stuck loops less often. Haiku data is useless here —
    # both arms sit at ~96%, pure ceiling. The subject has to be the local model.
    #
    # Arms are two models.yml entries that differ ONLY by motoko profile:
    #   ON  = motoko-local-qwen3-6-fmt          (profile ollama_fmt: +fmt)
    #   OFF = motoko-local-qwen3-6-35b-a3b-mxfp8 (profile ollama)
    # Both are local/$0 and drive the SAME loaded qwen3.6 — no extra VRAM.
    #
    # fmt is deliberately NOT added to the ollama/cloud profiles: doing that
    # before the A/B concludes would destroy the control.
    # ------------------------------------------------------------------
    if [[ "$RUN_AB_FMT" == "1" ]]; then
        # PREREGISTERED scope (M5 hardset prereg / M6). NOT the smoke+core set:
        # fmt removes syntax drift, so the experiment has to run where drift
        # actually occurs. The smoke set is largely drift-free — fizzbuzz has no
        # syntax drift to remove — which is the same category of mistake as
        # running the A/B against haiku at a 96% ceiling: a subject that cannot
        # exhibit the failure mode cannot show the remedy working.
        #
        # AMENDED 2026-07-29 after the first fully-instrumented run, which made
        # that exact mistake a THIRD time. The original set
        # (contract_rle_roundtrip, config_file_parser, contract_roman_numeral,
        # contract_sorted_merge, log_file_analyzer) was picked from banked rates
        # of 35-86%, but the OFF arm came in at 10/10 — and 4 of the 5 were 4/4.
        # P(10/10 | those rates) ~= 0.3%, so the set did not get lucky: THE
        # BASELINE MOVED. Every pre-2026-07-29 row predates the MOTOKO_REPO fix,
        # so no extension had ever loaded — not even DP7's per-edit `ailang
        # check`. With extensions live, the control arm alone clears that set.
        #
        # DO NOT PICK THIS SET FROM RAW PASS RATES. That is what produced both
        # earlier mistakes. A raw rate blends three things — how hard the
        # benchmark is, how strong the model was, and which harness era the rows
        # came from — so it moves when any of them moves, which is exactly how a
        # set chosen at 35-86% arrived at a 10/10 control arm.
        #
        # `ailang eval-elo <dir> --json` already solves this: it fits benchmark
        # difficulty as a latent parameter SEPARATE from model strength, so a
        # benchmark's ailang_elo is era-robust in a way its pass rate is not.
        # Selection rule — expected score under the standard ELO logistic:
        #
        #   E[pass] = 1 / (1 + 10^((bench_elo - model_elo) / 400))
        #
        # and keep benchmarks with E[pass] in ~0.20-0.70, nearest 0.50 first,
        # because DISCORDANT PAIRS are what McNemar consumes and discordance is
        # maximised where the outcome is least certain. Both tails are useless:
        # at E=0.93 both arms pass, at E=0.15 both arms fail, and neither
        # produces a pair that can move the test.
        #
        # Scored against subject ELO 2196 (motoko-local-qwen3-6-*, agent mode,
        # 69 benchmarks of coverage, non-provisional), the first amendment to
        # this list was wrong on 3 of 5 IN BOTH DIRECTIONS — red_black_tree
        # E=0.93 and csv_to_json_converter E=0.81 (raw rates said 56%/59%), and
        # docx_reimplement E=0.15. docx is deliberately NOT here: it is the
        # hardest benchmark we have and it fails ~100% compile-stuck, so it is
        # its own line of work, not an A/B row that can resolve anything.
        #
        #   config_file_parser .55 | parse_prec_climb .47 | legal_obligation .44
        #   ssa_constant_fold  .35 | bytecode_vm_trace .68
        # Confidence-selected, not frozen. The list below WAS hand-picked by the
        # ELO rule above, against subject ELO 2196 -- but a hardcoded set silently
        # goes stale the moment the subject's rating moves, which is exactly the
        # failure this comment warns about one paragraph earlier. Selecting at run
        # time from the ratings DB makes the rule self-applying instead of a
        # snapshot someone has to remember to redo.
        FMT_BENCH_LIST="${AILANG_AB_FMT_BENCHMARKS:-$(select_ab_benchmarks "motoko-local-qwen3-6-fmt" "${AILANG_AB_FMT_N:-6}")}"
        if [[ -z "$FMT_BENCH_LIST" ]]; then
            log "fmt A/B SKIPPED: confidence selection returned nothing (seed ratings: go run ./tools/eval-elo --mode agent --persist ~/.ailang/state/observatory.db <results_dir>)"
            ailang messages send controlplane \
                "fmt A/B skipped (${DATE}): no agent benchmark ratings, so the arm set could not be chosen by headroom. Running a stale set would produce another uninterpretable null." \
                --title "fmt A/B skipped (${DATE})" --from "nightly-eval" 2>/dev/null || true
            RUN_AB_FMT=0
        fi
        # N>=5 per the prereg. The previous 2 was inherited from the microRAG
        # block, not chosen for this experiment.
        FMT_TRIALS="${AILANG_AB_FMT_TRIALS:-5}"
        log "Wednesday A/B: motoko fmt extension (on vs off) — ELO-selected set, N=${FMT_TRIALS}"
        log "  benchmarks: ${FMT_BENCH_LIST}"
        for arm in on off; do
            case "$arm" in
                on)  m="motoko-local-qwen3-6-fmt" ;;
                off) m="motoko-local-qwen3-6-35b-a3b-mxfp8" ;;
            esac
            log "  fmt=${arm} model=${m}"
            "$BIN" eval-suite --agent \
                --models "$m" \
                --benchmarks "$FMT_BENCH_LIST" \
                --langs ailang \
                --output "${RESULTS_DIR}_fmt_${arm}" \
                --parallel 1 \
                --trials "$FMT_TRIALS" \
                --max-tokens-per-bench "$MAX_TOKENS_PER_BENCH" >> "$LOG" 2>&1 || \
                log "  fmt=${arm} eval-suite exited non-zero (see $LOG)"
        done

        # Paired analysis for the fmt arm too. Banking aggregates only would
        # leave THIS experiment — the one the pairing work was motivated by —
        # stuck with the same ~13pp detection floor that made the microRAG
        # weekly unable to conclude anything.
        FMT_PAIRED=$("$BIN" eval-paired --with-pairs=true "${RESULTS_DIR}_fmt_on" "${RESULTS_DIR}_fmt_off" 2>>"$LOG" || echo "")
        if [[ -n "$FMT_PAIRED" ]]; then
            log "fmt paired: $(echo "$FMT_PAIRED" | paired_summary)"
        else
            log "fmt paired: eval-paired produced no output (see $LOG)"
        fi

        # `|| true`: same as the microRAG arm — let the FMT_VALID guard see an
        # empty value instead of set -e killing the run here.
        FMT_ON=$(count_passes "${RESULTS_DIR}_fmt_on" || true)
        FMT_OFF=$(count_passes "${RESULTS_DIR}_fmt_off" || true)
        FMT_ON_T=$(find "${RESULTS_DIR}_fmt_on"/agent -name '*.json' -type f 2>/dev/null | wc -l | tr -d ' ')
        FMT_OFF_T=$(find "${RESULTS_DIR}_fmt_off"/agent -name '*.json' -type f 2>/dev/null | wc -l | tr -d ' ')
        log "fmt A/B result: on=${FMT_ON}/${FMT_ON_T} off=${FMT_OFF}/${FMT_OFF_T}"

        FMT_VALID=1
        for pair in "ON:${FMT_ON}:${FMT_ON_T}" "OFF:${FMT_OFF}:${FMT_OFF_T}"; do
            arm=${pair%%:*}; rest=${pair#*:}; p=${rest%%:*}; t=${rest#*:}
            if ! [[ "$p" =~ ^[0-9]+$ ]] || ! [[ "$t" =~ ^[0-9]+$ ]] || [[ "$t" -eq 0 ]] || [[ "$p" -eq 0 ]]; then
                log "fmt A/B INVALID: ${arm} arm pass='${p}' total='${t}' — refusing to bank"
                FMT_VALID=0
            fi
        done

        if [[ "$FMT_VALID" == "1" ]]; then
            FMT_HIST="$REPO/docs/static/benchmarks/fmt_ab.jsonl"
            FREC=$(python3 -c "
import json
on_p,on_t=${FMT_ON},${FMT_ON_T}; off_p,off_t=${FMT_OFF},${FMT_OFF_T}
print(json.dumps({
  'date':'${DATE}','experiment':'motoko_ext_fmt','lang':'ailang','trials':${FMT_TRIALS},
  'on_model':'motoko-local-qwen3-6-fmt','off_model':'motoko-local-qwen3-6-35b-a3b-mxfp8',
  'on_pass':on_p,'on_total':on_t,'off_pass':off_p,'off_total':off_t,
  'on_rate':round(on_p/on_t,4),'off_rate':round(off_p/off_t,4),
  'delta_pp':round(100*(on_p/on_t-off_p/off_t),1)}))" 2>/dev/null)
            if [[ -n "$FREC" && -n "$FMT_PAIRED" ]]; then
                FREC=$(python3 -c "
import json,sys
rec=json.loads(sys.argv[1]); paired=json.loads(sys.argv[2])
for k in ('pairs','only_on_passed','only_off_passed','unpaired','mcnemar','headroom'):
    rec[k]=paired.get(k)
print(json.dumps(rec))" "$FREC" "$FMT_PAIRED" 2>/dev/null || echo "$FREC")
            fi

            if [[ -n "$FREC" ]]; then
                echo "$FREC" >> "$FMT_HIST"
                log "fmt A/B persisted -> docs/static/benchmarks/fmt_ab.jsonl"
                if git -C "$REPO" add "$FMT_HIST" 2>>"$LOG" && ! git -C "$REPO" diff --cached --quiet -- "$FMT_HIST" 2>/dev/null; then
                    git -C "$REPO" commit -q \
                        -m "data(fmt): weekly A/B ${DATE} (on=${FMT_ON}/${FMT_ON_T} off=${FMT_OFF}/${FMT_OFF_T})" \
                        -m "Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>" 2>>"$LOG" || true
                fi
            fi
        fi

        ailang messages send controlplane \
            "Weekly fmt A/B (${DATE}): on=${FMT_ON}/${FMT_ON_T} off=${FMT_OFF}/${FMT_OFF_T} valid=${FMT_VALID}" \
            --title "motoko fmt A/B (${DATE})" --from "nightly-eval" 2>/dev/null || true
    fi
fi

# Use the rag_on results for regression detection (canonical arm)
RESULTS_AGENT="${RESULTS_DIR}_rag_on/agent"
export RESULTS_AGENT
PASS=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); print(f'{p}/{len(r)}')" 2>/dev/null || echo "?/?")
RATE=$(python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('${RESULTS_AGENT}/*.json')]; p=sum(1 for d in r if d.get('compile_ok') and d.get('runtime_ok') and d.get('stdout_ok')); n=len(r); print(f'{100*p//n if n else 0}%')" 2>/dev/null || echo "?%")

log "regression check result: ${PASS} (${RATE})"

# Durable cross-night history is classifier-owned. Absence is deliberately a
# loud degraded state; only a manual --bootstrap may create the history file.
HISTORY="$HOME/.ailang/state/nightly-eval-history.jsonl"
CLASSIFIED=$(python3 "$WT/tools/nightly_classify.py" \
    --tonight "$RESULTS_AGENT" \
    --history "$HISTORY" \
    --model "$MODEL" \
    --window-nights 5 \
    --min-nights 2 \
    --min-trials 4 \
    --escalate-after 3 \
    --update-history)

HEALTH=$(echo "$CLASSIFIED" | awk -F'\t' '$1=="HEALTH"{sub(/^HEALTH\t/,""); print}')
INVALID=$(echo "$CLASSIFIED" | awk -F'\t' '$1=="INVALID"{print $2"\t"$3"\t"$4"\t"$5"\t"$6}')
REGRESSIONS=$(echo "$CLASSIFIED" | awk -F'\t' '$1=="REGRESSION"{print $2"\t"$3"\t"$4"\t"$5"\t"$6"\t"$7}' | sed '/^[[:space:]]*$/d')
SUSTAINED=$(echo "$CLASSIFIED"   | awk -F'\t' '$1=="SUSTAINED-FAILURE"{print $2"\t"$3"\t"$4"\t"$5"\t"$6"\t"$7}' | sed '/^[[:space:]]*$/d')
SUSPECTED=$(echo "$CLASSIFIED"   | awk -F'\t' '$1=="SUSPECTED-FLAKE"{print $2"\t"$3"\t"$4"\t"$5"\t"$6}' | sed '/^[[:space:]]*$/d')
GAPS=$(echo "$CLASSIFIED"        | awk -F'\t' '$1=="GAP"{print $2"\t"$3"\t"$4"\t"$5"\t"$6}' | sed '/^[[:space:]]*$/d')
INSUFFICIENT=$(echo "$CLASSIFIED" | awk -F'\t' '$1=="INSUFFICIENT-HISTORY"{print $2"\t"$3"\t"$4"\t"$5"\t"$6}' | sed '/^[[:space:]]*$/d')
log "$HEALTH"

if [[ -n "$INVALID" ]]; then
    IFS=$'\t' read -r INVALID_REASON INVALID_TAINT INVALID_RATE INVALID_MEDIAN INVALID_CATEGORY <<< "$INVALID"
    INVALID_BANNER="INVALID nightly run: ${INVALID_REASON}; unmeasured ${INVALID_TAINT} ${INVALID_CATEGORY}; pass rate ${INVALID_RATE} vs trailing median ${INVALID_MEDIAN}."
    log "$INVALID_BANNER"
    ailang messages send controlplane \
        "${INVALID_BANNER}
${HEALTH}
No per-benchmark verdicts were filed.
Model: ${MODEL} | Tiers: ${BENCH_TIERS} | Results: ${RESULTS_DIR}" \
        --title "Nightly eval INVALID (${DATE})" \
        --from "nightly-eval" \
        --type "note" 2>/dev/null || true
else
if [[ -n "$REGRESSIONS" ]]; then
    RCOUNT=$(echo "$REGRESSIONS" | wc -l | tr -d ' ')
    log "REGRESSIONS (${RCOUNT}): $(echo "$REGRESSIONS" | cut -f1 | tr '\n' '; ')"

    # File per-regression detail to controlplane (human review).
    while IFS=$'\t' read -r BENCH CATS WINDOW NIGHTS CONSEC _ESCALATED; do
        [[ -z "$BENCH" ]] && continue
        ailang messages send controlplane \
            "Nightly eval REGRESSION: benchmark '${BENCH}' failed BOTH trials on ${DATE}.
Error category: ${CATS}
Rule: solid trailing window; prior window ${WINDOW} over ${NIGHTS} nights; failing ${CONSEC}/3.
Model: ${MODEL} (local, tiers:${BENCH_TIERS})
Results: ${RESULTS_DIR}
Investigate this solid-window break." \
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
Benchmarks: $(echo "$REGRESSIONS" | cut -f1 | tr '\n' ' ')
Model: ${MODEL} | Tiers: ${BENCH_TIERS}
Details filed to controlplane inbox." \
        --title "Nightly eval: ${RCOUNT} regression(s) (${DATE})" \
        --from "nightly-eval" 2>/dev/null || true
else
    log "no regressions — no previously-solid benchmark broke tonight"
fi

if [[ -n "$SUSTAINED" ]]; then
    SFCOUNT=$(echo "$SUSTAINED" | wc -l | tr -d ' ')
    log "SUSTAINED FAILURES (${SFCOUNT}, no Discord): $(echo "$SUSTAINED" | cut -f1 | tr '\n' '; ')"
    while IFS=$'\t' read -r BENCH CATS WINDOW NIGHTS CONSEC ESCALATED; do
        [[ -z "$BENCH" ]] && continue
        ailang messages send controlplane \
            "Nightly eval SUSTAINED FAILURE: benchmark '${BENCH}' failed all trials for ${CONSEC} consecutive nights.
Error category: ${CATS}
prior window ${WINDOW} over ${NIGHTS} nights is NOT solid; failing ${CONSEC}/3; escalated from ${ESCALATED}.
This is not a certified fresh break — triage as sustained failure / capability gap.
Model: ${MODEL} (local, tiers:${BENCH_TIERS})
Results: ${RESULTS_DIR}" \
            --title "Nightly sustained failure: ${BENCH} (${DATE})" \
            --from "nightly-eval" \
            --type "bug" 2>/dev/null || true
        log "  filed sustained failure to controlplane: ${BENCH}"
    done <<< "$SUSTAINED"
fi

if [[ -n "$SUSPECTED" ]]; then
    SCOUNT=$(echo "$SUSPECTED" | wc -l | tr -d ' ')
    SUSPECTED_BODY=$(echo "$SUSPECTED" | awk -F'\t' '{printf "%s (%s over %s nights, failing %s/3 toward escalation)\n",$1,$3,$4,$5}')
    log "suspected flakes (${SCOUNT}, no Discord): $(echo "$SUSPECTED" | cut -f1 | tr '\n' '; ')"
    ailang messages send controlplane \
        "Nightly eval: ${SCOUNT} suspected flake(s) on ${DATE} (no alert).
${SUSPECTED_BODY}
Model: ${MODEL} | Tiers: ${BENCH_TIERS}" \
        --title "Nightly eval: ${SCOUNT} suspected-flake(s) (${DATE})" \
        --from "nightly-eval" \
        --type "note" 2>/dev/null || true
fi

# All-fail window remains gap-finder territory; preserve its text contract.
if [[ -n "$GAPS" ]]; then
    GCOUNT=$(echo "$GAPS" | wc -l | tr -d ' ')
    log "non-regression failures (${GCOUNT}, no Discord): $(echo "$GAPS" | cut -f1 | tr '\n' '; ')"
    ailang messages send controlplane \
        "Nightly eval: ${GCOUNT} non-regression failure(s) on ${DATE} (flaky / already-failing / never-passed — no alert).
Benchmarks: $(echo "$GAPS" | cut -f1 | tr '\n' ' ')
Model: ${MODEL} | Tiers: ${BENCH_TIERS}
Not solid in the previous run → flaky noise or a known capability gap, not a fresh regression. Gap-finder candidates." \
        --title "Nightly eval: ${GCOUNT} non-regression failure(s) (${DATE})" \
        --from "nightly-eval" \
        --type "note" 2>/dev/null || true
fi

# Broadcast overall summary to controlplane inbox (local, no Discord on success).
# PASS already carries the "<passed>/<total>" fraction.
REG_NAMES=$( [[ -n "$REGRESSIONS" ]] && echo "$REGRESSIONS" | cut -f1 | tr '\n' ' ' || echo "none" )
SUSTAINED_NAMES=$( [[ -n "$SUSTAINED" ]] && echo "$SUSTAINED" | cut -f1 | tr '\n' ' ' || echo "none" )
GAP_NAMES=$( [[ -n "$GAPS" ]] && echo "$GAPS" | cut -f1 | tr '\n' ' ' || echo "none" )
SUSPECTED_NAMES=$( [[ -n "$SUSPECTED" ]] && echo "$SUSPECTED" | cut -f1 | tr '\n' ' ' || echo "none" )
INSUFFICIENT_BODY=""
if [[ -n "$INSUFFICIENT" ]]; then
    INSUFFICIENT_BODY=$(echo "$INSUFFICIENT" | awk -F'\t' '{printf "insufficient history: %s (%s over %s nights, failing %s/3 toward escalation)\n",$1,$3,$4,$5}')
fi
ailang messages send controlplane \
    "${HEALTH}
Nightly eval complete: ${PASS} (${RATE}) on ${DATE}.
Build: ${BUILD_VERSION} (committed ${SHORT})
Model: ${MODEL} | Tiers: ${BENCH_TIERS} | Trials: 2
Regressions: ${REG_NAMES}| Sustained failures: ${SUSTAINED_NAMES}| Suspected flakes: ${SUSPECTED_NAMES}| Non-regression failures: ${GAP_NAMES}
${INSUFFICIENT_BODY}" \
    --title "Nightly eval: ${PASS} (${DATE})" \
    --from "nightly-eval" 2>/dev/null || true
fi

log "=== nightly eval done ==="
