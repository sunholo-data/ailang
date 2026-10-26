#!/usr/bin/env bash
# test_mission_directives.sh — crash-safety arms for scripts/mission_directives.sh (#981).
#
# Issue #981: Gate 0 advanced the directive watermark before routing, a fire died, and two of
# Mark's answers were skipped for good. The watermark also equalled a comment's createdAt while
# the read was exclusive, so that comment could never come back. These arms drive the protocol
# the Gate-0 skill now prescribes — read with --watermark-file, route, then --mark <id> — through
# a simulated fire that can be killed at every step, and assert that nothing is lost and nothing
# is routed twice.
#
# Every arm builds its own fixture and runs the script under /bin/bash (the rig is bash 3.2).
# `gh` is a stub on PATH that serves "$MD_FIXDIR/<issue>.json"; nothing touches the network or
# ~/.ailang. MD_SCRIPT_UT points the suite at a mutated copy for mutation testing.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
SCRIPT_UT="${MD_SCRIPT_UT:-scripts/mission_directives.sh}"
pass=0; fail=0
ok()   { echo "ok $*";      pass=$((pass+1)); }
notok(){ echo "not ok $*";  fail=$((fail+1)); }

SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/md_scratch.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT

# Stub gh: `api user` answers the fleet login; `issue view N ...` serves the fixture.
mkdir -p "$SCRATCH/bin"
cat > "$SCRATCH/bin/gh" <<'EOF'
#!/bin/bash
case "$1 $2" in
  "api user") echo "sunholo-voight-kampff" ;;
  "issue view") f="$MD_FIXDIR/$3.json"; [ -f "$f" ] || { echo "stub gh: no fixture $f" >&2; exit 1; }; cat "$f" ;;
  *) echo "stub gh: unexpected: $*" >&2; exit 1 ;;
esac
EOF
chmod +x "$SCRATCH/bin/gh"
PATH="$SCRATCH/bin:$PATH"; export PATH
unset MISSION_DIRECTIVE_AUTHORS MISSION_REPO

md() { /bin/bash "$SCRIPT_UT" --repo test/repo "$@"; }

# Two allowlisted directives on issue 5 plus public noise and a self comment.
two_directives() {  # $1 = fixture dir
	mkdir -p "$1"
	cat > "$1/5.json" <<'EOF'
{"comments": [
 {"id":"IC_old","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-08-29T09:06:45Z","body":"D-49: A"},
 {"id":"IC_pub","author":{"login":"someone-else"},"createdAt":"2026-08-29T10:00:00Z","body":"D-49: B please"},
 {"id":"IC_new","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-08-30T08:11:12Z","body":"D50 - approve / D49 - A"},
 {"id":"IC_self","author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-08-30T09:00:00Z","body":"iteration report"}
]}
EOF
}

# ids the read presented as NEW (not labelled already-processed), in output order.
new_ids() { grep -E '^[^ ]+ @ [^ ]+ id=[^ ]+:$' | sed -E 's/^.* id=([^ ]+):$/\1/'; }
done_ids() { grep -F 'ALREADY PROCESSED' | sed -E 's/^.* id=([^ ]+) .*$/\1/'; }

# A simulated Gate-0 fire under the NEW protocol. $1 = kill point:
#   none | after_fetch | after_route_<n> | after_mark_<n>
# The ledger check before routing is the skill's crash-after-route reconcile: a NEW id the ledger
# already cites was routed by a fire that died before marking, so mark it and route nothing.
fire() {
	local kill="$1" wm="$2" ledger="$3" ids id n=0
	ids="$(md --issue 5 --watermark-file "$wm" 2>/dev/null | new_ids)"
	[ "$kill" = after_fetch ] && return 0
	for id in $ids; do
		n=$((n+1))
		grep -qx "$id" "$ledger" 2>/dev/null || echo "$id" >> "$ledger"
		[ "$kill" = "after_route_$n" ] && return 0
		md --issue 5 --watermark-file "$wm" --mark "$id" >/dev/null 2>&1 || echo "MARK-FAILED $id" >> "$ledger"
		[ "$kill" = "after_mark_$n" ] && return 0
	done
	return 0
}

