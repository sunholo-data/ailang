#!/usr/bin/env bash
# Sourced by test_mission_gate0_self_notices.sh after its helpers; no live reads.
# Every arm owns literal fixtures. Failures print labels, never captured bodies.

transcript_has() {  # label, literal expected metadata, output file(s)
  local label="$1" expected="$2"; shift 2
  if /usr/bin/grep -Fq -- "$expected" "$@"; then ok "$label"; else notok "$label (expected metadata: $expected)"; fi
}
transcript_lacks() {
  local label="$1" forbidden="$2"; shift 2
  if /usr/bin/grep -Fq -- "$forbidden" "$@"; then notok "$label"; else ok "$label"; fi
}
transcript_rc() {
  if [ "$2" -eq "$3" ]; then ok "$1 rc$3"; else notok "$1 expected rc$3 got rc$2"; fi
}

arm_transcript_old() {
  local d="$SCRATCH/transcript-old" rc c
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#old","body":"Transcript secret exposure on fire old-087: 12 key/file hit(s). Old trailing wording."}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-old "$rc" 0
  transcript_has 'transcript-old row' 'transcript: issue=5 at=2026-09-01T00:00:00Z hits=12 in-window=yes url=https://x/5#old fire=old-087' "$d/out"
  c="$(/usr/bin/grep -c '^transcript:' "$d/out")"
  if [ "$c" -eq 1 ]; then ok 'transcript-old one row'; else notok 'transcript-old one row'; fi
  transcript_lacks 'transcript-old no crash' 'crash:' "$d/out"
}

arm_transcript_current() {
  local d="$SCRATCH/transcript-current" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"SuNhOlO-VoIgHt-KaMpFf"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#current","body":"Transcript secret scan after fire current-087: 3 key/file hit(s)."}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self SUNHOLO-VOIGHT-KAMPFF \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-current "$rc" 0
  transcript_has 'transcript-current row' 'transcript: issue=5 at=2026-09-01T00:00:00Z hits=3 in-window=yes url=https://x/5#current fire=current-087' "$d/out"
  transcript_has 'transcript-current count' 'comments=1 self=1 crash=0 transcript=1 other=0' "$d/out"
}

arm_transcript_foreign() {
  local d="$SCRATCH/transcript-foreign" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"foreign"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret scan after fire foreign-087: 2 key/file hit(s)."}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-foreign "$rc" 0
  transcript_lacks 'transcript-foreign no row' 'transcript:' "$d/out"
  transcript_has 'transcript-foreign verdict zero' 'transcript notice(s) since watermark: 0' "$d/out"
}

arm_transcript_offset() {
  local d="$SCRATCH/transcript-offset" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":" Transcript secret scan after fire space-087: 1 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Quoted in prose: Transcript secret exposure on fire quote-087: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":2}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-offset "$rc" 0
  transcript_lacks 'transcript-offset no row' 'transcript:' "$d/out"
  transcript_has 'transcript-offset unclassified' 'crash=0 transcript=0 other=2' "$d/out"
}

arm_transcript_ascii() {
  local d="$SCRATCH/transcript-ascii" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret scan after fire unicode-087: ١٢ key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire empty-087:  key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","body":"Transcript secret scan after fire ascii-087: 12 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":3}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-ascii "$rc" 0
  transcript_has 'transcript-ascii positive' 'hits=12 in-window=yes url=- fire=ascii-087' "$d/out"
  transcript_lacks 'transcript-ascii rejects unicode' 'fire=unicode-087' "$d/out"
  transcript_lacks 'transcript-ascii rejects empty' 'fire=empty-087' "$d/out"
  transcript_has 'transcript-ascii arithmetic' 'crash=0 transcript=1 other=2' "$d/out"
}

arm_transcript_header_shape() {
  local d="$SCRATCH/transcript-header-shape" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret scan after fire period-087: 1 key/file hit(s)"},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire colon-087:1 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","body":"Transcript secret exposure on fire two words: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":3}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-header-shape "$rc" 0
  transcript_lacks 'transcript-header-shape no row' 'transcript:' "$d/out"
  transcript_has 'transcript-header-shape arithmetic' 'crash=0 transcript=0 other=3' "$d/out"
}

arm_transcript_window() {
  local d="$SCRATCH/transcript-window" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret exposure on fire before-087: 1 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire equal-087: 1 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","body":"Transcript secret scan after fire after-087: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":3}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2026-09-02T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-window "$rc" 0
  transcript_has 'transcript-window before' 'hits=1 in-window=no url=- fire=before-087' "$d/out"
  transcript_has 'transcript-window equal excluded' 'hits=1 in-window=no url=- fire=equal-087' "$d/out"
  transcript_has 'transcript-window after' 'hits=1 in-window=yes url=- fire=after-087' "$d/out"
  transcript_has 'transcript-window total' 'crash=0 transcript=3 other=0' "$d/out"
  transcript_has 'transcript-window verdict one' 'transcript notice(s) since watermark: 1' "$d/out"
}

