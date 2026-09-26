#!/bin/bash
# test_mission_memgate.sh — the boot stagger and the memory admission gate.
#
# WHY THIS EXISTS. The rig ran out of memory three times in two days
# (JetsamEvent 2026-09-04 05:08, 09-05 08:29, 09-05 09:23), each at ~60 MB free
# with a compressor holding 131 GB. The ollama caps added on 09-03 held — its
# footprint was 25.77 GB at all three events, identical to two decimals — so the
# growth term was the fleet itself: four missions, every plist carrying
# RunAtLoad=true, all firing within seconds of a boot.
#
# Two of the arms below pin bugs that were IN this code and shipped only because
# they were caught by running it:
#   * arm "uptime parses sec, not usec" — the first version used
#     `sed 's/.*sec *= *\([0-9]*\).*/\1/'`, whose greedy `.*` runs past `sec` into
#     the `sec` of `usec`. It reported 1788394233s of uptime on a box 20 minutes
#     old, which would have disabled the stagger silently in exactly the window
#     it exists for.
#   * arm "available is not free alone" — a free-only threshold cannot work: free
#     was 66 MB at the OOM event while 7.7 GB sat reclaimable in `inactive`.
#
# Extraction, not duplication (same contract as test_mission_stall.sh): the
# functions under test are awk'd out of mission-control.sh, so this suite cannot
# drift green against an edited driver.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"

TMP="${TMPDIR:-/tmp}/mc-memgate-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

for fn in _mc_uptime_secs _mc_boot_offset _mc_mem_snapshot _mc_mem_ok; do
  awk "/^${fn}\(\) \{/,/^\}\$/" "$DRIVER" > "$TMP/fn_${fn}.sh"
  [ -s "$TMP/fn_${fn}.sh" ] \
    || { echo "FAIL extraction: $fn not found in $DRIVER"; exit 1; }
  # shellcheck source=/dev/null
  . "$TMP/fn_${fn}.sh"
done
# Guard that we got the REAL parser, not a stub that happens to share the name.
grep -q 'kern.boottime' "$TMP/fn__mc_uptime_secs.sh" \
  || { echo "FAIL extraction: _mc_uptime_secs does not read kern.boottime"; exit 1; }
