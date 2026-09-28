#!/bin/bash
# test_cron_kicker.sh — the overdue predicate behind the cron backstop.
#
# WHY THIS EXISTS. The kicker's whole safety claim is that it disarms itself: it
# is installed permanently, fires every minute forever, and must do NOTHING while
# gui/<uid> is healthy. A backstop that keeps firing after the fault clears would
# double-run mission iterations — two controllers on one repo, which is the exact
# shape of damage the loops' own overlap guards exist to prevent.
#
# That claim cannot be checked by reading it. Arm 1 is the one that dies if anyone
# replaces the "counter has not moved" test with a health flag, a timestamp file,
# or anything else that survives a re-login.
#
# launchctl is stubbed, so the suite runs anywhere and cannot be made green by a
# healthy rig underneath it. The stub records every kickstart it is asked for.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
KICKER="$HERE/cron-kicker.sh"
[ -x "$KICKER" ] || { echo "FAIL: $KICKER missing or not executable"; exit 1; }

TMP="${TMPDIR:-/tmp}/kicker-test-$$"
mkdir -p "$TMP/bin" "$TMP/state"
trap 'rm -rf "$TMP"' EXIT

FAILED=0
ok()   { echo "  ok — $1"; }
fail() { echo "  FAIL — $1"; FAILED=1; }

