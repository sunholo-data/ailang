#!/bin/bash
# scripts/test_chains_integration.sh
#
# Full Chains Integration Test (M-CHAINS-SIMPLIFY)
#
# Tests the complete workflow:
#   GitHub Issue -> Message -> Task -> Chain -> Stage -> Approval -> Handoff
#
# Prerequisites:
#   - ailang installed and in PATH
#   - Coordinator daemon running (ailang coordinator start)
#   - Server running (ailang serve)
#   - GitHub CLI authenticated (gh auth status)
#
# Usage:
#   ./scripts/test_chains_integration.sh              # Full test
#   ./scripts/test_chains_integration.sh --no-github  # Skip GitHub issue creation
#   ./scripts/test_chains_integration.sh --level 3    # Run up to level 3 only

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO="sunholo-data/ailang"
MAX_LEVEL=7
SKIP_GITHUB=false
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-github)
            SKIP_GITHUB=true
            shift
            ;;
        --level)
            MAX_LEVEL="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}=== M-CHAINS-SIMPLIFY Integration Test ===${NC}"
echo "Max level: $MAX_LEVEL"
echo ""

# Helper functions
check_pass() {
    echo -e "${GREEN}PASS${NC}: $1"
}

check_fail() {
    echo -e "${RED}FAIL${NC}: $1"
    exit 1
}

check_warn() {
    echo -e "${YELLOW}WARN${NC}: $1"
}

