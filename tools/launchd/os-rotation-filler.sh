#!/usr/bin/env bash
# os-rotation-filler.sh — incremental, background OS cross-language/harness eval
# (M-EVAL-OS-CONTINUOUS-ROTATION). Runs a SMALL, time-boxed chunk of the
# 4-language sweep on otherwise-idle rig time, between releases, NEVER overlapping
# the nightlies (shared rig lock; yields immediately if busy). Accumulates into a
# rolling rotation dir and refreshes the OS/Local dashboard JSON once per full pass.
#
# Schedule via launchd StartInterval (see dev.ailang.os-rotation-filler.plist).
# Portable to macOS bash 3.2 (no mapfile/flock).
set -uo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO" || exit 1
# shellcheck source=tools/launchd/rig-lock.sh
source "$(dirname "$0")/rig-lock.sh"

LOG=/tmp/ailang-os-filler.log
log() { echo "[$(date '+%F %H:%M:%S')] $*" | tee -a "$LOG"; }

MODEL="${OS_FILLER_MODEL:-opencode-qwen3-5-35b-a3b-mxfp8}"
LANGS="${OS_FILLER_LANGS:-ailang,python,javascript,go}"
CHUNK="${OS_FILLER_CHUNK:-3}"                  # benchmarks per cycle
CHUNK_TIMEOUT="${OS_FILLER_TIMEOUT:-1500s}"    # ~25-min wall budget per chunk
ROLL="eval_results/rotation/os-rolling"        # accumulating rotation dir
CURSOR="$HOME/.ailang/state/os-filler-cursor"
BLACKOUT_START="${OS_FILLER_BLACKOUT_START:-04:00}"  # covers nightly + lang-eval + model reloads
BLACKOUT_END="${OS_FILLER_BLACKOUT_END:-07:00}"
AUTOPUSH="${OS_FILLER_PUSH:-0}"   # 0 = accumulate + commit LOCALLY only (safe default);
                                  # set OS_FILLER_PUSH=1 to autonomously push -> docs deploy.

# 1. Blackout window — stay clear of the scheduled nightly jobs.
if rig_in_blackout "$BLACKOUT_START" "$BLACKOUT_END"; then
  log "in blackout ${BLACKOUT_START}-${BLACKOUT_END} — skip"; exit 0
fi

# 2. ollama up?
if ! curl -s --max-time 3 http://localhost:11434/api/version >/dev/null 2>&1; then
  log "ollama unreachable — skip"; exit 0
fi

# 3. Yield if any rig job (nightly / lang-eval) holds the lock.
if ! rig_lock_acquire nowait; then
  log "rig busy — yielding"; exit 0
fi
log "rig lock acquired (filler)"

# 4. 4-language benchmark set (same rule as nightly-lang-eval). No spaces in ids,
#    so word-splitting into an array is bash-3.2 safe.
# shellcheck disable=SC2207
BENCHES=( $(for f in benchmarks/*.yml; do
  L=$(grep -E '^languages:' "$f" 2>/dev/null)
  echo "$L" | grep -q ailang || continue
  echo "$L" | grep -q python || continue
  echo "$L" | grep -q javascript || continue
  echo "$L" | grep -qE '\bgo\b' || continue
  basename "$f" .yml
done) )
TOTAL=${#BENCHES[@]}
if [ "$TOTAL" -eq 0 ]; then log "no 4-language benchmarks"; exit 0; fi

# 5. Round-robin cursor: take the next CHUNK benchmarks. (Robust cold-start; can
#    swap to --benchmarks-by-confidence once agent-mode ratings are warm.)
OFFSET=$(cat "$CURSOR" 2>/dev/null || echo 0)
case "$OFFSET" in (*[!0-9]*) OFFSET=0;; esac
PICK=""
i=0
while [ "$i" -lt "$CHUNK" ] && [ "$i" -lt "$TOTAL" ]; do
  idx=$(((OFFSET + i) % TOTAL))
  PICK="${PICK:+$PICK,}${BENCHES[$idx]}"
  i=$((i + 1))
done
NEXT=$(((OFFSET + CHUNK) % TOTAL))
WRAPPED=0; [ "$NEXT" -le "$OFFSET" ] && WRAPPED=1
mkdir -p "$(dirname "$CURSOR")"; echo "$NEXT" > "$CURSOR"
log "cycle: $PICK (offset $OFFSET/$TOTAL -> $NEXT, wrapped=$WRAPPED)"

# 6. Run the chunk — serial (single-GPU), accumulating via --skip-existing.
mkdir -p "$ROLL"
ailang eval-suite --agent --models "$MODEL" --benchmarks "$PICK" --langs "$LANGS" \
  --parallel 1 --microrag on --trials 3 --skip-existing --timeout "$CHUNK_TIMEOUT" \
  --output "$ROLL" >>"$LOG" 2>&1 || log "chunk had failures (continuing)"

# 7. Regenerate the OS/Local JSON from the cumulative rolling rotation; commit the
#    single file every cycle (keeps the tree clean), but only PUSH on a full-pass
#    wrap to keep deploy churn to ~1/pass. Push uses autostash-rebase for safety.
if ailang eval-publish "rolling-$(date +%Y%m%d)" --rotation "$ROLL" \
      --os-json docs/static/benchmarks/os/latest.json >>"$LOG" 2>&1; then
  if ! git diff --quiet -- docs/static/benchmarks/os/latest.json 2>/dev/null; then
    git add docs/static/benchmarks/os/latest.json
    git commit -q -m "data(os): incremental OS/Local rotation (offset $OFFSET)" \
      -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
    if [ "$AUTOPUSH" = "1" ] && [ "$WRAPPED" -eq 1 ]; then
      git pull --rebase --autostash origin dev >>"$LOG" 2>&1 || true
      git push origin dev >>"$LOG" 2>&1 && log "full pass complete — published + pushed" \
        || log "push failed (retry next wrap)"
    else
      log "committed locally (auto-push OFF — set OS_FILLER_PUSH=1 to publish; wrapped=$WRAPPED)"
    fi
  fi
fi
log "cycle done"
