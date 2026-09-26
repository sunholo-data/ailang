# M-TRACE-LABEL-AWARE: the tracer is below the type system, so IFC labels do not reach it

**Status**: Planned — **unblocked**. Its main open question (does the label propagate?) is
resolved favourably, and the IFC hole it would have inherited is fixed (`1af9f5f30`). M0 (the
`std/secret` doc correction) has shipped. The remaining unknowns are higher-order call sites and
the conservative-redaction policy, both stated below.
**Target**: v0.36.0
**Priority**: P0 — `std/secret` documents a guarantee the runtime does not provide. A user who
follows the documentation exactly still writes their credential to disk.
**Estimated**: 3–4 days (M0 is a ~30-minute doc correction that should land immediately)
**Dependencies**: [m-trace-tier-not-enforced](m-trace-tier-not-enforced.md) — **that doc's M1
closes the DEFAULT case; this one is what makes `deep` usable.** See [Layering](#layering-what-each-doc-fixes).
**Author**: design-doc-creator role, attended session 2026-09-09, at `dev` = `d7bb689b8`
**Source**: `fb_5c0baeb1ee5ae1eb` (mcp-public, 2026-09-08), with `fb_b023726953f2ee5a` as the
corrected follow-up in the same chain

---

## Problem Statement

`-emit-trace` records every function's arguments and results as strings. Tracing happens
**below the type system**, so IFC labels and sink refinements never reach it. The mechanism that
answers *"can we observe what it did"* silently defeats the mechanism that answers
*"what is it allowed to leak"*.

The reporter measured eighteen verbatim copies of a live Google bearer token in the trace of a
twelve-line program that deliberately prints only its first ten characters.

### The part that makes this P0: `std/secret` promises this cannot happen

[`std/secret.ail:5-7`](../../../std/secret.ail#L5) says, of the value `secret()` returns:

> the resolved value is labelled `<secret>` (M-TAINT-TYPES) so the type system forbids it from
> reaching **{not secret} sinks (logs, traces, network bodies)** without an explicit
> `! {Declassify}` step.

It names traces explicitly. That guarantee does not hold. Measured first-party at
`d7bb689b8`, default tier, no flags beyond `--emit-trace`:

```ailang
export func classify(raw: string) -> string<secret> ! {} = raw
export func firstChars(s: string<secret>, n: int) -> string ! {} = substring(s, 0, n)

export func main() -> () ! {IO} {
  let token = classify("CANARY_ya29_SUPERSECRET_VALUE") in
  println("token starts: ${firstChars(token, 6)}...")   -- prints 6 chars, deliberately
}
```

```
$ ailang run --caps IO --emit-trace jsonl labeltrace.ail | grep -c 'CANARY_ya29_SUPERSECRET_VALUE'
4
```

Four verbatim copies of a value carrying the very label documented to keep it out of traces.
The label governs *sinks* — `httpRequest`, and any function whose signature demands
`{not secret}`. The tracer is not a sink; it is underneath, and it serialises the value at every
call boundary on the way to one.

So an IFC-clean program leaks comprehensively through its own audit trail, and nothing in the
type system says otherwise.

### Why this is structural, not a configuration mistake

**The trace is the artifact you keep.** It is the provenance record — the thing an agent
framework retains, ships to a coordinator, and feeds to AILANG world for exactly the trust and
audit story that motivates the language. The highest-value artifact is the one with the least
protection, and its value grows with its retention.

The reporter's case (`fb_a71eab12139bee26`, the Daneel virtual employee) is a mailbox and client
documents on a machine whose disk is not encrypted, where `~/.ailang/state` is already 1.0 GB.
Turning tracing on for the audit story would write client correspondence verbatim into a growing
store — which is what the engagement's confidentiality terms forbid, and what the effect
capabilities and Drive ACLs were arranged to prevent.

**The team that adopts AILANG *for* the trust story is the team most likely to turn tracing on.**

### Root cause: the redaction decision is made after the type is gone

[`internal/eval/eval_operations.go:117-122`](../../../internal/eval/eval_operations.go#L117):

```go
if recorder, ok := e.effContext.(TraceRecorder); ok && recorder.HasTraceCollector() {
    argStrs := make([]string, len(args))
    for i, a := range args {
        argStrs[i] = a.String()          // <-- rendered here, unconditionally
    }
    recorder.RecordFunctionEnter(funcName, argStrs)
}
```

Two facts decide the design:

1. **The collector receives `[]string`, never values.** `RecordFunctionEnter(name string, args []string)`.
   By the time the trace subsystem has the data, the type — and therefore the label — is gone.
   No amount of work inside `internal/trace` can make the tracer label-aware.
2. **Labels are erased before runtime.** `internal/eval/value.go` has no label concept (V4). A
   runtime value cannot be asked whether it is secret.

The redaction decision therefore has to be made **at the render site**, which is the one place
that still has both the values and a compile-time identity for the call.

---

## Layering: what each doc fixes

These are two changes at the same seam, and they are not substitutes.

| | [m-trace-tier-not-enforced](m-trace-tier-not-enforced.md) | this doc |
|---|---|---|
| Fixes | `standard` (the **default**) records per-call values at all | `deep` records **labelled** values |
| After it ships | Secure by default; per-call observability unavailable at `standard` | Per-call observability available with labelled values redacted |
| Answers | "stop the default leak, and the OOM" | "**observability *and* confidentiality**, rather than a choice" |

**Ship the tier fix first.** It is a bug fix with no design unknowns and it closes the default
exposure. This doc is the one that answers the reporter's actual complaint — that today the
choice is observability *or* confidentiality — and it has real unknowns, below.

---

## Solution Design

### Overview — carry the label from compile time to the render site

The label is known statically. The render site has the AST node. So mark the call site at
compile time and let the evaluator honour the mark, exactly as
`ast.Identifier.ResolveAsBuiltin` (M-SMT-INTERP-SHOW, v0.36.0) carries a compile-time decision
to a place that cannot re-derive it.

```
typecheck        knows `token : string<secret>`
   ↓  records per-argument redaction flags on the App node
Core             App{ ..., RedactArgs: []bool, RedactResult: bool }
   ↓
eval_operations  renders "<redacted:secret>" instead of a.String()
   ↓
collector        never sees the value — cannot leak what it was never given
```

Rendering `<redacted:secret>` rather than dropping the argument keeps the trace's structural
value intact: the span tree still shows `exchangeRefresh → postFormBody → decode`, arity is
preserved, and the reader can see *that* a secret flowed without seeing *what*.

### Milestones

**M0 — correct the false promise** (~30 min) — **land immediately, ahead of the rest**
- [ ] `std/secret.ail:5-7` currently tells users traces are a protected sink. Until M2 ships they
      are not. Amend to state what is actually enforced (network/sink positions) and warn
      explicitly that `-emit-trace` captures the value
- [ ] Same for `ailang docs std/trace`, which the reporter notes says nothing about arguments and
      results being captured verbatim
- This is not a workaround; it is removing a security claim we do not honour. It should not wait
  for a sprint.

**Discovered while doing M0, and it generalises:** `ailang docs <module>` renders only the
**first line** of a doc comment. A multi-line safety note placed under the summary is invisible
in the exact tool a reader consults. The `std/secret` warning had to be folded into the first
line of both the module and the function comment to surface at all — which works, but is
load-bearing formatting nobody would guess. Two follow-ups worth their own item: render more
than the first line (or a dedicated `-- WARNING:` convention that `ailang docs` promotes), and
audit whether any other stdlib module has a safety note currently invisible for the same reason.

**M1 — plumb static redaction flags** (~1.5 days)
- [ ] Typechecker: at each application, determine per-argument whether the static type carries a
      label; record on the Core `App` node
- [ ] Evaluator: honour the flags at `eval_operations.go:118` — render `<redacted:LABEL>`, and
      **skip `a.String()` entirely** so the value is never materialised
- [ ] Same treatment for `RecordFunctionExit`'s result

**M2 — effect spans** (~1 day)
- [ ] `ailang.effect.args` / `ailang.effect.result` are the highest-risk surface — every
      `readFile` ships a file, every `httpRequest` a response body (`fb_b0237` measured this
      reaching an OTLP collector). Apply the same redaction
- [ ] Adversarial test: the reporter's OAuth shape — a labelled token crossing `postFormBody`,
      `json.decode`, `getString`, `substring` — with a canary asserted absent from the trace

**M3 — make the guarantee checkable** (~0.5 day)
- [ ] A canary test that greps the emitted trace for a labelled value and fails if present. This
      is the test whose absence let the `std/secret` promise drift from the truth
- [ ] `ailang docs std/secret` restored to a guarantee that now holds

### Files to Modify

- `std/secret.ail` — M0 doc correction, ~8 lines
- `docs/docs/guides/debugging.md` — trace capture semantics, ~20 lines
- `internal/types/` — label extraction at application sites, ~80 LOC
- `internal/core/core.go` — redaction flags on `App`, ~15 LOC
- `internal/eval/eval_operations.go` — honour the flags, ~25 LOC
- `internal/effects/context.go` — effect arg/result redaction, ~30 LOC
- tests — canary + per-label matrix, ~250 LOC

---

## Open Questions — these are real, and they are why this is a design and not a bug fix

1. ~~**Does a label survive through every path a value takes?**~~ **ANSWERED by a spike,
   2026-09-09 — mostly yes, with one hole that is a security defect in its own right.**

   Measured against a `{not secret}` sink, with the direct case as a firing control:

   | path | verdict |
   |---|---|
   | `sink(s)` — control | **BLOCKED** ✓ |
   | `let copied = s in sink(copied)` | **BLOCKED** ✓ |
   | `let r = { payload: s } in sink(r.payload)` | **BLOCKED** ✓ |
   | `let xs = [s] in match xs { ... x :: _ => sink(x) }` | **BLOCKED** ✓ |
   | `let f = \u. sink(s) in f(0)` — sink *inside* the lambda | **BLOCKED** ✓ |
   | `let f = \u. s in sink(f(0))` — label returned *from* a lambda | **LAUNDERED** ✗ |
   | `(\u. sink(s))(0)` | **LAUNDERED** ✗ |
   | `let f = \u. s in let g: string<secret> = f(0) in sink(g)` | **LAUNDERED** ✗ |

   **Good news for this design:** propagation through `let`, record fields and lists is sound, so
   static redaction is viable and does not depend on M-TAINT-TYPES Phase 2 for the common paths.

   **Bad news, and it is not about tracing:** a label is dropped from a lambda's inferred return
   type. `let f = \u. s in sink(f(0))` defeats the `{not secret}` guarantee in three tokens, and
   an explicit `string<secret>` annotation on the result does not restore it. The checker *does*
   look inside lambdas — a sink called within one is correctly blocked — so this is specifically
   the label being lost on the way out.

   **FIXED, commit `1af9f5f30`** — so this design no longer inherits the hole. Two gaps had to
   line up: `ifc_check.go`'s `Lambda` case returned `LabelBottom()`, and `labelOfCall`'s
   fallback (the path every local closure takes) joined only the *argument* labels, discarding
   the callee's own. Both now propagate; nine paths are pinned by
   `TestIFCClosureCannotLaunderALabel` with the direct call as a firing control.

   **Net effect on this doc: Open Question 1 is closed, favourably.** Label propagation is sound
   across `let`, record fields, lists, closures and annotated bindings, so static redaction is
   viable and does **not** depend on M-TAINT-TYPES Phase 2.

   One caveat carried forward: propagation is now *over*-approximate (a closure is labelled with
   what its body returns, used or not). For redaction that is the safe direction — it redacts
   more than strictly necessary, never less.
2. **Higher-order calls.** At `App` the callee may be a runtime closure whose parameter labels
   are not statically known at that site.
3. **Should redaction be conservative?** A defensible fallback: when a call site's argument type
   is *unknown or a type variable*, redact rather than render. Safe by default, at the cost of a
   less useful trace in polymorphic code. Recommended, but it is a policy call.
4. **Is a runtime representation needed after all?** If (1) and (2) leave gaps, the alternative
   is a distinct runtime value for labelled data whose `String()` redacts — closing them by
   construction, at the cost of touching the value representation (which
   [m-list-cons-quadratic](../m-list-cons-quadratic.md) shows is expensive: 902 `.Elements`
   references).

**These questions are why M0 must not wait.** However they resolve, the documentation is wrong
today.

---

## Verification Log

First-party at `dev = d7bb689b8`, v0.35.4+, darwin/arm64.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | An IFC-labelled value reaches the trace verbatim | The `string<secret>` repro above, default tier | **Confirmed. 4 verbatim copies** of a canary the program prints 6 characters of |
| V2 | `std/secret` documents traces as a protected sink | Read `std/secret.ail:5-7` | **Confirmed** — "{not secret} sinks (logs, traces, network bodies)" |
| V3 | The collector receives strings, never values — so it *cannot* be made label-aware internally | Read `collector.go:137` signature and `eval_operations.go:117-122` | **Confirmed.** `args []string`, rendered by `a.String()` at the call site |
| V4 | Labels are erased before runtime (negative existence — rules out asking the value) | `grep -n "Label" internal/eval/value.go` | **Confirmed — zero hits** |
| V5 | `std/secret` exists and is the natural home for the concept | `ls std/secret.ail` | **Confirmed** |
| V6 | The exporter-side measurement is **not** reproduced here | — | **Not verified.** `fb_b0237` measured otel transmission independently; this doc measures only what is *recorded* |

### Quorum trigger assessment

| # | Trigger | Fires? |
|---|---|---|
| 1 | Design-freeze items | **YES** — the conservative-redaction policy, and static-vs-runtime |
| 2 | Overrides shared machinery | **YES** — changes Core's `App` shape and the typechecker |
| 3 | Cost/KPI or banked schema | No |
| 4 | External-system premise | No — all in-repo |

Run the quorum. The open questions above should be resolved or explicitly parked first;
submitting with four live unknowns invites a reject that tells us what we already know.

---

## Non-Goals

- **Exporter behavior.** What leaves the machine is `fb_b0237`'s axis (V6).
- **Encrypting the trace store.** Real, and orthogonal — it protects at rest, not from the
  operator or the coordinator the trace is shipped to.
- **Declassification UX.** `! {Declassify}` already exists; this doc does not change it.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Redaction is a pure function of the static type |
| A2: Replayability | −0/+1 | A redacted trace cannot replay a secret-dependent run. That is the intended trade, and the structure — names, arity, control flow — is retained |
| A3: Effect Legibility | **+1** | Effect spans keep showing which effects ran, without their payloads |
| A4: Explicit Authority | **+1** | The central point: a label that governs network sinks should govern the log too. One annotation, one guarantee |
| A5: Bounded Verification | 0 | No verification surface |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | **+1** | A generating model cannot see that its audit trail is a leak. The type system can |
| A8: Minimal Syntax | **+1** | No new syntax — reuses the labels M-TAINT-TYPES already has |
| A9: Cost Visibility | 0 | Neutral |
| A10: Composability | **+1** | Makes the label mean one thing everywhere instead of one thing at sinks |
| A11: Structured Failure | 0 | Neutral |
| A12: System Boundary | **+1** | The trace store *is* a boundary; today it is an unmarked one |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 · [x] A3 (**+1**) · [x] A4 (**+1**) · [x] A7 (**+1**)

## Related Documents

- [m-trace-tier-not-enforced](m-trace-tier-not-enforced.md) — the paired fix; ship first
- [m-list-cons-quadratic](../m-list-cons-quadratic.md) — cited only for the cost of touching the
  runtime value representation (Open Question 4)
- [m-trace-feedback](../v1_1_0/m-trace-feedback.md) (neural 0.46) — **distinct**: a *consumer* of
  traces for harness diagnostics, not what traces record
- `fb_a71eab12139bee26` — M-TAINT-TYPES Phase 2, on which Open Question 1 may depend

---

**Document created**: 2026-09-09
**Last updated**: 2026-09-09
