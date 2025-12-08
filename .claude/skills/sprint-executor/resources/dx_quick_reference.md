# DX Quick Reference Card

Use this during sprint execution to quickly identify and act on DX improvements.

## Decision Matrix

| Time to Implement | Action |
|-------------------|--------|
| < 15 min | 🟢 **DO NOW** - Add to current milestone |
| 15-30 min | 🟡 **DEFER** - Add to TODO, do at milestone end if time allows |
| > 30 min | 🔴 **DESIGN DOC** - Create in `design_docs/planned/vX_Y/m-dx*.md` |

## Common Patterns

### 1. Boilerplate → Helper Function (5-15 min)
**Signal**: Copy/paste same code 3+ times
**Fix**: Extract to `*_helpers.go`
**Example**: `AssertNoErrors(t, p)`

### 2. Hard to Debug → Debug Flag (5-10 min)
**Signal**: Adding `fmt.Printf()` for visibility
**Fix**: Add `DEBUG_<SUBSYSTEM>=1` check
**Example**: `DEBUG_PARSER=1 ailang run test.ail`

### 3. Manual Workflow → Make Target (3-5 min)
**Signal**: Multi-step command repeated
**Fix**: Add `make <target>` with docs
**Example**: `make update-golden`

### 4. Confusing API → Documentation (10-20 min)
**Signal**: Looking up same API 3+ times
**Fix**: Add godoc, CLAUDE.md section, or `make doc`
**Example**: M-TESTING API patterns in CLAUDE.md

### 5. Bad Error → Actionable Message (5-15 min)
**Signal**: Error doesn't explain how to fix
**Fix**: Add context + suggestion + docs link
**Example**: "expected RPAREN at line 42, got COMMA. See: ..."

### 6. Verbose Test → Test Utility (10-20 min)
**Signal**: Test setup takes 10+ lines
**Fix**: Add helpers in `testctx/` or `*_helpers.go`
**Example**: `MakeString()`, `AssertLiteralInt()`

## ROI Formula

```
Time saved per use × Expected uses = Total savings
If Total savings > Implementation time → DO IT
```

**Examples**:
- Helper (10 min impl): 2 min × 20 uses = 40 min → ROI = 4x ✅
- Debug flag (8 min impl): 15 min × 5 uses = 75 min → ROI = 9x ✅
- Documentation (20 min impl): 5 min × 30 uses = 150 min → ROI = 7.5x ✅

## Reflection Questions (Ask After Each Milestone)

- [ ] What was painful during this milestone?
- [ ] What took longer than expected due to tooling gaps?
- [ ] What did I lookup multiple times?
- [ ] What errors/bugs could better tooling prevent?
- [ ] Did I write any boilerplate that could be a helper?
- [ ] Did I add debug print statements that should be a flag?
- [ ] Would this error message confuse others?

## Documentation Template (Per Milestone)

```markdown
## DX Improvements (Milestone X)

✅ **Applied**: <What> (<Time>)
   - Impact: <Benefit>
   - Files: <Locations>

📝 **Deferred**: Created M-DX* design doc (<Estimated time>)
   - Rationale: <Why needed>
   - Expected ROI: <Time saved>
   - File: design_docs/planned/vX_Y/m-dx*-name.md

💡 **Considered**: <What> (added to backlog / rejected because...)
```

## Design Doc Template (For Large DX Improvements)

When creating DX design docs for improvements >30 min:

```markdown
# M-DX*: [Title]

## Problem
What pain point does this solve? How did we discover it?

## Current Workaround
What do we do now? Why is it painful?

## Proposed Solution
What should we build? High-level approach.

## Estimated Effort
- Implementation: X hours
- Testing: Y hours
- Documentation: Z hours
- **Total**: X+Y+Z hours

## Expected ROI
- Time saved per use: A minutes
- Expected uses per sprint: B times
- Expected uses per year: C times
- **Total savings**: A × C minutes/year
- **ROI**: (A × C) / ((X+Y+Z) × 60) = D.D×

## Acceptance Criteria
- [ ] What defines "done"?
- [ ] How do we measure success?

## Related
- Sprint: M-XXX (where pain point discovered)
- Related DX improvements: M-DX*, M-DX*
```

## Debug Effect for Runtime Tracing (v0.4.10+)

When implementing features that need runtime tracing, use the Debug effect:

```ailang
import std/debug as Debug

func update(e: Entity) -> Entity ! {Debug} {
    Debug.check(e.health >= 0, "health must be non-negative");
    Debug.log("updating entity " ++ show(e.id));
    -- ... entity logic
}
```

**Key benefits:**
- **Write-only**: AILANG code writes, host collects - no branching on debug state
- **Ghost effect**: Erased in `--release` mode (zero runtime cost in production)
- **Structured output**: `DebugContext.Collect()` returns JSON-serializable data
- **Assertions don't throw**: `Debug.check(false, msg)` records failure but continues

**Run with Debug capability:**
```bash
ailang run --caps IO,Debug --entry main game.ail
```

**Note:** Use `Debug.check` not `Debug.assert` (`assert` is a reserved keyword).

## Quick Tips

- **Before implementation**: Ask "How can I make this easier for next time?"
- **During debugging**: "Could a debug flag save me 10+ min?"
- **For AILANG code**: "Should I add Debug.log/check here?"
- **During testing**: "Will I write this test pattern again?"
- **After error**: "Would this message confuse me tomorrow?"
- **End of milestone**: "What took the most time? Could tooling help?"

## Red Flags (DON'T DO THESE)

- ❌ Optimizing for hypothetical future use (no proven need)
- ❌ Over-engineering simple solutions (YAGNI)
- ❌ DX improvements with negative ROI (<1x)
- ❌ Tooling that only helps humans (AILANG is AI-first)
- ❌ Breaking changes to save small amounts of time

## Green Flags (DO THESE)

- ✅ Fixing pain you just experienced (proven need)
- ✅ Simple solutions with clear ROI (>2x)
- ✅ Tooling that helps AI understanding (determinism, clarity)
- ✅ Error messages with actionable fixes
- ✅ Documentation where you got confused
