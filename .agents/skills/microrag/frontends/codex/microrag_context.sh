#!/usr/bin/env bash
# microrag_context.sh — Codex CLI PreToolUse hook for Edit | Write | Read
#
# Functionally identical to Claude Code's microrag_context.sh. Codex CLI's
# hook stdin schema matches Claude Code's (.tool_name + .tool_input.file_path
# + .tool_input.content / .new_string), so the same shim works.
#
# Divergence from Claude Code shim: session id env var is CODEX_SESSION_ID
# instead of CLAUDE_SESSION_ID. Output envelope (hookSpecificOutput /
# additionalContext) is accepted by Codex CLI's hooks runtime.
#
# Master switch: set AILANG_MICRORAG_ENABLED=0 to disable entirely.

set +e

[ "${AILANG_MICRORAG_ENABLED:-1}" = "0" ] && exit 0

command -v ailang >/dev/null 2>&1 || exit 0
command -v jq     >/dev/null 2>&1 || exit 0

if [ -z "${AILANG_MICRORAG_SESSION:-}" ] && [ -n "${CODEX_SESSION_ID:-}" ]; then
    export AILANG_MICRORAG_SESSION="$CODEX_SESSION_ID"
fi

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)
case "$TOOL_NAME" in
    Edit|Write|Read|MultiEdit) ;;
    *) exit 0 ;;
esac

FILE_PATH=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)
[ -z "$FILE_PATH" ] && exit 0

case "$FILE_PATH" in
    */node_modules/*|*/.git/*|*/vendor/*|*/__pycache__/*) exit 0 ;;
    *.png|*.jpg|*.jpeg|*.gif|*.ico|*.svg|*.webp)          exit 0 ;;
    *.pdf|*.wasm|*.bin|*.exe|*.dylib|*.so)                exit 0 ;;
    *.zip|*.tar|*.gz|*.bz2)                               exit 0 ;;
esac

case "$TOOL_NAME" in
    Write)     CONTENT=$(echo "$HOOK_JSON" | jq -r '.tool_input.content // ""' 2>/dev/null) ;;
    Edit)      CONTENT=$(echo "$HOOK_JSON" | jq -r '.tool_input.new_string // ""' 2>/dev/null) ;;
    MultiEdit) CONTENT=$(echo "$HOOK_JSON" | jq -r '[.tool_input.edits[]?.new_string] | join("\n")' 2>/dev/null) ;;
    Read)      CONTENT="" ;;
esac

if [ "${#CONTENT}" -gt 4096 ]; then
    CONTENT="${CONTENT:0:4096}"
fi

ARGS=(micro-rag context --tool "$TOOL_NAME" --file "$FILE_PATH")
if [ -n "$CONTENT" ]; then
    ARGS+=(--content "$CONTENT")
fi

if command -v gtimeout >/dev/null 2>&1; then
    RESULT=$(gtimeout 3s ailang "${ARGS[@]}" 2>/dev/null)
elif command -v timeout >/dev/null 2>&1; then
    RESULT=$(timeout 3s ailang "${ARGS[@]}" 2>/dev/null)
else
    RESULT=$(ailang "${ARGS[@]}" 2>/dev/null)
fi
[ -z "$RESULT" ] && exit 0

INJECTION_TEXT=$(echo "$RESULT" | jq -r '.injection.injection_text // ""' 2>/dev/null)
[ -z "$INJECTION_TEXT" ] && exit 0

jq -n --arg ctx "$INJECTION_TEXT" '{
    hookSpecificOutput: {
        hookEventName: "PreToolUse",
        additionalContext: $ctx
    }
}' 2>/dev/null

exit 0
