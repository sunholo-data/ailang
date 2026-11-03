# Sprint Plan: M-VALIDATOR-CHECK-BUDGET-PARITY

**Design:** [approved design](m-validator-check-budget-parity.md)  
**Sprint ID:** M-VALIDATOR-CHECK-BUDGET-PARITY  
**Target:** v0.56.1  
**Status:** Planned; implementation awaits sprint approval/executor routing.  
**Duration:** 3 engineering days, including contingency; production acceptance depends on deployment.  
**Risk:** Medium (GC wall-time tradeoff and environment reproduction).

## Goal and current status

Pace the validator check child below its RSS cap, return useful budget refusals, and persist successful check cost. The instance-derived cap already exists at HEAD c60fa199 (v0.56.0); do not reimplement it or assume it has deployed. The approved design audits check, verify, metadata and client error propagation; verify pacing and compiler memory reductions remain deferred.

The clean checkout contains one shallow-history commit dated 2026-11-02. The 7-day velocity script ran but returned no usable LOC/day or milestone-duration evidence; its changelog matches are historical. A measured velocity cannot be inferred. Use a conservative planning assumption of about 175 changed lines/day, 520 lines total (roughly 230 implementation, 190 tests, 100 docs), with four 60–90 minute implementation sessions plus focused validation and a 25% contingency across three days. There is no Go toolchain here, so planning did not execute Go tests or coverage; execution requires the repository-supported toolchain.

## Registry reuse audit

Ran `ailang pkg search validator`, then `ailang pkg info sunholo/registry_validator` and `ailang pkg docs sunholo/registry_validator`. Candidate v0.1.2 validates manifests/exports/hashes in AILANG; it does not control the production Go subprocess watchdog, GC environment, HTTP refusal, or GCS metadata path. All four milestones choose **none**: extend existing Go infrastructure/docs; add no package dependency. No package-like capability is being rebuilt.

## Rulings and human-decision gate

Approved rulings: retain instance-derived cap and 128 MiB server headroom; check GOMEMLIMIT is cap minus 64 MiB; retain RSS watchdog and 200 s wall cap; add two optional schema-v2 metadata fields; retain client verbatim 422 behavior. No client or verify changes.

Still OPEN: design **High-Impact Decisions → Deploy path**, **Design Freeze → Mark approves the deploy route**, and **Implementation Plan → Phase 0**. General design approval does not select release-tag versus emergency rebuild. Before implementation begins, check for an explicit recorded route ruling as required by Design Freeze. If absent: **STOP: commit current work, report `BLOCKED: Mark must choose release-tag or emergency validator rebuild route`, and end the run**. Do not self-approve a route. Once recorded, copy the ruling into execution evidence. No additional human-decision gates.

## Milestones

### M1: Check pacing and sampled statistics (~160 LOC)

**Duration:** 60–90 minutes implementation plus focused validation.
**Dependencies:** None

Update bounded.go and validate.go with explicit child environment and stats returns; add bounded_env_test.go. Preserve ordinary compile errors and the existing watchdog. Cover the package path; explicitly document the existing bare-file fallback as outside this package-publish change.

**Acceptance criteria:**

- [ ] 896 MiB RSS cap produces GOMEMLIMIT=832MiB; inherited duplicate GOMEMLIMIT is replaced, other environment values preserved.
- [ ] Tiny caps cannot underflow into an enormous limit; unusable pacing budgets return an explicit error.
- [ ] Child env-echo proves injection; deterministic RSS harness proves success/failure peak and elapsed stats.
- [ ] Existing over-memory, timeout, child-failure, cap-derivation and Linux procRSS tests pass; wall cap stays 200 s.

**Risk/mitigation:** GC pacing can trade memory for time; preserve timeout and measure wall cost.

### M2: Environment-complete refusals (~150 LOC)

**Duration:** 60–90 minutes implementation plus focused validation.
**Dependencies:** M1

Carry startup cap source and actual instance limit into refusal context through main.go; add bounded_context.go and bounded_context_test.go for rendering and reproduction. Test memory and time refusals.

**Acceptance criteria:**

- [ ] Refusal includes cap/source, known instance limit or explicit unavailable marker, GOOS/GOARCH, effective child GOMAXPROCS, GOMEMLIMIT, 100 ms cadence, sampled peak and wall.
- [ ] Existing resource/limit/observed semantics remain, including the 600 MiB observed assertion; budget refusal is HTTP 422 and ordinary compile failure is distinct.
- [ ] Docker command uses the actual image tag or explicit missing-identity diagnostic, --platform linux/amd64, --cpus 2, --memory 1g and explicit GOMEMLIMIT/AILANG_NO_CACHE.
- [ ] Command uses disposable writable package staging, cold cache and dependency resolution matching validator preparation; document that Docker does not reproduce gVisor. Do not label unknown runtime details as measured.

