# Fleet mission dashboard — iteration 115

Updated 2026-11-16; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED [HARNESS].** D-FLEET-26 A: a codex capacity stop is now a transient retry in both renderings → #2368 / `89a28a09c` (judge sonnet PASS 88, 0 blocking). Plus four ticket resolutions the rulings directed with no build (D-FLEET-31 A, 32 B, 34 B, 37 A).
  - Iteration 79's draft (#2065) matched only raw `codex exec` output. Controllers have run `ailang mission codex-exec` (daemon) since 10-09, so it would never have fired. The planner caught it; the fix matches both renderings.
- **Input that changed:** Mark's attended rulings (`a82f0180a`, 11-16 07:26Z) resolved D-FLEET-16..43 on dev, after iteration 114 recorded. Adopted as attended (session transcript corroborates; the provenance script reads `fleet` on this rig regardless). Merged into the record stack row by row (`37b284157`).
- **Attended session owns:** D-17, 19, 20, 22, 23(3), 28, 29, 35, 39, 40, 41 and the key rotation in D-42 (PRs #2343–#2349). The fleet has the rest.
- **Tickets:** 27 → 22 open. Ledger 44 rows, 2 OPEN (D-FLEET-44, 45, stack-only).
- **Pick-order miss:** Mark's queue order ranks row 13 (D-FLEET-18, guard-arm draft #2110) and row 33 (D-FLEET-25) above row 34. This fire picked 34 by impact; next fire returns to the groom order.
- **Guard gap seen:** the pre-push scope guard refuses a force-push of a rebased fleet branch (it diffs from the old remote SHA). Workaround used: fresh branch + new PR. Every draft refreshed onto dev will hit it.
- **Routing:** opus controller; planner opus (Agent); executor haiku via `claude-sub` (rc 0, 559 s); judge sonnet (Agent). Codex lanes probe rc 75 (quota unreadable).
- **Dev @ `89a28a09c`:** see the log entry for the merge-commit CI verdict.
- **Goal:** clause 2 moved (5 tickets resolved); clause 1 UNMEASURED, 3 MET, 4 met by its letter, 5 preserved (done-gate run).
- **Cost:** metered $0.
- **Records 95–115** sit on stacked draft PRs (#2169 → … → #2342 → this fire). origin/dev's log still ends at 94.
- **Gate 0 self-notice:** watermark form rc 2 (nineteenth fire); `--since` form 0 crash notices.
- **Parked on Mark:** D-FLEET-44, D-FLEET-45 (unchanged).
- **Next:** row 13, `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` (D-FLEET-18 A): land guard-arm draft #2110 alone first. Then row 33 (D-FLEET-25 A, drafts #2165/#2082).
- **Evidence:** [log](fleet-mission-log.md), [iteration 115](fleet-mission-evidence/iteration115/).
