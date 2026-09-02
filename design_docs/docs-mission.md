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
8. `[PARKED]` **docs-8 · clause 1 · 126 overdue planned design docs (aggregate).** Everything under
   `design_docs/planned/` targeting v0.29.0 through v0.31.0 while the repo ships v0.34.0
   (DOCS-2-03). `design_docs/` is also outside the current sprint allowlist — moving a doc to
   `implemented/` is normally the mission CONTROLLER's own Gate-4 bookkeeping (as v1-mission does),
   not a sprint-executor task, so this can proceed without an allowlist change, just not via the
   automated inner loop. Sequence after docs-6/docs-7 land or are ruled out, since triaging 126
   docs by hand is exactly the kind of large sweep worth doing once against settled tooling.
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
10. `[PARKED]` **docs-3 · clause 6 · benchmark surface audit.** Blocked on nothing, but sequence it
   after docs-2 so the drift picture is known first.
11. `[PARKED]` **docs-4 · clause 5 · taxonomy pass.** The `docs/docs/guides/` directory holds 40+
   guides accreted over time. Deferred until clauses 1-3 are green — consolidating pages that are
   also factually stale does both jobs badly. Also blocked on docs-7's allowlist question, since
   `docs/docs/guides/` is nested.

---
**Document created**: 2026-08-28 (attended, with Mark). **Bar RATIFIED attended 2026-08-28** after
three quorum rounds — see `docs-0` for why by human decision rather than by a passing quorum. Sprints
route from `docs-2` onward; each queue item still passes its OWN design quorum at its own gate, which
is where the surviving iteration-0 objections re-enter.
