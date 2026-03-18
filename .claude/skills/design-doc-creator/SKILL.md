---
name: design-doc-creator
description: Create AILANG design documents in the correct format and location. Use when user asks to create a design doc, plan a feature, or document a design. Handles both planned/ and implemented/ docs with proper structure.
---

# Design Doc Creator

Create well-structured design documents for AILANG features following the project's conventions.

## Quick Start

**Most common usage:**
```bash
# User says: "Create a design doc for better error messages"
# This skill will:
# 1. AUTO-SEARCH for related design docs (Ollama neural embeddings)
# 2. Show matches from implemented/ and planned/ directories
# 3. Auto-populate "Related Documents" section in template
# 4. Proceed automatically (no confirmation needed)
# 5. Create design_docs/planned/better-error-messages.md
# 6. Fill template with proper structure
```

**Automatic Related Doc Search (v0.6.3+):**

When you run the create script, it automatically:
1. Converts doc name to search query (e.g., `m-dx2-better-errors` → `"better errors"`)
2. Runs **both** SimHash (instant) and Neural (better quality) searches
3. Shows results from both methods so you can compare
4. Merges unique results (neural preferred) for the template
5. Auto-populates the "Related Documents" section

```bash
$ .claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-semantic-caching

🔍 Searching for related design docs...

Implemented docs matching "semantic caching":
  [SimHash - instant]
  1. design_docs/implemented/v0_4_0/monomorphization.md (1.00)
  2. design_docs/implemented/v0_3_18/M-DX4-SPRINT-PLAN.md (0.95)
  [Neural - semantic matching]
  1. design_docs/implemented/v0_6_0/m-doc-sem-lazy-embeddings.md (0.45)
  2. design_docs/implemented/v0_6_0/semantic-caching-complete.md (0.42)

Planned docs matching "semantic caching":
  [SimHash - instant]
  1. design_docs/planned/v0_7_0/M-REPL1_persistent_bindings.md (1.00)
  [Neural - semantic matching]
  1. design_docs/planned/v0_7_0/semantic-caching-future.md (0.50)

ℹ Related docs found above - review them after creation if needed.
```

**Why dual search?** SimHash is instant but keyword-dependent. Neural finds semantically related docs even when keywords don't match. You see both so you can judge which results are more relevant.

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
- **Searches for related docs** using `ailang docs search --neural` (Ollama embeddings)
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

**Check existing work:** The create script auto-searches for related docs using both SimHash and neural embeddings. Review the results before filling in the template.

**Warning signs of fragmented design** (expand scope if you see these):
- Multiple maps tracking similar things
- Switch statements with growing case lists
- Bug fixes that add `|| specialCase` conditions

**Axiom Compliance:** Every feature must score against all 12 axioms. Hard violations on A1/A3/A4/A7 = automatic rejection, net score must be ≥ +2. See `resources/design_doc_structure.md` for full scoring matrix and examples.

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

**4. Read Related Docs Found by Search**

**IMPORTANT:** Before filling in the template, read the top 2-3 related docs found by the search. This ensures your design:
- Builds on existing patterns and conventions
- Avoids duplicating work already done
- References relevant prior art
- Identifies potential conflicts early

```bash
# The script outputs related docs like:
# Implemented docs matching "feature name":
#   [Neural - semantic matching]
#   1. design_docs/implemented/v0_6_0/similar-feature.md (0.45)
#   2. design_docs/implemented/v0_5_0/related-work.md (0.38)

# READ these docs before proceeding:
# - Look at their structure and patterns
# - Note any design decisions that apply
# - Check for overlap with your feature
# - Reference them in your "Related Documents" section
```

**What to look for in related docs:**
- Architecture patterns used
- Testing strategies employed
- Edge cases already handled
- Implementation trade-offs documented
- Success/failure metrics to compare against

**5. Customize the Template**

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

**6. Review and Commit**

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

## Notes

- All design docs should follow the template structure
- Update CHANGELOG.md when features ship (separate from design doc)
- Link design docs from README.md under version history
- Keep design docs focused - split large features into multiple docs
- Use M-XXX naming for milestone/major features
