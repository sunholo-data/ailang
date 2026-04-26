# Sprint Plan: M-CLI-ARGS-WIN (v0.14.2)

## Summary

Ship the P0 Windows hotfix from [m-cli-args-file-stdin.md](m-cli-args-file-stdin.md): add `-args-file <path>` and `-args-json -` (stdin) so Windows/PowerShell users can pass JSON entrypoint args without fighting shell quoting, **and** add a `windows-latest` job to `ci.yml` so the next Windows-only regression is caught at PR time.

**Duration:** 1 day (~3.5 hours of focused work)
**Dependencies:** None
**Risk Level:** Low (CLI plumbing) / Medium (Windows CI may surface latent test failures)
**Target version:** v0.14.2 patch
**Related design doc:** [design_docs/planned/v0_14_2/m-cli-args-file-stdin.md](m-cli-args-file-stdin.md)
**Reporter:** docparse maintainer, message `msg_20260426_221627_0d287ba3`

## Current Status Analysis

### Completed Recently (last 5 commits on `dev`)
- ✅ `bf113aba` — fix CI pipefail and opencode Windows test failures
- ✅ `b30c8057` — guard agent-only models, expand test suites
- ✅ `90159d3a` — gitignore eval_results, commit v0.14.1 perf tables
- ✅ `3442be2c` — chore: untrack eval_results raw run files

**Pattern:** recent work is small, focused, mostly chores + small fixes. No multi-day features in flight that would block this hotfix.

### Velocity
- Sprint is small enough (single-file CLI fix + CI matrix tweak) that velocity calculation is unnecessary.
- Realistic budget: 3.5 hours within one working session.

### Remaining from Design Doc
- ⏳ Phase 1: CLI plumbing (~80 LOC: 40 helper + 10 main_run + 30 wiring/help)
- ⏳ Phase 2: Tests (~150 LOC: unit + integration)
- ⏳ Phase 3: Windows CI matrix + smoke (~30-60 LOC YAML, plus test triage if anything is broken on Windows)

## Proposed Milestones

### Milestone 1: CLI Plumbing — `-args-file` + stdin (M1)

**Goal:** Make `-args-file <path>` and `-args-json -` resolve to a JSON string before the existing decoder runs, with mutual-exclusivity errors. No downstream code changes.

**Estimated:** ~50 LOC implementation + ~120 LOC tests = ~170 LOC
**Duration:** ~90 min

