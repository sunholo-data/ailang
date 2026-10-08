#!/bin/bash
# Synthetic Git safety and isolated single-mutant controls. Bash 3.2 only.
set -Eeuo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
ARMS='happy current not-dev ahead rebase merge am cherry revert sequencer bisect-log bisect-start modified staged outside untracked dirty-source dirty-destination incoming-source incoming-destination whitespace duplicate directory locked late-lock report disabled absent symlink non-git self missing-ref timeout missing-bounded changed invalid survival broken standalone linked-lock late-ref'
# M2 coverage is unconditional: removing the production seam must turn the suite red.
ARMS="$ARMS driver-disabled driver-overlap driver-dry driver-field driver-real driver-missing"
run_arm() {
  # An independent watchdog also bounds direct-Git timeout mutants.
  python3 - "$0" "$1" "$2" <<'PY'
import subprocess, sys
try:
    r = subprocess.run(['/bin/bash', sys.argv[1], '--arm', sys.argv[2], sys.argv[3]], timeout=45, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    print('\n'.join(x for x in r.stdout.splitlines() if not x.startswith('FIXTURE ')))
    if ('FIXTURE ' + sys.argv[2] + ' ready') not in r.stdout and r.returncode != 98:
        print('FAIL: fixture did not reach ready state', file=sys.stderr)
        sys.exit(98)
    if r.returncode == 0 and ('PASS ' + sys.argv[2] + ' (') not in r.stdout:
        print('FAIL: missing arm completion witness', file=sys.stderr)
        sys.exit(1)
    sys.exit(r.returncode)
except subprocess.TimeoutExpired:
    print('FAIL: independent arm watchdog expired', file=sys.stderr)
    sys.exit(99)
PY
}
case "${1:-}" in
  '')
    for arm in $ARMS; do run_arm "$arm" pristine; done
    echo "All pristine arms passed"; exit 0 ;;
  --mutations)
    for arm in $ARMS; do
      rc=0; run_arm "$arm" mutant || rc=$?
      if [ "$rc" = 0 ] || [ "$rc" -eq 98 ] || [ "$rc" -eq 99 ]; then
        echo "FAIL: mutant $arm survived or fixture/watchdog failed (rc=$rc)" >&2; exit 1
      fi
      run_arm "$arm" pristine
      echo "MUTATION $arm -> killed=yes (mutant rc=$rc; control rc=0)"
    done
    exit 0 ;;
  --arm) ARM=${2:?}; VARIANT=${3:?} ;;
  *) echo "unknown selector: $1" >&2; exit 2 ;;
esac
case " $ARMS " in *" $ARM "*) ;; *) echo "unknown arm: $ARM" >&2; exit 2;; esac
case "$VARIANT" in pristine|mutant) ;; *) exit 2;; esac
LAB=$(mktemp -d "${TMPDIR:-/tmp}/skill-sync-test.XXXXXX")
trap 'rm -rf "$LAB"' EXIT
TESTING=0
trap 'if [ "$TESTING" = 0 ]; then echo "fixture setup failed: $ARM" >&2; exit 98; fi' ERR
LAB=$(cd "$LAB" && pwd -P)
export HOME="$LAB/home" TMPDIR="$LAB/tmp"
mkdir -p "$HOME" "$TMPDIR" "$LAB/bin" "$LAB/driver/tools/launchd/lib"
# Drop inherited Git control variables before any fixture Git command.
for key in $(env | sed -n 's/^\(GIT_[A-Za-z0-9_]*\)=.*/\1/p'); do unset "$key"; done
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=skilllab GIT_AUTHOR_EMAIL=skilllab@invalid
export GIT_COMMITTER_NAME=skilllab GIT_COMMITTER_EMAIL=skilllab@invalid
export LC_ALL=C
REAL_GIT=$(command -v git); export REAL_GIT LAB ARM
cp "$HERE/lib/skill-sync.sh" "$LAB/driver/tools/launchd/lib/skill-sync.sh"
cp "$HERE/mission-control.sh" "$LAB/driver/mission-control.sh"
HELPER="$LAB/driver/tools/launchd/lib/skill-sync.sh"
DRIVER="$LAB/driver/mission-control.sh"
if [ "$VARIANT" = mutant ]; then
  python3 - "$HELPER" "$DRIVER" "$ARM" <<'PY'
