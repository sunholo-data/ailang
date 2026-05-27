# Sprint Plan: M-COORD-TAG-ROUTING-LASTMILE

**Design doc**: [m-coord-tag-routing-lastmile.md](m-coord-tag-routing-lastmile.md)
**Target**: v0.23.0
**Status**: Approved by user (2026-05-27)
**Estimated**: 1-2 days (~335 LOC)
**Risk level**: Medium (cloud-deploy verification gate is the only meaningful unknown — all local work is mechanical)

## Sprint Goal

Make `ailang messages send eval-rig "<task>" --requires agent:motoko` work end-to-end from any host. Cover three routing scenarios:
1. Studio-to-Studio (local tag match)
2. Laptop → cloud coordinator → Studio (Pub/Sub routing)
3. Laptop → cloud coordinator → cloud motoko Job (cloud-fallback when Studio offline)

After this sprint, M-COORD-MULTI-HOST-WORKERS is genuinely usable, not just architecturally complete.

## Scope decisions

| Decision | Choice | Rationale |
|---|---|---|
| Default HTTP port for local daemon | `8765` | Common dev-port range, no known collisions on macOS, override via `--port` |
| M2 flag syntax | `--requires agent:motoko,ollama:gemma4` (comma-separated) | Matches the existing single-flag-multi-value pattern in `worker_tags` config; familiar from gcloud/curl |
| M3 secret store | `ailang-openrouter-api-key` in `ailang-multivac` (prod) | Mirrors `ailang-anthropic-api-key`; the existing `-dev` secret was created today for verification — prod gets its own version |
| M3 Job memory | 2 GiB | Matches existing claude/codex Jobs; bump to 4 GiB only if OOM observed |
| M3 retry count | 1 (max) | motoko is non-idempotent in cost; one retry is acceptable, more is not |
| Cloud-fallback priority | Prefer bare-metal Studio over Cloud Run | Studio has prompt cache warm + lower latency + cheaper; implementable as Pub/Sub subscription ordering |
| `PORT` in config.yaml? | YES — additive — matches `worker_tags` pattern | Cheap; env var still wins over config; documented |

## Recent velocity

- 2026-05-26: v0.22.0 released — 102 commits, 18 CHANGELOG entries, motoko reconciliation + multi-host workers + iface fix all in one day
- 2026-05-25: M-COORD-MULTI-HOST-WORKERS — ~1,200 LOC across 5 milestones in ~6 hours
- 2026-05-26: M-MOTOKO-AILANG-RECONCILE (~45 min actual vs 2.75h est) + M-MOTOKO-V021-EFFECT-ROW-MIGRATION (~3h actual)
- **This sprint estimate**: ~335 LOC over 1-2 days. Decoupled milestones mean partial progress is shippable. Realistic.

## Milestones

### M1 — Local HTTP listener via launchd `PORT` env (~25 LOC, ~1 hour)

**What:**
- [ ] Edit `tools/launchd/dev.ailang.coordinator.plist.template`: add `EnvironmentVariables` dict with `PORT=8765`
- [ ] Edit `tools/launchd/install_coordinator.sh`: accept `--port <N>` flag (default 8765), substitute into template, document in `--help`
- [ ] Update `make/coord.mk`: thread `PORT=` variable through to the installer (`coord-install PORT=...`)
- [ ] Extend `cmd/ailang/coordinator_status.go`: probe `localhost:PORT/health`, add `HTTP: ✓ http://127.0.0.1:8765` line to status output
- [ ] Update `~/.ailang/config.yaml` schema in `internal/agentcfg/`: optional `coordinator.http_port` field; env wins over config
- [ ] Test: `make coord-install --dry-run` shows the planned plist with PORT block

**Acceptance:**
- [ ] `lsof -p $(cat ~/.ailang/coordinator.pid) -nP | grep "TCP.*LISTEN"` shows port 8765 within 60s of `make coord-install`
- [ ] `curl -s -H "X-API-Key: $AILANG_COORD_API_KEY" http://127.0.0.1:8765/health` returns 200 with `{"status":"ok"}`
- [ ] `ailang coordinator status` output includes the HTTP listener line
- [ ] `--port 9999` override works and the daemon binds to the chosen port

**Files:**
- MODIFY: `tools/launchd/dev.ailang.coordinator.plist.template` (+8 LOC)
- MODIFY: `tools/launchd/install_coordinator.sh` (+12 LOC)
- MODIFY: `make/coord.mk` (+3 LOC)
- MODIFY: `cmd/ailang/coordinator_status.go` (+20 LOC)
- MODIFY: `internal/agentcfg/coordinator_config.go` (optional ~5 LOC for config.yaml field)

