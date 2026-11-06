#!/bin/bash
# driver:idle-fire-writes-full-block-report — extracted block + real local Git, stub APIs.
# All state (including HOME and Git config) is private to this worktree-local mktemp.
# The extracted block invokes these functions/variables through eval; subshell
# isolation and literal mutation strings are intentional.
# shellcheck disable=SC2030,SC2031,SC2034,SC2329,SC2016
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
DRIVER=${IDLE_TEST_DRIVER:-$HERE/mission-control.sh}
HELPER=${IDLE_TEST_HELPER:-$HERE/lib/idle-fingerprint.sh}
fail() { echo "FAIL $*" >&2; exit 1; }
TMP=$(mktemp -d "$ROOT/.mission-idle-test.XXXXXX") || exit 1
trap 'rm -rf "$TMP"' EXIT
export HOME="$TMP/home" TMPDIR="$TMP/tmp" IDLE_STATE_DIR="$TMP/state"
export AILANG_STATE_DIR="$TMP/slots" STATE_DIR="$IDLE_STATE_DIR"
mkdir -p "$HOME" "$TMPDIR" "$STATE_DIR" "$AILANG_STATE_DIR"
unset MISSION_DRY_RUN MISSION_FIRE_ID
# Ambient scope guards/config must never turn a fixture into a write to the host repo.
for key in $(env | sed -n 's/^\(GIT_[A-Za-z0-9_]*\)=.*/\1/p'); do unset "$key"; done
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=idle-test GIT_AUTHOR_EMAIL=idle@invalid
export GIT_COMMITTER_NAME=idle-test GIT_COMMITTER_EMAIL=idle@invalid
export LC_ALL=C
case "${1:-}" in
  --mutations)
    /bin/bash "$0" >"$TMP/control" 2>&1 || { cat "$TMP/control"; fail 'pristine control'; }
    for mutant in excludes wrong-record keep-changed fail-closed age limit pending store-pin dry-run fire-check record-as-snapshot snapshot-keeps-stale; do
      cp "$DRIVER" "$TMP/driver"; cp "$HELPER" "$TMP/helper"
      case "$mutant" in
        excludes) sed '/:(exclude)/d' "$HELPER" >"$TMP/helper" ;;
        wrong-record) sed 's|_mi_file="$STATE_DIR/mission-${MISSION_NAME}-idle-fingerprint"|_mi_file="$STATE_DIR/mission-wrong-idle-fingerprint"|' "$DRIVER" >"$TMP/driver" ;;
        keep-changed) awk '/rm -f "\$_mi_file" # changed/{next} {print}' "$DRIVER" >"$TMP/driver" ;;
        fail-closed) sed '/# failed read consumed/a\
      exit 0
' "$DRIVER" >"$TMP/driver" ;;
        age) sed 's/"$_mi_age" -gt 86400/"$_mi_age" -gt 999999999/' "$DRIVER" >"$TMP/driver" ;;
        limit) sed 's/\[ "$n_in" -lt "$LIMIT" \]/true/' "$HELPER" >"$TMP/helper" ;;
        store-pin) sed 's/^env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT="[^"]*" \\$/  \\/' "$HELPER" >"$TMP/helper" ;;
        dry-run) sed 's/ && \[ "${MISSION_DRY_RUN:-0}" != 1 \]; then/; then/' "$DRIVER" >"$TMP/driver" ;;
        fire-check) sed 's/\*" fire=$fire at="\*) ;;/*) ;;/' "$HELPER" >"$TMP/helper" ;;
        record-as-snapshot) sed 's/1:--record) mode=record ;;/1:--record) mode=snapshot ;;/' "$HELPER" >"$TMP/helper" ;;
        snapshot-keeps-stale) awk '/^  rm -f "\$STATE_DIR\/mission-\$\{MISSION_NAME\}-idle-fingerprint.pending"$/ && !done {done=1; next} {print}' "$DRIVER" >"$TMP/driver" ;;
        pending) sed 's/\["failure", "timed_out", "cancelled", "action_required", "startup_failure"\]/["failure", "timed_out", "cancelled", "action_required", "startup_failure", null]/' "$HELPER" >"$TMP/helper" ;;
      esac
      if cmp -s "$DRIVER" "$TMP/driver" && cmp -s "$HELPER" "$TMP/helper"; then fail "mutant $mutant unchanged"; fi
      IDLE_TEST_DRIVER="$TMP/driver" IDLE_TEST_HELPER="$TMP/helper" /bin/bash "$0" >"$TMP/mutant.log" 2>&1
      rc=$?
      if [ "$rc" -eq 0 ] || ! grep -q '^FAIL ' "$TMP/mutant.log"; then
        cat "$TMP/mutant.log"; fail "$mutant survived or fixture failed"
      fi
      echo "MUTATION $mutant KILLED: $(sed -n '/^FAIL /{p;q;}' "$TMP/mutant.log")"
    done
    exit 0 ;;
  '') ;;
  *) exit 2 ;;
