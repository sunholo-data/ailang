# M-SMT-PARAMETRIC-ADT-DECLARATIONS: retain concrete error payload types

**Status**: Approved — implementation in progress (Mark, 2026-10-23)
**Target**: v0.54.18 candidate, separate from completed M-AUDIO-RELEASE
**Priority**: P1 (verifier correctness)
**Estimated**: 2–4 engineering days, including refutation and declaration-corpus tests
**Dependencies**: existing SMT declaration closure and typed Core information
**Created**: 2026-10-23

## Problem and verified scope

Canonical report `inbox_1792745981583_2823c9cc` from ailang-packages describes
`checkSize` in sunholo/audio clips: `Result[int, AudioError]`, with an imported
`ResourceLimit(string)` error constructor. The contract accepts all Err values
and requires successful counts to be nonnegative. Current AILANG emits no
AudioError declaration and declares Result's Err field as Int. Z3 rejects the
ResourceLimit application, then result; the function remains unproved.

This was reproduced on the completed audio implementation, not inferred from
an older report. `ailang check` succeeds; `verify --verbose --json` reports
0 verified, 1 error, 0 counterexamples. The emitted script contains:

```text
(declare-datatype Result ((Ok (Ok_0 Int)) (Err (Err_0 Int))))
```

Adding the ResourceLimit declaration alone would still leave the Err payload's
sort wrong. The affected path erases concrete generic arguments before filtering
and declaration generation. It must retain both declaration dependencies and
type-parameter bindings, without guessing missing types as Int.

The original report's Result[(), AudioError] also encounters unsupported unit
literals. Unit-value encoding is not required to fix the count-returning API.
The package keeps its useful contract; removing it or relabeling an error as a
proof is not a remedy.

## Reproduction

The following exact source type-checked and reproduced the error locally in a
scratch package (manifest name resource_probe/audio, edition1, exports both
modules). These are existing AILANG constructs, not proposed syntax:

```ailang
module resource_probe/audio/safety
export type AudioError = ResourceLimit(string) | InvalidFormat(string)
```

```ailang
module resource_probe/audio/main
import std/result (Result, Ok, Err)
import ./safety (AudioError, ResourceLimit)
export pure func checked(n: int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(n) }
```

`bin/ailang check /tmp/ailang_resource_probe/main.ail` succeeds. The corresponding
`verify --verbose --json` reports `unknown constant ResourceLimit (String)`.
Source baseline: completed audio commit a8bc30fc9 (metadata completion eabefc965).

## Goals and non-goals

Prove the valid count guard under its declared types, refute a broken guard,
and emit only well-sorted declarations for the reachable concrete ADT types.
Keep the Real/Int numeric-model qualification from the completed reporting work.

No unit-value support, higher-order types, runtime/error API changes, benchmark
KPI/status split, recursion proof expansion, or implicit broadening of the
existing cross-function callee-sort gate. Unsupported or ambiguous type shapes
must be skipped with a typed unsupported_encoding reason before invoking Z3.

## High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Preserve ADT type parameters; emit Z3 parametric datatypes | Defines the logical meaning of Result/Option fields | human approval | design | high |
| Preserve complete concrete sort applications through the verifier | Erasure can cause errors or misleading proofs | human approval | design | high |
| Constructor qualification follows checked node types | Nested instances cannot share an assumed outer return sort | human approval | design | high |
| Missing/ambiguous types produce structured skips | Prevents fake Int defaults and arbitrary declaration selection | human approval | design | med |
| Leave callee encodability expansion separate | Avoids silently bypassing an established guard | human approval | design | med |

### Design freeze

- [x] Approve the typed parametric encoding and conservative unsupported boundary.
- [x] Approve constructor type-context plumbing and unchanged callee gate.
- [x] Approve refutation, nested-instance and declaration-closure release gates.

## Solution design

### Typed declaration input

