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

# The launchd plist PATH is restricted (no ~/go/bin), so go-installed CLIs like
# the `motoko` shim aren't found by the eval executors' exec.LookPath — the motoko
# arm health-checks fail with "motoko CLI not found". ailang survives via its
# ~/.local/bin symlink; ensure ~/go/bin is also on PATH so motoko (and any other
# go-installed tool) resolves for eval-suite and the executors it spawns.
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"

# Load API keys (OPENROUTER_API_KEY, etc.) — the motoko executor's pre-flight
# health check requires OPENROUTER_API_KEY even for the local ollama profile, and
# the non-login launchd env doesn't have it. secrets.env uses `export KEY=...`.
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

# shellcheck source=tools/launchd/rig-lock.sh
source "$(dirname "$0")/rig-lock.sh"

LOG=/tmp/ailang-os-filler.log
log() { echo "[$(date '+%F %H:%M:%S')] $*" | tee -a "$LOG"; }

# Cross-harness TRIO on the SAME local qwen3.6: opencode (multi-turn) vs pi
# (minimal) vs motoko (AILANG-native). All three drive the ONE loaded qwen3.6
# model, so adding motoko adds runs but no extra VRAM pressure. Run serially
# (--parallel 1 below) so they never contend for the single GPU. This fills the
# harness-comparison columns on the OS/Local leaderboard — the motoko-vs-pi-vs-
# opencode KPI (see m-motoko-self-improvement-loop). Override with
# OS_FILLER_MODELS=a,b. (OS_FILLER_MODEL kept as single-model back-compat alias.)
# motoko ADDED 2026-06-16 after the ollama-loop-convergence fix (lean profile;
# validated 88.5% core single-trial — the rotation gives it 3-trial comparable
# numbers vs pi 95.6% / opencode 88.9%).
# qwen3.5 retired 2026-06-15: it already has a full 39/39 banked pass, and keeping
# it here halved qwen3.6's throughput + added ~100min/cycle of hangs. Now qwen3.6-only
# (the upgrade); compare qwen3.6-fresh vs qwen3.5-banked. Re-add the 3.5 pair to
# OS_FILLER_MODELS if a regression check is ever needed.
MODELS="${OS_FILLER_MODELS:-${OS_FILLER_MODEL:-opencode-qwen3-6-35b-a3b-mxfp8,pi-qwen3-6-35b-a3b-mxfp8,motoko-local-qwen3-6-35b-a3b-mxfp8}}"
LANGS="${OS_FILLER_LANGS:-ailang,python,javascript,go}"
CHUNK="${OS_FILLER_CHUNK:-3}"                  # benchmarks per cycle
CHUNK_TIMEOUT="${OS_FILLER_TIMEOUT:-1500s}"    # ~25-min wall budget per chunk
ROLL="eval_results/rotation/os-rolling"        # accumulating rotation dir
CURSOR="$HOME/.ailang/state/os-filler-cursor"
# AILANG-FULL tier pass (M-EVAL-OS-FRONTIER-COVERAGE): the 4-language set above is only
# 41/91 benchmarks (2/16 frontier) because it demands ailang+python+js+go. Set
# OS_FILLER_AILANG_FULL=1 to ALSO rotate the local models through the FULL
# core+stretch+frontier tiers in AILANG ONLY (own independent cursor), so
# frontier/stretch benchmarks enter the local rotation over time. Off by default.
AILANG_FULL="${OS_FILLER_AILANG_FULL:-0}"
FULL_TIERS="${OS_FILLER_FULL_TIERS:-core,stretch,frontier}"
FULL_CHUNK="${OS_FILLER_FULL_CHUNK:-2}"        # benchmarks per cycle for the ailang-only pass
FULL_CURSOR="$HOME/.ailang/state/os-filler-cursor-ailang-full"
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
  # M-EVAL-RELIABLE-GRADING: ailang-only reimplement benchmarks (grade_entrypoint marker) join
  # the rotation alongside the 4-language set. Checked FIRST because their block-list languages
  # ('languages:\n- ailang') wouldn't pass the single-line 4-language grep below. eval-suite skips
  # unsupported langs per benchmark (SupportsLanguage), so --langs with all 4 runs only ailang.
  if grep -qE '^grade_entrypoint:' "$f"; then basename "$f" .yml; continue; fi
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
# Reimplement benchmarks (M-EVAL-RELIABLE-GRADING) are long agentic tasks (docx ~50min on
# motoko-local): a single trial + a wider wall budget so a reimplement chunk doesn't always
# time out before the slow benchmark banks. --skip-existing still banks each model over cycles.
TRIALS=3; TMO="$CHUNK_TIMEOUT"
case "$PICK" in *reimplement*) TRIALS=1; TMO="5400s";; esac
# --bank-by-version (M-EVAL-VERSION-BANKING): bank under $ROLL/<ailang-version>/ so a new build/release
# re-evals from scratch and history accumulates per release (instead of --skip-existing freezing the
# pre-release numbers forever). --skip-existing still accumulates the chunked sweep WITHIN a version.
ailang eval-suite --agent --models "$MODELS" --benchmarks "$PICK" --langs "$LANGS" \
  --parallel 1 --microrag on --trials "$TRIALS" --skip-existing --bank-by-version --timeout "$TMO" \
  --output "$ROLL" >>"$LOG" 2>&1 || log "chunk had failures (continuing)"

