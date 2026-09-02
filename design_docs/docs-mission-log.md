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
