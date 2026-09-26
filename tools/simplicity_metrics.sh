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
# A LEAF imports nothing under the module: it adds no coupling, only code
# (config, statedir, proctree, simhash, strutil, httpjson...). The gated number
# is the NON-leaf count — that is where an agent's edit can ripple.
closure_leaf=0
for pkg in $(printf '%s\n' "$closure" | grep "^$MODULE/internal/"); do
  if ! go list -f '{{join .Imports " "}}' "$pkg" 2>/dev/null | grep "$MODULE/" >/dev/null; then
    closure_leaf=$((closure_leaf + 1))
  fi
done
closure_nonleaf=$((closure_internal - closure_leaf))
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
# The top-level command count is what an agent SEES at the top level: the rows
# the dispatch table renders into `ailang --help`. That is the quantity the
# <=20 gate names ("an agent can read ailang --help in one screen"), so it is
# the quantity measured — hidden rows in `dev`/`ops` do not count, and neither
# do aliases folded onto a visible row.
#
# It used to count `case "..."` labels in cmd/ailang/main.go. M-V1-SIMPLIFY-S5
# M1 replaced that switch with the table, leaving main.go with ZERO case labels
# — so the metric read 0 and the <=20 gate passed vacuously. Worse, under
# `set -euo pipefail` the empty grep exited 1 and killed this whole script:
# `make simplicity-metrics` and `make simplicity-audit` produced no output at
# all. A metric that cannot see its subject does not fail loudly; it reports
# green, or it takes the instrument down with it.
#
# The binary is the source of truth, so build it if there is no current one.
if [ ! -x bin/ailang ] || [ -n "$(find cmd/ailang internal -name '*.go' -newer bin/ailang -print -quit 2>/dev/null)" ]; then
  go build -o bin/ailang ./cmd/ailang >/dev/null 2>&1 || true
fi
if [ -x bin/ailang ]; then
  # Command rows in the generated `Commands:` block: two leading spaces, a name,
  # then its summary. Group footers and blank lines do not match.
  commands_top_level="$(bin/ailang --help 2>/dev/null \
    | sed -n '/^Commands:/,/^$/p' \
    | grep -cE '^  [a-z][a-z0-9-]*( \(|  )' || true)"
else
  echo "simplicity_metrics: cannot build bin/ailang — commands_top_level unmeasurable" >&2
  exit 1
fi
if [ -z "$commands_top_level" ] || [ "$commands_top_level" = "0" ]; then
  echo "simplicity_metrics: commands_top_level measured 0 — the help format changed and this" >&2
  echo "  metric can no longer see its subject. Fix the parser; do not let it report green." >&2
  exit 1
fi
# ---- help reachability -----------------------------------------------------
# The share of routes whose `--help` exits 0. Phase 3's goal sentence is "an
# agent can read `ailang --help` in one screen AND every command answers
# `--help`"; commands_top_level measures the first half, this the second.
# Before S5 M1, eight groups rejected --help outright.
#
# SAFETY, and it is not optional. The first version of tools/check_prompt_commands.sh
# probed with a token it had captured from text, and two faults compounded:
# `[a-z0-9-]+` matches a FLAG (`-` is literal inside a bracket expression), so
# `ailang test --format json` yielded the opener `ailang test --format`; and
# `ailang test --format --help` does not print help, it RUNS THE TEST SUITE.
# Several `ailang test` processes ground for half an hour. So every token here
# is READ OUT OF THE BINARY'S OWN GENERATED HELP and then validated against
# ^[a-z][a-z0-9-]*$ before it reaches argv. Nothing else is ever passed.
help_routes_of() {  # $@: the group path, empty for the top level
  NO_COLOR=1 bin/ailang "$@" --help 2>/dev/null \
    | sed -n '/^Commands:/,/^$/p' \
    | grep -E '^  [a-z][a-z0-9-]*( |$)' \
    | awk '{print $1}'
}

help_token_ok() {  # a bare command name, and nothing else, may reach argv
  case "$1" in
    [a-z]*) ;;
    *) return 1 ;;
  esac
  case "$1" in
    *[!a-z0-9-]*) return 1 ;;
  esac
  return 0
}

help_total=0
help_ok=0
help_failed=""
probe_help() {
  for tok in "$@"; do
    help_token_ok "$tok" || { echo "simplicity_metrics: refusing to probe non-name token '$tok'" >&2; return; }
  done
  help_total=$((help_total + 1))
  if NO_COLOR=1 bin/ailang "$@" --help </dev/null >/dev/null 2>&1; then
    help_ok=$((help_ok + 1))
  else
    help_failed="$help_failed $*"
  fi
}

