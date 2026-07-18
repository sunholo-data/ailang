# Sprint Plan: M-PROMPT-FOOTGUNS — Prompt-Teaching Footguns → Compiler Diagnostics

**Design doc**: [`m-prompt-footguns-to-diagnostics.md`](m-prompt-footguns-to-diagnostics.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-PROMPT-FOOTGUNS.json` (gitignored)
**Target**: v0.30.0 · **Priority**: P1 · **Risk**: Low
**Total estimate**: **~1.25 days** (PRIMARY ~1.0d · ghost-close ~0.25d · verification/docs ~0.5d, buffered/overlapping)
**Verified against**: HEAD `v0.29.2-393-ga69194e43` (2026-07-18) — re-verified this iteration
**Related**: GH #399 (mission iteration 47 park/ratify report)

## Ratified Scope (Mark, 2026-07-18)

Adopts the design doc's PARK-NOTE verbatim. **Plan and implement ONLY:**

- **Part A (PRIMARY)** — wire dormant `MOD002` + add `PAR_MODULE_PLACEMENT`, with gemini's error-recovery state-isolation fix.
- **Part B (ghost-close)** — CI-gated runnable example as a durable regression guard.
- **Verification / docs** — CHANGELOG, error-code regen, backlog stub for the severed part, superseded-doc archival (controller, on landing).

**DROPPED — do NOT implement:** Part C / Phase 3 (primitive-field-access `TC_PRIMITIVE_FIELD_ACCESS_001` unifier diagnostic). It is **severed** to an extension-lane backlog doc (`m-diag-primitive-field-suggestions.md`). No unifier change, no new TC code, no `internal/types/primitive_field_access.go` this sprint. The design doc's Phase-3 sections remain in the doc as historical context but are NOT executed.

## Premise Re-Verification at HEAD (a69194e43, 2026-07-18)

The doc was verified 2026-07-17 at `c9fb89d32`; HEAD has since advanced 39 commits to `a69194e43`. All load-bearing premises re-checked live this iteration:

| Premise | Doc anchor | HEAD status | Verdict |
|---|---|---|---|
| Two-module file → opaque `PAR_NO_PREFIX_PARSE` cascade | Problem Statement / row 4 | Reproduced: `PAR_NO_PREFIX_PARSE at :3:1: unexpected token in expression: module` | ✅ REAL |
| Misplaced (non-first, single) module → same cascade | row 5 | Reproduced: `PAR_NO_PREFIX_PARSE at :2:1: … module` | ✅ REAL |
| `MOD002` defined + registered but DORMANT (no emission site) | `codes.go:67-68`, `:267` | Confirmed — only defs/registry/`codes_test.go:114`; zero emitters | ✅ Dormant, reuse |
| `PAR_MODULE_PLACEMENT` does not exist | Design Freeze | grep: only doc/mission-log mentions, no code | ✅ Free to add |
| `TC_PRIMITIVE_FIELD_ACCESS_001` does not exist | row 13 | grep: only doc/log mentions | ✅ Free (but DROPPED) |
| `parseTopLevelDecl` switch, no MODULE case | `parser_decl.go:208` | Confirmed at `parser_decl.go:209` | ✅ Fire site |
| `case lexer.IMPORT:` precedent to mirror | doc `:336` | Now at `parser_decl.go:328` (line **drifted -8**) | ✅ present |
| `ParseFile` leading-module branch (only setter) | `parser_file.go:48-51` | Confirmed `:48-51`; calls `parseModuleDecl()` at `:49` | ✅ |
| `parseModuleDecl()` standalone helper exists | `parser_file.go:89` | Confirmed `:89` | ✅ No extraction needed |
| `reportMisplacedImport` + `errCountBefore` truncation precedent | `parser_file.go:372-398` / `391-395` | Confirmed `:372`, truncation `:391-395` | ✅ Mirror target |
| Unifier TCon-vs-`TRecordOpen` fallthrough (Part C site) | `unification_core.go:346` | Confirmed `:346`; tagged-union gate `:343-344` | ✅ (Part C DROPPED) |
| Unifier records mirror (Part C site) | `unification_records.go:404-407` | Confirmed `:405-407` | ✅ (Part C DROPPED) |
| MOD011 stale "per file" comment | `codes.go:93` | Confirmed still `:93` (`multiple module declarations per file`) | ✅ Fix in passing |
| `split_map_join.ail` guard program compiles at HEAD | row 14 | `ailang check` → No errors found (`join(",",list)`/`map(f,list)` orders live-verified) | ✅ |
| `string_split.ail` does NOT guard the pipeline | row 15 | Exists; split+chars+recursion only, no map/join | ✅ New example not redundant |
| verify-examples is a real CI gate incl. manifest stats | rows 16 | `make/examples.mk:18-28` + `ci.yml:205-209`; stats block at `manifest.json:2619` (total: 187) | ✅ |
| Footgun contract test infra | row 12 | `internal/diag/footgun_fixtures_test.go` present; `arity_diagnostic_test.go` present | ✅ |

**Premise drift found: NONE material.** Only cosmetic line-number drift — the `case lexer.IMPORT:` precedent moved from `parser_decl.go:336` (doc) to `:328` (HEAD). All other anchors match. The two live repros still produce the identical opaque cascade, so Part A remains REAL; Part B remains a GHOST (guard-only). No premise refuted; safe to execute as ratified.

## Milestone Breakdown

### M1 — Part A: module diagnostics (PRIMARY, ~1.0d)

Wire `MOD002` (duplicate) and add `PAR_MODULE_PLACEMENT` (misplaced) as coded, fix-carrying parse diagnostics, killing the opaque cascade.

**Implementation (mirror `reportMisplacedImport`):**
1. Add `Parser` fields `seenModule bool` / `firstModulePos` / `firstModulePath` (`parser.go`).
2. `ParseFile`'s leading-module branch (`parser_file.go:48-51`) is the **ONLY** place that SETS those fields.
3. New `reportMisplacedModule()` in `parser_file.go` (mirror `:372-395`): `seenModule` true → `MOD002` duplicate via `NewSuggestionError`; else → `PAR_MODULE_PLACEMENT`. Consume-and-truncate via `parseModuleDecl()` + `errCountBefore`.
4. `case lexer.MODULE: return p.reportMisplacedModule()` in `parseTopLevelDecl` (`parser_decl.go:209`, beside IMPORT at `:328`).
5. **Gemini state-isolation fix (critical):** recovery path MUST NOT mutate `seenModule`/`firstModulePos`/`firstModulePath`. So a module-less file with two late modules emits `PAR_MODULE_PLACEMENT` for BOTH (second is never a false `MOD002`).
6. Fix stale MOD011 comment (`codes.go:93`).

**Acceptance:** exactly one `MOD002` (no cascade) for two-module file; one `PAR_MODULE_PLACEMENT` for misplaced; 3 modules → 2 errors; two-late-modules → both `PAR_MODULE_PLACEMENT`; leading-module file unaffected. Golden test `internal/parser/module_placement_test.go` + footgun contract row. `make test-imports` + parser suite green.

### M2 — Part B: ghost-close guard (~0.25d)

Add `examples/runnable/split_map_join.ail` (the verified-compiling program from the doc) as a CI regression guard; the embedded prompt already teaches split→map→join.

**Acceptance:** `ailang check` clean; manifest entry + statistics `total` bump (avoid the classic verify-examples-red-via-manifest-drift); `make verify-examples` green. Prompt-deletion candidates RECORDED (not deleted — prompt stays 2535 lines).

### M3 — Verification, docs, backlog severance (~0.5d)

Regenerate `dist/error_codes.json`, CHANGELOG, file the extension-lane backlog stub for the DROPPED Part C, note the severance in the plan/doc.

**Acceptance:** `make ci` green; both repro files show new diagnostics; `dist/error_codes.json` shows MOD002 emitted + `PAR_MODULE_PLACEMENT` registered; CHANGELOG v0.30.0 entry; `m-diag-primitive-field-suggestions.md` backlog stub filed (records both candidate enrichment routes; zero code change); superseded docs archived by controller on landing.

## Day-by-Day

| Day | Work |
|---|---|
| **Day 1** | M1 in full: parser fields + `reportMisplacedModule` + `case lexer.MODULE` + gemini state-isolation + MOD011 comment fix + golden tests + footgun row. Run `make test-imports` + parser suite. |
| **Day 2 AM** | M2 (guard example + manifest + verify-examples) + M1 hardening. Begin M3: regen error codes, CHANGELOG, backlog stub. |
| **Day 2 PM (buffer, ~0.25d)** | M3 finish: `make ci` green, repro-file confirmation, file-size check. |

Total: **~1.25 days**.

## Files

**New:** `examples/runnable/split_map_join.ail` (~15) · `internal/parser/module_placement_test.go` (~120) · `design_docs/planned/v0_30_0/m-diag-primitive-field-suggestions.md` (backlog stub, ~40)
**Modified:** `internal/parser/parser_decl.go` (+3) · `internal/parser/parser_file.go` (+50) · `internal/parser/parser.go` (+3) · `internal/errors/codes.go` (MOD011 comment) · `internal/diag/footgun_fixtures_test.go` (+40) · `examples/manifest.json` (entry + stats) · `dist/error_codes.json` (regen) · `CHANGELOG.md`

**NOT touched (Part C severed):** `internal/types/unification_core.go`, `internal/types/unification_records.go`, `internal/types/primitive_field_access.go` (not created).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Recovery flips `seenModule` → false MOD002 on 2nd late module | Med | Explicit state-isolation rule (gemini fix) + golden test asserting two-late-modules → both `PAR_MODULE_PLACEMENT` |
| Cascade-truncation swallows a genuine error in a malformed 2nd module | Low | Mirror `errCountBefore` exactly; golden test with malformed duplicate (`module a-b`) asserting one placement error |
| Manifest statistics drift → verify-examples red | Med | Explicit task to bump `manifest.json:2619` statistics `total` alongside the entry |
| Line-number drift vs doc anchors | Low | Anchors re-verified at HEAD in this plan; `case lexer.IMPORT:` is `:328` not `:336` |

## Handoff

- Sprint JSON populated with 3 real milestones (no placeholders), live-verified file anchors, dropped-scope block, and non-goals.
- **SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-prompt-footguns-sprint-plan.md`
- **SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-PROMPT-FOOTGUNS.json`
- Next role: **sprint-executor** (Part A + Part B + verification/docs only; Part C is out).