esac
block=$(awk '/--- MISSION IDLE PRE-CHECK START ---/{on=1} on{print} /--- MISSION IDLE PRE-CHECK END ---/{exit}' "$DRIVER")
[ -n "$block" ] || fail extraction
H1=$(printf '%064d' 0); H2=$(printf '%064d' 1)
NOW=$(date +%s)
run() { # mission rc output record [work-item]
  ( export MISSION_NAME="$1" AILANG_MISSION_WORK_ITEM="${5:-}"
    STUB_RC=$2; STUB_OUT=$3; CALLS=0; LOGS=""
    export STATE_DIR="$TMP/block-$RANDOM" IDLE_STATE_DIR="$TMP/block-$RANDOM"
    IDLE_STATE_DIR=$STATE_DIR
    export AILANG_STATE_DIR="$STATE_DIR/slots"
    mkdir -p "$STATE_DIR" "$AILANG_STATE_DIR"
    f="$STATE_DIR/mission-${MISSION_NAME}-idle-fingerprint"
    [ -z "$4" ] || printf '%s\n' "$4" >"$f"
    if [ -n "${SEED_ROWS:-}" ]; then
      i=0; while [ "$i" -lt "$SEED_ROWS" ]; do echo "old-row $i"; i=$((i+1)); done \
        >"$AILANG_STATE_DIR/mission-${MISSION_NAME}-slot-verdicts.log"
    fi
    MC_DRIVER_ROOT=/nonexistent; TRANSIENT_RETRIES=3
    log() { LOGS="$LOGS|$*"; }
    _mc_bounded() {
      [ "$1" = 180 ] && [ "$2" = /bin/bash ] && [ "$3" = /nonexistent/tools/launchd/lib/idle-fingerprint.sh ] || fail 'bounded helper command'
      [ ! -t 0 ] && [ -z "$(cat)" ] || fail 'helper stdin not EOF'
      [ "${IDLE_STATE_DIR:-}" = "$STATE_DIR" ] || fail 'helper state seam'
      CALLS=$((CALLS+1)); MC_BOUNDED_OUT="$STUB_OUT"; return "$STUB_RC"
    }
    result() {
      kept=no; [ ! -f "$f" ] || kept=yes
      rows=0; slot="$AILANG_STATE_DIR/mission-${MISSION_NAME}-slot-verdicts.log"
      [ ! -f "$slot" ] || rows=$(wc -l <"$slot" | tr -d ' ')
      echo "$1 CALLS=$CALLS kept=$kept rows=$rows $LOGS"
      [ ! -f "$slot" ] || cat "$slot"
    }
    exit() { result "EXIT ${1:-0}"; builtin exit 0; }
    eval "$block"
    result FELLTHROUGH
  ) < /dev/null
}
out=$(run world 0 "noise
$H1 head=1234567" "$H1 at=$NOW")
case "$out" in 'EXIT 0 CALLS=1 kept=yes rows=1 '* ) ;; *) fail "unchanged: $out";; esac
printf '%s\n' "$out" | grep -Eq '^[0-9TZ:-]+ verdict=IDLE rc=0 attempt=1/3 elapsed_s=0 stamps=0 controller=none$' || fail "IDLE layout: $out"
out=$(SEED_ROWS=250 run world 0 "$H1" "$H1 at=$NOW")
case "$out" in 'EXIT 0 CALLS=1 kept=yes rows=200 '*) ;; *) fail "slot history not trimmed to 200: $(printf '%s' "$out" | head -1)";; esac
out=$(MISSION_DRY_RUN=1 run world 0 "$H1" "$H1 at=$NOW")
case "$out" in 'FELLTHROUGH CALLS=0 kept=yes rows=0 '*) ;; *) fail "dry run touched the idle record: $out";; esac
out=$(run world 0 "$H2" "$H1 at=$NOW")
case "$out" in 'FELLTHROUGH CALLS=1 kept=no rows=0 '*changed*) ;; *) fail "changed: $out";; esac
out=$(run world 0 "$H1" '')
case "$out" in 'FELLTHROUGH CALLS=0 kept=no rows=0 '*) ;; *) fail "no record: $out";; esac
for at in "$((NOW-86401))" garbled '' 999999999999999999999999999999999999999; do
  out=$(run world 0 "$H1" "$H1 at=$at")
  case "$out" in 'FELLTHROUGH CALLS=0 kept=no rows=0 '*expired*) ;; *) fail "expired ($at): $out";; esac
