#!/bin/bash
# check_acceptance_criteria.sh - Verify each acceptance criterion from sprint JSON
# Outputs per-criterion pass/fail as JSON for the evaluator skill

set -e

SPRINT_ID="${1:-}"

if [ -z "$SPRINT_ID" ]; then
    echo "Usage: $0 <sprint-id>"
    echo ""
    echo "Example: $0 M-CACHE"
    echo ""
    echo "Checks acceptance criteria from sprint JSON and outputs results"
    exit 1
fi

SPRINT_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"

if [ ! -f "$SPRINT_FILE" ]; then
    echo "Error: Sprint file not found: $SPRINT_FILE"
    exit 1
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Acceptance Criteria Check: $SPRINT_ID"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Count features and criteria
FEATURE_COUNT=$(jq '.features | length' "$SPRINT_FILE")
TOTAL_CRITERIA=0
CRITERIA_MET=0
FEATURES_PASSING=0

echo "Features in sprint: $FEATURE_COUNT"
echo ""

# Check each feature
for i in $(seq 0 $((FEATURE_COUNT - 1))); do
    FEATURE_ID=$(jq -r ".features[$i].id" "$SPRINT_FILE")
    FEATURE_DESC=$(jq -r ".features[$i].description" "$SPRINT_FILE")
    FEATURE_PASSES=$(jq -r ".features[$i].passes" "$SPRINT_FILE")
    CRITERIA_COUNT=$(jq ".features[$i].acceptance_criteria | length" "$SPRINT_FILE")

    TOTAL_CRITERIA=$((TOTAL_CRITERIA + CRITERIA_COUNT))

    echo "── Feature: $FEATURE_ID ─────────────────────────────────────"
    echo "   Description: $FEATURE_DESC"
    echo "   Passes: $FEATURE_PASSES"
    echo "   Criteria: $CRITERIA_COUNT"

    if [ "$FEATURE_PASSES" = "true" ]; then
        FEATURES_PASSING=$((FEATURES_PASSING + 1))
        CRITERIA_MET=$((CRITERIA_MET + CRITERIA_COUNT))
        echo "   Status: ✅ All criteria assumed met (passes=true)"
    elif [ "$FEATURE_PASSES" = "false" ]; then
        echo "   Status: ❌ Feature failed (passes=false)"
    else
        echo "   Status: ⏳ Not evaluated (passes=null)"
    fi

    # List each criterion
    for j in $(seq 0 $((CRITERIA_COUNT - 1))); do
        CRITERION=$(jq -r ".features[$i].acceptance_criteria[$j]" "$SPRINT_FILE")
        if [ "$FEATURE_PASSES" = "true" ]; then
            echo "     ✅ $CRITERION"
        else
            echo "     ❌ $CRITERION"
        fi
    done
    echo ""
done

# Calculate score
if [ "$TOTAL_CRITERIA" -gt 0 ]; then
    CRITERIA_PCT=$((CRITERIA_MET * 100 / TOTAL_CRITERIA))
    CRITERIA_SCORE=$((CRITERIA_MET * 30 / TOTAL_CRITERIA))
else
    CRITERIA_PCT=0
    CRITERIA_SCORE=0
fi

HARD_FAIL=false
if [ "$CRITERIA_PCT" -lt 50 ]; then
    HARD_FAIL=true
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Acceptance Criteria Summary"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Features passing: $FEATURES_PASSING / $FEATURE_COUNT"
echo "  Criteria met:     $CRITERIA_MET / $TOTAL_CRITERIA ($CRITERIA_PCT%)"
echo "  Score:            $CRITERIA_SCORE / 30"
echo "  Hard fail:        $([ "$HARD_FAIL" = true ] && echo "❌ YES (<50% criteria met)" || echo "✅ NO")"
echo ""

# Output JSON
echo "--- CRITERIA_JSON_START ---"
cat <<EOF
{
  "sprint_id": "$SPRINT_ID",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "features_total": $FEATURE_COUNT,
  "features_passing": $FEATURES_PASSING,
  "criteria_total": $TOTAL_CRITERIA,
  "criteria_met": $CRITERIA_MET,
  "criteria_pct": $CRITERIA_PCT,
  "criteria_score": $CRITERIA_SCORE,
  "hard_fail": $HARD_FAIL
}
EOF
echo "--- CRITERIA_JSON_END ---"
