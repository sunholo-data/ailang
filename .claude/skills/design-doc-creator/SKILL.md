---
name: Design Doc Creator
description: Create AILANG design documents in the correct format and location. Use when user asks to create a design doc, plan a feature, or document a design. Handles both planned/ and implemented/ docs with proper structure.
---

# Design Doc Creator

Create well-structured design documents for AILANG features following the project's conventions.

## Quick Start

**Most common usage:**
```bash
# User says: "Create a design doc for better error messages"
# This skill will:
# 1. AUTO-SEARCH for related design docs (instant SimHash)
# 2. Show matches from implemented/ and planned/ directories
# 3. Auto-populate "Related Documents" section in template
# 4. Ask for key details (priority, version target)
# 5. Create design_docs/planned/better-error-messages.md
# 6. Fill template with proper structure
```

**Automatic Related Doc Search (v0.6.0+):**

When you run the create script, it automatically:
1. Converts doc name to search query (e.g., `m-dx2-better-errors` → `"better errors"`)
2. Searches both `implemented/` and `planned/` directories using SimHash
3. Shows top 3 matches with similarity scores
4. Prompts to continue if matches found
5. Auto-populates the "Related Documents" section in the template

```bash
$ .claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-semantic-caching

🔍 Searching for related design docs...

Implemented docs matching "semantic caching":
1. design_docs/implemented/v0_5_11/m-doc-sem-lazy-embeddings.md (0.92)
2. design_docs/implemented/v0_4_0/monomorphization.md (0.78)

Planned docs matching "semantic caching":
1. design_docs/planned/v0_6_0/semantic-caching-future.md (0.95)

⚠ Related docs found. Review them before proceeding?
Continue creating new doc? (Y/n)
```

## When to Use This Skill

Invoke this skill when:
- User asks to "create a design doc" or "write a design doc"
- User says "plan a feature" or "design a feature"
- User mentions "document the design" or "create a spec"
- Before starting implementation of a new feature
- After completing a feature (to move to implemented/)

## Coordinator Integration

**When invoked by the AILANG Coordinator** (detected by GitHub issue reference in the prompt), you MUST output this marker at the end of your response:

```
DESIGN_DOC_PATH: design_docs/planned/vX_Y/design-doc-name.md
```

**Why?** The coordinator uses this marker to:
1. Read the design doc content for GitHub comments
2. Track artifacts across pipeline stages
3. Provide visibility to humans reviewing the issue

**Example completion:**
```
## Design Document Created

I've created the design document...

**DESIGN_DOC_PATH**: `design_docs/planned/v0_6_3/m-feature-design.md`
```

## Available Scripts

### `scripts/create_planned_doc.sh <doc-name> [version]`
Create a new design document in `design_docs/planned/`.

**Version Auto-Detection:**
The script automatically detects the current AILANG version from `CHANGELOG.md` and suggests the next version folder. This prevents accidentally placing docs in wrong version folders.

**Usage:**
```bash
# See current version and suggested target
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh
# Output: Current AILANG version: v0.5.6
#         Suggested next version: v0_5_7

# Create doc in planned/ root (no version)
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-dx2-better-errors

# Create doc in next version folder (recommended)
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh reflection-system v0_5_7
```

**What it does:**
- **Searches for related docs** using `ailang docs search` (SimHash, instant)
- Shows top 3 matches from both `implemented/` and `planned/`
- Auto-populates "Related Documents" section with clickable links
- Detects current version from CHANGELOG.md
- Suggests next patch version for targeting
- Creates design doc from template
- Places in correct directory (planned/ or planned/VERSION/)
- Fills in creation date
- Shows version context in output

### `scripts/move_to_implemented.sh <doc-name> <version>`
Move a design document from planned/ to implemented/ after completion.

**Usage:**
```bash
.claude/skills/design-doc-creator/scripts/move_to_implemented.sh m-dx1-developer-experience v0_3_10
```

**What it does:**
- Finds doc in planned/
- Copies to implemented/VERSION/
- Updates status to "Implemented"
- Updates last modified date
- Provides template for implementation report
- Keeps original until you verify and commit

