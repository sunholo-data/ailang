# ailang check on a package file alone fails MOD010; path-dep syntax "undocumented"

- **Date**: 2026-09-16
- **Class**: feature (item 1) + already-covered (item 2) — one lane-friction report, two items, verdicts split below
- **Recommend**: design-doc (item 1); item 2 is duplicate-of `docs/docs/guides/packages.md`
- **Searched**: `MOD010`, `package context`, `packageSearchDir`, `FindManifest`, `m-dx-package-check`, `path dependency`, `ailang.toml` (design_docs/, docs/docs/guides/, internal/pipeline/, internal/check/, cmd/ailang/)
- **Source**: ailang_only lane friction, task-a0a96c0e / sunholo-data/ailang-packages#64 (no-shell agent, pi `ailang_check` tool; evidence: REPORT.md in PR, Cloud Logging `ailang-agent-executor-pi-bk9wd`)

## Item 1 — `ailang check packages/test-pkg/hello.ail` fails MOD010 with no package context → design-doc

**The mechanism.** Single-file `ailang check` (`cmd/ailang/check.go`, `checkFile`) runs the
pipeline with no package awareness and MOD010 strict. MOD010 itself fires in
`internal/pipeline/pipeline_module.go` (`validateModulePath`), which compares the declared
`module sunholo/test_pkg/hello` against `loader.CanonicalModuleID(modID)` derived from the
file's path (`hello` when invoked from the file's directory). Meanwhile the pipeline *already*
anchors manifest discovery at the entry file's directory — `internal/pipeline/package_resolver.go`
(`packageSearchDir` → `pkg.FindManifest` walks upward, fixing ailang#671) — but only for import
resolution, not for module-path validation. Package mode relaxes MOD010 on the explicit
assumption that "the manifest validates module names" (`internal/check/package.go:117`), but
single-file check never reaches that branch because it never consults the manifest. The pi
`ailang_check` tool (`internal/executor/pi/profile_assets/ailang-lsp-lite.ts`) inherits the
strictness: it shells out to bare `ailang check <basename>` with cwd = the file's dir, so a
no-shell agent has exactly one path, and it dead-ends.

**Why not direct-fix.** The report's remedy ("locate the nearest ailang.toml and check within
that package") is one of at least three acceptable designs, and the choice changes a gate's
contract:

1. **Validate against the manifest mapping** — when FindManifest locates a package and the file
   maps to an exported module (or via `module_prefix`, cf. `resolveModuleToFile` in
   `internal/pkg/discover.go`), accept the declared module if it matches the package mapping.
   Strictest option; also *catches* wrong module names that `--package` currently tolerates.
2. **Relax MOD010 on discovered package context** — mirror package mode's
   `RelaxModules: true` whenever a manifest is found above the file. Simplest, but silently
   weakens a gate that currently fires for genuinely misplaced files.
3. **Error-message routing** — keep strict MOD010 but detect the package-file case in
   `validateModulePath`'s error and point at `ailang check --package` / the manifest. Cheapest;
   doesn't actually unblock the no-shell agent.

Option 2 is what the reporter asked for; option 1 is arguably the right one. That disagreement
plus a MOD010 contract change (AGENTS.md: `internal/` semantics-sensitive) puts this at row 3/4
→ design-doc. Note there is also prior art to reconcile: `m-dx-package-check.md` (implemented
v0.10.0) added `ailang check --package` after the docparse/billing lane hit the *cross-module*
flavour of this same friction — the design doc should position auto-discovery against that
existing explicit flag rather than replacing it silently.

## Item 2 — path-dependency manifest syntax "undocumented in-repo" → already-covered

The premise is wrong: the syntax IS documented in-repo, beside the registry form, in
`docs/docs/guides/packages.md` — an example manifest with
`"sunholo/json" = { path = "../json" }` (line 68), a dedicated **Path Dependencies** section
(`"shared/utils" = { path = "../utils" }`, ~line 185) sitting directly above **Git Dependencies**
and **Registry Dependencies**, `ailang add --path` in the CLI table, and a **Workspace Pattern**
section (~line 357) on multi-package repos. `docs/docs/guides/package-publishing.md` additionally
covers path-dep semantics end-to-end (dev vs publish rewriting). The canonical grammar also lives
in `internal/pkg/manifest.go` (`Dependency` doc comment, ~line 158). The real gap is
discoverability by a no-shell agent that never searched the docs — which argues for
prompt/skill surfacing (a `use-ailang`/prompt-manager concern), not for new docs. No doc change
warranted; if item 1 goes to a design doc, a one-line pointer to packages.md's dependency
sections in that doc's error-message design (option 3 above) would close the discoverability gap
for free.
