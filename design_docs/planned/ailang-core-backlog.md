# AILANG Core Backlog (triage rows)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Ensures calling a user function: VERIFIED on correct body but SKIPPED on broken body (proof/refutation asymmetry) | bug | design-doc | The verifier proves via one encoding (or inlining) and fails to refute via another, so VERIFIED is unsound whenever it cannot be mirrored by a counterexample search; fixing it needs a decision — encode callees as uninterpreted functions, inline them consistently on both paths, or add a consistency gate that demotes VERIFIED to SKIPPED when refutation encoding fails — and existing coverage (`m-contract-verification-coverage.md`) only splits skip taxonomy, not this asymmetry (searched: smt, ensures, uninterpreted, skipped). |
