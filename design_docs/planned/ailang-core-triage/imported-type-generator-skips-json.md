# Imported-type property generator skips (std/json Json) misclassified as vacuous

- **Date**: 2026-09-15
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/implemented/v0_31_0/m-property-generator-coverage.md
- **Searched**: deriveNamedType/no_generator (code: `internal/testing/derive.go`), "generator imported Json" and "Lane A no_generator" in design_docs/
- **Estimate**: omitted (duplicate-of)

The mechanism the report names is real and located correctly — `deriveNamedType` (`internal/testing/derive.go`) resolves only `r.executor.sourceFile.Decls`, same-file, and returns nil for anything else — but that bound is a **recorded design decision, not an unnoticed bug**. `design_docs/implemented/v0_31_0/m-property-generator-coverage.md` states twice: "imported/cross-module named types stay vacuous-skip in v1 (honest, loud)" (design decision table) and "Refined types and imported types remain B2/out-of-scope" (Lane B scope, V32-verified). The doc also shows the path forward for the report's main ask: Lane B2 — user-supplied/registered generators — is the sanctioned remedy for types the deriver can't reach (Json included), and is **deferred, BLOCKED ON a deterministic evaluator fuel/step budget** (quorum 2026-07-29), because deriving Json structurally would require evaluating inside stdlib modules the runner doesn't hold the source for.

The report's alternative ask — classifying imported-type skips separately from vacuous skips in `ailang test` exit-code semantics — is NOT covered by that doc (it covers ai-check verify skips, a different subsystem) and is a genuine gap: it changes `Success()`/exit-code contract, a gate's semantics, so per rubric it would be `design-doc`. But as filed, the report's headline (derive for imported stdlib types / document the workaround) is answered by the existing doc: the workaround is B2's deferred generator registration; the impact numbers (4/12, 7/8 skips on sunholo packages) are good new evidence for raising B2's priority, not new scope.

**Suggested next step if Daneel wants action**: a small design doc to (a) un-defer or part-deliver Lane B2 (registered generators for stdlib types like Json), or (b) add a distinct `skip_kind` (e.g. `no_generator_imported`) so structural skips don't fail `ailang test` package suites — cite this triage row plus the impact numbers in the package AGENT.md files.
