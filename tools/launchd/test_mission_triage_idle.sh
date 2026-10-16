#!/bin/bash
# Triage idle pre-check (M-TRIAGE-MISSION Part B): an unchanged queue fingerprint exits before
# any probe; a changed or never-recorded one proceeds; an unreadable one yields loudly; other
# missions never run it. Then the helper itself, against stubbed gh/ailang: format, what moves
# the hash and what must not, failed reads, and --record as the only writer.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
HELPER="$HERE/lib/triage-fingerprint.sh"
fail() { echo "FAIL $*"; exit 1; }
block=$(awk '/--- TRIAGE IDLE PRE-CHECK START ---/{on=1} on{print} /--- TRIAGE IDLE PRE-CHECK END ---/{exit}' "$DRIVER")
[ -n "$block" ] || fail "extraction"
case "$block" in *'exit 0'*) ;; *) fail "block never exits" ;; esac
case "$block" in *'tools/launchd/lib/triage-fingerprint.sh'*) ;; *) fail "block does not call the shared helper" ;; esac

TMP=$(mktemp -d -t mc-triage-idle); trap 'rm -rf "$TMP"' EXIT
H1=$(printf 'a%.0s' $(seq 1 64)); H2=$(printf 'b%.0s' $(seq 1 64))

run() {  # MISSION_NAME stub_rc stub_out recorded_line → "EXIT <rc> CALLS <n> | log…" or "FELLTHROUGH …"
  ( MISSION_NAME="$1"; STUB_RC="$2"; STUB_OUT="$3"; CALLS=0; LOGS=""
    STATE_DIR="$TMP/state-$$-$RANDOM"; mkdir -p "$STATE_DIR"; MC_DRIVER_ROOT=/nonexistent
    [ -n "$4" ] && printf '%s\n' "$4" > "$STATE_DIR/mission-triage-queue-fingerprint"
    log() { LOGS="$LOGS|$*"; }
    _mc_bounded() { CALLS=$((CALLS+1)); MC_BOUNDED_OUT="$STUB_OUT"; return "$STUB_RC"; }
    exit() { echo "EXIT ${1:-0} CALLS $CALLS $LOGS"; builtin exit 0; }
    eval "$block"
    echo "FELLTHROUGH CALLS $CALLS $LOGS" )
}
out=$(run triage 0 "$H1 prs=3 cards=2 issues=9" "$H1 prs=3 cards=2 issues=9")
case "$out" in "EXIT 0 CALLS 1 |triage: queue unchanged"*) ;; *) fail "unchanged did not idle: $out" ;; esac
out=$(run triage 0 "warning noise
$H2 prs=3 cards=2 issues=9" "$H1 prs=3 cards=2 issues=9")
case "$out" in "FELLTHROUGH CALLS 1 |triage: queue changed"*) ;; *) fail "changed did not proceed: $out" ;; esac
out=$(run triage 0 "$H1 prs=3 cards=2 issues=9" "")
case "$out" in "FELLTHROUGH CALLS 1 |triage: queue changed"*"none recorded"*) ;; *) fail "first fire (no record) did not proceed: $out" ;; esac
out=$(run triage 1 "triage-fingerprint: gh pr list failed" "$H1 x")
case "$out" in "EXIT 0 CALLS 1 |triage: cannot read"*) ;; *) fail "failed read: $out" ;; esac
out=$(run triage 1 "$H1 prs=3 cards=2 issues=9" "")
case "$out" in "EXIT 0 CALLS 1 |triage: cannot read"*) ;; *) fail "non-zero rc with a hash-shaped last line treated as a read: $out" ;; esac
out=$(run triage 124 "" "$H1 x")
case "$out" in "EXIT 0 CALLS 1 |triage: cannot read"*) ;; *) fail "timed-out read: $out" ;; esac
out=$(run triage 0 "" "")
case "$out" in "EXIT 0 CALLS 1 |triage: cannot read"*) ;; *) fail "empty output treated as a fingerprint: $out" ;; esac
out=$(run triage 0 "not-a-hash prs=1" "not-a-hash prs=1")
case "$out" in "EXIT 0 CALLS 1 |triage: cannot read"*) ;; *) fail "garbage equal to a garbage record treated as unchanged: $out" ;; esac
for m in v1 docs fleet world; do
  out=$(run "$m" 0 "$H1" "$H1")
  case "$out" in "FELLTHROUGH CALLS 0 "*) ;; *) fail "non-triage mission $m ran the check: $out" ;; esac
