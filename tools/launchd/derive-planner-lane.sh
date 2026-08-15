#!/bin/bash
# Derive the mission planner lane from a design document. Pure text only.

emit() {
  local result="$1" reason
  case "$result" in
    opus\ *)
      if [ "${MISSION_ANTHROPIC_AVAILABLE:-1}" = "0" ]; then
        reason=${result#opus }
        printf '%s anthropic-fallback:%s\n' \
          "${MISSION_PLANNER_ANTHROPIC_FALLBACK:-codex:gpt-5.6-sol}" "$reason"
        exit 0
      fi
      ;;
  esac
  printf '%s\n' "$result"
  exit 0
}

# Step 0: an explicit non-codex planner pin always wins.
case "${MISSION_PLANNER_MODEL:-}" in
  codex:*) ;;
  *) emit "opus fail-closed:env-pin" ;;
esac

# Step 1: require one readable document argument.
doc=${1:-}
if [ -z "$doc" ] || [ ! -r "$doc" ]; then
  printf '%s\n' "derive-planner-lane: design document is missing or unreadable" >&2
  emit "opus fail-closed:no-doc"
fi

# Step 2: read and validate the declaration.
planner_line=$(grep -m1 -E '^\*\*Planner-Lane\*\*:' "$doc" 2>/dev/null)
if [ -z "$planner_line" ]; then
  emit "opus fail-closed:planner-lane-field-missing"
fi
planner_value=${planner_line#\*\*Planner-Lane\*\*:}
planner_value=$(printf '%s\n' "$planner_value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
case "$planner_value" in
  codex-ok|opus-required) ;;
  *) emit "opus fail-closed:planner-lane-field-invalid" ;;
esac

# Step 3: an opus declaration needs no path analysis.
if [ "$planner_value" = "opus-required" ]; then
  emit "opus declared:opus-required"
fi

# Step 4: locate exactly one Files section and extract the first backticked
# token from every top-level bullet in it. Continuations and later tokens are
# prose. The awk matcher is the portable ERE equivalent of Files\b.
section_count=$(awk '
  BEGIN { count = 0 }
  {
    line = tolower($0)
    if (line ~ /^#{2,4}[[:space:]]+files([^[:alnum:]_]|$)/) count++
  }
  END { print count }
' "$doc")
if [ "$section_count" -ne 1 ]; then
  emit "opus fail-closed:no-files-section"
fi

paths=$(awk '
  BEGIN { in_files = 0 }
  {
    line = tolower($0)
    if (!in_files && line ~ /^#{2,4}[[:space:]]+files([^[:alnum:]_]|$)/) {
      in_files = 1
      next
    }
    if (in_files && ($0 ~ /^#{1,4}[[:space:]]/ || $0 ~ /^---$/)) exit
    if (in_files && $0 ~ /^- /) {
      if (match($0, /`[^`]+`/)) print substr($0, RSTART + 1, RLENGTH - 2)
      else print "__UNPARSABLE_PATH_ENTRY__"
    }
  }
' "$doc")

if [ -z "$paths" ]; then
  printf '%s\n' "derive-planner-lane: Files section has no path bullets" >&2
  emit "opus fail-closed:unparsable-path-entry"
fi

old_ifs=$IFS
IFS='
'
for path in $paths; do
  if [ -z "$path" ] || [ "$path" = "__UNPARSABLE_PATH_ENTRY__" ]; then
    IFS=$old_ifs
    emit "opus fail-closed:unparsable-path-entry"
  fi
  case "$path" in
    */*|*.md|*.sh|*.go|*.yml) ;;
    *) IFS=$old_ifs; emit "opus fail-closed:unparsable-path-entry" ;;
  esac
  case "$path" in
    /*|~*|*..*) IFS=$old_ifs; emit "opus fail-closed:path-not-in-codex-allowlist" ;;
  esac
  case "$path" in
    tools/launchd/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*) ;;
    *) IFS=$old_ifs; emit "opus fail-closed:path-not-in-codex-allowlist" ;;
  esac
done
IFS=$old_ifs

# Step 5: every declared path is approved infrastructure.
emit "codex declared:codex-ok"
