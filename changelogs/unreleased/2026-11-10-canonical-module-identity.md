### Fixed — A package module imported two ways was compiled, and its module-level code run, twice (2026-11-10)

A module of a package with `module_prefix` could be loaded under two identities in one build: once
under the prefix spelling its own package's files write (`docparse/types/document`) and once under
the published path a consumer writes (`pkg/sunholo/ailang_parse/types/document`). The loader cached
modules by the spelling of the import, so the same file was parsed, type-checked, given two
interfaces, and evaluated twice — every module-level binding ran twice at run time (measured on a
fixture: its init log printed twice). Inside a package, the bare self-import
(`import sunholo/linkedin/types`) was a third spelling of the same file.

Every spelling the package resolver answers now maps to ONE identity per build: a dependency's module
is named by its published `pkg/<vendor>/<name>/<module>` path, and the root package's own module keeps
the spelling the package itself writes (`<prefix>/<module>`, or `<vendor>/<name>/<module>` without a
prefix) — the ID it already had, so diagnostics for a package checked in its own repository do not
change. The loader keys its cache by that identity, rewrites the import to it, and keeps the author's
spelling for diagnostics. An IMP010 now names both: `symbol 'Triangle' not exported by
'pkg/acme/shapes/types/shape' (imported as 'shp/types/shape')`. Spellings the resolver rejects (an
unexported module, a missing file) are untouched and fail exactly as before.

Side effect: a dependency's module reached through its `module_prefix` spelling is no longer judged
against the ROOT package's `[effects].max` — that check applies to the root's own modules, and the
prefix spelling had made dependency modules look like the root's. Interface hashes (`interface_hash`
v2), `ailang iface` output, stdlib goldens and the compile-cache key format are unchanged.

This is M1b of [m-v1-nominal-type-identity](design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md):
module-qualified type identity (M2) would otherwise have rejected valid programs that met the same
type through both spellings (`Block vs Block`, 14 of docparse's 44 files in the M0 spike).
