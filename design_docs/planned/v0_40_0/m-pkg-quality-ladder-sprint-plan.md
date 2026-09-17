# Sprint Plan: M-PKG-QUALITY-LADDER — Sprint 1 "measure" (M1–M5)

**Design doc**: [m-pkg-quality-ladder.md](m-pkg-quality-ladder.md) (D1–D7 ratified by Mark, attended 2026-09-17)
**Sprint ID**: M-PKG-QUALITY-LADDER-S1
**Target**: v0.40.0
**Duration**: 3 days (~20 h) · **Risk**: medium (two registry-facing seams; one schema extension)
**Executor**: attended session 2026-09-17, worktree-isolated, PR to `dev`
**Status**: ✅ M1–M6 complete 2026-09-17 (M6 added mid-sprint by Mark: package inbox agents)

Sprints 2 ("route", M6–M9) and 3 ("tighten", M10–M12) are outlined at the end and are re-planned
against Sprint 1's measured outcome (the D6 shadow count), not now.

## Velocity

Last 7 days on `dev`: design-doc and triage commits plus the v0.39.4 release and the perf-sweep
feature (~1.4k LOC incl. tests). Comparable prior work — the parent doc's Sprint 1 (M1–M5, hash v2)
landed ~810 LOC + ~770 test LOC in 4 days. Target here: ~1,100 LOC + ~800 test LOC in 3 days, with
30% buffer already in the per-milestone hours.

## Measured base state (2026-09-17, v0.39.4-10-g6f3a96683)

| Fact | Where |
|---|---|
| `runAilangVerify` runs `ailang verify --json <abs file>` per file → MOD010 on every flat package; and decodes `[]struct{Status}` while verify emits `{results:[…]}` → 0 counted, 0 skipped | `cmd/registry-validator/validate.go:108-142`; design V2/V3 |
| `check --package` compiles each discovered file with `pipeline.Config{DryLink, RelaxModules: true}` and the loader's self-package resolution from the manifest | `internal/check/package.go:75-125` |
| `BuildCanonicalJSON` joins `packageDir/<modulePath>.ail` and does NOT relax MOD010 → 0/53 flat packages build; canonical restage breaks self-imports instead | `internal/pipeline/canonical_json.go:18-25`; design V7/V8 |
| `resolveModuleToFile(pkgDir, pkgName, modulePath)` exists, unexported | `internal/pkg/discover.go:128` |
| `PublishLimits{Overall 60s, PerModule 10s, MaxExportedModules 64}`; `ailang_parse` exports 56 | `internal/pkg/iface_subprocess.go:20-33` |
| `InterfaceHashV2` has no non-test caller | design V6 |
| Unknown `[release]` TOML section is ignored by shipped binaries | design V13 |
| `PUB0xx` codes unallocated | design V12 |
| `ailang pkg` has 8 subcommands, no `quality` | `cmd/ailang/pkg_commands.go` |

## Milestones

### M1 — One package-aware module resolution for verify + iface (5 h, ~220 LOC + ~200 test)

Tasks:
1. Export `pkg.ResolveModuleToFile` (rename, keep the unexported alias for existing callers) and add `pkg.ResolveExportedModules(dir, manifest) (map[modulePath]file, missing []string)`.
2. `pipeline.BuildCanonicalJSON`: when `packageDir != ""`, resolve `modulePath` through the manifest (flat root, `src/`, canonical subdir, `module_prefix`) instead of joining the path; run with `RelaxModules: true` so the flat layout's MOD010 is a warning, as `check --package` already does. Keep the std/ embedded fallback and the no-package path byte-identical.
3. `ailang verify --package <dir> [--json] [--timeout]`: discover sources (`check.DiscoverPackageSources`), compile each with the same config as `CheckPackageFiles`, run `verifyModulesFromPipeline`, aggregate; JSON = `{"package":…, "modules":[{file, verified, counterexample, skipped, errors, total_exported, results:[…]}], "verified":N, "total":N, "counterexample":N, "skipped":N}`. Single-file `ailang verify <file>` unchanged.
4. Fixtures under `internal/pkg/testdata/quality/`: `flat_self_import/` (deontic shape: root files, `pkg/<self>/types` import, 2 contracts), `prefixed/` (module_prefix shape). Tests: `InterfaceHashV2` builds both (signature set non-empty, deterministic across two runs); `verify --package` on `flat_self_import` reports `verified=2`.

Acceptance:
- [x] `ailang internal-dump-iface . sunholo/deontic/settle` succeeds on the real flat deontic 0.3.0 tarball layout (7 signatures; `verify --package` 7/7 — recorded 2026-09-17).
- [x] `ailang verify --package` on the fixture: `verified ≥ 2`, exit 0; the per-module JSON shape is pinned by a test.
- [x] Mutation arm: revert the `RelaxModules` line in `BuildCanonicalJSON` → fixture test fails.

### M2 — Validator wiring, shadow (4 h, ~200 LOC + ~150 test)

