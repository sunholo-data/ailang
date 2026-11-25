# Sprint Plan: M-PKG-PARALLEL-AGENTS, Sprint 1 (Phase 1: measured classes and boundary hygiene)

## Placement and Human-Decision Rules

- **Design:** [m-pkg-parallel-agents.md](m-pkg-parallel-agents.md). The design freeze was ratified on 2026-11-25 by Mark (attended: "ok these look good - lets continue to implementation"), after quorum round 2. D1–D4 and D7 were taken as recommended.
- **Rulings copied, not gates:**
  - **D7:** contract lines go into the v2 signature set only. `meta:contracts:v1` is a set-level marker. `interface_hash_v2` stays unchanged.
  - **D5:** PUB025 ships as a badge first, following the `ReleaseGateHardFrom` pattern. This is agent-resolved.
- **There are no human-decision gates in this sprint.** No decision used by Phase 1 is still open.
- **Execution:** attended and in-session, in the main ailang checkout (`dev`). Execution was authorised by the same instruction. No coordinator task or `ailang agent send` handoff is used: a `sprint-executor` inbox send dispatches a cloud task, and the user is driving this session.
- No commit, push, release or publish is authorised by this plan. They need separate approval.

## Summary

Make the package cascade classify changes from measured symbol-level signatures, including contracts, instead of "assume breaking". Make lock writes atomic. Make nested packages real boundaries. This is Phase 1 of the design. Phases 2–4 are listed at the end as later sprints, each to be planned when reached.

**Duration:** ~3 working days, as 6 milestones of one executor session (≤ 90 min) each
**Dependencies:** none (all Phase 1 premises are verified: V6, V7, V8, V12, V24, V25, V31, V32)
**Risk Level:** Medium. M2 changes what the interface builder records, and that data is persisted in the compile cache.

## Current Status Analysis

### Velocity
- Last 7 days on `dev`: ~12 feature/fix commits, including M-AI-PREFERRED-LANGUAGE M1–M4 and M-PIPE-OPERATOR. Typical milestone: 150–350 changed lines including tests, landed in one session.
- Capacity planned here: ~930 changed lines including tests over 6 milestones, at 150–250 lines/milestone, with a 25% buffer in the day count.

