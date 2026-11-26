#!/bin/bash
# test_mission_base.sh — non-vacuity test for the mission-base.sh base recorder + drift guard.
# bash 3.2.57-safe, no network: fabricates a scratch clone + temp AILANG_STATE_DIR, exactly
# like test_mission_heartbeat.sh fabricates its state. No real sibling anywhere.
set -u
unset MISSION_REPO MISSION_BASE_REF

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

new_fixture() {
  d=$(scratch_clone) || exit 1
  t=$(tmpdir) || exit 1
  if [ "$#" -gt 0 ]; then
    git -C "$d" config --local --add remote.origin.url "$1" || exit 1
  fi
}

run_helper() {
  (cd "$d" && MISSION_NAME=test AILANG_STATE_DIR="$t" /bin/bash "$HELPER" "$@")
}

check_record() {
  local out want
  if want=$(git -C "$d" rev-parse refs/remotes/origin/dev) &&
     out=$(run_helper record gate1) && [[ "$out" == "$want"$'\t'* ]] &&
     awk -F '\t' -v sha="$want" '$3=="base-gate1" && $5==sha {found=1} END {exit !(NR==1 && found)}' "$t/mission-test-base"; then
    ok "$1"
  else
    bad "$1"
  fi
}

observe_base() {
  local file="$t/mission-test-base" digest
  if [ ! -e "$file" ]; then printf 'absent\n'; return 0; fi
  digest=$(shasum -a 256 < "$file") || return 1
  printf '%s\n' "$digest"
  wc -l < "$file"
}

check_refusal() {
  local name="$1" actual="$2" before after rc
  shift 2
  before=$(observe_base) || exit 1
  run_helper "$@" > "$t/stdout" 2> "$t/stderr"; rc=$?
  after=$(observe_base) || exit 1
  if [ "$rc" -eq 3 ] && [ ! -s "$t/stdout" ] && [ "$before" = "$after" ] &&
     grep -Fxq "mission-base: CWD origin is $actual, expected $MISSION_REPO (MISSION_REPO)" "$t/stderr"; then
    ok "$name"
  else
    bad "$name"
  fi
}

new_fixture https://github.com/sunholo-data/ailang.git
MISSION_REPO=sunholo-data/ailang-world MISSION_BASE_REF=origin/dev \
  check_refusal 'wrong repo with ref override leaves base absent' sunholo-data/ailang record gate1
rm -rf "$d" "$t"

new_fixture https://github.com/sunholo-data/ailang.git
if run_helper record gate1 >/dev/null && [ -f "$t/mission-test-base" ]; then
  MISSION_REPO=sunholo-data/ailang-world MISSION_BASE_REF=origin/dev \
    check_refusal 'wrong repo with ref override preserves recorded base' sunholo-data/ailang record gate1
else
  bad 'wrong repo with ref override preserves recorded base (setup record failed)'
fi
rm -rf "$d" "$t"

for origin in https://github.com/sunholo-data/ailang git@github.com:Sunholo-Data/AILANG ssh://git@github.com/Sunholo-Data/AILANG; do
  for suffix in '' .git .git/; do
    new_fixture "$origin$suffix"
    MISSION_REPO=sunholo-data/ailang check_record "matching origin $origin$suffix records one row"
    rm -rf "$d" "$t"
  done
done

new_fixture https://github.com/sunholo-data/ailang.git
MISSION_REPO=SUNHOLO-DATA/AILANG check_record 'mixed-case MISSION_REPO matches a lowercase origin slug'
rm -rf "$d" "$t"

new_fixture
check_record 'unset MISSION_REPO permits record without an origin URL'
rm -rf "$d" "$t"

new_fixture
MISSION_REPO='' check_record 'empty MISSION_REPO permits record without an origin URL'
rm -rf "$d" "$t"

new_fixture
MISSION_REPO=sunholo-data/ailang check_refusal 'missing origin URL is refused' 'unavailable (missing origin)' record gate1
rm -rf "$d" "$t"

d=$(tmpdir); t=$(tmpdir)
GIT_CEILING_DIRECTORIES="${d%/*}" MISSION_REPO=sunholo-data/ailang \
  check_refusal 'outside Git is refused' 'unavailable (outside Git)' record gate1
rm -rf "$d" "$t"

for origin in https://other.example/sunholo-data/ailang https://github.com/sunholo-data/ailang/extra; do
  new_fixture "$origin"
  MISSION_REPO=sunholo-data/ailang \
    check_refusal "unsupported origin $origin is refused" 'unavailable (unsupported or ambiguous origin)' record gate1
  rm -rf "$d" "$t"
done

new_fixture https://github.com/sunholo-data/ailang
git -C "$d" config --local --add remote.origin.url '' || exit 1
MISSION_REPO=sunholo-data/ailang \
  check_refusal 'multiple origin URLs including a blank entry are refused' 'unavailable (unsupported or ambiguous origin)' record gate1
rm -rf "$d" "$t"

for command in snap drift; do
  set -- "$command"
  if [ "$command" = drift ]; then set -- "$@" gate1; fi
  new_fixture https://github.com/sunholo-data/ailang
  git -C "$d" symbolic-ref --delete refs/remotes/origin/HEAD || exit 1
  MISSION_REPO=sunholo-data/ailang-world \
    check_refusal "wrong repo for $command is refused before deriving a ref" sunholo-data/ailang "$@"
  rm -rf "$d" "$t"

  new_fixture https://github.com/sunholo-data/ailang
  if MISSION_BASE_REF=origin/dev run_helper record gate1 >/dev/null; then
    MISSION_REPO=sunholo-data/ailang-world MISSION_BASE_REF=origin/missing \
      check_refusal "wrong repo for $command is refused with an unresolvable ref" sunholo-data/ailang "$@"
  else
    bad "wrong repo for $command is refused with an unresolvable ref (setup record failed)"
  fi
  rm -rf "$d" "$t"
done

new_fixture
if rec=$(run_helper record gate1) && out=$(MISSION_REPO=sunholo-data/ailang run_helper last gate1) &&
   [ -n "$out" ] && [ "$out" = "${rec%%$'\t'*}" ]; then
  ok 'last returns a recorded SHA with MISSION_REPO set and no origin URL'
else
  bad 'last returns a recorded SHA with MISSION_REPO set and no origin URL'
fi
rm -rf "$d" "$t"

if [ "$fail" -eq 0 ]; then
  echo "PASS: $pass mission-base arms ran"
  exit 0
fi
echo "FAIL: $fail mission-base arms failed"
exit 1
