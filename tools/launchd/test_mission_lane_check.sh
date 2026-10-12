#!/bin/bash
# test_mission_lane_check.sh — mission-lane-check.sh classifies lanes the way a fire would meet them.
# Every external call is stubbed (plist, driver print-config, probes, pi runner, ailang), so this
# spends nothing and runs on CI. The iteration-17 shape (judge rungs dead, sonnet unreachable from a
# codex controller) must read NOT READY; quota-gated rungs must read untested, not dead.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
CHECK="$HERE/mission-lane-check.sh"
PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; echo "        got: $(printf '%s' "$2" | tr '\n' '|')"; }
has(){ case "$2" in *"$3"*) ok "$1";; *) bad "$1" "$2";; esac; }
hasnt(){ case "$2" in *"$3"*) bad "$1" "$2";; *) ok "$1";; esac; }

T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
H="$T/home"; R="$T/repo"; B="$T/bin"
mkdir -p "$H/Library/LaunchAgents" "$H/.config/ailang" "$R/tools/launchd/lib" "$R/scripts" "$R/.pi/extensions" "$B"

# The mission's "driver": print-config only, driven by CFG_* env.
cat > "$R/tools/launchd/mission-control.sh" <<'EOF'
#!/bin/bash
[ "${MISSION_PRINT_CONFIG:-0}" = 1 ] || exit 3
[ "${NO_PRINT:-0}" = 1 ] && exit 0
cat <<CFG
MISSION_NAME=$MISSION_PROFILE
MISSION_WORKDIR=$FAKE_REPO
MC_DRIVER_ROOT=$FAKE_REPO
PREFS=${CFG_PREFS:-codex:sol}
CONTROLLER_FALLBACK=
DESIGNER_MODEL=${CFG_DESIGNER:-codex:sol}
DESIGNER_FALLBACK=
PLANNER_MODEL=${CFG_PLANNER:-codex:sol}
PLANNER_FALLBACK=${CFG_PLANNER_FB:-}
EXECUTOR_MODEL=${CFG_EXECUTOR:-codex:sol}
EXECUTOR_FALLBACK=${CFG_EXECUTOR_FB:-}
EVALUATOR_MODEL=${CFG_EVALUATOR:-pi:judge}
EVALUATOR_FALLBACK=${CFG_EVALUATOR_FB:-}
CFG
EOF
# The lib the check sources: probe outcomes come from env (OVER = space-separated over-ration rungs).
cat > "$R/tools/launchd/lib/lane-probe.sh" <<'EOF'
. "$(dirname "${BASH_SOURCE[0]}")/codex-auth-profile.sh"
_mc_is_over_ration() { case " ${OVER:-} " in *" $1 "*) return 0;; esac; return 1; }
# Like the real lib, the probes apply the ration gate themselves and return 75 when blocked.
_mc_probe() { _mc_is_over_ration "$1" && return 75; echo "claude $1" >> "$CALLS"; case " ${DEAD_PROBES:-} " in *" $1 "*) return 2;; esac; return 0; }
_mc_probe_codex() { _mc_is_over_ration "codex:$1" && return 75; echo "codex $1" >> "$CALLS"; case " ${DEAD_PROBES:-} " in *" codex:$1 "*) return 1;; esac; return 0; }
_mc_probe_pi() { echo "piprobe $1" >> "$CALLS"; return 0; }
EOF
cp "$HERE/lib/codex-auth-profile.sh" "$R/tools/launchd/lib/codex-auth-profile.sh"
# The pi runner: verdict per model from PI_VERDICTS ("model=verdict ..."); FLAKY models fail once.
cat > "$R/scripts/mission_pi_run.sh" <<'EOF'
#!/bin/bash
while [ $# -gt 0 ]; do case "$1" in --model) M="$2";; --verdict) V="$2";; esac; shift 2; done
echo "pirun $M" >> "$CALLS"
v=ok
for p in ${PI_VERDICTS:-}; do [ "${p%%=*}" = "$M" ] && v="${p#*=}"; done
case " ${FLAKY:-} " in *" $M "*) if [ ! -f "$CALLS.$M" ]; then : > "$CALLS.$M"; v=empty_worktree; fi;; esac
printf '{"verdict":"%s"}\n' "$v" > "$V"
case "$v" in ok) exit 0;; empty_worktree) exit 10;; sandbox_unavailable) exit 15;; *) exit 1;; esac
EOF
chmod +x "$R/tools/launchd/mission-control.sh" "$R/scripts/mission_pi_run.sh"
echo x > "$R/.pi/extensions/a.ts"; echo x > "$R/CLAUDE.md"; echo x > "$R/AGENTS.md"; echo x > "$R/other.txt"
git -C "$R" init -q && git -C "$R" add -A && git -C "$R" -c user.name=t -c user.email=t@t commit -qm init
git -C "$R" update-ref refs/remotes/origin/dev HEAD

