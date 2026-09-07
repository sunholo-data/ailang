# M-CLOUD-PLANE-FALLBACK-LANES

**Status:** planned — **ANALYSIS TRUSTED, PROPOSAL NOT. Do not plan a sprint from §6 as it
stands.** Seven quorum rounds, never passed. §§1–5 (measurement, constraints, conflict surface)
were never challenged on substance and are the durable part. §6 has been patched seven times and
round 7 showed the patching itself is now generating defects — see §9.
**Created:** 2026-09-07
**Problem:** the cloud agent plane has no working model fallback. Every one of 34 agents
resolves to a chain of one, and the retry module that would walk a chain is unwired.

---

## 1. The measurement

Audited at `461e0949a` (v0.35.1, the release that repaired codex in prod the same day).

### 1.1 Every agent is pinned

`ResolveModelChain` short-circuits on an explicit pin:

```go
if agent.Model != "" { return []string{agent.Model}, nil }   // chain of ONE
```

In `config.cloud.yaml`, out of 34 agents:

| shape | count |
|---|---|
| `model:` only | 30 |
| **both `model:` and `role:`** | **4** |
| `role:` only (a real chain) | **0** |

The four are the pipeline: `design-doc-creator`, `sprint-planner`, `sprint-executor`,
`sprint-evaluator`. Each declares a `role:`, so each *looks* like it has a fallback chain.
The pin wins and the role never resolves.

### 1.2 The chain tail is computed and discarded

`ResolveModel` ends `return chain[0], nil`, and `ResolveModelChain` has **no other caller in
the repo** (verified repo-wide; the search finds the test callers, so the empty production
result is real). The comment directly above it reads *"The rest of the chain is no longer
discarded"*. It is discarded, one function below the sentence denying it.

### 1.3 The failure classifier is never called

`ClassifyFailure` — the transport-vs-model taxonomy carrying `status 429`, `rate_limit`,
`context deadline exceeded` — has **zero production callers**. Tests only. Nothing ever asks
whether a failure was retryable.

### 1.4 `ChainLinkIndex` is never written

Declared on `TaskRecord` (`store.go:24`) and on `pubsub.TaskCompletion` (`topics.go:64`),
**read in exactly one log line** (`stale_task_detector.go:133`), and assigned nowhere. It is
always 0, so that log always reports "chain link 1".

**Summary:** M3's data model landed; its behaviour did not. This is the same shape as the
mission-loop finding of 2026-09-05 — *a fallback chain that exists in config is not a
fallback chain that runs* — in a second place, unfixed.

---

## 2. Why this is now urgent rather than theoretical

Codex was repaired in prod on 2026-09-07 (first successful execution: `task-10410d4e`,
22342+547 tokens, $0.0382, zero 401s). The pipeline therefore has **exactly one working lane
and nothing behind it**, at the moment unsupervised ecosystem work becomes the goal.

A quota outage is the realistic trigger, and it is the case the current design handles
worst — see §4.

---

## 3. The constraint that shapes any fix: harness ≠ model

`RoleEntry` already carries `Executor` (the harness: codex / opencode / pi / motoko)
alongside `ModelName`. Resolved chains, cloud lane:

| role | [0] | [1] |
|---|---|---|
| designer | `opencode-or-kimi-k3` (opencode) | — *(chain of one)* |
| planner | `gpt5-6-sol` (codex) | `opencode-or-kimi-k3` (opencode) |
| executor | `gpt5-6-sol` (codex) | `opencode-or-deepseek-v4-flash` (opencode) |
| evaluator | `opencode-or-minimax-m3` (opencode) | `opencode-or-deepseek-v4-flash` (opencode) |

`ExecutorVariant` selects the Cloud Run Job and therefore the image, and each image installs
exactly one CLI. So **a fallback that changes harness cannot be an in-container walk** — it
needs a re-dispatch to a different Job.

Whether it changes harness is *forced*, not chosen:

