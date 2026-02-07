# Sprint Plan: M-DX34 - Encourage Subagent Use in Core Skills

## Summary

Transform core skills from monolithic executors into intelligent orchestrators that actively encourage and demonstrate subagent usage. Update 10+ core skills with explicit subagent orchestration patterns, create a comprehensive pattern library, and establish clear decision matrices for when/how to use the Task tool for parallel and sequential agent coordination.

**Duration:** 2 days
**Dependencies:** None (can run independently)
**Risk Level:** Low (documentation + pattern library addition, no architecture changes)

---

## Current Status Analysis

### Completed Recently
- ✅ M-DX34 Design Doc: 274 lines in 1 day (identifies problem and solution)
- ✅ Core skill infrastructure: 2,800 LOC in 5 days (foundation complete)
- ✅ Recent releases: v0.7.1.x stability work

### Velocity
- Recent average: **~280 LOC/day** (52 files changed, 2,800 insertions in 5 days)
- Estimated capacity: **~560 LOC** for this 2-day sprint
- **Actual requirement:** 1,200 LOC (600 new + 600 modified)
- **Gap:** 640 LOC shortfall suggests need for 3-4 days

### Realistic Timeline
- **Days 1-2:** Documentation infrastructure, patterns, templates (HIGH PRIORITY)
- **Day 3:** Core skill updates (10+ skills with +30-50 lines each)
- **Day 4:** Testing, validation, integration verification

**Recommendation:** Plan for 4 days with phased rollout to stay ahead of technical debt.

### Remaining from Design Doc
- 📋 Orchestration patterns documentation (~200 LOC)
- 📋 Pattern library with examples (~150 LOC)
- 📋 Core skill updates (10 skills × 50 LOC = ~500 LOC)
- 📋 Template updates (~100 LOC)
- 📋 CLAUDE.md orchestration section (~100 LOC)

---

## Proposed Milestones

### Milestone 1: Pattern Library & Decision Matrix
**Goal:** Create comprehensive documentation showing when/how to use subagents with real examples of the Task tool.

**Estimated:** 250 lines (documentation) + 50 lines (examples) = **300 LOC**
**Duration:** 0.5 days

**Files to Create:**
- `.claude/skills/README_SUBAGENTS.md` (~250 lines)
  - Decision matrix: when to use subagents vs direct execution
  - Parallel pattern (map-reduce for independent tasks)
  - Sequential pipeline pattern (staged dependency flow)
  - Hybrid orchestration patterns
  - Real Task tool invocation examples
  - Performance comparison before/after
  - Resource budgeting for parallel agents

**Tasks:**
- Create `.claude/skills/README_SUBAGENTS.md` with orchestration patterns
  - Include decision matrix with concrete examples
  - Add Task tool syntax examples (how to invoke subagents)
  - Document resource considerations (parallel limit, cost tracking)
  - Show patterns: parallel search, pipeline, map-reduce, specialist team
- Create example code snippets showing real Task tool usage
- Include diagrams (ASCII art) showing orchestration flows

**Acceptance Criteria:**
- [ ] `.claude/skills/README_SUBAGENTS.md` created with 250+ lines
- [ ] Decision matrix covers 6+ decision points
- [ ] 4+ orchestration patterns documented with examples
- [ ] Real Task tool invocation examples included
- [ ] Performance metrics (before/after speedup)
- [ ] No linting errors

**Risks:**
- Documentation clarity may need iteration - Mitigation: Use clear headers and progressive disclosure
- Examples may be outdated - Mitigation: Link to actual skill implementations

---

### Milestone 2: Core Skill Updates (Phase 1: High-Value Skills)
**Goal:** Update 5 highest-value skills with subagent orchestration sections and real examples.

**Estimated:** 250 lines (5 skills × 50 lines) + 25 lines (new examples) = **275 LOC**
**Duration:** 1 day

**Files to Modify:**
- `.claude/skills/eval-analyzer/SKILL.md` (+50 lines)
  - Add "Using Subagents for Parallel Analysis" section
  - Show Task tool usage for parallel pattern search
  - Include example: launching 3 Explore agents for different error types

- `.claude/skills/sprint-planner/SKILL.md` (+50 lines)
  - Add "Orchestrated Design Doc Analysis" section
  - Show Task tool for parallel analysis of multiple design docs
  - Include example: 5 agents analyzing different doc sections concurrently