**Tasks:**
1. Create `cmd/ailang/run_args_resolver.go` with `resolveArgsJSON(argsJSON, argsFile string, stdin io.Reader) (string, error)`. Handles: default `null`, inline JSON (passthrough), `-` stdin sentinel, file read, mutual exclusivity, empty-input error, UTF-8 BOM strip, trailing-whitespace trim.
2. In [cmd/ailang/main_run.go:43](cmd/ailang/main_run.go#L43), add `argsFileFlag := fs.String("args-file", "", "Path to a file containing JSON arguments (alternative to -args-json; bypasses shell quoting)")`.
3. After `fs.Parse` at [cmd/ailang/main_run.go:124](cmd/ailang/main_run.go#L124), call resolver with `os.Stdin` as the third arg. On error, print to stderr and `os.Exit(2)`. Use the resolved string in place of `*argsJSONFlag` in the `runFile` call at line 204.
4. Update help text at [cmd/ailang/help.go:211](cmd/ailang/help.go#L211): note that `--args-json -` reads from stdin, add `--args-file <path>` line.
5. Update prompt files: `cmd/ailang/prompts/devtools/v0.8.0.md`, `v0.8.0-compact.md`. One-line additions only.

**Acceptance Criteria:**
- [ ] `make build && make install` succeed
- [ ] `ailang run -args-file <path> --entry main <file>` runs an existing example with single-arg entrypoint
- [ ] `echo '<json>' | ailang run -args-json - --entry main <file>` works on macOS shell
- [ ] `ailang run -args-json '[1]' -args-file foo.json file.ail` exits 2 with message naming both flags
- [ ] `ailang run -args-file /no/such/path file.ail` exits 2 with wrapped path error
- [ ] Existing inline `-args-json '[...]'` invocations unchanged
- [ ] `make lint` clean

**Risks:**
- Stdin read could hang if user accidentally passes `-` without piping. Mitigation: doc the form clearly in help text; the stdin read returns immediately at EOF, so a closed stdin produces an empty-input error (exit 2), not a hang.
- BOM handling differs across Windows tools. Mitigation: explicit BOM-strip + dedicated test case.

### Milestone 2: Windows CI Matrix (M2)

**Goal:** Convert `ci.yml`'s `test` job to an OS matrix that includes `windows-latest`, with a PowerShell smoke step exercising both new input forms. Establish lasting Windows regression signal.

**Estimated:** ~40 LOC YAML + ~50 LOC test fixes/guards (variable depending on what's broken)
**Duration:** ~90 min (with 30 min buffer for triage of any pre-existing Windows test failures)

**Tasks:**
1. Read current `.github/workflows/ci.yml` `test` job; convert `runs-on: ubuntu-latest` to `strategy.matrix.os: [ubuntu-latest, windows-latest]` with `runs-on: ${{ matrix.os }}`.
2. Add `actions/checkout@v4` `with: { fetch-depth: 1 }` and explicit `core.autocrlf=false` to avoid line-ending churn in test fixtures.
3. Identify steps that won't run on Windows (e.g., `make` targets that shell out to POSIX tools). Either: (a) replace with cross-platform step (`go test ./...` directly), or (b) gate with `if: runner.os != 'Windows'` plus a one-line comment naming what blocks it.
4. Add a Windows-only PowerShell smoke step that builds `ailang.exe`, writes `args.json`, and runs both `ailang run -args-file args.json --entry main examples/<picked-example>.ail` and `Get-Content args.json | ailang run -args-json - --entry main examples/<picked-example>.ail`. Assert exit 0.
5. Triage any pre-existing Windows-only test failures: **fix or skip-with-reason**. Each skip needs a `// TODO(windows-ci): <reason>` comment and a tracked GH issue (open them inline).
6. Push branch, observe Windows job status. Iterate until green.
7. Add a one-paragraph note to `docs/docs/guides/development-workflow.md`: "CI runs on Windows; if your change touches paths, shell-outs, or CLI flag parsing, treat Windows failures as blocking."
8. Update `CHANGELOG.md` under v0.14.2 with the new flag, stdin support, **and** the Windows CI coverage.

**Acceptance Criteria:**
- [ ] `ci.yml` `test` job runs on `[ubuntu-latest, windows-latest]` matrix
- [ ] Windows job runs `go test ./...` (full suite, not a subset)
- [ ] Windows-only PowerShell smoke step is present and green
- [ ] If any tests are skipped on Windows, each has `// TODO(windows-ci):` comment + open issue
- [ ] CHANGELOG.md mentions all three changes
- [ ] PR description records: (a) any tests skipped on Windows + their issues, (b) which historical Windows-only failure (`bf113aba` opencode or `4f4fa419` path-mismatch) was replayed locally to validate the new CI would have caught it, (c) reminder that branch protection needs the new check marked **required** post-merge

**Risks:**
- **Latent Windows-only test failures**. Possible. Mitigation: budget allows triage; policy is fix or skip+issue, never disable the matrix.
- **Windows runner slowness/flakes** could exceed budget. Mitigation: if blocked >30 min, skip flaky tests with issue and ship; circle back in a follow-up. Don't let perfect Windows coverage block a P0 user-facing fix.
- **Branch protection toggle** (marking the check required) is a maintainer-only action, not done in this PR. Captured as a follow-up note in the PR description; success criteria explicitly include this reminder.

## Success Metrics

- ✅ Both new input forms work cross-shell (verified on macOS in dev, on Windows in CI).
- ✅ Existing `-args-json '<inline>'` invocations unchanged (zero regression in `make test`).
- ✅ `ci.yml` `test (windows-latest)` job is green.
- ✅ One historical Windows-only failure replayed and confirmed the new CI would catch it.
- ✅ CHANGELOG.md updated under v0.14.2.
- ✅ Reply sent to message `msg_20260426_221627_0d287ba3` confirming fix + version.
- ✅ Follow-up reminder filed/noted: mark `test (windows-latest)` as required in branch protection.

## Dependencies

None. The Go test suite already builds on Windows (per `build.yml`); the only unknown is whether all tests *pass* on Windows.

## Open Questions

1. **Which existing example to use for the smoke step?** Anything in `examples/` with a single-arg `main` entrypoint accepting JSON. M1 will pick one; M2 reuses it.
2. **Are we OK adding ~3-5 min to PR cycle time for Windows CI?** Design doc assumes yes; sprint-executor proceeds on that assumption.
3. **Branch protection update** — sprint-executor will leave a note in the PR; the actual toggle requires maintainer action and is out of scope for this sprint.

## Notes

- This is a P0 hotfix sprint, not a milestone. Single-day, two milestones, fits in one session.
- The CI matrix (M2) is the more important systemic change; M1 alone would unblock today's reporter but leave us blind to the next Windows bug.
- Sprint-executor should commit M1 and M2 separately so the CI bisect signal stays clean if Windows reveals a problem in M1.
