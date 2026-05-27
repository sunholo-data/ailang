# M-COORD-TAG-ROUTING-LASTMILE: Close the last-mile gaps preventing tag-routed dispatch

**Status**: IMPLEMENTED
**Target**: v0.23.0
**Priority**: P1 (Medium) — M-COORD-MULTI-HOST-WORKERS v0.22.0 shipped the routing machinery but a real user can't yet send a tag-routed message end-to-end without curling JSON
**Estimated**: 1-2 days (3 small, decoupled gap closures + a Cloud Run Job + verification matrix)
**Dependencies**:
- M-COORD-MULTI-HOST-WORKERS (v0.22.0, implemented) — provides `worker_tags`, tag matcher, heartbeat store, `workers list` CLI, and `requires` HTTP body field
- M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0, implemented) — provides the `motoko` executor for cloud-mode dispatch
- M-MOTOKO-V021-EFFECT-ROW-MIGRATION (v0.22.0, implemented) — motoko_agent now builds + runs cleanly on AILANG v0.22+, verified via 4/4 eval-smoke runs

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism. Tag routing is deterministic given the set of advertised tags (subset-inclusion + prefix-glob match — both pure functions). |
| A2: Replayability | 0 | No trace impact; tag-routing decisions are recorded in the existing message audit log. |
| A3: Effect Legibility | 0 | No new effects. `--requires` is just an argv parse → JSON field; HTTP and Pub/Sub effects already declared. |
| A4: Explicit Authority | +1 | Reinforces explicit authority: messages declare `requires` upfront, workers declare `worker_tags`, dispatch is set-membership of explicit declarations on both sides. No ambient routing. |
| A5: Bounded Verification | 0 | Pure tag-matcher is already covered by 26 unit tests (M-COORD-MULTI-HOST-WORKERS). New CLI flag has parser-side coverage but no new analysis required. |
| A6: Safe Concurrency | +1 | Goroutine writer (heartbeat) already in place; the new Cloud Run Job inherits the established Pub/Sub ack/nack contract (NACK on tag mismatch causes redelivery to the next eligible worker). |
| A7: Machines First | +1 | `--requires` makes routing intent legible at the CLI/JSON level — agents writing tasks can self-document which capability they need without an out-of-band convention. |
| A8: Minimal Syntax | +1 | Reuses existing flag-style conventions on `messages send` (no new keywords, no new file formats — just an additive arg). |
| A9: Cost Visibility | 0 | No cost-model changes. The cloud motoko Job inherits the existing per-Job billing-flag pattern. |
| A10: Composability | +1 | Strengthens composition between the messaging layer and the Cloud Run Jobs catalog: a single new entry in `cloudbuild-dev.yaml` + a one-line `agent_cli: motoko` in models.yml is all that's needed for full integration. |
| A11: Structured Failure | +1 | All three gaps surface failures more clearly than the status quo: HTTP-listener-off currently fails silently (port not bound); CLI gap currently rejects valid JSON at parse time; cloud-motoko-job-missing currently dispatches to "nowhere" with a confusing "no eligible worker" message after Pub/Sub retry exhaustion. |
| A12: System Boundary | +1 | Cloud Run Job binding crystallises the cost-control boundary (OPENROUTER_API_KEY is bound, ANTHROPIC_API_KEY explicitly is NOT, per the [EXECUTOR_SHAPE.md §8](../../docs/internal/EXECUTOR_SHAPE.md) rule that already governs every other executor Job). |

**Net Score: +7** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check
- [x] A1: No implicit nondeterminism
- [x] A3: No hidden side effects
- [x] A4: No ambient authority — the opposite, makes it more explicit
- [x] A7: Optimises for machine legibility (structured argv → JSON → routing)

## Problem Statement

After M-COORD-MULTI-HOST-WORKERS shipped in v0.22.0, the Mac Studio rig advertises `agent:motoko` (among 10 other tags) and the routing primitives are all in place (tag matcher, Pub/Sub attribute encoding, heartbeat store, unified `workers list`). The end-to-end **send** path, however, has three concrete gaps that prevent a real user from typing `ailang messages send eval-rig "task" --requires agent:motoko` and having it actually land:

