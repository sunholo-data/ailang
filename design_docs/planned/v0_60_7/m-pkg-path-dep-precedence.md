# M-PKG-PATH-DEP-PRECEDENCE: Direct local dependencies win over transitive registry pins

**Status**: Planned
**Target**: v0.60.7
**Priority**: P0 (High) — silent wrong-code execution under exit 0
**Estimated**: 3 days (13h implementation incl. tests/docs, ×2 buffer)
**Dependencies**: None. Reaffirms the v0.9.11 ruling in [m-pkg-resolver-direct-wins.md](../../implemented/v0_9_11/m-pkg-resolver-direct-wins.md) rather than changing it.
**Reported**: task-207a73de, found while preparing sunholo-data/ailang-packages#206 (stapledons mission iteration 109); repro originally on v0.56.1, re-confirmed on v0.60.6 (see Verification Log).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the lock's *source* for a name depends on the alphabetical order of sibling dep names (empirically confirmed, V3): renaming `crew` → `zcrew` flips the resolution. The claim pre-pass makes source selection a function of the manifest, not of names. |
| A2: Replayability | 0 | Lock schema unchanged; locks remain byte-replayable (entries are name-sorted, V7). |
| A3: Effect Legibility | 0 | No effect changes. |
| A4: Explicit Authority | 0 | No capability changes. The *manifest* gains de-facto authority it was already promised (packages guide line 378, V9). |
| A5: Bounded Verification | 0 | No verification changes. |
| A6: Safe Concurrency | 0 | Resolver is single-threaded. |
| A7: Machines First | +1 | An agent (or developer) testing a local patch gets exit 0 only when the patch is actually in the lock. Today exit 0 means "maybe". Manifest→lock inference becomes machine-trustworthy. |
| A8: Minimal Syntax | 0 | No manifest or lock schema changes. |
| A9: Cost Visibility | 0 | No resource changes. |
| A10: Composability | +1 | Monorepo path-dep layouts (the documented `sim/`+`ai/` pattern) compose with registry deps without alphabetical luck. |
| A11: Structured Failure | +1 | Version mismatch between a local claim and a transitive pin becomes a correctly-attributed `VersionConflictError` (today the root is mislabeled as the transitive requirer, V2/V13); shadowing becomes a hard invariant error instead of a post-write warning. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes name-ordering luck; introduces no new nondeterminism
- [x] A3 (Effects): no effect changes
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): aligns observable outcome (exit 0 + lock) with declared intent; not human convenience

## Problem Statement

The packages guide states: "Direct dependencies are authoritative — the resolver never silently downgrades them" ([docs/docs/guides/packages.md:378](../../../docs/docs/guides/packages.md), V9). For a direct **path** dependency, that sentence is false whenever a transitive dependency pins the same package: the resolver is first-come-first-served by name (`resolvedSet`, resolver.go), siblings are visited in alphabetical order, and a same-version collision is "silently deduplicated" per the v0.9.11 policy — with the registry copy kept when the transitive carrier's name sorts first.

