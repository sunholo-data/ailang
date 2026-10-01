# Rig binary rebuild+reinstall deferred to attended session — not executable from the triage agent

- **Date**: 2026-10-01
- **Class**: not-actionable
- **Recommend**: drop
- **Searched**: `rig`, `reinstall`, `quiet window`, `binary drift`, `messages-sub` across `design_docs/` and `docs/internal/`; cross-checked `daemon-liveness-binary-rebuild.md`, `nightly-eval-rig-lock-yield.md`
- **Estimate**: n/a (drop — no code change warranted)

## Why

This is an attended ops runbook, not an AILANG engineering report, and both of its preconditions
are unmeetable from this agent (an unattended Cloud Run container, `AILANG_TASK_ID` set, Linux,
no `gcloud`/ssh path to the rig):

1. **No rig access.** The procedure operates on `/Users/voightkampff/...` and `~/go/bin/ailang`
   on the Mac Studio; this container has no macOS users, no `~/go/bin/ailang`, and no remote
   execution route to the rig.
2. **No quiet window.** The task's own precondition ("do not install while missions run; 8
   mission/eval processes were live") requires observing rig process state at install time, and
   the deferral itself is marked "Mark attended" — i.e. deliberately queued for a human-attended
   session, which an unattended coordinator run cannot substitute for.

No design doc is warranted, because there is nothing left to decide — the engineering is done
and the procedure is already documented:

- **Both fixes are landed upstream.** Verified in this checkout: the poison-notification fix at
  `internal/daemon/daemon.go` (`"unresolvable for %s … acking so it stops blocking the
  subscription"`) and the Secret Manager webhook resolution at `internal/notify/secret_gcp.go`
  (`DiscordWebhookSecretSuffix = "discord-webhook-url"`).
- **`--messages-sub` on `daemon install` exists** (`cmd/ailang/daemon.go`, primary-subscription
  flag added 2026-09-30), so the plist regeneration step in the runbook is valid as written.
- **The exact reinstall command is already ruled on** in
  `docs/internal/message-plane-topology.md` ("The delivery layer: Pub/Sub subscriptions",
  measured on the rig 2026-09-11): `ailang daemon install --env prod --force
  --messages-sub messages-rig`, plus the trap that a hand-edited plist is silently reverted by
  the next `--force` install — reinstall, never re-edit. The build-from-temp-worktree route in
  the runbook (don't `git pull` a checkout with 7 uncommitted files) is the same
  `make quick-install`-avoidance control documented in trap 6 of the same doc.

## What makes it actionable

An attended shell on the Mac Studio during a quiet window (no mission/eval processes), then
execute the runbook verbatim: temp worktree at `origin/dev` → build → install →
`ailang daemon install --env prod --force --messages-sub messages-rig` → verify daemon log shows
`notify: discord channel registered (webhook from Secret Manager)` and `channels=[discord macos]`
→ remove the worktree. None of that belongs to core triage; it should ride with Mark's next
attended rig session as already deferred (2026-10-01).
