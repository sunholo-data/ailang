#!/bin/bash
# Self-test for check_acceptance_criteria.sh's TST traceability branch.
# Builds a throwaway git repo with a tiny Go module, a design doc and a sprint
# JSON, runs the checker there, and asserts per-criterion status. bash 3.2-safe.
# Run: /bin/bash .claude/skills/sprint-evaluator/scripts/test_check_acceptance_criteria.sh
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
CHECK="$HERE/check_acceptance_criteria.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/acc-crit-test.XXXXXX")
trap 'rm -rf "$TMP"' EXIT

if ! command -v go >/dev/null 2>&1 || ! command -v jq >/dev/null 2>&1; then
    echo "SKIP: needs go and jq"; exit 0
fi

cd "$TMP" || exit 1
git init -q .
mkdir -p pkg/demo scripts design_docs/planned .ailang/state/sprints
cat > go.mod <<'EOF'
module demo

go 1.21
EOF

# TestFail sits ABOVE the TST-DEMO-01 comment: a comment must label the func
# below it, not the nearest one above.
cat > pkg/demo/demo_test.go <<'EOF'
package demo

import "testing"

// TST-DEMO-02
func TestFail(t *testing.T) { t.Fatal("deliberately red") }

// TST-DEMO-01
func TestPass(t *testing.T) {}

func TestBody(t *testing.T) {
	t.Run("TST-DEMO-04 inside a body", func(t *testing.T) {})
}
EOF
printf '#!/bin/sh\n# TST-DEMO-05\nexit 0\n' > scripts/test_demo.sh

# Design doc declares 01..06; 03 and 06 are carried by no test. Quoting an ID
# in design_docs/ must not count as carrying it.
cat > design_docs/planned/demo.md <<'EOF'
| ID | Level | Given / When / Then |
|----|-------|---------------------|
| TST-DEMO-01 | unit | passes |
| TST-DEMO-02 | unit | fails |
| TST-DEMO-03 | unit | nobody wrote it |
| TST-DEMO-04 | unit | subtest |
| TST-DEMO-05 | e2e | shell |
| TST-DEMO-06 | unit | nobody wrote it either |
EOF

cat > .ailang/state/sprints/sprint_DEMO.json <<'EOF'
{
  "sprint_id": "DEMO",
  "design_doc": "design_docs/planned/demo.md",
  "features": [
    {
      "id": "M1",
      "description": "demo",
      "passes": true,
      "acceptance_criteria": [
        "Passing behaviour — TST-DEMO-01",
        "Failing behaviour — TST-DEMO-02",
        "Unwritten behaviour — TST-DEMO-03",
        "Subtest behaviour — TST-DEMO-04",
        "Shell-carried behaviour — TST-DEMO-05",
        "Mixed — TST-DEMO-01, TST-DEMO-02",
        "A prose criterion with no anchor at all"
      ]
    }
  ]
}
EOF

FAILS=0
# run <label> [ENV=VAL ...] → sets $JSON to the checker's JSON block
run() {
    local label="$1"; shift
    local out
    out=$(env "$@" /bin/bash "$CHECK" DEMO 2>&1)
    JSON=$(echo "$out" | sed -n '/--- CRITERIA_JSON_START ---/,/--- CRITERIA_JSON_END ---/p' | sed '1d;$d')
    if [ -z "$JSON" ] || ! echo "$JSON" | jq . >/dev/null 2>&1; then
        echo "FAIL [$label]: no parseable JSON"; echo "$out"; FAILS=$((FAILS + 1)); JSON='{}'
    fi
}
expect() {  # <label> <criterion index> <status>
    local got
    got=$(echo "$JSON" | jq -r ".features[0].criteria[$2].status")
    if [ "$got" = "$3" ]; then
        echo "ok   [$1] criterion $2 = $3"
    else
        echo "FAIL [$1] criterion $2: want $3, got $got ($(echo "$JSON" | jq -r ".features[0].criteria[$2].evidence"))"
        FAILS=$((FAILS + 1))
    fi
}

run default AUTOMATED_TESTS_PASS=unknown
expect default 0 met            # comment above TestPass → runs TestPass
expect default 1 missing        # TestFail runs red
expect default 2 missing        # no carrier (design_docs quote excluded)
expect default 3 met            # ID inside a body → enclosing TestBody
expect default 4 unverifiable   # shell carrier, no suite result
expect default 5 missing        # one of two IDs red → criterion missing
expect default 6 unverifiable   # prose path unchanged
got=$(echo "$JSON" | jq -c '.tst_uncovered')
if [ "$got" = '["TST-DEMO-03","TST-DEMO-06"]' ]; then
    echo "ok   [default] tst_uncovered = $got"
else
    echo "FAIL [default] tst_uncovered: want [\"TST-DEMO-03\",\"TST-DEMO-06\"], got $got"; FAILS=$((FAILS + 1))
fi
[ "$(echo "$JSON" | jq '.tst_declared')" = 6 ] || { echo "FAIL [default] tst_declared != 6"; FAILS=$((FAILS + 1)); }

run suite-green AUTOMATED_TESTS_PASS=true
expect suite-green 4 met        # shell carrier + green suite

run no-run ACCEPTANCE_RUN_TESTS=0 AUTOMATED_TESTS_PASS=true
expect no-run 1 met             # not running tests → trusts the suite result

if [ "$FAILS" -gt 0 ]; then echo "$FAILS failure(s)"; exit 1; fi
echo "all passed"
