## STATUS 2026-09-02 — ITERATION 5: docs-8's stale "126 overdue" corrected to a verified 54, 18 archived after independent re-verification caught 3 wrong claims; docs-3 credited from a second orphaned fire, blocked on a V1-owned inherited CI red

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree at
`origin/dev` tip (`50dd1a0aa`), clean. 0 directives on bookkeeping issue `#979` since the
watermark (6 comments, none allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new
ask. No docs-mission inbox traffic (20 unread canonical-inbox messages: motoko/V1 cross-mission
notifications, pkg feedback for unrelated packages, eval-suite runs — none addressed to
`mission-docs`).

Gate 1: `origin/dev` HEAD SHA-addressed check-runs showed 6 NOT-GREEN: `Build windows/macos/
ubuntu-latest` (cancelled/failure), `launchd drivers (bash 3.2)` (failure), `test` (failure) —
confirmed **inherited** from the parent commit too (identical failure set on both), and V1's own
mission log carries 10-22 prior mentions of these exact two check names, so this is known,
tracked, and out of this mission's domain (V1 owns `sunholo-data/ailang` per Gate 1's
repo-ownership scoping) — not actioned, only noted for the report.

**Gate 2 — died-mid-flight check found a second orphaned fire.** Open PR
[#1031](https://github.com/sunholo-data/ailang/pull/1031) (`docs/iter5-docs3-provenance-wiring`,
`MERGEABLE`) plus worktrees `.wt-docs-iter5-docs3`/`-eval`, zero "ITERATION 5" trace anywhere
(0/0/0 in charter/log/archive at pick time, known-present control `ITERATION 4` = 2/1 firing). A
prior fire had run the full inner loop for `docs-3` — codex executor, sonnet evaluator PASS
85/100 zero blocking, diff scope verified exactly 4 files — then died before Gate 4/5. Re-verified
first-party: diff scope re-confirmed via `gh pr diff --name-only`, `mergeStateStatus` is
`BLOCKED` on the same inherited red as Gate 1 found (not a stale-base problem — `git diff --stat
<PR base>..origin/dev -- <the 4 touched files>` is empty, so a rebase would not produce a
different check outcome; the red is on origin/dev's own current tip). **Credited, not re-run**;
left open as a resume point (queue row `docs-3`, now `[IN-SPRINT]`) rather than force-merged or
re-executed — not this mission's fix to make.

**PICK: `docs-8`** (natural next per iteration 4's own note — the only PARKED item explicitly
unblocked once docs-6/docs-7 resolved). **Reality-check first** (Gate 2 rule): re-ran
`derive_roadmap_versions.sh` at HEAD — the charter's own "126 overdue" figure was stale (count
drift since docs-2/docs-6 touched the same script family); the real, current overdue set (target
version < v0.34.0) is **54** docs, not 126.

**Execution — controller-run triage, not a sprint** (per this item's own charter text: moving a
doc to `implemented/` is CONTROLLER Gate-4 bookkeeping, `design_docs/` is outside
`MISSION_PLANNER_ALLOWLIST`, so this never was going to route through codex). Delegated the
54-doc cross-reference to 6 parallel `general-purpose`/sonnet Agent-tool sub-agents (9 docs each:
grep for implementation evidence, commit/changelog citations, known-positive controls on every
negative finding per rule 3a). Result: 20 IMPLEMENTED, 2 RULED-OUT, 1 NEEDS-DEEPER-INVESTIGATION,
31 STILL-PLANNED.

