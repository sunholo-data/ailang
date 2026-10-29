### IFC cross-module summary and cache foundation (Refs #1134; Refs #1132 part 1)

- Export deterministic v2 interface summaries carrying parameter labels, canonical forbidden-label lists and a reserved applied-label field, declared and intrinsic return labels, and scoped Declassify authority.
- Preserve joined constant/variable labels, recursive intrinsic taint, and explicit clean metadata in compact JSON and gob interfaces.
- Imported call enforcement and CLI/REPL integration remain scheduled for M3; this partial sprint does not yet enforce cross-module sinks.

- Advance the compile cache from dev v7 to v8 (retaining the v7 CoreTI shared-DAG encoding). Older entries recompile once from source; authenticated legacy schemas and incomplete/malformed v2 summaries are rejected. Library summary changes invalidate cached dependents.
- Package identity is unchanged for unlabelled exports: the v2 interface hash ignores the iface JSON schema version and hashes an export's IFC summary only when it carries labels. A package without labels keeps its `ifacev2:` hash across this release, so publishers and the registry validator need not upgrade in lockstep; a label change still changes the identity.
