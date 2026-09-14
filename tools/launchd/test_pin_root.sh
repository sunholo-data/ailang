#!/bin/bash
# End-to-end test for tools/launchd/lib/pin-root.sh under bash 3.2.
# Builds a real origin + a deliberately-stale clone, then asserts on OBSERVED behaviour.
set -uo pipefail

SP="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SRC_HELPER="$REPO_ROOT/tools/launchd/lib/pin-root.sh"
T="${TMPDIR:-/tmp}/pinlab.$$"
rm -rf "$T"; mkdir -p "$T"
# Normalise: macOS $TMPDIR carries a trailing slash, so the raw string would not match the
# realpath the driver reports, and the path assertions below would fail on punctuation.
T="$(cd "$T" && pwd)"
trap 'rm -rf "$T"' EXIT

# Hermetic identity + branch name. A CI runner has no user.name/user.email and may default
# `init.defaultBranch` to anything, so without these the lab commits fail there and the suite
# reports a git-config problem as a pin failure — a test that only passes on the author's box.
export GIT_AUTHOR_NAME=pinlab GIT_AUTHOR_EMAIL=pinlab@invalid
export GIT_COMMITTER_NAME=pinlab GIT_COMMITTER_EMAIL=pinlab@invalid
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null

# A scheduled driver re-execs with these variables exported. They describe the
# real driver's outer pin and must not leak into the synthetic drivers below:
# inherited AILANG_DRIVER_PINNED makes every case short-circuit as "already
# pinned", which turns CI green while the test fails on the rig it protects.
unset AILANG_DRIVER_PINNED AILANG_DRIVER_DRIFT AILANG_DRIVER_SRC AILANG_DRIVER_REF
unset AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA
unset PIN_AGE_SUPPORTED
unset AILANG_DRIVER_PIN_GATE_REFRESHED
unset MISSION_WORKDIR AILANG_DRIVER_MISSION_IS_DE_FORKED

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  FAIL: $1"; echo "        got: $2"; }
check(){ # name haystack needle
  case "$2" in *"$3"*) ok "$1";; *) bad "$1" "$(printf '%s' "$2" | tr '\n' '|' | tail -c 300)";; esac
}
checkeq(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1" "$2"; fi; }
checkno(){ case "$2" in *"$3"*) bad "$1" "$(printf '%s' "$2" | tr '\n' '|' | tail -c 300)";; *) ok "$1";; esac; }
# Line-anchored value assertion (Lane Rule 9): the fake driver prints both AGE=<n> and
# ENV_AGE=<n>, so a substring check on 'AGE=3' would alias ENV_AGE=3 and vacuously green a
# broken readback (MUT-D). Anchoring on ^NAME=value$ disambiguates.
checkline(){ # name haystack ^name=value$
  if printf '%s\n' "$2" | grep -q "^$3$"; then ok "$1"; else bad "$1" "$2"; fi;
}

# ---- build origin ----------------------------------------------------------
git init --quiet --bare "$T/origin.git"
git clone --quiet "$T/origin.git" "$T/seed" 2>/dev/null
cd "$T/seed"
git checkout --quiet -b dev
mkdir -p tools/launchd/lib
cp "$SRC_HELPER" tools/launchd/lib/pin-root.sh
cat > tools/launchd/fake-driver.sh <<'DRIVER'
#!/usr/bin/env bash
set -uo pipefail
REPO="${MISSION_WORKDIR:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$REPO" || exit 1
LOG=/dev/null
log() { echo "[drv] $*"; }
DRIVER_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [ -f "$DRIVER_ROOT/tools/launchd/lib/pin-root.sh" ]; then
  . "$DRIVER_ROOT/tools/launchd/lib/pin-root.sh"
  pin_root_to_committed_ref "$@"
else
  PIN_STATUS="STALE"; PIN_NOTE="helper absent"
fi
echo "STATUS=$PIN_STATUS"
echo "DRIFT=$PIN_DRIFT"
echo "AGE=$PIN_AGE"
echo "AGE_BASE_SHA=$PIN_AGE_BASE_SHA"
echo "ENV_AGE=${AILANG_DRIVER_AGE:-unset}"
echo "ENV_AGE_BASE_SHA=${AILANG_DRIVER_AGE_BASE_SHA:-unset}"
echo "ENV_REF=${AILANG_DRIVER_REF:-unset}"
echo "NOTE=$PIN_NOTE"
echo "REPO=$REPO"
echo "MARKER=$(cat "$REPO/MARKER" 2>/dev/null)"
echo "ARGS=$*"
DRIVER
chmod +x tools/launchd/fake-driver.sh
echo "STALE-CONTENT" > MARKER
git add -A >/dev/null; git commit --quiet -m "base"
git push --quiet origin dev 2>/dev/null
A=$(git -C "$T/seed" rev-parse HEAD)

# clone that will go stale
git clone --quiet --branch dev "$T/origin.git" "$T/clone" 2>/dev/null

