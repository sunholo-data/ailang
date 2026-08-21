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
if [ ! -s "$TMP_DIR/functions.sh" ]; then
    echo "FAIL: instrument failure: FMT_AB_TESTABLE_FUNCTIONS marker extraction from $DRIVER produced no text"
    exit 1
fi
# shellcheck source=/dev/null
. "$TMP_DIR/functions.sh"
if ! declare -F run_fmt_ab_measurements >/dev/null 2>&1; then
    echo "FAIL: instrument failure: run_fmt_ab_measurements is not defined after sourcing the extracted markers"
    exit 1
fi

INVOCATION_LOG="${FMT_AB_EMIT_LOG:-$TMP_DIR/invocations.log}"
STUB_HIT="$TMP_DIR/stub-hit"
cat > "$TMP_DIR/ailang-stub" <<'STUB'
#!/usr/bin/env bash
: > "$FMT_AB_STUB_HIT"
printf '%s\n' "$*" >> "$FMT_AB_INVOCATION_LOG"
STUB
chmod +x "$TMP_DIR/ailang-stub"

export FMT_AB_STUB_HIT="$STUB_HIT"
export FMT_AB_INVOCATION_LOG="$INVOCATION_LOG"
# Consumed by the sourced production function.
BIN="$TMP_DIR/ailang-stub"
FMT_BENCH_LIST="b0,b1,b2,b3,b4,b5"
FMT_TRIALS=5
RESULTS_DIR="$TMP_DIR/results"
MAX_TOKENS_PER_BENCH=4000000
LOG="$TMP_DIR/driver.log"
log() { printf '%s\n' "$*" >> "$LOG"; }

: > "$INVOCATION_LOG"
run_fmt_ab_measurements

# EMIT-ONLY: write the schedule artifact and stop, WITHOUT running this file's own assertions, so
# the Go consumer gets the driver's raw output rather than this script's verdict on it.
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

if [ "$FAILED" -eq 0 ]; then
    echo "fmt A/B schedule: OK"
    exit 0
fi
echo "fmt A/B schedule: FAILED"
exit 1
