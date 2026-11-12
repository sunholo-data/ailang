#!/bin/bash
# audit.sh — weekly simplicity audit: re-measure, diff against the last banked
# snapshot, and name what got more complex since. Exit 2 on any regression so
# it can gate; exit 0 otherwise.
#
# Usage:
#   audit.sh              # full (runs make test-core for the timing row)
#   audit.sh --fast       # skip the timed test run
#   audit.sh --against <snapshot.json>
#   audit.sh --offline    # skip the external-service rows (Sonar, GitHub): null + reason
#   audit.sh --services   # also the Cloud Run used-or-not check (gcloud, read-only, minutes)
#
# A metric absent from the old snapshot (or null in either) shows `?`, never
# WORSE, and cannot be a regression: a new row or an offline run is not a change.
#
# bash 3.2 / BSD grep compatible.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

FAST=""
AGAINST=""
EXT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --fast) FAST="--no-test" ;;
    --against) shift; AGAINST="$1" ;;
    --offline|--services) EXT="$EXT $1" ;;
    -h|--help) sed -n '2,17p' "$0"; exit 0 ;;
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
  exec /bin/bash tools/simplicity_metrics.sh $FAST $EXT
fi

NOW="$(mktemp -t simplicity.XXXXXX)"
trap 'rm -f "$NOW"' EXIT
# shellcheck disable=SC2086
/bin/bash tools/simplicity_metrics.sh $FAST $EXT --json --out "$NOW" >/dev/null

echo "simplicity audit — $(jq -r .date "$NOW") @ $(jq -r .commit "$NOW")  vs  $(jq -r .date "$AGAINST") @ $(jq -r .commit "$AGAINST")"
echo

# One row per metric: value before, value now, delta, direction verdict.
jq -r --slurpfile prev "$AGAINST" '
  ($prev[0].metrics) as $p
  | .metrics | to_entries[]
  | .key as $k | .value as $m
  | ($p | has($k)) as $known
  | ($p[$k].value // null) as $before
  | (if ($m.value|type)=="number" and ($before|type)=="number" then ($m.value - $before) else null end) as $d
  | (if $d == null then "?"
     elif $d == 0 then "="
     elif ($m.dir != "le" and $m.dir != "ge") then "changed"
     elif ($m.dir == "le" and $d > 0) or ($m.dir == "ge" and $d < 0) then "WORSE"
     else "better" end) as $v
  | [$k, (if $known then ($before|tostring) else "—" end), ($m.value|tostring), (if $d==null then "" elif $d>0 then "+"+($d|tostring) else ($d|tostring) end), $v]
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
echo "• unreachable functions by package (deadcode -test=false; cross-check -test=true + git log -S before deleting):"
jq -r '.metrics.unreachable_funcs | if .value == null then "(unmeasured: \(.reason))" else .detail end' "$NOW" | tr ' ' '\n' | sed 's/^/    /'
echo "• external services (SonarCloud, GitHub security):"
jq -r '.metrics | to_entries[]
  | select(.key | test("^(sonar_|codeql_|dependabot_|secret_scanning_)"))
  | "\(.key)=\(.value.value // "null")  \(if .value.value == null then "UNMEASURED: " + .value.reason else (.value.detail // "") end)"' "$NOW" | sed 's/^/    /'
echo "• Cloud Run services, 2xx requests in 30 days (--services):"
jq -r '.metrics.cloud_run_services_used
  | if .value == null then "(unmeasured: \(.reason))"
    else (.services[] | if .error != null then "\(.project)/\(.service)  ERROR \(.error)"
          else "\(.project)/\(.service)  2xx=\(.requests_2xx)\(if .truncated then "+ (capped)" else "" end) ips=\(.distinct_ips)  \([.classes[] | "\(.class)=\(.n)"] | join(" "))" end)
    end' "$NOW" | sed 's/^/    /'
echo "• skills citing an ailang command the binary does not have (count, command):"
# The dispatch table is the authority (help omits some on purpose): since the S5
# CLI reshape every command, group member and alias is a `Name: "x"` or an
# `Aliases: []string{..., "x", ...}` entry in cmd/ailang/commands_*.go. This used
# to grep `case "x"` in main.go, which has had no case labels since the reshape,
# so every cited command read as missing (found 2026-11-09).
# Scope is every *.md under .claude/skills (resources/ included — the skill body
# links there, and headless-runner's resources held 67 retired `ailang agent`
# calls the SKILL.md-only scan never saw). Only COMMAND-POSITION citations count:
# inline code (`ailang x`), line start / `$ ` prompt, `$(`, or after | ; & — so
# prose ("the ailang repo") no longer needs a stop list. Honest "Planned" rows
# (docs-sync feature_themes) still show; judge them, do not suppress them.
cmd_tables=""
for f in cmd/ailang/commands_*.go; do
  case "$f" in *_test.go) ;; *) [ -f "$f" ] && cmd_tables="$cmd_tables $f" ;; esac
done
if [ -n "$cmd_tables" ]; then
  # shellcheck disable=SC2086 # word-splitting the file list is intended
  known=$(grep -hoE '(Name:[[:space:]]*"[^"]+"|Aliases:[[:space:]]*\[\]string\{[^}]*\})' $cmd_tables \
    | grep -oE '"[^"]+"' | tr -d '"'
    # Bare package spellings (`ailang pkg-docs`, `ailang publish`) are generated
    # from the pkgLegacyTopLevel map's VALUES, not written as table rows.
    sed -n '/^var pkgLegacyTopLevel/,/^}/p' cmd/ailang/commands_groups.go 2>/dev/null \
      | grep -oE ':[[:space:]]*"[^"]+"' | grep -oE '"[^"]+"' | tr -d '"')
  known=$(printf '%s\n' "$known" | sort -u)
  find .claude/skills -name '*.md' -type f -print0 2>/dev/null \
    | xargs -0 grep -hoE '(^[[:space:]]*(\$ )?|`|\$\(|[|;&][[:space:]]*)ailang [a-z][a-z0-9-]*[a-z0-9]' 2>/dev/null \
    | grep -oE 'ailang [a-z][a-z0-9-]*[a-z0-9]$' | sort | uniq -c \
    | while read -r n _ c; do
        case "$c" in command|commands) continue ;; esac
        printf '%s\n' "$known" | grep -Fx -- "$c" >/dev/null || printf '    %4d  ailang %s\n' "$n" "$c"
      done | sort -rn | head -20
else
  echo "    (no cmd/ailang/commands_*.go dispatch table found — detector cannot run)"
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
