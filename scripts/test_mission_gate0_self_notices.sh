#!/usr/bin/env bash
# test_mission_gate0_self_notices.sh — self-contained 28-arm suite for scripts/mission_gate0_self_notices.sh.
#
# Every arm is SELF-CONTAINED: it builds its own fixture (heredoc only), its own `--gh-bin`
# stub, and its own scratch. No helper synthesises a comment, an author or a count; the only
# fixture builder is make_json reading literal lines from a heredoc. Every instrument
# invocation is an explicit `/bin/bash "$SCRIPT_UT" ...` — never the login shell (zsh),
# never `sh`. Fixture issues are numbered 5 (and 6 where two are needed), never 107/129, so no
# synthetic arm can accidentally satisfy a snapshot assertion.
#
# Row-90 property: each arm greps only the fields its behaviour owns. Classification arms
# (1-5) grep `crash:` lines by `at=` + one token; floor arms (7-14, 21, 22, 23, 24) their own
# message, rc, argv-log emptiness where stated; `report` alone greps `verdict:`; snapshot arms
# a full `issue:` line and `control:`; watermark arms (19-22) `watermark:`/`since:` only.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
SCRIPT_UT="${G0_SCRIPT_UT:-scripts/mission_gate0_self_notices.sh}"
FIXDIR=scripts/mission_gate0_self_notices_testdata
pass=0; fail=0
ok()   { echo "ok $*";      pass=$((pass+1)); }
notok(){ echo "not ok $*";  fail=$((fail+1)); }

SIG="⚠️ Mission iteration **FAILED to complete** (rc="

SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/g0_scratch.XXXXXX")"
trap 'chmod -R u+rw "$SCRATCH" 2>/dev/null; rm -rf "$SCRATCH"' EXIT

make_json() { cat > "$1"; }

# make_stub_gh <out> <fixture-dir> <argv-log>
make_stub_gh() {
  local out="$1" dir="$2" log="$3"
  {
    echo '#!/usr/bin/env bash'
    echo 'set -u'
    printf 'printf "%%s\\n" "$*" >> "%s"\n' "$log"
    printf 'dir="%s"\n' "$dir"
    echo 'case "$1 $2" in'
    echo '  "issue view") f="$dir/$3.json" ;;'
    echo '  "api repos/"*) f="$dir/${2##*/}.meta.json" ;;'
    echo '  *) echo "stub gh: no fixture" >&2; exit 1 ;;'
    echo 'esac'
    echo '[ -f "$f" ] || { echo "stub gh: no fixture" >&2; exit 1; }'
    echo 'cat "$f"'
  } > "$out"
  chmod +x "$out"
}

# run_instr <out> <err> [args...] — runs the instrument under /bin/bash
run_instr() {
  local out="$1" err="$2"; shift 2
  /bin/bash "$SCRIPT_UT" "$@" > "$out" 2> "$err"
}

# ── arm 1: anchored-signature (AC-M1-2) ────────────────────────────────────────
arm_anchored_signature() {
  local d out err rc
  d="$SCRATCH/a1"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"${SIG}143 — timeout or crash) at 2026-09-02 14:02 CEST. Log on the rig: \`/tmp/ailang-mission-world.log\`. The queue is untouched; the next interval will retry."},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#issuecomment-2","body":"${SIG}1 — timeout or crash) at 2026-09-07 04:47 CEST. Log on the rig: \`/tmp/ailang-mission-world.log\`. The queue is untouched; the next interval will retry. Further identical failures are silent until the rc changes or an iteration completes."},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","url":"https://x/5#issuecomment-3","body":" ${SIG}143 — not anchored"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-04T00:00:00Z","url":"https://x/5#issuecomment-4","body":"some prose that is quite long, about forty characters, ${SIG}143 — quoted"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-05T00:00:00Z","url":"https://x/5#issuecomment-5","body":"${SIG}abc — non-digit rc"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 5}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:2 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 1 ]; then ok "anchored-signature exits 1"; else notok "anchored-signature rc=$rc"; fi
  local c; c="$(/usr/bin/grep -c '^crash:' "$d/out")"
  if [ "$c" -eq 2 ]; then ok "anchored-signature two crash lines"; else notok "anchored-signature crash count=$c"; fi
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-01T00:00:00Z' "$d/out" && ok "anchored-signature has 09-01 (i)" || notok "anchored-signature missing 09-01"
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-02T00:00:00Z' "$d/out" && ok "anchored-signature has 09-02 (ii)" || notok "anchored-signature missing 09-02"
  /usr/bin/grep -q ': 2026-09-03T00:00:00Z\|at=2026-09-03T00:00:00Z' "$d/out" && notok "anchored-signature classified (iii)" || ok "anchored-signature no (iii)"
  /usr/bin/grep -q 'at=2026-09-04T00:00:00Z' "$d/out" && notok "anchored-signature classified (iv)" || ok "anchored-signature no (iv)"
  /usr/bin/grep -q 'at=2026-09-05T00:00:00Z' "$d/out" && notok "anchored-signature classified (v)" || ok "anchored-signature no (v)"
}

# ── arm 2: rc-extract (AC-M1-3) ───────────────────────────────────────────────
arm_rc_extract() {
  local d out err rc
  d="$SCRATCH/a2"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"${SIG}143 — timeout or crash) at T1"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#issuecomment-2","body":"${SIG}1 — timeout or crash) at T2"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 2}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:2 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 1 ]; then ok "rc-extract exits 1"; else notok "rc-extract rc=$rc"; fi
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-01T00:00:00Z rc=143 ' "$d/out" && ok "rc-extract T1 rc=143" || notok "rc-extract missing T1 rc=143"
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-02T00:00:00Z rc=1 ' "$d/out" && ok "rc-extract T2 rc=1" || notok "rc-extract missing T2 rc=1"
  /usr/bin/grep -q 'at=2026-09-01T00:00:00Z rc=1 ' "$d/out" && notok "rc-extract T1 rc=1 present" || ok "rc-extract T1 rc=1 absent"
  /usr/bin/grep -q 'at=2026-09-02T00:00:00Z rc=143 ' "$d/out" && notok "rc-extract T2 rc=143 present" || ok "rc-extract T2 rc=143 absent"
}

