# M-COORD-MULTI-HOST-WORKERS: Bare-metal Host Workers in the Coordinator System

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium — operational reliability + multiplies the rig's value)
**Estimated**: 3-4 days
**Dependencies**: M-PUBSUB-MESSAGING (v0.9.0, implemented), M-CLOUD-DISPATCH (v0.9.0, implemented), M-EVAL-LOCAL-OLLAMA (v0.24.0, planned — provides the first concrete worker host)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Routing is deterministic given capability advertisements + message attributes; no AILANG-level determinism change |
| A2: Replayability | +1 | All routing decisions visible via heartbeat data + Pub/Sub message attributes; task replay across workers is feasible |
| A3: Effect Legibility | 0 | No AILANG effect surface change |
| A4: Explicit Authority | +1 | Workers explicitly advertise capabilities; nothing implicit. Reuses existing JWT auth for cross-host trust |
| A5: Bounded Verification | 0 | n/a |
| A6: Safe Concurrency | +1 | At-most-once execution enforced via Pub/Sub ack; duplicate delivery handled at existing message-id dedup layer |
| A7: Machines First | +2 | Machines can dispatch tasks to specific hardware without a human-in-the-loop |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Workers report active_tasks + uptime; per-task cost attribution flows through existing chains infrastructure |
| A10: Composability | +1 | Composes with existing executors (claude, codex, opencode, motoko, managed_agents); capability tags compose with existing inbox routing |
| A11: Structured Failure | +1 | "No worker available" is a typed parked-message outcome, not an unhandled error; capability mismatch observable |
| A12: System Boundary | +1 | Worker boundary explicit via hostname + capability list; Pub/Sub crossings authenticated and logged |

**Net Score: +9** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Routing is deterministic; no implicit nondeterminism
- [x] A3 (Effects): No hidden side effects in AILANG code
- [x] A4 (Authority): All worker capabilities explicit in config; no ambient access
- [x] A7 (Machines First): All worker discovery / dispatch is structured / machine-parseable

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

The coordinator already supports two execution venues:

1. **Local coordinator daemon** (the laptop) — picks up messages, runs Claude Code / Codex / opencode in git worktrees, commits results
2. **Cloud Run Jobs** (via `coordinator execute-job`) — same agent images, but ephemeral cloud workers spawned per task

Both subscribe to Pub/Sub topics for message delivery (M-PUBSUB-MESSAGING v0.9.0). What's MISSING is a stable third venue: **a peer bare-metal host (the Mac Studio rig) running its own coordinator daemon, registered into the same Pub/Sub mesh as a worker for tasks that require its specific hardware** (local Ollama models, GPU memory, the warm-cached AILANG-tuned variants).

**Current State (v0.21.0):**

The Mac Studio runs the AILANG eval rig (gemma4:26b-ailang on local Ollama, 17-benchmark smoke tier, ~5 min/trial, 78% pass rate at Iter 6). Today, it operates in one of three brittle modes:

- **Manual**: a developer SSHs/Tailscales in and runs `make eval-smoke MODELS=...` interactively
- **launchd cron** (via `dev.ailang.rig-watchdog` and ad-hoc `make eval-smoke` runs): rotations triggered on a schedule, but with no way to target the rig from another machine
- **Out-of-band Claude Code session**: someone (often this agent) drives the rig from a long-running VSCode Claude Code session sitting ON the Studio

None of these compose with the messages system. If your laptop is in another building and you want "run a smoke rotation on the rig with the latest compiler", you have no clean path. `ailang messages send eval-suite ...` from the laptop today lands in the laptop's inbox.

**Pain points:**

