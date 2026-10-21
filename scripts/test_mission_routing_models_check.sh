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
    *) echo "unknown arm: $1" >&2; exit 2 ;;
  esac
}

if [ "$#" -eq 0 ]; then
  for arm in a b c d e instruments boundaries; do run_arm "$arm"; done
elif [ "$#" -eq 2 ] && [ "$1" = --only ]; then
  run_arm "$2"
else
  echo "usage: $0 [--only a|b|c|d|e|instruments|boundaries]" >&2
  exit 2
fi
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
