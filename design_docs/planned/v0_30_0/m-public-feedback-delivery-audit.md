# M-PUBLIC-FEEDBACK-DELIVERY-AUDIT: external feedback isn't reaching Discord (or the inbox)

**Status**: Planned
**Target**: v0.30.x (operational — not v1.0-language-gating, but it's the HUMAN-INPUT channel that feeds the whole data-led loop, so treat as P1)
**Priority**: P1 (silent loss of real user feedback — Kevin's messages vanished; the feedback flywheel is blind)
**Estimated**: 0.5–1d investigation + fix (2 parts; part 1 is a small code fix, part 2 needs Cloud Run logs)
**Dependencies**: `internal/apiserver/{feedback_tool,mcp,ratelimit}.go`, `internal/feedback/publisher.go`, `internal/daemon/handlers.go`, `internal/notify/discord.go`; live logs for `mcp.ailang.sunholo.com`
**Author**: Opus session, requested by Mark 2026-07-12 (Kevin's public feedback never appeared in Discord)

---

## Problem Statement

External users' public feedback is being lost silently. Eval-suite regression pings reach Discord
fine, but Kevin's public messages never appeared there. Verified 2026-07-12:

- The **`public-feedback` inbox → Discord path WORKS**: `nightly-eval` regression messages sit in
  the `public-feedback` inbox (Firestore, `ailang-multivac-dev`) with `EventType: "public-feedback"`
  and reach Discord (that's why the eval fades showed up).
- **No external MCP feedback has landed in `public-feedback` since 2026-05-04** — the last three
  `from: mcp-public` messages are Apr 27 / Apr 30 / May 4, then nothing. Kevin's messages are NOT in
  `public-feedback`, `feedback`, `controlplane`, or `pkg:sunholo/ailang`.

Two independent defects, one latent + one active:

### Defect A (latent, code-confirmed) — Discord only forwards the `public-feedback` inbox
`internal/daemon/handlers.go:72` tags `EventType: "public-feedback"` **only** when
`m.ToInbox == "public-feedback"`; every other inbox gets `EventType: "message"`. Since commit
`3fce16a80` ("Discord defaults to attention-worthy only"),
`internal/notify/discord.go:21` `DefaultDiscordEventTypes = ["pending_approval","failed","public-feedback"]`
— so `"message"` is dropped. But `internal/feedback/publisher.go` explicitly routes package feedback
to `pkg:<vendor>/<name>` inboxes instead of `public-feedback`. **Result: any external feedback about
a specific package is invisible on Discord by construction.**

### Defect B (ROOT-CAUSED 2026-07-12, Mark's hypothesis confirmed) — dev/prod env split
**The inbound path works fine.** Kevin's messages ARE in Firestore — in the **prod** project
(`ailang-multivac`), `public-feedback` inbox, 2026-06-30 (fb_c3427, fb_942b7, fb_343bd, fb_cef30…),
all triaged/read (his `flushStdout()` ask shipped as the `_io_flush` builtin the same day). The
"May-4 cliff" was an artifact of querying **dev** only: the public MCP (`mcp.ailang.sunholo.com`)
writes to **prod** Firestore, while the rig's notify daemon
(`com.sunholo.ailang.daemon.plist` → `AILANG_CLOUD_PROJECT=ailang-multivac-dev`, verified) subscribes
to **dev** only. Eval pings reach Discord because the rig *sends* them to dev; external feedback
never pings because *nothing listens to prod*. (The May-4 dev entries = when the public endpoint
last wrote to dev, i.e. the dev→prod cutover.)

**Fix direction:** the daemon subscribes to BOTH projects (second `EventSubscriber` on prod's
`ailang-messages` subscription — the daemon's `Run` already races two subscriptions, so adding a
second project is structural, not novel), OR a prod-side notifier bridges prod→Discord directly.
Prefer daemon-dual-subscribe: one binary, one webhook, no prod mutation. Also update agent runbooks:
**triaging public feedback requires `AILANG_CLOUD_PROJECT=ailang-multivac` (prod)** — every agent
session that checked dev-only has been blind to real users (this session included, until Mark
corrected it).

## Goals

External feedback (from any user, about AILANG or any package) reliably lands in an inbox AND pings
Discord — no silent loss. Success:
- A submitted public feedback message appears in Discord within seconds (end-to-end live test).
- Package-scoped feedback (`pkg:*` inbox) also reaches Discord.
- The May-4 cliff root-caused and fixed (or explained: "Kevin used docparse MCP").

## Solution Design (2 parts)

**Part 1 — fix Defect A (small, do first):** in `messageNotification` (handlers.go), tag
`EventType: "public-feedback"` for ALL externally-sourced feedback, not just the literal
`public-feedback` inbox — e.g. treat `public-feedback` OR any `pkg:*` inbox (or a `Source=external`
flag set by the publisher) as Discord-worthy. Alternatively add `"message"`-from-external to the
Discord allow-list. Add a fanout unit test per inbox class.

**Part 2 — root-cause Defect B (needs logs):** `gcloud logging read` on the Cloud Run service
behind `mcp.ailang.sunholo.com` for `submit_feedback` since May — are requests arriving? erroring?
rate-limited? Confirm which MCP Kevin used. Then fix the specific break (ratelimit tuning, param
validation, or a routing/config regression) with a regression test + a live end-to-end submit check.

## Conflict Surface

Touches the notify fan-out (shared by macOS + Discord + eval pings) and the public apiserver
feedback path. Must NOT: spam Discord with internal `message`-type traffic (keep the allow-list
tight — broaden it for *external-sourced* only, not all messages); regress the working
eval→public-feedback→Discord path; weaken the edge rate-limit's abuse protection.

## Verification Log

| Claim | Method | Result |
|---|---|---|
| public-feedback→Discord works | inbox list + eval pings on Discord (Mark) | Confirmed |
| Discord allow-list drops "message" | `internal/notify/discord.go:21` | Confirmed (3 types only) |
| Only public-feedback inbox tagged | `internal/daemon/handlers.go:72` | Confirmed |
| publisher routes pkg feedback to pkg:* inbox | `internal/feedback/publisher.go:64` | Confirmed |
| No external feedback since May 4 | `AILANG_STORAGE=gcp … messages list --inbox public-feedback` | Confirmed (last mcp-public 2026-05-04) |
| Kevin's messages absent from checked inboxes | list public-feedback/feedback/controlplane/pkg | Confirmed absent |

## Non-Goals

- Building a feedback *dashboard* (separate).
- Enabling the feedback-triage-gate in production (that's the parked m-feedback-gate-cloud-adapter ops task).

## Related Documents

- [m-feedback-triage-gate](../../implemented/v0_29_0/) + [m-feedback-gate-cloud-adapter] — the gate that sits on this path (off by default)
- [project_ailang_notify_daemon] (agent memory) — Pub/Sub → daemon → Registry fan-out architecture

---

**Document created**: 2026-07-12