arm_transcript_verdict_zero() {
  local d="$SCRATCH/transcript-verdict-zero" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret exposure on fire dozen-087: 12 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire one-087: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":2}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-verdict-zero "$rc" 0
  /usr/bin/grep '^verdict:' "$d/out" > "$d/verdict"
  transcript_has 'transcript-verdict-zero notices not hits' 'transcript notice(s) since watermark: 2' "$d/verdict"
  transcript_has 'transcript-verdict-zero disposition' 'exposure reported: key names are in the private controlplane notice; report the count and fire id, never a value or path' "$d/verdict"
}

arm_transcript_verdict_crash() {
  local d="$SCRATCH/transcript-verdict-crash" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"⚠️ Mission iteration **FAILED to complete** (rc=143 — crash"},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire later-087: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":2}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:1 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-verdict-crash "$rc" 1
  /usr/bin/grep '^verdict:' "$d/out" > "$d/verdict"
  transcript_has 'transcript-verdict-crash suffix' 'transcript notice(s) since watermark: 1' "$d/verdict"
  transcript_has 'transcript-verdict-crash newest remains crash' 'A FIRE DIED (newest rc=143 at 2026-09-01T00:00:00Z on #5)' "$d/verdict"
  transcript_has 'transcript-verdict-crash recovery' "run Gate 2's died-mid-flight traces (a) open PRs (b) worktrees (c) uncommitted state BEFORE picking" "$d/verdict"
  transcript_has 'transcript-verdict-crash disposition' 'exposure reported: key names are in the private controlplane notice; report the count and fire id, never a value or path' "$d/verdict"
}

arm_transcript_other() {
  local d="$SCRATCH/transcript-other" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"⚠️ Mission iteration **FAILED to complete** (rc=1 — crash"},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret exposure on fire old-087: 12 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-03T00:00:00Z","body":"Transcript secret scan after fire current-087: 1 key/file hit(s)."},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-04T00:00:00Z","body":"plain self comment"},
{"author":{"login":"foreign"},"createdAt":"2026-09-05T00:00:00Z","body":"plain foreign comment"},
{"author":{"login":"foreign"},"createdAt":"2026-09-06T00:00:00Z","body":"Transcript secret scan after fire foreign-087: 2 key/file hit(s)."}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":6}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:1 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-other "$rc" 1
  transcript_has 'transcript-other arithmetic' 'issue: 5 comments=6 self=4 crash=1 transcript=2 other=1' "$d/out"
}

arm_transcript_no_bodies() {
  local d="$SCRATCH/transcript-no-bodies" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#old","body":"Transcript secret exposure on fire old-087: 12 key/file hit(s). BODY_TAIL_SENTINEL_G087"},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","url":"https://x/5#current","body":"Transcript secret scan after fire current-087: 1 key/file hit(s). BODY_TAIL_SENTINEL_G087"}
]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":2}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-no-bodies "$rc" 0
  transcript_has 'transcript-no-bodies old metadata' 'hits=12 in-window=yes url=https://x/5#old fire=old-087' "$d/out"
  transcript_has 'transcript-no-bodies current metadata' 'hits=1 in-window=yes url=https://x/5#current fire=current-087' "$d/out"
  transcript_lacks 'transcript-no-bodies sentinel absent stdout and stderr' 'BODY_TAIL_SENTINEL_G087' "$d/out" "$d/err"
}

arm_transcript_prev() {
  local d="$SCRATCH/transcript-prev" rc c
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"plain"}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_json "$d/6.json" <<'JSON'
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire previous-087: 1 key/file hit(s)."}]}
JSON
  make_json "$d/6.meta.json" <<'JSON'
{"comments":1}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --prev-issue 6 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-prev "$rc" 0
  transcript_has 'transcript-prev issue6 row' 'transcript: issue=6 at=2026-09-02T00:00:00Z hits=1 in-window=yes url=- fire=previous-087' "$d/out"
  transcript_has 'transcript-prev verdict one' 'transcript notice(s) since watermark: 1' "$d/out"
  : > "$d/argv.log"
  run_instr "$d/out2" "$d/err2" --issue 6 --prev-issue 6 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc 'transcript-prev duplicate' "$rc" 0
  c="$(/usr/bin/grep -c '^transcript:' "$d/out2")"
  if [ "$c" -eq 1 ]; then ok 'transcript-prev no duplicate row'; else notok 'transcript-prev no duplicate row'; fi
  transcript_has 'transcript-prev no duplicate verdict' 'transcript notice(s) since watermark: 1' "$d/out2"
  c="$(/usr/bin/grep -c '^issue view 6 ' "$d/argv.log")"
  if [ "$c" -eq 1 ]; then ok 'transcript-prev one fetch'; else notok 'transcript-prev one fetch'; fi
}

