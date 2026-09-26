#!/bin/bash
# Mission scope guard (M-HARNESS-MISSION-LOOP M2): verdict table + a REAL push through an
# env-enabled core.hooksPath, which is how the driver turns the guard on per fire.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
HOOK="$HERE/githooks/pre-push"
DRIVER="$HERE/mission-control.sh"
fail() { echo "FAIL $*"; exit 1; }

MISSION_SCOPE_LIB=1 . "$HOOK"
check() {  # mission path want(allow|refuse)
  local got; got=$(_scope_verdict "$1" "$2"); got=${got%% *}; [ -n "$got" ] || got=allow
  [ "$got" = "$3" ] || fail "verdict MISSION_NAME='$1' $2: got $got want $3"
}
check v1 tools/launchd/mission-control.sh refuse
check docs internal/mission/ticket.go refuse
check motoko .claude/skills/mission-control/SKILL.md refuse
check v1 .claude/skills/sprint-executor/SKILL.md refuse
check v1 missions/v1.toml refuse
check v1 .pi/extensions/prepush-gate.ts refuse
check v1 cmd/ailang/mission_ticket.go refuse
check v1 internal/parser/parser.go allow
check v1 design_docs/v1-mission-log.md allow
check v1 .claude/skills/use-ailang/SKILL.md allow
check fleet tools/launchd/mission-control.sh allow
check fleet internal/types/unify.go refuse
check fleet std/list.ail refuse
check fleet design_docs/fleet-mission-log.md allow
check "" tools/launchd/mission-control.sh allow
echo 'PASS scope verdicts: product loops refused on harness paths, fleet refused on core, attended unaffected'

# End to end: git must actually run the hook when core.hooksPath comes ONLY from env.
T=$(mktemp -d "${TMPDIR:-/tmp}/scope-guard.XXXXXX"); trap 'rm -rf "$T"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
git init -q --bare "$T/origin.git"
git clone -q "$T/origin.git" "$T/w" 2>/dev/null
cd "$T/w" || fail "clone"
git -c user.name=t -c user.email=t@t commit -q --allow-empty -m base
git push -q origin HEAD:refs/heads/dev 2>/dev/null
git fetch -q origin
mkdir -p tools/launchd && echo x > tools/launchd/x.sh && git add . \
  && git -c user.name=t -c user.email=t@t commit -q -m harness
push() {  # MISSION_NAME → rc of a push with the guard enabled via env only
  env GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0="$HERE/githooks" \
    MISSION_SCOPE_TEST_REMOTE="$T/origin.git" MISSION_NAME="$1" \
    git push -q origin HEAD:refs/heads/"$2" 2>"$T/err"
}
push v1 b1 && fail "v1 push of tools/launchd/ was NOT refused (hook did not fire?)"
grep -q 'ailang mission ticket file' "$T/err" || fail "refusal does not name the ticket command: $(cat "$T/err")"
push fleet b2 || fail "fleet push of a harness path was refused: $(cat "$T/err")"
push "" b3 || fail "attended push (no MISSION_NAME) was refused"
# Control: without the env hooksPath the same v1 push goes through — proves the env is the switch.
MISSION_NAME=v1 MISSION_SCOPE_TEST_REMOTE="$T/origin.git" git push -q origin HEAD:refs/heads/b4 2>/dev/null \
  || fail "control: push without the env hooksPath should not run the guard"
echo 'PASS end-to-end: env-only core.hooksPath runs the guard; v1 refused, fleet and attended pass, control unguarded'

# The driver must enable it, below the controller spawn's env and pointing at this directory.
grep -q 'GIT_CONFIG_VALUE_.*tools/launchd/githooks' "$DRIVER" || fail "driver does not enable the scope guard"
echo 'PASS driver wires the scope guard'
