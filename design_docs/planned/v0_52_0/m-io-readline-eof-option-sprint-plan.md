# Sprint Plan: M-IO-READLINE-EOF-OPTION

## Summary
Implement the approved additive `std/io.readLineOpt` API so line consumers distinguish blank lines from EOF and retain final unterminated content. Preserve the existing `readLine` contract.

**Design:** [m-io-readline-eof-option.md](m-io-readline-eof-option.md)  
**Source task:** task-98827762  
**Duration:** 2 days, 12 hours (9 hours work + 3 hours contingency).  
**Estimated total:** 400 LOC (220 implementation/docs/example + 180 tests); new prompt copies are excluded from net LOC.  
**Risk:** Medium, primarily buffered-reader sharing and prompt publication.  
**State:** Planned; implementation awaits sprint approval through the coordinator pipeline.

## Current status and velocity
The working tree was clean before planning. Source inspection confirms the new name is absent and `ioReadLine` still swallows EOF. The approved design includes a systemic audit: async stdin closes its channel at EOF, and FS/Net/Process do not expose another ambiguous line reader. No wider reader refactor is needed.

The velocity script for the last seven days found only one visible commit (a docs dependency bump) and no usable diff metrics in this checkout. Historical changelog grep returned old entries rather than dated velocity; these are not evidence of current throughput. Use a conservative planning capacity of 200 net LOC/day, not a measured velocity. The design's 12-hour estimate is retained with 25% of the time explicitly reserved for integration uncertainty.

`std/VERSION` and the current changelog now say v0.52.0, superseding the design's v0.51.1 snapshot. Keep the approved artifact paths, add the feature under Unreleased, and select the actual shipping version at release review before finalizing builtin Since metadata. Do not bump the release in this sprint.

## Registry reuse audit
On 2026-10-03, `ailang pkg search stdin` returned no packages; `ailang pkg search io` returned application packages, none exposing a runtime stdin primitive. No suitable candidate warrants pkg info/docs inspection. The binary warned it may be stale; registry results are advisory and execution must rebuild locally. A package cannot recover EOF after the existing runtime discards it.

| Milestone | Decision | Rationale |
|---|---|---|
| M1 | none | Requires a core IO effect operation; reuse existing std/option and shared reader directly. |
| M2 | none | Tests and registrations belong to the host trace/eval harness, not a package. |
| M3 | none | Repository example, prompt and docs must teach the host API; no dependency needed. |

## Milestones

### M1: EOF-aware IO surface (~210 LOC)
**Estimated:** 100 implementation/docs + 110 tests.
**Dependencies:** none
**Budget:** 4 hours

**Files:** `internal/effects/io.go`, `internal/effects/io_test.go`, `internal/builtins/io.go`, `internal/builtins/io_test.go`, `std/io.ail`.

Load builtin-developer before implementation and run ailang prompt before editing std/io.ail. Add the effect handler, BuiltinSpec/type/metadata and wrapper. Write semantic and capability tests, including injected non-EOF errors, argument validation and interleaving both reader operations. Run focused effects/builtins tests, build, doctor builtins and check the example probe.

**Example coverage:** M3 creates the consumer fixture; M1 checks a temporary probe using Option matching.

**Acceptance criteria:**
- [ ] IO.readLineOpt and _io_readLineOpt return std/option Option[string] through effects.Call with the existing IO gate and unit argument convention.
- [ ] Normal and blank lines return Some; zero-byte EOF returns None; final unterminated content returns Some once, then repeated None.
- [ ] CRLF trimming matches readLine; non-EOF reader errors propagate; mixed reads share ctx.GetIOReader without losing buffered data.
- [ ] Existing TestIOReadLine_* pass unchanged; builtin type, metadata and doctor validation pass.

**Risk and mitigation:** Separate buffered readers would lose data; require ctx.GetIOReader and a dedicated mixed-read test. Mirror existing trimming without refactoring legacy behavior.

### M2: Trace, harness and CLI regression coverage (~80 LOC)
**Estimated:** 10 implementation/docs + 70 tests.
**Dependencies:** M1
**Budget:** 2 hours