## Workflow

### Creating a Planned Design Doc

**1. Gather Requirements**

Ask user:
- What feature are you designing?
- What version is this targeted for? (e.g., v0.4.0)
- What priority? (P0/P1/P2)
- Estimated effort? (e.g., 2 days, 1 week)
- Any dependencies on other features?

**⚠️ CRITICAL: Audit for Systemic Issues FIRST**

**Before writing a design doc for a bug fix, ALWAYS ask: "Is this part of a larger pattern?"**

**The Anti-Pattern (incremental special-casing):**
```
v1: Add feature for case A
v2: Bug! Add special case for B
v3: Bug! Add special case for C
v4: Bug! Add special case for D
...forever patching
```

**The Pattern to Follow (unified solutions):**
```
v1: Bug report for case B
    BEFORE writing design doc:
    1. Search for similar code paths
    2. Check if A, C, D have same gap
    3. Design ONE fix covering ALL cases
v2: Unified fix - no future bugs in this area
```

**Concrete Example (M-CODEGEN-UNIFIED-SLICE-CONVERTERS, Dec 2025):**
```
Bug reported: [SolarPlanet] return type panics

❌ Quick fix design doc: Add ConvertToSolarPlanetSlice
   (Will need ConvertToAnotherRecordSlice later...)

✅ Systemic design doc: Audit ALL slice types
   Found: []float64 ALSO broken!
   Found: []*ADTType partially broken!
   One unified fix covers all 3 gaps.
```

**Analysis Checklist (do BEFORE writing design doc):**
- [ ] Is this a one-off or part of a pattern?
- [ ] Search codebase for similar code paths
- [ ] Check if other types/cases have the same gap
- [ ] Look at git history - has this area been patched repeatedly?
- [ ] Design fix to cover ALL cases, not just the reported one

**🔍 Use `ailang docs search` to Check Existing Work:**

Before creating a design doc, search for existing implementations:

```bash
# Check if feature already implemented
ailang docs search --stream implemented "feature keywords"

# Check if design doc already planned
ailang docs search --stream planned "feature keywords"

# Use neural search for semantic matching (requires Ollama, slow ~20-30s)
ailang docs search --stream planned --neural "semantic description of feature"

# Search with JSON output for programmatic use
ailang docs search --stream implemented --json "keywords"
```

**Example workflow:**
```bash
# Before creating "lazy embeddings" design doc:
$ ailang docs search --stream implemented "embedding cache"
🔍 SimHash search: "embedding cache"
   Scanned: 42 docs

No matching documents found.
# ✅ Safe to create - not implemented yet

$ ailang docs search --stream planned "lazy embedding"
🔍 SimHash search: "lazy embedding"
   Scanned: 28 docs

1. design_docs/planned/v0_5_11/m-doc-sem-lazy-embeddings.md (0.85)
# ⚠️ Already planned - review existing doc first
```

**Key flags:**
- `--stream implemented` - Only search implemented/ directory
- `--stream planned` - Only search planned/ directory
- `--neural` - Use semantic embeddings (finds conceptually similar docs, **slow: ~20-30s**)
- `--limit N` - Return top N results
- `--json` - JSON output for scripting

**Performance note:** SimHash search (without `--neural`) is instant. Only use `--neural` when keyword matching isn't finding what you need.

**Warning Signs of Fragmented Design:**
- Multiple maps tracking similar things (`adtSliceTypes`, `recordTypes`...)
- Switch statements with growing case lists
- Functions named `handleX`, `handleY`, `handleZ` instead of unified `handle`
- Bug fixes that add `|| specialCase` conditions

**When these signs appear:** Expand scope of design doc to unify the system.

**⚠️ IMPORTANT: Axiom Compliance Check**

**Every feature must align with AILANG's 12 Design Axioms.**

The axioms are the non-negotiable principles that define AILANG's semantics. Any feature that violates an axiom must justify why the axiom itself should change.

