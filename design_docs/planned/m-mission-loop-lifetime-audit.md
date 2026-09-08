# M-MISSION-LOOP-LIFETIME-AUDIT

**Status**: audit complete, remediation not started
**Date**: 2026-09-08 (attended session, Mark)
**Scope**: all four mission loops over their whole lifetime, plus the repository ground state
**Why now**: the fleet is PAUSED until the binary mission control can take over. This audit
establishes what the shell-driven era actually produced, so the binary version inherits the
lessons rather than the defects.

---

## 1. What the loops produced

| Mission | Iterations | First | Last | Repo |
|---|---|---|---|---|
| v1 | 339 (index) | 2026-07-10 | 2026-09-08 | sunholo-data/ailang |
| world | 171 | — | 2026-09-08 | sunholo-data/ailang-world |
| motoko | 39 | 2026-08-12 | 2026-09-07 | sunholo-data/ailang |
| docs | 12 | 2026-08-28 | 2026-09-06 | sunholo-data/ailang |
| **total** | **561** | | | |

v1 averaged ~5.6 iterations/day for 60 days.

### Self-classification (v1, tagged window 309–348, 31 tagged rows)

| Tag | Count | Share |
|---|---|---|
| `[HARNESS]` | 17 | 55% |
| `[PRODUCT]` | 10 | 32% |
| `[ADMIN]` | 2 | 6% |
| `[REFUTATION]` | 2 | 6% |

Iterations 348–351 are all `[HARNESS]`.

### Where the effort actually went (files touched, 2026-08-25 → 09-08)

```
MISSION-DOCS   883      HARNESS-GO      464      HARNESS-SHELL  180
CLI            275      SKILLS          107      DOCS-SITE      129
LANG-CORE       16   <- compiler + stdlib, 14 days, 9 commits, one a revert pair
```

### Goal distance moved BACKWARDS

v1's own "docs before v1.0.0": `N=10` (09-02) → `N=13` (09-06) → `N=12` (09-07/08).
Fifty-plus iterations, two docs further from v1.0.0 than a week earlier.

---

## 2. What was genuinely good

These are not consolations; they are load-bearing and must survive into the binary.

1. **Independent judging works, and it caught real defects.** 50 iteration summaries mention
   the judge. It repeatedly failed the controller's own work — iteration 351's round 1 caught
   a sprint that had made production retry backoff uncancellable; 333's judge proved a
   milestone's headline test was measuring the milestone before it; 312's caught a regression
   the controller shipped in its own fix. `generator ≠ judge` earns its cost.

2. **The loop overturns itself on evidence.** 26 summaries are refutations, 2 tagged
   `[REFUTATION]` outright. Iteration 309 refuted an approved sprint's own unit; 314 refuted
   a fix by re-measuring it. A loop that can say "I was wrong" is the rarest property here.

3. **Nothing is permanently lost when a slot dies.** 17 summaries are recoveries, and this
   audit tested the claim independently: of **63 local branches with no PR**, after correcting
   for squash-merge, **zero** hold content absent from dev. The two apparent exceptions
   (`m-property-generator-coverage`, `m-arity-style-diagnostic`) are stale `planned/` copies of
   docs that now live in `design_docs/implemented/`. Died-mid-flight recovery genuinely works.

4. **The iteration index is a real instrument.** One line per iteration, regenerated wholesale,
   explicitly "GREP THIS BEFORE PICKING WORK". It is the reason this audit was possible at all.

5. **Decision ledgers stay clean.** World: 22 rows, **0 open**. motoko: 1 open with a
   pre-registered default date. v1: 1 open (D-61). Decisions are not silently accumulating.