Tasks:
1. Replace `runAilangVerify` with a call to the new `verify --package --json` and decode the real shape; add a per-package wall cap (`PublishLimits.VerifyWall`, default 120 s) — on cap, bank `skipped` for the remainder and log.
2. After `runAilangCheck` (lock written), compute `pkg.InterfaceHashV2` with `DefaultPublishLimits()` (raise `MaxExportedModules` to 128 — `ailang_parse` exports 56). Success → bank `interface_hash_v2`, `interface_signatures`; failure → bank `interface_v2_error` and log `v2=fail reason=…` (shadow; **not** a refusal, PUB005 is a badge until D6's N=20).
3. Publisher side (`pkg_publish.go`): compute V2 locally too and send `X-Interface-Hash-V2`; validator compares when both present → mismatch = `400 PUB005 version skew` with both values (the parent doc's M7 guard).
4. `PackageMetadata` schema `ailang.package-metadata/v2`: add `InterfaceHashV2 string`, `InterfaceSignatures []string`, `InterfaceV2Error string`, `Quality *QualityBlock` (all `omitempty`). `index.json` gains `contracts_total`. Existing readers (`pkg info`, explorer, `FetchMetadata`) keep working — pinned by a test that decodes a v1 fixture.
5. Seam test: `handlePublish` in validation-only mode on the `flat_self_import` fixture tarball returns metadata with `contracts_verified=2, contracts_total=2` and a non-empty `interface_hash_v2`; mutation arm: swap the decoder back to `[]struct{Status}` → test fails.

Acceptance:
- [x] Validator banks real contract counts and v2 identity; the shadow log line exists.
- [x] Skew guard test (publisher hash ≠ validator hash → 400 with both).
- [x] v1 metadata fixture still decodes.

### M3 — `ailang pkg quality [--json] [--strict] <dir>` (5 h, ~350 LOC + ~250 test)

Tasks:
1. `internal/pkg/quality.go`: `QualityReport(ctx, dir, Mode{Publisher|Server}, attested *Attested) (*Report, error)`; sections `compile`, `contracts`, `interface`, `effects` (Sprint 1: `max`, `ceiling_declared`, `rank_max` only — `measured` is Sprint 3), `release`, `docs`, `style` (`pure_ratio` from the v2 signature set's `pure` flags), `tests`/`smoke` as `attested` in Publisher mode (tests via `ailang test --package` JSON — the union with inline blocks is m-package-test-discovery; if it has not landed, report `files` only and say so in `notes`), `tier` (Sprint 1: computed but informational), `gates`, `badges` with the PUB table and the stability split (D3).
2. `cmd/ailang/pkg_quality.go` + dispatch in `pkg_commands.go`; human output mirrors the design's Example 1; `--strict` promotes badges to gates locally; exit 2 on any gate.
3. `publish --dry-run` prints the same report and exits non-zero on a gate; `publish` (non-dry) runs it before upload and refuses on publisher-local gates (smoke already does this — fold `runPrePublishSmoke` into the `attested.smoke` field, behaviour unchanged).
4. ailang-packages follow-up PR (separate repo): replace the phantom text in `AGENTS.md`, `.agents/skills/ailang-packages/SKILL.md`, `README.md` with the real flags.

Acceptance:
- [x] `ailang pkg quality --json` on the fixture validates against the schema in the design; `server` fields identical whether computed in Publisher or Server mode (seam test).
- [x] `--strict` turns PUB011 into a gate on a fixture with an uncontracted export.
- [x] `publish --dry-run` exits 2 on PUB001 when `CHANGELOG.md` is absent (after M4).

### M4 — `[release]` + `CHANGELOG.md` gate (3 h, ~180 LOC + ~120 test)

Tasks:
1. `ReleaseConfig{Kind, Notes string}` on `PackageManifest` (`[release]`); `Validate` accepts empty (grace) or one of `security|fix|feature|breaking`, else `PUB002`.
2. `pkg.ChangelogSection(dir, version) (text string, ok bool)`: accepts `## 0.8.2`, `## [0.8.2]`, `## v0.8.2 …`; non-empty body required.
3. Gates: PUB001 (section missing/empty), PUB002 — **grace window**: badges in v0.40.0, gates from v0.41.0 (constant `ReleaseGateHardFrom = "0.41.0"` compared to the validator's own version; mirrors `--allow-dotted-tool-names`). Bank `release.kind` + first 2 KB of the section in `metadata.json`; `release_kind` in `index.json`; `pkg info` prints both.
4. `ailang init package` scaffolds `CHANGELOG.md` with a `## <version>` section and `[release] kind = "feature"`.
5. Tarball inclusion: `CreateTarball` must ship `CHANGELOG.md` (check `shouldStageFile`/tarball allowlist — verify, do not assume).

Acceptance:
- [x] Fixture without CHANGELOG → PUB001 badge on v0.40.0 binary; test with the hard-from constant lowered → gate.
- [x] `metadata.json` carries `release.kind` and notes; `pkg info` shows them.
- [x] `init package` output passes `pkg quality` with zero gates.

### M6 — Every published package gets an agent inbox (added by Mark mid-sprint, 3 h, ~200 LOC + ~120 test) ✅

Measured: 29 hand-written `pkg-*` agents + one `pkg:sunholo/motoko_ext_*` family pattern serve 41/53 packages; **12 packages have no inbox** — a message to them is accepted and never dispatched (the `sunholo/email` incident of 2026-09-07). Every hand-written entry differs only in id/label/inbox/workspace/subdirectory/artifact_patterns, all derivable from `metadata.repository`.

Tasks: `pkg.ParseRepositoryURL` + `PackageAgentID`; `AgentRegistry.MaterializePackageAgents(index)` clones the `package_agent_template` config section (not an agent: it serves no inbox, so typo inboxes still bounce) per index package lacking an exact agent (hand-written wins, idempotent); daemon init + 10-min refresh; `buildRegistryFromConfig` materializes for the CLI readouts; `PUB021` badge when the URL is unparseable. **Follow-up outside this repo:** add the `package_agent_template:` section to `ailang-multivac/config/config.cloud.yaml` (workspace `sunholo-data/ailang-packages`, the standard `ailang_only` pi lane).

Acceptance:
- [x] Derived agent carries inbox/workspace/subdirectory/merge_branch from the URL; template fields (provider, policy) carried; slices not shared.
- [x] Hand-written entry wins; second materialize adds nothing; a family pattern is not mistaken for the template.
- [x] Non-GitHub / blob URLs derive nothing.

### M5 — Shadow instrumentation + docs (3 h, ~80 LOC + docs) ✅

Tasks:
1. Validator: one structured log line per publish `quality: pkg=… v2=ok|fail contracts=v/t tier=N gates=[…]`; `/api/stats` gains `v2_clean_streak` (count of consecutive publishes with `v2=ok`) so D6's N=20 is readable without log access.
2. Docs: `docs/docs/guides/packages.md` (or the package-authoring page) — changelog + `[release]`, `pkg quality`, what dry-run now checks; `ailang docs package-authoring` source; CHANGELOG.md entry under v0.40.0.
3. `make simplicity-audit` — one new command, two manifest sections, zero new env vars.

Acceptance:
- [x] `/api/stats` exposes `v2_clean_streak`; unit test on the counter.
- [x] Docs updated; `make ci` green.

## Day plan

| Day | Work |
|---|---|
| 1 | M1 (resolver + `verify --package` + V2 on flat layouts), M2.1–2.2 |
| 2 | M2.3–2.5, M3 |
| 3 | M4, M5, PR, ailang-packages follow-up PR |

## Files (Sprint 1)

- `internal/pkg/discover.go` (+30), `internal/pkg/quality.go` (NEW ~350), `internal/pkg/quality_test.go` (NEW ~250), `internal/pkg/manifest.go` (+40), `internal/pkg/registry_types.go` (+40), `internal/pkg/changelog.go` (NEW ~60) + test, `internal/pkg/iface_subprocess.go` (+10), `internal/pkg/testdata/quality/**` (NEW fixtures)
- `internal/pipeline/canonical_json.go` (~+25/−5) + test
- `cmd/ailang/verify.go` (+90 `--package`), `cmd/ailang/pkg_quality.go` (NEW ~150), `cmd/ailang/pkg_commands.go` (+8), `cmd/ailang/pkg_publish.go` (~+60/−20), `cmd/ailang/pkg_init.go` (+30), `cmd/ailang/pkg_info.go` (+15)
- `cmd/registry-validator/validate.go` (~+60/−70), `cmd/registry-validator/main.go` (+80), `cmd/registry-validator/handlers_api.go`/`cache.go` (+30), tests
- docs + CHANGELOG

## Risks

| Risk | Mitigation |
|---|---|
| `BuildCanonicalJSON` change alters `ailang iface` output for non-package files | packageDir == "" path untouched; existing `interface_output_test.go` + `internal_dump_iface_test.go` pin it |
| Validator deploy is release-gated — Sprint 1's server behaviour lands only at v0.40.0 | Publisher-side `pkg quality` is useful immediately; the shadow count starts at the release |
| m-package-test-discovery not landed → `tests` block incomplete | Report `files` only with an explicit `notes` entry; never fabricate inline counts |
| Grace-window constant forgotten | Test asserts the hard-from version is > the current `std/VERSION` while grace is intended, and a CHANGELOG note names v0.41.0 |

## Sprint 2 — "route" (M6–M9, outline; plan after the shadow count)

M6 ladder + PUB003/PUB004 (1 d) · M7 `AdjustAutonomyForTier` + Tier-3 checkpoint (0.5 d) · M8 `contract-regression` producer (0.5 d) · M9 cascade re-arm, **after M6/M7** (1 d). Entry condition: M2 banked on ≥ 1 real publish and the router's `U` path exercised by a fixture.

## Sprint 3 — "tighten" (M10–M12, outline)

M10 `effects_measured` via `collectRequiredEffects` + PUB010 (1 d) · M11 `[effects.scopes]` (1 d) · M12 `ailang policy derive` (1 d). Entry condition: Sprint 2 merged; D4 rank table in code.