**Files:** `internal/trace/schema.go`, `internal/trace/schema_test.go`, `internal/eval_harness/normalize.go`, `internal/eval_harness/normalize_test.go`, `cmd/ailang/ (existing CLI integration test suite)`.

Add trace and normalization registrations with focused tests. Reuse the existing CLI integration harness for stdin cases with bounded execution. Cover normal/blank input, empty input, final unterminated input and frozen legacy behavior.

**Example coverage:** Run the M3 fixture through the stdin integration harness once available.

**Acceptance criteria:**
- [ ] IO.readLineOpt is classified non-deterministic with focused trace coverage.
- [ ] Eval normalization recognizes readLineOpt with parenthesized and spaced call forms and imports std/io correctly.
- [ ] Pipe-driven CLI tests deliver a, blank, b for a newline-separated input and terminate; empty input exits immediately; unterminated final input is retained.
- [ ] Integration tests use a timeout to detect EOF spinning and assert output and exit status.

**Risk and mitigation:** A missing op registration or harness import silently undermines parity; test both explicitly and exercise the real CLI.

### M3: Runnable example, teaching and release validation (~110 LOC)
**Estimated:** 110 implementation/docs + 0 tests.
**Dependencies:** M1, M2
**Budget:** 3 hours

**Files:** `examples/runnable/io_readline_eof.ail`, `examples/manifest.json`, `docs/docs/reference/effects.md`, `changelogs/v0.32-current.md`, `cmd/ailang/prompts/, prompts/, docs/docs/prompts/ (new version and prompt-manager required indexes/hash metadata)`.

Load prompt-manager for new prompt publication and obtain the current teaching prompt before writing the example. Add a headless-safe echo loop, manifest entry, reference rows and changelog. Run final gates and record results for sprint-evaluator.

**Example coverage:** Create and verify examples/runnable/io_readline_eof.ail, including empty-stdin execution.

**Acceptance criteria:**
- [ ] examples/runnable/io_readline_eof.ail and examples/manifest.json are present, type-check, preserve blank lines, and exit on empty stdin.
- [ ] docs/docs/reference/effects.md documents readLineOpt and cross-references it from readLine; Unreleased changelog records the additive API.
- [ ] A new teaching prompt version is created through prompt-manager; historical prompt hashes remain unchanged and prompt freeze checks pass.
- [ ] make build, make test, make lint, make check-boundaries, make verify-examples and make check-prompt-freeze pass; regex_capture example remains working.

**Risk and mitigation:** Prompt freeze prohibits editing historical versions; use prompt-manager and run the freeze gate. Release numbering is reviewed before metadata publication.

## Day-by-day execution
- Day 1 (6 hours): M1 (4h), M2 (2h); focused tests after each change.
- Day 2 (6 hours): M3 (3h), integration fixes and final gates (3h contingency). Do not widen scope to VM or Go codegen wiring.

## Success metrics and verification
All new EOF branches and non-EOF error handling have focused assertions; no aggregate coverage percentage is invented for this checkout. The runnable example preserves blank lines and terminates immediately under headless verification. Legacy readLine tests remain unmodified. Run focused package tests first, then the final gates listed in M3 once the implementation is complete; collect output for sprint-evaluator. Run regex_capture using its existing module/capability flags as the Option-builtin regression fixture.

## Dependencies, assumptions and handoff
Existing std/option, effects.Call and ctx.GetIOReader are sufficient; no new external dependency or capability. Registry results do not justify a package abstraction. Unit-argument internals preserve the public zero-argument syntax. Non-EOF errors remain errors, even when accompanied by bytes.

The feature is interpreter-path parity with readLine; VM and compiled-mode support are outside this approved scope. Choose Option-only import unless verified house style favors constructor imports. Prompt version selection and shipping version are executor/release-review decisions; neither blocks planning.

The coordinator consumes the plan and progress JSON markers, presents sprint approval, and routes approved work to sprint-executor. No implementation or direct executor dispatch occurs during this planning stage. On completion of execution, use sprint-evaluator against the design and these criteria.