import sys
helper, driver, arm = sys.argv[1:]
p = helper
changes = {
'happy': ('_ss_git merge --ff-only origin/dev || rc=$?', ': # merge bypassed'),
'current': ('if [ "$ss_behind" = 0 ]; then', 'if false; then'),
'not-dev': ('if [ "$ss_branch" != dev ]; then', 'if false; then'),
'ahead': ('if [ "$ss_ahead" != 0 ]; then', 'if false; then'),
'modified': ('code=${record:0:2}; path=${record:3}', 'code=${record:0:2}; path=${record:3}; [ "$code" != " M" ] || continue'),
'staged': ('code=${record:0:2}; path=${record:3}', 'code=${record:0:2}; path=${record:3}; [ "$code" != "M " ] || continue'),
'outside': ('if [ "$d" = "$incoming" ] || [[ "$d" = "$incoming/"* || "$incoming" = "$d/"* ]]; then hit=1; fi', 'hit=1'),
'untracked': ('code=${record:0:2}; path=${record:3}', 'code=${record:0:2}; path=${record:3}; [ "$code" != "??" ] || continue'),
'dirty-source': ('dirty_paths+=("$other") # dirty source', ': # dirty source removed'),
'dirty-destination': ('dirty_paths+=("$path") # dirty destination (also ordinary paths)', 'case "$code" in *R*|*C*) ;; *) dirty_paths+=("$path");; esac'),
'incoming-source': ('incoming_paths+=("$path") # incoming first side', 'case "$code" in R*|C*) ;; *) incoming_paths+=("$path");; esac'),
'incoming-destination': ('incoming_paths+=("$other") # incoming second side', ': # incoming destination removed'),
'whitespace': ("while IFS= read -r -d '' record; do", 'while IFS= read -r record; do'),
'duplicate': ('[ "$prior" = 0 ] || continue', ': # no deduplication'),
'directory': (' || [[ "$d" = "$incoming/"* || "$incoming" = "$d/"* ]]', ''),
'locked': ('if [ -e "$ss_lock" ]; then', 'if false; then'),
'late-lock': ("_ss_verdict skip:index-locked 'Git refused held index lock; no retry'", "_ss_verdict error:merge-failed 'mutated lock mapping'"),
'report': ('if [ "$mode" = report ]; then', 'if false; then'),
'disabled': ('if [ "${AILANG_SKILL_SYNC:-1}" = 0 ]; then', 'if false; then'),
'absent': ('if [ ! -L "$link" ]; then', 'if false; then'),
'symlink': ('ss_checkout=$(readlink -f "$link" 2>/dev/null) || ss_checkout=\'\'', 'ss_checkout="$HOME/dev/sunholo-data/ailang"'),
'non-git': ("_ss_verdict skip:not-worktree 'target is not a worktree'; return 1;", "_ss_verdict current 'mutated non-worktree fallback'; return 1;"),
'self': ('if [ "$ss_checkout" = "$root" ]; then', 'if false; then'),
'missing-ref': ("_ss_verdict error:ref-unavailable 'origin/dev is unavailable'", "_ss_verdict current 'mutated missing ref fallback'"),
'missing-bounded': ('if ! type _pin_bounded >/dev/null 2>&1; then', 'if false; then'),
'changed': ('if [ "$before_head" != "$ss_head" ] || [ "$before_tip" != "$ss_tip" ] || [ "$before_branch" != "$ss_branch" ]; then', 'if false; then'),
'invalid': ('if [ "$mode" != apply ] && [ "$mode" != report ]; then', 'if false; then'),
 'survival': ('  return 0\n}\n', '  return 1\n}\n'),
'broken': ('if [ -z "$ss_checkout" ] || [ ! -d "$ss_checkout" ]; then', 'if false; then'),
'standalone': ('if [ "$ss_checkout" = "$root" ]; then', 'if false; then'),
'linked-lock': ('if [ -e "$ss_lock" ]; then', 'if false; then'),
'late-ref': ("_ss_verdict skip:git-refused 'Git refused concurrent checkout/ref state; no retry'", "_ss_verdict error:merge-failed 'mutated refusal mapping'"),
}
markers = dict(rebase='rebase-merge', merge='MERGE_HEAD', am='rebase-apply', cherry='CHERRY_PICK_HEAD', revert='REVERT_HEAD', sequencer='sequencer', **{'bisect-log':'BISECT_LOG', 'bisect-start':'BISECT_START'})
if arm in markers:
    marker = markers[arm]
    changes[arm] = ('for marker in rebase-merge rebase-apply MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD sequencer BISECT_LOG BISECT_START; do', 'for marker in ' + ' '.join(x for x in markers.values() if x != marker) + '; do')