1. **No way to direct work at a specific host** from anywhere. `ailang messages send eval-suite "..."` from the laptop reaches the laptop's coordinator, not the Studio's.
2. **No capability-aware dispatch**. Even if both the laptop and Studio subscribed to the same inbox, there's no logic that says "this task needs `ollama:gemma4-26b`, route to the host that advertises it."
3. **No health visibility**. On 2026-05-24 we discovered Tailscale had been down for 38 hours because we had no heartbeat between the Studio and the rest of the system. A worker registry with `last_seen` would have caught it in minutes.
4. **No durability on restart**. The Studio's coordinator daemon doesn't survive reboots — we just lived through this when macOS Tahoe 26.5 force-installed and killed everything.
5. **No way for cloud Pub/Sub push to reach the Studio**. The Studio sits behind Tailscale with no public HTTPS endpoint — push subscriptions don't work; pull-only is the practical mode.

**Impact:**

- The rig's hardware investment is underutilized — it can only be driven by its physical user
- We cannot scale rig usage to other contributors without giving them SSH and walking them through the runbook
- We cannot run automated "trigger this rig from a GitHub Actions workflow" — a real future need (e.g. "when a parser change lands, run a smoke rotation")
- The PAR016/PAR017 weekend work would have been substantially faster if the laptop could just queue a rotation and walk away, instead of "be physically near the Studio"

## Goals

**Primary Goal:** Make the Mac Studio (and other bare-metal hosts) **first-class workers in the coordinator + Pub/Sub system** — addressable by capability tags, with heartbeat-based health, surviving reboots, and reachable from anywhere via Pub/Sub pull subscription.

**Success Metrics:**

- From the laptop, in another building: `ailang messages send eval-rig "run smoke n=3"` lands on the Studio's coordinator within 5 seconds, executes there using local Ollama, and writes results back to Firestore where the laptop can see them
- `ailang coordinator workers list` shows all registered hosts with their capabilities, last heartbeat, and current task count
- Studio's coordinator survives reboots (launchd plist), Tailscale outages (Pub/Sub pull continues working as long as Studio has any internet path), and macOS auto-updates
- Capability-aware routing: a task tagged `requires:ollama:gemma4-26b-ailang` is automatically dispatched to whichever worker advertises that capability; if none, the task is parked with a clear "no worker available" message rather than failing silently
- Zero net new transport — uses existing Pub/Sub + Firestore infrastructure

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Worker identity scheme: hostname-based (`studio.eval-rig`), UUID-based (`worker-7a3c...`), or capability-tag-only (`eval-rig@any-host`) | Affects message routing API, debuggability, and how multiple Studios are handled | human | design | high |
| Where capability tags are declared: per-agent in `~/.ailang/config.yaml`, in a new top-level `worker:` block, or autodetected (e.g. `ollama list` populates `ollama:*` tags) | Affects user ergonomics + drift risk | human | design | med |
| Pub/Sub subscription topology: one subscription per host vs one per agent (workers compete from a shared queue) vs hybrid | Affects fan-out vs targeting semantics; biggest architectural choice | human | design | high |
| What happens when a routed-by-capability task has no available worker — park, error, or fall back to a cloud worker | Affects user-visible behavior on cold mornings | human | design | low |
| Whether to bind worker identity to the Tailscale node identity | Tailscale-bound = simpler auth, but couples worker identity to a third-party service we just had issues with | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Identity scheme**: recommend **hostname-based** (`studio.eval-rig`, `laptop.coordinator`, `cloud.cloud-run`) — debuggable, stable, composes with capability tags. UUIDs are over-engineered for ≤5 hosts foreseeable.
- [ ] **Capability tags location**: recommend **per-agent in `~/.ailang/config.yaml`** with optional autodetect for `ollama:*` tags (run `ollama list` at coordinator start). Drift visible via heartbeats reporting current tags.
- [ ] **Subscription topology**: recommend **one subscription per `(host, agent_inbox)` pair**, with messages tagged by required capability set. Routes hit the host whose advertised capabilities ⊇ required set. Single message claimed by exactly one worker via Pub/Sub ack.
- [ ] **No-worker-available behavior**: recommend **park with explicit message** (not error, not auto-fallback). Cloud-fallback is a v0.25 follow-up.
- [ ] **Tailscale binding**: recommend **independent** — worker identity is `hostname + config-file UUID`, signed via existing M-CLOUD-ENDPOINT-AUTH JWT flow. Tailscale becomes one of several possible reach-the-Studio transports, not the auth mechanism.

