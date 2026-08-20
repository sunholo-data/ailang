# Sprint plan — M-LIST-REPR-SPIKE (LC-1: list-representation benchmark spike)

**Design doc (AUTHORITY):** [`design_docs/planned/m-list-repr-spike.md`](m-list-repr-spike.md) — 495 lines, quorum-resolved, **not modified by this plan**.
The design doc WINS wherever it and this plan disagree. This plan adds only *how*, *in what order*, and *under which gates*, plus the first-party measurements taken to choose among the things the doc leaves open. **No threshold, observable, control, or clause of the kill criterion is restated loosely, relaxed, or dropped here** — every numeric literal (1.5, ≥8, 2.0, 2.5, 1.2, five trials, median-of-paired-ratios, five-more rerun) is referenced by pointer to the doc, not paraphrased into a gate.

**Programme position:** LC-1 of [`m-list-cons-cells-decomposition.md`](m-list-cons-cells-decomposition.md). This spike carries the **go/no-go kill criterion** for the remaining ~16 person-days (LC-2…LC-5). A false GO costs 16 days; a false STOP costs the programme. That asymmetry is why every gate below is a *measurement with a firing control*, not an assertion.

**Created:** 2026-08-20 (V1 mission iteration 236, sprint-planner stage)
**Branch / worktree:** `sprint/iter236-list-repr-spike` @ `/Users/voightkampff/dev/sunholo-data/.wt-iter236`
**Planned at HEAD:** `3ad6a2e46` (design-doc commit); **merge base with `origin/dev` AT PLANNING TIME: `8322d22b75adfce7a4aa284eaf3ad99afdd4b570`.** ⚠ This is a HISTORICAL record of where planning happened — it is **NOT** the base AC-7 should use, and it was 6 commits stale by iteration 238. AC-7 derives its base with `git merge-base origin/dev HEAD` (see §6 step 7).
**Milestones:** 6 · **Estimated:** ~1,800 Go LOC + ~450 lines of report markdown · **3.0 days** (top of the doc's 2–3-day box — see §7)
**Risk level:** MEDIUM · **Planner lane:** opus-required (doc line 10)
**Target:** v0.34.0

---

## 1. First-party verification log (this planning session)

Every row below was measured in this worktree at `3ad6a2e46` by the planner. Rows the *controller* measured are labelled `[C]` and were **not** re-run; everything else is `[P]` (planner, first-party). Negative claims carry a same-scope firing control, per the mission's "an empty search is a claim" rule.

| # | Claim | Command | Observed |
|---|---|---|---|
| P1 | Toolchain and platform of every number this sprint will produce | `go version`; `uname -sm`; `sw_vers` | `go1.26.6 darwin/arm64`; `Darwin arm64`; macOS 26.5.2 (25F84) |
| P2 | A package under `tools/internal/` **can** import `internal/eval` and **can** use `iter.Seq` — the design's two load-bearing compile-time premises | built a throwaway `tools/internal/zzprobe` with `head eval.Value; tail *cell; n int`, an `iter.Seq[eval.Value]` iterator, and real `&eval.IntValue{Value:1}` / `&eval.ListValue{Elements:…}` literals; `go vet ./tools/internal/zzprobe/` | **rc=0**, zero diagnostics. Probe removed; `git status --short` empty afterwards |
| P3 | AC-7's pathspec is a **valid, non-vacuous** instrument for this layout | with an uncommitted file at `tools/internal/zzprobe/p.go` staged with `git add -N`: `git diff --name-only HEAD -- internal/ cmd/ std/ examples/`; control `git diff --name-only HEAD` | scoped → **0 lines** (correct: `tools/internal/` is not selected); unscoped control → **`tools/internal/zzprobe/p.go`**, i.e. the instrument *does* see the file it must not report. **Consequence: AC-7's scoped command is green both when the spike is correct AND when nothing changed at all — see §4 M6 for the mandatory firing control** |
| P4 | CI's exposure to the spike is **wider than the design doc's Conflict Surface #2 states** | read `.github/workflows/ci.yml` (`:17-18`, `:101`, `:297-298`, `:377`, `:441-464`); `make/code-health.mk:14-20`, `:42-44`, `:68-71`; `make/test.mk:27-30` | CI `test` job (**ubuntu-latest**) runs `go test -timeout 300s ./...` at `:101`; CI `test-windows` job (**windows-latest**) runs the same at `:377`. `make vet` = `go vet $(go list ./... \| grep -v examples/agents)` → **includes** `tools/`. `make fmt-check` = `gofmt -l .` → **includes** `tools/`. `make test` (test.mk:30) runs under `GOPROXY=off` + poisoned `HTTP(S)_PROXY`, i.e. **offline** |
| P5 | `make lint` (golangci-lint) does **not** cover `tools/` (negative claim + firing control) | `make/code-health.mk:71` → `golangci-lint run ./cmd/... ./internal/... ./testutil/...`; control: `.golangci.yml:32-34` `exclude-dirs` lists only `examples/agents`, `scripts` | The **invocation scope**, not the config, is what excludes `tools/`. So a `tools/`-only lint smell cannot red CI — but `go vet` and `gofmt` still can |
| P6 | An **external-only** `_test` package is excluded from `test-coverage-gate`'s package selection | built `tools/internal/zzcov` with only a `package zzcov_test` file; `go list -f '{{.ImportPath}} TestGoFiles={{.TestGoFiles}} XTestGoFiles={{.XTestGoFiles}}'`; then `go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... \| grep -c zzcov`; control `… \| grep -c '.'` | `TestGoFiles=[] XTestGoFiles=[z_ext_test.go]`; selection hit count **0**; control **107** packages selected. `coverage.mk:18` uses exactly that `{{if .TestGoFiles}}` filter, so keeping **all** spike test files external (which clause (e) already forces for benchmarks) keeps the spike out of the 29% threshold denominator (`coverage.mk:12`) |
| P7 | `tools/` packages are in the module and are enumerated by `go list ./...` | `go list ./... \| grep '/tools/' \| wc -l` | **7** (`build-snapshot`, `eval-elo`, `eval-regrade`, `gen-error-codes`, `govulncheck-filter`, `ollama-tap`, `verify-model-pricing`). So a new `tools/internal/spike-listrep` is picked up by `go list ./...`-driven targets automatically |
| P8 | `rig.lock` is a **GPU** mutex, not a general rig mutex | `tools/launchd/rig-lock.sh:16` → `RIG_LOCK_DIR="${RIG_LOCK_DIR:-$HOME/.ailang/state/rig.lock.d}"`; `tools/launchd/mission-control.sh:15` → *"Iterations are cloud-model work: they NEVER take rig.lock (GPU mutex only…)"*; `.claude/skills/mission-control/SKILL.md:445` → *"default iterations never touch `rig.lock` — it is a GPU mutex only"* | Confirmed. See §3 for this sprint's posture |
| P9 | Recent per-commit Go LOC band (velocity basis) | `git log --since="14 days ago" --no-merges` × `git show --numstat -- '*.go'`, summing added+deleted | 11 code commits: `136+1`, `220+0`, `349+2`, `491+12`, `499+7`, `539+18`, `624+37`, `635+74`, `815+198`, `1216+70`, `1515+181`. **Band 137–1,696 lines changed/commit, median ≈ 561.** All six milestones below (140–420) sit inside the band |
| P10 | `bench-phase2a`'s regex cannot collide with this spike's benchmark names | `make/eval.mk:169-175` | `-bench='Benchmark(Native\|Eval)_'` **scoped to `./internal/eval/`**. The spike's `BenchmarkListRep_*` names in `./tools/internal/spike-listrep/` are unreachable from that target on both axes |
| C1 | `[C]` A nested `tools/internal/` package is **compiler-refused** to production importers | controller, this session: negative `go build ./internal/spikeprobe_consumer/` → **rc=1**, `use of internal package …/tools/internal/spikeprobe not allowed`; positive control `go build ./tools/scratchprobe_ok/` → **rc=0**, zero output (exit codes captured to file, not through a pipe) | Not re-run by the planner |
| C2 | `[C]` `make test` compiles + unit-tests the spike; benchmarks compile but do not execute | controller: `make/test.mk:30` is literally `$(GOTEST) -v $$($(GOCMD) list ./... \| grep -v /scripts \| grep -v /examples/agents)` | Not re-run (P4 independently corroborates the CI-side form) |
| C3 | `[C]` `check-file-sizes` walks only root-level `internal`/`cmd` | controller: `make/code-health.mk:125` `for file in $$(find internal cmd -name "*.go")` | Spike is outside the 800-line gate. Repo hygiene target of <500 lines/file still applies by choice |
| C4 | `[C]` `*ListValue` has exactly 2 methods repo-wide (`Type()` :88, `String()` :89; control `*ArrayValue` = 4); 2 free functions (`encodeJSONArray` :193, `encodeJSONObject` :220) | controller | Consumed as given by D3's table |
| C5 | `[C]` `go.mod` declares `go 1.26.6`; `iter.Seq` has **0** in-repo uses | controller | P2 independently confirms `iter.Seq` compiles here |
| C6 | `[C]` `_list_nth` bounds-checks at `internal/builtins/list.go:320`; patterns index at `internal/eval/eval_patterns.go:224`, `:244` | controller | Consumed as given by D3's table |

---

## 2. Where the design doc's acceptance criteria are ambiguous

Per the mission's standing defect ("plan/doc AC mismatch"), these are stated rather than silently resolved. **In every case the plan takes the stricter reading, and no threshold moves.** If the evaluator disagrees with a resolution, the doc wins and the plan is wrong — that is the intended failure mode.

**AMB-1 — AC-1's word "cell" is undefined at grid granularity.** AC-1 (doc:320-322) demands "every cell of B1–B8 × {C0, C1, C2(8), C2(32)}", but B1 is a 2×3 grid (doc:137), B2 has 4 sizes (:138), B3 has 2 (:139), B4 has 4 indices (:140). "Cell" could mean *row × arm* (32 cells) or *grid point × arm* (76). **Resolved: the strict reading — one cell = one (grid point × arm) pair.** Full enumeration in §5. This is the more expensive reading; taking the cheap one would let a passing report omit 44 measured points.

**AMB-2 — clause (d)'s observable has no row in the B-matrix.** Clause (d) (doc:218-223) and AC-3 (:328-331) require `Len()` measured "at n=4096 and n=65536" with ratio ≤ 1.2. The matrix rows B1–B8 (doc:135-144) contain **no `Len` row** — B4 is `nth` at an index, which is a different operation. **Resolved: this plan adds a plan-level benchmark `B-LEN`** (M4) supplying exactly that observable. It is *additive*: AC-1 still ranges over B1–B8 only, and the 1.2 threshold is untouched. **This is a genuine gap in the design doc**, not a plan invention — AC-3 cannot be satisfied without it.

**AMB-3 — no verdict is defined for an invalid instrument.** AC-2 (doc:323-325) reds if the C0 control fails to reach an L-ratio ≥ 8 ("instrument can't see a positive → B1 is invalid"), but AC-5 (:336-339) admits **only** GO or STOP and forbids hedging. A doc-literal reading would force a STOP on an instrument that has just been shown blind. **Resolved: control failure ⇒ NO VERDICT MAY BE EMITTED.** The sprint fails at M2 (Day 1), AC-2 is red, and D-19 is *not* re-opened — a STOP posted to #745 must never rest on a benchmark that could not see its own known positive. This is the reason M2 exists as a separate Day-1 milestone rather than being folded into the matrix run.

**AMB-4 — `-benchtime` is unpinned.** Protocol §1 (doc:161-165) pins `-count=1`, `-timeout=10m`, and a 12-minute subprocess deadline, but never `-benchtime`. Two invocations of "the same command" can then differ in `b.N`, changing variance (though not the ns/op normalisation). **Resolved: pin `-benchtime=1s` explicitly** (Go's default, made visible) on every cell, recorded verbatim in every matrix cell's command string per AC-1. Any per-cell deviation (see the B1 RSS risk, §7 R2) must be recorded with its reason in the report. No threshold changes; this only makes the recorded command reproducible.

**AMB-5 — clause (e)(3)'s "in one file" is singular across three representations.** Doc:233-235 requires "all field writes sit inside constructor functions in one file". Three candidates cannot share one file without merging their representations. **Resolved: read per-representation** — each candidate's field writes are confined to that candidate's own constructor file (`slicelist.go`, `conslist.go`, `chunklist.go`). The grep AC-4 demands is run once across the whole package and its hits partitioned by file.

**AMB-6 — AC-6 is already largely satisfied by the design doc itself.** D3's table with a `file:line` per row already exists at doc:248-257. **Resolved: the report reproduces that table as-built**, annotated per row with whether the spike's candidates actually implemented it and any operation clause (e) proved infeasible. No new citations are invented; if an operation is dropped, the row stays with the drop recorded.

**AMB-7 — AC-8 names `make test`, but CI's real surface is three gates wider and one OS wider (P4, P5).** CI runs bare `go test -timeout 300s ./...` on **ubuntu-latest** *and* **windows-latest**, plus `make vet` and `make fmt-check`, all of which include `tools/`. `make lint` does not. **Resolved: this plan ADDS gates** (§4 M1/M6) — it does not reinterpret AC-8, which is satisfied by `make test` alone. Two consequences the executor must honour: the spike's **unit tests must be OS-independent** (they run on windows-latest), and its per-package test time must stay **far below 300s**.

---

## 3. Execution environment and measurement posture

**Platform of record.** Every number in the report is `darwin/arm64`, `go1.26.6`, macOS 26.5.2 (P1). The report header records `go version` and machine identity, per doc:153-154. The ubuntu/windows CI legs **compile** the spike but never run its benchmarks (P4, C2) — no cross-platform number is claimed.

**GPU: not used, and the lock is not taken.** The spike is pure Go CPU + heap work. It never loads a model, never calls ollama, and never touches the GPU. **It therefore does NOT take `rig.lock` and MUST NOT take it** — `rig.lock` is a *GPU mutex only* (P8: `tools/launchd/rig-lock.sh:16`, `mission-control.sh:15`, `mission-control/SKILL.md:445`), and holding it for a ~1-hour benchmark sweep would needlessly block the GPU eval consumers. A later executor must not infer a lock requirement from the phrase "runs on the rig".

**But quietness is a measurement-validity fact, so it is recorded rather than enforced.** A concurrent GPU eval contends for memory bandwidth and cores and will inflate variance. The executor therefore **observes and records, without acquiring**:

```bash
ls "$HOME/.ailang/state/rig.lock.d" >/dev/null 2>&1 && echo "rig.lock HELD" || echo "rig.lock free"
```

taken immediately **before and after** the full matrix run, both lines pasted into the report header. If either reads HELD, the report says so; the five-trial / median-of-paired-ratios protocol (doc:161-183) is precisely the defence against that contention, and paired ordinal comparison means contention hits candidate and control alike.

**Wall clock, not GPU hours, is the budget.** §5 enumerates 76 measured points × 5 trials = **380 fresh-process invocations** per full pass, plus any predeclared tie/spread reruns. At `-benchtime=1s` plus process start, a full pass is estimated at **~30–90 minutes**. The executor records the actual elapsed wall-clock of the run in the report; that number, not an estimate, is what LC-4 will plan against.

---

## 4. Milestones

Standing rules for all six: **one bisectable commit per milestone**; `gofmt` clean; `go vet ./tools/...` rc=0; `go test -count=1 ./tools/internal/spike-listrep/...` rc=0; no file over 500 lines; every commit message carries `refs #676` (never `Fixes` — #676 is not fixed by a spike) and, for the final commit only, a pointer to the verdict.

---

### M1 — Package scaffold, D3-shaped API, C0 control, C1 cons cells
**Closes:** AC-4 legs 1–2 (of 3) for C0/C1 · **first checkpoint of** AC-7, AC-8
**Estimated:** ~400 LOC (≈250 impl + ≈150 smoke tests) · **~0.6 day**
**Depends on:** nothing

Create `tools/internal/spike-listrep/` (absent today — C-verified, and P7 shows `go list ./...` will pick it up immediately):

| File | Contents |
|---|---|
| `README.md` | `THROWAWAY — DO NOT IMPORT` banner; the compiler-boundary rationale (doc:52-57); the `listConsImpl` mirror quoted **next to** the original `internal/builtins/list.go:98-105` so drift is visible (doc:68-70) |
| `list.go` | The D3 surface (doc:248-257) as a Go interface: `Len() int`, `At(int) (eval.Value, bool)`, `All() iter.Seq[eval.Value]`, `ToSlice() []eval.Value`, `Uncons() (eval.Value, List, bool)`, `DropPrefix(int) List`; package-level `FromSlice`, `Empty`, `Cons` per arm |
| `slicelist.go` | **C0 control** — wraps the literal production `*eval.ListValue` (`internal/eval/value.go:84`), cons mirrored locally per doc:68-70 (no `EffContext` plumbing) |
| `conslist.go` | **C1** — `type ConsList struct{ head eval.Value; tail *ConsList; n int }`, all fields unexported, **all field writes inside constructors in this file** (AMB-5) |
| `spikelistrep_test.go` | `package spikelistrep_test` — smoke round-trips for C0 and C1 (build → `Len` → `All` → `ToSlice` → `Uncons` → `DropPrefix` → `At` in and out of bounds) |

**Design decisions taken here (plan-level, doc-compatible):**
- `iter.Seq` is used for `All()` — the doc's explicit "adopt, do not invent" ruling (doc:122). P2 confirms it compiles under `go1.26.6` in this exact position. This is the repo's **first** `iter.Seq` use (C5); the report says so.
- **All** test files live in the external `spikelistrep_test` package, never in-package. Clause (e) forces this for benchmarks; extending it to the smoke tests additionally keeps the spike out of `test-coverage-gate`'s denominator (P6, with control). The executor must not "helpfully" add an in-package test file.
- Element payloads are real `*eval.IntValue` / `*eval.StringValue` so the 16-byte interface headers are faithful in the memory accounting (doc:60-62; P2 confirms the literal forms).

**Acceptance criteria (all commands, all with rc captured):**
- `go build ./tools/internal/spike-listrep/` rc=0
- `go vet ./tools/...` rc=0 · `gofmt -l tools/internal/spike-listrep` prints nothing, **with the firing control** `gofmt -l .` run at the same time and shown to be capable of printing (i.e. it is the same instrument that gates CI, P4)
- `go test -count=1 ./tools/internal/spike-listrep/...` rc=0, and the output shows `--- PASS:` lines (not a vacuous zero-test pass)
- `go list -f '{{.TestGoFiles}} {{.XTestGoFiles}}' ./tools/internal/spike-listrep` shows `[]` then a non-empty list (P6's property holds as built)
- **`make test` green end-to-end** — this is the milestone where the spike enters CI, and the cheapest place to catch it (AC-8, first checkpoint). Note it runs offline (`GOPROXY=off`, P4); the spike must import only stdlib + `internal/eval`.
- **A negative-arm re-proof of the boundary, first-party:** a scratch `internal/` consumer importing the spike fails to build with `use of internal package … not allowed`, exit code **captured to a file, not read through a pipe**; scratch files deleted in the same milestone. (C1 measured this for a probe package; M1 re-proves it for the *real* package name.)

**Risks:** low. `internal/eval`'s import graph is stable and the package imports nothing that moves (doc:276).

---

### M2 — B1 (branching prepend) + B2 (linear build), and the control must fire first
**Closes:** AC-2's **control leg** · **contributes to** AC-1 (B1, B2 columns)
**Estimated:** ~210 LOC · **~0.4 day**
**Depends on:** M1

This milestone exists on Day 1, before C2 is written and before any candidate is trusted, because **AC-2 makes the instrument itself falsifiable**: if C0 does not show an L-ratio ≥ 8, B1 cannot see the failure it gates on and no verdict may be emitted at all (AMB-3).

- `branching_bench_test.go` — **B1** (doc:137): `m` prepends each onto **ONE retained base** of length `L`; **all `m` results kept live** with `runtime.KeepAlive` on the result slice. Grid `m ∈ {1024, 4096} × L ∈ {1024, 4096, 16384}`.
- `linear_bench_test.go` — **B2** (doc:138): fold `n` prepends each onto the newest list, `n ∈ {1600, 3200, 6400, 12800}` (the #676 repro ladder, V14).
- Both `-benchmem`, so **B7** (doc:144) is read off these two rows with no extra runs.

**Naming, so the runner can select exactly one cell** (protocol §1 requires "exactly one selected cell", doc:161-163). Table-driven `b.Run` subtests, not one function per grid point — the doc's `BenchmarkListRep_BranchingPrepend_Cons_m4096_L16384` (doc:306) is explicitly a *sketch* (doc:291), and subtests give the same anchored selectability at a fraction of the LOC:

```
BenchmarkListRep_B1_Branching/arm=C0/m=4096/L=16384
  -bench='^BenchmarkListRep_B1_Branching$/^arm=C0$/^m=4096$/^L=16384$'
```

The `BenchmarkListRep_` prefix cannot collide with `bench-phase2a` (P10: different regex **and** different package scope).

**Acceptance criteria:**
- Every B1 and B2 subtest name resolves to **exactly one** benchmark under its anchored regex (proved by running each regex and counting the reported benchmark lines = 1)
- **AC-2 control leg:** C0's `time(m, L=16384) ÷ time(m, L=1024) ≥ 8` at **both** `m ∈ {1024, 4096}`, measured under the full five-trial protocol once M5 lands, and provisionally here under a labelled dev-loop run
- **STOP CONDITION:** if the control leg fails, the sprint halts at M2 and reports AC-2 red. No verdict, no #745 comment (AMB-3).
- `go test -count=1 ./tools/internal/spike-listrep/...` still rc=0 and fast (benchmarks compile but do not run without `-bench` — C2)

**Risks:** see §7 R2 (B1's heaviest cell retains ~1.07 GB per iteration).

---

### M3 — C2: chunked/unrolled cons with the K ∈ {8, 32} sweep
**Closes:** nothing on its own · **extends** AC-4 legs 1–2 to C2 · **contributes to** AC-1 (C2(8), C2(32) columns)
**Estimated:** ~300 LOC (≈210 impl + ≈90 smoke tests) · **~0.4 day**
**Depends on:** M1 (API), M2 (so the C2 columns drop straight into existing benchmarks)

`chunklist.go` per doc:93-100: nodes hold up to `K` elements plus a tail pointer and cached total length; elements fill leftward; a prepend into a chunk with front slack is a slot write; on contention or a full chunk the prepender copies **at most one chunk** (≤ K) into a fresh node, i.e. O(K) worst case. `K` is a construction parameter, swept at 8 and 32 — the doc is explicit that this is a **parameter study of C2, not a third candidate** (doc:98-100).

**The doc's own escape hatch is adopted verbatim, not invented here:** "C2's shared-chunk prepend is implementable at O(K) worst-case without locks" is listed under *Unverified / needs measurement* with the disposition "failure ⇒ C2 columns marked infeasible (which the kill criterion tolerates if C1 passes)" (doc:419). If the executor cannot land it inside this milestone's budget, C2's columns are marked **infeasible with the reason recorded**, and the sprint proceeds — this is the primary descope lever (§7).

> **⚠ CONTROLLER ADJUDICATION, iteration 237 — THE DESCOPE LEVER WAS PULLED AND IS NOT ACCEPTED.
> A LATER EXECUTOR MUST NOT RE-PULL IT ON THE SAME GROUND.** Iteration 237's executor marked
> `C2(K=8)` and `C2(K=32)` infeasible, reasoning that the immutable `List` API gives no way to
> tell whether a chunk is uniquely owned, so a slack write can mutate a retained alias, and that
> *"pessimistically copying the leading chunk on every prepend removes the specified uncontended
> slot-write behavior."* The first half is correct; the conclusion does not follow, on two
> independent grounds, and the C2 commit was therefore **not landed**:
>
> 1. **The reason bears on the FAST PATH, not on any kill clause.** The doc's C2 bound is already
>    the contended one — *"the prepender copies **at most one chunk** (≤ K elements) into a fresh
>    node — O(K) = O(1) worst-case with a constant"* (doc:93-100). Copying unconditionally, with no
>    ownership detection at all, meets that bound by construction: K is a constant, so O(K) is
>    O(1) worst-case, which is what clause (a) tests. The uncontended slot write is an
>    average-case optimisation the kill criterion never measures. Both benefits the doc names —
>    *"recovering slice-like locality within chunks and amortizing per-element overhead across the
>    chunk"* — survive always-copy intact: per-element overhead is `16 + 16/K` B, i.e. **16.5 B at
>    K=32**, matching the doc's own *"~16-18 B/element for C2"* estimate (doc:417).
> 2. **The doc's tolerance is CONDITIONAL, and the condition cannot yet be evaluated.** doc:419
>    reads *"failure ⇒ C2 columns marked infeasible (**which the kill criterion tolerates if C1
>    passes**)"*. C1's clause (b) and (c) numbers are B3/B6, which are **M4** work and do not
>    exist. The lever was pulled before the predicate that licenses it could be read.
>
> This matters because C2 is the candidate designed to pass exactly where C1 is most at risk:
> clause (b) (iteration ≤ 2.0× at n=65536, where cons-cell pointer-chasing hurts most, and which
> chunking exists to fix) and clause (c) (per-element memory ≤ 2.5×, where the doc estimates
> C1 at 32 B/cell against C2's ~16-18 B/element). Dropping C2 before either is measured makes a
> STOP verdict reachable **by descope rather than by measurement**, on a gate that commits or
> cancels ~16 person-days.
>
> **Revised instruction for M3.** Implement C2 with the **always-copy leading chunk** prepend —
> no ownership tracking, no reference counting, no atomics, no locks — and measure it. That is a
> straightforward, obviously-correct persistent structure. If it then fails a clause, it fails on
> a **number**. The infeasibility lever remains available only for a reason that bears on the
> O(K) bound itself, and only once C1's clause-(b)/(c) numbers exist to make doc:419's condition
> readable.

**Acceptance criteria:**
- C2 satisfies the same external-API smoke round-trip as C0/C1, at **both** K values
- A **shared-chunk contention test** exists and passes: two lists share a chunk, both prepend, and both observe correct independent contents (this is the correctness claim the O(K) argument rests on)
- Fields unexported; all field writes inside `chunklist.go`'s constructors (AMB-5)
- If marked infeasible: the reason is committed in `README.md` in this same commit, and the C2 columns are annotated infeasible everywhere they appear

**Risks:** highest-uncertainty implementation in the sprint (§7 R1).

---

### M4 — Remaining benchmarks: B3, B4, B5, B-LEN, plus the B6 and B8 measurement programs
**Closes:** nothing on its own · **contributes to** AC-1 (B3–B8) and supplies clause (d)'s missing observable
**Estimated:** ~330 LOC · **~0.35 day**
**Depends on:** M1, M3

- `iter_bench_test.go` — **B3** (doc:139): sum `n` int elements through each arm's `All()`; `n ∈ {4096, 65536}`; metric ns/element.
- `nth_bench_test.go` — **B4** (doc:140): `i ∈ {0, n/4, n/2, n−1}` at `n=4096`. Informational, **not gated** (doc:140 says "measured, not gated").
- `materialize_bench_test.go` — **B5** (doc:141): full `ToSlice` copy-out at `n=4096`.
- `len_bench_test.go` — **B-LEN**: `Len()` at `n ∈ {4096, 65536}`. **Additive, per AMB-2** — AC-3 requires this observable and the doc's B-matrix has no row for it. Threshold unchanged (ratio ≤ 1.2, doc:222-223).
- `cmd/retained/main.go` — **B6** (doc:142): a dedicated **non-`testing.B`** `main` program. Per fresh process: `runtime.GC()` ×2 → measure a **same-process empty-workload baseline** → build an `n=100,000` list of **ONE shared singleton element** (so element cost cancels and structure cost remains) → `runtime.KeepAlive` → `runtime.GC()` ×2 → report the **baseline-adjusted retained-heap delta ÷ n**. Emits machine-readable JSON: raw empty baseline, post-workload counters, adjusted delta, B/element.
- `cmd/gcshape/main.go` — **B8** (doc:144): runs B2's `n=12,800` workload once per arm with `runtime.ReadMemStats` immediately before and after; reports deltas for `NumGC`, `PauseTotalNs`, and **endpoint** `HeapAlloc` before and after. **Reports no peak** — the doc removed that metric as unsupported by endpoint snapshots (doc:182-183, :420). The executor must not add a sampler to "improve" this.

**Why B6/B8 are `main` programs and not tests:** they must run in fresh processes with forced GC and their own deadlines (doc:161-166), and CI runs `go test ./...` on ubuntu **and windows** (P4) — a memory-measurement `Test*` would execute on three platforms with numbers nobody reads and flakiness everybody pays for. As `main` packages they are compiled by CI (vet + build) and executed only by M5's runner.

**Acceptance criteria:**
- Each new benchmark's anchored regex selects exactly one cell (same proof as M2)
- `go run ./tools/internal/spike-listrep/cmd/retained -arm=C0 -n=100000` emits parseable JSON with all four fields; likewise `cmd/gcshape`
- B6's singleton-element property is asserted by a unit test (the same `eval.Value` pointer is stored `n` times), because the whole "element cost cancels" argument depends on it
- `go vet ./tools/...` rc=0 including both `main` packages

---

### M5 — Deterministic measurement runner and adjudicator
**Closes:** nothing on its own — it is the **mechanism** AC-3 and AC-5 require · **Estimated:** ~420 LOC (≈260 runner + ≈160 unit tests) · **~0.6 day**
**Depends on:** M2, M4

Implements doc:156-183 exactly. Sequenced here (end of Day 2 / start of Day 3) rather than purely on Day 3 because the doc's own Timeline "starts B6/B8" on Day 2 and "finishes five-fresh-process B6/B8 runs" on Day 3 (doc:358-360) — the fresh-process machinery must exist by the end of Day 2 for that sequence to be possible. **No day is added.**

`cmd/runner/main.go`:
1. Executes each cell in a **fresh process, five trials**, each Go invocation with `-count=1`, an explicit `-timeout=10m`, `-benchmem`, `-benchtime=1s` (AMB-4), and **exactly one** anchored cell regex. Imposes a **12-minute subprocess deadline**; a wedged command is killed and the trial is **reported as invalid, never silently omitted** (doc:161-165).
2. Pairs candidate and C0 trials **by ordinal** (`candidate_i / control_i`) and computes every threshold ratio from the **median of the five paired ratios**. Clause (a)'s within-arm L-ratio and clause (d)'s within-arm n-ratio pair the two sizes by ordinal the same way (doc:166-169).
3. Emits **all** raw operands, **all five** paired ratios, their sorted order, and the median arithmetic.
4. Applies the **predeclared tie/spread rule**: a median exactly on a threshold, **or** five paired ratios spanning the threshold (values on both sides, equality touching both), triggers **five additional** fresh-process trials; the median of **all ten** is final; equality passes a `<=` clause and passes a `>=` control clause exactly as written. **No discretionary reruns, dropped trials, or alternate aggregation** (doc:170-176).
5. Drives B6 and B8 through their `main` programs with the same explicit deadlines and the same five-fresh-process/ten-on-tie rule (doc:177-181).
6. Also records, per invocation, `/usr/bin/time -l` max RSS as **non-gating context** (cheap on darwin; feeds §7 R2's ceiling check).

**Acceptance criteria — the adjudicator's arithmetic is itself unit-tested against synthetic trial vectors, because it is the thing the 16-day decision rests on:**
- median of five paired ratios computed correctly for odd-length vectors, including unsorted input
- **median exactly on the threshold** → triggers rerun (not a pass)
- **five ratios spanning the threshold** (some below, some above) → triggers rerun
- **one ratio exactly equal to the threshold, others strictly one side** → equality *touches both sides* → triggers rerun
- after a rerun, the **median of all ten** is used, and equality passes `<=` and passes `>=`
- an invalid (deadline-killed) trial is surfaced as invalid and **does not** silently shrink the sample
- a cell whose regex matches ≠ 1 benchmark is rejected before running

**Risks:** the tie/spread rule has four boundary cases that are easy to get subtly wrong; hence the six-case test list above is a hard AC, not a suggestion.

---

### M6 — Full matrix run, kill-criterion arithmetic, verdict, and report
**Closes:** **AC-1, AC-2 (full), AC-3, AC-4 (leg 3), AC-5, AC-6, AC-7 (final), AC-8 (final)**
**Estimated:** ~140 LOC (verdict-table emitter) + ~450 lines of report markdown · **~1.0 day**
**Depends on:** M1–M5

1. **Full matrix pass** under M5's runner: all 76 measured points × 5 trials (§5), plus any predeclared reruns. Record the elapsed wall-clock and the before/after `rig.lock` observation (§3) in the report header, alongside `go version` and machine identity (doc:153-154).
2. **AC-1:** every cell carries **the command and its printed number**. A cell asserted from theory is a red AC (doc:132-133, :320-322).
3. **AC-2 / AC-3:** the (a)/(b)/(c)/(d) arithmetic, with both operands measured — clause (c) denominated in **B6's measured C0 B/element, never the assumed 16 B derivation** (doc:209-211, :330-331). All raw trials, all paired ratios, sorted order, median arithmetic, and B6's every same-process empty baseline and baseline-adjusted delta.
4. **AC-4 leg 3:** the grep showing field writes confined to constructor files, **quoted in the report with a firing control**.

   > **⚠ CORRECTED AT ITERATION 238 — THE REGEX AS ORIGINALLY WRITTEN MATCHED `==`, SO IT
   > "FIRED" ON COMPARISONS AND THE CLAUSE WOULD HAVE FAILED ON CORRECT CODE.** Measured on the
   > M3–M5 tree: `\.(…)[[:space:]]*=` returned **3** hits, and all three were `==` comparisons
   > (`chunklist.go:43`, `:101`, `:106`) — because `=` is a prefix of `==`. All three sit in
   > `ChunkCons` and `Uncons`, i.e. **outside any constructor**, so the rule as written ("every
   > hit must be … inside a constructor") **fails on a tree with zero real violations**. Either
   > the M6 executor reads it literally and emits a spurious clause-(e) STOP on a ~16-person-day
   > decision, or it waves the three hits through — and a genuine violation lands in the *same
   > function* and looks identical at a glance (verified: an in-place front-slack slot write in
   > `ChunkCons` produces a 4th hit indistinguishable in form from the 3 benign ones).
   >
   > **Two further facts, both measured, that change how this leg must be read.**
   > (a) The assignment-only form returns **0** hits on correct code — so for THIS clause an
   > **empty result is the PASS**, and the "a grep that returns nothing is broken" rule does not
   > apply to it. The instrument is proved live by a deliberate probe, not by the codebase.
   > (b) Constructor writes are **composite literals** (`&ChunkList{elems: …}`, `&ConsList{head:
   > …}`, `&SliceList{value: …}`), which use `:` and are invisible to *both* regex forms — so no
   > `=`-shaped grep can verify the "inside a constructor" half at all. Check it separately.

   ```bash
   # (i) ASSIGNMENT-ONLY sweep — '=' NOT followed by '='. On correct code this is EMPTY.
   grep -rnE '\.(head|tail|n|elems|count|total|k)[[:space:]]*=[^=]' tools/internal/spike-listrep/*.go
   # (ii) PROOF THE INSTRUMENT FIRES — a deliberate probe, since (i) is legitimately empty:
   printf 'package x\nfunc f(l *T) { l.total = 1 }\n' > /tmp/ac4_probe.go
   grep -nE '\.(head|tail|n|elems|count|total|k)[[:space:]]*=[^=]' /tmp/ac4_probe.go   # MUST match
   # (iii) CONSTRUCTOR CHECK — composite literals, which (i) cannot see:
   grep -nE '&(SliceList|ConsList|ChunkList)\{' tools/internal/spike-listrep/*.go
   # every hit must be in slicelist.go / conslist.go / chunklist.go, inside that type's constructor
   ```
   Any hit from (i) outside a constructor body **fails clause (e)**. A missing match in (ii) means
   the instrument is broken and **no verdict may be emitted from it**. Also re-affirm legs 1–2: **all** benchmarks compiled in the external `_test` package, and if any benchmark needed a raw field, clause (e) **fails** (doc:231-235).
5. **AC-5 verdict:** a per-candidate pass/fail table over (a)–(e) with the arithmetic inline; **≥1 candidate passes all five → GO**, naming the chosen representation (ties broken by (c) then (b)); **zero → STOP**, with a comment posted on **#745** carrying the full matrix and explicitly re-opening D-19. **The verdict may not be "partial go"** (doc:237-240). On STOP the report records the posted comment URL. If AC-2's control leg failed, **no verdict is emitted at all** (AMB-3).
6. **AC-6:** D3's table reproduced as-built with its per-row `file:line` citations, annotated with anything clause (e) proved infeasible (AMB-6).
7. **AC-7 anti-goal proof — with the firing control P3 showed is mandatory.**

   > **⚠ THE BASE MUST BE *DERIVED*, NEVER TRANSCRIBED — CORRECTED AT ITERATION 238 AFTER THE
   > HARDCODED LITERAL WENT STALE AND PRODUCED A FALSE AC-7 FAILURE.** This step used to carry
   > `8322d22b7…` as a literal. Measured at iteration 238: that base is **6 commits** behind
   > `origin/dev`, and running the step verbatim returns **10** files
   > (`cmd/ailang/eval_censored*.go`, `internal/eval_analysis/*`) — every one of them from an
   > unrelated commit (`d5bcfa0c8`, the eval censored-pair analyzer) that landed after the plan
   > was written. Against the correct base the same command returns **0**, with the unscoped
   > control returning **15**, so the branch is clean and the *instrument* was what failed.
   > This is the same trap iteration 237 hit from the other side — its executor directive had to
   > order the base DERIVED because the plan's literal was already wrong then — and it was fixed
   > only in that directive, so the plan kept the stale literal for a second iteration. A literal
   > base goes stale every time `dev` moves; a derived one cannot. Note the failure direction:
   > it manufactures a **STOP by artifact rather than by measurement**, on a gate that cancels
   > ~16 person-days.

   ```bash
   BASE=$(git merge-base origin/dev HEAD)          # DERIVE it. Never paste a SHA here.
   echo "AC-7 base: $BASE"                          # record the resolved value in the report
   git diff --name-only "$BASE"..HEAD -- internal/ cmd/ std/ examples/   # MUST be empty
   git diff --name-only "$BASE"..HEAD | head -40                          # MUST list the spike files
   ```
   If the scoped arm is non-empty, **check the base before concluding AC-7 failed**: confirm each
   named file is reachable from your own branch's commits (`git log --oneline "$BASE"..HEAD --
   <file>`) rather than inherited from `dev`.
   The scoped command alone is green even if the branch is empty (P3). Both lines go in the report.
8. **AC-8 final:** `make test` green with the spike included. Plus the three gates the doc's Conflict Surface omits (AMB-7, P4): `make vet` rc=0, `make fmt-check` rc=0, and `go test -timeout 300s ./tools/...` rc=0 as a stand-in for the CI leg's per-package timeout.
9. **Repo standing rules** (not doc ACs, and outside AC-7's pathspec): a `CHANGELOG.md` entry, and the implemented-doc report filed per the design-doc convention. Note `make test-check-changelog` only self-tests the gate script (`make/test.mk:52-54`) and does not enforce an entry against this diff — the entry is added because the coding standards require it, not because CI would catch its absence.

---

## 5. AC-1 cell enumeration (the strict reading, AMB-1)

Arms: **C0, C1, C2(K=8), C2(K=32)** = 4.

| Row | Grid points | × arms | Measured points |
|---|---|---|---|
| B1 branching | 2 (`m`) × 3 (`L`) = 6 | 4 | **24** |
| B2 linear build | 4 (`n`) | 4 | **16** |
| B3 iteration | 2 (`n`) | 4 | **8** |
| B4 `nth` sweep | 4 (`i`) | 4 | **16** |
| B5 materialize | 1 | 4 | **4** |
| B6 retained bytes | 1 | 4 | **4** |
| B7 alloc count | *read off B1/B2 `-benchmem` — no new runs* | — | **0** |
| B8 GC shape | 1 | 4 | **4** |
| | | **Total (AC-1 scope)** | **76** |
| B-LEN (additive, AMB-2) | 2 (`n`) | 4 | **8** (outside AC-1) |

**76 measured points × 5 trials = 380 fresh-process invocations** per full pass (plus B-LEN's 40, plus reruns). This is the sprint's dominant wall-clock cost and the number §3's ~30–90-minute estimate is built on.

---

## 6. Milestone → acceptance-criterion mapping (exact)

`◐` = partial (named legs only); `●` = closed.

| Doc AC | M1 | M2 | M3 | M4 | M5 | M6 |
|---|---|---|---|---|---|---|
| **AC-1** matrix complete (doc:320) | | ◐ B1,B2 | ◐ C2 cols | ◐ B3–B8 | | **●** |
| **AC-2** branching shape, kill (a) (doc:323) | | ◐ **control leg** | | | | **●** |
| **AC-3** kill (b)/(c)/(d) arithmetic (doc:328) | | | | ◐ B3, B6, B-LEN exist | ◐ protocol implemented | **●** |
| **AC-4** kill (e) folded check (doc:332) | ◐ legs 1–2 (C0, C1) | ◐ benches compile in `_test` | ◐ legs 1–2 (C2) | ◐ benches compile in `_test` | | **●** leg 3 (grep) |
| **AC-5** verdict (doc:336) | | | | | ◐ adjudication mechanism | **●** |
| **AC-6** D3 API draft (doc:340) | ◐ surface implemented | | ◐ C2 implements it | | | **●** |
| **AC-7** anti-goal proof (doc:342) | ◐ checkpoint | ◐ | ◐ | ◐ | ◐ | **●** + firing control |
| **AC-8** CI green (doc:345) | ◐ **first entry into CI** | ◐ | ◐ | ◐ | ◐ | **●** + vet/fmt/timeout |

**Nothing in the doc's Success Criteria is unmapped, and this plan introduces no acceptance criterion of its own that could substitute for one.** The only additions are (i) `B-LEN`, which AC-3 requires and the matrix omits (AMB-2), and (ii) extra *gates* — vet, fmt-check, per-package timeout, and firing controls on AC-4's grep and AC-7's pathspec — all of which make existing ACs harder to pass vacuously.

---

## 7. Risks

**R1 — C2's lock-free O(K) shared-chunk prepend may not be implementable in budget.** *Likelihood: medium.* The doc itself lists it as unverified (doc:419). **Mitigation is the doc's own:** mark C2's columns infeasible with the reason recorded, and proceed — the kill criterion tolerates it if C1 passes. This is the sprint's **primary descope lever**, and it is doc-sanctioned rather than planner-invented. *It does not weaken any clause: a GO still requires one candidate to pass all five.* **⚠ NARROWED at iteration 237 — see the controller adjudication in §4 M3.** That sentence is true of the *verdict rule* and false of the *matrix*: dropping C2 removes the candidate designed to pass clauses (b) and (c) where C1 is most at risk, so a STOP can be reached by descope rather than by measurement. The lever is available only for a reason bearing on the O(K) bound itself, and only once C1's clause-(b)/(c) numbers exist.

**R2 — B1's heaviest cell retains ~1.07 GB per benchmark iteration.** At `m=4096, L=16384`, C0 keeps all 4,096 results live: 4,096 × 16,385 × 16 B ≈ **1.07 GB per op**, and B1's design mandates keeping them live (doc:137). If results accumulate across `b.N` iterations the process can reach many GB. **Mitigation:** the result slice is released at the **end of each `b.N` iteration** (`KeepAlive` inside the iteration, not outside the loop); the runner records `/usr/bin/time -l` max RSS per invocation; if any cell exceeds a **4 GB** ceiling the executor pins that cell to `-benchtime=1x` and records the deviation and its reason in the cell's command string (AMB-4 permits exactly this, with disclosure). The rig has a documented history of eval-driven OOM kernel panics — this ceiling is not theoretical.

**R3 — measurement contention from a concurrent GPU eval.** *Likelihood: medium on a shared rig.* **Mitigation:** §3's before/after `rig.lock` observation recorded in the header; ordinal pairing means contention hits candidate and control alike; the five-trial median plus the tie/spread rerun is the doc's designed defence. **Not mitigated by taking the lock** — that would block GPU consumers for an hour for a job that uses no GPU (P8).

**R4 — a red spike test reds CI for everyone.** *Likelihood: low.* The doc names this as "the one real surface" (doc:275-276), and P4/P5 show it is three gates and one OS wider than the doc states. **Mitigation:** M1 gates on `make test` + `make vet` + `make fmt-check` at the moment of CI entry; unit tests are smoke-level and OS-independent (they run on windows-latest); the package imports only stdlib + `internal/eval`, which does not move.

**R5 — 3.0 days is the top of the doc's 2–3-day box.** *Likelihood: this is a statement, not a risk event.* The plan does not claim 2 days. If the box binds, R1's lever (drop C2, or drop only K=32) removes ~0.4 day of implementation and 32 of the 76 measured points. **The kill criterion, its thresholds, the five-trial protocol, and the verdict procedure are not available as descope surface under any schedule pressure.**

**R6 — the adjudicator silently mis-handles a boundary case** and hands back a confident wrong verdict on a 16-day decision. **Mitigation:** M5's six named unit tests over synthetic trial vectors, covering exact-threshold, spanning, and equality-touch cases explicitly.

---

## 8. Success metrics / definition of done

- All 8 doc ACs closed per §6, or an explicitly reported red with the failing observable named
- **One explicit verdict: GO (naming the representation) or STOP (with the posted #745 comment URL)** — or, if AC-2's control leg failed, **no verdict and a red AC-2** (AMB-3)
- 6 bisectable commits, one per milestone
- `make test`, `make vet`, `make fmt-check` all green; `go test -timeout 300s ./tools/...` rc=0
- Zero changes under `internal/`, `cmd/`, `std/`, `examples/` — proved with **both** the scoped command and its firing control
- No file in the spike over 500 lines (the 800-line CI gate does not reach `tools/`, C3 — this is hygiene, self-imposed)
- Report filed as the implemented-doc update carrying D1's matrix, D2's verdict, and D3's as-built table
- `CHANGELOG.md` entry (repo standing rule; not a doc AC, and CI does not enforce it here)

**Explicitly NOT in scope** — these are LC-2's, and planning them here would inflate a 2–3-day gate into a fortnight: the production accessor layer/seam, the `listrep` analyzer (clause (e) ships a **grep**, a syntax-level record for LC-2's type-aware analyzer to supersede — doc:233-235), any migration of the 386 `ListValue{` construction sites, any change to `internal/eval`, `internal/builtins`, or any consumer, and the end-to-end AILANG `repro.ail` re-measurement (LC-4's, doc:285-288).

---

**Sprint plan created:** 2026-08-20 · **Planned at:** `3ad6a2e46` · **Merge base:** `8322d22b75adfce7a4aa284eaf3ad99afdd4b570`