grep -q 'Pages occupied by compressor' "$TMP/fn__mc_mem_snapshot.sh" \
  || { echo "FAIL extraction: _mc_mem_snapshot does not read the compressor"; exit 1; }

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "ok - $1"; }
bad()  { FAIL=$((FAIL+1)); echo "not ok - $1"; }
check(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1 (want=$3 got=$2)"; fi; }

# ---------------------------------------------------------------------------
# Fixtures. vm_stat and sysctl are stubbed at their seams so every arm runs the
# driver's real parser against real command output.
# ---------------------------------------------------------------------------

# Captured from the rig, 2026-09-05 13:20, healthy idle box (128 GB, 16 KB pages).
VMSTAT_HEALTHY='Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                  5741916.
Pages active:                                1054909.
Pages inactive:                               817951.
Pages speculative:                            299430.
Pages throttled:                                   0.
Pages wired down:                             280873.
Pages purgeable:                                30368.
Pages occupied by compressor:                      0.'

# Reconstructed from JetsamEvent-2026-09-05-092353.ips memoryStatus.memoryPages:
# free 4030, inactive 506169, speculative 177, purgeable 0, compressor 4316065.
VMSTAT_OOM='Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                     4030.
Pages active:                                 507649.
Pages inactive:                               506169.
Pages speculative:                               177.
Pages throttled:                                   0.
Pages wired down:                            3001288.
Pages purgeable:                                   0.
Pages occupied by compressor:                4316065.'

# The pathological middle case the compressor arm exists for: plenty of
# reclaimable inactive memory, but the machine is already deep in paging.
VMSTAT_COMPRESSED='Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                   200000.
Pages active:                                 507649.
Pages inactive:                              1400000.
Pages speculative:                               177.
Pages throttled:                                   0.
Pages wired down:                            3001288.
Pages purgeable:                                   0.
Pages occupied by compressor:                4316065.'

VMSTAT_FIXTURE="$VMSTAT_HEALTHY"
vm_stat() { printf '%s\n' "$VMSTAT_FIXTURE"; }

# The driver's defaults, restated so the arms below test the SHIPPED thresholds.
MEM_MIN_AVAIL_MB=$((16 * 1024))
MEM_MAX_COMP_MB=$((48 * 1024))

# ---------------------------------------------------------------------------
# 1. Uptime parse. The arm that pins the greedy-sed bug.
# ---------------------------------------------------------------------------
SYSCTL_OUT='{ sec = 1788604844, usec = 123456 } Fri Sep  5 13:00:44 2026'
sysctl() { printf '%s\n' "$SYSCTL_OUT"; }
date() { echo 1788608444; }   # boot + 3600s

check "uptime parses sec, not usec" "$(_mc_uptime_secs)" "3600"

SYSCTL_OUT='{ sec = 1788608400, usec = 900001 } Fri Sep  5 13:00:44 2026'
check "uptime inside the boot window" "$(_mc_uptime_secs)" "44"

# A usec larger than any plausible uptime is the exact shape the old bug
# produced; if the parser ever regresses this arm reads ~1.7 billion.
SYSCTL_OUT='{ sec = 1788608000, usec = 999999 } Fri Sep  5 13:00:44 2026'
check "large usec does not leak into the answer" "$(_mc_uptime_secs)" "444"

sysctl() { return 1; }   # no kern.boottime (non-Darwin, or restricted)
_mc_uptime_secs >/dev/null 2>&1 && bad "missing kern.boottime must rc=1" \
                                || ok "missing kern.boottime must rc=1"
unset -f date

# ---------------------------------------------------------------------------
# 2. Boot offsets. Spacing is the property, not the individual numbers: two
#    missions sharing an offset would reintroduce the stampede for that pair.
# ---------------------------------------------------------------------------
check "v1 is not delayed"        "$(_mc_boot_offset v1)"     "0"
check "world offset"             "$(_mc_boot_offset world)"  "420"
check "docs offset"              "$(_mc_boot_offset docs)"   "840"
check "motoko offset"            "$(_mc_boot_offset motoko)" "1260"
check "unknown mission does not borrow a slot" "$(_mc_boot_offset bogus)" "0"
check "empty mission name is safe"             "$(_mc_boot_offset)"       "0"

_offs=$(for m in v1 world docs motoko; do _mc_boot_offset "$m"; done | sort -n)
_uniq=$(printf '%s\n' "$_offs" | sort -nu)
check "every mission has a distinct offset" \
  "$(printf '%s\n' "$_offs" | wc -l | tr -d ' ')" \
  "$(printf '%s\n' "$_uniq" | wc -l | tr -d ' ')"

# The spacing must exceed a controller's startup preamble; the worst measured is
# the v1 slot that burned 240s on opus probes (iter-315 note in the driver).
_min_gap=99999
_prev=""
for o in $_offs; do
  [ -n "$_prev" ] && { g=$((o - _prev)); [ "$g" -lt "$_min_gap" ] && _min_gap=$g; }
  _prev=$o
done
if [ "$_min_gap" -ge 300 ]; then ok "offset spacing (${_min_gap}s) clears the startup preamble"
else bad "offset spacing (${_min_gap}s) is under the 300s preamble floor"; fi

# ---------------------------------------------------------------------------
# 3. Memory snapshot. Numbers are pages*16384/1MiB from the fixtures above.
# ---------------------------------------------------------------------------
VMSTAT_FIXTURE="$VMSTAT_HEALTHY"
check "healthy snapshot" "$(_mc_mem_snapshot)" "107651 0"

VMSTAT_FIXTURE="$VMSTAT_OOM"
check "OOM-event snapshot" "$(_mc_mem_snapshot)" "7974 67438"

# free alone was 62 MB at that event. If anyone rewrites available as free-only,
# this arm is what tells them the threshold can no longer be set sanely.
VMSTAT_FIXTURE="$VMSTAT_OOM"
_avail=$(_mc_mem_snapshot); _avail=${_avail%% *}
if [ "$_avail" -gt 1000 ]; then ok "available is not free alone (${_avail}MB vs 62MB free)"
else bad "available collapsed to free-only (${_avail}MB)"; fi

vm_stat() { return 1; }
_mc_mem_snapshot >/dev/null 2>&1 && bad "unreadable vm_stat must rc=1" \
                                 || ok "unreadable vm_stat must rc=1"
vm_stat() { printf 'not vm_stat output at all\n'; }
_mc_mem_snapshot >/dev/null 2>&1 && bad "garbage vm_stat must rc=1" \
                                 || ok "garbage vm_stat must rc=1"
vm_stat() { printf '%s\n' "$VMSTAT_FIXTURE"; }

# ---------------------------------------------------------------------------
# 4. The gate verdict. Both observed states, and the arm each one exercises.
# ---------------------------------------------------------------------------
gate() { VMSTAT_FIXTURE="$1"; s=$(_mc_mem_snapshot); _mc_mem_ok "${s%% *}" "${s##* }"; }

gate "$VMSTAT_HEALTHY"    && ok "healthy box starts an iteration" \
                          || bad "healthy box starts an iteration"
gate "$VMSTAT_OOM"        && bad "OOM-event box must be refused" \
                          || ok "OOM-event box must be refused"
gate "$VMSTAT_COMPRESSED" && bad "deep-paging box must be refused on the compressor arm" \
                          || ok "deep-paging box must be refused on the compressor arm"

# Boundaries, so a threshold change is a deliberate edit rather than an accident.
_mc_mem_ok "$MEM_MIN_AVAIL_MB" 0 && ok "avail exactly at the floor passes" \
                                 || bad "avail exactly at the floor passes"
_mc_mem_ok $((MEM_MIN_AVAIL_MB - 1)) 0 && bad "one MB under the floor must refuse" \
                                       || ok "one MB under the floor must refuse"
_mc_mem_ok 100000 "$MEM_MAX_COMP_MB" && ok "compressor exactly at the ceiling passes" \
                                     || bad "compressor exactly at the ceiling passes"
_mc_mem_ok 100000 $((MEM_MAX_COMP_MB + 1)) && bad "one MB over the ceiling must refuse" \
                                           || ok "one MB over the ceiling must refuse"

echo
# The boot stagger reads kern.boottime via `sysctl`, which lives in /usr/sbin. The v1 and
# docs plists set an EnvironmentVariables PATH that omits it, so the stagger shipped INERT
# on those missions (2026-09-05) — it logged "kern.boottime unreadable" and passed straight
# through. The driver must guarantee reachability itself rather than trusting the plist.
if grep -q 'export PATH=.*:/usr/sbin' "$DRIVER"; then
  echo "ok - driver puts /usr/sbin on PATH so sysctl (and the boot stagger) resolve"
  PASS=$((PASS+1))
else
  echo "FAIL - /usr/sbin missing from the driver PATH: boot stagger is inert on any plist that sets PATH"
  FAIL=$((FAIL+1))
fi
echo "PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
