# M-COORDINATOR-ROUTE-AUTHORITY-RECOVERY: narrowed recovery — M1 route authority + M2a OpenCode child preflight

**Status**: Planned (revision 2 — post-quorum-round-1, the one protocol-mandated revision)
**Target**: v0.35.0
**Priority**: P0 — recovery of the approved, narrowed slice of the retry-storm incident work
**Estimated**: 1.75 days (M1 apply + M1r reconcile ~0.75d, M2a ~1d)
**Dependencies**: None on unmerged code. Semantic coordination with the ALREADY-ON-DEV
dispatcher guard from `23066345e` (see M1r and Conflict Surface — this is the load-bearing one).
**Supersedes**: the recovered design `m-coordinator-child-env-opencode-retry-storm.md`
(599 lines, bundled inside commit `3500db0a7`, quorum round 2 BLOCKED) and its sprint plan
(460 lines, whose "narrow-refinement carve-out" quorum claim is REFUTED — treat as candidate
material only). This document is the single authoritative narrowed design.

## Human authority (frozen — do not re-litigate)

- **D-50: "approve"** (Mark, attended, stamped 2026-08-31). The narrowed recovery plan
  (`m-coordinator-child-env-opencode-retry-storm-recovery-plan.md`, on dev) is APPROVED and
  `execute sprint` is authorized for the narrowed scope.
- The recovery plan's Required Gate 1 (fresh quorum for the narrowed plan) applies to THIS
  document; round 1 ran and blocked (below); this is its mandated revision.
- No `.ail` code or AILANG-language claim appears in this design, so `ailang check`
  verification of language claims is not applicable.

## Quorum record and objection disposition (round 1 → this revision)

Round 1 (the 391-line version of this doc): **BLOCKED 3/3** — `gpt5-6-sol`,
`gemini-3-1-pro`, `oc-glm-5-2` all present, all reject, no absent reviewers. All three
objections were PREMISE objections; the controller ran each as a measurement this session
rather than forwarding prose. Disposition:

- **O1 (gpt5-6-sol)** — "preflight and `runExecutor` may be separate executor instances,
  so a resolution cache would not be shared." Premise FALSE AS STATED within one child
  process: the factory memoizes (V29). But it is TRUE as a constraint — the guarantee holds
  ONLY if preflight obtains its executor through the identical factory call. Converted into
  a NORMATIVE M2a constraint plus a pointer-identity test (new R6), with the process-local
  limit stated rather than overclaimed.
- **O2 (gemini-3-1-pro)** — "the doc never verified `ailang coordinator config diff`
  exists at base or reads live GCS." REFUTED by measurement (V28): the subcommand and the
  GCS-reading store both exist at base. The residually valid point stands and is fixed:
  a load-bearing rollout mitigation was ASSERTED with no Verification Log row. Row added.
- **O3 (oc-glm-5-2)** — "M1r is three prose bullets with no measured cell-by-cell
  evidence; an executor working from them could over- or under-reject." CONFIRMED, AND
  WORSE THAN FILED. The controller enumerated the full cross product of the two real
  authority tables: **104 cells, 39 disagreements, all in one direction — M1 over-rejects
  routes dev deliberately accepts; zero under-rejects — across FOUR causes, not the two
  this doc's round 1 found** (V30). The round-1 M1r bullets covered causes 1 and 4 only
  and would have left 25 cells still over-rejected. M1r is now a first-class specified
  sub-milestone with the full table and a real-function parity gate.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One agent config resolves to one provider/image/model tuple before dispatch |
| A2: Replayability | 0 | Attempt-count persistence is M3 — out of scope here |
| A3: Effect Legibility | 0 | No language-effect change |
| A4: Explicit Authority | +1 | The OpenCode child resolves one absolute executable and uses it for both health and execute |
| A5: Bounded Verification | +1 | 15-second internal health deadline; refusals are finite, typed, testable |
| A7: Machines First | +1 | Config validation rejects incoherent routes before a Cloud Run attempt is spent |
| A9: Cost Visibility | +1 | Health-before-clone converts a post-clone container burn into a cheap pre-side-effect refusal |
| A11: Structured Failure | +1 | Missing child is a typed permanent error, not a string parsed from stderr |
| A12: System Boundary | +1 | Coordinator→Cloud Run and Go→child-process boundaries validate inputs explicitly |
| A6/A8/A10 | 0 | Concurrency CAS work is M3/M4 (excluded); no syntax; resolver deliberately NOT generalized to five adapters |

**Net Score: +7** → Move forward. Hard-violation check: none.

## Problem Statement (measured incident, provenance labelled)

Task `task-a0628a5f` (2026-08-28, prod `ailang-multivac`): the coordinator combined the
global `default_provider=opencode` with the agent's `executor_variant=codex`, dispatching a
codex Cloud Run image that then ran `runExecutor("opencode")` and died on
`exec: "opencode": executable file not found in $PATH` — **after cloning 24,032 files**.
41 dispatch failures, 8 stale-terminalization attempts, 9 operator messages followed.
(Incident forensics: recovered design V1–V3/V9, first-party gcloud/log evidence 2026-08-29,
inherited; independently corroborated on dev by the commit message of `23066345e`, quoted in
Verification Log row V7 below.)

Two code facts at today's `origin/dev` (`8bb92ae2b`), both re-measured this session:

1. **The provider/variant split is still live in the daemon.**
   `daemon_tasks_exec.go:124-125` takes the provider from `d.coordConfig.DefaultProvider`;
   `:224-225` independently takes the Cloud Run image variant from `agent.ExecutorVariant`
   (V8). Config load already inherits the default into `agent.Provider` (V9), so ignoring
   the per-agent provider has no compatibility rationale.
2. **Dev ALREADY ships a dispatcher-side guard for this incident.** Commit `23066345e`
   (2026-08-31, after the recovered design was written) added
   `checkVariantProviderAgreement` + `providersForVariant` to
   `internal/dispatch/cloudrun/dispatcher.go`, grounded in the actual
   `docker/Dockerfile.agent-*` contents (V6, V7). The recovered design's premise "nothing
   ties the two together at HEAD" is now STALE. What remains missing on dev: the
   daemon-side single-route authority (provider env/budget/model still read from the global
   default), config-publication-time rejection, and the OpenCode child's own boundary.

