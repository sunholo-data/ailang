# M-MSG-GITHUB: GitHub Issues Backend for Messaging System

**Status**: ✅ Implemented
**Target**: v0.5.9
**Completed**: December 10, 2025
**Priority**: P1 (Medium)
**Actual Duration**: 1 day
**Dependencies**: Existing messaging system (v0.5.0+)

## Implementation Summary

All goals achieved:
- ✅ `ailang messages send --github` creates both local message AND GitHub Issue
- ✅ `ailang messages list` shows GitHub Issue numbers when available
- ✅ Session startup hook imports new GitHub Issues as messages
- ✅ Commits can reference `#123` and link to actual issues
- ✅ Full audit trail: local DB + GitHub Issues mirror each other
- ✅ **Bonus**: `ailang messages reply` adds comments to existing issue threads
- ✅ **Bonus**: Auto-label creation (bug, feature, from:agent, ailang-message)

**Key Implementation Files:**
- `internal/messaging/github.go` - GitHub client with auto-label creation
- `cmd/ailang/messages.go` - CLI with `--github`, `--type`, `reply` commands
- `scripts/hooks/session_start.sh` - Auto-imports GitHub issues on session start
- `~/.ailang/config.yaml` - GitHub configuration

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Single command creates issue + message; no manual GitHub workflow |
| Preserve Semantic Clarity | 0 | 0 | N/A - tooling feature, not language feature |
| Increase Determinism | + | +1 | Explicit `--github` flag; no automatic/hidden behavior |
| Lower Token Cost | + | +1 | Issue numbers in commits provide direct context links |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

The AILANG messaging system (`ailang messages`) provides excellent agent-to-agent and agent-to-user communication via SQLite. However:

**Current State:**
- Bug reports and feature requests from external projects (e.g., stapledons_voyage) only exist in SQLite
- No public record of issues for tracking in commits
- Cannot reference `#123` style issue numbers in commit messages
- GitHub Issues (industry standard for project tracking) unused for agent communications
- No way to send messages TO AILANG via GitHub Issues

**Impact:**
- Developers cannot track message history via standard GitHub workflows
- Commit messages lack linkage to the issues they address
- External contributors cannot participate via GitHub Issues
- Agent messages are siloed in local databases

## Goals

**Primary Goal:** Enable two-way sync between AILANG messages and GitHub Issues for bug/feature tracking.

**Success Metrics:**
- `ailang messages send --github` creates both local message AND GitHub Issue
- `ailang messages list` shows GitHub Issue numbers when available
- Session startup hook imports new GitHub Issues as messages
- Commits can reference `#123` and link to actual issues
- Full audit trail: local DB + GitHub Issues mirror each other

## Solution Design

### Overview

Add GitHub Issues as an optional backend that mirrors specific message types (bugs, features) to a configured GitHub repository. The SQLite database remains the primary store; GitHub provides persistence, public visibility, and commit linkage.

