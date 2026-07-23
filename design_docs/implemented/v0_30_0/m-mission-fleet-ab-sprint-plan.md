# Sprint Plan — M-MISSION-FLEET-AB (Phases A+B of m-mission-adaptive-multiprovider-routing)

**Sprint ID**: M-MISSION-FLEET-AB
**Design doc**: [m-mission-adaptive-multiprovider-routing.md](./m-mission-adaptive-multiprovider-routing.md)
**Branch / worktree**: `mission/iter28-fleet-ab` @ `/Users/voightkampff/dev/sunholo-data/ailang-wt-iter28`
**Status**: completed (all milestones ✅, 2026-07-14; awaiting independent evaluator)
**Risk**: MEDIUM (touches the live mission driver + spends OpenAI/Vertex credits per design doc)
**Estimate**: ~1.5–2.0 dev-days total (Phase A **already landed** → verification-only ~0.25d; Phase B ~1.25–1.75d)
**GH bookkeeping**: #329

---

## HEADLINE FINDING (re-scopes this sprint)

**Phase A is already implemented AND deployed.** Commit `3bee6b6df`
("feat(mission): fleet Phase A — quota-aware model preference probing in the driver",
2026-07-14 07:32) is on `origin/dev`, is an ancestor of the main-checkout HEAD, and the
main-checkout on-disk `tools/launchd/mission-control.sh` (what launchd reads) contains the Phase A
probe loop (`MISSION_MODEL_PREFS` + `_mc_probe` + `QUOTA_SIG` matching, 5 matches). The transient
`mission-control.sh.new` staging file has been cleaned up.

The landed Phase A already delivers **exactly** the redundancy-audit's "genuinely new" surface for
Phase A — the multi-candidate loop + quota-error-signature matching — and nothing more. It:
- walks `MISSION_MODEL_PREFS` (default `claude-fable-5,claude-opus-4-8`) with a 1-token probe;
- classifies quota (fall through) vs transient (retry once) vs auth-dead (fall through);
- auto-recovers to a higher-preference model when its probe succeeds again (no dates);
- **subsumes** the old standalone auth probe (probe doubles as the subscription-auth check);
- preserves the kill switch, overlap guard, stall watchdog, override-file pin, `MISSION_MODEL` pin,
  `MISSION_DRY_RUN` path, and the `ANTHROPIC_API_KEY` strip (subscription billing);
- honors the **third-vocabulary rule**: its header comment cites `internal/ai/routing.go`
  `AIRoutingPolicy.Order` as the semantics it reuses, with the one-paragraph justification the doc
  requires (selection must happen in bash *before* any Go/claude process exists — the AIRoutingPolicy
  struct is program-runtime and cannot be consulted at driver-launch time; it is reused as the
  *ordering contract*, not the *implementation*).

**Consequence:** Milestone A is **not a build** — it is a verification/hardening pass on already-live
code (bash-n, dry-run, fall-through smoke, deployment confirmation). This matches the audit's
"only … loop + signature matching" scope and the ~0.5d estimate (now mostly spent).

---

## Verification log (design-doc claims checked against HEAD, 2026-07-14)

