#!/bin/bash
# Mission scope guard (M-HARNESS-MISSION-LOOP M2): verdict table + a REAL push through an
# env-enabled core.hooksPath, which is how the driver turns the guard on per fire.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
HOOK="$HERE/githooks/pre-push"
DRIVER="$HERE/mission-control.sh"
fail() { echo "FAIL $*"; exit 1; }
# Scrub inherited git env config. Inside a mission fire the DRIVER exports GIT_CONFIG_* to turn
# this very guard on; inherited here, it guarded the "unguarded control" push and the suite read
# red at base (found by fleet iteration 1, 2026-09-26).
unset GIT_CONFIG_COUNT
for _v in $(env | grep -E '^GIT_CONFIG_(KEY|VALUE)_[0-9]+=' | cut -d= -f1); do unset "$_v"; done
unset _v

# shellcheck source=tools/launchd/githooks/pre-push
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
check v1 design_docs/fleet-mission.md refuse
check docs design_docs/fleet-mission-log.md refuse
check fleet .github/workflows/ci.yml refuse
check fleet go.mod refuse
check fleet internal/server/api.go refuse
check fleet cmd/ailang/main.go refuse
check fleet changelogs/v0.32-current.md allow
check fleet design_docs/planned/m-harness-mission-loop.md allow
check fleet docs/docs/guides/mission-bootstrap.md allow
check fleet internal/config/mission.go allow
check fleet make/test.mk allow
check fleet .ailang/state/sprints/sprint_X.json allow
check fleet scripts/test_mission_pi_run.sh allow
check v1 scripts/test_mission_pi_run.sh refuse
check fleet tools/pi-extensions/sandbox/index.ts allow
check fleet tools/pi-extensions/ailang-exec/index.ts refuse
check v1 tools/pi-extensions/sandbox/index.ts allow
# D-FLEET-18: exact support paths; product loops retain access.
check fleet tools/eval/motoko_connection_probe.sh allow # kills: drop path 1
check fleet tools/eval/test_motoko_connection_probe.sh allow # kills: drop path 2
check motoko tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
check motoko tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
check v1 tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
check v1 tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
check fleet tools/eval/qwen38-motoko-rerun.sh refuse # kills: widen to tools/eval/*
check fleet tools/eval/motoko_connection_probe.sh.bak refuse # kills: path 1 suffix glob
check fleet tools/eval/test_motoko_connection_probe.sh-x refuse # kills: path 2 suffix glob
check fleet tools/eval/xmotoko_connection_probe.sh refuse # kills: path 1 basename-prefix glob
check fleet tools/eval/xtest_motoko_connection_probe.sh refuse # kills: path 2 basename-prefix glob
check fleet scripts/motoko_connection_probe.sh refuse # kills: path 1 directory wildcard
check fleet scripts/test_motoko_connection_probe.sh refuse # kills: path 2 directory wildcard
# D-FLEET-36: both mirrors, including nested files.
for side in .claude .agents; do
  check fleet "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # kills: delete this side's design-doc-creator pattern
  check v1 "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # kills: move the pattern into _scope_is_harness
done
check fleet .claude/skills/design-doc-creator-x/SKILL.md refuse # kills: design-doc-creator/* -> design-doc-creator*
check fleet .agents/skills/design-doc-creator-x/SKILL.md refuse # kills: design-doc-creator/* -> design-doc-creator*
check fleet .claude/skills/use-ailang/SKILL.md refuse # kills: widen to .claude/skills/* (the D-FLEET-36 arm names one skill)
# D-FLEET-38: every enumerated arm, product loops unaffected, no over-widening.
D38=' benchmark-runner builtin-developer cloud-cascade-debug collaboration-hub coordinator-helper
  eval-analyzer eval-gap-finder gemini-cli-helper github-issue-triage local-model-onboarding
  local-ollama-eval parser-developer pr-monitor prompt-manager trace-debugger use-ailang '
for skill in $D38; do
  check fleet ".agents/skills/$skill/SKILL.md" allow # kills: delete this skill's arm
  check v1 ".agents/skills/$skill/SKILL.md" allow # kills: move this skill's arm into _scope_is_harness
  check fleet ".agents/skills/$skill-x/SKILL.md" refuse # kills: drop this arm's / boundary (<name>* admits siblings)
