# #1771 M1–M2 repair validation

2026-10-29. Implementation evidence only; independent evaluation pending. Original execution was explicitly M1–M2 ONLY. M3 imported caller enforcement/CLI/REPL wiring and M4 completion remain unstarted; Refs #1134 and Refs #1132, no closure.

## Provenance and retained authority

Existing PR branch `coordinator/task-a87f9332` retains all original commits: M1 `581526060`, M2 `182b47859`, attended package identity/fixture repair `dba22e04e`, and interfaces `57084f55f`. Normal dev merge `3cf79f836` incorporates #1719/#2007; M1 repair `440fdd33a` and single value lookup followup `907dcd372` preserve the current callable checker. No historical dirty/untracked work was salvaged. Ratified F4 comment: https://github.com/sunholo-data/ailang/pull/1771#issuecomment-6419400823 . Cache epoch v8 was explicitly coordinated with the explicit-empty owner, who reserved none.

## Acceptance map

| Approved criterion | Concrete proof |
|---|---|
| M1 explicit clean/function/nonfunction/synthetic summaries | TestIFCSummaries, TestIFCSyntheticSummary, TestIFCSummaryBuilder |
| M1 constant/variable/positive/forbidden/declared/intrinsic/authority data | TestIFCSummaries, TestIFCSummaryNestedDeclaredResult, TestIFCSummaryF4CanonicalShape, codec fixtures with joined labels/multiple forbidden labels |
| M1 recursive intrinsic + supplied imported producer | TestIFCSummaries and TestIFCExportImportedValueLookup (alias/join/lexical shadow); import facts enter identVal once, remain opaque to imported-demand enforcement |
| M1 no partial/crashing summary on defensive limit; no private rho publication | TestIFCExportAnalysisLimitIsDiagnostic, TestIFCExportSummaryRejectsMissingImportFacts, TestIFCExportSummaryDoesNotPublishRho; ordinary error propagates through iface/minimal runtime builders |
| M1 deterministic digest/security fields | TestIFCSummaryDigestAndCodecs; hash-projection field mutations include applied and multiple forbidden labels; repeated -count=20 tests |
| M1 retain package identity | Attended dba22e04e pins hashProjectionSchema to v1 and omits only validated unlabelled summaries; TestHashProjection_UnlabelledSummaryKeepsPreSummaryIdentity proves byte equality. The historical eml 559080a1 observation is retained as prior attended evidence, not claimed as a fresh measurement of today's changed external package. |
| M2 full/compact/gob persistence | TestCacheIFCRoundTrip, TestIFCSummaryDigestAndCodecs, TestIFCSummaryF4CanonicalShape; every parameter reserves canonical bottom applied, while fixtures prove nonbottom joined applied metadata is lossless |
| M2 malformed/legacy fail closed | TestIFCSummaryF4RejectsMalformedParam, TestCacheIFCRejectsAuthenticatedBadSchema, TestCacheIFCInjectedMalformedF4Rejected, TestCacheIFCValidatesInjectedDecoder, TestHashProjectionRejectsMalformedIFC |
| M2 cache migration | TestCacheIFCEpochFollowsSharedDAG, TestCacheIFCPreviousVersionKeyMisses, TestCacheArtifacts_Migration; v7→v8 preserves existing CoreTI shared-DAG codec |
| M2 dependency edits/cold-warm metadata verdicts | TestCacheIFCLibraryEditsInvalidateCaller, TestCacheIFCLegacySchemaRecompiles, TestCacheIFCColdWarmLocalVerdict; refinement/intrinsic/declared-return-only edits invalidate caller keys |
| M2 published interfaces | All 49 interfaces regenerated; stripping IFC fields/schema from the comparison gives zero non-IFC type/effect signature changes versus dev. |

F4 failing-first evidence: old decoder rejected the valid forbidden array and accepted the obsolete forbidden string. The supplied-producer alias test failed before removing the old name-keyed imported-result branch. v8 epoch assertion failed while the source still used v7. Raw logs are banked as `1771-f4-red.log`, `1771-m1-import-value-red.log`, `1771-m2-epoch-red.log` under `/tmp/ailang-backlog-20261029/`.

