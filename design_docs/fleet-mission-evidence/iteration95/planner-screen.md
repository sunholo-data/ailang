# Fleet iteration 95 — planner readiness screen

**NO-PLAN.** At supplied base `252680e1beb4e6cf1709a010cc1c45838b91b9f8`, no OPEN fleet decision row supplies evidenced authorization for a next step needing a production sprint plan. Existing proposals remain parked, and other remaining steps need human policy/scope decisions, attended owner work, or final design approval. No plan, sprint JSON, implementation, handoff, policy activation or ledger resolution was created.

**Verification limit:** live GitHub state could not be obtained. Both `gh` list/view and direct API reads failed to connect; the connected GitHub fallback requires reauthentication. Thus this is a completed local readiness screen with an explicit remote-verification gap, not certification that current PRs or issue comments are unchanged. No approval is inferred from unavailable data. The parent’s report of no directive on #2039 is consistent with the last banked observation, but was **not independently confirmed live** here.

## Scope and evidence read

Classification: **read-only planning-readiness analysis**, with only this report permitted as a write. Read `CLAUDE.md` completely first (152 lines), the entire `.claude/skills/sprint-planner/SKILL.md` (597 lines), and the entire `design_docs/fleet-mission.md` charter (408 lines). Located the fleet records with `rg --files design_docs | rg fleet-mission`; read the complete iteration index and recent log entries 92–94. Read the prior planner screen, observations, PR/ticket captures, and iteration-93 quorum/final-judge evidence. Oversized combined outputs were followed by bounded rereads of required text.