### Gap 1: Local daemon HTTP listener is off by default

`internal/coordinator/daemon.go:127-150` starts the health/messages HTTP server only when the `PORT` env var is set:

```go
if port := os.Getenv("PORT"); port != "" {
    go d.startHealthServer(port)
}
```

The `dev.ailang.coordinator.plist` launchd template (installed by `make coord-install`) does NOT set `PORT`. So on a local-mode Studio, the HTTP routes (`/api/messages`, `/health`, `/status`, etc.) are not reachable. The CLI's local-only SQLite path still works for fire-and-forget queuing, but tag-routed sends — which require the JSON body with the `requires` array — have nowhere to land. Confirmed on this Studio: `lsof -p <coordinator-pid> -nP | grep LISTEN` shows zero TCP listeners.

Cloud Run mode doesn't have this gap because Cloud Run mandates `PORT` for any HTTP service container, so the cloud `ailang-coordinator` service exposes `/api/messages` automatically.

### Gap 2: `ailang messages send` CLI doesn't accept `--requires`

The CHANGELOG entry for M-COORD-MULTI-HOST-WORKERS explicitly says:

> Send tag-routed messages via HTTP `POST /api/messages` with a `requires: ["..."]` array (the CLI `messages send` does not yet support `--requires` — uses SQLite-only path).

`cmd/ailang/messages_send.go` only exposes `--title`, `--from`, `--type`, `--github` (and a few session flags). There's no path to set `requires` from the CLI, forcing users to `curl -X POST http://...` with a hand-rolled JSON body. That's a poor enough UX that nobody in practice will use the tag-routing primitives — they'll fall back to the typed-inbox path (`ailang messages send eval-rig "task"` directly addresses the eval-rig inbox), which works today but doesn't exercise the routing layer that v0.22.0 built.

### Gap 3: No cloud motoko fallback — `cloudbuild-dev.yaml` doesn't build the image, no Cloud Run Job exists

`docker/Dockerfile.agent-motoko` has been in the repo since M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0). But `cloudbuild-dev.yaml` builds only `coordinator, dashboard, mcp, agent-base, agent, agent-go` — no `build-agent-motoko` step. And the Cloud Run Jobs catalog in `ailang-multivac` contains `ailang-agent-executor-{claude,codex,gemini,go,go-apikey,eval,eval-go,apikey,codex-apikey,gemini-go}` but **no `ailang-agent-executor-motoko`**.

Consequence: the cloud `ailang-coordinator` Cloud Run service is fully capable of receiving a `requires:["agent:motoko"]` message (HTTP listener is on, multi-host routing code is in v0.22.0 which the cloud service auto-deployed on dev push), but it has only one place to dispatch the work — the Studio via Pub/Sub. If the Studio is offline or fails the task, there's no cloud-side fallback. This is explicitly named as item #1 in M-COORD-MULTI-HOST-WORKERS's Future Work section:

> Cloud-fallback routing: if no bare-metal worker matches required tags, fall back to Cloud Run with appropriate executor variant

The motoko executor adapter (`internal/executor/motoko/`) is already coordinator-ready — it implements the standard `Executor` interface and is auto-discovered by the dispatch layer. It just needs an image + Job to actually run in.

## Goals

**Primary goal**: A user types `ailang messages send eval-rig "Write tests for parse.ail" --requires agent:motoko` from any host and the message is claimed by whichever worker (Studio bare-metal OR ephemeral Cloud Run Job) advertises `agent:motoko`. End-to-end success path is observable via `ailang chains view <id>`.

**Success metrics:**
1. `lsof -p <local-coordinator-pid>` shows a TCP listener on the configured port within 60s of `launchctl bootstrap`
2. `ailang messages send eval-rig "x" --requires agent:motoko` produces a message in the store with `requires:["agent:motoko"]` populated (verifiable via `ailang messages read <id> --raw`)
3. A `--requires agent:motoko` message sent **from the Studio** to itself completes successfully (Studio claims its own task via tag match)
4. A `--requires agent:motoko` message sent **from any other host** to the cloud `ailang-coordinator` URL with the API key reaches the Studio via Pub/Sub and completes
5. A `--requires agent:motoko` message sent **while the Studio is intentionally stopped** (`launchctl bootout`) is claimed by the cloud `ailang-agent-executor-motoko` Job and completes (cloud fallback)

