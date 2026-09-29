#!/usr/bin/env bash
set -eu

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
SCRIPT="$ROOT/tools/launchd/rig-watchdog-reconcile.sh"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
OLLAMA="$TMP/ollama.log"
LEDGER="$TMP/ledger.jsonl"
STATE="$TMP/state"
NOTIFY="$TMP/ailang"
NOTICES="$TMP/notices"

cat > "$NOTIFY" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$RIG_TEST_NOTICES"
SH
chmod +x "$NOTIFY"
export RIG_TEST_NOTICES="$NOTICES"

run_case() {
    RIG_RECONCILE_OLLAMA_LOG="$OLLAMA" RIG_RECONCILE_GATEWAY_LEDGER="$LEDGER" \
    RIG_RECONCILE_STATE_DIR="$STATE" RIG_RECONCILE_AILANG="$NOTIFY" \
    RIG_RECONCILE_MINUTE='2026/09/29 - 06:37|2026-09-29T06:37' \
    RIG_RECONCILE_NOW_EPOCH="${1:-10000}" "$SCRIPT"
}
gin() { printf '[GIN] 2026/09/29 - %s | 200 | POST "%s"\n' "$1" "$2"; }
ledger() { printf '{"ts":"2026-09-29T%s:30Z","path":"%s","status":200}\n' "$1" "$2"; }
reset_case() { rm -rf "$STATE"; : > "$OLLAMA"; : > "$LEDGER"; : > "$NOTICES"; }

# Equal counts, irrelevant endpoints, and in-progress minute are excluded.
reset_case
gin 06:37 /api/chat >> "$OLLAMA"; gin 06:37 /api/tags >> "$OLLAMA"; gin 06:38 /api/chat >> "$OLLAMA"
ledger 06:37 /api/chat >> "$LEDGER"
out=$(run_case); [ -z "$out" ]; [ ! -s "$NOTICES" ]

# Every contracted endpoint participates; ollama excess logs and alerts once.
reset_case
for p in /v1/chat/completions /api/chat /api/generate; do gin 06:37 "$p" >> "$OLLAMA"; done
ledger 06:37 /api/chat >> "$LEDGER"
out=$(run_case 10000); echo "$out" | grep -q 'BYPASS.*ollama=3 gateway=1'; [ "$(wc -l < "$NOTICES")" -eq 1 ]

# A duplicate tick is cursor-suppressed; a later excess inside the hour still logs.
out=$(run_case 10001); [ -z "$out" ]; [ "$(wc -l < "$NOTICES")" -eq 1 ]
rm -f "$STATE/cursor"
out=$(run_case 12000); echo "$out" | grep -q BYPASS; [ "$(wc -l < "$NOTICES")" -eq 1 ]
rm -f "$STATE/cursor"
run_case 13600 >/dev/null; [ "$(wc -l < "$NOTICES")" -eq 2 ]

# Gateway excess is drift, never a bypass alert; equality is not a bypass (> not >=).
reset_case
gin 06:37 /api/generate >> "$OLLAMA"
ledger 06:37 /api/generate >> "$LEDGER"; ledger 06:37 /api/chat >> "$LEDGER"
out=$(run_case); echo "$out" | grep -q 'reconciliation drift'; [ ! -s "$NOTICES" ]
reset_case
gin 06:37 /api/generate >> "$OLLAMA"; ledger 06:37 /api/generate >> "$LEDGER"
out=$(run_case); [ -z "$out" ]; [ ! -s "$NOTICES" ]

# Rotation/truncation is safe because each run scans the named complete minute.
reset_case
gin 06:37 /api/chat > "$OLLAMA"; ledger 06:37 /api/chat > "$LEDGER"
run_case >/dev/null
: > "$OLLAMA"; : > "$LEDGER"; rm -f "$STATE/cursor"
run_case >/dev/null

# Missing input is an observable error and never advances the cursor.
rm -f "$LEDGER"; rm -rf "$STATE"
out=$(run_case); echo "$out" | grep -q 'ledger unreadable'; [ ! -e "$STATE/cursor" ]

echo "rig watchdog bypass fixtures: PASS"