- **codex agents** (`sprint-planner`, `sprint-executor`): the tail is an OpenRouter model.
  codex only talks to OpenAI. → the fallback lands on a different image.
- **pi agents** (`design-doc-creator`, `sprint-evaluator`): the tail is an OpenRouter model
  and pi drives OpenRouter. → the fallback lands on the same image.

Both are the SAME mechanism — one re-dispatch through `advance` — differing only in which
image they land on. *Round 6, gpt6-astra: an earlier draft made the pi case an in-container
chain walk, a second execution path whose attempts consumed no `AttemptCount`, carried no
`AttemptID` and persisted no `ChainLinkIndex`. The cap and the fencing would then have bounded
one path and not the other. Deleted rather than specified: a second accounting model is how the
two drift, and the whole document is about that.* The cost is one container start on the
same-image case, which is the price of every attempt being counted the same way.

### 3.1 Two pre-existing inconsistencies this audit exposed

**(a) Registry and fleet disagree on harness for two agents.**

| agent | variant → provider | role head's `Executor` |
|---|---|---|
| `design-doc-creator` | **pi** | **opencode** |
| `sprint-evaluator` | **pi** | **opencode** |

The model *string* is identical either way, which is why nothing has broken. But it means
`RoleEntry.Executor` cannot be used naively to choose a fallback container — for these two it
would silently migrate them off pi. **Open question D1.**

**(b) There is no `opencode-go` variant.** Variants are `codex`, `codex-go`, `gemini`,
`gemini-go`, `opencode`, `pi`, `pi-go`, `motoko`, `eval`, `eval-go`. `sprint-executor` runs
`codex-go` because it needs the Go toolchain; its declared fallback is an opencode model, and
falling back would **lose Go** — for a Go repo, an executor that cannot run `go test` is close
to useless. **Open question D2.**

### 3.2 The pins are redundant

Every one of the four pins is **byte-identical** to its role's chain head:

| agent | pin | role head model |
|---|---|---|
| design-doc-creator | `openrouter/moonshotai/kimi-k3` | `openrouter/moonshotai/kimi-k3` |
| sprint-planner | `gpt-5.6-sol` | `gpt-5.6-sol` |
| sprint-executor | `gpt-5.6-sol` | `gpt-5.6-sol` |
| sprint-evaluator | `openrouter/minimax/minimax-m3` | `openrouter/minimax/minimax-m3` |

So dropping the pins changes the running model for **zero** agents while making the chains
real. It is a prerequisite, not a fix: with nothing walking the chain it is a no-op, and
shipping it alone would reproduce the exact disease this doc is about.

---

## 4. The retry tier that is missing is the one quota needs

Two failure tiers were designed (M3, D4):

| class | intended response | today |
|---|---|---|
| infrastructure (container died, OOM, preempt) | one re-dispatch on the next link | re-dispatches on the **same** link (`ChainLinkIndex` never advances) |
| model/provider (429, quota, stall) | in-container chain walk | **does not exist** |

Quota exhaustion **fails fast**: codex 401/429s and exits within seconds, and the job
publishes a completion with `status=failed`. The stale-task detector — documented as the
**SOLE re-dispatcher** (V23) — only fires on `age > timeout`, so it never sees a fast
failure. The completion handler treats every completion as terminal.

**Therefore: a quota outage today ends the task, with no retry of any kind.**

The V23 sole-re-dispatcher rule is load-bearing, not incidental: a second re-dispatcher
previously produced a 591-message amplification loop. Any fix must not simply add one.

Note also `transportSignatures` contains no `quota` / `usage limit` / `insufficient_quota`
entry. OpenAI returns 429 for both rate-limit and spent-quota, so `status 429` may match by
luck; a worded quota error classifies as model-class and is not retried. Under the current
architecture that is *accidentally correct* — retrying a spent bucket on the same provider is
futile — and becomes wrong the moment a cross-provider lane exists.

---

## 5. Conflict surface — what already exists, and what this adds