6. **World is the counter-example that proves the model can ship.** Same shell driver, same
   quota drought — and it delivered M1 (semantic world library), M2 (`ailang-worldd` daemon),
   and a queue-closure census instrument (PR #136, evaluator PASS 87/100). Its own census:
   41 of 89 rows closed. Whatever is wrong is not intrinsic to the loop design.

---

## 3. What was bad

### 3.1 The dominant failure class: declared but not walked

This audit found the same shape **four separate times**, and it is the single most valuable
finding for the binary migration:

| Control | Declared | Actually walked |
|---|---|---|
| Role fallback chains | `MISSION_<ROLE>_FALLBACK`, since 2026-08-26 | Nothing read it for weeks |
| Anthropic ration | gates routing | Gated the **controller** only; `_mc_probe` had no admission check, so designer and evaluator kept spending while the controller yielded |
| Ollama ration | `evaluateOllamaQuota` implements 10%/day pacing | Unreachable — gated behind a limits file that cannot exist, because the gauge has no reset |
| Quota admission tests | `test_codex_quota_admission.sh`, `test_ollama_quota_admission.sh` | Never listed in `make test-launchd-drivers`; never ran in CI |

A control that is declared, documented, commented and untested reads exactly like a working
one in every log line. **The binary must make "declared" and "walked" the same act.**

### 3.2 The kill switch is checked AFTER the probes

`tools/launchd/mission-control.sh`: the role-probe block ends at line 1435; the kill switch is
line 1437. A disabled mission still fires, still runs every Anthropic/Codex/Pi probe, and only
then discovers it is off.

Measured today: the docs fire at 04:41 spent until **04:49:36** on hung Anthropic probes before
logging `kill switch present — skip`. Nineteen minutes and four inference probes on a mission
that was already disabled. **This directly undermines "keep the loops paused".**

### 3.3 Slot mortality is high and concentrated at gate 3

Recent retained slot verdicts:

| Mission | COMPLETED | CRASHED/KILLED |
|---|---|---|
| v1 | 20 | 9 (31%) |
| world | 14 | 2 |
| docs | 9 | 1 (REAPED at gate-0) |
| motoko | 5 | 0 |

Deaths cluster at `gate-3`/`gate-3b` — the work gate. Three `KILLED_at=gate-3` in three days,
including World's final iteration today (stall watchdog: no progress across 5 samples/600s with
a descendant alive ≥2400s). v1's dashboard records two consecutive slots dying mid-flight, one
holding a green PR.

### 3.4 Quota was blind in two of three buckets until 2026-09-08

- `anthropic`: `capacity-unknown — UNRATIONED` while the ledger counted 1.38M tokens against
  nothing. The number existed the whole time at `GET /api/oauth/usage`.
- `ollama`: the fleet's only wholly unrationed bucket. Measured 36.1% → 43.1% → 69.4% between
  09-07 17:02 and 09-08 09:06 — ~2.1pp/h against a 0.42pp/h ration, with nothing to stop it
  before the 95% cutoff.

Both fixed 2026-09-08 (`00d4bcde9`, `d142fd4cd`, `9be7abb49`). Ollama is now paced on observed
rate, which needs neither capacity nor reset.

### 3.5 dev is left red by agent landings

`make check-git-exec` has been failing since `48e72ef7e` (the canary/reliability landing on
09-08), on two bare-name `git` exec sites. Every PR inherited it; it is what failed #1113,
whose own diff touches neither file. Fixed in `4e8074483` by routing through `internal/gitexec`
rather than growing the ten-line legacy baseline. 131 of 561 iteration summaries mention "red".

### 3.5b THREE dev reds from one landing, not one

`48e72ef7e` left three CI gates red on dev, and every PR inherited all three:

| Gate | Cause | Status |
|---|---|---|
| `check-git-exec` | two bare-name `git` exec sites | **fixed** `4e8074483` (routed through `internal/gitexec`) |
| `check-home-isolation` | six hand-rolled `HOME` overrides | **fixed** `84f4308c1` (routed through `testutil.SetHomeDir`) |
| `test-windows` | 15 tests, POSIX assumptions | **open** — diagnosed below |

**`test-windows` diagnosis (2026-09-08).** `cmd/ailang/mission_activation.go:158` carries an
UN-TAGGED inline copy of a POSIX privacy check:

```go
if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
    return errors.New("activation state directory must be private and not a symlink")
}
```

On Windows `os.MkdirAll(dir, 0700)` does not yield 0700, so the check fires, the state file is
never written, and 12 downstream tests then fail with `missing hold: … The system cannot find
the file specified`.

The package it belongs to already solved this: `internal/mission/activation` is split into
`lock_unix.go` (`//go:build darwin || linux`) and `lock_other.go`, and the latter states
plainly that **"local mission activation requires macOS or Linux host locking"**. So activation
is deliberately unix-only, and the 15 failures are tests exercising a feature the package
declares unsupported on that platform.

