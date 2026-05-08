# Sprint Plan: M-MOTOKO-EXT-PER-TASK

**Status**: Planned
**Target**: v0.19.0
**Estimated**: 2.5 working days (~15 hours)
**Source-of-truth design**: [m-motoko-ext-per-task.md](m-motoko-ext-per-task.md)

> This plan drives execution against the design doc. All architectural decisions, axiom scoring, risks, and rationale live there. This file is the milestone-by-milestone schedule.

> **⚠️ GATING PREREQUISITE**: This sprint cannot start until M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) has passed local validation. The user has explicitly sequenced this AFTER local benchmark verification of the v0.18.0 single-config adapter. Don't start until the v0.18.0 adapter has produced at least one paired-comparison row in eval results.

---

## Why this sprint, why now

- **v0.18.0's single-config Dockerfile is one-config-fits-all.** Per-extension benchmark analysis (the M-BENCH-MOTOKO-EXTENSIONS sprint queued in v0.18.0's Future Work) cannot work without per-invocation extension specification.
- **The threshold-measurement experiment becomes more actionable**: bundle answers (motoko-as-a-whole vs vanilla) are interesting; per-extension answers (which extension contributes the lift) are decision tools.
- **Cost-arbitrage thesis quantification**: "what's the minimum extension set to clear the AILANG-correctness threshold?" is a real procurement question. This sprint makes it answerable.
- **User-confirmed direction**: "I think it needs to be option B - each time its invoked it could have different packages specified" (2026-05-08 feedback after v0.18.0 evaluator PASS).

---

## Velocity calibration

Reference points:

| Sprint | Impl LOC | Test LOC | Total | Sprint length |
|---|---:|---:|---:|---:|
| M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) | ~900 | ~520 | ~1,420 | 1.5 days actual |
| M-MOTOKO-EVAL-INSTRUMENTATION (motoko-side) | ~250 | ~80 | ~330 | 0.5 days |
| Pi adapter (M-EXEC-PI, v0.14.2) | ~500 | ~350 | ~850 | 1 week |

**Planning target for this sprint**: ~250 LOC impl + ~180 LOC tests + ~30 LOC YAML/Dockerfile = ~460 LOC total. Smaller than v0.18.0 because we're refactoring an existing adapter, not building from scratch. **Estimate: 2.5 days** (with buffer).

---

## Milestone breakdown

Five phases, 14 milestones. Each milestone is a clean acceptance gate.

| # | ID | Title | Est. LOC | Phase | Depends on |
|---|---|---|---:|---|---|
| M0 | M0_MOTOKO_ENV_VAR | motoko-side `MOTOKO_REGISTRY_OVERRIDE` env var | ~5 (motoko repo) | Phase 1 | — |
| M1 | M1_POLICY_LOADER | Allowlist policy file + loader | ~80 + 60 tests | Phase 1 | — |
| M2 | M2_EXTENSIONS_MODULE | extensions.go (parse + cache + prepareRegistry) | ~150 | Phase 2 | M1 |
| M3 | M3_ADAPTER_WIRING | motoko.go integration: prepareRegistry before spawn | ~30 | Phase 2 | M2 |
| M4 | M4_EXTENSIONS_TESTS | extensions_test.go (≥80% coverage) | ~120 | Phase 2 | M2 |
| M5 | M5_DOCKERFILE | Strip extension install from Dockerfile.agent-motoko | -25/+10 | Phase 3 | — |
| M6 | M6_PREWARM | Pre-warm strategy implementation | ~50 | Phase 3 | M5 |
| M7 | M7_MULTIVAC_DOCS | Update agent-motoko-multivac-prs.md | -10/+30 | Phase 3 | M5 |
| M8 | M8_MODELS_SPLIT | models.yml: 4 → 8 motoko entries (bare/full pairs) | ~250 | Phase 4 | — |
| M9 | M9_METADATA_THREAD | Verify/wire model metadata → Task.Metadata | ~20 | Phase 4 | M3, M8 |
| M10 | M10_INTEGRATION_TEST | TestEvalSuite_MotokoBareVsFull | ~80 | Phase 4 | M3, M9 |
| M11 | M11_README | Update internal/executor/motoko/README.md | +50 | Phase 4 | M3 |
| M12 | M12_LOCAL_RUN | Live paired-comparison run with OPENROUTER_API_KEY | — | Phase 5 | All above |
| M13 | M13_CHANGELOG | CHANGELOG entry with M12 numbers | ~40 | Phase 5 | M12 |
| M14 | M14_FINALIZE | Move design doc + sprint plan to implemented/ | — | Phase 5 | M13 |

