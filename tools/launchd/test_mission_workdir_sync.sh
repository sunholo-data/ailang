#!/bin/bash
# mc_workdir_sync: fetch + ff-only the mission's standalone MISSION_WORKDIR clone. Bash 3.2.
# Why: 2026-10-15 the triage controller read a 14-commit-stale charter from its clone and
# enforced a rule the attended session had already amended (lib/skill-sync.sh).
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/lib/pin-root.sh"
. "$HERE/lib/skill-sync.sh"
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t
g() { git -c core.autocrlf=false -c init.defaultBranch=dev "$@" >/dev/null 2>&1; }
fail() { echo "FAIL $1"; exit 1; }

# Fixture: a bare origin, a writer clone that pushes, and the mission workdir clone.
g init --bare "$T/origin.git"
g clone "$T/origin.git" "$T/writer"; (cd "$T/writer" && g checkout -b dev && echo a > a && g add a && g commit -m a && g push origin dev)
g clone -b dev "$T/origin.git" "$T/work"
MC_DRIVER_ROOT="$T/writer"   # anything that is not the workdir (self-target guard)
advance() { (cd "$T/writer" && echo "$1" > "$1" && g add "$1" && g commit -m "$1" && g push origin dev); }
head_of() { git -C "$1" rev-parse HEAD; }

# 1. Behind and clean -> fetched and fast-forwarded to origin/dev.
advance b; advance c
mc_workdir_sync "$T/work"
[ "$(head_of "$T/work")" = "$(head_of "$T/writer")" ] || fail "behind clone not synced: $SKILL_SYNC_STATUS $SKILL_SYNC_NOTE"
case "$SKILL_SYNC_STATUS" in synced:2) : ;; *) fail "verdict $SKILL_SYNC_STATUS, want synced:2" ;; esac
echo 'PASS behind clean workdir is fetched and fast-forwarded'

# 2. Already current -> no change.
mc_workdir_sync "$T/work"
[ "$SKILL_SYNC_STATUS" = current ] || fail "current clone verdict $SKILL_SYNC_STATUS"
echo 'PASS current workdir is left alone'

# 3. Local commit (ahead) -> skipped, HEAD untouched, even with origin moved on.
(cd "$T/work" && echo mine > mine && g add mine && g commit -m mine)
mine=$(head_of "$T/work"); advance d
mc_workdir_sync "$T/work"
[ "$(head_of "$T/work")" = "$mine" ] || fail 'ahead clone was moved'
[ "$SKILL_SYNC_STATUS" = skip:ahead ] || fail "ahead verdict $SKILL_SYNC_STATUS"
echo 'PASS workdir with local commits is never moved'

# 4. Fetch failure -> reported, HEAD untouched.
g clone -b dev "$T/origin.git" "$T/work2"; before=$(head_of "$T/work2")
git -C "$T/work2" remote set-url origin "$T/nonexistent.git"
advance e
mc_workdir_sync "$T/work2"
[ "$SKILL_SYNC_STATUS" = error:fetch-failed ] || fail "fetch failure verdict $SKILL_SYNC_STATUS"
[ "$(head_of "$T/work2")" = "$before" ] || fail 'fetch failure moved HEAD'
echo 'PASS a failed fetch is reported and changes nothing'

# 5. Wiring: the driver calls it only behind the opt-in, and triage opts in.
grep -q 'if \[ "${MISSION_WORKDIR_SYNC:-0}" = 1 \] && type mc_workdir_sync' "$HERE/mission-control.sh" || fail 'driver does not gate mc_workdir_sync on MISSION_WORKDIR_SYNC'
grep -q '^  mc_workdir_sync "\$REPO"$' "$HERE/mission-control.sh" || fail 'driver does not sync $REPO'
grep -q '^MISSION_WORKDIR_SYNC="${MISSION_WORKDIR_SYNC:-1}"$' "$HERE/mission-env/mission-triage.env" || fail 'triage env does not opt in'
echo 'PASS driver gates the sync on MISSION_WORKDIR_SYNC and triage opts in'
