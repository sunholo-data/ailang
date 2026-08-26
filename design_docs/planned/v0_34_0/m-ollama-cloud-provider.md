# M-OLLAMA-CLOUD-PROVIDER: Ollama Cloud as a Second Route for Open-Weight Models

**Status**: Planned — **Phase 0 AND Phase 1 PASSED on the live rig 2026-08-26**; a full motoko agent benchmark runs on Ollama Cloud (V31) (signed in as `m@sunholo.com`,
plan `pro`). The zero-code premise is **confirmed end-to-end**: a cloud-suffixed request through
`localhost:11434/v1` on the live 0.32.14 daemon returned from a model with **zero local copies**
and loaded **nothing** on the GPU (V21), and all three harnesses are measured — not assumed — on
that endpoint (V19). Phases 1-3 are unblocked. **V22 (the quota-exhaustion contract) is still
unmeasured**, so AC8 and the Conflict Surface retry row remain provisional.
**Target**: v0.34.0
**Priority**: P1 (Medium) — no lane is blocked on it; it buys reach and a cost hedge
**Estimated**: ~14 hours across 3 phases (Phase 1 alone is ~1 hour and delivers most of the value)
**Dependencies**: An `ollama signin` on the rig (human action, Mark). No code dependencies.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval/harness infrastructure, not a language change. Most axioms are genuinely neutral;
scoring them 0 is the honest answer, not an evasion.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Model sampling is already nondeterministic; routing the same model to a different host does not add a new class of nondeterminism. Banked rows already record model+provider. |
| A2: Replayability | 0 | Trials bank the same JSONL regardless of route. No trace format change. |
| A3: Effect Legibility | 0 | No AILANG effects involved; this is Go-side harness config. |
| A4: Explicit Authority | +1 | Cloud access is gated on an explicit device-key signin, and the route is named in the model row (`-cloud` suffix). No ambient escalation: an unsigned rig gets a loud 401, not a silent local fallback. |
| A5: Bounded Verification | 0 | No change to type checking or Z3. |
| A6: Safe Concurrency | 0 | Concurrency is bounded by the plan tier and by the existing `--parallel` semaphore. |
| A7: Machines First | 0 | No prompt or output-format change. |
| A8: Minimal Syntax | 0 | No new AILANG syntax. Config-only (`models.yml` rows). |
| A9: Cost Visibility | +1 | Forces the imputed-cost question into the open (D1) rather than letting a subscription lane bank as `0/0` and silently corrupt cost-per-verified-success. Net **improves** cost legibility by making the imputation explicit and labelled. |
| A10: Composability | +1 | Reuses the existing `provider: ollama` path end-to-end — same rows, same harnesses, same banking. No new provider package, no new wire protocol. |
| A11: Structured Failure | 0 | Quota exhaustion surfaces as an HTTP error through the existing `error_category` path. AC7 requires it not be miscategorised as `api_error`. |
| A12: System Boundary | +1 | Makes a boundary crossing explicit that is currently invisible: today `provider: ollama` unambiguously means "on this GPU". After this, the `-cloud` suffix in `api_name` marks the row as leaving the machine. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — routing does not change sampling semantics
- [x] A3 (Effects): No hidden side effects — no AILANG effect surface touched
- [x] A4 (Authority): No ambient access granted — device-key signin is explicit and per-machine; unsigned = 401
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

The rig is one Mac Studio with one GPU. That constraint sets a hard ceiling on which
open-weight models the local lanes can ever run, and it serializes everything that does run.

**Current State:**

- `models.yml` carries **28** `provider: "ollama"` rows (V3) and **0** rows that reach anything
  but `localhost` (V3, V4). Every local lane shares one GPU.
