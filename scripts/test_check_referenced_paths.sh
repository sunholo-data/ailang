#!/bin/bash
# Self-test for the referenced-path gate. Bash 3.2 only.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_referenced_paths.sh"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/test-check-referenced-paths.XXXXXX") || exit 1
UNTRACKED="$REPO_ROOT/scripts/iter321_untracked_fixture.sh"
trap 'rm -rf "$WORK"; rm -f "$UNTRACKED"' EXIT HUP INT TERM

PASSED=0
FAILED=0
OUT=""
RC=0

pass() { echo "  PASS — $1"; PASSED=$((PASSED + 1)); }
fail() { echo "  FAIL — $1"; FAILED=$((FAILED + 1)); }

run_gate() {
	OUT=$(/bin/bash "$GATE" --scan-root "$1" --repo-root "$REPO_ROOT" 2>&1)
	RC=$?
}

make_floor_fixture() {
	dir="$1"
	mkdir -p "$dir/make" "$dir/.github/workflows"
	cat > "$dir/Makefile" <<'EOF'
check01: ; @bash scripts/check_autoclose.sh
check02: ; @bash scripts/check_boundaries.sh
check03: ; @bash scripts/check_changelog.sh
check04: ; @bash scripts/check_git_exec.sh
check05: ; @bash scripts/check_home_isolation.sh
check06: ; @bash scripts/check_pi_wire_budget.sh
check07: ; @bash scripts/check_protocol_closure.sh
check08: ; @bash scripts/check_skills.sh
check09: ; @bash scripts/check_tmpfile_hygiene.sh
check10: ; @bash scripts/mission_decisions.sh
check11: ; @bash scripts/test_check_autoclose.sh
check12: ; @bash scripts/test_check_changelog.sh
check13: ; @bash scripts/test_check_git_exec.sh
check14: ; @bash scripts/test_check_home_isolation.sh
check15: ; @bash scripts/test_check_protocol_closure.sh
check16: ; @bash scripts/test_check_tmpfile_hygiene.sh
check17: ; @bash tools/ci/motoko_smoke.sh
check18: ; @bash tools/eval/motoko_connection_probe.sh
check19: ; @bash tools/eval/test_motoko_connection_probe.sh
check20: ; @bash tools/freeze-stdlib.sh
EOF
}

echo "referenced-path gate:"

# Kills a gate that cannot pass on the repository it is meant to protect.
OUT=$(cd "$REPO_ROOT" && /bin/bash "$GATE" 2>&1)
RC=$?
if [ "$RC" -eq 0 ] && printf '%s\n' "$OUT" | grep -q 'referenced-paths: checked'; then
	pass "A1 clean tree passes"
else
	fail "A1 clean tree passes (rc=$RC)"
fi

# Kills today's defect: a make recipe retains a path after its file is deleted.
A2="$WORK/a2"
make_floor_fixture "$A2"
printf '%s\n' 'missing: ; @bash tools/launchd/does_not_exist.sh' >> "$A2/Makefile"
run_gate "$A2"
if [ "$RC" -ne 0 ] && printf '%s\n' "$OUT" | grep -qF 'tools/launchd/does_not_exist.sh'; then
	pass "A2 missing referenced path is refused"
else
	fail "A2 missing referenced path is refused (rc=$RC)"
fi

# Kills an enumerator that checks known make shapes but never looks at workflow additions.
A3="$WORK/a3"
make_floor_fixture "$A3"
run_gate "$A3"
BASE_COUNT=$(printf '%s\n' "$OUT" | sed -n 's/^referenced-paths: enumerated \([0-9][0-9]*\) paths$/\1/p')
cat > "$A3/.github/workflows/addition.yml" <<'EOF'
name: addition
jobs:
  probe:
    steps:
      - run: bash tools/launchd/workflow_only_missing.sh
EOF
run_gate "$A3"
MOVED_COUNT=$(printf '%s\n' "$OUT" | sed -n 's/^referenced-paths: enumerated \([0-9][0-9]*\) paths$/\1/p')
if [ "$RC" -ne 0 ] && [ "$MOVED_COUNT" -eq $((BASE_COUNT + 1)) ] && printf '%s\n' "$OUT" | grep -qF 'tools/launchd/workflow_only_missing.sh'; then
	pass "A3 workflow addition is enumerated and refused"
else
	fail "A3 workflow addition moves count (base=${BASE_COUNT:-?}, moved=${MOVED_COUNT:-?}, rc=$RC)"
fi

# Kills the unanchored-substring bug that invented paths inside longer path tokens.
A4="$WORK/a4"
make_floor_fixture "$A4"
printf '%s\n' 'noise: prompts/devtools/versions.json docs/scripts/sync-prompts.sh' >> "$A4/Makefile"
run_gate "$A4"
if [ "$RC" -eq 0 ] && ! printf '%s\n' "$OUT" | grep -qE 'tools/versions\.js|scripts/sync-prompts\.sh'; then
	pass "A4 path anchoring rejects substring false positives"
else
	fail "A4 path anchoring rejects substring false positives (rc=$RC)"
fi

# Kills a vacuous green when the input set disappears or the matcher breaks.
A5="$WORK/a5"
mkdir -p "$A5"
run_gate "$A5"
if [ "$RC" -ne 0 ] && printf '%s\n' "$OUT" | grep -qF 'instrument failure: enumeration returned 0 paths' && ! printf '%s\n' "$OUT" | grep -q 'referenced-paths: checked'; then
	pass "A5 anti-vacuity floor refuses zero paths"
else
	fail "A5 anti-vacuity floor refuses zero paths (rc=$RC)"
fi

# Kills the works-on-my-machine case where an untracked local file masks deletion.
A6="$WORK/a6"
make_floor_fixture "$A6"
printf '%s\n' '#!/bin/bash' > "$UNTRACKED"
printf '%s\n' 'untracked: ; @bash scripts/iter321_untracked_fixture.sh' >> "$A6/Makefile"
run_gate "$A6"
if [ "$RC" -ne 0 ] && printf '%s\n' "$OUT" | grep -qF 'untracked referenced path: scripts/iter321_untracked_fixture.sh'; then
	pass "A6 untracked-but-present path is refused"
else
	fail "A6 untracked-but-present path is refused (rc=$RC)"
fi

echo "==== $PASSED passed, $FAILED failed ===="
[ "$FAILED" -eq 0 ]
