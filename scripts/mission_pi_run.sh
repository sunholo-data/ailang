#!/usr/bin/env bash
# mission_pi_run.sh — run the pi executor lane under guards that can actually see
# the two failure modes we have measured, and emit a TYPED verdict instead of rc=0.
#
# WHY THIS EXISTS (measured 2026-08-26 from OpenRouter Broadcast traces, which are
# the provider's own side of the wire — see docs/docs/guides/debugging.md):
#
#   Every pi executor "silent failure" on record has the same shape on the wire:
#   the model streams ONLY reasoning tokens and never emits content or a tool call.
#   In the whole 08-18..08-22 broadcast corpus, 3 of 173 generations had no
#   finish_reason (cancelled); ALL THREE had `completion: ""` with output_tokens ==
#   reasoning_tokens. The other 170 all carried content or tool_calls. The signature
#   is clean, and it is NOT deepseek-specific — it also fired on z-ai/glm-5.2 under
#   OpenCode, on a different provider host.
#
#   The runs did not fail on their own. WE killed them:
#     * 2026-08-18: 7,130 reasoning tokens / 111s, killed by the old 300 MB NDJSON
#       size ceiling.
#     * 2026-08-19: 1,827 reasoning tokens / 73s, cancelled.
#
#   And the size ceiling was measuring the WRONG THING. pi's `message_update` event
#   carries the WHOLE accumulated message, not a delta (verified first-party in
#   pi 0.73.1, dist/core/agent-session.js:421-427), so NDJSON bytes grow QUADRATICALLY
#   in emitted tokens. 7,130 tokens produced 330 MB. Extrapolated to the model's
#   declared 65,536-token budget that is ~28 GB — i.e. the old ceiling silently
#   capped the lane at roughly 7,000 reasoning tokens, an accidental limit that no
#   prompt change could ever have fixed.
#
# WHAT THIS SCRIPT DOES ABOUT IT
#
#   1. FILTERS `message_update` out of the banked NDJSON. Nothing is lost: `message_end`
#      carries the complete final message including reasoning. Size becomes LINEAR, so
#      the disk-exhaustion hazard goes away without a ceiling that truncates real work.
#   2. Keeps a bounded ROLLING WINDOW of the most recent message_update records in a
#      separate SNAPSHOT file (between 1 and SNAP_EVERY records — the file is truncated
#      every SNAP_EVERY updates and refilled), so a run killed mid-turn still has its
#      newest text/thinking fragments and the cumulative `usage` at bounded cost.
#      HISTORY: this was a single-record snapshot, which held the FULL accumulated
#      message at pi 0.73.1. Since pi 0.84 message_update carries only a ~300-byte
#      DELTA (M-PI-HARNESS-UPGRADE V29), so one record would be forensically empty;
#      the window is what restores the guarantee. The filter in (1) is unchanged and
#      still correct — the quadratic-size hazard it was written for no longer exists
#      at 0.85.1, but the banked file staying free of updates is what makes (3) work.
#   3. Uses the banked file's mtime as a PROGRESS CLOCK. Because updates are filtered,
#      a content-free reasoning turn writes nothing to it — the clock freezes exactly
#      when the failure mode is occurring. This costs nothing and needs no parsing.
#   4. Distinguishes the two stalls, which need different responses:
#        banked frozen + snapshot advancing -> `reasoning_stall` (model thinking, no output)
#        banked frozen + snapshot frozen    -> `stream_dead`     (upstream host hung)
#      `stream_dead` is real and transient: a bare-id deepseek call was measured
#      hanging 90s with HTTP 200 and an empty body on 2026-08-26, while 14/14 retries
#      immediately afterwards succeeded across 6 different provider hosts.
#   5. Asserts the worktree CHANGED relative to a pre-launch content fingerprint, or that
#      commits were made since launch. A tree that is already dirty at launch does not count as work.
#      Of the three assertions the old recipe mandated, this is the only load-bearing one: `stopReason` is now known evadable
#      in BOTH directions — `length` pre-2026-08-13, and a clean `stop` at 625 tokens
#      post-fix — so it can neither confirm nor deny that work happened.
#
# EXIT CODES (the verdict is also written as JSON to --verdict)
#   0  ok               — pi finished, and either the worktree content differs from its pre-launch
#                         fingerprint or commits were made since launch
#   10 empty_worktree   — pi finished, the worktree is byte-identical to its pre-launch state (dirty or
#                         clean), and no commits were made. This is the false green in its pure form.
#   11 reasoning_stall  — killed: reasoning with no content/tool-call past the stall bound
#   12 stream_dead      — killed: no bytes at all past the stall bound
#   13 wall_timeout     — killed: exceeded --max-seconds
#   14 launch_failed    — pi could not start / bad arguments
#   15 sandbox_unavailable — extension or runtime dependency unavailable
#   16 sandbox_policy_invalid — explicit mission policy invalid
#   17 sandbox_not_ready — pi ran without sandbox initialization handshake
#   18 tool_hang        — killed: silent past the stall bound WHILE a tool call was open.
#                          The MODEL is fine; a command it ran never returned. NOT a lane
#                          failure — see hung_tool in the verdict JSON.
#   19 provider_quota   — pi finished, but its LAST assistant message is a provider refusal on
#                          capacity (HTTP 429/402, usage limit, quota, rate limit, credits). pi
#                          exits 0 on these. The MODEL did not fail and the work is truncated
#                          even if files changed. See provider_error / provider_errors in the verdict JSON.
#
# Bash 3.2 (rig default). No `declare -A`, no `${v,,}`, no `timeout(1)`.

