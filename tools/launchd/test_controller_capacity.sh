#!/bin/bash
# test_controller_capacity.sh — the controller's runtime-capacity path in mission-control.sh:
# retry loop -> demote/re-walk/pause -> slot verdict -> slot notify -> final rc block.
#
# WHY THIS EXISTS (fleet iteration 15, ticket driver:controller-fallback-skips-openrouter-
# ration-and-402-reads-as-crash). pi text-mode prints OpenRouter's credit refusal as a bare
# `402: {"message":...}`. RUNTIME_QUOTA_SIG knew four emitters and not that one, so a spent
# OpenRouter rung read as a CRASH: no demotion, no re-walk, and a "FAILED (rc=1) — timeout or
# crash" notice on top. Separately, a PAUSE (already announced once by the PAUSE branch's
# _mc_notify) fell through to the same generic crash notice and the rcfail marker.
#
# Extraction, not duplication: every block under test is awk'd out of the driver
# (MC_CAPACITY_DRIVER, default the real one) so this suite cannot drift green against an edited
# driver, and the mutation drill can point it at a scratch copy. The model is a FAKE `pi` on
# PATH; there are no probes and zero inference. bash 3.2: no declare -A, ${v,,}, mapfile.
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HERE/../.." && pwd)"
DRV="${MC_CAPACITY_DRIVER:-$HERE/mission-control.sh}"
PROBE_LIB="$HERE/lib/lane-probe.sh"

unset AILANG_STORAGE_MESSAGING AILANG_MESSAGES_PROJECT MISSION_RUNTIME_QUOTA_SIG MISSION_GH_ISSUE \
      MISSION_RUNTIME_QUOTA_REWALKS

ROOT="${TMPDIR:-/tmp}/ctl-capacity-$$"
rm -rf "$ROOT"; mkdir -p "$ROOT/x"
trap 'rm -rf "$ROOT"' EXIT

X="$ROOT/x"
# --- extraction (guarded: an empty extract would make every case vacuous) ------------------
awk '/^_mc_notify\(\) \{/,/^\}$/'            "$DRV" >  "$X/fns.sh"
awk '/^_mc_notice_title\(\) \{/,/^\}$/'      "$DRV" >> "$X/fns.sh"
awk '/^_mc_notice_suppressed\(\) \{/,/^\}$/' "$DRV" >> "$X/fns.sh"
awk '/^_mc_canon_id\(\) \{/,/^\}$/'          "$DRV" >> "$X/fns.sh"
awk '/^_mc_demote\(\) /'                     "$DRV" >> "$X/fns.sh"
awk '/^_mc_is_demoted\(\) \{/,/^\}$/'        "$DRV" >> "$X/fns.sh"
awk '/^_mc_set_controller\(\) \{/,/^\}$/'    "$DRV" >> "$X/fns.sh"
awk '/^select_model\(\) \{/,/^\}$/'          "$DRV" >> "$X/fns.sh"
awk '/^_mc_pi_session_exists\(\) \{/,/^\}$/' "$DRV" >> "$X/fns.sh"
awk '/^_mc_descendants\(\) \{/,/^\}$/'       "$DRV" >> "$X/fns.sh"
awk '/^_mc_kill_tree\(\) \{/,/^\}$/'         "$DRV" >> "$X/fns.sh"
awk '/^_mc_run_once\(\) \{/,/^\}$/'          "$DRV" >> "$X/fns.sh"
awk '/^_mc_bounded\(\) \{/,/^\}$/'           "$PROBE_LIB" >> "$X/fns.sh"
awk '/^_mc_rung_bucket\(\) \{/,/^\}$/'       "$PROBE_LIB" >> "$X/fns.sh"
awk '/^_mc_is_over_ration\(\) \{/,/^\}$/'    "$PROBE_LIB" >> "$X/fns.sh"
/usr/bin/grep -E '^log\(\) \{|^NOTIFY_TIMEOUT=|^HARD_TIMEOUT=|^STALL_GRACE=|^STALL_CHILD_AGE=|^STALL_INTERVAL=|^STALL_SAMPLES=|^RUNTIME_QUOTA_SIG=|^RUNTIME_QUOTA_REWALKS=|^TRANSIENT_RETRIES=|^TRANSIENT_BACKOFF=|^TRANSIENT_SIG=|^MISSION_LOG_FILE=|^pre_last_record=' "$DRV" > "$X/vars.sh"
awk '/^attempt=1$/{f=1} f{print} f&&/^done$/{exit}' "$DRV"                     > "$X/loop.sh"
awk '/^# --- SLOT VERDICT START ---/,/^# --- SLOT VERDICT END ---/' "$DRV"    > "$X/verdict.sh"
awk '/^# --- SLOT NOTIFY START ---/,/^# --- SLOT NOTIFY END ---/' "$DRV"      > "$X/slotnotify.sh"
awk '/^if \[ "\$RC" -ne 0 \]; then$/,/^exit "\$RC"$/' "$DRV"                  > "$X/final.sh"
for f in fns vars loop verdict slotnotify final; do
  [ -s "$X/$f.sh" ] || { echo "FAIL extraction: $f not found in $DRV"; exit 1; }
