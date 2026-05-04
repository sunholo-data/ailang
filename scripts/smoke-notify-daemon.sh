#!/usr/bin/env bash
# smoke-notify-daemon.sh - End-to-end smoke test for the notify daemon
#
# Publishes a synthetic TaskCompletion event to the dev environment's events
# topic, runs `ailang daemon` with --dry-run, and asserts the daemon logged a
# notification within 30s.
#
# Usage:
#   ./scripts/smoke-notify-daemon.sh
#
# Exits non-zero on any verification failure. Designed to be a release gate
# for M-MAC-NOTIFY-DAEMON before bumping the daemon to "stable" in v0.15.0
# and forward.

set -euo pipefail

PROJECT="ailang-multivac-dev"
TOPIC="ailang-dev-events"
ENV="dev"
TIMEOUT_SECONDS=30
LOG_FILE="$(mktemp -t ailang-daemon-smoke.XXXXXX)"

# Build a unique task_id so dedup doesn't suppress repeated smoke runs.
TASK_ID="smoke-$(date +%s)-$$"

# Locate the ailang binary.
AILANG_BIN="${AILANG_BIN:-./bin/ailang}"
if [ ! -x "$AILANG_BIN" ]; then
    AILANG_BIN="$(command -v ailang 2>/dev/null || true)"
fi
if [ -z "$AILANG_BIN" ] || [ ! -x "$AILANG_BIN" ]; then
    echo "✗ ailang binary not found (set AILANG_BIN or run from repo root after make build)"
    exit 1
fi

# Verify gcloud is set up.
if ! command -v gcloud >/dev/null 2>&1; then
    echo "✗ gcloud not on PATH"
    exit 1
fi

echo "→ starting daemon (env=$ENV, --dry-run)..."
"$AILANG_BIN" daemon run --env "$ENV" --dry-run > "$LOG_FILE" 2>&1 &
DAEMON_PID=$!
trap "kill -INT $DAEMON_PID 2>/dev/null || true; rm -f $LOG_FILE" EXIT

# Give the daemon a moment to subscribe.
sleep 2

if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
    echo "✗ daemon exited prematurely:"
    cat "$LOG_FILE"
    exit 1
fi

PAYLOAD="{\"task_id\":\"$TASK_ID\",\"agent_id\":\"smoke-test\",\"status\":\"completed\",\"num_turns\":1,\"cost_usd\":0.0001}"

echo "→ publishing synthetic TaskCompletion to $TOPIC (task_id=$TASK_ID)..."
gcloud pubsub topics publish "$TOPIC" --project="$PROJECT" --message="$PAYLOAD" >/dev/null

echo "→ waiting up to ${TIMEOUT_SECONDS}s for notification..."
ELAPSED=0
while [ "$ELAPSED" -lt "$TIMEOUT_SECONDS" ]; do
    if grep -q "$TASK_ID" "$LOG_FILE"; then
        echo "✅ notification fired within ${ELAPSED}s"
        echo "  log: $(grep "$TASK_ID" "$LOG_FILE" | head -1)"
        exit 0
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

echo "✗ no notification observed within ${TIMEOUT_SECONDS}s"
echo "  daemon log:"
sed 's/^/    /' "$LOG_FILE"
exit 1
