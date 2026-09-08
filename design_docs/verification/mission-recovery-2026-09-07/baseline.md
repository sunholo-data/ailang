# Mission fleet baseline — attended 2026-09-07

This is a dated observation, not a live dashboard. Canonical inbox: 20 unread, inspected via
list --unread --json without acknowledgement. Threads and local slot logs can advance independently.

## Current reporting destinations

| Mission | Active pointer / open thread | Note |
|---|---|---|
| V1 | mission-v1-gh-issue → [ailang#1072](https://github.com/sunholo-data/ailang/issues/1072) | week of September 7; bare mission-gh-issue=745 is stale and no longer the driver input |
| World | mission-world-gh-issue → [world#129](https://github.com/sunholo-data/ailang-world/issues/129) | week of September 7 |
| Motoko | mission-motoko-gh-issue → [ailang#1078](https://github.com/sunholo-data/ailang/issues/1078) | week of September 7 |
| Docs | mission-docs-gh-issue → [ailang#979](https://github.com/sunholo-data/ailang/issues/979) | still titled week of August 31; open at inspection, do not force rotation |

No pointers or directive watermarks were changed. Agent comments are feedback, not human directives.

## V1 failure classification

Read the matching termination and immediately preceding error in `/tmp/ailang-mission-control.log`.
Driver timestamps are local CEST; slot-verdict log timestamps below are UTC.

| Slot termination (UTC) | Observed cause | Recovery implication |
|---|---|---|
| September 6 23:25:16 | Codex usage-limit errors immediately before exit 1 at gate 3 | capacity exhausted mid-work; partial artifacts need inspection |
| September 7 01:27:24 | watchdog: no progress over five samples, descendant alive ≥2400s; Pi killed at gate 2 | stalled worker, not proof of quota failure; fence and reconcile |
| September 7 02:11:44 | Ollama HTTP 429 session usage limit immediately before exit 1 at gate 3 | fallback capacity also exhausted; a successful probe was insufficient |
| September 7 04:27:16 | same watchdog pattern, Pi killed at gate 3b | potentially completed external effects; no automatic rerun |
| September 7 06:08:26 | completed with Codex controller | loop termination only; consult accepted artifact evidence |

The last six sampled V1 slots contained four failures. This tail is not a seven-day fleet rate.
World last completed at 07:55:59 UTC; Docs last four recorded slots completed; Motoko completed
again at 09:00:05 UTC. A completion can bank a parked decision and make no product progress.

## Concrete hazards and owners

- **Wrong repository:** World #129 / fleet [#1075](https://github.com/sunholo-data/ailang/issues/1075)
  confirm unconditional pin workdir replacement. Installed AILANG_DRIVER_PIN=0 is intentional
  mitigation. The fixed helper must reach the configured ref before a controlled re-enable.
- **Baseline CI:** V1 already owns [#1074](https://github.com/sunholo-data/ailang/pull/1074), the
  inherited exec.go 807-line violation. Do not duplicate its extraction. V1 also owns #1073 /
  D-60 notification recovery; its seven shell-test failures are separate from this slice's Go tests.
- **Misleading weekly health:** World and Motoko reports say “Nothing parked on a human decision”
  while their thread bodies/digests list open decisions. Treat the charter decision ledger as
  authority; a future report fix must distinguish a missing measurement from zero blockers.
- **Role routing:** existing Agent-tool/provider-pin contradiction remains in the old driver.
  New role-run is opt-in, not yet wired as its replacement. It must not be claimed as a live fix.
- **Quota:** ledger updated 06:02:38 UTC, 10,371 tokens over two opencode stages; long-window
  capacity unknown and unrationed. This is measurement warmup, not successful capacity admission.
- **Shared STATUS heading contract:** Motoko reports rotation from an unlinted charter into a
  linted archive causes a known shape to become invalid (row 17). Existing mission owners retain
  the fix; avoid widening the lint constant or editing their active work.

## Rollout order

1. Land reviewed pin fix in the ref actually read by the live driver. Preserve World mitigation
   until that is verified. Same-repo and cross-repo synthetic re-exec must pass first.
2. Controlled World fire: record driver SHA, mission origin and charter reachability; only then
   accept restoration. Do not claim a stopped/restarted service or a fixture as a successful fire.
3. Exercise opt-in durable role dispatch with an isolated worktree/explicit state DB. Verify
   receipt and DB agree, duplicate delivery refuses, expired running work is ambiguous.
4. Add artifact acceptance and reconciliation resolution, then quota reservations before
   automatic mission stage advancement. Only then migrate one mission and expand onboarding.

No live worker was restarted, no model was probed/invoked, and no human decision was resolved
by this recovery increment. Message posting awaits explicit user authorization.
