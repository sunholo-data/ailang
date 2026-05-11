# Sprint Plan: M-EXT-PORTABILITY-GATE (v0.19.0)

**Design doc**: [m-ext-portability-gate.md](./m-ext-portability-gate.md)
**Target**: v0.19.0
**Estimated**: ~1200 LOC across 4 repos, 2-3 days at current velocity (M-MATCH-ADT-XCHECK shipped ~760 LOC in 2 hours)
**Risk level**: Medium — multi-repo coordination, but each milestone is bounded and each repo's part is independently testable
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-11

---

## Discovery (pre-planning)

Verified what already exists vs needs creating:

| Component | State |
|---|---|
| `internal/pkg/tarball.go::CreateTarball` | EXISTS (filters .ail/toml/AGENT.md only) — extend |
| `internal/builtins/pkg*.go` | NONE — create from scratch |
| `std/package.ail` | NONE — create |
| `std/extension.ail` | NONE — create |
| `internal/pkg/publish_validator.go` | NONE — create |
| `internal/pkg/manifest.go::PackageManifest` | EXISTS — extend with `Assets AssetConfig` field |
| `cmd/ailang/pkg_publish.go` line 84 | EXISTS — single integration point for publish gate |
| `cmd/registry-validator/` | EXISTS (main.go, handlers) — defer server-side enforcement to M4 follow-up |

**Velocity calibration**: M-MATCH-ADT-XCHECK (today) shipped ~760 LOC end-to-end in ~2 hours. 1200 LOC for this sprint is realistic for 1-2 sessions.

---

## Milestones

### M1 — Asset bundling (~250 LOC, ~3 hours)

**Goal**: AILANG packages can ship arbitrary files in `assets/` and resolve them at runtime.

**Tasks**:

1. **Extend `PackageManifest`** (`internal/pkg/manifest.go`):
   - New `Assets AssetConfig` field on `PackageManifest` struct (`toml:"assets"`)
   - `AssetConfig{ Files []string }` — optional declaration of expected assets
   - Validation: if `[assets].files` declared, every listed file MUST exist in `assets/` subdir at publish time

2. **Extend `CreateTarball`** (`internal/pkg/tarball.go`):
   - Walk `assets/**` and include all files in tarball
   - Preserve sort order + zero ModTime for hash determinism
   - Reject path traversal (already protected by ExtractTarball; mirror in CreateTarball)

3. **New `_pkg_asset_path` builtin** (`internal/builtins/pkg.go` — NEW FILE):
   - Signature: `(pkg_name: string, rel_path: string) -> Result[string, string] ! {FS}`
   - Resolves to `~/.ailang/cache/registry/<pkg>/<version>/assets/<rel>`
   - Returns `Err("package not installed: <pkg>")` if package absent
   - Returns `Err("asset not found: <rel>")` if file missing
   - Pure value path lookup — no IO beyond `fileExists` check

4. **New `std/package.ail`**:
   - Single export `assetPath` wrapping `_pkg_asset_path`
   - Doc comments explaining when to use (bundling helper scripts, schemas, templates)

5. **Tarball hash regression test** (`internal/pkg/tarball_test.go`):
   - Existing package without assets/: hash unchanged from v0.18.10
   - Package with assets/foo.txt: hash differs (expected) but stable across runs

6. **Acceptance**:
   - [ ] `ailang publish --dry-run` for test package with `assets/foo.txt` includes file in tarball (verified by tar listing)
   - [ ] `assetPath("test/pkg", "foo.txt")` returns Ok(absolute path) when package installed
   - [ ] Returns Err with clear message when package absent or asset missing
   - [ ] Existing motoko-ext-* packages republish-equivalent (no breakage)
   - [ ] Tarball hash deterministic across runs (locked via test)

---

### M2 — Pre-publish durability gate (~400 LOC, ~4 hours)

**Goal**: `ailang publish` rejects extensions whose tools crash in an empty workdir.

