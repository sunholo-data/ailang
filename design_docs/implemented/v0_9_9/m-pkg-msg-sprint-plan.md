# Sprint Plan: M-PKG-MSG — Package Messaging Graph

## Summary

Add structured package coordination messaging to AILANG: typed message schemas for 11 package event kinds, `pkg:` inbox addressing, auto-emission from publish/install events, and coordinator triage hooks. Bridges the existing package system (static graph) with the existing messaging system (dynamic coordination).

**Duration:** 4 days (3 milestones)
**Dependencies:** Package system (v0.9.5, implemented), messaging system (implemented)
**Risk Level:** Low — both systems exist; this wires them together

## Current Status Analysis

### Completed Recently
- M-PKG v0.9.5: Full package system (~1,000 LOC, 47 tests) in ~2 days
- M-PKG-REGISTRY v0.9.7: Registry client (~550 LOC, 9 tests) in ~1 day
- M-COST1 v0.9.6: Firestore optimization (~400 LOC, 9 tests) in ~1 day

### Velocity
- Recent average: ~500-700 LOC/day (implementation + tests)
- Messaging system already has inbox routing, semantic dedup, envelope slots
- Package system already has manifest parsing, hashing, lockfile diffing

### What Exists (Integration Points)
- `internal/messaging/inbox.go`: `InboxMessage` struct with `ToInbox` string field, full CRUD
- `internal/messaging/schema.go`: `inbox_messages` table with index on `(to_inbox, status, created_at)`
- `cmd/ailang/messages_send.go`: CLI send with `--type`, `--json` support
- `internal/pkg/manifest.go`: `PackageManifest` with name, version, hashes
- `internal/pkg/registry.go`: Registry client with publish/install
- `internal/pkg/lockfile.go`: Lockfile with content/interface hashes

## Proposed Milestones

### Milestone 1: M1_PKG_MESSAGE_SCHEMA — Package Message Types & Validation
**Goal:** Define typed Go structs for package coordination messages, validate against schema, store/retrieve via existing inbox system.
**Estimated:** ~200 LOC implementation + ~100 LOC tests = ~300 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/messaging/pkg_schema.go`:
   - `PackageMessageEnvelope` struct matching design doc JSON schema
   - `PackageRef` struct (name, from/to version, from/to interface hash, from/to content hash, change class, effect delta, breaking)
   - `PackageMessageKind` type with 11 constants (upgrade-available, interface-change-notice, etc.)
   - `ValidatePackageMessage(envelope)` — validate required fields per kind
   - Serialize/deserialize to/from `InboxMessage.Payload` JSON
2. Create `internal/messaging/pkg_schema_test.go`:
   - Validation tests for each message kind (required fields, missing fields)
   - Round-trip serialization tests
   - Edge cases: empty version, missing hash, invalid kind

**Acceptance Criteria:**
- [ ] All 11 message kinds defined as typed constants
- [ ] PackageMessageEnvelope struct with required/optional fields
- [ ] Validation rejects messages missing required fields per kind
- [ ] JSON round-trip preserves all fields
- [ ] `make test` passes, `make lint` clean

**Risks:**
- None significant — straightforward Go struct + validation code

---

### Milestone 2: M2_PKG_INBOX_ROUTING — Package-Scoped Inbox Addressing & CLI
**Goal:** Support `pkg:vendor/name`, `workspace:name`, `team:name` inbox prefixes in routing, filtering, and CLI.
**Estimated:** ~200 LOC implementation + ~80 LOC tests = ~280 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/messaging/pkg_routing.go`:
   - `ParseInboxAddress(addr)` → type (`pkg`, `workspace`, `team`, `plain`), name
   - `ListPackageInboxes(store)` — query distinct `to_inbox` values matching `pkg:%`
   - `AffectedWorkspaces(store, pkgName)` — find workspace inboxes that reference a package
2. Update `cmd/ailang/messages_send.go`:
   - Add `--pkg-json` flag for structured package message payloads
   - Validate that `--pkg-json` payloads conform to `PackageMessageEnvelope` schema
   - Auto-set `from` field based on `pkg:` prefix when sending from package context
