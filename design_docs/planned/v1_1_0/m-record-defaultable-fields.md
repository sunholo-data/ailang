# Type-Anchored Defaultable Record Fields

**Status**: Planned
**Target**: v1.1.0
**Priority**: P1 (Medium)
**Estimated**: 5 days
**Dependencies**: None
**Change class**: **Type-system change — security-sensitive** (`internal/types/`, `internal/ast/`, `internal/eval/` are treated as security- and semantics-sensitive per AGENTS.md; quorum review recommended before sprint planning)

**Source**: GitHub issue #905 (sunholo-data/ailang), triaged from the public-feedback inbox; reported by motoko_agent maintainers.

## Problem Statement

The motoko extension ABI (`pkg/sunholo/motoko_ext_abi/types.ExtensionHooks`, pinned at 2.2.0) is a **closed record**: `id`, `provided_tools`, and 8 function-typed hook fields (`on_describe_tools`, `on_build_system_prompt`, `on_budget_plan`, `on_pre_step`, `on_tool_policy`, `on_tool_handle`, `on_response_intercept`, `on_solver_candidate`). Because AILANG records are closed at their annotated type, every one of the ~17 published extensions must write **all 8 hook fields** even when it overrides one, and **adding a 9th field is a major ABI break** — measured in [m-motoko-dst-refactor-migration](../m-motoko-dst-refactor-migration.md) V30/V31: the 2.2.0→5.0 shape change forced a 25-line mechanical port of even the trivial pilot extension.

**Current State (verified)**:
- A record literal missing a field fails to type-check against the closed record type (`ailang check`, Verification Log V1).
- AILANG has **no record-update and no record-spread syntax** — `{ h | f = g }`, `{ ...h, f: g }`, and `{ h..., f: g }` are all parse errors (V2). The maintainer workaround (motoko_agent ADR-001: call `default_hooks()` and rebuild a full literal field-by-field referencing its fields) therefore costs every extension 8+ lines of pass-through noise.
- The ADR-001 default is a **runtime value**, not part of the type: a consumer reading the ABI's `types.ail` cannot see which fields are optional or what the no-op default is.

**Impact**: every current and future extension author (AI agents included — the scaffold template is the first thing they see), and every ABI evolution, which currently must coordinate a major-version repin across all consumers.

## Goals

1. The extension ABI can grow **additively**: a new hook field with a default is a minor-version bump; existing extensions keep type-checking unchanged.
2. **Optionality is type-visible**: a consumer reading the ABI module sees which fields may be omitted and what the default is.
3. A missing field is an **explicit type-level choice**, never an implicit runtime zero (CLAUDE.md §2, no-silent-fallbacks).

## Non-Goals

- No change to effect rows, capability gating, or evaluation semantics of records.
- No optional-field matching on arbitrary open records at runtime.
- No removal of the ADR-001 `default_hooks()` pattern — it remains valid; extensions migrate opportunistically.
- No host (motoko_agent) change is *required* by this feature.

## Design Options

All options stay inside the existing row-polymorphic record system (`types.Row{Labels, Tail}`; open-row annotation `{name: string | r}` verified at V3).

### Option A — `Option`-typed fields (available today, no language change)

Change ABI fields to `Option[(ExtCtx, ...) -> D ! {row}]`. Verified to compose (V4). **Rejected as the primary fix**: it makes optionality type-visible but *worsens* authoring (all 7 unused slots must be written as `None`), and the default still is not part of the type — the host invents the no-op at a match site.

### Option B — open rows with defaults

Declare `ExtensionHooks` open (`{ ...fields | r }`) so consumers return subsets; the host defaults missing fields at dispatch. **Rejected**: a host reading a missing field is precisely an implicit runtime fallback, violating the no-silent-fallback constraint; it also destroys the closed-record totality guarantee the ABI relies on and weakens closed effect rows.

### Option C — field-level optionality in the type

Syntax like `on_pre_step?: Hook` — the type marks fields that may be omitted; consumers that read the record see `Option[Hook]`. Omission becomes type-visible, but the default value still lives outside the type, so the ABI does not grow additively (the host must supply behavior for the absent field anyway). **Rejected as the primary fix**; A captures most of C without new syntax and C keeps the default off-type.

### Option D — **type-anchored defaults (derive-style; recommended)**

The ABI type declaration itself declares default values for designated fields. A record literal checked against that type **may omit defaultable fields**; the type-checker's elaboration pass composes the omitted fields from the declared defaults, producing the **full closed record**. Concretely, a separate top-level declaration in the ABI module (exact syntax TBD in sprint):

```ailang
-- in pkg/sunholo/motoko_ext_abi/types.ail
export type ExtensionHooks = { id: string, provided_tools: [string], on_pre_step: (ExtCtx, [Msg]) -> PreStepDecision ! {...}, ... }

default ExtensionHooks.on_pre_step  = \_ctx _msgs. PassThrough
default ExtensionHooks.on_tool_handle = \_ctx _call. Delegate
-- fields with no `default` declaration remain required
```

