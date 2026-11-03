#!/usr/bin/env bash
# test_eval_park_set.sh — fixture test for tools/eval-park-set.sh (rotation parking).
# Run: bash tools/launchd/test_eval_park_set.sh
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../.." && pwd)"
T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
fail=0
row() { # row VERSION BENCH MODEL OK CATEGORY [VALID]
  mkdir -p "$T/roll/$1/agent"
  local valid=""; [ "${6:-}" = "invalid" ] && valid=', "validity": {"valid": false}'
  printf '{"id":"%s","lang":"ailang","model":"%s","stdout_ok":%s,"error_category":"%s"%s}\n' \
    "$2" "$3" "$4" "$5" "$valid" > "$T/roll/$1/agent/$2_ailang_$3_$RANDOM$RANDOM.json"
}
M="a,b"
for v in v0.1.0 v0.2.0; do
  row $v dead a false timeout; row $v dead b false timeout          # 4 rows, 2 models, 0 passes -> parked
  row $v once a false timeout; row $v once b false timeout
  row $v alive a true none;   row $v alive b false logic_error      # has a pass -> not parked
done
row v0.2.0 once a true none                                         # one pass -> not parked
row v0.2.0 thin a false timeout; row v0.2.0 thin b false timeout    # only 2 rows -> not parked
for i in 1 2 3 4; do row v0.2.0 solo a false timeout; done          # 1 model only -> not parked
for i in 1 2; do row v0.2.0 crashy a false api_error invalid; row v0.2.0 crashy b false api_error invalid; done
row v0.0.1 old a false timeout; row v0.0.1 old b false timeout      # outside --versions 2 window
row v0.0.1 old a false timeout; row v0.0.1 old b false timeout

got="$(bash "$REPO/tools/eval-park-set.sh" --roll "$T/roll" --models "$M" --versions 2 2>/dev/null | tr '\n' ' ')"
want="dead "
if [ "$got" = "$want" ]; then echo "PASS park set = [$got]"; else echo "FAIL park set: got [$got] want [$want]"; fail=1; fi

got="$(bash "$REPO/tools/eval-park-set.sh" --roll "$T/nope" --models "$M" 2>/dev/null)"; rc=$?
if [ -z "$got" ] && [ "$rc" = 0 ]; then echo "PASS fail-open on missing dir"; else echo "FAIL fail-open: rc=$rc out=[$got]"; fail=1; fi

bash "$REPO/tools/eval-park-set.sh" --bogus >/dev/null 2>&1; rc=$?
if [ "$rc" = 2 ]; then echo "PASS usage error exits 2"; else echo "FAIL usage rc=$rc"; fail=1; fi
exit $fail
