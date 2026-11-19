# Sprint plan: rotate-log legacy headings

Ticket: `rotate-log:legacy-heading-format`, stapledon mission iteration 40, 2026-10-20. Fleet iteration 51; sprint `M-FLEET-ROTATE-LOG-LEGACY-HEADINGS`.
Scope classification: harness semantics fix; this deliverable is planning only. The controller supplied fixed design decisions D1–D8; implementation requires the executor stage. No dispatch, implementation, git writes, or real-log edits in this planner stage.

## Goal and estimate

Recognise all stapledon records, preserve notes' neighbours during rotation, refuse unparsed preamble records, and make no-archive runs byte-preserving with honest output. Real rotations leave one archive notice.
One intentional milestone, one working day, approximately 60–90 minutes of executor work plus separate controller integration. Estimate **370 LOC**: 105 implementation/comment/output, 220 tests, 41 fixture, 4 changelog. This is a scoped estimate with buffer, not a measured Go velocity. Dependencies: controller-provided design and access to the pinned stapledon source for fixture generation. Risk: medium (history preservation).
Registry reuse: **none** for M1; existing Go mission parser/rotation/index infrastructure is the reuse target. No package-like AILANG capability or `.ail` module; registry package search is inapplicable. No new package, script, dependency, or AILANG example.

## Verification: established controller measurements M1–M6

These are controller measurements outside the sandbox at **origin/dev f0f7df204**, not planner reruns. The original ticket saw 40 headings but only 4 parsed entries, zero archived and orphan rows 0..36; the later pinned corpus contains 41 headings.

- **M1 — reproduction:** scratch stapledon log from `sunholo-data/stapledons-godot` origin/main **50d8d5ba**, 297,635 bytes and 41 `## ` headings. HEAD-built binary, `--keep 20`: rc 0, “5 of 5 entries kept”, “0 entries rotated out”, orphans 0..36. Live bytes grew **297635 → 297936**; a four-line archive notice appeared near line 1572, immediately above entry 37 and after all 36 legacy records swallowed into preamble.
- **M2 — root cause:** `internal/mission/rotate.go` recognises canonical/Iteration/ITERATION numbered headings, STATUS variants, and ALL-CAPS notes only after an entry starts. All preceding text becomes preamble. `strayHeadings` starts at the first parsed entry, so an unrecognised record in preamble silently escapes the guard.
- **M3 — corpus:** 41 headings = 5 canonical (37–41) + 36 date-first legacy (33 numbered, 3 notes). Numbered shape: `## YYYY-MM-DD[ (qualifier)]: iteration N, title`, including iteration 0 and `(afternoon)`/`(night)`. Notes: `## 2026-09-27: pre-iteration 0 (attended bootstrap)`, `## 2026-09-27 (late): armed, attended`, and `## 2026-10-02: attended, M2 journey core LANDED (sprint R1-M2-JOURNEY) [PRODUCT]`; the last sits between iterations 8 and 9. Iterations 10, 13, 15, 25 have no heading (orphan credit elsewhere).
- **M4 — notice side effect:** unconditional `pre + notice + kept` repeats notices already in preamble. v1 log has 8 copies (lines 25–60); fleet log records hand-collapse seven times (lines 212, 271, 385, 486, 670, 766). This is the ticket's false/duplicated-notice path.
- **M5 — preserve contracts:** `TestRotateLog_UnderThresholdStillWritesTheIndex` (:151) requires the index even without rotation; `TestRotateLog_PreambleHeadingsDoNotTripTheGuard` (:399) permits structural preamble headings. Real permitted examples: `## How to read this file`, v1/motoko template `## N — YYYY-MM-DD — <headline>`, docs status `## (Charter sections — moved back 2026-10-01)`, motoko status `## Archived 2026-09-30: fork-era charter (...)`.
- **M6 — base gates:** `go test ./internal/mission/` rc 0 (~27 s); `go test ./cmd/ailang/ -run 'Rotate|MissionRotate'` rc 0; `gofmt -l internal/mission cmd/ailang` empty. Sizes: rotate.go 276 lines, index.go 217, rotate_test.go approximately 440. The requested focused gates were reported green at base; rerun them after implementation, including changelog validation.

Local read-only review confirms both recognition consumers (`strayHeadings` and `ignoredIndexHeadings`) and the unconditional live write. No inspected test pins duplicate notices; existing tests stay unchanged. Add a companion `rotate_legacy_test.go` to keep test files small.

## Fixed design D1–D8

