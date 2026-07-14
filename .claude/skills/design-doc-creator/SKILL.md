---
name: design-doc-creator
description: Create AILANG design documents in the correct format and location. Use when user asks to create a design doc, plan a feature, or document a design. Handles both planned/ and implemented/ docs with proper structure.
---

# Design Doc Creator

Create well-structured design documents for AILANG features following the project's conventions.

## Current State

- **Current version**: !'cat std/VERSION'
- **Planned docs**: !'ls design_docs/planned/ 2>/dev/null | head -10'
- **Recent implemented docs**: !'ls design_docs/implemented/ 2>/dev/null | tail -3'
- **Active changelog**: !'ls changelogs/ | grep current 2>/dev/null'

> **Use the data above first.** Only re-run these commands manually if the injected context is empty or you need to refresh after making changes.

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
- [ ] **VERIFIED every "AILANG does/doesn't support X" claim with `ailang check` (HARD GATE — see below)**
- [ ] Is this a one-off or part of a pattern?
- [ ] Search codebase for similar code paths
- [ ] Check if other types/cases have the same gap
- [ ] Look at git history - has this area been patched repeatedly?
- [ ] If citing eval data: segmented by recent date + confirmed the cited construct is the ACTUAL cause
- [ ] Design fix to cover ALL cases, not just the reported one

**⚠️ CRITICAL: Verify Every Language Claim Before Asserting It (HARD GATE)**

**Any statement of the form "AILANG does / does not support X" MUST be verified with a live
`ailang check` before it goes in the doc.** This is not optional. A design doc that
mischaracterizes the language sends the implementer to build something that already exists,
reject something that's fine, or fix a non-problem.

**The check takes 10 seconds:**
```bash
tmp=$(mktemp -d)
cat > $tmp/claim.ail <<'EOF'
module test/claim
-- the exact construct you're claiming is (un)supported
import std/list as L
export func main() -> () ! {} = ()
EOF
ailang check $tmp/claim.ail   # COMPILES = supported; PAR_/type error = not
rm -rf $tmp
```

**Required in the doc:** every "supported"/"unsupported" claim must carry either
(a) an `ailang check` result/transcript, or (b) a citation to a reference page or implemented
design doc. Unverified assertions are treated as unproven and the doc cannot proceed to sprint.

**Also verify the FREQUENCY claim** if the doc cites eval data ("fails in N models", "X% of
failures"): segment by recent date (not all-time aggregate — old baselines mix in pre-fix
runs), and confirm the cited construct is the ACTUAL cause of the failure, not a co-occurring
line. A regex that flags `as <word>` in an import is NOT proof the import is the cause.

**Case study (2026-06-03, this exact failure):** Two hand-written eval-gap docs asserted
language limitations without running `ailang check`:
- `m-type-constraints` claimed "AILANG has no typeclasses, use explicit comparators." FALSE —
  AILANG infers Ord/Num via dictionary passing; `mymax[a](x:a,y:a)=if x>y then x else y` runs on
  int AND string. The doc would have built an unnecessary feature.
- `m-import-alias` claimed "AILANG has no import aliases, implement them." FALSE — `import M as L`
  and `import M (f as g)` both compile. The cited "6% of failures" was a detection-heuristic
  false positive: 0 of 16 flagged failures actually failed on an alias. Doc was REJECTED.

Both errors were a 10-second `ailang check` away from being caught. The neural related-doc
search (below) is passive context; THIS gate is the active check. Do not skip it by
hand-writing content over the scaffold — fill the scaffold, and verify as you fill.

**The same "verify against the code, not your assumption" rule covers three more claim classes
the Verification Log routinely marks "Confirmed" without actually checking:**

1. **A newly-proposed diagnostic/error code MUST be verified unallocated.** Error codes
   (`MODxxx`, `PARxxx`, `TCxxx`, `EFF_*`, `Exxx`) are a shared namespace. Before writing
   `Success: exit ≠ 0 + MODnnn: …` into a doc, grep the codebase:
   ```bash
   grep -rn "MOD011" internal/ cmd/   # empty = free; any hit = TAKEN, pick the next free code
   ```
   Put the grep result in the Verification Log. The intended code is a *claim* like any other.

2. **A Conflict-Surface routing/mechanism claim MUST be verified by reading the code path, not
   inferred from observable output.** "Construct X routes to `runSingle`, never reaches
   `validateModulePath`" is a claim about control flow — confirm it by reading the dispatch, not
   by observing that `X` produces the right answer. A correct *output* can hide a wrong
   *mechanism*, and the wrong mechanism yields a wrong guard/fix that the implementer must then
   override mid-sprint.

3. **Every cited regression fixture MUST exist, and every claim about an existing test's
   BEHAVIOR must come from reading the test body.** The Conflict Surface's "Programs that MUST
   still work" list and any "test X asserts Y" statement are claims like the rest:
   ```bash
   ls examples/lambdas_higher_order.ail        # cite only files that exist
   sed -n '/func TestName/,/^}/p' path_test.go # read what it actually asserts
   ```
   Case study (2026-07-13, m-arity-style-diagnostic): the doc cited
   `examples/lambdas_higher_order.ail` + `examples/no_loops_fold.ail` as regression fixtures —
   neither exists — and claimed `TestCurriedMismatchStillFails` "asserts on the old bare string"
   requiring "the ONE intentional test-text change"; the test asserts only `err != nil`, so the
   planned edit was fictional. Both were one `ls`/one read away at doc time; the sprint-planner
   had to correct the premises mid-loop.

