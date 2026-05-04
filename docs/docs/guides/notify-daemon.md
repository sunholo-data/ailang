---
sidebar_label: Notification Daemon
title: macOS Notification Daemon
---

# macOS Notification Daemon

`ailang daemon` is a long-running process that pulls cloud events from Google Pub/Sub and surfaces them as native macOS notifications. Its primary jobs:

- **Public feedback** submitted via `mcp.ailang.sunholo.com` lands in the `public-feedback` inbox and pings you in real time (🌐 External feedback).
- **Coordinator tasks** that need approval (⏳), complete (✅), or fail (❌) ping the maintainer immediately.
- **Inbox messages** addressed to the laptop user surface as ✉️ notifications.

Without the daemon, the only between-session pickup for these events is the `check_public_feedback` SessionStart hook, which fires only when you start a new Claude Code session.

## Quick start

```bash
# One-time install: creates ~/Library/LaunchAgents/com.sunholo.ailang.daemon.plist
# and launchctl-loads it (auto-starts on login + restarts on crash).
make install-notify-daemon ENV=prod

# Verify it's running
ailang daemon status

# End-to-end smoke test (publishes a synthetic event to dev, waits for notification)
make smoke-notify-daemon

# Stop / remove
make uninstall-notify-daemon
```

For dev/test environments, pass `ENV=dev` or `ENV=test`.

## Subcommands

| Command | Purpose |
|---|---|
| `ailang daemon run [--env] [--dry-run]` | Foreground mode (used by launchd `ProgramArguments`). `--dry-run` logs notifications without firing them; useful for verification. |
| `ailang daemon install [--env] [--binary] [--force]` | Install plist + launchctl load. Refuses overwrite without `--force`. |
| `ailang daemon uninstall` | launchctl unload + remove plist. Idempotent. |
| `ailang daemon status` | Print launchctl state + last 10 log lines. |

The default subcommand is `run`, so `ailang daemon` alone starts the foreground loop.

## What gets notified

| Event | Trigger | Title |
|---|---|---|
| Task pending approval | `TaskStreamEvent.status = "pending_approval"` | ⏳ Approval needed |
| Task completed | `TaskStreamEvent.status = "completed"` | ✅ Task done |
| Task failed | `TaskStreamEvent.status = "failed"` | ❌ Task failed |
| Public feedback | InboxMessage with `to_inbox = "public-feedback"` | 🌐 External feedback |
| Other inbox message | Any other InboxMessage notification | ✉️ Message from `<from_agent>` |

Other Pub/Sub events (intermediate task states like `running` or `queued`) are silently dropped.

Click-actions on macOS notifications open the AILANG dashboard root (requires `terminal-notifier`; the `osascript` fallback path has no click-action support). Deep links to specific tasks/inboxes are intentionally not used — the dashboard is currently a single-page app without client-side routes for `/tasks/<id>` or `/inbox/<name>`, so deep links would 404.

## Known limitations

- **Notification icon is `terminal-notifier`'s default**, not the AILANG logo. macOS binds notification identity to the bundle that posted the notification; `terminal-notifier`'s `-appIcon` flag is ignored on recent macOS versions. The only reliable fix is to ship a signed `AILANG.app` bundle registered as a notification source, which is out of scope here.
- **launchd's default `PATH` excludes Homebrew prefixes.** Without our explicit `PATH` entry in the plist, the daemon would silently fall back to `osascript` (no click-action support) because `exec.LookPath("terminal-notifier")` would fail. The plist sets `PATH=/opt/homebrew/bin:/usr/local/bin:...` to fix this.

## Dedup

The daemon suppresses repeats within a sliding window:

- **Tasks**: 60s on `(task_id, status)` — prevents the same task event from re-notifying if Pub/Sub redelivers.
- **Messages**: 5min on `message_id`.

## Configuration

`~/.ailang/config/daemon.yaml` is optional. Every field has a sensible default:

```yaml
env: prod                  # dev|test|prod (default: prod)
inbox: ""                  # reserved for per-inbox routing (no effect today)
task_window_sec: 60        # task event dedup window
msg_window_sec: 300        # message event dedup window (5 min)
dry_run: false             # skip notifier; log instead
excludes_path: ""          # override path to notify_excludes.conf
```

`~/.ailang/config/notify_excludes.conf` (optional): one substring per line; matched against title and body. Lines starting with `#` are comments. Useful for muting noisy event categories without editing code.

## Logs and troubleshooting

The daemon writes stdout+stderr to `/tmp/ailang-daemon.log`. To tail:

```bash
tail -f /tmp/ailang-daemon.log
```

`ailang daemon status` shows the last 10 log lines too, plus the launchctl dictionary (PID, last exit status, etc.).

If notifications don't appear:

1. Verify launchd is running it: `launchctl list | grep com.sunholo.ailang.daemon`. The first column is the PID; if it's `-`, the process exited and is being respawned.
2. Check the log for subscription errors. `Resource not found` means the Pub/Sub subscription doesn't exist for the env you selected — verify the `client_subscriptions` block in the relevant `ailang-multivac/terraform/environments/<env>/terraform.tfvars`.
3. Ensure GCP ADC is configured: `gcloud auth application-default login`. The daemon authenticates the same way `gcloud` does.
4. Confirm `terminal-notifier` is installed (`brew install terminal-notifier`) for click-actions; otherwise the daemon falls back to `osascript display notification` (no click target).
5. macOS notification delivery itself can be muted by Focus modes — check **System Settings → Notifications → ailang** and **Focus → Notifications**.

## Relationship to the SessionStart hook

`scripts/hooks/check_public_feedback.sh` was the previous, pull-based pickup mechanism: it ran at session start and surfaced any unread public-feedback. The daemon supersedes it for between-session real-time delivery; the SessionStart hook was removed when this daemon shipped.

## Non-goals

- Windows / Linux notifications (macOS only).
- Notification history or read receipts.
- Interactive approval directly from a notification (phase 2 idea — would use `terminal-notifier`'s reply action).
