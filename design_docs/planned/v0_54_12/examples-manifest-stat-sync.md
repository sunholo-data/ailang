# Examples Manifest Statistics Sync — close the stale-stats and unmanifested-example holes

**Status**: Planned
**Target**: v0.54.12
**Priority**: P1 (Medium) — dev-CI reliability, repeat offender
**Estimated**: 1 day (2× the naive ~4h: tool ~2h, gate ~2h, self-tests ~1h, docs ~1h)
**Dependencies**: None
**Created**: 2026-10-15 (design-doc-creator, task-18a95848, digest issue #1835)

**Related Documents:**
- [M-DX-EXAMPLES-COVERAGE](../implemented/v0_30_0/m-dx-examples-coverage.md) — implemented v0.30.0: built the manifest system, the `verify-examples` CI gate, `validate_manifest --ci` and the `modules` backfill/drift tooling this doc extends. Distinct scope: that doc added coverage and the modules field; this doc fixes the `statistics` block and the disk→manifest direction, neither of which it covered.
- [spawn-process-result](v0_54_7/spawn-process-result.md) — the #1834 feature doc whose checklist says "+1 manifest entry" with no statistics regeneration step: the exact blind spot that produced half of this red (see Problem §1c).
- [PROGRAM](../../PROGRAM.md) — routing: this is an AILANG repo tooling fix, not a core change.

## Problem Statement

CI `test` job, step "Verify examples" (`make verify-examples`) went red on dev at
`f83fdd52` and stayed red through the v0.54.11 release commit `046cecb9`
([run 38742020131](https://github.com/sunholo-data/ailang/actions/runs/38742020131)):

```
manifest failed schema validation: statistics mismatch: recorded
{Total:235 Working:227 Broken:4} vs calculated {Total:236 Working:228 ...}
```

**Root cause chain** (verified from git history, not inferred):

1. The `statistics` block in `examples/manifest.json` is *derived data* — fully
   recomputable from the `examples` array — but it is maintained by **hand-editing
   JSON numbers**. `total` is an absolute counter; `coverage` is a 17-significant-digit
   float (`0.9661016949152542`) that no human or agent can regenerate by hand.
2. PR #1834 (`2553f7da`) added `runnable/process_spawn_result.ail` and hand-bumped
   `total 234→235`, `working 226→227` — correct in isolation.
3. PR #1836 (`f83fdd52`) added `runnable/map_strict_parity.ail` and its manifest
   entry but never touched `statistics`.
4. Both branches were internally consistent (each green in its own PR CI), but the
   squash merge of #1836 onto a dev that already contained #1834 kept one absolute
   counter: **236 entries, statistics saying 235/227**. Hand-maintained absolute
   counters are merge-unsafe: two consistent branches merge into an inconsistent
   tree, and the red surfaces only *after* merge, on dev.
5. The red was fixed by more hand arithmetic (`35a49b85` hand-edited `235/227 →
   236/228` and re-derived the coverage float) — the failure mode was repaired with
   the same tool that caused it.

**Detection existed; regeneration did not.** `manifest.Load → Validate`
(`internal/manifest/manifest.go:118-129,194-197`) is exactly the check that fired.
But the only tool that writes the committed manifest,
`scripts/backfill_manifest_modules.go`, *deliberately preserves `statistics`
verbatim* (`Statistics json.RawMessage`, line 56), and `manifest.Save()` — the sole
statistics-recomputing path in the codebase — has **no production caller** (grep:
only `internal/manifest/manifest_test.go:361`). So the CI failure message offers no
remediation command, and the correct remediation ("recompute the block") did not
exist as a tool.

**This is the second bite.** `make/ci.mk:19` records (comment, 2026-09-27): "the
closure count and **manifest statistics** each broke dev or a PR after a direct
push." Per CLAUDE.md principle 3 (systemic fixes), a second occurrence of the same
failure mode means the process is broken, not the person.

**The complementary hole is silent today.** A new example can land with **no
manifest entry at all** and CI stays green: `verify_examples.go` enumerates by
*disk walk* of `examples/runnable/` (lines 691-705), so the file is run and gated on
pass/fail, while `validate_manifest.go` checks only manifest→disk (it iterates
entries; missing-on-disk is a warning, unmanifested-on-disk is never checked). The
statistics gate is trivially bypassed by skipping the entry. This is not
hypothetical: **47 of 257 runnable `.ail` files (18%) currently have no manifest
entry** and dev is green (measurement below). One entry is also stale
(`lambda_expressions.ail` — no file on disk).

**Impact:** every agent/human adding an example must hand-perform arithmetic and
merge-fragile counter edits; concurrent example PRs race exactly like #1834×#1836
did; and the release train (v0.54.11) shipped over a red CI step.

## Goals

**Primary Goal:** make it impossible for an example to land with stale or missing
manifest statistics — regeneration becomes one deterministic command, and the CI
gate covers both drift directions.

**Success Metrics:**
- Adding an example = add file + entry, run `make sync-examples-manifest`, commit:
  zero hand-edited numbers, zero hand-computed floats.
- The statistics gate failure message names the exact regen command (like the
  modules drift lint already does).
- A runnable example file with no manifest entry and no explicit allowlist entry
  turns `make verify-examples` red.
- `make verify-examples` and `make ci-quick` green on dev after landing (dev is
  currently green at `377f96a0`; this doc must not regress it).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: One committed-manifest writer — extend `backfill_manifest_modules.go` into a full sync, not a second stats-only script | Two half-syncers (modules + stats) recreate the fragmentation this red came from | agent | design | low |
| D2: Recompute statistics inside the backfill's mirror-doc writer with a local typed struct (incl. `aspirational`), NOT via `internal/manifest.Save` round-trip | `Save` would drop the unmodeled `aspirational` field, reorder keys, and rewrite `generated_at` | agent | design | low |
| D3: Keep the `statistics` block in the committed file (schema `ailang.manifest/v1` unchanged); reject "compute at load" | Dropping a schema field churns all three manifest readers for no added invariant | human | design | med |
| D4: New unlisted-runnable gate ships seeded with today's measured 47 files, so the gate lands green and the backlog cleanup is an explicit follow-up | Unseeded, the gate would turn dev instantly red on 18% of runnable files | human | design | med |
| D5: Gate scope = `examples/runnable/` only (the set the verifier gates on); non-runnable fixture dirs (`tests/`, `docs/`, `snippets/`, …) stay out | Gating all of examples/ would demand manifesting test fixtures — a different project | human | design | med |
| D6: `internal/manifest.Statistics` gains the missing `Aspirational` field so the load-time check covers all six recorded fields | Today's recorded-vs-calculated comparison silently ignores one committed field | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1/D2 (agent): single sync tool, mirror-doc statistics recompute (verified:
  `Save` has no production caller; backfill preserves `statistics` verbatim today).
- [ ] D3/D4/D5 (human): confirm at design review — keep the block in-file, seed the
  allowlist, gate runnable-only. **A reviewer rejecting D4 or D5 must say what the
  gate should do about the 47 measured files.**
- [x] D6 (agent): additive field, backward-compatible (see Verification Log row 8).

### Deferred Decisions

The following are intentionally left open for the implementer:

- Allowlist representation: embedded Go map in `validate_manifest.go` (mirrors
  `skippedExamples` in `verify_examples.go`) vs. a committed sidecar JSON file —
  agent may choose; bias: embedded map (self-contained, greppable, precedented).
- Whether the sync tool also refreshes `generated_at`/`generator` when it changes
  content — agent may choose; bias: leave both untouched (they describe the last
  structural generation; churn invites merge conflicts).
- Exact wording/format of the two new gate failure messages — agent may choose.
- How the 47 seeded files get dispositioned over time (manifest entry vs. permanent
  allowlist as helper fixture vs. delete) — human at review, follow-up issue.

## Solution Design

### Overview

Split the fix along the two failure directions, reusing the architecture
M-DX-EXAMPLES-COVERAGE established for `modules` (backfill tool + `--ci` drift
lint + self-test) so statistics gets the identical lifecycle:

1. **Regeneration:** the backfill script (renamed in behavior, not in filename)
   becomes the single manifest sync: after resetting each entry's `modules` it
   recomputes the whole `statistics` block from the entries.
2. **Gating:** `validate_manifest.go --ci` (a) prints the regen command whenever
   `manifest.Load` fails, and (b) gains a disk→manifest coverage check for
   `examples/runnable/` backed by a committed allowlist seeded with today's 47
   unmanifested files.
3. **Honesty of the validator:** `internal/manifest.Statistics` models the
   `aspirational` field it currently ignores, so recorded-vs-calculated compares
   all six fields.

### Architecture

**Components:**

1. **`scripts/backfill_manifest_modules.go`** (sync writer): today's
   `doc`/`entry` mirrors stay; `Statistics json.RawMessage` is replaced by a typed
   `statisticsDoc` struct matching the committed key order (`total`, `working`,
   `aspirational`, `broken`, `coverage`, `experimental`). After the modules pass,
   recompute all six fields: `total=len(entries)`; per-status counters;
   `coverage=float64(working)/float64(total)` (Go's default float64 JSON encoding
   reproduces `0.9661016949152542` byte-for-byte — it is plain
   `228/236`). `--check` mode additionally reports statistics drift. Header comment
   updated: the script syncs all derived manifest fields, not just `modules`.
2. **`scripts/validate_manifest.go`** (gate): on `manifest.Load` failure, append
   the hint `if this is statistics/modules drift, run: make sync-examples-manifest`
   to the existing error output. After the modules pass, walk `examples/runnable/`,
   resolve every manifest entry via the shared `importextract.ResolvePath`, and
   fail `--ci` on any runnable file that is neither manifest-listed nor allowlisted.
   Files beginning with `_` are skipped (scratch-fixture convention, see
   `verify-examples-gate-selftest`'s `_gate_selftest_broken.ail`).
3. **`make/examples.mk`**: new `sync-examples-manifest` target (runs the script);
   `backfill-manifest-modules` retained as an alias. `validate-manifest-selftest`
   extended to prove both new gate legs fire on crafted temp manifests/allowlists
   (stale statistics → exit ≠ 0; unlisted runnable → exit ≠ 0) — guards the guard,
   same pattern as `verify-examples-gate-selftest`.
4. **`internal/manifest/manifest.go`**: `Statistics` struct gains
   `Aspirational int \`json:"aspirational"\``; `calculateStatistics` counts
   `StatusAspirational` into it. `Validate`'s equality then covers all six fields.
   Additive: the committed file already carries the field, so the current manifest
   loads with `4 == 4` and files without the field parse as `0`.

### Implementation Plan

**Phase 1: Sync tool (~2h)**
- [ ] Replace `Statistics json.RawMessage` with typed `statisticsDoc`; recompute
      after the modules pass; extend `--check`.
- [ ] Add `sync-examples-manifest` make target; keep old target as alias.
- [ ] Run it on dev: output must be byte-identical to the committed file (no diff)
      — this is the determinism acceptance test.

**Phase 2: Gate (~2h)**
- [ ] Regen hint on `manifest.Load` failure in `validate_manifest.go`.
- [ ] Disk→manifest coverage check for `examples/runnable/` + seeded allowlist
      (47 measured entries, each with reason `pre-existing unmanifested, measured
      2026-10-15`); `--ci` exits non-zero on unlisted files.
- [ ] Update the `verify-examples` recipe comment (make/examples.mk:16-17) to say
      statistics drift + unlisted examples, not only modules drift.

**Phase 3: Validator honesty + self-tests (~1h)**
- [ ] `Aspirational` field in `internal/manifest.Statistics` + counting + unit test.
- [ ] Extend `validate-manifest-selftest`: stale-stats leg and unlisted-runnable
      leg, both on temp copies with `trap` cleanup (never mutate the real manifest).

**Phase 4: Docs (~1h)**
- [ ] Update `docs/` example-contribution guidance and the example-addition
      checklists: "add entry, run `make sync-examples-manifest`, commit".
- [ ] CHANGELOG entry under v0.54.12; note in `design_docs/planned/v0_54_7/
  spawn-process-result.md`'s success-criteria pattern is NOT edited (historical).

### Files to Modify/Create

**Modified files:**
- `scripts/backfill_manifest_modules.go` (+~45/-10 LOC) — typed statisticsDoc, recompute, `--check` extension, header docs
- `scripts/validate_manifest.go` (+~70/-5 LOC) — load-failure hint, unlisted-runnable gate, seeded allowlist
- `make/examples.mk` (+~30/-5 LOC) — `sync-examples-manifest` target, alias, self-test legs
- `internal/manifest/manifest.go` (+~8 LOC) — `Aspirational` field + counting
- `internal/manifest/manifest_test.go` (+~25 LOC) — aspirational counting + mismatch case
- `changelogs/v0.32-current.md` (+~6 LOC) — v0.54.12 entry

**New files:** none (allowlist embedded in `validate_manifest.go` if the deferred
representation decision resolves that way; otherwise one sidecar JSON).

## Examples

### Example 1: Adding an example — before vs after

**Before (the #1836 workflow):** add `runnable/map_strict_parity.ail`, add a
manifest entry by hand, hand-edit `total`/`working` and hand-recompute a 17-digit
coverage float (or forget, go red on dev, hand-fix in `35a49b85`).

**After:**
```bash
# 1. write examples/runnable/map_strict_parity.ail
# 2. add its manifest entry (path/status/tags/description — authored, not derived)
# 3. one command:
make sync-examples-manifest   # regenerates modules AND statistics
# 4. commit file + manifest together
```

### Example 2: The gate message a stale PR now gets

```
✗ manifest failed schema validation: manifest validation failed: statistics
  mismatch: recorded {Total:235 Working:227 ...} vs calculated {Total:236 ...}
  if this is statistics/modules drift, run: make sync-examples-manifest
```

vs today's bare `statistics mismatch: recorded … vs calculated …` with no remediation.

### Example 3: The silent hole, closed

Drop `runnable/my_new_example.ail` in with no manifest edit:
```
✗ runnable/my_new_example.ail: on disk under examples/runnable/ but not in
  examples/manifest.json (and not allowlisted)
  add an entry, then run: make sync-examples-manifest
```

## Success Criteria

- [ ] `make sync-examples-manifest` on current dev produces **zero diff** (numbers already correct) — proves byte-identical recompute
- [ ] Corrupting `statistics.total` in a temp manifest makes `go run ./scripts/validate_manifest.go --ci` exit non-zero with the regen command in output
- [ ] An unlisted temp runnable file makes `--ci` exit non-zero; allowlisted files pass
- [ ] `make validate-manifest-selftest` proves both legs on temp copies
- [ ] `make verify-examples`, `make ci-quick`, `make test` green
- [ ] `internal/manifest` unit tests cover aspirational counting and the 6-field equality
- [ ] Docs updated (contribution checklist + CHANGELOG)
- [ ] Follow-up issue filed for the 47-file backlog disposition (gate ships seeded)

## Conflict Surface

Not a parser/typechecker/codegen change (none of the mandatory directories), so
this is the tooling conflict surface: every consumer of `examples/manifest.json`
and of the fields this design rewrites.

| Reader | What it reads | Effect of this change |
|---|---|---|
| `internal/manifest.Load` (via `scripts/validate_manifest.go`, `scripts/verify_examples.go`) | whole file; `Statistics` (now incl. `aspirational`) | none on green inputs; stricter equality on aspirational |
| `cmd/ailang/examples.go` mirror (`ExampleManifest`) | `examples[].{path,status,tags,description,expected,modules}`; deliberately omits statistics | none — struct ignores unknown fields; no schema change |
| `cmd/ailang/docs.go` (`docs --examples`) | `examples[].modules` | none — modules pass unchanged |
| `backfill_manifest_modules.go` round-trip | all fields verbatim (RawMessage) except modules (and now statistics) | statistics key order/values identical to committed; `milestones`/`generated_at`/`generator` preserved |
| `scripts/update_readme.go`, `tools/build-snapshot` | `examples_report.json` (NOT the manifest) | none |
| `GenerateREADMESection` (`internal/manifest`) | `Statistics.Working/Total` | reads the same values; no production caller (test-only) |

**Fixtures that MUST still work:** the current `examples/manifest.json` (236
entries, statistics 236/228, `aspirational: 4`) loads, validates, and syncs with
zero diff; `make verify-examples-gate-selftest` still passes (its `_`-prefixed
fixture is skipped by the new gate); `make check-referenced-paths` and
`check-changelog` (ci-quick) unaffected.

**What deliberately changes:** `validate_manifest --ci` exits non-zero on two new
drift classes (stale statistics — previously red only via the load error, now with
a remediation hint; unlisted runnable files — previously green, now red unless
allowlisted). Nothing that is green today goes red: the 47 measured files are
seeded into the allowlist in the same change.

## Testing Strategy

**Unit tests (`internal/manifest`):** aspirational counted in
`calculateStatistics`; `Validate` fails when recorded `aspirational` disagrees;
`Save`-round-trip unaffected (no production path uses it).

**Gate self-tests (`make validate-manifest-selftest`):** three legs on temp copies
(never the real manifest, `trap` cleanup):
1. stale `statistics.total` in a temp copy → `--ci` non-zero + hint printed
2. unlisted runnable fixture on disk (temp examples dir) → non-zero + entry named
3. control: current manifest + current disk → exit 0

**Integration:** `make verify-examples` end-to-end on dev (currently green at
`377f96a0`) must stay green after seeding; `make ci-quick` (pre-push hook path)
green.

**Manual:** corrupt stats in a scratch clone → push → pre-push hook refuses with
the regen command (the hook runs `ci-quick`, which runs this validator).

## Non-Goals

**Not in this feature:**
- Auto-generating manifest entries (tags/descriptions) for new examples —
  authoring stays human/agent; only *derived* fields become machine-synced.
- Removing the `statistics` block from schema v1 or bumping to manifest/v2 —
  rejected alternative, see Risks.
- Manifesting non-runnable fixture dirs (`tests/`, `docs/`, `snippets/`, `bench`,
  `debug/`, …) — never manifested by design (M-DX-EXAMPLES-COVERAGE scoped the
  manifest to the verified runnable set + curated quarantines).
- A CI auto-fix bot that regenerates and pushes — violates fail-loud (CLAUDE.md
  #2); the gate names the command, humans/agents run it.
- `examples/README.md`/`STATUS.md` numbers — different pipeline
  (`scripts/update_readme.go` reads `examples_report.json`, not the manifest).
- Dispositioning the 47 seeded files — follow-up issue (deferred decision).

## Timeline

**Day 1 (~6h):** Phase 1 sync tool (2h), Phase 2 gate + seeding (2h), Phase 3
validator + self-tests (1h), docs + CHANGELOG (1h). Buffer already included (2×
naive estimate).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Float formatting drift rewrites the coverage field spuriously | Low | Go `float64` division + default `encoding/json` — verified it reproduces `0.9661016949152542` exactly (Verification Log row 17); acceptance test: zero-diff on dev |
| Allowlist rots into a dumping ground for future lazy additions | Medium | Each entry carries a reason string (pattern: `skippedExamples`); follow-up issue owns disposition; PR reviewers see allowlist diffs as reviewable "explicitly undocumented" acts |
| Concurrent example PRs still race on the entries array itself | Low | Merge conflicts on the entries array are visible and mechanical (no absolute counters involved); statistics is regenerated, never merged — that was the actual failure mode |
| Stricter equality (`aspirational`) rejects some hypothetical older manifest | Low | Only manifest in the repo already carries the field; gate self-tests pin the behavior |
| "Compute at load, drop the block" reviewers re-raise D3 | Low | Answer is in Design Freeze: schema churn across three readers buys no new invariant; recorded block is also the diff-visible signal CI gates on |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One deterministic regeneration path (byte-identical output, zero-diff acceptance test) replaces hand arithmetic and hand-computed floats |
| A2: Replayability | 0 | No execution semantics touched |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | The invariant is checkable locally in seconds (part of `ci-quick`/pre-push), no full CI round-trip needed to know your example landed clean |
| A6: Safe Concurrency | 0 | Merge-safety of the *file* improves (no absolute counters to merge), but no runtime concurrency |
| A7: Machines First | +1 | Agents get a command + actionable failure message instead of hand-editing a 17-digit float — this red was produced by an agent doing exactly that |
| A8: Minimal Syntax | 0 | No language surface |
| A9: Cost Visibility | 0 | No resource semantics |
| A10: Composability | +1 | Reuses the M-DX-EXAMPLES-COVERAGE backfill+lint+selftest architecture and the shared `importextract.ResolvePath` |
| A11: Structured Failure | +1 | Gate failures carry the exact remediation command; fail-loud preserved (no silent auto-fix) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (regeneration is a pure function of the entries)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): the design *is* machine-first — it removes the only step that required human-style arithmetic

## Verification Log

Every "does/doesn't" claim above was checked live in this session (repo at
`377f96a0` = dev head) or cites a file:line. No Go toolchain was available in this
session, so no claim below rests on having *run* a script; code-path claims rest on
reading the code, history claims on `git show`, and the CI facts on the GitHub API.

| # | Claim | Evidence |
|---|---|---|
| 1 | Red mechanism: #1834 (`2553f7da`) hand-bumped `234→235`/`226→227`; #1836 (`f83fdd52`) added `runnable/map_strict_parity.ail` + entry with **no** statistics change; `35a49b85` hand-fixed `235/227→236/228` + coverage float; its message: "Both PRs bumped total 234→235 from the same base; the squash merges kept one bump" | `git show 2553f7da/ f83fdd52/ 35a49b85 -- examples/manifest.json` (diffs inspected in session) |
| 2 | Failing run 38742020131 = workflow "CI dev", event `push`, head `046cecb9d514bcada8c78fb9dd61be5c93f59596`, conclusion `failure` | GitHub API `GET /repos/sunholo-data/ailang/actions/runs/38742020131` (session) |
| 3 | Statistics mismatch is checked at load, inside `Validate` | `internal/manifest/manifest.go:118-129` (`Load` → `Validate`), `:194-197` (the exact error string) |
| 4 | Backfill preserves `statistics` verbatim; header documents "All manifest fields (… statistics …) are preserved" | `scripts/backfill_manifest_modules.go:20-22,56` and struct `doc` (RawMessage) — file read in full |
| 5 | `manifest.Save` (the only stats-recomputing path) has no production caller | `grep -rn "\.Save(" internal/ scripts/ cmd/ tools/` → only `internal/manifest/manifest_test.go:361` |
| 6 | `verify_examples.go` enumerates by disk walk, not from the manifest | `scripts/verify_examples.go:691-705` (`filepath.Walk(runnableDir)`); manifest used only for pinned stdout (`:106-160`) |
| 7 | `validate_manifest.go` checks only manifest→disk; unmanifested files never checked; missing-on-disk is warning-only | full read of `scripts/validate_manifest.go` (125 lines): entry loop `:66-98`, `missingCount` printed yellow, no inverse walk |
| 8 | Committed `statistics` block carries `aspirational: 4`; `internal/manifest.Statistics` does not model it; `calculateStatistics` buckets only working/broken/experimental | `examples/manifest.json` (raw block: total, working, aspirational, broken, coverage, experimental) vs `internal/manifest/manifest.go:85-92,262-278` |
| 9 | 47 of 257 runnable `.ail` files are unmanifested; 1 stale entry (`lambda_expressions.ail`) | session measurement: Python walk of `examples/runnable` + `importextract.ResolvePath` semantics (direct then `runnable/`, `scripts/internal/importextract`: `ResolvePath`) over the committed manifest; 236 entries, 235 resolvable, 25 non-runnable |
| 10 | Second bite: "2026-09-27: … the closure count and manifest statistics each broke dev or a PR after a direct push" | `make/ci.mk:19` (comment above `ci-quick`) |
| 11 | CI runs on push AND pull_request to main/dev; pre-push hook runs `ci-quick` in a throwaway worktree; GitHub-side squash merges bypass local hooks | `.github/workflows/ci.yml:3-6`; `make/ci.mk:17-21`; `scripts/hooks/pre-push:106-107` |
| 12 | The #1834 feature doc's checklist says "+1 manifest entry" with no statistics step | `design_docs/planned/v0_54_7/spawn-process-result.md:228,243,299` |
| 13 | `GenerateREADMESection` has no production caller | grep across repo → only `internal/manifest/manifest_test.go:287,305` |
| 14 | `cmd/ailang/examples.go` mirror omits statistics entirely (unknown-field-tolerant) | `cmd/ailang/examples.go:17-30` read |
| 15 | Go float64 division reproduces the committed coverage byte-for-byte | `228/236 = 0.9661016949152542` (session check) = committed value; both sides are IEEE-754 double shortest-round-trip encodings |
| 16 | Embedded exclusion-map pattern exists (precedent for the allowlist) | `scripts/verify_examples.go:170-176` (`skippedExamples` with reasons, incl. quarantine issue links) |
| 17 | No planned/implemented doc covers manifest statistics sync (duplicate gate) | skill auto-search (SimHash + neural, both empty) + manual grep over `design_docs/` for `examples/manifest`+statistics; closest = M-DX-EXAMPLES-COVERAGE (different scope, see Related Documents) |
| 18 | No new error code is proposed | design allocates none; gate failures reuse the existing `fmt.Errorf` strings at `manifest.go:196` and `validate_manifest.go` stderr paths — `grep -rn "MOD0\|TC0\|PAR0"` not needed, no codes written into this doc |

## References

- **Incident**: [CI run 38742020131](https://github.com/sunholo-data/ailang/actions/runs/38742020131) — "Verify examples" red at `046cecb9d`; first red `f83fdd52` (#1836); fix `35a49b85`
- **Digest thread**: issue #1835 (triage bookkeeping, week of 2026-10-12)
- **Prior art**: [M-DX-EXAMPLES-COVERAGE](../implemented/v0_30_0/m-dx-examples-coverage.md) — the backfill+lint+selftest pattern this design extends
- **Workflow docs**: `docs/docs/guides/` (development workflow), `make/examples.mk` header comments
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Disposition the 47 seeded allowlist files (manifest entry vs. permanent helper
  fixture vs. delete) — follow-up issue, human-at-review per Deferred Decisions.
- If the stale-statistics failure mode bites a third time *after* this ships, the
  friction→extension doctrine (AGENTS.md) fast-tracks a `.pi/extensions` hook that
  runs `make sync-examples-manifest` on example-manifest saves — twice is already
  the threshold this doc answers at the repo-tooling layer.