# Make the clone's already-installed gate stale while origin/dev retains the current gate. This
# models the bootstrap trap directly: the local predicate knows only the retired legacy key, but
# still carries the durable refresh hop added by this row. If the hop is removed, arm 8c reds.
perl -0pi -e 's/\(\(\.projects\[\$p\]\.hasCompletedProjectOnboarding == true\) or \(\.projects\[\$p\]\.hasTrustDialogAccepted == true\)\)/(\.projects[\$p].hasCompletedProjectOnboarding == true)/' "$T/clone/tools/launchd/lib/pin-root.sh"
if ! grep -q 'hasCompletedProjectOnboarding == true)' "$T/clone/tools/launchd/lib/pin-root.sh" ||
   grep -q 'hasCompletedProjectOnboarding == true) or' "$T/clone/tools/launchd/lib/pin-root.sh"; then
  echo "fixture error: failed to install stale local onboarding predicate" >&2
  exit 1
fi

# advance origin ONE commit: the clone is now genuinely behind
echo "FRESH-CONTENT" > MARKER
git commit --quiet -am "advance"
git push --quiet origin dev 2>/dev/null
cd "$T"

DRV="$T/clone/tools/launchd/fake-driver.sh"
export AILANG_DRIVER_PIN_DIR="$T/pinwt"

# Fake HOME carrying a synthetic ~/.claude.json, so the ONBOARDING GATE is exercised for real
# rather than switched off for the happy path. Marking the pin path onboarded here is what makes
# tests 1-5 meaningful: if the gate regressed to always-refuse they would go red, and if it
# regressed to always-allow test 7 would.
export HOME="$T/home"; mkdir -p "$HOME"
cat > "$HOME/.claude.json" <<JSON
{"projects": {"$T/pinwt": {"hasCompletedProjectOnboarding": true}}}
JSON

echo "== 1. happy path: stale clone re-execs into committed origin/dev =="
OUT=$(/bin/bash "$DRV" alpha beta 2>&1)
check "status is pinned"                 "$OUT" "STATUS=pinned"
check "drift measured as 1"              "$OUT" "DRIFT=1"
check "ROOT moved to the pin worktree"   "$OUT" "REPO=$T/pinwt"
check "reads FRESH content, not stale"   "$OUT" "MARKER=FRESH-CONTENT"
checkno "stale content is NOT read"      "$OUT" "MARKER=STALE-CONTENT"
check "args survive the re-exec"         "$OUT" "ARGS=alpha beta"

# AC-A: at the origin/dev tip, age is 0 and the baseline is origin/dev's full commit.
DEV=$(git -C "$T/seed" rev-parse "dev^{commit}" 2>/dev/null)
checkline "age is exactly 0 at the origin/dev tip"  "$OUT" "AGE=0"
checkline "baseline SHA is origin/dev's full commit" "$OUT" "AGE_BASE_SHA=$DEV"
check "note carries the pinned-age clause"            "$OUT" "pinned target 0 behind origin/dev (baseline"


# REGRESSION: the SECOND fire must pin too. The first implementation matched `worktree list`
# by string, which never matches a realpath-resolved entry, so fire 2 hit `worktree add` on an
# existing directory and refused. A one-shot fix that reports STALE forever after.
OUT2=$(/bin/bash "$DRV" 2>&1)
check "second consecutive fire pins"     "$OUT2" "STATUS=pinned"
check "and still reads FRESH content"    "$OUT2" "MARKER=FRESH-CONTENT"

echo "== 2. control: the clone really was stale (instrument check) =="
CTL=$(AILANG_DRIVER_PIN=0 /bin/bash "$DRV" 2>&1)
check "opt-out reports disabled"         "$CTL" "STATUS=disabled"
check "opt-out reads STALE content"      "$CTL" "MARKER=STALE-CONTENT"
check "opt-out root stays in the clone"  "$CTL" "REPO=$T/clone"

echo "== 3. no recursion: second pass returns without re-exec =="
REC=$(AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_DRIFT=7 /bin/bash "$DRV" 2>&1)
check "already-pinned short-circuits"    "$REC" "STATUS=pinned"
check "carries drift across the exec"    "$REC" "DRIFT=7"
# AC-C: age + baseline cross the exec on the already-pinned pass (T3 readback).
RAC=$(AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_DRIFT=7 AILANG_DRIVER_AGE=3 AILANG_DRIVER_AGE_BASE_SHA=0000000000000000000000000000000000000001 /bin/bash "$DRV" 2>&1)
checkline "carries age across the exec"        "$RAC" "AGE=3"
checkline "carries the age baseline across the exec" "$RAC" "AGE_BASE_SHA=0000000000000000000000000000000000000001"
# D-2 witness: missing age must carry as ?, never zero (kills MUT-G).
RMG=$(AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_DRIFT=7 /bin/bash "$DRV" 2>&1)
checkline "missing age carries as ?, never zero" "$RMG" "AGE=?"


echo "== 4. fetch failure is STALE + loud, never silent-ok =="
git -C "$T/clone" remote set-url origin "$T/does-not-exist.git"
FF=$(AILANG_DRIVER_FETCH_TIMEOUT=20 /bin/bash "$DRV" 2>&1)
check "status STALE"                     "$FF" "STATUS=STALE"
check "reason names the fetch"           "$FF" "git fetch origin failed"
check "driver still ran (fail-open)"     "$FF" "MARKER=STALE-CONTENT"
checkno "never reports pinned"           "$FF" "STATUS=pinned"
git -C "$T/clone" remote set-url origin "$T/origin.git"