### M2 — `--requires` CLI flag in `messages send` (~30 LOC + tests, ~1 hour)

**What:**
- [ ] Read `cmd/ailang/messages_send.go` to find the existing flag-set + `messagestore.NewMessage()` call site
- [ ] `grep -n "Requires" internal/messagestore/*.go cmd/ailang/messages_send.go` — confirm whether `Requires []string` field already exists (M-COORD-MULTI-HOST-WORKERS shipped the HTTP-side; CLI-side likely doesn't have it yet)
- [ ] Add `flag.Var(&requires, "requires", "comma-separated worker tags this message requires (e.g., agent:motoko,ollama:gemma4)")` with a `stringSlice` Value implementation
- [ ] Thread `requires []string` through to message construction
- [ ] If `messagestore.NewMessage()` doesn't have the field, add it (1-2 LOC) — fail loud if SQLite schema needs migration
- [ ] Write `TestSendWithRequires` in `messages_send_test.go`: invoke with `--requires agent:motoko,ollama:gemma4-26b-ailang`, assert the stored row has the 2-element slice
- [ ] Add a "Tag-routed sends" subsection to `docs/docs/guides/agent-messaging.md` with example invocation

**Acceptance:**
- [ ] `ailang messages send eval-rig "test" --requires agent:motoko` succeeds (exit 0)
- [ ] `ailang messages read <id> --raw | jq .requires` returns `["agent:motoko"]`
- [ ] Comma-separated form: `--requires "agent:motoko,ollama:gemma4-26b-ailang"` → 2-element slice
- [ ] `cmd/ailang/messages_send_test.go::TestSendWithRequires` passes
- [ ] `go test ./cmd/ailang/... -run TestSendWithRequires -count=20` is deterministic (no map-iteration nondeterminism)

**Files:**
- MODIFY: `cmd/ailang/messages_send.go` (+15 LOC)
- MODIFY: `cmd/ailang/messages_send_test.go` (+25 LOC)
- POSSIBLY: `internal/messagestore/store.go` (0-5 LOC — confirm or add `Requires []string`)
- MODIFY: `docs/docs/guides/agent-messaging.md` (+30 LOC)

### M3 — Cloud motoko executor Job (~150 LOC, ~3-4 hours including verification)

**Sub-tasks ship as ONE commit** (they're tightly coupled — none works without the others):

**M3.1 — Add `build-agent-motoko` step in `cloudbuild-dev.yaml`** (~20 LOC)

- [ ] Mirror the existing `build-agent-go` pattern (docker buildx + registry cache)
- [ ] `waitFor: ['setup-buildx']` only — no agent-base dependency (motoko's Dockerfile self-bases on golang:1.25 + bun)
- [ ] Push to `${_REGION}-docker.pkg.dev/$_TARGET_PROJECT/ailang/agent-motoko:latest`

**M3.2 — `ailang-agent-executor-motoko` Cloud Run Job (Terraform)** (~80 LOC)

- [ ] Mirror `ailang-agent-executor-claude` Job definition
- [ ] Image: `agent-motoko:latest` from M3.1
- [ ] Service account: `${prefix}-agent-executor` (existing)
- [ ] Memory: 2Gi, CPU: 2, timeout: 3600s, max retries: 1
- [ ] Secret bindings:
  - `OPENROUTER_API_KEY` ← `ailang-openrouter-api-key` (M3.3)
  - Explicitly NO `ANTHROPIC_API_KEY` (cost-control rule)
- [ ] Env: `MOTOKO_CONFIG=dogfood`, `AILANG_OBSERVATORY_URL`, etc. — mirror what existing executor Jobs set

**M3.3 — `ailang-openrouter-api-key` Terraform secret resource** (~20 LOC)

- [ ] Mirror `ailang-anthropic-api-key` resource block
- [ ] Grant `${prefix}-agent-executor` SA access (`roles/secretmanager.secretAccessor`)
- [ ] Document the manual `gcloud secrets versions add ailang-openrouter-api-key --data-file=...` step in the sprint's runbook — chicken-and-egg with the secret value is inherent to Terraform + Secret Manager and matches how `ailang-anthropic-api-key` was bootstrapped

**M3.4 — Coordinator agent registration in `config.cloud.yaml`** (~10 LOC)

- [ ] Add `motoko` agent under `coordinator.agents:`, mirror `claude` entry's structure
- [ ] `agent_cli: motoko`
- [ ] `worker_tags: [agent:motoko]`
- [ ] `executor_job: ailang-agent-executor-motoko` (matches the Cloud Run Job name from M3.2)

**Pre-flight check (BEFORE any code):**
- [ ] `gcloud run services describe ailang-coordinator --project=ailang-multivac --region=europe-west1 --format='value(spec.template.spec.containers[0].image)'`
- [ ] Confirm image SHA corresponds to a v0.22.0+ build (the `requires` field needs to be recognised). If pre-v0.22.0, manually trigger redeploy via cloudbuild before M3.4

**Acceptance:**
- [ ] `gcloud artifacts docker images list ${REGION}-docker.pkg.dev/$PROJECT/ailang --filter "package:agent-motoko"` returns a `:latest` tag built today
- [ ] `gcloud run jobs describe ailang-agent-executor-motoko --region $REGION --project=ailang-multivac` returns a healthy spec, ready to execute
- [ ] `gcloud secrets describe ailang-openrouter-api-key --project=ailang-multivac` returns metadata (value uploaded manually post-apply)
- [ ] `ailang coordinator workers list --type cloud-run --json | jq '.[] | select(.tags[]? == "agent:motoko") | .name'` returns `ailang-agent-executor-motoko`

**Files:**
- MODIFY: `cloudbuild-dev.yaml` (+20 LOC for build step)
- NEW or MODIFY: `tools/terraform/cloud-run-jobs.tf` (+80 LOC for Job def)
- MODIFY: `tools/terraform/secrets.tf` (+20 LOC for secret resource)
- MODIFY: `config.cloud.yaml` (+10 LOC for motoko agent entry)

### M4 — Docs + End-to-end verification matrix (~120 LOC docs + manual verification)

**What:**
- [ ] Update `docs/docs/guides/coordinator-workers.md`: new "HTTP endpoint" section (~50 LOC) covering default port, override, curl examples, the `requires` JSON body, GET `/health`
- [ ] Update `docs/docs/guides/agent-messaging.md`: "Tag-routed sends" subsection already in M2 — extend with cloud-coordinator URL example
- [ ] Add CHANGELOG entry to `changelogs/v0.10-current.md`: cover all 4 milestones + the 3 routing scenarios verified
- [ ] **Manual verification matrix** (results recorded in the sprint plan as it completes):

| # | Scenario | Command | Expected | Verified? |
|---|---|---|---|---|
| 1 | Studio→Studio (local tag match) | `ailang messages send eval-rig "fizzbuzz_smoke" --requires agent:motoko` from Studio shell | Studio claims, motoko runs, `ailang chains view <id>` shows compile=✓ runtime=✓ stdout=✓ | [ ] |
| 2 | Laptop→cloud→Studio (Pub/Sub routing) | Same command from laptop with `AILANG_COORDINATOR_URL` + API key | Message published to Pub/Sub, Studio's subscription claims, runs | [ ] |
| 3 | Cloud-fallback (Studio offline) | `launchctl bootout gui/$(id -u)/dev.ailang.coordinator` first, then same command | No Studio claim, cloud Job `ailang-agent-executor-motoko` spawns, runs, completes | [ ] |

**Acceptance:**
- [ ] All 3 manual scenarios PASS (or are explicitly deferred with a documented reason)
- [ ] CHANGELOG entry covers what shipped + the 3 verified scenarios
- [ ] Docs reference the live PR URLs / commit SHAs

**Files:**
- MODIFY: `docs/docs/guides/coordinator-workers.md` (+50 LOC)
- MODIFY: `docs/docs/guides/agent-messaging.md` (small extension to the M2 subsection)
- MODIFY: `changelogs/v0.10-current.md` (+40 LOC sprint summary)

## Sequencing

```
Day 1 morning (~3h):
  M1 (launchd PORT)     ──┐
                          ├──► M1+M2 verification (local end-to-end smoke)
  M2 (--requires flag) ──┘    Pre-flight: M2 grep for messagestore.Requires field

Day 1 afternoon (~3h):
  M3 pre-flight (gcloud check on ailang-coordinator image version)
                          │
  M3.1 cloudbuild step ──┐
  M3.2 Job def (TF)   ──┤── all four sub-tasks ship together
  M3.3 Secret (TF)    ──┤   (Terraform apply gates the publish)
  M3.4 config.cloud   ──┘

Day 2 morning (~2h):
  M3 deploy + cloud Job verification
  M4 verification matrix scenario 1 (Studio→Studio)
  M4 verification matrix scenario 2 (Laptop→cloud→Studio)

Day 2 afternoon (~1h):
  M4 verification matrix scenario 3 (cloud-fallback)
  M4 docs + CHANGELOG + commit + push
```

M1 and M2 can run in parallel (different files, fully local). M3.* must ship together (mutually dependent — the cloudbuild step is moot without the Job, the Job is moot without the secret, etc.).

## Total estimates

| Milestone | LOC est | Hours | Risk |
|---|---:|---:|---|
| M1 | 25 | 1 | Low — pure plist + Go-flag work |
| M2 | 30 | 1 | Low — small CLI change, deterministic test |
| M3.1-3.4 | 130 | 3-4 | Medium — Terraform apply + manual secret upload + pre-flight gcloud check |
| M4 | 120 (docs) + manual | 2 | Low — gated on M1-M3 |
| **Total** | **~305 LOC + docs** | **~7-8h** | **Medium overall** |

(Slightly under the design doc's 335 LOC estimate — I think `internal/messagestore` changes are 0 LOC, not 5, because v0.22.0's M-COORD-MULTI-HOST-WORKERS shipped the field already.)

## Risk management

| Risk | Likelihood | Mitigation |
|---|---|---|
| Cloud `ailang-coordinator` not yet on v0.22.0 (auto-deploy assumption) | Med | Pre-flight gcloud check at start of M3 — manual cloudbuild trigger if needed before continuing |
| `internal/messagestore.NewMessage` missing `Requires` field | Low | M2 starts with a grep; if missing, ~5 LOC addition + 1 schema migration line |
| Terraform `ailang-openrouter-api-key` apply blocks on missing secret value | High (always) | Document the manual `gcloud secrets versions add` step in the sprint runbook; identical pattern to `ailang-anthropic-api-key`'s bootstrap — well-trodden path |
| Cloud Run Job OOM at 2Gi for motoko + agent loop | Med | Mirror gemini/codex 2Gi default; bump to 4Gi if observed; the per-Job $0.30 cost cap bounds blast radius |
| Port 8765 collides on developer's machine | Low-Med | `--port` override flag; if collision is reported, change default in a quick patch |
| eval-smoke verification (scenario 2/3) needs OPENROUTER_API_KEY routing through to cloud Job | Med | M3.3 explicitly binds the prod secret; verify with a no-op task before declaring scenario 2 PASS |

## Success criteria for the whole sprint

- [ ] All four milestones (M1+M2+M3+M4) have all acceptance criteria checked
- [ ] All 3 routing scenarios in the M4 verification matrix PASS (or are explicitly deferred with reasoning)
- [ ] CHANGELOG entry committed
- [ ] Design doc + sprint plan moved from `planned/v0_23_0/` to `implemented/v0_23_0/`
- [ ] No new issues filed during execution that aren't either fixed or deferred-and-documented

## Out of scope (per design doc)

- Capability negotiation (memory/GPU/load metrics for dispatch)
- Collaboration Hub UI Workers panel
- Worker pool fairness / round-robin
- Tailscale ACL auto-discovery
- `ailang_bootstrap` plugin sync (orthogonal v0.22.0 followup)

## Day-by-day plan

**Day 1 (target: ~6 hours)**

| Slot | Work |
|---|---|
| 09:00-09:30 | Pre-flight: `gcloud run services describe ailang-coordinator` → confirm v0.22.0+ image; grep `internal/messagestore/` for `Requires` field |
| 09:30-10:30 | M1 — launchd plist + installer + status command (parallel with M2 if working with a co-pilot, otherwise serial) |
| 10:30-11:30 | M2 — `--requires` flag + test |
| 11:30-12:00 | M1+M2 local smoke test: `make coord-install` → `ailang messages send eval-rig "x" --requires agent:motoko` → `ailang messages read <id> --raw` |
| 13:00-15:00 | M3.1 + M3.2 — cloudbuild step + Terraform Job definition |
| 15:00-16:00 | M3.3 + M3.4 — secret resource + config.cloud.yaml; `terraform plan` review |
| 16:00-17:00 | `terraform apply` + manual secret upload + initial cloud Job smoke (no real task yet, just verify Job exists) |

**Day 2 (target: ~2-3 hours)**

| Slot | Work |
|---|---|
| 09:00-10:00 | M4 verification scenario 1 (Studio→Studio) |
| 10:00-11:00 | M4 verification scenario 2 (Laptop→cloud→Studio) — requires another terminal posing as the laptop, with `AILANG_COORDINATOR_URL` set to the cloud service |
| 11:00-12:00 | M4 verification scenario 3 (cloud-fallback) — `launchctl bootout` the Studio, send the message, verify Cloud Job claims |
| 12:00-13:00 | Docs + CHANGELOG + commit + push + finalize sprint (move design doc + plan to `implemented/v0_23_0/`) |

---

**Sprint plan created**: 2026-05-27
**Created by**: sprint-planner skill (claude-opus-4-7)
**Approved by**: user
