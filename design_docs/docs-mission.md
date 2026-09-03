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
| D-3 | OPEN | **ASK (iteration 6, 2026-09-03): one-time OK to use the shared mission-control skill's narrow-refinement carve-out for `docs-4`'s design brief, docs-mission's first use of it.** The brief (`design_docs/docs-4-brief.md`) blocked quorum twice; both rounds' objections were narrow, concrete (each reviewer supplied a verbatim `proposed_fix`), and disputed no design direction — round 1: an unprobed 9th orphan URL, an unverified section-heading pair; round 2: imprecise "URL-stable" wording against two intentional clause-5-authorised deletions, an unverified 3rd heading boundary, a cleanup step left as prose instead of an encoded acceptance criterion. The controller applied all fixes verbatim (commits `fbc289f6a`, `56acda30d`) rather than spending a 3rd $0.12 quorum round. Options: **(a)** OK it — the doc is design-ready, `sprint-planner` runs next iteration. **(b)** Reject it — the item goes back to a normal 3rd quorum round (cost: ~$0.12, one more iteration). Loop recommendation: (a) — both rounds' objections read as the reviewers doing their job on a genuinely fixable doc, not as a doc that shouldn't ship. Default if unanswered: stays parked at design-ready: no sprint runs, no cost accrues, `docs-4` sits at the top of next iteration's queue exactly as now. | `design_docs/docs-4-brief.md` §"Quorum log"; quorum artifacts `docs-4-brief-2026-09-03T10-54-51Z.json`, `docs-4-brief-2026-09-03T10-57-26Z.json`; carve-out rule text in `.claude/skills/mission-control/SKILL.md` Gate 2. |
<!-- decision-ledger:end -->

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `docs-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Every
iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the scarcest
model budget; the append-only history lives in the log + archive.

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

## STATUS 2026-09-03 — ITERATION 6: docs-4 taxonomy pass designed and scoped to one sprint (62 files, near-zero literal duplication measured); quorum blocked twice, closed via this mission's first narrow-refinement carve-out — sprint held pending Mark's one-time OK

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree at `origin/dev`
tip (`55891002f`), clean. 0 directives on bookkeeping issue `#979` since the watermark (7
comments, none allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new ask at Gate
0 (one added at Gate 4, see below). 16 unread canonical-inbox messages, none addressed to
`mission-docs` (design-doc-creator/pkg-sunholo/mission-v1/mission-world cross traffic).

Gate 1: `origin/dev` HEAD SHA-addressed check-runs showed 6 NOT-GREEN: `Build windows/macos/
ubuntu-latest` (cancelled/failure), `launchd drivers (bash 3.2)` (failure), `test` (failure), same
signature Gate 1 has flagged in iterations 1/2/4/5 — confirmed still V1's domain (owning mission
per Gate 1's repo-ownership scoping), not actioned here beyond a cross-mission heads-up
(`mission-v1` inbox, `inbox_1788431686434_383803af`) naming the two failing jobs and the two
most-recent commits, since neither looked docs-shaped.

**PICK: `docs-4`** (taxonomy pass, item 11). Both of its stated blockers are now cleared: clauses
1-3 are green (docs-2 covers 1+3; docs-9/docs-6/docs-10/docs-8 cover clause 1 further; docs-5
covers clause 2), and docs-7's allowlist question dissolved 2026-08-28. `docs-3` (item 10) stays
`[IN-SPRINT]`, unpicked — still blocked on the same inherited red named above (its own PR #1031
mergeability re-checked: still `MERGEABLE`/`BLOCKED` on identical failing jobs).

