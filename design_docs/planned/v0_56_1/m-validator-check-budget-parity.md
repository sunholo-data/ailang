# M-VALIDATOR-CHECK-BUDGET-PARITY: check budgets follow the box, and refusals a publisher can reproduce

**Status**: Implementation complete; verification blocked (unchanged core cleanup tests and unavailable lint); operational acceptance pending
**Target**: v0.56.1
**Priority**: P1 (fleet-wide publish gate trust; blocks R3-WORLDS via sunholo/celestial@0.6.0)
**Estimated**: 3–4 days
**Dependencies**: v0.56.0 memory-cap fix (already at HEAD, `cmd/registry-validator/bounded.go` — this doc designs the deploy verification and the residual work). Complements, does not depend on, [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/docs/references/axioms.mdx)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | GC pacing under `GOMEMLIMIT` is a host-boundary control in the class of the already-accepted `--timeout`/RSS cap; semantic evaluation of the checked package is unchanged |
| A2: Replayability | 0 | No trace-format change; refusals and metadata are records, not traces |
| A3: Effect Legibility | 0 | No language effect surface touched |
| A4: Explicit Authority | +1 | The memory budget stays explicit, caller-set authority over a real resource, and its *source* (cgroup/meminfo/fallback) is now stated in every refusal |
| A5: Bounded Verification | 0 | Z3/verify path untouched (see Deferred Decisions for the verify child) |
| A6: Safe Concurrency | 0 | Watchdog loop unchanged; one new env var on a child process |
| A7: Machines First | +1 | A refused agent gets machine-parseable context (cap, source, arch, GOMAXPROCS, peak, wall) and a reproduction command, instead of an instruction to compare apples (darwin/arm64 local) with oranges (linux/amd64 gVisor) |
| A8: Minimal Syntax | +1 | Zero new syntax; one env var, metadata fields, and message text |
| A9: Cost Visibility | +1 | Check memory and wall cost become banked, queryable per published version — the fleet distribution that would have caught this class before three publisher reports |
| A10: Composability | 0 | Orthogonal to quality-gate sections; server-sourced values only |
| A11: Structured Failure | +1 | Budget refusals stay typed 422s and gain the environment block; plain compile failures stay plain errors (asserted by `TestRunBounded_ChildFailureIsNotBudget`) |
| A12: System Boundary | +1 | The cap is *documented* as the instance boundary (derived, logged with source at startup, now also in the refusal) rather than an ambient constant |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (host boundary only, explicitly labeled)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): the change *is* machine-decidability of a refusal

## Problem Statement

