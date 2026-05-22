#!/usr/bin/env bash
# check_ollama_model.sh — Check whether a candidate model tag exists in the
# local Ollama installation, and if not, how big the download would be (via
# the public Ollama library API).
#
# Usage:
#   check_ollama_model.sh <tag>
#
# Example:
#   check_ollama_model.sh qwen3:32b

set -u

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <model-tag>   e.g. qwen3:32b"
  exit 1
fi

TAG="$1"

# Local check
if curl -s http://localhost:11434/api/tags 2>/dev/null | grep -q "\"name\":\"${TAG}\""; then
  echo "✓ ${TAG} is already pulled locally"
  curl -s -X POST http://localhost:11434/api/show \
    -d "{\"name\":\"${TAG}\"}" 2>/dev/null \
    | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
    print("  size_total_bytes :", d.get("size", "(unknown)"))
    info = d.get("model_info", {})
    print("  parameters       :", info.get("general.parameter_count", "(unknown)"))
    print("  quantization     :", d.get("details", {}).get("quantization_level", "(unknown)"))
    print("  context_length   :", info.get("general.context_length", "(unknown)"))
except Exception as e:
    print("  (could not parse show response:", e, ")")
' 2>/dev/null || true
  exit 0
fi

echo "✗ ${TAG} not pulled locally"
echo ""
echo "To pull:"
echo "  ollama pull ${TAG}"
echo ""
echo "Check the Ollama library page for size + capabilities:"
NAME_ONLY="${TAG%%:*}"
echo "  https://ollama.com/library/${NAME_ONLY}"
echo ""
echo "After pulling, add to ~/.config/opencode/opencode.jsonc:"
echo ""
cat <<JSON
  "ollama": {
    "models": {
      "${TAG}": { "name": "${TAG} (local)" }
    }
  }
JSON
echo ""
echo "Then add an entry to internal/eval_harness/models.yml using"
echo "opencode-gemma4-26b as a template (swap api_name and agent_model_name)."
