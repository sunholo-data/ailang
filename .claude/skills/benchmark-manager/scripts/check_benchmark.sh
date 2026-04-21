#!/usr/bin/env bash
# Validate a benchmark YAML file for common issues
# Usage: check_benchmark.sh <benchmark_file>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
. "$SCRIPT_DIR/../../_shared/scripts/eval_lib.sh"

BENCHMARK_FILE="${1:-}"

if [[ -z "$BENCHMARK_FILE" ]]; then
    echo "Usage: check_benchmark.sh <benchmark_file>"
    echo "Example: check_benchmark.sh benchmarks/json_parse.yml"
    exit 1
fi

if [[ ! -f "$BENCHMARK_FILE" ]]; then
    echo "Error: File not found: $BENCHMARK_FILE"
    exit 1
fi

ERRORS=0
WARNINGS=0

echo "Checking: $BENCHMARK_FILE"
echo ""

# Check for required fields (v0.14.0+: tier and tags are required by LoadSpec)
REQUIRED_FIELDS=("id" "description" "languages" "entrypoint" "caps" "expected_stdout" "tags")

for field in "${REQUIRED_FIELDS[@]}"; do
    if ! grep -q "^${field}:" "$BENCHMARK_FILE"; then
        echo "ERROR: Missing required field: $field"
        ((ERRORS++))
    fi
done

# Tier is optional (defaults to "core") but warn if missing
if ! grep -q "^tier:" "$BENCHMARK_FILE"; then
    echo "WARNING: No tier field — will default to 'core'"
    echo "  -> Valid tiers: smoke, core, stretch, vision"
    echo "  -> See benchmarks/CURATION.md for tier definitions"
    ((WARNINGS++))
else
    TIER=$(grep "^tier:" "$BENCHMARK_FILE" | head -1 | sed 's/tier:[[:space:]]*//' | tr -d '"' | tr -d "'" | tr -d '[:space:]')
    case "$TIER" in
        smoke|core|stretch|vision) ;;
        *)
            echo "ERROR: Invalid tier '$TIER' — must be one of: smoke, core, stretch, vision"
            ((ERRORS++))
            ;;
    esac
fi

# Validate tags against the canonical 12-tag taxonomy (spec.go ValidTagTaxonomy).
# Supports both flow style `tags: [a, b, c]` and block style (`- a` on each line).
# Max 3 tags per benchmark.
if grep -q "^tags:" "$BENCHMARK_FILE"; then
    TAGS_LINE=$(grep -m1 "^tags:" "$BENCHMARK_FILE")
    TAGS_CSV=""
    if [[ "$TAGS_LINE" =~ \[(.*)\] ]]; then
        # Flow style: tags: [foo, bar, baz]
        TAGS_CSV=$(echo "${BASH_REMATCH[1]}" | tr -d '[:space:]"'\''')
    else
        # Block style
        TAGS_RAW=$(awk '
            /^tags:/ {in_block=1; next}
            in_block && /^[[:space:]]*-/ {
                s=$0
                sub(/^[[:space:]]*-[[:space:]]*/, "", s)
                sub(/#.*$/, "", s)
                gsub(/"/, "", s)
                gsub(/\047/, "", s)
                gsub(/[[:space:]]+$/, "", s)
                if (s != "") print s
                next
            }
            in_block && /^[^[:space:]-]/ {in_block=0}
        ' "$BENCHMARK_FILE")
        TAGS_CSV=$(echo "$TAGS_RAW" | paste -sd ',' -)
    fi

    if [[ -z "$TAGS_CSV" ]]; then
        echo "ERROR: tags field is present but has no entries"
        ((ERRORS++))
    else
        TAG_COUNT=$(echo "$TAGS_CSV" | tr ',' '\n' | grep -c .)
        if [[ "$TAG_COUNT" -gt 3 ]]; then
            echo "ERROR: too many tags ($TAG_COUNT) — maximum is 3 per benchmark"
            ((ERRORS++))
        fi
        if ! validate_tags "$TAGS_CSV" 2>/dev/null; then
            echo "ERROR: non-canonical tag(s) in: $TAGS_CSV"
            echo "  -> Canonical tags: $(canonical_tags_csv)"
            echo "  -> See benchmarks/CURATION.md for tag definitions"
            ((ERRORS++))
        fi
    fi
fi

# Check for prompt vs task_prompt
if grep -q "^prompt:" "$BENCHMARK_FILE"; then
    echo "WARNING: Uses 'prompt:' instead of 'task_prompt:'"
    echo "  -> Models will NOT see the AILANG teaching prompt!"
    echo "  -> Change to 'task_prompt:' unless intentionally testing without teaching"
    ((WARNINGS++))
fi

if ! grep -q "^task_prompt:" "$BENCHMARK_FILE" && ! grep -q "^prompt:" "$BENCHMARK_FILE"; then
    echo "WARNING: No prompt or task_prompt field"
    echo "  -> Models will only see teaching prompt with no specific task"
    ((WARNINGS++))
fi

# Check for valid capabilities
if grep -q "^caps:" "$BENCHMARK_FILE"; then
    CAPS=$(grep "^caps:" "$BENCHMARK_FILE" | sed 's/caps:\s*//')
    VALID_CAPS=("IO" "FS" "Clock" "Net")

    for cap in "${VALID_CAPS[@]}"; do
        # Remove this cap from consideration
        :
    done

    # Simple check - look for common mistakes
    if echo "$CAPS" | grep -qi "io" && ! echo "$CAPS" | grep -q "IO"; then
        echo "WARNING: Capability 'io' should be 'IO' (case-sensitive)"
        ((WARNINGS++))
    fi
    if echo "$CAPS" | grep -qi "fs" && ! echo "$CAPS" | grep -q "FS"; then
        echo "WARNING: Capability 'fs' should be 'FS' (case-sensitive)"
        ((WARNINGS++))
    fi
fi

# Check YAML syntax (basic) - only if pyyaml is available
if python3 -c "import yaml" 2>/dev/null; then
    if ! python3 -c "import yaml; yaml.safe_load(open('$BENCHMARK_FILE'))" 2>/dev/null; then
        echo "ERROR: Invalid YAML syntax"
        ((ERRORS++))
    fi
else
    # Fallback: basic syntax check with Go
    if command -v yq &>/dev/null; then
        if ! yq '.' "$BENCHMARK_FILE" >/dev/null 2>&1; then
            echo "ERROR: Invalid YAML syntax"
            ((ERRORS++))
        fi
    else
        echo "INFO: Skipping YAML syntax check (pyyaml/yq not installed)"
    fi
fi

# Check for <LANG> placeholder in task_prompt
if grep -q "^task_prompt:" "$BENCHMARK_FILE"; then
    if ! grep -A 20 "^task_prompt:" "$BENCHMARK_FILE" | grep -q "<LANG>"; then
        echo "INFO: task_prompt doesn't use <LANG> placeholder"
        echo "  -> Consider using 'Write a program in <LANG> that...' for clarity"
    fi
fi

echo ""
echo "=== Summary ==="
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [[ $ERRORS -gt 0 ]]; then
    exit 1
fi

exit 0
