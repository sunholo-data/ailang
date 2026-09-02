#!/bin/bash
# session_start.sh - Claude Code SessionStart hook → Check User Inbox
#
# This script is called by Claude Code when a session starts.
# It checks the user inbox for new messages from autonomous agents
# using the ailang messages CLI, pinned to the CANONICAL store (prod Firestore).
# Hooks do not inherit a login shell, so the store is exported explicitly below.
#
# Hook configuration (.claude/hooks.json):
# {
#   "hooks": {
#     "SessionStart": {
#       "command": "scripts/hooks/session_start.sh",
#       "timeout": 10
#     }
#   }
# }
#
# Input:
#   stdin - JSON payload from Claude Code (hook event data)
#
# Environment variables:
#   CLAUDE_ENV_FILE - File to write environment variables to (provided by Claude Code)

set -euo pipefail

# Read hook JSON from stdin (Claude Code sends hook data via stdin, not env var)
HOOK_JSON=$(cat || echo "{}")

# Configuration
DEFAULT_STATE_DIR="${HOME}/.ailang/state"
STATE_DIR="${STATE_DIR:-$DEFAULT_STATE_DIR}"
LOG_FILE="${STATE_DIR}/hooks.log"
LOCK_FILE="${STATE_DIR}/session_start.lock"

# Detect project root (where .claude/ directory is)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"
}

log "=== Session Start Hook Started ==="

# Switch GitHub auth to AI agent account for commits/pushes
# This ensures all AI-generated commits are attributed to sunholo-voight-kampff
if command -v gh &> /dev/null; then
    CURRENT_GH_USER=$(gh api user --jq '.login' 2>/dev/null || echo "")
    if [ "$CURRENT_GH_USER" != "sunholo-voight-kampff" ]; then
        if gh auth switch --user sunholo-voight-kampff 2>/dev/null; then
            log "Switched GitHub auth to sunholo-voight-kampff"
        else
            log "Could not switch to sunholo-voight-kampff (account may not be configured)"
        fi
    else
        log "GitHub auth already set to sunholo-voight-kampff"
    fi
fi

# Get current version. Primary source: newest git tag (authoritative — set by
# the release workflow). Fallback: newest version string in CHANGELOG.md.
# The root CHANGELOG.md is now an index referencing per-series files
# (e.g. "v0.18-current.md"), so a naive grep/head would pin to an old series.
get_current_version() {
    local tag
    tag=$(git -C "$PROJECT_ROOT" tag --sort=-version:refname 2>/dev/null \
          | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1)
    if [ -n "$tag" ]; then
        echo "$tag"
        return
    fi
    local CHANGELOG="$PROJECT_ROOT/CHANGELOG.md"
    if [ -f "$CHANGELOG" ]; then
        grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' "$CHANGELOG" \
          | sort -t. -k1.2,1n -k2,2n -k3,3n \
          | tail -1
    else
        echo "unknown"
    fi
}

CURRENT_VERSION=$(get_current_version)
log "Current AILANG version: $CURRENT_VERSION"

# Prevent duplicate execution within 3 seconds
# Claude Code seems to run SessionStart hook twice on session start
if [ -f "$LOCK_FILE" ]; then
    LOCK_AGE=$(($(date +%s) - $(stat -f %m "$LOCK_FILE" 2>/dev/null || stat -c %Y "$LOCK_FILE" 2>/dev/null || echo 0)))
    if [ "$LOCK_AGE" -lt 3 ]; then
        log "Skipping duplicate execution (lock file age: ${LOCK_AGE}s)"
        # Output empty context to avoid breaking Claude Code
        jq -n \
          --arg context "📭 Agent inbox: Already checked in this session." \
          '{
            "hookSpecificOutput": {
              "hookEventName": "SessionStart",
              "additionalContext": $context
            }
          }'
        exit 0
    fi
fi

# Create lock file
touch "$LOCK_FILE"
log "Created lock file to prevent duplicate execution"

# Parse Claude hook JSON from stdin
SESSION_ID=$(echo "$HOOK_JSON" | jq -r '.sessionId // "unknown"')
USER_ID=$(echo "$HOOK_JSON" | jq -r '.userId // "unknown"')
log "Session ID: $SESSION_ID"
log "User ID: $USER_ID"

# GitHub issue import is now handled by the coordinator daemon's github_sync
# (see coordinator.github_sync.enabled in ~/.ailang/config.yaml)
# The hook no longer imports directly to avoid routing to wrong inbox.
# If coordinator is not running, issues can still be imported manually:
#   ailang messages import-github --inbox design-doc-creator
log "GitHub import handled by coordinator (github_sync enabled)"

