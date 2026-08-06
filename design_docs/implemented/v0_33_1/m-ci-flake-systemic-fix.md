# M-CI-FLAKE-SYSTEMIC-FIX: One Gating Convention, Bounded Waits, a Default-Deny Egress Boundary, and a Known-Offender Lint for the Go Test Suite

**Status**: **ALL MILESTONES LANDED** — SPRINT-PLANNED 2026-08-04 (iteration 142); M1 (`c440a1628`), M2 (`368f940cf`), M3 (`13c570063`), M4 (`4b47f8b0a`), **M5 (2026-08-06, iteration 148)**; `D5` DECIDED 2026-08-05 = Option A (Mark). **Full AC1–AC12 sweep re-run out-of-sandbox at M5 — all pass**; the one full-suite red observed was `#598` (an unrelated pid-file race, poison ruled out by a 3-arm negative control, and green on re-run: rc=0 / 107 ok). **MOVED to `design_docs/implemented/v0_33_1/` 2026-08-06 (iteration 149)** after the M5 merge `c9e1a4f98` went Gate-3b GREEN on `dev` (SHA-addressed). Sprint plan travelled with it. Issues closed: #583 (auto, via the PR body) · #494 · #509 · #587 · #561 — each verified in code first, not closed on this line's `Closes:` claim.
**Target**: v0.33.1
**Priority**: P0 (High) — flakes red-light `dev` CI, and a red `dev` outranks the mission queue every time it occurs
**Estimated**: ~~3–4 days (4 milestones)~~ **26h ≈ 4.5 days, 5 milestones** (planner revision, iteration 142 — the doc's M3 bundled a 460-LOC pure-Go linter with the workflow edits; splitting them isolates the only CI-touching commit)
**Sprint plan**: [`m-ci-flake-systemic-fix-sprint-plan.md`](./m-ci-flake-systemic-fix-sprint-plan.md)
**Dependencies**: None. ⚠ **Collision — CONTROLLER DECISION TAKEN 2026-08-05 (iteration 145), plan §6.1 option (b), not the planner's recommended (a):** PR **#532** rewrites `buildAilang` in `cmd/ailang/main_test.go` and the body under the `testing.Short()` gate in `serve_api_mcp_surface_test.go`, and touches `ci.yml` — i.e. exactly M2's surface. **M2 proceeds first; #532 rebases onto it afterwards.** Reasons, measured not assumed: (1) #532 is authored by `sunholo-voight-kampff` — *this loop's own PR*, so there is no external author to coordinate with and no review latency; (2) it has been `CONFLICTING`/`DIRTY` against `dev` since **2026-07-29** and untouched for 7 days, so **M2 does not make it any more conflicted than it already is** — the "resolve #532 first" cost is a pre-existing debt, not one M2 creates; (3) resolving a week-old conflict is a separate piece of work that would consume the iteration that Mark's `D5` answer exists to unblock. The re-application cost is symmetric either way (#532 replaces `buildAilang` wholesale; M2 wraps it — whoever goes second re-applies), so ordering was chosen on *unblocking value*, not on merge cost. **Follow-up owed:** comment on #532 recording this, so its rebase re-applies `HangGuardContext` to the new `sync.Once` body. PR **#569** (dependabot actions bump) touches `ci.yml` + `build.yml` — M4's surface, re-check before M4.
**BOTH COLLISIONS ARE NOW RESOLVED — this Dependencies block is retained for provenance only:**
**#532 was CLOSED as SUPERSEDED** (iteration 145, verified *by purpose* rather than by state: its
entire reason — one shared binary build instead of fourteen, to escape the Windows timeout — had
already landed independently as `#564`/`3c28cc322`, so it sat `OPEN`/`CONFLICTING` *because* it was
dead, not because it was blocking). **#569 was MERGED first** (`bc30912ea`, iteration 147) to clear
the `ci.yml`/`build.yml` collision before M4 touched those files. No open collision remains.
**Planner-Lane**: opus-required
**Authorized**: Mark Edmondson, 2026-08-04 — "Yes sprint a CI flake fix"
**Closes**: #583, #494, #509, #587 (CI flakes) · #561 (local-only network dependency — see Scope note)

> ⚠ **CONTROLLER CORRECTIONS, 2026-08-04 (iteration 142) — read before executing.** Four claims
> in this document were measured FALSE at HEAD `9feefa3a6` and are corrected in place, struck
> rather than deleted so the reasoning errors stay visible: **V22 is REFUTED** (superseded by
> **V33**), **AC3 is VACUOUS as written and M2 must not be executed against it**, **V30's count
> was 2 and is 1**, and **the leg count was 5 and is 6** (V34). The load-bearing one: the
> poisoned proxy does **not** cover AILANG's own `Net` effect, because `internal/effects` builds
> its transports by hand with `Proxy == nil`. Whether to close that hole is a production-runtime
> design question, ~~**PARKED for human decision**~~ **DECIDED 2026-08-05 — see below**
> (Deferred Decisions D5; escalated on `#559`).
>
> ✅ **`D5` IS DECIDED, 2026-08-05 (iteration 145) — OPTION A, by Mark.** Exact words:
> *"D5 - option A and then queue the B afterwards. I'm cool with 2."* So: the `Net` effect stays
> **OUTSIDE** the egress boundary for this sprint. AC3 is replaced by **AC3′(a/b/c)** below, the
> `internal/effects` egress tests move behind `RequiresLiveNetwork`, and the residual is asserted
> *as open* by the new **AC10(d)**. **Option B** (`Proxy: http.ProxyFromEnvironment` on the 6
> hand-built transports) is queued as its **own separate design item** with its own design pass and
> quorum — it must not ride in on a sprint scoped and reviewed as test-only. **M2, M3 and M4 are
> UNBLOCKED.**
>
> ✅ **The iteration-141 narrow-refinement carve-out is RATIFIED, 2026-08-05 (iteration 145).**
> The R3 quorum fix was applied under the Gate-2 narrow-refinement carve-out with no re-quorum on
> that fix; Mark accepted it as-is ("Carve-out disclosure … ACCEPTED as-is. No re-quorum needed").
> **The veto window is closed** — this doc no longer carries that caveat.

> **Version-dir justification**: current release is v0.33.0 (`std/VERSION`, release memory
> 2026-08-03). This is pure test-infrastructure hardening — no language surface, no runtime
> behavior — so the next patch dir `v0_33_1` is correct.

---

## Problem Statement

Five open issues keep turning `dev` CI red on commits that cannot have caused the failure
(three of the five triggering commits were **markdown-only**). Each red costs a triage slot,
and — worse — trains observers to re-run reds instead of reading them, which is exactly how the
mission lost 2 hours today to a **deterministic regression misfiled as a known flake**.

The five issues are symptoms of four structural classes, not five independent bugs:

| Class | Issue(s) | Mechanism |
|---|---|---|
| **C1. Third-party verdict** | #583 (CI), #561 (local) | A test's pass/fail depends on someone else's uptime (live `git clone` of GitHub, live `httpbin.org`) |
| **C2. Absolute-time assertion** | #509, #587 | A wall-clock bound calibrated on a warm machine fires on a cold runner; every such bound in the repo has already been **raised at least once**, chasing the runner instead of fixing the class |
| **C3. Unbounded subprocess** | #494 | A test helper with no deadline turns one hung child into a whole-package panic-red |
| **C4. The inert gate** | root cause of #583; latent under 7 files | `testing.Short()` guards **do nothing in CI** because `-short` is never passed anywhere — a vacuous gate that makes authors *believe* they have protected CI |

**The load-bearing finding (C4):** `-short` appears **nowhere** in `.github/workflows/`,
`make/`, or `Makefile` (Verification Log **V1** — the single grep hit is
`git rev-parse --short HEAD`). Seven first-party test files gate on `testing.Short()` (**V2**);
every one of those gates is inert in CI. Six of the seven gate self-contained tests that have
therefore been *running in CI all along, green* — proof the gates are unnecessary. The seventh,
`internal/pkg/gitcache_test.go`, gates a **live network clone**, so its inert gate is issue #583.

The repo currently has **three gating idioms and no shared helper** (V2, V5, V6):

