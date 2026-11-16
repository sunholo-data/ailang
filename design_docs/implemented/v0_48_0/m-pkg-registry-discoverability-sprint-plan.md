# Sprint Plan: M-PKG-REGISTRY-DISCOVERABILITY

## Summary

Make the AILANG package registry the first stop for package consumers and authors by exposing `AGENT.md` on package pages, adding deterministic discoverability feedback to package quality reports, and requiring registry reuse analysis in sprint planning.

**Duration:** 2.5 working days / approximately 18 hours  
**Dependencies:** Approved design `m-pkg-registry-discoverability.md`; live public registry for manual website and overlap checks  
**Risk Level:** Medium

## Current Status Analysis

### Completed Recently

- The v0.40.0 quality-ladder Sprint 1 shipped the shared publisher/server quality-report seam that this sprint extends.
- Package publishing already uploads `AGENT.md` and the registry index already records `has_agent_doc`; no storage migration is required.
- The approved design verified all relevant website, publisher, validator, registry, and planner call sites and reserved PUB008/PUB018/PUB022/PUB023.

### Velocity

- The seven-day velocity script found only one recent commit and no usable LOC metrics.
- Capacity is therefore based on the design's bottom-up estimate: approximately 620 changed LOC over 18 hours, including tests and duplicated harness resources.
- The estimate retains the design's 2× uncertainty buffer; it is not presented as an observed repository LOC/day rate.

### Remaining from Design Doc

- Website `AGENT.md` snapshot and safe raw-text rendering: approximately 100 LOC.
- Package quality findings, measurements, seam tests, and documentation: approximately 340 LOC.
- Registry-aware sprint-planner process in both harness variants: approximately 180 LOC.

## Registry Reuse Audit

The mandatory registry search was run before decomposing the work:

| Capability | Search | Result | Classification | Reason |
|---|---|---|---|---|
| Website package-guide rendering | `ailang pkg search "registry documentation agent guide"` | No packages found | `none` | This modifies the repository's Docusaurus registry UI and sync pipeline; it is not an AILANG package capability. |
| Package quality validation | `ailang pkg search "package quality validation"` | No packages found | `none` | The work extends the canonical built-in publisher/validator report seam and shared PUB namespace. |
| Sprint planning reuse gate | `ailang pkg search "sprint planning reuse"` | No packages found | `none` | The deliverable changes repository-local agent skills, templates, and progress schema. |

No package can be depended on or contributed to for these milestones. The executor should proceed with repository-native changes and preserve this audit in the progress JSON.

## Proposed Milestones

### Milestone 1: Publish the Agent Guide on Package Pages

**Goal:** Snapshot each published package's `AGENT.md` during docs sync and display it safely as raw text in the existing package detail component.  
**Estimated:** 75 LOC implementation + 25 LOC tests/docs = 100 LOC  
**Duration:** 4 hours

**Example files to update:**

- `docs/scripts/sync-registry.sh`
- `docs/src/components/PackageExplorer/PackageDetail.jsx`
- `docs/src/components/PackageExplorer/styles.module.css`
- `docs/docs/packages/index.mdx`

**Tasks:**

- Add bounded per-package `AGENT.md` snapshot fetching with visible warning-and-continue behavior.
- Add the Agent Guide panel, collapse behavior, loading/error states, and escaped `<pre>` rendering without a markdown dependency or `dangerouslySetInnerHTML`.
- Add focused sync/component coverage using existing test conventions and update the package index description.
- Run the sync fixture/live check and the Docusaurus build; inspect one package with a guide and one without.

**Acceptance Criteria:**

- [ ] Packages with `has_agent_doc: true` expose the snapshotted guide from `/registry/<vendor>/<name>/AGENT.md`.
- [ ] Missing guides render the PUB020-oriented placeholder and do not fail sync or page rendering.
- [ ] Author content is rendered as escaped raw text; no new npm dependency is added.
- [ ] `cd docs && npm run build` passes after registry sync.

**Risks:**

- Live registry latency or missing objects can make docs sync flaky. Mitigation: retain a ten-second fetch bound, log package-specific failures, and cover missing objects with a local fixture.

### Milestone 2: Add Discoverability Quality Feedback

**Goal:** Add deterministic PUB008/PUB018/PUB022/PUB023 feedback with identical publisher and validator inputs and explicit overlap-check failure reporting.  
**Estimated:** 160 LOC implementation/docs + 180 LOC tests = 340 LOC  
**Duration:** 10 hours

**Example files to update:**

- `internal/pkg/quality.go`, `internal/pkg/quality_test.go`
- `internal/pkg/registry.go`, `internal/pkg/registry_test.go`
- `cmd/ailang/pkg_quality.go`, `cmd/ailang/pkg_quality_test.go`
- `cmd/registry-validator/main.go`, `cmd/registry-validator/quality_shadow_test.go`
- `docs/docs/guides/package-publishing.md`, `docs/docs/guides/packages.md`

