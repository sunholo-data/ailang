# M-IFC-FUNCTION-VALUE-SINKS — Sink requirements travel with the function value

**Status**: Planned
**Target**: v0.54.19 (next patch; P0 soundness)
**Priority**: P0 (High — a security control is bypassable in one `let`)
**Tracks**: #1719
**Estimated**: 5 days (M1 2d, M2 2d, M3 1d)
**Dependencies**: None hard. Soft coordination with PR #1771 (M-IFC-CROSS-MODULE M1–M2); see
[Relationship to PR #1771](#relationship-to-pr-1771-m-ifc-cross-module).
**Planner-Lane**: opus-required (semantics change to `internal/types`)

> **Semantics change to `internal/types`.** Programs that compile today will be rejected. Every
> program rejected by this change passes a labelled value to a sink through an alias, a
> higher-order function or a wrapper. Those programs leak today, so the change is intended.
> M3 measures the breakage before the change ships. See [Breakage risk](#breakage-risk--how-to-measure-it).

## Summary

Check A (`x: T{not ℓ}`) and Check C (`x: T<ℓ>`, positive parameter coverage) fire only when the
callee of a call is written as **the bare name of a top-level function in the same module**. The
IFC pass models every value as a `Label` and nothing else. A function's argument requirements live in
a per-declaration table, `c.sigs`, which is keyed by name and read only at a call whose callee
expression *is* that name. Any other route to the same function value loses the requirement and
goes through the label-only "unknown callee" fallback: a `let` alias, a higher-order parameter, a
record field, a list or tuple element, an `if` branch, a function's return value, a function
literal, or a wrapper lambda.

The fix gives the IFC pass's abstract value a second component, a **callable demand**: what this
function value requires of each argument. It is computed where the function is declared, carried
through every binding form the label already travels through, and checked at **every
application**, whatever the callee expression is. Higher-order calls are checked against a
per-parameter **applied-at** summary. That summary is computed once per function with label
variables standing in for its parameters. Wrappers get **inferred** demands from the same pass.
Nothing changes in HM types, CoreTI, codegen or the iface schema.

## Problem Statement

### The repros (base `772703831` = origin/dev, also the installed v0.54.18 `cd33d05`)

Common prelude, written to every file (`module r/<name>`, checked from the parent directory so
the module path matches):

```ailang
pure func fromInbox(s: string) -> string<email> { s }

pure func gate(body: string{not email}) -> string { body }

pure func store(q: string<sqlsafe>) -> string { q }
```

| # | Shape | `main` body (plus any extra decl) | `ailang check` today | Must be |
|---|-------|-----------------------------------|----------------------|---------|
| R0 | **control**, direct call | `gate(fromInbox("hi"))` | **exit 1** `sink_refinement` | exit 1 (unchanged) |
| R1 | let alias (#1719) | `let g = gate in g(fromInbox("hi"))` | exit 0 | exit 1 |
| R2 | HOF param (#1719) | `apply1(gate, fromInbox("hi"))` with `pure func apply1(f: string -> string, x: string) -> string { f(x) }` | exit 0 | exit 1 |
| R3 | annotated let | `let g: string{not email} -> string = gate in g(fromInbox("hi"))` | exit 0 | exit 1 |
| R4 | `if` selection | `let g = if true then gate else gate in g(fromInbox("hi"))` | exit 0 | exit 1 |
| R5 | record field | `let r = { f: gate } in r.f(fromInbox("hi"))` | exit 0 | exit 1 |
| R6 | list element | `match [gate] { f :: _ => f(fromInbox("hi")), [] => "" }` | exit 0 | exit 1 |
| R7 | tuple element | `match (gate, 1) { (f, _) => f(fromInbox("hi")) }` | exit 0 | exit 1 |
| R8 | returned function | `getGate()(fromInbox("hi"))` with `pure func getGate() -> (string -> string) { gate }` | exit 0 | exit 1 |
| R9 | composed closure | `compose(gate, ident)(fromInbox("hi"))` with `pure func compose(f: string -> string, g: string -> string) -> (string -> string) { \x. f(g(x)) }` and `pure func ident(s: string) -> string { s }` | exit 0 | exit 1 |
| R10 | HOF applies the sink to its *own* tainted data | `h(gate, "clean")` with `pure func h(f: string -> string, x: string) -> string { let _ = f(fromInbox("x")) in x }` | exit 0 | exit 1 |
| R11 | HOF with a guarded param type | `apply2(gate, fromInbox("hi"))` with `pure func apply2(f: string{not email} -> string, x: string) -> string { f(x) }` | exit 0 | exit 1 |
| R12 | named wrapper, unannotated param | `wrap(fromInbox("hi"))` with `pure func wrap(y: string) -> string { gate(y) }` | exit 0 | exit 1 |
| R13 | lambda wrapper | `let w = \y. gate(y) in w(fromInbox("hi"))` | exit 0 | exit 1 |
| R14 | function-literal sink | `let w = func(y: string{not email}) -> string { y } in w(fromInbox("hi"))` | exit 0 | exit 1 |
| C0 | **control**, Check C direct | `store(fromInbox("hi"))` | **exit 1** `param_label_cover` | exit 1 (unchanged) |
| C1 | Check C via alias | `let s = store in s(fromInbox("hi"))` | exit 0 | exit 1 |
| C2 | Check C via HOF | `apply1(store, fromInbox("hi"))` | exit 0 | exit 1 |

Negative controls. These must stay **exit 0**, and all of them are exit 0 today:

| # | Shape | Body |
|---|-------|------|
| N1 | direct, clean | `gate("clean")` |
| N2 | alias + HOF, clean | `let g = gate in g(apply1(gate, "clean"))` |
| N3 | non-sink alias passed to HOF with tainted data | `let u = upper in apply1(u, fromInbox("hi"))` with `pure func upper(s: string) -> string { s }` |
| N4 | a parameter named `gate` shadows the sink | `k(upper, fromInbox("hi"))` with `pure func k(gate: string -> string, x: string) -> string { gate(x) }` |
| N5 | declassified before the aliased sink | `let g = gate in g(sanitize(fromInbox("hi")))`, with `pure func sanitize(raw: string<email>) -> string<sanitized> ! {Declassify} { "[sanitized]" }` and `main` declaring `! {Declassify}` |
| N6 | sink stored, never applied | `let r = { f: gate, n: 1 } in "ok"` |

All 25 files are in the session scratchpad (`dd2/r/*.ail`). The `ailang-dev` binary built from
`772703831` and the installed `~/go/bin/ailang` (v0.54.18, `cd33d05`) gave identical exit codes
for every row. R1 and R12 also **run** and print `hi` (`ailang run --entry main`). The check is
static only and has no runtime backstop.

> **Cache trap (from `TestIFCAnnotatedLetIsAlsoClosed`):** two builds of the same dirty tree
> share a compile-cache key, so a CLI re-check can grade a stale result. Every acceptance row
> below is pinned as a **unit test on `CheckModuleIFC`** and as a CLI check of a fresh file.

### Root cause (read from the code, `772703831`)

The IFC pass (`internal/types/ifc_check.go`) is a surface-AST walk whose abstract value is
**only a label**:

- `ifc_check.go:105-108`: `ifcVar{label Label; typ ast.Type}` is everything the pass knows
  about a bound value. It has no field for "this value is a function that requires X of its
  arguments".
- `ifc_check.go:71-74` and `:122-152`: argument requirements (`notLabel`, positive `label`)
  live in `ifcSig`, built **only for top-level `file.Funcs`** and stored in `c.sigs` **keyed by
  the declaration's name**.
- `ifc_check.go:378-383`: `labelOfCall` resolves the callee by `calleeName(call.Func)` and looks
  that **string** up in `c.sigs`. A lexically bound name with the same spelling is forced to the
  non-local path. `calleeName` (`:492-497`) returns `""` for any callee that is not an
  `*ast.Identifier` (`r.f`, `getGate()`, `(\y. …)`).
- `ifc_check.go:402-416`: Checks A and C run **only inside `if isLocal`**.
- `ifc_check.go:421-430`: every other callee takes the transparent fallback. It joins the
  callee's *label* with the argument labels and checks nothing.
- `ifc_check.go:186-190`: a top-level function name used as a **value** (`let g = gate`,
  `apply1(gate, …)`) is an `*ast.Identifier` absent from `env`, so it evaluates to `⊥`. The sig
  is not attached, so the requirement is dropped at the reference.
- `ifc_check.go:251-268` and `:301-317`: lambdas and function literals carry only their body
  label. Unannotated lambda params are not rebound, so the `gate(y)` inside `\y. gate(y)` sees
  `y` as `⊥` (R13). A function literal's declared `{not ℓ}` param never becomes a sig (R14).
  Named wrappers (R12) have the same problem: `checkFunc` seeds `y` with its declared label, `⊥`.
- `ifc_static_type.go:129-130`: in a written function type, `walkType` visits only the
  **return** type. `string{not email} -> string` (R3, R11) parses and type-checks, but the
  refinement is ignored.

Why the *type* does not carry it either. HM strips IFC annotations on purpose:
`typechecker.go:288-292` and `elaborate/file_funcs.go:440-442` lower `*ast.LabelledType` to its
base type, and `unification_core.go:231-241` (M-TAINT-TYPES M7) strips `TLabelled` before
unifying. The iface keeps positive labels but **drops refinements**
(`iface/builder.go:119-127`: "Refinements (`{not LABEL}`) are sink-side only and are not stored
on the type itself"). Nothing in the pipeline associates a function value with its argument
requirements. The association exists only between a *name* and a *declaration*.

**In one sentence:** the requirement is attached to the syntactic call `name(args)`, because
the pass's value domain has no place to put it on the function value.

### This is one defect class, not two bugs

The issue reports R1 and R2. The audit above found 15 bypass shapes (R1–R14, C1–C2) with **one**
cause, all measured. R12 (named wrapper) is the "flow through *unannotated* parameters" gap that
`docs/docs/guides/ifc-labels.mdx` (Limitations) already lists. It is in scope here because
closing R1–R11 without it leaves R13, `\y. gate(y)`, as a one-token bypass of the fix. The
effect system had the same defect class (#1326 named callbacks, #573 calls through record
fields, #1718 callbacks inside lists/tuples/ADTs). Its fix
(`design_docs/planned/v0_48_0/m-effect-latent-function-values.md`, implemented in #1708) is the
precedent this design follows: charge the latent requirement of a **function value**, and keep
the requirements written in a **function-type annotation**.

### Impact

- **Who:** every module that declares a sink. In-repo that is `examples/runnable/secrets/*`
  and `examples/runnable/contracts/inbox_*`. Outside the repo: `ailang-demos/leak_lab`, a public
  demo whose whole premise is "try to get the secret past the sink", the motoko fork's mirrors of
  the examples, and `ailang-packages/packages/gmail`.
- **How bad:** the guarantee in `ailang prompt` ("Identity Bypass Is Structurally Impossible",
  "the verifier proves that untrusted data never reaches a sensitive sink") is false in one
  token. An agent that writes idiomatic HOF code (`map(send, bodies)`) gets no protection and no
  warning. This is the silent widening CLAUDE.md §2 forbids, in a security control.

## Goals

**Primary goal:** every application of a function value whose parameters declare `{not ℓ}` or
`<ℓ>`, or have such requirements inferred, is checked against its actual arguments, wherever that
value came from in the module.

**Success metrics:**
- R1–R14 and C1–C2 rejected with `sink_refinement` / `param_label_cover`. R0 and C0 unchanged.
- N1–N6, every existing IFC unit test and the fixtures listed under Conflict Surface stay clean.
- Corpus delta (M3) shows **no new diagnostic** in any module that declares neither a refinement
  nor a positive parameter label, and every new diagnostic elsewhere is triaged as a true leak.
- No change to HM types, CoreTI, codegen, cache keys or the iface schema.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Requirements live in the **IFC abstract domain** (callable demand on the value), not in HM function types | Decides whether unification, CoreTI and codegen change; must agree with #1771's ratified "IFC stays a surface-AST analysis" | human | design | high |
| HOF calls are checked through a per-param **applied-at** summary, with an arg-join fallback for callees that have no summary | Decides precision (`map(gate, cleanList)` accepted) versus over-rejection | human | design | high |
| Wrapper demand inference (R12/R13) is **in scope** | Without it the fix is bypassable via `\y. gate(y)`; with it, wrappers that leak today are rejected | human | design | med |
| Reuse the existing `sink_refinement` / `param_label_cover` kinds; the message names the path the value took | Avoids new diagnostic codes; tooling keyed on kinds keeps working | agent | compile | low |
| Shape of `IFCParam.Forbidden` in #1771's iface summary (string vs list) | Inferred demands can forbid several labels on one param. A string-shaped field shipped now forces a v3 schema bump later | human | before #1771 merges | med |

### Design Freeze

- [ ] **F1** — Requirements live in the IFC domain, not in HM types (Alternative A rejected; see
  below).
- [ ] **F2** — HOF rule = applied-at summary for callees that have a summary, and the conservative
  arg-join for callees that do not.
- [ ] **F3** — Wrapper inference (M2b) ships in the same release as M1. If it slips, M1 does
  **not** ship alone: the issue would close while R13 stays open.
- [ ] **F4** — Ask on #1771 (before it merges): `IFCParam.Forbidden` becomes a sorted `[]string`
  and `IFCParam` reserves `applied` (an `IFCLabel`, the applied-at summary for function-typed
  params). Both fields go in the v2 schema #1771 introduces, so no second bump is needed.

## Solution Design

### Overview

1. **Callable demand as part of the IFC value (M1).** `ifcVar` gains `fn *ifcCallable`:
   positional per-parameter demands (`forbid []string` from `{not ℓ}`, `cover Label` from `<ℓ>`)
   plus a `via` trail used in diagnostics. A reference to a top-level function **as a value**
   resolves to its callable. So does a function literal with declared params, and so does a value
   whose written type is a function type with labelled or refined params (the F1 analogue of the
   effect design's L1). Callables travel with the label through `let`/`letrec`, block lets,
   `if`/`match` (join), records, record update and access, lists, arrays, tuples, ADT
   constructors, pattern variables (which inherit the scrutinee's callable, as they inherit its
   label today), and **call results**: a local callee's result callable is computed like
   `effectiveBodyLabel`, and a call whose arguments carry callables yields their join (R8, R9).
   **Checks A/C run at every application** against the callee expression's callable, whatever
   that expression is. The `isLocal` path becomes the special case "callee is a name whose
   callable comes from `c.sigs`".

2. **Higher-order calls (M2a).** If an argument carries a callable `D` and the callee `h` is
   local, `h`'s summary gives, for each parameter `i`, `applied[i]`: the join of the labels of
   every argument that `h`'s body (transitively) applies parameter `i` to. That label is computed
   with `h`'s parameters seeded as **label variables** `ρ_j` (the lattice already has `labelVar`,
   `labels.go:34-38`, and has no caller in `ifc_check.go` yet). At the call site, substitute
   `ρ_j ↦ actual label of arg j` and check `D` against the result. R2 (`ρ_x ↦ <email>`), R10
   (`applied = <email>` intrinsic) and C2 are rejected. N2 (`ρ_x ↦ ⊥`) and N3 (no callable) pass.
   **Fallback** for callees with no summary (imported until #1771, unknown closures): check `D`
   against the join of all other arguments' full labels and the callee expression's own label.
   That is sound on the same assumptions the transparent fallback already makes (APP-PURE).

3. **Wrapper inference (M2b).** The same `ρ`-seeded pass records a **demand on parameter `j`**
   whenever Check A or C, inside the body, sees an argument label containing `ρ_j` (for example
   `wrap`'s `gate(y)` gives `wrap.y : forbid email`). Lambdas and function literals get the same
   treatment, so `\y. gate(y)` is a callable with that demand (R13). R11 falls out because
   `apply2`'s declared `f : string{not email} -> string` makes `f(x)` record `forbid email` on
   `x`. Summaries iterate to a fixed point over the module's call graph, the same loop shape as
   #1771's `AnalyzeModuleIFC`. The lattice is finite (labels written in the module), so it
   terminates.

The **non-silent** walk (`checkFunc`) keeps today's environment seeding: declared labels, and
unannotated lambda params not rebound. Existing Check A/C outcomes therefore cannot change except
by adding errors. `ρ` variables exist only in the silent summary pass and are erased before any
label reaches a result, a diagnostic or (later) an iface. Erasure means substitution at call
sites, or dropping them from `intrinsic`.

### Architecture

**Components:**
1. **`ifcCallable`** (new, `internal/types/ifc_callable.go`): `params []ifcDemand`, `join`,
   `checkApplication(call, argLabels) []*TypeCheckError`, `fromSig(*ifcSig)`,
   `fromFuncType(ast.Type)`. `ifcDemand{forbid []string; cover Label; hasCover bool}`. The join is
   per position: union of `forbid`; meet of `cover`, which means "only ⊥ passes" when the two
   covers disagree. A join is always at least as strict as each side.
2. **Value resolution** (`ifc_check.go`, `flowOf`/`labelOf` → a `valueOf` that returns
   `(Label, *ifcCallable)`): the Identifier case consults `env`, then `c.sigs` (a top-level
   function referenced as a value), then the #1771 import table once that lands. One resolution
   function replaces the two parallel lookups in `labelOfCall` (`c.sigs[name]`, and after #1771
   `c.imports[name]`).
3. **Summary pass** (`ifc_summary_local.go`, new): per local function and per lambda or function
   literal, a `ρ`-seeded silent walk producing `{applied []Label, inferred []ifcDemand,
   resultCallable *ifcCallable}`, with a fixed-point loop and the existing `computing` cycle guard.
4. **Diagnostics:** the existing `newSinkError` / `newParamLabelCoverError` gain a `via` string
   naming the path, for example `value labelled <email> reaches parameter "body" of gate (via
   let g; passed to apply1 as f) which forbids it (string{not email})`. The kinds are unchanged.

### Implementation Plan

**M1: callable values (~2 days)**
- [ ] `ifcCallable` and `ifcDemand` with join and check, plus unit tests on the join algebra
  (strictness monotone).
- [ ] `ifcVar.fn`, propagated through every case `flowOf` already handles. Pattern variables
  inherit it. The callable of a record access is the record's, conservatively.
- [ ] Identifier-as-value resolves top-level funcs to `fromSig`. Function literals and annotated
  function types give `fromFuncType`. `walkType` is unchanged: the callable is a separate
  projection, so labels and callables cannot contaminate each other.
- [ ] `labelOfCall`: Checks A/C on the callee expression's callable for every application. Keep
  the `isLocal` result-label logic.
- [ ] Call-result callables: a local callee returns its summarised `resultCallable`. An unknown
  callee returns the join of its argument callables.
- [ ] Tests: R0–R9, R14, C0–C1, N1, N4–N6.

**M2: summaries (~2 days)**
- [ ] (a) The `ρ`-seeded silent pass computes `applied[i]` per function-typed or applied param.
  Call-site substitution. Arg-join fallback for callees without a summary. Tests: R2, R10, C2, N2,
  N3.
- [ ] (b) Inferred demands for decls, lambdas and function literals. Fixed point over the call
  graph. Tests: R11–R13, plus mutual recursion converging, plus `TestIFC_TransparentPassthrough_NoError`
  unchanged.
- [ ] Erase `ρ` everywhere it could escape. Assert no `labelVar` appears in any emitted diagnostic
  or in `effectiveBodyLabel`.

**M3: measurement, docs, examples (~1 day)**
- [ ] Corpus delta (see [Breakage risk](#breakage-risk--how-to-measure-it)) attached to the PR.
- [ ] `examples/runnable/secrets/sink_function_value.ail` (expected-fail, R1 + R2 + R13 shapes)
  and its manifest entry. Add the N-rows as a passing companion in the same file or a sibling.
- [ ] `docs/docs/guides/ifc-labels.mdx`: remove "flow through unannotated parameters" from
  Limitations, and add a "Sinks as values" subsection that explains the conservative HOF fallback
  for imported callees until #1771.
- [ ] Prompt: the IFC section's sink rule gets one line saying the rule holds through aliases and
  higher-order functions (via prompt-manager, a new prompt version).
- [ ] Changelog fragment `changelogs/unreleased/<date>-ifc-function-value-sinks.md`
  (`### Fixed`).

### Files to Modify/Create

**New files:**
- `internal/types/ifc_callable.go` (~180 LOC): `ifcCallable`, `ifcDemand`, join, application check.
- `internal/types/ifc_summary_local.go` (~220 LOC): `ρ`-seeded summary pass, applied-at, inferred demands, fixed point.
- `internal/types/ifc_function_value_test.go` (~300 LOC): every R, C and N row as a table test.
- `examples/runnable/secrets/sink_function_value.ail` (~40 LOC): expected-fail example.

**Modified files:**
- `internal/types/ifc_check.go` (+80/−30): `ifcVar.fn`, value resolution, application check in `labelOfCall`, `via` in messages. It is 632 lines today and new logic goes in the new files, staying under the 800 target.
- `internal/types/ifc_static_type.go` (+10): helper exposing a written function type's param annotations to `fromFuncType`. `walkType` itself is unchanged.
- `examples/manifest.json` (+15): new example entry.
- `docs/docs/guides/ifc-labels.mdx` (+30/−5): Limitations and "Sinks as values".
- `changelogs/unreleased/2026-10-XX-ifc-function-value-sinks.md` (new fragment).

## Alternatives Considered

**A. Labels and refinements in HM function types** (`TFunc` params carry `TLabelled` plus a
refinement, and unification checks them). *Rejected for this fix.* It is the principled long-term
answer (M-TAINT-TYPES Phase 3, label polymorphism) but:
(1) unification strips labels by design (M7, `unification_core.go:231-241`), and refinements need
**contravariant subtyping** on function parameters, which AILANG's HM has no notion of;
(2) it puts `TLabelled` into CoreTI and codegen, which have no case for it
(`pipeline_module_compile.go:218-224` comment);
(3) it contradicts the decision #1771's design ratified ("Enforcement stays a surface-AST
analysis — not elaboration", `m-ifc-cross-module-labels.md` §"Enforcement stays…"), so the two
P0 IFC fixes would pull in opposite directions;
(4) the regression surface is every program, not every program that declares a sink.
The callable demand in this design is "labels in the function type" **inside the IFC domain**. If
Phase 3 later moves labels into HM, the callable becomes the projection of that type, and the
acceptance tests here carry over unchanged.

**B. Conservative rejection: sink functions are second-class.** Any non-call use of a function
whose params declare `{not ℓ}` or `<ℓ>` is an error. *Rejected as the fix, kept as the emergency
fallback if F3 cannot be met.* It is about 50 LOC and closes R1–R9, R11 and C1–C2 for named sinks.
But it (i) does not close R12, R13 or R14, where the sink-ness is inferred or anonymous, so the
bypass survives; (ii) rejects N2, a safe and idiomatic program; and (iii) makes `map(send,
cleanBodies)` impossible. It is a policy that trades expressiveness for a guarantee it still does
not deliver.

**C. Inline aliases and HOF bodies (defunctionalise at the call site).** *Rejected.* It does not
terminate on recursive HOFs (`foldl`), cannot see imported bodies, and grows the analysed program
without bound. The applied-at summary is the finite version of the same idea.

**D. Runtime enforcement at sink entry.** *Rejected.* Labels have no runtime representation
("Labels are NOT a runtime check", `ailang prompt` IFC section). Runtime IFC is #1132's second
half and a separate design (`m-trace-label-aware.md`).

**E. Model sinks as effects** (a `Sink[not=email]` capability in the effect row, which already
rides HM function types). *Rejected.* A refinement constrains **data flowing in**, not an action
performed, so an effect row cannot say "argument 1 must not carry `<email>`". The effect fix
(#1326/#573/#1718) is borrowed as a pattern (latent requirement on the value, plus annotation
labels), not as a mechanism.

## Relationship to PR #1771 (M-IFC-CROSS-MODULE)

- **No overlap in scope.** #1771 (M1–M2: iface IFC summary, schema v2, dep-side intrinsic, cache
  and digest) states "Adjacent gap #1719 is out of scope". This design does not touch `internal/iface`,
  `internal/pipeline` cache code, the schema or the digest. It changes only how the caller-side
  walk *resolves and carries* a callee's requirements.
- **Shared representation, not HM types.** Both designs put argument requirements in an
  **IFC-owned summary** (#1771: `types.ImportedIFCSig{Params []IFCParam{Name, Label, Forbidden}, …}`;
  here: `ifcCallable`, built from local `ifcSig`). Neither needs labels in the HM function
  type. Cross-module export does **not** need the HM representation either: it needs the summary
  to carry what M2 infers.
- **Ordering:** **#1719 (this doc) lands first; #1771 rebases onto it.** #1771 is already
  CONFLICTING against dev (triage 2026-10-26) and idle since 10-12. Its caller-side wiring
  (`labelOfCall`: `if imported := c.imports[name] …`) is the code this design replaces with one
  value resolution. If #1771 rebases afterwards, its imported summary becomes one more source
  in `valueOf`'s Identifier case (`ifcCallable.fromImported`). `let g = lib.gate in g(x)` and
  `map(lib.gate, xs)` are then covered for free. If #1771 lands first instead, it ships the
  name-keyed imported branch and inherits this bug for imported sinks until this design moves
  that branch into `valueOf`. That costs an extra rebase but is not a blocker.
- **The one hard coordination point (F4):** inferred demands (M2b) can forbid **several**
  labels on one parameter, but #1771's `IFCParam.Forbidden` is a single `string`. Exported HOFs
  also need `applied`. These are the same kind of problem the authority-scoping design hit with
  `declassify: bool → []string` (`m-ifc-authority-scoping.md` §Relation): ship the list shape in
  v2, or pay a v3 bump. Exporting inferred demands and `applied` is **#1771's M3+ or a follow-up**,
  not this doc. Until then an imported HOF takes the arg-join fallback, which is conservative.
- **Inherited hole, stated explicitly:** the arg-join fallback for an imported HOF cannot see
  data that the imported function sources internally. That is exactly #1134's
  "imported intrinsic is ⊥" hole, and #1771's `Intrinsic` field closes it. This doc does not
  claim cross-module soundness.

## Conflict Surface

Touches `internal/types/` (an IFC pass that runs after typecheck). No parser, lexer, AST, HM,
elaborate, iface, codegen or VM change.

### Semantic positions touched

1. A **top-level function name in value position** (`let g = gate`, `apply1(gate, …)`,
   `{ f: gate }`, `[gate]`): today `⊥` with nothing attached, and it now carries a callable.
2. **Any application whose callee is not a bare local function name** (`g(x)`, `r.f(x)`,
   `getGate()(x)`, `(\y. …)(x)`): today it takes the transparent fallback, and it now also gets
   Checks A/C when the callee carries a callable.
3. **Written function types with labelled or refined params** (`f: string{not email} -> string`,
   `let g: … = …`): they parse and type-check today, the IFC pass ignores the params, and now they
   seed a callable.
4. **Call sites passing a function value to another function:** new applied-at or arg-join check.
5. **Bodies of functions, lambdas and function literals whose params reach a sink:** new
   inferred demands.

### What else lives here

| Position | Existing valid form | Why it is safe |
|----------|--------------------|----------------|
| 1 | Plain function references with no IFC annotations (`map(upper, xs)`, every HOF in std and examples) | `fromSig` of a sig with no `{not}` and no `<ℓ>` param is `nil`. No callable means no new check |
| 1 | A param or let that **shadows** a top-level sink name (N4) | Resolution consults `env` first, as `labelOfCall` does today at `:381`, so the shadowing binding's callable (here `nil`) wins |
| 2 | Calls through closures that capture labelled data (`TestIFCClosureCannotLaunderALabel`) | The result label is computed exactly as today. Checks are only added |
| 3 | Function-typed params with no labels (`f: string -> string`) | `fromFuncType` of an unlabelled type is `nil` |
| 3 | Effect annotations on function types (`(int) -> int ! {IO}`) | Untouched. The effect pass reads them, and the callable reads only param `LabelledType`s |
| 4 | Passing a non-callable value (`nil` fn) to any function | No check |
| 4 | A stored, never-applied sink (N6) | Demands are checked only at application or HOF hand-off, never at reference |
| 5 | Forwarding functions with no sink (`TestIFC_TransparentPassthrough_NoError`: `wrap(x) = x`) | No Check A/C fires on `ρ` inside, so no inferred demand |
| 5 | Unannotated lambda params whose name shadows an outer labelled binding | The non-silent walk keeps today's seeding (outer label retained). `ρ` exists only in the silent summary pass |

### Programs that MUST still work (baseline `ailang check` exit codes measured on `772703831`)

- `examples/runnable/secrets/gated_secret.ail`: 0
- `examples/runnable/secrets/positive_label_call.ail`: 0
- `examples/runnable/secrets/scoped_declassify.ail`: 0
- `examples/runnable/secrets/secret_demo.ail`: 0
- `examples/runnable/secrets/leak_attempt.ail`: **1**, expected-fail and must stay 1 with the same message
- `examples/runnable/contracts/inbox_v2_lib.ail`, `inbox_v2_app.ail`, `inbox_injection_v2.ail`: 0
- `benchmarks/prompt_injection/expected_ailang_safe.ail`: 0
- `examples/runnable/effect_latent_function_values.ail`: 0 (HOF passing named functions, no labels; the sibling-defect fixture)
- Unit tests: `TestIFC_*` (`ifc_check_test.go`), `TestIFCClosureCannotLaunderALabel`,
  `TestIFCAnnotatedLetIsAlsoClosed`, `TestIFCDeclared_McpOauthIdioms`, `TestIFCScoped*`,
  `TestIFCPositive*`, unchanged.

### What deliberately changes

Every program in R1–R14 and C1–C2, and any program of the same shape, is now rejected. The
migration path is the one the error already suggests: declassify before the sink, or annotate a
higher-order parameter with the refinement so the requirement is visible. There is no syntax
change.

## Breakage risk — how to measure it

**Bound by construction.** Every new diagnostic needs a callable demand, and every demand starts at
a written `{not ℓ}` or a positive `<ℓ>` parameter label (or, after #1771, an imported one). A
module that declares neither, and imports no such sink, **cannot** get a new error. Measured
annotation footprint (grep for `{not <ident>}` and for `name: T<label>` in `.ail` files):

| Corpus | `.ail` files | files with `{not ℓ}` | files with labelled params |
|--------|-------------:|---------------------:|---------------------------:|
| this repo (examples, std, benchmarks) | 778 | 8 (2 declare a sink param — `transmit` in `gated_secret`, `leak_attempt`; the rest are comments) | 9 |
| `ailang-packages` | 171 | 2 (`gmail/message.ail` record fields, not params) | 1 |
| `ailang-motoko` | 691 | 8 (mirrors of this repo's examples) | 7 |
| `ailang-demos` (`leak_lab`) | 401 | 7 | 7 |
| `ailang-parse` / `daneel` / `mk-main` / `ailang-world` / `stapledons_voyage` | 133 / 80 / 560 / 602 / 23 | 0 | 0 |

**Measured delta (M3, required in the PR):**
1. Add a corpus test `TestIFCFunctionValueCorpusDelta`, behind an env-gated skip. Put the gate in
   the `internal/config` Registry: new env vars must be registered there (forbidigo). Choose a
   flag-free form if one exists. The test runs `CheckModuleIFC` with the old and new resolution
   over `examples/`, `std/`, `benchmarks/` and any sibling checkouts that are present, and prints
   every diagnostic that appears only in the new run.
2. Run `make verify-examples`, `make test-core`, and focused `go test ./internal/types/ ./internal/repl/ ./internal/pipeline/`.
3. Run `ailang check` over `ailang-demos/leak_lab/examples/*.ail` and `ailang-packages/packages/gmail`.
   Expect no change unless they contain one of the R-shapes. If one does, it is a true leak:
   report it to that repo.
4. Exit criterion: zero new diagnostics in modules that have no annotations, and every other
   new diagnostic triaged in the PR body as a leak (keep the rejection) or as over-approximation.
   An over-approximation needs a human ruling before merge.

## Success Criteria

- [ ] R1–R14 and C1–C2: `CheckModuleIFC` returns exactly the expected kind
  (`sink_refinement` for R*, `param_label_cover` for C*). The message names the callee and the path
  the value took (`via …`).
- [ ] R0 and C0: unchanged (same kind, same position).
- [ ] N1–N6: zero violations.
- [ ] Every row is also a fresh-file CLI `ailang check` (exit codes as in the tables) in a
  cache-cleared temp dir.
- [ ] The REPL gate (`internal/repl/module_registry_load.go:35`) gives the same verdicts. It
  calls the same function, and one test pins this.
- [ ] Mutation check: reverting the application-site check alone makes R1, R3–R9 and C1 pass again.
  Reverting the applied-at logic alone makes R2, R10 and C2 pass again. Reverting inference alone
  makes R11–R13 pass again. Each subset must fail, which shows that each test exercises its
  mechanism.
- [ ] No `labelVar` appears in any diagnostic or in any `effectiveBodyLabel` result (assertion test).
- [ ] Corpus delta attached. `make test-core`, `make lint`, `make check-file-sizes` and
  `make verify-examples` are green.
- [ ] Guide, prompt line, example and changelog fragment updated.

## Testing Strategy

**Unit:** one table test per row (R, C, N), modelled on `TestIFCClosureCannotLaunderALabel`, with
the direct-call control as the first row, so that a broken fixture fails loudly. Algebra tests
for `ifcCallable.join`. A fixed-point test with mutual recursion between two wrappers.

**Integration:** CLI `ailang check` on the example file, plus the REPL-gate parity test.

**Regression surface:** the "Programs that MUST still work" list, pinned with exit codes.

## Deferred Decisions

- Exact field layout of `ifcCallable` and whether `via` is a string or a structured trail. Agent may choose.
- How arity mismatch is joined (two callables of different arity flowing to one binding):
  agent may choose, under one constraint. The result must be at least as strict as either side,
  for example by applying the union of all positions' demands to every argument.
- Whether lambda summaries are memoised by AST node or recomputed per enclosing walk. Agent may
  choose. A memo keyed by `*ast.Lambda` pointer is fine because the pass is per-module.
- Wording of the `via` trail in messages. Agent may choose, but it must name the callee and at least
  the first hop.

## Non-Goals

- **Cross-module** sinks and HOFs (#1134). Owned by #1771 and its follow-ups. This doc provides
  the resolution point they plug into (see the #1771 section).
- Labels and refinements in HM types, CoreTI or the iface. See Alternative A.
- Label-polymorphic *signatures* written by users (`∀ρ`). The `ρ` variables here are internal to
  the summary pass and never surface.
- Refinements on record fields (`gmail/message.ail`'s `text: string{not email}`). That is a
  separate position whose enforcement is not established by this doc.
- Runtime IFC and label-aware tracing (#1132, second half).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Over-approximation rejects safe code (the arg-join fallback, joined callables, conservative record-access callables) | Med | Bounded to modules that declare sinks (see the breakage table). The corpus delta is triaged in the PR. This is the same direction as 1af9f5f30 and #1707: over-rejection is visible, under-rejection is a silent leak |
| `ρ` leaks into a result label or diagnostic | High | Erasure is a single function. An assertion test covers it, and `labelVar` has no other callers today (grep shows only `labels.go:67`, `label_json.go:79`) |
| Fixed point does not converge or is slow | Low | Finite lattice. The loop shape is copied from #1771's `AnalyzeModuleIFC`. A cycle guard already exists |
| Merge conflict with #1771 in `labelOfCall` | Med | Ordering above: this lands first, and #1771 rebases its imported branch into `valueOf` |
| Docs over-promise again | Med | In M3, the prompt's "Identity Bypass Is Structurally Impossible" and the guide's "Why Identity Bypass Doesn't Work" both get a scope note (cross-module is still pending, #1134) |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Static analysis only. Fixed-point iteration in source order, so diagnostics are deterministic |
| A2: Replayability | 0 | No runtime change |
| A3: Effect Legibility | 0 | Effects untouched |
| A4: Explicit Authority | +1 | A sink's requirement can no longer be shed by renaming. Declassify stays the only way to lower a label |
| A5: Bounded Verification | +1 | Per-module summaries. The check stays local to one module plus callee summaries |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Idiomatic agent code (HOFs, aliases) gets the same guarantee as direct calls, and errors name the path the value took |
| A8: Minimal Syntax | +1 | No new syntax. Existing function-type annotations with refinements become meaningful |
| A9: Cost Visibility | 0 | No impact |
| A10: Composability | +1 | Sinks compose with HOFs instead of silently dropping out of the check |
| A11: Structured Failure | 0 | Existing typed error kinds are reused |
| A12: System Boundary | +1 | The sink boundary is enforced however the sink is reached |

**Net Score: +6** → proceed.

### Hard Violation Check
- [x] A1: no nondeterminism introduced.
- [x] A3: no hidden effects.
- [x] A4: no ambient authority. This change removes an implicit relabel.
- [x] A7: does not trade machine analysability for convenience.

## Verification Log

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | R0–R14, C0–C2 and N1–N6 exit codes as tabled | `ailang-dev check` (built from `772703831`) and installed v0.54.18 on fresh files in the scratchpad | Confirmed, identical on both binaries |
| V2 | R1 and R12 execute and print the tainted value | `ailang-dev run --entry main` | Confirmed (`hi`) |
| V3 | Checks A/C run only under `isLocal`; callee resolved by name | Read `ifc_check.go:377-431`, `:492-497` | Confirmed |
| V4 | A top-level function in value position evaluates to `⊥` with nothing attached | Read `ifc_check.go:186-190` (Identifier → env only); `c.sigs` is never consulted outside `labelOfCall` and `staticType`'s FuncCall case (`ifc_static_type.go:219-229`) | Confirmed |
| V5 | `ifcVar` has no callable component (negative existence) | Read `ifc_check.go:105-108`; `grep -rn -i "ifcCallable\|appliedAt\|ifcDemand" internal` gives no hits | Confirmed absent |
| V6 | Written function types ignore param labels in IFC | Read `ifc_static_type.go:129-130` | Confirmed (return only) |
| V7 | HM strips labels and refinements | Read `typechecker.go:288-292`, `elaborate/file_funcs.go:440-442`, `unification_core.go:231-241` | Confirmed |
| V8 | iface drops refinements | Read `iface/builder.go:119-127` | Confirmed |
| V9 | `labelVar` exists and is unused by the IFC pass (negative existence) | `grep -rn "LabelVar(\|labelVar{" internal` gives `labels.go:67` and `label_json.go:79` only | Confirmed |
| V10 | No other sink-enforcement path exists | `CheckSinkRefinement(` has no non-test caller outside `sink_check.go`. `CheckSinkLabel(` is called only from `ifc_check.go:410` | Confirmed |
| V11 | No new error kind is needed | `errors.go:58-61` defines `sink_refinement` and `param_label_cover`, which are reused | Confirmed |
| V12 | Proposed new files and example path are free | `ls internal/types/ifc_callable.go examples/runnable/secrets/sink_function_value.ail` gives "No such file" for both | Confirmed |
| V13 | #1771 scope excludes #1719 and its imported branch is name-keyed | `gh pr view 1771` body ("Adjacent gap #1719 is out of scope"); diff of `ifc_check.go` (`c.imports[name]` in `labelOfCall`); `ImportedIFCSig`/`IFCParam{Forbidden string}` in `ifc_summary.go` | Confirmed |
| V14 | #1771 is CONFLICTING and idle | PR comment 2026-10-26 (triage: NEEDS-REBASE, CONFLICTING/DIRTY); last code push 2026-10-12 | Confirmed |
| V15 | Regression fixtures exist with the stated exit codes | `ailang-dev check` on each listed file | Confirmed |
| V16 | Breakage footprint table | `grep -rl --include='*.ail' -E "\{not [a-z_]+\}"` and `": *[A-Za-z]+<[a-z_]+>"` per repo | Confirmed (counts as tabled) |
| V17 | Same defect class was fixed for effects | `design_docs/planned/v0_48_0/m-effect-latent-function-values.md`; commit `3d4f49720` (#1708) | Confirmed |
| V18 | REPL uses the same IFC entry point | `internal/repl/module_registry_load.go:35` calls `types.CheckModuleIFC` | Confirmed |

**Quorum:** skipped at the requester's instruction. Note that trigger 1 (design-freeze items
F1–F4) **fires**, so without a quorum a human must ratify F1–F4 before sprint planning.

## Related Documents

- `design_docs/planned/m-ifc-cross-module-labels.md`: #1134 / PR #1771, the companion. See the relationship section.
- `design_docs/implemented/v0_53_0/m-ifc-authority-scoping.md`: Check C's definition. Its "local callees" scoping is what C1 and C2 bypass.
- `design_docs/planned/v0_48_0/m-effect-latent-function-values.md`: same defect class for effects (#1326, #573), the precedent followed here.
- `design_docs/implemented/v0_26_0/m-secret-effect-remote-approval.md`: M5, the original `CheckModuleIFC` and its Phase 2 deferral note.
- `docs/docs/guides/ifc-labels.mdx`: user-facing semantics. Limitations is updated in M3.
- Issues: #1719 (this), #1134, #1132, #1718 (effect callbacks in containers, the sibling shape to R6/R7), #752 and #1707 (Check C).

## Future Work

- Export inferred demands and `applied` through the iface summary (#1771 follow-up, gated on F4).
- Label-polymorphic signatures in HM (M-TAINT-TYPES Phase 3), which would subsume `ifcCallable`.
- Sink refinements on record fields.

---

**Document created**: 2026-10-26
**Last updated**: 2026-10-26
