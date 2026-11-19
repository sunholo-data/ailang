# Iteration 99 idle screen (independent evaluator)

VERDICT: NONE (idle). Confidence: moderate-high.

## Premise / change versus iteration 98
| Check | Command | Result |
|---|---|---|
| Tickets | python json compare iteration98 vs 99 `tickets.json` | identical (27, same signatures, same slots_lost/last_seen) |
| OPEN rows | `diff` of iteration98 vs 99 `open-decisions.tsv` | byte-identical; 28 OPEN rows D-FLEET-16..23, 25..29, 31..45 |
| Harness code on origin/dev | `git log cd5f1da41..origin/dev -- design_docs/fleet-mission.md tools/launchd .claude/skills/mission-control internal/mission` | empty. Including `cmd/ailang` gives only ifc/verify/trace/effects/bytecode commits (language-core, scope-guard refused) |
| Mark directives | `gh issue view 2172/2039 --json comments` filtered to MarkEdmondson1234 | none |
| Fleet drafts | `gh pr list --author @me` | fleet/i98..i82 drafts (#2194, 2173, 2171, 2169, 2165, 2150, 2143, 2141, 2110, 2108); no new diff-bearing PR |

Nothing changed since the iteration 98 idle verdict.

## Per-ticket gating
Tickets are identical to iteration 98, where all 27 were mapped to an OPEN row (D-FLEET-17..20, 22, 23, 25, 26, 28, 31..40, 44, 45). None has a mechanical, in-scope fix independent of a ruling: remaining fixes land in language-core paths (`cmd/ailang`, `internal/*`), prod/terraform/coordinator config, `.pi/extensions`, `tools/eval`, or are policy class (quota, routing, billing guard) per gate-2-pick.md "Policy class parks for Mark (HD-2a)". Since no harness commit landed on fleet paths and no ledger status flipped, there is no new fact that could ungate any of them.

## Per-row step screen
Rows with an existing proposed diff: D-FLEET-21 (A built), D-FLEET-26 (A built), D-FLEET-25 (#2165), D-FLEET-36/38 (scope-guard arm #2108), D-FLEET-18 (#2110), D-FLEET-42 (#2141/#2143/#2150). Remaining rows (16, 17, 19, 20, 22, 23, 27, 28, 29, 31..35, 37, 39..41, 43..45) either recommend a no-change/defer option, begin with a step outside fleet scope or needing an attributable human choice (credentials, prod config, policy thresholds), or are parked by quorum block (44, 45). I found no row whose recommended option (or shared non-default step) begins with an in-scope step that lacks a diff and needs no human choice; this matches the iteration 98 table and I did not find a row it misclassified (spot-checked via TSV default lines: all defaults are "C"/"B"/"A as shipped", i.e. status quo).

## Strongest objection
The screen leans on iteration 98's per-row classification for rows I did not individually re-derive (e.g. D-FLEET-29 "fleet Go fixes do not reach the loops", D-FLEET-41 executor image, D-FLEET-43 idle-policy). If one of those begins with a small in-scope tools/launchd step with no diff, it would qualify as a PICK under "screen by step". Mitigation: nothing in this fire's inputs differs from 98, which already had an independent judge reach NONE; a different verdict would need a new reading, not new facts, and an idle fire costs one cycle.

## Re-derivation (resume)
Read first-party: full ledger rows at `fleet-mission.md:172` (D-29), `:183` (D-41), `:185` (D-43), Authority allowlist at `:73-91`, idle rule at `:97`, `gh pr list --search` (no open PR covers any of the three), `.claude/skills/sprint-evaluator/SKILL.md` (no missing-toolchain rule; so D-41 B has no diff), `tools/launchd/lib/idle-fingerprint.sh:23` (still refuses fleet; no diff for D-43 A). Log 94 and 98 judges' reasoning was read, not adopted.

| Row | (a) recommended / default | (b) begins with an in-scope step with no diff? | (c) proceeds without a human choice? | Verdict |
|---|---|---|---|---|
| D-FLEET-29 | A: attended `make quick-install` on the rig / C | No. A is an operational action, not a diff, and not a fleet-path edit. The only code option, B (driver builds binary per fire, `tools/launchd`, in scope), is non-recommended, non-default, and not shared by the other non-default option (A). | No. Row says B "needs your ruling because it changes what every loop executes" (HD-2a policy-adjacent). | not buildable |
| D-FLEET-41 | A: attended session edits `Dockerfile.agent-go` (`docker/**`, outside Authority) / C | No. A's first step is out of scope and explicitly attended; no scope-guard arm precedes it. B (sprint-evaluator skill rule in `.claude` and `.agents`, in scope, no diff) is NOT recommended and no step is shared by A and B. | B's edit is mechanical, but the rule builds only the recommended option or a step shared by all non-default options. Selecting B instead is a choice between options, which is the human's. | not buildable (log 94 already ruled out "selecting D-FLEET-41 B") |
| D-FLEET-43 | A: lift fleet exclusion in `idle-fingerprint.sh:23` plus Gate 5 text (in scope; no diff) / B | The step is in scope with no diff, so (b) is YES. | (c) NO by the row's own text: "built as a proposed diff only if you choose A", default B. It also gates no ticket, and the Ticket-driven-only bullet forbids self-sourcing; the idle rule's "ticket is already in the queue" justification does not apply. | not buildable (matches iteration 94 and 98) |

Spot-checks of rows iteration 98 called out of scope or attended-only:
- D-FLEET-22: A = attended prod coordinator config (`triage_only_inboxes`) plus revision roll. Prod config is outside Authority, and no in-scope first step exists; B is a Daneel-side change and C is a no-op. Confirmed attended-only.
- D-FLEET-37: Recommendation A = V1/attended owns asks 1 and 2 (coordinator code, `internal/coordinator/daemon.go`); B widens scope to `internal/coordinator/**`, explicitly not recommended. The only fleet-side step (filing ask 2 as its own ticket) is conditioned on the human's confirmation. Confirmed out of scope.

Result: the 98 classification of all three target rows and both spot-checks holds. The strongest objection is closed for D-29 and D-41 (no recommended in-scope diff-less step) and for D-43 it reduces to the charter reading that D-43's own "only if you choose A" condition fails (c), which is exactly the idle rule's "can proceed without a human choice" qualifier. Residual risk: D-43 turns on that wording, not a charter sentence naming it.

VERDICT (unchanged): NONE (idle). Confidence: moderate-high.