for c in $(help_routes_of); do probe_help "$c"; done
for g in $(printf '%s\n' dev ops eval); do
  probe_help "$g"
  for s in $(help_routes_of "$g"); do probe_help "$g" "$s"; done
done
# An enumerator that sees nothing reports 100% and passes vacuously — the exact
# failure commands_top_level had. The visible top level alone is 17 rows, so a
# total below 20 means the help format moved and this metric went blind.
if [ "$help_total" -lt 20 ]; then
  echo "simplicity_metrics: help_exit0_rate enumerated only $help_total routes — the help format" >&2
  echo "  changed and this metric can no longer see its subject. Fix the parser; do not report green." >&2
  exit 1
fi
help_exit0_rate=$((help_ok * 100 / help_total))

# Flag names across all FlagSet definitions (distinct), CLI sources only.
#
# This is a CENSUS, not a convergence measure, and the difference cost S5 M5 a
# milestone criterion. The sprint plan asked for this number to DROP when the
# output-format family converged; it read 384 -> 384, and it could not have
# done anything else: D1 keeps every superseded spelling registered until the
# caller sweep, so a release that converges a family ADDS the canonical name
# and removes nothing.
#
# The obvious repair — "exclude the spellings registered via aliasStringFlag" —
# was measured in M6 and rejected: it moves the number by ZERO (the one
# aliasStringFlag call registers "model", which six other sites register too),
# and where it did bite it would exclude the CANONICAL spelling, because the
# helper is what registers the new canonical name beside the old one. Watch
# output_format_spellings below for the family question instead.
#
# _test.go is excluded because a flag registered by a test harness is not CLI
# surface. Measured while writing this: `-update-cli-reference` in
# cli_reference_test.go is the only name the old census counted that no CLI
# invocation can pass, and it alone moved the number 384 -> 385.
flag_names="$(grep -rhoE '\.(String|Int|Bool|Duration|Float64|Int64|Var|StringVar|IntVar|BoolVar|DurationVar)\("[a-zA-Z0-9_-]+"' cmd/ailang --include='*.go' --exclude='*_test.go' 2>/dev/null | grep -oE '"[^"]+"' | sort -u | wc -l | tr -d ' ')"
# Distinct spellings that answer ONE question — "what format is the output" —
# over the family cmd/ailang/output_flags.go records: --json, --format,
# --pretty, --stream-json. This is the number the M5 criterion was reaching for
# and the census cannot show. It is a RATCHET at today's 4: a fifth spelling
# fails the gate, and the number falls to 1 when the caller sweep lets the
# superseded spellings be removed.
output_format_spellings="$(grep -rhoE '\.(String|Bool|StringVar|BoolVar)\("(json|format|pretty|stream-json)"' cmd/ailang --include='*.go' --exclude='*_test.go' 2>/dev/null | grep -oE '"(json|format|pretty|stream-json)"' | sort -u | wc -l | tr -d ' ')"

# ---- configuration routes --------------------------------------------------
# A call, not a string literal: a line where an unclosed `"` precedes the call
# (internal/gen/golang/effects.go's ExampleDoc text) is documentation, not a read.
getenv_all="$(grep -rnE 'os\.(Getenv|LookupEnv)\(' --include='*.go' internal cmd 2>/dev/null | grep -v '_test\.go:' | grep -vE '^[^:]*:[0-9]+:[^"]*"[^"]*os\.(Getenv|LookupEnv)\(' || true)"
getenv_sites_total="$(printf '%s\n' "$getenv_all" | grep -c . || true)"
# internal/testutil is test infrastructure (AILANG_LIVE_NET, AILANG_TEST_FAST_LOOP):
# its switches are opt-ins for the test lane, not configuration of the binary.
getenv_outside_config="$(printf '%s\n' "$getenv_all" | grep -v '^internal/config/' | grep -v '^internal/statedir/' | grep -v '^internal/testutil/' | grep -vE '(Getenv|LookupEnv)\("DEBUG_' | grep -c . || true)"
# The names the binary reads = the literal reads (DEBUG_* knobs, statedir) plus
# every Env* constant in internal/config — since S4, reads go through the
# Registry and a literal-name census would see almost none of them.
env_names="$( { printf '%s\n' "$getenv_all" | grep -oE '(Getenv|LookupEnv)\("[A-Z][A-Z0-9_]*"' | grep -oE '"[^"]+"'; grep -hoE 'Env[A-Za-z0-9]+ *= *"[A-Z][A-Z0-9_]*"' internal/config/*.go 2>/dev/null | grep -oE '"[^"]+"'; } | tr -d '"' | sort -u)"
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
# Independent "which backend" switches: env vars that each SELECT a backend on
# their own. Since S3 M3 the one switch is AILANG_STORAGE (+ per-store
# overrides, which refine it rather than compete); the four retired names are
# refused (read via os.Environ in config, so they no longer count here) and
# COORDINATOR_MODE is validated against the plane, not a selector — excluded.
# Count = 1 (the resolver in internal/config) + every OTHER package that still
# reads one of the names itself. config reads them through constants, so a
# literal-name census would miss the resolver; hence the explicit +1.
backend_readers="$(printf '%s\n' "$getenv_all" | grep -v '^internal/config/' | grep -E '"(AILANG_STORAGE|AILANG_MESSAGES_STORE|AILANG_COORDINATOR_REMOTE|AILANG_CHAINS_READ|AILANG_CHAINS_CLOUD)"' | sed 's/:.*//' | sort -u | wc -l | tr -d ' ' || true)"
backend_switches=$((1 + backend_readers))

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
  --argjson closure_internal "$closure_internal" --argjson closure_leaf "$closure_leaf" --argjson closure_nonleaf "$closure_nonleaf" \
  --argjson closure_leaks "$closure_leaks" --arg closure_leak_list "${closure_leak_list# }" \
  --argjson closure_platform "$closure_platform" --arg closure_platform_list "${closure_platform_list# }" \
  --argjson binary_internal "$binary_internal" --argjson internal_packages "$internal_packages" \
  --argjson internal_loc "$internal_loc" --argjson cmd_loc "$cmd_loc" --argjson cmd_files "$cmd_files" \
  --argjson commands_top_level "$commands_top_level" --argjson flag_names "$flag_names" \
  --argjson output_format_spellings "$output_format_spellings" \
  --argjson help_exit0_rate "$help_exit0_rate" --argjson help_total "$help_total" --arg help_failed "${help_failed# }" \
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
      closure_internal_packages: {value: $closure_internal, gate: null, dir: "le", how: "go list -deps over the language roots (leaves included)"},
      closure_nonleaf_packages:  {value: $closure_nonleaf, gate: 36, dir: "le", how: "closure packages that import something under the module — the coupling that can ripple"},
      closure_leaf_packages:     {value: $closure_leaf, gate: null, dir: "le", how: "closure packages importing nothing under the module"},
      closure_leak_roots:        {value: $closure_leaks, gate: 0, dir: "le", detail: $closure_leak_list, how: "third-party roots {sqlite3, otel sdk/exporters, grpc, websocket, gcp firestore/pubsub/storage/trace} in that closure"},
      closure_platform_packages: {value: $closure_platform, gate: 0, dir: "le", detail: $closure_platform_list, how: "platform packages reachable from the language roots"},
      binary_internal_packages:  {value: $binary_internal, gate: null, dir: "le", how: "go list -deps ./cmd/ailang"},
      internal_packages:         {value: $internal_packages, gate: null, dir: "le", how: "go list ./internal/..."},
      internal_loc:              {value: $internal_loc, gate: null, dir: "le", how: "non-test Go lines under internal/"},
      cmd_ailang_loc:            {value: $cmd_loc, gate: null, dir: "le", how: "non-test Go lines under cmd/ailang"},
      cmd_ailang_files:          {value: $cmd_files, gate: null, dir: "le", how: "non-test Go files under cmd/ailang"},
      commands_top_level:        {value: $commands_top_level, gate: 20, dir: "le", how: "visible command rows in the dispatch table, as rendered into ailang --help (hidden dev/ops rows and folded aliases excluded)"},
      help_exit0_rate:           {value: $help_exit0_rate, gate: 100, dir: "ge", detail: (($help_total|tostring) + " routes probed" + (if $help_failed == "" then "" else "; failing:" + $help_failed end)), how: "share of visible commands, groups and group members whose `--help` exits 0 — bare names read out of the generated help of the binary itself, validated against ^[a-z][a-z0-9-]*$, then probed"},
      flag_names_distinct:       {value: $flag_names, gate: null, dir: "le", how: "distinct FlagSet definition names in cmd/ailang non-test sources — a CENSUS; D1 keeps superseded spellings registered, so a convergence release cannot lower it (see output_format_spellings)"},
      output_format_spellings:   {value: $output_format_spellings, gate: 4, dir: "le", how: "distinct spellings of the ONE output-format question registered in cmd/ailang (json, format, pretty, stream-json — the family cmd/ailang/output_flags.go records); a ratchet, falls to 1 after the caller sweep"},
      getenv_sites_total:        {value: $getenv_sites_total, gate: null, dir: "le", how: "os.Getenv/LookupEnv call sites, non-test"},
      getenv_outside_config:     {value: $getenv_outside_config, gate: 0, dir: "le", how: "same, excluding internal/config, internal/statedir, internal/testutil and DEBUG_* knobs"},
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
