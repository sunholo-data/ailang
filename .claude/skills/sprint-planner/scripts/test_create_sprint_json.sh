#!/bin/bash
# Isolated placement regression; optional source path supports mutation drills.
set -eu
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
SOURCE=${1:-$SCRIPT_DIR/create_sprint_json.sh}
SOURCE=$(cd "$(dirname "$SOURCE")" && pwd)/$(basename "$SOURCE")
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT
passed=0
failed=0

check() {
    local arm=$1
    shift
    if "$@"; then
        echo "PASS: $arm"
        passed=$((passed + 1))
    else
        echo "FAIL: $arm"
        failed=$((failed + 1))
    fi
}

fixture() {
    CASE=$(mktemp -d "$TEST_ROOT/case.XXXXXX")
    REPO=$CASE/repo
    CALLER=$REPO
    TRIAGE=design_docs/planned/ailang-core-triage
    mkdir -p "$REPO/.claude/skills/sprint-planner/scripts" "$REPO/$TRIAGE" \
        "$REPO/design_docs/planned/v0_55_0" "$CASE/bin" "$CASE/home" "$CASE/tmp"
    COPIED=$REPO/.claude/skills/sprint-planner/scripts/create_sprint_json.sh
    cp "$SOURCE" "$COPIED"
    cat > "$CASE/bin/ailang" <<'STUB'
#!/bin/bash
printf '%s\n' "$*" >> "$STUB_LOG"
printf '[]\n'
STUB
    chmod +x "$CASE/bin/ailang"
    : > "$CASE/stub.log"
    cat > "$CASE/plan.md" <<'PLAN'
# Sprint fixture
### M1: First (~10 LOC)
- [ ] First criterion
### M2: Second (~10 LOC)
- [ ] Second criterion
PLAN
}

invoke() {
    local plan=$1 design=${2:-}
    # Assert command resolution in the same subshell/environment as the script.
    if (
        cd "$CALLER"
        export HOME="$CASE/home" TMPDIR="$CASE/tmp" STUB_LOG="$CASE/stub.log"
        export PATH="$CASE/bin:$PATH"
        [ "$(command -v ailang)" = "$CASE/bin/ailang" ] || exit 99
        /bin/bash "$COPIED" M-TEST "$plan" "$design"
    ) > "$CASE/output" 2>&1; then
        rc=0
    else
        rc=$?
    fi
}

one_diagnostic() {
    [ "$rc" -eq 2 ] && [ "$(wc -l < "$CASE/output" | tr -d ' ')" -eq 1 ] &&
        grep -q 'PLAN LOCATION' "$CASE/output" &&
        grep -q 'design_docs/planned/ailang-core-triage/' "$CASE/output"
}

refused() {
    check "$1" test "$rc" -eq 2
    check before-mkdir test ! -e "$CALLER/.ailang/state"
    check before-mkdir test ! -e "$REPO/.ailang/state"
    check before-import test ! -s "$CASE/stub.log"
    check diagnostic-contract one_diagnostic
}

accepted() {
    check "$1" test "$rc" -eq 0
    check "$1" grep -q '^messages import-github --labels bug,feature,ailang-message$' "$CASE/stub.log"
    check "$1" python3 - "$CALLER/.ailang/state/sprints/sprint_M-TEST.json" <<'PY'
import json, sys
with open(sys.argv[1]) as source:
    assert json.load(source)['sprint_id'] == 'M-TEST'
PY
}

fixture
invoke "$TRIAGE/x.md" # Nonexistent leaf must be refused before checking existence.
refused raw-refusal
fixture
cp "$CASE/plan.md" "$REPO/$TRIAGE/x.md"
invoke "$TRIAGE/x.md"
refused raw-refusal
fixture
invoke "./$TRIAGE/x.md"
refused dotted-refusal
fixture
CALLER=$CASE/elsewhere
mkdir -p "$CALLER"
invoke "$REPO/$TRIAGE/x.md"
refused foreign-cwd-refusal
fixture
ln -s "$REPO/$TRIAGE" "$REPO/alias"
cp "$CASE/plan.md" "$REPO/$TRIAGE/x.md"
invoke alias/x.md
refused canonical-refusal
fixture
cp "$CASE/plan.md" "$REPO/$TRIAGE/x.md"
ln -s "$REPO/$TRIAGE/x.md" "$REPO/alias.md"
invoke alias.md
refused canonical-refusal
fixture
invoke design_docs/planned/v0_55_0/../ailang-core-triage/x.md
refused canonical-refusal

for plan in design_docs/planned/v0_55_0/ok.md design_docs/planned/sprint-plan-ok.md; do
    fixture
    cp "$CASE/plan.md" "$REPO/$plan"
    invoke "$plan"
    accepted valid-control
done
fixture
plan=design_docs/planned/ailang-core-triage-extra/ok.md
mkdir -p "$(dirname "$REPO/$plan")"
cp "$CASE/plan.md" "$REPO/$plan"
invoke "$plan"
accepted boundary-control
fixture
plan=design_docs/planned/v0_55_0/ok.md
cp "$CASE/plan.md" "$REPO/$plan"
printf '# Triage design row\nBug Report: msg_20261029_000000_abcd\n' > "$REPO/$TRIAGE/design.md"
invoke "$plan" "$TRIAGE/design.md"
accepted design-control
check design-control grep -q '^messages read msg_20261029_000000_abcd --json$' "$CASE/stub.log"
check design-control grep -q '^messages list --json --limit 100$' "$CASE/stub.log"

echo "create-sprint-json: $passed passed, $failed failed"
[ "$failed" -eq 0 ]
