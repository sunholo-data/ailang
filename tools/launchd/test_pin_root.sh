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
printf '{"projects":{}}' > "$HOME/.claude.json"
UO=$(/bin/bash "$DRV" 2>&1)
check "refuses to pin"                   "$UO" "STATUS=STALE"
check "names the onboarding cause"       "$UO" "onboarded in Claude Code"
check "gives the exact human fix"        "$UO" "&& claude"
check "fire still runs, unpinned"        "$UO" "MARKER=STALE-CONTENT"
checkno "never reports pinned"           "$UO" "STATUS=pinned"

echo "== 8. the gate reads the MEASURED flag, not the one the error text names =="
# 76ee4056c: ailang-world has trust=false / onboarded=true and WORKS. So trust alone must not
# satisfy the gate, or we would be right by accident and wrong in the reason.
printf '{"projects":{"%s":{"hasTrustDialogAccepted":true}}}' "$T/pinwt" > "$HOME/.claude.json"
TR=$(/bin/bash "$DRV" 2>&1)
check "trust flag alone does NOT satisfy" "$TR" "STATUS=STALE"
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "$T/pinwt" > "$HOME/.claude.json"
OB=$(/bin/bash "$DRV" 2>&1)
check "onboarding flag DOES satisfy"      "$OB" "STATUS=pinned"

echo "== 8b. SOURCE-clone onboarding satisfies it — the case the first cut got WRONG =="
# Measured 2026-08-12: `claude -p` runs fine from ~/.ailang-driver-pin/v1 while ~/.claude.json
# has NO entry for it — a worktree inherits its source clone's trust. The first predicate checked
# only the worktree's own entry, so it would have refused a demonstrably working target on every
# fire and left the pin permanently off. This is the production shape: worktree absent, source ok.
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "$T/clone" > "$HOME/.claude.json"
SC=$(/bin/bash "$DRV" 2>&1)
check "source onboarding is enough"       "$SC" "STATUS=pinned"
check "and it really pinned"              "$SC" "MARKER=FRESH-CONTENT"

echo "== 8c. NEITHER onboarded => still refuse (the motoko shape) =="
# The gate must not have been widened into a no-op: a fresh clone with nothing onboarded anywhere
# is exactly what cost motoko iteration 1, and it must still be refused.
printf '{"projects":{"%s":{"hasCompletedProjectOnboarding":true}}}' "/some/unrelated/path" > "$HOME/.claude.json"
NN=$(/bin/bash "$DRV" 2>&1)
check "neither path onboarded => STALE"   "$NN" "STATUS=STALE"
check "message names BOTH paths"          "$NN" "nor its source clone"

echo "== 9. undeterminable (no jq) fails SAFE, not open =="
mkdir -p "$T/nojq"
for b in git mktemp date basename dirname cat rm sleep kill grep tr tail printf mkdir; do
  p=$(command -v $b 2>/dev/null); [ -n "$p" ] && ln -sf "$p" "$T/nojq/$b"
done
NJ=$(PATH="$T/nojq" /bin/bash "$DRV" 2>&1)
check "no jq => STALE, not pinned"       "$NJ" "STATUS=STALE"
check "says it could not verify"         "$NJ" "cannot verify Claude Code onboarding"

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