```
┌─────────────────────────────────────────────────────────────────┐
│                      Message Flow                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Agent/User                                                     │
│      │                                                          │
│      ▼                                                          │
│  ailang messages send --github --type bug "Fix parser"          │
│      │                                                          │
│      ├──────────────────┬───────────────────┐                   │
│      ▼                  ▼                   ▼                   │
│  SQLite DB         GitHub Issue         Stdout                  │
│  (primary)         (mirror)             (confirmation)          │
│                         │                                       │
│                         ▼                                       │
│                    Issue #123                                   │
│                         │                                       │
│                         ▼                                       │
│  Session Start Hook ────┴──▶ Create message from new issues     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Architecture

**Components:**

1. **GitHub Backend** (`internal/messages/github.go`):
   - Create issues via `gh` CLI (leverages existing auth)
   - List issues with specific labels
   - Map issues ↔ messages

2. **Schema Extension** (`internal/messages/schema.go`):
   - Add `github_issue_number` column
   - Add `github_repo` column
   - Add `message_type` enum (general, bug, feature)

3. **CLI Extensions** (`cmd/ailang/messages.go`):
   - `--github` flag to sync to GitHub
   - `--type` flag for message categorization
   - `--repo` flag to specify target repository (default from config)

4. **Startup Hook Enhancement** (`scripts/hooks/session_start.sh`):
   - Check for new GitHub issues with `ailang-message` label
   - Import as unread messages

5. **Configuration** (`~/.ailang/config.yaml`):
   - Default repository for GitHub sync
   - Labels to apply to issues
   - Labels to watch for incoming messages

### Authentication & Agent Identity

**Important:** The `gh` CLI authenticates as your GitHub user account. Issues will be created under YOUR username, not a bot account.

**Agent attribution strategy:**

Since we can't change the GitHub author, we use **labels + metadata** to identify the reporting agent:

```
┌─────────────────────────────────────────────────────────────────┐
│  GitHub Issue (created by MarkEdmondson1234)                    │
├─────────────────────────────────────────────────────────────────┤
│  Title: [stapledon] Bug: Parser crashes on nested ADTs          │
│  Labels: ailang-message, bug, from:stapledon                    │
│  Body:                                                          │
│    **Reporter:** stapledon (via ailang agent)                   │
│    **Message ID:** 601ae97d-ce6a-4e28-8b6a-58b253914a0c         │
│    ---                                                          │
│    Parser crashes on nested ADTs...                             │
└─────────────────────────────────────────────────────────────────┘
```

**Pre-flight checks before creating issues:**

1. ✅ Verify `gh` is installed: `gh --version`
2. ✅ Verify authentication: `gh auth status`
3. ✅ Load `github.expected_user` from config
4. ✅ **HARD FAIL** if active user ≠ expected user (no prompt, no bypass)
5. ✅ Provide exact switch command in error message

**Future option - GitHub App (out of scope for v0.5.9):**

A dedicated GitHub App could post as `ailang-bot[bot]`, enabling true bot identity. This would require:
- Creating a GitHub App in sunholo-data org
- Managing app installation tokens
- More complex auth flow

For v0.5.9, we use the simpler `gh` CLI approach with label-based attribution.

### Implementation Plan

**Phase 1: Schema & Core** (~4 hours)
- [ ] Add `github_issue_number`, `github_repo`, `message_type` to messages table
- [ ] Add migration for existing databases
- [ ] Create `MessageType` enum (general, bug, feature)
- [ ] Update `Message` struct with new fields
- [ ] Unit tests for schema changes

**Phase 2: GitHub Backend** (~6 hours)
- [ ] Create `internal/messages/github.go`
- [ ] Implement pre-flight checks:
  - [ ] `CheckGHInstalled() error` - verify `gh --version`
  - [ ] `CheckGHAuth() (username string, error)` - verify `gh auth status`, return active user
  - [ ] `LoadGitHubConfig() (*GitHubConfig, error)` - load from `~/.ailang/config.yaml`
  - [ ] `ValidateUser(config *GitHubConfig, activeUser string) error` - **HARD FAIL** if mismatch
  - [ ] `CheckRepoAccess(repo string) error` - verify push access
- [ ] Implement `CreateIssue(msg Message) (issueNumber int, error)`
  - [ ] Format title with `[from]` prefix
  - [ ] Add `from:agent-name` label dynamically
  - [ ] Include reporter metadata in issue body
- [ ] Implement `ListIssuesByLabel(repo, label string) ([]Issue, error)`
- [ ] Implement `GetIssue(repo string, number int) (Issue, error)`
- [ ] Map GitHub Issue ↔ Message conversion
- [ ] Unit tests with mocked `gh` responses

**Phase 3: CLI Integration** (~4 hours)
- [ ] Add `--github` flag to `messages send`
- [ ] Add `--type bug|feature|general` flag
- [ ] Add `--repo owner/repo` flag (optional, uses default)
- [ ] Show issue number in `messages list` output
- [ ] Add `messages sync` command for manual sync
- [ ] Update help text and examples

**Phase 4: Startup Hook** (~3 hours)
- [ ] Add `ailang messages import-github` command
- [ ] Query issues with `ailang-message` label not yet imported
- [ ] Create messages from new issues
- [ ] Mark issues as imported (add `imported` label or track in DB)
- [ ] Integrate into `session_start.sh`

**Phase 5: Documentation & Polish** (~3 hours)
- [ ] Update CLAUDE.md with GitHub sync workflow
- [ ] Add configuration guide
- [ ] Add examples to CLI help
- [ ] Update session_start hook documentation

### Files to Modify/Create

**New files:**
- `internal/messages/github.go` - GitHub backend (~250 LOC)
- `internal/messages/github_test.go` - Tests (~200 LOC)
- `internal/messages/types.go` - MessageType enum (~50 LOC)

**Modified files:**
- `internal/messages/schema.go` - Add columns (~30 LOC)
- `internal/messages/db.go` - Update queries (~50 LOC)
- `cmd/ailang/messages.go` - Add flags and commands (~100 LOC)
- `scripts/hooks/session_start.sh` - Add GitHub import (~20 LOC)
- `CLAUDE.md` - Documentation (~50 LOC)

## Examples

### Example 1: Send Bug Report with GitHub Issue

**Before (current):**
```bash
# Send message (local only)
ailang messages send ailang "Parser crashes on nested ADTs" \
  --title "Bug: Nested ADT parsing" --from "stapledons_voyage"