**Depends on**: M1 (smoke test may use assets API)

**Tasks**:

1. **New `internal/pkg/publish_validator.go`**:
   - `RunSmokeInTempDir(packageDir string, timeout time.Duration) (PassedSmoke, Output, error)`
   - Steps:
     - Create `os.MkdirTemp("", "ailang-smoke-")` 
     - Copy package source into the temp dir
     - Generate temp consumer module that imports the package and runs `_smoke.ail` 
     - Spawn `ailang run --caps Net,AI,SharedMem,IO,Env,Clock,FS,Process,Stream --ai-stub --entry main _smoke.ail`
     - 30s context timeout (kills process group on timeout — uses Setpgid pattern from v0.18.9 design)
     - Capture stderr + exit code
   - Returns: passed (bool), captured output (string), Go error for infra failures

2. **Extended `_smoke.ail` template** (in motoko-ext-compaction-ai as the canonical example, then referenced by docs):
   - Step 1 (existing): call `register_with_config(0)`, assert no panic
   - Step 2 (NEW): for each name in `hooks.provided_tools`, call `hooks.on_tool_handle(synthetic_ctx, ToolCallEnvelope{...})` with synthetic args
   - Each tool call must NOT crash — Handled or Delegate are both acceptable
   - Print "OK: tool <name> dispatched without crash" per tool

3. **Wire into `cmd/ailang/pkg_publish.go`**:
   - Before `pkg.CreateTarball(cwd)` (line 84), look for `_smoke.ail` in package root
   - If present: call `pkg.RunSmokeInTempDir(cwd, 30*time.Second)`
   - On smoke fail: print captured output, abort with `Err("publish blocked: smoke test failed")`
   - On no smoke: print warning "package has no _smoke.ail; recommended for v1.0+"
   - If `[extension]` block in manifest: REQUIRE smoke (extensions must have it)

4. **Tests** (`internal/pkg/publish_validator_test.go`):
   - Smoke pass case: package with passing `_smoke.ail` returns passed=true
   - Smoke fail case: package with crashing `_smoke.ail` returns passed=false + captures stderr
   - Smoke timeout case: 1-second timeout against a `sleep 60` smoke; verifies timeout fires + process killed
   - Temp-dir isolation: smoke runs in dir that has no inherited workdir state

5. **Acceptance**:
   - [ ] `ailang publish` for compaction_ai@0.1.2 (existing well-formed) passes smoke
   - [ ] `ailang publish` for hand-crafted broken package (smoke crashes) BLOCKS publish with clear error
   - [ ] No-smoke publishes warn but succeed (back-compat)
   - [ ] Timeout enforced (30s default, configurable via `[smoke] timeout_seconds`)
   - [ ] Process-group kill on timeout (no orphaned find/sleep processes)

---

### M3 — `requireWorkdirFile` + extension updates (~200 LOC across 3 repos, ~3 hours)

**Goal**: motoko_ext_omnigraph and motoko_ext_mcp adopt the new pattern; motoko_agent pins to fixed versions.

**Depends on**: M1 + M2

**Tasks** (cross-repo):

1. **AILANG repo — `std/extension.ail`** (NEW):
   - `requireWorkdirFile(workdir: string, rel: string) -> Result[(), string] ! {FS}`
   - Returns `Err("required workdir file not found: <rel>")` if absent
   - One-liner wrapping `fileExists`

2. **ailang-packages — `motoko-ext-omnigraph@0.2.1`**:
   - `register.ail` calls `requireWorkdirFile(workdir, "omnigraph/omnigraph.yaml")` first
   - On Err: returns hooks with `provided_tools: []` (silent skip)
   - On Ok: existing behavior (advertise tools)
   - Add `_smoke.ail` exercising every provided_tool