cat > "$H/Library/LaunchAgents/dev.ailang.mission-tst.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict><key>ProgramArguments</key><array><string>/bin/bash</string><string>$R/tools/launchd/mission-control.sh</string></array></dict></plist>
EOF
printf '#!/bin/bash\n[ "${DOCTOR_RC:-0}" = 0 ] && echo "1 mission(s): no drift" || { echo "drift: env differs"; exit 1; }\n' > "$B/ailang"
chmod +x "$B/ailang"

# CALLS is set by fresh() in THIS shell, before fresh; o=$(run ...): a $(...) assignment would be lost.
fresh() { CALLS="$T/calls.$RANDOM"; : > "$CALLS"; export CALLS; }
run() {
  HOME="$H" PATH="$B:$PATH" FAKE_REPO="$R" "$@" /bin/bash "$CHECK" tst 2>/dev/null; echo "RC=$?"; }

echo "== healthy"
fresh; o=$(run env); has "healthy → READY" "$o" "READY"; has "exit 0" "$o" "RC=0"
has "config row from ailang mission doctor" "$o" $'config\tok\t1 mission(s): no drift'
has "clone row" "$o" $'clone\tok'

echo "== iteration 17: sonnet judge from a codex controller + dead pi judges"
fresh; o=$(run env CFG_EVALUATOR=sonnet CFG_EVALUATOR_FB=pi:judge PI_VERDICTS="judge=sandbox_unavailable")
has "sonnet unreachable from codex controller" "$o" "sonnet=skip:unreachable from a codex controller"
has "pi judge dead with the runner's verdict" "$o" "pi:judge=dead:sandbox_unavailable(rc=15)"
has "evaluator dead" "$o" $'evaluator\tdead'
has "NOT READY" "$o" "NOT READY (1 dead)"; has "exit 1" "$o" "RC=1"

echo "== claude controller: the sonnet judge is reachable"
fresh; o=$(run env CFG_PREFS=claude-opus CFG_EVALUATOR=sonnet)
has "claude controller" "$o" "first usable: claude"
has "sonnet judge ok" "$o" $'evaluator\tok\tfirst usable: sonnet'

echo "== quota-gated rungs are untested, not dead"
fresh; o=$(run env OVER="codex:sol pi:judge")
has "gated role is warn" "$o" $'planner\twarn\tuntested'
hasnt "gated role is not dead" "$o" $'planner\tdead'
has "gated controller is warn" "$o" $'controller\twarn'
has "still READY" "$o" "READY"
hasnt "a gated rung is never called" "$(cat "$CALLS")" "codex sol"

echo "== a broken rung with a working fallback is ok"
fresh; o=$(run env DEAD_PROBES="codex:sol" CFG_PREFS=claude-opus CFG_PLANNER=codex:sol CFG_PLANNER_FB=pi:glm)
has "planner falls to pi" "$o" $'planner\tok\tfirst usable: pi:glm'
has "dead rung recorded" "$o" "codex:sol=dead:probe rc=1"

echo "== cache: one probe per distinct rung"
fresh; o=$(run env); n=$(grep -c '^codex sol$' "$CALLS")
[ "$n" = 1 ] && ok "codex:sol probed once for controller+designer+planner+executor" || bad "codex:sol probed $n times" "$(cat "$CALLS")"