**Total**: ~460 LOC + 14 milestones across 2.5 working days.

---

## Day-by-day

### Day 1 — Foundation + adapter (~6 hours)

**Morning (3h):**
- M0 (1h): motoko-side `MOTOKO_REGISTRY_OVERRIDE` env var — small PR on motoko_agent. Acceptance: `MOTOKO_REGISTRY_OVERRIDE=/tmp/x.ail motoko --version` works against an arbitrary path; backward-compat verified (env var unset = unchanged behavior).
- M1 (2h): `motoko_extensions_policy.yml` + `motoko_policy.go` loader. Tests: glob match (dev), explicit list (prod), env-not-found, malformed YAML.

**Afternoon (3h):**
- M2 (3h): `internal/executor/motoko/extensions.go` — `parseExtensionList`, `cacheDir` (sha256 hash including ailang version), `prepareRegistry` (orchestrates allowlist check + cache lookup + cold install + chmod-immutable).

**Acceptance gate (Day 1):**
- `go build ./internal/executor/motoko/` clean
- `go test ./internal/executor/motoko/ -run TestParseExtension` + `TestCacheDir` pass
- M0 verified manually against motoko binary

---

### Day 2 — Wiring + Docker + tests (~6 hours)

**Morning (3h):**
- M3 (1h): Wire `prepareRegistry` into `motoko.go::ExecuteStreaming`. Pre-spawn: call prepareRegistry; on success, set `MOTOKO_REGISTRY_OVERRIDE` env; on policy reject, return Result.Success=false (don't spawn).
- M4 (2h): Tests covering all 5 acceptance criteria from the design doc (BareNoExtensions, FullStack, AllowlistRejects, HotCacheHit, HashIncludesAilangVersion). Use mock binary like v0.18.0's TestExecute_MockBinary_FullPipeline.

**Acceptance gate (Day 2 morning):**
- All extension tests pass; coverage ≥80% on extensions.go
- `make test` whole-tree green

**Afternoon (3h):**
- M5 (1h): Strip extension install from Dockerfile.agent-motoko; verify `docker build` works
- M6 (1h): Pre-warm baked hashes for the 5 most-popular extension stacks
- M7 (30min): Update multivac-prs.md to reflect simpler kernel-only image
- M8 (30min): models.yml restructure — 4 → 8 motoko entries with metadata.motoko_extensions

**Acceptance gate (Day 2 afternoon):**
- `docker build -f docker/Dockerfile.agent-motoko -t agent-motoko:dev .` succeeds
- 8 motoko entries present in models.yml; agent_suite + harness_suite updated; `TestModelsYml_MotokoBareFullPairs` passes

---

### Day 3 — Integration + validation (~3 hours, half-day buffer)

**Morning (1.5h):**
- M9 (30min): Verify Task.Metadata threading works (or wire it if missing). Run `ailang eval-suite --models motoko-claude-haiku-4-5-bare --benchmarks <one-task> --dry-run` to confirm metadata flows through.
- M10 (1h): Integration test `TestEvalSuite_MotokoBareVsFull` — both Tasks construct correctly, produce different ProviderData["motoko_extensions"] values
- M11 (30min): Update internal/executor/motoko/README.md with new section

**Afternoon (1.5h):**
- M12 (1h): Live run against real OPENROUTER_API_KEY:
  ```bash
  ailang eval-suite \
    --models motoko-claude-haiku-4-5-bare,motoko-claude-haiku-4-5-full \
    --benchmarks <small-tier>
  ```
  Verify two rows with different cost/token/pass-rate profiles. Capture numbers.
- M13 (30min): CHANGELOG entry citing M12 numbers
- M14 (15min): Move design doc + sprint plan to `implemented/v0_19_0/`; add Implementation Report

**Acceptance gate (Day 3):**
- M12 produces at least one valid paired-comparison row
- CHANGELOG shows concrete delta (pass rate, cost, tokens)
- Design doc moved + status updated

---

## Dependency graph

```
M0 (motoko env var)
M1 (policy)
  └── M2 (extensions module)
        ├── M3 (adapter wiring)
        │     ├── M9 (metadata thread)
        │     │     └── M10 (integration test)
        │     │           └── M12 (live run)
        │     │                 └── M13 (CHANGELOG)
        │     │                       └── M14 (finalize)
        │     └── M11 (README)
        └── M4 (extension tests)
M5 (Dockerfile)
  ├── M6 (pre-warm)
  └── M7 (multivac docs)
M8 (models.yml split)  ────────┘
```

M5/M6/M7/M8 can run in parallel with M2/M3/M4 (different files, no overlap). Recommend single-threaded execution since this is a 2.5-day sprint where sub-agent overhead exceeds wins.

---

## External dependencies

| Dependency | Status | Owner | Notes |
|---|---|---|---|
| **M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) local validation PASS** | ⚠️ **GATING — pending** | mark | User has stated this sprint is gated on local validation of v0.18.0 surfacing no other gaps. Don't start until M12 of v0.18.0 produces at least one paired comparison. |
| motoko_agent fork on `motoko-dx-compaction-pending` (or merged to main by then) | ✅ ready | sunholo-data/motoko_agent | M0's env-var change lands as a small PR here |
| ailang-multivac dev access | ⚠️ verify | mark | Only needed for cloud smoke after M12; local M12 covers the validation |
| 9 motoko-ext-* packages registry-published | ✅ done | sunholo-data/ailang-packages | shipped 2026-05-07 |
| AILANG `ailang install` + `ailang lock` + `ailang generate-extension-registry` work in arbitrary tmpdirs | ✅ verified | this repo | All three commands accept `--config <path>` per existing implementation |