done
out=$(run world 0 "$H1" "$H1 missing-timestamp")
case "$out" in 'FELLTHROUGH CALLS=0 kept=no rows=0 '*expired*) ;; *) fail "missing at: $out";; esac
for kind in error timeout empty garbage hash-error; do
  rc=0; val=''
  case "$kind" in
    error) rc=1; val='idle-fingerprint: HTTP 502';;
    timeout) rc=124; val='timed out';;
    garbage) val='not-a-hash';;
    hash-error) rc=1; val="$H1";;
  esac
  out=$(run world "$rc" "$val" "$H1 at=$NOW")
  case "$out" in 'FELLTHROUGH CALLS=1 kept=no rows=0 '*cannot\ read*proceeding*) ;; *) fail "failed read $kind: $out";; esac
  case "$out" in *"rc=$rc"*"$val"*) ;; *) fail "lost failure diagnostic: $out";; esac
done
for m in fleet triage world; do
  wi=''; [ "$m" != world ] || wi=item-1
  out=$(run "$m" 0 "$H1" "$H1 at=$NOW" "$wi")
  case "$out" in 'FELLTHROUGH CALLS=0 kept=yes rows=0 '*) ;; *) fail "excluded $m/$wi: $out";; esac
done
ln_kill=$(grep -n 'kill switch present' "$DRIVER" | head -1 | cut -d: -f1)
ln_tr=$(grep -n 'TRIAGE IDLE PRE-CHECK END' "$DRIVER" | cut -d: -f1)
ln_idle=$(grep -n 'MISSION IDLE PRE-CHECK START' "$DRIVER" | cut -d: -f1)
ln_drain=$(grep -n '^_mc_drain_notices$' "$DRIVER" | head -1 | cut -d: -f1)
ln_probe=$(grep -n '^_an_probed=":"' "$DRIVER" | head -1 | cut -d: -f1)
[ "$ln_kill" -lt "$ln_tr" ] && [ "$ln_idle" -eq "$((ln_tr+2))" ] && [ "$ln_idle" -lt "$ln_drain" ] && [ "$ln_idle" -lt "$ln_probe" ] || fail placement
echo 'PASS driver arms, slot layout, stdin EOF, state seam and placement'

