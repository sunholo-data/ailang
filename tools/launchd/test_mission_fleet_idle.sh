#!/bin/bash
# Fleet idle pre-check (M-HARNESS-MISSION-LOOP M4): zero open tickets exits before any probe;
# open tickets proceed; an unreadable count yields loudly; other missions never run it.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
fail() { echo "FAIL $*"; exit 1; }
block=$(awk '/--- FLEET IDLE PRE-CHECK START ---/{on=1} on{print} /--- FLEET IDLE PRE-CHECK END ---/{exit}' "$DRIVER")
[ -n "$block" ] || fail "extraction"
case "$block" in *'exit 0'*) ;; *) fail "block never exits" ;; esac

run() {  # MISSION_NAME stub_rc stub_out → prints "EXIT <rc> CALLS <n> | log…" or "FELLTHROUGH …"
  ( MISSION_NAME="$1"; STUB_RC="$2"; STUB_OUT="$3"; CALLS=0; LOGS=""
    log() { LOGS="$LOGS|$*"; }
    _mc_bounded() { CALLS=$((CALLS+1)); MC_BOUNDED_OUT="$STUB_OUT"; return "$STUB_RC"; }
    exit() { echo "EXIT ${1:-0} CALLS $CALLS $LOGS"; builtin exit 0; }
    eval "$block"
    echo "FELLTHROUGH CALLS $CALLS $LOGS" )
}
out=$(run fleet 0 "0");          case "$out" in "EXIT 0 CALLS 1 |fleet: no open tickets"*) ;; *) fail "idle: $out" ;; esac
out=$(run fleet 0 "2");          case "$out" in "FELLTHROUGH CALLS 1 |fleet: 2 open"*) ;; *) fail "open work did not proceed: $out" ;; esac
out=$(run fleet 1 "unknown mission subcommand"); case "$out" in "EXIT 0 CALLS 1 |fleet: cannot count"*) ;; *) fail "failed read: $out" ;; esac
out=$(run fleet 0 "");           case "$out" in "EXIT 0 CALLS 1 |fleet: cannot count"*) ;; *) fail "empty output treated as a count: $out" ;; esac
out=$(run world 0 "0");          case "$out" in "FELLTHROUGH CALLS 0 "*) ;; *) fail "non-fleet mission ran the check: $out" ;; esac
echo 'PASS fleet idle pre-check: 0 → idle exit, N → proceeds, unreadable → loud yield, other missions skip'

# Placement: below the kill switch, above the first probe and the notice drain.
ln_kill=$(grep -n 'kill switch present' "$DRIVER" | head -1 | cut -d: -f1)
ln_idle=$(grep -n 'FLEET IDLE PRE-CHECK START' "$DRIVER" | cut -d: -f1)
ln_drain=$(grep -n '^_mc_drain_notices$' "$DRIVER" | head -1 | cut -d: -f1)
ln_probe=$(grep -n '^_an_probed=":"' "$DRIVER" | head -1 | cut -d: -f1)
[ "$ln_kill" -lt "$ln_idle" ] && [ "$ln_idle" -lt "$ln_drain" ] && [ "$ln_idle" -lt "$ln_probe" ] \
  || fail "placement kill=$ln_kill idle=$ln_idle drain=$ln_drain probe=$ln_probe"
grep -q '    fleet)  echo 1680 ;;' "$DRIVER" || fail "fleet has no boot-offset arm (it would join the boot stampede)"
echo 'PASS placement: after the kill switch, before the drain and every probe; boot offset set'
