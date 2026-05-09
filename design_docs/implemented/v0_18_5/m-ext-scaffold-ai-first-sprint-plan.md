# Sprint Plan: M-EXT-SCAFFOLD-AI-FIRST (v0.18.5)

**Sprint ID**: M-EXT-SCAFFOLD-AI-FIRST
**Target Version**: v0.18.5
**Design Doc**: [m-ext-scaffold-ai-first.md](m-ext-scaffold-ai-first.md)
**Estimated**: ~3-4 hours, ~250 LOC across implementation + tests + docs
**Risk Level**: Low (pure CLI scaffolding, no runtime/parser/typechecker changes)
**Created**: 2026-05-09

---

## Sprint Goal

Ship `ailang init motoko-extension --name <ns/pkg> --tools <csv> --effects <csv>` so an AI agent (or human) can scaffold a working motoko extension package in one command — collapsing today's ~30K-token doc-ingest tax to ~500 tokens of generated stubs.

**Out of scope** (deferred per design doc):
- Tier 2 generic `[extension_template]` block (separate sprint when 2nd host appears)
- Templates for non-motoko hosts
- Interactive TTY prompts (flag-only first; AI-friendly)
- Auto-publish (kept as a separate explicit step)

---

## Velocity Context

Today's velocity (recent v0.18.x sprints): typical 2-4 hour sprints landing 100-250 LOC each. This sprint sits squarely in that range. No surprises expected.

---

## Milestone Breakdown

3 milestones, sequential dependencies, ~3-4h total.

### M1 — Init type + flag parsing + validation

**Estimated**: 1.5 hours, ~80 LOC

**Description**: Register the new `motoko-extension` type in `ailang init`'s dispatch. Parse `--name`, `--tools`, `--effects` flags. Validate inputs (package name shape, effect names) with clear error messages.

**Files**:
- `cmd/ailang/init.go` (+10 LOC) — register new type, update help text
- `cmd/ailang/init_motoko_extension.go` (+70 LOC, new) — flag parsing, validation, dispatch

**Acceptance criteria**:
- `ailang init motoko-extension --help` shows the new type's flag list
- `ailang init motoko-extension --name foo/motoko_ext_bar --tools "T1,T2" --effects "FS,Process"` parses cleanly
- Invalid name (no `/`, missing `motoko_ext_` prefix) rejected with actionable error
- Invalid effect (not in canonical list) rejected with list of valid effects
- `deriveShortName("foo/motoko_ext_bar")` returns `"bar"` (round-trip helper)
- `deriveOutputDir("foo/motoko_ext_bar")` returns `"packages/motoko-ext-bar"` (mirrors ailang-packages convention)
- Existing `ailang init package` and `ailang init web-app` continue working
- Unit tests for flag parsing + validation + derive helpers

**Dependencies**: none

---

### M2 — Templates + file generation

**Estimated**: 1.5 hours, ~120 LOC + golden files

**Description**: Go string-constant templates for the 4 files (ailang.toml, register.ail, types.ail, `<short>.ail` with full 8-hook ExtensionHooks no-op stub). Render with `text/template`, write to disk under derived output dir. Generate a brief README.md hint linking to the tutorial.

**Files**:
- `cmd/ailang/init_motoko_extension_templates.go` (+100 LOC, new) — Go string constants for 4 file templates
- `cmd/ailang/init_motoko_extension.go` (+20 LOC) — render + write implementation
- `cmd/ailang/testdata/init_motoko_extension/openkb/*.golden` (snapshot files)

**Acceptance criteria**:
- All 4 files generated under the correct output dir with correct content
- `ailang.toml` includes registry `motoko_ext_abi` dep (NOT `path = ...`), correct exports list, effects from CLI flag, `[stability] level = "experimental"`
- `register.ail` is the canonical `register_with_config` wrapper that calls `make_hooks(...)` from the `<short>` module
- `<short>.ail` has all 8 ExtensionHooks fields populated with no-op defaults (each tool from `--tools` listed in `provided_tools` array)
- `types.ail` has a placeholder type with a TODO comment
- README.md links to build tutorial + publishing guide
- Golden files committed; snapshot diff catches unintended template drift
- No file uses `path = "../..."` for any dep (would defeat the publishability acceptance)

