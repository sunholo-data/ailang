# M-PKG-SMOKE-STAGED-LOCK: stage the *published* view (manifest + lock) for the temp-dir smoke

**Status**: Planned
**Target**: v0.54.7
**Priority**: P1 (any monorepo package with a path dependency and a `_smoke.ail` cannot publish — the publish gate is structurally broken for the path-dep workflow the CLI itself promotes)
**Estimated**: ~1.5–2 days (~450 LOC incl. tests)
**Dependencies**: None
**Author**: design-doc-creator session (task-98f9f628), from the v0.54.5 user report (macOS arm64, commit 296231d)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The staged smoke view becomes a pure function of the *published* manifest (pinned registry versions) instead of an accident of the developer's directory layout |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | The portability gate verifies what it claims to verify: the tarball a consumer installs, not a half-local hybrid that matches no real deployment |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | PUB015 currently reports `passed:false` with zero diagnostic output — an agent hitting it must re-run the smoke manually; the fix puts the cause in the machine-readable report |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | `pkg quality`, `publish --dry-run` and `publish` now share one staging semantics instead of diverging |
| A11: Structured Failure | +1 | PUB015 carries the smoke's failing output; `ailang lock` warns instead of silently ignoring a declared path dep |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: no new nondeterminism — resolution is pinned by exact versions written into the staged manifest
- [x] A3: no hidden side effects — the registry fetch during re-lock is the same one `ailang lock` performs
- [x] A4: no ambient access granted
- [x] A7: not human-convenience-over-machine — the primary beneficiary is the PUB015 report consumer (agents, CI)

## Problem Statement

`ailang publish` rewrites `[dependencies]` path entries to registry versions before staging
(`Rewrote dep sunholo/social_dynamics: path ../social-dynamics -> registry 0.3.0`,
`cmd/ailang/pkg_publish.go:rewritePathDepsForPublish`), but `RunSmokeInTempDir`
(`internal/pkg/publish_validator.go:copyPackageContents`) copies the developer's
`ailang.lock` **unchanged** into the temp dir. That lock still records the dependency as
`source: "path", path: "../social-dynamics"` — *relative to the source directory*, which the
temp dir does not have. The loader (`internal/pkg/loader.go:packageDir`) joins the relative
path with the lock's directory, the directory does not exist, and the smoke fails:

```
Error: failed to resolve package import "pkg/sunholo/social_dynamics/model": package directory not found: <tmp>/../social-dynamics
PUB015 _smoke.ail failed
```

**Reproduced empirically** (v0.54.5 binary, commit 296231d — the reported build), with a
two-package monorepo `a` (path dep `../b`, `_smoke.ail` importing `pkg/sunholo/b/model`):

