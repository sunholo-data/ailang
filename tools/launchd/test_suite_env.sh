#!/bin/bash
# Regression checks for the environment boundary around launchd driver suites.
set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
LAUNCHER="$ROOT/tools/launchd/lib/suite-env.sh"
T=$(mktemp -d) || exit 1
trap 'rm -rf "$T"' EXIT
PASS=0
FAIL=0
ok() { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL: $1 ($2)"; }

cat > "$T/env.sh" <<'PROBE'
env
PROBE

# A (kills launcher passing inherited env through): every poison must disappear.
export MISSION_NAME=poison
export MISSION_PLANNER_ALLOWLIST=poison
export AILANG_DRIVER_PINNED=poison
export CONTROLLER_ID=poison
export MODEL=poison
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=core.hooksPath
export GIT_CONFIG_VALUE_0=/nonexistent
/bin/bash "$LAUNCHER" "$T/env.sh" > "$T/env.out"
if grep -Eq '^(MISSION_NAME|MISSION_PLANNER_ALLOWLIST|AILANG_DRIVER_PINNED|CONTROLLER_ID|MODEL|GIT_CONFIG_COUNT|GIT_CONFIG_KEY_0|GIT_CONFIG_VALUE_0)=' "$T/env.out"; then
  bad "A poisoned variables are removed" "poison reached probe"
else
  ok "A poisoned variables are removed"
fi

# B (kills dropping HOME/PATH or exporting empty unset vars): exact values and unset state.
unset LC_ALL
cat > "$T/preserved.sh" <<'PROBE'
printf 'HOME=%s\nPATH=%s\n' "$HOME" "$PATH"
if [ -z "${LC_ALL+x}" ]; then echo 'LC_ALL=UNSET'; else echo 'LC_ALL=SET'; fi
PROBE
/bin/bash "$LAUNCHER" "$T/preserved.sh" > "$T/preserved.out"
if grep -Fqx "HOME=$HOME" "$T/preserved.out" && grep -Fqx "PATH=$PATH" "$T/preserved.out" && grep -Fqx 'LC_ALL=UNSET' "$T/preserved.out"; then
  ok "B HOME/PATH exact and LC_ALL unset"
else
  bad "B HOME/PATH exact and LC_ALL unset" "value or unset state changed"
fi

# C (kills || true and lost return codes): both failing and successful probes.
printf 'exit 7\n' > "$T/exit7.sh"
printf 'exit 0\n' > "$T/exit0.sh"
/bin/bash "$LAUNCHER" "$T/exit7.sh"; rc7=$?
/bin/bash "$LAUNCHER" "$T/exit0.sh"; rc0=$?
if [ "$rc7" -eq 7 ] && [ "$rc0" -eq 0 ]; then ok "C exit codes propagate"; else bad "C exit codes propagate" "rc7=$rc7 rc0=$rc0"; fi

# D (kills dropped args or stdin): the probe consumes both in one invocation.
cat > "$T/args.sh" <<'PROBE'
printf 'ARGS=<%s><%s>\n' "$1" "$2"
IFS= read -r line
printf 'STDIN=<%s>\n' "$line"
PROBE
printf 'input line\n' | /bin/bash "$LAUNCHER" "$T/args.sh" 'first arg' second > "$T/args.out"
if grep -Fqx 'ARGS=<first arg><second>' "$T/args.out" && grep -Fqx 'STDIN=<input line>' "$T/args.out"; then
  ok "D arguments and stdin pass through"
else
  bad "D arguments and stdin pass through" "probe output changed"
fi

# E (kills a vacuous clean-only test): poison must fail directly, then pass cleaned.
export MISSION_PLANNER_ALLOWLIST='tools/launchd/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*'
export MISSION_PLANNER_ANTHROPIC_FALLBACK=codex:gpt-6.1-sol
export TMPDIR="$T"
/bin/bash "$ROOT/tools/launchd/test_mission_routing.sh" > "$T/routing-direct.out" 2>&1
direct_rc=$?
if [ "$direct_rc" -ne 0 ] && grep -Fq 'FAIL: planner falls back to Codex Sol when Anthropic is unavailable' "$T/routing-direct.out"; then
  ok "E direct poisoned control reproduces"
else
  bad "E direct poisoned control reproduces" "control did not reproduce (rc=$direct_rc)"
fi
/bin/bash "$LAUNCHER" "$ROOT/tools/launchd/test_mission_routing.sh" > "$T/routing-clean.out" 2>&1
clean_rc=$?
if [ "$clean_rc" -eq 0 ]; then ok "E cleaned routing suite passes"; else bad "E cleaned routing suite passes" "rc=$clean_rc"; fi

# F (kills a new/raw suite recipe line): suites run only via run-suites.sh, which
# launches each one through suite-env.sh; the target runs no test_*.sh itself.
awk '
  /^LAUNCHD_SUITES :=/ { in_list=1; next }
  in_list { if ($0 ~ /test_[^[:space:]]*\.sh/) listed++; if ($0 !~ /\\$/) in_list=0 }
  /^test-launchd-drivers:/ { in_target=1; next }
  in_target && /^[^#[:space:].][^:]*:/ { exit }
  in_target && /test_[^[:space:]]*\.sh/ && !/\/bin\/bash -n/ { raw++ }
  in_target && /run-suites\.sh \$\(LAUNCHD_JOBS\) \$\(LAUNCHD_SUITES\) --serial \$\(LAUNCHD_SERIAL_SUITES\)/ { runner++ }
  END { if (listed < 40 || raw != 0 || runner != 1) exit 1 }
' "$ROOT/make/test.mk"
if [ "$?" -eq 0 ] && grep -Fq '"$here/suite-env.sh" "$suite"' "$ROOT/tools/launchd/lib/run-suites.sh"; then
  ok "F all suites run via run-suites.sh through the launcher"
else
  bad "F all suites run via run-suites.sh through the launcher" "raw suite line, missing runner call, or runner bypasses suite-env.sh"
fi

# G (kills a runner that drops a failure, a silent worker, or the clean env):
# MISSION_NAME=poison is still exported from A.
RUNNER="$ROOT/tools/launchd/lib/run-suites.sh"
printf 'echo "MISSION_NAME=<${MISSION_NAME-}>"\nsleep 2\n' > "$T/g_pass1.sh"
printf 'sleep 2\necho g2 done\n' > "$T/g_pass2.sh"
printf 'echo g_fail says no\nexit 3\n' > "$T/g_fail.sh"
start=$(date +%s)
/bin/bash "$RUNNER" 2 "$T/g_pass1.sh" "$T/g_pass2.sh" > "$T/g_ok.out" 2>&1; g_ok=$?
g_wall=$(( $(date +%s) - start ))   # overlapped ~2s; one after another >= 4s
/bin/bash "$RUNNER" 2 "$T/g_pass2.sh" --serial "$T/g_fail.sh" > "$T/g_bad.out" 2>&1; g_bad=$?
/bin/bash "$RUNNER" 2 "$T/g_pass2.sh" "$T/no_such_suite.sh" > /dev/null 2>&1; g_missing=$?
if [ "$g_ok" -eq 0 ] && grep -Fq 'all 2 suites passed' "$T/g_ok.out" \
   && grep -Fqx 'MISSION_NAME=<>' "$T/g_ok.out" && [ "$g_wall" -le 3 ] \
   && [ "$g_bad" -eq 1 ] && grep -Fq 'g_fail.sh (rc=3)' "$T/g_bad.out" && grep -Fq 'g_fail says no' "$T/g_bad.out" \
   && [ "$g_missing" -eq 2 ]; then
  ok "G runner: failures and missing suites refuse, env is clean, suites overlap (wall ${g_wall}s)"
else
  bad "G runner: failures and missing suites refuse, env is clean, suites overlap" "ok=$g_ok bad=$g_bad missing=$g_missing wall=${g_wall}s"
fi

echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