echo "== 5. driver absent FROM THE REF => refuse to exec into nothing =="
# The driver must EXIST locally (so $0 resolves and the guard is what fires) but be absent from
# origin/dev — the shape a rename-on-dev would produce. A test asserting bash's own
# "No such file" would pass without the guard ever running.
cp "$T/clone/tools/launchd/fake-driver.sh" "$T/clone/tools/launchd/local-only.sh"
MD=$(AILANG_DRIVER_PIN_DIR="$T/pinwt2" /bin/bash "$T/clone/tools/launchd/local-only.sh" 2>&1)
check "guard fires, not bash"            "$MD" "has no tools/launchd/local-only.sh"
check "status STALE"                     "$MD" "STATUS=STALE"
checkno "did NOT exec into nothing"      "$MD" "No such file"

echo "== 6. non-repo source => STALE, not a crash =="
mkdir -p "$T/plain/tools/launchd/lib"
cp "$SRC_HELPER" "$T/plain/tools/launchd/lib/pin-root.sh"
cp "$T/clone/tools/launchd/fake-driver.sh" "$T/plain/tools/launchd/"
NR=$(/bin/bash "$T/plain/tools/launchd/fake-driver.sh" 2>&1)
check "non-repo is STALE"                "$NR" "STATUS=STALE"
check "reason names the repo problem"    "$NR" "not a git repository"

echo "== 7. un-onboarded pin target => REFUSE, do not exec into a probe-hang =="
# The regression this exists for: a pin worktree is by construction a path Claude Code has never
# seen, and pinning into one makes every model probe hang to its timeout, then the driver refuses
# with "NO usable model in prefs" — reading as a quota outage. Cost motoko its whole first fire
# (charter V22). Staleness is the strictly smaller harm, so the pin must decline.
# An EMPTY .projects map is "nothing is onboarded yet" — the ordinary refusal, NOT schema drift.
# The gate distinguishes them on `.projects | length`, so this fixture stays the natural one.
printf '{"projects":{}}' > "$HOME/.claude.json"
UO=$(/bin/bash "$DRV" 2>&1)
check "refuses to pin"                   "$UO" "STATUS=STALE"
check "names the onboarding cause"       "$UO" "onboarded in Claude Code"
check "gives the exact human fix"        "$UO" "&& claude"
check "fire still runs, unpinned"        "$UO" "MARKER=STALE-CONTENT"
checkno "never reports pinned"           "$UO" "STATUS=pinned"

echo "== 8. current trust flag satisfies the onboarding gate =="
# Measured, not assumed: on 2026-08-26 hasCompletedProjectOnboarding was absent from 15/15
# live project entries while hasTrustDialogAccepted was present in 15/15 (control), so the
# 2026-08-12 preference for hasCompletedProjectOnboarding is void.
printf '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/pinwt" > "$HOME/.claude.json"
TR=$(/bin/bash "$DRV" 2>&1)
check "trust flag alone DOES satisfy"     "$TR" "STATUS=pinned"
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "$T/pinwt" > "$HOME/.claude.json"
OB=$(/bin/bash "$DRV" 2>&1)
check "legacy onboarding flag DOES satisfy" "$OB" "STATUS=pinned"

echo "== 8b. SOURCE-clone onboarding satisfies it — the case the first cut got WRONG =="
# Measured 2026-08-12: `claude -p` runs fine from ~/.ailang-driver-pin/v1 while ~/.claude.json
# has NO entry for it — a worktree inherits its source clone's trust. The first predicate checked
# only the worktree's own entry, so it would have refused a demonstrably working target on every
# fire and left the pin permanently off. This is the production shape: worktree absent, source ok.
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "$T/clone" > "$HOME/.claude.json"
SC=$(/bin/bash "$DRV" 2>&1)
check "source onboarding is enough"       "$SC" "STATUS=pinned"
check "and it really pinned"              "$SC" "MARKER=FRESH-CONTENT"

echo "== 8c. SOURCE-clone trust satisfies it — the exact live rig shape =="
printf '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/clone" > "$HOME/.claude.json"
SCT=$(/bin/bash "$DRV" 2>&1)
check "fetched gate overrides stale local predicate" "$SCT" "STATUS=pinned"
check "and source trust really pinned"    "$SCT" "MARKER=FRESH-CONTENT"

echo "== 8d. NEITHER onboarded => still refuse (the motoko shape) =="
# The gate must not have been widened into a no-op: a fresh clone with nothing onboarded anywhere
# is exactly what cost motoko iteration 1, and it must still be refused.
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "/some/unrelated/path" > "$HOME/.claude.json"
NN=$(/bin/bash "$DRV" 2>&1)
check "neither path onboarded => STALE"   "$NN" "STATUS=STALE"
check "message names BOTH paths"          "$NN" "nor its source clone"
checkno "refreshed gate does not delete refusal" "$NN" "STATUS=pinned"

