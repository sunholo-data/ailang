#!/bin/bash
# Auto-format Go files after Edit/Write tool calls
# Input comes as JSON via stdin from Claude Code

file_path=$(jq -r '.tool_input.file_path')

# Only format Go files
if [[ "$file_path" == *.go ]]; then
  # gofmt -w modifies in place; suppress errors for partial/invalid syntax
  if gofmt -w "$file_path" 2>/dev/null; then
    echo "✓ Formatted $file_path"
  fi
fi

# Always exit 0 so we don't block the workflow on syntax errors
exit 0
