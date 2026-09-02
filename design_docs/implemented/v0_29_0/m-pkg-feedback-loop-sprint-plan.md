# Sprint Plan: M-PKG-FEEDBACK-LOOP (Test + activate per-package feedback + macOS notifier)

## Summary

Validate the package-routing wire shipped in M-AGENT-MCP-ONBOARDING by exercising it end-to-end against test/prod Firestore, ship a feedback-shaped agent template (vs the release-sync `pkg-update.md`), audit other CLI commands for the same `AILANG_STORAGE=gcp` blind spot, and ship a minimum-viable macOS notifier so the user knows when feedback arrives — replacing the SessionStart polling hook with ambient notifications.

**Duration:** 1.5 days (5 milestones, ~550 LOC)
**Dependencies:** M-AGENT-MCP-ONBOARDING shipped to prod (`submit_feedback` accepts `package` + `auto_dispatch`); 11 `pkg-*` agents already configured in `ailang-multivac/config/config.cloud.yaml`
**Risk Level:** Low-medium — most work is validation + template authoring; M5 (macOS notifier) introduces launchd which is fiddly but well-documented

**Design doc:** [m-pkg-feedback-loop.md](m-pkg-feedback-loop.md)

---

## Current Status Analysis

### Completed (M-AGENT-MCP + M-AGENT-MCP-ONBOARDING)
- ✅ MCP catalog: 23 tools live at `https://mcp.ailang.sunholo.com/mcp/`
- ✅ `submit_feedback` accepts `package` + `auto_dispatch` args (M-AGENT-MCP-ONBOARDING)
- ✅ `openStore()` honors `AILANG_STORAGE=gcp` (one CLI command fixed; others still hardcoded)
- ✅ Firestore `(to_inbox, created_at)` index live in prod
- ✅ Cloud Run auto-redeploy on `:latest` SHA change works
- ✅ Self-test feedback `fb_b70e709844d62c97` confirmed in prod

### Velocity (last sprint)
- M-AGENT-MCP-ONBOARDING shipped 4 milestones in ~1 hour focused execution
- The dev → test → prod chain is well-rehearsed and now auto-deploys without manual intervention
- Build chain ~25 min per env

### Remaining from Design Doc
All 5 milestones scoped here. No prior partial work.

---

## Proposed Milestones

### M1 — Integration test harness for `submit_feedback` routing (~120 LOC, ~3h)

**Goal:** Prove that the `package` + `auto_dispatch` matrix actually routes correctly to Firestore (currently only smoke-tested locally).

**Tasks:**
- New `internal/feedback/integration_test.go` with build tag `//go:build integration` (excluded from `go test ./...` by default)
- 6 test cases:
  1. Default routing → `to_inbox=public-feedback`, `category=docs`
  2. `package=sunholo/auth, auto_dispatch=false` → `to_inbox=pkg:sunholo/auth`, `category=docs`
  3. `package=sunholo/auth, auto_dispatch=true` → `to_inbox=pkg:sunholo/auth`, `category=auto:docs`
  4. Invalid package format → `FieldError{Code: "invalid_package"}`
  5. Empty body → `FieldError{Code: "missing_field", Field: "body"}` (sanity check pre-existing validation still works alongside new args)
  6. Cleanup: TestMain deletes any test-tagged docs after run
- New `make test-feedback-integration` target (sets `AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac-test`, runs the tagged tests)
- Document in `internal/feedback/README.md` how to run locally + creds needed

**Acceptance:**
- [ ] `make test-feedback-integration` passes against test env Firestore
- [ ] All 6 test cases assert the right `to_inbox` + `category` shape
- [ ] Cleanup TestMain leaves test env tidy (verify by running twice — second run creates 0 net new docs)
- [ ] Integration tests are NOT included in `go test ./...` (build tag check)

---

### M2 — `pkg-feedback.md` agent template (~150 LOC, ~3h)

