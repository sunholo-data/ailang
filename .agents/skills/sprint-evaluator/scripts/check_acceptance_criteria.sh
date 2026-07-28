#!/bin/bash
# check_acceptance_criteria.sh — Independently verify each acceptance criterion.
#
# This script does NOT trust the sprint JSON's `passes` field (that's written
# by the generator, which is what we're evaluating). Instead it pattern-matches
# common criterion shapes and runs concrete checks against the working tree:
#
#   - "CHANGELOG.md has <version> ..."  → grep changelogs/
#   - "examples/<name>.ail created"     → [ -f examples/<name>.ail ]
#   - "make test ... pass"              → read AUTOMATED_TESTS_PASS env
#   - "<file>.go has <X>"               → grep the file
#   - "git tag present"                 → git tag | grep
#   - prose criteria with no verifiable anchor → "unverifiable"
#
# Each criterion becomes one of:
#   ✅ verified-met      (concrete proof found)
#   ❌ verified-missing  (concrete proof absent)
#   ⚠️  unverifiable     (no machine-checkable anchor in the criterion text)
#
# Scoring (strict): points awarded only for verified-met.
#   score = (verified_met / total_criteria) * 30
# Unverifiable criteria count AGAINST the score — the generator's job is to
# write criteria that CAN be checked, or to provide concrete evidence.

set -e

SPRINT_ID="${1:-}"
# Optional env inputs from evaluate_sprint.sh:
AUTOMATED_TESTS_PASS="${AUTOMATED_TESTS_PASS:-unknown}"
AUTOMATED_LINT_CLEAN="${AUTOMATED_LINT_CLEAN:-unknown}"

if [ -z "$SPRINT_ID" ]; then
    echo "Usage: $0 <sprint-id>"
    echo ""
    echo "Environment:"
    echo "  AUTOMATED_TESTS_PASS=true|false   (from evaluate_sprint.sh)"
    echo "  AUTOMATED_LINT_CLEAN=true|false   (from evaluate_sprint.sh)"
    exit 1
fi

SPRINT_FILE=".ailang/state/sprints/sprint_${SPRINT_ID}.json"
if [ ! -f "$SPRINT_FILE" ]; then
    echo "Error: Sprint file not found: $SPRINT_FILE"
    exit 1
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Acceptance Criteria Check: $SPRINT_ID  (independent verification)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Mode: evaluator does NOT trust sprint JSON 'passes' field"
echo ""