**Risk/mitigation:** Host parity cannot be guaranteed; label residual differences and test the runnable command.

### M3: Persist check cost in publish metadata (~110 LOC)

**Duration:** 60–90 minutes implementation plus focused validation.
**Dependencies:** M1, M2

Add CheckPeakRSS and CheckWallMS to internal/pkg/registry_types.go; plumb through handlePublish in main.go. Create registry_types_check_stats_test.go and main_check_stats_test.go, using the existing publish harness or a narrow injection seam.

**Acceptance criteria:**

- [ ] JSON names are check_peak_rss (bytes) and check_wall_ms (milliseconds), both omitempty; legacy metadata decodes and populated metadata round-trips.
- [ ] Successful publish path persists sampled child RSS and check-only wall time, without counting dependency downloads as child time.
- [ ] A deterministic handler test proves stats reach stored metadata; unavailable or unsampled RSS stays zero/omitted, never fabricated.
- [ ] Existing metadata schema and quality fields remain compatible; verification child is unchanged.

**Risk/mitigation:** Short checks may escape sampling; retain explicit zero/omitted semantics.

### M4: Troubleshooting and release evidence (~100 LOC)

**Duration:** 60–90 minutes implementation plus focused validation.
**Dependencies:** M2, M3

Update docs/docs/guides/package-publishing.md with a budgets anchor and tested command; add changelogs/unreleased/2026-11-02-validator-check-budget-parity.md opening with ###. Store validation notes under design_docs/planned/v0_56_1/evidence/m-validator-check-budget-parity/.

**Acceptance criteria:**

- [ ] Guide explains every refusal field, sampled-RSS limitations, host/runtime differences, cold dependencies, wide records and deep let chains; no invented module/signature ceilings.
- [ ] Existing package tarball or existing fixture serves as the command example; no new .ail files or syntax are required.
- [ ] Run focused final gates and record outcomes; Docker/staging evidence records image identity, cap source, peak and wall when infrastructure is available.
- [ ] Record paced versus unpaced check measurements for an existing GC-heavy package without raising 200 s cap; report unavailable performance evidence explicitly.
- [ ] Production rollout remains subject to the cited open deploy-route decision; do not claim celestial publish or worlds unblock without registry evidence.

**Risk/mitigation:** Cloud credentials or image access may be unavailable; separate local completion from unverified production acceptance.

## Day-by-day schedule

- Day 1: Resolve the cited design-freeze gate from recorded human direction; M1, then M2. Run targeted validator tests after each milestone.
- Day 2: M3 and M4; validate metadata and reproduction instructions with existing fixtures. Keep sessions within 90 minutes; stop at milestone boundaries.
- Day 3: Focused final gates, Docker/staging measurements when available, and contingency. Record release handoff and remaining external acceptance.

## Validation and success metrics

Run `go test ./cmd/registry-validator ./internal/pkg -count=1` for touched packages during milestones. Final executor gate is `make test-changed TEST_CHANGED_BASE=c60fa199` (includes test-core), `make fmt`, `make lint`, `make check-boundaries`, `make check-file-sizes`, and `make check-changelog`. CI owns the full test suite. No stdlib or example source changes are planned, so stdlib/example gates and the AILANG syntax gate do not apply. Use new Go companion test files above; main.go is 611 lines and must remain below 800 (move new context code into bounded_context.go). Track behavior coverage of env override, tiny cap, memory/time refusal, normal failure, success stats, legacy decode and persisted metadata rather than inventing an unmeasured coverage percentage.

Before rollout, exercise the command against the actual built image and existing package tarball. The plain Docker command in the design is illustrative: it omits explicit arch/env/cache controls and validator dependency preparation. Correct those within the approved reproduction scope; do not promise gVisor equivalence. GOMEMLIMIT is a soft Go runtime target, not proof that every remaining RSS refusal is a genuinely oversized package. Sampled MiB formatting can truncate fractional excess; retain raw byte values in stats and avoid asserting peak must print limit+1 MiB.

## Deployment and external acceptance

After the route ruling, follow cloudbuild-registry-validator.yaml release/emergency procedure; no branch-head deploy. Record `/version`, image identity, startup derived cap/source (expected 896 MiB on a confirmed 1 GiB instance), successful publish metadata and a staging 422. A publisher-owned celestial@0.6.0 retry and registry index evidence, followed by worlds dependency acceptance, complete the operational success criteria. Package publishing is outside the planning task; do not trigger it here. If deployment or publisher access is unavailable, report those criteria pending rather than marking production success.

## Handoff

The coordinator receives this plan and its populated JSON via the final artifact markers. Plan approval/merge routes to sprint-executor; this planning task does not start implementation or send a duplicate autonomous task. Historical issue #1811 is context, not an issue to auto-close. No current issue number was supplied.
