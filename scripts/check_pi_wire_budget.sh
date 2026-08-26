#!/usr/bin/env bash
# check_pi_wire_budget.sh — assert the output budget pi ACTUALLY puts on the wire.
#
# WHY A SCRIPT AND NOT A GO TEST. Every existing guard on this number compares one
# config file to another, and none of them is the wire. That is how a 2x understatement
# sat green in CI for weeks: models.yml declared 65536, pi's own registry declared 65536,
# `TestPiModelsConfigMatchesRegistry` compared exactly those two and passed — while every
# request went out at 32000.
#
# The clamp lives downstream of both files, in pi-ai:
#
#   buildBaseOptions(model, options)            // dist/providers/simple-options.js:4
#     maxTokens: options?.maxTokens
#                ?? (model.maxTokens > 0 ? Math.min(model.maxTokens, 32000) : undefined)
#
# and pi-coding-agent passes no `options.maxTokens` for main agent turns (only compaction
# and branch-summarisation do). EVERY provider routes through buildBaseOptions.
#
# Measured 2026-08-26 by this method: declaring 20000 sent 20000; declaring 65536 sent
# 32000. So the rule is exactly min(declared, 32000).
#
# HOW IT ASSERTS. It makes one real (cent-fraction) pi call, then reads the request back
# from OpenRouter's OWN Broadcast trace in the prod observatory, and compares:
#
#   declared  what pi's registry says   (`pi --list-models`)
#   expected  min(declared, 32000)      (the clamp)
#   actual    max_completion_tokens on the wire
#
# Fails if actual != expected. That catches a raised/removed upstream clamp, a config
# drift, and a models.yml row that has drifted from either.
#
# SCOPE: OpenRouter-routed pi models only — Broadcast is an OpenRouter feature, so local
# ollama lanes produce no trace. They are subject to the same clamp (same code path); it
# simply cannot be confirmed this way.
#
# Bash 3.2 (rig default). No `timeout(1)`.

set -u

MODEL="${1:-openrouter/deepseek/deepseek-v4-flash-0731}"
CLAMP="${PI_WIRE_CLAMP:-32000}"
WAIT_SECS="${PI_WIRE_WAIT_SECS:-90}"
OBS="${AILANG_OBSERVATORY_URL:-https://dashboard.ailang.sunholo.com}"

case "$MODEL" in
  --help|-h)
    echo "usage: check_pi_wire_budget.sh [pi-model-id]"
    echo "  default: openrouter/deepseek/deepseek-v4-flash-0731"
    echo "  env: PI_WIRE_CLAMP (default 32000), PI_WIRE_WAIT_SECS (default 90)"
    exit 0 ;;
esac

command -v pi  >/dev/null 2>&1 || { echo "FAIL: pi not on PATH"; exit 2; }
command -v jq  >/dev/null 2>&1 || { echo "FAIL: jq not on PATH"; exit 2; }
[ -n "${OPENROUTER_API_KEY:-}" ] || {
  echo "FAIL: OPENROUTER_API_KEY unset — source ~/.config/ailang/secrets.env"; exit 2; }