# Read the CANONICAL message store, not this machine's local SQLite.
#
# Hooks do not inherit a login shell, so exports in ~/.zshenv are NOT in scope here.
# Without pinning the store the hook queries local SQLite and reports a number that
# has nothing to do with the shared inbox: measured 2026-08-26, the banner said
# "16 unread" while the canonical store held 74, including four months of public
# feedback nobody had seen. A quiet banner is the single most effective way to make
# a full inbox look empty, so pin it explicitly rather than inheriting.
#
# Overridable: set AILANG_MESSAGES_STORE/_PROJECT before the hook to point elsewhere
# (e.g. =local to inspect this machine's private inbox).
export AILANG_MESSAGES_STORE="${AILANG_MESSAGES_STORE:-gcp}"
export AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}"

# Cap the startup injection at 5 most-recent unread messages; full backlog
# is still visible via `ailang messages list --unread`.
INBOX_DUMP_LIMIT="${AILANG_INBOX_DUMP_LIMIT:-5}"
ALL_MESSAGES_JSON=$(ailang messages list --unread --json 2>/dev/null || echo "[]")
UNREAD_COUNT=$(echo "$ALL_MESSAGES_JSON" | jq 'length' 2>/dev/null || echo "0")

# A binary older than v0.34.0 ignores AILANG_MESSAGES_STORE silently — it reads local
# SQLite and exits 0, so the count above would be wrong with no error anywhere. Probe
# with an INVALID value: a current binary refuses it, an old one lists normally.
STORE_LABEL="canonical (prod)"
if AILANG_MESSAGES_STORE=__invalid__ ailang messages list --unread --json >/dev/null 2>&1; then
    # An invalid store value SUCCEEDED, so this binary is not reading the variable
    # at all — the count above came from local SQLite regardless of what we exported.
    STORE_LABEL="LOCAL ONLY — binary predates v0.34.0, run 'make quick-install'"
    log "WARNING: ailang binary ignores AILANG_MESSAGES_STORE; inbox count is local-only"
fi
MESSAGES_JSON=$(echo "$ALL_MESSAGES_JSON" | jq ".[:${INBOX_DUMP_LIMIT}]" 2>/dev/null || echo "[]")

# Function to query brain for relevant context based on recent files
# Bound a command's runtime. macOS ships no coreutils `timeout`, so use perl's alarm.
# Guards against `ailang cache search` blocking ~30s on its Ollama embedding request
# whenever an eval job holds the GPU with a large model (see AILANG_BRAIN_TIMEOUT).
run_bounded() {
    local secs="$1"; shift
    perl -e 'alarm(shift @ARGV); exec @ARGV' "$secs" "$@" 2>/dev/null
}

get_brain_context() {
    local BRAIN_CONTEXT=""
    local BRAIN_TIMEOUT="${AILANG_BRAIN_TIMEOUT:-3}"

    # Check if brain hooks are disabled
    if [ "${AILANG_BRAIN_HOOKS:-1}" = "0" ]; then
        echo ""
        return
    fi

    # Check if ailang supports cache command
    if ! command -v ailang &> /dev/null; then
        echo ""
        return
    fi

    # Check if any brain DB exists (graceful bootstrap)
    if [ ! -f "$PROJECT_ROOT/.ailang/state/brain.db" ] && [ ! -f "$HOME/.ailang/state/brain.db" ]; then
        echo ""
        return
    fi

    # Get recently modified files for context query
    local RECENT_FILES
    RECENT_FILES=$(git diff --name-only HEAD~3 HEAD 2>/dev/null | head -5 | tr '\n' ',' | sed 's/,$//')

    local BRAIN_RESULTS=""
    if [ -n "$RECENT_FILES" ]; then
        # Search brain based on recently touched files
        BRAIN_RESULTS=$(run_bounded "$BRAIN_TIMEOUT" ailang cache search --context "$RECENT_FILES" --limit 2 2>/dev/null || echo "")
    fi

    # Also do a general search based on current branch/task
    local BRANCH_NAME
    BRANCH_NAME=$(git branch --show-current 2>/dev/null || echo "")
    if [ -n "$BRANCH_NAME" ] && [ "$BRANCH_NAME" != "dev" ] && [ "$BRANCH_NAME" != "main" ]; then
        local BRANCH_RESULTS
        BRANCH_RESULTS=$(run_bounded "$BRAIN_TIMEOUT" ailang cache search "${BRANCH_NAME//-/ }" --limit 1 2>/dev/null || echo "")
        if [ -n "$BRANCH_RESULTS" ] && ! echo "$BRANCH_RESULTS" | grep -q "No results"; then
            BRAIN_RESULTS="${BRAIN_RESULTS}
${BRANCH_RESULTS}"
        fi
    fi

    # Only output if we got results
    if [ -n "$BRAIN_RESULTS" ] && ! echo "$BRAIN_RESULTS" | grep -q "No results"; then
        # Filter to just the result lines (skip metadata)
        local FILTERED
        FILTERED=$(echo "$BRAIN_RESULTS" | grep -E '^\s+[0-9]+\.|^\s+[0-9]+[dhm] ago' | head -6)
        if [ -n "$FILTERED" ]; then
            BRAIN_CONTEXT="
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🧠 BRAIN: Relevant knowledge for this session
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$FILTERED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            log "Brain context injected ($(echo "$FILTERED" | wc -l | tr -d ' ') entries)"
        fi
    fi

    echo "$BRAIN_CONTEXT"
}