# Fire-start snapshot (evaluator finding, iteration 91): the record must be the inputs the
# controller STARTED from, so the snapshot block runs once per fire before the controller.
sblock=$(awk '/--- MISSION IDLE SNAPSHOT START ---/{on=1} on{print} /--- MISSION IDLE SNAPSHOT END ---/{exit}' "$DRIVER")
[ -n "$sblock" ] || fail 'snapshot extraction'
snap() { # mission rc write(yes|no) [work-item]
  ( export MISSION_NAME="$1" AILANG_MISSION_WORK_ITEM="${4:-}" MISSION_FIRE_ID=fire-9
    STUB_RC=$2; WRITE=$3; CALLS=0; LOGS=""
    STATE_DIR="$TMP/snap-$RANDOM"; mkdir -p "$STATE_DIR"
    pf="$STATE_DIR/mission-${MISSION_NAME}-idle-fingerprint.pending"
    echo stale >"$pf"
    MC_DRIVER_ROOT=/nonexistent
    log() { LOGS="$LOGS|$*"; }
    _mc_bounded() {
      [ "$1" = 180 ] && [ "$2" = /bin/bash ] && [ "$3" = /nonexistent/tools/launchd/lib/idle-fingerprint.sh ] \
        && [ "$4" = --snapshot ] || fail 'snapshot argv'
      [ ! -t 0 ] && [ -z "$(cat)" ] || fail 'snapshot stdin not EOF'
      [ "${IDLE_STATE_DIR:-}" = "$STATE_DIR" ] || fail 'snapshot state seam'
      [ "$(cat "$pf" 2>/dev/null)" != stale ] || fail 'stale pending not cleared before the snapshot'
      CALLS=$((CALLS+1)); [ "$WRITE" != yes ] || echo "$H1 fire=fire-9 at=1" >"$pf"
      MC_BOUNDED_OUT="$H1 x"; return "$STUB_RC"
    }
    eval "$sblock"
    has=no; [ ! -f "$pf" ] || has=yes
    echo "CALLS=$CALLS pending=$has $LOGS" ) < /dev/null
}
out=$(snap world 0 yes)
case "$out" in 'CALLS=1 pending=yes |world: idle snapshot taken for fire-9'*) ;; *) fail "snapshot ok: $out";; esac
out=$(snap world 1 yes)
case "$out" in 'CALLS=1 pending=no '*'cannot be recorded'*) ;; *) fail "failed snapshot kept a pending: $out";; esac
out=$(snap world 0 no)
case "$out" in 'CALLS=1 pending=no '*'cannot be recorded'*) ;; *) fail "rc0 without a file: $out";; esac
for m in fleet triage; do
  out=$(snap "$m" 0 yes)
  case "$out" in 'CALLS=0 pending=yes '*) ;; *) fail "snapshot ran for $m: $out";; esac
done
out=$(snap world 0 yes item-1)
case "$out" in 'CALLS=0 pending=yes '*) ;; *) fail "snapshot ran for a work item: $out";; esac
ln_snap=$(grep -n 'MISSION IDLE SNAPSHOT START' "$DRIVER" | cut -d: -f1)
ln_dry=$(grep -n 'log "DRY RUN ok:' "$DRIVER" | head -1 | cut -d: -f1)
ln_pid=$(grep -n '> "\$PIDFILE"' "$DRIVER" | head -1 | cut -d: -f1)
ln_fire=$(grep -n '^MISSION_FIRE_ID=' "$DRIVER" | head -1 | cut -d: -f1)
ln_ctl=$(grep -n 'CONTROLLER_PID=\$!' "$DRIVER" | head -1 | cut -d: -f1)
[ -n "$ln_snap" ] && [ "$ln_dry" -lt "$ln_snap" ] && [ "$ln_pid" -lt "$ln_snap" ] && [ "$ln_fire" -lt "$ln_snap" ] \
  && [ "$ln_snap" -lt "$ln_ctl" ] || fail "snapshot placement dry=$ln_dry pid=$ln_pid fire=$ln_fire snap=$ln_snap ctl=$ln_ctl"
