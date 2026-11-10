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

### Fixed — A module_prefix import could load another package's module; transitive prefixed packages failed to load (2026-11-10)

A `module_prefix` spelling was looked up in one build-wide table, so when two dependencies declared
the same prefix, one package's own `import px/types/t` could load the OTHER package's file, and a
prefixed package that was only a transitive dependency could not resolve its own prefix imports at all
(LDR001). A prefix spelling is now read in the importing package's scope: its own prefix first, then
its declared dependencies'; the build-wide table answers only a prefix no package in scope claims.

### Changed — Module diagnostics name the canonical module and what the author wrote (2026-11-10)

IMP010: `symbol 'Triangle' not exported by 'pkg/acme/shapes/types/shape (imported as
shp/types/shape)'`. LDR002 cycles show each rewritten edge (`… -> pkg/acme/shapes/cyc/b (imported as
shp/cyc/b) -> …`) and no longer print the cycle twice. Type and effect errors in a dependency module
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
