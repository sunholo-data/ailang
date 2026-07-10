# M-PKG-FEEDBACK-LOOP: Validate the per-package feedback loop end-to-end

**Status**: Planned
**Target**: v0.15.x (follow-up to M-AGENT-MCP-ONBOARDING)
**Priority**: P1 (closes the validation gap on the package-routing wire we just shipped — and unlocks the autonomous package agents to actually act on user feedback)
**Estimated**: 1.5 days, ~550 LOC
**Dependencies**: M-AGENT-MCP-ONBOARDING shipped (`submit_feedback` accepts `package` + `auto_dispatch` args), package agents already configured in `ailang-multivac/config/config.cloud.yaml`

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a **validation + activation** sprint for infrastructure that already exists. We're not adding new capability — we're proving the per-package feedback loop works end-to-end and giving the package agents a template that does the right thing for user feedback (vs the current pkg-update.md release-sync template).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A2: Replayability | **+1** | Test harness runs deterministically against test env Firestore; results auditable |
| A4: Explicit Authority | **+1** | `auto_dispatch=true` is the explicit handshake users opt into; default false maintains the current safe posture |
| A7: Machines First | **+2** | Closes the loop where an agent reports a bug → another agent triages it. Without this, feedback files but doesn't flow |
| A11: Structured Failure | **+1** | Test harness asserts the structured response shape and the routed-to inbox/category, not just "it didn't error" |
| A12: System Boundary | 0 | No new boundaries; tightens the contract on existing ones |

**Net Score: +5** → **Decision: Move forward**

---

## Problem Statement

### What we shipped (M-AGENT-MCP-ONBOARDING)

`submit_feedback(... , package, auto_dispatch)` — when `package` is set, routes to `pkg:<vendor>/<name>` inbox where the autonomous package agent watches. When `auto_dispatch=true`, the Pub/Sub notification gets `category=auto:<original>` so the coordinator can filter for "user authorizes immediate action".

### What we have NOT proven works

1. **Inbox routing actually lands at the right inbox in cloud-mode Firestore** — local smoke test passed but we didn't observe the cloud-side message
2. **The autonomous `pkg-sunholo-auth` agent (and 10 sister agents) actually fires** when a `pkg:sunholo/auth` message arrives — they were designed for AILANG release sync, not feedback
3. **The pkg-update.md template — the only template wired today — produces sensible output** when handed a feedback message instead of a release-sync trigger. Almost certainly no, but we've never verified the failure mode
4. **`auto_dispatch=true` actually changes behavior** at the coordinator side — the new `category=auto:` prefix has no consumer yet

### The deeper risk

If a user submits feedback today with `package=sunholo/auth, auto_dispatch=true`, the cloud agent **will fire on the message and run a release-sync template against bug-report content**. Best case: garbled output. Worst case: the agent attempts a "version bump" PR against ailang-packages based on the user's bug description.

We need the validation layer + a feedback-shaped template before encouraging anyone to flip `auto_dispatch=true`.

### Cloud-mode CLI parity (smaller, related)

`cmd/ailang/messages.go::openStore()` was the canonical example of a CLI command hardcoding SQLite. We just fixed that one — but the same pattern likely exists for `chains`, `approvals`, `tasks`, etc. Each one is a small fix; the bunch should be audited together.

---

## Proposed Plan

### M1: Integration test harness for `submit_feedback` routing

A Go integration test (`internal/feedback/integration_test.go`, build-tagged so it only runs in CI with GCP creds) that exercises the full publish path against test-env Firestore and asserts:

- Default routing → `to_inbox=public-feedback`, `category=<input>`
- `package=sunholo/auth, auto_dispatch=false` → `to_inbox=pkg:sunholo/auth`, `category=<input>` (NOT prefixed)
- `package=sunholo/auth, auto_dispatch=true` → `to_inbox=pkg:sunholo/auth`, `category=auto:<input>`
- Invalid package → `FieldError{Code: "invalid_package", Field: "package"}`
- All three valid submissions land as Firestore docs with the right `to_inbox`, `category`, `from_agent`, `message_type` attributes

