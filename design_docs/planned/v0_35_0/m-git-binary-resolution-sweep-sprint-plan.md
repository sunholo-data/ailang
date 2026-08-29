# Sprint Plan: M-GIT-BINARY-RESOLUTION-SWEEP

**Design doc**: [m-git-binary-resolution-sweep.md](m-git-binary-resolution-sweep.md) (quorum-refined, 666 lines)
**Target**: v0.35.0
**Planned at**: 2026-08-28, iteration 298
**Base**: `sprint/iter298-git-exec-sweep` @ `110bbb7c0` (origin/dev `999d4c1dd` + the design-doc commit)
**Risk**: M1 medium (new shared package + new CI gate + new AST machinery), M2-M4 low (mechanical)

---

## 0. Scope discipline — READ FIRST

This plan covers all four milestones, but **only M1 is scheduled for execution in this
iteration, and M1 is designed to be independently executable and independently landable as
its own PR.**

M1 is the milestone that matters. It fixes the *contract* (failure semantics, caching
semantics, absolute-path refusal), builds the *instrument* (the AST enumerator and its CI
gate), and converts only the 4 Sonar-flagged sites plus `cmd/ailang/help.go`. Everything in
M1 is either new machinery or a 2-line mechanical conversion of a site whose error handling
already consumes `Run`/`Output`.

M2-M4 are **bulk conversions against a contract M1 froze**. They add no new decision, no new
package, no new gate. They ratchet a baseline file downward, 42 sites then 41 then 6. They
land later, each as its own PR, and each is safe to defer indefinitely because the M1 gate
already prevents the class from *growing* while they wait.

**M1 landing criterion**: the branch is mergeable on its own when every AC in §3.5 holds.
Nothing in M2-M4 is a prerequisite for merging M1, and no AC in M1 references a post-M1
count.

---

## 1. Velocity and sizing

Recent comparable work in this repo (gate-shaped sprints): `M-ARCH-BOUNDARIES`
(`scripts/check_boundaries.sh` + make target + ci.yml step, ~260 LOC, 1.5d),
`check_tmpfile_hygiene` + its 11-arm self-test, `check_protocol_closure` + self-test. All
three landed as script+target+ci-step+self-test bundles. M1 is the same shape plus a new Go
package and a new Go AST tool, so it is the largest of the family.

| Milestone | Impl LOC | Test LOC | Est. |
|---|---|---|---|
| M1 | ~430 | ~430 | 1.5d |
| M2 | ~90 (42 sites x ~2 lines) | 0 new (existing coordinator tests) | 1d |
| M3 | ~85 (41 sites) | 0 new | 1d |
| M4 | ~15 (6 sites) | 0 new | 0.5d |

---

## 2. Measured facts this plan is built on

Every number below was **re-measured on the pristine tree at `110bbb7c0`** during planning,
not copied from the design doc. Where a measurement contradicts the doc it is called out in
§6.