# ────────────────────────────────────────────────────────────────
# Criterion verification — returns one of: met / missing / unverifiable
# plus a short evidence string.
#
# Input:  criterion text on stdin-like argument
# Output: "<status>|<evidence>"
# ────────────────────────────────────────────────────────────────
verify_criterion() {
    local c="$1"
    local lc
    lc=$(echo "$c" | tr '[:upper:]' '[:lower:]')

    # ── File existence claims ──────────────────────────────────────
    # "examples/foo.ail created", "examples/foo.ail and bar.ail created"
    # Collect every examples/*.ail pattern mentioned.
    local example_files
    example_files=$(echo "$c" | grep -oE 'examples/[a-zA-Z0-9_/.-]+\.ail' || true)
    if [ -n "$example_files" ]; then
        local all_found=true
        local found_list=""
        local missing_list=""
        while IFS= read -r f; do
            if [ -f "$f" ]; then
                found_list="$found_list $f"
            else
                all_found=false
                missing_list="$missing_list $f"
            fi
        done <<< "$example_files"
        if $all_found; then
            echo "met|files exist:$found_list"
        else
            echo "missing|files absent:$missing_list"
        fi
        return
    fi

    # ── "make test" / "make verify-examples" claims ────────────────
    if echo "$lc" | grep -qE "make test.*pass|make verify-examples.*pass|all tests.*pass|tests.*green"; then
        if [ "$AUTOMATED_TESTS_PASS" = "true" ]; then
            echo "met|automated_checks.tests_pass=true"
        elif [ "$AUTOMATED_TESTS_PASS" = "false" ]; then
            echo "missing|automated_checks.tests_pass=false"
        else
            echo "unverifiable|tests_pass not reported by evaluator runner"
        fi
        return
    fi

    # ── "lint ... clean" / "no lint" claims ────────────────────────
    if echo "$lc" | grep -qE "lint.*clean|no lint|linting.*pass"; then
        if [ "$AUTOMATED_LINT_CLEAN" = "true" ]; then
            echo "met|automated_checks.lint_clean=true"
        elif [ "$AUTOMATED_LINT_CLEAN" = "false" ]; then
            echo "missing|automated_checks.lint_clean=false"
        else
            echo "unverifiable|lint_clean not reported"
        fi
        return
    fi

    # ── CHANGELOG claims ───────────────────────────────────────────
    # "CHANGELOG.md has v0.11.3 section ..."  → grep changelogs/ + CHANGELOG.md
    local cl_version
    cl_version=$(echo "$c" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+[a-zA-Z0-9.+-]*' | head -1 || true)
    if echo "$lc" | grep -qE "changelog"; then
        if [ -n "$cl_version" ]; then
            if grep -rqF "$cl_version" changelogs/ CHANGELOG.md 2>/dev/null; then
                echo "met|$cl_version in changelogs/"
            else
                echo "missing|$cl_version not found in changelogs/ or CHANGELOG.md"
            fi
        else
            # Generic "CHANGELOG updated" — check it was modified in recent commits
            if git log --since="14 days ago" --name-only --pretty=format: 2>/dev/null | grep -q "CHANGELOG\|changelogs/"; then
                echo "met|CHANGELOG modified in recent commits"
            else
                echo "unverifiable|no version mentioned and no recent CHANGELOG commits"
            fi
        fi
        return
    fi

    # ── Git tag claims ─────────────────────────────────────────────
    if echo "$lc" | grep -qE "git tag.*present|tag.*created|version.*tagged"; then
        if [ -n "$cl_version" ] && git tag 2>/dev/null | grep -qF "$cl_version"; then
            echo "met|git tag $cl_version present"
        elif [ -n "$cl_version" ]; then
            echo "missing|git tag $cl_version NOT found"
        else
            echo "unverifiable|no version extracted from criterion"
        fi
        return
    fi

    # ── GitHub release claims ──────────────────────────────────────
    if echo "$lc" | grep -qE "github release.*created|release.*published"; then
        if [ -n "$cl_version" ] && command -v gh >/dev/null 2>&1; then
            if gh release view "$cl_version" >/dev/null 2>&1; then
                echo "met|gh release $cl_version exists"
            else
                echo "missing|gh release $cl_version NOT found"
            fi
        else
            echo "unverifiable|no gh CLI or no version"
        fi
        return
    fi

    # ── Design doc move claims ─────────────────────────────────────
    # "Design docs moved from planned/v0_11_3/ to implemented/v0_11_3/"
    if echo "$lc" | grep -qE "design doc.*moved.*implemented|moved.*planned.*implemented"; then
        local dest_path
        dest_path=$(echo "$c" | grep -oE 'implemented/v[0-9_]+' | head -1 || true)
        local src_path
        src_path=$(echo "$c" | grep -oE 'planned/v[0-9_]+' | head -1 || true)
        local dest_ok=false
        local src_empty=false
        [ -n "$dest_path" ] && [ -d "design_docs/$dest_path" ] && [ "$(ls -A "design_docs/$dest_path" 2>/dev/null)" ] && dest_ok=true
        if [ -n "$src_path" ]; then
            if [ ! -d "design_docs/$src_path" ] || [ -z "$(ls -A "design_docs/$src_path" 2>/dev/null)" ]; then
                src_empty=true
            fi
        else
            src_empty=true
        fi
        if $dest_ok && $src_empty; then
            echo "met|dest populated, src empty/gone"
        elif $dest_ok; then
            echo "missing|dest populated but src not cleaned"
        else
            echo "missing|dest dir empty or absent"
        fi
        return
    fi

    # ── Messages ack claims ────────────────────────────────────────
    # "ailang messages ack ce6e078e"  → check via ailang messages list
    local msg_ids
    msg_ids=$(echo "$c" | grep -oE '[a-f0-9]{8}' | head -3 || true)
    if echo "$lc" | grep -qE "messages ack|msg.*ack|ack.*msg"; then
        if [ -n "$msg_ids" ] && command -v ailang >/dev/null 2>&1; then
            local unread
            unread=$(ailang messages list --unread --compact 2>/dev/null || echo "")
            local any_unread=false
            for mid in $msg_ids; do
                if echo "$unread" | grep -qF "$mid"; then
                    any_unread=true
                fi
            done
            if $any_unread; then
                echo "missing|one or more msg IDs still unread:$msg_ids"
            else
                echo "met|msg IDs not in unread list:$msg_ids"
            fi
        else
            echo "unverifiable|no msg IDs found or ailang CLI unavailable"
        fi
        return
    fi

    # ── Grep-for-symbol claims ─────────────────────────────────────
    # Extract plausible code identifiers: camelCase or snake_case with ≥2
    # internal breaks (to avoid matching random English words like "Repro").
    # A token qualifies if it has at least one lowercase→uppercase transition
    # (camelCase) OR contains an underscore with alnum on both sides.
    local symbols
    symbols=$(echo "$c" | grep -oE '[A-Za-z_][A-Za-z0-9_]{3,}' \
        | grep -E '([a-z][A-Z]|_[a-zA-Z0-9])' \
        | grep -vE '^(examples|planned|implemented|passes|returns|breaks|fails|panics|handles|changed|created|verified|tested|exists)$' \
        | head -3 || true)
    if [ -n "$symbols" ] && echo "$lc" | grep -qE "desugar|implement|return|export|panic|reject|register|wire|builtin|handler|variant|scanf|parsef|scansF|parsesF|scansf|parsesf"; then
        local all_found=true
        local found_list=""
        local missing_list=""
        while IFS= read -r sym; do
            [ -z "$sym" ] && continue
            if grep -rq --include="*.go" --include="*.ail" -- "$sym" internal/ std/ 2>/dev/null; then
                found_list="$found_list $sym"
            else
                all_found=false
                missing_list="$missing_list $sym"
            fi
        done <<< "$symbols"
        if $all_found && [ -n "$found_list" ]; then
            echo "met|symbols found:$found_list"
        elif [ -z "$missing_list" ]; then
            echo "unverifiable|no identifier-shaped tokens in criterion"
        else
            echo "missing|symbols absent:$missing_list"
        fi
        return
    fi

    # ── Teaching-prompt claims ─────────────────────────────────────
    # Only count as met if prompts/ was touched IN THE SPRINT WINDOW (since
    # the sprint's earliest `started` timestamp) — not the loose "last 14 days".
    if echo "$lc" | grep -qE "teaching prompt|prompts/|agent-prompt"; then
        if [ ! -d prompts ]; then
            echo "unverifiable|no prompts/ directory"
            return
        fi
        local sprint_start
        sprint_start=$(jq -r '[.features[].started | select(. != null)] | min // empty' "$SPRINT_FILE" 2>/dev/null || true)
        if [ -n "$sprint_start" ] && [ "$sprint_start" != "null" ]; then
            if git log --since="$sprint_start" --name-only --pretty=format: 2>/dev/null | grep -q "^prompts/"; then
                echo "met|prompts/ modified since sprint start ($sprint_start)"
            else
                echo "missing|prompts/ NOT modified since sprint start ($sprint_start)"
            fi
        else
            # No sprint start timestamp — fall back to uncommitted check only
            if git diff --name-only HEAD 2>/dev/null | grep -q "^prompts/" \
               || git status --porcelain 2>/dev/null | grep -q " prompts/"; then
                echo "met|prompts/ has uncommitted changes"
            else
                echo "missing|prompts/ not in working-tree diff and no sprint start to scope git log"
            fi
        fi
        return
    fi

    # ── release-manager invocation / process claims (unverifiable) ──
    if echo "$lc" | grep -qE "release-manager.*invoked|install script.*updated"; then
        # install script update check
        if echo "$lc" | grep -q "install script"; then
            if git log --since="14 days ago" --name-only --pretty=format: 2>/dev/null | grep -qE "install.*sh|install.*script"; then
                echo "met|install script touched recently"
            else
                echo "missing|no recent install script changes"
            fi
            return
        fi
        echo "unverifiable|release process claim — human must verify"
        return
    fi

    # ── Fall-through: prose criterion, no machine anchor ───────────
    echo "unverifiable|no recognizable pattern (prose criterion)"
}

# ────────────────────────────────────────────────────────────────
# Main loop
# ────────────────────────────────────────────────────────────────
FEATURE_COUNT=$(jq '.features | length' "$SPRINT_FILE")
TOTAL_CRITERIA=0
CRITERIA_MET=0
CRITERIA_MISSING=0
CRITERIA_UNVERIFIABLE=0
FEATURES_FULLY_VERIFIED=0

echo "Features in sprint: $FEATURE_COUNT"
echo ""

# Collect per-feature JSON fragments
FEATURE_JSON_FRAGS=""

for i in $(seq 0 $((FEATURE_COUNT - 1))); do
    FEATURE_ID=$(jq -r ".features[$i].id" "$SPRINT_FILE")
    FEATURE_DESC=$(jq -r ".features[$i].description" "$SPRINT_FILE")
    FEATURE_PASSES_CLAIMED=$(jq -r ".features[$i].passes" "$SPRINT_FILE")
    CRITERIA_COUNT=$(jq ".features[$i].acceptance_criteria | length" "$SPRINT_FILE")

    TOTAL_CRITERIA=$((TOTAL_CRITERIA + CRITERIA_COUNT))

    echo "── Feature: $FEATURE_ID ─────────────────────────────────────"
    echo "   Description: $(echo "$FEATURE_DESC" | head -c 80)"
    echo "   Criteria: $CRITERIA_COUNT   (JSON claims passes=$FEATURE_PASSES_CLAIMED — ignored)"
    echo ""

    F_MET=0
    F_MISSING=0
    F_UNVERIFIABLE=0
    CRITERIA_JSON_FRAGS=""

    for j in $(seq 0 $((CRITERIA_COUNT - 1))); do
        CRITERION=$(jq -r ".features[$i].acceptance_criteria[$j]" "$SPRINT_FILE")
        RESULT=$(verify_criterion "$CRITERION")
        STATUS="${RESULT%%|*}"
        EVIDENCE="${RESULT#*|}"

        case "$STATUS" in
            met)
                echo "     ✅ $CRITERION"
                echo "        └─ $EVIDENCE"
                F_MET=$((F_MET + 1))
                CRITERIA_MET=$((CRITERIA_MET + 1))
                ;;
            missing)
                echo "     ❌ $CRITERION"
                echo "        └─ $EVIDENCE"
                F_MISSING=$((F_MISSING + 1))
                CRITERIA_MISSING=$((CRITERIA_MISSING + 1))
                ;;
            unverifiable|*)
                echo "     ⚠️  $CRITERION"
                echo "        └─ $EVIDENCE"
                F_UNVERIFIABLE=$((F_UNVERIFIABLE + 1))
                CRITERIA_UNVERIFIABLE=$((CRITERIA_UNVERIFIABLE + 1))
                ;;
        esac

        # Escape criterion + evidence for JSON
        CRIT_ESC=$(printf '%s' "$CRITERION" | jq -Rs .)
        EVID_ESC=$(printf '%s' "$EVIDENCE" | jq -Rs .)
        FRAG=$(printf '{"criterion": %s, "status": "%s", "evidence": %s}' "$CRIT_ESC" "$STATUS" "$EVID_ESC")
        if [ -z "$CRITERIA_JSON_FRAGS" ]; then
            CRITERIA_JSON_FRAGS="$FRAG"
        else
            CRITERIA_JSON_FRAGS="$CRITERIA_JSON_FRAGS,$FRAG"
        fi
    done

    echo ""
    echo "   Feature tally: ✅ $F_MET  ❌ $F_MISSING  ⚠️  $F_UNVERIFIABLE"
    if [ "$F_MISSING" -eq 0 ] && [ "$F_UNVERIFIABLE" -eq 0 ] && [ "$F_MET" -eq "$CRITERIA_COUNT" ]; then
        FEATURES_FULLY_VERIFIED=$((FEATURES_FULLY_VERIFIED + 1))
        echo "   Status: ✅ All criteria independently verified"
    elif [ "$F_MISSING" -gt 0 ]; then
        echo "   Status: ❌ Concrete gaps — $F_MISSING criterion(s) failed verification"
    else
        echo "   Status: ⚠️  Partially verified — $F_UNVERIFIABLE prose criterion(s) need human review"
    fi
    echo ""

    FEAT_FRAG=$(cat <<EOF
{"id": "$FEATURE_ID", "criteria_total": $CRITERIA_COUNT, "verified_met": $F_MET, "verified_missing": $F_MISSING, "unverifiable": $F_UNVERIFIABLE, "criteria": [$CRITERIA_JSON_FRAGS]}
EOF
)
    if [ -z "$FEATURE_JSON_FRAGS" ]; then
        FEATURE_JSON_FRAGS="$FEAT_FRAG"
    else
        FEATURE_JSON_FRAGS="$FEATURE_JSON_FRAGS,$FEAT_FRAG"
    fi