Build tag `integration` keeps it out of `go test ./...` by default; CI runs it via a dedicated step with `AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac-test` against the test env (NOT prod — the test creates real Firestore docs and tags them with a `test_marker` for cleanup).

**Why test env not prod**: tests should not pollute the public-feedback inbox that humans triage. ailang-multivac-test has the same MCP service account + Firestore setup; perfect for this.

**Cleanup**: a `TestMain` that, after all tests, deletes any Firestore doc with `from_agent="mcp-public-test"` to keep the test env tidy across runs.

### M2: `pkg-feedback.md` agent template

The package agents need a template that recognizes "this is user feedback, not a release-sync trigger" and triages accordingly. Lives at `ailang-multivac/config/templates/pkg-feedback.md` (alongside the existing `pkg-update.md`).

The template should:

- Read the inbox message's payload (the user's feedback body)
- Branch on `category`:
  - `bug` → run package tests with the user's snippet if provided; if reproduces, open a GitHub issue in `sunholo-data/ailang-packages` with `bug` label and a "reproduced ✅" note
  - `feature` → file a GitHub issue with `enhancement` label and a "needs-design" tag
  - `docs` → check if the user's claim is right by reading the relevant package README; if yes, open a docs PR; if no, reply "couldn't reproduce" via `ailang messages send`
  - `limitation` → file in `design_docs/planned/` of the package
- Always end with a `submit_feedback`-shaped reply to the user via `ailang messages send <user-inbox>` with the action taken (closes the loop)
- `auto_dispatch=true` flag is honored: if false, the template stops at "filed for triage" without taking any external action

### M3: End-to-end loop test

Send a real `submit_feedback` against test env with `package=sunholo/auth, auto_dispatch=true, category=docs`, then poll for:

1. The Firestore doc lands in `pkg:sunholo/auth` inbox within 5s
2. The `pkg-sunholo-auth` cloud agent picks it up within 60s (per coordinator polling cadence)
3. A new GitHub issue appears in `sunholo-data/ailang-packages` with the right body within 5min
4. A reply message lands in a designated test-reply inbox confirming the agent's action

Wire the new template into `config.cloud.yaml` so `pkg-*` agents use `pkg-feedback.md` when `message_type=feedback` and `pkg-update.md` for everything else (the existing release-sync flow keeps working). Do this via a new YAML field `template_by_message_type:` since the coordinator config already handles per-message-type dispatch.

If the test fails, fail loudly with the specific stage that broke — that's the actionable signal.

### M4: Cloud-mode CLI parity audit

Grep `cmd/ailang/` for other functions that hardcode `messaging.OpenStore(dbPath)` or `coordinator.OpenStore(dbPath)` without consulting `AILANG_STORAGE`. Apply the same fix as M-AGENT-MCP-ONBOARDING M5 (the openStore fix) — route through `internal/storage/backend.go::NewBackends` when the env var is set.

Suspects to audit:
- `chains` — `ailang chains list`, `ailang chains show`
- `approvals` — `ailang approvals list`
- `tasks` — `ailang tasks list`
- `coordinator` daemon code
- Any other "open my local DB" pattern

Each fix is ~5-15 LOC. Estimate ~5-7 commands need the same change. Total ~80 LOC across all of them. Add a regression test that picks one CLI command and asserts it honors AILANG_STORAGE=gcp by checking which env var was read.

### M5: Minimum-viable notifier for new feedback (carve-out from M-MAC-NOTIFY-DAEMON) — **SUPERSEDED**

> **Status (2026-05-04): Superseded by full [M-MAC-NOTIFY-DAEMON](m-mac-notify-daemon.md), pulled forward from v1.0.0 to v0.15.0.** The full sprint subsumes this carve-out: the public-feedback handler ships as part of the umbrella `ailang daemon` command, and removal of `scripts/hooks/check_public_feedback.sh` is handled there. This M5 section is retained for historical context only and **must not be implemented separately** — sprint-executor should treat this milestone as already completed via the full daemon sprint.

The user's immediate need is "tell me when new public-feedback or pkg:* feedback arrives". The full M-MAC-NOTIFY-DAEMON design (planned, not implemented) covers task approvals, completions, registry events, etc. — too big for this sprint.

This milestone ships the **minimum viable** subset:

