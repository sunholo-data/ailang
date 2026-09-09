# M-SMT-INTERP-SHOW: Type-directed `show` normalization — unblock Z3 verification of string-building functions

**Status**: Approved — freeze items ratified 2026-09-09, sprint in progress
**Target**: v0.36.0
**Priority**: P0 (the contract/IFC story's load-bearing case — string construction — cannot be proved today)
**Estimated**: 2–2.5 days (3 milestones, each independently committable; M1 alone closes the report)
**Dependencies**: None. Independent of, and complementary to,
[m-contract-verification-coverage](../m-contract-verification-coverage.md) (which
re-*categorizes* skips; this doc *removes* a class of them).
**Author**: design-doc-creator role, attended session 2026-09-09, at `dev` = `8e8e8c9bf` (v0.35.4)
**Revision**: r3 — two quorum rounds, both **blocked**, four objections in total, all four
accepted and fixed in full, none argued. r2 answered Q1 (silent fallback on a `CoreTypeInfo`
miss) and Q2 (cross-module ref in the `int` rewrite); r3 answers Q3 (M3 claimed a type and an
origin the diagnostic layer cannot derive) and Q4 (two node-construction bugs that would have
failed `ValidateCoreTypeInfo`). Per the design-doc-creator re-quorum-ONCE guardrail, a third
round was **not** spent: see [Quorum verification log](#quorum-verification-log) for the
handover state. The architecture was not challenged in either round — all four objections were
specification-precision defects, and each is now measured or spelled out at the node level.
**Source**: public feedback `fb_913ee851c83c0d8c` (mcp-public → `public-feedback`, 2026-09-08 20:13Z),
filed against v0.35.3-7-ga097d7718-dirty, Z3 4.15.4, darwin/arm64.

---

## Problem Statement

`ailang verify` skips any function whose body builds a string with `"${...}"`
interpolation, reporting `uses an unencodable builtin: show` — for a function whose
source contains no `show` call. Since `++` is list-only, interpolation is *the* way to
build a string in AILANG, so **no string-building function can carry a Z3-verified
contract.**

### The minimal pair (reproduced first-party at v0.35.4)

Two functions, character-identical contracts, differing only in whether the body
interpolates:

```ailang
module interp
import std/string (contains)

export pure func noInterp(a: string) -> string
requires { not(contains(a, "\n")) } ensures { contains(result, "X") }
{ "X" }                     -- ✓ VERIFIED (64ms)

export pure func withInterp(a: string) -> string
requires { not(contains(a, "\n")) } ensures { contains(result, "X") }
{ "X${a}" }                 -- ⚠ SKIPPED: uses an unencodable builtin: show
```

### Root cause — a spurious node inserted by the desugar

[`internal/parser/parser_literals.go:112`](../../../internal/parser/parser_literals.go#L112)
desugars each interpolation hole to `show(<expr>)` and folds the segments into a
`concat_String` chain. This happens **at parse time, before any type is known**, so the
`show` wrapper is inserted unconditionally — including around holes that are already
`string`, where `show` is the identity.

`show` is a genuine `$builtin` of type `∀α. α -> string`
([`internal/builtins/show.go:55`](../../../internal/builtins/show.go#L55)), not a
type-class method resolved at elaboration. Nothing downstream monomorphizes or removes it,
so the Core the verifier sees is `App($builtin.show, [a])`, and
[`internal/smt/encodable.go:715`](../../../internal/smt/encodable.go#L715) rejects every
`$builtin` name without an SMT mapping.

**The in-source comment at
[`parser_literals.go:103-105`](../../../internal/parser/parser_literals.go#L103) is wrong**
and has been misleading readers: it claims the `Show` class "dispatches to the right
instance at elaboration time (`show_Int`, `show_String≡id`, etc.)". No such dispatch
happens. `show_Int`/`show_String`/`show_Bool`/`show_Float` appear *only* in the frozen
interface table
([`internal/iface/builtin_freeze.go:81-84`](../../../internal/iface/builtin_freeze.go#L81));
`ailang builtins list` registers exactly one `show`, and it is the polymorphic one.

### The proof was always available — the desugar hides it

`concat_String` is **already** encoded as SMT-LIB `str.++`
([`internal/smt/types.go:303`](../../../internal/smt/types.go#L303)). Writing the same
functions with explicit `concat_String` verifies today, including the reporter's archetypal
RFC 5322 case (measured, §Verification Log V4):

```ailang
export pure func hdr(to: string, subject: string) -> string
requires { not(contains(to, "\r")), not(contains(to, "\n")) }
ensures  { contains(result, "\r\n\r\n") }
{ concat_String(concat_String(concat_String("To: ", to),
    concat_String("\r\nSubject: ", subject)), "\r\n\r\n") }
-- ✓ VERIFIED (38ms)
```

So the verifier is not missing a theory. The encoder handles every part of this already;
a node the parser inserted for a job it does not need to do is standing in front of it.

### Impact

**On the security story this language is pitched at.** String construction is exactly where
the injection classes live — header injection (a CR/LF in a `to` or `subject` splits an
RFC 5322 header block), SQL injection (M-TAINT-TYPES' own problem statement), prompt
injection (the Guardians scenario). All three are properties of a *constructed* string. The
`requires` guard is expressible and reads well; the act of building the string is what
disqualifies the function from proof.

**On IFC.** Label violations are today caught by Z3 through `ensures` contracts rather than
by the typechecker. If contracts cannot cover functions that build strings, the current
enforcement route has a hole exactly where the taint story is most load-bearing. (An
argument *for* `fb_a71eab12139bee26`'s Phase 2 type-level enforcement, not against it.)

**On our own examples.** Measured across `examples/runnable/contracts/*.ail` (35 files):
of 6 skips, **2 are this bug** — and both are pure string-hole cases that M1 fixes:

| File | Function | Body |
|---|---|---|
| `string_verify.ail:10` | `safeConcat` | `"${a}${b}"` |
| `showcase.ail:68` | `prefixedLength` | `strLength("${prefix}${body}")` |

The language's own contract showcase cannot prove its own string examples.

**On the KPI.** `verify_skipped` feeds `isVerifiedSuccess`
(`internal/observatory/cost_per_verified_success.go`); every one of these skips disqualifies
a run from the headline cost-per-verified-success metric.

---

## Goals

**Primary goal**: make `show` disappear from Core wherever its argument's type makes it
redundant or trivially encodable, so that string-building functions verify without the
author changing a character of source.

**Success metrics:**

1. `withInterp` above **verifies** (not skips) — the reporter's minimal pair goes green.
2. `safeConcat` and `prefixedLength` verify; the `show` skip count across
   `examples/runnable/contracts/` goes **2 → 0**.
3. A `"${s}"`, `"${b}"` (bool) and `"${n}"` (int) hole all verify; `${f}` (float) still
   skips, with a message that names the measured argument **type** (never the origin — see Q3).
4. `ailang fmt` round-trips every interpolation form byte-identically (M1 must not touch
   the surface AST — see Conflict Surface).
5. Runtime output of every existing program is byte-identical (`make verify-examples`).

---

## Solution Design

### Overview

One **type-directed Core→Core pass**, `ShowNormalizer`, run in the compile pipeline where
`CoreTypeInfo` is available. For each `App($builtin.show, [x])` it looks up `x`'s type and
rewrites into Core the SMT encoder *already* accepts:

| Type of `x` | Rewrite | Encodable because | Verifier change needed |
|---|---|---|---|
| `string` | `x` (elide) | `show` on a string is the identity (V1) | **none** |
| `bool` | `if x then "true" else "false"` | `core.If` + string literals already encode (V5) | **none** |
| `int` | `$builtin._string_intToStr(x)` | one new SMT mapping (M2) | 1 map entry |
| `float` | unchanged | Z3 has no Real→String | — (skips, better message) |
| anything else / type var | unchanged | no encoding exists | — (skips, better message) |

Two of the three wins need **zero changes to the verifier**. Only `int` requires an SMT-layer
addition, and it is a two-line table entry.

This is deliberately not an SMT-local fix. `IsSMTEncodable(funcName, meta, body)` receives no
type information, and Core nodes carry no inline types (types live in a side table,
`types.CoreTypeInfo`, keyed by NodeID — V6). Making the verifier type-aware would mean
threading a sort oracle through `encodable.go`, `codegen.go` and the callee resolver. Doing
the type-directed decision **once, where the types already are**, keeps the SMT layer untyped
and gives the runtime the same simplification for free.

### Architecture

**Prior art to follow exactly**: `DebugEraser`
([`internal/pipeline/debug_erasure.go`](../../../internal/pipeline/debug_erasure.go), 252 LOC)
is the same shape — a Core→Core walk that removes builtin calls, returns a new `*core.Program`,
preserves `Meta`, and mints no node IDs. `ShowNormalizer` reuses that structure; it differs only
in consulting `CoreTypeInfo` and in the `bool`/`int` cases, which construct replacement nodes.

**Placement**: immediately after monomorphization in both pipeline paths —
[`pipeline_module_compile.go:353-362`](../../../internal/pipeline/pipeline_module_compile.go#L353)
and [`pipeline_single.go:432-435`](../../../internal/pipeline/pipeline_single.go#L432) —
and unconditionally, so the `DisableMonomorphization` path still gets it (catching fewer holes,
since a polymorphic `${x}` only resolves to `string` after specialization).

The specializer maintains `CoreTypeInfo` for cloned nodes
([`specialize_clone.go:28-29`](../../../internal/pipeline/specialize_clone.go#L28) —
`s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))`), so type lookups are valid
post-specialization. **Running after Specialize is what makes the pass see a generic
helper's `string` instantiation at all.**

**Fail-loud, per CLAUDE.md Principle 2 — two cases that must not be conflated** (revised in
r2; see [Quorum verification log](#quorum-verification-log) Q1):

| What the lookup returns | Meaning | Pass behavior |
|---|---|---|
| A type outside the rewrite rows — a `TVar`, a record, an ADT, `float` | **Legitimate.** The hole is genuinely polymorphic or genuinely unencodable | Leave `show` unchanged. This is the honest residue M3 reports on |
| **No entry at all** for `arg.ID()` | **A broken compiler invariant.** `ValidateCoreTypeInfo` runs earlier in the same pipeline (`pipeline_module_compile.go:335`) and already declares that every Core node must have an entry, calling a miss "a compiler bug"; a miss *here* means a later pass (the specializer's cloner) dropped one | **Hard error**, naming the function, the node ID and the pass. Never degrade to leaving the node in place |

The distinction matters because degrading a metadata failure to "leave it alone" would surface
as an ordinary `unencodable builtin: show` skip — and after M3 would be *misreported* as "this
hole's type is not encodable", when the truth is that the compiler lost the type. That is a
silent fallback masking a compiler bug, exactly what Principle 2 forbids. The pass never guesses
a type, and never absorbs a missing one.

**Every minted node needs a `CoreTypeInfo` entry** (revised in r3; see Q4).
`ValidateCoreTypeInfo` checks `coreTI.Has(expr.ID())` for **every** node it walks —
`Lit` and `VarGlobal` are visited as leaves and still checked
(`validate_coretypeinfo.go:78,93,90`) — so a minted node without an entry is recorded as a gap
and fails the compile. The exact obligation, per rewrite row:

| Row | Nodes minted (each needs a fresh ID **and** a `CoreTypeInfo.Set`) | Type to register | Reused unchanged |
|---|---|---|---|
| `string` | *none* — the `App` and the `show` `VarGlobal` are **dropped** | — | `x`, with its existing ID and entry |
| `bool` | `core.If`; `core.Lit{StringLit,"true"}`; `core.Lit{StringLit,"false"}` | `string` for all three | `x` as `If.Cond` — **do not re-register it**; it already has a valid ID and entry |
| `int` | `core.App`; its `core.VarGlobal{Ref:{"$builtin","_string_intToStr"}}` | `string` for the `App`, `int -> string` for the `VarGlobal` | `x` as the sole `App.Arg` — **do not re-register it** |

Dropped nodes' stale `CoreTypeInfo` entries are harmless (the map is keyed by ID and nothing
walks it independently of the tree), so no removal is required. The pass takes a
`*types.CoreTypeInfo` and an ID allocator, exactly as `Specializer` does.

### Implementation Plan

**M1: `ShowNormalizer` — string elision + bool rewrite** (~6h) — *closes the report on its own*
- [ ] `internal/pipeline/show_normalize.go`: walk mirroring `DebugEraser.eraseExpr`
- [ ] `string` case: replace `App(show,[x])` with `x`
- [ ] `bool` case: replace with `core.If{Cond: x, Then: Lit "true", Else: Lit "false"}`,
      registering fresh node IDs in `CoreTypeInfo`
- [ ] Wire into both pipeline paths after `Specialize`
- [ ] Delete the false comment at `parser_literals.go:103-105`; replace with a pointer to this pass
- [ ] Regression fixtures (Conflict Surface §Programs that MUST still work)

**M2: SMT encoding for `intToStr`** (~3h)
- [ ] `StringBuiltinSpecial["_string_intToStr"] = {Op: "str.from_int", IntToStrMode: true}`
      emitting `(ite (>= n 0) (str.from_int n) (str.++ "-" (str.from_int (- n))))`
      (`internal/smt/types.go`)
- [ ] `int` case in `ShowNormalizer`: emit the **application**, not the bare function —
      `core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_string_intToStr"}}, Args: []core.CoreExpr{x}}`
      (r3 fix; the r2 text named only the `VarGlobal`, which would have left an unapplied
      `int -> string` where a `string` is required — see Q4). Use **the `$builtin`
      ref, never the `std/string` wrapper.** `AddBuiltinsToGlobalEnv` binds every registered
      builtin under `$builtin` (`elaborate/core.go:137`), so the rewritten Core carries **no
      module dependency** and a program that never imports `std/string` still links and runs
      (V17, V18). This was the r1 quorum's second blocking objection — see Q2
- [ ] Bonus, same table: `StdlibStringToSMT["intToStr"] = "_string_intToStr"`, so a
      *user-written* `import std/string (intToStr)` verifies too (V8 measured it skipping today)
- [ ] Z3 exactness tests (the encoding must match `strconv.Itoa`, incl. negatives and 0 — V7)

**M3: honest diagnostic for the residue** (~3h) — *revised in r3; see Q3*

`encodable.go` has neither `CoreTypeInfo` nor source provenance, and a surviving
`$builtin.show` may equally have come from an explicit user `show(x)` call. So M3 must
**not** claim the origin, and must not invent the type. It carries the type it already knows
along a channel that already exists:

- [ ] Add `ShowResidue []ShowResidueNote` to `core.DeclMeta` (`internal/core/core.go:430`).
      `ShowNormalizer` — which resolved the argument type by construction — appends one note
      (the type's name) per `show` it deliberately left in place, keyed to the enclosing
      function. The pass can name that function: decls carry it as `LetRec.Bindings[].Name` /
      `Let.Name`, the same mapping `findFunctionBody` uses (`verify_callee_gate.go:133`)
- [ ] `IsSMTEncodable(funcName, meta, body)` **already receives `meta`**
      (`encodable.go:44`) — no new plumbing. When the blocker is `show` and a note exists,
      the message names the measured type:
      `Function "fmtPrice" applies show to a float; Z3 has no string encoding for that type.
      Encodable show arguments: string, bool, int.`
- [ ] The interpolation fact goes in the **Hint**, as advice rather than a diagnosis of this
      instance: `"${x}" desugars to show(x) — if you did not write a show call, look for an
      interpolation hole of this type.` True for an explicit user call as well, so it cannot
      misdiagnose one
- [ ] If no note exists (e.g. the pass was disabled), fall back to today's type-free message
      rather than guessing a type
- [ ] Same treatment for `std/string.floatToStr`

### Files to Modify/Create

**New files:**
- `internal/pipeline/show_normalize.go` — the pass, ~180 LOC (modelled on `debug_erasure.go`)
- `internal/pipeline/show_normalize_test.go` — unit + regression, ~250 LOC

**Modified files:**
- `internal/pipeline/pipeline_module_compile.go` — wire pass after Specialize, ~6 LOC
- `internal/pipeline/pipeline_single.go` — same, ~6 LOC
- `internal/parser/parser_literals.go` — delete the false `show_Int`/`show_String≡id` comment, ~4 LOC
- `internal/smt/types.go` — M2: `StringBuiltinSpecial` entry + `IntToStrMode` field
  (+ the optional `StdlibStringToSMT` bonus entry), ~12 LOC
- `internal/smt/codegen_apps.go` — M2: emit the `ite` form for `IntToStrMode`, ~10 LOC
- `internal/core/core.go` — M3: `ShowResidue` field + `ShowResidueNote` type on `DeclMeta`, ~10 LOC
- `internal/smt/encodable.go` — M3: message + hint driven by the note, ~20 LOC
- `examples/runnable/contracts/string_verify.ail`, `showcase.ail` — no source change;
  they become VERIFIED. Update any inline comment that documents the skip.
- `CHANGELOG.md`, `docs/LIMITATIONS.md` — remove the "string building cannot be verified" limitation

---

## Conflict Surface

This change touches `internal/pipeline/` (Core rewriting), `internal/parser/` (comment only),
and `internal/smt/`. The Core AST is shared by the evaluator, the bytecode VM, the tracer and
the verifier, so the surface is enumerated below.

### Syntactic positions touched

**None.** No grammar production, token position, lexer state or AST shape changes. M1–M3 are
entirely post-parse. The only `internal/parser/` edit is deleting an incorrect comment.

The position actually touched is *semantic*: every `App` node whose callee is
`VarGlobal{Module: "$builtin", Name: "show"}` in the post-specialization Core.

### What else lives here

| Producer of `$builtin.show` in Core | Shape | Effect of the pass |
|---|---|---|
| Interpolation desugar (`parser_literals.go:112`) | `show(hole)` inside a `concat_String` left-spine | **rewritten** by type — the intended target |
| A user's explicit `show(x)` call | identical Core | **also rewritten.** Semantics preserved: `show(s:string) ≡ s` (V1), `show(b:bool) ≡ if b …` (V2). A user who writes `show(s)` on a string gets the same bytes |
| `show` applied to a type variable (generic helper, monomorphization disabled) | `show(x)` with a `TVar` in `CoreTypeInfo` | **untouched** — fail-loud, no guess |
| `internal/gen/lower/expr.go:655` (`Show` class → `_show`) | Go-codegen lowering, a different IR | **untouched** — the pass runs on `core`, not `stmt` |
| REPL's own Show instances (`repl_commands.go:336`) | separate dictionary, quotes strings (`%q`) | **untouched** — different path; note the REPL's `show` on a string is *not* the identity, which is why the pass keys off the `$builtin` ref, never the bare name |

### Disambiguation strategy

The pass matches on `*core.App` whose `Func` is `*core.VarGlobal` with
`Ref.Module == "$builtin" && Ref.Name == "show"` and exactly one argument, then branches on
`CoreTypeInfo.Get(arg.ID())`. Three distinct guards, all structural — no name-only matching
(which would catch a user-defined `show`; see the interaction below), and no rewrite without a
resolved concrete type.

### Interaction with the `export func show` shadowing bug (inbox_1788928007403_88e5056a)

A **second, separately-filed** defect reports that a module's own `export func show` is silently
ignored — the builtin wins at both type-check and eval, no warning, wrong answer, exit 0
(reproduced at v0.35.4, V8). It is a different lane (name resolution), but the two meet at one
point that this doc must record:

> The parser's desugar emits an **unhygienic** `ast.Identifier{Name: "show"}`
> (`parser_literals.go:113`). Today that always resolves to the builtin via
> `globalEnv["show"]` ([`elaborate/core.go:137`](../../../internal/elaborate/core.go#L137)),
> which is *why* the shadowing bug does not currently corrupt interpolation. **If the
> shadowing bug is fixed the obvious way — "let a module's own top-level definition win" —
> then `"${x}"` would silently start calling the user's `show`,** turning one silent-wrong-answer
> bug into another with a wider blast radius.

Whoever takes that bug must make the desugar reference the builtin **hygienically** (emit the
`$builtin.show` global ref directly, or a name users cannot bind) in the same change. Recorded
here so the two lanes do not collide; **not** fixed in this doc.

### Programs that MUST still work

| Fixture | Exercises | Assertion |
|---|---|---|
| `examples/runnable/contracts/string_verify.ail` | `"${a}${b}"` in a contracted function | runtime output byte-identical; `safeConcat` flips SKIPPED → VERIFIED |
| `examples/runnable/contracts/showcase.ail` | `strLength("${prefix}${body}")` | output identical; `prefixedLength` flips SKIPPED → VERIFIED |
| `internal/format/interp_test.go` | `fmt` re-sugaring of `concat_String(show(x), …)` | **byte-identical.** `ailang fmt` parses only (`cmd/ailang/fmt.go:154`), never reaching Core — V9 |
| `internal/pipeline/effect_row_show_interp_test.go` (#386) | effect rows through `show` inside `mapE`/`foldlE` | must stay green, including its must-*reject* controls. `show` is `IsPure: true`, so eliding it cannot widen an effect row; the must-reject cases must still reject |
| A module using `${n}` (int) that does **not** `import std/string` | M2's rewrite target resolving without an import | compiles, links and runs; no missing-module panic (V17) |
| `make verify-examples` (full corpus) | every `${...}` in the repo | zero output diffs |
| `internal/pipeline/builtin_golden_types_test.go` | frozen builtin *types* | unchanged — the `show` builtin itself is neither removed nor retyped |

### What deliberately changes

1. **`ailang verify` results change from SKIPPED to VERIFIED** for string-, bool- and
   int-hole functions. That is the point. Baselines and any pinned `verify_skipped` count
   move — the post-release eval baselines must be re-run, and the shift is expected, not a
   regression.
2. **`AILANG_TRACE=deep` emits one fewer span per elided hole.** No test or tool pins a span
   count for interpolation (V10), but trace transcripts recorded before the change will not be
   span-identical afterwards.
3. **The bytecode VM executes fewer ops** for interpolation (one builtin call per string hole
   removed). Observable only as a small speedup.

Anything not in this list that changes is a regression.

---

## Examples

### Example 1: the reporter's RFC 5322 composer

**Before** — the archetypal "pure contract-bearing core, thin effectful shell":

```ailang
export pure func buildMessage(to: string, from: string, subject: string, body: string) -> string
requires { not(contains(to, "\r")), not(contains(to, "\n")), ... }
ensures  { contains(result, "\r\n\r\n") }
{ "To: ${to}\r\nFrom: ${from}\r\n\r\n${body}" }

$ ailang verify mail.ail
  ⚠ SKIPPED buildMessage
    Reason: Function "buildMessage" uses an unencodable builtin: show
```

**After** — source unchanged:

```
$ ailang verify mail.ail
  ✓ VERIFIED buildMessage  41ms
```

### Example 2: what the pass does to Core

```
-- source:            "n=${count}, ok=${flag}, who=${name}"   (count:int, flag:bool, name:string)

-- Core today:        concat_String(concat_String(concat_String(concat_String(
                        "n=", show(count)), ", ok="), show(flag)), …, show(name))

-- Core after M1+M2:  concat_String(concat_String(concat_String(concat_String(
                        "n=", _string_intToStr(count)), ", ok="),
                        if flag then "true" else "false"), …, name)
                                                              ^^^^ show elided entirely
```

### Example 3: the residue, honestly reported (M3)

```ailang
export pure func fmtPrice(p: float) -> string
ensures { contains(result, "$") } { "$${p}" }

-- Before M3:
--   Reason: Function "fmtPrice" uses an unencodable builtin: show
--   Hint:   Z3 has no SMT-LIB encoding for show. Either remove its use, refactor to
--           use a supported builtin, or narrow the function's contracts.
--
-- After M3 (type from the pass's DeclMeta note — measured, not inferred;
--           origin NOT claimed, because provenance does not exist):
--   Reason: Function "fmtPrice" applies show to a float; Z3 has no string
--           encoding for that type. Encodable show arguments: string, bool, int.
--   Hint:   "${x}" desugars to show(x) — if you did not write a show call, look
--           for an interpolation hole of this type.
```

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Rewrite in the **shared compile pipeline** rather than verify-locally | Changes the Core every program compiles to, not just verified ones. Buys a runtime simplification and keeps the SMT layer untyped; costs a wider blast radius | human | design | high |
| Elide `show(x:string)` **entirely** rather than mapping it to an encodable identity builtin | Simplest and fastest, but erases the node from traces | agent | design | med |
| Encode `int` via `_string_intToStr` + `ite`/`str.from_int` rather than leaving int holes unverified | Rests on Z3 string-theory semantics — an external contract. `str.from_int` returns `""` for negatives, so the naive encoding would be silently wrong | human | design | med |
| Do **not** fix the `export func show` shadowing bug here | Keeps this doc to one lane, but leaves a live trap for whoever fixes that one | human | design | low |

### Design Freeze

**Both ratified 2026-09-09 by Mark (attended session). Sprint may proceed.**

- [x] **Pipeline-wide vs verify-local placement — GRANTED: pipeline-wide.**
      The decisive argument is not blast radius but soundness posture: a verify-local rewrite
      would have the verifier prove properties of a Core tree the evaluator never executes.
      Pipeline-wide keeps "what was proved" and "what runs" the **same tree**. Secondary: the
      SMT layer stays untyped (no sort oracle threaded through `encodable.go`, `codegen.go`
      and `callee_resolver.go`), and M3's diagnostic needs the pass to exist regardless, since
      the pass is the only place the argument type is known. The trace-span change (Conflict
      Surface §3.2) is accepted: the elided span represented a no-op.
- [x] **M2's Z3 dependence — GRANTED**, and weaker than r1 assessed. `str.from_int` returning
      `""` for a negative argument is **SMT-LIB standard behavior**, not a Z3 implementation
      quirk, so the dependence is on the specification. Further, the `ite` form never passes a
      negative to `str.from_int` at all — a future solver that began handling negatives would
      not change our result. The CI exactness assertions (M2) mean a solver upgrade that does
      break it fails loudly rather than silently producing wrong proofs.
      *Pre-existing and out of scope:* Z3's `Int` is unbounded while AILANG's `int` is 64-bit.
      That gap already applies to every arithmetic encoding in the fragment; this doc neither
      widens nor narrows it.

---

## Success Criteria

- [ ] The reporter's minimal pair: `withInterp` VERIFIED, `noInterp` still VERIFIED
- [ ] `examples/runnable/contracts/` — `show`-blocked skips 2 → 0, no new skips, no new violations
- [ ] `${s}` (string), `${b}` (bool), `${n}` (int, incl. negative and 0) holes all verify
- [ ] `${f}` (float) still skips, with the M3 message naming the argument type from the
      pass's `DeclMeta` note, and the interpolation fact confined to the Hint
- [ ] An **explicit** `show(p: float)` call — not from interpolation — gets the same message
      and is not misdiagnosed (Q3)
- [ ] `make verify-examples` — zero runtime output diffs across the corpus
- [ ] `ailang fmt` — byte-identical on every interpolation fixture in `internal/format/interp_test.go`
- [ ] `#386` effect-row tests green, **including their must-reject controls**
- [ ] `make test`, `make lint`, `make check-file-sizes` clean
- [ ] CHANGELOG + `docs/LIMITATIONS.md` updated; reply sent to `fb_913ee851c83c0d8c`

## Testing Strategy

**Unit (`show_normalize_test.go`):**
- One case per type row (string / bool / int / float / TVar / record), asserting the exact
  post-pass Core shape and that `CoreTypeInfo` has an entry for every node the pass mints
- Fail-loud: a **`TVar`/record/ADT/float** type leaves the node untouched; a **missing**
  `CoreTypeInfo` entry is a hard error (assert on the error, not on a degraded skip)
- Nested holes: `"${show(x)}"` — a user's explicit `show` inside a hole, i.e. `show(show(x))`

**Semantic equivalence (the load-bearing property):**
- Table-driven runtime comparison of `show(v)` vs the rewrite for each type, over an
  adversarial value list: `""`, embedded `"`, embedded `\n`, a 100-char string (proves no
  `maxWidth=80` truncation applies — V1), `0`, `-5`, `math.MinInt`, `true`/`false`

**SMT exactness (M2):** assert `unsat` on the negation of `showInt(n) = strconv.Itoa(n)` for a
sampled range plus the symbolic property `str.len(showInt(n)) > 0` — the encoding must be exact,
not merely satisfiable (V7 is the hand-run version of this).

**Regression-surface (one per Conflict Surface fixture):** listed in that section; each pins
exact output, not just exit code (CLAUDE.md: a panic exits non-zero, so exit-code tests go
green on a crash).

## Deferred Decisions

- Whether the `bool` rewrite emits `core.If` or a new `show_Bool` builtin — **agent may
  choose**; `If` is preferred (zero verifier change, V5) but a builtin is acceptable if the
  `If` shape trips `hasDeepPatterns`.
- Node-ID allocation strategy for minted nodes — **agent may choose**; mirroring
  `Specializer.freshNodeID`'s high-watermark is the obvious route.
- Whether M3's message is a new `RejectionCode` or new text on `RejectUnencodable` —
  **agent may choose**; text-only is preferred (no JSON schema change). The *source* of the
  type is not open: it must be the `DeclMeta` note the pass records, never inferred in
  `encodable.go` (Q3).

## Non-Goals

- **Encoding `show` for float, list, record or ADT holes.** Z3 has no Real→String theory, and
  a composite rendering would have to reproduce `showValue`'s depth/width elision
  (`maxDepth=3`, `maxWidth=80`, `elisionPrefix=20`) exactly — a large exact-match surface for
  little verification gain.
- **Fixing the `export func show` shadowing bug** (`inbox_1788928007403_88e5056a`). Recorded
  in Conflict Surface as an interaction; routed separately.
- **Full provenance on synthesized nodes** ("this `show` came from interpolation at line N").
  M3 gets the same user benefit from message text alone. See Future Work.
- **IFC Phase 2 type-level enforcement.** This doc widens what the *existing* Z3 route can
  cover; it does not change the enforcement architecture.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `show(s:string)` is not exactly the identity in some case we missed | High — silent output change corpus-wide | Source says so explicitly (`show.go:118-120`, `// Return string without quotes (identity for strings)`); verified at runtime incl. quotes, newlines, empty and 100-char (V1); `make verify-examples` diffs the whole corpus |
| Z3's `str.from_int` differs across versions | Med — M2 encodings silently wrong | The `ite` form is standard SMT-LIB, verified against 4.15.4 (V7); exactness assertions run in CI, so a solver upgrade that breaks it fails loudly |
| Minted nodes missing from `CoreTypeInfo` | Med — `ValidateCoreTypeInfo` panic or a downstream pass fault | That validator already runs in-pipeline and will catch it; unit test asserts an entry per minted node |
| A `CoreTypeInfo` miss on a `show` argument is absorbed as an ordinary skip, masking a compiler bug | Med — M3 then misreports a metadata failure as an unencodable hole type | The pass **errors** on a missing entry rather than degrading (Architecture §Fail-loud, Q1); the unit suite asserts the error, not a skip |
| The `int` rewrite emits a `std/string` module ref instead of the `$builtin` ref | High — missing-module panic on programs that never import `std/string` | The spec names the exact node to emit; V17/V18 measure that the `$builtin` ref needs no import; a regression fixture pins a no-import program (Q2) |
| Trace-span change breaks a consumer we did not find | Low | Grepped: nothing pins interpolation span counts (V10); called out in Conflict Surface §3.2 |
| Elision hides a `show` a reader expected in a trace | Low | Accepted; the node is semantically absent, and `AILANG_TRACE=deep` still shows the `concat_String` chain |

---

## Verification Log

Every load-bearing claim, re-verified first-party at `dev = 8e8e8c9bf` (v0.35.4), Z3 4.15.4,
darwin/arm64, 2026-09-09. Positive **and** negative-existence claims both carry rows.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | `show(s:string)` is exactly `s` — no quoting, escaping or truncation | Read `internal/builtins/show.go:118-120` (`case *eval.StringValue: return val.Value`, comment "identity for strings"); ran `show` on `""`, `he said "hi"`, `a\nb`, and a 100-char string (> `maxWidth=80`) | **Confirmed.** All identity; the 100-char string returned `len=100`, so `truncateIfNeeded` does not apply to bare strings |
| V2 | `show(true)="true"`, `show(false)="false"`, `show(-5)="-5"` | `ailang run` | **Confirmed** |
| V3 | The reported skip reproduces at v0.35.4 | `ailang verify interp.ail` | **Confirmed.** `⚠ SKIPPED withInterp … unencodable builtin: show`; `noInterp` VERIFIED (64ms) |
| V4 | `concat_String` chains — including the RFC 5322 header case — verify **today** | `ailang verify wk.ail` | **Confirmed.** `viaConcat` 4.7ms, `hdr` 38.2ms, both VERIFIED. The encoder is not missing a theory |
| V5 | The bool rewrite target (`if b then "true" else "false"` inside `concat_String`) encodes with **no verifier change** | Hand-wrote `boolTag`, ran `ailang verify` | **Confirmed.** VERIFIED 31.7ms |
| V6 | Core nodes carry **no** inline type annotations; types live in `types.CoreTypeInfo` keyed by NodeID (negative-existence — the design's reason for a pipeline pass over an SMT-local fix) | Read `internal/core/core.go:57-150` (`App`, `VarGlobal`, `Lit` — no type field); `internal/types/typeinfo.go:49-90` | **Confirmed** |
| V7 | Z3's `str.from_int` returns `""` for negatives; the proposed `ite` encoding is **exact** | Ran Z3 directly: `(str.from_int (- 5))` → `sat`, `s = ""`. Then asserted the negation of `showInt(42)="42"`, `showInt(-5)="-5"`, `showInt(0)="0"` and of `str.len(showInt(n))>0` | **Confirmed.** All four `unsat` — the naive encoding would be silently wrong; the `ite` form is exact |
| V8 | `_string_intToStr` (`std/string.intToStr`) matches `show` on ints, and **is not** in the SMT tables today | `ailang run` on `42/-5/0` (identical to `show`); `ailang verify` on `intTag` | **Confirmed.** Skips naming `std/string.intToStr` — so M2 is exactly two table entries |
| V9 | `ailang fmt` never reaches Core, so a Core pass cannot affect formatting (safety property for the fmt round-trip) | Read `cmd/ailang/fmt.go:154,169` — `parser.New(lexer.New(src, path))`, no `pipeline.Run` | **Confirmed** |
| V10 | **No** existing Core→Core pass special-cases `show` (negative-existence — the pass is new, not a duplicate) | `grep -rn '"show"' internal/pipeline/ internal/link/ --include=*.go` (excluding tests) | **Confirmed — zero hits** |
| V11 | **No** golden file pins `$builtin.show` inside a Core tree (negative-existence — bounds the test churn) | `grep -rln '\$builtin\.show' --include=*.golden --include=*.json internal/ cmd/` | **Confirmed — zero hits** |
| V12 | `show` is **not** special-cased in IFC/taint checking, so eliding it cannot widen or launder a label (negative-existence) | `grep -n "show\|concat_String" internal/types/ifc_check.go internal/types/sink_check.go` | **Confirmed — zero hits** |
| V13 | `show_Int`/`show_String`/`show_Bool`/`show_Float` are **not** registered runtime builtins — the `parser_literals.go` comment claiming elaboration-time dispatch is false | `ailang builtins list \| grep -i show` → one entry, `show [pure] $builtin`. The four names appear only in `internal/iface/builtin_freeze.go:81-84` | **Confirmed** |
| V14 | The `export func show` shadowing bug is live at v0.35.4 (basis for the Conflict Surface interaction) | Ran a module defining `export func show(x:int)->string { "MINE" }`, then `show(7)` | **Confirmed.** Prints `7`, not `MINE` |
| V15 | The 2/6 skip measurement across `examples/runnable/contracts/` | Ran `ailang verify` over all 35 files, tallied `Reason:` lines | **Confirmed.** 6 skips: 2 × `show`, 1 × `std/string.trim`, 1 × no-ensures, 2 × unencodable callee type |
| V16 | `show` is `IsPure: true`, so elision cannot change an effect row (#386 safety) | Read `internal/builtins/show.go:24` (`IsPure: true`) | **Confirmed** |
| V17 | `_string_intToStr` resolves with **no** `import std/string` — M2's rewrite injects no module dependency (added r2, answering Q2) | Ran a module with no imports calling `_string_intToStr(-5)` | **Confirmed.** Compiles, effect-checks, runs, prints `-5`. No link or module error |
| V19 | `ValidateCoreTypeInfo` requires an entry for **every** node, `Lit` and `VarGlobal` included (basis for the r3 minting table — added r3, answering Q4) | Read `internal/pipeline/validate_coretypeinfo.go:78` (`if !v.coreTI.Has(expr.ID()) { v.recordGap(expr) }`, before the child switch) and the `*core.Lit` / `*core.VarGlobal` leaf cases at :90,:93 | **Confirmed.** A minted `Lit "true"` with no entry fails the compile |
| V20 | `IsSMTEncodable` already receives `*core.DeclMeta`, so M3's type note needs **no new plumbing** (added r3, answering Q3) | Read `internal/smt/encodable.go:44`; `core.DeclMeta` at `internal/core/core.go:430` | **Confirmed** |
| V21 | The pass can name the function it is rewriting, so a per-function note is keyable (added r3, answering Q3) | Read `findFunctionBody` (`cmd/ailang/verify_callee_gate.go:133`) — decls are `*core.LetRec` with `Bindings[].Name`, or `*core.Let` with `.Name` | **Confirmed** |
| V18 | That call reaches the encoder as a **`$builtin`** ref, not a `std/string` one (added r2, answering Q2) | `ailang verify` on a contracted function calling it | **Confirmed.** Blocker reported as `_string_intToStr`, bare. `firstUnencodableBuiltin` prints a bare name only on the `Ref.Module == "$builtin"` branch; the `std/string` branch prints `std/string.<name>` — which is exactly what V8's `intToStr` case produced. The two messages differing is the proof |

### Quorum trigger assessment

Two of the four attended-session triggers fire, so this doc **should** go to
`ailang design-quorum` before sprint planning:

| # | Trigger | Fires? | Why |
|---|---|---|---|
| 1 | Design-freeze items present | **YES** | Two: pipeline-wide placement, and the Z3 dependence |
| 2 | Overrides shared machinery | No | The pass *adds* to the pipeline; every Conflict Surface row is "reuse" or "untouched" |
| 3 | Cost/KPI semantics or banked schema | No | No schema or predicate change. `verify_skipped` **values** shift (intended); its meaning does not |
| 4 | Load-bearing premise about an external system | **YES** | M2 rests on Z3 string-theory semantics (V7). Verified against 4.15.4, but Z3 is not ours |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Pure syntactic rewrite of a deterministic identity; no new nondeterminism |
| A2: Replayability | 0 | Traces lose one span per elided hole (Conflict Surface §3.2); replay of *programs* is unaffected — the elided call had no observable effect |
| A3: Effect Legibility | 0 | `show` is `IsPure` (V16); effect rows are unchanged, and #386's controls must stay red |
| A4: Explicit Authority | 0 | No authority surface touched |
| A5: Bounded Verification | **+1** | The headline gain: an entire class of functions moves from "cannot be encoded" into the decidable fragment, with no new solver theory |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | **+1** | A generating model writes idiomatic `"${...}"` and gets a proof, instead of being pushed to hand-fold `concat_String` chains to satisfy the verifier. M3 also removes a diagnostic that sends readers hunting a call that does not exist |
| A8: Minimal Syntax | **+1** | Removes a spurious node rather than adding syntax; the fix is invisible in source |
| A9: Cost Visibility | 0 | One fewer builtin call per string hole; no cost accounting changes |
| A10: Composability | **+1** | Contract-bearing string builders now compose with the rest of the verified fragment — the "pure core, thin effectful shell" pattern the language's own guidance asks for |
| A11: Structured Failure | **+1** | M3 replaces a misleading skip reason with one that names the real cause and the encodable types |
| A12: System Boundary | 0 | No boundary crossing changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects; `show` is pure (V16)
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): +1 — optimizes for machine-checkable output, not human convenience

---

## Related Documents

**Implemented (inform this design):**
- [m-smt-fragment-expansion](../../implemented/v0_8_0/m-smt-fragment-expansion.md) (0.31) —
  added the string/list builtin SMT mappings this doc extends. `concat_String → str.++` comes
  from there; M2 adds one entry in the same tables.
- [m-verify-runtime-contracts](../../implemented/v0_6_1/m-verify-runtime-contracts.md) (0.30) —
  the `--verify-contracts` runtime path. Unaffected: contracts are evaluated on values, and
  the rewrite is value-preserving.

**Planned (checked for overlap — all distinct):**
- [m-contract-verification-coverage](../m-contract-verification-coverage.md) — *complementary,
  not overlapping.* That doc splits `verify_skipped` into `skipped` vs `not_applicable`; this
  one removes a class of genuine `skipped`. Landing both makes the encoder-coverage figure both
  correctly categorized and smaller.
- [m-verify-bounded-unrolling-false-counterexample](../m-verify-bounded-unrolling-false-counterexample.md)
  (0.27) — recursion-depth soundness, orthogonal to the builtin-encoding surface.

**Source reports:**
- `fb_913ee851c83c0d8c` — the report this doc answers
- `inbox_1788928007403_88e5056a` — the `export func show` shadowing bug (separate lane; see
  Conflict Surface for the interaction that couples them)
- `fb_a71eab12139bee26` — IFC Phase 2, cited by the reporter as the downstream stake

## Future Work

- **Node provenance for synthesized desugar output.** A flag on `ast.FuncCall` marking
  parser-synthesized nodes would let M3 say "the `show` at line 12 col 20 came from a `${}`
  hole" exactly, and would let `internal/format/interp.go` re-sugar from a marker instead of
  pattern-matching the `show(...)` shape — making `ailang fmt`'s round-trip exact rather than
  heuristic.
- **Float holes via a rational-to-decimal encoding**, if a benchmark ever demands it.
- **Hygienic desugar**, jointly with the `export func show` shadowing fix.

---

## Quorum verification log

**Round 1** — `ailang design-quorum` at 2026-09-09T05:39:02Z, reviewers `gpt5-6-sol` +
`gemini-3-1-pro`, controller verdict `pass`. **Synthesis: BLOCKED** (2/2 reviewers reject,
$0.0809 total, 20106 in / 554 out tok). Artifact:
`.ailang/state/mission-quorum/m-smt-interp-show-2026-09-09T05-39-02Z.json`.

Both objections were accepted in full and fixed; neither was argued.

**Q1 — `gpt5-6-sol`**: *"The proposed handling of a missing `CoreTypeInfo` entry is a silent
fallback, not 'fail-loud': it leaves `show` unchanged and later reports an ordinary SMT-
encodability skip, masking a broken compiler invariant. M3 can then misdiagnose that metadata
failure as an unsupported interpolation type."*

**Correct, and the r1 text called the wrong behavior "fail-loud".** r2 splits the lookup into
two outcomes (Architecture §Fail-loud): a *present but unsupported* type is legitimate residue
and leaves `show` alone; a *missing entry* is a broken invariant — `ValidateCoreTypeInfo` runs
earlier in the same pipeline and already calls such a miss a compiler bug — and is now a hard
error naming the function, node ID and pass. Testing Strategy and the Risks table were updated
to assert the error rather than a degraded skip.

**Q2 — `gemini-3-1-pro`**: *"The M1/M2 design rewrites `show(x:int)` to a `core.App` calling
`std/string.intToStr`. This injects a cross-module dependency into the post-typechecked Core
AST. If a user's code uses integer interpolation but does not explicitly `import std/string`,
the compiler/VM will fail to resolve the `std/string.intToStr` global reference … causing a
missing module panic on previously valid programs."*

**The hazard is real; the r1 doc invited the reading by writing the rewrite target as
`intToStr(x) → _string_intToStr`, which reads as the `std/string` wrapper.** Measured rather
than argued: `AddBuiltinsToGlobalEnv` binds every registered builtin under `$builtin`
(`elaborate/core.go:137`), so the target is `$builtin._string_intToStr` and carries no module
dependency — a module with **no imports at all** calling it compiles, links and runs (V17), and
the verifier reports it via the `$builtin` branch, not the `std/string` one (V18). r2 names the
exact node to emit, adds both verification rows, adds a no-import regression fixture to the
Conflict Surface, and adds a Risks row so an implementer who emits the wrong ref is caught.

**Round 2** — re-quorum at 2026-09-09T05:42:18Z, same reviewers, controller `pass`.
**Synthesis: BLOCKED** again (2/2 reject, $0.0934, 23345 in / 648 out tok). Artifact:
`.ailang/state/mission-quorum/m-smt-interp-show-2026-09-09T05-42-18Z.json`. Both objections
were **new** (neither reviewer repeated an r1 point), both were accepted, and both were
specification-precision defects rather than architecture faults.

**Q3 — `gpt5-6-sol`**: *"M3 promises a type- and interpolation-specific diagnostic that the
proposed layer cannot produce. The document itself verifies that `encodable.go` has neither
`CoreTypeInfo` nor source provenance, while the same surviving `$builtin.show` shape can come
from an explicit user call. Therefore a message such as 'interpolates a … (float)' is either
impossible to derive or will misdiagnose explicit `show` calls."*

**Correct on both halves — r2's M3 overclaimed twice.** The type is now carried from the one
place that knows it (the pass, by construction) to the one place that reports it, along a
channel that already exists: `IsSMTEncodable` already takes `*core.DeclMeta` (V20), and the
pass can key a note per function (V21). The **origin** claim is dropped entirely — the message
states only what is measured ("applies show to a float"), and the interpolation fact moves into
the Hint as conditional advice that is true for an explicit `show` call as well. Provenance
proper stays in Future Work, where r1 already put it.

**Q4 — `gemini-3-1-pro`**: *"In M2, the int case specifies emitting a `core.VarGlobal` (an
unapplied function of type `int -> string`) instead of a `core.App` applying that function to
`x`. Furthermore, the Architecture section explicitly instructs registering a fresh node ID …
for the 'reused argument node' (which already possesses a valid ID and type entry), while
entirely failing to mandate type registration for the actually newly minted sub-nodes
(`Lit "true"`, `Lit "false"`, and the new `core.VarGlobal` and `core.App`)."*

**Both are real bugs in the r2 spec, and an implementer following it literally would have
produced a malformed tree.** Fixed exactly: the `int` row now names the full
`core.App{Func: VarGlobal{…}, Args: [x]}`, and the Architecture section replaces the prose with
a per-row minting table listing every node that needs a fresh ID and a `CoreTypeInfo.Set`, the
type to register for each, and — explicitly — that the reused argument node keeps its existing
entry and must **not** be re-registered. V19 records the validator behavior the table is
written against.

### Handover state

Two rounds spent, four objections, all fixed. Per the guardrail a third round is **not** spent
here: the remaining gate is human ratification of the two Design Freeze items (pipeline-wide
placement, and depending on Z3's `str.from_int` semantics), which no reviewer can grant. Both
freeze items are unchanged since r1 and neither round objected to them. A third quorum is
reasonable **after** those two are ratified, if the ratifier wants one; sprint planning should
not start before they are.

---

**Document created**: 2026-09-09
**Last updated**: 2026-09-09 (r2, post-quorum)
