#!/usr/bin/env bash
# brain_on_prompt.sh — UserPromptSubmit hook: semantic BRAIN lookup
# keyed on the user's prompt text (not repo state).
#
# Event-gated: only injects additionalContext when the top result's
# score clears AILANG_BRAIN_MIN_SCORE (default 0.70) and the prompt
# is long enough to be worth searching on.
#
# 0.70 calibrated 2026-07-31 on the ollama:embeddinggemma corpus. This hook is
# looser than the microrag one — it searches ALL namespaces (including the long
# ailang-examples blobs, which carry a higher baseline similarity) and injects
# up to 3 hits rather than 1, so it needs at least as strict a bar. Measured:
# on-topic 0.74-0.85, off-topic 0.45-0.59 short / 0.65 for real long repo-ops
# prompts. The previous 0.55 fired on 2 of 8 short off-topic prompts and on
# every long one. Keep this in step with userPromptRelevanceFloor in
# internal/microrag/userprompt.go.
#
# Master switch: AILANG_BRAIN_ON_PROMPT=0 disables.

set +e

[ "${AILANG_BRAIN_ON_PROMPT:-1}" = "0" ] && exit 0
[ "${AILANG_BRAIN_HOOKS:-1}"     = "0" ] && exit 0

command -v ailang >/dev/null 2>&1 || exit 0
command -v jq     >/dev/null 2>&1 || exit 0

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
PROMPT=$(echo "$HOOK_JSON" | jq -r '.prompt // ""' 2>/dev/null)

MIN_LEN="${AILANG_BRAIN_MIN_PROMPT_LEN:-40}"
[ "${#PROMPT}" -lt "$MIN_LEN" ] && exit 0

MIN_SCORE="${AILANG_BRAIN_MIN_SCORE:-0.70}"
LIMIT="${AILANG_BRAIN_PROMPT_LIMIT:-3}"

if command -v gtimeout >/dev/null 2>&1; then TIMEOUT=(gtimeout 3s)
elif command -v timeout >/dev/null 2>&1; then TIMEOUT=(timeout 3s)
else TIMEOUT=(); fi

# Flags must come BEFORE the positional query (Go stdlib flag parser).
RESULT=$("${TIMEOUT[@]}" ailang cache search -json -limit "$LIMIT" -scope both "$PROMPT" 2>/dev/null)
[ -z "$RESULT" ] && exit 0

INJECTION=$(echo "$RESULT" | jq -r --arg min "$MIN_SCORE" '
  (.results // [])
  | map(select((.score // 0) >= ($min | tonumber)))
  | if length == 0 then "" else
      "🧠 BRAIN (prompt-matched):\n" +
      ([.[] | "  • [" + (.namespace // "?") + "] " + (.key // "?")
              + " (score: " + ((.score * 100 | floor) / 100 | tostring) + ")\n"
              + "    " + ((.content // "") | gsub("\\s+"; " ") | .[0:180])
       ] | join("\n"))
    end
' 2>/dev/null)

[ -z "$INJECTION" ] && exit 0

jq -n --arg ctx "$INJECTION" '{
  hookSpecificOutput: {
    hookEventName: "UserPromptSubmit",
    additionalContext: $ctx
  }
}' 2>/dev/null

exit 0