# ── launchctl stub ───────────────────────────────────────────────────────────
# Renders one job from $STUB_RUNS / $STUB_STATE / $STUB_INTERVAL, in the exact
# shape real `launchctl print` emits (tab-indented; verified against
# gui/501/dev.ailang.mission-control on macOS 26.5.2, 2026-09-05).
cat > "$TMP/bin/launchctl" <<'STUB'
#!/bin/bash
case "$1" in
  print)
    case "$2" in
      gui/[0-9]*)
        target="${2#gui/}"
        case "$target" in */*) ;; *) [ "${STUB_DOMAIN_MISSING:-0}" = "1" ] && exit 125; exit 0 ;; esac
        ;;
    esac
    echo "$2" >> "$STUB_PRINTLOG"
    [ "${STUB_MISSING:-0}" = "1" ] && exit 1
    printf '\tstate = %s\n' "${STUB_STATE:-not running}"
    printf '\truns = %s\n' "${STUB_RUNS:-0}"
    printf '\tpended nondemand spawn = interval\n'
    [ -n "${STUB_INTERVAL:-}" ] && printf '\trun interval = %s seconds\n' "$STUB_INTERVAL"
    exit 0
    ;;
  kickstart) echo "$2" >> "$STUB_KICKLOG"; [ "${STUB_KICK_FAIL:-0}" = "1" ] && exit 1; exit 0 ;;
esac
exit 1
STUB
chmod +x "$TMP/bin/launchctl"

export PATH="$TMP/bin:$PATH"
export AILANG_KICKER_STATE="$TMP/state"
export AILANG_KICKER_LOG="$TMP/kicker.log"
export AILANG_KICKER_LABELS="dev.ailang.test-job"
export STUB_KICKLOG="$TMP/kicks"
export STUB_PRINTLOG="$TMP/prints"
LABEL_STATE="$TMP/state/dev.ailang.test-job"

reset() { : > "$STUB_KICKLOG"; rm -f "$LABEL_STATE"; }
kicks() { [ -s "$STUB_KICKLOG" ] && wc -l < "$STUB_KICKLOG" | tr -d ' ' || echo 0; }

# Guard against a vacuously green suite: if the stub were unreachable the script
# would skip every job and every "did not kick" arm below would pass for the
# wrong reason. Prove a kick CAN be observed before asserting any absence.
reset
export STUB_INTERVAL=60 STUB_RUNS=7 STUB_STATE="not running"
echo "7 1" > "$LABEL_STATE"   # counter frozen since epoch 1 — maximally overdue
"$KICKER"
[ "$(kicks)" = "1" ] || { echo "FAIL harness: the stub never saw a kickstart — every arm below would be vacuous"; exit 1; }
echo "harness live (a kick is observable)"

echo "arm 1: healthy launchd — a counter that moves is NEVER kicked"
reset
STUB_RUNS=100; "$KICKER"                      # first sight: record only
for n in 101 102 103 104 105; do
    # Age the recorded timestamp past a full interval, then move the counter, as
    # a healthy launchd does: it spawned the job itself, on time.
    prev=$(cut -d' ' -f1 "$LABEL_STATE")
    echo "$prev $(( $(date +%s) - STUB_INTERVAL - 5 ))" > "$LABEL_STATE"
    STUB_RUNS=$n; "$KICKER"
done
if [ "$(kicks)" = "0" ]; then ok "5 healthy intervals, 0 kicks"; else fail "kicked $(kicks)x while launchd was spawning on its own"; fi

echo "arm 2: wedged launchd — a frozen counter past its interval IS kicked"
reset
STUB_RUNS=5; "$KICKER"
echo "5 $(( $(date +%s) - STUB_INTERVAL - 5 ))" > "$LABEL_STATE"
STUB_RUNS=5; "$KICKER"
if [ "$(kicks)" = "1" ]; then ok "counter stuck at 5 past its interval — kicked once"; else fail "expected exactly 1 kick, got $(kicks)"; fi

echo "arm 3: first sight is never kicked"
reset
STUB_RUNS=5; "$KICKER"
if [ "$(kicks)" = "0" ]; then ok "no state file — recorded, not kicked"; else fail "kicked a job on first observation"; fi

echo "arm 4: a RUNNING job is never kicked, however old its counter"
# mission-control routinely runs 2-3h on a 90m interval. Treating that as overdue
# would fire a second controller into a live one.
reset
STUB_STATE="running"; STUB_RUNS=5; "$KICKER"
echo "5 1" > "$LABEL_STATE"                    # frozen since epoch 1
STUB_STATE="running"; "$KICKER"
if [ "$(kicks)" = "0" ]; then ok "running job left alone"; else fail "kicked a job that was already running"; fi
STUB_STATE="not running"

echo "arm 5: a job with no run interval is skipped, not guessed at"
# KeepAlive and StartCalendarInterval jobs have no interval to be overdue against.
reset
STUB_INTERVAL=""; STUB_RUNS=5; "$KICKER"
echo "5 1" > "$LABEL_STATE"
STUB_INTERVAL=""; "$KICKER"
if [ "$(kicks)" = "0" ]; then ok "no interval — skipped"; else fail "kicked an interval-less job"; fi
STUB_INTERVAL=60

echo "arm 6: an unknown label is skipped without error"
reset
STUB_MISSING=1 "$KICKER"; rc=$?
if [ "$rc" = "0" ] && [ "$(kicks)" = "0" ]; then
    ok "absent job — exit 0, no kick"
else
    fail "rc=$rc kicks=$(kicks) on an absent job"
fi

echo "arm 7: a failed kick retries on the job's cadence, not every minute"
reset
export STUB_KICK_FAIL=1
STUB_RUNS=5; "$KICKER"
echo "5 $(( $(date +%s) - STUB_INTERVAL - 5 ))" > "$LABEL_STATE"
"$KICKER"; "$KICKER"; "$KICKER"     # two further passes, both within the interval
if [ "$(kicks)" = "1" ]; then ok "1 attempt, then backs off for a full interval"; else fail "expected 1 attempt, got $(kicks) — a failing kick is retrying every pass"; fi
unset STUB_KICK_FAIL

# All new arms use isolated state and logs. The notifier is always a test stub;
# no arm can invoke the installed ailang binary or its message store.
export AILANG_KICKER_AILANG="$TMP/bin/ailang"
export STUB_NOTIFYLOG="$TMP/notifies"
cat > "$AILANG_KICKER_AILANG" <<'STUB'
#!/bin/bash
printf '%s|%s|%s|%s\n' "$AILANG_STORAGE_MESSAGING" "$AILANG_MESSAGES_PROJECT" "$*" "${STUB_NOTIFY_MODE:-ok}" >> "$STUB_NOTIFYLOG"
[ "${STUB_NOTIFY_MODE:-ok}" = "hang" ] && sleep 60
[ "${STUB_NOTIFY_MODE:-ok}" = "fail" ] && exit 1
exit 0
STUB
chmod +x "$AILANG_KICKER_AILANG"
count_log() { if [ -f "$AILANG_KICKER_LOG" ]; then grep -c "$1" "$AILANG_KICKER_LOG"; else echo 0; fi; }

echo "arm 8: mission labels come from the registry"
export AILANG_KICKER_STATE="$TMP/state-8" AILANG_KICKER_LOG="$TMP/log-8"
export AILANG_KICKER_REGISTRY="$TMP/registry-8"
unset AILANG_KICKER_LABELS
mkdir -p "$AILANG_KICKER_REGISTRY"
printf 'name = "v1"\n' > "$AILANG_KICKER_REGISTRY/v1.toml"
printf 'name    = "fleet"\n' > "$AILANG_KICKER_REGISTRY/fleet.toml"
printf 'name = "stapledon"\n' > "$AILANG_KICKER_REGISTRY/stapledon.toml"
: > "$STUB_PRINTLOG"
"$KICKER"
if grep -q '/dev.ailang.mission-control$' "$STUB_PRINTLOG" &&
   grep -q '/dev.ailang.mission-fleet$' "$STUB_PRINTLOG" &&
   grep -q '/dev.ailang.mission-stapledon$' "$STUB_PRINTLOG" &&
   ! grep -q '/dev.ailang.mission-v1$' "$STUB_PRINTLOG"; then
    ok "v1, fleet and stapledon labels derived correctly"
else fail "registry labels missing or v1 mapped incorrectly"; fi

echo "arm 9: unreadable registry falls back and logs once"
export AILANG_KICKER_STATE="$TMP/state-9" AILANG_KICKER_LOG="$TMP/log-9"
export AILANG_KICKER_REGISTRY="$TMP/absent-registry"
: > "$STUB_PRINTLOG"
"$KICKER"; "$KICKER"; "$KICKER"
if grep -q '/dev.ailang.mission-fleet$' "$STUB_PRINTLOG" &&
   grep -q '/dev.ailang.mission-stapledon$' "$STUB_PRINTLOG" &&
   [ "$(count_log 'REGISTRY-UNREADABLE')" = "1" ]; then
    ok "fallback includes all missions; one log across 3 runs"
else fail "fallback coverage or log count wrong"; fi

echo "arm 10: absent Aqua domain reports and notifies once"
export AILANG_KICKER_STATE="$TMP/state-10" AILANG_KICKER_LOG="$TMP/log-10"
export AILANG_KICKER_LABELS="dev.ailang.test-job"
: > "$STUB_KICKLOG"; : > "$STUB_PRINTLOG"; : > "$STUB_NOTIFYLOG"
mkdir -p "$AILANG_KICKER_STATE"
echo "5 1" > "$AILANG_KICKER_STATE/dev.ailang.test-job"
export STUB_DOMAIN_MISSING=1
"$KICKER"; "$KICKER"; "$KICKER"
if [ "$(kicks)" = "0" ] &&
   [ "$(count_log 'SESSION-LOST')" = "1" ] &&
   [ "$(wc -l < "$STUB_NOTIFYLOG" | tr -d ' ')" = "1" ] &&
   [ -f "$AILANG_KICKER_STATE/SESSION-LOST" ] &&
   grep -q "^gcp|ailang-multivac|messages send user .*marker: $AILANG_KICKER_STATE/SESSION-LOST; recovery: log in to the rig's GUI session" "$STUB_NOTIFYLOG" &&
   [ ! -s "$STUB_PRINTLOG" ]; then
    ok "one outage report, one notification, no labels or kicks"
else fail "outage was silent, repeated, or attempted kickstart"; fi

echo "arm 11: restoration resets overdue clocks"
unset STUB_DOMAIN_MISSING
echo "5 1" > "$AILANG_KICKER_STATE/dev.ailang.test-job"
"$KICKER"
if [ "$(count_log 'SESSION-RESTORED')" = "1" ] &&
   [ ! -e "$AILANG_KICKER_STATE/SESSION-LOST" ] &&
   [ "$(kicks)" = "0" ] &&
   [ "$(cut -d' ' -f2 "$AILANG_KICKER_STATE/dev.ailang.test-job")" -gt 1 ]; then
    ok "restored once, first pass observed without kicking"
else fail "restore marker, logging, or overdue state wrong"; fi

echo "arm 12: healthy domain has no session alert"
export AILANG_KICKER_STATE="$TMP/state-12" AILANG_KICKER_LOG="$TMP/log-12"
: > "$STUB_NOTIFYLOG"
"$KICKER"
if [ "$(count_log 'SESSION-')" = "0" ] && [ ! -s "$STUB_NOTIFYLOG" ]; then
    ok "healthy domain stays quiet"
else fail "healthy domain raised a session alert"; fi

echo "arm 13: failed or hung notifier is fail-soft and bounded"
export AILANG_KICKER_STATE="$TMP/state-13" AILANG_KICKER_LOG="$TMP/log-13"
export STUB_DOMAIN_MISSING=1
export STUB_NOTIFY_MODE=fail
"$KICKER"; fail_rc=$?
rm -f "$AILANG_KICKER_STATE/SESSION-LOST"
export STUB_NOTIFY_MODE=hang AILANG_KICKER_NOTIFY_TIMEOUT=2
start=$(date +%s)
"$KICKER"; hang_rc=$?
elapsed=$(( $(date +%s) - start ))
if [ "$fail_rc" = "0" ] && [ "$hang_rc" = "0" ] &&
   [ "$(count_log 'NOTIFY rc=1$')" = "1" ] &&
   [ "$(count_log 'NOTIFY rc=124$')" = "1" ] &&
   [ "$elapsed" -lt 10 ]; then
    ok "send failure and timeout keep kicker at rc 0"
else fail "notifier rc/timeout wrong: fail_rc=$fail_rc hang_rc=$hang_rc rc1=$(count_log 'NOTIFY rc=1$') rc124=$(count_log 'NOTIFY rc=124$') elapsed=${elapsed}s"; fi

echo
[ "$FAILED" = "0" ] && { echo "PASS — all arms"; exit 0; }
echo "FAILED"; exit 1
