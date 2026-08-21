#!/usr/bin/env bash
# shellcheck disable=SC2034 # Variables are consumed by extracted production functions.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
DRIVER="$REPO_ROOT/tools/launchd/nightly-eval.sh"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/fmt-ab-schedule.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT
FAILED=0

pass() { echo "PASS: $*"; }
fail() { echo "FAIL: $*"; FAILED=1; }

# Source the production function without executing the nightly driver's top-level rig work.
sed -n '/^# BEGIN FMT_AB_TESTABLE_FUNCTIONS$/,/^# END FMT_AB_TESTABLE_FUNCTIONS$/p' "$DRIVER" > "$TMP_DIR/functions.sh"
# ANTI-VACUITY FLOOR on this test's own ENUMERATOR. The marker pair above is what decides which
# production text is under test, so a renamed or deleted marker silently empties the whole suite.
# Without this the failure surfaces as `run_fmt_ab_measurements: command not found` (rc=127), which
# is loud but reads as a broken test rather than as a broken gate. An empty extraction FAILS here,
# by name, and never passes.
if [ ! -s "$TMP_DIR/functions.sh" ]; then
    echo "FAIL: instrument failure: FMT_AB_TESTABLE_FUNCTIONS marker extraction from $DRIVER produced no text"
    exit 1
fi
# shellcheck source=/dev/null
. "$TMP_DIR/functions.sh"
for fn in run_fmt_ab_smoke run_fmt_ab_measurements; do
    if ! declare -F "$fn" >/dev/null 2>&1; then
        echo "FAIL: instrument failure: $fn is not defined after sourcing the extracted markers"
        exit 1
    fi
done

INVOCATION_LOG="${FMT_AB_EMIT_LOG:-$TMP_DIR/invocations.log}"
STUB_HIT="$TMP_DIR/stub-hit"
cat > "$TMP_DIR/ailang-stub" <<'STUB'
#!/usr/bin/env bash
: > "$FMT_AB_STUB_HIT"
printf '%s\n' "$*" >> "$FMT_AB_INVOCATION_LOG"
if [ "${1:-}" = "eval-suite" ] && [ "${FMT_AB_SMOKE_STATE:-}" != "" ]; then
  output=""
  trials=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --output) output="$2"; shift 2 ;;
      --trials) trials="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  if [ "$trials" = "1" ]; then
    mkdir -p "$output/agent"
    case "$FMT_AB_SMOKE_STATE" in
      on)
        printf '%s\n' '{"fmt_hook_state":"on","fmt_hook_events":[{"status":"formatted","file":"x.ail"}]}' > "$output/agent/smoke.json" ;;
      off)
        printf '%s\n' '{"fmt_hook_state":"off","fmt_hook_events":[{"status":"formatted","file":"x.ail"}]}' > "$output/agent/smoke.json" ;;
      invalid)
        # ON arm resolved, but the banked treatment-integrity verdict is FALSE: the extension was
        # named in resolved_extensions and delivered nothing. This is doc 5.2 requirement (a) --
        # "firing is not delivering" -- and it is a DIFFERENT refusal branch from fmt_hook_state.
        printf '%s\n' '{"fmt_hook_state":"on","validity":{"valid":false,"reason":"treatment_unproven"},"fmt_hook_events":[]}' > "$output/agent/smoke.json" ;;
      norow)
        : ;;   # bank nothing: exercises the row-count refusal branch
      *)
        printf '%s\n' '{"fmt_hook_state":"off"}' > "$output/agent/smoke.json" ;;
    esac
  fi
fi
STUB
chmod +x "$TMP_DIR/ailang-stub"

export FMT_AB_STUB_HIT="$STUB_HIT"
export FMT_AB_INVOCATION_LOG="$INVOCATION_LOG"
BIN="$TMP_DIR/ailang-stub"
FMT_BENCH_LIST="b0,b1,b2,b3,b4,b5"
FMT_TRIALS=5
RESULTS_DIR="$TMP_DIR/results"
MAX_TOKENS_PER_BENCH=4000000
LOG="$TMP_DIR/driver.log"
log() { printf '%s\n' "$*" >> "$LOG"; }

: > "$INVOCATION_LOG"
run_fmt_ab_measurements