📖 **Canonical Reference:** [Design Axioms](/docs/references/axioms) on the website

### The 12 Axioms (Quick Reference)

| # | Axiom | Core Principle |
|---|-------|----------------|
| 1 | **Determinism** | Execution must be replayable; nondeterminism is explicit, typed, localized |
| 2 | **Replayability** | Traces are inspectable, serializable, and suitable for verification |
| 3 | **Effects Are Legible** | Side effects are explicit, typed, and machine-readable |
| 4 | **Explicit Authority** | No implicit access; capabilities are declared and constrained |
| 5 | **Bounded Verification** | Local, automatable checks; not whole-program proofs |
| 6 | **Safe Concurrency** | Parallelism must not introduce scheduling-dependent meaning |
| 7 | **Machines First** | Semantic compression, decidable structure, toolability |
| 8 | **Syntax Is Liability** | Only add syntax that reduces ambiguity or improves analysis |
| 9 | **Cost Is Meaning** | Resource implications visible in types and to tools |
| 10 | **Composability** | Features must compose without semantic blind spots |
| 11 | **Failure Is Data** | Errors are structured, typed, inspectable, composable |
| 12 | **System Boundary** | Crossing between intent/execution, authority/action is explicit |

### 🧭 Axiom Scoring Matrix

**Score the proposed feature against each applicable axiom:**

| Axiom | −1 (violates) | 0 (neutral) | +1 (strengthens) |
|-------|---------------|-------------|------------------|
| **A1: Determinism** | adds implicit nondeterminism | — | increases reproducibility |
| **A2: Replayability** | breaks trace reconstruction | — | improves audit/replay |
| **A3: Effect Legibility** | hides side effects | — | makes effects more visible |
| **A4: Explicit Authority** | grants ambient access | — | enforces capability bounds |
| **A5: Bounded Verification** | requires global reasoning | — | enables local checks |
| **A6: Safe Concurrency** | introduces races | — | preserves determinism |
| **A7: Machines First** | optimizes for human prose | — | improves machine analysis |
| **A8: Minimal Syntax** | adds syntactic surface | — | reduces ambiguity |
| **A9: Cost Visibility** | hides resource costs | — | makes costs explicit |
| **A10: Composability** | breaks when combined | — | composes cleanly |
| **A11: Structured Failure** | uses unstructured exceptions | — | typed error handling |
| **A12: System Boundary** | implicit boundary crossing | — | explicit transitions |

### Decision Rules

**Hard violations (automatic rejection):**
- Any score of **−1** on Axioms 1, 3, 4, or 7 → **REJECT** (core invariants)

**Soft scoring:**
- Net score across all axioms **≥ +2**: Move forward to draft
- Net score **0 to +1**: Needs stronger justification
- Net score **< 0**: Reject or redesign

### 💡 Scoring Examples

| Feature | Axiom Scores | Net | Decision |
|---------|--------------|-----|----------|
| ✅ Entry-module Prelude | A7:+1, A8:+1, A1:+1 | **+3** | Accept |
| ✅ Auto-cap inference | A3:+1, A8:+1, A4:0 | **+2** | Accept |
| ✅ Structured error traces | A2:+1, A11:+1, A7:+1 | **+3** | Accept |
| ❌ Global mutable state | A1:−1, A4:−1, A6:−1 | **−3** | Reject (violates A1) |
| ❌ Implicit DB connections | A3:−1, A4:−1, A12:−1 | **−3** | Reject (violates A3, A4) |
| ⚠️ Optional type annotations | A7:0, A5:+1, A8:−1 | **0** | Needs justification |

### 🧠 Philosophical Grounding

AILANG treats computation as **navigation through a fixed semantic structure**, not creation of new possibilities.

Key implications:
- **Effects are permissions**, not actions
- **Traces are primary artifacts**, not debugging aids
- **Cost is physical reality**, not incidental overhead

📖 See [Philosophical Foundations](/docs/references/philosophical-foundations) for the full motivation.

### What AILANG Deliberately Excludes

These are design choices, not limitations:

