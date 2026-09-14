# Sprint Evaluation — m-debugcacheforms-flaky-on-macos-ci (round 2)

**Evaluator:** Sonnet 5 (independent; round 1 was PASS 88/100, zero blocking, at `36a5cc8ee`)
**Reviewed SHA:** `e15d27a3b` (delta since r1: `git diff --stat 36a5cc8ee e15d27a3b` = 3 files,
`internal/pipeline/pipeline_stderr_capture_test.go` +5/−2, the design doc +2/−1, the mutation-audit
doc +15)
**Base:** `05a6457d1` (= `origin/dev`)
**Platform:** darwin/arm64, go1.26.6, Darwin 25.6.0 (Mac Studio). A local green still certifies
nothing about the `macos-latest` GitHub Actions runner (AC6, controller-owned, out of scope here).

All round-1 numbers were re-derived from scratch on this tree; none carried forward.

## Verdict

**SCORE 98/100 — PASS**

No blocking findings. All five non-blocking round-1 findings are CLOSED: each was fixed with a
narrow, correct patch, and I independently reproduced every claim the patches make — including two
new mutant re-runs and one restore-skip probe on the current tree — before accepting the doc text
as true of HEAD.

## Score breakdown by rubric category

| Category | Points | Score | Notes |
|---|---|---|---|
| Tests Pass | 20 | **20** | AC1/AC2/AC4 all rc=0 on this tree; `gofmt -l` clean on both touched files |
| Lint Clean | 10 | **10** | `golangci-lint run ./internal/pipeline/...` → `0 issues.` |
| Acceptance Criteria | 30 | **30** | AC1/AC2/AC4 reproduced exactly; the AC3 gaps from r1 (incomplete red-set characterization) are now resolved — the docs state the same red sets I independently re-measured on this tree |
| Code Quality | 15 | **15** | The r1 fix is a minimal, correct 4-line diff (`original := os.Stderr` + `os.Stderr != original`); no regression, no scope creep, restores byte-identical |
| Documentation | 15 | **13** | V21 transcribed accurately, MUT-3b's `rc=2`→`rc=1` corrected, a `judge r1` quorum-log row added — all verified true of HEAD. −2: the correction lives in a bolted-on "Round-1 judge corrections" section in the audit doc rather than folded into the original per-mutant rows, and finding 6 (the unverifiable "first gatedReader draft" historical claim) remains unaddressed (informational, not one of the five adjudicated below) |
| Design Fidelity | 10 | **10** | Fix matches exactly what round 1 asked for; no unrelated changes |
| **Total** | **100** | **98** | **PASS** |

## Blocking findings

None.

## Required work — results

### 1. AC2 (fixed form), AC1, AC4

| Gate | Command | rc | Evidence |
|---|---|---|---|
| AC2 | `go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=1 -timeout 60s -v` | **0** | `--- PASS: TestCapturePipelineStderr_GatedReaderLosesNothing`, `--- PASS: TestCapturePipelineStderr_PanicRestoresStderr`, `--- PASS: TestCapturePipelineStderr_DrainTimeoutIsLoud (0.05s)`, `--- PASS: TestCapturePipelineStderr_CopyErrorIsLoud`; `ok ... 0.539s` |
| AC1 | `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` | **0** | `ok ... 0.506s` |
| AC4 | `go test ./internal/pipeline/ -run TestPipelineModulePhases_DebugCacheFormsAndCounters -count=20 -race` | **0** | `ok ... 3.175s` |
| sanity | re-ran AC2 once more after all mutation-drill restores below | **0** | `ok ... 0.397s`, all four `--- PASS` lines |

### 2. Finding-3 drill, re-run on this tree

- Backed up `internal/pipeline/pipeline_module_phases_test.go` (sha256
  `280368400e3d5c04...` — confirmed **byte-identical to round 1's helper file**; the r2 delta only
  touched `pipeline_stderr_capture_test.go`).
- Applied the restore-skip mutant (commented out `os.Stderr = old`, the sole line removed): landed
  (sha256 changed to `ba42017e775a...`), `go vet ./internal/pipeline/` → **rc=0** (mutant builds).
- Ran `go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1 -timeout 60s -v`:
  **rc=1**, `pipeline_stderr_capture_test.go:104: os.Stderr was not restored to the original after
  the loop` — exactly the text the r2 commit message claims, and the test is now RED where in round
  1 it was a silent PASS under the identical mutant.
- Restored: `shasum -a 256` back to `280368400e3d5c04...` — confirmed equal. `git status --porcelain`
  clean (only the r1 report, an untracked evaluation artifact, remains). Re-ran AC2 post-restore:
  rc=0, four `--- PASS` lines.

### 3. Is the new assertion vacuous another way?