## Solution Design

### Overview

Extend `AgentConfig` in [internal/coordinator/agent_registry.go](internal/coordinator/agent_registry.go) with a small set of fields for worker identity + capabilities. Extend the Pub/Sub adapter to subscribe to capability-filtered subscriptions (using existing attribute filtering). Add a heartbeat mechanism that writes to Firestore. Add a small `ailang coordinator workers` CLI surface. Wire a launchd plist for Studio-style hosts.

**No new transport, no new database, no new agent CLI.** Builds entirely on M-PUBSUB-MESSAGING primitives.

### Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│  ANY CLIENT (laptop, GitHub Action, cloud agent)                            │
│  ailang messages send eval-rig "run smoke n=3" --requires "ollama:gemma4-*" │
└───────────────────────────────────┬────────────────────────────────────────┘
                                    │ writes to Firestore + publishes to Pub/Sub
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Pub/Sub topic: ailang-messages (existing from M-PUBSUB)                    │
│  Per-message attributes (new):                                              │
│    inbox: eval-rig                                                          │
│    requires: ollama:gemma4-26b-ailang                                       │
└───────────────────────────────────┬────────────────────────────────────────┘
                                    │ subscriptions filter by inbox + requires
       ┌────────────────────────────┼───────────────────────────────────────┐
       ▼                            ▼                                       ▼
┌──────────────────┐  ┌───────────────────────────────┐  ┌────────────────────────────┐
│ laptop.eval-rig  │  │ studio.eval-rig                │  │ cloud.eval-rig             │
│ caps don't match │  │ caps: [ollama:gemma4-26b-      │  │ caps: [generic]            │
│ subscription     │  │        ailang, gpu:m4-max,     │  │ subscription doesn't match │
│ filter excludes  │  │        local-models]           │  │ requires:ollama:*          │
└──────────────────┘  │ MATCHES — pulls + acks         │  └────────────────────────────┘
                      └────────────┬──────────────────┘
                                   │
                                   ▼
                       ┌────────────────────────────┐
                       │ studio coordinator daemon  │
                       │   - launchd-managed        │
                       │   - heartbeat every 60s    │
                       │   - reports active_tasks   │
                       │   - executes via opencode  │
                       │     + local ollama         │
                       └────────────┬───────────────┘
                                    │
                                    ▼
                       ┌────────────────────────────┐
                       │ Firestore: result + spans  │
                       │ Pub/Sub: completion event  │
                       └────────────────────────────┘
