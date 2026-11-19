# Fleet mission dashboard — iteration 123

Updated 2026-11-19; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED and RESOLVED [HARNESS].** Row 66 `gate0:self-notices-rc2-no-watermark-file` is done.
  - [#2421](https://github.com/sunholo-data/ailang/pull/2421) → `cf540337e`. When every watermark file is absent, Gate 0 step 6a now does a DEGRADED whole-issue read instead of exiting rc 2. A present-but-invalid file is still rc 2. Root cause: since #981 the file is created only on `--mark`.
  - Independent judge (sonnet) **PASS 91**, 0 blocking, 11/11 mutants. PR CI 24/24 green, dev CI 18/18 green.
  - Done-gate: world dry-runs healthy and degraded, both rc 0, pins unchanged. `make test-launchd-drivers` 58/58 at the merge.
  - Resolved about 14 h after filing. Both stapledon and world had filed it.
- **Dev:** green at `cf540337e`.
- **Hidden crash surfaced:** stapledon's `-prev` issue #410 has a 2026-11-13 crash notice the floor had hidden. Stapledon has been told in the resolve reply.
- **Tickets:** 18 open (was 19).
  - Row 65 `quorum:codex-gpt-6.2-sol-not-allowlisted`: the fix is out of fleet scope and the lane question is HD-2a.
  - Seven tickets whose attended fixes merged on 11-16 (#2343, #2347, #2348, #2349) are still open. They were the attended session's claim. New ask **D-FLEET-46**: may the fleet resolve them?
- **Routing:** controller opus. Planner opus, executor haiku and judge sonnet were all Agent-tool spawns, foreground. The designer was not needed for a mechanical fix.
- **Codex lanes:** every `gpt-6.2-sol` probe still fails, so planner falls back to opus and executor to haiku.
- **Cost:** metered $0. Planner 152,523 tok, executor 90,778, judge 66,764.
- **Records 95–123** sit on stacked draft PRs (… → #2420 → this fire).
- **Parked on Mark:** D-FLEET-46 (new), D-FLEET-44 and D-FLEET-45 (unchanged).
- **Next:** row 65 (re-measure, likely a park), then #2411 after the attended `make pi-assets`. Otherwise D-42's fleet half.
- **Evidence:** [log](fleet-mission-log.md) and [iteration123/](fleet-mission-evidence/iteration123/).
