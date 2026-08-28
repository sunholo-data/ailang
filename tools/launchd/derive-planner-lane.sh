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

# PER-MISSION PATH ALLOWLIST (M-DOCS-MISSION, 2026-08-28).
#
# The allowlist below used to be three literal patterns hardcoded in the Step-4
# `case`. That is exactly right for a mission that plans COMPILER changes: a cheap
# planner has no business planning `internal/`. But it silently defeats a mission
# whose whole subject matter is `docs/` — every docs design doc emits
# "opus fail-closed:path-not-in-codex-allowlist", so the mission's cheap planner
# pin reads as configured while OPUS actually runs, every iteration. Measured
# 2026-08-28 with a discriminating control: a `docs/` doc failed closed while an
# identical `tools/launchd/` doc passed to the pinned pi lane.
#
# So the allowlist becomes per-mission DATA with the infra list as the default —
# v1/world/motoko are byte-for-byte unaffected, and the docs mission widens it in
# its own env file. It is still an allowlist: anything not named is still denied.
#
# `set -f` is LOAD-BEARING, not tidiness. Unquoted `$PLANNER_ALLOWLIST` in a `for`
# undergoes PATHNAME EXPANSION, and this script runs with cwd = the repo, so
# `tools/launchd/*` would expand into the actual file list and then match none of
# the paths a design doc declares. Measured: without `set -f`, even
# `tools/launchd/x.sh` was DENIED by its own literal pattern.
PLANNER_ALLOWLIST="${MISSION_PLANNER_ALLOWLIST:-tools/launchd/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*}"

_path_allowed() {
  _p="$1"; _save_ifs=$IFS; _rc=1
  set -f
  IFS='|'
  for _pat in $PLANNER_ALLOWLIST; do
    # SC2254 is INTENTIONAL here: the allowlist entries ARE globs (`docs/*`), so the
    # expansion must be matched as a pattern. Quoting it would make `tools/launchd/*`
    # match only a literal path ending in an asterisk — i.e. deny everything.
    # shellcheck disable=SC2254
    case "$_p" in $_pat) _rc=0; break ;; esac
  done
  IFS=$_save_ifs; set +f
  return $_rc
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
  if ! _path_allowed "$path"; then
    IFS=$old_ifs; emit "opus fail-closed:path-not-in-codex-allowlist"
  fi
done
IFS=$old_ifs

# Step 5: every declared path is approved infrastructure. Emit the lane actually
# pinned, not a hardcoded literal — the driver and the skill use this value
# VERBATIM, so naming the wrong lane here routes a pinned planner somewhere else.
#
# BOTH branches now emit the full pinned value. Until 2026-08-28 only the `pi:`
# branch did, and `codex:*` collapsed to a bare `codex` — DROPPING THE MODEL. The
# comment above already described that bug; it had been fixed for one branch and
# left in the other.
#
# It was invisible on V1 by coincidence: its pin is `codex:gpt-5.6-sol` and the
# consumer's default is also sol, so the value being dropped happened to equal the
# value fallen back to. That coincidence does not hold for a mission pinned to a
# CHEAPER tier — the docs mission pins `codex:gpt-5.6-luna` ($0.20/$1.20 per M) and
# would have silently planned on gpt-5.6-sol ($2/$10) on every single iteration, on
# the mission created specifically to be cheap.
#
# The skill's own rule already assumes the full form ("Any `codex:*` result enters
# the codex planner recipe"), and a bare `codex` does not match its
# `^([a-z_]+):(.+)$` split at all — so the emitted value was not merely lossy, it
# was unparseable by its documented consumer.
case "${MISSION_PLANNER_MODEL:-}" in
  codex:*|pi:*) emit "${MISSION_PLANNER_MODEL} declared:codex-ok" ;;
  # Unreachable in practice — Step 0 already refuses anything else — but a
  # defensive default beats an empty emit if that gate is ever widened.
  *)            emit "codex declared:codex-ok" ;;
esac