if arm == 'timeout':
    changes[arm] = ('_pin_bounded 5 /bin/bash -c', '/bin/bash -c')
if arm.startswith('driver-'):
    p = driver
    seam = '''SKILL_SYNC_STATUS="error:helper-missing"; SKILL_SYNC_NOTE="skill-sync helper absent"
if [ -f "$MC_DRIVER_ROOT/tools/launchd/lib/skill-sync.sh" ]; then
  . "$MC_DRIVER_ROOT/tools/launchd/lib/skill-sync.sh"
fi
'''
    apply = '''if type mc_skill_sync >/dev/null 2>&1; then mc_skill_sync apply; fi
log "skill-sync=$SKILL_SYNC_STATUS $SKILL_SYNC_NOTE"
'''
    changes.update({
      'driver-disabled': ('if [ -f "$KILL_SWITCH" ]; then', seam + 'if type mc_skill_sync >/dev/null 2>&1; then mc_skill_sync apply; fi\nif [ -f "$KILL_SWITCH" ]; then'),
      'driver-overlap': ('if [ -f "$PIDFILE" ]; then', seam + 'if type mc_skill_sync >/dev/null 2>&1; then mc_skill_sync apply; fi\nif [ -f "$PIDFILE" ]; then'),
      'driver-dry': ('then mc_skill_sync report; fi', 'then mc_skill_sync apply; fi'),
      'driver-field': (' | skill-sync=$SKILL_SYNC_STATUS"; exit 0', '"; exit 0'),
      'driver-real': (apply, ''),
      'driver-missing': ('SKILL_SYNC_STATUS="error:helper-missing"', 'SKILL_SYNC_STATUS="current"'),
    })
s = open(p).read()
old, new = changes[arm]
# index.lock has two checks; mutate only the preflight occurrence.
if arm == 'survival':
    pos = s.index('mc_skill_sync()')
    old = s[pos:]
    new = old.replace('  return 0\n}', '  return 1\n}')
count = s.count(old)
if old == new or count == 0 or (count != 1 and arm not in ('locked', 'linked-lock')):
    print('ZERO/AMBIGUOUS SUBSTITUTION', arm, count, file=sys.stderr); sys.exit(98)
s = s.replace(old, new, 1 if arm in ('locked', 'linked-lock') else count)
if arm == 'driver-real':
    needle = '# 3c. MEMORY GATE'
    if s.count(needle) != 1: sys.exit(98)
    s = s.replace(needle, apply + '\n' + needle)
open(p, 'w').write(s)
PY
fi
fail() { echo "FAIL $ARM: $*" >&2; exit 1; }
eq() { [ "$1" = "$2" ] || fail "expected [$2], got [$1]"; }
# Set up each arm independently, including driver-only arms: fixture setup cannot be skipped.
g() { "$REAL_GIT" -c core.hooksPath=/dev/null "$@"; }
g init -q --bare "$LAB/origin.git"
g clone -q "$LAB/origin.git" "$LAB/push" 2>/dev/null
g -C "$LAB/push" checkout -q -b dev
printf 'base\n' > "$LAB/push/tracked"
printf 'outside\n' > "$LAB/push/outside"
printf 'whitespace\n' > "$LAB/push/space
name"
g -C "$LAB/push" add -A
g -C "$LAB/push" commit -qm base
g -C "$LAB/push" push -q origin dev
g clone -q -b dev "$LAB/origin.git" "$LAB/target"
g -C "$LAB/push" config core.hooksPath /dev/null
g -C "$LAB/target" config core.hooksPath /dev/null
BASE=$(g -C "$LAB/target" rev-parse HEAD)
case "$ARM" in
 incoming-source|incoming-destination) g -C "$LAB/push" mv tracked incoming-renamed ;;
 dirty-destination|untracked|directory) printf 'incoming\n' > "$LAB/push/added" ;;
 whitespace) printf 'incoming\n' >> "$LAB/push/space
