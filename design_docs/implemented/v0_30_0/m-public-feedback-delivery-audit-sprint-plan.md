# Sprint Plan: M-PUBLIC-FEEDBACK-DELIVERY-AUDIT

**Design doc**: [m-public-feedback-delivery-audit.md](./m-public-feedback-delivery-audit.md)
**Mission**: mission-control iteration 24 (Mark's NEXT-FIRST, P1 — silent loss of external user feedback)
**Planner model**: claude-opus-4-8
**Risk level**: medium (touches the shared notify fan-out + adds a cross-project Pub/Sub fan-in; the code is executor-testable but the live end-to-end proof requires a human daemon reload)
**Estimated**: 0.5–1d executor work, split into 3 code milestones (≤0.5d each) + 1 premise-gate + 1 parked-human handoff

---

## Executive Summary

Two independent defects cause external user feedback to be lost:

- **Defect A (code, executor-fixable):** the daemon tags `EventType: "public-feedback"` ONLY for the literal `public-feedback` inbox (`internal/daemon/handlers.go:72`). Package feedback lands in `pkg:*` inboxes → gets `EventType: "message"` → dropped by Discord's allow-list (`internal/notify/discord.go:21`, three types only).
- **Defect B (dev/prod split, executor-fixable code + human daemon reload):** the public MCP writes to **prod** Firestore/PubSub (`ailang-multivac`), but the rig notify daemon subscribes only to **dev** (`ailang-multivac-dev`, per the plist). Nothing listens to prod → external feedback never pings Discord.

Part 2's "read Cloud Run logs" investigation in the design doc is **already resolved** (root cause = dev/prod split). This sprint plans the **fix + verification**, not a logs-spelunking milestone.

### Plan-stage reality check (verified at HEAD; corrections to the design doc)

| Design-doc premise | Verified finding | Impact on plan |
|---|---|---|
| "the daemon's `Run` already races two subscriptions, so adding a second project is structural, not novel" | **Partly misleading.** `Run` races two *subscriptions* (`EventsSub`+`MessagesSub`) but both through a SINGLE `EventSubscriber` (`d.sub`) bound to ONE `*pubsub.Client`/project, plus a SINGLE `fetcher` bound to one project's Firestore (`daemon.go:52-60`, `cmd/ailang/daemon.go:96,123`). Adding a *project* = a real multi-project fan-in: second pubsub client + second subscriber + second fetcher, and either a `Daemon` refactor to hold N (sub,fetcher) tuples or two `Daemon` instances. | Sized as a real refactor milestone (M2), not a one-line list edit. |
| Part 2 "needs Cloud Run logs" | **Already root-caused** (dev/prod split). | No logs milestone. Plan = fix + verify. |
| A **new prod Pub/Sub subscription** may be needed (Terraform / park-for-human) | **FALSE — no new infra needed.** `projects/ailang-multivac/subscriptions/ailang-messages-laptop` ALREADY EXISTS (verified via `gcloud pubsub subscriptions list --project ailang-multivac`), topic `ailang-messages`, ackDeadline 30s, ACTIVE. Prefix `ailang` + base `messages-laptop` = existing sub. | No Terraform. No park-for-human ops for infra. |
| Daemon may lack IAM to read prod | **FALSE — access already exists.** Daemon runs under ADC as `m@sunholo.com`, who is `roles/owner` on BOTH `ailang-multivac-dev` AND `ailang-multivac` (verified via `gcloud projects get-iam-policy`). Owner ⊇ pubsub.subscriber. (Note: neither project grants pubsub.subscriber to the *user* explicitly — access is via Owner. Dev works today for the same reason.) | No IAM grant needed. |
| Daemon config's `MessagesSub: "messages-laptop"` is the correct base for prod | **Confirmed.** `Client.Subscription(base)` prepends the client prefix (`internal/pubsub/client.go:53`). Prod client (prefix `ailang`) + `messages-laptop` → `ailang-messages-laptop` ✓. Dev (prefix `ailang-dev`) → `ailang-dev-messages-laptop` ✓ (both exist). | Reuse the same base sub name for both projects. |

**Net effect of the reality check:** the ONLY hard external dependency (a live prod end-to-end test) requires Mark to reload the daemon with the new env — a launchctl op that is auto-denied for the agent on this rig. Everything else is executor code + executor-runnable tests. There is **no unresolved blocking premise** — so the first milestone is a fast confirmation gate, not an open spike.

---

## Milestones

### M0 — Premise-confirmation gate (≤0.1d, FIRST)

Fast, read-only re-confirmation before touching code (all pre-verified by the planner; the executor re-checks to guard against drift since planning):

- Confirm `projects/ailang-multivac/subscriptions/ailang-messages-laptop` still exists and is ACTIVE:
  `gcloud pubsub subscriptions list --project ailang-multivac --format="value(name)" | grep ailang-messages-laptop`
- Confirm active ADC identity is an Owner on prod:
  `gcloud auth list` + `gcloud projects get-iam-policy ailang-multivac --flatten="bindings[].members" --filter="bindings.role:roles/owner"`
- Confirm `EnvProject["prod"] = {Project: "ailang-multivac", Prefix: "ailang"}` in `internal/daemon/config.go`.

**Acceptance:** all three confirmations pass. If ANY fails (e.g. subscription was deleted), STOP and escalate to Mark — the fix approach changes (would then need a Terraform-managed prod subscription, a park-for-human step). Given planner verification, this is expected to pass immediately.

**Verification:** paste command outputs into the sprint JSON `gates` field. No code changes in M0.

---

### M1 — Defect A: broaden Discord allow-list for externally-sourced feedback (≤0.3d)

**Goal:** `pkg:*` inbox feedback (and any external-sourced feedback) reaches Discord, without spamming Discord with internal `message` traffic.

**Approach (chosen):** In `messageNotification` (`internal/daemon/handlers.go:68`), treat an inbox as external-feedback-worthy when `ToInbox == "public-feedback"` OR `strings.HasPrefix(ToInbox, "pkg:")`. Give both the `EventType: "public-feedback"` tag (which the Discord allow-list already accepts) and the 🌐 external-feedback shape (with the inbox name in the body so `pkg:*` is distinguishable). Everything else keeps `EventType: "message"` (Discord drops it; macOS still shows it) — preserving the "don't spam Discord with internal traffic" constraint.

- Add a small helper `isExternalFeedbackInbox(inbox string) bool` (public-feedback OR `pkg:` prefix) so the rule is named and unit-testable, and so a future `Source=external` flag can extend it in one place.
- Do NOT add `"message"` to `DefaultDiscordEventTypes` (that would leak all internal inbox traffic to Discord — explicitly forbidden by the Conflict Surface). Broaden by *tagging external sources as public-feedback*, not by widening the allow-list.

**Files:** `internal/daemon/handlers.go` (~15 LOC), `internal/daemon/handlers_test.go` (~40 LOC).

**Acceptance criteria:**
- A message to `pkg:sunholo/auth` yields `EventType: "public-feedback"` and a 🌐 title, with `pkg:sunholo/auth` visible in the body.
- A message to `public-feedback` still yields the existing 🌐 / `public-feedback` behavior (no regression to `TestMessageNotification_PublicFeedback` / `TestDaemon_PublicFeedbackEventFiresDedicatedNotification`).
- A message to an internal inbox (`user`, `sprint-executor`, `controlplane`) still yields `EventType: "message"` (dropped by Discord).
- `NewDiscordChannel(...).Accepts("public-feedback")` is true and `.Accepts("message")` is false (unchanged `DefaultDiscordEventTypes`).

**Verification (executor-runnable):**
- New table test `TestMessageNotification_PkgInboxIsExternalFeedback` covering public-feedback / `pkg:*` / internal, following the existing `handlers_test.go` idiom.
- `go test ./internal/daemon/... ./internal/notify/...` green.

---

### M2 — Defect B: daemon dual-subscribe (dev + prod) (≤0.5d)

**Goal:** one daemon process listens to BOTH `ailang-multivac-dev` and `ailang-multivac` messages subscriptions (and, at minimum, prod's messages sub), fanning out to the same notifier/Discord webhook. No prod mutation, one binary, one webhook.

**Approach (primary): make `Daemon` hold N message sources.**
The blocker is that `Daemon` binds one `sub` + one `fetcher`. Refactor so a message source is a `(EventSubscriber, MessageFetcher, subName, projectLabel)` tuple; `Run` iterates all sources in parallel (extend the existing WaitGroup/errCh pattern in `daemon.go:83`). Task events stay dev-only (the rig emits eval pings to dev — do NOT double-fan prod task events; only messages need prod). Keep the message dedup window per-source or shared (message IDs are globally unique `fb_*`/`msg_*`, so a shared dedup is safe and simpler).

CLI wiring (`cmd/ailang/daemon.go:daemonRun`): after building the dev client/subscriber/fetcher as today, additionally build a prod client (`pubsub.NewClient(ctx, "ailang-multivac", "ailang")`) + subscriber + a prod Firestore fetcher (a second `storage.Backends` / messaging store scoped to `AILANG_CLOUD_PROJECT=ailang-multivac`), and register both as message sources. Gate the second source behind a config/flag so single-project mode still works (see below).

**Configuration surface (how the daemon learns to dual-subscribe):**
- Add `FileConfig.ExtraMessageEnvs []string` (yaml `extra_message_envs`) and/or a repeatable `--also-subscribe prod` flag to `daemon run`. Each extra env is resolved through the existing `EnvProject` map to `(project, prefix)`, so the daemon reads that project's `messages-laptop` sub.
- Default remains single-project (backward compatible). The rig opts in via daemon.yaml `extra_message_envs: [prod]` (or the plist adds `--also-subscribe prod`).
- Rationale for env-list over hardcoding prod: keeps the "how does it learn its project" answer explicit and testable, and avoids baking prod into every user's daemon.

**Fallback approach (if the `Daemon` refactor proves too invasive within the time box):** run TWO `Daemon` instances in `daemonRun` (dev + prod), each with its own sub+fetcher, sharing the same `reg.FanOut(...)` notifier. Smaller code, keeps `Daemon` frozen; trade-off is two dedup windows (harmless — disjoint message IDs) and two `Run` goroutines. Record which approach was taken in the sprint JSON notes.

**Firestore fetcher scoping (important):** prod message notifications carry only a MessageID; `handleMessageEvent` fetches the full doc from Firestore. The prod source's fetcher MUST read **prod** Firestore, not dev. Verify how `storage.NewBackends` selects its project (reads `AILANG_CLOUD_PROJECT`); if it is process-global env-driven, the executor must construct the prod messaging store explicitly (pass project to the firestore/messaging constructor) rather than mutating the global env — otherwise the two fetchers collide. This is the highest-risk detail in the sprint; call it out in the PR.

**Files:** `internal/daemon/daemon.go` (source-list refactor OR none if two-instance), `internal/daemon/config.go` (extra-envs field + resolution, ~25 LOC), `cmd/ailang/daemon.go` (prod client/fetcher wiring + flag, ~40 LOC), tests in `internal/daemon/daemon_test.go` (~60 LOC).

**Acceptance criteria:**
- With dual-subscribe enabled, a message delivered on the **prod** messages source fires exactly one notification through the notifier (fake subscriber keyed per-source, following `daemon_test.go`'s `fakeSubscriber`/`fakeFetcher` idiom).
- With dual-subscribe DISABLED (default), behavior is byte-for-byte today's single-dev-project behavior (all existing `daemon_test.go` tests pass unchanged).
- The prod source's fetcher resolves against a prod-scoped store (proven in test by a fetcher that returns a prod message only for the prod source).
- No prod resource is created/modified by executor code (subscription already exists; verified in M0).
- Dedup still suppresses a duplicate prod message; a dev and a prod message with the SAME id (won't happen in practice) is documented as harmless.

**Verification (executor-runnable):**
- `TestDaemon_DualProjectMessageFiresOnce` + `TestDaemon_SingleProjectUnchanged` in `daemon_test.go`.
- `go test ./internal/daemon/...` and `go build ./...` green.
- `make test` for the touched packages; `make lint`.
- **NOT executor-runnable:** live prod delivery (requires the daemon running with the new env + a real prod Firestore/PubSub round-trip). Deferred to the parked human checklist.

---

### M3 — Runbook + config docs + CHANGELOG (≤0.2d)

**Goal:** stop future agent sessions being blind to prod, and document the opt-in.

- Update agent runbook / memory-adjacent docs: **triaging public feedback requires `AILANG_CLOUD_PROJECT=ailang-multivac` (prod)** — every dev-only check has been blind to real users. Put this in `docs/docs/guides/agent-messaging.md` (or the daemon/cloud-messaging guide) and note it in the design doc's Related Documents.
- Document the new `extra_message_envs` / `--also-subscribe` opt-in in the daemon docs (`docs/docs/guides/cloud-messaging-integration.md` or the daemon section) with the exact daemon.yaml snippet and plist edit.
- `CHANGELOG.md` entry under v0.30.x: Defect A + Defect B fixes, dual-subscribe opt-in, runbook note.

**Acceptance criteria:**
- Runbook explicitly names the prod project for public-feedback triage.
- Dual-subscribe opt-in documented with copy-pasteable config.
- CHANGELOG updated; docs build passes (`npm run build` in `docs/` if docs changed).

**Verification:** `make verify-examples` (if any example touched — none expected), docs build if docs changed.

---

## Day-by-day breakdown

**Day 1 (0.5–0.7d):**
- Morning: **M0** premise gate (~15 min) → **M1** Defect A fix + tests (~2h). Run `go test ./internal/daemon/... ./internal/notify/...`.
- Afternoon: start **M2** — decide refactor vs two-instance; verify `storage.NewBackends` project selection (the fetcher-scoping risk); implement prod source + config flag + tests.

**Day 2 (0.3–0.5d, only if M2 spills):**
- Finish **M2** tests (dual-project fires-once, single-project-unchanged), `make test`/`make lint`.
- **M3** runbook + CHANGELOG + docs build.
- Open PR into `dev` from an isolated worktree; hand the parked checklist to Mark.

If M2's `Daemon` refactor is clean, the whole sprint lands inside Day 1 + a short Day-2 doc pass.

---

## Parked for human (Mark) — daemon reload + live prod verification

Agent `launchctl` ops are auto-denied on this rig; the daemon reads on-disk env at load time. After the PR merges and `make install` ships the new binary, **Mark** must reload the daemon with the prod source enabled and run the live check. Exact steps:

1. **Enable dual-subscribe** — edit `~/.ailang/config/daemon.yaml` to add:
   ```yaml
   extra_message_envs: [prod]
   ```
   (or edit `~/Library/LaunchAgents/com.sunholo.ailang.daemon.plist` ProgramArguments to add `--also-subscribe prod` — whichever surface the executor shipped; the PR will state which).
   The plist keeps `--env dev` + `AILANG_CLOUD_PROJECT=ailang-multivac-dev` for the primary (dev) source; prod is the *additional* source.

2. **Reload the daemon:**
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.sunholo.ailang.daemon.plist
   launchctl load   ~/Library/LaunchAgents/com.sunholo.ailang.daemon.plist
   tail -f /tmp/ailang-daemon.log   # expect a startup line listing BOTH dev and prod message sources
   ```
   (Do NOT unload while a mission-control iteration is running.)

3. **Live end-to-end test (prod, general feedback):** submit a test public feedback via the prod MCP (`mcp.ailang.sunholo.com`) or:
   ```bash
   AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac \
     ailang messages send public-feedback "sprint verify: prod ping" --title "M-PUBLIC-FEEDBACK verify" --from mcp-public
   ```
   Expect a 🌐 External feedback ping in Discord within seconds.

4. **Live end-to-end test (prod, package feedback — proves Defect A):**
   ```bash
   AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac \
     ailang messages send "pkg:sunholo/ailang" "sprint verify: pkg ping" --title "M-PUBLIC-FEEDBACK pkg verify" --from mcp-public
   ```
   Expect a 🌐 ping showing the `pkg:sunholo/ailang` inbox.

5. **Confirm no regression:** dev eval-regression pings still reach Discord (they already did — this only ADDS a prod source).

6. Report back on the tracking issue; the executor records the live-check outcome in the sprint JSON `gates`.

**Note:** the send in steps 3–4 requires `AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac` *only for the send* (per agent-messaging memory — never set globally, it breaks local SQLite reads).

---

## Conflict-surface guardrails (must hold)

- **No Discord spam from internal traffic:** broaden by tagging *external-sourced* inboxes (public-feedback + `pkg:*`) as `public-feedback`, NOT by adding `"message"` to `DefaultDiscordEventTypes`. Internal `user`/`controlplane`/agent inbox messages stay `EventType: "message"` (Discord-dropped).
- **No regression to the working eval→public-feedback→Discord path:** existing `public-feedback` behavior and all current `daemon_test.go` tests pass unchanged; dual-subscribe is opt-in and defaults off.
- **No weakening of edge rate-limits:** this sprint does NOT touch `internal/apiserver/ratelimit.go` — the inbound MCP path is confirmed working (Kevin's messages ARE in prod Firestore); the defect is delivery/notify, not ingestion.
- **No prod mutation from executor code:** the prod subscription already exists; the daemon only *reads* it. No Terraform, no new IAM.

---

## Success metrics

- Defect A: `pkg:*` feedback reaches Discord (unit-proven; live-proven by Mark in parked step 4).
- Defect B: daemon dual-subscribes prod (unit-proven fires-once; live-proven by Mark in parked step 3).
- Zero regression: all existing `internal/daemon` + `internal/notify` tests green; default single-project behavior byte-identical.
- Runbook names the prod project for feedback triage.
- CHANGELOG entry present; docs build green if docs changed.

## Open premises for the executor to resolve first

1. **`storage.NewBackends` project selection (M2, highest risk):** confirm whether it reads `AILANG_CLOUD_PROJECT` process-globally or accepts an explicit project. If global, the executor must build the prod messaging store WITHOUT mutating the shared env (else dev/prod fetchers collide). Resolve before wiring M2.
2. **Refactor vs two-instance (M2):** pick based on how invasive the `Daemon` source-list refactor is; both satisfy acceptance — record the choice.

No blocking infra/IAM premise remains (all resolved at plan stage — see reality-check table).