# ── arm 3: since-window (AC-M1-4) ─────────────────────────────────────────────
arm_since_window() {
  local d out err arun brun rc
  d="$SCRATCH/a3"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"${SIG}143 — T1"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#issuecomment-2","body":"${SIG}143 — T2"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","url":"https://x/5#issuecomment-3","body":"${SIG}143 — T3"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 3}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/outA" "$d/errA" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-02T00:00:00Z --control 5:3 --gh-bin "$d/stub"; arun=$?
  run_instr "$d/outB" "$d/errB" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-03T00:00:00Z --control 5:3 --gh-bin "$d/stub"; brun=$?
  if [ "$arun" -eq 1 ] && [ "$brun" -eq 0 ]; then ok "since-window exits A1 B0"; else notok "since-window rc A=$arun B=$brun"; fi
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-01T00:00:00Z .*in-window=no' "$d/outA" && ok "since-window A T1 no" || notok "since-window A T1 no missing"
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-02T00:00:00Z .*in-window=no' "$d/outA" && ok "since-window A T2 no" || notok "since-window A T2 no missing"
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-03T00:00:00Z .*in-window=yes' "$d/outA" && ok "since-window A T3 yes" || notok "since-window A T3 yes missing"
  /usr/bin/grep -q 'in-window=yes' "$d/outB" && notok "since-window B has in-window=yes" || ok "since-window B all no"
  local ca cb; ca="$(/usr/bin/grep -c '^crash:' "$d/outA")"; cb="$(/usr/bin/grep -c '^crash:' "$d/outB")"
  if [ "$ca" -eq 3 ] && [ "$cb" -eq 3 ]; then ok "since-window crash count 3 both"; else notok "since-window crash count A=$ca B=$cb"; fi
}

# ── arm 4: self-filter (AC-M1-5) ───────────────────────────────────────────────
arm_self_filter() {
  local d out err rc c
  d="$SCRATCH/a4"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"SUNHOLO-VOIGHT-KAMPFF"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"${SIG}143 — Ta"},
 {"author":{"login":"MarkEdmondson1234"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#issuecomment-2","body":"${SIG}143 — Tb"},
 {"author":{"login":"stranger-42"},"createdAt":"2026-09-03T00:00:00Z","url":"https://x/5#issuecomment-3","body":"${SIG}143 — Tc"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 3}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:1 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 1 ]; then ok "self-filter exits 1"; else notok "self-filter rc=$rc"; fi
  c="$(/usr/bin/grep -c '^crash:' "$d/out")"
  if [ "$c" -eq 1 ]; then ok "self-filter one crash line"; else notok "self-filter crash count=$c"; fi
  /usr/bin/grep -q '^crash: issue=5 at=2026-09-01T00:00:00Z' "$d/out" && ok "self-filter carries Ta" || notok "self-filter missing Ta"
  /usr/bin/grep -q 'at=2026-09-02T00:00:00Z\|at=2026-09-03T00:00:00Z' "$d/out" && notok "self-filter classified foreign author" || ok "self-filter no foreign author"
  # D3 arithmetic on the ONE fixture where comments != self: self= counts the --self author only,
  # other= is self - crash (never comments - crash). Added at evaluator round 1 (B1): the
  # `other=$((COMMENTS-CRASH))` mutant survived all 86 assertions because every other fixture has
  # comments == self. Two tokens this arm's behaviour sets; not a full issue: line (row 90).
  /usr/bin/grep '^issue: 5 ' "$d/out" | /usr/bin/grep -q ' self=1 ' && ok "self-filter self=1 token" || notok "self-filter self= token wrong: $(/usr/bin/grep '^issue: 5 ' "$d/out")"
  /usr/bin/grep '^issue: 5 ' "$d/out" | /usr/bin/grep -q ' other=0$' && ok "self-filter other=0 token (self-crash, not comments-crash)" || notok "self-filter other= token wrong: $(/usr/bin/grep '^issue: 5 ' "$d/out")"
}

# ── arm 5: prev-issue (AC-M1-6) ───────────────────────────────────────────────
arm_prev_issue() {
  local d outA errA outB errB arun brun crun drun n
  d="$SCRATCH/a5"; mkdir -p "$d"
  make_json "$d/129.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-07T00:00:00Z","url":"https://x/129#issuecomment-1","body":"plain report, not a notice"}
]}
EOF
  make_json "$d/129.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_json "$d/107.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-07T02:47:28Z","url":"https://x/107#issuecomment-1","body":"${SIG}1 — timeout or crash) at the kill"}
]}
EOF
  make_json "$d/107.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/outA" "$d/errA" --issue 129 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-07T00:00:00Z --control 107:1 --gh-bin "$d/stub"; arun=$?
  run_instr "$d/outB" "$d/errB" --issue 129 --prev-issue 107 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-07T00:00:00Z --control 107:1 --gh-bin "$d/stub"; brun=$?
  if [ "$arun" -eq 0 ]; then ok "prev-issue run A miss rc=0"; else notok "prev-issue run A rc=$arun"; fi
  n="$(/usr/bin/grep -c '^crash: issue=107' "$d/outA")"
  if [ "$n" -eq 0 ]; then ok "prev-issue run A no 107 crash"; else notok "prev-issue run A leaked 107 ($n)"; fi
  if [ "$brun" -eq 1 ]; then ok "prev-issue run B hit rc=1"; else notok "prev-issue run B rc=$brun"; fi
  n="$(/usr/bin/grep -c '^crash: issue=107 at=2026-09-07T02:47:28Z rc=1 in-window=yes' "$d/outB")"
  if [ "$n" -eq 1 ]; then ok "prev-issue run B exactly one line"; else notok "prev-issue run B line count=$n"; fi
  # Run C: prev is NOT the control issue, so only the prev read can surface 107's notice
  # (dropping $PREV_ISSUE from the read loop turns rc 1 into rc 0 — judge mutant m3).
  run_instr "$d/outC" "$d/errC" --issue 129 --prev-issue 107 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-07T00:00:00Z --control 129:0 --gh-bin "$d/stub"; crun=$?
  if [ "$crun" -eq 1 ]; then ok "prev-issue run C (prev != control) hit rc=1"; else notok "prev-issue run C rc=$crun"; fi
  n="$(/usr/bin/grep -c '^crash: issue=107 at=2026-09-07T02:47:28Z rc=1 in-window=yes' "$d/outC")"
  if [ "$n" -eq 1 ]; then ok "prev-issue run C exactly one 107 line"; else notok "prev-issue run C line count=$n"; fi
  # Run D: an unreadable prev issue is an instrument failure (rc 2), never a silent skip
  run_instr "$d/outD" "$d/errD" --issue 129 --prev-issue 999 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-07T00:00:00Z --control 129:0 --gh-bin "$d/stub"; drun=$?
  if [ "$drun" -eq 2 ] && [ -s "$d/errD" ]; then ok "prev-issue run D unreadable prev rc=2 named floor"; else notok "prev-issue run D rc=$drun"; fi
}

