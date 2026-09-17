# verify: VERIFIED not refutable when ensures calls a user function (asymmetric SMT gating)

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `uninterpreted`, `smt`, `callee sort gate`, `verify skipped`, design doc dirs `implemented/v0_30_0/m-smt-callee-sort-gate.md`, `planned/m-contract-verification-coverage.md`, `planned/m-verify-bounded-unrolling-false-counterexample.md`
- **Estimate**: n/a (design-doc)

The report is sound and the reproduction is precise: an `ensures` clause referencing a
user function (`mayWriteTo`) yields VERIFIED on the correct body, but the same clause
yields SKIPPED ("calls user function that is not SMT-encodable in this context") on a
deliberately broken body. That asymmetry proves the VERIFIED was not a refutation-backed
proof — the same encoding gap that blocks the refutation attempt must also block the
proof, or the proof path must encode the callee (uninterpreted function or inlined
definition). This is related to, but NOT covered by, M-SMT-CALLEE-SORT-GATE
(`design_docs/implemented/v0_30_0/m-smt-callee-sort-gate.md`), which gates *signature
sorts* of callees to avoid Z3 crashes and standardized the skip-with-reason path — it
does not address proof-vs-refutation asymmetry. M-CONTRACT-VERIFICATION-COVERAGE splits
the skip counter; also unrelated. No design doc found that rules on this (`none found`
for terms: verified refutable, ensures callee asymmetry).

Why design-doc, not direct-fix: there are at least two acceptable remedies with
different semantics and cost — (a) encode user callees in ensures as uninterpreted SMT
functions (stronger soundness, but uninterpreted symbols may make ensures vacuously
provable again unless the callee's own contract is conjoined — i.e. compositional
verification), and (b) make gating symmetric so both paths skip with the same reason
(weaker but honest). Choice (a) vs (b) changes the verifier's trust contract
(what VERIFIED means), which is exactly row 3/row 4 of the rubric. The implementer
should also confirm the mechanism in `internal/smt/callee_resolver.go` (`ResolveCallees`,
`IsSMTEncodableForCallee`) and wherever the proof obligation for ensures is encoded,
since the report shows the proof path reaching a verdict the refutation path cannot
even attempt. Daneel's positive-control rule (break each contract, expect VIOLATION)
is the detection pattern worth referencing in the doc.

**TRIAGE_FILE:** `design_docs/planned/ailang-core-triage/verify-ensures-verified-without-encodable-refutation.md`
**RECOMMEND:** `design-doc`

TRIAGE_FILE: design_docs/planned/ailang-core-triage/verify-ensures-verified-without-encodable-refutation.md
RECOMMEND: design-doc