# ── arm 1: boundary — a comment AT the watermark instant is returned (#852 literal) ──────────
arm_boundary() {
	local d="$SCRATCH/a1" out
	mkdir -p "$d"
	cat > "$d/852.json" <<'EOF'
{"comments": [
 {"id":"IC_kwDOP3Mz2c8AAAABRYdHpg","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-08-29T09:06:45Z","body":"D-49: A"},
 {"id":"IC_kwDOP3Mz2c8AAAABReRotQ","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-08-30T08:11:12Z","body":"D50 - approve / D49 - A"}
]}
EOF
	out="$(MD_FIXDIR="$d" md --issue 852 --since 2026-08-30T08:11:12Z 2>/dev/null)"
	if printf '%s\n' "$out" | grep -q 'D50 - approve' && printf '%s\n' "$out" | grep -q 'id=IC_kwDOP3Mz2c8AAAABReRotQ'; then
		ok "1 boundary: comment at exactly the watermark is returned, with its id"
	else
		notok "1 boundary: comment at exactly the watermark was skipped — got: $out"
	fi
	if printf '%s\n' "$out" | grep -q 'D-49: A$'; then notok "1 boundary: comment BEFORE the watermark leaked"; else ok "1 boundary: older comment stays below the watermark"; fi
}

# ── arm 2: kill at every step — next clean fire loses nothing and routes nothing twice ──────
arm_kill_points() {
	local kp d wm ledger c_old c_new
	for kp in after_fetch after_route_1 mid_mark_1 after_mark_1 after_route_2 after_mark_2 none; do
		d="$SCRATCH/a2-$kp"; two_directives "$d"
		wm="$d/wm"; ledger="$d/ledger"; : > "$ledger"
		export MD_FIXDIR="$d"
		if [ "$kp" = mid_mark_1 ]; then
			# Died inside --mark: the write is tmp+rename, so a death before the rename leaves the
			# old file intact plus a stray tmp. Reproduce exactly that state.
			fire after_route_1 "$wm" "$ledger"
			echo "garbage-from-a-dead-writer" > "$wm.tmp.dead01"
		else
			fire "$kp" "$wm" "$ledger"
		fi
		fire none "$wm" "$ledger"        # the next fire, clean
		fire none "$wm" "$ledger"        # and one more: steady state routes nothing new
		c_old=$(grep -cx IC_old "$ledger"); c_new=$(grep -cx IC_new "$ledger")
		if [ "$c_old" = 1 ] && [ "$c_new" = 1 ] && ! grep -q 'MARK-FAILED\|IC_pub\|IC_self' "$ledger"; then
			ok "2 kill@$kp: both directives routed exactly once"
		else
			notok "2 kill@$kp: IC_old routed $c_old, IC_new routed $c_new — ledger: $(tr '\n' ' ' < "$ledger")"
		fi
		unset MD_FIXDIR
	done
}

# ── arm 3: out-of-order mark and late arrival never jump an unprocessed directive ───────────
arm_out_of_order() {
	local d="$SCRATCH/a3" wm out
	two_directives "$d"; wm="$d/wm"; export MD_FIXDIR="$d"
	md --issue 5 --watermark-file "$wm" --mark IC_new >/dev/null 2>&1   # newer one first, then "die"
	out="$(md --issue 5 --watermark-file "$wm" 2>/dev/null | new_ids | tr '\n' ' ')"
	if [ "$out" = "IC_old " ]; then ok "3 out-of-order: older unmarked directive still presented"; else notok "3 out-of-order: expected 'IC_old', got '$out'"; fi

	d="$SCRATCH/a3b"; mkdir -p "$d"; wm="$d/wm"; export MD_FIXDIR="$d"
	cat > "$d/5.json" <<'EOF'
{"comments": [
 {"id":"IC_a","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-01T00:00:00Z","body":"first"}
]}
EOF
	md --issue 5 --watermark-file "$wm" >/dev/null 2>&1                # this fire reads IC_a only
	cat > "$d/5.json" <<'EOF'
{"comments": [
 {"id":"IC_a","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-01T00:00:00Z","body":"first"},
 {"id":"IC_late","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-01T00:05:00Z","body":"arrived mid-iteration"},
 {"id":"IC_late2","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-01T00:06:00Z","body":"and a follow-up"}
]}
EOF
	md --issue 5 --watermark-file "$wm" --mark IC_a >/dev/null 2>&1      # marks what it routed
	out="$(md --issue 5 --watermark-file "$wm" 2>/dev/null | new_ids | tr '\n' ' ')"
	if [ "$out" = "IC_late IC_late2 " ]; then ok "3 late-arrival: directives posted mid-iteration are presented next fire"; else notok "3 late-arrival: expected 'IC_late IC_late2', got '$out'"; fi
	unset MD_FIXDIR
}