name" ;;
 *) printf 'incoming\n' >> "$LAB/push/tracked" ;;
esac
g -C "$LAB/push" add -A; g -C "$LAB/push" commit -qm incoming-one
printf 'two\n' > "$LAB/push/second"
g -C "$LAB/push" add -A; g -C "$LAB/push" commit -qm incoming-two
g -C "$LAB/push" push -q origin dev
g -C "$LAB/target" fetch -q origin
TIP=$(g -C "$LAB/target" rev-parse origin/dev)
export AILANG_SKILL_SYNC_CHECKOUT="$LAB/target"
export MC_DRIVER_ROOT="$LAB/driver"
EXPECTED=synced:2; MERGES=1; MODE=apply; MARKER=''
case "$ARM" in
 current) g -C "$LAB/target" merge -q --ff-only origin/dev; EXPECTED=current; MERGES=0 ;;
 not-dev) g -C "$LAB/target" checkout -q -b attended; EXPECTED=skip:not-dev; MERGES=0 ;;
 ahead) printf 'local\n' > "$LAB/target/local"; g -C "$LAB/target" add -A; g -C "$LAB/target" commit -qm local; EXPECTED=skip:ahead; MERGES=0 ;;
 rebase|am|sequencer) case "$ARM" in rebase) MARKER=rebase-merge;; am) MARKER=rebase-apply;; *) MARKER=sequencer;; esac; mkdir "$LAB/target/.git/$MARKER"; EXPECTED=skip:operation-in-progress; MERGES=0 ;;
 merge|cherry|revert|bisect-log|bisect-start) case "$ARM" in merge) MARKER=MERGE_HEAD;; cherry) MARKER=CHERRY_PICK_HEAD;; revert) MARKER=REVERT_HEAD;; bisect-log) MARKER=BISECT_LOG;; *) MARKER=BISECT_START;; esac; printf '%s\n' "$BASE" > "$LAB/target/.git/$MARKER"; EXPECTED=skip:operation-in-progress; MERGES=0 ;;
 modified|staged|duplicate) printf 'dirty\n' >> "$LAB/target/tracked"; if [ "$ARM" = staged ]; then g -C "$LAB/target" add tracked; fi; EXPECTED=skip:dirty-range; MERGES=0 ;;
 outside) printf 'dirty outside\n' >> "$LAB/target/outside" ;;
 untracked) printf 'dirty untracked\n' > "$LAB/target/added"; EXPECTED=skip:dirty-range; MERGES=0 ;;
 dirty-source) g -C "$LAB/target" mv tracked dirty-renamed; EXPECTED=skip:dirty-range; MERGES=0 ;;
 dirty-destination) g -C "$LAB/target" mv tracked added; EXPECTED=skip:dirty-range; MERGES=0 ;;
 incoming-source) printf 'dirty\n' >> "$LAB/target/tracked"; EXPECTED=skip:dirty-range; MERGES=0 ;;
 incoming-destination) printf 'dirty\n' > "$LAB/target/incoming-renamed"; EXPECTED=skip:dirty-range; MERGES=0 ;;
 whitespace) printf 'dirty\n' >> "$LAB/target/space
