# Sprint Evaluation — M-COORDINATOR-WINDOWS-PACKAGE-TIMEOUT-HEADROOM (mission v1, iteration 348)

**Evaluator**: independent sprint-evaluator (Sonnet 5), no relation to the executor (pi/deepseek-v4-flash).
**Sprint HEAD**: `73d4cee75` on base `39dfb6a84`. Worktree: `/Users/voightkampff/.ailang-driver-pin/.eval-wt-v1-iter348` (detached, unmodified after evaluation — all mutation testing was applied and reverted via `cp` backups, never `git checkout`).

## VERDICT: **FAIL — 35 / 100** (bar is 70)

**Hard fail triggered**: Acceptance Criteria category (<50% of criteria genuinely met when weighted by
what they claim to verify). The headline acceptance criterion for M3 — verified against the sprint's
own real PR CI run, exactly as the design doc itself designates as authoritative — is **provably false**.

---

## Executive summary

The sprint's central deliverable, `tools/ci/headroom`, **cannot parse real `go test` output**. Its
`okRe` regex requires the literal text `ok\t` (a tab immediately after "ok"), but Go pads `ok` with two
spaces before the tab to align columns with `FAIL` (`ok  \t<pkg>\t<N>s`). Every unit-test fixture in
`main_test.go` was hand-typed with a single tab, so the whole suite is green — but it never once
exercised genuine `go test` output. Against real output, on this rig, on GitHub Actions ubuntu-latest,
and on GitHub Actions windows-latest, the parser matches **zero** `ok` lines, triggering the tool's own
anti-vacuity guard (`::error:: headroom: parsed 0 packages from non-empty log — parser may be stale`),
which returns exit 1 — and since `go_rc` is 0 (the suite passed), the wiring's own exit-code logic
correctly propagates that 1 to the step, turning **every green CI run permanently red**.

This is not a theoretical finding: **the sprint's own PR (#1102, github.com/sunholo-data/ailang) is
red on both `test` and `test-windows` right now**, for exactly this reason, confirmed by pulling the
job logs directly. The M3 acceptance criterion that names this exact PR-CI check as the authoritative
verification for the (locally-unexecutable) pwsh leg is therefore falsified by the sprint's own
designated instrument.

The parts of the design that were the subject of two design-quorum rounds — the bash `set -e` /
`|| rc=$?` guards and the pwsh `$ErrorActionPreference`/`$PSNativeCommandUseErrorActionPreference`
prepends — are genuinely sound and independently verified (both locally via a harness that runs the
literal extracted step body, and on the real Windows runner). The defect is narrow but total: a single
wrong assumption about column padding in a regex, never checked against a real `go test` invocation
before being wired into both CI legs.

---

## 1. Does it do what the design says? (commands + real output)

| # | Command | rc | Observed |
|---|---|---|---|
| 1a | `grep -n 'timeout 416s' .github/workflows/ci.yml` | n/a | 4 hits (lines 89, 114, 479, 499) — design/plan/JSON all claim "exactly two"; see §5 |
| 1b | `grep -n '172.5\|1.80\|1.34' .github/workflows/ci.yml` | n/a | present in both comment blocks — MATCHES |
| 1c | `go test -count=1 ./internal/cihygiene/` | 0 | `ok ... internal/cihygiene`, `TestGoTestTimeoutIsDerived` passing — MATCHES |
| 1d | `go test -count=1 ./tools/ci/headroom/` | 0 | all 6 named tests pass — MATCHES on synthetic fixtures, **but see §2/§3: vacuous against real input** |
| 1e | `grep -n 'go_test_headroom.log\|RUNNER_TEMP/headroom\|tools/ci/headroom' .github/workflows/ci.yml \| wc -l` | n/a | 9 (≥8 required) — MATCHES |
| 1f | `grep -n 'go_test_headroom.log\|gated_integration.log' .github/workflows/ci.yml` | n/a | disjoint line sets — MATCHES |
| 1g | **The M3 PR-CI acceptance criterion itself** — "the test-windows job on this sprint's PR must show ... no `::error:: headroom` line ... job verdict == the suite's verdict" | — | **VIOLATED.** PR #1102, run `34152632548`: job `test` → FAILURE, job `test-windows` → FAILURE. Both fail at exactly the headroom-wired step (`Run tests with timeout (M-DX11)` / `Run Go test suite`) with `##[error] headroom: parsed 0 packages from non-empty log — parser may be stale` / `##[error]Process completed with exit code 1.`, while every `go test` line visible in both logs is a clean passing `ok` line. `gh run view 34152632548 --repo sunholo-data/ailang --json jobs --jq '.jobs[] | select(.name=="test") | {conclusion, steps: [.steps[] | select(.conclusion=="failure") | .name]}'` → `{"conclusion":"failure","steps":["Run tests with timeout (M-DX11)"]}` — isolates the failure to exactly this sprint's step, nothing else. |

