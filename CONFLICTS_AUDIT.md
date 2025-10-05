# AILANG Documentation Conflicts Audit

**Date**: 2025-10-05
**Current Actual Version**: v0.3.0-alpha2 (Recursion + Blocks + Records COMPLETE)

---

## Critical Conflicts Found

### 1. README.md - Mixed Version References

**Issues:**
- Line 10: Claims "v0.3.0 (Clock & Net Effects)" but we're actually v0.3.0-alpha2
- Line 48-63: Says "AILANG v0.2.0 now executes" - OUTDATED, we're past v0.2.0
- Line 143: "The REPL is the **most complete** part of AILANG v0.1.0" - OUTDATED
- Line 170: "## What Works in v0.1.0" - ENTIRE SECTION OUTDATED
- Line 204: "Modules parse and type-check correctly but cannot execute until v0.2.0" - **FALSE, modules DO execute since v0.2.0!**
- Line 220-242: "## What's Coming in v0.2.0" - OUTDATED, v0.2.0 is complete
- Line 421: "Not yet. v0.1.0 is an MVP" - OUTDATED FAQ
- Line 424: "Module files type-check but cannot execute until v0.2.0" - **FALSE**
- Line 437: "Module files type-check but cannot execute (runtime coming in v0.2.0)" - **FALSE**

**Impact**: **CRITICAL** - Users and AI assistants think modules don't execute!

### 2. docs/LIMITATIONS.md - Completely Outdated

**Issues:**
- Line 1: "# AILANG v0.1.0 Known Limitations" - OUTDATED VERSION
- Line 3: "Last Updated: October 2, 2025" - Same date as v0.3.0 work!
- Line 11: "What Doesn't: Module execution" - **FALSE, modules DO execute**
- Line 12: "When Fixed: v0.2.0" - v0.2.0 is COMPLETE
- Line 15-24: Entire section "Critical Limitation: Module Execution" - **FALSE**
- Line 48: "`ailang run example.ail` ❌ Fails" - **FALSE, works fine**
- Line 86: "Target: v0.2.0 (estimated 1-2 weeks after v0.1.0)" - **OUTDATED**

**Impact**: **CRITICAL** - This is the #1 referenced doc for limitations, and it's completely wrong!

### 3. docs/guides/ai-prompt-guide.md - Outdated Prompt

**Issues:**
- Line 9: "## The AILANG Prompt (v0.2.0-rc1)" - OUTDATED
- Line 16: "## Current Version: v0.2.0-rc1 (Module Execution + Effects)" - OUTDATED
- Missing: Recursion (v0.3.0-alpha2)
- Missing: Block expressions (v0.3.0-alpha2)
- Missing: Records (v0.3.0-alpha2)
- Line 30: "NO pattern guards (if in match arms - parsed but not evaluated)" - Still true
- Line 32: "Let expressions limited to 3 nesting levels" - Still true

**Impact**: **HIGH** - AI assistants reading this won't know about recursion/blocks/records

### 4. prompts/v0.3.0.md - Most Up-to-Date ✅

**Status**: **CORRECT** - This is the most accurate document!

**Contains:**
- ✅ v0.3.0-alpha2 features (recursion, blocks, records)
- ✅ Accurate limitations (no record update syntax, no guards)
- ✅ Correct examples
- ✅ Working syntax examples

**Recommendation**: Make this the **single source of truth**

---

## Recommended Action Plan

### Priority 1: Critical Fixes (Block AI Confusion)

