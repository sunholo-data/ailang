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
| D-3 | RESOLVED | **ASK (iteration 6, 2026-09-03): one-time OK to use the shared mission-control skill's narrow-refinement carve-out for `docs-4`'s design brief, docs-mission's first use of it.** The brief (`design_docs/docs-4-brief.md`) blocked quorum twice; both rounds' objections were narrow, concrete (each reviewer supplied a verbatim `proposed_fix`), and disputed no design direction — round 1: an unprobed 9th orphan URL, an unverified section-heading pair; round 2: imprecise "URL-stable" wording against two intentional clause-5-authorised deletions, an unverified 3rd heading boundary, a cleanup step left as prose instead of an encoded acceptance criterion. The controller applied all fixes verbatim (commits `fbc289f6a`, `56acda30d`) rather than spending a 3rd $0.12 quorum round. Options: **(a)** OK it — the doc is design-ready, `sprint-planner` runs next iteration. **(b)** Reject it — the item goes back to a normal 3rd quorum round (cost: ~$0.12, one more iteration). Loop recommendation: (a) — both rounds' objections read as the reviewers doing their job on a genuinely fixable doc, not as a doc that shouldn't ship. Default if unanswered: stays parked at design-ready: no sprint runs, no cost accrues, `docs-4` sits at the top of next iteration's queue exactly as now.  **ANSWERED — (a) APPROVED — use the narrow-refinement carve-out for `docs-4`, with ONE condition to satisfy before the sprint runs: close `gemini-3-1-pro`'s recurring objection class exhaustively. Verify EVERY section-cut boundary asserted in the brief and record each as a Verification Log row — not only the B3/B4/B5 that reviewers happened to name. Round 1 asked for B4/B5; round 2 asked for B3 "with the same rigor V29 just gave B4/B5". That is ONE class recurring at new instances, which is the shape that predicts a 3rd round finds a 4th. The check is a single-command verification with no design judgment (the controller already ran it twice, as V29 and V30), so it costs about nothing against the ~$0.12 plus one 6h iteration a 3rd quorum round would cost. Recorded against the carve-out rather than hidden: the skill's own discriminating test does NOT cleanly license it here — no reviewer started passing (round 2 was 3 of 3 reject) and the objections are spread across three surfaces (wording precision, verification, acceptance-criterion encoding), which the rule says means keep revising. Granted anyway for two reasons. First, round 2 is confounded: `oc-glm-5-2` was ABSENT in round 1, so its round-2 reject is a first look at the doc rather than a failure to converge, and both reviewers who saw BOTH rounds narrowed. Second, blast radius is low — a reversible docs taxonomy pass that still passes through the normal executor and evaluator gates. Precedent is scoped to `docs-4` alone and does not generalise to docs-mission.** (Mark Edmondson, attended 2026-09-03, recorded directly in this ledger.)| `design_docs/docs-4-brief.md` §"Quorum log"; quorum artifacts `docs-4-brief-2026-09-03T10-54-51Z.json`, `docs-4-brief-2026-09-03T10-57-26Z.json`; carve-out rule text in `.claude/skills/mission-control/SKILL.md` Gate 2.  **Attended ruling 2026-09-03** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Mark gave this ruling in an attended session; the agent transcribed it and the commit is authored by the fleet bot, because this machine's git identity is the fleet bot for every session including Mark's. Provenance is therefore the attended session, NOT the commit author — and nothing in `scripts/mission_decisions.sh` or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-03, positive-controlled), so no control is weakened by saying so plainly rather than asserting an authorship that did not happen. design_docs/docs-4-brief.md section "Quorum log" (lines 19-42); quorum artifacts docs-4-brief-2026-09-03T10-54-51Z.json (round 1, oc-glm-5-2 absent, degraded to N-1) and docs-4-brief-2026-09-03T10-57-26Z.json (round 2, 3 of 3 present, 3 of 3 reject); carve-out discriminating rule at .claude/skills/mission-control/SKILL.md lines 751-754 ("the signal is not the round COUNT, it is where the objections LAND"); attended session 2026-09-03, ruling given by Mark after review of both rounds objection-by-objection.|
<!-- decision-ledger:end -->

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `docs-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Every
iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the scarcest
model budget; the append-only history lives in the log + archive.

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

## STATUS 2026-09-03 — ITERATION 7: docs-3's "V1-owned inherited red" verdict RE-MEASURED and found stale — dev had already fixed it; rebased and landed [PR #1031](https://github.com/sunholo-data/ailang/pull/1031); docs-4 still held on D-3

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD detached
at `origin/dev` tip (`70e453060`), clean. 0 directives on bookkeeping issue `#979` since the
watermark (8 comments, none allowlisted). Decision ledger valid, 3 rows — D-1/D-2 `RESOLVED`,
**D-3 still `OPEN`** (the narrow-refinement-carve-out ask for `docs-4`, unanswered since
iteration 6; last touch on that row is the fleet bot itself, not an attended ruling — correctly
left open, not actioned). 20 unread canonical-inbox messages, none addressed to `mission-docs`
(eval-suite run notifications, `mission-world`/`mission-v1` cross traffic, `pkg:*` package
inboxes) — none acked, per the "ack --all sweeps outbound cross-mission inboxes" hazard, and none
outrank the queue.

