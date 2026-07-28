# Sprint Plan: M-Z3-HARD-TIMEOUT

**Status**: ✅ **IMPLEMENTED 2026-07-28** (V1 mission iteration 116) — PR
[#514](https://github.com/sunholo-data/ailang/pull/514) → squash `9253ec8a8`, dev CI green (all 14
checks, including `test-windows` and `Build windows-latest`, which matter because this sprint added
a Windows-tagged file). M1 + M2 complete as planned; M3's follow-up issue filed by the controller as
[#513](https://github.com/sunholo-data/ailang/issues/513) — the executor drafted the text but had no
network egress from its sandbox. Evaluator (sonnet; generator≠judge vs the `codex:gpt-5.6-sol`
executor): **PASS 90/100 round 1, zero blocking findings**.

**Issue**: [#510 — Z3 solver invocation has no hard wall-clock bound](https://github.com/sunholo-data/ailang/issues/510)
**Design doc**: **none, deliberately** — see §0.
**Target**: v0.31.0
**Milestones**: 3 (M1 P0 · M2 P0 · M3 P1)
**Estimate**: **~5h (0.6 working day)** — see §4 for why this is slightly above the 0.5d brief.
**Risk**: Low — one file changed, no caller-visible shape change, no new dependency.
**Branch**: `dev`. `internal/smt` is in neither `CORE_PKGS` nor `DASHBOARD_PKGS` in
`scripts/check_boundaries.sh`, so no boundary rule fires — but `make check-boundaries` still runs in CI.

---

## 0. Why there is no design doc

Per the mission precedent set at iteration 112 (#497 / `m-docs-gate-not-required`): the issue itself
carries the problem statement, the evidence, a calibrated impact bound, and a suggested fix plus test
strategy. There is exactly one defensible implementation and the acceptance criteria are mechanical.
A design doc here would restate the issue and delay a bounded-wait fix that a sibling scheduled loop
is currently exposed to. **Planned directly from #510.**

---

## 1. Premise re-verification (first-party, this session, at `53d3ac727`)

The mission has been burned by laundering another author's claim into fact (iterations 25, 105, 111).
Every load-bearing claim in #510 was re-checked here. **All confirmed. Nothing refuted.** One material
addition.

### CONFIRMED — the bug

- `internal/smt/solver.go:147-148` is `exec.Command` + `CombinedOutput()`. No `exec.CommandContext`,
  no `context`, no `Process.Kill`, no process group anywhere in the file (the file does not even
  import `context` or `syscall`).
- The only bound is Z3's cooperative `-T:<secs>`, built at `solver.go:137-143`, floored to 5 when
  `int(config.Timeout.Seconds()) < 1`.
- `smt.Solve` has exactly two call sites — `cmd/ailang/verify.go:427` and `cmd/ailang/ai_check.go:370`
  — both inside a per-function loop, neither wrapping the call in any timeout of its own.
- `--timeout` is documented **per-function** at `cmd/ailang/verify.go:25` / `:42` and
  `cmd/ailang/ai_check.go:48` / `:65`, default `5s`. A hard *per-call* wall-clock bound therefore
  matches the documented semantics exactly.

### CONFIRMED — the calibration (re-measured, not inherited)

The issue's "`-T:` works in the normal case" datapoint was **reproduced independently** this session:
a generated 100-pigeon/99-hole unsat instance (500,052 SMT-LIB lines) under `z3 -smt2 -T:1` printed
`timeout` and exited after **1.085s** (issue reported 1.159s — same conclusion, different machine
load). Tracing that through `Solve`: `outputStr == "timeout"` matches no `unsat`/`sat`/`unknown`
prefix, falls to the `strings.Contains(outputStr, "timeout")` branch at `solver.go:173`, and returns
`StatusUnknown` + `Error: "solver timeout"`.

**So this is a backstop gap, not an everyday hang.** The plan must not overstate it, and the fix must
not perturb the cooperative path that already works.

### CONFIRMED — the impact claims the issue flagged as its own

Both of #510's "unverified by the reader" claims check out:

- **Eval harness is already protected.** `internal/eval_harness/verify.go:47` `RunAICheck` calls
  `SetProcessGroup(cmd)`, then `waitWithGuards(cmd, timeout*3+10*time.Second, maxRSS)`
  (`verify.go:71-72`). `waitWithGuards` (`internal/eval_harness/memlimit.go:104`) kills the whole
  process group on timeout **and drains `Wait` after the kill**. Banked eval runs genuinely cannot
  hang on this.
- **Agent/interactive use is not.** `cmd/ailang/prompts/agent/convergence-workflow.md:72-73` reads
  "**Your convergence signal is `ailang ai-check FILE`**", and `internal/eval_harness/agent_prompt.go`
  instructs `ailang ai-check` at lines 463, 543, 560, 593, 612 (including ":6. **Only finish once BOTH
  ai-check passes AND output matches!**"). A wedged solver stalls that loop with no diagnostic.

### MATERIAL ADDITION — a second unbounded site the issue does not mention

`grep -rn "exec\.Command" internal/smt/` returns **exactly two** lines:

```
internal/smt/solver.go:147:  cmd := exec.Command(z3Path, args...)
internal/smt/solver.go:271:  out, err := exec.Command(z3Path, "--version").Output()   // Z3Version()
```

`Z3Version()` is the same bug class — an unbounded wait on the same external binary — and it is
**not** on a cold path: `cmd/ailang/verify_print.go:23` calls it while printing the header of every
human-mode `ailang verify` run. A z3 that wedges on `--version` hangs `verify` before a single
function is encoded.

Per CLAUDE.md Principle 3 (SYSTEMIC FIXES — AUDIT BEFORE PATCHING), fixing only the site the issue
named would leave a known instance of the same defect in the same file. **`Z3Version()` is IN SCOPE**
(M2). See §3 for the scope rationale in full.

---

## 2. Technical approach

### 2.1 The bound

```go
// internal/smt/solver.go
const solverKillGrace = 2 * time.Second   // headroom for Z3's own cooperative -T: to land first

// effective bound: never shorter than either what Z3 was told OR what the caller configured
hard := max(config.Timeout, time.Duration(timeoutSecs)*time.Second) + solverKillGrace
ctx, cancel := context.WithTimeout(context.Background(), hard)
defer cancel()

cmd := exec.CommandContext(ctx, z3Path, args...)
setProcessGroup(cmd)                                     // build-tagged, see 2.2
cmd.Cancel = func() error { return killProcessGroup(cmd.Process.Pid) }
cmd.WaitDelay = solverKillGrace                          // see 2.3
output, err := cmd.CombinedOutput()
```

**Why `max(...)` and not just `config.Timeout + grace`.** `timeoutSecs` is
`int(config.Timeout.Seconds())`, floored to 5 when `< 1`. So `--timeout 500ms` tells Z3 `-T:5` and
today legitimately runs for ~5s. A bound of `500ms + grace` would hard-kill a *cooperating* solver
and change behaviour for sub-second configs. `max` makes the hard bound provably ≥ both what Z3 was
told and what the caller asked for, so **no run that completes today starts failing.** (The
truncation itself — `--timeout 900ms` silently becoming `-T:5` — is a separate latent oddity; noted
in §5, deliberately not touched here.)

### 2.2 Kill the group, not just the leader

`cmd.Cancel` defaults to `Process.Kill()`, which kills only the leader. If z3 is a wrapper script
(one of the failure modes #510 names), the real solver is a child and survives. So the command runs
with `Setpgid: true` and `Cancel` sends `SIGKILL` to `-pid`.

`internal/smt` must implement these itself in two new build-tagged files rather than importing
`eval_harness.SetProcessGroup`/`KillProcessGroup`:

- `internal/smt` is imported only by `cmd/ailang`. Importing `internal/eval_harness` from it would
  drag the eval harness into the compiler CLI's verification path — a layering inversion (harness
  depending on core is fine; core-adjacent depending on harness is not).
- The duplicated surface is ~35 LOC of `syscall.Kill`/`SysProcAttr`.

**Rejected alternative:** extract a shared `internal/procgroup` and refactor `eval_harness` onto it.
Correct DRY, but it edits the guard that is *currently the only thing protecting banked eval runs*,
for zero functional gain, and blows the budget. Principle 3 asks us not to miss other instances of the
*bug* (we don't — M2 covers the second site); it is not a mandate to refactor a working guard in
another layer. Recorded as a note, not a milestone.

`killProcessGroup` must return `nil` on `ESRCH` (group already gone) — otherwise a solver that exits
in the same instant the deadline fires surfaces a spurious Cancel error.

### 2.3 Reap, and don't wedge on inherited pipes

`CombinedOutput` gives `Stdout`/`Stderr` non-`*os.File` writers, so `os/exec` creates pipes and copy
goroutines, and `Wait` blocks until those goroutines finish. **A grandchild that inherited the pipe
write end keeps `Wait` blocked even after the leader is killed** — the "killed parent leaving
children" hat the brief calls out. `cmd.WaitDelay` (Go 1.20+; repo is on go 1.26.5) bounds exactly
this: after the context is done, `Wait` waits at most `WaitDelay` for I/O, then closes the pipes and
returns. `CombinedOutput` → `Run` → `Wait` also performs the reap, so no zombie is left.

Group-kill and `WaitDelay` are belt *and* braces on purpose: group-kill is the primary mechanism,
`WaitDelay` is what makes the bound *unconditional* (it also covers Windows, where
`KillProcessGroup` can only reach the leader).

### 2.4 Preserve the caller-visible shape

Immediately after `result.Duration`/`result.RawOutput` are set, and **before** the
`unsat`/`sat`/`unknown` prefix checks:

```go
if errors.Is(ctx.Err(), context.DeadlineExceeded) {
    result.Status = StatusUnknown
    result.Error  = "solver timeout"
    return result, nil
}
```

Identical to what `solver.go:173-177` already produces for the cooperative path, so no caller sees a
new shape. **The check goes first, before the prefix checks, deliberately:** output from a killed
process is untrustworthy, and `StatusUnknown` is the sound direction — we must never report
`StatusVerified` from a process we killed. This also pre-empts `solver.go:180`, which would otherwise
turn the `signal: killed` exit into a `StatusError` (a new shape, and a wrong one).

No changes at `cmd/ailang/verify.go:427` or `cmd/ailang/ai_check.go:370`. No `--help` text changes.

### 2.5 No silent fallbacks (CLAUDE.md Principle 2)

`Solve` is a verification surface: the expiry path returns an explicit, structured
`StatusUnknown`/`"solver timeout"` — never a bare "verified", never a swallowed error, never a
degraded retry. `Z3Version()` (M2) is a *display* path whose existing contract is already
"empty string on any failure"; keeping `""` on timeout is a UI default, which Principle 2 explicitly
permits. That asymmetry is intentional and is called out in M2's criteria so a reviewer does not read
it as an inconsistency.

### 2.6 The regression test, and how it is proven non-vacuous

Shape (as #510 suggests): a fake solver written to `t.TempDir()` that ignores `-T:` entirely.

```sh
#!/bin/sh
sleep 30 &            # grandchild: inherits the stdout/stderr pipe
echo $! > "$CHILDPID"
sleep 30              # leader
```

Injected via `SolverConfig{Z3Path: fake, Timeout: 1 * time.Second}` — `Z3Path` is already an
honoured config field (`solver.go:108`), so no production code exists only for the test.

With `Timeout: 1s` → `timeoutSecs == 1` → `hard = max(1s, 1s) + 2s = 3s`. The fake sleeps 30s.

Assertions:
1. **Bounded completion** — `Solve` returns in well under the 30s the fake would take
   (assert `elapsed < 10s`; expected ~3s).
2. **Shape preserved** — `Status == StatusUnknown` and `Error == "solver timeout"` exactly.
3. **Child cleanup** — the pid in `$CHILDPID` is gone: poll `syscall.Kill(pid, 0)` for `ESRCH`,
   bounded (≤2s, 50ms interval). Not an unbounded `for` — Standing rule 6 applies to our own tests too.

**Why this cannot pass without the fix:** pre-fix, `CombinedOutput` blocks until the 30s leader exits,
so assertion 1 fails by ~20s. Even a leader-only kill leaves the grandchild holding the pipe, so
`Wait` still blocks — assertions 1 and 3 both fail. There is no configuration of the old code under
which this test is green.

**Proof procedure (mandatory, M1 acceptance):**
1. Write and run the test **before** touching `solver.go`. Capture the verbatim `--- FAIL` output.
2. Paste that transcript into the M1 commit body and into `notes` in the sprint JSON.
3. Implement, re-run, confirm PASS and that `elapsed` is ~3s.

**Do NOT** prove this with `git stash` / `git checkout` of `solver.go` — CLAUDE.md Principle 0
forbids those, and the red-first transcript is stronger evidence anyway.

**No skips.** The test needs no z3, so it runs everywhere including CI. It is `//go:build !windows`
(the process-group assertion is POSIX). It must contain **zero** `t.Skip` calls — grep-checkable, and
an explicit acceptance criterion, because this repo has twice been burned by a silently-skipped z3
test (see the comment at `.github/workflows/ci.yml:55-59` and the no-silent-skip gate at `:82-91`).
No entry needs adding to that gate — the gate exists for *binary-gated* tests, and this one is not.
Existing `Z3Available()` skips elsewhere in `solver_test.go` are untouched.

Fake-solver sleeps are 30s but the test finishes in ~3s; even the red run is ~30s, comfortably inside
CI's `go test -timeout 300s` per-package budget.

---

## 3. Scope decisions

| Question | Decision | Rationale |
|---|---|---|
| `Z3Version()` (`solver.go:271`) | **IN SCOPE** (M2) | Same defect class, same file, same binary, and it is on the header path of every human-mode `verify` (`verify_print.go:23`). It is the *only* other `exec.Command` in `internal/smt`. Cost ≈ 20 LOC because M1 already builds the helper. Leaving it would mean the audit found a second instance and knowingly shipped it — the exact anti-pattern Principle 3 names. |
| Total-run budget (bound N functions, not just each call) | **OUT OF SCOPE — file a follow-up** | Correctly identified as a real residual: bounding each call does not bound a run of N functions. But `--timeout` is *documented* as per-function in two commands; a total budget changes documented CLI semantics and needs decisions this sprint has no mandate for (new flag or overload the existing one? what happens to functions already verified when the budget expires — partial results, and in what JSON shape? does `ai-check`'s consumer contract change?). Doing it quietly here would be exactly the "silent expansion" the brief warns against. **M3 files the issue**, cross-linked to #510. |
| Extract shared `internal/procgroup` | **OUT — note only** | §2.2. Touching the live eval guard for zero functional gain. |
| `int(config.Timeout.Seconds())` truncation (`--timeout 900ms` → `-T:5`) | **OUT — noted in §5** | Pre-existing, orthogonal, and changing it *would* alter behaviour. The `max(...)` bound means it causes no regression here. |

---

## 4. Milestones

### M1 — Hard wall-clock bound + process-group kill on `Solve` (~3h, ~90 impl + ~130 test LOC) **P0**

**Tasks**
1. `internal/smt/process_unix.go` (`//go:build !windows`, ~25 LOC): unexported `setProcessGroup(*exec.Cmd)`
   (`SysProcAttr{Setpgid: true}`) and `killProcessGroup(pid int) error` (`syscall.Kill(-pid, SIGKILL)`,
   `nil` on `ESRCH`).
2. `internal/smt/process_windows.go` (`//go:build windows`, ~20 LOC): `setProcessGroup` no-op,
   `killProcessGroup` → `os.FindProcess(pid).Kill()`, with the same comment
   `internal/eval_harness/process_windows.go` carries about job objects. `WaitDelay` is what actually
   bounds Windows.
3. `internal/smt/solver.go` (~45 LOC): `solverKillGrace` const; `hard` computed per §2.1;
   `exec.CommandContext` + `setProcessGroup` + `cmd.Cancel` + `cmd.WaitDelay`; the
   `context.DeadlineExceeded` branch placed **before** the prefix checks; doc comment on `Solve`
   stating the bound and that expiry yields `StatusUnknown`/`"solver timeout"`.
4. `internal/smt/solver_timeout_test.go` (`//go:build !windows`, ~130 LOC): fake-solver helper +
   the three assertions of §2.6. **Written and run red first.**

**Acceptance criteria**
- `TestSolve_HardTimeout_FakeSolverIgnoringT` returns in <10s against a fake that sleeps 30s, with
  `Status == StatusUnknown` and `Error == "solver timeout"` (exact string).
- The fake's background grandchild pid is confirmed dead within 2s of `Solve` returning, via a
  **bounded** `syscall.Kill(pid, 0)` → `ESRCH` poll.
- The red-first transcript (`--- FAIL`, elapsed ≈30s, pre-fix) is recorded verbatim in the commit body
  and the sprint JSON `notes`. Non-negotiable — this is the non-vacuity proof.
- `grep -c "t.Skip" internal/smt/solver_timeout_test.go` == 0.
- The new tests need **no** z3 binary: they pass with `AILANG_Z3_PATH=/nonexistent` in the environment.
- No behaviour change on the cooperative path: with real z3, the §1 pigeonhole instance at
  `--timeout 1s` still returns `StatusUnknown`/`"solver timeout"` in ~1.1s, i.e. it is Z3's own `-T:`
  that fires, not ours. (Z3-gated check; if z3 is absent the executor must say so out loud, not
  silently pass over it.)
- `go test ./internal/smt/...` green; existing `TestSolve_*_Z3`, `TestSolve_Z3NotFound` unchanged and
  still passing.
- `cmd/ailang/verify.go` and `cmd/ailang/ai_check.go` are **not modified** (`git diff --stat` shows
  only `internal/smt/`).

---

### M2 — Same bound on `Z3Version()` (~0.75h, ~20 impl + ~50 test LOC) **P0**

**Tasks**
1. `internal/smt/solver.go`: add `versionProbeTimeout = 5 * time.Second` const; route
   `Z3Version()` through the same `exec.CommandContext` + `setProcessGroup` + `Cancel` + `WaitDelay`
   path as M1. No signature change, no config plumbing — a fixed const keeps it deterministic and
   dependency-free.
2. Comment stating explicitly that `""`-on-failure is retained and *why* it is not a Principle 2
   violation (display path, §2.5).
3. Test using `t.Setenv("AILANG_Z3_PATH", fakeSleepScript)` — `FindZ3()` checks that env var first
   (`solver.go:71-77`), so no signature change is needed to inject the fake.

**Acceptance criteria**
- `TestZ3Version_HardTimeout` returns `""` in <8s against a fake `--version` that sleeps 30s.
- Its grandchild is confirmed dead (bounded poll, as M1).
- Existing `TestZ3Version` (real z3, `Z3Available()`-gated) still passes unchanged.
- `grep -rn "exec\.Command(" internal/smt/` returns **zero** non-`Context` call sites — the audit
  closes. This is the milestone's real exit condition.
- Zero `t.Skip` in the new test.

---

### M3 — Docs, changelog, and the total-run follow-up (~1h, ~40 LOC) **P1**

**Tasks**
1. `CHANGELOG.md` `[Unreleased]` → `### Fixed`: entry naming both call sites, the bound formula, the
   preserved `StatusUnknown`/`"solver timeout"` shape, and `Fixes #510`.
2. One sentence in the `--timeout` help text of `cmd/ailang/verify.go:42` and
   `cmd/ailang/ai_check.go:65` noting the hard backstop is the per-function timeout **plus a 2s
   grace** — so the documented semantics stay true after the change.
3. `gh issue create` for the total-run budget, cross-linked to #510, carrying the four open design
   questions from §3 verbatim so the follow-up starts from a real scope rather than a slogan.
4. Record the `internal/procgroup` duplication as a note on that issue (not a new issue — it is a
   tidiness item, and filing noise is its own cost).

**Acceptance criteria**
- Changelog entry present under `[Unreleased] → Fixed`, mentions #510.
- Both help strings updated; `ailang verify --help` and `ailang ai-check --help` render correctly.
- Follow-up issue exists, is linked from #510, and states plainly that it is *not* fixed by this sprint.
- `make test` green; `make lint` clean (note: `.golangci.yml` enables only govet/ineffassign/
  staticcheck/unused/misspell — gosec is **not** enabled, so no G204 exemption dance is needed).

---

## 5. Notes, risks, and things deliberately not done

- **Executor sandbox.** The brief's standing caveat applies: if the executor runs under a sandbox,
  anything binding a loopback socket is **UNINFORMATIVE, not a failure**. Nothing in this sprint binds
  a socket; it does `exec` a script from `t.TempDir()` and send `SIGKILL` within its own process group.
  If *that* is blocked, report it as an environment limitation and re-run unsandboxed — do **not**
  weaken an assertion to make it pass.
- **`cmd.Cancel` + already-exited process.** If the solver exits in the same instant the deadline
  fires, `syscall.Kill(-pid, ...)` returns `ESRCH`. `killProcessGroup` must map that to `nil`, or a
  benign race surfaces as a spurious error. Called out in M1 task 1; a likely source of a flaky test
  if missed.
- **Grace period value.** 2s is a judgement call: long enough that a cooperating Z3's own `-T:`
  reliably lands first (measured margin above: Z3 self-terminated in 1.085s against a 1s `-T:`, i.e.
  ~0.085s of overshoot — 2s is >20x that), short enough to be a real bound. Named const, one place to
  change.
- **Sub-second `--timeout` truncation** (§3) is untouched and causes no regression because the bound
  uses `max(...)`. Worth a future look; not worth coupling to a backstop fix.
- **Estimate honesty (§4 totals ~4.75h ≈ 0.6d, vs the 0.5d brief).** ~330 LOC, of which ~180 is test.
  The production change is genuinely small (~110 LOC across three files); the overshoot is entirely the
  process-group test rig, which is also the only thing that makes the fix *provable*. Cutting it to
  hit 0.5d would produce exactly the vacuous pass this plan exists to prevent. If a hard time-box is
  needed, **M3 is the compressible one** — fold its changelog entry into M2's commit and file the
  follow-up issue by hand.
- **What could still break** (per the no-premature-victory rule): the bound is proven against a fake
  solver, not against a genuinely wedged real z3 — that state is not reproducible on demand. The claim
  after this sprint is "`Solve` and `Z3Version` cannot block longer than a computed bound regardless of
  what the z3 binary does", proven by construction plus a fake that ignores `-T:`. It is **not**
  "verification can no longer hang" — a run of N functions is still N bounds, which is the follow-up.

---

## 6. Execution status (iteration 116)

- **M1 — COMPLETE:** hard `Solve` deadline, process-group kill/reap, no-skip fake-solver regression,
  red-first proof, Windows cross-compilation, and real-Z3 cooperative timeout probe all verified.
- **M2 — COMPLETE:** bounded `Z3Version`, grandchild cleanup regression, and systemic audit closed
  with zero plain `exec.Command(` sites under `internal/smt`.
- **M3 — PARTIAL:** changelog and both CLI help surfaces are complete and locally verified. The
  total-run-budget GitHub follow-up could not be created because this sandbox cannot reach
  `api.github.com`; its exact ready-to-file text is recorded in the sprint JSON.
