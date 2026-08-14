#!/bin/bash
# test_hook_stdout.sh — a Claude Code hook must not hand its stdout to a background child.
#
# WHY THIS EXISTS (2026-08-14, motoko mission iteration 5, queue item 5a).
# `scripts/hooks/session_start.sh` ended with
#     ailang docs embed-warmup --quiet --timeout 3m &
# commented "(non-blocking)". That is true of the SCRIPT and false of every CONSUMER that captures
# its stdout: a backgrounded child inherits the stdout descriptor, so a `$(...)`-style capture
# cannot observe EOF until the CHILD exits — no matter how promptly the script itself `exit 0`s.
# Claude Code captures hook stdout (that is how the SessionStart banner reaches a session), so the
# hook was effectively held open for as long as the warmup ran, bounded only by the warmup's own
# `--timeout 3m` = 180s. The mission driver caps each model probe at 120s and reports expiry as
#     model <m> probe timed out after 120s — captured output: ''
# — empty, because `claude -p` emits nothing until it completes. Every model in the preference list
# fails identically, so the driver concludes "NO usable model in prefs ... Refusing" and burns the
# fire. Fleet evidence when this was found: v1 47 refusals/186 fires, motoko 6/11, and world — the
# one mission whose checkout has no `.claude/settings.json` and therefore no hooks — 0/89.
# `quota-limited` had never once fired, so the refusals were never about quota.
#
# The test below is BEHAVIOURAL on purpose. A grep for `&` without a redirect would pin the
# spelling; this pins the property, by stubbing a slow `ailang` and asserting that capturing the
# hook's stdout still returns promptly. Deleting the `>/dev/null 2>&1` makes it RED.
set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HOOK="$REPO_ROOT/scripts/hooks/session_start.sh"
FAILED=0
STUB_SLEEP=10        # how long the stubbed warmup blocks
THRESHOLD_MS=5000    # capture must return well inside the stub's runtime

pass() { echo "  ok   — $1"; }
fail() { echo "  FAIL — $1"; FAILED=1; }

echo "hook stdout containment:"

# --- anti-vacuity floor: the enumeration must actually find hooks ----------------------------
# A missing directory greps to zero exactly like a clean one (mission-control rule 3a(i-d)), so
# assert the scope exists and is non-empty rather than reading its emptiness.
if [ ! -d "$REPO_ROOT/scripts/hooks" ]; then
  fail "scripts/hooks/ does not exist — enumerator is broken, not the tree"
  exit 1
fi
HOOK_COUNT=$(find "$REPO_ROOT/scripts/hooks" -name '*.sh' -type f | wc -l | tr -d ' ')
if [ "$HOOK_COUNT" -lt 1 ]; then
  fail "found 0 hook scripts — instrument failure, refusing to report a green"
  exit 1
fi
pass "enumerated $HOOK_COUNT hook script(s)"

if [ ! -f "$HOOK" ]; then
  fail "session_start.sh not found at $HOOK"
  exit 1
fi

# --- behavioural arm --------------------------------------------------------------------------
STUB_DIR=$(mktemp -d -t hookstdout) || { fail "mktemp -d failed"; exit 1; }
trap 'rm -rf "$STUB_DIR"' EXIT

# A stub `ailang` that is instant for everything the hook reads, and SLOW for the warmup — so the
# only thing that can delay the capture is the warmup's hold on stdout.
#
# The stub MUST report at least one unread message. `session_start.sh` has an early `exit 0` on
# the no-unread-messages path that returns BEFORE the warmup line, so a stub answering `[]` never
# reaches the code under test and the timing arm passes for the wrong reason — measured: the first
# draft of this test passed against a deliberately mutated hook (mission-control rule 3i, an
# observable that is not downstream of the mechanism).
cat > "$STUB_DIR/ailang" <<STUB
#!/bin/bash
if [ "\${1:-}" = "docs" ] && [ "\${2:-}" = "embed-warmup" ]; then
  : > "\$HOOK_WARMUP_MARKER"
  sleep $STUB_SLEEP
  exit 0
