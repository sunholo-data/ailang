# Sprint Plan: M-PKG-AUTONOMOUS-UPDATES

## Summary
Connect AILANG's messaging, registry, and cloud execution systems so package updates flow autonomously: message → coordinator → agent → publish → provenance. Implements graduated autonomy (Class A/B/C) with version history for registry website discovery.

**Duration:** 5 days (11 hours active development)
**Dependencies:** None — all prerequisite systems are implemented (M-PKG-REGISTRY v0.9.7, M-PKG-MSG v0.8.0, Cloud execution v0.9.0+)
**Risk Level:** Medium — touches coordinator dispatch path and registry storage layout
**Design Doc:** [m-pkg-autonomous-updates.md](m-pkg-autonomous-updates.md)

## Current Status Analysis

### Completed Recently
- M-PKG-REGISTRY: Full registry with validator on Cloud Run (~800 LOC)
- M-PKG-ECOSYSTEM: 14 packages published, transitive resolution working (~200 LOC)
- Resolver: Transitive registry deps fix (~160 LOC)

### Velocity
- Recent 14-day average: ~144 LOC/day (2018 insertions across 14 days)
- Estimated capacity: ~720 LOC for this 5-day sprint
- Target: ~670 LOC (implementation + tests), well within capacity

### Remaining from Design Doc
- Phase 1: Subdirectory field + single package agent MVP (~100 LOC)
- Phase 2: Cloud config + multi-package (config only, minimal code)
- Phase 3: Dependent notification + dynamic autonomy (~160 LOC)
- Phase 4: Provenance + history + AGENT.md + cascade (~360 LOC)

## Proposed Milestones

### Milestone 1: M1_SUBDIRECTORY_FIELD
**Goal:** Add `Subdirectory` field to `AgentConfig` and wire it through worktree setup so agents can target monorepo subdirectories.

**Estimated:** 60 LOC implementation + 40 LOC tests = 100 LOC
**Duration:** Day 1 (2 hours)

**Tasks:**
- Add `Subdirectory string` field to `AgentConfig` in `internal/coordinator/agent_registry.go`
- Wire subdirectory through `WorktreeManager` — agent's working dir becomes `worktree_path/subdirectory`
- Update `executeTask()` to pass subdirectory to executor working dir
- Update invoke template variable substitution to include `{{.Subdirectory}}`
- Verify `pkg:sunholo/auth` format works as inbox name in `InboxMessageAdapter` (colon in SQLite)
- Write unit test: `AgentConfig` with `Subdirectory` → correct worktree path

**Acceptance Criteria:**
- [ ] `AgentConfig.Subdirectory` field exists with YAML/JSON tags
- [ ] Worktree agent execution uses `subdirectory` as working dir when set
- [ ] `pkg:sunholo/auth` inbox format works in message adapter (no colon issues)
- [ ] Unit tests pass for subdirectory path resolution
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Colon in inbox name may break SQLite queries — Mitigation: test with actual `ailang messages send pkg:sunholo/auth "test"`

### Milestone 2: M2_AUTONOMY_ROUTER
**Goal:** Implement dynamic autonomy routing that adjusts agent approval settings based on incoming package message change class.

**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** Day 2 (2.5 hours)

**Tasks:**
- Create `internal/coordinator/autonomy_router.go` with `AdjustAutonomyForChangeClass()`
- Implement `classifyChange()` mapping all 11 `PackageMessageKind` values to Class A/B/C
- Wire hook into `executeTask()` in `daemon_tasks_exec.go` — call before task dispatch
- Write tests covering: all 11 message kinds → correct class, breaking flag override, effect widening → always C

**Acceptance Criteria:**
- [ ] `AdjustAutonomyForChangeClass()` returns modified config for Class A (skip_approval, auto_merge)
- [ ] Class B sets `auto_approve_handoffs: true`, Class C sets all false
- [ ] `classifyChange()` handles all 11 `PackageMessageKind` constants
- [ ] Effect widening always maps to Class C regardless of breaking flag
- [ ] Non-package messages pass through unchanged
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- None significant — straightforward mapping logic

### Milestone 3: M3_DEPENDENT_NOTIFICATION
**Goal:** When a package publishes, automatically notify all packages that depend on it by sending `upgrade-available` messages to their `pkg:*` inboxes.

**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** Day 3 (2.5 hours)

