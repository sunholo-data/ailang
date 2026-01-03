#!/bin/bash
# Full E2E Integration Test for M-COORD-GITHUB-AUTO-ROUTING
# Tests the complete workflow: Issue → Design → Sprint → Implementation → PR
#
# Usage:
#   ./scripts/test_full_workflow.sh              # Interactive mode
#   ./scripts/test_full_workflow.sh --auto       # Auto-approve all stages
#   ./scripts/test_full_workflow.sh --dry-run    # Just show what would happen

set -e

REPO="sunholo-data/ailang"
AUTO_APPROVE=false
DRY_RUN=false
POLL_INTERVAL=10  # seconds between checks
MAX_WAIT=300      # 5 minutes max wait per stage

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --auto) AUTO_APPROVE=true; shift ;;
        --dry-run) DRY_RUN=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() { echo -e "${BLUE}[$(date +%H:%M:%S)]${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; }

echo ""
echo "═══════════════════════════════════════════════════════════════════════"
echo "           Full E2E Workflow Test - M-COORD-GITHUB-AUTO-ROUTING"
echo "═══════════════════════════════════════════════════════════════════════"
echo ""
echo "This test will:"
echo "  1. Create a GitHub issue with coordinator:feature label"
echo "  2. Wait for coordinator to pick it up and start design"
echo "  3. Wait for design doc, then approve → sprint planning"
echo "  4. Wait for sprint plan, then approve → implementation"
echo "  5. Wait for implementation, then approve → merge"
echo "  6. Verify PR is created"
echo ""
if [ "$AUTO_APPROVE" = true ]; then
    log "Mode: AUTO-APPROVE (will approve each stage automatically)"
else
    log "Mode: INTERACTIVE (will wait for manual approval at each stage)"
fi
if [ "$DRY_RUN" = true ]; then
    log "Mode: DRY-RUN (no actual changes)"
    exit 0
fi
echo ""

# Check prerequisites
log "Checking prerequisites..."

# Check gh auth
if ! gh auth status &>/dev/null; then
    error "Not authenticated to GitHub. Run: gh auth login"
    exit 1
fi
CURRENT_USER=$(gh api user -q '.login')
success "GitHub authenticated as: $CURRENT_USER"

# Check coordinator is running
if ! ./bin/ailang coordinator status 2>/dev/null | grep -q "running"; then
    warn "Coordinator not running. Starting it..."
    ./bin/ailang coordinator start &
    sleep 3
    if ! ./bin/ailang coordinator status 2>/dev/null | grep -q "running"; then
        error "Failed to start coordinator"
        exit 1
    fi
fi
success "Coordinator is running"
echo ""

# Create test issue
ISSUE_TITLE="[E2E Test] Add greeting message feature $(date +%Y%m%d_%H%M%S)"
ISSUE_BODY="## Feature Request

Add a simple greeting message feature to AILANG.

### Requirements
- Create a new builtin function \`greet\` that returns a greeting
- Add tests for the new builtin
- Update documentation

### Acceptance Criteria
- [ ] Builtin function works: \`greet(\"World\")\" returns \"Hello, World!\"
- [ ] Unit tests pass
- [ ] Documentation updated

---
*Auto-generated E2E test issue*"

log "Creating test issue..."
ISSUE_URL=$(gh issue create \
    --repo "$REPO" \
    --title "$ISSUE_TITLE" \
    --body "$ISSUE_BODY" \
    --label "coordinator:feature" \
    2>&1)
ISSUE_NUM=$(echo "$ISSUE_URL" | grep -oE '[0-9]+$')
success "Created issue #$ISSUE_NUM: $ISSUE_URL"
echo ""

# Function to wait for a label
wait_for_label() {
    local label=$1
    local timeout=$2
    local elapsed=0

    log "Waiting for '$label' label on issue #$ISSUE_NUM..."
    while [ $elapsed -lt $timeout ]; do
        if gh issue view "$ISSUE_NUM" --repo "$REPO" --json labels -q '.labels[].name' | grep -q "$label"; then
            return 0
        fi
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
        echo -n "."
    done
    echo ""
    return 1
}

# Function to wait for a comment containing text
wait_for_comment() {
    local pattern=$1
    local timeout=$2
    local elapsed=0

    log "Waiting for comment containing '$pattern'..."
    while [ $elapsed -lt $timeout ]; do
        if gh issue view "$ISSUE_NUM" --repo "$REPO" --json comments -q '.comments[].body' | grep -q "$pattern"; then
            return 0
        fi
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
        echo -n "."
    done
    echo ""
    return 1
}

# Function to add approval label
add_approval() {
    local label=$1
    if [ "$AUTO_APPROVE" = true ]; then
        log "Auto-approving: Adding '$label' label..."
        gh issue edit "$ISSUE_NUM" --repo "$REPO" --add-label "$label"
        success "Added '$label' label"
    else
        echo ""
        echo "═══════════════════════════════════════════════════════════════════════"
        echo "  MANUAL APPROVAL REQUIRED"
        echo "═══════════════════════════════════════════════════════════════════════"
        echo ""
        echo "  Review the work on issue #$ISSUE_NUM and add the '$label' label"
        echo "  URL: $ISSUE_URL"
        echo ""
        read -p "  Press Enter when you've added the label, or 'q' to quit: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Qq]$ ]]; then
            warn "Test aborted by user"
            exit 0
        fi
    fi
}

# Import the issue to coordinator
log "Importing issue to coordinator inbox..."
./bin/ailang messages import-github --labels "coordinator:feature" 2>&1 | head -5
success "Issue imported"
echo ""

# ═══════════════════════════════════════════════════════════════════════
# STAGE 1: Design Doc Creation
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  STAGE 1: Design Document Creation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log "Waiting for coordinator to pick up task and start working..."
if wait_for_comment "Working" $MAX_WAIT; then
    success "Coordinator started working on task"
else
    warn "No 'Working' comment found yet - coordinator may still be processing"
fi

log "Waiting for design document to be created..."
if wait_for_label "needs-design-approval" $MAX_WAIT; then
    success "Design document ready for review!"
else
    warn "Design stage timed out. Check coordinator logs."
    log "Tail of coordinator log:"
    tail -20 ~/.ailang/logs/coordinator.log
fi

# Show the design doc comment
log "Latest comments on issue:"
gh issue view "$ISSUE_NUM" --repo "$REPO" --json comments -q '.comments[-1].body' | head -30
echo "..."
echo ""

add_approval "design-approved"

# ═══════════════════════════════════════════════════════════════════════
# STAGE 2: Sprint Planning
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  STAGE 2: Sprint Planning"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log "Waiting for sprint plan to be created..."
if wait_for_label "needs-sprint-approval" $MAX_WAIT; then
    success "Sprint plan ready for review!"
else
    warn "Sprint stage timed out. Check coordinator logs."
fi

log "Latest comments on issue:"
gh issue view "$ISSUE_NUM" --repo "$REPO" --json comments -q '.comments[-1].body' | head -30
echo "..."
echo ""

add_approval "sprint-approved"

# ═══════════════════════════════════════════════════════════════════════
# STAGE 3: Implementation
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  STAGE 3: Implementation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log "Waiting for implementation to complete..."
if wait_for_label "needs-merge-approval" $MAX_WAIT; then
    success "Implementation complete, ready for merge review!"
else
    warn "Implementation stage timed out. Check coordinator logs."
fi

log "Latest comments on issue:"
gh issue view "$ISSUE_NUM" --repo "$REPO" --json comments -q '.comments[-1].body' | head -30
echo "..."
echo ""

add_approval "merge-approved"

# ═══════════════════════════════════════════════════════════════════════
# STAGE 4: Merge & PR
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  STAGE 4: Merge & Pull Request"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

log "Waiting for merge and PR creation..."
sleep 10  # Give coordinator time to process

# Check if issue was closed
ISSUE_STATE=$(gh issue view "$ISSUE_NUM" --repo "$REPO" --json state -q '.state')
if [ "$ISSUE_STATE" = "CLOSED" ]; then
    success "Issue #$ISSUE_NUM closed successfully!"
else
    warn "Issue not yet closed (state: $ISSUE_STATE)"
fi

# Check for PR
log "Checking for pull request..."
PR_URL=$(gh pr list --repo "$REPO" --state open --json url,title -q ".[] | select(.title | contains(\"$ISSUE_NUM\") or contains(\"E2E Test\")) | .url" | head -1)
if [ -n "$PR_URL" ]; then
    success "Pull request created: $PR_URL"
else
    warn "No PR found yet. Check worktrees for pending changes."
    log "Active worktrees:"
    ./bin/ailang coordinator worktrees 2>&1 | head -10
fi

# ═══════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════════════"
echo "  E2E Workflow Test Complete!"
echo "═══════════════════════════════════════════════════════════════════════"
echo ""
echo "  Issue: $ISSUE_URL"
echo "  State: $ISSUE_STATE"
if [ -n "$PR_URL" ]; then
    echo "  PR: $PR_URL"
fi
echo ""
echo "  Check coordinator logs for details:"
echo "    tail -100 ~/.ailang/logs/coordinator.log"
echo ""
echo "  Check task status:"
echo "    ./bin/ailang coordinator tasks"
echo ""

# Cleanup option
if [ "$ISSUE_STATE" != "CLOSED" ]; then
    read -p "Close test issue now? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        gh issue close "$ISSUE_NUM" --repo "$REPO" --comment "E2E test completed"
        success "Issue closed"
    fi
fi
