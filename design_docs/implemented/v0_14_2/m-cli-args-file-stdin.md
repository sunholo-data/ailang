# M-CLI: `-args-file`, Stdin Input, and Windows Runtime CI

**Status**: Implemented (v0.14.2, 2026-04-27)
**Target**: v0.14.2 (patch — Windows hotfix + lasting Windows CI signal)
**Priority**: P0 (Windows users currently cannot reliably pass JSON args, and we have no CI signal to catch the next Windows-only regression)
**Estimated**: ~3.5 hours (2h CLI fix + 1.5h Windows CI matrix)
**Dependencies**: None

## Two-Part Scope

This doc covers **two coupled changes** because they share a root cause:

1. **The bug** — `ailang run -args-json` is unusable on PowerShell.
2. **Why it shipped** — `ci.yml` runs all tests on `ubuntu-latest` only. `build.yml` cross-builds a Windows binary but never executes it. Fixing only (1) would leave us blind to the next Windows-only regression.

Both ship in v0.14.2.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure CLI plumbing; resolved JSON string is identical regardless of input source |
| A2: Replayability | 0 | No trace surface change |
| A3: Effect Legibility | 0 | No new effects; FS read happens in CLI layer, not program |
| A4: Explicit Authority | 0 | File path is provided explicitly by the operator on the command line |
| A5: Bounded Verification | 0 | No type-checking impact |
| A6: Safe Concurrency | 0 | Synchronous file read at startup |
| A7: Machines First | +1 | Eliminates a shell-quoting class of failure that currently makes AI agents loop on Windows |
| A8: Minimal Syntax | 0 | New flag mirrors existing `-allow-env-file` precedent |
| A9: Cost Visibility | 0 | No runtime cost change |
| A10: Composability | +1 | Stdin form composes with shell pipelines and `jq`/`echo`/here-strings on every shell |
| A11: Structured Failure | +1 | New error paths (file not found, invalid JSON from file/stdin) reuse existing JSON-decode error format |
| A12: System Boundary | 0 | No boundary semantics change |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — same JSON in, same execution out
- [x] A3 (Effects): No hidden side effects — FS read is at CLI argument-resolution time, before program load
- [x] A4 (Authority): No ambient access — operator names the file path explicitly
- [x] A7 (Machines First): Strengthens machine usability (AI agents on Windows currently fail this)

## Problem Statement

PowerShell rewrites or strips quotes before they reach the AILANG binary, so `ailang run -args-json '["robot"]' file.ail` fails on Windows in two ways:

1. **Quotes stripped → invalid JSON:**
   ```
   Error: failed to decode arguments: invalid JSON: invalid character 'A' looking for beginning of value
   ```

2. **Array re-tokenized into positional args → AILANG enters batch mode and treats the JSON content as filenames:**
   ```
   Error: cannot read file 'robot': open robot: The system cannot find the file specified.
   ```

