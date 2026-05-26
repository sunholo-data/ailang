# Sprint Plan: M-COORD-MULTI-HOST-WORKERS

**Design doc**: [m-coord-multi-host-workers.md](m-coord-multi-host-workers.md)
**Target**: v0.22.0
**Status**: Approved by user (Design Freeze items confirmed 2026-05-25)
**Estimated**: 22 hours / 3-4 days
**Risk level**: Medium (touches Pub/Sub adapter — production message path)

## Design Freeze (Approved)

| Decision | Choice |
|---|---|
| Worker identity scheme | **Hostname-based** (`studio.eval-rig`, `laptop.coordinator`) |
| Subscription topology | **One per `(host, inbox)` pair** with client-side tag-subset check |
| No-matching-worker behavior | **Park with explicit `parked: no_worker_matched` status** |
| Where tags are declared | **Per-agent in `~/.ailang/config.yaml`** with optional `ollama:*` autodetect |
| Tailscale binding | **Independent** — auth via existing M-CLOUD-ENDPOINT-AUTH JWT |

## Recent velocity (data-driven)

From the last 14 commits (2026-05-22..25):
- Commits: 19
- Mix: 10 planning/docs/runbook + 3 parser code (PAR016/PAR017) + 5 leaderboards/reports + 1 prompt
- **Actual code delivered** (the 3 parser commits): ~50 LOC each, ~150 LOC total over a weekend
- **Realistic velocity**: ~200-300 LOC/day of actual code + tests when focused
- This sprint estimates ~1,200 LOC total. Achievable in 3-4 days of focused work.

## Sprint Goal

Make the Mac Studio (and other bare-metal hosts) first-class workers in the existing coordinator + Pub/Sub system. Zero new transports — extend `AgentConfig` with worker identity/tags, add tag-subset filtering to the Pub/Sub adapter, add heartbeats + `workers` CLI, and a launchd onboarding path.

**Concrete end-state win**: From the laptop, `ailang messages send eval-rig "smoke n=3" --requires ollama:gemma4-26b-ailang` routes to the Studio over Pub/Sub (no Tailscale dependency), executes there, results land in Firestore.

## Milestones

### M1: Worker identity + tags in AgentConfig (~4h, ~150 LOC)

**What**: Extend `AgentConfig` struct in [internal/coordinator/agent_registry.go](internal/coordinator/agent_registry.go) with two new fields:
- `WorkerTags []string yaml:"worker_tags"` — what this agent advertises
- `WorkerHostID string yaml:"worker_host_id"` — defaults to `os.Hostname()`

Plus a pure tag matcher in new file `internal/coordinator/tag_matcher.go`:
- `Matches(required []string, advertised []string) bool`
- Set inclusion + glob (`ollama:*` advertised matches `ollama:gemma4-26b-ailang` required)

**Acceptance criteria:**
- [ ] `AgentConfig` accepts new fields; empty defaults = match-all + hostname (backwards compat)
- [ ] `agent_config_test.go` exercises new fields including defaults
- [ ] `TagMatch` unit tests cover: exact match, glob match, subset, empty, malformed input
- [ ] All existing coordinator tests still pass

**Files:**
- NEW: `internal/coordinator/tag_matcher.go` (~80 LOC)
- NEW: `internal/coordinator/tag_matcher_test.go` (~120 LOC)
- MODIFY: `internal/coordinator/agent_registry.go` (~10 LOC)
- MODIFY: `internal/coordinator/agent_config_test.go` (~30 LOC)

**Dependencies**: None — pure schema + new file.

### M2: Pub/Sub subscription tag-filter (~6h, ~250 LOC)

**What**: Extend [internal/coordinator/pubsub_adapter.go](internal/coordinator/pubsub_adapter.go) to:
- Accept `(host_id, tags)` at adapter construction
- Subscription's server-side filter: `attributes.inbox = "X"` (coarse)
- Client-side filter: `TagMatch(msg.Attributes["requires"], adapter.tags)` before ack
- On mismatch: nack the message so Pub/Sub redelivers to another worker
- Message attributes parsed: `requires` (comma-separated tag list)

**Acceptance criteria:**
- [ ] PubSubInboxAdapter constructor takes host_id + tags
- [ ] Messages without `requires` attribute behave exactly as today (match all)
- [ ] Messages with `requires` are ack'd only on tag subset match
- [ ] Non-matching messages get NACK, observable in Pub/Sub metrics
- [ ] Integration test with Firestore emulator: 2 mock workers, single tag-routed task, exactly one claims
- [ ] No regression in existing Pub/Sub message flow

**Files:**
- MODIFY: `internal/coordinator/pubsub_adapter.go` (~80 LOC)
- NEW: `internal/coordinator/pubsub_adapter_tag_filter_test.go` (~170 LOC)
- MODIFY: `internal/messaging/sender.go` (~20 LOC) — add `--requires` propagation to Pub/Sub attributes

