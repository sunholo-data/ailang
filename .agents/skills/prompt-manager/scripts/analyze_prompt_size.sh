#!/usr/bin/env bash
# Analyze AILANG teaching prompt for optimization opportunities
#
# Usage:
#   analyze_prompt_size.sh <prompt_file>
#
# Example:
#   analyze_prompt_size.sh prompts/v0.3.16.md

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <prompt_file>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 prompts/v0.3.16.md" >&2
    exit 1
fi

PROMPT_FILE="$1"

if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: File not found: $PROMPT_FILE" >&2
    exit 1
fi

echo "=== Prompt Size Analysis ==="
echo "File: $PROMPT_FILE"
echo ""

# Overall metrics
TOTAL_WORDS=$(wc -w < "$PROMPT_FILE" | tr -d ' ')
TOTAL_LINES=$(wc -l < "$PROMPT_FILE" | tr -d ' ')
TOTAL_CHARS=$(wc -c < "$PROMPT_FILE" | tr -d ' ')

echo "--- Overall Metrics ---"
echo "Total words: $TOTAL_WORDS (target: <4000)"
echo "Total lines: $TOTAL_LINES (target: <200)"
echo "Total chars: $TOTAL_CHARS"
echo ""

# Check against targets
if [ "$TOTAL_WORDS" -gt 4000 ]; then
    REDUCTION_NEEDED=$((TOTAL_WORDS - 4000))
    PERCENT_OVER=$(( (TOTAL_WORDS - 4000) * 100 / TOTAL_WORDS ))
    echo "⚠️  OVER TARGET by $REDUCTION_NEEDED words ($PERCENT_OVER%)"
else
    echo "✅ Within target word count"
fi
echo ""

# Section-by-section analysis
echo "--- Section Size Analysis (sorted by word count) ---"
awk '
    /^## / {
        if (section != "") {
            print words "\t" section
        }
        section = $0
        words = 0
        next
    }
    {
        words += NF
    }
    END {
        if (section != "") {
            print words "\t" section
        }
    }
' "$PROMPT_FILE" | sort -rn | head -20

echo ""

# Code block count
CODE_BLOCKS=$(grep -c '```' "$PROMPT_FILE" || echo "0")
CODE_BLOCKS=$((CODE_BLOCKS / 2))  # Each block has opening and closing
echo "--- Code Examples ---"
echo "Code blocks: $CODE_BLOCKS (target: 5-10 comprehensive)"
echo ""

# Table count
TABLE_COUNT=$(grep -c '^|' "$PROMPT_FILE" || echo "0")
TABLE_LINES=$(grep -c '^|' "$PROMPT_FILE" || echo "0")
echo "--- Tables ---"
echo "Table rows: $TABLE_LINES (target: 10+ tables for reference data)"
echo ""

# Link analysis
INTERNAL_LINKS=$(grep -o '\[.*\](docs/.*\.md)' "$PROMPT_FILE" | wc -l | tr -d ' ')
EXTERNAL_LINKS=$(grep -o '\[.*\](http.*\)' "$PROMPT_FILE" | wc -l | tr -d ' ')
echo "--- External References ---"
echo "Internal doc links: $INTERNAL_LINKS (target: 10+)"
echo "External links: $EXTERNAL_LINKS"
echo ""

# Optimization candidates
echo "--- Optimization Opportunities ---"

# Check for prose-heavy builtin docs
BUILTIN_MENTIONS=$(grep -c '_[a-z_]*(' "$PROMPT_FILE" || echo "0")
if [ "$BUILTIN_MENTIONS" -gt 20 ]; then
    echo "⚠️  High builtin mentions ($BUILTIN_MENTIONS) - consider table format + reference 'ailang builtins list'"
fi

# Check for scattered examples
if [ "$CODE_BLOCKS" -gt 15 ]; then
    echo "⚠️  Many code blocks ($CODE_BLOCKS) - consider consolidating into comprehensive examples"
fi

# Check for low table usage
if [ "$TABLE_LINES" -lt 20 ]; then
    echo "⚠️  Few tables ($TABLE_LINES rows) - consider converting prose lists to tables"
fi

# Check for minimal doc links
if [ "$INTERNAL_LINKS" -lt 5 ]; then
    echo "⚠️  Few doc links ($INTERNAL_LINKS) - consider moving detailed content to docs/ and linking"
fi

# Check for verbose patterns
VERBOSE_PATTERNS=$(grep -i -E "(unfortunately|however|note that|keep in mind|remember that)" "$PROMPT_FILE" | wc -l | tr -d ' ')
if [ "$VERBOSE_PATTERNS" -gt 10 ]; then
    echo "⚠️  Verbose language patterns ($VERBOSE_PATTERNS) - consider more direct phrasing"
fi

echo ""

# Summary recommendations
echo "--- Optimization Recommendations ---"

if [ "$TOTAL_WORDS" -lt 4000 ]; then
    echo "✅ Prompt is within target size"
else
    REDUCTION_TARGET=$(( (TOTAL_WORDS - 4000) * 100 / TOTAL_WORDS ))
    echo "Suggested actions to reduce by ~$REDUCTION_TARGET%:"
    echo "  1. Convert builtin prose to tables (saves ~1000 tokens)"
    echo "  2. Consolidate $CODE_BLOCKS code blocks into 5-10 comprehensive examples (saves ~500 tokens)"
    echo "  3. Move detailed explanations to docs/ and link (saves ~800 tokens)"
    echo "  4. Add quick reference section at top (improves AI context efficiency)"
    echo "  5. Remove historical context, move to CHANGELOG.md (saves ~300 tokens)"
    echo ""
    echo "Reference: .claude/skills/prompt-manager/resources/prompt_optimization.md"
fi