- New `ailang daemon notify-feedback` subcommand (NOT `ailang daemon` — leave that name for the full daemon later)
- Subscribes to `messages-laptop` Pub/Sub subscription, filters to `to_inbox` ∈ {`public-feedback`, `pkg:*`}
- Fires macOS notification via `terminal-notifier` (with `osascript` fallback) — pattern lifted from `scripts/hooks/session_end_speak.sh`
- Title: `"✉️ AILANG: $inbox"` (e.g. `"✉️ AILANG: pkg:sunholo/auth"`)
- Body: `"$title — $from_agent"`, capped at 200 chars
- Click action: `open https://dashboard.ailang.sunholo.com/inbox/$inbox` (assuming the dashboard supports inbox-deep-link; if not, opens the inbox list)
- Dedup: same `message_id` not re-notified within 5 min (rare with Pub/Sub at-least-once but worth guarding)
- launchd plist at `scripts/ailang-daemon-notify-feedback.plist` so it auto-starts on login + restarts on crash; `make install-feedback-daemon` target sets it up
- `--dry-run` flag logs without firing notifications (for testing)
- Removes `scripts/hooks/check_public_feedback.sh` and its entry in `.claude/settings.json` SessionStart — its role is now covered by ambient notifications instead of session-start polling

**Acceptance**:
- After `make install-feedback-daemon`, `launchctl list | grep ailang` shows it running
- Submit a test feedback to test env's pkg:sunholo/auth → mac notification fires within ~5s
- Killing the daemon via `launchctl unload` and re-loading restores notifications without duplicates
- check_public_feedback.sh removed from .claude/settings.json; no orphan files left

**Out of scope for this M5** (deferred to full M-MAC-NOTIFY-DAEMON):
- Task approval / completion / failure notifications
- Registry-publish notifications  
- Per-event notification config (notify_excludes.conf)
- Sound/Group customization
- The `ailang daemon` umbrella command (this just adds `notify-feedback` as a subcommand)

---

## Implementation Plan (1.5 days, ~550 LOC)

### M1 — Integration test harness (~120 LOC, ~3h)

- New file `internal/feedback/integration_test.go` with build tag `//go:build integration`
- 6 test cases covering the matrix above
- TestMain cleanup function
- New `make test-feedback-integration` target (separate from `make test`) that sets the env vars and runs `go test -tags integration ./internal/feedback/`
- Document in `internal/feedback/README.md` how to run locally + what creds are needed

**Acceptance**: `make test-feedback-integration` passes against test env Firestore. CI gets a new optional job (manual trigger only — costs Firestore writes; not a per-PR check).

### M2 — pkg-feedback.md template (~150 LOC, ~3h)

- New `ailang-multivac/config/templates/pkg-feedback.md`
- Update `ailang-multivac/config/config.cloud.yaml` to add `template_by_message_type:` to all 11 `pkg-*` agents — feedback messages use the new template, everything else stays on pkg-update.md
- Update `internal/coordinator/agent_config.go` to honor `template_by_message_type` (currently it only reads `template_file`). Defaults to `template_file` when no override matches.
- Test the YAML schema doesn't break the bootstrap flow — `terraform plan` should be clean

**Acceptance**: Submitting feedback to `pkg:sunholo/auth` with `auto_dispatch=true` causes the cloud agent to spawn with `pkg-feedback.md` (verified via Cloud Run logs); release-sync messages still use `pkg-update.md`.

### M3 — End-to-end loop test (~50 LOC, ~2h)

- New script `scripts/integration/test_pkg_feedback_loop.sh` that submits feedback against test env and polls
- Capture timing of each stage (submit → inbox → agent fire → GitHub issue)
- Document expected vs actual; fail loud on any stage breakdown

**Acceptance**: Script run produces a clean PASS or a clear FAIL pointing at the broken stage.

### M4 — Cloud-mode CLI parity audit (~80 LOC, ~2h)

- Grep + fix: `chains`, `approvals`, `tasks`, plus any others surfaced
- One regression test that asserts the env-var honor (mock the env, check which backend was constructed)
- Update CHANGELOG with the list of CLI commands now cloud-aware

