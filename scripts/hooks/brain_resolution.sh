#!/bin/bash
# brain_resolution.sh - Capture git commit resolutions into the AILANG brain
#
# User-level PostToolUse hook — works in any git project.
# Detects git commit commands and stores resolution frames automatically.
#
# Two storage strategies:
#   1. Resolution frames (append-only, timestamped key) — commit history
#   2. Design doc frames (upsert, stable key per doc) — always-current content
#
# Enrichment patterns (configurable via AILANG_BRAIN_ENRICH_PATTERNS):
#   - design_docs/*.md, docs/*.md — design documents
#   - CHANGELOG.md — release history
#   - std/*.ail — stdlib definitions
#
# Claude Code PostToolUse hook sends JSON on stdin with:
#   { "tool_name": "Bash", "tool_input": { "command": "..." }, "tool_output": "..." }

set -euo pipefail

# Check if brain hooks are disabled
if [ "${AILANG_BRAIN_HOOKS:-1}" = "0" ]; then
    exit 0
fi

# Read hook JSON from stdin
HOOK_JSON=$(cat || echo "{}")

# Only process Bash tool calls
TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)
if [ "$TOOL_NAME" != "Bash" ]; then
    exit 0
fi

# Check if this is a git commit command
COMMAND=$(echo "$HOOK_JSON" | jq -r '.tool_input.command // ""' 2>/dev/null)
if ! echo "$COMMAND" | grep -qE 'git\s+commit'; then
    exit 0
fi

# Check if ailang is available
if ! command -v ailang &> /dev/null; then
    exit 0
fi

# Must be in a git repo
if ! git rev-parse --is-inside-work-tree &>/dev/null; then
    exit 0
fi

# Try to get the latest commit message
COMMIT_MSG=$(git log -1 --format="%s" 2>/dev/null || echo "")
if [ -z "$COMMIT_MSG" ]; then
    exit 0
fi

# Get diff stats for the commit
DIFF_SUMMARY=$(git diff --stat HEAD~1 HEAD 2>/dev/null | tail -1 || echo "")
CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD 2>/dev/null | tr '\n' ',' | sed 's/,$//' || echo "")

# Enrich: Collect content from key files touched by this commit
# Matches by path prefix — works with any depth of subdirectories
ENRICH_CONTENT=""
MAX_ENRICH_BYTES=2048  # Cap enrichment at 2KB — just enough for key context

# Track design/doc files for separate upsert
DOC_FILES=()

while IFS= read -r file; do
    [ -z "$file" ] && continue

    # Match key file types by path prefix and extension
    # This covers design_docs/planned/v0_9_3/foo.md, docs/guides/bar.md, etc.
    MATCHED=false
    IS_DOC=false
    case "$file" in
        design_docs/*.md|design_docs/*/*.md|design_docs/*/*/*.md|design_docs/*/*/*/*.md) MATCHED=true; IS_DOC=true ;;
        docs/*.md|docs/*/*.md|docs/*/*/*.md|docs/*/*/*/*.md) MATCHED=true; IS_DOC=true ;;
        CHANGELOG.md|CHANGELOG*.md) MATCHED=true; IS_DOC=true ;;
        std/*.ail|stdlib/*.ail) MATCHED=true ;;
    esac

    # Allow override via env var (pipe-delimited extra patterns)
    if [ "$MATCHED" = false ] && [ -n "${AILANG_BRAIN_ENRICH_EXTRA:-}" ]; then
        IFS='|' read -ra EXTRA <<< "$AILANG_BRAIN_ENRICH_EXTRA"
        for pattern in "${EXTRA[@]}"; do
            # shellcheck disable=SC2254
            case "$file" in
                $pattern) MATCHED=true; [[ "$file" == *.md ]] && IS_DOC=true; break ;;
            esac
        done
    fi

    if [ "$MATCHED" = true ] && [ -f "$file" ] && [ -r "$file" ]; then
        # Track doc files for strategy 2
        if [ "$IS_DOC" = true ]; then
            DOC_FILES+=("$file")
        fi

        # Enrich resolution content (strategy 1)
        CURRENT_SIZE=${#ENRICH_CONTENT}
        REMAINING=$((MAX_ENRICH_BYTES - CURRENT_SIZE))

        if [ "$REMAINING" -gt 200 ]; then
            ENRICH_CONTENT="${ENRICH_CONTENT}
--- ${file} ---
$(head -c "$REMAINING" "$file" 2>/dev/null || true)
"
        fi
    fi
done < <(git diff --name-only HEAD~1 HEAD 2>/dev/null)

# Build the put-resolution command
RESOLUTION_ARGS=(
    --commit-msg "$COMMIT_MSG"
    --diff-summary "$DIFF_SUMMARY"
    --files "$CHANGED_FILES"
    --source "hook:commit"
)

# Add enrichment content if we collected any
if [ -n "$ENRICH_CONTENT" ]; then
    RESOLUTION_ARGS+=(--enrich "$ENRICH_CONTENT")
fi

# Auto-embed if an embedder is configured (silently skipped if not)
RESOLUTION_ARGS+=(--embed)

# Strategy 1: Store append-only resolution (fire-and-forget)
ailang cache put-resolution "${RESOLUTION_ARGS[@]}" >/dev/null 2>&1 &

# Strategy 2: Upsert each doc as a standalone frame with stable key
# Stores summary (first 2KB) + file path pointer, not full content.
# Key format: doc_<filename_without_ext> — overwrites on each commit
MAX_DOC_LINES=200    # First ~200 lines captures title, status, problem, key decisions
MAX_DOC_BYTES=2048   # Hard cap at 2KB
for doc in "${DOC_FILES[@]}"; do
    DOC_BASENAME=$(basename "$doc" .md)
    DOC_KEY="doc_${DOC_BASENAME}"

    # Read summary: first N lines, capped at byte limit
    DOC_SUMMARY=$(head -n "$MAX_DOC_LINES" "$doc" 2>/dev/null | head -c "$MAX_DOC_BYTES" || true)
    if [ -z "$DOC_SUMMARY" ]; then
        continue
    fi

    # Append file path pointer so consumers know where to find the full doc
    DOC_CONTENT="${DOC_SUMMARY}

---
Full document: ${doc}"

    ailang cache put "$DOC_KEY" \
        --ns "design-docs" \
        --content "$DOC_CONTENT" \
        --source "hook:commit" \
        --embed \
        >/dev/null 2>&1 &
done

exit 0