1. `testing.Short()` — 7 files — **silently inert in CI** (the dangerous one)
2. env opt-**out** (`os.Getenv("CI")`/`GITHUB_ACTIONS`) — 2 files — works in CI, but leaves
   local `make test` network-dependent (#561), and its semantics are re-derived per file
3. env opt-**in** (`*_live_test.go`, skip unless var set) — 6 files — the reliable pattern

**Scope note on #561 (hard gate G6):** `internal/effects/net_test.go:363` already skips in CI
(`GITHUB_ACTIONS` is always set by GitHub Actions — V5). #561 is therefore a **local-developer
problem** — `make test` on a laptop or the rig reaches the public internet — **not** a CI flake.
This doc fixes it as part of unifying idiom 2 into idiom 3, but does not claim it ever reddened CI.

**Impact:**
- Every flake red on `dev` blocks the mission loop (a red `dev` outranks the queue), costs a
  full log-read to triage, and to external viewers reads as an unresolved regression (#417 concern).
- Per CLAUDE.md §3 (systemic fixes) and the mission's repeated vacuous-gate retros, patching the
  five issues individually would leave the *generator* — three idioms, one silently inert, no
  enforcement — fully intact. The next live-network test lands next month and this recurs.

## Goals

**Primary Goal:** After this sprint, the default lanes (all 6 CI `go test` legs and local
`make test`) run behind a **poisoned proxy** (`HTTP_PROXY=HTTPS_PROXY=http://127.0.0.1:9`) — a
protocol-level **default-deny for HTTP(S) egress only for clients that consult proxy environment
variables**, notably Go's default transport and `git`, rather than an enumerated host list. It
does not cover raw TCP/SSH or hand-built `http.Transport` values with a nil `Proxy`. Within that
measured scope, third-party uptime cannot red a default lane; bounded child processes and
fail-loud wiring address the other flake classes without repeating `testing.Short()`'s silent
degradation.

**Enforcement boundary vs. legibility check (scope stated plainly):** the poisoned lane is the
anti-recurrence mechanism; gatelint is a *legibility check for known offenders*, NOT the
enforcement boundary — an enumerated list cannot be complete and is not claimed to be. The
boundary covers: any-host HTTP(S) through Go's **default** transport (V25) and `git` https
clones (V26), both measured fail-fast under the poison. **What is NOT prevented (named residual
risks):** (a) a test using raw TCP, SSH, or a subprocess that ignores proxy env vars still has
ambient network access in the default lane (measured open — V27); **(b) — ADDED 2026-08-04
(iteration 142), and it is the larger hole — AILANG's OWN `Net` effect is outside the boundary.**
`internal/effects` builds **6** `http.Transport{}` literals by hand (`net.go:96,212,587`,
`stream_ndjson.go:80`, `stream_sse.go:70,329`), **none** of which sets `Proxy`, and
`ProxyFromEnvironment` appears in **0** first-party files (control: 4 first-party files build an
`http.Transport{}`, so the instrument sees positives). A hand-built transport has `Proxy == nil`
= no proxy consulted, so the poison is inert for the repo's principal HTTP client — which is the
exact code path of `#561`. Measured V33. Closing (b) is a **production runtime change** with
SSRF-guard interactions. Decision `D5` = Option A deliberately leaves it open in this sprint;
AC10(d) asserts the residual, and the separate `m-net-effect-proxy-boundary` follow-up (Option B)
will close it. AC10(d) is the tripwire that goes red when that follow-up lands. Reviewers must
flag by hand any new test that dials raw TCP/SSH or constructs its own `http.Transport`.

**Success Metrics:**
- `testing.Short()` occurrences in first-party `*_test.go`: **7 files → 0** (enforced by gate)
- Env opt-out (`Getenv("CI")`) gating in first-party `*_test.go`: **2 files → 0** (enforced by gate)
- ~~Default `go test ./...` performs **zero HTTP(S) egress** — mechanically denied~~
  **CORRECTED 2026-08-04 (iteration 142): this claim was FALSE as written and is withdrawn.**
  The poison mechanically denies HTTP(S) egress only for clients that consult proxy env vars —
  Go's *default* transport and `git`. It does **not** cover AILANG's own `Net` effect (residual
  (b) in Goals; V33), which is precisely where `#561` lives. The honest metric: the poisoned
  full suite fails in **1** package (`internal/pkg`, the `git` clone — V30 as corrected), and
  the `internal/effects` HTTP tests **pass through the poison to the live internet**. Non-HTTP
  egress AND first-party hand-built transports are named residuals, not claimed guarantees
- Hard-coded absolute wall-clock hang-guards in the four cited sites: replaced by
  deadline-derived bounds; **zero** future "raise the timeout again" commits for this class
- Both instruments prove they can fire on **every CI run**: gatelint self-tests against
  known-positive fixtures, and each poisoned `go test` step asserts its own poison env in the
  same `run:` block (removing the env fails that step); gatelint's removal/rename itself reds CI
  (registered in the existing no-silent-skip step)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Ban `testing.Short()` and env opt-out gating; standardize on explicit opt-in via one `internal/testutil` helper** (do NOT start passing `-short` in CI) | Passing `-short` would instead flip 7 inert gates to *silent mass-skipping* in CI — trading a flake for invisible coverage loss. Six of the 7 gated tests are proven CI-safe by months of green runs (V2, V3); only explicit opt-in is legible | human | design | med |
| **Hang-guards are derived from `t.Deadline()` (minus grace), never hard-coded** — and are distinguished from *assertion* bounds, which must be relative, never absolute | Every hard-coded bound in the repo has already been raised at least once chasing runners (V10, V12, ci.yml comments). Derivation ends the class; it also guarantees a hung subtest fails *before* the package-wide panic that reds every test in the package (#494's amplification) | human | design | med |
| **The lint gate (`gatelint`) = a plain Go test, not a shell script or workflow step** | A Go test runs automatically on all 6 CI legs that run `go test ./...` — ci.yml `test` + `test-windows` AND build.yml's 4-entry matrix (ubuntu, macOS amd64, macOS arm64, Windows; V34) — with zero workflow edits. Shell gates would need per-workflow wiring and can't run on the Windows legs uniformly (rig bash is 3.2; scripts/ has zero Windows coverage) | human | design | med |
| **Anti-vacuity floor**: gatelint self-tests against known-positive fixtures every run, and its PASS is asserted by name in ci.yml's existing "no silent skips" step | Without this, gatelint is the next `testing.Short()` — an assertion nobody notices has stopped firing. The repo already invented this pattern once (ci.yml:76-90 asserts `--- PASS:` per gated integration test — V9); reuse it, don't reinvent | human | design | low |
| **The enforcement boundary is a poisoned proxy in every default lane** (`HTTP_PROXY=HTTPS_PROXY=http://127.0.0.1:9` on the 6 CI `go test` legs + `make test`'s `$(GOTEST)` line); gatelint R3 is thereby demoted to a legibility/known-offender lint | An enumerated denylist cannot be complete (quorum objection, upheld). The poison is a boundary for HTTP(S) through Go's default transport (V25) and `git` HTTPS clones (V26). Measured, deliberately open residuals are raw TCP/SSH and hand-built transports with nil `Proxy`, including AILANG's `Net` effect (V27, V33, D5=A); AC10(c/d) keep them visible and reviewers flag new instances by hand | human | design | med |
| **gatelint R3 (legibility lint, not the boundary): narrow host list + checked-in allowlist, scoped to `*_test.go` only** | Measured FP surface: `https://github.com/` appears in **14** first-party test files, nearly all inert string fixtures (V16). A generic rule would either drown in false positives or grow escape hatches until vacuous. Narrow list ({`httpbin.org`, `ailang-packages`}) + allowlist = zero current FPs in test scope. Scoping to `*_test.go` is measured-necessary, not stylistic: R3's tokens appear in **6 production `.go` files** plus gatelint's own source once written (V23) — an unrestricted rule would red CI on the commit introducing it | human | design | low |
| Subprocess bounding uses `exec.CommandContext` + `cmd.WaitDelay` (Go 1.26.5 — V17) | `WaitDelay` is precisely the fix for #494's observed goroutine profile (`Cmd.Wait` + `watchCtx` both blocked); without it, ctx cancellation can still leave `Wait` blocked on I/O pipes | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved (all resolved by this doc; sprint-executor
may proceed):

- [x] Ban-and-migrate, not `-short`-in-CI (rationale in decision table row 1)
- [x] Opt-in env var name: `AILANG_LIVE_NET=1` (verified unallocated — V15)
- [x] Gate lives at `internal/testutil/gatelint/` and runs as `go test` (decision row 3)
- [x] Hang-guard grace constant: 20s below `t.Deadline()` (rationale: must beat the
      per-binary `-timeout 300s` panic with margin for cleanup + output flush)
- [x] #509's absolute `eventOneBudget` check is **deleted**, not widened (its own comment says
      "this check is redundant guardrail" and names the gap check as load-bearing — V10)
- [x] Enforcement boundary = poisoned proxy in all default lanes; sentinel frozen at
      `http://127.0.0.1:9` (discard port, no listener; measured fail-fast — V22, V25, V26)
- [x] `RequiresLiveNetwork` **hard-fails** (not skips, not unsets) when `AILANG_LIVE_NET=1` is
      combined with a poisoned proxy env — Go caches the proxy env process-wide on first use, so
      runtime unsetting silently does nothing (measured — V29); the live lane simply never sets
      the poison
- [x] gatelint walker scope: `*_test.go` files only, own package dir excluded (V23 — quorum
      objection 1, adopted)

## Solution Design

### Overview

One new package-internal convention, one shared helper file, four call-site migrations, one
default-deny egress boundary on the default lanes, one legibility gate. No language, runtime,
or compiler changes.

The repo keeps exactly **one** gating idiom: *tests that need something the default environment
does not guarantee (live network, live local daemon) opt IN via `testutil.RequiresLive*` or the
existing `*_live_test.go` pattern; everything else just runs.* Absolute wall-clock values survive
only as **relative assertions** (e.g. #509's `gap >= 200ms`, which is about *ordering*, not
runner speed) or as **derived hang-guards** (`testutil.HangGuard`). Subprocess helpers always
carry a context. The default lanes themselves run behind a poisoned proxy, so HTTP(S) egress is
denied by mechanism even for a future test that never heard of the convention. And a `gatelint`
Go test — self-tested against fixtures on every run — keeps the convention *legible* so it
cannot silently rot the way `testing.Short()` did.

### Architecture

**Components:**

1. **`internal/testutil/gate.go`** — the single gating door:
   - `RequiresLiveNetwork(t)` — `t.Skip` unless `AILANG_LIVE_NET=1`. When the var IS set, the
     test *runs* — so an opted-in broken network fails loudly, never silently skips (per #583's
     fix requirement). Skip message names the var, so the skip is self-documenting in `-v` output.
     **Poison interaction (explicit):** the live lane simply *never sets* the poison env; the
     helper additionally **hard-fails** (`t.Fatalf`, not skip) if `AILANG_LIVE_NET=1` while a
     proxy var points at the sentinel `127.0.0.1:9`. It must NOT try to unset the poison at
     runtime: Go caches the proxy env process-wide on first use, so unsetting after any prior
     request in the binary silently does nothing (measured — V29). Fail-loud per CLAUDE.md §2.
   - `HangGuard(t, cap time.Duration) time.Duration` — returns
     `min(cap, time.Until(t.Deadline()) − 20s)`, floored at 1s; returns `cap` unchanged when no
     deadline is set (`-timeout=0`). A hung operation therefore fails its own subtest with a
     clean error *before* the 300s per-binary panic reds the entire package.
   - `HangGuardContext(t, cap) (context.Context, context.CancelFunc)` — convenience wrapper.

2. **`internal/testutil/subproc.go`** — bounded subprocess runner:
   - `RunBounded(t, dir string, cap time.Duration, bin string, args ...string) (stdout, stderr string, exitCode int)` —
     `exec.CommandContext` with a `HangGuardContext`, `cmd.WaitDelay = 5 * time.Second`, and
     `cmd.Cancel` left at default (Kill). Windows-safe (kills the direct child; process-group
     grandchild cleanup is a non-goal — see Non-Goals; `internal/executor/motoko/procgroup_unix`
     is the precedent if it's ever needed).
   - This mirrors — and centralizes — the correct pattern that already exists at
     `cmd/ailang/main_run_pipe_test.go:69` (V10), which #494's helpers never got.

3. **Call-site migrations** (full inventory in the classification table below):
   - `internal/pkg/gitcache_test.go` (#583): `testing.Short()` → `testutil.RequiresLiveNetwork(t)`.
   - `internal/effects/net_test.go` (#561): env opt-out → `RequiresLiveNetwork` on both live
     subtests (`TestNetHttpPost/httpbin`, `TestNetBodySizeLimit/...` — V13); **plus** a new
     deterministic `httptest.Server`-based test covering the `netHTTPPost` success path and the
     non-2xx path, so removing live coverage from the default run *adds* coverage rather than
     subtracting it (stdlib `net/http/httptest` — no cross-package import, no boundary concern).
     Also fix the #561 defect proper: the `err == nil` branch hard-fails on an upstream 5xx body
     (V13); a non-2xx from a live host must be treated like the transport errors the test already
     tolerates (exact mechanism: Deferred Decisions).
   - `cmd/ailang/main_test.go` (#494): `runAilangBin` (line 477) and `buildAilang` (line 445)
     currently use bare `exec.Command` with no deadline (V11) → route through
     `testutil.RunBounded` / `HangGuardContext`. One hung `ailang` child then fails only its own
     test instead of panicking all of `cmd/ailang`.
   - `cmd/ailang/main_run_pipe_test.go` (#509): **delete** the `eventOneBudget` absolute check
     (lines 143–159 — self-described "redundant guardrail", V10); keep the load-bearing relative
     assertion `gap >= 200ms` (lines 129–141) untouched; replace the free-standing
     `time.After(4 * time.Second)` collect deadline (line 108) with the test's existing 10s ctx
     timeline so there is one deadline, derived once.
   - `internal/eval_harness/reference_solutions_test.go` (#587): the 60s `timeout` (line 89) is
     a **hang guard**, not an assertion — the test asserts *output*, never latency (V12). Two
     changes: (a) one unasserted **warm-up run** per language before the subtest loop (a trivial
     program, generous `HangGuard(t, 120s)` budget), so interpreter cold-start — the entire
     observed cost, 60.59s/31.07s vs 0.06s warm (issue #587 data) — is paid once, outside any
     asserted budget; (b) per-case budget becomes `testutil.HangGuard(t, 120*time.Second)`.
   - Remaining 6 inert-gated files: **delete the vacuous gate** (classification below).

4. **`internal/testutil/gatelint/`** — the legibility gate (known-offender lint; the
   enforcement boundary is component 5):
   - `scan.go` — exported `Scan(root string) []Violation`, walking `internal/`, `cmd/`,
     `runtime/`, `std/`, `tests/` only, **matching only `*_test.go` files**, skipping
     dot-directories (`.claude/` holds 5,840 stale worktree test-file copies that contaminate
     naive greps — V19) and `testdata/`, **and excluding its own package directory**
     (`internal/testutil/gatelint/`). Both scope restrictions are measured-necessary: R3's
     tokens appear in 6 production `.go` files today (V23), and the linter's own `scan.go` /
     `gatelint_test.go` must contain the rule tokens to implement and assert them — an
     unrestricted walker would false-positive all seven and red CI on the very commit that
     introduces it (quorum objection 1, adopted).
   - Rules (all three measured for FP surface **in `*_test.go` scope** before selection — V2,
     V5, V16, V24):
     - **R1**: `testing.Short(` in any first-party `*_test.go` → violation. Zero legitimate
       uses remain post-migration.
     - **R2**: `Getenv("CI")` / `Getenv("GITHUB_ACTIONS")` in any first-party `*_test.go` →
       violation (the opt-out idiom; use `RequiresLiveNetwork`). Current matches: exactly the 2
       files being migrated (V5). (`internal/coordinator/provider_script_test.go` is the second
       match; M2 audits it and either migrates it or allowlists it with a reason.)
     - **R3**: file is a first-party `*_test.go` that contains `httpbin.org` or
       `ailang-packages` AND is not `*_live_test.go` AND does not call
       `testutil.RequiresLiveNetwork(` AND is not in the checked-in allowlist (`allowlist.go`,
       each entry carrying a mandatory reason string) → violation. Seed allowlist (**5 entries,
       all measured inert**): the 2 parser files embedding `httpbin.org` in error-message
       fixtures (V16), plus the 3 files using `ailang-packages` as fixture strings —
       `internal/coordinator/agent_registry_test.go` (workspace path strings),
       `internal/messaging/config_test.go` (registry-mapping fixtures),
       `internal/pkg/manifest_test.go` (manifest-parse fixture) (V24; V16 never measured this
       token — found during the revision pass).
   - `gatelint_test.go` — **the anti-vacuity floor**:
     - `TestGateLint_SelfTest` runs `Scan` over `testdata/fixtures/` containing one deliberate
       violation per rule (R1, R2, R3), one clean test file, AND one **non-test `.go` fixture
       containing R3's tokens that must produce zero violations** (locks the `*_test.go`
       scoping so it cannot silently widen back to the V23 false-positive class) — the
       instrument demonstrably sees positives AND stays quiet on clean code, every CI run.
     - `TestGateLint_Repo` runs `Scan` over the real repo root and asserts zero violations.
   - **Loud-failure wiring**: add `TestGateLint_SelfTest` to ci.yml's existing
     "Assert binary-gated integration tests ran (no silent skips)" step (both the Linux step
     ~line 84 and the PowerShell step ~line 327 — V9), which greps for the literal
     `--- PASS: <name>`. Deleting, renaming, or skip-gating gatelint then reds CI by itself.
     build.yml needs no *gatelint* wiring (it runs `go test` over all packages — V8 — so
     gatelint executes there automatically on all three OSes), but it does gain the poison env
     (component 5).

5. **Default-lane poison wiring** — the enforcement boundary itself:
   - ci.yml's two `go test` steps (Linux ~:74, Windows ~:318), build.yml's **4-entry matrix** `go test` step (one step, 4 legs),
     and the `$(GOTEST)` invocation line of `make test` (make/test.mk:15-17) all gain
     `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 NO_PROXY=localhost,127.0.0.1`.
     Port 9 (discard) has no listener: any proxied egress fails in ~0ms (V25, V26). `NO_PROXY`
     is belt-and-braces for non-Go subprocesses talking to local daemons; Go itself already
     bypasses loopback (V28).
   - **In-step guard (anti-vacuity):** each poisoned CI step asserts its own poison env in the
     same `run:` block before invoking `go test` (bash `test` / PowerShell `if` per leg) —
     silently removing the env fails the step that depended on it, the same fail-loud pattern
     as the no-silent-skip assertion (V9).
   - **Toolchain traffic stays outside the boundary — by an EXPLICIT prefetch, never by relying
     on cache state** (rewritten after quorum R3; the original claim was refuted — see V31).
     Every poisoned step is immediately preceded by an **unpoisoned `go mod download all`**:
     both ci.yml `go test` legs, build.yml's matrix job, and — as a prerequisite of the poisoned
     `$(GOTEST)` line — `make test`. The poisoned step then runs with **`GOPROXY=off`** in
     addition to the poison env, so any dependency that was *not* prefetched fails **explicitly
     and deterministically** ("module lookup disabled") rather than as a proxy-connection error
     that would be indistinguishable from the flakes this doc exists to remove.
     **Why the original reasoning was wrong (V31):** `setup-go`'s `cache: true` is a cache
     *attempt*, not a guarantee, and neither the production build nor `make test`'s `build`
     prerequisite pulls **test-only** modules. Measured: **247** test-only dependency packages
     exist that `go list -deps ./...` never reaches, including `github.com/stretchr/testify` and
     `github.com/pmezard/go-difflib`, and the repo has **no `vendor/`** — so on a cold or missed
     cache the poisoned step would have needed the network to fetch them and would have failed.
     That would have made the enforcement boundary itself a new, nondeterministic flake source —
     precisely class C1, re-introduced by the mechanism meant to close it. The poison remains
     step-scoped env, never job- or shell-global.
   - **`internal/testutil/egress_posture_test.go`** — the posture probe (AC10):
     (a) a transport *explicitly* configured with the poison sentinel GETs an unlisted host
     (`https://example.com`) and asserts the `proxyconnect … 127.0.0.1:9 … connection refused`
     failure — deterministic on any machine, zero live network (the dial dies at the dead
     loopback port), proves the sentinel stays dead and denial is fail-fast;
     (b) a loopback `httptest` GET under the lane's poison env asserts success — the bypass
     that local-server tests and the new `httptest` replacement coverage rely on (V28); runs on
     every default lane, skips with a named message under bare unpoisoned `go test` (it
     measures the *lane*, not the machine);
     (c) under `AILANG_LIVE_NET=1` only: a raw `net.Dial` to a public host:443 asserts it
     SUCCEEDS despite the poison — documenting the open route honestly (V27). (c) lives in the
     live lane *by design*: a default-lane assertion that egress succeeds would itself
     re-create class C1.
   - The opt-in live lane (AC4, future nightly job) never sets the poison; `RequiresLiveNetwork`
     hard-fails on the mis-combination (component 1, V29).

### Classification of the 7 inert `testing.Short()` files (V2, V3, V20)

Each gate was read in context (V3). "Runs in CI today" is a fact for all seven — the gate is
inert — so deleting a gate changes nothing in CI; it only removes the false belief.

| File | What the gate guards | Disposition |
|---|---|---|
| `internal/pkg/gitcache_test.go:49` | **live `git clone` of github.com/sunholo-data/ailang-packages** (line 57) | → `RequiresLiveNetwork` (**this is #583**) |
| `internal/pipeline/validate_effects_test.go:55` | perf-scaling test, self-contained, comment says "robust on CI" | delete gate |
| `internal/gen/golang/contracts_integration_test.go:20,249` | compiles+runs Go code — toolchain always present in CI | delete gates |
| `internal/ai/ollama/client_test.go:96` | live **local** Ollama daemon — but the test *already* probe-skips when the daemon is absent (lines 106–109, V20), which is its real gate in CI | delete inert gate; keep probe-skip (see Non-Goals) |
| `internal/effects/process_test.go:437` | ~5s WaitDelay timing test, self-contained | delete gate |
| `internal/pkg/publish_validator_test.go:109,181` | timeout tests against local ailang binary, internally bounded | delete gates |
| `cmd/ailang/serve_api_mcp_surface_test.go:17` | builds+drives serve-api binary, self-contained | delete gate |

**Local behavior change (documented, intentional):** after M2, `go test -short ./...` no longer
skips these tests. `-short` was never wired into any repo entry point (V1), so nothing scripted
changes; a human habit of `-short` for speed should become package selection.

### Issue → fix mapping

| Issue | Class | Fixed by | Component |
|---|---|---|---|
| #583 live clone reds CI | C1 + C4 | opt-in gate on the clone test; the poisoned lane denies the https-clone class outright (V26); R3 keeps it legible | 1, 3, 4, 5 |
| #494 unbounded subprocess hang | C3 | `RunBounded` in `main_test.go` helpers; failure de-amplified from package-red to single-test-red | 2, 3 |
| #509 redundant absolute guardrail | C2 | guardrail deleted; relative assertion kept; single derived deadline | 3 |
| #587 node cold-start vs 60s bound | C2 | warm-up run + derived hang-guard replaces the twice-raised constant | 3 |
| #561 local `make test` needs network | C1 (local only) | opt-in gate + deterministic `httptest` replacement coverage + non-2xx tolerance | 1, 3 |
| *(next flake of any class)* | C4 | C1-next over HTTP(S): denied by the poisoned lane (a boundary, not a list); C1-next over raw TCP/SSH: **not mechanically prevented** — named residual (Non-Goals) + R3 extension; idiom rot: gatelint R1–R3 with per-run self-test, PASS asserted by name in CI; boundary rot: in-step env guard | 4, 5 |

### Implementation Plan

**M1: `testutil` gate + bounded-subprocess helpers** (~0.5–1 day)
- [ ] `internal/testutil/gate.go`: `RequiresLiveNetwork`, `HangGuard`, `HangGuardContext`
- [ ] `internal/testutil/subproc.go`: `RunBounded` (ctx + `WaitDelay`)
- [ ] Unit tests incl.: opt-in var set → helper does NOT skip; `RunBounded` kills a
      deliberately-sleeping child within the derived bound (falsifiable — remove the ctx and the
      test hangs); `HangGuard` flooring and no-deadline fallback
- [ ] Commit

**M2: call-site migrations** (~1–1.5 days)
- [ ] `gitcache_test.go` → `RequiresLiveNetwork` (#583)
- [ ] `net_test.go` → `RequiresLiveNetwork` on both live subtests; add `httptest`-based
      deterministic coverage (success + non-2xx); tolerate live non-2xx (#561)
- [ ] `main_test.go` helpers → `RunBounded`/`HangGuardContext` (#494)
- [ ] `main_run_pipe_test.go` → delete `eventOneBudget` block; unify collect deadline with ctx (#509)
- [ ] `reference_solutions_test.go` → warm-up run + `HangGuard(t, 120s)` (#587)
- [ ] Delete the 6 remaining inert `testing.Short()` gates per classification table; audit
      `internal/coordinator/provider_script_test.go` (R2's second match) and migrate or allowlist
- [ ] Full suite green locally under the poison (V30's command — `go test -count=1` over the
      make-test package list with the poison env). V30 shows pre-sprint exactly
      `internal/effects` + `internal/pkg` fail and no third package lurks, so green here proves
      the whole migration surface
- [ ] Commit

**M3: gatelint + egress-posture probe (pure Go, no workflow edits)** (~1 day)
- [ ] `internal/testutil/gatelint/{scan.go,allowlist.go}` + `testdata/fixtures/` (walker:
      `*_test.go` only, own package excluded; 5-entry seed allowlist per V16+V24)
- [ ] `gatelint_test.go`: `TestGateLint_SelfTest` (fixtures incl. the non-test R3-token fixture,
      exact-set assertion) + `TestGateLint_Repo` (real tree, zero violations)
- [ ] `internal/testutil/egress_posture_test.go` (AC10 a/b/c/d), including the deliberately-open
      `effects_nil_proxy_remains_open` subtest for D5=A
- [ ] Run AC10(c) and AC10(d) in the `AILANG_LIVE_NET=1` acceptance leg
- [ ] Commit (landed as `13c570063`)

**M4: default-lane poison wiring — the only CI-touching commit** (~1 day)
- [ ] Poison env + in-step guard on ci.yml's two `go test` steps and build.yml's 4-entry matrix step
- [ ] make/test.mk: poison env on the `$(GOTEST)` line only; prefetch modules unpoisoned and use
      `GOPROXY=off` during the poisoned test step
- [ ] ci.yml: add `TestGateLint_SelfTest` + package path to BOTH no-silent-skip steps
      (Linux ~line 84, PowerShell ~line 327)
- [ ] Run the AC12 cold-cache drill and verify all 6 CI legs
- [ ] Commit (landed as `4b47f8b0a`)

**M5: docs + verification sweep** (~0.5 day)
- [ ] `changelogs/` entry (v0.33.1); note the `-short` local behavior change
- [ ] `docs/docs/guides/development-workflow.md` (or debugging.md): the one-idiom convention,
      `AILANG_LIVE_NET`, how to write a live test, how to satisfy/extend gatelint, the poisoned
      default lanes and their **named residual** (raw TCP/SSH and hand-built transports are not
      blocked — reviewers should flag them by hand)
- [ ] Run every AC command below; paste outputs into the implementation report
- [ ] Comment-and-close #583/#494/#509/#587/#561 referencing the ACs (controller commits/closes)
- [ ] Commit

### Files to Modify/Create

**New files:**
- `internal/testutil/gate.go` (~80 LOC) — RequiresLiveNetwork, HangGuard, HangGuardContext
- `internal/testutil/gate_test.go` (~120 LOC)
- `internal/testutil/subproc.go` (~70 LOC) — RunBounded
- `internal/testutil/subproc_test.go` (~100 LOC) — incl. hung-child kill test
- `internal/testutil/gatelint/scan.go` (~150 LOC) — walker + R1/R2/R3
- `internal/testutil/gatelint/allowlist.go` (~30 LOC) — path → reason map
- `internal/testutil/gatelint/gatelint_test.go` (~120 LOC) — self-test + repo scan
- `internal/testutil/gatelint/testdata/fixtures/` (~5 small files incl. the non-test
  R3-token fixture; non-`.go` extension so no tooling ambiguity)
- `internal/testutil/egress_posture_test.go` (~60 LOC) — AC10's mechanism canary + loopback
  bypass + live-lane open-route probe

**Modified files:**
- `internal/pkg/gitcache_test.go` (+2/−3) — swap gate
- `internal/effects/net_test.go` (+60/−10) — swap gates, httptest coverage, non-2xx tolerance
- `cmd/ailang/main_test.go` (+15/−10) — bounded helpers
- `cmd/ailang/main_run_pipe_test.go` (+5/−20) — delete guardrail, unify deadline
- `internal/eval_harness/reference_solutions_test.go` (+25/−5) — warm-up + derived guard
- `internal/pipeline/validate_effects_test.go`, `internal/gen/golang/contracts_integration_test.go`,
  `internal/ai/ollama/client_test.go`, `internal/effects/process_test.go`,
  `internal/pkg/publish_validator_test.go`, `cmd/ailang/serve_api_mcp_surface_test.go`
  (−3 to −8 each) — delete inert gates
- `internal/coordinator/provider_script_test.go` (audit; migrate or allowlist)
- `.github/workflows/ci.yml` (+10/−2) — register `TestGateLint_SelfTest` in both no-silent-skip
  steps; poison env + in-step guard on both `go test` steps
- `.github/workflows/build.yml` (+5) — poison env + in-step guard on the **4-entry matrix** `go test` step (one step, 4 legs)
- `make/test.mk` (+1/−1) — poison env on the `$(GOTEST)` line of `test:` only

## Examples

### Example 1: the #583 gate (inert → explicit opt-in)

**Before** (`internal/pkg/gitcache_test.go:48-51` — author believes CI is protected; it is not):
```go
func TestGitCache_Resolve_RealRepo(t *testing.T) {
	if testing.Short() {                 // -short is never passed in CI → gate is inert
		t.Skip("skipping integration test in short mode")
	}
```

**After:**
```go
func TestGitCache_Resolve_RealRepo(t *testing.T) {
	testutil.RequiresLiveNetwork(t)      // skips unless AILANG_LIVE_NET=1; runs (and can
	                                     // fail loudly) when opted in — never silently inert
```

### Example 2: the #494 helper (unbounded → derived bound)

**Before** (`cmd/ailang/main_test.go:483-488` — one hung child panics the whole package at 300s):
```go
	cmd := exec.Command(binPath, args...)
	cmd.Dir = projectRoot
	...
	err = cmd.Run()   // no deadline: blocks until the package-wide -timeout panic
```

**After:**
```go
	stdout, stderr, exitCode := testutil.RunBounded(t, projectRoot, 60*time.Second, binPath, args...)
	// internally: HangGuardContext (min(cap, t.Deadline()−20s)) + exec.CommandContext + WaitDelay
	// a hang now fails THIS test with a clean message, ~4min before the package panic
```

### Example 3: what gatelint reports (and its self-test)

```
--- FAIL: TestGateLint_Repo (0.02s)
    gatelint_test.go:41: internal/foo/bar_test.go:17: R1 testing.Short() is inert in CI
        (no entry point passes -short — see design_docs/.../m-ci-flake-systemic-fix.md).
        Use testutil.RequiresLiveNetwork / delete the gate.
```
`TestGateLint_SelfTest` asserts the scanner finds exactly {R1,R2,R3} in `testdata/fixtures/` and
nothing in the clean fixture — so a scanner that goes blind fails CI the same run it goes blind.

## Success Criteria

Each AC states its command and answers the G5 question — *would it still pass if the claim were
false?* Scope note per V19: every grep is scoped to first-party dirs; `.claude/` is excluded.

- [ ] **AC1 (idiom eliminated):**
  `grep -rln 'testing\.Short()' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests`
  → only `internal/testutil/gatelint/testdata/` fixture path(s).
  *Falsifiable?* Yes — any surviving or new gate adds a path. Instrument control: pre-sprint the
  same command returns 7 files (V2).
- [ ] **AC2 (opt-out idiom eliminated except the reviewed escape hatch):** same grep for
  `Getenv("CI")\|Getenv("GITHUB_ACTIONS")` → only
  `internal/coordinator/provider_script_test.go`, the allowlisted-with-reason Unix
  shell/grandchild signal-semantics exception recorded by M2; no live-network gate may use the
  idiom. Pre-sprint control: 2 files (V5).
- [ ] ~~**AC3**~~ — **SUPERSEDED. The original single command was VACUOUS; `D5` is now DECIDED
  (Option A), so AC3 is replaced by AC3′(a/b/c) below.**
  ~~`HTTPS_PROXY=… HTTP_PROXY=… go test ./internal/pkg/ ./internal/effects/` → PASS~~
  **Measured 2026-08-04 (iteration 142), first-party, HEAD `9feefa3a6`:** the
  `./internal/effects/` half of this command **already passes pre-sprint, through the poison,
  by reaching the live internet** — `--- PASS: TestNetHttpPost/httpPost_to_httpbin.org`, rc=0
  poisoned and rc=0 unpoisoned, byte-identical outcomes, with `CI`/`GITHUB_ACTIONS`/
  `SKIP_NET_TESTS` all confirmed UNSET so it genuinely ran rather than skipped (V33). An AC that
  passes pre-sprint for the very behaviour it is meant to forbid is the mission's own
  **vacuous-gate** class — the same defect this document was written to close.
  **V22, which claimed the opposite, is REFUTED — see the struck row in the Verification Log.**
  Retained struck, not deleted, because the reasoning error is the point.
- [ ] **AC3′(a) — the poison is a real boundary where it actually governs (mechanical,
  non-vacuous):**
  `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/pkg/`
  → **PASS**. *Pre-sprint control, measured:* this exact command **FAILS**
  (`--- FAIL: TestGitCache_Resolve_RealRepo`, `git clone … exit status 128`), and unpoisoned it is
  `ok 3.189s`. Both directions observable, so the AC can fail.
- [ ] **AC3′(b) — the `internal/effects` egress tests are behind the opt-in, not reaching the
  internet by default.** This is the half the poison **cannot** test (V33), so it is asserted by
  lane behaviour instead of by the poison:
  `go test -count=1 -v ./internal/effects -run 'TestNetHttpPost|TestNetBodySizeLimit'` with
  `AILANG_LIVE_NET` **unset** → output contains `--- SKIP: TestNetHttpPost/httpPost_to_httpbin.org`
  and `--- SKIP: TestNetBodySizeLimit/small_response_under_limit`, **and** `--- PASS:` for the new
  deterministic `httptest` subtests. *Pre-sprint control, measured:* the same command today shows
  `--- PASS: TestNetHttpPost/httpPost_to_httpbin.org` in 0.41s, having genuinely reached the
  internet. Both directions therefore observable — forgetting to gate it leaves a `PASS` where a
  `SKIP` is required, which fails this AC.
- [ ] **AC3′(c) — whole-surface:**
  `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 $(go list ./... | grep -v /scripts)`
  → rc=0. *Pre-sprint control, measured at plan time:* rc=1, **105 ok / 1 FAIL** (`internal/pkg`),
  30 no-test-files. (The doc originally predicted 2 FAILs; the true baseline is 1 — planner
  finding R-C, and V30 is corrected accordingly.)
  *Known limit, and it is now asserted rather than assumed:* AILANG's own `Net` effect is
  **outside** this boundary by decision `D5`=A — see **AC10(d)**, which measures that residual
  openly. Raw TCP, SSH, and proxy-ignoring subprocesses are likewise outside it (measured open —
  V27), named in Goals/Non-Goals and asserted by AC10(c).
- [ ] **AC4 (opt-in path actually runs — the gate is not vacuous in the other direction):**
  `AILANG_LIVE_NET=1 go test ./internal/pkg -run TestGitCache_Resolve_RealRepo -v` on a networked
  machine **without the poison env** (the live lane never sets it) → output contains
  `--- PASS: TestGitCache_Resolve_RealRepo` (a `SKIP` fails this AC).
- [ ] **AC5 (hung child is contained):** `go test ./internal/testutil -run TestRunBounded_KillsHungChild -v`
  → PASS, and the test asserts the child was killed within the derived bound.
  *Falsifiable?* Yes — implemented against a child that sleeps far past the bound; if bounding
  is broken the test itself hangs/fails.
- [ ] **AC6 (#509 guardrail gone, assertion kept):**
  `grep -c 'eventOneBudget' cmd/ailang/main_run_pipe_test.go` → 0, **and**
  `grep -c 'minGap' cmd/ailang/main_run_pipe_test.go` → ≥1 (the second grep is the control
  proving the file/instrument is still being read — an empty file would pass the first alone).
- [ ] **AC7 (#587 constant gone):** `grep -c '60 \* time.Second' internal/eval_harness/reference_solutions_test.go`
  → 0, and warm-up run present (`grep -c 'warm' …` ≥1); `go test ./internal/eval_harness -run TestReferenceSolutions -v` PASS.
- [ ] **AC8 (gatelint sees positives every run):** `go test ./internal/testutil/gatelint -v` →
  `--- PASS: TestGateLint_SelfTest` AND `--- PASS: TestGateLint_Repo`. Manual falsification drill
  (documented in M4 report): add `testing.Short()` to any scratch first-party test file →
  `TestGateLint_Repo` FAILS; revert.
- [ ] **AC9 (gate cannot be silently dropped):** ci.yml's no-silent-skip steps (both OS legs)
  grep for `--- PASS: TestGateLint_SelfTest`. *Scope check (the vacuous-AC trap):* that step's
  `go test -run` invocation must include `./internal/testutil/gatelint` in its package list —
  reviewer must verify the package path is in the command, not just the name in the loop, since
  a name greped from a log that never ran the package would fail loudly (no PASS line), which is
  the desired direction.
- [ ] **AC10 (egress posture — the evasion probe, per quorum objection 2):**
  `go test ./internal/testutil -run TestEgressPosture -v` → PASS.
  (a) an unlisted-host GET through an explicitly-configured poison-sentinel transport fails with
  `proxyconnect … 127.0.0.1:9 … connection refused`. *Would it pass if the claim were false?*
  No — if anything listens on port 9, or the transport reaches the host directly, the request
  behaves differently and (a) fails. Pre-sprint mechanism measurement: V25.
  (b) a loopback `httptest` GET under the lane's poison succeeds (skips, named, outside a
  poisoned lane) — fails if the poison ever starts intercepting loopback, which would break the
  deterministic replacement coverage. Pre-sprint: V28.
  (c) `AILANG_LIVE_NET=1` leg only: a raw `net.Dial` to a public host:443 SUCCEEDS despite the
  poison — the AC asserts the **documented open route**, not a pretended closure (V27). If a
  future OS-level block closes the route, (c) fails and this doc's residual claim must be
  updated — the hole stays visible either way.
  (d) **`AILANG_LIVE_NET=1` leg only — the `D5`=A residual, asserted as OPEN rather than
  pretended closed.** Added 2026-08-05 (iteration 145) as the honesty half of Mark's Option A,
  mirroring (c). Build an `http.Client` **the way `internal/effects` does** — a hand-constructed
  `&http.Transport{}` with `Proxy` left nil — and GET a public URL with the poison env set; it
  **SUCCEEDS**, proving the poison does not govern AILANG's own `Net` effect. The test must
  additionally assert, in the same run, that a transport with `Proxy: http.ProxyFromEnvironment`
  set **FAILS** with `proxyconnect … 127.0.0.1:9` — that second half is the **control**, and it
  is what makes (d) non-vacuous: without it, a green (d) is indistinguishable from a poison env
  that was never set in the first place (rule 3a — an empty/passing result needs a known-positive
  in the same call). *Would (d) still pass if the claim were false?* No. If someone later sets
  `Proxy: http.ProxyFromEnvironment` on the 6 transports (i.e. adopts **Option B**), the first
  half starts failing and (d) reds — which is exactly the desired signal: **the residual closed,
  so this AC and the Non-Goals text must be retired together.** (d) is therefore the tripwire
  that will tell the Option-B design item that its work has landed.
  *Provenance:* the residual it measures is **V33** — 6 hand-built `http.Transport{}` in
  `internal/effects` (`net.go:96,212,587`, `stream_ndjson.go:80`, `stream_sse.go:70,329`), none
  setting `Proxy`; `ProxyFromEnvironment` in **0** first-party files.
  **WIDENED 2026-08-06 (iteration 148), measured first-party — the residual is larger than V33
  recorded.** V33's scope is `internal/effects`; a repo-wide sweep
  (`grep -rn 'http\.Transport{' --include='*.go' ./internal ./cmd ./runtime`, 11 hits, control
  firing) finds a **7th** proxy-ignoring literal outside it:
  `internal/executor/managed_agents/client.go:141`, which sets only `ResponseHeaderTimeout` and
  `IdleConnTimeout` and therefore bypasses the poison identically. First-party total: **7
  literals across 4 files** (`internal/effects` contributes 6 across **3** files — the doc and
  plan previously said "6 in 4 files", conflating the two scopes). AC10(d) is unaffected: it
  asserts the *mechanism* (a `Proxy`-nil transport bypasses the poison), which is
  file-independent, so it remains the correct Option-B tripwire. What changes is the documented
  EXTENT of the residual, which Option B must therefore close in 4 files, not 3.
- [ ] **AC11 (boundary cannot be silently dropped, and the lanes cannot cross):**
  `grep -c '127.0.0.1:9' .github/workflows/ci.yml` → ≥2 (both legs);
  `grep -c '127.0.0.1:9' .github/workflows/build.yml` → ≥1;
  `grep -c '127.0.0.1:9' make/test.mk` → ≥1; and each poisoned step's `run:` block asserts the
  env before invoking `go test`, so deleting the env fails that step directly (not a later
  grep). Lane-crossing guard fires:
  `AILANG_LIVE_NET=1 HTTPS_PROXY=http://127.0.0.1:9 go test ./internal/pkg -run TestGitCache_Resolve_RealRepo`
  → FAIL with the poisoned-live-lane message (V29 is why this must hard-fail rather than unset).
  *Falsifiable both directions:* AC4 proves the live lane runs; this proves the mis-configured
  combination cannot silently pass or skip.
- [ ] **AC12 (the boundary is deterministic on a COLD module cache — per quorum R3, the
      objection that a cache miss would turn the mechanism itself into a flake):** with a
      throwaway module cache, prefetch unpoisoned and then run the full poisoned suite against
      that same cache —
      `GOMODCACHE=$(mktemp -d) go mod download all` then, with that identical `GOMODCACHE`,
      `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 GOPROXY=off go test ./...`
      → the suite's result must be **indistinguishable from the warm-cache run** (same
      pass/fail set; zero `module lookup disabled` and zero `proxyconnect` errors attributable
      to dependency resolution). Record the observed output in the M3 report.
      *Would this still pass if the claim were false?* No, and that is the point: **247**
      test-only dependency packages exist that no production build pulls (V32), so if the
      prefetch step were missing or incomplete, `GOPROXY=off` makes the failure explicit and
      this AC fails. Pre-sprint, the equivalent run without the prefetch is expected to FAIL —
      run it once in that form first and record it, so the AC is demonstrated non-vacuous
      against a known-positive rather than assumed.
- [ ] All tests passing on all 6 CI legs (ci.yml test, test-windows; build.yml ubuntu, macOS amd64, macOS arm64, Windows)
- [ ] Documentation updated (M5); CHANGELOG entry

## Testing Strategy

**Unit tests:** `gate.go` (skip/no-skip both directions, poisoned-live-lane hard-fail, deadline
math edge cases: no deadline, deadline < grace), `subproc.go` (normal exit codes, stderr
capture, hung-child kill), `gatelint` (fixture exact-set, clean-file zero, allowlist honored,
dot-dir exclusion — fixture under a fake `.hidden/` must NOT be flagged — and non-test-file
exclusion — the R3-token non-test fixture must NOT be flagged), `egress_posture_test.go`
(AC10 a/b always, c/d in the live lane).

**Integration:** the migrated tests themselves on all CI legs — every leg now runs *behind the
boundary*, so the poisoned posture is exercised on every CI run, not just in a drill; AC3's
poisoned-proxy command as the local drill; AC4's opt-in run (networked machine, manual or
nightly).

**Manual:** AC8's falsification drill; one full CI round-trip on a scratch commit before M4 close.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact non-2xx handling in the live `net_test.go` subtests when opted in (issue #561 option 1
  `t.Logf`+tolerate vs. skip-on-5xx-page) — **agent may choose**; the deterministic `httptest`
  coverage is the load-bearing assertion either way
- `RunBounded` return-shape (struct vs. tuple) and whether `buildAilang` shares it or only gains
  a ctx — **agent may choose**
- Fixture file naming/extension under `gatelint/testdata/` — **agent may choose**
- Whether the warm-up program in `reference_solutions_test.go` is per-language table-driven or
  hard-coded per runner — **agent may choose**
- Whether `AILANG_LIVE_NET=1` gets a nightly opt-in CI job (would restore scheduled live
  coverage of the clone path) — **human at review**; not required to close the five issues
- **D5 — ✅ DECIDED 2026-08-05 (iteration 145): OPTION A. No longer blocking; M2/M3/M4 are
  UNBLOCKED.** *Does AILANG's own `Net` effect come inside the egress boundary?* → **No, not in
  this sprint.** Decided by Mark, relayed through the attended session, verbatim: *"D5 - option A
  and then queue the B afterwards. I'm cool with 2."* Consequences, all applied to this doc:
  (1) AC3 → **AC3′(a/b/c)**, narrowed to `./internal/pkg/` where the poison actually governs;
  (2) the `internal/effects` egress tests move behind `RequiresLiveNetwork` (AC3′(b) asserts the
  `SKIP` in the default lane, with the pre-sprint `PASS` as its control); (3) the residual is
  asserted **as open** by the new **AC10(d)**, which carries its own known-positive control and
  will go RED if Option B ever lands — making it the tripwire that retires itself;
  (4) **Option B is queued as its own design item** with its own design pass + quorum, per Mark's
  item 2. The original parked text is retained below for provenance.
  **HUMAN DECISION — parked 2026-08-04 (iteration 142), escalated on `#559`.** Measured (V33):
  the 6 hand-built `http.Transport{}` literals in `internal/effects` set no `Proxy`, so the
  poison is inert for them and AC3 passes pre-sprint by reaching the live internet. Options as
  stated in AC3: **(A)** leave `Net` outside — narrow AC3 to `./internal/pkg/`, move the
  `internal/effects` egress tests behind `RequiresLiveNetwork`, and assert the residual openly
  via a new AC10(d); the sprint then closes 4 of 5 issues mechanically and `#561` by opt-in
  migration. **(B)** bring `Net` inside — set `Proxy: http.ProxyFromEnvironment` on all 6
  transports; this is a **production runtime change** that interacts with `net.go`'s pinned-IP
  SSRF guard and needs its own design pass + quorum, so it is out of scope for a
  test-infrastructure sprint as framed. **Controller recommendation: (A) now, (B) as a separate
  queued design item** — (B) is the more correct end state but must not ride in on a sprint that
  was scoped, reviewed and quorum-cleared as test-only. M1, M3 and M5 are unaffected by this
  decision and can proceed; **M2 and M4 cannot**.

## Non-Goals

**Not attempted in this feature:**
- **`-race` in CI** — different goal (finding real bugs vs. stabilizing verdicts) and a
  substantial CI-time cost; measured absent today (V-H: 0 hits, control 7 `-timeout` hits).
  Explicitly deferred, not designed in.
- **Retry-on-failure as remedy** — rejected outright: a retry hides a deterministic regression
  exactly as well as it hides a flake; the mission paid that cost today.
- **Generic third-party-host taint analysis** — measured FP surface too large (V16: 14 files
  contain `https://github.com/` innocently). R3's narrow-list+allowlist is the honest bounded
  version; the bound is documented, not silent — and R3 is a legibility lint, NOT the
  enforcement boundary (the poisoned lane is; component 5).
- **Blocking proxy-bypassing egress (raw TCP, SSH, proxy-ignoring subprocesses, and hand-built
  `http.Transport` values)** — the poisoned proxy denies HTTP(S) only when the client consults
  proxy environment variables; raw TCP connects and a nil-`Proxy` transport bypasses it
  (measured V27/V33). OS-level egress blocking (packet filters, network namespaces) across 6 CI legs
  including macOS and Windows runners would be its own design and is out of scope this sprint.
  **These are deliberately open residual routes under D5=Option A**, stated in Goals and kept
  visible by AC10(c/d). Reviewers must flag any new test that dials raw TCP/SSH or constructs its
  own `http.Transport`. AC10(d) will turn red when `m-net-effect-proxy-boundary` adopts Option B.
- **Process-group (grandchild) cleanup on Windows** — #494's observed failure is the direct
  child; `WaitDelay` addresses it. Unix procgroup precedent exists if ever needed.
- **Unifying `buildAilang` with `testutil.FindAilangBinary`** — real overlap (V14), but
  build-caching semantics differ; out of scope beyond adding the ctx bound.
- **Replacing the ollama probe-and-skip** (`client_test.go:106-109`, V20) with opt-in — the
  probe-skip is a *silent-skip* smell but has never flaked CI; folding it in would grow the
  sprint. Recorded for a future silent-skip audit.
- **`//go:build integration` standardization** — considered (11 first-party build-tag lines
  exist, 2 are `integration` — V7) and rejected as the primary convention: build tags hide
  gated code from compilation entirely (rot risk: gated tests stop compiling and nobody sees),
  whereas env-gated tests still compile on every run. The 2 existing `integration`-tagged files
  keep their dedicated make entry (`Makefile:264`) unchanged.

## Timeline

**CORRECTED 2026-08-06 (iteration 148)** to the 5-milestone split that actually shipped — this
section previously carried the stale 4-milestone structure and contradicted the Implementation
Plan above (found by the M5 evaluator, reproduced first-party before the fix).

**Day 1:** M1 (helpers + unit tests) — landed `c440a1628`
**Day 2 → 2.5:** M2 (migrations, poisoned-proxy green) — landed `368f940cf`
**Day 3:** M3 (gatelint + egress posture probe — **pure Go, zero workflow edits**) — landed `13c570063`
**Day 3.5–4:** M4 (default-lane poison wiring — **the ONLY CI-touching commit**) — landed `4b47f8b0a`
**Day 4.5:** M5 (docs, CHANGELOG, AC sweep, issue closes) — landed 2026-08-06

**Total: planner-revised 26h ≈ 4.5 days, 5 milestones** (the original "3–4 days, 4 milestones"
estimate is superseded — see the Estimated field in the header).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| gatelint false positive blocks an unrelated PR | Med | Allowlist with mandatory reason strings; error message links this doc + shows the exact line; R1/R2 are exact-string with measured-zero legitimate remainder |
| Deleting an inert gate exposes a test that was *locally* relied on via `-short` | Low | Gates are inert in **CI** (V1) so CI cannot regress; local behavior change documented in CHANGELOG (M4); all 7 classified individually (V3), not batch-deleted |
| `HangGuard` math wrong near the 300s package deadline → later panic anyway | Med | Dedicated unit tests for the edge cases (no deadline; remaining < grace → 1s floor); grace=20s frozen in Design Freeze |
| ci.yml no-silent-skip step edit conflicts with concurrent sprints (file is actively edited) | Med | The edit is +2 lines per step, appended to existing lists; commit M3 promptly (shared-checkout discipline); rebase-early per worktree-churn memory |
| Warm-up run masks a *real* interpreter-startup regression | Low | Warm-up is unasserted by design — startup performance was never what this test asserts (V12); eval-harness benchmarks (`make/eval.mk:169`) remain the perf instrument |
| `provider_script_test.go` (R2's second match) turns out to genuinely need CI detection | Low | M2 audits it; allowlist-with-reason is the sanctioned escape, so the sprint cannot wedge on it |
| Poisoned lane breaks a test that legitimately talks HTTP to a non-loopback local service | Low | Loopback is bypassed (V28) and the full poisoned suite already passes everywhere except the 2 known migration packages (V30); `NO_PROXY` is extendable per-lane if a LAN-daemon case ever appears |
| A future test dials raw TCP/SSH or constructs a proxy-bypassing `http.Transport` and reintroduces class C1 outside the boundary | Med | The named residual (Goals, Non-Goals); AC10(c/d) keep both holes visible on every live-lane run; R3's host list extends in one line; the workflow doc tells reviewers to flag both forms by hand |

## Conflict Surface

Not a parser/typechecker change, but test infrastructure touches many packages — enumerated
concretely (G4):

1. **Files touched span 8 packages** (`internal/pkg`, `internal/effects`, `internal/pipeline`,
   `internal/gen/golang`, `internal/ai/ollama`, `internal/eval_harness`, `cmd/ailang`,
   `internal/testutil`) **+ ci.yml, build.yml, make/test.mk**. Concurrent mission agents share
   this working tree — commit each milestone promptly; no branch creation in the main checkout.
2. **ci.yml no-silent-skip steps** are the most-contended lines (edited by the jq gate, z3 gate,
   and Windows-smoke sprints historically). The M3 edit appends to an existing list — check
   `gh pr list` for in-flight ci.yml PRs before committing. The poison wiring additionally
   touches the `go test` steps themselves (env + one guard line each) in ci.yml AND build.yml —
   same contended files, same discipline.
3. **`internal/testutil` is imported by test files across layers.** It must remain
   dependency-light: `gate.go`/`subproc.go` import only stdlib + `testing`, so no
   `check-boundaries` (make/code-health.mk:139) implication. gatelint's walker likewise reads
   files as bytes — it imports no compiler packages.
4. **`go test ./...` runs in 6 CI legs and `make test`** (V34). gatelint adds one fast package
   (~ms — it reads ~937 test files' bytes once); no timeout budget concern. Anything gatelint
   flags blocks ALL of those legs at once — which is the point, but means an FP is
   maximum-visibility; hence the allowlist escape and reason strings.
5. **`make test` uses `go list ./...` with `grep -v /scripts`** (make/test.mk:15-17) — gatelint's
   package is under `internal/`, so included automatically. make/test.mk DOES change for the
   boundary: the poison env lands on the `$(GOTEST)` line only, leaving the `build` prerequisite
   unpoisoned so module fetch and compilation stay outside the boundary (V31).
6. **Programs/tests that MUST still work post-change** (regression fixtures, all verified to
   exist in V2/V3/V10/V12 reads): `TestGitCache_Resolve_RealRepo` (under `AILANG_LIVE_NET=1`),
   `TestRunCommand_PipedStdoutFlushesPerLine` (gap assertion intact), `TestCLI_Exit_Code42` and
   the other exit-code tests via bounded helper, `TestReferenceSolutions_JS` full table,
   `TestRunSmokeInTempDir_Timeout` (loses only its inert gate — its internal 1s deadline logic
   untouched), the two `-tags integration` files (unchanged, out of scope), all 6
   `*_live_test.go` files (already conform to the surviving idiom — untouched).
7. **What deliberately changes**: `go test -short ./...` local semantics (no skips — documented);
   default `make test` no longer exercises live httpbin (coverage replaced by httptest);
   `eventOneBudget` check ceases to exist (its own comment declares it redundant); default lanes
   deny HTTP(S) egress at the process-env boundary (poison), and live-network tests exist only
   in the unpoisoned opt-in lane.

## Verification Log

Every codebase claim above, one row each: the command run at HEAD `313504576` (2026-08-04) and
its observed output. Negative/empty results carry a known-positive control in the same row (G2).
All file/dir scopes are first-party (`./internal ./cmd ./runtime ./std ./tests`); `.claude/` is
excluded everywhere per V19. Shell: zsh; glob-shaped flag values quoted throughout (G3).

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | `-short` is never passed to `go test` in CI or make | `grep -rn '\-short' .github/workflows/ make/ Makefile` | 1 hit — `.github/workflows/ci.yml:215: git rev-parse --short HEAD` (false positive, not go test). **Control:** `grep -rn 'go test' .github/workflows/ make/ Makefile` → 15 hits in those same files — instrument sees positives |
| V2 | Exactly 7 first-party test files gate on `testing.Short()` | `grep -rln 'testing\.Short()' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests` | 7 files: `internal/pipeline/validate_effects_test.go`, `internal/gen/golang/contracts_integration_test.go`, `internal/ai/ollama/client_test.go`, `internal/effects/process_test.go`, `internal/pkg/gitcache_test.go`, `internal/pkg/publish_validator_test.go`, `cmd/ailang/serve_api_mcp_surface_test.go` |
| V3 | Per-file classification of each gate (table above) | `grep -n -B3 -A6 'testing\.Short()' <each file>` + full reads of guarded bodies | Contexts read for all 7; guarded content as tabulated (perf test :55, Go-toolchain integration :20/:249, ollama probe :96, 5s timing :437, bounded timeout tests :109/:181, serve-api :17, live clone :49) |
| V4 | gitcache test does a real clone of a live GitHub repo | Read `internal/pkg/gitcache_test.go:47-79` | Line 56-58: `cache.Resolve("https://github.com/sunholo-data/ailang-packages", "main", …)`; comment line 47: "requires git and network access" |
| V5 | Exactly 2 first-party test files use env opt-out gating; net_test already skips in CI (so #561 is local-only) | `grep -rln 'Getenv("CI")\|Getenv("GITHUB_ACTIONS")' --include='*_test.go' <first-party scope>` + read `internal/effects/net_test.go:361-401` | 2 files: `internal/coordinator/provider_script_test.go`, `internal/effects/net_test.go`. net_test lines 363-365: skip when `SKIP_NET_TESTS`/`CI`/`GITHUB_ACTIONS` set — GitHub Actions always sets `GITHUB_ACTIONS` |
| V6 | 6 first-party `*_live_test.go` files use the reliable opt-in idiom | `find ./internal ./cmd ./runtime ./std ./tests -name '*_live_test.go'` | 6 files (notify/discord, ai/anthropic/cache, 3× executor/managed_agents, cmd/ailang/chains) |
| V7 | Build tags are a near-unused gating convention; `integration` tag has a make entry | `grep -rn '//go:build' --include='*_test.go' <first-party scope>` ; `grep -n 'tags integration' Makefile` | 11 lines total; exactly 2 are `integration` (`internal/feedback/integration_test.go:1`, `internal/telemetry/gcp_integration_test.go:1`); rest are `!js`/`!windows`/`darwin` (+2 string-content matches in `debug_test.go`). `Makefile:264` runs `go test -tags integration ./internal/feedback/` |
| V8 | `go test` runs in 6 CI legs + make test; build.yml (the workflow #583/#587/#509 red-lit) is one of them | `grep -rn 'go test' .github/workflows/ make/ Makefile` ; read `build.yml:15-39,65` | ci.yml:74 & :318 (`go test -timeout 300s ./...`, Linux + Windows jobs); ci.yml:84 & :327 (gated re-runs); build.yml's 4-entry matrix runs ubuntu, macOS amd64, macOS arm64, and Windows; make/test.mk:15-17 |
| V9 | An anti-silent-skip CI gate pattern already exists to reuse | Read ci.yml:76-91 and :320-331 | Both jobs re-run named tests `-v` and `grep -q -- "--- PASS: $t"`, failing with `::error::` if absent — the exact anti-vacuity mechanism gatelint plugs into |
| V10 | #509's failing check is self-described as redundant; the load-bearing assertion is relative; a correct bounded-subprocess pattern already exists in the same file | Read `cmd/ailang/main_run_pipe_test.go:55-168` | Lines 148-149: "The load-bearing assertion is the gap check above (EVENT_1 → EVENT_2 ≥ 200ms); this check is redundant guardrail." Absolute check lines 150-159 (`eventOneBudget` 1500ms/3500ms). Free-standing `time.After(4 * time.Second)` line 108; 10s ctx line 67; `exec.CommandContext` + deferred `Process.Kill()` lines 69, 78-81 |
| V11 | #494's helpers are unbounded | Read `cmd/ailang/main_test.go:427-499` | `buildAilang` (line 445): `exec.Command("go","build",…)` no ctx; `runAilangBin` (line 477): `exec.Command(binPath, args...)` + `cmd.Run()` no ctx; `TestCLI_Exit_Code42` at line 531 uses it (line 542) |
| V12 | #587's 60s bound is a hang guard already raised once; the test asserts output, not latency; JS runner pays node startup per subtest | Read `internal/eval_harness/reference_solutions_test.go:56-114`, `internal/eval_harness/runner_extra_langs.go:27-67` | Line 89: `timeout := 60 * time.Second`; comment lines 82-88 records the raise ("fizzbuzz hits the same 30s cliff on Windows"). Assertions are `RuntimeOk` + exact-output only. `JSRunner.Run` LookPaths `node` and invokes per call — no warm-up exists |
| V13 | #561's defect: the live test tolerates transport errors but hard-fails on an upstream 5xx body; a second test shares the guard | Read `internal/effects/net_test.go:361-404` | `err == nil` branch (372-382) hard `t.Errorf` when body lacks `httpbin.org`; `err != nil` branch (383-385) is tolerant `t.Logf`. `TestNetBodySizeLimit` (390-404) has the same CI-env guard |
| V14 | **Negative:** no gating/skip helper exists in `internal/testutil` today | `ls internal/testutil/` + `grep -rn 'func ' internal/testutil/*.go` | Only `ailangbin.go` (+test): binary-discovery helpers (`FindAilangBinary`, `RequireAilangOnPath`, …). **Control:** the grep DID list those 6 functions — instrument sees positives; no `Requires*Network`/`Skip*`/`HangGuard` symbol exists |
| V15 | **Negative:** proposed env var `AILANG_LIVE_NET` (and alternatives) unallocated | `grep -rc 'AILANG_LIVE_NET' --include='*.go' <first-party scope>` (likewise `RUN_NET_TESTS`, `AILANG_NET_TESTS`) | 0 files each. **Control:** same command shape for `SKIP_NET_TESTS` → 1 file (`internal/effects/net_test.go`) — instrument sees positives |
| V16 | Generic URL-literal linting would have a large FP surface; narrow hosts have a small one | `grep -rl 'httpbin\.org' --include='*_test.go' <first-party scope>` ; `grep -rln 'https://github\.com/' --include='*_test.go' <same>` | `httpbin.org`: 3 files (2 parser fixture files + net_test). `https://github.com/`: **14 files**, incl. manifest/dispatch/messaging fixtures that never dial — generic rule rejected; allowlist seeded from these |
| V17 | `cmd.WaitDelay` is available | `grep '^go ' go.mod` | `go 1.26.5` (WaitDelay: Go ≥ 1.20); ci.yml/build.yml setup-go pin `1.26.5` |
| V18 | Gate-as-script precedent exists but doesn't run on Windows legs | `grep -rn 'check-boundaries' Makefile make/` → `make/code-health.mk:139` → `bash scripts/check_boundaries.sh` | Bash-based CI gate precedent confirmed; it is wired as a make target (Linux job only) — supports the gate-as-Go-test decision for 5-leg coverage |
| V19 | First-party test-file count is 937; `.claude/` contaminates naive greps but never affects go tooling | `find ./internal ./cmd ./runtime ./std ./tests -name '*_test.go' \| wc -l` → 937; naive repo-wide count → 6780 (5,840 under `.claude/`); `go list ./... \| grep -c '/\.claude/'` → 0, **control** total packages → 139 | As stated (measured 2026-08-04 at HEAD 313504576); gatelint must skip dot-dirs and all doc counts are first-party-scoped |
| V20 | The ollama test's real CI gate is a probe-skip, so deleting its inert Short() gate is safe | Read `internal/ai/ollama/client_test.go:93-110` | Lines 106-109: `CheckConnection` error → `t.Skipf("Ollama not running (expected in CI): %v", err)` |
| V21 | Issue narratives as characterized (markdown-only triggers, goroutine profile, 60.59s vs 60s, 503 body, guardrail-vs-assertion data) | `gh issue view {583,494,509,587,561} --repo sunholo-data/ailang` | Read in full 2026-08-04; quoted facts match: #583 macOS auth-prompt clone failure on md-only commit; #494 `Cmd.Wait`+`watchCtx` blocked goroutines, 5m panic, md-only commit; #509 gap=0.5008s PASSED while absolute 1.5s budget failed at 1.724s; #587 fizzbuzz 60.59s vs `:89` 60s, green descendant with identical code; #561 503 falls into `err == nil` hard-fail, CI-skipped |

| ~~V22~~ | ~~**AC3 is non-vacuous — measured, not argued.** The poisoned-proxy command FAILS pre-sprint in `internal/effects`~~ **REFUTED 2026-08-04 (iteration 142) — SUPERSEDED BY V33. The rc=1 this row observed was NOT caused by the poison; it was `httpbin.org` returning its own error page, i.e. `#561` firing naturally. The poison never touched the request (V33). The controller's original vacuity prediction was CORRECT and this row's "refutation" of it was an artifact of a third-party outage coinciding with the measurement.** Retained struck, not deleted, because the reasoning error is the point | `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test ./internal/effects/ -run 'TestNetHttpPost\|TestNetBodySizeLimit'` | **rc=1, FAIL** (1.143s). Verbose control confirms the subtest actually RAN and the assertion fired: `=== RUN TestNetHttpPost/httpPost_to_httpbin.org` then `net_test.go:380: Expected response containing 'httpbin.org', got: <html>` → `--- FAIL: TestNetHttpPost`. **Mechanism (and a refuted controller hypothesis):** the controller predicted this AC would be VACUOUS, reasoning that a dead proxy yields a *transport error* which V13 shows is tolerated by the `err != nil` → `t.Logf` branch. **Refuted:** the proxy returns an HTTP error *page*, so `err == nil` and control reaches the strict body assertion (net_test.go:372-382) instead. The AC holds — for a mechanism neither the doc nor the controller had stated precisely, now recorded so a reviewer does not re-derive it |

| # | Claim | Command | Observed |
|---|---|---|---|
| V23 | An unrestricted R3/walker would false-positive **6 production files** (+ gatelint's own source once written) — walker must match `*_test.go` only (objection 1, adopted and quantified) | `grep -rl 'httpbin\.org' --include='*.go' ./internal ./cmd ./runtime ./std ./tests \| grep -v '_test\.go$'` ; same for `ailang-packages` | `httpbin.org`: 2 files (`internal/eval_harness/httpmock.go`, `cmd/ailang/eval_benchmark.go`). `ailang-packages`: 4 files (`internal/coordinator/agent_registry.go`, `internal/executor/motoko/healthcheck.go`, `cmd/ailang/pkg_commands.go`, `cmd/ailang/init_motoko_extension.go`). Plus `gatelint/scan.go`+`gatelint_test.go`, which must contain the rule tokens = the linter would red CI on its own introducing commit. **Control:** same tokens with `--include='*_test.go'` → 3 files / 4 files — instrument sees positives in both scopes |
| V24 | 3 of the 4 `ailang-packages` matches in `*_test.go` are inert fixtures → seed allowlist needs 5 entries, not 2 (V16 never measured this token) | `grep -n 'ailang-packages' internal/coordinator/agent_registry_test.go internal/messaging/config_test.go internal/pkg/manifest_test.go` | `agent_registry_test.go:606,618,682` (workspace path strings `/tmp/ailang-packages`), `config_test.go:146-158` (registry-mapping fixtures), `manifest_test.go:231,241` (manifest-parse fixture URL) — none dials. **Control:** the 4th match, `gitcache_test.go`, genuinely clones (V4) |
| V25 | The poison denies HTTP(S) egress to **any** host (a boundary, not a list): unlisted-host GET fails fast | Go probe (poison env set at process start): `(&http.Client{Timeout: 5s}).Get("https://example.com/")` | `err=Get "https://example.com/": proxyconnect tcp: dial tcp 127.0.0.1:9: connect: connection refused` — fails at the dead loopback port, zero egress. **Control:** identical request, fresh process, no poison → `status=200` |
| V26 | `git` honors the poison for https remotes → the #583 clone class is inside the boundary | `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 git ls-remote https://github.com/sunholo-data/ailang-packages HEAD` | `fatal: unable to access '…': Failed to connect to 127.0.0.1 port 9 after 0 ms`, rc=128. **Control:** same command unpoisoned → `78439f92… HEAD`, rc=0 |
| V27 | The poison does **NOT** block raw TCP — the open route, measured rather than hidden | Same probe process (poison env set): `net.DialTimeout("tcp", "github.com:443", 5s)` | `CONNECTED to 140.82.121.4:443 (poison did NOT block)` — positive result; this IS the residual documented in Goals/Non-Goals and asserted by AC10(c) |
| V28 | Loopback bypasses the poison → httptest replacement coverage and local-daemon tests are unaffected by the boundary | Same probe process (poison env set): GET against a fresh `httptest.NewServer` URL (`http://127.0.0.1:<port>`) | `status=200 (poison bypassed for loopback)`. **Control:** the same client + poison against a non-loopback host fails (V25, same program, adjacent check) |
| V29 | Unsetting the proxy env at runtime does NOT un-poison an already-used transport (process-wide cache) → `RequiresLiveNetwork` must hard-fail on the poisoned-live combination, never unset | Go probe: poisoned GET → refused; `os.Unsetenv` both vars; GET again in the same process | Second GET **still** `proxyconnect tcp: dial tcp 127.0.0.1:9: connect: connection refused` — the cleared env had no effect. **Control:** clean process with env never set → `status=200`, so the persistence is the cache, not the network |
| V30 | ~~The poisoned **full suite** fails in exactly the 2 packages this doc migrates~~ **COUNT CORRECTED 2026-08-04 (iteration 142) to ONE package (`internal/pkg`)** — the substance survives: no third egress test lurks in first-party scope. `internal/effects` does not fail because the poison never reaches it (V33), not because it performs no egress | `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 $(go list ./... \| grep -v /scripts)` | rc=1; **104 packages `ok`, 2 FAIL**: `internal/effects` (`--- FAIL: TestNetHttpPost`) and `internal/pkg` (`--- FAIL: TestGitCache_Resolve_RealRepo`) — i.e. #561 and #583 exactly, nothing else. **Control:** unpoisoned `go test -count=1 ./internal/pkg/` → `ok 2.860s`, proving the failure is poison-induced, not pre-existing |
| V31 | ~~Toolchain module fetch happens before/outside the poisoned step in all lanes~~ **REFUTED at quorum R3 — the ordering was verified, the absence of downloads was NOT** | `grep -n 'setup-go\|cache\|go mod download' .github/workflows/ci.yml .github/workflows/build.yml` ; read make/test.mk:13-17 | ci.yml: `setup-go@v6` + `cache: true` at :28-31, :272-275, :343-346; build.yml: setup-go :48-50 + explicit `go mod download` at :59; make/test.mk: `test: build`. **These commands verify workflow ORDERING, not that `go test` performs no download** — the reviewer's exact catch, and it is correct. Superseded by V32 |
| V32 | **Test-only modules exist that no production build pulls, so a cache miss WOULD have hit the poisoned step** (the measurement V31 needed and never took) | `comm -13 <(go list -deps ./... \| sort -u) <(go list -deps -test ./... \| sort -u) \| grep -cE '^[a-z]+\.[a-z]+/'` ; same without `-c` for names ; `ls -d vendor` | **247** test-only dependency packages, including `github.com/stretchr/testify` and `github.com/pmezard/go-difflib`. `vendor/` **absent** (`no vendor/`), so these resolve over the network. **Control:** `go list -deps -test ./... \| wc -l` → **1242** total, so the 247 is a difference between two populated sets, not an empty-instrument artifact. Drives the design change: explicit unpoisoned `go mod download all` before every poisoned step, plus `GOPROXY=off` on the poisoned step so a prefetch miss fails loudly instead of masquerading as an egress violation |

| **V33** | **THE POISON IS INERT FOR AILANG'S OWN `Net` EFFECT — so AC3 is vacuous and V22 is refuted.** Poisoned and unpoisoned runs are outcome-identical, and the mechanism is a hand-built transport with `Proxy == nil` | (a) `env HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/effects/ -run TestNetHttpPost` ; (b) same without the env ; (c) the same poisoned run with `-v` ; (d) `grep -rn 'http\.Transport{' internal/effects/*.go` ; (e) `grep -rln 'ProxyFromEnvironment' --include='*.go' ./internal ./cmd ./runtime ./std ./tests` | (a) **rc=0 `ok 0.767s`**; (b) **rc=0 `ok 0.724s`** — *identical*, so the poison changed nothing; (c) `=== RUN TestNetHttpPost/httpPost_to_httpbin.org` → `--- PASS: TestNetHttpPost (0.65s)`, so it **ran and reached the live internet** rather than skipping (skip-var control in the same block: `CI`, `GITHUB_ACTIONS`, `SKIP_NET_TESTS` all **UNSET**); (d) **6** hand-built transports (`net.go:96,212,587`, `stream_ndjson.go:80`, `stream_sse.go:70,329`), and `grep -A8 … \| grep -c Proxy` → **0**; (e) **0** files. **Controls:** `grep -rln 'http\.Transport{'` first-party → **4** files, so both greps see positives; and the `git`-clone half DOES honor the poison — `go test ./internal/pkg/ -run TestGitCache_Resolve_RealRepo` poisoned → **rc=1** (`git clone failed … exit status 128`), unpoisoned → **rc=0** — which is what proves the poison env itself was live and correctly formed in the very same shell. Measured by the controller at HEAD `9feefa3a6` |
| **V34** | The default lane is **6** `go test` legs, not 5 | Read `.github/workflows/build.yml:15-39,65` ; `grep -n 'go test' .github/workflows/build.yml` | `build.yml`'s `matrix.include` has **4** entries — `ubuntu-latest`, `macos-latest/amd64`, `macos-latest/arm64`, `windows-latest` — i.e. **3 OSes but 4 jobs**, each running `go test` at `:65`. Plus ci.yml's `test` + `test-windows` = **6**. The doc's "5 CI legs" (V8 and 5 other sites) counted OSes, not jobs; every "5 legs" statement should read 6. Consequence is scope, not soundness: the poison wiring of M4 must cover 4 `build.yml` jobs |

**Provenance note:** V1, V2 (count), V5 (count), V-H (`-race`: 0 hits, control `-timeout`: 7),
and V19 were first measured by the mission controller earlier today at the same HEAD and
re-confirmed/extended here where stated more precisely (file lists, line reads). V22 was measured
by the controller at doc-review time to close the one AC whose non-vacuity claim had no
verification row (rule 3b(vi): a document's Verification Log must cover the commands its
acceptance criteria name). V23–V31 were measured by the designer during the bounded quorum
revision (2026-08-04, same HEAD), in response to the two blocking objections: V23–V24 quantify
and adopt objection 1 (gemini-3-1-pro); V25–V31 ground the objection-2 resolution
(gpt5-6-sol) — the poisoned proxy is promoted from acceptance drill to enforcement boundary,
with its closed routes (any-host HTTP(S), git-https), open routes (raw TCP), loopback bypass,
env-cache semantics, and whole-suite blast radius each measured rather than asserted. V32 was
measured by the controller at quorum R3. No claim above extends a measurement beyond its stated
scope.

## Quorum verification log

Three reviewer rounds; **every blocking objection was upheld, none was argued around**, and the
controller re-measured each one first-party before acting on it (a reviewer finding is a claim
until measured — the same rule the doc applies to itself).

| Round | Reviewers | Verdict | Objection | Resolution |
|---|---|---|---|---|
| **R1** | `gpt5-6-sol` (present), `gemini-3-1-pro` (present) | **BLOCKED** (both reject) | (1) *gpt5-6-sol:* an enumerated denylist is not a boundary — a new hostname, SSH, raw TCP or a proxy-ignoring subprocess passes gatelint and every AC, contradicting the anti-recurrence claim. (2) *gemini-3-1-pro:* R3 is unscoped, so gatelint would flag its own source and red CI on the commit that introduces it | Full designer revision pass. **(1)** resolved via the reviewer's own option (c): poisoned proxy promoted from AC to **enforcement boundary**, gatelint explicitly demoted to a legibility lint, residual (raw TCP/SSH) measured OPEN (V27) and asserted *as open* by AC10(c). **(2)** adopted: walker scoped to `*_test.go`, own package excluded, plus a non-test fixture that must yield zero violations. Controller pre-measured (2)'s blast radius before routing: **7** false positives, not the 1 reported (V23) |
| **R2** | `gemini-3-1-pro` (present); ⚠ `gpt5-6-sol` **ABSENT** | **BLOCKED — N−1 DEGRADED** | *gemini-3-1-pro:* arithmetic — 7 gated files − 1 migrated = **6** to delete, but the prose said 5 | **Narrow-refinement carve-out** (ratified; charter line ~720). Reviewer's verbatim fix applied at both sites after the controller confirmed the count against the doc's own classification table. ⚠ Absence reason recorded by name, never a silent pass: **pre-flight budget refusal, zero spend** — est. $0.1185 for a ~17.5k-token doc against the $0.10 default cap. *The doc had grown past the cap **because of the R1 revision**, so the reviewer who raised the hardest objection was silently priced out of reviewing its own resolution* |
| **R3** | `gpt5-6-sol` (present, cap raised to $0.20) | **BLOCKED** | *gpt5-6-sol:* **V31 is an invalid premise** — `setup-go cache: true` is a cache *attempt*, and neither the production build nor `make test`'s `build` prerequisite pulls test-only modules, so a cold/missed cache would require HTTPS module retrieval **inside** the poisoned step. The cited commands verify workflow ordering, not the absence of downloads | **Upheld and confirmed by controller measurement** (V32): **247** test-only dependency packages incl. `testify`/`go-difflib`, no `vendor/`, control 1242 total. Left unfixed, the enforcement boundary would itself have become a nondeterministic flake source — class C1 re-introduced by the mechanism meant to close it. Reviewer's verbatim fix applied: explicit unpoisoned `go mod download all` before every poisoned step, `GOPROXY=off` on the poisoned step so a prefetch miss fails *explicitly*, V31 struck and replaced by V32, and the demanded cold-cache drill added as **AC12** |

**R3 was not a required round** — the carve-out permitted routing after R2. The controller ran it
because R1's hardest objection had driven a structural redesign that `gpt5-6-sol` never saw, and
budget headroom was ~97% unspent. It found a real defect that would otherwise have reached a
sprint. Recorded as evidence that a *degraded* quorum round deserves completion rather than
acceptance when the absence was an artifact of cost rather than a verdict.

**Carve-out disclosure (for the human, before execution):** the R3 fix was applied by the
controller under the narrow-refinement carve-out — determinism-class, concrete reviewer-authored
`proposed_fix`, design direction uncontested (the objection *strengthens* the poisoned-proxy
boundary rather than disputing it). **No re-quorum was run on the R3 fix itself.** Mark may veto
before the sprint runs.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is test-infrastructure, not language surface — most axioms are unaffected (0). Scored for
completeness:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | CI/local verdicts stop depending on third-party uptime and runner temperature — the test suite becomes a function of the code |
| A2: Replayability | +1 | A red CI run becomes reproducible (same code → same verdict); no more "re-run at same SHA passed" (#494, #509, #587 all exhibit this today) |
| A3: Effect Legibility | +1 | A test's network dependence becomes an explicit, greppable declaration (`RequiresLiveNetwork`) instead of an inert idiom |
| A4: Explicit Authority | +1 | HTTP(S) network access is opt-in via a named env var and denied by default at the lane boundary; the non-HTTP residual is named, not silent |
| A5: Bounded Verification | +1 | Every subprocess/wait carries a derived bound; gatelint is a local, decidable check |
| A6: Safe Concurrency | 0 | No concurrency semantics change (the #494 fix bounds a wait; it doesn't alter concurrency) |
| A7: Machines First | +1 | Deterministic CI verdicts are exactly what the autonomous mission loop consumes; flakes are noise injected into agent decision-making |
| A8: Minimal Syntax | 0 | No language syntax involved |
| A9: Cost Visibility | 0 | Neutral (warm-up adds ~seconds to one Windows test package; hang-guards reduce worst-case CI minutes) |
| A10: Composability | 0 | Helpers compose with existing `*_live_test.go` idiom; no new composition surface |
| A11: Structured Failure | +1 | Hangs become clean single-test failures with messages, instead of package-wide `panic: test timed out` dumps |
| A12: System Boundary | +1 | The repo↔internet boundary in tests becomes explicit — mechanically enforced for HTTP(S) (the poisoned lane), documented-and-probed for the raw-TCP/SSH residual |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — strictly removed
- [x] A3 (Effects): no hidden side effects — network use becomes declared
- [x] A4 (Authority): no ambient access granted — ambient access revoked
- [x] A7 (Machines First): optimizes machine-consumable CI verdicts

## Related Documents

Neural search top matches (all < 0.45 → no duplicate/overlap per the coverage gate; distinctions noted):

**Implemented (may inform design):**
- [m-diagnostic-coverage](../../implemented/v0_29_0/m-diagnostic-coverage.md) (0.44) — compiler
  diagnostics coverage, not CI test gating; unrelated mechanism
- [m-poly-adt-sprint-plan](../../implemented/v0_7_1/m-poly-adt-sprint-plan.md) (0.39) — unrelated
- [m-parser-loop-sprint-plan](../../implemented/v0_7_0/m-parser-loop-sprint-plan.md) (0.39) — unrelated

**Planned (checked for overlap — none duplicates this):**
- [m-nightly-sustained-failure-label-sprint-plan](../../planned/v1_0_0/m-nightly-sustained-failure-label-sprint-plan.md)
  (0.42) — labels sustained *eval* failures in the nightly loop; different pipeline (eval bank,
  not Go CI), complementary
- [m-eval-rig-reliability](../../planned/v0_29_0/m-eval-rig-reliability.md) (0.40) — rig/eval
  infrastructure reliability, not the Go test suite; complementary
- [m-trace-feedback](../../planned/v1_1_0/m-trace-feedback.md) (0.38) — unrelated

## References

- GitHub issues: [#583](https://github.com/sunholo-data/ailang/issues/583),
  [#494](https://github.com/sunholo-data/ailang/issues/494),
  [#509](https://github.com/sunholo-data/ailang/issues/509),
  [#587](https://github.com/sunholo-data/ailang/issues/587),
  [#561](https://github.com/sunholo-data/ailang/issues/561)
- Existing anti-silent-skip gate: `.github/workflows/ci.yml:76-91, 320-331` (the pattern gatelint extends)
- Correct bounded-subprocess precedent: `cmd/ailang/main_run_pipe_test.go:67-81`
- In-repo prior art for replacing live httpbin: `internal/eval_harness/httpmock.go:16-27`
  (v0.23.0 eval decision — "non-deterministic verdicts"; this doc propagates the same decision
  to unit tests via stdlib `httptest`)
- [Design Axioms](/docs/references/axioms)
- CLAUDE.md §2 (no silent fallbacks / fail loudly) and §3 (systemic fixes, audit before patching)

## Future Work

- Nightly `AILANG_LIVE_NET=1` opt-in CI job for scheduled live coverage of the clone + httpbin paths
- Silent-skip audit: inventory probe-and-skip patterns (ollama V20, `findRepoRoot` skips) and
  decide which deserve the no-silent-skip PASS assertions
- `-race` on a scheduled (non-blocking) CI lane — explicitly deferred from this sprint
- Extend gatelint R3's host list as new third-party hosts appear in tests (legibility only —
  the boundary does not depend on the list)
- OS-level egress blocking for the non-HTTP residual (raw TCP/SSH), if that route ever
  materializes as a real flake class — would supersede AC10(c)'s documented-open assertion
- **⚠ CLASS C2's GENERATOR SURVIVES THIS SPRINT — gatelint has NO rule for absolute timeouts,
  and M2 has just WOKEN a test that carries one** (recorded 2026-08-05, iteration 145, from an
  evaluator observation the controller then reproduced and quantified). This is the same shape as
  the `exec.Command` item below, and it is recorded here with its measured number for the same
  reason: so it is a known deferral rather than a rediscovered surprise.
  **Measured at the M2 commit:** `context.WithTimeout(context.Background(), N)` — a *fixed*
  budget, not one derived from the test deadline — appears at **31** call sites in first-party
  `*_test.go`, while only **2** files use the deadline-derived `HangGuard`/`t.Deadline()` form
  (control: both greps return positives, so neither count is a broken instrument). gatelint's
  rules are **R1** (`testing.Short(`), **R2** (`Getenv("CI")`/`GITHUB_ACTIONS`), **R3** (host
  list) — **none** matches this class. So the sprint hand-fixes exactly two C2 instances
  (`eventOneBudget`, the `60 * time.Second` constant) and leaves 31 unguarded.
  **Why it is sharper than a plain backlog note:** `cmd/ailang/serve_api_mcp_surface_test.go:60`
  holds `context.WithTimeout(context.Background(), 30*time.Second)`, and that test was **dormant
  behind an inert gate until M2 removed it** (task 6). It now runs on every `go test ./cmd/ailang`
  and takes **10.34s measured on a fast local M-series machine — a 2.9× margin against its own
  fixed budget.** CI runners, and `test-windows` in particular, are materially slower than that;
  a fixed budget that is comfortable locally and marginal on CI is precisely the shape of `#494`
  and `#583`. M2 did not introduce this defect — the test carried it all along — but M2 is what
  made it live, so the risk arrives with **M4**, the commit that turns the default lanes on across
  all 6 legs. **Watch this test on the first `dev` run after M4**, and prefer a deadline-derived
  bound (`testutil.HangGuardContext`) over raising the constant.
  *Not fixed here by choice, per Standing rule 2:* a gatelint **R4** for this class needs its own
  scoping pass (31 seeds, and the legitimate-absolute-timeout question is real), exactly as §6.3
  concluded for the 62 unbounded `exec.Command` sites.

---

**Document created**: 2026-08-04
**Last updated**: 2026-08-04 — bounded quorum revision: objection 1 (gemini-3-1-pro) adopted and
quantified (walker/R3 scoped to `*_test.go`, own package excluded — V23, V24); objection 2
(gpt5-6-sol) resolved by promoting the poisoned proxy from acceptance drill to the enforcement
boundary of every default lane, with closed routes, open routes, and blast radius measured
(V25–V31) and the raw-TCP/SSH residual named rather than claimed away
