#!/usr/bin/env bash
# test_rig_model_health.sh — fixture test for tools/rig-model-health.sh (dark-model alarm).
# Run: bash tools/launchd/test_rig_model_health.sh   (HEALTH=path overrides the script under test)
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../.." && pwd)"
HEALTH="${HEALTH:-$REPO/tools/rig-model-health.sh}"
T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
fail=0
row() { # row DIR MODEL N [invalid]
  mkdir -p "$1"; local valid=""; [ "${4:-}" = "invalid" ] && valid=', "validity": {"valid": false}'
  printf '{"id":"b%s","lang":"ailang","model":"%s","stdout_ok":true%s}\n' "$3" "$2" "$valid" > "$1/b$3_ailang_$2_$RANDOM.json"
}
check() { # NAME DIR MODELS WANT_RC WANT_DARK
  local out rc; out="$(bash "$HEALTH" --vdir "$2" --models "$3" 2>/dev/null)"; rc=$?
  local dark; dark="$(printf '%s\n' "$out" | sed -n 's/^DARK //p' | tr '\n' ' ')"
  if [ "$rc" = "$4" ] && [ "$dark" = "$5" ]; then echo "PASS $1"; else echo "FAIL $1: rc=$rc dark=[$dark] want rc=$4 dark=[$5]"; fail=1; fi
}
for i in 1 2 3; do row "$T/dark" a $i; row "$T/dark" b $i; done
check "silent model is DARK" "$T/dark" a,b,c 3 "c "
check "all models banking is healthy" "$T/dark" a,b 0 ""
row "$T/fresh" a 1
check "fresh bank (peer < min-peer) is not an alarm" "$T/fresh" a,b 0 ""
mkdir -p "$T/empty"
check "empty bank is not an alarm" "$T/empty" a,b 0 ""
for i in 1 2 3; do row "$T/crash" a $i; row "$T/crash" b $i invalid; done
check "crash-only rows count as dark" "$T/crash" a,b 3 "b "
bash "$HEALTH" --vdir "$T/dark" >/dev/null 2>&1; rc=$?
if [ "$rc" = 2 ]; then echo "PASS usage error exits 2"; else echo "FAIL usage rc=$rc"; fail=1; fi
exit $fail
