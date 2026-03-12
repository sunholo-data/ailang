#!/bin/bash
# Generate codebase statistics for the AILANG website
# This script outputs JSON that can be merged into codebase_stats.json

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_FILE="${1:-$ROOT_DIR/docs/static/codebase_stats.json}"

cd "$ROOT_DIR"

# Get current version: prefer AILANG_VERSION env var, then git tag
# (git describe --abbrev=0 can miss tags on other branches, e.g. release tags
# not merged back to dev — use AILANG_VERSION override in CI)
VERSION="${AILANG_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")}"

# Normalize: ensure version starts with 'v' prefix
if [[ "$VERSION" != "dev" && "$VERSION" != v* ]]; then
  VERSION="v${VERSION}"
fi

# Get timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Count Go code
GO_TOTAL=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
GO_PROD=$(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
GO_TEST=$(find . -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
GO_FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | wc -l | tr -d ' ')

# Count AILANG code
AIL_TOTAL=$(find . -name "*.ail" -not -path "./.git/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
AIL_FILES=$(find . -name "*.ail" -not -path "./.git/*" | wc -l | tr -d ' ')

# Count stdlib
STD_TOTAL=$(find ./std -name "*.ail" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")
STD_FILES=$(find ./std -name "*.ail" 2>/dev/null | wc -l | tr -d ' ' || echo "0")

# Count TypeScript/React
TS_TOTAL=$(find ./ui ./docs/src -name "*.ts" -o -name "*.tsx" 2>/dev/null | grep -v node_modules | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")
TS_FILES=$(find ./ui ./docs/src \( -name "*.ts" -o -name "*.tsx" \) 2>/dev/null | grep -v node_modules | wc -l | tr -d ' ' || echo "0")

# Count Shell scripts
SH_TOTAL=$(find . -name "*.sh" -not -path "./.git/*" -not -path "./node_modules/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
SH_FILES=$(find . -name "*.sh" -not -path "./.git/*" -not -path "./node_modules/*" | wc -l | tr -d ' ')

# Count Documentation (exclude all node_modules)
DOC_TOTAL=$(find . \( -name "*.md" -o -name "*.mdx" \) -not -path "*/node_modules/*" -not -path "./.git/*" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
DOC_FILES=$(find . \( -name "*.md" -o -name "*.mdx" \) -not -path "*/node_modules/*" -not -path "./.git/*" | wc -l | tr -d ' ')

# Count Design docs specifically
DESIGN_TOTAL=$(find ./design_docs -name "*.md" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")
DESIGN_FILES=$(find ./design_docs -name "*.md" 2>/dev/null | wc -l | tr -d ' ' || echo "0")

# Count website docs
WEBSITE_DOCS=$(find ./docs/docs -name "*.md" -o -name "*.mdx" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

# Count prompts (just prompts/, not cmd/ailang/prompts which is a copy)
PROMPTS_TOTAL=$(find ./prompts -name "*.md" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

# Count CHANGELOG
CHANGELOG_TOTAL=$(wc -l < CHANGELOG.md 2>/dev/null | tr -d ' ' || echo "0")

# Git stats
COMMITS=$(git log --oneline 2>/dev/null | wc -l | tr -d ' ')
CONTRIBUTORS=$(git log --format='%aN' 2>/dev/null | sort -u | wc -l | tr -d ' ')

# Total characters for token estimation
TOTAL_CHARS=$(find . -type f \( -name "*.go" -o -name "*.ail" -o -name "*.md" -o -name "*.mdx" -o -name "*.ts" -o -name "*.tsx" -o -name "*.sh" \) -not -path "./vendor/*" -not -path "./node_modules/*" -not -path "./.git/*" -not -path "./ui/node_modules/*" -not -path "./docs/node_modules/*" | xargs cat 2>/dev/null | wc -c | tr -d ' ')

# Estimate tokens (~4 chars per token for code)
EST_TOKENS=$((TOTAL_CHARS / 4))

# Calculate totals
IMPL_TOTAL=$((GO_TOTAL + AIL_TOTAL + TS_TOTAL + SH_TOTAL))

# Create the current stats entry
CURRENT_STATS=$(cat <<EOF
{
  "version": "$VERSION",
  "timestamp": "$TIMESTAMP",
  "lines": {
    "go_production": $GO_PROD,
    "go_test": $GO_TEST,
    "go_total": $GO_TOTAL,
    "ailang_examples": $AIL_TOTAL,
    "ailang_stdlib": $STD_TOTAL,
    "typescript": $TS_TOTAL,
    "shell": $SH_TOTAL,
    "documentation": $DOC_TOTAL,
    "design_docs": $DESIGN_TOTAL,
    "website_docs": $WEBSITE_DOCS,
    "prompts": $PROMPTS_TOTAL,
    "changelog": $CHANGELOG_TOTAL,
    "implementation_total": $IMPL_TOTAL
  },
  "files": {
    "go": $GO_FILES,
    "ailang": $AIL_FILES,
    "stdlib": $STD_FILES,
    "typescript": $TS_FILES,
    "shell": $SH_FILES,
    "design_docs": $DESIGN_FILES,
    "documentation": $DOC_FILES
  },
  "tokens": {
    "total_characters": $TOTAL_CHARS,
    "estimated_tokens": $EST_TOKENS
  },
  "git": {
    "commits": $COMMITS,
    "contributors": $CONTRIBUTORS
  }
}
EOF
)

# Check if output file exists and has history
if [ -f "$OUTPUT_FILE" ]; then
  # Read existing history
  EXISTING=$(cat "$OUTPUT_FILE")

  # Check if this version already exists in history
  if echo "$EXISTING" | jq -e ".history[] | select(.version == \"$VERSION\")" > /dev/null 2>&1; then
    # Update existing version entry
    UPDATED=$(echo "$EXISTING" | jq --argjson current "$CURRENT_STATS" '
      .current = $current |
      .history = [.history[] | if .version == $current.version then $current else . end] |
      .lastUpdated = $current.timestamp
    ')
  else
    # Append new version to history
    UPDATED=$(echo "$EXISTING" | jq --argjson current "$CURRENT_STATS" '
      .current = $current |
      .history = (.history + [$current]) |
      .lastUpdated = $current.timestamp
    ')
  fi

  echo "$UPDATED" | jq '.' > "$OUTPUT_FILE"
else
  # Create new file with initial history
  cat > "$OUTPUT_FILE" <<EOF
{
  "current": $CURRENT_STATS,
  "history": [
    $CURRENT_STATS
  ],
  "lastUpdated": "$TIMESTAMP"
}
EOF
fi

echo "Codebase stats updated: $OUTPUT_FILE"
echo ""
echo "Summary for $VERSION:"
echo "  Go code:        $GO_TOTAL lines ($GO_PROD production + $GO_TEST test)"
echo "  AILANG:         $AIL_TOTAL lines ($AIL_FILES files)"
echo "  Documentation:  $DOC_TOTAL lines ($DOC_FILES files)"
echo "  Est. tokens:    $EST_TOKENS (~$(echo "scale=1; $EST_TOKENS/1000000" | bc)M)"
echo "  Git commits:    $COMMITS"
