# Examples Manifest Statistics Sync — Sprint Plan

**Status:** Planned; awaiting coordinator plan approval/merge before execution.
**Sprint ID:** M-EXAMPLES-MANIFEST-STAT-SYNC
**Design:** [Approved design](examples-manifest-stat-sync.md)
**Source task:** task-18a95848; planning task task-85a3a46e; digest #1835.
**Target:** v0.54.12. **Duration:** 1 working day (~6 hours including buffer).
**Estimate:** 320 added/changed LOC (implementation/config/docs ~210; tests ~110).
**Risk:** Low to medium: deterministic serialization and safe grandfathering of existing omissions.

## Scope and approval

The handoff explicitly approves the prior design. Treat D3/D4/D5 as accepted for planning: retain statistics in schema v1, seed the existing omission allowlist, and gate runnable/ only. The unchecked design-review boxes are historical; this plan records the handoff authorization without changing the design. Implement no compiler semantics or .ail source changes. The coordinator plan PR merge is the execution approval; return artifacts to that pipeline without independently triggering a second executor.

## Current state and velocity

Read CLAUDE.md and inspected the writer, validator, manifest types, make recipes, and CI wiring. The working tree was clean; current branch is coordinator/task-85a3a46e, std/VERSION v0.54.11. The systemic audit in the design covers writer, readers, missing entries, and the ignored aspirational field. Implementation remains outstanding: statistics are RawMessage in the writer, validator has only schema/modules checks, and make lacks a sync target. The self-test currently calls the writer against the real manifest: remove that side effect.

The 7-day velocity helper found only the shallow snapshot commit 04496274 and no usable LOC metrics. Historical throughput cannot be inferred from this checkout. Estimate bottom-up: four 60–75 minute milestones plus 60–90 minutes for integration and investigation (~6h total). Each fits a 60–90 minute executor run; commit after each milestone. Existing files are 125–417 lines, below the 800-line limit; no companion split is currently needed.

## Registry reuse

Ran `ailang pkg search manifest`, `ailang pkg info sunholo/registry_validator`, and `ailang pkg docs sunholo/registry_validator`. The candidate validates package publication (ailang.toml, exports, hashes), not examples JSON. All milestones: **none** for registry dependency; extend existing Go tooling and reuse internal/manifest and scripts/internal/importextract. No parallel statistics-only writer.

## Milestones

### M1: Complete statistics validation (~45 LOC)

**Dependencies:** None

**Files:** internal/manifest/manifest.go; internal/manifest/manifest_test.go. **Time:** 60–75 minutes.

- [ ] Add Aspirational to internal/manifest.Statistics and count StatusAspirational without changing schema v1.
- [ ] Unit tests prove all status counters, empty-manifest coverage, and rejection of a mismatched aspirational count; go test ./internal/manifest passes.

**Examples:** Existing examples/manifest.json and examples/runnable/ are the integration corpus; use temporary copies for negative controls. No new committed .ail feature example.

### M2: Sync all derived manifest fields (~85 LOC)

**Dependencies:** M1

**Files:** scripts/backfill_manifest_modules.go; make/examples.mk. **Time:** 60–75 minutes.

- [ ] Extend the existing mirror-doc writer to recompute all six statistics fields, including zero-total coverage; preserve authored fields, timestamps, generator, and key order.
- [ ] Add make sync-examples-manifest and retain backfill-manifest-modules as an alias; --check detects modules or statistics drift without writing.
- [ ] On a clean valid manifest sync produces byte-identical output; repeated sync is idempotent. A stale temporary manifest is repaired, then passes --check.

**Examples:** Existing examples/manifest.json and examples/runnable/ are the integration corpus; use temporary copies for negative controls. No new committed .ail feature example.

### M3: Gate unlisted runnable examples (~110 LOC)

**Dependencies:** M2

**Files:** scripts/validate_manifest.go (embedded allowlist); make/examples.mk. **Time:** 60–75 minutes.

