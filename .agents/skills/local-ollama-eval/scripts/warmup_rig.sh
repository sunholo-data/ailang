#!/usr/bin/env bash
# warmup_rig.sh — pre-warm the local-Ollama eval rig before kicking off a rotation.
#
# Two layers get warmed:
#
#   1. MCP server (mcp.ailang.sunholo.com)
#      Cloud Run-backed, ~1-5s cold-start latency on first hit per idle window.
#      Sending a single `initialize` request brings the instance up; subsequent
#      MCP calls during the rotation hit it warm (~90ms). Without this, the
#      first benchmark in a rotation pays an extra 1-5s on every MCP tool call
#      until the instance is up.
#
#   2. Model + KV-cache prefix
#      ollama loads gemma4:26b (~17 GB weights) and caches the ~100k-token
#      opencode framework prompt in its KV cache. Empirically observed
#      2026-05-23: the first opencode invocation paid 42-92s for this; every
#      subsequent invocation hit warm prefix cache. By doing one throwaway
#      "say hello" call up front, that 42-92s cost is amortized once for the
#      whole rotation rather than re-paid per trial.
#
# Idempotent: safe to run multiple times. The MCP ping is ~100ms. The model
# warmup is fast (~30-60s) when the model is already loaded; longer if cold.
#
# Usage:
#   warmup_rig.sh [provider/model]    # default: ollama/gemma4:26b
#
# Exit codes:
#   0  — both warmups succeeded
#   1  — MCP unreachable (rotation can still proceed; agent falls back to CLI)
#   2  — opencode warmup failed (rotation will hit cold prefill on first trial)

set -uo pipefail

MODEL="${1:-ollama/gemma4:26b}"
MCP_URL="${AILANG_MCP_URL:-https://mcp.ailang.sunholo.com/mcp/}"

echo "Warming up rig for model=$MODEL ..."

# ─── 1. MCP server (Cloud Run wake) ────────────────────────────────────────
echo -n "  MCP server ($MCP_URL): "
START_MS=$(perl -MTime::HiRes=time -e 'printf "%.0f", time*1000')
if curl -sf -m 10 "$MCP_URL" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json, text/event-stream" \
    -H "MCP-Protocol-Version: 2024-11-05" \
    -X POST \
    -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"ailang-warmup","version":"0.1"}}}' \
    > /dev/null 2>&1; then
    ELAPSED_MS=$(( $(perl -MTime::HiRes=time -e 'printf "%.0f", time*1000') - START_MS ))
    echo "ok (${ELAPSED_MS}ms)"
    MCP_OK=1
else
    echo "FAIL (continuing — agent will fall back to CLI)"
    MCP_OK=0
fi

# ─── 2. Model + KV-cache prefix warmup ─────────────────────────────────────
# Send one throwaway opencode invocation. The exact prompt content doesn't
# matter for warmup — what matters is that the full opencode framework prompt
# (~100k tokens) gets prefilled into ollama's KV cache. Subsequent calls then
# hit the cache and pay only the marginal token cost.

WORKSPACE=$(mktemp -d -t ailang_warmup_XXXX)
trap "rm -rf $WORKSPACE" EXIT

echo -n "  Model + framework prefix: "
START_S=$(date +%s)

if ! command -v opencode >/dev/null 2>&1; then
    echo "FAIL (opencode not on PATH)"
    exit 2
fi

# Cap warmup at 5 minutes. If it takes longer, something is wrong with
# the rig and we shouldn't keep waiting.
( cd "$WORKSPACE" && \
  opencode run --format json --dangerously-skip-permissions \
    --model "$MODEL" \
    "Say 'hello' once and stop. No code." \
    > warmup.out 2> warmup.err ) &
OPENCODE_PID=$!

for i in $(seq 1 300); do
    if ! kill -0 $OPENCODE_PID 2>/dev/null; then break; fi
    sleep 1
done

if kill -0 $OPENCODE_PID 2>/dev/null; then
    kill -9 $OPENCODE_PID 2>/dev/null
    wait $OPENCODE_PID 2>/dev/null
    WALL=$(( $(date +%s) - START_S ))
    echo "TIMEOUT after ${WALL}s (cap was 300s; model may not be loaded — check ollama state)"
    exit 2
fi

wait $OPENCODE_PID
RC=$?
WALL=$(( $(date +%s) - START_S ))

if [ $RC -ne 0 ]; then
    echo "FAIL (exit=$RC, ${WALL}s) — check $WORKSPACE/warmup.err"
    head -10 "$WORKSPACE/warmup.err" 2>/dev/null | sed 's/^/    /'
    exit 2
fi
echo "ok (${WALL}s)"

# ─── Summary ───────────────────────────────────────────────────────────────
echo ""
echo "Rig is warm:"
if [ "$MCP_OK" -eq 1 ]; then
    echo "  ✓ MCP responsive"
else
    echo "  ⚠ MCP unreachable — agent will fall back to CLI discovery"
fi
echo "  ✓ Model loaded, framework-prefix KV cache populated"
echo ""
echo "Subsequent benchmark trials should hit warm cache from their first call."

exit 0