# ── arm 6: no-bodies (AC-M1-7) ───────────────────────────────────────────────
arm_no_bodies() {
  local d outc rc sn
  d="$SCRATCH/a6"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"${SIG}143 — timeout or crash) tail SENTINEL-DO-NOT-PRINT-7f3a"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#issuecomment-2","body":"SENTINEL-DO-NOT-PRINT-7f3a"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 2}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:1 --gh-bin "$d/stub"; rc=$?
  cat "$d/out" "$d/err" > "$d/combined"
  sn="$(/usr/bin/grep -c 'SENTINEL-DO-NOT-PRINT-7f3a' "$d/combined")"
  if [ "$sn" -eq 0 ]; then ok "no-bodies sentinel absent"; else notok "no-bodies sentinel count=$sn"; fi
  /usr/bin/grep -q 'url=https://x/5#issuecomment-1' "$d/out" && ok "no-bodies url present" || notok "no-bodies url missing"
}

# ── arm 7: usage (AC-M1-8) ─────────────────────────────────────────────────────
arm_usage() {
  local d out err rc
  d="$SCRATCH/a7"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#issuecomment-1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  # (a) neither --since nor --watermark-file
  run_instr "$d/oa" "$d/ea" --issue 5 --repo test/test --self sunholo-voight-kampff --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ usage: --issue <digits>, --repo, --self and exactly one of --since | --watermark-file are required' "$d/ea" && [ "$rc" -eq 2 ] && ok "usage (a) neither" || notok "usage (a) rc=$rc"
  # (b) both
  printf '2026-09-08T06:19:00Z' > "$d/wm"
  run_instr "$d/ob" "$d/eb" --issue 5 --repo test/test --self sunholo-voight-kampff --since 2026-09-08T06:19:00Z --watermark-file "$d/wm" --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ usage: --issue <digits>, --repo, --self and exactly one of --since | --watermark-file are required' "$d/eb" && [ "$rc" -eq 2 ] && ok "usage (b) both" || notok "usage (b) rc=$rc"
  # (c) empty --issue
  run_instr "$d/oc" "$d/ec" --issue "" --repo test/test --self sunholo-voight-kampff --since 2026-09-08T06:19:00Z --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ usage: --issue <digits>, --repo, --self and exactly one of --since | --watermark-file are required' "$d/ec" && [ "$rc" -eq 2 ] && ok "usage (c) empty issue" || notok "usage (c) rc=$rc"
  # (d) passed-but-empty --prev-issue is F0, never a silent single-issue read (planner finding)
  : > "$d/argvd.log"; make_stub_gh "$d/stubd" "$d" "$d/argvd.log"
  run_instr "$d/od" "$d/ed" --issue 5 --prev-issue "" --repo test/test --self sunholo-voight-kampff --since 2026-09-08T06:19:00Z --control 5:0 --gh-bin "$d/stubd"; rc=$?
  /usr/bin/grep -q '✗ usage: ' "$d/ed" && [ "$rc" -eq 2 ] && ok "usage (d) empty prev-issue" || notok "usage (d) rc=$rc"
  if [ ! -s "$d/argvd.log" ]; then ok "usage (d) empty argv"; else notok "usage (d) argv non-empty"; fi
}

# ── arm 8: gh-missing (AC-M1-8) ───────────────────────────────────────────────
arm_gh_missing() {
  local d out err rc
  d="$SCRATCH/a8"; mkdir -p "$d"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin /nonexistent/gh; rc=$?
  /usr/bin/grep -q '✗ gh binary unreachable: /nonexistent/gh' "$d/err" && [ "$rc" -eq 2 ] && ok "gh-missing unreachable rc2" || notok "gh-missing rc=$rc"
}

# ── arm 9: gh-timeout (AC-M1-8) ───────────────────────────────────────────────
arm_gh_timeout() {
  local d out err rc stub
  d="$SCRATCH/a9"; mkdir -p "$d"
  stub="$d/stub"
  printf '#!/usr/bin/env bash\n/bin/sleep 3\n' > "$stub"; chmod +x "$stub"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$stub" --gh-timeout 0.5; rc=$?
  /usr/bin/grep -q '✗ gh timed out after 0.5s for issue #5' "$d/err" && [ "$rc" -eq 2 ] && ok "gh-timeout timed out rc2" || notok "gh-timeout rc=$rc"
}

# ── arm 10: gh-failed (AC-M1-8) ───────────────────────────────────────────────
arm_gh_failed() {
  local d out err rc stub
  d="$SCRATCH/a10"; mkdir -p "$d"
  stub="$d/stub"
  printf '#!/usr/bin/env bash\nexit 7\n' > "$stub"; chmod +x "$stub"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$stub"; rc=$?
  /usr/bin/grep -q '✗ gh failed (rc=7) for issue #5' "$d/err" && [ "$rc" -eq 2 ] && ok "gh-failed rc7 rc2" || notok "gh-failed rc=$rc"
}

# ── arm 11: malformed (AC-M1-8) ────────────────────────────────────────────────
arm_malformed() {
  local d out err rc
  d="$SCRATCH/a11"; mkdir -p "$d"
  make_json "$d/5.json" <<'EOF'
{"nope": []}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 0}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -qE '^✗ malformed comments payload for issue #5: ' "$d/err" && [ "$rc" -eq 2 ] && ok "malformed F4 rc2" || notok "malformed rc=$rc"
}

