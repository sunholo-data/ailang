# IFC export analysis must not reject unit-pattern parameters (m-ifc-unit-param-summary-name)

**Status**: Planned
**Target**: v0.57.5
**Priority**: P0 (regression: `ailang check` and `ailang run` fail on valid programs)
**Estimated**: 1 day
**Dependencies**: None (builds on the IFC cross-module summary foundation, v0.55.0, and the module-function summary fixed point, v0.56.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Synthesized names are index-based (`arg0`, `arg1`, …), so the persisted summary is a deterministic function of the source module — no name mangling that depends on analysis order. |
| A2: Replayability | 0 | No change to traces or execution. |
| A3: Effect Legibility | 0 | No effect-surface change. |
| A4: Explicit Authority | +1 | The IFC persistence boundary keeps its fail-loud corruption sentinel (empty name stays rejected at `Validate`/decode); we fix the producer, not the guard. |
| A5: Bounded Verification | +1 | The fix is local (one mint site, ~5 LOC) and verifiable by unit + integration tests; no new analysis pass. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | Restores `check`/`run` acceptance of a construct AI-generated code already produces (the repro is 3 lines); a rejecting compiler is the most expensive machine-facing failure mode. |
| A8: Minimal Syntax | +1 | No new syntax; accepts an existing, previously-working shape (`func f(())`) with no parser change. |
| A9: Cost Visibility | 0 | No cost-surface change. |
| A10: Composability | +1 | Reuses the existing `arg%d` synthesis convention from `IFCSummaryFromScheme`, keeping the two summary producers canonical in the same way. |
| A11: Structured Failure | +1 | The user-facing error no longer mislabels a non-exported function as an "export"; remaining failures keep their typed, actionable wording. |
| A12: System Boundary | 0 | No boundary changes; summaries still cross the persistence boundary only for exports. |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

`ailang check` — and `ailang run`, which shares the compile pipeline — fails on a valid module
that declares a function with a **unit-pattern parameter**:

```ail
module r
func f(()) -> int = 1
export func main() -> () ! {IO} = ()
```

```
Error: interface build error in r: IFC export analysis: export f: missing IFC parameter name
```

**Verified transcript** (ailang v0.57.4, binary `dec3e87`, 2026-11-06, see Verification Log V1):
exit 1 with exactly the error above. The same failure occurs for `export func f(()) -> int = 1`
(V2) and for `ailang run` (V3). The user's real-world case was a **non-exported** function
(`processCwds`), reported as "export processCwds" — which also exposes the misleading error
wording: the IFC pass validates and reports **every** module function as though it were an
export.

This is a regression: the code was written 2026-10-27 and passed on the binary in use then
(v0.56-era). The summary machinery that now hard-fails was wired in v0.55.0 ("IFC
cross-module summary and cache foundation", changelogs/v0.32-current.md, 2026-10-29) and
extended to a module-wide fixed point in v0.56.0 (m-ifc-function-value-sinks, #1719,
2026-10-29). The repo checkout available to this design session is a shallow clone (depth 1),
so the exact introducing commit cannot be bisected here; the version window v0.55.0–v0.57.4
is established from the changelog.

**Current State:**
- `func f(())` parses to `ast.Param{Name: "", Type: nil}` (`internal/parser/parser_lambda.go`,
  `parseParams`: a non-IDENT, non-reserved token becomes a param with no name and no type) —
  the **only** construct that mints a nameless parameter (V4: tuple patterns `(a, b)` and
  annotated unit patterns `((): ())` are parse errors; the wildcard `func f(_)` gets
  `Name: "_"` and passes).
- Elaboration binds the anonymous parameter as `""` (`internal/elaborate/expr_simple.go`,
  `normalizeLambda` copies `p.Name` verbatim), and type checking, effect checking and
  evaluation all accept it — only the IFC summary pass rejects it.
- `types.AnalyzeModuleIFC` (`internal/types/ifc_summary.go`) builds an `ImportedIFCSig` for
  **every** `file.Funcs` entry, calls `s.Validate()` on each, and fails with
  `"export %s: %w"`. `Validate` rejects `IFCParam.Name == ""` (line 102) because an empty
  name is the **decode-corruption sentinel**: a zero-valued or badly decoded summary is how
  missing metadata is detected at persistence boundaries ("recompile module from source",
  v0.55.0 changelog: "incomplete/malformed v2 summaries are rejected").

**Impact:**
- Any module using a unit-pattern parameter — a previously legal, concise spelling for "one
  argument of unit type, ignored" — cannot be checked, run, or compiled. Both named and
  wildcard workarounds exist (`func f(u: ())`, `func f(_)`) but require editing working code.
- AI-generated code hits this without warning; a `check` regression is the most expensive
  failure class for the program's machine-decidability goal (A7).

## Root Cause

Two independent defects, one failure:

1. **The summary producer assumes every parameter is named.** `buildIFCSig`
   (`internal/types/ifc_check.go`) copies `p.Name` from the AST into `ifcParam.name`, and
   `AnalyzeModuleIFC` mints `IFCParam{Name: p.name, …}` (`internal/types/ifc_summary.go:186`)
   verbatim. For a unit-pattern parameter this is `""`, which `Validate` — correctly, for the
   decode boundary — treats as corrupted metadata. The producer and the validator disagree
   about whether a name always exists.
2. **The validation error mislabels scope.** `AnalyzeModuleIFC` validates all module
   functions (needed: exported callers' intrinsics depend on local callees via
   `convergeExportIntrinsics` and `producerCallableResult`), but the error prefix is
   unconditionally `"export %s"`. A non-exported function like the user's `processCwds` is
   reported as an export, and persistence-boundary strictness is applied to summaries that
   are never persisted (both call sites — `internal/iface/builder.go:247` and
   `internal/runtime/runtime.go:494` — consume only exported names).

## Goals

**Primary Goal:** `ailang check` and `ailang run` accept modules whose functions have
unit-pattern parameters, with no change to IFC enforcement, the iface schema, or the
corruption sentinel at decode boundaries.

**Success Metrics:**
1. The 3-line repro (non-exported `func f(())`) and its exported variant both pass
   `ailang check` and run.
2. The persisted summary for such a function is canonical and deterministic: params are
   positional, arity-matched, and carry synthesized non-empty names.
3. `Validate` still rejects a hand-corrupted summary with an empty param name (the existing
   `TestIFCSummaryValidation` "param name missing" case stays green).
4. All 530 `examples/` + `std/` modules keep byte-identical `check` output (no summary change
   for modules without nameless params).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Synthesize positional names (`arg%d`) at the `AnalyzeModuleIFC` mint site, not in `buildIFCSig`/parser | Mint-site synthesis leaves `ifcEnv` keys untouched (`""` binding, exactly today's behaviour) and cannot collide with user identifiers; parser/buildIFCSig synthesis would change flow-analysis environments (see Risk R1) | agent (this design; ratified here) | design | low |
| D2: Keep validating **all** module functions (exports and non-exports) in `AnalyzeModuleIFC`, but fix the error prefix to name the function accurately | Trade-off between fail-loud defence-in-depth and applying persistence strictness beyond the persistence boundary; post-D1 no valid program can fail (V11), so validation of non-exports becomes a pure internal-invariant tripwire | human (approve or flip to exports-only, see Alternatives A2) | design | low |
| D3: Reuse the existing `arg%d` convention from `IFCSummaryFromScheme` rather than a new reserved name shape | Canonical-form consistency across the two summary producers; `hash_projection` includes the full summary JSON for labelled exports, so the name shape becomes part of package identity | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — mint-site synthesis (ratified in this doc; see Verification Log V8 for the
  collision argument).
- [ ] D2 — human approval: validate-all-with-honest-wording (recommended) vs
  exports-only-validation (alternative A2). Both are implementable in < 10 LOC; the choice
  is a doctrine call (fail-loud breadth vs boundary-scoped strictness), not a technical one.
- [x] D3 — `arg%d` naming (follows the existing convention; no reviewer latitude needed).

## Solution Design

### Overview

Fix the producer, not the guard. When minting `IFCParam` in `AnalyzeModuleIFC`, a parameter
whose surface name is empty (unit-pattern `()`) receives a deterministic positional name
`arg<i>` — the same convention `IFCSummaryFromScheme` already uses for synthetic exports with
no surface body. `Validate`'s empty-name rejection stays byte-for-byte intact as the decode
corruption sentinel. Independently, the analysis error stops calling every function an
"export".

### Architecture

**Components:**

1. **Name synthesis at the mint site** (`internal/types/ifc_summary.go`, the
   `AnalyzeModuleIFC` export loop, currently line 186):
   ```go
   for i, p := range sig.params {
       name := p.name
       if name == "" {
           name = fmt.Sprintf("arg%d", i) // unit-pattern params; matches IFCSummaryFromScheme
       }
       …
       s.Params = append(s.Params, IFCParam{Name: name, …})
   }
   ```
   `sig.params` is built 1:1 from `fn.Params` (`buildIFCSig` appends one `ifcParam` per
   `ast.Param`), so the index `i` is the parameter position — the same position every
   consumer already uses (`ifc_validate.go` arity check, `importedIFCResult`,
   `producerCallableResult` all index positionally; V6).

2. **Error wording** (same function, currently line 190): replace
   `fmt.Errorf("export %s: %w", fn.Name, err)` with wording that distinguishes scope, e.g.
   `fmt.Errorf("function %s: %w", fn.Name, err)` for non-exports (keep `"export %s"` when
   `fn.IsExport`, which exists on `ast.FuncDecl`, V7). With D2 approved as recommended, this
   is the only wording change; with A2 chosen instead, validation is skipped for
   `!fn.IsExport` entirely and the prefix stays `"export %s"`.

Nothing else changes. `buildIFCSig`, `ifcEnv` construction, `checkFunc`, the summary fixed
point (`convergeExportIntrinsics`), `GobEncode`/`UnmarshalJSON`, `hash_projection`, and the
iface schema are untouched. The `""` binding in `ifcEnv` for anonymous parameters remains
exactly as today (and is safe: a nameless param can carry no declared label or `{not ℓ}`
refinement, because `parseParams` only parses `LabelledType` annotations after
`IDENT COLON` — V9 — and two nameless params both bind bottom labels, so the `env[""]`
overwrite in `checkFunc` is lossless today; see Future Work).

### Implementation Plan

**Phase 1: Fix + unit tests** (~3 hours)
- [ ] Mint-site name synthesis in `internal/types/ifc_summary.go` (~3 LOC).
- [ ] Error-prefix change in the same loop (~2 LOC).
- [ ] Unit tests in `internal/types/ifc_summary_test.go`: non-exported `func f(())`
  (the reported repro), `export func f(())`, mixed `func g(x: string, ())`
  (→ `Params[1].Name == "arg1"`), `func h((), ())` (→ `arg0`, `arg1` distinct),
  `func f(_, ())` / `f((), _)` (wildcard interplay), and a gob round-trip of a synthesized
  summary through `GobEncode`/`GobDecode`/`Validate`.

**Phase 2: Integration + regression fixtures** (~3 hours)
- [ ] Pipeline/builder-level test: compile the 3-line repro through the module-compile path
  that produced "interface build error" (`internal/pipeline/pipeline_module_compile.go:533`
  wraps `iface.BuildInterfaceWithTypesAndConstructors`) and assert success, mirroring the
  existing iface test style (`internal/iface/hash_projection_ifc_test.go`,
  `internal/iface/ifc_summary_test.go`).
- [ ] Verify `Validate` sentinel intact: `TestIFCSummaryValidation` "param name missing" case
  unmodified and green.
- [ ] Run `make test-core` plus the full `make test`; confirm no check-output change on
  `examples/` and `std/` (the v0.57.1 changelog's 530-module identical-output method).

**Phase 3: Documentation** (~1 hour)
- [ ] Changelog entry under v0.57.5 (Fixed), citing this doc and the repro.
- [ ] No language-reference change needed: unit-pattern parameters are already legal syntax
  (`ailang prompt` surface unchanged); confirm no LIMITATIONS entry claims otherwise.

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `internal/types/ifc_summary.go` — name synthesis + error prefix (~5 LOC changed).
- `internal/types/ifc_summary_test.go` — new `TestAnalyzeModuleIFCNamelessParam`-style unit
  tests (~70 LOC added).
- `internal/iface/` (or `internal/pipeline/`) — one integration regression test (~40 LOC,
  location per existing conventions; executor latitude).
- `changelogs/v0.32-current.md` — v0.57.5 entry (~15 LOC).

## Examples

### Example 1: The reported repro (non-exported)

**Before (v0.57.4):**
```
$ ailang check r.ail
Error: interface build error in r: IFC export analysis: export f: missing IFC parameter name
```

**After:**
```
$ ailang check r.ail
✓ No errors found!
```
The summary for `f` (internal, and in `out` for future reuse) carries
`Params: [{name: "arg0", label: ⊥, forbidden: [], applied: ⊥}]` — arity 1, positionally
identical to what `func f(u: ())` produces.

### Example 2: Mixed named/anonymous

```ail
module m
func g(x: string, ()) -> string = x
export func main() -> () ! {IO} = ()
```
`Params[0].Name == "x"` (unchanged), `Params[1].Name == "arg1"` (synthesized). No collision
is possible: `arg1` exists only in the persisted summary, never as an `ifcEnv` key.

## Success Criteria

- [ ] `ailang check` passes on the non-exported repro, the exported variant, and the mixed
  `func g(x: string, ())` variant (acceptance: new unit + integration tests).
- [ ] `ailang run` passes on the repro (acceptance: integration test or manual transcript).
- [ ] Synthesized names are deterministic and positional; `arg0`/`arg1` do not collide with
  user-named parameters in the persisted form (acceptance: unit test with
  `func f(arg0: int, ())`, which must yield `Params[1].Name == "arg1"` — no keying on names).
- [ ] `TestIFCSummaryValidation` ("param name missing" case) unchanged and green —
  `Validate` still rejects empty names at persistence boundaries.
- [ ] All tests passing (`make test`, `make test-core`); `make fmt`/`make lint` clean;
  `make check-boundaries` clean (no new import edges — `internal/types` gains nothing).
- [ ] Changelog updated; no doc claims AILANG lacks unit-pattern parameters.

## Testing Strategy

**Unit tests (`internal/types/ifc_summary_test.go`):**
- `AnalyzeModuleIFC` on modules containing, in one table: `func f(())` non-exported,
  `export func f(())`, `func g(x: string, ())`, `func h((), ())`, `func f(_, ())`,
  `func f((), _)`, `func f(_: (), ())` — all must return nil error (all fail identically
  today, V5/V10) and mint expected names.
- Gob round-trip: `GobEncode` → `GobDecode` → `Validate` on a synthesized summary.
- Negative control: a hand-built `IFCParam{Name: ""}` still fails `Validate` (already
  covered by `TestIFCSummaryValidation`; keep as-is).

**Integration tests:**
- Compile the 3-line repro through the module-compile pipeline (the exact path that emitted
  "interface build error") and assert success.
- `examples/runnable/secrets/` IFC fixtures (`gated_secret.ail`, `sink_function_value.ail`,
  `sink_function_value_ok.ail` — all verified to exist, V12) still produce identical
  check output: labelled-parameter and sink behaviour must be untouched.

**Manual testing:**
- `ailang check` + `ailang run` on the repro with a locally built binary; before/after
  transcript recorded in the sprint report.

## Conflict Surface

*(Required: this design touches `internal/types/` and `internal/iface/`.)*

1. **Positions extended:** none. No syntax, type, or effect position changes. The change is
   inside the summary producer for an AST shape that already parses, typechecks and
   evaluates.
2. **Other constructs in the same position:** the `IFCParam.Name` field is minted at exactly
   one AST-derived site (`ifc_summary.go:186`). The other mint site is
   `IFCSummaryFromScheme` (`ifc_summary_synthetic.go:22`), which already synthesizes
   `arg%d` — the two producers become consistent. Consumers of `Name`:
   `Validate` (sentinel), `ifc_validate.go` (positional arity only, V6),
   `hash_projection.go` (hashes the full summary JSON for labelled exports — names feed
   package identity, but no module with a nameless-param export could ever publish under
   v0.55.0–v0.57.4 because building it errored, so there is no identity compatibility to
   preserve, V13).
3. **Disambiguation:** N/A (no parser/typechecker change). The synthesis is keyed on
   `p.name == ""`, which by V4 can only arise from the unit-pattern parameter shape.
4. **Programs that MUST still work (fixtures):**
   - `examples/runnable/secrets/gated_secret.ail`, `sink_function_value.ail`,
     `sink_function_value_ok.ail` (verified to exist, V12) — labelled params, sinks,
     function-value requirements.
   - `internal/types/ifc_summary_test.go::TestIFCSummaryShadowingAndBottom` — asserts
     summaries for **non-exported** locals (`local`, `forward`, `shadow`) are present in the
     `out` map: the map must keep covering all functions (this is why D2 does not narrow the
     map itself).
   - `internal/iface/hash_projection_ifc_test.go::TestIsUnlabelled` — "a parameter name is
     not security information": renaming must not flip labelled-ness; synthesis of a name
     for an unlabelled param keeps the summary unlabelled and the identity projection
     unchanged.
   - The 530 `examples/`/`std/` modules (changelog v0.57.1 method).
5. **Deliberate changes (intentional incompatibilities):**
   - The error string changes for the (previously always-fatal) analysis-failure path:
     `"export f: …"` becomes `"function f: …"` for non-exports. No test asserts the current
     wording (grep-verified: the only occurrences of the prefix are the two non-test
     mint sites, V14), so nothing downstream pins it.
   - Persisted summaries for exports with nameless params gain `arg<i>` names where no
     valid persisted form existed before — strictly additive.

## Verification Log

*(Hard-gate record: every language/behaviour claim above was checked live or by reading the
cited code.)*

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `ailang check` fails on the 3-line non-exported repro | Live `ailang check` (v0.57.4, `dec3e87`), 2026-11-06 | exit 1, `interface build error … export f: missing IFC parameter name` |
| V2 | Exported variant also fails | Live check on `export func f(()) -> int = 1` | same error, `export f` |
| V3 | `ailang run` also fails | Live `ailang run` on repro | same error |
| V4 | `()` is the only nameless-param construct; tuples `(a,b)`, annotated `((): ())`, and lambda `\(). …` are parse errors; `_` binds the name `"_"` | Live check on 6 crafted modules (t1–t6) | confirmed: `f(_)` passes today; all others PAR_* or the IFC error; `parseParams` read (`internal/parser/parser_lambda.go:158`) shows the non-IDENT branch mints `Name:""`, `Type:nil` |
| V5 | Mixed `(_, ())`, `((), _)`, `(_: (), ())`, `((), ())`, `(x, ())` all fail identically today | Live check (t7, t5) | confirmed — one unified fix covers all shapes |
| V6 | IFC summary consumers are positional, not name-keyed | Read `internal/iface/ifc_validate.go` (arity check `len(item.IFC.Params) != arity`), `importedIFCResult`, `producerCallableResult` (index-based); `TestIsUnlabelled` asserts names are not security information | confirmed |
| V7 | `ast.FuncDecl.IsExport` exists for scope-accurate wording | Read `internal/ast/ast_decl.go:55`; parser sets it (`internal/parser/parser_func.go` "Handle export prefix") | confirmed |
| V8 | Mint-site synthesis cannot collide with user identifiers in flow analysis | Read `buildIFCSig` → `ifcParam.name` feeds `ifcEnv` keys (`ifc_check.go:195-198`); mint site (`ifc_summary.go:186`) feeds only the persisted `IFCParam`. Synthesis at the mint site leaves `ifcEnv` keyed `""` exactly as today | confirmed — a user-written `func f(arg0, ())` cannot create an env-key collision because the synthesized name never enters any env |
| V9 | A nameless param can never carry a declared label or `{not ℓ}` refinement | Read `parseParams`: `LabelledType` is parsed only in the `IDENT COLON` branch; non-IDENT params get `Type: nil` | confirmed — synthesized params always have `Label ⊥`, `Forbidden []` |
| V10 | Regression window v0.55.0–v0.57.4; exact commit not bisectable | `git log` on this checkout (shallow, depth 1); changelog dates the summary foundation (v0.55.0, 2026-10-29) and the module-wide summary fixed point (v0.56.0, #1719, 2026-10-29) | confirmed as stated; doc says "window", not "commit" |
| V11 | Post-fix, no valid program can fail `Validate` in `AnalyzeModuleIFC` | Enumerated `Validate` failure modes against the mint loop: names (synthesized), `Forbidden` (always `[]string{}`), label arrays (`EncodeIFCLabel` always sets both), declassify (sorted before validate), `HasReturnLabel` (set with `Return`), label names (from IDENT syntax, never `""`/`"*"`/unsorted) | confirmed — remaining failures are internal-invariant breaks, which D2 keeps as fail-loud tripwires |
| V12 | Cited regression fixtures exist | `ls examples/runnable/secrets/` → `gated_secret.ail`, `sink_function_value.ail`, `sink_function_value_ok.ail`, … | confirmed |
| V13 | No published identity exists for nameless-param exports | Building such a module errored in v0.55.0–v0.57.4 (V2), so no `ifacev2:` hash of a nameless-param summary can exist in the wild | confirmed |
| V14 | No test asserts the current `"export %s"` analysis-error wording | `grep -rn "export %s\|IFC export analysis" --include="*_test.go" internal/` → no hits | confirmed |
| V15 | Duplicate/coverage gate: no existing planned or implemented doc covers this | `create_planned_doc.sh` dual search (SimHash + neural) on "ifc unit param summary name" → (none found) in both `implemented/` and `planned/` | confirmed |

## Alternatives Considered

**A1: Relax `Validate` to allow empty param names.** Rejected: the empty name is the
decode-corruption sentinel ("missing IFC summary; recompile module from source" family);
weakening it trades a verified guard for producer convenience, and `TestIFCSummaryValidation`
pins the rejection as intended behaviour. The bug is in the producer.

**A2: Validate only `fn.IsExport` functions in `AnalyzeModuleIFC`.** Viable and simpler in
intent (persistence-boundary strictness only where persistence happens — both call sites
consume only exports). Not recommended as the default: after D1, validation of non-exports
can only fire on internal invariant breaks, where failing loudly is the repo doctrine, and
the `out` map must keep covering all functions anyway (`TestIFCSummaryShadowingAndBottom`
reads non-exported summaries). Presented to the human as the D2 alternative; if chosen, the
`"export %s"` prefix stays for the validated remainder and the wording change is dropped.

**A3: Synthesize names in the parser or `buildIFCSig`** (give the anonymous param a real
identifier). Rejected: it changes `ifcEnv` keys and the elaborator's `""` binding, risking
identifier collisions (a user-written `arg0` param) and flow-analysis drift in
security-sensitive code (`internal/types/` is semantics-sensitive per AGENTS.md) — all to
solve a problem that exists only in the persisted summary.

**A4: Reject unit-pattern parameters in the parser.** Rejected: it worked before v0.55,
the construct is meaningful ("one unit-typed argument, ignored"), and outlawing previously
valid source is the worst resolution for machine-generated code (A7/A8).

## Deferred Decisions

- Exact integration-test location (`internal/iface` vs `internal/pipeline`) — agent may
  choose, following existing conventions.
- Whether to also emit a `go vet`-style internal comment documenting the `env[""]`
  overwrite for multiple nameless params (harmless today per V9) — agent may choose.

## Non-Goals

- **Cross-module enforcement of non-exported summaries** — the `out` map keeps covering all
  functions for future M3 reuse, but nothing new consumes non-exported entries.
- **Parser support for tuple-pattern parameters** (`func f((a, b))`) — a separate language
  feature; today it is a parse error and stays one.
- **Changing `Validate`, the iface schema, or the cache version** — no schema bump: the fix
  produces summaries that already validate under the current canonical form.
- **Fixing the `env[""]` collision for multiple nameless params in flow analysis** —
  impossible to observe (V9); revisit only if labelled nameless params ever become parseable.

## Timeline

**Week 1** (1 day total):
- Phase 1: fix + unit tests (morning)
- Phase 2: integration test + full-suite verification (afternoon)
- Phase 3: changelog (end of day)

**Total: ~1 day** (realistic 2× estimate of the ~3-hour mechanical work, per the skill's
estimation guidance).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| R1: Synthesized names leak into flow analysis and collide with user identifiers | High (security-sensitive package) | Mitigated by design: synthesis happens only at the persisted-summary mint site; `ifcEnv` keys are untouched (V8). Test `func f(arg0: int, ())` explicitly. |
| R2: Hash-projection identity changes for labelled exports | Med | Only for modules that could not publish at all before (V13); unlabelled summaries are dropped from the projection entirely (`IsUnlabelled → nil`, existing behaviour), so unlabelled identity is untouched. |
| R3: Some unforeseen AST shape still yields `Name == ""` with a label | Low | V9 proves annotations cannot attach to non-IDENT params; the mixed-shape test table (V5) covers every nameless variant the parser accepts. |
| R4: D2 rejected late, forcing rework | Low | Both D2 options are < 10 LOC in the same function; the choice is isolated to the wording/validation lines. |

## Related Documents

**Implemented (may inform design):**
- [m-ifc-function-value-sinks](../implemented/v0_56_0/m-ifc-function-value-sinks.md) —
  introduced the module-function summary fixed point whose export analysis surfaces this bug
  (v0.56.0, #1719).
- [m-ifc-cross-module-labels / IFC cross-module summary and cache foundation]
  (changelogs/v0.32-current.md, v0.55.0) — defined `ImportedIFCSig`, `Validate`'s
  persistence-boundary role, and the empty-name corruption sentinel this design preserves.

**Planned (check for overlap):**
- (none found — dual SimHash + neural search returned no matches, V15)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- IFC labels guide: `docs/docs/guides/ifc-labels.mdx`
- Changelog: `changelogs/v0.32-current.md` — v0.55.0 summary foundation, v0.56.0
  function-value sinks, v0.57.1 "identical check output" verification method
- Code: `internal/types/ifc_summary.go` (mint site line 186, `Validate` line 102),
  `internal/types/ifc_check.go` (`buildIFCSig`), `internal/parser/parser_lambda.go`
  (`parseParams`), `internal/iface/builder.go:247`, `internal/pipeline/pipeline_module_compile.go:533`

## Future Work

- If tuple-pattern parameters (`func f((a, b): (int, int))`) are ever added to the parser,
  the mint site already generalizes: any pattern param without a surface name gets `arg<i>`,
  and `patternVars` covers destructured labels in bodies.
- If labelled unit-pattern params ever become parseable, revisit the `env[""]` overwrite
  (Non-Goal above) — the collision would then become observable.

---

**Document created**: 2026-11-06
**Last updated**: 2026-11-06
