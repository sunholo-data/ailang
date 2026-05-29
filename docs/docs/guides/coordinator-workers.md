---
sidebar_position: 12
title: Coordinator Workers (Multi-Host)
---

# Coordinator Workers: Multi-Host Routing

**Status:** v0.22.0 (M-COORD-MULTI-HOST-WORKERS)
**Audience:** developers running an AILANG coordinator on a bare-metal host (e.g. a Mac Studio rig) AND on the team's cloud setup.

This guide covers the **worker-tag routing** layer that lets the same Pub/Sub topic carry messages to specific hardware (e.g. "this task needs `ollama:gemma4-26b-ailang`, route to the Studio"). It builds on the existing M-PUBSUB-MESSAGING and M-CLOUD-DISPATCH foundation — there is no new transport.

## TL;DR

```bash
# On the Studio (or any worker host):

# 1. Install the coordinator daemon as a launchd LaunchAgent
make coord-install \
    TAGS="ollama:gemma4-26b-ailang,gpu:m4-max-40core,local-models" \
    HOST_ID="studio.eval-rig"

# 2. Edit ~/.ailang/config.yaml to advertise these tags on the right agent
#    (the installer prints the exact YAML to add)
$EDITOR ~/.ailang/config.yaml

# 3. Daemon auto-starts at login + survives reboots
ailang coordinator status

# From anywhere:

# Send a tag-routed message via the HTTP API
curl -X POST http://localhost:1957/api/messages \
    -H "Content-Type: application/json" \
    -d '{
      "inbox": "eval-rig",
      "from": "user",
      "title": "smoke n=3",
      "requires": ["ollama:gemma4-26b-ailang"]
    }'

# See all known workers (bare-metal heartbeats + Cloud Run history)
ailang coordinator workers list
ailang coordinator workers list --json
ailang coordinator workers list --type bare-metal
ailang coordinator workers list --type cloud-run --since 24h
```

## How it works

```
ANY CLIENT                                          Pub/Sub
(laptop, GitHub Action, cloud agent)               ailang-messages
                                                    │
  POST /api/messages                                │ attributes:
  { ..., "requires": ["ollama:gemma4-..."] }        │   inbox: eval-rig
                ↓                                   │   requires: ollama:gemma4-...
        daemon stores + publishes                   ↓
                                       ┌────────────┴──────────────┐
                                       ▼                            ▼
                            ┌────────────────────┐    ┌────────────────────────┐
                            │ studio coordinator  │   │ laptop coordinator       │
                            │ advertises:         │   │ advertises:              │
                            │   ollama:gemma4-...│   │   code, docs              │
                            │ → tag-subset MATCH  │   │ → tag-subset MISMATCH    │
                            │ → ACK, process      │   │ → NACK, redeliver        │
                            └─────────┬──────────┘    └────────────────────────┘
                                      ↓
                            executes via local Ollama,
                            writes result to Firestore + Pub/Sub completion
```

The matching rule is **set inclusion with prefix-glob support**: every tag in `requires` must be satisfied by at least one advertised tag (exact match, or family glob like `ollama:*` covers `ollama:gemma4-26b-ailang`).

Workers with **no advertised tags** behave exactly as before for messages with no `requires` attribute (backwards-compat). But they **cannot claim** tag-routed messages — preserves the "untagged worker can't steal tag-routed work" guarantee.

## Three worked examples

### Example 1: Studio onboarding from a clean Mac

Run once on the Studio (when physically present, since launchctl needs interactive auth on first load):

```bash
# 1. Prereqs (one-time): gcloud + ADC + project
brew tap homebrew/cask && brew install --cask google-cloud-cli   # or use official installer
gcloud auth login
gcloud auth application-default login
gcloud config set project ailang-multivac-dev
gcloud auth application-default set-quota-project ailang-multivac-dev

# 2. Verify Firestore + Pub/Sub are reachable
gcloud firestore databases list >/dev/null   # should not error
gcloud pubsub topics list --limit 1 >/dev/null

# 3. Install the launchd LaunchAgent
make coord-install \
    TAGS="ollama:gemma4-26b-ailang,gpu:m4-max-40core,local-models" \
    HOST_ID="studio.eval-rig"

# 4. The installer prints YAML to add to ~/.ailang/config.yaml — paste it
#    onto the `eval-rig` (or whichever) agent block.

# 5. Verify
launchctl list | grep dev.ailang.coordinator
tail -f /tmp/ailang-coordinator-launchd.log
ailang coordinator status
```

### Example 2: Sending a tag-routed task from elsewhere

