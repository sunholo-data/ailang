# M-GEMINI-EVALUATOR-DIFF-BRIDGE: inject a sprint diff bundle into a sandboxed gemini evaluator

**Status**: Planned
**Target**: v0.30.x — mission infrastructure
**Priority**: P1 (model-diversity capability for the mission evaluator; unblocks generator≠judge)
**Estimated**: ~2 days (M1 bundle builder ~0.75d; M2 directive + verdict parse ~0.75d; M3 caller seam + degradation ~0.5d)
**Dependencies**:
- `internal/executor/managed_agents/managed_agents.go` (the sandboxed executor; `CapRemoteSandbox`, `Input=[Directive]` + `SystemInstruction`, no repo upload — verified at HEAD, lines 164–176)
- `internal/eval_harness/managed_agents_bridge.go` (the EXTRACT-OUT precedent this doc mirrors, INVERSE direction)
- `cmd/ailang/exec.go` (`ailang exec gemini` → `resolveAgenticExecutorName("gemini")=="managed_agents"`, verified line 317–322; `--json` output shape verified line 281–294)
- `internal/mission/quorum/*` (the frozen Go verdict contract `quorum.ReviewResult` and its `AgenticRunner` seam — for boundary reference, NOT reuse; see Conflict Surface)
**Author**: Designer role, mission-control fleet item c1 (Mark directive on issue #399, 2026-07-17)

**Requested by**: Mark on bookkeeping issue #399 — "once we have gemini via managed agents and openai we can use one of those instead for evaluator? so default can be gemini (if able to git clone the codebase etc)? otherwise sonnet-5." This iteration confirmed the Vertex Managed Agents backend now returns reliably (4/4 bounded `ailang exec gemini` probes, 8–11s, ~$0.01/call) — the operational blocker from iterations 36/37/38 has cleared.

---

## Problem statement

The mission wants the Google/gemini provider (via Vertex Managed Agents) usable as a sprint **EVALUATOR** — model diversity so the generator (Opus/codex) is not also the judge. But the managed_agents executor runs the agent in a **Google-hosted server-side sandbox** (`executor.CapRemoteSandbox`). Verified at HEAD (`managed_agents.go:164–176`), the interaction request body carries ONLY:

```go
Input: []inputBlock{{Type: "user_input", Content: []contentBlock{{Type: "text", Text: task.Directive}}}},
SystemInstruction: task.SystemPrompt,
```

There is **no repo/file upload**. So a gemini evaluator sees **no local repo**: it cannot inspect a sprint's UNCOMMITTED worktree changes, nor re-run local tests. This is exactly Mark's "if able to git clone the codebase" gap — and note that even a `git clone` of public `origin/dev` would NOT contain the sprint's local, uncommitted diff (a sprint is evaluated in a worktree before its branch is pushed).

**The precedent to mirror (INVERSE direction).** `internal/eval_harness/managed_agents_bridge.go` already bridges the sandbox for the EXECUTOR use case in the **extract-out** direction: it appends `managedAgentsBridgeInstruction` to the system prompt so the agent dumps its solution as a fenced code block, then `extractCode`/`writeSolutionFromResponse` parse it back out. It deliberately lives in the harness (not the executor) because it is caller POLICY — "the executor stays policy-free" (file header, lines 8–11; echoed in `managed_agents.go:163`).

This doc builds the **INVERSE: INJECT-IN**. Given a sprint worktree, produce a size-bounded "diff bundle" and inject it INTO the evaluator directive text, so the sandboxed gemini agent can read and reason about the changes. The agent is **reasoning-only** (no local test re-runs) and must emit a **structured verdict** a parser can extract, mirroring how the extract-out bridge parses a fenced block.

### What is NOT the problem (premises re-checked against the repo)

- **The verdict is not already a frozen Go struct for sprint evaluation.** Re-checking the brief's design-question 2: the only frozen Go verdict contract is `quorum.ReviewResult` (`internal/mission/quorum/reviewer.go:39–61`) — verdict ∈ {pass,reject} + `strongest_objection` + `catch` + optional `proposed_fix`. That is a **design-doc review** verdict, a different domain from **sprint evaluation** (score 0–100, pass≥70, per-criterion + blockers — the sprint-evaluator skill's `evaluation_report_schema.md`). The brief's phrase "frozen verdict JSON via the coordinator executor layer" refers to `quorum`'s design-review contract, which we must **not overload**. See design-question 2 resolution and the Conflict Surface for how we avoid competing with it.
- **`ailang exec gemini` already routes to managed_agents.** Verified `exec.go:317` — no new routing needed; the seam exists. `--json` already emits `{success, output, error, input_tokens, output_tokens, cost_usd, duration_ms, num_turns}` (line 281–294) — the bundle-in path uses the existing `Directive`/`SystemPrompt` fields, nothing new in the executor.

