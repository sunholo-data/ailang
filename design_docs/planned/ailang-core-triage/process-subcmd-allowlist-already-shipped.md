# Per-subcommand process allowlist requested (git push --force leak in Daneel's audit trail)

- **Date**: 2026-09-15
- **Class**: already-covered
- **Recommend**: duplicate-of `design_docs/implemented/v0_38_0/m-process-subcmd-allowlist.md`
- **Searched**: `find design_docs -iname "*subcmd*"`, `grep -ril "allowlist" design_docs`, `grep execProcess examples/`, live run of `examples/runnable/process_subcmd_allowlist.ail`
- **Estimate**: n/a (no code change)

The report measures v0.36.0, but M-PROCESS-SUBCMD shipped in v0.38.0. The design doc
`design_docs/implemented/v0_38_0/m-process-subcmd-allowlist.md` now says
**"✅ Implemented (v0.38.0 target, 2026-09-11)"** with the exact `cmd:sub[:sub…]` syntax
the requester proposes (their first-choice shape, `git:status,git:log,git:commit,git:push`,
matches the shipped grammar verbatim) and names Daneel as consumer-on-record (#1137).
Verified live on the current build, not just read:
`ailang run --caps IO,Process --process-allowlist "echo:hello" examples/runnable/process_subcmd_allowlist.ail`
→ `allowed: hello world` / `refused: NotAllowed(echo bye)`; an unlisted entry refuses with
`NotAllowed(echo)`. `cmd/ailang/help.go` (`--process-allowlist` help text) now documents
"cmd:sub narrows to a subcommand chain (git:status,gh:pr:list)", an e2e test exists at
`cmd/ailang/process_subcmd_e2e_test.go`, and the runnable example is self-documenting.
The doc Discoverability complaint in the report is also already addressed on HEAD (CLI
reference + example + updated Status line).

No new work warranted. Reply to the requester pointing at the doc, the example, and the
upgrade path (v0.36.0 → v0.38.0+); their `git:status,git:log,git:commit,git:push` line
should work as-written. If they want to re-file anything, the only residual gap on the
doc itself is the NOT-built items: denylist flag and Phase 2 signature-level scoping.
