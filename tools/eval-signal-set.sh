#!/usr/bin/env bash
# eval-signal-set.sh — print the SATURATED benchmark ids from the published
# dashboard ratings (M-EVAL-ELO-PRIORITY-ROTATION), one id per line.
#
# "Saturated" = ELO band "Trivial" in ratings.<mode>.benchmarks[] of the unified
# dashboard JSON (docs/static/benchmarks/latest.json), which blends the cloud
# baseline with the local rotation (eval-report --merge). Callers subtract this
# set from a benchmark list to get the SIGNAL set (non-saturated + unrated).
#
# Usage:
#   tools/eval-signal-set.sh [--json PATH] [--mode agent|standard] [--lang LANG]
#
# --lang prefers the per-language fit (ratings.<mode>.byLang.<lang>) when it
# exists and is non-empty, falling back to the blended top-level list.
#
# FAIL-OPEN CONTRACT: any data problem (missing file, missing jq, absent/renamed
# ratings keys, corrupt JSON) prints NOTHING, notes the reason on stderr, and
# exits 0 — so schedulers degrade to "everything is signal" (today's behavior)
# instead of wedging. Only a usage error exits non-zero.
set -uo pipefail

JSON="docs/static/benchmarks/latest.json"
MODE="agent"
LANG_KEY=""
while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON="${2:?--json needs a path}"; shift 2 ;;
    --mode) MODE="${2:?--mode needs agent|standard}"; shift 2 ;;
    --lang) LANG_KEY="${2:?--lang needs a language}"; shift 2 ;;
    *) echo "usage: $0 [--json PATH] [--mode agent|standard] [--lang LANG]" >&2; exit 2 ;;
  esac
done

if ! command -v jq >/dev/null 2>&1; then
  echo "eval-signal-set: jq not found — emitting empty saturated set (fail-open)" >&2
  exit 0
fi
if [ ! -f "$JSON" ]; then
  echo "eval-signal-set: $JSON not found — emitting empty saturated set (fail-open)" >&2
  exit 0
fi

if ! jq -r --arg mode "$MODE" --arg lang "$LANG_KEY" '
      (.ratings[$mode] // {}) as $m
      | (($m.byLang[$lang].benchmarks // []) as $bl
         | if ($lang != "" and ($bl | length) > 0) then $bl else ($m.benchmarks // []) end)
      | .[] | select(.saturated == true) | .id
    ' "$JSON" 2>/dev/null; then
  echo "eval-signal-set: could not read ratings.$MODE from $JSON — emitting empty saturated set (fail-open)" >&2
  exit 0
fi