# ── arm 12: truncated (AC-M1-8) ────────────────────────────────────────────────
arm_truncated() {
  local d out err rc
  d="$SCRATCH/a12"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"a"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#2","body":"b"}
]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 5}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ comment array truncated for issue #5: got 2 of 5' "$d/err" && [ "$rc" -eq 2 ] && ok "truncated F5 rc2" || notok "truncated rc=$rc"
}

# ── arm 13: control-required (AC-M1-9) ────────────────────────────────────────
arm_control_required() {
  local d out err rc
  d="$SCRATCH/a13"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ control required: --control <issue>:<count>' "$d/err" && [ "$rc" -eq 2 ] && ok "control-required F6 rc2" || notok "control-required rc=$rc"
}

# ── arm 14: control-mismatch (AC-M1-9) ───────────────────────────────────────
arm_control_mismatch() {
  local d out err rc c
  d="$SCRATCH/a14"; mkdir -p "$d"
  make_json "$d/107.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/107#1","body":"${SIG}143 — one"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/107#2","body":"${SIG}143 — two"}
]}
EOF
  make_json "$d/107.meta.json" <<'EOF'
{"comments": 2}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 107 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 107:4 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '✗ CONTROL MISMATCH: issue 107 expect=4 got=2' "$d/out" && [ "$rc" -eq 2 ] && ok "control-mismatch F7 rc2" || notok "control-mismatch rc=$rc"
  c="$(/usr/bin/grep -c '^crash:' "$d/out")"
  if [ "$c" -eq 2 ]; then ok "control-mismatch table first (2 crash)"; else notok "control-mismatch crash count=$c"; fi
}

# ── arm 15: signature-source (AC-M1-10) ─────────────────────────────────────────
arm_signature_source() {
  local d outA errA outB errB outC errC rc
  d="$SCRATCH/a15"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  cat > "$d/fileA" <<'EOF'
--body "⚠️ Mission iteration **FAILED to complete** (rc=$RC — timeout or crash) at ..."
EOF
  printf 'echo "no signature here"\n' > "$d/fileB"
  # run A: driver-src ok, reaches network, control 5:0 ok
  run_instr "$d/outA" "$d/errA" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --driver-src "$d/fileA" --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "signature-source: $d/fileA ok" "$d/outA" && [ "$rc" -eq 0 ] && ok "signature-source A ok" || notok "signature-source A rc=$rc"
  # run B: missing signature, fresh argv log, pre-network
  : > "$d/argvB.log"
  make_stub_gh "$d/stubB" "$d" "$d/argvB.log"
  run_instr "$d/outB" "$d/errB" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --driver-src "$d/fileB" --gh-bin "$d/stubB"; rc=$?
  /usr/bin/grep -q "✗ signature not found in driver source: $d/fileB" "$d/errB" && [ "$rc" -eq 2 ] && ok "signature-source B F8 rc2" || notok "signature-source B rc=$rc"
  if [ ! -s "$d/argvB.log" ]; then ok "signature-source B empty argv"; else notok "signature-source B argv non-empty"; fi
  # run C: no driver-src
  run_instr "$d/outC" "$d/errC" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q 'signature-source: unchecked' "$d/outC" && ok "signature-source C unchecked" || notok "signature-source C rc=$rc"
}

# ── arm 16: report (AC-M1-11) ─────────────────────────────────────────────────
arm_report() {
  local d outA errA outB errB outC errC rc
  d="$SCRATCH/a16"; mkdir -p "$d"
  make_json "$d/129.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/129#1","body":"plain"}]}
EOF
  make_json "$d/129.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_json "$d/107.json" <<EOF
{"comments": [
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/107#1","body":"${SIG}143 — n1"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T12:02:13Z","url":"https://x/107#2","body":"${SIG}143 — n2"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T15:21:07Z","url":"https://x/107#3","body":"${SIG}143 — n3"},
 {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T19:27:22Z","url":"https://x/107#4","body":"${SIG}143 — n4"}
]}
EOF
  make_json "$d/107.meta.json" <<'EOF'
{"comments": 4}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  A="--issue 129 --prev-issue 107 --repo test/test --self sunholo-voight-kampff"
  run_instr "$d/outA" "$d/errA" $A --since 2026-09-03T00:00:00Z --control 107:4 --gh-bin "$d/stub"; rc=$?
  tail -n 1 "$d/outA" > "$d/lastA"
  if [ "$rc" -eq 0 ] && /usr/bin/grep -E '^verdict: 0 crash notice\(s\) since 2026-09-03T00:00:00Z on #129,#107 — no death signal from the driver; Gate 2.s traces \(a\)-\(c\) still run \[mission_gate0_self_notices\.sh, control 107:4, watermarks literal\]$' "$d/lastA" >/dev/null; then
    ok "report A rc0 exact"; else notok "report A rc=$rc"; fi
  run_instr "$d/outB" "$d/errB" $A --since 2026-09-02T16:00:00Z --control 107:4 --gh-bin "$d/stub"; rc=$?
  tail -n 1 "$d/outB" > "$d/lastB"
  if [ "$rc" -eq 1 ] && /usr/bin/grep -E '^verdict: 1 crash notice\(s\) since 2026-09-02T16:00:00Z on #129,#107 — A FIRE DIED \(newest rc=143 at 2026-09-02T19:27:22Z on #107\): run Gate 2.s died-mid-flight traces \(a\) open PRs \(b\) worktrees \(c\) uncommitted state BEFORE picking, and credit the orphan in the log; "the queue is untouched" describes the charter, not the work \[mission_gate0_self_notices\.sh, control 107:4, watermarks literal\]$' "$d/lastB" >/dev/null; then
    ok "report B rc1 exact"; else notok "report B rc=$rc"; fi
  printf '2026-09-03T00:00:00Z' > "$d/wm1"; printf '2026-09-03T00:00:00Z' > "$d/wm2"
  run_instr "$d/outC" "$d/errC" $A --watermark-file "$d/wm1" --watermark-file "$d/wm2" --control 107:4 --gh-bin "$d/stub"; rc=$?
  tail -n 1 "$d/outC" > "$d/lastC"
  if [ "$rc" -eq 0 ] && /usr/bin/grep -E ', watermarks 2/2\]$' "$d/lastC" >/dev/null; then ok "report C watermarks 2/2"; else notok "report C rc=$rc"; fi
  if /usr/bin/grep -q 'untouched;\|will retry\|nothing happened' "$d/outB"; then notok "report B forbidden word"; else ok "report B no forbidden words"; fi
}

