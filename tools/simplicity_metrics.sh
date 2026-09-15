#!/bin/bash
# simplicity_metrics.sh — the release-gate table from
# design_docs/planned/m-v1-simplification-program.md, as one JSON snapshot.
#
# Measures COUNT and DUPLICATION, which the 800-line file-size gate cannot see:
# how much of the repo a language command links, how many routes exist to the
# same setting, how many commands/env-vars/packages an agent has to hold in
# context. Every number here is reproducible from the command that produced it
# (the "how" field in the JSON says which).
#
# Usage:
#   tools/simplicity_metrics.sh                 # print table, bank snapshot
#   tools/simplicity_metrics.sh --no-test       # skip the timed test-core run
#   tools/simplicity_metrics.sh --json          # JSON only, no table
#   tools/simplicity_metrics.sh --out <file>    # bank to <file> instead of
#                                               #   .ailang/state/simplicity/<date>.json
#
# bash 3.2 compatible (the rig has no bash 4): no associative arrays, no ${v,,}.
# macOS awk counts bytes under a UTF-8 locale; every size here is bytes on purpose.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

MODULE="github.com/sunholo-data/ailang"
RUN_TEST=1
JSON_ONLY=0
OUT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --no-test) RUN_TEST=0 ;;
    --json) JSON_ONLY=1 ;;
    --out) shift; OUT="$1" ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 64 ;;
  esac
  shift
done

command -v jq >/dev/null || { echo "jq is required" >&2; exit 69; }
command -v go >/dev/null || { echo "go is required" >&2; exit 69; }

# ---- the language closure -------------------------------------------------
# What `run/check/fmt/prompt/repl` need. Phase 1's closure test asserts on the
# same roots; keep the two lists identical (the test reads this file's list).
LANGUAGE_ROOTS="internal/pipeline internal/eval internal/effects internal/builtins internal/format internal/repl internal/prompt internal/loader internal/link internal/lsp internal/vm internal/gen/golang internal/smt"
# Third-party roots a language binary must not link. Specific on purpose
# (amended 2026-09-15): the OTel API and cloud.google.com/go/auth are legitimate
# for internal/trace and the gemini client; the SDK/exporters and firestore/
# pubsub/storage/trace are the platform. The ollama client SDK is an HTTP client
# library, not a boundary.
LEAK_ROOTS="github.com/mattn/go-sqlite3 go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/exporters github.com/GoogleCloudPlatform/opentelemetry-operations-go google.golang.org/grpc github.com/gorilla/websocket cloud.google.com/go/firestore cloud.google.com/go/pubsub cloud.google.com/go/storage cloud.google.com/go/trace"
# Platform packages the language closure must not reach. internal/ai, secrets,
# mcp_client and auth/gcp measured clean and are part of the language (the AI
# effect is a language feature). internal/platform/* is where the seams live.
PLATFORM_PKGS="internal/platform internal/coordinator internal/observatory internal/storage internal/executor internal/eval_harness internal/messaging"

roots=""
for r in $LANGUAGE_ROOTS; do roots="$roots ./$r"; done
# shellcheck disable=SC2086
closure="$(go list -deps $roots 2>/dev/null | sort -u)"
closure_internal="$(printf '%s\n' "$closure" | grep "^$MODULE/internal/" | wc -l | tr -d ' ')"
closure_leaks=0
closure_leak_list=""
for leak in $LEAK_ROOTS; do
  if printf '%s\n' "$closure" | grep "^$leak" >/dev/null; then
    closure_leaks=$((closure_leaks + 1))
    closure_leak_list="$closure_leak_list $leak"
  fi
done
closure_platform=0
closure_platform_list=""
for p in $PLATFORM_PKGS; do
  if printf '%s\n' "$closure" | grep -E "^$MODULE/$p(\$|/)" >/dev/null; then
    closure_platform=$((closure_platform + 1))
    closure_platform_list="$closure_platform_list $p"
  fi
done
binary_internal="$(go list -deps ./cmd/ailang 2>/dev/null | grep "^$MODULE/internal/" | wc -l | tr -d ' ')"
internal_packages="$(go list ./internal/... 2>/dev/null | wc -l | tr -d ' ')"

# ---- size ------------------------------------------------------------------
loc_of() { find "$@" -name '*.go' ! -name '*_test.go' -print0 2>/dev/null | xargs -0 cat 2>/dev/null | wc -l | tr -d ' '; }
internal_loc="$(loc_of internal)"
cmd_loc="$(loc_of cmd/ailang)"
cmd_files="$(find cmd/ailang -name '*.go' ! -name '*_test.go' | wc -l | tr -d ' ')"