**v0.23.0 (M-COORD-TAG-ROUTING-LASTMILE)** added the `--requires` CLI flag, so you no longer need to hand-roll `curl POST` to send tag-routed messages. The CLI handles the HTTP plumbing.

```bash
# From the Studio shell (sender = receiver):
ailang messages send eval-rig "smoke n=3 on iter6 config" \
    --requires ollama:gemma4-26b-ailang \
    --from $(hostname) \
    --title "Run the smoke tier with the current PAR016+PAR017 compiler"

# From a laptop hitting the cloud coordinator:
AILANG_COORDINATOR_URL=https://your-cloud-coordinator.example.com \
COORDINATOR_API_KEY=$(gcloud secrets versions access latest \
    --secret=ailang-coordinator-api-key) \
ailang messages send eval-rig "smoke n=3 on iter6 config" \
    --requires ollama:gemma4-26b-ailang \
    --from laptop.dev

# Either way: the daemon's POST /api/messages stamps the `requires` tags
# as Pub/Sub attributes. The Studio's coordinator subscription matches the
# tag set, claims the message, executes via local Ollama, posts the
# completion back to Firestore + Pub/Sub. Other workers see the requires
# attribute, fail the match check, NACK, and stay idle.
```

The raw curl form still works and is the right tool for non-AILANG callers:

```bash
curl -X POST http://127.0.0.1:8765/api/messages \
    -H "Content-Type: application/json" \
    -d '{
      "inbox": "eval-rig",
      "from": "laptop.dev",
      "title": "smoke n=3",
      "content": "Run the smoke tier",
      "requires": ["ollama:gemma4-26b-ailang"]
    }'
```

#### HTTP endpoint configuration

v0.23.0 also moves the local daemon's HTTP listener from "opt-in via manual `PORT` env" to "on by default":

| Where | How |
|---|---|
| Default port | `8765` (binds 127.0.0.1) |
| Override at install time | `make coord-install PORT=9000` (or `tools/launchd/install_coordinator.sh --port 9000`) |
| Override at runtime | `AILANG_COORD_HTTP_PORT` or `PORT` env var (env wins over the plist) |
| Check it's up | `curl -s http://127.0.0.1:8765/health` → `{"status":"ok"}` |
| Or via CLI | `ailang coordinator status` shows `HTTP: ✓ http://127.0.0.1:8765` |

If you've installed the daemon before v0.23.0, the plist won't have `PORT` set and `--requires` will fail with an actionable error. Fix: `make coord-install` (idempotent — re-renders the plist with the new defaults).

#### Available routes

| Route | Auth | What |
|---|---|---|
| `GET /health` | open | Liveness probe, returns `{"status":"ok"}` |
| `POST /api/messages` | Bearer when `COORDINATOR_API_KEY` set; open otherwise | Submit a message; accepts `requires: [...]` for tag-routing |
| `GET /status` | Bearer | Daemon status JSON (mirror of `ailang coordinator status --json`) |
| `GET /pending` | Bearer | Tasks awaiting approval |
| `GET /chains/active` | Bearer | Currently-running chains |
| `POST /pubsub/push` | (cloud mode only) | Pub/Sub push receiver |
| `POST /github/webhook` | (cloud mode only) | GitHub webhook receiver |

The Bearer auth middleware is permissive on local-mode installs (no `COORDINATOR_API_KEY` env var) and strict on cloud deployments (Cloud Run secret-bound). Don't expose port 8765 publicly without setting `COORDINATOR_API_KEY`.

### Example 3: Health visibility — the lesson from 2026-05-22

On 2026-05-22, Tailscale.app crashed on the Studio and stayed down for 38 hours because nothing was watching for "the rig has gone silent." With heartbeats, this becomes a one-liner:

```bash
ailang coordinator workers list --json
# [
#   {
#     "host_id": "studio.eval-rig",
#     "type": "bare-metal",
#     "tags": ["ollama:gemma4-26b-ailang", "gpu:m4-max-40core", "local-models"],
#     "last_seen": "2026-05-24T13:42:01Z",
#     "active_tasks": 1,
#     "version": "v0.22.0",
#     "uptime_secs": 7283,
#     "alive": true
#   },
#   {
#     "host_id": "laptop.dev",
#     "type": "bare-metal",
#     "tags": ["code", "research", "docs"],
#     "last_seen": "2026-05-22T13:55:00Z",
#     "alive": false,
#     "reason": "no heartbeat for 38h7m"
#   }
# ]
```

A monitoring job that polls `workers list --json` and pages on `alive: false` for any expected host is a 10-line cron away.

## CLI reference

### `ailang coordinator workers list`

Unified view of all known workers — bare-metal hosts via the heartbeat backend, Cloud Run history via the existing task store.