# ── arm 17: snapshot-107 (AC-M1-12/13) ─────────────────────────────────────────
arm_snapshot_107() {
  local d out err rc wc_lines
  d="$SCRATCH/a17"; mkdir -p "$d"
  cp $FIXDIR/107.json "$d/107.json"
  cp $FIXDIR/107.meta.json "$d/107.meta.json"
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 107 --repo sunholo-data/ailang-world --self sunholo-voight-kampff \
    --since 2026-09-08T00:00:00Z --control 107:4 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 0 ]; then ok "snapshot-107 rc0"; else notok "snapshot-107 rc=$rc"; fi
  local c; c="$(/usr/bin/grep -c '^crash:' "$d/out")"
  if [ "$c" -eq 4 ]; then ok "snapshot-107 four crash lines"; else notok "snapshot-107 crash count=$c"; fi
  /usr/bin/grep -q '^crash: issue=107 at=2026-09-02T12:02:13Z rc=143 in-window=no' "$d/out" && ok "snapshot-107 line 12:02" || notok "snapshot-107 missing 12:02"
  /usr/bin/grep -q '^crash: issue=107 at=2026-09-02T15:21:07Z rc=143 in-window=no' "$d/out" && ok "snapshot-107 line 15:21" || notok "snapshot-107 missing 15:21"
  /usr/bin/grep -q '^crash: issue=107 at=2026-09-02T19:27:22Z rc=143 in-window=no' "$d/out" && ok "snapshot-107 line 19:27" || notok "snapshot-107 missing 19:27"
  /usr/bin/grep -q '^crash: issue=107 at=2026-09-07T02:47:28Z rc=1 in-window=no' "$d/out" && ok "snapshot-107 line 02:47" || notok "snapshot-107 missing 02:47"
  /usr/bin/grep -q 'at=2026-09-03T10:57:25Z' "$d/out" && notok "snapshot-107 has 09-03 correction" || ok "snapshot-107 no 09-03 correction"
  /usr/bin/grep -q '^issue: 107 comments=43 self=43 crash=4 other=39' "$d/out" && ok "snapshot-107 issue line" || notok "snapshot-107 issue line missing"
  /usr/bin/grep -q '^control: issue=107 expect=4 got=4 ok' "$d/out" && ok "snapshot-107 control ok" || notok "snapshot-107 control missing"
  wc_lines="$(wc -l < "$d/argv.log")"
  if [ "$wc_lines" -eq 2 ]; then ok "snapshot-107 two stub calls"; else notok "snapshot-107 stub-call lines=$wc_lines"; fi
  /usr/bin/grep -Fxq 'issue view 107 --repo sunholo-data/ailang-world --json comments' "$d/argv.log" && ok "snapshot-107 argv issue-view exact" || notok "snapshot-107 argv issue-view"
  /usr/bin/grep -Fxq 'api repos/sunholo-data/ailang-world/issues/107' "$d/argv.log" && ok "snapshot-107 argv api-issue exact" || notok "snapshot-107 argv api-issue"
}

# ── arm 18: snapshot-129 (AC-M1-12) ──────────────────────────────────────────
arm_snapshot_129() {
  local d out err rc c
  d="$SCRATCH/a18"; mkdir -p "$d"
  cp $FIXDIR/129.json "$d/129.json"
  cp $FIXDIR/129.meta.json "$d/129.meta.json"
  cp $FIXDIR/107.json "$d/107.json"
  cp $FIXDIR/107.meta.json "$d/107.meta.json"
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 129 --prev-issue 107 --repo sunholo-data/ailang-world --self sunholo-voight-kampff \
    --since 2026-09-08T00:00:00Z --control 107:4 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 0 ]; then ok "snapshot-129 rc0"; else notok "snapshot-129 rc=$rc"; fi
  /usr/bin/grep -q '^issue: 129 comments=14 self=14 crash=0 other=14' "$d/out" && ok "snapshot-129 issue line" || notok "snapshot-129 issue line missing"
  c="$(/usr/bin/grep -c '^crash: issue=129' "$d/out")"
  if [ "$c" -eq 0 ]; then ok "snapshot-129 no 129 crash"; else notok "snapshot-129 129 crash=$c"; fi
}

# ── arm 19: watermark-older (AC-M1-15) ────────────────────────────────────────
arm_watermark_older() {
  local d out err rc
  d="$SCRATCH/a19"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  printf '2026-09-08T06:19:00Z' > "$d/wm_a"
  printf '2026-09-05T00:00:00Z' > "$d/wm_b"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_a" --watermark-file "$d/wm_b" --control 5:0 --gh-bin "$d/stub"; rc=$?
  [ "$rc" -eq 0 ] || { notok "watermark-older rc=$rc"; return; }
  /usr/bin/grep -q "watermark: $d/wm_a = 2026-09-08T06:19:00Z ok" "$d/out" && ok "watermark-older wm_a ok" || notok "watermark-older wm_a"
  /usr/bin/grep -q "watermark: $d/wm_b = 2026-09-05T00:00:00Z ok" "$d/out" && ok "watermark-older wm_b ok" || notok "watermark-older wm_b"
  /usr/bin/grep -q '^since: 2026-09-05T00:00:00Z (older of 2 readable watermark(s) of 2)' "$d/out" && ok "watermark-older since" || notok "watermark-older since"
}

