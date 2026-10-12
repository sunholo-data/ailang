# M3 validation: pure rows and SMT admission

Refs #1443. Continues M1–M2 and the attended fix at `af0fd073` from PR #1782. M4 remains unstarted.

## Implementation

SMT admission uses the keyword or an explicit closed row containing only Declassify (including `Declassify[label=x]`) or no effects. Entry functions and callees share the predicate. Surface declarations are carried separately, keyed by Core metadata identity so imported declarations keep their provenance. Verification never changes `IsPure`. Contract fallback cannot bypass the row gate.

Seven contradictory declarations in the two inbox examples and two prompt-injection references remove `pure`. All four files are formatted; existing inline attack comments are preserved above their declarations because the formatter refuses their original positions. IFC fixture callers declare Declassify, and MCP coverage pins scoped Declassify as effectful.

HashProjection's constant purity pin and cache version v7 are unchanged. No full `make test`, push, prompt migration, bundle generation or M4 work was performed.

## Validation

- Admission matrix: keyword, missing row, explicit empty row, bare/scoped Declassify, IO, Debug, mixed rows and open rows.
- Cross-module scoped Declassify helper inlines and verifies; no-row pure admission remains valid.
- Safe reference: 3 verified, 0 violations, 0 skipped, 0 errors.
- Injected reference: 2 verified, 1 violation on `injectedForward`, 0 skipped, 0 errors.
- A failing regression demonstrated IO callee admission bypass through contract fallback before the fix; it passes after the gate.
- Z3-dependent tests guard availability for Windows. No path-output assertions depend on native separators.
- `make verify-examples`: pass (246 passed, 0 failed, 9 skipped); both inbox v2 examples pass. Manifest: 228 modules checked, zero drift; one pre-existing missing-on-disk entry does not fail the gate.
- `tools/verify-stdlib.sh`: pass, all 48 interfaces stable.
- `make test-core`: pass under the temporary subreaper.
- `make check-file-sizes`, `make check-boundaries`, `make check-changelog`, source formatting and `git diff --check`: pass.
- Admission helpers `smtAdmissible` and `rowAdmissible`: 100% statement coverage each.
- `go test ./internal/smt ./internal/types ./internal/apiserver`: pass under the temporary subreaper.
- `make lint`: pass, zero issues.

The container's PID 1 leaves killed descendants as zombies. Existing SMT timeout tests failed on that environment condition; the focused tests and core gate run under a temporary Linux subreaper outside the repository, as in M2. No timeout/runtime implementation was changed.

## Coordinator PR body

Preserve SMT verification after checked pure signatures reject effectful declarations. Migrate seven Declassify signatures and admit explicit closed Declassify-only rows independently of purity, including scoped labels and cross-module callees. Safe and injected prompt-injection outcomes remain pinned.

Refs #1443. This PR supersedes #1782 and retains its M1, M2 and attended purity-hash fix. M3 is complete; M4 remains for a later run. HashProjection's purity pin and cache v7 remain unchanged.

Work branch: `coordinator/task-c0bf79fb`. Local commit only; coordinator owns PR publication.