# ---- CLI surface -----------------------------------------------------------
# Until Phase 3 lands a dispatch table, the top-level command count is the
# distinct quoted labels on `case` lines of main.go's command switch. Aliases
# count once each (they are separate routes an agent can take).
commands_top_level="$(grep -oE 'case "[a-z0-9-]+"' cmd/ailang/main.go | sort -u | wc -l | tr -d ' ')"
# Flag names across all FlagSet definitions (distinct).
flag_names="$(grep -rhoE '\.(String|Int|Bool|Duration|Float64|Int64|Var|StringVar|IntVar|BoolVar|DurationVar)\("[a-zA-Z0-9_-]+"' cmd/ailang --include='*.go' 2>/dev/null | grep -oE '"[^"]+"' | sort -u | wc -l | tr -d ' ')"

# ---- configuration routes --------------------------------------------------
getenv_all="$(grep -rnE 'os\.(Getenv|LookupEnv)\(' --include='*.go' internal cmd 2>/dev/null | grep -v '_test\.go:' || true)"
getenv_sites_total="$(printf '%s\n' "$getenv_all" | grep -c . || true)"
# internal/testutil is test infrastructure (AILANG_LIVE_NET, AILANG_TEST_FAST_LOOP):
# its switches are opt-ins for the test lane, not configuration of the binary.
getenv_outside_config="$(printf '%s\n' "$getenv_all" | grep -v '^internal/config/' | grep -v '^internal/testutil/' | grep -vE '(Getenv|LookupEnv)\("DEBUG_' | grep -c . || true)"
env_names="$(printf '%s\n' "$getenv_all" | grep -oE '(Getenv|LookupEnv)\("[A-Z][A-Z0-9_]*"' | grep -oE '"[^"]+"' | tr -d '"' | sort -u)"
env_distinct="$(printf '%s\n' "$env_names" | grep -c . || true)"
doc_sources="CLAUDE.md README.md docs/docs/guides/debugging.md"
for f in .claude/rules/*.md; do doc_sources="$doc_sources $f"; done
[ -f docs/docs/reference/env-vars.md ] && doc_sources="$doc_sources docs/docs/reference/env-vars.md"
env_documented=0
for v in $env_names; do
  # shellcheck disable=SC2086
  if grep -qw -- "$v" $doc_sources 2>/dev/null; then env_documented=$((env_documented + 1)); fi
done
if [ "$env_distinct" -gt 0 ]; then env_documented_pct=$((env_documented * 100 / env_distinct)); else env_documented_pct=0; fi
# Independent "which backend" switches. Shrinks to 1 (+ per-store overrides) in Phase 2.5.
backend_switches=0
for v in AILANG_STORAGE AILANG_MESSAGES_STORE AILANG_COORDINATOR_REMOTE AILANG_CHAINS_READ AILANG_CHAINS_CLOUD COORDINATOR_MODE; do
  if printf '%s\n' "$env_names" | grep -x "$v" >/dev/null; then backend_switches=$((backend_switches + 1)); fi
done

# ---- duplication -----------------------------------------------------------
# Top-level (non-method) func names declared in >= 3 distinct non-test packages.
# Interface-satisfying methods are excluded by construction; `init`/`main`/`New`
# are legitimately repeated.
dup_symbols="$(
  grep -rn --include='*.go' -E '^func [a-zA-Z_][a-zA-Z0-9_]*\(' internal cmd 2>/dev/null \
    | grep -v '_test\.go:' \
    | sed -E 's#^([^:]+)/[^/:]+:[0-9]+:func ([a-zA-Z_][a-zA-Z0-9_]*)\(.*#\2 \1#' \
    | grep -vE '^(init|main|New|Run|Name|String) ' \
    | sort -u \
    | awk '{n[$1]++} END{for(k in n) if(n[k]>=3) print k}' \
    | wc -l | tr -d ' ')"

# ---- discoverability -------------------------------------------------------
pkgs_no_doc=0
pkgs_no_doc_list=""
for d in $(go list -f '{{.Dir}}' ./internal/... 2>/dev/null); do
  rel="${d#"$REPO_ROOT"/}"
  if ! grep -lqE '^// Package [a-zA-Z0-9_]+' "$d"/*.go 2>/dev/null; then
    # test-only packages have no non-test files; skip those
    if find "$d" -maxdepth 1 -name "*.go" ! -name "*_test.go" | grep . >/dev/null; then
      pkgs_no_doc=$((pkgs_no_doc + 1))
      pkgs_no_doc_list="$pkgs_no_doc_list $rel"
    fi
  fi
done

skill_trees=1
if [ -d .agents/skills ] && [ ! -L .agents/skills ]; then
  if ! diff -rq .agents/skills .claude/skills >/dev/null 2>&1; then skill_trees=2; fi
fi

# Always-on instruction surface for a Claude Code session: CLAUDE.md, rules
# with no `paths:` frontmatter, every skill's description line, and the
# auto-memory index if it is where this machine keeps it.
surface=0
surface=$((surface + $(wc -c < CLAUDE.md)))
for f in .claude/rules/*.md; do
  if ! head -5 "$f" | grep '^paths:' >/dev/null; then surface=$((surface + $(wc -c < "$f"))); fi
done
skill_desc_bytes="$(grep -h '^description:' .claude/skills/*/SKILL.md 2>/dev/null | wc -c | tr -d ' ')"
surface=$((surface + skill_desc_bytes))
mem_index="$HOME/.claude/projects/-$(pwd | tr '/' '-' | sed 's/^-//')/memory/MEMORY.md"
mem_bytes=0
[ -f "$mem_index" ] && mem_bytes="$(wc -c < "$mem_index" | tr -d ' ')"
surface_with_memory=$((surface + mem_bytes))

tracked_files="$(git ls-files | wc -l | tr -d ' ')"
root_entries="$(ls -A1 | wc -l | tr -d ' ')"
root_sprint_files="$(find . -maxdepth 1 -name 'sprint[-_]*' | wc -l | tr -d ' ')"
planned_docs="$(find design_docs/planned -name '*.md' | wc -l | tr -d ' ')"

# ---- test-core -------------------------------------------------------------
test_core_seconds=null
if [ "$RUN_TEST" -eq 1 ]; then
  start=$(date +%s)
  if make -s test-core >/dev/null 2>&1; then
    test_core_seconds=$(( $(date +%s) - start ))
  else
    test_core_seconds='"FAILED"'
  fi
fi

# ---- emit ------------------------------------------------------------------
date_tag="$(date +%Y-%m-%d)"
commit="$(git rev-parse --short HEAD)"
[ -z "$OUT" ] && OUT=".ailang/state/simplicity/${date_tag}.json"
mkdir -p "$(dirname "$OUT")"

jq -n \
  --arg date "$date_tag" --arg commit "$commit" \
  --argjson closure_internal "$closure_internal" \
  --argjson closure_leaks "$closure_leaks" --arg closure_leak_list "${closure_leak_list# }" \
  --argjson closure_platform "$closure_platform" --arg closure_platform_list "${closure_platform_list# }" \
  --argjson binary_internal "$binary_internal" --argjson internal_packages "$internal_packages" \
  --argjson internal_loc "$internal_loc" --argjson cmd_loc "$cmd_loc" --argjson cmd_files "$cmd_files" \
  --argjson commands_top_level "$commands_top_level" --argjson flag_names "$flag_names" \
  --argjson getenv_sites_total "$getenv_sites_total" --argjson getenv_outside_config "$getenv_outside_config" \
  --argjson env_distinct "$env_distinct" --argjson env_documented "$env_documented" --argjson env_documented_pct "$env_documented_pct" \
  --argjson backend_switches "$backend_switches" \
  --argjson dup_symbols "$dup_symbols" \
  --argjson pkgs_no_doc "$pkgs_no_doc" --arg pkgs_no_doc_list "${pkgs_no_doc_list# }" \
  --argjson skill_trees "$skill_trees" \
  --argjson surface "$surface" --argjson surface_with_memory "$surface_with_memory" \
  --argjson tracked_files "$tracked_files" --argjson root_entries "$root_entries" --argjson root_sprint_files "$root_sprint_files" \
  --argjson planned_docs "$planned_docs" \
  --argjson test_core_seconds "$test_core_seconds" \
  '{
    date: $date, commit: $commit,
    metrics: {
      closure_internal_packages: {value: $closure_internal, gate: 40, dir: "le", how: "go list -deps over the language roots"},
      closure_leak_roots:        {value: $closure_leaks, gate: 0, dir: "le", detail: $closure_leak_list, how: "third-party roots {sqlite3, otel sdk/exporters, grpc, websocket, gcp firestore/pubsub/storage/trace} in that closure"},
      closure_platform_packages: {value: $closure_platform, gate: 0, dir: "le", detail: $closure_platform_list, how: "platform packages reachable from the language roots"},
      binary_internal_packages:  {value: $binary_internal, gate: null, dir: "le", how: "go list -deps ./cmd/ailang"},
      internal_packages:         {value: $internal_packages, gate: null, dir: "le", how: "go list ./internal/..."},
      internal_loc:              {value: $internal_loc, gate: null, dir: "le", how: "non-test Go lines under internal/"},
      cmd_ailang_loc:            {value: $cmd_loc, gate: null, dir: "le", how: "non-test Go lines under cmd/ailang"},
      cmd_ailang_files:          {value: $cmd_files, gate: null, dir: "le", how: "non-test Go files under cmd/ailang"},
      commands_top_level:        {value: $commands_top_level, gate: 20, dir: "le", how: "distinct case labels in cmd/ailang/main.go until the dispatch table lands"},
      flag_names_distinct:       {value: $flag_names, gate: null, dir: "le", how: "distinct FlagSet definition names in cmd/ailang"},
      getenv_sites_total:        {value: $getenv_sites_total, gate: null, dir: "le", how: "os.Getenv/LookupEnv call sites, non-test"},
      getenv_outside_config:     {value: $getenv_outside_config, gate: 0, dir: "le", how: "same, excluding internal/config, internal/testutil and DEBUG_* knobs"},
      env_vars_distinct:         {value: $env_distinct, gate: null, dir: "le", how: "distinct literal names read"},
      env_vars_documented_pct:   {value: $env_documented_pct, gate: 100, dir: "ge", detail: ($env_documented|tostring), how: "named in CLAUDE.md, .claude/rules, debugging.md, README, reference/env-vars.md"},
      backend_switches:          {value: $backend_switches, gate: 1, dir: "le", how: "independent which-backend env vars still read"},
      dup_symbol_names:          {value: $dup_symbols, gate: null, dir: "le", how: "top-level func names declared in >=3 packages (tools/dup_symbols.sh lists them)"},
      packages_without_doc:      {value: $pkgs_no_doc, gate: 0, dir: "le", detail: $pkgs_no_doc_list, how: "internal packages with no // Package comment"},
      skill_trees:               {value: $skill_trees, gate: 1, dir: "le", how: ".agents/skills present and differing from .claude/skills"},
      instruction_surface_bytes: {value: $surface, gate: 25600, dir: "le", how: "CLAUDE.md + unscoped rules + skill descriptions"},
      instruction_surface_with_memory_bytes: {value: $surface_with_memory, gate: null, dir: "le", how: "plus the MEMORY.md index on this machine"},
      tracked_files:             {value: $tracked_files, gate: 8000, dir: "le", how: "git ls-files"},
      root_entries:              {value: $root_entries, gate: null, dir: "le", how: "entries at repo root incl. untracked"},
      root_sprint_files:         {value: $root_sprint_files, gate: 0, dir: "le", how: "sprint_* / sprint-* files at repo root"},
      planned_design_docs:       {value: $planned_docs, gate: null, dir: "le", how: "markdown files under design_docs/planned"},
      test_core_seconds:         {value: $test_core_seconds, gate: 15, dir: "le", how: "wall time of make test-core"}
    }
  }' > "$OUT"

if [ "$JSON_ONLY" -eq 1 ]; then cat "$OUT"; exit 0; fi

echo "simplicity metrics — $date_tag @ $commit  (banked: $OUT)"
echo
jq -r '
  .metrics | to_entries[] |
  . as $e |
  ($e.value.value|tostring) as $v |
  (if $e.value.gate == null then "—" else ($e.value.gate|tostring) end) as $g |
  (if $e.value.gate == null then "" elif ($e.value.value|type) != "number" then "?"
   elif ($e.value.dir == "le" and $e.value.value <= $e.value.gate) or ($e.value.dir == "ge" and $e.value.value >= $e.value.gate) then "ok" else "OPEN" end) as $s |
  [$e.key, $v, $g, $s] | @tsv' "$OUT" | awk -F'\t' 'BEGIN{printf "%-40s %10s %8s  %s\n","metric","value","gate",""} {printf "%-40s %10s %8s  %s\n",$1,$2,$3,$4}'
