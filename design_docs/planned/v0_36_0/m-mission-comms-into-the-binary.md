# M-MISSION-COMMS-INTO-THE-BINARY: Decisions Are Issues, Reports Are Links, Telemetry Leaves the Thread

**Status**: Planned — **CONTINGENT on HD-1**. The execution plan below assumes HD-1(a)
(charter canonical). If Mark rules (b) or (c), Phase 1's writer direction inverts and this
doc requires revision, not merely re-planning. Nothing is executable until HD-1 is ratified.
**Target**: v0.36.0
**Priority**: P1
**Estimated**: 4 days (Phase 1: 2d, Phase 2: 1d, Phase 3: 1d)
**Dependencies**: None. Builds on `internal/messaging.GitHubClient` (V8), which must gain per-call deadlines as part of Phase 1 (V18).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Replaces 7 ad-hoc `gh issue comment` shell-outs (V7) with one typed call path; the same iteration state produces the same artifact. |
| A2: Replayability | +1 | Reports become structured records with stable ids, so an iteration's comms can be replayed from banked state rather than scraped from prose. |
| A3: Effect Legibility | +1 | Network effects move from ambient `gh` subprocess calls scattered through a 1250-line script to a declared client with `PreFlightChecks` (V8). |
| A4: Explicit Authority | +1 | Author gating (the provenance contract behind `mission_answer.sh`) gets ONE enforcement point via `ValidateUser`, instead of being a convention re-implemented per call site. |
| A5: Bounded Verification | +1 | Go tests replace bash-3.2-only suites that only run on a macOS CI job (V6, V10). |
| A6: Safe Concurrency | 0 | No concurrency change. Bounded-call behaviour is preserved as-is. |
| A7: Machines First | +1 | The core of the doc. A 52KB linear thread (V1) is queryable by nobody; labelled issues are queryable by both the loop and a board. |
| A8: Minimal Syntax | 0 | No language surface. New CLI subcommands only. |
| A9: Cost Visibility | +1 | Per-iteration metered spend is prose inside a 1,951-char comment today; it becomes a structured field. |
| A10: Composability | +1 | Rides the existing `ailang messages` plane and `GitHubClient` rather than adding a parallel channel. |
| A11: Structured Failure | +1 | Replaces `&&`-chained shell-outs whose failure is silent (V11 — a measured, named incident class) with typed errors. |
| A12: System Boundary | +1 | The GitHub boundary becomes one explicit, testable crossing instead of 7 implicit ones (6 write + 1 read, V7), five of which are unbounded today. |

**Net Score: +10** → **Decision: Proceed TO RATIFICATION, then planning.** The axiom score clears the bar on the design's merits; it does not and cannot clear HD-1, which is a human ratification the score has no view over. Sprint planning starts after the Design Freeze checklist is complete, not after this table.

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — strictly removes shell-level nondeterminism.
- [x] A3 (Effects): No hidden side effects — the point is to make an existing hidden effect explicit.
- [x] A4 (Authority): No ambient access granted — narrows ambient `gh` authority to one gated client.
- [x] A7 (Machines First): Not optimizing for human convenience — optimizes for machine queryability; the human ergonomics are a consequence.

### Decision Thresholds

Net +10, no −1 on A1/A3/A4/A7 → **✅ Proceed**.

## Problem Statement

### The measured state of the channel

`sunholo-data/ailang#972` ("V1 mission bookkeeping — week of 2026-08-31"), measured 2026-09-03:

| Metric | Value |
|---|---|
| Comments | 27 |
| Total characters | 52,677 |
| Mean comment | 1,951 chars (max 3,430) |
| Comments authored by the loop | **26 of 27** |
| Comments authored by the human | **1** — the string `D-54 b`, **6 characters** |

The human's total input to the channel is 6 characters out of 52,677 — a signal ratio of roughly
**1:8,800**. The thread is not a discussion that grew long; it is a write-only firehose with a
decision occasionally embedded in it.

### The channel has already lost decisions inside itself

This is not a hypothetical ergonomics complaint. The thread contains, verbatim:

- *"Retraction — iteration 308 re-asked two decisions you had already answered"* (2,458 chars)
- *"Addendum — D-51 and D-52 were answered while this iteration was running"* (1,025 chars)

The loop asked for rulings it had already been given, because a ruling is a 6-character comment in
a linear stream carrying ~2KB every 90 minutes. **A decision has state; a thread does not.** That
mismatch is the root cause, and it is already costing correctness, not just attention.

### Three channels are multiplexed onto one linear stream

| Channel | Direction | Volume | Lifetime | Right shape |
|---|---|---|---|---|
| Iteration reports | loop → human | ~2KB × 16/day (v1) | write-once, never re-read | a link to the in-repo log |
| **Decisions and asks** | **both ways** | **0–3 open fleet-wide (V5)** | **stateful until answered** | **an issue** |
| Telemetry (model switch, lane degraded, pin drift) | loop → nobody | 312–830 chars, 4 of 27 (V3) | disposable | the message plane |