## Solution Design

### Overview

Three decoupled gap closures, each landing as an independent commit:

1. **M1 — Local HTTP listener via launchd `PORT` env**: edit the plist template + installer script, document the new default port, extend `ailang coordinator status` to report the bound listener
2. **M2 — `--requires` CLI flag**: small Go change in `cmd/ailang/messages_send.go` to accept `--requires` and thread it into the existing message construction path; identical for both the SQLite local path and the HTTP path
3. **M3 — Cloud motoko executor Job**: 4 sub-tasks landing as one commit — `build-agent-motoko` step in `cloudbuild-dev.yaml`, push step, Cloud Run Job definition (Terraform), and `ailang-openrouter-api-key` binding in `ailang-multivac` prod secrets

### Architecture

```
                ┌──────────────────────────────────────────────────────────┐
                │  ailang messages send eval-rig "task" --requires X       │
                │  (M2: new flag threads into Message.Requires field)      │
                └────────────────────────┬─────────────────────────────────┘
                                         │
                       ┌─────────────────┴─────────────────┐
                       │                                   │
                       ▼ local SQLite path                 ▼ HTTP path
              (works today)                       POST /api/messages
                                                  {"requires":[...], ...}
                                                          │
                       ┌──────────────────────────────────┴────────┐
                       │                                            │
                       ▼ local daemon                                ▼ cloud coordinator
            (M1: PORT env now set in plist                  (already exposed —
            so /api/messages is reachable)                  Cloud Run mandates PORT)
                       │                                            │
                       │                                            │ Pub/Sub publish
                       │ local SubscriptionInbox                    ▼
                       │ matches tags directly                Studio bare-metal claims
                       ▼                                      (if alive + tags match)
            Local executor dispatch                                  OR
                                                            (M3: NEW cloud fallback)
                                                            ailang-agent-executor-motoko
                                                            Cloud Run Job spawned
                                                            with motoko + AILANG v0.22+
```

### M1 — Local HTTP listener via launchd `PORT` env (~25 LOC across plist + installer + docs)

**Decision**: pick a stable default port. Options: `8765` (common dev port, no collisions on macOS), `9001` (alternative). **Recommendation: `8765`** — can be overridden by `make coord-install PORT=...`.

Files to modify:
- `tools/launchd/dev.ailang.coordinator.plist.template`: add `<key>EnvironmentVariables</key><dict>...PORT=8765...</dict>` block
- `tools/launchd/install_coordinator.sh`: accept `--port <N>` flag (default 8765), substitute into template, print final port to stdout, document `--port` in the `--help` text
- `make/coord.mk`: thread `PORT=` through to the installer
- `docs/docs/guides/coordinator-workers.md`: add a "HTTP endpoint" paragraph documenting the default port + how to override + sample `curl POST /api/messages`
- `cmd/ailang/coordinator_status.go`: extend the existing `ailang coordinator status` output to include the discovered HTTP listener (probe localhost:PORT/health, report ✓ or ✗ with the port number — uses the existing PID-from-plist convention)

Testing:
- Restart the daemon via `make coord-install` → `lsof -p <pid> -nP | grep LISTEN` shows the port within 60s of plist load
- `curl -s -H "X-API-Key: $AILANG_COORD_API_KEY" http://127.0.0.1:8765/api/messages` returns 200 with current message list

### M2 — `--requires` CLI flag in `messages send` (~30 LOC including tests)

Files to modify:
- `cmd/ailang/messages_send.go`: add `flag.Var(&requires, "requires", "comma-separated worker tags this message requires (e.g., agent:motoko,ollama:gemma4)")` where `requires` is a `stringSlice` flag variant; thread it through to `messagestore.NewMessage()` (which already accepts a `Requires []string` field per the M-COORD-MULTI-HOST-WORKERS work)
- `internal/messagestore/`: confirm `NewMessage` / `InsertMessage` carry `Requires []string` (likely already there from v0.22.0 work; if not, a 1-line schema addition)
- `cmd/ailang/messages_send_test.go`: add a `TestSendWithRequires` that flags `--requires agent:motoko,ollama:gemma4-26b-ailang` and asserts the resulting stored row has the expected slice
- `docs/docs/guides/agent-messaging.md`: add a "Tag-routed sends" subsection with the example invocation

