# Sprint Plan: M-FLEET-INDEX-REGENERATE

## Summary

Add `ailang mission rotate-log <name> --index-only [--stream log|status]` and preserve numbered orphan rows and hand-written blockquotes whenever either mode regenerates an index. One shared index writer addresses both fleet tickets without adding a command verb or changing normal rotation's structural-heading safety check.

**Iteration:** fleet 45, planner role. **Base inspected:** `37d6641b308a3e8ffb4d5a2023ba4faa89a5254a`. **Date:** 2026-10-17.
**Tickets:** `mission:index-regenerate-needs-rotatable-archive` (T1); `rotate-log:index-regen-drops-orphan-rows` (T2, ailang#1306 half 2 only).
**Design authority:** controller directive D1–D5 in this task. No disagreement or measured reason to reopen those decisions.
**Milestones:** one, M1. These changes share the index writer, fixtures, and CLI dispatch; separating them would leave either preservation or the index-only path incomplete.
**Duration:** one 75–90 minute executor run, including validation. **Estimate:** 180 implementation + 240 test + 30 documentation LOC = 450 added/changed LOC, excluding mechanically moved lines. **Risk:** medium (history preservation and stream path selection).

This is a plan, not implementation or gate evidence. The planner produces only this file and the sprint JSON; the controller commits and routes execution. No messages, git writes, code edits, or real mission-document writes are part of planning.

## Current status and reuse

- `internal/mission/rotate.go` is 339 lines. `RotateLog` rejects a live file with no parsed entries before reading its archive, refuses interleaved structural headings, and then rewrites log, archive, and index. Its inline index writer currently discards existing index-only content.
- `parseLog`, `entryHeadingRe`, `statusHeadingRe`, `statusNoteRe`, `noteHeadingRe`, `indexLine`, `strayHeadings`, and `writeAtomic` already provide the grammar, rendering, safety check, and atomic replacement. Reuse them; do not invent a second heading grammar or writer.
- Existing tests cover cumulative indexing, rune-safe title truncation, status suffixes, successful rotation and structural refusal. `cmd/ailang/mission_rotate_test.go` already tests flag parsing and registry-origin versus external-workdir targeting, so CLI tests are required here too.
- `cmd/ailang/mission_cmd.go` is already 506 lines. Move its rotation handler into `cmd/ailang/mission_rotate.go` while updating help in the original file; remove its now-unused `strconv` import. Keep the existing registry resolver and target selection behavior.
- Recent directly comparable change `e7628b05e` (2026-10-05) added 238/deleted 34 lines across registry resolution, handler, and 184 lines of CLI tests. There are no commits to the inspected rotation files in the last seven days; no measured LOC/day rate is available. The 90-minute estimate is task-based and includes buffer, not a claim of measured throughput.
- **Registry reuse, M1: `none`.** This is a Go-internal CLI/file-preservation repair, not a package-like AILANG capability. Registry package searches are inapplicable; no package candidates, dependencies, or messages are needed. Reuse the existing internal mission helpers and CLI test fixtures. No `.ail` files or teaching-prompt gate apply.

## Measured Motoko baseline (D3)

Measured now on disposable copies of the three Motoko log/index files, using the `entryHeadingRe` expression extracted from production `rotate.go`; titles below use the existing rune truncation and pipe escaping. This is heading/projection measurement, not a run of the future flag.

**Existing hand-index number set (39 numbers):**

```text
{0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39}
```

**Live + archive parsed number set, and expected regenerated set (39 numbers):**

```text
{0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39}
```

The live file has zero entries. The archive has **39**, not 40, numbered headings: it has no `## 6` heading. At archive line 524, iteration 7's title says `recovered iteration 6: item 5b had landed, but the mission record never did`. Do not invent row 6 or change the real archive to make the prose claim “0–39” literal. Both sets already agree.

Only these rendered titles differ; all other numbered rows match current `indexLine` output:

| Row | Hand-index title | Title regenerated from heading |
|---|---|---|
| 38 | the judge failed the controller, not the code, and it was right — row 16 lands, row 6s is done, measured and parked on how its design was approved | the judge failed the controller, not the code, and it was right [HARNESS] |
| 39 | the audit the ruling ordered found a reader class three rounds had missed; round-3 quorum blocked 3/3 on a new environment-dependence, row 6s parked on D-MOTOKO-P2-1 [HARNESS] | the audit the ruling ordered found a reader class three rounds had missed, and the fresh quorum blocked on a defect nobody had named either [HARNESS] |

Preserve this entire block verbatim, including its historically stale explanation; updating that prose is outside this sprint:

```markdown
> **Reset 2026-09-30 (`D-MOTOKO-RESET-1`).** Iterations 0–39 below are the fork era (ABI 2.2 →
> motoko main migration); their full entries are all in `motoko-mission-log-archive.md`, and the live
> log starts empty at iteration 40. Row numbers these entries cite are the ARCHIVED queue's — the live
> queue in `motoko-mission.md` starts at row 20. This note is hand-written: `rotate-log` cannot
> regenerate this index for motoko today (queue row 19), so the next regeneration may drop it.
```

Baseline SHA-256 (read-only originals, measured via their temporary copies):

| File in `design_docs/` | SHA-256 |
|---|---|
| `motoko-mission-log.md` | `e52c2ea2a15169874e23d239e59c7e8f2801bd0b167dea5aa28cf46a80d11887` |
| `motoko-mission-log-archive.md` | `1ae8cce1d39e47d1e9a283289947d9b8d8b46ec549f0ebbf5961ffbb43d7da6c` |
| `motoko-mission-index.md` | `22003abe0fbee232a93f490fff55c35a95cead1e378b49873bcfae91d81327e6` |

All three inputs decode as UTF-8. Executor acceptance is set equality, exact Reset block preservation, unchanged live/archive hashes, and valid regenerated UTF-8 on a **temporary copy**. Whole-index byte equality is expressly not acceptance: rows 38 and 39 must follow source headings.

## M1 — Regenerate indexes without rotating, preserving historical additions

**Dependencies:** none beyond the existing rotation implementation and this approved controller design. **Status:** pending. **Estimated LOC:** 450. **Example:** synthetic log/status/index fixtures and the copied Motoko acceptance case; no real mission documents or `.ail` examples are edited.

### Implementation contract

1. Extract the existing index rendering into `internal/mission/index.go`. Share derived log/archive/index paths and the entry merge/dedupe/render/write machinery between `RotateLog(logPath, keep)` and a small `RegenerateIndex(logPath)` entry point. Keep `RotateLog`'s public signature and normal archive/body handling. Both callers must use the same preservation-aware index writer and existing `writeAtomic`.
2. Read the existing index before replacement; a missing index or optional archive is ordinary first-use state. Surface other read/write errors instead of silently treating unreadable history as empty. For normal rotation, prepare the index input/preservation data before mutating live/archive files. This sprint does not introduce a transaction across the three normal-rotation outputs.
3. `RegenerateIndex` reads the selected live file and its archive, parses both without applying the rotation-only structural refusal, and writes only the derived index. A present live file with zero entries or a preamble is valid when its archive has entries. If both inputs have zero parsed records, return an actionable error before any write, even if an old index has orphan rows. Numbered entries and recognized notes count as parsed records, retaining today's parser semantics. Missing live input and genuine I/O failures still return errors.
4. Index-only never writes, normalizes, truncates, moves, or creates either source file. For a missing optional archive, it remains absent. It does not accept or use a keep count. Normal rotation still defaults to keep 20, requires keep >= 1, archives full older bodies and refuses interleaved structural sections in the live file before writing anything.
5. **Structural headings in index-only:** inspect both files using the existing heading grammar. Ignore unrecognized `## ` headings for indexing, including structural preamble headings and interleaved sections; do not turn them or their section text into an index row or title. Keep `parseLog`'s recognized notes and numbered headings unchanged. Count ignored headings across both files and print `non-entry headings ignored: <count>` when nonzero; print nothing extra when zero. A nonmatching heading inside an entry's stored body never contributes to `indexLine`, which uses only the parsed heading title. This permissive branch is exclusive to indexing, where no bodies move. Do not weaken `RotateLog`'s existing `strayHeadings` guard.
6. **Numbered orphan rows in both modes:** read raw table rows from the existing index, parsing only the first cell as a nonnegative iteration number (including 0). If no parsed numbered record from either source has that number, preserve that row's original text verbatim. Merge it with generated rows in descending numeric order, e.g. 183, orphan 182, 181. Preserve raw titles, escaped pipes, spacing, date and Unicode; do not re-render or truncate orphan rows. If duplicate orphan rows already exist, retain their original relative order at that numeric position. Diagnostics list each preserved number once, ascending, e.g. `orphan rows preserved: 182, 999`; no orphan diagnostic when none exist. Count preserved rows in `IndexEntries`. Use returned result metadata so the CLI prints diagnostics for both paths; the library does not print directly.
7. A source entry wins over every existing index row with its number: regenerate that row using `indexLine`, replacing hand-edited dates/titles. Preserve current numbered dedupe and date ordering and current rune-safe 150-character truncation/pipe escaping. Notes with Num -1 (`—`) retain source date+title dedupe and follow numbered rows. **Do not preserve orphan `—` rows**: these have no stable iteration identity; only notes present in source files are regenerated.
8. **Hand note in both modes:** capture every blockquote block between the existing generated prose and first iteration-table header. Preserve exact block text and order and reinsert after the generated prose, before the table header. Include quoted blank lines (`>`) within a block and retain separation between multiple blocks. Do not copy old table headers or arbitrary old prose; do not collect quotes from table content or records. Repeated runs must neither duplicate notes nor change preserved orphan text. Missing existing index means no preserved content.
9. Keep path naming unchanged: log mode reads `<name>-mission-log.md` + `-log-archive.md` and writes `<name>-mission-index.md`; status mode reads `<name>-mission-status-archive.md` + `-status-archive-old.md` and writes `<name>-mission-status-index.md`. Status and log indexes must remain separate.
10. Move `missionRotateLog` to `cmd/ailang/mission_rotate.go` and add boolean `--index-only` parsing plus an explicit `keepProvided` flag. Reject any explicit `--keep` with `--index-only`, including `--keep 20` and `--keep 0`, in either argument order, before registry load or filesystem writes. Preserve retired `--status`, unknown stream and missing-name error behavior. Resolve the registry/root once using today's shared/external mapping, then dispatch to regeneration or rotation. Index-only success prints its index path/count and preservation/ignored-heading diagnostics, without claiming entries were rotated or sources rewritten.
11. Update both general rotate-log help and handler usage/unknown-flag text to describe mutually exclusive `[--keep N | --index-only]` and `[--stream log|status]`. Explain default rotation keep 20 and index-only writing only the index. No new command verb, environment variable or fallback path.

### Files and boundaries

| Path | Planned change |
|---|---|
| `internal/mission/rotate.go` | Delegate index/path work, retain rotation safety/body behavior, extend result metadata |
| `internal/mission/index.go` | New shared preservation-aware index helpers and index-only entry point |
| `internal/mission/rotate_index_test.go` | New table-driven tests beside `rotate_test.go`, including temp-copy Motoko case |
| `internal/mission/rotate_test.go` | Existing normal rotation and refusal tests retained; only focused assertions if needed |
| `cmd/ailang/mission_rotate.go` | Extracted rotation handler with flag dispatch and diagnostics |
| `cmd/ailang/mission_cmd.go` | Remove extracted handler/unused import, update rotate-log usage |
| `cmd/ailang/mission_rotate_test.go` | Extend existing temporary registry fixtures and flag/output tests |
| `.claude/skills/mission-control/resources/gate-4-record.md` | Brief index-only recovery/preservation guidance next to existing rotation instructions |
| `.agents/skills/mission-control/resources/gate-4-record.md` | Identical resource edit |
| `changelogs/unreleased/2026-10-17-mission-index-regenerate.md` | New fragment with `### Added` and `### Fixed` sections |

Stay under 500 lines in `rotate.go` and the new `index.go`; placing new tests in their companion file avoids growing the existing 422-line `rotate_test.go`. Extracting the handler also brings the already 506-line `mission_cmd.go` below 500. Do not split unrelated functionality.

Gate-4 prerequisite was measured with `rg -n 'rotate-log'` on both resources: each prescribes `ailang mission rotate-log ${MISSION_NAME} --keep 20` at line 33, with registry-root guidance at lines 44–53. They are currently byte-identical. Therefore edit **both copies identically**, retaining the normal rotation prescription and root warning. Add the index-only recovery example with `AILANG_MISSION_REGISTRY` explicitly pointing to the intended record checkout, explain no `--keep`, and mention retained orphan rows/notes and ignored structural headings. These are future operating instructions, not authorization to run against a real mission file during the sprint.

Executor change scope is limited to the listed paths within `internal/mission/**`, `cmd/ailang/mission*`, `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, and `changelogs/**`. Although `design_docs/**` is broadly allowed by the directive, **no real mission log, archive, index, charter or other mission document may be modified**. The sprint JSON is this planner's explicitly required state artifact; no other state/config/driver/code paths are authorized. Do not seed files under `$HOME` or the real repository's `design_docs/` for tests.

### Test and mutation matrix

Every writable fixture lives in `t.TempDir()`. Reuse CLI helpers `writeSyntheticRepo`, `use`, `snapshot`, `synthWrite`, and the existing `captureStdout`. Use `t.Setenv` and temporary registry origins, and do not make these process-global CLI tests parallel. Table-drive both operation modes and both streams where applicable. Give each required arm a named subtest so each isolated controller mutation has an observable failure.

| Test arm | Fixture and assertion | Isolated mutation it must catch |
|---|---|---|
| `PreserveOrphanRow/{rotate,index-only}` | Source 181/183/184, index row 182 with unusual spacing, escaped pipe and Unicode; byte-exact row survives between 183 and 181; count includes it | Remove orphan-row merge |
| `PrintOrphanNumbers/{rotate,index-only}` (CLI) | Seed 182 and 999, capture output, require `orphan rows preserved: 182, 999`; no orphan output for an all-source index | Remove only orphan diagnostic printing |
| `PreserveHandNote/{rotate,index-only}` | Reset-like multiline block plus second quote block survives verbatim before table exactly once, including a quoted blank line | Drop preservation/reinsertion of quotes |
| `IndexOnlySourcesUnchanged/{log,status}` | Compare source bytes and sha256 before/after, use structurally interleaved source, verify archive file and live file unchanged; absent archive stays absent | Add any normal-rotation source write to index-only |
| `IndexOnlyEmptyLive` | Preamble-only live file and numbered archive entry produce complete index; source bytes unchanged | Reinstate live-only zero-entry rejection |
| `IndexOnlyEmptyBoth` | Empty/preamble-only sources, including an old index with a numbered orphan, error and all fixture bytes unchanged | Omit combined zero-record check |
| `RejectKeepWithIndexOnly` (CLI) | Both argument orders and explicit 20/0 fail with usage text before registry loading and writes; index-only without keep succeeds | Remove explicit-keep incompatibility check |
| `NormalRotationArchives` | Keep 1 moves full old bodies into archive and retains newest live, with a preamble; existing rotation regression tests remain green | Skip archive append or normal rotation dispatch |
| `NormalRotationRefusesStructure` | Interleaved live Backlog heading returns named refusal and leaves every fixture file unchanged | Remove or bypass normal stray-heading guard |
| `IndexOnlyIgnoresStructuralHeadings` | Both files include structural sections; especially status live/archive with multiple interleaved headings. Success, exact ignored count printed, no structural title/body in rows, recognized numbered/status-note entries retained | Reuse rotation refusal or render stray headings as rows or omit count |
| `SourceTitlesWinAndNotesDedupe` | Hand-edited existing numbered row is replaced; duplicate source notes dedupe by date+title, distinct notes remain, orphan `—` row removed | Preserve every old row indiscriminately or dedupe all notes by -1 |
| `StreamAndRootSelection` (CLI) | New index-only arms for shared registry origin, external workdir, log/status suffixes; decoy copies stay byte-identical | Write cwd/workdir for shared mission or collide status/log indexes |
| `PreservationIdempotent` | Run twice, including no archive and multiple quotes/orphans, require identical index bytes and no duplicate rows/notes | Append preserved content repeatedly |
| `MotokoSnapshot` | Read real three input files only to copy into `t.TempDir()`; derive expected hand number set and Reset block from input. Regenerate copied log: compare sets, exact block, source sha256, `utf8.Valid` and heading-derived titles | Lose history/note, mutate sources, byte-truncate title |

Keep existing format/combined-heading/UTF-8 tests. Add a long Unicode title in the synthetic case so UTF-8 validity remains meaningful even if future Motoko headings shorten. The Motoko fixture's measured base sets and rows 38/39 are recorded above; snapshot acceptance derives sets from its copied inputs rather than freezing future mission history to this base.

### Work sequence (day 1, one run)

1. Minutes 0–20: add named failing preservation and index-only tests using temporary fixtures; extend existing CLI flag tests. Confirm failures identify missing behavior, not environmental setup.
2. Minutes 20–55: extract shared index writer/path handling, implement preservation and `RegenerateIndex`, retain normal safety behavior; extract handler, implement mutually exclusive parsing and diagnostics.
3. Minutes 55–70: finish status/orphan-output/UTF-8/temp-copy acceptance cases; update help, mirrored Gate-4 resources and changelog fragment.
4. Minutes 70–90: run only the validation below, inspect allowed-file diff, record actual exit codes and any sandbox limitations. Controller independently performs isolated mutation checks and review; planner/executor do not self-approve.

### Acceptance criteria

- [ ] Existing verb accepts index-only for log and status, with no keep required; explicit keep plus index-only is a usage error before registry access. Correct target and suffix mapping remains covered.
- [ ] Index-only writes only its index; live/archive are byte-identical, an absent archive remains absent, zero live entries work, preambles/interleaved structure do not block, zero parsed records across both sources error without writes, and ignored structural headings are counted without becoming rows.
- [ ] Both modes share one index writer. Every numbered orphan row survives verbatim in numeric position and its number is printed; existing numbered entries regenerate from source; notes retain date+title dedupe and orphan `—` rows are not retained.
- [ ] Hand blockquote blocks remain verbatim between regenerated preamble and table. Repeated runs preserve content exactly once. Index rendering remains UTF-8-safe.
- [ ] Normal rotation still archives full bodies and refuses interleaved live structure without changing any files; preamble-only structural headings above entries still work.
- [ ] On temporary Motoko copies, regenerated number set equals the hand-index set, Reset block is exact, live/archive sha256 are unchanged, and generated index is valid UTF-8. At this base both sets are `{0–5, 7–39}` and only titles 38/39 change. Real mission files are untouched.
- [ ] Required named test arms, including CLI stdout and explicit-keep order cases, can detect their corresponding isolated mutations. New tests use only temporary writable fixtures.
- [ ] Only listed implementation/documentation paths change; `rotate.go` and `index.go` remain below 500 lines, CLI help documents the flag, both Gate-4 resource edits are identical, and the dated fragment has `### Added` and `### Fixed`.
- [ ] Required gates and focused new tests below have recorded outcomes. Sandbox-dependent failures are **UNINFORMATIVE UNDER SANDBOX** and require controller evidence outside the sandbox before acceptance; they do not count as product failures or passes.

## Validation and evidence

The controller reports that all five gates were run at base `37d6641b3`. This planner does not rerun or independently certify those results. Executor runs exactly these gates plus the new focused tests; no full-suite or additional make gates are prescribed:

```bash
go build ./cmd/ailang ./internal/mission/...
go test ./internal/mission/ -count=1
go test ./cmd/ailang/ -run 'Mission|Rotate' -count=1
gofmt -l internal/mission cmd/ailang
go vet ./internal/mission/
```

Build, test and vet require actual exit 0; gofmt also requires no listed files. Focused new tests may use the named test/subtest filters above during development and isolated controller mutation checks. Each mutation must make its named arm fail while the restored final implementation passes; no additional persistent mutation artifacts are delivered. Do not use destructive git operations to apply or restore mutations.

Record command, exit code and relevant output. A socket-bind, external-path permission, or other sandbox-only failure is **UNINFORMATIVE UNDER SANDBOX**; even a sandbox run cannot replace required controller validation evidence. Leave such evidence pending for the controller rather than claiming acceptance or adding workarounds. No acceptance command writes a fixture under `$HOME` or the real `design_docs/`; CLI invocations in tests must resolve a `t.TempDir()` registry.

## Risks and completion

The main risks are losing raw orphan formatting, confusing preamble quotes with table content, changing status path suffixes, or weakening the normal safety refusal while sharing code. Raw-row preservation, source-priority tests, separate mode dispatch and the mutation matrix address these. File errors must surface; there is no silent unreadable-history fallback. Multi-file normal rotation is still nontransactional, as today; no broader storage redesign belongs here.

No open design decisions remain. The controller owns commits, implementation authorization, independent evaluation and ticket disposition; link #1306 as a reference because this sprint addresses only half 2, not an automatic whole-issue closure.