done
/usr/bin/grep -q RUNTIME_QUOTA_SIG "$X/loop.sh" && /usr/bin/grep -q TRANSIENT_SIG "$X/loop.sh" \
  || { echo "FAIL extraction: retry loop cut short (missing RUNTIME_QUOTA_SIG/TRANSIENT_SIG)"; exit 1; }
/usr/bin/grep -q '^_mc_run_once()' "$X/fns.sh" && /usr/bin/grep -q '^select_model()' "$X/fns.sh" \
  && /usr/bin/grep -q '^_mc_is_over_ration()' "$X/fns.sh" \
  || { echo "FAIL extraction: function boundaries not found"; exit 1; }
[ "$(/usr/bin/grep -c . "$X/vars.sh")" -ge 13 ] || { echo "FAIL extraction: driver variable lines not found"; exit 1; }

# --- per-case lab ----------------------------------------------------------------------
fail=0; npass=0
LAB=""

# Fake pi: reads $LAB/script/<modelkey>.<n> (or .default): line1 rc, line2 stamp ('-' = none),
# rest = text to print. Logs the model it was asked for to $LAB/pi.argv. A `landed` file makes
# it append a `## ` record to the mission log, as a finishing iteration would.
make_lab() {
  LAB="$ROOT/lab.$1"; rm -rf "$LAB"
  mkdir -p "$LAB/bin" "$LAB/script" "$LAB/state" "$LAB/repo"
  : > "$LAB/pi.argv"; : > "$LAB/sends"; : > "$LAB/ailang.argv"
  cat > "$LAB/bin/pi" <<'STUB'
#!/bin/bash
model=""; prev=""
for a in "$@"; do [ "$prev" = "--model" ] && model="$a"; prev="$a"; done
printf '%s\n' "$model" >> "$MC_LAB/pi.argv"
key=$(printf '%s' "$model" | tr '/:' '__')
n=$(grep -c "^${model}\$" "$MC_LAB/pi.argv")
f="$MC_LAB/script/$key.$n"; [ -f "$f" ] || f="$MC_LAB/script/$key.default"
[ -f "$f" ] || { echo "NOSCRIPT for $model attempt $n"; exit 99; }
rc=$(sed -n 1p "$f"); stamp=$(sed -n 2p "$f")
sed -n '3,$p' "$f"
[ "$stamp" != "-" ] && printf '%s\t%s\t%s\t%s\t\n' "$(date +%s)" "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$stamp" "1" >> "$MC_HB"
[ -f "$MC_LAB/landed" ] && printf '## Iteration landed\n' >> "$MC_LAB/mission-log.md"
exit "$rc"
STUB
  cat > "$LAB/bin/ailang" <<'STUB'
#!/bin/bash
printf '%s\n' "$*" >> "$MC_LAB/ailang.argv"
prev=""
for a in "$@"; do [ "$prev" = "--title" ] && printf '%s\n' "$a" >> "$MC_LAB/sends"; prev="$a"; done
exit 0
STUB
  printf '#!/bin/bash\nexit 0\n' > "$LAB/bin/gh"
  printf '#!/bin/bash\nexit 1\n' > "$LAB/bin/claude"
  cp "$LAB/bin/claude" "$LAB/bin/codex"
  chmod +x "$LAB/bin/"*
}

# script MODEL N RC STAMP TEXT — N may be `default`.
script() { # model n rc stamp text
  local key; key=$(printf '%s' "$1" | tr '/:' '__')
  printf '%s\n%s\n%s\n' "$3" "$4" "$5" > "$LAB/script/$key.$2"
}