echo 'PASS fire-start snapshot: argv, stdin EOF, stale cleared, failures leave none, exclusions, placement'

# Real Git fixture. Git mutations below are ONLY inside this disposable repository.
REAL_GIT=$(command -v git); export REAL_GIT
BIN="$TMP/bin"; S="$TMP/stubs"; mkdir -p "$BIN" "$S"
export STUB_DIR="$S" MISSION_NAME=world MISSION_REPO=example/world
export MISSION_DOC=design_docs/world-mission.md MISSION_WORKDIR="$TMP/work"
export MISSION_DIRECTIVE_AUTHORS=' MarkEdmondson1234 , second-author '
"$REAL_GIT" init -q --bare "$TMP/origin" || fail 'bare init'
"$REAL_GIT" clone -q "$TMP/origin" "$MISSION_WORKDIR" 2>"$TMP/git.err" || fail clone
"$REAL_GIT" -C "$MISSION_WORKDIR" symbolic-ref HEAD refs/heads/dev
mkdir -p "$MISSION_WORKDIR/design_docs"
charter() {
  cat >"$MISSION_WORKDIR/$MISSION_DOC" <<DOC
<!-- decision-ledger:start -->
| ID | STATUS | Decision | Evidence |
| D-WORLD-1 | $1 | a ruling | evidence |
<!-- decision-ledger:end -->
DOC
}
commit() {
  "$REAL_GIT" -C "$MISSION_WORKDIR" add . || fail 'fixture add'
  "$REAL_GIT" -C "$MISSION_WORKDIR" -c core.hooksPath=/dev/null commit -qm fixture || fail 'fixture commit'
  "$REAL_GIT" -C "$MISSION_WORKDIR" -c core.hooksPath=/dev/null push -q origin HEAD:dev || fail 'fixture push'
}
charter OPEN; echo product >"$MISSION_WORKDIR/product.txt"; commit
"$REAL_GIT" -C "$MISSION_WORKDIR" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/dev
cat >"$BIN/git" <<'STUB'
#!/bin/bash
case "$*" in *' fetch '*) [ -z "${STUB_FETCH_FAIL:-}" ] || { echo 'fetch failed' >&2; exit 1; };; esac
exec "$REAL_GIT" "$@"
STUB
cat >"$BIN/gh" <<'STUB'
#!/bin/bash
[ -z "${STUB_GH_FAIL:-}" ] || { echo 'HTTP 502' >&2; exit 1; }
case "$*" in
  'api --paginate repos/example/world/issues/99/comments?per_page=100') cat "$STUB_DIR/comments";;
  'api repos/example/world/commits/'*'/check-runs --paginate') echo "$2" >"$STUB_DIR/checks_path"; cat "$STUB_DIR/checks";;
  *) echo "unexpected gh $*" >&2; exit 9;;
esac
STUB
cat >"$BIN/ailang" <<'STUB'
#!/bin/bash
[ "$*" = 'messages list --unread --inbox mission-world --json --limit 1000' ] || { echo "unexpected ailang $*" >&2; exit 9; }
# The reader (launchd bash driver) and the writer (controller zsh) inherit different stores; the
# helper must pin the canonical one itself or the two hash different inboxes.
[ "${AILANG_STORAGE_MESSAGING:-}" = gcp ] && [ -n "${AILANG_MESSAGES_PROJECT:-}" ] \
  || { echo "store not pinned: ${AILANG_STORAGE_MESSAGING:-unset}" >&2; exit 8; }