arm_transcript_control_isolation() {
  local d="$SCRATCH/transcript-control-isolation" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"plain"}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_json "$d/6.json" <<'JSON'
{"comments":[
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"⚠️ Mission iteration **FAILED to complete** (rc=143 — control"},
{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-02T00:00:00Z","body":"Transcript secret scan after fire control-087: 1 key/file hit(s)."}
]}
JSON
  make_json "$d/6.meta.json" <<'JSON'
{"comments":2}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 6:1 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-control-isolation "$rc" 0
  transcript_has 'transcript-control-isolation crash only control' 'control: issue=6 expect=1 got=1 ok' "$d/out"
  transcript_has 'transcript-control-isolation verdict zero' 'transcript notice(s) since watermark: 0' "$d/out"
  transcript_lacks 'transcript-control-isolation no control rows' 'transcript: issue=6' "$d/out"
}

arm_transcript_f8_current_missing() {
  local d="$SCRATCH/transcript-f8-current-missing" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":0}
JSON
  cat > "$d/driver" <<'DRIVER'
⚠️ Mission iteration **FAILED to complete** (rc=
Transcript secret exposure on fire fixture
DRIVER
  : > "$d/argv.log"
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --driver-src "$d/driver" --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-f8-current-missing "$rc" 2
  transcript_has 'transcript-f8-current-missing diagnostic' 'current transcript signature not found in driver source:' "$d/err"
  if [ ! -s "$d/argv.log" ]; then ok 'transcript-f8-current-missing empty argv'; else notok 'transcript-f8-current-missing empty argv'; fi
}

arm_transcript_single_literal() {
  local d="$SCRATCH/transcript-single-literal" rc
  mkdir -p "$d"
  make_json "$d/5.json" <<'JSON'
{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","body":"Transcript secret scan after fire literal-087: 1 key/file hit(s)."}]}
JSON
  make_json "$d/5.meta.json" <<'JSON'
{"comments":1}
JSON
  make_stub_gh "$d/stub" "$d" "$d/argv.log"
  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub"; rc=$?
  transcript_rc transcript-single-literal "$rc" 0
  transcript_has 'transcript-single-literal positive' 'fire=literal-087' "$d/out"
  sed 's/^TRANSCRIPT_CURRENT=".*"$/TRANSCRIPT_CURRENT="Changed prefix "/' "$SCRIPT_UT" > "$d/mut.sh"
  if cmp -s "$SCRIPT_UT" "$d/mut.sh"; then notok 'transcript-single-literal mutation did not apply'; return; fi
  /bin/bash "$d/mut.sh" --issue 5 --repo test/test --self sunholo-voight-kampff \
    --since 2000-01-01T00:00:00Z --control 5:0 --gh-bin "$d/stub" > "$d/out2" 2> "$d/err2"; rc=$?
  transcript_rc 'transcript-single-literal changed constant' "$rc" 0
  transcript_lacks 'transcript-single-literal changed constant unclassified' 'transcript:' "$d/out2"
}

transcript_arms() { printf '%s\n' 'transcript-old transcript-current transcript-foreign transcript-offset transcript-ascii transcript-header-shape transcript-window transcript-verdict-zero transcript-verdict-crash transcript-other transcript-no-bodies transcript-prev transcript-control-isolation transcript-f8-current-missing transcript-single-literal'; }
run_transcript_arm() {
  case "$1" in
    transcript-old) arm_transcript_old ;;
    transcript-current) arm_transcript_current ;;
    transcript-foreign) arm_transcript_foreign ;;
    transcript-offset) arm_transcript_offset ;;
    transcript-ascii) arm_transcript_ascii ;;
    transcript-header-shape) arm_transcript_header_shape ;;
    transcript-window) arm_transcript_window ;;
    transcript-verdict-zero) arm_transcript_verdict_zero ;;
    transcript-verdict-crash) arm_transcript_verdict_crash ;;
    transcript-other) arm_transcript_other ;;
    transcript-no-bodies) arm_transcript_no_bodies ;;
    transcript-prev) arm_transcript_prev ;;
    transcript-control-isolation) arm_transcript_control_isolation ;;
    transcript-f8-current-missing) arm_transcript_f8_current_missing ;;
    transcript-single-literal) arm_transcript_single_literal ;;
    *) return 1 ;;
  esac
  return 0
}
