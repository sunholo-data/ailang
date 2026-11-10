### Added — doc-sample gate: every AILANG sample in the docs passes `ailang check` or is tagged (2026-11-10)

`make check-doc-samples` (new; in `make ci`, `make ci-quick` and so the dev pre-push hook, and the
CI `checks` job) checks, with a binary built from the same commit (`tools/docsamples`; isolated
from `AILANG_*` env and the compile cache; ~3 s for 536 samples):

- every fenced AILANG block in `docs/docs/**`, `README.md`, `docs/LIMITATIONS.md` and the active
  teaching prompt's source file;
- every `.ail` file the docs import with raw-loader (77 files: `make verify-examples` covers only
  `examples/runnable/`), and AILANG embedded as a template literal in `docs/src` (the landing page).

Info-string tags classify the blocks that are not complete programs: ```` ```ailang parse ````
(must parse), `error=CODE` (the FIRST error must be exactly that code; the gate's own classes are
refused), `file=path.ail` (multi-file examples), `sketch` / `repl` (not run, and listed in the
shrink-only `scripts/doc_samples_unchecked.txt`; a new one needs a `# reviewed:` marker). A block
that defines `main` must carry its own `module` line. Known failures are listed in the shrink-only
`scripts/doc_samples_baseline.txt` (page + content hash + first code). Both lists shrink **as sets**
against the PR base or the push's previous tip: any new line fails, as does a listed block that
passes, vanishes or fails with a different code. Structural defects — AILANG fenced as
`typescript`/`text`/untagged (keyword hints plus an `ailang fmt` parse probe), unknown tags, fences
that never close or swallow the next block, `<CodeBlock>` rendering an `.ail` file as another
language, unresolvable imports — fail and cannot be listed; the one exception (a stray fence in the
frozen prompt v0.16.12) is named in `tools/docsamples/structural.go`. Design: M-V1-DOCS-FOR-1-0 §A.

### Changed — the site highlights AILANG; 213 samples re-fenced or tagged (2026-11-10)

A Prism `ailang` grammar is registered (`docs/src/theme/`). Mechanical burn-down, no sample content
changed: 119 fenced blocks re-fenced (`typescript`/untagged/`text` → `ailang`, `ailang repl`,
`ailang sketch`; CLI output → `text`), 73 `<CodeBlock language="typescript">` imports switched to
`ailang`, 21 anti-examples / notation / elided-body blocks tagged. Baseline: 283 entries (218 site
blocks, 8 imported files, 1 landing-page literal, 1 LIMITATIONS, 55 prompt; 30 of them MOD014, a
`main` without a `module` line). ROADMAP `D-67` ruling 7 replaces "an inline block is a defect"
(docs-mission clause 2, docs-sync rule 2, coding-standards) with "an inline block must pass the gate
or be tagged"; the grammar lives in `.claude/rules/docs-review.md`.
