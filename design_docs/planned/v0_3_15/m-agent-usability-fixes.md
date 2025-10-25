# M-AGENT-USABILITY: Fix Agent Messaging UX Issues

**Status**: ✅ COMPLETE
**Target**: v0.3.15
**Priority**: P1 (Medium - Blocks autonomous agent workflows)
**Estimated**: 1 day (4h investigation + 2h fixes + 2h testing/docs)
**Actual**: 1 hour (documentation fixes only, no code changes needed)
**Dependencies**: None

## Implementation Summary

All documentation has been updated to use the correct `ailang agent send` syntax. The issues were:
1. ❌ References to non-existent `./bin/send-message` binary → ✅ Fixed (2 files updated)
2. ❌ Confusion about `--payload` flag → ✅ Clarified (doesn't exist, payload is positional arg)
3. ✅ `send_message.go` example program was already removed from codebase
4. ✅ All test commands verified working with complex JSON

**Files Updated:**
- `.claude/skills/agent-inbox/TEST_PROCEDURE.md` - Line 140-146
- `design_docs/implemented/v0_3_14/agent-inbox-acknowledgment.md` - Line 215

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Eliminates confusion about non-existent binaries |
| Preserve Semantic Clarity | 0 | 0 | No impact on language semantics |
| Increase Determinism | + | +1 | Consistent CLI interface reduces failures |
| Lower Token Cost | 0 | 0 | Marginal - better docs might slightly reduce context |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The agent messaging system has usability issues that prevent users (both human and AI) from successfully sending messages to agents.

**Current State:**
- Documentation and examples reference non-existent `./bin/send-message` binary
- Error messages when using correct `ailang agent send` syntax are confusing
- JSON payload escaping issues cause "invalid character" errors
- No clear examples showing correct syntax for complex payloads

**Impact:**
- **Who is affected?** Users trying to send messages to autonomous agents (sprint-planner, eval-analyzer, etc.)
- **How significant?** Blocks entire autonomous agent workflow - users cannot trigger agents at all
- **Evidence:** User attempted to send debug task to ailang-dev-cycle agent and hit multiple errors:
  1. `./bin/send-message` - no such file
  2. `ailang agent send --payload` - invalid JSON error with hyphens

## Goals

**Primary Goal:** Make agent messaging work reliably on first attempt with clear documentation

**Success Metrics:**
- Zero references to non-existent binaries in docs/examples
- All example commands in docs are copy-paste runnable
- JSON payloads with special characters work without manual escaping
- Clear error messages when syntax is wrong

## Solution Design

### Overview

This is primarily a **documentation and validation fix**, not a code change. We need to:
1. Audit all documentation for incorrect `./bin/send-message` references
2. Validate all example commands actually work
3. Add comprehensive examples showing edge cases (special chars, complex JSON)
4. Improve CLI error messages if needed

### Architecture

**No architectural changes** - the `ailang agent send` command works correctly. The issues are:
- **Documentation drift**: Examples got out of sync with implementation
- **Insufficient validation**: Examples aren't tested before being published
- **Missing edge cases**: No examples showing how to handle special characters

### Implementation Plan

**Phase 1: Documentation Audit** (~2 hours)
- [ ] Search codebase for all references to `send-message` binary
- [ ] Search for `--payload` flag usage (doesn't exist)
- [ ] Compile list of all agent send examples in docs
- [ ] Test each example command to verify it works
- [ ] Document which files need updates

**Phase 2: Fix Documentation** (~1.5 hours)
- [ ] Replace `./bin/send-message` with `ailang agent send`
- [ ] Fix payload syntax: `ailang agent send <agent-id> '<json>'`
- [ ] Add examples section showing:
  - Simple payload: `ailang agent send sprint-planner '{"task": "plan"}'`
  - Complex payload with special chars
  - Multi-line JSON formatting
  - Common errors and fixes
- [ ] Update CLAUDE.md with correct patterns
- [ ] Update skill docs if affected

**Phase 3: CLI Improvements (Optional)** (~0.5 hours)
- [ ] Check error message when JSON parsing fails
- [ ] Add hint about quoting if helpful
- [ ] Consider adding `--payload-file` flag for complex JSON

**Phase 4: Testing & Validation** (~2 hours)
- [ ] Test all updated examples manually
- [ ] Add test that validates example syntax (optional)
- [ ] Update agent-inbox skill test procedures
- [ ] Verify CLAUDE.md examples work

### Files to Modify/Create

**Documentation files to audit:**
- `CLAUDE.md` - Check for send-message references
- `docs/docs/guides/claude-code-integration.mdx` - Has multiple examples
- `docs/docs/guides/agent-workflows.mdx` - Agent messaging examples
- `docs/docs/guides/hooks-setup.mdx` - Hook examples using agent send
- `.claude/skills/agent-inbox/SKILL.md` - Skill documentation
- `.claude/skills/agent-inbox/TEST_PROCEDURE.md` - Test instructions
- `CHANGELOG.md` - Release notes with examples
- `design_docs/planned/M-CLAUDE-CODE-HEADLESS.md` - Future work docs
- `design_docs/implemented/v0_3_14/agent-inbox-acknowledgment.md` - Implementation docs

**Potential code changes:**
- `cmd/ailang/agent.go` - Improve error messages (optional, ~10 LOC)
- Add `--payload-file` flag support (optional, ~30 LOC)

## Examples

### Example 1: Simple Task Message (CORRECT)

**Before (BROKEN):**
```bash
# This doesn't exist!
./bin/send-message ailang-dev-cycle '{"task": "debug_and_fix"}'

# This syntax is wrong!
ailang agent send ailang-dev-cycle --payload '{"task": "debug"}'
```

**After (CORRECT):**
```bash
# Correct syntax: ailang agent send <agent-id> '<json-payload>'
ailang agent send ailang-dev-cycle '{"task": "debug_and_fix"}'
```

### Example 2: Complex JSON with Special Characters

**Problem:** JSON with hyphens, quotes, or newlines causes errors

**Solution 1 - Single quotes (recommended):**
```bash
ailang agent send sprint-planner '{
  "task": "debug_and_fix",
  "bug_description": "println import regression - examples fail",
  "test_file": "examples/snippets/v3_3/imports_basic.ail",
  "test_command": "./bin/ailang run --caps IO file.ail"
}'
```

**Solution 2 - Payload file (for very complex JSON):**
```bash
# Create payload file
cat > task.json <<'EOF'
{
  "task": "debug_and_fix",
  "bug_description": "Complex description with \"quotes\" and 'apostrophes'",
  "context": "Multi-line\ntext\nhere"
}
EOF

# Send from file (if we add --payload-file flag)
ailang agent send sprint-planner --payload-file task.json
```

### Example 3: Sending to User Inbox

**Correct syntax:**
```bash
# Note: --to-user flag, then payload (no agent-id needed)
ailang agent send --to-user '{"message": "Task complete", "status": "done"}'

# Can specify sender
ailang agent send --to-user --from "sprint-planner" '{"message": "Sprint plan ready"}'
```

## Success Criteria

- [ ] Zero grep hits for `./bin/send-message` in docs/examples
- [ ] Zero grep hits for `--payload` flag usage
- [ ] All example commands in docs are copy-paste runnable (tested)
- [ ] CLAUDE.md has correct agent send examples
- [ ] Agent-inbox skill docs are correct
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Validation script:**
```bash
# Extract all agent send commands from docs
grep -r "ailang agent send" docs/ CLAUDE.md .claude/ |
  grep -v "\.md.bak" |
  cut -d: -f2- > /tmp/agent_send_examples.txt

# Test each one (manual verification)
# Automated testing would require mocking or test inboxes
```

**Manual testing checklist:**
1. Send simple message: `ailang agent send sprint-planner '{"task": "test"}'`
2. Send complex JSON with special chars
3. Send to user inbox: `ailang agent send --to-user '{"test": "message"}'`
4. Trigger error: `ailang agent send` (no args) - check error message quality
5. Trigger JSON error: `ailang agent send test 'bad-json'` - check error message

## Non-Goals

**Not in this feature:**
- **Full CLI redesign** - Just fixing current issues
- **Interactive prompts** - Keep scriptable interface
- **YAML/TOML support** - JSON is sufficient for now
- **Agent response handling** - Separate feature (use `--wait` flag)

## Timeline

**Day 1** (6 hours):
- Morning (2h): Documentation audit, compile list of issues
- Afternoon (4h): Fix all documentation, update examples, test

**Total: ~6 hours across 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Examples in docs get stale again | Medium | Add CI check that validates example syntax |
| Complex JSON still hard to use | Low | Document `--payload-file` as optional enhancement |
| Miss some docs during audit | Low | Use comprehensive grep patterns |

## References

- Current implementation: [cmd/ailang/agent.go](../../cmd/ailang/agent.go) - Lines 317-340
- Agent inbox skill: [.claude/skills/agent-inbox/SKILL.md](../../.claude/skills/agent-inbox/SKILL.md)
- User guide: [docs/docs/guides/claude-code-integration.mdx](../../docs/docs/guides/claude-code-integration.mdx)
- Hooks guide: [docs/docs/guides/hooks-setup.mdx](../../docs/docs/guides/hooks-setup.mdx)

## Future Work

**Optional enhancements (v0.3.16+):**
- `ailang agent send --payload-file <file>` for complex JSON
- `ailang agent send --validate-only` to test JSON without sending
- `ailang agent send --wait` improvements (better timeout handling)
- Shell completion for agent IDs
- Interactive mode: `ailang agent send --interactive` (prompts for fields)

---

**Document created**: 2025-10-25
**Last updated**: 2025-10-25
