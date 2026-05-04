# M-MAC-NOTIFY-DAEMON: Mac Notification Daemon for Cloud Events

**Status**: Planned
**Target**: v0.15.0
**Priority**: P1 — required for human-in-the-loop approval workflow on laptop and real-time external feedback (`public-feedback` inbox)
**Estimated**: 2-3 days
**Source**: Need to know when cloud agents need approval, when tasks complete, or when external agents file public feedback via the MCP `submit_feedback` tool. Pulled forward from v1.0.0 to v0.15.0 after motoko_agent integration surfaced the need for real-time public-feedback notification (the SessionStart hook in `scripts/hooks/check_public_feedback.sh` only fires at session start; this daemon adds between-session pings).
**Supersedes**: `M5: Minimum-viable feedback notifier` carve-out in [m-pkg-feedback-loop.md](m-pkg-feedback-loop.md)

---

## Problem

The cloud coordinator runs agents asynchronously. A human needs to:
1. Know when a task requires their approval (design-approved, sprint-approved, merge-approved labels)
2. Know when an agent completes work (success or failure)
3. Know when a message arrives in their inbox from an agent

Currently the only way to see this is to open the dashboard or run `ailang messages watch --pubsub` manually. There is no background notification — the human must poll.

The local Claude Code session hooks (`session_end_speak.sh`) already demonstrate the right pattern: fire a macOS notification via `osascript`/`terminal-notifier` at key moments. We need the same pattern for cloud events.

---

## Solution: `ailang daemon` Subcommand + launchd Plist

A persistent background process (`ailang daemon`) that:
1. Pulls from the `events-laptop` Pub/Sub subscription continuously
2. Parses event types and fires macOS notifications for actionable events
3. Is managed by `launchd` so it starts on login and restarts on crash

### Events to surface as notifications

| Event type | Notification trigger | Title | Body |
|-----------|---------------------|-------|------|
| `TaskStreamEvent{status: "pending_approval"}` | Task waiting for human approval | "⏳ Approval needed" | `{agent}: {task_title}` |
| `TaskStreamEvent{status: "completed"}` | Task finished successfully | "✅ Task done" | `{agent}: {task_title} ({turns} turns, ${cost})` |
| `TaskStreamEvent{status: "failed"}` | Task failed | "❌ Task failed" | `{agent}: {task_title} — {error}` |
| `InboxMessage` on `messages-laptop` sub | New message from agent | "✉️ Message from {from}" | `{title}: {preview}` |
| `InboxMessage` with `to_inbox=public-feedback` | External feedback via MCP | "🌐 External feedback" | `[{category}] {title}` (click → Firestore doc URL) |
| Package published to registry | Registry update | "📦 Package published" | `{vendor}/{name} v{version}` |

### Click action

Using `terminal-notifier`, clicking the notification opens the dashboard at the relevant task URL:
```
-execute "open 'https://dashboard.ailang.sunholo.com/tasks/{task_id}'"
```

---

## Implementation

### 1. `ailang daemon` command (`cmd/ailang/daemon.go`)

```go
// ailang daemon [--project PROJECT] [--inbox INBOX] [--env dev|test|prod]
//
// Pulls from events-laptop and messages-laptop Pub/Sub subscriptions.
// Fires macOS notifications for actionable events.
// Runs forever; designed to be managed by launchd.
```

Key behaviours:
- Pulls both `SubEventsLaptop` and `SubMessagesLaptop` in parallel goroutines
- Ack only after notification fires (prevents loss on crash)
- Deduplicates: same task_id + status combo within 60s is not re-notified
- Respects `~/.ailang/config/notify_excludes.conf` (same pattern as hook_excludes)
- `--dry-run` flag logs notifications without firing osascript (for testing)

### 2. Notification function (`internal/notify/macos.go`)

Reuses the `terminal-notifier`/`osascript` pattern from `session_end_speak.sh`:

```go
func Notify(n Notification) error {
    // Try terminal-notifier first (supports click actions)
    // Fall back to osascript display notification
}

type Notification struct {
    Title    string
    Subtitle string
    Body     string
    Sound    string // default "Glass"
    Group    string // prevents stacking, e.g. "ailang-task-{taskID}"
    URL      string // click action (open URL)
}
```

### 3. launchd plist (`scripts/ailang-daemon.plist`)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ...>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sunholo.ailang.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/ailang</string>
        <string>daemon</string>
        <string>--env</string>
        <string>prod</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/ailang-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/ailang-daemon.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>AILANG_CLOUD_PROJECT</key>
        <string>ailang-multivac</string>
    </dict>
</dict>
</plist>
```

Install: `ailang daemon install` copies plist to `~/Library/LaunchAgents/` and runs `launchctl load`.

### 4. `ailang daemon install/uninstall/status` subcommands

```
ailang daemon install [--env dev|test|prod]   # install launchd plist, start daemon
ailang daemon uninstall                        # unload + remove plist
ailang daemon status                           # show launchctl status + last 10 log lines
ailang daemon run                              # foreground mode (default, used by launchd)
```

---

## Config (`~/.ailang/config/daemon.yaml`)

```yaml
env: prod                    # which cloud environment to watch
inbox: mark                  # your inbox name for message notifications
notify:
  approvals: true
  completions: true
  failures: true
  messages: true
  packages: true
  min_cost_usd: 0.0          # only notify completions above this cost (0 = all)
sounds:
  approval: "Glass"
  completion: "Ping"
  failure: "Basso"
  message: "Pop"
```

---

## Testing

```bash
# Unit: mock Pub/Sub, verify notification calls
go test ./internal/notify/...
go test ./cmd/ailang/ -run TestDaemon

# Integration: send a test message, verify notification fires within 30s
ailang daemon run --dry-run &
ailang messages send website-builder "test" --title "daemon test"
# expect: dry-run log shows notification for inbox message
```

---

## Non-goals

- Windows/Linux notifications (macOS only for now)
- Notification history / read receipts
- Interactive approval from notification (phase 2 — use terminal-notifier's reply action)