fi
case "\${1:-}" in
  messages)
    if [ "\${2:-}" = "list" ]; then
      echo '[{"id":"stub-0000","from":"test-harness","title":"stub unread message","created_at":"2026-01-01T00:00:00Z","content":"drives the hook past its no-messages early exit"}]'
    fi
    ;;
  *) : ;;
esac
exit 0
STUB
chmod +x "$STUB_DIR/ailang"

# Control: prove the stub is reachable and really is slow on the warmup path. Without this a
# broken PATH would make the timing arm pass vacuously (mission-control rule 3a).
_c_start=$(date +%s)
PATH="$STUB_DIR:$PATH" ailang docs embed-warmup --quiet >/dev/null 2>&1
_c_elapsed=$(( $(date +%s) - _c_start ))
if [ "$_c_elapsed" -ge 5 ]; then
  pass "control: stubbed warmup is genuinely slow (${_c_elapsed}s) — timing arm is meaningful"
else
  fail "control: stubbed warmup returned in ${_c_elapsed}s — stub not on PATH, timing arm would be vacuous"
  exit 1
fi

# The arm itself: capture the hook's stdout exactly as a hook consumer does. STATE_DIR is pointed
# at a scratch directory so the hook's 3-second duplicate-execution lock (written by whatever real
# session ran last) cannot send us down its early-return path.
export HOOK_WARMUP_MARKER="$STUB_DIR/warmup_reached"
rm -f "$HOOK_WARMUP_MARKER"
_start=$(date +%s)
_captured=$( cd "$REPO_ROOT" && CLAUDE_PROJECT_DIR="$REPO_ROOT" PATH="$STUB_DIR:$PATH" \
             STATE_DIR="$STUB_DIR/state" HOOK_WARMUP_MARKER="$HOOK_WARMUP_MARKER" \
             /bin/bash "$HOOK" 2>/dev/null </dev/null )
_elapsed=$(( $(date +%s) - _start ))

# CONTROL FIRST, and it is the one that matters. `session_start.sh` early-returns on several paths
# (no unread messages, duplicate-execution lock) that never reach the warmup at all. On those paths
# the timing arm below passes instantly whether or not the defect is present — which is exactly how
# the first draft of this test went green against a deliberately mutated hook.
if [ -f "$HOOK_WARMUP_MARKER" ]; then
  pass "control: the hook reached the background-warmup line — timing arm is testing the right code"
else
  fail "control: the hook never reached the background-warmup line (early return) — timing arm is VACUOUS, not green"
  echo "         The stub must drive the hook past its no-unread-messages exit."
  echo "hook stdout containment: FAILED"
  exit 1
fi

if [ "$_elapsed" -lt $(( THRESHOLD_MS / 1000 )) ]; then
  pass "capturing session_start.sh stdout returned in ${_elapsed}s with a ${STUB_SLEEP}s background warmup still running"
else
  fail "capturing session_start.sh stdout took ${_elapsed}s — a background child is holding stdout open."
  echo "         Fix: redirect it, e.g. 'ailang docs embed-warmup ... >/dev/null 2>&1 &'."
  echo "         Do NOT fix this by raising the mission driver's probe timeout."
fi

# The hook must still SAY something — a hook that produces nothing would pass the timing arm
# trivially, which is the vacuous-pass this suite exists to refuse.
if [ -n "$_captured" ]; then
  pass "hook still emits its context banner ($(printf '%s' "$_captured" | wc -c | tr -d ' ') bytes)"
else
  fail "hook produced no stdout — timing arm above is vacuous"
fi

if [ "$FAILED" -eq 0 ]; then
  echo "hook stdout containment: OK"
  exit 0
fi
echo "hook stdout containment: FAILED"
exit 1