**Independent re-verification BEFORE any file moved** (generator≠judge — the classifying agents
were the "generator", a separate adversarial sonnet sub-agent was the judge, per this run's
explicit operator mandate that no work lands on the controller's own verdict). Spawned one
independent auditor to re-run every cited command itself against the 22 highest-stakes claims (20
IMPLEMENTED + 2 RULED-OUT — the ones that trigger a file move or a rule-out stamp).
**Caught 3 of 22 (14%) wrong, 2 of them outright reversals**: `m-eval-slim-prompt-self-discovery`
was classified IMPLEMENTED on general MCP-plumbing evidence, but the doc's OWN specific artifacts
(a tagged slim prompt, a committed A/B report) never existed — the experiment was built, A/B
tested (mixed-to-negative, 82%→65%), and explicitly deleted (`2de1ef963`); the live
`local-ollama-eval` skill documents the approach as "tried and deliberately abandoned." Moving it
to `implemented/` would have misfiled an abandoned experiment as shipped — REFUTED, kept under
`planned/`, header updated with the evidence instead. `m-eval-fmt-weakmodel-ab-M6-motoko-ext` was
classified RULED-OUT on a `models.yml` "RETIRED" comment, but read in full that comment retires
only the nightly *scheduling* because its model arm was decommissioned from the rig — the
extension itself was never measured, so the doc's own "BUILT + INTEGRATED, firing not yet
observed" status is still accurate; REFUTED, left untouched. `m-eval-stream-health-retry`
downgraded IMPLEMENTED→WEAK: the cited evidence file was for a different, adjacently-named
M-number; the real TTFT/idle-timeout detection genuinely landed in the opencode/pi executors, but
the doc's actual point — retry-on-stream-death and correct `stream_death` labeling instead of
generic `api_error` — did not; left untouched (STILL-PLANNED).

**Outcome: LANDED.** 18 confirmed-implemented docs archived: `git mv` from `planned/vX_Y/` to
`implemented/vX_Y/` (created `implemented/v0_33_2/`, didn't exist), 27 files total including
sprint-plan companions (Mark's "plans travel with their doc" convention). 1 ruled out via header
update with evidence (`m-eval-slim-prompt-self-discovery.md`). The 31 genuinely-still-planned docs
are now this mission's accurate, individually-pickable backlog — no new aggregate queue item
needed; a future iteration picks any one of them directly from `design_docs/planned/`.

**Cost**: metered **$0.00** of $1 ceiling — all 7 sub-agents were Agent-tool sonnet spawns
(Anthropic quota, not metered). Quota buckets: sonnet (6 classifier sub-agents + 1 independent
verifier + controller session).

Full record: `design_docs/docs-mission-log.md` §ITERATION 5.

# Docs Mission — STATUS archive

Append-only. Newest entry at the TOP (Gate 4 moves the 4th-newest charter stamp here, per
`docs-mission.md`'s rotation rule). The charter itself always carries the newest 3.

## STATUS 2026-09-02 — ITERATION 4 (crediting ITERATION 3): docs-5/6/10 landed by an orphaned fire, credited retroactively; docs-1 LANDED after a real evaluator FAIL/fix/PASS cycle

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Main checkout `dev` diverges
from `origin/dev` by design (9 ahead / 20 behind — attended commits stranded, loop lands via
worktree→PR; per this file's own note); the pin worktree this loop actually runs from was clean at
`origin/dev`'s tip throughout. Running skill differs from `origin/dev` (missing heartbeat stamps
and the attended-ledger-edits section added since) — read the delta, confirmed the pin worktree's
own scripts (`tools/launchd/mission-heartbeat.sh`, `scripts/mission_decisions.sh`,
`scripts/mission_answer.sh`) already exist, so followed the newer instructions rather than the
stale loaded copy. 0 directives on bookkeeping issue `#979` since the watermark (4 comments, none
allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new ask.

Gate 1: `CI` and `Deploy Documentation to GitHub Pages` both `success` on `origin/dev` HEAD;
SHA-addressed check-runs showed 16 checks, one non-green (`SonarCloud Code Analysis`), confirmed
inherited from the parent commit too — V1's domain (repo owner), not actioned.

**PICK: `docs-1`, after crediting a second died-mid-flight fire.** Gate 2 found `docs-5`/`docs-6`/
`docs-10` all merged on `origin/dev` (PRs #997/#1004/#1010) while the charter still tagged them
`[NEXT]` — see ITERATION 3's retroactive entry in the log for full detail and re-verification
(fresh `make verify-examples`: 211/0/6, `check_examples.sh`: 173/2/42, both matching the landing
PRs' own claimed counts). Also found PR #1016 (`MERGEABLE`) recovering a complete `docs-1` brief +
sprint plan from a second orphaned worktree; merged it after re-running its one flaky failing check
(`launchd drivers (bash 3.2)`, unrelated to a 2-file markdown PR, green on re-run — rule 3d).

**Execution**: routed `docs-1` to `codex:gpt-5.6-luna` using the recovered plan verbatim.
Round-1 delivery (`tools/messaging/docs_inbox_router.sh`) FAILED independent evaluation (sonnet,
own worktree) 58/100 — a genuinely empty poll result crashed the router instead of reporting
`checked=0 forwarded=0`, live-reproduced by the evaluator. Fix routed back to the same executor;
controller independently reproduced both the original crash and the fix with hand-built fixtures
before re-committing. Round-2 evaluation: PASS 90/100, zero blocking.

**Outcome: LANDED.** [PR #1018](https://github.com/sunholo-data/ailang/pull/1018) squash-merged →
`e65e96b15`. Polled the merge commit to full CI completion: 15/16 green, the same inherited
SonarCloud red as Gate 1, not actioned.

**Metered cost this iteration: $0.00** of $1 ceiling — codex and sonnet are both subscription-lane
per this mission's routing table. Quota buckets: codex (executor, 2 rounds), sonnet (evaluator, 2
rounds + controller session).

Queue is now empty of `[NEXT]` items; `docs-8` (126 overdue planned docs) is the natural next pick
once picked up (already unblocked per its own sequencing note) but was not started this iteration
(Standing rule 1, one backlog item per iteration).

Full record: `design_docs/docs-mission-log.md` §ITERATION 3, §ITERATION 4.

## STATUS 2026-08-31 — ITERATION 2: recovered a died-mid-flight fire (docs-9 RULED OUT, PR #973 landed); Gate-0 weekly sweep found docs-10

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. `dev` == `origin/dev` at
pick time (`c16911e0b`), no divergence. 11 unread canonical-inbox messages, none docs-mission
directives (V1's own controlplane traffic, `docparse`/`aitana-platform` package feedback for a
different product, eval-suite run notifications) — same finding as iteration 1. Zero directives on
bookkeeping issue `#953` since the watermark (`scripts/mission_directives.sh`, 0 of 16 comments).
Decision ledger valid, 2 rows, both `RESOLVED` (D-1, D-2) — no new ask.

Gate 1: `origin/dev` HEAD check-runs showed **two** non-green: `SonarCloud Code Analysis` and
`launchd drivers (bash 3.2)`, both confirmed **inherited** from the immediate parent commit
(`c16911e0b`, identical conclusions on both) — not caused by anything this mission is about to do,
already flagged to V1 (repo owner) by iteration 1, not re-flagged. Skill copy confirmed matching
`origin/dev` (`cmp` clean).

**PICK: none fresh — Gate 2's died-mid-flight check found a complete, unlanded prior fire.** Open
PR `#973` (`sprint/iter2-docs-9`) plus three orphaned worktrees, zero "ITERATION 2" trace anywhere
in charter/log/archive (0/0/0, known-present control `ITERATION 1` = 1). The prior fire had run
the full inner loop on `docs-9` to completion — `[RULED OUT]`, the intro.mdx staleness claim was a
permanent false-positive of `check_versions.sh` Check 3 — and died before Gate 4/5. Re-verified
first-party rather than trusted: intro.mdx's version annotations are ship-versions (5 bullets, 5
different versions, confirmed by direct read); `prompts/v0.16.0.md` vs `v0.16.6.md` diff only the
title line; all three worktrees clean (no uncommitted state — the fire finished, it just never
landed); PR `#973` `MERGEABLE`/`CLEAN`, 21 checks, none non-green.

**Outcome: LANDED.** Squash-merged [PR #973](https://github.com/sunholo-data/ailang/pull/973) →
`ad7542ba5`. Local `dev` fast-forwarded. CI polled to completion on the merge commit itself:
`Deploy Documentation to GitHub Pages` green; `CI` conclusion `failure` — but check-runs isolate it
to the SAME two reds (`SonarCloud Code Analysis`, `launchd drivers (bash 3.2)`), both confirmed
identical-conclusion on the parent commit — inherited, not introduced by this merge. Orphaned
worktrees removed.

**Gate 0 weekly external-issue sweep** (first iteration after the Monday 2026-08-31 07:00 CEST
rotation boundary — `#953` created before it): 92 open issues enumerated (`--limit 100`, asserted
against `jq length` = 92 — first attempt used `--limit 50` and silently truncated, caught before
recording). Per-issue `#N\b` grep across charter/log/archive/dashboard, known-positive control
(`#953` → 6) and known-absent control (`#88214` → 0) both firing. First pass read 92/92 orphaned —
wrong, self-caught: a zsh 1-indexed-array bug (`${FILES[0]}` empty) made every grep run with no
file argument. Corrected: **89/92 orphaned**, 87 plainly out of domain, **2 in-domain**:
[#670](https://github.com/sunholo-data/ailang/issues/670)/[#654](https://github.com/sunholo-data/ailang/issues/654),
both showing `make verify-examples` (this mission's own verify-profile gate) never actually
verifies output and has no anti-vacuity floor. Re-confirmed live at HEAD (both defects still
present). Filed as new queue item **`docs-10`**, positioned after `docs-6`.

**Metered cost this iteration: $0.00** of $1 ceiling — no new model-role spawns; this iteration was
controller-session verification + bookkeeping only. Quota buckets: sonnet (controller).

Bookkeeping issue rotated: `#953` → `#979` (Monday 07:00 boundary rule; `#953` had 16 comments,
under the 80 threshold, but was created before this week's boundary).

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
