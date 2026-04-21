#!/usr/bin/env bash
# eval_lib.sh — shared helpers for eval-related skills.
#
# Source this from other skill scripts:
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   # shellcheck disable=SC1091
#   . "$SCRIPT_DIR/../../_shared/scripts/eval_lib.sh"
#
# After sourcing you can call:
#   dev_models_csv           # -> "gpt5-1-instant,claude-haiku-4-5,gemini-3-flash"
#   extended_models_csv      # -> 5-model extended set
#   canonical_tags_csv       # -> 12-tag taxonomy
#   resolve_ailang_bin       # -> path to ailang binary (ailang on PATH or bin/ailang)
#   ensure_repo_root         # cd to the repo root (looks for go.mod + Makefile)
#
# All helpers are idempotent and have no side effects other than echoing values.

_eval_lib_repo_root() {
  # Walk up from CWD looking for go.mod + Makefile (AILANG repo markers).
  local dir="$PWD"
  while [ "$dir" != "/" ]; do
    if [ -f "$dir/go.mod" ] && [ -f "$dir/Makefile" ] && [ -d "$dir/internal/eval_harness" ]; then
      echo "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

ensure_repo_root() {
  local root
  if ! root="$(_eval_lib_repo_root)"; then
    echo "eval_lib: not inside an AILANG repo (no go.mod + internal/eval_harness)" >&2
    return 1
  fi
  cd "$root" || return 1
}

# Parse a top-level list key out of models.yml.
# Usage: _eval_lib_yaml_list <key>
# The YAML format we rely on is:
#   key:
#     - "value1"   # optional comment
#     - "value2"
# where the first non-`- ` line (or blank line) ends the list.
_eval_lib_yaml_list() {
  local key="$1"
  local yml
  yml="$(_eval_lib_repo_root)/internal/eval_harness/models.yml" || return 1
  [ -f "$yml" ] || { echo "eval_lib: models.yml not found at $yml" >&2; return 1; }
  awk -v key="$key" '
    $0 ~ "^"key":"            { in_block=1; next }
    in_block && /^[[:space:]]*-/ {
      # strip leading "- ", quotes, and trailing comment
      s=$0
      sub(/^[[:space:]]*-[[:space:]]*/, "", s)
      sub(/#.*$/, "", s)
      gsub(/"/, "", s)
      gsub(/[[:space:]]+$/, "", s)
      if (s != "") print s
      next
    }
    in_block && /^[^[:space:]-]/ { in_block=0 }
  ' "$yml"
}

dev_models_csv() {
  _eval_lib_yaml_list "dev_models" | paste -sd ',' -
}

extended_models_csv() {
  _eval_lib_yaml_list "extended_suite" | paste -sd ',' -
}

# Canonical 12-tag taxonomy from internal/eval_harness/spec.go (ValidTagTaxonomy).
# Single source of truth — if this drifts, the eval harness will reject benchmarks
# and the mismatch becomes obvious at load time.
canonical_tags_csv() {
  echo "adt_pattern_match,algorithmic,contracts,data_transform,effects_io,error_handling,functional,records,recursion,state_machine,string_algo,type_safety"
}

# Validate that a comma-separated tag list uses only canonical tags.
# Usage: validate_tags "contracts,records,functional"  # -> 0 if all valid
validate_tags() {
  local tags="$1"
  local canon
  canon="$(canonical_tags_csv)"
  local bad=""
  local t
  IFS=',' read -ra arr <<< "$tags"
  for t in "${arr[@]}"; do
    t="$(echo "$t" | tr -d '[:space:]')"
    [ -z "$t" ] && continue
    if [[ ",$canon," != *",$t,"* ]]; then
      bad="$bad $t"
    fi
  done
  if [ -n "$bad" ]; then
    echo "eval_lib: non-canonical tag(s):$bad" >&2
    echo "eval_lib: canonical tags: $canon" >&2
    return 1
  fi
}

resolve_ailang_bin() {
  if command -v ailang >/dev/null 2>&1; then
    echo "ailang"
    return 0
  fi
  local root
  root="$(_eval_lib_repo_root)" || return 1
  if [ -x "$root/bin/ailang" ]; then
    echo "$root/bin/ailang"
    return 0
  fi
  echo "eval_lib: ailang not found on PATH and bin/ailang not built" >&2
  return 1
}