set -u

MODEL=""
DIRECTIVE=""
WORKDIR=""
OUT=""
VERDICT=""
MAX_SECONDS="${MISSION_PI_MAX_SECONDS:-1800}"
STALL_SECONDS="${MISSION_PI_STALL_SECONDS:-600}"
POLL_SECONDS="${MISSION_PI_POLL_SECONDS:-10}"
SNAP_EVERY="${MISSION_PI_SNAP_EVERY:-50}"

usage() {
  cat >&2 <<'EOF'
usage: mission_pi_run.sh --model M --directive FILE --workdir DIR --out NDJSON
                         [--verdict JSON] [--max-seconds N] [--stall-seconds N]

  --model         pi model id, e.g. openrouter/deepseek/deepseek-v4-flash-0731
  --directive     file whose contents are delivered to pi on stdin
  --workdir       worktree pi runs in; its git diff is the load-bearing assertion
  --out           path for the FILTERED ndjson (message_update removed)
  --verdict       path for the verdict JSON (default: <out>.verdict.json)
  --max-seconds   wall-clock cap        (default 1800, env MISSION_PI_MAX_SECONDS)
  --stall-seconds no-progress cap       (default 600,  env MISSION_PI_STALL_SECONDS)

Deliberately generous stall default: a legitimate thinking turn on a hard sprint
runs for minutes. This bound exists to catch a turn that never ENDS, not one that
takes a while. The measured failures froze at 111s and 73s.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --model)         MODEL="$2"; shift 2 ;;
    --directive)     DIRECTIVE="$2"; shift 2 ;;
    --workdir)       WORKDIR="$2"; shift 2 ;;
    --out)           OUT="$2"; shift 2 ;;
    --verdict)       VERDICT="$2"; shift 2 ;;
    --max-seconds)   MAX_SECONDS="$2"; shift 2 ;;
    --stall-seconds) STALL_SECONDS="$2"; shift 2 ;;
    -h|--help)       usage; exit 0 ;;
    *) echo "mission_pi_run.sh: unknown argument '$1'" >&2; usage; exit 14 ;;
  esac
done

[ -n "$MODEL" ] && [ -n "$DIRECTIVE" ] && [ -n "$WORKDIR" ] && [ -n "$OUT" ] || {
  echo "mission_pi_run.sh: --model, --directive, --workdir and --out are all required" >&2
  usage; exit 14
}
[ -n "$VERDICT" ]   || VERDICT="${OUT}.verdict.json"