# ── arm 4: dedupe — a processed id re-read at the inclusive boundary is context only ────────
arm_dedupe() {
	local d="$SCRATCH/a4" wm out err
	two_directives "$d"; wm="$d/wm"; export MD_FIXDIR="$d"
	md --issue 5 --watermark-file "$wm" --mark IC_old >/dev/null 2>&1
	md --issue 5 --watermark-file "$wm" --mark IC_new >/dev/null 2>&1
	out="$(md --issue 5 --watermark-file "$wm" 2>"$d/err")"; err="$(cat "$d/err")"
	if [ -z "$(printf '%s\n' "$out" | new_ids)" ] && [ "$(printf '%s\n' "$out" | done_ids)" = IC_new ]; then
		ok "4 dedupe: boundary re-read is labelled ALREADY PROCESSED, nothing new"
	else
		notok "4 dedupe: expected only IC_new as processed — got: $out"
	fi
	if printf '%s' "$err" | grep -q '0 new'; then ok "4 dedupe: summary counts 0 new"; else notok "4 dedupe: summary did not say 0 new — $err"; fi
	unset MD_FIXDIR
}

# ── arm 5: legacy bare-timestamp file — read, warned, upgraded with line 1 still bare ──────
arm_legacy() {
	local d="$SCRATCH/a5" wm out err l1
	two_directives "$d"; wm="$d/wm"; export MD_FIXDIR="$d"
	printf '2026-08-30T08:11:12Z\n' > "$wm"
	out="$(md --issue 5 --watermark-file "$wm" 2>"$d/err")"; err="$(cat "$d/err")"
	if [ "$(printf '%s\n' "$out" | new_ids | tr '\n' ' ')" = "IC_new " ]; then ok "5 legacy: #981 boundary comment re-presented as NEW"; else notok "5 legacy: expected IC_new, got: $out"; fi
	if printf '%s' "$err" | grep -q 'LEGACY' && printf '%s' "$err" | grep -q 'BOUNDARY'; then ok "5 legacy: LEGACY and BOUNDARY warnings on stderr"; else notok "5 legacy: missing warnings — $err"; fi
	md --issue 5 --watermark-file "$wm" --mark IC_new >/dev/null 2>&1
	l1="$(head -1 "$wm")"
	if printf '%s\n' "$l1" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' && grep -q '^processed IC_new ' "$wm"; then
		ok "5 legacy: upgraded file keeps a bare timestamp on line 1 (mission_gate0_self_notices.sh reads line 1)"
	else
		notok "5 legacy: bad upgraded file — $(tr '\n' '|' < "$wm")"
	fi
	# The other reader of this file is Gate 0 step 6a. Its watermark parse runs before any gh call;
	# a failing --gh-bin then stops it at a floor, after it has printed its verdict on the file.
	printf '#!/bin/bash\nexit 1\n' > "$d/nogh"; chmod +x "$d/nogh"
	out="$(/bin/bash scripts/mission_gate0_self_notices.sh --issue 5 --repo test/repo --self sunholo-voight-kampff \
		--watermark-file "$wm" --control 5:0 --gh-bin "$d/nogh" 2>/dev/null)"
	if printf '%s\n' "$out" | grep -qF "watermark: $wm = $l1 ok"; then
		ok "5 legacy: mission_gate0_self_notices.sh accepts the upgraded file"
	else
		notok "5 legacy: mission_gate0_self_notices.sh rejected the upgraded file"
	fi
	unset MD_FIXDIR
}