case "$MODEL" in
  openrouter/*) ;;
  *) echo "SKIP: $MODEL is not OpenRouter-routed; Broadcast produces no trace for it."
     echo "      The clamp still applies (same buildBaseOptions path), but this"
     echo "      instrument cannot confirm it. Not a pass — nothing was measured."
     exit 3 ;;
esac

SLUG="${MODEL#openrouter/}"

# --- declared, per pi's own registry -----------------------------------------
# Read the EXACT integer from the config pi actually loads, not from
# `pi --list-models`. That table prints rounded human units ("32K"), so expanding it
# gives 32768 for a declared 32000 — harmless when the clamp dominates, but a FALSE
# FAIL for any budget that is not a round power of two. The display is confirmation,
# never the measurement.
PI_MODELS_JSON="${PI_MODELS_JSON:-$HOME/.pi/agent/models.json}"
if [ -f "$PI_MODELS_JSON" ]; then
  DECLARED=$(jq -r --arg id "$SLUG" '
      [.providers[]?.models[]? | select(.id==$id) | .maxTokens] | first // empty' \
      "$PI_MODELS_JSON" 2>/dev/null)
else
  DECLARED=""
fi
if [ -z "$DECLARED" ]; then
  # Not in our config: pi falls back to its own 16384 default (model-registry.js:457).
  DECLARED=16384
  echo "NOTE: '$SLUG' is not in $PI_MODELS_JSON — pi's 16384 default applies."
fi

# Cross-check against the display so a config/registry mismatch is still visible.
# `pi --list-models` writes its table to STDERR, not stdout (pi 0.73.1) — redirecting
# 2>/dev/null yields an empty table and a bogus "pi does not know this model".
DECLARED_H=$(pi --list-models 2>&1 | awk -v s="$SLUG" '$2==s {print $4; exit}')
[ -n "$DECLARED_H" ] || { echo "FAIL: pi does not know model '$SLUG'"; exit 2; }

EXPECTED="$DECLARED"
[ "$DECLARED" -gt "$CLAMP" ] && EXPECTED="$CLAMP"

# --- watermark BEFORE the probe ----------------------------------------------
# Without this a stale span from an earlier run would be read as this run's result —
# the check would "pass" having measured nothing.
window_start() { date -u -v-2H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ; }
window_end()   { date -u -v+1d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d 'tomorrow'    +%Y-%m-%dT%H:%M:%SZ; }

fetch_spans() {
  curl -s --max-time 45 "$OBS/api/observatory/spans?limit=1000&start_after=$(window_start)&start_before=$(window_end)"
}
newest_pi_ts() {
  jq -r '[.[] | select(.name=="LLM Generation")
          | select((.attributes["span.metadata.openrouter_generation.app.title"]//"")=="pi")
          | .start_time] | max // "none"' 2>/dev/null
}

BEFORE=$(fetch_spans | newest_pi_ts)
echo "pre-probe newest pi span: $BEFORE"

# --- the probe ----------------------------------------------------------------
# stdin MUST be closed: pi waits forever on an open stdin (zero bytes on BOTH stdout
# and stderr is that wedge, not a provider fault).
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
pi --mode json --no-session --no-tools --model "$MODEL" -p 'reply with exactly: ok' \
   < /dev/null > "$TMP/probe.json" 2>"$TMP/probe.err"
PI_RC=$?
if [ "$PI_RC" -ne 0 ] || [ ! -s "$TMP/probe.json" ]; then
  echo "FAIL: pi probe produced no output (rc=$PI_RC). stderr tail:"
  tail -3 "$TMP/probe.err"
  exit 2
fi

# --- read the request back from the provider's own record ---------------------
ACTUAL=""; DEADLINE=$(( $(date +%s) + WAIT_SECS ))
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  sleep 10
  SPANS=$(fetch_spans)
  NEWEST=$(printf '%s' "$SPANS" | newest_pi_ts)
  [ "$NEWEST" = "$BEFORE" ] && continue          # nothing new yet
  ACTUAL=$(printf '%s' "$SPANS" | jq -r --arg slug "$SLUG" '
      [.[] | select(.name=="LLM Generation")
           | select((.attributes["span.metadata.openrouter_generation.app.title"]//"")=="pi")]
      | sort_by(.start_time) | last
      | (.attributes["gen_ai.completion"]|tostring|fromjson? // {})
      | (.rawRequest.max_completion_tokens // .rawRequest.max_tokens // empty)' 2>/dev/null)
  [ -n "$ACTUAL" ] && break
done

if [ -z "$ACTUAL" ]; then
  echo "INCONCLUSIVE: no new pi span ingested within ${WAIT_SECS}s (pre=$BEFORE now=${NEWEST:-?})."
  echo "  The probe itself succeeded, so this is the ingest path, not the lane."
  echo "  An empty window is NOT proof of anything — re-run before drawing a conclusion."
  exit 4
fi

echo "model:    $MODEL"
echo "declared: $DECLARED  (pi --list-models: $DECLARED_H)"
echo "expected: $EXPECTED  (min(declared, clamp=$CLAMP))"
echo "actual:   $ACTUAL    (max_completion_tokens on the wire)"

if [ "$ACTUAL" = "$EXPECTED" ]; then
  echo "PASS: the wire carries the expected budget."
  [ "$DECLARED" -gt "$CLAMP" ] && echo "NOTE: declared ($DECLARED) exceeds the clamp; $CLAMP is what the model actually gets."
  exit 0
fi

echo "FAIL: wire budget is $ACTUAL, expected $EXPECTED."
if [ "$ACTUAL" -gt "$CLAMP" ]; then
  echo "  actual EXCEEDS the clamp — pi-ai may have raised or removed it."
  echo "  If so, the pi rows in internal/eval_harness/models.yml are now UNDER-declared"
  echo "  and should be raised, along with PI_WIRE_CLAMP here and in models_headroom_test.go."
fi
exit 1