**Verdict on item 1**: M1 fully delivers. M2's unit tests pass but do not test the real contract (see
§2/§3). M3's wiring mechanics are sound (see §3) but the end-to-end acceptance criterion — the one
the design explicitly elevates to "authoritative, because pwsh can't be run locally" — **fails on the
sprint's own PR**.

---

## 2. Mutation non-vacuity, per milestone (own diff, own tests)

### M1 — `-timeout 416s` + `TestGoTestTimeoutIsDerived`

| Mutation | Command | Result |
|---|---|---|
| Revert Linux `-timeout 416s`→`300s` | `sed -i '' '114s/-timeout 416s/-timeout 300s/' .github/workflows/ci.yml && go test -count=1 -run TestGoTestTimeoutIsDerived ./internal/cihygiene/` | **RED** — `reverted to the unexplained "go test -timeout 300s ./..."` — killed correctly |
| Delete derivation token `172.5` | `sed -i '' '/172.5/d' .github/workflows/ci.yml && go test ...` | **RED** — `missing factor "172.5"` — killed correctly |
| **Own-diff revert** (restore ci.yml to `39dfb6a84`, i.e. pre-M1 state) | `git show 39dfb6a84:.github/workflows/ci.yml > /tmp/x && cp /tmp/x .github/workflows/ci.yml && go test ...` | **RED** (missing all 3 tokens) — confirms the test is not defended by some earlier milestone; it is the sole guard for this property |
| Restore (all arms) | `cp` backup restore | **GREEN** in every case |

M1's gate is non-vacuous and correctly implemented. No findings here.

### M2 — `tools/ci/headroom` unit tests

| Mutation | Test | Result |
|---|---|---|
| Remove `failRe` branch | `TestParseGoTestOutput` | **RED** — "got 4 records, want 5" — killed correctly |
| Neutralize anti-vacuity guard (`if false && len(data)>0 && n==0`) | `TestEmptyParseOnNonEmptyInput` | **RED** — falls through to the floor message instead — killed correctly (the floor is a real backstop for this specific guard) |
| Neutralize package-count floor (`if false && n<minPackages`) | `TestPackageCountFloor` | **RED** — exit 0 instead of 1 — killed correctly |
| **[unnamed by the plan] Remove `budget<=0` validation** (`err!=nil\|\|budget<=0` → `err!=nil`) | full suite (`go test -count=1 ./tools/ci/headroom/`) | **SURVIVES** — all 6 tests still green. Runtime probe with the mutation applied: `headroom fixture.log 0` prints `9223372036854775807% of budget` (int64-overflowed `math.Round(+Inf)`) instead of a usage error. The **shipped code is correct** (verified: `headroom fixture.log 0` and `headroom fixture.log -100` both correctly exit 2), but no test asserts it — a regression here would ship silently. |
| **[unnamed by the plan] Remove seconds tie-break** (`return records[i].pkg < records[j].pkg` branch of the `sort.Slice` comparator) | full suite | **SURVIVES** — all 6 tests green (`TestTopN` only uses distinct durations, never exercises the tie case). The design's own stated goal ("ties broken by package name ascending — deterministic CI logs") is unverified. |
| **[unnamed by the plan] Relax the usage arg-count guard** (`len(args)<3` → `len(args)<2`) | full suite | **SURVIVES** — all 6 tests green. Runtime probe: invoking `headroom fixture.log` (log path only, no budget) with the mutation applied **panics** (`index out of range [2] with length 2`) instead of printing the intended usage message — an ungraceful crash, not caught by any test. |