1. Smoke **passes in the package dir** (`ailang run --entry main _smoke.ail` → `hello from b`).
2. Manually staging exactly what `copyPackageContents` stages (ailang.toml + `*.ail` + the
   developer's `ailang.lock`) into a temp dir and running there:

   ```
   Warning: failed to hash dependency sunholo/b at /tmp/b: failed to walk directory: ...
   Error: ... package directory not found: /tmp/b
   ```

3. `ailang pkg quality --strict .` → `✗ PUB015 _smoke.ail failed`, and
   `ailang pkg quality --json .` shows the smoke section as
   `{"present": true, "passed": false, "seconds": ...}` — **no output field exists**; the
   `SmokeResult.Output` that `RunSmokeInTempDir` captured is discarded by
   `runAttestedChecks` (`cmd/ailang/pkg_quality.go`), and the PUB015 finding message is the
   bare string `"_smoke.ail failed"` (`internal/pkg/quality.go`).

So the bug hits **both** `ailang publish` and `ailang pkg quality` (which runs the same
`RunSmokeInTempDir`): any package with a path dependency and a `_smoke.ail` gets a PUB015
gate with no diagnostic, and cannot publish at all. The in-tree workaround is to pin the
sibling from the registry instead of a path (`sunholo/crew_core` pins
`sunholo/social_dynamics = "0.3.0"`, whose content hash matches local source) — i.e. the
path-dep workflow is currently unusable for publishing, silently pushing monorepos onto
hand-maintained version pins.

**Impact:** every monorepo package using the documented path-dependency workflow
(`ailang.toml` `"dep" = { path = "../dep" }`). The gate fails in the *confusing* direction:
the smoke is fine, the package is publishable, but the staging is wrong.

### Related defect (same root, reported alongside)

In a consumer with path deps on both A and B, where A pins B from the registry, `ailang lock`
silently resolves B from the registry and **ignores the consumer's own path entry for B**
(mechanism: `internal/pkg/resolver.go` — root deps are visited in sorted order; A's registry
pin resolves B first; when the root's `B = {path=...}` is processed, `visited["B"]` is
already true and the `resolvedSet` version check passes when versions match, so the path
branch is skipped without a word). The consumer's local edits to B are then silently
ignored. This deserves at least a warning.

## Goals

**Primary Goal:** the temp-dir smoke runs against the *published* view — the rewritten
manifest **and** a lock regenerated from it — so path-dep packages publish and PUB015
carries the smoke's output when it fails.

**Success Metrics:**
1. The repro above (`a` with path dep `../b` + `_smoke.ail`) publishes (dry-run) and reports
   `smoke: present true passed true`, with the smoke resolving B from the registry.
2. A *failing* smoke reports PUB015 with the smoke's captured output (human and `--json`).
3. `ailang lock` prints a warning when a root path dep was shadowed by a registry resolution.
4. `sunholo/crew_core`-style registry pins are no longer required to publish from a monorepo.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| The staged smoke resolves deps from the **registry** (published view), not from staged sibling copies of path deps | Defines what the portability gate *means*: it must test what a consumer installs | human | design | high |
| Re-lock the staged copy (regenerate the lock against the rewritten staged manifest) rather than rewriting path entries inside the developer's lock | Lock entries carry content hashes that must match the registry copy, not local source; re-lock reuses one well-tested code path (`ResolveLock`) | agent | design | med |
| `pkg quality`'s smoke uses the same published view as `publish` | One staging semantics; today `pkg quality` path-dep packages fail PUB015 anyway (verified), so this is a message improvement, not a behavior regression | agent | design | med |
| PUB015 surfaces the smoke output via the existing `SmokeSection.Notes` field (bounded excerpt) instead of a new JSON field | The smoke section is uploaded as the attested block and banked by the validator; a bounded reuse keeps the schema stable | agent | design | low |
| The registry-shadowing fix is a **warning** in `ailang lock`, not a resolution-precedence change | Changing precedence (root path wins over transitive registry pin) alters lock output for existing monorepos; warning first, semantics later | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Registry-view staging (Decision 1) — chosen per the reporter's expectation ("the smoke
  resolves B from the registry, as the manifest rewrite does") and the gate's charter
  (M-EXT-PORTABILITY-GATE: "no consumer-state assumed"). Alternative rejected below.
- [x] Shadowing fix is warning-only (Decision 5).

## Solution Design

### Overview

`RunSmokeInTempDir` currently copies files and runs; it must instead **assemble the
published view** in the temp dir, then run:

1. `copyPackageContents` (as today — file set unchanged, see Phase 4 note).
2. **Rewrite path deps in the *staged* manifest** to registry version strings, resolving
   dep paths against the *original* `packageDir` (the logic currently in
   `cmd/ailang/pkg_publish.go:rewritePathDepsForPublish`, generalized and moved to
   `internal/pkg`). Idempotent: a manifest with no path deps is untouched, so the publish
   flow (which already rewrote the cwd manifest before calling in) is unaffected.
3. **Re-lock the staged dir**: drop the copied `ailang.lock` and regenerate it with
   `ResolveLock(stagedManifest, stagedDir, ...)` — exactly what the reporter validated by
   hand ("passes in the staged dir after switching the dep to `0.3.0` and running
   `ailang lock`"). This fetches missing registry deps into the local package cache, the
   same code path `ailang lock` uses (honoring `AILANG_REGISTRY`).
4. Run the smoke (as today).

Separately: **surface the output** (PUB015) and **warn on shadowing** (`ailang lock`).

### Options Considered

**A. Stage the published view: rewrite staged manifest + re-lock the staged dir (CHOSEN).**
The smoke tests exactly what a consumer gets. If the path dep's local version is not
published, the re-lock fails with `failed to download <dep>@<version> from registry` —
which is the correct pre-publish signal (the tarball would be uninstallable), and for
`pkg quality` it replaces today's cryptic `package directory not found: <tmp>/../b` with
the honest reason. If local source has drifted from the published version, the smoke tests
the *published* content while the compile check in the real dir still tests local source —
each instrument measuring what it claims.

**B. Rewrite the path entries inside the staged lock (REPORTER'S ALTERNATIVE, REJECTED).**
Converting `source:"path"` entries to `source:"registry"` in place keeps the local content
hash, which (a) does not match the registry copy's hash when source has drifted →
hash-mismatch warnings, wrong view; and (b) still needs a download when the exact version
isn't in the local cache — converging on A's fetch while doing it worse. A is the same work
with correct hashes (`ResolveLock` computes them from the registry copy).

**C. Stage path-dep siblings into the temp tree so relative lock paths resolve (REJECTED).**
The staged manifest would keep path deps, so the smoke would pass on local sources while
the published tarball references registry versions — the gate would verify a deployment
that does not exist. Inverts the portability gate's purpose (M-EXT-PORTABILITY-GATE was
created precisely because smoke-in-authors-workdir hid consumer breakage).

### Architecture

**Components:**

1. **`internal/pkg/publish_rewrite.go` (new)** — `RewritePathDeps(targetDir, resolveFromDir string) (bool, error)`:
   reads `<targetDir>/ailang.toml`, and for each `[dependencies]` entry with `path`,
   loads the dep manifest at `resolveFromDir/<path>`, rewrites the entry to
   `"<name>" = "<depVersion>"` (the existing whitespace-tolerant regex from
   `rewritePathDepsForPublish`), writes the staged manifest back. `cmd/ailang/pkg_publish.go`
   keeps calling the same logic for its cwd rewrite (tarball creation still needs the
   rewritten manifest on disk in cwd — unchanged).

2. **`RunSmokeInTempDir`** — after `copyPackageContents`: call `RewritePathDeps(tmpDir,
   packageDir)`; on error return it as an infrastructure error (the smoke cannot be
   staged, mirroring "cannot read dependency … at publish time"). Then remove the staged
   `ailang.lock` and `ResolveLock` in `tmpDir`. On re-lock error: do **not** silently
   proceed — write the staged lock *without* it the run would produce today's confusing
   failure, so return the resolve error via a failing `SmokeResult` (Passed=false, Output
   carrying the resolve error), so the author sees the real cause under PUB015.

3. **Output surfacing** — `runAttestedChecks` (`cmd/ailang/pkg_quality.go`): on
   `!res.Passed`, set `smoke.Notes` to a bounded excerpt (≤ 2 KiB, e.g. first line +
   last N bytes of `res.Output`, plus `timed out after Xs` when `res.TimedOut`).
   `BuildQualityReport` (`internal/pkg/quality.go`): the PUB015 finding message becomes
   `"_smoke.ail failed: " + firstLine(s.Notes)`. No new JSON field; the attested upload
   stays schema-stable. `SmokeSection.NotRun` semantics are untouched (Notes is already
   used verbatim by the not-run badge).

4. **Lock shadowing warning** — `pkgLockCommand` (`cmd/ailang/pkg_commands.go`), after
   `ResolveDependencies`: for each root manifest dep with `Path != ""`, if the resolved
   entry's `Source != "path"`, print a yellow stderr warning:
   `dependency <name> is declared as a path dep but was resolved from the registry (already pinned via a transitive dependency); your local <name> source is NOT in this lock`.
   Style: mirror `warnSilentRatchet` (best-effort, stderr, non-fatal). Same check in
   `pkgLockCheck` (`lock --check`) as drift-style output is optional latitude.

### Conflict Surface

Not a parser/typechecker/codegen change (`internal/pkg` + `cmd/ailang` tooling only), so
no syntactic conflict surface. **Behavioral conflict surface** — existing flows that must
keep working:

1. **Publish of a package with *no* path deps**: rewrite is a no-op, re-lock of a
   registry-only manifest reproduces an equivalent lock (same resolution path as
   `ailang lock`); smoke unchanged. Fixture: `TestRunSmokeInTempDir_Pass`
   (self-contained package) must still pass byte-for-byte in outcome.
2. **`pkg quality --no-run`**: stages nothing, runs nothing — untouched.
3. **Extension packages** (PUB014 mandatory-smoke, M-EXT-PORTABILITY-GATE): same staging
   path; extension smoke that relies on `assets/` still works (assets are staged today and
   the tarball ships them).
4. **Registry smoke runs inside the pipeline/validator** — the validator never executes
   smoke (attested-only, by design); unchanged.
5. **Intentional change**: `pkg quality` / `publish` on a package whose path dep is *not
   published at the local version* now fails PUB015 with `failed to download <dep>@<ver>
   from registry` instead of `package directory not found: <tmp>/../<dep>`. This is the
   gate becoming honest, and is called out in the changelog.

### Implementation Plan

**Phase 1: staged published view** (~0.5 day)
- [ ] Extract `RewritePathDeps` to `internal/pkg/publish_rewrite.go` (generalized over
      target dir / resolution dir); `pkg_publish.go` calls it (behavior identical).
- [ ] `RunSmokeInTempDir`: staged-manifest rewrite + drop staged lock + `ResolveLock` +
      failing-SmokeResult-on-resolve-error.
- [ ] Test: the two-package repro (path dep + `_smoke.ail` importing `pkg/<dep>/...`)
      against an `httptest` registry (pattern: `internal/pkg/resolver_test.go:282`,
      `AILANG_REGISTRY` env) — smoke passes, lock in temp dir is registry-sourced.
- [ ] Test: dep not published at local version → PUB015 with `failed to download` in the
      message.

**Phase 2: PUB015 output surfacing** (~0.25 day)
- [ ] `runAttestedChecks`: bounded Notes excerpt + timeout marker.
- [ ] `quality.go`: PUB015 message includes `firstLine(notes)`.
- [ ] Test: failing smoke (existing `TestRunSmokeInTempDir_Fail` fixture) → report's
      smoke section carries the crash line; `--json` includes it via `notes`.

**Phase 3: lock shadowing warning** (~0.25 day)
- [ ] `pkgLockCommand` post-resolution comparison; stderr warning.
- [ ] Test: consumer with path deps on A and B, A pins B from the (mock) registry →
      warning present, lock still written (shadowing behavior itself unchanged).

**Phase 4: file-set consistency (agent latitude — may be deferred)** (~0.25 day)
- [ ] `copyPackageContents`/`shouldStageFile` and `CreateTarball`'s walk predicate are
      near-duplicates that have already drifted (tarball ships `CHANGELOG.md`, smoke
      staging does not; staging ships the lock, tarball does not — intentional). Extract
      one shared published-file-set predicate with the lock as the smoke-only addition, or
      document the divergence in both comments. Deciding *fix vs. document* is implementer
      latitude.

### Files to Modify/Create

**New files:**
- `internal/pkg/publish_rewrite.go` — `RewritePathDeps` (~120 LOC, mostly moved from `cmd/ailang/pkg_publish.go`)
- `internal/pkg/publish_rewrite_test.go` (~100 LOC)

**Modified files:**
- `internal/pkg/publish_validator.go` — staged re-lock in `RunSmokeInTempDir` (+~40 LOC)
- `cmd/ailang/pkg_publish.go` — rewrite logic moves out; call site stays (−~70/+5 LOC)
- `cmd/ailang/pkg_quality.go` — `runAttestedChecks` Notes excerpt (+~15 LOC)
- `internal/pkg/quality.go` — PUB015 message (+~3 LOC)
- `cmd/ailang/pkg_commands.go` — shadowing warning (+~30 LOC)
- `internal/pkg/publish_validator_test.go` — path-dep smoke tests (+~120 LOC)

## Examples

### The reported case, before and after

**Before (v0.54.5, reproduced):**
```
$ ailang publish --dry-run        # in sunholo/social_dynamics-consuming package
  → Rewrote dep sunholo/social_dynamics: path "../social-dynamics" → registry 0.3.0
  ✗ PUB015 _smoke.ail failed
Error: publish blocked by 1 quality gate(s) — see PUB codes above
$ ailang pkg quality --json . | jq .smoke
{ "source": "attested", "present": true, "passed": false, "seconds": 0.033 }
```

**After:**
```
$ ailang publish --dry-run
  → Rewrote dep sunholo/social_dynamics: path "../social-dynamics" → registry 0.3.0
  smoke:     [attested] present true passed true (0.4s)   # resolved from the registry
$ ailang pkg quality --json . | jq .smoke   # when a smoke genuinely fails
{
  "source": "attested", "present": true, "passed": false,
  "notes": "exit 1: Error: module loading error: ... "
}
  ✗ PUB015 _smoke.ail failed: exit 1: Error: module loading error: ...
```

### The shadowing warning

```
$ ailang lock                    # consumer with path deps on A and B; A pins B@0.3.0
✓ Generated ailang.lock (2 packages)
  → sunholo/a@0.2.0 (path: ../a)
  → sunholo/b@0.3.0 (registry)
⚠ dependency sunholo/b is declared as a path dep but was resolved from the registry
  (already pinned via sunholo/a); your local sunholo/b source is NOT in this lock
```

## Success Criteria

- [ ] Two-package repro (path dep + `_smoke.ail` importing the dep) passes
      `RunSmokeInTempDir` against a mock registry; the staged dir's lock entries are all
      `source: "registry"` (acceptance test, Phase 1)
- [ ] Same repro: `ailang publish --dry-run` completes with smoke passed (acceptance:
      run the repro end-to-end with a local `AILANG_REGISTRY` mirror)
- [ ] Failing smoke → PUB015 message contains the smoke's first error line, in human
      output and in `--json` via `smoke.notes` (acceptance test, Phase 2)
- [ ] Unpublished path dep → PUB015 names `failed to download <dep>@<version>`, not
      `package directory not found` (acceptance test, Phase 1)
- [ ] Shadowed path dep → warning on stderr, exit code unchanged (acceptance test, Phase 3)
- [ ] Existing suite green: `go test ./internal/pkg/... ./cmd/ailang/...` (incl.
      `TestRunSmokeInTempDir_Pass/_Fail/_Timeout/_Isolation`), `make check-boundaries`
      (`internal/pkg` must not import `cmd/`)
- [ ] CHANGELOG entry (user-visible: path-dep packages can publish; PUB015 now carries output)

## Testing Strategy

**Unit tests:**
- `RewritePathDeps`: whitespace-variant manifest forms (moved cases from the current
  publish flow), missing dep manifest → error, idempotence on already-rewritten manifest.
- Shadowing comparison logic: root path dep vs resolved registry entry.

**Integration tests:**
- `httptest` registry (existing pattern, `resolver_test.go:282`): publish B to the mock,
  then `RunSmokeInTempDir` on A; assert pass + staged lock content. Plus the
  not-published-at-version failure case.
- Failing-smoke output propagation through `measurePackageQuality` → `BuildQualityReport`.

**Manual testing:**
- The sunholo monorepo case end-to-end: `crew_core` (or a scratch package) with the path
  dep restored to `sunholo/social_dynamics = { path = "../social_dynamics" }` →
  `ailang publish --dry-run` and `ailang pkg quality --strict`.

## Verification Log

Every load-bearing claim above, checked against code or a live binary (v0.54.5, commit
296231d — the reported build):

| # | Claim | Evidence |
|---|-------|----------|
| V1 | The staged smoke copies the developer's `ailang.lock` unchanged | `internal/pkg/publish_validator.go` `shouldStageFile`: `case rel == LockFileName: return true` with comment "the smoke needs it to resolve deps" |
| V2 | Path lock entries resolve relative to the lock's directory | `internal/pkg/loader.go` `packageDir` case `"path"`: `filepath.Join(pl.rootDir, dir)` → `<tmp>/../social-dynamics`; empirically reproduced: `package directory not found: /tmp/b` |
| V3 | `publish` rewrites the manifest but not the lock | `cmd/ailang/pkg_publish.go` `rewritePathDepsForPublish` writes only `tomlPath`; no lock touch anywhere in the flow |
| V4 | The smoke passes in the package dir and fails when staged | Reproduced (transcript in Problem Statement): `ailang run --entry main _smoke.ail` → `hello from b`; staged copy → `package directory not found: /tmp/b` |
| V5 | `pkg quality --strict` reports PUB015 with no output; `--json` has no output field | Reproduced: gate `{'code': 'PUB015', 'msg': '_smoke.ail failed'}`, smoke section `{present, passed:false, seconds}` — `SmokeSection` (quality.go) has no Output/Error field; `runAttestedChecks` discards `res.Output` |
| V6 | Re-locking the staged dir against the rewritten manifest makes the smoke pass | Reporter-verified ("passes in the staged dir after switching the dep to 0.3.0 and running ailang lock"); mechanism re-verified by reading `ResolveLock`/`ResolveDependencies` (registry resolution path) |
| V7 | `ailang lock` silently ignores a root path dep shadowed by a transitive registry pin | Code-read of `internal/pkg/resolver.go`: root deps iterated sorted; shadowed branch hits `visited[name]`→early return, then `resolvedSet` equal-version check → path entry silently skipped; `directDeps` only records *version* deps, so `DirectVersion` is empty for the path case |
| V8 | No existing test covers path-dep smoke staging | `internal/pkg/publish_validator_test.go`: all 6 tests use the self-contained `writeMinimalPackage` fixture; no test creates a dependency or an `ailang.lock` |
| V9 | No new error code needed (PUB015 reused; no code allocation) | `grep -rn "PUB015"`: allocated and live in `quality.go` (this doc reuses it); the lock warning follows the code-less `warnSilentRatchet` stderr pattern (`pkg_commands.go:597`) |
| V10 | No planned/implemented design doc covers this (duplicate gate) | create-script auto-search (SimHash + neural) on "pkg smoke staged lock": no matches in either directory; manual review found the *related-but-distinct* docs listed below |
| V11 | Three parallel path-dep→registry rewrite sites exist (systemic pattern) | `cmd/ailang/pkg_publish.go` (dep's **local** version), `cmd/registry-validator/validate.go` (registry **latest**, string-replace), `internal/pkg/resolver.go` (registry latest for transitive path deps) — noted as future work; this doc unifies the client side only |
| V12 | Staged file set and tarball file set have drifted (CHANGELOG.md) | `internal/pkg/tarball.go` includes `rel == ChangelogFile`; `shouldStageFile` does not |

Language-construct verification gate: not applicable — no AILANG-language claims are made;
all verification above is code-read or live-repro against the reported binary.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact Notes excerpt format (first line vs first+last lines, 1–2 KiB bound) — agent may
  choose, provided it is bounded and includes the first error line.
- Whether `lock --check` also reports shadowing as drift — agent may choose (default: yes,
  as informational output, not exit-code-affecting).
- Phase 4 file-set unification: extract a shared predicate vs. document the divergence —
  agent may choose based on how far the refactor ripples.
- Whether the staged re-lock uses `AILANG_QUIET_WARNINGS` and what generator string the
  staged lock records (it is discarded with the temp dir; suggest
  `"ailang publish smoke stage"` for debuggability).
- Whether `pkgPublishCommand` should stop rewriting the developer's cwd `ailang.toml` in
  favor of staging-then-tarballing (would remove the save/restore dance at
  `pkg_publish.go:~70`); out of scope here, noted in Future Work.

## Non-Goals

- **Changing resolver precedence** (root path dep beats transitive registry pin) — warning
  only; a precedence change alters every existing monorepo lock and needs its own doc.
- **Fixing the registry validator's rewrite-to-latest divergence** (V11) — separate hazard,
  separate doc if it bites.
- **Shipping `ailang.lock` in the tarball** — the lock stays developer/consumer state;
  only the staged smoke view changes.
- **Vendoring path deps into the tarball** — publish must reference registry versions
  (existing invariant, unchanged).
- **Reverting the `sunholo/crew_core` registry pin** — possible follow-up after this
  ships, but it is a package-repo change, not part of this feature.

## Timeline

**Week 1** (~2 days):
- Phase 1 (staged published view) + tests — day 1
- Phases 2–3 (output surfacing, shadowing warning) + tests — day 2
- Phase 4 + CHANGELOG + manual monorepo verification — remainder of day 2

**Total: ~2 days** (estimate doubled from the naive 1 day, per the 2× rule)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Smoke staging now needs the registry (network or cache) where it used to be offline | Med | Same requirement `ailang lock` already imposes on consumers; `AILANG_REGISTRY` mirrors and the local package cache cover offline CI; deps cached during the author's own `ailang lock` are reused |
| Path dep not yet published at the local version now fails PUB015 with a *different* error | Low | It already fails PUB015 today with a strictly worse message; changelog calls the new error out as intentional |
| Re-lock adds wall time to `publish`/`quality` (registry fetches) | Low | Fetch happens outside the smoke's 30s timeout, only for uncached versions; cached case is one hash pass |
| `ResolveLock` in-process inside `RunSmokeInTempDir` mutates global resolver/loader state | Med | Verify `internal/pipeline` loader globals are not set by `ResolveLock` (it is pure resolution); if they are, shell out to `ailang lock` in the staged dir instead — implementer verifies with `make check-boundaries` + full test run |
| Publish flow double-rewrite (cwd + staged) diverges over time | Low | Both call the one `internal/pkg.RewritePathDeps`; the staged call is idempotent on an already-rewritten manifest |

## Related Documents

**Implemented (may inform design):**
- [m-ext-portability-gate](../../implemented/v0_18_11/m-ext-portability-gate.md) — created `RunSmokeInTempDir` ("smoke test in temp-dir with no workdir state"); this doc fixes the staging to honor that charter for path deps
- [m-pkg-lock-portability](../../implemented/v0_10_0/m-pkg-lock-portability.md) — made lock paths relative to the lock's dir; that relative form is exactly what breaks when the lock is copied to a temp dir
- [m-pkg-quality-ladder](../../implemented/v0_40_0/m-pkg-quality-ladder-sprint-plan.md) — the PUB-report machinery this doc extends (PUB015 output surfacing)
- [m-pkg-transitive-lock-fix](../../implemented/v0_10_0/m-pkg-transitive-lock-fix.md) — resolver/lock history context

**Planned (check for overlap):**
- None (auto-search + manual review; see Verification Log V10)

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- User report: v0.54.5 (296231d), "publish/quality smoke stage copies path-dep ailang.lock"
- In-tree workaround: `sunholo/crew_core` pins `sunholo/social_dynamics = "0.3.0"`

## Future Work

- Unify all three path-dep→registry rewrite sites (V11), pinning the *declared* version
  consistently instead of registry-latest in the validator.
- Stop mutating the developer's `ailang.toml` during publish (stage-then-tarball), removing
  the save/restore dance.
- Resolver precedence for root path deps vs transitive registry pins (post-warning decision).

---

**Document created**: 2026-10-12
**Last updated**: 2026-10-12
