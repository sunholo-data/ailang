# SPRINT PLAN — M-COORDINATOR-WINDOWS-PACKAGE-TIMEOUT-HEADROOM

**Design doc (approved, quorum round 3):**
`design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom.md`
**Machine-readable sprint object:** `sprint_m-coordinator-windows-package-timeout-headroom.json`
**Worktree:** `.planner-wt-v1-iter348` (detached), HEAD `30d11b0bd` — the design doc commit itself
(its first-party measurements were taken at `1b8e7eab1`; the workflow claims below were
re-verified against THIS worktree's HEAD).
**Milestones:** 3 (M1 budget + gate · M2 headroom tool · M3 wiring) · **Estimated:** 2 days (M1
0.5d / M2 0.5d / M3 1d)

---

## 0. Planning-session verification — every ci.yml line the doc names, checked against THIS worktree

The executor must work from the **Observed** column, not from the doc, where they differ.
All commands run 2026-09-07 in this worktree at `30d11b0bd`.

| # | Doc claim | Command run | Observed output (trimmed) | Verdict |
|---|---|---|---|---|
| P1 | `-timeout 300s` at **ci.yml:101** (Linux) and **:471** (Windows); only other 60s is `ailang check` at :292 | `grep -n '\-timeout' .github/workflows/ci.yml` | `86:    # -timeout is PER TEST BINARY ...` · `101:        go test -timeout 300s ./...` · `292: ... ailang check --timeout 60s ...` · `471:        go test -timeout 300s ./...` | **MATCHES exactly** |
| P2 | stale comment block ci.yml:86-88 | `nl -ba ci.yml \| sed -n '86,92p'` | block is four lines, **86–89**: 86–88 the history, 89 `# 300s still catches genuine hangs (M-DX11's intent) without the flake.`; step name `Run tests with timeout (M-DX11)` at **90** | **STALE — doc one line short; actual block is 86–89** |
| P3 | stale "(vs 60s on Linux)" at ci.yml:456 | `nl -ba ci.yml \| sed -n '454,471p'` | step `Run Go test suite` at 454, `shell: pwsh` 455, stale comment spans **456–458**; **459–460 is a separate, valid instruction** ("If a test is genuinely Windows-incompatible … `// TODO(windows-ci)`") that must be KEPT; env 461–465, `run: |` 466, poison checks 467–470, `go test -timeout 300s ./...` at 471 | **MATCHES for the cited line; refined range 456–458, keep 459–460** |
| P4 | gated-integration steps at ci.yml:111 and :480, log `gated_integration.log` | `grep -n 'gated_integration.log\|Assert binary-gated' ci.yml` + `nl -ba` of 108–116 / 477–483 | Linux: step NAME at **109**, `run: |` 110, `go test` line 111, `tee gated_integration.log` at **112**; Windows: step NAME at **477**, the `Tee-Object -FilePath gated_integration.log` line at **480** | **Linux slightly stale** (doc's 111 is the step's first command line; step itself is 109, log first appears 112); **Windows 480 MATCHES** |
| P5 | build precedent at ci.yml:627 | `grep -n 'go build -o' ci.yml`; `sed -n '626,628p'` | `627:      run: go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter` (step name `Build filter` at 626) — **quoted verbatim** | **MATCHES exactly** |
| P6 | `tools/ci/` has only `motoko_smoke.sh`; no `go test` parser exists | `ls -la tools/ci/`; `grep -rln 'go test\|--- PASS\|^ok[[:space:]]' tools/ci/`; control `head -3 tools/ci/*` | one file, `motoko_smoke.sh` (1771 B, exec); grep exit 1 (no matches); control prints its shebang + header comment | **MATCHES (V14)** — new tool under `tools/ci/headroom/` is warranted |
| P7 | pwsh not installed on this rig (V16/V22) | `command -v pwsh`; `command -v bash` | pwsh: exit 1, no output; control bash: `/bin/bash` | **MATCHES** — Windows leg unverifiable locally; see M3 |
| P8 | green-gate command coverage | `go list ./tools/ci/...`; `go test ./tools/ci/...` | `go: warning: "./tools/ci/..." matched no packages` (list rc=0); `no packages to test` (**test rc=1**) before M2 exists | **M1's green gate must omit `./tools/ci/...`** — measured; see M1 acceptance |
| P9 | job ids for the gate test | `grep -n '^  [a-z][a-z0-9_-]*:' ci.yml` | `17:  test:` · `391:  test-windows:` | **MATCHES** — gate targets jobs `test` and `test-windows` |
| P10 | clean tree, module path | `git log --oneline -1`; `git status --porcelain`; `head -1 go.mod` | HEAD `30d11b0bd docs(design): m-coordinator-windows-...`; status empty; `module github.com/sunholo-data/ailang` | tool import path: `github.com/sunholo-data/ailang/tools/ci/headroom` |
| P11 | cihygiene test-pattern reuse | `grep -m1 '^package' internal/cihygiene/*_test.go`; `sed -n '40,110p' workflow_timeouts_test.go` | both files `package cihygiene` (internal test package); `loadWorkflows(t)` helper + `workflow` struct (yaml.v3, `workflowDir = "../../.github/workflows"`) load and parse **every** workflow file with anti-vacuity asserts | M1's gate test **reuses `loadWorkflows` + `workflow`**; M3's YAML edits are syntax-gated by the existing tests |

**Post-edit line drift (both M1 and M3):** the replacement comment blocks are longer than the
blocks they replace (P2/P3), so the `go test` lines move OFF 101/471 after M1, and everything in
M3 shifts again. The doc's M1 acceptance ("two hits (lines 101 and 471)") is correct about the
HIT COUNT, not the line numbers — assert the count and the values, never fixed line numbers.

## 1. Executor contract and global constraints

- **No git write operations by the executor.** The controller commits, one commit per milestone.
  Read-only git (`status`/`diff`/`show`) is fine. Mutation checks below restore files via
  `cp` backups in `/tmp`, never `git checkout`.
- **No production `internal/coordinator` changes** anywhere in this sprint (design non-goal).
- **The instrument never re-runs the test suite**; it consumes the log the `go test ./...` step
  already produces.
- **Exit-code discipline:** neither leg may swallow `go test`'s output or exit code. Both shells
  need their guards — bash: `|| rc=$?` on every command whose failure must not abort the step;
  pwsh: `$ErrorActionPreference = 'Continue'` + `$PSNativeCommandUseErrorActionPreference = $false`
  prepended before the first guarded native command.
- **Two distinct non-zero exit paths for the tool, kept distinct in code AND tests:**
  (i) a slow package is a data point — WARN-ONLY, exit 0; (ii) a broken instrument — anti-vacuity
  (non-empty log → 0 packages) or below-floor count — `::error::` + exit 1. The tool's only other
  non-zero is usage error (exit 2). No FAIL tier ships.
- **The tool is BUILT before use**, output to `$RUNNER_TEMP`, before the `go test` step in each
  job. Precedent verified at P5: ci.yml:627 `run: go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter`.
  The build step is intentionally UNGUARDED — a broken instrument fails the job loudly
  (bash `set -e` does it on Linux; explicit `if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }` on pwsh).
- **Every milestone is independently committable and leaves the tree green.** Gates:
  - M1: `go build ./... && go test -count=1 ./internal/cihygiene/`
    (`./tools/ci/...` matches no Go packages before M2 — measured at P8, `go test` exits rc=1 with
    "no packages to test"; do not read that as a failure of the milestone).
  - M2, M3: `go build ./... && go test -count=1 ./internal/cihygiene/ ./tools/ci/...`
- **`-count=1` on cihygiene runs, always.** Those tests `os.ReadFile` workflow files from OUTSIDE
  the package directory; the Go test cache does not track those inputs and will happily report a
  stale `ok` after a workflow edit. (Package-level fact, pattern from P11.)
- Local shell for acceptance commands: **bash, not zsh** (process substitution `<(...)` and
  `|| rc=$?` semantics below are written for bash; the rig's `/bin/bash` confirmed at P7).
  Local runs: `export RUNNER_TEMP="${RUNNER_TEMP:-/tmp}"` first.

---

## M1 — the derived budget (workflow comment + value + cihygiene gate) — 0.5d

### Files

| File | Action |
|---|---|
| `.github/workflows/ci.yml` | **Modify.** Replace comment block lines **86–89** (P2) with the derivation comment; change `-timeout 300s` → `-timeout 416s` at line **101**. Replace comment lines **456–458** (P3; **keep 459–460**) with the short-form derivation comment; change `-timeout 300s` → `-timeout 416s` at line **471**. |
| `internal/cihygiene/workflow_go_test_timeout_test.go` | **Create.** Package `cihygiene`; test `TestGoTestTimeoutIsDerived`. |

### Ordered steps

1. **Linux comment** (replaces lines 86–89, immediately above `- name: Run tests with timeout (M-DX11)`).
   Insert exactly:

   ```yaml
       # -timeout 416s is a DERIVED, PROVISIONAL operational budget (right-censored
       # input — see design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom.md):
       #   budget = worst steady-state package x runner variance x safety factor
       #          = 172.5s (cmd/ailang, slowest pkg on non-failing dev run e5a325a20)
       #          x 1.80x  (runner variance: aggregate package-seconds 599.7 -> 1080.3,
       #                    72f9cfeca vs 8e3927950)
       #          x 1.34x  (per-package excess variance the aggregate hides; a FLOOR —
       #                    coordinator was cut off at the old 300s ceiling on 72f9cfeca)
       #          = 416.1s -> 416s
       # The old 300s sat INSIDE the measured noise band (172.5 x 1.80 = 310.5s > 300s).
       # Still catches genuine hangs (M-DX11's intent); DRIFT is caught by the headroom
       # instrument (tools/ci/headroom), not by this value.
   ```

2. **Linux value:** line 101 `-timeout 300s` → `-timeout 416s` (leaves the four poison-check
   lines 97–100 untouched).
3. **Windows comment** (replaces lines 456–458 only; the Windows-incompat instruction 459–460
   stays **verbatim**):

   ```yaml
         # -timeout 416s: same DERIVED PROVISIONAL budget as the Linux test job —
         # 172.5s worst steady-state package x 1.80x runner variance x 1.34x safety
         # factor = 416.1s; derivation in the test job's comment and
         # design_docs/planned/v0_36_0/m-coordinator-windows-package-timeout-headroom.md.
         # (The previous "5min (vs 60s on Linux)" note was stale: Linux has used the
         # same 300s since M-DX11.)
   ```

4. **Windows value:** line 471 `-timeout 300s` → `-timeout 416s` (poison checks 467–470 untouched).
5. **Gate test** `internal/cihygiene/workflow_go_test_timeout_test.go`, `package cihygiene`,
   reusing `loadWorkflows(t)` and the `workflow` struct from `workflow_timeouts_test.go`
   (same package — P11). `TestGoTestTimeoutIsDerived` asserts:
   - **Value (YAML parse):** job `test` has a step whose `Run` contains `go test -timeout 416s ./...`;
     job `test-windows` has a step whose `Run` contains `go test -timeout 416s ./...`. Assert by
     presence-of-match over both jobs' steps (not by step index — steps shift).
   - **Anti-revert:** NO step in either job's `Run` contains `go test -timeout 300s ./...`.
   - **Comment (raw text):** YAML comments are NOT part of the parsed `Run` payloads, so the
     derivation comment can only be checked on raw bytes: `os.ReadFile("../../.github/workflows/ci.yml")`
     must contain all three tokens `172.5`, `1.80`, `1.34`. Missing any token → `t.Errorf` naming
     the token and the derivation requirement.
6. Run M1 acceptance (below). Then the mutation check (below). Tree must be green before handoff.

### Acceptance commands and expected output

```bash
grep -n 'timeout 416s' .github/workflows/ci.yml
```
Expected: exactly **two** hits, both `go test -timeout 416s ./...` (one in job `test`, one in job
`test-windows`). Line numbers will be ≥101 and ≥471 — comment growth shifts them (P-drift note);
the count and content are the assertion, not the numbers.

```bash
grep -n '172.5\|1.80\|1.34' .github/workflows/ci.yml
```
Expected: hits inside BOTH comment blocks (Linux block above `Run tests with timeout (M-DX11)`;
Windows block inside `Run Go test suite`) — at minimum the three derivation factors appear in the
Linux block and in the Windows short form.

```bash
go test -count=1 ./internal/cihygiene/
```
Expected: `ok  github.com/sunholo-data/ailang/internal/cihygiene <N>s` — with
`TestGoTestTimeoutIsDerived` running and passing.

### Mutation killed, and how to prove it

**Mutation:** revert `-timeout 416s` → `-timeout 300s` in either leg, or delete the derivation
comment → `TestGoTestTimeoutIsDerived` turns red. This is the guard that stops the budget
silently drifting back to an unexplained constant.

```bash
cp .github/workflows/ci.yml /tmp/ci.yml.m1bak
sed -i '' 's/-timeout 416s/-timeout 300s/' .github/workflows/ci.yml    # darwin sed; GNU sed: sed -i
go test -count=1 -run TestGoTestTimeoutIsDerived ./internal/cihygiene/  # EXPECT: FAIL
cp /tmp/ci.yml.m1bak .github/workflows/ci.yml                           # restore (NO git checkout)
go test -count=1 ./internal/cihygiene/                                  # EXPECT: ok
```
Second arm (comment deletion): `sed -i '' '/172.5/d' .github/workflows/ci.yml` → gate red →
restore with the same `cp` → green.

### NOT in this milestone

- No `tools/ci/headroom/` code (M2). No build step, no headroom invocation, no step-body
  restructuring (M3). No changes to the gated-integration steps (P4: step names at 109/477;
  never touched). No job-level `timeout-minutes` or `-p` changes. Nothing under
  `internal/coordinator/`.

---

## M2 — the headroom instrument (`tools/ci/headroom` + unit tests) — 0.5d

### Files

| File | Action |
|---|---|
| `tools/ci/headroom/main.go` | **Create.** Parser + report + WARN-ONLY thresholds + anti-vacuity guard + package-count floor. |
| `tools/ci/headroom/main_test.go` | **Create.** Six named tests over fixtures (see test-plan mapping below). |

### Interface contract (executor MUST implement to this contract — the doc's acceptance commands depend on it)

- **CLI:** `headroom <logfile|-> <budget-seconds>` — log path FIRST, budget SECOND (every doc
  example, e.g. `headroom go_test_headroom.log 416`). `-` reads stdin. Usage error → exit **2**.
- **Records parsed (duration mandatory, suffixes tolerated):**
  `ok\t<pkg>\t<N.NNN>s [ (cached)] [ [no tests to run]]` and `FAIL\t<pkg>\t<N.NNN>s`.
  A `FAIL <pkg> [build failed]` line, a bare `FAIL`, and every other line shape parse to NO
  record — that is what the runtime guard is for (P6/V11 shapes are the fixtures).
- **Constants:** `budgetWarnPct = 75`, `budgetHighWarnPct = 90`, `topN = 5`, `minPackages = 50`.
- **Report (stdout, per the doc's example):** header
  `HEADROOM: top 5 slowest packages (budget 416s)`; ranked rows `pkg seconds (pct% of budget)`
  with ` [go test FAIL]` appended when the package's line was `FAIL`; sorted by seconds
  descending, ties broken by package name ascending (deterministic CI logs).
  Percent = `round(100 * seconds / budget)` (both doc fixtures agree: 100/416→24, 380/416→91).
- **WARN tiers — WARN-ONLY, both report, NEITHER reds the job:**
  pct ≥ 75 → `::warning::headroom: <pkg> at <pct>% of budget (warn threshold 75%)`;
  pct ≥ 90 → a higher-severity `::warning::` naming the 90% tier. Exit stays 0.
- **Anti-vacuity guard (exit 1):** input non-empty (≥1 byte) AND zero records → stdout gets
  `::error:: headroom: parsed 0 packages from non-empty log — parser may be stale`, exit 1.
- **Package-count floor (exit 1):** record count < `minPackages` → stdout gets
  `::error:: headroom: parsed <N> packages (< 50) — parser may be stale`, exit 1.
  Check order: zero-on-non-empty first (its message), then the floor. (An EMPTY log therefore
  fails via the floor message — 0 < 50 — which is correct: a legit full-suite run reports
  ~127–131 packages, V19.)
- **Structure:** `main()` = `os.Exit(run(os.Args, os.Stdin, os.Stdout))` over a testable
  `run(args []string, stdin io.Reader, stdout io.Writer) int`, so exit paths are unit-tested
  without `os.Exit`.
- Annotation lines (`::error::`/`::warning::`) go to STDOUT (GitHub scrapes step stdout).

### Ordered steps

1. `mkdir -p tools/ci/headroom`; write `main.go` to the contract above.
2. Write `main_test.go` — the six tests, all fixtures inline (no network, no `go test`
   invocation — the tool NEVER re-runs the suite). Every fixture uses ≥50 packages where the
   floor must pass (generate 60 synthetic `ok` lines in-test, then perturb).
3. Run M2 acceptance (below), then the three mutation checks (below).

### Acceptance commands and expected output

Setup for local runs (rig has no Actions-provided `RUNNER_TEMP`):

```bash
export RUNNER_TEMP="${RUNNER_TEMP:-/tmp}"
```

```bash
go test -count=1 ./tools/ci/headroom/
```
Expected: `ok  github.com/sunholo-data/ailang/tools/ci/headroom <N>s` — all six tests pass.

**Clean-checkout build + invoke** (objection 4's defect — the instrument must be BUILT, then the
BINARY invoked, never the source dir). Uses the checked-in COLD fixture
`tools/ci/headroom/testdata/real_go_test_output.txt` (128 records — 127 `ok` + 1 `FAIL` — ≥ floor
50):

```bash
rm -f "$RUNNER_TEMP/headroom"                 # simulate clean checkout: no binary present
go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom && test -x "$RUNNER_TEMP/headroom"
"$RUNNER_TEMP/headroom" tools/ci/headroom/testdata/real_go_test_output.txt 416; echo "rc=$?"
```
Expected: report headed `HEADROOM: top 5 slowest packages (budget 416s)` naming this repo's
actual slowest packages; **rc=0** (128 records ≥ floor 50; nothing at/over 75%).

**Cached-run proof** (a `(cached)` line has NO duration and must still count toward the floor).
Uses the checked-in CACHE-WARM fixture `tools/ci/headroom/testdata/real_go_test_output_cached.txt`
(128 `ok` records, 105 of them `(cached)`):

```bash
"$RUNNER_TEMP/headroom" tools/ci/headroom/testdata/real_go_test_output_cached.txt 416; echo "rc=$?"
```
Expected: report printed; **rc=0** (the 105 `(cached)` lines parse as 0-second records that still
count toward the floor, so a cache-warm run does not red the job for the wrong reason).

**WARN-ONLY proof** (a slow package is a data point, not a failure). Uses an inline 60-package
fixture (59 at 100 s + one at 380 s):

```bash
"$RUNNER_TEMP/headroom" <(for i in $(seq 1 59); do printf 'ok\tgithub.com/x/pkg%d\t100s\n' "$i"; done; printf 'ok\tgithub.com/x/pkg60\t380s\n') 416; echo "rc=$?"
```
Expected: report lists `github.com/x/pkg60 380s (91% of budget)` and a `::warning::` line naming
the 90% tier; **rc=0**.

**Anti-vacuity (exit path ii-a):**

```bash
"$RUNNER_TEMP/headroom" <(printf 'garbage\nnot a go test line\n') 416; echo "rc=$?"
```
Expected: `::error:: headroom: parsed 0 packages from non-empty log — parser may be stale`;
**rc=1** (non-zero — a broken instrument, not a slow package).

**Package-count floor (exit path ii-b):**

```bash
"$RUNNER_TEMP/headroom" <(for i in $(seq 1 3); do printf 'ok\tgithub.com/x/pkg%d\t100s\n' "$i"; done) 416; echo "rc=$?"
```
Expected: `::error:: headroom: parsed 3 packages (< 50) — parser may be stale`; **rc=1**.

### Mutations killed, and how to prove them

Test↔mutation mapping (from the design's test plan):

| Test | Asserts | Mutation it kills |
|---|---|---|
| `TestParseGoTestOutput` | parses `ok`/`FAIL`/`(cached)`/`[no tests to run]` lines | removing the `FAIL`-line branch — a failed package vanishes from the report |
| `TestPercentOfBudget` | 100 s / 416 s = 24% | wrong divisor / off-by-one in % |
| `TestThresholds` | 75% and 90% both warn, exit 0 | changing a threshold constant / making a tier red the job |
| `TestTopN` | top-5, descending | sorting bug / wrong N |
| `TestEmptyParseOnNonEmptyInput` | garbage → `::error::` + exit 1 | removing the anti-vacuity guard — format drift silently disables the instrument |
| `TestPackageCountFloor` | 3-of-130-style parse → `::error::` + exit 1 | removing the floor — a parser finding 3 of 130 packages is treated as healthy |

Apply/revert pattern for each (no git writes):

```bash
cp tools/ci/headroom/main.go /tmp/headroom-main.bak
# arm 1 — kill the FAIL branch:      sed -i '' 's/^\s*failRe.*$//' tools/ci/headroom/main.go  (or equivalent edit removing the FAIL alternative)
# arm 2 — kill anti-vacuity:         neutralise the zero-record guard (e.g. make its condition constant-false)
# arm 3 — kill the floor:            sed -i '' 's/minPackages = 50/minPackages = 0/' tools/ci/headroom/main.go
go test -count=1 -run 'TestParseGoTestOutput|TestEmptyParseOnNonEmptyInput|TestPackageCountFloor' ./tools/ci/headroom/   # EXPECT: the mapped test FAILs
cp /tmp/headroom-main.bak tools/ci/headroom/main.go
go test -count=1 ./tools/ci/headroom/                                                                   # EXPECT: ok
```
(The sed patterns reference the executor's own identifiers — adjust names to the code as written;
the REQUIRED property is: each mutation turns exactly its mapped test red, and restore turns it
green again.)

### NOT in this milestone

- No workflow edits at all (M3 wires it; M1 already changed values). No cihygiene changes.
- No blocking/FAIL threshold tier (calibration follow-up in the design's out-of-scope rows).
- The tool is never invoked by CI in this milestone; the binary is built only by the acceptance
  commands above.

---

## M3 — wire the instrument into both CI legs, preserving exit codes — 1d

### Files

| File | Action |
|---|---|
| `.github/workflows/ci.yml` | **Modify only.** Four insertions/edits, all in jobs `test` and `test-windows`. (Line numbers below are POST-M1 and will re-shift as blocks are inserted — anchor on step names, not numbers.) |

### Ordered steps

1. **Linux build step** — insert immediately BEFORE the derivation comment block that heads
   `- name: Run tests with timeout (M-DX11)` (so the derivation comment stays attached to the
   test step; P2 anchor):

   ```yaml
       - name: Build headroom instrument
         run: go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom

   ```

   Intentionally unguarded — `set -e` makes a non-zero build fail the job loudly. This follows
   the repo's existing precedent, quoted from P5/this worktree at ci.yml:627:
   `run: go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter` (step `Build filter`),
   with the deliberate deviation **output to `$RUNNER_TEMP`, not `./bin/`** (per-job temp dir:
   tree hygiene, two-job isolation, fresh-build guarantee — design §(b)).

2. **Windows build step** — insert immediately BEFORE `- name: Run Go test suite` (P3 anchor):

   ```yaml
       - name: Build headroom instrument
         shell: pwsh
         run: |
           go build -o "$env:RUNNER_TEMP/headroom.exe" ./tools/ci/headroom
           if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

   ```

   Unguarded by preference-prepends — the explicit `exit $LASTEXITCODE` is the loud failure.

3. **Linux test step body** — keep the env block and the four poison checks unchanged; replace
   the single line `go test -timeout 416s ./...` (the M1-updated line) with:

   ```bash
           go_rc=0
           hr_rc=0
           go test -timeout 416s ./... > go_test_headroom.log 2>&1 || go_rc=$?
           cat go_test_headroom.log                      # panic and FAIL/ok lines stay visible
           "$RUNNER_TEMP/headroom" go_test_headroom.log 416 || hr_rc=$?
           if [ $go_rc -ne 0 ]; then exit $go_rc; fi     # a failing suite still fails the job
           exit $hr_rc
   ```

   Redirect-and-capture, **no pipe** — the pipeline's exit code would be `tee`/`cat`'s; here
   `$?` is read right off the `go test` command. Both guarded commands (`go test`, headroom)
   carry `|| rc=$?` so `set -e` aborts nothing (V17 semantics, verified).

4. **Windows test step body** — keep the four poison checks unchanged (they exit via explicit
   `exit 1`, so preference changes below cannot make them silent); replace
   `go test -timeout 416s ./...` with:

   ```powershell
           $ErrorActionPreference = 'Continue'
           $PSNativeCommandUseErrorActionPreference = $false
           go test -timeout 416s ./... *> go_test_headroom.log
           $go_rc = $LASTEXITCODE
           Get-Content go_test_headroom.log
           & "$env:RUNNER_TEMP/headroom.exe" go_test_headroom.log 416
           $hr_rc = $LASTEXITCODE
           if ($go_rc -ne 0) { exit $go_rc }
           exit $hr_rc
   ```

   The prepends go immediately before `go test` — after the poison checks — so exactly the two
   native calls whose failure must not throw are guarded (V18 semantics; UNVERIFIED-LOCALLY, P7).

5. **Collision check:** `grep -n 'go_test_headroom.log\|gated_integration.log' .github/workflows/ci.yml`
   must show the two log names on disjoint lines (gated-integration steps at P4 are untouched;
   they keep `gated_integration.log` only).
6. **YAML syntax gate (local, real):** `go test -count=1 ./internal/cihygiene/` — the package
   yaml.v3-parses EVERY workflow file with anti-vacuity asserts (P11), so a malformed ci.yml is
   red here even though no cihygiene assertion targets the new steps. Expect `ok`, and both M1
   gate assertions still green.
7. Run the M3 acceptance below, then the mutation checks.

### Acceptance commands and expected output

**Wiring present, both legs:**

```bash
grep -n 'go_test_headroom.log\|RUNNER_TEMP/headroom\|tools/ci/headroom' .github/workflows/ci.yml
```
Expected: **≥8 hits** — Linux build step (1 line), Windows build step (1), Linux test block
(log redirect, `cat`, headroom call: 3), Windows test block (`*>` log, `Get-Content`,
`headroom.exe` call: 3). In BOTH jobs `test` and `test-windows`.

**Gate unaffected:** `go test -count=1 ./internal/cihygiene/` → `ok`.

**Clean-checkout build + invoke with a failing-suite fixture (local harness):**

```bash
export RUNNER_TEMP="${RUNNER_TEMP:-/tmp}"
rm -f "$RUNNER_TEMP/headroom"                                     # clean-checkout simulation
go build -o "$RUNNER_TEMP/headroom" ./tools/ci/headroom

# recorded fixture: 59 ok packages + a panic + a FAIL line (60 records ≥ floor 50)
{ for i in $(seq 1 59); do printf 'ok\tgithub.com/sunholo-data/ailang/internal/pkg%d\t12.3s\n' "$i"; done
  printf 'panic: test timed out after 6m56s\n'
  printf 'FAIL\tgithub.com/sunholo-data/ailang/internal/coordinator\t416.100s\n'; } > /tmp/fixture_fail.log

# simulate `go test` with a stub that prints the fixture and exits 1
mkdir -p /tmp/fakebin
printf '#!/bin/bash\ncat /tmp/fixture_fail.log\nexit 1\n' > /tmp/fakebin/go
chmod +x /tmp/fakebin/go

# run the EXACT Linux step body (paste from ci.yml, steps 1+3 lines) under bash -e:
cd "$(git rev-parse --show-toplevel)"   # already there; keep cwd repo-root
env PATH="/tmp/fakebin:$PATH" RUNNER_TEMP="$RUNNER_TEMP" bash -e -c '
go_rc=0
hr_rc=0
go test -timeout 416s ./... > go_test_headroom.log 2>&1 || go_rc=$?
cat go_test_headroom.log
"$RUNNER_TEMP/headroom" go_test_headroom.log 416 || hr_rc=$?
if [ $go_rc -ne 0 ]; then exit $go_rc; fi
exit $hr_rc
' > /tmp/step_stdout.log 2>&1
echo "step rc=$?"
grep -q 'panic: test timed out' /tmp/step_stdout.log && echo "PANIC VISIBLE: yes"
grep -q 'HEADROOM: top 5 slowest packages (budget 416s)' /tmp/step_stdout.log && echo "REPORT: yes"
rm -f go_test_headroom.log
```
Expected: **step rc=1** (the go-test code, not headroom's — headroom parsed 60 packages and
exits 0 here), **PANIC VISIBLE: yes**, **REPORT: yes**, and a `::warning::` for
`internal/coordinator at 100% of budget` in `/tmp/step_stdout.log`. (`go_test_headroom.log` is a
CI artifact; delete it after the harness so the worktree stays clean.)

**`set -e`-guard proof (the same run doubles as it):** because the harness ran under `bash -e`
and the panic IS visible, the `|| go_rc=$?` guard prevented the abort-before-`cat` failure mode
(V17). The mutation arm below proves the mechanism is load-bearing.

**Pass-fixture control (exit path i):** regenerate the fixture as 130 `ok ... 12.3s` lines
(no FAIL, nothing ≥75%), stub `go` with `exit 0`, re-run the step body. Expected: report printed,
no `::warning::`/`::error::`, **step rc=0**.

### Mutations killed, and how to prove them

All three are applied to `/tmp`-copies-or-backups and reverted by `cp`; each is demonstrated with
the harness above (fail fixture, `bash -e`, stub `go`):

| # | Mutation (apply with backup+sed/edit) | Proof it is killed | Restore |
|---|---|---|---|
| 1 | **Drop the go-test exit preservation:** delete `if [ $go_rc -ne 0 ]; then exit $go_rc; fi` (and the `|| go_rc=$?`) | re-run harness: failing suite would make the step exit `$hr_rc`=0 — **green despite a failed suite** ("exit codes through pipes lie" reintroduced). Expect step rc=0 with the mutation | `cp` restore; re-run; expect rc=1 with panic visible |
| 2 | **Restore the `set -e`-unsafe form:** `go test ... 2>&1; go_rc=$?` (remove `\|\| go_rc=$?`) | re-run harness under `bash -e`: the script aborts at the failing `go test` — **rc non-zero but the panic is NOT in stdout** (red job with no test logs). Assert: `grep -c 'panic' /tmp/step_stdout.log` = 0 | `cp` restore; re-run; panic visible again |
| 3 | **Remove the build step:** `rm -f "$RUNNER_TEMP/headroom"` before the harness (binary absent) | headroom call fails `No such file or directory`, captured as `hr_rc=127`; with a PASS fixture the step still **exits 1** — job reds regardless of test results (objection 4's defect, demonstrated loud) | rebuild the binary; re-run; rc=0 |

### Windows acceptance — verified by the sprint PR's own `test-windows` job (NOT locally)

pwsh is not installed on this rig (P7: `command -v pwsh` → exit 1; control `/bin/bash`), so the
pwsh step body CANNOT be executed locally. The M3 Windows acceptance is the sprint PR's own CI.
In the PR's `test-windows` job log, look for exactly:

1. A step named **Build headroom instrument**, green, ordered BEFORE **Run Go test suite**.
2. In **Run Go test suite**'s log: the real per-package lines — `ok <pkg> <N>s` /
   `FAIL <pkg> <N>s` (and, if the suite fails, the `panic: test timed out ...` text) — proving
   the log was not swallowed.
3. Then a `HEADROOM: top 5 slowest packages (budget 416s)` block with ranked rows and
   `(...% of budget)` percentages.
4. Zero or more `::warning::headroom:` lines are acceptable; a `::error:: headroom:` line is NOT
   (it would mean the parser broke against live Windows output).
5. Job verdict == the test suite's verdict: green when `go test` passed, red when it failed.
   If `go test` fails, the log MUST still show the panic line — per the design's residual-risk
   note, a wrong pwsh guard fails LOUD (red job, missing log), which is the acceptable failure
   mode and the signal that the guard needs fixing.

### NOT in this milestone

- No changes to `tools/ci/headroom` code (M2's), the budget values or derivation comments (M1's).
- No changes to the gated-integration steps (P4) or their log name; no job-level
  `timeout-minutes` (45/25 both exceed the ~7 min budget); no `-p` parallelism changes.
- No `internal/coordinator` production changes, no `t.Parallel()` additions (follow-up row per
  the design).

---

## Green gates per milestone

| Milestone | Gate (run in this worktree before handoff) |
|---|---|
| M1 | `go build ./... && go test -count=1 ./internal/cihygiene/` (omit `./tools/ci/...` — no Go packages there until M2; `go test` exits rc=1 "no packages to test", measured at P8) |
| M2 | `go build ./... && go test -count=1 ./internal/cihygiene/ ./tools/ci/...` |
| M3 | `go build ./... && go test -count=1 ./internal/cihygiene/ ./tools/ci/...` + the M3 harness cases green |

## Planner notes

- **Sizing:** the doc's 0.5/0.5/1.0 day split is credible. M3 is correctly the largest: three
  local harness acceptance cases plus three demonstrated mutations plus YAML edits, with the
  Windows leg's verification intentionally deferred to PR CI. No milestone is mis-sized; the plan
  adds no milestones.
- **M1 gate command deviation, justified:** the mission constraint names
  `go test ./internal/cihygiene/ ./tools/ci/...` as the green gate, but `./tools/ci/...` matches
  no Go packages until M2 exists (`go: warning: matched no packages`; `no packages to test`,
  rc=1 — P8, verified in this worktree). M1's gate therefore omits it; M2/M3 restore it.
- **cihygiene cache trap:** always `-count=1` for that package in every acceptance/gate command —
  its tests read files outside the package dir, which the Go test cache does not track.
- **Em-dash literals:** the `::error::` messages in the contract contain `—` exactly as the
  design writes them; the executor should copy them verbatim so the M3 PR-log greps match.
- **Sprint JSON shape:** five `sprint_*.json` files exist in the repo. The plan's JSON matches
  the shape of the most structured precedent, `sprint_m-dx27-docs-search-github-fallback.json`
  (`sprint_id`, `title`, `design_doc`, `sprint_plan`, `mission`, `status`, `target_version`,
  `estimated_days/hours`, `scope{…}`, `milestones[{id,title,files,estimated_loc,estimated_hours,
  dependencies,verification_refs,acceptance_criteria,passes}]`, `final_gates`,
  `verification_log`), extended with: per-milestone `acceptance_commands` and `mutation_killed`
  (required by this sprint's brief), and an `executor_contract` block in the style of
  `sprint_m-motoko-group-kill-and-lsof-containment.json`. `verification_refs` point at design-doc
  rows as `design:V<n>` and at this plan's verification table as `P<n>`.
