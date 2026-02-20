# M-DX34: Encourage Subagent Use in Core Skills

**Status:** 📋 Planned
**Priority:** P1 (Medium)
**Complexity:** S (1-2 days)
**Target Version:** v0.7.2
**Created:** 2026-02-02
**Author:** design-doc-creator
**GitHub Issue:** #128

---

## Problem Statement

Core skills in AILANG are not adequately utilizing the Task tool (subagent system) for complex, multi-step tasks that would benefit from parallelization and specialized expertise. This leads to:

**Current pain points:**
1. **Sequential execution** - Skills do single-threaded work when parallel subagents could speed up tasks
2. **Monolithic implementations** - Skills try to handle everything themselves rather than delegating
3. **Missed specialization** - Not leveraging specialized agents (e.g., Explore agent for codebase search)
4. **Reduced resilience** - No fallback when a single approach fails; subagents could try alternatives
5. **Poor scalability** - As tasks grow complex, single-agent approaches become unwieldy

**Concrete examples:**
- `eval-analyzer` could use Explore agent for finding patterns across benchmarks
- `sprint-planner` could launch parallel agents to analyze different design docs simultaneously
- `test-coverage-guardian` could use specialized agents for coverage analysis vs. test generation
- `codebase-organizer` could parallelize file analysis across multiple agents

**Impact:**
- **Performance:** Tasks that could complete in 2 minutes take 10+ minutes sequentially
- **Quality:** Missing insights that specialized agents would catch
- **Maintainability:** Skills become complex monoliths instead of orchestrators

---

## Goals

**Primary Goal:** Transform core skills from monolithic executors into intelligent orchestrators that leverage subagents for improved performance and quality.

**Success Metrics:**
- [ ] 10+ core skills updated with subagent orchestration patterns
- [ ] 50% reduction in execution time for complex multi-step tasks
- [ ] Documentation and examples showing when/how to use subagents
- [ ] Measurable improvement in task success rates through specialization
- [ ] Clear patterns established for parallel vs. sequential agent execution

---

## Solution Design

### Overview

Add explicit prompting and structural patterns to core skills that encourage subagent use when appropriate. This includes:

1. **Prompt engineering** - Clear instructions on when to use subagents
2. **Pattern library** - Common orchestration patterns (parallel search, staged pipeline, etc.)
3. **Skill templates** - Updated templates that include subagent sections
4. **Progressive disclosure** - Load subagent guidance only when relevant

### Architecture

```
Skill Execution Flow:
┌─────────────┐
│ Core Skill  │
└──────┬──────┘
       │ Analyzes task complexity
       ▼
┌──────────────┐
│ Orchestrator │ ← NEW: Decides subagent strategy
└──────┬───────┘
       │
       ├── Parallel Execution Pattern
       │   ├── Task Tool → Explore agent (search)
       │   ├── Task Tool → Analysis agent
       │   └── Task Tool → Validation agent
       │
       ├── Sequential Pipeline Pattern
       │   ├── Task Tool → Design agent
       │   └── Task Tool → Implementation agent
       │
       └── Hybrid Pattern
           ├── Parallel search phase
           └── Sequential implementation phase
```

### Implementation Plan

**Phase 1: Pattern Development (4 hours)**
- [ ] Document common subagent orchestration patterns
- [ ] Create decision matrix for when to use subagents
- [ ] Write example code showing Task tool usage

**Phase 2: Core Skill Updates (6 hours)**
- [ ] Update eval-analyzer to use Explore for pattern search
- [ ] Update sprint-planner for parallel design doc analysis
- [ ] Update test-coverage-guardian for parallel coverage checks
- [ ] Update codebase-organizer for parallel file analysis
- [ ] Update perf-reviewer for parallel benchmark runs
- [ ] Update design-spec-auditor for parallel spec validation
- [ ] Update github-issue-triage for parallel issue analysis
- [ ] Update docs-sync for parallel doc validation
- [ ] Update builtin-developer for parallel validation checks
- [ ] Update model-manager for parallel model testing

**Phase 3: Documentation & Templates (2 hours)**
- [ ] Update skill-builder template with subagent sections
- [ ] Create "When to Use Subagents" guide
- [ ] Add examples to each updated skill
- [ ] Update CLAUDE.md with orchestration patterns

### Files to Modify

**New Files:**
- `.claude/skills/README_SUBAGENTS.md` (~200 lines) - Orchestration patterns guide
- `.claude/skills/skill-builder/templates/skill_with_subagents.md` (~100 lines)
- `design_docs/implemented/v0_7_2/m-dx34-subagent-encouragement.md` (~150 lines)