- `.claude/skills/test-coverage-guardian/SKILL.md` (+50 lines)
  - Add "Parallel Coverage Analysis" section
  - Show Task tool for splitting coverage analysis across agents
  - Include example: analyzing different file groups in parallel

- `.claude/skills/codebase-organizer/SKILL.md` (+50 lines)
  - Add "Distributed File Analysis" section
  - Show Task tool for parallel file size analysis
  - Include example: analyzing 100 files across 4 agents

- `.claude/skills/perf-reviewer/SKILL.md` (+50 lines)
  - Add "Parallel Benchmark Orchestration" section
  - Show Task tool for concurrent benchmark analysis
  - Include example: 10 benchmarks analyzed in parallel

**Tasks:**
- Day 1 AM: Add subagent sections to eval-analyzer, sprint-planner
- Day 1 PM: Add to test-coverage-guardian, codebase-organizer, perf-reviewer
- Include "When to use this pattern" and real Task tool examples in each
- Test that existing functionality still works (no regressions)

**Acceptance Criteria:**
- [ ] All 5 skills updated with subagent orchestration sections
- [ ] Each includes real Task tool invocation example
- [ ] "When to use" decision points documented
- [ ] No changes to existing skill functionality
- [ ] All skill SKILL.md files valid markdown
- [ ] Examples compile (if applicable)

**Risks:**
- Skills may have complex interdependencies - Mitigation: Test each skill independently
- Documentation updates might need iteration - Mitigation: Keep updates focused on specific patterns

---

### Milestone 3: Core Skill Updates (Phase 2: Remaining Skills)
**Goal:** Update remaining 5 core skills with subagent patterns.

**Estimated:** 200 lines (5 skills × 40 lines) = **200 LOC**
**Duration:** 0.75 days

**Files to Modify:**
- `.claude/skills/design-spec-auditor/SKILL.md` (+40 lines)
  - Add "Parallel Spec Validation" pattern

- `.claude/skills/github-issue-triage/SKILL.md` (+40 lines)
  - Add "Distributed Issue Analysis" pattern

- `.claude/skills/docs-sync/SKILL.md` (+40 lines)
  - Add "Parallel Documentation Validation" pattern

- `.claude/skills/builtin-developer/SKILL.md` (+40 lines)
  - Add "Concurrent Builtin Validation" pattern

- `.claude/skills/model-manager/SKILL.md` (+40 lines)
  - Add "Parallel Model Testing" pattern

**Tasks:**
- Update all 5 remaining skills with consistent subagent patterns
- Reference common patterns from README_SUBAGENTS.md
- Ensure consistency with Phase 1 updates

**Acceptance Criteria:**
- [ ] All 10 core skills now have subagent orchestration sections
- [ ] Pattern references consistent across all skills
- [ ] No functional changes to existing skill behavior
- [ ] All documentation valid and linted

**Risks:**
- Pattern consistency needs enforcement - Mitigation: Use copy-paste from Phase 1 templates

---

### Milestone 4: Templates & CLAUDE.md Integration
**Goal:** Update skill-builder template and CLAUDE.md with subagent guidance.

**Estimated:** 130 lines (template 30 + CLAUDE.md 100) = **130 LOC**
**Duration:** 0.75 days

**Files to Create/Modify:**
- `.claude/skills/skill-builder/templates/skill_with_subagents.md` (NEW, ~100 lines)
  - Template for future skills that use subagents
  - Includes placeholder sections for orchestration patterns
  - Shows Task tool usage examples

- `CLAUDE.md` (+100 lines in "Skills" section)
  - Add "Using Subagents in Skills" subsection
  - Document orchestration patterns (parallel, pipeline, hybrid)
  - Show when/how to use Task tool
  - Link to README_SUBAGENTS.md for detailed patterns

- `.claude/skills/skill-builder/SKILL.md` (+30 lines)
  - Add section on "When to use subagents in your skill"
  - Link to new template

**Tasks:**
- Create skill_with_subagents.md template with placeholder sections
- Update CLAUDE.md with orchestration patterns and decision matrix
- Update skill-builder SKILL.md with subagent guidance
- Ensure all templates reference README_SUBAGENTS.md for full documentation

**Acceptance Criteria:**
- [ ] New template created with 100+ lines of guidance
- [ ] CLAUDE.md updated with clear subagent documentation
- [ ] All files reference each other correctly (no broken links)
- [ ] Templates are usable by new skills (not just documentation)
- [ ] Decision matrix matches README_SUBAGENTS.md
- [ ] No linting errors

