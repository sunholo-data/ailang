# Sprint Evaluation — m-debugcacheforms-flaky-on-macos-ci (round 1)

**Evaluator:** Sonnet 5 (independent, has not seen the design/plan/audit authorship — generator ≠ judge)
**Reviewed SHA:** `36a5cc8ee` (worktree `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter353-eval`, detached)
**Base:** `05a6457d1` (= `origin/dev`)
**Platform:** darwin/arm64, go1.26.6, Darwin 25.6.0 (Mac Studio). **A local green — including this evaluation's
green — certifies nothing about the `macos-latest` GitHub Actions runner named in AC6; AC6 is
controller-owned and out of scope here.**

## Verdict

**SCORE 88/100 — PASS**

No blocking findings. The fix is mechanistically sound, matches the quorum-approved r3 design
almost verbatim, and every executor-owned acceptance command (AC1–AC5) reproduces exactly as
claimed — including, in one case, a byte-identical mutation-landing SHA-256 between my
independently-authored mutant edit and the audit's recorded hash (MUT-4). The deductions below are
evidence-completeness and test-quality gaps, not correctness defects.

## Score breakdown by rubric category

| Category | Points | Score | Notes |
|---|---|---|---|
| Tests Pass | 20 | **20** | AC1/AC2/AC4/AC5 all rc=0; whole package green under `-race` (332 pass, 0 fail) and under `-count=2` (664 pass, 0 fail, no leakage) |
| Lint Clean | 10 | **10** | `golangci-lint run ./internal/pipeline/...` → `0 issues.`; `gofmt -l` on both touched `.go` files → clean |
| Acceptance Criteria | 30 | **28** | All 5 executor-owned ACs objectively met exactly as written. −2: AC3's mutation drill, while structurally correct, is under-characterized — MUT-1 and MUT-2a both have a broader red set than the single "killer test" the doc/audit name (see Mutation drill table) |
| Code Quality | 15 | **12** | Clean, well-commented, matches the Architecture sketch almost verbatim. −3: one demonstrated weak/near-vacuous assertion (§ Vacuous-assertion drill), and the mutation drill as executed did not surface the broader blast radii of MUT-1/MUT-2a |
| Documentation | 15 | **8** | Changelog entry (H3) is accurate, complete, and correctly resolves D-2 (`## [Unreleased]` header). −7: the design doc's own Verification Log was never updated post-sprint — V21 still reads "UNMEASURED at r3" despite the mutation-audit doc producing exactly the block intended for transcription (M2's stated goal), and the mutation table's MUT-3b row still claims `rc=2`, which the sprint's own commit message and the audit both record as wrong (`rc=1`) and which I independently reproduced as `rc=1` |
| Design Fidelity | 10 | **10** | Implementation matches the Design Freeze and Architecture sketch essentially line-for-line: named-return `defer`, writer→hook→bounded-wait→reader ordering, nil-opts production invariant preserved, Windows skip untouched, no scope creep beyond H1–H3 |
| Regression Surface Coverage (conditional) | — | N/A | Not triggered — `internal/pipeline/` is not in the trigger list |
| Performance Verification (conditional) | — | N/A | Not a perf sprint |
| **Total** | **100** | **88** | **PASS (≥70, no hard fails)** |

## Blocking findings

None.

## Non-blocking findings

1. **(Documentation, moderate) Design doc Verification Log not updated post-sprint.** The design
   doc at HEAD still has `V21 | ... | UNMEASURED at r3` (line 601) even though
   `m-debugcacheforms-flaky-on-macos-ci-mutation-audit.md` explicitly contains a "V21 block (for
   transcription into the design doc's Verification Log)" with the real, measured content. The
   sprint plan's M2 goal explicitly required this transcription. It did not happen (or happened and
   was reverted/lost) — a stale claim survived into the doc of record.
2. **(Documentation, moderate) Uncorrected factual error in the design doc's mutation table.** Line
   415 of the design doc still states MUT-3b "exits `rc=2` with a goroutine dump." The M2 commit
   message (`36a5cc8ee`) records a deviation that the real value is `rc=1`, the mutation-audit doc
   records `rc=1`, and I independently reproduced `rc=1` three times (once as part of the six-mutant
   drill, once in a clean re-verification). The design doc was never corrected to match the sprint's
   own finding.
