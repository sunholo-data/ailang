# Fleet mission dashboard — iteration 145

Updated 2026-11-27; scheduled unattended. [Bookkeeping #2445](https://github.com/sunholo-data/ailang/issues/2445).

- **Outcome: BUILT + JUDGED [HARNESS].** `idle-fingerprint:blind-to-nightly-rig-red` (stapledon 123) → [#2481](https://github.com/sunholo-data/ailang/pull/2481) at `ccee1caca`. The idle fingerprint now hashes open `nightly-rig` issues' `The nightly …` failure reports, so a red night wakes an idle product mission; loop triage notes do not. Independent judge (sonnet): **PASS 82 → 88**, 0 blocking, 27/27 mutants killed.
- **Dev: RED** at `a5bbd37c8`, required `checks`: two links the attended docs overhaul (`d03096c91`) left pointing at moved guides (`.claude/rules/coordinator.md`, `.claude/skills/local-ollama-eval/SKILL.md`). Outside fleet Authority; V1 owns dev reds, and V1's loop is disabled. #2481 and the record PRs carry the same inherited red; everything else on #2481 is green.
- **Open tickets:** 20 (one new). D-FLEET-44/45/46 gate the top ones; the rest are outside fleet Authority or waiting on dev.
- **Routing:** controller opus. Planner and executor codex `gpt-6.2-sol` (1 + 2 runs). Judge sonnet on the Agent tool, read-only (2 rounds).
- **Cost:** metered $0.
- **Parked on Mark:** D-FLEET-44, D-FLEET-45, D-FLEET-46 unchanged. D-FLEET-48 is moot; close it when convenient. Fixing the two context-doc links (or baselining them) would unblock #2478 and #2481.
- **Next:** once dev's `checks` is green, rebase and merge #2481, done-gate, resolve the ticket; land the record PRs.
- **Evidence:** [log](fleet-mission-log.md) and [iteration145/](fleet-mission-evidence/iteration145/).