# ── arm 6: two missions, one script, no cross-talk ──────────────────────────────────────────
arm_multi_mission() {
	local d="$SCRATCH/a6" a b
	mkdir -p "$d"; export MD_FIXDIR="$d"
	cat > "$d/7.json" <<'EOF'
{"comments": [{"id":"IC_v1","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-02T00:00:00Z","body":"v1 answer"}]}
EOF
	cat > "$d/8.json" <<'EOF'
{"comments": [{"id":"IC_w","author":{"login":"world-human"},"createdAt":"2026-09-02T00:00:00Z","body":"world answer"},
              {"id":"IC_wm","author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-02T00:00:01Z","body":"not a world principal"}]}
EOF
	md --issue 7 --watermark-file "$d/wm-7" --mark IC_v1 >/dev/null 2>&1
	MISSION_DIRECTIVE_AUTHORS=world-human md --issue 8 --watermark-file "$d/wm-8" --mark IC_w >/dev/null 2>&1
	a="$(md --issue 7 --watermark-file "$d/wm-7" 2>/dev/null | done_ids)"
	b="$(MISSION_DIRECTIVE_AUTHORS=world-human md --issue 8 --watermark-file "$d/wm-8" 2>/dev/null | done_ids)"
	if [ "$a" = IC_v1 ] && [ "$b" = IC_w ] && ! grep -q IC_w "$d/wm-7" && ! grep -q IC_v1 "$d/wm-8" && ! grep -q IC_wm "$d/wm-8"; then
		ok "6 multi-mission: per-mission files and allowlists stay separate"
	else
		notok "6 multi-mission: cross-talk — wm-7: $(tr '\n' '|' < "$d/wm-7") wm-8: $(tr '\n' '|' < "$d/wm-8")"
	fi
	unset MD_FIXDIR
}

# ── arm 7: refusals leave the watermark byte-identical ──────────────────────────────────────
arm_refusals() {
	local d="$SCRATCH/a7" wm rc
	two_directives "$d"; wm="$d/wm"; export MD_FIXDIR="$d"
	md --issue 5 --watermark-file "$wm" --mark IC_old >/dev/null 2>&1
	cp "$wm" "$d/before"
	md --issue 5 --mark IC_new >/dev/null 2>&1; rc=$?
	if [ "$rc" = 1 ]; then ok "7 refuse: --mark without --watermark-file"; else notok "7 refuse: --mark without --watermark-file rc=$rc"; fi
	md --issue 5 --watermark-file "$wm" --mark IC_nope >/dev/null 2>&1; rc=$?
	if [ "$rc" = 1 ] && cmp -s "$wm" "$d/before"; then ok "7 refuse: unknown id, file unchanged"; else notok "7 refuse: unknown id rc=$rc or file changed"; fi
	md --issue 5 --watermark-file "$wm" --mark IC_pub >/dev/null 2>&1; rc=$?
	if [ "$rc" = 1 ] && cmp -s "$wm" "$d/before"; then ok "7 refuse: non-allowlisted id, file unchanged"; else notok "7 refuse: non-allowlisted id rc=$rc or file changed"; fi
	md --issue 5 --watermark-file "$wm" --since 2026-01-01T00:00:00Z >/dev/null 2>&1; rc=$?
	if [ "$rc" = 1 ]; then ok "7 refuse: --since with --watermark-file"; else notok "7 refuse: --since with --watermark-file rc=$rc"; fi
	md --issue 5 --watermark-file "$wm" >/dev/null 2>&1
	if cmp -s "$wm" "$d/before"; then ok "7 read-only: a plain read never writes the watermark"; else notok "7 read-only: a plain read changed the watermark"; fi
	unset MD_FIXDIR
}

# ── arm 8: CONTROL — the old protocol, driven the same way, does lose a directive ───────────
# Old Gate 0: read --since <line 1>, write the newest createdAt, THEN route. Kill before routing.
# If this arm ever stops seeing a loss, the harness above has gone blind, not the bug away.
arm_control_old_protocol() {
	local d="$SCRATCH/a8" wm ledger out newest
	two_directives "$d"; wm="$d/wm"; ledger="$d/ledger"; : > "$ledger"; export MD_FIXDIR="$d"
	out="$(md --issue 5 --since 1970-01-01T00:00:00Z 2>/dev/null)"
	newest="$(printf '%s\n' "$out" | grep -E '^[^ ]+ @ ' | sed -E 's/^[^ ]+ @ ([^ :]+Z).*$/\1/' | sort | tail -1)"
	printf '%s\n' "$newest" > "$wm"                         # mark-before-route ... then the fire dies
	out="$(md --issue 5 --since "$(head -1 "$wm")" 2>/dev/null)"
	if printf '%s\n' "$out" | grep -q 'D-49: A$'; then
		notok "8 control: old mark-before-route did NOT lose IC_old — the suite cannot see the #981 failure"
	else
		ok "8 control: old mark-before-route loses IC_old (the #981 failure is visible to this harness)"
	fi
	unset MD_FIXDIR
}

arm_boundary
arm_kill_points
arm_out_of_order
arm_dedupe
arm_legacy
arm_multi_mission
arm_refusals
arm_control_old_protocol

echo "mission_directives: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
