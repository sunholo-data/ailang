# Sprint Plan: M-NOTIFY-CHANNELS (Build Order Step 2)

**Sprint ID**: M-NOTIFY-CHANNELS
**Design doc**: [m-notify-channels-framework.md](m-notify-channels-framework.md) — Phase 1
**Target**: v0.24.0
**Risk**: Low (new self-contained package; no wiring into the daemon yet — that's Step 3)
**Estimated**: ~2 days (~370 LOC impl + ~250 LOC tests)
**Lead channel**: Discord (incoming webhook) — user-confirmed

## Goal

A pluggable Go outbound notification framework in `internal/notify`, ported from Aitana's `BaseChannel`+`ChannelRegistry`, with one working channel (Discord webhook) that can deliver a `Notification` to the developer's phone. Outbound-first; inbound (reply-to-approve) is deferred. No daemon wiring — the dispatcher (Step 3) calls `registry.Get(name).Send(...)`.

## Discovery / design

- Port target: `aitana-labs/aitana-skills/backend/channels/{base.py,registry.py,discord.py}`. We port the *design* (interface, registry-as-output-router, env-gated fail-closed registration, per-transport chunking), not the code.
- Outbound-first interface: `Channel{ Name() string; Send(ctx, target string, n Notification) error }`. Inbound `Verify`/`Parse` deferred.
- Discord `Send` = authenticated POST to the webhook URL with `{"content": chunk}`, chunked at 2000 chars. Inject an `httpDoer` (`Do(*http.Request)`) so tests mock HTTP with no network.
- Fail-closed: the Discord channel registers only if its webhook-URL secret is set (env `AILANG_DISCORD_WEBHOOK_URL` for local; Secret Manager path in cloud is Step 3+). No secret → not registered → cannot send.

## Milestones

### M1 — Framework core (~140 LOC + ~110 test)
- `internal/notify/channel.go`: `Channel` interface + `Notification` struct (`EventType`, `Title`, `Body`, `DeepLink`, `Severity`, `Metadata`).
- `internal/notify/registry.go`: `Registry` (port of Aitana `ChannelRegistry`) — `Register` (idempotent on same instance, error on a *different* instance for the same name), `Get` (error if unknown), `Names` (stable order).
- Tests: register/get/names, same-instance idempotency, different-instance error, unknown-get error.
- **Acceptance**: `go test ./internal/notify/` green; registry refuses different-instance double-register.

### M2 — chunk + Discord adapter (~150 LOC + ~100 test)
- `internal/notify/chunk.go`: `chunkMessage(s string, limit int) []string` (port of Aitana `_chunk.py`; never splits mid-rune; single chunk if under limit).
- `internal/notify/discord.go`: `DiscordChannel` implementing `Channel`; `Send` renders the notification, chunks at 2000, POSTs each chunk; non-2xx → typed error. Injectable `httpDoer` (defaults to `http.DefaultClient`).
- Tests (mock httpDoer, no network): happy path asserts URL + JSON body + chunk count; non-2xx → error; long message splits.
- **Acceptance**: `go test ./internal/notify/` green; no real network in tests.

### M3 — env-gated registration + smoke contract + docs (~80 LOC + ~40 test + docs)
- `internal/notify/register.go`: `RegisterChannels(reg *Registry)` that registers `DiscordChannel` only if `AILANG_DISCORD_WEBHOOK_URL` is set (logs a skip otherwise). Daemon boots fine with zero channels.
- `internal/notify/smoke_test.go`: contract test enrolling every registered channel — `Name()` non-empty; missing-secret → not registered.
- `docs/docs/guides/notification-channels.md`: add-a-channel how-to (the AILANG analogue of Aitana's adapter-howto).
- CHANGELOG entry.
- **Acceptance**: `make ci`-scope green (fmt/vet/lint + `go test ./internal/notify/`); how-to written.

## Success Metrics

- `internal/notify` ≥ 85% coverage; gofmt/vet/golangci-lint clean.
- Discord channel sends a real notification when `AILANG_DISCORD_WEBHOOK_URL` is set (manual one-off), and is *not* registered when it's unset (smoke test).
- Zero coupling to the coordinator daemon — pure library this step.

## Non-Goals (this sprint)

- Dispatcher / events-topic subscription (Step 3).
- Second channel + dead-letter (Step 4).
- Inbound / reply-to-approve (future milestone).
- `notifications.channels` config registry (Step 3 — this sprint gates Discord on a single env var).
