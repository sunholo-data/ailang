# M-IFC-CROSS-MODULE — IFC Label Refinements Surviving Module Boundaries

**Status**: Planned
**Target**: v0.39.0
**Priority**: P0 (High — security-sensitive semantics change to `internal/types`)
**Estimated**: 4 days
**Dependencies**: M-SECRET-EFFECT / M-TAINT-TYPES label lattice (shipped); `TLabelled` gob cache codec (0bc1a7b57); closure-laundering fix (1af9f5f30)

> **Semantics change to `internal/types`.** `internal/types/`, per AGENTS.md, is a
> security- and semantics-sensitive package. This doc changes what a *correct* program
> is (previously-clean programs across module boundaries will now be rejected), and
> touches the iface schema. It requires the full gate: design approval → sprint plan →
> execute.

## Problem Statement

GitHub issue #1134 (daneel, measured on v0.36.0): IFC label refinements are enforced
**within** a module but **lost across a module boundary**. If a library exports

```
gate : (input: string{not email}) -> bool
```

then a caller module passing an `email`-labelled value to `gate` compiles cleanly.
This is the concrete blocker for IFC Phase 2 (#1132 part 1: labels enforced as *type
errors*, not only via Z3 contracts).

**Root cause** (confirmed in code): `CheckModuleIFC` (`internal/types/ifc_check.go`)
is a deliberately self-contained surface-AST pass over `unit.Surface.Funcs` — local
declarations only. When the callee is imported, `labelOfCall` falls into the
"Unknown or imported callee: transparent" branch, which:

1. **Skips Check A entirely** — the imported callee's `{not ℓ}` parameter refinements
   are never checked against argument labels at the call site.
2. **Under-approximates the result label** — an imported function returning
   `<secret>` (e.g. one wrapping `secret()`) yields `⊥` in the caller's taint env,
   so downstream caller-side sinks are also unprotected.

Two aggravating facts make this worse than a single missing check:

- The **iface builder already preserves labels** — `applyLabelsFromAST`
  (`internal/iface/builder.go`) re-wraps exported param/return positions in
  `TLabelled`, and that survives cache gob (0bc1a7b57). But refinements are
  *sink-side* metadata and are explicitly **not stored on the type** ("Refinements
  (`{not LABEL}`) are sink-side only and are not stored on the type itself"). So the
  caller's type env gets the labelled scheme, then **unification strips TLabelled**
  (`unification_core.go` M7: "labels are projection metadata … never participate in
  unification") — the label never becomes a caller-visible constraint.
- `CheckSinkRefinement` / `CheckDeclassify` (`sink_check.go`) still have **zero
  callers**; the only live enforcement is the per-module surface walk.

**Impact:** any taint-sink contract a library declares is unenforceable by its
importers — silently. This is exactly the "silent widening" failure mode CLAUDE.md
Section 2 forbids, in a security control.

**Out of scope** (tracked elsewhere): label-aware tracing (#1132 second half —
`m-trace-label-aware.md`), Z3-contract enforcement, and full label-polymorphic
inference (M-TAINT-TYPES Phase 3). Single-dep repro only; both directions (sink
refinement *and* labelled return) are in scope here.

## Proposed Solution

Keep IFC as a **surface-AST analysis**; add a serializable **IFC summary** to the
module interface so a caller's `CheckModuleIFC` walk can treat imported callees
exactly like local ones. Three coordinated pieces:

### 1. Iface IFC summary (internal/iface, schema v2)

Extend `IfaceItem` with an optional IFC projection built at export time:

```
ifc: {
  params:    [{ label: "secret" | "", notLabel: "email" | "" }, ...]  // per param, source label + sink refinement
  return:    "secret" | ""        // declared return label (⊥ = none)
  intrinsic: "secret" | ""       // body's effective label, params at ⊥ (see §2)
  declassify: bool               // ! {Declassify} in the effect row
}
```

This is metadata, **not a type change**: `TLabelled` wrapping stays as-is; refinements
get a home outside the type system (they cannot ride `TLabelled` because a refinement
is not a label — it forbids one). Serialize alongside the existing scheme in the
iface JSON + gob, and **include it in the iface digest**. Schema string bumps to
`ailang.iface/v2`.

**Fail-loud semantics (no silent widening):**

- The summary is emitted for **every** export (possibly empty). An iface that is
  readable but lacks IFC metadata is by construction an older schema; existing
  digest validation (`internal/pipeline/cache_artifacts.go`,
  M-CACHE-MODULE-ID-ENCODING) forces recompilation rather than silently accepting
  a v1 iface. No fallback to "assume clean".
- Within a v2 summary, an export with no declared return label is **transparent** —
  its caller-side result label is `join(intrinsic, argLabels)` — the same rule as
  local transparent functions (APP-PURE), now including the callee's own intrinsic
  taint. This errs in the safe (over-approximating) direction, consistent with the
  1af9f5f30 closure-laundering fix.

### 2. Dep-side intrinsic label at export build

A caller cannot see a library function's body, so "transparent" cannot mean "⊥
unless declared" — that re-creates the under-approximation. Instead, the **dep's own
compile** computes the body's effective label with parameters seeded at ⊥ (the
existing `effectiveBodyLabel` fixpoint in `ifc_check.go`) and stores it as
`intrinsic`. A library wrapper that calls `secret()` with no return annotation then
exports `intrinsic: "secret"`, and every importer sees the taint without any
annotation burden. `declassify: true` exports instead promise
`return` authoritatively, breaking the taint chain as locally.

### 3. Caller-side enforcement (internal/types + pipeline + repl)

Generalize `CheckModuleIFC` to take the import summary:

```go
func CheckModuleIFC(file *ast.File, imports map[string]ImportedIFCSig) []*TypeCheckError
```

`labelOfCall` routes an imported callee to the same logic as a local one: Check A
runs against `params[i].notLabel`, and `calleeResultLabel` is
`return` if declared or declassifying, else `join(intrinsic, argLabels)`. Callers:

- `internal/pipeline/pipeline_module_compile.go` — build the map from
  `imports.ExternalTypes`'s owning ifaces (the summary travels with `IfaceItem`,
  and `pipeline_module_imports.go` already walks those).
