#!/bin/bash
# mission-lane-check.sh NAME [--no-pi-task] — is every lane this mission routes to actually usable?
#
# m-mission-light-profile Phase 1. Docs iteration 17 (2026-10-01) ran 51 minutes and changed
# nothing: its judge lane was dead (pi sandbox runtime never installed in the clone; the global
# and repo pi extensions loaded twice), and nothing had checked before the mission was armed.
# `ailang mission doctor` checks CONFIGURATION drift only; this checks RUNTIME readiness.
#
# It walks the same chains a fire walks, with the fire's own code:
#   * the declared routing comes from the mission's own driver (MISSION_PRINT_CONFIG=1), started
#     from the path in its launchd plist — so env, defaults and pin are what a fire would use;
#   * claude/codex rungs use the driver's probes from lib/lane-probe.sh in that same tree;
#   * pi rungs run a one-file task through scripts/mission_pi_run.sh — the real role launcher,
#     so the sandbox dependency, extension loading, model and provider quota are all exercised.
# Rungs the ration gate blocks are reported as skipped, never called.
#
# Output: one TAB-separated row per check — mission, check, ok|dead|warn|skip, detail — then
# READY or NOT READY. Exit 0 only when every role, the controller and the config are usable.
# Bash 3.2 (the rig's /bin/bash): no associative arrays.
set -uo pipefail

NAME="${1:-}"; [ -n "$NAME" ] || { echo "usage: mission-lane-check.sh NAME [--no-pi-task]" >&2; exit 2; }
PI_TASK=1; [ "${2:-}" = "--no-pi-task" ] && PI_TASK=0
STALE_MAX="${MISSION_LANE_STALE_MAX:-50}"
PI_TASK_SECONDS="${MISSION_LANE_PI_SECONDS:-240}"
# Attended only: exercise rungs the ration gate would skip (spends their quota/metered cents).
IGNORE_RATION="${MISSION_LANE_IGNORE_RATION:-0}"

DEAD=0
row() { printf '%s\t%s\t%s\t%s\n' "$NAME" "$1" "$2" "$3"; [ "$2" = dead ] && DEAD=$((DEAD+1)); return 0; }
log() { printf '  · %s\n' "$*" >&2; }

# --- 1. the declared routing, from the mission's own driver ------------------------------------
PLIST="$HOME/Library/LaunchAgents/dev.ailang.mission-${NAME}.plist"
[ "$NAME" = v1 ] && PLIST="$HOME/Library/LaunchAgents/dev.ailang.mission-control.plist"
[ -f "$PLIST" ] || { row config dead "no launchd plist at $PLIST"; echo "NOT READY"; exit 1; }
DRIVER=$(plutil -extract ProgramArguments json -o - "$PLIST" 2>/dev/null | tr ',' '\n' | tr -d '[]"\\' | grep 'mission-control.sh$' | head -1)
[ -f "$DRIVER" ] || { row config dead "plist names no readable driver ($DRIVER)"; echo "NOT READY"; exit 1; }

CFG=$(MISSION_PROFILE="$NAME" MISSION_PRINT_CONFIG=1 /bin/bash "$DRIVER" 2>/dev/null | grep -E '^[A-Z_]+=')
cfg() { printf '%s\n' "$CFG" | sed -n "s/^$1=//p" | head -1; }
ROOT=$(cfg MC_DRIVER_ROOT); WORKDIR=$(cfg MISSION_WORKDIR)
if [ -z "$ROOT" ] || [ ! -f "$ROOT/tools/launchd/lib/lane-probe.sh" ]; then
  row config dead "the driver printed no routing (MISSION_PRINT_CONFIG unsupported — is its pin older than this check?)"
  echo "NOT READY"; exit 1
fi

# --- 2. configuration drift: the existing doctor, unchanged ------------------------------------
if _doc=$(ailang mission doctor "$NAME" 2>&1); then row config ok "$(printf '%s' "$_doc" | grep -v '^⚠\|make quick-install' | tail -1)"
else row config dead "$(printf '%s' "$_doc" | grep -v '^⚠\|make quick-install' | tail -1)"; fi

