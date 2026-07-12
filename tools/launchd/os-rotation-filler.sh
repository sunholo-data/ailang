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
# PRIMARY PASS: rotate the local models through the FULL core+stretch+frontier
# tiers in AILANG ONLY (own independent cursor). AILANG is the point of the
# server, so full AILANG coverage is the default — every core/stretch/frontier
# benchmark enters the local rotation and local ELO becomes comparable to cloud.
# Set OS_FILLER_AILANG_FULL=0 to disable the AILANG-first pass.
AILANG_FULL="${OS_FILLER_AILANG_FULL:-1}"
FULL_TIERS="${OS_FILLER_FULL_TIERS:-core,stretch,frontier}"
FULL_CHUNK="${OS_FILLER_FULL_CHUNK:-3}"        # benchmarks per cycle for the ailang-only pass
# FULL_CURSOR / LAP_MARKER are per-VERSION (set in-block below) so a wrap is a true
# full AILANG lap for that release and coverage resets automatically on a new one.

# CROSS-LANGUAGE pass control. By default the cross-language (python/javascript/go)
# sweep runs AUTOMATICALLY once AILANG coverage is complete for the version — AILANG
# stays FIRST, then the rig keeps going into the other languages on its own. Set
# OS_FILLER_4LANG=1 to FORCE it to interleave early (run every cycle alongside the
# AILANG-first pass) for cross-language signal before AILANG fully fills. Set
# OS_FILLER_AILANG_FULL=0 for a legacy pure cross-language rig.
FORCE_4LANG="${OS_FILLER_4LANG:-0}"
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
mkdir -p "$ROLL"   # ensure the rolling dir exists for both passes (was previously only created inside the 4-lang block)

# The rig is AILANG-FIRST: fill EVERY core+stretch+frontier AILANG benchmark for
# the current version FIRST; only once that coverage is "in" does it hand rig time
# to the cross-language (python/javascript/go) pass. A new release banks under a
# fresh $ROLL/<version>/, so coverage resets and AILANG-first resumes automatically.
# Both passes bank into the SAME $ROLL/<version>/ via --bank-by-version + --skip-existing.
VERSION="$(tr -d '[:space:]' < std/VERSION 2>/dev/null || true)"
VDIR="$ROLL/${VERSION}/agent"
OFFSET=0; WRAPPED=0        # surfaced in the step-7 commit/log; set by whichever pass runs

# ── PRIMARY: AILANG-first full-tier pass ────────────────────────────────────
# Rotate the local models through the FULL core+stretch+frontier tiers in AILANG
# ONLY (independent, per-version cursor). This is what pulls EVERY stretch/frontier
# benchmark into the local rotation — the 4-language set can't, since those
# benchmarks aren't implemented in all 4 langs, which is what caps AILANG at ~45%.
AILANG_DONE=0
if [ "$AILANG_FULL" != "1" ]; then
  AILANG_DONE=1   # AILANG-first disabled -> legacy pure cross-language rig
