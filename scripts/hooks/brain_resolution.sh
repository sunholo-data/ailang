#!/bin/bash
# brain_resolution.sh - Capture git commit resolutions into the AILANG brain
#
# PostToolUse hook for Bash — detects git commit commands and stores
# resolution frames automatically. Runs async, <200ms budget.
#
# Two storage strategies:
#   1. Resolution frames (append-only, timestamped key) — commit history
#   2. Design doc frames (upsert, stable key per doc) — always-current content
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

# Extract commit info from the tool output
TOOL_OUTPUT=$(echo "$HOOK_JSON" | jq -r '.tool_output // ""' 2>/dev/null)

# Try to get the latest commit message
COMMIT_MSG=$(git log -1 --format="%s" 2>/dev/null || echo "")
if [ -z "$COMMIT_MSG" ]; then
    exit 0
fi

# Get diff stats for the commit
DIFF_SUMMARY=$(git diff --stat HEAD~1 HEAD 2>/dev/null | tail -1 || echo "")
CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD 2>/dev/null | tr '\n' ',' | sed 's/,$//' || echo "")

# Enrich: Collect content from key files touched by this commit
# Design docs, CHANGELOG, and stdlib files get their content stored in the resolution
ENRICH_CONTENT=""
MAX_ENRICH_BYTES=2048  # Cap enrichment at 2KB — just enough for key context

# Also track design docs for separate upsert
DESIGN_DOCS=()

while IFS= read -r file; do
    [ -z "$file" ] && continue

    # Track design docs for upsert (strategy 2)
    case "$file" in
        design_docs/*.md)
            if [ -f "$file" ] && [ -r "$file" ]; then
                DESIGN_DOCS+=("$file")
            fi
            ;;
    esac

    # Enrich resolution content (strategy 1)
    case "$file" in
        design_docs/*.md|CHANGELOG.md|std/*.ail)
            if [ -f "$file" ] && [ -r "$file" ]; then
                CURRENT_SIZE=${#ENRICH_CONTENT}
                REMAINING=$((MAX_ENRICH_BYTES - CURRENT_SIZE))

                if [ "$REMAINING" -gt 200 ]; then
                    ENRICH_CONTENT="${ENRICH_CONTENT}
--- ${file} ---
$(head -c "$REMAINING" "$file" 2>/dev/null || true)
"
                fi
            fi
            ;;
    esac
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

# Strategy 2: Upsert each design doc as a standalone frame with stable key
# Stores summary (first 2KB) + file path pointer, not full content.
# This keeps frames lean for search/injection while giving embeddings enough signal.
# Key format: design_doc_<filename_without_ext> — overwrites on each commit
MAX_DOC_LINES=200    # First ~200 lines captures title, status, problem, key decisions
MAX_DOC_BYTES=2048   # Hard cap at 2KB
for doc in "${DESIGN_DOCS[@]}"; do
    # Derive stable key from path: design_docs/planned/v0_9_3/m-brain.md -> design_doc_m-brain
    DOC_BASENAME=$(basename "$doc" .md)
    DOC_KEY="design_doc_${DOC_BASENAME}"

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