# ── arm 20: watermark-degraded (AC-M1-15) ─────────────────────────────────────
arm_watermark_degraded() {
  local d out err rc
  d="$SCRATCH/a20"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  printf '2026-09-08T06:19:00Z' > "$d/wm_a"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_a" --watermark-file "$d/wm_absent" --control 5:0 --gh-bin "$d/stub"; rc=$?
  [ "$rc" -eq 0 ] || { notok "watermark-degraded rc=$rc"; return; }
  /usr/bin/grep -q "watermark: $d/wm_absent ABSENT — DEGRADED: single-watermark read (iter-52 rule wants two)" "$d/out" && ok "watermark-degraded ABSENT" || notok "watermark-degraded ABSENT line"
  /usr/bin/grep -q '^since: 2026-09-08T06:19:00Z (older of 1 readable watermark(s) of 2)' "$d/out" && ok "watermark-degraded since" || notok "watermark-degraded since"
}

# ── arm 21: watermark-unavailable (AC-M1-15) ──────────────────────────────────
arm_watermark_unavailable() {
  local d out err rc
  d="$SCRATCH/a21"; mkdir -p "$d"
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/p1" --watermark-file "$d/p2" --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "✗ watermark channel unavailable: no readable watermark among 2 file(s): $d/p1 $d/p2" "$d/err" && [ "$rc" -eq 2 ] && ok "watermark-unavailable F9 rc2" || notok "watermark-unavailable rc=$rc"
  if [ ! -s "$d/argv.log" ]; then ok "watermark-unavailable empty argv"; else notok "watermark-unavailable argv non-empty"; fi
}

# ── arm 22: watermark-invalid (AC-M1-15) ──────────────────────────────────────
arm_watermark_invalid() {
  local d out err rc
  d="$SCRATCH/a22"; mkdir -p "$d"
  # (a) empty
  : > "$d/wm_empty"
  run_instr "$d/oa" "$d/ea" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_empty" --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "✗ watermark invalid: $d/wm_empty (empty)" "$d/ea" && [ "$rc" -eq 2 ] && ok "watermark-invalid empty" || notok "watermark-invalid empty rc=$rc"
  # (b) bad beside ok — malformed not outvoted
  printf '2026-09-08T06:19:00Z' > "$d/wm_a"
  printf 'not-a-date' > "$d/wm_bad"
  run_instr "$d/ob" "$d/eb" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_a" --watermark-file "$d/wm_bad" --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "✗ watermark invalid: $d/wm_bad (not-ISO8601 'not-a-date')" "$d/eb" && [ "$rc" -eq 2 ] && ok "watermark-invalid not-ISO8601" || notok "watermark-invalid not-ISO8601 rc=$rc"
  # (c) unreadable
  printf '2026-09-08T06:19:00Z' > "$d/wm_unreadable"
  chmod 000 "$d/wm_unreadable"
  if [ ! -r "$d/wm_unreadable" ]; then
    run_instr "$d/oc" "$d/ec" --issue 5 --repo test/test --self sunholo-voight-kampff \
      --watermark-file "$d/wm_unreadable" --control 5:0 --gh-bin "$d/stub"; rc=$?
    /usr/bin/grep -q "✗ watermark invalid: $d/wm_unreadable (unreadable)" "$d/ec" && [ "$rc" -eq 2 ] && ok "watermark-invalid unreadable" || notok "watermark-invalid unreadable rc=$rc"
  else
    notok "watermark-invalid precondition: still readable — running as root?"
  fi
  chmod u+w "$d/wm_unreadable" 2>/dev/null
  # (d) --since not-a-date
  run_instr "$d/od" "$d/ed" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since not-a-date --control 5:0 --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "✗ watermark invalid: --since (not-ISO8601 'not-a-date')" "$d/ed" && [ "$rc" -eq 2 ] && ok "watermark-invalid --since" || notok "watermark-invalid --since rc=$rc"
}

# ── arm 23: watermark-strict (AC-M1-16) ───────────────────────────────────────
arm_watermark_strict() {
  local d out err rc now60 val
  d="$SCRATCH/a23"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  # run 1: Feb-30 (prefix, P4)
  printf '2026-02-30T00:00:00Z' > "$d/wm_feb"
  : > "$d/ag1.log"; make_stub_gh "$d/stub1" "$d" "$d/ag1.log"
  run_instr "$d/o1" "$d/e1" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_feb" --control 5:0 --gh-bin "$d/stub1"; rc=$?
  /usr/bin/grep -q "✗ watermark invalid: $d/wm_feb (not-canonical '2026-02-30T00:00:00Z'" "$d/e1" && [ "$rc" -eq 2 ] && ok "watermark-strict feb30 prefix rc2" || notok "watermark-strict feb30 rc=$rc"
  [ -s "$d/ag1.log" ] && notok "watermark-strict feb30 argv non-empty" || ok "watermark-strict feb30 empty argv"
  # run 2: 99-99
  printf '2026-99-99T99:99:99Z' > "$d/wm_g"
  : > "$d/ag2.log"; make_stub_gh "$d/stub2" "$d" "$d/ag2.log"
  run_instr "$d/o2" "$d/e2" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_g" --control 5:0 --gh-bin "$d/stub2"; rc=$?
  /usr/bin/grep -q "(not-canonical '2026-99-99T99:99:99Z')" "$d/e2" && [ "$rc" -eq 2 ] && ok "watermark-strict 99-99 rc2" || notok "watermark-strict 99-99 rc=$rc"
  [ -s "$d/ag2.log" ] && notok "watermark-strict 99-99 argv non-empty" || ok "watermark-strict 99-99 empty argv"
  # run 3: far-future 9999
  printf '9999-01-01T00:00:00Z' > "$d/wm_9"
  : > "$d/ag3.log"; make_stub_gh "$d/stub3" "$d" "$d/ag3.log"
  run_instr "$d/o3" "$d/e3" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_9" --control 5:0 --gh-bin "$d/stub3"; rc=$?
  /usr/bin/grep -q "(future '9999-01-01T00:00:00Z' > now+300s)" "$d/e3" && [ "$rc" -eq 2 ] && ok "watermark-strict future rc2" || notok "watermark-strict future rc=$rc"
  [ -s "$d/ag3.log" ] && notok "watermark-strict future argv non-empty" || ok "watermark-strict future empty argv"
  # control 1: known-good reaches stub
  printf '2026-09-08T06:19:00Z' > "$d/wm_ctl"
  : > "$d/ag4.log"; make_stub_gh "$d/stub4" "$d" "$d/ag4.log"
  run_instr "$d/o4" "$d/e4" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_ctl" --control 5:0 --gh-bin "$d/stub4"; rc=$?
  /usr/bin/grep -q "watermark: $d/wm_ctl = 2026-09-08T06:19:00Z ok" "$d/o4" && [ "$rc" -eq 0 ] && ok "watermark-strict ctl1 ok rc0" || notok "watermark-strict ctl1 rc=$rc"
  [ -s "$d/ag4.log" ] && ok "watermark-strict ctl1 argv non-empty" || notok "watermark-strict ctl1 argv empty"
  # control 2: now-60s
  now60="$(python3 -c 'import datetime as d; print((d.datetime.now(d.timezone.utc)-d.timedelta(seconds=60)).strftime("%Y-%m-%dT%H:%M:%SZ"))')"
  printf '%s' "$now60" > "$d/wm_now"
  : > "$d/ag5.log"; make_stub_gh "$d/stub5" "$d" "$d/ag5.log"
  run_instr "$d/o5" "$d/e5" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --watermark-file "$d/wm_now" --control 5:0 --gh-bin "$d/stub5"; rc=$?
  /usr/bin/grep -q "watermark: $d/wm_now = $now60 ok" "$d/o5" && [ "$rc" -eq 0 ] && ok "watermark-strict ctl2 now-60 ok" || notok "watermark-strict ctl2 rc=$rc"
  [ -s "$d/ag5.log" ] && ok "watermark-strict ctl2 argv non-empty" || notok "watermark-strict ctl2 argv empty"
}