3. **(Test quality, minor, empirically demonstrated) One near-vacuous assertion.**
   `TestCapturePipelineStderr_GatedReaderLosesNothing`'s post-loop check
   (`if got := os.Stderr; got == nil { t.Fatal(...) }`) does not verify restoration to the *original*
   `os.Stderr` — only that it is non-nil. I mutated the fixed-form helper to skip the
   `os.Stderr = old` restore line entirely and re-ran all four `TestCapturePipelineStderr_*` tests:
   `PanicRestoresStderr`, `DrainTimeoutIsLoud`, and `CopyErrorIsLoud` all correctly failed
   ("os.Stderr was not restored ..."), but `GatedReaderLosesNothing` **passed** — its own restore
   check is blind to this exact defect class. Not suite-blind (3 of 4 tests catch it), but the
   assertion does not test what its position implies.
4. **(Evidence completeness, minor) MUT-1 and MUT-2a blast radii understated.** See the Mutation
   drill table below — both mutants redden more than the single test the design/audit name as the
   "killer." This does not weaken the fix (if anything it shows more redundant coverage than
   claimed); it is an audit-completeness gap in how the drill's results were recorded.
5. **(Scope-of-claim, minor) MUT-3b's "declared non-hermetic, red in ≤60s" understates the real-CI
   blast radius.** That characterization is only true of the scoped acceptance command
   (`-run TestCapturePipelineStderr_DrainTimeoutIsLoud -timeout 60s`). The actual CI invocation
   (V14: `go test -v $(go list ./...)`, no per-package `-timeout` override) would use `go test`'s
   default 10-minute timeout, and — because the panic is unrecovered inside the test's own goroutine
   — a full-package run hangs *the entire test binary*, not just the one test: my whole-package
   repro under this mutant, bounded at `-timeout 90s`, showed 205 of 332 tests reported before the
   process was killed; the remaining ~127 tests (including two of the other three new regression
   tests) never ran and their status is simply unknown, not "passed." This is an acceptable,
   declared trade-off for a loud-failure design (Principle 2) but the "≤60s, cheap" framing doesn't
   carry over to the real CI command.