echo "== 8e. schema drift is distinct from ordinary un-onboarded refusal =="
printf '{"projects":{"%s":{"lastCost":1.25}}}' "/some/unrelated/path" > "$HOME/.claude.json"
SD=$(/bin/bash "$DRV" 2>&1)
check "schema drift => STALE"             "$SD" "STATUS=STALE"
check "reason names schema drift"         "$SD" "Claude Code schema drift"
check "reason names legacy key"           "$SD" "hasCompletedProjectOnboarding"
check "reason names current key"          "$SD" "hasTrustDialogAccepted"
checkno "not ordinary onboarding refusal" "$SD" "neither $T/pinwt nor its source clone"
checkno "not the unreadable diagnosis"    "$SD" "cannot read a .projects object"

echo "== 8f. an UNREADABLE ~/.claude.json is NOT schema drift — three refusals, three sentences =="
# The whole point of the drift diagnosis is to tell the next reader that the GATE needs a new key.
# Saying that about a malformed or missing file sends them to fix the wrong thing, which is this
# gate's own original defect one level down. So invalid JSON, a non-object `.projects`, and a
# missing file all get the unreadable sentence instead.
printf '{"projects": ' > "$HOME/.claude.json"           # truncated => invalid JSON
BJ=$(/bin/bash "$DRV" 2>&1)
check "invalid JSON => STALE"             "$BJ" "STATUS=STALE"
check "names the unreadable cause"        "$BJ" "cannot read a .projects object"
checkno "not called schema drift"         "$BJ" "Claude Code schema drift"
checkno "not ordinary onboarding refusal" "$BJ" "Run once, interactively"
checkno "never reports pinned"            "$BJ" "STATUS=pinned"

printf '{"projects":"not-an-object"}' > "$HOME/.claude.json"
NO=$(/bin/bash "$DRV" 2>&1)
check "non-object .projects => STALE"     "$NO" "STATUS=STALE"
check "also names the unreadable cause"   "$NO" "cannot read a .projects object"
checkno "also not called schema drift"    "$NO" "Claude Code schema drift"

rm -f "$HOME/.claude.json"
MF=$(/bin/bash "$DRV" 2>&1)
check "missing file => STALE"             "$MF" "STATUS=STALE"
check "missing file names it unreadable"  "$MF" "cannot read a .projects object"

echo "== 9. undeterminable (no jq) fails SAFE, not open =="
mkdir -p "$T/nojq"
for b in git mktemp date basename dirname cat rm sleep kill grep tr tail printf mkdir; do
  p=$(command -v $b 2>/dev/null); [ -n "$p" ] && ln -sf "$p" "$T/nojq/$b"
done
NJ=$(PATH="$T/nojq" /bin/bash "$DRV" 2>&1)
check "no jq => STALE, not pinned"       "$NJ" "STATUS=STALE"
check "says it could not verify"         "$NJ" "cannot verify Claude Code onboarding"

echo "== 10. repository identity is independent of clone/worktree identity =="
. "$SRC_HELPER"
mkdir -p "$T/identity"
for repo in driver same foreign missing; do git init --quiet "$T/identity/$repo"; done
git -C "$T/identity/driver" remote add origin https://GitHub.COM/team/Repo.git/
git -C "$T/identity/same" remote add origin git@github.com:team/Repo.git
git -C "$T/identity/foreign" remote add origin ssh://git@github.com/team/repo.git
_pin_same_repo "$T/identity/driver" "$T/identity/same"; RC=$?
checkeq "SSH and HTTPS same-origin separate clones" "$RC" "0"
git -C "$T/identity/same" remote set-url origin ssh://git@GITHUB.COM/team/Repo.git
_pin_same_repo "$T/identity/driver" "$T/identity/same"; RC=$?
checkeq "SSH scheme and HTTPS origins are equivalent" "$RC" "0"
_pin_same_repo "$T/identity/driver" "$T/identity/foreign"; RC=$?
checkeq "repository path case remains significant" "$RC" "1"
for shape in same foreign; do
  MISSION_WORKDIR="$T/identity/$shape"
  _set_pin_workdir "$T/pinwt" "$T/identity/driver"; RC=$?
  EXPECT="$T/identity/$shape"; [ "$shape" = same ] && EXPECT="$T/pinwt"
  checkeq "inferred $shape repository workdir" "$RC:$MISSION_WORKDIR" "0:$EXPECT"
done
for flag in 0 1; do
  AILANG_DRIVER_MISSION_IS_DE_FORKED="$flag"
  SHAPE=foreign; EXPECT="$T/pinwt"
  [ "$flag" = 1 ] && SHAPE=same && EXPECT="$T/identity/same"
  MISSION_WORKDIR="$T/identity/$SHAPE"
  _set_pin_workdir "$T/pinwt" "$T/identity/driver"; RC=$?
  checkeq "flag $flag overrides OPPOSITE inference" "$RC:$MISSION_WORKDIR" "0:$EXPECT"
done
for flag in '' yes 2; do
  AILANG_DRIVER_MISSION_IS_DE_FORKED="$flag"; MISSION_WORKDIR="$T/identity/same"
  _set_pin_workdir "$T/pinwt" "$T/identity/driver" 2>"$T/reason"; RC=$?
  checkeq "invalid flag '$flag' fails without changing path" "$RC:$PIN_STATUS:$MISSION_WORKDIR" "1:STALE:$T/identity/same"