On 2026-11-02 the publish of `sunholo/celestial@0.6.0` (22 files, 155 signatures, PR #174, needed by `sunholo/worlds@0.1.0` for the R3-WORLDS sprint) was refused:

> `ailang check --package .` exceeded the registry validator's memory budget (384 MiB; observed 384 MiB) and was stopped.

The publisher followed the refusal's own advice — `AILANG_NO_CACHE=1 /usr/bin/time -l ailang check --package .` — and measured **4.15 s real, max RSS 241,500,160 B, peak footprint 195,756,920 B** on darwin arm64 (the Studio, AILANG v0.54.18; publishing binary v0.55.0-dirty). The validator "observed" **≥ 384 MiB** — more than 1.6× the local peak — on the same quality-passing package.

### Root cause (fixed at HEAD in v0.56.0 — the fix needs a doc, a deploy, and residuals)

The deployed v0.55.0 validator bounded the check child at a **hardcoded 384 MiB** sized for its original 512 MiB Cloud Run instance. The instance was resized to 2 vCPU / 1 GiB (ailang-multivac terraform `2cf025e`, after `sunholo/ailang_parse@0.55.0` hit the 200 s wall cap 7/7 on 1 vCPU) and the cap did not move. v0.56.0 (at HEAD, released in this repo) derives the cap at startup: **instance limit − 128 MiB**, read from cgroup v2, cgroup v1, or `/proc/meminfo` (Cloud Run gen1/gVisor), logged with its source; 384 MiB remains only when no limit is readable (`cmd/registry-validator/bounded.go`, `instanceRSSCap`; `main.go` sets `checkRSSCap` at startup and logs `check memory cap 896 MiB (…)` on the 1 GiB box; `TestInstanceRSSCap_FollowsLimit` covers 512 MiB/1 GiB/v1/gVisor/fallback/tiny). On a 1 GiB instance the cap becomes 896 MiB against a measured local peak of ~241 MB: **celestial@0.6.0 needs no trimming once v0.56.0+ is deployed.**

This is the **third incident of the same class**, each discovered by a publisher instead of by telemetry:

| Incident | Budget | Instance reality | Discovered by |
|---|---|---|---|
| #1811, 2026-10 (crew_core/crew_lab) | none | 512 MiB → whole-instance OOM-kills, bare 503 | publisher CI |
| ailang_parse@0.54.0/0.55.0, 2026-11 | 200 s wall | 1 vCPU too slow | publisher reports |
| celestial@0.6.0, 2026-11-02 (this doc) | 384 MiB RSS | 1 GiB instance | publisher report |

### What the v0.56.0 fix does NOT cover (the residual this doc designs)

1. **The fix is undeployed.** The validator moves on release tags only (`cloudbuild-registry-validator.yaml` header; the 2026-09-17 PUB005 fleet outage made this rule) and `2cf025e` only resized the box. Until a `^v.*` tag ≥ v0.56.0 deploys, every publish of a >384-MiB-peak package is still refused.
2. **The refusal's measurement advice yields incomparable numbers.** It tells the publisher to measure locally — but the validator measures the child's `VmRSS` (100 ms polling, `procRSS`) on linux/amd64, 2 vCPU, gVisor, in a fresh temp dir with dependencies re-resolved by `pkg.ResolveDependencies`, while the prescribed local command measures darwin/arm64 on a many-core Studio with an already-warm dependency tree. `AILANG_NO_CACHE=1` disables only the compile cache (`internal/config/compiler.go`: "1 disables the compile cache for this run") — it does not make the *environments* comparable. Expected contributors to the 1.6× gap: fewer GC CPUs on 2 vCPU (heap overshoot past the GOGC target before collection), amd64 vs arm64 layout, cold-container dependency compile, gVisor accounting. None is separable today because the refusal carries no environment context.
3. **"observed 384 MiB" == the limit exactly** — but the current `runBounded` reports the *sampled peak* and only kills when a sample strictly exceeds the cap, so a true peak must print ≥ 385 MiB. The exact equality is itself evidence the deployed v0.55.0's "observed" field did not carry a true sampled peak (its code is not inspectable from this checkout). Whatever it measured, the number it printed was not one a publisher could reproduce or reason from.
4. **No telemetry.** Nothing banks check peak RSS or wall time on *successful* publishes (`ValidationResult` in `internal/pkg/registry_types.go` has no such fields), so the fleet distribution that would right-size the instance and the cap is invisible; every budget incident reaches core as a hand-written report.
5. **The child gets no memory pacing.** `runBounded` sets no `cmd.Env` at all (verified: no `cmd.Env`/`GOMEMLIMIT`/`GOGC` anywhere in `cmd/registry-validator` non-test sources). Go's GC runs at its default GOGC target regardless of the cap; the process is killed at the wall instead of being paced under it. `GOMEMLIMIT` (Go ≥ 1.19; this repo builds on Go 1.26.10) exists precisely for this.
6. **"Tell us what to trim" has no answer anywhere.** `docs/docs/guides/package-publishing.md` documents no budget troubleshooting and no known peak drivers (the #1811 lesson — wide records and deep let chains — is changelog-only).

**Impact:** every publisher whose package's validator-side peak exceeds the cap (currently still 384 MiB until deploy) is blocked with a message whose numbers cannot be reproduced, and AILANG core learns about each occurrence from user reports. R3-WORLDS is blocked today.

## Goals

**Primary Goal:** Registry validator budget refusals carry reproducible, environment-complete context, the check child is paced under the budget instead of killed at it, and check cost is banked per publish so this incident class is detected by telemetry, not by publishers.

**Success Metrics:**
1. `sunholo/celestial@0.6.0` publishes on the deployed v0.56.1 validator (cap 896 MiB vs measured local peak ~241 MB — headroom ≥ 3.5×), unblocking R3-WORLDS.
2. A budget refusal names: the cap, its source path, the instance limit, arch, `GOMAXPROCS`, `GOMEMLIMIT`, the sampling cadence, the sampled peak, and the wall time — enough for a publisher (or an agent) to decide "my package is genuinely over budget" vs "environment mismatch" without asking core.
3. The refusal's reproduction instruction is *parity-correct*: a one-command docker reproduction of the validator environment (same image, arch, CPU, memory limit, cold cache), replacing apples-to-oranges `/usr/bin/time -l` advice.
4. Every successful publish banks `validation.check_peak_rss` and `validation.check_wall_ms`; a one-line registry query yields the fleet distribution.
5. A package whose peak would exceed the cap *because of GC overshoot* completes under `GOMEMLIMIT`; hard refusals are reserved for genuinely oversize packages (and remain 422, verbatim through `ailang publish`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Set `GOMEMLIMIT = checkRSSCap − 64 MiB` on the check child (kill watchdog retained as backstop) | Converts kill-at-wall refusals into GC pacing; wrong margin either thrashes GC (wall-cap risk, the ailang_parse failure mode) or leaves the cap unreachable | agent | design | med |
| Bank `validation.check_peak_rss` + `validation.check_wall_ms` in `ValidationResult` (schema v2, `omitempty`, v1 decodes) | New public metadata surface; follows the proven `contracts_error` pattern | agent | design | med |
| Refusal message gains an environment block + docker reproduction command (client prints 422 bodies verbatim, so no publisher upgrade is forced) | Message text is the only channel most publishers will ever read | agent | design | low |
| Deploy path: release tag (rule of 2026-09-17) vs emergency manual rebuild for the celestial unblock | Speed of unblocking R3-WORLDS vs the release-gate guardrail | **human** (Mark) | design | low |
| Wall cap stays 200 s; interplay with GOMEMLIMIT measured, not assumed | GC pacing costs wall time; 200 s is the next budget in line | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Cap derivation from instance limit (decided and implemented in v0.56.0 — this doc ratifies it)
- [x] **RULED (Mark, attended 2026-11-02): release tag, not an emergency rebuild.** Done: v0.56.0 and v0.56.1 deployed the derived cap (prod logs `check memory cap 896 MiB (/sys/fs/cgroup/memory/memory.limit_in_bytes limit 1024 MiB)`), and `sunholo/celestial@0.6.0` published 2026-11-02 19:03 UTC (HTTP 200, 17.9 s). Phase 0 is complete; executors start at Phase 1. Original item: Mark approves the deploy route for the celestial unblock (tag v0.56.1 release vs `gcloud builds submit` manual rebuild from the v0.56.0 tag, per the emergency-rebuild recipe in `cloudbuild-registry-validator.yaml`)
- [x] `GOMEMLIMIT` margin = 64 MiB below the cap (agent-decided; revisitable at runtime scale — see Deferred Decisions)

## Solution Design

### Overview

Three code changes inside `cmd/registry-validator` plus one metadata/doc change, layered on the already-implemented derived cap:

1. **Pace, then police.** Give the check child a `GOMEMLIMIT` derived from the same cap the watchdog enforces, so Go's GC has authority to keep the process under the budget. The kill watchdog stays as the physical backstop; a refusal after pacing means the package is genuinely over budget (non-heap growth or a real blowup), which is exactly what a 422 should mean.
2. **Refusals that carry their own world.** The `budgetError` gains the environment block (cap + source path + instance limit, arch, GOMAXPROCS, GOMEMLIMIT, sampling cadence) and replaces the local-measurement advice with a parity-correct docker reproduction of the validator environment.
3. **Bank the cost.** `runBounded` already tracks `peak`; return it (and elapsed wall) up through `runAilangCheck` into `ValidationResult` so every successful publish records its check cost. The fleet distribution is what sizes the next instance and the next cap — by measurement, not by incident.

### Architecture

**Components:**

1. **`bounded.go` — env + reporting.**
   - `runBounded` gains an `env []string` parameter (or a small `checkEnv(rssCap)` helper tested directly); `runAilangCheck` passes `GOMEMLIMIT=<cap−64 MiB>` appended to `os.Environ()`. Today `cmd.Env` is unset on every `exec.Command` in the validator — the child inherits the container env; this is the first explicit addition.
   - `budgetError` gains fields for the environment block; `Error()` renders cap, source, instance limit, arch (`runtime.GOARCH`), `GOMAXPROCS`, `GOMEMLIMIT`, poll cadence, sampled peak, elapsed wall — then the reproduction pointer (docker command) instead of the current "measure locally … and report the numbers" advice, which this incident proved incomparable. Exact final wording is agent latitude.
   - `runBounded` returns `(output, stats, err)` where `stats = {peakRSS, wall}` — peak is already computed; it is currently discarded on success.
2. **`validate.go` / `main.go` — plumbing.** `runAilangCheck` returns the stats; `handlePublish` copies them into the metadata `ValidationResult` it already builds (`main.go:268` area, where `runAilangCheck(tempDir)` is called).
3. **`internal/pkg/registry_types.go` — schema.**
   ```go
   // ValidationResult — additions (schema v2, omitempty; v1 records decode):
   CheckPeakRSS uint64 `json:"check_peak_rss,omitempty"` // bytes, sampled VmRSS peak
   CheckWallMS  int64  `json:"check_wall_ms,omitempty"`
   ```
   Same shape rules as the v2 signature fields: `omitempty`, old records decode, no reader changes required.
4. **`docs/docs/guides/package-publishing.md` — the "what to trim" page.** Budget troubleshooting: what the refusal fields mean, the docker reproduction, and the known peak drivers (wide records + deep let chains — the #1811 shape; module count/signature count with measured ceilings; dependency-graph cold compile).

### Implementation Plan

**Phase 0: unblock (ops, no code, ~hours) — DONE 2026-11-02 via the v0.56.0/v0.56.1 release tags (see Design Freeze ruling); skip it.**
- [ ] Mark picks the deploy route (Design Freeze item); either tag the v0.56.1 release or emergency-rebuild the validator image from the v0.56.0 tag (`gcloud builds submit --config cloudbuild-registry-validator.yaml --substitutions=_REGION=europe-west1,_TARGET_PROJECT=ailang-registry,TAG_NAME=manual-<sha>`).
- [ ] Verify deployment: `curl https://registry.ailang.sunholo.com/version` reports ≥ v0.56.0 (the skew guard exists for exactly this).
- [ ] Republish `sunholo/celestial@0.6.0` (publisher side needs no upgrade; the check runs on the validator's bundled binary, `docker/Dockerfile.registry-validator`).
- [ ] R3-WORLDS proceeds (`sunholo/worlds@0.1.0` dependency satisfied).

**Phase 1: refusal parity (~1 day)**
- [x] `budgetError` environment block + wording; docker reproduction command referencing the published image (`${_REGION}-docker.pkg.dev/ailang-registry/ailang-registry/validator:<tag>` with `--memory 1g --cpus 2`).
- [x] Tests: message contains cap, source, arch, GOMAXPROCS, GOMEMLIMIT, peak; `TestRunBounded_OverMemoryIsKilledAndNamed` updated for the new fields (it currently asserts `observed == "600 MiB"` — keep that assertion true).
- [x] `pkg_publish.go` untouched (already prints 422 bodies verbatim, `StatusUnprocessableEntity` case).

**Phase 2: GOMEMLIMIT pacing (~1 day)**
- [x] `checkEnv(rssCap)` helper + injection in `runAilangCheck`; margin 64 MiB.
- [x] Tests: helper math (1 GiB → `GOMEMLIMIT=832MiB`); a `runBounded` test asserting the child *saw* the env (portable child: `sh -c 'echo $GOMEMLIMIT'`-shaped, or assert on the helper + an env-echo integration case).
- [x] Watchdog unchanged: kill still fires on sampled `VmRSS > checkRSSCap` (GOMEMLIMIT excludes some non-Go-RSS, so the backstop stays authoritative).

**Phase 3: telemetry + docs (~1 day)**
- [x] `runBounded` returns stats; plumb to `ValidationResult.CheckPeakRSS/CheckWallMS`; v1-decode test in `internal/pkg` following the `registry_types` test pattern.
- [x] `package-publishing.md` budget troubleshooting section (fields, reproduction, trim guidance).
- [x] Read-side: nothing required (fields are informational; a registry query/`ailang pkg` report can render them later — Non-Goal).

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `cmd/registry-validator/bounded.go` — env helper, stats return, budgetError fields/rendering (~80 LOC)
- `cmd/registry-validator/bounded_test.go` — new assertions, keep existing ones true (~60 LOC)
- `cmd/registry-validator/validate.go` — pass env, return stats (~20 LOC)
- `cmd/registry-validator/main.go` — bank stats into metadata (~15 LOC)
- `internal/pkg/registry_types.go` — two fields (~6 LOC)
- `internal/pkg/registry_types_test.go` (or the existing registry test file) — v1-decode case (~20 LOC)
- `docs/docs/guides/package-publishing.md` — troubleshooting section (~80 lines)

## Examples

### Example 1: the refusal a publisher sees

**Before** (the incident, verbatim):
```
`ailang check --package .` exceeded the registry validator's memory budget (384 MiB; observed 384 MiB) and was stopped.
The package itself may be valid: measure locally with a cold cache —
`AILANG_NO_CACHE=1 /usr/bin/time -l ailang check --package .` — and report the numbers to AILANG core.
```
→ numbers that cannot be reproduced (see Problem Statement §2–3); local cold check peaks 241 MB; the publisher can only file a report.

**After** (field-for-field, illustrative wording):
```
`ailang check --package .` exceeded the registry validator's memory budget
(cap 896 MiB ← /sys/fs/cgroup/memory.max 1024 MiB, minus 128 MiB headroom;
observed peak 903 MiB sampled at 100 ms; wall 41.2 s).
Environment: linux/amd64, GOMAXPROCS=2, GOMEMLIMIT=832MiB, gVisor, cold container.
Your local measurement does NOT reproduce this environment. To reproduce:
  docker run --rm --cpus 2 --memory 1g -v "$PWD":/pkg -w /pkg \
    europe-west1-docker.pkg.dev/ailang-registry/ailang-registry/validator:v0.56.1 \
    ailang check --package .
If it exceeds the cap there too, trim peak drivers (see
docs/guides/package-publishing.md#budgets); otherwise report the numbers above to AILANG core.
```

### Example 2: telemetry replacing incident reports

**Before:** peak RSS of every successful check is discarded (`runBounded` computes `peak`, uses it only for the kill decision); the cap's history — 512 MiB box, then 1 GiB box, then whatever comes next — is reconstructed from changelogs and user reports.

**After:** `metadata.json` for every published version carries
```json
"validation": { "compiles": true, "check_peak_rss": 251658240, "check_wall_ms": 52300, ... }
```
and one query over the registry answers "what is the fleet's 99th-percentile check peak?" — the number that sizes the next instance *before* a publisher is refused.

## Success Criteria

- [ ] `sunholo/celestial@0.6.0` published; R3-WORLDS unblocked (acceptance: `ailang pkg search`/registry index shows the version; `sunholo/worlds@0.1.0` deploys).
- [ ] A budget refusal on a deployed validator contains cap, cap source, instance limit, arch, GOMAXPROCS, GOMEMLIMIT, sampled peak, wall, and a parity-correct reproduction command (acceptance: updated `TestRunBounded_OverMemoryIsKilledAndNamed` + new assertions; manual 422 against a staging validator with a synthetic over-cap package).
- [ ] `TestProcRSS_ReadsSelfOnLinux` still passes (the reader Cloud Run uses is still exercised on Linux).
- [ ] `TestInstanceRSSCap_FollowsLimit` passes unchanged — the v0.56.0 cap derivation this doc ratifies is not regressed.
- [ ] `runBounded`'s child sees `GOMEMLIMIT = checkRSSCap − 64 MiB` (acceptance: env-echo test).
- [ ] A publish on the new validator banks non-zero `check_peak_rss`/`check_wall_ms`; a v1 metadata record decodes with both absent (acceptance: `internal/pkg` decode test).
- [ ] `package-publishing.md` documents the refusal fields, reproduction command, and trim guidance.
- [ ] All tests passing (`make test`), `make fmt`, `make lint` clean; `make check-boundaries` clean (metadata field lives in `internal/pkg`, validator consumes — direction already legal).
- [ ] Documentation updated (changelog entry for v0.56.1 under a continuation of the v0.56.0 entry, linking this doc).

## Testing Strategy

**Unit tests** (`cmd/registry-validator`):
- `budgetError` rendering: every environment field present; existing `resource`/`limit`/`observed` assertions preserved.
- `checkEnv` math; `runBounded` child-sees-env case.
- Stats return: peak and wall from a fixed-RSS child (`fixedRSS` harness already exists).
- No new error-code namespace: this feature adds **no `PUBnnn`/`MODxxx` code** — the refusal is the 422 body string (`main.go` `jsonError(w, http.StatusUnprocessableEntity, "%v", budgetErr)`); nothing to allocate.

**Integration tests:**
- `handlePublish` path banks `check_peak_rss`/`check_wall_ms` into the metadata written to GCS (existing `main_test.go`/`handlers_api_test.go` harness with a fake bucket).
- Metadata decode: v1 record without the fields decodes; v2 record with them round-trips.

**Manual testing:**
- Staging validator (or `docker run` of the built image, `--memory 1g --cpus 2`): publish an over-cap synthetic package → refusal contains the full block; publish a normal package → banked stats visible in `metadata.json`.
- GOMEMLIMIT interplay: a GC-overshoot-shaped package completes under pacing; wall time recorded; if it approaches 200 s, that is the *measured* input to the wall-cap decision (High-Impact Decisions table), not an assumption.

## Deferred Decisions

- GOMEMLIMIT margin (64 MiB) — agent may retune at runtime scale with banked peak data; also whether to set `GOGC` alongside.
- Whether `runAilangVerify` (Z3 child, 120 s `verifyWallCap`) gets the same env pacing + stats treatment — same pattern, separate surface; defer until check-side data lands.
- Exact refusal wording and whether the reproduction command pins `:latest` or an explicit tag — agent may choose.
- A read-side rendering of check cost in `ailang pkg` reports / the package explorer — deferred until data accumulates.
- Sampling cadence (100 ms) — revisit only if a banked peak-vs-`VmHWM` discrepancy shows missed peaks (see Verification Log row V10).

## Non-Goals

- **Raising the wall cap (200 s) or changing request timeouts** — measured separately once GOMEMLIMIT lands (ailang_parse precedent: wall failures were CPU starvation, fixed by the 2 vCPU resize, not by the cap).
- **Runtime-level memory budgets in AILANG itself** — that is [m-mem-budget-runtime](../v0_31_0/m-mem-budget-runtime.md) (`MEM001`, extension lane); this doc is validator-infrastructure only.
- **Making the check heap-lean (compiler memory work)** — [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) owns compile cost; the #1811 constraint-multiplication fix already shipped in v0.54.8.
- **Client-side changes** — `ailang publish` already prints 422 bodies verbatim; publishers on v0.54.x benefit without upgrading.
- **Publishing the validator image as a general "check environment" for users** — the reproduction command is a debugging affordance, not a supported product surface.

## Timeline

**Week 1** (~4 days):
- Phase 0 deploy + celestial republish (hours, gated on Mark's route decision)
- Phase 1 refusal parity (1 day)
- Phase 2 GOMEMLIMIT (1 day)
- Phase 3 telemetry + docs (1 day)
- Testing, changelog, release tag v0.56.1 (which also carries the Phase-0 deploy if the tag route is chosen)

**Total: ~4 days across 1 week.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| GOMEMLIMIT GC death spiral pushes wall past 200 s (the ailang_parse failure mode returns via the memory door) | Med | Margin (cap − 64 MiB) leaves GC headroom; wall cap unchanged as the bounded trade; measured in manual testing before default-on; can be set to "log-only" via one env var flip |
| gVisor/gen2 exposes no cgroup files and `/proc/meminfo` reports host RAM, not the instance limit → cap too high | Med | Already the case in v0.56.0's derived cap; the refusal's cap-source line makes it visible; keep the kill watchdog; banked peaks would show 503s if the cap overshot the box |
| Banked `check_peak_rss` is a sampled peak (100 ms) and under-reports fast spikes | Low | Accepted: it is a *distribution* instrument, not a gate; `VmHWM` comparison deferred (V10) |
| Refusal message grows past what publishers read | Low | Environment block is fixed-width, machine-parseable lines first, prose second |
| Metadata schema growth (v2 field creep) | Low | Two `omitempty` fields, v1 decodes untouched — the exact pattern the v2 signature fields shipped under |
| Deploy regression on the ONE validator service (the 2026-09-17 PUB005 fleet outage class) | Med | Release-tag route is the standing rule; manual rebuild recipe documented; rollback is `gcloud run services update --image …:v0.56.0` |

## Related Documents

**Implemented (informs this design):**
- v0.56.0 changelog entry "registry validator memory cap follows the instance limit" ([changelogs/v0.32-current.md](../../../changelogs/v0.32-current.md)) — the fix this doc ratifies and extends; no design doc existed for it (V2 below)
- [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) (planned v0.55.0) — #1811's aftermath; owns compile cost itself, supplies the linux/x86 measurement methodology (VmHWM polling) this doc reuses
- [m-mem-budget-runtime](../v0_31_0/m-mem-budget-runtime.md) (planned v0.31.0) — runtime-level `MEM001` budget; complementary lane, cited to keep the boundary explicit

**Planned (check for overlap):**
- None overlapping — the create-script search (SimHash + neural) returned no matches for "validator check budget parity"; the closest thematic docs are listed above with their distinct scopes.

**Operational references:**
- `cloudbuild-registry-validator.yaml` — release-gated deploy rule, emergency rebuild recipe, rollback tags
- `docker/Dockerfile.registry-validator` — the reproduction environment (linux/amd64, Go 1.26.10, bundled std)

## References

- [Design Axioms](/docs/docs/references/axioms.mdx) — the 12 non-negotiable principles
- [docs/docs/guides/package-publishing.md](/docs/docs/guides/package-publishing.md) — the guide gaining the budget troubleshooting section
- Issue/inbox source: registry validator memory budget report for `sunholo/celestial@0.6.0` (2026-11-02), same class as the `ailang_parse@0.54.0/0.55.0` validator-budget reports

## Verification Log

Every load-bearing claim above was checked against the code in this checkout (HEAD = "Release v0.56.0", clean tree; the sandbox has an `ailang` v0.55.0 binary but **no Go toolchain**, so test *behaviors* are cited from reading the test bodies, not re-executed — the sprint-planner should re-run them at execution time):

| # | Claim | Evidence |
|---|-------|----------|
| V1 | v0.56.0 fix exists at HEAD: cap = instance limit − 128 MiB, sources cgroup v2/v1/meminfo, logged at startup | Read `cmd/registry-validator/bounded.go` (`checkRSSHeadroom`, `instanceRSSCap`, `memLimitSources`) and `cmd/registry-validator/main.go` (`checkRSSCap, capSource = instanceRSSCap(os.ReadFile)` + log line) |
| V2 | No design doc existed for that fix | `grep -rln "memory cap\|instanceRSSCap\|checkRSSCap\|memory budget" design_docs/` → no doc matches (only unrelated hits); v0.56.0 changelog entry links no Design doc (the doc gap this task closes) |
| V3 | Incident match: changelog names `sunholo/celestial@0.6.0` refused at 384 MiB on a 1 GiB box, 2026-11-02, and the 2cf025e instance resize after ailang_parse@0.55.0 wall failures | Read `changelogs/v0.32-current.md` v0.56.0 first entry (lines 9–21) |
| V4 | The refusal text and the local-measurement advice are exactly as the publisher quoted | Read `budgetError.Error()` in `bounded.go` — string contains both sentences verbatim |
| V5 | The validator runs `ailang check --package .` via `runBounded(dir, checkRSSCap, checkWallCap, procRSS, ...)` and turns a `budgetError` into HTTP 422 | Read `cmd/registry-validator/validate.go` (`runAilangCheck`) and `main.go` `handlePublish` (`jsonError(w, http.StatusUnprocessableEntity, ...)`) |
| V6 | The client prints 422 bodies verbatim (no client change needed) | Read `cmd/ailang/pkg_publish.go` `StatusUnprocessableEntity` case |
| V7 | `runBounded` sets no child env; no GOMEMLIMIT/GOGC anywhere in validator sources | `grep -rn "GOMEMLIMIT\|GOGC\|Setenv\|cmd.Env" cmd/registry-validator/*.go` (non-test) → no hits (negative-existence claim, per the doc-skill rule) |
| V8 | `ValidationResult` has no check-peak/wall fields today | Read `internal/pkg/registry_types.go` `ValidationResult` struct (full field list: compiles/effects/contracts*/ailang_version/contracts_error/contracts_counterexample) |
| V9 | `AILANG_NO_CACHE` disables only the compile cache | Read `internal/config/compiler.go` (`EnvNoCache` doc: "1 disables the compile cache for this run") |
| V10 | Current code reports the *sampled peak* and kills only when a sample strictly exceeds the cap → printed `observed` must be ≥ limit+1 MiB; the reported "observed 384 MiB" (== limit) cannot come from the current code, so the deployed v0.55.0's observed semantics differed and are not inspectable here (single squashed commit "Release v0.56.0") | Read `runBounded` (peak tracking, `rss > rssCap` kill, `budgetError{observed: mib(peak)}`); `git log` shows one commit |
| V11 | Existing tests that must stay true: `TestRunBounded_OverMemoryIsKilledAndNamed` (asserts `observed == "600 MiB"`, `limit == "384 MiB"`, prompt kill), `TestInstanceRSSCap_FollowsLimit` (7 cases incl. 512 MiB → 384 MiB), `TestProcRSS_ReadsSelfOnLinux`, `TestRunBounded_ChildFailureIsNotBudget` | Read the bodies in `cmd/registry-validator/bounded_test.go` |
| V12 | The validator's check child is the validator's own bundled linux/amd64 binary with bundled std (arch/platform difference vs darwin/arm64 publisher is real) | Read `docker/Dockerfile.registry-validator` (CGO_ENABLED=0 GOOS=linux GOARCH=amd64; `COPY std/ /usr/local/share/ailang/std/`) |
| V13 | Validator deploys move on release tags only; manual rebuild recipe and rollback exist | Read the header comment and steps of `cloudbuild-registry-validator.yaml` |
| V14 | No new error-code allocation needed | The budget refusal is the 422 body string, not a `PUBnnn`/`MODxxx` code (V5); `grep` shows no code constant introduced by this design |

## Future Work

- `runAilangVerify` (Z3) gains the same pacing + stats (Deferred Decision).
- `VmHWM` (`/proc/<pid>/status`) as a second, sampling-free peak source for the banked stat.
- Alerting on the banked peak distribution (the fleet 99th percentile vs the derived cap) so instance resizing is data-led, not incident-led.
- Read-side: check cost surfaced in the package explorer and `ailang pkg` output.

---

**Document created**: 2026-11-02
**Last updated**: 2026-11-02

## Executor verification — 2026-11-03

Implementation is committed on `coordinator/task-76187768`. Independent review found no concrete implementation defect and awarded acceptance 30/30, but returned **FAIL, 65/100** because the required core gate is red and lint tooling is unavailable (also deducted for this previously stale status). Both touched Go packages, go vet, formatting, boundaries, file sizes and changelog checks pass. See [execution evidence](evidence/m-validator-check-budget-parity/execution.md). Docker/staging, GC-heavy measurements and new production rollout remain pending; no performance benefit or new production success is claimed.
