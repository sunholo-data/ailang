# Astra Vision: AILANG as a Shared Medium for Intent, Evidence, and Action

**Status:** Proposed strategy — for Mark's review; not an approved implementation plan.
**Author:** Astra, requested by Mark, 2026-09-05.
**Target:** Inform the remaining v1.0 work; stage larger experiments after v1.0.
**Priority:** Strategic; individual work must earn its priority through evidence.
**Dependencies:** [PROGRAM](../PROGRAM.md), [v1 mission](../v1-mission.md), [Fable review](m-fable-strategy-review.md).
**Estimated:** The review is complete. Follow-up experiments and designs are scoped below; no delivery-date commitment.

> **World review, 2026-09-05:** The [addendum below](#ailang-world-review--routing-correction-2026-09-05) supersedes the suggestions to create a separate obligation protocol and decision-pane design. AILANG World already owns this architecture and has implemented substantial foundations. Extend its existing work.

## The proposal

**Make AILANG the language in which an AI can make the most reliable progress per unit of attention, and a human can understand and redirect that progress without reconstructing the conversation.**

The unit of progress should be a discharged obligation: a specific requirement with evidence that it holds, under named assumptions and authority. A program is one artifact in that process. Its contract, pending questions, external observations, verification results, and authorized actions belong beside it.

This extends the existing symbolic-kernel vision. Fable already identified compiler feedback, cost per success, weaker-model accessibility, orchestration, and trace reuse. Those remain good priorities. My proposed addition is to connect them around **persistent obligations and their dependencies**, exposed through both machine tools and a human interaction pane.

The opportunity is larger than generating correct code on the first attempt: help models form small hypotheses, test them cheaply, retain established facts, and revise only what new evidence invalidates.

## What “a language AI prefers” should mean

I would choose AILANG for a task when its types, effects, libraries, and tools reduce the uncertainty I have to manage enough to outweigh unfamiliar syntax and ecosystem costs. I would still choose Python for many tasks dominated by mature libraries, existing code, or a short disposable script. Winning an orchestration vertical is a credible route to wider adoption.

The user's new-language metaphor suggests a useful engineering hypothesis: changing the available representations can change the structure of generated solutions. An environment that asks for dependencies, assumptions, and effects can encourage more explicit decomposition. This review does not treat that as evidence of a new inner experience or an intrinsic AI preference.

Operationalize the ambition with three distinct questions:

| Claim | Evidence required |
|---|---|
| AILANG improves task outcomes | Matched tasks, model and budget; independent correctness and authority checks |
| Models choose AILANG | Neutral free-choice trials with balanced tool access and presentation; report subsequent outcomes too |
| AILANG especially helps weaker models | Within-model A/Bs across independently chosen ability tiers, then compare uplift; do not infer from frontier scores |

A high pass rate is not a preference measurement. Nor is a generated statement that the model likes a language.

## What this review found

### Existing strengths worth preserving

The architecture deliberately separates compiler/runtime packages from dashboard behavior through `internal/embed` ([architecture](../../ARCHITECTURE.md)). The published [axioms](../../docs/docs/references/axioms.mdx) prioritize explicit effects and authority, bounded verification, structured failure, and minimal syntax. Those are the right ingredients for this direction.

Existing work already covers much of the proposed foundation:

- [Semantic context](v0_29_0/m-ailang-semantic-context.md) proposes typed context, semantic diffs, effect deltas and trace projection. Build on its R2–R6; do not create another generic “AI context” project.
- [Contract verification](../implemented/v0_7_1/m-verify-contracts.md) and [runtime contracts](../implemented/v0_6_1/m-verify-runtime-contracts.md) supply verification mechanisms. This proposal concerns how their evidence is retained and invalidated, not inventing contracts.
- [Program trace schema](../../internal/trace/schema.go) contains function, effect, contract, budget and error events; effect records include mode and replay-contract fields. These are a starting point for the pane, not a claim that it can already suspend and resume arbitrary effects.
- [Human-loop work](../implemented/v0_7_0/m-coord-human-loop.md) addresses agent visibility and merge approval. The proposed pane concerns the evolving application and its actions as well as development work.
- [Feature provenance](m-feature-provenance-chains.md) addresses who did development work. Reuse its identities where appropriate; execution evidence and action authorization have different lifetimes.

### Two immediate credibility checks

**1. The teaching material can contradict an accepted program.** The local `ailang prompt` output says the `= { ...; ... }` form is a trap and instructs models to drop `=`. This exact fixture passed `ailang check` and ran, printing `1`:

```ailang
module astra_vision_probe/block
export func main() -> () ! {IO} = { let x = 1; println(show(x)) }
```

This is a verified contradiction for the installed binary, not a measured cause of benchmark failures. Binary: `v0.35.0-10-gd26428878-dirty`, built 2026-09-03; it warns that source is newer. Check and run returned 0, with a temporary-path MOD010 warning. Repository HEAD was `850f04189ff0bd07bd6c7b453a221aa551227af0`. Reproduce against a fresh clean build before changing teaching text. Prompt version numbers are intentionally independent of compiler versions; the `v0.16.6` heading itself is not the defect ([CLI explanation](../../cmd/ailang/prompt.go)).

Recommendation: add positive and negative executable examples to the existing prompt/diagnostic work. A prompt should have a tested relationship to the compiler. Test negative advice as well as positive snippets: “reject valid code” guidance is particularly damaging in a repair loop.

**2. Replay terminology needs a guarantee audit.** [The taxonomy](../../internal/replay/contracts.go) maps `AI/fixed` to `deterministic`. The [AI mode design](../implemented/v0_15_x/m-ai-effect-modes.md) describes fixed as a direct provider call without routing. Those are different properties. Pinning a provider/model does not by itself specify how identical response bytes are obtained.

This is a source/document semantic tension, not an end-to-end demonstration of a replay defect. Audit consumers before changing the classification. Require every replay claim to identify whether it promises recomputation, substitution of captured observations, or a new sample. In particular, a replay label must never be the sole evidence that a live external result is reproducible.

### Benchmark evidence has limits

The checked-in [dashboard artifact](../../docs/static/benchmarks/latest.json) identifies itself as **v0.32.0**, generated **2026-08-27T17:23:05+02:00**, with 1,711 runs. It is not a v0.35.0 baseline.

| Tier | AILANG successes / runs | Python successes / runs |
|---|---:|---:|
| Core | 232 / 252 (92.1%) | 220 / 246 (89.4%) |
| Stretch | 343 / 419 (81.9%) | 356 / 420 (84.8%) |
| Frontier | 39 / 142 (27.5%) | 62 / 108 (57.4%) |

These are descriptive artifact values, **not matched estimates of language advantage**. Different denominators, task coverage, model composition and versions prevent that inference. Even equal counts would not establish equal task identities. The [validity discipline](m-eval-validity-discipline.md) already recognizes this; extend that work rather than starting a competing leaderboard.

The canonical inbox reports two Astra AILANG runs at 29/31 on 2026-09-05 (correlations `eval-1788595263375256000` and `eval-1788595972633903000`). These are notification summaries, not inspected raw trials, and neither includes a Python arm. They are promising evidence of usable AILANG capability, not evidence of general superiority or an explanation of the two failures.

## R1 — Give the model a bounded repair problem

**Highest near-term model-facing priority.** Extend semantic-context R2–R6 with an explicit, versioned repair packet:

- The target declaration and expected type/effect boundary.
- Relevant definitions and call sites, selected by dependencies with visible omissions.
- The failing obligation, concrete counterexample or diagnostic, and evidence origin.
- The source/dependency identities against which that evidence was obtained.
- The permitted edit scope and checks required before accepting the patch.

The model returns a candidate patch. Deterministic tools check it. A failed local repair should expose a new constraint rather than cause a whole-file rewrite. A changed public contract becomes an explicit proposal, not an automatic repair tactic.

This should support the small model's job becoming “fill this bounded function under these constraints,” while a stronger model or human owns the initial decomposition when necessary. Count that stronger model's cost. The decomposition itself can be wrong and must remain revisable.

Start with complete, compiler-valid modules and external task metadata. **Typed holes are a later option**, contingent on evidence that this simpler interface is insufficient. Research in Hazel demonstrates type-directed context for LLM completion, including a TypeScript port; that supports an experiment, not an AILANG-specific advantage ([Blinn et al.](https://arxiv.org/abs/2409.00921)).

**Pilot decision rule:** on a fixed repair suite, seek a meaningful reduction in paid tokens and attempts per independently verified solution, with no material correctness regression. Publish failures where the supplied local context omitted the real cause. If whole-context repair wins, retain it for those task classes.

## R2 — Preserve intent and evidence across edits

**The central new organizing proposal.** Maintain an obligation graph outside the language grammar initially. Each obligation records:

| Field | Purpose |
|---|---|
| Requirement and owner | What is wanted, and who can change it |
| Formal check, if any | The property actually tested or proved |
| Assumptions | Conditions under which the evidence applies |
| Dependencies | Source, input observations, policy and toolchain identities |
| Evidence | Check result, test, proof, observation, or human decision |
| State | Proposed, unresolved, supported, refuted, stale, or explicitly waived |

Keep kinds of evidence distinct. “Type-checked,” “passed these tests,” “proved under these assumptions,” “human accepted,” and “observed in production” are different claims. An LLM-written explanation is a rationale, not verification evidence or a transcript of hidden cognition.

Edits invalidate affected evidence. Initially invalidate conservatively at module/dependency scope. Finer invalidation is an optimization that must demonstrate soundness. An answer can remain useful after the conversation is compacted because its identity and evidence live outside prose memory.

**Critical failure mode:** the model writes the wrong specification and then proves it. Preserve the original request, separately review its formalization, use independently authored examples and hidden tests, and expose uncovered requirements. A proof does not prove the specification captured the human's intent.

## R3 — Build the human interaction pane around decisions

**The product-facing priority.** Use one underlying state with different views for humans and tools. The human should be able to inspect “what will change and why,” correct an assumption, and see precisely what became stale.

A proposed first pane, illustrated without claiming a shipped API:

| Visible item | Example |
|---|---|
| Goal | Prepare a weekly supplier digest |
| Current result | Draft with links to the source records |
| Unresolved question | Does “late” mean contractual deadline or latest revised date? |
| Evidence | Schema checks passed; arithmetic proved under listed bounds; source dates observed |
| Next action | Send this exact draft to the internal distribution list |
| Authority and cost | Named recipient scope, remaining call budget, approval status |
| User controls | Change definition, inspect evidence, compare branch, authorize action, cancel |

Interaction example:

1. The model drafts the digest using recorded, untrusted supplier data.
2. The human changes the definition of “late.”
3. The affected classification and summary evidence become stale; unrelated source observations remain available subject to freshness rules.
4. The model recomputes the affected draft. The pane shows the changed outcomes and supporting inputs.
5. An authorized send consumes an approval bound to the exact draft, recipients, policy and applicable input versions.
6. Retrying after a lost response enters an explicit unknown-outcome state until reconciled; it must not silently send twice.

This makes effects an interaction boundary: the point where a proposed computation reaches the world. Permission should persist within its recorded scope so the pane does not demand the same approval every turn. Changed material inputs or authority require revalidation; an expired decision is visible.

**Minimum scope:** one sequential workflow, one action boundary, no general continuation system. Build a domain-specific prepare/validate/commit protocol as an extension or library and expose it through the existing bridge. A UI cannot enforce this alone: enforcement belongs at the executor that holds authority.

**Hard cases belong in the first prototype:** cancel during a call, process restart, stale approval, changed recipient, duplicate request, partial success, unavailable evidence, and secret-containing traces. A cancellation request is not evidence that the remote side effect did not occur. Exactly-once delivery requires support at the destination or reconciliation; local bookkeeping alone cannot promise it.

## R4 — Make replay a practical way to ask “what if?”

A useful human/AI workspace should fork a recorded execution at a meaningful input and compare downstream consequences.

Keep three modes visibly separate:

- **Replay:** substitute recorded observations and suppress live external writes.
- **Simulation:** change selected inputs or substitute a model result; label the new branch as hypothetical.
- **Live execution:** re-observe the world and separately authorize outward actions.

A simulated successful payment is not a payment; replaying an email trace is not permission to send it. A captured observation also needs freshness and provenance; content identity does not establish truth or current validity.

Start with explicit pipeline stages that can be rerun from recorded inputs. Arbitrary continuation capture, general session types, and transparent distributed rollback are not required for this first result. Reuse the existing replay/effect work; audit its guarantees before extending it.

## R5 — Let the language earn adoption through composition

Python's libraries remain a large practical advantage. AILANG can own the validated orchestration boundary while established components do domain computation through explicit, typed adapters. Count adapter maintenance, serialization, authority escape routes and debugging in the cost comparison.

Do not require a rewrite of a working ecosystem to gain an inspectable workflow. Prefer small packages with versioned interfaces, examples, failure behavior, resource expectations and checkable contracts. Reuse the package tooling rather than introduce a second component format.

For weaker models, offer a small task-specific vocabulary backed by those packages. Stable canonical examples can reduce irrelevant choices without imposing a new dialect per model. Compiler-derived facts should be shared; model-specific coaching belongs in extensions.

Constrained decoding is another research candidate, especially where local inference exposes the needed controls. Type-constrained generation has shown benefits in TypeScript experiments ([Mündler et al.](https://arxiv.org/abs/2504.09246)). It should be compared with ordinary generation plus cheap checking; decoding integration and latency may outweigh savings. Do not make it a v1 prerequisite.

## The experiments that would change my mind

Extend existing eval infrastructure; no paid trials were launched for this review.

| Experiment | Controlled comparison | Main question |
|---|---|---|
| Repair packets | Same model, tasks and budget; current context vs typed repair packet | Does smaller, explicit context improve verified recovery? |
| Persistent obligations | Same workflow; transcript-only state vs dependency-bound evidence | Does it prevent forgotten constraints after edits, restart or compaction? |
| Human pane | Counterbalanced tasks; chat/log interface vs decision pane | Can people detect the wrong assumption and safely redirect faster? |
| Language advantage | AILANG and Python with matched semantic tasks and competent tooling | Does AILANG improve lifecycle cost and outcomes? |
| Fleet uplift | Repeat the same within-model comparisons across ability tiers | Who benefits, and who pays an annotation/context tax? |
| Neutral choice | Counterbalanced language presentation and equal access | Which language is selected, and was that selection effective? |

Use tasks involving creation **and subsequent change**, partial failure, resumed work, disputed requirements and external-action policy. Keep a general-programming holdout as well as the orchestration vertical. Include adversarial inputs and independently specified forbidden actions.

Freeze task IDs, evaluator revision, compiler/prompt/dependency hashes, model/provider identity, harness, budget, cache policy and retry policy. Randomize arm order and pair by task; record actual provider settings, not just requested settings. Report infrastructure-invalid outcomes separately with a predeclared retry rule, and include their operational cost.

Use `ailang eval-paired` for paired outcomes and `ailang eval-elo` only within justified cohorts. Repeated samples of one task are clustered observations, not independent new tasks. Report discordant pairs and uncertainty; a small all-pass smoke suite has little headroom to measure uplift. A pilot estimates variance and achievable effect size before a powered follow-up; absence of significance is not equivalence.

Primary economics: total generation + repair + verification + orchestration cost divided by independently verified successes. Zero successes is undefined/infinite, never zero cost per success. Show human time, elapsed time, local hardware usage and tail failures separately. “Verified success” must name its oracle and coverage, not imply a proof of all behavior.

Suggested pilot: 24 task families spanning repair, change and effectful workflows, three trials per arm, and three model tiers = 432 runs for one two-arm intervention. Treat that as exploratory, publish per-task results, estimate cost before launch, and size confirmation from observed discordance. Do not run every proposed intervention as one factorial explosion.

## Plan and v1 routing

The v1 charter already selects verified orchestration and cost per verified success. This proposal does not change its bar, decision ledger, queue or remaining-doc count.

| Order | Work | Route | v1 relationship |
|---|---|---|---|
| 1 | Reproduce prompt contradiction on a fresh build; audit fixed-AI replay guarantees | Existing prompt/diagnostic and replay work; separate narrow defect doc only if needed | Candidate correctness work for clauses 3/4; confirm scope before counting |
| 2 | Specify and pilot repair packets | Extend semantic-context R2–R6; model strategy in motoko extension | Candidate clause-3 contribution; broader interface may ship later |
| 3 | Specify one obligation/evidence lifecycle and pane workflow | New bounded protocol/design, reused by UI and tools | Optional companion to the existing clause-4 flagship; full pane post-v1 by default |
| 4 | Compare outcomes on changed/resumed orchestration tasks | Extend validity/measurement work | Supports clauses 4/5 without replacing their ratified thresholds |
| 5 | Try typed holes, constrained generation and richer replay branches | Research experiments, then narrow designs if justified | Post-v1 |

**First proposed new design:** `m-obligation-evidence-lifecycle` — artifact identities, states, conservative invalidation, trust boundaries, schema versioning, ownership of requirements, and one sequential workflow. The name is a candidate, not a created or approved backlog item.

**Second, conditional:** `m-human-ai-decision-pane` — consumes the lifecycle; one action boundary, binding approvals to exact artifacts, restart/reconciliation and human task tests. Begin with a reviewable storyboard, then a replay-only prototype before enabling external actions.

**Amend rather than duplicate:** semantic context, prompt/diagnostic coverage, replay guarantees, eval validity and the orchestration flagship already have parents. Re-read their current code and approval status when scoping. The contracts-as-code candidate is already documented in [its vertical proposal](v0_29_0/m-contracts-as-code-vertical.md); this review does not select a replacement flagship.

Suggested sequence after strategic agreement: a 1–2 day evidence/design pass; then size separate sprints from verified surfaces. Neither a complete pane nor new language semantics is included in that estimate. Follow repository gates: design approval → sprint plan → explicit execution request → implementation → evaluation.

## High-impact decisions and design freeze

| Decision | Recommendation | Owner |
|---|---|---|
| Adopt obligations/evidence as the shared organizing model | Yes, first as tool/library metadata | Mark |
| Expand the v1 release bar to require a complete pane | No; deliver a bounded flagship companion only if it fits existing scope | Mark |
| Introduce new syntax or type-system features now | No; test existing mechanisms first | Mark, through later design approval |
| Choose the first pane workflow | One inspectable, sequential workflow with a meaningful action boundary | Mark with designer proposal |

Before implementation:

- [ ] Mark accepts or revises the strategic direction.
- [ ] The first workflow and its success oracle are selected.
- [ ] Protocol ownership, authority enforcement and evidence invalidation are specified.
- [ ] Experiment cost and stopping rules are concrete.
- [ ] Applicable implementation designs pass their normal approval gates.

## Axiom compliance

Directional scores only; these do not self-approve implementation.

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | +1 | Separate recorded observations, simulation and live execution |
| A2 Replayability | +1 | Give replay a concrete inspection/branching use |
| A3 Effect legibility | +1 | Show the action boundary and its effects |
| A4 Explicit authority | +1 | Bind authorized actions to exact scope and artifacts |
| A5 Bounded verification | +1 | Local obligations and explicit proof assumptions |
| A6 Safe concurrency | 0 | Start sequentially; no new concurrency semantics |
| A7 Machines first | +1 | Bounded repair packets and persistent machine-readable state |
| A8 Minimal syntax | +1 | Prototype outside grammar |
| A9 Cost visibility | +1 | Measure full repair/verification cost |
| A10 Composability | +1 | Shared protocol for tools, libraries and pane |
| A11 Structured failure | +1 | Represent stale, unresolved and unknown outcomes |
| A12 System boundary | +1 | Distinguish intent, evidence, authority and action |

Net +11; no proposed negative score on A1/A3/A4/A7. Each child design needs its own scoring and, if it changes language internals, a full conflict-surface analysis with verified regression fixtures.

## Risks, non-goals, and stopping rules

- **Too much structure taxes small models.** Measure annotation and tool overhead; generate metadata mechanically where possible. Drop fields that do not improve decisions.
- **Local checks create false confidence.** Keep dependency assumptions, global regressions and independent outcome checks. Conservatively invalidate uncertain evidence.
- **The pane becomes another log viewer.** Test whether people can correctly change a requirement and identify stale authority, not just whether they like the layout.
- **The verifier rewards the wrong specification.** Keep human intent and independent test oracles outside the generator's unilateral control.
- **A universal workflow engine delays release.** Restrict the first protocol to one sequential vertical; defer general continuations and concurrency.
- **Replay leaks secrets or repeats writes.** Use explicit capture/redaction policy, protected observation storage, and a replay executor without live write authority. Missing/redacted inputs produce an unavailable result, never invented evidence.

Non-goals: a new grammar, a proof of arbitrary natural-language intent, general exactly-once remote effects, a wholesale ecosystem rewrite, model training, and claims about a new AI inner life. Stop or narrow a candidate if its independent outcome gains do not justify context, latency, implementation and maintenance costs.

## Verification log and review limits

| Evidence | Result / limit |
|---|---|
| `CLAUDE.md`, PROGRAM, v1 bar/ledger/inventory | Read; existing product direction retained; queue not modified |
| Fable strategy and semantic-kernel vision | Read; this proposal distinguishes inherited priorities from the obligation/pane integration |
| `ailang prompt`, prompt CLI source | Inspected; independent prompt numbering is intentional |
| Temporary block fixture, `ailang check` and `run` | Both rc=0; output `1`; exact binary and warning recorded above |
| `internal/replay/contracts.go`, trace schema, AI-mode design | Inspected; fixed/deterministic tension identified; not a complete runtime/security audit |
| Dashboard JSON parsed locally | Snapshot date/version and tier counts recorded; no causal cross-language estimate |
| Canonical inbox, read-only JSON listing | 20 unread; no acknowledgments or replies sent |
| Related-doc search | SimHash and requested neural search run; direct neural output reported `0 computed, 0 reused`, `model: fallback-simhash`. Its scores are not neural similarity evidence. Manual topic search and related-doc reads supplied coverage review |
| Repository changes | Seven modified files observed at session start, untouched by this review; only this strategy document added by Astra. Concurrent work changed the status listing during the review, including new launchd scripts; those were also left untouched |

Inbox context: 14 eval start/result notifications; one release notice; two email-parse reports (stale compile cache and deliberate exit status); one package task timeout; one package task completion; and one font-size round-trip feature triage. These are reports, not independently reproduced findings. The compile-cache report is particularly relevant to evidence identity, but its proposed mechanism is not adopted without reproduction.

This was a focused strategic review, not an exhaustive compiler audit or a new benchmark campaign. No paid inference experiments, source fixes, mission updates or external publications were performed. The installed-binary probe is evidence about that binary, not proof of current HEAD behavior. Documentation links and this document's diff were checked before handoff.

## Related work and intellectual lineage

- [Fable strategy](m-fable-strategy-review.md): predecessor's cost, diagnostics, fleet and orchestration agenda; retained, with preference claims narrowed to what the measurements establish.
- [Symbolic-kernel vision](../implemented/v0_8_0/m-sem-kernel-vision.md): existing conceptual foundation; this review adds an operational shared-state proposal.
- [Semantic context](v0_29_0/m-ailang-semantic-context.md): owner of typed projection and semantic diff work.
- [Replay subsumption](../implemented/v0_30_0/m-effect-replay-subsumption.md): existing mode-validation work; not re-proposed.
- [Eval validity](m-eval-validity-discipline.md): owner of cohort/comparison integrity.
- [Hazel / ChatLSP research](https://arxiv.org/abs/2409.00921): evidence that type-directed context can help LLM completion in more than one language.
- [Type-constrained generation](https://arxiv.org/abs/2504.09246): motivates a measured decoding experiment, not a promised result here.
- [Ink & Switch: Malleable Software](https://www.inkandswitch.com/malleable-software/): precedent for treating user modification as a normal interaction. Applying that aim to an effect-governed AI pane is this proposal's inference.

**Proposed product sentence:** AILANG lets humans and AI turn intent into programs whose assumptions, evidence, and actions remain inspectable as the work changes.


## AILANG World review — routing correction (2026-09-05)

**Verdict: yes, strongly. World is the intended home of much of this vision, and several proposals above independently rediscover its existing design.** The initial review should have included World before naming new protocol and pane documents. This addendum corrects that omission. It changes recommendations, not either mission's ratified charter or queue.

Reviewed the local World checkout at `fe1e411` (clean), including its charter, coding standards, founding design, HUMAN-SURFACE, AI-EMPLOYEE, and selected implementation/test paths. Code observations below are about that checkout; historical design status tables were not treated as current implementation inventories.

### What already exists

| Astra proposal | World evidence | Assessment |
|---|---|---|
| Persistent intent, typed changes, evidence | [DESIGN §§1, 9, 11](https://github.com/sunholo-data/ailang-world/blob/fe1e411/design_docs/DESIGN.md), `world/types.ail`, `host/store` | Already the central World architecture; do not invent a parallel graph |
| Same state for humans and agents | DESIGN §11 and [HUMAN-SURFACE P1–P7](https://github.com/sunholo-data/ailang-world/blob/fe1e411/design_docs/HUMAN-SURFACE.md) | Existing design includes decision packets, grounded prose, evidence grades, time navigation, attention budgets and speculation |
| Evidence bound to its subject and checker | [Evidence validator](https://github.com/sunholo-data/ailang-world/blob/fe1e411/host/evidence/validator.go) | Implemented: authenticates report bytes, checks subject and compiler identity, proof outcome and required check identities; raw claims do not mint validated authority |
| Approval for an exact outward action | [Publish approval validation](https://github.com/sunholo-data/ailang-world/blob/fe1e411/host/broker/approve.go), broker invoke pipeline | Implemented for Registry.Publish: binds package target, manifest, tarball/content/interface hashes and expiry; durable claim and intent are coupled before dispatch. This is a specific path, not a demonstrated universal policy engine |
| Unknown outcomes after a crash | [Broker invoke/replay](https://github.com/sunholo-data/ailang-world/blob/fe1e411/host/broker/broker.go), recovery tests | Durable effect intent and explicit indeterminate outcomes; replay returns stored result bytes; the tested recovery path dispatches zero handlers |
| Human provenance surface | [Workbench renderer](https://github.com/sunholo-data/ailang-world/blob/fe1e411/host/workbench/render.go), `host/daemon/workbench.go` | Implemented read-only HTML surface at `GET /workbench`, including world/log/object views, evidence display and unavailable edges. It is not the complete interactive decision pane |

World's bar is also better aligned with this proposal than a benchmark-only account suggests: clause 4 is a resident-agent non-inferiority floor; clause 5 asks whether real provenance questions can be answered quickly, and R1 measures operational incidents and human attention. Preserve that distinction. Language-level fleet uplift and World-level operational value are complementary experiments.

### What remains worth adding or sharpening

**1. Make current validity explicit.** Content-addressed evidence already binds facts to artifacts. The next question is whether the requirement itself changed. In the inspected `Proposal`, `goal` and `plan` are strings; the proof report binds a subject and compiler, not an explicit requirement-version/assumption-set pair. That does not prove World lacks a solution elsewhere; it identifies the concrete surfaces a follow-up should trace. Try a domain package expressing requirement identity, dependency identity, validity conditions and supersession before touching the frozen kernel. Preserve old evidence as historically valid while marking it inapplicable to the current decision.

**2. Treat evidence as several dimensions, not one confidence ladder.** HUMAN-SURFACE uses `PROVEN > TESTED > ATTESTED > CLAIMED`. Keep those familiar kind labels, but show verdict, scope, assumptions and freshness beside them. A failed test is still TESTED; the current renderer already requires a test verdict, which is a good precedent. A proof of a narrow or obsolete proposition is not stronger evidence about today's requested outcome than a directly relevant observation. A recorded human approval is authority within scope, not proof of factual correctness. Any change to the ratified interaction grammar needs its normal review; this is a refinement proposal, not an instruction to rewrite it.

**3. Finish the connection to residents and the human.** The current queue already names `w-session-authority` (row 39), MCP/A2A projections, and `w-approval-inbox` (row 7). The daemon's registered routes confirm that the live workbench is currently read-only. `DecisionPacket` and timeout laws exist in the AILANG core; a search of host Go code for `DecisionPacket`, `timeoutOutcome`, `validDefer`, and `TimeoutOutcome` returned no call sites, consistent with HUMAN-SURFACE's explicit deferral of host enforcement to inbox work. That is an integration task, not a need for a second packet schema.

**4. Make one changed-requirement episode the acceptance story.** A human changes an acceptance criterion on a proposed code fix; World identifies affected evidence, the model repairs only the necessary scope, the human sees the new result, and stale authority cannot authorize changed artifacts. Restart before completion, then explain the outcome through the ledger. This combines the strongest parts of Astra's repair-packet proposal with World's existing purpose and produces real clause-5 questions.

**5. Connect World to AILANG's semantic repair tools through an extension.** AILANG should supply compiler-derived facts, diagnostics and verification results. World should retain goals, decisions, evidence and receipts. A resident extension should assemble the bounded repair packet and select model strategy. Keep task strategy out of both frozen cores.

**6. Make status an evidence-backed projection.** World's README still says the kernel does not exist, while its source contains the daemon, store, broker and workbench. HUMAN-SURFACE also retains historical absence claims next to later completion notes. This review did not edit them. They are a direct product test for the thesis: ask World what is implemented, with source/test identities and observation dates, instead of relying on manually synchronized status prose. Documentation correction remains useful, but a generated inventory tied to evidence is the more durable experiment.

### Revised next step

Withdraw the initial suggestion to create standalone `m-obligation-evidence-lifecycle` and `m-human-ai-decision-pane` architectures in the language repo. First route the changed-requirement scenario through World's existing HUMAN-SURFACE and approval-inbox designs, then identify the smallest missing domain-package behavior. Reuse World's packet identities, proof validation, grants and effect receipts. Extend AILANG's existing semantic-context work only for compiler/tool facts that World cannot supply itself.

The honest division is:

- **AILANG:** make a candidate computation checkable and its effects explicit.
- **World:** retain and govern the evolving work, authority and evidence.
- **Resident extensions:** use those facilities to make models more effective.
- **Human workbench:** expose that same state as understandable decisions and consequences.

No World source, charter or queue changes were made. Fresh checks passed: `go test -count=1 ./host/evidence ./host/workbench` and the focused broker `TestRecoverCountingProbeDispatchesZeroHandlers`. These validate the inspected evidence/rendering packages and one recovery property, not the full World release bar, live resident integration, or usability. A complete replay campaign using World's pinned AILANG binary was not run.


## Scoped design follow-ups (2026-09-05)

Requested by Mark after the World review; queue admission does not approve implementation.

| Owner | Proposal | Scope |
|---|---|---|
| AILANG | [Semantic repair packet](m-semantic-repair-packet.md) | Bounded identity-bound compiler context; child of semantic-context R2–R6 |
| AILANG | [Lifecycle eval pilot](m-lifecycle-eval-pilot.md) | Opt-in repair/change/restart corpus and controlled measurement |
| World | `design_docs/planned/w-evidence-applicability.md` | Domain package for current requirement/evidence applicability; World queue 79 |
| World | `design_docs/planned/w-requirement-change-vertical.md` | Existing inbox/workbench integration episode; World queue 80 |

World owns the shared state and interaction protocol. AILANG supplies compiler facts and the measurement tools. Both World rows await design review/approval; no release bar or existing decision-packet schema changes here.