## Local gates at the M2 checkpoint

- Full types/iface/pipeline/link/runtime packages passed. Full cmd package had only the intentional F4 stdout golden mismatch; repaired fixture and focused interface/cache/IFC command tests pass. Final integrated package gate is pending the independent effect/empty dev integration.
- Focused current IFC/fresh-CLI regression suites pass; no parser/HM/effect semantics changed here. `make test-core` passed on macOS with GOFLAGS=-p=1.
- `make lint`: zero issues. Size, boundaries and changelog gates pass.
- `make verify-examples`: 245 modules, zero drift/missing entries, manifest valid; all 49 stdlib interfaces stable after intentional F4 regeneration.
- 18 fresh CLI consumer comparisons against a freshly built actual dev `9dc6890919aa001deebee4361719456ee087440f`: zero status flips, zero first-diagnostic changes. Includes local IFC/leak lab controls, original inbox/secret examples and Gmail package check. This is a bounded differential, not a claim that every external corpus is clean.
- All new assertions are data/diagnostic based, with no native path separator assumptions or unguarded external solvers.
- No full `make test`, no coordinator dispatch/approval, no self-evaluation or merge.

## Reviewable large-module artifact evidence

The retained original candidates could not complete on current semantics:

1. Stapledon committed `81494c923d19cb02b97430a984f7230aaef75b70`, `sim/protocol_test.ail`: dependency `sunholo/social_dynamics@0.4.0/tasks.ail:73` fails `accept` with `internal invariant: missing callback row for typed application 1070 (argument 1)`. A fresh actual dev 9dc compiler reproduces both the consumer and direct dependency check. Saved dependency SHA256 `261faeb15fa083aed19e7f94dbe5af5e9b21273134c41ef2f90147fdaf92797a`; root/platform owner classifies it. No effect/consumer fix here.
2. Historical Motoko `b1ffa3397b0941bf6d9e59331a8c539e19784f8b`, original 6,340-line `src/core/session.ail`: fails the independently owned conservative mixed-record `fixture_rt` gate. Its lock also records old toolchain/content warnings; it is not claimed as a passing current consumer. Retry after actual explicit-empty integration and the separately approved consumer migration.

Both archives preserve committed source; live checkouts and their dirty work were untouched. No artifact ceiling was raised or bypassed.

A passing representative is Stapledon's 438-line navigation entry at the same pinned commit. Its actual transitive footprint is 30 modules, 4,962 source lines/274,917 source bytes, 29,473 CoreTI entries, 1,750 distinct DAG nodes summed per module, 384 exports, 54 exported types and 51 aliases. It exercises nested astronomical record aliases and data-heavy dependencies. Per-source hashes, every artifact byte count and per-module type counts are in the companion `m-ifc-cross-module-labels-repair-artifacts.json`.

Dedicated unchanged binary: cold 3.606s, warm 0.176s; warm 30 hits/0 misses and every artifact stamp hash/mtime unchanged. All artifacts total 6,988,878 bytes; IFC summaries 127,534 bytes. Maximum blob 937,929 bytes (1.40% of 64 MiB); maximum module 957,751 bytes (0.714% of 128 MiB). Navigation's own iface is 128,663 bytes, CoreTI DAG 30,193 bytes, full artifact set 548,709 bytes. These are real encoded current artifacts, not extrapolations of the pre-DAG plain-gob representation. They establish bounded metadata overhead for this representative; they do not replace the blocked original Motoko retry.

## Remaining integration

Normal-merge newer dev including #1849 and the effect-row/explicit-empty work only after it actually lands; retain its conversion helper move/digest fixes, regenerate v2 goldens, rerun affected exact-source gates and original Motoko measurement. Then push the existing PR branch after account verification and request independent evaluation. M3/M4 remain pending.