```

The arrows from M-PUBSUB (existing): laptop → Pub/Sub → cloud workers. This design adds: laptop → Pub/Sub → bare-metal workers, with capability-filtered subscriptions doing the routing.

### Components

1. **Extended `AgentConfig`** ([internal/coordinator/agent_registry.go:100](internal/coordinator/agent_registry.go#L100)):
   - New field `WorkerCapabilities []string yaml:"worker_capabilities"` — what this agent can serve
   - New field `WorkerHostID string yaml:"worker_host_id"` — defaults to `os.Hostname()`; explicit override for tests
   - The existing `Capabilities []string` (documentation-only) is left untouched for backwards compat
   - Defaults: empty `WorkerCapabilities` ⇒ matches any message (current behavior); empty `WorkerHostID` ⇒ uses hostname

2. **Capability matcher** (new file `internal/coordinator/capability_matcher.go`):
   - Pure function `Matches(required []string, advertised []string) bool` — set-inclusion with glob support for tag families (e.g., `ollama:*` advertised matches `ollama:gemma4-26b-ailang` required)
   - Used by the Pub/Sub adapter to filter incoming messages

3. **Pub/Sub subscription filter** (extends [internal/coordinator/pubsub_adapter.go](internal/coordinator/pubsub_adapter.go)):
   - Subscription created with attribute filter: `inbox = "X"` (coarse, server-side)
   - Client-side filter: `Matches(message.requires, agent.advertised_capabilities)` before ack
   - Non-matching messages: NACK so Pub/Sub redelivers to another worker
   - Reuses existing M-PUBSUB subscription primitives — same attribute filtering model already documented for `agent_id` / `workspace` / `provider`

4. **Heartbeat writer** (new goroutine wired in [daemon.go](internal/coordinator/daemon.go)):
   - Every 60s, writes `{host_id, capabilities, active_tasks, last_seen, version, uptime}` to Firestore collection `worker_heartbeats`
   - TTL: 5 min — workers that don't heartbeat for 5 min are considered offline

5. **`ailang coordinator workers` CLI** (new subcommand in `cmd/ailang/coordinator_workers.go`):
   - `workers list` — shows all known workers, capabilities, last heartbeat
   - `workers tag <host_id> <add|remove> <tag>` — runtime capability mutation (writes to overlay, doesn't touch source config)
   - `workers ping <host_id>` — sends a `system:heartbeat`-tagged no-op task and reports round-trip latency

6. **launchd plist template** (`tools/coord/dev.ailang.coordinator.plist.template`):
   - `KeepAlive` on the coordinator daemon
   - Captures stdout/stderr to `~/.ailang/logs/coordinator-daemon.log`
   - Idempotent install via `tools/coord/install_daemon.sh`

### Implementation Plan

**Phase 1: Worker identity + capabilities in config** (~4 hours)
- [ ] Add `WorkerCapabilities`, `WorkerHostID` to `AgentConfig` struct
- [ ] Backwards-compatible defaults (empty = match-all / hostname)
- [ ] Unit tests for capability matcher (set inclusion + glob)
- [ ] Update `agent_config_test.go` with examples

**Phase 2: Pub/Sub subscription filter** (~6 hours)
- [ ] Extend `PubSubInboxAdapter` to accept `(host_id, capabilities)` at construction
- [ ] Server-side filter: `inbox = "X"`; client-side: capability subset check
- [ ] Ack messages only AFTER capability match; mismatch = nack
- [ ] Integration test: two mock workers, single task, exactly one claims

**Phase 3: Heartbeat + `workers` CLI** (~6 hours)
- [ ] Heartbeat goroutine writing to `worker_heartbeats` Firestore collection
- [ ] `ailang coordinator workers list` reads the collection
- [ ] `ailang coordinator workers ping` round-trip probe via `system:heartbeat` capability
- [ ] (Deferred to follow-up PR: a "Workers" panel in the Collaboration Hub)

**Phase 4: launchd persistence + Studio onboarding** (~4 hours)
- [ ] `tools/coord/dev.ailang.coordinator.plist.template`
- [ ] `tools/coord/install_daemon.sh` — generates plist with user paths, loads it
- [ ] Updates to the M-EVAL-LOCAL-OLLAMA runbook for the new onboarding path
- [ ] End-to-end test: install daemon on Studio, register `ollama:gemma4-26b-ailang` capability, send smoke task from laptop, verify Studio executes

**Phase 5: Documentation + acceptance** (~2 hours)
- [ ] `docs/docs/guides/coordinator-workers.md` — concept guide
- [ ] CHANGELOG.md entry under v0.24.0
- [ ] Update `.claude/rules/coordinator.md` with the new patterns

### Files to Modify/Create

**New files:**
- `internal/coordinator/capability_matcher.go` — pure capability matching (~80 LOC)
- `internal/coordinator/capability_matcher_test.go` — unit tests (~120 LOC)
- `internal/coordinator/heartbeat.go` — heartbeat writer goroutine (~120 LOC)
- `internal/coordinator/heartbeat_test.go` — (~80 LOC)
- `cmd/ailang/coordinator_workers.go` — `workers` subcommand (~150 LOC)
- `tools/coord/dev.ailang.coordinator.plist.template` — launchd template (~40 lines)
- `tools/coord/install_daemon.sh` — onboard script (~100 lines)
- `docs/docs/guides/coordinator-workers.md` — concept guide (~200 lines)

**Modified files:**
- `internal/coordinator/agent_registry.go` — extend AgentConfig (~10 LOC)
- `internal/coordinator/pubsub_adapter.go` — capability-aware subscription + ack (~80 LOC)
- `internal/coordinator/daemon.go` — heartbeat goroutine wiring (~20 LOC)
- `cmd/ailang/coordinator.go` — register `workers` subcommand (~10 LOC)
- `docs/docs/guides/coordinator.md` — link to new workers guide (~5 LOC)
- `CHANGELOG.md` — v0.24.0 entry (~10 LOC)
- `.claude/rules/coordinator.md` — pattern documentation (~15 LOC)
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — onboarding addendum (~30 LOC)

## Examples

### Example 1: Studio onboarding

**Before** (today): no clean way; the rig is driven by a human at the keyboard or by a Claude Code session sitting on the box.

**After:**

```bash
# On the Studio, one-time setup:
cd ~/dev/sunholo/ailang
tools/coord/install_daemon.sh \
  --capabilities "ollama:gemma4-26b-ailang,gpu:m4-max-40core,local-models"