| Fact | Command | Observed |
|---|---|---|
| Bare-name git exec sites, non-test, regex | `grep -rEn 'exec\.Command(Context)?\([^)"]*"git"' cmd internal --include='*.go' \| grep -v '_test\.go' \| wc -l` | **93** across **18** files |
| Same, narrow regex | `... -E 'exec\.Command(Context)?\((ctx, )?"git"' ...` | **93**, `diff` vs wide = empty |
| Same, **go/ast** prototype (built during planning) | walk `*ast.CallExpr`, `exec.Command` arg 0 / `exec.CommandContext` arg 1, `strconv.Unquote == "git"` | **93** (cmd 44 + internal 49), **per-file diff vs regex = EMPTY** |
| `Command` vs `CommandContext` split | grep on the site list | 70 / 23 (= 93) |
| `LookPath("git")` non-test | `grep -rn 'LookPath("git")' cmd internal --include='*.go' \| grep -v '_test\.go'` | 1 hit: `cmd/ailang/help.go:186` |
| `gitBinary(` refs in `help_stale_test.go` | grep | 6 (lines 223, 230, 240, 243, 367, 427); plus `resolveGit` at 387, 406 and `exec.ErrNotFound` at 395, 396 |
| Boundary policed sets | `grep -n 'CORE_PKGS\|DASHBOARD_PKGS\|CORE_SURFACE_PKGS' scripts/check_boundaries.sh` | `CORE_PKGS=(parser types eval core elaborate effects builtins lexer ast pipeline runtime link iface)`, `DASHBOARD_PKGS=(server coordinator observatory messaging)`, `CORE_SURFACE_PKGS=(parser types core elaborate pipeline)` — **`gitexec` is in none** |
| `exec.Cmd.Err` deferred contract | standalone Go probe on go1.26.6: `(&exec.Cmd{Path:"", Args:[...], Err: fmt.Errorf("gitexec: %w: no", sentinel)}).Run()` | `Run` returns the wrapped error, `errors.Is(err, sentinel) == true`; same for `.Output()`. **Behaviourally verified, not doc-verified** |
| Go / module | `head -3 go.mod; go version` | `go 1.26.6`, `go1.26.6 darwin/arm64` |
| Linters enabled | `.golangci.yml` | `govet ineffassign staticcheck unused misspell` — **no gosec**, so no G204 interference. `exclude-dirs` includes `scripts` |
| Formatting scope | `make fmt-check` | `gofmt -l .` — **repo-wide, testdata included**. This is why gate fixtures must be `.txt`, never `.go` (§6.7) |

### 2.1 Per-file baseline this sprint will seed (post-M1)

M1 converts 4 sites: `prompt_freeze_check_git.go` 1→0, `prompt_freeze_core.go` 1→0,
`worktree.go` 16→15, `coordinator_cloud.go` 15→14. `help.go` holds **zero** bare sites (its
git calls are variable-first-arg, `help.go:204,222`), so the help.go migration does not move
the count. Residual = **89 across 16 files**. The executor MUST re-derive this by measurement
at execution time and use the measured value; the table below is the expected result, and a
disagreement is a signal to stop and investigate, not to edit the table.

```
internal/coordinator/worktree.go 15
cmd/ailang/coordinator_cloud.go 14
cmd/ailang/coordinator_browse.go 11
internal/coordinator/merge.go 8
internal/coordinator/approval_processor.go 6
internal/pkg/gitcache.go 5
internal/coordinator/artifact_discovery.go 5
internal/coordinator/daemon_tasks_exec_run.go 4
cmd/ailang/coordinator_inspect.go 4
cmd/ailang/chains_diff.go 4
internal/coordinator/daemon_tasks_worktrees.go 3
cmd/ailang/messages_send.go 3
cmd/ailang/coordinator_utils.go 3
cmd/ailang/coordinator_cloud_github.go 2
internal/eval_harness/gemini_evaluator_bridge.go 1
internal/coordinator/observatory_sync.go 1
```
Sum = 89.

---

## 3. M1 — helper + flagged sites + gate (1.5d, THIS ITERATION)

### 3.1 Task breakdown

