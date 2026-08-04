# Sprint Plan: M-CI-FLAKE-SYSTEMIC-FIX

**Design doc**: [m-ci-flake-systemic-fix.md](m-ci-flake-systemic-fix.md)
**Target**: v0.33.1 (current `std/VERSION` = v0.33.0)
**Planned at HEAD**: `9feefa3a6e80dd21eac6d7445ba09b5aa2c4a727` (`dev` == `origin/dev`, tree clean)
**Planner**: claude-opus-5, mission iteration 142
**Planner lane**: `opus` (derived — see "Lane derivation" below; the doc's `Planner-Lane: codex-ok` field does not and cannot yield a codex lane)
**Risk level**: **HIGH** — one milestone edits `.github/workflows/ci.yml`, `.github/workflows/build.yml` and `make/test.mk`. A bad edit reds `dev` for every concurrent session and outranks the whole mission queue.

**Revised estimate: 26h ≈ 4.5 executor-days across 5 milestones** (design doc says 3–4 days / 4 milestones — see "Estimate verdict").

---

## 0. Executive summary for the controller

The design doc is well-verified and its four structural classes (C1–C4) are real. I re-measured
21 of its 32 verification rows first-party at HEAD `9feefa3a6`. **Nineteen confirmed exactly.
Three are refuted, and one of the refutations is structural**: the doc's central mechanism —
"the poisoned proxy is a default-deny egress **boundary**" — does **not** cover AILANG's own HTTP
client, which is the single most likely source of future HTTP egress in this repo and is the
exact code path of one of the five issues (#561).

That does not kill the sprint. The poison still closes the `git`-https class (#583, measured), and
every other component of the design stands. But it changes three things the plan must carry:

1. **AC3 as written is vacuous for `internal/effects`** — it passes *pre-sprint*. Corrected below.
2. **The Goals paragraph overclaims.** "Default `go test ./...` performs zero HTTP(S) egress —
   mechanically denied" is false; it is denied only for clients using Go's *default* transport
   and for `git`. Restated honestly in M5's doc work; **flagged to Mark as a doc correction**.
3. **M2's exit criterion loses half its power.** The doc says a green poisoned full suite proves
   the whole migration surface. It cannot: the `internal/effects` half passes under poison both
   before and after. A substitute instrument is specified in M2.

Separately, there is a **live conflict** the doc could not have named: PR **#532** (open, state
`CONFLICTING`/`DIRTY`) rewrites `buildAilang` in `cmd/ailang/main_test.go` — the exact helper M2
wraps — and also rewrites the lines directly under the `testing.Short()` gate in
`cmd/ailang/serve_api_mcp_surface_test.go`, and touches `ci.yml`. PR **#569** (dependabot) touches
`ci.yml` **and** `build.yml`. Both are M2/M4 targets.

---

## 1. Verification of design-doc premises (re-measured at HEAD `9feefa3a6`)

Every command below was run first-party in this session, in the main checkout, on branch `dev`.
Negative/empty results carry a known-positive control in the same block. Counts are quoted with
`| wc -l`, never truncated with `head`.

### 1.1 CONFIRMED (19 rows)

| Row | Claim | Command | Observed | Verdict |
|---|---|---|---|---|
| V1 | `-short` never passed to `go test` | `grep -rn -- '-short' .github/workflows/ make/ Makefile` | 1 hit, `ci.yml:215 git rev-parse --short HEAD`. **Control**: `grep -rn 'go test' <same>` → 15 | CONFIRMED |
| V2 | 7 first-party `*_test.go` gate on `testing.Short()` | `grep -rln 'testing\.Short()' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests \| wc -l` | **7**, same file list as the doc | CONFIRMED |
| V4 | gitcache test clones a live GitHub repo | read `internal/pkg/gitcache_test.go:45-60` | line 47 comment "requires git and network access"; `cache.Resolve("https://github.com/sunholo-data/ailang-packages", …)` | CONFIRMED |
| V5 | 2 first-party `*_test.go` use env opt-out | `grep -rln 'Getenv("CI")\|Getenv("GITHUB_ACTIONS")' --include='*_test.go' <scope>` | `internal/coordinator/provider_script_test.go`, `internal/effects/net_test.go` | CONFIRMED |
| V6 | 6 `*_live_test.go` files | `find <scope> -name '*_live_test.go' \| wc -l` | **6** | CONFIRMED |
| V8 | `go test` steps in ci.yml ×2 + build.yml ×1 matrix + make/test.mk | `grep -rn 'go test' .github/workflows/ make/ Makefile` + reads | ci.yml:74, :318; build.yml:65; make/test.mk:17 `$(GOTEST)` | CONFIRMED (**but leg count is wrong — see 1.2**) |
| V9 | anti-silent-skip pattern exists to extend | read `ci.yml:76-91`, `:320-331` | bash `grep -q -- "--- PASS: $t"` loop (Linux); PowerShell `Select-String -SimpleMatch` loop (Windows) | CONFIRMED |
| V10 | #509's absolute check is self-described redundant; relative assertion is load-bearing | `grep -n 'eventOneBudget\|minGap\|time.After(4\|redundant guardrail' cmd/ailang/main_run_pipe_test.go` | `:149` "this check is redundant guardrail"; `:150` `eventOneBudget := 1500 * time.Millisecond`; `:135` `const minGap = 200 * time.Millisecond`; `:108` `time.After(4 * time.Second)`; `:67` 10s ctx | CONFIRMED |
| V11 | #494's helpers are unbounded | `grep -n 'func buildAilang\|func runAilangBin\|exec.Command' cmd/ailang/main_test.go` | `:466` `exec.Command("go","build",…)`, `:483` `exec.Command(binPath, …)`, **plus `:26` `exec.Command("go","run","./cmd/ailang",…)` the doc never names** | CONFIRMED + gap (1.3) |
| V12 | #587's 60s bound; test asserts output not latency | `grep -n 'timeout\|60 \* time.Second\|warm' internal/eval_harness/reference_solutions_test.go` | `:89 timeout := 60 * time.Second`; `:92` passes it to `runner.Run`; zero `warm` hits | CONFIRMED |
| V13 | #561's `err == nil` branch hard-fails on a non-2xx body | read `internal/effects/net_test.go:355-410` | `:372-382` `t.Errorf` when body lacks `httpbin.org`; `:383-385` tolerant `t.Logf`; `TestNetBodySizeLimit` shares the CI guard | CONFIRMED |
| V14 | no gating helper in `internal/testutil` today | `ls internal/testutil/` | `ailangbin.go`, `ailangbin_test.go` only. **Control**: `go list -f '{{.Imports}}'` returned 9 stdlib imports — instrument sees positives | CONFIRMED |
| V15 | `AILANG_LIVE_NET` unallocated | `grep -rc 'AILANG_LIVE_NET' --include='*.go' <scope> \| grep -v ':0' \| wc -l` → **0**. **Control**: `grep -rl 'SKIP_NET_TESTS' <same>` → 1 file | CONFIRMED |
| V16 | narrow-host FP surface small, generic surface large | `grep -rl 'httpbin\.org' --include='*_test.go' <scope> \| wc -l` → **3**; `grep -rln 'https://github\.com/' <same> \| wc -l` → **14** | CONFIRMED |
| V17 | `cmd.WaitDelay` available | `grep '^go ' go.mod` | `go 1.26.5` | CONFIRMED |
| V19 | 937 first-party test files | `find <scope> -name '*_test.go' \| wc -l` | **937** | CONFIRMED |
| V20 | ollama's real gate is a probe-skip | read `internal/ai/ollama/client_test.go:93-112` | `:106-109` `t.Skipf("Ollama not running (expected in CI): %v", err)` | CONFIRMED |
| V23/V24 | unrestricted R3 would FP on 6 production files; 5-entry seed allowlist | `grep -rl 'httpbin\.org' --include='*.go' <scope> \| grep -v '_test\.go$'` → 2; same for `ailang-packages` → 4; `--include='*_test.go'` → 3 / 4 | exactly the 6 production files the doc names; **control** both token greps return positives in test scope | CONFIRMED |
| V25 | poison denies any-host HTTP(S) through the **default** transport | Go probe, poison env at process start, `http.Client{}.Get("https://example.com/")` | `proxyconnect tcp: dial tcp 127.0.0.1:9: connect: connection refused`. **Control**, same binary no poison → `status= 200` | CONFIRMED |
| V26 | `git` honors the poison for https | `HTTPS_PROXY=… git ls-remote https://github.com/sunholo-data/ailang-packages HEAD` | `fatal: … Failed to connect to 127.0.0.1 port 9 after 0 ms`. **Control** unpoisoned → `78439f92f07b7e23adae571fbeb3520cce085d4c HEAD` | CONFIRMED |
| V27 | poison does **not** block raw TCP (named residual) | same probe, `net.DialTimeout("tcp","github.com:443",8s)` under poison | `CONNECTED to 140.82.121.3:443` | CONFIRMED (residual is real) |
| V28 | loopback bypasses the poison | same probe, GET a fresh `httptest.NewServer` URL under poison | `status= 204` | CONFIRMED |
| V29 | runtime `os.Unsetenv` does not un-poison an already-used transport | same probe: poisoned GET → refused; unset all 4 proxy vars; GET again | still `proxyconnect … connection refused`. **Control** clean process → 200 | CONFIRMED |
| V32 | 247 test-only dep packages, no `vendor/` | `comm -13 <(go list -deps ./... \| sort -u) <(go list -deps -test ./... \| sort -u) \| grep -cE '^[a-z]+\.[a-z]+/'` | **247**, incl. `stretchr/testify`, `pmezard/go-difflib`. `ls -d vendor` → No such file. **Control**: `go list -deps -test ./... \| wc -l` → **1242** | CONFIRMED exactly |
| — | AC control tokens live at HEAD | `grep -c` each | `eventOneBudget` = **4**, `minGap` = **3**, `60 * time.Second` in `reference_solutions_test.go` = **1** | CONFIRMED |
| — | `internal/testutil` cannot create an import cycle | `go list -deps ./internal/testutil \| grep sunholo` | **1** line — itself. Stdlib-only | CONFIRMED |
| — | core packages already import `internal/testutil` from tests, boundary gate green | `grep -rl 'internal/testutil' --include='*_test.go' ./internal ./cmd \| wc -l` → **18**, incl. `internal/pipeline/jwt_test.go` (`pipeline` IS in `CORE_PKGS`, `scripts/check_boundaries.sh:43`) | precedent exists and is green today | CONFIRMED |

### 1.2 REFUTED (3 rows) — say so plainly

#### R-A. **V22 is REFUTED. AC3 is vacuous for `internal/effects`.**

The doc's V22 claims the poisoned-proxy command **fails** pre-sprint in `internal/effects`, and
builds AC3's entire non-vacuity argument on it (including a narrative about the controller's
"refuted hypothesis"). It does not fail. Measured:

```
$ HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 \
    go test -count=1 -v ./internal/effects/ -run 'TestNetHttpPost|TestNetBodySizeLimit'
=== RUN   TestNetHttpPost/httpPost_to_httpbin.org
--- PASS: TestNetHttpPost/httpPost_to_httpbin.org (0.41s)
--- PASS: TestNetBodySizeLimit/small_response_under_limit (0.37s)
ok  	github.com/sunholo-data/ailang/internal/effects	1.017s
```

Env control in the same block: `CI=[unset] GITHUB_ACTIONS=[unset] SKIP_NET_TESTS=[unset]` — the
subtests were **not** skipped, they ran and reached `httpbin.org` in 0.41s **through the poison**.

#### R-B. **Root cause of R-A, and the structural finding: the poison is not a boundary for first-party HTTP.**

```
$ grep -rn -A6 'http\.Transport{' internal/effects/*.go | grep -c 'Proxy'
0
$ grep -rl 'http\.Transport{' --include='*.go' ./internal ./cmd | wc -l
4
$ grep -rl 'ProxyFromEnvironment' --include='*.go' ./internal ./cmd | wc -l
0
```

`internal/effects/net.go:96,212,587`, `stream_ndjson.go:80`, `stream_sse.go:70,329` all construct
bare `&http.Transport{…}` literals. A hand-built `http.Transport` has `Proxy == nil`, i.e. **no
proxy**, unless `Proxy: http.ProxyFromEnvironment` is set explicitly — and **zero** first-party
files set it. `net.go` additionally pins `DialContext` to a pre-validated IP (SSRF hardening),
so it dials the target directly by construction.

Consequence: **AILANG's own `Net` effect — the repo's principal HTTP client — is entirely outside
the poisoned-proxy boundary.** This is a larger hole than the doc's named raw-TCP residual,
because it is first-party code the repo owns, uses in production, and will keep adding tests for.

*I am not proposing to close it in this sprint.* Adding `Proxy: http.ProxyFromEnvironment` to
`net.go` would (a) be a **runtime behavior change**, which this doc explicitly disclaims, and
(b) likely defeat the pinned-IP SSRF guard, since a proxy resolves the hostname itself. That is a
separate design decision for Mark. See §6 "Decisions for the controller".

#### R-C. **V30 is REFUTED in its count; CONFIRMED in its "nothing else lurks" half.**

The doc claims the poisoned full suite fails in **exactly 2** packages. Measured (full run,
`go test -count=1 $(go list ./... | grep -v /scripts)` under the poison):

```
EXIT_RC=1
ok  count: 105          no-test-files: 30
--- FAIL: TestGitCache_Resolve_RealRepo (0.16s)
FAIL	github.com/sunholo-data/ailang/internal/pkg	4.265s
```

**Exactly 1 package fails, not 2**: `internal/pkg`. `internal/effects` passes (per R-A/R-B).
Control: `go test -count=1 ./internal/pkg/` unpoisoned → `ok 3.189s`, so the failure is
poison-induced, not pre-existing.

The useful half survives: **no third egress-dependent package lurks** anywhere in first-party
scope. That premise is CONFIRMED and M2 can rely on it.

#### R-D. **CI leg count is 5 in the doc; it is 6.**

`build.yml`'s matrix has **four** include entries, not three:
`ubuntu-latest`, `macos-latest`/amd64, `macos-latest`/arm64, `windows-latest`
(`.github/workflows/build.yml:20-39`). Plus `ci.yml` `test` and `test-windows` = **6 `go test`
legs**. Minor, but the doc's final Success Criterion literally says "all 5 CI legs" and M4's
poison wiring must be right on all of them. One `build.yml` step edit covers all four matrix legs.

### 1.3 Premises I could not check, or checked only partially

| Premise | Why not checked | Mitigation in the plan |
|---|---|---|
| V3 (per-file classification of all 7 gates) | Requires reading 7 guarded bodies in full; I spot-read `gitcache_test.go` (V4) and `client_test.go` (V20) and both matched | M2 task 6 re-reads each gate in context before deleting it, one file at a time, and records the disposition |
| V7 (11 `//go:build` lines, 2 `integration`) | Out of the sprint's change surface (Non-Goal) | none needed |
| V18 (`check-boundaries` is bash-only) | Superseded — I verified the stronger fact directly: `internal/testutil` is stdlib-only and 18 test files across core packages already import it, green | n/a |
| V21 (five GitHub issue narratives) | Would require 5 `gh issue view` calls; the code-side facts they assert (V10/V11/V12/V13) all CONFIRMED independently | M5 re-reads each issue when writing the closing comments |
| V31 | Already struck by the doc itself and replaced by V32 (CONFIRMED) | n/a |
| **AC12's cold-cache drill** | A throwaway `GOMODCACHE` + `go mod download all` is a multi-GB fetch and a full second suite run (~25 min). Not run at plan time | **M4 must run it**, and must run the *un*-prefetched form first to prove the AC is non-vacuous (see M4 task 5) |
| Behavior of build.yml's test step shell on `windows-latest` | Would need a CI round-trip | M4 task 2 pins `shell: bash` on that step rather than guessing; see M4 risk table |

### 1.4 New hazards found at plan time (not in the doc)

1. **PR #532 is a direct, live collision with M2.**
   `gh pr view 532 --json mergeable,mergeStateStatus` → `{"m":"CONFLICTING","s":"DIRTY"}`,
   not a draft, last updated 2026-07-29. Files:
   `.github/workflows/ci.yml`, `cmd/ailang/main_test.go`, `cmd/ailang/serve_api_mcp_surface_test.go`,
   `cmd/ailang/prompt_test.go`, `cmd/ailang/budget_scoping_e2e_test.go`, `cmd/ailang/pkg_lock_ratchet_test.go`.
   Its diff **replaces `buildAilang` wholesale** with a `sync.Once` + `os.MkdirTemp` +
   `TestMain`-cleanup design (to fix a Windows 300s-timeout red), and rewrites the lines directly
   beneath the `testing.Short()` gate in `serve_api_mcp_surface_test.go`. M2 edits both.
   → **Controller decision required before M2** (§6).

2. **PR #569** (dependabot, actions group) touches `.github/workflows/ci.yml` **and**
   `.github/workflows/build.yml` — both M4 targets.

3. **The `httptest` replacement coverage for #561 needs two capability flags the doc never
   mentions.** `internal/effects/context.go:194-195,219-220`:
   `AllowHTTP: false` (https-only) and `AllowLocalhost: false` (127.x blocked) are the defaults.
   An `httptest.NewServer` URL is `http://127.0.0.1:<port>` — blocked twice over.
   `httptest.NewTLSServer` fails cert verification instead. The new test **must** set
   `ctx.Net.AllowHTTP = true` and `ctx.Net.AllowLocalhost = true`. Two lines, but a
   non-obvious 30-minute dead end otherwise, and it means the deterministic replacement exercises
   a different capability posture than the live test did — worth one sentence in the test comment.

4. **C3's generator survives this sprint.** Bare unbounded `exec.Command` in first-party
   `*_test.go`: **62 repo-wide**, **25 in `cmd/ailang` alone**
   (`grep -rn 'exec\.Command(' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests | wc -l`;
   control: `exec.CommandContext` in `./cmd/ailang` tests = 2). The doc's M2 bounds **2** of them
   and misses a third *in the same file* (`cmd/ailang/main_test.go:26`, the `go run ./cmd/ailang`
   helper). gatelint has no rule for this class. Per CLAUDE.md §3 this is exactly the
   "patch the symptom, leave the generator" shape the doc otherwise avoids. See §6.

5. **`.ailang/` is gitignored but 45 sprint JSONs are tracked** (`git check-ignore -v` → `.gitignore:77`;
   `git ls-files .ailang/state/sprints/ | wc -l` → 45). The controller must
   `git add -f .ailang/state/sprints/sprint_M-CI-FLAKE-SYSTEMIC-FIX.json`.

6. **`actionlint` is installed** (`/opt/homebrew/bin/actionlint`) — M4 gets a real pre-commit
   syntax gate on the workflow edits instead of "push and hope".

---

## 2. Lane derivation

The design doc's line 8 reads `**Planner-Lane**: codex-ok (mechanical Go test-infra work …)`.
`tools/launchd/derive-planner-lane.sh` requires the bare token and therefore emits
`opus fail-closed:planner-lane-field-invalid`. The controller tested a corrected copy of the
field: it yields `opus fail-closed:path-not-in-codex-allowlist`. **`opus` is the correct lane
either way** — "fix the field to get a codex lane" would not work and is not proposed.

Independently, this sprint is a **poor codex fit**: M4's verdicts depend on GitHub Actions
round-trips, and M1/M3's verdicts depend on subprocesses and loopback sockets that a
`workspace-write` sandbox denies (§5).

---

## 3. Milestone breakdown

Five milestones, not the doc's four. **The split is deliberate and is the plan's main structural
change**: the doc's M3 bundles a pure-Go package (gatelint, zero blast radius on `dev`) with the
CI-workflow edits (maximum blast radius on `dev`) in one commit. Constraint: helpers and tests
land first, the enforcement boundary lands last, and the *only* commit that can red `dev` for
concurrent sessions is isolated and revertable by itself.

Ordering constraint that must not be relaxed: **M2 (migrations) must land before M3 (gatelint)**,
because `TestGateLint_Repo` asserts zero violations against the real tree and would red on the
7 `testing.Short()` files if it landed first.

| # | Milestone | LOC | Hours | Blast radius on `dev` | Closes |
|---|---|---|---|---|---|
| M1 | `testutil` gate + bounded subprocess helpers | ~370 | 5.0 | **None** — new package, nothing imports it yet | (enables AC5) AC5 |
| M2 | Call-site migrations + `httptest` replacement coverage | ~250 net | 7.0 | **Medium** — 9 test files in 8 packages; a break is a named test failure, one `git revert` | AC1, AC2, AC3′, AC4, AC6, AC7 |
| M3 | gatelint + egress posture probe (pure Go, **no workflow edits**) | ~460 | 5.0 | **Medium-high** — lands a gate that runs on all 6 legs at once; but zero workflow files touched | AC8, AC10 |
| M4 | **Default-lane poison wiring — the only CI-touching commit** | ~45 | 6.0 | **MAXIMUM** — reds `dev` for every concurrent session if wrong | AC9, AC11, AC12 |
| M5 | Docs, CHANGELOG, AC sweep, issue closes | ~140 | 3.0 | None | remaining checklist |
| | **Total** | **~1265** | **26.0** | | AC1–AC12 |

---

### M1 — `internal/testutil` gate + bounded subprocess helpers

**Estimate**: ~370 LOC, 5.0h. **Blast radius: none.** New package files only; no existing file is
modified; nothing imports the new symbols yet. `go test ./...` gains one fast package.

**Tasks**
1. `internal/testutil/gate.go` (~90 LOC)
   - `liveNetworkStatus() (ok bool, reason string)` — **the predicate must be extracted**, see
     the anti-vacuity note below.
   - `RequiresLiveNetwork(t *testing.T)` — thin wrapper: `t.Skip` unless `AILANG_LIVE_NET=1`;
     `t.Fatalf` (never skip, never unset) if `AILANG_LIVE_NET=1` **and** any of
     `HTTP_PROXY`/`HTTPS_PROXY`/`http_proxy`/`https_proxy` points at `127.0.0.1:9` (V29 is why:
     Go caches the proxy env process-wide, so unsetting is a silent no-op).
   - `HangGuard(t, cap) time.Duration` → `min(cap, time.Until(t.Deadline()) − 20s)`, floored at
     1s; returns `cap` unchanged when `t.Deadline()` reports no deadline.
   - `HangGuardContext(t, cap) (context.Context, context.CancelFunc)`.
2. `internal/testutil/subproc.go` (~70 LOC) — `RunBounded` via `exec.CommandContext` +
   `cmd.WaitDelay = 5 * time.Second`, `cmd.Cancel` left at default (Kill). Return shape is the
   implementer's choice (doc: Deferred Decisions).
3. `internal/testutil/gate_test.go` (~130 LOC), `internal/testutil/subproc_test.go` (~80 LOC).

**Anti-vacuity: the mutation each new assertion must fail under**

| Test | Mutation that must make it RED | How to prove the mutation landed |
|---|---|---|
| `TestLiveNetworkStatus_OptInSetRuns` (`AILANG_LIVE_NET=1` → `ok == true`) | invert the predicate to `!= "1"` | `git diff internal/testutil/gate.go` shows the flipped comparison **before** the run |
| `TestLiveNetworkStatus_UnsetSkips` (unset → `ok == false`, reason names the var) | make the predicate return `true` unconditionally | same |
| `TestRequiresLiveNetwork_PoisonedLiveLaneFatal` | change `t.Fatalf` to `t.Skip` | same |
| `TestRunBounded_KillsHungChild` (child `sleep 60`, cap 2s; assert wall time < 10s **and** non-zero exit) | replace `exec.CommandContext` with `exec.Command` | `grep -c 'exec.CommandContext' internal/testutil/subproc.go` → 0 before the run. **Note**: under this mutation the test **hangs** rather than fails — the executor must run it with `-timeout 30s` so the mutation produces a bounded red, and must record that timeout, not a hang, is the observed mutant signal |
| `TestHangGuard_FloorsAtOneSecond` | change the floor constant to `0` | `git diff` shows the constant |
| `TestHangGuard_NoDeadlineReturnsCap` (run under `-timeout 0`) | return `0` when no deadline | `git diff` |

> **Why `liveNetworkStatus` must be extracted (this is the load-bearing design note for M1).**
> You cannot assert "`t.Skip` was **not** called" from inside the test that would be skipped —
> the runtime unwinds. Every in-process attempt to test the no-skip direction of a skip helper is
> vacuous by construction. This mission has shipped that class of assertion before. Testing the
> **predicate** makes both directions real. `RequiresLiveNetwork` itself stays a 3-line wrapper
> whose only untested behavior is calling `t.Skip`, which AC4 covers end-to-end from outside.

**Acceptance (M1)**
```bash
go test -count=1 -timeout 60s ./internal/testutil -v          # all new tests PASS
go vet ./internal/testutil && golangci-lint run ./internal/testutil/...
go list -deps ./internal/testutil | grep sunholo | wc -l      # → 1 (itself): stdlib-only, no cycle
```
Closes **AC5** (`go test ./internal/testutil -run TestRunBounded_KillsHungChild -v` → PASS).

**UNINFORMATIVE UNDER SANDBOX**: `TestRunBounded_*` spawns subprocesses. A `workspace-write`
sandbox verdict on these is not evidence. **Controller must re-run `go test ./internal/testutil`
outside the sandbox** before M1 is considered green.

---

### M2 — Call-site migrations + deterministic replacement coverage

**Estimate**: ~250 LOC net (+320 / −70), 7.0h. **Blast radius: medium.** Nine test files across
eight packages. A mistake shows up as a named test failure in `go test ./...` locally and on CI;
one `git revert` restores. No production code, no workflows.

**Do this first (5 min, before any edit):**
```bash
git fetch origin && git log --oneline origin/dev -1     # confirm base
gh pr view 532 --json mergeable,mergeStateStatus,updatedAt
```
If #532 has been rebased or merged since planning, re-read `cmd/ailang/main_test.go` from disk
before touching `buildAilang`. See §6 for the controller's decision on #532.

**Tasks**
1. `internal/pkg/gitcache_test.go` — `testing.Short()` → `testutil.RequiresLiveNetwork(t)` (#583).
2. `internal/effects/net_test.go` (#561):
   - `RequiresLiveNetwork` on both live subtests (`TestNetHttpPost/httpPost to httpbin.org`,
     `TestNetBodySizeLimit/small response under limit`), replacing the `SKIP_NET_TESTS`/`CI`/
     `GITHUB_ACTIONS` opt-out.
   - Tolerate a live non-2xx in the opted-in path (the V13 defect). Mechanism is the
     implementer's choice per Deferred Decisions.
   - **New deterministic `httptest` coverage** for `netHTTPPost` success + non-2xx.
     **Must set `ctx.Net.AllowHTTP = true` and `ctx.Net.AllowLocalhost = true`** — both default
     `false` (`internal/effects/context.go:219-220`) and an `httptest` URL is
     `http://127.0.0.1:<port>`, blocked by both. Comment why.
3. `cmd/ailang/main_test.go` (#494) — route `runAilangBin` (`:483`) and `buildAilang` (`:466`)
   through `testutil.RunBounded` / `HangGuardContext`. **Also bound `:26`** (the
   `exec.Command("go","run","./cmd/ailang",…)` helper) — the doc's V11 missed it and it is the
   same defect in the same file.
4. `cmd/ailang/main_run_pipe_test.go` (#509) — delete the `eventOneBudget` block (`:148-159`);
   keep `minGap` (`:135-140`) untouched; replace `time.After(4 * time.Second)` (`:108`) with the
   existing 10s ctx (`:67`).
5. `internal/eval_harness/reference_solutions_test.go` (#587) — one unasserted warm-up run per
   language before the subtest loop (generous `HangGuard(t, 120*time.Second)`), then per-case
   budget = `testutil.HangGuard(t, 120*time.Second)`.
6. Delete the 6 remaining inert `testing.Short()` gates, **re-reading each guarded body in
   context first** (V3 is the one premise I did not fully re-verify):
   `internal/pipeline/validate_effects_test.go:55`,
   `internal/gen/golang/contracts_integration_test.go:20,249`,
   `internal/ai/ollama/client_test.go:96` (keep the `:106-109` probe-skip),
   `internal/effects/process_test.go:437`,
   `internal/pkg/publish_validator_test.go:109,181`,
   `cmd/ailang/serve_api_mcp_surface_test.go:17`.
7. Audit `internal/coordinator/provider_script_test.go` (R2's second match) — migrate or record
   an allowlist reason for M3.

**Anti-vacuity**

| Assertion | Pre-sprint control (measured today) | Mutation that must make it RED |
|---|---|---|
| AC1 grep → 0 files | **7** files at HEAD | leave any one gate in place |
| AC2 grep → 0 files | **2** files at HEAD | leave `net_test.go`'s opt-out |
| AC6 `eventOneBudget` → 0 | **4** at HEAD; `minGap` control **3** → must stay ≥1 | delete the wrong block |
| AC7 `60 * time.Second` → 0 | **1** at HEAD | leave the constant |
| **AC3′ (see below)** | `internal/pkg` poisoned = FAIL, unpoisoned = ok (measured) | skip-gate the wrong test |
| **New: default-lane skip proof** | pre-sprint `go test -v ./internal/effects -run TestNetHttpPost` **RUNS and PASSES** `httpPost to httpbin.org` (measured) | forget to gate it → it still RUNS, assertion red |

**AC3 must be corrected — it is vacuous as written (finding R-A).** Replace with:

- **AC3a (mechanical, non-vacuous):**
  `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/pkg/`
  → PASS. Pre-sprint control: this exact command **FAILS** (`--- FAIL: TestGitCache_Resolve_RealRepo`,
  `git clone … exit status 128`), and unpoisoned it is `ok 3.189s`. Non-vacuous, measured.
- **AC3b (replaces the `internal/effects` half, which the poison cannot test):**
  `go test -count=1 -v ./internal/effects -run 'TestNetHttpPost|TestNetBodySizeLimit'` with
  `AILANG_LIVE_NET` **unset** → output contains
  `--- SKIP: TestNetHttpPost/httpPost_to_httpbin.org` and
  `--- SKIP: TestNetBodySizeLimit/small_response_under_limit`, **and**
  `--- PASS:` for the new `httptest` subtests. Pre-sprint control: the same command shows
  `--- PASS: TestNetHttpPost/httpPost_to_httpbin.org` in 0.41s having actually reached the
  internet. Both directions therefore observable.
- **AC3c (whole-surface):**
  `HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 $(go list ./... | grep -v /scripts)`
  → rc=0. Pre-sprint control recorded in this plan: rc=1, **105 ok / 1 FAIL** (`internal/pkg`),
  30 no-test-files. Note the doc predicted 2 FAILs; the true baseline is 1 (finding R-C).

**Acceptance (M2)**
```bash
grep -rln 'testing\.Short()' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests | wc -l   # → 0
grep -rln 'Getenv("CI")\|Getenv("GITHUB_ACTIONS")' --include='*_test.go' ./internal ./cmd ./runtime ./std ./tests | wc -l  # → 0 (or 1 with a recorded allowlist reason)
grep -c 'eventOneBudget' cmd/ailang/main_run_pipe_test.go   # → 0
grep -c 'minGap' cmd/ailang/main_run_pipe_test.go           # → ≥1  (instrument-alive control)
grep -c '60 \* time.Second' internal/eval_harness/reference_solutions_test.go  # → 0
grep -ci 'warm' internal/eval_harness/reference_solutions_test.go             # → ≥1
go test -count=1 $(go list ./... | grep -v /scripts)                          # rc=0
HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 go test -count=1 $(go list ./... | grep -v /scripts)   # rc=0   [AC3c]
go test -count=1 -v ./internal/effects -run 'TestNetHttpPost|TestNetBodySizeLimit'                                   # [AC3b]
AILANG_LIVE_NET=1 go test -count=1 ./internal/pkg -run TestGitCache_Resolve_RealRepo -v                              # [AC4] → --- PASS, not SKIP
```
Closes **AC1, AC2, AC3′(a/b/c), AC4, AC6, AC7**.

**UNINFORMATIVE UNDER SANDBOX**: everything in `cmd/ailang` (subprocess-heavy), the new
`httptest` subtests (loopback socket), `internal/eval_harness` (spawns `node`/`uv`), and AC4
(live network). **Controller must re-run the full suite and AC3b/AC3c/AC4 outside the sandbox.**

---

### M3 — gatelint + egress posture probe (pure Go, zero workflow edits)

**Estimate**: ~460 LOC, 5.0h. **Blast radius: medium-high but structurally contained.** The
moment this lands, `TestGateLint_Repo` runs on all 6 CI legs and on every developer's
`make test`; a false positive is a maximum-visibility red. **But it touches no workflow file**, so
revert = delete one directory + one file, and no other session's CI configuration changes.

Mitigation for the FP risk: run `go test ./internal/testutil/gatelint` against the real tree
immediately before committing, and keep the M2→M3 gap short (a concurrent session adding a
`testing.Short()` between M2 and M3 would red M3 on landing).

**Tasks**
1. `internal/testutil/gatelint/scan.go` (~160 LOC) — `Scan(root string) []Violation`. Walks
   `internal/`, `cmd/`, `runtime/`, `std/`, `tests/`; **matches `*_test.go` only**; skips
   dot-directories and `testdata/`; **excludes its own package directory**. Both restrictions are
   measured-necessary (V23, CONFIRMED: 6 production files carry R3's tokens).
   Rules R1 (`testing.Short(`), R2 (`Getenv("CI")` / `Getenv("GITHUB_ACTIONS")`),
   R3 (`httpbin.org` / `ailang-packages` in a non-`*_live_test.go` that does not call
   `testutil.RequiresLiveNetwork(` and is not allowlisted).
2. `internal/testutil/gatelint/allowlist.go` (~30 LOC) — path → mandatory reason string. Seed
   with the 5 measured-inert entries (2 parser `httpbin.org` fixture files;
   `internal/coordinator/agent_registry_test.go`, `internal/messaging/config_test.go`,
   `internal/pkg/manifest_test.go` for `ailang-packages`), plus `provider_script_test.go` if M2's
   audit chose allowlist-with-reason.
3. `internal/testutil/gatelint/testdata/fixtures/` — one deliberate violation per rule; one clean
   test fixture; **one non-test `.go`-shaped fixture containing R3's tokens that must yield zero
   violations** (locks the `*_test.go` scoping); one fixture under a fake `.hidden/` dir that must
   not be flagged. Non-`.go` extension so `go build`/`go vet` never see them.
4. `internal/testutil/gatelint/gatelint_test.go` (~130 LOC) — `TestGateLint_SelfTest`
   (exact-set assertion over fixtures) and `TestGateLint_Repo` (real tree, zero violations).
5. `internal/testutil/egress_posture_test.go` (~70 LOC) — AC10 (a) poison-sentinel transport GET
   of `https://example.com` asserts `proxyconnect … 127.0.0.1:9 … connection refused`;
   (b) loopback `httptest` GET under the lane's poison succeeds, skipping with a **named** message
   outside a poisoned lane; (c) `AILANG_LIVE_NET=1` only: raw `net.Dial` to a public host:443
   SUCCEEDS despite the poison — the honestly-asserted open route (V27, CONFIRMED).

**Anti-vacuity**

| Assertion | Mutation that must make it RED | Proof the mutation landed |
|---|---|---|
| `TestGateLint_SelfTest` R1 arm | delete the R1 branch from `scan.go` | `grep -c 'R1' internal/testutil/gatelint/scan.go` drops, shown before the run |
| `TestGateLint_SelfTest` non-test-fixture arm (must find **zero**) | widen the walker from `*_test.go` to `*.go` | `git diff scan.go` shows the changed suffix filter |
| `TestGateLint_SelfTest` dot-dir arm | remove the dot-dir skip | `git diff` |
| `TestGateLint_Repo` | **manual falsification drill (AC8)**: create `internal/pipeline/scratch_gatelint_test.go` containing `testing.Short()` → `TestGateLint_Repo` FAILs naming that path → delete the file → PASS | **`grep -c 'testing.Short()' internal/pipeline/scratch_gatelint_test.go` → 1 before the failing run, and `ls` → No such file after.** The drill's rc means nothing without both |
| AC10(a) | point the probe transport at a live proxy instead of `127.0.0.1:9` | `git diff` on the sentinel constant |
| AC10(b) | make the poison intercept loopback (add `127.0.0.1` removal from `NO_PROXY`) | `git diff` on the test's env setup |

**Acceptance (M3)**
```bash
go test -count=1 -v ./internal/testutil/gatelint    # --- PASS: TestGateLint_SelfTest  AND  --- PASS: TestGateLint_Repo
go test -count=1 -v ./internal/testutil -run TestEgressPosture           # AC10 (a)+(b)
AILANG_LIVE_NET=1 go test -count=1 -v ./internal/testutil -run TestEgressPosture   # AC10 (c)
go test -count=1 $(go list ./... | grep -v /scripts)                     # rc=0 — gatelint is green against the REAL tree
golangci-lint run ./internal/testutil/...
# falsification drill, with the mutation-landed proof:
cat > internal/pipeline/scratch_gatelint_test.go <<'EOF'
package pipeline
import "testing"
func TestScratchGatelintDrill(t *testing.T) { if testing.Short() { t.Skip("drill") } }
EOF
grep -c 'testing.Short()' internal/pipeline/scratch_gatelint_test.go     # MUST print 1 — proof the mutation landed
go test -count=1 -run TestGateLint_Repo ./internal/testutil/gatelint; echo "drill rc=$?"   # MUST be non-zero
rm internal/pipeline/scratch_gatelint_test.go
go test -count=1 -run TestGateLint_Repo ./internal/testutil/gatelint; echo "restored rc=$?" # MUST be 0
git status --porcelain | grep scratch_gatelint || echo "drill file removed"
```
Closes **AC8, AC10**.

**UNINFORMATIVE UNDER SANDBOX**: AC10(b) binds a loopback listener (`httptest.NewServer`);
AC10(c) needs live network. `TestGateLint_*` is pure file-reading and **is** sandbox-authoritative.
**Controller re-runs AC10 outside the sandbox.**

---

### M4 — Default-lane poison wiring — **THE ONLY CI-TOUCHING COMMIT**

**Estimate**: ~45 LOC, 6.0h (most of it verification and a CI round-trip, not typing).
**Blast radius: MAXIMUM.** This edits `.github/workflows/ci.yml`,
`.github/workflows/build.yml`, `make/test.mk`. A bad edit reds `dev` for every concurrent
session and outranks the mission queue. It is the last code milestone by design, is one commit,
and is revertable in isolation.

**Pre-commit gate — all of these must be green BEFORE the commit lands, in this order:**

| # | Gate | Command | Why |
|---|---|---|---|
| G-a | Rebase onto current `origin/dev` | `git fetch origin && git log --oneline HEAD..origin/dev \| wc -l` → 0 | workflow files are the most contended in the repo |
| G-b | Re-check in-flight workflow PRs | `gh pr list --state open --json number,files --jq '.[] \| select(.files[].path \| test("workflows/")) \| .number'` | #532 and #569 both touch these files today |
| G-c | **Workflow syntax** | `actionlint .github/workflows/ci.yml .github/workflows/build.yml` → rc=0 | `actionlint` is installed; catches the class that reds `dev` before push |
| G-d | Simulate each edited step locally | run the exact `run:` body of each edited step in a shell, with the poison + `GOPROXY=off`, after an unpoisoned `go mod download all` | proves the step's own guard fires and the prefetch is sufficient |
| G-e | **AC12 cold-cache drill, negative direction FIRST** | see task 5 | a drill that has never been seen to fail is not a drill |
| G-f | Post-push bounded CI poll | `gh run list --branch dev --limit 3` polled against a `date +%s` deadline ≤ 30 min | red must be caught and reverted in minutes, not at the next iteration |

**Tasks**
1. **ci.yml — Linux leg** (`test` job): before the `go test` step at `:74`, add an *unpoisoned*
   `go mod download all` step. On the `go test` step add
   `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 NO_PROXY=localhost,127.0.0.1 GOPROXY=off`
   plus an **in-step bash guard** asserting the env in the same `run:` block before invoking
   `go test`. Same for the Windows leg at `:318`, with a **PowerShell** guard
   (`if ($env:HTTP_PROXY -ne 'http://127.0.0.1:9') { Write-Error …; exit 1 }`) — the two legs use
   different shells (`ci.yml:311 shell: pwsh`) and one guard body cannot serve both.
2. **build.yml** — the matrix `Run tests` step at `:65`. **Pin `shell: bash` on that step.**
   `build.yml` declares no `defaults.run.shell` (`grep -n 'shell:\|defaults:' .github/workflows/build.yml`
   → exactly 1 hit, at `:86`), so on `windows-latest` the step currently runs under the runner
   default (pwsh); adding a bash-shaped env guard without pinning the shell is the single
   most likely way this milestone reds `dev`. The matrix has **4** entries
   (ubuntu, macos-amd64, macos-arm64, windows) — one step edit, four legs.
   The existing `go mod download` at `:59` must become `go mod download all` (V32: 247 test-only
   packages that `go mod download` alone does not necessarily pull).
3. **make/test.mk** — poison env on the `$(GOTEST)` line of `test:` **only** (`:17`), leaving the
   `build` prerequisite (`:15`) unpoisoned. Add `go mod download all` as a prerequisite or an
   explicit preceding recipe line.
4. **ci.yml no-silent-skip registration (AC9)** — add `TestGateLint_SelfTest` to the `-run`
   regex, `./internal/testutil/gatelint` to the **package list**, and the name to the assertion
   loop, in **both** steps (Linux `:84`, PowerShell `:327`). *The vacuity trap the doc names is
   real*: a name in the loop without the package in the command yields no `--- PASS:` line and
   fails loudly, which is the safe direction — but the reviewer must still confirm the package
   path is present.
5. **AC12 cold-cache drill — run the FAILING form first.**
   ```bash
   # (i) negative control: no prefetch. MUST FAIL, and the plan records that it did.
   M=$(mktemp -d /Users/voightkampff/dev/sunholo-data/gomodcache.XXXX)
   GOMODCACHE=$M HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 GOPROXY=off \
     go test -count=1 ./internal/testutil/... ; echo "negative-control rc=$? (MUST be non-zero)"
   # (ii) the AC proper: prefetch unpoisoned into the SAME cache, then run poisoned.
   GOMODCACHE=$M go mod download all
   GOMODCACHE=$M HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 GOPROXY=off \
     go test -count=1 $(go list ./... | grep -v /scripts) ; echo "AC12 rc=$? (MUST be 0)"
   rm -rf "$M"
   ```
   Note `mktemp -d` is rooted at a **sibling of the repo, never `/tmp`** — a `/tmp`-rooted path
   fails `TestIsTempPath` (`internal/loader`) and `TestSolve_HardTimeout_FakeSolverIgnoringT`
   (`internal/smt`) for the *location*, producing a red CI can never reproduce.

**Anti-vacuity**

| Assertion | Mutation that must make it RED | Proof the mutation landed |
|---|---|---|
| In-step env guard (each of 3 steps) | delete the `HTTP_PROXY=` line from that step's `env:` | `git diff` on the workflow file, then run the step body locally — it must exit non-zero *before* `go test` starts |
| AC9 PASS registration | rename `TestGateLint_SelfTest` | the no-silent-skip loop reds with `::error::` |
| AC11 greps | `grep -c '127.0.0.1:9' .github/workflows/ci.yml` → ≥2, `build.yml` → ≥1, `make/test.mk` → ≥1; pre-sprint control: all three are **0** today | remove one |
| AC11 lane-crossing guard | `AILANG_LIVE_NET=1 HTTPS_PROXY=http://127.0.0.1:9 go test ./internal/pkg -run TestGitCache_Resolve_RealRepo` → **FAIL** with the named message | change M1's `t.Fatalf` to `t.Skip` → this passes-as-skip, red |
| AC12 | skip the prefetch | the negative control in task 5(i), recorded |

**Acceptance (M4)**
```bash
actionlint .github/workflows/ci.yml .github/workflows/build.yml     # rc=0
grep -c '127.0.0.1:9' .github/workflows/ci.yml        # → ≥2
grep -c '127.0.0.1:9' .github/workflows/build.yml     # → ≥1
grep -c '127.0.0.1:9' make/test.mk                    # → ≥1
grep -c 'internal/testutil/gatelint' .github/workflows/ci.yml   # → ≥2 (package path in BOTH no-silent-skip steps)
grep -c 'TestGateLint_SelfTest' .github/workflows/ci.yml         # → ≥4 (regex + loop, both legs)
AILANG_LIVE_NET=1 HTTPS_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/pkg -run TestGitCache_Resolve_RealRepo   # MUST FAIL, named message
make test                                              # rc=0 under the newly-poisoned line
# AC12 drill: both directions, per task 5
# then push and poll:
gh run list --branch dev --limit 3
```
Closes **AC9, AC11, AC12**, and the "all CI legs green" criterion (**6** legs, not 5 — finding R-D).

**UNINFORMATIVE UNDER SANDBOX**: essentially all of it. Workflow behavior is only observable
via a real GitHub Actions run; `make test` and the AC12 drill spawn subprocesses and need module
fetch. **The controller must run every M4 gate outside the sandbox and must personally watch the
first `dev` CI run after this commit**, with a revert command staged:
`git revert --no-edit <M4-sha>`.

---

### M5 — Docs, CHANGELOG, AC sweep, issue closes

**Estimate**: ~140 LOC, 3.0h. **Blast radius: none.**

**Tasks**
1. `changelogs/v0.18-current.md` — a `[v0.33.1]` section. Must state the **local behavior
   change**: `go test -short ./...` no longer skips anything; and that `make test` now runs behind
   a poisoned proxy.
2. `docs/docs/guides/development-workflow.md` — the one-idiom convention, `AILANG_LIVE_NET`, how
   to write a live test, how to satisfy/extend gatelint, the poisoned default lanes.
3. **Restate the boundary claim honestly (this is a correction, not a doc chore).** The doc's
   Goals paragraph says the default lanes deny HTTP(S) egress as a *boundary*. Per finding R-B
   that is true only for clients using Go's **default** transport, and for `git`. Every
   first-party `http.Transport{…}` literal (**6 of them, in 4 files, 0 setting
   `ProxyFromEnvironment`**) bypasses it — including `internal/effects/net.go`, the `Net` effect
   itself. The guide and the design doc must both say so, and the reviewer instruction becomes:
   *flag by hand any new test that (a) dials raw TCP/SSH, **or (b) constructs its own
   `http.Transport`**.*
4. Run the full AC1–AC12 sweep (as corrected) and paste every output into the implementation
   report — including the pre-sprint controls recorded in §1 of this plan, so each AC is shown
   non-vacuous against a known positive rather than assumed.
5. Draft closing comments for #583, #494, #509, #587, #561 referencing the ACs. **The controller
   commits and closes**, not the executor.

**Acceptance (M5)**
```bash
grep -c 'v0.33.1' changelogs/v0.18-current.md              # → ≥1
grep -ci 'AILANG_LIVE_NET' docs/docs/guides/development-workflow.md   # → ≥1
grep -ci 'http.Transport' docs/docs/guides/development-workflow.md    # → ≥1 (the R-B residual is documented)
make verify-examples && make check-file-sizes
go test -count=1 $(go list ./... | grep -v /scripts)       # rc=0
```

---

## 4. Estimate verdict — the doc's 3–4 days is low

**Revised: 26h ≈ 4.5 executor-days.** Four reasons, in order of size:

1. **M4 is not a 0.5-day task.** The doc folds workflow wiring into a 1-day M3 alongside a
   ~460-LOC linter. But M4's cost is not LOC — it is six sequential verification gates (G-a..G-f),
   two different shell dialects for the same guard, a `shell: bash` pin on a 4-leg matrix step
   whose current Windows behavior nobody has checked, an AC12 cold-cache drill that must be run
   **twice** (negative control, then the AC), and a CI round-trip. Budget 6h and expect to use it.
2. **The doc's own milestone split is not bisectable.** Splitting M3 into gatelint (pure Go) and
   poison wiring (workflows) is required by the sprint's hard constraint and adds a commit
   boundary, a verification pass, and a green-suite run.
3. **M2 grew.** Three additions the doc did not scope: the `AllowHTTP`/`AllowLocalhost` discovery
   for the `httptest` coverage (finding 1.4.3), the third unbounded helper at
   `main_test.go:26` (1.4.4), and a probable manual merge against PR #532 (1.4.1).
4. **Calibration against recent same-shape sprints in this repo**: `M-RECORDED-STREAM-API-S1`
   730 LOC / 3.75d; `M-EVAL-STANDARD-CONFIDENCE-GATING` 460 LOC / 4d;
   `M-MCP-EXACT-TOOL-SURFACE-LANE-B` planned 1450 LOC / 3.5d after its planner revised the doc's
   17h to 25h. Recent raw velocity (`git diff --stat` over 7 days, `*.go` only) is
   `152 files changed, 11541 insertions(+), 1495 deletions(-)` — high, but that is aggregate
   across concurrent agents, not one executor's serial throughput. ~280 LOC/day of
   *test-infrastructure with mandatory falsification drills* is the honest per-executor rate here;
   1265 LOC / 280 ≈ 4.5 days.

**Do not cut ACs to hit 4 days.** If a hard box is needed, the only clean cut is deferring **M5's
doc work** to a follow-up (−3h), accepting that the `-short` behavior change ships undocumented
for a few days. Cutting M4's gates instead is how `dev` goes red.

---

## 5. Sandbox caveat (for whichever executor runs this)

Gate verdicts produced inside a `workspace-write` sandbox are **not evidence** for anything
touching sockets, `httptest`, subprocesses, or the network — and this sprint is almost entirely
that. Steps explicitly marked **UNINFORMATIVE UNDER SANDBOX** above:

| Milestone | Sandbox-uninformative | Sandbox-authoritative |
|---|---|---|
| M1 | `TestRunBounded_*` (subprocess) | `TestHangGuard_*`, `TestLiveNetworkStatus_*` (pure) |
| M2 | all of `cmd/ailang`, `internal/eval_harness`, the new `httptest` subtests, AC3b/AC3c/AC4 | the AC1/AC2/AC6/AC7 greps |
| M3 | `TestEgressPosture` (a)(b)(c) | `TestGateLint_SelfTest`, `TestGateLint_Repo` (file reads only) |
| M4 | everything | nothing |
| M5 | `go test ./...` | the greps, `make check-file-sizes` |

The executor **must not report a pass/fail verdict** on an uninformative step; it flags it, and
the controller re-runs it outside the sandbox. Measured cost of the controller re-runs in a
Claude-agent session at plan time: full poisoned suite ≈ 12 min; `./internal/effects` ≈ 1s;
`./internal/pkg` ≈ 3s. AC12's drill adds ~25 min. All affordable.

---

## 6. Decisions the controller must carry (not the executor's to make)

1. **PR #532 vs M2.** #532 is `CONFLICTING`/`DIRTY`, open since 2026-07-29, and rewrites
   `buildAilang` — the exact helper M2 wraps — for a legitimate reason (14 redundant binary
   builds pushing `cmd/ailang` over the Windows 300s ceiling; that is *also* a CI-flake fix).
   Three options: (a) land #532 first, then M2 wraps the `sync.Once` body in `HangGuardContext`
   (cleanest, but #532 needs a conflict resolution); (b) M2 proceeds and #532 is rebased onto it
   afterwards; (c) M2 absorbs #532's `sync.Once` change. **My recommendation: (a).** #532's fix
   is orthogonal and strictly reduces the surface M2 has to bound. **This is a Mark/controller
   call, not an executor call.**
2. **The R-B hole: does `internal/effects/net.go` get `Proxy: http.ProxyFromEnvironment`?**
   Doing so would make the poison a genuine boundary for the repo's own HTTP client — but it is a
   **production runtime behavior change** (the doc disclaims any) and probably breaks the pinned-IP
   SSRF guard, since a proxy resolves the hostname itself. **My recommendation: do NOT do it in
   this sprint.** Document the hole (M5 task 3), and open a follow-up design item.
3. **Should gatelint gain an R4 for unbounded `exec.Command` in `*_test.go`?** 62 first-party
   test files have one; the sprint bounds 3. Class C3's *generator* otherwise survives, which is
   the pattern CLAUDE.md §3 exists to prevent. **My recommendation: not in this sprint** (an R4
   with a 62-entry seed allowlist is a sprint of its own) — but record it as a named Future Work
   item with the measured count, so it is not rediscovered as a surprise.
4. **The doc's Goals paragraph and V22/V30/leg-count need correcting** (findings R-A, R-C, R-D).
   Whether that is an edit to the design doc before execution or a note in the implementation
   report is the controller's call. Leaving V22 uncorrected risks an executor "verifying" AC3 by
   running a command that passes for the wrong reason.

---

## 7. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| M4 reds `dev` for all concurrent sessions | **High** | `actionlint` pre-commit (G-c), local step simulation (G-d), single isolated commit, staged `git revert`, bounded post-push CI poll ≤30 min (G-f) |
| build.yml matrix step's guard breaks on `windows-latest` | High | pin `shell: bash` on that step (M4 task 2); it is the most likely single failure |
| PR #532 / #569 merge under the sprint | Med | G-a/G-b re-check before M2 and before M4; §6.1 decision taken up front |
| gatelint FP reds all 6 legs at once | Med | measured-zero FP surface today (V16/V23/V24 CONFIRMED); allowlist-with-reason escape; run against the real tree immediately before committing M3; keep M2→M3 gap short |
| A concurrent session adds a `testing.Short()` between M2 and M3 | Med | re-run `TestGateLint_Repo` at M3 commit time, not at M3 write time |
| Executor "verifies" AC3 with the doc's vacuous command | Med | AC3 replaced by AC3a/b/c in this plan with pre-sprint controls quoted; §6.4 |
| `httptest` coverage blocked by `AllowLocalhost:false` and burns an hour | Low-Med | named in 1.4.3 with the exact two fields and file:line |
| AC12 drill is skipped as "obviously fine" | Med | negative control must be run and recorded FIRST (M4 task 5(i)) |
| Worktree under `/tmp` | Med | worktree pinned to `/Users/voightkampff/dev/sunholo-data/.wt-iter142-ci-flake` (verified free at plan time; 14 worktrees exist); `mktemp -d` in AC12 also rooted at a sibling |
| Sprint JSON silently not committed | Low | `.ailang/` is gitignored (`.gitignore:77`) but 45 sprint JSONs are tracked — use `git add -f` |

---

## 8. Success metrics

- `testing.Short()` in first-party `*_test.go`: **7 → 0** (pre-sprint control measured)
- `Getenv("CI")`/`Getenv("GITHUB_ACTIONS")` in first-party `*_test.go`: **2 → 0**
- Poisoned full suite: **1 FAIL → 0 FAIL** (pre-sprint baseline measured: 105 ok / 1 FAIL / 30 no-test-files)
- Absolute wall-clock hang-guards at the four cited sites: **replaced by deadline-derived bounds**
- Every new assertion has a named mutation, and every mutation drill has a landed-proof command
- All **6** CI legs green (ci.yml test + test-windows; build.yml × 4 matrix entries)
- The residual is *documented*, not claimed away: raw TCP/SSH (V27, CONFIRMED) **and** first-party
  custom `http.Transport` (finding R-B, new)

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_33_1/m-ci-flake-systemic-fix-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-CI-FLAKE-SYSTEMIC-FIX.json`
