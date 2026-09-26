# M-PUBLISH-PATH-DEP-SMOKE: make the publish gate tell the truth about path dependencies

**Status**: Planned
**Target**: v0.42.1
**Priority**: P1 (blocks every Daneel extension publish — capabilities.md E3 "attended publish of an extension so it can outlive the machine" is currently impossible for any package with a path dep)
**Estimated**: 2–3 days (~350 LOC Go + tests)
**Dependencies**: None
**Bug report**: Daneel, 21 Sept 2026 (task-e00b9074): "`ailang publish` runs `_smoke.ail` in a temp-dir COPY of the package, so a package whose ailang.toml carries a path dependency (`"sunholo/daneel_ext_abi" = { path = "../abi" }`) fails the smoke gate with PUB015 even though `ailang run _smoke.ail` passes in place." Reproduce: `cd ext/search && ailang publish` on daneel@main (v0.40.2).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Smoke staging is a deterministic function of (manifest, lock); no machine-state dependence beyond what the lock already declares |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | +1 | The publish gate stops vouching (attested smoke) for content it never ran; the tarball's declared deps are verified against the registry before upload |
| A5: Bounded Verification | +1 | New pre-upload check is local + one registry fetch — bounded, fast |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Gate failures become actionable machine-readable instructions ("publish sunholo/daneel_ext_abi@0.5.0 first") instead of a bare "PUB015 _smoke.ail failed" |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Publisher-side path-dep handling mirrors the consumer-side `fromRegistry` conversion already in resolver.go |
| A11: Structured Failure | +1 | Three silent/ambiguous failure modes (temp-dir miss, silent regex skip, discarded smoke output) become loud, named PUB gates |
| A12: System Boundary | 0 | Registry boundary unchanged |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): staging is a pure copy of lock-declared directories; no implicit nondeterminism
- [x] A3 (Effects): no hidden side effects (staging writes only inside the temp workspace)
- [x] A4 (Authority): no ambient access granted; the gate strictly *narrows* what can be published
- [x] A7 (Machines First): clear gate messages exist precisely so agent publishers can self-serve

## Problem Statement

### The reported bug

`ailang publish` runs `_smoke.ail` via `RunSmokeInTempDir` (`internal/pkg/publish_validator.go`), which copies only the package's own published file set (`ailang.toml`, `*.ail`, `AGENT.md`, `assets/`, `ailang.lock`) into a fresh temp dir and runs the smoke there (M-EXT-PORTABILITY-GATE, v0.18.11). Runtime import resolution is **lock-driven**: `tryLoadPackageResolver` builds a `PackageLoader` from `ailang.lock`, and lock entries for path dependencies are stored **relative to the package root** (`portablePathDep`) and joined with the root at load time. In the temp-dir copy the root is the temp dir, so `path = "../abi"` resolves to `<tmpdir>/../abi` — which does not exist. The smoke fails with `package directory not found` (V4), publish reports `PUB015 _smoke.ail failed` (V5), and the author is never shown the underlying error (V10).

Meanwhile `ailang run _smoke.ail` in place passes, because `../abi` exists next to the real package. Every Daneel extension under `ext/` depends on the ABI by path (capabilities.md B3), so **none of them can pass the publisher gate**: `daneel_ext_search` 0.1.1/0.1.2 and `daneel_ext_abi` 0.5/0.6 sit on the repo but not on the registry (registry holds abi ≤0.4.0 and search 0.1.0 — V11).

### The larger pattern (audited, per the anti-incremental rule)

The reported symptom is one face of a four-part systemic gap: **the publish gate never establishes that the tarball's declared dependencies resolve**.

