# V1 Mission Dashboard — snapshot, overwritten every iteration

**Updated**: 2026-09-04 (iteration 326) · **Release**: v0.35.0 · **dev**: GREEN

## Where the goal stands
**N = 12 design docs remaining before v1.0.0** — unmoved this iteration (HARNESS work).

## Last iteration (326) — LANDED [HARNESS]
dev was red on `lint`/`make fmt-check`: `646bda1e1` put four unformatted `internal/observatory`
files on origin/dev **15 seconds after commit**, via the Stop hook that auto-publishes dev. Root
cause is a shape, not a slip — `format_go.sh` (which does run `gofmt -w`) is wired as a PostToolUse
hook matching `Edit|Write`, so it fires for the Claude Edit/Write tools and for nothing else: not
for bash/sed edits, not at all for codex/pi executors. The publish step had no gate.
A **second instance landed mid-iteration** (`17a363ca6`, unsorted imports, auto-pushed 13 min later)
— which is the argument for gating the publish step rather than any one editing path.
Fixed both files; added a committed-blob gofmt gate to `push_dev_on_stop.sh` (18 test arms).
Judge sonnet PASS 86/100, zero blocking. `lint` now `success` on `c5227e6d7`.

## Next picks
1. `m-autopush-gate-followups` — five measured, non-blocking gaps in the new gate (start with the
   test harness polluting the real shared `autopush.log`; it is cheap).
2. `m-release-manager-skill-split` — the standing queue head: move the 18-image walkthrough out of
   `release-manager/SKILL.md` and ratchet `check-context-docs` back down 625 → 596.
3. `m-acceptance-criterion-green-at-base` — pre-registered; instance 2 is the Gate-5 skill-edit trigger.

## Loop cadence + routing
Controller `claude:claude-opus-5`. Executor `codex:gpt-5.6-sol` (probe rc=0 this fire).
Evaluator `sonnet`, always in its own worktree. Designer rotation pointer:
`pi:ollama/deepseek-v4-flash:0731-cloud` (untouched — no designer ran).
Spawn-pin hook is ARMED (`MISSION_CONTROL_ACTIVE=1`): an Agent spawn without `MISSION-ROLE:` is denied.

## Parked on Mark
**Nothing.** Decision ledger: 54 rows, **0 OPEN**. No directives outstanding on #972.

## Quota / cost posture
metered **$0.00** of the $5 iteration ceiling this fire. codex + sonnet are quota buckets; no quorum
ran (a dev-red fix-forward has no doc to review). Billing tripwire CLEAN.

## Standing sore spots
- **SonarCloud has been red on dev for several consecutive commits** and is unowned. Non-required,
  so nothing blocks on it — which is exactly how a required check eventually gets missed.
- `m-ci-serial-gate-masking`: one early red in a long sequential job hides every gate behind it
  (45 gates on 2026-09-03, 21 again on 2026-09-04). Second instance in three days.