echo "== retry: one no-op finish is noise, two are a lane problem"
fresh; o=$(run env FLAKY=judge)
has "flaky judge ok after retry" "$o" $'evaluator\tok'
[ "$(grep -c '^pirun judge$' "$CALLS")" = 2 ] && ok "runner called twice" || bad "runner calls" "$(cat "$CALLS")"
fresh; o=$(run env PI_VERDICTS="judge=empty_worktree")
has "persistent no-op is dead" "$o" "pi:judge=dead:empty_worktree(rc=10)"

echo "== the pi task's worktree carries CLAUDE.md (session-protocol gate)"
grep -q "'/CLAUDE.md'" "$CHECK" && ok "cone includes CLAUDE.md" || bad "cone" "$(grep sparse "$CHECK")"
[ -z "$(git -C "$R" worktree list | sed 1d)" ] && ok "task worktrees removed" || bad "leftover worktrees" "$(git -C "$R" worktree list)"

echo "== failure modes of the check itself"
fresh; o=$(run env NO_PRINT=1); has "no print-config → config dead" "$o" "printed no routing"; has "exit 1" "$o" "RC=1"
fresh; o=$(run env DOCTOR_RC=1); has "config drift → dead" "$o" $'config\tdead\tdrift: env differs'

echo "== mission-arm.sh: the one arming surface"
ARM="$HERE/mission-arm.sh"; K="$H/.ailang/state/mission-tst.disabled"; mkdir -p "$H/.ailang/state"
arm() { HOME="$H" PATH="$B:$PATH" FAKE_REPO="$R" "$@" 2>&1; echo "RC=$?"; }
echo "paused for test" > "$K"
fresh; o=$(arm env PI_VERDICTS="judge=sandbox_unavailable" /bin/bash "$ARM" tst --check)
has "--check reports NOT READY" "$o" "RC=1"; [ -f "$K" ] && ok "--check leaves the pause file" || bad "--check removed the pause file" ""
fresh; o=$(arm env PI_VERDICTS="judge=sandbox_unavailable" /bin/bash "$ARM" tst)
has "NOT READY refuses to arm" "$o" "NOT arming tst"; [ -f "$K" ] && ok "pause file kept on refusal" || bad "armed despite NOT READY" ""
fresh; o=$(arm env /bin/bash "$ARM" tst --force)
has "--force without --reason is refused" "$o" "RC=2"
fresh; o=$(arm env PI_VERDICTS="judge=sandbox_unavailable" /bin/bash "$ARM" tst --force --reason "attended test")
has "forced arm" "$o" "tst armed"; [ ! -f "$K" ] && ok "pause file removed" || bad "pause file still there" ""
kept=$(ls "$K".released-* 2>/dev/null | head -1)
grep -q 'FORCE-ARMED .* attended test' "$kept" 2>/dev/null && ok "force reason recorded in the kept note" || bad "no force record" "$(cat "$kept" 2>/dev/null)"
grep -q 'paused for test' "$kept" 2>/dev/null && ok "original pause note kept" || bad "pause note lost" ""
rm -f "$K".released-*; echo "paused again" > "$K"
fresh; o=$(arm env /bin/bash "$ARM" tst)
has "READY arms" "$o" "tst armed"; has "exit 0" "$o" "RC=0"; [ ! -f "$K" ] && ok "pause file removed on READY" || bad "not armed" ""

echo "== one copy of the probes"
D="$HERE/mission-control.sh"
grep -q '^\. "\$MC_DRIVER_ROOT/tools/launchd/lib/lane-probe.sh"' "$D" && ok "driver sources lib/lane-probe.sh" || bad "driver does not source the lib" ""
n=$(grep -cE '^_mc_(probe|probe_codex|probe_pi|bounded|is_over_ration|load_ration|rung_bucket|ration_reason|ration_unreadable|reset_hint)\(\)' "$D")
[ "$n" = 0 ] && ok "driver defines none of the probe/ration functions" || bad "driver still defines $n of them" ""
grep -q 'MISSION_PRINT_CONFIG' "$D" && ok "driver has print-config" || bad "no print-config" ""

echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