[ -z "${STUB_MAIL_FAIL:-}" ] || { echo 'mail failed' >&2; exit 1; }
echo 'warning only on stderr' >&2
cat "$STUB_DIR/mail"
STUB
chmod +x "$BIN/git" "$BIN/gh" "$BIN/ailang"
export PATH="$BIN:$PATH"
# Hostile caller store: what the launchd driver would pass if anything set it, or local.
export AILANG_STORAGE_MESSAGING=local; unset AILANG_MESSAGES_PROJECT
reset_stub() {
  echo '[]' >"$S/comments"; echo '[]' >"$S/mail"
  echo '{"total_count":0,"check_runs":[]}' >"$S/checks"
  echo 99 >"$STATE_DIR/mission-world-gh-issue"
}
fp() { /bin/bash "$HELPER" "$@" 2>"$TMP/herr"; }
hash() { fp | cut -d' ' -f1; }
bad() {
  out=$(fp "$@"); rc=$?
  [ "$rc" -ne 0 ] && [ -z "$out" ] || fail "failed read accepted rc=$rc: $out"
  [ -s "$TMP/herr" ] || fail 'no failure diagnostic'
}
reset_stub
base=$(fp) || fail "good helper: $(cat "$TMP/herr")"
printf '%s\n' "$base" | grep -Eq '^[0-9a-f]{64} head=[0-9a-f]{7} ledger=1 open=1 directives=0 inbox=0 red=0$' || fail "format: $base"
h0=${base%% *}
[ ! -f "$STATE_DIR/mission-world-idle-fingerprint" ] || fail 'bare run wrote record'
# Both exclusion patterns must work on the rig Git, using commits in a real origin.
echo log >"$MISSION_WORKDIR/design_docs/world-mission-log.md"; commit
[ "$(hash)" = "$h0" ] || fail 'record-only design doc commit moved hash'
mkdir -p "$MISSION_WORKDIR/verification/world-iter9"
echo record >"$MISSION_WORKDIR/verification/world-iter9/x.txt"; commit
[ "$(hash)" = "$h0" ] || fail 'record-only verification commit moved hash'
# Red checks are read on the branch TIP (which a record-only merge moves), not on head=.
tip=$("$REAL_GIT" -C "$MISSION_WORKDIR" rev-parse origin/dev)
grep -q "/commits/$tip/check-runs" "$S/checks_path" || fail "check-runs not read on the branch tip ($tip): $(cat "$S/checks_path")"
echo changed >>"$MISSION_WORKDIR/product.txt"; commit
h1=$(hash); [ -n "$h1" ] && [ "$h1" != "$h0" ] || fail 'non-record head did not move hash'
# Charter is read from origin, never the working tree; ledger-only record commits still wake.
charter RESOLVED
[ "$(hash)" = "$h1" ] || fail 'working tree charter read'
commit
base=$(fp); h0=${base%% *}
[ "$h0" != "$h1" ] && [[ "$base" = *'ledger=1 open=0'* ]] || fail 'ledger flip did not move hash'
echo '[{"id":1,"user":{"login":"bot"},"updated_at":"t1"}]' >"$S/comments"
[ "$(hash)" = "$h0" ] || fail 'non-allowlisted comment moved hash'
printf '%s\n%s\n' '[]' '[{"id":2,"user":{"login":"second-author"},"updated_at":"t1"}]' >"$S/comments"
h1=$(hash); [ -n "$h1" ] && [ "$h1" != "$h0" ] || fail 'allowlisted directive did not move hash'
sed 's/t1/t2/' "$S/comments" >"$S/edit"; mv "$S/edit" "$S/comments"
[ "$(hash)" != "$h1" ] || fail 'directive edit did not move hash'
reset_stub; echo '[{"message_id":"msg-1","id":"internal-1"}]' >"$S/mail"
[ "$(hash)" != "$h0" ] || fail 'unread mail did not move hash'
reset_stub; echo '[{"message_id":"msg-1"}]' >"$S/mail"; ha=$(hash)
echo '[{"message_id":"msg-2"}]' >"$S/mail"
[ "$(hash)" != "$ha" ] || fail 'inbox id swap (same count) did not move hash'
reset_stub; echo null >"$S/mail"
[ "$(hash)" = "$h0" ] || fail 'empty nil slice rejected'
reset_stub
for conclusion in failure timed_out cancelled action_required startup_failure; do
  printf '{"total_count":1,"check_runs":[{"name":"ci","conclusion":"%s"}]}\n' "$conclusion" >"$S/checks"
  [ "$(hash)" != "$h0" ] || fail "red $conclusion did not move hash"
