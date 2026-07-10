# M-NOTIFY-CHANNELS-FRAMEWORK: Outbound Notification Channels (Adapter Framework)

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium)
**Estimated**: ~1.5 weeks (outbound-first; inbound reply-to-approve deferred)
**Dependencies**: M-MSG-AUTO-TRIAGE-PIPELINE (the notification bus that drives this), internal/pubsub (events topic, shipped)

> **Scope split:** [m-msg-auto-triage-pipeline.md](m-msg-auto-triage-pipeline.md) owns the notification **bus** — it decides *what* to notify and *which channel name(s)* to route to, then calls `registry.Get(name).Send(intent)`. **This doc** owns the channel **framework**: the adapter interface, the registry, the concrete transports (Discord, Google Chat, email), the channel *choice*, and the security model. The seam is the `Channel.Send` call.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

A pluggable outbound-delivery framework. Neutral on language semantics; positives come from explicit authority (every channel gated by an env-var secret, fail-closed) and structured failure (delivery errors are typed + dead-lettered, never silent).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Delivery is an IO side effect; no language determinism impact |
| A2: Replayability | +1 | Each send is logged as an event; delivery attempts/outcomes are reconstructable |
| A3: Effect Legibility | +1 | A `Send` is an explicit, single-purpose effect at one interface — no channel emits implicitly |
| A4: Explicit Authority | +1 | Every channel is env-var-gated and fail-closed (no secret → not registered → cannot send); mirrors Aitana's non-negotiable gating + the cascade topic's authenticated-source rule |
| A5: Bounded Verification | +1 | The adapter contract (`Name` + `Send`, later `Verify`/`Parse`) is locally checkable; a smoke contract test enrolls every channel |
| A6: Safe Concurrency | 0 | Sends are independent; framework adds no shared mutable state beyond the registry (write-once at startup) |
| A7: Machines First | +1 | Notification intents are structured (event type + payload + deep link), not free-text blobs |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | 0 | Channel sends are cheap; no resource-model change |
| A10: Composability | +1 | New channels are added by implementing one interface + registering once; composes with the bus unchanged |
| A11: Structured Failure | +1 | `Send` returns a typed error; failures route to `ailang-dead-letter`; missing-secret rejects loudly (no v5-style "log + accept") |
| A12: System Boundary | +1 | Each channel is an explicit egress boundary with its own secret + verification (for future inbound) |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No nondeterminism in language; delivery is boundary IO
- [x] A3 (Effects): Sends are explicit, logged effects at one interface
- [x] A4 (Authority): Fail-closed — no secret means the channel is not registered and cannot send
- [x] A7 (Machines First): Structured notification intents, not human-convenience free text

## Problem Statement

AILANG has no human-reachable outbound notification channel. The bus design in [m-msg-auto-triage-pipeline.md](m-msg-auto-triage-pipeline.md) can decide *that* the developer should be notified (e.g., a design doc is awaiting approval), but it has nowhere to send it except a Claude Code session (only live during a session), a GitHub issue (only if watched), or the dashboard (only if open).

**Current State:**

- **Zero outbound channel code** in the AILANG repo — no Discord, Slack, Telegram, Google Chat, SMTP, or transactional email. Confirmed by search.
- **Notifications can't reach the developer off-session.** When the dev is away from the terminal (the increasingly common case as work moves to laptop / phone / cloud), approval-pending and task-failed events are invisible until a session starts.
- **No pluggable delivery abstraction.** Without a framework, every new channel would be ad-hoc wiring scattered across the coordinator — the exact fragmentation CLAUDE.md §3 warns against.

**Proven prior art exists (different codebase).** The Aitana v6 platform (`aitana-labs/aitana-skills/backend/channels/`, Python) ships a mature **Channels Framework**: a `BaseChannel` ABC (`name`, `verify_webhook`, `parse_inbound`, `send`) + a `ChannelRegistry` that both mounts inbound webhooks *and* exposes `registry.get(name).send(...)` for outbound. It has working Discord, Telegram, email (Mailgun), and WhatsApp adapters, a fail-closed security model, env-var gating, a per-channel chunker, and a cross-channel smoke contract test — with a documented "~4h to add a channel" budget. **We should port this design (not the code) into Go**, not reinvent it.

**Impact:**

