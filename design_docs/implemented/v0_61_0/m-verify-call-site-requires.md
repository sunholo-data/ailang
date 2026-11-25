# M-VERIFY-CALL-SITE-REQUIRES — check a callee's `requires` at each call (program M3)

**Status**: Implemented, shipped in v0.61.0 (see §Result). **Program**: [M-VERIFY-SOUNDNESS-COVERAGE](../../planned/v0_61_0/m-verify-soundness-coverage-program.md) M3 (gap S1).
**Ruling**: D1 = (a), Mark, 2026-11-24: a call that may violate its callee's `requires` is a
`counterexample` with reason code `requires_violation`.
**Quorum**: no trigger fires. D1 is ruled, so there are no freeze items; the change is local to
`internal/smt` (no shared machinery overridden); no cost, KPI or banked-data schema; every premise
is in-repo and logged below.

## Problem

`ailang verify` assumes what a callee guarantees but never checks that the caller's arguments meet
the callee's `requires`. `caller(n) requires { n < 0 }` calling `one(n)` with `requires { n > 0 }`
reports both VERIFIED (program V1). At runtime the call fails its precondition.

**Blast radius (M0 gap map, 2026-11-24):** 20 results reach a callee with a `requires`, 11 of them
VERIFIED. The four downstream ones discharge the callee's `requires` from their own (read by hand),
so they should still verify; the change adds checks, it is not expected to flip downstream proofs.

## Design

**Obligations ride the call-facts channel.** Call facts (`codegen_call_facts.go`) already carry
exactly the path scoping an obligation needs: facts from an `if` branch are wrapped as
`(ite c then-facts else-facts)`, facts under a `let` or `match` arm are wrapped in that binder, and
facts never escape a helper's formal parameters. An obligation has the same algebra (a conjunction
of guarded clauses, `true` on a path not taken). So a call to `g` in the verified function's body
adds the fact `(=> $req_ok_g ($req_g args))`, scoped by the same wrappers. `$req_ok_g` is never
declared: every assertion that mentions it binds it with `let`.

- **Postcondition query** (today's query): facts are asserted under `$req_ok_* = true`, so callee
  preconditions are *assumed* at each call. This is assume-guarantee: sound because they are checked
  separately, below. A model that broke the postcondition only by calling `g` outside its
  precondition is reported as the precondition violation, not as an ensures failure.
- **One obligation query per callee `g`**: the same declarations, the caller's `requires`, every
  fact under `$req_ok_* = false` (assumptions only), and
  `(assert (not (let (($req_ok_g true) …others false…) (and facts…))))`. Sat means some call to `g`
  on a reachable path breaks `g`'s `requires`.

**Where `$req_g` comes from.**
- *Inline callees* (`define-fun`, `ResolveCallees`): a `(define-fun $req_g (params) Bool …)` built
  from `g`'s `requires`, encoded under the same `$a_` parameter renames as its body.
- *Contract callees* (`ContractSpec`): the `requires` conjunction is already encoded per call with
  argument substitution (`encodeContractExprWithSubs`). The obligation reuses it.
- If `g`'s `requires` cannot be encoded, a call to it cannot be checked: the caller is **skipped**
  with that reason (fail closed), never VERIFIED.

**Verdicts.** Obligation queries run only when the postcondition query verified. The first
obligation query that is sat makes the result `counterexample`, `reason_code: requires_violation`,
naming the callee. The same post-processing as a postcondition counterexample applies: a model
that may rest on an unrolling frontier is `inconclusive`, and so is one over an uninterpreted
builtin. Unknown makes the result `unknown`.

**Scope of v1 (stated in the docs, not silent):** obligations are emitted for calls in the verified
function's *own body*. *Update, same day:* the first three below are now checked, through
per-definition `$ob_` predicates (program doc, "M3/M4 inside definitions"); calls inside
`requires`/`ensures` still are not. The v1 list was:
- calls inside a helper's body, which are checked when that helper is verified, if it has a contract;
- calls inside a lambda passed to `map`/`filter`/`foldl`;
- calls inside the unrolled body of a recursive function, including its own recursive call;
- calls inside `requires`/`ensures`.

Each of these sits inside a `define-fun` or a contract, where a free obligation symbol cannot
appear. A guard refuses any declaration that would contain one.

## Verification log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Call facts are scoped by `if`/`let`/`match` | `encodeFactIfHint`, `encodeFactLet` (`codegen_call_facts.go`), match arms (`codegen_call_facts_control.go:31,41`) |
| V2 | Helper bodies are encoded before the contracted body | `resolveCalleesFromRoots` runs in `EncodeFunction` before Step 4; unrolling at Step 1.6 |
| V3 | Resolved calls already add a per-call fact from a helper `define-fun` | `encodeResolvedApplication` + `activeHelperConditions` (`codegen_call_facts_helpers.go`) |
| V4 | Contract callees already encode `requires` with argument substitution | `instantiateContract` → `encodeContractExprWithSubs` (`codegen_xmod_contract.go`) |
| V5 | `requires_violation` is not an existing reason code | `internal/verification/evidence.go` lists none; `git grep requires_violation` finds only design docs |

## Acceptance

- The S1 probe: `caller` is `✗ VIOLATION`, `requires_violation`, naming `one`; `one` stays verified.
- A call under a branch whose condition implies the callee's `requires` still verifies (path scoping).
- A call under a `let` whose bound value meets the `requires` verifies; one that does not is a violation.
- A contract-path callee (body not encodable, contract encodable) is checked the same way.
- An unencodable callee `requires` skips the caller with a reason.
- Mutation: drop the obligation emission → the S1 test fails.
- Gap map: no downstream VERIFIED flips (or each flip is explained as a real caller bug).

## Result (2026-11-24)

- `internal/smt/codegen_obligations.go` (channel, query assembly), `verify_obligations.go`
  (verdicts); `$req_g` built in `callee_resolver.go`; contract path in `codegen_call_facts.go`.
- `call_requires_test.go` pins S1 (`caller` VIOLATION), branch and `let` scoping, the contract path,
  fail-closed `requires`, and a callee contract that calls a `requires` function unguarded (the
  caller is not blamed). Mutation-tested: disabling obligation emission fails three cases; removing
  the contract-template guard fails the last.
- Gap map: no verdict changed across 765 results. The 11 verified results that call a function with
  a `requires` all had every such call checked (`requires_checked`); 0 unchecked.
