# Docs Mission — STATUS archive

Append-only. Newest entry at the TOP (Gate 4 moves the 4th-newest charter stamp here, per
`docs-mission.md`'s rotation rule). The charter itself always carries the newest 3.

## STATUS 2026-08-28 — ITERATION 0 RUN: quorum BLOCKED 3x across 2 revisions (1 human-directed mid-flight), **still not ratified** — parked for Mark

First unattended fire of `dev.ailang.mission-docs` (kill switch had been removed since the prior
stamp). Gate 0 found no docs-mission-specific inbox traffic and no comments on bookkeeping issue
`#953`, so proceeded per Gate 2's QUORUM-AT-PICK: this charter had no quorum artifact yet, so ran
`ailang design-quorum` on it before any routing, exactly as any picked design doc would get.

**Round 1** (`docs-mission-2026-08-28T06-29-56Z.json`, $0.057): BLOCKED, all three reviewers
rejected — two premise objections on clause 7 (no verified read command for the prod inbox;
`send`/`forward` asserted as working with zero verification evidence, unlike every other claim in
the doc) and one design objection (queue item docs-1's inbox-routing deliverable has no identified
implementation path inside the mission's own blast-radius allowlist).

**Controller measurement before revising (rule 3f — measure premises, don't just forward them):**
ran `ailang messages send docs-quorum-test-scratch "..."` then
`ailang messages forward --to docs-mission --reason "..." <id>` then
`AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list --inbox
docs-mission --unread --json`, live against prod Firestore — the message arrived and was readable,
confirming both CLI primitives work with no registration and no `internal/`/`cmd/` change involved.
Paired with a known-positive control (`pkg:sunholo/ailang_parse`, real unread message) and a
known-negative one (a fabricated inbox name, `null`) so the read instrument's emptiness elsewhere
is trusted. This refuted 2 of 3 round-1 objections directly; the third (blast-radius scope for
docs-1) is a genuine open design question, not a measurable premise.

**Revision**: spawned a pinned-sonnet designer sub-agent (independent process from this
controller's own pass verdict) with the measurements above, to make ONE bounded, targeted fix:
cite the verified commands and correct read incantation in clause 7, reframe docs-1 as building a
**trigger** (not the already-working primitives) with the blast-radius question surfaced as an
explicit, unanswered, one-word ask for Mark, and reconcile the CURRENT-GOAL/Queue inconsistency in
the Queue's favor. Confirmed scope held: clauses 1-6, Guardrails, Routing policy, and the
ARMED-BUT-SILENT/not-yet-ratified STATUS language are all byte-identical to before.

**Round 2** (`docs-mission-2026-08-28T06-35-00Z.json`, $0.062): BLOCKED again, different surface
each time — sign of a doc still being probed broadly, not close to convergence (Gate 2's
round-tracking rule). gpt5-6-sol re-rejected the SAME docs-1 scope point (correctly — it's the
genuine open question, deliberately left unanswered rather than invented). gemini-3-1-pro raised a
NEW objection calling `derive-planner-lane.sh`'s Step 0 opus-fallback a "silent fallback" violating
the no-silent-fallbacks axiom. oc-glm-5-2 raised a NEW objection that the Repo Profile's CI
path-filter claims are asserted without a workflow-file citation.

**Controller measurement on round 2 (before parking, not before a 3rd revision — Gate 2 allows one
revision + one re-quorum, both now spent):**
- `gemini-3-1-pro`'s objection is **REFUTED**: `tools/launchd/derive-planner-lane.sh` lines 57-58
  read *"Step 0: only a VETTED non-opus lane may proceed to the path analysis; anything else fails
  closed to opus"* — a deliberate, documented, LABELED fail-closed-to-safety default (every exit
  path emits a distinct reason token, e.g. `opus fail-closed:env-pin`, per lines 63-64's own
  comment explaining exactly why: *"every non-codex value emitted 'opus fail-closed:env-pin', so
  the lane would read as pinned in the driver log while actually running opus"* — i.e. this script
  was ALREADY hardened against the silent-failure shape the reviewer describes). Not a bug to fix;
  the doc's framing (route around it via pin choice) is correct.
- `oc-glm-5-2`'s objection is **PARTIALLY CONFIRMED**: `.github/workflows/ci.yml`'s `on.push` has
  no `paths:` key (lines 3-9) — the doc's "no push paths filter" claim is TRUE. But
  `docusaurus-deploy.yml`'s `on.push.paths` is **broader** than the Repo Profile states: besides
  `docs/**`, `prompts/**`, `llms.txt`, `CHANGELOG.md`, it ALSO triggers on
  `.github/workflows/docusaurus-deploy.yml`, `internal/**`, `cmd/**`, `go.mod`, `go.sum`, `web/**`
  (WASM/REPL rebuild triggers) — meaning V1's own code changes can fire this mission's Gate-3b/
  Gate-1 watched workflow too. Worth a citation fix next revision; not itself a ratification
  blocker, and it sharpens the V1/docs shared-CI-signal risk flagged in round 1's controller note.

**A directive landed MID-ITERATION and changed the disposition above.** While round 2 was being
written up, Mark reviewed the round-1/2 blast-radius objection in an attended session and committed
`29a467cac` ("widen planner allowlist tools/launchd/* -> tools/*"), authorising exactly the scope
docs-1 needed, verified in both directions before asking. Per this skill's mid-iteration-directive
rule, actioned in-iteration rather than deferred: spawned a second pinned-sonnet designer revision
folding the resolution into Guardrails/docs-1, plus the two round-2 measured fixes (the
"silent"→"deliberate" fallback wording, the full CI path-filter citation), and re-ran the quorum a
third time.

**Round 3** (`docs-mission-2026-08-28T06-49-08Z.json`, $0.078): **BLOCKED AGAIN — a third distinct
surface each round, no reviewer has passed yet.** gpt5-6-sol escalated from "no extension mechanism"
to demanding a full conflict-surface/protocol spec for docs-1 (dedup keys, retry/timeout semantics,
scheduling ownership) inside the CHARTER itself — this asks the ratification gate to absorb docs-1's
own future sprint-planning scope, which the charter's own Guardrails explicitly says most items
don't need ("prefer a Gate-2 reality-check straight into a sprint"). gemini-3-1-pro objected that
Clause 1's `audit_design_docs.sh`/`check_versions.sh` citations are unverified — **REFUTED by a
check already run earlier this iteration**: `ls .claude/skills/docs-sync/scripts/` lists both files
(plus `check_examples.sh`, `derive_roadmap_versions.sh`, `generate_report.sh`). oc-glm-5-2 raised a
meta-objection that an admittedly-unratified doc cannot simultaneously read as an operational
charter with a live queue — true of any draft under quorum review, not specific to this doc's
content, and not something a text revision resolves.

**Disposition: STOP HERE and `PARKED-needs-human-review` on `docs-0`.** Gate 2's protocol is one
revision + one re-quorum; this iteration already spent that budget AND a second bounded revision
(justified only because genuinely new information — Mark's live decision — arrived mid-iteration,
not because round 2 merely re-blocked). Per the round-tracking rule, objections have spread across
a *new* surface each round rather than localising or any reviewer starting to pass — the doc is
either still immature or the reviewers are progressively raising the bar past what a charter (vs. a
feature design doc) should need before its OWN queue items get their own design/sprint treatment.
Either reading argues for a human decision, not a fourth revision. No sprint routes this iteration
(the charter's own gate), so docs-1/docs-2/docs-3/docs-4 stay `[NEXT]`/`[PARKED]` unchanged.
Planner/executor/sprint-evaluator: N/A — no sprint executed. generator(designer)≠judge was supplied
structurally by the 3-reviewer quorum, independent of both the sonnet controller and the sonnet
designer sub-agent, across all three rounds.

**Metered cost, corrected total: $0.197** of the $1 ceiling (three quorum rounds: $0.057 + $0.062 +
$0.078). Quota: sonnet (controller + two designer sub-agent runs). Plus one unrelated, separately
justified skill fix this iteration (see Gate 5 retro): the shared skill's Gate-0 kill-switch/queue/
log-path preflight literals were V1-only and silently wrong for every other mission — fixed in
`0e341cc57`, 5th instance of the "bare `~/.ailang/state/` literal in this shared skill" class.

**Metered cost this iteration: $0.119** of the $1 ceiling (two quorum rounds, $0.057 + $0.062).
Quota buckets: sonnet (controller + designer sub-agent).

## STATUS 2026-08-28 — ITERATION 0 PENDING: charter written, **not yet ratified**, loop armed-but-silent

Charter drafted attended with Mark 2026-08-28. Bar clauses 1-7 are Mark's own selection and must
still be **ratified through the design quorum at iteration 0** before any sprint routes.

**ARMED-BUT-SILENT as of 2026-08-28 07:59.** `dev.ailang.mission-docs` is installed and
bootstrapped; the kill switch `~/.ailang/state/mission-docs.disabled` is set, so every fire exits at
Gate 0 until it is removed deliberately. The whole chain is proven live at **zero token cost**:
launchd fired the job, the plist's `MISSION_PROFILE=docs` resolved, the source clone was correctly
`ailang-docs` (so `WorkingDirectory` took effect), the driver pin advanced to latest `origin/dev` at
0 behind, the kill switch caught it, exit 0. **Iteration 0 is charter ratification, run ATTENDED** —
it is not a sprint.

**Two infrastructure defects were found and fixed *before* the loop ever ran**, both in
`derive-planner-lane.sh`, and both of the same shape: a cheap pin that reads as configured in the
driver log while an expensive model actually runs.

1. **Fail-closed to opus on every docs item.** The path allowlist was infra-only, so every docs
   design doc emitted `opus fail-closed:path-not-in-codex-allowlist` — routing the planner to the
   most expensive model in the fleet, on the mission built to avoid it. Fixed as per-mission data
   (`MISSION_PLANNER_ALLOWLIST`; default unchanged, so v1/world/motoko are byte-for-byte
   unaffected).
2. **The emitted lane dropped its model.** Step 5 emitted a hardcoded bare `codex` for any `codex:*`
   pin. Invisible on V1 by coincidence — its pin is `gpt-5.6-sol` and the consumer default is also
   sol, so the dropped value equalled the fallback. Not invisible here: this mission pins
   `gpt-5.6-luna` ($0.20/$1.20 per M) and would have planned on `gpt-5.6-sol` ($2/$10) every
   iteration. Found end-to-end, by routing a real docs doc through the *pinned* driver with the live
   mission env — not by reading the script.

Both measured with discriminating controls. `tools/launchd/test_mission_routing.sh` is at **34/34**,
with 9 new assertions covering both directions of each risk (widening must work and must not become
a hole; the lane must keep its model, asserted with two different codex models so a re-hardcoded
literal cannot satisfy both).
