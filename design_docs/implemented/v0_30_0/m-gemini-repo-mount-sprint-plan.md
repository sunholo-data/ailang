# M-GEMINI-REPO-MOUNT Phase 2 — Sprint Plan (Clone-over-egress capability)

**Design doc**: [`design_docs/planned/v0_30_0/m-gemini-repo-mount.md`](m-gemini-repo-mount.md) — **Phase 2 — Clone-over-egress capability** section ONLY (lines 269–556). The SUPERSEDED `repository`/`inline` mount design and the Phase-1/1b spike records are historical and are **NOT** in scope.
**Target**: v0.30.0
**Status**: PLAN-READY (design doc greenlit by Mark #399 2026-07-18: "clone over egress approved" + "apply both fixes, ship it")
**Sprint ID**: M-GEMINI-REPO-MOUNT
**JSON progress**: [`.ailang/state/sprints/sprint_M-GEMINI-REPO-MOUNT.json`](../../../.ailang/state/sprints/sprint_M-GEMINI-REPO-MOUNT.json)
**Planner model**: claude-opus-4-8
**Risk level**: Low–Medium

---

## Goal

Give the `managed_agents` (Gemini/Vertex) executor an **opt-in egress-enabled sandbox** so the hosted agent can `git clone` the public AILANG repo at a target revision, run review / `ailang check` in-sandbox, and return a structured verdict — upgrading Gemini from reasoning-only review (prompt-packed diff) to in-sandbox verification. This is the LIVE-VERIFIED option (d) "clone-over-egress" (probes Q/R, HEAD `806b3b4a4`). **No mount, no GCS, no inline, no encoder.**

## Lane & frozen-core boundary (honored)

Executor-contract widening + executor/eval-harness plumbing in the **extension lane**. NO AILANG language surface, NO parser/lexer/AST/type-system/eval/VM change, NO motoko-core change. The only wire change is one JSON request field; the only Go-contract change is one typed `Task` field + one capability constant + one shared validation helper + two CLI flags + two eval-option fields.

## Execution environment

Isolated git worktree branched from `origin/dev` (NEVER the shared main working tree). Opus sprint-executor. **Each milestone is one commit.** PR into `dev`; CI + docs build must be green.

## Velocity basis

~150 LOC/day recent baseline (matches doc's ≤150 LOC production-Go budget). Total production Go ≈145 LOC across 4 milestones; tests excluded. Estimated 1–2 focused days.

---

## Anchor verification (done at plan time, working tree, 2026-07-18)

All design-doc anchors re-confirmed against HEAD before planning — the executor will re-check line numbers on entry (files drift):

- `executor.go`: `Task` struct present (`Metadata`, `ExtraEnv` fields), `Capability` type + constant block, `CapRemoteSandbox` present, `Capabilities()` interface method present.
- `managed_agents.go`: `envRaw := json.RawMessage(`{"type":"remote"}`)` hardcode present; `context.WithTimeout(ctx, timeout)` → `sendInteraction(reqCtx, …)` propagation present.
- `managed_agents/` reads **NO** `Metadata` keys (grep empty) — negative claim holds; the typed field replaces the planned key.
- `exec.go`: `resolveAgenticExecutorName`, `executeCLI`, `--api-only` flag present.
- `gemini_evaluator_bridge.go`: `BuildDiffBundle`, `EvalOptions`, `RunGeminiEvaluator`, `EvalRunner(ctx …)`, `DefaultGeminiRunner`, `VerificationDegraded`/`DegradedReason`, and the `VerificationDegraded ⇒ DegradedReason non-empty` invariant present.

**IMPORTANT**: all line:number references in the design doc are advisory. Locate symbols by name, not line number.

---

## Milestones (each ≤1 day, each = one commit)

### M1 — Typed egress capability gate + env wiring

**Deliverable**: `Task.RequiresEgress` + `CapNetworkEgress` + shared `ValidateTaskCapabilities` (executor.go); `buildEnvironment` replaces the `managed_agents.go` `envRaw` hardcode; golden tests assert BOTH JSON shapes + loud non-capability rejection. **No live call.**

**Production changes** (~45 LOC):
1. `internal/executor/executor.go`:
   - Add `RequiresEgress bool` field to `Task` (zero value `false` = today's behavior; doc-comment it).
   - Add `CapNetworkEgress Capability = "network_egress"` to the capability constant block (alongside `CapRemoteSandbox`).
   - Add shared helper `ValidateTaskCapabilities(task *Task, exec Executor) error` returning a loud error when `task.RequiresEgress == true` and `exec.Capabilities()` does not contain `CapNetworkEgress`. ONE grep-able enforcement point.
2. `internal/executor/managed_agents/managed_agents.go`:
   - Replace the `envRaw := json.RawMessage(`{"type":"remote"}`)` hardcode with `buildEnvironment(task) (json.RawMessage, error)`:
     - `RequiresEgress == false` → **byte-identical** `{"type":"remote"}`.
     - `RequiresEgress == true` → `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}` (probe Q/R shape).
   - Advertise `CapNetworkEgress` in `Capabilities()`. Package continues to read NO Metadata keys.

**Tests** (`managed_agents_test.go` + an `executor` package test):
- Golden: `RequiresEgress == false` → byte-identical `{"type":"remote"}`.
- Golden: `RequiresEgress == true` → exactly `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}`.
- `ValidateTaskCapabilities` with a fake non-capability executor + `RequiresEgress == true` → loud error; assert **no** `Execute`/`ExecuteStreaming` is called.
- `ValidateTaskCapabilities` with `RequiresEgress == false` on any executor → no-op (nil error).

**Acceptance**: doc criteria "Default unchanged", "Egress shape pinned", "RequiresEgress set on non-CapNetworkEgress executor → loud pre-dispatch error".
**Dependencies**: none.
**Risks**: widening the shared `Task` contract touches all executors' `Capabilities()` surface — mitigated: additive; only managed_agents ADVERTISES the cap; other executors' `Capabilities()` are READ, never edited.

---

### M2 — CLI flags

**Deliverable**: `--clone-repo`/`--clone-sha` in `cmd/ailang/exec.go` + `task.RequiresEgress` wiring + loud non-managed_agents / `--api-only` rejection + help text + parsing tests.

**Production changes** (~40 LOC):
- Register `--clone-repo <url>` and `--clone-sha <sha>` in the `runExec` flag block.
- `--clone-repo` set → validate resolved executor is `managed_agents` (via `resolveAgenticExecutorName`); otherwise, OR with `--api-only`, **exit non-zero with a clear error** — never ignore. Fast CLI UX check layered over M1's shared gate.
- `--clone-sha` without `--clone-repo` → error.
- On success: set `task.RequiresEgress = true` on the `executor.Task` built in `executeCLI`, call the shared `ValidateTaskCapabilities` before dispatch, and prepend the canonical clone preamble (see M3 / doc "Canonical clone preamble") to the directive.
- Update `ailang exec` help text.

**Tests** (exec parsing tests):
- `ailang exec claude --clone-repo …` (any non-managed_agents resolution) → non-zero exit + clear error.
- `ailang exec gemini --api-only --clone-repo …` → non-zero exit + clear error.
- `--clone-sha` without `--clone-repo` → error.
- `ailang exec gemini --clone-repo … --clone-sha …` → parses, sets `RequiresEgress = true`, builds the preamble.

**Acceptance**: doc criteria for the two non-zero-exit cases + the `--clone-sha`-without-`--clone-repo` error.
**Dependencies**: M1 (needs `RequiresEgress` + `ValidateTaskCapabilities`).
**Risks**: flag-registration collision — low; grep the flagset first.

---

### M3 — Eval-harness clone-review

**Deliverable**: `EvalOptions.CloneRepoURL/CloneSHA` in `gemini_evaluator_bridge.go` → clone directive + HEAD/pinned evidence check + unchanged-fallback regression tests + timeout/cancellation tests.

**Production changes** (~55 LOC — the doc's "if this grows past ~70, cut here, not elsewhere" caps this milestone):
- Extend `EvalOptions` with optional `CloneRepoURL` + `CloneSHA`.
- In `RunGeminiEvaluator`: when set, build the **clone-review directive** instead of packing the full diff; when unset, `BuildDiffBundle` path is **unchanged**.
- Canonical clone preamble (doc "Canonical clone preamble"):
  - **HEAD review** (`CloneSHA` empty): `git clone --depth 1 <public-url>` (probe-R recipe); echo `git rev-parse HEAD`. Evidence = a syntactically-valid, non-empty 40-hex echo; the bridge **records** it as the reviewed revision.
  - **Arbitrary SHA** (`CloneSHA` set): **shallow fetch-by-SHA** — `git init && git remote add origin <url> && git fetch --depth 1 origin <sha> && git checkout --detach FETCH_HEAD`; echo `git rev-parse HEAD`. Evidence = echo must **equal** `CloneSHA`. (NOT a full clone — iteration-52 re-quorum fix; both modes shallow/bounded.)
  - Directive then runs review / `ailang check` (agent may fetch a pinned Linux `ailang` release binary over the same egress) and emits the verdict JSON the bridge already parses.
- Evidence check is **conditional**: pinned ⇒ echo must equal `CloneSHA` (mismatch ⇒ degraded); HEAD ⇒ echo must be valid non-empty 40-hex (recorded as reviewed revision; missing/invalid ⇒ degraded). A valid HEAD echo is **NOT** degraded.
- Deadline-exceeded from the runner → `VerificationDegraded: true` + non-empty `DegradedReason` (reuse the existing invariant). Never a retry, never a clean pass.
- Verdict return rides the EXISTING text-output parsing (`managed_agents_bridge.go` unchanged). Caller ctx threading unchanged.

**Tests** (bridge tests with a fake/injected `EvalRunner`):
- Clone options unset → `BuildDiffBundle` fallback path unchanged (regression test).
- `CloneSHA` set + mismatched `rev-parse HEAD` echo → `VerificationDegraded == true`, non-empty `DegradedReason`.
- HEAD review (`CloneSHA` empty) + valid non-empty 40-hex echo → `VerificationDegraded == false`, echoed SHA recorded as reviewed revision (**positive** test — HEAD reviews must pass cleanly).
- Either mode + missing/empty/invalid echo → `VerificationDegraded == true`, non-empty `DegradedReason`.
- Fake runner returning a `context.DeadlineExceeded`-class error → `VerificationDegraded == true`, non-empty `DegradedReason` (never a retry/clean pass).
- Caller-ctx-honored assertion: the env-builder + clone-review path creates no fresh background context (caller ctx reaches `sendInteraction` via existing `WithTimeout` propagation + eval-bridge threading).

**Acceptance**: doc criteria for fallback-unchanged, pinned-mismatch degraded, HEAD-valid-pass, missing/invalid degraded, timeout-structured-degraded, caller-ctx-honored.
**Dependencies**: M1 (RequiresEgress). Independent of M2 but shares the clone-preamble text — keep the preamble in one place (a small shared builder) so M2's CLI preamble and M3's bridge preamble don't drift.
**Risks**: preamble duplication between exec.go and the bridge — mitigate with a single preamble builder helper.

---

### M4 — Live E2E + docs

**Deliverable**: `AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`-gated (CI-skipped) end-to-end clone→check→verdict run; `ailang exec` help + docs updated; evidence recorded in the design doc.

**Production changes** (~5 LOC + docs):
- Extend the EXISTING `managed_agents_live_test.go` probe harness (probes A–R) with a gated E2E: env-enabled sandbox clones the public repo (exercise the **fetch-by-SHA** path to confirm the provider supports `git fetch --depth 1 <sha>`), runs the directive, returns a parsed verdict. Missing ADC ⇒ **SKIP** (never a pass). Stays manual-only, out of default CI.
- Update `ailang exec` help + docs for the new flags (`docs/` guide + CLI help, single source of truth).
- Record the E2E evidence (SHA cloned, verdict) in `design_docs/planned/v0_30_0/m-gemini-repo-mount.md`.

**Acceptance**: doc criteria "Live-gated E2E (skipped in default CI; missing ADC is a SKIP)" + "All tests passing; `ailang exec` help + docs updated".
**Dependencies**: M1, M2, M3.
**Risks**: E2E requires live ADC + real Vertex spend; if ADC unavailable in the executor's environment, the live run SKIPs — the executor records that the gate was exercised as a SKIP and hands the actual live confirmation to Mark (do not fabricate a pass). Fetch-by-SHA provider support is the one premise M4 confirms live; if the provider rejects `fetch --depth 1 <sha>`, surface it loudly as a design-doc follow-up (do NOT fall back to a full clone silently).

---

## No-silent-fallback compliance (design invariants the executor must preserve)

- `RequiresEgress == false` default request stays **byte-identical** to today.
- Typed gate: `RequiresEgress` on a non-`CapNetworkEgress` executor → loud shared pre-dispatch error before any network I/O (programmatic AND CLI callers).
- Clone flags on non-managed_agents or with `--api-only` → loud CLI error, never ignored.
- Clone-review with no valid clone evidence → `VerificationDegraded` with reason — never silently downgraded to prompt-packed, never a clean pass on absent evidence.
- Deadline expiry → `VerificationDegraded` with reason — never a silent retry, never a clean pass.

## Callers that MUST continue to work unchanged (regression guard)

`ailang exec gemini "<directive>"` (all current flag combos, no clone flags) → byte-identical requests; coordinator/factory callers building `executor.Task` without `RequiresEgress` (zero-value no-op); `RunGeminiEvaluator` with default `EvalOptions` (diff-bundle path); injected-runner test seams; `managed_agents_bridge` extract-out behavior; all Claude/OpenAI/Anthropic/OpenRouter/Ollama `ailang exec` paths.

## LOC budget

**≤150 LOC production Go** (tests excluded): executor.go field+constant+helper+call-sites ~15–20; managed_agents env builder + cap ~30; exec.go flags/validation/preamble ~40; eval-bridge options/directive/evidence ~55 (≈145 total). If eval-bridge share exceeds ~70, cut scope in the directive templating (M3), not elsewhere.

## Files touched (Conflict Surface)

- `internal/executor/executor.go` — `Task.RequiresEgress`, `CapNetworkEgress`, `ValidateTaskCapabilities`.
- `internal/executor/managed_agents/managed_agents.go` — `envRaw` → `buildEnvironment`; `Capabilities()` gains `CapNetworkEgress`.
- `internal/executor/managed_agents/types.go` — **unchanged** (`Environment` already `json.RawMessage`).
- `internal/executor/managed_agents/managed_agents_test.go` — golden/rejection tests (new).
- `internal/executor/managed_agents/managed_agents_live_test.go` — extends probes A–R with gated E2E (manual-only).
- `cmd/ailang/exec.go` — flag block, `executeCLI` task literal (sets `RequiresEgress`, calls shared validation), help.
- `internal/eval_harness/gemini_evaluator_bridge.go` — `EvalOptions`, `RunGeminiEvaluator`, degraded-invariant reuse, ctx threading unchanged.
- `internal/eval_harness/managed_agents_bridge.go` — **unchanged** (reused as-is for verdict return).

**Explicitly NOT touched**: every non-managed_agents executor's own code; parser/lexer/AST/type-system/eval/VM; the motoko core. Frozen-core boundary holds.

## Open questions / blocks surfaced

1. **M4 live E2E needs ADC + Vertex spend + fetch-by-SHA provider support** — the one genuinely live-gated piece. If ADC is absent in the executor's worktree environment, M4's E2E SKIPs and the real live confirmation is a hand-off to Mark. The executor must NOT mark M4 "passing" on a SKIP alone; it should report M4 as "code + docs done; live run SKIPPED (no ADC) / CONFIRMED (evidence recorded)".
2. **fetch-by-SHA is INCORPORATED, not VERIFIED-LIVE** — the arbitrary-SHA `git fetch --depth 1 origin <sha>` path was adopted from the iteration-52 re-quorum (gemini-3-1-pro) but only `--depth 1` HEAD clone was probe-proven (Probe R). M4's E2E is where this premise gets its live confirmation. Low risk (shallow-by-construction), but it is the single unverified premise.
3. **Preamble single-source** — M2 (CLI) and M3 (bridge) both build the clone preamble; keep it in one helper so they can't drift (planner note, not a doc requirement).

## Success metrics

- All new unit/golden tests pass; `make test` green; `make lint` clean.
- Default (`RequiresEgress == false`) request byte-identical to pre-sprint.
- Every design-doc acceptance-criteria checkbox has a corresponding passing test (except the live-gated E2E, which is SKIP-or-live-confirm).
- `ailang exec` help + docs updated for `--clone-repo`/`--clone-sha`.
- Frozen-core boundary preserved (no parser/type/motoko change).

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-gemini-repo-mount-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-GEMINI-REPO-MOUNT.json`
