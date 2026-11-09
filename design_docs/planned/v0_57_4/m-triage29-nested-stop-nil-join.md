# M-TRIAGE29-NESTED-STOP-NIL-JOIN: macOS `TestRuntimeNestedStopUsesRemainingDeadline` nil-Join recurrence — diagnosis and fix design

**Status**: Planned (investigation complete; fix is test-only and awaits human design approval — no implementation, sprint planning, executor dispatch, merge or deployment is authorized by this document)
**Target**: v0.57.4
**Priority**: P2 (non-required CI lane, rare flake; but it is the standing AC fixture for ownership/remaining-deadline semantics and its failures pollute release gates)
**Estimated**: ~0.5 day (≈10 LOC in one test file + CI soak)
**Dependencies**: None
**Author**: design-doc-creator (coordinator task-f8320de1, triage29), 2026-11-05

---

## Problem Statement

`internal/effects.TestRuntimeNestedStopUsesRemainingDeadline` intermittently fails
on GitHub macOS runners with:

```
process_runtime_lifetime_test.go:288: stalled nested cleanup reported success
--- FAIL: TestRuntimeNestedStopUsesRemainingDeadline (0.04s)
```

i.e. `managedProcess.Join(ctx)` returns **nil** where the test asserts that a
nested-cleanup stall must NOT report success within the 120 ms stop deadline.