**Tasks:**
- Implement `emitDependentNotifications()` in `cmd/ailang/pkg_publish.go`
- Add `dependsOn()` helper to check if a package manifest lists a given dependency
- Call from publish success path after existing `emitPublishMessages()`
- Determine change class from interface hash comparison (changed = minor, same = patch)
- Write test: mock registry index with 3 packages, publish A → B and C get messages, D (no dep) doesn't

**Acceptance Criteria:**
- [ ] After successful `ailang publish`, dependent packages receive `upgrade-available` messages
- [ ] Only packages that actually depend on the published package get notified
- [ ] Change class set to "minor" when interface hash changed, "patch" when unchanged
- [ ] Self-notification skipped (package doesn't notify itself)
- [ ] Best-effort: registry fetch failure doesn't block publish
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Registry index fetch latency adds time to publish — Mitigation: best-effort, don't block on failure

### Milestone 4: M4_PROVENANCE_AND_HISTORY
**Goal:** Add provenance tracking and version history to registry metadata, plus AGENT.md as a first-class artifact. Implement cascade scheduling with circuit breaker.

**Estimated:** 200 LOC implementation + 110 LOC tests = 310 LOC
**Duration:** Day 4-5 (4 hours)

**Tasks:**
- Add `ProvenanceInfo` struct to `internal/pkg/registry_types.go`
- Add `VersionHistory` and `HistoryEntry` structs to `internal/pkg/registry_types.go`
- Extend `IndexEntry` with `LastUpdated`, `UpdatedBy`, `LatestSummary` fields
- Create `internal/coordinator/history_collector.go` — event collector during task lifecycle
- Modify publish flow to embed provenance from env vars / message context
- Modify registry validator to upload `AGENT.md` as separate blob
- Modify registry validator to accept and store `history.json`
- Implement `ScheduleCascadeUpdate()` with topological sort in `cascade_scheduler.go`
- Add `CascadeCircuitBreaker` struct (3 failures = pause + alert)
- Add `ailang pkg provenance vendor/name@version` CLI command
- Add `ailang pkg history vendor/name@version` CLI command
- Write tests: ProvenanceInfo JSON roundtrip, HistoryEntry ordering, cascade topo sort, circuit breaker threshold

**Acceptance Criteria:**
- [ ] `ProvenanceInfo` JSON-serializable with backward compat (all fields optional except `change_class`)
- [ ] `VersionHistory` captures ordered message trail with timestamps and statuses
- [ ] `IndexEntry` includes `last_updated`, `updated_by`, `latest_summary`
- [ ] AGENT.md uploaded as separate blob at `packages/vendor/name/version/AGENT.md`
- [ ] `history.json` stored alongside `metadata.json` in registry
- [ ] `ScheduleCascadeUpdate()` returns packages in correct topological order
- [ ] Circuit breaker pauses cascade after 3 consecutive failures
- [ ] `ailang pkg provenance` displays provenance chain
- [ ] `ailang pkg history` displays version history timeline
- [ ] All new structs have JSON marshal/unmarshal tests
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Registry validator changes require Docker rebuild + redeploy — Mitigation: use Cloud Build trigger
- Backward compat with existing metadata.json — Mitigation: all new fields are `omitempty`

## Config Milestone (No Code): M5_CLOUD_CONFIG
**Goal:** Add 14 package agent entries to `config.cloud.yaml` + workspace mapping + invoke template.

**Note:** This is config-only work in the `ailang-multivac` repo. Can be done in parallel or after code milestones.

**Tasks:**
- Add `sunholo-data/ailang-packages` workspace mapping
- Add 14 package agent entries (following the template in design doc)
- Create `pkg-update.md` invoke template
- Upload config via `make config-upload`
- Deploy to dev: `make apply ENV=dev`

## Success Metrics
- Test coverage: All new files have >80% coverage
- `make test` passing
- `make lint` clean
- `make verify-examples` passing
- Documentation: CHANGELOG.md updated, design doc remains current

## Dependencies
- Registry validator accessible (Cloud Run, already deployed)
- `ailang-packages` repo accessible for integration testing
- `ailang-multivac` repo accessible for config updates (M5)

## Open Questions
- None — all high-impact decisions resolved in design doc

## Notes
- LOC estimates include both implementation and test code
- Config milestone (M5) is in a separate repo (`ailang-multivac`) and can be done independently
- Milestones are ordered by dependency: M1 → M2 → M3 → M4 (M5 parallel)
- Each milestone is independently testable and committable