done
unset AILANG_DRIVER_MISSION_IS_DE_FORKED
for shape in missing absent; do
  MISSION_WORKDIR="$T/identity/$shape"
  _set_pin_workdir "$T/pinwt" "$T/identity/driver" 2>"$T/reason"; RC=$?
  checkeq "unknown $shape retains caller STALE and path" "$RC:$PIN_STATUS:$MISSION_WORKDIR" "1:STALE:$T/identity/$shape"
  check "unknown $shape emits reason" "$(cat "$T/reason")" "WARN"
done
MISSION_WORKDIR="$T/identity/same"
_set_pin_workdir "$T/pinwt" "$T/identity/missing" 2>"$T/reason"; RC=$?
checkeq "unknown driver origin retains path" "$RC:$PIN_STATUS:$MISSION_WORKDIR" "1:STALE:$T/identity/same"
for origin in '' not-an-origin 'ssh://git@github.com:2222/team/Repo.git'; do
  git -C "$T/identity/missing" config remote.origin.url "$origin"
  MISSION_WORKDIR="$T/identity/missing"
  _set_pin_workdir "$T/pinwt" "$T/identity/driver" 2>"$T/reason"; RC=$?
  checkeq "indeterminate origin '$origin' fails closed" "$RC:$PIN_STATUS:$MISSION_WORKDIR" "1:STALE:$T/identity/missing"
done
git -C "$T/identity/missing" config --unset-all remote.origin.url
git -C "$T/identity/missing" config --add remote.origin.url https://github.com/team/Repo.git
git -C "$T/identity/missing" config --add remote.origin.url https://github.com/team/Other.git
MISSION_WORKDIR="$T/identity/missing"
_set_pin_workdir "$T/pinwt" "$T/identity/driver" 2>"$T/reason"; RC=$?
checkeq "ambiguous multiple origins fail closed" "$RC:$PIN_STATUS:$MISSION_WORKDIR" "1:STALE:$T/identity/missing"
git -C "$T/identity/missing" config --unset-all remote.origin.url
unset MISSION_WORKDIR
_set_pin_workdir "$T/pinwt" "$T/identity/driver"; RC=$?
checkeq "absent incoming workdir defaults to driver source" "$RC:$MISSION_WORKDIR" "0:$T/pinwt"
unset MISSION_WORKDIR

# Exercise the production plain call across the actual driver re-exec, not just helpers.
printf '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/clone" > "$HOME/.claude.json"
echo FOREIGN-CONTENT > "$T/identity/foreign/MARKER"
FR=$(MISSION_WORKDIR="$T/identity/foreign" /bin/bash "$DRV" 2>&1)
check "foreign work repo survives real pinned re-exec" "$FR" "STATUS=pinned"
check "foreign work repo remains selected" "$FR" "REPO=$T/identity/foreign"
check "foreign charter content remains selected" "$FR" "MARKER=FOREIGN-CONTENT"
echo UNKNOWN-CONTENT > "$T/identity/missing/MARKER"
UK=$(MISSION_WORKDIR="$T/identity/missing" /bin/bash "$DRV" 2>&1)
check "unknown origin production call stays STALE" "$UK" "STATUS=STALE"
check "unknown origin production path unchanged" "$UK" "REPO=$T/identity/missing"
checkno "unknown origin never exports pinned success" "$UK" "STATUS=pinned"
PLAIN=$(grep -c '^  _set_pin_workdir "$wt" "$src" || return 1$' "$SRC_HELPER")
checkeq "production call is in current shell" "$PLAIN" "1"

# ---- M1 PIN_AGE lab arms (sections 11-16) ----
# Restore the onboarding fixture and confirm the clone's origin URL (restored in case 4) still
# points at the lab origin before the SHA-ref arms below.
printf '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/clone" > "$HOME/.claude.json"

# REAL_GIT is resolved ONCE here and baked into every PATH shim as a literal (Lane Rule 10).
REAL_GIT="$(command -v git)"

# -------- §11 AC-B pin-age-sha : pin to SHA $A, origin/dev exactly 3 ahead --------
echo "== 11. AC-B: pin to a SHA; origin/dev is exactly 3 ahead =="
# origin/dev is currently 1 (base+advance) ahead of A; push two empty commits to make it 3.
git -C "$T/seed" commit --quiet --allow-empty -m "ac-b-1"
git -C "$T/seed" commit --quiet --allow-empty -m "ac-b-2"
git -C "$T/seed" push --quiet origin dev
checkeq "lab control: origin/dev is exactly 3 ahead of A" "$(git -C "$T/seed" rev-list --count "$A..origin/dev")" "3"
B11=$(AILANG_DRIVER_REF="$A" /bin/bash "$DRV" 2>&1)
check "sha pin reports pinned"                   "$B11" "STATUS=pinned"
checkline "sha pin has exact line AGE=3"          "$B11" "AGE=3"
checkline "sha pin still has exact line DRIFT=0"  "$B11" "DRIFT=0"
DEV11=$(git -C "$T/seed" rev-parse "dev^{commit}")
check "sha pin note names age 3 and the baseline SHA" "$B11" "pinned target 3 behind origin/dev (baseline $DEV11)"

