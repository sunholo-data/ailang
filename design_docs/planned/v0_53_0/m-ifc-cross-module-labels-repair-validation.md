# #1771 M1–M2 repair validation

2026-10-29. Implementation evidence only; independent evaluation pending. Original execution was explicitly M1–M2 ONLY. M3 imported caller enforcement/CLI/REPL wiring and M4 completion remain unstarted; Refs #1134 and Refs #1132, no closure.

## Provenance and retained authority

Existing PR branch `coordinator/task-a87f9332` retains all original commits: M1 `581526060`, M2 `182b47859`, attended package identity/fixture repair `dba22e04e`, and interfaces `57084f55f`. Normal dev merge `3cf79f836` incorporates #1719/#2007; M1 repair `440fdd33a` and single value lookup followup `907dcd372` preserve the current callable checker. No historical dirty/untracked work was salvaged. Ratified F4 comment: https://github.com/sunholo-data/ailang/pull/1771#issuecomment-6419400823 . Cache epoch v8 was explicitly coordinated with the explicit-empty owner, who reserved none.

## Acceptance map

| Approved criterion | Concrete proof |
|---|---|
| M1 explicit clean/function/nonfunction/synthetic summaries | TestIFCSummaries, TestIFCSyntheticSummary, TestIFCSummaryBuilder |
| M1 constant/variable/positive/forbidden/declared/intrinsic/authority data | TestIFCSummaries, TestIFCSummaryNestedDeclaredResult, TestIFCSummaryF4CanonicalShape, codec fixtures with joined labels/multiple forbidden labels |
| M1 recursive intrinsic + supplied imported producer | TestIFCSummaries, TestIFCExportImportedValueLookup (alias/join/container/lexical shadow) and TestIFCExportLocalProducerValues (direct/alias/join/returned factory/shadow/storage/typed handoff); facts enter identVal once, remain opaque to imported-demand enforcement |
| M1 no partial/crashing summary on defensive limit; no private rho publication | TestIFCExportAnalysisLimitIsDiagnostic, TestIFCExportSummaryRejectsMissingImportFacts, TestIFCExportSummaryDoesNotPublishRho; ordinary error propagates through iface/minimal runtime builders |
| M1 deterministic digest/security fields | TestIFCSummaryDigestAndCodecs; hash-projection field mutations include applied and multiple forbidden labels; repeated -count=20 tests |
| M1 retain package identity | Attended dba22e04e pins hashProjectionSchema to v1 and omits only validated unlabelled summaries; TestHashProjection_UnlabelledSummaryKeepsPreSummaryIdentity proves byte equality. The historical eml 559080a1 observation is retained as prior attended evidence, not claimed as a fresh measurement of today's changed external package. |
| M2 full/compact/gob persistence | TestCacheIFCRoundTrip, TestIFCSummaryDigestAndCodecs, TestIFCSummaryF4CanonicalShape; every parameter reserves canonical bottom applied, while fixtures prove nonbottom joined applied metadata is lossless |
| M2 malformed/legacy fail closed | TestIFCSummaryF4RejectsMalformedParam, TestCacheIFCRejectsAuthenticatedBadSchema, TestCacheIFCInjectedMalformedF4Rejected, TestCacheIFCValidatesInjectedDecoder, TestHashProjectionRejectsMalformedIFC |
| M2 cache migration | TestCacheIFCPreviousVersionKeyMisses (all prior epochs v1–v7, permitting future migrations), TestCacheArtifacts_Migration; v7→v8 preserves existing CoreTI shared-DAG codec |
| M2 dependency edits/cold-warm metadata verdicts | TestCacheIFCLibraryEditsInvalidateCaller, TestCacheIFCLegacySchemaRecompiles, TestCacheIFCColdWarmLocalVerdict; refinement/intrinsic/declared-return-only edits invalidate caller keys |
| M2 published interfaces | All 49 interfaces regenerated; stripping IFC fields/schema from the comparison gives zero non-IFC type/effect signature changes versus dev. |

F4 failing-first evidence: old decoder rejected the valid forbidden array and accepted the obsolete forbidden string. The supplied-producer alias test failed before removing the old name-keyed imported-result branch. A separate failing-first record-container test confirms the supplied producer retains the callee value's existing flow label. The final cache regression uses valid old keys for every v1–v7 epoch, without pinning future migrations to v8. Raw logs are banked as `1771-f4-red.log`, `1771-m1-import-value-red.log`, `1771-import-callee-label-red.log` under `/tmp/ailang-backlog-20261029/`.

## Local gates at the M2 checkpoint

- Official `make test-changed` passes on clean 8f6b11d0f: all six changed packages, including full cmd 238.520s and pipeline 22.815s, then the entire language-core loop. This includes the corrected F4 stdout golden.
- Current IFC/fresh-CLI regressions and core pass on macOS with GOFLAGS=-p=1. No parser/HM/effect semantics are implemented by this IFC repair; independently owned effect changes are integrated unchanged.
- `make lint`: zero issues. Size, boundaries and changelog gates pass.
- `make verify-examples`: 245 modules, zero drift/missing entries, actual manifest: 252 total / 244 working / 4 broken / 4 aspirational valid; all 49 stdlib interfaces stable after intentional F4 regeneration.
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

## Final combined integration

Normal dev merge 8f6b11d0f incorporates actual #2005/80793, preserving its moved iface converter, explicit-empty flags/rendering and std/ai callback rows. All 49 v2 interfaces were regenerated; stripping only IFC/schema gives zero type/effect signature differences against actual 80793. No old prompt bytes or registry entries were changed by this repair. Example manifest counts are parsed from the actual combined tree rather than copied from older checkpoints.

