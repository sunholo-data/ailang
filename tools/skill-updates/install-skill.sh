#!/bin/bash
# Install updated ailang-feedback skill to global location
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIR="$HOME/.claude/skills/ailang-feedback"

echo "Installing ailang-feedback skill..."

mkdir -p "$DEST_DIR/scripts"
cp "$SCRIPT_DIR/ailang-feedback-SKILL.md" "$DEST_DIR/SKILL.md"
cp "$SCRIPT_DIR/send_response.sh" "$DEST_DIR/scripts/send_response.sh"
chmod +x "$DEST_DIR/scripts/send_response.sh"

echo "✅ Skill installed to $DEST_DIR"
echo ""
echo "Files installed:"
echo "  - $DEST_DIR/SKILL.md"
echo "  - $DEST_DIR/scripts/send_response.sh"
echo ""
echo "Test with: ~/.claude/skills/ailang-feedback/scripts/send_response.sh --help"
