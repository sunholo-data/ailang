#!/usr/bin/env bash
# probe_context_curve.sh — measure ollama prefill latency vs prompt size.
#
# Hypothesis (2026-05-23): the agent slowdown in long opencode sessions is
# caused by linear prefill-cost-vs-context growth, NOT by the framework
# prefix being uncached. Each turn re-sends the entire accumulated history,
# and prefill latency on gemma4:26b grows linearly with prompt size.
#
# This probe isolates the question: how does ollama prefill latency scale
# with prompt size, on this rig, for this model? Hits /api/generate directly
# (NOT /v1/chat/completions which has chat history overhead).
#
# Output: CSV-formatted timing for prompts of 1k / 5k / 10k / 25k / 50k /
# 100k / 150k / 200k input tokens. Then plot the curve and decide what
# "max tokens per session" is operationally viable.
#
# Usage:
#   probe_context_curve.sh                  # default: gemma4:26b
#   probe_context_curve.sh qwen3:32b        # any pulled model

set -uo pipefail

MODEL="${1:-gemma4:26b}"

# Verify ollama is up
if ! curl -sf -m 3 http://localhost:11434/api/tags >/dev/null; then
    echo "ollama not responding on :11434" >&2
    exit 1
fi

# Build padding: ~1 token per word (rough). "The quick brown fox..." = 9 words ~ 9 tokens
# Pad in 1k chunks so we can scale precisely.
PAD_1K=""
for _ in $(seq 1 110); do
    PAD_1K+="The quick brown fox jumps over the lazy dog. "
done
# PAD_1K is now ~1000 words ≈ 1k tokens

# Probe sizes (tokens, approximate)
SIZES=(1000 5000 10000 25000 50000 100000 150000 200000)

# CSV header
echo "size_tokens,load_ms,prompt_eval_count,prompt_eval_ms,prompt_tok_per_s,eval_count,eval_ms,total_wall_s,response_first_60"

for SIZE in "${SIZES[@]}"; do
    # Build a prompt of approximately SIZE tokens
    REPEAT=$(( SIZE / 1000 ))
    PROMPT=""
    for _ in $(seq 1 $REPEAT); do
        PROMPT+="$PAD_1K"
    done
    PROMPT+=" Now answer: what's 2+2? One number only."

    # Send and capture timing. num_predict=5 keeps generation small so we measure prefill dominantly.
    BODY=$(jq -nc --arg model "$MODEL" --arg prompt "$PROMPT" \
      '{model: $model, prompt: $prompt, stream: false, options: {num_predict: 5}}')

    START=$(date +%s%N)
    RESP=$(curl -s -m 1800 -X POST http://localhost:11434/api/generate -d "$BODY")
    END=$(date +%s%N)
    WALL_S=$(awk "BEGIN{printf \"%.1f\", ($END - $START)/1e9}")

    LOAD=$(echo "$RESP" | jq -r '.load_duration // 0' | awk '{printf "%.0f", $1/1e6}')
    P_COUNT=$(echo "$RESP" | jq -r '.prompt_eval_count // 0')
    P_MS=$(echo "$RESP" | jq -r '.prompt_eval_duration // 0' | awk '{printf "%.0f", $1/1e6}')
    E_COUNT=$(echo "$RESP" | jq -r '.eval_count // 0')
    E_MS=$(echo "$RESP" | jq -r '.eval_duration // 0' | awk '{printf "%.0f", $1/1e6}')
    P_TPS=$(awk "BEGIN{if ($P_MS>0) printf \"%.0f\", $P_COUNT*1000/$P_MS; else printf \"0\"}")
    RESP_HEAD=$(echo "$RESP" | jq -r '.response // ""' | head -c 60 | tr '\n' ' ')

    echo "${SIZE},${LOAD},${P_COUNT},${P_MS},${P_TPS},${E_COUNT},${E_MS},${WALL_S},\"${RESP_HEAD}\""
done