Note the volumes. Open decisions across all four missions are typically **0–3 at a time** (V5), so
"one issue per open decision" is ~2–4 new issues per week — trivial volume, and each gets a title,
an open/closed state, and a notification. The scarce, high-value channel is drowned by the
abundant, low-value one.

### Why the noise exists (do not simply delete it)

The telemetry comments are not carelessness. They are the accumulated **fix** for silent-fallback
bugs. From `tools/launchd/mission-control.sh:670-676` (V11), verbatim:

> Until now a lane demotion was `log`ged here and NOWHERE ELSE — none of this driver's four
> `gh issue comment` sites covers it, so the human channel saw nothing. That is exactly how the
> World mission spent FIVE iterations (18/19/21/22) silently demoted from codex to opus, each
> mis-attributed to a spent quota, before iter-23 found the real cause.

Each time something was invisible, another `gh issue comment` site was added — four then, six
now (V7). So **the thread's noise is the scar tissue of Critical Principle 2** ("no silent
fallbacks"). Deleting the telemetry comments without giving that telemetry somewhere else to go
would re-create the exact class the comments were added to fix. Telemetry must **move**, not
disappear. This is the single most important constraint on this design.

### Why this is also the right first extraction from shell

The mission driver is 25 shell files / 6,148 lines, of which `mission-control.sh` alone is
**1,250 lines** (V6). The repo's own coding standard says 1200+ lines MUST be split — and
`make check-file-sizes` cannot see it, because that gate globs `internal cmd -name "*.go"` only
(V9a). The driver is over the repo's stated ceiling with no gate to say so.

The comms surface is the **best-conditioned** slice to move first:

1. It is a narrow, well-defined seam: 6 write sites plus 1 read site, individually enumerated (V7).
2. The Go capability **already exists and is tested** — `internal/messaging.GitHubClient` provides
   `CreateIssue`, `CloseIssue`, `AddComment`, `GetIssueComments`, `ListIssuesByLabel`,
   `EnsureLabel`, `AddLabelToIssue`, `ValidateUser`, `PreFlightChecks` (V8). This phase is
   **wiring, not new capability**.
3. It has an independent reason to happen now (the comms restructure), so the extraction is not
   speculative refactoring.
4. No Go code currently posts mission reports (V9b, positive-controlled), so there is no existing
   implementation to reconcile — a greenfield home under an existing package.

Doing the comms restructure in shell would mean building it on the surface being retired.

## Goals

**Primary goal**: Make the human↔loop decision channel *stateful and small*, move iteration
reports to links over the record that already exists, relocate telemetry off the human thread
without re-introducing silent fallbacks, and land all of it as the first extraction of mission
driver logic into the `ailang` binary.

**Success metrics:**

1. Human-relevant characters per week in the GitHub thread drop from ~52,677 to **< 8,000**
   (target: report comments ≤ 400 chars each).
2. Every open decision is an **open GitHub issue** with a one-line title; `ailang mission decisions
   --open` lists them across all four missions in one call.
3. **Zero** telemetry comments on the bookkeeping thread, with a demonstrated equivalent signal on
   the message plane (a lane demotion must still reach the human within one fire — the V11 bar).
4. The 6 `gh issue comment` sites and the `gh issue view` reader in `mission_directives.sh` are
   replaced by `ailang mission` subcommands; `grep -n "gh issue" tools/launchd/*.sh scripts/mission_*.sh` returns no CALL sites (comment mentions permitted).
5. Charter markdown remains the single source of truth; a divergence check proves issues are a
   projection, never a second writer.

## High-Impact Decisions

| # | Decision | Options | Who decides | Cost to change later |
|---|---|---|---|---|
| HD-1 | Source of truth for decisions | (a) charter markdown, issues are a one-way projection **(recommended)**; (b) GitHub issues canonical, charter generated; (c) both | **Mark** | **High** — (c) is dual-write and the fleet has been bitten by that class before (feedback dispatch; message-health over-reporting). Reversing later means reconciling two divergent histories. |
| HD-2 | Where telemetry goes | (a) `ailang messages` with a `mission-telemetry` type, surfaced in the weekly digest **(recommended)**; (b) observatory spans only; (c) a second, separate GitHub issue | **Mark** | Medium — (b) risks failing the V11 bar (a human must see a demotion within one fire). |
| HD-3 | Report comment retained at all? | (a) one ≤400-char comment per iteration linking the log commit **(recommended)**; (b) one weekly digest comment only; (c) none | Mark | Low — presentation only. |
| HD-4 | Projects board scope | (a) decisions + top-N queue rows, org-level so it spans `ailang` and `ailang-world` **(recommended)**; (b) decisions only; (c) skip Projects | Mark | Low — a board is a view; deleting it loses nothing. |

### Design Freeze

Before `sprint-executor` starts, these must be checked off:

- [ ] **HD-1 ratified.** The doc assumes (a). If Mark picks (b) or (c), Phase 1's writer direction inverts and the doc needs revision, not just re-planning.
- [ ] **HD-2 ratified**, including the explicit acceptance bar: a lane demotion is visible to the human within one fire (the V11 standard).
- [ ] **The `mission-decision` label schema is fixed** (`mission:v1|world|motoko|docs` + `mission-decision`), because `ListIssuesByLabel` and the Phase 3 board both key off it.
- [ ] **Confirmed**: retiring the thread-comment channel does not break the directive path. Gate 0's directive read moves to `ailang mission directives`, and `mission_directives.sh`'s author gate must be preserved exactly (it is the provenance root — see `mission_answer.sh`'s contract).

## Solution Design

### Overview

Three phases, each independently shippable and independently revertible.

- **Phase 1 (2d)** — `ailang mission` subcommands in Go, replacing the shell's GitHub I/O. Behaviour-preserving: same comments, same thread. This is the extraction, with no policy change, so a regression is attributable to the port and nothing else.
- **Phase 2 (1d)** — the policy change: decisions become issues, reports shrink to links, telemetry moves to the message plane.
- **Phase 3 (1d)** — the org-level Projects board as a read-only view over Phase 2's labelled issues.

Phase 1 before Phase 2 is deliberate: porting and changing policy simultaneously makes any
resulting defect ambiguous between the two.

### Architecture

```
                       ┌─────────────────────────────────────┐
  charter markdown ───▶│ internal/mission/comms  (NEW)        │
  (SOURCE OF TRUTH)    │  - ledger parse (reuse validator)    │
                       │  - report render (≤400 chars)        │
                       │  - decision → issue projection       │
                       └──────────────┬──────────────────────┘
                                      │ one-way, never reads back as truth
                       ┌──────────────▼──────────────────────┐
                       │ internal/messaging.GitHubClient      │
                       │  CreateIssue / CloseIssue / AddComment│
                       │  GetIssueComments / ListIssuesByLabel │
                       │  ValidateUser  (EXISTS TODAY — V8)    │
                       └──────────────┬──────────────────────┘
                                      │
              ┌───────────────────────┼────────────────────────┐
              ▼                       ▼                        ▼
     decision issues          weekly thread            Projects board
     (stateful, 0-3 open)     (≤400-char links)        (Phase 3, view only)

  telemetry ──▶ ailang messages (type: mission-telemetry) ──▶ weekly digest
               [must satisfy the V11 bar: visible within one fire]
```

**The dual-write guard, restated correctly (round 2).** The first draft said "issue → charter is
not a code path that exists" while *also* saying a decision is answered on the issue. Those are
contradictory: an answer posted to an issue that can never reach the charter is silently lost
input, which is worse than the thread it replaces. gpt5-6-sol blocked on this and was right.

The resolution is that **a comment on a decision issue is an authorized INPUT EVENT, not canonical
state**. The distinction is what keeps the charter canonical without making the channel one-way:

1. `ailang mission directives` polls open `mission-decision` issues under a bounded timeout.
2. A comment is accepted only if the author is on the **positive allowlist** (C6/V16 — not merely
   "not a bot"), and the body carries an exact decision id plus valid answer syntax. The port also
   preserves the **self-direction guard** (V20b): if the allowlist ever contains the loop's own
   authenticated account, the reader refuses rather than letting the loop steer itself.
3. It emits an **idempotent directive keyed by the GitHub comment id**, so re-polling cannot double-apply.
4. That directive flows through the **existing validated answer path** (`mission_answer.sh` and the
   `mission_decisions.sh` validator) which writes the charter row — the same path, and the same
   validation, an attended ruling uses today.
5. **Only after the charter row is confirmed RESOLVED** does projection close the issue.

So there is still no path that treats issue text as truth: issue text is a *request* to run the
existing writer, and the charter remains the only thing that decides. Unauthorized or malformed
comments are rejected **loudly** (Critical Principle 2), never silently dropped.

### Implementation Plan

**Phase 1 — extract the I/O (2 days)**

1. `internal/mission/comms/client.go` — thin façade over `GitHubClient` carrying mission identity (name, repo, issue number) so call sites stop passing `MISSION_GH_ISSUE` around as an env string.
2. `cmd/ailang/mission_comms.go` — `ailang mission report`, `ailang mission directives`, `ailang mission telemetry`.
3. Replace the 6 `gh issue comment` sites (V7, enumerated by line) in `mission-control.sh` with `ailang mission report --…`. Preserve the existing bounded-call wrapper (`_mc_bounded 30`) semantics — a hung network call must still not wedge a fire.
4. Replace `mission_directives.sh`'s `gh issue view` reader with `ailang mission directives --json`, **preserving the author gate byte-for-byte**.
5. Go tests for each, including a mission-identity table so all four missions are covered.

**Phase 2 — the policy change (1 day)**

6. `internal/mission/comms/decisions.go` — project OPEN ledger rows to labelled issues; close the issue when the row goes RESOLVED. Idempotent: re-running an iteration must not open duplicates (keyed on decision id).
7. Report renderer capped at 400 chars: what landed, goal distance, cost, link to the log commit.
8. Telemetry re-routed to `ailang messages` with type `mission-telemetry`. Because the existing notification is conditional and warns-and-continues (V17), the telemetry path treats a disabled or failed notification as a **hard error** — a demotion that cannot be delivered must fail loudly rather than bank silently.
9. `ailang mission decisions --open [--all-missions]`.

**Phase 3 — the board (1 day)**

10. Org-level Project, populated from `ListIssuesByLabel`. Read-only view; no writer.
11. `make check-mission-projection` — asserts every OPEN charter row has exactly one open issue and vice versa. This is the anti-divergence gate for HD-1.

### Files to Modify/Create

- `internal/mission/comms/client.go` — NEW, ~180 LOC. Mission-identity façade over `GitHubClient`.
- `internal/mission/comms/decisions.go` — NEW, ~220 LOC. Ledger-row → issue projection, idempotent by decision id.
- `internal/mission/comms/report.go` — NEW, ~140 LOC. The ≤400-char renderer.
- `internal/mission/comms/comms_test.go` — NEW, ~300 LOC. Table-driven across all four missions.
- `cmd/ailang/mission_comms.go` — NEW, ~200 LOC. Subcommand wiring, matching `messages_*.go` conventions.
- `tools/launchd/mission-control.sh` — MODIFY, −120/+40. Removes 6 `gh issue comment` sites; net reduction moves the file toward the 1200-line ceiling it currently exceeds (V6).
- `scripts/mission_directives.sh` — MODIFY, −40/+15. Reader swapped; author gate untouched.
- `make/code-health.mk` — MODIFY, +12. `check-mission-projection`; optionally widen `check-file-sizes` to shell (V9a gap).

## Conflict Surface

Round 1 of quorum rejected 3/3 on the absence of this section, and was right to. The original
disclaimer applied the *narrow* trigger list (parser/typechecker/codegen) when the gate is about
**any** machinery the change overlaps. Writing it surfaced two defects in the plan below, one of
which would have been a security-shaped regression.

**What positions does this change occupy, and what already lives in them?**

| # | Existing machinery | Overlap | Decision | Why |
|---|---|---|---|---|
| C1 | `internal/messaging.GitHubClient` | The write path: issues, comments, labels | **REUSE the primitives, but NOT as-is — it must gain per-call deadlines** | Exports every needed primitive (V8), but has **no per-call timeout** (V18). Round 1-2 wording said "reuse as-is" while also promising uniformly bounded calls; those are incompatible, and gpt5-6-sol blocked on it in round 3. Bounding is now in scope for Phase 1. |
| C2 | `internal/coordinator.GitHubPoster` **read side** — `GetRecentHumanComments`, `GetLatestHumanComment`, `IsBotUser` (V12) | Reads human comments off an issue since a timestamp, filtering bots | **REUSE for the directive reader** | This is a real reuse the first draft missed: `mission_directives.sh` hand-rolls a `since` cursor and bot filtering that already exist in tested Go. |
| C3 | `internal/coordinator.GitHubPoster.PostFeedback` / `PostFeedbackInRepo` (V12) | Posts a comment to an issue | **DO NOT reuse for mission reports** | Signature is feedback-shaped (`iteration`, `channel`) and it cannot create or close issues, which decision projection requires. Using it would mean bending a coordinator concept to carry mission reports. Mission reports call `GitHubClient.AddComment` directly. |
| C4 | `cmd/ailang/messages_send.go:108` — `syncToGitHub := *github \|\| knownGitHubTypes[category]`, where `knownGitHubTypes = {bug, feature}` (V13) | Decides whether a sent message is mirrored to GitHub | **DO NOT EXTEND — and assert it stays unextended** | **The defect this section caught.** `syncMessageToGitHub` creates a GitHub **ISSUE** per message (V14). Had `mission-telemetry` been added to that map, every lane demotion would open an issue — putting on GitHub, more loudly, exactly what this doc removes. A test asserts `mission-telemetry ∉ knownGitHubTypes`. |
| C5 | Pub/Sub notification on send (`messages_send.go:171-185`) → notify daemon → macOS + Discord | The existing human-delivery path | **REUSE, but it does NOT meet the V11 bar as-is — the design must harden it** | **The defect gemini's objection exposed (V17).** The notify is conditional on `cfg.PubSub.Enabled`, and a notifier error prints to stderr and continues with `notified=false`. A loop's stderr goes to a `/tmp` file nobody reads, so telemetry could vanish exactly as silently as the pre-2026-08-12 lane demotions. Phase 2 therefore treats a disabled-or-failed notification **on the telemetry path** as a hard error, not a warning. |
| C6 | `scripts/mission_directives.sh` author **allowlist** (`:74-82`, jq positive list, dies if empty) | The provenance root for human directives | **PRESERVE SEMANTICS EXACTLY — do not substitute `IsBotUser`** | **The second defect this section caught.** C2's `IsBotUser` is a *denylist* ("is this a bot"); the directive gate is an *allowlist* ("is this Mark"). They are not equivalent — swapping them would let any non-bot human author a directive. Reuse C2's fetching and cursor; keep the allowlist check. |
| C7 | `scripts/mission_decisions.sh` ledger validator | Validates the charter's decision table | **REUSE, read-only** | The projection reads validated ledger state. It never writes the charter, which is what makes HD-1 structural. |
| C8 | The 6 `gh issue comment` sites + 1 `gh issue view` reader (V7) | Today's entire mission↔GitHub surface | **REPLACE** | The extraction itself. Phase 1 is behaviour-preserving so a defect is attributable to the port. |
| C9 | GitHub Projects automation | — | **NONE EXISTS** (V15) | Greenfield; nothing to conflict with. Phase 3 adds the first. |

**Programs that MUST still work post-change:**

1. `scripts/mission_directives.sh` — a directive from an allowlisted author is still ingested at Gate 0; a comment from a non-allowlisted human is still ignored (C6 is the regression bar).
2. `ailang messages send --type bug` — still opens a GitHub issue. C4 must not disturb the existing `{bug, feature}` behaviour.
3. `scripts/mission_answer.sh` — untouched; ledger rows keep their format and the validator keeps passing.
4. All four mission drivers complete an iteration and write a slot verdict (the `START_EPOCH` path fixed in `8b6b4409c`).

**What deliberately changes:** iteration reports stop being ~2KB of prose on the thread and become
≤400-char links; open decisions gain a second representation as issues (a projection, never a
second writer); telemetry leaves the thread for the message plane.

## Examples

### Example 1: A decision, today vs. after

**Today** — buried at position 21 of 27 in a 52KB thread, answered with 6 characters:

```
sunholo-voight-kampff  2284 chars  **Iteration 325 — the hook is ARMED (sprint 4/4)…**
                                   …[2,000 chars of report]…
                                   **Decisions for you:** D-54 — ship rule A or hold row 50?
MarkEdmondson1234         6 chars  D-54 b
```

**After** — a stateful object with a readable title:

```
$ ailang mission decisions --open --all-missions
#1041  [v1]     D-54  ship rule A as ratified, or hold row 50 for the fixture migration?
#112   [world]  D-WORLD-31  same question, world ledger
(2 open, 0 stale >7d)
```

Answering does **not** close the issue directly. The comment becomes an allowlisted, idempotent
directive; the existing validated answer path writes the charter row; and only once that row reads
RESOLVED does projection close the issue. `check-mission-projection` then proves the two agree.

### Example 2: A lane demotion — the V11 regression bar

The failure this design must not re-introduce: World ran five iterations silently demoted because
no comment site covered lane degradation (V11).

```
# Phase 2 acceptance test, run against a simulated demotion:
$ ailang mission telemetry --event lane-degraded --role executor --from codex:gpt-5.6-sol --to opus
✓ banked to message plane (type=mission-telemetry, id=msg_…)
✓ notification raised — visible to human within this fire

$ ailang messages list --unread --json | jq '[.[]|select(.type=="mission-telemetry")]|length'
1
```

The thread stays clean **and** the demotion is still visible within one fire. If the second
assertion fails, Phase 2 is not shippable — telemetry has been deleted rather than moved.

## Success Criteria

- [ ] `grep -n "gh issue" tools/launchd/*.sh scripts/mission_*.sh` returns no call sites, and all six ported paths are bounded (today only 1 of 6 is — V7).
- [ ] Weekly human-relevant characters in the bookkeeping thread < 8,000 (from 52,677).
- [ ] Every OPEN ledger row across all four missions has exactly one open labelled issue; `make check-mission-projection` green.
- [ ] A simulated lane demotion is visible to the human within one fire with **zero** thread comments (the V11 bar).
- [ ] `mission-telemetry` is absent from `knownGitHubTypes` and no telemetry message creates a GitHub issue (C4/V13/V14), asserted by test, not by inspection.
- [ ] A comment from a non-allowlisted human is still ignored by the directive reader (C6/V16) — the allowlist semantics survive the port.
- [ ] `mission_directives.sh`'s author gate is byte-identical pre/post — verified by diffing the gate block, not by asserting behaviour.
- [ ] No code path exists from issue body → charter state (dual-write structurally impossible, HD-1).
- [ ] All four mission drivers run one full iteration on the new path with no `stderr` output.
- [ ] Go tests cover all four mission identities; `make test-launchd-drivers` still green on bash 3.2.
- [ ] Documentation updated: CHANGELOG.md, `docs/docs/guides/agent-messaging.md`.

## Testing Strategy

**Go unit tests** (`internal/mission/comms`): table-driven over the four missions; the projection
is tested for idempotence by running it twice against the same ledger and asserting one issue.

**Mutation arms** — each of these must turn a test RED when applied, or the test is vacuous:

1. Remove the idempotence key → duplicate issues on re-run.
2. Drop the author gate in the directives reader → an unauthorised comment is accepted.
3. Delete the telemetry notification raise → the V11 bar assertion fails.
4. Let the report renderer exceed 400 chars → the cap assertion fails.
5. Add `mission-telemetry` to `knownGitHubTypes` (C4) → the "telemetry never reaches GitHub" assertion fails. This arm is the guard on the defect the Conflict Surface caught: without it, a one-line map edit silently turns every lane demotion into a GitHub issue.
6. Replace the directive allowlist check with `IsBotUser` (C6) → the "non-allowlisted human is ignored" assertion fails. Guards the provenance root against the denylist/allowlist substitution.
7. Disable `cfg.PubSub.Enabled`, then emit telemetry (C5/V17) → the "a demotion that cannot be delivered fails loudly" assertion fails. Guards against reusing the warn-and-continue path unmodified.
8. Feed a decision-issue comment from a NON-allowlisted author → the charter must not move and the issue must not close. Guards the input-event path from becoming an unauthenticated writer.

**Explicit anti-vacuity requirement.** The existing heartbeat suite went green for ~10 iterations
while production died every fire, because the harness supplied `START_EPOCH` — the exact variable
the driver forgot (fixed 2026-09-03, `8b6b4409c`; see `m-mission-slot-heartbeat.md`). Tests here
must not construct the state they are meant to verify. Where a test extracts or stubs, it must
assert the production path supplies the same inputs — the guard arm added in `8b6b4409c` is the
pattern to follow.

**Integration**: one full dry-run iteration per mission against a scratch issue before cutover.

## Open Objections Carried Out of Quorum (PENDING — not resolved)

Three rounds were run (2026-09-03/04, $0.28 metered, 3/3 reviewers present every round, blocked
each time). Rounds 1 and 2's objections are all closed. **Two from round 3 remain open and are
recorded here rather than argued or hidden**, per the skill's rule for exhausted rounds:

| # | Objection | Status | What closing it needs |
|---|---|---|---|
| P-1 | `GitHubClient` has no per-call deadline (gpt5-6-sol, round 3) | **CLOSED 2026-09-04.** Fixed in `761b81371`: `ErrExecTimeout` sentinel, 30s default taken from the driver's own `_mc_bounded 30`, all 14 call sites routed through `execCommandCtx`. Mutation-tested against a REAL child process. | Nothing. |
| P-2 | The 5-step input-event flow names `mission_answer.sh` / `mission_decisions.sh` but neither is in Files to Modify, and how `mission_directives.sh` selects its source issue is unverified (oc-glm-5-2, round 3) | **CLOSED 2026-09-04 (V19, V20).** The objection was right to demand the check; the check shows the three scripts need **no modification**. `mission_directives.sh` is already parameterised by `--issue`, so N issues is N invocations — a caller concern. `mission_answer.sh` and `mission_decisions.sh` are *invoked unchanged*, which is precisely the point of routing through the existing validated path: reusing a writer means not editing it. Files to Modify is therefore correct as written. | Nothing. The port must, however, preserve the three security properties recorded in V20 — especially the self-direction guard. |

**Consequence:** this doc is **not cleared for `sprint-planner`.** It is materially stronger than
round 0 — quorum caught four defects that would otherwise have shipped (C4, C6, the decision-answer
contradiction, V17) plus one factual error in the author's own Verification Log (V7's miscount) —
but blocked is blocked, and P-2 in particular is real design work, not wording.