**Modified Files:**
- `.claude/skills/eval-analyzer/SKILL.md` (+50 lines) - Add subagent patterns
- `.claude/skills/sprint-planner/SKILL.md` (+50 lines) - Parallel analysis
- `.claude/skills/test-coverage-guardian/SKILL.md` (+50 lines) - Parallel coverage
- `.claude/skills/codebase-organizer/SKILL.md` (+50 lines) - Parallel file analysis
- `.claude/skills/perf-reviewer/SKILL.md` (+50 lines) - Parallel benchmarks
- `.claude/skills/design-spec-auditor/SKILL.md` (+40 lines) - Parallel validation
- `.claude/skills/github-issue-triage/SKILL.md` (+40 lines) - Parallel triage
- `.claude/skills/docs-sync/SKILL.md` (+40 lines) - Parallel sync
- `.claude/skills/builtin-developer/SKILL.md` (+40 lines) - Parallel validation
- `.claude/skills/model-manager/SKILL.md` (+40 lines) - Parallel testing
- `.claude/skills/skill-builder/templates/skill_template.md` (+30 lines) - Subagent section
- `CLAUDE.md` (+100 lines) - Orchestration patterns documentation

**Total:** ~1,200 lines (600 new, 600 modified)

---

## Examples

### Before: Sequential Pattern Search (eval-analyzer)
```markdown
## Current Workflow

When analyzing eval results:
1. Read each benchmark file sequentially
2. Search for patterns one at a time
3. Process results linearly
4. Generate report

Time: ~10 minutes for full baseline
```

### After: Parallel Subagent Orchestration
```markdown
## Orchestrated Workflow

When analyzing eval results:
1. **Decide orchestration strategy** based on task size
2. **Launch parallel subagents:**
   ```
   # Parallel pattern search
   - Task tool → Explore agent: "Find all error patterns in eval_results/"
   - Task tool → Explore agent: "Search for timeout patterns"
   - Task tool → Analysis agent: "Analyze token usage patterns"
   ```
3. **Collect and merge results** from all agents
4. **Generate unified report** with cross-agent insights

Time: ~2 minutes (5x faster through parallelization)
```

### Common Orchestration Patterns

```markdown
## 1. Parallel Search Pattern
When: Multiple independent searches needed
Example: Finding all uses of a deprecated API

Launch multiple Explore agents concurrently:
- Agent 1: Search in `internal/`
- Agent 2: Search in `cmd/`
- Agent 3: Search in `tools/`

## 2. Staged Pipeline Pattern
When: Tasks have dependencies
Example: Design → Sprint → Implementation

Sequential agents with handoff:
- design-doc-creator → sprint-planner → sprint-executor

## 3. Map-Reduce Pattern
When: Processing many similar items
Example: Analyzing 50 benchmark results

Map phase (parallel):
- 10 agents each process 5 benchmarks
Reduce phase:
- Single agent merges all results

## 4. Specialist Team Pattern
When: Different expertise needed
Example: Full codebase review

Parallel specialists:
- Security agent: Check for vulnerabilities
- Performance agent: Find bottlenecks
- Style agent: Check conventions
- Test agent: Verify coverage
```

---

## Success Criteria

- [ ] 10+ core skills updated with subagent orchestration sections
- [ ] Documented patterns for parallel, sequential, and hybrid orchestration
- [ ] Measurable performance improvements (benchmark before/after)
- [ ] Examples showing real Task tool invocations
- [ ] skill-builder template includes subagent guidance
- [ ] CLAUDE.md updated with "When to Use Subagents" section
- [ ] No regressions in existing skill functionality
- [ ] Clear decision matrix for subagent vs. direct execution

---

## Timeline

**Week 1 (Days 1-2):**
- Day 1 AM: Document orchestration patterns, create decision matrix
- Day 1 PM: Update first 5 core skills (eval-analyzer, sprint-planner, etc.)
- Day 2 AM: Update remaining 5 core skills
- Day 2 PM: Documentation, templates, testing, and validation

**Buffer:** 0.5 days for testing and refinement

---

## Related Documents

- [M-AGENT-PROTOCOL](../../implemented/v0_5_0/M-AGENT-PROTOCOL.md) - Agent communication foundation
- [Agent Tool Documentation](../../../CLAUDE.md#available-tools) - Task tool reference
- [Skill Builder Guide](../../../.claude/skills/SKILLS_GUIDE.md) - Skill development patterns
- [Coordinator Architecture](../../../docs/docs/guides/coordinator.md) - Multi-agent orchestration

---

## Notes

- Focus on high-value skills that process multiple files or run multiple analyses
- Ensure backward compatibility - skills should still work without subagents
- Add progressive disclosure - only show subagent options when relevant
- Consider adding metrics to track subagent usage and performance gains
- Future work: Auto-detect when subagents would help based on task complexity

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| **A1: Determinism** | +1 | Subagent results are deterministic and reproducible |
| **A2: Replayability** | +1 | Agent traces improve audit and debugging |
| **A3: Effect Legibility** | 0 | No change to effect visibility |
| **A4: Explicit Authority** | 0 | Uses existing capability model |
| **A5: Bounded Verification** | +1 | Each subagent verifiable independently |
| **A6: Safe Concurrency** | +1 | Parallel agents have no shared state |
| **A7: Machines First** | +1 | Improves machine-to-machine coordination |
| **A8: Minimal Syntax** | 0 | No syntax changes |
| **A9: Cost Visibility** | +1 | Agent costs tracked per invocation |
| **A10: Composability** | +1 | Agents compose without interference |
| **A11: Structured Failure** | +1 | Agent failures are structured data |
| **A12: System Boundary** | 0 | No boundary changes |

**Net Score: +8** ✅ Strong alignment with axioms