# ── arm 24: watermark-parser-control (AC-M1-17) ───────────────────────────────
arm_watermark_parser_control() {
  local d out err rc stubdate
  d="$SCRATCH/a24"; mkdir -p "$d/stubdate"
  # PATH-shadowing stub date
  stubdate="$d/stubdate/date"
  printf '#!/usr/bin/env bash\nexit 1\n' > "$stubdate"; chmod +x "$stubdate"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  : > "$d/argv.log"; make_stub_gh "$d/stub" "$d" "$d/argv.log"
  PATH="$d/stubdate:$PATH" /bin/bash "$SCRIPT_UT" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-08T06:19:00Z --control 5:0 --gh-bin "$d/stub" > "$d/out" 2> "$d/err"; rc=$?
  /usr/bin/grep -q '✗ timestamp parser unusable: neither BSD nor GNU date parsed the control value' "$d/err" && [ "$rc" -eq 2 ] && ok "watermark-parser-control D7a-ctl rc2" || notok "watermark-parser-control rc=$rc"
  if [ ! -s "$d/argv.log" ]; then ok "watermark-parser-control empty argv"; else notok "watermark-parser-control argv non-empty"; fi
  /usr/bin/grep -q 'watermark invalid' "$d/out" "$d/err" && notok "watermark-parser-control mentions 'watermark invalid'" || ok "watermark-parser-control no 'watermark invalid'"
}

# ── arm 25: driver-literal — the REAL driver's notice prefix == the instrument's SIG_TEXT ──
arm_driver_literal() {
  local d rc drv ext sig
  d="$SCRATCH/a25"; mkdir -p "$d"
  drv="${G0_DRIVER_SRC:-tools/launchd/mission-control.sh}"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"plain"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --driver-src "$drv" --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q "^signature-source: $drv ok" "$d/out" && [ "$rc" -eq 0 ] && ok "driver-literal instrument accepts driver" || notok "driver-literal rc=$rc"
  ext="$(/usr/bin/grep -o -F '⚠️ Mission iteration **FAILED to complete** (rc=' "$drv" | head -1)"
  sig="$(sed -n 's/^SIG_TEXT="\(.*\)"$/\1/p' "$SCRIPT_UT")"
  if [ -n "$ext" ] && [ "$ext" = "$sig" ]; then ok "driver-literal driver prefix == SIG_TEXT"; else notok "driver-literal drift: driver='$ext' script='$sig'"; fi
}

# ── arm 26: single-literal — python reads the bash SIG_TEXT, not its own copy ──
arm_single_literal() {
  local d rc c
  d="$SCRATCH/a26"; mkdir -p "$d"
  make_json "$d/5.json" <<EOF
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"${SIG}143 — x"}]}
EOF
  make_json "$d/5.meta.json" <<'EOF'
{"comments": 1}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  # positive control: the unmutated instrument classifies the notice
  run_instr "$d/o1" "$d/e1" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:1 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 1 ]; then ok "single-literal control classifies"; else notok "single-literal control rc=$rc"; fi
  # mutant: scratch COPY with SIG_TEXT changed; must NOT classify (control 5:0)
  sed 's/^SIG_TEXT="\(.*\)\*\*FAILED to complete\*\*/SIG_TEXT="\1FAILED to complete/' "$SCRIPT_UT" > "$d/mut.sh"
  if cmp -s "$SCRIPT_UT" "$d/mut.sh"; then notok "single-literal mutation did not apply"; return; fi
  /bin/bash "$d/mut.sh" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub" > "$d/o2" 2> "$d/e2"; rc=$?
  c="$(/usr/bin/grep -c '^crash:' "$d/o2")"
  if [ "$rc" -eq 0 ] && [ "$c" -eq 0 ]; then ok "single-literal mutated SIG_TEXT classifies 0"; else notok "single-literal mutant rc=$rc crash=$c"; fi
}