3. **ailang-packages — `motoko-ext-mcp@0.2.1`**:
   - Bundle `assets/mcp-call.mjs` (move from motoko_agent's scripts/ to here)
   - `register.ail` calls `assetPath("sunholo/motoko_ext_mcp", "mcp-call.mjs")` 
   - On Err: returns hooks with `provided_tools: []`
   - On Ok: passes resolved path to exec.ail
   - `exec.ail` lines 120 + 132: replace `${workdir}/scripts/mcp-call.mjs` with the resolved path
   - Add `_smoke.ail`
   - Add `[assets].files = ["mcp-call.mjs"]` to ailang.toml

4. **motoko_agent — pin bump**:
   - `ailang.toml` + `ailang.lock`: motoko_ext_omnigraph 0.2.0 → 0.2.1, motoko_ext_mcp 0.2.0 → 0.2.1
   - `runtime-process.ts`: REMOVE the `MOTOKO_PROFILE_DIR` workaround if no longer needed (assess after the omnigraph self-skip lands)
   - Acceptance test: `cd /tmp/empty-workdir && motoko "test"` should NOT crash on omnigraph or mcp

5. **Acceptance**:
   - [ ] `std/extension.requireWorkdirFile` shipped + AILANG_RELAX_MODULES=1 ailang check clean
   - [ ] `motoko-ext-omnigraph@0.2.1` published; `_smoke.ail` passes in temp-dir
   - [ ] `motoko-ext-mcp@0.2.1` published with `assets/mcp-call.mjs`; `_smoke.ail` passes in temp-dir
   - [ ] motoko_agent type-checks (make check_core green) with new pins
   - [ ] motoko_explore (originally-affected workdir) verifies no longer crashing

---

## Day-by-day breakdown

| Session | Milestones | Hours | Deliverable |
|---|---|---|---|
| 1 (am) | M1: Asset bundling | 3h | tarball + assetPath + tests |
| 1 (pm) | M2: Publish gate | 4h | publish_validator + extended smoke template + integration |
| 2 (am) | M3 part 1: AILANG std/extension + omnigraph 0.2.1 | 1.5h | requireWorkdirFile shipped + omnigraph published |
| 2 (am) | M3 part 2: mcp 0.2.1 with bundled asset | 1.5h | mcp published with mcp-call.mjs in assets/ |
| 2 (pm) | M3 part 3: motoko pin bump + acceptance | 1h | motoko PR updated; verify in fresh workdir |
| 2 (pm) | Release v0.19.0 + CHANGELOG | 1h | tag, push, release notes |

Total: ~12 hours = 2 sessions.

---

## Repo coordination

| Repo | Branch | What lands |
|---|---|---|
| `sunholo-data/ailang` | `dev` | M1 (tarball + builtins + std/package.ail) + M2 (publish_validator + smoke gate) + M3 part 1 (std/extension.ail) + CHANGELOG + v0.19.0 release |
| `sunholo-data/ailang-packages` | `feat/portability-v0.19.0` (NEW) | motoko-ext-omnigraph 0.2.1 + motoko-ext-mcp 0.2.1 + per-package `_smoke.ail`s |
| `sunholo-voight-kampff/motoko_agent` | `feature/compaction-ai-profile-switching` (existing PR #16) | pin bump commit |
| `sunholo-data/ailang` | `cmd/registry-validator/` | M4 follow-up sprint — server-side enforcement (NOT in this sprint scope) |

---

## Success metrics

- **Bug class closed**: `ailang publish` rejects extensions whose tools crash in an empty workdir.
- **Asset bundling unblocks**: motoko_ext_mcp ships `mcp-call.mjs` inside the package; consumers no longer need to vendor it.
- **Self-disable pattern documented**: `std/extension.requireWorkdirFile` + omnigraph as canonical example.
- **Bug-discovery shifts left**: future "registers fine but tool crashes" bugs caught at publish time, not consumer-runtime time.
- **No regression**: all existing motoko-ext-* packages publish unchanged (or with the new pattern adopted incrementally).

---

## Risks

| Risk | Mitigation |
|---|---|
| Smoke gate too strict for legacy packages | Soft-launch: warn-only when no `_smoke.ail`. Hard-fail only when `[extension]` block declared. |
| Asset bundling breaks tarball-hash determinism | Sort assets lexically + zero ModTime (mirror existing pattern); regression test locks the hash. |
| Smoke timeout fires on slow CI | Default 30s, override via `[smoke] timeout_seconds`. |
| Process-group kill on timeout breaks on Windows | POSIX-only Setpgid path; Windows keeps existing CommandContext (separate codepath, build tag). Same approach as M-PROCESS-PGID-CLEANUP design (planned v0.18.9). |
| Cross-repo coordination has merge conflicts | Land AILANG-side first (M1+M2+M3 part 1 in one commit chain); then ailang-packages (no AILANG dep change after); then motoko_agent (just pin bump). |
| `_smoke.ail` for tools with side effects (Process, Net) needs mocking | Use `--ai-stub` for AI; document that Process/Net tools must guard against missing infrastructure (return Delegate, not crash). |

---

## Files modified

| Repo | File | Change | LOC |
|---|---|---|---|
| ailang | `internal/pkg/manifest.go` | + `Assets AssetConfig` field | +30 |
| ailang | `internal/pkg/tarball.go` | walk assets/ subdir | +25 |
| ailang | `internal/pkg/tarball_test.go` | hash determinism + assets test | +60 |
| ailang | `internal/pkg/publish_validator.go` | NEW — smoke runner + temp-dir + timeout | +180 |
| ailang | `internal/pkg/publish_validator_test.go` | NEW — pass/fail/timeout tests | +120 |
| ailang | `internal/builtins/pkg.go` | NEW — `_pkg_asset_path` builtin | +60 |
| ailang | `cmd/ailang/pkg_publish.go` | wire smoke gate before tarball | +25 |
| ailang | `std/package.ail` | NEW — `assetPath` export | +30 |
| ailang | `std/extension.ail` | NEW — `requireWorkdirFile` export | +20 |
| ailang | `internal/pipeline/testdata/builtin_types.golden` | regen for new builtin | +1 |
| ailang | `changelogs/v0.10-current.md` | [v0.19.0] section | +50 |
| ailang | `std/VERSION` | v0.18.10 → v0.19.0 | 1 |
| ailang | `docs/src/constants/version.js` | STABLE_RELEASE | 1 |
| ailang | `docs/docs/reference/extension-portability.md` | NEW — pattern guide | +120 |
| ailang-pkgs | `packages/motoko-ext-omnigraph/register.ail` | self-disable pattern | +25 |
| ailang-pkgs | `packages/motoko-ext-omnigraph/_smoke.ail` | NEW | +30 |
| ailang-pkgs | `packages/motoko-ext-omnigraph/ailang.toml` | 0.2.0 → 0.2.1 | 1 |
| ailang-pkgs | `packages/motoko-ext-mcp/register.ail` | use assetPath | +20 |
| ailang-pkgs | `packages/motoko-ext-mcp/exec.ail` | replace workdir hardcode | +10 |
| ailang-pkgs | `packages/motoko-ext-mcp/assets/mcp-call.mjs` | NEW — bundled bridge | +75 (copied) |
| ailang-pkgs | `packages/motoko-ext-mcp/_smoke.ail` | NEW | +30 |
| ailang-pkgs | `packages/motoko-ext-mcp/ailang.toml` | + [assets] block, 0.2.0 → 0.2.1 | +5 |
| motoko_agent | `ailang.toml` + `ailang.lock` | pin 0.2.0 → 0.2.1 | +4 |
| **Total** | | | **~920 LOC** |

(Estimate revised down from ~1200 because the design doc was conservative. Realistic actual at current velocity: ~900-1000 LOC.)
