#!/usr/bin/env bash
# benchmarks/openrouter_cost_compare/run.sh
#
# Cost-comparison benchmark for the M-AI-OPENROUTER milestone.
#
# Runs the same prompt against several OpenRouter-routed models and emits
# a CSV with per-model cost / token / latency, extracted from each call's
# trace ResolvedRoute payload.
#
# Usage:
#   export OPENROUTER_API_KEY=sk-or-...
#   ./benchmarks/openrouter_cost_compare/run.sh > results.csv
#
# This script is INTENTIONALLY NOT WIRED INTO `make ci`. It makes real,
# billable HTTP calls to OpenRouter. When OPENROUTER_API_KEY is unset, it
# exits 0 with a skip message so it is safe to invoke in any context.

set -euo pipefail

PROMPT="Summarize the AILANG language in exactly two sentences."

MODELS=(
  "anthropic/claude-sonnet-4.5"
  "openai/gpt-5-mini"
  "google/gemini-2.5-flash"
  "meta-llama/llama-3.3-70b-instruct"
)

# --- pre-flight ---
if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
  echo "skip: OPENROUTER_API_KEY not set; live OpenRouter benchmarks not run." >&2
  echo "      set OPENROUTER_API_KEY=sk-or-... to run this benchmark." >&2
  exit 0
fi

if ! command -v ailang >/dev/null 2>&1; then
  echo "skip: ailang not found in PATH; run 'make install' or 'make quick-install' first." >&2
  exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "skip: jq not found in PATH; install jq to parse trace payloads." >&2
  exit 0
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Minimal AILANG program that issues a single AI call. The prompt itself
# is passed in via stdin so we don't need to template per-model files.
cat > "$WORKDIR/probe.ail" <<'AILANG_EOF'
module probe

import std/ai (call)
import std/io (println)

export func main() -> () ! {AI, IO} {
  let response = call("Summarize the AILANG language in exactly two sentences.");
  println(response)
}
AILANG_EOF

# CSV header
echo "model,input_tokens,output_tokens,cached_tokens,cost_usd,latency_ms"

total_in=0
total_out=0
total_cached=0
total_cost=0
total_lat=0
total_runs=0

for model in "${MODELS[@]}"; do
  trace_file="$WORKDIR/trace_${model//\//_}.jsonl"

  # Time the call wall-clock; %s%N would be cleaner but isn't portable.
  start_ms=$(python3 -c 'import time; print(int(time.time() * 1000))')

  if ! ailang run \
        --caps IO,AI \
        --ai "openrouter/$model" \
        --emit-trace jsonl \
        --entry main \
        "$WORKDIR/probe.ail" \
        > "$WORKDIR/stdout.txt" \
        2> "$trace_file"; then
    echo "$model,error,error,error,error,error"
    continue
  fi

  end_ms=$(python3 -c 'import time; print(int(time.time() * 1000))')
  latency_ms=$(( end_ms - start_ms ))

  # Extract the AI effect event's route payload from the trace.
  route_json="$(jq -c -r 'select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route // empty' "$trace_file" | head -1)"

  if [[ -z "$route_json" ]]; then
    echo "$model,no-route,no-route,no-route,no-route,$latency_ms"
    continue
  fi

  in_tok=$(jq -r '.prompt_tokens // 0' <<<"$route_json")
  out_tok=$(jq -r '.completion_tokens // 0' <<<"$route_json")
  cached=$(jq -r '.cached_tokens // 0' <<<"$route_json")
  cost=$(jq -r '.cost_usd // "0"' <<<"$route_json")

  echo "$model,$in_tok,$out_tok,$cached,$cost,$latency_ms"

  total_in=$(( total_in + in_tok ))
  total_out=$(( total_out + out_tok ))
  total_cached=$(( total_cached + cached ))
  total_cost=$(python3 -c "print(f'{float('$total_cost') + float('$cost'):.6f}')")
  total_lat=$(( total_lat + latency_ms ))
  total_runs=$(( total_runs + 1 ))
done

if (( total_runs > 0 )); then
  echo "TOTAL,$total_in,$total_out,$total_cached,$total_cost,$total_lat"
fi