| Doc claim | Reality at HEAD | Verdict |
|---|---|---|
| Phase A = replace time-box override with probe loop, ~0.5d | **Already landed** `3bee6b6df` + deployed to main checkout | ✅ DONE (re-scoped to verify) |
| Auth probe at `mission-control.sh:128`; override at `:76–95` | Both existed at those lines pre-Phase-A; Phase A **replaced** them (auth probe folded into `select_model`) | ✅ (now historical) |
| `AIRoutingPolicy{Order,AllowFallback,Require,Prefer}` in `internal/ai/routing.go` | Confirmed verbatim (struct at routing.go:49) | ✅ |
| `gpt-5.6-sol` registered in models.yml 2026-07-12 | Confirmed: `gpt5-6-sol`, `api_name: gpt-5.6-sol`, `provider: openai`, `env_var: OPENAI_API_KEY`, `agent_cli: codex` | ✅ |
| Gemini provider reads `GEMINI_API_KEY` (rig: ABSENT) | **PARTIAL — the doc's worry is real but avoidable.** `ailang exec gemini --api-only` (exec.go:369) reads `GEMINI_API_KEY` → would FAIL on rig. BUT the eval-harness path (`ai_handlers.go` → `gemini.NewVertexAIClient`) uses **Vertex ADC**, and `models.yml gemini-3-pro` is configured `env_var: ""`, `gcp_project: ailang-dev` (ADC). **ADC token verified working on rig** (`gcloud auth application-default print-access-token` → OK). → **Phase B routes Gemini via the models.yml/Vertex-ADC path, NOT via `exec --api-only`.** | ✅ mitigated |
| OpenAI reachable on rig | `OPENAI_API_KEY` present; `gpt5-6-sol` uses it | ✅ |
| `internal/ai/{openai,gemini}` `handler.Call/CallJson` surface | Confirmed: `handler.go` `Call(input)`, `CallJson(input, schema)`, `CallWithContext`, `GenerateWithDetails` | ✅ |
| Prior-attempt ledger `m-unified-ai-control-plane.md` | Read fully. It is the STALLED heavy path: route ALL AI through `ailang exec` + schema migrations + eval-suite rewrite; its "Future Work" literally lists "Multi-provider fallback: try claude, fallback to gemini." **Phase B must NOT rebuild `ailang exec` unification** — it makes direct in-session provider calls. | ✅ avoided |
| Routing-evidence rows land in the mission log | Confirmed: `v1-mission.md` Gate 4 records "routing evidence row" per iteration in `v1-mission-log.md` | ✅ |

---

## Existing machinery Phase B rides on (do NOT rebuild — redundancy audit BINDING)

- **Text providers**: `internal/ai/openai` (`gpt-5.6-sol` via `OPENAI_API_KEY`) and
  `internal/ai/gemini` (`gemini-3-pro` via Vertex ADC). Both expose `handler.Call/CallJson`.
- **Model registry**: `internal/eval_harness/models.yml` — resolves `gpt5-6-sol` and `gemini-3-pro`
  to api_name + provider + auth; `ailang run --ai <model>` already constructs the right client
  (`ai_handlers.go` — ADC-first for Gemini, then `GOOGLE_API_KEY`).
- **The Claude controller's own review is IN-SESSION** — it is the running mission-control skill's
  own judgement, **NOT an API call** (per doc gate 6). No third provider call.

**Genuinely new (per audit) = ALL of Phase B's quorum logic**: N-reviewer orchestration,
reject-by-default scoring with a required "strongest objection" field, verdict synthesis,
catch-rate tracking hooks, and the artifact format.

## Existing CLI surface for one-shot text calls (Critical Principle 1)

Checked `ailang --help` subcommands, `cmd/ailang/exec.go`, `ai_handlers.go`, `chains_chat.go`.
Findings:
- `ailang run --ai <model> --caps AI` runs an `.ail` program that calls `AI.call` — heavyweight,
  needs an AILANG wrapper program. **Not ideal for a shell hook.**
- `ailang exec <provider> --api-only "<prompt>"` is a one-shot API text call — **but its Gemini
  branch reads `GEMINI_API_KEY` (absent on rig)**, so it works for OpenAI but not Gemini here.
- **DECISION**: Phase B's reviewer caller is a small Go helper (`ailang eval-review` OR a
  reused/extended internal entry) that constructs `openai` + `gemini` handlers **through the
  models.yml resolver** (so Gemini gets Vertex ADC), and returns structured JSON via `CallJson`.
  We do **not** invent a new top-level binary; we add one thin subcommand that reuses
  `eval_harness.GlobalModelsConfig` + `internal/ai` — the same primitives `ai_handlers.go` uses.
  This keeps us off the stalled `ailang exec` unification path while reusing shipped auth wiring.

---

## Milestones