### Remaining from the design doc
- ⏳ **Phase 1** (this sprint): atomic lock write; C4 (signatures through publish, contract lines, short-circuit removal, PUB025); C6 (nested-manifest boundary, import hints).
- 📋 Phase 2: C1 re-exports (Sprint 2).
- 📋 Phase 3: C2 checkpoints, C3 leases, full Phase 0 run, V35 (Sprint 3).
- 📋 Phase 4: C5 moves (Sprint 4).
- **Moved out of C6:** package-mode inline test discovery (TST-PKGPAR-23). It already has its own planned design, [m-package-test-discovery.md](../m-package-test-discovery.md) (issue #959, a 2-day estimate). This doc **depends on** it rather than duplicating it.

## Registry Reuse Audit

Every milestone changes the AILANG toolchain itself (`internal/pkg`, `internal/iface`, `internal/messaging`, `cmd/ailang`). Searches: `ailang pkg search lockfile`, `semver`, `interface diff`. None of the hits is a Go toolchain component.

| Milestone | Decision | Reason |
|-----------|----------|--------|
| M1–M6 | `none` | Compiler and CLI internals; no registry package can supply Go code inside the toolchain |

## AILANG Syntax Gate

No milestone writes showcase `.ail` modules. Test fixtures are minimal scratch packages written inside Go tests, which is the existing pattern (e.g. `effect_ceiling_reachable_test.go`). Contracts, effects and inline-test rows: skipped, because these are not showcase or demo modules.

## Proposed Milestones

### M1 ATOMIC-LOCK: atomic `ailang.lock` writes ✅
**Goal:** a concurrent reader never sees a truncated lock (V12: 16 of ~9,700 runs failed with `unexpected end of JSON input`).
**Estimated:** 25 implementation + 60 tests = **85 LOC**

- Add `writeFileAtomic(path, data, perm)` in `internal/pkg` (temp file in the same dir, fsync, `os.Rename`). There is no shared helper to reuse (V17).
- `LockFile.Save` (`internal/pkg/lockfile.go:87`) uses it.
- Test: one goroutine saving in a loop and four loading, for 5 s → 0 parse failures. Assert that no temp file is left behind.

**Acceptance criteria:**
- [x] TST-PKGPAR-14 passes; it fails when `Save` is reverted to `os.WriteFile` (check this once by mutation, locally). Pre-fix run: 1,363 torn reads in 2 s.
- [ ] `make test-changed` is green.

### M2 CONTRACT-SIGS: contracts in the interface and signature set ✅
**Goal:** contract changes become visible to classification without changing `interface_hash_v2` (D7; V24).
**Estimated:** 140 implementation + 110 tests = **250 LOC**

1. Add `Contracts []string` to `iface.IfaceItem` (`internal/iface/iface.go`). Populate it in `Builder.Build` next to the `fd.Deprecation()` block (`internal/iface/builder.go:338-341`), from `fd.Properties` with `Kind` ∈ {`RequiresKind`, `EnsuresKind`}. Each entry is a normalised string `requires <expr>` / `ensures <expr>`, built from a deterministic AST printer with whitespace collapsed. Entries keep source order.
2. Persist `Contracts` through the compile cache (`internal/pipeline/cache_store.go:247,337`, the same as `ReplacedBy`). A warm cache must not drop them. This is the risk called out below.
3. Add `Contracts []string \`json:"contracts,omitempty"\`` to `FuncJSON` (`internal/iface/json.go:141-152`).
4. `HashProjection` sets `out.Contracts = nil`, next to the Tier/ReplacedBy strip (`hash_projection.go:72`), so `interface_hash_v2` is byte-identical for every existing package.
5. `SignatureSet` emits `meta:contracts:v1` once per set, plus `mod:contract:<fn>:<escaped normalised>` per contract.
6. `classifyChange` (`internal/messaging/pkg_events.go:155`):
   - If either side lacks `meta:contracts:v1`, return **U**. This check comes after the existing legacy/U checks.
   - If the `mod:contract:` lines differ in any way, return **C**.
   - Otherwise, apply the existing same/superset rule to the non-contract lines.

**Acceptance criteria:**
- [x] TST-PKGPAR-19 passes, covering all six cases in its row (11 sub-cases).
- [x] **Golden check:** `interface_hash_v2` of every package under `internal/pkg/testdata` is unchanged before and after. Add an explicit test comparing against recorded hashes. *Done as:* `TestInterfaceHashV2_ContractsInSignaturesNotHash` through the real build, plus the real-package check that ailang-parse's hash is identical across the old and new binaries (618 → 704 signatures). No recorded-hash fixtures existed to compare against.
- [x] A warm-cache run yields the same `Contracts` as a cold one (`TestCacheIfaceRoundTripKeepsContracts`, plus `ailang iface` run cold and warm). The cache key went from v13 to v14.
- [x] `make test-changed`; `make check-file-sizes`. `builder.go` is at 672 lines: if the addition crosses 800, put the contract extraction in a new `internal/iface/contracts.go`. *Done:* extraction is in `contracts.go`. The one `test-changed` failure is pre-existing (`TestRealRegistry_PostMigrationSplitCounts`, from the committed `prompts/versions.json`). Stdlib goldens were refrozen for `audio` and `string`.

**Risks:** the AST expression printer may not be deterministic for all expressions. Mitigation: use the existing printer used by `ailang fmt` if it round-trips; otherwise canonicalise via core. Record the choice in the milestone notes.

### M3 PUBLISH-CLASS: measured classes at publish ✅
**Goal:** publish classifies from signatures, so additive changes stay deterministic and removals and contract changes are C (V6, V7, V31).
**Estimated:** 120 implementation + 120 tests = **240 LOC**

- `cmd/ailang/pkg_publish.go`:
  - Fill `PackageVersionInfo.Signatures` for **old**, from registry metadata `InterfaceSignatures` for the previously published version (read the existing old-info lookup first and reuse its source), and for **new**, from `pkg.InterfaceHashV2(...)`.
  - Replace the hash-equality short-circuit and module-count heuristic in `mapChangeClassToSchema` (`:512-534`) with `classifyChange` whenever both sides carry signatures. Keep today's behaviour for legacy pairs.
  - Set the local `changeClass` (`:368-379`) from the same result.
- Fill `AffectedExports` with the removed and changed signatures (old − new).
- `pkg_publish.go` is at 595 lines. If the change approaches 800, move the classification into `cmd/ailang/pkg_publish_classify.go`.

**Acceptance criteria:**
- [x] TST-PKGPAR-01 (superset → B, `breaking=false`), TST-PKGPAR-02 (removal → C, `affected_exports` names it), TST-PKGPAR-03 (a B envelope → deterministic dispatch), and TST-PKGPAR-28 (equal hash with a contract diff → C).
- [x] The existing `TestAutonomyRouter_LegacyEnvelopeRoutingUnchanged`, `TestEmitUpgradeAvailable` and `TestEmitUpgradeAvailable_V2PairCarriesBreaking` stay green. *Note:* the v2-pair fixtures gained the marker, because without it a pair is now U by design.
- [ ] `make test-changed`; `make check-file-sizes`.

**Risks:** registry metadata for the previous version may be missing signatures for older publishes. That correctly yields U through the existing path, and is asserted by a test case.

### M4 PUB025: release kind vs measured class ✅
**Goal:** `[release] kind` stops being an unchecked claim (V8, V19, V32).
**Estimated:** 50 implementation + 60 tests = **110 LOC**

- In `internal/pkg/quality.go`, add `PUB025`:
  - `kind` ∈ {fix, security, feature} with a measured C → a **badge** (a warning in `pkg quality` and at publish) while the running ailang version is below `ReleaseGatePUB025HardFrom`, then a gate.
  - `kind = breaking` with a measured A → warning only.
- The measured class comes from M3's result, passed in through `QualityInputs`.
- Register the code in `internal/errors/codes.go`.

**Acceptance criteria:**
- [x] TST-PKGPAR-04 passes (10 cases), plus the grace-schedule and previous-version tests.
- [x] The badge appears in `ailang pkg quality --json` output. A real run on ailang-parse 0.60.0 shows `measured_class: U` vs 0.59.0 and no PUB025, which is correct for a pre-marker previous version.
- [ ] `make test-changed`.

**Ruling copied (D5):** grace length is one minor version, mirroring PUB001/PUB002. The hard-from constant is set to the next minor after release: this sprint ships in v0.63.0 (v0.62.0 was cut before it was committed), so `ReleaseKindGateHardFrom = "0.64.0"`.

### M5 NESTED-BOUNDARY: a nested `ailang.toml` is a package boundary ✅
**Goal:** a package nested inside another is excluded from the outer package's compile, discovery, content hash and tarball (V3).
**Estimated:** 70 implementation + 90 tests = **160 LOC**

- Add one helper, `pkg.isNestedPackageRoot(dir, root)`: true when a directory below `root` contains `ailang.toml`.
- Skip such directories in:
  - the tarball walk (`internal/pkg/tarball.go:43`);
  - the content-hash walk (`internal/pkg/hasher.go`);
  - `copyPackageContents` (`publish_validator.go:160-180`);
  - package-mode module discovery (`internal/pkg/discover.go`, `loader_layout.go`).
- Check every other `filepath.Walk` in `internal/pkg` and record each site in the notes.

**Acceptance criteria:**
- [x] TST-PKGPAR-05: given a root package containing `packages/sub/ailang.toml`, `ailang publish --dry-run` at the root lists no `packages/sub/` files and raises no MOD010 for them. Real binary: 3 files/685 B with MOD010 → 2 files/560 B without.
- [x] The root content hash is unchanged by editing a file under `packages/sub/` (in `TestNestedPackage_IsABoundary`).
- [x] Existing ailang-packages monorepo fixtures (sibling, non-nested packages) are unchanged (full `internal/pkg` and `internal/check` suites green).
- [ ] `make test-changed`.

### M6 IMPORT-HINTS: import-path guidance ✅ (rescoped)

> **Rescoped during execution.** V4 was refuted live: a dependent's `import docparse/services/markdown_parser` resolves on v0.61.1 and dev, because bare imports resolve via the module_prefix map (`c59f8f916`). The IMP hint would have advised against a working import, so it was dropped. The `pkg docs` import lines were kept: they surface the unambiguous `pkg/` form. The false claim was corrected in four places.
**Goal:** an agent importing an `[exports]`-spelled path is told the `pkg/` path (V4).
**Estimated:** 50 implementation + 50 tests = **100 LOC**

- Where IMP010 / module-not-found is raised for an import (`internal/loader/loader_errors.go:34`), add a hint when the unresolved path's first segment equals a dependency's `module_prefix`: "did you mean `pkg/<vendor>/<name>/<rest>`?"
- `cmd/ailang/pkg_docs.go` prints one copy-pasteable `import pkg/<vendor>/<name>/<module>` line per exported module.

**Acceptance criteria:**
- [x] ~~A test with a dependent importing `docparse/services/x`: the error contains the `pkg/` form~~. Dropped: the import succeeds.
- [x] `ailang pkg docs` on a test package prints the import lines (`TestPrintAgentDoc_CanonicalImportLines`). The real `sunholo/ailang_parse` prints 79 lines.
- [ ] `make test-changed`.

## Success Metrics

- TST-PKGPAR-01, -02, -03, -04, -05, -14, -19 and -28 pass, plus the new golden-hash, warm-cache and IMP-hint tests.
- `interface_hash_v2` is unchanged for every existing fixture. This is the compatibility promise of D7.
- **Changelog fragment:** `changelogs/unreleased/2026-11-25-pkg-measured-change-classes.md`, opening with `### `.
- **Docs:** `docs/docs/guides/` package-publishing page: PUB025, measured classes, nested packages. Find the existing page with `ls docs/docs/guides | grep -i pack`.
- **Gates per milestone:** `make test-changed`, plus `make check-file-sizes` (and `make check-changelog` on the last milestone). The full suite runs in CI and is not an executor gate.

## Dependencies
- None blocking. Phase 1 is independent of Phases 2–4.

## Open Questions
- None. No decision is still open for Phase 1.

## Later sprints (planned when reached)

| Sprint | Design phase | Content | Pre-work |
|--------|--------------|---------|----------|
| 2 | Phase 2 | C1 re-exports: parser header loop, identity-preserving import resolution, v2 inclusion, "moved" `TC_DEPRECATED_001` variant, facade ceiling, MOD016, MOD003 removal (TST-PKGPAR-06, -09, -10, -16); engine parity | Conflict Surface fixtures (V28) |
| 3 | Phase 3 | C2 checkpoints (digest identity, snapshot-validate, HEAD lock, MOD017/018/019, PUB026, lock v2) and C3 leases and `pkg commit` (TST-PKGPAR-07, -08, -13, -15, -17, -18, -20..22, -24, -25) | Full Phase 0 run; V35 |
| 4 | Phase 4 | C5 moves (`pkg move`, `[[migrations]]`, PUB027, `import_rewrites`, deterministic-first dispatch for C migration-requests) (TST-PKGPAR-11, -12, -26, -27); guide | Sprint 2 (facades) |

## Notes
- **Working tree:** `dev` already has unrelated uncommitted changes (docs, `.ailang/state`) from other work. This sprint touches only Go files under `internal/` and `cmd/`, the design dir and one changelog fragment. Stage by explicit path when a commit is approved.
- The quorum pool change from earlier today (`codex:gpt-6.2-astra`, `gemini-4-carbon`, `claude-fable-5-2@claude-p`; quorum-only Astra allowance) is uncommitted in the same tree. It is separate from this sprint and should be committed separately.