**Dependencies**: M1 (uses `TagMatch`).

**Risk note**: Production message path. Must keep test-after-each-change discipline; lint and the existing integration tests must remain green after every commit.

### M3: Heartbeat + unified `workers` CLI (~6h, ~400 LOC)

**What**:
1. **Heartbeat writer** (new file `internal/coordinator/heartbeat.go`):
   - Goroutine started by `daemon.go`
   - Every 60s, writes `{host_id, tags, active_tasks, last_seen, version, uptime_secs}` to Firestore collection `worker_heartbeats`
   - TTL: 5 min — workers with no heartbeat for 5 min are `alive: false`

2. **`ailang coordinator workers` subcommand** (new file `cmd/ailang/coordinator_workers.go`):
   - `workers list` — UNIFIED VIEW:
     - Reads `worker_heartbeats` (live bare-metal hosts)
     - Reads existing `tasks` Firestore collection (Cloud Run history, grouped by executor)
     - Aggregates with `type: bare-metal | cloud-run` column
   - `workers list --type bare-metal | cloud-run` — filter
   - `workers list --since 7d` — time window for Cloud Run history
   - `workers list --json` — machine-parseable output
   - `workers ping <host_id>` — round-trip probe via `system:heartbeat` tag

**Acceptance criteria:**
- [ ] Heartbeat goroutine starts/stops cleanly with the daemon
- [ ] `worker_heartbeats` collection has the right schema
- [ ] `workers list` shows BOTH bare-metal hosts (live) AND Cloud Run history (last 7d default)
- [ ] `workers list --json` parses cleanly
- [ ] `workers ping` returns round-trip latency for a live host
- [ ] `workers ping <non-existent-host>` exits non-zero with clear error

**Files:**
- NEW: `internal/coordinator/heartbeat.go` (~120 LOC)
- NEW: `internal/coordinator/heartbeat_test.go` (~80 LOC)
- NEW: `cmd/ailang/coordinator_workers.go` (~180 LOC)
- MODIFY: `internal/coordinator/daemon.go` (~20 LOC) — wire heartbeat goroutine
- MODIFY: `cmd/ailang/coordinator.go` (~10 LOC) — register `workers` subcommand

**Dependencies**: M1 (uses `WorkerTags`).

### M4: launchd persistence + Studio onboarding (~4h, ~250 LOC)

**What**:
1. **launchd plist template** (`tools/coord/dev.ailang.coordinator.plist.template`):
   - `KeepAlive` true so daemon respawns after crashes
   - stdout/stderr → `~/.ailang/logs/coordinator-daemon.log`
   - User-specific paths templated

2. **Onboarding script** (`tools/coord/install_daemon.sh`):
   - Idempotent — re-runs are no-ops
   - Takes `--tags "tag1,tag2,..."` (avoids `--caps` collision)
   - Generates plist from template with user's actual paths
   - Loads it via `launchctl load`
   - Writes/merges `worker_tags` block into `~/.ailang/config.yaml`
   - Optional `--dry-run` flag

**Acceptance criteria:**
- [ ] Plist template renders with correct paths via the install script
- [ ] `install_daemon.sh --tags "ollama:gemma4-26b-ailang" --dry-run` shows planned changes without applying
- [ ] After install, `launchctl list | grep ailang` shows the daemon
- [ ] `sudo reboot` on the Studio → daemon respawns within 60s
- [ ] Re-running `install_daemon.sh` on an already-onboarded host is a no-op (idempotent)
- [ ] On uninstall (`install_daemon.sh --uninstall`), daemon is fully removed

**Files:**
- NEW: `tools/coord/dev.ailang.coordinator.plist.template` (~40 lines)
- NEW: `tools/coord/install_daemon.sh` (~150 lines)
- NEW: `tools/coord/uninstall_daemon.sh` (~30 lines)
- NEW: `make/coord.mk` (~30 lines) — wraps install/uninstall as make targets

**Dependencies**: M1-M3 (needs the runnable daemon with new fields).

### M5: Documentation + end-to-end acceptance (~2h, ~250 LOC)

**What**:
1. **Concept guide** (`docs/docs/guides/coordinator-workers.md`):
   - Architecture diagram
   - Three worked examples (Studio onboarding, tag-routed task, health visibility)
   - Migration note: existing single-host setups unaffected (no `worker_tags` = match-all)

2. **CHANGELOG entry** under v0.22.0

3. **Updates to .claude/rules/coordinator.md** — new patterns

4. **Update to .claude/skills/local-ollama-eval/resources/rig_operations_runbook.md** — "Register Studio as a worker" section

