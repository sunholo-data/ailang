
## 77 — 2026-11-02 — `agent-tool:workspace-routing-discovery-timeout` folds into D-FLEET-24 (one codex-internal transport error, recovered by `followup_task`; none in 652 later sessions); the fold measures option B's premise once; no other ticket routable; weekly sweep 0 items; independent sonnet JUDGE_VERDICT [HARNESS]

**Pick**: iteration 76's **Next**, `agent-tool:workspace-routing-discovery-timeout` (stapledon iteration 9, 10-02, 1 slot, `blocking=none`). It was the one open ticket with neither a queue row nor a decision row. It measured as a symptom of D-FLEET-24, so it was a bookkeeping pick. No second item was taken (standing rule 1 allows one): all 32 open tickets now have a queue row or an OPEN decision row, and row 16's trigger (`quorum:tier2-salvage-fold`, "its first non-test caller") has not fired. `git grep RunQuorumWithEscalation -- '*.go'` outside tests finds only its definition (`internal/mission/quorum/quorum.go:100,112`); the control, the same grep including tests, finds 9.

**Orphans / prior record**: no open `fleet/*` PR (iteration 76's #2038 merged → `ffabe3278`). No `fleet/i77*` branch or worktree before this fire. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (19 comments, none allowlisted; the watermark is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to `origin/dev` in both the resolved symlink target and the pin copy. The pin is at `origin/dev` (`AILANG_DRIVER_AGE=0`).
- Gate-1 base `ffabe32784b2cf44c0bca57e723e60de46a7a905@2026-11-02T07:58:06Z`.
- Dev CI at `ffabe3278`: `CI` and `Build and Release` success. 22 checks, one not green: the inherited `ailang-core-dev (ailang-multivac-deploy)` Cloud Build red. Not required, outside fleet scope, V1-owned; recorded, not picked.
- Ledger valid, 40 rows, 25 OPEN. 32 tickets open. The newest filing is 10-28.
- Codex is over its daily ration (`MISSION_OVER_RATION=codex`). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing; [census](fleet-mission-evidence/iteration77/census.md)):
- **Harness surface**: `git grep -i 'workspace routing\|routing discovery'` at `ffabe3278` finds no harness file (2 unrelated v0.9.0 design docs). 0 hits in the 14 `/tmp/ailang-mission-*.log` driver logs.
- **Census**: codex session store `~/.codex/sessions/2026/10`, 1,049 files (10-01..10-30; no `2026/11` directory). Three wording variants match the same 27 files. Most of the 65 matching records quote the ticket. The occurrences are: (1) 10-02, stapledon iteration 9: the `evaluator_transport` sub-agent's turn ended `workspace routing discovery timed out` (`codex_error_info: other`, `codex_exec` 0.159.2), and the parent received it as a FINAL_ANSWER. (2) 10-09: a `workspace routing discovery unauthorized (401)` variant in 3 attended `codex-tui` 0.162.0 stapledon sessions, which is not a mission fire and not ticketed. After 10-09 there are 652 files, 39 of which made 107 `spawn_agent` calls, and none has an occurrence.
- **The chain**: the controller's `spawn_agent(model: "sonnet")` returned ``Unknown model `sonnet` for spawn_agent`` (08:18:16Z; paired by `call_id`). The controller then spawned `evaluator_transport` (`gpt-6.1-sol`), which ran `claude-sub --model claude-sonnet-5-5 … --print` at 08:49:56Z. The codex turn errored at 09:03:15Z, `followup_task` followed at 09:04:32Z, and the turn completed at 09:19:56Z with "PASS, 91/100".
- **Option B's premise**: both sessions' `turn_context` reads `sandbox_policy: danger-full-access`, `approval_policy: never`, which is the driver's launch (`tools/launchd/mission-control.sh:2431`, `--dangerously-bypass-approvals-and-sandbox`). D-FLEET-24 B said `claude-sub` inside the codex sandbox was unmeasured. There is now one reading on the launch the driver actually uses.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned, because nothing in fleet scope needed building. This is not a spawn failure. Resolver output: designer `recipe claude:claude-opus-5-5 declared:provider-pin`, planner `agent-tool opus fail-closed:env-pin`, executor `recipe claude:claude-haiku-5-5 declared:provider-pin`. Rotation pointer not advanced.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only; resolver `agent-tool sonnet declared:alias-pin`. Given the draft records and told to refute them. JUDGE_DETAIL
- **Recorded**: queue row 59 (`[PARKED · D-FLEET-24]`); row 31 points to it; D-FLEET-24 gains an iteration-77 re-measure (7 tickets, 25 of 63 lost slots; B's premise measured once; recommendation A unchanged). Ticket stays open. 32 tickets open.
- **Weekly external-issue sweep** (first fire after the Monday rotation; the last was iteration 61, 10-26): 57 open issues enumerated, matching the search total of 57. 55 have zero mentions across the charter, log, log archive, status archive and dashboard. The positive control `#1306` hit 13 times, and the negative control (a fresh, unpublished literal) fired. The 7 new since 10-26 are `motoko_agent` `std/process` ×4 (#2034–#2037), `stapledons_godot` language ×2 (#2029, #1980) and triage's bookkeeping issue (#1974). All are outside Authority, so 0 queue items. Table: [issue-sweep.tsv](fleet-mission-evidence/iteration77/issue-sweep.tsv).
- **Done-gate**: not applicable (no harness change).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no ticket filed since 10-28; this one was filed 10-02 and stays open). Clause 3 MET (32 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `ffabe32784b2cf44c0bca57e723e60de46a7a905@2026-11-02T07:58:06Z`. Routing note (driver): `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: JUDGE_TOKENS. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- A separate decision row: the transport exists only because of D-FLEET-24's bare-alias spawn, and options A and B both remove it.
- Resolving the ticket as "native, not ours": resolve needs a `--sha` on `origin/dev`, and the fix that removes the exposure is D-FLEET-24's ruling, which has not been made.
- Ticketing the 10-09 `401` variant: attended `codex-tui` sessions, not a mission fire, and no product loop filed it.

**Retro lane**: no skill edit.
- (a) Iteration 76 named "a log census is only as wide as its wording" as a candidate Gate-2 rule. This fire ran three wording variants and they agreed (27 files each). The census's real hazard here was the opposite one: **a census over agent transcripts counts quotes of the ticket as occurrences** (27 files matched, 2 sessions held the event). The two share a root: a match is not an event until the record type says so. Still not edited, because a rule edit is live for every mission; it is flagged in the report alongside (a) from iteration 76.
- (b) Eighth consecutive fire (70 was the last code landing) without a landable pick. This is the first fire where no open ticket lacks a disposition: all 32 have a queue row or an OPEN decision row, and none can be built until a ruling lands. D-FLEET-24 alone gates 7 tickets and 25 of 63 lost slots. Until rulings land, each fire can only re-measure. The report asks Mark to rank the rulings rather than adding a new decision row.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71–77. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: no routable ticket. Check `ailang mission ticket open --json` for a filing newer than 10-28, and the ledger for a ruling (first D-FLEET-24, then -18, -36, -38, -41). Take whichever lands first. If neither has, record a short no-op fire rather than re-measuring a parked ticket.