The fix is therefore a platform guard, not a portable permission check: give the `cmd/ailang`
check the same build-tagged treatment so it refuses on Windows with the package's own message,
and tag the tests to match. This is a decision about someone else's just-landed subsystem, so
it is recorded here rather than taken unilaterally.

Verified not caused by this session's work: the Windows failure set is byte-identical at
`4e8074483` (before) and `84f4308c1` (after) — 15 tests, nothing added.

### 3.6 Agent handoff never fires — 7 design docs stranded in PRs

| PR | Design doc | Lines |
|---|---|---|
| 1033 | `m-pkg-deterministic-lockfile` | +157 |
| 1093 | `m-unroutable-inbox-visibility` | +180 |
| 1094 | `m-coordinator-remote-honour-or-reject` | +76 |
| 1098 | `m-coordinator-reopen-remote` | +62 |
| 1099 | `coordinator-remote-flag-strict` | +42 |
| 1100 | `m-surface-remote-approval-registry-load-failure` | +37 |
| 1101 | …`-sprint-plan` | +34 |

Produced by design-doc-creator/sprint-planner via the coordinator, never collected. This is the
known `ProcessApprovalRequest` → `OnAgentApproved` gap. **This is the one place real unfinished
work is sitting.**

### 3.6b Three of the stranded PRs were chain-integrity TESTS, not work

Reviewed and merged 2026-09-08, then **reverted the same day**. The task prompts said so
outright — "this is a chain-integrity test as much as a design task", "The point is to
exercise the handoff, not to produce a large design" — and the review checked whether each
described defect was real (it was) without checking whether the work was ever intended.
`design_docs/planned/` is the queue the loops draw from, so a test artifact left there
becomes committed work the next time a loop picks it up.

**The defects they describe ARE real and were verified against dev HEAD on 2026-09-08.**
Recorded here so reverting the docs does not lose the findings:

| Defect | Evidence at dev HEAD |
|---|---|
| `coordinator list\|logs\|diff\|pending\|status` accept `--remote` and silently act on the LOCAL store | `coordinator_list.go` and `coordinator_inspect.go` contain **zero** `remoteCoordinatorSelected` calls, while `approve`/`reject` honour it at `coordinator_actions.go:26,140` |
| `coordinator reopen` is local-only | no `remoteCoordinatorSelected` in `coordinator_lifecycle.go` |
| hand-rolled flag parsers silently swallow unknown flags | `coordinator_lifecycle.go` has **zero** `default:` cases |

Whoever picks these up should write the design fresh rather than restoring a doc that was
authored to exercise a handoff.

**The genuinely-intended one was merged and kept**: #1093
`m-unroutable-inbox-visibility` — a message to an inbox no agent serves is accepted, marked
unread and never dispatched, with zero signal to sender or operator. Verified: no unroutable
check exists anywhere in the send path.

**Two more were already implemented** and were closed with reasons: #1100 (registry-load
failure on remote approval — `checkRegistryCanDispatch` now fails before mutation) and #1101,
its sprint plan, whose M1 tests exist in `coordinator_approvals_registry_test.go`.

### 3.7 Accumulated clutter

330 local branches, 128 worktrees. 264 branches had a merged PR; 63 had no PR and hold nothing
absent from dev.

**Cleared 2026-09-08** (attended, Mark): 107 stale worktrees removed and 320 branches deleted,
after snapshotting every branch SHA to `.ailang/state/cleanup-2026-09-08/` so the set is
recoverable. 128 → 21 worktrees, 330 → 10 branches, ~83 GB reclaimed. Two worktrees were
correctly REFUSED and left alone: one locked by a live Claude session, one now removed. The
four pinned driver roots, the eval-infra worktrees and every open-PR branch were preserved.

---

## 4. The Anthropic probe hang — evidence, not a conclusion

24 probe timeouts on 2026-09-08 in the retained logs, across **all four** Anthropic models
(`claude-fable-5-1` ×8, `sonnet` ×8, `claude-opus-5` ×4, `claude-sonnet-5` ×4). Every one:
`rc=124`, **empty captured output**. Per the probe's own comment, an empty capture on timeout
means a hang or backoff loop, not an error reply.