# Manually create GitHub issue separately
gh issue create --title "Bug: Nested ADT parsing" --body "..."

# No linkage between the two
```

**After:**
```bash
# Single command creates both
ailang messages send ailang "Parser crashes on nested ADTs" \
  --title "Bug: Nested ADT parsing" \
  --from "stapledons_voyage" \
  --type bug \
  --github

# Output:
# ✓ Verified: gh authenticated as MarkEdmondson1234
# ✓ Message sent to ailang inbox
# ✓ GitHub Issue created: sunholo-data/ailang#142
#   Title: [stapledons_voyage] Bug: Nested ADT parsing
#   Labels: ailang-message, bug, from:stapledons_voyage
#
# 💡 Use "Fixes #142" in commits to auto-close

# Later, list messages shows issue number
ailang messages list
# ID                                    TITLE                    GITHUB
# 601ae97d-ce6a-4e28-8b6a-58b253914a0c  Bug: Nested ADT parsing  #142
```

### Example 2: Import GitHub Issues as Messages

**Scenario:** External user creates issue on GitHub:
```
Title: Feature request: string interpolation
Labels: ailang-message, feature
Body: Please add f-string style interpolation...
```

**On session start:**
```bash
# session_start.sh automatically runs:
ailang messages import-github

# Output in system reminders:
# 📬 AGENT INBOX: 1 new message(s) imported from GitHub
# • #156 "Feature request: string interpolation" (sunholo-data/ailang)
```

### Example 3: Commit with Issue Reference

**After addressing a bug:**
```bash
git commit -m "$(cat <<'EOF'
Fix nested ADT parsing panic

Fixes #142

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"

# GitHub automatically:
# - Links commit to issue #142
# - Closes issue when merged to main
```

### Example 4: Error Handling (gh not configured)

```bash
# If gh CLI not installed
ailang messages send ailang "Bug report" --github
# ✗ Error: gh CLI not found
# Install from: https://cli.github.com/
# Message saved locally but NOT synced to GitHub

# If wrong account active (config.expected_user = "MarkEdmondson1234")
ailang messages send ailang "Bug report" --github
# ✗ Error: GitHub account mismatch
#   Config expected: MarkEdmondson1234
#   gh active user:  rw-markedmondson
#
# Fix with: gh auth switch --user MarkEdmondson1234
# Then retry: ailang messages send ailang "Bug report" --github
#
# ✓ Message saved locally (GitHub sync skipped)

# If not authenticated at all
ailang messages send ailang "Bug report" --github
# ✗ Error: gh not authenticated
# Run: gh auth login
# Message saved locally but NOT synced to GitHub
```

**Key principle:** Messages are ALWAYS saved to SQLite. GitHub sync failures don't lose data.

### Example 5: Configuration

**~/.ailang/config.yaml:**
```yaml
github:
  # Default repository for --github flag
  default_repo: sunholo-data/ailang

  # REQUIRED: Expected GitHub username (must match `gh auth status`)
  # This prevents accidentally creating issues under wrong account
  expected_user: MarkEdmondson1234

  # Labels to add when creating issues
  create_labels:
    - ailang-message
    - from-agent

  # Labels to watch for incoming messages
  watch_labels:
    - ailang-message
    - needs-agent-response

  # Auto-import on session start (default: true)
  auto_import: true