**D1 — legacy parsing and recognition.** Add the numbered pattern (or equivalent):
```text
^## (\d{4}-\d{2}-\d{2})(?:\s*\(([^)]*)\))?:\s+(?i:iteration)\s+(\d+)\b[,:]?\s*(.*)$
```
Map to `LogEntry{Num: N, Date, Title}`; title drops `iteration N, ` and retains a qualifier as `(afternoon) title`. Test the numbered form before the note fallback:
```text
^## (\d{4}-\d{2}-\d{2})(?:\s*\(([^)]*)\))?:\s+(.+)$
```
Fallback maps to `Num: -1`, same date, title `(qualifier) rest`. Both forms start records anywhere, including before any numbered entry. `pre-iteration 0` is a note. Recognise both forms in `strayHeadings` and `ignoredIndexHeadings`; neither is stray/ignored. Extend `entryHeadingRe`'s fleet-format comment and retain its “deliberately tolerant” rationale.

**D2 — stable neighbour order.** Before sorting, assign each numbered entry its Num key and each note the key of the nearest preceding numbered entry in original file order, or -1 before any numbered entry. Sort entry/key pairs stably by that key (do not mutate Num or derive keys inside the comparator). This preserves numbered ordering and notes' file order, including the October 2 note between 8 and 9. Update misleading rotation-note ordering comments as needed; index row sorting/format stays unchanged.

**D3 — no archives, no live write.** If the cut is zero, do not write the live log or archive; still prepare/write the index per M5. Report `LogAfter == LogBefore`. Only when `Archived == 0`, add one CLI line: `live log was not modified (no entries archived)`. Preserve the other output and command behaviour.

**D4 — exactly one generated notice on real rotation.** Strip every generated blockquote from preamble by matching its first line `> **Older entries are ARCHIVED.**` (followed by generated text) and following `> ` continuation lines, not fuzzy prose. Remove each complete generated four-line block, preserve surrounding prose verbatim, then emit one fresh notice. Do not clean notices during a D3 no-op.

**D5 — preamble anti-vacuity guard.** In RotateLog, before all writes and before the generic zero-entry error, inspect preamble `## ` headings. Refuse the first unparsed record-like heading: starts with digit-led `\d{4}-\d{2}-\d{2}`, starts with a decimal number, or contains the word `iteration` followed by a decimal number (case-insensitive). Name that heading in the error. All M5 structural/template headings must pass. Preserve the existing post-entry stray-heading guard. This new refusal does not apply to `RegenerateIndex`/`--index-only`.

**D6 — generated fixture and meaningful tests.** Executor checks in `internal/mission/testdata/stapledon-log-headings.txt`, generated directly from the pinned M3 source, never hand-typed:
```bash
git -C ~/dev/sunholo-data/stapledons-godot show origin/main:design_docs/stapledon-mission-log.md | grep '^## ' > internal/mission/testdata/stapledon-log-headings.txt
```
Verify origin/main is 50d8d5ba with read-only `git rev-parse`; if it moved, use that pinned commit in the same `git show` pipeline and record provenance. Require pipeline success (pipefail) and exactly 41 non-empty heading lines. Fixture loader removes trailing `\r` per line for Windows `* text=auto`; do not change `.gitattributes`. Build the test log as each original heading plus one body line. Assert 41 entries (38 numbered, 3 notes), no preamble record headings, correct Date/qualifier Title/Num, zero stray/ignored headings, and the exact expected heading sequence after rotation. Keep 20 means archive 21, keep 20; archive equals the oldest 21 in file order, with the October 2 note between 8 and 9. Index contains all numbered rows **0–9, 11, 12, 14, 16–24, 26–41** (38 distinct numbers); fixture-only test starts without old orphan rows.
Add table-driven parser cases including notes before the first number and `pre-iteration 0`. D3 cases keep equal to/greater than count assert unchanged SHA256, index written, archive absent/unchanged. D4 cases: three notices → one after real rotation; one → one across two real rotations (append a numbered entry to force the second). Preserve adjacent prose. D5 cases cover all three predicate branches, name the first offending heading, and compare pre-seeded log/archive/index bytes before/after; also cover zero parsed entries, all M5 allowed headings, and index-only remaining permissive. Use a rejected shape such as `## 2026-10-01 — iteration 9 the dash form`, adjusting only if D1 accepts it.

**D7 — changelog.** Create `changelogs/unreleased/2026-10-20-rotate-log-legacy-headings.md`, 2–5 lines total, with exact opening:
```text
### Fixed — mission rotate-log reads date-first legacy headings, never no-ops silently, and writes one archive notice
```