Extend the SMT declaration model to preserve each ADT's type parameters and its
field types before conversion. Use an explicit type-variable environment when
rendering a generic definition; its A/e binders are formal SMT sorts, not Int.
A concrete AST TypeApp such as Result[int, AudioError] renders the full sort
application `(Result Int AudioError)`. Keep the existing sequence representation
for lists and monomorphic declaration output where the semantics are unchanged.
Unresolved variables and unsupported shapes return typed errors, not defaults.

Z3 accepts this directly; the following declaration/constructor form was tested:

```text
(declare-datatype AudioError ((ResourceLimit (ResourceLimit_0 String))
                            (InvalidFormat (InvalidFormat_0 String))))
(declare-datatypes ((Result 2))
  ((par (A E) ((Ok (Ok_0 A)) (Err (Err_0 E))))))
; Concrete result sort: (Result Int AudioError)
; Typed construction: ((as Err (Result Int AudioError)) (ResourceLimit "negative"))
; Typed construction: ((as Ok (Result Int AudioError)) n)
```

A standalone Z3 script with the valid guard returns unsat for the negated
postcondition. Replacing the successful value with -1 returns sat. This validates
the proposed logical form; it is not a claim that AILANG already emits it.

### Dependency closure

Seed the existing demand closure from full parameter/return types, body nodes,
and requires/ensures expressions. Resolve constructor references to owning ADTs,
including nested constructor patterns, match guards and callee bodies that are
already admitted by the existing gate. Preserve the dependencies of generic type
arguments and ADT fields. The current constructor-name seed alone does not map
ResourceLimit back to AudioError; merely widening the map to all imported ADTs
would reintroduce unrelated-declaration cascade failures.

Keep declarations deterministically ordered and reuse the existing dependency
SCC/group machinery for recursive types. Do not satisfy a missing sort by keeping
an unrelated alias. Resolve qualified ownership from the imported module graph;
if an ambiguous short name cannot be represented safely, skip that function with
a diagnostic instead of selecting a declaration by map iteration.

### Checked constructor context

Thread the existing pipeline CoreTI into the verifier's per-function encoding
options where needed. Use checked signatures and node types to determine each
constructor application's concrete result sort. Propagate expected sorts through
branches/lets and qualify polymorphic constructors when Z3 needs disambiguation.
Nested Result/Option instances must not inherit one global outer return hint.
If type information is unavailable or not ground, reject that encoding explicitly.

Run structural well-sortedness checks before invoking the solver. Preserve
ordinary solver errors/unknown/timeouts as unproved. Existing callee signature
rejections stay in place unless a separately tested, explicit follow-up design
opens that path; this sprint must not make old unsupported callees appear proved.

## Implementation phases and files

1. Typed ADT/sort declarations and dependency closure (~1 day):
   internal/smt/from_ast.go, types.go, filter.go and declaration graph helpers.
   Preserve metadata for generic binders; remove Int defaults on the new path.
2. Constructor type context (~1 day): internal/smt/verify.go, codegen.go,
   codegen_apps.go and contract/body traversal. Thread pipeline CoreTI from CLI
   verify/ai-check entry points if required; preserve the shared driver behavior.
3. Regression corpus and docs (~1–2 days): isolated package repro, single-file
   and package JSON checks, true and deliberately false contracts, nested generic
   instances, imported type dependencies, deterministic declarations and full gates.

The implementer may choose internal structure/helper names and test file layout.
Using native parametric declarations is the proposed direction; replacing them
with per-function ad-hoc Int special cases is not delegated.

## Acceptance criteria and conflict surface

- [ ] The exact imported ResourceLimit count-guard repro is verified with no
      unknown constants, missing sorts, unsupported substitutions or solver errors.
- [ ] Replacing valid successful n with -1 produces a genuine counterexample;
      a broken contract cannot be skipped on a path the valid one proves.
- [ ] Err's payload is AudioError and Ok's payload is Int in the actual script;
      preserving only one of these properties fails the test.