Gate 1: `origin/dev` HEAD (`70e453060`) SHA-addressed check-runs: 14 checks, only `test`
non-green and it was `pending` (CI genuinely in-flight, not red) — no dev-red to action.

**PICK: `docs-3`** (item 10, `[IN-SPRINT]`), via the blocked-external-row re-verification rule
(Gate 2): iterations 1/2/4/5/6 had each re-asserted "still V1-owned inherited red, not fixable by
rebase" on [PR #1031](https://github.com/sunholo-data/ailang/pull/1031) without re-running the
predicate. Re-measured this iteration: `test`/`Build ubuntu-latest`/`launchd drivers (bash 3.2)`
were GREEN on two independent recent `dev` commits (`08ab6ba7c`, `5506424f8`), while the PR's own
head (`178072e3f`) was ~40 commits behind `origin/dev` — the red was base-inherited and dev had
already been fixed underneath it. The predicate had flipped; nobody had re-run it.

**Execution (mechanical — no new design/plan/execute pass owed; the diff was already designed,
executed, and independently evaluated PASS 85/100 in iteration 5).** Reused the existing
`.wt-docs-iter5-docs3` worktree (already on the PR branch), fetched + `git rebase origin/dev`
(clean, no conflicts — confirmed no overlapping files touched on `dev` since the PR's base),
verified `git diff --stat origin/dev HEAD` was byte-for-byte the same 4 files as before the
rebase, reverted incidental regenerated-file drift picked up while locally verifying
`make docs-build` (`design-docs.md`, `current.md`, `roadmap/index.md`, `packages-sidebar.json` —
sync-script byproducts, never meant to be committed — rule: a control you record is a control you
spend, applied to accidental commits instead), force-pushed, watched all 16 PR checks go green
(`test`, `docs-build`, `SonarCloud`, `build`, both `Build *-latest`, `launchd drivers`,
`govulncheck`, `lint`, `CodeQL`, `docs-gate`, `docs-changes`, `test-windows`, `Analyze Go`),
scanned the PR title/body for GitHub auto-close keywords (none, control fired correctly on a
known-bad string), squash-merged → `663237dc7`. Re-polled CI on the **merge commit itself**
(squash-merge produces a different commit than the tested PR head) to completion: 16/16 green
except a `SonarCloud Code Analysis` red confirmed present on the merge commit's own parent
(`41ea6e5ff`) — pre-existing, V1's domain, not caused by this diff.

**generator≠judge — independent evaluator spawned post-hoc** (per this run's standing
instruction that a judge is required even for a mechanical landing the controller itself
verified): `Agent(subagent_type="general-purpose", model="sonnet")`, given none of the
controller's own findings, independently re-pulled the merge commit's diff, re-pulled check-runs
for both the merge commit and its parent, and independently confirmed the pre-rebase PR head was
genuinely stale (91 commits behind `origin/dev`, not an ancestor after the rebase). **Verdict:
PASS** — diff scoped to exactly the 4 files with content matching the stated purpose, no scope
creep, CI check-runs match the controller's claim exactly, SonarCloud red independently confirmed
pre-existing on the parent commit.

**Local verification note (not blocking, filed as a follow-up):** `make docs-build` fails on ANY
fresh checkout, including a pristine `origin/dev` — `docs/src/data/packages-sidebar.json` (tracked)
references package doc pages that are gitignored/generated by `docs/scripts/sync-registry.sh`,
which CI's `docusaurus-deploy.yml` runs before `make docs-build` but this mission's own `Makefile`
target does not. Running `sync-registry.sh` first reproduces CI's real gate and the build then
proceeds correctly (confirmed — hit a second, unrelated `Cannot read properties of undefined
(reading 'id')` SSG error afterward that was not chased further, since CI's own `docs-build` job
— which runs the full pipeline including a fresh registry sync — passed cleanly on both the PR
and the merge commit). Iteration 5/6's evaluator had already flagged the symptom as "identical on
baseline and branch" without finding the missing step; this iteration found it. Worth a
`docs-sync`/Makefile fix so local verification is self-contained, out of scope for docs-3 itself.

**Routing evidence**: no designer/planner/executor spawned (nothing to design/plan/execute — the
code pre-existed from iteration 5's orphaned fire and was already evaluated); one evaluator spawn,
`sonnet` (Agent tool, distinct from the controller's own session model) — PASS. Controller session:
sonnet.

**Cost**: metered $0.00 of $1 ceiling (no codex/pi/quorum calls this iteration — rebase, CI polling,
merge, and evaluator spawn are all quota-bucket or free). Quota buckets: sonnet (controller session
+ evaluator sub-agent).

**docs-4 unchanged**: still `[IN-SPRINT]`, design-ready, held on D-3 (unanswered). Not re-picked —
Standing rule 1 (one backlog item per iteration) and D-3 is a judgment park, not a capacity one; no
predicate to re-run, only Mark can answer it.

Full record: `design_docs/docs-mission-log.md` §ITERATION 7.

## STATUS 2026-09-04 — ITERATION 8: docs-4 LANDED — D-3's condition satisfied, sprint executed, an independent evaluator caught one real defect neither the controller nor the executor saw, fixed and re-verified PASS 97/100

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD detached
at `origin/dev` tip (`2b5750ad9`), clean. 0 directives on bookkeeping issue `#979` since the
watermark. Decision ledger valid, 3 rows — D-1/D-2/D-3 all `RESOLVED` (D-3 answered by Mark,
attended, 2026-09-03, in the prior iteration's window: APPROVED the narrow-refinement carve-out
for docs-4, with ONE condition — close `gemini-3-1-pro`'s recurring section-boundary-verification
objection class exhaustively, not only the B3/B4/B5 rounds happened to name). 16 unread
canonical-inbox messages (mission-v1/mission-world cross traffic, `pkg:*` package-agent task
events, an `ailang-parse-claude`↔`aitana-platform` thread) — none addressed to `mission-docs`,
none outrank the queue.

Gate 1: `origin/dev` HEAD (`2b5750ad9`) — `CI`/`Build and Release` both `success`;
SHA-addressed check-runs: 16 checks, only `SonarCloud Code Analysis` non-green, confirmed
inherited and already tracked by V1 (its own iteration-326 report names the same red as "1
inherited SonarCloud") — not this mission's domain, not actioned.

**PICK: docs-4** (item 11, `[IN-SPRINT]`, held on D-3 — now resolved). Before routing, satisfied
D-3's condition: grepped the brief for every Phase-B section-cut boundary and found B1's carry-over
(the `Automated Feedback (Advanced)` bash snippet moved from `cross-project-messaging.mdx` into
`agent-messaging.md`) had no Verification Log row — B3/B4/B5 all had one (V29/V30), B1 didn't.
Measured directly (`grep -nE '^##+ ' docs/docs/guides/cross-project-messaging.mdx`): line 232
heading immediately followed by line 257's next H2, both genuine H2s, body matching the claimed
~15-line bash block exactly. Added as V31, committed and pushed
([`df36055ce`](https://github.com/sunholo-data/ailang/commit/df36055ce)), CI green.

**Routing (Gate 3), full pipeline, per the routing table:**
- **sprint-planner**: `codex:gpt-5.6-luna` (cross-provider recipe, ephemeral detached worktree,
  30-min bounded run). First attempt died on a wrapper-script escaping bug (the directive's own
  parentheses broke an unquoted heredoc that had interpolated it into the wrapper file) — caught by
  `bash -n` syntax-checking the wrapper before every subsequent launch, never again by trusting a
  launcher's own exit code (Standing rule 7's "a notification for a command containing `&` means
  launched, not done" — this was the same class one level up, a malformed *script*, not a stale
  read). Second attempt: rc=0, produced `design_docs/docs-4-sprint-plan.md` +
  `sprint_docs-4.json`, 6 milestones (Phase A, B1-B5 in order), every brief acceptance check
  encoded as a literal command including the mandatory sync-registry cleanup ordered before the
  scope check — [`72902585d`](https://github.com/sunholo-data/ailang/commit/72902585d), CI green.
- **sprint-executor**: same codex lane, isolated worktree, multi-milestone snapshot protocol
  (`.snap/M<k>/`, zero git-write operations delegated). rc=0, all 6 `.snap/` directories present
  and cumulative. Reconstructed 6 individual commits from the snapshots in the pin worktree
  (sha256-verified byte-identical to the executor's own final tree before committing) —
  [`2a336cfde`..`67a76e0a9`](https://github.com/sunholo-data/ailang/commit/67a76e0a9).
  **The executor's own in-sandbox acceptance run reported 2 failures out of 8; both required
  controller follow-up, for different reasons:**
  - Check 6 (`sync-registry.sh && make docs-build`) failed in-sandbox
    ("Could not fetch registry index") — a `codex --sandbox workspace-write` network restriction,
    not a real defect: re-run unsandboxed by the controller (rule: in-sandbox verdicts are not
    evidence, generator≠judge extends to the controller's own re-verification duty), both commands
    succeeded cleanly. **But the SAME unsandboxed rebuild surfaced a genuine, different failure**:
    `make docs-build` threw `Docusaurus found broken links!` — 4 dangling relative-path links
    (`./cross-project-messaging.mdx` × 3 in `agent-workflows.mdx`/`claude-code-integration.mdx`/
    `hooks-setup.mdx`, `./development.md` × 1 in `getting-started.mdx`) that acceptance check 4's
    `guides/`-prefixed grep pattern never matches (a sibling-directory relative link inside
    `docs/docs/guides/` carries no `guides/` prefix) — the exact same instrument gap V14's
    inbound-link discovery had. Fixed as **M7**: retargeted the 3 cross-project-messaging
    references to `agent-messaging.md` (B1's merge target) and the development.md reference to
    `/docs/guides/development-workflow` (matching the precedent the brief already set for
    `debugging.md`'s equivalent link). Both edited files sit under `docs/docs/guides/**`
    (the brief's declared blast radius) though outside its Files-list enumeration — required by
    B1/B2's own deletion criterion ("every inbound link is retargeted"), not scope creep. Rebuilt
    clean (exit 0, only 3 pre-existing broken-anchor warnings confirmed present identically on
    the pre-sprint base commit) —
    [`1afa42f37`](https://github.com/sunholo-data/ailang/commit/1afa42f37), CI green.
  - Check 5 (both halves: `ailang messages ack --all` count, `_ollama_embed` exclusion) failed
    in-sandbox AND out-of-sandbox — investigated and found to be a defect in the BRIEF's own
    acceptance criteria at authoring time, not an execution defect. First half: the brief claims
    "was 4: hooks-setup and cross-project-messaging gone", but the brief's own V8 verification row
    never listed `cross-project-messaging.mdx` among the 4 matching files, and `hooks-setup.mdx`
    has TWO occurrences pre-sprint (one inside the trimmed `Message System` section, one inside a
    separate, never-in-scope `Quick Start § 3. Check Messages` subsection) — so the count could
    never drop to 2 regardless of correct execution; the real, achievable count is 4 (unchanged,
    since 2 of the 4 files — `agent-workflows.mdx`, `claude-code-integration.mdx` — were never
    in scope to edit at all). Second half: `_ollama_embed` legitimately survives in
    `semantic-caching-how-to.mdx`'s `Embeddings Doctrine` section, which the brief explicitly
    preserves (only `Two-Tier Search Architecture` is trimmed) — verified by comparing pre/post
    heading positions of every occurrence. This is filed as a brief-authoring defect, not
    force-passed or hidden: check 5 is unsatisfiable as literally written, and both corrected
    expectations were independently re-derived by the round-1 evaluator (see below) before being
    trusted.

**generator≠judge — independent evaluator, `sonnet` (Agent tool, isolated worktree from the repo,
distinct from the codex executor):**
- **Round 1: FAIL 68/100.** One BLOCKING finding, found independently and not flagged by the
  controller's own review: M1's sidebar rewrite silently dropped two NON-GUIDE ids —
  `prompts/index`, `prompts/current` — while dissolving the `Prompts` category, violating Appendix
  B's explicit "non-guide ids unchanged" rule and creating two fresh nav-orphaned pages (both still
  exist on disk with live inbound links) — exactly the clause-3 orphan-page defect class this whole
  sprint exists to eliminate, invisible to all 8 of the brief's `guides/`-scoped acceptance checks.
  All other findings (the check-5 corrections, the M7 self-correction's scope, 3 pre-existing
  broken-anchor warnings, B1-B5 boundary fidelity, 9-orphan wiring, 3-deletion count, no content
  rewrites) verified clean.
  Reproduced first-party before acting (rule: a judge's finding is a claim too): re-ran the
  evaluator's exact sidebar-id diff command, confirmed both ids missing, confirmed both files exist
  on disk with real inbound links from 4 other pages. Fixed as **M8**: restored both ids to their
  original relative position (immediately before `guides/ai-prompt-guide`, their sole surviving
  category sibling) — 2-line surgical diff. Rebuilt clean, cleanup step clean —
  [`0750f8dbf`](https://github.com/sunholo-data/ailang/commit/0750f8dbf), CI green (20 checks,
  only the same inherited SonarCloud red).
- **Round 2: PASS 97/100** (fresh worktree, same evaluator model, carrying forward round 1's
  findings by name per the multi-round protocol). Confirmed-fixed: the sidebar-id diff now shows
  only the intended set; both restored ids reachable and wired at their original position; fix
  commit surgical (2 insertions, 0 deletions, nothing else touched); full clean rebuild from a
  fresh `npm install`, no new warnings; acceptance checks 1-4/6-8 re-spot-checked, no regressions.
  3 points held back only for the still-open, already-disclaimed check-5 brief defect and the 3
  pre-existing broken-anchor warnings — neither this sprint's nor this fix's responsibility.

**Landed**: 8 commits total (`df36055ce` V31 → `72902585d` plan → `2a336cfde..67a76e0a9` M1-M6 →
`1afa42f37` M7 link-fix → `0750f8dbf` M8 sidebar-restore), all CI-green on `origin/dev`. `docs-4`
tag flipped `[IN-SPRINT]` → `[LANDED]` above.

**Routing evidence**: designer NOT spawned (doc already existed, quorum-passed, only the D-3
condition's one narrow gap needed closing — done by the controller directly, matching how V29/V30
were closed in iteration 6, no design judgment involved); planner `codex:gpt-5.6-luna` (recipe
path, 2 attempts, 1 wrapper-script bug, not a lane failure); executor same lane (1 attempt, clean);
evaluator `sonnet` × 2 rounds (Agent tool, distinct provider from the codex executor — generator≠
judge holds both rounds). Controller session: opus.

**Cost**: metered **$0.00** of $1 ceiling — codex is quota-bucket, not billed per-token on this
lane; both evaluator rounds were Anthropic-quota Agent-tool spawns. Quota buckets: opus
(controller), codex (planner + executor, quota-bucket lane), sonnet (evaluator × 2).

**Progress**: N = **11** design docs remaining before v1.0.0 backlog exhausted (down from 12 —
docs-4 was the last `[IN-SPRINT]`/`[NEXT]` item; the queue's remaining rows are all `[LANDED]` or
`[RULED OUT]`). Next iteration's pick is a fresh queue draw from `design_docs/planned/` (docs-8's
31 confirmed still-planned docs) since this charter's own enumerated backlog is now exhausted.

Full record: `design_docs/docs-mission-log.md` §ITERATION 8.

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
10. `[LANDED]` **docs-3 · clause 6 · benchmark surface audit / provenance wiring.** Blocked on
   nothing, sequenced after docs-2 so the drift picture is known first. **Landed by an orphaned
   "iteration 5" fire** (a died-mid-flight run using `design_docs/docs-3-brief.md` +
   `docs-3-sprint-plan.md`, brief+plan landed via [PR #1023](https://github.com/sunholo-data/ailang/pull/1023)):
   wires `benchmarkFetchWithSource()` into the 4 `<DataProvenance>` call sites
   (`EloLeaderboard`, `BenchmarkStandaloneGallery`, `BenchmarkDashboard`, `BenchmarkExplorer`)
   that could never show the "⚠ stale (fallback copy)" badge, since only `ValueDashboard` passed
   `source=`. Independent evaluator (sonnet, isolated worktree): PASS 85/100, zero blocking. Diff
   scope verified exactly 4 files.
   **Landed iteration 7**: iterations 1/2/4/5/6 each re-confirmed the same "V1-owned inherited
   red" verdict on [PR #1031](https://github.com/sunholo-data/ailang/pull/1031) without
   re-measuring whether it was still true (Gate 2's blocked-external-row predicate rule) —
   it wasn't. `origin/dev`'s tip had since been fixed for `test`/`Build ubuntu-latest`/
   `launchd drivers (bash 3.2)` (confirmed green on two independent recent dev commits,
   `08ab6ba7c` and `5506424f8`, while the PR's stale head (`178072e3f`, ~40 commits behind)
   still showed all three red). Rebased the existing worktree
   (`.wt-docs-iter5-docs3`, already on the PR branch) onto `origin/dev`, verified the diff scope
   was unchanged (still exactly the 4 files), reverted incidental regenerated-artifact drift from
   local verification (`design-docs.md`, `current.md`, `roadmap/index.md`,
   `packages-sidebar.json` — sync-script byproducts, never meant to be committed), and
   force-pushed. All 16 PR checks including `test`, `docs-build`, `SonarCloud` went green;
   squash-merged as `663237dc7`. Re-verified CI green on the **merge commit itself** (not just
   the PR head — squash-merge produces a different commit), 16/16 checks except a pre-existing
   `SonarCloud Code Analysis` red confirmed present on the merge commit's own parent
   (`41ea6e5ff`, not touched by this diff) — inherited, V1's domain, not blocking.
   **Local verification note**: `make docs-build` alone fails on ANY fresh checkout (including a
   pristine `origin/dev`) because the tracked `docs/src/data/packages-sidebar.json` references
   package doc pages that are gitignored/generated (`docs/scripts/sync-registry.sh`, which CI's
   `docusaurus-deploy.yml` runs before `make docs-build` and this mission's own Makefile target
   does not) — a local-environment gap, not a code defect; running `sync-registry.sh` first
   reproduces CI's real gate. Iteration 5/6's evaluator had already flagged the symptom as
   "identical on baseline and branch"; this iteration found the actual missing step. Worth a
   `docs-sync`/Makefile follow-up so `make docs-build` is self-contained for local verification,
   but out of scope for docs-3 itself.
11. `[LANDED]` **docs-4 · clause 5 · taxonomy pass — LANDED iteration 8.** D-3's attended approval
   (below) carried one condition: close `gemini-3-1-pro`'s recurring section-boundary-verification
   objection class exhaustively (every Phase-B boundary, not only the B3/B4/B5 reviewers named).
   Satisfied first: B1's carry-over boundary in `cross-project-messaging.mdx` had no Verification
   Log row; added V31 (`232:## Automated Feedback (Advanced)` immediately followed by
   `257:## Semantic Search`, both genuine H2s, body matching the claimed ~15-line bash block
   exactly) — [`df36055ce`](https://github.com/sunholo-data/ailang/commit/df36055ce).
   Routed to `sprint-planner` (`codex:gpt-5.6-luna`, cross-provider recipe): 6-milestone plan
   faithfully translating Phase A + B1-B5 in order, every brief acceptance check encoded as a
   literal command — [`72902585d`](https://github.com/sunholo-data/ailang/commit/72902585d).
   Routed to `sprint-executor` (same codex lane, isolated worktree, snapshot-per-milestone
   protocol, zero git writes): M1-M6 executed, 3 deletions + 9 orphans wired + 5 redundant
   sections trimmed at their exact verified boundaries —
   [`2a336cfde`..`67a76e0a9`](https://github.com/sunholo-data/ailang/commit/67a76e0a9) (6 commits).
   **Two in-sandbox check failures were sandbox artifacts, not real defects** (re-verified
   unsandboxed by the controller per generator≠judge discipline): `sync-registry.sh`'s network
   fetch is blocked under codex's `workspace-write` sandbox; outside it, both `sync-registry.sh`
   and `make docs-build` succeeded cleanly. **One in-sandbox check failure WAS real**: the
   executor's own acceptance-check run correctly flagged both halves of check 5 (`ailang messages
   ack --all` count, `_ollama_embed` exclusion) — investigated and found to be a defect in the
   BRIEF's own acceptance criteria, not the execution: `cross-project-messaging.mdx` never
   contained the ack-all string (contradicts its own "was 4: hooks-setup and cross-project-
   messaging gone" reasoning — V8 never listed it), and `_ollama_embed` legitimately survives in
   the explicitly-preserved `Embeddings Doctrine` section by the brief's own design. **A genuine,
   independently-confirmed build-breaking defect was also found and fixed**: `make docs-build`
   failed with 4 dangling relative-path links (`./cross-project-messaging.mdx` × 3,
   `./development.md` × 1) that check 4's `guides/`-prefixed grep pattern never matched — fixed as
   M7 (controller-authored, within the brief's declared `docs/docs/guides/**` blast radius, required
   by B1/B2's own "retarget every inbound link" deletion criterion) —
   [`1afa42f37`](https://github.com/sunholo-data/ailang/commit/1afa42f37).
   **generator≠judge, independent evaluator (`sonnet`, isolated worktree) round 1: FAIL 68/100** —
   one BLOCKING finding neither the controller nor the executor caught: M1's sidebar rewrite
   silently dropped two NON-GUIDE ids (`prompts/index`, `prompts/current`) while dissolving the
   `Prompts` category, violating Appendix B's explicit "non-guide ids unchanged" rule and creating
   two fresh nav-orphaned pages — exactly the clause-3 defect class this sprint exists to
   eliminate, invisible to all 8 of the brief's `guides/`-scoped acceptance checks. Verified
   first-party before acting (full sidebar-id diff, confirmed both files still on disk with live
   inbound links), fixed as M8 (2-line surgical restore to original relative position) —
   [`0750f8dbf`](https://github.com/sunholo-data/ailang/commit/0750f8dbf). **Round 2 (fresh
   worktree, same evaluator model): PASS 97/100**, blocking finding confirmed-fixed, fix commit
   confirmed surgical (2 insertions, 0 deletions, nothing else touched), full rebuild clean, no
   regressions. CI green on the merge tip (20 checks, only the pre-existing inherited SonarCloud
   red — V1's domain). 8 commits total, zero metered spend (codex is quota-bucket, not billed
   per-token on this lane).

---
**Document created**: 2026-08-28 (attended, with Mark). **Bar RATIFIED attended 2026-08-28** after
three quorum rounds — see `docs-0` for why by human decision rather than by a passing quorum. Sprints
route from `docs-2` onward; each queue item still passes its OWN design quorum at its own gate, which
is where the surviving iteration-0 objections re-enter.