**Dependencies**: M1 (need flags + output dir derivation)

---

### M3 — End-to-end tests + tutorial doc update

**Estimated**: 1 hour, ~50 LOC + docs

**Description**: Integration test that scaffolds a sample extension to a temp dir, then runs `ailang lock` + `ailang check` against the generated package and asserts both pass with zero modifications. Update `build-a-motoko-extension.md` so Step 1 leads with `ailang init motoko-extension`; existing manual scaffolding becomes an appendix.

**Files**:
- `cmd/ailang/init_motoko_extension_test.go` (+50 LOC, new) — table-driven flag-validation tests + end-to-end lock+check test
- `docs/docs/guides/build-a-motoko-extension.md` (-30/+10 LOC) — Step 1 collapses to one-liner; "Manual scaffolding" appendix appended

**Acceptance criteria**:
- Integration test scaffolds a sample extension and runs `ailang lock + ailang check` — both pass with zero edits to generated files
- All 4 PR-#8-style failure modes are structurally impossible from the generated output (verified via golden file inspection: no `path = "../.."`, no inline-in-host placement, name shape correct)
- Tutorial doc Step 1 leads with `ailang init motoko-extension`; manual scaffolding moved to appendix
- `make test` clean
- `make lint` clean

**Dependencies**: M2 (need real generated output to test)

---

## Day-by-Day Plan

**Single-day sprint (target: today, evening session)**

| Time | Milestone | Work |
|------|-----------|------|
| Hour 1 | M1 | Register init type, flag parsing, validation, derive helpers + tests |
| Hour 2-3 | M2 | Template constants for 4 files, render + write, golden files |
| Hour 4 | M3 | End-to-end lock+check test + tutorial doc update |

**Total: ~4 hours focused work**

---

## Success Metrics

- All 3 milestones pass acceptance criteria
- `make test` and `make lint` clean
- Generated package passes `ailang lock + ailang check` with zero edits (the load-bearing test)
- All four arniwesth/motoko_agent#8 failure modes structurally impossible from generated output
- Tutorial doc updated; new init command is the front-page UX
- Existing `ailang init package` / `ailang init web-app` regress-tested unchanged

**Post-sprint validation** (NOT a sprint milestone):
- AI-driven extension creation: ask claude-code or motoko itself to "add an extension that does X"; measure tokens used end-to-end vs the 30K baseline
- This validation can happen in a separate session once the sprint lands

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| ExtensionHooks contract gains a 9th field; templates emit only 8 → generated code stops compiling | Medium | M3's `ailang lock + ailang check` integration test catches this immediately. Golden files document the current expected shape |
| Template syntax bugs (Go text/template gotchas with backticks/braces) | Low | M2 uses raw Go string consts (backtick strings) where possible; complex variable interpolation tested in unit tests |
| User wants non-motoko hosts (premature demand for Tier 2) | Low | Doc explicitly scopes Tier 1 to motoko-extension only; design doc already names M-EXT-SCAFFOLD-GENERIC-TEMPLATES as the Tier 2 follow-up |
| Sprint underestimated due to test infrastructure complexity | Low | If M3 balloons, drop the end-to-end integration test (keep snapshot tests only). Snapshot tests alone catch template drift; the integration test is "nice-to-have" defense-in-depth |

---

## References

- Design doc: [m-ext-scaffold-ai-first.md](m-ext-scaffold-ai-first.md)
- Empirical evidence the friction matters: [arniwesth/motoko_agent#8](https://github.com/arniwesth/motoko_agent/pull/8)
- Canonical reference shape the scaffold mirrors: [motoko-ext-exa-search](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-exa-search)
- Complementary feature: [M-AILANG-EXT-REGISTRY-GEN (v0.17.1)](../implemented/v0_17_1/m-ailang-ext-registry-gen.md)

---

**Document created**: 2026-05-09