name"; EXPECTED=skip:dirty-range; MERGES=0 ;;
 directory) mkdir "$LAB/target/added"; printf 'blocker\n' > "$LAB/target/added/child"; EXPECTED=skip:dirty-range; MERGES=0 ;;
 locked) printf 'held\n' > "$LAB/target/.git/index.lock"; EXPECTED=skip:index-locked; MERGES=0 ;;
 late-lock) EXPECTED=skip:index-locked ;;
 report) EXPECTED=synced:2; MERGES=0; MODE=report ;;
 disabled) export AILANG_SKILL_SYNC=0; EXPECTED=skip:disabled; MERGES=0 ;;
 absent) unset AILANG_SKILL_SYNC_CHECKOUT; EXPECTED=skip:symlink-absent; MERGES=0 ;;
 symlink) mkdir -p "$HOME/.claude/skills" "$LAB/target/skills/mission-control"; ln -s "$LAB/target/skills/mission-control" "$HOME/.claude/skills/mission-control"; unset AILANG_SKILL_SYNC_CHECKOUT ;;
 non-git) mkdir "$LAB/non-git"; export AILANG_SKILL_SYNC_CHECKOUT="$LAB/non-git"; EXPECTED=skip:not-worktree; MERGES=0 ;;
 self) export MC_DRIVER_ROOT="$LAB/target"; EXPECTED=skip:self-target; MERGES=0 ;;
 missing-ref) g -C "$LAB/target" update-ref -d refs/remotes/origin/dev; EXPECTED=error:ref-unavailable; MERGES=0 ;;
 timeout) EXPECTED=error:git-timeout; MERGES=0 ;;
 missing-bounded) EXPECTED=error:bounded-unavailable; MERGES=0 ;;
 changed) EXPECTED=skip:state-changed; MERGES=0 ;;
 invalid) MODE=bogus; EXPECTED=error:invalid-mode; MERGES=0 ;;
 broken) mkdir -p "$HOME/.claude/skills"; ln -s "$LAB/absent" "$HOME/.claude/skills/mission-control"; unset AILANG_SKILL_SYNC_CHECKOUT; EXPECTED=skip:symlink-broken; MERGES=0 ;;
 standalone) unset MC_DRIVER_ROOT; mkdir "$LAB/target/subdir"; cd "$LAB/target/subdir"; EXPECTED=skip:self-target; MERGES=0 ;;
 linked-lock) g -C "$LAB/target" checkout -q --detach; mv "$LAB/target" "$LAB/main"; g -C "$LAB/main" worktree add -q "$LAB/target" dev; lock=$(g -C "$LAB/target" rev-parse --git-path index.lock); printf 'linked held\n' > "$lock"; EXPECTED=skip:index-locked; MERGES=0 ;;
 late-ref) LATE_TIP=$(g -C "$LAB/target" commit-tree "$TIP^{tree}" -m unrelated); export LATE_TIP; EXPECTED=skip:git-refused ;;
 survival) export AILANG_SKILL_SYNC=0; EXPECTED=skip:disabled; MERGES=0 ;;
esac
# Snapshots compare all target bytes/inventories, including refs, index and Git logs.
snapshot() {
 python3 - "$LAB/target" "$1" <<'PY'
import hashlib, os, sys
root, dest = sys.argv[1:]
roots = [root]
gitfile = os.path.join(root, '.git')
if os.path.isfile(gitfile):
    gitdir = open(gitfile).read().strip().split(': ', 1)[1]
    assert gitdir.startswith(os.path.dirname(root) + '/')
    roots.append(gitdir)
with open(dest, 'w') as out:
 for d, dirs, files in (entry for base in roots for entry in os.walk(base)):
  dirs.sort()
  for n in sorted(dirs + files):
   p = os.path.join(d, n); rel = os.path.relpath(p, root)
   if os.path.islink(p): value = 'L:' + os.readlink(p)
   elif os.path.isdir(p): value = 'D'
   else: value = hashlib.sha256(open(p, 'rb').read()).hexdigest()
   out.write(repr(rel) + ' ' + value + '\n')
PY
}
snapshot "$LAB/before"
TARGET_GIT_DIR=$(g -C "$LAB/target" rev-parse --absolute-git-dir)
cp "$TARGET_GIT_DIR/index" "$LAB/index.before"
cp "$LAB/target/outside" "$LAB/outside.before"
HEAD_BEFORE=$(g -C "$LAB/target" rev-parse HEAD)
# Command-specific spy leaves real Git in charge of the lab repository.
cat > "$LAB/bin/git" <<'SHIM'
#!/bin/bash
set -eu
printf '%s\n' "$*" >> "$LAB/git.calls"
case "$*" in
 *'merge --ff-only origin/dev')
   echo merge >> "$LAB/merge.calls"
   if [ "$ARM" = late-lock ]; then printf 'late held\n' > "$LAB/target/.git/index.lock"; fi
   if [ "$ARM" = late-ref ]; then "$REAL_GIT" -C "$LAB/target" update-ref refs/remotes/origin/dev "$LATE_TIP"; fi
   ;;
 *'rev-parse --show-toplevel') if [ "$ARM" = timeout ]; then /bin/sleep 8; fi ;;
 *"rev-parse --verify HEAD^{commit}")
   if [ "$ARM" = changed ]; then
     if [ -f "$LAB/head.seen" ]; then "$REAL_GIT" -C "$LAB/target" update-ref refs/remotes/origin/dev "$BASE"; else touch "$LAB/head.seen"; fi
   fi ;;
