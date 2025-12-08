#!/bin/bash
# Install AILANG VS Code extension

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
EXT_SRC="$PROJECT_ROOT/.vscode/extensions/ailang"
EXT_DEST="$HOME/.vscode/extensions/ailang"

echo "Installing AILANG VS Code extension..."

# Remove old extension if exists
if [ -d "$EXT_DEST" ]; then
    echo "Removing old installation..."
    rm -rf "$EXT_DEST"
fi

# Copy extension
echo "Copying extension to $EXT_DEST..."
cp -r "$EXT_SRC" "$EXT_DEST"

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Restart VS Code or run 'Developer: Reload Window'"
echo "  2. Open any .ail file to see syntax highlighting"
echo ""
echo "If colors don't appear:"
echo "  - Check bottom-right shows 'AILANG' as the language"
echo "  - Try Cmd+Shift+P > 'Change Language Mode' > 'AILANG'"
