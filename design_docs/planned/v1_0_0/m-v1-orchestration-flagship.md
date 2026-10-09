# M-V1-ORCHESTRATION-FLAGSHIP: a small orchestration on the real framework that proves the v1 claim

**Status**: Planned (r6; Quorum guardrail spent: r1 and r2 both BLOCKED; every r2 objection is dispositioned below with new evidence rows. The sprint-planner re-verifies at plan time. Design Freeze open for Mark)
**Target**: v1.0.0, clause 4 of the [v1.0 bar](../../v1-mission.md#the-v10-bar--v2-product-shaped-ratified-2026-07-11-mark-supersedes-the-2026-07-10-hygiene-bar)
**Priority**: P0 for v1.0 (the clause-4 NEW-DOC unit counted since `D-51`)
**Estimated**: flagship-owned milestones ~1.5 weeks; it waits on the prerequisite docs listed below
**Created**: 2026-10-09 · **Revised**: r4 2026-10-09 · **Author**: attended session with Mark (Claude Opus 5.5)
**Quorum trigger**: #1 (Design Freeze). r1, r2 and r3 were scoped as a standalone program; r4 is a new scope, so it gets a fresh quorum.

---

## The claim

Mark, 2026-10-09: the **AILANG packages plus the AILANG message system are the orchestration
framework**. The research passes confirm the framework exists and runs:

- **Nodes are packages and agents.** Packages carry compile-time effect ceilings, a v2 interface hash
  over every export's type and effects, and publishing is refused when Z3 refutes a contract (V1).
  Each package gets a `pkg:` inbox agent, which acts only by submitting AILANG programs that `ailang
  run --policy` type-checks and admits against an operator policy (V2).
- **Edges are messages.** The plane carried 759 messages, 61 handoffs and 99 approvals in the last
  7 days (V3).
- **Authority escalates by type.** A dependency that widens its effects or regresses a contract is
  change class C, which forces human approval and turns off auto-merge (V4).

What it lacks is **typed edges**: payloads are strings, and an AILANG program cannot send through
its effect row. [m-typed-message-plane](m-typed-message-plane.md) adds them without disturbing
today's traffic. Once it lands, v1 can claim (F1):

> ***"AILANG orchestrates AI agents as typed, verified packages: every agent acts only through
> type-checked programs admitted against your policy; packages publish proved interfaces; the
> messages between them are typed, budgeted effects; and any widening of authority stops for a
> human."***

## What the flagship is

**A small, public, runnable orchestration**, `examples/flagship/`, built from the real pieces: two
packages, typed messages and policy. Its CI job proves every clause of the claim on every PR, and a
committed broken copy (a mutant) shows each property failing.

| Step | Package | What runs | What it proves |
|---|---|---|---|
| 1 | `flagship/triage` | `callTyped(prompt, decisionJson())` with model A, then a second opinion with model B (`step(model, …)`, V9). A Z3-proved `resolve` gate maps the two typed decisions to one of a closed set of actions (the email-parse pattern, V10) | Typed model output; proved decision logic; budgets (`! {AI @limit=4}`) |
| 2 | `flagship/triage` | `sendTyped("pkg:flagship/responder", "ailang.flagship.action/v1", actionJson(), action)` under `! {Msg}` | Typed, budgeted message; the send sits inside the effect row and the policy |
| 3 | `flagship/responder` | `recvTyped(inbox, "ailang.flagship.action/v1", actionJson())` decodes the action. It drafts a reply with a model, and a Z3-proved allowlist gate (the Daneel pattern, V10) decides the recipient | Typed receive; proved authority over the side effect |
| 4 | both | Each package declares `[effects].max` and runs only under `ailang run --policy` with a committed policy | Authority checked before anything runs |
| 5 | `flagship/responder` v2 | A variant that widens its effects (adds `Net`) is classified by the package cascade | Effect widening → change class C → human approval (V4) |

The customer text is labelled `<secret>`-style PII and reaches the model and the sink only through
explicit, scoped `Declassify` steps.

**The five mutants CI must see fail:**

| Mutant | Change | Must fail at |
|---|---|---|
| S | The PII placed into a prompt without `Declassify` | `ailang check` (needs P3) |
| G | The `resolve` gate widened by one action | `ailang verify` |
| P | `responder` asks for `FS` | `ailang run --policy` admission |
| T | `triage` sends a payload of the wrong type for `ailang.flagship.action/v1` | `sendTyped`'s schema check, as a typed error with the schema in `enforce` (message plane M0 and M3). It fails at compile time only if message-plane F2 binds schema ids to types; with plain string ids (the recommended v1 shape) the check runs at send |
| W | `responder` v2 widens `[effects].max` | the cascade classifier → class C, approval required |

Plus **replay**: a recorded run (`-emit-trace jsonl`) replays through `ailang replay` with zero
provider calls (M2a below).

## What the flagship is not

- **Not motoko or the AILANG-only lane as headline.** motoko has zero production runs and no
  licence, ties pi at 63% more cost, and its own design concludes the AILANG runtime cannot contain it.
  The lane has five open escapes (V5). Both stay on their own tracks and are cited as production use
  of the pattern once their gaps close.
- **Not declared agent contracts.** Checking the routing graph itself (which kinds an inbox accepts
  and emits) is [m-package-protocol-manifests](../m-package-protocol-manifests.md), v1.1.
- **Not a promoted production pipeline.** email-parse, Daneel and docparse are private and hold real
  data (V10). The flagship copies their two strongest patterns.

## Prerequisites (owned by other docs; the flagship cannot ship until they land)

| # | What | Owner | State |
|---|---|---|---|
| P1 | Higher-order effect soundness: a `runTools` dispatch reading files currently passes `ailang check` (V6) | [m-effect-row-var-unification](m-effect-row-var-unification.md) (#616), [m-effect-latent-function-values](../v0_48_0/m-effect-latent-function-values.md) (#1326, #573) | Designed; executor PR #1708 and plan PR #1678 await approval |
| P2 | `run --policy` escapes closed | #1548 ([m-run-policy-result-unforgeable](../v0_53_0/m-run-policy-result-unforgeable.md)), #1569 ([m-fs-deny-write-dir-rename](../v0_52_6/m-fs-deny-write-dir-rename.md)), #1607 ([m-pkg-registry-confinement](../v0_52_6/m-pkg-registry-confinement.md)), #1720 | Designed for three of four |
| P3 | Secrets cannot reach a prompt or output, across modules and in traces (V7) | New `m-ifc-ai-and-io-sinks` (to write), [m-ifc-cross-module-labels](../m-ifc-cross-module-labels.md) (#1134), [M-TRACE-LABEL-AWARE](../v0_36_0/m-trace-label-aware.md) | The sinks doc is not yet written |
| P4 | Typed edges | [m-typed-message-plane](m-typed-message-plane.md) M0–M3 | New 2026-10-09 |
| P5 | Typed JSON codecs (`callTyped`, `sendTyped`) | [m-json-codecs](m-json-codecs.md) | New 2026-10-09; explicit codec values, proved expressible today |

## Flagship-owned milestones

**M1: Replay serves recorded AI results** (~2 days)
- **Record:** the existing `ailang run -emit-trace jsonl`, which already writes each AI effect's
  `op_name`, `args` and `result` (V8).
- **Replay:** the existing `ailang replay <trace>`, whose documented purpose is to re-execute and
  compare against the baseline. During replay, an `AI` effect is served from the next recorded `AI`
  event whose `op_name` and `args` match. A call with no match is a mismatch (exit 1, naming the call),
  never a provider call. `--live-ai` keeps today's behaviour.
- **Scope of what's new:** one opt-in flag, `--live-ai`, which does not exist today (V16). There
  is no new trace format and no new effect mode. The default behaviour of `ailang replay` on AI
  effects changes, as stated in the Conflict Surface.
- **Match rule:** the *n*-th AI effect during replay must have the same `op_name` and byte-equal
  `args` as the *n*-th recorded AI event. AI effects are evaluated in program order (the batch
  evaluator is single-threaded), so order is defined. Any other shape is a mismatch.
- P3's trace-label work is required first.

**M2: Budget exhaustion is reported under `--policy`** (~1 day)
- A budget-exhausted `--policy` run reports `error_kind: "budget_exhausted"` in its result. Today
  that line says `ok:true` and the failure appears only on stderr (V11).

**M3: Cascade classification is runnable locally** (~1–2 days)

How the chain works today, every step cited:
1. `ailang publish --dry-run` builds the tarball, prints it and returns **before** uploading and
   before any message is emitted (`cmd/ailang/pkg_publish.go:138-141`, V14).
2. A real publish calls `emitPublishMessages` (`pkg_publish.go:309`). That calls the envelope
   builders `EmitUpgradeAvailable`, `EmitInterfaceChangeNotice` and `EmitEffectWideningWarning`
   (`internal/messaging/pkg_events.go:37` and following; called at `pkg_publish.go:345-360`), each of
   which builds a `PackageMessageEnvelope` and then hands it to `sendPackageMessage`, which validates
   and inserts into a store (`pkg_events.go:265-280`).
3. The coordinator's `AdjustAutonomyForChangeClass` / `ClassifyChange` turn an envelope into autonomy
   (`internal/coordinator/autonomy_router.go:21-80`, V4).

The change:
- Split each `Emit*` into **build** (returns the envelope) and **send** (today's insert). Both paths
  then use the same builders, with no second classifier.
- **Where the previous version comes from:** today `emitPublishMessages` takes it from the
  package's **lockfile** entry (`pkg.LoadLockFile(cwd)`, `pkg_publish.go:324-340`, V17).
- Add `ailang publish --dry-run --against <dir>`, which builds the same `PackageVersionInfo` from
  `<dir>`'s manifest and lockfile with the hashers publish already uses. That is a local previous
  version, so CI runs offline.
- For each envelope it prints the kind, the `ClassifyChange` class and the resulting `SkipApproval` /
  `AutoMerge` / `AutoApproveHandoffs`. Both are pure functions of the envelope and agent config, with
  no I/O (`autonomy_router.go:21-80`, V4). It writes no store and uploads nothing.
- CI runs it on `responder` v2 against v1 and asserts `effect-widening-warning` → class C →
  `SkipApproval=false, AutoMerge=false` (mutant W).

**M4: The flagship packages and CI job** (~4 days)
- `examples/flagship/triage` and `examples/flagship/responder`, with committed policies, fixtures and the five mutants.
- A required CI job on `dev` runs:
  - `ailang check`;
  - `ailang verify`;
  - the admitted runs under `--policy`;
  - each mutant's expected failure;
  - record and replay;
  - `publish --dry-run` on the W variant asserting class C.

**M5: Benchmarks and positioning** (~2 days)
- Rewrite the three `tier: vision` orchestration benchmarks (V12) as fixture-backed, multi-step,
  byte-matching tasks, and move them into `eval-core`.
- Lead the README with the flagship and the F1 sentence. Reconcile "The Deterministic Language for AI
  Coders" and the guide's "AILANG is not an agent framework" with it (V13), and drop "no competitor"
  in favour of naming the combination.

## Conflict Surface

| Question | Answer |
|---|---|
| Positions extended | M1: what `ailang replay` does on an `AI` effect. M2: the `--policy` result JSON. M3: `ailang publish --dry-run` output and the `Emit*` builders. M4: a new required CI job and two example packages. M5: three benchmark YAMLs' `tier`, and the README |
| What already lives there | M1: `ailang replay baseline.jsonl` as a regression check (devtools prompts), owned by `cmd/ailang/replay.go`; non-AI effects unchanged. M2: the pi `ailang_run` tool (`cmd/ailang/pi_assets/ailang-exec.ts`) parses that line. M3: `--dry-run` shows the tarball and returns before upload (`pkg_publish.go:138-141`); every real publish uses the `Emit*` builders. M4: every PR's CI; `examples/` is scanned by the examples corpus. M5: `tier` is read by `eval-suite -tier` (`make eval-core` → `eval-suite -tier core`, `make/eval.mk:58-60`; 23 benchmarks are `tier: core` today), by `eval_elo.go`'s promotion rules (`:289-303`) and by `cache_ops.go:126`; `internal/eval_harness/ai_caps.go:17-29` documents why these three can't byte-match today |
| How it disambiguates | M1: only `AI` effects with a recorded match change; `--live-ai` restores. M2: adds a field; `decision` keeps its meaning. M3: `--against` is new; plain `--dry-run` output unchanged; the build/send split keeps send's behaviour identical. M4: the job is fixture-only, has no network or provider calls (`--ai-stub-fixtures`, local message store), and has a time budget (F4). M5: the rewritten benchmarks get **new ids**, so no ELO or baseline history is reinterpreted; the old `vision` files are retired, not edited in place |
| Must still work | Replay of non-AI programs and the `replay.go` tests; the pi tool's parse; every existing `publish --dry-run` caller and real publish (golden test of the emitted envelopes before and after the split); `make eval-core` on the existing 23 benchmarks |
| Deliberate changes | `ailang replay` of an AI program no longer calls a model by default (release note) |

## Design Freeze

- [ ] **F1 (direction only, now):** that the v1 claim is about the package + message framework, as
  under "The claim". **The bar text itself is not amended now.** The sentence is ratified at the v1
  release gate, and only for the clauses whose prerequisites have landed and whose mutant passes in
  CI: verify, then freeze. If P4/P5 (typed edges) slip, the release sentence drops the "typed,
  budgeted messages" clause and ships the nodes-only claim.
- [ ] **F2:** the flagship is this two-package orchestration, not a standalone program, motoko or the lane.
- [ ] **F3:** declared agent contracts (graph type-checking) stay v1.1.
- [ ] **F4:** the flagship CI job's time budget. Recommend ≤3 minutes, fixture-only, required on `dev`.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | M1 replays AI results deterministically |
| A2 Replayability | +1 | Record → replay is a CI-checked property |
| A3 Effect Legibility | +1 | Messages, model calls and authority are all in effect rows the CI checks |
| A4 Explicit Authority | +1 | Admission per package, and escalation on widening, shown failing in CI |
| A5 Bounded Verification | +1 | Both gates are pure and Z3-proved |
| A6 Safe Concurrency | 0 | — |
| A7 Machines First | +1 | Typed decisions and messages replace string parsing |
| A8 Minimal Syntax | 0 | No syntax of its own |
| A9 Cost Visibility | +1 | Model and message budgets declared, enforced, reported (M2) |
| A10 Composability | +1 | Adds no orchestration machinery of its own; it composes the package, message and policy systems, as extended by the prerequisite docs P4/P5 |
| A11 Structured Failure | +1 | Every failure mode is typed and shown by a mutant |
| A12 System Boundary | +1 | Model, message and side-effect boundaries are all checked |

**Net +10.** No −1.

## Changes since r4 (quorum BLOCKED 3/3, 2026-10-09)

- **glm (mutant W's chain unverified):** M3 now cites each step: dry-run returns before emission
  (V14), the `Emit*` builders and `sendPackageMessage`, and the router. It specifies the build/send
  split and an offline `--against <dir>`.
- **kimi (freezing a bar sentence on same-day docs; A10 contradicts the prerequisites):** F1 now
  ratifies direction only. The sentence is ratified per clause at the release gate after its
  prerequisite lands and its mutant passes, with a nodes-only fallback. A10's justification corrected.
- **gemini (Conflict Surface omits M4/M5):** M4 and M5 rows added. They cover tier consumers
  (`eval-suite -tier`, ELO promotion, cache ops), new benchmark ids so history is not reinterpreted,
  and a fixture-only CI job with a time budget.

## Round-2 objections and dispositions (quorum r2 BLOCKED, 2026-10-09; guardrail spent)

- **glm (where the previous version comes from; whether the classifier can run outside the
  coordinator):** from the lockfile (V17), which `--against` mirrors from a directory. `ClassifyChange`
  and `AdjustAutonomyForChangeClass` are pure (V4).
- **kimi (replay match rule unspecified; `--live-ai` unverified; "no new mode" contradicted):** the
  match rule and ordering are specified. `--live-ai` is stated as new (V16), and the default-behaviour
  change is named.
- **gemini (motoko and docparse claims uncited):** V5 now cites motoko's lane design, and V10 cites
  repository visibility.

## Verification Log (2026-10-09, `ailang v0.52.3-44`, `origin/dev` `c92739681`)

| # | Claim | Evidence |
|---|---|---|
| V1 | Packages are checked units | `internal/pipeline/effect_ceiling.go` (compile-time `[effects].max`); `internal/pkg/hasher_v2.go` (`sha256:ifacev2:` over names, types, effects); `internal/pkg/quality.go` PUB006 (publish refused on a Z3-refuted contract); `ailang pkg stats`: 48 packages, 376 versions |
| V2 | Package agents act only through admitted programs | `ailang-multivac/config/config.cloud.yaml:178-228` (`package_agent_template`), 29 explicit `pkg:` agents plus derived, `tool_policy: ailang_only`, `policy_path pkg-ailang-only.toml`; `internal/executor/toolpolicy.go:40-41` |
| V3 | The plane runs | `ailang messages activity --hours 168`: 759 messages, 61 handoffs, 99 approval requests, 127 completions |
| V4 | Widening forces a human | `internal/coordinator/autonomy_router.go:51-80`: `PkgMsgEffectWidening` → `ChangeClassC`; `AdjustAutonomyForChangeClass` sets `SkipApproval=false`, `AutoMerge=false`, `AutoApproveHandoffs=false` for C; `ailang publish --help`: `--dry-run` "Create tarball and show what would be published, without uploading" |
| V5 | motoko and the lane are not headline-ready | `design_docs/planned/m-motoko-ailang-only-lane.md` (Spike A ruled out: the AILANG runtime cannot contain motoko; containment is box-level); motoko: 0 prod/test executions in 90 days, lane A/B 55/69 vs pi 55/69 at $6.86 vs $4.20, `licenseInfo: null`; lane escapes #1548, #1569, #1607, #1720, #1547 |
| V6 | Higher-order effects leak | `tools3.ail`: a `! {FS}` dispatch passed to `runTools` in `main ! {IO, AI}` → ✓ No errors found! |
| V7 | Secrets reach prompts and output | `ifc1.ail` `call("my key is ${k}")` and `ifc2.ail` `println(k)` with `k = secret(…)` → ✓ No errors found! |
| V8 | Traces record AI calls; `replay` is a baseline comparison | `-emit-trace jsonl`: `{"event":"effect",…,"effect":{"effect_name":"AI","op_name":"callJsonResult","args":[…],"result":"…"}}`; `changelogs/v0.8-cloud-features.md:233-234`; `cmd/ailang/prompts/devtools/v0.8.0-compact.md:60-62`; `cmd/ailang/replay.go`; with no model, `ailang replay` reports `no AI model configured` |
| V9 | Two models per run work in the default mode | `twom.ail`: `step(model, msgs, [])` called with `"model-a"` and `"model-b"` from `main ! {IO, AI}` → ✓ No errors found! |
| V10 | The production patterns copied | Visibility: `gh repo view sunholo-data/{docparse,email-parse,daneel} --json isPrivate` → `true`; `ailang-parse` → `false`. email-parse (no CI): `packages/eparse/triage.ail` 30–60 (labels, `{not llm}` SQL sink), 575–670 (`normalise` under `Declassify`, `ensures` over 5 literals). Daneel (private): `tools/daneel_policy.ail` send allowlist; `ci.sh` runs `ailang verify`, `smoke.sh` mutation-checks it |
| V11 | Budget exhaustion misreported under policy | Capability test 2026-10-09: a budget-exhausted `--policy` run prints `decision ok:true`, failure on stderr only |
| V12 | Orchestration benchmarks are `vision` tier | `benchmarks/{ai_effect_summarize,ai_effect_json_schema,multi_agent_handoff}.yml` → `tier: vision` |
| V14 | The publish emission chain | `cmd/ailang/pkg_publish.go:138-141` (`if *dryRunFlag { … return nil }` before upload); `:309` `emitPublishMessages` → `openPkgMsgStore()`; `:345-360` calls to `EmitUpgradeAvailable`, `EmitInterfaceChangeNotice`, `EmitEffectWideningWarning`; `internal/messaging/pkg_events.go:37` (builder), `:265-280` `sendPackageMessage` (validate, then `InsertInboxMessage`) |
| V15 | Benchmark tier consumers | `make/eval.mk:58-60` (`eval-core` → `eval-suite -tier core`); 23 `benchmarks/*.yml` with `tier: core`; `cmd/ailang/eval_elo.go:289-303` (tier promotion); `cmd/ailang/cache_ops.go:126`; `internal/eval_harness/ai_caps.go:17-29` (vision prose placeholders can't byte-match) |
| V16 | `--live-ai` does not exist yet | `git grep -n "live-ai" origin/dev -- cmd internal` → empty |
| V17 | Publish reads the previous version from the lockfile | `cmd/ailang/pkg_publish.go:324-340` (`lf, err := pkg.LoadLockFile(cwd)` → `oldInfo := messaging.PackageVersionInfo{Version, InterfaceHash, ContentHash, Effects, Exports}`) |
| V13 | Current positioning | `README.md` "The Deterministic Language for AI Coders"; `docs/docs/guides/ailang-vs-agents.mdx` "not an agent framework" |

## Related Documents

[m-typed-message-plane](m-typed-message-plane.md) · [m-json-codecs](m-json-codecs.md) ·
[m-package-protocol-manifests](../m-package-protocol-manifests.md) (v1.1) ·
[m-fable-strategy-review](../m-fable-strategy-review.md) R6 ·
[M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) ·
[m-contracts-as-code-vertical](../v0_29_0/m-contracts-as-code-vertical.md) (folds in as the proved gates)

---

**Document created**: 2026-10-09 · **Last updated**: 2026-10-09 (r4)
