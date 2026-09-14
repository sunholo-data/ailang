#!/bin/bash
# audit.sh — weekly simplicity audit: re-measure, diff against the last banked
# snapshot, and name what got more complex since. Exit 2 on any regression so
# it can gate; exit 0 otherwise.
#
# Usage:
#   audit.sh              # full (runs make test-core for the timing row)
#   audit.sh --fast       # skip the timed test run
#   audit.sh --against <snapshot.json>
#
# bash 3.2 / BSD grep compatible.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

FAST=""
AGAINST=""
while [ $# -gt 0 ]; do
  case "$1" in
    --fast) FAST="--no-test" ;;
    --against) shift; AGAINST="$1" ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 64 ;;
  esac
  shift
done

SNAPDIR=.ailang/state/simplicity
mkdir -p "$SNAPDIR"
if [ -z "$AGAINST" ]; then
  # newest snapshot that is not today's
  today="$(date +%Y-%m-%d)"
  AGAINST="$(find "$SNAPDIR" -maxdepth 1 -name '*.json' ! -name "$today.json" | sort | tail -1 || true)"
fi
if [ -z "$AGAINST" ] || [ ! -f "$AGAINST" ]; then
  echo "no previous snapshot to diff against; banking a baseline instead" >&2
  # shellcheck disable=SC2086
  exec /bin/bash tools/simplicity_metrics.sh $FAST
fi

NOW="$(mktemp -t simplicity.XXXXXX)"
trap 'rm -f "$NOW"' EXIT
# shellcheck disable=SC2086
/bin/bash tools/simplicity_metrics.sh $FAST --json --out "$NOW" >/dev/null

echo "simplicity audit — $(jq -r .date "$NOW") @ $(jq -r .commit "$NOW")  vs  $(jq -r .date "$AGAINST") @ $(jq -r .commit "$AGAINST")"
echo

# One row per metric: value before, value now, delta, direction verdict.
jq -r --slurpfile prev "$AGAINST" '
  ($prev[0].metrics) as $p
  | .metrics | to_entries[]
  | .key as $k | .value as $m
  | ($p[$k].value // null) as $before
  | (if ($m.value|type)=="number" and ($before|type)=="number" then ($m.value - $before) else null end) as $d
  | (if $d == null then "?"
     elif $d == 0 then "="
     elif ($m.dir == "le" and $d > 0) or ($m.dir == "ge" and $d < 0) then "WORSE"
     else "better" end) as $v
  | [$k, ($before|tostring), ($m.value|tostring), (if $d==null then "" elif $d>0 then "+"+($d|tostring) else ($d|tostring) end), $v]
  | @tsv' "$NOW" \
  | awk -F'\t' 'BEGIN{printf "%-40s %10s %10s %8s  %s\n","metric","before","now","delta",""} {printf "%-40s %10s %10s %8s  %s\n",$1,$2,$3,$4,$5}'

regressions="$(jq -r --slurpfile prev "$AGAINST" '
  ($prev[0].metrics) as $p
  | .metrics | to_entries[]
  | select(.value.gate != null)
  | select((.value.value|type)=="number" and (($p[.key].value // null)|type)=="number")
  | select((.value.dir=="le" and .value.value > $p[.key].value) or (.value.dir=="ge" and .value.value < $p[.key].value))
  | .key' "$NOW")"

echo
echo "── what to look at ──────────────────────────────────────────────"
# Concrete names behind the counts, so the reader can act without re-deriving.
echo "• duplicate symbol names (>=3 packages):"
"$(dirname "$0")/dup_symbols.sh" 3 | head -15 | sed 's/^/    /'
echo "• packages without a package comment:"
jq -r '.metrics.packages_without_doc.detail' "$NOW" | tr ' ' '\n' | sed 's/^/    /'
echo "• platform packages / third-party roots reachable from the language core:"
jq -r '[.metrics.closure_platform_packages.detail, .metrics.closure_leak_roots.detail] | join(" ")' "$NOW" | tr ' ' '\n' | sed 's/^/    /'
echo "• skills citing an ailang command the binary does not have:"
if [ -f cmd/ailang/main.go ]; then
  # The case labels are the authority (help omits some on purpose); aliases share
  # a case line, so match the quoted name anywhere on one. Prose words that follow
  # "ailang" in a sentence ("the ailang repo") are dropped by the stop list.
  grep -rhoE 'ailang [a-z][a-z0-9-]*[a-z0-9]' .claude/skills/*/SKILL.md 2>/dev/null | sort -u \
    | while read -r _ c; do
        case "$c" in repo|repos|commands|command|binary|source|sources|language|project|package|packages|program|programs|code|syntax|file|files|team|core|users|user|ecosystem|world|parse|registry|dev|cli|module|modules|stdlib|std|prompt-version) continue ;; esac
        grep -E "case .*\"$c\"" cmd/ailang/main.go >/dev/null || echo "    ailang $c"
      done | sort -u | head -20
fi

# Bank today's snapshot as the new reference.
cp "$NOW" "$SNAPDIR/$(jq -r .date "$NOW").json"
echo
if [ -n "$regressions" ]; then
  echo "REGRESSIONS since $(jq -r .date "$AGAINST"):"
  printf '%s\n' "$regressions" | sed 's/^/  - /'
  exit 2
fi
echo "no regressions since $(jq -r .date "$AGAINST")"