done
check fleet .agents/skills/model-manager/SKILL.md refuse # kills: replace the arms with .agents/skills/*
check fleet .agents/skills/use-ailang-extra/SKILL.md refuse # kills: use-ailang/* -> use-ailang*
check fleet .claude/skills/use-ailang/SKILL.md refuse # kills: also admit .claude/skills/use-ailang/*
# Exactly 16: every other .agents skill in the tree stays refused (mission-*/sprint-* are
# harness, design-doc-creator is D-FLEET-36). The count floor keeps the loop from passing empty.
n=0
for d in "$HERE"/../../.agents/skills/*/; do
  skill=$(basename "$d")
  case "$skill" in mission-*|sprint-*|design-doc-creator) continue ;; esac
  case "$D38" in *[[:space:]]"$skill"[[:space:]]*) continue ;; esac
  check fleet ".agents/skills/$skill/SKILL.md" refuse # kills: admit a 17th .agents skill
  n=$((n+1))
done
[ "$n" -ge 10 ] || fail "only $n non-admitted .agents skills enumerated; the exactly-16 rows ran (nearly) empty"
echo 'PASS scope verdicts: product loops refused on harness paths (incl. the fleet charter), fleet held to its allowlist, attended unaffected'

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
push() {  # MISSION_NAME BRANCH [push options] → rc with the guard enabled via env only
  local mission="$1" branch="$2"
  shift 2
  env GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0="$HERE/githooks" \
    MISSION_SCOPE_TEST_REMOTE="$T/origin.git" MISSION_NAME="$mission" TMPDIR="$T" \
    git push -q "$@" origin "HEAD:refs/heads/$branch" 2>"$T/err"
}
push v1 b1 && fail "v1 push of tools/launchd/ was NOT refused (hook did not fire?)"
grep -q 'ailang mission ticket file' "$T/err" || fail "refusal does not name the ticket command: $(cat "$T/err")"
push fleet b2 || fail "fleet push of a harness path was refused: $(cat "$T/err")"
push "" b3 || fail "attended push (no MISSION_NAME) was refused"
# Control: without the env hooksPath the same v1 push goes through — proves the env is the switch.
MISSION_NAME=v1 MISSION_SCOPE_TEST_REMOTE="$T/origin.git" git push -q origin HEAD:refs/heads/b4 2>/dev/null \
  || fail "control: push without the env hooksPath should not run the guard"
echo 'PASS end-to-end: env-only core.hooksPath runs the guard; v1 refused, fleet and attended pass, control unguarded'

record_git() {
  git "$@" || fail "($arm) fixture git $*"
}

record_commit() {
  record_git -c user.name=t -c user.email=t@t commit -qm "$1"
}

record_file() {
  mkdir -p "$(dirname "$1")" || fail "($arm) create parent directory for $1"
  printf '%s\n' "$2" >"$1" || fail "($arm) write $1"
  record_git add -- "$1"
  record_commit "$2"
}

record_repo() {
  T=$(mktemp -d /tmp/scope-guard-record.XXXXXX) || fail "($arm) mktemp"
  trap 'rm -rf "$T"' EXIT
  record_git init -q --bare "$T/origin.git"
  record_git init -q "$T/w"
  cd "$T/w" || fail "($arm) cd to pushing repo"
  record_git remote add origin "$T/origin.git"
  record_git checkout -qb dev
  record_git -c user.name=t -c user.email=t@t commit -qm base --allow-empty
  record_git push -q origin dev
  record_git checkout -qb rec
}

advance_dev() (
  cd "$T/dev" || fail "($arm) cd to dev fixture"
  record_file design_docs/fleet-mission-log.md "$1"
  record_git push -q origin dev
)

rebased_record() {
  record_file design_docs/triage-mission-log.md record
  record_git push -q origin rec
  old_rec=$(record_git rev-parse HEAD) || fail "($arm) resolve old record tip"
  record_git clone -q --branch dev "$T/origin.git" "$T/dev"
  advance_dev inherited || exit 1
  record_git fetch -q origin
  record_git -c user.name=t -c user.email=t@t rebase -q origin/dev

  git merge-base --is-ancestor "$old_rec" HEAD
  [ "$?" -eq 1 ] || fail "($arm) fixture did not produce a non-fast-forward update"
  local paths
  paths=$(record_git diff --name-only "$old_rec" HEAD) || fail "($arm) diff old record"
  grep -Fxq 'design_docs/fleet-mission-log.md' <<<"$paths" || fail "($arm) old record diff lacks the inherited Fleet log"
  assert_record_diff
}

assert_record_diff() {
  local paths
  paths=$(record_git diff --name-only origin/dev...HEAD) || fail "($arm) diff from dev"
  [ "$paths" = design_docs/triage-mission-log.md ] || fail "($arm) expected triage-only PR diff, got: $paths"
}

check_record_push() {
  local label="$1" want_rc="$2" want_tip="$3" diagnostic="$4" rc=0 tip
  shift 4
  push triage rec "$@" || rc=$?
  tip=$(record_git --git-dir="$T/origin.git" rev-parse --verify refs/heads/rec) || fail "($arm) resolve bare record tip"
  [ "$rc" -eq "$want_rc" ] && [ "$tip" = "$want_tip" ] ||
    fail "($arm) $label expected rc=$want_rc and rec=$want_tip, got rc=$rc and rec=$tip"$'\n'"$(cat "$T/err")"
  if [ -n "$diagnostic" ]; then
    grep -Fq -- "$diagnostic" "$T/err" ||
      fail "($arm) $label expected diagnostic '$diagnostic', got: $(cat "$T/err")"
  fi
}

run_record_case() (
  local arm="$1" old_rec new_rec lease
  record_repo
  if [ "$arm" = c ]; then
    record_git fetch -q origin
    record_file tools/launchd/x.sh 'previously published harness change'
    record_git push -q origin rec
    record_file design_docs/triage-mission-log.md record
    new_rec=$(record_git rev-parse HEAD) || fail "($arm) resolve appended record tip"
    check_record_push 'fast-forward record' 0 "$new_rec" ''
    record_file tools/launchd/x.sh 'new harness change'
    check_record_push 'fast-forward harness change' 1 "$new_rec" 'MISSION_NAME=triage may not push tools/launchd/x.sh'
    echo 'PASS (c) fast-forward checks only the newly pushed delta'
    exit 0
  fi

  rebased_record
  new_rec=$(record_git rev-parse HEAD) || fail "($arm) resolve rebased record tip"
  lease="--force-with-lease=refs/heads/rec:$old_rec"
  case "$arm" in
    a)
      check_record_push 'rebased record' 0 "$new_rec" '' "$lease"
      echo 'PASS (a) rebased record accepts the inherited Fleet log'
      ;;
    b)
      record_file tools/launchd/x.sh 'branch-authored harness change'
      check_record_push 'rebased harness change' 1 "$old_rec" 'MISSION_NAME=triage may not push tools/launchd/x.sh' "$lease"
      echo 'PASS (b) rebased record refuses a branch-authored harness path'
      ;;
    d)
      advance_dev advanced || exit 1
      check_record_push 'stale dev' 1 "$old_rec" 'refs/heads/rec: stale refs/remotes/origin/dev' "$lease"
      record_git fetch -q origin
      check_record_push 'fetched dev retry' 0 "$new_rec" '' "$lease"
      ;;
    d-missing)
      record_git update-ref -d refs/remotes/origin/dev
      check_record_push 'missing dev' 1 "$old_rec" 'refs/heads/rec: refs/remotes/origin/dev unavailable; fetch origin and retry' "$lease"
      ;;
    e)
      record_file tools/launchd/x.sh 'branch-authored harness change'
      record_git rm -q -- tools/launchd/x.sh
      record_commit 'remove branch-authored harness change'
      assert_record_diff
      check_record_push 'rebased record with reverted harness change' 1 "$old_rec" 'MISSION_NAME=triage may not push tools/launchd/x.sh' "$lease"
      echo 'PASS (e) a later revert cannot hide a branch-authored harness path'
      ;;
  esac
)

for arm in a b c d d-missing e; do
  run_record_case "$arm" || exit 1
  if [ "$arm" = d-missing ]; then
    echo 'PASS (d) stale and missing dev refuse the push, and fetching allows the retry'
  fi
done

# The driver must enable it, below the controller spawn's env and pointing at this directory.
grep -q 'GIT_CONFIG_VALUE_.*tools/launchd/githooks' "$DRIVER" || fail "driver does not enable the scope guard"
echo 'PASS driver wires the scope guard'