**Gate 3 — designer.** No design doc existed for docs-4. Spawned this iteration's rotation
designer (`claude:claude-fable-5`, Agent tool, `model="fable"`) with an explicit judgment call:
given 62 files, decide with evidence whether this is one sprint or needs decomposition into
sprint-sized sub-docs (the charter's standing multi-week-item rule). Result:
`design_docs/docs-4-brief.md` (committed `1c74a4971` on branch `docs-4-brief` off `origin/dev`).
**One sprint, not a decomposition** — a pairwise line-overlap instrument across all 1,830 guide
pairs found only 2 pairs sharing ≥4 identical lines; the real redundancy is 5 recurring
command-block sections across 4-7 files, not page-level duplication; six 2026-08-17 "audit pass"
commits already did the page-level merging. Every one of the 62 files (+11 in `evaluation/`) gets
an explicit disposition (Appendix A); the target sidebar tree is fully specified (Appendix B);
29-row Verification Log with commands (two self-caught instrument bugs: BSD `sed` `\?`, zsh
non-word-splitting).

**Quorum-at-pick (mandatory, no prior artifact for this doc).** Round 1
(`docs-4-brief-2026-09-03T10-54-51Z.json`): BLOCKED — `gpt5-6-sol` and `gemini-3-1-pro` both
reject (`oc-glm-5-2` absent, degraded N-1, not silently passed), both objections narrow
verification-completeness gaps with concrete `proposed_fix` (V6 probed only 8 of 9 orphan URLs;
no row proved the B4/B5 section-cut headings actually exist/are adjacent). Controller measured
both directly — single commands, no design judgment, so not re-routed through the designer per
Gate 2's rule — both premises held (9th orphan also `200`; both heading pairs exact). Round 2
(`docs-4-brief-2026-09-03T10-57-26Z.json`, the mandatory one re-quorum): BLOCKED again, all three
reviewers present and reject — `gpt5-6-sol` (the "URL-stable" scope line overclaims against the
two intentional clause-5-authorised deletions), `gemini-3-1-pro` (B3's cut boundary needed the
same line-exact proof V29 just gave B4/B5), `oc-glm-5-2` (the `sync-registry.sh`/`make docs-build`
side-effect cleanup was V23b prose, never an encoded acceptance step — an executor following the
checklist literally could commit 4 mutated tracked files). All three narrow, concrete, no design-
direction dispute → applied the **narrow-refinement carve-out** (bounded 2nd revision, reviewers'
verbatim fixes, no 3rd quorum round): reworded URL-stable's scope, added V30 (B3 boundary
confirmed at lines 180/253, genuine H2s), encoded the `git checkout --` cleanup as acceptance
criterion 6/7 rather than prose. Committed `fbc289f6a` (round-1 fixes), `56acda30d` (round-2 +
carve-out record). **This is docs-mission's first use of the carve-out** — per the skill's
ratification rule the doc is design-ready, but the sprint (planner/executor) is held pending
Mark's one-time OK rather than routed straight through, so `sprint-planner` and `sprint-executor`
were not spawned this iteration. `sprint-evaluator` accordingly has nothing to independently
judge yet — no code landed, so no generator≠judge step was owed this iteration.

**Routing evidence**: designer `claude:claude-fable-5` (Fable diet: 1 doc, 0 revision-designer-
runs — both quorum-response edits were controller-measured, not re-spawned); planner/executor/
evaluator not spawned (blocked on ratification, not a probe failure — no fallback chain
traversed). Quorum reviewer cost: round 1 $0.1297, round 2 $0.1219 = **$0.2516 metered**, well
under the $1 ceiling and the $10/doc quorum cap.

**Cost**: metered $0.2516 of $1 ceiling (2 quorum rounds, OpenRouter-billed reviewers). Quota
buckets: fable (designer, 1 bounded run), sonnet (controller session).

