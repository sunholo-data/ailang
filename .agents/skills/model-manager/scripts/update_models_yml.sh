#!/usr/bin/env bash
# Add a new model to models.yml configuration
# Usage: update_models_yml.sh <friendly-name> <api-name> <provider> <input-price> <output-price> [description]

set -euo pipefail

if [ $# -lt 5 ]; then
    echo "Usage: $0 <friendly-name> <api-name> <provider> <input-per-1k> <output-per-1k> [description]"
    echo ""
    echo "Arguments:"
    echo "  friendly-name : Short name for the model (e.g., gpt5-1)"
    echo "  api-name      : Exact API model name (e.g., gpt-5.1)"
    echo "  provider      : openai, anthropic, or google"
    echo "  input-per-1k  : Input price per 1K tokens (e.g., 0.00125)"
    echo "  output-per-1k : Output price per 1K tokens (e.g., 0.01)"
    echo "  description   : Optional description"
    echo ""
    echo "Examples:"
    echo "  $0 gpt5-1 'gpt-5.1' openai 0.00125 0.01 'GPT-5.1 with adaptive reasoning'"
    echo "  $0 gemini-3-pro 'gemini-3-pro-preview-11-2025' google 0.002 0.012"
    exit 1
fi

FRIENDLY_NAME="$1"
API_NAME="$2"
PROVIDER="$3"
INPUT_PRICE="$4"
OUTPUT_PRICE="$5"
DESCRIPTION="${6:-$FRIENDLY_NAME model}"

MODELS_YML="internal/eval_harness/models.yml"

if [ ! -f "$MODELS_YML" ]; then
    echo "✗ models.yml not found at: $MODELS_YML"
    echo "Are you in the AILANG project root?"
    exit 1
fi

echo "Adding model to models.yml:"
echo "  Friendly name: $FRIENDLY_NAME"
echo "  API name: $API_NAME"
echo "  Provider: $PROVIDER"
echo "  Pricing: \$$INPUT_PRICE / \$$OUTPUT_PRICE per 1K tokens"
echo "  Description: $DESCRIPTION"
echo ""

# Create backup
cp "$MODELS_YML" "$MODELS_YML.backup"
echo "✓ Created backup: $MODELS_YML.backup"

# Determine env var based on provider
case "$PROVIDER" in
    openai)
        ENV_VAR="OPENAI_API_KEY"
        ;;
    anthropic)
        ENV_VAR="ANTHROPIC_API_KEY"
        ;;
    google)
        ENV_VAR="GOOGLE_API_KEY"
        ;;
    *)
        echo "✗ Unknown provider: $PROVIDER"
        echo "Supported: openai, anthropic, google"
        exit 1
        ;;
esac

# Create model entry
# Note: This is a simple append - for production, you'd want YAML-aware editing
MODEL_ENTRY="
  # $DESCRIPTION
  $FRIENDLY_NAME:
    api_name: \"$API_NAME\"
    provider: \"$PROVIDER\"
    description: \"$DESCRIPTION\"
    env_var: \"$ENV_VAR\"
    agent_cli: null  # Agent CLI support not yet implemented
    pricing:
      input_per_1k: $INPUT_PRICE
      output_per_1k: $OUTPUT_PRICE
    notes: |
      Added: $(date +%Y-%m-%d)
      Test before adding to production suites.
"

# Check if model already exists
if grep -q "^  $FRIENDLY_NAME:" "$MODELS_YML"; then
    echo "⚠ Model '$FRIENDLY_NAME' already exists in models.yml"
    echo ""
    read -p "Overwrite? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        rm "$MODELS_YML.backup"
        exit 1
    fi

    # Remove existing entry (simple approach - stops at next model)
    sed -i.tmp "/^  $FRIENDLY_NAME:/,/^  [a-z]/{ /^  $FRIENDLY_NAME:/d; /^  [a-z]/!d; }" "$MODELS_YML"
    rm "$MODELS_YML.tmp"
fi

# Find insertion point (after last model, before "# Default model")
# This is a simple approach - insert before the "default:" line
if grep -q "^# Default model for benchmarks" "$MODELS_YML"; then
    # Insert before "# Default model"
    sed -i.tmp "/^# Default model for benchmarks/i\\
$MODEL_ENTRY
" "$MODELS_YML"
    rm "$MODELS_YML.tmp"
else
    # Append to end of models section
    echo "$MODEL_ENTRY" >> "$MODELS_YML"
fi

echo "✓ Updated models.yml"

# Validate YAML syntax (if python3 available)
if command -v python3 &> /dev/null; then
    if python3 -c "import yaml; yaml.safe_load(open('$MODELS_YML'))" 2>/dev/null; then
        echo "✓ YAML syntax valid"
    else
        echo "✗ YAML syntax error!"
        echo "Restoring backup..."
        mv "$MODELS_YML.backup" "$MODELS_YML"
        exit 1
    fi
else
    echo "⚠ python3 not available, skipping YAML validation"
fi

rm "$MODELS_YML.backup"

echo ""
echo "✓ Ready to test"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff $MODELS_YML"
echo "  2. Test model: .claude/skills/model-manager/scripts/run_test_benchmark.sh $FRIENDLY_NAME"
echo "  3. Add to suites (dev_models, extended_suite, benchmark_suite) if needed"