### Milestone A — Verify + harden the already-landed Phase A driver (~0.25d)

**This is verification, not construction.** Phase A code is live. Confirm it is correct and safe.

**Tasks**
1. `bash -n tools/launchd/mission-control.sh` → syntax clean (already confirmed in planning).
2. `MISSION_DRY_RUN=1 tools/launchd/mission-control.sh` → prints `DRY RUN ok … prefs=…`, fires **no**
   probes, exits 0. (Bounded: single invocation, no wait.)
3. Fall-through smoke: `MISSION_MODEL_PREFS="bogus-model-xyz,claude-opus-4-8" MISSION_DRY_RUN=0`
   with a **≤2-min bounded** run that asserts the log shows `bogus-model-xyz … falling through` then
   selects opus. (Kill after model selection line; do NOT let the full 6h iteration run.)
4. Confirm invariants **unchanged** vs pre-Phase-A: kill switch (`grep KILL_SWITCH`), overlap guard
   (`pgrep -f "claude -p Run one mission"`), stall watchdog (`_mc_stalled`), `ANTHROPIC_API_KEY` strip,
   subscription-billing (probe uses `claude -p`, never the API).
5. Confirm deployment: main-checkout on-disk script contains `_mc_probe` (5 matches — confirmed).
   **No deployment action needed** (already deployed).

**Acceptance** — ✅ ALL MET (verified 2026-07-14, exec)
- [x] `bash -n` clean (exit 0).
- [x] dry-run exits 0 with no probes (`DRY RUN ok … prefs=claude-fable-5,claude-opus-4-8`; step-3 dry-run precedes step-4 `select_model`).
- [x] fall-through selects the next candidate within the bounded window: real `select_model` block (sed `88,134p`) + stubbed `claude` → `bogus-model-xyz … falling through` → selects `claude-opus-4-8` (`probe ok`), bounded 120s, zero real spend, no iteration launched.
- [x] all six safety invariants present and unchanged: kill switch (:149), overlap guard (:159), stall watchdog (:73), `ANTHROPIC_API_KEY` strip (:41), subscription-billing probe uses `claude -p` never API (:97/:102), override+env pins (:90/:110/:112).
- [x] deployment confirmed: main-checkout script has `_mc_probe` (3 matches). No driver edit made → no deployment action needed.

**Risk**: LOW. Read-only verification of live code. The one live-ish test (task 3) is time-boxed and
killed before the real iteration starts, so it cannot wedge the loop.

---

### Milestone B1 — Reviewer caller + reject-by-default prompt (~0.5d)

Build the off-Anthropic reviewer invocation as a thin Go subcommand reusing shipped primitives.

**Tasks**
1. Add subcommand `ailang design-review` (name TBD; reuses `eval_harness.GlobalModelsConfig` +
   `internal/ai` handlers — **no new provider code, no `ailang exec` unification**). Inputs: a
   design-doc path (or stdin), a reviewer model id (`gpt5-6-sol` | `gemini-3-pro`). Output: one
   reviewer's structured JSON verdict via `handler.CallJson(prompt, schema)`.
2. Gemini path MUST resolve through the models.yml/ADC route (`gemini-3-pro`, `env_var: ""`,
   Vertex ADC) — assert it does NOT require `GEMINI_API_KEY` (the rig-absent var).
3. Reviewer prompt = **reject-by-default**: score against the design-doc-creator hard gates
   (premise verification, Conflict Surface, axiom compliance) + a **required non-empty
   `strongest_objection` field** + a `verdict` ∈ {`pass`,`reject`} + `catch` free-text.
4. Per-call **budget cap**: `--max-cost-usd` (default $0.10/reviewer); refuse + record if the
   model's models.yml pricing × expected tokens would exceed it. No silent fallback (Principle 2).

