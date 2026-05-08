# M-MOTOKO-EXT-PER-TASK: Per-Invocation motoko Extension Configuration

**Status**: Planned
**Target**: v0.19.0
**Priority**: P1 (High — unblocks per-extension benchmark analysis; replaces the v0.18.0 single-config Dockerfile with a per-invocation architecture)
**Estimated**: 1.5–2 days (~250 LOC across both repos)
**Dependencies**:
- ✅ M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) — ships the single-config adapter baseline this design REPLACES. Local-validation pass on v0.18.0 is the gating event for this sprint to start.
- ✅ M-MOTOKO-EVAL-INSTRUMENTATION (motoko commits 0c006be + 84fa449) — schema v1 JSONL contract motoko emits per-task is unchanged by this work
- 🔵 motoko-side `MOTOKO_REGISTRY_OVERRIDE` env var support (~5 LOC change to motoko_agent's startup; this doc tracks it as M0)

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-08

**Replaces** (architecturally; v0.18.0 ships first as the kernel-with-baked-extensions baseline): the build-time extension-baking model in `docker/Dockerfile.agent-motoko`. This sprint converts that to a kernel-only base image with per-task extension installation.

---

## Problem Statement

`docker/Dockerfile.agent-motoko` (v0.18.0) clones `sunholo-data/motoko_agent@84fa449` and installs whichever motoko-ext-* packages are pinned in *that fork's* `ailang.toml`. One-config-fits-all.

This breaks the moment two consumers want different extension sets:

1. **Per-user customization**: User A wants `compose + omnigraph + context_mode`; User B wants `compose + a2a + mcp`. Today they need separate Docker image builds — high ops overhead, can't compare in one eval-suite run.

2. **Per-extension benchmark analysis** (the **M-BENCH-MOTOKO-EXTENSIONS** sprint proposed in v0.18.0's "Future Work"): "What's the harness lift from `omnigraph` specifically?" requires running `motoko-bare` and `motoko-with-omnigraph` against the same task, same model. Today: impossible without rebuilding the image between runs.

3. **Threshold-measurement experiment**: the v0.18.0 sprint's strategic goal is "does motoko's tuned harness lift cheap models?". Without per-extension breakdown, the answer is "yes, the bundle of extensions does X" — not "the lift comes from extension Y; extensions Z and W contribute marginally." The bundle answer is interesting; the per-extension answer is **actionable** for harness designers.

4. **Cost-arbitrage thesis quantification**: cheap-model + tuned-harness ≈ frontier-model is the strategic claim. With per-extension data: "what's the *minimum* extension set to clear the AILANG-correctness threshold at price point P?" becomes a real procurement question. Without it: handwaving about "the harness".

**Current State (v0.18.0):**
- 1 Docker image per build
- 4 motoko-* models.yml entries (haiku/sonnet/glm-5/gemma-4) all running the SAME extension set
- Per-extension contribution: **unmeasurable**

**Impact:**
- **Strategic**: the v0.18.0 threshold-measurement experiment ships an answer at the wrong granularity (bundle, not per-extension)
- **Tactical**: M-BENCH-MOTOKO-EXTENSIONS (proposed v0.19+ in v0.18.0's Future Work) needs this OR a separate per-image-rebuild architecture
- **Operational**: each new motoko consumer (downstream user, fork maintainer) needs a Docker image build — supply-chain choke point

---

## Goals

**Primary Goal:** Replace v0.18.0's build-time extension baking with per-invocation extension specification. One base `agent-motoko` image; each Execute call self-configures via `Task.Metadata["motoko_extensions"]`.

**Success Metrics:**
- Base image size: smaller than v0.18.0 (no baked extension packages)
- Cold-cache install time per task: ≤ 30s for a typical 4-extension stack (network-bound; one-shot per (extension-list, image) tuple)
- Hot-cache install time: ≤ 100ms (just symlink + env var)
- `ailang eval-suite --models motoko-claude-haiku-4-5-bare,motoko-claude-haiku-4-5-full --benchmarks <agent-tier>` produces TWO comparable rows in a single run — proving per-extension comparison works without image rebuild
- Allowlist policy file blocks an unauthorized package install with a clear error in cloud (prod) mode

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Per-invocation extension config (vs build-arg parameterization or runtime ConfigMap) | Architecturally correct for multi-tenant + per-extension benchmarking; user-confirmed today as "the right answer" | human | design | high (commits to runtime install path; reverting means rebuilding the cache + config layer) |
| Motoko-side env var `MOTOKO_REGISTRY_OVERRIDE` (vs writing into motoko's source tree) | Keeps motoko_agent fork unchanged on disk per task; eval-harness adapter owns its own tmpdir; no race conditions with parallel task execution | human | design | low (one ~5-LOC change to motoko's startup script) |
| Content-hash cache keyed by sorted extension list (vs per-task fresh install) | Without caching, every task pays the install cost (~10-30s); with caching, only the first task per (extension-list, image) tuple pays | human | design | med (cache invalidation is the hard part; mitigated by including ailang version in the hash) |
| Allowlist policy as a YAML file in this repo (vs env var or runtime API) | Single source of truth, version-controlled, reviewable in PRs; no runtime mutation surface for malicious agents | human | design | low |
| Bare/full pair convention for models.yml entries (vs deeply parameterized model names like `motoko-haiku-4-5-with-compose-and-omnigraph`) | Two clean rows per model is mechanically diffable; deep parameterization explodes the model count and breaks A7 (Machines First) | agent | implementation | low |
| Cache directory under `~/.ailang-motoko-cache/` (vs `/tmp` or workspace-local) | Per-user cache survives container restarts (in cloud, mount the user's GCS-backed cache as a volume); workspace-local would force re-install per task | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Per-invocation specification via `Task.Metadata["motoko_extensions"]` (NOT a new dedicated `Task.Extensions` field — namespace-prefixed keys keep the `Task` struct stable)
- [x] motoko-side env var name: `MOTOKO_REGISTRY_OVERRIDE` (mirrors motoko's existing `MOTOKO_SESSION_ID` / `MOTOKO_CONFIG` env-var convention)
- [x] Cache directory: `~/.ailang-motoko-cache/<sha256-of-sorted-list>/` — the directory contains the generated `registry_generated.ail` + `ailang.lock` + a minimal `ailang.toml`
- [ ] Allowlist policy file location: `internal/eval_harness/motoko_extensions_policy.yml` (recommended) vs `cmd/ailang/policy.yml` (alternative). Pick before M5.
- [ ] Pre-warm strategy: bake the top-5 most-popular extension hashes into the base image at build time, OR populate them at first container start? Pick before M3.

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Cache eviction policy (LRU? size cap? TTL?) — agent may choose; default to none for v0.1 (cache grows unbounded; revisit if a cloud Job hits disk-quota limits in dev)
- Whether to warn vs error when an extension version requested in `Task.Metadata` is not in the lockfile after install (i.e. registry resolved a different version) — agent may choose; recommend warn + log + use whatever was resolved
- Per-extension OTEL span attributes (e.g. `motoko.extensions.installed = ["a2a@0.1.1", ...]`) — agent may choose; nice-to-have for observability but not required
- The exact CLI surface for inspecting the cache (`ailang motoko-cache list` / `clear` / `inspect`) — defer to a follow-up `M-MOTOKO-CACHE-CLI` if usage warrants
- Whether `models.yml` entries can reference the policy file by name (e.g. `motoko_policy: "audited-prod"`) for env-aware extension filtering — defer to a follow-up if multiple policy variants are needed

---

## Solution Design

### Overview

motoko_agent is a "kernel" (the AILANG runtime + tool dispatcher + bun TUI wrapper). Extensions are a "userland" (motoko-ext-* packages installed on top via `[extensions]` in `ailang.toml` → `ailang generate-extension-registry` → motoko reads `registry_generated.ail` at startup).

This design moves extension installation from **build time** (today) to **per Execute call**. The Docker image becomes the kernel only. Each Execute writes a per-task ailang.toml in a content-hashed cache dir, runs `ailang install + lock + generate-extension-registry`, and points motoko at the result via `MOTOKO_REGISTRY_OVERRIDE=<path>`.

### Architecture

**Components:**

1. **motoko-side env var support** (~5 LOC in `motoko_agent/scripts/run-agent.sh` or equivalent startup):
   - When `MOTOKO_REGISTRY_OVERRIDE` is set and points to a readable file, use it as the registry source instead of `src/core/ext/registry_generated.ail`.
   - Backward-compat: when unset, behavior matches v0.18.0 exactly (uses the in-tree registry).

2. **Adapter `Execute` flow extension** (`internal/executor/motoko/motoko.go` + new `internal/executor/motoko/extensions.go`):
   - Parse `task.Metadata["motoko_extensions"]` → `[]string` of `vendor/name@version` refs (empty = bare motoko).
   - **Allowlist check** against `motoko_extensions_policy.yml` for the current env (via `AILANG_ENV` env var: `dev` / `test` / `prod`). Reject with a clear error if any package isn't allowed.
   - Compute `hash = sha256(sorted(extensions) + ailang_version + motoko_commit)`. Including ailang_version in the hash ensures cache invalidation when AILANG releases a new stdlib that affects extension semantics.
   - **Cache lookup**: if `~/.ailang-motoko-cache/<hash>/registry_generated.ail` exists, use directly (warm-cache fast path).
   - **Cold cache**: in `~/.ailang-motoko-cache/<hash>/`:
     a. Write a minimal `ailang.toml` with `[extensions].packages = [<list>]` + the standard `config_import` / `hooks_import` / `registry_import` / `output` fields (mirroring motoko_agent's own ailang.toml structure)
     b. Run `ailang install` (resolves + downloads packages from registry)
     c. Run `ailang lock` (records exact versions + hashes)
     d. Run `ailang generate-extension-registry` (produces `registry_generated.ail`)
     e. `chmod -R a-w` the directory (immutable cache entry; protects against concurrent task corruption)
   - Set `MOTOKO_REGISTRY_OVERRIDE=<hash-dir>/registry_generated.ail` in the spawn env.
   - Spawn motoko as before. Motoko reads the override-pointed registry; everything else is unchanged.

3. **Allowlist policy file** (`internal/eval_harness/motoko_extensions_policy.yml`):
   ```yaml
   environments:
     dev:
       allow_pattern: "sunholo/motoko_ext_*"   # glob — any sunholo registry extension OK in dev
     test:
       allow_explicit:
         - "sunholo/motoko_ext_compose@0.1.0"
         - "sunholo/motoko_ext_omnigraph@0.1.0"
         # ... explicit list, not pattern
     prod:
       allow_explicit:
         - "sunholo/motoko_ext_compose@0.1.0"   # audited
         - "sunholo/motoko_ext_omnigraph@0.1.0"  # audited
   ```
   Adapter loads at startup (cached), checks per-task. Rejection path: `Result.Success = false`, `Result.Error = "extension <pkg> not in allowlist for env <env>"`.

4. **Dockerfile rewrite** (`docker/Dockerfile.agent-motoko`):
   - REMOVE: `git clone motoko_agent` + `ailang install motoko-ext-*` + `ailang generate-extension-registry`
   - KEEP: bun runtime install + AILANG binary inheritance from `agent-base`
   - ADD: pre-clone motoko_agent stem (kernel only — `ailang.toml` has empty `[extensions].packages = []`); copy startup script honoring `MOTOKO_REGISTRY_OVERRIDE`
   - Result: smaller image (no extension binaries baked); cold start is per-task install latency (mitigated by cache layer)

5. **models.yml restructure** — replace 4 entries with 8 (bare/full pairs):
   ```yaml
   motoko-claude-haiku-4-5-bare:
     # ... (all v0.18.0 fields)
     metadata:
       motoko_extensions: ""

   motoko-claude-haiku-4-5-full:
     # ... (same fields, different model_family suffix)
     metadata:
       motoko_extensions: "sunholo/motoko_ext_compose@0.1.0,sunholo/motoko_ext_omnigraph@0.1.0,sunholo/motoko_ext_context_mode@0.1.0,sunholo/motoko_ext_a2a@0.1.1"
   ```
   Same pattern for sonnet, glm-5, gemma-4. `agent_suite` membership doubles from 4 to 8 motoko entries.

6. **Eval-harness Task construction** (`internal/eval_harness/agent_runner.go` or equivalent):
   - When constructing a `Task` for a motoko model, copy the model's `metadata` map into `Task.Metadata`. The `motoko_extensions` key flows through automatically.

### Implementation Plan

**Phase 1: Foundation** (~3 hours)
- [ ] M0: motoko-side `MOTOKO_REGISTRY_OVERRIDE` env var support — small PR on `sunholo-data/motoko_agent` `motoko-dx-compaction-pending` branch (or dev branch, depending on review state). 5 LOC + 1 inline test asserting the override path is honored when the env var is set.
- [ ] M1: Allowlist policy file (`internal/eval_harness/motoko_extensions_policy.yml`) + loader (`internal/eval_harness/motoko_policy.go`, ~80 LOC): `LoadMotokoPolicy(env string) (*MotokoPolicy, error)` + `IsAllowed(pkg string) (bool, string)`. Tests: glob-match, explicit-list, env-not-found, malformed-policy.

**Phase 2: Adapter refactor** (~6 hours)
- [ ] M2: New `internal/executor/motoko/extensions.go` (~150 LOC):
  - `parseExtensionList(s string) ([]string, error)` — comma-split + validate `vendor/name@version` shape
  - `cacheDir(extensions []string, ailangVersion, motokoCommit string) string` — content-hash dir
  - `prepareRegistry(ctx, extensions []string, policy *MotokoPolicy) (registryPath string, err error)` — orchestrates allowlist check + cache lookup + cold install + chmod-immutable
- [ ] M3: Wire into `motoko.go::ExecuteStreaming`: before subprocess spawn, call `prepareRegistry`; on success, add `MOTOKO_REGISTRY_OVERRIDE=<path>` to env; on policy reject, return Result with `Success: false, Error: <reason>` (don't spawn motoko at all)
- [ ] M4: Tests: `TestPrepareRegistry_BareNoExtensions`, `TestPrepareRegistry_FullStack`, `TestPrepareRegistry_AllowlistRejects`, `TestPrepareRegistry_HotCacheHit`, `TestPrepareRegistry_HashIncludesAilangVersion` (~120 LOC, target ≥80% coverage on `extensions.go`)

**Phase 3: Dockerfile + wiring** (~2 hours)
- [ ] M5: Rewrite `docker/Dockerfile.agent-motoko` — strip extension installation; add a `kernel-only` motoko_agent clone with empty `[extensions].packages`; honor `MOTOKO_REGISTRY_OVERRIDE`. Verify `docker build` succeeds + `motoko --version` resolves
- [ ] M6: Pre-warm strategy implementation (whichever the Design Freeze decision lands on)
- [ ] M7: Update `docker/agent-motoko-multivac-prs.md` to reflect the new architecture (smaller cloudbuild, no per-config image variants needed)

**Phase 4: Models split + integration** (~3 hours)
- [ ] M8: `internal/eval_harness/models.yml` — replace 4 motoko entries with 8 bare/full pairs; update `agent_suite` + `harness_suite` memberships
- [ ] M9: Eval-harness Task construction — thread model `metadata` into `Task.Metadata` (verify this already happens via `agent_runner.go`'s existing logic; if not, add)
- [ ] M10: Integration test: `TestEvalSuite_MotokoBareVsFull` — constructs both a bare and a full Task, runs against the mock binary fixture, asserts both produce valid Results with different `Result.ProviderData["motoko_extensions"]` values
- [ ] M11: Update `internal/executor/motoko/README.md` with the per-invocation-extensions section + the `Task.Metadata["motoko_extensions"]` API

**Phase 5: Validation + finalize** (~2 hours)
- [ ] M12: Local end-to-end run: `ailang eval-suite --models motoko-claude-haiku-4-5-bare,motoko-claude-haiku-4-5-full --benchmarks <small-tier>` against real OPENROUTER_API_KEY. Verify both rows produce non-zero results with different cost / token / pass-rate profiles
- [ ] M13: CHANGELOG entry under `[Unreleased]` with the per-extension comparison numbers from M12
- [ ] M14: Move design doc + sprint plan to `design_docs/implemented/v0_19_0/` with implementation report

### Files to Modify/Create

**New files (this repo):**
- `internal/executor/motoko/extensions.go` (~150 LOC) — core per-invocation extension logic
- `internal/executor/motoko/extensions_test.go` (~120 LOC) — unit tests
- `internal/eval_harness/motoko_extensions_policy.yml` (~30 lines YAML) — env-aware allowlist
- `internal/eval_harness/motoko_policy.go` (~80 LOC) — policy loader + IsAllowed check
- `internal/eval_harness/motoko_policy_test.go` (~60 LOC) — policy tests

**Modified files (this repo):**
- `internal/executor/motoko/motoko.go` (+~30 LOC, -~5 LOC) — call `prepareRegistry` before spawn; thread MOTOKO_REGISTRY_OVERRIDE into env
- `internal/executor/motoko/README.md` (+~50 lines) — new section on per-invocation extension API
- `docker/Dockerfile.agent-motoko` (+~10 LOC, -~25 LOC) — strip extension install; smaller kernel-only build
- `docker/agent-motoko-multivac-prs.md` (+~30 LOC, -~10 LOC) — updated multivac PR steps
- `internal/eval_harness/models.yml` (+~250 lines) — 4 → 8 motoko entries, agent_suite + harness_suite updates
- `internal/eval_harness/ttft_config_test.go` (+1 line) — harness_suite count adjustment
- `changelogs/v0.10-current.md` (+~40 lines) — `[Unreleased]` entry with M12 measurement numbers

**Modified (motoko_agent repo, separate small PR before this sprint starts):**
- `scripts/run-agent.sh` (or equivalent startup) (+~5 LOC) — honor `MOTOKO_REGISTRY_OVERRIDE` env var
- motoko's own test (+~20 LOC) — assert override path is read when env var is set

---

## Examples

### Example 1: Per-extension comparison in a single eval-suite run

**Before (v0.18.0):**
```bash
# To compare motoko-with-omnigraph vs motoko-bare on the same model:
docker build -f Dockerfile.agent-motoko-bare ...
docker build -f Dockerfile.agent-motoko-with-omnigraph ...
# Now run two separate eval-suites against two separate Cloud Run Jobs.
# Compare results manually. Repeat for every extension combination.
```

**After (this design):**
```bash
ailang eval-suite \
  --models motoko-claude-haiku-4-5-bare,motoko-claude-haiku-4-5-full \
  --benchmarks tier_a

# Output (single run, two rows):
# Model                              | Pass | Cost   | Tokens | Cache Read
# -----------------------------------|------|--------|--------|-----------
# motoko-claude-haiku-4-5-bare       | 24/41| $0.38  | 340K   | 12K
# motoko-claude-haiku-4-5-full       | 33/41| $1.18  | 920K   | 256K
```

The `-full` variant lifts pass rate by 22% at 3.1x cost. Per-extension lift becomes a measurable axis.

### Example 2: Policy-rejection in prod

**Cloud Run Job env: `AILANG_ENV=prod`** with the policy file's prod block listing only audited packages.

```bash
# Coordinator dispatches a task with an unaudited extension:
ailang messages send coordinator '{
  "type": "task",
  "executor_variant": "motoko",
  "metadata": {
    "motoko_extensions": "sunholo/motoko_ext_compose@0.1.0,suspicious/random_ext@0.1.0"
  }
}'

# Adapter rejects BEFORE spawning motoko:
# Result.Success = false
# Result.Error = "extension suspicious/random_ext@0.1.0 not in allowlist for env prod"
# DurationMS = 12 (just the policy check; no LLM cost)
```

### Example 3: Hot-cache hit (zero-overhead repeat invocation)

```bash
# First invocation with a particular extension stack — cold cache, ~22s install
$ ailang eval-suite --models motoko-claude-haiku-4-5-full --benchmarks tier_a/task_001
# [Cache MISS for hash sha256:abc123... — installing 4 extensions...]
# [Cache populated in ~/.ailang-motoko-cache/abc123...]

# Second invocation with same stack — hot cache, ~50ms
$ ailang eval-suite --models motoko-claude-haiku-4-5-full --benchmarks tier_a/task_002
# [Cache HIT for hash sha256:abc123... — using cached registry]
```

---

## Conflict Surface

This is **not** a parser/lexer/typechecker change — it's an addition to the executor adapter framework + a new Task.Metadata key + a Dockerfile rewrite. The Conflict Surface section is therefore not strictly required for code-correctness reasons. However, the per-invocation API has its own surface that must be respected:

### Touchpoints in the Task / executor framework

- `executor.Task.Metadata` is a `map[string]string` already used by other executors for provider-specific options. **Mitigation**: namespace all motoko-introduced keys with `motoko_` prefix (`motoko_extensions`, `motoko_profile`, etc.) so collisions with other executors' keys are syntactically impossible.
- `executor.Result.ProviderData["motoko_extensions"]` (new) — surfaces the resolved extension list back to the harness for analysis. Additive; existing consumers ignore unknown ProviderData keys.
- `models.yml` `metadata: { ... }` field — already supported by `LoadModelsConfig`; reading it for motoko entries is a no-op for non-motoko entries.

### Programs that MUST still work

After this sprint ships, the following must remain functional:

1. **v0.18.0 motoko adapter behavior with empty `Task.Metadata["motoko_extensions"]`**: should resolve to bare motoko (no extensions installed). Validates the bare-default path.
2. **Existing models.yml expansion**: every non-motoko model entry continues to resolve correctly. The new `metadata` field on motoko entries is ignored by other executors.
3. **`ailang eval-suite --models agent_suite`**: includes both bare and full motoko variants now (8 motoko entries instead of 4); doesn't break the eval-suite cost-budget defaults.
4. **Cloud Run dispatch with `--executor motoko`**: works against the new kernel-only image; the cache layer mounted as a volume (or per-container-restart populated) handles per-task installs.
5. **`make test`** + **`make ci`**: the new policy loader + extensions module pass full test suite + lint clean.

These become regression test fixtures in M10.

### What deliberately changes

- `motoko-claude-haiku-4-5` (and the 3 sister entries) are **renamed** to `motoko-claude-haiku-4-5-bare` (no extensions, harness-lift baseline) and `motoko-claude-haiku-4-5-full` is added (full extension stack). The old name is removed; consumers using `--models motoko-claude-haiku-4-5` get a clear "model not found; did you mean motoko-claude-haiku-4-5-bare or motoko-claude-haiku-4-5-full?" error. The CHANGELOG entry surfaces this rename prominently.
- `Dockerfile.agent-motoko` no longer ships any motoko-ext-* packages baked in. Production deployments using the old image continue to work (motoko's `ailang.toml` already pins extensions; `MOTOKO_REGISTRY_OVERRIDE` unset = use in-tree registry). New deployments using the new image MUST set `motoko_extensions` per task or get bare motoko.

---

## Testing Strategy

**Unit tests** (`internal/executor/motoko/extensions_test.go` + `internal/eval_harness/motoko_policy_test.go`):
- `TestPrepareRegistry_BareNoExtensions` — empty extension list → empty registry, no install
- `TestPrepareRegistry_FullStack` — multi-extension stack → all packages resolved, lockfile written, registry generated
- `TestPrepareRegistry_AllowlistRejects` — extension not in policy → clear error, no install attempt
- `TestPrepareRegistry_HotCacheHit` — second call with same hash → no `ailang install` invocation (mock the install function and assert it's NOT called)
- `TestPrepareRegistry_HashIncludesAilangVersion` — same extension list, different ailang version → different hash → cache miss
- `TestPolicyLoad_DevAllowsPattern` — dev policy with `allow_pattern: "sunholo/motoko_ext_*"` → matches glob
- `TestPolicyLoad_ProdRejectsUnaudited` — prod policy with explicit list → rejects packages not in list
- `TestPolicyLoad_MalformedFile` — invalid YAML → clear error pointing at the file

**Integration tests:**
- `TestEvalSuite_MotokoBareVsFull` — constructs Tasks for both bare and full variants, runs against the mock motoko binary fixture, asserts both produce valid Results with different `Result.ProviderData["motoko_extensions"]` values
- `TestModelsYml_MotokoBareFullPairs` — every motoko-* model has either `-bare` or `-full` suffix; both variants exist for each base model

**Cross-repo smoke (after motoko-side env var lands):**
- Manual: `MOTOKO_REGISTRY_OVERRIDE=/tmp/test-registry.ail motoko "say hello"` against a small canned registry; verify motoko reads from the override path

**Regression-surface tests** (per Conflict Surface above):
- `TestEvalSuite_OtherExecutorsUnaffected` — non-motoko entries in `agent_suite` continue to expand correctly
- `TestExecutorTask_MetadataPassthrough` — non-motoko executors ignore `Task.Metadata["motoko_*"]` keys cleanly

**Coverage target:** ≥80% on `extensions.go` and `motoko_policy.go`.

---

## Non-Goals

**Not in this feature:**
- Cache eviction (LRU, TTL, size cap) — defer to a follow-up if cloud Job dev disks fill up under real eval-suite load
- A CLI tool for cache management (`ailang motoko-cache list/clear/inspect`) — defer to `M-MOTOKO-CACHE-CLI` follow-up if usage warrants
- Per-extension OTEL span attributes beyond the basic `ProviderData["motoko_extensions"]` round-trip — nice-to-have for observability but not required for the per-extension benchmark experiment
- Changes to motoko's own extension ABI — this sprint is purely about WHICH extensions get loaded; the extensions themselves are unchanged
- A web UI for browsing the cache or comparing extension stacks — the eval-suite output is sufficient for the v0.19 measurement; a UI is M-MOTOKO-EXT-DASHBOARD territory

---

## Timeline

**Day 1** (~6 hours):
- M0 (motoko-side env var, ~1h)
- M1 (policy file + loader, ~2h)
- M2-M3 (extensions module + adapter wiring, ~3h)

**Day 2** (~6 hours):
- M4 (extension tests, ~2h)
- M5-M7 (Dockerfile + multivac-prs.md + pre-warm, ~3h)
- M8-M9 (models.yml split + Task metadata threading, ~1h)

**Day 3 (half-day buffer)** (~3 hours):
- M10-M11 (integration test + README, ~1.5h)
- M12-M14 (local validation + CHANGELOG + finalize, ~1.5h)

**Total: ~15 hours across 2.5 working days.**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `ailang install` fails mid-task (registry outage, network glitch) — task fails with a confusing error | Medium | Adapter wraps install in `Result.Success = false, Error = "extension install failed: <details>"` with a clear message + ProviderData["install_phase"] = "ailang_install"; the eval-harness retries the run (per existing flake-handling) |
| Allowlist policy file gets out of sync with what's actually published to the registry — task fails with "package not found" instead of "package not allowed" | Low | Adapter checks allowlist BEFORE attempting install; clear error in this case; M1's tests cover the policy-rejection path |
| Cache directory grows unbounded in long-running cloud Jobs | Medium | Defer eviction policy (Non-Goal); document as known limit; revisit if a cloud Job hits disk-quota issues in dev |
| Concurrent tasks racing on the same cache directory (two Tasks with same extension hash arriving simultaneously) | Low | `chmod -R a-w` after population means concurrent tasks see either "directory exists, use it" or "directory missing, install". Use file-lock during install (`flock` on the cache dir) to serialize concurrent cold-cache populations. |
| motoko-side `MOTOKO_REGISTRY_OVERRIDE` change breaks motoko's existing TUI users | Low | Backward-compat: when env var unset, behavior matches v0.18.0 exactly. Motoko TUI doesn't set the env var, so it sees no behavior change. |
| Per-task install adds ~20s latency for cold cache; eval-suite for 50 tasks pays 50x that without caching | High (initial), Low (after caching lands) | Cache is core to the design — without it, this design isn't viable. M10's integration test must verify hot-cache hits skip the install. Pre-warm strategy (M6) baking popular hashes into the base image cuts first-task cost too. |
| Cloud Run Job stateless containers lose the cache between Job invocations | Medium | Two mitigations: (1) per-container the cache survives across multiple Tasks dispatched to the same Job execution (most common case); (2) optional GCS-backed cache mount as a follow-up if cross-Job warm cache becomes necessary |
| Policy file becomes a permissions/governance bottleneck (every new extension needs a PR review) | Low | Acceptable cost — policy review IS the audit. Dev env uses pattern-match for fast iteration; only prod requires explicit-list updates. |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Content-hash cache means same extension list → same registry → same motoko behavior; reproducible across runs |
| A2: Replayability | +1 | Cache directory is a portable artifact — debug a task's behavior by inspecting `~/.ailang-motoko-cache/<hash>/registry_generated.ail` |
| A3: Effect Legibility | +1 | All effects (install network, lock file write, registry generation) are gated behind the per-task `prepareRegistry` boundary; no implicit side effects |
| A4: Explicit Authority | +2 | Allowlist policy file IS explicit-authority for which packages each env can install; without it, the adapter could install arbitrary code; this is the textbook A4 win |
| A5: Bounded Verification | +1 | Per-task allowlist check + per-task install isolation means a failure is bounded to the task; no global state corruption |
| A6: Safe Concurrency | 0 | File-lock on cache dir during install handles concurrent Tasks; otherwise no concurrency surface change |
| A7: Machines First | +2 | Per-extension benchmark rows are mechanically diffable in eval-suite output; the bare/full naming convention is sortable + parseable; this is THE point |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Per-extension cost rollup means "this extension cost $X to add to the harness lift" becomes a leaderboard pivot |
| A10: Composability | +2 | The extension list IS the composition primitive — any subset of motoko-ext-* packages can be combined per task. Composability IS the design. |
| A11: Structured Failure | +1 | Allowlist rejection produces a typed Error with the offending package; install failures point at the failed phase (`install` / `lock` / `generate`) |
| A12: System Boundary | +1 | The per-task tmpdir + cache dir model establishes a clean boundary between motoko's filesystem (read-only kernel) and the eval-harness's per-task state (writable cache) |

**Net Score: +13** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — content-hash cache makes runs reproducible
- [x] A3 (Effects): All install effects gated behind the explicit `prepareRegistry` boundary
- [x] A4 (Authority): Allowlist policy IS the authority gate — no ambient install permissions
- [x] A7 (Machines First): The whole sprint's purpose is making per-extension data mechanically diffable

---

## References

- **Source proposal**: User feedback after M-MOTOKO-EXECUTOR-ADAPTER eval (2026-05-08): "I think it needs to be option B - each time its invoked it could have different packages specified"
- **Predecessor sprint** (ships first as the single-config baseline): [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../v0_18_0/m-motoko-executor-adapter.md)
- **Predecessor's predecessor** (motoko's schema-v1 instrumentation, the JSONL contract this design depends on): [`motoko_agent/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md`](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md)
- **Canonical executor contract**: [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md)
- **Closest analogue for cache pattern**: AILANG's own `~/.ailang/cache/compile/modules/` directory (per-package iface caching) — this design uses the same hashing + chmod-immutable pattern at a different granularity
- **Extension registry generator**: `cmd/ailang/ext_registry_gen.go` (M-AILANG-EXT-REGISTRY-GEN, v0.17.1) — this design relies on `ailang generate-extension-registry` working in arbitrary tmpdirs, which it already does
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

---

## Future Work

Features that build on this:

- **M-BENCH-MOTOKO-EXTENSIONS** (proposed v0.19+ in v0.18.0's Future Work) becomes **trivial** after this design ships — just add more `models.yml` entries with different `metadata.motoko_extensions` combinations. No code change needed. Expected next step.
- **M-MOTOKO-CACHE-CLI** — `ailang motoko-cache list` / `clear` / `inspect <hash>` for debugging + capacity management. Defer until usage warrants.
- **M-MOTOKO-EXT-MARKETPLACE** — UI surface for browsing motoko-ext-* packages with pre-baked benchmark numbers per extension. Long-term; requires per-extension benchmark data (this sprint produces it).
- **GCS-backed cross-container cache** — for Cloud Run Jobs that want warm-cache hits across Job invocations. Defer until disk-quota or cold-cache-cost becomes a measurable production issue.
- **Extension dependency graph visualization** — once we have N≥10 motoko-ext-* packages with measured contributions, render the lift contribution as a stacked-bar chart per model. UI work; out of scope for this sprint.
- **M-MOTOKO-EXT-PROVENANCE** — sign extension package versions with a deploy-time signature so the cache can verify the registry-resolved package matches what was approved. Supply-chain hardening; defer until threat model surfaces a need.