# Function to check for active sprint and return context
get_sprint_context() {
    local SPRINT_DIR="$PROJECT_ROOT/.ailang/state/sprints"
    local SPRINT_CONTEXT=""

    if [ -d "$SPRINT_DIR" ]; then
        for SPRINT_FILE in "$SPRINT_DIR"/sprint_*.json; do
            if [ -f "$SPRINT_FILE" ] && grep -q '"status": "in_progress"' "$SPRINT_FILE" 2>/dev/null; then
                local SPRINT_ID=$(jq -r '.sprint_id // "unknown"' "$SPRINT_FILE" 2>/dev/null)
                local DESIGN_DOC=$(jq -r '.design_doc // ""' "$SPRINT_FILE" 2>/dev/null)
                local TOTAL_MILESTONES=$(jq '.features | length' "$SPRINT_FILE" 2>/dev/null || echo 0)
                local COMPLETED=$(jq '[.features[] | select(.passes == true)] | length' "$SPRINT_FILE" 2>/dev/null || echo 0)
                local NEXT_MILESTONE=$(jq -r '.features[] | select(.passes == null) | .id' "$SPRINT_FILE" 2>/dev/null | head -1)
                local NEXT_DESCRIPTION=$(jq -r --arg id "$NEXT_MILESTONE" '.features[] | select(.id == $id) | .description' "$SPRINT_FILE" 2>/dev/null)
                local NEXT_CRITERIA=$(jq -r --arg id "$NEXT_MILESTONE" '.features[] | select(.id == $id) | .acceptance_criteria // [] | .[]' "$SPRINT_FILE" 2>/dev/null | head -5)

                if [ -n "$NEXT_MILESTONE" ]; then
                    SPRINT_CONTEXT="
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🏃 ACTIVE SPRINT: $SPRINT_ID
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Progress: $COMPLETED/$TOTAL_MILESTONES milestones complete
Design doc: $DESIGN_DOC
JSON file: $SPRINT_FILE

▶ NEXT MILESTONE: $NEXT_MILESTONE
  $NEXT_DESCRIPTION

Acceptance criteria:
$(echo "$NEXT_CRITERIA" | sed 's/^/  • /')

💡 Use this milestone's criteria to guide your work.
   Run checkpoint when done: .claude/skills/sprint-executor/scripts/milestone_checkpoint.sh $NEXT_MILESTONE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                else
                    SPRINT_CONTEXT="
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🏃 ACTIVE SPRINT: $SPRINT_ID (ALL MILESTONES COMPLETE!)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Progress: $COMPLETED/$TOTAL_MILESTONES milestones complete ✅

💡 Sprint appears complete. Consider:
   1. Update JSON status to \"completed\"
   2. Final commit with sprint summary
   3. Move design doc to implemented/
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                fi
                log "Found active sprint: $SPRINT_ID ($COMPLETED/$TOTAL_MILESTONES complete)"
                break
            fi
        done
    fi
    echo "$SPRINT_CONTEXT"
}

if [ "$UNREAD_COUNT" -eq 0 ]; then
    log "No unread messages"

    # Export empty message indicator to Claude Code environment
    if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
        echo "export AGENT_INBOX_COUNT=0" >> "$CLAUDE_ENV_FILE"
        log "Exported AGENT_INBOX_COUNT=0 to CLAUDE_ENV_FILE"
    fi

    # Check for active sprint and brain context even when no inbox messages
    SPRINT_CONTEXT=$(get_sprint_context)
    BRAIN_CONTEXT=$(get_brain_context)

    # Output context (will appear in system reminders)
    echo "📦 AILANG $CURRENT_VERSION"
    echo "📭 Agent inbox: No unread messages — store: $STORE_LABEL"
    if [ -n "$SPRINT_CONTEXT" ]; then
        echo "$SPRINT_CONTEXT"
    fi
    if [ -n "$BRAIN_CONTEXT" ]; then
        echo "$BRAIN_CONTEXT"
    fi

    log "Outputted context to Claude"
    exit 0
