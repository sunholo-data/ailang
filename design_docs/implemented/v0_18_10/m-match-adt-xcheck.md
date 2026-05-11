# M-MATCH-ADT-XCHECK: typechecker rejects match arms with constructors from a foreign ADT

**Status**: Implemented (v0.18.10)
**Target**: v0.18.10 (or v0.19.0 — type-checker work is non-trivial)
**Priority**: P1 (real bug-class enabler — silent runtime panics that pass `ailang check`)
**Estimated**: ~2-3 days (~400-600 LOC including tests + the new error code + LSP wiring)
**Dependencies**: None (pure typechecker change in `internal/types/`)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-11
**Source**: motoko_agent integration testing 2026-05-11. Two crashes in arniwesth/motoko_agent#16 ([motoko_ext_compaction_ai 0.1.0 → 0.1.1 → 0.1.2](https://github.com/sunholo-data/ailang-packages/pull/16)) where the package's `register.ail` had:

```ailang
match getString(json, "model") {  -- returns Option[string] (Some/None)
  Err(_) => default_config().model,
  Ok(m) => m
}
```

The arms reference `Err`/`Ok` (constructors of `Result`), but the value's type is `Option[string]` (constructors `Some`/`None`). At runtime: "no pattern matched in match expression" panic when the field IS present — the typechecker never noticed the constructor-ADT mismatch.

---

## Problem statement

AILANG's match-arm typechecker currently allows any constructor pattern in any position regardless of whether that constructor belongs to the scrutinee's ADT. Specifically:

```ailang
let x: Option[int] = Some(42) in
match x {
  Err(e) => "err: ${e}",   -- Err is from Result, not Option — should be a type error
  Ok(n)  => "ok: ${n}",    -- same for Ok
  None   => "none"          -- this one IS from Option but is dead code (never reached)
}
```

This compiles cleanly today. At runtime, the `Err`/`Ok` arms can never match (the value is always `Some(42)` or `None`), and if `Some(42)` is the value, the match panics with "no pattern matched in match expression" since `None` is the only Option arm.

