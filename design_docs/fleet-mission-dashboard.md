# Fleet mission dashboard — iteration 118

Updated 2026-11-17; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: LANDED [HARNESS].** Row 33, D-FLEET-25 A, is built end to end.
  - World-fork text: #2396 → `ad1fdac57` (judge PASS 93). Draft #2082 is closed.
  - Fleet-wide rulings ledger `design_docs/fleet-rulings.md`, plus a report-only `ailang mission doctor` check: #2402 → `031941ada` (build judge PASS 91). Draft #2165 is closed.
- **How the mechanism works:**
  - Forbid-only literal detectors over three scopes (registry, mission skills, charters), with explicit `supersedes` between rulings.
  - Every finding is a `note`, so the exit code, lane-check and arming are unchanged.
  - Notes print first, so lane-check's last-line cell keeps the drift reason.
  - Seeded with D-FR-1 (World is not a fork) and D-FR-2 (generator≠judge is a preference), both transcribed with their sources.
- **Design path:**
  - The GLM R2 draft was rewritten by the opus designer: 698 → 309 lines, with the provenance binding dropped.
  - Quorum R3 blocked 0/3 on unverified citations. R4 blocked 1/3 on one narrow surface; both rejects were measured as non-defects and their fixes applied verbatim (R4.1 carve-out).
  - The planner caught one false design claim (note ordering vs lane-check). The plan judge gave PASS 82.
- **Build:** haiku did M1–M7 plus a Windows-path fix round. Mutants: 22/22 executor, 6/6 controller, 10/14 judge (survivors non-blocking).
- **Tickets:** 20 → 19 open; `policy:fleet-rulings-not-propagated-to-charters-and-skill` is resolved.
- **Attended follow-up:** link lines in the world and stapledon charters, which live in other repos. Doctor reports `ruling-link-missing` for both.
- **Rig observation:** doctor shows `env-source-drift` on all 6 missions (`AILANG_CODEX_RUNTIME`, `AILANG_DRIVER_REF`, `CODEX_HOME`), so it exits 1 today. Pre-existing and not ticketed.
- **Routing:**
  - opus controller; designer opus via `claude-sub`; planner opus (Agent); executor haiku via `claude-sub`; judge sonnet (Agent, own worktrees).
  - Codex lanes: probe rc 75 (quota unreadable).
- **Cost:** metered $0 inner loop. Quorum self-reported $0.54.
- **Records 95–118** sit on stacked draft PRs (#2169 → … → #2386 → this fire).
- **Gate 0 self-notice:** watermark form rc 2 (twenty-second fire).
- **Parked on Mark:** D-FLEET-44, D-FLEET-45 (unchanged).
- **Next:** row 41 (D-33 A), rows 51/53 (D-36/38 A, #2108), D-42's fleet half (#2141–#2150).
- **Evidence:** [log](fleet-mission-log.md), [iteration 118](fleet-mission-evidence/iteration118/).
