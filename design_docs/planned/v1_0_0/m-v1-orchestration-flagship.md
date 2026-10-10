# M-V1-ORCHESTRATION-FLAGSHIP: the package cascade, made provable, as the v1.0 flagship

**Status**: Planned (r7, re-centred on the package cascade on Mark's direction 2026-10-09: *"yep this is what I meant for v1.0.0"*). New scope, so a fresh quorum. Design Freeze open for Mark.
**Target**: v1.0.0, clause 4 of the [v1.0 bar](../../v1-mission.md#the-v10-bar--v2-product-shaped-ratified-2026-07-11-mark-supersedes-the-2026-07-10-hygiene-bar)
**Priority**: P0 for v1.0 (the clause-4 NEW-DOC unit counted since `D-51`)
**Estimated**: flagship-owned milestones ~1.5 weeks after #1730; it also waits on the prerequisites below
**Created**: 2026-10-09 · **Revised**: r7 2026-10-09 · **Author**: attended session with Mark (Claude Opus 5.5)

---

## The claim

Mark, 2026-10-09: *the AILANG packages plus the AILANG message system are the orchestration
framework.* The framework already runs in production (V1–V5):

1. **A package publishes.** The registry pins its interface (v2 hash over every export's type and
   effects) and refuses a contract Z3 refutes. The effect ceiling is enforced at compile time.
2. **Dependents are notified on a typed edge.** `ailang publish` sends a `CascadeEnvelopeFields` to the
   `ailang-cascade` Pub/Sub topic, which only the coordinator's service account may publish to. The
   envelope carries the change class A/B/C, both interface and content hashes, `effects_widened`, and
   the previous and new effect ceilings.
3. **Routine changes are verified, not delegated.** For class A/B the cloud coordinator bumps the
   dependency, regenerates `ailang lock`, and must pass `ailang check` and `ailang test` before it
   commits. No AI is involved.
4. **AI only where interpretation is needed.** Class C (exports removed or effects widened), an
   unknown class, or any failure of step 3 escalates to the package's agent. That agent works in the
   AILANG-only lane: it acts only by submitting programs that `run --policy` type-checks and admits.
5. **People keep authority.** Package agents run with `auto_merge: false` and `skip_approval: false`,
   so a widening never lands without a human (V5).

The same plane carries the **high-volume** package work: feedback and bug reports routed to `pkg:`
inboxes and fixed by those agents. There are dozens of `[agent] pkg-sunholo-*` PRs in the last three
weeks across ailang-parse, email-parse and docparse (V6).

Proposed v1 sentence (F1, verified per clause at release):

> ***"In AILANG, packages orchestrate themselves: interfaces and effects are pinned and proved at
> publish, changes travel as typed envelopes, routine updates are verified by the compiler and tests
> instead of delegated to a model, AI agents act only through type-checked programs admitted against
> your policy, and any widening of authority waits for a human."***

## The finding that comes first: the cascade does not dispatch today (#1730)

Since at least 2026-09-17, **no cascade notification has been dispatched**. In every one of the 7
`ailang-cascade` publish windows in 90 days, the coordinator logged `inbox message not found — NOT
dispatching` or `… does not exist in the store — dropping` for the notification's id (V7).

The cause:
- `publish` writes its legacy inbox row to the **publisher's local SQLite**, and reuses that row's
  id as the cascade notification's `message_id`.
- The adapter must hydrate every notification from the **cloud** store (`pubsub_adapter.go:222-236`).
  It never finds the row, so it never uses the envelope it already has.

Filed as **#1730** (P0, part of v1.0.0). Nothing in this flagship can be demonstrated until it is
fixed, so it is prerequisite **P0**.

## What the flagship is

**The cascade, run end to end on fixture packages in CI, with each property shown passing and shown
failing.**

Fixtures live in `examples/flagship/`:
- **`flagship/schemas`:** the dependency. It publishes v1, then three successor versions, one for
  each scenario below.
- **`flagship/triage`:** a dependent that is itself a small AI pipeline (the clause-4 "verified
  multi-step AI pipeline"). It makes two typed model calls (`callTyped` with
  [m-json-codecs](m-json-codecs.md)) under `! {AI @limit=4}`, and its decisions pass a Z3-proved gate
  (the email-parse pattern, V8). Its customer text is secret-labelled. Its `ailang test` runs against
  committed fixtures, so the cascade's test gate exercises the AI pipeline deterministically, with no
  provider calls.

| Scenario | `flagship/schemas` change | What must happen | Proves |
|---|---|---|---|
| **S1 additive** | adds an export (class B) | Envelope class B → deterministic bump → `check` + `test` green → commit; no agent invoked | Typed edge; verification instead of delegation |
| **S2 behaviour change behind an unchanged interface** | changes what an exported function returns, with the same signature, so the classifier computes class A (content-only) | Deterministic bump → `triage`'s `ailang test` **fails** → escalation to the `triage` package agent, never a silent green | Tests catch what an interface hash cannot see |
| **S3 widening** | widens `[effects].max` (adds `Net`) | `effects_widened: true` → class C → AI path; the resulting PR is not auto-merged | Authority widening waits for a human |
| **S4 feedback** | none; a feedback message to `pkg:flagship/triage` | Routed to the package agent; any program the agent runs is admitted by `run --policy` (a fixture policy refusing `FS` makes a mutant fail) | The high-volume path, and admission |

**Mutants CI must see fail:**
- `triage`'s gate widened by one action → `ailang verify` fails.
- `triage`'s secret placed into a prompt → `ailang check` fails (needs P3).
- `triage` asks for `FS` under the policy → admission refused.
- S1 with the deterministic path forced to commit despite a red `check` → the bump test fails.
- S3 published as class B → the envelope test fails, because `effects_widened` must force C.

## Prerequisites

| # | What | Owner | State |
|---|---|---|---|
| **P0** | **The cascade dispatches** | **#1730** | Filed 2026-10-09 |
| P1 | Higher-order effect soundness (a `runTools` dispatch reading files passes `ailang check`, V9) | m-effect-row-var-unification (#616), m-effect-latent-function-values (#1326, #573) | Designed; PRs #1708 and #1678 await approval |
| P2 | `run --policy` escapes closed | #1548, #1569, #1607, #1720 | Designed for three of four |
| P3 | Secrets cannot reach a prompt or output (V10), across modules and in traces | `m-ifc-ai-and-io-sinks` (to write), m-ifc-cross-module-labels (#1134), M-TRACE-LABEL-AWARE | — |
| P4 | Typed codecs for `triage`'s model calls | [m-json-codecs](m-json-codecs.md) | New 2026-10-09 |
| P5 | Inbox kinds (completion, handoff, approval) typed to the cascade envelope's standard; one package envelope instead of two | [m-typed-message-plane](m-typed-message-plane.md) | New 2026-10-09; not on the S1–S4 critical path |

## Flagship-owned milestones

**M1: The deterministic path is tested** (~2 days)
- `deterministicCascadeBump` (`cmd/ailang/coordinator_cloud_cascade.go:65`) has no direct test today (V11).
- Add tests on fixture packages:
  - toml bump, lock, check and test all pass → a commit;
  - each step fails → an error that the wrapper turns into escalation;
  - `classifyDispatchPath` maps A/B → deterministic, and C or unknown → AI.

**M2: The envelope carries widening honestly** (~1 day)
- A test asserts that a publish whose ceiling widens produces `effects_widened: true` and class C.
- It is built from the same code `publish` uses: `effectsWidened` and `mapChangeClassToSchema` at
  `pkg_publish.go:~500-510`, extracted so the test can call them without a network.

**M3: End-to-end flagship job** (~4 days)
- `examples/flagship/` with the fixtures, policies and mutants.
- A required CI job runs S1–S4 against a **local** coordinator: SQLite store and an in-process
  publisher, with no GCP. It also runs the five mutants. The job is fixture-only, makes no provider
  calls, and has a time budget (F3).

**M4: Production evidence and positioning** (~2 days)
- `ailang pkg cascade status` and a dashboard line report real cascades (publishes, deterministic
  bumps, escalations, human merges) from the prod plane. The README leads with the flagship, the F1
  sentence and those live numbers.
- The three `tier: vision` orchestration benchmarks are rewritten as fixture-backed multi-step tasks
  under new ids (V12).

## Conflict Surface

| Question | Answer |
|---|---|
| Positions extended | Tests around `deterministicCascadeBump` and `classifyDispatchPath`; a small extraction from `pkg_publish.go` (M2); `examples/flagship/`; a new required CI job; benchmark files (new ids) |
| What already lives there | The cloud cascade path (`coordinator_cloud_cascade.go`), the publish path, `cascade_scheduler_test.go`, `pkg_cascade_status_test.go`, every PR's CI, and the `eval-suite -tier` consumers (`make/eval.mk:58-60`, `eval_elo.go:289-303`) |
| How it disambiguates | M1 and M2 are tests plus a pure extraction (no behaviour change). M3 runs locally (no GCP, no providers). Benchmarks get new ids, so no history is reinterpreted |
| Must still work | Real publishes (golden test of the envelope before and after the M2 extraction) and the existing cascade tests |
| Deliberate changes | None in production behaviour. #1730's fix, which is owned by that issue, is the only production change the flagship depends on |

## Design Freeze

- [ ] **F1:** the direction now (the flagship is the package cascade plus package agents). The bar
  sentence is ratified per clause at the release gate after its prerequisite lands and its scenario
  or mutant passes in CI. The fallback, if P3 slips, drops the secret-flow clause.
- [ ] **F2:** `flagship/triage` (an AI-pipeline package) as the dependent, so the cascade's test gate
  exercises clause 4's AI properties.
- [ ] **F3:** the flagship job's time budget; recommend ≤4 minutes, fixture-only, required on `dev`.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Routine updates are decided by check + test, not a model |
| A2 Replayability | +1 | The AI package's tests run on committed fixtures inside the cascade gate |
| A3 Effect Legibility | +1 | `effects_widened` and ceilings travel in the envelope and decide the path |
| A4 Explicit Authority | +1 | An IAM-restricted topic, policy admission for agents, and humans for widening |
| A5 Bounded Verification | +1 | Z3 at publish and in `triage`'s gate |
| A6 Safe Concurrency | 0 | — |
| A7 Machines First | +1 | Typed envelopes instead of prose; the agent is invoked only when needed |
| A8 Minimal Syntax | 0 | No syntax |
| A9 Cost Visibility | +1 | AI is spent only on escalations; budgets are declared in `triage` |
| A10 Composability | +1 | Uses the package, message and policy systems as they are |
| A11 Structured Failure | +1 | A failed bump escalates; it is never silently green |
| A12 System Boundary | +1 | Dependency edges and agent actions are checked boundaries |

**Net +10.** No −1.

## Verification Log (2026-10-09; `origin/dev` `c92739681`; prod project `ailang-multivac`)

| # | Claim | Evidence |
|---|---|---|
| V1 | Packages are checked units | `internal/pipeline/effect_ceiling.go`; `internal/pkg/hasher_v2.go`; `internal/pkg/quality.go` PUB006; `ailang pkg stats`: 48 packages, 376 versions |
| V2 | Publish sends a typed cascade envelope on an IAM-restricted topic | `cmd/ailang/pkg_publish.go` ~455–510 (`cascadePublisher`, `CascadeEnvelopeFields{RootPackage, ChangeClass, …, EffectsWidened, PrevEffectCeiling, NewEffectCeiling}`); `internal/pubsub/publisher.go:70-100` ("publish IAM is restricted to the coordinator service account"); topic `projects/ailang-multivac/topics/ailang-cascade` |
| V3 | Deterministic first, then AI | `cmd/ailang/coordinator_cloud_cascade.go:20-64` (`classifyDispatchPath`: A, B → deterministic; otherwise AI), `:65-130` (toml bump → `ailang lock` → `ailang check` → `ailang test`; any failure "escalating to AI") |
| V4 | The coordinator receives the envelope | `internal/coordinator/pubsub_adapter.go:166-186` (copies `RootPackage`, `ChangeClass`, hashes, `EffectsWidened`, ceilings onto the message) |
| V5 | Package agents never auto-merge | `ailang-multivac/config/config.cloud.yaml:197-211` (`package_agent_template`: `auto_merge: false`, `skip_approval: false`) |
| V6 | High-volume package-agent work | `gh search prs --owner sunholo-data`: dozens of `[agent] pkg-sunholo-{ailang-parse,email,docparse}` PRs, 2026-09-25 to 10-07 |
| V7 | The cascade does not dispatch (#1730) | Cloud Monitoring `pubsub.googleapis.com/topic/send_message_operation_count`, `ailang-cascade`, 90 days: 7 publish minutes (09-17 20:28; 09-29 07:38, 07:48, 08:53; 09-30 17:23; 10-02 18:13; 10-07 15:17). Coordinator logs in each window: `inbox message not found … NOT dispatching` or `… does not exist in the store — dropping` for local-store ids (`msg_20261007_171616_782f344c` at 15:16:22 UTC, etc.); `pubsub_adapter.go:222-236`. The publish side is V7b |
| V7b | #1730 publish side: the local id becomes the cascade id | `cmd/ailang/pkg_publish.go:310` `store, err := openPkgMsgStore()` → `cmd/ailang/pkg_msg.go:266-268` `messaging.OpenStore(messaging.GetDefaultDatabasePath())` → `internal/messaging/store.go:84-90` (`statedir.Path("collaboration.db")`, local SQLite); `emitDependentNotifications` (`pkg_publish.go:374`): `msgID, err := messaging.EmitUpgradeAvailable(store, …)` then `cascadeMsgID := msgID` before `PublishCascade` |
| V8 | The AI-pipeline pattern | email-parse (private): `packages/eparse/triage.ail` 30–60 (labels, `{not llm}` sink), 575–670 (`normalise` under `Declassify`, `ensures` over 5 literals) |
| V9 | Higher-order effects leak | `tools3.ail` (`! {FS}` dispatch to `runTools` inside `main ! {IO, AI}`) → ✓ No errors found! |
| V10 | Secrets reach prompts and output | `ifc1.ail` / `ifc2.ail` (`call("my key is ${k}")`, `println(k)`) → ✓ No errors found! |
| V11 | The deterministic path is untested | `git ls-tree origin/dev cmd/ailang internal/coordinator`: cascade tests are `cascade_scheduler_test.go` and `pkg_cascade_status_test.go` only; no test file for `coordinator_cloud_cascade.go` |
| V12 | Orchestration benchmarks are `vision` tier | `benchmarks/{ai_effect_summarize,ai_effect_json_schema,multi_agent_handoff}.yml`; `make/eval.mk:58-60` |

## Related Documents

[m-json-codecs](m-json-codecs.md) · [m-typed-message-plane](m-typed-message-plane.md) ·
M-PKG-AUTONOMOUS-CASCADE-SAFE / M-PKG-CASCADE-DETERMINISTIC-FIRST (implemented, v0.16.0) ·
[M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) ·
[m-package-protocol-manifests](../m-package-protocol-manifests.md) (v1.1)

---

**Document created**: 2026-10-09 · **Last updated**: 2026-10-09 (r7)