# --- 3. the fire's environment, then the fire's probes ------------------------------------------
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY   # as the driver does: subscription lanes
export MISSION_NAME="$NAME"
. "$ROOT/tools/launchd/lib/lane-probe.sh"
[ "$IGNORE_RATION" = 1 ] && _mc_is_over_ration() { return 1; }   # after sourcing: overrides the lib's

# --- 4. the clone the item worktrees come from ---------------------------------------------------
CLONE=$(cd "$(git -C "$WORKDIR" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)/.." 2>/dev/null && pwd -P)
if [ -n "$CLONE" ]; then
  # The repo's own base: dev here, main in stapledons-godot (mission-base.sh derives it the same way).
  _base=origin/dev
  git -C "$CLONE" rev-parse -q --verify "$_base" >/dev/null 2>&1 || _base=$(git -C "$CLONE" symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null || echo origin/main)
  _behind=$(git -C "$CLONE" rev-list --count "HEAD..$_base" 2>/dev/null || echo "?")
  if [ "$_behind" = "?" ]; then row clone warn "$CLONE: cannot compare with $_base"
  elif [ "$_behind" -gt "$STALE_MAX" ]; then row clone warn "$CLONE is $_behind commits behind $_base (fix: git -C $CLONE merge --ff-only $_base when clean)"
  else row clone ok "$CLONE $_behind behind $_base"; fi
fi

# --- 5. one rung --------------------------------------------------------------------------------
# Results are cached per rung (planner and executor often share one) as "|rung=verdict|" pairs —
# "|", not ":", because rung names contain colons. check_rung sets RUNG_V rather than echoing,
# so it runs in THIS shell and the cache survives (a $(...) call would update a subshell's copy).
CACHE="|"
pi_task() { # MODEL → ok | dead:<verdict>
  local m="$1" t wt rc v
  # The cone carries CLAUDE.md + AGENTS.md: the session-protocol gate makes a pi role read
  # CLAUDE.md before it may write, and a .pi-only cone left MiniMax unable to (2026-10-02).
  t=$(mktemp -d "/tmp/lanecheck.XXXXXX") || { echo "dead:mktemp"; return; }
  wt="$t/wt"
  if ! git -C "$CLONE" worktree add -q --detach --no-checkout "$wt" "$(git -C "$ROOT" rev-parse HEAD)" 2>/dev/null \
     || ! git -C "$wt" sparse-checkout set --no-cone '/.pi/' '/CLAUDE.md' '/AGENTS.md' 2>/dev/null || ! git -C "$wt" checkout -q 2>/dev/null; then
    git -C "$CLONE" worktree remove --force "$wt" >/dev/null 2>&1; rm -rf "$t"; echo "dead:worktree"; return
  fi
  printf 'Create a file named LANE_OK in the current directory containing the single word ok. Do nothing else.\n' > "$t/directive"
  bash "$ROOT/scripts/mission_pi_run.sh" --model "$m" --directive "$t/directive" --workdir "$wt" \
    --out "$t/out.ndjson" --verdict "$t/verdict.json" --max-seconds "$PI_TASK_SECONDS" --stall-seconds 120 >/dev/null 2>&1
  rc=$?
  v=$(jq -r '.verdict // "unknown"' "$t/verdict.json" 2>/dev/null || echo unknown)
  git -C "$CLONE" worktree remove --force "$wt" >/dev/null 2>&1; rm -rf "$t"
  [ "$rc" -eq 0 ] && echo ok || echo "dead:$v(rc=$rc)"
}
check_rung() { # RUNG → RUNG_V = ok | skip:<why> | dead:<why>
  local r="$1" m out="" rc=0
  case "$CACHE" in *"|$r="*) RUNG_V=$(printf '%s\n' "$CACHE" | tr '|' '\n' | awk -v k="$r=" 'index($0,k)==1{print substr($0,length(k)+1); exit}'); return ;; esac
  case "$r" in
    codex:*) m="${r#codex:}"; _mc_probe_codex "$m" >/dev/null 2>&1; rc=$? ;;
    pi:*)    m="${r#pi:}"
             if _mc_is_over_ration "$r"; then out="skip:over ration"
             elif [ "$PI_TASK" = 1 ] && [ -n "$CLONE" ]; then
               out=$(pi_task "$m")
               # One retry on a no-op finish: openrouter/minimax-m3 passed and failed this same
               # task minutes apart (2026-10-02). Two misses is a lane problem; one is noise.
               case "$out" in dead:empty_worktree*) out=$(pi_task "$m") ;; esac
             else _mc_probe_pi "$m" >/dev/null 2>&1; rc=$?; fi ;;
    *)       m="${r#claude:}"; _mc_probe "$m" >/dev/null 2>&1; rc=$? ;;
  esac
  if [ -z "$out" ]; then
    case "$rc" in 0) out=ok ;; 75) out="skip:over ration" ;; *) out="dead:probe rc=$rc" ;; esac
  fi
  CACHE="$CACHE$r=$out|"
  RUNG_V="$out"
}

