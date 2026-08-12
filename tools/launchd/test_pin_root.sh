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

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  FAIL: $1"; echo "        got: $2"; }
check(){ # name haystack needle
  case "$2" in *"$3"*) ok "$1";; *) bad "$1" "$(printf '%s' "$2" | tr '\n' '|' | tail -c 300)";; esac
}
checkno(){ case "$2" in *"$3"*) bad "$1" "$(printf '%s' "$2" | tr '\n' '|' | tail -c 300)";; *) ok "$1";; esac; }

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
if [ -f "$REPO/tools/launchd/lib/pin-root.sh" ]; then
  . "$REPO/tools/launchd/lib/pin-root.sh"
  pin_root_to_committed_ref "$@"
else
  PIN_STATUS="STALE"; PIN_NOTE="helper absent"
fi
echo "STATUS=$PIN_STATUS"
echo "DRIFT=$PIN_DRIFT"
echo "NOTE=$PIN_NOTE"
echo "REPO=$REPO"
echo "MARKER=$(cat "$REPO/MARKER" 2>/dev/null)"
echo "ARGS=$*"
DRIVER
chmod +x tools/launchd/fake-driver.sh
echo "STALE-CONTENT" > MARKER
git add -A >/dev/null; git commit --quiet -m "base"
git push --quiet origin dev 2>/dev/null

# clone that will go stale
git clone --quiet --branch dev "$T/origin.git" "$T/clone" 2>/dev/null

# advance origin ONE commit: the clone is now genuinely behind
echo "FRESH-CONTENT" > MARKER
git commit --quiet -am "advance"
git push --quiet origin dev 2>/dev/null
cd "$T"

DRV="$T/clone/tools/launchd/fake-driver.sh"
export AILANG_DRIVER_PIN_DIR="$T/pinwt"

echo "== 1. happy path: stale clone re-execs into committed origin/dev =="
OUT=$(/bin/bash "$DRV" alpha beta 2>&1)
check "status is pinned"                 "$OUT" "STATUS=pinned"
check "drift measured as 1"              "$OUT" "DRIFT=1"
check "ROOT moved to the pin worktree"   "$OUT" "REPO=$T/pinwt"
check "reads FRESH content, not stale"   "$OUT" "MARKER=FRESH-CONTENT"
checkno "stale content is NOT read"      "$OUT" "MARKER=STALE-CONTENT"
check "args survive the re-exec"         "$OUT" "ARGS=alpha beta"

echo "== 2. control: the clone really was stale (instrument check) =="
CTL=$(AILANG_DRIVER_PIN=0 /bin/bash "$DRV" 2>&1)
check "opt-out reports disabled"         "$CTL" "STATUS=disabled"
check "opt-out reads STALE content"      "$CTL" "MARKER=STALE-CONTENT"
check "opt-out root stays in the clone"  "$CTL" "REPO=$T/clone"

echo "== 3. no recursion: second pass returns without re-exec =="
REC=$(AILANG_DRIVER_PINNED=deadbee AILANG_DRIVER_DRIFT=7 /bin/bash "$DRV" 2>&1)
check "already-pinned short-circuits"    "$REC" "STATUS=pinned"
check "carries drift across the exec"    "$REC" "DRIFT=7"

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

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
