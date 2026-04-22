#!/usr/bin/env bash
# audit_skill_usage.sh — rank Claude Code skills by actual invocation count
# across session transcripts for this project.
#
# Data sources:
#   - Transcripts: ~/.claude/projects/-Users-mark-dev-sunholo-ailang/*.jsonl
#   - Project skills: .claude/skills/*/SKILL.md
#   - User skills:    ~/.claude/skills/*/SKILL.md
#
# Invocation markers counted per skill:
#   "skill":"<name>"         — Skill() tool call in transcript
#
# Output: markdown table sorted by invocation count ascending (unused first).

set -euo pipefail

PROJECT_ROOT="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
TRANSCRIPT_DIR="${CLAUDE_TRANSCRIPT_DIR:-$HOME/.claude/projects/-Users-mark-dev-sunholo-ailang}"
PROJECT_SKILLS="$PROJECT_ROOT/.claude/skills"
USER_SKILLS="$HOME/.claude/skills"
DAYS="${AUDIT_DAYS:-30}"

if [ ! -d "$TRANSCRIPT_DIR" ]; then
    echo "Transcript directory not found: $TRANSCRIPT_DIR" >&2
    exit 1
fi

# Collect transcripts modified within the window
CUTOFF_EPOCH=$(date -v-"${DAYS}"d +%s 2>/dev/null || date -d "${DAYS} days ago" +%s)
TRANSCRIPTS=()
while IFS= read -r f; do
    MTIME=$(stat -f %m "$f" 2>/dev/null || stat -c %Y "$f" 2>/dev/null)
    if [ -n "$MTIME" ] && [ "$MTIME" -ge "$CUTOFF_EPOCH" ]; then
        TRANSCRIPTS+=("$f")
    fi
done < <(find "$TRANSCRIPT_DIR" -maxdepth 1 -name '*.jsonl' -type f 2>/dev/null)

TOTAL_TRANSCRIPTS="${#TRANSCRIPTS[@]}"

# Gather skill names + sources
declare -a ROWS
collect_skills() {
    local src="$1" dir="$2"
    [ -d "$dir" ] || return 0
    for skill_md in "$dir"/*/SKILL.md; do
        [ -f "$skill_md" ] || continue
        local skill_dir name
        skill_dir=$(dirname "$skill_md")
        name=$(basename "$skill_dir")
        ROWS+=("$name|$src|$skill_dir")
    done
}
collect_skills "project" "$PROJECT_SKILLS"
collect_skills "user"    "$USER_SKILLS"

# Count invocations per skill across the transcript window
format_relative() {
    local ts="$1"
    [ -z "$ts" ] || [ "$ts" = "0" ] && { echo "never"; return; }
    local now diff
    now=$(date +%s)
    diff=$(( now - ts ))
    if   [ "$diff" -lt 3600 ];    then echo "$(( diff / 60 ))m ago"
    elif [ "$diff" -lt 86400 ];   then echo "$(( diff / 3600 ))h ago"
    elif [ "$diff" -lt 2592000 ]; then echo "$(( diff / 86400 ))d ago"
    else                               echo "$(( diff / 2592000 ))mo ago"
    fi
}

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

for row in "${ROWS[@]}"; do
    name="${row%%|*}"
    rest="${row#*|}"
    src="${rest%%|*}"

    count=0
    last_mtime=0
    for f in "${TRANSCRIPTS[@]}"; do
        # Exact-match the skill name field; avoid substring collisions
        hits=$(grep -c "\"skill\":\"${name}\"" "$f" 2>/dev/null || true)
        hits="${hits:-0}"
        if [ "$hits" -gt 0 ] 2>/dev/null; then
            count=$(( count + hits ))
            mt=$(stat -f %m "$f" 2>/dev/null || stat -c %Y "$f" 2>/dev/null || echo 0)
            [ "$mt" -gt "$last_mtime" ] && last_mtime="$mt"
        fi
    done

    printf '%d\t%s\t%s\t%d\n' "$count" "$name" "$src" "$last_mtime" >> "$TMPFILE"
done

# Emit markdown, sorted by count ascending then name ascending
echo "# Skill Usage Audit (last ${DAYS} days, ${TOTAL_TRANSCRIPTS} transcripts)"
echo
echo "| Skill | Source | Invocations | Last used |"
echo "|-------|--------|-------------|-----------|"
sort -t $'\t' -k1,1n -k2,2 "$TMPFILE" | while IFS=$'\t' read -r count name src last_mtime; do
    rel=$(format_relative "$last_mtime")
    printf '| %s | %s | %d | %s |\n' "$name" "$src" "$count" "$rel"
done

echo
TOTAL_SKILLS="${#ROWS[@]}"
ZERO_USE=$(awk -F'\t' '$1 == 0' "$TMPFILE" | wc -l | tr -d ' ')
echo "**Summary**: ${TOTAL_SKILLS} skills scanned, ${ZERO_USE} unused in last ${DAYS} days."