else
  # Full tier set: every benchmark whose `tier:` is one of $FULL_TIERS. Language
  # filtering is left to eval-suite (--langs ailang skips any that don't support it).
  # shellcheck disable=SC2207
  BENCHES_FULL=( $(for f in benchmarks/*.yml; do
    T=$(grep -E '^tier:' "$f" 2>/dev/null | head -1 | sed -E 's/^tier:[[:space:]]*//; s/[[:space:]]*#.*//; s/"//g' | tr -d '[:space:]')
    case ",$FULL_TIERS," in (*",$T,"*) basename "$f" .yml;; esac
  done) )
  FULL_TOTAL=${#BENCHES_FULL[@]}
  FULL_CURSOR="$ROLL/${VERSION}/.ailang-full-cursor"   # per-version: a wrap == a true full lap for this release
  LAP_MARKER="$ROLL/${VERSION}/.ailang-full-lapped"
  if [ "$FULL_TOTAL" -eq 0 ]; then
    log "ailang-full: no benchmarks for tiers $FULL_TIERS — skip"
    AILANG_DONE=1
  elif [ -f "$LAP_MARKER" ]; then
    # Already made one full AILANG lap for this version — coverage is "in". (A
    # benchmark the local model genuinely can't pass must not block forever.)
    AILANG_DONE=1
    log "ailang-full: version $VERSION already lapped — cross-language pass active"
  else
    # Coverage signal: AILANG is "in" when EVERY full-tier benchmark has a banked
    # ailang result for EVERY local model (== what --skip-existing treats as done).
    # Min distinct-ailang-benches across models = the bottleneck harness.
    FULL_SET=",$(IFS=,; printf '%s' "${BENCHES_FULL[*]}"),"
    MIN_COV=999999
    _OIFS="$IFS"; IFS=','
    for m in $MODELS; do
      IFS="$_OIFS"
      cov=0
      for b in $(ls "$VDIR"/*_ailang_"${m}"_*.json 2>/dev/null | sed -E "s#.*/##; s/_(trial[0-9]+_)?ailang_.*//" | sort -u); do
        case "$FULL_SET" in *",$b,"*) cov=$((cov + 1));; esac
      done
      [ "$cov" -lt "$MIN_COV" ] && MIN_COV="$cov"
      IFS=','
    done
    IFS="$_OIFS"
    [ "$MIN_COV" = "999999" ] && MIN_COV=0

    if [ "$MIN_COV" -ge "$FULL_TOTAL" ]; then
      AILANG_DONE=1
      : > "$LAP_MARKER"   # coverage complete -> mark so future cycles short-circuit to cross-language
      log "ailang-full: coverage COMPLETE for $VERSION ($MIN_COV/$FULL_TOTAL per model) — handing rig time to cross-language pass"
    else
      # Not complete -> spend this cycle advancing AILANG coverage.
      F_OFFSET=$(cat "$FULL_CURSOR" 2>/dev/null || echo 0)
      case "$F_OFFSET" in (*[!0-9]*) F_OFFSET=0;; esac
      F_PICK=""; j=0
      while [ "$j" -lt "$FULL_CHUNK" ] && [ "$j" -lt "$FULL_TOTAL" ]; do
        fidx=$(((F_OFFSET + j) % FULL_TOTAL))
        F_PICK="${F_PICK:+$F_PICK,}${BENCHES_FULL[$fidx]}"
        j=$((j + 1))
      done
      F_NEXT=$(((F_OFFSET + FULL_CHUNK) % FULL_TOTAL))
      F_WRAPPED=0; [ "$F_NEXT" -le "$F_OFFSET" ] && F_WRAPPED=1
      mkdir -p "$(dirname "$FULL_CURSOR")"; echo "$F_NEXT" > "$FULL_CURSOR"
      [ "$F_WRAPPED" = "1" ] && : > "$LAP_MARKER"   # first full lap done -> hand off next cycle
      OFFSET="$F_OFFSET"; WRAPPED="$F_WRAPPED"
      log "ailang-full cycle: $F_PICK (offset $F_OFFSET/$FULL_TOTAL -> $F_NEXT, coverage $MIN_COV/$FULL_TOTAL, wrapped=$F_WRAPPED)"
      F_TRIALS=3; F_TMO="$CHUNK_TIMEOUT"
      case "$F_PICK" in *reimplement*) F_TRIALS=1; F_TMO="5400s";; esac
      # --bank-by-version (M-EVAL-VERSION-BANKING): bank under $ROLL/<ailang-version>/ so a new
      # build/release re-evals from scratch and history accumulates per release.
      ailang eval-suite --agent --models "$MODELS" --benchmarks "$F_PICK" --langs ailang \
        --parallel 1 --microrag on --trials "$F_TRIALS" --skip-existing --bank-by-version --timeout "$F_TMO" \
        --output "$ROLL" >>"$LOG" 2>&1 || log "ailang-full chunk had failures (continuing)"
    fi
  fi
fi