# 6b. AILANG-FULL tier pass (opt-in): rotate the local models through the FULL
#     core+stretch+frontier tiers in AILANG ONLY, using an INDEPENDENT cursor so it
#     doesn't perturb the 4-language round-robin above. This is what pulls frontier
#     and stretch benchmarks into the local rotation (the 4-language set can't, since
#     those benchmarks aren't implemented in all 4 languages). Banks into the SAME
#     $ROLL/<version>/ dir (--skip-existing accumulates), so it flows into both the
#     OS/Local JSON and the unified dashboard below.
if [ "$AILANG_FULL" = "1" ]; then
  # Full tier set: every benchmark whose `tier:` is one of $FULL_TIERS. Language
  # filtering is left to eval-suite (--langs ailang skips any that don't support it).
  # shellcheck disable=SC2207
  BENCHES_FULL=( $(for f in benchmarks/*.yml; do
    T=$(grep -E '^tier:' "$f" 2>/dev/null | head -1 | sed -E 's/^tier:[[:space:]]*//; s/[[:space:]]*#.*//; s/"//g' | tr -d '[:space:]')
    case ",$FULL_TIERS," in (*",$T,"*) basename "$f" .yml;; esac
  done) )
  FULL_TOTAL=${#BENCHES_FULL[@]}
  if [ "$FULL_TOTAL" -eq 0 ]; then
    log "ailang-full: no benchmarks for tiers $FULL_TIERS — skip"
  else
    F_OFFSET=$(cat "$FULL_CURSOR" 2>/dev/null || echo 0)
    case "$F_OFFSET" in (*[!0-9]*) F_OFFSET=0;; esac
    F_PICK=""
    j=0
    while [ "$j" -lt "$FULL_CHUNK" ] && [ "$j" -lt "$FULL_TOTAL" ]; do
      fidx=$(((F_OFFSET + j) % FULL_TOTAL))
      F_PICK="${F_PICK:+$F_PICK,}${BENCHES_FULL[$fidx]}"
      j=$((j + 1))
    done
    F_NEXT=$(((F_OFFSET + FULL_CHUNK) % FULL_TOTAL))
    F_WRAPPED=0; [ "$F_NEXT" -le "$F_OFFSET" ] && F_WRAPPED=1
    mkdir -p "$(dirname "$FULL_CURSOR")"; echo "$F_NEXT" > "$FULL_CURSOR"
    log "ailang-full cycle: $F_PICK (offset $F_OFFSET/$FULL_TOTAL -> $F_NEXT, wrapped=$F_WRAPPED)"
    F_TRIALS=3; F_TMO="$CHUNK_TIMEOUT"
    case "$F_PICK" in *reimplement*) F_TRIALS=1; F_TMO="5400s";; esac
    ailang eval-suite --agent --models "$MODELS" --benchmarks "$F_PICK" --langs ailang \
      --parallel 1 --microrag on --trials "$F_TRIALS" --skip-existing --bank-by-version --timeout "$F_TMO" \
      --output "$ROLL" >>"$LOG" 2>&1 || log "ailang-full chunk had failures (continuing)"
  fi