Full record: `design_docs/docs-mission-log.md` §ITERATION 6.


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
4. `[LANDED]` **docs-5 · clause 2 · examples hygiene — fix the 9 genuinely-failing runnable
   examples.** `block_demo.ail`, `test_module_minimal.ail`, `simple_func_match.ail`,
   `match_arm_block.ail`, `match_in_block.ail`, `nested_match_minimal.ail` have no `main`
   entrypoint (helper/library files being swept as if runnable); `batch_processing.ail` and
   `cli_args_demo.ail` need an `Env` capability the generic checker never grants. In scope:
   `examples/*` is a multi-level-safe allowlist entry, so adding explicit `main` wrappers or
   capability-usage comments is a normal sprint. Do NOT touch
   `.claude/skills/docs-sync/scripts/check_examples.sh` itself (see docs-6). Source:
   `docs/docs-sync-findings.md` DOCS-2-05 through DOCS-2-13.
   **Landed by an orphaned "iteration 3" fire that died before Gate 4/5**: `0e8314549`
   ("docs(examples): docs-5 — add main entrypoints to 7 parser-regression fixtures", #997),
   credited and re-verified by iteration 4 (see log). `batch_processing.ail`/`cli_args_demo.ail`
   remain genuinely out of scope (Env-capability gap, not this item's job).
5. `[LANDED]` **docs-6 · clause 1 · fix `check_examples.sh`'s
   absolute-path bug.** The instrument this mission's whole clause-1/2 sweep depends on has been
   silently over-reporting broken examples (see docs-2's findings, DOCS-2-01). Fixing it means
   editing `.claude/skills/docs-sync/scripts/check_examples.sh`, which sits outside this mission's
   blast-radius allowlist (`.claude/skills/*` only covers `mission-control/SKILL.md` and
   `design-doc-creator/*`). Also folds in DOCS-2-04 (the audit_design_docs.sh vs
   derive_roadmap_versions.sh design-doc population-count mismatch — same script family, same
   scope question).
   **Landed by the same orphaned "iteration 3" fire**: `95396664b` ("fix(docs-sync): docs-6 —
   check_examples.sh absolute-path bug + audit scope comments", #1004). Re-verified by iteration 4
   with a fresh independent run: `173 passed / 2 failed (batch_processing.ail, cli_args_demo.ail —
   the known Env-capability gap) / 42 skipped`, matching the commit's own claim exactly.
6. `[LANDED]` **docs-10 · clause 1 · `make verify-examples` is vacuous on two independent axes —
   found by Gate 0's weekly external-issue sweep (2026-08-31), not by this mission's own tooling.**
   Docs-mission's Repo Profile names `make verify-examples` as one of its two verify-profile gates
   ("every published example still verifies"). Two open, unmentioned-anywhere-in-this-charter
   issues, both measured with a mutant-and-restore, show it cannot actually do that:
   [#670](https://github.com/sunholo-data/ailang/issues/670) — `expected.stdout` in
   `examples/manifest.json` is never compared (`scripts/verify_examples.go`'s `runExample` passes
   an example on `err == nil` alone; corrupting one entry's `expected.stdout` to a deliberately
   wrong literal still returns `rc=0`); [#654](https://github.com/sunholo-data/ailang/issues/654) —
   `scripts/validate_manifest.go` prints its `checked` count but never asserts it against a floor,
   so an enumeration that silently returns zero (a glob change, a path move) still prints a green
   "✓ … in sync" line and exits 0. Re-confirmed still present at this iteration's HEAD (`grep -c
   "checked == 0"` and `grep -c "expected.stdout"` both **0** in the two scripts). In scope:
   `scripts/*` is Go but not `internal/**`/`cmd/**`, so it is editable by this mission per the
   Guardrails' non-write-gate correction — it plans on `opus` rather than the cheap `codex` lane
   (not in `MISSION_PLANNER_ALLOWLIST`), which is a cost note, not a blocker. Sequenced after
   docs-6 since both are fixes to the sweep's own instruments, not to docs content.
   **Landed by the same orphaned "iteration 3" fire**: `9c5deac58` ("fix(docs-sync): docs-10 —
   verify-examples vacuous on two axes (#670, #654)", #1010) — indexed pinned `expected.stdout`
   once in `main()`, compared trailing-newline-insensitively, and added an anti-vacuity floor
   (atomic counter, race-checked under `--parallel`) that fails loudly if zero comparisons ran.
   Re-verified by iteration 4 with a fresh independent run of `make verify-examples`:
   **211 passed, 0 failed, 6 skipped**, manifest validator "193 modules checked, 0 drift" — a real,
   non-vacuous green, not inherited from the pre-fix instrument.
7. `[RULED OUT]` **docs-7 · "the mission cannot edit its own published content" — DISSOLVED
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
8. `[LANDED]` **docs-8 · clause 1 · 126 overdue planned design docs (aggregate) — corrected and
   triaged.** The "126" figure was itself stale (`derive_roadmap_versions.sh` count drift since
   docs-2/docs-6): re-run at this iteration's HEAD, the real overdue set (target version <
   v0.34.0) was **54** docs. All 54 were classified against the live codebase (grep evidence,
   commit citations, changelog cross-refs), then independently re-verified by a second,
   adversarial agent before any file moved — generator≠judge caught **3 of 22** high-stakes
   claims wrong (2 outright reversals: an abandoned/deleted experiment mis-read as shipped, a
   "retired" A/B mis-read as a settled negative when only its rig model was decommissioned; 1
   downgraded to partial). Result: **18 docs confirmed genuinely implemented**, moved to
   `design_docs/implemented/vX_Y/` with their sprint-plan companions (27 files total, git
   renames); **1 ruled out** with evidence written into its own header
   (`m-eval-slim-prompt-self-discovery.md`, PARKED, kept under `planned/` per this mission's
   negative-result convention, matching docs-9/docs-7); **1 flagged NEEDS-DEEPER-INVESTIGATION**
   (`m-fmt-deterministic-feedback` — its extension lives in a separate package repo not vendored
   here); **1 flagged partial/WEAK** (`m-eval-stream-health-retry` — TTFT/idle detection landed,
   the retry+correct-labeling half the doc is actually about did not); **31 confirmed genuinely
   STILL-PLANNED** — this is now the mission's accurate backlog (down from an unverified 126),
   individually pickable from `design_docs/planned/` by any future iteration without needing a
   new aggregate item. Full 54-row and 22-row tables in iteration 5's log entry.
9. `[LANDED]` **docs-1 · clause 7 · build the inbox-routing TRIGGER.** `send` and `forward` are
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
   **Landed iteration 4**: a died-mid-flight "iteration 3" left a complete brief + sprint plan
   unlanded (recovered as [PR #1016](https://github.com/sunholo-data/ailang/pull/1016)); iteration
   4 executed the plan on `codex:gpt-5.6-luna`, producing `tools/messaging/docs_inbox_router.sh`
   ([PR #1018](https://github.com/sunholo-data/ailang/pull/1018) → `e65e96b15`). Round-1 evaluator
   (sonnet, independent worktree) FAILED it 58/100: a genuinely empty (valid, zero-item) poll
   result crashed the router instead of reporting `checked=0 forwarded=0` — the ordinary,
   most-common case for a low-traffic inbox. Fixed and independently reproduced by the controller
   both broken (on the round-1 commit) and fixed (on the round-2 commit) with hand-built empty-
   response fixtures, not taken on the executor's word. Round-2 evaluator PASSED 90/100, zero
   blocking. CI green on the merge commit itself (`e65e96b15`) except the pre-existing,
   not-this-mission's-domain `SonarCloud Code Analysis` red (confirmed inherited from the parent
   commit, V1's territory per Gate 1's repo-ownership scoping). No launchd wiring in this sprint
   (explicitly out of scope per the brief) — the router runs by hand or under a future job.
10. `[IN-SPRINT]` **docs-3 · clause 6 · benchmark surface audit / provenance wiring.** Blocked on
   nothing, sequenced after docs-2 so the drift picture is known first. **Landed by an orphaned
   "iteration 5" fire** (a died-mid-flight run using `design_docs/docs-3-brief.md` +
   `docs-3-sprint-plan.md`, brief+plan landed via [PR #1023](https://github.com/sunholo-data/ailang/pull/1023)):
   wires `benchmarkFetchWithSource()` into the 4 `<DataProvenance>` call sites
   (`EloLeaderboard`, `BenchmarkStandaloneGallery`, `BenchmarkDashboard`, `BenchmarkExplorer`)
   that could never show the "⚠ stale (fallback copy)" badge, since only `ValueDashboard` passed
   `source=`. Independent evaluator (sonnet, isolated worktree): PASS 85/100, zero blocking. Diff
   scope verified exactly 4 files. [PR #1031](https://github.com/sunholo-data/ailang/pull/1031)
   is `MERGEABLE` but `mergeStateStatus: BLOCKED` — required checks `test`/`build` are red, and
   this is confirmed **inherited from `origin/dev`'s own current tip** (identical failures on
   HEAD and its parent — Build macos/windows/ubuntu, `launchd drivers (bash 3.2)`, `test`), a
   known V1-owned red (V1's own log carries 10+/22+ mentions of these two check names; motoko
   independently flagged the Windows half to V1 in the last hour). Not this mission's domain or
   fixable by a rebase (the red is on origin/dev's tip itself, not stale base). PR left open,
   verified and ready to merge the moment V1's red clears — resume point, not a re-pick.
11. `[IN-SPRINT]` **docs-4 · clause 5 · taxonomy pass — DESIGN-READY, sprint held on D-3.** Both
   original blockers cleared (clauses 1-3 green; docs-7's allowlist question dissolved). Design
   brief `design_docs/docs-4-brief.md` scopes it to ONE sprint (62 files, near-zero literal
   duplication measured, every file dispositioned) and passed quorum via this mission's first
   narrow-refinement carve-out (2 blocked rounds, both closed with the reviewers' own verbatim
   fixes — see iteration 6's log). Per the carve-out's ratification rule, `sprint-planner` does
   not run until Mark gives the one-time OK (D-3, below) — this is not a re-pick, it resumes the
   moment D-3 answers.

---
**Document created**: 2026-08-28 (attended, with Mark). **Bar RATIFIED attended 2026-08-28** after
three quorum rounds — see `docs-0` for why by human decision rather than by a passing quorum. Sprints
route from `docs-2` onward; each queue item still passes its OWN design quorum at its own gate, which
is where the surviving iteration-0 objections re-enter.
