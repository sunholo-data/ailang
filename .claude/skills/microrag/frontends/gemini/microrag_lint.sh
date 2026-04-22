#!/usr/bin/env bash
# microrag_lint.sh — Gemini CLI PostToolUse hook for Edit | Write on *.ail
#
# Functionally identical to Claude Code's microrag_lint.sh. Only diff is
# GEMINI_SESSION_ID env var fallback.

set +e

[ "${AILANG_MICRORAG_ENABLED:-1}" = "0" ] && exit 0

command -v ailang >/dev/null 2>&1 || exit 0
command -v jq     >/dev/null 2>&1 || exit 0

if [ -z "${AILANG_MICRORAG_SESSION:-}" ] && [ -n "${GEMINI_SESSION_ID:-}" ]; then
    export AILANG_MICRORAG_SESSION="$GEMINI_SESSION_ID"
fi

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)
case "$TOOL_NAME" in
    Edit|Write|MultiEdit) ;;
    *) exit 0 ;;
esac

FILE_PATH=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)
[ -z "$FILE_PATH" ] && exit 0

case "$FILE_PATH" in
    *.ail) ;;
    *) exit 0 ;;
esac

case "$TOOL_NAME" in
    Write)     CODE=$(echo "$HOOK_JSON" | jq -r '.tool_input.content // ""' 2>/dev/null) ;;
    Edit)      CODE=$(echo "$HOOK_JSON" | jq -r '.tool_input.new_string // ""' 2>/dev/null) ;;
    MultiEdit) CODE=$(echo "$HOOK_JSON" | jq -r '[.tool_input.edits[]?.new_string] | join("\n")' 2>/dev/null) ;;
esac
[ -z "$CODE" ] && exit 0

if [ "${#CODE}" -gt 8192 ]; then
    CODE="${CODE:0:8192}"
fi

if command -v gtimeout >/dev/null 2>&1; then
    RESULT=$(gtimeout 3s ailang micro-rag lint-builtin --file "$FILE_PATH" --code "$CODE" 2>/dev/null)
elif command -v timeout >/dev/null 2>&1; then
    RESULT=$(timeout 3s ailang micro-rag lint-builtin --file "$FILE_PATH" --code "$CODE" 2>/dev/null)
else
    RESULT=$(ailang micro-rag lint-builtin --file "$FILE_PATH" --code "$CODE" 2>/dev/null)
fi
[ -z "$RESULT" ] && exit 0

INJECTION_TEXT=$(echo "$RESULT" | jq -r '
    if (.nudges // []) | length > 0
    then [.nudges[].injection_text] | join("\n")
    else ""
    end
' 2>/dev/null)
[ -z "$INJECTION_TEXT" ] && exit 0

jq -n --arg ctx "$INJECTION_TEXT" '{
    hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: $ctx
    }
}' 2>/dev/null

exit 0
