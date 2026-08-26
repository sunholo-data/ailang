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

# Step 0: only a VETTED non-opus lane may proceed to the path analysis; anything
# else fails closed to opus.
#
# `pi:` joined `codex:` here for M-OLLAMA-CLOUD-PROVIDER (Mark 2026-08-26, "the
# better models do planning only"), so an Ollama Cloud planner such as
# pi:ollama/kimi-k3:cloud is reachable at all. Without this the pin was a SILENT
# NO-OP: every non-codex value emitted "opus fail-closed:env-pin", so the lane
# would read as pinned in the driver log while actually running opus — the
# routing-never-enforced failure this file exists to prevent.
#
# Both lanes get the SAME treatment deliberately: the doc must still declare the
# Planner-Lane field and every path must still clear the allowlist below. The
# declaration value stays the literal `codex-ok` for compatibility with existing
# design docs; read it as "a vetted non-opus lane is ok", not "codex specifically".
case "${MISSION_PLANNER_MODEL:-}" in
  codex:*|pi:*) ;;
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

# Step 5: every declared path is approved infrastructure. Emit the lane actually
# pinned, not a hardcoded "codex" — the driver uses this value VERBATIM, so
# naming the wrong lane here would route a pi planner to codex.
case "${MISSION_PLANNER_MODEL:-}" in
  pi:*) emit "${MISSION_PLANNER_MODEL} declared:codex-ok" ;;
  *)    emit "codex declared:codex-ok" ;;
esac
