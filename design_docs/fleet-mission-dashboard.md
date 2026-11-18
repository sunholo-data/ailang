# Fleet mission dashboard — iteration 120

Updated 2026-11-18; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: BUILT, NOT LANDED [HARNESS].** Rows 51/53 (D-FLEET-36 A and D-FLEET-38 A): the scope-guard arm admits `design-doc-creator` (both mirrors) and exactly the 16 named `.agents` skill mirrors. It is on [#2413](https://github.com/sunholo-data/ailang/pull/2413), judged sonnet PASS 93 with 0 blocking, and supersedes draft #2108.
  - Rebuilt fresh, because #2382 had touched the same files. A new test loop keeps every other `.agents` skill refused; #2108's test let a 17th through.
  - Mutants killed: planner 17/17, executor 3/3, controller 2/2, judge 12/12. `make test-launchd-drivers` 58/58 twice.
  - Premise at HEAD: 14 of the 16 are rename-only, and 2 are now byte-identical. All 16 kept, per the ruling.
- **Blocked by dev's red:** `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` at `022f68f1a` (attended `8e04a29c5`; it needs `"gpt-6.1-sol": "gpt-6.2-sol",` in `ratifiedRoleMigrations`). This is outside fleet scope and was handed to the `user` inbox at iteration 119. It blocks #2413, #2411 and the record stack.
- **#2411 (row 41)** still also needs the attended `make pi-assets` commit on `fleet/i119-unowned-dirty`.
- **Tickets:** 19 open (unchanged; rows 41/51/53 resolve after landing).
- **Routing:**
  - Controller: opus.
  - Designer: not spawned (pre-ruled diff).
  - Planner: opus (Agent).
  - Executor: haiku via `claude-sub`.
  - Judge: sonnet (Agent, own worktree).
  - Codex lanes: probe rc 75 (quota unreadable).
- **Cost:** metered $0.
- **Records 95–120** sit on stacked draft PRs (… → #2412 → this fire).
- **Gate 0 self-notice:** watermark form rc 2 (twenty-fourth fire).
- **Parked on Mark:** D-FLEET-44 and D-FLEET-45 (unchanged). New observation: `release-manager` is in `sync-agents-skills.sh`, but the fleet guard refuses its `.agents` path.
- **Next:** once dev is green, land #2413 (then retire draft #2108) and #2411. Then push the 8-site `[0-9]` edit and the sync-list extension on a fresh branch, and resolve both tickets after the done-gate. Otherwise D-42's fleet half (#2141–#2150).
- **Evidence:** [log](fleet-mission-log.md) and [iteration120/](fleet-mission-evidence/iteration120/).