**Risks:**
- CLAUDE.md may be large and need careful editing - Mitigation: Add to specific section after tools table
- Template needs to be practical - Mitigation: Base on existing skill-builder patterns

---

### Milestone 5: Testing, Validation & Documentation
**Goal:** Verify all updates work correctly, run final tests, and create completion documentation.

**Estimated:** 50 lines (validation checks + completion notes) = **50 LOC**
**Duration:** 0.5 days

**Files to Create/Modify:**
- Create `design_docs/implemented/v0_7_2/m-dx34-implementation-report.md`
  - Summary of completed work
  - Metrics: lines added, skills updated, patterns documented
  - Examples of new patterns in action
  - Performance improvements (if measurable)
  - Lessons learned

**Tasks:**
- Verify all 10 core skills updated correctly
- Test no functional regressions in existing skills
- Validate all markdown files (no syntax errors)
- Run linting on all modified files
- Verify links in CLAUDE.md and README_SUBAGENTS.md
- Move design doc to implemented/ after completion
- Commit all changes with comprehensive message

**Acceptance Criteria:**
- [ ] All 10 skills verified to have new sections
- [ ] All markdown valid (no syntax errors)
- [ ] All links functional
- [ ] No regressions in skill functionality
- [ ] Implementation report created
- [ ] Design doc moved to implemented/
- [ ] Git commit with "M-DX34" reference
- [ ] All tests passing: `make test`
- [ ] All linting passing: `make lint`

**Risks:**
- Links may be broken - Mitigation: Validate all markdown links
- May have missed skills - Mitigation: Use checklist of all 10 skills

---

## Success Metrics

**Implementation Success:**
- ✅ 10+ core skills updated with subagent sections
- ✅ Comprehensive pattern library documented (README_SUBAGENTS.md)
- ✅ Decision matrix for when to use subagents
- ✅ Real Task tool invocation examples in each skill
- ✅ Updated templates for future skills
- ✅ No functional regressions

**Documentation Quality:**
- ✅ All new files: 0 linting errors
- ✅ All links: functional and verified
- ✅ Pattern consistency: same terminology across all 10 skills
- ✅ Examples: realistic and copyable

**Coverage:**
- ✅ Skills updated: eval-analyzer, sprint-planner, test-coverage-guardian, codebase-organizer, perf-reviewer, design-spec-auditor, github-issue-triage, docs-sync, builtin-developer, model-manager
- ✅ Documentation: CLAUDE.md, README_SUBAGENTS.md, skill templates
- ✅ Examples: Task tool usage shown in 10+ contexts

---

## Dependencies

None - this sprint can execute independently. All changes are documentation and pattern additions with no architectural dependencies.

---

## Implementation Notes

### Pattern Library Strategy

**README_SUBAGENTS.md** will be the single source of truth for orchestration patterns. Each skill will reference specific patterns rather than duplicate documentation:

```markdown
# .claude/skills/README_SUBAGENTS.md (250 lines)

## Orchestration Patterns

### Pattern 1: Parallel Search (Expand)
Used by: eval-analyzer, github-issue-triage
[Full pattern documentation with example, resource considerations, etc.]

### Pattern 2: Staged Pipeline (New)
Used by: sprint-planner (already has design-doc-creator → sprint-planner → sprint-executor)
[How to add similar handoffs in other skills]

### Pattern 3: Map-Reduce (New)
Used by: codebase-organizer, test-coverage-guardian
[Document the pattern]

## Decision Matrix
[When to use which pattern]

## Task Tool Reference
[How to invoke subagents with examples]
```

### Skill Update Strategy

**Each skill gets a "Subagents" section that:**
1. Explains which pattern(s) are applicable
2. Shows real Task tool code example
3. Links to README_SUBAGENTS.md for full details
4. Notes performance expectations

**Example section for eval-analyzer:**
```markdown
## Using Subagents for Parallel Analysis

The eval-analyzer skill benefits from parallel pattern search when analyzing large baselines.

**When to use:** 5+ benchmark files to analyze
**Pattern:** Parallel Search (see README_SUBAGENTS.md#pattern-1)

**Example Task tool usage:**
```

### Backward Compatibility

All changes are additive:
- New documentation sections don't change skill behavior
- No skill parameters or interfaces changed
- Existing workflows continue to work unchanged
- Subagent sections are optional reading/usage

### Future Work