---

## Goals

**Primary goal**: Ship a reasoning-only gemini sprint-evaluation CAPABILITY — a policy-layer bridge that packages a sprint worktree's uncommitted diff into a bounded bundle, injects it into a managed_agents directive, and parses a structured verdict back out — WITHOUT touching the executor's policy-free contract or the frozen `quorum.ReviewResult` design-review contract.

**Success metrics**:
1. Given a worktree with N changed files, `BuildDiffBundle` returns deterministic text: `git diff` of uncommitted changes + full contents of changed files, truncated with a LOUD marker over a byte ceiling.
2. Truncation, binary/generated-file drop, and backend error each set an explicit `VerificationDegraded` marker on the verdict — a partial/lenient gemini verdict can never masquerade as a full pass.
3. A structured verdict (`score`, `pass`, `blockers[]`) is parseable from the agent's fenced-block response, with a hard error (never a silent coerced pass) on malformed output — mirroring `ParseReviewResult`.
4. Every element independently covered by `go test ./internal/eval_harness/...`; each test fails on a base binary (feature absent) and passes after.

---

## Non-Goals (explicit scope boundary — reasoning-only)

- **OUT: local test re-runs by gemini.** The sandbox cannot see the repo; the directive states the agent is reasoning-only over the supplied bundle. It reasons about the diff, it does not execute it.
- **OUT: multi-turn sandbox reuse** (the `managed_agents.go:154` M2.5 follow-up). One bounded interaction per evaluation.
- **OUT: making gemini the DEFAULT evaluator.** This ships the capability; the mission default stays sonnet-5. Flipping the default is a separate routing-policy change gated on evidence (see `m-mission-agentic-provider-routing.md`), not this doc.
- **OUT: GCS upload of the repo / real file bridging.** Bundle-in-directive only, byte-bounded. A future doc may add GCS if the ceiling proves too tight.
- **OUT: reusing/overloading `quorum.ReviewResult`.** The sprint verdict is a distinct shape; extending the frozen design-review contract is out of scope (see Conflict Surface).

---

## Solution design

### Overview

