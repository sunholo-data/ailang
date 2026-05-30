# Notification Channels

AILANG's notification framework (`internal/notify`) delivers outbound
notifications — "a design doc is awaiting approval", "a task failed" — to wherever
you'll actually see them. It is the Go port of the design behind Aitana's v6
Channels Framework: a tiny `Channel` interface plus a `Registry` that the
notification daemon uses as an output router.

> **Status (v0.24.0):** outbound-only. The macOS desktop notifier and a Discord
> webhook channel ship today. Inbound ("reply *approve* to apply the
> `design-approved` label") is a planned follow-up. See
> [m-notify-channels-framework](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_24_0/m-notify-channels-framework.md).

## Concepts

- **`Notification`** — the transport-independent payload (`Title`, `Subtitle`,
  `Body`, `URL`, plus macOS-only `Sound`/`Group`). Channels render whichever
  fields they support; the rest are ignored.
- **`Channel`** — an outbound transport: `Name() string` + `Send(ctx, Notification) error`.
- **`Registry`** — `name → Channel`, used as the output router
  (`registry.Get("discord").Send(...)`). `Register` is idempotent for the same
  instance and errors on a *different* instance for the same name (catches
  double-registration bugs).
- **Fail-closed registration** — a channel registers only if its secret is
  present. No secret → not registered → the daemon still boots, just without that
  output.

## Channels that ship today

| Channel | Name | Secret | Notes |
|---------|------|--------|-------|
| macOS desktop | `macos` | none (host capability) | via `terminal-notifier`/`osascript`; no-op off Darwin |
| Discord | `discord` | `AILANG_DISCORD_WEBHOOK_URL` | incoming webhook; chunked at 2000 chars |

### Enabling Discord

1. In your Discord server: **Channel → Settings → Integrations → Webhooks → New Webhook**, copy the URL.
2. Provide the URL where the daemon runs (treat it as a secret — anyone with the URL can post). The daemon resolves it from the env var first, then the macOS login Keychain:

   **macOS (recommended — Keychain, no plaintext on disk):**
   ```bash
   security add-generic-password -U -A -a "$USER" -s ailang-discord-webhook -w 'https://discord.com/api/webhooks/…'
   ```
   The daemon runs as a user LaunchAgent and reads it via
   `security find-generic-password -s ailang-discord-webhook -w`. Update the
   value any time by re-running the command (the `-U` flag overwrites).

   **Or via env var** (any host; takes precedence over the Keychain):
   ```bash
   export AILANG_DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/…"
   ```
3. The channel registers automatically on next daemon start. Neither source set → not registered (macOS-only).

> **Live check:** `AILANG_DISCORD_WEBHOOK_URL='…' go test ./internal/notify/ -run TestDiscordLive -count=1 -v` sends a real test message through the production path.

## Adding a new channel (~1–2h)

Mirror `internal/notify/discord.go`:

1. **Implement `Channel`** in `internal/notify/<name>.go`:
   ```go
   type SlackChannel struct {
       webhookURL string
       http       httpDoer // injected so tests need no network
   }

   func (c *SlackChannel) Name() string { return "slack" }

   func (c *SlackChannel) Send(ctx context.Context, n Notification) error {
       for _, chunk := range chunkMessage(render(n), slackLimit) {
           if err := c.post(ctx, chunk); err != nil {
               return err // typed error -> caller dead-letters
           }
       }
       return nil
   }
   ```
   Handle your transport's length limit with `chunkMessage`. Return a typed error
   on transport failure or any non-2xx response.

2. **Gate registration on a secret** in `internal/notify/register.go`:
   ```go
   if url := os.Getenv("AILANG_SLACK_WEBHOOK_URL"); url != "" {
       if err := reg.Register(NewSlackChannel(url)); err == nil {
           registered = append(registered, "slack")
       }
   }
   ```
   Never construct a channel that panics on a missing secret — gate at the
   registration site so a host without the secret still boots.

3. **Test with no network** — inject a fake `httpDoer`; cover the happy path
   (assert URL + body + chunk count), a non-2xx response, and a transport error.
   The cross-channel contract test (`smoke_test.go`) enrolls every registered
   channel automatically (non-empty `Name` matching its registry key).

## Routing

The daemon stamps every `Notification` with an `EventType` (`pending_approval`,
`completed`, `failed`, `public-feedback`, `message`) and the fan-out asks each
channel whether it accepts the type. The macOS channel implements no filter and
takes everything (passive desktop feed); Discord ships with a curated allow-list
so the phone only buzzes for things that need your attention.

### Default Discord filter

```
DefaultDiscordEventTypes = ["pending_approval", "failed", "public-feedback"]
```

So out of the box: ⏳ approval-needed, ❌ task-failed, and 🌐 external-feedback
ping the phone. ✅ task-done and routine ✉️ inbox messages do **not** — they
still land on macOS.

### Overriding the filter

```go
// All-quiet remote (still register the channel for visibility, but no pings):
discord.SetEventTypes([]string{})

// Pager-grade: only the truly urgent
discord.SetEventTypes([]string{"failed"})

// Inverse — accept everything (revert to firehose):
discord.SetEventTypes(nil)
```

A channel that implements `EventFilter` and rejects an event is **skipped** by
the fan-out — it does not count toward the remote-authoritative ack quota and
does not trigger a retry. So filtering doesn't break the "ack if any remote
delivered" policy: if all remote channels filter out the event, the daemon
falls back to local-best-effort and acks on macOS success.

## Dead-lettering persistent failures

`internal/notify/registry.go::SendAll` returns an error when every accepting
remote channel fails. The daemon nacks → Pub/Sub redelivers. To prevent infinite
retry on a permanently-broken channel (e.g. a revoked Discord webhook), configure
**subscription-level dead-lettering** on `ailang-{env}-messages-laptop` and
`ailang-{env}-events-laptop`, routing to the existing `ailang-{env}-dead-letter`
topic after N attempts:

```bash
gcloud pubsub subscriptions update ailang-dev-messages-laptop \
  --project=ailang-multivac-dev \
  --dead-letter-topic=ailang-dev-dead-letter \
  --max-delivery-attempts=5
```

(Same for `ailang-dev-events-laptop`.) Pub/Sub does the routing for us — no
in-daemon dead-letter publishing code is needed.