# -------- §12 AC-C2 pin-age-export : pin to SHA $A, origin/dev exactly 9 ahead --------
echo "== 12. AC-C2: real re-exec exports AILANG_DRIVER_AGE=9 to the driver =="
for i in 3 4 5 6 7 8; do git -C "$T/seed" commit --quiet --allow-empty -m "ac-c2-$i"; done
git -C "$T/seed" push --quiet origin dev
checkeq "lab control: origin/dev is exactly 9 ahead of A" "$(git -C "$T/seed" rev-list --count "$A..origin/dev")" "9"
unset AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA
B12=$(AILANG_DRIVER_REF="$A" /bin/bash "$DRV" 2>&1)
DEV12=$(git -C "$T/seed" rev-parse "dev^{commit}")
checkline "real exec exports AILANG_DRIVER_AGE=9 to the driver" "$B12" "ENV_AGE=9"
checkline "pinned pass reads back exact line AGE=9"            "$B12" "AGE=9"
checkline "exported baseline equals origin/dev's full commit"   "$B12" "ENV_AGE_BASE_SHA=$DEV12"

# -------- §13 AC-D pin-age-old-helper --------
echo "== 13. AC-D: old helper (lacks PIN_AGE) gets a pre-handoff compat warning =="
mkdir -p "$T/oldbuild"
python3 - "$SRC_HELPER" "$T/oldbuild/pin-root.sh" "$T/oldbuild/fake-driver.sh" "$T/clone/tools/launchd/fake-driver.sh" <<'OLDPY' > "$T/oldbuild/report"
import sys, re
helper_src, hdst, ddst, dsrc = sys.argv[1:5]
h = open(helper_src).read()
out = []
def note(name, n):
    out.append('%s=%d' % (name, n))
def cut(name, old):
    n = h.count(old); h2 = h.replace(old, '', 1); out.append('%s=%d' % (name, n)); return h2
# t4clear: the T4 comment + unset line. Run BEFORE the export-name cut, which would otherwise
# steal the AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA names off this unset line.
m4 = re.search(r'  # Clear inherited age readings on an unpinned invocation:.*?unset AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA\n', h, re.S)
note('t4clear', 1 if m4 else 0)
if m4: h = h[:m4.start()] + h[m4.end():]
# t6block: the whole T6 computation (ends on the combined AILANG_DRIVER_AGE/AGE_BASE_SHA assignment).
m = re.search(r'\n  # PIN_AGE .*\n.*?AILANG_DRIVER_AGE_BASE_SHA="\$origin_dev_sha"\n', h, re.S)
note('t6block', 1 if m else 0)
if m: h = h[:m.start()] + h[m.end():]
# T5 refresh-hop handshake comment + unset
m5 = re.search(r'    # Capability handshake for PIN_AGE:.*?unset PIN_AGE_SUPPORTED\n    PIN_AGE="\?"\n', h, re.S)
note('t5handshake', 1 if m5 else 0)
if m5: h = h[:m5.start()] + h[m5.end():]
# T5 warning line
m6 = re.search(r'    if \[ "\${PIN_AGE_SUPPORTED:-}" != "1" \]; then printf.*?lacks PIN_AGE; notice suppressed" >&2; fi\n', h)
note('t5warn', 1 if m6 else 0)
if m6: h = h[:m6.start()] + h[m6.end():]
# header rows
hdr = '#         PIN_AGE     commits on fetched origin/dev NOT reachable from the pinned target ("?" if unknown);\n#                     crosses the re-exec via AILANG_DRIVER_AGE\n#         PIN_AGE_BASE_SHA  full origin/dev SHA captured after the gate refresh that the age was\n#                     measured against ("?" if unknown); crosses the re-exec via AILANG_DRIVER_AGE_BASE_SHA\n'
note('hdr', h.count(hdr)); h = h.replace(hdr, '')
# init block
note('init', h.count('PIN_AGE="?"\nPIN_AGE_BASE_SHA="?"\nPIN_AGE_SUPPORTED=1\n')); h = h.replace('PIN_AGE="?"\nPIN_AGE_BASE_SHA="?"\nPIN_AGE_SUPPORTED=1\n', '')
# readback block
note('readback', h.count('    PIN_AGE="${AILANG_DRIVER_AGE:-?}"\n    PIN_AGE_BASE_SHA="${AILANG_DRIVER_AGE_BASE_SHA:-?}"\n')); h = h.replace('    PIN_AGE="${AILANG_DRIVER_AGE:-?}"\n    PIN_AGE_BASE_SHA="${AILANG_DRIVER_AGE_BASE_SHA:-?}"\n', '')
# note clause
note('note_clause', h.count('; pinned target ${PIN_AGE} behind origin/dev (baseline ${PIN_AGE_BASE_SHA})')); h = h.replace('; pinned target ${PIN_AGE} behind origin/dev (baseline ${PIN_AGE_BASE_SHA})', '')
# export names (single replacement removes them from the export line)
note('export', h.count(' AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA')); h = h.replace(' AILANG_DRIVER_AGE AILANG_DRIVER_AGE_BASE_SHA', '')
open(hdst, 'w').write(h)
d = open(dsrc).read()
for i, l in enumerate(['echo "AGE=$PIN_AGE"\n','echo "AGE_BASE_SHA=$PIN_AGE_BASE_SHA"\n','echo "ENV_AGE=${AILANG_DRIVER_AGE:-unset}"\n','echo "ENV_AGE_BASE_SHA=${AILANG_DRIVER_AGE_BASE_SHA:-unset}"\n','echo "ENV_REF=${AILANG_DRIVER_REF:-unset}"\n']):
    out.append('drv%d=%d' % (i, d.count(l))); d = d.replace(l, '', 1)