**D8 — bounded scope.** Implementation touches only the milestone files below within `internal/mission/`, the one D3 output line in `cmd/ailang/mission_rotate.go`, and the named fragment. `index.go` changes only recognition in `ignoredIndexHeadings`; row format/ordering is untouched. Exclude normalize.go / `ailang mission normalize`, `.gitattributes`, real mission logs, unrelated CLI changes, and all other paths except these two plan artifacts.

## One milestone: M1_ROTATE_LOG_LEGACY_HEADINGS

Files: `internal/mission/rotate.go`; **new** `internal/mission/rotate_legacy_test.go` (existing `rotate_test.go` unchanged); `internal/mission/index.go` (recognition only, D1); `internal/mission/testdata/stapledon-log-headings.txt`; `cmd/ailang/mission_rotate.go` (D3's one output line only); `changelogs/unreleased/2026-10-20-rotate-log-legacy-headings.md`.
Day 1 sequence: generate/count fixture and add failing arms → implement D1/D2/D5 → D3/D4 and fragment → focused gates and disposable mutation drills → report evidence for controller integration. No dependent milestone or dummy milestone. Split task estimate: 15 min fixture/red tests, 35 min implementation, 25 min gates/drills, 15 min buffer. Controller corpus time is separate.

Acceptance checklist:

- [ ] D1–D8 hold; fixture/control, parser fields, note placement, 21/20 split and every expected index row asserted; all existing tests green unchanged.
- [ ] D3 hashes match with an index written; D4 3→1 and 1→1; D5 named refusal leaves all three files byte-identical and allows M5 headings; index-only guard unchanged.
- [ ] Every mutation below turns its intended arm red with recorded rc/diagnostic; unmodified disposable baseline passes.
- [ ] `go test ./internal/mission/` → rc 0.
- [ ] `go test ./cmd/ailang/ -run 'Rotate|MissionRotate'` → rc 0.
- [ ] `gofmt -l internal/mission cmd/ailang` → rc 0, empty output.
- [ ] `make check-changelog` → rc 0; exact fragment opening and 2–5 lines.
- [ ] Controller-only outside-sandbox checks below pass before landing; no executor claim substitutes for them.

## Arm / mutation sensitivity table

Use `cp` into `mktemp` disposable source copies (or saved-copy restore), baseline green first. Never use `git checkout <file>`. Record each mutant's rc and intended failed assertion; compile failures alone do not establish sensitivity. Do not mutate the real fixture/logs.

| Arm | Mutation that must turn it red |
|---|---|
| Stapledon fixture parse/21–20 split | Delete numbered date-first recognition: only 5 canonical + notes remain; count/archive assertions fail. |
| Unnumbered note count/placement | Disable unnumbered date-first recognition: notes fall into preamble/body; guard or entry count fails. |
| Pre-iteration note Num | Make numbered regex accept `pre-iteration 0`: Num becomes 0 rather than -1. |
| Exact archive order | Restore `sort.Slice` by Num: October 2 note moves ahead of iteration 0. |
| D3 unchanged live SHA256 | Write live log unconditionally: hash differs. |
| D4 exact notice count | Skip old-notice stripping: 3 copies become 4. |
| D5 named error + three unchanged files | Delete preamble guard: rejected shape silently succeeds/no-ops. |
| Fixture instrument control | Assert exactly 41 non-empty lines before use; missing/truncated fixture fails explicitly as instrument failure, never passes. |

## Controller integration and risks

**Controller, outside sandbox:** run `make test-launchd-drivers`. Compare **HEAD binary vs new binary**, `ailang mission rotate-log <name> --keep 20`, against independently seeded **SCRATCH** copies of every real log: v1, fleet, docs, triage, motoko, world from `~/dev/sunholo-data/ailang-world`, and stapledon. Copy associated archives/indexes too and use scratch registry paths; compare parsed entry counts, archived counts and resulting live/archive/index files. Only allowed differences: stapledon now rotates, and duplicate notices collapse. Record any other difference as failure for investigation. Do not edit real logs. Do not prescribe `make test` or `go build ./...`.

Risk controls: stable decorated keys avoid comparator/index drift and note reordering; anchored notice matching preserves prose; the preamble predicate preserves templates while detecting silently skipped records; shared recognition keeps index-only counts honest. Use CR stripping only in the fixture loader. Fixture provenance/count and mutation controls prevent vacuous passes. Controller corpus checking catches cross-mission changes before landing.
Every socket-touching command result must be labelled **UNINFORMATIVE UNDER SANDBOX**; controller M1–M6 remain established outside-sandbox evidence. Planner validation is limited to JSON/schema and scope review, with rc/results recorded in sprint JSON notes. One milestone is intentional; retain it if the validator's only error is minimum two.
