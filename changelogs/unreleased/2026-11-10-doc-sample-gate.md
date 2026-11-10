### Added — doc-sample gate: every AILANG block in the docs passes `ailang check` or is tagged (2026-11-10)

`make check-doc-samples` (new, in `make ci` and the CI `checks` job) extracts every fenced
AILANG block from `docs/docs/**`, `README.md`, `docs/LIMITATIONS.md` and the active teaching
prompt's source file and checks it with a binary built from the same commit (`tools/docsamples`;
isolated from `AILANG_*` env and the compile cache; ~2–3 s for 453 blocks). Info-string tags
classify the blocks that are not complete programs: ```` ```ailang parse ```` (must parse),
`sketch` and `repl` (counted, not run), `error=CODE` (must fail with that code — an anti-example
that starts passing fails the gate), `file=path.ail` (multi-file examples). AILANG fenced as
`typescript` or untagged is a failure. Known failures are listed in the shrink-only
`scripts/doc_samples_baseline.txt`, keyed by page + content hash; CI refuses growth against
`origin/dev`, a listed block that passes, and a listed block that no longer exists.
`make test-check-doc-samples` holds the fixture controls. Design: M-V1-DOCS-FOR-1-0 §A.

### Changed — the site highlights AILANG; 135 blocks re-fenced or tagged (2026-11-10)

A Prism `ailang` grammar is registered (`docs/src/theme/`), so the 100 AILANG blocks fenced as
`typescript` for highlighting are now ```` ```ailang ```` and checked. Burn-down in the same change,
fence lines only: 114 mis-fenced blocks re-fenced, 21 anti-examples / notation / elided-body blocks
tagged; the baseline went 325 → 259 entries (208 site, 1 LIMITATIONS, 50 prompt).
ROADMAP `D-67` ruling 7 replaces "an inline block is a defect" (docs-mission clause 2, docs-sync
rule 2, coding-standards) with "an inline block must pass the gate or be tagged"; the tag grammar
lives in `.claude/rules/docs-review.md`.
