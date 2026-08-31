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

## Human Decision Ledger (authoritative current state)

This marked table — not STATUS prose — is the source of truth for which decisions are open.
Validate with `scripts/mission_decisions.sh --check`; list the asks with
`scripts/mission_decisions.sh --open`. Rows are append-only, IDs are never reused, and a human
answer changes the row to `RESOLVED` in the same iteration that consumes the directive.

<!-- decision-ledger:start -->
| ID | Status | Decision / recorded answer | Evidence |
|---|---|---|---|
| D-1 | RESOLVED | **ANSWERED (Mark, attended 2026-08-28): option (a) — fix `check_examples.sh` in this mission, and add `.claude/skills/docs-sync/scripts/*` to `MISSION_PLANNER_ALLOWLIST` so it plans on the cheap lane too.** Rationale: the defect is in docs-sync's OWN instrument, this mission is its only heavy consumer, and V1 has no stake in it — routing to V1 would add a handoff for no benefit. Note the ask was RE-FRAMED before being answered: the original wording ("widen the allowlist **so the mission can** fix its tooling") presumed the allowlist is a write gate. It is not — outside the list the work is fully editable, it just plans on `opus`. So the widening buys planner COST, not capability, and the fix was never blocked. **The finding**: `check_examples.sh` passes ABSOLUTE paths to `ailang`, tripping false `MOD010` failures — raw 12/29/176 against a corrected 166 pass / 9 genuine fail / 42 no-module. Unparks `docs-6`. | Measured first-party by the controller and independently re-derived from scratch by the sprint-evaluator across 217 `examples/runnable/*.ail`: same 9 failures, same mechanism, same corrected split. `docs/docs-sync-findings.md` DOCS-2-01 and DOCS-2-04. PR [#955](https://github.com/sunholo-data/ailang/pull/955) → `a8f904aac`. |
| D-2 | RESOLVED | **MOOT — nothing to widen; the premise was false (measured, attended 2026-08-28).** The ask assumed `docs/*` is "a SINGLE-LEVEL glob" reaching only files directly under `docs/`. It is not. These are `case` glob patterns, not shell pathname expansion, so `*` matches `/` — `docs/docs/**` and `docs/src/**` were reachable the entire time. The second half of the premise was also false: the allowlist is not a write gate, so even a genuine miss would mean *plans on opus*, never *cannot fix*. **DOCS-2-02 (the stale `v0.16.0` reference in `docs/docs/intro.mdx`) and every nested-page item deferred on this basis are UNBLOCKED and should be picked up.** Note the failure mode for future iterations: this row cited the charter's own Guardrails bullet as its evidence, and that bullet was an unverified human-authored claim — so a false statement in the charter became self-citing. A charter claim about what a mechanism DOES now has to carry the command that demonstrates it. | Discriminating control, run against the live mission env and the real `derive-planner-lane.sh`: `docs/docs/intro.mdx` → `codex:gpt-5.6-luna declared:codex-ok`, while `internal/eval_harness/models.yml` → `opus fail-closed:path-not-in-codex-allowlist` in the same run. Separately, `grep -rn MISSION_PLANNER_ALLOWLIST tools/ .claude/` returns only `derive-planner-lane.sh` and the driver's `export` — no write-scope enforcement exists anywhere, including `sprint-executor`. |
<!-- decision-ledger:end -->

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `docs-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Every
iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the scarcest
model budget; the append-only history lives in the log + archive.

## STATUS 2026-08-28 — ITERATION 1: docs-2 LANDED; the sync tool it depends on was found broken, and fixing it needs two allowlist decisions

First real sprint since ratification. Gate 0: kill switch armed; billing CLEAN; gh
`sunholo-voight-kampff`. No docs-mission-specific inbox traffic (11 unread in the canonical inbox,
all either V1's own reports, a stale coordinator task `task-a0628a5f` failing on a missing
`opencode` binary — unreachable from this session's local coordinator, not this mission's doing,
left unacked for its owner — or `mcp-public` package feedback for a different package); no
directives on bookkeeping issue `#953` since the watermark. Gate 1: `origin/dev` HEAD `d8fc0e1e5`
had one genuine red, `launchd drivers (bash 3.2)` failing on a wall-clock timing test
(`descendant discovery … 600s arm cap`) — confirmed FLAKY, not ours: same test failed on 2 other
unrelated commits in the preceding hour interspersed with passes, and it re-passed on this
iteration's own PR. Flagged to V1 (repo owner) via controlplane; not actioned further.