The original delivered Motoko gate now passes on exact clean 8f6b11d0f and consumer 083ef437: `gmake check_core` reports 60/60 core, 9/9 extension boots and runtime prerequisites. Its 5,824-line session checks cold 43.982s (143 misses) and warm 1.610s (143 hits/0 misses), with every stamp hash/mtime unchanged. The full 143-module graph is 58,174 source lines, 186,493 CoreTI entries, 19,296 distinct DAG nodes summed per module and 1,365 exports. Actual total artifacts 52,171,074 bytes, IFC payload 473,459 bytes; maximum blob 3,100,965 bytes < 64 MiB / maximum module 3,400,869 bytes < 128 MiB. Session itself has 256,531-byte DAG, 1,555,276-byte Core and 361,513-byte iface; no ceiling bypass. Source hashes and exact per-module sizes are retained in the companion artifact JSON.

Published alias dependency #2008 head d5f783585 was normally merged into this existing PR at a9971a364, with no competing implementation. Root later merged #2008 normally at 749045717; ancestry-only respinning is unnecessary because the identical tested source is already contained. Affected alias/IFC/cache/interface/CLI tests pass on the combined a997 source, including generic aliases and declared-tail controls. All 49 accepted v2 interfaces remain stable after that integration.

Original Stapledon 81494c923 / social_dynamics 0.4.0 proof now passes on a997: direct dependency NoCache 2.250s; protocol cold 18.678s (97 misses), warm 1.332s (97 hits/0 misses), unchanged stamp hashes/mtimes. Maximum blob 3,606,290 bytes / maximum module 4,664,163 bytes, below unchanged ceilings. The original actual-dev80793 failure and source SHA remain preserved as counterfactual evidence. This focused post-alias proof is separate from the full 8f Motoko/native proof; no duplicate full core/Motoko run is claimed for the alias integration.

Final PR publication uses the required bot account and existing branch, without force. Root owns independent final acceptance/CI and merge. Overall sprint stays in_progress; M3/M4 remain null/unstarted; Refs #1134 and Refs #1132, no issue closure or completion archival.

## Independent M1 producer-value correction

Root's RED at 4d25ee178 found a falsely bottom local producer alias; subsequent REDs covered returned values and a known callback passed to a local helper. Root classified the final repair under the already-approved conservative closure/APP-PURE rule: in M1 export analysis only, named local/supplied/current builtin producer references carry the same existing label fact as their eta closure. Known seeds remain exactly `secret` and `_secret_read`. Lexical bindings, local declarations and actual constructor evidence take precedence. Synthetic declared-return scheme handling is reused unchanged.

| Producer value use | Before, named / eta | Final, named / eta |
|---|---|---|
| Passed to an applying local helper | bottom / secret | secret / secret |
| Passed to an ignoring local helper | bottom / secret | secret / secret |
| Returned/stored without application | bottom / secret | secret / secret |

The last two rows deliberately retain the design's existing conservative closure policy. Our intermediate stored-reference-bottom assumption was not a ratified requirement. No new contextual scalar replay, symbolic return terms, label-polymorphic inference or probe summaries are introduced. Ordinary CheckModuleIFC caller policy, HM and M3/REPL import wiring remain unchanged. Finite intrinsic convergence/work guard, precise typed handoff, declared-return and scoped/bare Declassify result rules remain covered. Pure unlabelled producers still carry bottom. Paired controls cover nested declared return labels, local/supplied producers, both known builtin seeds and lookup shadowing.

Raw RED/green evidence: `1771-review-local-source-alias.log`, `1771-local-producer-red.log`, `1771-local-returned-producer-red.log`, `1771-known-callback-producer-review.log`, `1771-known-callback-eta-controls.json`, `1771-known-seed-parity-red.log`, `1771-producer-eta-parity-green.log`, `1771-known-seed-extern-green.log`, and `1771-parity-policy-focused.log` in the scratch evidence directory. Independent evaluation remains separate.

## Metadata corpus and bodyless declarations

The established IFC corpus harness was reused through read-only Go overlays against the SAME frozen source index `d2e14731f559d4c7cb90b7e16e33a99252687f3c15138a38f80f477b10b69e85`. Baseline 5d4fa361e versus final parity: 2,628 files; 23 identical parser failures; 29,165 baseline function summaries; all 2,568 successfully analysed all-bottom modules byte-identical; zero changes to previously successful metadata; zero private rho leaks. No new analysis errors. The companion `m-ifc-cross-module-labels-repair-metadata-corpus.json` preserves the exact rows and outcomes.

Two duplicate 27-line extern-only world stubs hit the original intrinsic-convergence work guard because absent bodies were never memoized. Root approved the narrow original M1 extern/synthetic fix: bodyless declarations now converge using only trusted declared facts, without a fabricated body. Existing declared parameter/return/refinement/authority metadata is preserved. The two baseline errors disappear (2→0); all other corpus metadata is unchanged. `TestIFCExportBodylessDeclarations` covers multiple externs alongside a normal labelled function; its old-source RED is `1771-extern-multi-red.log`.

Final integrated compiler/interface/consumer gates are recorded above after the actual effect-row/explicit-empty dev merge. The frozen metadata comparison is a focused M1 analysis comparison, not a claim that every program type-checks or that deferred imported-call enforcement has shipped.
