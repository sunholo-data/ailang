# Docs Mission — the published AILANG website tells the truth about AILANG, concisely

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md)); advanced by a scheduled
outer loop on the always-on rig.
**North star**: A reader arriving at the website gets an accurate, current, and *non-redundant*
account of what AILANG is and does — with no page contradicting the shipped binary, no example that
does not run, and no third copy of something already said twice.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration — the SAME unforked skill every mission uses (M-MISSION-PORTABILITY). The
subject-matter skill it drives is
[.claude/skills/docs-sync/SKILL.md](../.claude/skills/docs-sync/SKILL.md), which already implements
most of clauses 1-3 and must be USED, not reimplemented (Critical Principle 1).
**Scheduling**: launchd `dev.ailang.mission-docs`, `StartInterval` 21600 (6h), behind the billing
guard. Staggered against v1 (5400s), world (14400s) and motoko (46800s) — `StartInterval` counts
from load, so the phase offset is set by when the job is bootstrapped.
**Log**: [docs-mission-log.md](docs-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue [#953](https://github.com/sunholo-data/ailang/issues/953)
(created 2026-08-28; seeded into `~/.ailang/state/mission-docs-gh-issue`; rotates weekly). Every
iteration posts its report there; driver crashes too.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/docs-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `docs` (driver: `MISSION_NAME`; every
  `~/.ailang/state/mission-docs-*` path is namespaced away from the V1 loop)
- **Checkout**: `/Users/voightkampff/dev/sunholo-data/ailang-docs` — **its own clone of this repo**.
  The V1 loop mutates this same repo every 90 minutes from a different tree; two loops in one
  working tree is precisely the concurrent-agent hazard. Separate clone = the motoko precedent.
- **Bookkeeping issue**: `#953`, rotates weekly; live number in `~/.ailang/state/mission-docs-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `Deploy Documentation to GitHub Pages` (the docs gate;
  per `.github/workflows/docusaurus-deploy.yml`'s `on.push.paths`, it is path-filtered on
  `docs/**`, `prompts/**`, `llms.txt`, `CHANGELOG.md`, `.github/workflows/docusaurus-deploy.yml`,
  and — as WASM/REPL rebuild triggers — `internal/**`, `cmd/**`, `go.mod`, `go.sum`, `web/**`. That
  last group means V1's own Go-source commits can also trigger this mission's watched deploy
  workflow, not just docs-mission's own commits), and `CI` (which runs on every push — per
  `.github/workflows/ci.yml`'s `on.push`, this repo has no push paths filter, so a docs-only commit
  still runs full CI and Gate 3b must wait for it rather than reading its absence as "not
  applicable").
- **Verify profile**: `docs-site` — a THIRD profile, because neither shipped profile fits a
  website. `go-compiler` rebuilds a Go toolchain this mission never touches; `ailang-code` treats
  the binary as the gate for AILANG source. Here the gates are:
  - `make docs-build` — the Docusaurus production build (this is the real gate; the deploy
    workflow runs the same thing, so a green local build is the cheap pre-image of Gate 3b)
  - `make verify-examples` — every published example still verifies
  - `ailang check <file>` — for any individual `.ail` touched
  - **No compile step and no dual-binary staleness class.** The `go-compiler` rules about
    `~/go/bin/ailang` vs `bin/ailang` drifting do NOT apply — but the *binary that runs
    `verify-examples` still can be stale*, so confirm `ailang --version` before quoting its output.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `docs-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Every
iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the scarcest
model budget; the append-only history lives in the log + archive.

## STATUS 2026-08-28 — **BAR RATIFIED ATTENDED**; queue reordered so the next fire touches the website

Mark closed `docs-0` by human decision after reading iteration 0's result. Two changes, both his:

1. **The bar is ratified** — not by a passing quorum, but because the seven clauses are Mark's own
   attended selection, so a quorum blocking them second-guesses the human it exists to represent.
   The objections that survived measurement were about **queue items' implementation**, not the bar;
   they are preserved and re-enter at each item's own design gate, where they are actionable.
2. **`docs-2` promoted above `docs-1`.** The original ordering was an authoring error that put
   clause-7 infrastructure ahead of every clause that changes a published page. The measurement that
   forced this: after a full iteration, **files changed under `docs/` was zero**.

**The mechanism-level lesson, routed to the shared skill rather than to this mission:** charter
ratification and design review are different jobs. Running a backlog-bearing governance document
through a design quorum blocks at the wrong gate — the reviewers keep finding new things to say
about work that has not been designed yet, so each round raises *new* objections instead of
converging. Three rounds, no convergence, **6 of 9 objections refuted by direct measurement**.

Iteration 0's real yield is kept and was never the ratification: two corrections to the
human-authored charter (the CI path-filter citation, the fail-closed framing), a refuted
"silent fallback" claim, and one **fleet-wide** skill fix (`0e341cc57`) — the shared Gate-0
Current-State block had been reading V1's kill switch, queue and log for *every* non-V1 mission.

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

## CURRENT GOAL

1. **Iteration 0 (definition)**: ratify the bar below with Mark through the design quorum, then
   score the backlog against it into `required` / `nice` / `post`. Output: an ordered queue in this
   doc. Standing up the `docs-mission` inbox-routing **trigger** that clause 7 depends on is queue
   item **docs-1** (see Queue and clause 7) — a later, separate iteration, not part of iteration 0
   itself. (Reconciled 2026-08-28 during quorum revision: this line previously said routing was
   "also at iteration 0," contradicting the Queue's listing of docs-1 as its own 1-iteration item
   after docs-0. The Queue is correct — docs-1 carries an open scope question, below, that must not
   gate ratification.)
2. **Then**: work the queue through the inner loop (design-doc → sprint-plan → execute → evaluate),
   one sprint-sized item per iteration, recording routing evidence every time.

## The bar — what "the website is up to date" means (RATIFY with Mark at iteration 0)

Mark selected all seven clauses attended on 2026-08-28; clauses 5-7 are his additions.

- **Clause 1 — Code/docs drift.** No page contradicts the shipped binary. Feature status pages match
  actual implementation, version constants are current, and design docs that moved
  `planned/` → `implemented/` are reflected on the site. The `docs-sync` skill's
  `audit_design_docs.sh` / `check_versions.sh` are the instruments — use them.
- **Clause 2 — Examples compile and run.** Every `.ail` under `examples/` still passes
  `make verify-examples`, and every site page importing one via raw-loader resolves. A published
  snippet that no longer runs is worse than no snippet: it teaches wrong syntax to both humans and
  the models we benchmark. Website examples are IMPORTED from `examples/`, never inlined — a page
  with an inline code block is itself a clause-2 defect.
- **Clause 3 — Site build health.** `make docs-build` green, `Deploy Documentation to GitHub Pages`
  passing, no broken internal links, no orphaned pages unreachable from the nav.
- **Clause 4 — New features get pages.** Shipped features with no documentation page at all are
  tracked and closed. Read `CHANGELOG.md` and the release history against the nav; the gap is the
  work. This is the only generative clause — scope one page per iteration, not a batch.
- **Clause 5 — Concision and anti-sprawl.** *(Mark, 2026-08-28.)* The site is organised and
  categorised, and says each thing **once**. Redundant pages are merged or deleted, not left to rot
  beside their replacement; overlapping guides are consolidated; the nav reflects a real taxonomy
  rather than accretion order. **Deletion is in scope for this clause** — it is the only clause
  whose work is usually *removal*, and the loop must not quietly substitute "add a clarifying note"
  for "delete the duplicate". A page that exists only because nobody dared remove it fails this
  clause.
- **Clause 6 — Benchmark report maintenance.** *(Mark, 2026-08-28.)* The published benchmark
  surface is current and honest: `docs/static/benchmarks/{latest,os/latest,os/history}.json` and the
  pages that render them. **Beware the known traps, which are documented and must not be
  rediscovered**: os-rolling data has been stale-but-plausible before (check dates *first*, a low
  broad pass-rate is usually frozen pre-fix data and not a regression); baselines are NOT poolable
  across the three local-model boundaries (07-21..08-03 untuned flags, 08-13 budgets+num_ctx,
  08-17 ollama 0.32.1→0.32.14); and v0.30.0 cost data is INVALID (reasoning tokens unrecorded).
  This clause maintains the *report*; it does not re-run or re-bank evals, and it takes no rig lock.
- **Clause 7 — Doc-related requests are answered.** *(Mark, 2026-08-28.)* The mission owns its own
  inbox and works it: doc-related GitHub issues on `sunholo-data/ailang`, and `ailang messages`
  requests routed to the **`docs-mission`** inbox.
  **This clause has no delivery mechanism (trigger) yet — building one is queue item docs-1, not an
  assumption baked into this charter.**

  **Verified before being written down** (2026-08-28, both rc=0, run live against prod Firestore,
  not simulated):
  ```
  export AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac

  # 1. send to a scratch inbox (simulating externally-originated traffic)
  ailang messages send docs-quorum-test-scratch "iteration-0 quorum premise check..." \
    --title "docs-mission premise check 20260828T063048Z" --from "docs-quorum-probe"
  # => Message sent to 'docs-quorum-test-scratch' (ID: inbox_1787898648354_20138c15)

  # 2. forward it to docs-mission — NOTE: flags must come BEFORE the message id, or you get
  #    "Error: --to flag is required"
  ailang messages forward --to docs-mission --reason "quorum premise verification" \
    inbox_1787898648354_20138c15
  # => Forwarded message from 'docs-quorum-test-scratch' to 'docs-mission'

  # 3. confirm arrival with the CORRECT canonical-store read command (see this repo's root
  #    CLAUDE.md — cited explicitly here so this clause doesn't leave it implicit):
  ailang messages list --inbox docs-mission --unread --json
  # => returned the forwarded message; acked as test cleanup
  ```
  Controls run alongside: a known-positive read (`--inbox "pkg:sunholo/ailang_parse"` returned a
  real unread message) and a known-negative read (a made-up inbox name returned `null`) — so the
  read instrument is confirmed not vacuously empty either way.

  This confirms `ailang messages send <inbox>` takes a free-form inbox name (`docs-mission` needs
  no registration) and `ailang messages forward --to <inbox> <id>` works — both **already-existing
  CLI verbs, verified with no `internal/`/`cmd/` change involved**. What is NOT built is a
  **trigger**: something that decides *when* to call `forward` for doc-related traffic sitting in
  `public-feedback`, `pkg:<vendor>/<name>`, or GitHub issues. Public feedback dispatch has never
  worked end-to-end (36/36 failures since 2026-04-27, ailang#900), so a trigger must **poll** those
  sources — it cannot assume anything currently pushes traffic anywhere. See docs-1 in the Queue for
  scope. The canonical store is prod Firestore; a bare `ailang messages list` (no env vars) reads
  local SQLite and will show an empty, reassuring, wrong inbox.

## Guardrails (mission-specific; the skill's Standing Rules always apply on top)

- **Blast radius is `docs/`, `examples/`, `README.md`, `CHANGELOG.md`, plus `tools/`
  (non-`internal/`, non-`cmd/`).** Enforced mechanically at the planner gate by
  `MISSION_PLANNER_ALLOWLIST`. Widened from `tools/launchd/*` to `tools/*` on 2026-08-28 (Mark,
  attended — commit `29a467cac`) specifically to unblock docs-1's inbox-routing trigger, which
  clause 7 makes a mandatory deliverable but which had no buildable path under the narrower list.
  `internal/**` and `cmd/**` remain denied — if an item genuinely needs a change there, that is a
  **V1 backlog item, not a docs item** — file it across and move on. Do not widen the allowlist
  further to make an item fit; the allowlist is the definition of this mission's scope, not an
  obstacle to it.
- **No designer on Fable.** The shared skill's designer rotation is
  `claude:claude-fable-5 → pi:ollama/deepseek-v4-flash:0731-cloud` (kimi removed fleet-wide
  2026-08-28, V1 charter D-48); this mission seeds the rotation at sonnet (see Routing policy) and
  must not fall to the Fable entry. Fable is reserved for high-cognition spec synthesis, and docs
  items are drift repair. Most items here need **no design doc at all** — prefer a Gate-2 reality-check straight
  into a sprint. If an item truly needs deep design, that is a signal it belongs in V1.
- **Deleting published pages is an outward-facing change.** Clause 5 makes removal routine, which
  makes it dangerous. Before deleting or merging a page: check inbound internal links, and leave a
  redirect where the URL was public. "Nothing links to it" must be *measured*, not assumed — an
  empty search is a claim, not a fact.
- **Never re-run or re-bank evals.** Clause 6 maintains the report only. This mission takes **no
  `rig.lock`** and must never start GPU work; the nightly eval rotation and three sibling missions
  share that hardware.
- **Verify before publishing a syntax claim.** Any AILANG syntax that lands on the website must
  pass `ailang check` first. A wrong claim in published docs is worse than no claim: it trains both
  readers and the models we benchmark against.

## Routing policy

Uses the **shared** per-role routing from `mission-control`, with this mission's overrides in
`~/.config/ailang/mission-docs.env` (versioned copy:
[tools/launchd/mission-env/mission-docs.env](../tools/launchd/mission-env/mission-docs.env)).

The ladder is ordered by **cost type, not model quality** (Mark, attended 2026-08-28): spend the
subscriptions already paid for, then the flat-rate route, and only reach metered dollars at the last
rung. Every step down is cheaper in real money than the one above it.

| Rung | Cost type | Lane |
|---|---|---|
| 1 | subscription (already paid for) | `claude-sonnet-5` / `codex:gpt-5.6-luna` |
| 2 | flat-rate (Ollama Cloud pro) | `pi:ollama/<model>:cloud` |
| 3 | metered, cheapest available | `pi:openrouter/<same weights>` |

Rungs 2 and 3 are deliberately **the same weights on two routes**, so exhausting the flat-rate quota
(whose denominator is unpublished) costs money and never capability.

| Role | Chain | Notes |
|---|---|---|
| **Controller** | `claude-sonnet-5` → `codex:gpt-5.6-luna` | The driver's controller ladder speaks only Anthropic model IDs plus ONE codex fallback, so it cannot express rungs 2-3. Down from opus — this is the biggest line item, a long session re-reading a ~3.8k-line skill each fire. |
| **Designer** | `claude:claude-sonnet-5` (seed) | Seed only: the designer is a skill-side **rotation**, not a chain — the driver has no `MISSION_DESIGNER_FALLBACK`. Seeded off the Fable default. Most docs items need no design doc at all (see Guardrails). |
| **Planner** | `codex:gpt-5.6-luna` → `pi:ollama/glm-5.3-flash:cloud` → `pi:openrouter/z-ai/glm-5.3-flash` | Starts at rung 1b, **not** sonnet, and that is forced: `derive-planner-lane.sh` Step 0 accepts only `codex:*` or `pi:*`, so a bare `sonnet` pin emits the labeled fail-closed default `opus fail-closed:env-pin` and deliberately runs opus (a documented fail-closed-to-safety default, not a silent one — every exit path emits a distinct reason token precisely so the lane never reads as pinned in the driver log while actually running opus). Drops the fleet's kimi-k3 rung — free on flat-rate, but $3/$15 per M on its OpenRouter twin, i.e. 40x glm-5.3-flash and dearer than gpt-5.6-sol. As the last rung of a cost ladder that is backwards. |
| **Executor** | `codex:gpt-5.6-luna` → `pi:ollama/glm-5.3-flash:cloud` → `pi:openrouter/z-ai/glm-5.3-flash` | The high-volume role, so the ladder pays most here. The driver dedupes the shared probe, so planner+executor cost one probe. |
| **Evaluator** | `sonnet` → `pi:ollama/minimax-m3:cloud` → `pi:openrouter/minimax/minimax-m3` | **Deliberately vendor-disjoint from the executor at every rung.** Two reasons, the second structural: (1) on a mission whose generator is cheap on purpose, a cheap judge means nobody catches anything and the loop reports success it did not earn; (2) the executor chain is OpenAI → Z-AI → Z-AI, so an evaluator walking the *same* ladder would share a vendor at every rung — precisely the shared blind spot generator≠judge exists to prevent. |

**Verified before being written down** (2026-08-28, both rc=0): `codex exec --model gpt-5.6-luna`
returned "ok" on the subscription lane (7,930 tokens); `ollama run glm-5.3-flash:cloud` returned
"ok". Live OpenRouter prices read the same day — `z-ai/glm-5.3-flash` $0.075/$0.250 per M at
1,310,720 context. `deepseek/deepseek-v4-flash-0731` is marginally cheaper ($0.060/$0.120) but
glm-5.3-flash is the newer generation at a trivial delta on this mission's volume; recorded so the
choice reads as a decision rather than an oversight.

**Metered ceiling**: `$1`/iteration (fleet default is `$5`). Rungs 1-2 are subscription and
flat-rate, so a fire spending real dollars has already fallen to rung 3 — a low ceiling makes that a
loud stop instead of a silent bill.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

1. `[LANDED]` **docs-0 · ratify this charter — RATIFIED ATTENDED 2026-08-28 (Mark).** Closed by
   human decision after three quorum rounds, not by a passing quorum. The reasoning, recorded so a
   later iteration does not reopen it: the bar's seven clauses are **Mark's own selection**, made
   attended, so a quorum blocking them is second-guessing a decision by the human it exists to
   represent. And the objections that survived measurement were not about the **bar** at all —
   round 3's surviving `gpt5-6-sol` objection asks `docs-1` to inventory existing ingestion
   machinery before adding a router, which is a **queue-item design** question. This charter carries
   an entire backlog, so the quorum kept finding new things to say about work not yet designed, and
   each round raised *new* objections rather than converging.
   **The mechanism-level lesson (route to the shared skill, not to this mission):** charter
   ratification and design review are different jobs, and running a backlog-bearing governance doc
   through a design quorum blocks at the wrong gate. Every surviving objection is preserved and
   re-enters at its own item's design gate, where it is actionable.
   Iteration 0's real yield was elsewhere and is kept: 6 of 9 objections refuted by direct
   measurement, two corrections to the human-authored charter, and one fleet-wide skill fix
   (`0e341cc57`).
2. `[NEXT]` **docs-2 · clauses 1+3 · FIRST REAL SWEEP — the first item that touches the website.**
   Run `docs-sync` end to end and turn its output into a scored, clause-tagged queue. This converts
   "the website is probably drifting" into a measured backlog.
   **Promoted above `docs-1` on 2026-08-28 (Mark, attended).** The original ordering was an
   authoring error: it put infrastructure for clause 7 ahead of every clause that changes a
   published page, so after a full iteration the count of files changed under `docs/` was **zero**.
   This item also uses the **existing** `docs-sync` skill rather than new machinery, which answers
   `gpt5-6-sol`'s reuse objection instead of arguing with it (Critical Principle 1). Est. 1
   iteration.
3. `[NEXT]` **docs-1 · clause 7 · build the inbox-routing TRIGGER.** `send` and `forward` are
   verified working primitives (see clause 7's verification log) — no `internal/`/`cmd/` change is
   needed for those, and this item should not touch Go code. The missing piece is a **trigger**:
   something that periodically decides doc-related traffic exists (in `public-feedback`,
   `pkg:<vendor>/<name>`, or GitHub issues on `sunholo-data/ailang`) and calls
   `forward --to docs-mission` — on top of a dispatch path that has never worked end-to-end (36/36
   failures, ailang#900), so this must poll rather than assume a push.
   The blast radius was widened to include `tools/` on 2026-08-28 (Mark, attended — commit
   `29a467cac`) for exactly this purpose, so docs-1 may now proceed once picked; a script such as
   `tools/messaging/docs_inbox_router.sh` is in scope (see Guardrails). Deliverable: a message sent
   from outside, observed arriving via the verified read command in clause 7, acked. Est. 1
   iteration.
4. `[PARKED]` **docs-3 · clause 6 · benchmark surface audit.** Blocked on nothing, but sequence it
   after docs-2 so the drift picture is known first.
5. `[PARKED]` **docs-4 · clause 5 · taxonomy pass.** The `docs/docs/guides/` directory holds 40+
   guides accreted over time. Deferred until clauses 1-3 are green — consolidating pages that are
   also factually stale does both jobs badly.

---
**Document created**: 2026-08-28 (attended, with Mark). **Bar RATIFIED attended 2026-08-28** after
three quorum rounds — see `docs-0` for why by human decision rather than by a passing quorum. Sprints
route from `docs-2` onward; each queue item still passes its OWN design quorum at its own gate, which
is where the surviving iteration-0 objections re-enter.