# The run cds into $WORKDIR, so every path we hand the filter must be absolute or the
# output lands somewhere the caller is not looking. Resolve rather than require.
case "$OUT" in /*) ;; *) OUT="$(pwd)/$OUT" ;; esac
case "$VERDICT" in /*) ;; *) VERDICT="$(pwd)/$VERDICT" ;; esac
case "$DIRECTIVE" in /*) ;; *) DIRECTIVE="$(pwd)/$DIRECTIVE" ;; esac
preflight_fail() { # rc verdict error
  jq -n --arg verdict "$2" --arg error "$3" --argjson rc "$1" \
    '{verdict:$verdict, rc:$rc, fenced:false, error:$error, provider_errors:0, provider_error:""}' > "$VERDICT"
  echo "pi lane verdict: $2 (rc=$1): $3" >&2
  exit "$1"
}
[ -f "$DIRECTIVE" ] || preflight_fail 14 launch_failed "directive file not found: $DIRECTIVE"
[ -d "$WORKDIR" ] || preflight_fail 14 launch_failed "workdir not found: $WORKDIR"
WORKDIR=$(cd "$WORKDIR" && pwd -P)

RUNNER_ROOT=$(cd "$(dirname "$0")/.." && pwd -P)
EXT_SRC="$RUNNER_ROOT/tools/pi-extensions/sandbox"
FENCE_SRC="$RUNNER_ROOT/tools/pi-extensions/worktree-fence.ts"
POLICY_SRC="${MISSION_PI_SANDBOX_POLICY_SOURCE:-$EXT_SRC/sandbox.mission.json}"
[ -f "$EXT_SRC/index.ts" ] && [ -f "$EXT_SRC/package.json" ] && [ -f "$FENCE_SRC" ] || \
  preflight_fail 15 sandbox_unavailable "runner sandbox extension source is missing"

# Resolve linked worktree metadata without relying on git's newer --path-format.
GITDIR=$(git -C "$WORKDIR" rev-parse --path-format=absolute --git-dir 2>/dev/null) || \
  GITDIR=$(git -C "$WORKDIR" rev-parse --git-dir 2>/dev/null) || \
  preflight_fail 16 sandbox_policy_invalid "workdir is not a git repository"
COMMON_DIR=$(git -C "$WORKDIR" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || \
  COMMON_DIR=$(git -C "$WORKDIR" rev-parse --git-common-dir 2>/dev/null) || \
  preflight_fail 16 sandbox_policy_invalid "cannot resolve common gitdir"
case "$GITDIR" in /*) ;; *) GITDIR="$WORKDIR/$GITDIR" ;; esac
case "$COMMON_DIR" in /*) ;; *) COMMON_DIR="$WORKDIR/$COMMON_DIR" ;; esac
GITDIR=$(cd "$GITDIR" && pwd -P) || preflight_fail 16 sandbox_policy_invalid "invalid gitdir"
COMMON_DIR=$(cd "$COMMON_DIR" && pwd -P) || preflight_fail 16 sandbox_policy_invalid "invalid common gitdir"
MAIN_ROOT=$(cd "$COMMON_DIR/.." && pwd -P) || preflight_fail 15 sandbox_unavailable "main checkout unavailable"

# The dependency is untracked, so it lives in whichever checkout ran `npm install`. Look in
# the runner's own checkout, then the RUNNER's main checkout (a pin worktree's common gitdir),
# then the WORKDIR's. The runner's main checkout is the one that matters for missions whose
# work repo is not this repo (World's WORKDIR is an ailang-world worktree, with no node_modules).
RUNNER_COMMON=$(git -C "$RUNNER_ROOT" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || RUNNER_COMMON=""
RUNNER_MAIN=""
[ -n "$RUNNER_COMMON" ] && RUNNER_MAIN=$(cd "$RUNNER_COMMON/.." 2>/dev/null && pwd -P)
NODE_MODULES="${MISSION_PI_SANDBOX_NODE_MODULES:-}"
if [ -z "$NODE_MODULES" ]; then
  for candidate in "$EXT_SRC/node_modules" "${RUNNER_MAIN:-/nonexistent}/tools/pi-extensions/sandbox/node_modules" \
                   "$MAIN_ROOT/tools/pi-extensions/sandbox/node_modules"; do
    if [ -f "$candidate/@anthropic-ai/sandbox-runtime/package.json" ]; then NODE_MODULES="$candidate"; break; fi
  done
fi
[ -f "$NODE_MODULES/@anthropic-ai/sandbox-runtime/package.json" ] || \
  preflight_fail 15 sandbox_unavailable "@anthropic-ai/sandbox-runtime is unavailable"

STAGE_PARENT="${MISSION_PI_SANDBOX_STAGE_PARENT:-/tmp}"
case "$STAGE_PARENT/" in "$WORKDIR/"*|"$HOME/"*|"$WORKDIR/"|"$HOME/") \
  preflight_fail 15 sandbox_unavailable "private stage parent cannot be inside workdir or HOME" ;; esac
case "$STAGE_PARENT" in /*) ;; *) \
  preflight_fail 15 sandbox_unavailable "private stage parent must be absolute" ;; esac
[ -d "$STAGE_PARENT" ] || \
  preflight_fail 15 sandbox_unavailable "private stage parent cannot be workdir or HOME"
STAGE=$(mktemp -d "$STAGE_PARENT/mission-pi.XXXXXX") || \
  preflight_fail 15 sandbox_unavailable "cannot create private stage"
mkdir -p "$STAGE/sandbox" || { rm -rf "$STAGE"; preflight_fail 15 sandbox_unavailable "cannot create sandbox stage"; }
cp "$EXT_SRC/index.ts" "$EXT_SRC/package.json" "$STAGE/sandbox/" && \
  cp "$EXT_SRC/mission.ts" "$FENCE_SRC" "$STAGE/" && \
  ln -s "$NODE_MODULES" "$STAGE/sandbox/node_modules" || {
    rm -rf "$STAGE"; preflight_fail 15 sandbox_unavailable "cannot stage sandbox extension"
  }
# index.ts resolves ./mission.ts next to itself.
mv "$STAGE/mission.ts" "$STAGE/sandbox/mission.ts" || {
  rm -rf "$STAGE"; preflight_fail 15 sandbox_unavailable "cannot stage mission policy module"
}

if ! jq -e 'type=="object" and .enabled!=false and
  (.filesystem|type=="object" and (.allowWrite|type=="array" and all(.[];type=="string")) and (.denyWrite|type=="array" and all(.[];type=="string")) and (.denyRead|type=="array" and all(.[];type=="string"))) and
  (.network|type=="object" and (.allowedDomains|type=="array" and all(.[];type=="string")) and (.deniedDomains|type=="array" and all(.[];type=="string")))' "$POLICY_SRC" >/dev/null 2>&1; then
  rm -rf "$STAGE"; preflight_fail 16 sandbox_policy_invalid "canonical mission sandbox policy is missing or invalid"
fi
# Only this worktree's own gitdir and the shared object store. NOT the common refs/heads or
# logs: those hold every branch of the main checkout (dev included), so a fenced executor could
# move any ref. A branch-attached worktree therefore cannot `git commit` under the fence (the
# ref update is denied) — its work is left uncommitted and counted by porcelain, exactly the
# codex lane's contract, and the controller commits it.
if ! jq --arg gitdir "$GITDIR" --arg objects "$COMMON_DIR/objects" \
  '.filesystem.allowWrite += [$gitdir,$objects]' "$POLICY_SRC" > "$STAGE/policy.json"; then
  rm -rf "$STAGE"; preflight_fail 16 sandbox_policy_invalid "cannot generate mission sandbox policy"
fi
READY_FILE="$STAGE/ready"
CLAUDE_TMP="${MISSION_PI_CLAUDE_TMP_DIR:-/tmp/claude}"
mkdir -p "$CLAUDE_TMP" || { rm -rf "$STAGE"; preflight_fail 15 sandbox_unavailable "cannot create sandbox runtime temp directory"; }
export PI_SANDBOX_POLICY_FILE="$STAGE/policy.json" PI_SANDBOX_READY_FILE="$READY_FILE" PI_FENCE_ROOT="$WORKDIR"

SNAP="${OUT}.snapshot.ndjson"
ERR="${OUT}.stderr"
: > "$OUT"; : > "$SNAP"; : > "$ERR"

# mtime in epoch seconds. BSD stat (darwin) first, GNU stat second — the rig is
# darwin but CI legs are not, and a silently-empty stat would freeze the clock and
# make every run look stalled.
mtime_of() {
  stat -f %m "$1" 2>/dev/null || stat -c %Y "$1" 2>/dev/null || echo 0
}
now() { date +%s; }

# Content fingerprint of the worktree relative to its CURRENT HEAD: what porcelain lists,
# the bytes of every tracked change, and the bytes of every untracked file. rc!=0 = no git.
worktree_fingerprint() {
  ( cd "$1" || exit 1
    git rev-parse --git-dir >/dev/null 2>&1 || exit 1
    { git status --porcelain=v1 -z --untracked-files=all
      printf '\0--diff--\0'
      if git rev-parse -q --verify HEAD >/dev/null; then
        git diff --binary --no-ext-diff --no-textconv HEAD
      else   # unborn branch (commits case E): no HEAD to diff against
        git diff --binary --no-ext-diff --no-textconv --cached
        git diff --binary --no-ext-diff --no-textconv
      fi
      printf '\0--untracked--\0'
      git ls-files -o --exclude-standard -z | xargs -0 git hash-object --
    } | git hash-object --stdin )
}

# Deliver the directive on stdin and CLOSE it — pi waits forever on an open stdin,
# which is a wedge the mission loop has hit before.
#
# The awk filter is the whole trick. It must be awk and not a bash read-loop: pi
# emits message_update at ~3 MB/s during a long turn, and a shell loop becomes the
# bottleneck and backpressures the model.
#
# awk's `print > file` truncates on FIRST open and appends until `close(file)`; closing
# every SNAP_EVERY records therefore truncates-and-refills, so the snapshot holds the
# most recent 1..SNAP_EVERY message_update deltas instead of growing without bound.
#
# `set -m` matters and is not cosmetic: without job control a background job shares the
# script's process group, so the `kill -- -PID` below would either fail or — if the pid
# collided with our own pgid — kill this script. With it, the job leads its own group and
# the negative-pid kill reaches pi's children, which is the whole point of killing at all.
BASE_HEAD=$(git -C "$WORKDIR" rev-parse --verify -q HEAD 2>/dev/null) || BASE_HEAD=""
PRE_FP=$(worktree_fingerprint "$WORKDIR") || preflight_fail 14 launch_failed "cannot fingerprint worktree before launch"
PREDIRTY_FILES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
# Each extension once: a worktree carrying .pi/extensions would otherwise load the global
# and repo copies of the same tools, and pi exits rc=1 on the conflict (lib/pi-ext-args.sh).
. "$RUNNER_ROOT/tools/launchd/lib/pi-ext-args.sh"
PI_EXT_ARGS=(); while IFS= read -r _l; do PI_EXT_ARGS+=("$_l"); done < <(_mc_pi_ext_args "$WORKDIR")
set -m
(
  # RUN IN THE WORKTREE. pi edits files relative to its CWD, and --workdir is what we
  # later assert the git diff of. Without this cd the two are different directories:
  # the model does real work somewhere else and the verdict reads empty_worktree — or,
  # worse, reads `ok` off a worktree that was dirty for unrelated reasons. Caught live
  # 2026-08-26 when a run reported 4 tool executions and 0 changed files, and the
  # model's own closing message said it could not find the file and created it.
  cd "$WORKDIR" || exit 14
  # AILANG_MISSION_STAGE=1: an unattended stage. The session-protocol gate then does not
  # demand an `ailang messages` call before unlocking — a call that cannot reach the store
  # from inside this sandbox and hung every executor run of World iter-208 (2026-09-29).
  AILANG_MISSION_STAGE=1 AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
    pi --mode json --no-session ${PI_EXT_ARGS[@]+"${PI_EXT_ARGS[@]}"} -e "$STAGE/sandbox/index.ts" -e "$STAGE/worktree-fence.ts" --model "$MODEL" < "$DIRECTIVE" 2>"$ERR" |
    awk -v out="$OUT" -v snap="$SNAP" -v every="$SNAP_EVERY" '
      /"type":"message_update"/ {
        n++
        if (n % every == 1) { close(snap) }
        print $0 > snap; fflush(snap)
        next
      }
      { print $0 >> out; fflush(out) }
    '
  echo "${PIPESTATUS[0]}" > "$STAGE/pi.rc"
) &
RUNNER_PID=$!

START=$(now)
LAST_OUT_M=$(mtime_of "$OUT")
PROGRESS_AT=$START
# Snapshot mtime AS OF the last progress event. The stall verdict compares against
# this, not against the previous poll: the question is "has the model emitted anything
# at all during THIS stall window", and a per-poll comparison answers a different
# (and much noisier) question — it misses any snapshot cadence slower than the poll.
SNAP_AT_PROGRESS=$(mtime_of "$SNAP")
OUTCOME=""

# Never poll slower than a third of the stall window, or the first observation can
# land past the bound and report a stall that never happened.
if [ "$POLL_SECONDS" -gt $((STALL_SECONDS / 3)) ] && [ $((STALL_SECONDS / 3)) -gt 0 ]; then
  POLL_SECONDS=$((STALL_SECONDS / 3))
fi

while :; do
  sleep "$POLL_SECONDS"

  # Liveness is checked AFTER the sleep and BEFORE the stall arithmetic. The reverse
  # order reports a stall for any run that completes inside one poll interval.
  if ! kill -0 "$RUNNER_PID" 2>/dev/null; then
    OUTCOME="finished"
    break
  fi

  T=$(now)
  OUT_M=$(mtime_of "$OUT")
  SNAP_M=$(mtime_of "$SNAP")

  # Any write to the FILTERED file is progress: a tool call, a turn boundary, a
  # completed message. Reasoning deltas are excluded by construction, which is
  # precisely why this clock detects the failure and a raw byte counter cannot.
  if [ "$OUT_M" != "$LAST_OUT_M" ]; then
    LAST_OUT_M="$OUT_M"
    PROGRESS_AT="$T"
    SNAP_AT_PROGRESS="$SNAP_M"
  fi

  if [ $((T - PROGRESS_AT)) -ge "$STALL_SECONDS" ]; then
    if [ "$SNAP_M" != "$SNAP_AT_PROGRESS" ]; then
      OUTCOME="reasoning_stall"   # thinking hard, emitting nothing usable
    else
      OUTCOME="stream_dead"       # nothing at all is arriving
    fi
    break
  fi

  if [ $((T - START)) -ge "$MAX_SECONDS" ]; then
    OUTCOME="wall_timeout"
    break
  fi
done

if [ "$OUTCOME" != "finished" ]; then
  # Kill the process GROUP: pi spawns children, and killing only the shell leaves
  # the model streaming into a filter nobody reads.
  kill -TERM -"$RUNNER_PID" 2>/dev/null || kill -TERM "$RUNNER_PID" 2>/dev/null
  sleep 2
  kill -KILL -"$RUNNER_PID" 2>/dev/null || kill -KILL "$RUNNER_PID" 2>/dev/null
fi
wait "$RUNNER_PID" 2>/dev/null
PI_RC=$(cat "$STAGE/pi.rc" 2>/dev/null) || PI_RC=14
case "$PI_RC" in ''|*[!0-9]*) PI_RC=14 ;; esac
if [ -f "$READY_FILE" ]; then FENCED=true; else FENCED=false; fi

ELAPSED=$(( $(now) - START ))
DIFF_LINES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
# #1096: porcelain alone is blind to a committing executor. Pre-dirty: judged against PRE_FP, see worktree_fingerprint.
# A failed post-run fingerprint (lost .git) stays "unchanged", matching the old porcelain-0 reading.
WORKTREE_CHANGED=false
POST_FP=$(worktree_fingerprint "$WORKDIR") && [ "$POST_FP" != "$PRE_FP" ] && WORKTREE_CHANGED=true
# A moved HEAD alone is not work: a backward reset or lost .git must still read empty_worktree.
HEAD_AFTER=$(git -C "$WORKDIR" rev-parse --verify -q HEAD 2>/dev/null) || HEAD_AFTER=""
if [ -n "$BASE_HEAD" ]; then
  COMMITS=$(git -C "$WORKDIR" rev-list --count "$BASE_HEAD..HEAD" 2>/dev/null) || COMMITS=0
elif [ -n "$HEAD_AFTER" ]; then
  COMMITS=$(git -C "$WORKDIR" rev-list --count HEAD 2>/dev/null) || COMMITS=0
else
  COMMITS=0
fi
case "$COMMITS" in ''|*[!0-9]*) COMMITS=0 ;; esac
AGENT_END=$(grep -c '"type":"agent_end"' "$OUT" 2>/dev/null | tr -d ' ')
# pi emits tool_execution_START/_UPDATE/_END, never a bare "tool_execution" — an
# exact-match grep on the bare name silently reports 0 on a run that used tools.
# Counting _end (completed calls) is the number a reader actually wants.
TOOL_CALLS=$(grep -c '"type":"tool_execution_end"' "$OUT" 2>/dev/null | tr -d ' ')
OUT_BYTES=$(wc -c < "$OUT" 2>/dev/null | tr -d ' ')

# A silent stream with a tool call still OPEN is not a dead upstream: pi is waiting on
# the command, and no model is involved. Measured 2026-09-29: all three World iter-208
# executor runs (deepseek on openrouter AND ollama) were banked `stream_dead` while each
# sat in `ailang messages list --unread`, which hangs inside this sandbox (its network
# allowlist has no Google endpoints, and the CLI had no deadline). "deepseek is dead"
# was read off those verdicts; the lane never got to answer.
HUNG_TOOL=""
if [ "$OUTCOME" = "stream_dead" ]; then
  _starts=$(grep -c '"type":"tool_execution_start"' "$OUT" 2>/dev/null | tr -d ' ')
  if [ "${_starts:-0}" -gt "${TOOL_CALLS:-0}" ]; then
    HUNG_TOOL=$(grep '"type":"tool_execution_start"' "$OUT" | tail -1 | \
      jq -r '(.toolName // "tool") + ": " + ((.args.command // .args.path // "") | tostring | .[0:200])' 2>/dev/null)
    [ -n "$HUNG_TOOL" ] || HUNG_TOOL="(unparsed tool call)"
    OUTCOME="tool_hang"
  fi
fi
HUNG_TOOL_JSON=$(printf '%s' "$HUNG_TOOL" | jq -Rs .)

# Provider refusals: pi exits 0 and reports them as an assistant message_end with stopReason "error".
# Measured: World iter-187 planner, 4 x "429 … usage limit", banked ok rc 0 with 10 files changed.
# Per-line fromjson? so a truncated line (killed write) cannot zero the count.
PROVIDER_CAPACITY_RE='^\s*(402|429)\b|usage.?limit|quota|rate.?limit|insufficient.?credits|too many requests'
PERR=$(grep '"type":"message_end"' "$OUT" 2>/dev/null | jq -cRn --arg re "$PROVIDER_CAPACITY_RE" '
  [inputs | fromjson? | .message? | select(type == "object" and .role == "assistant")] as $a
  | [$a[] | select(.stopReason == "error")] as $e
  | ($a | last) as $l
  | {n: ($e | length),
     last: ((($e | last) // {}) | .errorMessage // "" | .[0:300]),
     quota: (($l != null) and ($l.stopReason == "error") and (($l.errorMessage // "") | test($re; "i")))}' 2>/dev/null)
PROVIDER_ERRORS=$(printf '%s' "$PERR" | jq -r '.n // 0' 2>/dev/null); case "$PROVIDER_ERRORS" in ''|*[!0-9]*) PROVIDER_ERRORS=0 ;; esac
PROVIDER_ERROR=$(printf '%s' "$PERR" | jq -r '.last // ""' 2>/dev/null)
PROVIDER_QUOTA=$(printf '%s' "$PERR" | jq -r '.quota // false' 2>/dev/null); [ "$PROVIDER_QUOTA" = true ] || PROVIDER_QUOTA=false
_err_lines=$(grep '"type":"message_end"' "$OUT" 2>/dev/null | grep -c '"stopReason":"error"' | tr -d ' ')
if [ "${_err_lines:-0}" -gt 0 ] && [ "$PROVIDER_ERRORS" -eq 0 ]; then   # anti-vacuity
  PROVIDER_ERRORS="$_err_lines"; PROVIDER_ERROR="(unparsed provider error)"; PROVIDER_QUOTA=false
  echo "pi lane verdict: WARNING — $_err_lines message_end line(s) carry stopReason error but none parsed" >&2
fi
PROVIDER_ERROR_JSON=$(printf '%s' "$PROVIDER_ERROR" | jq -Rs .)

case "$OUTCOME" in
  finished)
    if [ "$FENCED" = false ]; then VERDICT_NAME="sandbox_not_ready"; RC=17
    elif [ "$PROVIDER_QUOTA" = true ]; then VERDICT_NAME="provider_quota"; RC=19
    elif [ "$PI_RC" -ne 0 ]; then VERDICT_NAME="launch_failed"; RC=14
    elif [ "$WORKTREE_CHANGED" = true ] || [ "$COMMITS" -gt 0 ]; then VERDICT_NAME="ok"; RC=0
    else VERDICT_NAME="empty_worktree"; RC=10; fi ;;
  reasoning_stall) VERDICT_NAME="reasoning_stall"; RC=11 ;;
  stream_dead)     VERDICT_NAME="stream_dead";     RC=12 ;;
  tool_hang)       VERDICT_NAME="tool_hang";       RC=18 ;;
  wall_timeout)    VERDICT_NAME="wall_timeout";    RC=13 ;;
  *)               VERDICT_NAME="launch_failed";   RC=14 ;;
esac

cat > "$VERDICT" <<EOF
{
  "verdict": "$VERDICT_NAME",
  "rc": $RC,
  "fenced": $FENCED,
  "model": "$MODEL",
  "pi_rc": $PI_RC,
  "elapsed_seconds": $ELAPSED,
  "worktree_changed_files": ${DIFF_LINES:-0},
  "predirty_files": ${PREDIRTY_FILES:-0},
  "worktree_changed_since_start": $WORKTREE_CHANGED,
  "base_head": "$BASE_HEAD",
  "head_after": "$HEAD_AFTER",
  "commits_since_start": $COMMITS,
  "tool_executions": ${TOOL_CALLS:-0},
  "hung_tool": $HUNG_TOOL_JSON,
  "provider_errors": ${PROVIDER_ERRORS:-0},
  "provider_error": $PROVIDER_ERROR_JSON,
  "agent_end_events": ${AGENT_END:-0},
  "ndjson_bytes_filtered": ${OUT_BYTES:-0},
  "ndjson": "$OUT",
  "snapshot": "$SNAP",
  "stderr": "$ERR"
}
EOF
rm -rf "$STAGE"

echo "pi lane verdict: $VERDICT_NAME (rc=$RC) after ${ELAPSED}s — ${DIFF_LINES:-0} changed files, ${PREDIRTY_FILES:-0} pre-dirty, changed since start: $WORKTREE_CHANGED, ${TOOL_CALLS:-0} tool executions, $COMMITS commits" >&2
[ -n "$HUNG_TOOL" ] && echo "pi lane verdict: the MODEL did not stall — pi was waiting on a tool call that never returned: $HUNG_TOOL" >&2
[ "$RC" -eq 19 ] && echo "pi lane verdict: the PROVIDER refused on capacity — park the lane, not the model: $PROVIDER_ERROR" >&2
[ "$RC" -ne 19 ] && [ "$PROVIDER_ERRORS" -gt 0 ] && echo "pi lane verdict: $PROVIDER_ERRORS provider error(s) during the run; last: $PROVIDER_ERROR" >&2
exit "$RC"
