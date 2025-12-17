#!/bin/bash
# Benchmark Ollama embedding models for AILANG CLI
# Usage: ./tools/benchmark_embeddings.sh
#
# Tests actual `ailang docs search --neural` command with design_docs corpus
# Measures end-to-end performance including cache operations

set -e

CORPUS="design_docs/planned"
QUERY="parser error handling"
LIMIT=3

echo "=== AILANG Embedding Model Benchmark ==="
echo "Corpus: $CORPUS"
echo "Query: \"$QUERY\""
echo "Limit: $LIMIT results"
echo ""

# Check Ollama is running
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "ERROR: Ollama is not running. Start with: ollama serve"
    exit 1
fi

# Build ailang
echo "Building ailang..."
make build > /dev/null 2>&1

# Count docs in corpus
DOC_COUNT=$(find "$CORPUS" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
echo "Documents in corpus: $DOC_COUNT"
echo ""

# Results file
RESULTS_FILE="/tmp/embed_benchmark_results.txt"
echo "" > "$RESULTS_FILE"

benchmark_model() {
    local model="$1"

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Testing: $model"
    echo ""

    # Check if model exists, pull if not
    if ! ollama list 2>/dev/null | grep -q "$model"; then
        echo "  Pulling $model..."
        ollama pull "$model" > /dev/null 2>&1 || { echo "  SKIP: Failed to pull $model"; return; }
    fi

    # Clear cache for cold start test
    echo "  Clearing embedding cache..."
    rm -rf ~/.ailang/cache/embeddings/*.json 2>/dev/null || true

    # Cold start (no cache)
    echo "  Cold start (embedding all docs)..."
    local start=$(python3 -c 'import time; print(time.time())')

    AILANG_OLLAMA_MODEL="$model" ./bin/ailang docs search \
        --path "$CORPUS" \
        --neural \
        --neural-candidates "$DOC_COUNT" \
        --limit "$LIMIT" \
        "$QUERY" 2>&1 | tail -10

    local end=$(python3 -c 'import time; print(time.time())')
    local cold_ms=$(python3 -c "print(int(round(($end - $start) * 1000, 0)))")
    echo ""
    echo "  ⏱️  Cold start: ${cold_ms}ms"

    # Warm start (cached)
    echo ""
    echo "  Warm start (cached embeddings)..."
    start=$(python3 -c 'import time; print(time.time())')

    AILANG_OLLAMA_MODEL="$model" ./bin/ailang docs search \
        --path "$CORPUS" \
        --neural \
        --neural-candidates "$DOC_COUNT" \
        --limit "$LIMIT" \
        "$QUERY" 2>&1 | tail -5

    end=$(python3 -c 'import time; print(time.time())')
    local warm_ms=$(python3 -c "print(int(round(($end - $start) * 1000, 0)))")
    echo ""
    echo "  ⏱️  Warm start: ${warm_ms}ms"
    echo ""

    # Store results
    echo "$model $cold_ms $warm_ms" >> "$RESULTS_FILE"
}

# Run benchmarks
benchmark_model "embeddinggemma:300m-qat-q4_0"
benchmark_model "embeddinggemma:300m-qat-q8_0"
benchmark_model "embeddinggemma:latest"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "                    SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
printf "%-30s %12s %12s\n" "Model" "Cold (ms)" "Warm (ms)"
printf "%-30s %12s %12s\n" "------------------------------" "------------" "------------"

while read -r line; do
    if [ -n "$line" ]; then
        model=$(echo "$line" | awk '{print $1}')
        cold=$(echo "$line" | awk '{print $2}')
        warm=$(echo "$line" | awk '{print $3}')
        printf "%-30s %12s %12s\n" "$model" "$cold" "$warm"
    fi
done < "$RESULTS_FILE"

echo ""
echo "Notes:"
echo "- Cold = First run, embeds all $DOC_COUNT docs"
echo "- Warm = Second run, uses cached embeddings"
echo "- Set model: export AILANG_OLLAMA_MODEL=embeddinggemma:300m-qat-q4_0"
echo "- Or add to ~/.ailang/config.yaml under embeddings.ollama.model"
