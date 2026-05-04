# M-MAC-NOTIFY-DAEMON — Sprint Plan

**Sprint ID**: M-MAC-NOTIFY-DAEMON
**Target**: v0.15.0
**Design Doc**: [m-mac-notify-daemon.md](m-mac-notify-daemon.md)
**Estimated Duration**: 2-3 days (~16-22 hours)
**Risk Level**: Low
**Created**: 2026-05-04
**Author**: Claude (sprint-planner)
**Status**: ✅ COMPLETE (2026-05-04, all 4 milestones merged on `dev`)

---

## Goal

Ship a persistent macOS notification daemon (`ailang daemon`) that pulls real-time events from cloud Pub/Sub subscriptions and surfaces them as native macOS notifications, including a dedicated handler for **`public-feedback` inbox messages** so external feedback submitted via `mcp.ailang.sunholo.com` reaches the maintainer between sessions instead of only at SessionStart.

## Motivating Evidence

1. **External consumers are now real.** `motoko_agent` integration (2026-05-04) confirms the `submit_feedback` MCP path works end-to-end (`fb_d3920906975b66e2`). Today the only between-session pickup is the SessionStart hook in `scripts/hooks/check_public_feedback.sh` — fine for episodic checking, insufficient for responsive triage as more external projects adopt AILANG.
2. **The carve-out exists but is incomplete.** `m-pkg-feedback-loop.md` M5 has a "minimum viable feedback notifier" carve-out, explicitly framed as a stop-gap. Pulling the full daemon forward delivers M5's scope plus the rest of the cloud-event surface (task approvals, completions, failures) for ~30% additional LOC.
3. **All infrastructure prerequisites already exist.** `SubEventsLaptop` and `SubMessagesLaptop` constants in `internal/pubsub/topics.go:24,28`; subscription resources in `terraform/pubsub.tf:193`; notification reference pattern in `~/.claude/hooks/session_end_speak.sh`. No new cloud infra; no new effects.

## Velocity Basis

Recent sprints (2026-04-28 → 2026-05-04):

| Sprint | Milestones | Days | LOC/day |
|---|---|---|---|
| M-AI-OPENROUTER (M1-M4) | 4 | ~5 | ~200 |
| M-SMT-CROSS-MODULE-FUNCTIONS (M1-M5) | 5 | ~3 | ~280 |

This sprint estimates 4 milestones across 2.5 days at ~250 LOC/day average — comfortably within velocity.

---

## Milestones

### M1 — `internal/notify/macos.go` (foundation)

**Estimated LOC**: ~150 implementation + ~80 tests (230 total)
**Estimated Hours**: 4
**Dependencies**: none

**Scope**:
- New package `internal/notify/`.
- `Notification` struct: `Title`, `Subtitle`, `Body`, `Sound`, `Group`, `URL`.
- `Notify(n Notification) error` — tries `terminal-notifier` first (supports click action), falls back to `osascript display notification`.
- Sound mapping: `"Glass" | "Ping" | "Basso" | "Pop"`.
- Group key prevents stacking (e.g. `ailang-task-{taskID}` collapses repeat notifications).
- Detects missing binaries gracefully — returns typed `ErrNotifierUnavailable` when neither path is present (Linux/Windows dev environments).

**Test surface**:
- Unit tests stub `exec.Command` via a package-level `var execCommand = exec.Command` injection point (standard Go pattern).
- Tests cover: terminal-notifier path, osascript fallback, both unavailable, Group/URL field passthrough.
- No integration test in M1 — daemon-level integration covered in M4.

**Acceptance**:
- `go test ./internal/notify/...` passes.
- Manual smoke: a tiny test program calls `notify.Notify(n)` and a mac notification appears.

**Files**:
- `internal/notify/macos.go` (new)
- `internal/notify/macos_test.go` (new)

---

### M2 — `cmd/ailang/daemon.go run` (Pub/Sub consumer)

**Estimated LOC**: ~280 implementation + ~140 tests (420 total)
**Estimated Hours**: 6
**Dependencies**: M1

