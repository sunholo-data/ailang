# Docs Mission Log

Append-only. One entry per iteration, newest at the BOTTOM.
Created 2026-08-28. Iteration 0 (charter ratification) has not yet run.

#### Design-quorum review — `design_docs/docs-mission.md` (2026-08-28T06:29:56Z)

- **Synthesis: BLOCKED** (total $0.0569, 15839 in / 990 out tok)
- `gpt5-6-sol` → **reject** ($0.0358, 5172/331 tok) — Clause 7 and queue item docs-1 require implementing inbox routing despite the document stating that the existing feedback-dispatch path is broken, while the mission’s mechanical allowlist forbids changes under `internal/` or `cmd/`. No extension mechanism within the permitted blast radius is identified, so a mandatory deliverable cannot proceed under the charter as written.
- `gemini-3-1-pro` → **reject** ($0.0133, 5545/181 tok) — The document establishes Clause 7 (working the `docs-mission` inbox) but leaves the inbox read path as an unverified premise. It warns that a bare `ailang messages list` reads a local SQLite shadow database instead of the canonical prod Firestore, but fails to provide or verify the correct command to actually read the prod inbox, rendering the task unexecutable.
- `oc-glm-5-2` → **reject** ($0.0079, 5122/478 tok) — Premise verification failure on clause 7's core deliverable: the doc asserts 'ailang messages send <inbox> takes a free-form inbox name, so docs-mission needs no registration' and 'ailang messages forward <id> --to docs-mission exists' as verified facts, but presents NO verification log — no command output, no rc, no date — unlike every other verification claim in the doc which is explicitly dated and rc=0 stamped. This is the single clause whose queue item (docs-1) is 'build the inbox routing,' yet the doc already asserts the primitives exist without evidence. If the premise is wrong, the entire docs-1 sprint plan collapses.
- controller (in-session, not an API call) → **pass** — Charter is well-scoped and internally consistent. Clauses 1-6 map to existing instruments (docs-sync skill, make docs-build/verify-examples, CHANGELOG/nav diff, benchmark JSON files) rather than inventing new machinery. Clause 7 honestly states it has no delivery mechanism yet and names that as iteration-0 work rather than assuming one. Guardrails correctly restrict blast radius to docs/examples/README/CHANGELOG and forbid GPU/rig.lock use. Routing policy's cost ladder (subscription -> flat-rate -> metered) is measured (codex gpt-5.6-luna and ollama glm-5.3-flash:cloud both probed rc=0 same day) rather than assumed, and the evaluator is kept vendor-disjoint from the executor at every rung (generator!=judge). One residual risk not addressed in the doc: this mission and V1 share one repo (sunholo-data/ailang) from separate clones, and V1's CI path filters include CHANGELOG.md, so a collision on that file between missions is possible; not a ratification blocker, flagging for the backlog. PASS from the controller.
- Blocking objections (return to author before planning):
  - gpt5-6-sol: Clause 7 and queue item docs-1 require implementing inbox routing despite the document stating that the existing feedback-dispatch path is broken, while the mission’s mechanical allowlist forbids changes under `internal/` or `cmd/`. No extension mechanism within the permitted blast radius is identified, so a mandatory deliverable cannot proceed under the charter as written.
  - gemini-3-1-pro: The document establishes Clause 7 (working the `docs-mission` inbox) but leaves the inbox read path as an unverified premise. It warns that a bare `ailang messages list` reads a local SQLite shadow database instead of the canonical prod Firestore, but fails to provide or verify the correct command to actually read the prod inbox, rendering the task unexecutable.
  - oc-glm-5-2: Premise verification failure on clause 7's core deliverable: the doc asserts 'ailang messages send <inbox> takes a free-form inbox name, so docs-mission needs no registration' and 'ailang messages forward <id> --to docs-mission exists' as verified facts, but presents NO verification log — no command output, no rc, no date — unlike every other verification claim in the doc which is explicitly dated and rc=0 stamped. This is the single clause whose queue item (docs-1) is 'build the inbox routing,' yet the doc already asserts the primitives exist without evidence. If the premise is wrong, the entire docs-1 sprint plan collapses.

#### Design-quorum review — `design_docs/docs-mission.md` (2026-08-28T06:35:00Z)

- **Synthesis: BLOCKED** (total $0.0617, 18177 in / 886 out tok)
- `gpt5-6-sol` → **reject** ($0.0381, 5923/283 tok) — Queue item docs-1 is not executable under the charter’s own mechanically enforced blast radius. Its proposed implementations require `tools/` or CI configuration, while the allowlist permits only `docs/`, `examples/`, `README.md`, and `CHANGELOG.md`; leaving this as an open question does not provide a valid route to deliver mandatory clause 7.
- `gemini-3-1-pro` → **reject** ($0.0150, 6377/190 tok) — The document violates the 'no silent fallbacks' axiom by acknowledging a silent fallback in derive-planner-lane.sh ('a bare sonnet pin emits opus fail-closed:env-pin and silently runs opus') and choosing to work around it via pin selection instead of fixing the script. A script that silently runs the fleet's most expensive model on an invalid pin is a critical hazard that must be fixed, not routed around.
- `oc-glm-5-2` → **reject** ($0.0085, 5877/413 tok) — Premise verification is systematically asserted, not verified. The doc claims CI workflow behavior — 'this repo has no push paths filter, so a docs-only commit still runs full CI' — without citing or linking the actual workflow YAML. The claim that `Deploy Documentation to GitHub Pages` is path-filtered on `docs/**, prompts/**, llms.txt, CHANGELOG.md` is presented as fact but the workflow file is never shown or referenced by path. If either premise is wrong, the entire Gate 3b polling strategy (wait for CI vs. read its absence as 'not applicable') is built on sand. This is the exact failure mode Gate 1 exists to prevent.
- controller (in-session, not an API call) → **pass** — Round-2 verdict. The designer's revision (spawned as a pinned sonnet sub-agent, independent of this controller's round-1 pass verdict) addresses all three round-1 objections: (1) gpt5-6-sol's blast-radius objection — docs-1 now explicitly states send/forward need no internal/cmd change and surfaces the tools/ scope question as an open, one-word-answerable ask for Mark rather than assuming an answer; (2) gemini-3-1-pro's missing-read-command objection — the correct AILANG_MESSAGES_STORE=gcp/AILANG_MESSAGES_PROJECT read command is now cited explicitly in clause 7; (3) oc-glm-5-2's unverified-primitives objection — clause 7 now carries a dated, rc=0-stamped verification log for send/forward/list run live against prod Firestore, with positive and negative read controls, matching the doc's existing verification-log style. I independently ran these exact commands myself before handing them to the designer (send to a scratch inbox, forward --to docs-mission, list --inbox docs-mission --unread confirming arrival), so this is not merely the designer's claim -- it is a first-party-verified fact. The CURRENT-GOAL/Queue inconsistency is also resolved in favor of the Queue's sequencing. Scope was held narrow: clauses 1-6, Guardrails, Routing policy, and the not-yet-ratified/ARMED-BUT-SILENT STATUS language are all confirmed unchanged. PASS from the controller.
- Blocking objections (return to author before planning):
  - gpt5-6-sol: Queue item docs-1 is not executable under the charter’s own mechanically enforced blast radius. Its proposed implementations require `tools/` or CI configuration, while the allowlist permits only `docs/`, `examples/`, `README.md`, and `CHANGELOG.md`; leaving this as an open question does not provide a valid route to deliver mandatory clause 7.
  - gemini-3-1-pro: The document violates the 'no silent fallbacks' axiom by acknowledging a silent fallback in derive-planner-lane.sh ('a bare sonnet pin emits opus fail-closed:env-pin and silently runs opus') and choosing to work around it via pin selection instead of fixing the script. A script that silently runs the fleet's most expensive model on an invalid pin is a critical hazard that must be fixed, not routed around.
  - oc-glm-5-2: Premise verification is systematically asserted, not verified. The doc claims CI workflow behavior — 'this repo has no push paths filter, so a docs-only commit still runs full CI' — without citing or linking the actual workflow YAML. The claim that `Deploy Documentation to GitHub Pages` is path-filtered on `docs/**, prompts/**, llms.txt, CHANGELOG.md` is presented as fact but the workflow file is never shown or referenced by path. If either premise is wrong, the entire Gate 3b polling strategy (wait for CI vs. read its absence as 'not applicable') is built on sand. This is the exact failure mode Gate 1 exists to prevent.

## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)