**PICK: `docs-2` (queue head), no design doc needed per Guardrails — brief `docs-2-brief.md`
already existed from a prior session (Planner-Lane declaration only).** Baselined the five
`docs-sync` diagnostics myself before routing (rule 3e): all rc=0, but `check_examples.sh`
reported **12 passed / 29 failed** out of 41 verdicts — alarming, so I checked the instrument
before trusting the result (rule 1's stale-binary-under-tests class). Root cause: the script
invokes `ailang run` with ABSOLUTE paths (`find "$RUNNABLE_DIR" …`), and any example declaring
`module examples/runnable/X` then fails `MOD010` because the module-path check compares against
the absolute path, not a repo-relative one. Built a fresh ldflags-stamped scratch binary
(`bin/ailang`, gitignored) and re-ran with RELATIVE paths: **166 pass / 9 genuine fail / 42
no-module**, across all 217 `examples/runnable/*.ail` files. The script's own raw numbers are an
INSTRUMENT ARTIFACT, not a language regression — handed to the planner as VERIFIED-BY-ME rather
than something to re-derive from zero.

**Routing**: controller `claude-sonnet-5` (session) · planner `codex:gpt-5.6-luna` (probe rc=0,
worktree `.planner-wt-iter1-docs-2` off `origin/dev`) · executor `codex:gpt-5.6-luna` (same lane,
per the mission's subscription-first ladder; own worktree `.wt-iter1-docs-2`, no git writes, per
the cross-provider recipe) · evaluator `sonnet` (Agent-tool pin, own isolated worktree
`.wt-iter1-docs-2-eval`) — generator≠judge holds (OpenAI codex vs Anthropic sonnet). Both codex
runs are subscription-lane (rung 1), so **metered=$0.00** of $1.

**EVALUATOR RE-DERIVED EVERY LOAD-BEARING CLAIM FROM SCRATCH, NOT FROM THE EXECUTOR'S REPORT.**
Built its own binary, wrote an independent 217-file sweep script, and got the identical 166/9/42
split with the identical 9 failing filenames; independently confirmed the `check_examples.sh`
absolute-path mechanism by isolating it (`ailang run --caps IO $(pwd)/…` fails MOD010, the same
command with a relative path succeeds); independently confirmed two further findings the executor
made beyond my own baseline — `batch_processing.ail`/`cli_args_demo.ail` need an `Env` capability
the generic checker never grants, and `audit_design_docs.sh` (159/1030) vs
`derive_roadmap_versions.sh` (126/682) report different design-doc population totals. **PASS
92/100, zero blocking.** Non-blocking: sprint JSON's `status` field left `"planned"` (hygiene),
no CHANGELOG entry (debatable for an internal page), and the executor's "reproduction" of my
pre-supplied baseline numbers restated rather than blind-derived them — true, and the evaluator's
OWN from-scratch derivation is what makes that harmless here.

**LANDED**: [PR #955](https://github.com/sunholo-data/ailang/pull/955) → squash `a8f904aac`. All
20 checks green including `test` (23m), `docs-build` (10m), and — re-confirmed on the MERGE COMMIT
itself, not just the PR head, per Gate 3b's squash-produces-a-new-commit rule — every check bar one
non-required SonarCloud quality-gate red that is INHERITED (same failure on the immediate parent
commit `8a993bb89`, before this PR ever merged; coverage/security-rating on new code, unrelated to
a docs-only diff). Flagged to V1; not our domain.

**THE ITERATION'S BEST FINDING WASN'T IN THE SPRINT'S SCOPE TO FIX.** Folding the findings into the
queue (docs-5 through docs-8) surfaced that this mission's OWN blast-radius allowlist blocks it
from fixing most of what it just found: `docs/*` is a single-level glob that excludes
`docs/docs/**` (where the actual stale-version bug and nearly all published content live), and
`.claude/skills/docs-sync/**` (where the checker's own bug lives) isn't covered at all. Filed as
`D-1` and `D-2` in a newly-created Human Decision Ledger (this mission had none yet — Gate 0's
`mission_decisions.sh --check` returned no block; created following V1's exact format, validated
2 rows). Queue renumbered: docs-2 → LANDED, new docs-5 ([NEXT], in-scope examples hygiene for the
9 genuine failures) / docs-6 / docs-7 (both PARKED on D-1/D-2) / docs-8 (PARKED, design-doc
triage, doesn't need an allowlist change but needs docs-6/7 settled first) inserted before the
renumbered docs-1/docs-3/docs-4.

**RETRO — NO SKILL EDIT.** One friction (this iteration's own `${#pending}` numeric-check on
`gh pr checks` output, handled inline with a `case … [!0-9]*)` guard per this file's own
prescription — worked as documented, not a gap) — below the ≥2-instance bar for a skill change.

Full record: `design_docs/docs-mission-log.md` §ITERATION 1.

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
  (non-`internal/`, non-`cmd/`).** This is a **POLICY** guardrail — a rule this mission follows,
  enforced by the loop reading it here. Widened from `tools/launchd/*` to `tools/*` on 2026-08-28
  (Mark, attended — commit `29a467cac`) to unblock docs-1's inbox-routing trigger.

  **⚠ CORRECTION 2026-08-28 (attended) — an earlier version of this bullet said the blast radius was
  "enforced mechanically at the planner gate by `MISSION_PLANNER_ALLOWLIST`". That was FALSE, it was
  authored by a human, and it cost the mission real work before anyone checked it.** Two facts,
  each measured with a discriminating control:

  1. **`MISSION_PLANNER_ALLOWLIST` is not a write gate and never was.** It appears in exactly two
     places in the whole repo — `derive-planner-lane.sh` and the driver's `export` — and nowhere
     else; there is no write-scope enforcement in `sprint-executor` at all. What it decides is
     **which model may PLAN the work**: every declared path inside the list → the cheap pinned
     planner; anything outside → `opus fail-closed:path-not-in-codex-allowlist`. A path outside the
     list is *expensive to plan*, never *impossible to edit*.
  2. **`docs/*` is NOT single-level.** These are `case` glob patterns, not shell pathname
     expansion, so `*` matches `/`. Measured: `docs/docs/intro.mdx` routes to
     `codex:gpt-5.6-luna declared:codex-ok`, while the `internal/` control is denied in the same
     run. Nested pages under `docs/docs/**` and `docs/src/**` were always reachable.

  **How this propagated, because the mechanism matters more than the fact.** The false sentence was
  written into this charter by a human; iteration 1 read it, believed it, deferred DOCS-2-01 and
  DOCS-2-02 as mechanically impossible, and then opened decision `D-2` **citing this very bullet as
  its evidence**. A charter is read as ground truth by every iteration, so an unverified mechanical
  claim in it does not merely mislead once — it becomes self-citing, and the loop cannot tell it
  from a measurement. **Rule: a claim in this charter about what a mechanism DOES must carry the
  command that demonstrates it, or must not be stated.** Where a restriction is a judgement rather
  than a mechanism, say "policy" — as this bullet now does.
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
2. `[LANDED]` **docs-2 · clauses 1+3 · FIRST REAL SWEEP.** Ran `docs-sync` end to end
   ([PR #955](https://github.com/sunholo-data/ailang/pull/955) → `a8f904aac`) and scored 13
   findings into `docs/docs-sync-findings.md`. Headline finding: `check_examples.sh` passes
   ABSOLUTE paths to `ailang run`, tripping false `MOD010` module-path-mismatch failures for
   nearly every example — its own raw 12/29/176 counts are unreliable; a corrected relative-path
   sweep gives 166 pass / 9 genuine fail / 42 no-module across 217 files. Independently re-derived
   by the sprint-evaluator (sonnet) from scratch — PASS 92/100, zero blocking. Follow-ups spawned
   as docs-5 through docs-8 below.
3. `[RULED OUT]` **docs-9 · clause 1 · "intro.mdx is stale at v0.16.0" — DISSOLVED 2026-08-31
   (controller live-repro): the check's premise was false, measured.** `check_versions.sh` Check 3
   greps the first `vX.Y.Z` string anywhere in `intro.mdx` and compares it to the latest prompt
   file, with no awareness that the "Recent Additions" section intentionally lists each feature's
   historical **ship**-version (`IFC Labels (v0.16.0)`, `Tiered Eval Suite (v0.14.0)`, `List-only
   ++ (v0.13.0)`, `Three-tier OTEL Tracing (v0.12.0)`, `New Stdlib Modules (v0.12.0)` — none match
   `STABLE_RELEASE`/`ACTIVE_PROMPT` by design, and none ever will). Diffing `prompts/v0.16.0.md`
   against `prompts/v0.16.6.md` found zero IFC/`declassify`/`T<label>` content changes across all
   six intervening revisions — the IFC Labels annotation is factually correct as `v0.16.0` and
   bumping it would misrepresent when the feature shipped. **Nothing in `intro.mdx` needed to
   change.** The real defect was the instrument: Check 3 is deleted in this sprint (see the
   script's own header comment for why), since a bare regex over prose cannot distinguish
   "current-version claim" from "historical ship-version annotation" without a marker convention
   the file doesn't have. Kept as `[RULED OUT]` rather than deleted, matching `docs-7`'s
   convention, because the interesting part is the mechanism: a queue item inherited a tool's raw
   `[STALE]` line without checking whether the tool's premise was sound.
4. `[NEXT]` **docs-5 · clause 2 · examples hygiene — fix the 9 genuinely-failing runnable
   examples.** `block_demo.ail`, `test_module_minimal.ail`, `simple_func_match.ail`,
   `match_arm_block.ail`, `match_in_block.ail`, `nested_match_minimal.ail` have no `main`
   entrypoint (helper/library files being swept as if runnable); `batch_processing.ail` and
   `cli_args_demo.ail` need an `Env` capability the generic checker never grants. In scope:
   `examples/*` is a multi-level-safe allowlist entry, so adding explicit `main` wrappers or
   capability-usage comments is a normal sprint. Do NOT touch
   `.claude/skills/docs-sync/scripts/check_examples.sh` itself (see docs-6). Source:
   `docs/docs-sync-findings.md` DOCS-2-05 through DOCS-2-13.
5. `[NEXT]` **docs-6 · clause 1 · fix `check_examples.sh`'s
   absolute-path bug.** The instrument this mission's whole clause-1/2 sweep depends on has been
   silently over-reporting broken examples (see docs-2's findings, DOCS-2-01). Fixing it means
   editing `.claude/skills/docs-sync/scripts/check_examples.sh`, which sits outside this mission's
   blast-radius allowlist (`.claude/skills/*` only covers `mission-control/SKILL.md` and
   `design-doc-creator/*`). Also folds in DOCS-2-04 (the audit_design_docs.sh vs
   derive_roadmap_versions.sh design-doc population-count mismatch — same script family, same
   scope question).
6. `[RULED OUT]` **docs-7 · "the mission cannot edit its own published content" — DISSOLVED
   2026-08-28 (attended): the premise was false, measured.** This item asserted that
   `MISSION_PLANNER_ALLOWLIST`'s `docs/*` is a single-level glob excluding `docs/docs/**` and
   `docs/src/**`. It is not — these are `case` glob patterns, not shell pathname expansion, so `*`
   matches `/`. Control run against the live env and the real script: `docs/docs/intro.mdx` →
   `codex:gpt-5.6-luna declared:codex-ok`, while `internal/eval_harness/models.yml` →
   `opus fail-closed:path-not-in-codex-allowlist` in the same run. And the allowlist is not a write
   gate in the first place (see D-2 and the Guardrails correction), so even a real miss would mean
   *plans on opus*, never *cannot fix*. **Nothing was ever blocked.** The work this item was
   guarding is now `docs-9`. Kept as `[RULED OUT]` rather than deleted, because the interesting
   part is the mechanism: this item was created from a false sentence in this charter, which then
   got cited back as its own evidence — see the Guardrails correction for the rule that closes it.
7. `[PARKED]` **docs-8 · clause 1 · 126 overdue planned design docs (aggregate).** Everything under
   `design_docs/planned/` targeting v0.29.0 through v0.31.0 while the repo ships v0.34.0
   (DOCS-2-03). `design_docs/` is also outside the current sprint allowlist — moving a doc to
   `implemented/` is normally the mission CONTROLLER's own Gate-4 bookkeeping (as v1-mission does),
   not a sprint-executor task, so this can proceed without an allowlist change, just not via the
   automated inner loop. Sequence after docs-6/docs-7 land or are ruled out, since triaging 126
   docs by hand is exactly the kind of large sweep worth doing once against settled tooling.
8. `[NEXT]` **docs-1 · clause 7 · build the inbox-routing TRIGGER.** `send` and `forward` are
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
9. `[PARKED]` **docs-3 · clause 6 · benchmark surface audit.** Blocked on nothing, but sequence it
   after docs-2 so the drift picture is known first.
10. `[PARKED]` **docs-4 · clause 5 · taxonomy pass.** The `docs/docs/guides/` directory holds 40+
   guides accreted over time. Deferred until clauses 1-3 are green — consolidating pages that are
   also factually stale does both jobs badly. Also blocked on docs-7's allowlist question, since
   `docs/docs/guides/` is nested.

---
**Document created**: 2026-08-28 (attended, with Mark). **Bar RATIFIED attended 2026-08-28** after
three quorum rounds — see `docs-0` for why by human decision rather than by a passing quorum. Sprints
route from `docs-2` onward; each queue item still passes its OWN design quorum at its own gate, which
is where the surviving iteration-0 objections re-enter.