**Scope**:
- New `cmd/ailang/daemon.go` registering `ailang daemon` and the `run` subcommand.
- `daemon run [--env dev|test|prod] [--dry-run] [--config PATH]`:
  - Reads `~/.ailang/config/daemon.yaml` (schema in design doc) — uses sensible defaults if missing.
  - Spawns three parallel goroutines pulling:
    1. `SubEventsLaptop` — task approval / completion / failure events.
    2. `SubMessagesLaptop` — generic inbox notifications.
    3. **`SubMessagesLaptop` again with `to_inbox=public-feedback` filter** — dedicated handler that fetches the Firestore doc on receipt and emits the "🌐 External feedback" notification with click-through to the dashboard or Firestore console.
  - Per-event dedup window: 60s for tasks (key = `task_id+status`), 5min for messages (key = `message_id`).
  - Ack-after-notify: only acks the Pub/Sub message after `notify.Notify(...)` returns nil.
  - `--dry-run`: logs the notification payload but does not call `notify.Notify` — used for tests and integration checks.
  - Honors `~/.ailang/config/notify_excludes.conf` (one event-type substring per line; matched against title).
- Graceful shutdown on SIGINT/SIGTERM with goroutine wait.

**Test surface**:
- Unit tests with a stubbed Pub/Sub client (use the `pstest` package from cloud.google.com/go/pubsub).
- Tests cover: event routing (TaskStreamEvent vs InboxMessage vs public-feedback); dedup window enforcement; exclude-list matching; --dry-run behavior; ack-after-notify ordering.
- No real GCP calls in tests.

**Acceptance**:
- `go test ./cmd/ailang/ -run TestDaemonRun` passes.
- `ailang daemon run --dry-run` against a `pstest` server produces logged notifications for synthesized events.

**Files**:
- `cmd/ailang/daemon.go` (new)
- `cmd/ailang/daemon_test.go` (new)
- `cmd/ailang/help.go` (add `daemon` to help)

---

### M3 — install/uninstall/status + launchd plist

**Estimated LOC**: ~150 implementation + ~60 tests (210 total)
**Estimated Hours**: 4
**Dependencies**: M2

**Scope**:
- `scripts/ailang-daemon.plist.tmpl` — launchd plist template (env-var-substituted at install time so dev vs prod paths can differ).
- `daemon install [--env]` — copies plist (with substitutions) to `~/Library/LaunchAgents/com.sunholo.ailang.daemon.plist`, runs `launchctl load`, prints status. Detects already-installed and refuses to overwrite without `--force`.
- `daemon uninstall` — `launchctl unload`, deletes plist, leaves logs.
- `daemon status` — `launchctl list | grep com.sunholo.ailang.daemon`, last 10 lines of `/tmp/ailang-daemon.log`.
- `daemon run` is the foreground mode launchd invokes (already built in M2; this milestone adds the surface around it).

**Test surface**:
- Tests use a temp `HOME` and stub `launchctl` via `execCommand` injection.
- Tests cover: install creates plist with correct substitutions; install refuses overwrite; uninstall removes file; status output formatting.
- No actual launchd invocation in CI.

**Acceptance**:
- `go test ./cmd/ailang/ -run TestDaemonInstall` passes.
- Manual: `make install-notify-daemon` (target added in M4) results in `launchctl list | grep ailang` showing the daemon.

**Files**:
- `scripts/ailang-daemon.plist.tmpl` (new)
- `cmd/ailang/daemon_install.go` (new)
- `cmd/ailang/daemon_install_test.go` (new)

---

### M4 — Integration test, public-feedback handler verification, ship

**Estimated LOC**: ~80 implementation + ~50 tests (130 total)
**Estimated Hours**: 4 (mostly ops/docs)
**Dependencies**: M3

**Scope**:
- **End-to-end smoke test (manual, scripted)**: `scripts/smoke-notify-daemon.sh` — installs daemon in `--dry-run`, submits a test feedback via `curl` to dev MCP, asserts the daemon logs a notification within 30s. CI-skip; documented as a release gate.
- **Verify prod terraform laptop subscription** — confirm `messages-laptop` exists in `ailang-multivac/terraform/environments/prod/terraform.tfvars` `client_subscriptions = [...]`. If missing, add it; coordinate with multivac CI to apply.
- **Config schema doc** at `docs/docs/guides/notify-daemon.md` covering: `~/.ailang/config/daemon.yaml` schema, install/uninstall, troubleshooting, log location.
- **Makefile target**: `make install-notify-daemon` → `ailang daemon install --env prod`.
- **Remove the SessionStart hook** entry from `.claude/settings.json` (the daemon makes it redundant) AND delete `scripts/hooks/check_public_feedback.sh`. Acceptance criterion in design doc.
- **CHANGELOG.md** entry under v0.15.0.
- **Move design doc + sprint plan** from `design_docs/planned/v0_15_0/` to `design_docs/implemented/v0_15_x/` once shipped.

