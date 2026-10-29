# M-PKG-SMOKE-STAGED-LOCK: stage the *published* view (manifest + lock) for the temp-dir smoke

**Status**: Implemented and accepted — independent round2 PASS95/100; merged #2013; v0.56.0 UNRELEASED
**Target**: v0.56.0 (unreleased); original target v0.54.7 retained as history
**Priority**: P1 (a smoke importing a dependency through a portable relative path lock fails isolated staging, blocking the documented monorepo publishing workflow)
**Estimated**: Historical ~1.5–2 days (~450 LOC incl. tests); re-estimate only during approved sprint planning
**Dependencies**: Preserve landed package-authority D2 (#1720/#1964) and locked asset snapshots (#1849)
**Companion plan**: [approved v0.56.0 sprint plan](m-pkg-smoke-staged-lock-sprint-plan.md) — execution approved at2026-10-29 12:23UTC
**Author**: design-doc-creator session (task-98f9f628), from the v0.54.5 user report (macOS arm64, commit 296231d)

## Accepted implementation (2026-10-29)

Independent round2 **PASS95/100** evaluated exact
`165f87a12ceb0e20c04216c472004c11685e417b`; actual final CI completed
27checks (20SUCCESS/7SKIPPED, no failed/cancelled/pending). Root accepted and
normally merged [PR2013](https://github.com/sunholo-data/ailang/pull/2013) as
`96d69f05b6e84bd6666e8fa91ff948f803b67d92` at20:53:24UTC; issue1773
closed20:53:25UTC. This is implemented, accepted and **unreleased**, without a tag
or version bump. The [plan](m-pkg-smoke-staged-lock-sprint-plan.md),
[validation map](m-pkg-smoke-staged-lock-validation.md) and
[evaluation evidence](evidence/m-pkg-smoke-staged-lock/README.md) are archived together.

Round1 **FAIL75** at0853 and both actual failed-run identities remain preserved.
The later accepted dependency/new-head CI does not turn those failures into passes.
The design's earlier approval/execute and pending-gate text below is historical
to its recorded checkpoint; this section states the final disposition.

---

## Current-dev Refresh and Ownership (2026-10-29)

Reviewed against actual dev `80793dbcf42423027be4bab1a0088da54e2323d8`.
The attended user explicitly approved this refreshed design, relayed by root on
2026-10-29 (approval recorded at 11:57:17 UTC). The approval covers existing D2
compatibility, staged asset consistency and diagnostic propagation as preservation
of approved policies; no new authority or precedence choice was introduced.
The user later explicitly approved execution of companion plan commit0c3e8f406
at2026-10-29 12:23UTC, relayed by root. Live publication and coordinator
approval/actions remain unauthorized. This record does not revive the historical
coordinator task or infer execution approval from the old merged design PR.

The original design was delivered by task `task-98f9f628` through merged
[design-only PR #1775](https://github.com/sunholo-data/ailang/pull/1775), commit
`4c89ce659e20f2458516546892142445ba352399` (2026-10-12). The current production
record for that task says **failed**, despite the merged artifact; its PR body's
claimed planner trigger is not evidence of a current implementation owner.
Live issue #1773, current open-PR title/body searches, and all 16 currently active
production tasks (pending/queued/running/pending_approval) show no staging owner.
A separate latest-1000-task historical page finds that original design task;
this is a bounded search, not a claim that no task ever existed. The generic
#1773 reference in M-AILANG-LANE-EXECUTOR-SHADOW concerns coordinator execution,
not package-staging implementation.

The original high-impact choices remain: smoke tests the **published registry
view**, and root-path shadowing is **warning-only**. Passing a validated previous
git pin to the existing resolver preserves already-approved D2; it introduces no
new resolution or authority policy. Bounded staging/error diagnostics complete
the original PUB015 goal. Any authority bypass, precedence change, live
publication, validator rewrite-to-latest fix or expanded published file set would
be new scope and is not proposed.

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

**Net Score: +5** → **Decision: Move forward**

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
`RunSmokeInTempDir`): when the smoke imports the dependency through the generated
relative path lock, staging raises PUB015 without its diagnostic and blocks publish.
A smoke that never uses the dependency, or a legacy absolute-path lock still able
to reach author files, is not evidence of a correct published-view stage. The in-tree workaround is to pin the
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

### Actual-dev offline verification (2026-10-29)

The defect remains present on dev 80793, including the reporter's secondary
shadowing case. The current embedded teaching prompt v0.16.11 was loaded before
writing scratch AILANG. A two-package fixture uses different source and asset
markers for same-version local B versus provisioned registry B:

| Instrument | Observed result |
|------------|-----------------|
| Local A path-lock + smoke | exit 0, `LOCAL_SOURCE` and `LOCAL_SOURCE_ASSET` |
| Actual `pkg quality --json` on A | compile=true; exit 2; PUB015 is the sole gate; smoke passed=false and no notes |
| Isolated stage with copied relative path lock | exit 1, dependency directory `<stage>/../b` missing |
| Rewritten staged manifest + regenerated registry lock | exit 0; lock source=registry; smoke prints `REGISTRY_SOURCE` and `REGISTRY_SOURCE_ASSET` |
| Root path B shadowed by A's same-version registry B | exit 0; B remains registry-sourced; no stderr warning |
| `pkg quality --no-run` | exit 0; smoke present=true/not_run=true; no PUB015 failure |

`AILANG_PACKAGE_ROOT` points to scratch provisioned bytes. `AILANG_REGISTRY`
points only to an unavailable loopback endpoint; no external registry or live
publish is involved. Its unrelated export-overlap badge records that unavailable
endpoint; the sole quality gate is the reproduced smoke defect. Session evidence
is `1773-readonly-repro/{results.json,extra-results.json,*.log}` under
`/tmp/ailang-backlog-20261029`, with the harness beside it. This is a source-read
and scratch verification, not an implemented product fix.

## Goals

**Primary Goal:** the temp-dir smoke runs against the *published* view — the rewritten
manifest **and** a lock regenerated from it — so path-dep packages publish and PUB015
carries the smoke's output when it fails.

**Success Metrics:**
1. The repro above (`a` with path dep `../b` + `_smoke.ail`) publishes (dry-run) and reports
   `smoke: present true passed true`, with the smoke resolving B from the registry.
2. A *failing* smoke reports PUB015 with the smoke's captured output (human and `--json`).
3. `ailang lock` prints a warning when a root path dep was shadowed by a non-path resolution, naming its actual source; resolution and exit codes stay unchanged.
4. `sunholo/crew_core`-style registry pins are no longer required to publish from a monorepo.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| The staged smoke resolves deps from the **registry** (published view), not from staged sibling copies of path deps | Defines what the portability gate *means*: it must test what a consumer installs | human | design | high |
| Re-lock the staged copy (regenerate the lock against the rewritten staged manifest) rather than rewriting path entries inside the developer's lock | Lock entries carry content hashes that must match the registry copy, not local source; re-lock reuses the current code path (`ResolveLockFrom` with validated previous git metadata; regenerated registry entries) | agent | design | med |
| `pkg quality`'s smoke uses the same published view as `publish` | One staging semantics; today `pkg quality` path-dep packages fail PUB015 anyway (verified), so this is a message improvement, not a behavior regression | agent | design | med |
| PUB015 surfaces the smoke output via the existing `SmokeSection.Notes` field (bounded excerpt) instead of a new JSON field | The smoke section is uploaded as the attested block and banked by the validator; a bounded reuse keeps the schema stable | agent | design | low |
| The registry-shadowing fix is a **warning** in `ailang lock`, not a resolution-precedence change | Changing precedence (root path wins over transitive registry pin) alters lock output for existing monorepos; warning first, semantics later | human | design | med |

### Design Freeze

The selected alternatives below are retained from the original design. They were
ratified with the 2026-10-29 compatibility refresh. Design approval authorizes
planning only; it does not authorize execution. No human semantic decision
remains open inside this approved scope.

Approved rulings (retain them during planning and execution):

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
3. **Re-lock the staged dir without losing D2 provenance**: read and validate any
   copied previous `ailang.lock` **before removing it**; only a genuinely missing
   lock is treated as no previous lock. Regenerate via
   `ResolveLockFrom(stagedManifest, stagedDir, generator, validatedPrevious)` and
   save the returned new lock before launching smoke. The previous lock supplies
   only the resolver's existing unchanged-git carry-forward under confinement or
   `AILANG_PACKAGE_ROOT`; it does not supply copied path entries or registry
   hashes. Path-to-registry dependencies are freshly resolved from the rewritten
   manifest, as `ailang lock` does. Malformed prior locks and changed/tampered git
   pins fail closed. Missing registry downloads still honor `AILANG_REGISTRY`
   through the existing authority chokepoint; no new HTTP/git/home-write path or
   fallback is introduced.
4. Run the smoke (as today).

Separately: **surface the output** (PUB015) and **warn on shadowing** (`ailang lock`).

### Options Considered

**A. Stage the published view: rewrite staged manifest + re-lock the staged dir (CHOSEN).**
The smoke tests exactly what a consumer gets. If the path dep's local version is not
published and downloading is permitted, re-lock fails with
`failed to download <dep>@<version> from registry`. Under an exclusive package root
or confinement, the existing missing-provision/operator-authority refusal is the
correct result instead; these contexts must not bypass their authority controls.
Each is the correct pre-publish signal (the tarball would be uninstallable), and for
`pkg quality` it replaces today's cryptic `package directory not found: <tmp>/../b` with
the honest reason. If local source has drifted from the published version, the smoke tests
the *published* content while the compile check in the real dir still tests local source —
each instrument measuring what it claims.

**B. Rewrite the path entries inside the staged lock (REPORTER'S ALTERNATIVE, REJECTED).**
Converting `source:"path"` entries to `source:"registry"` in place keeps the local content
hash, which (a) does not match the registry copy's hash when source has drifted →
hash-mismatch warnings, wrong view; and (b) still needs a download when the exact version
isn't in the local cache — converging on A's fetch while doing it worse. A is the same work
with correct hashes (`ResolveLockFrom` computes them from the registry copy).

**C. Stage path-dep siblings into the temp tree so relative lock paths resolve (REJECTED).**
The staged manifest would keep path deps, so the smoke would pass on local sources while
the published tarball references registry versions — the gate would verify a deployment
that does not exist. Inverts the portability gate's purpose (M-EXT-PORTABILITY-GATE was
created precisely because smoke-in-authors-workdir hid consumer breakage).

### Architecture

**Components:**

1. **`internal/pkg/publish_rewrite.go` (new)** — a silent
   `RewritePathDeps(targetDir, resolveFromDir string) ([]PathDepRewrite, error)`
   helper (structured Go rewrite facts; empty slice when unchanged): reads `<targetDir>/ailang.toml`, and for each `[dependencies]` entry with `path`,
   loads the dep manifest at `resolveFromDir/<path>`, rewrites the entry to
   `"<name>" = "<depVersion>"` (the existing whitespace-tolerant regex from
   `rewritePathDepsForPublish`), writes the staged manifest back. `cmd/ailang/pkg_publish.go`
   keeps calling the same logic for its cwd rewrite, converting the result to its
   existing bool and rendering publish progress there. The shared helper must not
   print: quality staging calls it while producing JSON. This return-shape
   refinement preserves existing output/schema rather than adding a policy.
   Tarball creation still needs the rewritten manifest on disk in cwd — unchanged.

2. **`RunSmokeInTempDir`** — after `copyPackageContents`, assemble one published
   staging view. Call `RewritePathDeps(tmpDir, packageDir)`, validate the copied
   previous lock, remove it, load the rewritten staged manifest, invoke
   `ResolveLockFrom(..., validatedPrevious)`, then save only the regenerated lock
   in `tmpDir`. A previous lock's path entries must never be the execution lock.
   `ResolveDependenciesFrom` currently consults previous metadata only for git
   resolution, so unchanged D2 pins remain usable without git while registry
   content/interface hashes are recomputed. Re-lock failure returns a failed
   `SmokeResult` with its cause; staging infrastructure errors may remain `rErr`,
   provided the caller reports them as failing PUB015 with Notes. Smoke must
   never run with the discarded lock or without the required new lock.

   Existing #1849 discovery derives per-compilation asset targets from this
   staged manifest/lock: self assets point to the stage, dependency module and
   asset bytes to the same resolved registry directory. Retain missing-target
   tombstones, malformed-file errors and exclusive `AILANG_PACKAGE_ROOT`; do not
   add an independent asset resolver, reuse author directories or choose a newer
   cached fallback. No effect/runtime semantics change is needed.

3. **Output surfacing** — `runAttestedChecks` currently discards **both** `rErr`
   and `res.Output`; it must report either kind of failure in existing
   `SmokeSection.Notes`. Stage/rewrite/lock failures stay failed smoke/PUB015,
   without a new error code, schema field or exit-policy change. Notes is bounded
   to **2048 bytes**, preserving the first meaningful error diagnostic plus a
   bounded tail/context and a timeout marker when appropriate. Preserve that
   diagnostic **before** the existing `truncateOutput` tail-only 8 KiB truncation
   can erase it; merely taking the first line of an already-truncated string
   cannot satisfy the output goal. Long/Unicode output must remain within the
   byte bound and serializable; if the first diagnostic alone exceeds the cap,
   retain its bounded prefix with a truncation marker. `BuildQualityReport` uses the same bounded first
   diagnostic in the PUB015 finding, e.g.
   `"_smoke.ail failed: " + firstDiagnostic(s.Notes)`, so human and JSON reports
   agree. `SmokeSection.NotRun`, absent-smoke behavior and `--no-run` remain
   unchanged. The companion plan M2 fixes concrete first-diagnostic selection,
   prefix/trace filtering and byte-budget tests before execution approval.
   Staging resolution remains outside the existing smoke subprocess timeout;
   timeout semantics are not silently broadened.

4. **Lock shadowing warning** — `pkgLockCommand` (`cmd/ailang/pkg_commands.go`), after
   `ResolveDependenciesFrom`: for each root manifest dep with `Path != ""`, if the resolved
   entry's `Source != "path"`, print a yellow stderr warning:
   `dependency <name> is declared as a path dep but was resolved from <actual source> (already pinned via a transitive dependency); your local <name> source is NOT in this lock`.
   Use the resolved source label: do not call a git resolution "registry".
   Do not change resolver visitation, precedence, versions, ordering or exit codes.
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
5. **Intentional change**: `pkg quality` / `publish` on a path dependency not
   available as the published exact version fails PUB015 with the actual
   registry/provisioning/authority cause, rather than a copied-path staging
   failure. An operator-enabled download error can say `failed to download`;
   confined/root-exclusive errors retain their current remedy. Call this out in
   the changelog without promising one universal diagnostic string.
6. **Already-approved git confinement**: valid unchanged git pins carry forward
   from validated previous metadata without git or home writes. Missing or
   changed URL/tag/rev/subdir/hash/interface still refuses. Existing fixtures:
   `TestResolveGitConfined_CarryForwardMatrix`, `TestResolveGitConfined_RevPin`,
   `TestGitPackageRoot` in `internal/pkg/resolver_git_test.go` (bodies reviewed).
7. **Locked asset snapshots**: keep `TestPackageLoaderAssetTargets`,
   `TestAssetTargetsForDiscovery`, `TestPipelineLockedAssetTargets` and
   `TestPkgAssetPath_LockedTargets` behavior: staged self/root targets, exact
   dependency version, missing locked-target failure and no newer-cache fallback.
   No parser, typechecker or runtime surface is extended.

### Implementation Plan

These are retained design phases, not execution authorization. Design approval
now authorizes companion planning; implementation still requires a separate
explicit execute instruction after plan review.

**Phase 1: staged published view** (~0.5 day)
- [x] Extract `RewritePathDeps` to `internal/pkg/publish_rewrite.go` (generalized over
      target dir / resolution dir); `pkg_publish.go` calls it (behavior identical).
- [x] `RunSmokeInTempDir`: staged rewrite; validate previous lock before removal;
      `ResolveLockFrom` with existing git carry-forward; save regenerated registry
      lock; failing result/Notes on staging or resolution error, with no fallback.
- [x] Test: the two-package repro (path dep + `_smoke.ail` importing `pkg/<dep>/...`)
      against an `httptest` registry (pattern: `internal/pkg/resolver_test.go:282`,
      `AILANG_REGISTRY` env) — smoke passes, lock in temp dir is registry-sourced.
- [x] Test: unavailable exact registry version → bounded PUB015 cause; permitted
      download and confined/root-exclusive errors retain their respective remedies.
- [x] Test: local and registry same-version source **and asset** bytes differ;
      stage uses registry markers, root assets remain staged, author inputs unchanged.
- [x] Test: valid confined/root git pin passes without git/HTTP/home writes;
      absent/changed/tampered/malformed prior metadata refuses as today.

**Phase 2: PUB015 output surfacing** (~0.25 day)
- [x] `runAttestedChecks`: bound Notes for `rErr` and failed process output;
      retain first diagnostic before lossy truncation, plus timeout marker.
- [x] `quality.go`: PUB015 message includes the same bounded first diagnostic.
- [x] Test: failing smoke (existing `TestRunSmokeInTempDir_Fail` fixture), staging
      infrastructure error, timeout, leading noise and >8 KiB output → human/JSON
      Notes retains first error within 2048 bytes; `--no-run` remains not-run.

**Phase 3: lock shadowing warning** (~0.25 day)
- [x] `pkgLockCommand` post-resolution comparison; stderr warning.
- [x] Test: consumer with path deps on A and B, A pins B from the (mock) registry →
      warning present, lock still written (shadowing behavior itself unchanged).

**Phase 4: file-set consistency (agent latitude — may be deferred)** (~0.25 day)
- [x] `copyPackageContents`/`shouldStageFile` and `CreateTarball`'s walk predicate are
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

- [x] Two-package repro (path dep + `_smoke.ail` importing the dep) passes
      `RunSmokeInTempDir` against a mock registry; the staged dir's lock entries are all
      `source: "registry"` (acceptance test, Phase 1)
- [x] Same repro: `ailang publish --dry-run` completes with smoke passed (acceptance:
      run the repro end-to-end with a local `AILANG_REGISTRY` mirror)
- [x] Failing smoke → PUB015 message contains the smoke's first error line, in human
      output and in `--json` via `smoke.notes` (acceptance test, Phase 2)
- [x] Unavailable published path-dep version → PUB015 names the real download,
      provisioning or authority cause and remedy, not a copied staging path;
      no confinement/package-root bypass (acceptance test, Phase 1)
- [x] Valid previous git pins carry forward in confined and PACKAGE_ROOT cases;
      mismatched/missing/tampered/malformed metadata fails closed without git,
      network or home writes; rewritten path entries are not carried forward
- [x] Distinct local/registry module and asset markers prove the published view;
      staged root assets and #1849 missing-target/no-newer-fallback controls pass
- [x] Stage `rErr`, timeout and long/Unicode output preserve the first meaningful
      error within 2048-byte Notes, including after >8 KiB output; no schema change
- [x] Shadowed path dep → warning on stderr, exit code unchanged (acceptance test, Phase 3)
- [x] Existing suite green: `go test ./internal/pkg/... ./cmd/ailang/...` (incl.
      `TestRunSmokeInTempDir_Pass/_Fail/_Timeout/_Isolation`), `make check-boundaries`
      (`internal/pkg` must not import `cmd/`)
- [x] CHANGELOG entry (user-visible: path-dep packages can publish; PUB015 now carries output)

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

### Historical verification (2026-10-12)

The original observations below were checked against code or the reported
v0.54.5 commit 296231d. They are historical; the current-dev refresh rows follow.
Line numbers and "no overlap" claims describe that earlier snapshot:

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

### Current-dev refresh verification (2026-10-29)

| ID | Claim | Evidence on actual dev 80793 |
|----|-------|---------------------------|
| C1 | Copied path-lock bug and missing PUB015 Notes remain | Offline A/B fixture: local source+asset pass; actual quality compile=true/PUB015-only gate with no Notes; copied stage fails missing sibling |
| C2 | Regenerated registry lock selects correct source and assets | Rewritten stage + `ailang lock`, using preprovisioned PACKAGE_ROOT, prints distinct registry module and asset markers |
| C3 | Nil previous metadata breaks confined/root git carry-forward | `lockcheck.go` ResolveLock delegates nil; `resolver_git.go` provisionedGitDep explicitly refuses nil; existing CarryForwardMatrix and GitPackageRoot tests pass (10.555s), with valid pin/no git and nil/mismatch refusal controls |
| C4 | Existing resolver can preserve authority without global loader mutation | ResolveLockFrom → ResolveDependenciesFrom uses per-call graph/client state; previous metadata consulted for git only; HTTP/git/HOME I/O stays behind authority.go; no pipeline loader-global writes |
| C5 | Infrastructure errors are discarded too; first error can be truncated | `pkg_quality.go` runAttestedChecks only handles rErr==nil; `publish_validator.go` truncateOutput keeps only final 8 KiB; `quality.go` PUB015 is a fixed bare message |
| C6 | Existing snapshot behavior fits staged manifest+new lock | pipeline/package_assets.go → PackageLoader.AssetTarget; self/root retained, missing target tombstone retained, PACKAGE_ROOT exclusive; actual fixture proves exact dependency source+asset bytes |
| C7 | Shadowing remains silent and no-run semantics must remain | Same-version root path B/transitive registry B fixture: B remains registry, exit 0/stderr empty. `quality --no-run`: present/not_run true, exit 0 |
| C8 | Original task/artifact exists, no current staging owner found | Original task 98f9 failed; PR 1775 merged 4c89. Current open PR search and 16 active prod tasks have no match; latest 1000 historic rows bounded as described above |
| C9 | No test in current smoke suite covers dependent staging | Read all six RunSmokeInTempDir tests; writeMinimalPackage is self-contained with no dependency or lock. Authority and asset fixtures named above exist and their assertion bodies were reviewed |
| C10 | PUB015/Notes already allocated; no new code/schema needed | internal/pkg/quality.go defines SmokeSection.Notes and PUB015; current quality JSON fixture confirms absence is a propagation gap, not a missing JSON field |

No new language feature or limitation is asserted. Scratch AILANG follows the
loaded v0.16.11 prompt and successfully runs in its valid local and registry
views; all new mechanism claims are tied to source reads or recorded controls.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact Notes presentation (first diagnostic plus optional final context) — agent
  may choose within the 2048-byte bound, preserving the first meaningful error
  before lossy truncation and retaining timeout context. This is not latitude to
  discard staging `rErr` or treat a failed assembly as `NotRun`.
- `lock --check` shadowing output was deferred by the approved companion plan;
  the implemented warning follows successful normal lock writes only.
- Phase 4 file-set unification: document the current divergence by default, or
  extract a predicate only while preserving the existing tarball file set and
  intentional staging differences. No additional published file class is approved.
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
| Smoke staging may need resolution authority even when author smoke was offline | Med | Exact provisioned registry entries can resolve offline. Missing entries and legacy registry transitive path/index lookups retain canonical refusal under confinement; do not bypass authority or promise all cached graphs are network-free. Carry unchanged git pins via validated previous metadata. |
| Path dep not yet published at the local version now fails PUB015 with a *different* error | Low | It already fails PUB015 today with a strictly worse message; changelog calls the new error out as intentional |
| Re-lock adds wall time to `publish`/`quality` (registry fetches) | Low | Fetch happens outside the smoke's 30s timeout, for missing registry metadata/bytes as permitted by authority; fully provisioned exact registry manifests avoid downloads |
| Staging accidentally discards the previous git pin or uses unguarded I/O | Med | Current ResolveLockFrom has per-call graph/client state and no pipeline loader-global mutation (C4); reuse it in-process with validated previous metadata. Missing/mismatched pins fail closed. Existing authority/boundary tests enforce HTTP/git/HOME chokepoints; do not shell out as a bypass. |
| Publish flow double-rewrite (cwd + staged) diverges over time | Low | Both call the one `internal/pkg.RewritePathDeps`; the staged call is idempotent on an already-rewritten manifest |

## Related Documents

**Implemented (may inform design):**
- [m-ext-portability-gate](../../implemented/v0_18_11/m-ext-portability-gate.md) — created `RunSmokeInTempDir` ("smoke test in temp-dir with no workdir state"); this doc fixes the staging to honor that charter for path deps
- [m-pkg-lock-portability](../../implemented/v0_10_0/m-pkg-lock-portability.md) — made lock paths relative to the lock's dir; that relative form is exactly what breaks when the lock is copied to a temp dir
- [m-pkg-quality-ladder](../../implemented/v0_40_0/m-pkg-quality-ladder-sprint-plan.md) — the PUB-report machinery this doc extends (PUB015 output surfacing)
- [m-pkg-transitive-lock-fix](../../implemented/v0_10_0/m-pkg-transitive-lock-fix.md) — resolver/lock history context

**Current compatibility references (landed code):**
- [m-pkg-authority-chokepoint](../../planned/v0_54_19/m-pkg-authority-chokepoint.md) — #1964 landed the single authority boundary; preserve D2 previous-pin carry-forward, exclusive package root and canonical refusal. This design adds staging, not authority.
- [m-pkg-asset-lock-resolution](m-pkg-asset-lock-resolution.md) — #1849 landed per-compilation/root/locked dependency asset targets and missing-target tombstones. Staging supplies a valid new lock to that existing mechanism.

**Overlap search:** the original V10 "None" was historical. The current ownership
search is bounded in Current-dev Refresh above; no competing staging implementation
was found. Preserve this same document rather than create another design carrier.

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
**Last updated**: 2026-10-29 (approved M1–M4 implemented; local gates passed; independent evaluation/PR CI pending)

## Executor completion (2026-10-29)

All four approved milestones are implemented. Changed package/core and structural
gates passed on immutable source `37c957b3a`; subsequent actual audit integration
changed no Go/AIL source. [Validation and acceptance map](m-pkg-smoke-staged-lock-validation.md)
records red/green evidence, the strict registry-byte CLI discriminator, the
independent signal-cause correction, and pending exact PR Linux/Windows and
independent evaluation gates. Checkboxes record executor verification, not an
evaluator verdict or release. The existing document and plan remain planned
until root accepts the independent review.
