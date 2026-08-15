#!/usr/bin/env bash
# Validate or print the authoritative decision ledger embedded in a mission doc.
set -uo pipefail

MODE="${1:---open}"
DOC="${MISSION_DOC:-design_docs/v1-mission.md}"
if [ "${2:-}" = "--file" ]; then DOC="${3:-}"; fi

case "$MODE" in --check|--open|--all) ;; *)
  echo "usage: $0 [--check|--open|--all] [--file mission.md]" >&2; exit 2 ;;
esac
[ -r "$DOC" ] || { echo "mission_decisions: unreadable doc: $DOC" >&2; exit 1; }

awk -F '|' -v mode="$MODE" '
  function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
  /<!-- decision-ledger:start -->/ { starts++; inside=1; next }
  /<!-- decision-ledger:end -->/   { ends++; inside=0; next }
  inside && $0 ~ /^\|[[:space:]]*D-/ {
    id=trim($2); status=trim($3); answer=trim($4); evidence=trim($5)
    rows++
    if (seen[id]++) { printf "duplicate decision ID: %s\n", id > "/dev/stderr"; bad=1 }
    if (status != "OPEN" && status != "RESOLVED") {
      printf "invalid status for %s: %s\n", id, status > "/dev/stderr"; bad=1
    }
    if (answer == "" || evidence == "") {
      printf "empty decision/evidence field for %s\n", id > "/dev/stderr"; bad=1
    }
    if (mode == "--all" || (mode == "--open" && status == "OPEN"))
      printf "%s\t%s\t%s\n", id, status, answer
  }
  END {
    if (starts != 1 || ends != 1) {
      printf "expected exactly one decision-ledger block (start=%d end=%d)\n", starts, ends > "/dev/stderr"; bad=1
    }
    if (rows == 0) { print "decision ledger has no rows" > "/dev/stderr"; bad=1 }
    if (bad) exit 1
    if (mode == "--check") printf "decision ledger valid: %d rows\n", rows
  }
' "$DOC"