Three mutations not named in the design/plan were found and all three **survive** the full M2 test
suite — a genuine gap, though two of the three affect only unexercised inputs of currently-correct
code, and the third (arg-count) exposes a latent crash path.

### M3 — CI wiring (no automated Go test exists for this milestone; verified via a harness that
extracts and runs the *literal* step body from `ci.yml`, never retyped)

Extraction: `sed -n '108,118p' .github/workflows/ci.yml` (the Linux `run:` block, byte-identical to
what ships). Harness ran under `bash -e`, with a stubbed `go` on `PATH`, `RUNNER_TEMP` pointed at a
scratch dir with a freshly built `headroom` binary.

| Mutation | Fixture | Mutated result | Control (unmutated) result |
|---|---|---|---|
| Drop `if [ $go_rc -ne 0 ]; then exit $go_rc; fi` | 130 clean single-tab `ok` records (parses fine) + `go` stub `exit 1` | step **exits 0** — a failed suite goes green | step **exits 1** — correctly fails |
| Restore `set -e`-unsafe form (`go test ...; go_rc=$?`, drop `\|\|`) | `go` stub prints a FAIL+panic fixture, `exit 1`, under `bash -e` | step aborts before `cat`; **panic count in captured stdout = 0** (silent red, no log) | panic **visible** in stdout, step exits 1 |
| Remove the build step (`rm -f "$RUNNER_TEMP/headroom"`) | 130-record clean pass fixture, `go` stub `exit 0` | step **exits 1** — job reds despite a fully passing suite (design's stated "loud, not silent" intent — confirmed) | step exits 0 |
| *(not a hypothetical mutation — this is the shipped code, unmodified)* | **REAL** `go test ./...` output: this rig (darwin, go1.26.6), AND the sprint's own live PR on GitHub Actions ubuntu-latest and windows-latest | step **exits 1** even though the underlying suite fully passed | — (there is no "control" for this row; this IS the control, and it fails) |

The three named M3 mutations are correctly killed by the wiring mechanics in isolation. But the row
that matters — running the real, unmutated, shipped code against real `go test` output — is the one
that fails, and it fails identically whether run locally or on GitHub's own runners.

---

## 3. The exit-code / output-preservation contract

Verified via the extracted-step-body harness (§2, M3 table) plus first-party GitHub Actions logs:

- **Passing suite, real fixture**: harness case using this repo's own genuine `go test -count=1 ./...`
  output (128 `ok` lines, captured on this rig, rc=0) run through the *unmutated* Linux step body:
  **step exits 1** (not 0) — confirmed live on GitHub Actions PR #1102 job `test`, run `34152632548`,
  identical symptom (`##[error]Process completed with exit code 1` immediately after `##[error]
  headroom: parsed 0 packages...`, with no `FAIL` line anywhere above it in the log).
- **Failing suite with a panic** (synthetic panic + `FAIL` line via a stubbed `go`, real go1.26.6-format
  `FAIL\tfailpkg\t0.283s` line, `go` stub exits 1): step exits 1, panic text **is** printed to stdout,
  matching the "must not swallow go test output" contract. This case happens to *mask* the parser bug,
  because `go_rc=1` already forces the exit before `hr_rc` is consulted.
- **`set -e` guard proof**: reproduced independently (§2 table) — removing `|| go_rc=$?` causes the
  script to abort *before* `cat`, verified via `grep -c panic` = 0 on the mutated run vs. panic visible
  on the control.
- **pwsh leg — executed, not just inspected**: pwsh is confirmed absent on this rig
  (`command -v pwsh` → exit 1, control `command -v bash` → `/bin/bash`), so I could not run it here.
  However the design itself designates the sprint's own PR `test-windows` job as the authoritative
  check for exactly this reason, and I pulled that job's real log (`gh run view --job
  101837895074 --repo sunholo-data/ailang --log`): the `$ErrorActionPreference = 'Continue'` /
  `$PSNativeCommandUseErrorActionPreference = $false` prepend **does work as designed** — the log was
  printed via `Get-Content`, `headroom.exe` ran, `$LASTEXITCODE` was captured correctly at each step —
  but it hit the identical parser defect: `##[error] headroom: parsed 0 packages from non-empty log —
  parser may be stale` followed by `##[error]Process completed with exit code 1.`, on a fully-passing
  suite. **The pwsh wiring mechanics are sound; the parser they wrap is not.**

---

## 4. The instrument's own honesty (anti-vacuity, edge cases)

Ran the actual built `tools/ci/headroom` binary against every case requested:

| Input | Result | Assessment |
|---|---|---|
| Real `go test` output, passing (`ok  \t<pkg>\t<N>s`, this repo, this rig) | `::error:: headroom: parsed 0 packages from non-empty log — parser may be stale`, rc=1 | **BLOCKING** — see §0 |
| Real `(cached)` line (no duration at all — Go's actual cached format) | silently dropped, not counted, not reported (with 59 other well-formed lines, floor still satisfied, exit 0) | the `(cached)` regex branch is **dead code** against real Go output; Go never emits a duration next to `(cached)` |
| CRLF line endings | `::error:: headroom: parsed 0 packages ...`, rc=1 | fails safe (loud, not silent), but untested/undocumented; plausible extra failure mode if the pwsh `*>` redirect ever normalizes line endings (unverified — no pwsh) |
| `?   pkg [no test files]` | silently ignored (not a record) | correct — not a timing record |
| Panic line in the middle of otherwise-valid records | records before/after both counted correctly, exit 0 | correct |
| Budget `0` | `usage: headroom <logfile\|-> <budget-seconds>`, rc=2 | correct, and confirmed NOT survived by removing the guard (see §2 unnamed mutation) — code is right, just untested |
| Negative budget (`-416`, `-- -416`) | rc=2 both forms | correct |
| Missing file | `::error:: headroom: cannot read log: ... no such file or directory`, rc=1 | non-silent, but shares exit code 1 with the anti-vacuity/floor cases — the contract's "two distinct non-zero paths" is really three conditions sharing one code; not a functional bug, just an imprecise contract description |
| Empty file | `::error:: headroom: parsed 0 packages (< 50) — parser may be stale`, rc=1 | matches the contract note exactly (fails via the floor, not the zero-on-non-empty branch) |
| Truncated log (mid-line cutoff) | `::error:: headroom: parsed 1 packages (< 50) ...`, rc=1 | fails safe |

The anti-vacuity/floor machinery is **well-built and correctly non-vacuous where it applies** — every
one of its own named mutations was correctly killed (§2). The tragedy is that it is doing exactly its
job: catching a parser that cannot read its primary input. A well-built smoke detector is not a
substitute for a working parser.

---

## 5. Is anything claimed that is not true?

- **The M3 PR-CI acceptance criterion** ("no `::error:: headroom` line; job verdict == the suite's
  verdict") is **false**, checked against the sprint's own PR #1102 — see §1 row 1g and §3.
- **"exactly two hits" for `grep -n 'timeout 416s' .github/workflows/ci.yml`** appears three times
  (design doc M1 acceptance, sprint plan M1 acceptance, sprint JSON `final_gates`) and is wrong in all
  three: the real count is **4**, because the derivation comment prose itself contains the literal
  substring `-timeout 416s` (ci.yml:89, ci.yml:479) in addition to the two `go test` command lines
  (ci.yml:114, ci.yml:499). Verified this was true even at the M1-only commit (`git show
  a3cbbca0c:.github/workflows/ci.yml | grep -c 'timeout 416s'` → 4). This is a **documentation
  accuracy defect, not a functional one** — `TestGoTestTimeoutIsDerived` itself is unaffected, because
  it parses YAML `Run` payloads (which exclude comments), not raw grep counts.
- **Negative control** (independently confirmed, not taken on the controller's word): `go build ./...`
  fails identically on `cmd/wasm` (`function main is undeclared in the main package`) both at this
  sprint's HEAD and at a freshly-fetched, pristine `origin/dev` (`1b8e7eab1`), added as a temporary
  `git worktree` and removed afterward. Not attributable to this sprint — confirmed, not assumed.
- **Sprint JSON is stuck at `"status": "planned"`, all three milestones `"passes": false`.** Nothing
  in the artifact record claims this sprint passed its own gates before being handed to evaluation,
  which is at least consistent with — though not proof of — the defect never having been caught.
- The design doc's own historical provenance data (the five-commit CI timing table, V1–V22 entries
  describing the *original* problem) predates this sprint (produced by the controller across three
  quorum rounds) and was **not** re-derived by me; I treated it as out of scope for this evaluation,
  which is about the sprint's *code*, not the antecedent design's measurements.

---

## Findings, classified

### BLOCKING

1. **`okRe` does not match real `go test` output.** Root cause: the regex requires `^ok\t` (tab
   immediately after "ok"), but Go pads "ok" with two spaces to align with "FAIL"'s width before the
   tab (`ok  \t<pkg>\t<N>s`). Confirmed on this rig (go1.26.6/darwin) and on the sprint's own live PR
   CI (both `test` and `test-windows` jobs, GitHub Actions ubuntu-latest/windows-latest).
   Repro: `go test -timeout 60s ./internal/cihygiene/... ./tools/ci/... > /tmp/realgo.log 2>&1; export
   RUNNER_TEMP=/tmp; go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom; "$RUNNER_TEMP/headroom"
   /tmp/realgo.log 416` → `::error:: headroom: parsed 0 packages from non-empty log — parser may be
   stale`, rc=1.
2. **Consequence of #1: the sprint's own PR is red on both CI legs, right now**, exclusively at the
   step this sprint wired. Repro: `gh run view 34152632548 --repo sunholo-data/ailang --json jobs
   --jq '.jobs[] | select(.name=="test") | {conclusion, steps:[.steps[]|select(.conclusion=="failure")|.name]}'`
   → `{"conclusion":"failure","steps":["Run tests with timeout (M-DX11)"]}`; same for
   `test-windows`/`Run Go test suite` (job `101837895074`).
3. **The `(cached)` suffix handling is unreachable against real Go output** — a cached line never
   carries a duration, which the regex requires. Given `setup-go`'s `cache: true` on both jobs, cache
   hits are a realistic occurrence in this repo's own CI, compounding #1 rather than being a separate
   hypothetical.

### NON-BLOCKING

4. Three mutations not named by the plan survive the full M2 test suite: removing the `budget<=0`
   guard, removing the seconds tie-break in the sort comparator, and relaxing the arg-count usage
   check (the last one causes an unhandled panic when triggered). The shipped code is correct on two
   of the three (budget validation, arg-count guard both work in the current binary); the tie-break
   behavior is untested either way.
5. `grep -c 'timeout 416s' .github/workflows/ci.yml` is claimed to be 2 in the design doc, sprint
   plan, and sprint JSON `final_gates`; actual is 4 (comment prose duplicates the value). Does not
   affect the real automated gate (`TestGoTestTimeoutIsDerived`).
6. CHANGELOG.md was not touched, per `.claude/rules/coding-standards.md`'s "every change requires
   CHANGELOG.md."
7. CRLF-terminated logs are untested and undocumented as a failure mode; plausible relevance to the
   pwsh leg's `*>` redirect (unverified — no pwsh available).
8. Three semantically distinct failure conditions (missing file / anti-vacuity / below-floor) share
   exit code 1, slightly blurring the stated "two distinct non-zero exit paths" contract. All three
   fail loudly (non-silent), so this is a documentation-precision issue, not a functional one.
9. Sprint JSON never advanced past `"status": "planned"`; all milestones `"passes": false`.

### Positive / working (for calibration — not everything failed)

- M1 (derived budget + `TestGoTestTimeoutIsDerived`) is fully correct and non-vacuously gated —
  confirmed with 2 named mutations plus a full own-diff revert to the pre-M1 base, all correctly
  killed and restored.
- The bash `set -e` / `|| rc=$?` guards and the pwsh `$ErrorActionPreference` /
  `$PSNativeCommandUseErrorActionPreference` prepends — the subject of two design-quorum rounds — are
  independently verified sound, both via a local harness running the literal extracted step body and
  via the real Windows CI run.
- `go vet ./tools/ci/... ./internal/cihygiene/`, `gofmt -l tools/ci internal/cihygiene`,
  `golangci-lint run ./tools/ci/... ./internal/cihygiene/...`, and `make fmt-check` are all clean.
- `go build ./...` and `go test -count=1 ./...` both pass repo-wide (aside from the pre-existing,
  independently-confirmed `cmd/wasm` failure).
- File sizes are well within targets (168 / 191 / 68 lines).
- `git diff --name-only` matches the approved file list exactly; hard-unchanged paths
  (`internal/coordinator/`, `tools/ci/motoko_smoke.sh`, the two pre-existing cihygiene test files)
  are genuinely untouched.

---

## Gates run

| Gate | Command | rc | Result |
|---|---|---|---|
| vet | `go vet ./tools/ci/... ./internal/cihygiene/` | 0 | clean |
| unit tests | `go test -count=1 ./tools/ci/... ./internal/cihygiene/` | 0 | `ok` both packages (but see §2/§3 — vacuous against real input) |
| fmt | `gofmt -l tools/ci internal/cihygiene` | 0 | no output (clean) |
| repo fmt-check | `make fmt-check` | 0 | clean |
| lint | `golangci-lint run ./tools/ci/... ./internal/cihygiene/...` | 0 | `0 issues.` |
| full build | `go build ./...` (sprint HEAD) | 1 | fails on `cmd/wasm` only |
| full build (negative control) | `go build ./...` at pristine `origin/dev` (1b8e7eab1, via temp `git worktree`) | 1 | **identical** `cmd/wasm` failure — confirmed pre-existing, independently, not attributed to this sprint |
| full test suite | `go test -count=1 ./...` | 0 | all 128 packages `ok`, 0 `FAIL` |

---

## What I could NOT verify, and why

- **pwsh step-body execution on this rig** — pwsh is not installed (`command -v pwsh` → exit 1,
  control `command -v bash` → `/bin/bash`). Mitigated by pulling the sprint's own real
  `test-windows` job log from GitHub Actions (PR #1102, job `101837895074`), which the design itself
  names as the authoritative check for this exact reason — that log is authoritative first-party
  evidence, not a substitute I chose; it shows the wiring works and the parser doesn't.
- **The design doc's own historical provenance data** (the five-commit timing table and the original
  failing run's exact per-package seconds, V1–V5) — this is the controller's antecedent measurement
  feeding three quorum rounds, produced before this sprint's code existed. I did not re-derive it;
  it is not part of this sprint's diff.
- **Whether CRLF actually occurs on the Windows leg in practice** — plausible given PowerShell's
  handling of captured native-command output, but I could not execute pwsh to confirm one way or the
  other.

---

## Recommendation

**FAIL.** Route back to the executor (or a fresh cycle) with the fix scoped narrowly: correct `okRe`
to tolerate Go's column-padding (`^ok\s+\t` or equivalent — matching the *actual* two-space-plus-tab
shape confirmed above, ideally derived from a field-split on tabs rather than a hand-anchored prefix),
add at least one fixture in `main_test.go` built from **real, captured** `go test` output (not
hand-typed), and re-run the exact verification this report used: build the binary, feed it this
repo's own `go test -count=1 ./internal/cihygiene/... ./tools/ci/...` output, and confirm rc=0. Then
re-check the sprint's own PR CI run — that is the only check that actually settles this, and it is
free to run since the PR already exists.