done
echo 'PASS triage idle pre-check: unchanged → idle exit, changed/unrecorded → proceeds, unreadable → loud yield, other missions skip'

# Placement: below the kill switch, above the first probe and the notice drain.
ln_kill=$(grep -n 'kill switch present' "$DRIVER" | head -1 | cut -d: -f1)
ln_idle=$(grep -n 'TRIAGE IDLE PRE-CHECK START' "$DRIVER" | cut -d: -f1)
ln_drain=$(grep -n '^_mc_drain_notices$' "$DRIVER" | head -1 | cut -d: -f1)
ln_probe=$(grep -n '^_an_probed=":"' "$DRIVER" | head -1 | cut -d: -f1)
[ "$ln_kill" -lt "$ln_idle" ] && [ "$ln_idle" -lt "$ln_drain" ] && [ "$ln_idle" -lt "$ln_probe" ] \
  || fail "placement kill=$ln_kill idle=$ln_idle drain=$ln_drain probe=$ln_probe"
grep -q '    triage) echo 2520 ;;' "$DRIVER" || fail "triage has no boot-offset arm (it would join the boot stampede)"
# Seam: the file the driver reads is the file the helper writes.
grep -q '"$STATE_DIR/mission-triage-queue-fingerprint"' "$DRIVER" || fail "driver reads a different state path"
grep -q 'fp_file="$state_dir/mission-triage-queue-fingerprint"' "$HELPER" || fail "helper writes a different state path"
grep -q 'state_dir="${TRIAGE_STATE_DIR:-$HOME/.ailang/state}"' "$HELPER" || fail "helper default state dir is not the driver's"
echo 'PASS placement: after the kill switch, before the drain and every probe; boot offset set; reader/writer share one path'

# --- The helper, against stubs --------------------------------------------------------------
command -v jq >/dev/null 2>&1 || fail "jq not on PATH (the helper needs it; /usr/bin/jq ships with macOS)"
BIN="$TMP/bin"; mkdir -p "$BIN"
cat > "$BIN/gh" <<'EOF'
#!/bin/bash
case "$1 $2" in
  "pr list")    [ -n "${STUB_GH_FAIL:-}" ] && { echo "HTTP 502" >&2; exit 1; }; cat "$STUB_DIR/prs.json" ;;
  "issue list") cat "$STUB_DIR/issues.json" ;;
  *) echo "unexpected gh $*" >&2; exit 9 ;;
esac
EOF
cat > "$BIN/ailang" <<'EOF'
#!/bin/bash
[ "$1 $2" = "coordinator approvals" ] || { echo "unexpected ailang $*" >&2; exit 9; }
echo "⚠ Binary may be stale" >&2
cat "$STUB_DIR/cards.json"
EOF
chmod +x "$BIN/gh" "$BIN/ailang"
S="$TMP/stub"; ST="$TMP/hstate"; mkdir -p "$S" "$ST"
fp() { STUB_DIR="$S" TRIAGE_STATE_DIR="$ST" PATH="$BIN:$PATH" /bin/bash "$HELPER" "$@" 2>"$TMP/herr"; }
reset_stub() {
  echo '[{"number":10,"headRefOid":"aaa"},{"number":11,"headRefOid":"bbb"}]' > "$S/prs.json"
  echo '[{"number":20,"updatedAt":"2026-10-01T00:00:00Z"},{"number":99,"updatedAt":"2026-10-15T00:00:00Z"}]' > "$S/issues.json"
  echo '{"pending":[{"id":"apr-1"},{"id":"apr-2"}],"orphans":[]}' > "$S/cards.json"
  echo 99 > "$ST/mission-triage-gh-issue"; rm -f "$ST/mission-triage-queue-fingerprint"
}
reset_stub
base=$(fp) || fail "helper rc!=0 on good input: $(cat "$TMP/herr")"
printf '%s\n' "$base" | grep -Eq '^[0-9a-f]{64} prs=2 cards=2 issues=1$' || fail "helper output shape: $base"
[ -f "$ST/mission-triage-queue-fingerprint" ] && fail "a bare run wrote the fingerprint file (only --record may)"
h0=${base%% *}