**Exact failure evidence (this triage's subject):**

| Run / job | Commit | Result | Duration |
|---|---|---|---|
| Build and Release run [42007729357](https://github.com/sunholo-data/ailang/actions/runs/42007729357), job [127178632414](https://github.com/sunholo-data/ailang/actions/runs/42007729357/job/127178632414), `Build macos-latest`, step "Run tests" | `ea1486ca` | `--- FAIL: TestRuntimeNestedStopUsesRemainingDeadline (0.04s)`, `process_runtime_lifetime_test.go:288: stalled nested cleanup reported success` (log retrieved 2026-11-05) | **0.04 s** |
| Green direct parent f6a95d75, jobs 127160679878 / 127160680003 (both `Build macos-latest`) | `f6a95d75` | PASS / PASS | **0.15 s / 0.19 s** |
| Current head run 42008879770, jobs 127182506508 / 127182506928 (`Build macos-latest`) | `218b94b1` | PASS / PASS (both Build macOS legs; run conclusion `success`) | **0.22 s / 0.14 s** |

The failed run's **required** checks (`test`, ubuntu-latest per `.github/workflows/ci.yml:97`) passed; only the non-required `Build macos-latest` leg failed, and its two sibling matrix legs were cancelled by fail-fast (the "two cancelled matrix legs / extra skipped steps" in the task brief — mechanics, not diagnostics). `git compare f6a95d75...ea1486ca` shows exactly 2 commits touching only `changelogs/`, `docs/src/constants/version.js`, `std/VERSION` — **no runtime, test or workflow source changed between the green parent and the red head**, so the failure is scheduling-dependent, not a regression.

**Historical recurrence — same assertion, same fast-fail signature, always macOS Build legs, always green on re-run:**

- 2026-11-05: job 127178632414 (`ea1486ca`, release PR #2130) — 0.04 s, line 288 (this investigation).
- Fleet iteration 84: Build and Release run 41827278668, job 126565127880 (PR #2116, head `96e1691`, markdown-only diff) — `--- FAIL: TestRuntimeNestedStopUsesRemainingDeadline (0.04s)`; failed in none of the other 24 Build-and-Release runs; **re-run completed SUCCESS**, recorded as timing flake (`design_docs/fleet-mission-evidence/iteration84/gates.md:52`).
- PR #2010: macOS Build job 123447755475 — fails at `process_runtime_lifetime_test.go:288`; no causality assigned to its documentation diff (`design_docs/triage-mission-log.md:444`, and issue #2069 comment 6599231990).
- `design_docs/fleet-mission-log.md:1069`: same test, `Build macos-latest`, 0.04 s, diff was markdown, job re-run green, "recorded as a timing flake, not filed".
- `design_docs/fleet-mission-log-archive.md:1562`: same test, 0.03 s, on a PR that touched no Go.
- **Distinct and not this failure**: Linux/container failures of the same test at line **323** ("nested fixture survived bounded stop") are a container artifact — the fixture becomes a **zombie adopted by PID 1** in containers whose init does not reap (already documented in `design_docs/implemented/v0_54_10/spawn-process-result-sprint-plan.md` and `v0_56_0/m-pkg-asset-lock-resolution-sprint-plan.md`; reproduced on this investigation's container: `ps -o pid,ppid,stat -p <pid>` → `PPID 1, STAT Z`). macOS CI line-288 nil-Join (0.03–0.04 s) and container line-323 zombie survival are **different assertion sites with different causes**.

**Prior state of knowledge**: cross-vendor MiniMax review reached an independent CONCUR on a process-lifecycle hypothesis class, but root cause was unproven and no synthetic reproduction existed (task brief; triage-mission-log iteration 25). This document supplies the first deterministic synthetic reproduction and a complete causal proof. No standalone investigation doc existed (duplicate gate verified — see Verification Log), so recurrence tracking should reconcile here rather than spawn new docs.

## Root Cause (proven)

**The test's readiness gate orders the wrong process's `Bind`.** The fixture's
nested child writes the marker the test gates on, but the OUTER helper — the
process the test actually stops — can still be between "spawn nested" and
`runlife.Bind` when the stop request arrives. A stop consumed before `Bind`
is completed **cleanly by design** (unbound session → `Complete(nil)` → clean
receipt), so `Join` legitimately returns nil and trips the assertion. It is a
fixture/readiness race (hypothesis H1), not a runtime defect.

### The causal chain, verified against code

Relevant code (all read and cited, current `218b94b1`):

- Helper flow, `internal/effects/process_runtime_lifetime_test.go` `TestMain`,
  nested mode: `runlife.Bootstrap` → `NewManagedProcess(nested …)` → write
  `path+".nested-pid"` → **`runlife.Bind(cleanup)`** → write own `path` marker
  → sleep. The marker `path+".nested"` that the test polls (line 274–282) is
  written **by the nested child** after ITS OWN `Bind` — it proves the nested
  child is bound but **does not order the outer helper's `Bind`**.
- `internal/runlife/session.go` `Session.stop`: `if s.stopped || s.completed`
  guard → `stopped=true` → `var err error; if cleanup != nil { err = cleanup(ctx) }`
  → `s.Complete(err)`. With no `Bind` yet, `cleanup == nil` → `err == nil`.
- `Session.Complete(nil)` writes a receipt frame with **empty** `Error`, then
  `terminate(0)` exits the process.
- `internal/runlife/child.go` `AfterStart` reader: a terminal receipt with empty
  `Error` leaves `c.err == nil`; `Child.Join` then returns `nil`.
- `internal/effects/process_managed.go` `waitLoop` sets `mp.receiptErr = nil`;
  `managedProcess.Join` returns `errors.Join(ResolveGroupError(…), writeErr, receiptErr)`
  with the exit-status pre-filter (`*exec.ExitError` and `context.Canceled` ignored)
  → **nil**. The child's exit(0)/exit(5) is an `ExitError` or nil either way, so it
  cannot save the assertion.

### Why the 0.04 s duration proves pre-Bind stop (and excludes H2/H3)

The expected PASS shape is: readiness gate → `RequestStopContext(120ms)` →
outer cleanup stalls on the nested "slow" child for ≈118 ms → error receipt →
`Join` non-nil at ≈120 ms. Every observed green run is 0.14–0.22 s total
(≈120 ms stall + process startup).

A bound cleanup **cannot** return nil in this fixture: the nested child
(slow mode) cleanup is `<-ctx.Done(); return ctx.Err()` — always
`DeadlineExceeded` — so its receipt always carries `Error`, so the outer
cleanup's `errors.Join(stopErr, nested.Join(ctx))` is always non-nil, so the
outer receipt is always dirty. Therefore:

1. `Join` returned nil at 0.04 s **via the `<-mp.done` branch with `ctx.Err() == nil`**
   (the `<-ctx.Done` branch would return a non-nil `context.DeadlineExceeded`).
2. The done branch requires `waitLoop` to have completed, hence `receiptErr`
   was assigned, hence it is **provably nil**, hence `Child.Join` saw
   `c.err == nil` → the outer helper wrote a terminal receipt with **empty Error**.
3. Empty Error ⇒ `Complete(nil)` ⇒ either `cleanup == nil` (pre-Bind stop) or a
   bound cleanup returning nil — and (2nd item above) a bound cleanup cannot
   return nil. **⇒ pre-Bind stop.**

The red run's 0.04 s (no 118 ms stall) is the timing fingerprint of exactly this
state; the 0.03–0.04 s durations of all four historical recurrences match it.

### Hypothesis discrimination (as requested by the task brief)

| Hypothesis | Verdict | Evidence |
|---|---|---|
| **H1: helper readiness/Bind race** (stop consumed before the outer helper's `Bind`) | **CONFIRMED — root cause** | Mechanism proven deterministically (synthetic variant A, 12/12); fingerprint match (0.03–0.04 s across 5 occurrences vs 0.14–0.22 s green); code chain above; explains macOS-only + load-dependence (the outer main goroutine must be starved across the whole nested-child startup ≈10–40 ms, which only a heavily oversubscribed runner produces). |
| **H2: lifetime receipt propagation defect** (a dirty receipt lost / mis-ordered on the parent side) | **RULED OUT** | The receipt channel behaved exactly as designed in every run: green runs propagate the error receipt and `Join != nil`; in the red run the receipt was **clean and correctly delivered** — `Child.Join` returned nil because `c.err == nil`, i.e. propagation worked. No code path drops a dirty receipt without another error surfacing (`errors.Join(c.err, c.requestErr)`; request-pipe-closed-after-completion only returns nil when the receipt was already clean — `internal/runlife/child.go` `Join`). |
| **H3: legitimate completion before stop** (fixture finished naturally, cancellation raced a valid completion) | **RULED OUT as an independent cause; CONFLATED with H1 by design** | Neither the outer (nested mode) nor the nested (slow mode) helper ever calls `runlife.Complete` naturally (they sleep forever after `Bind`). The "completion" in the red run is the *session-level* designed-legitimate completion of an **unbound** stop — legitimate for the runtime (nothing bound ⇒ nothing to clean ⇒ clean receipt is correct early-shutdown semantics) but not legitimate for what the test asserts (the nested cleanup chain never ran: `.remaining` files would be absent; the test fails at line 288 before reading them). This is why the failure looks like a "success" lie while the protocol is honest. |

**Consequence**: this is a **test-fixture race**, not a runtime bug. The
ownership, remaining-deadline and receipt semantics under test are correct and
must be preserved; the designed clean-completion-on-unbound-stop behavior is
load-bearing for real early shutdown (a host SIGTERM arriving before a runtime
binds must yield a clean receipt) and must NOT be "fixed".

## Goals

**Primary Goal:** Make `TestRuntimeNestedStopUsesRemainingDeadline` deterministic in
what it measures (post-Bind nested stall under a remaining deadline) by closing
the readiness/Bind race in the fixture gate — with **zero** changes to runtime
semantics, public API, or process guarantees.

**Success Metrics:**
- The unbound-stop window is unreachable by the test (gate proves both `Bind`s).
- Focused repetitions stay green: `-run TestRuntimeNestedStopUsesRemainingDeadline -count=N` (N ≥ 50) with no nil-Join occurrence, on Linux CI and across ≥ 10 macOS Build runs.
- Green runs keep the ≈120 ms-plus timing shape (the assertion still exercises the stall, not a vacuous path).
- `git diff` of the eventual fix confined to `internal/effects/process_runtime_lifetime_test.go` (plus optionally the `TestMain` helper's marker ordering); no changes in `internal/runlife/`, `internal/effects/process_managed.go`, or any production file.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at the test readiness gate (wait for the OUTER helper's post-Bind marker) rather than changing `Session.stop` semantics | The alternative would alter early-shutdown receipt semantics — a process-guarantee change this task explicitly forbids and which would break real hosts | human (this doc recommends the test-only route) | design | low |
| Keep the unbound-stop clean-completion semantics byte-identical | Load-bearing for host SIGTERM-before-bind; owned by the runlife protocol contract | human (ratify "no semantic change") | design | high if violated |
| Do NOT fold into M-PROCESS-EXEC-PARENT-LIFETIME (exec #2034) | Different layer and different cause (ordinary `Process.exec` has no `runlife.Prepare` at all; its problem is invisibility to shutdown, not a fixture race). No evidence of shared cause; the task brief pre-authorized folding only with such evidence | human (confirm scope separation) | design | low |
| Which marker(s) the gate waits on (own marker + nested marker, vs reordering the helper's `Bind` before nested spawn) | Both close the race; gate-on-both-markers is the smallest diff and does not touch helper mode behavior | agent (implementer may choose either variant A1 or A2 below) | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Human approves the test-only fix route and ratifies that `Session.stop`'s unbound-clean-completion semantics are intentionally preserved.
- [ ] Human confirms this stays out of M-PROCESS-EXEC-PARENT-LIFETIME / #2034 scope.

## Solution Design

### Overview

The test currently waits for `marker+".nested"` (written by the nested child
after its own `Bind`). Strengthen the gate so that a stop can only be sent once
**both** lifetimes are provably bound. Two equivalent variants; A1 is
recommended (smallest diff, no helper behavior change).

**Variant A1 (recommended): gate on both markers.** In
`TestRuntimeNestedStopUsesRemainingDeadline`, replace the single-file
readiness poll with a poll for both `marker` (the outer helper's own ready file,
written after `runlife.Bind`) and `marker+".nested"`. Once both exist, any stop
is by construction processed post-Bind (`Bind` completed under `s.mu` before
the marker write; `Session.stop` takes the same `s.mu`, so it observes the
registered cleanup).

**Variant A2 (equivalent): reorder the helper.** In `TestMain` nested mode,
call `runlife.Bind` before spawning the nested child. Then `marker+".nested"`
existing implies the outer `Bind` already ran. (Analysis: a stop arriving
between `Bind` and spawn cannot be observed by this test — the gate file does
not exist yet — so the race is equally closed; but this reorders fixture mode
behavior and is a slightly larger change. Note that reordering alone without
keeping the gate would leave a different latent window: cleanup could run with
`nested == nil` and write `path+".joined`"; the existing gate covers it, which
is why A2 must retain the gate too.)

**Explicitly rejected alternatives:**
- Making `Session.stop` treat an unbound stop as an error, or having `Bind`
  record "stop arrived before bind" for the parent to observe: changes the
  public shutdown receipt contract for every real runtime invocation (host
  SIGTERM during early startup must remain a clean shutdown). Violates the
  task's preservation constraints.
- Making the test tolerant of nil Join (skipping on "fast clean completion"):
  that deletes the assertion under test — the remaining-deadline propagation
  would no longer be verified on any run that loses the race, silently
  weakening the AC fixture for `design_docs/planned/v0_56_2/m-process-exec-parent-lifetime.md`
  (which mirrors this test as its receipt AC).
- Increasing the 120 ms budget to "wait out" the race: the race is binary
  (pre-Bind or not), not a budget overrun; a larger budget changes nothing
  about the window and weakens the remaining-deadline assertion.

### Implementation Plan (proposed — not authorized by this document)

**Phase 1: Gate fix** (~1 hour)
- [ ] Variant A1: extend the readiness loop in `TestRuntimeNestedStopUsesRemainingDeadline` (lines ~271–281) to require both `marker` and `marker+".nested"`; keep the 5 s deadline and 1 ms poll.
- [ ] Add a one-line comment stating why: the outer marker is written post-`Bind`; the nested marker alone does not order the outer `Bind`, and an unbound stop completes cleanly by design (cite this doc).

**Phase 2: Verification** (~2 hours + CI soak)
- [ ] Focused: `go test ./internal/effects -run 'TestRuntimeNestedStopUsesRemainingDeadline' -count=50` (Linux; note the container line-323 zombie artifact may require asserting only up to line 288 semantics locally, or running on a reaping host).
- [ ] Package: `go test ./internal/effects -count=1`.
- [ ] macOS validation rides the normal Build and Release lane; historical recurrence rate (≈5 known reds across hundreds of runs) means ≥10 consecutive green macOS Build legs is the practical soak bar.

**Phase 3: Bookkeeping** (~0.5 hour)
- [ ] Move this doc to implemented/ once the fix lands; reconcile future recurrences here first.

### Files to Modify/Create

**Modified files (proposed):**
- `internal/effects/process_runtime_lifetime_test.go` — readiness gate, ~5–10 LOC.
  (Variant A2 additionally reorders ~6 lines in the `TestMain` helper block.)

**No production files.** `internal/runlife/**`, `internal/effects/process_managed.go`,
`stream_process.go`, `worker_owner.go` remain untouched.

## Synthetic Reproduction and Controls (evidence captured this session)

All synthetic runs were isolated, focused, on a disposable Linux container
(go1.26.10, 5 visible cores with ~2-CPU cgroup quota), using a **temporary
non-committed test file** in package `effects` (created, run, then deleted; the
working tree is clean). No paid/provider calls were made. Three variants:

**Variant A — ungated immediate stop (mechanism reproduction).** Spawn the
nested-mode helper via `NewManagedProcess`, then `RequestStopContext(120ms)`
immediately, without any readiness gate — forcing the stop frame to be
consumed by a possibly-unbound session.

Result: **12/12 runs reproduced the exact CI line-288 signature** —
`joinErr == nil` with `receiptErr == nil` (clean receipt), `exitErr == nil`,
`.remaining`/`.nested-pid` files absent (outer cleanup never ran). Sample log:

```
EVIDENCE[A-ungated] elapsed=9ms stopErr=<nil> joinErr=<nil> rawStopErr=<nil>
receiptErr=<nil> writeErr=<nil> exitErr=<nil> stopDeadlineSet=true
files=map[:false .joined:false .nested:false .nested-pid:false .nested.remaining:false .remaining:false]
REPRODUCED: nil Join — the CI line-288 'stalled nested cleanup reported success' signature
```

(Elapsed 9–90 ms — inside the 120 ms budget, matching the CI red's 0.04 s
fingerprint. The nil fields here are *stable* reads: `Join` returned via the
`done` branch, so `waitLoop` had completed and `receiptErr` was already
assigned.)

**Variant B — strengthened gate (positive control, the proposed fix).** Same
spawn, but stop only after **both** `marker` (outer, post-Bind) and
`marker+".nested"` exist. 5/5 iterations:

```
EVIDENCE[B-gated] elapsed=121ms stopErr=<nil> joinErr=context deadline exceeded ...
files=map[:true .joined:false .nested:true .nested-pid:true .nested.remaining:true .remaining:true]
```

`Join != nil` at ≈120 ms with both `.remaining` files present — the full
remaining-deadline chain executed exactly as the test intends. (Note: variant
B's `receiptErr`/`exitErr` field reads race `waitLoop` when `Join` returns via
`ctx.Done`; they are not evidence of receipt content and are not cited as
such. Variant A's reads are stable for the reason above.)

**Variant C — original gate, repetitions.** The existing test's exact gate
(nested marker only), 40 iterations per run: **0/40 nil-Joins** unloaded,
**0/40** with 10 CPU spinners, and **0/240 across 6 parallel `go test`
instances** (280 iterations total counting the first loaded run). The natural
race did **not** reproduce on this Linux box — consistent with a rare,
OS-scheduler-dependent window that macOS CI runners hit under full-suite load
(`.github/workflows/build.yml:83` runs the entire verbose suite in one macOS
job). **Honest limitation:** the load-dependent sub-window (outer helper
starved across nested-child startup ≈10–40 ms) remains unreproduced naturally
here; what is proven deterministically is the *mechanism* (A), the *fingerprint
match* of the CI red (0.04 s ⇒ clean receipt ⇒ pre-Bind stop, by the deduction
chain above), and the *sufficiency of the fix* (B).

**Why macOS Build legs only:** the required `test` job runs on
`ubuntu-latest` (`.github/workflows/ci.yml:97`); macOS executes this test only
inside the Build and Release matrix, where the whole suite runs concurrently on
a 4-vCPU runner. The starve-the-outer-helper window needs exactly that
oversubscription.

## Conflict Surface (test fixture)

This design does not touch parser/typechecker/codegen, but the fixture is
shared by four tests, so the surface is enumerated:

1. **Positions extended:** none — only the readiness gate of one test changes.
2. **Other constructs in the same position:** the `TestMain` helper's marker
   files are also consumed by `TestRuntimeOwnedWorkersRequireReceipt` (its own
   readiness poll on `marker`, unchanged — that test gates on the child it
   stops itself, which is correct) and `TestRuntimeLateStopJoinsBackpressuredInheritedOutput`
   (gates on `marker` + reads `.writer-pid`; unchanged). Variant A1 touches
   neither.
3. **Disambiguation:** none needed; pure file-existence polling.
4. **Programs that must still work (fixtures):** all five helper modes
   (`good`, `cleanup-output`, `missing`, `forged`, `natural`, plus `nested`/
   `slow`/`late-output`) with their existing marker names — variant A1 adds
   no new files, it only waits for the existing outer marker.
5. **Deliberate changes:** none to helper modes; the nested test may take
   slightly longer to enter its stop phase (one extra poll), no semantic change.

## Success Criteria

- [ ] Variant A1 (or A2) gate implemented; `go test ./internal/effects -count=1` green on Linux.
- [ ] Focused `-count=50` run has zero nil-Join occurrences.
- [ ] ≥10 consecutive `Build macos-latest` runs green with no recurrence at line 288 (existing recurrence record: ~5 reds across hundreds of runs).
- [ ] Green runs retain ≥0.14 s duration shape (stall still exercised).
- [ ] Diff confined to `internal/effects/process_runtime_lifetime_test.go`; `internal/runlife` untouched (verify: `git diff --stat` + `make check-boundaries`).
- [ ] This doc moved to implemented/ with an implementation report reconciling the historical recurrence list above.

## Testing Strategy

**Unit/focused tests:** variant B control from this doc is the template: gated
stop must yield `Join != nil` at ≈120 ms with both `.remaining` files present.
If the implementer wants a permanent guard, a focused assertion may be added
that the readiness gate waits on both markers — but no new helper mode should
be added (keeping the fixture surface minimal).

**Integration:** unchanged — the existing suite is the AC.

**Manual/CI:** the macOS Build lane is the only environment with the observed
failure; the soak criterion above is the acceptance bar. Local Linux runs can
validate only up to line-288 semantics (container zombie artifact at line 323).

## Deferred Decisions

- Whether to also assert (in the test, after the gate) that the outer helper's
  own marker pre-dates the stop request — unnecessary for correctness; **agent
  may decide** while implementing.
- Whether to record the fingerprint (elapsed < 370 ms, `.remaining` absent) as
  an explicit diagnostic message in the test failure text to speed up future
  triage — **agent may decide**.
- Soak length (10 vs 25 macOS runs) — **human may set** at review.

## Non-Goals

- **No change to `runlife` session semantics** — unbound-stop clean completion
  is designed, correct, and load-bearing for real early shutdown.
- **No folding into M-PROCESS-EXEC-PARENT-LIFETIME / issue #2034** — different
  layer (ordinary synchronous `Process.exec` performs no `runlife.Prepare` and
  registers no worker; its gap is invisibility to host shutdown), no shared
  cause established by this investigation. The two docs remain separate;
  exec2034 keeps mirroring this test as its receipt AC.
- **No attempt to make the macOS runner stop oversubscribing** (workflow-level
  `-p` tuning etc.) — the race is closed at the fixture; scheduler load is not
  a defect.
- **No chase of the container line-323 zombie artifact** — already documented
  as environmental (PID 1 non-reaping); out of scope here.
- **No runtime/production code changes of any kind** in this task's scope.

## Timeline

**Single session** (~4 hours + CI soak):
- Gate fix + focused verification: 1–2 h.
- Package run, boundary check, PR, macOS soak starts: 1–2 h.
- Soak observation rides normal release activity over the following days.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gate change masks a real future receipt-propagation bug (test now "too deterministic") | Med | The gate only orders *when* the stop is sent; the post-stop chain (remaining-deadline files, stall, error receipt, group teardown, bounded survival) is asserted unchanged. Variant B shows all of it still runs. |
| Rare residual race (marker file write vs `Bind` memory visibility) | Low | Marker writes are `os.WriteFile` after `Bind` returns and after `Session.stop`'s mutex ordering; `Bind` completes under `s.mu` before the file write, and `Session.stop` takes the same mutex — there is no remaining interleaving where the gate is satisfied and cleanup is unregistered. |
| Fix lands but a future recurrence appears at line 288 anyway | Low | Reconcile into this doc; the fingerprint table (elapsed, `.remaining` presence) distinguishes pre-Bind recurrence from any new mechanism immediately. |
| macOS flake ownership confusion (Build leg is non-required) | Med | Keep this doc as the single standing diagnosis; cite it from the gate records instead of re-classifying as "timing flake, not filed" each time (as happened in iterations 84/25). |

## Verification Log (skill hard gates)

| # | Claim | Verification |
|---|---|---|
| 1 | Failing assertion is line 288 nil-Join, 0.04 s, job 127178632414 | Job log downloaded via GitHub API 2026-11-05: `process_runtime_lifetime_test.go:288: stalled nested cleanup reported success` + `--- FAIL: … (0.04s)`; log excerpt quoted above. |
| 2 | Green parent f6a95d75 jobs 127160679878/127160680003 PASS 0.15 s/0.19 s; current 218b94b run 42008879770 completed success, macOS jobs PASS 0.22 s/0.14 s | Job logs fetched (grep hits quoted in this session's evidence: `--- PASS: TestRuntimeNestedStopUsesRemainingDeadline (0.15s)` etc.); run status from GitHub API. |
| 3 | ea1486ca vs f6a95d75 differ only in changelog/version files (identical runtime/test/workflow source) | `compare/f6a95d75...ea1486ca`: 2 commits, files = `changelogs/unreleased/…`(removed), `changelogs/v0.32-current.md`, `docs/src/constants/version.js`, `std/VERSION`. |
| 4 | Test gates on `marker+".nested"` only; outer helper `Bind`s after spawning nested | Read `internal/effects/process_runtime_lifetime_test.go` lines 271–281 (poll; line 277 "nested runtime not ready") and `TestMain` nested-mode block (Bootstrap → spawn → pid file → Bind → own marker). |
| 5 | Unbound stop completes cleanly (`cleanup==nil → Complete(nil) → empty-Error receipt → Child.Join nil → managedProcess.Join nil`) | Read `internal/runlife/session.go` `Session.stop`/`Complete`, `internal/runlife/child.go` `AfterStart`/`Join`, `internal/effects/process_managed.go` `waitLoop`/`Join` at current HEAD 218b94b1. |
| 6 | Bound cleanup can never return nil in this fixture (slow mode always errors) | Read the helper closures: slow mode `<-ctx.Done(); return ctx.Err()`; nested mode `errors.Join(stopErr, nested.Join(ctx))`; nested `Join` nil requires a clean nested receipt, which slow mode can never emit. |
| 7 | Join-nil at 0.04 s ⟹ done-branch ⟹ receiptErr provably nil ⟹ clean receipt ⟹ pre-Bind stop | Deduction from `managedProcess.Join` (done branch requires ctx fresh; `ctx.Done` branch returns non-nil) + waitLoop assignment order (receiptErr set before `close(mp.done)`) + claim 6. |
| 8 | Mechanism reproduces deterministically; strengthened gate closes it; original gate holds unloaded/loaded on Linux | Synthetics A (12/12 nil-Join), B (5/5 non-nil ≈120 ms, files present), C (0/40 + 0/40 + 0/240; logs quoted above); temp file deleted, `git status` clean. |
| 9 | macOS runs this test only in Build legs; Build runs the whole verbose suite in one job | `.github/workflows/ci.yml:97` (`runs-on: ubuntu-latest` for required test); `.github/workflows/build.yml:83` (`go test -v $(go list ./... )`). |
| 10 | No standalone investigation doc exists for this failure (duplicate gate) | `ailang docs search "nested stop nil join"`, `"runtime lifetime worker cancellation receipt"`, `"process parent lifetime SIGTERM exec"` → no planned/implemented doc about this test's nil-Join failure; repo grep for `TestRuntimeNestedStopUsesRemainingDeadline` in `**/*.md` finds only baseline/AC references (v0_56_2, v0_54_7, v0_54_12, v0_56_0, v0_57_3, fleet/mission/tri logs, retros) — bookkeeping, not diagnosis. |
| 11 | Container line-323 failures are zombies adopted by PID 1 (separate cause) | Baseline focused run on this container failed at line 323; `ps -o pid,ppid,stat -p 3967` → `3967 1 Z`; matches prior documented container artifact. |
| 12 | Historical recurrences list | `design_docs/fleet-mission-evidence/iteration84/gates.md:52`, `design_docs/fleet-mission-log.md:1069`, `design_docs/fleet-mission-log-archive.md:1562`, `design_docs/triage-mission-log.md:444` (job 123447755475), issue #2069 comment 6599231990. |

Negative-existence claims: **no** production-code alternative exists that both
preserves unbound-stop clean semantics and makes this fixture assertion
sensitive to pre-Bind stops — verified by reading the full path the receipt
takes (`Child.Join` returns nil whenever the receipt is clean; the parent
cannot distinguish unbound-clean from bound-clean without new protocol frames,
which would change the public shutdown contract). That is why the fix belongs
in the test gate.

Quorum note: `ailang design-quorum` was **not** run — the task brief explicitly
grants no paid/provider calls for this investigation. Independent non-provider
review (cross-vendor MiniMax CONCUR on the hypothesis class, root cause then
unproven) is recorded in triage-mission-log iteration 25; this doc upgrades
that hypothesis to a mechanism-proven diagnosis.

## Related Documents

**Planned (check for overlap):**
- [m-process-exec-parent-lifetime.md](../v0_56_2/m-process-exec-parent-lifetime.md) — ordinary synchronous `Process.exec` parent-lifetime reach (issue #2034). Adjacent, **not** shared-cause; this test remains its receipt AC. Explicitly out of this doc's scope per the task brief and per the layer analysis above.
- [spawn-process-result-sprint-plan.md](../../implemented/v0_54_10/spawn-process-result-sprint-plan.md) — documents the container line-323 zombie artifact baseline.

**Bookkeeping referencing this test (historical baseline/AC, reconciled here):**
- `design_docs/fleet-mission-evidence/iteration84/gates.md` (job 126565127880, 0.04 s)
- `design_docs/fleet-mission-log.md`, `design_docs/fleet-mission-log-archive.md`, `design_docs/triage-mission-log.md` (job 123447755475, issue #2069 comment 6599231990)
- `design_docs/planned/v0_56_2/m-process-exec-parent-lifetime.md` (AC mirror), `design_docs/implemented/v0_57_3/m-serve-api-typed-arg-binder-implementation.md`, `docs/sprint-retros/M-VM-LIST-TAIL-SHARE-RESIDUAL-retro.md`

## References

- [Design Axioms](/docs/references/axioms)
- CI evidence: run [42007729357](https://github.com/sunholo-data/ailang/actions/runs/42007729357) / job 127178632414 (red, `ea1486ca`); parent run 42002424439 jobs 127160679878/127160680003 (green, `f6a95d75`); current run 42008879770 (green, `218b94b1`)
- Issue #2069 comment 6599231990 (historical recurrence bookkeeping)
- Code: `internal/effects/process_runtime_lifetime_test.go`, `internal/effects/process_managed.go`, `internal/runlife/session.go`, `internal/runlife/child.go`, `internal/proctree/procgroup_unix.go`

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a scheduling-dependent race from a verification gate; runtime determinism untouched (production code unchanged). |
| A2: Replayability | 0 | No trace/eval impact. |
| A3: Effect Legibility | 0 | No effect-surface changes. |
| A4: Explicit Authority | 0 | Ownership/authority paths unchanged; explicitly preserved. |
| A5: Bounded Verification | +1 | The AC fixture becomes reliably reproducible within its existing bounded deadline. |
| A6: Safe Concurrency | +1 | Eliminates a false concurrency failure that mis-attributed a designed clean shutdown path to a runtime defect. |
| A7: Machines First | 0 | No language/diagnostic surface change. |
| A8: Minimal Syntax | 0 | No syntax. |
| A9: Cost Visibility | 0 | No cost model change. |
| A10: Composability | 0 | Fixture-local. |
| A11: Structured Failure | +1 | Failure text can now distinguish pre-Bind recurrence from new mechanisms (deferred diagnostic), and the false "success" failure disappears. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +5** → **Decision: Move forward (test-only fix, upon human approval).**

Hard violation check: no −1 on A1/A3/A4/A7 (all production semantics preserved).

## Future Work

- If a *second* fingerprint of pre-Bind-stop recurrence ever appears after the
  gate fix (it should not, by the mutex argument in Risks), consider a
  runlife-protocol "bound" flag as a separate design — never as a silent
  semantic change here.
- The remaining-deadline AC in exec #2034 keeps mirroring this test; when that
  sprint runs, its receipt assertions inherit this doc's gate rationale.

---

**Document created**: 2026-11-05
**Last updated**: 2026-11-05
