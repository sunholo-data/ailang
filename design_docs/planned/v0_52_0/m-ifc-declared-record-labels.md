# M-IFC-DECLARED-RECORD-LABELS — labels written inside a type are label sources

**Status**: Planned. Quorum round 0 BLOCKED (annotated-let precision; unverified call rule / r3 mechanism) → fixed (hand-off rule, V10). Round 1 BLOCKED on premise-verification only (String() fidelity, V12, pipeline order) → closed by V12–V14 without a third round (re-quorum guardrail spent).
**Target**: v0.52.0
**Priority**: P0 (security: silent loss of an IFC guarantee)
**Estimated**: 1 day
**Issues**: #1523 (primary), #1527 (duplicate, filed from the prod inbox message `inbox_1790965638045_48309b44`)
**Triage**: `design_docs/planned/ailang-core-triage/ifc-declared-record-field-label-loss.md`
**Dependencies**: none. M-SECRET-EFFECT M5 (`internal/types/ifc_check.go`) shipped; closure-laundering fix (1af9f5f30) shipped.

**Quorum triggers fired**: none of the four mechanical triggers strictly (no freeze items, no
shared machinery overridden, no cost/schema surface, all premises in-repo). Run anyway because the
change is a **security change to the type checker** (attended instruction).

---

## Problem Statement