`os.Stderr != original` is a **pointer-identity** comparison (`*os.File`), and `original` is a local
variable holding a live reference captured before the loop — so it cannot be garbage-collected or
have its address reused by an unrelated `os.Pipe()` allocation while the test still references it
(Go's memory safety guarantees no two live, distinct objects alias the same address). For the check
to spuriously report equality while the restore is genuinely broken, one of these would have to be
true, and none is:
- A concurrent test reassigning `os.Stderr` back to the true original mid-check — ruled out: these
  tests are explicitly sequential (doc comment, no `t.Parallel()` anywhere in the package per V8),
  and the check runs in the same goroutine immediately after the loop with no yield point that could
  let another top-level test interleave.
- The allocator reusing `original`'s exact address for the last iteration's fresh pipe writer — ruled
  out by the live-reference argument above.
- `f()` itself touching the identity of `os.Stderr` — ruled out; `f` only calls
  `io.WriteString(os.Stderr, payload)`, a write through the current pointer, never a reassignment.

No other path can make `os.Stderr == original` hold except the intended restore actually running.
The assertion is sound.

### 4. Are the three doc changes true of e15d27a3b?

Re-measured rather than trusted:

- **V21** (design doc): claims fixed-form = four `--- PASS:` lines rc=0 (✓, AC2 above); MUT-4 rc=1
  with `expected the helper to panic on a non-nil copy error` (✓ — re-checked in round 1 and the
  underlying file is unchanged, confirmed byte-identical this round); MUT-1 panic text `capturePipelineStderr: copier failed: read |0: file already closed` (✓ — re-applied MUT-1 fresh on
  this tree, exact match, plus the `-skip` run still reddens `TestPipelineModulePhases_DebugCacheFormsAndCounters`, confirming the claimed red set `{GatedReaderLosesNothing,
  DebugCacheFormsAndCounters}` holds on e15d27a3b); MUT-2a red set
  `{GatedReaderLosesNothing, DrainTimeoutIsLoud, CopyErrorIsLoud}` (✓ — re-applied MUT-2a fresh,
  `-skip` of all three → rc=0, confirming no other test is affected); MUT-2b/MUT-3a/MUT-4 sole-killers
  and MUT-3b non-hermetic — unchanged from round 1, same helper file, not re-run a fourth time (code
  path untouched by the delta).
- **Mutation table MUT-3b row** (design doc): now states `rc=1` with `panic: test timed out after
  1m0s`. Re-applied MUT-3b fresh on this tree: **rc=1**, exact panic text match. The row's added claim
  that a real CI (`go test ./...`, no per-package `-timeout`) would hang the whole
  `internal/pipeline` binary to `go test`'s default is correct — Go's documented default test
  timeout is 10 minutes absent a `-timeout` flag, and V14 confirms the CI invocation carries none.
- **Mutation-audit "Round-1 judge corrections" section**: every bullet cross-checked above (red
  sets, restore-to-original fix + rc, MUT-3b rc) is true of the tree at e15d27a3b.

No number in any of the three doc changes was found to be false or unsupported.

## Disposition of the five round-1 findings

1. **Design doc Verification Log V21 stale ("UNMEASURED at r3")** — **CLOSED**. V21 now carries the
   real measured content; re-verified true of HEAD in §4 above.
2. **Uncorrected `rc=2` claim for MUT-3b** — **CLOSED**. Design doc now says `rc=1`; re-measured
   `rc=1` independently on e15d27a3b.
3. **Near-vacuous `os.Stderr == nil` assertion in `GatedReaderLosesNothing`** — **CLOSED**. Replaced
   with `os.Stderr != original`; proven RED against the exact restore-skip mutant that round 1 used
   to demonstrate the old check's blindness, and confirmed not vacuous by any other path (§3).
4. **MUT-1 / MUT-2a blast radii understated in the docs** — **CLOSED**. Both docs now record the
   full red sets I measured; re-applied both mutants fresh this round and the recorded sets match
   exactly (no member added or missing).
5. **MUT-3b "≤60s, cheap" framing understated the real-CI hang risk** — **CLOSED**. The design doc's
   mutation table and the audit's corrections section now explicitly state the un-scoped-CI blast
   radius (hang to `go test`'s default 10-minute timeout), matching what I measured in round 1 and
   re-confirmed is an accurate default-timeout claim.

## Notes

- Restore discipline held throughout: every mutant in this round was `cp`-restored and hash-verified
  equal to the pre-mutation SHA-256 before moving to the next; final `git status --porcelain` shows
  only the untracked r1 report, no diff against e15d27a3b.
- Finding 6 from round 1 (the unverifiable claim about an early `gatedReader` draft wrapping a
  `strings.Reader`) was informational and not one of the five requiring adjudication; it remains
  unverifiable from this worktree (no `.snap/` present) but is consistent with the final code and is
  not counted against the score.