5. **End-to-end test** (manual): from this Studio (or simulated laptop), use the actual deployed dev project to:
   - Set `worker_tags: ["ollama:gemma4-26b-ailang"]` on the Studio's coordinator
   - Start daemon
   - From elsewhere, send a tag-routed test message
   - Verify Studio claims, executes (a trivial echo task), and posts back

**Acceptance criteria:**
- [ ] `docs/docs/guides/coordinator-workers.md` exists with all 3 examples
- [ ] CHANGELOG updated
- [ ] `.claude/rules/coordinator.md` reflects new patterns
- [ ] Runbook updated
- [ ] End-to-end test passes (or documented "blocked on X")

**Files:**
- NEW: `docs/docs/guides/coordinator-workers.md` (~200 lines)
- MODIFY: `CHANGELOG.md` (~10 lines)
- MODIFY: `.claude/rules/coordinator.md` (~15 lines)
- MODIFY: `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` (~30 lines)
- MODIFY: `docs/docs/guides/coordinator.md` (~5 lines) — link to new guide

**Dependencies**: M1-M4 (documents what was built).

## Sequencing

```
M1 (foundation: schema + matcher) ─┬─► M2 (Pub/Sub filter, uses TagMatch)  ─┐
                                   └─► M3 (heartbeat + CLI, uses WorkerTags)─┴─► M4 (launchd uses daemon) ─► M5 (docs)
```

**M2 and M3 can run in parallel** if executed by sub-agents — they touch different files, both depend only on M1.

**M4 depends on M3** (the daemon must be heartbeat-capable for launchd to be useful).

## Total estimates

| Milestone | LOC est | Hours | Has tests | Acceptance criteria |
|---|---:|---:|---:|---:|
| M1 | 240 | 4 | 120 LOC | 4 |
| M2 | 270 | 6 | 170 LOC | 6 |
| M3 | 410 | 6 | 80 LOC | 6 |
| M4 | 250 | 4 | manual | 6 |
| M5 | 250 | 2 | manual | 5 |
| **Total** | **1420** | **22** | | **27** |

## Risk management

| Risk | Likelihood | Mitigation |
|---|---|---|
| M2 regresses existing message flow | Med | Each commit must pass existing pubsub tests; new tag-filter behavior is OPT-IN (no `requires` attribute = behave as today) |
| Firestore quota / cost on heartbeat writes | Low | 60s interval × N hosts is trivial; budget check in CI shows zero impact |
| launchd plist breaks user environment | Med | Idempotent installer with `--dry-run`; explicit plist filename; backup of any replaced file; explicit `--uninstall` |
| Pub/Sub attribute filters don't support subset semantics natively | Low | Already known — using coarse server filter + client-side check. Documented in design doc. |
| Two workers race on the same message | Low | Pub/Sub ack semantics + existing message-id dedup handle this |

## Success criteria for the whole sprint

Combining acceptance criteria from M1-M5:
- [ ] 27 milestone-specific acceptance criteria met (see per-milestone lists)
- [ ] All tests passing
- [ ] `make lint` clean
- [ ] `make check-file-sizes` clean (no file >800 lines)
- [ ] CHANGELOG updated
- [ ] `docs/docs/guides/coordinator-workers.md` published
- [ ] End-to-end test: Studio claims a tag-routed message and executes it

## Out of scope (deferred to follow-up)

- Collaboration Hub UI panel for workers (v0.25)
- Auto-fallback routing when no worker matches (v0.25)
- Per-worker auth keys (v0.25 hardening)
- Cross-host session continuity
- Tailscale-tag auto-discovery
- Worker pool fairness / load balancing

## Day-by-day plan

**Day 1** (~4h): M1 (config + matcher)
- Add fields to AgentConfig
- Write tag_matcher.go + tests
- Verify existing tests still pass
- Commit at end of day

**Day 2** (~6h): M2 (Pub/Sub filter)
- Extend PubSubInboxAdapter
- Add tag-filter integration test
- Verify production message flow unaffected
- Commit at end of day

**Day 3** (~6h): M3 (heartbeat + CLI)
- heartbeat.go + tests
- coordinator_workers.go + manual smoke
- Wire heartbeat into daemon
- Commit at end of day

**Day 4** (~6h): M4 + M5 (launchd + docs)
- plist template + install script
- Concept guide + CHANGELOG
- End-to-end test from a real client to the Studio
- Final commit + push

## Coordinator marker output

After this sprint is complete, the final commit should include:
- `Fixes` references to any GitHub issues this work closes
- Updated CHANGELOG.md under v0.22.0
- Design doc moved from `design_docs/planned/v0_22_0/` to `design_docs/implemented/v0_24_x/`

---

**Sprint plan created**: 2026-05-25
**Created by**: sprint-planner skill (claude-opus-4-7)
**Approved by**: user (Design Freeze items confirmed)
