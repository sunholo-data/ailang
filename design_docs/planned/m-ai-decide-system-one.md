# M-AI-DECIDE-SYSTEM-ONE: Typed Decisions as an AILANG Primitive (TypeSafe Jev via OpenRouter)

**Status**: Planned
**Created**: 2026-09-18
**Target**: v0.40.1 (Phase 1: package + pricing row + shadow measurement; no core change) · Phase 2 gated on Phase 1 data
**Priority**: P1 — a 400 ms / $0.00003 typed-decision primitive for the places the harness currently spends a frontier-LLM turn or a substring heuristic
**Estimated**: Phase 1 ≈ 1.5 days · Phase 2 ≈ 2–3 days (only if Phase 1 clears its gate)
**Dependencies**: None new. `std/net`, `std/env`, `std/json` and the motoko extension ABI already provide everything Phase 1 needs (verified below).

> **Premise correction (read first).** The request called Jev "a new language that dovetails into AILANG".
> **Jev is not a language — it is a model.** TypeSafe AI's "System One" models are decision models: unstructured
> state in, *typed* answers out (`noul` = calibrated yes/no probability, `choice` = categorical with a
> distribution, `score` = ordinal with a distribution), each with a calibrated `confidence`. There is no
> syntax, type system or runtime to integrate. What *does* dovetail is the shape: a System One answer is a
> closed ADT with an explicit probability distribution, which is precisely what AILANG's typed, effect-explicit,
> `Result`-returning style wants at a decision point. That is the integration this doc designs.

## PROGRAM.md Routing

**Lane: extension** (Phase 1). Nothing in `internal/{parser,types,eval,core,elaborate,effects,builtins,lexer,ast,pipeline,runtime,link,iface}` is touched. Phase 1 is (a) an AILANG **package** `sunholo/decisions` in `ailang-packages`, (b) one pricing row in `internal/modelreg/models.yml`, and (c) a shadow measurement that produces the data Phase 2 is gated on. Default bias applied: *it can be an extension, so it is one* — the spike proved the whole call is expressible in pure AILANG today (Key Fact 5).

**Phase 2 (conditional) = AILANG lane, stdlib-only**: a `std/ai.decide` builtin backed by a Go `DecisionProvider` so decision calls get the AI effect's budget/cap/trace-span accounting that `std/net` calls do not have (Key Fact 7). No parser/typechecker surface either way.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The model is **measurably non-deterministic** (Key Fact 4: same input → `billing` 0.65–0.73 across 5 calls). The design keeps this explicit: `decide` is `! {Net, Env}` (Phase 2: `! {AI}`), never pure; the *full distribution* + model id are returned and must be recorded, never argmax alone (D6). No implicit non-determinism is introduced; an explicit effect is how AILANG already handles every other oracle. |
| A2: Replayability | +1 | Parsing is split from the call: `parseAnswers: string -> Result[...]` is pure and unit-testable against banked fixtures (Files); the effectful shell is ~20 lines. A recorded response replays without a network. |
| A3: Effect Legibility | +1 | Signature declares `{Net, Env}`; no hidden side effects. Phase 2 moves it under `{AI}` where it belongs semantically. |
| A4: Explicit Authority | 0 | Uses the existing `Net` capability (+ `--net-allow-domains` if the operator scopes it) and the already-registered `OPENROUTER_API_KEY`. No new ambient access. Phase 2's optional `TYPESAFE_API_KEY` goes through `internal/config` Registry (forbidigo), never a bare `os.Getenv`. |
| A5: Bounded Verification | +1 | Answers are a closed ADT; unknown answer `type` is an `Err`, not a default. |
| A6: Safe Concurrency | 0 | One HTTP round trip; no shared state. |
| A7: Machines First | +1 | The whole point: a decision returns machine-consumable typed values with a distribution instead of prose a human (or a second LLM) must interpret. |
| A8: Minimal Syntax | +1 | No new syntax; a package exporting three types and three functions. |
| A9: Cost Visibility | +1 | Adds the missing pricing row so the observatory stops reporting Jev spans as `unpriced` (Key Fact 8). Phase 2 brings per-call cost into `ailang trace`. Phase 1 documents loudly that `std/net` calls are NOT in the AI budget (a known limitation, not a silent one). |
| A10: Composability | +1 | Composes with the motoko extension ABI (`on_pre_step`, `on_solver_candidate` already carry `{Net, Env, AI}` — Key Fact 9), with `std/ai` (an LLM arm answers the *same* `Question` values), and with `Result`. |
| A11: Structured Failure | +1 | `DecideError = MissingKey \| Http(int, string) \| Transport(string) \| BadResponse(string)`; no empty-Ok, no default threshold (D4). |
| A12: System Boundary | 0 | An external vendor behind OpenRouter — the same boundary every OpenRouter model already sits behind. |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the effect is explicit and the distribution is surfaced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Key Facts (verified 2026-09-18, this session)

Every load-bearing premise below was checked live; the command or file that produced it is cited. Premises about the vendor (1, 2, 4, 12) are about a system we do not control and are the reason quorum trigger #4 fires (see Quorum).

