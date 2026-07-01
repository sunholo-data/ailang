#!/usr/bin/env bash
#
# verify_examples.sh — gate against rotted examples (M-SNAKE-FEEDBACK / M3).
#
# WHY THIS EXISTS: 8 examples shipped using `++` on strings (list-only since
# v0.13.0) and never ran; 2 more were missing `import std/io`. None were caught
# because nothing gated examples in CI — examples_report.json is generated for
# the MCP snapshot, not as a fail-the-build check. A 0→1 user copies a broken
# example and concludes the language is broken. This closes that gap.
#
# CONTRACT:
#   * EVERY top-level examples/*.ail MUST type-check (`ailang check`).
#   * Runnable examples MUST run without "Error:" (we grep output, not just the
#     exit code — module evaluation errors currently exit 0, a separate bug).
#   * AI/network examples and KNOWN runtime bugs are skipped from the RUN phase
#     but still type-checked — each skip is LOGGED with a reason (no silent caps).
#
# Usage: ./tools/verify_examples.sh [--verbose]
set -uo pipefail

VERBOSE=0
[ "${1:-}" = "--verbose" ] && VERBOSE=1

AILANG="./bin/ailang"
if [ ! -x "$AILANG" ]; then
  AILANG="$(command -v ailang || true)"
fi
if [ -z "$AILANG" ]; then
  echo "error: ailang binary not found (run 'make build')" >&2
  exit 2
fi

# Reason a top-level example is skipped from the RUN phase (still type-checked).
# Empty = not skipped. Keep this list SHORT and each entry justified; a skipped
# example is one we are knowingly not running, so it must point at a real cause.
run_skip_reason() {
  case "$1" in
    ai_modes)              echo "needs AI capability + live API key (network)";;
    ai_openrouter_routing) echo "needs AI capability + OpenRouter key (network)";;
    effect_budget_demo)    echo "KNOWN BUG: IO @limit is cumulative across the call chain, not per-function — budget-scoping follow-up";;
    *)                     echo "";;
  esac
}

fail=0; checked=0; ran=0; skipped=0
for f in examples/*.ail; do
  base="$(basename "$f" .ail)"

  # 1. Type-check (required for ALL top-level examples).
  if ! cout="$("$AILANG" check "$f" 2>&1)"; then
    echo "✗ TYPE-CHECK FAIL: $f"
    echo "$cout" | grep -iE 'error' | head -3 | sed 's/^/    /'
    fail=1
    continue
  fi
  checked=$((checked + 1))

  # 2. Run (unless explicitly skipped).
  reason="$(run_skip_reason "$base")"
  if [ -n "$reason" ]; then
    echo "⊘ run-skip  $base — $reason"
    skipped=$((skipped + 1))
    continue
  fi

  rout="$("$AILANG" run --caps IO,FS,Clock,Rand --entry main "$f" 2>&1)"
  rc=$?
  if [ $rc -ne 0 ] || echo "$rout" | grep -q 'Error:'; then
    echo "✗ RUN FAIL: $f"
    echo "$rout" | grep -iE 'error' | head -3 | sed 's/^/    /'
    fail=1
    continue
  fi
  ran=$((ran + 1))
  [ $VERBOSE -eq 1 ] && echo "✓ $base"
done

echo ""
echo "examples: $checked type-checked · $ran ran clean · $skipped run-skipped (logged above)"
if [ $fail -eq 0 ]; then
  echo "✓ verify-examples PASSED"
else
  echo "✗ verify-examples FAILED — fix the example or migrate off a broken pattern"
fi
exit $fail
