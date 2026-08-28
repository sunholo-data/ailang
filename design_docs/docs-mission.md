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
**Human-facing reporting**: GitHub issue #**TBD** — created at arming time, seeded into
`~/.ailang/state/mission-docs-gh-issue`. Every iteration posts its report there; driver crashes too.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/docs-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `docs` (driver: `MISSION_NAME`; every
  `~/.ailang/state/mission-docs-*` path is namespaced away from the V1 loop)
- **Checkout**: `/Users/voightkampff/dev/sunholo-data/ailang-docs` — **its own clone of this repo**.
  The V1 loop mutates this same repo every 90 minutes from a different tree; two loops in one
  working tree is precisely the concurrent-agent hazard. Separate clone = the motoko precedent.
- **Bookkeeping issue**: `#TBD`, rotates weekly; live number in `~/.ailang/state/mission-docs-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `Deploy Documentation to GitHub Pages` (the docs gate,
  path-filtered on `docs/**`, `prompts/**`, `llms.txt`, `CHANGELOG.md`), and `CI` (which runs on
  every push — this repo has no push paths filter, so a docs-only commit still runs full CI and
  Gate 3b must wait for it rather than reading its absence as "not applicable").
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

## STATUS 2026-08-28 — ITERATION 0 PENDING: charter written, **not yet ratified**, loop armed-but-silent

Charter drafted attended with Mark 2026-08-28. Bar clauses 1-7 are Mark's own selection and must
still be **ratified through the design quorum at iteration 0** before any sprint routes. The kill
switch `~/.ailang/state/mission-docs.disabled` is set; every fire exits at Gate 0 until it is
removed deliberately.

One infrastructure defect was found and fixed *before* the loop existed, because it would have
silently inverted the mission's entire purpose: `derive-planner-lane.sh` fails closed to **opus**
for any path outside an infra-only allowlist, so every docs design doc would have routed the
planner to the most expensive model in the fleet while the driver log reported the cheap pin.
Measured with a discriminating control; fixed as per-mission data (`MISSION_PLANNER_ALLOWLIST`,
default unchanged); five regression assertions added to `tools/launchd/test_mission_routing.sh`
(25/25 green).

## CURRENT GOAL

1. **Iteration 0 (definition)**: ratify the bar below with Mark through the design quorum, then
   score the backlog against it into `required` / `nice` / `post`. Output: an ordered queue in this
   doc. **Also at iteration 0**: stand up the `docs-mission` inbox routing that clause 7 depends on
   (see clause 7 — it has no delivery mechanism yet).
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
  **This clause has no delivery mechanism yet — building it is iteration 0 work, not an assumption.**
  What is verified: `ailang messages send <inbox>` takes a free-form inbox name, so `docs-mission`
  needs no registration, and `ailang messages forward <id> --to docs-mission` exists. What is NOT
  built: anything that *routes* doc-related traffic there. Public feedback lands in
  `public-feedback` and `pkg:<vendor>/<name>`, and **feedback dispatch has never worked** (36/36
  failures since 2026-04-27, ailang#900) — so do not assume arrival. The canonical store is prod
  Firestore; a bare `ailang messages list` reads local SQLite and will show an empty, reassuring,
  wrong inbox.

## Guardrails (mission-specific; the skill's Standing Rules always apply on top)

- **Blast radius is `docs/`, `examples/`, `README.md`, `CHANGELOG.md`.** Enforced mechanically at
  the planner gate by `MISSION_PLANNER_ALLOWLIST`. If an item genuinely needs a change under
  `internal/` or `cmd/`, that is a **V1 backlog item, not a docs item** — file it across and move
  on. Do not widen the allowlist to make an item fit; the allowlist is the definition of this
  mission's scope, not an obstacle to it.
- **No designer on Fable.** The shared skill's designer rotation is
  `claude:claude-fable-5 → pi:ollama/kimi-k3:cloud`; this mission seeds it at the kimi lane and
  should stay there. Fable is reserved for high-cognition spec synthesis, and docs items are drift
  repair. Most items here need **no design doc at all** — prefer a Gate-2 reality-check straight
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

This mission **promotes the flat-rate Ollama Cloud lanes from first-fallback to primary** — a
deliberate, scoped departure from the fleet policy ratified 2026-08-26. That policy's own reasoning
licenses the exception: the objection was the absence of role-specific evidence plus the cost of
learning it somewhere consequential. A docs loop is the cheapest venue to generate exactly that
evidence, and every routing row recorded here feeds the promotion rule for the fleet.

| Role | Lane | Why |
|---|---|---|
| **Controller** | `claude-sonnet-5` (fallback `codex:gpt-5.6-sol`) | Biggest line item — a long session re-reading a ~3.8k-line skill each fire. Opus-first is right for a loop shipping compiler changes, not for one fixing doc drift. |
| **Designer** | `pi:ollama/kimi-k3:cloud` | Flat-rate. Rotation seeded off Fable deliberately (see Guardrails). |
| **Planner** | `pi:ollama/kimi-k3:cloud` → `pi:openrouter/moonshotai/kimi-k3` | Strongest open-weight model measured externally, in the role Mark already scoped it to. |
| **Executor** | `pi:ollama/deepseek-v4-flash:0731-cloud` → `pi:openrouter/deepseek/deepseek-v4-flash-0731` | The high-VOLUME role, so flat-rate pays here. Same weights the fleet already uses as first fallback — a route change, not a capability change. |
| **Evaluator** | `sonnet` → `pi:ollama/minimax-m3:cloud` → `pi:openrouter/minimax/minimax-m3` | **Deliberately NOT down-tiered.** On a mission whose generator is cheap on purpose, the judge is the wrong economy: a cheap executor checked by a cheap judge means nobody catches anything and the loop reports success it did not earn. Also keeps generator≠judge structurally sound (Anthropic judging DeepSeek) and off the executor's quota pool. |

**Metered ceiling**: `$1`/iteration (fleet default is `$5`). Every primary lane is flat-rate or
subscription, so a fire spending real dollars has already fallen off its intended route — a low
ceiling makes that a loud stop instead of a silent bill.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

1. `[NEXT]` **docs-0 · iteration 0 · ratify this charter.** Quorum-review clauses 1-7 with Mark;
   score the backlog into required/nice/post. Est. 1 iteration.
2. `[NEXT]` **docs-1 · clause 7 · build the inbox routing.** Clause 7 is the only clause with no
   delivery mechanism. Decide and implement how doc-related traffic reaches the `docs-mission`
   inbox, against the known-broken dispatch path (ailang#900). Deliverable: a message sent from
   outside, observed arriving, acked. Est. 1 iteration.
3. `[NEXT]` **docs-2 · clauses 1+3 · first real sweep.** Run `docs-sync` end to end and turn its
   output into a scored, clause-tagged queue. This is the item that converts "the website is
   probably drifting" into a measured backlog. Est. 1 iteration.
4. `[PARKED]` **docs-3 · clause 6 · benchmark surface audit.** Blocked on nothing, but sequence it
   after docs-2 so the drift picture is known first.
5. `[PARKED]` **docs-4 · clause 5 · taxonomy pass.** The `docs/docs/guides/` directory holds 40+
   guides accreted over time. Deferred until clauses 1-3 are green — consolidating pages that are
   also factually stale does both jobs badly.

---
**Document created**: 2026-08-28 (attended, with Mark). Iteration 0 ratifies it via the quorum
before any sprint routes.