`ailang check` accepts this (v0.51.0-29, issue #1523):

```ailang
export type RawCode = { raw: string<authcode> }
export func mintRaw() -> RawCode ! {Rand[mode=crypto]} = { raw: uuid4() }
pure func logLine(s: string{not authcode}) -> string = "log: ${s}"
export func leak() -> string ! {Rand[mode=crypto]} {
  let c = mintRaw();
  logLine(c.raw)          -- <authcode> reaches a {not authcode} sink: NOT reported
}
```

The inline-record form `let r = {raw: asToken(x)}; logLine(r.raw)` is rejected.

### Root cause (read from the code, V1–V3)

`CheckModuleIFC` is a value-flow analysis over the surface AST. Its only label **sources** are:

1. a *top-level* `<ℓ>` on a parameter type (`buildIFCSig`: `p.Type.(*ast.LabelledType)`),
2. a *top-level* `<ℓ>` on a return type (same, `fn.ReturnType`),
3. `secret()` / `_secret_read` (`ifcBuiltinSourceLabels`).

A label written **anywhere else inside a type** is never read: a record field, the body of a
declared `type`, a list / tuple element, an ADT constructor field, an alias
`type Code = string<authcode>`, a `let` annotation, or a lambda parameter annotation.
`RecordAccess` returns the record's flow label (`labelOf(e.Record)`), which is ⊥ when the record was
built from unlabelled data, so `c.raw` is ⊥. The inline-record case "works" only because the
label rode in on the *value* (`asToken(x)` has a top-level return label).

So this is not a field-access bug. It is one missing rule — **a label in a type annotation is a
source for every value of that type** — and it has 17 measured manifestations (V4), all accepted
today:

| # | Shape | Path |
|---|---|---|
| p1 | `c: RawCode`, `logLine(c.raw)` | declared record, param |
| p2 | `let c = mk(x); logLine(c.raw)` (`mk -> RawCode`) | declared record, call result (the #1523 repro) |
| p3 | `s: Session` (`code: RawCode`), `s.code.raw` | nested declared record |
| p4 | `mkS(u).code.raw` | nested, call result |
| p5 | `match c { {raw} => logLine(raw) }` | record destructuring |
| p6 | `match s { {code: {raw: r}, ...} => logLine(r) }` | nested destructuring |
| p7 | `c: {raw: string<authcode>}` | inline record **type annotation** |
| p8 | `t: Tok` (`type Tok = Tok(string<authcode>) \| NoTok`), `match t { Tok(s) => logLine(s) }` | ADT constructor field |
| p9 | `match Tok(x) { Tok(s) => logLine(s) }` | ADT constructor application |
| p10 | `p: Pair` (`type Pair = (string<authcode>, int)`), `match p { (s, _) => … }` | tuple alias |
| p11 | `xs: [string<authcode>]`, `match xs { h :: _ => logLine(h) }` | list element |
| p12 | `let c: RawCode = {raw: x}; logLine(c.raw)` | let annotation |
| p13 | `let f = func(c: RawCode) -> string { logLine(c.raw) }` | lambda parameter annotation |
| r1 | `c: Code` (`type Code = string<authcode>`), `logLine(c)` | alias of a labelled type |
| r2 | `let t: string<authcode> = plain(x); logLine(t)` | top-level label on a let (V13 of m-mcp-oauth) |
| r3 | `logLine(getRaw(mk(x)))`, `getRaw(c: RawCode) -> string = c.raw` | getter launders the field |
| r4 | `cs: [RawCode]`, `match cs { c :: _ => logLine(c.raw) }` | list of declared records |

Fixtures: `probe.ail` (p1–p13), `probe2.ail` (r1–r4); all → `✓ No errors found!` on `a59bbdbeb`.

**Impact:** any secret held in a declared type, a collection, or an annotated binding is
invisible to the checker. `sunholo/mcp_oauth` (T7) currently designs around it with a CI lint.

## Goals

**Primary goal:** every `<ℓ>` written inside a type annotation that the module's IFC pass can see
is a source label for the values of that type, so all 17 shapes above are rejected with the
existing `information-flow violation` error — while field projection stays **field-precise**
(an unlabelled sibling field of a labelled record stays clean).

**Success metrics:**
- p1–p13 and r1–r4 rejected, each by a named Go test.
- Field precision: `s.user` (unlabelled sibling of `s.code`) passes a `{not authcode}` sink for
  param, call-result, unannotated-let and **annotated-let** `s`, and through a getter
  `getUser(s)` (positive tests; all five compile today, V9/V11).
- No existing program changes verdict: `make verify-examples`, `ailang verify
  examples/runnable/contracts/inbox_injection_v2.ail` output unchanged, existing `ifc_*_test.go`
  green.

## Solution Design

### The rule: flow label ⊔ declared label

Every expression gets two things:

- **flow label** — what the existing walk computes: labels carried in by values (unchanged).
- **static type** — an `ast.Type` the pass can name *without* the HM checker, or `nil`:
  - `Identifier` → the type bound in the IFC env (function param annotation, lambda/func-literal
    param annotation, `let` annotation, or the static type of the let value);
  - `FuncCall` to a module-local function → its declared return type;
  - `FuncCall` to a constructor of a module-local ADT → that type's name;
  - `r.f` → the type of field `f` in `staticType(r)` (resolving module-local declared names,
    aliases, and `LabelledType` wrappers);
  - anything else → `nil`.

The **declared label** of a type, `deep(T)`, is the join of every `<ℓ>` appearing anywhere in `T`,
resolving module-local type names (memoised, cycle-guarded): `LabelledType` → ℓ ⊔ deep(base);
record → ⊔ fields; list/array/tuple/type-app → ⊔ elements/args (and the constructor's body if it
is module-local); ADT → ⊔ constructor fields; function type → deep(return) only (a function value
carries what it returns, as the Lambda case already rules).

`labelOf(e) = flowOf(e) ⊔ deep(staticType(e))`, used at every consumer (sink Check A, Check B,
joins, match scrutinee) — exactly where `labelOf` is used today.

**Field precision** comes from one case: when `staticType(r.f)` is known,
`flowOf(r.f) = flowOf(r)` — the record's *dynamic* label only — and the declared part comes from
`deep(type of f)`, not `deep(type of r)`. When it is not known, `flowOf(r.f) = labelOf(r)` (the
whole record's full label — today's conservative rule, unchanged).

### Hand-off: one rule for every place a value moves into a typed position

A value moves into a position with a declared type `T` in four places: a `let` (T = the
annotation, or the value's own static type when unannotated), a call argument (T = the callee's
parameter annotation), a function body (T = the declared return type, used when computing the
callee summary), and a field of a record literal handed to one of those. The binding / summary
stores the **hand-off label** `handOff(e, T)`:

- `T` unknown (`nil`) → `labelOf(e)` (full — today's behaviour);
- `staticType(e)` is **syntactically the same type** as `T` (`String()` equality of the two
  `ast.Type`s) → `flowOf(e)`. The receiving position re-derives the declared part from `T` itself
  (via `deep` on use or field projection), so nothing is lost;
- `e` is a record literal and `T` resolves to a record type → ⊔ over fields
  `handOff(value_i, fieldType(T, name_i))` (literal fields not in `T` → their full label);
- `e` is a block / `let … in` → hand off its result expression (bindings threaded as usual);
  `if` / `match` → ⊔ of the arms' hand-offs;
- otherwise → `labelOf(e)` (full).

The type-equality guard is what keeps an annotation from **lowering** a label: HM strips labels in
unification (V8), so `let s: {user: string, code: string<x>} = v` type-checks even when `v`'s
static type puts `<x>` on `user`. Different `String()` → full label stored → sound.

This answers the precision cases end to end (each is a named test):

| Program | Without hand-off | With hand-off |
|---|---|---|
| `let s: Session = mkS(u); logLine(s.user)` | `<authcode>` (annotated let stored full) | ⊥ — `Session` = `Session` |
| `logLine(mkS(u).user)`, `mkS(u) -> Session = {user: u, code: mk(u)}` | `<authcode>` (summary of a record literal joins its fields' full labels) | ⊥ — summary is hand-off of the literal against `Session`: `user` ⊥, `code`'s `<authcode>` is accounted for by `fieldType(Session, code)` |
| `getUser(s: Session) -> string = s.user`; `logLine(getUser(s))` | `<authcode>` (argument `s` joined with its full label, APP-PURE) | ⊥ — argument handed off against the param type `Session` |
| `getRaw(c: RawCode) -> string = c.raw`; `logLine(getRaw(mk(x)))` (r3) | — | **rejected** — see below |

**Getters (r3) and the call rule.** The existing call rule (V10, read from `labelOfCall` /
`calleeResultLabel`) is: a local callee with no explicit return label yields
`effectiveBodyLabel(callee) ⊔ ⊔args` (APP-PURE), where `effectiveBodyLabel` walks the body with
every parameter's flow label seeded to ⊥. This fix keeps that rule and changes two inputs:
(1) arguments contribute their **hand-off** label against the parameter's annotation instead of
their full label; (2) `effectiveBodyLabel` seeds each parameter's **type** (from its annotation)
alongside the ⊥ flow label. Seeding the *type* is not Option A (seeding `deep(T)` as a flow label):
inside `getRaw`, `c.raw` = ⊥ ⊔ `deep(string<authcode>)` = `<authcode>`, so the summary is
`<authcode>` and r3 is rejected; inside `getUser`, `s.user` = ⊥ ⊔ `deep(string)` = ⊥, so its
summary is ⊥. Without the type seed, `c` in the summary walk has no type and `c.raw` is ⊥ — the
seed is what closes r3, and it is field-precise for the same reason projection is.
A body that returns or forwards the whole parameter (`= c`, `= f(c)` with `f` unknown) gets
`deep(RawCode)` via `labelOf(c)` — sound.

### Why this shape (alternatives)

| Option | Verdict |
|---|---|
| A. Seed every param/let with `deep(T)` as its flow label | Rejected: `s.user` would carry `<authcode>` from its sibling `s.code` — false positives on every record mixing a secret and metadata (the common case: a session). |
| B. Read HM types (CoreTI) for each surface node | Rejected for this fix: `TLabelled` is stripped in unification (`unification_core.go` M7) and the pass is deliberately surface-only (ifc_check.go header). Bigger blast radius than the hole. |
| C. Field-name heuristic (`.raw` is labelled if any declared type has a labelled `raw`) | Rejected: unsound across same-named fields and imprecise. |
| **D. flow ⊔ declared, field-precise projection (above)** | Chosen: additive, local to `ifc_check.go`, needs no new AST or type-checker state. |

### Monotonicity (why nothing that is rejected today becomes accepted)

By induction on the walk: `deep(·) ⊇ ⊥`, and every `flowOf` case is the old `labelOf` case with
sub-expressions' labels replaced by labels that are ⊇ the old ones: an Identifier's stored label
is `handOff(value, T)`, and `handOff(e, T) ⊇ flowOf(e) ⊇ old labelOf(e)` (each hand-off branch
returns either `flowOf`, `labelOf`, or a join of field hand-offs whose old counterpart was the
join of the fields' old labels); `flowOf(r.f) = flowOf(r) ⊇ old labelOf(r)`; a call's result is
`effectiveBodyLabel ⊔ ⊔ handOff(arg_i, P_i)` and the old one was the same with old (⊆) labels —
the call rule itself (V10) is unchanged, only its inputs grow. So
`new labelOf(e) ⊇ old labelOf(e)` for every `e`. The
change only **adds** rejections. Destructuring (`match`) keeps today's rule (every pattern
variable gets the scrutinee's full label), which is now non-⊥ for declared types — conservative,
not field-precise; recorded as a follow-up precision item, not a security gap.

### Scope boundary

- **Module-local types only.** A type imported from another module is not resolvable by this
  single-module pass (`deep` of an unknown name is ⊥ — today's behaviour). That is the
  cross-module hole tracked by `design_docs/planned/m-ifc-cross-module-labels.md`, explicitly not
  fixed here and not regressed (its fixture `inbox_v2_app.ail` must keep its current verdict).
- **Generic declared types** (`type Box[a] = {v: a}`): `staticType(b.v)` is `nil` (a field typed by
  a type parameter is not resolved), so `b.v` falls back to `labelOf(b)` = flow ⊔ deep(`Box[string<s>]`)
  ⊇ `<s>` — sound, not precise.

## Conflict Surface

**Files touched:** `internal/types/ifc_check.go` (and a new `internal/types/ifc_static_type.go`
for `staticType`/`deep`/`fieldType`, to keep `ifc_check.go` under the size target), tests in
`internal/types/`. No change to `internal/parser`, `internal/ast`, `internal/elaborate`,
`internal/iface`, the HM checker, codegen or runtime.

1. **Positions extended:** `RecordAccess`, `Identifier`, `FuncCall` (result), `Let`/`Block`
   bindings, `Lambda`/`FuncLit` params, `FuncDecl` params in the IFC walk only.
2. **Other constructs in those positions:** record access on a *tagged-union* receiver
   (already rejected earlier by `VerifyTaggedUnionFieldAccesses`); a receiver not bound in the IFC env
   (e.g. a top-level function or an imported name) → staticType `nil` →
   unchanged; `Let.Type` placeholder `SimpleType{"unknown"}` (parser_expr.go:142, error recovery)
   → not a declared name → deep ⊥; unannotated lambda params keep today's behaviour (not rebound,
   so an outer binding of the same name still taints — over-approximation retained).
3. **Disambiguation:** none needed — no syntax changes; every new label comes from an annotation
   the parser already produces.
4. **Programs that MUST still work (same verdict):**
   - `examples/runnable/contracts/inbox_injection_v2.ail` (`ailang verify`: same per-function results)
   - `examples/runnable/contracts/inbox_v2_lib.ail` + `inbox_v2_app.ail` (declares `Mail = {body: string<email>, …}`; no `{not}` sinks, Z3-only violations)
   - `examples/runnable/secrets/secret_demo.ail`, `gated_secret.ail` (must check clean), `leak_attempt.ail` (must still be rejected)
   - `examples/runnable/contracts/interpolation_verify.ail`
   - `~/dev/sunholo-data/docparse/docparse_api/services/api_keys.ail` IFC pattern (top-level `<apikey>` return, `{not apikey}` sink, `Declassify` hasher — only top-level labels, so unaffected by construction; re-checked with the new binary)
   - all existing `internal/types/ifc_check_test.go`, `ifc_closure_test.go`, `inference_label_test.go`
5. **Intentional incompatibilities:** programs that pass a value whose *declared* type carries
   `<ℓ>` into a `{not ℓ}` sink are now rejected (that is the fix). A function with an explicit
   return label whose body projects a declared-labelled field it does not cover now gets the
   existing Check-B `Declassify` error. `let x: T<ℓ> = e` now attaches `<ℓ>` (closes m-mcp-oauth
   V13 — the classifier function is no longer the only way to label).

## Implementation Plan

1. **Tests first** (`internal/types/ifc_declared_test.go`): one negative test per row p1–p13,
   r1–r4; positive precision tests; inline-record regression.
2. `ifc_static_type.go`: `typeDecls` index from `file.Decls`, constructor→type index,
   `deepLabel(ast.Type)`, `fieldType(ast.Type, field)`, `staticType(expr, env)`.
3. `ifc_check.go`: env value becomes `{label, typ}`; `labelOf = flowOf ⊔ deep(staticType)`;
   `handOff(e, T)` used by Let/Block bindings, call arguments and the callee summary;
   RecordAccess/Lambda/FuncLit/params per the rules above; `effectiveBodyLabel` seeds
   param **types** (so getters like r3 carry the field label).
4. Mutation test: revert the RecordAccess/deep join on a scratch copy; the mutant must compile and
   the new tests must fail.
5. Docs: `docs/docs/guides/ifc-labels.mdx` (labels in declared types, annotations; precision of
   destructuring); changelog fragment.

## Axiom Compliance

| Axiom | Score | Note |
|---|---|---|
| A1 determinism | 0 | pure static analysis, deterministic walk order unchanged |
| A3 explicit effects | 0 | no effect change |
| A4 machine decidability | +1 | label of a value is now decidable from its declared type, as a reader expects |
| A7 semantic transparency | +2 | a label written in a type now means what it says; removes a silent hole |
| Safety (fail loudly) | +1 | silent acceptance → loud rejection |
Net +4, no hard violations.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | IFC sources are only top-level param/return labels + `secret()` | read `buildIFCSig`, `ifcBuiltinSourceLabels`, `labelOf` in `internal/types/ifc_check.go` @ a59bbdbeb | Confirmed |
| V2 | `RecordAccess` returns the record's label only | `ifc_check.go` `case *ast.RecordAccess: return c.labelOf(e.Record, env)` | Confirmed |
| V3 | Only callers of `CheckModuleIFC` are the module pipeline and the REPL loader | `grep -rn "CheckModuleIFC(" internal cmd` → `pipeline_module_compile.go:223`, `repl/module_registry_load.go:35` | Confirmed |
| V4 | All 17 shapes accepted today; inline record rejected | `ailang check` (repo build of a59bbdbeb) on `probe.ail`, `probe2.ail` → `✓ No errors found!`; `inline.ail` → `information-flow violation` | Confirmed |
| V5 | Type decls are in `file.Decls` as `*ast.TypeDecl`; record def is `*ast.RecordType`, tuple/list/ident alias is `*ast.TypeAlias`, ADT is `*ast.AlgebraicType` | `internal/parser/parser_type_decl.go` (`parseTypeDeclBody`), `internal/parser/parser_file.go` (`file.Decls = append(file.Decls, decl)` for TYPE) | Confirmed |
| V6 | `[T]` parses as `TypeApp{"list", [T]}`; uppercase names are `SimpleType`, lowercase non-builtin are `TypeVar` | `internal/parser/parser_type.go` | Confirmed |
| V7 | No repo `.ail` declares a labelled field type with a `{not}` sink in the same module | `grep` over `examples/ std/` for `string<…>` in types: only `inbox_v2_lib.ail` (`Mail`, no sinks) | Confirmed |
| V8 | HM unification strips `TLabelled` (why option B is out) | quoted in `m-ifc-cross-module-labels.md` from `unification_core.go` M7 | Cited |
| V9 | Positive shapes (`s.user` for param / call / let) compile today | `ailang check pos.ail` → `✓ No errors found!` | Confirmed |
| V10 | Existing call rule: local callee without explicit return label → `effectiveBodyLabel(name) ⊔ joinLabels(argLabels)`; explicit return label or `Declassify` → `sig.returnLabel`; unknown callee → `labelOf(call.Func) ⊔ joinLabels(argLabels)`; `effectiveBodyLabel` seeds params with `LabelBottom()` and no type | read `labelOfCall`, `calleeResultLabel`, `effectiveBodyLabel` in `ifc_check.go` @ a59bbdbeb (`return LabelJoin(c.effectiveBodyLabel(sig.decl.Name), joinLabels(argLabels))`; `env[p.name] = LabelBottom()`) | Confirmed |
| V11 | Annotated-let and getter positive shapes compile today | `ailang check` on `let s: Session = mkS(u); logLine(s.user)` and `getUser(s: Session) -> string = s.user; logLine(getUser(s))` → clean | Confirmed (re-run at sprint start as a test) |
| V12 | HM accepts a binding whose annotation's labels differ from the value's (why hand-off needs the type-equality guard) | `ailang check` on `type A = {user: string<authcode>, code: string}`; `let s: {user: string, code: string<authcode>} = mkA(u); s.user` → `✓ No errors found!` (fixture `v12.ail`) | Confirmed |
| V13 | `ast.Type.String()` is label-faithful and positional: `(*LabelledType).String()` renders `base<ℓ>` / `base{not ℓ}`; `RecordType`, `TupleType`, `TypeApp`, `FuncType`, `ListType` render each component with its own `String()`; a declared name renders as the name (`SimpleType`), so an alias never equals its expansion | read `internal/ast/ast_type.go` and `RecordType.String` in `internal/ast/ast_decl.go` @ a59bbdbeb | Confirmed. Consequence: alias-vs-expansion hand-off takes the **full-label** branch — sound, imprecise (over-taints, never under-taints) |
| V14 | `VerifyTaggedUnionFieldAccesses` exists and runs before `CheckModuleIFC` in module compile | `internal/types/tagged_union_predicate.go:133`; `internal/pipeline/pipeline_module_compile.go` calls it immediately before `types.CheckModuleIFC(unit.Surface)` | Confirmed |

## Success Criteria

- [ ] Negative tests for p1–p13, r1–r4 fail before the fix and pass after
- [ ] Positive precision tests pass before and after
- [ ] `ailang verify examples/runnable/contracts/inbox_injection_v2.ail` output identical before/after
- [ ] `make verify-examples`, `make test-core`, scoped `go test` green
- [ ] Mutation test: mutant compiles, ≥1 new test fails
- [ ] `ifc-labels.mdx` and changelog fragment updated

## Follow-ups (not in this doc)

- Field-precise destructuring (`match` on a declared record binds each var to its field's label).
- Imported declared types — covered by `m-ifc-cross-module-labels.md`.

## Related Documents

- `design_docs/implemented/v0_16_0/m-taint-types.md` — label lattice
- `design_docs/planned/m-ifc-cross-module-labels.md` — the module-edge hole (out of scope)
- `design_docs/planned/v0_52_0/m-mcp-oauth-package.md` — T7, V13–V16 (consumer of this fix)
- `docs/docs/guides/ifc-labels.mdx`