---

## Risks (sprint-execution-specific)

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `ailang install` is slow (~10-30s per cold cache); 14 milestones × multiple test runs balloons total clock time | Medium | Low (just slower; doesn't block correctness) | M4's tests use mocks for the install path; only M10 + M12 actually invoke install |
| motoko-side M0 PR review is slow (cross-repo dependency) | Low | Medium (M3 onwards waits) | M0 is 5 LOC; should land same-day; if it doesn't, fall back to writing the registry directly into motoko's source tree per task (uglier but unblocks) |
| M12 live run produces ambiguous numbers (bare ≈ full) | Low | Low (the measurement existing IS the deliverable; ambiguous numbers are still publishable findings) | Document as an interesting result, not a sprint failure; defer interpretation to follow-up |
| Cache file-locking is harder than expected on shared filesystems (Cloud Run Job) | Medium | Low | Use `flock(2)` directly; document as Linux-specific (Cloud Run Jobs are Linux); Mac/Windows local dev may need a polyfill |
| Pre-warm strategy adds complexity to the Dockerfile (M6) | Medium | Low | Defer pre-warm to a follow-up if time-constrained; first-task latency cost is acceptable for v0.1 |

---

## Done criteria (sprint-level)

- [ ] M0–M14 all complete
- [ ] `make test` + `make lint` whole-tree green
- [ ] `ailang eval-suite --models motoko-claude-haiku-4-5-bare,motoko-claude-haiku-4-5-full --benchmarks <small-tier>` produces real, distinct paired-comparison rows
- [ ] CHANGELOG cites concrete numbers (pass rate, cost, tokens, cache fields) for both bare and full
- [ ] Allowlist policy file rejects an unauthorized package install with clear error in prod env
- [ ] Hot-cache test verifies repeat invocation skips the install step
- [ ] Design doc + sprint plan moved to `implemented/v0_19_0/` with Implementation Report appended

---

## References

- **Design doc** (source of truth): [m-motoko-ext-per-task.md](m-motoko-ext-per-task.md)
- **Predecessor**: [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../v0_18_0/m-motoko-executor-adapter.md)
- **Canonical executor contract**: [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md)
- **Source proposal**: User feedback after v0.18.0 evaluator PASS (2026-05-08): "Option B - each time its invoked it could have different packages specified"