Local path (SQLite) and HTTP path (POST /api/messages) both consume the same `Requires` field internally, so this flag works identically whether the daemon is reachable on HTTP or not.

### M3 — Cloud motoko executor Job (~150 LOC across cloudbuild + Job def + secrets binding)

Sub-tasks:

**M3.1 — `build-agent-motoko` step in `cloudbuild-dev.yaml`** (~20 LOC):

Mirrors the existing `build-agent-go` pattern using docker buildx + registry cache. No dependencies on agent-base — `Dockerfile.agent-motoko` already FROMs its own base (golang:1.25 + bun).

**M3.2 — Cloud Run Job definition** (~80 LOC, Terraform):

Mirror the existing `ailang-agent-executor-claude` Job definition. Key bindings:
- Image: `${_REGION}-docker.pkg.dev/${PROJECT}/ailang/agent-motoko:latest`
- Service account: `${prefix}-agent-executor` (existing — already has Firestore + Pub/Sub permissions)
- Secret bindings:
  - `OPENROUTER_API_KEY` ← `ailang-openrouter-api-key` (NEEDS to be created — currently only `ailang-dev-openrouter-api-key` exists; the prod secret needs to be added via Terraform alongside this Job)
  - Explicitly **NO** `ANTHROPIC_API_KEY` binding (per EXECUTOR_SHAPE.md §8 cost-control rule that already governs the gemini/codex jobs)
- Memory: 2Gi (matches the existing claude/codex jobs)
- Timeout: 3600s (1 hour wall-clock)
- Max retries: 1 (motoko is non-idempotent in cost; one retry is acceptable, more is not)

**M3.3 — `ailang-openrouter-api-key` Terraform secret resource** (~20 LOC):

Mirror the existing `ailang-anthropic-api-key` Terraform resource. The actual value comes from a 1Password / Vault entry; Terraform creates the empty Secret Manager resource and grants the agent-executor SA access; the value is uploaded once by the operator via `gcloud secrets versions add` (matches how the dev one was created today).

**M3.4 — Coordinator agent registration in `config.cloud.yaml`** (~10 LOC):

Add a `motoko` agent definition mirroring the existing `claude`/`codex` cloud agents, with `agent_cli: motoko` and `worker_tags: [agent:motoko]` so the cloud coordinator's tag matcher recognises it as a valid dispatch target.

### How the three pieces compose end-to-end

After M1+M2+M3:

```bash
# Direct local: send from Studio to Studio (tag match on bare-metal)
ailang messages send eval-rig "Add unit tests for parse.ail" \
  --requires agent:motoko \
  --from user

# Cloud-routed: send from a laptop to the cloud coordinator
AILANG_COORDINATOR_URL=https://ailang-coordinator-xxx.run.app \
AILANG_COORD_API_KEY=$(gcloud secrets versions access latest --secret=ailang-coordinator-api-key) \
ailang messages send eval-rig "Add unit tests for parse.ail" \
  --requires agent:motoko

# Either way, the message lands on the right worker.
# Verify the path:
ailang chains view <chain-id> --spans
#   → shows which worker claimed it, duration, tokens, cost
```

## Conflict Surface

Not applicable — this design doesn't touch `internal/parser/`, `internal/types/`, `internal/codegen/`, `internal/elaborate/`, or any other code path that participates in language semantics. All three milestones are operational/infrastructure changes:
- M1: launchd plist + Go code that reads `os.Getenv("PORT")` (already existing)
- M2: flag parsing in a CLI subcommand + threading into an existing struct field
- M3: Docker build + Cloud Run Job + Terraform secret + YAML config

The closest thing to a conflict surface is the **default port choice** for M1 — port 8765 might collide with another developer's tool. Mitigation: the `--port` override flag is the supported escape hatch; document the override; if collision is reported, change the default. Low risk, easy to redo.

## Files Modified

