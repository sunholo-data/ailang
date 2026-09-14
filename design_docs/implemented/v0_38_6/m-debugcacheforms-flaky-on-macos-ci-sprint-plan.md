# Sprint Plan: M-DEBUGCACHEFORMS-FLAKY-ON-MACOS-CI — drain the capture pipe before closing its read end

**Sprint ID:** `v1_iter353_debugcacheforms`
**Design doc (approved):** [m-debugcacheforms-flaky-on-macos-ci.md](m-debugcacheforms-flaky-on-macos-ci.md) (r3, quorum round 2 all reviewer fixes applied)
**Target:** v0.38.6 · **Mission:** V1, iteration 353
**Duration:** 1 day (hard ceiling; plan estimate 0.5 day, ~4 hours — matches the design doc's estimate)
**Dependencies:** None
**Risk Level:** Low (test-file-only change; 3 call sites, one helper, one ordering invariant)

## Summary

Fix the `capturePipelineStderr` test helper in `internal/pipeline/pipeline_module_phases_test.go`: its
teardown closes the pipe's read end before the copier goroutine has drained it, so on a loaded macOS CI
runner the tail (or all) of the captured `[CACHE]` lines is discarded and
`TestPipelineModulePhases_DebugCacheFormsAndCounters` goes red intermittently. The fix reorders the
teardown to **writer → drain-wait hook → bounded wait → reader** inside a panic-safe named-return
`defer`, adds a channel-gated-reader seam so the old ordering is deterministically red, and drills
six named mutants. The CI green streak (AC6) is a controller-owned post-merge observation, not an
executor gate.

## Current Status Analysis

### Velocity
- Design-doc estimate: 0.5 day, one milestone of real work (~4 h). This plan splits it into **two
  atomic commits** (M1 = the fix + proof, M2 = the record + sweep) because the sprint JSON requires
  ≥2 milestones and because the controller commits per milestone.
- **Estimated total LOC (script-scan line): 131 over the whole sprint** —
  ~+80/−8 in `internal/pipeline/pipeline_module_phases_test.go` (the doc says +~110/−8 if H2 lands in
  the phases file too; +~80 is the figure if H2's tests live in a sibling `pipeline_stderr_capture_test.go`),
  +1–3 in `changelogs/v0.32-current.md` (see D-2 below), +~40 in the mutation-audit record.

### Planner re-verification at this worktree (HEAD `b794560ca`)

The doc's Verification Log was measured at `05a6457d1`. I re-measured the load-bearing rows here;
no drift. Where doc and worktree disagree, the table quotes both ("refutation is the loop working"):

| # | Check | Doc says | Measured here | Verdict |
|---|-------|----------|---------------|---------|
| P1 | Helper location & defect | `pipeline_module_phases_test.go:28-44`; teardown `os.Stderr = old; w.Close(); r.Close(); <-done`; HEAD-order at `:41→42` (V5) | `sed -n '1,50p'` shows def at `:28`, doc comment at `:19-27`, `os.Stderr = old` `:39`, `w.Close()` `:40`, `r.Close()` `:41`, `<-done` `:42` | ✓ matches |
| P2 | Call sites | 3 calls + def + comment = 5 hits (`:19,:28,:226,:239,:256`, V4) | `grep -n 'capturePipelineStderr'` → exactly `:19 :28 :226 :239 :256` | ✓ matches |
| P3 | Tests in file | `grep -c '^func Test'` = 8 (V12); file 403 LOC | 8; 403 LOC | ✓ matches |
| P4 | HangGuard signature | `func HangGuard(t *testing.T, cap time.Duration) time.Duration` at `internal/testutil/gate.go:88` (V18) — **not reused** (needs `*testing.T`) | `sed -n '86,107p' internal/testutil/gate.go` → signature exactly as quoted | ✓ matches |
| P5 | Baselines (AC1/AC4/AC5, V10/V3/V11) | all `rc=0`; `go build ./...` is `rc=1` at base (`cmd/wasm` has no native `main`) — **excluded from every gate** | AC1 `ok 0.649s rc=0`; AC4 `-count=20 -race` `ok 3.171s rc=0`; AC5 build+vet `rc=0`; AC2 baseline `no tests to run rc=0`; `gofmt -l` clean | ✓ re-measured green |
| P6 | Base-commit drift | verified at `05a6457d1` | this worktree is `b794560ca`; `git log --oneline 05a6457d1..HEAD -- internal/pipeline/` → **empty** (no commits touched the package between the two) | ✓ equivalent base |

**Measured disagreements (doc text quoted vs. repo truth; the resolution is stated in the task text):**

- **D-1 — test count in H2.** The doc's Implementation Plan bullet H2 says "add `gatedReader` + **the
  three tests** (`GatedReaderLosesNothing`, `PanicRestoresStderr`, `DrainTimeoutIsLoud`)". The doc's
  r3-amended Architecture §8, Testing Strategy, **AC2** (which demands four `--- PASS:` lines) and the
  mutation table (**MUT-4**) all require a **fourth** test, `TestCapturePipelineStderr_CopyErrorIsLoud`.
  *Resolution:* the r3 amendment wins — **four** tests are built; the H2 bullet is stale r1 text. AC2 is
  load-bearing, so nothing hinges on the bullet.
- **D-2 — "unreleased section" does not exist.** H3 says "one line in `changelogs/v0.32-current.md`
  under the unreleased section (`Fixed`)". Measured: `grep -ni unreleased changelogs/v0.32-current.md`
  → 2 hits, both at `:4870` and `:4916` inside historical v0.37-era prose describing a changelog-gate
  rule about the ROOT index; in this file the live convention is per-version
  headers (`## [v0.38.5] - 2026-09-13`) containing `### Fixed — <one-line title>` entries.
  *Resolution (controller-corrected):* the executor opens a `## [Unreleased]` header at the top of
  the file (the shape this file carried until the v0.38.5 release consumed it — `git log -S'## [Unreleased]'`
  shows every release renaming it; the release manager will rename it again) with a single
  `### Fixed — …` entry beneath it. NOT a `## [v0.38.6]` header: the loop never picks release versions.
- **D-3 — mutation roster.** The mission prompt mentions "MUT-1/MUT-2/MUT-3". The doc (which wins)
  names six mutants: MUT-1, MUT-2a, MUT-2b, MUT-3a, MUT-3b, MUT-4 (AC3 requires drilling all six and
  recording the MUT-4 result as V21). All six are in the test-plan table below.

## Proposed Milestones

### M1: Helper seam + drain-before-close reorder + regression tests + mutation drill (~2.5 hours, ~88 LOC changed)

**Goal:** Land hunks H1 and H2 (design doc "Solution Design"): rewrite the helper as
`capturePipelineStderrWith(opts, f) (out string)` with the panic-safe named-return `defer` teardown
(restore → `w.Close()` → drain-wait hook → bounded wait → `r.Close()` → copy-error check →
`out = buf.String()`), keep `capturePipelineStderr(f)` as a one-line `nil`-opts wrapper, and add the
`gatedReader` type plus four regression tests that are **red on the HEAD ordering by construction**.
Then run the six-mutant drill (table below) to prove the kills are structural.

**Estimated:** ~+80/−8 LOC in `internal/pipeline/pipeline_module_phases_test.go` or a sibling
`pipeline_stderr_capture_test.go` (executor's choice per the doc's Deferred Decisions; the helper
itself stays in the phases file at `:28-44`).
**Dependencies:** None

**Acceptance criteria owned by M1 (copied verbatim from the design doc's Success Criteria):**

- [ ] **AC1** `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` → `rc=0`
  (baseline `rc=0`, V10). Can fail for this diff: H1 changes the helper all six assertions read
  through.
- [ ] **AC2** `go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=1 -timeout 60s -v`
  → `rc=0` AND the output contains all four of
  `--- PASS: TestCapturePipelineStderr_GatedReaderLosesNothing`,
  `--- PASS: TestCapturePipelineStderr_PanicRestoresStderr`,
  `--- PASS: TestCapturePipelineStderr_DrainTimeoutIsLoud`,
  `--- PASS: TestCapturePipelineStderr_CopyErrorIsLoud`
  (baseline: `no tests to run`, so the `--- PASS` lines are the load-bearing half).
- [ ] **AC3 — deterministic MUT-1 drill.** With `_ = r.Close()` moved above the release hook, run
  `go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1`
  **five times**: `rc=1` on every run, failing on iteration 0 with the helper's explicit
  `copier failed: … file already closed` panic (r2: an explicit copy-error failure, not an
  empty-capture equality failure). Restore: `rc=0`. No parameter is tuned between runs; if any run
  is `rc=0` the gate is not holding the reader before its first `Read` and the test is wrong, not
  slow. All rcs go in the checkpoint. The same drill for MUT-2a, MUT-2b, MUT-3a, MUT-3b and MUT-4
  (one run each suffices — each is structural) is recorded alongside, and the MUT-4 /
  `CopyErrorIsLoud` result is recorded in the Verification Log as V21.
- [ ] **AC4** `go test ./internal/pipeline/ -run TestPipelineModulePhases_DebugCacheFormsAndCounters -count=20 -race`
  → `rc=0` (baseline `rc=0`, V3). Cannot *prove* the flake is gone (see AC6) but can catch a
  regression in the reordered teardown under the race detector.

(AC5 is formally closed in M2's acceptance sweep; M1 still runs `go vet ./internal/pipeline/` as part
of every drill iteration — see the table's proof column.)

**Mutation test plan (all six doc-named mutants; the drill is M1's core, drilled BEFORE handoff):**

Restore-from-copy discipline applies to every row: `cp internal/pipeline/pipeline_module_phases_test.go
internal/pipeline/pipeline_module_phases_test.go.bak` *before* the mutant, `cp …bak` file back *after*,
and `shasum -a 256` of the restored file asserted equal to the pre-mutation hash recorded in
`.snap/drill/`. **NEVER `git checkout -- <file>`** — the M1 edits are uncommitted executor work and a
checkout would destroy them.

| mutation | exact edit (in the file holding `capturePipelineStderrWith`) | expected result | killer test | proof the mutant LANDED and BUILDS before reading the test |
|----------|--------------------------------------------------------------|-----------------|-------------|-------------------------------------------------------------|
| MUT-1 (H1 reorder — the HEAD defect) | In the `defer` teardown, move `_ = r.Close()` from after the bounded-wait `select` to immediately after `_ = w.Close()` — i.e. above the `opts.release()` hook and the bounded wait | `rc=1` on **all 5** consecutive runs, failing on iteration 0 via the helper's explicit `copier failed: … file already closed` panic; after restore `rc=0` | `go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1` ×5 | `shasum -a 256 <file>` ≠ pre-mutation hash (edit landed); `go vet ./internal/pipeline/` `rc=0` (it builds — a non-compiling red is not a kill); both rcs logged |
| MUT-2a (H1 seam) | Make `capturePipelineStderrWith` ignore `opts.wrap` — delete or bypass `src = opts.wrap(r)` so `src` stays `r` | `rc=1` (1 run suffices): the `reads ≥ 1` assertion is red because the wrapper's `Read` never runs | same suite, `GatedReaderLosesNothing` once | same two proofs (sha256 differs; vet `rc=0`) |
| MUT-2b (H1 seam) | Make `capturePipelineStderrWith` ignore `opts.release` — the hook is never invoked | `rc=1` (1 run): the gate never opens, the bounded wait expires in the test's `drainTimeout: 2s`, the helper panics `… did not drain …` | same suite, `GatedReaderLosesNothing` once | same two proofs |
| MUT-3a (H1 teardown) | Remove the `defer` — inline the teardown after `f()` instead of in a deferred closure | `rc=1` (1 run): a panicking `f()` skips the inlined teardown; `os.Stderr` stays swapped; the restore assertion is red | `go test ./internal/pipeline/ -run TestCapturePipelineStderr_PanicRestoresStderr -count=1` | same two proofs |
| MUT-3b (H1 teardown) | Replace the bounded `select` wait with a bare receive: `copyErr = <-copied` | `rc=2` (1 run) via `go test`'s own timeout with a goroutine dump — the never-released wrapper parks the bare wait forever. **Declared non-hermetic** (red in ≤60 s, not instantly) | `go test ./internal/pipeline/ -run TestCapturePipelineStderr_DrainTimeoutIsLoud -count=1 -timeout 60s` | same two proofs |
| MUT-4 (H1 teardown, r2 astra) | Discard the copy error: `_, _ = io.Copy(&buf, src); copied <- nil` in the copier goroutine | `rc=1` (1 run): the sentinel read error is swallowed, a partial capture returns as success, the "panics with the sentinel" assertion is red. Result recorded as **V21** | `go test ./internal/pipeline/ -run TestCapturePipelineStderr_CopyErrorIsLoud -count=1` | same two proofs |

A mutant that is **not** red (after the two proofs pass) is a design defect to report back to the
controller — never a parameter to tune. There are no sleeps, byte caps, or iteration counts to raise.

**M1 gates (all runnable in the executor's sandbox — no network, no sockets; `go build ./...` is
deliberately absent, `rc=1` at base per V11):**

- G0 (preflight, before any edit): re-run the baseline battery — AC1, AC4, AC5 commands — all three
  must be `rc=0`; record stdout tails + rcs to `.snap/drill/baselines.txt`. A red baseline means the
  sandbox is broken, not the diff: stop and report.
- G1 (fixed-form green): AC2 exactly as written — `rc=0` **and** all four `--- PASS:` lines present.
  Also AC1 and AC4 `rc=0`, `gofmt -l internal/pipeline/pipeline_module_phases_test.go [+ sibling if
  used]` prints nothing.
- G2 (drill): all six table rows executed with landing-hash, vet-rc, test-rc evidence; every restore
  hash-checked equal to its pre-mutation hash; final restored tree re-runs AC2 `rc=0`.
- G3 (handoff): snapshot every created-or-modified file to `.snap/M1/` (see Lane Rules), update the
  sprint JSON (`features[0].passes = true`, note with the drill rc summary).

**Risks:** (i) H2 file placement crosses the 500-LOC comfort band → mitigated: sibling file is
pre-authorized by the doc. (ii) Bounded-wait panic fires spuriously on a loaded runner → 10 s default
vs. a microsecond drain makes this ~impossible, and the panic is loud with named causes. (iii) MUT-3b's
kill is the `go test` timeout (non-hermetic) → declared as such in the doc and above; one run, 60 s cap.

### M2: Changelog line + mutation-audit record + acceptance sweep (~1.5 hours, ~43 LOC)

**Goal:** Land hunk H3 and make the drill auditable: one `### Fixed —` changelog line (per D-2, under a
new `## [Unreleased]` header at the top of `changelogs/v0.32-current.md`), plus a new
record `design_docs/planned/v0_38_6/m-debugcacheforms-flaky-on-macos-ci-mutation-audit.md` (~40 LOC)
containing: the rc of every mutant run (MUT-1 ×5, MUT-2a/2b/3a/3b/4 ×1 each, incl. the vet `rc=0` and
landing-sha256 lines), the **exact edit that constituted each mutant** (one-line diff-style
descriptions from the M1 table), each restore-hash verification, the fixed-form AC2 output tail, and
the V21 row (MUT-4 / `CopyErrorIsLoud` and the MUT-1 panic text) for transcription into the design
doc's Verification Log. Then re-run the full acceptance sweep on the final tree.

**Estimated:** +1–3 LOC changelog (D-2) + ~40 LOC audit record.
**Dependencies:** M1

**Acceptance criteria owned by M2 (verbatim where the doc has a checkbox):**

- [ ] **AC5** `go build ./internal/pipeline/ && go vet ./internal/pipeline/` → `rc=0` (baseline
  `rc=0`, V11).
- [ ] All tests passing
- [ ] Changelog updated (H3)

*Sweep mapping:* "All tests passing" is evidenced in the sandbox by `go test ./internal/pipeline/ -count=1`
(`rc=0`, the doc's Example-2 whole-package green) plus the AC1/AC2/AC4 re-runs; the repo-wide suite
runs on the PR in CI, which the controller reads — it is not an executor gate (the executor has no
network). The sweep command list: AC1, AC2, AC3's restored-tree re-check (`rc=0`), AC4, AC5,
`gofmt -l` on every touched file — paste all outputs into `.snap/M2/acceptance-sweep.txt`.

**AC6 is NOT executor-owned.** Copied verbatim for completeness:

- [ ] **AC6 — CI observation, stated as such.** The mechanism is a scheduling race; local greens
  certify nothing about the runner. After the merge commit lands on `dev`, the controller measures
  with `gh run list --workflow "Build and Release" --branch dev --limit 25` and, for every
  `Build macos-latest` job that concludes `failure`, reads the job log for
  `--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters`. **Pass:** zero such jobs across
  25 consecutive post-landing runs (≤50 macOS job executions), counting only jobs in which the test
  reached a `--- PASS`/`--- FAIL` line (fail-fast-cancelled sibling legs are neither).
  **Baseline:** 4 attributable failures in the 25 pre-landing runs (≤50 macOS executions, ~8% per
  execution, V2), plus a 5th on merge SHA `45f02deb3` (job `102044971960`). **Power, honestly:**
  at ~8% per execution a clean 50-execution streak has ~1.5% (0.92^50) probability with no fix, so
  AC6 is strong corroboration but still not proof — AC2/AC3 are the proof. A single green re-run of
  a failed job is NOT evidence of anything (the sibling macOS job in the failing run was green at
  the same commit, V2).

Ownership: the **controller** runs AC6 after merge (it needs `gh` + network + time over 25 runs). The
executor does not run AC6, is not graded on it, and the sprint closes for the executor at M2.

**M2 gates:**
- G4: `grep -n 'capturePipelineStderr\|### Fixed' changelogs/v0.32-current.md | head -3` shows the new
  line under `## [Unreleased]`; `wc -l design_docs/planned/v0_38_6/*mutation-audit.md` ≥ 30 and the file
  contains all six mutant rows with rcs; `jq -e .` on the sprint JSON (`features[1].passes = true`).
- G5: acceptance sweep outputs all green, captured in `.snap/M2/acceptance-sweep.txt`.
- G6 (handoff): cumulative snapshot to `.snap/M2/` (see Lane Rules).

## Lane Rules (binding on the executor)

1. **The executor is a cross-provider lane with NO git write access.** It must **not** run
   `git add`, `git commit`, `git stash`, `git checkout`, `git reset`, or `git push` — in particular
   never `git checkout -- <file>` to undo anything (the M1 edits are uncommitted; a checkout would
   delete them). All mutation-drill restores are **copy-based**: `cp file file.bak` before the mutant,
   `cp file.bak file` after, then assert `shasum -a 256 file` equals the pre-mutation hash, then
   `rm file.bak`. (`shasum -a 256` is at `/usr/bin/shasum` here; `sha256sum` is an acceptable
   equivalent on a Linux sandbox — record which was used.)
2. **The controller commits, one commit per milestone** (M1 commit = the phases/sibling `_test.go`
   changes; M2 commit = changelog + audit + sweep evidence). The executor hands off by completing the
   milestone gates and snapshotting; the controller reviews `.snap/M<k>/` before committing it.
3. **Snapshot after every milestone, cumulative.** After M1: every file created or modified by the
   executor exists again under `.snap/M1/` preserving its repo-relative path, plus a `SHA256SUMS`
   file (`shasum -a 256 <each file>` output). After M2: `.snap/M2/` contains **everything** from
   `.snap/M1/` plus the changelog and the audit record (cumulative, so `.snap/M2/` is the full final
   tree of touched files). Drill scratch (`.snap/drill/`) is evidence, not a deliverable. Example:
   `mkdir -p .snap/M1/internal/pipeline && cp internal/pipeline/pipeline_module_phases_test.go .snap/M1/internal/pipeline/ && (cd .snap/M1 && shasum -a 256 internal/pipeline/pipeline_module_phases_test.go > SHA256SUMS)`.
4. **Sandbox gates only.** Every gate above is `go test ./internal/pipeline/ …`,
   `go build ./internal/pipeline/`, `go vet ./internal/pipeline/`, `gofmt -l <files>`, `grep`, `cp`,
   `mkdir`, `shasum`, or `jq`. No network, no sockets, no `gh`. `go build ./...` is excluded
   everywhere (`rc=1` at base — `cmd/wasm` has no native `main`, V11).
5. Nothing is tuned during the drill. A mutant that survives (passed both landing/build proofs and
   still `rc=0`) is a design defect: stop, snapshot, report to the controller.

## Day-by-Day (Day 1, ~4 h)

- **M1 (≈2.5 h):** H1 helper seam + named-return `defer` reorder (0.5 h) · H2 `gatedReader` + four
  tests (1 h) · six-mutant drill with sha256/vet/rc evidence + restores (1 h).
- **M2 (≈1.5 h):** changelog line under `## [Unreleased]` per D-2 (0.25 h) · mutation-audit record incl.
  V21 (0.5 h) · acceptance sweep + cumulative snapshot + sprint-JSON update (0.75 h).
- **Controller (outside the 4 h):** review `.snap/`, commit M1 then M2, open/merge the PR, start the
  AC6 watch (25 consecutive `dev` runs); do not close the mission row on AC6 alone.

## Success Metrics

- AC1–AC5 green in the sandbox; AC2's four `--- PASS:` lines present; AC3's drill fully red-then-green
  with sha256 evidence; AC6 passed to the controller with its stated power (~1.5% false-clean
  probability at the measured base rate).
- Documentation: `changelogs/v0.32-current.md` (+1 entry line), `…-mutation-audit.md` (new, ~40 LOC).

## Dependencies

None in-repo. External: the CI watch (AC6) depends on GitHub Actions and is the controller's.

## Open Questions

None for the executor — the doc's Design Freeze is exhaustive (teardown order, 10 s bounded wait,
≤4 KiB payload assertion, nil-opts invariant, Windows `t.Skip` untouched at `:201-215`). Deferred to
the implementer per the doc: H2 file placement, `N ≥ 50` exact value, seam/struct/field names.

## Notes

- H2 is declared as "H2 _is_ the killer" in the doc's mutation table and H3 carries no mutation
  (prose) — hence no MUT rows for those hunks above.
- The `.snap/` tree is executor-produced evidence in the worktree; the controller decides whether to
  commit it. The durable, always-committed evidence is the mutation-audit record (M2).
- Planner: kimi-k3 (V1 iter 353, planner lane, worktree `b794560ca`). Design doc: quorum-approved r3.