# What it does:
#  - generates ~/Library/LaunchAgents/dev.ailang.coordinator.plist
#  - writes the worker_capabilities block into ~/.ailang/config.yaml
#  - launches the daemon with launchctl load
#  - emits a heartbeat to Firestore

# Verify from anywhere:
ailang coordinator workers list
# HOST              CAPABILITIES                                              LAST SEEN    TASKS
# studio.eval-rig   ollama:gemma4-26b-ailang, gpu:m4-max-40core, local-models  3s ago       0
# laptop.dev        code, research, docs                                       2s ago       1
```

### Example 2: Sending a capability-routed task

```bash
# From the laptop, no Tailscale needed (uses Pub/Sub):
ailang messages send eval-rig \
  '{"type":"eval-rotation","tier":"smoke","trials":3}' \
  --requires "ollama:gemma4-26b-ailang" \
  --title "Smoke N=3 on iter6 config"

# What happens:
#  1. Message lands in Pub/Sub with attributes {inbox: eval-rig, requires: ollama:gemma4-26b-ailang}
#  2. Studio's coordinator subscription matches; it claims and acks the message
#  3. Studio executes via local Ollama
#  4. Result lands in Firestore + a completion event publishes to Pub/Sub
#  5. Laptop's `ailang messages list --inbox eval-rig` shows the completion
```

### Example 3: Health visibility (the lesson from 2026-05-22..24)

```bash
ailang coordinator workers list --json
# [
#   {
#     "host_id": "studio.eval-rig",
#     "capabilities": ["ollama:gemma4-26b-ailang", "gpu:m4-max-40core", "local-models"],
#     "last_seen": "2026-05-24T13:42:01Z",
#     "active_tasks": 1,
#     "version": "0.24.0",
#     "uptime_secs": 7283,
#     "alive": true
#   },
#   {
#     "host_id": "laptop.dev",
#     "capabilities": ["code", "research", "docs"],
#     "last_seen": "2026-05-22T13:55:00Z",
#     "alive": false,
#     "reason": "no heartbeat for 38h7m"
#   }
# ]
```

This is the alert path that would have caught the Friday 2026-05-22 Tailscale outage in minutes instead of 38 hours.

## Success Criteria

- [ ] `AgentConfig` accepts `worker_capabilities` and `worker_host_id` fields; defaults are backwards-compatible
- [ ] Capability matcher unit tests cover: exact match, glob match (`ollama:*`), subset (advertised ⊇ required), no-match
- [ ] `PubSubInboxAdapter` filters messages by capability subset; non-matching messages are nack'd
- [ ] Integration test: two mock workers with different capabilities, single task with `requires`, exactly one claims it
- [ ] `ailang coordinator workers list` shows all known workers with heartbeat data
- [ ] `ailang coordinator workers ping <host>` does a round-trip and reports latency
- [ ] launchd plist + install script: Studio's coordinator survives `sudo reboot`
- [ ] End-to-end: from laptop, send a smoke rotation task targeting `requires:ollama:gemma4-26b-ailang`; verify it executes on Studio (not laptop, not cloud); result lands in Firestore
- [ ] Heartbeat persists across a Tailscale outage (Pub/Sub uses Google infrastructure, not Tailscale)
- [ ] All tests passing; lint clean
- [ ] CHANGELOG.md updated
- [ ] `docs/docs/guides/coordinator-workers.md` exists with the three examples above
- [ ] `.claude/skills/local-ollama-eval` runbook updated with the new onboarding path

## Testing Strategy

**Unit tests:**
- `capability_matcher_test.go` — exact, glob, subset, empty, malformed inputs
- `heartbeat_test.go` — heartbeat writes, TTL expiration, missing-host handling
- `agent_config_test.go` — defaults, backwards compat

**Integration tests:**
- `pubsub_adapter_capability_filter_test.go` — two mock workers, capability-routed message claims exactly once
- End-to-end test using Firestore emulator: heartbeat write + read via `workers list`

**Manual testing:**
- The end-to-end Studio scenario from Example 2 — from laptop, route a smoke task to Studio over Pub/Sub
- Studio reboot test: kill the daemon, reboot, verify launchd respawns within 60s
- Heartbeat outage test: stop the Studio's daemon, verify `workers list` shows it offline within 5 min

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Whether to surface workers in the Collaboration Hub UI** — out of scope for this milestone; v0.25 follow-up
- **How to handle worker auth secrets** — for now use the existing M-CLOUD-ENDPOINT-AUTH JWT flow; per-worker keys are a v0.25 hardening task — agent may choose
- **Whether to add automatic capability fallback** (e.g., if no worker has `ollama:gemma4-26b-ailang` but one has `ollama:gemma4-*`, accept the match) — agent may design but defer until we see real need
- **Whether `workers tag` mutates `~/.ailang/config.yaml` directly or writes to a separate runtime overlay** — agent may choose; recommendation is overlay (config remains source of truth, runtime tags visible separately)
- **Pub/Sub subscription provisioning** — auto-provision on `coordinator start` vs require explicit setup; agent may choose

## Non-Goals

**Not attempted in this feature:**
- **Cross-machine session continuity** (resuming a Claude Code session on a different host) — sessions remain host-local
- **Worker load balancing across multiple identically-capable hosts** — if you have two Studios with the same capabilities, both will see the message and the first to ack wins. Fair distribution is a v0.25 problem
- **Tailscale-based mesh discovery** — workers are configured explicitly, not auto-discovered. Auto-discovery is future hardening
- **GPU sharding within a single host** — one ollama instance per host; no concurrent model serving on the same GPU
- **Authentication/authorization beyond what M-CLOUD-ENDPOINT-AUTH already provides** — same JWT, same trust model
- **Solving the Tailscale.app 1.98.2 crash bug** (responsible for the 2026-05-22 outage) — that's a Tailscale.app bug; this work routes around it by using Pub/Sub for message delivery (independent of Tailscale)

## Timeline

**Day 1** (~4h):
- Phase 1 (config + capabilities)

**Day 2** (~6h):
- Phase 2 (Pub/Sub subscription filter)

**Day 3** (~6h):
- Phase 3 (heartbeat + CLI)

**Day 4** (~6h):
- Phase 4 (launchd) + Phase 5 (docs)
- End-to-end test laptop → Studio

**Total: ~22 hours over 3-4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pub/Sub attribute filter doesn't natively support subset semantics → client-side filtering causes redundant wakeups | Low | Capability tags are small; client-side filter is microseconds. Coarse inbox-name filter at Pub/Sub avoids broad fan-out |
| Workers compete and one claims tasks it can't actually execute (e.g. ollama not running) | Med | Per-task pre-flight check before ack; nack pushes message back. `system:heartbeat` ping path detects this proactively |
| launchd plist install script over-mutates user env / breaks other launchd jobs | Med | Idempotent installer with `--dry-run` flag; explicit plist filename; logs all writes; backup of any replaced file |
| Heartbeat collection grows unboundedly | Low | Per-host doc keyed by `host_id`; periodic GC sweep cleans docs older than 7 days |
| Two workers race on the same message | Low | Pub/Sub at-least-once delivery + ack semantics handle this — only one ack wins; duplicate delivery handled at message-id dedup (already present in M-PUBSUB) |
| Capability mismatch leaves messages parked forever | Med | Per-message TTL: if unclaimed after N hours, surface in `ailang messages list --orphaned` with clear remediation |
| Studio's Pub/Sub credentials drift (service account key rotation) | Low | Reuses M-CLOUD-ENDPOINT-AUTH flow; rotation cadence already documented there |

## Conflict Surface

**Not applicable.** This work touches `internal/coordinator/`, `cmd/ailang/coordinator*.go`, and new files under `tools/coord/`. No changes to `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`. No language-semantic surface touched.

The only "conflict surface" inside the coordinator is the message routing logic: routing-by-inbox-name continues to work unchanged; capability filters are additive. Agents with no `worker_capabilities` set behave exactly as today (match-everything).

## Related Documents

**Implemented foundations (this builds on):**
- [m-pubsub-messaging.md](../../implemented/v0_9_0/m-pubsub-messaging.md) — Pub/Sub transport layer. **Required reading.** This design extends its attribute-filter model.
- [m-cloud-dispatch.md](../../implemented/v0_9_0/m-cloud-dispatch.md) — Cloud Run Job dispatch. Workers here are local-execution analogues of the same pattern.
- [m-cloud-endpoint-auth.md](../../implemented/v0_9_0/m-cloud-endpoint-auth.md) — JWT auth flow used by workers.
- [M-AGENT-PROTOCOL.md](../../implemented/v0_5_0/M-AGENT-PROTOCOL.md) — Original messages system; capability tags extend its routing semantics.
- [m-cloud-eval-workers.md](../v0_13_0/m-cloud-eval-workers.md) — Earlier related design (Cloud-Run-specific eval workers); this doc generalizes the worker concept beyond Cloud Run.

**Companion v0.24.0 work:**
- [m-eval-local-ollama.md](m-eval-local-ollama.md) — Operational reliability for Studio's local Ollama setup. This doc adds the *control plane*; that doc handles the *data plane* of the rig.
- [m-eval-openrouter-baseline-rotation.md](m-eval-openrouter-baseline-rotation.md) — Once workers are real, OR baselines can be routed to whichever worker is cheapest/fastest.
- [m-eval-rating-efficiency.md](m-eval-rating-efficiency.md) — ELO scoring; cross-worker comparison becomes trivial when both run identical task payloads from the same queue.

**Operational reference:**
- `.claude/rules/coordinator.md` — Coordinator daemon rules
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — Studio rig operations (will gain an "as-worker" section)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [M-PUBSUB-MESSAGING](../../implemented/v0_9_0/m-pubsub-messaging.md) — the transport foundation
- [Pub/Sub attribute filtering](https://cloud.google.com/pubsub/docs/filtering) — for Phase 2's subscription filters
- [docs/internal/EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — executor contract this work composes with
- The 2026-05-22..24 Tailscale outage as motivation for heartbeat visibility

## Future Work

- **Cloud-fallback routing**: if no bare-metal worker matches required capabilities, fall back to Cloud Run with appropriate executor variant
- **Worker auto-discovery via Tailscale tags** — workers tagged `ailang-worker` in Tailscale ACLs auto-register
- **Capability negotiation**: workers report richer hardware metrics (memory headroom, current GPU utilization) so dispatch picks the best, not just any, matching worker
- **Cross-host session continuity**: resume a long-running Claude Code session on a different host
- **Worker pool fairness**: when two workers have identical capabilities, distribute tasks fairly via round-robin or load-aware
- **Per-task budget enforcement**: workers enforce per-task wall-clock and cost ceilings advertised at message-send time
- **A "Workers" panel in the Collaboration Hub UI** — same data as `workers list`, but real-time

---

**Document created**: 2026-05-24
**Last updated**: 2026-05-24