An extension then writes only what it overrides:

```ailang
export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  { id: "openkb", provided_tools: [...], on_tool_handle: myHandler }   -- 5 fields completed by elaboration
}
```

**Where the default lives**: type level. It is source in the ABI package, versioned with the package, and visible to any consumer reading `types.ail`. It is *not* a runtime value injected by consumers or hosts.

**What the signature then states**: nothing changes. `register_with_config` still returns the full closed `ExtensionHooks`; downstream consumers see all fields present. The delta is purely at the construction site.

| | A (Option fields) | B (open rows) | C (field `?`) | **D (anchored defaults)** |
|---|---|---|---|---|
| Additive ABI growth | ✗ (major) | ✓ | ✗ | **✓ (minor)** |
| Default visible in type | ✗ | ✗ | ✗ | **✓** |
| Author writes only overrides | ✗ (worse) | ✓ | ✓ | **✓** |
| No implicit runtime fallback | ✓ | ✗ | ✗ (host supplies) | **✓** (elaboration is type-level) |
| New syntax surface | none | none | record types | one new top-level `default` decl |

## Effect-Row Interaction

A defaulted hook still declares its **own** effects — the default is an ordinary expression checked at its declaration site against the field's declared effect row (defaults live in the same module as the type, so the check is local and early). An extension's override must have an effect row subsumed by the slot's declared row, exactly as today (existing effect subsumption, `internal/types/effect_subsumption.go`). Defaults never widen a slot's closed row and never grant a capability the slot did not already declare — the default expression's `--caps` budget is the ABI package's, not the consumer's.

## No-Silent-Fallback Constraint

- Omitting a **non-defaultable** field remains a type error — unchanged behavior (V1).
- Omitting a **defaultable** field is an explicit, type-level completion: the elaborator inserts the declared default, and the completion is **recorded in the elaboration trace** (deterministic, replayable). The runtime evaluator never invents values; there is no null, no zero-value coercion, no host-side `if missing then`.
- The default is data in the versioned ABI package — an auditable source artifact, not ambient behavior.

## ABI Versioning & Migration

1. **Adding a hook with a declared default = minor bump** (e.g. 2.2.0 → 2.3.0). Existing extensions pinned to 2.2.0 keep compiling against their pin; extensions repinning to 2.3.0 compile unchanged and silently receive the new default (recorded in their elaboration trace). Adding a hook *without* a default remains a major break, by design.
2. **~17 existing extensions need no change.** They may simplify to override-only literals opportunistically. ADR-001 `default_hooks()` remains correct runtime code; no forced migration.
3. **Host is unaffected**: it consumes completed closed records as today. Pin consistency across host/extension is enforced by `ailang lock` exactly as now.
4. The scaffolder templates (`cmd/ailang/init_motoko_extension_templates.go`) are updated *after* the language feature ships, in a follow-up, so generated packages demonstrate the override-only idiom.

## Conflict Surface

Touches `internal/ast/` (new decl kind), `internal/types/` (unification + elaboration), `internal/eval/` only via the elaborated AST (no evaluator change).