3. Add CLI commands in `cmd/ailang/pkg_msg.go`:
   - `ailang pkg notify-upgrade <pkg>@<version>` — emit upgrade-available from manifest diff
   - `ailang pkg affected-by <pkg>@<version>` — list workspaces depending on package
4. Update `cmd/ailang/messages_crud.go`:
   - `ailang msg inbox --channel pkg:sunholo/auth` — filter by package-scoped inbox
   - Support `pkg:*` wildcard to list all package inboxes
5. Tests for routing, parsing, and CLI integration

**Acceptance Criteria:**
- [ ] `pkg:vendor/name` addresses parsed and routed correctly
- [ ] `ailang pkg notify-upgrade sunholo/auth@0.2.0` emits structured message
- [ ] `ailang pkg affected-by sunholo/auth@0.2.0` lists dependent workspaces
- [ ] `ailang msg inbox --channel pkg:sunholo/auth` filters correctly
- [ ] `make test` passes, `make lint` clean

**Risks:**
- Package dependency graph traversal for `affected-by` may need lockfile parsing — keep to direct deps only in v1

---

### Milestone 3: M3_PKG_EVENT_EMISSION — Auto-Emit from Package Events & Coordinator Hooks
**Goal:** Wire publish/install/lock events to automatically emit structured package messages. Add coordinator triage stubs.
**Estimated:** ~250 LOC implementation + ~100 LOC tests = ~350 LOC
**Duration:** 2 days

**Tasks:**
1. Create `internal/messaging/pkg_events.go`:
   - `EmitUpgradeAvailable(store, oldManifest, newManifest)` — compare manifests, emit if interface hash changed
   - `EmitEffectWideningWarning(store, oldManifest, newManifest)` — compare effect ceilings
   - `EmitFromLockfileDiff(store, oldLock, newLock)` — detect added/upgraded/removed deps, emit per-dep messages
   - `EmitDeprecationNotice(store, pkg, exports, reason)` — manual trigger
2. Wire into `internal/pkg/registry.go`:
   - After successful `publish`: call `EmitUpgradeAvailable` with old/new manifest
   - After successful `install`: call `EmitFromLockfileDiff` with old/new lockfile
3. Create `internal/messaging/pkg_triage.go`:
   - `TriagePackageMessage(store, msg)` — classify actionability (no-action, verify-local, migrate, escalate)
   - `DeduplicatePackageReports(store, pkgName)` — cluster compatibility reports by package + version
4. Create `internal/messaging/pkg_status.go`:
   - `UpdateMessageLifecycle(store, msgID, newStatus)` — enforce valid transitions
   - `SupersedeOlderMessages(store, pkgName, version)` — mark old upgrade-available as superseded
5. Tests for event emission, triage classification, lifecycle transitions

**Acceptance Criteria:**
- [ ] `ailang publish` emits `upgrade-available` when interface hash changes
- [ ] `ailang install` emits lockfile-diff messages for upgraded deps
- [ ] Effect widening detected and warned
- [ ] Message lifecycle transitions enforced (open → ack → in-progress → completed)
- [ ] Older upgrade-available messages superseded by newer releases
- [ ] Compatibility reports deduplicated by package + version
- [ ] `make test` passes, `make lint` clean
- [ ] CHANGELOG.md updated
- [ ] Design doc moved to `design_docs/implemented/`

**Risks:**
- Registry publish flow may not have access to "old" manifest — may need to cache previous version locally or query registry
- Mitigation: For v1, compare against lockfile's recorded hashes rather than re-fetching old manifest

---

## Success Metrics
- Test coverage: maintain current levels (47+ package tests, new messaging tests)
- All 11 message kinds validated and documented
- Package events auto-emit structured messages
- CLI surface matches design doc
- `make test` and `make lint` clean throughout
- CHANGELOG.md updated with feature summary

## Dependencies
- Package system (v0.9.5) — implemented ✅
- Messaging system — implemented ✅
- Registry client (v0.9.7) — implemented ✅

## Open Questions
- Should `ailang pkg affected-by` traverse transitive deps or just direct? (Recommend: direct only for v1)
- Should auto-emission be opt-in via `ailang.toml` flag or always-on? (Design doc says opt-in Phase 1)
- Should compatibility reports include full test output or just pass/fail summary? (Recommend: summary only for v1)