And one fact about the child (M2a's target), re-measured:

3. **The OpenCode Cloud Run child spends before it checks.** `executeCloudTask` clones the
   plugin repo (`coordinator_cloud.go:247`), clones the workspace (`:268`), injects
   AGENTS.md (`:289`), and only at `:371` calls `runExecutor`, which is the first place a
   missing binary can surface. No `HealthCheck` call exists anywhere in the three
   cloud-child files (V17, V18). The adapter itself re-consults `PATH` at execution time:
   `HealthCheck` does `exec.LookPath(opencodePath)` (`opencode.go:568`) but
   `ExecuteStreaming` hands the original configured string — default bare `"opencode"`
   (`:48-50`) — to `exec.CommandContext` (`:156`), so health and execute can select
   different files (V15).

## Scope

### IN — M1: immutable execution-route authority (the committed unit `3500db0a7`)

Apply the already-committed, already-reviewed unit: `internal/coordinator/execution_route.go`
(+test), `agent_config.go`, `daemon.go`, `daemon_tasks_exec.go`, `daemon_tasks_budget.go`,
`daemon_tasks_init.go`, `cmd/ailang/coordinator_config.go` (+test),
`internal/dispatch/cloudrun/dispatcher.go` (+test). Root cause addressed: the incoherent
`provider=opencode` × `executor_variant=codex` route. One `ExecutionRoute` resolved from the
agent becomes the sole authority for job variant, `AILANG_PROVIDER`, model, and budget
lookup; incoherent pairs are rejected typed-and-permanent at config publication, at the
daemon, and defensively inside `Dispatch` before `RunJob`.

**M1 application contract (measured, not assumed):**

- The unit merges cleanly onto `8bb92ae2b` by THREE-WAY merge only:
  `git merge-tree --write-tree --merge-base=3500db0a7~1 HEAD 3500db0a7` → rc=0, merged tree
  `82b57a7dc` (V4). Plain `git apply --check` REJECTS the patch
  (`dispatcher_test.go` context drift from `23066345e`) (V5). The executor must apply via
  cherry-pick (3-way), never via `git apply`.
- The merged tree was extracted read-only (`git archive 82b57a7dc`) and measured green:
  `go build` over coordinator/dispatch/cmd rc=0; `go test ./internal/dispatch/cloudrun`
  rc=0; `go test ./internal/coordinator/...` rc=0; the six `TestConfigCAS_*` tests all PASS
  (V10, V11). Green is NECESSARY, not sufficient — see M1r.
- **Bundled markdown disposition:** `3500db0a7` also carries the 599-line overbroad design
  and the 460-line sprint plan as files under `design_docs/planned/v0_35_0/` (V2). They
  land for attribution (recovery-plan disposition table), but the SAME commit series must
  edit both files' status headers to `Superseded — see
  m-coordinator-route-authority-recovery.md; quorum round 2 BLOCKED; sprint-plan §0 quorum
  carve-out claim REFUTED (iteration 305)`. Landing them unannotated would put a
  doc claiming executable approval on dev.
- **Operator preflight before deploy** (no config write in this sprint): run
  `ailang coordinator config diff` — the subcommand exists at base and its store reads
  live GCS (V28) — and confirm no live cloud agent's route is rejected by the reconciled
  matrix + agent policy (in particular none configured with `eval`/`eval-go` variants,
  which remain agent-level-refused under policy P2 below). Rejection of a live legitimate
  route is a pause condition, not a fallback cell.

### IN — M1r: matrix reconciliation (first-class sub-milestone; the round-2 quorum fix)

**Why this exists.** Dev and M1 each carry a route-authority table:

- dev's guard: `knownVariants` (13 variants), `providersForVariant`,
  `binarylessProviders`, enforced by `checkVariantProviderAgreement`
  (`internal/dispatch/cloudrun/dispatcher.go:67-112`; V7);
- M1's `cloudProviderVariants` + `NormalizeExecutionVariant`
  (`3500db0a7:internal/coordinator/execution_route.go:49-83`; V12).

The controller enumerated the full cross product — 8 providers (`""`, `claude`, `gemini`,
`codex`, `opencode`, `pi`, `motoko`, `managed_agents`) × 13 variants (`""`, `default`,
`go`, `gemini`, `gemini-go`, `codex`, `codex-go`, `opencode`, `pi`, `pi-go`, `motoko`,
`eval`, `eval-go`) = **104 cells**. Dev accepts 49; M1-as-committed accepts 10; the 10 are
a strict subset of the 49, so there are **39 disagreements, every one an M1 over-reject of
a route dev deliberately accepts, and zero under-rejects** (V30). Four distinct causes.

**The 39 disagreeing cells, in full** (dev verdict / M1-as-committed verdict; fix keyed to
the reconciled semantics below):

| # | Provider | Variant | dev guard | M1 as committed | Cause | Fix |
|---|---|---|---|---|---|---|
| 1 | `managed_agents` | `""` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 2 | `managed_agents` | `default` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 3 | `managed_agents` | `go` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 4 | `managed_agents` | `gemini` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 5 | `managed_agents` | `gemini-go` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 6 | `managed_agents` | `codex` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 7 | `managed_agents` | `codex-go` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 8 | `managed_agents` | `opencode` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 9 | `managed_agents` | `pi` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 10 | `managed_agents` | `pi-go` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 11 | `managed_agents` | `motoko` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 12 | `managed_agents` | `eval` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 13 | `managed_agents` | `eval-go` | accept (binaryless) | reject: unknown cloud provider | C1 | S3 |
| 14 | `""` | `""` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 15 | `""` | `default` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 16 | `""` | `go` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 17 | `""` | `gemini` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 18 | `""` | `gemini-go` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 19 | `""` | `codex` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 20 | `""` | `codex-go` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 21 | `""` | `opencode` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 22 | `""` | `pi` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 23 | `""` | `pi-go` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 24 | `""` | `motoko` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 25 | `""` | `eval` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 26 | `""` | `eval-go` | accept (image default in charge) | reject: unknown cloud provider | C2 | S4 |
| 27 | `claude` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 28 | `claude` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 29 | `gemini` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 30 | `gemini` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 31 | `codex` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 32 | `codex` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 33 | `opencode` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 34 | `opencode` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 35 | `pi` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 36 | `pi` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 37 | `motoko` | `eval` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 38 | `motoko` | `eval-go` | accept (eval image has every CLI) | reject: unsupported pair | C3 | S2 |
| 39 | `pi` | `pi-go` | accept (Dockerfile.agent-pi-go bakes in pi) | reject: unsupported pair | C4 | S5 |

**Four causes, each naming its fix:**

- **C1 — `managed_agents` (13 cells).** Dev: binaryless, never refused on image grounds;
  M1 has no such key → "unknown cloud provider". Fix S3.
- **C2 — empty provider `""` (13 cells).** Dev returns nil early on `provider == ""`
  ("the image default is in charge"); M1 has no `""` key. Fix S4.
- **C3 — `eval`/`eval-go` variants (12 cells).** Dev maps them to `nil` = any provider
  (agent-eval bakes in every CLI); M1 has no wildcard concept. Fix S2 (+ policy P2, which
  keeps the agent-level refusal WITHOUT diverging the pair validator).
- **C4 — `pi-go` under `pi` (1 cell).** Dev's Dockerfile-grounded table has it; M1 lacks
  the cell. Fix S5.

**Reconciled semantics (normative — two executors must write the same code from this):**

Pair-level validity — ONE semantics; the coordinator's `ValidateExecutionRoute` accept-set
must EQUAL dev's `checkVariantProviderAgreement` accept-set over all 104 cells:

- **S1** Known variant set = dev's 13 `knownVariants` keys. Unknown variant → typed
  permanent error.
- **S2** `wildcardVariants = {"eval", "eval-go"}`: pair-valid with EVERY provider
  (represented as a named set checked BEFORE per-provider lookup, mirroring dev's `nil`
  value semantics).
- **S3** `binarylessCloudProviders = {"managed_agents"}`: pair-valid with EVERY known
  variant (no binary needed on `PATH`; checked before per-provider lookup, mirroring dev's
  `binarylessProviders` early return).
- **S4** Provider `""`: pair-valid with EVERY known variant (image default in charge;
  mirrors dev's early return).
- **S5** Binaried providers keep per-provider variant sets:
  `claude → {"", default, go}` (empty normalizes to `default` via
  `NormalizeExecutionVariant`), `gemini → {gemini, gemini-go}`,
  `codex → {codex, codex-go}`, `opencode → {opencode}`, `pi → {pi, pi-go}`,
  `motoko → {motoko}`. Unknown provider → typed permanent error.

Agent-route POLICY — layered ABOVE pair validity, applied ONLY at the agent call sites
(`ResolveExecutionRoute` in the daemon, and `validateCoordinatorConfigBytes` at config
publication), NEVER inside the pair validator:

- **P1** Provider must be non-empty (guaranteed by config inheritance, V9; defensively
  refused with a typed error if violated).
- **P2** `executor_variant ∈ {eval, eval-go}` is refused for coordinator agents with a
  typed permanent error naming the policy ("eval job variants are not agent routes") —
  this preserves the recovered design's freeze (eval variants stay outside the AGENT
  matrix) without making the pair validator diverge from dev.
- **P3** The pair must satisfy S1–S5.

`Dispatch()`'s defensive check uses the PAIR level only, so any programmatic eval or
empty-provider dispatch keeps exactly dev's committed behavior (V14).

**Fallout in the committed M1 tests, named and bounded:** M1's off-diagonal rejection
test includes `{"eval","eval"}` and `{"claude","eval-go"}` expectations (V25 patch
content). M1r MOVES those expectations from the pair-validator test to the new policy
tests. No other committed expectation changes; any further test edit is a design
amendment.

**M1r acceptance = real-function parity, not prose.**
`TestRouteMatrixParityWithDispatcherGuard` (new, in `internal/dispatch/cloudrun`, which
already imports `internal/coordinator` — the reverse import would cycle):

- providers := the 8 listed above; variants := the 13 keys read from the REAL
  `knownVariants` map (so a future variant addition automatically widens the sweep);
- for each of the 104 cells, compute `acceptDev := checkVariantProviderAgreement(v,p) == nil`
  and `acceptCoord := coordinator.ValidateExecutionRoute("parity-probe", p, v) == nil`;
- `exemptions := map[cell]string{}` — **EMPTY**. The mechanism exists in the test; every
  entry MUST carry a non-empty justification string; the test FAILS on (a) any
  non-exempted cell where `acceptDev != acceptCoord`, naming the cell, and (b) any
  exempted cell whose justification is empty. An unexplained exemption cannot pass.
- Companion `TestAgentRoutePolicy_RefusesEvalVariants` pins P2 at `ResolveExecutionRoute`,
  so policy strictness cannot silently migrate into the pair validator (which would break
  parity) and cannot silently vanish (which would break the freeze).

**Instrument honesty (carried from the controller's measurement, V30):** the 39-cell table
above derives from a faithful PYTHON REPLICA of the two Go functions, validated against
dev's committed test vectors (`dispatcher_test.go:427-428`: `managed_agents` in the codex
and pi images is NOT refused; the replica agrees). A replica is evidence, not proof — the
parity test above drives the two REAL functions and is the acceptance gate.

### IN — M2a: the selected OpenCode Cloud Run child, and ONLY it

One resolved executable used for BOTH health and execute; health before clone/plugin work;
an internal 15-second health deadline; a missing child as a TYPED, PERMANENT,
PRE-SIDE-EFFECT failure.

**Files:**
- `internal/executor/opencode/opencode.go` (+ existing test files) — single-resolution
  contract (~60 LOC delta)
- `cmd/ailang/coordinator_cloud.go`, `cmd/ailang/coordinator_cloud_executor.go` (+ new
  `coordinator_cloud_preflight_test.go`) — hoisted preflight (~40 LOC delta + tests)

**Design:**

1. `OpenCodeExecutor` gains `resolveExecutable() (string, error)`:
   - empty configured path → typed refusal (the constructor already fails loudly on empty
     config — `TestNewOpenCodeExecutor_EmptyConfigFailsLoudly` exists at base (V24) — the
     resolver re-checks so the invariant is local, not caller-dependent);
   - bare name → `exec.LookPath`; the result MUST be absolute;
   - absolute configured path → `os.Stat`; must be a regular file with ≥1 Unix execute bit;
   - only SUCCESSFUL resolution is cached (a missing binary installed later can recover);
   - `HealthCheck` and `ExecuteStreaming` both call it, and `exec.CommandContext` receives
     ONLY its returned absolute path. The bare configured string never reaches
     `CommandContext` again.
2. `executeCloudTask` (the `execute-job` child): when the route provider is `opencode`,
   run `HealthCheck` under `context.WithTimeout(ctx, 15*time.Second)` BEFORE plugin clone
   (`:247`), workspace clone (`:268`), and AGENTS.md injection (`:289`). On refusal:
   publish the existing failure completion carrying a typed permanent marker
   (`PermanentDispatchError` arrives with M1 — absent at base, V19) and exit non-zero
   having performed ZERO clone/plugin side effects. The 15s constant is production-fixed;
   tests inject a shorter duration through a seam and separately assert the production
   constant equals 15s.
   **NORMATIVE (from quorum O1):** the preflight MUST obtain its executor via
   `executor.GlobalFactory().GetExecutor(provider)` — the IDENTICAL call `runExecutor`
   makes (`coordinator_cloud_executor.go:41`) — and MUST NOT construct one via the
   opencode constructor directly. The factory MEMOIZES per process (V29:
   `internal/executor/factory.go:100-130` — RLock hit returns the existing
   `f.executors[name]`; miss builds once under write lock and stores at `:128`), so within
   one Cloud Run child the preflight executor and the executing executor are the SAME
   instance and the resolution cache is shared by construction. **Stated limit:** the
   factory is process-local; this guarantees nothing across processes or containers. Each
   child process re-resolves once — intended behavior, not a defect.
3. Scope guard: the preflight branch conditions on `provider == "opencode"` only. Codex,
   Pi, Motoko, and Claude adapters get ZERO diffs (their resolution paths differ — Claude
   has NVM candidate discovery, Motoko a split health path — and they do not share the
   measured incident's threat model; each needs its own design + quorum).
4. Layering: M1's route validation should make a wrong image unreachable; M2a still holds
   because it also catches the OTHER cause of the same symptom — an image that should
   contain `opencode` but does not (image drift/build regression) — and pins the
   health-approved binary against late `PATH` substitution. Explicit non-goals: no
   signature/hash attestation, no parent-directory ownership checks, no full stat→execve
   TOCTOU elimination (matches `m-git-binary-resolution-sweep` wording; APIs stay separate).

**M2a refusal branches — enumerated, each with one build-preserving neutering mutation.**
Every mutation keeps all identifiers referenced (`if false && <cond>` or swap-argument
form), so "mutant does not build" can never masquerade as "guard fired". Protocol per
mutation: back up file to a temp dir, apply, assert it still builds, run ONLY the named
test, require nonzero exit, restore, require green.

| # | Refusal branch / guarantee | Named test (exists at base? / new) | Neutering mutation that must turn it red |
|---|---|---|---|
| R1 | Empty configured executable | `TestNewOpenCodeExecutor_EmptyConfigFailsLoudly` (base, V24) + new resolver-level arm | `if false && e.opencodePath == ""` in `resolveExecutable` |
| R2 | Bare name not on `PATH` (missing child) | `TestHealthCheck_MissingBinary` (base, V24), extended to assert the TYPED permanent error | `if false && lookErr != nil` (fall through to raw exec attempt) |
| R3 | Resolved path not a regular file, or no Unix execute bit | new `TestResolveExecutable_RejectsDirAndNonExecutable` (point at a dir; at a `chmod 644` file) | `if false && (!info.Mode().IsRegular() \|\| info.Mode()&0o111 == 0)` |
| R4 | `LookPath` returns a non-absolute path (poisoned relative `PATH` entry, e.g. `.`) | new `TestResolveExecutable_RejectsRelativeLookup` | `if false && !filepath.IsAbs(resolved)` |
| R5 | Health exceeds the internal 15s deadline | new `TestExecuteJobPreflight_HealthDeadline`: deadline-capturing fake asserts ctx deadline ≈ now+15s; channel-blocked fake exits on `ctx.Done()` under an injected millisecond seam | pass parent `ctx` instead of `healthCtx` to `HealthCheck` (both remain in scope; `cancel` still deferred) |
| R6 | Preflight and execution share ONE executor instance — and therefore one resolution cache (quorum O1) | new `TestExecuteJobPreflight_SharesFactoryInstance`: POINTER-IDENTITY — the executor the preflight uses `==` (same pointer) the one `runExecutor` obtains via `GlobalFactory().GetExecutor(provider)` | preflight builds its own instance through the opencode constructor instead of the factory (all imports/calls remain referenced; builds) |
| R7 | Poisoned `PATH` between health and execute cannot select a different file | new `TestExecuteStreaming_UsesHealthApprovedAbsolutePath`: resolve under `PATH=A`, switch to `PATH=B`, prove marker shows absolute `A/opencode` ran (rides on the R6 shared instance) | pass `e.opencodePath` (configured bare name) instead of the resolved absolute path to `CommandContext` |
| R8 | Preflight refusal must precede side effects | new `TestExecuteJobPreflight_FailsBeforeCloneOrPlugin`: fake clone/plugin counters assert 0 on refusal | reorder: move the preflight call after the plugin-clone step (all calls still present; builds) |

A guard is not a gate until something reds when you remove it: each of R1–R8 is only
"done" when its mutation run is recorded red-then-green.

### OUT of scope — stated so a reviewer can hold the line

- **M3 (durable reservation / consumed-lease / terminal-CAS)** and **M4
  (reservation-before-effect, 10s/60s deadlines, winner-only notification)**: excluded.
  They change shared SQLite+Firestore storage transitions, and the sprint-plan-discovered
  fact stands: the repo has NO Firestore emulator CI facility, so their acceptance is
  currently unsatisfiable without new infrastructure. Each needs its own design + fresh
  quorum with emulator PASS evidence (recovery plan gates 5–6).
- **Codex/Pi/Motoko/Claude executable changes**: excluded. The measured incident's threat
  model is the missing OpenCode child; a five-adapter sweep is exactly the overbreadth the
  recovery plan removed. A shared `internal/executor/executable.go` resolver (absent at
  base, V16) is future work justified by its own design.
- Cloud Run images, Terraform, job `maxRetries`, live multivac config edits, message
  transport semantics.

## Acceptance Criteria — lane-annotated, every command measured at pristine base

Platform caveat: every "measured rc" below is **darwin/arm64 only** (go1.26.6, macOS
26.5.2, zsh, outside any sandbox unless noted). windows/ubuntu legs are UNRUN.
**Never gate on `go build ./...`** — it is rc=1 AT BASE (`cmd/wasm` has no native main;
controller-verified this session).

### Executor lane (satisfiable inside a `workspace-write` sandbox — no loopback binds)

`internal/executor/opencode`, `internal/dispatch/cloudrun`, and `cmd/ailang` scoped arms
below avoid `httptest`: zero `httptest` hits in the first two packages' tests, control =
the same grep instrument fires on 6 `cmd/ailang` test files and 4 coordinator test files
(V21, V22).

- [ ] `go build ./internal/coordinator/... && go build ./internal/dispatch/... && go build ./cmd/ailang` — base rc=0/0/0 (V20); must stay rc=0.
- [ ] `go test ./internal/executor/opencode -count=1` — base rc=0, 1.9s (V20); post-M2a must stay rc=0 and, run with `-v`, must contain `--- PASS` lines for R1–R7's named adapter tests.
- [ ] `go test ./internal/dispatch/cloudrun -count=1` — base rc=0 (V20); post-M1 measured rc=0 on the merged tree already (V11); post-M1r `-v` MUST contain `--- PASS: TestRouteMatrixParityWithDispatcherGuard` (all 104 cells, empty exemption set).
- [ ] `go test ./internal/coordinator -run 'TestResolveExecutionRoute|TestDispatchTasksCloud_|TestValidateCloudExecutionRoutes|TestTaskMaxCostForProvider|TestAgentRoutePolicy' -count=1` — base rc=0 with `[no tests to run]` (VACUOUS at base by construction, V23); post-M1/M1r non-vacuity clause: `-v` output must contain ≥7 `--- PASS` lines naming these tests, including `TestAgentRoutePolicy_RefusesEvalVariants`. This arm deliberately avoids the 4 loopback-binding coordinator test files.
- [ ] `go test ./cmd/ailang -run 'TestConfigCAS_|TestExecuteJobPreflight' -count=1` — base rc=0 (5 pre-existing ConfigCAS tests); post-M1 the three new `TestConfigCAS_*` route tests PASS (measured PASS on merged tree, V11); post-M2a the preflight tests (R5, R6, R8) PASS.
- [ ] R1–R8 mutation protocol: each mutation builds, its named test reds, restore greens. Recorded per-branch in the sprint log. Plus one M1r mutation: remove the `pi-go` cell from the reconciled matrix → `TestRouteMatrixParityWithDispatcherGuard` reds naming cell `(pi, pi-go)`.

### Controller gates (run OUTSIDE the sandbox — loopback and full-suite obligations)

- [ ] `go test ./internal/coordinator/...` — base rc=0 (V20; binds loopback in 4 files, V21 — unsatisfiable in the executor sandbox lane, hence controller-owned).
- [ ] `make test`, `make lint` — regression gates (recovery plan gate 3; `make test`/`lint` base rc=0 per sprint-plan §2, inherited — controller re-measures on the resulting tree).
- [ ] `make check-boundaries` — base rc=0 (V20).
- [ ] `go test -race ./internal/coordinator ./internal/executor/opencode ./internal/dispatch/cloudrun` — focused race arm (recovery plan gate 3).
- [ ] Operator preflight: `ailang coordinator config diff` (exists at base, reads live GCS — V28); no live agent's route is rejected by the reconciled matrix + policy P1/P2. A rejection pauses the sprint (design amendment, not a fallback cell).
- [ ] Changelog entry in `changelogs/v0.32-current.md` (`## [Unreleased]`; root CHANGELOG is index-only — V26): route rejection, M1r parity (39 cells reconciled), OpenCode preflight + 15s deadline.

## Conflict Surface

| Surface | What touches it | Collision analysis (measured) |
|---|---|---|
| `internal/dispatch/cloudrun/dispatcher.go` / `_test.go` | dev commit `23066345e` (variant/provider guard) vs M1's `ValidateExecutionRoute` call | Textual: 3-way clean (V4) though `git apply` fails (V5). Semantic: TWO matrices; 39/104 cells disagree (V30, table in M1r). Resolved by M1r: pair-level parity with an EMPTY exemption set enforced by `TestRouteMatrixParityWithDispatcherGuard`; agent-level eval policy pinned by its own test so it cannot leak into the pair validator. |
| `internal/coordinator/daemon_tasks_exec.go` | M1 rewrites the dispatch loop (108 lines); same-day dev commit `e0b12bf5f` touched `backstop_sweep.go`/`stale_task_detector.go` — file-disjoint from M1's set (V6-adjacent; file lists compared) | No overlap in files; merge measured clean. But the coordinator package is under ACTIVE daily change (two commits within 3 days) — executor must re-fetch and re-run the merge check at execution time rather than trusting this doc's SHA. |
| `design_docs/planned/v0_35_0/` | `3500db0a7` bundles the superseded design + refuted sprint plan (V2) | Must land status-annotated as Superseded (see M1 application contract) or dev carries a doc asserting approval that quorum round 2 refused. |
| `internal/executor/opencode/opencode.go` | M2a single-resolution change vs `m-git-binary-resolution-sweep` (planned, on dev) | Deliberately separate APIs; only the security WORDING is shared (no directory-ownership/signature/TOCTOU claims). No shared resolver is created (V16). |
| `internal/executor/factory.go` | M2a's R6 pointer-identity guarantee DEPENDS on factory memoization (V29) but changes nothing in the file | Read-only dependency; if a future change makes `GetExecutor` non-memoizing, `TestExecuteJobPreflight_SharesFactoryInstance` reds — the dependency is pinned, not assumed. |
| `internal/coordinator` store/notification paths | `m-coord-dispatch-integrity` (planned) — inbound dedup | Disjoint: this sprint changes no store transition (M3/M4 excluded). |
| `cmd/ailang/coordinator_cloud.go` | M2a hoists preflight above clone steps | The file is not in M1's set; no in-flight dev commit touches it in `3500db0a7~1..origin/dev` for the relevant region (drift log shows only dispatcher changes under dispatch/, V6). |

## Verification Log (2026-08-31, worktree `.wt-v1-iter309` at `8bb92ae2b`)

V1–V27: first-party, this session (round 1). V28–V30: **controller-measured, this
session, round-2 revision pass** — measured by the controller in this worktree and handed
down with citations; used as instructed, not re-derived.
Shell: zsh; glob-shaped flags quoted throughout; exit codes captured without pipes.
`$WT` = `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter309`.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | Worktree is detached at origin/dev, clean | `git -C $WT rev-parse HEAD origin/dev; git -C $WT status --porcelain` | Both `8bb92ae2bfa729a6889353bf8de88258d865d56e`; porcelain empty |
| V2 | M1 commit = 13 files incl. TWO design markdowns | `git -C $WT show --stat 3500db0a7` and `show --name-only` | 13 files, 1783 insertions(+), 67 deletions(-); includes `design_docs/planned/v0_35_0/m-coordinator-child-env-opencode-retry-storm.md` (599) and `...-sprint-plan.md` (460) |
| V3 | M1 unit is NOT on dev | `git -C $WT ls-tree origin/dev -- internal/coordinator/execution_route.go` (empty) with same-directory control `... -- internal/coordinator/daemon_tasks_exec.go` | Route file: no output, rc=0-empty; control: `100644 blob e664b0c85... daemon_tasks_exec.go` — instrument sees positives in the same path |
| V4 | M1 merges cleanly at HEAD by 3-way | `git -C $WT merge-tree --write-tree --merge-base=3500db0a7~1 HEAD 3500db0a7` | rc=0, merged tree `82b57a7dc79090e35ff166918311de15b28e3ee7` |
| V5 | Plain `git apply` CANNOT apply M1 | `git -C $WT diff 3500db0a7~1 3500db0a7 > /tmp/ail309/m1.patch; git -C $WT apply --check /tmp/ail309/m1.patch` | rc=1: `error: patch failed: internal/dispatch/cloudrun/dispatcher_test.go:2` — cherry-pick (3-way) is the only valid application path |
| V6 | The drift source is a NEW dispatcher guard on dev | `git -C $WT log --oneline 3500db0a7~1..origin/dev -- internal/dispatch/cloudrun/` | Exactly one commit: `23066345e fix(dispatch): refuse a job whose executor binary cannot exist in its image` |
| V7 | Dev already guards variant/provider at dispatch, Dockerfile-grounded, incl. `pi-go`, `eval*`, `managed_agents` | `sed -n '62,110p' $WT/internal/dispatch/cloudrun/dispatcher.go` | `knownVariants` incl. `"pi-go": true`; `providersForVariant` maps `"pi-go": {"pi"}`, `"eval": nil // any`, comments cite `docker/Dockerfile.agent-*`; `binarylessProviders = {"managed_agents": true}` |
| V8 | Daemon provider/variant split live at HEAD | `grep -n "DefaultProvider" $WT/internal/coordinator/daemon_tasks_exec.go`; `grep -n "ExecutorVariant" <same>` | `124: if d.coordConfig != nil && d.coordConfig.DefaultProvider != ""` / `125: provider = d.coordConfig.DefaultProvider`; `224: if agent.ExecutorVariant != ""` / `225: params.ExecutorVariant = agent.ExecutorVariant` |
| V9 | Config load inherits default into `agent.Provider` | `sed -n '570,595p' $WT/internal/coordinator/agent_config.go` | `if agent.Provider == "" { agent.Provider = cfg.DefaultProvider }` |
| V10 | Merged tree (dev+M1) BUILDS | extract `git -C $WT archive 82b57a7dc \| tar -x -C /tmp/ail309/merged`; `go build ./internal/coordinator/... ./internal/dispatch/... ./cmd/ailang` | extract rc=0; build rc=0 |
| V11 | Merged tree TESTS green in the affected packages | in `/tmp/ail309/merged`: `go test ./internal/dispatch/cloudrun -count=1`; `go test ./internal/coordinator/... -count=1`; `go test ./cmd/ailang -run 'TestConfigCAS_' -count=1 -v` | rc=0 (0.47s); rc=0 (20.98s); rc=0 with `--- PASS` for all six `TestConfigCAS_*` incl. the three M1 additions |
| V12 | M1's validator REJECTS cells dev's guard allows | temp probe test in merged tree calling `ValidateExecutionRoute` (added, run, removed) | `("probe","pi","pi-go") -> permanent dispatch error … unsupported provider/executor_variant pair`; `("probe","managed_agents","codex"/"pi") -> … unknown cloud provider` |
| V13 | `managed_agents` is a registered coordinator provider | `grep -rn "managed_agents" $WT/internal/coordinator --include='*.go'` | `provider_executor.go:12: _ "github.com/sunholo-data/ailang/internal/executor/managed_agents"`; `executor_registration_test.go:12` lists it among registered names |
| V14 | Dev tests PIN that managed_agents/eval images are never refused | `sed -n '395,440p' $WT/internal/dispatch/cloudrun/dispatcher_test.go` | Table cases: `{"managed_agents in codex image","codex","managed_agents",false}`, `{"eval image runs anything","eval","opencode",false}` — calling `checkVariantProviderAgreement` directly (so they survive M1 textually; the divergence is semantic, per V12/V30) |
| V15 | OpenCode adapter re-consults PATH at execute; default is bare name | `grep -n "LookPath\|CommandContext\|opencodePath" $WT/internal/executor/opencode/opencode.go`; `sed -n '145,160p'` and `sed -n '563,585p'` | Def: `:48-50` `opencodePath := cfg.OpenCodePath; if == "" { = "opencode" }`; execute `:156 cmd := exec.CommandContext(ctx, opencodePath, args...)`; health `:568 exec.LookPath(opencodePath)` then `:573` runs `--version` — same STRING, not a pinned resolved path |
| V16 | No shared executor resolver exists at base | `ls $WT/internal/executor/executable.go` (rc=1, ENOENT) with control `ls $WT/internal/executor/opencode/opencode.go` (rc=0) | Negative + same-package positive control both quoted |
| V17 | The cloud child performs NO health check | `grep -n "HealthCheck" $WT/cmd/ailang/coordinator_cloud.go coordinator_cloud_executor.go coordinator_cloud_github.go` → rc=1; controls: `grep -n "GetExecutor" coordinator_cloud_executor.go` → `:41` (same file); same grep pattern over all of `cmd/ailang` hits `doctor.go:697 exec.HealthCheck(ctx)` (instrument proves positive on the same pattern, same package) | No HealthCheck in any of the three execute-job files |
| V18 | Clone/plugin spend precedes the first possible binary failure | `grep -n "func \|clone\|plugin\|runExecutor" $WT/cmd/ailang/coordinator_cloud.go` | plugin clone `:247`, workspace clone `:268`, `injectAgentsMD` `:289`, `runExecutor` `:371` — executor obtained only inside `runExecutor` (`coordinator_cloud_executor.go:41`) |
| V19 | `PermanentDispatchError` is absent at base and arrives with M1 | `grep -rn "PermanentDispatchError" $WT/internal $WT/cmd --include='*.go'` → rc=1; control `grep -rn "DispatchParams" $WT/internal/coordinator --include='*.go'` → `cloud_dispatcher.go:16` etc.; `grep -c "PermanentDispatchError" /tmp/ail309/m1.patch` → 18 | Negative with same-scope positive control; the type is introduced by the M1 unit |
| V20 | Pristine-base gate measurements | `go build ./internal/coordinator/...`; `./internal/dispatch/...`; `./cmd/ailang`; `go test ./internal/coordinator/...`; `./internal/dispatch/...`; `./internal/executor/opencode -count=1`; `make check-boundaries` | rc=0, 0, 0, 0, 0, 0 (opencode 1.946s), 0 |
| V21 | Whole-package coordinator tests bind loopback | `grep -rln "httptest" $WT/internal/coordinator --include='*_test.go'` | 4 files: `push_handlers_test.go`, `github_webhook_test.go`, `secret_approval_bridge_test.go`, `daemon_http_test.go` — hence controller-gate, not executor obligation |
| V22 | The executor-lane packages bind NO loopback | same grep over `$WT/internal/executor/opencode` and `$WT/internal/dispatch/cloudrun` → rc=1; controls: `ls` lists their 6 test files; the identical grep instrument fires on 6 `cmd/ailang` test files | Negative + two positive controls (instrument and scope) |
| V23 | Scoped-arm base rcs | `go test ./internal/executor/opencode -run 'TestHealthCheck_MissingBinary\|TestHealthCheck_WithFakeBinary\|TestExecuteStreaming_BinaryNotFound' -count=1 -v`; `go test ./internal/coordinator -run 'TestResolveExecutionRoute' -count=1`; `go test ./cmd/ailang -run 'TestConfigCAS_' -count=1` | rc=0 with 3 `--- PASS`; rc=0 `[no tests to run]` (vacuous at base — AC carries a non-vacuity clause); rc=0 |
| V24 | Base tests M2a builds on EXIST, with read bodies | `grep -hn "^func Test" $WT/internal/executor/opencode/*_test.go` | `TestExecuteStreaming_BinaryNotFound` (:119), `TestHealthCheck_MissingBinary` (:137), `TestHealthCheck_WithFakeBinary` (:146), `TestNewOpenCodeExecutor_EmptyConfigFailsLoudly` (:139 of opencode_test.go) |
| V25 | M1 adds 13 named test functions | `grep "^+func Test" /tmp/ail309/m1.patch` | 13 names incl. `TestDispatchTasksCloud_HistoricalDefaultProviderCannotSplitRoute`, `TestConfigCAS_RejectsProviderVariantMismatchBeforeWriting`, `TestDispatchRejectsProviderVariantMismatchBeforeRunJob` |
| V26 | Changelog target file | `ls $WT/changelogs/ \| grep -i current` | `v0.32-current.md` |
| V27 | Toolchain/platform for every green above | `go version; uname -m; sw_vers -productVersion` | `go1.26.6 darwin/arm64`, `arm64`, macOS `26.5.2` — **windows/ubuntu legs unrun** |
| V28 | `ailang coordinator config diff` exists at base and reads live GCS (rollout mitigation is real) — **controller-measured** | read `$WT/cmd/ailang/coordinator_config.go`: `case "diff":` at `:212`; usage string `usage: ailang coordinator config diff <file>` at `:312`; `gcsConfigStore` at `:101-128` reads `gs://<bucket>/<object>` via `client.Bucket(...).Object(...)`; same-path control `grep -c "func " coordinator_config.go` | Subcommand and GCS-reading store both present; control = 14 functions (instrument sees positives in the same file). Quorum O2's premise REFUTED; the missing log row (the valid residue) is this row |
| V29 | Within one Cloud Run child, preflight and `runExecutor` get the SAME executor instance IFF both use `GlobalFactory().GetExecutor` — **controller-measured** | read `$WT/internal/executor/factory.go:100-130`: RLock fast path returns existing `f.executors[name]`; on miss, builds once under write lock and stores `f.executors[name] = exec` at `:128`. Call chain: `runExecutor` defined `coordinator_cloud_executor.go:26`, sole call at `coordinator_cloud.go:371`, obtains via `executor.GlobalFactory().GetExecutor(provider)` | Factory memoizes per process → shared instance and shared resolution cache within a child, PROVIDED preflight uses the identical factory call (normative M2a constraint + R6 pointer-identity test). LIMIT: process-local; no cross-process/container claim |
| V30 | Full cross product of the two authority tables: 104 cells, 39 disagreements, ALL M1-over-rejects, FOUR causes — **controller-measured** | Python replica of `checkVariantProviderAgreement` (`dispatcher.go:67-112`) and `cloudProviderVariants`+`NormalizeExecutionVariant` (`3500db0a7:execution_route.go:49-83`) enumerating 8 providers × 13 variants; replica validated against dev's committed vectors `dispatcher_test.go:427-428` (managed_agents in codex/pi images NOT refused — replica agrees) | 39/39 over-rejects, 0 under-rejects. Causes: `managed_agents` (13 cells), empty provider `""` (13), `eval`/`eval-go` wildcard (12), `pi-go` (1). INSTRUMENT HONESTY: a replica is evidence, not proof — the M1r acceptance gate drives the two REAL Go functions (`TestRouteMatrixParityWithDispatcherGuard`) |

## Where this session's measurements CONTRADICT or refine inherited claims

1. **"Nothing at HEAD ties variant to provider" (recovered design, V5/V14 of its log) is
   STALE.** `23066345e` landed a Dockerfile-grounded dispatcher guard on dev on 2026-08-31,
   after the recovered design's 2026-08-29 log. The recovered design was RIGHT at its base
   (`45503bac6`) and is WRONG at today's HEAD. Consequence: M1's contribution narrows to
   daemon-side route authority + config-publication rejection + typed errors; its
   dispatcher-level check becomes a second, REDUNDANT-but-divergent matrix — hence M1r.
2. **"The M1 unit applies cleanly" (controller, first-party) is TRUE but
   instrument-dependent.** 3-way merge: clean (V4, independently reproduced). Plain
   `git apply --check`: FAILS (V5). An executor scripting `git apply` would wrongly
   conclude the unit no longer applies.
3. **M1-as-committed would REGRESS 39 of 104 route cells — four causes, not two.**
   Round 1 of THIS doc found `pi/pi-go` + `managed_agents` by probe (V12) and wrote M1r as
   prose bullets covering only those. Quorum round 1 (oc-glm-5-2) objected that the bullets
   were unmeasured; the controller's full enumeration (V30) CONFIRMED AND EXPANDED the
   finding: 39 disagreements — `managed_agents` (13), empty provider (13), `eval`/`eval-go`
   wildcards (12), `pi-go` (1) — every one an over-reject, zero under-rejects. The round-1
   bullets would have left 25 cells still over-rejected: the exact regression the reviewer
   predicted. Zero cherry-pick conflicts and fully green suites (V10, V11) hide ALL of it,
   because each side's tests exercise only its own table. Textual cleanliness and green
   suites are not route-semantics equivalence; M1r's real-function parity test is.
4. **The recovered design understated the child gap.** It said health checks "use LookPath
   but later execute the original bare name". Measured: the execute-job path has NO health
   check at all — nothing to preflight-fail before two clones and an AGENTS.md injection
   (V17, V18). M2a is therefore an ADDITION of a boundary, not a tightening of one.
5. **Quorum O1's premise was half-right in the useful direction.** Separate instances
   would indeed break cache sharing — but the factory memoizes (V29), so the risk is real
   only if preflight bypasses the factory. The fix is a normative constraint plus a
   pointer-identity gate (R6), not a design change.
6. Iteration 305's overbreadth claim about the `.wt-iter302` M2–M4 diff was NOT re-verified
   here (the parked diff was not read; it stays parked per the recovery plan) — recorded as
   still-inherited, not confirmed.

## Non-Goals

Everything under "OUT of scope" above, plus: no retry-accounting change (a permanent child
refusal still flows through today's completion path until M3/M4 land — M2a changes WHERE
and HOW CHEAPLY the child fails and gives the failure a type, not how often the coordinator
redials); no new wire protocols; no edits to live multivac config or historical messages.

## Implementation Plan

### M1 + M1r — apply and reconcile route authority (~0.75 day)

Verbatim executor procedure (the committing is the controller's; the executor prepares the
tree and evidence):

1. `git fetch origin && git rev-parse origin/dev` — if origin/dev has moved past
   `8bb92ae2b`, re-run the V4 merge-tree probe against the NEW head before anything else;
   a non-zero rc there is a pause condition (this doc's merge evidence is SHA-pinned).
2. Apply `3500db0a7` by cherry-pick (3-way). Do NOT use `git apply` (V5).
3. M1r per the normative spec (S1–S5, P1–P3): reshape the pair validator (wildcard
   `eval`/`eval-go` cells, binaryless `managed_agents`, empty-provider rule, `pi-go`
   cell); relocate the eval refusal to agent-route policy P2 at `ResolveExecutionRoute` +
   config validation; move the two committed off-diagonal eval expectations into the
   policy tests; add `TestRouteMatrixParityWithDispatcherGuard` (104 cells, empty
   exemption set) and `TestAgentRoutePolicy_RefusesEvalVariants`.
4. Status-annotate the two bundled markdowns as Superseded (M1 application contract above).
5. Evidence: executor-lane ACs above (incl. the pi-go parity mutation), then hand the
   controller gates over.

### M2a — OpenCode child boundary (~1 day)

1. `resolveExecutable()` + single-resolution rewiring in
   `internal/executor/opencode/opencode.go`; extend the three existing binary/health tests
   (V24) to assert the typed error; add R3/R4/R7 tests.
2. Hoist the opencode-only preflight into `executeCloudTask` ahead of `:247`
   (plugin clone), obtaining the executor ONLY via `GlobalFactory().GetExecutor(provider)`
   (normative, V29), with the 15s internal `context.WithTimeout` and an injectable
   duration seam; add R5/R6/R8 tests with fake clone/plugin counters and the
   pointer-identity assertion.
3. Run the R1–R8 mutation protocol; record red-then-green per branch.
4. Changelog entry in `changelogs/v0.32-current.md` (V26).

Milestone review stop: any new provider/variant cell beyond the M1r spec (S1–S5), any
non-empty parity exemption, or any diff in codex/pi/motoko/claude adapter files, is a
design amendment — pause, do not extend.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| A live cloud agent is configured with an `eval`/`eval-go` variant (the ONLY intentional strictness, now at policy level P2) | Deploy-time permanent rejection of a working lane | Operator preflight AC (`ailang coordinator config diff`, V28) BEFORE deploy; a hit pauses the sprint for a design amendment — no fallback cell may be added ad hoc |
| The two matrices drift again after this sprint | Silent re-divergence, same class as finding #3 | `TestRouteMatrixParityWithDispatcherGuard` fails the build on ANY cell disagreement (empty exemption set; an unexplained exemption fails the test by construction); variants are read from the REAL `knownVariants` map so new variants auto-widen the sweep |
| Policy strictness migrates into the pair validator (or silently vanishes) | Parity breaks, or the recovered design's eval freeze is lost | `TestAgentRoutePolicy_RefusesEvalVariants` pins P2 at `ResolveExecutionRoute`, separately from parity |
| A future factory change drops memoization | R6's shared-cache guarantee silently voids | The pointer-identity test reds — the dependency on `factory.go` memoization (V29) is pinned by a test, not assumed |
| `internal/coordinator` moves under the sprint (two commits in 3 days measured) | Cherry-pick conflicts or stale evidence | Step 1 of the M1 procedure re-probes merge-tree at execution time; conflicts route back to the controller, never resolved ad hoc by the executor |
| 15s health deadline too short under cold-start CPU contention in Cloud Run | False-permanent refusals | The deadline bounds a local `--version` exec, not a network call; the constant is a single named const — raising it is a one-line reviewed change, and the deadline-capturing test pins whatever value is chosen |
| Preflight-refusal path emits a completion the coordinator retries anyway | Operator sees repeated (bounded-by-today's-behavior) failures | Documented explicitly as expected until M3/M4: M2a makes each failure cheap and typed; it does not change redial accounting. The typed marker is forward-compatible with M4's classifier |
| Sandbox executor cannot run whole-package coordinator tests | AC falsely reported green on a partial run | Lane split is explicit: the executor arm is `-run`-scoped with a non-vacuity clause; whole-package runs are controller gates by name |

## Rollout and Rollback

1. Land M1+M1r and M2a as ordinary dev commits behind the controller's gates. No config
   write, no Terraform, no image change is part of this sprint.
2. Operator preflight (config diff vs reconciled matrix + policy, V28) before the next
   coordinator deploy; the current live planner route is `codex/codex` (recovered design
   V8, inherited), so validation is expected to be a no-op.
3. Canary (operator-authorized, separate from this sprint's mergeable artifact): one
   low-cost `codex/codex` no-op task; verify the single route log line
   (agent/provider/variant/model together) and, for an opencode-routed task, the preflight
   log line appearing BEFORE any clone output.
4. Rollback = revert the commits; no schema, storage, or config migration exists in this
   scope (M3's fields were excluded precisely so rollback stays a plain revert).

## Related Documents

- [m-coordinator-child-env-opencode-retry-storm-recovery-plan.md](m-coordinator-child-env-opencode-retry-storm-recovery-plan.md) — the approved narrowing this doc executes (M1/M2a slice)
- Recovered design + sprint plan (land with `3500db0a7`, status-annotated Superseded) — incident forensics V1–V21 remain the evidentiary record for the incident itself
- [m-git-binary-resolution-sweep.md](m-git-binary-resolution-sweep.md) — git-specific resolver; shared wording, separate APIs
- [m-coord-dispatch-integrity.md](m-coord-dispatch-integrity.md) — inbound dedup; no shared state with this sprint
- `23066345e` — the on-dev dispatcher guard this design must stay cell-for-cell consistent with (M1r parity gate)

---

**Document created**: 2026-08-31 (mission iteration 309)
**Last updated**: 2026-08-31 (revision 2 — post-quorum round 1, objections O1/O2/O3 dispositioned)