### The condition nobody has tested

```
03:07:53  v1     iteration STARTS (claude -p controller), runs long
04:28-04:38  world  claude-fable-5-1, sonnet, claude-opus-5 — ALL time out
04:43-04:49  docs   claude-sonnet-5, sonnet — ALL time out
```

The 2026-09-07 fix (`f81f5b808`) reproduced the probe clean "single and 4-way concurrent, and
under a scrubbed launchd-like env". That reproduction covered probes against *each other*. It
did **not** cover a probe issued while a long-lived `claude -p` mission controller is in flight,
which is exactly the condition above.

### The counter-example that stops this being a conclusion

At **08:28**, World's controller probe **succeeded** (`probe ok`) while v1's 07:57 iteration was
still running. So concurrency with a live controller is **not sufficient** on its own. Something
else differed at 04:28 — candidate discriminators, none yet tested: elapsed time of the holding
iteration (1h20m vs 31m), OAuth refresh timing relative to the 5-hour window (reset 10:20Z), or
provider-side latency.

### What the fix on 09-07 did and did not do

It made rc=2 failures **say why** — previously the error text was discarded. That worked and is
why we can see "empty capture" today. It did not, and was not intended to, stop the hang.

### Next step

Instrument, do not theorise: record `claude -p` start/end timestamps per probe alongside the
holding iteration's age, and correlate against the OpenRouter-style provider trace if one exists
for the subscription lane. The search space is now narrow. **Cost of leaving it: each hang burns
2 × 120s per model per fire, and silently degrades Anthropic lanes to ollama — which is what
drove the ollama gauge to 69.4%.**

---

### 3.8 Two timing-sensitive tests that will keep reddening CI

Neither was touched by this session; both are recorded because a red that recurs and gets
explained away each time is how a real regression eventually hides behind a known flake.

| Test | Assertion | Measured |
|---|---|---|
| `test_driver_notify.sh` "hanging gh comment" | `elapsed -le 7` wall-clock, on a 2s timeout | failed at **8s** under concurrent load; 3/3 unloaded passes |
| `TestIterationWaitingDeadlineExpiresDurably` | first `s.Run` returns `waiting` with `TimeoutSeconds = 1` | `git rev-parse --absolute-git-dir: signal: killed` — the 1s stage budget kills the git subprocess |

The second is rare and load-correlated: **0/15** isolated runs and **0/5** batches of ten
reproduced it, but it failed twice while the machine was also running the full suite, lint
and CI polling — and once on the GitHub runner. An earlier note in this session called it
"does not reproduce locally" on the strength of 3 and 8 passing runs; that was overstated,
and the correction is recorded here rather than quietly dropped.

The honest fix is not to widen either bound. For the second, the question to settle is
whether a work item's `TimeoutSeconds` should be charged against the repository probe at
`runtime_stage.go:226,238` at all — a one-second work item is currently unusable not because
its work is slow but because `git rev-parse` is inside its budget.

## 5. Remediation backlog

Ordered by value, not effort.

1. **Collect the 7 stranded design-doc PRs** (§3.6). Real content, sitting unmerged.
2. **Move the kill switch above the probe block** (§3.2). One-line reorder; makes the pause real.
3. **Land #1113** — already MERGEABLE, was blocked only by the inherited dev red.
4. **Delete 327 branches and prune worktrees** (§3.7). Mechanical, zero risk.
5. **Instrument the Anthropic probe hang** (§4). Narrow search space, high recurring cost.
6. **Make "declared = walked" structural in the binary** (§3.1) — the migration's single most
   important inherited lesson.
7. **Investigate gate-3 slot mortality** (§3.3), including whether the stall watchdog's
   no-progress definition is right for a long evaluator turn.

## 6. What this says about the binary migration

The shell loop's failures are overwhelmingly **integration** failures, not logic failures: a
control exists, is correct in isolation, and is not wired to the path that matters. Four
independent instances in one audit. The binary migration's value is therefore not "the same
logic in Go" — it is a single place where admission, dispatch and gating are *the same object*
rather than a shell function and its four call sites, three of which forgot to call it.

Deaths at gate 3 and unbounded probe cost are the other two recurring shapes, and both are
scheduling concerns the binary already owns in `mission iterate`.