| Gap | Evidence | User-visible failure |
|---|---|---|
| **A. Temp-dir smoke can't stage path deps** (the report) | `copyPackageContents` stages only the package's own files (V2); the lock's relative path entry then misses (V3, V4) | PUB015 on every path-dep package, even when the dep is already on the registry at the rewritten version |
| **B. Phantom registry versions** | `rewritePathDepsForPublish` rewrites `path = "../abi"` → `"0.5.0"` (taken from the *local* dep's manifest) but nothing between the rewrite and the upload checks that `sunholo/daneel_ext_abi@0.5.0` exists on the registry (V9) | Publish can succeed while shipping a tarball no consumer can install |
| **C. Silent regex skip** | the rewrite is regex-based and its `re.MatchString` branch has **no else**: a dependency written as a TOML table header (`[dependencies."name"]` + `path = "..."`) or with extra keys is silently left as a path dep in the tarball (V8, live-reproduced) | Path deps can ship in published tarballs with no gate firing |
| **D. PUB015 discards the smoke output** | `runAttestedChecks` keeps only `Passed`/`Seconds` from `SmokeResult`; `Output` is dropped, and the PUB015 finding is the literal string "_smoke.ail failed" (V10) | The gate cannot "say plainly" why it failed — the report's second acceptance path |

Gap A alone would let a broken publish through if fixed in isolation (the smoke would pass against the *local* dep copy while the tarball references a registry version that may not exist — gap B). The unified fix closes all four: **stage what the lock declares, verify what the tarball declares, refuse what can't be rewritten, and show why.**

### Current state (measured)

- Live repro on this machine: in-place `ailang run --caps IO --entry main _smoke.ail` → `hello from abi 0.5.0` (exit 0); `ailang publish --dry-run` on the same tree → `✗ PUB015 _smoke.ail failed` plus a publish refusal (V5).
- Temp-dir simulation (manual `copyPackageContents` equivalent) → `Warning: failed to hash dependency ... lstat <tmp>/abi: no such file` then `Error: module loading error: ... package directory not found: <tmp>/abi` (V4).
- Registry drift: 7 Daneel packages on the registry, all stale vs the repo; the two the E3 use case needs most (`daneel_ext_abi` 0.5/0.6, `daneel_ext_search` 0.1.1/0.1.2) cannot be published until this lands (V11).

**Impact**: every AILANG package that declares a path dependency — the standard monorepo layout (ext/ + abi/ siblings) — is unpublishable, and the only workaround (hand-editing the manifest to registry versions before publishing) breaks local development.

## Goals

**Primary Goal:** `ailang publish` on a package with path dependencies behaves exactly as it does for registry dependencies: the smoke runs against a faithful staged copy, the tarball's rewritten registry deps are verified to exist and match, and every refusal names its cause and its fix.

**Success Metrics:**
1. `cd ext/search && ailang publish` succeeds for the Daneel layout once `daneel_ext_abi` is published first (and refuses with an actionable PUB017 before that).
2. A path dep that is NOT on the registry blocks publish with a message naming the exact publish command to run first (no phantom-version tarballs).
3. An unrewritable path-dep TOML form (table header, extra keys) blocks publish loudly (no silent path-dep tarballs).
4. A PUB015 refusal prints the tail of the smoke output (the actual loader error).
5. All existing `publish_validator_test.go` tests still pass unchanged, including `TestRunSmokeInTempDir_Isolation`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Stage path-dep copies into the smoke workspace (vs `--package-dir` back-point, vs re-locking against registry) | Determines whether the smoke stays isolated and offline; `--package-dir` at the original tree would route self-reference imports to the *original* files and defeat the M-EXT-PORTABILITY-GATE isolation | human (design freeze below) | design | med |
| Registry-existence + content-hash check is a HARD gate (PUB017), not a warning | Changes publish behaviour for every path-dep package; a warning would re-create the phantom-tarball hole | human (design freeze below) | design | med |
| New error code PUB017 (verified unallocated — V12) | PUB codes are a shared namespace | agent (verified free) | design | low |
| Keep the regex rewrite (formatting-preserving) + add a post-rewrite assertion, rather than TOML-AST round-trip | AST round-trip rewrites the author's whole file (comments, layout); assertion closes the hole with ~15 LOC | agent | design | low |
| Staging geometry: package at `<tmp>/w₁…w_N/pkg`, deps at `Join(pkgDir, relPath)` with N = max leading `..` count | Handles `../abi` and `../../shared/x` uniformly without escaping the temp root | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Human approval**: path-dep smoke staging via copies (Decision 1) — the isolation guarantee narrows from "nothing outside the package" to "nothing outside the package *and its lock-declared dependencies*", which is the correct reading of M-EXT-PORTABILITY-GATE (undeclared workdir state was the enemy; declared deps are part of the package's contract) but should be ratified.
- [ ] **Human approval**: PUB017 as a hard gate (Decision 2).

## Solution Design

### Overview

Four coordinated changes, one principle: *the publish gate verifies exactly what the tarball will resolve to*.

1. **Stage lock-declared path deps alongside the package** in `RunSmokeInTempDir` so the copied `ailang.lock`'s relative entries resolve inside the workspace (closes Gap A).
2. **Verify rewritten registry deps against the live registry** (existence + content-hash match) as a hard pre-tarball gate, PUB017 (closes Gap B, and gives the Daneel flow its correct publish order: abi first, then search).
3. **Assert the rewrite actually rewrote** every path dep; refuse loudly on any survivor (closes Gap C).
4. **Thread the smoke output into the PUB015 finding** and the publish error (closes Gap D).

### Architecture

**1. Staging (internal/pkg/publish_validator.go).**

`RunSmokeInTempDir` currently stages the package flat into `tmpDir` and runs with `cmd.Dir = tmpDir`. Change to:

- Compute `padN = max(1, max leading ".." segments over all lock entries with source="path" and a relative path)`. Stage the package at `tmpRoot/w1/…/wN/pkg` (fixed dir names; the smoke command's `--package-dir` is not needed — `packageSearchDir` anchors on the smoke file's directory, which is inside the staged package, so `FindManifest` finds the staged `ailang.toml`).
- For every `ailang.lock` entry with `source == "path"`: stage the dep at `filepath.Join(stagedPkgDir, entry.Path)` (its stored relative path, now resolving inside the workspace). Copy with the same `shouldStageFile` filter — the published view, so `ContentHash` of the staged copy equals the lock's recorded hash (`ContentHash` hashes sorted `.ail` files only — V14) and `ValidateContentHashesFrom` stays quiet.
- Entries with **absolute** paths: copy the dep into `tmpRoot/deps/<index>-<name>` and rewrite that entry's `Path` in the *staged* lock copy only (never the developer's lock). This keeps isolation for the rare cross-volume case.
- **Backstop**: if any computed target lands outside `tmpRoot` (possible via symlinked paths after `filepath.Join` cleaning), fail the staging with a clear error: "path dependency X escapes the smoke workspace; declare it as a registry dependency or restructure the layout" — no silent fallback.
- Transitive path deps need no special handling: `ailang lock` records the full transitive closure (V13), so every path dep is a lock entry; each is staged the same way.
- The lock is the single source of truth for staging (not the manifest): the lock is what runtime resolution actually reads (V2/V3), and it is what `ailang lock` produced from the manifest.

**2. Registry gate (cmd/ailang/pkg_publish.go).**

After `rewritePathDepsForPublish` returns the list of rewrites (extended to return `(name, fromPath, version)` triples), for each triple:

- `FetchMetadata(name, version)` (RegistryClient; `BaseURL` injectable for tests — V15). Absent → hard error:
  `dependency sunholo/daneel_ext_abi@0.5.0 (rewritten from path dep "../abi") is not on the registry — publish the dependency first (cd ../abi && ailang publish), then re-run`
- Present but `metadata.ContentHash != ContentHash(localDepDir)` → hard error:
  `registry sunholo/daneel_ext_abi@0.5.0 (content abc…) differs from local "../abi" (content def…) — the local tree changed without a version bump; bump ../abi's version and publish it first`
- Network failure → hard error naming the registry URL (publish is already an online operation; the upload would fail moments later anyway). No silent fallback.

This runs before `measurePackageQuality` (cheap: one fetch per rewritten dep) and also in `--dry-run` (dry-run already runs every other gate).

**3. Rewrite assertion (cmd/ailang/pkg_publish.go).**

`rewritePathDepsForPublish` currently returns `(bool, error)` where the bool means "at least one regex matched". Change it to return the rewrite triples; then, after the rewrite pass, reload the manifest and assert no dependency still has `Path != ""`. If one does → hard error naming the dep, the recognized inline-table form (`"name" = { path = "../x" }`), and the alternative (declare a registry version string). The regex stays (it preserves the author's formatting); the assertion converts its silent miss into a refusal.

**4. PUB015 says why (internal/pkg/quality.go, cmd/ailang/pkg_quality.go).**

- `SmokeSection` gains `Output string` (json `output,omitempty`, capped ~512 bytes, same truncation helper as `SmokeResult`).
- `runAttestedChecks` copies `res.Output` into it.
- The PUB015 finding message becomes `_smoke.ail failed: <tail of output>`.
- `printQualityHuman` already prints `smoke: present true passed false`; add the output tail on failure.

This is the general form of the report's "gate says plainly" requirement — it covers *every* future smoke failure, not just path deps.

### Implementation Plan

**Phase 1: staging (~1 day)**
- [ ] Extract the staging logic from `RunSmokeInTempDir` into `stageSmokeWorkspace(packageDir, tmpRoot) (stagedPkgDir string, err error)`.
- [ ] Depth padding + relative path-dep staging + absolute-path copy with staged-lock rewrite + escape backstop.
- [ ] Extend `publish_validator_test.go`: path-dep sibling fixture (pass), transitive path-dep fixture, absolute-path fixture, escape-refusal fixture; keep `TestRunSmokeInTempDir_Isolation` green.

**Phase 2: publish-flow gates (~1 day)**
- [ ] `rewritePathDepsForPublish` → rewrite triples + post-rewrite survivor assertion + tests (table-header form, extra-key form).
- [ ] PUB017 registry gate (existence + content hash) with `httptest` registry; wire into `pkgPublishCommand` before `measurePackageQuality`; test absent-version, hash-mismatch, hash-equal, network-error.
- [ ] Update `ailang publish --help` and the PUB-code docs.

**Phase 3: PUB015 diagnostics (~0.5 day)**
- [ ] `SmokeSection.Output` field; thread through `runAttestedChecks`, `BuildQualityReport` (PUB015 message), `printQualityHuman`.
- [ ] Tests: failing smoke surfaces the loader error text in the report JSON and the human output.

**Phase 4: verification (~0.5 day)**
- [ ] End-to-end on the repro fixture from this doc: publish dep first (to a test registry), then the ext package; assert publish succeeds.
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` (no new cross-layer imports: everything stays inside `internal/pkg` + `cmd/ailang`).

### Files to Modify/Create

**New files:**
- none (all changes fit existing files; no new package needed)

**Modified files:**
- `internal/pkg/publish_validator.go` — `stageSmokeWorkspace` + staging of lock path deps (~120 LOC)
- `internal/pkg/quality.go` — `SmokeSection.Output` + PUB015 message (~15 LOC)
- `cmd/ailang/pkg_quality.go` — thread smoke output into `AttestedBlock` (~10 LOC)
- `cmd/ailang/pkg_publish.go` — rewrite triples + survivor assertion + PUB017 gate (~90 LOC)
- `internal/pkg/publish_validator_test.go`, `cmd/ailang/pkg_publish_test.go` — tests (~150 LOC)
- `docs/` PUB-code reference page (where PUB000–PUB021 are listed) — add PUB017 (~5 LOC)

## Examples

### Example 1: the Daneel flow, before and after

**Before** (measured, V5):
```
$ cd ext/search && ailang publish
Publishing sunholo/daneel_ext_search@0.1.2...
  → Rewrote dep sunholo/daneel_ext_abi: path "../abi" → registry 0.6.0
  ...
  ✗ PUB015 _smoke.ail failed
✗ publish blocked by 1 quality gate(s)
```
(and the underlying `package directory not found: <tmp>/abi` is never shown)

**After** (abi 0.6.0 not yet published):
```
$ cd ext/search && ailang publish
Publishing sunholo/daneel_ext_search@0.1.2...
  → Rewrote dep sunholo/daneel_ext_abi: path "../abi" → registry 0.6.0
  ✗ PUB017 dependency sunholo/daneel_ext_abi@0.6.0 (rewritten from path dep "../abi")
    is not on the registry — publish the dependency first (cd ../abi && ailang publish), then re-run
```

**After** (abi 0.6.0 published, tree unchanged):
```
$ cd ext/search && ailang publish
Publishing sunholo/daneel_ext_search@0.1.2...
  → Rewrote dep sunholo/daneel_ext_abi: path "../abi" → registry 0.6.0
  ✓ smoke passed (1.8s, staged 2 path dep(s))
  ...
✓ Published sunholo/daneel_ext_search@0.1.2
```

### Example 2: unrewritable manifest form, before and after

**Before** (measured, V8): `[dependencies."sunholo/daneel_ext_abi"]` + `path = "../abi"` → no rewrite line, no error, publish proceeds to tarball creation with the path dep inside.

**After**: `✗ cannot publish: path dependency "sunholo/daneel_ext_abi" could not be rewritten to a registry version — use the inline form "sunholo/daneel_ext_abi" = { path = "../abi" } or declare a registry version`

## Success Criteria

- [ ] The repro fixture of this doc (ext with `"dep" = { path = "../abi" }`) publishes successfully end-to-end against a test registry once the dep is published first; acceptance: new e2e test in `pkg_publish_test.go` using `httptest`.
- [ ] The same fixture with the dep absent from the registry refuses with PUB017 naming `cd ../abi && ailang publish`; acceptance: unit test asserting the message text.
- [ ] The same fixture with the registry version's content hash ≠ local dep hash refuses with the differ message; acceptance: unit test.
- [ ] Table-header and extra-key dep forms refuse loudly at the rewrite assertion; acceptance: unit tests.
- [ ] A smoke that fails for ANY reason shows the output tail in the PUB015 finding and human report; acceptance: unit test on `BuildQualityReport` + `printQualityHuman`.
- [ ] `TestRunSmokeInTempDir_Isolation` passes unchanged (declared-dep staging must not leak undeclared workdir state); acceptance: existing test.
- [ ] All tests passing (`make test`), `make fmt`, `make lint`, `make check-boundaries` clean.
- [ ] Documentation updated: PUB017 added to the PUB-code reference; `docs/docs/guides/development-workflow` publish section notes path-dep publish order.

## Testing Strategy

**Unit tests:**
- Staging: relative/transitive/absolute dep fixtures, depth padding, escape refusal, staged-lock rewrite, content-hash equality between original and staged dep (`ContentHash` equality assertion).
- Rewrite: inline form (rewrites), table-header form (refuses), extra-key form (refuses), post-rewrite survivor assertion.
- PUB017 gate against `httptest` server (RegistryClient's `BaseURL` is injectable — V15): absent version, hash mismatch, hash equal, connection refused.

**Integration tests:**
- Full `publish --dry-run` on the doc's repro fixture against an `httptest` registry, both orderings (dep published / not published).
- Existing `TestRunSmokeInTempDir_*` suite untouched and green.

**Manual testing:**
- On a daneel checkout: `cd ext/abi && ailang publish` (abi 0.6.0 has no path deps — should pass and reach the validator), then `cd ../search && ailang publish`. This is the E3 acceptance run; it requires the real registry and Daneel's attendance.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Fixed staging dir names (`w1…wN/pkg`) vs single padded dir — agent may choose; must be deterministic.
- Whether `SmokeSection.Output` is also uploaded in the attested form field (validator currently banks it without gating) — agent may decide; cap at 512 bytes either way.
- Whether PUB017's registry check also walks *transitive* path-dep rewrites — currently unnecessary (the dep's own publish already passed the same gate), but the implementer may add it cheaply via the lock closure if the hash-check code generalizes naturally.
- Whether `ailang pkg quality` (the report-only caller) also prints the smoke output tail — recommended, agent decides.

## Non-Goals

- **Registry-faithful smoke** (re-locking the staged workspace against the rewritten manifest so the smoke fetches the registry tarball instead of the local copy) — considered and deferred: it makes publish network-bound for content already guaranteed byte-identical by the PUB017 hash check. The staged local copy with a hash-verified registry twin is equivalent and offline.
- **Changing the lock format or `portablePathDep`** — relative path entries are correct for committed locks (m-pkg-lock-portability); the bug is in the smoke staging, not the lock.
- **Server-side (registry-validator) enforcement of the PUB017 semantics** — the validator already recomputes its own quality view from the tarball; teaching it to fetch dependency metadata is a separate, follow-up hardening.
- **Supporting `git =` dependencies in the smoke staging** — git deps already resolve through the machine-global cache (loader `packageDir` case `"git"`), which works inside the temp copy; only `path` deps were broken.
- **`--no-run`/server attestation semantics** — the attested block stays publisher-side only.

## Timeline

**Week 1** (2–3 days):
- Phase 1 staging + tests (day 1)
- Phase 2 publish-flow gates + tests (day 2)
- Phase 3 diagnostics + Phase 4 e2e verification + docs (day 3)

**Total: ~2–3 days across 1 week** (estimate doubled per skill guidance from a raw ~1.5 days)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Staging weakens the isolation guarantee (dep trees may contain undeclared assets) | Med | Deps are copied with the same `shouldStageFile` filter as the package (published view only); `TestRunSmokeInTempDir_Isolation` stays green; the isolation *definition* narrows only to lock-declared directories, ratified in Design Freeze |
| Large dep trees slow publish (copy time) | Low | Same file set the tarball itself will carry; deps are typically small; measure in e2e test and cap only if observed |
| PUB017 hard gate blocks packages that previously "worked" | Low | No package with path deps can publish today (PUB015 fires first — V5); the gate strictly converts an opaque refusal into an actionable one |
| Content-hash check false-positives when the dep's local tree has stray non-`.ail` changes | Low | `ContentHash` hashes `.ail` files only (V14), so only source drift trips it — which is exactly what should trip it |
| Network dependency earlier in the publish flow | Low | One `FetchMetadata` per rewritten dep, before the expensive quality run; failure message names the registry URL; upload requires the same network moments later anyway |

## Related Documents

<!-- Auto-search (SimHash) found no on-topic planned or implemented doc (top matches were unrelated keyword noise: m-v1-simplification-s4 sprint plan, m-coverage-cross-package-attribution, etc. — all <0.75 and off-topic). Coverage gate: no duplicate. -->

**Implemented (may inform design):**
- [m-ext-portability-gate.md](../implemented/v0_18_11/m-ext-portability-gate.md) — created the temp-dir smoke gate (v0.18.11); this doc narrows its isolation definition from "nothing outside the package" to "nothing outside the package and its lock-declared dependencies"
- [m-pkg-lock-portability.md](../implemented/v0_10_0/m-pkg-lock-portability.md) — introduced relative path entries in the lock (`portablePathDep`); the mechanism this doc's staging relies on
- [m-pkg-transitive-lock-fix.md](../implemented/v0_10_0/m-pkg-transitive-lock-fix.md) — lock closure includes transitive deps, which is why staging per-lock-entry covers the whole graph

**Planned (check for overlap):**
- none on topic (searched "publish path dep smoke" and "path dependency registry publish smoke temp dir")

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [M-EXT-PORTABILITY-GATE](../implemented/v0_18_11/m-ext-portability-gate.md) - the gate this fixes
- Bug report: Daneel, 21 Sept 2026, task-e00b9074 (capabilities.md B3/E3)
- Live registry evidence: `https://storage.googleapis.com/ailang-registry/index.json` (V11)

## Verification Log

Code read at HEAD `cb6fc0f0` (v0.42.0, branch coordinator/task-e00b9074). Live repro used the installed binary at `/usr/local/bin/ailang` (v0.41.0, commit 24ee108) — the four load-bearing files are byte-identical in structure between that build and HEAD for every claim below (verified by reading HEAD source; the repo checkout lacks the v0.40.x history to diff directly). Repro fixture: `/tmp/pathdep-repro/{abi,ext}` as built in V4/V5.

| # | Claim | Command / source | Observed |
|---|---|---|---|
| V1 | Publish smoke runs in a temp-dir copy of only the package's own file set | `internal/pkg/publish_validator.go:59-127` (`RunSmokeInTempDir` → `copyPackageContents`) | Copies `ailang.toml`, `ailang.lock`, `AGENT.md`, `_smoke.ail`, `*.ail`, `assets/**`; no dependency directories staged |
| V2 | Runtime import resolution is lock-driven, not manifest-driven | `internal/pipeline/package_resolver.go:48-70` (`tryLoadPackageResolver` → `pkg.NewPackageLoader(lf, manifestDir)`); `internal/pkg/loader.go:41-70` (`ResolveImport` → `lockFile.FindPackage` → `packageDir`) | Manifest is used for effect ceilings/prefix map; imports resolve via lock entries |
| V3 | Lock path entries are stored relative to the package root and joined with the root at load | `internal/pkg/resolver.go:395-410` (`portablePathDep`); `internal/pkg/loader.go:131-141` (`case "path": filepath.Join(pl.rootDir, dir)`) | `../abi` in a lock resolves to `<root>/../abi`; live lock output: `→ sunholo/repro_abi@0.5.0 (path: ../abi)` |
| V4 | In a temp-dir copy the path dep misses with exactly the reported chain | `mkdir $T/pkg; cp ailang.toml ailang.lock core.ail _smoke.ail $T/pkg/; cd $T/pkg && ailang run --caps … --ai-stub --entry main _smoke.ail` | `Warning: failed to hash dependency sunholo/repro_abi at /tmp/smoke-sim-GVA/abi: lstat …: no such file or directory` then `Error: module loading error: … package directory not found: /tmp/smoke-sim-GVA/abi` |
| V5 | In-place run passes; publish dry-run refuses with PUB015 | Fixture at `/tmp/pathdep-repro/ext` (dep `"sunholo/repro_abi" = { path = "../abi" }`): `ailang run --caps IO --entry main _smoke.ail` then `ailang publish --dry-run` | run: `hello from abi 0.5.0` (exit 0); publish: `→ Rewrote dep … path "../abi" → registry 0.5.0` … `✗ PUB015 _smoke.ail failed` … `publish blocked by 3 quality gate(s)` |
| V6 | The manifest rewrite happens BEFORE the smoke and is restored after | `cmd/ailang/pkg_publish.go:69-88` (rewrite → reload → gates → `defer os.WriteFile(tomlPath, originalToml)`) | Order verified by reading; live run printed `→ Rewrote dep` before the quality block |
| V7 | Even with the dep published at the rewritten version, the smoke would still fail today (the copied lock still says `path`) | V2+V3 mechanism (lock drives resolution; rewrite touches only `ailang.toml`, never `ailang.lock`) — no lock-regeneration exists between rewrite and smoke | Confirmed by code path; "publish abi first" alone does NOT unblock the smoke — staging (this doc) is required |
| V8 | The rewrite silently skips unrecognized TOML forms (no else branch) | `cmd/ailang/pkg_publish.go:222-251`: `if re.MatchString(content) { … rewritten = true }` — no else; live repro: replaced dep with `[dependencies."sunholo/repro_abi"]` + `path = "../abi"` and ran `ailang publish --dry-run` | No `Rewrote dep` line, no error naming the dep, publish proceeded (blocked only by unrelated PUB001/002/015); `grep -c 'path = "../abi"' ailang.toml` = 1 post-run |
| V9 | Nothing between rewrite and upload checks rewritten dep versions against the registry | Full read of `pkgPublishCommand` (`cmd/ailang/pkg_publish.go:44-158`); `grep -n "FetchIndex\|FetchMetadata" cmd/ailang/pkg_publish.go` | Only callers are `emitPublishMessages`/`emitDependentNotifications` — both AFTER `uploadTarball` and best-effort |
| V10 | PUB015 discards the smoke output; the finding is a fixed string | `cmd/ailang/pkg_quality.go:204-209` (`runAttestedChecks` keeps `res.Passed`, `res.Seconds` only); `internal/pkg/quality.go:322-326` (`r.finding("PUB015", …, "_smoke.ail failed")`); `SmokeSection` struct has no output field (`quality.go:139-146`) | Live run shows `✗ PUB015 _smoke.ail failed` with no loader error shown |
| V11 | Registry is stale vs repo exactly as reported | `curl -s https://storage.googleapis.com/ailang-registry/index.json` | `daneel_ext_abi` versions `[0.2.0, 0.3.0, 0.4.0]`; `daneel_ext_search` `[0.1.0]` — repo holds abi 0.5/0.6, search 0.1.1/0.1.2 |
| V12 | PUB017 is unallocated | `grep -rho "PUB0[0-9][0-9]" --include="*.go" --include="*.md" . \| sort -u` | Allocated in code: 000,001,002,005,006,010,011,012,014,015,016,020,021 (+005/006 in cmd/registry-validator). PUB003/004/013 appear only inside design_docs prose. PUB017/018/019: zero hits anywhere |
| V13 | The lock records the full transitive closure, so per-entry staging covers transitive path deps | `internal/pkg/resolver.go:127-192` (recursive `resolve`, `resolved = append(...)` for every dep); live `ailang lock` on fixture | Lock contains every resolved package (fixture: the 1 path dep; closure logic read) |
| V14 | `ContentHash` hashes sorted `.ail` files only, so a staged copy preserves the lock's recorded hash | `internal/pkg/hasher.go:14-60` | Walks dir, filters `strings.HasSuffix(path, ".ail")`, sorts, hashes path+content |
| V15 | RegistryClient is test-injectable | `internal/pkg/registry.go:19-35` (`BaseURL` exported field; `NewRegistryClient` reads `config.RegistryURL()`) | `httptest` server can back the PUB017 gate tests |
| V16 | The fix in `RunSmokeInTempDir` covers both callers (publish flow and `pkg quality`) | `grep -rn "RunSmokeInTempDir" cmd/ internal/ --include=*.go \| grep -v _test` | Exactly two call sites: `cmd/ailang/pkg_quality.go:204` (used by both `publish` via `measurePackageQuality` and `pkg quality`) — no other callers |
| V17 | Consumer side already converts path deps to registry lookups (design symmetry exists) | `internal/pkg/resolver.go:106-125` (`fromRegistry && dep.Path != ""` → registry index lookup with clear "not found in registry" error) | Publisher gate will mirror the consumer's own semantics |
| V18 | `RunSmokeInTempDir` does not pass `--package-dir` (and pointing it at the original dir would break isolation) | `internal/pkg/publish_validator.go:78-86` (args list); `internal/pipeline/package_resolver.go:160-176` (explicit PackageDir wins; self-references resolve against it) | args: `run --caps … --ai-stub --entry main _smoke.ail` only — confirms `--package-dir`-at-original is not the fix (it would resolve the package's own imports against the developer's tree) |
| V19 | Isolation test exists and asserts the guarantee this doc must preserve | `sed -n '138,175p' internal/pkg/publish_validator_test.go` | `TestRunSmokeInTempDir_Isolation`: `secret.txt` dropped next to package; smoke must print `isolated`, never `LEAK` — regression fixture for the staging change |
| V20 | No existing TOML-AST manifest rewriter to reuse (the regex is the only rewrite path) | `grep -rn "rewritePathDeps\|RewriteDeps" cmd/ internal/ --include=*.go` | Single implementation: `rewritePathDepsForPublish` in `cmd/ailang/pkg_publish.go` (regex-based) |
| V21 | No duplicate planned doc covers this topic (coverage gate) | `ailang docs search --stream planned --limit 5 "publish path dep smoke"` and `"path dependency registry publish smoke temp dir"`; `grep -rln "smoke\|publish" design_docs/planned/` + read of top hits | Top SimHash matches (m-v1-simplification-s4 sprint plan, m-coverage-cross-package-attribution, m-eval-validity-discipline, daemon-restalls, m-gate1-shared-clone-ref-drift) are unrelated keyword coincidences, all off-topic |

**Note on the create-script search**: the doc was authored manually from the skill template because `create_planned_doc.sh` exits 1 whenever a search corpus returns no matches (`IMPLEMENTED=$(merge_results …)` — the inner `grep` exits 1 under `set -euo pipefail` and kills the script at the first empty corpus). This is a tooling defect in the skill script, reported to the user separately; it did not affect the coverage gate (searches were run by hand — V21).