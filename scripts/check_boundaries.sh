#!/usr/bin/env bash
#
# check_boundaries.sh — Architecture boundary gate for AILANG.
#
# Enforces the LOGICAL layering documented in ARCHITECTURE.md over the
# CURRENT physical internal/ tree (no core//apps/ dirs exist yet — that is
# the deferred Phase 4 restructure). See design_docs/planned/v0_7_0/
# m-arch-boundaries.md.
#
# Layers (mapped to real internal/ packages):
#   CORE      = the compiler/runtime: parser, types, eval, core, elaborate,
#               effects, builtins, lexer, ast, pipeline, runtime, link, iface
#   DASHBOARD = the apps/services: server, coordinator, observatory, messaging
#   BRIDGE    = internal/embed (+ internal/runtime) — the ONLY sanctioned path
#               for the dashboard to reach the compiler.
#
# Rules enforced:
#   Rule 1: no CORE package imports any DASHBOARD package.
#   Rule 2: no DASHBOARD package imports the compiler front/middle end
#           (parser/types/eval/core/elaborate/pipeline) DIRECTLY — it must go
#           through internal/embed.
#
# Matching is anchored on quoted Go import paths of the form
#   "github.com/sunholo-data/ailang/internal/<pkg>"
# (NOT a naive un-anchored grep, which never matches real import lines).
#
# Exit 0 = clean. Exit 1 = at least one BOUNDARY VIOLATION (offending
# file:line printed). Exit 2 = usage/setup error.

set -euo pipefail

# Go module path (from go.mod). Do NOT hardcode a guess — this is the anchor
# for every import match, so a wrong value silently makes the gate pass on
# everything (a false pass). Kept as a literal but validated against go.mod below.
MODULE="github.com/sunholo-data/ailang"

# Resolve repo root (script lives in <root>/scripts/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

# Layer package sets (real internal/ dirs only).
CORE_PKGS=(parser types eval core elaborate effects builtins lexer ast pipeline runtime link iface)
DASHBOARD_PKGS=(server coordinator observatory messaging)
# The compiler surface a dashboard package must NOT import directly (must go
# through internal/embed). Deliberately narrower than CORE_PKGS:
#   - runtime/link/iface/etc. are lower-level and not the point of the embed
#     bridge, so they are not policed here.
#   - `eval` is INTENTIONALLY EXCLUDED: internal/embed's OWN public API takes
#     and returns eval.Value (e.g. `embed.ToGo(v eval.Value)`), so a dashboard
#     caller of the sanctioned bridge is forced to name eval.Value. That is the
#     bridge value type, not a behavioral dependency on the evaluator. This is
#     documented in ARCHITECTURE.md > "Architecture boundaries". If embed ever
#     re-exports its own Value alias, add `eval` back here.
CORE_SURFACE_PKGS=(parser types core elaborate pipeline)

# Guard against module-path drift (a wrong MODULE => every match silently
# fails => false pass). Fail loudly instead.
if [ -f go.mod ]; then
  GOMOD_MODULE="$(awk '/^module /{print $2; exit}' go.mod)"
  if [ "$GOMOD_MODULE" != "$MODULE" ]; then
    echo "SETUP ERROR: MODULE ('$MODULE') does not match go.mod ('$GOMOD_MODULE')." >&2
    echo "Update MODULE in scripts/check_boundaries.sh — otherwise the gate is a no-op." >&2
    exit 2
  fi
fi

VIOLATIONS=0

# join_alt a b c -> a|b|c  (for an alternation inside an anchored regex)
join_alt() {
  local IFS='|'
  echo "$*"
}

# search_imports <src-dir> <alternation-of-pkgs> — print matching file:line
# import lines. Anchors on the quoted module import path so only real Go
# import statements match. Empty output = no matches.
search_imports() {
  local src_dir="$1"
  local alt="$2"
  # Regex: a quoted "github.com/sunholo-data/ailang/internal/(pkg1|pkg2|...)"
  # The trailing (/[^"]*)?" allows sub-packages (none today, but future-proof).
  local pattern="\"${MODULE}/internal/(${alt})(/[^\"]*)?\""
  if command -v rg >/dev/null 2>&1; then
    rg --no-heading --line-number --glob '*.go' -e "$pattern" "$src_dir" 2>/dev/null || true
  else
    grep -rEn --include='*.go' -e "$pattern" "$src_dir" 2>/dev/null || true
  fi
}

report_rule() {
  local title="$1"
  local out="$2"
  if [ -n "$out" ]; then
    echo ""
    echo "BOUNDARY VIOLATION: $title"
    echo "$out" | sed 's/^/  /'
    VIOLATIONS=1
  fi
}

echo "Checking architecture boundaries (logical layers over internal/)..."

# Rule 1: CORE must not import DASHBOARD.
dashboard_alt="$(join_alt "${DASHBOARD_PKGS[@]}")"
for pkg in "${CORE_PKGS[@]}"; do
  dir="internal/${pkg}"
  [ -d "$dir" ] || continue
  out="$(search_imports "$dir" "$dashboard_alt")"
  report_rule "core package 'internal/${pkg}' imports a dashboard package (core must never depend on apps)" "$out"
done

# Rule 2: DASHBOARD must not import the compiler surface directly (go via embed).
surface_alt="$(join_alt "${CORE_SURFACE_PKGS[@]}")"
for pkg in "${DASHBOARD_PKGS[@]}"; do
  dir="internal/${pkg}"
  [ -d "$dir" ] || continue
  out="$(search_imports "$dir" "$surface_alt")"
  report_rule "dashboard package 'internal/${pkg}' imports the compiler directly (must go through internal/embed)" "$out"
done

echo ""
if [ "$VIOLATIONS" -ne 0 ]; then
  echo "FAILED: architecture boundary violations found (see above)."
  echo "See ARCHITECTURE.md > 'Architecture boundaries' for the allowed import directions."
  exit 1
fi

echo "OK: no architecture boundary violations."
exit 0
