#!/bin/bash
# test_mission_base.sh — non-vacuity test for the mission-base.sh base recorder + drift guard.
# bash 3.2.57-safe, no network: fabricates a scratch clone + temp AILANG_STATE_DIR, exactly
# like test_mission_heartbeat.sh fabricates its state. No real sibling anywhere.
set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
HELPER="${MISSION_BASE_HELPER:-$ROOT/tools/launchd/mission-base.sh}"
pass=0
fail=0

ok() { pass=$((pass + 1)); echo "ok $pass - $1"; echo "PASS: $1"; }
bad() { fail=$((fail + 1)); echo "not ok - $1"; }
check() { if eval "$2"; then ok "$1"; else bad "$1"; fi; }
tmpdir() { mktemp -d "${TMPDIR:-/tmp}/mission-base.XXXXXX"; }

# A scratch clone with a single commit A, refs/remotes/origin/dev pinned to it and origin/HEAD
# pointing at it, so the helper's derived REF (origin/dev) resolves with no remote and no sibling.
scratch_clone() {
  local d
  d=$(tmpdir)
  git init -q "$d" 2>/dev/null
  ( cd "$d" && git config user.email test@test && git config user.name test \
      && echo a > f && git add f && git commit -qm A \
      && git update-ref refs/remotes/origin/dev HEAD \
      && git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/dev )
  echo "$d"
}

# Like scratch_clone, but the default branch is main: origin/main + origin/HEAD, NO origin/dev.
scratch_clone_main() {
  local d
  d=$(tmpdir)
  git init -q "$d" 2>/dev/null
  ( cd "$d" && git config user.email test@test && git config user.name test \
      && echo a > f && git add f && git commit -qm A \
      && git update-ref refs/remotes/origin/main HEAD \
      && git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main )
  echo "$d"
}

# snap-format: snap prints <40-hex-sha><TAB><ISO8601-UTC> and exits 0 for origin/dev.
d=$(scratch_clone); t=$(tmpdir)
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" snap); rc=$?
check "snap-format prints 40-hex-sha<TAB>ISO8601-UTC and exits 0" \
  "[ '$rc' -eq 0 ] && printf '%s' \"$out\" | awk -F '\t' 'NF==2 && \$1 ~ /^[0-9a-f]{40}\$/ && \$2 ~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z\$/ {ok=1} END {exit !ok}'"
rm -rf "$d" "$t"

# record-last-roundtrip: record gate1 appends exactly one base-gate1 row; last returns that SHA.
d=$(scratch_clone); t=$(tmpdir)
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1); rc=$?
sha=${out%%$'\t'*}
check "record-last-roundtrip appends one base-gate1 row and last returns the SHA" \
  "[ '$rc' -eq 0 ] && [ \"\$(grep -c 'base-gate1' '$t/mission-test-base')\" -eq 1 ] && [ \"\$(cd '$d' && MISSION_NAME=test AILANG_STATE_DIR='$t' '$HELPER' last gate1)\" = '$sha' ]"
rm -rf "$d" "$t"

# heartbeat-untouched: the record stays OFF the heartbeat (protects the driver's slot-verdict reader).
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 >/dev/null)
check "heartbeat-untouched leaves mission-test-heartbeat absent" \
  "[ ! -e '$t/mission-test-heartbeat' ]"
rm -rf "$d" "$t"

# nonvacuity-drift-fires: record at A, advance the shared ref to B (sibling fetch sim), drift exits 1.
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 >/dev/null)
( cd "$d" && echo b > f && git add f && git commit -qm B && git update-ref refs/remotes/origin/dev HEAD )
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" drift gate1); rc=$?
check "nonvacuity-drift-fires exits 1 and prints DRIFT A -> B" \
  "[ '$rc' -eq 1 ] && printf '%s' \"$out\" | grep -q 'DRIFT base gate1' && printf '%s' \"$out\" | grep -q ' -> '"
rm -rf "$d" "$t"

# steady-control: without the mutation, drift exits 0 (base gate1 steady at ...).
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 >/dev/null)
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" drift gate1); rc=$?
check "steady-control exits 0 with base gate1 steady" \
  "[ '$rc' -eq 0 ] && printf '%s' \"$out\" | grep -q 'base gate1 steady at'"
rm -rf "$d" "$t"

# no-record-absent-file: no state file at all -> drift exits 2 with no base-gate1 record yet.
d=$(scratch_clone); t=$(tmpdir)
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" drift gate1 2>&1); rc=$?
check "no-record-absent-file exits 2 with no base-gate1 record yet" \
  "[ '$rc' -eq 2 ] && printf '%s' \"$out\" | grep -q 'no base-gate1 record yet'"
rm -rf "$d" "$t"

