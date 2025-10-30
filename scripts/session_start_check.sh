#!/bin/bash
# Quick Session Start Verification
# Optimized for both cloud and local environments

set -euo pipefail

# Source environment detection
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/detect_environment.sh"

# Setup PATH based on environment
setup_ailang_path

echo "=== 🚀 Session Start Verification ==="
echo ""

# 1. Environment
ENV=$(detect_environment)
echo "1️⃣  Environment: $ENV"
if [[ "$ENV" == "cloud" ]]; then
    echo "   🌐 Running in Claude Web UI"
else
    echo "   💻 Running locally"
fi
echo ""

# 2. AILANG Binary
echo "2️⃣  AILANG Binary:"
if command -v ailang &>/dev/null; then
    VERSION=$(ailang --version 2>&1 | head -1)
    echo "   ✅ $VERSION"
else
    echo "   ❌ Not found in PATH"
    echo "   💡 Run: make quick-install"
    exit 1
fi
echo ""

# 3. Builtin Registry
echo "3️⃣  Builtin Registry:"
BUILTIN_CHECK=$(ailang doctor builtins 2>&1)
if echo "$BUILTIN_CHECK" | grep -q "All builtins are valid"; then
    TOTAL=$(echo "$BUILTIN_CHECK" | grep "Total:" | awk '{print $2}')
    echo "   ✅ Healthy ($TOTAL builtins)"
else
    echo "   ❌ Invalid"
    echo "$BUILTIN_CHECK" | sed 's/^/      /'
    exit 1
fi
echo ""

# 4. Agent Inbox
echo "4️⃣  Agent Inbox:"
INBOX_CHECK=$(ailang agent inbox --unread-only claude-code 2>&1)
if echo "$INBOX_CHECK" | grep -q "No messages"; then
    echo "   ✅ Clear (no unread messages)"
else
    echo "   📬 Unread messages:"
    echo "$INBOX_CHECK" | sed 's/^/      /'
fi
echo ""

# 5. Git Status
echo "5️⃣  Git Repository:"
if git status &>/dev/null; then
    BRANCH=$(git branch --show-current)
    echo "   Branch: $BRANCH"

    if [[ -z $(git status --short) ]]; then
        echo "   ✅ Clean working tree"
    else
        echo "   ⚠️  Uncommitted changes:"
        git status --short | sed 's/^/      /'
    fi
else
    echo "   ❌ Not a git repository"
fi
echo ""

# 6. Skills & Agents
echo "6️⃣  Skills & Agents:"
if [[ -d .claude/skills ]]; then
    SKILL_COUNT=$(ls -1 .claude/skills/ | grep -v "^\." | wc -l)
    echo "   ✅ $SKILL_COUNT skills available"
else
    echo "   ⚠️  No skills directory"
fi

if [[ -d .claude/agents ]]; then
    AGENT_COUNT=$(ls -1 .claude/agents/*.md 2>/dev/null | wc -l)
    echo "   ✅ $AGENT_COUNT agents available"
else
    echo "   ⚠️  No agents directory"
fi
echo ""

echo "=== ✅ Session Ready! ==="
echo ""

# Show helpful commands
echo "💡 Helpful commands:"
echo "   make help              - Show all available make targets"
echo "   make test              - Run test suite"
echo "   make verify-examples   - Verify example files"
if [[ "$ENV" == "cloud" ]]; then
    echo "   export PATH=\$PATH:/root/go/bin  - Add ailang to PATH (if needed)"
fi