**Tasks:**

- Extend quality inputs and docs measurements for AGENT.md coverage and registry overlap.
- Implement sorted, self-excluding exact-export overlap in the shared registry package.
- Wire AGENT.md content and overlap measurements into publisher and validator assembly, making unavailable overlap measurement a visible finding.
- Add table-driven finding, determinism, strict-mode, overlap, publisher-wiring, and server-shadow tests.
- Document all four PUB codes and author actions.

**Acceptance Criteria:**

- [ ] Gap fixtures emit PUB008, PUB018, PUB022, and PUB023; clean fixtures emit none.
- [ ] PUB018/PUB022 follow existing strict warning promotion while PUB008/PUB023 remain informational.
- [ ] Overlap results exclude self, are sorted and bounded in output, and a failed lookup is never silent.
- [ ] Publisher and validator produce byte-equivalent `server` quality sections for equivalent inputs.
- [ ] Existing quality-report readers remain compatible through additive/omitempty fields.

**Risks:**

- Registry lookup makes local quality measurement network-sensitive. Mitigation: separate measurement failure from report construction and emit the explicit not-run PUB023 badge.
- Exact export matches can be false positives. Mitigation: keep PUB023 informational and identify candidates for author judgment.

### Milestone 3: Enforce Registry Reuse in Sprint Planning

**Goal:** Make registry search and a populated reuse decision mandatory in both planner harnesses and their machine-readable handoff.  
**Estimated:** 150 LOC skill/templates/scripts + 30 LOC validation coverage = 180 LOC  
**Duration:** 4 hours

**Example files to update:**

- `.agents/skills/sprint-planner/SKILL.md` and `.claude/skills/sprint-planner/SKILL.md`
- `.agents/skills/sprint-planner/scripts/create_sprint_json.sh` and the `.claude/` variant
- `.agents/skills/sprint-planner/resources/sprint_plan_template.md` and the `.claude/` variant

**Tasks:**

- Add mandatory step 0 using `pkg search`, `pkg info`, and `pkg docs`, with `depend`/`contribute`/`none` decisions per implementable milestone.
- Add Registry Reuse Audit sections to both templates and analysis checklists.
- Add `registry_reuse` to generated sprint state, placeholder validation, and executor handoff examples.
- Dry-run both JSON generators and compare the semantic blocks across harness variants.

**Acceptance Criteria:**

- [ ] Both planner variants require the same registry audit semantics with only harness-specific path/name differences.
- [ ] Both scripts emit valid JSON with a populated `registry_reuse` array and reject placeholder/unpopulated state before handoff.
- [ ] Plan and handoff templates carry the audit decisions for executor/evaluator inspection.
- [ ] Script-level dry runs validate with `jq` and contain at least one real reuse decision per implementable milestone.

**Risks:**

- The two harness copies can drift. Mitigation: implement together and compare normalized blocks in a focused test or verification command.

## Day-by-Day Plan

### Day 1 (4 hours)

- Complete Milestone 1 test-first.
- Run docs sync/build and inspect guide-present/guide-absent behavior.

### Day 2 (10 hours)

- Complete Milestone 2 test-first, starting with shared pure quality and overlap tests.
- Wire publisher and validator only after shared tests pass.
- Run focused Go tests, then `make test`, `make fmt`, `make lint`, and `make check-boundaries`.

### Day 3 (4 hours)

- Complete Milestone 3 in both harness trees.
- Dry-run both sprint JSON generators, compare variants, and rerun repository checks affected by documentation/scripts.

## Success Metrics

- Package pages show safe raw `AGENT.md` text whenever a guide is published and degrade visibly when it is absent.
- Four deterministic, actionable PUB findings are covered on both publisher and server paths.
- Both planner harnesses require and persist registry reuse decisions.
- Tests: focused unit/integration tests plus full `make test` pass.
- Quality: `make fmt`, `make lint`, and `make check-boundaries` pass.
- Documentation: package index, publishing guide, and packages guide are updated.
- Dependency control: `docs/package.json` is unchanged.

## Dependencies

- Milestone 2 relies on the existing v1 quality report and registry index APIs; no planned quality-ladder Sprint 2/3 work is required.
- Milestone 3 is independent of Milestones 1 and 2 and may be implemented after either, but all three must pass before sprint completion.
- Manual live checks require public registry availability; deterministic fixture tests remain the acceptance source if the service is unavailable.

## Open Questions

- None blocking. The live AGENT.md API endpoint, formatted-markdown rendering, broader overlap semantics, and sprint-evaluator enforcement remain explicitly deferred by the approved design.

## Notes

- This sprint does not rename the package convention from `AGENT.md` to `AGENTS.md`.
- All new quality findings are feedback badges; this sprint does not alter stability gates, effect rankings, or release-hardening thresholds.
- The executor should preserve publisher/server seam equivalence and avoid adding a second report-construction path.