**Goal:** Give the package agents a template that does the right thing for user feedback (vs the release-sync `pkg-update.md` they have today).

**Tasks:**
- New `ailang-multivac/config/templates/pkg-feedback.md` — Anthropic-style prompt:
  - Reads inbox message payload (user's feedback body)
  - Branches on `category` from the Pub/Sub attributes:
    - `bug` → run package tests with snippet if provided; if reproduces, open GH issue with `bug` label + "reproduced ✅"
    - `feature` → file GH issue with `enhancement` label + "needs-design"
    - `docs` → check claim against package README; open docs PR if right, send "couldn't reproduce" reply if wrong
    - `limitation` → file in package's `design_docs/planned/`
  - Always ends with `submit_feedback`-shaped reply via `ailang messages send <user-inbox>`
  - Honors `auto_dispatch=false` (category lacks `auto:` prefix) → stops at "filed for triage"
- Update `ailang-multivac/config/config.cloud.yaml` to add `template_by_message_type:` block to all 11 `pkg-*` agents — feedback messages use the new template, everything else stays on `pkg-update.md`
- Update `internal/coordinator/agent_config.go` to honor `template_by_message_type` (currently only reads `template_file`)
- Verify `terraform plan` clean after the YAML change

**Acceptance:**
- [ ] `pkg-feedback.md` template exists at the right path
- [ ] All 11 `pkg-*` agents in `config.cloud.yaml` have `template_by_message_type: { feedback: pkg-feedback.md }`
- [ ] `internal/coordinator/agent_config.go` resolves the template per message type with `template_file` as the fallback
- [ ] `terraform plan` is clean (no resource drift from the YAML change)
- [ ] Sending a release-sync message still uses `pkg-update.md` (regression check)

---

### M3 — End-to-end loop test (~50 LOC, ~2h)

**Goal:** Prove the full loop works: `submit_feedback(package, auto_dispatch=true)` → cloud agent fires → GitHub issue lands → user notified.

**Tasks:**
- New `scripts/integration/test_pkg_feedback_loop.sh` — bash script:
  1. POST `submit_feedback` against test env MCP with `package=sunholo/auth, auto_dispatch=true, category=docs, body="Test loop run at $(date)"`
  2. Poll Firestore `pkg:sunholo/auth` inbox until message appears (timeout 10s)
  3. Poll Cloud Run logs for `pkg-sunholo-auth` agent fire (timeout 60s)
  4. Poll GitHub `sunholo-data/ailang-packages` for new issue with the timestamp in body (timeout 5min)
  5. Print PASS or FAIL with the specific stage that broke
- Capture per-stage timing in the output

**Acceptance:**
- [ ] Script runs end-to-end against test env
- [ ] Clean PASS prints all 4 stage timings
- [ ] On any stage breakdown, script fails loud with the specific stage + a debug command to inspect

---

### M4 — Cloud-mode CLI parity audit (~80 LOC, ~2h)

**Goal:** Other CLI commands (`chains`, `approvals`, `tasks`) currently hardcode SQLite — apply the same fix as M-AGENT-MCP-ONBOARDING M5.

**Tasks:**
- Grep `cmd/ailang/` for `messaging.OpenStore`, `coordinator.OpenStore`, similar patterns
- Apply the `storage.NewBackends` fix to each
- One regression test that mocks `AILANG_STORAGE=gcp` and asserts the right backend was constructed
- CHANGELOG entry listing each command now cloud-aware

**Acceptance:**
- [ ] At least 4 CLI commands fixed (chains, approvals, tasks, plus one bonus)
- [ ] Regression test passes
- [ ] `AILANG_STORAGE=gcp ailang chains list --limit 3` returns cloud-side chain data (not empty)

---

### M5 — Minimum-viable macOS feedback notifier (~200 LOC, ~3h)

**Goal:** Replace `check_public_feedback.sh` (SessionStart polling hook) with ambient macOS notifications when new feedback arrives. Carve-out from the broader `M-MAC-NOTIFY-DAEMON` design.

**Tasks:**
- New `cmd/ailang/daemon_notify_feedback.go` — `ailang daemon notify-feedback` subcommand
  - Subscribes to `messages-laptop` Pub/Sub subscription
  - Filters to `to_inbox` ∈ {`public-feedback`, `pkg:*`}
  - Dedup: `message_id` not re-notified within 5 min (in-memory map, keyed by ID, expires entries)
  - `--dry-run` flag for testing
- New `internal/notify/macos.go` — `Notify(n Notification) error`
  - Try `terminal-notifier` first (supports click actions)
  - Fall back to `osascript` display notification
  - Pattern from `scripts/hooks/session_end_speak.sh`
- New `scripts/ailang-daemon-notify-feedback.plist` — launchd config
  - StandardOutPath / StandardErrorPath under `~/Library/Logs/ailang/`
  - KeepAlive=true (auto-restart on crash)
  - RunAtLoad=true (start on login)
- New `make install-feedback-daemon` and `make uninstall-feedback-daemon` targets
- **Delete** `scripts/hooks/check_public_feedback.sh` and its entry in `.claude/settings.json` SessionStart array
- Document in `docs/docs/guides/agent-mcp.md`: how to install the daemon

**Acceptance:**
- [ ] `make install-feedback-daemon` succeeds; `launchctl list | grep ailang` shows it running
- [ ] Submit test feedback to test env → macOS notification fires within 5s
- [ ] `make uninstall-feedback-daemon` cleanly stops + removes the launchd entry
- [ ] Notification dedup: re-publishing the same message_id within 5 min doesn't fire a second notification
- [ ] `scripts/hooks/check_public_feedback.sh` removed and `.claude/settings.json` no longer references it
- [ ] `docs/docs/guides/agent-mcp.md` has install instructions

---

## Success Metrics (sprint-level)

- **All 5 milestones acceptance criteria pass**
- **Tests passing**: `make ci` clean (M1 integration tests are tagged-out by default), `make test-feedback-integration` passes when env is set
- **Documentation updated**:
  - [ ] `internal/feedback/README.md` (new) — integration test how-to
  - [ ] `docs/docs/guides/agent-mcp.md` — daemon install
  - [ ] `changelogs/v0.10-current.md` — M-PKG-FEEDBACK-LOOP entry
- **End-to-end loop works**: M3 script exits 0 against test env
- **User no longer needs to remember to check the inbox** — the notifier handles it ambiently

## Dependencies

- M-AGENT-MCP-ONBOARDING shipped (✅ done — `submit_feedback` accepts package + auto_dispatch)
- 11 `pkg-*` agents in `ailang-multivac/config/config.cloud.yaml` (✅ already configured)
- Test env is healthy (Firestore + MCP service running) — verify before M1
- `terminal-notifier` available on the laptop (`brew install terminal-notifier`) — fallback to osascript if not

## Open Questions

1. **Should M5 daemon also subscribe to events-laptop for task notifications?** Recommend NO for v1 — keep scope tight; the full daemon ships in M-MAC-NOTIFY-DAEMON.
2. **Should pkg-feedback template gate `bug` actions on test results?** Recommend YES — agent runs the package's tests; if the user's snippet reproduces, the agent has higher confidence opening the issue.
3. **What's the right test marker for cleanup?** Recommend `from_agent="mcp-public-test"` (distinguishable from real `mcp-public` traffic).

## Notes

- All 5 milestones can run in dependency-ordered sequence; M2 + M4 are independent of M1 + M3 + M5 and could parallelize. Sprint-executor decides.
- M5 deletes the SessionStart hook → make sure the daemon is installed before deleting the hook (M5 task ordering matters)
- The dev → test → prod chain for ailang-multivac changes (M2's config.cloud.yaml + the agent_config.go change) is well-rehearsed
- Tagging M1's integration tests with `//go:build integration` keeps `go test ./...` fast
