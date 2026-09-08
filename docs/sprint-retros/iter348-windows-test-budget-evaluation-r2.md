# Sprint Evaluation — iter348 `m-coordinator-windows-package-timeout-headroom` — ROUND 2

**Tree evaluated:** `1f95c084c` (detached, worktree `.eval-wt-v1-iter348-r2`)
**Base for the round-2 delta:** `73d4cee75` (round-1's tree). **Sprint base:** `39dfb6a84`.

## Verdict

**SCORE: 61/100 — FAIL** (bar: 70)

**HARD FAIL triggered:** Tests Pass. `tools/ci/headroom`'s own test suite is currently **RED on
the real target platform** — both Windows jobs on this exact commit (`test-windows` in `ci.yml`
and `Build windows-latest` in `build.yml`) fail with the identical assertion
`parse(testdata/real_go_test_output_cached.txt): got 23 records, want 128`. This is not
speculative: it is the live, current state of PR #1102 at HEAD `1f95c084c`, and I reproduced it
byte-for-byte locally (see NEW-1 below). A sprint whose entire subject is "the Windows CI leg"
currently fails its own tests **on Windows**. Round 1's core complaint — "the sprint's own PR was
red on the step this sprint wired" — is still true in round 2, for a new reason.

Round 1's three BLOCKING findings (the `^ok\t` regex, the resulting PR-red, the dead `(cached)`
branch) are genuinely fixed at the behavioral level — I attacked them adversarially and they held.
But the fix shipped a new, untested byte-level assumption (exact-match `"(cached)"` with no CRLF
tolerance) exactly analogous in kind to the byte-level assumption that caused round 1's failure,
and nobody ran it against the platform it exists for before calling the milestones "passing" in
the sprint JSON.

## Round 1 findings — adjudicated on the CURRENT tree

