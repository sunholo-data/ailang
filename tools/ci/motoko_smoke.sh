#!/usr/bin/env bash
# M-RIG-RELIABILITY guardrail: catch AILANG changes that break real-world CONSUMER code at PR
# time, instead of silently bricking the eval rig for hours (the v0.27.0 \uXXXX lexer break
# stalled the motoko harness ~35h with zero CI signal). See design_docs/planned/m-eval-rig-reliability.md.
#
#   (1) ALWAYS: ailang check the vendored consumer corpus (parser/type/effect regression net).
#   (2) IF PRESENT: ailang check the real motoko .ail core (rig/dev machine; not cloud CI).
set -uo pipefail
cd "$(dirname "$0")/../.."
fail=0
noise='stale|quick-install|version mismatch|MOD010|Running under|omit --relax'

CORPUS=internal/eval_harness/testdata/consumer_smoke.ail
echo "== consumer corpus: $CORPUS =="
if ailang check --relax-modules "$CORPUS" 2>&1 | grep -q "No errors found"; then
  echo "  OK — corpus compiles"
else
  echo "  FAIL — an AILANG change broke real-world consumer constructs:"
  ailang check --relax-modules "$CORPUS" 2>&1 | grep -vE "$noise" | grep -iE "error|PAR|TC|expected" | head -8
  fail=1
fi

MOTOKO="${MOTOKO_REPO:-$HOME/dev/mk-ast}"
if [ -f "$MOTOKO/src/core/supervisor.ail" ]; then
  echo "== motoko core: $MOTOKO/src/core/supervisor.ail =="
  if (cd "$MOTOKO" && ailang check src/core/supervisor.ail 2>&1 | grep -q "No errors found"); then
    echo "  OK — motoko core compiles"
  else
    echo "  FAIL — this AILANG build would brick the rig (motoko core no longer compiles):"
    (cd "$MOTOKO" && ailang check src/core/supervisor.ail 2>&1 | grep -vE "$noise" | grep -iE "error|PAR|TC|expected" | head -8)
    fail=1
  fi
else
  echo "== motoko core: skipped (fork not at $MOTOKO — cloud CI relies on the corpus above) =="
fi

[ "$fail" = 0 ] && echo "GUARDRAIL: PASS" || echo "GUARDRAIL: FAIL"
exit $fail