esac
if [ "$ARM" = duplicate ] && [[ "$*" = *'status --porcelain=v1 -z'* ]]; then
 "$REAL_GIT" "$@"; "$REAL_GIT" "$@"; exit 0
fi
exec "$REAL_GIT" "$@"
SHIM
chmod +x "$LAB/bin/git"
export PATH="$LAB/bin:$PATH" BASE
# Accelerate only the primitive's polling sleep, retaining the real primitive/cap.
# Timeout fixture uses /bin/sleep directly, so it remains an actual hanging Git call.
sleep() { /bin/sleep 0.01; }
. "$HERE/lib/pin-root.sh"
PIN_STATUS=witness; PIN_NOTE=witness; PIN_BOUNDED_OUT=witness
. "$HELPER"
if [ "$ARM" = missing-bounded ]; then unset -f _pin_bounded; fi
if [[ "$ARM" = driver-* ]]; then
  # Extract actual guards and the contiguous pidfile/dry-run/apply/stagger seam.
  # The earlier provider/config code is intentionally excluded.
  python3 - "$DRIVER" "$LAB/seam.sh" <<'PY'
import sys
s = open(sys.argv[1]).read()
k = s.index('if [ -f "$KILL_SWITCH" ]; then')
ke = s.index('\nfi', k) + 3
p = s.index('if [ -f "$PIDFILE" ]; then')
b = s.index('BOOT_WINDOW="${MISSION_BOOT_WINDOW:-900}"', p)
e = s.index('\nfi', b) + 3
# Include moved-source mutants immediately before guards as well.
if 'mc_skill_sync apply; fi\nif [ -f "$KILL_SWITCH"' in s:
 k = s.rindex('SKILL_SYNC_STATUS=', 0, k)
if 'mc_skill_sync apply; fi\nif [ -f "$PIDFILE"' in s:
 p = s.rindex('SKILL_SYNC_STATUS=', 0, p)
assert k < ke and p < b < e
open(sys.argv[2], 'w').write(s[k:ke] + '\n' + s[p:e] + '\nlog continuation\n')
PY
  # Sourceable lab spy (overwrites copied helper only, never production).
  cat > "$LAB/driver/tools/launchd/lib/skill-sync.sh" <<'SPY'
echo source >> "$LAB/sync.calls"
mc_skill_sync() { echo "$1" >> "$LAB/sync.calls"; SKILL_SYNC_STATUS=synced:2; SKILL_SYNC_NOTE='spy note'; }
SPY
  [ "$ARM" != driver-missing ] || rm "$LAB/driver/tools/launchd/lib/skill-sync.sh"
  # Execute extracted production code in a fresh process, not the helper-loaded caller.
  cat > "$LAB/runner.sh" <<'RUNNER'
set -eu
log() { printf '%s\n' "$*" >> "$LAB/driver.log"; }
_mc_uptime_secs() {
  if [ -f "$LAB/sync.calls" ] && grep -qx apply "$LAB/sync.calls" && grep -q '^skill-sync=' "$LAB/driver.log"; then
    echo boot:ready >> "$LAB/order"
  else
    echo boot:before-sync >> "$LAB/order"
  fi
  echo 99999
}
_mc_boot_offset() { echo 0; }
MC_DRIVER_ROOT="$LAB/driver"; KILL_SWITCH="$LAB/disabled"; PIDFILE="$LAB/pid"
MISSION_NAME=lab; MISSION_REPO=lab; MISSION_DOC=lab; REPO="$LAB/target"; PREFS=lab; HARD_TIMEOUT=1
MISSION_DESIGNER_MODEL=lab; MISSION_PLANNER_MODEL=lab; MISSION_EXECUTOR_MODEL=lab; MISSION_EVALUATOR_MODEL=lab
_lane_degraded=''; PIN_STATUS=pinned; PIN_DRIFT=0
case "$ARM" in
 driver-disabled) touch "$KILL_SWITCH";;
 driver-overlap) echo $$ > "$PIDFILE";;
 driver-dry|driver-field) MISSION_DRY_RUN=1;;