- **Who:** the developer (single maintainer, running parallel agents on an always-on Mac Studio, increasingly notified from a phone/laptop) and the autonomous pipeline that needs a way to escalate to a human.
- **How significant:** this is the missing egress that makes the autonomous triage pipeline actually *autonomous* — without it, the dev must keep checking a session/dashboard. It's the difference between "the pipeline can run while I'm away" and "I have to babysit it."

## Goals

**Primary Goal:** A pluggable Go notification-channel framework with at least one working outbound channel (Discord or Google Chat), so the bus can deliver an approval-pending notification to the developer's phone with no terminal open.

**Success Metrics:**

- Outbound reach: from **session/dashboard-only** → at least one always-on push channel (mobile).
- Add-a-channel cost: a new outbound channel is **one interface impl + one registration block** (target < 3h, mirroring Aitana's ~4h budget which includes inbound).
- Fail-closed: 100% of channels reject (don't register) when their secret is missing — verified by the smoke contract test.
- Zero silent drops: every send failure is typed and dead-lettered.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Outbound-first** interface (`Name` + `Send`); inbound (`Verify`/`Parse` → reply-to-approve) deferred to a later phase | Keeps the v1 interface minimal and shippable; full bidirectional now would 3x the surface for a need we don't have yet | human | design | med |
| **Lead channel = Discord (or Google Chat) via incoming webhook** | Webhook outbound needs only a URL secret (no bot, no signature verification, no Cloud Function) → lowest friction + mobile push; Aitana has a working Discord adapter to mirror | human | design | med |
| **Email is optional**, delivered via a Cloud Function on the events topic (not the local dispatcher) | Email needs a provider account + HMAC + secret mgmt; keeping it optional + cloud-side means the Studio-local path has zero email dependency | human at review | implementation | med |
| Port Aitana's **adapter + registry** pattern (fail-closed, env-gated, registry-as-output-router) | This is the proven shape; deviating risks re-learning Aitana's lessons (v5 "log + accept" footgun, double-register bug, length limits) | agent | design | low |
| **Secrets live in config/env (local) and Secret Manager (cloud)**, never in repo | Security boundary; a leaked webhook URL = anyone can spam the dev | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Outbound-first interface; inbound deferred — **confirmed**
- [ ] Lead channel: Discord vs Google Chat (user is on Google Workspace — Google Chat is the "native" option; Discord is the lowest-friction + has Aitana prior art). Pick one for v1.
- [ ] Email: in scope for this milestone or deferred entirely to a follow-up? (Recommended: defer the Cloud Function; ship the webhook channel first.)
- [ ] Secret storage convention (config key names + Secret Manager paths)

## Solution Design

### Overview

Port Aitana's Channels Framework into a new Go package `internal/notify`, trimmed to **outbound-first**. A `Channel` interface has a `Name()` and a `Send(ctx, target, intent)`. A process-singleton `Registry` (write-once at startup, env-gated) maps name → channel. The notification dispatcher (defined in the bus doc) calls `Registry.Get(name).Send(...)`. The first concrete adapter is a webhook channel (Discord or Google Chat) — a single authenticated HTTPS POST with internal length chunking. Email is an optional Cloud Function adapter on the events topic, added later if the webhook channel proves insufficient.

The inbound half (webhook verify → parse → "reply-to-approve") is explicitly **future work** — the interface is designed so it can grow `Verify`/`Parse` methods without breaking outbound adapters.

### Architecture

**Components:**

1. **`Channel` interface** (`internal/notify/channel.go`): `Name() string` + `Send(ctx, target string, intent Notification) error`. Outbound-only for v1. (Future: optional `Inbound` sub-interface with `Verify(headers, body) bool` + `Parse(payload) (*Inbound, bool)`.)
2. **`Notification` struct**: structured intent — `EventType` (approval_pending, task_failed, …), `Title`, `Body`, `DeepLink` (GitHub issue / dashboard URL), `Severity`, `Metadata`. The bus constructs this; channels render it for their transport.
3. **`Registry`** (`internal/notify/registry.go`): port of Aitana's `ChannelRegistry` — `Register(ch)` (idempotent on same instance, errors on a different instance for the same name), `Get(name)`, `Names()`. Used by the dispatcher as the output router (Aitana's exact `registry.get(name).send(...)` pattern).
4. **Webhook adapter** (`internal/notify/discord.go` or `googlechat.go`): implements `Send` as one authenticated POST; chunks at the transport's length limit (Discord 2000, Google Chat 4096). Env-gated: not registered if its webhook-URL secret is unset.
5. **Email adapter (optional)** (`functions/notify-email/`): Cloud Function subscriber on the events topic → transactional provider. Lives cloud-side so the local path has no email dependency.
6. **Env-gated registration** (`internal/notify/register.go`): wires channels at coordinator startup behind `if secret != ""` checks — the non-negotiable gate that keeps the daemon booting without secrets (Aitana's hard-won rule).

### Port Mapping (Aitana Python → AILANG Go)

| Aitana (Python) | AILANG (Go) | Notes |
|---|---|---|
| `BaseChannel` ABC (`verify_webhook`, `parse_inbound`, `send`) | `Channel` interface (`Name`, `Send`) | Outbound-first; inbound methods deferred |
| `OutboundMessage` (text, format, metadata) | `Notification` struct | Adds structured `EventType` + `DeepLink` for the bus use case |
| `ChannelRegistry` (register / get / mount_webhooks) | `Registry` (Register / Get / Names) | Drop webhook mounting in v1 (no inbound); keep `Get` as output router |
| env-var gate at registration site | `if secret != ""` at `register.go` | Same fail-closed-on-missing-secret rule |
| `_chunk.py::chunk_message` | `internal/notify/chunk.go` | Per-transport length limits |
| `test_smoke_all_channels.py` contract test | `internal/notify/smoke_test.go` | Every channel enrolls: valid send → ok, missing secret → not registered |
| Mailgun email adapter (HMAC, In-Reply-To) | `functions/notify-email/` (optional) | Cloud-side; only if webhook proves insufficient |

### Channel Choice Comparison

| Channel | Outbound friction | Mobile push | Two-way later | Infra needed | Prior art |
|---|---|---|---|---|---|
| **Discord** (incoming webhook) | Lowest — one webhook URL, plain POST | Excellent | Yes (bot, later) | None (local POST) | Aitana `discord.py` |
| **Google Chat** (incoming webhook) | Low — one webhook URL | Good (Workspace app) | Yes (Chat app) | Google Workspace (user has it) | — |
| **Telegram** (bot send) | Low — bot token + chat id | Excellent | Yes (bot) | None | Aitana `telegram_.py` |
| **Email** (Mailgun/SendGrid) | High — provider acct, HMAC, Cloud Fn | Poor (no push) | Hard (inbound parsing) | Provider + Secret Mgr + Cloud Fn | Aitana `email_.py` |
| **GCP Cloud Monitoring channels** | Medium — alert policy plumbing | via email/SMS/app | No | Monitoring alert policies | — |

**Recommendation:** Lead with **Discord** (lowest friction, mobile push, Aitana prior art to mirror) *or* **Google Chat** (native to the user's Google Workspace). Both are incoming-webhook based, so outbound v1 is a single authenticated POST with no Cloud Function. **Email is optional** and deferred — it's the heaviest option and has no mobile push, so it's a poor fit for the "page me on my phone" need. The "gcloud/Cloud Monitoring channels" route is awkward for app-event notifications (it's built for infra alerts), so it's not recommended as the primary path.

### Implementation Plan

**Phase 1: Framework + first webhook channel** (~1 week) — **shipped (M-NOTIFY-CHANNELS, v0.23.0)**
- [x] `Channel` interface (`internal/notify/channel.go`). **Note:** reuses the *existing* `internal/notify.Notification` (the macOS notifier's type) rather than a new struct — discovered an `internal/notify` package + `internal/daemon` dispatcher already exist; macOS is now a `Channel` (`MacOSChannel`)
- [x] `Registry` ported from Aitana (`internal/notify/registry.go`) — register/get/names, same-instance idempotency, different-instance error
- [x] `chunk.go` per-transport length splitter (rune-safe)
- [x] Webhook adapter: **Discord** (`internal/notify/discord.go`) — `Send` = authenticated POST + chunk
- [x] Env-gated registration (`register.go`) — Discord registered only if `AILANG_DISCORD_WEBHOOK_URL` set
- [x] Smoke contract test enrolling the channel (`smoke_test.go`: missing secret → not registered)
- [x] Unit tests: send happy path (mock HTTP), chunking, non-2xx + transport error → typed error

**Phase 2: Second channel + dead-letter integration** (~0.5 week)
- [ ] Add a second channel (whichever of Discord/Google Chat wasn't v1) to prove the framework generalizes
- [ ] Wire `Send` failures into `ailang-dead-letter` (via the dispatcher in the bus doc)
- [ ] Docs: write [docs/docs/guides/notification-channels.md](../../../docs/docs/guides/notification-channels.md) with an "add a channel" how-to (mirror Aitana's adapter-howto)

**Future (separate milestone): inbound / two-way**
- Grow the interface with `Verify` + `Parse`; mount inbound webhooks; enable reply-to-approve (reply "approve" → apply `design-approved`). Port Aitana's `handle_webhook` flow + signature verification.

### Files to Modify/Create

**New files:**
- `internal/notify/channel.go` (~80 LOC) — `Channel` interface + `Notification` struct
- `internal/notify/registry.go` (~120 LOC) — registry (port of Aitana `ChannelRegistry`)
- `internal/notify/chunk.go` (~50 LOC) — length-limit splitter
- `internal/notify/discord.go` **or** `googlechat.go` (~120 LOC) — first webhook adapter
- `internal/notify/register.go` (~60 LOC) — env-gated startup registration
- `internal/notify/*_test.go` (~250 LOC) — unit + smoke contract tests
- `docs/docs/guides/notification-channels.md` (~120 LOC) — add-a-channel how-to (the AILANG analogue of Aitana's `channels-adapter-howto.md`)
- `functions/notify-email/` (~150 LOC, **optional/deferred**) — Cloud Function email adapter

**Modified files:**
- `internal/messaging/config.go` (+40 LOC) — channel secret config keys (webhook URL, etc.)
- (bus doc owns the dispatcher that calls `Registry.Get(name).Send(...)`)

## Examples

### Example 1: The `Channel` interface (Go port of Aitana's `BaseChannel.send`)

```go
// internal/notify/channel.go
type Notification struct {
    EventType string            // "approval_pending", "task_failed", ...
    Title     string
    Body      string
    DeepLink  string            // GitHub issue or dashboard URL
    Severity  string            // "info", "warn", "error"
    Metadata  map[string]string
}

type Channel interface {
    Name() string
    Send(ctx context.Context, target string, n Notification) error
}
```

### Example 2: Discord webhook adapter (mirrors `discord.py::send` chunking)

```go
// internal/notify/discord.go
func (c *DiscordChannel) Send(ctx context.Context, target string, n Notification) error {
    for _, chunk := range chunkMessage(render(n), 2000) { // Discord 2000-char limit
        body, _ := json.Marshal(map[string]string{"content": chunk})
        if err := postWebhook(ctx, c.webhookURL, body); err != nil {
            return fmt.Errorf("discord send: %w", err) // typed error → dead-letter
        }
    }
    return nil
}
```

### Example 3: Env-gated registration (Aitana's non-negotiable rule, in Go)

```go
// internal/notify/register.go — coordinator startup
func RegisterChannels(reg *Registry, cfg Config) {
    if url := cfg.DiscordWebhookURL; url != "" {
        reg.Register(NewDiscordChannel(url))
    } else {
        log.Info("discord channel not registered: webhook URL not set")
    }
    // ... google chat, etc. Same gate. Daemon boots fine with zero channels.
}
```

### Example 4: The bus → channel seam (from the other doc)

```go
// dispatcher (in m-msg-auto-triage-pipeline) selects channel name(s) by event filter,
// then delegates delivery to this framework — Aitana's registry-as-output-router pattern:
for _, name := range matchedChannels {
    if err := registry.Get(name).Send(ctx, target, notification); err != nil {
        publishDeadLetter(ev, err)
    }
}
```

## Success Criteria

- [ ] `Channel` interface + `Registry` implemented; registry refuses different-instance double-register
- [ ] One webhook channel (Discord or Google Chat) delivers a real notification to the dev's phone
- [ ] Channel is **not registered** when its secret is unset (verified by smoke test)
- [ ] `Send` failures return a typed error that the dispatcher dead-letters
- [ ] Long messages are chunked at the transport limit
- [ ] A second channel added with no framework changes (proves the abstraction)
- [ ] All tests passing (≥ 85% on `internal/notify`)
- [ ] `notification-channels.md` how-to written (add-a-channel guide)

## Testing Strategy

**Unit tests** (mirror Aitana's `tests/channels/` structure):
- `Send` happy path — mock HTTP client, assert request body/URL
- Chunking — message over the limit splits correctly
- Error path — non-2xx response → typed error
- Registry — same-instance idempotent, different-instance errors, `Get` unknown → error

**Contract test** (`smoke_test.go`, port of `test_smoke_all_channels.py`):
- Every registered channel: valid `Send` → ok; missing secret → not registered. New channels auto-enroll.

**Manual testing:**
- Trigger a real `approval_pending` event → confirm the notification arrives on the phone with a working deep link.

## Deferred Decisions

- Discord vs Google Chat as the v1 lead channel — **human at review** (Workspace-native vs lowest-friction + prior art)
- Notification rendering (plain vs rich embed / card) — **agent may choose** (start plain)
- Retry/backoff before dead-lettering — **agent may choose**
- Whether email ships in this milestone or a follow-up — **human at review** (recommended: defer)
- Internal naming within `internal/notify` — **agent may choose**

## Non-Goals

- **Inbound / two-way (reply-to-approve)** — deferred to a future milestone; the interface is designed to grow into it. [Future Work.]
- **Email as the primary channel** — it's optional and deferred (no mobile push, heaviest infra).
- **Skill/agent routing of inbound messages** (Aitana's `select_skill` / identity resolution) — that's Aitana's chat-platform need, not AILANG's notification need.
- **Reusing Aitana's Python code directly** — different language/runtime; we port the *design*, not the code.
- **GCP Cloud Monitoring notification channels** — not a fit for app-event notifications; not pursued.

## Timeline

**Week 1** (~10h): interface + registry + chunk + first webhook adapter + env-gating + tests
**Week 1.5** (~5h): second channel + dead-letter wiring + how-to doc

**Total: ~15 hours** (email Cloud Function adds ~6h if/when pursued)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Webhook URL leaks → notification spam to the dev | Med | Treat URL as a secret (Secret Manager cloud / env local); never in repo; rotate if leaked |
| Over-notifying (every event pushes) | Med | The bus's per-channel `event_types` filter (other doc) gates volume; default = approval_pending + task_failed only |
| Framework under-designed → rework when inbound is added | Low | Interface explicitly designed for an inbound sub-interface; Aitana already proved the shape |
| Transport length limits cause truncation | Low | `chunk.go` per-transport limits, ported from Aitana |
| Provider/transport outage drops a notification | Med | Typed error → dead-letter; the dashboard + GitHub channels remain as redundant paths |

## Related Documents

**This milestone's partner:**
- [m-msg-auto-triage-pipeline.md](m-msg-auto-triage-pipeline.md) — the notification bus that drives this framework (defines the dispatcher + channel registry config)

**Prior art (Aitana v6 Channels Framework — Python, different repo, the design we port):**
- `aitana-labs/aitana-skills/backend/channels/base.py` — `BaseChannel` ABC + message models (source of the Go `Channel` interface)
- `aitana-labs/aitana-skills/backend/channels/registry.py` — `ChannelRegistry` incl. the `get(name).send(...)` output-router pattern
- `aitana-labs/aitana-skills/backend/channels/discord.py`, `email_.py`, `telegram_.py` — concrete adapters
- `aitana-labs/aitana-skills/docs/integrations/channels-adapter-howto.md` — the ~4h add-a-channel operating manual (model for our how-to)

**AILANG code this builds on:**
- [internal/pubsub/topics.go](../../../internal/pubsub/topics.go) — events topic (notification source), dead-letter topic

**Possibly relevant (from neural search):**
- [design_docs/planned/v0_13_0/m-dashboard-pubsub-events.md](../v0_13_0/m-dashboard-pubsub-events.md) — dashboard's Pub/Sub event consumption (a sibling consumer of the events topic)

## References

- [Design Axioms](/docs/references/axioms)
- Aitana Channels Framework adapter how-to (prior art): `aitana-labs/aitana-skills/docs/integrations/channels-adapter-howto.md`

## Future Work

- **Inbound / reply-to-approve**: grow the interface with `Verify` + `Parse`, mount inbound webhooks, map "approve"/"reject" replies to GitHub approval labels (port Aitana's `handle_webhook` + signature verification)
- **Email Cloud Function** (if the webhook channel proves insufficient for some event classes)
- **More channels** (Telegram, Slack) — each is one interface impl once the framework exists
- **Per-event channel preferences** (e.g., approval → Discord, weekly digest → email)

---

**Document created**: 2026-05-28
**Last updated**: 2026-05-28