# --- 6. the controller, then each role ----------------------------------------------------------
CTL_PROVIDER=""
_ctl_rows=""
IFS=',' read -r -a _ctl <<<"$(cfg PREFS),$(cfg CONTROLLER_FALLBACK)"
for r in "${_ctl[@]}"; do
  [ -n "$r" ] || continue
  check_rung "$r"; v="$RUNG_V"; _ctl_rows="$_ctl_rows $r=$v"
  if [ "$v" = ok ] && [ -z "$CTL_PROVIDER" ]; then
    case "$r" in codex:*) CTL_PROVIDER=codex ;; pi:*) CTL_PROVIDER=pi ;; *) CTL_PROVIDER=claude ;; esac
  fi
done
if [ -n "$CTL_PROVIDER" ]; then row controller ok "first usable: $CTL_PROVIDER ·$_ctl_rows"
else case "$_ctl_rows" in *=dead:*) row controller dead "no usable rung ·$_ctl_rows" ;;
     *) row controller warn "untested — every rung over ration ·$_ctl_rows" ;; esac; fi

for role in DESIGNER PLANNER EXECUTOR EVALUATOR; do
  _rows=""; _usable=""
  IFS=',' read -r -a _chain <<<"$(cfg ${role}_MODEL),$(cfg ${role}_FALLBACK)"
  for r in "${_chain[@]}"; do
    [ -n "$r" ] || continue
    check_rung "$r"; v="$RUNG_V"
    # A claude-lane judge is reached through Claude's Agent tool, which only a claude
    # controller has (iteration 17: agent-tool:sonnet-unavailable). Usable is not reachable.
    if [ "$role" = EVALUATOR ] && [ "$v" = ok ] && [ "$CTL_PROVIDER" != claude ]; then
      case "$r" in codex:*|pi:*) ;; *) v="skip:unreachable from a $CTL_PROVIDER controller" ;; esac
    fi
    _rows="$_rows $r=$v"
    [ "$v" = ok ] && [ -z "$_usable" ] && _usable="$r"
    case "$v" in dead:*) _broken=1 ;; esac
  done
  role_l=$(printf '%s' "$role" | tr 'A-Z' 'a-z')
  # A rung the ration gate skipped is UNTESTED, not broken: quota rolls over and the fire already
  # waits for it. Only a rung that was tried and failed makes a role dead.
  if [ -n "$_usable" ]; then row "$role_l" ok "first usable: $_usable ·$_rows"
  elif [ "${_broken:-0}" = 1 ]; then row "$role_l" dead "no usable rung ·$_rows"
  else row "$role_l" warn "untested — every rung skipped (over ration or unreachable) ·$_rows"; fi
  _broken=0
done

if [ "$DEAD" -eq 0 ]; then echo "READY"; exit 0; fi
echo "NOT READY ($DEAD dead)"; exit 1