open(ddst, 'w').write(d)
sys.stdout.write('\n'.join(out) + '\n')
OLDPY
# Every checked transformation must have matched >=1 line (fixture-rot guard, never vacuous green).
OK=1
while read -r kv; do val=${kv#*=}; if [ "$val" -lt 1 ]; then OK=0; echo "fixture error: old-helper substitution '$kv' matched 0 lines" >&2; fi; done < "$T/oldbuild/report"
[ "$OK" -eq 1 ] || exit 1
if ! /bin/bash -n "$T/oldbuild/pin-root.sh"; then echo "fixture error: old helper does not parse" >&2; exit 1; fi
if /bin/bash -n "$T/oldbuild/fake-driver.sh"; then :; else echo "fixture error: old fake-driver does not parse" >&2; exit 1; fi
[ "$(grep -c 'PIN_AGE' "$T/oldbuild/pin-root.sh")" -eq 0 ] || { echo "fixture error: old helper still mentions PIN_AGE" >&2; exit 1; }
[ "$(grep -c 'AILANG_DRIVER_AGE' "$T/oldbuild/pin-root.sh")" -eq 0 ] || { echo "fixture error: old helper still mentions AILANG_DRIVER_AGE" >&2; exit 1; }
[ "$(grep -c 'rev-list --count "HEAD..$ref"' "$T/oldbuild/pin-root.sh")" -eq 1 ] || { echo "fixture error: old helper lost the drift line" >&2; exit 1; }
# Commit the old helper + old fake-driver on a branch from dev, push to the lab origin.
git -C "$T/seed" checkout --quiet -b oldhelper
cp "$T/oldbuild/pin-root.sh" "$T/seed/tools/launchd/lib/pin-root.sh"
cp "$T/oldbuild/fake-driver.sh" "$T/seed/tools/launchd/fake-driver.sh"
git -C "$T/seed" add -A >/dev/null; git -C "$T/seed" commit --quiet -m "oldhelper fixture"
git -C "$T/seed" push --quiet origin oldhelper
git -C "$T/seed" checkout --quiet dev
AD1=$(AILANG_DRIVER_REF=origin/oldhelper /bin/bash "$DRV" 2>&1)
check "old-helper pin still reports pinned"            "$AD1" "STATUS=pinned"
check "pre-handoff compatibility warning fires"        "$AD1" "lacks PIN_AGE"
checkno "no fabricated AGE=0 under an old helper"       "$AD1" "AGE="
AD2=$(AILANG_DRIVER_AGE=99 AILANG_DRIVER_AGE_BASE_SHA=bogus AILANG_DRIVER_REF=origin/oldhelper /bin/bash "$DRV" 2>&1)
checkno "inherited bogus AGE is not carried into the target" "$AD2" "AGE="
# Consumer witness: the REAL new-consumer age decision (mission-control.sh extraction) logs unknown
# for a genuinely unset PIN_AGE under set -u, rc=0 (kills MUT-F's consumer half).
awk '/^# --- DRIVER PIN AGE DECISION START ---/,/^# --- DRIVER PIN AGE DECISION END ---/' "$REPO_ROOT/tools/launchd/mission-control.sh" > "$T/agedec.sh"
if [ ! -s "$T/agedec.sh" ]; then echo "FATAL: age decision extraction produced nothing" >&2; exit 1; fi
rm -rf "$T/agedec"; mkdir -p "$T/agedec"
ADEC=$(/bin/bash -c '
  set -uo pipefail
  log(){ printf "LOG:%s\n" "$*"; }
  PIN_STATUS=pinned; PIN_AGE_FILE="$1/pin-age"; AILANG_DRIVER_AGE_WARN=25
  _pin_age_degraded=""
  . "$2"
  echo "DECISION_RC:$?"
  [ -n "$_pin_age_degraded" ] && echo "DEGRADED:$_pin_age_degraded"
' _ "$T/agedec" "$T/agedec.sh" 2>&1)
case "$ADEC" in *"DECISION_RC:0"*"LOG:driver pin age: unknown"*) ok "extracted new-consumer age decision logs unknown for unset PIN_AGE";; *"LOG:driver pin age: unknown"*"DECISION_RC:0"*) ok "extracted new-consumer age decision logs unknown for unset PIN_AGE";; *) bad "extracted new-consumer age decision logs unknown for unset PIN_AGE" "$ADEC";; esac