*Round 4, oc-glm-5-2: the problem statement names an unwired retry module, and the proposal
then read as if new retry machinery were being bolted onto the completion handler and the
detector. The proposal does in fact wire that module — it was never said, so the doc invited
exactly this reading.* Explicitly:

| piece | today | this design |
|---|---|---|
| `ClassifyFailure` (`retry_chain.go`) | correct, zero callers | **wired**, extended with the quota class |
| `transportSignatures` | correct, unreachable | **reused**; quota signatures added alongside |
| `ResolveModelChain` | tail discarded | **wired**; tail consumed by `nextLink` |
| `ShouldReDispatch` + `AttemptCount` CAS | live, cap enforced | **reused unchanged** as the dispatch bound |
| `MaxTaskExecutions = 2` | live | reused (D4 asks whether it should change) |
| `ChainLinkIndex` (record + completion) | declared, never written | **written** by P2/P3 |
| stale-task detector | sole dispatcher, age-gated | same component, **second eligibility predicate** |
| `nextLink` selector | — | **new** (class-aware target choice) |
| `AttemptID` fencing | — | **new**, and needs a schema + emitter change: `topics.go` field, dispatcher env var, `execute-job` echo (P3.5) |

Nothing in `retry_chain.go` is bypassed or duplicated. The two additions exist because no
current code can express "which link may this failure class use" or "which execution produced
this completion".

---

## 6. Proposal

**Sequencing rule: no phase may ship a lane that is known to fail.** *Round 6, oc-glm-5-2:
P2 advanced and re-dispatched while P4 (which derives the right container) was sequenced after
it, so for the whole gap a codex agent's fallback would dispatch an OpenRouter model into a
codex container the doc itself proves cannot reach OpenRouter — a guaranteed failure, shipped
knowingly.* P4's capability derivation is therefore **not a later phase**: it is part of
`advance` in P2, and P1+P2 ship together or not at all. What remains genuinely sequenceable is
P0 (the Go-capable executor tail) and P5 (visibility).

**P1 — make the chain real (prerequisite).** Drop the four redundant `model:` pins so the
roles resolve. Zero change to which model runs. Ships with P2, never alone.

