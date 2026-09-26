#!/bin/bash
# mission-base.sh — record the shared origin/dev reading; classify later disagreement as drift.
# bash 3.2.57-safe: no associative arrays, no ${v,,}, no GNU timeout. Portable to ubuntu/windows/bash5.
set -u
STATE_DIR="${AILANG_STATE_DIR:-$HOME/.ailang/state}"
# Dedicated per-mission base state file. NOT the heartbeat: the driver's slot-verdict
# reader takes its verdict from the heartbeat's LAST row, so a trailing base-* row would
# flip REAPED->CRASHED and degrade the crash-site at= label. (mission-control.sh:1480-1502; Verif. 10)
BASE="$STATE_DIR/mission-${MISSION_NAME:-v1}-base"
REF="${MISSION_BASE_REF:-origin/dev}"

snap() {  # FULL sha<TAB>ISO8601-UTC from the CURRENT shared ref (no fetch: we read the shared .git)
  local sha iso
  sha=$(git rev-parse "$REF" 2>/dev/null) || { echo "mission-base: cannot resolve $REF" >&2; return 1; }
  iso=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  printf '%s\t%s\n' "$sha" "$iso"
}

record() {  # snap EXACTLY ONCE, append base-<label> row to the per-mission base file; echo sha<TAB>iso
  local label="$1" rec sha iso epoch attempt
  rec=$(snap) || return $?
  sha=${rec%%$'\t'*}; iso=${rec#*$'\t'}  # single-read invariant: the SHA recorded is the SHA of exactly one rev-parse
  epoch=$(date +%s); attempt="${MISSION_ATTEMPT:-1}"
  printf '%s\t%s\tbase-%s\t%s\t%s\n' "$epoch" "$iso" "$label" "$attempt" "$sha" >> "$BASE" \
    || { echo "mission-base: cannot write $BASE" >&2; return 1; }
  printf '%s\t%s\n' "$sha" "$iso"
}

last() {  # the full SHA last recorded under base-<label>; exits 1 when no matching row exists
  # (glm R2 verbatim fix) — so the missing-record guard in drift() fires for BOTH the
  # missing-file case and the exists-but-no-matching-label case, never a silent empty string
  awk -F '\t' -v l="base-$1" '$3==l {sha=$5} END {if (sha) {print sha; exit 0} else exit 1}' "$BASE"
}

drift() { # compare the last base-<label> row against a fresh snap; exit 1 on disagreement
  local label="$1" old new oldn n
  # (gemini R2 verbatim fix) — check for an empty string explicitly rather than relying on
  # the exit code: an empty `old` must be NO-RECORD (exit 2), never a false DRIFT with an
  # empty old SHA
  old=$(last "$label"); [ -n "$old" ] || { echo "mission-base: no base-$label record yet" >&2; return 2; }
  new=$(git rev-parse "$REF" 2>/dev/null) || return 1
  if [ "$old" = "$new" ]; then
    echo "base $label steady at $new"; return 0
  fi
  n=$(git rev-list --count "$old..$new" 2>/dev/null || echo '?')
  echo "DRIFT base $label $old -> $new ($n commits) — shared clone moved under this read"; return 1
}

case "${1:-}" in
  snap) snap ;;
  record) [ $# -ge 2 ] || { echo "mission-base: record requires a label" >&2; exit 2; }; record "$2" ;;
  last) [ $# -ge 2 ] || { echo "mission-base: last requires a label" >&2; exit 2; }; last "$2" ;;
  drift) [ $# -ge 2 ] || { echo "mission-base: drift requires a label" >&2; exit 2; }; drift "$2" ;;
  *) echo "usage: mission-base.sh {snap|record|last|drift} [label]" >&2; exit 2 ;;
esac