# -------- §14 AC-N age-measure-fails (two PATH-shim sub-arms) --------
echo "== 14. AC-N: failed age rev-list / failed baseline resolution stay loud, never zero =="
mkdir -p "$T/shim1"
cat > "$T/shim1/git" <<EOF
#!/usr/bin/env bash
REAL_GIT="$REAL_GIT"
prev=""; range=""
for a in "\$@"; do [ "\$prev" = "--count" ] && range="\$a"; prev="\$a"; done
# Fail only rev-list --count whose range does NOT begin HEAD.. (the age shape; drift is the control).
if [ -n "\$range" ]; then case "\$range" in HEAD\.\.*) ;; *) echo "ACN1 fail" >&2; exit 1;; esac; fi
exec "\$REAL_GIT" "\$@"
EOF
chmod +x "$T/shim1/git"
N1=$(PATH="$T/shim1:$PATH" AILANG_DRIVER_REF="$A" /bin/bash "$DRV" 2>&1)
checkline "failed age rev-list yields AGE=?"        "$N1" "AGE=?"
checkno "failed age rev-list fabricates no zero"    "$N1" "AGE=0"
check "pin still reports pinned when age fails"     "$N1" "STATUS=pinned"
checkline "drift control still succeeds during age failure" "$N1" "DRIFT=0"
mkdir -p "$T/shim2"
cat > "$T/shim2/git" <<EOF
#!/usr/bin/env bash
REAL_GIT="$REAL_GIT"
for a in "\$@"; do case "\$a" in *"origin/dev^{commit}"*) echo "ACN2 fail" >&2; exit 1;; esac; done
exec "\$REAL_GIT" "\$@"
EOF
chmod +x "$T/shim2/git"
N2=$(PATH="$T/shim2:$PATH" AILANG_DRIVER_REF="$A" /bin/bash "$DRV" 2>&1)
checkline "failed baseline resolution keeps AGE and baseline ?" "$N2" "AGE=?"
checkline "failed baseline resolution keeps AGE and baseline ? (2)" "$N2" "AGE_BASE_SHA=?"
check "failed baseline resolution is loud on stderr" "$N2" "driver pin age: unknown (?); origin/dev baseline resolution failed"

# -------- §15 AC-P pin-age-baseline-moves (moving-baseline shim) --------
echo "== 15. AC-P: origin/dev moves under the fire; age stays anchored to captured baseline B =="
rm -f "$T/acp_b" "$T/acp_count"
mkdir -p "$T/acpdir" "$T/shimP"
cat > "$T/shimP/git" <<EOF
#!/usr/bin/env bash
REAL_GIT="$REAL_GIT"
intercept=0
for a in "\$@"; do case "\$a" in *"origin/dev^{commit}"*) intercept=1;; esac; done
if [ "\$intercept" = 1 ]; then
  B="\$(\$REAL_GIT -C "$T/seed" rev-parse --verify --quiet 'origin/dev^{commit}' 2>/dev/null)"
  printf '%s\n' "\$B" > "$T/acp_b"
  printf 'x\n' >> "$T/acp_count"
  \$REAL_GIT -C "$T/seed" commit --quiet --allow-empty -m "acp-move" 2>/dev/null
  \$REAL_GIT -C "$T/seed" push --quiet origin dev 2>/dev/null
  \$REAL_GIT -C "$T/clone" fetch --quiet origin 2>/dev/null
  printf '%s\n' "\$B"
  exit 0
fi
exec "\$REAL_GIT" "\$@"
EOF
chmod +x "$T/shimP/git"
P1=$(PATH="$T/shimP:$PATH" AILANG_DRIVER_REF="$A" /bin/bash "$DRV" 2>&1)
B15=$(cat "$T/acp_b" 2>/dev/null)
CNT15=$(wc -l < "$T/acp_count" 2>/dev/null | tr -d ' ')
checkeq "baseline shim fired exactly once" "$CNT15" "1"
MOVED15=$(git -C "$T/seed" rev-parse origin/dev)
if [ "$MOVED15" != "$B15" ]; then ok "moved origin/dev differs from captured baseline B"; else bad "moved origin/dev differs from captured baseline B" "$MOVED15"; fi
EXP15=$(git -C "$T/seed" rev-list --count "$A..$B15")
checkline "reported age is B's count, not the moved ref's" "$P1" "AGE=$EXP15"
checkline "exported baseline SHA equals captured B"       "$P1" "ENV_AGE_BASE_SHA=$B15"
check "note baseline equals captured B"                 "$P1" "baseline $B15"

# -------- §16 AC-Q pin-age-nondefault-ref (lab half) --------
echo "== 16. AC-Q: origin/feature ref is 26 behind origin/dev; AGE=26 =="
git -C "$T/seed" push --quiet origin "$A:refs/heads/feature"
CNT16=$(git -C "$T/seed" rev-list --count "$A..origin/dev")
ADD16=$((26 - CNT16))
i=0
while [ "$i" -lt "$ADD16" ]; do git -C "$T/seed" commit --quiet --allow-empty -m "ac-q-$i"; i=$((i+1)); done
git -C "$T/seed" push --quiet origin dev
checkeq "lab control: origin/feature is exactly 26 behind origin/dev" "$(git -C "$T/seed" rev-list --count "origin/feature..origin/dev")" "26"
Q16=$(AILANG_DRIVER_REF=origin/feature /bin/bash "$DRV" 2>&1)
check "feature-ref pin reports pinned"         "$Q16" "STATUS=pinned"
checkline "feature-ref pin has exact line AGE=26" "$Q16" "AGE=26"
checkline "driver sees exported AILANG_DRIVER_REF=origin/feature" "$Q16" "ENV_REF=origin/feature"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
