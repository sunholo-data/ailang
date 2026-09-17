# Effect checker resolves let-bound lambda to top-level function of same name

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: "effect checker", "shadowing", "lexical" across design_docs/; inspected `internal/pipeline/validate_effects.go`
- **Estimate**: omitted (design-doc route)

**Why (mechanism, verified in code):** `collectRequiredEffects` in `internal/pipeline/validate_effects.go` (the `*core.App` case, ~line 338) resolves a call to a `core.Var` callee through the **name-keyed `declaredEffects` map** (populated only from top-level function signatures) *before* consulting `CoreTypeInfo`, which holds the type checker's lexically-correct binding for the let-bound lambda. So shadowing a top-level function with a pure let-bound lambda falsely propagates the top-level's effect row (`{IO, Clock}` in the repro). The comment at the fallback ("Use declared effects instead of CoreTypeInfo to avoid contamination") shows the priority order is deliberate, not an accident — which is exactly why this isn't a two-line flip: swapping to CoreTypeInfo-first risks regressing whatever "contamination" that ordering was added to fix, and the alternative ("share the type checker's resolver", per the backlog's direct-fix call) means threading scoped let-bindings through `collectRequiredEffects`/`validateDecl`, which is more than DIRECT_FIX_MAX_LINES across the effect-checker call paths. There are at least three defensible remedies (CoreTypeInfo-first, scoped-binding map, resolver sharing) — a reviewer could reasonably disagree among them, and the change alters the effect gate's resolution contract.

**Prior coverage note:** this exact report already has a backlog row (`design_docs/planned/ailang-core-backlog.md`, 2026-09-15) labelled `direct-fix` on the grounds that "sharing the type checker's resolver" involves no semantics decision. I disagree with that call: the `declaredEffects`-first ordering is a documented contamination workaround, so unifying resolvers is a semantics-bearing change to the effect gate, and the reporter's repro (Daneel calendar host, false "Missing effects: Clock, IO") shows real blast radius. Recommend running it through design-doc-creator rather than a hot patch.

**Reporter's workaround** (rename the local) is valid meanwhile and worth quoting in the doc.