done

# Scoring (strict): only verified-met counts toward the 30-point allocation.
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
echo " Acceptance Criteria Summary  (independent verification)"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Features fully verified:  $FEATURES_FULLY_VERIFIED / $FEATURE_COUNT"
echo "  Criteria verified met:    $CRITERIA_MET / $TOTAL_CRITERIA ($CRITERIA_PCT%)"
echo "  Criteria verified missing: $CRITERIA_MISSING"
echo "  Criteria unverifiable:    $CRITERIA_UNVERIFIABLE  (prose — need human judgment)"
echo "  Score:                    $CRITERIA_SCORE / 30"
echo "  Hard fail:                $([ "$HARD_FAIL" = true ] && echo "❌ YES (<50% verified met)" || echo "✅ NO")"
echo ""
if [ "$CRITERIA_UNVERIFIABLE" -gt 0 ]; then
    echo "  Note: ⚠️  criteria count as NOT met — they require either:"
    echo "    1. Reworded to include a machine-checkable anchor, OR"
    echo "    2. Human review to mark them verified"
    echo ""
fi

echo "--- CRITERIA_JSON_START ---"
cat <<EOF
{
  "sprint_id": "$SPRINT_ID",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "mode": "independent_verification",
  "features_total": $FEATURE_COUNT,
  "features_fully_verified": $FEATURES_FULLY_VERIFIED,
  "criteria_total": $TOTAL_CRITERIA,
  "criteria_verified_met": $CRITERIA_MET,
  "criteria_verified_missing": $CRITERIA_MISSING,
  "criteria_unverifiable": $CRITERIA_UNVERIFIABLE,
  "criteria_pct": $CRITERIA_PCT,
  "criteria_score": $CRITERIA_SCORE,
  "hard_fail": $HARD_FAIL,
  "features": [$FEATURE_JSON_FRAGS]
}
EOF
echo "--- CRITERIA_JSON_END ---"
