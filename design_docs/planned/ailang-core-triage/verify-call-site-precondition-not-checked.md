# verify: callee `requires` never discharged at call sites (inlined-body verification is unsound)

- **Date**: 2026-09-29
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `precondition`, `requires`, `call site`, `inline callee` across `design_docs/`; checked `planned/m-contract-verification-coverage.md`, `planned/m-verify-bounded-unrolling-false-counterexample.md`, `implemented/v0_30_0/m-smt-callee-sort-gate.md`, and the existing triage row `verify-ensures-verified-without-encodable-refutation.md`. No doc rules on call-site precondition obligations (`none found` for terms: requires obligation, call site, precondition discharge).
- **Estimate**: n/a (design-doc)

Reproduced at v0.47.0 (f5bf729): a caller with `requires { n < 0 }` calling `half(n)` where `half` declares `requires { n >= 0 }` reports VERIFIED on both functions. The mechanism is exactly as the report states, and the code itself admits the gap:

1. **Encodable callees are inlined body-only.** `internal/smt/callee_resolver.go` (`ResolveCallees` → `buildDefineFun`) emits a `define-fun` for the callee body and never consults its `meta.Contracts`; the caller's VC is checked against the inlined body with no `requires` obligation generated at the call site.
2. **The contract fallback path omits requires entirely.** `callee_resolver.go:~127` — "Requires clauses are skipped because they reference param names that cannot be substituted without call-site arg tracking (**deferred to M4**)".
3. **A third, related unsoundness:** `internal/smt/codegen_xmod_contract.go` (`EncodeCalleeByContract`) *does* have param→arg substitution and asserts callee `requires` at the call site — but as an **axiom** (assumed), not a proof obligation. A caller violating the precondition makes the callee's asserted `ensures` vacuously supportive of the caller's proof. So the substitution machinery the fix needs already exists; what's missing is the obligation/verdict plumbing.

Why design-doc, not direct-fix: this changes the verifier's trust contract — what VERIFIED means for any module that uses `requires` (row 4), and there are at least three acceptable remedies someone could disagree over (row 3): (a) emit `requires_f(args)` as a checked obligation under the caller's path condition for every call (compositional assume-guarantee, matching the reporter's expected behaviour and giving counterexamples at the caller); (b) restrict inlining/contract-fallback to require-free callees and SKIP with a reason otherwise (weaker but honest, mirrors the callee-sort-gate approach); (c) assert requires as axioms **plus** a disjunction-of-calls side condition. The remedy choice also interacts with two parked docs that edit the same verdict/counterexample plumbing — `m-verify-bounded-unrolling-false-counterexample.md` (sat-vs-violation ladder, `cmd/ailang/verify.go`) and `m-contract-verification-coverage.md` (skip classification) — so whichever lands needs textual coordination, and the M4 deferral comment in `callee_resolver.go` should be resolved or superseded by the doc. The real-world impact case (sunholo/economic `roundDiv1e8` int64 bounds passing through widened callers) is worth recording in the doc as the motivating scenario.

**TRIAGE_FILE:** `design_docs/planned/ailang-core-triage/verify-call-site-precondition-not-checked.md`
**RECOMMEND:** `design-doc`