- Agent evals on local/agent-only rows are force-clamped to `--parallel 1`
  ([eval_suite.go:271](../../../cmd/ailang/eval_suite.go#L271)) because concurrent trials thrash
  ollama and crash motoko. Serialization is a deliberate safety measure, not a bug.
- The models that would most sharpen the "local agentic vs cloud frontier" thesis —
  `qwen3.5:397b`, `mistral-large-3:675b`, `deepseek-v4-pro`, `nemotron-3-ultra`,
  `nemotron-3-super` — **do not fit the box at any quantization**. They are unreachable today
  except through OpenRouter, at per-token cost, on a different harness path.
- OpenRouter is currently the only route to open-weight models, so every open-weight datapoint
  carries a single vendor's routing, pricing, and availability as a confound.

**Impact:**

- **Who**: the eval rotation, the mission executor fleet (V1, World, motoko missions), and
  day-to-day agent use.
- **How significant**: Ollama Cloud publishes **19** models (V8), of which roughly 14 already
  exist as OpenRouter rows in `models.yml`. So the honest framing is *not* "new capability" —
  it is **a second price and a second route for models we already run, plus first-ever access
  to five that the rig physically cannot host**. The second route is what makes an
  OpenRouter-vs-Ollama-Cloud comparison possible at all; the five are the new reach.

## Goals

**Primary Goal:** Reach Ollama Cloud models from every existing AILANG harness without adding a
provider package, and without letting a subscription-priced lane corrupt cost-per-verified-success.

**Success Metrics:**

- A cloud model completes an agent-mode benchmark through motoko with **zero** changes to
  `internal/ai/ollama/`, motoko config, pi config, or opencode config (Phase 1 exit gate).
- The tokens-per-usage-unit exchange rate is **measured** at level 1 and level 4, so the
  question "how many trials does $20 buy" has a number instead of a guess.
- No cloud-ollama row banks a trial with `pricing: 0/0` (the local-lane default), and every
  imputed price is labelled as imputed.
- Quota exhaustion is distinguishable from model failure in `error_category`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1**: What imputed per-token prices cloud-ollama rows carry — **and fixing V27 first, without which any price yields `cost_usd = 0`** | The KPI is the mission's scoreboard, and `0/0` is not merely "missing" — `ResolveCostProvenance` maps `0/0` to **`free-local`** (V18), a *positively false* label meaning "on-device, zero marginal cost". A subscription lane banked that way claims to be something it is not. The **mechanism** to do this right already exists (`AuthLaneSubscription` → `list-price-equivalent`, V18), so the decision is narrowed to *which numbers to impute*, not *how to represent them*. | **human (Mark)** | design | **med** |
| **D2**: Whether cloud-ollama rows enter the banked eval rotation at all, or stay executor/day-to-day only | Determines whether an opaque, resetting quota can starve a nightly rotation mid-run and bank a cohort with a hole in it. One model × one core+stretch+frontier pass ≈ 17M tokens (V10, V11). | **human (Mark)** | design | **high** |
| **D3**: Device-key signin vs Bearer API key as the access path | Picks the integration shape. Device key = zero code (V5 is not on the path). Bearer = ~20 LOC across two hardcoded call sites, but is the only way to reach cloud without a local daemon (cloud CI, Cloud Run). | agent | design | low |
| **D4**: Whether the `--parallel 1` local clamp should exempt cloud rows | The clamp exists for GPU thrash, which does not apply to cloud rows — but it keys off `SupportsAgentEval && !SupportsStandardEval`, not on the route ([eval_suite.go:271](../../../cmd/ailang/eval_suite.go#L271)), so a cloud row would be wrongly serialized. Exempting it trades rotation wall-clock against the plan's concurrency cap. | agent | compile | med |
| **D5**: Plan tier to standardize on | Free = 1 concurrent model, Pro = 3, Max = 10; **Max is paused for new subscriptions** (V9), so Pro is the practical ceiling today. | **human (Mark)** | design | low |
| **D6**: Route marked by model-name suffix vs an explicit `route:`/`endpoint:` field | Raised by quorum review (gpt5-6-sol): inferring authority and concurrency policy from a *name string* is implicit where A4/A12 want explicit. **There is no existing route/endpoint/base_url field in `models.yml` to reuse** — 0 such keys against a 120-key `provider:` control (V13) — so this is add-vs-infer, not reuse-vs-duplicate. The suffix is however what ollama's own server parses (V2), so a separate field would need syncing with it and could disagree. | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — imputed per-token prices for subscription-metered rows. *Recommendation: impute
      from the OpenRouter twin and let the EXISTING `AuthLaneSubscription` → `list-price-equivalent`
      path label it (V18) — that constant's own doc comment describes this exact case ("the run went
      through a subscription/OAuth lane and was never billed"), and 18 of 30 banked trials already
      carry it. Most cloud models have a twin; the five rig-infeasible ones do not and need an
      explicit call from you.*
- [ ] **D2** — banked rotation vs executor-only. *Recommendation: executor/day-to-day only until
      the Phase 2 calibration produces an exchange rate. This is a sequencing recommendation, not
      a permanent exclusion.*

D3/D4/D5 are not freeze items: D3 and D4 are agent-resolvable and D5 is a low-cost human pick
that does not block Phase 1.

## Solution Design

### Overview

**The integration is a naming convention, not a code path.** Ollama's local server proxies cloud
models transparently, and it does so on the OpenAI-compatible route — not just `/api/*`:

```
routes.go:1916  r.POST("/v1/chat/completions", ..., cloudPassthroughMiddleware(...), ...)
```

Cloud selection is a **suffix parsed off the model name by string manipulation** — no pull, no
manifest, no weights on disk (V2, `internal/modelref/modelref.go:99-118`):

- `kimi-k3:cloud` → base `kimi-k3`, source cloud
- `gpt-oss:120b-cloud` → base `gpt-oss:120b`, source cloud

AILANG's ollama provider reaches `localhost:11434/v1` (`step.go:345`, V5), and motoko, pi and
opencode are each **measured** on the same endpoint (**V19** — this was repo lore until Phase 0;
quorum round 1 was right to refuse it as an assumption). A cloud model is therefore reachable by
changing one string in a `models.yml` row — **confirmed live in V21**, including a 397B model that
fits nowhere on this hardware.

### Architecture

**Components:**

1. **Access path (device key)**: `ollama signin` on the rig registers the machine's ed25519 key.
   The vendored Go client auto-signs any request whose host is `ollama.com`
   (`api/client.go:119`), and the local server signs its own proxied cloud requests. No key
   plumbing in AILANG. **The rig is currently signed out** (V6) — this is the one human
   prerequisite.

2. **Model rows**: new `models.yml` entries with `provider: "ollama"` and an `api_name` carrying
   the `-cloud`/`:cloud` suffix. Same `agent_cli`, same `motoko_profile`, same banking. The
   suffix is the only thing marking the row as leaving the machine, which is why AC2 requires it
   be visible in banked output.

3. **Cost treatment (D1)**: an imputed, explicitly-labelled price. This is the only component
   that is genuinely new logic rather than configuration.

4. **Quota gauge**: `GET https://ollama.com/api/usage` exists and is auth-gated (V7: 401, against
   a 404 control) — it is **undocumented**, absent from both the published docs and the vendored
   `docs/` tree. It is the difference between "we cannot safely put evals on this" and "we can,
   with a burn-down gauge". Its response body is **unknown** — nobody has authenticated against
   it yet — so Phase 2 must inspect it before anything depends on its shape.

**Not built**: a provider package. `internal/ai/ollama/` reaches cloud unchanged — V2 in vendored
0.32.15 source, and V21 against the live 0.32.14 daemon.
Writing `internal/ai/ollamacloud/` would duplicate the streaming, idle-timeout, temperature and
max-tokens work that M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT and friends already landed.

### Implementation Plan

**Phase 0: Discovery spike — falsify the foundational premise** (~2 hours, GATES EVERYTHING)
- [x] Record the **effective** base URL + model string actually used by motoko, pi and opencode
      (read the live configs; do not infer from `models.yml`) → **V19 PASSED**
- [~] **While still signed out**, send a cloud-suffixed request to the live 0.32.14 daemon →
      **V20 PARTIAL.** The no-fallback half passed (typed `not_found_error`, nothing on the GPU);
      the signed-out half was overtaken by the signin and now rests on V7's direct 401. Re-testing
      would mean `ollama signout` on a shared rig — not worth disrupting concurrent evals
- [x] `ollama signin` (human: Mark) → `POST /api/me` returns `m@sunholo.com`, plan `pro` (**V24**)
- [x] Cloud request through the shared `/v1` endpoint; remote execution proven by a zero-local-copy
      model and an untouched `ollama ps` → **V21 PASSED**. *Scope note*: this verifies the endpoint
      **contract** all three harnesses share (V19), not three separate end-to-end agent runs —
      those are Phase 1. Stated so the row is not read as more than it is
- [x] **Gate: PASSED.** The zero-code premise held. Two corrections fell out of actually running
      it — AC2 was unachievable as written (the proxy strips the suffix; the bank stores the row
      key) and AC3 was mis-scoped (`/api/usage` is not locally proxied). Both are fixed above

**Phase 1: Reach it** (~1 hour, gated on Phase 0)
- [ ] Add ONE `models.yml` row: `gpt-oss:20b-cloud` (usage level 1, cheapest probe)
- [ ] Run one smoke benchmark through motoko; confirm zero diffs outside `models.yml`

**Phase 2: Measure the quota AND the exhaustion contract** (~4 hours)
- [ ] Authenticate against `GET /api/usage` and record what the body actually contains
- [ ] Calibration run: read usage → run one benchmark of known token volume against a **level 1**
      model → re-read usage. Derive tokens-per-unit.
- [ ] Repeat against a **level 4** model (`deepseek-v4-pro:cloud`) to derive the weight multiplier
- [ ] **Observe the actual exhaustion response** — deliberately exhaust a low-level model in a tight
      loop and record the exact HTTP status and body → **V22**. AC7/AC8 and the Conflict Surface
      retry row must then be rewritten to match the observed contract
- [ ] Publish the exchange rate and the implied trials-per-$20 into this doc, and feed D1/D2

**Phase 3: Wire it in** (~10 hours, **gated on D1 + D2**)
- [ ] Implement the D1 pricing treatment with an explicit imputed-cost label
- [ ] Add rows for the five rig-infeasible models (`qwen3.5:397b`, `mistral-large-3:675b`,
      `deepseek-v4-pro`, `nemotron-3-ultra`, `nemotron-3-super`)
- [ ] Resolve D4: exempt cloud rows from the single-GPU `--parallel` clamp, bounded by the
      plan's concurrency cap
- [ ] Ensure quota exhaustion maps to a distinct `error_category`, not `api_error`
- [ ] Optional (D3): plumb `OLLAMA_API_KEY` through the two hardcoded dummy-key sites

### Files to Modify/Create

**New files:**
- `internal/eval_harness/ollama_cloud_pricing.go` — imputed-cost treatment per D1, ~120 LOC.
  Exact shape depends on the D1 outcome.

**Modified files:**
- `internal/eval_harness/models.yml` — new cloud rows, ~30 LOC/row × 6 rows
- `cmd/ailang/eval_suite.go` — D4, narrow the `--parallel 1` clamp so it keys on the route and
  not only on `SupportsAgentEval && !SupportsStandardEval`, ~15 LOC
- `internal/ai/ollama/step.go` — D3 only, replace the hardcoded `"ollama"` dummy key at line 345, ~10 LOC
- `internal/ai/ollama/streamstep.go` — D3 only, same at line 251, ~10 LOC
- `docs/docs/guides/evaluation.md` — document the `-cloud` suffix convention and the signin prerequisite

## Conflict Surface

<!-- Added in quorum revision round 1 (gpt5-6-sol reject). The skill mandates this section for
     parser/typechecker/codegen changes, which this is not — but the objection's substance stands:
     this design overlaps live pricing, banking, concurrency, retry and error-classification
     machinery, and "no existing field to collide with" is a claim that must be measured, not
     assumed. Every row's evidence is a Verification Log entry. -->

This design adds **no new syntax and no new provider package**; its entire conflict surface is
with existing eval-harness machinery it must either reuse or deliberately override.

| Existing mechanism | Verified current behavior (evidence) | Proposed overlap | Decision | Rejected alternative | Invariant-preserving test |
|---|---|---|---|---|---|
| `models.yml` route metadata | **No** `route`/`endpoint`/`base_url`/`host` key exists — 0 hits vs a 120-key `provider:` control (**V13**). Endpoint comes solely from `OLLAMA_HOST`, defaulting to `localhost:11434` (`client.go:43-57`) | Cloud rows need a way to say "this leaves the machine" | **Infer from the `-cloud` name suffix** — it is what ollama's own server parses (**V2**), so it cannot drift from the actual route | An explicit `route:` field: rejected because a second source of truth could *disagree* with the suffix ollama actually honours. Revisit under **D6** | A cloud-suffixed row and a local row resolve to different routes; banked `model` retains the suffix (**AC2**) |
| Pricing schema `pricing.{input,output}_per_1k` (`models.go:41,81`) | `ResolvedMaxCostUSD` derives the cost gate as `input×64 + output×32`, capped $0.50 (`models.go:140-146`). `0/0` yields **no cost gate** | Cloud rows must not inherit the local `0/0` | **Impute non-zero prices (D1)**; the existing WORK-gate `budgets.max_tokens_per_bench` still applies as it does for local rows | Leaving `0/0` and adding a parallel "subscription cost" field: rejected — forks the cost axis and breaks comparability | A cloud row resolves a non-zero cost gate; unit test asserts no cloud row can resolve `0/0` |
| Cost provenance labelling (`internal/executor/cost.go:111-142`) | `ResolveCostProvenance` returns **`free-local`** whenever both prices are 0 (**V18**). `CostListPriceEquivalent` exists and its doc comment reads *"the run went through a subscription/OAuth lane and was never billed"*; `AuthLaneSubscription` exists; 18/30 banked trials already carry `list-price-equivalent` | Cloud rows are a subscription lane | **Reuse `AuthLaneSubscription` → `list-price-equivalent` unchanged** | A new provenance constant: rejected — the existing one describes this case verbatim, and adding a synonym would split the same concept across two labels | A cloud trial banks `cost_provenance: "list-price-equivalent"`, never `free-local` |
| `error_category` taxonomy (`metrics.go:177-207`) | `quota_exhausted` ("Provider account/key cap reached") and `rate_limit` ("429 — transient") **both already exist** as distinct categories (**V14**) | Ollama Cloud quota exhaustion needs a category | **Reuse `quota_exhausted`**; do **not** add a category | Inventing `ollama_quota`: rejected — duplicates an existing concept. Falling back to `api_error`: rejected — per CLAUDE.md that means "cause unknown" | A quota-exhausted trial banks `quota_exhausted`, not `rate_limit` and not `api_error` (**AC7**) |
| Retry predicate `isRetryableError` (`ai_agent.go:204-231`) | Returns **true** on any error whose string contains `"429"` — a substring match, with no distinction between a transient rate-limit and an exhausted plan (**V15**) | Ollama Cloud signals over-quota through HTTP status | **Override**: quota exhaustion must be non-retryable. This is the design's one behavioral change to shared code | Leaving it: rejected — it would retry into a spent bucket, the documented codex-probe failure mode, burning wall-clock on a bucket that cannot recover within the window | Test: a 429 carrying quota-exhaustion semantics returns `false` from the retry predicate, while a transient 429 still returns `true` (**AC8**) |
| Single-GPU clamp (`eval_suite.go:271-275`) | When `--agent` and `--parallel > 1`, any model matching `SupportsAgentEval(m) && !SupportsStandardEval(m)` forces `maxConcurrent = 1` and `break`s (**V12**). It keys on eval-mode support, **not** on route | Cloud rows are not GPU-bound and should not be serialized | **Narrow the predicate (D4)** so it excludes cloud-routed rows | Deleting the clamp: rejected — genuinely local rows still need it, and removing it re-creates the 2026-06-22 GPU-contention loss | Both directions tested: a cloud row is not clamped; a local row still is |
| Ollama request deadlines (`step.go:26-64`, `streamstep.go`, `idlereader.go`) | `ollamaV1Timeout()` bounds the `/v1` call; under streaming, `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` is a **mandatory** hard deadline that rejects `0`/negative at construction rather than meaning unbounded (CLAUDE.md) | Over-cap cloud requests are **queued, not rejected** (**V9**) — a queue is an unbounded wait unless capped | **Reuse the existing deadline machinery unchanged** — it already forbids the unbounded case, which is precisely the guarantee a queue needs | Adding a cloud-specific timeout: rejected — a second timeout knob to keep in sync, when the existing one already has the right semantics | A queued cloud request fails at the configured deadline with `timeout`, never hangs |

**What deliberately changes**: `isRetryableError` gains a quota-exhaustion carve-out, and the
single-GPU clamp narrows. Both are intentional and both are tested in *both* directions so the
existing invariant (local rows serialized, transient 429s retried) is preserved.

## Examples

### Example 1: Reaching a model the rig cannot host

**Before** — `qwen3.5:397b` is unreachable on the local route; only an OpenRouter row exists:

```yaml
or-qwen3-5-35b-a3b:
  api_name: "qwen/qwen3.5-35b-a3b-20260224"   # 35B, the largest that fits
  provider: "openrouter"
```

**After** — the 397B reachable through the identical local path:

```yaml
motoko-cloud-qwen3-5-397b:
  api_name: "qwen3.5:397b-cloud"    # the suffix is the whole integration
  provider: "ollama"                # unchanged provider
  agent_cli: "motoko"
  motoko_profile: "ollama"          # unchanged profile -> localhost:11434/v1
```

### Example 2: The pricing hazard D1 exists to prevent

Copying an existing local row is the obvious move and is **wrong**:

```yaml
pricing:
  input_per_1k: 0.0     # correct for local (the GPU is already paid for)
  output_per_1k: 0.0    # WRONG for cloud — subscription-metered, not free
```

Banked that way, a cloud row reports `cost_usd = 0` and outranks every paid model on
cost-per-verified-success by construction. The failure is silent: nothing errors, the number is
just wrong, and it is wrong in the direction that flatters the lane.

## Success Criteria

- [ ] **AC1**: A cloud model completes an agent-mode benchmark via motoko with zero diffs outside
      `models.yml` (`git diff --stat` shows only that file)
- [ ] **AC2** *(corrected by Phase 0 — the original was unachievable as written)*: route must be
      recoverable from banked data alone. The `-cloud` suffix **cannot** carry it: the proxy
      rewrites the model field to the base name before dispatch, so the response says
      `qwen3.5:397b` for a `qwen3.5:397b-cloud` request (**V21**), and the banked `model` is the
      **models.yml row key** (`agent.friendlyName`, `repair.go:49`) which never contains
      `api_name` at all (**V17**, **V23**). Route identity must therefore come from the row-key
      naming convention (e.g. `motoko-cloud-*`) or an explicit banked field — decide under **D6**
- [x] **AC3 SATISFIED** — `/api/usage` authenticates and its body is documented in **V26**. Two
      scope corrections came out of it: it is **not proxied by the local daemon** (404 at
      `localhost:11434`, **V24**) so it needs a direct Bearer call, and **it reports no
      denominator** — see V26 and the amended risk row
- [ ] **AC4**: Tokens-per-usage-unit measured at level 1 **and** level 4; both recorded here
- [ ] **AC5**: No cloud row banks with `pricing: 0/0`; imputed prices carry an explicit label
- [ ] **AC6**: Cloud rows are not force-clamped to `--parallel 1` (D4), and honour the tier
      concurrency cap
- [ ] **AC7**: A quota-exhausted request banks the **existing** `error_category` `quota_exhausted`
      (V14) — not `rate_limit` (that constant means "429 — transient"), not `api_error` (per
      CLAUDE.md that means "cause unknown" and would hide this), and no new category is added
- [ ] **AC8**: `isRetryableError` returns **false** for quota exhaustion and **true** for a
      transient rate-limit. Today it returns true for any error string containing `"429"` (V15), so
      without this the harness retries into a spent bucket. **The discriminator must be written
      against the response shape observed in V22, not assumed** — exhaustion may be 429, 402, 403
      or a 200 payload, and a carve-out keyed on the wrong shape is worse than none
- [x] **AC9 DONE** *(added by Phase 1; scope narrowed by V32 — **standard mode only**, agent mode already reports correctly)*: the ollama provider reports real token counts. `client.go:203-205`
      and `step.go:495-497` hardcode 0 (**V27**); the `/v1` response already carries `usage` and the
      native path carries `PromptEvalCount`/`EvalCount`. Until this lands, imputed pricing cannot
      produce a non-zero `cost_usd`, and the `budgets.max_tokens_per_bench` WORK gate has nothing
      to count on this path. **Affects every ollama row, not just cloud** — it was masked because
      they are all priced `0/0`
- [ ] All tests passing
- [ ] Documentation updated (`docs/docs/guides/evaluation.md`)
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- D1 pricing treatment: an imputed price is produced and labelled; a cloud row can never silently
  resolve to `0/0`
- D4 clamp logic: a cloud-suffixed row is not caught by the single-GPU serialization guard, while
  a genuinely local row still is (both directions — the guard must not be simply removed)
- D3 (if taken): the API key reaches the `/v1` client; absence produces a typed error rather than
  a silent dummy-key 401

**Integration tests:**
- One live smoke benchmark against a level-1 cloud model through motoko
- Quota-exhaustion path produces the AC7 category. The unit test stubs the response, but the
  **stub must be built from the shape observed in V22** — quorum round 1 (gemini-3-1-pro) correctly
  refused an earlier version of this doc that designed the carve-out against an *imagined* 429

**Manual testing:**
- `ollama signin`, then `POST /api/me` returns an account (currently `unauthorized`, V6)
- The Phase 2 calibration run (inherently manual — it reads an external dashboard/endpoint)

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Naming scheme for cloud rows** (`motoko-cloud-*` vs `oc-*` vs `motoko-oc-*`) — agent may
  choose; should be consistent with the existing `motoko-or-*` OpenRouter convention
- **Which of the 14 duplicate models (if any) get cloud rows** — agent may choose, guided by the
  Phase 2 exchange rate. The 5 rig-infeasible ones are in scope by default; the duplicates are
  only worth a row if they enable an OpenRouter-vs-Ollama-Cloud A/B worth running
- **Whether the quota gauge is a CLI subcommand, a library call, or a script** — agent may
  choose; nothing in this design depends on the surface
- ~~Retry/backoff behaviour on quota exhaustion~~ — **no longer deferred.** Quorum review
  surfaced that `isRetryableError` already returns true on any `"429"` substring (V15), so
  "agent may choose" would have silently meant "retries into a spent bucket". Now a mandatory
  design element: see the Conflict Surface retry row and **AC8**

## Non-Goals

- **A separate `internal/ai/ollamacloud/` provider package** — the existing ollama provider
  already reaches cloud unchanged (V1, V2); a second package would fork the streaming and
  timeout work for no benefit
- **Replacing OpenRouter** — this adds a second route to compare against, it does not retire the
  first. Retiring a route would destroy the comparison this doc is partly justified by
- **Retiring any local GPU lane** — local remains the zero-marginal-cost floor and the honest
  "on-device" datapoint. Cloud is additive
- **Ollama Cloud for the AILANG registry, docs search, or μRAG embeddings** — embeddings stay
  local (a cloud round-trip per embed would be slower and quota-metered for no gain)
- **Solving the "50x of an undisclosed base" problem analytically** — it is not solvable from
  published data (V9). Phase 2 measures it instead

## Timeline

**Week 1** (~4 hours):
- Phase 1: signin, one row, one smoke benchmark (~1h)
- Phase 2: usage endpoint inspection + calibration at levels 1 and 4 (~3h)
- **Gate**: report the exchange rate to Mark; resolve D1 + D2

**Week 2** (~10 hours):
- Phase 3: pricing treatment, model rows, clamp exemption, error category
- Documentation and tests

**Total: ~14 hours across 2 weeks**, but Phase 1 is ~1 hour and delivers the executor/day-to-day
value on its own. Phases 2-3 exist to make the eval lane *safe*, not to make cloud models
*reachable*.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Quota exhausts mid-rotation, banking a cohort with a hole | **High** | D2 keeps cloud rows out of banked rotations until the exchange rate is known. AC7 makes exhaustion legible rather than a mystery `api_error` |
| Imputed pricing corrupts cost-per-verified-success | **High** | D1 is a design-freeze item and needs Mark. Precedent exists (list-price-equivalent, ratified 2026-07-28), so this is applying a decision, not inventing one |
| The models we want are the ones that drain quota fastest | Med | Structural, not fixable: `deepseek-v4-pro` is explicitly named a level-4 "extra heavy" model (V9). Phase 2 measures the level-4 multiplier specifically so this is priced in rather than discovered mid-rotation |
| `/api/usage` is undocumented and may change or vanish without notice | Med | Treat it as best-effort. Do not make banking correctness depend on it — it is a gauge, not a gate. V26 records the observed shape so drift is detectable |
| **The quota gauge has no denominator** — corrected 2026-08-26, this was overstated in the original draft | **High** | V26: `/api/usage` returns consumption (`usage`, per-model `request_count`) but publishes **no limit**, so "% remaining" and any pre-flight "refuse to start if quota is low" check are **not implementable from it**. What IS implementable: consumption *rate* and a trip-wire on absolute request counts. The earlier claim that this endpoint makes eval rotations safe does not survive contact with the body — it makes them *observable*, which is less. D2 should weigh that |
| Pro's 3-model concurrency cap slows rotations | Low | Requests over the cap are **queued, not rejected** (V9), so this costs wall-clock and not failures. Agent evals are already clamped to `--parallel 1` today, so this is not a regression |
| Ollama Cloud availability/routing differs from OpenRouter for the same model, confounding A/B | Med | Keep the OpenRouter rows. Any cross-route comparison is a paired run, never a pooled aggregate — the same discipline the three local-model baseline boundaries already require |
| Harness retries into a spent quota bucket | **High** | `isRetryableError` retries any `"429"` today (V15). AC8 makes the carve-out a tested requirement rather than implementer latitude. Surfaced by quorum review, not by design — it would have shipped |
| Rig ollama (0.32.14, V6) drifts from the vendored lib (0.32.15) | Low | Both carry the cloud proxy. Verify behaviourally after any ollama upgrade rather than trusting the version number |

## Verification Log

Every row measured 2026-08-26 at `b6fb1dc42dfd8ac55e1b291a3a1a1bf5818b1f44`, on
`Voights-Mac-Studio.local`, against vendored `github.com/ollama/ollama@v0.32.15` (per `go.mod`)
and the live rig daemon. Negative-existence claims are paired with a known-positive control in
the same call. **No `ailang check` rows appear because this design makes no claim about the
AILANG language** — it is harness/config infrastructure. A **Conflict Surface** section IS
present: the changed files are not in the parser/typechecker/codegen set that mandates one, but
quorum round 1 (gpt5-6-sol) correctly held that the design overlaps live pricing, banking,
concurrency, retry and error-classification machinery, so the section was added and rows
V12-V18 measured to support it.

**V1-V18** measured at authoring. **V19-V21, V23-V25** measured on the live rig 2026-08-26 after
`ollama signin` (account `m@sunholo.com`, plan `pro`) — these are the Phase 0 spike that quorum
round 1 required. **V20 is PARTIAL** and **V22 remains PENDING**; both say so in their own rows
rather than being quietly dropped.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | **NEG**: the vendored Go client never reads `OLLAMA_API_KEY`; it signs with the device key instead | `grep -rc OLLAMA_API_KEY $M/api/ $M/envconfig/` · control `grep -rl OLLAMA_API_KEY $M \| wc -l` | **0** hits in `api/`+`envconfig/`; control **9** files module-wide (docs, `cmd/launch`, tests). Auth is `auth.Sign` gated on `envconfig.UseAuth() \|\| c.base.Hostname() == "ollama.com"` (`api/client.go:119`) |
| V2 | **POS**: local `/v1/chat/completions` carries cloud passthrough; cloud selection is a pure suffix parse | `grep -n cloudPassthroughMiddleware $M/server/routes.go`; read `$M/internal/modelref/modelref.go:99-118` · **NEG** control `grep -ric "quota\|usage\|ratelimit" $M/server/cloud_proxy.go` vs `grep -c cloudProxyBaseURL` | `routes.go:1916` wraps `/v1/chat/completions` (also `/v1/completions`, `/v1/embeddings`, `/v1/responses`, `/v1/messages`). `parseSourceSuffix` matches `:cloud` and `-cloud` by `strings.LastIndex`, no I/O. Quota accounting in cloud_proxy.go: **0** hits; control **7** — the proxy reports no usage |
| V3 | **NEG**: no cloud-ollama rows exist in `models.yml` today | `grep -c 'api_name:.*cloud' internal/eval_harness/models.yml` · control `grep -c 'provider: "ollama"'` | **0** cloud rows; control **28** ollama rows |
| V4 | **NEG**: AILANG has no ollama.com or cloud-suffix handling of its own | `grep -rn 'ollama\.com\|"cloud"' internal/ai/ cmd/ailang/ \| grep -v _test` · control `grep -rn 11434 internal/ cmd/ \| grep -v _test \| wc -l` | **3** hits, all unrelated (`coordinator_lifecycle.go:158` COORDINATOR_MODE, `chains_post.go:88,173` observatory spool); **0** ollama.com references. Control **33** localhost:11434 hits — the local path is pervasive, the cloud path absent |
| V5 | **POS**: the `/v1` delegation hardcodes a dummy key at exactly two sites (blocks direct-to-cloud; irrelevant to the device-key path) | `grep -n 'openai.NewClient("ollama"' internal/ai/ollama/*.go \| grep -v _test` | `step.go:345`, `streamstep.go:251` — two sites, both literal `"ollama"` |
| V6 | **POS**: the rig runs 0.32.14 and is **signed out** | `curl -s localhost:11434/api/version`; `curl -s -XPOST localhost:11434/api/me` | `{"version":"0.32.14"}`; `{"error":"unauthorized","signin_url":"…"}` (URL redacted). Vendored lib is 0.32.15 per `go.mod:25` — both carry the proxy |
| V7 | **POS**: `/v1/chat/completions` and an **undocumented** `/api/usage` exist on ollama.com and are auth-gated | four `curl -s -o /dev/null -w '%{http_code}'` probes incl. a bogus-route control | `POST /v1/chat/completions` **401**; `GET /api/usage` **401**; `GET /v1/models` **200**; control `GET /api/definitely-not-real` **404**. 401≠404 ⇒ routes exist. Body of `/api/usage` **NOT observed** — unauthenticated |
| V8 | **POS**: catalogue size, and the 5 rig-infeasible models | `curl -s https://ollama.com/v1/models \| python3 -c "…len(data)…"` | **19** models. Cross-referencing `models.yml`: ~14 have an OpenRouter twin; `qwen3.5:397b`, `mistral-large-3:675b`, `deepseek-v4-pro:0813`, `nemotron-3-ultra`, `nemotron-3-super` do not fit the rig |
| V9 | **NEG**: no absolute quota figure is published for Pro | WebFetch of [ollama.com/pricing](https://ollama.com/pricing) | Pro defined only as *"50x more than Free"*; Free only as *"light usage"*. Published: metering on model weight × (input, cached input, output) tokens; **usage level 1-4** with `gpt-oss:20b` level 1 and **`deepseek-v4-pro` level 4**; session limit resets **5h**, weekly **7d**; email at **90%**; over-concurrency requests **queued, not rejected**; Free/Pro/Max concurrency **1/3/10**; **Max paused for new subscriptions**. No absolute token number anywhere |
| V10 | **POS**: measured per-trial token demand of an agentic benchmark | aggregate over 30 banked trials in `eval_results/baseline_v1_0/agent/*.json` | Median total tokens/trial: sonnet-4-6 **369,020**; haiku-4-5 **317,120**; opencode-or-deepseek-v4-flash **284,843**; gpt5-6-luna **245,277**; opencode-or-glm-5-2 **126,166**. Max observed **600,747**. ~99% is *input* — the agentic loop re-sends context each turn |
| V11 | **POS**: benchmark counts per tier, for the full-pass estimate | `grep -h '^tier:' benchmarks/*.yml \| sort \| uniq -c` (93 files) | core **23**, stretch **25**, frontier **8** ⇒ 56 for the post-release default. At V10's ~300k median ⇒ **≈17M tokens per model per pass** |
| V12 | **POS**: the single-GPU clamp exists, keys on eval-mode support not route, and hard-sets `maxConcurrent = 1` *(added in quorum round 1 — gemini-3-1-pro correctly flagged this premise as driving D4/AC6 while unlogged)* | `grep -n -A3 SupportsAgentEval cmd/ailang/eval_suite.go` | `eval_suite.go:272`: `if GlobalModelsConfig.SupportsAgentEval(m) && !GlobalModelsConfig.SupportsStandardEval(m)` → `:274 *maxConcurrent = 1` → `:275 break`, guarded at `:271` by `*agent && *maxConcurrent > 1`. Clamp block is **271-275**; line 270 is the comment (the doc's original `:270` cite is corrected) |
| V13 | **NEG**: `models.yml` has no route/endpoint field to reuse or collide with | `grep -cE '^\s+(base_url\|endpoint\|route\|api_base\|host):' internal/eval_harness/models.yml` · control `grep -cE '^\s+provider:'` | **0** route-ish keys; control **120** `provider:` keys. Endpoint resolves solely from `OLLAMA_HOST` (default `localhost:11434`, `client.go:43-57`) |
| V14 | **POS**: `quota_exhausted` and `rate_limit` already exist as distinct categories — no new category is needed | `sed -n '177,207p' internal/eval_harness/metrics.go` | `ErrorCategoryQuotaExhausted = "quota_exhausted"` // *"Provider account/key cap reached (e.g. OpenRouter monthly limit)"*; `ErrorCategoryRateLimit = "rate_limit"` // *"429 — transient, distinct from monthly cap"*; `ErrorCategoryAPI = "api_error"` // *"fallback when no more specific cause is known"* |
| V15 | **POS**: the retry predicate would retry a quota exhaustion — the design's one shared-code override | `sed -n '204,231p' internal/eval_harness/ai_agent.go` | `isRetryableError` does `strings.Contains(errStr, "429") → return true`, a **substring** match with no transient-vs-exhausted distinction. Called from `GenerateWithRetry` (`ai_agent.go:172-176`) looping to `cfg.MaxRetries` |
| V16 | **POS**: pricing schema and the cost gate derived from it | `grep -n 'input_per_1k' internal/eval_harness/models.go`; `sed -n '140,146p'` same file | `models.go:41,81`: `InputPer1K float64 \`yaml:"input_per_1k"\``. `ResolvedMaxCostUSD` = `Budgets.MaxCostUSD` if > 0, else `input×64 + output×32` capped at `$0.50` ⇒ `0/0` gives **no cost gate** |
| V17 | **POS**: banked rows already carry `model` and `cost_provenance`, so AC2 needs no schema change | read one row of `eval_results/baseline_v1_0/agent/*.json` | keys include `model`, `cost_usd`, `cost_provenance`, `model_family`; e.g. `model='claude-haiku-4-5'`, `cost_usd=0.1079` |
| V18 | **POS**: `0/0` does not merely omit a label — it produces the **positively false** `free-local`; and the correct subscription label already exists and is already in use | `sed -n '415,428p' internal/eval_harness/metrics.go`; `sed -n '108,145p' internal/executor/cost.go`; tally `cost_provenance` over the 30 banked trials | `ResolveCostProvenance`: `Pricing.InputTokenCost == 0 && OutputTokenCost == 0 → CostFreeLocal` (`cost.go:140-142`); `standardModeCostProvenance` mirrors it (`metrics.go:423`). `CostListPriceEquivalent = "list-price-equivalent"` documented as *"the run went through a subscription/OAuth lane and was never billed"*; `AuthLaneSubscription` exists. Banked tally: **list-price-equivalent 18, metered 11, absent 1** |
| V19 | **POS** — all three harnesses are on `localhost:11434/v1`, measured from their live configs (was repo lore; quorum round 1 correctly refused it) | read `~/.config/opencode/opencode.jsonc`, `/Users/voightkampff/dev/mk-ast/.motoko/config/ollama/config.json` (the CANONICAL worktree per MOTOKO.md), `~/.pi/agent/models.json` | **opencode**: `"baseURL": "http://localhost:11434/v1"`. **motoko** (`ollama` profile): `"http://localhost:11434/v1"`, model form `ollama/qwen3.5:35b-a3b-mxfp8`. **pi**: `"baseUrl": "http://localhost:11434/v1"`, `"apiKey": "ollama"` |
| V20 | **PARTIAL** — the no-silent-fallback half of A4 holds; the signed-out half is now **untestable without signing out**, which would disrupt a shared rig | bogus `:cloud` model vs bogus local model against the live daemon, with `ollama ps` before/after · plus V7 (`POST ollama.com/v1/chat/completions` unauthenticated → **401**) | Both return a typed `{"error":{"type":"not_found_error"}}` and load **nothing** on the GPU — no pull, no local execution, no fallback. **Caveat, stated not hidden**: signin happened before this row could be measured, so "signed-out cloud request fails loudly" rests on V7 (a direct 401) rather than a proxy-path observation |
| V21 | **POS** — the foundational zero-code premise, confirmed live: cloud models answer through the local `/v1` endpoint, prove remote execution, and include one that fits nowhere on this hardware | `POST localhost:11434/v1/chat/completions` with `gpt-oss:20b-cloud` then `qwen3.5:397b-cloud`; `ollama list \| grep -c gpt-oss` (control: `grep -c qwen`); `ollama ps` | `gpt-oss:20b-cloud` → 200 in **0.72s**; `qwen3.5:397b-cloud` → `content: "OK"`, `finish_reason: stop`. Local copies of gpt-oss: **0** (control: **5** qwen). `ollama ps` unchanged throughout — only the *running eval's* `qwen3.8:27b-mxfp8` + `embeddinggemma`; the cloud calls loaded nothing and did not disturb the in-flight eval. **Response `model` is the BASE name** (`qwen3.5:397b`), suffix stripped — see AC2 |
| V23 | **POS** — the banked model string is the models.yml **row key**, not `api_name`, so no `api_name` suffix can ever reach the bank | `grep -n 'NewRunMetrics(' internal/eval_harness/*.go` | `repair.go:49` passes `r.agent.friendlyName`. Corroborated by V17: banked `model='claude-haiku-4-5'` (the key) while `models.yml:682` `api_name: "claude-haiku-4-5-20251001"` |
| V24 | **NEG** — `/api/usage` is **not** in the local daemon's route table, so it cannot ride the device key like inference does | `curl -s -o /dev/null -w '%{http_code}' localhost:11434/api/usage` · control `POST localhost:11434/api/me` | `/api/usage` → **404** (body `404 page not found`); control `/api/me` → **200** `{"email":"m@sunholo.com","plan":"pro",...}`. Confirms signin took effect AND that usage needs a direct Bearer call to ollama.com |
| V25 | **OBS** — reasoning models burn the output budget before emitting content, so cloud rows need the same max-tokens floor the local ones carry | the two V21 calls | `gpt-oss:20b-cloud` at `max_tokens: 20`: `content: ""`, `reasoning` populated, `finish_reason: "length"` — all 20 tokens went to reasoning. `qwen3.5:397b-cloud` at `max_tokens: 2000` spent **336 completion tokens** to answer `"OK"`. Same mechanism as the 2026-08-26 pi:deepseek finding; `resolveOllamaMaxTokens` already floors this (`step.go`) |
| V22 | **PENDING** — the **actual** quota-exhaustion response: exact HTTP status and body. AC7/AC8 and the Conflict Surface retry row are written against this, not against an assumed 429 | Phase 2: tight loop against a level-1 model until exhaustion | *Not yet measured — requires a live plan* |
| V26 | **POS/NEG** — `/api/usage` works and per-model request counting is accurate, **but it publishes no limit, so headroom is not computable** | `GET https://ollama.com/api/usage -H "Authorization: Bearer $OLLAMA_API_KEY"`, read before and after a 7-request / ~6,000-token burst | **200.** Shape: `activity{cost, period{type:"last_4_weeks",starting_at,ending_at}, models[]}` + `limits{session{usage,models[]}, weekly{usage,models[]}}`, where each `models[]` entry is `{name, request_count}`. **Accurate**: counted exactly my calls (`gpt-oss:20b` 1→4→6, `qwen3.5:397b` 1) — an independent third instrument confirming remote execution. **`usage` stayed `0`** across all 7 requests and ~6k tokens, so it is coarse and is *not* a token counter. **`activity.cost` stayed `"0.00000"` and `activity.models` stayed `[]`** — the provider reports no per-token cost, which is direct evidence that D1 imputation is the only option. **Critically: `limits.{session,weekly}` contain ONLY `usage` and `models` — no `limit`, `max`, `remaining` or `reset_at`.** The gauge is a numerator with no denominator |
| V27 | **NEG/POS — Phase 1 blocker the design did not anticipate.** The ollama provider **hardcodes token counts to 0**, so imputed pricing (D1) is *necessary but not sufficient*: `cost_usd` is 0 regardless | `grep -rn 'InputTokens\|OutputTokens\|TotalTokens' internal/ai/ollama/*.go` · control: same grep on `internal/ai/openrouter/` | `client.go:203-205` and `step.go:495-497` both set `InputTokens/OutputTokens/TotalTokens: 0` with the comment *"Ollama doesn't report tokens the same way"*. **Control**: `openrouter/chat.go:193-194` reads `result.Usage.PromptTokens`. **The comment is stale** — the `/v1` path returns usage in standard OpenAI shape (measured: `{"prompt_tokens":76,"completion_tokens":900,"total_tokens":976}`), and the native path exposes `PromptEvalCount`/`EvalCount`. Invisible until now because **every** ollama row is priced `0/0`, so cost was 0 by both routes at once |
| V28 | **POS — the standard-mode cloud eval PASSED end-to-end, and provenance came out right** | `ailang eval -model motoko-cloud-gpt-oss-20b -benchmark fizzbuzz -langs ailang`, then read the banked row + `/api/usage` | `compile_ok`/`runtime_ok`/`stdout_ok` all **true** — real AILANG fizzbuzz from the cloud model. `cost_provenance` = **`metered`**, NOT `free-local` — the non-zero imputed pricing did its job (contrast V18). Cloud `request_count` 6→7 confirms the call went remote. **But** `input/output/total_tokens` all **0** and `cost_usd` **0**, per V27 |
| V29 | **POS — D4 confirmed live, not just inferred from source**: a cloud row IS wrongly serialized by the single-GPU clamp | `ailang eval-suite --agent --models motoko-cloud-gpt-oss-20b --benchmarks fizzbuzz --dry-run` | Emits `⚠ Local/agent-only model on the single-GPU rig — forcing --parallel 1 (was 10)`. The row is not GPU-bound at all. V12 predicted this from the predicate; V29 observes it |
| V30 | **NEG — the motoko agent path is blocked by a PRE-EXISTING fault, not by anything in this design** | canary run + `ls -t $TMPDIR/motoko-stderr-*.log` + `grep -c motoko_ext_abi <lock>` | Canary fails: `module loading error: ... package "sunholo/motoko_ext_abi" not found in ailang.lock`, **before any model is contacted**. Identical failures logged at **09:33 and 09:37**, an hour before this row existed (10:38) ⇒ model-independent. Main checkout has **no `ailang.lock`**; `mk-ast` has it with **2** `motoko_ext_abi` hits. Same class as the known motoko fork-sync lockfile issue |
| V31 | **POS — THE DESIGN'S CENTRAL CLAIM, PROVEN END-TO-END.** A full motoko agent-mode benchmark runs on Ollama Cloud with zero code change: only a `models.yml` row | `ailang eval-suite --agent --models motoko-cloud-gpt-oss-20b --benchmarks fizzbuzz --langs ailang` (after the V30 blocker was cleared) | **Success 1/1 (100%)**, 40s wall-clock. Banked row: `stdout_ok=true`, `error_category='none'`, `duration_ms=35094`. Cloud-side `request_count` 7→18 confirms remote. The canary that had timed out at 4m0s for four consecutive runs now passes |
| V32 | **POS/NEG — AC9's scope is NARROWER than V27 implied.** Token accounting works in **agent** mode and is zeroed only on the **direct-provider** path | same banked row as V31, vs the V28 standard-mode row | Agent row: `input_tokens=282048`, `output_tokens=1971`, `total_tokens=284019`, **`cost_usd=0.00868`**, `cost_provenance='metered'`. Standard row (V28): all tokens **0**, `cost_usd` **0**. motoko supplies its own accounting, so it bypasses the hardcoded zeros at `client.go:203-205`/`step.go:495-497`. **AC9 therefore fixes standard mode; agent mode is already correct.** Also corroborates V10's ~300k/trial estimate from banked history — measured 284k |
| V33 | **POS — first quota calibration: `usage` finally moved off zero** | `/api/usage` before and after the V31 run | `session.usage` **0 → 0.002** across 11 requests / ~290k tokens, all on the **level-1** model `gpt-oss:20b`. `activity.cost` remained **"0.00000"**, reconfirming no per-token billing. **Rate: ~0.002 usage units per ~290k-token agent trial at level 1.** Extrapolating V11's 56-benchmark pass (~17M tokens) ⇒ **~0.12 units per model per full pass**. **The denominator is still unpublished (V26)**, so "0.12 of what?" is unanswered — do NOT read 0.002 as 0.2% without evidence the cap is 1.0. The level-4 multiplier remains unmeasured |
| V34 | **POS — AC9 FIXED and verified end-to-end.** The native paths now report real token counts, so an imputed price finally produces a real `cost_usd` | new `internal/ai/ollama/tokens.go` (`tokenTally`) wired into `client.go` Generate and `step.go`'s native tool path; re-ran the V28 standard-mode benchmark | Same benchmark, before → after: `input 0 → 27032`, `output 0 → 760`, `total 0 → 27792`, **`cost_usd 0 → 0.00090976`**, still `stdout_ok=true` / `cost_provenance='metered'`. 3 unit tests pin the streamed shape (metric-free chunks then a terminal chunk), the trailing-empty-chunk case, and that a genuinely metric-free provider still yields 0 rather than a fabricated count. `go test ./internal/ai/... ./internal/eval_harness/...` → **9 packages pass, 0 failures**. Applies to **local** rows too: `POST localhost:11434/api/chat` on `qwen3.8:27b-mxfp8` returns `prompt_eval_count=11`, `eval_count=4` — the same fields the tally reads — so the `max_tokens_per_bench` WORK gate now has real counts even though local pricing stays `0/0` |
| V35 | **OBS — a better route-identity signal than the name suffix exists, for D6.** `ollamaapi.ChatResponse` carries `RemoteModel` and `RemoteHost` | read `api/types.go:520-535` (vendored 0.32.15) | `RemoteModel string \`json:"remote_model,omitempty"\`` and `RemoteHost string \`json:"remote_host,omitempty"\`` — populated by the daemon for cloud-proxied responses. This is the provider *telling us* the request went remote, rather than us inferring it from a name suffix the proxy then strips (V21). **Not implemented** — recorded so D6 can weigh it against the suffix and an explicit `route:` field. Note it is on the native ChatResponse; whether the `/v1` path surfaces an equivalent is unverified |

## Quorum Review Record

Two rounds run 2026-08-26 via `ailang design-quorum` (gpt5-6-sol + gemini-3-1-pro,
reject-by-default). Total cost **$0.167**. Artifacts in `.ailang/state/mission-quorum/`.
Both rounds returned **blocked**, and every objection was accepted rather than argued.

| Round | Reviewer | Objection | Resolution |
|---|---|---|---|
| 0 | gemini-3-1-pro | The `eval_suite.go` clamp premise drives D4/AC6 but has no Verification Log row | **Accepted.** Added V12 — and the cite was wrong too (270 is the comment; the block is 271-275), corrected throughout |
| 0 | gpt5-6-sol | No Conflict Surface, despite overlapping pricing, banking, concurrency, retry and error-classification machinery | **Accepted**, though the doc was technically exempt (files not in the parser/typechecker/codegen set). The section paid for itself immediately — see below |
| 1 | gpt5-6-sol | The zero-code premise is verified only against *vendored 0.32.15 source*; the rig runs 0.32.14 and no harness config was inspected. Phase 1 was testing the foundational claim, not implementing a verified design | **Accepted.** Phase 1 recast as **Phase 0**, a gating discovery spike; V19-V21 added as explicitly pending; assertive language made conditional |
| 1 | gemini-3-1-pro | AC8 mandates a "429 carve-out" against an **unobserved** exhaustion contract — it could be 402, 403 or a 200 payload | **Accepted.** V22 added; AC8 and the Testing Strategy now require the carve-out be written against the observed shape |

**What the review actually caught** (i.e. what would otherwise have shipped):

1. **A live bug in shared code.** V15: `isRetryableError` returns true on any `"429"` substring, so
   quota exhaustion would have retried into a spent bucket. The original doc left retry as
   implementer latitude — "agent may choose" would silently have meant "retries forever".
2. **A positively false label.** V18: `0/0` pricing maps to `free-local`, not merely "unlabelled" —
   a subscription lane would have claimed to be on-device. The correct label
   (`list-price-equivalent` + `AuthLaneSubscription`) already existed, shrinking D1 from
   "invent a treatment" to "pick the numbers" and dropping it high → med.
3. **Two concepts nearly reinvented.** V14: `quota_exhausted` already exists. V13: no route field
   exists (0 vs a 120-key control), turning suffix-vs-field from a guess into an informed D6.
4. **Overclaimed status.** The design is sound; the doc asserted certainty it had not earned.

**The re-quorum-once guardrail is now exhausted.** A third round is deliberately NOT run — the
remaining objections cannot be closed by more review, only by the Phase 0 spike, and grinding
rounds is the documented failure mode that parks sound designs (m-ailang-fmt, iter 49). This doc
goes to Mark with its gaps labelled.

## Related Documents

<!-- Auto-populated by Ollama neural search on "ollama cloud provider" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_0/m-eval-ollama-local-models.md](../../implemented/v0_7_0/m-eval-ollama-local-models.md) (0.51) — **top neural match; genuinely distinct.** That doc onboards *local* GPU-resident models; this one adds a *remote* route reached through the same daemon. It is the closest prior art for the `models.yml` row shape, and its `pricing: 0/0` convention is precisely what D1 must NOT inherit
- [design_docs/implemented/v0_8_1/m-cloud-storage.md](../../implemented/v0_8_1/m-cloud-storage.md) (0.44) — unrelated ("cloud" as storage backend, not inference)
- [design_docs/implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md](../../implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md) (0.40) — local rotation mechanics
- [design_docs/implemented/v0_16_0/m-ai-openrouter-provider.md](../../implemented/v0_16_0/m-ai-openrouter-provider.md) — not surfaced by search, but the **direct structural precedent**: the last time a multi-model open-weight provider was onboarded
- [design_docs/planned/m-ollama-v1-streaming-idle-timeout.md](../m-ollama-v1-streaming-idle-timeout.md) — the streaming `/v1` work this design deliberately reuses rather than forks

**Planned (checked for overlap — none found):**
- [design_docs/planned/v1_1_0/global-collaboration-hub.md](../v1_1_0/global-collaboration-hub.md) (0.45) — warn-band score, no topical overlap
- [design_docs/planned/docparse-billing/m-billing-docparse-billing-agent-payment.md](../docparse-billing/m-billing-docparse-billing-agent-payment.md) (0.44) — billing for docparse, not model routing
- [design_docs/planned/docparse-billing/responsibility-multivac.md](../docparse-billing/responsibility-multivac.md) (0.43) — no overlap

*Duplicate/coverage gate: highest neural score on a planned doc is **0.45** and on an implemented
doc **0.51**, both below the 0.75/0.65 reject thresholds and inside the 0.45-0.65 warn band. The
warn-band docs were read; the distinction from `m-eval-ollama-local-models` is stated above.
SimHash returned 1.00/0.95 on topically unrelated docs — keyword noise, not a duplicate signal.*

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [docs.ollama.com/cloud](https://docs.ollama.com/cloud) — cloud models, signin, API keys
- [ollama.com/pricing](https://ollama.com/pricing) — tiers, usage levels, reset windows (V9)
- Vendored source: `github.com/ollama/ollama@v0.32.15` — `server/routes.go:1916`,
  `server/cloud_proxy.go`, `internal/modelref/modelref.go:99-118`, `api/client.go:119`
- [MOTOKO.md](../../../MOTOKO.md) — which motoko checkout evals actually use
- [design_docs/PROGRAM.md](../../PROGRAM.md) — routing lanes; this is an **extension**, not a core change

## Future Work

- **OpenRouter vs Ollama Cloud paired A/B** on the ~14 duplicate models — same weights, two
  routes. Isolates vendor routing/quantization effects from model capability. Must be a paired
  run, never a pooled aggregate
- **Bearer-key path (D3) for keyless environments** — Cloud Run, cloud CI, and managed agents have
  no local ollama daemon to proxy through. ~20 LOC at the two V5 sites
- **Cloud-model executors for the mission fleet** — a 397B executor is strictly stronger than
  anything that fits the rig. Gated on the D2 quota picture, since missions run unattended and a
  mid-iteration quota stall would wedge the loop
- **Automated quota burn-down gauge** off `/api/usage`, wired into the rotation as a pre-flight
  check, so a rotation refuses to start rather than dying halfway

---

**Document created**: 2026-08-26
**Last updated**: 2026-08-26