**Concretely (reporter's topology, re-confirmed on v0.60.6, V1/V2/V3):**

`sim/ailang.toml` declares `"sunholo/crew_core" = "0.6.0"` (registry) and
`"sunholo/social_dynamics" = { path = "<clone>" }` (local, patched). `crew_core` pins
`"sunholo/social_dynamics" = "0.4.0"` exactly. Because `crew_core` sorts before
`social_dynamics`, the registry copy of social_dynamics@0.4.0 is resolved first; the
direct path dep then hits the same-version dedup path and is **dropped**:

```
$ ailang lock
⚠ dependency sunholo/social_dynamics is declared as a path dep but was resolved
  from registry (already pinned via a transitive dependency); your local
  sunholo/social_dynamics source is NOT in this lock
✓ Generated ailang.lock (2 packages)
  → sunholo/social_dynamics@0.4.0 (registry)
  → crew_core@0.6.0 (registry)
EXIT=0
```

The lock's content hash is the **unpatched registry copy's**. A developer who thinks they are testing their patch gets exit 0 on unpatched code. If the clone's version is bumped to 0.4.1, the same operation instead fails:

```
Error: dependency resolution failed: version conflict: sunholo/social
  already resolved: 0.4.0
  transitive requires: 0.4.1 (via sunholo/sim)
...
EXIT=1
```

Note the second error is doubly wrong: the **root** (`sim`) is labeled the transitive requirer, and there is no "root requires" line, because the post-resolution authoritative check only tracks deps declared with `Version` (registry pins) — path deps never enter `directDeps` (V5).

**There is no supported way to test a patched dependency locally against a consumer.** The reporter's team fell back to editing a scratch copy of the registry cache plus the lock's content hash — i.e., forging the artifacts the toolchain is supposed to guarantee.

**Impact:**
- Any developer or agent iterating on a dependency against a consumer (the standard prepublish workflow) — silent wrong-code execution under exit 0.
- Any CI trusting `ailang lock --check`: the check re-resolves with the same flawed precedence, so drift is *not* even reported — the shadowed lock looks "current".
- The failure is order-coupled: identical manifests behave differently depending on sibling package names (V3), which is A1 nondeterminism in practice.

## Goals

**Primary Goal:** A direct path (or git) dependency in a manifest always supplies the resolved entry for its name — at equal version it overrides any transitive registry pin, and at unequal version resolution fails with a correctly-attributed structured conflict. Exit 0 then implies the local source is in the lock.

**Success Metrics:**
- Reporter's exact topology (carrier sorts before the path dep, versions equal): lock entry is `source: "path"` with the local clone's content hash, no warning, exit 0 (new resolver test, red today).
- Same topology with the clone bumped: `VersionConflictError` naming the root's local version and the transitive requirer; exit 1; actionable suggestion.
- A shadowed direct path dep can no longer produce a lock at all: the invariant trips at the resolver chokepoint (`ResolveDependenciesFrom`), covering `lock`, `lock --check`, publish smoke staging, and the registry validator — not just the lock writer.
- Zero regressions: all existing resolver/lock/publish/confinement tests pass unmodified except the two that assert the warning-instead-of-error behavior.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: A manifest's local (path/git) claims beat registry pins for the same name **at equal version** | Extends the v0.9.11 "same version from multiple sources → silently deduplicate" policy with a deterministic source rule; makes the guide's "authoritative" sentence true | human (design review — semantic extension of a 2026-03-24 ruling) | design | med |
| D2: At **unequal** version, local claims do NOT override — resolution fails with a structured conflict | Reaffirms the human ruling of 2026-03-24 ("published package dependency contracts are inviolable; the resolver fails with structured conflicts"); silent override would break flat exact-version semantics | human (reaffirmation, recorded here) | design | low |
| D3: No new manifest surface (`[replace]` / `ailang.local.toml`) in this feature | The queued Go-style-replace feature ([local-dep-replace-override.md](../ailang-core-triage/local-dep-replace-override.md)) covers cross-version overrides; this doc must not pre-empt its open trade-offs | human (design review) | design | low |
| D4: Mechanism = claim pre-pass (pending map consulted by every registry/git resolution), not mere class reordering | Class reordering (path deps before registry siblings) is provably insufficient: a path *sibling's subtree* can still claim the name first (see Architecture, cross-subtree fixture) | agent | design | high |
| D5: Shadowed-path-dep detection moves from a post-write warning in `pkgLockCommand` to a hard error inside `ResolveDependenciesFrom`, before any lock write | Warning-after-write cannot protect `--check`, publish staging, or the validator, and exits 0 (the dangerous part); chokepoint placement covers all four callers | agent | design | low |
| D6: Accept one-time lock drift for previously-shadowed repos | Repos with a shadowed lock will fail `lock --check` once and rewrite to the path source on the next `ailang lock`. This is the fix propagating, but it is a deliberate user-visible incompatibility | human | compile | low |

### Design Freeze

- [x] D1 — equal-version local-wins (this doc; needs design-review sign-off)
- [x] D2 — unequal-version still fails loudly (reaffirms 2026-03-24 ruling)
- [x] D3 — no new manifest surface in this feature
- [x] D5 — hard error at the resolver chokepoint
- [ ] D6 sign-off — reviewer must confirm the one-time `--check` break for affected repos is acceptable (audit from Phase 4 informs this)

## Solution Design

### Overview

Three coordinated changes, all inside the existing resolver semantics (flat deps, exact versions, lock schema untouched):

1. **Claim pre-pass.** When the resolver begins resolving a manifest `m` (root or any package), it first *claims* every dependency of `m` declared with `path`/`git`: it loads the claimed dep's manifest to learn its version and records `claims[name] = {version, owner: m}` in a shared map — before recursing into any sibling subtree. Any later attempt to resolve `name` from `registry` (or git) consults `claims`: equal version → the attempt is satisfied by the pending claim (no download, no registry entry); unequal version → `VersionConflictError`. When `m`'s own iteration reaches the claimed dep, it resolves normally (source `path`/`git`) and the claim is retired.
2. **Conflict attribution.** `VersionConflictError` learns about local claims: `DirectVersion` is populated from the claim's manifest version and a new `DirectSource` field renders `root requires: 0.4.1 (path dep: ../clone)`. The mislabeled "(via sunholo/sim)" line disappears because the requirer is now correctly the transitive package.
3. **Chokepoint invariant.** After resolution, `ResolveDependenciesFrom` verifies every root manifest dep declared with `Path` resolved to `Source == "path"` (the current `warnShadowedPathDeps` predicate, inverted into a check). Violation returns a typed error ("resolver invariant violated: … file a bug"), so no lock is ever written from a shadowed resolution. The post-write warning in `cmd/ailang` is deleted.

With (1) alone, D1's guarantee holds structurally, which is what makes (3) a tripwire rather than a load-bearing check — and what makes the cross-subtree hole (below) impossible instead of merely unlikely.

### Architecture

**Why first-come-first-served is unfixable by reordering.** The obvious smaller fix — visit path/git deps before registry deps at each level — closes the reporter's case but leaves this one open: root declares `X = { path }` and `Y = { path }`, `Y` sorts before `X`, and `Y`'s subtree pins `X@0.4.0` from the registry. `Y`'s subtree completes first, registers `X` from the registry, and the root's path claim on `X` dedups against it silently. Any sibling-subtree-first ordering can shadow a later sibling's claim; only making claims *before any subtree recursion starts* closes the class. (D4)

**Components (all in `internal/pkg/resolver.go` unless noted):**

1. **`depClaims` map** — `map[string]localClaim`, `localClaim{version string, source string, ownerDir string}`. Populated at the top of `resolve()` for the manifest being resolved (path deps via `LoadManifest` of the target dir; git deps via `resolveGitSource`, which is idempotent/cached per `gitcache.go`). If a claim already exists for a name (two manifests claiming the same local dep): equal version → keep; unequal → `VersionConflictError` immediately (this is a genuine two-local-sources conflict, today only caught by traversal luck).
2. **Registry/git resolution gates** — in the existing `dep.Path != ""` / `dep.Git != ""` / registry branches, before any download or cache lookup, consult `depClaims[depName]`: claimed-and-equal → skip (the claim owner will emit the entry); claimed-and-unequal → `VersionConflictError{DirectVersion: claim.version, DirectSource: …, RequestedBy: m.Package.Name}`. The `fromRegistry` path→registry conversion (`dep = Dependency{Version: entry.Latest}`) funnels into the registry branch, so it is gated automatically.
3. **Claim retirement** — when the owner manifest's iteration reaches the claimed dep, resolve it exactly as today (path/git branch: recurse subtree, append `ResolvedPackage` with source `path`/`git`), then delete the claim. Post-resolution, `len(depClaims) != 0` is an internal error (every owner iterates all its deps).
4. **`VersionConflictError` extension** — add `DirectSource string` (renders as `(path dep: ../clone)` / `(git dep: …)`); populate `DirectVersion` for path/git root deps (today `directDeps` only collects `dep.Version != ""` entries, V5). Message for D2's case: `root requires: 0.4.1 (path dep: ../clone)` + `transitive requires: 0.4.0 (via sunholo/crew@0.6.0)` + suggestions: keep the clone at 0.4.0 while iterating (the supported local-override workflow, now documented), or republish crew against 0.4.1.
5. **Post-resolution invariant** — for each root dep with `Path != ""` (and each git dep): the resolved entry must exist with `Source == "path"`/`"git"`. Typed error `ErrShadowedLocalDep` (name, declared source, actual source). This replaces `warnShadowedPathDeps`: the function and its call site in `pkgLockCommand` (after `lf.Save`, V6) are deleted; the invariant runs inside `ResolveDependenciesFrom`, so `lock`, `lock --check` (`CheckLock` → `ResolveLockFrom`), `publish` smoke staging, and the registry validator all fail loudly without per-caller wiring.
6. **Docs** — rewrite the "Direct dependencies are authoritative" paragraph in the packages guide to state source precedence (local claims win at equal version; unequal = structured error) and document the local-testing workflow: *keep the clone's `version` at the pinned version while iterating; `ailang lock` now picks up your edits; bump and republish when the patch ships.*

Determinism note: sibling iteration stays alphabetical (unchanged); lock bytes are name-sorted (V7), so only the *winner* of a name collision changes, never ordering of the lock.

### Implementation Plan

**Phase 1: Resolver claims** (~5 hours)
- [ ] Add `localClaim` + `depClaims` pre-pass in `resolve()` (path deps first; git claims share `resolveGitSource` cache)
- [ ] Gate the registry and git branches (incl. the `fromRegistry` conversion path) on `depClaims`
- [ ] Claim retirement on owner iteration + empty-claims post-check
- [ ] New resolver tests: reporter topology (carrier sorts first, equal version → `source: "path"`, local content hash); cross-subtree fixture (two path siblings, earlier sibling's subtree pins the later one's name)

**Phase 2: Conflict attribution + chokepoint invariant** (~4 hours)
- [ ] `DirectSource` field; populate `DirectVersion` for path/git root deps; rewrite the suggestion block for the D2 case
- [ ] `ErrShadowedLocalDep` invariant in `ResolveDependenciesFrom` (root path/git deps resolved with matching source)
- [ ] Delete `warnShadowedPathDeps` + call site; move nothing else — `pkgLockCommand` unchanged otherwise
- [ ] Update `cmd/ailang/pkg_lock_warnings_test.go` to assert the hard error (and that no lock file is written on shadowing)

**Phase 3: Conflict-surface regression tests** (~3 hours)
- [ ] Version-mismatch case (clone at 0.4.1): `VersionConflictError` with `DirectVersion=0.4.1`, `RequestedBy` = the transitive carrier, message contains `(path dep:` and both suggestions
- [ ] Direct **git** dep shadowed by transitive registry pin (same shape as the path case)
- [ ] Two manifests claiming the same name locally at different versions → structured conflict at pre-pass
- [ ] Full `make test` (existing fixtures below must pass: transitive path→registry conversion, direct-vs-transitive version conflict, same-version no-conflict, lock portability/relative paths, publish smoke staging, confinement git carry-forward)

**Phase 4: Docs, changelog, audit** (~3 hours)
- [ ] packages.md: source-precedence paragraph + local-testing workflow; CHANGELOG entry (user-visible: previously-shadowed locks will drift once — D6)
- [ ] Audit first-party repos (ailang-packages, stapledons-godot `sim/`): count manifests with a direct path dep whose committed lock records the same name with `source != "path"` — the one-time `--check` blast radius, reported back for the D6 sign-off

### Files to Modify/Create

**Modified files:**
- `internal/pkg/resolver.go` (~+90/−15 LOC) — claims pre-pass, branch gates, claim retirement, `DirectSource`, `ErrShadowedLocalDep`, post-resolution invariant
- `internal/pkg/resolver_test.go` (~+180 LOC) — reporter topology, cross-subtree, git variant, double-claim, mismatch attribution tests
- `cmd/ailang/pkg_commands.go` (~−8 LOC) — delete `warnShadowedPathDeps` call (and the warning's write)
- `cmd/ailang/pkg_lock_warnings.go` (deleted, −25 LOC) — replaced by the resolver invariant
- `cmd/ailang/pkg_lock_warnings_test.go` (~+40/−60 LOC) — rewrite the shadow tests as hard-error tests at the `ResolveDependenciesFrom` level; keep the "conflict doesn't rewrite a committed lock" assertions
- `docs/docs/guides/packages.md` (~±20 LOC) — precedence paragraph + workflow
- `CHANGELOG.md` — entry

**New files:** none.

## Examples

### Example 1: Testing a local patch against a consumer (the reporter's case)

**Before (v0.60.6, empirically confirmed, V1):**
```
$ # sim/ailang.toml: crew = "0.6.0" (registry), social = { path = "../clone" }  (clone at 0.4.0, patched)
$ ailang lock ; echo $?
⚠ dependency sunholo/social is declared as a path dep but was resolved from registry ...
✓ Generated ailang.lock (2 packages)
  → sunholo/social@0.4.0 (registry)
0
$ # the patch is NOT in the build; exit 0 lies
```

**After:**
```
$ ailang lock ; echo $?
✓ Generated ailang.lock (2 packages)
  → sunholo/social@0.4.0 (path: ../clone)
0
$ # lock's content_hash is the local clone; edits are picked up on every lock/run
```

### Example 2: Clone version bumped to 0.4.1 while crew pins 0.4.0

**Before (V2):**
```
Error: dependency resolution failed: version conflict: sunholo/social
  already resolved: 0.4.0
  transitive requires: 0.4.1 (via sunholo/sim)      # root mislabeled as transitive
```

**After:**
```
Error: dependency resolution failed: version conflict: sunholo/social
  root requires: 0.4.1 (path dep: ../clone)
  already resolved: 0.4.0
  transitive requires: 0.4.0 (via sunholo/crew@0.6.0)

resolution aborted

suggestion:
  - keep the local clone's version at 0.4.0 while testing your patch, then
    bump and republish sunholo/social
  - or republish sunholo/crew against sunholo/social@0.4.1
```
(Exit 1 in both; only the attribution and suggestion change — per D2 the bump still requires action.)

## Success Criteria

- [ ] Reporter topology test (carrier sorts before path dep, equal versions): resolved entry `source: "path"`, content hash equals the local clone's `ContentHash`, no warning — `TestResolve_DirectPathDepWinsOverTransitiveRegistryPin`
- [ ] Cross-subtree test: two path siblings where the earlier-sorted sibling's subtree pins the later one's name → path still wins — `TestResolve_PathClaimSurvivesSiblingSubtree`
- [ ] Version-bump test: `VersionConflictError` with `DirectVersion=0.4.1`, `DirectSource` rendered, `RequestedBy` = transitive carrier, exit 1 — `TestResolve_PathDepMismatch_AttributedConflict`
- [ ] Invariant test: a root path dep that resolves non-path (unit-level against the new check) → `ErrShadowedLocalDep`, and `pkgLockCommand` writes no lock on that path
- [ ] `ailang lock --check` reports drift exactly once for a previously-shadowed committed lock, and the subsequent `ailang lock` rewrites it to `source: "path"`
- [ ] All tests passing (`make test`); `make fmt`, `make lint`, `make check-boundaries` clean
- [ ] Documentation updated (packages.md precedence + workflow; CHANGELOG)
- [ ] First-party audit script/report committed (Phase 4) with the D6 blast-radius number

## Testing Strategy

**Unit tests (`internal/pkg/resolver_test.go`, mock registry as in `TestResolveDependencies_TransitiveRegistryWithPathDeps`):**
- All four new fixtures above; plus two-local-claims-same-name (equal → dedup path, unequal → pre-pass conflict) and the git-dep variant
- Determinism: assert the resolved slice and lock bytes are identical across 10 repeated resolutions

**Integration tests (`cmd/ailang/`):**
- Update `pkg_lock_warnings_test.go`: shadowing is now an error and no lock is written; the read-only/drift-never-rewrites properties are preserved
- `pkg_lock_check_test.go`: shadowed committed lock → drift line names the source change
- Keep green unmodified: `pkg_confinement_*` (git carry-forward must still work with claim gates), publish smoke staging (`publish_stage.go` → `ResolveLockFrom`), registry validator

**Regression fixtures ("programs that MUST still work" — verified to exist, V8):**
- `TestResolveDependencies_TransitiveRegistryWithPathDeps` — the fromRegistry path→registry conversion (legacy pre-rewrite tarballs)
- `TestResolveDependencies_VersionConflict_DirectVsTransitive` and `TestResolveDependencies_SameVersionNoConflict` — the v0.9.11 semantics, unchanged by D1/D2
- `TestLockFile_PathDepsAreRelativeAndPortable` (lock portability; `portablePathDep`)
- The stapledons `sim/`+`ai/` monorepo layout described in packages.md "Lock File Portability"

**Manual testing:**
- Re-run the V1–V3 repro script (mock registry, `AILANG_REGISTRY` + fake `HOME`) end-to-end against the built binary for both cases

## Deferred Decisions

- Exact wording/format of `DirectSource` rendering in the conflict error — agent may choose (must contain the relative path and the word `path dep:`/`git dep:`)
- Whether `depClaims` lives in a small struct passed to `resolve` vs a closure variable — agent may choose
- Whether git claims participate in the pre-pass or only gate at their branch (both are correct; git claims pay an early `resolveGitSource`) — agent may choose, confinement tests must stay green
- Audit implementation (one-off script vs `ailang pkg` subcommand) — agent may choose; if a subcommand, it needs its own doc row in the changelog
- Test fixture naming/organization — agent may choose

## Non-Goals

- **Cross-version local override** (Go-style `replace`, `ailang.local.toml`, `[replace]` + `AILANG_DEV=1`) — queued as its own design-doc in the backlog ([local-dep-replace-override.md](../ailang-core-triage/local-dep-replace-override.md)); D2 keeps this feature's semantics compatible with whatever it decides
- **Multiple versions per name / semver ranges** — flat exact-version deps are reaffirmed, not changed (v0.9.11 non-goal, kept)
- **Fixing the `fromRegistry` conversion's `entry.Latest` fallback** — a pre-rewrite-legacy tarball's transitive path dep resolves to the registry *latest*, not a pinned version; pre-existing, orthogonal, and unreachable for post-rewrite packages (`publish_rewrite.go`); noted for the backlog
- **`pkg tree` display of precedence** — cosmetic; the tree does not show resolution sources for registry deps today
- **Lock schema changes** — none; `source`/`path`/content-hash fields already express everything needed

## Timeline

**Week 1** (15 hours):
- Phase 1: Resolver claims + new fixtures (7h)
- Phase 2: Attribution + chokepoint invariant + cmd cleanup (5h)
- Phase 3: Regression sweep + full `make test` (3h)

**Week 2** (7 hours):
- Phase 4: docs, CHANGELOG, first-party audit + D6 sign-off (4h)
- Quorum/review buffer, polish, manual repro rerun (3h)

**Total: ~22 hours across 2 weeks** (≈13h raw × buffer)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Claim pre-pass loads path-dep manifests (and git sources) before their turn — extra IO for large graphs | Low | One `LoadManifest` per local dep, already done later today (net zero); `resolveGitSource` is cached and idempotent |
| Behavior change surfaces latent conflicts in existing repos that were silently deduped "the wrong way" | Med | D6: one-time drift, audited (Phase 4) before sign-off; error messages are structured and actionable |
| Subtle interaction with confinement / `AILANG_PACKAGE_ROOT` git carry-forward (#1720 D2) | Med | Claims never *fetch* for path deps; git claims reuse the existing cached/carry-forward path; confinement matrix tests must stay green (Phase 3 gate) |
| The invariant fires on an unforeseen legitimate topology | Med | It fires only when a root-manifest path dep resolves non-path — the exact condition warned today; empirically unreachable after the pre-pass; if it fires, that's the tripwire doing its job (bug report, not data loss — no lock is written) |

## Related Documents

**Directly governing (read before review):**
- [m-pkg-resolver-direct-wins.md](../../implemented/v0_9_11/m-pkg-resolver-direct-wins.md) — introduced `resolvedSet`, sorted iteration, `VersionConflictError`, and the two rulings this doc reaffirms/extends: "inviolable contracts" (version mismatch → fail) and "same version from multiple sources → silently deduplicate" (this doc adds the missing source rule). Its planned follow-up `m-pkg-upgrade-chain-tooling.md` (v0.10.0) never shipped; the DX pain it was to address is exactly what the reporter hit.
- [local-dep-replace-override.md](../ailang-core-triage/local-dep-replace-override.md) + backlog row "Local dependency override (Go-style `replace`)" in [ailang-core-backlog.md](../ailang-core-backlog.md) — the cross-version override feature, deliberately not duplicated here (D3; duplicate gate: distinct scope — precedence fix vs new surface)
- [m-pkg-deterministic-lockfile.md](../v0_35_0/m-pkg-deterministic-lockfile.md) and [m-pkg-lock-portability](../../implemented/v0_10_0/m-pkg-lock-portability.md) — lock determinism/portability invariants preserved
- [packages.md "Version Conflict Detection"](../../../docs/docs/guides/packages.md) — the sentence this doc makes true

*(The auto-search matches in the scaffold were noise — unrelated to dependency resolution — and were replaced by the above, found by targeted search over `design_docs/` for path dep / resolver / override.)*

## References

- [Design Axioms](/docs/references/axioms) — scoring above
- Task task-207a73de (coordinator); sunholo-data/ailang-packages#206 prep; design doc `design_docs/planned/r1/m4-7av-sortedjson-keyed-sort.md` V-a (the work that was blocked by this)
- Cargo's path-dependency override and `[patch]` semantics — prior art for "local source wins at compatible version"; Go's `replace` — prior art for the queued cross-version feature

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Case 1 (equal version): path dep silently replaced by registry pin, exit 0 | Reproduced live on the installed v0.60.6 binary with a mock registry (`AILANG_REGISTRY=http://127.0.0.1:8765`, fake `HOME`, sim manifest `{crew="0.6.0", social={path="../clone"}}`, crew pinning social `"0.4.0"`): output `⚠ … resolved from registry …`, `→ sunholo/social@0.4.0 (registry)`, `EXIT=0`, lock `source: "registry"` with the registry copy's hash. Matches the reporter's v0.56.1 transcript verbatim in substance. |
| V2 | Case 2 (clone at 0.4.1): `version conflict`, exit 1, root mislabeled as requirer | Same harness, clone bumped: `version conflict: sunholo/social / already resolved: 0.4.0 / transitive requires: 0.4.1 (via sunholo/sim)`, `EXIT=1`. No "root requires" line because path deps never enter `directDeps` (`resolver.go`: `if dep.Version != ""` only). |
| V3 | Outcome depends on alphabetical sibling order | Same harness, carrier renamed `sunholo/crew` → `sunholo/zcrew` (sorts after the path dep): `→ sunholo/social@0.4.0 (path: ../clone)`, exit 0, no warning. Identical manifest shape, opposite result. |
| V4 | Mechanism: first-come-first-served + same-version silent dedup | `internal/pkg/resolver.go`: `sort.Strings(depNames)`; path branch `if existingVer, already := resolvedSet[depName]; already { if != → conflict; else fallthrough-skip }` — equal version adds no entry, keeps the first (registry) one. |
| V5 | Post-resolution "authoritative" check is blind to path deps | `directDeps` collected only from `dep.Version != ""` entries (`resolver.go`); a `{ path = … }` dep has empty `Version`. |
| V6 | Warning is post-write and cannot affect exit status | `cmd/ailang/pkg_commands.go:529`: `warnShadowedPathDeps(os.Stderr, …)` runs after `lf.Save`, function is `func(…)` printing only (`pkg_lock_warnings.go:13`); sole call site (grep). |
| V7 | Lock bytes are name-sorted, so sibling traversal order doesn't affect lock layout | `internal/pkg/lockfile.go:51-66` — `LockedFromResolved` sorts by name before save. |
| V8 | Existing order-sensitive tests pin child-before-parent post-order within one subtree, not root sibling order | `internal/pkg/resolver_test.go:126-130` (`[test/c, test/b]`, single-dep root), `:61` (single-dep root); grep for `resolved[0]`/`names[0]` across `cmd/ailang/*_test.go` — none pin multi-sibling root order. |
| V9 | The guide promises "Direct dependencies are authoritative" | `docs/docs/guides/packages.md:378`. |
| V10 | v0.9.11 policy "Same version from multiple sources → No conflict — Silently deduplicate" and the 2026-03-24 "inviolable contracts" ruling | `design_docs/implemented/v0_9_11/m-pkg-resolver-direct-wins.md` (policy table + Design Decision block) — this doc extends the former with a source rule and reaffirms the latter. |
| V11 | No source-precedence logic exists today | `grep -rn "precedence" internal/pkg/*.go cmd/ailang/pkg_*.go` (non-test) → one hit: the `warnShadowedPathDeps` comment saying it "never changes precedence". |
| V12 | The queued replace-feature has no doc yet | Backlog row + `design_docs/planned/ailang-core-triage/local-dep-replace-override.md` (triage artifact, Recommend: design-doc, "no doc covers local overrides"). |
| V13 | Conflict error text used in Examples is the live format | `VersionConflictError.Error()` in `resolver.go` (read; V2 transcript matches). |
| V14 | All four resolver consumers route through `ResolveDependenciesFrom` | grep: `pkg_commands.go:517` (lock), `lockcheck.go:33` via `ResolveLockFrom` (lock --check + publish smoke staging `publish_stage.go:31`), `cmd/registry-validator/validate.go:58` — hence the chokepoint covers them all. |
| V15 | Tooling note | No Go toolchain in this session, so the reproduction ran against the prebuilt v0.60.6 `ailang` binary with a local mock registry instead of a scratch `go test`; the binary is built from this tree's HEAD (74d34c5d, `ailang version` = v0.60.6) whose resolver matches the cited code, so V1–V3 are live evidence for HEAD, not just v0.56.1. The scratch Go test used during drafting was deleted; Phase 1 fixtures formalize it. |

## Future Work

- The Go-style `replace` / local-override surface (backlog, D3) — this doc's D2 error message will gain a third suggestion pointing at it if it ships
- `entry.Latest` fallback in the fromRegistry path→registry conversion (see Non-Goals) — propose a backlog row: legacy tarballs should resolve to a version pinned at publish time, not registry latest
- Upgrade-chain tooling (the never-shipped v0.9.11 follow-up): automating the "republish dependents against the new version" path that D2 still requires

---

**Document created**: 2026-11-21
**Last updated**: 2026-11-21
