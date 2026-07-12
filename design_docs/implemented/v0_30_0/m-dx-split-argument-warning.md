# M-DX-SPLIT-ARG: Compile-Time Warning for Reversed `split` Arguments

**Status**: Implemented (2026-07-12, mission iteration 17, PR #356 → `8339b6421`)
**Target**: v0.9.5 (shipped in the v0.30.0 line)
**Priority**: P2 (DX — silent wrong results, observed twice in AI-generated code)
**Estimated**: 1 day
**Dependencies**: None
**Milestone ID**: M-DX-SPLIT-ARG
**Created**: 2026-03-25
**Source**: DocParse agent message `eafa7e06` (split argument order DX trap)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change — compile-time warning only |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Catches likely bugs at compile time |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | Directly addresses AI code synthesis confusion — machines get better feedback |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +1 | Turns silent wrong result into visible warning |
| A12: System Boundary | 0 | No change |

**Net Score: +4** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Warning only, no semantic change
- [x] A7 (Machines First): Directly improves AI-generated code quality
- [x] A8 (Minimal Syntax): No new syntax

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"same-typed argument swap" class of bugs**. Other functions vulnerable:

| Function | Signature | Swap risk |
|----------|-----------|-----------|
| `split(s, delimiter)` | `(string, string)` | **HIGH** — observed twice |
| `find(hay, needle)` | `(string, string)` | Medium |
| `contains(hay, needle)` | `(string, string)` | Medium |
| `startsWith(s, prefix)` | `(string, string)` | Low (name is clear) |
| `compare(a, b)` | `(string, string)` | Low (symmetric) |

**Root cause**: When both arguments have the same type, the type system provides no guardrail. The warning system should be extensible to other functions with known swap traps.

---

## Problem Statement

### The Bug

```ailang
-- WRONG: silently returns ["api/keys/user123"] (1-element list)
let parts = split("/", name)

-- CORRECT: returns ["api", "keys", "user123"]
let parts = split(name, "/")
```

Both compile and run without error. The wrong version returns a list containing the original string unsplit, because it tries to split the literal `"/"` by the delimiter `name` (e.g., `"api/keys/user123"`), which doesn't appear in `"/"`.

### Why AI Models Get This Wrong

This has been observed **twice** in AI-generated AILANG code. Root causes:

1. **`join` and `split` have opposite argument orders:**
   ```ailang
   split(string, delimiter)  -- data-first
   join(delimiter, list)     -- config-first (!!!)
   ```
   These are inverse operations. An AI that learns `join(sep, xs)` naturally writes `split(sep, s)`.

2. **Higher-order functions are func-first:**
   ```ailang
   map(f, xs)       -- function first
   filter(p, xs)    -- predicate first
   foldl(f, acc, xs) -- function first
   ```
   `split` feels like the delimiter is the "operation" argument, like `f` in `map(f, xs)`.

3. **Python/JS use method syntax:** `"a,b,c".split(",")` — the delimiter is the only argument. No ordering ambiguity exists. AILANG's free-function style forces a choice that other languages avoid.

4. **Both arguments are `string`** — the type system cannot distinguish them.

### Why Not Swap the Order?

Swapping to `split(delimiter, string)` would:
- Break all existing code (`examples/runnable/string_split.ail`, tests, user packages)
- Break consistency with Go's `strings.Split(s, sep)` which AILANG deliberately mirrors
- Create a migration burden disproportionate to the benefit
- Still leave the `join`/`split` inconsistency (just in the other direction)

### The Real Fix: Better Signals

The correct approach is to **keep the current order** but provide compile-time warnings and better documentation so the mistake is caught immediately.

---

## Goals

1. Emit a compile-time warning when `split` is called with a short string literal as first arg and a variable/expression as second arg
2. Update prompts and docs to explicitly call out the `join`/`split` ordering difference
3. Design the warning system to be extensible for other same-typed-arg functions

---

## Proposed Solution

### Phase 1: Compile-Time Heuristic Warning

Add a post-elaboration warning pass that detects likely reversed `split` arguments.

**Heuristic**: Warn when:
- First arg is a string literal of 1-3 characters (typical delimiters: `","`, `"/"`, `" "`, `"\n"`, `"::"`)
- Second arg is NOT a string literal (it's a variable, function call, or other expression)

```
warning: split(s, delimiter) takes the string first, delimiter second.
  --> api_keys.ail:42:15
   |
42 |   let parts = split("/", name)
   |               ^^^^^^^^^^^^^^^^
   = hint: did you mean split(name, "/")?
   = note: split("/", name) splits "/" by "name", which likely returns ["/"]
```

**Implementation location**: `internal/pipeline/` — add a new `warn_split_args.go` pass that runs after elaboration, before codegen. Hook into the existing diagnostics/warning infrastructure.

**Key files**:
- `internal/pipeline/warn_split_args.go` (new)
- `internal/pipeline/pipeline.go` (register the pass)
- `internal/builtins/string_ops.go` (reference for split registration)

### Phase 2: Documentation & Prompt Updates

Update the teaching prompt to explicitly flag the ordering:

```
- `split(s, delimiter)` - Split string by delimiter (**string first, delimiter second**)
  - WARNING: `split` and `join` have DIFFERENT argument orders:
    - `split(s, delim)` — data first
    - `join(delim, xs)` — separator first
  - Common mistake: `split(",", s)` — this splits "," by s, returning [","]
```

**Key files**:
- `prompts/v0.9.0.md` (or current version)
- `docs/docs/guides/` (if a string ops guide exists)

### Phase 3 (Future): `join` Argument Order Consistency

Consider a separate design doc for migrating `join(delimiter, list)` to `join(list, delimiter)` for consistency with `split`. This is a breaking change requiring:
- Deprecation warning for old order
- Version gate (e.g., v1.0)
- Package migration tooling

**Not in scope for this doc** — just flagged as the long-term systemic fix.

---

## Test Plan

1. **Warning triggers correctly:**
   - `split(",", name)` -> warning
   - `split("/", path)` -> warning
   - `split("\n", text)` -> warning
   - `split("::", qualified)` -> warning (2-char delimiter)

2. **No false positives:**
   - `split(name, ",")` -> no warning (correct usage)
   - `split("hello world", " ")` -> no warning (long first arg is clearly data)
   - `split(a, b)` -> no warning (both variables, can't tell)
   - `split(",", ",")` -> no warning (both literals, edge case)

3. **Warning doesn't block compilation** — code still runs, just with a diagnostic

4. **Prompt verification** — run eval benchmarks that involve `split` to confirm AI models get the order right with updated prompts

---

## Migration

No migration needed — this is a new warning, not a breaking change. Existing correct code is unaffected. Existing incorrect code will now show a warning.