esac
. "$LAB/seam.sh"
RUNNER
  echo "FIXTURE $ARM ready"
  TESTING=1
  /bin/bash "$LAB/runner.sh"
  case "$ARM" in
    driver-disabled|driver-overlap) [ ! -e "$LAB/sync.calls" ] || fail 'sync invoked before guard'; [ ! -e "$LAB/order" ] || fail 'boot reached';;
    driver-dry|driver-field) eq "$(cat "$LAB/sync.calls")" "$(printf 'source\nreport')"; grep -q 'DRY RUN ok:.* | skill-sync=synced:2$' "$LAB/driver.log" || fail 'dry field absent'; [ ! -e "$LAB/order" ] || fail 'boot reached';;
    driver-real) eq "$(cat "$LAB/sync.calls")" "$(printf 'source\napply')"; grep -qx 'skill-sync=synced:2 spy note' "$LAB/driver.log" || fail 'note absent';
      eq "$(cat "$LAB/order")" boot:ready
      # Also assert the seam remains ahead of the production stagger declaration.
      python3 - "$LAB/seam.sh" <<'PY'
import sys
s=open(sys.argv[1]).read(); assert s.index('mc_skill_sync apply') < s.index('BOOT_WINDOW=')
PY
      grep -qx continuation "$LAB/driver.log" || fail 'continuation absent';;
    driver-missing) grep -qx 'skill-sync=error:helper-missing skill-sync helper absent' "$LAB/driver.log" || fail 'missing helper hidden'; [ ! -e "$LAB/sync.calls" ] || fail 'missing helper called';;
  esac
  snapshot "$LAB/after"; cmp "$LAB/before" "$LAB/after" || fail 'driver touched target'
  echo "PASS $ARM (isolated production seam)"; exit 0
fi
echo "FIXTURE $ARM ready"
TESTING=1
start=$(date +%s)
mc_skill_sync "$MODE"
echo survived > "$LAB/sentinel"
elapsed=$(( $(date +%s) - start ))
eq "$SKILL_SYNC_STATUS" "$EXPECTED"
eq "$PIN_STATUS/$PIN_NOTE/$PIN_BOUNDED_OUT" witness/witness/witness
[ -f "$LAB/sentinel" ] || fail 'caller did not survive'
actual=0; [ ! -f "$LAB/merge.calls" ] || actual=$(wc -l < "$LAB/merge.calls" | tr -d ' ')
eq "$actual" "$MERGES"
if [ "$ARM" = timeout ]; then [ "$elapsed" -lt 7 ] || fail "unbounded runtime $elapsed"; fi
if [ "$EXPECTED" = skip:dirty-range ]; then eq "$SKILL_SYNC_NOTE" 'colliding paths=1'; fi
if [ "$ARM" = report ]; then eq "$SKILL_SYNC_NOTE" 'would-sync 2 commits (report mode; checkout unchanged)'; fi
case "$ARM" in disabled|survival|invalid|missing-bounded|absent) [ ! -e "$LAB/git.calls" ] || fail 'unexpected Git call';; esac
case "$ARM" in
 happy|outside|symlink) eq "$(g -C "$LAB/target" rev-parse HEAD)" "$TIP"; cmp "$LAB/outside.before" "$LAB/target/outside" || fail 'dirty outside bytes changed' ;;
 late-lock) eq "$(g -C "$LAB/target" rev-parse HEAD)" "$HEAD_BEFORE"; eq "$(cat "$LAB/target/.git/index.lock")" 'late held'; cmp "$LAB/index.before" "$TARGET_GIT_DIR/index" || fail 'index changed' ;;
 changed|late-ref) eq "$(g -C "$LAB/target" rev-parse HEAD)" "$HEAD_BEFORE"; cmp "$LAB/index.before" "$TARGET_GIT_DIR/index" || fail 'index changed' ;;
 *) snapshot "$LAB/after"; cmp "$LAB/before" "$LAB/after" || fail 'target snapshot changed' ;;
esac
[ -z "$MARKER" ] || [ -e "$LAB/target/.git/$MARKER" ] || fail 'operation marker removed'
echo "PASS $ARM (status=$SKILL_SYNC_STATUS; merge=$actual; caller survived)"