done
for conclusion in null '"success"' '"skipped"' '"neutral"'; do
  printf '{"total_count":1,"check_runs":[{"name":"ci","conclusion":%s}]}\n' "$conclusion" >"$S/checks"
  [ "$(hash)" = "$h0" ] || fail "green/pending $conclusion moved hash"
done
reset_stub
STUB_FETCH_FAIL=1 bad; STUB_GH_FAIL=1 bad; STUB_MAIL_FAIL=1 bad
for part in comments checks mail; do
  for broken in 'not json' '{}' ''; do
    reset_stub; echo "$broken" >"$S/$part"; bad
  done
done
reset_stub; jq -n '[range(1000) | {message_id:(tostring)}]' >"$S/mail"; bad
reset_stub; MISSION_DIRECTIVE_AUTHORS=' , ' bad
reset_stub; rm "$STATE_DIR/mission-world-gh-issue"; bad
reset_stub; echo '99oops' >"$STATE_DIR/mission-world-gh-issue"; bad
for marker in start end; do
  reset_stub; echo "<!-- decision-ledger:$marker -->" >"$MISSION_WORKDIR/$MISSION_DOC"; commit; bad
done
echo 'charter without ledger' >"$MISSION_WORKDIR/$MISSION_DOC"; commit
fp | grep -q 'ledger=0 open=0' || fail 'absent ledger is not zero rows'
# Missing symbolic ref uses origin/dev; a present origin/HEAD selects its own branch.
"$REAL_GIT" -C "$MISSION_WORKDIR" symbolic-ref --delete refs/remotes/origin/HEAD
fp >/dev/null || fail 'origin/dev fallback'
"$REAL_GIT" -C "$MISSION_WORKDIR" -c core.hooksPath=/dev/null push -q origin HEAD:main
"$REAL_GIT" -C "$MISSION_WORKDIR" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main
fp >/dev/null || fail 'origin/HEAD main branch'
# Observe the default state path; never seed it to manufacture an unchanged assertion.
observe_default() {
  if [ -e "$HOME/.ailang/state" ]; then find "$HOME/.ailang/state" -type f -exec shasum -a 256 {} \; | sort; else echo absent; fi
}
before=$(observe_default)
file="$STATE_DIR/mission-world-idle-fingerprint"; pend="$file.pending"
fp --record >/dev/null; rc=$?
[ "$rc" -ne 0 ] && [ ! -f "$file" ] || fail 'record without a fire id'
MISSION_FIRE_ID=fire-1 fp --record >/dev/null; rc=$?
[ "$rc" -ne 0 ] && [ ! -f "$file" ] || fail 'record without a snapshot'
fp --snapshot >/dev/null; rc=$?
[ "$rc" -ne 0 ] && [ ! -f "$pend" ] || fail 'snapshot without a fire id'
snapline=$(MISSION_FIRE_ID=fire-1 fp --snapshot) || fail "snapshot: $(cat "$TMP/herr")"
[ ! -f "$file" ] || fail 'snapshot wrote the record'
grep -Eq '^[0-9a-f]{64} head=.* fire=fire-1 at=[0-9]+$' "$pend" || fail "pending format: $(cat "$pend")"
[ "$(sed 's/ fire=fire-1 at=[0-9]*$//' "$pend")" = "$snapline" ] || fail 'pending differs from output'
MISSION_FIRE_ID=fire-2 fp --record >/dev/null; rc=$?
[ "$rc" -ne 0 ] && [ ! -f "$file" ] && [ -f "$pend" ] || fail 'another fire recorded this snapshot'
# The evaluator's race: an input that changes DURING the fire must wake the next one.
snaph=${snapline%% *}
echo mid-fire >>"$MISSION_WORKDIR/product.txt"; commit
# origin/HEAD points at main since the branch-choice arms above.
"$REAL_GIT" -C "$MISSION_WORKDIR" -c core.hooksPath=/dev/null push -q origin HEAD:main || fail 'fixture push main'
[ "$(hash)" != "$snaph" ] || fail 'mid-fire change did not move the live hash'
rec=$(MISSION_FIRE_ID=fire-1 fp --record) || fail "record: $(cat "$TMP/herr")"
[ "${rec%% *}" = "$snaph" ] && [ "$(cut -d' ' -f1 "$file")" = "$snaph" ] || fail 'record recomputed instead of promoting the snapshot'
[ ! -f "$pend" ] || fail 'promoted snapshot left pending'
grep -Eq ' fire=fire-1 at=[0-9]+$' "$file" || fail 'record timestamp'
[ "$before" = "$(observe_default)" ] || fail 'default HOME state was written'
# Re-record against the current inputs for the dynamic seam below.
MISSION_FIRE_ID=fire-3 fp --snapshot >/dev/null || fail 're-snapshot'
MISSION_FIRE_ID=fire-3 fp --record >/dev/null || fail 're-record'
# Dynamic seam: the actual helper writes a record, then the extracted driver calls
# that helper through its real argv and reads the same file. Test both slot dirs.
mkdir -p "$TMP/driver-root/tools/launchd/lib"
cp "$HELPER" "$TMP/driver-root/tools/launchd/lib/idle-fingerprint.sh"
for slots in consolidated fallback; do
  out=$(
    MC_DRIVER_ROOT="$TMP/driver-root"; TRANSIENT_RETRIES=3
    export AILANG_MISSION_WORK_ITEM=''
    [ "$slots" != fallback ] || unset AILANG_STATE_DIR
    CALLS=0
    log() { :; }
    _mc_bounded() { shift; CALLS=$((CALLS+1)); MC_BOUNDED_OUT=$("$@" 2>&1); }
    exit() { echo "SEAM EXIT ${1:-0} CALLS=$CALLS"; builtin exit 0; }
    eval "$block"
    echo 'SEAM FELLTHROUGH'
  )
  [ "$out" = 'SEAM EXIT 0 CALLS=1' ] || fail "dynamic $slots seam: $out"
  slot_dir=${AILANG_STATE_DIR:-$STATE_DIR}
  [ "$slots" != fallback ] || slot_dir=$STATE_DIR
  grep -q 'verdict=IDLE rc=0' "$slot_dir/mission-world-slot-verdicts.log" || fail "$slots slot state path"
done
old=$(cat "$file"); STUB_GH_FAIL=1 MISSION_FIRE_ID=fire-4 bad --snapshot
[ "$(cat "$file")" = "$old" ] && [ ! -f "$pend" ] || fail 'failed snapshot clobbered the record or left a pending'
for m in fleet triage ''; do
  out=$(MISSION_NAME="$m" fp); rc=$?
  [ "$rc" -eq 2 ] && [ -z "$out" ] || fail "refusal $m: rc=$rc"
done
for args in '--bad' '--record extra' '--snapshot extra'; do
  # Deliberate splitting exercises the two-argument usage refusal.
  # shellcheck disable=SC2086
  fp $args >/dev/null; rc=$?
  [ "$rc" -eq 2 ] || fail "usage $args: rc=$rc"
done
echo 'PASS helper sections, exclusions, failed reads, branch choice, atomic record and refusals'