- ❌ **LSP/IDE servers** — AIs use CLI/API, not text editors
- ❌ **Multiple syntaxes** — one canonical way per concept
- ❌ **Implicit behaviors** — all effects are explicit
- ❌ **Human concurrency patterns** — CSP channels → static task graphs
- ❌ **Familiar syntax** — if it adds entropy, reject it

### Reference Documents

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [VISION_BENCHMARKS.md](../../../benchmarks/VISION_BENCHMARKS.md) - Vision goals and success metrics

**2. Choose Document Name**

**Naming conventions:**
- Use lowercase with hyphens: `feature-name.md`
- For milestone features: `m-XXX-feature-name.md` (e.g., `m-dx2-better-errors.md`)
- Be specific and descriptive
- Avoid generic names like `improvements.md`

**3. Run Create Script**

```bash
# If version is known (most cases)
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh feature-name v0_4_0

# If version not decided yet
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh feature-name
```

**4. Customize the Template**

The script creates a comprehensive template. Fill in:

**Header section:**
- Feature name (replace `[Feature Name]`)
- Status: Leave as "Planned"
- Target: Version number (e.g., v0.4.0)
- Priority: P0 (High), P1 (Medium), or P2 (Low)
- Estimated: Time estimate (e.g., "3 days", "1 week")
- Dependencies: List prerequisite features or "None"

**Problem Statement:**
- Describe current pain points
- Include metrics if available (e.g., "takes 7.5 hours")
- Explain who is affected and how

**Goals:**
- Primary goal: One-sentence main objective
- Success metrics: 3-5 measurable outcomes

**Solution Design:**
- Overview: High-level approach
- Architecture: Technical design
- Implementation plan: Break into phases with tasks
- Files to modify: List new/changed files with LOC estimates

**Examples:**
- Show before/after code or workflows
- Make examples concrete and runnable

**Success Criteria:**
- Checkboxes for acceptance tests
- Include "All tests passing" and "Documentation updated"

**Timeline:**
- Week-by-week breakdown
- Realistic estimates (2x your initial guess!)

**5. Review and Commit**

```bash
git add design_docs/planned/feature-name.md
git commit -m "Add design doc for feature-name"
```

### Moving to Implemented

**When to move:**
- Feature is complete and shipped
- Tests are passing
- Documentation is updated
- Version is tagged/released

**1. Run Move Script**

```bash
.claude/skills/design-doc-creator/scripts/move_to_implemented.sh feature-name v0_3_14
```

**2. Add Implementation Report**

The script provides a template. Add:

**What Was Built:**
- Summary of actual implementation
- Any deviations from plan

**Code Locations:**
- New files created (with LOC)
- Modified files (with +/- LOC)

**Test Coverage:**
- Number of tests
- Coverage percentage
- Test file locations

**Metrics:**
- Before/after comparison table
- Show improvements achieved

**Known Limitations:**
- What's not yet implemented
- Edge cases not handled
- Performance limitations

**3. Update design_docs/README.md**

Add entry under appropriate version:

```markdown
### v0.3.14 - Feature Name (October 2024)
- Brief description of what shipped
- Key improvements
- [CHANGELOG](../CHANGELOG.md#v0314)
```

**4. Commit Changes**

```bash
git add design_docs/implemented/v0_3_14/feature-name.md design_docs/README.md
git commit -m "Move feature-name design doc to implemented (v0.3.14)"
git rm design_docs/planned/feature-name.md
git commit -m "Remove feature-name from planned (moved to implemented)"
```

## Design Doc Structure

See [resources/design_doc_structure.md](resources/design_doc_structure.md) for:
- Complete template breakdown
- Section-by-section guide
- Best practices for each section
- Common mistakes to avoid

## Best Practices

### 1. Be Specific

**Good:**
```markdown
**Primary Goal:** Reduce builtin development time from 7.5h to 2.5h (-67%)
```

**Bad:**
```markdown
**Primary Goal:** Make development easier
```

### 2. Include Metrics

**Good:**
```markdown
**Current State:**
- Development time: 7.5 hours per builtin
- Files to edit: 4
- Type construction: 35 lines
```