# EMIT-ONLY: write the schedule artifact and stop, WITHOUT running this file's own assertions.
# The Go consumer (TestFmtDriverScheduleSatisfiesOrderIntegrity) needs the driver's raw output, not
# this script's verdict on it: if the Go test only ran the full suite, a wrong schedule would red
# the shell assertions first and CheckFmtOrderIntegrity would never be reached -- so the analyzer
# call would be decorative for every schedule defect it is cited to catch.
if [ "${FMT_AB_EMIT_ONLY:-0}" = "1" ]; then
    if [ -z "${FMT_AB_EMIT_LOG:-}" ]; then
        echo "FAIL: instrument failure: FMT_AB_EMIT_ONLY=1 requires FMT_AB_EMIT_LOG outside the temp dir"
        exit 1
    fi
    if [ ! -s "$INVOCATION_LOG" ]; then
        echo "FAIL: instrument failure: emit-only produced zero invocations"
        exit 1
    fi
    echo "fmt A/B schedule: emit-only artifact at $INVOCATION_LOG"
    exit 0
fi

if [ -f "$STUB_HIT" ]; then
    pass "control: stub was invoked"
else
    fail "instrument failure: stub was never invoked"
fi

count=$(wc -l < "$INVOCATION_LOG" | tr -d ' ')
if [ "$count" -eq 0 ]; then
    fail "instrument failure: enumeration found zero invocations"
elif [ "$count" -eq 12 ]; then
    pass "schedule emitted exactly 12 invocations"
else
    fail "schedule emitted $count invocations, want 12"
fi

expected='b0:on b0:off b1:off b1:on b2:on b2:off b3:off b3:on b4:on b4:off b5:off b5:on'
actual=$(awk '
{
  bench=""; model=""
  for (i=1; i<=NF; i++) {
    if ($i=="--benchmarks") bench=$(i+1)
    if ($i=="--models") model=$(i+1)
  }
  arm=(model=="motoko-local-qwen3-6-fmt" ? "on" : "off")
  printf "%s%s:%s", (NR==1 ? "" : " "), bench, arm
}
END { print "" }
' "$INVOCATION_LOG")
if [ "$actual" = "$expected" ]; then
    pass "counterbalanced sequence: $actual"
else
    fail "counterbalanced sequence = '$actual', want '$expected'"
fi

bad_width=$(awk '
{
  for (i=1; i<=NF; i++) if ($i=="--benchmarks" && $(i+1) ~ /,/) bad++
}
END { print bad+0 }
' "$INVOCATION_LOG")
if [ "$bad_width" -eq 0 ]; then
    pass "every invocation names exactly one benchmark"
else
    fail "$bad_width invocation(s) name multiple benchmarks"
fi

run_smoke_case() {
    local state="$1" want_measurements="$2" label="$3" measurement_count
    : > "$INVOCATION_LOG"
    : > "$LOG"
    RUN_AB_FMT=1
    DATE=2026-08-21
    export FMT_AB_SMOKE_STATE="$state"
    PATH="$TMP_DIR:$PATH" run_fmt_ab_smoke || true
    if [ "$RUN_AB_FMT" = "1" ]; then
        run_fmt_ab_measurements
    fi
    measurement_count=$(awk '
      /eval-suite/ {
        for (i=1; i<=NF; i++) if ($i=="--trials" && $(i+1)==5) count++
      }
      END { print count+0 }
    ' "$INVOCATION_LOG")
    if [ "$measurement_count" -eq "$want_measurements" ]; then
        pass "$label: measurement invocation count=$measurement_count"
    else
        fail "$label: measurement invocation count=$measurement_count, want $want_measurements"
    fi
}

ln -s "$TMP_DIR/ailang-stub" "$TMP_DIR/ailang"
run_smoke_case off 0 "failing smoke"
if grep -q "fmt_hook_state contract failed: observed 'off'" "$LOG"; then
    pass "failing smoke logs the specific observed contract"
else
    fail "failing smoke did not log the fmt_hook_state contract and observed value"
fi
run_smoke_case invalid 0 "invalid-treatment smoke"
if grep -q "treatment-integrity contract failed: observed validity.valid='false'" "$LOG"; then
    pass "invalid-treatment smoke logs the specific observed contract"
else
    fail "invalid-treatment smoke did not log the treatment-integrity contract and observed value"
fi
run_smoke_case norow 0 "no-banked-row smoke"
if grep -q "banked-row contract failed: observed row_count=0" "$LOG"; then
    pass "no-banked-row smoke logs the specific observed contract"
else
    fail "no-banked-row smoke did not log the banked-row contract and observed value"
fi
run_smoke_case on 12 "happy smoke"

if [ "$FAILED" -eq 0 ]; then
    echo "fmt A/B schedule: OK"
    exit 0
fi
echo "fmt A/B schedule: FAILED"
exit 1
