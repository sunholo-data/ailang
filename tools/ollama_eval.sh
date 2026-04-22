#!/usr/bin/env bash
# ollama_eval.sh — run ailang eval-suite with Ollama model warmup/teardown.
#
# Pins the model in RAM for the eval run (no cold starts between sequential
# runs) and unloads it immediately when done (frees RAM for normal laptop use).
#
# Usage:
#   ./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci
#   ./tools/ollama_eval.sh --models opencode-gemma4-26b --benchmarks fizzbuzz --agent-timeout 300
#
# All extra flags are forwarded to ailang eval-suite --agent.
#
# Environment:
#   OLLAMA_EVAL_KEEPALIVE  keepalive duration while eval runs (default: 60m)
#   OLLAMA_HOST            Ollama API base URL (default: http://localhost:11434)

set -euo pipefail

OLLAMA_EVAL_KEEPALIVE="${OLLAMA_EVAL_KEEPALIVE:-60m}"
OLLAMA_API="${OLLAMA_HOST:-http://localhost:11434}"
MODELS_YML="internal/eval_harness/models.yml"

# --- helpers ---

ollama_warmup() {
    local ollama_model="$1"
    echo "→ Warming up: $ollama_model (keepalive=${OLLAMA_EVAL_KEEPALIVE})"
    if curl -sf --max-time 300 "${OLLAMA_API}/api/generate" \
        -d "{\"model\":\"${ollama_model}\",\"prompt\":\"hi\",\"stream\":false,\"keep_alive\":\"${OLLAMA_EVAL_KEEPALIVE}\"}" \
        > /dev/null 2>&1; then
        echo "  ✓ $ollama_model loaded and pinned"
    else
        echo "  ⚠ Could not warm up $ollama_model — is Ollama running?"
        echo "    Start with: open -a Ollama  (or: ollama serve)"
        exit 1
    fi
}

ollama_unload() {
    local ollama_model="$1"
    echo "→ Unloading: $ollama_model"
    curl -sf --max-time 10 "${OLLAMA_API}/api/generate" \
        -d "{\"model\":\"${ollama_model}\",\"keep_alive\":\"0\"}" \
        > /dev/null 2>&1 \
        && echo "  ✓ $ollama_model unloaded (RAM freed)" \
        || echo "  ⚠ Unload failed (model may already be idle)"
}

# Extract ollama model name from models.yml for a single model ID.
# Returns empty string if not an Ollama-backed model.
get_ollama_model_name() {
    local model_id="$1"
    [[ -f "$MODELS_YML" ]] || return 0
    awk "/^  ${model_id}:/{found=1} found && /agent_model_name:/{print; exit}" "$MODELS_YML" \
        | grep -oE '"ollama/[^"]+"' | tr -d '"' | sed 's|^ollama/||'
}

# --- parse --models from the forwarded args ---

MODELS_VAL=""
for i in "$@"; do
    if [[ "${PREV:-}" == "--models" ]]; then
        MODELS_VAL="$i"
    fi
    PREV="$i"
done

if [[ -z "$MODELS_VAL" ]]; then
    echo "Usage: $0 --models <model> [eval-suite flags...]" >&2
    echo "Example: $0 --models opencode-gemma4-e4b --benchmarks fizzbuzz" >&2
    exit 1
fi

# --- collect Ollama models to warm up ---

OLLAMA_MODELS=()
IFS=',' read -ra MODEL_IDS <<< "$MODELS_VAL"
for id in "${MODEL_IDS[@]}"; do
    name=$(get_ollama_model_name "$id")
    [[ -n "$name" ]] && OLLAMA_MODELS+=("$name")
done

if [[ ${#OLLAMA_MODELS[@]} -eq 0 ]]; then
    echo "No Ollama-backed models in: $MODELS_VAL"
    echo "Running eval-suite directly..."
    exec ailang eval-suite --agent "$@"
fi

# --- warmup ---

echo "=== Ollama Eval Setup ==="
for m in "${OLLAMA_MODELS[@]}"; do
    ollama_warmup "$m"
done
echo ""

# Register teardown on EXIT (normal finish, Ctrl-C, or error)
teardown() {
    echo ""
    echo "=== Ollama Eval Teardown ==="
    for m in "${OLLAMA_MODELS[@]}"; do
        ollama_unload "$m"
    done
}
trap teardown EXIT

# --- run ---

echo "=== Running eval-suite ==="
ailang eval-suite --agent \
    --agent-parallel 1 \
    --parallel 0 \
    --agent-timeout 300 \
    "$@"
