#!/usr/bin/env bash
# test_mission_answer.sh — arms for the attended-ruling writer.
#
# Every arm asserts a SPECIFIC outcome, not merely a non-zero exit: a script that
# refused everything would pass a rc-only suite while being useless.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
SCRIPT_UT=scripts/mission_answer.sh
pass=0; fail=0
ok()   { echo "ok $*";      pass=$((pass+1)); }
notok(){ echo "not ok $*";  fail=$((fail+1)); }

FIX="$(mktemp)"; trap 'rm -f "$FIX"' EXIT
write_fixture() {
	cat > "$FIX" <<'EOF'
# Fixture charter

| D-1 | OPEN | a row OUTSIDE the ledger block that must never be touched | none |

<!-- decision-ledger:start -->
| ID | Status | Decision | Evidence |
|---|---|---|---|
| D-1 | RESOLVED | already answered | prior evidence |
| D-2 | OPEN | should the matcher accept `d == m \|\| strings.HasPrefix(d, m)` as one clause? | measured first-party |
<!-- decision-ledger:end -->
EOF
}

# ARM 1 — happy path rewrites the row, and ONLY that row.
write_fixture
out=$(MISSION_ATTENDED_NAME="Test Human" MISSION_ATTENDED_EMAIL="human@example.com" \
	"$SCRIPT_UT" --id D-2 --answer "A — accept it" --file "$FIX" 2>&1); rc=$?
if [ $rc -eq 0 ]; then ok "1a happy path exits 0"; else notok "1a happy path rc=$rc: $out"; fi
grep -q '^| D-2 | RESOLVED |' "$FIX" && ok "1b D-2 flipped to RESOLVED" || notok "1b D-2 not RESOLVED"
grep -q 'ANSWERED — A — accept it' "$FIX" && ok "1c answer text recorded" || notok "1c answer text missing"
grep -q 'Attended ruling' "$FIX" && ok "1d provenance stamped in evidence" || notok "1d provenance missing"
# The escaped pipe is the arm that catches a naive split: it must survive verbatim.
grep -q 'd == m \\|\\| strings.HasPrefix' "$FIX" && ok "1e escaped pipes preserved" || notok "1e escaped pipes CORRUPTED"
grep -q '^| D-1 | OPEN | a row OUTSIDE' "$FIX" && ok "1f row outside the ledger untouched" || notok "1f row outside the ledger was edited"
grep -q '^| D-1 | RESOLVED | already answered' "$FIX" && ok "1g unrelated resolved row untouched" || notok "1g unrelated row changed"

# ARM 2 — an already-RESOLVED row is refused, and the file is left byte-identical.
write_fixture
before=$(shasum "$FIX" | cut -d' ' -f1)
out=$(MISSION_ATTENDED_NAME="Test Human" MISSION_ATTENDED_EMAIL="human@example.com" \
	"$SCRIPT_UT" --id D-1 --answer "x" --file "$FIX" 2>&1); rc=$?
after=$(shasum "$FIX" | cut -d' ' -f1)
[ $rc -ne 0 ] && ok "2a already-RESOLVED refused" || notok "2a re-answered a resolved row"
[ "$before" = "$after" ] && ok "2b file unmodified on refusal" || notok "2b file mutated on refusal"
echo "$out" | grep -q 'not OPEN' && ok "2c refusal names the reason" || notok "2c refusal reason unclear: $out"

# ARM 3 — an ID that is not in the ledger is refused.
write_fixture
out=$(MISSION_ATTENDED_NAME="Test Human" MISSION_ATTENDED_EMAIL="human@example.com" \
	"$SCRIPT_UT" --id D-99 --answer "x" --file "$FIX" 2>&1); rc=$?
[ $rc -ne 0 ] && ok "3a unknown ID refused" || notok "3a unknown ID accepted"

# ARM 4 — the fleet bot may not author an attended ruling. THE load-bearing arm.
write_fixture
out=$(MISSION_ATTENDED_NAME="Voight-Kampff (bot)" MISSION_ATTENDED_EMAIL="151556158+sunholo-voight-kampff@users.noreply.github.com" \
	"$SCRIPT_UT" --id D-2 --answer "self-resolved" --file "$FIX" 2>&1); rc=$?
[ $rc -ne 0 ] && ok "4a fleet-bot identity refused" || notok "4a THE LOOP COULD RESOLVE ITS OWN ROW"
grep -q '^| D-2 | OPEN |' "$FIX" && ok "4b row still OPEN after bot refusal" || notok "4b bot resolved the row"

# ARM 4c — the EMAIL guard alone, under a human-looking name. Without this arm the
# email check has no killer: arm 4a passes on the name guard by itself.
write_fixture
out=$(MISSION_ATTENDED_NAME="Fleet Operator" MISSION_ATTENDED_EMAIL="151556158+sunholo-voight-kampff@users.noreply.github.com" \
	"$SCRIPT_UT" --id D-2 --answer "self-resolved" --file "$FIX" 2>&1); rc=$?
[ $rc -ne 0 ] && ok "4c fleet-bot EMAIL refused under a human name" || notok "4c EMAIL guard has no effect"
grep -q '^| D-2 | OPEN |' "$FIX" && ok "4d row still OPEN after email refusal" || notok "4d bot email resolved the row"

# ARM 5 — --dry-run reports without writing.
write_fixture
before=$(shasum "$FIX" | cut -d' ' -f1)
MISSION_ATTENDED_NAME="Test Human" MISSION_ATTENDED_EMAIL="human@example.com" \
	"$SCRIPT_UT" --id D-2 --answer "x" --file "$FIX" --dry-run >/dev/null 2>&1
after=$(shasum "$FIX" | cut -d' ' -f1)
[ "$before" = "$after" ] && ok "5a --dry-run does not write" || notok "5a --dry-run wrote"

echo "---"
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