# runcase NAME — runs the extracted blocks in a subshell. Inputs (globals): CHAIN, REWALKS,
# MARKER_PRE (empty = absent), MODE (full|final-only). Outputs: $LAB/out (stdout+stderr
# of the run), $LAB/err (stderr only), CASE_RC.
CHAIN=""; REWALKS=""; MARKER_PRE=""; MODE=full; CASE_RC=0
MARKER=""
runcase() {
  MARKER="$LAB/state/mission-capmission-rcfail.episode"
  rm -f "$MARKER"; [ -n "$MARKER_PRE" ] && printf '%s' "$MARKER_PRE" > "$MARKER"
  printf '# mission\n' > "$LAB/mission.md"; : > "$LAB/mission-log.md"
  (
    set -uo pipefail
    export PATH="$LAB/bin:/usr/bin:/bin"
    export MC_LAB="$LAB"
    export MISSION_TIMEOUT=20 MISSION_STALL_GRACE=20 MISSION_TRANSIENT_BACKOFF=0 MISSION_NOTIFY_TIMEOUT=5
    export MISSION_STALL_INTERVAL=30
    [ -n "$REWALKS" ] && export MISSION_RUNTIME_QUOTA_REWALKS="$REWALKS"
    MISSION_NAME=capmission; MSG_FROM=capacity-test
    LOG="$LAB/driver.log"; : > "$LOG"
    STATE_DIR="$LAB/state"; _mc_slot_state="$LAB/state"
    MISSION_DOC="$LAB/mission.md"; REPO="$LAB/repo"; PIDFILE="$LAB/pid"
    MC_DRIVER_ROOT="$REPO_ROOT"; PROMPT="fake prompt"
    MC_PI_SESSION_ID="cap-session-$$"; MC_PI_RESUME_PROMPT="resume"
    MC_HB="$LAB/state/mission-capmission-heartbeat"; export MC_HB
    START_EPOCH=$(date +%s)
    OVERRIDE_FILE="$LAB/no-override"; MISSION_MODEL=""; PREFS=""; PROBE_TIMEOUT=5
    CONTROLLER_FALLBACK="$CHAIN"
    MC_DEMOTED=""; [ "$MODE" = full ] && MC_PAUSED=0
    . "$X/vars.sh"
    . "$X/fns.sh"
    # Seams: the billing reader is out of scope (nothing is over ration); probes all pass for pi.
    _mc_load_ration() { MC_OVER_RATION=""; }
    _mc_probe() { return 1; }
    _mc_probe_codex() { return 1; }
    _mc_probe_pi() { return 0; }
    _mc_stalled() { return 1; }
    # _mc_bounded polls with a 2s sleep; keep the lab quick. Other sleeps pass through.
    sleep() { case "$1" in 2) command sleep 0.2 ;; *) command sleep "$@" ;; esac; }
    if [ "$MODE" = full ]; then
      select_model || { echo "FAIL harness: no initial controller"; exit 90; }
      . "$X/loop.sh"
      . "$X/verdict.sh"
      . "$X/slotnotify.sh"
      echo "STATE paused=${MC_PAUSED} demoted=[${MC_DEMOTED}] controller=${CONTROLLER_ID:-none} rewalks=${_mc_rewalks:-0}"
    else
      RC=1
    fi
    . "$X/final.sh"
  ) >"$LAB/out" 2>"$LAB/err"
  CASE_RC=$?
  cat "$LAB/err" >> "$LAB/out"
}

# --- assertions ------------------------------------------------------------------------
cname=""
cfail=0
begin() { cname="$1"; cfail=0; make_lab "$1"; CHAIN=""; REWALKS=""; MARKER_PRE=""; MODE=full; }
expect() { # description, condition-rc
  if [ "$2" -ne 0 ]; then cfail=1; echo "  ($cname) unmet: $1"; fi
}
count_sends() { /usr/bin/grep -c "$1" "$LAB/sends"; true; }
has_log() { /usr/bin/grep -qF -- "$1" "$LAB/driver.log"; }
state_has() { /usr/bin/grep -qF -- "$1" "$LAB/out"; }
pi_count() { /usr/bin/grep -cxF "$1" "$LAB/pi.argv"; true; }
endcase() {
  if [ "$cfail" -eq 0 ]; then echo "PASS $cname"; npass=$((npass+1)); else echo "FAIL $cname"; fail=1; echo "  --- out:"; sed 's/^/  | /' "$LAB/out" | tail -25; echo "  --- sends:"; sed 's/^/  | /' "$LAB/sends"; fi
}
A="pi:openrouter/a"; B="pi:openrouter/b"; C="pi:openrouter/c"
CH3="$A,$B,$C"
P402='402: {"message":"Insufficient credits. Add more using https://openrouter.ai/settings/credits"}'
DONE_STAMP=complete