**Bad:**
```markdown
**Current State:**
- Development takes a long time
```

### 3. Break Into Phases

**Good:**
```markdown
**Phase 1: Core Registry** (~4 hours)
- [ ] Create BuiltinSpec struct
- [ ] Implement validation
- [ ] Add unit tests

**Phase 2: Type Builder** (~3 hours)
- [ ] Create DSL methods
- [ ] Add fluent API
- [ ] Test with existing builtins
```

**Bad:**
```markdown
**Implementation:**
- Build everything
```

### 4. Link to Examples

**Good:**
```markdown
See existing M-DX1 implementation:
- design_docs/implemented/v0_3_10/M-DX1_developer_experience.md
- internal/builtins/spec.go
```

**Bad:**
```markdown
Similar to other features
```

### 5. Realistic Estimates

**Rule of thumb:**
- 2x your initial estimate (things always take longer)
- Add buffer for testing and documentation
- Include time for unexpected issues

**Good:**
```markdown
**Estimated**: 3 days (6h implementation + 4h testing + 2h docs + buffer)
```

**Bad:**
```markdown
**Estimated**: Should be quick, maybe 2 hours
```

## Document Locations

```
design_docs/
├── planned/              # Future features
│   ├── feature.md        # Unversioned (version TBD)
│   └── v0_4_0/           # Targeted for v0.4.0
│       └── feature.md
└── implemented/          # Completed features
    ├── v0_3_10/          # Shipped in v0.3.10
    │   └── feature.md
    └── v0_3_14/          # Shipped in v0.3.14
        └── feature.md
```

**Version folder naming:**
- Use underscores: `v0_3_14` not `v0.3.14`
- Match CHANGELOG.md tags exactly
- Create folder when first doc needs it

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow overview)
2. **Execute as needed**: Scripts create/move docs
3. **Load on demand**: `resources/design_doc_structure.md` (detailed guide)

## Coordinator Integration (v0.6.2+)

The design-doc-creator skill integrates with the AILANG Coordinator for automated workflows.

### Autonomous Workflow

When configured in `~/.ailang/config.yaml`, the design-doc-creator agent can:
1. Automatically receive GitHub issues via `github_sync`
2. Create design docs from issue descriptions
3. Hand off to sprint-planner on completion

```yaml
coordinator:
  agents:
    - id: design-doc-creator
      inbox: design-doc-creator
      workspace: /path/to/ailang
      capabilities: [research, docs]
      trigger_on_complete: [sprint-planner]
      auto_approve_handoffs: false
      session_continuity: true

  github_sync:
    enabled: true
    interval_secs: 300
    target_inbox: design-doc-creator
```

### Sending Tasks to design-doc-creator

```bash
# Via CLI
ailang messages send design-doc-creator "Create design doc for semantic caching" \
  --title "Feature: Semantic Caching" --from "user"

# The coordinator will:
# 1. Pick up the message
# 2. Create a git worktree
# 3. Execute the design-doc-creator workflow
# 4. Create approval request for human review
# 5. On approval, trigger sprint-planner with handoff message
```

### Handoff Message to sprint-planner

On completion, design-doc-creator sends:
```json
{
  "type": "design_doc_ready",
  "correlation_id": "task-123",
  "design_doc_path": "design_docs/planned/v0_6_3/m-semantic-caching.md",
  "session_id": "claude-session-abc",
  "discovery": "Key findings from GitHub issue analysis"
}
```

### Human-in-the-Loop

With `auto_approve_handoffs: false`:
1. Design doc is created in worktree
2. Approval request shows git diff in dashboard
3. Human reviews design doc quality
4. Approve → Merges to main, triggers sprint-planner
5. Reject → Worktree preserved for manual fixes

## Notes

- All design docs should follow the template structure
- Update CHANGELOG.md when features ship (separate from design doc)
- Link design docs from README.md under version history
- Keep design docs focused - split large features into multiple docs
- Use M-XXX naming for milestone/major features