**P2 — walk the chain, through ONE advancement function.** `advance(task, class) ->
(targetLink, variant, ok)` is the sole way any component moves a task along the chain. It calls
`nextLink` (class rule + existence), derives the variant by capability (P4's rule), and returns
`ok=false` when no usable target exists — which the caller must turn into a terminal failure.

**Both triggers go through it, and neither advances on its own.**

| trigger | who classifies | who advances |
|---|---|---|
| failed completion (fast) | completion handler, via `ClassifyFailure` | `advance(..., class)` |
| `age > timeout` (no completion) | detector, class is **infrastructure** by construction | `advance(..., infra)` |

*Round 6, gemini-3-1-pro: with the handler calling `nextLink` and the detector advancing to
`n+1` by itself, the selector was not the single authority the doc claimed — an infrastructure
failure on the final link would dispatch out of bounds instead of terminating (Acceptance 7),
and a fast failure already moved to link *t* would be advanced a second time. Two components
advancing under different rules is the disease this document is about.* The detector therefore
never computes a target; it asks `advance` like everyone else, and re-dispatches only what it
is handed.

**P3 — classify fast failures, and give the dispatcher a second trigger.**

*Revised after quorum round 1, which rejected the first version unanimously and correctly:
it cited §4's proof that the detector only fires on `age > timeout`, then relied on that same
detector to collect fast failures. The trigger condition has to change; saying "return it to
a retryable state" did not change it.*

The invariant V23 protects is **one component dispatches** — not *one trigger*. The 591-message
amplification came from two components each dispatching, not from a component having two
reasons to. So:

1. **The completion handler never dispatches.** On a failed completion it calls
   `ClassifyFailure` and acts on a **three-way** class, not two:

   | class | examples | response |
   |---|---|---|
   | `FailureModel` | refusal, wrong answer, code that will not compile | terminal, unchanged — a second lane gives a second opinion, not a fix |
   | `FailureTransport` | `status 5xx`, `connection reset`, idle stall | advance to link *n+1* (any provider; a retry may simply succeed) |
   | **`FailureQuota`** *(new)* | `insufficient_quota`, `usage limit`, `quota exceeded`, and `status 429` carrying a quota body | advance to **the next link with a DIFFERENT provider**, skipping same-provider links |

   *Round 3, gemini-3-1-pro — the objection that mattered most: §2 names a quota outage as the
   urgent trigger, §4 shows worded quota errors classify as model-class, and the previous P3.1
   said model-class stays terminal. The design therefore did not solve the problem it was
   written for, and deferred the contradiction to an open question. **D5 is resolved here, not
   deferred.*** Quota is neither model nor transport: a spent bucket will not clear on retry
   (unlike a rate-limit) and says nothing about the request (unlike a refusal). Its defining
   property is that **the same provider cannot succeed**, so it is the one class that must skip
   same-provider links rather than merely advancing. That is also why it could not simply be
   folded into `FailureTransport`: advancing to another codex link on a dry OpenAI bucket burns
   the attempt cap for a guaranteed second failure.
2. **The stale-task detector gains a second eligibility predicate.** Today:
   `age > timeout && AttemptCount < Max`. Adds: `status == retry_requested && AttemptCount < Max`,
   eligible **immediately, with no age gate**. It remains the sole dispatcher.
3. **The bound is the poll interval, not the timeout.** The detector ticks every **2 minutes**
   (`stale_task_detector.go:30`), so a fast failure waits ≤ one tick — not the 2h agent timeout.
   That is the answer to the stall objection; if 2 minutes is too slow for the pipeline, the
   fix is a wake signal on the same component, never a second dispatcher.
4. **The transition is guarded by the SELECTOR, not by an index bound, or the task
   terminates.** One function decides the target:

   ```
   nextLink(chain, currentIndex, class) -> (targetIndex, ok)
   ```

   It embodies the class rule (transport: next link; quota: next link whose provider differs)
   and is the single authority on whether a target exists. `retry_requested` is set **only if
   `ok`** **and `ShouldReDispatch(...)` returns true** — the existing predicate, CALLED, never
   restated.

   *Round 5, gemini-3-1-pro: the guard read `AttemptCount+1 < MaxTaskExecutions`, which is a
   fatal off-by-one. Verified in source: `AttemptCount` is "how many Cloud Run executions this
   task has consumed" (`store.go:17`), so it is already 1 during the first execution, and
   `1+1 < 2` is false at the default cap. The fallback machinery would have been dead on
   arrival — the exact "declared but never runs" failure this document exists to end, authored
   into its own fix.* The lesson is not "fix the arithmetic": it is that a second copy of the
   cap predicate is a second thing to keep in sync, so P3 calls `ShouldReDispatch`
   (`AttemptCount < MaxTaskExecutions`) instead of stating a bound of its own.

   *Round 4, gemini-3-1-pro — a genuine interaction bug between the round-2 and round-3 fixes,
   each correct alone. The guard was `n+1 < len(chain)`, but the quota rule may have to skip
   several same-provider links. If every remaining link shares the failed provider, `n+1 <
   len(chain)` still evaluates true while no valid target exists — so the task moved to
   `retry_requested` pointing at a link the detector cannot dispatch: precisely the
   uncollectible zombie Acceptance 7 forbids.* An index bound cannot express "a link this
   failure could actually use", so the bound must BE the selector. Same reason the whole
   document exists: two things that must agree, kept in two places, drift.
   Otherwise the task goes terminal-failed immediately, carrying why ("chain exhausted at link
   n" / "attempt cap reached"). *Round 2, gemini-3-1-pro: the unguarded version stranded a task
   in a non-terminal state forever whenever the chain or the cap was spent — the detector's own
   `AttemptCount < Max` gate would then ignore it permanently. A retry state that no component
   will ever collect is a zombie, and worse than a clean failure.*

4b. **The transition is ONE conditional write on `(status, AttemptID)` jointly.** Not a
   validate-then-write: the fence is the write's own precondition — "set `retry_requested` at
   link *t* **if** status is still `running` **and** `AttemptID` is still A".

   *Round 5, gpt6-astra: with a status-only CAS plus a separate `AttemptID` check, a handler
   could validate attempt A, the detector could re-dispatch as attempt B (status back to
   `running`), and the stale handler's write would then succeed against B — a superseded
   completion advancing or terminating the current attempt, defeating Acceptance 6. Any gap
   between the check and the write reopens the race; only a joint precondition closes it.*
   A second delivery of the same completion finds the task already `retry_requested` (or already
   re-dispatched and `running` on a new `AttemptID`) and is dropped as a no-op. *Round 3,
   gpt6-astra: fencing covers superseded attempts, but a duplicate of the CURRENT attempt passed
   both guards — `AttemptID` still current, task still non-terminal — and could advance the chain
   twice or terminate a queued retry as exhausted. Pub/Sub is at-least-once, so duplicates are
   the expected case, not the exotic one.*

5. **Completions are fenced by attempt — and the transport for that does not exist yet.**
   Every dispatch stamps `AttemptID` (monotonic, persisted in the same write as the status)
   into the job env; the job echoes it into its completion; the handler discards any completion
   whose `AttemptID` is not the task's current one, logging it as stale.

   **Required, not assumed — three concrete changes:** (a) add `AttemptID` to
   `pubsub.TaskCompletion` (`topics.go`); (b) pass `AILANG_ATTEMPT_ID` in the dispatcher's env
   block alongside `AILANG_PROVIDER`; (c) have `execute-job` echo it into every completion it
   publishes, including the failure paths and the deferred completion guard.

   *Round 5, oc-glm-5-2: the previous text asserted the completion "carries it back on
   `pubsub.TaskCompletion`". Verified: `grep Attempt internal/pubsub/topics.go` returns
   **nothing** — the field does not exist, and a job does not emit arbitrary env into its
   completion. The doc had cited `topics.go:64` for `ChainLinkIndex` and then assumed the same
   for a field it never checked, which made the one primitive the design calls unavoidable also
   unimplementable as written.* Cited evidence for one field is not evidence for its neighbour. *Round 2, gpt6-astra: a terminal-status
   guard cannot reject a late completion once the task is deliberately non-terminal again, and
   a dispatch-side `AttemptCount` CAS says nothing about which execution produced a given
   completion. Without fencing, a slow failure from link n can terminate — or advance — a task
   already running link n+1.* This is the one genuinely new primitive in the design, and it is
   required: `ChainLinkIndex` alone cannot distinguish two executions on the same link.

Add quota signatures **only once P4 gives them somewhere to go.**

**P4 — cross-harness re-dispatch.** Choose the fallback link's `ExecutorVariant`, and target
that Job. Applies to fallback links only (index ≥ 1); the head always keeps the agent's declared
variant, so no current behaviour changes.

*Round 7, gemini-3-1-pro: this paragraph opened by mandating derivation "from its `Executor`"
(round-2 text) and the next paragraph mandated deriving from capability "not by reading
`RoleEntry.Executor`" (round-3 text). Both were left in — mutually exclusive instructions in
adjacent paragraphs. Pure patch residue: the round-3 fix was appended without deleting what it
replaced.*

**P4 derives from CAPABILITY, not from the registry's preference.** The variant for a
fallback link is chosen by asking *can the agent's current harness reach this model?* — not by
reading `RoleEntry.Executor`:

- the agent's declared harness **can** run the link (pi or opencode driving an OpenRouter
  model) → **keep the agent's own variant** and re-dispatch to the SAME job with the new model.
  Not an in-container walk — a re-dispatch that happens to land on the same image;
- it **cannot** (codex, which reaches only OpenAI, handed an OpenRouter model) → switch to the
  variant that can, because here the switch is *forced*, not preferred.

*Round 3, oc-glm-5-2: the previous rule derived the variant from `RoleEntry.Executor`, which
§3.1(a) records as disagreeing with the fleet for `design-doc-creator` and `sprint-evaluator`.
That rule would have migrated both off pi onto opencode on first fallback — silently resolving
the disputed field in the registry's favour, which is exactly what the no-silent-fallbacks
axiom forbids. The claim that 3.1(a) was "not silently resolved" was false as written.* Under
the capability rule those two agents stay on pi, which drives their OpenRouter fallbacks
directly, and **D1 stops being load-bearing**: the registry/fleet disagreement no longer changes
any dispatch.

**P4 is blocked on P0 (below) for `sprint-executor`.** *Round 2, oc-glm-5-2: §3.1(b) concedes
that its only declared fallback is an opencode model in a container with no Go toolchain, and
the doc then proposed shipping anyway. For a Go repo that is not a degraded lane, it is a
broken one — and this is the agent most likely to exhaust quota under heavy pipeline work. It
would have shipped a fallback that satisfies acceptance 1's letter and fails its intent.*

**P0 — give the executor role a fallback that can actually build (prerequisite for P4).**
Either add an `opencode-go` image (mirroring `codex-go`/`pi-go`, which already exist for
exactly this reason) and point the executor tail at it, or give the `executor` role a
Go-capable tail on an existing image. Until one lands, `sprint-executor` keeps a single lane
and P4 ships for the other three agents only — stated, not silently.

**P5 — degradation must be visible.** A run that completes on link 1 is not the same result
as one on link 0. Surface the link in the completion and the chain view; a silent downgrade
to a cheaper model is a measurement bug in waiting.

---

## 7. Open questions for review

- **D1. — NO LONGER LOAD-BEARING (round 3).** The capability rule in P4 keeps
  `design-doc-creator` / `sprint-evaluator` on pi, so the registry/fleet disagreement changes no
  dispatch. It remains a real inconsistency worth tidying, but it no longer gates this work.
- **D2. — RESOLVED to a prerequisite (round 2), no longer open.** `sprint-executor` needs Go
  and its only declared fallback loses it. This is now **P0**, blocking P4 for that agent.
  What remains for review is only *which* form P0 takes: a new `opencode-go` image, or a
  Go-capable tail on an existing one.
- **D3.** `designer` is a one-element chain, so `design-doc-creator` has no fallback even
  after P1–P4. Add a tail, or accept single-lane?
- **D4.** Should `MaxTaskExecutions = 2` still hold when the second attempt is a *different
  provider*? The cap was set when a retry meant the same lane twice.
- **D5. — RESOLVED IN THE DESIGN (round 3), no longer open.** Quota is its own third class
  (P3.1): it must advance to a *different provider*, because a spent bucket cannot clear on
  retry. Distinguished from a rate-limit by body/wording, not by the bare `429` status.

---

## 8. Acceptance

1. A codex agent whose provider returns 429/quota completes on its fallback lane, in a
   container that can run it, within `MaxTaskExecutions`.
2. `ChainLinkIndex` is non-zero in the banked record for such a run — currently unreachable.
3. The link used is visible without reading logs.
4. A drought can be simulated deterministically (the mission loop's `MISSION_DRY_RUN=1` with
   a dead pin is the precedent) — the failure must not require waiting for a real outage.
5. Exactly one component re-dispatches (V23 preserved), provable by test.
6. A completion from a superseded attempt is discarded, not applied — provable by replaying a
   link-n completion at a task already running link n+1.
7. A task whose chain or attempt cap is exhausted reaches a terminal state, never
   `retry_requested`. No task can sit in a retry state no component will collect.
8. `sprint-executor`'s fallback lane can run `go test`, or it is documented as having no
   fallback lane — never a lane that dispatches and cannot build.

---

## 9. Review history

**Round 1 (2026-09-07) — BLOCKED, 3/3 reject.** `gpt6-astra`, `gemini-3-1-pro` and
`oc-glm-5-2` independently raised the same objection: P3 was self-contradicting. §4 proves the
stale-task detector fires only on `age > timeout` and therefore never observes a fast failure;
P3 then relied on that detector to collect fast failures. Marking a task "retryable" does not
alter the predicate that decides eligibility. Two reviewers added the consequences — an
unbounded stall until the timeout, and no specified bound against duplicate dispatch.

The objection was accepted in full, not argued. P3 now changes the detector's **eligibility
predicate** explicitly, states the bound (one 2-minute poll tick, not the 2h timeout), and
names the existing `AttemptCount` compare-and-set as the duplicate-dispatch bound. The
distinction that was missing: V23 requires one dispatching *component*, not one *trigger*.

**Round 2 (2026-09-07) — BLOCKED, 3/3 reject, all objections new.** Round 1's contradiction was
accepted as fixed; the reviewers moved to deeper ground and found three independent defects:
no attempt-scoped fencing (a late completion from link *n* could terminate or advance a task
already on link *n+1*); an unguarded `retry_requested` transition that strands a task forever
once the chain or attempt cap is spent; and P4 shipping a Go-less fallback for the one agent
that most needs a working one, which the doc had itself flagged and then deferred.

All three accepted. The fixes are P3.4 (guarded transition or terminate), P3.5 (`AttemptID`
fencing — the one new primitive, and unavoidable, since `ChainLinkIndex` cannot distinguish two
executions on the same link), and P0 (a Go-capable executor fallback, promoted from open
question D2 to a prerequisite that blocks P4 for `sprint-executor`). Acceptance criteria 6-8
make each falsifiable.

**Round 3 (2026-09-07) — BLOCKED, 3/3 reject, all objections new again.** The most serious came
from gemini-3-1-pro: the design did not solve the problem it opens with. Quota was named as the
urgent trigger in §2, shown to classify as model-class in §4, and model-class was specified as
terminal in P3.1 — so a quota outage still ended the task, with the contradiction parked in
open question D5. gpt6-astra found that attempt-fencing did not cover a duplicate of the
*current* attempt, which at-least-once delivery makes routine. oc-glm-5-2 showed P4's derivation
rule silently resolved the §3.1(a) registry/fleet disagreement in the registry's favour, while
the doc claimed it did not.

All accepted. Quota becomes a third failure class that skips same-provider links (D5 closed in
the design); the retry transition becomes a state-conditional CAS so duplicates are no-ops; and
P4 now derives the fallback variant from **capability** — can this harness reach this model —
rather than from the registry's preference, which keeps both pi agents on pi and closes D1 as
non-load-bearing.

**Author's note on the pattern.** Three rounds, and the blocking defect was the same shape each
time: a problem correctly identified in the analysis and then quietly depended upon, or
deferred, in the proposal — the detector's trigger (R1), the Go toolchain (R2), quota
classification (R3). The analysis sections were not the weak part; the join between analysis
and proposal was. Worth carrying into the next design doc rather than rediscovering.

**Round 4 (2026-09-07) — BLOCKED, N-1 (gpt6-astra ABSENT on budget; recorded as degraded, not
as a pass).** gemini-3-1-pro found an interaction bug between the round-2 and round-3 fixes:
the termination guard tested an index bound (`n+1 < len(chain)`) while the quota rule may skip
several same-provider links, so a chain whose remaining links all share the failed provider
would pass the guard with no dispatchable target — the zombie state Acceptance 7 forbids.
Fixed by making the class-aware selector `nextLink` the single authority on both *which* link
and *whether one exists*; an index bound cannot express the question.

oc-glm-5-2 objected that the design ignored the existing retry module. The proposal did wire
it — `ClassifyFailure`, `ShouldReDispatch`, the `AttemptCount` CAS and the chain resolver are
all reused — but never said so. Added §5, a conflict-surface table separating reused from new,
and justifying the only two additions.

**Round 5 (2026-09-07) — BLOCKED, 3/3 reject, full panel (budget raised so gpt6-astra was not
absent).** Two objections were that the doc asserted what it had not verified, and both were
right when checked against source. gemini-3-1-pro: the transition guard `AttemptCount+1 <
MaxTaskExecutions` is a fatal off-by-one — `AttemptCount` is executions *consumed*
(`store.go:17`), so it is 1 during the first run and `1+1 < 2` is false, meaning the fallback
would never fire. Fixed by CALLING the existing `ShouldReDispatch` rather than restating its
bound. oc-glm-5-2: `AttemptID` does not exist on `pubsub.TaskCompletion` — grep returns
nothing — so the design's one unavoidable primitive was unimplementable as written; now stated
as three concrete changes (schema field, dispatcher env, job echo). gpt6-astra: a status-only
CAS plus a separate fence check is a TOCTOU; the fence must be the write's own precondition on
`(status, AttemptID)` jointly.

Two of these are the same author error in a new costume: citing evidence for one field
(`ChainLinkIndex` at `topics.go:64`) and assuming its neighbour, and writing a fresh copy of a
predicate that already existed. Both are the failure this document is *about* — a second
definition that drifts, and a declared mechanism that never runs — reproduced inside its own
proposal.

**Round 6 (2026-09-07) — BLOCKED, 3/3 reject, and the objections went back UP a level.** After
two rounds of implementation-grain defects this round returned three structural ones, all with
a single root cause: the design had grown **multiple paths doing one job under different
rules**. The detector advanced by itself while the handler used the selector (gemini-3-1-pro);
the pi case walked the chain in-container with no `AttemptCount`/`AttemptID`/`ChainLinkIndex`
accounting while the codex case re-dispatched with full accounting (gpt6-astra); and P2 shipped
a dispatch path before P4 taught it which container to use, guaranteeing failure in between
(oc-glm-5-2).

Fixed by collapsing all of it to one function: `advance(task, class)` is the only way any
component moves along the chain, both triggers call it, the in-container walk is **deleted**
rather than specified, and P4's capability rule moves inside P2 so no phase ships a lane known
to fail.

**Stopped here at Mark's instruction (round 6 was the agreed limit). NOT converged** — the doc
is materially better but has never passed, and the honest read is below.

**Round 7 (2026-09-07) — BLOCKED, 3/3 reject. STOPPING POINT, and the reason is the objections'
provenance, not their count.** Two of the three were *caused by round 6's own fix*:

- **gpt6-astra:** unifying *who* computes the target did not make it applied *once*. The handler
  advances the stored index and writes `retry_requested` at link *t*; the detector then calls
  `advance` again and seeks *t+1*. On a two-link chain the handler moves 0→1 and the detector
  seeks link 2, terminates as exhausted, and the single fallback never runs. Round 6 unified the
  selector; it did not specify a pending-target or consume-without-advancing transition.
- **oc-glm-5-2:** routing both triggers through `advance(task, class)` created a parameter the
  detector cannot supply — it never observed the failure, and P3.5's schema work adds only
  `AttemptID`, not the class. The round-6 restructure made the design uncallable on one of its
  two paths.
- **gemini-3-1-pro:** P4 opened with the round-2 derivation rule and immediately contradicted it
  with the round-3 rule, both left in place. Patch residue, fixed above.

**Assessment.** Rounds 1–5 found defects that pre-existed the review. Round 7 found defects
*introduced by round 6*, plus a contradiction from layering fixes without removing what they
replaced. That is the signature of patching past its useful point: the document is accumulating
scars faster than it is resolving objections.

**Recommendation:** keep §§1–5 — the measurement is verified, cited, and unchallenged across
seven rounds, and the pins-are-redundant and harness-disagreement findings are independently
useful. **Re-derive §6 from scratch** against the constraints now understood (single
application per failure, class must be persisted or re-derivable by every caller, no phase
shipping a known-broken lane, one accounting model). A fresh derivation by an author without
seven rounds of investment in this particular shape is more likely to succeed than an eighth
patch by this one.
