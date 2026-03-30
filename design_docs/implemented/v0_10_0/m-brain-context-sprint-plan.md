# Sprint Plan: M-BRAIN-CONTEXT

## Summary
Add contextual brain injection during active Claude Code sessions by hooking into `PreToolUse(Read)`, so brain knowledge surfaces when files are read — not just at session start.

**Duration:** 1 day (4-5 hours)
**Dependencies:** M-BRAIN (v0.9.2 — ✅ implemented)
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- ✅ M-BRAIN (v0.9.2): Full brain infrastructure — SQLite, CLI, SessionStart + commit hooks
- ✅ brain_session.sh: SessionStart injection from git diff context
- ✅ brain_resolution.sh: PostToolUse commit capture with enrichment

### Velocity
- Recent average: ~150-200 LOC/day (design docs + implementation)
- This sprint: ~125 LOC total — well within single-day capacity

### Remaining from Design Doc
- ⏳ M1: Core hook script (~80 LOC)
- ⏳ M2: Settings registration + integration test (~15 LOC config + manual test)
- ⏳ M3: Docs + CHANGELOG (~35 LOC)

## Proposed Milestones

### Milestone 1: M1_CORE_HOOK
**Goal:** Create `brain_context.sh` with cooldown, budget, and relevance filtering
**Estimated:** 80 LOC implementation + 0 LOC tests (shell script, tested manually) = 80 LOC
**Duration:** 2-3 hours

**Tasks:**
1. Create `~/.ailang/hooks/brain_context.sh` with:
   - Read hook JSON from stdin, extract file_path
   - File path filtering (skip binaries, node_modules, .git)
   - Directory-level cooldown via temp file (`/tmp/ailang_brain_cooldown_<pid>`)
   - Session token budget tracking (soft warning at limit)
   - `ailang cache search --context <file_path> --limit 2` query
   - Relevance threshold filtering (score >= 0.25)
   - Formatted output matching existing brain injection style
2. Make executable (`chmod +x`)
3. Test manually: `echo '{"tool_name":"Read","tool_input":{"file_path":"internal/types/unify.go"}}' | ~/.ailang/hooks/brain_context.sh`

**Acceptance Criteria:**
- [ ] Hook reads file_path from PreToolUse JSON stdin
- [ ] Skips binary/non-source files
- [ ] Directory cooldown prevents duplicate queries within 5 minutes
- [ ] Session budget tracks cumulative tokens, warns (not blocks) at limit
- [ ] Only outputs results with score >= 0.25
- [ ] Exits silently (exit 0) on any error, missing ailang, empty brain
- [ ] Output format matches brain_session.sh style

**Risks:**
- `ailang cache search --context` may not support file path input directly — Mitigation: fall back to `ailang cache search` with directory/filename as keyword

### Milestone 2: M2_REGISTER_AND_TEST
**Goal:** Register hook in settings.json and verify end-to-end
**Estimated:** 15 LOC config + manual testing = 15 LOC
**Duration:** 30 minutes

**Tasks:**
1. Add PreToolUse Read hook entry to `~/.claude/settings.json`
2. Verify hook fires on Read tool calls in a test session
3. Verify cooldown works (read two files in same dir, only one injection)
4. Verify budget tracking (multiple reads, check budget file)
5. Verify graceful degradation (test with empty brain)

**Acceptance Criteria:**
- [ ] Hook registered in `~/.claude/settings.json` PreToolUse section
- [ ] Hook fires when Read tool is used
- [ ] Cooldown prevents noise on multiple reads in same directory
- [ ] Budget file tracks cumulative token usage
- [ ] No errors when brain is empty or ailang unavailable

**Risks:**
- Existing PreToolUse hooks may conflict — Mitigation: check matcher specificity, ensure Read matcher coexists with `*` matcher

### Milestone 3: M3_DOCS_CHANGELOG
**Goal:** Document the feature and update CHANGELOG
**Estimated:** 35 LOC docs
**Duration:** 30 minutes

**Tasks:**
1. Update CHANGELOG.md with M-BRAIN-CONTEXT entry
2. Update design doc status if needed
3. Brief update to brain-cache docs if they exist

**Acceptance Criteria:**
- [ ] CHANGELOG.md updated with feature description
- [ ] Design doc references accurate
- [ ] All existing tests still pass (`make test`)

## Success Metrics
- Brain context injected when reading files with relevant brain knowledge
- No perceptible latency increase on Read calls
- Cooldown and budget prevent context window pollution
- All existing tests passing: ✅
- All linting passing: ✅

## Dependencies
- `ailang cache search` CLI must support file path context queries (already implemented in M-BRAIN)
- `~/.claude/settings.json` must support multiple PreToolUse entries (confirmed — existing `*` matcher + new `Read` matcher)

## Open Questions
- None — design freeze complete, all decisions resolved

## Notes
- This is infrastructure code (shell hooks + JSON config), not Go code — no `make test` coverage impact
- The hook lives in `~/.ailang/hooks/` (user-level), not in the project repo
- Impact measurement will come from real usage over the next week