**Case study (2026-07-12, m-module-less-run-fail-loud):** the doc's Verification Log marked two
things "Confirmed" that a code-check refuted. (a) It proposed error code `MOD011` — already the
live module-path-collision diagnostic since v0.10.9; the reality-check reassigned it to `MOD014`
(one `grep` would have caught it). (b) It marked "bare-expr eval unaffected — routes to
`runSingle`, never reaches `validateModulePath`" Confirmed, on the strength of `1+1 → 2` still
working; the *mechanism* was false (a bare-expr **file** does reach `validateModulePath`, and the
parser mirrors the expression into `Statements`+`Decls`), so the doc's proposed 3-way
`Funcs||Statements||Decls` guard would have broken `ailang run 1+1`. The executor had to override
it to `Funcs`-only mid-sprint. Both were a one-command code-check away at doc time.

**⚠️ CRITICAL: Conflict Surface Analysis (REQUIRED for parser/typechecker/codegen changes)**

**If the design touches `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`:**

The design doc MUST include a **Conflict Surface** section (see [resources/design_doc_structure.md](resources/design_doc_structure.md)) enumerating:
1. What syntactic/semantic positions does this change extend?
2. What OTHER valid constructs already live in those positions?
3. How does the parser/typechecker disambiguate?
4. Which existing programs MUST still work post-change? (3-5 fixtures)
5. What deliberately changes (intentional incompatibilities)?

**Why this is required**: most language regressions come from "I didn't realize X also uses this position." The author is the only one who can credibly enumerate the conflict surface. Reviewers can sanity-check but can't generate the list.

**Concrete case study**: M-TAINT-TYPES (v0.14.3) added `T{not LABEL}` refinement syntax without enumerating that `func ... -> bool { not f(x) }` already used the same `{ not <ident>` prefix in function bodies. The 2-token-lookahead disambiguation was insufficient. The motoko_agent fork (still on v0.13.0) hit ~14 mis-parses when migrating to v0.15.x. Caught months after release. See [M-PARSER-REFINEMENT-LOOKAHEAD changelog entry](../../../changelogs/v0.10-v0.17-bytecode-vm.md) for the fix.

**The honest answer is almost never "no conflicts"**: if the section says that for a parser/typechecker change, the author hasn't looked hard enough.

**Check existing work:** The create script auto-searches for related docs using both SimHash and neural embeddings. Review the results before filling in the template.

**Warning signs of fragmented design** (expand scope if you see these):
- Multiple maps tracking similar things
- Switch statements with growing case lists
- Bug fixes that add `|| specialCase` conditions

**Axiom Compliance:** Every feature must score against all 12 axioms. Hard violations on A1/A3/A4/A7 = automatic rejection, net score must be ≥ +2. See `resources/design_doc_structure.md` for full scoring matrix and examples.

**Quorum review (OPTIONAL, off-Anthropic — M-MISSION-FLEET-AB Phase B):** Before a
design doc proceeds to planning, you MAY run an independent N-reviewer quorum that scores it
against these same hard gates (premise verification, Conflict Surface, axiom compliance) using
non-Anthropic frontier models — a cheap (cents/doc), off-quota second opinion that catches bad
premises before an Opus sprint is spent on them. This is an **optional documented step, not a
contract change**: the design-doc-creator flow is unchanged if you skip it.

```bash
# One reviewer (reject-by-default; exits non-zero if it can't produce a verdict):
ailang design-review design_docs/planned/vX_Y/my-doc.md --reviewer gpt5-6-sol --json

# Full quorum (parallel reviewers + your own IN-SESSION verdict as the controller):
ailang design-quorum design_docs/planned/vX_Y/my-doc.md \
  --reviewers gpt5-6-sol,gemini-3-1-pro \
  --controller-verdict pass --controller-note "<your in-session judgement>" \
  --mission-log design_docs/v1-mission-log.md
```

Reviewers are **reject-by-default** (each must state a `strongest_objection`). Synthesis: any
present reviewer or the controller rejects → **blocked** (exit 3); the objection goes back to you,
the author, before planning. A reviewer that is unreachable / over-budget / mis-authed is recorded
by NAME with its reason and the quorum degrades to N−1 — never a silent pass. Each run writes a
machine artifact under `.ailang/state/mission-quorum/` (seeds the Phase E assignment table) and,
with `--mission-log`, a human markdown block. Gemini reviewers use **Vertex ADC** (not
`GEMINI_API_KEY`). See `ailang design-quorum --help`.

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

**4. Duplicate / Coverage Gate (MANDATORY — do this before creating the file)**

**Before creating any doc, read the top matches from the search and apply this gate:**

| Similarity score | Action |
|---|---|
| ≥ 0.75 (neural) on a planned doc | **REJECT** — the topic is already queued. Reply with the path and explain what's already covered. Do NOT create a new doc. |
| ≥ 0.65 (neural) on an implemented doc | **REJECT** — already shipped. Reply with the path + version it shipped in. Do NOT create a new doc. |
| 0.45–0.65 on any doc | **Warn** — read the doc, confirm your topic is genuinely distinct before proceeding. Note the distinction explicitly in the "Related Documents" section. |
| < 0.45 | Proceed normally. |

**Rejection reply format (use this when rejecting):**
```
⛔ Duplicate / Already Covered

This topic is already addressed in:
  [doc title](path/to/doc.md) — [planned v0.X.Y / implemented in vX.Y.Z]

Key overlap: <one sentence on what the existing doc covers that would duplicate this request>

If your request is genuinely distinct, please clarify how it differs from the above.
```

**Read related docs found by search** before proceeding (non-rejected cases):
- Look at their structure and patterns
- Note any design decisions that apply
- Check for overlap with your feature
- Reference them in your "Related Documents" section

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