This is a **real bug class** that's bitten production code:
- motoko_ext_compaction_ai 0.1.1 (arniwesth/motoko_agent#16): match Result arms against Option value → crash on first turn
- Likely many similar latent bugs in extensions and user code, masked because the value happens to fall through to a working arm

**Why it slips through**: AILANG's `match` typechecker checks that:
1. Each arm's body has a compatible type
2. The constructor's argument arity matches its definition

But it does NOT check that the constructor BELONGS to the scrutinee's ADT. Each constructor is type-checked in isolation against its OWN ADT's definition, not cross-checked against the scrutinee's type.

## Goals

1. **Reject `match Option[T] { Err(_) => ... }` at compile time** with a clear error pointing at the offending constructor and naming both ADTs.
2. **Preserve current behavior for valid matches** — no regression on the 1000+ existing match expressions in stdlib + examples + tests.
3. **Don't reject open-row patterns** — match-on-record-row remains type-checked as today.
4. **Provide an "all variants of foreign ADT" warning** as a stretch goal: when ALL arms of a match come from a foreign ADT, emit a "did you mean to import the right constructors? Some/None vs Ok/Err is a common confusion" hint.

## Non-goals

- **Exhaustiveness analysis** for ADT matches — that's a separate sprint (M-MATCH-EXHAUSTIVENESS, not yet planned). This sprint is purely about rejecting impossible arms.
- **Open-record / row-polymorphic match analysis** — those use a different matching mechanism that already type-checks correctly.
- **Constructor name shadowing across modules** — when two ADTs define a constructor with the same name (rare but possible), we'll prefer the imported / qualified one based on the scope-resolution rule already in place.

## Solution sketch

The match-arm typechecker (`internal/types/match.go` or wherever pattern checking lives) currently does roughly:

```
for each arm in match expr:
    typecheck pattern against arm's bound vars
    typecheck arm body in extended scope
    unify all arm body types
```

We need to add a step BEFORE the per-arm typecheck:

```
let scrutinee_type = typecheck(scrutinee)
let scrutinee_adt = resolve_adt(scrutinee_type)  -- e.g. "Option", "Result", "list", ...
for each arm in match expr:
    let arm_constructor = head_of_pattern(pattern)
    if arm_constructor is a constructor name (not a wildcard / variable / literal):
        let arm_adt = lookup_adt_for_constructor(arm_constructor)
        if arm_adt != scrutinee_adt:
            error: TC_MATCH_FOREIGN_CONSTRUCTOR {
              constructor: arm_constructor,
              expected_adt: scrutinee_adt,
              actual_adt: arm_adt,
              position: pattern_position,
            }
    -- existing per-arm typecheck continues
```

Edge cases to handle:
- **Wildcard `_` pattern**: always allowed regardless of scrutinee type.
- **Variable binding `x`**: always allowed (binds the scrutinee value).
- **Literal pattern `42`, `"foo"`, `true`**: allowed iff scrutinee type is `int`/`string`/`bool` respectively.
- **Tuple pattern `(a, b)`**: requires scrutinee type to be tuple of matching arity.
- **List pattern `[]`, `x :: rest`**: requires scrutinee type to be `list[T]`.
- **Record pattern `{field: x, ...}`**: requires scrutinee type to be record with those fields (already type-checked).
- **Constructor-with-fields `Some(x)` against `Option[T]`**: typecheck `x` against `T` (existing behavior — keep).
- **Polymorphic constructor `Some` from a generic ADT**: works as today via the type-substitution machinery.
- **Foreign module constructors**: when the constructor is qualified (`std.option.Some`), look up the ADT in that module's exports.

## API change / error code

New error code: `TC_MATCH_FOREIGN_CONSTRUCTOR`

Sample error message:

```
type error in <file>: match arm constructor 'Err' belongs to ADT 'Result',
not 'Option' (the scrutinee's type) at <file>:<line>:<col>

The match expression scrutinizes a value of type Option[string], but the
arm constructor 'Err' is from the Result ADT. AILANG match arms must use
constructors of the scrutinee's ADT.

Did you mean to use 'None' instead? Option's constructors are: Some, None.
Result's constructors are: Ok, Err.

Common confusion: std/json.getString returns Option[T] (Some/None), NOT
Result[T, _] (Ok/Err). std/json.decode returns Result[Json, string].
See: https://ailang.sunholo.com/docs/reference/option-vs-result
```

## Acceptance

- [ ] New error code `TC_MATCH_FOREIGN_CONSTRUCTOR` defined + registered in error catalog
- [ ] Typechecker rejects `match Option[T] { Err(_) => ..., Ok(_) => ... }` with the new error
- [ ] Typechecker rejects `match Result[T, E] { Some(_) => ..., None => ... }` symmetrically
- [ ] Typechecker rejects `match list[T] { Some(_) => ..., None => ... }` (no list-vs-Option confusion either)
- [ ] All existing stdlib + examples + tests pass (no regression on valid matches)
- [ ] LSP integration: surface the new error inline in the editor with the constructor underlined
- [ ] Updated docs at `docs/reference/option-vs-result.md` (canonical "which is which")
- [ ] CHANGELOG entry under v0.18.10

## Why this matters for AI-author workflows

The motoko agent (and similar AI coding agents using AILANG packages) iteratively writes extension code, type-checks via `ailang check`, declares "done" when type-check passes. Today, the loop accepts buggy code that crashes at runtime — which then surfaces in the consumer (e.g. motoko's TUI on first turn), forcing a multi-version package iteration cycle (0.1.0 → 0.1.1 → 0.1.2 in this case).

With this fix, the AI's `ailang check` step REJECTS the buggy code immediately, the AI gets a precise error pointing at the wrong constructor, and the loop self-corrects in one iteration. This compounds across all extension packages, all user-written AILANG code, and all eval-harness benchmark code that pattern-matches.

The downstream runtime probe (motoko_agent's `make verify_extensions` + ailang-packages' `make verify-extensions`, both shipped during the bug investigation) is the belt-and-suspenders safety net for OTHER runtime panics (panicking `readFile`, recursive panics in compiled effect code, etc.) and remains valuable. But this typechecker fix is the upstream root cause closure for the specific Option/Result confusion class.

## Related work

- M-MATCH-EXHAUSTIVENESS (not yet planned): warn when a match doesn't cover all constructors of the scrutinee's ADT. This sprint's xcheck is necessary but not sufficient — even with foreign constructors rejected, you can still write `match Option[int] { Some(x) => x }` and panic on `None`. Exhaustiveness is the next-level fix.
- AILANG runtime error format (separate quality-of-life sprint): "no pattern matched in match expression" should include file:line. Today it doesn't, which is what made the motoko_agent bug so hard to triage.

## Refs

- Bug that motivated this: motoko_ext_compaction_ai 0.1.1 → 0.1.2 transition in [arniwesth/motoko_agent#16](https://github.com/arniwesth/motoko_agent/pull/16)
- Downstream safety nets shipped during the investigation:
  - [arniwesth/motoko_agent#16 commit 2f282c3](https://github.com/sunholo-voight-kampff/motoko_agent/commit/2f282c3) — motoko's `make verify_extensions`
  - [sunholo-data/ailang-packages#16](https://github.com/sunholo-data/ailang-packages/pull/16) — ailang-packages' `make verify-extensions` + per-package `_smoke.ail` pattern
