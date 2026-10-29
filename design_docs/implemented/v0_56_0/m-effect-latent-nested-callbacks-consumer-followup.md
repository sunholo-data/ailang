# Post-merge #1963 consumer classification — resolved by accepted explicit-empty contracts

**Current status:** The original classification below is preserved as pre-amendment history. Consumer compatibility is resolved by accepted PR #2005; the separate alias-ownership invariant follow-up remains active.

Independent read-only classification after language agent's pinned four-arm reduction. No compiler or consumer edits. Preserve the original source PASS/gate report and the broader consumer failure evidence; neither claims the original25-file production subset covered the full60-file core selection.

## Verdict

The reduced behavior matches the approved **parameter-wide conservative hand-off model**. It does not establish a violation of AC4 or a deviation of implementation from the written four-site algorithm. This is a newly measured consumer compatibility limitation of that model. A finer per-field exemption or explicit closed-empty arrow support would be a precision/language design change requiring approval, not a justified one-line semantic filter.

## Pinned four-arm result

| Case | Base c04a44780 | #1958 only752cebf3 | #1963 only87f84698 | Combined970a4669 |
| --- | --- | --- | --- | --- |
| mixed | PASS | PASS | REJECT | REJECT |
| concrete | PASS | PASS | REJECT | REJECT |
| literal | PASS | PASS | PASS | PASS |
| implicit | PASS | PASS | REJECT | REJECT |
| single_concrete | PASS | PASS | PASS | PASS |
| all_io | PASS | PASS | PASS | PASS |
| actual_four_pure | PASS | PASS | REJECT | REJECT |

The misleadingly named `concrete` case writes !{} on its pure sibling; it does not produce an internally concrete arrow under current elaboration. A single nonempty concrete IO field and two nonempty concrete IO fields both stay accepted across all arms. Literal storage stays accepted. An all-implicit store-only record hand-off is rejected on nested/combined exactly as intended. Actual fixture_rt and the four-!{} scratch migration both reject only on nested/combined, naming11 effects. The five affected core entries remain a real release compatibility gate; #1958's own guard is not the cause.

## Why this follows the approved contract

Design214 specifies charging a nested hand-off when a **callee parameter** nests an open arrow, even when the callee only stores it. Design223 explicitly ratifies store-only nested hand-off as an intended breaking flip. Design333/334 documents keepAll([logIt]) and non-generic constructor rejection. Sites(a)/(b) explicitly prescribe one boolean per parameter plus union of all actual nested arrow rows. The implementation does exactly that: any open sibling activates the parameter mask, then all siblings contribute their rows.

AC4 (design325/346) protects direct tuple/list/record literal storage and generic length. Those controls still pass. It does not specify a per-field exemption for mixed record parameters once the parameter's hand-off mask activates. The documented all-concrete storage control still passes when every arrow is actually concrete.

## Why the suggested !{} migration fails

Independently inspected internal/elaborate/file_funcs.go398–424. A function-type annotation starts with a fresh open epsilon tail. The parser/elaborator branch is selected only when len(typ.Effects)>0; an empty !{} row follows the same implicit branch as an omitted annotation, sets ConcreteEffectContract=false and ImplicitEffectContract=true. This source file is byte-identical between basec04a44780 and reviewed nested86d3a5f8. The empty-row distinction was absent before #1963; it is newly exposed by the nested hand-off rule. Therefore adding !{} to the four pure FixtureOverrides fields cannot make the parameter all-concrete, and should not be presented as a verified migration.

## Boundaries and next decision

The record mixes four implicit/pure slots and concrete effectful siblings. The broad union is conservative and can overcharge storage-only factories, but it is the frozen architecture. Merely dropping actual argument arrows marked ConcreteEffectContract would also drop an explicitly effectful actual callback handed to an implicit callee slot, weakening soundness. Any narrower algorithm must track callee source-position ownership with discriminators for implicit effectful callbacks, mixed siblings, aliases/imports and erase-subset-charge.

The current evidence supports a separately approved consumer adaptation under the existing conservative rule, or a new approved precision/closed-empty-arrow design. It does not authorize either. Keep #1958 M4 and broader consumer enablement pending that concrete decision. Preserve original60→55, play_chunks migration and successful pre-nested60/60 evidence alongside the new nested-only55/60 result.

Evidence:1958-nested-mixed-matrix.json,1958-nested-four-pure-matrix.json,1958-fixture-rt-four-arm.json and sources in1958-nested-mixed-repro/. Reviewer inspected actual source variants, all result JSON, approved freeze/AC4 language and the production mask/publication/elaboration code; no duplicate four-arm tests were run.


## Accepted compatibility resolution — 2026-10-29

Mark approved and executed the [explicit-empty callback amendment](m-effect-explicit-empty-callback-contract.md), independently accepted at 98/100 and merged in PR #2005 at `80793dbcf42423027be4bab1a0088da54e2323d8`. This supplies closed concrete pure callback contracts without weakening the conservative parameter-wide charging rule. The exact four pure fields were delivered on the pinned fork at `083ef43726981099a0bad4fedda18fc72105d6fa` after actual **60/60 core and 9/9 extension** proof, including fresh no-cache checks with initial deadlines and passing serial retries retained. See [final compatibility evidence](m-effect-explicit-empty-callback-contract-compatibility.json). The earlier 55/60 outcomes and failed four-field hypothesis above remain historical evidence, not erased failures or current blockers.

This resolves the fixture compatibility follow-up only. The separately owned alias-hidden callback-tail ownership invariant repair remains active and is not completed by this amendment. Target v0.56.0 remains unreleased; no upstream or live rig deployment is implied.
