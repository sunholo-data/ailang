---
name: M-PKG-INTRA-PACKAGE-IMPORTS
description: Intra-package imports must work without a lock file, and bare canonical paths inside the owning package should resolve to siblings
type: planned
---

# M-PKG-INTRA-PACKAGE-IMPORTS: intra-package imports for multi-module packages

**Status**: IMPLEMENTED
**Target**: v0.19.0 (small, additive; co-locate with M-NET-BINARY-BODIES and M-COORDINATOR-INBOX-WILDCARDS)
**Priority**: P1 — silently constrains package authoring; every multi-module package author hits it on day one and works around it by inlining types
**Estimated**: ~120 LOC core + ~200 LOC tests + docs (~4–6h)
**Dependencies**: none (additive; M-DX-RELIMPORT shipped Mar 2026, this fixes the gap that feature was supposed to close)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-12
**Source**: agent inbox message [913c0aa4](msg_20260512_144239_913c0aa4) from `demos/linkedin`
— `ailang publish sunholo/linkedin@0.1.0` fails because `auth.ail` cannot
import `sunholo/linkedin/types` even though `types.ail` is a sibling in
the same tarball. User worked around it by inlining the type definitions
into every module. Inspection of the existing registry shows every published
multi-module package (`gemini_live`, `gcp_auth`, `billing-*`, `firestore`,
`motoko-ext-*`) has had to do the same thing.

---

## Problem statement

A multi-module AILANG package looks like this:

```
packages/linkedin/
├── ailang.toml      # name = "sunholo/linkedin", exports = [types, auth, ...]
├── types.ail        # module sunholo/linkedin/types
├── auth.ail         # imports types
├── posts.ail        # imports types
└── comments.ail     # imports types
```

The author asks the obvious question: "How do I import `types` from `auth`?"
There are three documented forms, **and all three are broken in different
ways on dev (verified against v0.18.11)**:

| Form they try | What happens | Verdict |
|---|---|---|
| `import sunholo/linkedin/types` (the form that matches the module declaration) | LDR001 — falls through to project-relative resolution at [internal/loader/loader.go:220-224](../../../internal/loader/loader.go#L220-L224), which looks for `./sunholo/linkedin/types.ail` and finds nothing | unresolvable |
| `import ./types` (M-DX-RELIMPORT, [internal/loader/loader.go:140-143](../../../internal/loader/loader.go#L140-L143)) | Normalized to `pkg/sunholo/linkedin/types`, then routed through the `pkg/` branch at [internal/loader/loader.go:144-148](../../../internal/loader/loader.go#L144-L148), which fails with: `package import "pkg/sunholo/linkedin/types" requires ailang.toml and ailang.lock; run 'ailang init package' and 'ailang lock'` | "use relative imports" advice is wrong without a lock file |
| `import pkg/sunholo/linkedin/types` (canonical) | Same as `./types` — requires `ailang.lock` to exist | wrong dependency: self-references shouldn't need a lock file |

### Root cause

The `ModuleLoader.pkgLoader` field is only set when
[`tryLoadPackageResolver`](../../../internal/pipeline/pipeline_module.go#L89)
finds **both** `ailang.toml` and `ailang.lock`. When you're authoring a brand
new package, the lock file may not exist yet (or may be deliberately omitted
during validation). With no `pkgLoader`, even `pkg/<self>/...` imports fail.

`internal/pkg/loader.go:41-44` already has correct self-reference logic:

```go
// Self-reference: if the import matches the current package name,
// resolve against the root directory (intra-package imports).
if pl.isSelfReference(pkgName) {
    pkgDir = pl.rootDir
}
```

But that branch is only reached when the `ModuleLoader.pkgLoader` field is
set — and the registry validator's check step
([`cmd/registry-validator/main.go:482-489`](../../../cmd/registry-validator/main.go#L482-L489))
*does* generate an empty lock file before invoking `ailang check --package .`,
so this path *should* work for at least form 3. The LinkedIn report ("the
validator unpacks files … and apparently doesn't expose siblings to each
other as importable") strongly suggests it doesn't, but the failure mode I
reproduced locally is upstream of the validator — `ailang check --package .`
already fails on this layout without a lock. Either the validator's lock
generation is silently producing an unusable file, or there's a second bug
between lock generation and pkgLoader setup. **Both need to be unwound as
part of this milestone.**

### Why this matters

- **Every multi-module registry package today has worked around this** by
  inlining types into each module (verified by grepping the
  `sunholo-data/ailang-packages` working tree — no `.ail` source file in any
  package imports a sibling within the same package; only consumer-facing
  `AGENT.md` snippets do).
- **The natural code shape** for any non-trivial domain — `types.ail` +
  `service.ail` + `client.ail` — is unreachable in AILANG today. Authors
  either flatten into one big file or duplicate the type aliases.
- **M-DX-RELIMPORT (March 2026) was supposed to fix this.** Its design doc
  promised `import ./plan` for siblings. The feature shipped, but it depends
  on a working `pkgLoader`, which depends on a lock file, which the author
  doesn't have yet. So in practice the `./` form only works for *consumers*
  of an installed package, not the package's own modules.

---

## Proposal

Two changes, both small and orthogonal.

### Change 1: bootstrap a self-only `PackageLoader` from `ailang.toml` alone

Today, `tryLoadPackageResolver` requires `ailang.toml` AND `ailang.lock`.
Loosen it: if `ailang.toml` exists but `ailang.lock` doesn't (or is empty),
construct a `PackageLoader` with an empty lock file. The loader's
`isSelfReference` check
([`internal/pkg/loader.go:227-233`](../../../internal/pkg/loader.go#L227-L233))
only consults `ailang.toml`, so a lock-less loader can already resolve
self-references; it just can't resolve *external* dep imports — which is the
correct behavior for an un-locked package anyway (those should be a
`ailang.lock` missing error, not a generic LDR001).

Concretely, in
[`internal/pipeline/pipeline_module.go:89`](../../../internal/pipeline/pipeline_module.go#L89):

```go
// Before:
if pkgResolver := tryLoadPackageResolver("."); pkgResolver != nil {
    modLoader.SetPackageResolver(pkgResolver)
    ...
}

// After:
if pkgResolver := tryLoadPackageResolver("."); pkgResolver != nil {
    modLoader.SetPackageResolver(pkgResolver)
    ...
} else if pkgResolver := tryLoadSelfOnlyPackageResolver("."); pkgResolver != nil {
    // ailang.toml exists but no ailang.lock — wire a self-reference-only
    // resolver so intra-package imports work during authoring.
    modLoader.SetPackageResolver(pkgResolver)
}
```

`tryLoadSelfOnlyPackageResolver` constructs `pkg.NewPackageLoader(emptyLock, ".")`.
External `pkg/<other>/...` imports under this resolver fail with a clearer
message: "package `pkg/<other>` requires `ailang.lock`; run `ailang lock`".

### Change 2: route the bare canonical form (`import sunholo/linkedin/types`) through the self-reference path

The fact that `import sunholo/linkedin/types` — the form that visually
matches the module declaration — falls through to project-relative resolution
is a UX cliff. Three options, in order of preference:

**Option A (recommended): treat bare canonical paths whose first two segments
match the current package's name as self-references.**

In `loader.go`, before falling through to the modulePrefixMap branch at
line 190, check: if `ml.pkgLoader != nil` AND the bare path starts with
the current package's `<vendor>/<name>`, treat it as `pkg/<vendor>/<name>/...`
and route through the package resolver's self-reference path.

Cost: ~30 LOC + tests. No language change. Existing `pkg/` and `./` forms
keep working unchanged.

**Option B: deprecate the bare form and emit a helpful error.**

When `sunholo/linkedin/types` is unresolvable AND matches the current
package's name prefix, emit: `bare imports must use ./types or
pkg/sunholo/linkedin/types within the same package`.

Cost: ~10 LOC. Doesn't fix the UX cliff; just narrates it. Worth doing
regardless, as a fallback for cases where Option A's heuristic doesn't fire.

**Option C: do nothing and lean on `./`.**

Possible if M-DX-RELIMPORT's `./` form becomes universally documented as
*the* way to do this. But `./` doesn't appear in the published-package
ecosystem at all (`grep`), suggesting authors don't find it. And it
foregoes the chance to make module names round-trip with import paths,
which is an AILANG design value.

I recommend **A + B** together: route the bare form for the common case,
and keep B as a safety net for cases where the heuristic misfires (e.g.
a module declaration whose path doesn't match `ailang.toml`'s `name`).

---

## Acceptance criteria

1. **Authoring a fresh multi-module package compiles without `ailang lock`.**
   Test fixture: `internal/pkg/testdata/intra_pkg_fresh/` with `ailang.toml`,
   `types.ail`, `service.ail` (where `service.ail` imports `types`). Running
   `ailang check --package .` succeeds with no lock file present.

2. **All three import forms resolve identically.** For the same fixture:
   - `import sunholo/repro_pkg/types` (bare canonical) ✓
   - `import ./types` (relative) ✓
   - `import pkg/sunholo/repro_pkg/types` (canonical pkg/) ✓

3. **Registry validator publishes a multi-module package.** Manual repro
   against `https://ailang-registry-validator-mdpoxgrptq-ew.a.run.app`:
   construct LinkedIn's original cross-module layout (this morning's git
   history in `sunholo-data/ailang-packages`, per the inbox message) and
   confirm it now passes validation. **The validator may or may not need
   code changes** — verify whether Change 1 is sufficient once it lands, or
   whether the validator has its own broken lock-generation path.

4. **External `pkg/<other>/...` imports still require a lock.** Negative
   test: a package whose `service.ail` imports `pkg/sunholo/firestore/client`
   without `ailang lock` errors with a *better* message than today's
   ("requires ailang.lock; run `ailang lock`"), not a generic LDR001.

5. **No regression in existing registry packages.** Run `ailang check` over
   every published `sunholo/*` package — none of them use intra-package
   imports today (they inline), so this should be no-op verification, but
   it surfaces any accidental coupling between the new self-reference path
   and external-dep resolution.

---

## Out of scope

- **Documenting the canonical form across all import surfaces.** Once this
  ships, AILANG syntax docs, `ailang prompt`, and the bootstrap teaching
  prompts should be updated to show intra-package imports — but that's a
  docs task, not a language change. Track separately.
- **Refactoring published packages to drop the inline-types workaround.**
  Each package owner can revisit this once the constraint is lifted; we
  shouldn't bundle a multi-package refactor with the language fix.
- **The bigger "module name should imply file path" design question.**
  AILANG already chose `module <path>` as the canonical identity; this
  milestone just makes the package-author UX match that promise. Any
  further unification (e.g. autoderiving module names from filesystem
  paths) is a separate v1.x conversation.

---

## Risks

- **Heuristic for Option A misfires when `ailang.toml`'s `name` and the
  module declaration's prefix disagree.** Mitigated by the `_test.go`
  fixture covering the mismatch case, and by Option B's fallback emitting
  a clear "use ./ or pkg/" error.
- **Self-only resolver could mask "missing ailang.lock" errors that should
  block a build.** Mitigated by acceptance criterion 4 — external imports
  must still error clearly; only self-references get the lock-less path.
- **Validator may have its own bug independent of `ailang check`.**
  Acceptance criterion 3 forces us to validate end-to-end. If the validator
  needs a separate fix, scope it under this same milestone rather than
  splitting into two — the user's failure mode is "publish doesn't work,"
  not "check doesn't work."

---

## Implementation plan (rough)

1. **M1** (~1h): Add `tryLoadSelfOnlyPackageResolver` + fixture-based test
   in `internal/pipeline/`. No production wiring yet; just demonstrate the
   self-only resolver compiles and resolves siblings.

2. **M2** (~1.5h): Wire M1 into `pipeline_module.go:89` as a fallback.
   Run the existing test suite + `make verify-examples` + the new fixture.

3. **M3** (~1.5h): Option A — bare-canonical → self-reference routing in
   `internal/loader/loader.go` before the modulePrefixMap fallback. Includes
   the "module decl doesn't match ailang.toml name" Option B fallback.

4. **M4** (~1h): End-to-end validator test. Construct a multi-module tarball
   in a CLI test (`cmd/registry-validator/main_test.go`), POST to a local
   instance, assert validation passes.

5. **M5** (~30m): Docs — update `prompts/v0.19.md` (or current), syntax
   reference, and a new `examples/intra_package_imports.ail` showing the
   three forms working.

---

## References

- [LinkedIn inbox message 913c0aa4](msg_20260512_144239_913c0aa4) — original report
- [M-DX-RELIMPORT](../../implemented/v0_9_4/m-dx-relative-imports.md) — March 2026 feature this completes
- [internal/loader/loader.go:120-225](../../../internal/loader/loader.go#L120-L225) — module loader dispatch
- [internal/pkg/loader.go:29-55](../../../internal/pkg/loader.go#L29-L55) — package loader with self-reference logic
- [internal/pipeline/pipeline_module.go:85-96](../../../internal/pipeline/pipeline_module.go#L85-L96) — pkgLoader wiring
- [cmd/registry-validator/main.go:430-515](../../../cmd/registry-validator/main.go#L430-L515) — validator's check step + lock generation