- [ ] Append make sync-examples-manifest remediation to load errors and use the same command for modules drift.
- [ ] Walk only examples/runnable recursively, normalize resolved paths through importextract.ResolvePath, skip underscore-prefixed scratch files, and fail --ci for files absent from both manifest and explicit allowlist.
- [ ] Seed an embedded reason-bearing allowlist with the measured existing omissions (design baseline 47); remeasure before editing and explain any discrepancy rather than silently admitting new files.
- [ ] Temporary-fixture checks prove an unlisted file fails with its path, an allowlisted file passes, and the underscore scratch convention preserves the existing gate self-test.
- [ ] Update verify-examples comments and diagnostics for statistics/modules drift and missing entries; retain report generation and exit propagation.

**Examples:** Existing examples/manifest.json and examples/runnable/ are the integration corpus; use temporary copies for negative controls. No new committed .ail feature example.

### M4: Prove gates and document contribution workflow (~80 LOC)

**Dependencies:** M3

**Files:** make/examples.mk; docs/docs/guides/development-workflow.md; .agents/skills/sprint-executor/SKILL.md; changelogs/unreleased/2026-10-15-examples-manifest-stat-sync.md. **Time:** 60–75 minutes.

- [ ] Extend validate-manifest-selftest using isolated temporary manifests and example directories with trap cleanup; remove its current real-manifest rewrite.
- [ ] Keep the existing modules-drift negative control; add stale-statistics and unlisted-runnable controls that assert the expected diagnostic as well as failure, plus a valid positive control.
- [ ] Document add file + authored entry + make sync-examples-manifest + commit in docs/docs/guides/development-workflow.md and the sprint-executor example checklist.
- [ ] Add changelogs/unreleased/2026-10-15-examples-manifest-stat-sync.md opening with ###; record the allowlist disposition backlog in the existing #1835 digest for follow-up without closing that umbrella issue.
- [ ] make test-changed, make validate-manifest-selftest, make verify-examples-gate-selftest, make verify-examples, make check-file-sizes, and make check-changelog pass; restore generated report artifacts unrelated to the change without discarding other work.

**Examples:** Existing examples/manifest.json and examples/runnable/ are the integration corpus; use temporary copies for negative controls. No new committed .ail feature example.

## Day 1 execution and validation

Morning: M1 → M2; early afternoon: M3 → M4; remaining time: focused integration gates and review of serialized diff. Estimates include existing self-test adaptation and approximately 25% integration buffer. Do not expand into cleaning up the 47-file backlog or editing historical feature designs.

Use `go test ./internal/manifest` for M1 and the appropriate temporary-fixture checks for M2/M3. M4 runs `make test-changed` (touched packages plus test-core) and the targeted gates listed above. The full suite belongs to PR CI; this replaces the design's broad `make test` requirement for executor validation. `make ci-quick` does not currently run the validator, contrary to the design's pre-push assertion: rely on the actual `make verify-examples` CI call site, and do not claim pre-push coverage or broaden ci-quick in this sprint.

Success means deterministic regeneration, actionable statistics failure, rejection of newly unlisted runnable examples, all six statistics fields checked, and green targeted gates. Compare manifest bytes before/after on the clean baseline and verify a second sync is a no-op. Negative controls must assert their intended reason so a compilation error cannot masquerade as a working gate. Use `--manifest` and `--dir` fixtures, and add optional writer path flags if needed to isolate sync tests. Normalize native filesystem paths before membership checks and diagnostic assertions.

## Dependencies, risks, and handoff

No external runtime dependency or .ail edits; the AILANG syntax gate does not apply. Keep generated_at/generator untouched. Preserve all existing mirror fields; do not round-trip through manifest.Save. Resolve the design's seeded count against the actual checkout before implementation; count changes require explicit explanation. Missing-on-disk manifest entries retain existing warning behavior. Zero enumeration remains an instrument failure. Temporary negative controls must not modify the committed manifest or runnable tree.

Issue #1835 is the triage digest, not automatically a closeable implementation issue; PRs #1834/#1836 are context only. Carry #1835 as a reference and keep the backlog disposition visible there. Implementation approval and executor handoff are owned by the coordinator on plan-PR merge. Planning creates only this document and the sprint JSON; no implementation, messages, issue mutations, or merge are performed here.
