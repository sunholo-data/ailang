# Fleet mission dashboard — iteration 122

Updated 2026-11-18; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED and RESOLVED [HARNESS].** Rows 51/53 (D-FLEET-36 A and D-FLEET-38 A) are done.
  - [#2419](https://github.com/sunholo-data/ailang/pull/2419) → `d2f2ec238`. The 8 `\d` greps in design-doc-creator are now `[0-9]`. The sync list gained exactly the 16 named skills, and 14 `.agents` mirrors were regenerated (2 were already identical).
  - A fresh judge (sonnet) scored **PASS 92**, 0 blocking, on top of iteration 121's PASS 93. PR CI 24/24 green, dev CI 18/18 green.
  - Done-gate: fleet dry-runs under a synthetic HOME, healthy and degraded, both rc 0, pins unchanged. `make test-launchd-drivers` 58/58.
  - Both tickets were resolved with the merge SHA.
- **Dev:** green at `d2f2ec238`.
- **#2411 (row 41)** still needs the attended `make pi-assets` commit.
- **Tickets:** 19 open (was 21). Two are new from stapledon and untriaged (rows 65/66):
  - `quorum:codex-gpt-6.2-sol-not-allowlisted`
  - `gate0:self-notices-rc2-no-watermark-file`
- **Routing:** controller opus; judge sonnet (Agent tool, own worktree). Designer, planner and executor were not spawned, because the work was carried from iteration 121.
- **Codex lanes:** every `gpt-6.2-sol` probe fails `model_not_found` ("not supported when using Codex with a ChatGPT account"), so planner falls back to opus and executor to haiku. This is a routing-policy question for Mark.
- **Cost:** metered $0.
- **Records 95–122** sit on stacked draft PRs (… → #2417 → this fire). The stack's base is draft #2171, a design parked for human review.
- **Gate 0 self-notice:** watermark form rc 2, for the twenty-sixth fire running. This is now ticketed as row 66.
- **Parked on Mark:** D-FLEET-44 and D-FLEET-45 (unchanged).
- **Observations for Mark:**
  - The codex `model_not_found` failure above.
  - `sprint-planner/scripts/create_sprint_json.sh:204` has the same `\d` bug, unticketed.
  - `release-manager` is in the sync list but refused by the guard.
- **Next:** rows 65/66 by `slots_lost`. Row 65 may be policy class (HD-2a). Row 66 is a mechanical candidate in harness scope. Then #2411 after the attended sync. Otherwise D-42's fleet half.
- **Evidence:** [log](fleet-mission-log.md) and [iteration122/](fleet-mission-evidence/iteration122/).
