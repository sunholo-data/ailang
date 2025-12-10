# Sprint Plan: M-MSG-GITHUB - GitHub Issues Backend

## Summary
Enable two-way sync between AILANG messages and GitHub Issues, allowing bug reports and feature requests to be tracked in both SQLite and GitHub with issue number references in commits.

**Duration:** 3 days (~20 hours)
**Dependencies:** Existing messaging system (internal/messaging/), gh CLI installed
**Risk Level:** Medium (external API dependency)

## Current Status Analysis

### Completed Recently
- v0.5.9: Compilation timeout detection with stack dumps
- v0.5.8: Go Codegen Type Safety (~400 LOC)
- Recent fixes: If-else codegen, array runtime functions

### Velocity
- Recent average: ~100-150 LOC/day from CHANGELOG
- Estimated capacity: ~500-700 LOC for this sprint

### Existing Implementation
- `internal/messaging/messages.go` - Core store (346 LOC)
- `cmd/ailang/messages.go` - CLI (570 LOC)
- Database: `~/.ailang/state/collaboration.db`
- Current message fields: id, from_agent, to_inbox, title, payload, status, etc.

## Proposed Milestones

### Milestone 1: Schema & Config (M1-SCHEMA)
**Goal:** Add GitHub-related fields to messages and create config loading
**Estimated:** ~120 LOC implementation + ~80 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- Add `github_issue_number`, `github_repo`, `message_type` columns to inbox_messages
- Create migration for existing databases
- Add `MessageType` enum: general, bug, feature
- Create `GitHubConfig` struct in `internal/messaging/config.go`
- Load config from `~/.ailang/config.yaml`

**Files:**
- `internal/messaging/schema.go` - Add migration (~30 LOC)
- `internal/messaging/config.go` - NEW: Config loading (~90 LOC)
- `internal/messaging/inbox.go` - Update InboxMessage struct (~20 LOC)
- `internal/messaging/config_test.go` - NEW: Tests (~80 LOC)

**Acceptance Criteria:**
- [ ] New columns exist in database
- [ ] Existing messages unaffected (NULL defaults)
- [ ] Config loads from ~/.ailang/config.yaml
- [ ] Config validation: expected_user required
- [ ] All tests passing

### Milestone 2: GitHub Backend (M2-GITHUB)
**Goal:** Implement gh CLI integration with pre-flight checks
**Estimated:** ~200 LOC implementation + ~150 LOC tests = 350 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/messaging/github.go`
- Implement pre-flight checks: gh installed, authenticated, user validation
- Implement `CreateIssue()` with labels and metadata in body
- Implement `ListIssuesByLabel()` for import
- Parse `gh` CLI JSON output

**Files:**
- `internal/messaging/github.go` - NEW: GitHub backend (~200 LOC)
- `internal/messaging/github_test.go` - NEW: Tests with mocked gh (~150 LOC)

**Acceptance Criteria:**
- [ ] `CheckGHInstalled()` detects missing gh CLI
- [ ] `CheckGHAuth()` returns active username
- [ ] `ValidateUser()` HARD FAILS on mismatch with config.expected_user
- [ ] `CreateIssue()` formats title with [from] prefix
- [ ] `CreateIssue()` adds from:agent-name label
- [ ] `ListIssuesByLabel()` returns issues as slice
- [ ] All tests passing with mocked responses

**Risks:**
- gh CLI output format changes - Mitigation: Pin expected JSON fields, add version check

### Milestone 3: CLI Integration (M3-CLI)
**Goal:** Add --github, --type, --repo flags to messages send
**Estimated:** ~100 LOC implementation + ~50 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- Add flags to `runMessagesSend()`: --github, --type, --repo
- Call GitHub backend when --github specified
- Show issue number in success output
- Update `runMessagesList()` to show GitHub issue column
- Update help text

**Files:**
- `cmd/ailang/messages.go` - Add flags and GitHub sync (~80 LOC)
- `cmd/ailang/messages_test.go` - NEW: CLI tests (~50 LOC)

**Acceptance Criteria:**
- [ ] `--github` flag triggers issue creation after message save
- [ ] `--type bug|feature|general` flag sets message_type
- [ ] `--repo owner/repo` overrides config default
- [ ] Issue number saved to database and displayed
- [ ] Error handling: message saved even if GitHub fails
- [ ] Help text updated with new flags

### Milestone 4: Import Command (M4-IMPORT)
**Goal:** Import GitHub issues as messages on startup
**Estimated:** ~120 LOC implementation + ~80 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `runMessagesImportGitHub()` function
- Query issues with configured watch_labels
- Skip already-imported issues (check github_issue_number)
- Create message from issue content
- Update session_start.sh hook

**Files:**
- `cmd/ailang/messages.go` - Add import-github subcommand (~100 LOC)
- `scripts/hooks/session_start.sh` - Add GitHub import call (~20 LOC)
- `cmd/ailang/messages_import_test.go` - NEW: Tests (~80 LOC)

**Acceptance Criteria:**
- [ ] `ailang messages import-github` imports new issues
- [ ] Duplicate issues not re-imported
- [ ] session_start.sh calls import-github automatically
- [ ] Import respects config.auto_import setting
- [ ] Import output shows count of new messages

### Milestone 5: Documentation & Polish (M5-DOCS)
**Goal:** Update documentation and add examples
**Estimated:** ~100 LOC
**Duration:** 0.5 days

**Tasks:**
- Update CLAUDE.md with GitHub sync workflow
- Add example config to docs
- Update messages help text
- Add error message guidance (gh install, auth switch)

**Files:**
- `CLAUDE.md` - Add GitHub messages section (~50 LOC)
- `cmd/ailang/messages.go` - Update printMessagesHelp (~30 LOC)
- `docs/guides/messages.md` - NEW: Messages guide (~100 LOC, optional)

**Acceptance Criteria:**
- [ ] CLAUDE.md documents GitHub sync workflow
- [ ] Example config shows all options
- [ ] Help text explains new flags
- [ ] Error messages provide fix instructions

## Success Metrics
- Test coverage: >80% for new code
- All tests passing
- `make lint` clean
- Examples: `ailang messages send --github --type bug` works end-to-end
- Documentation updated

## Dependencies
- gh CLI installed on system
- GitHub account authenticated via gh
- Config file at ~/.ailang/config.yaml with expected_user

## Open Questions
- None - design doc approved

## Timeline Summary

| Day | Milestone | Hours | LOC |
|-----|-----------|-------|-----|
| 1 (AM) | M1-SCHEMA | 4h | 200 |
| 1 (PM) | M2-GITHUB (start) | 4h | 175 |
| 2 (AM) | M2-GITHUB (complete) | 4h | 175 |
| 2 (PM) | M3-CLI | 4h | 150 |
| 3 (AM) | M4-IMPORT | 4h | 200 |
| 3 (PM) | M5-DOCS | 2h | 100 |

**Total: ~22 hours, ~1000 LOC**

## Notes
- Messages ALWAYS saved to SQLite first; GitHub is optional enhancement
- GitHub sync failures are non-fatal - clear error message, local save succeeds
- Account validation is strict (HARD FAIL) to prevent wrong-account issues
- Config file required for GitHub features; graceful skip if not present
