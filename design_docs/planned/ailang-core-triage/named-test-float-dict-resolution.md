# Named-test bodies fail float comparisons — missing Fractional dictionary in lowered path

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `named test`, `named-test` in design_docs/ (found m-named-test-blocks, m-module-let-func-resolution, m-property-generator-coverage, m-ci-flake-systemic-fix — none covers dictionary resolution in test bodies); `Fractional`, `missing dictionary method`, `expected float arguments` in design_docs/ and internal/
- **Estimate**: omitted (design-doc)

The report describes `test "name" { ... }` bodies that compare floats failing at runtime with `missing dictionary method: prelude::Fractional::Int::add` (and `expected float arguments` for direct `==`), while the identical expression works under `ailang run`. The mechanism is in `internal/testing/executor.go` (`EvaluateNamedTestBodyExprs`): named-test bodies are not evaluated directly — they are printed back to AILANG source (`PrintAILANGSource` via `FoldTestBody` in `internal/testing/test_body_lowering.go`), appended to the stripped module source, re-elaborated through the pipeline, and evaluated as Core. Somewhere in that re-elaboration the typeclass dictionary wiring for numeric operations on floats diverges from the standard `ailang run` path (likely numeric literal defaulting / dictionary passing at elaboration of the appended free expression, which must be wrapped in a `Block` because `ElaborateFile` skips top-level `*ast.Let`). The area is documented in `design_docs/implemented/v0_29_0/m-named-test-blocks.md` (the lowering pipeline) but that doc does not rule on dictionary resolution, and no planned doc covers it.

This is not a direct fix: the divergence sits in elaboration semantics for the synthetic module the test executor builds, the fix could live in the executor's source construction or in elaboration proper, and getting float defaulting right in one path but not the other is exactly the kind of semantics decision (row 3/4) that needs a design doc. There is also prior art of the same *shape* of bug — `m-wasm-typecheck-float-divergence` (v0.22.0) and `m-poly-ord-defaulting-regression` (v0.11.4) were both float-defaulting divergences between evaluation paths — worth consulting for the pattern.

Existing workarounds noted in the report (int-only comparisons, contract clauses) are reasonable stopgaps until fixed.