- `internal/repl/module_registry_load.go` (#1114 gate) — same wiring, so REPL and
  CLI agree.

Unknown/other callees (local closures, unimported builtins) keep the existing
transparent join — that path is already conservative post-1af9f5f30 and untouched.

### Enforcement stays a surface-AST analysis — not elaboration

Explicitly considered and rejected for this phase: moving label flow into
elaboration/CoreTI. Reasons: (a) `TLabelled` is deliberately confined to
`internal/types` + `internal/iface`; the backend has no case for it, so the current
design has zero regression surface in the HM core or codegen; (b) unification strips
labels by design (M7 — labels are projection metadata), so an elaboration-based
enforcement would require unification to participate in labels, a semantics change
to type inference itself; (c) the summary mechanism gives the caller exactly the
cross-module facts the intramodule walk lacks. Revisit only if a future phase needs
label-polymorphic inference (Phase 3). This doc records the decision so it is not
re-litigated per-sprint.

### Cache interaction

- `TLabelled` already survives the compile cache (gob codec, 0bc1a7b57) — no work.
- The IFC summary is new serialized state on `IfaceItem`; it must round-trip both
  gob (cache) and JSON (published/compact iface), and its bits must be part of the
  iface **digest** so a dep whose labels changed invalidates dependents' cached
  ifaces — a stale cache must never hand a caller a pre-v2 (label-silent) summary.
- `CheckModuleIFC` runs at the **caller's** compile time every compile (it is not
  cached), so its cost stays where it is today: linear in the caller's surface AST
  plus one summary lookup per imported call.

### Error message shape (names the violating edge)

Errors must name the caller → callee edge, not just the call site, so an agent can
act without reopening the library:

```
information-flow violation: value labelled <secret> reaches parameter "input" of
imported function strings.gate, which forbids it (string{not secret})
  edge: app.ail:12:5 → strings.gate(input) — declassify the value through a
  function declaring ! {Declassify}, or do not pass it across this edge
```

Same shape class as the existing `newSinkError` (`SinkRefinementError` /
`DeclassifyRequiredError` kinds in `internal/types/errors.go`), extended with the
imported symbol name. Declass errors for imported callees point at the **library's**
declared return label so the fix is unambiguous about which side must change.

## Implementation Plan

1. **Iface summary + schema v2** — `IfaceItem.IFC`, JSON/gob serialization, digest
   inclusion, schema bump. (`internal/iface`)
2. **Dep-side intrinsic computation** — export `effectiveBodyLabel` from the
   per-module checker at interface-build time. (`internal/types`, `internal/iface`)
3. **Caller-side wiring** — `CheckModuleIFC(file, imports)`; pipeline + REPL gates.
   (`.pipeline`, `internal/repl`)
4. **Tests** — the #1134 repro both directions (sink refinement across a boundary;
   labelled return across a boundary); intrinsic propagation without annotation;
   declassify across a boundary; closure-arg into imported sink (1af9f5f30
   interplay: closure body label is the arg label Check A sees); iface cache
   round-trip of the summary; old-schema iface rejected loudly.

## Risks & Trade-offs

- **Breaking change:** programs that passed labelled values into library sinks now
  fail. That is the point (#1132), but it needs a changelog note and a prompt/docs
  sync (label semantics are taught by `ailang prompt`).
- **Over-approximation friction:** transparent imported callees join `intrinsic`,
  which can block a caller whose data never actually crosses the label — same
  conservative direction as 1af9f5f30; acceptable for a security control and
  resolvable via explicit return labels or `! {Declassify}` on the library side.
- **Schema churn:** v2 bump invalidates cached ifaces once; digest validation makes
  that a recompile, not a silent accept.

## Success Criteria

- The #1134 repro (gate function moved to another module) fails to compile with an
  error naming the violating edge.
- The mirror case (library returns `<secret>`, caller's `{not secret}` sink) fails
  in the caller module.
- No `TLabelled` reaches CoreTI/codegen; unification behavior unchanged.
- A dep whose IFC summary changes invalidates dependents' cached ifaces.
- REPL (#1114 gate) and CLI produce identical verdicts.

## Related Documents

- `design_docs/planned/v0_36_0/m-trace-label-aware.md` — the other half of #1132
  (label-aware tracing); explicitly out of scope here.
- `design_docs/implemented/v0_26_0/m-secret-effect-remote-approval.md` — M5: the
  intramodule `CheckModuleIFC` design this extends (and its Phase 2 deferral note).
- `design_docs/implemented/v0_14_3/m-smt-cross-module-types.md` — cross-module
  contract precedent (interfaces carrying per-export check obligations).
- `design_docs/planned/v0_36_0/m-cache-module-id-encoding.md` — digest/cache
  invalidation machinery the schema bump rides on.