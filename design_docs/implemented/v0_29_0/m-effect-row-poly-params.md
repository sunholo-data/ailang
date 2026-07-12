# M-EFFECT-ROW-POLY-PARAMS — Effect-row polymorphism on higher-order arguments

**Status**: Planned — P1 (blocks demos repo upgrade to v0.21.0)
**Target**: v0.22.x
**Priority**: P1 — `sunholo/demos` is pinned to v0.20.1 and cannot upgrade until this lands; affects every stdlib higher-order API that takes a handler/callback (`std/stream`, `std/smoke`, future `std/ai/streaming`, `std/process`).
**Estimated**: ~2-3 days (type-checker effect-row substitution at call sites; stdlib rollout; demo migration mechanical after)
**Dependencies**: None directly. Closely related to [M-LAMBDA-OPEN-RECORD-PATTERN](m-lambda-open-record-pattern.md) — same "generalize doesn't quantify the row variable" class of bug, but for effect rows instead of record rows.
**Source**: Surfaced 2026-05-21 in [`sunholo/demos` message msg_20260521_165838_fbdf60b9](.) — 10 demos (streaming/*, linkedin, co-presenter) broke on v0.20.1→v0.21.0 bump. Pin reverted.

## Reproducer

```ailang
module repro

import std/stream (onEvent, transmit, StreamConn, StreamEvent, Message)
import std/io (println)

export func handleEvent(event: StreamEvent, conn: StreamConn) -> bool ! {IO, Stream} {
  match event {
    Message(msg) => {
      let _ = println("got: ${msg}");
      let _ = transmit(conn, "ack");
      true
    },
    _ => true
  }
}

export func main(conn: StreamConn) -> unit ! {Stream, IO} {
  -- Lambda capturing conn, delegating to a named helper with declared effects
  onEvent(conn, \event. handleEvent(event, conn))
}
```

Result on v0.21.0:

```
Error: type unification failed at [function application]: failed to unify parameter 1:
  failed to unify effect rows: incompatible closed rows:
  r1 has extra labels [], r2 has extra labels [IO Stream]
```

The lambda `\event. handleEvent(event, conn)` infers a closed effect row `{IO, Stream}` from its body and cannot unify with `onEvent`'s declared `(StreamEvent) -> bool` (empty closed row).

**Workaround that works today (no capture):**

```ailang
-- Top-level named function — generalized scheme, row instantiates at call site
export func handler(event: StreamEvent) -> bool ! {IO} { ... }
onEvent(conn, handler)   -- works
selectEvents([src], handler)   -- works
```

So named-function references behave correctly (their scheme is row-polymorphic at instantiation time). Lambdas don't.

## Root cause

Three sites are implicated. First two are confirmed; third is hypothesized.

### 1. Stdlib advertises a closed empty row on handler params

[`std/stream.ail`](../../std/stream.ail) declares handlers as `(StreamEvent) -> bool` with no effect annotation, which the elaborator reads as the empty closed row `{}`. The intended signature is row-polymorphic:

```ailang
export func onEvent(conn: StreamConn, handler: (StreamEvent) -> bool ! {e}) -> unit ! {Stream | e}
```

[`std/smoke.ail`](../../std/smoke.ail) already ships this row-polymorphic shape (`dispatchAllTools`, `dispatchTool`) — but is never exercised from a user call site, so the downstream effect-checker gap (#3) was unflagged.

### 2. Parser accepts `{e}` as row var, iface emits row var, effect checker silently freezes it

[`internal/parser/parser_effect.go:64-78`](../../internal/parser/parser_effect.go#L64-L78) recognizes lowercase identifiers in effect rows as row variables (`isRowVar` check). The emitted iface for `onEvent` (verified) has `tail: rowvar ε_annot1` on the handler parameter's effect row — so the data shape reaches the type checker correctly.

But: applying the fix to `std/stream.ail` (locally, then rebuilding the binary) and re-running the reproducer still produces the same error. The type checker reports `r1 has extra labels []` — empty closed row — at the call site, ignoring the row-var tail in the iface.

### 3. (Hypothesized) Effect-row scheme instantiation collapses the row variable to empty

Most likely failure mode, by analogy with M-LAMBDA-OPEN-RECORD-PATTERN: at the call site, when `onEvent`'s scheme is instantiated, the row variable `ε_annot1` is either:

- (a) Not quantified by `generalize` for effect rows the way it is for type variables — so it appears at instantiation but already bound to `{}` from definition-site solving, or
- (b) Quantified, instantiated fresh, but the freshening doesn't propagate into the function-typed parameter's effect row — so the lambda argument unifies against the original (now-closed) effect row.

Minimal user-space repro (no stdlib involvement) — file [/tmp/test_rowpoly.ail](.) during investigation:

```ailang
export func apply_with_io[e](f: (int) -> int ! {e}) -> int ! {IO | e} {
  let x = f(42);
  let _ = _io_println("done");
  x
}

export func io_double(n: int) -> int ! {IO} { ... }

export func main() -> unit ! {IO} {
  let r = apply_with_io(io_double);   -- error: "Missing effects: e"
  _io_println("got ${r}")
}
```

Error: `Missing effects: e` — the effect checker treats the row variable `e` as a literal effect name rather than instantiating it to `{IO}` from `io_double`'s type. Confirms #3.

## Goals

1. The reproducer above (lambda capturing `conn`, delegating to a named helper with declared effects) passes `ailang check` and `ailang run` on v0.22.x.
2. `apply_with_io[e](f) -> ! {IO | e}` minimal repro type-checks and effect-checks when called with a function whose row is `{IO}`.
3. `std/stream` handler signatures updated to row-polymorphic form (`(StreamEvent) -> bool ! {e}` with outer `! {Stream | e}`) for `onEvent`, `runEventLoop`, `selectEvents`, `withStream`, `withSSE`.
4. `std/smoke` row-poly handlers stay correct (regression coverage — they shipped working-by-accident).
5. Named-function-reference callsites (current workaround: `multiHandler`) continue to work — the fix MUST NOT regress that.

## Proposed approach

1. **Confirm root cause via test.** Add a failing test in `internal/types/effect_row_poly_test.go` for the minimal repro (`apply_with_io`). Verify the failure is at scheme-instantiation time (not at parsing or unification).
2. **Trace `generalize` for effect rows.** Inspect [`internal/types/types_v2.go`](../../internal/types/types_v2.go) around `Scheme.RowVars` (line ~442-525) — verify effect-row variables are collected during generalization and freshly instantiated. Likely fix: extend the row-quantification path to walk function-typed parameter positions for nested effect rows.
3. **Test scheme instantiation produces fresh row vars in nested function positions.** Add an internal test that schemes the type `(int) -> int ! {e} -> int ! {IO | e}` and instantiates it twice, verifying the two `e` row variables are independent fresh tails.
4. **Update `std/stream.ail` signatures** for `onEvent`, `runEventLoop`, `selectEvents`, `withStream`, `withSSE` — row-polymorphic handler param + outer `! {Stream | e}`.
5. **Migration of demos**: `sunholo/demos` can drop its v0.20.1 pin and use lambda-capturing handlers. Coordinate via reply to `msg_20260521_165838_fbdf60b9`.
6. **Documentation**: add a "Closure-capturing event handlers" recipe to [`docs/docs/guides/streaming.md`](../../docs/docs/guides/streaming.md); add the row-poly recipe to the v0.16.0+ teaching prompt's effect-system section; add v0.20→v0.22 migration guide.

## Acceptance criteria

- [ ] Reproducer at top of this doc passes `ailang check` and `ailang run` (with `Stream`, `IO` caps).
- [ ] Minimal `apply_with_io[e]` repro passes.
- [ ] `std/stream` handler signatures row-polymorphised; existing examples (`stream_multi_source.ail`, `stream_process_source.ail`, `stream_websocket.ail`, `stream_sse.ail`) still pass `make verify-examples`.
- [ ] Existing examples that use raw builtins (`_io_println` etc.) inside handlers MIGRATED to wrapped functions (`println`, `transmit`) — the v0.21.0 workaround pattern is removed from canonical examples.
- [ ] `sunholo/demos` 10 broken demos type-check on the new build with their lambda-capturing handlers intact. Coordinate verification with demo author.
- [ ] No regression on `examples/runnable/stream_multi_source.ail` (named-function-reference path).
- [ ] Regression test `internal/types/effect_row_poly_test.go` covers:
  - Lambda + closure capture + named-helper delegation → PASS
  - Lambda inline with effectful body → PASS
  - Lambda passed where empty-row expected (signature without `! {e}`) → FAIL with clear message
  - Named-function ref to row-polymorphic param → PASS (existing path)
  - Nested higher-order: `(a -> b ! {e1}) -> (b -> c ! {e2}) -> a -> c ! {e1 | e2}` → PASS
- [ ] Teaching prompt updated with row-poly example for handler-style APIs.

## Risks

- **HM-with-rows generalization is subtle.** Over-generalizing effect rows could silently admit handlers that fire effects the caller didn't declare — the opposite failure mode. Mitigation: the third regression test (FAIL case) is the guard.
- **Stdlib iface churn.** Once handler signatures change, every consumer's iface hash shifts. Cached compile artifacts in `~/.ailang/cache/registry` for downstream packages will need invalidation. Mitigation: bump stdlib version in the binary; the version-mismatch warning already surfaces in `ailang check` output.
- **Demo migration timing.** `sunholo/demos` is paused on v0.20.1. We need to ship this BEFORE any future v0.22 feature that the demos repo would want, so the upgrade is one step not two.
- **Interaction with effect refinement (`@limit`, `@min`)**: handler row instantiation must preserve refinement annotations. Out of scope for v1 of this fix but a follow-up regression risk.

## Out of scope

- Row subtyping (different feature; AILANG uses row polymorphism not subtyping).
- Generalizing record rows on lambda parameters — that's [M-LAMBDA-OPEN-RECORD-PATTERN](m-lambda-open-record-pattern.md), a sibling.
- Capability-budget propagation through row variables (`{IO @limit=N | e}` semantics) — separate design.
- Effect-row polymorphism on type aliases or ADT constructors.

## Related

- [M-LAMBDA-OPEN-RECORD-PATTERN](m-lambda-open-record-pattern.md) — same generalize-doesn't-quantify-row-vars class of bug, for record rows. Likely the fix touches the same `generalize` code path; consider bundling.
- [M-TAINT-TYPES](../implemented/v0_16_0/) — sink check / `Declassify` (referenced in the demo author's question #3; unrelated mechanism but same class of "rule on by default since vX, surfacing on upgrade").
- `std/smoke.ail` — already ships the row-polymorphic pattern (`dispatchAllTools`, `dispatchTool`); proof that the pattern was always *intended* but never verified end-to-end.
- M-EXT-AUTHOR-DX (v0.20.1) — `_smoke.ail` scaffold uses `dispatchAllTools`. If row-poly works after this fix, extension authors get cleaner smoke files automatically.

## Interim guidance (already in reply to `sunholo/demos`)

While this is in flight, the workarounds are:

1. Stay on v0.20.1 if relying on lambda-capturing handlers (the demos repo's current state).
2. For no-capture: pass a top-level named function (works today).
3. For static-conn capture: bind conn at module top-level (loses per-request scoping).
4. For dynamic-conn capture: raw builtins (`_stream_send`, `_io_println`) inside the lambda — runtime cap budgets still apply at builtin-fire time; loses static effect-row enforcement.

These workarounds are documented in the reply at `msg_20260521_173515_2e7e0119`. They should be removed from canonical examples once this milestone lands.