# ── SECONDARY: cross-language comparisons (python/javascript/go) ─────────────
# Runs once AILANG coverage is "in" for the version (automatic hand-off above), or
# immediately when forced early via OS_FILLER_4LANG=1. Same rule as nightly-lang-eval:
# the 4-language-capable benchmark pool. ailang cells are already banked from the
# AILANG-first pass, so --skip-existing effectively adds only python/javascript/go here.
if [ "$AILANG_DONE" = "1" ] || [ "$FORCE_4LANG" = "1" ]; then
  # shellcheck disable=SC2207
  BENCHES=( $(for f in benchmarks/*.yml; do
    # M-EVAL-RELIABLE-GRADING: ailang-only reimplement benchmarks (grade_entrypoint marker) join
    # the rotation alongside the 4-language set. Checked FIRST because their block-list languages
    # ('languages:\n- ailang') wouldn't pass the single-line 4-language grep below.
    if grep -qE '^grade_entrypoint:' "$f"; then basename "$f" .yml; continue; fi
    L=$(grep -E '^languages:' "$f" 2>/dev/null)
    echo "$L" | grep -q ailang || continue
    echo "$L" | grep -q python || continue
    echo "$L" | grep -q javascript || continue
    echo "$L" | grep -qE '\bgo\b' || continue
    basename "$f" .yml
  done) )
  TOTAL=${#BENCHES[@]}
  if [ "$TOTAL" -eq 0 ]; then
    log "no 4-language benchmarks — skipping cross-language pass"
  else
    # Round-robin cursor: take the next CHUNK benchmarks.
    OFFSET=$(cat "$CURSOR" 2>/dev/null || echo 0)
    case "$OFFSET" in (*[!0-9]*) OFFSET=0;; esac
    PICK=""; i=0
    while [ "$i" -lt "$CHUNK" ] && [ "$i" -lt "$TOTAL" ]; do
      idx=$(((OFFSET + i) % TOTAL))
      PICK="${PICK:+$PICK,}${BENCHES[$idx]}"
      i=$((i + 1))
    done
    NEXT=$(((OFFSET + CHUNK) % TOTAL))
    WRAPPED=0; [ "$NEXT" -le "$OFFSET" ] && WRAPPED=1
    mkdir -p "$(dirname "$CURSOR")"; echo "$NEXT" > "$CURSOR"
    log "cross-language cycle: $PICK (offset $OFFSET/$TOTAL -> $NEXT, wrapped=$WRAPPED)"
    TRIALS=3; TMO="$CHUNK_TIMEOUT"
    case "$PICK" in *reimplement*) TRIALS=1; TMO="5400s";; esac
    ailang eval-suite --agent --models "$MODELS" --benchmarks "$PICK" --langs "$LANGS" \
      --parallel 1 --microrag on --trials "$TRIALS" --skip-existing --bank-by-version --timeout "$TMO" \
      --output "$ROLL" >>"$LOG" 2>&1 || log "cross-language chunk had failures (continuing)"
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

# 7b. Version-trend: keep docs/static/benchmarks/os/history.json fresh for the
#     CURRENT version EVERY cycle (idempotent append/replace of the $VERSION entry;
#     NO --reset — that's a post-release-only action). This is what makes the
#     "Local-rig version trend" panel fill in as coverage grows. Previously
#     history.json was updated ONLY by the manual post-release snapshot, so between
#     releases the trend went stale/empty (the "weeks of no data" bug). Now the rig
#     refreshes it itself as the AILANG-first pass banks results.
if [ -n "$VERSION" ] && [ -d "$ROLL/$VERSION" ]; then
  if bash tools/os-release-snapshot.sh "$VERSION" >>"$LOG" 2>&1; then
    git add docs/static/benchmarks/os/history.json 2>>"$LOG" || true
    if ! git diff --cached --quiet -- docs/static/benchmarks/os/history.json 2>/dev/null; then
      git commit -q -m "data(os): refresh version-trend history for ${VERSION}" \
        -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>" 2>>"$LOG" || true
      if [ "$AUTOPUSH" = "1" ]; then
        git pull --rebase --autostash origin dev >>"$LOG" 2>&1 || true
        git push origin dev >>"$LOG" 2>&1 && log "version-trend history pushed (${VERSION})" \
          || log "version-trend push failed (retry next cycle)"
      else
        log "version-trend history committed locally (auto-push OFF)"
      fi
    fi
  else
    log "version-trend snapshot failed for ${VERSION} (continuing)"
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

# 9. Bucket sync (M-EVAL-DATA-HOSTING-DECOUPLE): push the 3 refreshed JSONs to the
#    private GCS bucket EVERY cycle so the docs site — which fetches them at runtime
#    via the dashboard's /benchmarks/ route — shows new data within ~1 min, with no
#    site rebuild / GitHub Pages deploy. Independent of AUTOPUSH (git stays the
#    in-build fallback); opt out with BENCH_BUCKET_SYNC=0. Never fails the cycle.
BENCH_SYNC="${BENCH_BUCKET_SYNC:-1}"
BENCH_BUCKET="${BENCHMARKS_BUCKET:-ailang-multivac-dev-benchmarks}"
if [ "$BENCH_SYNC" = "1" ] && command -v gsutil >/dev/null 2>&1; then
  synced=0
  for rel in latest.json os/latest.json os/history.json; do
    src="docs/static/benchmarks/$rel"
    [ -f "$src" ] || continue
    if gsutil -q -h "Cache-Control:public, max-age=60" cp "$src" "gs://$BENCH_BUCKET/$rel" >>"$LOG" 2>&1; then
      synced=$((synced + 1))
    else
      log "bucket sync failed for $rel (continuing)"
    fi
  done
  [ "$synced" -gt 0 ] && log "bucket sync: $synced/3 JSONs → gs://$BENCH_BUCKET (runtime dashboard fetch)"
elif [ "$BENCH_SYNC" = "1" ]; then
  log "bucket sync skipped: gsutil not on PATH"
fi

log "cycle done"