## Deferred Decisions

The executor may decide these without escalating:

- Exact subcommand naming under `ailang mission` (`report`/`post`, `directives`/`inbox`).
- Whether `comms` is one package or splits into `comms` + `projection`.
- Issue body formatting, provided the title is one line and the decision id is present.
- Whether Phase 3's board is populated by a workflow or an `ailang` subcommand.

## Non-Goals

- **Porting the rest of the driver.** Gate sequencing, model probing, the stall watchdog and the pin logic all stay in shell. This doc is the comms seam only; it establishes the pattern for later extractions but does not perform them.
- **Changing decision semantics.** The ledger format, `mission_answer.sh`'s provenance contract, and the reject-by-default quorum are untouched.
- **Retiring the weekly issue.** It remains, carrying short links.
- **Changing the message plane's storage or topology.**

## Timeline

| Day | Work |
|---|---|
| 1 | Phase 1 items 1–2: package + subcommands, Go tests. |
| 2 | Phase 1 items 3–5: shell call sites swapped, directives reader ported, bash suite green. Ship Phase 1 (behaviour-preserving). |
| 3 | Phase 2: projection, report cap, telemetry re-route, `decisions --open`. Mutation arms. |
| 4 | Phase 3: board + `check-mission-projection`; docs + CHANGELOG. |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Telemetry is deleted rather than moved, re-creating the V11 silent-fallback class | Medium | **High** | The V11 bar is an explicit success criterion with its own mutation arm (#3). Phase 2 does not ship if it fails. |
| Dual-write divergence between charter and issues | Medium | High | HD-1 fixes charter as truth; no issue→charter code path exists; `check-mission-projection` gates it. |
| The port breaks the directive/provenance path — the loop's only human-trust root | Low | **High** | Author gate ported byte-for-byte and diffed, not re-implemented. Phase 1 is behaviour-preserving so any break is attributable. |
| A vacuous test lets a broken port ship | Medium | High | Four named mutation arms; explicit anti-vacuity rule drawn from the `START_EPOCH` incident. |
| Four loops run this concurrently mid-migration | Medium | Medium | Phase 1 is behaviour-preserving, so mixed old/new fires are safe. Cutover is per-mission via the driver pin. |
| Scope creep into "port the whole driver" | High | Medium | Non-Goals names it; Phases 2–3 have no shell dependency. |

## Related Documents

- [m-mission-slot-heartbeat.md](../v1_0_0/m-mission-slot-heartbeat.md) (neural 0.36) — the D-52 slot-verdict instrument. Directly relevant: its extracted-block test seam is what hid `START_EPOCH` for ~10 iterations, and this doc's anti-vacuity rule is drawn from that failure. Distinct topic: that doc is about slot verdicts, this one about the human channel.
- [m-mission-elo-routing.md](../m-mission-elo-routing.md) (neural 0.33) — mission routing. Distinct: routing policy, not comms.
- [m-ailang-native-harness.md](../m-ailang-native-harness.md) — the "move work into the binary" thesis at the *coding harness* level. This doc is the same instinct applied to mission infrastructure. Distinct surface, no overlap.
- [m-driver-pin-rollout.md](../m-driver-pin-rollout.md) — the pin mechanism this migration cuts over through.

## References

**Verification Log** — every load-bearing claim, with the command that proves it. Measured 2026-09-03.

| # | Claim | Evidence |
|---|---|---|
| V1 | v1 thread: 27 comments, 52,677 chars, mean 1,951, max 3,430; 26 bot / 1 human at 6 chars | `gh issue view 972 --repo sunholo-data/ailang --json comments -q '[.comments[].body\|length]\|…'` plus per-author listing |
| V2 | Threads are ALREADY weekly — "per week" is not the missing lever | Titles: `V1 mission bookkeeping — week of 2026-08-31` (#972), world #107, motoko #663 |
| V3 | 4 of 27 comments are pure telemetry (312–830 chars) | Two `🔁 Controller model:` switches, one lane-degraded, one pin-drift, in the #972 listing |
| V4 | The linear thread has already lost answered decisions | Comment titles "Retraction — iteration 308 re-asked two decisions you had already answered"; "Addendum — D-51 and D-52 were answered while this iteration was running" |
| V5 | Open decisions fleet-wide are 0–3, not dozens | v1 54 rows / 0 open; motoko 6 / 0; world 18 / 1; docs 3 / 1 (ledger validator output) |
| V6 | Shell surface 25 files / 6,148 lines; `mission-control.sh` alone 1,250 lines, over the repo's own 1200 MUST-split rule | `wc -l tools/launchd/*.sh tools/launchd/lib/*.sh scripts/mission_*.sh`; `.claude/rules/coding-standards.md` |
| V7 | **6** `gh issue comment` call sites + 1 `gh issue view` reader. **Corrected in round 2**: the first draft said 7, from a `grep -c` that counted a comment mention. Only ONE of the six is bounded | Enumerated, not counted: call sites at `mission-control.sh` **126, 886, 899, 1206, 1224, 1241**; `:672` is prose inside the LANE-DEGRADATION comment. Only `:1206` is wrapped in `_mc_bounded 30` — the other five are unbounded network calls that can hang a fire, which the Go port bounds uniformly. Reader: `scripts/mission_directives.sh:79` |
| V8 | `GitHubClient` already provides every needed primitive | `grep -n "func (c \*GitHubClient)" internal/messaging/github.go` → `CreateIssue:251`, `CloseIssue:442`, `AddComment:545`, `GetIssueComments:499`, `ListIssuesByLabel:334`, `EnsureLabel:636`, `ValidateUser:144`, `PreFlightChecks:224` |
| V9a | **NEGATIVE**: `check-file-sizes` cannot see shell files | `make/code-health.mk:150` globs `find internal cmd -name "*.go"` only |
| V9b | **NEGATIVE**: no Go code posts mission reports to GitHub today | `grep -rln "MISSION_GH_ISSUE" --include="*.go" internal/ cmd/` → empty. **Positive control on the same instrument**: `grep -rln "syncMessageToGitHub" --include="*.go"` → 2 files, so the grep is not silently broken. Widened once: `IssueComment\|CreateComment` finds `internal/coordinator/github_comments.go` — comment *capability* exists in Go, but is not wired to the mission driver |
| V9c | **NEGATIVE**: `internal/mission/` today holds only `quorum/` — the package is a natural, uncontested home | `find internal/mission -type f -name "*.go"` → all paths under `internal/mission/quorum/` |
| V10 | The bash suites run only on a macOS CI job, deliberately pinned to bash 3.2 | `.github/workflows/ci.yml:559` `launchd-drivers` job |
| V11 | Telemetry comments are the accumulated fix for a real silent-fallback incident — so telemetry must MOVE, not be deleted | `tools/launchd/mission-control.sh:670-676`, naming World iterations 18/19/21/22 silently demoted for five iterations |
| V12 | `internal/coordinator.GitHubPoster` has BOTH a read side and a comment writer — but no issue create/close | `grep -n "^func " internal/coordinator/github_comments.go` → `IsBotUser:31`, `GetRecentHumanComments:54`, `GetLatestHumanComment:108`, `ExtractFeedbackFromComments:123`, `PostFeedback:141`, `PostFeedbackInRepo:148`. No `CreateIssue`/`CloseIssue`, so decision projection cannot use it (C2/C3) |
| V13 | GitHub sync on send is CATEGORY-GATED, and the gated set is exactly `{bug, feature}` | `cmd/ailang/messages_send.go:104-108` — `knownGitHubTypes := map[string]bool{CategoryBug: true, CategoryFeature: true}`; `syncToGitHub := *github \|\| knownGitHubTypes[category]`. A new `mission-telemetry` type does NOT sync unless added there (C4) |
| V14 | `syncMessageToGitHub` creates an **ISSUE**, not a comment — so a mistaken C4 extension would be worse than today's thread comments | `cmd/ailang/messages_github.go:265` comment "creates a GitHub issue for the message"; called only at `messages_send.go:233` under the C4 gate |
| V15 | **NEGATIVE**: no GitHub Projects automation exists anywhere in the repo — Phase 3 is greenfield | `grep -rln "gh project\|projectsV2\|ProjectV2" --include="*.go" --include="*.sh" --include="*.yml" internal/ cmd/ tools/ scripts/ .github/` → empty. **Positive control**: the same instrument over `gh issue` returns `tools/launchd/mission-control.sh`, `tools/messaging/docs_inbox_router.sh`, `scripts/test_github_integration.sh` |
| V16 | The directive gate is a positive AUTHOR ALLOWLIST, not a bot denylist — the two are not interchangeable (C6) | `scripts/mission_directives.sh:74-82` builds a jq allowlist array from a comma-separated env value and dies if it resolves to zero authors; `IsBotUser` (V12) tests bot-ness instead. Substituting it would admit any non-bot human |
| V17 | The Pub/Sub notification C5 relies on is CONDITIONAL and degrades to a stderr warning — so reusing it unmodified would NOT satisfy the V11 bar | `cmd/ailang/messages_send.go:171-185`: guarded by `if cfg != nil && cfg.PubSub != nil && cfg.PubSub.Enabled`; both the constructor error and `notifier.Notify` error paths `fmt.Fprintf(os.Stderr, ...)` and fall through with `notified = false`, after which only `warnIfFiledButUndispatchable` runs. Raised by quorum round 2 (gemini-3-1-pro) against an UNVERIFIED premise in round 1's C5 — the check refuted the reuse-as-is plan rather than confirming it |
| V18 | **NEGATIVE**: `GitHubClient` has NO per-call deadline — so "reuse as-is" and "uniformly bounded" could not both be true | `grep -n "context.WithTimeout\|Timeout\|deadline" internal/messaging/github.go` → empty. **Positive control**: the same instrument finds `context.WithTimeout` in `internal/projecteval/projecteval.go`, `internal/coordinator/daemon_lifecycle.go`, `internal/coordinator/provider_script.go`. Raised by quorum round 3 (gpt5-6-sol) as an A5 risk capable of wedging a fire |
| V19 | **P-2 CLOSED.** `mission_directives.sh` is ALREADY parameterised per-issue, so polling N decision issues needs NO change to it — it is invoked once per issue | `scripts/mission_directives.sh:27-40`: `--issue` (required, numeric-validated) and `--repo` flags; `SINCE` cursor; the GitHub read is `gh issue view "$ISSUE" --repo "$REPO"`. Multi-issue is a caller concern, not a script change |
| V20 | Three security properties of the directive gate the port MUST preserve — each verified in source, not inferred | `mission_directives.sh`: (a) positive allowlist, default `MarkEdmondson1234`, `ascii_downcase` both sides (`:25`, `:82-90`); (b) **SELF-DIRECTION GUARD** — dies if the allowlist contains the authenticated `gh api user` login, because otherwise the loop could steer itself by commenting on its own bookkeeping issue (`:55-71`, called out in-source as "the one failure that would be invisible in the report"); (c) refuses an empty or zero-length allowlist (`:22-23`, `:77`) |

**No language claims.** This doc asserts nothing about AILANG syntax or semantics, so the
`ailang check` gate does not apply. **The Conflict Surface section is above** — the first draft
wrongly disclaimed it by applying the narrow parser/typechecker trigger list; quorum round 1
rejected 3/3 on exactly that, and writing the section caught two defects (C4, C6).

**Quorum triggers (attended session):** trigger 1 fires (design-freeze items HD-1…HD-4 are for
Mark to ratify) and trigger 2 fires (it overrides shared machinery — one driver used by all four
missions). Quorum is therefore **required** before planning.

## Future Work

- Extract the remaining driver phases (gate sequencing, probing, stall watchdog) into `internal/mission` once this seam proves the pattern. `mission-control.sh` is 1,250 lines against a 1,200 ceiling (V6); comms is the first cut, not the last.
- Widen `check-file-sizes` to shell so the ceiling is enforced rather than merely stated (V9a).
- Feed decision issues into the executor plane so a ruling can dispatch work directly, closing the loop this fleet has never closed (`project_agent_handoff_chain_never_fired`).
