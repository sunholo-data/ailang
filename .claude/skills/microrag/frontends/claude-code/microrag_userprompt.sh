#!/usr/bin/env bash
# microrag_userprompt.sh — Claude Code UserPromptSubmit hook for μRAG.
#
# When the user submits a prompt, embed it against ailang-syntax +
# ailang-builtins corpora and inject the top hit if it clears the relevance
# floor. This is the right-fit hook for embedding-based retrieval (per
# ADR-002) — the prompt is dense intent-bearing text, exactly what cosine
# similarity excels at.
#
# Master switches:
#   AILANG_MICRORAG_ENABLED=0 disables.
#   AILANG_MICRORAG_USERPROMPT=0 disables only this hook (leaves others on).

set +e

[ "${AILANG_MICRORAG_ENABLED:-1}"     = "0" ] && exit 0
[ "${AILANG_MICRORAG_USERPROMPT:-1}"  = "0" ] && exit 0

command -v ailang >/dev/null 2>&1 || exit 0
command -v jq     >/dev/null 2>&1 || exit 0

if [ -z "${AILANG_MICRORAG_SESSION:-}" ] && [ -n "${CLAUDE_SESSION_ID:-}" ]; then
    export AILANG_MICRORAG_SESSION="$CLAUDE_SESSION_ID"
fi

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
PROMPT=$(echo "$HOOK_JSON" | jq -r '.prompt // ""' 2>/dev/null)
[ -z "$PROMPT" ] && exit 0

# Engine enforces a min length; this is a cheap pre-filter to avoid the
# subprocess on trivially-short prompts ("ok", "yes", etc.).
MIN_LEN="${AILANG_MICRORAG_USERPROMPT_MIN_LEN:-20}"
[ "${#PROMPT}" -lt "$MIN_LEN" ] && exit 0

# Cap prompt length sent to engine — avoid huge payloads (paste-bombs etc.).
if [ "${#PROMPT}" -gt 4096 ]; then
    PROMPT="${PROMPT:0:4096}"
fi

if command -v gtimeout >/dev/null 2>&1; then
    RESULT=$(gtimeout 3s ailang micro-rag user-prompt --prompt "$PROMPT" 2>/dev/null)
elif command -v timeout >/dev/null 2>&1; then
    RESULT=$(timeout 3s ailang micro-rag user-prompt --prompt "$PROMPT" 2>/dev/null)
else
    RESULT=$(ailang micro-rag user-prompt --prompt "$PROMPT" 2>/dev/null)
fi
[ -z "$RESULT" ] && exit 0

INJECTION_TEXT=$(echo "$RESULT" | jq -r '.injection.injection_text // ""' 2>/dev/null)
[ -z "$INJECTION_TEXT" ] && exit 0

jq -n --arg ctx "$INJECTION_TEXT" '{
    hookSpecificOutput: {
        hookEventName: "UserPromptSubmit",
        additionalContext: $ctx
    }
}' 2>/dev/null

exit 0