| File | Change | LOC est. |
|---|---|---|
| `tools/launchd/dev.ailang.coordinator.plist.template` | Add `EnvironmentVariables` dict with PORT | +8 |
| `tools/launchd/install_coordinator.sh` | Accept `--port`, substitute, document | +12 |
| `make/coord.mk` | Thread `PORT=` through | +3 |
| `cmd/ailang/coordinator_status.go` | Probe HTTP listener + report bound port | +20 |
| `cmd/ailang/messages_send.go` | Add `--requires` flag | +15 |
| `cmd/ailang/messages_send_test.go` | Test for `--requires` round-trip | +25 |
| `internal/messagestore/store.go` (if needed) | Confirm `Requires []string` on InsertMessage | 0-5 |
| `docs/docs/guides/coordinator-workers.md` | "HTTP endpoint" + tag-routed examples | +50 |
| `docs/docs/guides/agent-messaging.md` | "Tag-routed sends" subsection | +30 |
| `cloudbuild-dev.yaml` | `build-agent-motoko` step | +20 |
| `tools/terraform/cloud-run-jobs.tf` (or new) | `ailang-agent-executor-motoko` Job | +80 |
| `tools/terraform/secrets.tf` | `ailang-openrouter-api-key` Secret resource | +20 |
| `config.cloud.yaml` | `motoko` agent registration | +10 |
| `changelogs/v0.10-current.md` | M-COORD-TAG-ROUTING-LASTMILE entry | +40 |
| **Total** | | **~335 LOC** |

## Success Criteria

- [ ] M1: `lsof -p $(cat ~/.ailang/coordinator.pid) -nP | grep "TCP.*LISTEN"` shows a port within 60s of `make coord-install`
- [ ] M1: `curl -s -H "X-API-Key: $KEY" http://127.0.0.1:8765/health` returns 200 with `{"status":"ok"}`
- [ ] M1: `ailang coordinator status` output includes a `HTTP: ✓ http://127.0.0.1:8765` line
- [ ] M2: `ailang messages send eval-rig "test" --requires agent:motoko` succeeds and `ailang messages read <id> --raw | jq .requires` returns `["agent:motoko"]`
- [ ] M2: `ailang messages send eval-rig "test" --requires "agent:motoko,ollama:gemma4-26b-ailang"` accepts comma-separated tags and stores them as a 2-element slice
- [ ] M2: `cmd/ailang/messages_send_test.go::TestSendWithRequires` passes
- [ ] M3.1: `gcloud artifacts docker images list ${REGION}-docker.pkg.dev/$PROJECT/ailang --filter "package:agent-motoko"` returns a `:latest` tag built today
- [ ] M3.2: `gcloud run jobs describe ailang-agent-executor-motoko --region $REGION` returns a healthy spec
- [ ] M3.3: `gcloud secrets describe ailang-openrouter-api-key --project ailang-multivac` returns metadata (value uploaded out-of-band)
- [ ] M3.4: `ailang coordinator workers list --type cloud-run` includes `motoko` in the cloud agent catalog
- [ ] **End-to-end (P0 acceptance):**
  - [ ] Studio-to-Studio: `ailang messages send eval-rig "fizzbuzz_smoke" --requires agent:motoko` lands on the Studio (visible in `ailang chains view`), motoko binary invoked, completes successfully
  - [ ] Laptop-to-cloud-to-Studio: same message sent against the cloud coordinator URL with the API key is routed to the Studio via Pub/Sub, completes
  - [ ] Cloud-fallback: with the Studio stopped (`launchctl bootout gui/$(id -u)/dev.ailang.coordinator`), the same message is claimed by the cloud Cloud Run motoko Job and completes

## Out of Scope