A new policy-layer file `internal/eval_harness/gemini_evaluator_bridge.go` (mirroring `managed_agents_bridge.go`'s location and rationale) with three pure, independently-testable pieces:

1. **`BuildDiffBundle(worktree string, opts BundleOptions) (Bundle, error)`** — walks `git diff` + changed-file contents, applies the size/truncation policy, returns a `Bundle{Text string, Truncated bool, DroppedFiles []string, Bytes int}`.
2. **`BuildEvaluatorDirective(designDoc, sprintPlan, acceptanceCriteria string, bundle Bundle) string`** — composes the reasoning-only evaluator directive: the bundle + the design/plan/criteria + a LOUD reasoning-only instruction + a fenced-verdict-schema instruction (mirrors `managedAgentsBridgeInstruction`).
3. **`ParseGeminiVerdict(response string, degraded DegradationInfo) (GeminiVerdict, error)`** — extracts the fenced JSON verdict (reusing the fence logic from the sibling bridge), validates it, and stamps the `VerificationDegraded` marker from `degraded` (bundle truncation / drop / backend error). Malformed → hard error, never a coerced pass.

The **caller** (a thin `mission-evaluator` helper, see design-question 1) wires these onto `ailang exec gemini --json` (or the executor factory directly), reading the worktree, invoking the executor with the composed directive, and parsing the verdict.

### The new verdict type (distinct from quorum.ReviewResult, mirrors the sprint-evaluator skill schema)

```go
// GeminiVerdict is the reasoning-only sprint-evaluation verdict returned by a
// sandboxed gemini evaluator. It intentionally mirrors the sprint-evaluator
// skill's evaluation_report_schema.md (score/result/blockers), NOT
// quorum.ReviewResult (pass/reject/strongest_objection) — those are different
// domains (sprint eval vs design-doc review) and must not be conflated.
type GeminiVerdict struct {
    Score    int      `json:"score"`               // 0..100, pass threshold 70 (sprint-evaluator convention)
    Pass     bool     `json:"pass"`                // score >= 70 AND no blockers
    Blockers []string `json:"blockers"`            // hard-fail reasons; non-empty ⇒ Pass=false regardless of score

    // VerificationDegraded is set (with a non-empty reason) whenever the
    // bundle was truncated / files were dropped / the backend errored, so a
    // partial or lenient verdict can NEVER masquerade as a full pass.
    // (Carried watch-item from fleet item (c); CLAUDE.md "no silent fallbacks".)
    VerificationDegraded bool   `json:"verification_degraded"`
    DegradedReason       string `json:"degraded_reason,omitempty"`
}
```

`ValidateGeminiVerdict` enforces: `Score` ∈ [0,100]; if `len(Blockers)>0` then `Pass==false`; if `VerificationDegraded` then `DegradedReason` non-empty. Any violation is a hard error (Principle 2). **A degraded verdict may still carry a Pass value from the model, but the caller MUST treat `VerificationDegraded==true` as "not a full pass" for routing decisions** — the marker is the guard, exactly as the fleet (c) watch-item requires.

### Size / truncation policy (design-question 3)

- **Byte ceiling**: `BundleOptions.MaxBytes`, default **256 KiB** of bundle text (well under managed_agents token limits; a real gemini context is ~1M tokens but the directive competes with the design doc + plan, so we stay conservative and configurable).
- **Drop order (largest-blast-radius first)**: (1) binary files (detected via NUL byte in first 8 KiB) — never included, only listed; (2) generated/vendored files (path globs: `*.pb.go`, `dist/`, `vendor/`, `*.min.js`, `*_generated.go`, `go.sum`); (3) then, if still over ceiling, the largest remaining changed files by byte size, dropped whole (never mid-file) until under ceiling.
- **LOUD truncation marker**: every drop appends a line to the bundle text AND to `Bundle.DroppedFiles`:
  ```
  === BUNDLE TRUNCATED: dropped internal/foo/big_generated.go (412 KiB, generated) — NOT shown to evaluator ===
  ```
  No silent drop. `Bundle.Truncated=true` propagates to `DegradationInfo` → `GeminiVerdict.VerificationDegraded`.
- **Untracked (new) files are INCLUDED (quorum catch, gemini-3-1-pro 2026-07-16).** `git diff` and `git diff --cached` do NOT show untracked files, and a sprint frequently CREATES new files (this very sprint adds two new bridge files). Omitting them would make the evaluator blind to the sprint's most important additions. The bundle therefore captures the complete changed-file set via `git -C <wt> status --porcelain -z` (which enumerates modified, staged, AND untracked `??` entries), and for untracked files it synthesizes a diff-equivalent view (a `+++ NEW FILE: <path>` header followed by the full file contents), then de-dupes against the tracked-file appendices. Acceptance test `TestBuildDiffBundle_IncludesUntrackedNewFiles` guards this.
- The unified diff of tracked changes (`git -C <wt> diff` + `git -C <wt> diff --cached`) plus the untracked-file headers are ALWAYS included first and are never dropped; only full-file-content appendices are subject to the ceiling (the change signal is load-bearing; whole-file context is the enrichment). A new/untracked file's own body IS subject to the ceiling (it can be arbitrarily large), but its `+++ NEW FILE` header line is retained even when its body is dropped, so a dropped new file is never silently invisible.

### Reasoning-only directive (design-question 4, mirrors managedAgentsBridgeInstruction)

The directive appends a constant instruction (compile-time const, like `managedAgentsBridgeInstruction`):

```
IMPORTANT — Reasoning-only sprint evaluation. You are running in a sandbox with NO access
to the repository or local tests. Everything you can inspect is in the DIFF BUNDLE below.
Do NOT claim you ran any test. Judge ONLY from the supplied diff + file contents against the
design doc's acceptance criteria. At the very END of your response, output your verdict as a
single fenced ```json block with fields: score (0-100), pass (bool), blockers (string[]).
If the bundle is marked TRUNCATED, note in blockers which unseen files limit your confidence.
```

### Caller seam (design-question 1)

A thin `RunGeminiEvaluator(ctx, worktree, designDoc, sprintPlan string, opts EvalOptions) (*GeminiVerdict, error)` helper in `eval_harness` (policy layer — mirrors `RunAgenticReviewer`'s boundary in `quorum`), which:
1. Calls `BuildDiffBundle` → gets `Bundle` + `DegradationInfo`.
2. Calls `BuildEvaluatorDirective` → directive text.
3. Invokes the managed_agents executor via an injected runner function (default impl shells `ailang exec gemini --json --workspace <tmp>`; unit tests inject a stub returning canned output — no live Vertex call in tests). On executor error, `DegradationInfo.BackendError` is set → verdict is `VerificationDegraded` with the error, NEVER a fabricated pass.
4. Calls `ParseGeminiVerdict` → validated verdict, degradation stamped.

**Consumed by**: the mission-control Gate-4 evaluator lane, when routing chooses gemini (per `m-mission-agentic-provider-routing.md`). That routing wiring is out of scope here; this doc ships the callable capability + its Go tests. Executor stays policy-free; the frozen `quorum.ReviewResult` is untouched.

### Files to create / modify

| File | Change | LOC (est) |
|------|--------|-----------|
| `internal/eval_harness/gemini_evaluator_bridge.go` | NEW — `BuildDiffBundle`, `BuildEvaluatorDirective`, `ParseGeminiVerdict`, `ValidateGeminiVerdict`, `GeminiVerdict`, `Bundle`, `BundleOptions`, `DegradationInfo`, `EvalOptions`, `RunGeminiEvaluator` | ~320 |
| `internal/eval_harness/gemini_evaluator_bridge_test.go` | NEW — table tests for bundle build/truncation/drop-order, directive composition, verdict parse/validate, degradation stamping, RunGeminiEvaluator with stub runner | ~380 |
| `internal/eval_harness/managed_agents_bridge.go` | MODIFY (small) — export `lastFencedBlock` as `LastFencedBlock` (or add a tiny shared helper) so the inject-in parser reuses the extract-out fence logic; no behavior change | ~5 |

No changes to `internal/executor/` (policy-free preserved). No changes to `internal/mission/quorum/` (frozen contract preserved). No changes to `cmd/ailang/exec.go` (seam already exists).

---

## Conflict Surface (MANDATORY)

This item touches executor invocation + evaluator wiring + sits adjacent to a frozen verdict contract. Enumerated seams and how each is protected:

### 1. Executor policy-free contract (`internal/executor/managed_agents/`)
- **What already lives here**: the executor builds `Input=[Directive]` + `SystemInstruction` and is DELIBERATELY policy-free (`managed_agents.go:158–163`; "The executor itself stays policy-free"). The eval harness owns bridging.
- **How we avoid breaking it**: we add ZERO code to the executor. The diff bundle goes into `task.Directive` (or `SystemPrompt`) — the exact fields the executor already reads and forwards verbatim. The bridge lives in `eval_harness/`, mirroring `managed_agents_bridge.go`'s stated rationale. Verified boundary: the sibling extract-out bridge already sets a precedent of harness-side-only bridging with no executor edits.

### 2. The frozen `quorum.ReviewResult` design-review contract (`internal/mission/quorum/reviewer.go`)
- **What already lives here**: `ReviewResult{Verdict pass|reject, StrongestObjection, Catch, ProposedFix}`, its `reviewSchema`, `ValidateReviewResult`, and the `AgenticRunner`/`agenticCaller` seam. Explicitly frozen ("The verdict CONTRACT is untouched", `agentic_caller.go:56`; "NOT part of the frozen verdict contract", `reviewer.go:54`).
- **The conflict**: the brief says "reuse the existing quorum/evaluator verdict JSON shape … do NOT invent a competing schema." Re-checked: `quorum.ReviewResult` is a **design-doc review** verdict (pass/reject), NOT a **sprint-evaluation** verdict (score/pass/blockers). Forcing sprint evaluation into `ReviewResult` would either (a) mutate the frozen contract, or (b) lose the score/blockers granularity the sprint-evaluator skill requires. **Resolution**: introduce a SEPARATE `GeminiVerdict` that mirrors the **sprint-evaluator skill's `evaluation_report_schema.md`** (the actual sprint-eval contract), not `quorum.ReviewResult`. This is not "inventing a competing schema" — it is matching the correct existing schema for this domain (sprint eval) while leaving the design-review frozen contract byte-identical. We add NO fields to `ReviewResult`, no new `Verdict` enum values, and do not import `quorum` into `eval_harness`.
- **Fixtures that MUST still pass unchanged**: `go test ./internal/mission/quorum/...` (all — we touch nothing here). Specifically `quorum_test.go`, `reviewer_test.go`, `agentic_caller_test.go`, `escalate_test.go`, `tier2_test.go`.

### 3. `ailang exec gemini` routing (`cmd/ailang/exec.go`)
- **What already lives here**: `resolveAgenticExecutorName("gemini")=="managed_agents"` (line 317), `--json` output shape (line 281–294), `executeCLI` sets `GCPProject`/`GCPLocation` from env (line 354–355).
- **How we avoid breaking it**: no code change to exec.go. The default `RunGeminiEvaluator` runner SHELLS `ailang exec gemini --json` (or calls the executor factory directly, same as `executeCLI` does) — it consumes the existing seam. If we call the factory directly we replicate `executeCLI`'s task construction; either way exec.go's contract is unchanged, and `exec_test.go` continues to pass.

### 4. The shared fence-parsing helper (`managed_agents_bridge.go:lastFencedBlock`)
- **The overlap**: both the extract-out (code block) and inject-in (verdict JSON block) paths need "last fenced block". Rather than duplicate, we export/share it.
- **Risk**: renaming a used unexported symbol. **Mitigation**: keep `lastFencedBlock` and add an exported thin wrapper (or rename + update the single internal call site in `extractCode`), then run `go test ./internal/eval_harness/...` to confirm `writeSolutionFromResponse`/`extractCode` behavior is unchanged. This is the "rename definitions too, test between each change" discipline from coding-standards.

### 5. Intentional (non-)changes
- **Deliberately unchanged**: the executor, the quorum contract, exec.go routing, the sprint-evaluator skill's markdown schema.
- **Deliberately new**: `GeminiVerdict` (sprint-eval domain), the diff-bundle builder, the reasoning-only directive const, the `VerificationDegraded` marker (no such marker exists today — grep confirmed zero hits for `VerificationDegraded`/`verification_degraded`/`bundle_truncated` in `internal/`).

**The honest answer is not "no conflicts":** the real conflict is the schema-domain confusion in the brief (sprint-eval vs design-review), resolved above by matching the correct existing schema instead of overloading the frozen one.

---

## Milestone breakdown

### M1 — Diff bundle builder + size/truncation policy (~0.75d)

**Deliverable**: `BuildDiffBundle(worktree, BundleOptions) (Bundle, error)` in `gemini_evaluator_bridge.go`, plus `Bundle`, `BundleOptions`, `DegradationInfo` types.

**Behavior**: enumerates the changed-file set via `git -C <wt> status --porcelain -z` (modified + staged + untracked), runs `git -C <wt> diff` + `git -C <wt> diff --cached` for tracked changes, synthesizes `+++ NEW FILE` headers + bodies for untracked files, appends full contents of changed (non-binary, non-generated) files, applies drop-order + byte ceiling, emits LOUD truncation markers.

**Acceptance criteria (each a Go test; each fails on base binary, passes after)**:
- `TestBuildDiffBundle_IncludesDiffAndFiles`: a temp git repo with 2 modified `.go` files → bundle contains both the unified diff and both files' full contents.
- `TestBuildDiffBundle_IncludesUntrackedNewFiles` (quorum catch): a temp git repo with a brand-new untracked `.go` file → bundle contains a `+++ NEW FILE: <path>` header AND the file's contents (proves `git diff` alone would have missed it).
- `TestBuildDiffBundle_DropsBinaryAndGenerated`: a modified `*.pb.go` and a NUL-containing file → both listed in `DroppedFiles`, NOT in `Text`, and a LOUD marker line present for each.
- `TestBuildDiffBundle_TruncatesOverCeiling`: `MaxBytes` set small, one large changed file → `Truncated==true`, file in `DroppedFiles`, marker present, and the `git diff` itself is STILL present (never dropped).
- `TestBuildDiffBundle_Deterministic`: two calls on the same worktree → byte-identical `Text` (drop order is stable/sorted).

### M2 — Reasoning-only directive + structured verdict parse/validate (~0.75d)

**Deliverable**: `BuildEvaluatorDirective`, `GeminiVerdict`, `ParseGeminiVerdict`, `ValidateGeminiVerdict`; `lastFencedBlock` shared/exported.

**Acceptance criteria**:
- `TestBuildEvaluatorDirective_ReasoningOnly`: directive contains the reasoning-only instruction, the bundle text, the design-doc/criteria, and the fenced-json verdict-schema instruction; a truncated bundle adds the "note unseen files in blockers" line.
- `TestParseGeminiVerdict_ExtractsFencedJSON`: response with prose + a final ```json verdict → parsed `{score,pass,blockers}`.
- `TestValidateGeminiVerdict_RejectsMalformed`: score>100, or blockers non-empty with `pass==true`, or degraded with empty reason → hard error (never coerced pass). Also: non-JSON / missing fence → error.
- `TestLastFencedBlock_UnchangedForExtractOut`: the shared/renamed helper still returns identical results for the extract-out cases in `managed_agents_bridge_test.go` (regression guard for Conflict-Surface seam 4).

### M3 — Caller seam + degradation stamping (~0.5d)

**Deliverable**: `RunGeminiEvaluator(ctx, worktree, designDoc, sprintPlan, opts)` with an injectable runner; degradation from truncation/drop/backend-error stamped onto the verdict.

**Acceptance criteria**:
- `TestRunGeminiEvaluator_StubHappyPath`: stub runner returns a valid fenced verdict → `RunGeminiEvaluator` returns it, `VerificationDegraded==false`.
- `TestRunGeminiEvaluator_TruncationStampsDegraded`: bundle over ceiling → returned verdict has `VerificationDegraded==true` + non-empty reason, EVEN IF the stub's verdict said `pass:true` (the marker is caller-enforced, not model-trusted).
- `TestRunGeminiEvaluator_BackendErrorIsDegradedNotPass`: stub runner returns `Success=false` → verdict is `VerificationDegraded==true` with the executor error text; result is NOT a fabricated pass (Principle 2). No live Vertex call in any test (stub only).

**Non-vacuity note**: every test above references a symbol that does not exist on the base binary, so `go test` fails to compile/pass before implementation and passes after — the design-doc-creator non-vacuity gate.

---

## Design questions — resolved answers

1. **Where does the evaluator path get invoked?** A new policy-layer helper `RunGeminiEvaluator` in `internal/eval_harness/` (mirroring `managed_agents_bridge.go`'s harness-side location and `quorum.RunAgenticReviewer`'s boundary), consuming the existing `ailang exec gemini`→managed_agents seam via an injectable runner. NOT a coordinator-level path; NOT an executor change. The mission-control Gate-4 evaluator lane calls it when routing selects gemini (routing wiring is out of scope — `m-mission-agentic-provider-routing.md`). Executor stays policy-free.

2. **Verdict contract**: introduce a SEPARATE `GeminiVerdict` (score/pass/blockers + degradation marker) that matches the **sprint-evaluator skill's `evaluation_report_schema.md`** — the correct existing schema for the sprint-eval domain. We do NOT overload `quorum.ReviewResult` (verified to be a design-doc-review verdict: pass/reject/strongest_objection, a different domain); its frozen contract stays byte-identical. This resolves the brief's "don't invent a competing schema" by matching the right existing schema, not the wrong frozen one.

3. **Size/truncation policy**: 256 KiB default ceiling (configurable via `BundleOptions.MaxBytes`); drop order = binary → generated/vendored → largest-remaining-whole-file; the `git diff` is never dropped; every drop emits a LOUD `=== BUNDLE TRUNCATED: … ===` marker in-text and in `DroppedFiles`. No silent drop.

4. **Degradation marker**: `GeminiVerdict.VerificationDegraded` (+ `DegradedReason`), stamped by the CALLER whenever the bundle truncated / dropped files / the backend errored. Caller-enforced (not model-trusted): a stub/model `pass:true` under truncation still surfaces `VerificationDegraded==true`. A degraded verdict is never treated as a full pass for routing. (Carried fleet-(c) watch-item; CLAUDE.md "no silent fallbacks".)

5. **Scope boundary**: reasoning-only. Explicitly OUT: local test re-runs by gemini, multi-turn sandbox reuse, making gemini the DEFAULT evaluator, GCS file bridging, overloading `quorum.ReviewResult`. This doc ships the CAPABILITY; the default stays sonnet-5.

---

## Axiom compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 minimal frozen core | 0 | No core change; no executor change. |
| A2 route-to-extension bias | +1 | Pure eval-harness policy layer; the archetypal extension, mirroring the existing bridge. |
| A3 no silent fallbacks | +1 | LOUD truncation markers + `VerificationDegraded` marker; degraded/backend-error never coerced to pass. |
| A4 deterministic | +1 | `BuildDiffBundle` is deterministic (sorted drop order, `TestBuildDiffBundle_Deterministic`). |
| A5 bounded waits | +1 | Reuses the executor's per-task timeout + bounded single interaction; no unbounded loop. |
| A6 semantic transparency | +1 | Structured verdict (score/pass/blockers) + explicit degradation reason. |
| A7 no core-floor violation | 0 | None. |
| A8 reuse over rebuild | +1 | Reuses managed_agents executor, `ailang exec gemini` seam, `lastFencedBlock`, sprint-eval schema. |
| A9 testability | +1 | Every element unit-tested with stub runner; no live Vertex in tests. |
| A10 observability | +1 | Verdict + degradation surfaced; executor already emits cost/turns via `--json`. |
| A11 composability | +1 | Three pure functions + a thin caller; composes with future routing. |
| A12 fail-loud | +1 | Malformed verdict = hard error; backend error = degraded, never silent pass. |

**Net**: +10, no −1 on A1/A3/A4/A7. Passes.

---

## Related documents

- `internal/eval_harness/managed_agents_bridge.go` — the EXTRACT-OUT precedent this doc mirrors (inverse).
- `design_docs/planned/v0_30_0/m-mission-agentic-provider-routing.md` — the routing layer that will later CHOOSE gemini as evaluator (out of scope here; this ships the capability it routes to).
- `design_docs/planned/v0_30_0/m-mission-quorum-agentic-verify.md` — the design-doc-review quorum whose frozen `ReviewResult` this doc deliberately does NOT overload.
- `.claude/skills/sprint-evaluator/resources/evaluation_report_schema.md` — the sprint-eval schema `GeminiVerdict` mirrors.

## Verification log

| Claim | Method | Result |
|-------|--------|--------|
| managed_agents request carries only Input+SystemInstruction, no repo upload | Read `managed_agents.go:164–176` | Confirmed |
| Executor is deliberately policy-free; harness owns bridging | Read `managed_agents.go:158–163`, `managed_agents_bridge.go:8–11` | Confirmed |
| Extract-out bridge appends instruction + parses fenced block | Read `managed_agents_bridge.go:38–46,110–190` | Confirmed |
| `ailang exec gemini` routes to managed_agents | Read `exec.go:317–322` | Confirmed |
| `--json` output shape (success/output/tokens/cost/turns) | Read `exec.go:281–294` | Confirmed |
| `quorum.ReviewResult` is pass/reject design-review, NOT score/blockers sprint-eval | Read `reviewer.go:20–61`, `agentic_caller.go` | Confirmed — brief's "reuse quorum verdict" premise is a domain mismatch; resolved by matching sprint-eval schema instead |
| Sprint-eval verdict shape is score/pass/blockers (0–100, pass≥70) | Read `evaluation_report_schema.md` | Confirmed |
| No existing `VerificationDegraded`/`bundle_truncated` marker | `grep -rn` in `internal/` | Confirmed zero hits — this marker is new |
| Target version folder exists | `ls design_docs/planned/v0_30_0/` | Confirmed |
| `git diff`/`diff --cached` omit untracked files; sprints create new files | design-quorum reviewer gemini-3-1-pro objection 2026-07-16 (git semantics) | Confirmed — doc CORRECTED to enumerate via `git status --porcelain` + synthesize `+++ NEW FILE` view; `TestBuildDiffBundle_IncludesUntrackedNewFiles` added |

## Quorum record

Run 2026-07-16T23:03:39Z (`.ailang/state/mission-quorum/m-gemini-evaluator-diff-bridge-2026-07-16T23-03-39Z.json`), reviewers `gpt5-6-sol,gemini-3-1-pro`, controller in-session verdict `pass`. Synthesis: **proceed** (exit 0) on controller verdict with BOTH external reviewers degraded to absent (N−1→N−2): `gpt5-6-sol` unreachable (OpenAI structured-output infra bug — `response_format` requires `proposed_fix` in `required`, an env-side schema mismatch, not a doc objection); `gemini-3-1-pro` `invalid` (response truncated mid-JSON). The gemini reviewer's partial objection was nonetheless VALID and actionable — untracked files are invisible to `git diff` — and has been incorporated (untracked-file inclusion + new acceptance test) rather than waved through. This is exactly the reject-by-default value: a degraded/partial reviewer still surfaced a real gap.

**DESIGN_DOC_PATH**: `design_docs/planned/v0_30_0/m-gemini-evaluator-diff-bridge.md`
