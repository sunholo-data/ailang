#!/bin/bash
# mission-base.sh — record the shared default-branch reading (MISSION_BASE_REF, else origin/HEAD's target); classify later disagreement as drift.
# bash 3.2.57-safe: no associative arrays, no ${v,,}, no GNU timeout. Portable to ubuntu/windows/bash5.
set -u
STATE_DIR="${AILANG_STATE_DIR:-$HOME/.ailang/state}"
# Dedicated per-mission base state file. NOT the heartbeat: the driver's slot-verdict
# reader takes its verdict from the heartbeat's LAST row, so a trailing base-* row would
# flip REAPED->CRASHED and degrade the crash-site at= label. (mission-control.sh:1480-1502; Verif. 10)
BASE="$STATE_DIR/mission-${MISSION_NAME:-v1}-base"
REF=""  # set once by resolve_ref in the dispatcher (snap/record/drift only)

github_slug() {
  local slug pattern='^[A-Za-z0-9-]+/[A-Za-z0-9_.-]+$'
  case "$1" in
    https://github.com/*) slug=${1#https://github.com/} ;;
    git@github.com:*) slug=${1#git@github.com:} ;;
    ssh://git@github.com/*) slug=${1#ssh://git@github.com/} ;;
    *) return 1 ;;
  esac
  slug=${slug%/}
  slug=${slug%.git}
  [[ "$slug" =~ $pattern ]] || return 1
  printf '%s\n' "$slug"
}

verify_repo() {
  local expected="${MISSION_REPO:-}" origin actual
  [ -n "$expected" ] || return 0
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    actual='unavailable (outside Git)'
  # A terminator preserves empty trailing entries through command substitution.
  elif ! origin=$(git config --local --get-all remote.origin.url 2>/dev/null && printf '.'); then
    actual='unavailable (missing origin)'
  else
    origin=${origin%$'\n.'}
    if [ -z "$origin" ]; then
      actual='unavailable (missing origin)'
    elif actual=$(github_slug "$origin"); then
      if [ "$(printf '%s' "$actual" | LC_ALL=C tr '[:upper:]' '[:lower:]')" = \
           "$(printf '%s' "$expected" | LC_ALL=C tr '[:upper:]' '[:lower:]')" ]; then
        return 0
      fi
    else
      actual='unavailable (unsupported or ambiguous origin)'
    fi
  fi
  printf 'mission-base: CWD origin is %s, expected %s (MISSION_REPO)\n' "$actual" "$expected" >&2
  return 3
}

resolve_ref() {  # explicit MISSION_BASE_REF wins; else origin/HEAD's symbolic target; else FAIL (no silent default)
  if [ -n "${MISSION_BASE_REF:-}" ]; then REF="$MISSION_BASE_REF"; return 0; fi
  REF=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null) && [ -n "$REF" ] && return 0
  echo "mission-base: cannot derive the base ref: refs/remotes/origin/HEAD is unset and MISSION_BASE_REF is empty." >&2
  echo "mission-base: fix: export MISSION_BASE_REF=origin/<default-branch>, or run 'git remote set-head origin --auto' in this repo." >&2
  return 1
}

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
  local label="$1" old new n
  # (gemini R2 verbatim fix) — check for an empty string explicitly rather than relying on
  # the exit code: an empty `old` must be NO-RECORD (exit 2), never a false DRIFT with an
  # empty old SHA
  old=$(last "$label"); [ -n "$old" ] || { echo "mission-base: no base-$label record yet" >&2; return 2; }
  new=$(git rev-parse "$REF" 2>/dev/null) || { echo "mission-base: cannot resolve $REF" >&2; return 1; }
  if [ "$old" = "$new" ]; then
    echo "base $label steady at $new"; return 0
  fi
  n=$(git rev-list --count "$old..$new" 2>/dev/null || echo '?')
  echo "DRIFT base $label $old -> $new ($n commits) — shared clone moved under this read"; return 1
}

case "${1:-}" in
  snap)
    verify_repo || exit 3
    resolve_ref || exit 1
    snap ;;
  record)
    [ $# -ge 2 ] || { echo "mission-base: record requires a label" >&2; exit 2; }
    verify_repo || exit 3
    resolve_ref || exit 1
    record "$2" ;;
  last) [ $# -ge 2 ] || { echo "mission-base: last requires a label" >&2; exit 2; }; last "$2" ;;
  drift)
    [ $# -ge 2 ] || { echo "mission-base: drift requires a label" >&2; exit 2; }
    verify_repo || exit 3
    resolve_ref || exit 1
    drift "$2" ;;
  *) echo "usage: mission-base.sh {snap|record|last|drift} [label]" >&2; exit 2 ;;
esac