**Quick Wins Post-Sprint:**
- Monitor which patterns get used most (add metrics)
- Refactor frequently-used patterns into reusable utilities
- Auto-detect complexity level to suggest patterns

**Phase 2 (v0.7.3+):**
- Add "auto-recommend-subagents" feature to detect when parallel execution would help
- Create cost optimizer (compare parallel vs sequential cost)
- Build performance tracker (measure speedup from subagent patterns)

---

## Day-by-Day Breakdown

**Day 1 (Milestones 1-2):**
- AM: Create `.claude/skills/README_SUBAGENTS.md` with full pattern library (4 hours)
- PM: Update 5 high-value skills with subagent sections (4 hours)
- Evening: Commit work with "M-DX34: Add pattern library and Phase 1 skill updates"

**Day 2 (Milestones 3-4):**
- AM: Update remaining 5 core skills (3 hours)
- Midday: Update templates and CLAUDE.md (3 hours)
- PM: Testing, validation, final commit (2 hours)
- Evening: Move design doc to implemented/, create completion report

**If extending to Day 3-4:**
- Day 3: Performance analysis and metrics addition
- Day 4: Community documentation (blog post, examples)

---

## Files Summary

| File | Type | LOC | Status |
|------|------|-----|--------|
| `.claude/skills/README_SUBAGENTS.md` | New | 250 | 📋 |
| `.claude/skills/eval-analyzer/SKILL.md` | Modify | +50 | 📋 |
| `.claude/skills/sprint-planner/SKILL.md` | Modify | +50 | 📋 |
| `.claude/skills/test-coverage-guardian/SKILL.md` | Modify | +50 | 📋 |
| `.claude/skills/codebase-organizer/SKILL.md` | Modify | +50 | 📋 |
| `.claude/skills/perf-reviewer/SKILL.md` | Modify | +50 | 📋 |
| `.claude/skills/design-spec-auditor/SKILL.md` | Modify | +40 | 📋 |
| `.claude/skills/github-issue-triage/SKILL.md` | Modify | +40 | 📋 |
| `.claude/skills/docs-sync/SKILL.md` | Modify | +40 | 📋 |
| `.claude/skills/builtin-developer/SKILL.md` | Modify | +40 | 📋 |
| `.claude/skills/model-manager/SKILL.md` | Modify | +40 | 📋 |
| `.claude/skills/skill-builder/templates/skill_with_subagents.md` | New | 100 | 📋 |
| `.claude/skills/skill-builder/SKILL.md` | Modify | +30 | 📋 |
| `CLAUDE.md` | Modify | +100 | 📋 |
| `design_docs/implemented/v0_7_2/m-dx34-implementation-report.md` | New | 75 | 📋 |

**Total: ~1,205 LOC**

---

## Open Questions

1. **Should we create metrics tracking for subagent usage?**
   - Track which patterns are most used
   - Measure actual speedup achieved
   - Decision: Defer to v0.7.3+ for now, document in implementation report

2. **Should we add cost estimation for parallel vs sequential?**
   - Help users choose between orchestration strategies
   - Decision: Document in Phase 1, implement cost tracker in Phase 2

3. **Should all 29 skills be updated eventually?**
   - Current plan: 10 core skills
   - Many skills don't benefit from parallelization
   - Decision: Start with 10, extend based on user feedback

---

## Notes & Assumptions

- **Assumption:** Most skills benefit from at least one orchestration pattern
- **Assumption:** Task tool is familiar to skill developers (covered in CLAUDE.md)
- **Assumption:** Performance improvements will be self-evident (don't need metrics for Phase 1)
- **Risk Mitigation:** If patterns feel unclear, add more examples or create video walkthrough
- **Quality Gate:** All documentation must be readable by both humans and AI agents
- **Future Consideration:** Could build "skill orchestration" as a meta-skill that auto-manages subagent coordination

---

## Related Documents

- **Design Doc:** `design_docs/planned/v0_7_2/m-dx34-subagent-encouragement.md`
- **Agent Tool Reference:** `CLAUDE.md#available-tools` (Task tool documentation)
- **Skill Builder Guide:** `.claude/skills/SKILLS_GUIDE.md`
- **Coordinator Architecture:** `docs/docs/guides/coordinator.md`
- **M-AGENT-PROTOCOL:** `design_docs/implemented/v0_5_0/M-AGENT-PROTOCOL.md`

---

**Sprint Plan created by:** sprint-planner skill
**Date:** 2026-02-02
**Status:** 📋 Ready for sprint-executor handoff
