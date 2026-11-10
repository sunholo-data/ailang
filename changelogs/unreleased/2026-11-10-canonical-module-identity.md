### Fixed — A package module imported two ways was compiled, and its module-level code run, twice (2026-11-10)

A module of a package with `module_prefix` could be loaded under two identities in one build: once
under the prefix spelling its own package's files write (`docparse/types/document`) and once under
the published path a consumer writes (`pkg/sunholo/ailang_parse/types/document`). The loader cached
modules by the spelling of the import, so the same file was parsed, type-checked, given two
interfaces, and evaluated twice — every module-level binding ran twice at run time (measured on a
fixture: its init log printed twice). Inside a package, the bare self-import
(`import sunholo/linkedin/types`) was a third spelling of the same file, and a package without an
`[exports]` list accepted still more (`pkg/acme/shapes/shp/types/shape`).

Every spelling that resolves to one package file now maps to ONE identity per build, derived from the
file it resolves to, not from the spelling: a dependency's module is named by its published
`pkg/<vendor>/<name>/<module>` path, and the root package's own module keeps the spelling the package
itself writes (`<prefix>/<module>`, or `<vendor>/<name>/<module>` without a prefix) — the ID it
already had, so diagnostics for a package checked in its own repository do not change. The loader
keys its cache by that identity, rewrites the import to it, and keeps the author's spelling for
diagnostics. Spellings the resolver rejects (an unexported module, a missing file) fail as before.

### Changed — Imports resolve only within the importing package's declared scope (2026-11-10)

An import is resolved in the scope of the package that wrote it: that package, the packages it
declares in `[dependencies]`, and std. Before, a `module_prefix` spelling was looked up in one
build-wide table and a failed lookup was retried in the root package's scope, so a package could load
ANOTHER package's module that shared its prefix, a module of a package it never declared, or even a
file of the root package that depends on it — and a prefixed package that was only a transitive
dependency could not resolve its own imports at all. These are now errors (none occurs in the
downstream corpora or the cached registry packages; the one real case of a dependency importing the
root package — a motoko extension — is a warning):

- `LDR006: package aaa/a imports "cc/x" from package eee/e, which it does not declare; add "eee/e" to
  [dependencies] in aaa/a's ailang.toml, or fix the import` — also for a `pkg/` path into an
  undeclared package. Fix: declare the dependency, or fix the import. A dependency importing the ROOT
  package still resolves, with a warning: `WARNING LDR006 (warning): package aaa/a imports "ap/util",
  which is a module of the root package acme/app; a dependency importing the package that depends on it
  is allowed for now but is planned to become an error`.
- `LDR005: package acme/app imports "sh/x", but its dependencies aaa/one and zzz/two all provide it under
  module_prefix "sh"; import the one you mean by its published path (pkg/aaa/one/x)`. Fix: import it by
  its `pkg/<vendor>/<name>/…` path.
- `LDR001: module pkg/ccc/c/x (imported as cc/x), imported by package aaa/a, was not found in its scope
  (aaa/a itself and the packages it declares): …` — a dependency's missing module fails in its own
  scope instead of silently resolving somewhere else.
- `LDR001: import "cc/Init" names "Init.ail" in package ccc/c, but the file on disk is spelled
  "init.ail"; module paths are case-sensitive` — the same error on macOS as the "not found" on Linux.
  Fix: match the file's case.
- `LDR003: module file … is loaded under two identities, A and B; import it by one spelling` — one
  file reached by two paths no resolver unifies (a symlink into a package). Fix: import it one way.

A root package's own file imported by its path and by its package name (`import util` and
`import acme/app/util`) is one module now, initialised once.

### Changed — Module diagnostics name the canonical module and what the author wrote (2026-11-10)

IMP010: `symbol 'Triangle' not exported by 'pkg/acme/shapes/types/shape (imported as
shp/types/shape)'`. LDR002 cycles show each rewritten edge (`… -> pkg/acme/shapes/cyc/b (imported as
shp/cyc/b) -> …`), including how the first module was imported, and no longer print the cycle twice. Type and effect errors in a dependency module
name its declared path too (`type error in pkg/acme/shapes/ops/area (module shp/ops/area)`).

Side effect of the identity fix: a dependency's module reached through its `module_prefix` spelling is
no longer judged WHOLE against the ROOT package's `[effects].max`. The root's ceiling still bounds a
dependency wherever the root's code references it — a root capped at `[IO]` that calls, returns or
stores a dependency's FS or Net function is rejected under either spelling, before anything runs.
Interface hashes (`interface_hash` v2), `ailang iface` output, stdlib goldens and the compile-cache key
format are unchanged. This is M1b of
[m-v1-nominal-type-identity](design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md): module-qualified
type identity (M2) would otherwise have rejected valid programs that met the same type through both
spellings (`Block vs Block`, 14 of docparse's 44 files in the M0 spike).