fi

# 7. Regenerate the OS/Local JSON from the cumulative rolling rotation; commit the
#    single file every cycle (keeps the tree clean), but only PUSH on a full-pass
#    wrap to keep deploy churn to ~1/pass. Push uses autostash-rebase for safety.
if ailang eval-publish "rolling-$(date +%Y%m%d)" --rotation "$ROLL" \
      --os-json docs/static/benchmarks/os/latest.json >>"$LOG" 2>&1; then
  # Stage first, THEN check the staged diff — `git diff --quiet -- <file>` ignores
  # untracked files, so on the first cycle (file not yet in git) it would silently
  # skip the commit and the dashboard would never publish. `git add` + --cached
  # detects both new and modified files.
  git add docs/static/benchmarks/os/latest.json 2>>"$LOG" || true
  if ! git diff --cached --quiet -- docs/static/benchmarks/os/latest.json 2>/dev/null; then
    git commit -q -m "data(os): incremental OS/Local rotation (offset $OFFSET)" \
      -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
    if [ "$AUTOPUSH" = "1" ]; then
      # Publish every cycle that adds data so the dashboard fills incrementally.
      git pull --rebase --autostash origin dev >>"$LOG" 2>&1 || true
      git push origin dev >>"$LOG" 2>&1 && log "published + pushed (cycle; wrapped=$WRAPPED)" \
        || log "push failed (retry next cycle)"
    else
      log "committed locally (auto-push OFF — set OS_FILLER_PUSH=1 to publish)"
    fi
  fi
fi

# 8. UNIFIED dashboard: fold the local rotation into the MAIN leaderboard
#    (docs/static/benchmarks/latest.json) so on-device models sit alongside the
#    cloud frontier in the primary tables — the whole point of the on-device
#    roster is cloud comparison. This re-runs eval-report on the CLOUD baseline
#    of the current release WITH --merge <this rotation version>, but only when
#    that release's cloud baseline already exists (mid-cycle before a release it
#    won't, and cloud-only stays authoritative until post-release re-publishes).
#    Committed to the same LOCAL-only-by-default policy as the OS JSON above.
AILANG_VER="$(tr -d '[:space:]' < std/VERSION 2>/dev/null || true)"
if [ -n "$AILANG_VER" ] && [ -d "eval_results/baselines/${AILANG_VER}" ]; then
  if bash tools/publish-unified-dashboard.sh "$AILANG_VER" >>"$LOG" 2>&1; then
    git add docs/static/benchmarks/latest.json 2>>"$LOG" || true
    if ! git diff --cached --quiet -- docs/static/benchmarks/latest.json 2>/dev/null; then
      git commit -q -m "data(dashboard): unify local rotation into main leaderboard (${AILANG_VER})" \
        -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
      if [ "$AUTOPUSH" = "1" ]; then
        git pull --rebase --autostash origin dev >>"$LOG" 2>&1 || true
        git push origin dev >>"$LOG" 2>&1 && log "unified dashboard pushed (${AILANG_VER})" \
          || log "unified push failed (retry next cycle)"
      else
        log "unified dashboard committed locally (auto-push OFF)"
      fi
    fi
  else
    log "unified dashboard publish skipped/failed for ${AILANG_VER} (continuing)"
  fi
else
  log "no cloud baseline for ${AILANG_VER:-unknown} yet — unified publish deferred to post-release"
fi
log "cycle done"