- [ ] Requires/ensures-only ADT references, nested patterns, imported declarations,
      and nested/two distinct generic instances retain correct field sorts.
- [ ] Unsupported unit/higher-order/open-type cases skip honestly; callee gate,
      legacy status/exit policies and numeric-model labels stay explicit.
- [ ] Unrelated imported declaration failures stay isolated; repeated encodings
      are byte-identical, including mutual dependency groups.
- [ ] Focused SMT/CLI tests and required integrated tests/lint/boundaries pass;
      Z3-dependent tests guard availability on Windows.

No language syntax, runtime evaluation or new error API. The conflict surface is
SMT signature translation, ADT declarations, constructor/pattern encoding and
shared verify/ai-check call paths. Existing anchors to retain include
internal/smt/codegen_decl_closure_test.go (record-to-ADT and mutual groups),
internal/smt/filter_test.go (unrelated aliases), and
cmd/ailang/verify_callee_sort_gate_e2e_test.go (unsupported callee cases).
The contract roots are mandatory inputs; this must not repeat a body-only traversal.

## Verification log (2026-10-23)

| Premise | Evidence | Result |
|---|---|---|
| Report reaches core | Canonical message list JSON, no ack/read side effect | Full self-contained upstream report and correlation ID |
| Source is valid AILANG | Exact scratch package source above; bin/ailang check | Pass |
| Current failure | bin/ailang verify --verbose --json | 0 verified/1 error; ResourceLimit undeclared |
| Generic arguments erased | Read ASTTypeToSMTSort TypeApp branch | Returns constructor name alone except list |
| Generic fields flattened | Read ExtractADTTypesWithRecords and ASTTypeToSMTSort fallback | Extracted Result has both fields Int in actual script |
| Filter omits necessary ownership | Read collectSortsFromBody and buildNeededSortSet | No Var/VarGlobal constructor-owner lookup; signature args already erased |
| Type information exists upstream | Read pipeline.Artifacts.CoreTI in pipeline.go | Existing types.CoreTypeInfo available for plumbing |
| Proposed Z3 form | Executed scratch result.smt2 and deliberately broken.smt2 | unsat for valid guard; sat for broken success value |
| Related design is distinct | Read implemented v0_30_0 m-smt-callee-sort-gate | Gates callees; does not repair this function's own generic result declaration |
| Duplicate search | Scaffold SimHash/neural plus direct parametric/ResourceLimit search | Neural fallback is not trustworthy similarity evidence; no equivalent design found |

## Axiom compliance and risks

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Ordered typed declaration closure |
| A2 Replayability | +1 | Inspectable actual typed SMT script |
| A3 Effect legibility | 0 | Static pure verification only |
| A4 Explicit authority | 0 | No runtime capabilities |
| A5 Bounded verification | +1 | Ground typed fragment or explicit skip |
| A6 Safe concurrency | 0 | No concurrency semantics change |
| A7 Machines first | +1 | Types and unsupported reasons retained |
| A8 Minimal syntax | 0 | Existing AILANG and SMT constructs |
| A9 Cost visibility | 0 | Solver budgets preserved |
| A10 Composability | +1 | Result and imported error types agree |
| A11 Structured failure | +1 | Reject malformed encoding before solver |
| A12 System boundary | +1 | Encoded types match checked language boundary |

Net +7; no A1/A3/A4/A7 hard violation. Main risks are constructor ambiguity,
nested instance type drift, unrelated declaration pollution and false-positive
proofs; typed node context, bounded rejection and negative-control tests are gates.

## Related documents

- [Callee-sort gate](../../implemented/v0_30_0/m-smt-callee-sort-gate.md).
- [Completed audio sprint](../../implemented/v0_54_18/m-audio-release-sprint-plan.md).
- [Verification coverage addendum](../m-contract-verification-coverage.md#audio-follow-up-2026-10-23-additive-package-evidence): reporting only, not this encoding fix.
- [Design axioms](../../../docs/docs/references/axioms.mdx).