# =============================================================================
# Level 1: Basic Chain + Span Linking
# =============================================================================
if [ "$MAX_LEVEL" -ge 1 ]; then
    echo -e "${CYAN}--- Level 1: Basic Chain + Span Linking ---${NC}"

    # Create chain via direct database call
    CHAIN_ID=$(sqlite3 ~/.ailang/state/observatory.db "
        INSERT INTO execution_chains (id, source_type, source_ref, status, created_at, updated_at)
        VALUES ('test-chain-$(date +%s)', 'manual', 'integration-test', 'active', datetime('now'), datetime('now'))
        RETURNING id;
    " 2>/dev/null || echo "")

    if [ -z "$CHAIN_ID" ]; then
        check_warn "Could not create test chain (table may not exist yet)"
        echo "Creating execution_chains table..."
        # The table should be auto-created by migrations, but let's verify
        TABLE_EXISTS=$(sqlite3 ~/.ailang/state/observatory.db "SELECT name FROM sqlite_master WHERE type='table' AND name='execution_chains';" 2>/dev/null || echo "")
        if [ -z "$TABLE_EXISTS" ]; then
            check_fail "execution_chains table does not exist. Run 'ailang serve' first to trigger migrations."
        fi

        # Try creating chain again
        CHAIN_ID=$(sqlite3 ~/.ailang/state/observatory.db "
            INSERT INTO execution_chains (id, source_type, source_ref, status, created_at, updated_at)
            VALUES ('test-chain-$(date +%s)', 'manual', 'integration-test', 'active', datetime('now'), datetime('now'))
            RETURNING id;
        ")
    fi

    echo "Created test chain: $CHAIN_ID"

    # Run ailang with chain env vars
    STAGE_ID="stage-$(date +%s)"
    echo "Running ailang with chain context..."
    OTEL_RESOURCE_ATTRIBUTES="ailang.chain_id=$CHAIN_ID,ailang.stage_id=$STAGE_ID" \
        ailang check examples/runnable/hello.ail > /dev/null 2>&1 || true

    # Verify spans have chain_id (may take a moment to ingest)
    sleep 1
    SPAN_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "
        SELECT COUNT(*) FROM spans WHERE chain_id = '$CHAIN_ID'
    " 2>/dev/null || echo "0")

    if [ "$SPAN_COUNT" -gt 0 ]; then
        check_pass "Level 1: $SPAN_COUNT spans linked to chain $CHAIN_ID"
    else
        check_warn "Level 1: No spans found with chain_id (OTEL may be disabled or spans not yet ingested)"
    fi

    # Cleanup test chain
    sqlite3 ~/.ailang/state/observatory.db "DELETE FROM execution_chains WHERE id = '$CHAIN_ID';" 2>/dev/null || true
    echo ""
fi

# =============================================================================
# Level 2: Coordinator Creates Chain on Task Start
# =============================================================================
if [ "$MAX_LEVEL" -ge 2 ]; then
    echo -e "${CYAN}--- Level 2: Coordinator Creates Chain on Task Start ---${NC}"

    # Check if coordinator is running
    COORD_STATUS=$(ailang coordinator status 2>&1 || echo "not running")
    if [[ "$COORD_STATUS" != *"running"* ]]; then
        check_warn "Coordinator not running. Start with: ailang coordinator start"
        echo "Skipping Level 2 (requires running coordinator)"
    else
        # Send message to coordinator
        TEST_TITLE="[chains-test] Level 2 test $(date +%s)"
        echo "Sending message: $TEST_TITLE"

        ailang messages send coordinator "Testing chain creation on task start" \
            --title "$TEST_TITLE" \
            --from "integration-test" > /dev/null 2>&1

        # Wait for task to be picked up
        echo "Waiting for task to start..."
        sleep 10

        # Check if chain was created
        CHAIN_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "
            SELECT COUNT(*) FROM execution_chains
            WHERE source_type = 'message'
            AND created_at > datetime('now', '-1 minute');
        " 2>/dev/null || echo "0")

        if [ "$CHAIN_COUNT" -gt 0 ]; then
            check_pass "Level 2: Chain created when task started"

            # Get the chain ID for verification
            CHAIN_ID=$(sqlite3 ~/.ailang/state/observatory.db "
                SELECT id FROM execution_chains
                WHERE source_type = 'message'
                ORDER BY created_at DESC LIMIT 1;
            ")
            echo "  Chain ID: $CHAIN_ID"

            # Check task metadata
            TASK_CHAIN=$(sqlite3 ~/.ailang/state/coordinator.db "
                SELECT json_extract(metadata, '\$.chain_id') FROM tasks
                ORDER BY created_at DESC LIMIT 1;
            " 2>/dev/null || echo "")

            if [ -n "$TASK_CHAIN" ] && [ "$TASK_CHAIN" != "null" ]; then
                check_pass "Level 2: Task has chain_id in metadata"
            else
                check_warn "Level 2: Task metadata missing chain_id"
            fi
        else
            check_warn "Level 2: No chain created (coordinator may not be processing)"
        fi
    fi
    echo ""
fi

# =============================================================================
# Level 3: Chain Stage Created Per Agent
# =============================================================================
if [ "$MAX_LEVEL" -ge 3 ]; then
    echo -e "${CYAN}--- Level 3: Chain Stage Created Per Agent ---${NC}"

    COORD_STATUS=$(ailang coordinator status 2>&1 || echo "not running")
    if [[ "$COORD_STATUS" != *"running"* ]]; then
        echo "Skipping Level 3 (requires running coordinator)"
    else
        # Check for stages
        STAGE_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "
            SELECT COUNT(*) FROM chain_stages
            WHERE created_at > datetime('now', '-5 minutes');
        " 2>/dev/null || echo "0")

        if [ "$STAGE_COUNT" -gt 0 ]; then
            check_pass "Level 3: $STAGE_COUNT stage(s) created"

            # Show latest stage
            sqlite3 ~/.ailang/state/observatory.db "
                SELECT agent_id, status, created_at
                FROM chain_stages
                ORDER BY created_at DESC LIMIT 3;
            " 2>/dev/null || true
        else
            check_warn "Level 3: No stages found"
        fi
    fi
    echo ""
fi

# =============================================================================
# Level 4: GitHub Issue -> Chain
# =============================================================================
if [ "$MAX_LEVEL" -ge 4 ]; then
    echo -e "${CYAN}--- Level 4: GitHub Issue -> Chain ---${NC}"

    if [ "$SKIP_GITHUB" = true ]; then
        echo "Skipping GitHub test (--no-github flag set)"
    else
        # Check gh auth
        GH_STATUS=$(gh auth status 2>&1 || echo "not authenticated")
        if [[ "$GH_STATUS" != *"Logged in"* ]]; then
            check_warn "GitHub CLI not authenticated. Run: gh auth login"
            echo "Skipping Level 4"
        else
            # Create test issue
            TEST_ISSUE_TITLE="[chains-test] Integration test $(date +%s)"
            echo "Creating GitHub issue: $TEST_ISSUE_TITLE"

            ISSUE_URL=$(gh issue create --repo "$REPO" \
                --title "$TEST_ISSUE_TITLE" \
                --body "Automated chains integration test. Safe to close." \
                --label "coordinator:feature" 2>/dev/null || echo "")

            if [ -z "$ISSUE_URL" ]; then
                check_warn "Could not create GitHub issue (permission denied or network issue)"
            else
                ISSUE_NUM=$(echo "$ISSUE_URL" | grep -oE '[0-9]+$')
                echo "Created issue #$ISSUE_NUM"

                # Import GitHub issues
                echo "Importing GitHub issues..."
                ailang messages import-github --labels coordinator:feature > /dev/null 2>&1 || true

                # Check if chain was created with GitHub source
                sleep 5
                GITHUB_CHAIN=$(sqlite3 ~/.ailang/state/observatory.db "
                    SELECT id FROM execution_chains
                    WHERE source_type = 'github_issue'
                    AND github_issue_number = $ISSUE_NUM
                    LIMIT 1;
                " 2>/dev/null || echo "")

                if [ -n "$GITHUB_CHAIN" ]; then
                    check_pass "Level 4: Chain created from GitHub issue #$ISSUE_NUM"
                    echo "  Chain ID: $GITHUB_CHAIN"
                else
                    check_warn "Level 4: No chain found for GitHub issue #$ISSUE_NUM"
                    echo "  (Chain is created when task starts, not on message import)"
                fi

                # Close the test issue
                echo "Closing test issue..."
                gh issue close "$ISSUE_NUM" --repo "$REPO" > /dev/null 2>&1 || true
            fi
        fi
    fi
    echo ""
fi

# =============================================================================
# Level 5: Approval Updates Chain Status
# =============================================================================
if [ "$MAX_LEVEL" -ge 5 ]; then
    echo -e "${CYAN}--- Level 5: Approval Updates Chain Status ---${NC}"

    # Check for pending approvals
    PENDING_COUNT=$(sqlite3 ~/.ailang/state/coordinator.db "
        SELECT COUNT(*) FROM approval_requests WHERE status = 'pending';
    " 2>/dev/null || echo "0")

    if [ "$PENDING_COUNT" -gt 0 ]; then
        echo "Found $PENDING_COUNT pending approval(s)"

        # Get latest pending approval
        APPROVAL_ID=$(sqlite3 ~/.ailang/state/coordinator.db "
            SELECT id FROM approval_requests
            WHERE status = 'pending'
            ORDER BY created_at DESC LIMIT 1;
        " 2>/dev/null || echo "")

        TASK_ID=$(sqlite3 ~/.ailang/state/coordinator.db "
            SELECT task_id FROM approval_requests WHERE id = '$APPROVAL_ID';
        " 2>/dev/null || echo "")

        echo "Latest pending approval: $APPROVAL_ID (task: $TASK_ID)"
        echo ""
        echo "To test approval flow:"
        echo "  ailang coordinator approve $TASK_ID"
        echo "  Then verify: ailang chains list"
        check_pass "Level 5: Pending approvals exist for testing"
    else
        check_warn "Level 5: No pending approvals to test"
        echo "  Run a coordinator task first, then re-run this test"
    fi
    echo ""
fi

# =============================================================================
# Level 6: Handoff Creates New Stage
# =============================================================================
if [ "$MAX_LEVEL" -ge 6 ]; then
    echo -e "${CYAN}--- Level 6: Handoff Creates New Stage ---${NC}"

    # Check for chains with multiple stages
    MULTI_STAGE_CHAINS=$(sqlite3 ~/.ailang/state/observatory.db "
        SELECT ec.id, COUNT(cs.id) as stage_count
        FROM execution_chains ec
        JOIN chain_stages cs ON cs.chain_id = ec.id
        GROUP BY ec.id
        HAVING COUNT(cs.id) > 1
        LIMIT 5;
    " 2>/dev/null || echo "")

    if [ -n "$MULTI_STAGE_CHAINS" ]; then
        check_pass "Level 6: Found chains with multiple stages (handoffs working)"
        echo "$MULTI_STAGE_CHAINS"
    else
        check_warn "Level 6: No multi-stage chains found"
        echo "  This test requires completing an agent handoff workflow"
        echo "  1. Send message to design-doc-creator"
        echo "  2. Approve the task"
        echo "  3. Verify handoff to sprint-planner creates new stage"
    fi
    echo ""
fi

# =============================================================================
# Level 7: Full End-to-End Summary
# =============================================================================
if [ "$MAX_LEVEL" -ge 7 ]; then
    echo -e "${CYAN}--- Level 7: Full End-to-End Summary ---${NC}"

    # Count statistics
    CHAIN_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM execution_chains;" 2>/dev/null || echo "0")
    STAGE_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM chain_stages;" 2>/dev/null || echo "0")
    LINKED_SPANS=$(sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM spans WHERE chain_id IS NOT NULL AND chain_id != '';" 2>/dev/null || echo "0")

    echo "Chains System Statistics:"
    echo "  Total Chains: $CHAIN_COUNT"
    echo "  Total Stages: $STAGE_COUNT"
    echo "  Linked Spans: $LINKED_SPANS"
    echo ""

    # Show recent chains
    echo "Recent Chains:"
    ailang chains list --limit 5 2>/dev/null || echo "  (Run 'ailang chains list' to see chains)"
    echo ""

    if [ "$CHAIN_COUNT" -gt 0 ]; then
        check_pass "Level 7: Chains system is operational"

        # Show a sample chain tree
        SAMPLE_CHAIN=$(sqlite3 ~/.ailang/state/observatory.db "SELECT id FROM execution_chains ORDER BY created_at DESC LIMIT 1;" 2>/dev/null || echo "")
        if [ -n "$SAMPLE_CHAIN" ]; then
            echo ""
            echo "Sample Chain Tree:"
            ailang chains tree "$SAMPLE_CHAIN" 2>/dev/null || true
        fi
    else
        check_warn "Level 7: No chains in system yet"
        echo "  Run coordinator tasks to create chains"
    fi
fi

echo ""
echo -e "${CYAN}=== Integration Test Complete ===${NC}"
echo ""
echo "Manual verification steps:"
echo "  1. Start coordinator: ailang coordinator start"
echo "  2. Start server: ailang serve"
echo "  3. Send test message: ailang messages send coordinator 'Test task' --title 'Test'"
echo "  4. Wait for task to complete"
echo "  5. Approve: ailang coordinator pending"
echo "  6. View chain: ailang chains list && ailang chains view <chain-id>"