- **Capability negotiation** (workers reporting memory/GPU/load metrics so dispatch picks the *best* matching worker, not just *any*) — item #3 in M-COORD-MULTI-HOST-WORKERS's Future Work, separate sprint
- **A Collaboration Hub UI "Workers" panel** — item #7 in Future Work, frontend work belongs in a separate UI-focused sprint
- **Worker pool fairness / round-robin** when N workers all match — item #5, separate sprint
- **Auto-discovery via Tailscale tags** — item #2, requires Tailscale ACL integration, separate sprint
- **The `ailang_bootstrap` plugin sync** — pending TODO from the v0.22.0 release that requires a machine with both repos cloned; orthogonal to this sprint

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Port 8765 collides with another tool on a developer's machine | Low-Med | Low — instant detection (daemon fails to bind) | `--port` override flag works around it; document in coordinator-workers.md; if multiple reports, change default |
| Terraform for `ailang-openrouter-api-key` requires manual `gcloud secrets versions add` after `terraform apply` (chicken-and-egg with the value) | High — every fresh GCP project | Low — one-time per environment | Document the secret-upload step in the sprint's runbook; identical pattern to how `ailang-anthropic-api-key` was bootstrapped |
| Cloud `ailang-agent-executor-motoko` Job consumes more memory than the 2Gi budget | Med | Med — Job OOMs mid-task, wastes spend | Mirror the gemini/codex 2Gi default; if OOMs observed in practice, bump to 4Gi; the per-Job cost cap ($0.30 from M-EVAL-SWEET-SPOT-FOLLOWUP-P1) bounds blast radius |
| The cloud coordinator hasn't auto-deployed to v0.22.0 (assumed but unverified) | Med | High — `requires` field isn't recognised, dispatch fails | Pre-flight check in M3.4: `gcloud run services describe ailang-coordinator --format='value(spec.template.spec.containers[0].image)'` — if image SHA is pre-v0.22.0, trigger a redeploy manually before M3.4 |
| `internal/messagestore.NewMessage` doesn't yet have a `Requires` field (assumed shipped in v0.22.0 but unverified) | Low | Low — extra ~5 LOC | Quick `grep -n "Requires" internal/messagestore/*.go` at the start of M2 will confirm; if missing, add the field + a schema migration line |

## Related Documents

- **Predecessor**: [m-coord-multi-host-workers.md](../../implemented/v0_22_0/m-coord-multi-host-workers.md) — provided the routing machinery; named this sprint as item #1 in its Future Work
- **Dependency**: [m-motoko-executor-adapter.md](../../implemented/v0_18_0/m-motoko-executor-adapter.md) — provided the motoko executor + JSONL schema-v1 contract; this sprint adds the cloud deployment story
- **Sibling**: [m-motoko-v021-effect-row-migration.md](../../implemented/v0_22_0/m-motoko-v021-effect-row-migration.md) — got motoko_agent's build working again on v0.21+; this sprint puts it into production rotation
- **Cost-control rule reference**: [EXECUTOR_SHAPE.md §8](../../../docs/internal/EXECUTOR_SHAPE.md) — the rule that motoko Cloud Run Jobs must NOT bind ANTHROPIC_API_KEY
- **Pinned motoko revision**: [internal/executor/motoko/README.md](../../../internal/executor/motoko/README.md) — currently pins commit `258a039` on `feature/v021-effect-row-migration`; this sprint will use whatever's on motoko_agent main after PR #33 merges

## Timeline

Single-session sprint estimated under 1 day if M3.3's Terraform-then-manual-secret-upload step doesn't block. Decoupled milestones can run in parallel after a brief sequencing decision:

```
M2 (CLI flag — fully local, no infra) ──┐
                                        ├──► M1+M2 verification (local end-to-end)
M1 (launchd PORT env — fully local) ────┘
                                        │
                                        ▼
                              M3.1 (cloudbuild)
                                        │
                                        ▼
                              M3.2 (Job def)
                                        │
                                        ▼
                              M3.3 (secret) — manual gate
                                        │
                                        ▼
                              M3.4 (config.cloud.yaml + cloud verification)
```

## Open Questions

- [ ] Should we make `PORT` a config.yaml setting too (in addition to env var) so users can override without re-installing the plist? Probably yes — cheap to add, matches the `worker_tags` pattern of "configurable in yaml, overridable by env"
- [ ] Should cloud-mode dispatch prefer bare-metal over Cloud Run when both are eligible, or just dispatch to whichever responds first? **Recommendation: prefer bare-metal** (Studio has motoko's prompt cache warm, lower latency, lower cost than spinning a Cloud Run Job). Implementable as a Pub/Sub subscription ordering hint or a "claim priority" field on heartbeats. Could be deferred to the capability-negotiation sprint.

---

**Document created**: 2026-05-27
**Last updated**: 2026-05-27