fi

log "Found $UNREAD_COUNT unread message(s)"

# Export message count to Claude Code environment
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    echo "export AGENT_INBOX_COUNT=$UNREAD_COUNT" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_COUNT=$UNREAD_COUNT to CLAUDE_ENV_FILE"
fi

# Export messages as environment variable (base64 encoded to handle special characters)
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    MESSAGES_B64=$(echo "$MESSAGES_JSON" | base64)
    echo "export AGENT_INBOX_MESSAGES='$MESSAGES_B64'" >> "$CLAUDE_ENV_FILE"
    log "Exported AGENT_INBOX_MESSAGES (base64 encoded JSON) to CLAUDE_ENV_FILE"
fi

# Get sprint context (brain context is now handled by user-level hook ~/.ailang/hooks/brain_session.sh)
SPRINT_CONTEXT=$(get_sprint_context)

# If many messages (5+), try to generate triage summary
TRIAGE_SUMMARY=""
if [ "$UNREAD_COUNT" -ge 5 ]; then
    TRIAGE_OUTPUT=$(ailang messages triage --threshold 0.50 2>/dev/null || echo "")
    if [ -n "$TRIAGE_OUTPUT" ] && echo "$TRIAGE_OUTPUT" | grep -q "Cluster"; then
        TRIAGE_SUMMARY="
📊 TRIAGE SUMMARY (clustered by intent):
$TRIAGE_OUTPUT"
        log "Triage summary generated for $UNREAD_COUNT messages"
    fi
fi

# Build formatted context string for Claude
# The MESSAGES_JSON comes from the CLI and has format:
# [{"id":"msg_xxx","from_agent":"test","to_inbox":"user","title":"Title","payload":"...","status":"unread","created_at":"..."}]
INBOX_HEADER="📬 AGENT INBOX: $UNREAD_COUNT unread — store: $STORE_LABEL"
if [ "$UNREAD_COUNT" -gt "$INBOX_DUMP_LIMIT" ] 2>/dev/null; then
    INBOX_HEADER="$INBOX_HEADER (showing $INBOX_DUMP_LIMIT most recent)"
fi

CONTEXT_MESSAGE=$(cat <<EOF
📦 AILANG $CURRENT_VERSION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$INBOX_HEADER
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

$(echo "$MESSAGES_JSON" | jq -r '.[] | "ID: \(.id)\nFrom: \(.from_agent)\nTitle: \(.title)\nTime: \(.created_at)\n"')
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 Triage: 'ailang messages list --unread --json' (full IDs + bodies;
   'messages read' marks read as a side effect — triage via --json instead)
   After handling: 'ailang messages ack <id>' per message
   ('ack --all' also sweeps outbound cross-mission inboxes — avoid)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$TRIAGE_SUMMARY
$SPRINT_CONTEXT
EOF
)

log "Context message prepared (length: ${#CONTEXT_MESSAGE} chars)"

# Output plain text to stdout
# Note: This appears in system reminders but may have length limits
# Messages are NOT marked as read - Claude must acknowledge them with: ailang messages ack <id>
echo "$CONTEXT_MESSAGE"

# Background: warm embedding cache for neural search (non-blocking)
# This prevents cold-cache hangs when design-doc-creator uses --neural
#
# `>/dev/null 2>&1` IS LOAD-BEARING — without it this line is non-blocking for the SCRIPT and
# blocking for every CONSUMER that captures our stdout (2026-08-14, motoko mission iteration 5).
# A backgrounded child inherits stdout, so a `$(...)`-style capture cannot see EOF until the CHILD
# exits, however promptly the script itself `exit 0`s. Claude Code captures hook stdout (that is
# how this banner reaches the session), so the SessionStart hook was effectively held open for as
# long as the warmup ran — bounded only by its own `--timeout 3m`, i.e. 180s against the mission
# driver's 120s model-probe cap. Measured: capture arm 8433ms vs redirected arm 8ms vs 237ms with
# the background child removed; one real hook capture took 96,377ms. That is what produced the
# fleet's "probe timed out — captured output: ''" refusals (47/186 fires on v1, 6/11 on motoko,
# 0/89 on world, the one mission with no hooks at all). See queue item 5a.
if command -v ailang &> /dev/null; then
    ailang docs embed-warmup --quiet --timeout 3m >/dev/null 2>&1 &
    log "Started background embedding cache warmup (PID: $!)"
fi

log "=== Session Start Hook Completed ==="
exit 0
