# `pkg quality` hides smoke output; smoke staging silently drops non-`assets/` data files

- **Date**: 2026-09-27
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `grep -rli smoke design_docs/`, `grep -rln _smoke prompts/`, `ls design_docs/planned/ailang-core-triage/`, plus direct code verification (terms: smoke, staging, assets, PUB015). No existing doc or prior triage row covers smoke-output surfacing or the staging file set; closest is `pub012-inline-tests-false-absence.md` (PUB012 only).
- **Estimate**: omitted — design-doc (see rubric rows 4/5; change spans ≥4 files and extends the versioned `ailang.package-quality/v1` JSON schema)

**Verified in code, both claims confirmed.**

(1) Output dropped: `RunSmokeInTempDir` (`internal/pkg/publish_validator.go`) already captures combined stdout+stderr, truncated to 8 KiB, into `SmokeResult.Output`. But `runAttestedChecks` in `cmd/ailang/pkg_quality.go` uses only `res.Passed` and `res.Duration`; `printQualityHuman` prints just `smoke: [attested] present %v passed %v`, and PUB015 (`internal/pkg/quality.go`, `BuildQualityReport`) carries the bare message `"_smoke.ail failed"`. `SmokeSection` has no `Output` field, so `--json` cannot carry it either. Notably `ailang publish` is not affected in the same way — the reporter's complaint is specifically the `pkg quality` surfacing path.

(2) Staging drop: `copyPackageContents`/`shouldStageFile` stage only `ailang.toml`, `ailang.lock`, `AGENT.md`, `_smoke.ail`, `*.ail`, and `assets/**`. Subtly worse than "fixtures/ never copied": the walk creates *every* non-excluded directory in the temp dir via `os.MkdirAll`, but skips the files inside them — so `fixtures/` exists but is empty, and `readFile("fixtures/a.json")` fails with file-not-found. Because (1) swallows the output, the user sees only `PUB015` with no reason. `ailang run` in the real package dir passes, so the two views silently diverge.

The reporter also correctly notes the docs gap: no prompt version mentions `_smoke.ail` or the staging file set in its Publishing section (`grep _smoke prompts/` → zero hits).

**Why design-doc, not direct-fix.** This is one report but a multi-part fix with real decisions:

- Part 1 spans `cmd/ailang/pkg_quality.go` (human print + wire `res.Output` through) *and* `internal/pkg/quality.go` (`SmokeSection` gains an `output` field) — already >1 file (row 5) — and extends the `ailang.package-quality/v1` schema that the dashboard/validator consume (row 4). Whether the validator side must learn the field, and truncation policy in the schema, deserve a written ruling.
- Part 2 offers at least three remedies the reporter himself alternates between ("document in prompt and help, **and/or** warn when a non-staged directory exists"): docs-only, a publisher-side warning, or surfacing a badge like "X files not staged". Choosing among them (and whether the warning belongs in `RunSmokeInTempDir`, which also backs `publish`, vs in `pkg quality` alone) is exactly row 3. A warn-on-unstaged-dir also needs a definition of "data directory" that doesn't false-positive on `.git`-adjacent or build dirs.
- There is a smaller latent question worth recording in the same doc: empty `fixtures/` dirs appearing in the smoke workspace is arguably a staging bug (either skip dir creation for unstaged dirs or stage them), independent of the warning.

**Suggested shape for the eventual doc**: (a) thread `SmokeResult.Output` into `SmokeSection` (optional `output` field, last-8KiB-truncated) and print it indented on failure in both human and JSON output; (b) decide warn-vs-badge for directories present-but-unstaged next to `_smoke.ail`; (c) add a Publishing-section note to the teaching prompt (`prompts/`, versioned file) and to `pkg quality --help`: "data files must live under `assets/` — only `assets/**` is staged for smoke and publish"; (d) fix or document the empty-dir artifact in `copyPackageContents`.
