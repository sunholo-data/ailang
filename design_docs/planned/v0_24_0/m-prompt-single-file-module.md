# M-PROMPT-SINGLE-FILE-MODULE: Teach Single-File Module Convention for Benchmarks

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Medium)
**Estimated**: 0.5 day (prompt update + eval verification)
**Dependencies**: None

## Problem Statement

For the `multi_module_imports` benchmark, models write multiple `module benchmark/X`
declarations in a single `.ail` file. AILANG requires exactly one module declaration per file.
The eval harness provides exactly one solution file. Models produce multiple module declarations
because they are accustomed to multi-file project layouts from Python, JavaScript, and Go, and they
try to simulate that structure inside the single allowed file.

**Current State:**
- `multi_module_imports` benchmark fails in 4/4 models that attempt it with 100% compile_error.
- Error: `PAR_NO_PREFIX_PARSE: unexpected token in expression: module` on every second `module`
  declaration.
- No teaching prompt explains the one-module-per-file rule or the correct workaround.
- Models write code like:
  ```ailang
  module benchmark/math_utils
  func add(x: int, y: int) -> int = x + y

  module benchmark/string_utils     -- ❌ second module declaration = parse error
  func concat(a: string, b: string) -> string = a ++ b
  ```

**Impact:**
- **Affected benchmarks**: `multi_module_imports`, and any benchmark requiring multiple logical
  "modules" expressed in one file.
- **Severity**: Hard compile failure (blocker).
- **Frequency**: Affects all models equally — it's a universal assumption about multi-file projects,
  not a capability gap.

## Goals

**Primary Goal:** Eliminate multi-module-declaration compile errors by adding a prominent callout to
the teaching prompt explaining the single-file constraint and the correct workaround.

**Success Metrics:**
- Zero `PAR_NO_PREFIX_PARSE: unexpected token ... module` errors in the next eval rotation.
- `multi_module_imports` benchmark shows ≥50% improvement in compile success rate.
- Models generate a single `module benchmark/solution` declaration with all types and functions
  co-located in one module.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Callout placement in prompt (early vs. "Common Mistakes" table) | Affects model attention; early placement has higher recall | agent | implementation | low |
| Whether to show namespace-prefix convention as workaround | Without a workaround, models may guess incorrectly | agent | implementation | low |
| Whether to rename benchmark module from `benchmark/solution` hint | A consistent module name hint reduces errors further | human | design | low |

### Design Freeze

No high-change-cost decisions — all are low. No design freeze items required.

## Solution Design

### Overview

Add a prominent single-sentence rule to the AILANG teaching prompt:

> Benchmark solutions are always a SINGLE FILE with ONE module declaration
> (`module benchmark/solution`). Never write multiple module declarations.
> To simulate multiple logical modules, define types and functions in the same module,
> using naming prefixes (e.g. `math_add`, `string_concat`) to namespace them.

Also add an example showing the before/after to make the rule concrete.

### Implementation Plan

**Phase 1: Locate prompt injection points** (~30 minutes)
- [ ] Find all teaching prompt / eval system prompt files
- [ ] Confirm which are injected before `multi_module_imports` benchmark runs

**Phase 2: Write and insert callout** (~1 hour)
- [ ] Draft the rule text and before/after example
- [ ] Insert into "Common Mistakes" table AND as a standalone callout block
- [ ] Verify the example code compiles with `ailang check`

**Phase 3: Eval verification** (~2 hours)
- [ ] Re-run `multi_module_imports` benchmark
- [ ] Confirm zero `PAR_NO_PREFIX_PARSE: ... module` errors

### Files to Modify/Create

**Modified files:**
- Teaching/eval prompt file(s) (paths TBD after Phase 1 survey) (+15 LOC)

## Examples

### What models currently generate (compile error)

```ailang
-- ❌ WRONG: Two module declarations in one file — parse error on line 5
module benchmark/math_utils

pure func add(x: int, y: int) -> int = x + y

module benchmark/string_utils          -- PAR_NO_PREFIX_PARSE here

pure func concat(a: string, b: string) -> string = a ++ b
```

### Correct single-module layout

```ailang
-- ✅ CORRECT: One module declaration, all code in the same module
module benchmark/solution

-- "math_utils" functions live here, namespaced by name prefix
pure func math_add(x: int, y: int) -> int = x + y
pure func math_mul(x: int, y: int) -> int = x * y

-- "string_utils" functions live here, namespaced by name prefix
pure func string_concat(a: string, b: string) -> string = a ++ b
pure func string_upper(s: string) -> string = to_upper(s)
```

### Callout text to add to prompt

```
WARNING: ONE module per file. Benchmark solutions are a SINGLE FILE.

The module declaration must be: `module benchmark/solution`
Never write a second `module` keyword. If you need "multiple modules",
define all types and functions in the same module and use naming
prefixes to group them (e.g. `math_add`, `string_concat`).
```

## Success Criteria

- [ ] Teaching prompt contains the one-module-per-file rule with a before/after example
- [ ] The "correct" example compiles with `ailang check`
- [ ] Zero `PAR_NO_PREFIX_PARSE: ... module` errors in next eval run on `multi_module_imports`
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Compile verification:**
- `ailang check` on the "correct" example in the prompt — must pass

**Eval verification:**
- Re-run `multi_module_imports` after prompt update
- Inspect model outputs: confirm single module declaration in generated files

**Manual testing:**
- Read the updated prompt section and verify clarity

## Deferred Decisions

- Exact prefix convention recommendation — agent may choose
- Whether to add this rule to a "hard rules" box vs. the "Common Mistakes" table — agent may choose
- Whether all benchmarks should standardise on `benchmark/solution` as the module name — human at review

## Non-Goals

- Adding multi-file support to the eval harness — out of scope for this doc; separate feature
- Changing the AILANG one-module-per-file rule — this is correct language design, not a bug

## Timeline

- Day 1 (half day): Locate prompts, draft rule + example, verify, run eval

**Total: ~0.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Model ignores callout and still writes two modules | Medium | Place rule in both the "Common Mistakes" table AND a standalone WARNING block |
| Naming-prefix workaround confuses models further | Low | Keep the example minimal; use obvious prefixes |
| Different eval benchmarks use a different module name convention | Low | Check all benchmark module names during Phase 1 and standardise if needed |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Single-module files are easier to typecheck and verify |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Directly eliminates 100% compile-error rate on this benchmark |
| A8: Minimal Syntax | +1 | Teaches models to use existing syntax correctly; no new syntax needed |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | 0 | No change to composition rules |
| A11: Structured Failure | +1 | Replaces opaque parse error with a pre-empted, explained constraint |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Teaching single-file convention benefits machine generation directly

## Related Documents

- `design_docs/planned/v0_24_0/m-import-alias.md` — companion P1 prompt-gap fix (import aliases)
- `design_docs/planned/v0_23_0/m-eval-slim-prompt-self-discovery.md` — related prompt structure work

## References

- **Failing benchmark**: `benchmarks/multi_module_imports/`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Consider making the eval harness explicitly reject second `module` declarations with a
  user-facing error that says "benchmark solutions must have exactly one module declaration"
  rather than a generic parse error — separate improvement ticket

---

**Document created**: 2026-06-03
**Last updated**: 2026-06-03