**Pick**: `docs-0` (queue head; unattended run, no allowlisted directive or genuine regression to
outrank it — Gate 0 found zero docs-mission-specific inbox traffic and zero comments on
bookkeeping issue #953).

**Routing evidence**:
| Role | Model | Outcome |
|---|---|---|
| Controller | `claude-sonnet-5` (session) | Ran Gate 0-3 preflight/quorum across 3 rounds, measured premises live (prod Firestore send/forward, `derive-planner-lane.sh` source, `docusaurus-deploy.yml`/`ci.yml` path filters, `docs-sync` script existence), applied the mid-iteration directive, wrote a separately-justified skill fix. |
| Designer | `claude:claude-sonnet-5` (Agent-tool pin, per this mission's seed — not a fallback) — spawned TWICE | Revision 1: scoped to round-1 objections + controller's measurements. Revision 2: folded in Mark's `29a467cac` decision + round-2 measured fixes. Confirmed post-hoc both times: clauses 1-6, CURRENT GOAL body, STATUS ARMED-BUT-SILENT language untouched outside the intended sections. |
| Planner / Executor / Sprint-evaluator | N/A | Not spawned — no sprint routed. The charter's own gate ("iteration 0 ratifies it via the quorum before any sprint routes") blocks all sprint work until ratification, and ratification did not land this iteration. |
| generator≠judge | quorum reviewers `gpt5-6-sol` / `gemini-3-1-pro` / `oc-glm-5-2` | All three are independent of the sonnet controller AND the sonnet designer sub-agent — this iteration's actual deliverable (the charter text) got independent review structurally, without needing a separate sprint-evaluator role that had nothing to evaluate. |

**Mid-iteration directive**: while round 2 was in progress, Mark (attended) reviewed the
round-1/2 blast-radius objection directly and committed `29a467cac`, widening
`MISSION_PLANNER_ALLOWLIST` from `tools/launchd/*` to `tools/*` — answering what became `D-DOCS-1`
before it was even formally asked. Actioned in-iteration per this skill's mid-iteration-directive
rule: spawned a second pinned-sonnet designer revision folding the resolution in, plus the two
round-2 measured fixes, and re-ran the quorum a third time.

**Outcome**: **PARKED — needs-human-review** on `docs-0`, after THREE quorum rounds (round 1
$0.057, round 2 $0.062, round 3 $0.078) and TWO designer revisions (the second justified only by
Mark's live mid-iteration answer, not by round 2 merely re-blocking). Round 3 blocked on three
entirely new objections — no reviewer has passed in any round, and objections keep landing on a
fresh surface each time rather than converging, so continuing to revise unattended risks chasing a
moving bar rather than closing a bounded gap. Stopped per Gate 2's round-tracking rule and Standing
Rule 2 (never force a guardrail). The charter is NOT ratified. No other queue item advanced.

**Key find**: 6 of the 9 objections across three rounds were refuted or resolved by direct
measurement, none needing a human call — `ailang messages send`/`forward --to` verified working
end-to-end against prod Firestore; the derive-planner-lane.sh "silent fallback" objection refuted
(deliberately labeled fail-closed-to-opus design); the CI "no push paths filter" claim confirmed
true for `ci.yml` while `docusaurus-deploy.yml`'s real filter is broader than first stated (also
`internal/**`, `cmd/**`, `go.mod`, `go.sum`, `web/**`); and round 3's `audit_design_docs.sh`/
`check_versions.sh` existence objection refuted by an `ls` I'd already run earlier this same
iteration. ONE objection (blast-radius scope for docs-1) was resolved not by measurement but by
Mark's live mid-iteration decision. THREE round-3 objections remain unresolved: gpt5-6-sol wants a
full conflict-surface/protocol spec for docs-1 written into the charter itself (arguably scope
creep — that belongs in docs-1's own sprint plan, per the charter's own "most items need no design
doc" guardrail); oc-glm-5-2's objection that an admittedly-unratified doc reads as operationally
binding is a structural property of any doc mid-quorum-review, not something a text edit fixes.

**Cost**: metered $0.197 of $1 ceiling (three quorum rounds) · quota: sonnet (controller + two
designer sub-agent runs). Plus a separately-justified skill fix (see FLAGGED/retro), zero additional
metered cost.

**Next**: Mark's read on the two open round-3 objections decides whether a 4th (bounded) revision
is worth spawning, or whether the quorum bar itself needs recalibrating for a mission-charter
document vs. a feature design doc (this doc has now gone through more rounds with less convergence
than typical feature docs this fleet quoriums — worth watching for a 2nd instance before treating
it as a pattern). Once resolved, docs-1 (already unblocked on blast radius) is likely the next
pick.

**Ruled out**: the two round-1 premise objections about `ailang messages send`/`forward` not
working for an unregistered inbox — refuted by live measurement against prod Firestore. The
`derive-planner-lane.sh` opus-fallback being an unfixed "silent fallback" bug — refuted; it is a
deliberately labeled fail-closed-to-safety default (script lines 57-64). Round 3's claim that
`docs-sync`'s `audit_design_docs.sh`/`check_versions.sh` are unverified — refuted, both files exist
(`ls .claude/skills/docs-sync/scripts/`).

**DECISIONS FOR MARK**:
- **D-DOCS-1 — RESOLVED** (attended, commit `29a467cac`): blast radius widened to include `tools/`
  (non-`internal/`, non-`cmd/`). Folded into the charter.
- **D-DOCS-2 (new)** — round 3's gpt5-6-sol objection wants docs-1's full implementation protocol
  (source enumeration, dedup keys, retry/timeout semantics, scheduling ownership) specified in the
  CHARTER before ratification. Is that right for a mission charter, or does that belong in docs-1's
  own sprint plan once picked (per the charter's existing "most items need no design doc" guardrail)?
  One word: (a) charter must specify it now / (b) defer to docs-1's own sprint planning.
- **D-DOCS-3 (new)** — round 3's oc-glm-5-2 objection is that an unratified doc cannot simultaneously
  read as operationally binding (live queue, live launchd job). Is this charter-specific, or an
  inherent tension in quorum-reviewing any live mission doc that should be a standing skill note
  instead? One word: (a) add an "UNRATIFIED — proposed, not binding" banner to this charter / (b)
  it's inherent, not this doc's problem, dismiss.
- Given two open objections after 3 rounds and no reviewer pass yet: would you rather review the
  current diff yourself, or should the loop keep revising unattended? One word: (a) I'll review it
  myself / (b) keep going unattended.

**FLAGGED**: one skill fix applied (Gate 5, ≥2-friction bar met by the established "bare
`~/.ailang/state/` literal" class — designer-rotation, `mission-gh-issue`, `mission-dashboard.md`,
now the kill-switch/queue/log paths, 5th instance): the shared skill's Gate-0 "Current State"
preflight literals (`mission-control.disabled`, `v1-mission.md`, `v1-mission-log.md`) are V1-only
and silently wrong for every other mission — fixed in `0e341cc57`, main-checkout worktree (the
symlinked skill resolves there, not to this pinned worktree — confirmed via `readlink`/`stat -f %i`
before editing). No role fell back, no routing-policy violation, no billing anomaly.

#### Design-quorum review — `design_docs/docs-mission.md` (2026-08-28T06:49:08Z)

- **Synthesis: BLOCKED** (total $0.0783, 23594 in / 943 out tok)
- `gpt5-6-sol` → **reject** ($0.0489, 7677/351 tok) — The mandatory docs-1 trigger has no conflict-surface analysis: the document proposes a new polling script without inventorying existing GitHub ingestion, public-feedback dispatch, message routing, scheduling, deduplication, or acknowledgement machinery and without justifying why none can be reused. Verification of `send` and `forward` does not establish that a parallel router is the correct extension point.
- `gemini-3-1-pro` → **reject** ($0.0188, 8275/190 tok) — Clause 1 asserts the existence and utility of 'audit_design_docs.sh' and 'check_versions.sh' within the 'docs-sync' skill without providing any verification evidence. This violates the premise verification hard gate, as these critical codebase instruments are asserted as working tools but entirely lack execution traces or proof of existence, contrasting sharply with the rigorously verified CLI commands in Clause 7.
- `oc-glm-5-2` → **reject** ($0.0106, 7642/402 tok) — The doc's own STATUS section records that iteration 0 quorum was BLOCKED twice and the charter is 'still not ratified,' yet the Queue and CURRENT GOAL already treat clauses 1-7 as the binding bar. The doc is simultaneously an unratified draft and an operational charter with a live launchd job, 5 clauses already tagged to queue items, and a cost ladder claimed as 'verified before being written down.' A self-admittedly unratified document cannot also be the source of truth for its own ratification — the doc pre-commits to its own bar before the quorum that exists to check it has run.
- controller (in-session, not an API call) → **pass** — Round-3 verdict. All three round-2 objections now resolved: (1) gpt5-6-sol's docs-1 blast-radius objection -- Mark himself, attended, authorized widening MISSION_PLANNER_ALLOWLIST from tools/launchd/* to tools/* mid-iteration (commit 29a467cac, verified both directions before asking per that commit's own message), and this revision folds that into the Guardrails/docs-1 text with a citation; (2) gemini-3-1-pro's silent-fallback objection -- refuted by measurement in round 2 (derive-planner-lane.sh Step 0 is a deliberately labeled fail-closed-to-opus default, every exit path emits a distinct reason token specifically to prevent silent misreads) and now the doc's own wording ('silently runs opus' -> 'deliberately runs opus (labeled fail-closed default)') matches that measurement instead of contradicting it; (3) oc-glm-5-2's CI-citation-completeness objection -- the Repo Profile now cites the actual on.push.paths list from docusaurus-deploy.yml (confirmed via cat, includes internal/**, cmd/**, go.mod, go.sum, web/**, the workflow file itself, beyond the four paths previously stated) and confirms ci.yml has no push paths filter. I independently verified all of these commands/files myself, not merely accepted the designer's claims. Scope check: git diff shows only Repo Profile, Guardrails, Routing-policy Planner row, and Queue docs-1 touched; clauses 1-6, CURRENT GOAL, and STATUS are untouched. PASS from the controller.
- Blocking objections (return to author before planning):
  - gpt5-6-sol: The mandatory docs-1 trigger has no conflict-surface analysis: the document proposes a new polling script without inventorying existing GitHub ingestion, public-feedback dispatch, message routing, scheduling, deduplication, or acknowledgement machinery and without justifying why none can be reused. Verification of `send` and `forward` does not establish that a parallel router is the correct extension point.
  - gemini-3-1-pro: Clause 1 asserts the existence and utility of 'audit_design_docs.sh' and 'check_versions.sh' within the 'docs-sync' skill without providing any verification evidence. This violates the premise verification hard gate, as these critical codebase instruments are asserted as working tools but entirely lack execution traces or proof of existence, contrasting sharply with the rigorously verified CLI commands in Clause 7.
  - oc-glm-5-2: The doc's own STATUS section records that iteration 0 quorum was BLOCKED twice and the charter is 'still not ratified,' yet the Queue and CURRENT GOAL already treat clauses 1-7 as the binding bar. The doc is simultaneously an unratified draft and an operational charter with a live launchd job, 5 clauses already tagged to queue items, and a cost ladder claimed as 'verified before being written down.' A self-admittedly unratified document cannot also be the source of truth for its own ratification — the doc pre-commits to its own bar before the quorum that exists to check it has run.

## ITERATION 1 — 2026-08-28T07:26Z

**Pick**: `docs-2` (queue head, `[NEXT]`). No allowlisted directive and no genuine regression
outranked it — 11 unread inbox messages, all triaged as non-directive (V1's own reports/approvals,
a stale coordinator task `task-a0628a5f` unreachable from this local session, and `mcp-public`
package feedback for a different package); zero comments on bookkeeping issue `#953` since the
watermark.

**Routing evidence**:
| Role | Model | Outcome |
|---|---|---|
| Controller | `claude-sonnet-5` (session) | Gate 0-2 preflight; baselined all 5 docs-sync diagnostics on pristine `origin/dev` before routing (rule 3e); discovered and diagnosed the `check_examples.sh` absolute-path `MOD010` instrument defect first-party, before handing it to the planner as a verified fact. |
| Designer | not spawned | Guardrails: "most items here need no design doc at all" — `docs-2-brief.md` (a routing declaration, not a design doc) already existed from a prior session. No quorum needed. |
| Planner | `codex:gpt-5.6-luna` (probe rc=0; ephemeral detached worktree `.planner-wt-iter1-docs-2` off `origin/dev`) | Produced `sprint_docs-2.json` + `sprint-plan-docs-2.md`, 3 milestones, faithfully carrying forward the controller's verified findings and the blast-radius constraints (single-level `docs/*`, no `.claude/skills/docs-sync/**`). Both artifacts copied to the main checkout paths; worktree removed. |
| Executor | `codex:gpt-5.6-luna` (same lane; directive worktree `.wt-iter1-docs-2`, `sprint/iter1-docs-2` branch, no git-write operations per the cross-provider recipe) | Built the prescribed ldflags-stamped scratch binary, ran the corrected relative-path sweep (166/9/42, matching the controller's baseline exactly), wrote `docs/docs-sync-findings.md` (13 scored findings), asserted `git check-ignore -v` empty. Controller reviewed the uncommitted diff and finalized the commit (`b896d70e6`) crediting the executor. |
| Evaluator | `sonnet` (Agent-tool pin; own isolated worktree `.wt-iter1-docs-2-eval` from the executor's commit) | Independently re-derived every load-bearing claim from a fresh build and its own from-scratch 217-file sweep script (not reusing the executor's) — identical 166/9/42 split, identical 9 failing filenames, confirmed the Env-capability gap and the population-count mismatch by direct command. **PASS 92/100, zero blocking.** |

**Outcome**: **LANDED**. [PR #955](https://github.com/sunholo-data/ailang/pull/955) → squash
`a8f904aac`. All required + non-required checks green on the PR head; re-verified on the MERGE
COMMIT itself (Gate 3b's squash-produces-a-new-commit rule) — 20 checks, 19 green, one INHERITED
non-required SonarCloud red (identical conclusion on the immediate parent commit `8a993bb89`,
before this PR ever merged; a coverage/security-rating gate over "new code" unrelated to a
docs-only diff — flagged to V1, not actioned).

**Key find**: `.claude/skills/docs-sync/scripts/check_examples.sh` passes ABSOLUTE paths to
`ailang run`, so any example declaring `module examples/runnable/X` fails `MOD010` — the script's
own raw output (12 passed / 29 failed / 176 skipped) has been silently over-reporting broken
examples, possibly for as long as the script has existed, and nobody noticed because a
`docs-site`-profile checkout normally has no `bin/ailang` to trigger the failure mode at all. A
freshly built binary with relative paths gives the true split: 166 pass / 9 genuine fail / 42
no-module across 217 files, independently reproduced exactly by the evaluator. Two further genuine
findings: `batch_processing.ail`/`cli_args_demo.ail` need an `Env` capability the generic checker
never grants; `audit_design_docs.sh` and `derive_roadmap_versions.sh` report different design-doc
population totals (159/1030 vs 126/682) with no stated scope difference.

**Also found, load-bearing for the queue**: this mission's OWN blast-radius allowlist cannot fix
most of what it just found. `MISSION_PLANNER_ALLOWLIST`'s `docs/*` is single-level, excluding
`docs/docs/**` (where the stale-version bug and essentially all published content live);
`.claude/skills/docs-sync/**` isn't covered at all (where the checker's own bug lives). Filed as
`D-1`/`D-2` in a newly created Human Decision Ledger (none existed yet on this mission's charter —
created following V1's exact format and marker syntax, validated with
`scripts/mission_decisions.sh --check`, 2 rows).

**Cost**: metered $0.00 of $1 ceiling — both codex runs rode the subscription-lane rung
(`gpt-5.6-luna`), never falling to a flat-rate or metered rung. Quota buckets: sonnet (controller +
evaluator).

**Next**: `docs-5` (queue head, `[NEXT]`) — in-scope examples hygiene for the 9 genuine failures,
answerable without waiting on D-1/D-2. `docs-1` (clause 7 trigger) is next after that if docs-5
is picked first, or vice versa; both are unblocked.

**Ruled out**: nothing this iteration contradicted a prior finding. The `launchd drivers (bash
3.2)` CI red observed at Gate 1 was confirmed FLAKY (same test failed on 2 unrelated commits in
the preceding hour, interspersed with passes on other commits; re-passed on this iteration's own
PR) rather than attributed to anything in this diff.

**DECISIONS FOR MARK**:
- **D-1 (new)** — widen `.claude/skills/docs-sync/*` into the planner allowlist so the mission can
  fix its own sync tooling's absolute-path bug? (a) widen the allowlist (b) route to V1 as shared
  infra (c) leave it, require every sprint to work around it by hand.
- **D-2 (new)** — widen `MISSION_PLANNER_ALLOWLIST`'s `docs/*` entry to cover `docs/docs/**` and
  `docs/src/**`, where nearly all published content and the stale-version bug actually live? (a)
  widen to `docs/**` (b) widen per-item as needed (c) leave narrow, route nested-content fixes
  elsewhere.

**FLAGGED**: two pre-existing CI reds on `sunholo-data/ailang` observed and reported to V1 (the
repo owner for CI-outranks-queue purposes) via the `controlplane` inbox — the `launchd drivers`
flake (confirmed transient) and the inherited SonarCloud dev-branch quality-gate red (confirmed
pre-existing on the parent commit). Neither actioned further; outside this mission's domain.

**Retro**: no skill edit — the one friction this iteration (asserting `gh pr checks`' pending-count
is numeric before comparing it) was already covered by this skill's existing `case … [!0-9]*)`
prescription and worked as documented; not a gap.

## ITERATION 2 — 2026-08-31T07:58Z (recovering a died-mid-flight prior fire)

**Pick**: none picked fresh — Gate 2's died-mid-flight check found a complete, unlanded iteration
already sitting on the repo: an open PR (`sprint/iter2-docs-9`, #973), three orphaned worktrees
(`.wt-iter2-docs-9`, `.planner-wt-iter2-docs-9`, `.wt-iter2-docs-9-eval`), and zero trace of
"ITERATION 2" anywhere in the charter/log/archive (`grep -c "ITERATION 2"` = 0/0/0, known-present
control `grep -c "ITERATION 1\b"` = 1 in the log — instrument confirmed working). That prior fire
had run the full inner loop on `docs-9` (queue head at the time) to completion and died before
Gate 4/5. This iteration's deliverable was to **verify and land it**, per the skill's died-
mid-flight instruction, not redo the work.

**What the orphaned fire found (docs-9, re-verified first-party before landing, not taken on
trust)**: `docs-9` claimed `docs/docs/intro.mdx` was stale — IFC Labels tagged `v0.16.0` while the
active teaching prompt is `v0.16.6`. Live-repro found the claim false: the "Recent Additions"
section intentionally annotates each feature's historical **ship**-version, not a "current
version" claim — five bullets, five different version numbers (`v0.16.0`/`v0.14.0`/`v0.13.0`
(x2)/`v0.12.0` (x2)), confirmed by direct read. `diff <(grep -i "declassify|IFC|T<label>|taint"
prompts/v0.16.0.md) <(... v0.16.6.md)` shows only the title line differs across all six
intervening revisions — zero IFC/declassify content changed, so bumping the annotation would
misrepresent when the feature shipped. The actual defect was the instrument:
`check_versions.sh` Check 3 grepped the first `vX.Y.Z`-shaped string anywhere in `intro.mdx` with
no way to distinguish "current-version claim" from "historical ship-version annotation" —
permanently false-positiving. Fix: delete Check 3 (Checks 1/2, unambiguous "must match" semantics,
untouched), rule `docs-9` `[RULED OUT]` with the refutation evidence, fix a self-contradictory
evidence-summary line the evaluator caught, annotate `docs-sync-findings.md` DOCS-2-02 refuted.
Zero changes to `docs/docs/intro.mdx` itself — nothing there was actually stale.

**Routing evidence (from the orphaned fire's PR body, re-verified rather than transcribed)**:
| Role | Model | Outcome |
|---|---|---|
| Planner+Executor | `codex:gpt-5.6-luna` declared, fell back to session `sonnet` alias (Agent tool in that session could not express a `provider:model` pin) — **FLAGGED** by that fire's own PR body | Diff scope exactly 4 files, none under `internal/`/`cmd/` |
| Evaluator | `opus` (re-routed off the `sonnet` fallback to preserve generator≠judge, since the assumed OpenAI/Z-AI-disjoint chain didn't materialize) | **PASS 79/100, zero blocking** — independently re-derived the semantic claim via its own `prompts/v0.16.0.md` vs `v0.16.6.md` diff, confirmed diff scope, re-ran every acceptance check from scratch |

**This iteration's own verification before landing** (not inherited): re-ran the ship-version spot
check on `docs/docs/intro.mdx` directly (5 distinct version numbers, confirmed); re-ran the
`prompts/v0.16.0.md`↔`v0.16.6.md` diff directly (title-only); confirmed all three worktrees had
zero uncommitted state (`git status --porcelain` empty in all three — the fire finished cleanly,
it just never reached Gate 4/5); confirmed `#973` `mergeable=MERGEABLE`, `mergeStateStatus=CLEAN`,
21 checks, none non-green.

**Outcome**: **LANDED**. Squash-merged [PR #973](https://github.com/sunholo-data/ailang/pull/973)
→ `ad7542ba5`. Local `dev` fast-forwarded to match. CI (`CI`, `Deploy Documentation to GitHub
Pages`) polled on the merge commit itself (Gate 3b's squash-produces-a-new-commit rule) — see
STATUS stamp for final read; both were still `in_progress` as this entry was written and were
polled to completion before the report was sent (Standing rule 7 — turn kept alive with chained
bounded polls, not ended mid-wait). Orphaned worktrees removed; remote branch `sprint/iter2-docs-9`
was never pushed (local-only), nothing to delete there.

**Gate 0 weekly external-issue sweep (first iteration after the Monday 07:00 CEST rotation
boundary — issue #953 created 2026-08-28T05:58:39Z / 07:58 CEST, before today's 07:00 CEST
boundary)**: enumerated `gh issue list --repo sunholo-data/ailang --state open --limit 100` → 92
issues, asserted against `gh issue list ... | jq length` → 92 (match; first attempt used
`--limit 50` and silently truncated to 50 — caught by the length assertion before it shipped, per
rule 3a(i)). Grepped each `#N\b` across this mission's own corpus (charter, log, status archive,
dashboard) with a known-positive control (`#953` → 4/2/0/0=6) and a known-absent control
(`#88214` → 0/0/0/0) both firing correctly. **First pass showed 92/92 orphaned — that number was
wrong and self-caught before being recorded**: the per-file loop indexed a zsh array as
`${FILES[0]}`..`${FILES[3]}`, and zsh arrays are 1-indexed, so `${FILES[0]}` is empty and every
`grep` ran with no file argument, reading a closed/empty stdin and silently returning empty rather
than "0". Re-run with `${FILES[1]}`..`${FILES[4]}`: **89 of 92 orphaned**, control counts
unchanged. Full per-issue table (charter/log/archive/dashboard/total) generated at
`/tmp/docs_sweep_table.md` this iteration; not inlined here (92 rows) but reproducible verbatim
from the command above — available on request.
Of the 89 orphans, 87 are plainly outside this mission's domain (motoko_agent, Z3/SMT encoder,
formatter, ailang-parse, email-parse, world, nightly-eval, CLI/effect-system internals — V1's or a
sibling mission's territory). **Two are not**: [#670](https://github.com/sunholo-data/ailang/issues/670)
and [#654](https://github.com/sunholo-data/ailang/issues/654), both about `make verify-examples` —
this mission's own Repo Profile names it as one of two verify-profile gates. Re-confirmed live at
this iteration's HEAD (not taken on the issue's word): `grep -c "checked == 0"
scripts/validate_manifest.go` → 0 (no anti-vacuity floor, #654's claim), `grep -c "expected.stdout"
scripts/verify_examples.go` → 0 (never compared, #670's claim). Both still genuinely present.
Batched into ONE new queue row, `docs-10`, positioned after `docs-6` (same class of work — fixing
the sweep's own instruments, not docs content) — per Gate 0 rule 5, a sweep finding never outranks
the existing pick by itself, and this one didn't: `docs-9`'s recovery was already this iteration's
item before the sweep ran.

**Cost**: metered $0.00 of $1 ceiling (the orphaned fire's own subscription-lane codex/sonnet/opus
runs; this iteration made no further model-role spawns — Gate 2's reality-check, the sweep, and
Gate 4/5 bookkeeping are all controller-session work). Quota buckets: sonnet (controller, this
session).

**Next**: `docs-5` or `docs-1` (both `[NEXT]`, both unblocked, as iteration 1 already noted) —
`docs-10` (new) is also unblocked and cheap; `docs-6` should probably precede `docs-5` since it
fixes the instrument `docs-5`'s own acceptance criteria will need to trust.

**Ruled out**: nothing new contradicted; docs-9's rule-out (see above) was the orphaned fire's
finding, re-confirmed rather than re-derived from zero.

**DECISIONS FOR MARK**: none — D-1 and D-2 are already `RESOLVED` in the ledger; no new ask this
iteration.

**FLAGGED**: (1) the orphaned fire's own PR body flags that its session's Agent tool could not
express `codex:gpt-5.6-luna` as a pin and fell back to `sonnet`/`opus` aliases — carried forward
here rather than re-litigated, since re-probing a capability limit is itself a rule-3a(i) vacuous
check (Gate 3 roles table). (2) `launchd drivers (bash 3.2)` is still red on `origin/dev`'s
immediate ancestry (confirmed failing on 2 of the last 2 non-in-progress commits checked) —
already flagged to V1 (repo owner) by iteration 1; not re-flagged, just noted so this entry doesn't
read as newly-discovered.

**Retro**: process note, not a skill edit (single-instance, below the ≥2-friction bar) — this
iteration's own sweep script hit the shared skill's own documented zsh 1-indexed-array trap
(rule 3a's "AND THE ARRAY THIS RULE JUST PRESCRIBED IS 1-INDEXED IN ZSH" clause) on its first
draft and self-corrected via the length-assertion discipline the same rule prescribes. Filed here
as a confirmation the existing rule works as written, not as a gap.

## ITERATION 3 — died mid-flight, credited retroactively by iteration 4 (2026-09-02)

**Pick**: this iteration's own record was never written — Gate 2's died-mid-flight check (iteration
4) found three merged PRs bearing this mission's routing conventions and a fourth, unmerged, sitting
in an orphaned worktree, with zero trace of "ITERATION 3" anywhere in the charter/log
(`grep -c "ITERATION 3\b"` = 0/0, known-present control `grep -c "ITERATION 2\b"` = 1/1 in log/none
elsewhere — instrument confirmed working). The fire worked the queue in order (`docs-5` → `docs-6` →
`docs-10`, all `[NEXT]` per iteration 2's own **Next** field) then began `docs-1`, producing a
complete brief + sprint plan before dying — recovered separately as iteration 4's own pick (see
below). This entry is retroactive bookkeeping only; none of the verification below was re-derived
from zero by iteration 4 beyond what's noted, since the PRs' own routing/test-plan sections and a
fresh `make verify-examples`/`check_examples.sh` run (iteration 4, see its own entry) already
constitute independent confirmation.

**Outcome**: **LANDED** (3 items).
- `docs-5` (examples hygiene): [PR #997](https://github.com/sunholo-data/ailang/pull/997) →
  merged 2026-09-01T03:21Z. Added `main` entrypoints to 7 fixtures; fixed a manifest module-drift
  on 5 of them (evaluator-caught). `batch_processing.ail`/`cli_args_demo.ail` (the other 2 of the
  original 9 findings) already had entrypoints on `dev` — resolved independently, out of scope.
  Evaluator round 1 blocking (manifest drift), round 2 PASS.
- `docs-6` (fix `check_examples.sh`'s absolute-path bug + DOCS-2-04 scope comments):
  [PR #1004](https://github.com/sunholo-data/ailang/pull/1004) → merged 2026-09-01T10:19Z.
  Routing: planner+executor `codex:gpt-5.6-luna`, evaluator `sonnet` — **PASS 98/100**, zero
  blocking, mutation-checked (reverting the fix reproduces the pre-fix 12/29/176 counts).
- `docs-10` (verify-examples vacuous on two axes, #670/#654):
  [PR #1010](https://github.com/sunholo-data/ailang/pull/1010) → merged 2026-09-01T21:44Z.
  Routing: designer `claude-sonnet-5`, planner+executor fell back to `opus` (Agent tool in that
  session could not express the `codex:gpt-5.6-luna` pin — same limitation iteration 2 recorded,
  FLAGGED again there, not re-litigated here), evaluator `sonnet` — **PASS 97/100**, zero blocking,
  11 acceptance criteria plus 5 adversarial mutations independently re-run from scratch.

**Iteration 4's own re-verification before crediting these landed** (not inherited): fresh
`make verify-examples` run — 211 passed, 0 failed, 6 skipped, manifest "193 modules checked, 0
drift, 1 missing-on-disk", "✅ verify-examples: all examples pass and manifest is in sync"; fresh
`check_examples.sh` run — 173 passed, 2 failed (`batch_processing.ail`, `cli_args_demo.ail` — the
known, explicitly out-of-scope Env-capability gap), 42 skipped, matching PR #1004's own claimed
counts exactly. Both are real, non-vacuous greens under `docs-10`'s anti-vacuity floor, not
inherited from the pre-fix instrument.

**Cost**: metered $0.00 (codex/sonnet/opus are all subscription-lane per this mission's routing
table; no `pi:`/managed_agents/quorum-reviewer calls in any of the three PRs' routing sections).
Quota buckets: codex (planner+executor, docs-6), opus (planner+executor fallback, docs-10),
claude-sonnet-5 (designer, docs-10), sonnet (evaluator, all three).

**Next**: `docs-1` (iteration 4's own pick — see below).

**Ruled out**: nothing new; no ruled-out claims in any of the three PR bodies.

**DECISIONS FOR MARK**: none — nothing in these three PRs raised a new ask.

**FLAGGED**: `docs-10`'s planner+executor could not express `codex:gpt-5.6-luna` via the Agent tool
in that session and fell back to `opus`, per PR #1010's own Routing section — same capability
limitation iteration 2 already flagged; carried forward rather than re-litigated (re-probing a
capability limit is a rule-3a(i) vacuous check).

**Retro**: no skill edit — this entry's only job is closing the "orphaned iteration" gap Gate 2's
died-mid-flight rule warns about (skipping a number silently). Two consecutive died-mid-flight
fires (this one, and the docs-1 planner run recovered as PR #1016) in the same short window is a
pattern worth naming for whoever reviews mission health, not a single-instance skill gap.

## ITERATION 4 — 2026-09-02T04:08Z

**Pick**: Gate 2's died-mid-flight check found an open, `MERGEABLE` PR
([#1016](https://github.com/sunholo-data/ailang/pull/1016)) recovering iteration 3's complete
`docs-1` brief + sprint plan from an orphaned worktree (`.wt-docs-iter3-docs1` under the pin root,
detached at the same commit as the PR head, clean, no uncommitted state), plus a second, unrelated
stale worktree (`.planner-wt-iter3-docs-5`, a superseded routing-declaration commit predating
`docs-5`'s already-landed fix — pruned, nothing to recover). Also found `docs-5`/`docs-6`/`docs-10`
all landed on `origin/dev` while the charter still tagged them `[NEXT]` — see iteration 3's own
retroactive entry above. Picked `docs-1` (this iteration's real item) after re-verifying the other
three genuinely landed rather than redoing them.

**Gate 1 CI health**: `CI` and `Deploy Documentation to GitHub Pages` both `success` on
`origin/dev`'s HEAD; SHA-addressed `commits/<sha>/check-runs` showed **16** checks, one NOT-GREEN
(`SonarCloud Code Analysis: failure`) — confirmed inherited from the parent commit too (not from
any recent merge), and out of this mission's domain per Gate 1's repo-ownership scoping
(`sunholo-data/ailang` is V1's territory for CI reds); not actioned, noted for the report.

**PR #1016's own CI**: one job red (`launchd drivers (bash 3.2)`) on a PR touching only two
markdown planning files — a known-flaky wall-clock-deadline test in the launchd driver suite (the
failing assertion was `descendant discovery refuses on the real wall-clock deadline`), confirmed
unrelated to the diff (same check reads `success` on both the PR's base and `origin/dev`'s current
tip). Re-ran the failed job (rule 3d — negative control before crediting a red to anything): green
on re-run, confirming the flake diagnosis rather than assuming it. Merged
[PR #1016](https://github.com/sunholo-data/ailang/pull/1016) → `4f94edf70`.

**Execution**: routed `docs-1` to `codex:gpt-5.6-luna` (executor, per the mission's routing table)
in an isolated worktree, using the recovered brief + sprint plan verbatim (no re-planning). Codex
probe rc=0 before the real run.

- **Round 1**: codex built `tools/messaging/docs_inbox_router.sh` (238 lines). Controller
  independently re-verified every acceptance command before committing (not trusting the
  executor's self-report): `bash -n` rc=0, `shellcheck` rc=0, `--selftest` rc=0 (reproduced
  checked=5/forwarded=3 then checked=3/forwarded=0 duplicate suppression), forced total-store
  failure rc=1 with empty stdout and a clear stderr message, no process-wide
  `export AILANG_MESSAGES*`, forward shape (`--to docs-mission --reason "..." <id>`) matches
  clause 7's verification log verbatim, diff scope exactly one new file under `tools/`. Committed,
  opened [PR #1018](https://github.com/sunholo-data/ailang/pull/1018).
- **Evaluator round 1** (sonnet, own isolated worktree, generator≠judge held — codex executor vs
  sonnet evaluator, distinct providers): **FAIL 58/100**. Live-reproduced a BLOCKING bug:
  `poll_messages`/`poll_github` fed a `jq` command substitution into a `while read` heredoc
  (`<<EOF\n$(jq ...)\nEOF`); when the store legitimately returns zero items — the ordinary,
  most-common state for a low-traffic poller — the heredoc still contains one blank line, so the
  loop executes once with `message_id=""` and hits the "item without an ID" guard, aborting the
  ENTIRE run on a VALID empty read. Non-blocking: no `pipefail` on `is_doc_related`'s
  `jq | grep` pipe (lower risk, input pre-validated upstream); `record_key` is O(n²) per append
  (not a concern at this router's traffic).
- **Round 2**: routed the evaluator's exact finding back to `codex:gpt-5.6-luna` (same worktree,
  continuing work, not a re-plan). Fix: capture `jq` output into a variable, gate loop entry on
  non-emptiness (`done <<<"$rows"` behind `[ -n "$rows" ]`), applied to both poll functions; added
  a `--selftest` empty-poll case. Controller independently reproduced BOTH the original failure (on
  the round-1 commit, with hand-built `[]`-returning fixtures distinct from the script's own
  `--selftest` fixtures) and the fix (on the round-2 commit, same fixtures) — not taken on the
  executor's or the evaluator's word alone. Re-ran the forced-failure regression check (still rc=1,
  clean) and the export/scope checks (still clean). Committed, pushed.
- **Evaluator round 2** (sonnet, same isolated worktree, updated to the new commit): **PASS
  90/100**, zero blocking. Independently rebuilt its own `[]`-returning fixtures (not the script's),
  confirmed the round-1 bug is genuinely fixed, re-confirmed the failure-regression check, and
  scanned the round-2 diff itself for new defects (jq-failure propagation via `|| die`, and whether
  `[ -n "$rows" ]` could misfire on an all-empty-field-but-real item — tested and confirmed no).
  Non-blocking: no CHANGELOG entry (correctly scoped low-severity — internal ops tool, not
  public-facing).

**Gate 3b**: merged [PR #1018](https://github.com/sunholo-data/ailang/pull/1018) (squash) →
`e65e96b15`. Polled the merge commit itself (squash produces a new SHA) with bounded chained waits
until all 16 checks settled: 15 green, the same pre-existing `SonarCloud Code Analysis` red as Gate
1 found (confirmed inherited, V1's domain, not actioned). **LANDED.**

**Cost**: metered $0.00 (codex is subscription-lane; both evaluator rounds were sonnet Agent-tool
sub-agents, Anthropic quota, not metered $). Quota buckets: codex (executor, 2 rounds), sonnet
(evaluator, 2 rounds + controller session).

**Next**: queue is now empty of `[NEXT]` items — remaining rows are `[PARKED]`
(`docs-3`, `docs-4`, `docs-8`) or `[LANDED]`/`[RULED OUT]`. `docs-8` (126 overdue planned design
docs) is the only PARKED item explicitly sequenced to become available now that `docs-6`/`docs-7`
are resolved ("Sequence after docs-6/docs-7 land or are ruled out") — flagging as the natural next
pick for a future iteration, not picked here since Standing rule 1 caps this iteration at one
backlog item (bookkeeping-only picks aside) and `docs-1` was already substantial.

**Ruled out**: nothing new.

**DECISIONS FOR MARK**: none — no new ask this iteration.

**FLAGGED**: none new. `SonarCloud Code Analysis` remains red on `origin/dev` (inherited, V1's
domain) — noted, not re-flagged as newly-discovered (already visible in this mission's iteration 1
FLAGGED row).

**Retro**: no skill edit this iteration (no ≥2-friction gap found in the shared skill itself — the
died-mid-flight, negative-control, and generator≠judge disciplines all worked exactly as
documented). Process observation for this mission's own charter, not the shared skill: two
consecutive died-mid-flight fires in one short window (iteration 3, and the `docs-1` planner run
recovered as PR #1016) is a pattern worth Mark's attention if it recurs a third time — not yet at
the evidence bar for a charter change.

## ITERATION 5 — 2026-09-02T17:00Z

**Pick**: `docs-8` (126 overdue planned design docs, aggregate) — the natural next item per
iteration 4's own note, the only `[PARKED]` row explicitly unblocked once docs-6/docs-7 resolved.
Also credited a second orphaned fire for `docs-3` (bookkeeping, not this iteration's pick).

**Gate 2 — orphaned fire credited first.** `gh pr list --author sunholo-voight-kampff --state
open` returned PR #1031 (`docs/iter5-docs3-provenance-wiring`, `MERGEABLE`) alongside worktrees
`.wt-docs-iter5-docs3`/`-eval` — zero "ITERATION 5" trace anywhere in charter/log/archive at pick
time (0/0/0, known-present control `ITERATION 4` = 2/1). A prior fire ran docs-3's full inner loop
(codex executor, sonnet evaluator independent worktree, PASS 85/100 zero blocking) and died before
Gate 4/5. Re-verified rather than trusted: `gh pr diff --name-only` confirms exactly the 4 claimed
files (`BenchmarkDashboard`, `BenchmarkExplorer`, `BenchmarkStandaloneGallery`,
`EloLeaderboard`/`index.jsx`); `mergeStateStatus: BLOCKED` on required checks `test`/`build`, and
this is base-inherited (SHA-addressed check-runs on `origin/dev` HEAD `50dd1a0aa` **and** its
parent `251835e14` show the identical 6-check red: `Build windows/macos/ubuntu-latest`,
`launchd drivers (bash 3.2)`, `test`) — a known, tracked, V1-owned issue (V1's own log:
`grep -c "macos-latest"` = 10, `grep -c "launchd drivers (bash 3.2)"` = 22; motoko independently
flagged the Windows half to V1 within the hour). `git diff --stat <PR base>..origin/dev -- <the 4
files>` is empty, so a rebase would not change the outcome (the red is on origin/dev's own tip,
not a stale base) — this is genuinely blocked on infrastructure outside this mission's domain, not
a landing this iteration can force. Queue row updated to `[IN-SPRINT]`, PR left open as a resume
point.

**Gate 2 reality-check on the pick itself.** `docs-8`'s own charter text claimed "126 overdue
planned design docs." Re-ran `.claude/skills/docs-sync/scripts/derive_roadmap_versions.sh` at
this iteration's HEAD (v0.34.0): **54** docs currently target a version below v0.34.0. The 126
figure had drifted since docs-2/docs-6 (which fixed a *different* population-count mismatch
between this script and `audit_design_docs.sh`, DOCS-2-04) — the count itself kept moving as
`design_docs/planned/` was edited by other work. Rule 3b(ix) applies: a count is only true in the
scope it was taken in; 126 was never re-verified before being carried forward as this item's
headline number.

**Execution — controller-run triage, explicitly NOT routed through the sprint pipeline.** The
charter's own docs-8 text says moving a doc to `implemented/` is CONTROLLER Gate-4 bookkeeping,
and `design_docs/` sits outside `MISSION_PLANNER_ALLOWLIST` (confirmed:
`echo $MISSION_PLANNER_ALLOWLIST` lists `tools/*|.claude/skills/mission-control/SKILL.md|
.claude/skills/design-doc-creator/*|docs/*|examples/*|README.md|CHANGELOG.md|
.claude/skills/docs-sync/scripts/*` — no `design_docs/*` entry), so a codex-executor lane would
fail-closed to opus regardless. Delegated the 54-doc cross-reference to **6 parallel
general-purpose/sonnet Agent-tool sub-agents**, 9 docs each, each required to grep for
implementation evidence (function/type/CLI names, CHANGELOG entries, `implemented/` cross-refs)
and pair every negative finding with a known-positive control in the same file (rule 3a).

**Classification result (all 54)**:

| Classification | Count | Docs |
|---|---|---|
| IMPLEMENTED (candidate move) | 20 | m-verify-stdlib-stale-path, m-coordinator-inbox-wildcards, m-mission-adaptive-multiprovider-routing, m-anthropic-cache-hit-rate, m-eval-fmt-weakmodel-ab-M5-hardset-prereg, m-eval-fmt-weakmodel-ab-M5-hardset-results, m-eval-token-headroom, m-parser-reserved-keyword-diagnostics, HANDOVER-mission-loop-unified-telemetry, m-mission-loop-unified-telemetry, m-stdlib-freeze-gate-path-mismatch, m-ollama-cloud-provider, m-property-generator-coverage, m-property-seed-determinism, m-recorded-stream-api-preread, m-recorded-stream-api, m-eval-rig-reliability, m-eval-slim-prompt-self-discovery*, m-eval-stream-health-retry*, m-pkg-feedback-loop |
| RULED-OUT (candidate) | 2 | m-parser-block-let-separator, m-eval-fmt-weakmodel-ab-M6-motoko-ext* |
| NEEDS-DEEPER-INVESTIGATION | 1 | m-fmt-deterministic-feedback (extension lives in an unvendored separate package repo) |
| STILL-PLANNED | 31 | m-agent-step-cancellation, m-ailang-semantic-context, m-anthropic-sandbox, m-apple-container-local-eval-sandbox, m-cascade-observability, m-concurrency-leverage, m-contracts-as-code-vertical, m-coord-thinking-levels, m-dx27-docs-search-github-fallback, m-eval-openrouter-baseline-rotation, m-eval-results-folder-structure, m-motoko-ext-per-task, m-pure-prng, m-serve-api-live-tool-registry, m-stdlib-html-streaming, m-arch-boundaries-eval-exclusion-tighten, m-diag-primitive-field-suggestions, m-parmap-effectful, m-ai-structured-step, m-comments-for-ai-authors, m-decision-entropy-monitor, m-eval-experiment-registry, m-eval-kimi-k3-agentic, m-managed-agents-model-eval, m-mcp-2026-07-28-adoption, m-mem-budget-runtime, m-eval-standard-mode-input-files-gap, m-coverage-cross-package-attribution, m-named-test-body-check-semantics, m-net-effect-proxy-boundary (M1 of 4 landed), m-typeclass-int-fractional-dictionary-gap |

(* = reclassified by the independent verifier below, see next section — these 3 did NOT get the
listed disposition.)

**Independent re-verification, before any file moved (generator≠judge).** The 6 classifier
sub-agents are the "generator"; per this run's explicit mandate that no work lands on the
controller's own verdict, a 7th, independent, adversarially-framed sonnet sub-agent re-ran every
cited command itself against the 22 highest-stakes claims (the 20 IMPLEMENTED + 2 RULED-OUT —
the ones that trigger an irreversible-ish file move or a rule-out stamp; the 31 STILL-PLANNED and
1 NEEDS-DEEPER-INVESTIGATION were out of scope, since nothing was about to act on them). It did
not see or trust the generator's transcriptions — every grep was re-run from the citation, every
control re-verified.

**Result: 19 of 22 CONFIRMED, 3 of 22 (14%) overturned — 2 outright reversals, 1 downgrade:**
- **`m-eval-slim-prompt-self-discovery` — REFUTED (IMPLEMENTED → RULED-OUT).** The generator's
  evidence (MCP `prompt_get`/`stdlib_search` plumbing exists; one runbook note recording a live
  demo) was real but generic — it was not evidence that *this specific* experiment's proposal
  shipped. The doc's own "done" criteria (a tagged `v0.10.0-slim` prompt, a committed A/B rotation
  report) were never met. The slim prompt files were built, A/B-tested (mixed-to-negative,
  82%→65% pass rate across iterations), and then **deleted** (`2de1ef963`, 2026-06-13); the live
  `local-ollama-eval` skill documents progressive-disclosure as "tried and deliberately
  abandoned," and states opencode does not use the AILANG MCP server as a tool source at all.
  Moving this to `implemented/` would have misfiled a killed experiment as a shipped feature.
  Header updated in place with the evidence (kept under `planned/`, matching this mission's
  docs-9/docs-7 convention for a negative result rather than deletion).
- **`m-eval-fmt-weakmodel-ab-M6-motoko-ext` — REFUTED (RULED-OUT → stays STILL-PLANNED).** The
  generator read a `models.yml` "RETIRED 2026-09-02" comment as a settled negative verdict. Read
  in full, the comment retires only the nightly *scheduling* because qwen3.6, its rig model arm,
  was decommissioned under the single-on-device-LLM rule — the extension itself was never
  measured (the doc's own status, "BUILT + INTEGRATED, firing not yet observed," is still
  accurate; reviving needs onboarding a replacement model, "new work, not a flag flip" per the
  doc). Marking this RULED-OUT would have misrepresented an open, unmeasured question as closed.
  Left untouched.
- **`m-eval-stream-health-retry` — WEAK (IMPLEMENTED → downgraded, stays STILL-PLANNED).** The
  generator's specific citation (`internal/ai/ollama/idlereader.go`) is tagged for a different,
  adjacently-numbered feature (`M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT`, #618) — a wrong-file error.
  The *real*, relevant wiring does exist (TTFT/idle-timeout split threaded into the opencode and
  pi executors, `ttft_timeout:` keys live per-model in `models.yml`) — Goals 1-2 of the doc. But
  Goals 3-4 (retry on stream death before consuming trial budget; label the failure
  `stream_death` instead of the generic `api_error` it still falls through to) are not built —
  `error_categorizer.go` has zero handling for TTFT/idle-timeout strings. This is the doc's
  actual point, un-shipped. Left untouched.

**Outcome: LANDED.** 18 confirmed-implemented docs moved: `git mv` from
`design_docs/planned/vX_Y/` to `design_docs/implemented/vX_Y/` (created `implemented/v0_33_2/`,
did not previously exist), 27 files total (18 primary docs + 9 sprint-plan companions, per Mark's
"plans travel with their doc" convention) — `m-eval-rig-reliability.md`'s companion is
`m-rig-reliability-sprint-plan.md` (name doesn't match the doc's own stem; caught by reading the
plan's own `Source design doc:` line rather than a naive filename match). Move targets, by
version: v0_29_0 (`m-verify-stdlib-stale-path`, `m-coordinator-inbox-wildcards`,
`m-eval-rig-reliability`+plan, `m-pkg-feedback-loop`+plan), v0_30_0
(`m-mission-adaptive-multiprovider-routing`), v0_31_0 (`m-anthropic-cache-hit-rate`+plan,
`m-eval-fmt-weakmodel-ab-M5-hardset-prereg`, `-results`, `m-eval-token-headroom`,
`m-property-generator-coverage`+2 lane-plans, `m-property-seed-determinism`+plan), v0_32_0
(`m-recorded-stream-api`+plan, `-preread`), v0_33_1 (`m-parser-reserved-keyword-diagnostics`),
v0_33_2 (`HANDOVER-mission-loop-unified-telemetry`, `m-mission-loop-unified-telemetry`+plan),
v0_33_4 (`m-stdlib-freeze-gate-path-mismatch`+plan), v0_34_0 (`m-ollama-cloud-provider`). 1 ruled
out via header update (`m-eval-slim-prompt-self-discovery.md`). The 31 genuinely-still-planned
docs are now this mission's accurate backlog — individually pickable from `design_docs/planned/`
by any future iteration; no new aggregate queue item created, since a corrected, per-doc-evidenced
list in this log entry is a better resource than a synthetic queue row would be.

**Note on system load**: `git status`/`git mv` ran unusually slowly (one compound command hit the
2-minute foreground cap; a follow-up `git status` backgrounded and took ~1 min) — load average
36-37 on a 16-CPU box at measurement time. Verified the moves had actually landed via direct `ls`
against the filesystem (independent of the slow `git status`) before concluding anything was
wrong; all 27 renames were confirmed correct once `git status` finally returned. Not a bug in the
commands, a symptom of concurrent fleet load — not actioned further.

**Cost**: metered $0.00 of $1 ceiling — all 7 sub-agents (6 classifiers + 1 independent verifier)
were Agent-tool sonnet spawns, Anthropic subscription quota, not metered $. Quota buckets: sonnet
(7 sub-agents + controller session).

**Next**: the 31 STILL-PLANNED docs listed above are individually pickable by any future
iteration — no new item needed. `docs-3` ([PR #1031](https://github.com/sunholo-data/ailang/pull/1031))
is verified and ready; merge it the moment `origin/dev`'s `test`/`build` red clears (V1's fix, not
ours to chase). `docs-4` (taxonomy pass) remains `[PARKED]`, sequenced after docs-3 lands.
`m-fmt-deterministic-feedback` (NEEDS-DEEPER-INVESTIGATION) would need a look at the separate
`sunholo/motoko_ext_fmt` package repo, not this one — flagged, not queued (out of this repo's
reach).

**Ruled out**: `m-eval-slim-prompt-self-discovery` (evidence in its own header now). Two
generator misclassifications refuted before landing (see above) — not "ruled out" in the queue
sense, but recorded here as the loop's independent-review discipline working exactly as designed.

**DECISIONS FOR MARK**: none — no new ask this iteration.

**FLAGGED**: `origin/dev`'s `test`/`Build macos/windows/ubuntu-latest`/`launchd drivers (bash
3.2)` red remains (inherited, V1's domain — same red Gate 1 has flagged in iterations 1/2/4, not
newly discovered; blocks PR #1031 from merging). `m-fmt-deterministic-feedback` needs
investigation in a different repo.

**Retro — no skill edit.** No ≥2-instance friction against the shared skill itself this
iteration — the died-mid-flight, generator≠judge, and rule-3a/3b disciplines all worked exactly
as documented, and the one real friction (slow `git status` under system load) was environmental,
not a skill defect. Process observation for this mission's own charter: the generator≠judge step
this iteration was not a formality — it reversed 2 of 20 IMPLEMENTED verdicts and would have
misfiled a killed experiment as shipped and a live, unmeasured extension as settled-negative had
it not run. Worth noting for any future large-batch classification sweep in this mission: the
independent-verification step is not optional overhead, it is where the real errors were caught.

---

## ITERATION 6 — 2026-09-03T10:54Z

**Gate 0.** Kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree at
`origin/dev` tip (`55891002f`), clean. Watermark check on bookkeeping issue `#979`: 0 directives
from `MarkEdmondson1234` since `2026-09-02T06:29:45Z` (7 comments total, all non-allowlisted).
Decision ledger valid, 2 rows, both `RESOLVED`, no OPEN rows — no unparking needed. 16 unread
canonical-inbox messages: `design-doc-creator` self-completion, `pkg-sunholo-ailang-parse`/
`pkg-sunholo-test-pkg`/`pkg-sunholo-external-backend` package task completions/failures,
`mission-v1`/`mission-world` cross-mission traffic, `ailang-parse-claude`↔`aitana-platform`
package feedback threads — none addressed to `mission-docs`, none actioned.

**Gate 1.** `git fetch origin`; pin worktree HEAD (`55891002f`, detached) == `origin/dev`, no
divergence to repair. SHA-addressed check-runs on `origin/dev` HEAD: 17 total, 6 NOT-GREEN —
`Build windows-latest` (cancelled), `Build macos-latest` ×2 (cancelled), `Build ubuntu-latest`
(failure), `launchd drivers (bash 3.2)` (failure), `test` (failure). Per-workflow read confirms:
`CI` completed/failure @ `55891002f`; `Deploy Documentation to GitHub Pages` in_progress (docs
mission's own watched deploy gate, unaffected so far). Failing job names (`test`, `launchd
drivers (bash 3.2)`) are Go-test-suite and launchd-driver territory — not docs-mission's domain,
and not caused by a docs-mission commit (the two most recent commits on `origin/dev`,
`fix(resident)` and `fix(docker)`, are resident/A2A and docker-pi-pin work). Per Gate 1's
repo-ownership scoping (V1 owns `sunholo-data/ailang`), this is V1's red to fix; sent a
cross-mission heads-up to `mission-v1`'s inbox (`inbox_1788431686434_383803af`) naming the exact
failing jobs and commits rather than actioning it here. Not re-flagged as new — same signature
Gate 1 has recorded in iterations 1/2/4/5.

**Gate 2 — pick.** Queue re-read: item 10 (`docs-3`, `[IN-SPRINT]`) re-checked —
`gh pr view 1031` still `MERGEABLE`/`BLOCKED` on the identical failing-check set named above, not
a stale-base problem (unchanged since iteration 5), so not actionable this iteration; left as the
resume point it already was. Item 11 (`docs-4`, was `[PARKED]`): both stated blockers re-verified
cleared — clauses 1-3 green (docs-2 covers 1+3 directly; docs-9/docs-6/docs-10/docs-8 close out
clause 1's remaining findings; docs-5 closes clause 2), and docs-7 (the allowlist-blocks-nested-
paths premise) was dissolved 2026-08-28. **Picked `docs-4`.**

**Gate 3 — designer.** No existing design doc for docs-4 (`grep -ril` across `design_docs/`
returns nothing but the charter/log/dashboard themselves). Spawned the rotation designer
(`claude:claude-fable-5`, no prior rotation-pointer file → starts at `claude`, Agent tool
`model="fable"`, run in background, ~1001s/67 tool calls/256.7k tokens) with an explicit brief:
given 62 files under `docs/docs/guides/` (measured, matching the charter's own count), decide
with evidence whether the taxonomy pass fits one sprint-sized doc or needs decomposition into
2-4 smaller docs per the charter's standing multi-week-item rule, and produce accordingly.

**Result: one sprint, not a decomposition** —
`design_docs/docs-4-brief.md` (committed `1c74a4971` on new branch `docs-4-brief`, off detached
HEAD `55891002f` = `origin/dev`; not pushed). The designer's own pairwise instrument (line-overlap
≥4 identical non-heading/>40-char lines, all 1,830 guide-pair combinations) found only 2 pairs
sharing ≥4 lines — literal page-level duplication is close to zero. The real redundancy is
section-level: the same `ailang messages`/`make services-start`/`ailang coordinator status`
command blocks recur across 4-7 files each, and six 2026-08-17 "audit pass" commits already did
the page-level merges the charter's "40+ guides accreted" language implied still needed doing.
What's left is nav + cruft: a tracked `.bak` file (since 2025-11-13, 404s live), 9 pages built and
served but absent from `sidebars.js` (orphaned — unreachable from the nav), a coordinator cluster
split across two top-level categories, ~5 pages in the wrong audience bucket. The brief enumerates
every one of the 62 files' disposition (Appendix A) and the target sidebar tree (Appendix B):
Phase A is mechanical (`git rm` the `.bak`, rewrite `sidebars.js` 63→70 guide ids); Phase B is 5
enumerated redundancy removals (2 page deletions with inbound-link retargeting, 3 section trims to
one-line pointers) — each justified against an explicit deletion criterion (every section already-
present/stale-false/repo-internal AND every inbound link retargeted in the same commit) rather
than asserted. 29-row Verification Log with commands; two self-caught instrument bugs recorded
rather than hidden (BSD `sed` rejects `\?` in basic regex — first orphan-check pass reported all
61 pages as orphans; zsh does not word-split an unquoted `$files` — first pairwise-duplication
pass errored). One genuine baseline finding: bare `make docs-build` is red at HEAD regardless of
this sprint (a tracked `packages-sidebar.json` points at gitignored `packages/sunholo/*` ids) —
the deploy workflow's actual form, `sync-registry.sh && make docs-build`, is green; the brief's
acceptance uses the workflow form and flags the `make/docs.mk` fix as a separate, out-of-allowlist
row (Deferred section).

**Quorum-at-pick** (mandatory — no prior artifact existed for this doc;
`.ailang/state/mission-quorum/` had none matching `docs-4`). Round 1
(`docs-4-brief-2026-09-03T10-54-51Z.json`, $0.1297): BLOCKED. `gpt5-6-sol` reject — V6 probed only
8 of the 9 claimed orphan URLs, omitting `evaluation/cost-and-speed-budgets`. `gemini-3-1-pro`
reject — no Verification Log row proved the B4/B5 section-cut headings (`Message System`,
`SessionStart Hook Behavior`, `Two-Tier Search Architecture`, `Embeddings Doctrine`) actually
exist or are adjacent. `oc-glm-5-2` absent (invalid), degraded to N-1, not silently passed. Both
present objections carried a concrete, verbatim `proposed_fix` and disputed no design direction —
per Gate 2's rule, the controller measured both directly (single commands, no design judgment,
so not re-routed through the Fable designer): `curl` on the 9th orphan URL → `200` (matches the
other 8); `grep -nE '^##+ '` on both files confirmed both heading pairs exactly as the phases
assert. Committed as new Verification Log row (corrected V6) and a new row V29 (`fbc289f6a`).

Round 2, the standard flow's mandatory single re-quorum
(`docs-4-brief-2026-09-03T10-57-26Z.json`, $0.1219): BLOCKED again, all three reviewers present
and reject. `gpt5-6-sol` — the doc's "URL-stable" scope claim overclaims against the two
intentional, clause-5-authorised deletions (`cross-project-messaging`, `development`), which do
404 their own URLs since no redirects plugin exists (V18). `gemini-3-1-pro` — B3's cut boundary
(`getting-started.mdx` § *For AI Agents: CLI Integration*) was only regex-matched (V16), never
proven to be a genuine H2 with exact text — the same rigor V29 had just given B4/B5, not yet
given to B3. `oc-glm-5-2` — V23b documents that the build baseline mutates 4 tracked files
(`design-docs.md`, `prompts/current.md`, `roadmap/index.md`, `packages-sidebar.json`) and that
they must be `git checkout --`-restored before committing, but that restore lived only in prose,
never as an encoded acceptance step — an executor following the numbered acceptance list literally
could commit the mutated files or misdiagnose acceptance criterion 7's failure. All three
objections: narrow, concrete (each carried its own verbatim `proposed_fix`), and none disputed the
design direction (clause 5's deletion authorization, the restructure, the five trims) — meeting
Gate 2's narrow-refinement carve-out criteria exactly. Applied the carve-out: reworded the
URL-stable scope line to explicitly exempt the two authorised deletions; added row V30 (B3's
boundary confirmed exact — line 180 `## <Icon name="bot" inline /> For AI Agents: CLI Integration`
through, not including, line 253 `## <Icon name="user" inline /> For Human Developers: Manual
Installation`, both genuine H2s); rewrote acceptance criteria 6-7 to encode the `git checkout --`
cleanup as a mandatory step, ordered before the `git diff --stat` check. Committed `56acda30d`.
No third quorum round spent.

**This is docs-mission's first use of the narrow-refinement carve-out.** Per the skill's
ratification rule (a controller-authored gate change needs a one-time human OK on its first use
per mission before the SPRINT it unblocks runs), `sprint-planner` and `sprint-executor` were **not
spawned** this iteration — the design is ready but held. Filed as decision **D-3** (OPEN) in the
charter's ledger. `sprint-evaluator` accordingly had nothing to independently judge (no code
landed this iteration, so no generator≠judge step was owed — an evaluator spawned against zero
diff would be theater, not review).

**Gate 3b.** Not the sprint-landing kind (no code, no PR, no required-check gate to wait on) — but
this record's own push needed the discipline anyway. Before pushing, `git fetch origin dev` showed
`origin/dev` had moved (`55891002f` → `e620929cd`, three new V1 commits, none touching
`design_docs/**`); rebased `docs-4-brief` onto the new tip (clean, no conflicts — disjoint files),
which **renumbered every commit SHA cited above** (`7de9ca9ed`→`1c74a4971`,
`141ffb5e9`→`fbc289f6a`, `d3a87808e`→`56acda30d`; corrected everywhere in this log and the
charter's D-3 row after the fact, per rule 3b(v)(b) — a SHA cited before it is final is a claim
that expires). Re-checked `origin/dev`'s new tip: still red on the same signature (`test`,
`launchd drivers`, `Build ubuntu-latest`) plus a newly-red `lint` — not improved, not this
mission's fix either way. Pushed `docs-4-brief:dev` directly (matching this mission's established
pattern — docs-2-brief and every prior iteration's record landed the same way, direct push, no
PR), confirmed via a second `fetch` that `origin/dev` now equals the pushed tip.

**Gate 4.** Dashboard overwritten (`design_docs/docs-mission-dashboard.md`). STATUS rotation:
adding iteration 6 pushed the charter to 4 stamps; moved iteration 2's stamp to
`docs-mission-status-archive.md` (prepended after the file's 4-line header, ahead of iteration
1's existing entry) — archive line-count grew by exactly the 49 lines removed from the charter
(assertion run before commit). Queue item 11 (`docs-4`) retagged `[PARKED]` → `[IN-SPRINT]` with
its design-ready state and D-3 pointer. Decision ledger: added D-3 (OPEN), validated
(`scripts/mission_decisions.sh --check` → "decision ledger valid: 3 rows").

**Routing evidence**: designer `claude:claude-fable-5` (rotation pointer was unset, this run is
the mission's first — write `claude:claude-fable-5` to
`~/.ailang/state/mission-docs-designer-rotation` at Gate 5 so the next designer spawn advances to
`pi:ollama/deepseek-v4-flash:0731-cloud`); planner/executor/evaluator **not spawned** — blocked on
D-3 ratification, not a probe failure, so no fallback chain was traversed and none is owed in this
row. Fable diet: 1 doc, 1 initial run + 0 designer-spawned revisions (both quorum-response fixes
were controller-measured single commands, per Gate 2's rule not to re-route trivial verification
back through the designer) — within budget.

**Cost**: metered $0.2516 of $1 ceiling (round-1 quorum $0.1297 + round-2 $0.1219, both
OpenRouter-billed reviewer calls). Quota buckets: fable (1 designer run), sonnet (controller
session, all Gate 0-2/4 work + both quorum-objection verifications).

**Next**: `docs-4`'s sprint (planner → executor → evaluator) resumes the moment D-3 is answered —
if OKed, it is next iteration's pick outright (no re-verification of the design needed, only a
freshness check per Gate 2's standard reality-check); if unanswered, `docs-4` still ranks first
in the queue at design-ready, zero cost accruing either way. `docs-3` (item 10, PR #1031) remains
the other resume point, blocked on V1's inherited CI red — re-check `mergeable` fresh each
iteration rather than assuming it is still true. 31 individually-evidenced STILL-PLANNED docs
from iteration 5 remain available as a fallback pick if both docs-3 and docs-4 are still blocked
next time.

**Ruled out**: nothing this iteration — no hypothesis was refuted, only two designer premises
were verification-completed (already true, not overturned).

**DECISIONS FOR MARK**: **D-3** (OPEN, new this iteration) — one-time OK for docs-mission's first
use of the narrow-refinement carve-out on `docs-4-brief.md`. See the ledger row for the full ask;
loop recommends **(a) OK it**. Default if unanswered: `docs-4` stays parked at design-ready, no
cost, no harm.

**FLAGGED**: `origin/dev`'s `test`/`Build macos/windows/ubuntu-latest`/`launchd drivers (bash
3.2)` red persists (inherited, V1's domain, same signature flagged in iterations 1/2/4/5 — sent a
fresh cross-mission notice this iteration since it now also blocks a second PR, docs-4's future
sprint in addition to docs-3's #1031).

**Retro — no skill edit.** No ≥2-instance friction against the shared skill itself this
iteration: the quorum-at-pick, narrow-refinement-carve-out, and Gate-1 repo-ownership-scoping
disciplines all worked exactly as documented, and the ratification gate did exactly the job it
exists to do (held a controller-authored bypass mechanism to a human check on its first live use
in this mission, at zero cost while waiting). One process observation for this mission's own
future iterations, not the shared skill: a controller-measured quorum-objection fix (a single
`curl`/`grep`, no design judgment) is cheaper and just as valid as re-spawning the Fable designer
for a one-line correction — worth keeping as the default rather than reflexively re-routing every
objection back through the designer role.

## ITERATION 7 — 2026-09-03T17:20Z

**Gate 0.** Kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD
detached at `origin/dev` tip (`70e453060`), clean working tree. Watermark check on bookkeeping
issue `#979`: 0 directives from `MarkEdmondson1234` since `2026-09-03T10:32:25Z` (8 comments
total, none allowlisted). Decision ledger valid, 3 rows — D-1/D-2 `RESOLVED`, **D-3 still `OPEN`**
(last touch on that row is the fleet bot's own iteration-6 commit, not an attended ruling —
correctly left open, not actioned, not re-asked). 20 unread canonical-inbox messages: `eval-suite`
run notifications, `mission-world`/`mission-v1` cross-mission traffic (including one `approvals`
inbox item that is `mission-world`'s own D-WORLD-31, not this mission's), `pkg:*` package task
notices, `design-doc-creator` self-completion — none addressed to `mission-docs`, none acted on or
acked (ack --all would sweep outbound cross-mission inboxes per the standing hazard).

**Gate 1.** `git fetch origin`; local `dev` branch (shared with the main checkout, `e3318d045`)
is far behind `origin/dev`, expected — the pin worktree's detached HEAD (`70e453060`) already
equals `origin/dev`'s tip, so no divergence to repair for this loop's own reads. SHA-addressed
check-runs on `origin/dev` HEAD: 14 total, only `test` non-green and it was `pending` (CI
genuinely in-flight on a fresh push, not a red) — no dev-red to action this iteration.

**Gate 2 — pick, via the blocked-external-row re-verification rule.** Both queue items were
`[IN-SPRINT]`/blocked: `docs-4` on D-3 (no predicate to re-run, only Mark can answer — left
alone), `docs-3` on a claimed "V1-owned inherited CI red" that iterations 1/2/4/5/6 had each
*re-asserted* without re-running the predicate (iteration 6 explicitly wrote "not a stale-base
problem... not actionable this iteration" — this was WRONG, and nobody had checked). Re-measured
from scratch: `gh api commits/<sha>/check-runs` on two independent recent `dev` commits
(`08ab6ba7c`, `5506424f8`) showed `test`/`Build ubuntu-latest`/`launchd drivers (bash 3.2)` all
`success`, while PR #1031's own head (`178072e3f`) was confirmed (post-hoc, by the evaluator)
**91 commits behind** `origin/dev`. The red was base-inherited the whole time and `dev` had
already been fixed underneath it — the blocker was stale, not the diff. **Picked `docs-3`.**

**Execution — mechanical, no new design/plan/execute pass owed.** The diff was already designed,
executed, and independently evaluated (sonnet, isolated worktree, PASS 85/100, zero blocking) in
iteration 5; this iteration's job was landing, not generating. Reused the existing
`.wt-docs-iter5-docs3` worktree (already checked out on the PR branch, clean except an untracked
`.snap/` executor-snapshot leftover). `git fetch origin dev && git rebase origin/dev` — clean, no
conflicts (`git diff --stat` of the base range showed dev's only overlapping-directory change,
`BenchmarkDashboard/ModelComparisonTable.jsx`, was a different file from the PR's own
`BenchmarkDashboard/index.jsx`). Verified `git diff --stat origin/dev HEAD` was the identical 4
files, same line counts, before and after the rebase. Attempted local `make docs-build`
verification: failed twice, both instrument gaps rather than code defects — (1) the worktree had
no `node_modules` (fixed via a symlink to the pin worktree's copy, confirmed byte-identical
`package-lock.json` first); (2) `docs/src/data/packages-sidebar.json` (tracked) references
`docs/docs/packages/sunholo/*.mdx` pages that are gitignored/generated by
`docs/scripts/sync-registry.sh`, which CI's `docusaurus-deploy.yml` runs before `make docs-build`
but this mission's own Makefile target does not — ran `sync-registry.sh` by hand (fetched 42 live
packages), hit a second, unrelated `Cannot read properties of undefined (reading 'id')` SSG error
on retry, did not chase it further since CI's own `docs-build` job (which runs the real pipeline
including a fresh registry sync) was the authoritative gate and later passed cleanly on both the
PR and the merge commit. Reverted the incidental regenerated-file diffs the local attempts left
behind (`docs/docs/design-docs.md`, `docs/docs/prompts/current.md`, `docs/docs/roadmap/index.md`,
`docs/src/data/packages-sidebar.json` — all sync-script byproducts, never meant to be committed)
before pushing. Force-pushed (`git push --force-with-lease`). Scanned the PR title/body for
GitHub auto-close keywords per Gate 4's rule (none matched; the known-bad control
`this fixes #1` fired correctly in the same breath). Watched all 16 PR checks settle green —
`test` (24m17s), `docs-build` (10m43s), `SonarCloud`, `build`, `Build ubuntu/macos/windows-latest`,
`launchd drivers (bash 3.2)`, `govulncheck`, `lint`, `CodeQL`, `docs-gate`, `docs-changes`,
`test-windows`, `Analyze Go` — confirmed `mergeable: MERGEABLE` (polled past the initial `UNKNOWN`
reading), squash-merged → `663237dc70b683c2257dda59943ada395205426a`.

**Post-merge verification (the merge commit is a DIFFERENT commit from the tested PR head).**
Polled `gh api commits/<merge-sha>/check-runs` to completion: 16 checks, 15 green, one
`SonarCloud Code Analysis: failure`. Checked the merge commit's own parent (`41ea6e5ff`, untouched
by this diff) — identical `SonarCloud` failure — confirmed pre-existing/inherited, V1's domain per
Gate 1's repo-ownership scoping, not blocking.

**generator≠judge — independent evaluator, spawned post-hoc.** This run's standing instruction
requires a judge even for a mechanical landing the controller itself verified (no code was
generated this iteration, but the controller still made the judgment call that a stale-base
diagnosis was correct and safe to act on). Spawned `Agent(subagent_type="general-purpose",
model="sonnet")` with none of the controller's own findings handed in as fact — told only what to
check and where. It independently re-pulled the merge commit's diff (confirmed exactly 4 files,
31/17, content matching the stated `benchmarkFetchWithSource`/`source=` wiring, no scope creep, no
syntax red flags), independently re-pulled check-runs for both the merge commit and its parent
(same 15-green/1-SonarCloud-red split on both), and independently confirmed the pre-rebase PR head
was genuinely stale (not an ancestor of `origin/dev` post-rebase, 91 commits behind). **Verdict:
PASS.** No findings.

**Outcome: LANDED.** `docs-3` (item 10) tagged `[LANDED]` in the charter. `docs-4` (item 11)
unchanged — still `[IN-SPRINT]`/design-ready/held on D-3, not re-picked (Standing rule 1, one
backlog item per iteration; D-3 has no predicate to re-run).

**Next**: `docs-4` is the only remaining queue item — if D-3 is OKed, `sprint-planner` runs next
iteration with no re-verification of the design needed beyond Gate 2's standard freshness check;
if unanswered, it stays at design-ready, zero cost accruing. 31 individually-evidenced
STILL-PLANNED docs from iteration 5's backlog sweep remain available as a fallback pick if D-3 is
still unanswered next time.

**Ruled out**: the "docs-3's CI red is not a stale-base problem" claim, standing since iteration 5
and explicitly re-asserted in iteration 6 — REFUTED. It was a stale-base problem; `dev` had fixed
the underlying red and nobody had re-run the check for 2+ iterations. No shared-skill gap here —
the skill already documents both the blocked-external-row re-verification rule (Gate 2) and the
base-inherited-red-vs-genuine-red distinction (Gate 3b) in detail; this mission's own iterations
simply hadn't applied them to this row. Filed as a mission-log lesson, not a skill edit.

**DECISIONS FOR MARK**: **D-3** (OPEN, unchanged, re-asked implicitly by staying in this report
per the digest contract, not a new ask) — one-time OK for docs-mission's first use of the
narrow-refinement carve-out on `docs-4-brief.md`. See the ledger row for the full ask; loop
recommends **(a) OK it**. Default if unanswered: `docs-4` stays parked at design-ready, no cost,
no harm.

**FLAGGED**: none new. The `test`/`Build *-latest`/`launchd drivers (bash 3.2)` red flagged in
iterations 1/2/4/5/6 as V1-owned and inherited is **CONFIRMED FIXED** on `dev` as of this
iteration — do not re-flag it without re-checking; the only remaining known-inherited red is
`SonarCloud Code Analysis`, still V1's domain, unchanged.

**Retro — no skill edit.** The friction this iteration surfaced (a stale external-block verdict
carried unquestioned across 5 iterations) is a case the shared skill already covers in detail at
two separate points (Gate 2's blocked-external-row rule, Gate 3b's base-inherited-red rule) — the
gap was in this mission's *application* of an existing rule, not in the rule's absence, so no
skill edit is owed (Gate 5 routing: ≥2 recorded instances against the *same skill gap* is the bar,
and there is no skill gap here to point at). Recorded as a mission-log lesson instead: when a
queue row's blocking reason names another mission's red, re-run the actual check before repeating
the verdict, especially after several iterations have passed — the CI landscape moves out from
under a stale note faster than the note gets re-read.

## ITERATION 8 — 2026-09-04T02:19Z

**Pick: docs-4** (item 11, `[IN-SPRINT]`, held on D-3). D-3 resolved by Mark (attended,
2026-09-03, recorded directly in the decision ledger under the ATTENDED LEDGER EDITS contract) in
the window since iteration 7: APPROVED the narrow-refinement carve-out for `docs-4-brief.md`, with
one condition — close `gemini-3-1-pro`'s recurring section-boundary-verification objection class
exhaustively (every Phase-B section-cut boundary the brief asserts, not only the B3/B4/B5 the
reviewers happened to name across the two blocked rounds).

**Satisfying D-3's condition.** Audited the brief's Phase B for every item that defines a boundary,
not just a whole-page deletion: B1 (partial carry-over from a deleted page), B2 (whole-page
deletion, no boundary), B3/B4/B5 (in-page trims). B3/B4/B5 already had Verification Log rows
(V29, V30) from the quorum rounds; B1's carry-over boundary — the `Automated Feedback (Advanced)`
section in `cross-project-messaging.mdx`, whose ~15-line bash block moves into `agent-messaging.md`
— had none. Measured directly: `grep -nE '^##+ ' docs/docs/guides/cross-project-messaging.mdx` →
`232:## Automated Feedback (Advanced)` immediately followed by `257:## Semantic Search`, both
genuine H2s (`## `, not H3/bold), body = one intro line + a single ` ```bash ` fence spanning lines
236-253 (15 lines of bash, matching the brief's own "~15-line bash block" description), then a
`---` separator before the next heading. Boundary exact. Added as V31 to the Verification Log,
committed and pushed directly from the pin worktree (detached HEAD, no local `dev` branch — that
branch belongs to a different worktree; pushed via `git push origin <sha>:dev` rather than moving
a local ref) — `df36055ce`. CI green (16 checks, only the pre-existing inherited SonarCloud red).

**Routing — sprint-planner.** `tools/launchd/resolve-role-spawn.sh planner` → `recipe
codex:gpt-5.6-luna declared:codex-ok`. Preflight probe rc=0. First real run: the wrapper script
that launches the bounded `codex exec` subprocess embedded the directive text directly into an
*unquoted* heredoc used to write the wrapper file to disk — the directive's own parentheses
(`"Phase A (A1 sidebar cruft removal..."`) broke the wrapper's shell syntax, and the launched
background process died instantly with `syntax error near unexpected token '('`. Caught by
checking the rc-file artifact and the wrapper's own stderr log rather than trusting the tool's
"command completed" notification (Standing rule 7's launcher-vs-work distinction — the notification
was for a shell that forked-and-immediately-exited, not for codex actually running). Fixed by
rewriting the wrapper with placeholder tokens substituted via `sed` after writing (never
interpolating the directive text into the heredoc itself), and — new habit for the rest of the
iteration — running `bash -n` on every generated wrapper before launching it. Second run: rc=0
after ~13 minutes, produced `design_docs/docs-4-sprint-plan.md` (228 lines) and `sprint_docs-4.json`
(6 milestones: M1 Phase A, M2 B1, M3 B2, M4 B3, M5 B4, M6 B5+final-gates), both well-formed
(`jq -e .` valid, no placeholder/TODO strings), no files touched outside the two artifacts.
Verified the plan faithfully translates the brief's own sequencing and acceptance checks before
trusting it (read all 6 milestones' task lists against Appendix A/B and the brief's 8 acceptance
commands — all present, in order, check 6's cleanup step correctly ordered before check 7).
Copied into the main checkout (destination paths confirmed empty first, no clobber), committed,
worktree removed — `72902585d`. CI green.

**Routing — sprint-executor.** Same lane (`codex:gpt-5.6-luna`), fresh isolated worktree (sibling
of the repo, never `/tmp`), preflight probe rc=0. Directive: execute M1-M6 in order, snapshot
cumulative file state into `.snap/M<k>/` after each milestone (multi-milestone recipe — the
executor cannot commit inside this sandbox at all, and the plan's own 6-milestone granularity
called for per-milestone commits for bisectability), zero git-write operations delegated (the
recipe's mandatory exception, the one `git checkout --` cleanup command in check 6, explicitly
carved out). Real run: rc=0 after ~17 minutes, all 6 `.snap/M<k>/` directories present and
cumulative (confirmed by listing each — DELETED markers correctly present for the 3 deletions,
carried forward across subsequent snapshots).

Reconstructed 6 individual commits in the pin worktree by copying each `.snap/M<k>/` over the tree
in order and committing (handling DELETED markers as `git rm`). Verified byte-identity against the
executor's own final worktree state before trusting the reconstruction: `git diff --name-only
72902585d..HEAD` → exactly the 12 files the brief's Files section lists; per-file `diff` against
the executor's live worktree → all MATCH (or consistently deleted in both). Commits:
`2a336cfde` (M1) `01fc61531` (M2) `28957e86a` (M3) `72795d71b` (M4) `b4290838b` (M5) `67a76e0a9`
(M6).

**The executor's own in-sandbox acceptance run reported checks 5 and 6 as FAILED, out of 8.**
Per this loop's standing rule that an in-sandbox gate verdict is not evidence (`codex
--sandbox workspace-write` sees a different filesystem/network view than the real tree),
re-ran all 8 checks myself, unsandboxed, in the pin worktree.

*Check 6* (`sync-registry.sh && make docs-build`) — in-sandbox failure was
`Could not fetch registry index — skipping package page generation`, then `make: ***
[docs-build] Error 6`. Confirmed sandbox-only: unsandboxed, `sync-registry.sh` fetched 42
packages cleanly (real network access), and `make docs-build` proceeded past that step —
but then failed for a DIFFERENT, real reason: `[cause]: Error: Docusaurus found broken links!`
— 4 dangling relative-path links: `./cross-project-messaging.mdx` referenced from
`agent-workflows.mdx:377`, `claude-code-integration.mdx:90`, `hooks-setup.mdx:207` (all
"Related Documentation" list/table entries); `./development.md` referenced from
`getting-started.mdx:405` ("Next Steps" list). None of these were caught by acceptance check 4
(`grep -rlE "guides/(cross-project-messaging|development)([^a-z0-9_-]|$)"`) because a
sibling-directory relative link inside `docs/docs/guides/` never carries the `guides/` prefix
that pattern requires — the identical instrument gap the brief's own V14 inbound-link discovery
step had (V14 used the same `guides/`-prefixed pattern and consequently never found these 4
either, at authoring time). Verified `agent-workflows.mdx`/`claude-code-integration.mdx` were
never in the brief's declared blast radius by *enumeration* (Files list) but ARE within it by
*rule* (`docs/docs/guides/**`), and B1/B2's own deletion criterion ("every inbound link is
retargeted in the same commit") makes fixing them a completion of an already-specified hard
requirement, not new scope. Fixed: retargeted the 3 cross-project-messaging references to
`./agent-messaging.md` (the B1 merge target — already documents inter-project messaging per the
brief's own V12) and the development.md reference to `/docs/guides/development-workflow`
(matching the precedent the brief set for `debugging.md`'s equivalent link). Re-ran the build:
`[SUCCESS] Generated static files in "build"`; only 3 remaining warnings (broken anchors on
`notify-daemon`, `strict-fallbacks`, `wasm-ai-step-byo-key`), confirmed pre-existing by diffing
against the pre-sprint base commit (`72902585d`) — none of the three target/source pages were
touched by this sprint. Ran the mandatory cleanup, confirmed clean, committed — `1afa42f37`.

*Check 5* (both halves) — failed in-sandbox AND unsandboxed, identically. Investigated rather
than dismissed or force-fixed. First half (`ailang messages ack --all` count expected to drop
from 4 to 2 "hooks-setup and cross-project-messaging gone"): the brief's OWN V8 verification row
lists the pre-sprint 4 matching files as `agent-messaging, hooks-setup, agent-workflows,
claude-code-integration` — `cross-project-messaging.mdx` was never among them (confirmed:
`git show 2a336cfde:docs/docs/guides/cross-project-messaging.mdx | grep 'ailang messages
ack --all'` → no match). And `hooks-setup.mdx` has the string TWICE pre-sprint: once inside the
`Message System` section (removed by B4, correctly) and once inside a separate `Quick Start §
3. Check Messages` subsection that was never in the brief's B4 scope and is never mentioned
anywhere else in the brief — so the file can never fully disappear from `grep -l` regardless of
correct execution. The achievable, correct count is 4 (unchanged) — `agent-workflows.mdx` and
`claude-code-integration.mdx` were never edited by any milestone (Appendix A: "keep, unchanged"
for both) and both legitimately still match. Second half (`_ollama_embed` expected to no longer
list `semantic-caching-how-to.mdx`): checked heading positions of every occurrence pre- and
post-trim (`git show b4290838b:...` vs current) — pre-sprint 5 occurrences, 2 inside `Two-Tier
Search Architecture` (B5's scope, now gone) and 3 inside `Embeddings Doctrine`/`Killer Examples`
(explicitly preserved by the brief: "Preserve `Embeddings Doctrine` and the remaining caching
how-to content"); post-sprint exactly the 3 preserved ones remain. This half is unsatisfiable by
the brief's own explicit design decision to keep Embeddings Doctrine. Filed both as measured
brief-authoring defects (a test-plan row's expected value nobody checked at authoring time,
same class this skill already names for mutation-kill claims) — not silently patched around, not
force-passed; the corrected expected values were independently re-derived by the round-1 evaluator
below before being trusted as more than the controller's own claim.

**generator≠judge — independent evaluator, `sonnet`, isolated worktree (never shared with the
controller's own verification tree).**

*Round 1.* Directive: independently verify the full diff against the brief and plan, with specific
instructions to independently re-derive (not take on faith) the controller's three self-flagged
corrections — the check-5 analysis above (both halves) and the M7 fix's correctness/scope. Result:
**FAIL, 68/100.** Confirmed the controller's three self-flagged items correct on independent
re-derivation (re-ran the pre-sprint grep on the base commit directly, re-checked heading
boundaries, re-ran the build from a clean worktree with its own `npm install`). But found ONE
finding neither the controller's own verification pass nor the executor's acceptance run had
caught, from doing an independent FULL pre/post sidebar-id-set diff (not scoped to `guides/` the
way the brief's own check 3 is): `diff <(git show 72902585d:docs/sidebars.js | grep -oE
"'[a-zA-Z0-9_/.-]+'" | sort -u) <(grep -oE "'[a-zA-Z0-9_/.-]+'" docs/sidebars.js | sort -u)` showed
two unauthorized removals beyond the intended set — `'prompts/current'`, `'prompts/index'` — both
NON-GUIDE ids that lived inside the dissolved `Prompts` category (which also held
`guides/ai-prompt-guide`, correctly re-homed to the top level). Appendix B's own header states
"non-guide ids unchanged"; the brief nowhere authorizes touching `docs/docs/prompts/*`. Both pages
still exist on disk (`docs/docs/prompts/index.md`, `docs/docs/prompts/current.md`) with live
inbound links from `why-ailang.mdx`, `start-here/for-ai-agents.mdx`, `agent-integration.mdx`,
`ai-prompt-guide.mdx` — so they had become fresh nav-unreachable orphans, exactly the clause-3
defect class this entire sprint exists to eliminate, created as collateral damage of the M1
category-dissolution edit and invisible to all 8 of the brief's `guides/`-scoped acceptance checks.
BLOCKING.

Reproduced independently before acting (rule: a judge's finding is a claim too, however clean the
rest of its report reads): re-ran the exact diff command myself, confirmed the two removals,
confirmed both files exist on disk, confirmed the 4 inbound links via grep. Real, and not
previously caught by anyone — the controller's own full-diff review (`git diff --stat`) only
checked file *count and identity* against the brief's Files list, never the *sidebar-id content*
of `sidebars.js` beyond the guide-id count check 3 already runs.

Fixed as **M8**: restored `prompts/index` and `prompts/current` immediately before
`guides/ai-prompt-guide` — their exact original relative position inside the now-dissolved
category, now sitting beside their sole surviving former sibling. 2-line surgical diff. Re-ran
the full sidebar-id diff (now shows only the intended set: 2 category labels removed, 2 guide ids
removed, 9 orphans wired — no more unauthorized removals); re-ran the build (clean, same 3
pre-existing warnings); guide-id count unchanged at 70 (non-guide ids don't count toward it);
mandatory cleanup clean. Committed, pushed — `0750f8dbf`. CI green (20 checks, only the same
inherited SonarCloud red).

*Round 2.* Fresh isolated worktree, same evaluator model, directive carrying forward round 1's
BLOCKING finding and all NON-BLOCKING items by name (per the multi-round protocol — a remedy
directive is an instrument too, so every measurement in it was re-taken on the current tree rather
than copied from round 1's reading). Instructed to re-run the exact diff command itself (not trust
the fix commit's message), confirm the fix commit's surgicality via `git show --stat`, rebuild from
a completely fresh `npm install`, and spot-check (not re-derive from scratch) 2-3 of round 1's
already-clean findings. Result: **PASS, 97/100.** Blocking finding confirmed-fixed (diff now shows
only the intended set, both ids reachable at their original-equivalent position, files exist on
disk); fix commit confirmed surgical (`git show --stat`: 1 file, 2 insertions, 0 deletions); full
clean rebuild reproduced independently (fresh `npm install`, no new warnings beyond the same 3
pre-existing); acceptance checks 1-4/6-8 re-spot-checked with no regression. 3 points held back
only for the still-open check-5 brief defect and the 3 pre-existing broken-anchor warnings, neither
attributable to this sprint or this fix.

**Outcome: LANDED.** `docs-4` (item 11) tagged `[LANDED]` in the charter, full record written
above the queue. 8 commits total on `dev`, all individually CI-green: `df36055ce` (V31) →
`72902585d` (plan) → `2a336cfde`..`67a76e0a9` (M1-M6, 6 commits) → `1afa42f37` (M7 link-fix) →
`0750f8dbf` (M8 sidebar-restore).

**Next**: the charter's own enumerated queue is now fully `[LANDED]`/`[RULED OUT]` — no `[NEXT]`
or `[IN-SPRINT]` row remains. The next iteration's pick is a fresh draw from
`design_docs/planned/`, specifically the 31 individually-evidenced STILL-PLANNED design docs
iteration 5's docs-8 sweep already classified and left pickable (no new aggregate audit needed —
that sweep is the mission's accurate, current backlog). Recommend the next iteration runs Gate 0's
weekly external-issue sweep check (due-date dependent) before picking, since the charter's own
queue no longer provides a default next item.

**Ruled out**: nothing new this iteration beyond the two brief-authoring defects already filed
under check 5 above (not "ruled out" in the sense of a refuted hypothesis — they're confirmed
defects in the brief's acceptance text, just not ones warranting a fix-the-code response).

**DECISIONS FOR MARK**: none — D-3 answered and consumed this iteration, ledger 3 rows, 0 OPEN.

**FLAGGED**: none new. The known-inherited `SonarCloud Code Analysis` red remains V1's domain,
unchanged, confirmed present on every commit this iteration's CI touched.

**Retro — no skill edit.** Two frictions this iteration, both single-instance so far (below the
≥2-instance bar for a skill edit), recorded here as watch-items rather than acted on:
1. The wrapper-script heredoc-interpolation bug (directive text with parentheses breaking an
   unquoted heredoc) is a NEW mechanism, distinct from every previously-documented codex-recipe
   false-green (stdin, delivery-assertion, sandbox-socket-denial, gate-list-not-baselined). The
   generalisable fix — never interpolate a multi-line directive into a heredoc used to author a
   script; write placeholder tokens and `sed`-substitute after, then `bash -n` before launching —
   is now this iteration's own applied practice but not yet promoted to the shared skill. If a
   second mission hits the same class, that is the trigger.
   2. The `guides/`-prefixed-only inbound-link grep pattern (both check 4's own acceptance command
   AND the brief's V14 discovery step) is blind to sibling-directory relative links by
   construction, and this is the SECOND time a design-doc author's link-discovery instrument in
   this mission has under-enumerated inbound links this way (docs-3/docs-1 didn't hit it since
   their edited pages had no sibling-relative-link carriers, but the underlying grep-pattern gap
   is generic to any `docs/docs/guides/**` deletion). Worth surfacing to `design-doc-creator` as a
   standard inbound-link-search snippet (both `guides/<name>` absolute AND `./<name>` relative,
   scoped to the same directory) if a THIRD instance appears in this or another docs-taxonomy-style
   sprint — one more instance from the ≥2 bar.

Both frictions are now first-party-verified working practice for this iteration; recorded so the
NEXT iteration (or a sibling mission reading this log) inherits the lesson even before either
crosses the skill-edit bar.

## ITERATION 9 — 2026-09-05T11:19Z

**Pick**: `docs-11` — `design_docs/planned/v0_29_0/m-dx27-docs-search-github-fallback.md`
(GitHub code-search fallback for `ailang docs search` outside the source tree). Fresh draw from
the 31-doc STILL-PLANNED backlog docs-8 (iteration 5) certified, since the charter's own
enumerated queue (items 1-11) is now entirely `[LANDED]`/`[RULED OUT]`.

**Gate 0**: kill switch armed; billing tripwire CLEAN; gh account `sunholo-voight-kampff`
confirmed active. Bookkeeping issue `#979` — `scripts/mission_directives.sh` found 0 allowlisted
directives since the watermark (`2026-09-03T18:44:06Z`, of 10 comments). Decision ledger valid
(`scripts/mission_decisions.sh --check`), 3 rows, 0 OPEN at pick time. Canonical inbox (Firestore,
`store: gcp` header confirmed) had 30 unread messages: 26 routine eval-suite/task-completion
notifications, an `AILANG v0.35.0 Released` notice, and two user-submitted feedback items
(a language feature request for process exit codes, a compile-cache staleness bug) both filed
against core-language surfaces — out of docs-mission's scope, not actioned, left for whichever
mission ingests general feedback. None addressed to `mission-docs`, none allowlisted, none a
directive. Weekly external-issue sweep checked and found NOT due: `#979` created 2026-08-31, next
Monday-07:00-CEST boundary is 2026-09-07.

**Gate 1**: `git fetch origin` — local `dev` and `origin/dev` identical (`087fbea63`), no
reconcile needed. Per-workflow check: `CI` success, `Deploy Documentation to GitHub Pages`
success. SHA-addressed `commits/<sha>/check-runs` on `origin/dev` HEAD: `checks=13`, 4 NOT-GREEN,
all `pending` — confirmed via `gh run list` these are genuinely in-flight `Build and Release` runs
for the last 3 commits (`createdAt` 10:43:57Z/10:51:16Z/10:57:54Z, `status: in_progress`), not a
red. Not actioned; not blocking.

**Gate 2 — reality-check and cross-mission-ownership screen.** The charter's "Next" note (iteration
8) pointed at the 31-doc backlog. Before picking, checked the one item flagged with partial
progress — `m-net-effect-proxy-boundary` ("M1 of 4 landed") — for ownership, per Gate 1's
repo-ownership scoping and Gate 2's PR/worktree-attribution rules for a shared repo:
`git log --oneline --all --grep="net-effect-proxy"` returned commits citing "iteration 145/150/
155/156" and "D-6", "AC10(d) tripwire", "CI-flake M2" — none of which are this mission's own
numbering (docs-mission is at iteration 0-8) or vocabulary. This is V1's active, in-progress,
multi-milestone item. RULED OUT as a pick before any routing cost was spent — not a refuted
hypothesis, an attribution catch.
Picked `m-dx27-docs-search-github-fallback.md` instead. Verified NOT already landed: `git log
--all --grep="dx27\|docs-search-github"` shows only the original 2026-01-28 creation commit
(`a111c5662`), no PR, no implementation activity. Verified the problem statement still reproduces
at HEAD: `ailang --version` → `v0.35.0-61-g087fbea63`; `internal/docsearch/embed.go` exists but
implements an unrelated semantic-search embedding cache, not the "embed docs in binary"
alternative the doc considered and rejected; `cmd/ailang/docs_search.go`'s `findDocsDir()` (line
183) still returns exactly `"no documentation directory found (tried design_docs/ and docs/)"` —
the doc's quoted error, verbatim, at the current binary version.

**Gate 2 — QUORUM-AT-PICK.** No quorum artifact existed for this doc (`ls
.ailang/state/mission-quorum/ | grep dx27` → empty before this iteration; the doc predates the
quorum gate). Ran `ailang design-quorum` with a controller pass verdict and note before any
routing, per the standing rule for pre-quorum docs.

**Round 1 — BLOCKED, 3/3 reject** (artifact
`m-dx27-docs-search-github-fallback-2026-09-05T11-05-41Z.json`, cost $0.0456):
- **gpt5-6-sol**: the doc's central success case (unauthenticated GitHub code search at
  10 req/min) had no Verification Log entry — an unverified premise. Also: "fall back to
  unauthenticated on invalid token" is a silent fallback, forbidden by the project's no-silent-
  fallbacks axiom.
- **gemini-3-1-pro**: ~600 LOC of new HTTP client / git-remote parser / cache layer proposed to
  be wired directly into `internal/docsearch/search.go` and `cmd/ailang/docs_search.go`, violating
  PROGRAM.md's "default bias: extension, not core change." No Conflict-Surface section existed.
- **oc-glm-5-2**: a `github://` prefix sentinel hardcoded into the shared `Search()` function is
  the same core-vs-extension violation from a different angle; repeats the silent-fallback
  objection.

None disputed that the feature itself was worth building — all three objected to premise
verification, architecture placement, or fallback behavior. Routed to the designer role for a
revision (Gate 3's ordinary any-reject path, not yet the narrow-refinement carve-out — round 1's
objections spanned multiple surfaces and were not yet narrow).

**Designer spawn** (`tools/launchd/resolve-role-spawn.sh designer <doc>` → `recipe
claude:claude-sonnet-5 declared:provider-pin` — this mission's own env pin, distinct from the
fleet designer rotation, per the Repo Profile note that a mission's own pin wins over the shared
seed). Billing guard confirmed CLEAN (`test -z "$ANTHROPIC_API_KEY" && test -z
"$ANTHROPIC_AUTH_TOKEN"`) before every `claude-sub` invocation. 1-token probe (`claude-sub -p
--model claude-sonnet-5 'reply with exactly: ok'`, bounded 120s) → rc=0. Real run: directive file
(4,716 bytes, well over the 200-byte delivery-assertion floor) naming all three objections
verbatim and instructing live verification rather than argument; launched via `Bash
run_in_background:true` wrapping a bounded 30-min `date +%s` deadline loop (single-level
backgrounding — the outer call blocks on the inner work, so the harness's completion notification
reflects real completion, not a launcher return). Completed in ~9 minutes, `rc=0`.

**What the designer did** (verified first-party by the controller before accepting, not taken on
the designer's word): live-`curl`'d GitHub's actual code-search API —
`curl -sD - "https://api.github.com/search/code?q=contracts+repo:sunholo-data/ailang" -o /dev/null`
→ `HTTP/2 401` ("Requires authentication"; no unauthenticated tier exists at all, falsifying the
doc's central claim), and the same request with `Authorization: Bearer $(gh auth token)` →
`HTTP/2 200` with `x-ratelimit-resource: code_search`, `x-ratelimit-limit: 10` (not the doc's
claimed 30 — that number was the generic REST limit, never checked against the actual resource).
Rewrote Success Case, Rate Limit Case, and Configuration around "authentication required,"
removed all silent-fallback language project-wide in the doc, and redesigned the implementation
around a `SearchBackend` interface (mirroring `internal/observatory/backend.go` and
`internal/ai/provider.go`'s existing interface-selection patterns) so that
`internal/docsearch/search.go`'s `Search()` function is not modified at all — backend selection
moves entirely into the CLI layer. Added a Conflict Surface section (cache path, env var,
`GITHUB_TOKEN` reuse, all grep-verified). Diff: `git diff --stat` → 1 file, +341/-133.

**Round 2 — BLOCKED again, 3/3 reject** (artifact
`m-dx27-docs-search-github-fallback-2026-09-05T11-11-47Z.json`, cost $0.0731):
- **gpt5-6-sol**: the revised Conflict Surface checks only cache paths and env-var names, not
  whether equivalent HTTP-client/git-remote-parsing code already exists elsewhere before
  proposing new implementations — fails the route-to-extension/reuse gate despite the new
  interface.
- **gemini-3-1-pro**: independently named the same gap, specifically citing that
  `coordinator_cloud_github.go` already handles GitHub PR creation and the doc never evaluates
  reusing it before proposing a redundant ~210 LOC (150 LOC HTTP client + 60 LOC git-remote
  parser) from scratch.
- **oc-glm-5-2**: the doc asserts `internal/docsearch/search.go`'s actual `Search()` signature and
  `SearchOptions`/`SearchResult`/`SearchStats` types as fact, with no Verification Log entry
  proving they match reality — the Verification Log only covered the external GitHub API.

**Controller-verified both claims directly** (single-command, no design judgment) before deciding
how to route: `find . -iname "coordinator_cloud_github.go"` → `cmd/ailang/coordinator_cloud_github.go`
(411 lines); `grep -n "^func "` shows `getGitHubOwnerRepo(ctx, workDir)` at line 87 — parses
`git remote get-url origin`, matches both the HTTPS and SSH-style GitHub remote URL prefixes, splits
`owner/repo` — functionally identical to what the doc's Phase 2 proposed writing from scratch; and
`createGitHubPR`/`addGitHubLabels` in the same file establish an existing
`Authorization: Bearer <token>` + `Accept: application/vnd.github+json` +
`X-GitHub-Api-Version: 2022-11-28` request pattern. Separately, `grep -n "func Search\b|type
SearchOptions|type SearchResult|type SearchStats" internal/docsearch/search.go` confirmed all four
of the doc's claims (lines 18/31/38/62) match verbatim — oc-glm-5-2's underlying premise was
correct, the objection was a valid process gap (no log entry), not a wrong claim.

**Applied the narrow-refinement carve-out** (`.claude/skills/mission-control/SKILL.md` Gate 2):
both round-2 objections carried a concrete, reviewer-named fix and neither disputed the
`SearchBackend` design direction — a completeness/reuse-check gap and a verification-log gap, not
an architecture dispute. The controller applied both fixes directly rather than spawning a second
designer run or spending a third ~$0.07 quorum round: Phase 2 revised from "write
`detectGitHubRepo`/`parseGitRemote` from scratch" to "extract `getGitHubOwnerRepo` into a new
shared `internal/gitutil` package, update `coordinator_cloud_github.go`'s call site to use it,
delete the private original" (net LOC estimate revised from ~510 to ~455 — the extraction removes
more duplication than it adds); added the missing Verification Log row citing the grep above.
Also noted, honestly, that the HTTP-client pattern is followed for header consistency but not
extracted into a shared client, since each caller's request/response shape differs enough (PR
creation vs. label POST vs. code-search GET) that a generic wrapper would only be a thin
convenience, not a real dedup.

This is docs-mission's **SECOND** use of the shared skill's narrow-refinement carve-out. The
FIRST (iteration 6, `docs-4`) was explicitly scoped by Mark's own ratifying decision (D-3) to that
item alone ("Precedent is scoped to `docs-4` alone and does not generalise to docs-mission") — so
this is a fresh first-use-per-item ratification, not a re-application of an existing grant. Per
the carve-out's own rule ("the FIRST doc to use the carve-out is surfaced to Mark for a one-time
OK before its sprint runs"), the sprint is HELD — `sprint-planner` does not run this iteration.
Filed as decision **D-4** in the ledger, modeled on D-3's format: two options with consequences, a
loop recommendation ((a) OK it), and a safe unanswered-default (stays parked, no cost).

**No planner, executor, or evaluator spawned this iteration.** This is a correct application of
Standing rule 8 (a judgment park, not a capacity park — D-4 is answerable in one word) and
Standing rule 2 (never force a guardrail through) — there is no sprint plan yet to execute and
nothing to hand an independent judge, so spawning either role would either sit idle awaiting a
plan that does not exist, or (worse) tempt the controller to fabricate a plan to give them
something to do. The generator≠judge requirement is satisfied vacuously here in the correct
sense: no generation happened outside the design-doc revision itself, which the quorum's three
independent reviewers already judged (twice).

**Outcome: PARKED.** `docs-11` added to the charter's queue as item 12, tagged `[PARKED]`, held on
**D-4**. Design doc committed with both revision rounds and the controller's carve-out fixes.

**Ruled out**: `m-net-effect-proxy-boundary` as a pick — it is V1's own item, not a hypothesis that
was tested and refuted, but recorded here so a future iteration does not re-derive the same
attribution check from scratch.

**Cost**: metered **$0.119** of $1 ceiling — two quorum rounds ($0.0456 + $0.0731), both
OpenRouter-billed reviewer calls. Quota buckets: sonnet (controller session), the designer's
underlying model billed via `claude-sub`'s subscription-only path (billing guard confirmed CLEAN
throughout, never touched the metered API key).

**Routing evidence**: designer `claude:claude-sonnet-5` — recipe path (`declared:provider-pin`),
1 bounded background run via `claude-sub`, ~9 minutes, rc=0, for the round-1 revision. Round-2
fixes applied by the controller directly, per the carve-out's own allowance for single-command
verification with no design judgment (matching how V29/V30 and `docs-4`'s D-3 condition were
closed by the controller in prior iterations). Quorum: 2 rounds, `gpt5-6-sol`/`gemini-3-1-pro`/
`oc-glm-5-2` all present both rounds (`absent_reviewers: []` both times) — no degrade. Planner,
executor, evaluator: not spawned, per the park classification above. Controller session: sonnet.

**DECISIONS FOR MARK**: **D-4** (NEW) — one-time OK to use the narrow-refinement carve-out for
`m-dx27-docs-search-github-fallback.md`, docs-mission's second use of it. See the ledger row for
the full ask, options, and recommendation. Default if unanswered: `docs-11` stays parked at
design-ready, no sprint runs, no cost accrues.

**FLAGGED**: none new. `Build and Release`'s 4 in-flight checks on `origin/dev` HEAD (Gate 1) are
not yet resolved as of this iteration's Gate 5 — worth a glance next iteration only if they land
non-green; not actioned here since they were `pending`, not red, at every point they were checked.

**Retro — no skill edit.** One friction this iteration, below the ≥2-instance bar: the 31-doc
STILL-PLANNED list docs-8 certified (iteration 5) mixes items this mission can freely pick
(`m-dx27`, no other mission's activity) with items another mission actively owns
(`m-net-effect-proxy-boundary`, V1's). The list itself carries no ownership annotation — the
"M1 of 4 landed" note that flagged this one as worth checking was a lucky textual hint, not a
structural signal, and a doc with no such hint (a clean `[STILL-PLANNED]` with zero visible
progress markers) would need the same `git log --grep` check run blind, every time, with no cue to
run it. Recorded as a watch-item rather than acted on: if a future iteration is caught by an
UN-hinted collision from this same list, the fix is annotating docs-8's 31-doc table with an
ownership/last-touched column at classification time (a one-time pass over 31 docs) rather than
re-deriving attribution per pick indefinitely — that crosses the ≥2-instance bar for a skill or
charter-table edit.
