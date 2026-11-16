#!/bin/bash
# coordinator_hook.sh - Forward a Claude Code hook event to the AILANG coordinator.
#
# Replaces the native `type: "http"` hook entries in settings.json. That hook type
# surfaces ECONNREFUSED as a "Stop hook error" whenever the coordinator daemon is
# offline, even though the session is not instrumented. This wrapper swallows
# connection failures so local sessions stay quiet.
#
# Reads JSON payload from stdin; always exits 0.

PAYLOAD=$(cat)
URL="${1:-http://127.0.0.1:1957/api/hooks/claude}"

# /api/hooks/claude is the dashboard's hub ingest class
# (internal/server/ingest_auth.go). The default URL is loopback (exempt), but
# $1 can point anywhere: present AILANG_HUB_TOKEN when set, nothing when unset.
AUTH_HEADER=()
if [ -n "${AILANG_HUB_TOKEN:-}" ]; then
  AUTH_HEADER=(-H "X-AILANG-Ingest-Token: ${AILANG_HUB_TOKEN}")
fi

# --connect-timeout 1: fast-fail when the daemon isn't listening (portable: GNU + BSD curl).
# --max-time 3: cap total latency when it is listening but slow.
# || true + redirects: never surface errors to Claude Code.
curl -sS --connect-timeout 1 --max-time 3 -X POST "$URL" \
  -H "Content-Type: application/json" ${AUTH_HEADER[@]+"${AUTH_HEADER[@]}"} \
  -H "X-Ailang-Task-Id: ${AILANG_TASK_ID:-}" \
  -H "X-Ailang-Chain-Id: ${AILANG_CHAIN_ID:-}" \
  -H "X-Ailang-Stage-Id: ${AILANG_STAGE_ID:-}" \
  -H "X-Ailang-Message-Id: ${AILANG_MESSAGE_ID:-}" \
  --data-binary "$PAYLOAD" >/dev/null 2>&1 || true

exit 0