```

**Account verification flow:**
```
┌─────────────────────────────────────────────────────────────────┐
│  ailang messages send --github                                  │
│      │                                                          │
│      ▼                                                          │
│  Load config.yaml                                               │
│      │                                                          │
│      ▼                                                          │
│  github.expected_user = "MarkEdmondson1234"                     │
│      │                                                          │
│      ▼                                                          │
│  gh auth status → active_user = "rw-markedmondson"              │
│      │                                                          │
│      ▼                                                          │
│  expected_user != active_user?                                  │
│      │                                                          │
│      ├─── YES ──▶ ✗ Error: Account mismatch                     │
│      │            Run: gh auth switch --user MarkEdmondson1234  │
│      │            (message saved locally, GitHub sync aborted)  │
│      │                                                          │
│      └─── NO ───▶ ✓ Proceed with issue creation                 │
└─────────────────────────────────────────────────────────────────┘
```

## Success Criteria

- [ ] `ailang messages send --github --type bug` creates issue and message
- [ ] Issue number stored in SQLite and displayed in `messages list`
- [ ] `ailang messages import-github` imports labeled issues as messages
- [ ] Session start hook imports new GitHub issues automatically
- [ ] Duplicate issues are not re-imported (tracked by issue number)
- [ ] **Account validation HARD FAILS** if `gh` user ≠ `config.expected_user`
- [ ] Clear error message with exact `gh auth switch` command
- [ ] Messages always saved locally even when GitHub sync fails
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added to CLI help

## Testing Strategy

**Unit tests:**
- Schema migration (new columns, defaults)
- Message type enum validation
- GitHub Issue ↔ Message conversion
- Import deduplication logic
- Account validation: `expected_user` vs `gh auth status` mismatch → error
- Account validation: matching users → proceed

**Integration tests:**
- Mock `gh` CLI responses
- End-to-end send with `--github`
- Import from mock issue list
- Database round-trip with new fields

**Manual testing:**
- Create issue via CLI, verify on GitHub
- Create issue on GitHub, verify import
- Test with multiple GitHub accounts
- Test with missing `gh` CLI (graceful error)

## Non-Goals

**Not in this feature:**
- GitHub Discussions support - different API, defer to future
- Two-way comment sync - complexity; issues are one-shot for now
- Issue status sync (open/closed) - just track issue number
- PR integration - separate feature
- GitLab/Bitbucket support - GitHub-first, extensible later

## Timeline

**Day 1** (~8 hours):
- Phase 1: Schema & Core
- Phase 2: GitHub Backend (start)

**Day 2** (~8 hours):
- Phase 2: GitHub Backend (complete)
- Phase 3: CLI Integration

**Day 3** (~4 hours):
- Phase 4: Startup Hook
- Phase 5: Documentation & Polish

**Total: ~20 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `gh` CLI not installed | Medium | Graceful error with install instructions |
| Wrong GitHub account active | Medium | Check account before create, warn if mismatch |
| Rate limiting on GitHub API | Low | Use `gh` CLI (handles auth/limits); add retry |
| Schema migration breaks existing DB | High | Backup before migrate; columns have defaults |
| Duplicate issue creation | Medium | Check existing issue number before create |

## References

- [AILANG Messaging System](../../../internal/messages/) - Current implementation
- [Session Start Hook](../../../scripts/hooks/session_start.sh) - Startup workflow
- [GitHub CLI Documentation](https://cli.github.com/manual/) - `gh` command reference
- [Collaboration Hub](../../../internal/server/) - Related message infrastructure

## Future Work

- **GitHub Discussions support**: For longer-form conversations
- **Issue status sync**: Update message status when issue closed
- **Comment sync**: Two-way comment threads
- **PR linkage**: Auto-link PRs that fix issues
- **Multi-repo support**: Sync to multiple repos based on message metadata
- **GitLab/Bitbucket**: Extensible backend for other platforms

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10
