#!/usr/bin/env bash
# Reconcile the last complete minute of ollama requests with rig-gate's ledger.
# All paths and clocks are injectable so fixtures never touch live rig state.
set -u

OLLAMA_LOG=${RIG_RECONCILE_OLLAMA_LOG:-/tmp/ollama-serve-launchd.log}
GATEWAY_LEDGER=${RIG_RECONCILE_GATEWAY_LEDGER:-/Users/Shared/ailang/rig-gate.jsonl}
STATE_DIR=${RIG_RECONCILE_STATE_DIR:-$HOME/.ailang/state/rig-watchdog-reconcile}
AILANG_BIN=${RIG_RECONCILE_AILANG:-ailang}
NOW_EPOCH=${RIG_RECONCILE_NOW_EPOCH:-$(date +%s)}

mkdir -p "$STATE_DIR" 2>/dev/null || { echo "[WATCHDOG] bypass detector error: cannot create $STATE_DIR"; exit 0; }
LOCK_DIR="$STATE_DIR/run.lock"
mkdir "$LOCK_DIR" 2>/dev/null || exit 0
trap 'rmdir "$LOCK_DIR" 2>/dev/null || true' EXIT

if [ ! -r "$OLLAMA_LOG" ]; then
    echo "[WATCHDOG] bypass detector error: ollama log unreadable: $OLLAMA_LOG"
    exit 0
fi
if [ ! -r "$GATEWAY_LEDGER" ]; then
    echo "[WATCHDOG] bypass detector error: gateway ledger unreadable: $GATEWAY_LEDGER"
    exit 0
fi

if [ -n "${RIG_RECONCILE_MINUTE:-}" ]; then
    GIN_MINUTE=${RIG_RECONCILE_MINUTE%%|*}
    JSON_MINUTE=${RIG_RECONCILE_MINUTE#*|}
else
    TARGET_EPOCH=$((NOW_EPOCH - 60))
    if date -u -r "$TARGET_EPOCH" "+%Y/%m/%d - %H:%M|%Y-%m-%dT%H:%M" >/dev/null 2>&1; then
        MINUTES=$(date -u -r "$TARGET_EPOCH" "+%Y/%m/%d - %H:%M|%Y-%m-%dT%H:%M")
    else
        MINUTES=$(date -u -d "@$TARGET_EPOCH" "+%Y/%m/%d - %H:%M|%Y-%m-%dT%H:%M")
    fi
    GIN_MINUTE=${MINUTES%%|*}
    JSON_MINUTE=${MINUTES#*|}
fi

CURSOR_FILE="$STATE_DIR/cursor"
if [ "$(cat "$CURSOR_FILE" 2>/dev/null)" = "$JSON_MINUTE" ]; then
    exit 0
fi

ENDPOINT_RE='(/v1/chat/completions|/api/chat|/api/generate)'
OLLAMA_COUNT=$(awk -v minute="$GIN_MINUTE" -v endpoints="$ENDPOINT_RE" '$0 ~ minute && $0 ~ endpoints {n++} END {print n+0}' "$OLLAMA_LOG")
GATEWAY_COUNT=$(awk -v minute="$JSON_MINUTE" -v endpoints="$ENDPOINT_RE" '$0 ~ ("\"ts\":\"" minute) && $0 ~ ("\"path\":\"" endpoints "\"") {n++} END {print n+0}' "$GATEWAY_LEDGER")

if [ "$OLLAMA_COUNT" -gt "$GATEWAY_COUNT" ]; then
    echo "[WATCHDOG] GPU GATEWAY BYPASS minute=$JSON_MINUTE ollama=$OLLAMA_COUNT gateway=$GATEWAY_COUNT"
    LAST_ALERT_FILE="$STATE_DIR/last-alert"
    LAST_ALERT=$(cat "$LAST_ALERT_FILE" 2>/dev/null || echo 0)
    case "$LAST_ALERT" in ''|*[!0-9]*) LAST_ALERT=0;; esac
    if [ $((NOW_EPOCH - LAST_ALERT)) -ge 3600 ]; then
        BODY="GPU gateway bypass detected for $JSON_MINUTE: ollama=$OLLAMA_COUNT gateway=$GATEWAY_COUNT"
        "$AILANG_BIN" messages send controlplane "$BODY" --title "[rig] GPU gateway bypass" --from rig-watchdog >/dev/null 2>&1 || \
            echo "[WATCHDOG] bypass detector error: controlplane notification failed"
        tmp="$LAST_ALERT_FILE.tmp.$$"
        printf '%s\n' "$NOW_EPOCH" > "$tmp" && mv "$tmp" "$LAST_ALERT_FILE"
    fi
elif [ "$GATEWAY_COUNT" -gt "$OLLAMA_COUNT" ]; then
    echo "[WATCHDOG] gateway reconciliation drift minute=$JSON_MINUTE ollama=$OLLAMA_COUNT gateway=$GATEWAY_COUNT"
fi

tmp="$CURSOR_FILE.tmp.$$"
printf '%s\n' "$JSON_MINUTE" > "$tmp" && mv "$tmp" "$CURSOR_FILE"
exit 0
