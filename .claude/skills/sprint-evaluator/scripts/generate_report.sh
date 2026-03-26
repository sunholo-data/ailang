#!/bin/bash
# generate_report.sh - Create evaluation report JSON
# Combines automated check results with scoring into a structured report

set -e

SPRINT_ID="${1:-}"
SCORE="${2:-0}"
RESULT="${3:-fail}"
ROUND="${4:-1}"

if [ -z "$SPRINT_ID" ]; then
    echo "Usage: $0 <sprint-id> <score> <result> <round>"
    echo ""
    echo "Example: $0 M-CACHE 85 pass 1"
    echo ""
    echo "Creates evaluation report at .ailang/state/evaluations/"
    exit 1
fi

# Create evaluations directory if needed
EVAL_DIR=".ailang/state/evaluations"
mkdir -p "$EVAL_DIR"

REPORT_FILE="$EVAL_DIR/eval_${SPRINT_ID}_round_${ROUND}.json"
SPRINT_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"

# Load sprint metadata
DESIGN_DOC=""
SPRINT_PLAN=""
if [ -f "$SPRINT_FILE" ]; then
    DESIGN_DOC=$(jq -r '.design_doc // ""' "$SPRINT_FILE")
    SPRINT_PLAN=$(jq -r '.markdown_plan // .sprint_plan // ""' "$SPRINT_FILE")
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Generating Evaluation Report"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Sprint:  $SPRINT_ID"
echo "  Score:   $SCORE / 100"
echo "  Result:  $RESULT"
echo "  Round:   $ROUND"
echo "  Report:  $REPORT_FILE"
echo ""

# Generate the report
cat > "$REPORT_FILE" <<EOF
{
  "sprint_id": "$SPRINT_ID",
  "evaluation_round": $ROUND,
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "design_doc": "$DESIGN_DOC",
  "sprint_plan": "$SPRINT_PLAN",
  "total_score": $SCORE,
  "pass_threshold": 70,
  "result": "$RESULT",
  "score_breakdown": {
    "tests_pass": 0,
    "lint_clean": 0,
    "acceptance_criteria": 0,
    "code_quality": 0,
    "documentation": 0,
    "design_fidelity": 0
  },
  "hard_fails": [],
  "feedback": [],
  "notes": ""
}
EOF

echo "✅ Report written to: $REPORT_FILE"
echo ""
echo "Next steps:"
echo "  1. Update score_breakdown with actual per-category scores"
echo "  2. Add hard_fails array if any hard fail conditions triggered"
echo "  3. Add feedback array with specific issues (if result=fail)"
echo "  4. Update notes with evaluation summary"
echo ""
echo "The evaluator skill will update this file with detailed results."