# --- hermetic-path ---------------------------------------------------------------------
begin hermetic-path
got=$(PATH="$LAB/bin:/usr/bin:/bin"; command -v pi; command -v ailang; command -v gh; command -v claude; command -v codex)
want=$(printf '%s\n' "$LAB/bin/pi" "$LAB/bin/ailang" "$LAB/bin/gh" "$LAB/bin/claude" "$LAB/bin/codex")
[ "$got" = "$want" ]; expect "fakes shadow every provider CLI ($got)" $?
endcase

# --- 402 classification ----------------------------------------------------------------
begin 402-rewalk-then-complete
CHAIN="$CH3"
script openrouter/a default 1 - "$P402"
script openrouter/b default 0 complete "done"
runcase
[ "$CASE_RC" -eq 0 ]; expect "process rc 0 (got $CASE_RC)" $?
state_has "demoted=[ $A]"; expect "only a demoted" $?
has_log "controller re-walk 1/4: $A → $B"; expect "re-walk log line" $?
has_log "slot-verdict: COMPLETED"; expect "verdict COMPLETED" $?
[ "$(count_sends 'FAILED')" -eq 0 ] && [ "$(count_sends 'PAUSED')" -eq 0 ]; expect "no FAILED/PAUSED sends" $?
endcase

begin 402-all-rungs-pause
CHAIN="$CH3"
for m in a b c; do script openrouter/$m default 1 - "$P402"; done
for pre in "" 1; do
  MARKER_PRE="$pre"; : > "$LAB/pi.argv"; : > "$LAB/sends"
  runcase
  [ "$CASE_RC" -eq 1 ]; expect "process rc stays 1 (pre='$pre', got $CASE_RC)" $?
  state_has "demoted=[ $A $B $C]"; expect "a,b,c each demoted (pre='$pre')" $?
  [ "$(pi_count openrouter/a)" -eq 1 ] && [ "$(pi_count openrouter/b)" -eq 1 ] && [ "$(pi_count openrouter/c)" -eq 1 ]
  expect "each model run exactly once, no same-rung retry (pre='$pre')" $?
  has_log "slot-verdict: PAUSED-NO-CAPACITY"; expect "verdict PAUSED-NO-CAPACITY (pre='$pre')" $?
  [ "$(count_sends 'PAUSED — no capacity')" -eq 1 ]; expect "exactly 1 pause notice (pre='$pre')" $?
  [ "$(count_sends 'Mission iteration FAILED')" -eq 0 ]; expect "0 crash notices (pre='$pre')" $?
  has_log "iteration paused for provider capacity (rc=1) — not a crash"; expect "pause-aware log line (pre='$pre')" $?
  if [ -z "$pre" ]; then [ ! -e "$MARKER" ]; else [ "$(cat "$MARKER")" = "1" ]; fi
  expect "rcfail marker byte-identical (pre='$pre')" $?
done
endcase

begin 402-rewalk-bound
CHAIN="$CH3"; REWALKS=1
for m in a b c; do script openrouter/$m default 1 - "$P402"; done
runcase
state_has "rewalks=1"; expect "exactly 1 re-walk" $?
[ "$(pi_count openrouter/c)" -eq 0 ]; expect "bound stops before rung c" $?
has_log "slot-verdict: PAUSED-NO-CAPACITY"; expect "verdict PAUSED-NO-CAPACITY" $?
[ "$(count_sends 'PAUSED — no capacity')" -eq 1 ] && [ "$(count_sends 'Mission iteration FAILED')" -eq 0 ]
expect "1 pause notice, 0 FAILED" $?
endcase

begin prior-402-no-poison
CHAIN="$CH3"
script openrouter/a default 1 - "$P402"
script openrouter/b default 1 - "boom"
runcase
state_has "demoted=[ $A]"; expect "a demoted" $?
[ "$(/usr/bin/grep -c 'demoted=\[[^]]*openrouter/b' "$LAB/out")" -eq 0 ]; expect "b NOT demoted (a's 402 must not leak into b's slice)" $?
has_log "slot-verdict: CRASHED at="; expect "verdict CRASHED" $?
[ "$(count_sends 'Mission iteration FAILED (rc=1)')" -eq 1 ] && [ "$(count_sends 'PAUSED')" -eq 0 ]; expect "1 FAILED send, 0 pause" $?
[ "$(cat "$MARKER" 2>/dev/null)" = "1" ]; expect "marker = 1" $?
endcase

