#!/usr/bin/env bash
# Analyze recent velocity from CHANGELOG and git commits

set -euo pipefail

DAYS="${1:-7}"  # Default to last 7 days

echo "Analyzing velocity for last $DAYS days..."
echo

# Extract LOC counts from recent CHANGELOG entries
echo "=== Recent CHANGELOG Entries ==="
if [[ -f CHANGELOG.md ]]; then
    # Get recent entries with LOC counts
    grep -E "(Total:|~[0-9]+ LOC)" CHANGELOG.md | head -10 || echo "No LOC metrics found"
fi
echo

# Analyze recent commits
echo "=== Recent Commits (last $DAYS days) ==="
git log --oneline --since="$DAYS days ago" | head -20
echo

# Calculate files changed
echo "=== Files Changed (last $DAYS days) ==="
FILES_CHANGED=$(git diff --stat HEAD~1 HEAD 2>/dev/null | tail -1 || echo "N/A")
echo "$FILES_CHANGED"
echo

# Summary
echo "=== Velocity Summary ==="
echo "Based on CHANGELOG entries and git history, estimate:"
echo "- Average LOC/day from recent milestones"
echo "- Typical milestone duration"
echo "- Current development pace"
echo
echo "Use this data to estimate realistic sprint capacity."