| Flag | Default | Description |
|---|---|---|
| `--type` | (all) | `bare-metal` or `cloud-run` |
| `--since` | `7d` (168h) | Time window for Cloud Run history |
| `--max-age` | `5m` | Bare-metal staleness cutoff (hosts older than this are `alive: false`) |
| `--json` | off | Machine-parseable output |
| `--state-dir` | `~/.ailang/state` | Override coordinator state directory |

### `ailang coordinator workers ping <host_id>`

Round-trip probe a live worker. Currently does the heartbeat-store lookup; the full Pub/Sub round-trip via the `system:heartbeat` tag is deferred (see Future Work).

## Configuration

Worker advertisement lives in `~/.ailang/config.yaml` under each agent. Two new fields:

```yaml
coordinator:
  agents:
    - id: eval-rig
      label: "Mac Studio eval rig"
      inbox: eval-rig
      workspace: /Users/you/dev/sunholo/ailang
      provider: claude
      # M-COORD-MULTI-HOST-WORKERS: identify this host + what it can serve
      worker_host_id: studio.eval-rig
      worker_tags:
        - ollama:gemma4-26b-ailang
        - gpu:m4-max-40core
        - local-models
```

**Defaults preserve all existing behavior** — agents without `worker_tags` or `worker_host_id` behave exactly as in v0.23.x. Empty `worker_host_id` falls back to `os.Hostname()`. Empty `worker_tags` means the agent processes any message that has no `requires` attribute.

The terminology distinction matters: `worker_tags` are routing attributes, **NOT** AILANG's `--caps IO,FS` effect-system capabilities. Don't conflate the two.

## Hard-won lessons (from the 2026-05-22..24 incident)

1. **Pub/Sub is the only reliable transport.** Tailscale died, the Studio was unreachable on the LAN, but the daemon kept polling Pub/Sub (via Google's infrastructure, not Tailscale). Tag-routed messages would have continued to flow.
2. **macOS auto-updates kill long-running processes.** The launchd plist's `KeepAlive` recovers from that — but only if the daemon was installed via the LaunchAgent, not run by hand. Use `make coord-install`.
3. **iTerm2-spawned shells can carry stale state across crashes.** If you launch the coordinator from an interactive iTerm2 session, closing iTerm2 (or its parent crashing) takes the daemon with it. The LaunchAgent runs in its own process tree.
4. **gcloud is rarely in non-login bash PATH.** The installer's robust gcloud detection handles this — but watch for it if you write similar tooling.

## Implementation details

For the contracts and code paths underneath this guide:

- **Tag matching** (`internal/coordinator/tag_matcher.go`): pure `TagMatches(required, advertised) bool` with set-inclusion + prefix-glob (`ollama:*`). 17 unit tests cover edge cases including empty sets, case sensitivity, and the `system:heartbeat` probe tag.
- **Pub/Sub adapter filter** (`internal/coordinator/pubsub_adapter.go`): `HandleNotification` calls `parseRequiresAttr` + `shouldClaim` BEFORE the Firestore fetch. Mismatched messages return a non-nil error → Pub/Sub treats this as NACK → message is redelivered to another worker.
- **Heartbeat writer** (`internal/coordinator/heartbeat.go`): goroutine writing every 60s (configurable). The default `MemoryHeartbeatStore` works in-process; a `FirestoreHeartbeatStore` is on the v0.25 roadmap and slots into the same `HeartbeatStore` interface.
- **HTTP propagation** (`internal/coordinator/daemon_http.go`): `POST /api/messages` accepts a `requires` array on the JSON body and threads it through to `pubsub.MessageAttributes.Requires` → comma-separated `requires` attribute on the Pub/Sub message.

## Future work

- **Firestore-backed `HeartbeatStore`** so `workers list` from any host sees every worker
- **Per-task active-task tracking** on the daemon (snapshots currently report 0)
- **`workers ping` round-trip via Pub/Sub** with a `system:heartbeat` tag
- **Auto-fallback routing** to Cloud Run when no bare-metal worker matches
- **Collaboration Hub UI panel** for workers — same data as `workers list` but live
- **Auto-detected tags** at coordinator start (run `ollama list`, populate `ollama:*`)

## Related docs

- [coordinator.md](coordinator.md) — coordinator daemon basics
- [coordinator-setup.md](coordinator-setup.md) — setting up the coordinator for external projects
- [m-coord-multi-host-workers.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_22_0/m-coord-multi-host-workers.md) — the design doc this implements
- [m-eval-local-ollama.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_24_0/m-eval-local-ollama.md) — operational reliability for the Studio's eval rig (companion milestone)