echo '[{"number":11,"headRefOid":"bbb"},{"number":10,"headRefOid":"aaa"}]' > "$S/prs.json"
[ "$(fp | cut -d' ' -f1)" = "$h0" ] || fail "order of gh output changed the hash"
echo '[{"number":20,"updatedAt":"2026-10-01T00:00:00Z"},{"number":99,"updatedAt":"2026-10-16T09:00:00Z"}]' > "$S/issues.json"
[ "$(fp | cut -d' ' -f1)" = "$h0" ] || fail "the bookkeeping thread's own activity changed the hash (every fire would look changed)"
reset_stub; echo '[{"number":10,"headRefOid":"aaa"},{"number":11,"headRefOid":"ccc"}]' > "$S/prs.json"
[ "$(fp | cut -d' ' -f1)" != "$h0" ] || fail "a new PR head did not change the hash"
reset_stub; echo '{"pending":[{"id":"apr-1"},{"id":"apr-3"}]}' > "$S/cards.json"
[ "$(fp | cut -d' ' -f1)" != "$h0" ] || fail "a new pending card did not change the hash"
reset_stub; echo '[{"number":20,"updatedAt":"2026-10-02T00:00:00Z"},{"number":99,"updatedAt":"2026-10-15T00:00:00Z"}]' > "$S/issues.json"
[ "$(fp | cut -d' ' -f1)" != "$h0" ] || fail "an issue update did not change the hash"
echo 'PASS helper: sorted, bookkeeping thread excluded, PR head / card / issue update each move the hash'

reset_stub
out=$(STUB_GH_FAIL=1 fp); rc=$?
[ "$rc" -ne 0 ] && [ -z "$out" ] || fail "gh failure: rc=$rc out=$out"
grep -q 'gh pr list failed' "$TMP/herr" || fail "gh failure not named on stderr: $(cat "$TMP/herr")"
echo '{"error":"permission denied"}' > "$S/cards.json"
out=$(fp); rc=$?
[ "$rc" -ne 0 ] && [ -z "$out" ] || fail "approvals JSON without .pending accepted: rc=$rc out=$out"
reset_stub; echo 'not json' > "$S/prs.json"
out=$(fp); rc=$?
[ "$rc" -ne 0 ] && [ -z "$out" ] || fail "unparseable PR list accepted: rc=$rc out=$out"
echo 'PASS helper: failed or unparseable reads exit non-zero and print no fingerprint'

reset_stub
rec=$(fp --record) || fail "--record rc!=0"
[ "$(cat "$ST/mission-triage-queue-fingerprint")" = "$rec" ] || fail "--record did not write what it printed"
[ "${rec%% *}" = "$h0" ] || fail "--record hash differs from a bare run"
reset_stub; echo '{"pending":"x"}' > "$S/cards.json"; echo "$rec" > "$ST/mission-triage-queue-fingerprint"
fp --record >/dev/null && fail "--record succeeded on a failed read"
[ "$(cat "$ST/mission-triage-queue-fingerprint")" = "$rec" ] || fail "a failed --record clobbered the last good fingerprint"
echo 'PASS helper: --record writes exactly what it prints, and never on a failed read'