**Acceptance** — ✅ ALL MET (verified 2026-07-14, exec; live smokes bounded)
- [x] `ailang design-review <doc> --reviewer gpt5-6-sol --json` returns valid JSON (verdict/strongest_objection/catch). Live: reject, $0.0055.
- [x] Same for `--reviewer gemini-3-1-pro` **without** `GEMINI_API_KEY` set (ADC path proven; ran under `env -u GEMINI_API_KEY` → reject, $0.0019). *(picked `gemini-3-1-pro`, the newer ADC pro, per the parked question.)*
- [x] Empty `strongest_objection`/missing field → hard error, not a pass (`ValidateReviewResult`). Budget-cap breach → structured error, zero spend (unit test asserts caller not invoked).
- [x] Unit tests: schema-conformance (stubbed handler), budget-cap refusal, ADC-vs-apikey resolution. No superseded tests existed to remove (new package).
- **Deviation (declared):** pre-flight budget estimate now scales to the ACTUAL doc size (chars/4), not a fixed 16k — the fixed estimate over-refused a small doc at gpt5-6-sol pricing. POST-check still uses real provider token counts.

**Risk**: MEDIUM. Real credit spend (OpenAI + Vertex). Mitigated by budget cap + cents/doc scale.

---

### Milestone B2 — Quorum orchestration + verdict synthesis + artifact (~0.5–0.75d)

Compose N reviewers into one quorum verdict with graceful degrade and a recordable artifact.

**Tasks**
1. Orchestrator: run `gpt5-6-sol` + `gemini-3-pro` reviewers (B1) **in parallel**; the Claude
   controller's own review is **in-session** (the mission-control skill's judgement — NOT an API
   call). Synthesis rule (from doc): **any `reject` → the objection goes back to the doc author
   before planning; unanimous `pass` → proceed.**
2. **Graceful degrade (Principle 2 for verdicts)**: a provider unreachable/over-budget → quorum
   proceeds with N−1 and the verdict record **explicitly names the absent reviewer** with the
   reason (`unreachable` / `budget` / `auth`). A missing reviewer is NEVER a silent pass.
3. **Artifact format** (seeds Phase E):
   - Machine: `.ailang/state/mission-quorum/<doc-slug>-<iso8601>.json` —
     `{doc, iso_ts, reviewers:[{model, verdict, strongest_objection, catch, cost_usd, present:bool,
     absent_reason?}], synthesis:{verdict, blocking_objections:[]}, controller_in_session:{verdict,
     note}}`.
   - Human: a short markdown block appended to the mission log's routing-evidence row (Gate 4),
     one line per reviewer + the synthesis verdict.
4. **Catch-rate tracking hook**: each reviewer record carries whether its objection was
   subsequently actioned (a `landed: null` field the doc author/evaluator flips) — the seed for the
   audit's "drop a reviewer whose objections never land". No dashboard, just the recorded field.
5. **Invocation point (documented HOOK, not a rewired skill — Non-Goal respected)**: the quorum is
   invoked at the **design-doc-creator gate / mission-control Gate 3** as an *optional documented
   step* — the design-doc-creator SKILL.md gets a one-paragraph "Quorum review (optional)" note
   pointing at `ailang design-review` + the orchestrator; **no inner-loop skill CONTRACT changes.**

**Acceptance** — ✅ ALL MET (verified 2026-07-14, exec; live smokes bounded ≤5 min)
- [x] Orchestrator on a real design doc writes a well-formed JSON artifact (`.ailang/state/mission-quorum/<slug>-<iso>.json`) + a mission-log markdown block (both inspected).
- [x] Kill one provider (`env -u OPENAI_API_KEY`) → gpt5-6-sol recorded absent (reason `auth`), NAMED in `absent_reviewers`, proceeds with N−1 (gemini present), still emits artifact. Never a silent pass.
- [x] `any-reject → blocked` (both reviewers rejected the fixture → exit 3) and `unanimous-pass → proceed` (covered by unit test `TestRunQuorum_UnanimousPassProceeds`) both exercised. All-absent → blocked (refuses zero signal).
- [x] design-doc-creator SKILL.md documents the optional quorum hook (one paragraph; no contract change).
- [x] Full-quorum cost recorded ($0.0074 for 2 reviewers) and ≤ budget cap × N. `controller_in_session` recorded as a distinct non-API entry.

