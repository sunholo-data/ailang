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
    [ "${STUB_MISSING:-0}" = "1" ] && exit 1
    printf '\tstate = %s\n' "${STUB_STATE:-not running}"
    printf '\truns = %s\n' "${STUB_RUNS:-0}"
    printf '\tpended nondemand spawn = interval\n'
    [ -n "${STUB_INTERVAL:-}" ] && printf '\trun interval = %s seconds\n' "$STUB_INTERVAL"
    exit 0
    ;;
  kickstart) echo "$2" >> "$STUB_KICKLOG"; exit 0 ;;
esac
exit 1
STUB
chmod +x "$TMP/bin/launchctl"

export PATH="$TMP/bin:$PATH"
export AILANG_KICKER_STATE="$TMP/state"
export AILANG_KICKER_LOG="$TMP/kicker.log"
export AILANG_KICKER_LABELS="dev.ailang.test-job"
export STUB_KICKLOG="$TMP/kicks"
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
cat > "$TMP/bin/launchctl" <<'STUB'
#!/bin/bash
case "$1" in
  print)
    printf '\tstate = not running\n\truns = %s\n\trun interval = %s seconds\n' "${STUB_RUNS:-0}" "${STUB_INTERVAL:-60}"
    exit 0 ;;
  kickstart) echo "$2" >> "$STUB_KICKLOG"; exit 1 ;;   # kick always fails
esac
exit 1
STUB
chmod +x "$TMP/bin/launchctl"
STUB_RUNS=5; "$KICKER"
echo "5 $(( $(date +%s) - STUB_INTERVAL - 5 ))" > "$LABEL_STATE"
"$KICKER"; "$KICKER"; "$KICKER"     # two further passes, both within the interval
if [ "$(kicks)" = "1" ]; then ok "1 attempt, then backs off for a full interval"; else fail "expected 1 attempt, got $(kicks) — a failing kick is retrying every pass"; fi

echo
[ "$FAILED" = "0" ] && { echo "PASS — all arms"; exit 0; }
echo "FAILED"; exit 1