1. **Update README.md**:
   - Change "v0.1.0" to "v0.3.0-alpha2" throughout
   - Remove section "What's Coming in v0.2.0" (it's done!)
   - Update FAQ to say modules DO execute
   - Update "For AI agents" footer to reference `prompts/v0.3.0.md`

2. **Update docs/LIMITATIONS.md**:
   - Change title to "AILANG v0.3.0-alpha2 Known Limitations"
   - REMOVE "Critical Limitation: Module Execution" (fixed in v0.2.0!)
   - Update to current limitations:
     - Pattern guards parsed but not evaluated
     - Let nesting limited to 3 levels
     - Record update syntax NOT implemented
     - No error propagation `?` operator

3. **Update docs/guides/ai-prompt-guide.md**:
   - Change to v0.3.0-alpha2
   - Add recursion, blocks, records to "What Works"
   - OR: Replace entire file with "See prompts/v0.3.0.md for canonical prompt"

### Priority 2: Centralization Strategy

**Decision**: Make `prompts/v0.3.0.md` the **canonical source of truth**

**Changes needed:**

1. **CLAUDE.md**: ✅ Already references `prompts/v0.3.0.md` (we just added this!)

2. **README.md**: Add at top:
   ```markdown
   **📖 For AI Code Generation**: The canonical AILANG syntax reference for AI assistants is [prompts/v0.3.0.md](prompts/v0.3.0.md)
   ```

3. **docs/guides/ai-prompt-guide.md**: Replace with redirect:
   ```markdown
   # AI Prompt Guide

   **Canonical Source**: [prompts/v0.3.0.md](../../prompts/v0.3.0.md)

   The AI teaching prompt is maintained in the `prompts/` directory and validated through eval benchmarks.
   ```

4. **All other docs**: Reference `prompts/v0.3.0.md` for "what works"

### Priority 3: Establish Update Process

**Rule**: When a language feature is implemented, update in this order:

1. **prompts/vX.Y.md** - Canonical truth (used by evals)
2. **CLAUDE.md** - Update "Current Status" section
3. **README.md** - Update "What Works" summary
4. **CHANGELOG.md** - Document the change

**DON'T update separately**:
- ❌ docs/LIMITATIONS.md (derive from prompts)
- ❌ docs/guides/ai-prompt-guide.md (redirect to prompts)
- ❌ Scattered version references (centralize to prompts)

---

## Verification Checklist

After fixes, verify:

- [ ] All version references say v0.3.0-alpha2 or later
- [ ] No document says "modules can't execute"
- [ ] All documents agree on what's implemented
- [ ] `prompts/v0.3.0.md` referenced as canonical source
- [ ] No conflicting feature lists

---

## Files to Update

### High Priority (CRITICAL)
1. `README.md` - Lines 10, 48, 63, 143, 170-242, 421, 424, 437
2. `docs/LIMITATIONS.md` - Entire file outdated
3. `docs/guides/ai-prompt-guide.md` - Version and features

### Medium Priority
4. `examples/README.md` - Check version references
5. `design_docs/README.md` - Check if it references outdated info

### Low Priority (After above)
6. Search all `*.md` for "v0.1.0" and "v0.2.0" references
7. Update any design docs marked "planned" that are actually complete

---

## Diff: What Actually Works (v0.3.0-alpha2)

### ✅ COMPLETE
- Module execution ✅ (v0.2.0)
- Effect system (IO, FS) ✅ (v0.2.0)
- Recursion ✅ (v0.3.0-alpha2)
- Block expressions ✅ (v0.3.0-alpha2)
- Records (literals + field access) ✅ (v0.3.0-alpha2)
- Pattern matching ✅ (v0.1.0)
- Type classes ✅ (v0.1.0)
- ADTs ✅ (v0.1.0)
- REPL ✅ (v0.1.0)

### ❌ NOT IMPLEMENTED
- Record update syntax `{r | field: val}`
- Pattern guards `pattern if condition =>`
- Error propagation `?`
- Let nesting beyond 3 levels
- Typed quasiquotes
- CSP concurrency
- Session types

---

## Next Steps

1. ✅ CLAUDE.md - Already references prompts/v0.3.0.md
2. ⏳ Fix README.md critical false claims
3. ⏳ Fix LIMITATIONS.md (completely rewrite)
4. ⏳ Update ai-prompt-guide.md (redirect to prompts)
5. ⏳ Establish "prompts as truth" policy

**Owner**: Should be done by next commit to avoid confusing users/AI