**Risk**: MEDIUM. Orchestration is the genuinely-new engineering. Concurrency + degrade paths are the
sharp edges — covered by the N−1 degrade test.

---

## Day-by-day

| Day | Work |
|---|---|
| **Day 1 AM** | Milestone A (verify/harden landed Phase A — bash-n, dry-run, bounded fall-through smoke, invariant + deployment confirmation). Then B1 start: scaffold `ailang design-review`, wire models.yml resolver + `CallJson`, ADC-path assertion. |
| **Day 1 PM** | B1 finish: reject-by-default prompt + schema, budget cap, unit tests (schema, budget, ADC-vs-key). |
| **Day 2 AM** | B2: orchestrator (parallel reviewers), synthesis rule, artifact writer (JSON + mission-log markdown), N−1 degrade path. |
| **Day 2 PM** | B2 finish: catch-rate field, design-doc-creator SKILL.md hook paragraph, bounded integration tests (degrade, any-reject, unanimous-pass), cost check. CHANGELOG + example. Commit + PR. |

## Success metrics
- Phase A: all six safety invariants proven unchanged; dry-run + fall-through smoke green.
- Phase B: quorum runs a real design doc for **cents**, emits both artifacts, degrades to N−1
  **visibly**, and is reachable via a documented hook — **zero inner-loop skill-contract changes**.
- Gemini reviewer works on the rig **without `GEMINI_API_KEY`** (Vertex ADC), proving the doc's
  flagged risk is mitigated, not blocking.
- Tests added; superseded tests removed (coding-standards). `make test` + `bash -n` green.

## Bounded-wait discipline (mission gate 8)
Every live-ish test is time-boxed: dry-run is a single invocation; the fall-through smoke is killed
at the model-selection log line (≤2 min); the degrade integration test caps at ≤5 min. **No unbounded
polls anywhere** — this is the iteration-13 wedge lesson.

## Deployment / activation
- **Phase A: NONE required** — already committed AND on-disk in the main checkout (launchd reads it).
- **Phase B: no driver deployment** — Phase B is an on-demand tool + a skill-doc hook, invoked inside
  a mission iteration; it does not modify `mission-control.sh`, so no launchd/on-disk sync is needed.
- If any future Phase-B change DID touch `mission-control.sh`: the main tree is currently **dirty**
  (`sprint_M-CODEGEN-IR.json`, `BenchmarkGallery.jsx` — sibling work) → an automated `git pull` is
  **forbidden** (Principle 0). Controller may `git pull --ff-only` **only** after verifying the dirty
  paths don't include `tools/launchd/`; otherwise **park for human**. (Not triggered this sprint.)

## Non-Goals (this sprint)
- Phases C, D, E (cross-provider executors, local-GPU lane, assignment table) — explicitly OUT.
- Rebuilding the `ailang exec` unification (the stalled ledger path) — avoided by design.
- Changing inner-loop skill **contracts** — only a documented optional hook is added.
- Any GPU work / `rig.lock` touch — none.
- Reading subscription quota gauges directly — the probe is the deliberate substitute (unchanged).

## Parked for human
- **Was Phase A meant to be built THIS sprint?** It landed ~3h before planning (`3bee6b6df`), so this
  sprint verifies rather than builds it. Flag to controller in case a duplicate build was expected.
- **`gemini-3-pro` = "latest pro"?** Doc says "gemini (latest pro)"; rig has `gemini-3-pro` +
  `gemini-3-1-pro` in models.yml. Executor should pick `gemini-3-1-pro` if it's the newer pro and
  ADC-reachable; else `gemini-3-pro`. (Both are Vertex-ADC; either works — human/executor pick.)
- **Quorum-on-sprint-plans?** Doc says design docs "and optionally sprint plans". This sprint scopes
  the hook to **design docs only**; sprint-plan quorum deferred unless the controller wants it now.