# no-record-missing-label: state file exists but has no matching base- row -> exit 2, never false DRIFT.
d=$(scratch_clone); t=$(tmpdir)
printf '0\tnow\tbase-other\t1\tdeadbeef\n' > "$t/mission-test-base"
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" drift gate1 2>&1); rc=$?
check "no-record-missing-label exits 2 with no base-gate1 record yet (never false DRIFT)" \
  "[ '$rc' -eq 2 ] && printf '%s' \"$out\" | grep -q 'no base-gate1 record yet'"
rm -rf "$d" "$t"

# positive grep control: last against a known-matching record returns it (instrument runs).
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 >/dev/null)
out=$(cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" last gate1); rc=$?
check "positive grep control: last returns a known-matching record" \
  "[ '$rc' -eq 0 ] && printf '%s' \"$out\" | grep -Eq '^[0-9a-f]{40}\$'"
rm -rf "$d" "$t"

# T1 derived-main-record: default branch main, no origin/dev -> snap/record resolve via origin/HEAD.
d=$(scratch_clone_main); t=$(tmpdir)
(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" snap >/dev/null); rc1=$?
out=$(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1); rc2=$?
sha=${out%%$'\t'*}; want=$(cd "$d" && git rev-parse refs/remotes/origin/main)
check "derived-main-record snaps and records origin/main's SHA and last returns it" \
  "[ '$rc1' -eq 0 ] && [ '$rc2' -eq 0 ] && [ '$sha' = '$want' ] && [ \"\$(cd '$d' && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR='$t' '$HELPER' last gate1)\" = '$want' ]"
rm -rf "$d" "$t"

# T2 override-beats-origin-head: explicit MISSION_BASE_REF wins over origin/HEAD.
d=$(scratch_clone_main); t=$(tmpdir)
a=$(cd "$d" && git rev-parse HEAD)
( cd "$d" && echo b > f && git add f && git commit -qm B && git update-ref refs/remotes/origin/release HEAD )
b=$(cd "$d" && git rev-parse HEAD)
out=$(cd "$d" && MISSION_BASE_REF=origin/release MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1); rc=$?
sha=${out%%$'\t'*}
check "override-beats-origin-head records the override ref's SHA, not origin/HEAD's" \
  "[ '$rc' -eq 0 ] && [ '$sha' = '$b' ] && [ '$sha' != '$a' ]"
rm -rf "$d" "$t"

# T3 no-origin-head-fails-loudly: origin/dev exists but origin/HEAD is unset -> rc 1, both remedies, no row.
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && git symbolic-ref --delete refs/remotes/origin/HEAD)
err=$(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 2>&1 >/dev/null); rc=$?
check "no-origin-head-fails-loudly exits 1 naming MISSION_BASE_REF and set-head, writes no state" \
  "[ '$rc' -eq 1 ] && printf '%s' \"$err\" | grep -q 'MISSION_BASE_REF' && printf '%s' \"$err\" | grep -q 'set-head' && [ ! -e '$t/mission-test-base' ]"
rm -rf "$d" "$t"

# T4 drift-uses-derived-ref: origin/dev stays at A, origin/HEAD's target (main) moves to B -> drift fires.
d=$(scratch_clone_main); t=$(tmpdir)
(cd "$d" && git update-ref refs/remotes/origin/dev HEAD)
(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" record gate1 >/dev/null)
( cd "$d" && echo b > f && git add f && git commit -qm B && git update-ref refs/remotes/origin/main HEAD )
b=$(cd "$d" && git rev-parse HEAD)
out=$(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" drift gate1); rc=$?
check "drift-uses-derived-ref exits 1 with DRIFT base gate1 and the new origin/main SHA" \
  "[ '$rc' -eq 1 ] && printf '%s' \"$out\" | grep -q 'DRIFT base gate1' && printf '%s' \"$out\" | grep -q '$b'"
rm -rf "$d" "$t"

# T5 last-needs-no-ref: last reads only the state file, so it works with no origin/HEAD.
d=$(scratch_clone); t=$(tmpdir)
(cd "$d" && git symbolic-ref --delete refs/remotes/origin/HEAD)
printf '0\tnow\tbase-gate1\t1\tcafef00dcafef00dcafef00dcafef00dcafef00d\n' > "$t/mission-test-base"
out=$(cd "$d" && env -u MISSION_BASE_REF MISSION_NAME=test AILANG_STATE_DIR="$t" "$HELPER" last gate1); rc=$?
check "last-needs-no-ref exits 0 and prints the row's SHA with no origin/HEAD" \
  "[ '$rc' -eq 0 ] && [ '$out' = 'cafef00dcafef00dcafef00dcafef00dcafef00d' ]"
rm -rf "$d" "$t"

if [ "$fail" -eq 0 ]; then
  echo "PASS: $pass mission-base arms ran"
  exit 0
fi
echo "FAIL: $fail mission-base arms failed"
exit 1
