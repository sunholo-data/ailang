# Sprint Plan: M-HARNESS-MISSION-LOOP, Sprint 1 (Phases 1–2)

**Design doc**: [m-harness-mission-loop.md](m-harness-mission-loop.md). HD-1..HD-6 were ratified by Mark (attended) on 2026-09-26.
**Scope**: Phase 1 (intake and scope guard) and Phase 2 (the fleet loop). Phase 3a/3b is out of scope; 3b stays gated on the 3a spike.
**Duration**: 3.5 days nominal (design estimate 3.5d plus about 15% buffer, and the buffer is absorbed by M5's live checks)
**Risk**: medium. The code is small; the risk is reach, meaning edits that land somewhere nothing reads (see mission-loop-change Gates 0–3).
**Worktree**: `.claude/worktrees/harness-mission-loop` (branch `sprint/harness-mission-loop`, base `origin/dev` 52a53a5d3)

## Facts established during planning (they shape the milestones)

| # | Fact | Consequence |
|---|---|---|
| F1 | `ailang messages send` refuses an inbox that is neither agent-served nor declared triage when the registry is authoritative (`cmd/ailang/messages_send_inbox_guard.go`) | `mission-fleet` must be declared in `triage_only_inboxes` in `ailang-multivac/config/config.cloud.yaml`, and it goes live only when the prod branch is promoted (Mark's step). Until then the ticket command passes the guard's force path explicitly, and says so |
| F2 | Inbox messages have only `unread`/`read`; `ack` = mark read; `list --json` does not mark read | Protocol: the fleet loop reads with `list`/`ticket open`, **never** `messages read`, and marks tickets read only when resolving. "Open" = unread in `mission-fleet` |
| F3 | Titles are deduped per inbox (`InboxMessageExistsByTitle`) | One message per **occurrence**, titled `[harness] <signature> · <mission>#<iter>`, with `correlation_id = harness:<signature>`. `slots_lost` = occurrences per signature, computed rather than updated (this settles the design's deferred decision) |
| F4 | No git hooks are installed in any mission clone, and git is 2.54 | The scope guard is a `pre-push` hook enabled **per fire** through `GIT_CONFIG_COUNT/KEY/VALUE` → `core.hooksPath`. No repo config is touched, and attended sessions are unaffected |
| F5 | Mission PR branches use a `mission/` prefix | A CI backstop is possible but is deferred (the hook comes first; see Deferred) |
| F6 | Skills are symlinks into the **main checkout working tree** (design V12) | Skill edits reach loops only when the main checkout is at or past the commit. The main checkout is currently behind origin with a dirty tracked `internal/proctree/child.pid`, so delivery needs Mark's OK to update it. This is reported and not worked around |
| F7 | `missions/*.toml` + `ailang mission install <name>`; `_mc_boot_offset` has a per-name case | Fleet gets a toml, an env file, a boot offset (1680) and a clone at `~/dev/sunholo-data/ailang-fleet` |

## Milestones

### M1: `ailang mission ticket` (file / open / resolve), about 380 LOC with tests
- `internal/mission/ticket.go`: the `Ticket` schema (mission, iteration, fire_started, signature, slot_verdict, blocking ∈ {none,item,all}, evidence ≤ 2 KB, workaround), `Validate`, `Title()`, `CorrelationID()`, and `GroupOpen([]InboxMessage) []OpenSignature` (ranked by occurrences desc, then oldest first).
- `cmd/ailang/mission_ticket.go`: `ticket file` (validates, inserts to `mission-fleet` through `openStore`, runs the inbox guard), `ticket open [--count|--json]` (does not mark read), and `ticket resolve <signature> --resolution TEXT [--sha SHA]` (replies to each origin `mission-<name>` inbox once, then marks every occurrence read).
- **Acceptance**: unit tests cover validation (bad blocking value, empty signature, oversized evidence truncated with a marker), stable titles, and grouping/ranking. An integration test against a local SQLite store runs file ×3 across 2 signatures → `open --count` = 2 → resolve one → `open --count` = 1, and the origin inbox holds the reply. Mutation check: drop the mark-read and the count test goes red.

### M2: scope guard, about 150 LOC
- `tools/launchd/githooks/pre-push`: when `MISSION_NAME` is set and the push target is `sunholo-data/ailang`, classify every changed path. A product mission (anything except `fleet`) touching a harness path is refused; `fleet` touching a product path is refused. The refusal message names `ailang mission ticket file`. Mission bookkeeping (`design_docs/<name>-mission*.md`) is always allowed. With no `MISSION_NAME` (an attended session) it does nothing. Pushes to other repos (world) do nothing.
- Driver: export `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0=$MC_DRIVER_ROOT/tools/launchd/githooks` before the controller spawn.
- `tools/launchd/test_mission_scope_guard.sh`, wired in `make/test.mk`: table (MISSION_NAME × path class × target repo) → allow/refuse, plus an end-to-end push into a bare repo proving the env-based hooksPath actually fires. Mutation-tested.

### M3: skill rules (Gate 0 and Gate 2), about 80 lines of markdown
- `gate-2-pick.md`: steps 2 and 5 become "file a ticket, record `[HARNESS] ticket:<signature>` in the queue row". Add a **fleet branch**: admissibility inverted; the queue = `ailang mission ticket open --json`; policy class (routing, quota, ration, billing guard, lane order) parks a decision row for Mark (HD-2a); no self-sourced audits.
- Gate 0 (product loops): read `mission-<name>` for `harness-resolved` replies and unpark the matching rows.

### M4: the fleet mission, about 250 lines (config and charter)
- `missions/fleet.toml` (interval 21600s, boot_offset 1680), `tools/launchd/mission-env/mission-fleet.env`, and a `fleet)` arm in `_mc_boot_offset`.
- Driver pre-check: `MISSION_NAME=fleet` and `ailang mission ticket open --count` = 0 → log `fleet: no open tickets — idle` and exit 0 **before probes**. If the count command fails, yield loudly (never treat a failure as zero). Driver test with stubbed `ailang` for count=0, count=2 and rc≠0.
- `design_docs/fleet-mission.md` charter (goal, authority row and scope, ranking, policy classes, done-gate = mission-loop-change pre-flight), plus seeded `fleet-mission-log.md` and `fleet-mission-index.md`.
- `ailang-multivac/config/config.cloud.yaml`: declare `mission-fleet` triage (a dev commit in that repo; the prod promote is Mark's).

### M5: live verification and docs
- Dry-run `MISSION_PROFILE=fleet` with 0 open tickets (idle exit, no probes) and with 1 planted ticket (proceeds to `DRY RUN ok`).
- Plant one ticket per mission (v1, docs, motoko, world) on the prod store with a `sprint-test:` signature; `open` shows 4 occurrences under 1 signature; resolve it; each origin inbox receives the reply.
- Install: clone `ailang-fleet`, copy the env to `~/.config`, `ailang mission install fleet`. **The kill switch `mission-fleet.disabled` stays in place**: turning the fleet loop on is Mark's call.
- Docs: CHANGELOG, the mission-loop-change skill (adds fleet, the scope guard, and the V15 dry-run pitfall), `docs/internal/message-plane-topology.md` (the `mission-fleet` inbox).
- `make test-launchd-drivers`, `go test ./internal/mission/... ./cmd/ailang/...`, and `make lint` green.

## Deferred
- CI backstop for `mission/*` PR branches (F5). Add it if the hook is ever bypassed (`--no-verify`).
- The candidate-SHA half of "open work" in the design's pre-check needs `harness-stable` (Phase 3b).

## Success metrics (sprint)
- All five milestones' acceptance checks pass. Each new gate has a positive control and a mutation check.
- Zero harness-path changes pushed by product loops after install. Measure in 14 days with the design's goal-1 method.