**Acceptance**: `AILANG_STORAGE=gcp ailang chains list`, `ailang approvals list`, `ailang tasks list` all return cloud data instead of empty SQLite results.

### M5 — Minimum-viable feedback notifier (~200 LOC, ~3h)

- New `cmd/ailang/daemon_notify_feedback.go` — `ailang daemon notify-feedback` subcommand
- New `internal/notify/macos.go` — terminal-notifier + osascript fallback (pattern from `scripts/hooks/session_end_speak.sh`)
- New `scripts/ailang-daemon-notify-feedback.plist` — launchd config
- New `make install-feedback-daemon` / `make uninstall-feedback-daemon` targets
- Delete `scripts/hooks/check_public_feedback.sh` + remove its entry from `.claude/settings.json` SessionStart array
- Document in `docs/docs/guides/agent-mcp.md`: how to install the daemon for laptop notifications

**Acceptance**:
- `make install-feedback-daemon` sets up launchd; `launchctl list | grep ailang.feedback` shows it
- Test submission produces a macOS notification within 5s
- Old hook removed cleanly

---

## Risks & Tradeoffs

1. **Cost of running M3 against test env** — Cloud Build minutes + Firestore writes. Mitigation: keep M3 manual-trigger only, not per-PR.
2. **`pkg-feedback.md` template quality** — first version will probably be too aggressive (close-as-duplicate too eagerly) or too passive. Mitigation: start with category=docs only auto-dispatching; bug/feature/limitation file-and-stop in v1.
3. **CI flakiness from network calls** — M1 and M3 talk to live Firestore. Mitigation: M1 isolates with build tag; M3 is opt-in; both surface a clear "creds missing" message rather than failing silently.
4. **Bigger CLI parity gap than expected** — if 10+ commands need the fix, scope creeps. Mitigation: cap at the 4 named (chains/approvals/tasks + one bonus); document remaining as a separate cleanup pass.

## Out of Scope (for v1)

- Read-feedback MCP tool (still doesn't make sense in the public-facing MCP — would need auth)
- Per-package custom prompts in pkg-feedback.md (one universal template is fine for v1)
- Full GitHub issue dedup (let the existing `ailang messages send --github` handle duplicate detection)
- Replacing pkg-update.md (it's still the right template for AILANG-release-driven sync work)

## Open Questions

1. **Should `auto_dispatch=true` require a higher rate limit** to prevent abuse (a user could spam authorized actions)? Recommend: no for v1, the LB-level 5/min already covers; revisit if abuse appears.
2. **Should the reply message land in a public inbox or DM the contact field?** Recommend: a public `pkg-feedback-replies` inbox by default; DM to `contact` only if the user provided a specific channel and opted in via a future `reply_via` arg.
3. **Should the M3 test run on every prod release as a smoke check?** Recommend: yes, add to `post-release` skill — it's the canonical "is the feedback loop healthy" signal.

---

## Success Metrics

- **M1**: 6 integration tests passing, exits 0 after cleanup leaves the test env tidy
- **M2**: `terraform plan` clean after the config change; release-sync still works (regression check)
- **M3**: end-to-end loop completes in <5min, GitHub issue opens with right labels + body
- **M4**: 4+ CLI commands now honor `AILANG_STORAGE=gcp` (verified by grep)
- **Sprint outcome**: package agents are now safe to enable for real user feedback — `auto_dispatch=true` produces sensible action; `auto_dispatch=false` files cleanly. Closes the loop the M-AGENT-MCP-ONBOARDING ship started.

---

## References

- [M-AGENT-MCP](m-agent-mcp-website.md) — original sprint introducing submit_feedback
- [M-AGENT-MCP-ONBOARDING](m-agent-mcp-onboarding.md) — added package + auto_dispatch args (current state)
- [`internal/feedback/publisher.go`](../../../internal/feedback/publisher.go) — publish path being tested
- [`ailang-multivac/config/config.cloud.yaml`](../../../../ailang-multivac/config/config.cloud.yaml) — package agent definitions (11 pkg-* agents)
- [`internal/messaging/config.go::RepoForInbox`](../../../internal/messaging/config.go) — pkg: prefix routing logic that's already wired