1. **Positions extended**: record literals checked against an annotated record type that has `default` declarations; one new top-level declaration form.
2. **Existing constructs in those positions**: full record literals (must keep working verbatim); record subsumption of supersets; open-row annotations `{f | r}`; ADT constructor records; nested record literals. A bare `default X.f = e` top-level decl does not collide with the record-literal grammar (unlike Option C's in-type `?`, which would extend the type grammar where none exists today).
3. **Disambiguation**: `default` parsed as a new top-level keyword introducing a declaration; record literal and type grammars untouched.
4. **Fixtures that MUST still work** (exist, verified — V5): `examples/record_cons_pattern.ail`, `examples/record_in_result.ail`, `examples/record_list_extraction.ail`, `examples/extension_self_disable.ail`, the Option-typed-field pattern (V4).
5. **Intentional incompatibilities**: none for existing programs — programs that previously failed to type-check (omitted fields) now succeed; nothing that compiled before stops compiling.

## Testing Strategy

- `internal/types` unit tests: default application, omitted-required-field still errors, default-vs-slot effect row mismatch errors, nested/annotated positions.
- Elaboration-trace test asserting the completion is recorded.
- Golden regression: fixtures above byte-identical post-change.
- Interface cache: new default info must survive gob/JSON interface marshalling (see Risk 1) or fail loudly.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Interface cache cannot marshal new decl kind (a live precedent: `CACHE_WRITE_FAILED` on `RowVar` marshal, V3) | Medium | Marshal-or-error at encode time (no silent fallback); cache key includes defaults fingerprint. Fixing the existing `RowVar` marshal gap is in scope. |
| Interaction with row-polymorphic unification / subsumption | Medium | Focused unification tests; defaults resolved before unification, not during. |
| Security-sensitive area; bad completion could hide a capability grant | High→Low | Defaults checked at declaration site against the slot's row; quorum review before sprint; trace records every completion. |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Completion is a deterministic, traced elaboration step |
| A2: Replayability | +1 | Trace records which defaults fired; replay reproduces the record |
| A3: Effect Legibility | +1 | Default's effect row checked at declaration; no hidden effects |
| A4: Explicit Authority | 0 | No capability changes; defaults cannot widen a slot's row |
| A5: Bounded Verification | +1 | Locally checkable; errors at declaration and literal sites |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Extensions authored by AI agents shrink to override-only literals; optionality machine-readable from the type |
| A8: Minimal Syntax | +1 | One new top-level decl; record grammar untouched |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Composes with existing subsumption and row polymorphism |
| A11: Structured Failure | 0 | Missing-required-field error unchanged and structured |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** ✅ Proceed to quorum/planning.

### Hard Violation Check

- [x] A1 (Determinism): completion is deterministic and traced — no implicit nondeterminism
- [x] A3 (Effects): default's effects declared and checked at its declaration site
- [x] A4 (Authority): no ambient access; defaults live in the already-trusted ABI package
- [x] A7 (Machines First): reduces token cost for AI-authored extensions; optionality is type-readable

## References

- **Motivation**: issue #905 (sunholo-data/ailang); motoko_agent ADR-001 `default_hooks()` workaround (runtime-value default)
- **ABI break measurement**: [m-motoko-dst-refactor-migration.md](../m-motoko-dst-refactor-migration.md) (V30, V31: additive-at-type-level but field-forced break; 12 packages pin `motoko_ext_abi = "2.2.0"`)
- **Scaffolding / author flow**: [motoko-extension-development.md](../../docs/docs/guides/motoko-extension-development.md); `cmd/ailang/init_motoko_extension_templates.go`
- **Record system docs**: [language-syntax.md](../../docs/docs/reference/language-syntax.md) (record subsumption; row polymorphism section)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | A record literal missing a field fails against a closed record type | `ailang check` on `type Hooks = {id, f}` vs literal `{id: "x"}` | **Confirmed** — type error: "record field mismatch: expected 1 fields, got 2 … Hint: Use open record syntax" (transcript in session, 2026-09-13) |
| V2 | No record-update / record-spread syntax exists | `ailang check` on `{ h \| f = g }`, `{ ...h, f: g }`, `{ h..., f: g }` | **Confirmed** — all three are parse errors (PAR_UNEXPECTED_TOKEN / PAR_NO_PREFIX_PARSE), so ADR-001's workaround requires a full field-by-field literal |
| V3 | Open-row record annotations type-check (`{name: string \| r}`) | `ailang check` | **Confirmed** — `✓ No errors found!`; also surfaced `CACHE_WRITE_FAILED: json: MarshalKind: unsupported kind *types.KRow` on `RowVar` (non-fatal, cache disabled for such modules) — feeds Risk 1 |
| V4 | `Option[func-ty]` fields compose today | `ailang check` on Option-typed hook field + match | **Confirmed** — `✓ No errors found!` |
| V5 | Regression fixtures exist | `ls examples/record_cons_pattern.ail record_in_result.ail record_list_extraction.ail extension_self_disable.ail` | **Confirmed** — all four exist |
| V6 | ExtensionHooks has 8 function-typed hook fields + `id` + `provided_tools`, closed effect rows | Read `cmd/ailang/init_motoko_extension_templates.go` `tmplImplAil` (v0.18.5+, mirrors ABI 2.2.0 shape) | **Confirmed** |
| V7 | ABI growth is currently a forced break for all pinned consumers | [m-motoko-dst-refactor-migration.md](../m-motoko-dst-refactor-migration.md) V30/V31/V5 (12 packages pin 2.2.0; pilot port = 25 changed lines) | **Confirmed** (cited measurement, not re-measured) |
| V8 | Checks ran against the installed v0.38.5 binary (commit 2a2377b) while the workspace shallow clone holds a single newer HEAD commit (agent-set deploy work) | `ailang --version`, `git log --oneline -1`, `git cat-file` on build SHA | **Confirmed** — build SHA absent from the shallow clone, so a HEAD-diff of `internal/types` is impossible; the checks above exercise unchanged core record semantics from the installed release binary |

## Open Questions (for quorum / sprint planning)

1. Exact surface syntax: standalone `default Type.field = expr` vs annotation inside the type. This doc fixes the *semantics* (defaults are declarations in the defining module, resolved at elaboration) and prefers the standalone decl for the smaller grammar footprint.
2. Should a completed field be re-observable (e.g. a strictness flag or trace-only)? Current answer: trace-only; the completed value is indistinguishable from a hand-written one — identical artifact, so refusing elaboration would buy no integrity.
3. Should `default` declarations be exported through package interfaces (yes — they must be, for cross-package ABI use), and what is the interface-version implication of adding/removing a default?