1. **Jev is live on OpenRouter as `typesafe/jev-1.13`, beta, since 2026-09-17.** `GET /api/v1/models/typesafe/jev-1.13/endpoints` → endpoint `typesafe/jev-1.13-20260917`, `modality: "text->decisions"`, `context_length: 32000`, pricing `prompt: 0.000000042` ($0.042/MTok), `completion: 0`, `supported_parameters: []`. It is **not** in the general `/api/v1/models` listing (445 models, zero hits for jev/typesafe — beta models are hidden), and `typesafe/jev-latest` 404s on OpenRouter (`jev-latest` is the alias on TypeSafe's *direct* API only). Any code that discovers models from `/api/v1/models` will not see it.
2. **The calling convention is a dedicated endpoint, not chat completions.** `POST https://openrouter.ai/api/alpha/decisions` (documented at `openrouter.ai/docs/api/api-reference/alphadecisions/…`; schema `DecisionsRequest{model, state: string|object|array, questions: {name: NoulQ|ChoiceQ|ScoreQ}, session_id?, provider?, trace?, user?}` → `DecisionsResponse{model, answers, usage{input_tokens, output_tokens, cost}, id, provider}`). Probes of `/api/v1/systemone`, `/api/v1/system-one`, `/api/v1/decisions` all 404. The request/response bodies are byte-compatible with TypeSafe's direct `POST https://api.typesafe.ai/v1/systemone` (fixture: `typesafe-ai/system-one-adapter-python` `tests/cassettes/…typesafe_response…json`), so a later direct-API transport is a URL + header swap.
3. **Measured latency and cost (curl, this machine):** first call 508 ms total, then 329–980 ms over 5 repeats; provider-side 246 ms per the Broadcast span. Cost per call $0.0000178 (3 questions, 424 in / 73 out tokens) and $0.0000310 (4 questions, 737 in). The 32k context is the hard input ceiling.
4. **Not deterministic.** Five identical requests: `department.billing` ∈ {0.73, 0.71, 0.65, 0.66, 0.66}, `technical` ∈ {0.27…0.35}; argmax stable; `is_urgent` noul flipped 0.99/1.0. A `score` answer with a peaked distribution was stable. **Consequence:** an AILANG binding must be an effect and callers must persist the distribution, not the label.
5. **A zero-Go path exists and works.** `examples/runnable/decide_jev.ail` (banked this session, 87 lines; becomes the package's `decide.ail`) builds the request with `std/json (jo, kv, js, ja)`, reads the key with `std/env.getEnv`, posts with `std/net.httpRequest("POST", …)`, decodes with `std/json.decode`, and returns `Result[[{name, answer: Answer}], string]`. `ailang check` → `✓ No errors found!` (one `STRICT_FALLBACK_001` on the recursion base case, resolved with `@allow_empty_ok`). `ailang run --caps Net,Env,IO` → three typed answers in **518 ms wall-clock including type-check**. Output (real AILANG friction as state): `lane: choice=motoko_extension conf=0.6 … auto_fixable: noul p=0.89 … severity: score=2.77 conf=0.77 dist=[…{key: 3, p: 0.84}]`.
6. **The config-driven provider path is NOT available for this (negative-existence).** `internal/ai/configdriven/shapes.go:31` — `request_shape="custom"` returns `"reserved for schema v2; not implemented in v1"`; and `internal/ai/configdriven/jsonpath.go:92-105` `extractString` errors unless `response_path` resolves to a JSON *string* (`answers` is an object). So `[[ai_provider]]` cannot carry a Decisions call without Go changes; that is Phase 2's job, not a config file's.
7. **`std/net` calls carry no cost or trace-span accounting; `std/ai` calls do (negative-existence).** `grep -n "Budget\|Cost\|cost\|trace\|span" internal/effects/net.go` → no matches. `internal/ai/span.go` records provider spans with error classification and `internal/observatory/pricing.go` prices AI spans. **Consequence:** a Phase-1 decision is invisible to `ailang trace` cost totals and to the per-effect AI budget — visible only through OpenRouter's Broadcast traces (next fact). This is the single strongest argument for Phase 2 and is stated in the package's AGENT.md.
8. **OpenRouter Broadcast already captures Decisions calls, but our observatory cannot price them.** Prod observatory (`dashboard.ailang.sunholo.com/api/observatory/spans`, 30-min window) shows 16 spans for the spike: `LLM Generation` with `gen_ai.operation.name: "decisions"`, full `rawRequest`, `tokens_in: 424, tokens_out: 73`, `model: "typesafe/jev-1.13-20260917"`, and **`ailang.cost.unpriced: true, unpriced_reason: model not in registry`**. Resolution is `internal/modelreg/resolve.go`: `Resolve` tries the literal name against key/`api_name`/`aliases`, then **normalises only the query** (`normalizeModelName`, lines 132-140: trailing `-YYYYMMDD` stripped, `.`→`-`) and retries — `matchAPINameOrAlias` (lines 111-126) compares the row side literally. So the dated wire name `typesafe/jev-1.13-20260917` becomes `typesafe/jev-1-13`, which matches an `api_name` of `typesafe/jev-1.13` **only if** the row also declares `aliases: ["typesafe/jev-1-13"]`. One row with both spellings prices the bare and the dated id; a row with `api_name` alone prices only the bare id (verified by reading the code path, not inferred from output). (Note the observatory span carries `ailang.cost.*` attributes — it is our OTLP receiver stamping OpenRouter's Broadcast export, `internal/observatory/otlp_receiver.go`.)
9. **The motoko extension ABI can host a Jev call with no core change.** `ailang-packages/packages/motoko-ext-abi/types.ail:128-142`: `on_pre_step`, `on_tool_handle`, `on_response_intercept`, `on_solver_candidate` are typed `! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream}`; `on_build_system_prompt` and `on_tool_policy` are **pure** (no effect row). A confidence-gated tool policy therefore has to precompute in `on_pre_step` and stash via `state_key`/SharedMem; a done-gate second opinion fits `on_solver_candidate` directly (`FinalizeDecision = Accept \| ContinueWithFeedback \| NoDecision`).
10. **Name collision avoided.** `sunholo/motoko_ext_decision_framework` already exists (v0.2.2) — it is a keyword-gated *prompt patch* with "no LLM call". Ours is `sunholo/decisions` (plural, no "framework") and its AGENT.md cross-links to make the difference explicit.
11. **Env-var registry state.** `OPENROUTER_API_KEY` is registered (`internal/config/providers.go:14,26`). `grep -rn TYPESAFE internal/config/` → nothing; a direct-API transport therefore needs a Registry row first (forbidigo blocks a bare `os.Getenv`). Phase 1 uses only the OpenRouter key.
12. **Vendor-stated limits** (`docs.typesafe.ai/concepts/state.md`, `…/confidence.md`): text only; English primary, "CJK … lower accuracy"; `confidence` is a statistic *derived from* the distribution, absent on `noul`; the vendor's own guidance is "thresholds scale with risk … your code encodes the risk tolerance" — i.e. the threshold belongs to the caller, which is also our no-silent-fallbacks rule.
13. **A control arm exists for free.** `typesafe-ai/system-one-adapter-python` is a drop-in that answers the *same* `state + questions` protocol with any chat LLM (structured outputs or prompted JSON, `llm_answer_mode="probabilities"`). We do not need the Python package: an OpenRouter chat-completions call with `response_format: json_schema` derived from the same `Question` list (per-label probabilities requested, normalised client-side exactly as the adapter does) is the AILANG-native equivalent — sent via `std/net.httpRequest`, not `std/ai`, for the deadline reason in Key Fact 15 — and gives the A/B "is Jev better than a cheap LLM classifier *for our decisions*" that the routing gate (Phase 1 M3) needs.
14. **Labelled ground truth for the first consumer exists in-repo — smaller than first counted.** *(Corrected during M3.)* 45 design docs matched a routing-section grep, but only **20** state their lane legibly enough to label (12 extension / 6 AILANG fix / 1 core-floor / 1 other). `tools/decisions/routed_docs.sh` holds the hand-read labels with the quoted evidence sentence per row. Their Problem Statement/Goals are the `state` (routing section and evidence sentence stripped); the declared lane is the label.

15. **Bounded waits — what each transport actually enforces (quorum objection, verified).** `std/net.httpRequest` runs under `ctx.Net.Timeout`, default **30 s** (`internal/effects/context.go:221-233`; `internal/effects/net.go:86,186,550` set `Timeout: ctx.Net.Timeout` on every client), operator-tunable per run with `--net-timeout` (`cmd/ailang/main_run.go:77`). **The `std/ai` single-shot path has no deadline (negative-existence):** `grep -rn "WithTimeout\|Deadline\|Timeout" internal/effects/ai*.go internal/builtins/ai*.go internal/ai/handler.go internal/ai/openrouter/client.go` finds only doc strings; `internal/ai/httpjson.go:71` and `internal/ai/openrouter/client.go:124` fall back to `http.DefaultClient`, whose `Timeout` is 0. **Consequence for this doc:** the Jev call is bounded at 30 s by construction; the LLM control arm must NOT go through `std/ai.callJsonResult` — it goes through the same `httpRequest` (OpenRouter chat completions, structured outputs) so both arms share one enforced deadline and one transport. The missing `std/ai` deadline is a pre-existing gap filed to the AILANG-fix backlog (Future Work), not fixed here.

## Problem Statement

The harness makes a large number of small classification decisions, and today each one is made in one of two bad ways:

- **Hand-written heuristics in Go.** `internal/coordinator/retry_chain.go:70 ClassifyFailure` is substring matching; `error_category` carries `api_error` as an explicit catch-all "cause unknown" (`internal/eval_harness/metrics.go:214`). These are cheap and deterministic but cannot say *how sure* they are, so every miss is silent until someone mines the bank (the 2026-09-18 failure-analysis pass produced 25 docs from exactly that mining).
- **A full frontier-LLM turn.** Routing friction into a PROGRAM.md lane, deciding whether a sprint candidate is done (`on_solver_candidate`), triaging 62 pending approvals — each is seconds-to-minutes and cents-to-dollars on fable/opus/astra, returns prose that must be parsed, and consumes the quota the mission loop keeps running out of (quota-scarcity is a standing priority).

Neither gives a **calibrated** answer. TypeSafe's model gives exactly that — typed answer + distribution + confidence — in ~400 ms for ~$0.00003, and AILANG already has the right vocabulary to receive it: a closed ADT inside a `Result` behind an explicit effect. The gap is a ~100-line package, a pricing row, and a measurement that tells us which decision points it is actually good at. Without the measurement we would be adding a vendor on vibes; with it, Phase 2 has data.

## Goals

**Primary goal:** Make "ask a typed question of the current state and get a calibrated typed answer" a one-import AILANG primitive, and measure it on one real harness decision before wiring it anywhere that acts.

**Success metrics:**
1. `import pkg/sunholo/decisions/decide (decide, Noul, Choice, Score)` works from any AILANG program with `--caps Net,Env`; round trip ≤ 1 s p95 from the rig.
2. Jev spans in the prod observatory price correctly (`ailang.cost.unpriced` absent) — verified by a `modelreg` test resolving `typesafe/jev-1.13-20260917`.
3. Shadow lane-router agreement on the 45-doc set is **reported** (Jev vs declared lane; LLM-arm vs declared lane; Jev vs LLM), with confidence-stratified breakdown. The number itself is not a pass/fail — the gate is that it exists and is banked under `.ailang/state/decisions/`.
4. Zero code paths *act* on a Jev answer in Phase 1 (shadow only). Phase 2 scope is chosen from the measurement, not assumed.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. Primitive lives in a **package** (`sunholo/decisions`) first; `std/ai.decide` only after Phase 1 data | PROGRAM.md default bias; a package ships today with no core touch; stdlib promotion is a one-way door for the API shape | agent (rule is already ratified in PROGRAM.md §4) | design | med |
| D2. Phase 1 transport = **OpenRouter `/api/alpha/decisions` only**; direct TypeSafe API deferred | One key already registered and Broadcast-visible; direct API means a new env var, separate billing and a second trace instrument to reconcile | agent | design | low |
| D3. Answer representation = closed ADT `Answer = NoulA(float) \| ChoiceA{choice, confidence, probabilities: [{key, p}]} \| ScoreA{score, confidence, probabilities}`; distributions as ordered `[{key, p}]` records | It is the type every consumer pattern-matches on; changing it later breaks every caller | agent | design | med |
| D4. **No default confidence threshold anywhere in the package, and no two-way gate.** Callers pass thresholds; `gate(answer, minConf)` returns a three-way `Gate` (`Act \| Escalate \| Ungateable`) so a variant mismatch can never read as "low confidence" | Thresholds scale with the stakes of the action (vendor guidance + our no-silent-fallbacks rule); a package default would be a silent policy | agent | design | low |
| D5. **First consumer = shadow lane-router over the 45 routed design docs, plus an LLM control arm** — never an acting consumer in Phase 1 | Directly serves the program's heart (§4 routing), has in-repo ground truth, costs < $0.10, and settles "is Jev better than a cheap LLM at *our* decisions" before anything acts on it | **human (Mark) — RATIFIED 2026-09-18** | design | low |
| D6. Every decision call persists `{model, request_hash, answers-with-distributions, usage, latency_ms}`; argmax alone is never banked | Non-determinism (Key Fact 4) means a bare label is unreproducible and un-auditable | agent | compile | med |
| D7. The `models.yml` row is **priced but not in any suite list**, `provider: openrouter`, no `agent_*` fields | A `text->decisions` model cannot answer a chat prompt; `eval-suite` with no `--models` runs `dev_models × every tier` (real spend) and would bank 100% `api_error` rows for it | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **D5 — first consumer: the shadow lane-router (M3) with the LLM control arm.** Ratified by Mark, attended session 2026-09-18 ("lets sprint plan and execute it then as an extension"). Alternatives considered: (b) sub-classifying banked `api_error` rows (no ground truth — would need hand labels first); (c) a `on_solver_candidate` done-gate second opinion in motoko (acts on the loop; Phase 2 material once (a) shows the model is good at AILANG-shaped state). Mark to confirm or redirect.
- [x] D1, D2, D3, D4, D6, D7 — agent-resolvable under existing ratified rules (PROGRAM.md §4, no-silent-fallbacks, eval-suite scope rule); recorded here so the sprint does not re-derive them.

## Solution Design

### Overview

Three phases, only the first two funded by this doc:

- **Phase 0 — spike (DONE 2026-09-18).** The wire protocol, latency, cost, non-determinism and a pure-AILANG implementation are all measured (Key Facts 1–5, 8).
- **Phase 1 — package + pricing + shadow measurement (this sprint).** Ship `sunholo/decisions`, price the model, run the lane-router shadow with an LLM control arm, bank the agreement report.
- **Phase 2 — conditional stdlib promotion.** If Phase 1 shows the model is useful on AILANG-shaped decisions (agreement with declared lane ≥ the LLM arm's, or better-calibrated at equal agreement), add `std/ai.decide` + a Go `DecisionProvider` (OpenRouter first, TypeSafe direct second) so decisions get AI budget/cap/trace accounting, and pick the first *acting* consumer from the measurement. Phase 2 gets its own design doc; this one only reserves the seam.

### Architecture

```
AILANG program / motoko extension hook
        │  import pkg/sunholo/decisions/decide (decide, Noul, Choice, Score, Answer)
        ▼
sunholo/decisions/decide.ail            ! {Net, Env}
   buildRequest : (model, state: Json, [{name, q}]) -> string        -- pure
   parseAnswers : (string, [{name, q}]) -> Result[[{name, answer}], DecideError]  -- pure, fixture-tested
   decide       : shell = getEnv + httpRequest + parseAnswers
        │  POST https://openrouter.ai/api/alpha/decisions   (Bearer OPENROUTER_API_KEY)
        ▼
OpenRouter ──► TypeSafe (jev-1.13)         ~250 ms provider-side
        │
        └──► Broadcast trace ──► prod observatory (priced via internal/modelreg once the row lands)
```

Phase 2 replaces the middle box with `_ai_decide` → `ai.Handler` → `DecisionProvider` (optional capability interface next to `StreamingProvider`, `internal/ai/provider.go:469`), keeping the AILANG-facing types identical so Phase-1 callers migrate by changing one import.

### Conflict Surface: reuse `std/ai` now, or package first? (quorum objection, oc-glm-5-2)

The overlap is real and named in Key Fact 7: `std/ai` already has budget frames, the AI cap, provider spans with cost, and a typed `AIError`; a Decisions call routed through it would be visible to `ailang trace` on day one. The reviewer is right that "PROGRAM.md says extension" is a bias, not an argument. The argument:

| | **A. Package over `std/net` (chosen for Phase 1)** | **B. `DecisionProvider` in Go now** |
|---|---|---|
| Files touched | `ailang-packages` + one `models.yml` row + a tool | `internal/ai/provider.go` (new optional interface), `internal/ai/openrouter/` (new endpoint + response type — the Decisions body is not a chat completion, so none of `chat.go`/`step.go` is reusable), `internal/ai/handler.go` (dispatch), `internal/effects/ai*.go` + `internal/builtins/ai*.go` (new `_ai_decide` builtin), `std/ai.ail` (new types + func) |
| Release needed | No — a package publishes independently; the row can ride any release | Yes — a binary change; the next tag |
| Conflict Surface section | Not required (no core files) | Required: `internal/effects` + `internal/builtins` are on the list; plus the `AIError` code set gains no new code but `Response` gains a `Decisions` shape that every existing provider must reject explicitly (else `CodeInternal` "not implemented" — the existing convention at `provider.go:447-453`) |
| What is invisible in Phase 1 | ≈ 45 + 20 calls ≈ **$0.01** of `trace` cost, with Broadcast as the independent record (Key Fact 8) | Nothing |
| Migration cost when the seam moves | One tool (`tools/decisions/lane_shadow.ail`) and one example change an import; **there are zero other callers by design** (shadow only, Success Criteria) | None — but if M3 shows the ADT shape is wrong (e.g. `Score` wants the levels back, or `state` wants `Json` vs `string`), that shape is already frozen into `std/ai`, a one-way door |
| What Phase 1 learns that B would need | Whether the `Question`/`Answer` ADT is the right stdlib shape, measured on a real consumer, before it is frozen | — |

**Decision: A for Phase 1, B for Phase 2, with the ADT carried over unchanged unless M3 says otherwise.** The deciding weights are the release coupling (a stdlib change waits for a tag; the measurement can run today) and the one-way door: `std/ai`'s surface is what every eval model is taught, and adding a decision primitive there should follow one measured consumer, not precede it. The cost of A's invisibility is bounded and tiny; the cost of B's premature freeze is not. Phase 2's doc inherits this table as its opening Conflict Surface row and adds the `Response`/`AIError` analysis above in full.

**Package API (Phase 1):**

```ailang
module sunholo/decisions/decide

export type Question =
  | Noul(string)                                     -- instructions -> P(yes)
  | Choice(string, [{ key: string, desc: string }])  -- instructions, options
  | Score(string, [string])                          -- instructions, ordered levels (index = score)

export type Answer =
  | NoulA(float)
  | ChoiceA({ choice: string, confidence: float, probabilities: [{ key: string, p: float }] })
  | ScoreA({ score: float, confidence: float, probabilities: [{ key: string, p: float }] })

export type DecideError =
  | MissingKey(string) | Http(int, string) | Transport(string) | BadResponse(string)
  | MissingAnswer(string) | WrongVariant(string)

export type Decision = { model: string, answers: [{ name: string, answer: Answer }],
                         input_tokens: int, output_tokens: int, cost_usd: float, id: string }

export func decide(model: string, state: Json, questions: [{ name: string, q: Question }])
  -> Result[Decision, DecideError] ! {Net, Env}

-- Gate is the ONLY policy-shaped value the package produces, and it is three-way on
-- purpose: a wrong variant is never reported as "not confident enough" (quorum
-- objection, gemini-3-1-pro — a two-way Option conflated the two and would have sent
-- callers down their escalation path on a type mismatch).
export type Gate =
  | Act(string)          -- confidence >= minConfidence: the label (ChoiceA) or the rounded level (ScoreA)
  | Escalate(float)      -- confidence <  minConfidence: carries the confidence so the caller can log it
  | Ungateable(string)   -- NoulA has no confidence: threshold its probability directly (message says so)

-- Pure helpers (no policy, no defaults):
export func answer(d: Decision, name: string) -> Result[Answer, DecideError]   -- Err(MissingAnswer(name)), never None
export func gate(a: Answer, minConfidence: float) -> Gate                        -- exhaustive match forces callers to handle all three
export func expectedScore(a: Answer) -> Result[float, DecideError]              -- Σ level·p; Err(WrongVariant) unless ScoreA
export func parseAnswers(body: string, questions: [{ name: string, q: Question }]) -> Result[Decision, DecideError]
export func questionsToJsonSchema(questions: [{ name: string, q: Question }]) -> string   -- pure; the ONLY LLM-facing export (a schema, not an answer)
```

**What the package does and does not guarantee about provenance (quorum objections, gemini-3-1-pro round 1 and gpt6-astra round 2).** `parseAnswers` is exported because replay from a banked body is an A2 requirement, and any exported decoder can be fed forged JSON — so the package makes **no type-level claim** that an `Answer` came from a System One model. What it does guarantee: (a) it ships **no LLM-facing helper** — nothing that takes a chat-completion body and yields an `Answer`; the only LLM-facing export is `questionsToJsonSchema`, which produces a schema, not an answer; (b) every `Decision` carries the wire `model` and `id`, and D6 requires both to be banked with every row, so provenance is auditable per row; (c) `AGENT.md` states in one line that `Answer.confidence` is meaningful only for rows whose `Decision.model` is a System One model. The shadow runner's separate `LlmAnswer` type is a discipline for *our* tool, not a guarantee for third parties — the doc no longer claims otherwise.

**Bounded waits:** `decide` inherits the Net effect's enforced deadline (30 s default, `--net-timeout` to change) — Key Fact 15. The package's AGENT.md states the number; consumers that need tighter bounds set the flag or, in Phase 2, the AI budget frame.

`state` is `Json`, not `string`, on purpose: the vendor recommends an object state with named parts ("the material you would present to a panel of experts"), and callers that already hold `Json` should not re-encode.

### Implementation Plan

**M1 — the package (0.5 day)**
1. Create `ailang-packages/packages/decisions/` from the spike: split into `buildRequest` (pure), `parseAnswers` (pure), `decide` (shell); add `DecideError`, `Decision` (with `usage.cost` from the response), helpers `answer`/`gate`/`expectedScore` and the `Gate` ADT.
2. `ailang.toml`: `name = "sunholo/decisions"`, `[effects] max = ["Net", "Env"]`, `ailang = ">=0.40.0"`, tags `["ai", "decisions", "typesafe", "openrouter", "classification"]`, `ai_summary` stating the non-determinism + the "not in AI budget/trace cost" limitation verbatim.
3. Fixtures: bank the three spike responses (support ticket; AILANG friction ×2) under `packages/decisions/fixtures/*.json`; tests call `parseAnswers` on them and on three malformed bodies (missing `answers`, unknown `type`, missing named question) asserting the `Err` variant — runnable with `--caps ""`.
4. `AGENT.md`: what it is, the protocol, D4/D6 as rules for consumers, the limitation from Key Fact 7, the cross-link to `motoko_ext_decision_framework` (Key Fact 10).
5. `ailang publish` per the `ailang-packages` skill; verify `import pkg/sunholo/decisions/decide` resolves from a fresh dir.

**M2 — price the model (0.25 day)**
1. Add `or-typesafe-jev-1-13` to `internal/modelreg/models.yml`: `api_name: "typesafe/jev-1.13"`, `aliases: ["typesafe/jev-1-13"]` (the normalised form the resolver produces from the dated wire name — Key Fact 8), `provider: "openrouter"`, `env_var: "OPENROUTER_API_KEY"`, `pricing: {input_per_1k: 0.000042, output_per_1k: 0.0}`, `max_output_tokens: 28800`, `description` naming `text->decisions` and that it must not be used as a chat model. **No** `agent_cli`/`agent_model_name`, and not added to any suite list (D7).
2. Test in `internal/modelreg`: `Resolve("typesafe/jev-1.13-20260917")` and `Resolve("typesafe/jev-1.13")` both hit the row; `PriceTokens(…, 424, 73)` ≈ $0.0000178 within 1e-9. Check `TestModels_PricingIsSlugConsistent` and any "0/0 = free-local" labelling logic do not misclassify a `0` output price with a non-zero input price (memory: `0/0` once mapped to a positively-false `free-local` label).
3. Confirm on prod after the next deploy that a fresh Decisions span prices (the observatory pricing path is deploy-gated; note it in the sprint as a post-merge check, not a blocker).

**M3 — shadow lane-router measurement (0.75 day)** *(D5, pending Mark)*
1. `tools/decisions/lane_shadow.ail` (AILANG, `! {FS, Net, Env, IO}`): for each of the 45 routed design docs, `state = {problem_statement, goals, files_to_modify}` (first ~6k chars — well under 32k), `questions = { lane: Choice(PROGRAM §4 lanes + "mission"/"registry" as seen in the corpus), touches_core: Noul(...), severity: Score(P2/P1/P0) }`; label = the doc's declared lane. Run Jev; run the **control arm** through the same `std/net.httpRequest` against OpenRouter chat completions with `response_format: json_schema` generated from the same `Question` list — requesting per-label probabilities, never a bare label — against one cheap chat model (the implementer picks from `dev_models`; record which). Same 30 s deadline, same transport, own `LlmAnswer` type (Example 2; Key Fact 15). Persist per-doc rows per D6 to `.ailang/state/decisions/lane_shadow_<date>.jsonl`.
2. `tools/decisions/lane_shadow_report.sh` (or a `--report` mode): agreement Jev↔label, LLM↔label, Jev↔LLM; agreement stratified by Jev `confidence` tertile (does confidence predict correctness? — the calibration claim is the vendor's headline and the thing worth checking); mean latency and total cost per arm. Markdown table to stdout, committed under `design_docs/planned/m-ai-decide-system-one-shadow-report.md`.
3. Add the report's one-line summary to this doc's "Where we are" footer and to PROGRAM.md §5b as a **candidate** extension row (`decision-gate`), status "measured, not acting".

### Files to Modify/Create

- `~/dev/sunholo-data/ailang-packages/packages/decisions/decide.ail` — the package (~150 LOC; from the 85-line spike + error ADT + helpers)
- `~/dev/sunholo-data/ailang-packages/packages/decisions/ailang.toml` — manifest (~30 LOC)
- `~/dev/sunholo-data/ailang-packages/packages/decisions/AGENT.md` — agent-facing doc (~80 lines)
- `~/dev/sunholo-data/ailang-packages/packages/decisions/tests/parse_test.ail` — pure fixture tests (~80 LOC)
- `~/dev/sunholo-data/ailang-packages/packages/decisions/fixtures/*.json` — three banked responses + three malformed
- `internal/modelreg/models.yml` — one row (+12 lines)
- `internal/modelreg/resolve_test.go` — dated-id resolution + price assertion (+25 LOC)
- `tools/decisions/lane_shadow.ail` — shadow runner incl. `LlmAnswer`, `parseLlmAnswers`, `chatBody` (~260 LOC), M3
- `tools/decisions/lane_shadow_report.sh` — aggregation (~60 LOC), M3
- `examples/runnable/decide_jev.ail` — **already banked** (the Phase-0 spike, `ailang check` clean; with no key it prints `ERROR: OPENROUTER_API_KEY not set` and exits 0). M1 adds its `examples/manifest.json` entry (`go run ./scripts/validate_manifest.go --ci` passes with it unlisted, but the entry is the house rule). `examples/manifest.json:227` lists `claude_haiku_call.ail` as `status: "working"` with no network exclusion — it passes verify because it handles every failure and exits 0. `decide_jev.ail` follows the same contract: missing `OPENROUTER_API_KEY` → prints `MissingKey` and exits 0 (already the spike's behaviour)
- `docs/docs/packages/sunholo/decisions.md` — package page (~60 lines), linking to the vendor docs and stating the limitations
- `CHANGELOG.md` — entry under Unreleased
- `design_docs/PROGRAM.md` — §5b candidate row (1 line), after M3

## Examples

### Example 1: Route a compile-error friction (the spike, verbatim behaviour)

```ailang
import pkg/sunholo/decisions/decide (decide, Noul, Choice, Score, gate, Act, Escalate, Ungateable, answer)
import std/json (jo, kv, js)

let state = jo([
  kv("compiler_error", js("No instance for Num[string] in scope … Use ++ to concatenate strings.")),
  kv("offending_line", js("\"Config valid: \" + show(valid)")),
  kv("frequency", js("402/1769 compile failures across 42 models"))]);
let qs = [
  { name: "lane", q: Choice("Which PROGRAM.md lane?", [
      { key: "ailang_fix", desc: "Change AILANG: stdlib, error text, defaulting, or the teaching prompt" },
      { key: "motoko_extension", desc: "A harness extension, e.g. an auto-fix hook rewriting + to ++" },
      { key: "core_floor_fix", desc: "A defect in the frozen motoko core" },
      { key: "model_capability", desc: "The model genuinely lacks the ability" }]) },
  { name: "auto_fixable", q: Noul("A deterministic rewrite rule could fix this without an LLM.") }];

match decide("typesafe/jev-1.13", state, qs) {
  Err(e) => …,                                    -- typed: MissingKey | Http | Transport | BadResponse
  Ok(d)  => match answer(d, "lane") {
    Err(e) => …,                                  -- MissingAnswer("lane"): a protocol fault, not a decision
    Ok(a)  => match gate(a, 0.7) {                -- the CALLER owns the threshold (D4)
      Act(lane)       => route(lane),             -- conf ≥ 0.7: act
      Escalate(conf)  => escalateToController(d, conf),   -- conf < 0.7 (spike measured 0.60): a frontier turn decides
      Ungateable(why) => …                        -- cannot happen for a Choice; the compiler makes you say so
    }
  }
}
```

Measured on 2026-09-18: `lane = motoko_extension` with `p = 0.70, conf = 0.60`; `auto_fixable = 0.89`. `gate(…, 0.7)` would have returned `Escalate(0.60)` — the correct behaviour for a 0.59/0.37 split between two defensible lanes — and a caller cannot reach `escalateToController` through a wrong-variant path, because that is a different constructor.

### Example 2: Same questions, LLM control arm (Phase 1 M3 — lives in `tools/decisions/`, NOT in the package)

```ailang
-- tools/decisions/lane_shadow.ail (excerpt). The control arm reuses the package's
-- Question type and buildRequest-style helpers, but produces its OWN record type:
-- an LLM's self-reported probabilities are not a calibrated Answer and must never
-- be able to flow into gate.
import pkg/sunholo/decisions/decide (Question, questionsToJsonSchema)
import std/net (httpRequest)

type LlmAnswer = { name: string, label: string,
                   self_reported: [{ key: string, p: float }],   -- requested per label, normalised to 1
                   source: string }                              -- "llm:<model>" — provenance travels with the row

-- Same 30 s Net deadline as the Jev arm; same transport; only the endpoint and body differ.
let schema = questionsToJsonSchema(qs)   -- for each Choice/Score: an object of per-label numbers in [0,1]; for Noul: one number
match httpRequest("POST", "https://openrouter.ai/api/v1/chat/completions", headers,
                  chatBody(controlModel, state, schema)) {           -- response_format: json_schema, strict
  Ok(resp) => parseLlmAnswers(resp.body, qs),  -- -> Result[[LlmAnswer], DecideError]; renormalises, flags sums ≠ 1 in the row
  Err(e)   => …
}
```

The schema **asks the LLM for a distribution** (mirroring the adapter's `llm_answer_mode="probabilities"` + `normalize_probabilities`), so the arm is not one-hot — but the report labels its spread as *self-reported* and its "confidence" as derived from a self-report, which is itself one of the things M3 measures (does an LLM's self-reported spread predict its own errors as well as Jev's calibrated one predicts Jev's?). `parseLlmAnswers` and `LlmAnswer` are **not exported by the package** (quorum objection, gemini-3-1-pro: an exported adapter would let uncalibrated calls masquerade as `Answer` and defeat every `gate` call).

## Success Criteria

- [ ] `sunholo/decisions` published; `ailang check` clean; fixture tests pass with `--caps ""`
- [ ] Live call from the rig returns three typed answers in ≤ 1 s p95 over 20 calls (banked in the shadow jsonl)
- [ ] `modelreg` test resolves and prices `typesafe/jev-1.13-20260917`; model appears in **no** suite list (grep test)
- [ ] Shadow report committed with all three agreements + confidence-stratified accuracy + cost/latency per arm
- [ ] No code path acts on a Jev answer (grep for `pkg/sunholo/decisions/` importers outside `tools/decisions` and `examples/` is empty)
- [ ] AGENT.md and the package page state, verbatim, the two limitations: non-deterministic distributions; not in AI budget/trace cost in Phase 1
- [ ] CHANGELOG + PROGRAM.md §5b candidate row
- [ ] Documentation updated

## Testing Strategy

- **Pure core, fixture-tested:** `parseAnswers` against banked real responses and malformed bodies — this is where the ADT contract lives, and it needs no network.
- **Mutation check (house rule):** delete the `Some("score")` arm and confirm a fixture test fails; return `Ok` with an empty list on a missing question and confirm the `MissingKey`… test fails.
- **Live smoke (attended only):** `ailang run --caps Net,Env,IO examples/runnable/decide_jev.ail` — not in CI (needs a key); the shadow run is the recurring live check.
- **Pricing:** unit test in `internal/modelreg`; post-deploy check that a new Broadcast span has no `ailang.cost.unpriced`.
- **Gate exhaustiveness:** a fixture test feeds a `ScoreA`, a `NoulA` and a `ChoiceA` to `gate` and asserts `Act`/`Ungateable`/`Escalate` respectively — the three-way contract is the test, not a review note.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Which cheap chat model is the M3 control arm — agent may choose from `dev_models`; must record it in the report header.
- Exact `state` field set per design doc in M3 (problem statement + goals at minimum) — agent may choose; keep < 8k chars.
- Whether `Decision.cost_usd` comes from OpenRouter's `usage.cost` (present in the response) or is recomputed from tokens × registry — agent may choose; prefer the wire value and note it.
- Package test harness (`testing-utils` package vs bare `ailang test`) — agent follows whatever the `ailang-packages` skill prescribes today.

## Non-Goals

**Not attempted in this feature:**
- **Any acting consumer** (routing that fires, a done-gate that blocks, approval triage that decides) — Phase 2, chosen from data. Approvals in particular are the operator's; nothing here ranks or resolves them.
- **A `std/ai` builtin or Go provider** — Phase 2 with its own doc, so its API is shaped by Phase 1's measurement and Conflict Surface (it will touch `internal/effects` + `internal/builtins`).
- **Direct TypeSafe API transport / `TYPESAFE_API_KEY`** — needs a Registry row and a second billing relationship; no benefit until OpenRouter's alpha endpoint proves inadequate.
- **Fine-tuning or feedback to the vendor** — nothing in the doc sends our data anywhere except the OpenRouter call itself (same boundary as every other OpenRouter eval).
- **Multilingual state** — vendor says English primary; our states are English.
- **A `Map` type or new syntax** — distributions are ordered records, matching the rest of `std/json`.

## Timeline

| Day | Work |
|-----|------|
| 0 (done) | Spike: protocol, latency, cost, non-determinism, pure-AILANG implementation, Broadcast visibility |
| 1 AM | M1 package: split pure/shell, error ADT, helpers, fixtures + tests, AGENT.md, publish |
| 1 PM | M2 pricing row + resolve/price test; example + package page; CHANGELOG |
| 2 | M3 shadow runner + LLM arm + report; PROGRAM.md candidate row; this doc's footer |

Realistic total ≈ 1.5–2 days. Phase 2 not scheduled.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| The OpenRouter endpoint is **alpha** and may move or change shape | Transport is one function (`decide`'s shell); `parseAnswers` is fixture-pinned so a shape change fails loudly in tests, not silently in a consumer. Direct-API fallback is a URL/header swap (Key Fact 2). |
| Jev's answers look plausible on demo states but are weak on AILANG-shaped state | That is exactly what M3 measures before anything acts; the LLM arm gives the counterfactual. |
| Consumers bank argmax and lose the distribution | D6 is stated in AGENT.md; `Decision` carries the full answers so the lazy path is also the correct one. |
| A default threshold creeps in "for convenience" | D4; `gate` requires the threshold argument and returns a three-way ADT; code-review line item. |
| The pricing row makes the model selectable by eval tooling | D7; grep test that no suite list contains it; no `agent_*` fields. |
| `std/net` calls are invisible to the AI budget → a runaway shadow loop spends unnoticed | Shadow runner is bounded by the 45-doc list and prints its own running cost from `usage.cost`; Broadcast is the independent check (Key Fact 8). |
| A stalled OpenRouter alpha endpoint hangs a caller | Bounded by the Net effect's 30 s deadline on every call, both arms (Key Fact 15); `Transport("timeout…")` is a typed `Err`, and the shadow runner counts timeouts per arm in the report. |
| An LLM-arm result is mistaken for a calibrated answer | Distinct `LlmAnswer` type in `tools/decisions/`, provenance field, no LLM-facing export in the package; provenance (`Decision.model`, `Decision.id`) is banked with every row (D6) so an LLM-sourced row is identifiable, not merely discouraged (Example 2). |
| The vendor reads our design-doc problem statements | Same exposure as every OpenRouter eval run already has; nothing sensitive is in `state` beyond public design docs. |

## Quorum — status 2026-09-18

Two rounds run (`.ailang/state/mission-quorum/m-ai-decide-system-one-2026-09-18T12-30-13Z.json`, `…T12-34-58Z.json`; $0.18 + $0.19). **Both blocked; every objection accepted and folded in:**

| Round | Reviewer | Objection | Resolution in this doc |
|---|---|---|---|
| 1 | gpt6-astra, oc-glm-5-2 | No bounded wait specified for either arm | Key Fact 15 (Net = 30 s enforced; `std/ai` = **none**, grep-verified); LLM arm moved to the Net transport; `std/ai` gap filed to backlog |
| 1 | gemini-3-1-pro | Exported LLM→`Answer` adapter would be one-hot, confidence 1.0, defeating gates | Adapter removed from the package; LLM arm has its own `LlmAnswer` type and requests per-label probabilities |
| 2 | gpt6-astra | `parseAnswers` is exported, so "no constructor path" was false | Claim withdrawn; provenance is by banked `Decision.model`/`id` (D6), stated as such |
| 2 | gemini-3-1-pro | `actIf → Option` conflates wrong-variant with low-confidence (A11) | Replaced by three-way `Gate = Act \| Escalate \| Ungateable`; `answer`/`expectedScore` return `Result`, not `Option` |
| 2 | oc-glm-5-2 | Reuse-vs-build of `std/ai` not weighed | New Conflict Surface table under Architecture |

Per the design-doc-creator guardrail (re-quorum **once**), a third round is the operator's call, not this session's. Round-2 objections were all answerable in-doc (no external action needed), so the doc is handed over as *revised, not re-verified by quorum*.


Triggers that fired: **#1** (D5 is a freeze item for Mark) and **#4** (Key Facts 1, 2, 4, 12 are about a vendor we do not control).

## Related Documents

**Implemented (inform the design):**
- [design_docs/implemented/v0_5_0/M-GAME-E2-ai-effect.md](design_docs/implemented/v0_5_0/M-GAME-E2-ai-effect.md) (0.34) — the original AI oracle; the seam Phase 2 would extend
- [design_docs/implemented/v0_31_0/m-ai-reasoning-effort.md](design_docs/implemented/v0_31_0/m-ai-reasoning-effort.md) (0.33) — precedent for an optional provider capability interface
- [design_docs/implemented/v0_32_0/m-eval-standard-confidence-gating.md](design_docs/implemented/v0_32_0/m-eval-standard-confidence-gating.md) — a *different* "confidence" (ELO fit over benchmarks), cited to avoid confusion; unrelated mechanism

**Planned (checked for overlap — none is the same topic; best neural score 0.36):**
- [design_docs/planned/20260918_refused_python_refused.md](design_docs/planned/20260918_refused_python_refused.md) (0.36) — a failure-analysis doc; a possible future *consumer* (sub-classifying refusals), not overlap
- [design_docs/planned/m-codex-billing-lane-resolution.md](design_docs/planned/m-codex-billing-lane-resolution.md) (0.35)
- [design_docs/planned/m-astra-vision.md](design_docs/planned/m-astra-vision.md) (0.34)

**Packages:**
- `ailang-packages/packages/motoko-ext-decision-framework` — prompt-patch extension, no LLM; name collision avoided (Key Fact 10)
- `ailang-packages/packages/motoko-ext-abi/types.ail` — hook effect rows (Key Fact 9)

## References

- [Design Axioms](/docs/references/axioms) · [Philosophical Foundations](/docs/references/philosophical-foundations) · [Design Lineage](/docs/references/design-lineage)
- TypeSafe AI — [Introducing System One Models & Jev](https://typesafe.ai/blog/introducing-system-one-models-and-jev) · [docs index](https://docs.typesafe.ai/llms.txt) · [Primitives](https://docs.typesafe.ai/primitives.md) · [Confidence](https://docs.typesafe.ai/confidence.md) · [State](https://docs.typesafe.ai/concepts/state.md)
- OpenRouter — [Jev 1.13 model page](https://openrouter.ai/typesafe/jev-1.13) · [Alpha Decisions API reference](https://openrouter.ai/docs/api/api-reference/alphadecisions/submit-a-decisions-questions-and-answers-request.md) · [launch post](https://x.com/OpenRouter/status/2100744709589316009)
- [`typesafe-ai/system-one-adapter-python`](https://github.com/typesafe-ai/system-one-adapter-python) — same protocol over chat LLMs; source of the byte-exact direct-API fixture
- Spike artifacts: `examples/runnable/decide_jev.ail` (in-repo); curl bodies and the 16 Broadcast spans were session-local and are summarised in Key Facts 1–4, 8

## Future Work

- **Phase 2:** `std/ai.decide` + Go `DecisionProvider` (OpenRouter, then TypeSafe direct) with budget/cap/span accounting; first acting consumer chosen from the M3 report — candidates in measured order: PROGRAM §4 lane pre-router with escalation on low confidence; `on_solver_candidate` done-gate second opinion; `api_error` sub-classification of banked rows; Daneel mail-intent routing (separate repo, separate doc).
- **Speculative fan-out:** the vendor's pattern of asking many cheap questions in one call fits the eval analyzer — one call per failed row asking `{root_cause_lane, prompt_defect, auto_fixable, benchmark_defect}` would replace much of the 25-doc mining pass with a banked distribution per row.
- **AILANG-fix backlog (filed from this doc's quorum):** `std/ai` single-shot calls carry no HTTP deadline (Key Fact 15). Every other oracle in the runtime is bounded; this one should get a `ctx.AI.Timeout` with the same shape as `ctx.Net.Timeout` and a `--ai-timeout` flag routed through `internal/config`. Separate doc; not a Phase 1 dependency because Phase 1 does not use that path.
- **Calibration audit over time:** once rows accumulate under D6, a reliability diagram (confidence vs empirical accuracy) per question type — the vendor's headline claim, checked on our data.

## Where we are (2026-09-18, end of Phase 1)

Phase 1 shipped: `sunholo/decisions` 0.1.1 published (ailang-packages `1ad2f60`), pricing row + tests, the shadow runner and its report. **Shadow result (n=20):** Jev 14/20 vs declared lane; glm-5.3-flash control 12/17 with 3 timeouts; the oracles agreed 15/17; Jev ~33× faster and ~11× cheaper. Four of Jev's six misses are docs where both oracles disagree with the label at ≥ 0.86 — our docs routed AILANG-substrate work as "extension". **Phase 2 gate: premise licensed, acting consumer NOT yet** — first fix the label set / sharpen PROGRAM §4's "extension" wording, re-run, then choose. Full reading: [m-ai-decide-system-one-shadow-report.md](m-ai-decide-system-one-shadow-report.md).

---
*Spike, measurements and this doc: 2026-09-18, attended session. Phase 1 executed the same day.*