Primary authority is [charter Authority](design_docs/fleet-mission.md#authority-the-write-scope--enforced-by-toolslaunchdgithookspre-push), lines 77–95, and [Guardrails](design_docs/fleet-mission.md#guardrails-on-top-of-the-skills-standing-rules), lines 97–130. Policy changes park under HD-2a; the draft exception screens the recommended/shared **first step**, and does not approve policy or a design. Scope widening lands alone before dependent paths become available on the promoted pinned guard.

Recent record anchors:

- `design_docs/fleet-mission-index.md:15–17`: iterations 94, 93 and 92, read in the context of the complete index.
- `design_docs/fleet-mission-log.md:1461–1505`: iteration 93 drafted D-FLEET-25’s mechanism, then parked it after two blocked quorums and NEEDS-HUMAN-REVIEW. Its Next at lines 1494–1497 expressly requires human direction and final design approval before planning; it forbids redrafting without a ruling.
- `design_docs/fleet-mission-log.md:1507–1542`: iteration 94’s screen, nine preserved drafts, 25 captured tickets, 26 OPEN decision rows, and no new production authorization. Lines 1524 and 1534 state the competing-candidate exclusions and resume prerequisites.
- [Iteration-94 ticket capture](design_docs/fleet-mission-evidence/iteration94/tickets.json) contains 25 signatures. [Iteration-93 signature map](design_docs/fleet-mission-evidence/iteration93/executor-readiness.md), lines 101–133, maps all 25 to decisions. The two CI probe signatures map through charter queue entries 13/19 to D-FLEET-18, rather than occurring literally in that ledger row.

These captures are prior evidence, not a fresh ticket or remote query. No messages command was run.

## GitHub reads and provenance

Read-only `gh` attempts:

```text
gh pr list --repo sunholo-data/ailang --state open --limit 100 --json number,title,url,isDraft,headRefName,headRefOid,updatedAt,baseRefName
gh issue view 2039 --repo sunholo-data/ailang --json number,title,state,url,body,comments,updatedAt
gh api 'repos/sunholo-data/ailang/pulls?state=open&per_page=100' --jq '<fleet/ metadata filter>'
gh api repos/sunholo-data/ailang/issues/2039/comments --paginate --jq '<comment metadata and body>'
```

Each returned `error connecting to api.github.com`. The last two failed at approximately `2026-11-07T19:49:14Z`; that is attempt time, not a successful observation time. Connected GitHub read-only calls for issue #2039, its comments, and PR #2165 all returned `UNAUTHORIZED: This app connection requires reauthentication`.

[Issue #2039](https://github.com/sunholo-data/ailang/issues/2039) is the charter’s current bookkeeping issue (`fleet-mission.md:21,34`). Last banked evidence at `fleet-mission-log.md:1515` and [iteration94/observations.md](design_docs/fleet-mission-evidence/iteration94/observations.md):6 reports 21 comments and zero allowlisted directives, with the whole thread examined because its watermark was absent. This corroborates the parent historically; it cannot rule out later comments. No directive was marked or acknowledged. Before promoting any row based on new GitHub approval, the controller must obtain a successful current comment/PR read and attribute the exact decision and approved artifact.

## Existing parked PRs

**All states below are recorded OPEN/draft in the iteration-94 capture, not independently verified current states.** Source: [open-prs.json](design_docs/fleet-mission-evidence/iteration94/open-prs.json), nine records; charter/log supply the recorded head prefixes and dependencies. The snapshot’s `updatedAt` values are PR update timestamps, not API observation times. Current state, head freshness, reviews, new PR completeness and mergeability remain unknown from this screen.

| PR | Recorded head / role | Human prerequisite; why no new sprint plan |
|---|---|---|
| [#2065](https://github.com/sunholo-data/ailang/pull/2065) | `edb099371`, D-FLEET-26 A | Choose A for transient capacity classification. Existing proposed diff already supplies that step; refresh landing checks after authorization. |
| [#2067](https://github.com/sunholo-data/ailang/pull/2067) | `bdaad2825`, D-FLEET-21 A | Choose A for the CI poll/time-budget change; prior evaluator PASS is not a policy ruling. |
| [#2082](https://github.com/sunholo-data/ailang/pull/2082) | `399cf799b`, D-FLEET-25 A/B shared correction | Authorize A/B’s factual World-fork/reach correction. This is separate from the mechanism design. |
| [#2108](https://github.com/sunholo-data/ailang/pull/2108) | `261fb3b64`, independent D-FLEET-36/-38 guard commits | Corresponding A scope rulings for each commit. Land the authorized scope arm alone first; downstream grep/sync work waits for that promoted guard. One row’s ruling cannot approve the other commit. |
| [#2110](https://github.com/sunholo-data/ailang/pull/2110) | `875581f46`, D-FLEET-18 A guard | Choose A; land this guard arm alone first. Probe repair/retirement needs the admitted scope and its own remaining gates. |
| [#2141](https://github.com/sunholo-data/ailang/pull/2141) | `4ea46f05e`, D-FLEET-42 A M1 | Attended A acceptance for per-lane secret scope; host hook is separately attended. |
| [#2143](https://github.com/sunholo-data/ailang/pull/2143) | `7f1e13d5f`, D-FLEET-42 A M2, stacked on #2141 | Same A ruling, dependency-order integration and refreshed landing checks. |
| [#2150](https://github.com/sunholo-data/ailang/pull/2150) | `26b713887`, D-FLEET-42 A M3, stacked on #2143 | Same A ruling. M1–M3 are already proposed; historical healthy/degraded probes supply evidence, not authorization. |
| [#2165](https://github.com/sunholo-data/ailang/pull/2165) | `985aa0161`, D-FLEET-25 mechanism design | Human direction on a corrected design, required independent review and explicit final design approval before planning. Proposal existence does not supply approval. |

D-FLEET-33 also has an existing [proposed patch](design_docs/fleet-mission-evidence/iteration66/unowned-dirty-proposed.diff), not an additional PR in that capture.

## Every OPEN decision row

Line references below are exact row locations in `design_docs/fleet-mission.md` at the supplied base. All 26 have Status OPEN. There is no D-FLEET-30 row; D-FLEET-24 is RESOLVED and implemented, so neither supplies new open work.

| Row / charter line | Required human prerequisite / next authorized boundary |
|---|---|
| D-FLEET-16, **163** | Choose Phase 3b skill-source mechanism. A begins with attended user-skill retirement; pi `--skill` and Codex parity tails do not bypass that choice. Default C. |
| D-FLEET-17, **164** | Attended/V1 ownership of out-of-scope CI failures, or explicit B authority expansion. Historical red-check observations do not grant standing authority. Default C. |
| D-FLEET-18, **165** | Choose scope expansion A or owner routing B. A’s first guard step is already #2110; no authorized downstream probe work before its separate landing. Default C. |
| D-FLEET-19, **166** | Attended quota-project decision/change after checking other rig consumers, or owner authorization for out-of-scope Pub/Sub code. Default C. |
| D-FLEET-20, **167** | Choose quorum OpenAI subscription routing A or credit top-up B. Both are routing/billing or outside fleet code scope. Default C. |
| D-FLEET-21, **168** | Rule on CI poll/time budget. A already exists as #2067; no duplicate plan. Default C. |
| D-FLEET-22, **169** | Attended prod coordinator inbox declaration and revision roll, or choose Daneel’s controlplane route. This includes the captured reporting recurrence. Default C. |
| D-FLEET-23, **170** | Assign scratch/failure-reporting and Step-5b work to attended infra/coordinator/V1 ownership, or choose narrow fleet expansion B. The guard-first step is a non-recommended option, not shared by all non-default options. Default C. |
| D-FLEET-25, **172** | Choose A/B; #2082 covers only their factual correction. Mechanism #2165 needs corrected-design direction and explicit final design approval before any sprint plan. Default C. |
| D-FLEET-26, **173** | Choose transient retry A versus capacity demotion B. A is already #2065. Finished-record pushing is a separately scoped design, not automatically approved. Default C. |
| D-FLEET-27, **174** | Confirm Luna ruling; recommended/default A keeps shipped behavior and needs no implementation plan. B would require an explicit routing ruling plus attended env changes. |
| D-FLEET-28, **175** | Attended/V1 authorization for quorum admission on unknown quota, or explicit one-invocation bypass. Admission policy/out-of-scope command; default C. |
| D-FLEET-29, **176** | Attended safe shared-binary installation/rebuild ownership, or explicit B per-fire binary authority. Recommended A starts outside the worktree and has no new fleet sprint to plan. Default C. |
| D-FLEET-31, **177** | Answer the remaining output-cap policy question. Recommended/default A is no further code change; B changes seating policy, C changes out-of-scope model config. Ticket is not resolved by silence. |
| D-FLEET-32, **178** | Explicitly waive in-run cap ask 3 (recommended B), or choose A and approve its design. Earlier margin authorization did not authorize a new in-run cap. Default C. |
| D-FLEET-33, **179** | Approve existing extension patch and eval-impact statement under D-FLEET-3; assign the out-of-scope embedded `pi_assets` copy, or approve B scope widening. Existing patch needs a ruling, not a duplicate plan. |
| D-FLEET-34, **180** | Explicitly waive shorter pi thresholds (recommended B), or reopen D-FLEET-4/-7 via an A design decision. Default thresholds unchanged; no waiver inferred. |
| D-FLEET-35, **181** | Owner/attended registry and pi-model cap changes and confirmation of wire limits; optional 402 reclassification needs its own policy choice. Recommended A starts out of scope. Default unchanged. |
| D-FLEET-36, **182** | Choose A fleet scope expansion or B attended grep fix. A guard is already independently proposed on #2108; actual skill changes and bootstrap work remain gated. |
| D-FLEET-37, **183** | Attended/V1 ownership for coordinator/worktree and unexplained exit-1 asks; explicitly give ask 2 a home before resolving. B is broad non-recommended coordinator scope expansion. |
| D-FLEET-38, **184** | Choose A for 16 generated-mirror paths or B attended reconciliation. A guard already exists on #2108; sync extension follows its landing. Ten content merges remain attended; design-doc-creator is D-36. Default C. |
| D-FLEET-39, **185** | Attended/V1 retry/terminal-state ownership; first investigate how attempt-0 lease release allowed retry reacquisition. A covers all cloud-task completion drops; A2 alone does not. Default C. |
| D-FLEET-40, **186** | Choose attended/V1 shutdown-strand repair plus stated non-goals (A), or broad coordinator design (B). Existing split/push/focused-test work is already built; residual work is outside scope. Default C. |
| D-FLEET-41, **187** | Recommended A is attended Docker toolchain installation. In-scope skill option B requires its own choice: it is not A’s first step or common to A/B. Default C. |
| D-FLEET-42, **188** | Choose A for already-proposed #2141 → #2143 → #2150; separately attend host hook, credential rotations and redaction permission. Default B remains detection only. |
| D-FLEET-43, **189** | **Explicitly choose A before even building the fleet idle-skip proposal.** Row says “built as a proposed diff only if you choose A”; no ticket-backed exception overrides this. Default B keeps full fires. |

Seven OPEN rows do not have their own captured ticket: D-FLEET-16/-21/-27/-29/-41/-42/-43. They were still screened. Conversely, multiple captured tickets share D-FLEET-18/-39/-40, so 25 tickets must not be mistaken for 25 decision rows. Current ledger OPEN count is 26.

## D-FLEET-25 approval boundary and contrary candidates

The recorded #2165 design is frozen at SHA-256 `bad5f6a2f40a1f6bdcca2495be6b4b92c0e31386d61822e4e2de7341375e1eac`. [Quorum round 2](design_docs/fleet-mission-evidence/iteration93/quorum-r2.md):1–8 records BLOCKED after the one revision; lines 49–53 explicitly deny third-revision/code authorization and call it an unapproved sprint input. Deleting an entire claimset can evade authorization discovery; immutable approval binding, provenance strength, supersession, canonical bytes, helper verification and runtime roots also need resolution. [Final independent judge](design_docs/fleet-mission-evidence/iteration93/design-judge.md) returns NEEDS-HUMAN-REVIEW. Its missing-pick-artifact observation was a reviewed-base visibility limitation, explained by [readiness notes](design_docs/fleet-mission-evidence/iteration93/readiness-notes.md); that explanation does not clear the substantive blockers.

The strongest alternative is to reuse the idle draft exception to revise #2165 again. The explicit iteration-93 stop and current D-FLEET-25 row override that inference: draft eligibility was authority to propose, not approval of A or the final design. A reviewer’s proposed fix is not an attributable human ruling. Likewise, D-FLEET-43’s unbuilt A, D-FLEET-41’s in-scope B, and D-FLEET-16’s in-scope tails do not provide approved planning inputs.

**Resume condition:** obtain fresh remote evidence; identify an attributable scope/policy decision and any required final design approval for the exact artifact. Then screen whether the next step is new planning, review/landing of an existing proposal, or attended owner work. An A ruling alone does not automatically approve a blocked final design. Execution and seed activation retain separate gates.

## Preservation and completion

Initial worktree status was clean. HEAD was checked twice before writing and matched the supplied base. No git write operation, tracked-file edit, inbox command, subprocess agent, mission-control iteration, test, socket probe or GPU job ran. No background artifact was started, so no polling or deferred wait is needed. Only root `planner-screen.md` was created, intentionally untracked; final HEAD/status checks are recorded below.

Final verification: HEAD `252680e1beb4e6cf1709a010cc1c45838b91b9f8`; status exactly `?? planner-screen.md`; tracked and staged diffs empty. Report matrix coverage checked against the actual ledger: 26/26 OPEN rows, no missing or duplicate row.
