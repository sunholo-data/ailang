#!/bin/bash
# Generated logs only; --only selects a named arm for mutation verification.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/mission_routing_models_check.sh"
SCRATCH="$(mktemp -d "$ROOT/.routing-models-test.XXXXXX")" || exit 2
trap 'rm -rf "$SCRATCH"' EXIT
pass=0
fail=0

expect() {
  local name="$1" want="$2" rc
  shift 2
  /bin/bash "$CHECK" "$@" > "$SCRATCH/out" 2>&1
  rc=$?
  if [ "$rc" -eq "$want" ]; then
    echo "PASS $name rc=$rc"
    pass=$((pass + 1))
  else
    echo "FAIL $name: rc=$rc want=$want"
    cat "$SCRATCH/out"
    fail=$((fail + 1))
  fi
}

fixture() {
  cat > "$SCRATCH/log.md" <<EOF
# Mission log
## 52 — 2026-10-21 — Old routing
Prose mentions gpt-6-luna outside evidence.
**Routing evidence**: base-gate1=1579ef50c@2026-10-21T00:04:58Z.
- **Controller**: \`$1\` (tok: not reported).
- **Planner**: resolver routed to the pin, probe rc 0.
- **Executor**: \`$2\`, 1 run, rc 0 (122,274 tok).
- **Evaluator**: sonnet, foreground (69,103 tok).

**Ruled out**: gpt-5.6-sol is historical prose.
## 53 — 2026-10-21 — Latest routing
**Routing evidence**: base-gate1=9aebccf9b@2026-10-21T00:04:58Z.
- **Controller**: \`codex:gpt-6.1-sol\` (tok: not reported).
- **Executor**: \`gpt6-1-sol\`, rc 0.

**Ruled out**: gpt-6-astra is historical prose.
EOF
}

findings() {
  local name="$1"
  shift
  if [ "$#" -gt 0 ]; then
    printf '%s\n' "$@" > "$SCRATCH/expected"
  else
    : > "$SCRATCH/expected"
  fi
  if cmp -s "$SCRATCH/expected" "$SCRATCH/out"; then
    echo "PASS $name"; pass=$((pass + 1))
  else
    echo "FAIL $name"; cat "$SCRATCH/out"; fail=$((fail + 1))
  fi
}

run_arm() {
  case "$1" in
    a)
      fixture codex:gpt-6.1-sol gpt6-1-sol
      expect 'a canonical allowed aliases' 0 --file "$SCRATCH/log.md" --entry 52
      ;;
    b)
      fixture gpt-6-astra gpt-5.6-luna
      expect 'b retired routing' 1 --file "$SCRATCH/log.md" --entry 52
      if grep -F "$SCRATCH/log.md:5: gpt-6-astra" "$SCRATCH/out" >/dev/null &&
         grep -F "$SCRATCH/log.md:7: gpt-5.6-luna" "$SCRATCH/out" >/dev/null &&
         [ "$(wc -l < "$SCRATCH/out" | tr -d ' ')" -eq 2 ]; then
        echo 'PASS b original lines and every token'; pass=$((pass + 1))
      else
        echo 'FAIL b missing original line/token or extra findings'; cat "$SCRATCH/out"; fail=$((fail + 1))
      fi
      ;;
    c)
      fixture claude:claude-opus-5-5 gemini-4-argon
      expect 'c ignore prose outside routing' 0 --file "$SCRATCH/log.md" --entry 52
      ;;
    d)
      fixture gpt-6-astra gpt6-1-sol
      expect 'd missing entry is instrument failure' 2 --file "$SCRATCH/log.md" --entry 99
      ;;
    e)
      fixture gpt-6-astra gpt6-1-sol
      expect 'e last entry in file order' 0 --file "$SCRATCH/log.md"
      expect 'e explicit non-last entry' 1 --file "$SCRATCH/log.md" --entry 52
      # Numerical order is deliberately different from file order.
      sed 's/^## 53 /## 2 /' "$SCRATCH/log.md" > "$SCRATCH/unordered.md"
      expect 'e file order over numeric order' 0 --file "$SCRATCH/unordered.md"
      ;;
    instruments)
      expect 'missing file' 2 --file "$SCRATCH/missing.md"
      printf '## 52 — 2026-10-21 — No block\nprose\n' > "$SCRATCH/no-block.md"
      expect 'missing block' 2 --file "$SCRATCH/no-block.md"
      expect 'missing arguments' 2
      expect 'unknown argument' 2 --unknown
      expect 'missing file value' 2 --file
      expect 'invalid entry' 2 --file "$SCRATCH/no-block.md" --entry nope
      expect 'missing entry value' 2 --file "$SCRATCH/no-block.md" --entry
      expect 'duplicate file' 2 --file "$SCRATCH/no-block.md" --file "$SCRATCH/no-block.md"
      ;;
    boundaries)
      fixture GPT-6.1-SOL openai/gpt-6.1-sol
      expect 'case and punctuation' 0 --file "$SCRATCH/log.md" --entry 52
      sed '/^\*\*Ruled out\*\*/d' "$SCRATCH/log.md" > "$SCRATCH/no-field.md"
      expect 'entry boundary without next field' 0 --file "$SCRATCH/no-field.md" --entry 52
      printf '## 1 — 2026-10-21 — End of file\n**Routing evidence**: GPT-6-ASTRA.\n' > "$SCRATCH/end.md"
      expect 'inline token at EOF' 1 --file "$SCRATCH/end.md"
      if grep -F "$SCRATCH/end.md:2: GPT-6-ASTRA" "$SCRATCH/out" >/dev/null; then
        echo 'PASS original token case'; pass=$((pass + 1))
      else
        echo 'FAIL original token case'; fail=$((fail + 1))
      fi
      ;;
    w)
      cat > "$SCRATCH/log.md" <<'EOF'
## 1 — 2026-10-21 — World routing
**Routing evidence:**
Controller: gpt-6.1-sol
EOF
      expect 'w world allowed routing' 0 --file "$SCRATCH/log.md"
      findings 'w clean findings'
      sed 's/gpt-6.1-sol/gpt-6-astra/' "$SCRATCH/log.md" > "$SCRATCH/retired.md"
      expect 'w world retired routing' 1 --file "$SCRATCH/retired.md"
      findings 'w original line/token' "$SCRATCH/retired.md:3: gpt-6-astra"
      ;;
    s)
      cat > "$SCRATCH/log.md" <<'EOF'
## 44 — 2026-10-21 — Stapledon routing
- **Routing evidence:**
  - Controller: gpt-6.1-sol
  - Designer: gpt-6-astra
- **Ruled out:** historical prose
EOF
      expect 's indented retired routing' 1 --file "$SCRATCH/log.md" --entry 44
      findings 's original line/token' "$SCRATCH/log.md:4: gpt-6-astra"
      cat > "$SCRATCH/log.md" <<'EOF'
## 44 — 2026-10-21 — Stapledon sibling boundary
- **Routing evidence:**
  - Controller: gpt-6.1-sol
  - Designer: gpt-6-luna
- **Ruled out:** gpt-6-astra
EOF
      expect 's sibling-only retired token excluded' 0 --file "$SCRATCH/log.md" --entry 44
      findings 's clean sibling findings'
      ;;
    v)
      cat > "$SCRATCH/log.md" <<'EOF'
## 1 — 2026-10-21 — V1 routing
**Routing evidence.** Controller gpt-6.1-sol completed planning.
Executor gpt-6-astra completed execution.
EOF
      expect 'v continuation retired routing' 1 --file "$SCRATCH/log.md"
      findings 'v original line/token' "$SCRATCH/log.md:3: gpt-6-astra"
      ;;
    p)
      cat > "$SCRATCH/log.md" <<'EOF'
## 1 — 2026-10-21 — Docs routing
**Routing evidence (from the orphaned fire's PR body, re-verified rather than transcribed)**:
| Role | Model |
| --- | --- |
| Executor | gpt-6-astra |
EOF
      expect 'p parenthetical heading table routing' 1 --file "$SCRATCH/log.md"
      findings 'p original line/token' "$SCRATCH/log.md:5: gpt-6-astra"
      ;;
    f)
      cat > "$SCRATCH/log.md" <<'EOF'
## 1 — 2026-10-21 — Fleet routing
**Routing evidence**:
- **Controller**: gpt-6.1-sol
- **Executor**: gpt-6-astra
EOF
      expect 'f bold-label bullets retained' 1 --file "$SCRATCH/log.md"
      findings 'f original line/token' "$SCRATCH/log.md:4: gpt-6-astra"
      ;;
    q)
      cat > "$SCRATCH/log.md" <<'EOF'
## 44 — 2026-10-21 — Quoted fleet ticket
- **Routing evidence:**
  - Controller: gpt-6.1-sol
  - Ticket: `quorum:gpt6-1-sol-openai-api-credits-exhausted`
EOF
      expect 'q allowed model in ticket compound' 0 --file "$SCRATCH/log.md"
      findings 'q clean ticket findings'
      cat > "$SCRATCH/log.md" <<'EOF'
## 44 — 2026-10-21 — Retired compound
- **Routing evidence:**
  - Executor: gpt-6-astra-unavailable
EOF
      expect 'q retired model in compound' 1 --file "$SCRATCH/log.md"
      findings 'q original compound line/token' "$SCRATCH/log.md:3: gpt-6-astra-unavailable"
      cat > "$SCRATCH/log.md" <<'EOF'
## 44 — 2026-10-21 — Dotted allowed compound
- **Routing evidence:**
  - Executor: gpt-6.1-sol-high
EOF
      expect 'q allowed dotted model in compound' 0 --file "$SCRATCH/log.md"
      findings 'q clean dotted findings'
      ;;
    *) echo "unknown arm: $1" >&2; exit 2 ;;
  esac
}

if [ "$#" -eq 0 ]; then
  for arm in a b c d e instruments boundaries w s v p f q; do run_arm "$arm"; done
elif [ "$#" -eq 2 ] && [ "$1" = --only ]; then
  run_arm "$2"
else
  echo "usage: $0 [--only a|b|c|d|e|instruments|boundaries|w|s|v|p|f|q]" >&2
  exit 2
fi
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
