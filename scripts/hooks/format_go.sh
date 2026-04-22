#!/bin/bash
# PostToolUse hook for Edit/Write on .go files.
#   1) gofmt -w the edited file
#   2) golangci-lint the file's package (same linters as `make lint`)
# Issues are surfaced via hookSpecificOutput.additionalContext so the
# next turn sees them. Always exits 0 so partial-syntax edits don't
# block the workflow.

set +e

HOOK_JSON=$(cat 2>/dev/null || echo "{}")
file_path=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)

case "$file_path" in
  *.go) ;;
  *) exit 0 ;;
esac

if gofmt -w "$file_path" 2>/dev/null; then
  echo "✓ Formatted $file_path" >&2
fi

command -v golangci-lint >/dev/null 2>&1 || exit 0

pkg_dir=$(dirname "$file_path")
[ -d "$pkg_dir" ] || exit 0

if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT=(gtimeout 8s)
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT=(timeout 8s)
else
  TIMEOUT=()
fi

LINT_RAW=$("${TIMEOUT[@]}" golangci-lint run --path-mode=abs "$pkg_dir" 2>&1)
LINT_RC=$?

if echo "$LINT_RAW" | grep -qE "can't load config|the Go language version"; then
  jq -n --arg ctx "golangci-lint config/toolchain error (run 'make install-lint'):
$LINT_RAW" '{
    hookSpecificOutput: {
      hookEventName: "PostToolUse",
      additionalContext: $ctx
    }
  }' 2>/dev/null
  exit 0
fi

# Filter to issues that mention the just-edited file, matching `make lint`'s filters.
abs_file=$(cd "$pkg_dir" 2>/dev/null && pwd)/$(basename "$file_path")
FILE_ISSUES=$(echo "$LINT_RAW" \
  | grep -F "$abs_file" \
  | grep -v "(related information)" \
  | grep -v "QF[0-9]" \
  | grep -v "ST[0-9]" \
  | grep -v "SA1019:" \
  | grep -v "SA9003:" \
  | grep -v "SA5011:" \
  | grep -v "SA5012:" \
  | grep -v "is unused")

[ -z "$FILE_ISSUES" ] && exit 0

jq -n --arg ctx "golangci-lint issues in $file_path:
$FILE_ISSUES" '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: $ctx
  }
}' 2>/dev/null

exit 0