**Current State:**
- Inline `-args-json '<json>'` is the only supported way to pass entrypoint arguments.
- A real Gemini agent on Windows looped through 5 PowerShell quoting strategies (single quotes, escaped quotes, variable bypass, here-string, temp file) and still could not get JSON args through to AILANG.
- Reported by the docparse maintainer (message `msg_20260426_221627_0d287ba3`, 2026-04-26).
- Current workarounds (`@'...'@` here-string, `cmd /c ailang run ...`) require human shell expertise that AI agents do not have.
- **Systemic gap:** `.github/workflows/ci.yml` runs every test job on `ubuntu-latest`. `.github/workflows/build.yml` cross-compiles a Windows binary but never executes it. Result: Windows-only failures (this bug, the recent opencode test failure in `bf113aba`, last week's path-mismatch in `4f4fa419`) ship to users and surface only via downstream feedback.

**Impact:**
- All Windows users of `ailang run -args-json` (PowerShell is the default shell on `windows-latest` GitHub Actions runners).
- All AI coding agents driving AILANG on Windows.
- Indirect impact on docparse's Windows test surface — they have **no Windows CI** today and want to add `windows-latest`, which will exercise this path.
- Every future change risks introducing a Windows-only regression with zero pre-merge signal until we add a Windows job to `ci.yml`.

## Goals

**Primary Goal:** Unblock Windows users today with a non-quoting input path, and add Windows runtime CI so the next Windows-only regression is caught at PR time, not by a downstream user.

**Success Metrics:**
- `ailang run -args-file args.json --entry main story_studio.ail` works identically on bash, zsh, PowerShell, and `cmd.exe`.
- `Get-Content args.json | ailang run -args-json - --entry main story_studio.ail` works on PowerShell.
- `.github/workflows/ci.yml` runs the Go test suite (`go test ./...`) on a `windows-latest` job on every PR and on pushes to `dev`/`main`. The job is **required** to merge.
- The new Windows job runs at least one PowerShell-invoked end-to-end smoke that exercises `-args-file` and `-args-json -` against a real example.
- Docparse maintainer can swap their AILANG invocation to use `-args-file` and confirms the agent loop disappears.
- One historical Windows-only failure is replayed locally against the new CI job to confirm it would have caught it.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add a separate `-args-file` flag (vs. overloading `-args-json` with `@path` syntax) | Mirrors existing `-allow-env-file` precedent; keeps `-args-json` semantics unchanged for existing users | human | design | low |
| Use `-args-json -` (single dash) as the stdin sentinel | Standard UNIX convention; unambiguous since `null` is the only literal default | human | design | low |
| Resolve to a single effective JSON string in `runCommand()` before calling `runFile` | Confines the change to one site; both batch and module paths benefit without refactoring | agent | design | low |
| Mutual exclusivity policy for `-args-json` and `-args-file` | Prevents silent override surprises; explicit error is friendlier than precedence rule | human | design | low |
| Add `windows-latest` job to `ci.yml` and mark it **required** in branch protection | Without "required", Windows breakage is advisory and gets ignored; this is the durable fix that prevents the next bug | human | design | med (branch-protection edit + cycle-time impact) |
| Run the **full Go test suite** on Windows, not just a smoke test | A smoke catches today's bug but misses tomorrow's; the marginal cost (~3-5 min CI time) is worth full coverage | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Use a separate `-args-file` flag (not `@path` overload of `-args-json`)
- [x] Stdin sentinel is `-args-json -` (not `-args-stdin` flag)
- [x] Specifying both `-args-json <non-default>` and `-args-file <path>` is an error (exit code 2, message: "specify exactly one of -args-json or -args-file")
- [x] Specifying both `-args-json -` and `-args-file <path>` is an error (same message)
- [x] File contents must be valid UTF-8 JSON; trailing newlines/whitespace are trimmed; empty file = error ("empty -args-file")
- [ ] **Maintainer (human) confirms:** Windows job in `ci.yml` will be marked required in branch protection after first green run. Without this confirmation, Windows CI is advisory only and the systemic fix is incomplete.
- [ ] **Maintainer (human) confirms:** acceptable to add ~3-5 min to PR CI wall-clock for Windows runtime tests.

## Solution Design

### Overview

Add two equivalent input paths for entrypoint JSON arguments:

1. **`-args-file <path>`** — read JSON from a file (cleanest; sidesteps shell quoting entirely).
2. **`-args-json -`** — read JSON from stdin (pipe-friendly, composes with `jq`/here-docs/here-strings).

Both resolve to the same internal `argsJSON string` value that `decodeEntrypointArgs` already consumes ([cmd/ailang/run_helpers.go:505](cmd/ailang/run_helpers.go#L505)). No change downstream of the CLI flag-parsing layer.

### Architecture

**Components:**

1. **Flag declaration** ([cmd/ailang/main_run.go:43](cmd/ailang/main_run.go#L43)) — add `argsFileFlag := fs.String("args-file", "", "Path to a file containing JSON arguments (alternative to -args-json; bypasses shell quoting)")`.
2. **Resolver function** (new helper in `cmd/ailang/main_run.go` or `cmd/ailang/run_helpers.go`) — `resolveArgsJSON(argsJSON, argsFile string, stdin io.Reader) (string, error)` that returns the effective JSON string and enforces mutual exclusivity.
3. **Wiring** ([cmd/ailang/main_run.go:204](cmd/ailang/main_run.go#L204)) — call resolver after `fs.Parse`, before `runFile`. Pass the resolved string in place of `*argsJSONFlag`.

The resolver is the only new logic. The downstream call chain (`runFile` → `executeModuleEntrypoint` → `decodeEntrypointArgs`) sees a normal JSON string and is unchanged.

**Resolution rules (in order):**

| `-args-json` | `-args-file` | Result |
|---|---|---|
| default (`"null"`) | empty | `"null"` (current behavior) |
| `-` | empty | read all of stdin, trim trailing whitespace |
| any other | empty | use literal value (current behavior) |
| default (`"null"`) | `<path>` | read file, trim trailing whitespace |
| any non-default | `<path>` | error: "specify exactly one of -args-json or -args-file" |

### Implementation Plan

**Phase 1: CLI plumbing** (~45 min)
- [ ] Add `argsFileFlag` to `runCommand()` flag set.
- [ ] Implement `resolveArgsJSON(argsJSON, argsFile string, stdin io.Reader) (string, error)`.
- [ ] Call resolver after `fs.Parse`; on error, print to stderr and `os.Exit(2)`.
- [ ] Update help text in `cmd/ailang/help.go` (the `--args-json` line and the run-command usage).

**Phase 2: Tests** (~45 min)
- [ ] Unit tests for `resolveArgsJSON` — table-driven, covering all 5 rows above plus: file-not-found, file-empty, file-with-trailing-newline, stdin-empty.
- [ ] Integration test: invoke compiled `ailang` binary with `-args-file` against an existing example, assert exit code 0 and expected output.
- [ ] Integration test: same example with `-args-json -` and JSON piped on stdin.

**Phase 3: Windows runtime CI matrix** (~1.5 hours — co-equal goal, not a tacked-on smoke)

The current `ci.yml` is single-OS. We extend its primary `test` job into an OS matrix so the full Go test suite runs on Windows on every PR.

- [ ] Edit `.github/workflows/ci.yml`: convert the `test` job to a matrix with `os: [ubuntu-latest, windows-latest]` (and optionally `macos-latest` if marginal cost is acceptable).
- [ ] Make Windows-specific path/shell tweaks: use `shell: bash` for cross-platform steps where reasonable, or guard with `if: runner.os != 'Windows'` for steps that genuinely cannot run on Windows (e.g. `make` targets that shell out to POSIX-only tools). Document each guard with a one-line comment naming what blocks it.
- [ ] Add a PowerShell-invoked end-to-end smoke step to the Windows job: build the binary, write `args.json`, run `ailang run -args-file args.json` and `Get-Content args.json | ailang run -args-json -` against an existing example, assert exit 0.
- [ ] Replay one historical Windows-only failure (commit `bf113aba` — opencode test failure, or `4f4fa419` — path-mismatch) locally against the new matrix to confirm it would have caught it. Document the result in the PR description.
- [ ] Confirm green on `dev`. Open a separate one-line follow-up PR (or coordinate with the maintainer) to mark the `test (windows-latest)` check **required** in branch protection — this is the durable change that makes Windows breakage block merges.

### Files to Modify/Create

**New files:**
- `cmd/ailang/run_args_resolver.go` — `resolveArgsJSON` helper, ~40 LOC.
- `cmd/ailang/run_args_resolver_test.go` — table-driven tests, ~120 LOC.

**Modified files:**
- `cmd/ailang/main_run.go` — add `argsFileFlag`, call resolver after `fs.Parse`, ~10 LOC.
- `cmd/ailang/help.go` — extend `--args-json` line and add `--args-file` line, ~5 LOC.
- `cmd/ailang/prompts/devtools/v0.8.0.md` and `v0.8.0-compact.md` — mention `--args-file`, ~2 LOC each.
- `.github/workflows/ci.yml` — convert `test` job to OS matrix including `windows-latest`; add PowerShell smoke step exercising `-args-file` and `-args-json -`, ~30-60 LOC depending on number of guards needed.
- `CHANGELOG.md` (under v0.14.2) — note the new flag, stdin support, **and** the new Windows CI coverage, ~6 LOC.
- `docs/docs/guides/development-workflow.md` — one paragraph: "CI now runs on Windows; if your change touches paths, shell-outs, or CLI flag parsing, expect to see Windows-specific failures and treat them as blocking."

## Examples

### Example 1: PowerShell user (the failing case from the bug report)

**Before:**
```powershell
PS> ailang run -args-json '["robot"]' --entry main story_studio.ail
Error: failed to decode arguments: invalid JSON: invalid character 'A' looking for beginning of value
```

**After (file form):**
```powershell
PS> Set-Content args.json '["robot"]'
PS> ailang run -args-file args.json --entry main story_studio.ail
# runs cleanly
```

**After (stdin form):**
```powershell
PS> '["robot"]' | ailang run -args-json - --entry main story_studio.ail
# runs cleanly
```

### Example 2: bash/zsh user (unchanged behavior)

```bash
$ ailang run -args-json '["robot"]' --entry main story_studio.ail   # still works
$ ailang run -args-file args.json --entry main story_studio.ail     # also works
$ jq '.robot' input.json | ailang run -args-json - --entry main story_studio.ail
```

### Example 3: Mutual exclusivity error

```bash
$ ailang run -args-json '[1]' -args-file args.json --entry main file.ail
Error: specify exactly one of -args-json or -args-file
exit status 2
```

## Success Criteria

**CLI fix:**
- [ ] `-args-file <path>` reads JSON from the named file and runs the entrypoint with the decoded arguments.
- [ ] `-args-json -` reads JSON from stdin and runs the entrypoint with the decoded arguments.
- [ ] Specifying both flags exits with code 2 and a clear error message.
- [ ] An empty file or empty stdin under either form errors with a clear message (not a generic JSON decode error).
- [ ] Existing `-args-json '<inline>'` invocations continue to work identically (no behavior change).

**Windows CI:**
- [ ] `ci.yml`'s `test` job runs the full Go test suite on `windows-latest` on every PR and on pushes to `dev`/`main`.
- [ ] The Windows job includes a PowerShell-invoked smoke that exercises `-args-file` and `-args-json -` against a real example.
- [ ] At least one historical Windows-only failure is replayed against the new matrix and confirmed to fail (then the test in question is fixed or guarded), proving the CI would have caught it.
- [ ] First green Windows run is achieved on `dev`; follow-up to mark the check **required** is filed (or done) and noted in the PR description.

**Process / housekeeping:**
- [ ] All tests passing (`make test`, `make ci`).
- [ ] CHANGELOG.md updated under v0.14.2.
- [ ] Reply sent to message `msg_20260426_221627_0d287ba3` confirming the fix and the released version.

## Testing Strategy

**Unit tests** (`cmd/ailang/run_args_resolver_test.go`):
- All 5 rows of the resolution table.
- File not found: returns wrapped error.
- File empty (zero bytes or only whitespace): returns "empty -args-file" error.
- File with trailing newline: returned string has the newline trimmed.
- Stdin empty: returns "empty stdin for -args-json -" error.
- Stdin with trailing newline: trimmed.

**Integration tests** (against compiled binary):
- One existing example with a single-arg entrypoint, exercised three ways: inline, `-args-file`, stdin. All three produce identical stdout.
- Mutual exclusivity check: assert exit 2 and stderr substring.

**Manual testing:**
- On macOS/Linux: run all three forms against `examples/` to confirm no regression.
- On Windows (or via Windows CI job): run `-args-file` and stdin forms in PowerShell and `cmd.exe`.

## Deferred Decisions

- File-size cap on `-args-file` — agent may pick a sensible limit (suggest 16 MiB) or leave unbounded with a comment.
- Whether to log "(read N bytes from <path>)" at info level when `-args-file` is used — agent may choose; default to silent.
- Whether `jsonl`-style multi-document files are supported under `-args-file` — **no, single JSON document only.** Multi-document is a future feature if asked for.

## Non-Goals

- **A general `<flag>-file` mechanism for every string flag.** This audit (see Problem Statement) found `-args-json` is the only inline-JSON flag with this trap. We do not preemptively add `-budget-report-file`, etc.
- **Auto-detecting that we're on PowerShell and shimming.** The fix is to give the operator a non-quoting path, not to play whack-a-mole with shell quirks.
- **Replacing or deprecating `-args-json`.** Inline JSON remains the default and recommended form on POSIX shells.
- **Rewriting AILANG's batch-mode positional-args handling** (which is what produces the second failure mode). The new flag sidesteps it entirely.

## Timeline

**Single sprint, ~3.5 hours total:**
- Phase 1 (CLI plumbing): 45 min
- Phase 2 (tests): 45 min
- Phase 3 (Windows runtime CI matrix): 1.5 hours
- Buffer for Windows-specific CI debugging (path separators, line endings, missing POSIX tools): 30 min

Targets v0.14.2 patch release.

The 1.5h estimate for Phase 3 assumes one or two `make` targets need a `runner.os` guard. If `make test` runs cleanly on Windows out of the box, Phase 3 collapses to ~45 min.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Stdin reader interferes with REPL or interactive flows | Low | Stdin is only read when `-args-json -` is set explicitly; never speculatively. |
| Hidden BOM in `-args-file` content (Windows `Set-Content` can add UTF-8 BOM) | Med | Strip a leading UTF-8 BOM (`﻿`) in the resolver before returning the string; add a test for it. |
| Operator passes `-args-file` to a non-existent path and gets a confusing error | Low | Resolver wraps the error: `failed to read -args-file <path>: <os.PathError>`. |
| Mutual-exclusivity rule frustrates users who expected one to override the other | Low | Error message names both flags so the fix is obvious; documented in `CHANGELOG.md` and help text. |
| Adding Windows CI surfaces a long tail of pre-existing Windows-only test failures | Med | Plan budgets 30 min for triage; the explicit policy is to **fix or skip-with-reason** each one in this PR rather than disable the matrix. A skipped test must include a comment `// TODO(windows-ci): <reason>` and a tracking issue. |
| Windows CI flakiness (slower runners, different line-ending handling) creates noisy failures | Med | Set `core.autocrlf=false` in checkout step; pin Go version; if a test is flaky-on-Windows-only, fix the test (don't add a retry). Quarantine criterion: 3+ flakes in a week → fix or skip with issue. |
| Windows CI adds 3-5 min to PR cycle time | Low | Run Windows in parallel with Linux in the matrix (no serial penalty). Accept the wall-clock cost — the alternative is shipping more bugs to Windows users. |
| Branch-protection change (marking Windows required) is forgotten after first green run, leaving the check advisory | Med | Success criteria explicitly require the follow-up; checked off in the PR description, not just in this design doc. |

## Related Documents

<!-- Auto-search returned only weak matches (top neural score 0.35). This is a fresh CLI-ergonomics area, not a continuation of prior work. -->

**Implemented (weakly related, not load-bearing):**
- [design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md](design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md) — prior CLI-ergonomics precedent

**Direct precedent in current code (not a design doc):**
- `-allow-env-file` flag at [cmd/ailang/main_run.go:57](cmd/ailang/main_run.go#L57) — same shape as the proposed `-args-file`.

## References

- Bug report: agent inbox message `msg_20260426_221627_0d287ba3` (docparse maintainer, 2026-04-26)
- Affected flag: [cmd/ailang/main_run.go:43](cmd/ailang/main_run.go#L43)
- Decoder (unchanged): [cmd/ailang/run_helpers.go:505](cmd/ailang/run_helpers.go#L505)
- [Design Axioms](/docs/references/axioms)

## Future Work

- Audit other CLI flags for shell-quoting hazards (none found in this audit, but worth re-checking when new flags are added).
- Add a `make ci-windows` target that runs the Windows-specific smoke locally via Docker if maintainers want pre-push verification.
- Once Windows is stable in `ci.yml`, consider adding `macos-latest` to the matrix as well — same systemic argument applies, though we have less direct evidence of macOS-only regressions today.
- Once Windows CI has a few weeks of green history, expand it from "test job" to also run `make verify-examples` and `make test-imports` on Windows.

---

**Document created**: 2026-04-26
**Last updated**: 2026-04-26
