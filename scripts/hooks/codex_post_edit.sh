#!/bin/bash
# Translate Codex apply_patch input into the file_path shape used by the
# existing Claude Code format hooks. One patch may edit multiple files.
set +e

command -v jq >/dev/null 2>&1 || exit 0
HOOK_JSON=$(cat 2>/dev/null || echo '{}')
ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
TOOL_NAME=$(printf '%s' "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)

case "$TOOL_NAME" in
  apply_patch)
    FILES=$(printf '%s' "$HOOK_JSON" | jq -r '.tool_input.command // ""' 2>/dev/null \
      | sed -nE 's/^\*\*\* (Add|Update) File: (.*)$/\2/p' \
      | awk '!seen[$0]++')
    ;;
  Edit|Write)
    FILES=$(printf '%s' "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)
    ;;
  *) exit 0 ;;
esac

[ -n "$FILES" ] || exit 0
CONTEXT=""
while IFS= read -r file_path; do
  [ -n "$file_path" ] || continue
  case "$file_path" in /*) resolved="$file_path" ;; *) resolved="$ROOT/$file_path" ;; esac
  [ -f "$resolved" ] || continue
  payload=$(printf '%s' "$HOOK_JSON" | jq --arg file "$resolved" \
    '.tool_name = "Write" | .tool_input.file_path = $file' 2>/dev/null)
  case "$resolved" in
    *.go) formatter="$ROOT/scripts/hooks/format_go.sh" ;;
    *.ail) formatter="$ROOT/scripts/hooks/format_ail.sh" ;;
    *) continue ;;
  esac
  output=$(printf '%s' "$payload" | /bin/bash "$formatter")
  ctx=$(printf '%s' "$output" | jq -r '.hookSpecificOutput.additionalContext // empty' 2>/dev/null)
  [ -z "$ctx" ] || CONTEXT="${CONTEXT}${CONTEXT:+
}$ctx"
done <<EOF
$FILES
EOF

[ -z "$CONTEXT" ] || jq -n --arg ctx "$CONTEXT" '{
  hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}
}'
exit 0