**Test surface**:
- Smoke script (manual gate, not CI).
- Unit test asserting that `ailang daemon run` correctly subscribes to all three goroutines (the public-feedback dedicated path).

**Acceptance**:
- `make ci` passes.
- Smoke script reports "✅ notification fired within 30s".
- `launchctl list | grep ailang.daemon` reports running.
- SessionStart hook removed; `ailang messages list --inbox public-feedback` (or equivalent) still works for batch review.
- CHANGELOG entry references both design doc and sprint plan.

**Files**:
- `scripts/smoke-notify-daemon.sh` (new)
- `docs/docs/guides/notify-daemon.md` (new)
- `Makefile` (edit — add target)
- `.claude/settings.json` (remove hook entry)
- `scripts/hooks/check_public_feedback.sh` (delete)
- `CHANGELOG.md` (edit)
- `design_docs/planned/v0_15_0/m-mac-notify-daemon.md` → `implemented/v0_15_x/`
- `design_docs/planned/v0_15_0/m-mac-notify-daemon-sprint-plan.md` → `implemented/v0_15_x/`
- (cross-repo follow-up if needed) `ailang-multivac/terraform/environments/prod/terraform.tfvars`

---

## Cumulative Estimates

| | LOC | Hours |
|---|---|---|
| M1 | 230 | 4 |
| M2 | 420 | 6 |
| M3 | 210 | 4 |
| M4 | 130 | 4 |
| **Total** | **990** | **18** |

≈ 2.5 working days at observed velocity.

## Day-by-Day Schedule

**Day 1**:
- AM: M1 (notify package + tests). 4h.
- PM: M2 first half — daemon scaffolding, single goroutine pulling SubEventsLaptop, dedup helper. 4h.

**Day 2**:
- AM: M2 second half — second + third goroutines (messages-laptop generic + public-feedback dedicated), --dry-run, exclude-list, ack-after-notify, tests. 4h.
- PM: M3 (plist template, install/uninstall/status, tests). 4h.

**Day 3**:
- AM: M4 (smoke script, prod tfvars verification, docs, Makefile target, hook removal, CHANGELOG, doc move). 4h.

## Risks

1. **Pub/Sub `pstest` package coverage gaps**: if `pstest` doesn't faithfully simulate ack semantics, the ack-after-notify test may need a different stub. Mitigation: fall back to a hand-rolled stub interface; minor extra test code (~30 LOC).
2. **terminal-notifier missing on contributor laptops**: the package falls back to osascript, but UX is degraded (no click action). Documented in M4.
3. **prod tfvars `messages-laptop` may need adjustment**: if the existing client_subscription doesn't filter to public-feedback, M2's third goroutine cannot work as designed. Mitigation: M2 falls back to in-process filtering on the generic messages-laptop subscription if no dedicated sub exists.

## Acceptance — sprint level

- [ ] All four milestones merged on `dev`.
- [ ] `make ci` passes.
- [ ] `launchctl list | grep ailang.daemon` reports running on Mark's laptop after `make install-notify-daemon`.
- [ ] A test feedback submission via `curl` to `mcp.ailang.sunholo.com/mcp/` produces a "🌐 External feedback" notification within 30s.
- [ ] `scripts/hooks/check_public_feedback.sh` and its `.claude/settings.json` entry removed.
- [ ] CHANGELOG.md v0.15.0 entry references sprint plan + design doc.
- [ ] Both docs land in `design_docs/implemented/v0_15_x/`.

## Cross-repo Coordination

This sprint may require **one terraform change in `ailang-multivac`** (add or adjust the `messages-laptop` `client_subscription` filter for public-feedback). M4 verifies this. If a change is needed, the sprint-executor will:

1. Open a PR against `ailang-multivac` (`sunholo-data:ailang-multivac`) using the standard branch-and-PR flow under the agent account.
2. Block on its merge and Cloud Build deploy before claiming M4 acceptance.
3. Document the cross-repo dependency in the sprint completion notes.

## Hand-off

On completion, sprint-executor sends a `plan_complete` message to `coordinator` with:
- `sprint_id: M-MAC-NOTIFY-DAEMON`
- `commits: [...]` (final SHAs)
- `chain_id` for the execution trace
- `next_actions: ["release v0.15.0 includes notify daemon", "monitor public-feedback notifications for first week"]`