| # | Task | Files | LOC |
|---|---|---|---|
| T1 | `internal/gitexec` package: `ErrUnresolvable`, `resolveWith` seam, mutex-guarded **success-only** cache, `Path()`, `Command()`, `CommandContext()` | `internal/gitexec/gitexec.go` (NEW) | ~130 |
| T2 | Refusal-branch + caching tests (B1-B6, §3.3) | `internal/gitexec/gitexec_test.go` (NEW) | ~230 |
| T3 | **go/ast enumerator** (the doc's HID-6 commitment; bash cannot do this) | `tools/check-git-exec/main.go` (NEW) | ~180 |
| T4 | Enumerator unit tests, incl. the ADDITION arms E1-E6 | `tools/check-git-exec/main_test.go` (NEW) | ~200 |
| T5 | Gate driver: fixture control → enumerate → ratchet → LookPath invariant → residual print | `scripts/check_git_exec.sh` (NEW) | ~130 |
| T6 | Gate fixtures | `scripts/testdata/git_exec_gate_positive.txt` (NEW), `scripts/testdata/git_exec_gate_blindspot.txt` (NEW) | ~40 |
| T7 | Baseline, **seeded by fresh measurement after T8** | `scripts/git_exec_baseline.txt` (NEW) | 16 lines |
| T8 | Convert the 4 flagged sites (2-line diffs, conventions preserved verbatim) | `cmd/ailang/prompt_freeze_check_git.go:82`, `cmd/ailang/prompt_freeze_core.go:194`, `internal/coordinator/worktree.go:308`, `cmd/ailang/coordinator_cloud.go:459` | ~10 |
| T9 | help.go migration: delete `resolveGit`, replace `gitBinary()` body with a `gitexec.Path()` adapter mapping error→`""` | `cmd/ailang/help.go` | ~-25/+8 |
| T10 | Rewrite `help_stale_test.go` against the new seam (drop `resolveGit` cases → they move to `gitexec_test.go`; keep the 6 `gitBinary()` call sites working) | `cmd/ailang/help_stale_test.go` | ~-45/+10 |
| T11 | Gate bash self-test (mirrors `scripts/test_check_tmpfile_hygiene.sh` idiom, incl. `ARMS_EXPECTED` floor) | `scripts/test_check_git_exec.sh` (NEW) | ~200 |
| T12 | Make targets + `ci:` aggregate + **two** ci.yml steps (§3.4) | `make/code-health.mk`, `make/ci.mk`, `.github/workflows/ci.yml` | ~14 |
| T13 | Changelog entry | `changelogs/v0.32-current.md` (NOT root `CHANGELOG.md` — see §3.4) | ~12 |
| T14 | Run the mutation protocol (§3.3) and record evidence in the sprint log | — | — |

### 3.2 Design refinements this plan makes to the doc (all justified in §6)

- **R1 — `gitBinary()` is kept as a 3-line adapter, not deleted.** The doc says "delete both".
  `resolveGit` genuinely moves to `gitexec` and is deleted from `help.go`. But the doc's own
  rationale requires *somewhere* in `help.go` that maps `gitexec.Path()`'s error to `""` so
  `gitHead`/`gitDirty` keep their "undeterminable → SHOW the warning" semantics. Keeping
  `gitBinary()` as that one mapping site localises the semantics, keeps 6 test call sites
  compiling, and is the minimal diff. `resolveGit`'s deletion is the falsifiable part and is
  asserted by AC9.
- **R2 — the AST/regex cross-check runs over the REAL TREE ONLY, never over the fixtures.**
  Measured during planning: on a fixture holding a multi-line `exec.Command(\n"git", …)` the
  AST arm returns 1 and the regex arm returns 0. The doc's HID-6 clause 2 ("a disagreement
  between them is a gate FAILURE") and clause 3 (ship a multi-line fixture) contradict each
  other if the cross-check is applied to the fixture. Scope the cross-check to `cmd/` +
  `internal/`, where both arms measure 93 today and 89 after M1.
- **R3 — `internal/gitexec/` is NOT excluded from the exec enumeration.** The doc excludes it.
  It does not need to be: `gitexec` never calls `exec.Command("git", …)` (it calls
  `exec.Command(absPath, …)`), so excluding it only creates a directory where a bare site
  could hide. Only the **`LookPath("git")` invariant** is scoped to `internal/gitexec/`.
- **R4 — an unparseable `.go` file is a gate FAILURE (exit 2), not a skip.** The prototype
  built during planning printed to stderr and continued; that is a silent hole (a syntax-broken
  file hides every site in it). The shipped enumerator must exit 2 naming the file.
- **R5 — the variable-first-arg fixture is a NEGATIVE control with an asserted count of 0**,
  not a positive that moves the count. An AST enumerator keyed on a `"git"` *string literal*
  cannot see `g := "git"; exec.Command(g, …)` — that is the design's own declared residual
  (HID-6 clause 4). Asserting it stays 0 makes the declared residual falsifiable: if someone
  later adds dataflow, the test fails and forces the residual list to be updated. The
  **count-moves-by-addition** requirement is carried by the multi-line fixture (E1), which
  moves the count by exactly +1.
- **R6 — the AST enumerator lives in `tools/`, not `scripts/`.** `.golangci.yml` sets
  `exclude-dirs: [..., scripts]`, so a Go program under `scripts/` would silently escape
  `make lint`. `tools/` already holds nine `package main` Go tools with tests
  (`tools/gen-error-codes/main_test.go` is the precedent).

### 3.3 Test plan and mutation protocol

Every refusal branch M1 adds gets one test AND one **neutering** mutation of the form
`if false && <cond>` — never a deletion, because a deleted block can fail to COMPILE and that
red masquerades as the guard firing. Protocol per mutant, in this order:

1. `cp <file> /tmp/m1-backup/<file>` (restore from this backup — **never** `git checkout --`,
   the tree is uncommitted by construction).
2. Apply the mutation.
3. **Assert the mutant LANDED**: `sha256sum` before ≠ after, and `grep -c 'if false &&' <file>` moved.
4. **Assert the mutant BUILDS**: `go build ./internal/gitexec/... ./cmd/ailang/...` rc=0 (Go tool)
   or `bash -n scripts/check_git_exec.sh` rc=0 (bash gate). **Only then** read a test result.
5. Run the named test; require rc≠0.
6. Restore from backup; re-run; require rc=0.

| # | Branch / property | Test | Neutering mutation the test must kill |
|---|---|---|---|
| B1 | `look()` errors → refuse | `TestResolveWith_LookError` : `resolveWith(fail)` returns err with `errors.Is(_, ErrUnresolvable)` | `if false && err != nil` in the error check |
| B2 | `look()` returns a relative path → refuse | `TestResolveWith_RelativeRefused` : `"git"` and `"./git"` both refused | `if false && !filepath.IsAbs(p)` |
| B3 | failed resolve → `Cmd.Err` carries it | `TestCommand_DeferredError` : build a Cmd under an injected failing resolver, assert the error identity comes back from **`Run()`**, not from construction | `if false && resolveErr != nil` at the `Cmd.Err` assignment |
| B4 | success path → `Cmd.Path` is the absolute path | `TestCommand_UsesAbsolutePath` : `resolveWith(-> "/abs/git")`, assert `cmd.Path == "/abs/git"` and `cmd.Args[0] == "/abs/git"`; `CommandContext` honours a cancelled ctx | swap the assignment to the bare literal `Path: "git"` (builds; B4's absolute assertion kills it) |
| B5 | success IS cached | `TestPath_CachesSuccess` : two `Path()` calls, `look` invoked exactly once | guard the cache store: `if false && err == nil { cached = p }` |
| B6 | failure is NOT cached (HID-3) | `TestPath_DoesNotCacheFailure` : first `Path()` fails, second `Path()` re-invokes `look` and returns a later success | **"cache the failure too"** — store the error in the cache slot on failure; the second call returns the stale error without re-invoking `look` |
| G1 | fixture missing → exit 2 "INSTRUMENT BROKEN" | self-test S2 | `if false && [ ! -f "$POSITIVE_FIXTURE" ]` |
| G2 | fixture yields 0 matches → exit 2 | self-test S2b (fixture present but emptied) | `if false && [ "$FIXTURE_COUNT" -eq 0 ]` |
| G3 | a file above its baseline count → exit 1, names file:line | self-test S3 | `if false && [ "$n" -gt "$base" ]` |
| G4 | a file below its baseline count → exit 1 "tighten the baseline" | self-test S4 | `if false && [ "$n" -lt "$base" ]` |
| G5 | a file with matches absent from the baseline → exit 1 | self-test S3b (new file) | `if false && [ -z "$base" ]` |
| G6 | `LookPath("git")` outside `internal/gitexec/` → exit 1 | self-test S5 | `if false && [ "$LOOKPATH_OUTSIDE" -gt 0 ]` |
| G7 | AST vs regex disagree **on the real tree** → exit 1 | self-test S6 (plant a multi-line git site in a temp tree copy: AST sees it, regex does not) | `if false && [ "$AST_TOTAL" -ne "$RX_TOTAL" ]` |

Enumerator unit tests (`tools/check-git-exec/main_test.go`), all **addition** arms:

| # | Arm | Kills |
|---|---|---|
| E1 | temp tree count = N; copy in the multi-line fixture as `.go`; count = **N+1** | "enumerator is line-oriented" — i.e. a regression to `grep`. This is the ADDITION arm; a removal-only test would still pass under it |
| E2 | blind-spot fixture (`g := "git"` + `bash -c "git status"`) adds **exactly 0** | a silent widening of the enumerator that leaves the declared residual list stale |
| E3 | `exec.Command("git-lfs", …)` and `exec.Command("gitfoo")` add 0 | prefix-matching instead of `strconv.Unquote(lit) == "git"` |
| E4 | `exec.CommandContext(ctx, "git", …)` counted at arg index **1**; a `"git"` in arg 2 is not | an off-by-one in the arg index |
| E5 | fixture named `x_test.go` adds 0; the **same bytes** renamed `x.go` add 1 | a `_test.go` exclusion that is vacuous (matches nothing) rather than suffix-based |
| E6 | a file with a Go syntax error → **exit 2**, naming the file | R4: silent skip of unparseable files |

Bash self-test (`scripts/test_check_git_exec.sh`) carries an `ARMS_EXPECTED` floor exactly as
`scripts/test_check_tmpfile_hygiene.sh:9` does, so a self-test that stops running arms fails
loudly instead of reporting green.

### 3.4 CI wiring — derived from the repo, not from memory

**`make ci` is NOT invoked anywhere in `.github/workflows/ci.yml`.** Verified:
`grep -n 'run: make ' .github/workflows/ci.yml` lists 31 individual targets and `ci` is not
among them. A target wired only into `make ci` **does not run in CI**. Both edits are
therefore required:

1. **`make/code-health.mk`** — add beside `check-boundaries` (currently line 161-162, exactly):
   ```
   check-boundaries: ## Check architecture layer boundaries (CI gate)
   	@bash scripts/check_boundaries.sh
   ```
   Insert after line 162:
   ```
   check-git-exec: ## Refuse bare-name git exec sites outside internal/gitexec (CI gate)
   	@bash scripts/check_git_exec.sh

   test-check-git-exec: ## Run the git-exec gate's own self-test (bash 3.2)
   	@/bin/bash scripts/test_check_git_exec.sh
   	@/bin/bash -n scripts/check_git_exec.sh
   	@go test ./tools/check-git-exec/... -count=1
   ```
   (the `bash -n` + self-test pairing mirrors `test-check-tmpfile-hygiene` at
   `make/code-health.mk:170-172`.)

2. **`make/ci.mk` line 11** — append `check-git-exec test-check-git-exec` to the `ci:`
   prerequisite list. The line currently reads (truncated):
   ```
   ci: deps fmt-check vet lint test test-nightly-classifier test-coverage-badge test-lowering verify-no-shim ... check-tmpfile-hygiene test-check-autoclose test-check-changelog test-check-protocol-closure test-check-tmpfile-hygiene check-prompt-freeze ## Run full CI verification, including workflow gates
   ```

3. **`.github/workflows/ci.yml`** — the boundary step is exactly:
   ```
   132:    - name: Check architecture boundaries
   133:      run: make check-boundaries
   ```
   Insert **after line 133** (before `- name: Verify install guide consistency` at line 135):
   ```yaml
       - name: Check git exec sites
         run: make check-git-exec

       - name: Git exec gate self-test
         run: make test-check-git-exec
   ```
   These land in the `test:` job (`runs-on: ubuntu-latest`, ci.yml:17-18), which has Go set up
   and runs `make install` at line 60, so `go test ./tools/...` is available.

**Changelog**: `make check-changelog` runs `scripts/check_changelog.sh`, whose rule is
structural — the root `CHANGELOG.md` may contain **exactly one heading**, the archive table's
(`## Changelog Archives`); *any* other heading is release-note content and fails the gate.
Verified at base: `make check-changelog` rc=0, printing
`✓ CHANGELOG.md is index-only and links changelogs/v0.32-current.md`, and
`grep -n '^#' CHANGELOG.md` returns exactly two lines (`# AILANG Changelog`,
`## Changelog Archives`). **The M1 entry goes under `## [Unreleased]` in
`changelogs/v0.32-current.md`. Do not touch root `CHANGELOG.md`.**

### 3.5 Acceptance criteria, each with its **measured base rc**

Measured on the pristine worktree at `110bbb7c0` before any edit. Falsifiability column
answers: *what would this still pass under, if the claim were false?*

| # | Acceptance command | Base (measured) | Required after M1 | Falsifiable? |
|---|---|---|---|---|
| AC1 | `go build ./internal/gitexec/... ./cmd/ailang/... ./internal/coordinator/...` | **rc=1** (`pattern ./internal/gitexec/...: lstat: no such file or directory`) | rc=0 | YES — red at base for the right reason (package absent) |
| AC2 | `go test ./internal/gitexec/... -count=1` | **rc=1** (setup failed, no such package) | rc=0 | YES |
| AC3 | AST enumerator total over `cmd internal` = **89** (via `make check-git-exec` output, or the tool directly) | **93** | 89 | YES |
| AC4 | `grep -rln 'LookPath("git")' cmd internal --include='*.go' \| grep -v '_test\.go' \| grep -cv '^internal/gitexec/'` | **1** (`cmd/ailang/help.go`) | **0** | YES |
| AC5 | `make check-git-exec` | **rc=2** (`No rule to make target 'check-git-exec'`) | rc=0 | YES |
| AC6 | `make test-check-git-exec` | **rc=2** (no such target) | rc=0 | YES |
| AC7 | `grep -c 'make check-git-exec' .github/workflows/ci.yml` | **0** (rc=1) | `>= 1` | YES |
| AC8 | `grep -c 'make test-check-git-exec' .github/workflows/ci.yml` | **0** (rc=1) | `>= 1` | YES |
| AC9 | `grep -c 'check-git-exec' make/ci.mk` | **0** (rc=1) | `>= 1` | YES |
| AC10 | `grep -c 'func resolveGit' cmd/ailang/help.go` | **1** | **0** | YES |
| AC11 | `grep -c 'gitexec' changelogs/v0.32-current.md` | **0** (rc=1) | `>= 1` | YES |
| AC12 | `make check-git-exec 2>&1 \| grep -c 'variable first argument'` (residual classes printed every run — HID-6 cl.4) | **rc=2**, no output | `>= 1` | YES |
| AC13 | Gate anti-vacuity: `mv scripts/testdata/git_exec_gate_positive.txt /tmp/ && bash scripts/check_git_exec.sh; rc=$?` | n/a (no fixture, no gate) | **rc=2**, output contains `INSTRUMENT BROKEN` | YES |
| AC14 | Mutation evidence: for each of B1-B6 and G1-G7, the mutant (a) has a different sha256 from the backup, (b) **builds** (`go build …` rc=0 / `bash -n …` rc=0), and (c) the named test returns rc≠0; restoring gives rc=0 | n/a | all 13 pass | YES by construction |
| AC15 | AST arm and regex arm agree on `cmd internal` (both 89) | both **93** (per-file `diff` = empty, verified) | both 89, diff empty | YES (89 ≠ 93) |
| **REGRESSION GATES — rc=0 at base AND after. NOT acceptance criteria; they cannot fail for the right reason.** |
| R-a | `go test ./cmd/ailang/... -count=1` | rc=0 (24.3s) | rc=0 | no |
| R-b | `go test ./internal/coordinator/... ./internal/pkg/... ./internal/eval_harness/... -count=1` | rc=0 | rc=0 | no |
| R-c | `make check-boundaries` | rc=0 | rc=0 | no |
| R-d | `make check-file-sizes` | rc=0 | rc=0 | no |
| R-e | `make check-changelog` | rc=0 | rc=0 | no (AC11 is its falsifiable partner) |
| R-f | `gofmt -l cmd internal tools scripts` | empty, rc=0 | empty | no |
| R-g | `go vet ./cmd/ailang/...` | rc=0 | rc=0 | no |
| **REJECTED — already red on the untouched tree; would measure the repo, not the change** |
| X-1 | `go build ./...` | **rc=1**: `# github.com/sunholo-data/ailang/cmd/wasm` / `runtime.main_main·f: function main is undeclared in the main package` | — | **REJECTED**. Use AC1's narrowed form |

### 3.6 M1 risks

| Risk | Mitigation |
|---|---|
| `go run`/`go test` in the gate slows CI | `tools/check-git-exec` is stdlib-only and tiny; the `test:` job already runs 31 make targets. If it matters, the gate can `go build -o` once into `$TMPDIR` |
| The gate's ratchet blocks unrelated PRs that legitimately add a git call | That is the point (HID-6). The escape hatch is an explicit, reviewed edit to `scripts/git_exec_baseline.txt`, which is exactly the audit trail wanted |
| `help_stale_test.go` rewrite loses coverage | The `resolveGit` cases (lines 387-406) move to `gitexec_test.go` as B1/B2 and are strengthened there with mutants; net coverage goes up, not down |
| Concurrent-session collision | Re-verified at plan time: none of the 4 flagged files nor `help.go` is under active edit; the tree is clean at `110bbb7c0` |

---

## 4. M2-M4 — mechanical conversions (separate iterations)

Each is its own PR, each ratchets `scripts/git_exec_baseline.txt` downward, each adds **no**
new decision. The contract is frozen by M1; the only per-site work is replacing
`exec.Command("git", args…)` with `gitexec.Command(args…)` and
`exec.CommandContext(ctx, "git", args…)` with `gitexec.CommandContext(ctx, args…)`,
preserving each site's `cmd.Dir` vs `-C` convention verbatim.

### M2 — `internal/coordinator` (1d, 42 sites, 7 files) — ✅ completed iteration 300
`worktree.go` 15, `merge.go` 8, `approval_processor.go` 6, `artifact_discovery.go` 5,
`daemon_tasks_exec_run.go` 4, `daemon_tasks_worktrees.go` 3, `observatory_sync.go` 1.
Acceptance: baseline sum = **47** (89 − 42, re-derived); `make check-git-exec` rc=0;
`go test ./internal/coordinator/... -count=1` rc=0 (base rc=0, regression gate).
The pristine M2 base with its matching 89-site baseline is rc=0. The falsifiable negative
control is the converted tree with that 89-site baseline preserved: rc=1 with seven
`tighten the baseline` diagnostics. Ratcheting the baseline to 47 restores rc=0.
Note: 8 sites here and in M3 already discard the error: 6 in M2
(`approval_processor.go` 2, `daemon_tasks_exec_run.go` 2, `worktree.go` 2) and 2 in M3
(`coordinator_browse.go` 2).
Conversion **preserves** that silent swallow; the design explicitly does not fix them.

Execution deviation for PR #958: the mechanical conversion's first Sonar analysis was red
on new-code coverage (31.0%) and duplicated-line density (3.9%). The executor added focused
real-repository tests for merge, worktree inspection/cleanup, and both auto-commit paths;
factored artifact-path accumulation to remove the reported duplicate block; and replaced the
four reported repeated Git tokens with package constants. A local coverprofile mapped 50 of
57 diff-added executable lines as covered (87.7%) before push; Sonar itself must remeasure
after the follow-up commit is pushed.

### M3 — `cmd/ailang` (1d, 41 sites, 7 files)
`coordinator_cloud.go` 14, `coordinator_browse.go` 11, `chains_diff.go` 4,
`coordinator_inspect.go` 4, `messages_send.go` 3, `coordinator_utils.go` 3,
`coordinator_cloud_github.go` 2.
Acceptance: baseline sum = **6**; `go test ./cmd/ailang/... -count=1` rc=0.

### M4 — tools layer (0.5d, 6 sites, 2 files)
`internal/pkg/gitcache.go` 5, `internal/eval_harness/gemini_evaluator_bridge.go` 1.
Acceptance: baseline sum = **0**; the baseline file asserts emptiness; the positive
invariant (exactly one `LookPath("git")`, inside `internal/gitexec/`) is the standing gate
thereafter, kept non-vacuous forever by the fixture control (AC13).

---

## 5. Success metrics

- M1: 93 → 89 bare sites; 1 → 1 `LookPath("git")` but **relocated** into `internal/gitexec/`;
  gate live in CI as two ci.yml steps; 13 mutants killed.
- M4: 0 bare sites; residual classes (variable first argument, dataflow/aliasing, shell
  strings, `syscall.Exec`, non-Go surfaces) printed by the gate on every run and **not**
  claimed as covered.
- Sonar: the 4 flagged new-code `go:S4036` issues stop invoking `exec.Command("git", …)`.
  The rating is **not** asserted to reach A — the resolver's own `LookPath` is expected to
  keep one finding, tracked as `M-GIT-RESOLVER-S4036-CLOSURE`.

---

## 6. Contradictions and staleness found while measuring

Reported because a refutation is the useful output, not agreement.

1. **The doc's HID-6 clauses 2 and 3 are mutually inconsistent as written.** Clause 2 makes
   an AST/regex disagreement a gate FAILURE; clause 3 ships a fixture *designed* to make them
   disagree. Measured: on such a fixture the AST arm returns **1** and the regex arm returns
   **0**. Resolved by R2 — the cross-check is scoped to `cmd/` + `internal/` only.
2. **The doc never lists the Go file that HID-6 requires.** "Files to Modify/Create" has
   `scripts/check_git_exec.sh` (~120 LOC bash) as the enumerator, but bash cannot walk
   `go/ast`. `tools/check-git-exec/main.go` + `main_test.go` (~380 LOC) are missing from the
   doc's file list and from its LOC estimate. Added as T3/T4.
3. **Excluding `internal/gitexec/` from the exec enumeration (doc, gate step 2) is both
   unnecessary and a hole.** `gitexec` calls `exec.Command(absPath, …)`, never
   `exec.Command("git", …)`, so it would never be flagged; the exclusion only creates a
   directory where a bare site can hide. Only the `LookPath` invariant needs that scoping (R3).
4. **The doc's V17 multi-line premise is CONFIRMED, and now confirmed by a second, stronger
   instrument.** A real `go/ast` enumerator built during planning returns **93** with a
   **per-file diff against the regex of exactly zero**. The doc's V17 used a whitespace-collapsing
   python regex; an AST walk is a genuinely independent arm and it agrees. The blind spot is
   prospective, as the doc says.
5. **`exec.Cmd.Err` was doc-verified (`go doc`) but not behaviour-verified.** Verified during
   planning on go1.26.6 that `(&exec.Cmd{Path:"", Err: wrapped}).Run()` and `.Output()` both
   return the wrapped error and satisfy `errors.Is`. The failure contract is sound.
6. **`worktree.go` now holds 16 git sites, not the doc's implied 16-via-"15 more".** Consistent,
   but the executor must re-measure: 9 of the 18 sweep files changed between the doc's original
   base `e38c0c493` and `999d4c1dd`.
7. **A `.go` fixture would break `make fmt-check`.** `fmt-check` runs `gofmt -l .` repo-wide and
   `gofmt` does **not** skip `testdata/`. Fixtures must stay `.txt`; `go/parser.ParseFile`
   accepts source by bytes regardless of extension, so this costs nothing (R6 rationale).
8. **`.golangci.yml` excludes `scripts/`.** A Go enumerator placed under `scripts/` would escape
   `make lint` silently. Hence `tools/` (R6).
9. **The doc's `gitBinary` deletion is under-specified** — deleting it removes the only place
   that maps a resolution error to the `""` that `gitHead`/`gitDirty` treat as "undeterminable →
   show the warning". Kept as a 3-line adapter (R1).

### Acceptance criteria that could NOT be made falsifiable

- **R-a … R-g** (§3.5): the seven regression gates. All are rc=0 at base and required rc=0
  after, so none can fail *for the right reason*; they are listed as regression gates, not as
  acceptance criteria. `make check-changelog` (R-e) is the one with a falsifiable partner
  (AC11, `grep -c 'gitexec' changelogs/v0.32-current.md`: 0 → ≥1).
- **The Sonar success metric.** SonarCloud only re-analyses on a merged push and its leak
  period is defined against `previous_version = v0.33.2`, so "the 4 flagged issues are
  resolved" cannot be asserted from the branch at all. It is deliberately excluded from §3.5
  and recorded as a post-merge observation.
- **`go build ./...`** (X-1): rejected outright, already rc=1 at base.