begin prose-402-not-capacity
CHAIN="$CH3"
script openrouter/a default 1 - "the provider returned 402: credits exhausted
HTTP 402 insufficient credits"
runcase
state_has "demoted=[]"; expect "not demoted" $?
[ "$(/usr/bin/grep -c 're-walk' "$LAB/driver.log")" -eq 0 ]; expect "no re-walk line" $?
has_log "slot-verdict: CRASHED at="; expect "verdict CRASHED" $?
[ "$(count_sends 'Mission iteration FAILED')" -eq 1 ] && [ "$(count_sends 'PAUSED')" -eq 0 ]; expect "1 FAILED, 0 pause" $?
endcase

begin generic-crash-once
CHAIN="$A"
script openrouter/a default 1 - "boom"
runcase
[ "$(count_sends 'Mission iteration FAILED (rc=1)')" -eq 1 ]; expect "run 1: 1 generic send" $?
[ "$(cat "$MARKER" 2>/dev/null)" = "1" ]; expect "run 1: marker written 1" $?
: > "$LAB/sends"; MARKER_PRE=1
runcase
[ "$(count_sends 'FAILED')" -eq 0 ]; expect "run 2: 0 sends" $?
has_log "notice suppressed"; expect "run 2: notice suppressed log line" $?
endcase

begin generic-crash-mcpaused-unset
CHAIN="$A"; MODE=final-only
runcase
[ "$(count_sends 'Mission iteration FAILED (rc=1)')" -eq 1 ]; expect "exactly 1 generic send" $?
[ ! -s "$LAB/err" ]; expect "0 bytes on stderr ($(wc -c < "$LAB/err" | tr -d ' ') bytes)" $?
endcase

# --- existing emitters unchanged -------------------------------------------------------
begin 429-unchanged
CHAIN="$CH3"
script openrouter/a default 1 - '429: {"error":"rate limited"}'
script openrouter/b default 0 complete "done"
runcase
state_has "demoted=[ $A]"; expect "a demoted" $?
has_log "controller re-walk 1/4: $A → $B"; expect "re-walk" $?
has_log "slot-verdict: COMPLETED"; expect "COMPLETED" $?
endcase

begin usage-limit-unchanged
CHAIN="$CH3"
for m in a b c; do script openrouter/$m default 1 - "You've hit your usage limit"; done
runcase
has_log "slot-verdict: PAUSED-NO-CAPACITY"; expect "paused" $?
[ "$(count_sends 'PAUSED — no capacity')" -eq 1 ] && [ "$(count_sends 'Mission iteration FAILED')" -eq 0 ]; expect "1 pause notice, 0 FAILED" $?
endcase

begin transient-unchanged
CHAIN="$CH3"
script openrouter/a 1 1 - "API Error: Overloaded"
script openrouter/a 2 0 complete "done"
runcase
[ "$(pi_count openrouter/a)" -eq 2 ] && [ "$(pi_count openrouter/b)" -eq 0 ]; expect "same rung retried, no re-walk" $?
state_has "demoted=[]"; expect "no demotion" $?
has_log "slot-verdict: COMPLETED"; expect "COMPLETED" $?
endcase

begin rc0-old-402-normal
CHAIN="$CH3"; MARKER_PRE=1
script openrouter/a default 0 complete "$P402"
runcase
state_has "demoted=[]"; expect "no demotion" $?
has_log "slot-verdict: COMPLETED"; expect "COMPLETED" $?
[ "$(/usr/bin/grep -c . "$LAB/sends")" -eq 0 ]; expect "0 sends" $?
[ ! -e "$MARKER" ]; expect "marker removed" $?
endcase

begin landed-record-precedence
CHAIN="$CH3"; MARKER_PRE=1
: > "$LAB/landed"
for m in a b c; do script openrouter/$m default 1 - "$P402"; done
runcase
state_has "paused=1"; expect "run did pause" $?
[ "$(count_sends 'killed post-record')" -eq 1 ]; expect "landed-record notice sent" $?
[ "$(count_sends 'Mission iteration FAILED')" -eq 0 ]; expect "no generic FAILED" $?
[ ! -e "$MARKER" ]; expect "marker removed (if wins over elif)" $?
endcase

begin watchdog-kill-stays-killed
CHAIN="$CH3"
script openrouter/a default 143 - "$P402"
runcase
state_has "demoted=[]"; expect "no demotion" $?
has_log "slot-verdict: KILLED at="; expect "verdict KILLED" $?
[ "$(pi_count openrouter/b)" -eq 0 ]; expect "no re-walk" $?
endcase

echo "controller-capacity: $npass cases passed, fail=$fail"
exit $fail