| # | Finding | Status | Evidence |
|---|---|---|---|
| 1 | `okRe` required `^ok\t`; real lines are `ok`+2 spaces+TAB; parser matched ZERO real records | **CLOSED** | `parse()` now splits on TAB, no anchored prefix. Applying round-1's exact regex back onto this tree (`git show 73d4cee75:tools/ci/headroom/main.go`) and running `go test ./tools/ci/headroom/...` reproduces the round-1 red (`TestParseRealGoTestOutput`: 1/128 and 0/128 records) — confirming the new tests are a real regression guard, not vacuous. Restored cleanly afterward (`git status` clean). |
| 2 | Consequence of 1: PR #1102 was red on `test`/`test-windows` at exactly this step | **PARTIALLY CLOSED / RECURRED FOR A NEW REASON** | The *original* cause (zero records parsed from a clean suite) is fixed — see NEW-1's cold-fixture PASS. But PR #1102 at `1f95c084c` is **still red on `test-windows`** (`gh api repos/sunholo-data/ailang/commits/1f95c084c/check-runs`), now because of NEW-1 (CRLF corruption of the checked-in `(cached)` fixture on Windows checkout). The *symptom* round 1 named (own-PR-red on this step) is not resolved on the tree being evaluated. |
| 3 | `(cached)` regex branch was dead code — real cached line carries no duration | **CLOSED** | `parse()` now has an explicit `fields[2] == "(cached)"` branch producing a 0-second record that still counts toward the floor; verified against a real captured cache-warm fixture (`testdata/real_go_test_output_cached.txt`, 105 `(cached)` lines, bytes confirmed via `od -c` to be genuine `go test` output, not hand-typed — see item 4 below). The branch is exercised (not dead) and is unit-tested (`TestParseRealGoTestOutput/cached`, `TestParseGoTestOutput`'s `types (cached)` case). It is, however, the same code path that NEW-1 breaks on Windows — closed in intent, fragile in execution. |
| 4 | Three mutations survived M2's suite: `budget<=0` removal, sort tie-break removal, `len(args)<3` relaxation (which panics) | **CLOSED** | Reproduced all three mutations by hand against the current `main.go` and re-ran `go test -count=1 ./tools/ci/headroom/...`: `budget<=0` removal → **KILLED** (`TestBudgetValidation`, new in round 2); tie-break removal → **KILLED** (`TestTopNTieBreak`, new); `len(args)<3` → `<2` → **KILLED**, test now catches the exact index-out-of-range panic round 1 named (`TestUsageError`). Round 2 added exactly the three named tests. |
| 5 | "exactly two hits" for `grep -c 'timeout 416s' .github/workflows/ci.yml` claimed in design doc + sprint plan + sprint JSON; real count is 4 | **PARTIAL** | The **sprint JSON** was genuinely fixed: `final_gates` now asserts `grep -c 'go test -timeout 416s' .github/workflows/ci.yml -eq 2` (a different, correct string — verified: `grep -c 'go test -timeout 416s' .github/workflows/ci.yml` → 2), and the M1 `acceptance_commands` entry was reworded to say "EXPECT: four hits" for the bare string. **The design doc (line 476) and sprint plan (line ~134) were NOT touched by this round's diff** (`git diff 73d4cee75..1f95c084c -- design_docs/.../m-coordinator-windows-package-timeout-headroom.md` has no hunk touching that region) and still assert "two hits (lines 101 and 471)" for `grep -n 'timeout 416s' .github/workflows/ci.yml`, which is verifiably false: `grep -c 'timeout 416s' .github/workflows/ci.yml` → **4** today. A round-1-named false claim survives unfixed in two of the three documents it was filed against. |
| 6 | `CHANGELOG.md` not updated | **CLOSED** | `changelogs/v0.32-current.md` (which `CHANGELOG.md` points to as canonical) gained a substantive `### Fixed` entry describing the TAB-split parser, the real-captured fixtures, and the three new mutation tests. Satisfies `.claude/rules/coding-standards.md`'s documentation rule. Note: the entry's claim that a `(cached)` line "parses as a 0-second record that still counts toward the package-count floor" is exactly the claim NEW-1 shows is false on a Windows checkout of the shipped fixture — see "Claims checked" below. |
| 7 | CRLF-terminated logs untested/undocumented | **OPEN — AND NOW ACTIVELY BLOCKING** | Not only still untested: the checked-in `(cached)` fixture is *itself* corrupted to CRLF by a real Windows `actions/checkout` (`.gitattributes` has `* text=auto` but no `eol=lf` pin for `testdata/*.txt`, unlike `*.golden`/`prompts/*.md`/`*.sh` which are pinned), and that corruption is exactly what reds `test-windows` and `Build windows-latest` right now. Round 1 filed this as a documentation gap; on this tree it is a live production defect. Escalated to **BLOCKING** (NEW-1). |
| 8 | Three failure conditions (cannot-read-log, anti-vacuity, floor) share exit code 1, contract described two | **CLOSED** | Doc comment now reads "1 — a broken INSTRUMENT, three conditions: cannot-read-log, anti-vacuity ..., or a package count below the floor. All three fail loudly." (`tools/ci/headroom/main.go` lines 9–11). Matches the code, which always returned 1 for all three; only the comment was wrong before. |
| 9 | Sprint JSON stuck at `"status": "planned"`, all milestones `"passes": false` | **PARTIAL** | Mechanically fixed: `status` → `"completed"`, all three milestones → `"passes": true` (`git diff 73d4cee75..1f95c084c -- sprint_m-coordinator-windows-package-timeout-headroom.json`). But the **substance** of the M2 and M3 "passes: true" claims is false against the tree's own live CI: M2's own test suite is red on Windows, and M3's design-designated acceptance ("Windows leg verified by the sprint PR's own test-windows job... green") is the opposite of what `gh api .../check-runs` shows for `1f95c084c`. The field was updated; the fact it asserts was not re-verified against the platform it names. |

**Re-derived diff, `73d4cee75..1f95c084c`:** matches the controller's list exactly — same 8 files,
same insert/delete counts (`git diff --stat 73d4cee75..1f95c084c`). Nothing omitted from the
controller's summary.

## NEW findings

### NEW-1 — BLOCKING: the shipped `(cached)` fixture is CRLF-corrupted on Windows checkout, and this reds the sprint's own PR right now

**Root cause:** `parse()`'s cached-line detection is an exact string match, `fields[2] == "(cached)"`
(`tools/ci/headroom/main.go`). `.gitattributes` declares `* text=auto` and pins `eol=lf` for
`*.golden`, `prompts/*.md`, `*.sh` — but **not** for `tools/ci/headroom/testdata/*.txt`. On a
Windows GitHub Actions runner (`actions/checkout`'s default `core.autocrlf=true`), every `\n` in
the checked-in `real_go_test_output_cached.txt` becomes `\r\n` on checkout. `fields[2]` for a
cached line then reads `"(cached)\r"`, which does not equal `"(cached)"`; the line falls through to
the numeric branch, `strconv.ParseFloat` fails on `"(cached)"` (Go's `strings.Fields` silently
strips the trailing `\r` as whitespace, so the failure is on the literal text, not on stray
whitespace), and the record is dropped entirely — not zero-second, just gone. 105 of 128 expected
records vanish.

**Live reproduction (this exact commit, no simulation):**
```
$ gh api repos/sunholo-data/ailang/commits/1f95c084c/check-runs --jq '.check_runs[] | select(.name=="test-windows" or .name=="Build windows-latest") | {name, status, conclusion}'
{"conclusion":"failure","name":"Build windows-latest","status":"completed"}
{"conclusion":"failure","name":"test-windows","status":"completed"}

$ gh api repos/sunholo-data/ailang/actions/jobs/101843789237/logs --allow-escape-sequences | grep -A2 'FAIL: TestParseRealGoTestOutput'
--- FAIL: TestParseRealGoTestOutput (0.00s)
    --- FAIL: TestParseRealGoTestOutput/cached (0.00s)
        main_test.go:118: parse(testdata/real_go_test_output_cached.txt): got 23 records, want 128
```

**Local reproduction (byte-identical failure, CRLF simulated by hand, no CI needed):**
```bash
python3 -c "
data = open('tools/ci/headroom/testdata/real_go_test_output_cached.txt','rb').read()
open('tools/ci/headroom/testdata/real_go_test_output_cached.txt','wb').write(data.replace(b'\n', b'\r\n'))
"
go test -count=1 -run TestParseRealGoTestOutput -v ./tools/ci/headroom/...
#     --- FAIL: TestParseRealGoTestOutput/cached (0.00s)
#         main_test.go:118: parse(testdata/real_go_test_output_cached.txt): got 23 records, want 128
# (restored from backup immediately after)
```
23 == 128 − 105 (the cached records), exactly as the corruption theory predicts, and exactly what
the live Windows job reports. Both jobs fail with the identical line number and message — this is
not a flake.

**Consequence for the design's own acceptance bar:** the design doc states "Windows leg verified by
the sprint PR's own test-windows job... a green 'Build headroom instrument' step BEFORE 'Run Go
test suite'... job verdict equals the suite's verdict" and calls this the *authoritative* check
because `pwsh` cannot be run locally. That check is currently failing.

### NEW-2 — NON-BLOCKING: the `status != "ok" && status != "FAIL"` filter is untested and currently unreachable dead weight

Removing it (`perl` mutation, restored after) leaves all 10 headroom tests green:
```bash
perl -0pi -e 's/if status != "ok" && status != "FAIL" \{\n\t\t\tcontinue \/\/ a `\?` line is not a timing record\n\t\t\}\n//' tools/ci/headroom/main.go
go test -count=1 ./tools/ci/headroom/...   # ok — SURVIVED
```
In practice this is currently harmless: every real "not ok/FAIL" line go test emits (`?` lines) has
a non-numeric third field, so the downstream `ParseFloat` failure filters it anyway. But the
in-code comment claims this guard is what rejects "a `?` line," and no test defends that specific
claim — a future line shape where field 2 is numeric-looking (unlikely but not proven impossible)
would silently slip through with no test noticing.

### NEW-3 — NON-BLOCKING (real, but latent): `len(toks) == 0` guard is untested and IS load-bearing — removing it panics

Unlike NEW-2, this one is not moot:
```bash
# with the guard removed:
go test -run TestEdgeCasesNoPanic  # -> panic: runtime error: index out of range [0] with length 0
# on input "ok\tgithub.com/x/pkg\t\n"  (a trailing tab with an empty/whitespace-only duration field)
```
No fixture in the suite exercises a truncated/empty duration field (e.g., a CI log cut off mid-line
by a killed process). The guard correctly prevents a panic on that shape today, but nothing in the
test suite says so — it's an accidental save, not a verified one. If it's ever "simplified away" as
apparently-redundant code, the tool would crash instead of producing its intended
`::error::`/anti-vacuity message on a truncated log.

### NEW-4 — NON-BLOCKING: `pkg := strings.TrimSpace(fields[1])` is untested

Removing the `TrimSpace` also leaves all tests green (survives). Real `go test` package fields
never carry stray whitespace after a TAB split, so this is very low risk, but it is, like NEW-2/3,
an assumption about byte shape with zero fixture coverage — the same class of gap that produced
both round 1's defect and NEW-1.

### NEW-5 — NON-BLOCKING: design doc / sprint plan still assert a disproven "exactly two hits" claim (see finding 5, PARTIAL)

Filed here again because it is separately citable: `design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom.md:476` and the sprint plan's corresponding passage were in scope for this round's fix pass (the JSON's sibling claim WAS fixed) but were skipped.

### NEW-6 — NON-BLOCKING: `final_gates`'s "approved file list" assertion is stale

`sprint_m-coordinator-windows-package-timeout-headroom.json`'s `final_gates` still says: *"git diff
--name-only against base matches the approved file list: .github/workflows/ci.yml,
internal/cihygiene/workflow_go_test_timeout_test.go, tools/ci/headroom/main.go,
tools/ci/headroom/main_test.go"* — four files. The real diff against `39dfb6a84` touches ten files
(adds `changelogs/v0.32-current.md`, both design docs, the sprint JSON itself, and two testdata
fixtures):
```bash
git diff --name-only 39dfb6a84..1f95c084c
# .github/workflows/ci.yml
# changelogs/v0.32-current.md
# design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom-sprint-plan.md
# design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom.md
# internal/cihygiene/workflow_go_test_timeout_test.go
# sprint_m-coordinator-windows-package-timeout-headroom.json
# tools/ci/headroom/main.go
# tools/ci/headroom/main_test.go
# tools/ci/headroom/testdata/real_go_test_output.txt
# tools/ci/headroom/testdata/real_go_test_output_cached.txt
```
Harmless in isolation (docs/testdata additions are clearly in scope), but it is a self-check
command in the sprint's own contract that would fail if run literally, and nobody appears to have
run it literally.

## Fixture honesty check (deliverable item 4)

`TestParseRealGoTestOutput`'s two fixtures were checked for authenticity, not taken on faith:
- `testdata/real_go_test_output.txt`: byte-level inspection (`od -c`) confirms genuine `go test`
  padding (`ok` + 2 spaces + TAB) and a real, specific flaky-test transcript —
  `--- FAIL: TestSolve_HardTimeout_FakeSolverIgnoringT`, citing `ailang#602` by issue number, with
  three realistic retry-log lines. This is not something a model would plausibly hand-fabricate;
  it matches a real, reproducible local flake in `internal/smt` (confirmed live: I hit the same
  flaky test twice in my own full-suite runs, then it passed clean on retry — see Gates below).
- `testdata/real_go_test_output_cached.txt`: 128 `ok` lines, 105 literally reading `(cached)`,
  matching the test's own asserted counts exactly.
- **Round-1-bug defense proven**, per the directive's explicit instruction: I restored round 1's
  exact `^ok\t`/`^FAIL\t` regex parser (`git show 73d4cee75:tools/ci/headroom/main.go`) onto the
  current test file and reran — both `TestParseGoTestOutput` and `TestParseRealGoTestOutput` go red
  (1 and 0 records parsed, respectively, instead of 5 and 128). The new test is NOT undefended.
  Restored cleanly afterward (`git status` clean, confirmed).

## Mutation matrix

Applied by hand against `tools/ci/headroom/main.go` on this tree, one at a time, each followed by
`go test -count=1 ./tools/ci/headroom/...` and a restore from a pre-mutation backup (never
`git checkout --`).

| Mutation | Named in plan? | Result |
|---|---|---|
| Remove `budget <= 0` validation | Yes (round 1 #4) | **KILLED** — `TestBudgetValidation` |
| Remove sort tie-break | Yes (round 1 #4) | **KILLED** — `TestTopNTieBreak` |
| Relax `len(args) < 3` → `< 2` | Yes (round 1 #4) | **KILLED** — `TestUsageError` (catches the exact panic round 1 named) |
| Restore round-1's `^ok\t`/`^FAIL\t` regex parser wholesale | Implicit (the whole point of round 2) | **KILLED** — `TestParseGoTestOutput`, `TestParseRealGoTestOutput` both red |
| Remove the `(cached)` special-case branch | No | **KILLED** — `TestParseGoTestOutput`, `TestParseRealGoTestOutput/cached` |
| Relax `len(fields) < 3` → `< 2` | No | **KILLED** — panics (`index out of range [2]`) inside `TestParseGoTestOutput` |
| Flip `failed: status == "FAIL"` → always `false` | No | **KILLED** — `TestParseGoTestOutput`/`TestParseRealGoTestOutput` FAIL-count assertions |
| Remove `status != "ok" && status != "FAIL"` filter | No | **SURVIVED** (NEW-2, non-blocking — currently moot downstream) |
| Remove `len(toks) == 0` guard | No | **SURVIVED** by the shipped suite, but **DOES panic** on an untested edge case I constructed (NEW-3) |
| Remove `pkg := strings.TrimSpace(...)` | No | **SURVIVED** (NEW-4, non-blocking, low real-world risk) |
| Change cached `secondsStr` from `"0"` to `"(cached)"` | No | **SURVIVED** (cosmetic, no test asserts the report line's literal text for a cached row) |

7 named/implicit mutations, all killed. Of 5 additional mutations I derived from the round-2 diff
itself (not named anywhere in the plan), 4 survived — one of which (NEW-3) is a genuine crash path.

## CI wiring re-verification (deliverable item 2)

Re-verified M1's gate is unaffected by this round (`internal/cihygiene`'s
`TestGoTestTimeoutIsDerived` passes; `go vet`/`gofmt` clean) and re-ran the bash exit-code
composition logic using the **literal** Linux step body extracted from `ci.yml` (not retyped):

```bash
start=$(grep -n '^      run: |$' .github/workflows/ci.yml | head -1 | cut -d: -f1)
sed -n "$((start+1)),$((start+12))p" .github/workflows/ci.yml | sed 's/^        //' > /tmp/literal_step_body.sh
```
Ran it three times with a `fakebin/go` stub on `PATH` (per the design doc's own fakebin recipe),
never editing the extracted script:

| Scenario | fake `go` behavior | `step_rc` | Result |
|---|---|---|---|
| Passing suite (60 fast packages) | prints fixture, exit 0 | **0** | Report printed, no warnings — correct |
| Failing suite with a panic (simulated timeout + FAIL line) | prints fixture, exit 1 | **1** | Panic text visible in captured log (`cat`), HEADROOM report still printed with `[go test FAIL]` marker and a 100%-budget `::warning::`, `go_rc` correctly takes precedence — correct |
| `go test` itself exits 0 but the log is garbage (anti-vacuity case) | prints garbage, exit 0 | **1** | `hr_rc` (from the tool's own anti-vacuity exit 1) correctly propagates even though `go_rc=0` — correct |

The bash exit-code composition (`go_rc` precedence, then `hr_rc`) is sound and unchanged by this
round's diff (`ci.yml` is not in the `73d4cee75..1f95c084c` diff at all). The **pwsh leg could not
be executed** (`command -v pwsh` → not found, confirmed, not invented) — a structural read of the
Windows step body shows equivalent `$go_rc`/`$hr_rc`/precedence logic, but this is read-only
verification, not an execution. The design designates the live `test-windows` job as authoritative
for the pwsh leg precisely because of this gap, and that job is failing (NEW-1) — so this leg is
**NOT independently confirmed sound** this round, only structurally plausible.

## Claims checked against sources (deliverable item 5)

- **Commit message / CHANGELOG:** "a `(cached)` line ... parses as a 0-second record that still
  counts toward the package-count floor" — **true of the code, false in practice for this shipped
  tree on Windows** (NEW-1). The claim describes intent correctly but was never checked against the
  platform where it matters.
- **Sprint JSON `final_gates`:** `grep -c 'go test -timeout 416s' .github/workflows/ci.yml -eq 2` —
  verified true (`grep -c 'go test -timeout 416s' .github/workflows/ci.yml` → 2).
  `grep -c 'go_test_headroom.log' .github/workflows/ci.yml -ge 6` — verified true (→ 6).
  Hard-unchanged-paths check (`internal/coordinator/`, `tools/ci/motoko_smoke.sh`,
  `internal/cihygiene/workflow_timeouts_test.go`, `internal/cihygiene/gate_wiring_test.go`) —
  verified true (`git diff 39dfb6a84 1f95c084c --exit-code -- ...` → exit 0, no diff).
- **Sprint JSON milestone `"passes": true` (M2, M3):** **false** against the tree's own live CI —
  see NEW-1, finding 9.
- **Design doc / sprint plan "exactly two hits" for `grep -n 'timeout 416s'`:** **false**, real
  count is 4 (finding 5 / NEW-5).

## Gates run (all rc captured via `rc=$?` immediately, no pipes)

| Gate | Command | rc | Result |
|---|---|---|---|
| `go vet` | `go vet ./tools/ci/... ./internal/cihygiene/` | 0 | clean |
| headroom+cihygiene tests | `go test -count=1 ./tools/ci/... ./internal/cihygiene/` | 0 | clean (on this — non-Windows — machine) |
| `gofmt` | `gofmt -l tools/ci internal/cihygiene` | 0 | clean, empty output |
| `golangci-lint` | `golangci-lint run ./tools/ci/... ./internal/cihygiene/...` | 0 | "0 issues" (CI's own `lint` job also green) |
| full suite | `go test -count=1 -timeout 416s ./...` | 1 (twice), 0 on isolated retry | `internal/smt`'s `TestSolve_HardTimeout_FakeSolverIgnoringT` — a KNOWN pre-existing flake (the fixture's own comment cites `ailang#602`); confirmed unrelated to this sprint by isolating and rerunning (`go test -timeout 60s ./internal/smt/...` → clean). Controller's quoted rc=0 run simply didn't hit the flake — both readings are consistent with a timing-sensitive flake, not a wrong measurement. |
| `go build ./...` | `go build ./...` | 1 | `cmd/wasm`: "function main is undeclared" — reproduced directly on this tree; matches round 1's documented pristine-`origin/dev` negative control exactly (citing round 1 for the pristine comparison, not re-running it; I did run it fresh here rather than only citing) |
| PR CI | `gh api repos/sunholo-data/ailang/commits/1f95c084c/check-runs` | — | `test-windows`: **failure**. `Build windows-latest` (separate `build.yml` workflow, same commit): **failure**, same root cause (`TestParseRealGoTestOutput/cached`). `test` (Linux, ci.yml): **in_progress** at time of writing — polled for ~5 minutes without completion; not blocking the verdict since NEW-1 is independently reproduced and decisive on its own. `lint`, `govulncheck`, `CodeQL`, `Analyze Go`, macOS/Linux/ubuntu builds, `launchd drivers (bash 3.2)`: all **success**. |

## What I could NOT verify, and why

- **The Windows `pwsh` leg's exit-code composition, by direct execution.** `pwsh` is not installed
  on this machine (`command -v pwsh` → exit 1, confirmed, not assumed). I read the script
  structurally instead; it mirrors the bash leg's `$go_rc`/`$hr_rc` precedence pattern. **NOT RUN.**
- **The Linux `test` job's final conclusion on `1f95c084c`.** Still `in_progress` after ~5 minutes
  of polling within this evaluation's time budget; I did not block my report on it (per the
  "bounded waits only, never end the turn on a wait" rule). Given `.gitattributes` does not force
  `eol=lf` on the fixtures but Linux `actions/checkout` does not rewrite line endings by default
  (unlike Windows's `core.autocrlf=true`), I expect it to pass NEW-1's specific defect, but this is
  an expectation, not a verified result — re-check with:
  `gh api repos/sunholo-data/ailang/commits/1f95c084c/check-runs --jq '.check_runs[] | select(.name=="test")'`.
- **Whether a `?` line with a numeric-looking third field can occur in real Go tooling** (relevant
  to NEW-2). I did not find one in any Go version's documented output shapes and treated the risk
  as theoretical/low, but I did not exhaustively audit the Go toolchain source for every verbose or
  future flag combination.
- **The macOS jobs' relationship to `tools/ci/headroom`'s tests** — both `Build macos-latest` jobs
  report `success`, but I did not fetch their full logs to confirm they exercise this package at
  all (build.yml may only build, not test, on macOS). Not needed for the verdict either way.

## Summary for the controller

Round 2 correctly fixed the three BLOCKING defects named in round 1, and fixed four of five
NON-BLOCKING ones fully (4, 6, 8, and half of 9 mechanically). But it shipped a **new** untested
byte-level assumption of the exact same shape and severity as the one round 1 caught — and this one
is not hypothetical: it is failing the sprint's own PR on Windows, in the workflow this sprint
exists to fix, right now. Round 1's finding 7 predicted almost exactly this outcome and was
filed as non-blocking; it should be treated as the headline defect for round 3.

**Minimum fix for round 3:** either (a) trim `\r` before the `(cached)` exact-match comparison (and
audit every other exact-match/suffix comparison in `parse()` for the same assumption), or (b) pin
`tools/ci/headroom/testdata/*.txt` to `eol=lf` in `.gitattributes` (matching the existing
`*.golden`/`*.sh` precedent) so the checked-in fixture round-trips identically on every runner —
and add a fixture or synthetic test that actually exercises a CRLF-terminated line, so this class of
defect cannot recur a third time undetected. Also: finish the "two hits → four hits" text fix in
the design doc and sprint plan (round 2 only fixed the JSON copy), and re-verify the sprint JSON's
`"passes": true` milestones against actual CI before claiming completion again.