# ── arm 27: control-auto — per-repo control table, unknown repo is a floor ──
arm_control_auto() {
  local d rc i
  d="$SCRATCH/a27"; mkdir -p "$d"
  {
    echo '{"comments": ['
    for i in 1 2 3 4 5 6 7 8 9 10; do
      [ "$i" -gt 1 ] && echo ','
      printf ' {"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-%02dT00:00:00Z","url":"https://x/852#%d","body":"%s143 — n"}' "$i" "$i" "$SIG"
    done
    echo ']}'
  } > "$d/852.json"
  make_json "$d/852.meta.json" <<'EOF'
{"comments": 10}
EOF
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/o1" "$d/e1" --issue 852 --repo sunholo-data/ailang --self sunholo-voight-kampff \
    --since 2026-09-20T00:00:00Z --control auto --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '^control: issue=852 expect=10 got=10 ok' "$d/o1" && [ "$rc" -eq 0 ] && ok "control-auto ailang 852:10" || notok "control-auto ailang rc=$rc"
  cp "$FIXDIR/107.json" "$d/107.json"; cp "$FIXDIR/107.meta.json" "$d/107.meta.json"
  run_instr "$d/o2" "$d/e2" --issue 107 --repo sunholo-data/ailang-world --self sunholo-voight-kampff \
    --since 2026-09-20T00:00:00Z --control auto --gh-bin "$d/stub"; rc=$?
  /usr/bin/grep -q '^control: issue=107 expect=4 got=4 ok' "$d/o2" && [ "$rc" -eq 0 ] && ok "control-auto world 107:4" || notok "control-auto world rc=$rc"
  : > "$d/argv3.log"; make_stub_gh "$d/stub3" "$d" "$d/argv3.log"
  run_instr "$d/o3" "$d/e3" --issue 5 --repo test/unknown --self sunholo-voight-kampff \
    --since 2026-09-20T00:00:00Z --control auto --gh-bin "$d/stub3"; rc=$?
  /usr/bin/grep -q '✗ control required: no control for repo test/unknown' "$d/e3" && [ "$rc" -eq 2 ] && ok "control-auto unknown repo F6 rc2" || notok "control-auto unknown rc=$rc"
  if [ ! -s "$d/argv3.log" ]; then ok "control-auto unknown repo empty argv"; else notok "control-auto unknown repo argv non-empty"; fi
}

# ── arm 28: snapshot-1380 — real sunholo-data/ailang fleet -prev thread ──
arm_snapshot_1380() {
  local d rc c
  d="$SCRATCH/a28"; mkdir -p "$d"
  cp "$FIXDIR/1380.json" "$d/1380.json"; cp "$FIXDIR/1380.meta.json" "$d/1380.meta.json"
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 1380 --repo sunholo-data/ailang --self sunholo-voight-kampff \
    --since 2026-10-01T00:00:00Z --control 1380:3 --gh-bin "$d/stub"; rc=$?
  if [ "$rc" -eq 1 ]; then ok "snapshot-1380 rc1"; else notok "snapshot-1380 rc=$rc"; fi
  c="$(/usr/bin/grep -c '^crash: issue=1380' "$d/out")"
  if [ "$c" -eq 3 ]; then ok "snapshot-1380 three crash lines"; else notok "snapshot-1380 crash count=$c"; fi
  /usr/bin/grep -q '^crash: issue=1380 at=2026-10-02T02:18:21Z rc=1 in-window=yes' "$d/out" && ok "snapshot-1380 line 10-02 02:18" || notok "snapshot-1380 missing 02:18"
  /usr/bin/grep -q '^crash: issue=1380 at=2026-10-02T21:56:53Z rc=1 in-window=yes' "$d/out" && ok "snapshot-1380 line 10-02 21:56" || notok "snapshot-1380 missing 21:56"
  /usr/bin/grep -q '^crash: issue=1380 at=2026-10-04T04:28:01Z rc=143 in-window=yes' "$d/out" && ok "snapshot-1380 line 10-04 04:28" || notok "snapshot-1380 missing 04:28"
  /usr/bin/grep -q '^issue: 1380 comments=26 self=26 crash=3 other=23$' "$d/out" && ok "snapshot-1380 issue line" || notok "snapshot-1380 issue line: $(/usr/bin/grep '^issue: 1380 ' "$d/out")"
  /usr/bin/grep -q '^control: issue=1380 expect=3 got=3 ok' "$d/out" && ok "snapshot-1380 control ok" || notok "snapshot-1380 control missing"
}

# ── dispatcher ────────────────────────────────────────────────────────────────
run_arm() {
  case "$1" in
    anchored-signature) arm_anchored_signature ;;
    rc-extract) arm_rc_extract ;;
    since-window) arm_since_window ;;
    self-filter) arm_self_filter ;;
    prev-issue) arm_prev_issue ;;
    no-bodies) arm_no_bodies ;;
    usage) arm_usage ;;
    gh-missing) arm_gh_missing ;;
    gh-timeout) arm_gh_timeout ;;
    gh-failed) arm_gh_failed ;;
    malformed) arm_malformed ;;
    truncated) arm_truncated ;;
    control-required) arm_control_required ;;
    control-mismatch) arm_control_mismatch ;;
    signature-source) arm_signature_source ;;
    report) arm_report ;;
    snapshot-107) arm_snapshot_107 ;;
    snapshot-129) arm_snapshot_129 ;;
    watermark-older) arm_watermark_older ;;
    watermark-degraded) arm_watermark_degraded ;;
    watermark-unavailable) arm_watermark_unavailable ;;
    watermark-invalid) arm_watermark_invalid ;;
    watermark-strict) arm_watermark_strict ;;
    watermark-parser-control) arm_watermark_parser_control ;;
    driver-literal) arm_driver_literal ;;
    single-literal) arm_single_literal ;;
    control-auto) arm_control_auto ;;
    snapshot-1380) arm_snapshot_1380 ;;
    *) echo "unknown arm: $1" >&2; exit 2 ;;
  esac
}

ONLY=""
if [ "$#" -gt 0 ]; then
  if [ "$1" = "--only" ]; then
    ONLY="$2"
  else
    echo "unknown argument: $1" >&2
    exit 2
  fi
fi

if [ -n "$ONLY" ]; then
  run_arm "$ONLY"
else
  for arm in anchored-signature rc-extract since-window self-filter prev-issue no-bodies \
             usage gh-missing gh-timeout gh-failed malformed truncated control-required \
             control-mismatch signature-source report snapshot-107 snapshot-129 \
             watermark-older watermark-degraded watermark-unavailable watermark-invalid \
             watermark-strict watermark-parser-control \
             driver-literal single-literal control-auto snapshot-1380; do
    run_arm "$arm"
  done
fi

echo "---"
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