6. **(Unverifiable claim, informational)** The M2 commit message and mutation-audit doc describe
   an executor deviation ("the first gatedReader draft wrapped a `strings.Reader` and would have
   made MUT-1 spuriously green — corrected before the drill"). The `.snap/` executor evidence tree
   is not present in this worktree (per the mission brief), so this specific historical claim about
   an earlier draft cannot be independently checked. The final code is consistent with the stated
   fix (the wrapper installs the pipe's own read end, per the doc comment at
   `pipeline_stderr_capture_test.go:34-38`), so this is not treated as a discrepancy — just noted as
   unverifiable from the artifacts available to this evaluation.

## Gate re-run table (darwin/arm64, go1.26.6)

| Gate | Command | rc | Load-bearing output |
|---|---|---|---|
| AC1 | `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` | 0 | `ok ... 0.719s`; all 8 subtests PASS incl. `DebugCacheFormsAndCounters` |
| AC2 | `go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=1 -timeout 60s -v` | 0 | `--- PASS: TestCapturePipelineStderr_GatedReaderLosesNothing`, `--- PASS: TestCapturePipelineStderr_PanicRestoresStderr`, `--- PASS: TestCapturePipelineStderr_DrainTimeoutIsLoud (0.05s)`, `--- PASS: TestCapturePipelineStderr_CopyErrorIsLoud` — byte-for-byte the audit's claimed tail |
| AC3 | six-mutant drill | see Mutation drill table | — |
| AC4 | `go test ./internal/pipeline/ -run TestPipelineModulePhases_DebugCacheFormsAndCounters -count=20 -race` | 0 | `ok ... 3.167s` |
| AC5 | `go build ./internal/pipeline/ && go vet ./internal/pipeline/` | 0 / 0 | no output (clean) |
| extra | `go test ./internal/pipeline/ -race -count=1` (whole package) | 0 | `ok ... 25.015s`, 332 PASS / 0 FAIL |
| extra | `go test ./internal/pipeline/ -count=2` (whole package) | 0 | 664 PASS / 0 FAIL — hermetic across a doubled run |
| extra | `go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=2 -race -timeout 60s` | 0 | 8 PASS (4×2), no DATA RACE |
| extra | `golangci-lint run ./internal/pipeline/...` | 0 | `0 issues.` |
| extra | `gofmt -l` on both touched `.go` files | — | no output (clean) |

## Mutation drill table

Every mutant: copy-restored (`cp`/`sha256`), never `git checkout`. Pre-mutation hash of
`internal/pipeline/pipeline_module_phases_test.go` at HEAD: `280368400e3d5c04...` (confirmed equal
to the audit's stated pre-mutation hash before any mutation, and confirmed equal again after every
restore below).

| Mutant | Landed (sha≠pre) | `go vet` rc | Designated test rc | Failure text matches audit | **True red set (measured via `-skip`)** | Sole-killer? |
|---|---|---|---|---|---|---|
| MUT-1 (×5 runs) | ✓ | 0 | 1×5 | ✓ exact (`copier failed: read \|0: file already closed`) | `{GatedReaderLosesNothing, TestPipelineModulePhases_DebugCacheFormsAndCounters}` — deterministically reproduced 3/3 extra solo runs. Whole-package run crashes the binary (unrecovered panic); 200/332 tests never ran | **No** — broader than claimed. Reverting the fix reintroduces the exact original production flake in the SAME test the sprint exists to protect, which is reassuring for the fix but means the audit's "killer: GatedReaderLosesNothing" is incomplete |
| MUT-2a | ✓ | 0 | 1 | ✓ exact (`reader wrapper never invoked (reads=0)`) | `{GatedReaderLosesNothing, DrainTimeoutIsLoud, CopyErrorIsLoud}` — confirmed via `-skip` of all three → rc=0 | **No** — 3-test blast radius, not 1. (`PanicRestoresStderr` is unaffected — it doesn't use `opts.wrap`) |
| MUT-2b | ✓ | 0 | 1 (2.0s, matches `drainTimeout: 2s`) | ✓ exact (`copier did not drain within 2s ...`) | `{GatedReaderLosesNothing}` only — confirmed via `-skip` → rc=0, 331 pass | **Yes** |
| MUT-3a | ✓ | 0 | 1 | ✓ exact (`os.Stderr was not restored after a panicking f()`) | `{PanicRestoresStderr}` only — confirmed via `-skip` → rc=0, 331 pass | **Yes** |
| MUT-3b | ✓ | 0 | 1 (non-hermetic, `go test`'s own `-timeout`, ~60.5s) | ✓ (`panic: test timed out after 1m0s`); rc=1 matches the audit's stated *deviation* from the plan's rc=2 — reproduced 3× | Declared non-hermetic; whole-package run at `-timeout 90s` hangs the ENTIRE binary, 205/332 tests reported before kill, remainder unknown (see Finding 5) | Sole-designated, but see Finding 5 for real-CI scope |
| MUT-4 | ✓ (**landing hash `4fc52002e415...706e` byte-identical to the audit's recorded hash** — independent confirmation of the audit's own evidence, not just its narrative) | 0 | 1 | ✓ exact (`expected the helper to panic on a non-nil copy error`) | `{CopyErrorIsLoud}` only — confirmed via `-skip` → rc=0, 331 pass | **Yes** |

All six restores verified byte-identical to the pre-mutation hash by `shasum -a 256` before
proceeding to the next mutant. Final restored tree: `git status --porcelain` clean, `git diff
--stat` empty against HEAD.

## Non-vacuity result (M1 diff reverted, new test file kept)

Reverting only `pipeline_module_phases_test.go` to its pre-M1 (`05a6457d1`) shape while keeping
`pipeline_stderr_capture_test.go` at HEAD produces a **package-level compile failure**, not a
run-time red:

```
internal/pipeline/pipeline_stderr_capture_test.go:85:10: undefined: capturePipelineStderrWith
internal/pipeline/pipeline_stderr_capture_test.go:85:37: undefined: captureOpts
internal/pipeline/pipeline_stderr_capture_test.go:148:2: undefined: capturePipelineStderrWith
internal/pipeline/pipeline_stderr_capture_test.go:148:29: undefined: captureOpts
internal/pipeline/pipeline_stderr_capture_test.go:182:2: undefined: capturePipelineStderrWith
internal/pipeline/pipeline_stderr_capture_test.go:182:29: undefined: captureOpts
```

All four new tests are swept into the same failure: `[build failed]`. Three
(`GatedReaderLosesNothing`, `DrainTimeoutIsLoud`, `CopyErrorIsLoud`) fail because they directly
reference the removed `capturePipelineStderrWith`/`captureOpts` symbols; the fourth
(`PanicRestoresStderr`) doesn't reference either symbol but is swept into the same compile failure
because Go compiles per-package, not per-test-function. **Verdict: non-vacuous — none of the four
new tests can pass, or even run, against the pre-M1 helper; they fail to COMPILE.**

## Vacuous-assertion drill

Walked every assertion in all four new tests, asking "what else could make this pass?":

- `GatedReaderLosesNothing`: exact-equality `got != payload` — solid (confirmed kills MUT-1/MUT-2a
  variants). `reads.Load() < 1` — solid (confirmed sole discriminator for MUT-2a's wrapper-bypass
  symptom within this test). **Post-loop `os.Stderr == nil` check — weak, see Finding 3, empirically
  demonstrated non-firing on a real restore-skip defect.**
- `PanicRestoresStderr`: `recover() == nil → Fatal` and `os.Stderr != original → Fatal` — both
  solid; the recover requirement means a non-panicking `f()` or a swallowed panic both correctly
  fail.
- `DrainTimeoutIsLoud`: requires a panic, requires the panic message to be a `string` containing
  `"did not drain"`, requires `os.Stderr` restored — all three independently falsifiable and I
  confirmed the first two fire correctly under MUT-2a's wrapper-bypass (a different message: no
  panic at all → `"expected the helper to panic on drain timeout"`).
- `CopyErrorIsLoud`: requires a panic, requires the message to contain the sentinel's exact error
  text, requires restore — solid; the sentinel text is generated fresh per test run
  (`errors.New("sentinel read failure")`) so this cannot pass by coincidence with an unrelated
  panic message.

No assertion in the four new tests is *unconditionally* true (all can be made to fail by some
mutation), so "vacuous" in the strict always-passes sense does not apply anywhere; Finding 3 is
reported as a weak/mis-targeted assertion rather than a true vacuity.

## Hermeticity

- Whole package, `-count=2`: 664/664 pass, no interference (332×2 exactly).
- Whole package, `-count=2 -race`, scoped to `TestCapturePipelineStderr_*`: 8/8 pass, no `DATA RACE`.
- `DrainTimeoutIsLoud` deliberately leaks one goroutine per run (parked forever on `<-gate`, since
  the gate is never released in that test). Confirmed this leaked goroutine does not touch shared
  state after being parked (it blocks before its first underlying `Read` and before incrementing its
  `reads` counter), so it cannot corrupt a later test's buffer or counter. `os.Stderr`/`r`/`w` are
  all restored/closed by the teardown *before* the panic in that path, matching the design doc's
  claim at line 415.
- MUT-3b's `-timeout` reliance: acceptable as a **declared** non-hermetic kill for the *scoped*
  acceptance command (AC2's `-timeout 60s`), but its blast radius under a real, un-timeout-scoped
  package run is a full-binary hang bounded only by `go test`'s default (10 min) or the CI job
  budget (30 min) — materially larger than "red in ≤60s" (Finding 5). I judge this acceptable to
  ship (it's inherent to a loud-panic design and the mutant would have to be reintroduced by a
  future edit, which the other three regression tests plus code review would likely catch first),
  but it should be called out explicitly rather than left implicit.

## Evidence-integrity notes

- **Changelog** (`changelogs/v0.32-current.md`): accurate. Every factual claim in the added
  `### Fixed` entry (helper location, defect mechanism, "4 of the last 25" figure, teardown
  ordering, bounded 10s wait, copy-error propagation, "four helper tests plus a six-mutant drill")
  is true of the tree at HEAD, verified above.
- **Mutation-audit doc**: every landing-hash/vet-rc/test-rc/failure-text claim I could
  independently re-derive was correct, including one byte-identical SHA-256 (MUT-4) between my
  independently-typed mutation and the audit's recorded hash — strong evidence the audit reflects a
  real run, not a fabricated one. Its two weaknesses are the incomplete blast-radius
  characterizations for MUT-1/MUT-2a (Finding 4) and the fact that the "V21 block ... for
  transcription" was produced but never actually landed in the design doc (Finding 1).
- **Design doc V-rows describing shipped code**: V1–V19 describe pre-implementation measurements
  and are out of scope for "true of the tree at HEAD" (they describe CI history and probes, not the
  shipped helper). **V20** describes a standalone probe (`/tmp/iter353_probe2/...`), explicitly
  external to the repo, and is accurately labeled as such — not a claim about HEAD. **V21** is the
  one V-row that should describe the shipped code's measured behavior and does not: it still reads
  "UNMEASURED at r3" (Finding 1). The **mutation table** (not a V-row, but a claims table describing
  the shipped code) has one uncorrected error: MUT-3b's `rc=2` (Finding 2).

## Summary of what changed the score from a clean pass

The implementation, its correctness, and its test coverage are all solid — every gate an
independent judge can re-run came back exactly as claimed, in one case down to a byte-identical
hash. The deductions are entirely about two documents (`m-debugcacheforms-flaky-on-macos-ci.md`'s
Verification Log/mutation table) not being reconciled with the sprint's own findings, plus one
weak assertion. None of these block landing; they're worth a fast follow-up edit to the design doc
(update V21, fix the `rc=2`→`rc=1` line) but are not gating this sprint.
