# M-NIGHTLY-FLAKE-GUARD: Variance Guard for the Nightly Eval Regression Detector

**Status**: Implemented 2026-07-28 (mission iteration 113) — PR [#504](https://github.com/sunholo-data/ailang/pull/504), squash `038d9322d`, dev CI green. Evaluator sonnet PASS 87/100 round 1, zero blocking.

> **One deliberate deviation from this doc, approved at plan time.** The escalation backstop ships as
> `consec >= K AND no record in the current consecutive-failure run has class == "regression"`, NOT
> the literal `consec == K` written below. The literal form loses the escalation *forever* if the
> classifier misses the Kth night, silently breaking this doc's own "never unpaged past night 3"
> guarantee. The replacement is still exactly-once and replay-deterministic, and changed no verdict
> in the live replay; `test_Escalation_missed_third_night_fires_on_fourth` fails under the literal rule.
>
> **Two premises below were refuted first-party during the sprint** — corrected here rather than
> rewritten in place: (1) the false-alarm corpus is **45 issues across 24 distinct benchmarks**, not
> the 4 `json_parse` issues cited; two more (#499, #500) fired on 2026-07-28 *while the sprint ran*.
> (2) `/tmp` directories **outlive their contents**, so only **five** nights of real data survived
> (07-24…07-28), not the six-to-seven a directory count suggests — which also means `--bootstrap`
> can never fill a W=5 window (ceiling: 4 prior nights).
>
> **Landing posture:** night 1 after merge has no JSONL, so the run is loudly DEGRADED and files
> zero issues, leading the controlplane summary with the warning. Run
> `python3 tools/nightly_classify.py --bootstrap` once before the next 05:00 nightly for immediate
> coverage (idempotent).

**Target**: v1.0.0 (mission queue item, iter-105; harness/tooling lane — no compiler change)
**Priority**: P2 (recurring false-alarm cost: 4 GitHub issues from one bimodal benchmark, each a Gate-0 triage slot + the #417 external-optics problem)
**Estimated**: ~1.3 days (4 milestones, each independently commit-able; +2h in quorum round 1 for the history-file atomicity/locking contract, +0.5h in round 2 for the ownership-checked lock — see Milestones)
**Dependencies**: `tools/launchd/nightly-eval.sh` (the detector); `ailang messages send` GitHub-sync semantics (`cmd/ailang/messages_send.go`)
**Extends**: [m-eval-regression-detector-contract](../../implemented/v0_29_0/m-eval-regression-detector-contract.md) — that contract defined flaky-vs-persistent *within one night* and solid-vs-not *against one prior night*; this doc extends "solid" to a trailing multi-night window. It also finally delivers that doc's never-landed pinning test (see Verification Log V15).

> **Scope note (routing):** This change does NOT touch `internal/parser/`, `internal/types/`,
> `internal/elaborate/`, codegen, or any compiler surface. It edits one shell script, adds one
> Python tool under `tools/`, and adds no Go code. Per the design-doc contract, a **Blast radius**
> section replaces the Conflict Surface section for *output consumers*; the state file this design
> introduces gets its own **State & locking conflict surface (audit)** section covering the repo's
> existing state-store, locking, and atomic-write machinery (added in quorum round 1).

---

## Problem statement

The nightly regression detector (inline `python3` heredoc, `tools/launchd/nightly-eval.sh:271-330`)
classifies a benchmark as `REGRESSION` when it fails **all** trials tonight (≥2 trials, ≥1
non-infra category) but passed **all** trials in **the single most recent prior run**. A
`REGRESSION` sends `ailang messages send controlplane … --type bug`, and `--type bug` **implies
GitHub-issue creation inline in the send command** (`cmd/ailang/messages_send.go:100-106` →
`syncMessageToGitHub`, title prefixed `[nightly-eval]` at `internal/messaging/github.go:265`),
plus one Discord ping.

A benchmark that is merely **bimodal on the local model** defeats this N=1-night memory. Verified
banked history for `json_parse` on `opencode-qwen3-5-35b-a3b-mxfp8` (agent mode, rag_on arm):

| night | pass/trials | error categories on fail | detector outcome |
|---|---|---|---|
| 2026-07-22 | no json_parse data in banked dir | — | — |
| 2026-07-23 | 1/2 | compile_error | flaky-but-recovered, no alert |
| 2026-07-24 | 2/2 | — | — |
| 2026-07-25 | 0/2 | thrash_aborted, compile_error | **REGRESSION → issue #480** |
| 2026-07-26 | 2/2 | — | — (self-recovered, zero action) |
| 2026-07-27 | 0/2 (+0/2 rag_off) | thrash_aborted/compile_error | **REGRESSION → issue #485** |

Four GitHub issues on this one benchmark — #286, #292 (June), #480, #485 (July) — all CLOSED as
noise. Each costs a mission Gate-0 triage slot, and while open reads to external viewers as an
unresolved regression (#417). The detector has no concept of variance: every pass→fail flip after
a single good night looks like a fresh solid→broken break.

Two aggravating defects ride along:

1. **History substrate is `/tmp`** (`nightly-eval.sh:244-248` globs `/tmp/nightly_eval_*_rag_on/agent`)
   — only 6 nights currently survive (2026-07-22…27; the June runs are gone), and `/private/tmp`
   does not survive reboot. The detector's amnesia is *caused* by its substrate.
2. **`was_solid_in_prev` certifies "solid" from a single trial** (`len(seen) >= 1`,
   `nightly-eval.sh:326`): a prior night that happened to bank one passing trial fully certifies
   the benchmark, maximizing flip sensitivity.

---

## Verification Log

Every load-bearing claim, with what establishes it. Checked live at `f0ddd264e`
(branch `design/m-nightly-flake-guard`) on 2026-07-27.

| # | Claim | Evidence |
|---|---|---|
| V1 | Detector is an inline python3 heredoc; no Go code | `nightly-eval.sh:271-330` (read) |
| V2 | rag_on is the canonical arm for tonight's side | `nightly-eval.sh:233` `RESULTS_AGENT="${RESULTS_DIR}_rag_on/agent"` |
| V3 | Prior side = lexically-last `/tmp/nightly_eval_*_rag_on/agent`, exactly ONE run; missing prior → fail-safe to GAP | `nightly-eval.sh:244-253` |
| V4 | `was_solid_in_prev` accepts a single prior trial as "solid" | `nightly-eval.sh:326` `return len(seen) >= 1 and all(seen)` |
| V5 | Persistent-failure gate: ≥2 trials, all fail, ≥1 category outside `{api_error, timeout, executor_error}` | `nightly-eval.sh:278,306` |
| V6 | REGRESSION → per-bench controlplane msg `--type bug` (`:346`) + one Discord ping (`:361`); GAP → quiet `--type note` (`:378`) | `nightly-eval.sh:346-386` (read) |
| V7 | **What creates the GitHub issue**: the `ailang messages send` CLI itself, synchronously — `knownGitHubTypes = {bug, feature}`; `syncToGitHub := *github \|\| knownGitHubTypes[category]` → `syncMessageToGitHub`. **No daemon, no launchd job, no category filter elsewhere.** Title `[%s] %s` from `internal/messaging/github.go:265`. Therefore the fix belongs in the *script's label→type mapping*, not in Go messaging code: any `--type note` send already creates no issue. | `cmd/ailang/messages_send.go:100-106,218-240` (read); `internal/messaging/github.go:265` (read) |
| V8 | **CORRECTION to the mission-queue row** (`design_docs/v1-mission.md:1231`, "fold in: compare like-for-like CONDITIONS — #485 compared against yesterday's `_rag_on` while today produced both"): **wrong — the comparison already IS like-for-like.** Both sides use the rag_on arm: tonight via `nightly-eval.sh:233`, prior via the `*_rag_on/agent` glob at `:245`. The rag_off dir is never consulted. The mission should drop this bullet. | `nightly-eval.sh:233,245` (read) |
| V9 | `json_parse` banked history 1/2 → 2/2 → 0/2 → 2/2 → 0/2 (07-23…07-27, rag_on); 07-27 also 0/2 on rag_off — reconciles iter-105's "0/4" (it pooled both arms of the Monday A/B run). All fails carry non-infra categories (compile_error/thrash_aborted) → each 0/2 night passes the V5 gate | Recomputed from `/tmp/nightly_eval_2026072{3..7}_rag_on/agent/json_parse_*.json` + `_rag_off` (this session) |
| V10 | Issues #286 (2026-06-06), #292 (06-08), #480 (07-25), #485 (07-27): all titled `[nightly-eval] Nightly regression: json_parse (…)`, all CLOSED | `gh issue view 286 292 480 485` (this session) |
| V11 | #480 closed "transient — recovered without action" (iter-103); #485 closed "REFUTED AS NOISE" with the same longitudinal table (iter-105) | `gh issue view 480/485 --json comments` (read) |
| V12 | #417 exists, CLOSED: "Do the nightly eval regressions gate a release, or is blocking manual?" — the external-optics cost of stale open alarms | `gh issue view 417` |
| V13 | Observatory DB has `trial_history` (per-trial outcome schema) but it has **ZERO writers** — schema-only, part of unimplemented ELO trial recording. `eval_baselines` records token stats on PASS trials only (`UpdatePassedTrial`), no fail history | `internal/observatory/migrate_v16.go:34-47`; `grep -rn "trial_history\|RecordTrial\|InsertTrial"` → only DDL + migrate list + one test table-existence check; `internal/observatory/baselines.go:81` |
| V14 | Only 6 nights of history survive in `/tmp` (20260722–27); June dirs gone. Negative-existence: no durable per-night pass/fail store exists anywhere (V13 + no other candidate found) | `ls -d /tmp/nightly_eval_*` (this session) |
| V15 | The v0.29 contract doc's acceptance item "pin with a fixture test" **never landed**: no nightly classifier test file exists anywhere in the repo; also its `INFRA_CATEGORIES includes stream_death` item is NOT in the live script (line 278 has only 3 categories) | `find . -name "*nightly*"` → only the two shell scripts + plist; `nightly-eval.sh:278` |
| V16 | Duplicate-title guard: `messages send` refuses a duplicate title per inbox without `--force`; nightly titles embed the date so each night is unique | `cmd/ailang/messages_send.go:141-152` |
| V17 | `nightly-lang-eval.sh` (weekly language eval) has NO regression classifier — only summary sends; unaffected by this change | `grep REGRESSION tools/launchd/nightly-lang-eval.sh` → none |
| V18 | The no-prior-run degradation IS loud in the log ("no prior run found — flaky failures will NOT alert tonight") but is NOT surfaced in the controlplane summary body | `nightly-eval.sh:253` + summary send at `:392-398` (read) |
| V19 *(round 1)* | The whole nightly (eval + classifier + sends) already runs under the rig lock: script sources `rig-lock.sh` and blocks on `rig_lock_acquire wait` before any work — scheduled runs cannot overlap each other | `nightly-eval.sh:40-42` (read) |
| V20 *(round 1)* | Repo shell-lock convention: atomic `mkdir` (macOS has no flock) + stale-steal (6h default) + wait/nowait modes + EXIT-trap release | `tools/launchd/rig-lock.sh:16-41` (read) |
| V21 *(round 1)* | Go lock twin adds PID-liveness stealing (dead-holder recovery beyond mtime staleness); same dir + semantics as the shell lock | `internal/riglock/riglock.go:136-183` (read) |
| V22 *(round 1)* | Repo atomic-write convention exists: temp file + validate + `os.Rename`, explicitly tested for rollback-on-error | `internal/eval_analysis/dashboard_io.go:152-199`; `export_docusaurus_test.go:174-196` (read) |
| V23 *(round 1)* | The `~/.ailang/state/` watermark/cursor files are bare non-atomic `echo >` overwrites — a convention that exists but is NOT adequate for a sole history source | `os-rotation-filler.sh:295,367` (read) |
| V24 *(round 1)* | `~/.ailang/state/` EXISTS on this rig (holds mission watermarks, rotation cursor, observatory/eval DBs, `rig.lock.d`) — the D1 premise is now verified, not assumed; the defensive `makedirs` is adopted anyway for fresh machines | `ls ~/.ailang/state/` (controller + this session) |
| V25 *(round 1)* | Title-dedup runs BEFORE SQLite insert and before GitHub sync — a refused duplicate send exits 1 having created nothing; all four nightly send titles embed `${DATE}` | `cmd/ailang/messages_send.go:136-153` (read); `nightly-eval.sh:352,366,383,397` (read) |
| V26 *(round 1)* | `AILANG_NIGHTLY_EVAL_DRY_RUN=1` exits BEFORE the eval and before the classifier — dry-run cannot append history. The ordinary same-date-rerun vector is a manual full invocation, which reuses the same dated `/tmp` dir (`RESULTS_DIR` embeds `YYYYMMDD`) | `nightly-eval.sh:147-151` (read) |

**UNCONFIRMED** (stated as such, not asserted):

- **U1** — That #286/#292 (June) were the *same* bimodal flip: the June banked dirs are gone from
  `/tmp` (V14), so the underlying trial data cannot be replayed. What IS confirmed: same benchmark,
  same title pattern, both closed 2026-06-12 — the exact day the current GAP/solid-in-prev
  classifier landed (its comment cites "5 such false alerts fired on 2026-06-12",
  `nightly-eval.sh:262`). Close comments for the June pair were not audited.
- **U2** — The precise macOS mechanism that removed the June dirs (reboot wipe vs periodic tmp
  cleanup). The *observable* is enough for the design: history in `/tmp` demonstrably does not
  persist beyond ~a week on this rig.

---

## What this design decides

### D1 — History substrate: a classifier-owned JSONL with an explicit atomicity/idempotency contract (not `/tmp` globbing, not the observatory DB)

`~/.ailang/state/nightly-eval-history.jsonl` — the `state/` dir is verified present on this rig
(V24: it already holds the mission watermarks, rotation cursor, and observatory DB), and is
created defensively anyway (`os.makedirs(os.path.dirname(history), exist_ok=True)`) so a fresh
machine cannot crash on first run. One record per (date, bench, model, arm):

```json
{"date":"2026-07-27","bench":"json_parse","model":"opencode-qwen3-5-35b-a3b-mxfp8","arm":"rag_on","trials":2,"passes":0,"cats":["compile_error","thrash_aborted"],"class":"suspected-flake"}
```

- Written by the classifier at the end of each nightly run (rag_on arm only — V2/V8: that is the
  canonical arm; rag_off stays out of the record just as it is out of the comparison today).
- Read as the trailing window for D2 — **excluding tonight's own date** (D1.2). `/tmp` result dirs
  remain the *source* of tonight's trials; they stop being the *memory*.
- **Bootstrap is EXPLICIT and manual, never automatic on absence** (D1.5). An absent file always
  means DEGRADED (D5) — never a silent auto-heal.
- If the file goes missing/corrupt: **loud degradation** per D5, never a silent "no regressions".

Round-1 quorum correctly rejected the original framing of this file as "append-only": a bare
append has no idempotency under rerun, no concurrent-writer story, and no crash-safety, and any of
those can corrupt the sole history source or re-fire an escalation. The file therefore carries an
explicit contract:

**D1.1 — Record identity & idempotent reruns.** The unique key is **(date, bench, model, arm)**.
`--update-history` is a read-modify-write of the whole file: load, drop any existing records
matching tonight's keys, append tonight's records, write back (atomically, D1.4). A same-date
rerun — an ordinary event: the script is manually invokable and reuses the same dated `/tmp` dir
(V26) — therefore **replaces** its earlier records instead of duplicating them. If a load finds
same-key duplicates anyway (pre-contract file, hand edits), resolution is deterministic:
**last record in file order wins**, and the next update compacts them away. (A pure append-log
with read-side dedup was considered and dropped: duplicates accumulate unboundedly and every
future reader must re-implement the resolution rule; compaction keeps the file the single
unambiguous source.) Rewrite cost is trivial: ~40 benchmarks × 1 line/night ≈ a few KB per night.

**D1.2 — Tonight never pollutes its own window.** The D2 window is defined over records with
`date` **strictly earlier than tonight's date**, where tonight's date is parsed from the
`--tonight` dir name (`nightly_eval_YYYYMMDD_rag_on`), not wall clock — deterministic under
replay. This holds regardless of write ordering, so on a rerun where tonight's record already sits
in the file from the first invocation, window statistics are unchanged: verdicts are a pure
function of (tonight-dir, history-minus-tonight). Combined with D1.1, a rerun reproduces
**byte-identical verdicts — including a K=3 escalation, which re-classifies identically on the
rerun but cannot re-deliver**: exactly-once delivery is anchored on the existing title-dedup
guard — `messages send` refuses a duplicate title per inbox *before* SQLite insert and *before*
GitHub sync (V25), and every nightly title embeds the date (`:352,:366,:383,:397`).

**D1.3 — Concurrent writers: bounded mkdir lock with PID + ownership-token liveness.** Scheduled
runs are already serialized end-to-end by the rig lock (`nightly-eval.sh:40-42` sources
`rig-lock.sh` and blocks on `rig_lock_acquire wait`, V19), but a standalone
`nightly_classify.py --update-history` (manual rerun, debugging) holds no lock. The classifier
therefore takes its own lock around the read-modify-write, reusing the repo's established
convention (`rig-lock.sh` / `internal/riglock`, V20/V21) for the *acquisition* primitive: atomic
`os.mkdir` of `~/.ailang/state/nightly-eval-history.lock.d`, bounded acquisition of **60 s**
total, then **fail loud** — non-zero exit as an operational error, never a silent skip. We
deliberately do NOT take the rig GPU lock itself: wrong granularity (6h stale window sized for GPU
jobs, and a standalone classifier run must not queue for hours behind an eval).

Round 2 of the quorum rejected an mtime-only stale-steal here, and was right: `os.replace`
prevents a *torn* file but not a *lost update*, so a live-but-paused holder whose lock is stolen
can resume and overwrite the new owner's read-modify-write, and can then delete a lock it no
longer owns. Staleness-by-mtime is a safety *premise*, not an implementation detail, and it is
unsupported. The lock is therefore **ownership-checked**, per the reviewer's specification:

- The lock directory stores the holder's **PID plus a random ownership token**.
- **Steal only when the holder PID is confirmed dead** via `os.kill(pid, 0)`. A holder that has
  exceeded the 10-minute staleness threshold but is still **alive is never stolen from** — the
  acquisition simply keeps waiting to its 60 s bound and then fails loudly.
- **Conservative recovery rule for unreadable metadata** (lock dir present but PID/token missing,
  truncated, or unparseable — a pre-contract or partially-created lock): treat the holder as
  *possibly live*. Steal only after the 10-minute staleness threshold AND only then, logging the
  steal and the reason at WARN; if the metadata becomes readable during the wait, revert to the
  PID-liveness rule.
- After acquisition, **verify the token before entering the critical section**; a token mismatch
  means another process owns the lock — abort loudly rather than proceed.
- **Release removes the lock only if the token still matches**, so an old holder can never delete
  a replacement owner's lock.

Tests (M2): a holder that exceeds the stale threshold but remains **alive** is not stolen from and
the waiter fails loudly at the bound; an **old holder resuming after its lock was replaced**
detects the token mismatch and does not write; ownership-checked release **cannot delete another
process's lock**.

**D1.4 — Crash-safe updates.** Writes follow the repo's atomic-write convention
(`writeJSONAtomic`, `internal/eval_analysis/dashboard_io.go:152-199` — temp + validate + rename,
V22): write the full new content to `<history>.tmp.<pid>` in the same directory, flush +
`os.fsync`, then `os.replace` (atomic rename on POSIX). A crash mid-write leaves the previous
file fully intact plus a stray temp file, removed on the next locked update. Torn or unparseable
lines can therefore only come from pre-contract files or hand edits; the reader skips them
**individually and loudly** — the skip count appears in the D5 health line — and the next
compaction drops them. The bare `echo >` overwrite used by the watermark/cursor files (V23) is
explicitly NOT reused: losing one cursor tick is self-healing; losing the sole history source
is not.

**D1.5 — Bootstrap is an explicit one-off flag, never an automatic reaction to file absence.**
Round 2 of the quorum caught a genuine contradiction between the round-1 D1 and D5: both fired on
the *same* trigger (history file absent), one auto-healing from `/tmp` and one demanding a loud
DEGRADED state. Since absence alone cannot distinguish a fresh deploy from a mid-week deletion,
the auto-bootstrap would have silently swallowed exactly the failure D5 exists to announce — and
would have broken this doc's own "history file deleted mid-week" acceptance test. Per the
reviewer's fix, **automatic bootstrap on file absence is removed**:

- `nightly_classify.py` takes an explicit **`--bootstrap`** flag, used **manually, once, at
  initial deployment**, which seeds the history from whatever `/tmp/nightly_eval_*_rag_on/agent`
  dirs still exist (the seeding parser is the same code path that reads tonight's dir).
- **Without that flag, an absent history file strictly triggers the D5 DEGRADED state** — every
  would-be verdict becomes INSUFFICIENT-HISTORY and the nightly summary leads with the degradation
  warning. The nightly script never passes `--bootstrap`.

This makes absence a one-way loud signal: the only way history appears is a human running the
bootstrap deliberately, so a mid-week deletion can never quietly repair itself.

Why not the observatory DB: `trial_history` looks purpose-built but is schema-only with zero
writers (V13); using it means adding a Go writer to the eval-suite, a migration-coupled read path
from a shell script, and coupling to half-implemented ELO columns — that blows the 1-day box for
zero nightly-specific benefit. Recorded as Rejected Alternative R3, with the note that a future
`trial_history` implementation can seed itself from this JSONL.

Why not keep `/tmp`: it is the root cause of the detector's amnesia (V14). Any multi-night rule
built on `/tmp` silently degenerates back to tonight-vs-one-night after every reboot/cleanup.

### D2 — The guard rule: trailing-window solidity (option b), with a sustained-fail escalation backstop (option a demoted to backstop)

Per benchmark, tonight's **persistent failure** (unchanged V5 gate) is classified against the
trailing window of the last **W = 5 prior nights with data** for that benchmark (nights, not
calendar days; a night where the benchmark wasn't run contributes nothing; **records dated
`>= tonight` are excluded** per D1.2, so a rerun can never count tonight as its own history):

Let `p̂` = window passes / window trials, with minimum evidence **MIN_NIGHTS = 2** and
**MIN_TRIALS = 4**.

| Window state | Label | Rationale |
|---|---|---|
| evidence < MIN (fewer than 2 prior nights or 4 prior trials) | **INSUFFICIENT-HISTORY** | Cannot distinguish flake from break; fail safe, but LOUDLY (D5) |
| `p̂ == 1.0` (perfectly solid across the whole window) | **REGRESSION** | Solid-for-a-week → broken is the genuine signal; pages same-night, zero added latency |
| `0 < p̂ < 1` (any fail in the window) | **SUSPECTED-FLAKE** | The benchmark's own recent history proves it flips; a one-night all-fail is expected behaviour, not news |
| `p̂ == 0` (window all-fail) | **GAP** | Already-failing / never-passed; unchanged semantics |

**Escalation backstop** (so the guard stops crying wolf without going mute): a benchmark that has
now failed **all trials for K = 3 consecutive nights** escalates to REGRESSION **once** (exactly
at the 3rd night; nights 4+ do not re-fire — the window then drains toward GAP). The consecutive
count is **label-agnostic across SUSPECTED-FLAKE and INSUFFICIENT-HISTORY** (revised in quorum
round 1: the original SUSPECTED-FLAKE-only backstop left low-history chains unbounded); only a
`p̂ == 0` window (GAP, never-passed/already-failing) is excluded — that is gap-finder territory,
same as today. This makes the worst case provable: **no benchmark that has ever passed goes
unpaged past its 3rd consecutive failing night.** A same-date rerun re-derives the escalation
verdict identically but cannot re-deliver it (D1.2/V25).

**Replay on the real verified history (V9):**

- **2026-07-25** (tonight 0/2): window = 07-23 (1/2) + 07-24 (2/2) → nights=2 ✓, trials=4 ✓,
  p̂ = 3/4 = 0.75 → **SUSPECTED-FLAKE**. Consecutive all-fail nights = 1 < 3, no escalation.
  **Issue #480 is not filed.**
- **2026-07-27** (tonight 0/2): window = 07-23…07-26 → p̂ = 5/8 = 0.625 → **SUSPECTED-FLAKE**
  (07-26 passed, so consecutive all-fail = 1). **Issue #485 is not filed.**
- **Genuine-regression control** (synthetic): 5 solid nights (10/10) then 0/2 tonight →
  p̂ = 1.0 → **REGRESSION, pages the same night**. The common case — a compiler/prompt change
  breaking a reliably-passing benchmark — loses **zero** detection latency.

**The trade, named plainly:** what pays for the suppression is the *recently-flaky* (and, per
D6, the *newly-added*) benchmark. If a benchmark flips within the window and then breaks for
real, it does not page same-night; it pages via the K=3 backstop on its 3rd consecutive failing
night — **48h of added latency vs today's detector**. We accept this: for a local-model
benchmark whose recent history already contains failures, a same-night page was never
trustworthy — all four historical pages on exactly this pattern were noise (V10/V11). The full
per-class latency contract, including the two conceded classes, is D6's.

### D3 — Kill `len(seen) >= 1`: minimum evidence replaces `was_solid_in_prev`

`was_solid_in_prev` is deleted, subsumed by D2's window: a REGRESSION verdict now requires
≥ 2 distinct prior nights AND ≥ 4 prior trials, all passing. A single-trial prior night can no
longer certify "solid" on its own. Differing per-night trial counts are handled by construction —
the window pools *trials*, so a 1-trial night simply contributes less evidence toward the
MIN_TRIALS floor instead of distorting a per-night average.

### D4 — Labels and channel routing (the complete table)

| Label | Condition | Log | controlplane | Discord | GitHub issue |
|---|---|---|---|---|---|
| (pass ≥1 trial tonight) | unchanged | line | — | — | — |
| (all-fail, all infra cats) | unchanged | line | — | — | — |
| **REGRESSION** | D2 solid-window, or K=3 escalation | line | per-bench msg, `--type bug` | 1 ping | **yes** (implied by `--type bug`, V7) |
| **SUSPECTED-FLAKE** *(new)* | all-fail + mixed window | line w/ window stats | ONE aggregated msg per night, `--type note`, body carries per-bench `passes/trials over N nights` **+ consecutive-fail count toward K** (`failing 2/3 nights toward escalation`) | no | **no** |
| **GAP** | all-fail + all-fail window | line | aggregated `--type note` (unchanged, keeps "Gap-finder candidates" phrasing) | no | no |
| **INSUFFICIENT-HISTORY** *(new)* | all-fail + evidence < MIN | **LOUD** line naming the shortfall | named per-bench in the nightly summary body, with the shortfall **+ consecutive-fail count toward K** | no | no |

The GitHub-issue cut happens entirely in the script's label→`--type` mapping — no Go change
(V7). Escalated REGRESSIONs use the same `--type bug` path; their body says "escalated: 3rd
consecutive all-fail night" plus the window label it escalated from (suspected-flake or
insufficient-history) so triage knows which rule fired.

### D5 — NO SILENT FALLBACKS (mission Critical Principle 2)

Today's behaviour: missing prior run logs loudly (`nightly-eval.sh:253` — acceptable) but the
controlplane summary (`:392`) is silent about the degradation (V18 — not acceptable). New contract:

- The classifier emits a **history health line** every night:
  `history: <file> | <B> benchmarks, <N> nights, newest <date>` — into the log AND the
  controlplane summary body.
- History file absent/unreadable/corrupt → every would-be verdict becomes INSUFFICIENT-HISTORY,
  and the summary body carries
  `⚠ history unavailable (<reason>) — regression detection DEGRADED tonight` as its first line.
  The nightly never reports "Regressions: none" while secretly unable to tell.
  **Absence is not special-cased into a silent repair**: there is no automatic bootstrap
  (D1.5), so a mid-week deletion degrades loudly exactly like a corrupt file, and only an
  explicit human `--bootstrap` run can re-seed. This closes the round-2 D1/D5 contradiction.
- Benchmarks individually below the evidence floor are named in the summary
  (`insufficient history: json_parse (1 night/2 trials)`), not folded into "none".
- **Every suppressed all-fail benchmark is named every night it is suppressed**, with its window
  stats and its consecutive-fail count toward the K=3 escalation
  (`suspected flake: json_parse (5/8 over 4 nights, failing 1/3 toward escalation)`). D6's
  acceptance of delayed paging leans on this line: the delay is deliberate and *visible*, never
  silent — and it is acceptance-tested, not just promised (see Acceptance criteria).

### D6 — What genuine detection keeps, and what it concedes (acceptance-gated)

Round-1 quorum correctly caught this section overclaiming ("pages exactly as today" — false for
two benchmark classes). The precise contract, per class:

- **Solid-window benchmarks** (p̂ = 1.0 across ≥2 nights / ≥4 trials): page **same-night, exactly
  as today**. This is the common genuine case — a compiler/prompt change breaking a
  reliably-passing benchmark — and it loses zero latency. Locked by the synthetic solid→broken
  control.
- **Newly added benchmarks** *(concession)*: eligible for same-night REGRESSION only from their
  3rd night with data (2 prior nights × 2 trials meets both floors). One that passes night 1 and
  dies on night 2 is INSUFFICIENT-HISTORY — named in the summary with its escalation counter
  (D5) — and pages via the K=3 backstop on its 3rd consecutive failing night: **two nights (48h)
  later than today's detector** (the round-1 review cited 72h, counting the three failing nights
  inclusively; measured as *added* latency vs today it is 48h). Mitigating: the nightly's tier
  set derives from `benchmarks/*.yml`, so a new benchmark arrives by reviewed commit, not
  unannounced.
- **Recently recovered benchmarks** *(concession)*: any failure inside the trailing W=5 window
  demotes a new break to SUSPECTED-FLAKE; a real death pages via K=3, ≤48h added latency, until
  a clean 5-night window restores same-night paging automatically. This is the trade D2 names
  plainly — it is the exact pattern whose same-night pages were 4-for-4 noise (V10/V11).

Why we accept the bounded delay rather than the round-1 alternative (allow REGRESSION at
MIN_NIGHTS=1 when the single prior night is p̂ = 1.0): that exception recreates today's V4
flip-sensitivity precisely where evidence is thinnest. `json_parse` itself was 2/2 on 07-24
(V9) — a benchmark *added* that day would have re-filed #480 verbatim on 07-25 under the
exception. Rejected as R7. The delay we accept instead is **bounded** (label-agnostic K=3: no
ever-passing benchmark survives 3 failing nights unpaged, D2), **visible** (D5 names every
suppressed benchmark nightly with its counter — the review's word "silently" does not hold
against D5, and D5's naming is itself acceptance-tested), and **rare** (new-benchmark additions
are reviewed commits).

Locked by three acceptance tests: solid→broken same-night control; new-benchmark timeline
(pass n1, all-fail n2–n4 → INSUFFICIENT-HISTORY, SUSPECTED-FLAKE, escalated REGRESSION exactly
once at n4); flaky-then-died escalation. The guard stops crying wolf; barking is
regression-tested.

---

## Solution shape

**New file: `tools/nightly_classify.py`** (~250 LOC, stdlib only — the heredoc's logic, extracted
and extended; also the sole owner of the history file's lock + atomic-update contract, D1.1–D1.4):

```
python3 tools/nightly_classify.py \
    --tonight  /tmp/nightly_eval_YYYYMMDD_rag_on/agent \
    --history  ~/.ailang/state/nightly-eval-history.jsonl \
    --window-nights 5 --min-nights 2 --min-trials 4 --escalate-after 3 \
    --update-history

# one-off, MANUAL, at initial deployment only (D1.5) — the nightly never passes this:
python3 tools/nightly_classify.py --bootstrap --history ~/.ailang/state/nightly-eval-history.jsonl
```

stdout: the same TSV contract the shell already consumes (`LABEL\tbench\t[cats]`), plus
`HEALTH\t…` and `INSUFFICIENT\t…` lines. Tonight's date is parsed from the `--tonight` dir name
(deterministic under replay; no wall-clock dependence). Pure functions (`parse_results_dir`,
`load_window`, `classify_bench`) with the CLI as a thin shell, so the logic is unit-testable —
the heredoc today is untested (V15). Exit non-zero only on operational errors (unreadable
tonight-dir, lock-acquisition timeout per D1.3); a degraded history is a *loud classification*,
not a crash, so one flaky file can't kill the whole nightly.

**`tools/launchd/nightly-eval.sh`**: heredoc (lines 271-330) replaced by one invocation; routing
block extended per D4 (net change ≈ −60 shell lines). The script stays the single owner of
*sending*; the tool owns *classifying*. Deployment: launchd reads the on-disk script in the main
checkout — lands with the normal merge to dev + rig pull; no plist change.

**New file: `tools/test_nightly_classify.py`** (stdlib `unittest`; `make test-nightly-classifier`
target, wired into `make ci`). First Python test in `tools/` — kept dependency-free deliberately.

---

## Blast radius

What consumes the nightly classifier's output, and what changes for each:

| Consumer | Path | Change |
|---|---|---|
| **Discord** | `messages send public-feedback` → Pub/Sub → notify daemon → Discord | Fewer pings (suspected-flake never sends). No daemon/filter change — we simply don't send. |
| **GitHub issues** | `--type bug` → inline `syncMessageToGitHub` (V7) | Only REGRESSION (incl. escalations) files issues. No Go change; the `[nightly-eval]` title contract is untouched. |
| **controlplane inbox** | `messages send controlplane` (SQLite; read by agents at session start) | One NEW nightly title pattern: `Nightly eval: N suspected-flake(s) (DATE)`. Summary body gains the history-health line (D5). Title-dedup unaffected — dates keep titles unique (V16). |
| **Mission Gate-0 triage** | mission-control reads open `[nightly-eval]` issues + inbox | Triage load drops from issue-per-flip to note-skim; escalated issues carry their rule in the body so triage can distinguish them. |
| **eval-gap-finder** | GAP controlplane note ("Gap-finder candidates" phrasing) | Unchanged text contract. Suspected-flake is deliberately NOT fed to the gap-finder — a flipping benchmark is not a capability gap. |
| **`/tmp` result dirs** | tonight's trials + μRAG A/B analysis | Still written, still the source of tonight's data; only the *cross-night memory* moves to the JSONL. |
| **`nightly-lang-eval.sh`** | separate weekly script | No classifier there (V17); unaffected. |
| **Observatory DB** | `eval_baselines`, dead `trial_history` | Untouched. JSONL noted as a future seed for `trial_history` if ELO trial recording ever lands. |

---

## State & locking conflict surface (audit)

Added in quorum round 1 (gpt5-6-sol: "audit existing state-storage/locking machinery before
introducing a bespoke file"). Everything the repo/rig already has for state files and mutual
exclusion, and what this design reuses vs. deliberately avoids — each row backed by a
Verification Log entry:

| Machinery | Where | Verdict for the history file |
|---|---|---|
| **Rig GPU lock (shell)** | `tools/launchd/rig-lock.sh` — atomic `mkdir` lock (macOS has no flock) + 6h stale-steal, wait/nowait modes; `nightly-eval.sh` sources it and blocks at `:40-42` (V19/V20) | Already serializes all *scheduled* nightly runs end-to-end, so launchd-vs-launchd concurrency cannot happen. NOT reused as the history lock: wrong granularity — it guards GPU hours with a 6h staleness window, and a standalone classifier invocation must not queue behind a 4-hour eval. Its **acquisition pattern** (atomic mkdir + bounded acquisition) is reused in D1.3 with file-appropriate parameters (60 s bound); its **mtime-only stale-steal is deliberately NOT reused** — quorum round 2 established that staleness alone does not give mutual exclusion, so D1.3 gates stealing on PID liveness + an ownership token (10 min staleness survives only as the conservative recovery threshold for unreadable lock metadata). |
| **Rig GPU lock (Go)** | `internal/riglock/` — same dir/semantics + PID-liveness steal (V21) | Not linked: the classifier is Python/stdlib. PID-liveness stealing is over-engineering for a milliseconds-long critical section — mtime staleness at 10 min suffices; noted as the upgrade path if that judgment proves wrong. |
| **Atomic JSON writes** | `writeJSONAtomic`, `internal/eval_analysis/dashboard_io.go:152-199` — temp file + validate + rename (V22) | Convention **adopted** in D1.4 (Python replica: same-dir temp, fsync, `os.replace`). Not directly callable from Python, so the contract is pinned by its own unit test instead of shared code. |
| **Watermark/cursor files** | `os-filler-cursor` (`os-rotation-filler.sh:357,367`), `mission-*-last-seen` — bare `echo >` overwrites, non-atomic (V23) | Explicitly **NOT** the model: a torn cursor loses one tick and self-heals next cycle; a torn history file corrupts the detector's sole memory. |
| **SQLite stores** | `observatory.db`, `eval_baselines.db`, messaging DBs under `~/.ailang/state/` (WAL mode) | Rejected as substrate in R3 (V13: `trial_history` has zero writers). No interaction: the classifier never opens them. |
| **`messages send` title-dedup** | `cmd/ailang/messages_send.go:136-147` — refuses a duplicate title per inbox **before** SQLite insert and before GitHub sync, exit 1 (V25) | **Reused** as the exactly-once *delivery* guard for same-date reruns (D1.2): every nightly title embeds the date, so a rerun's re-derived REGRESSION/escalation send is refused and no second issue is created. |

Conclusion: nothing existing can be adopted wholesale (the two locks guard the wrong resource at
the wrong granularity; the atomic-write helper is Go-side), but both *conventions* transfer, and
the delivery-dedup guard transfers as-is. The bespoke surface is limited to ~40 lines of Python
implementing the two conventions, each pinned by tests (M2).

---

## Milestones (total ≈ 1.25 days — grew +2h in quorum round 1; stated, not absorbed)

**M1 — Extract the classifier, behavior-preserving (~2.5h)**
`tools/nightly_classify.py` replicating today's exact logic (trial grouping/dedup, V5 gate,
one-prior-night solid check) + `nightly-eval.sh` calls it + `tools/test_nightly_classify.py`
covering the v0.29 contract's fixture matrix (pass/pass, pass/fail, solid→allfail, infra-only —
finally landing that doc's pinning test, V15). Commit-able alone: pure refactor, alert behaviour
byte-identical on the TSV contract.

**M2 — Durable history + the D1 contract + loud degradation (~4h; was 2h — the round-1
atomicity/locking contract and the round-2 ownership-checked lock live here)**
JSONL read-modify-write with (date, bench, model, arm) key-replace (D1.1), strict
date-before-tonight window exclusion (D1.2), bounded mkdir lock with **PID + ownership-token**
liveness checking (D1.3), temp + fsync + `os.replace` writes (D1.4), `os.makedirs(...,
exist_ok=True)`, the explicit `--bootstrap` flag (D1.5 — and NO automatic bootstrap on absence),
D5 health/degradation lines. Classification rule still M1's. Tests: same-date rerun replaces
records + yields byte-identical verdicts; pre-seeded duplicate keys resolve last-wins and compact;
concurrent invocation (held lock → bounded wait → loud non-zero exit, file never torn);
**stale-but-ALIVE holder is not stolen from** (waiter fails loudly at the 60 s bound);
**old holder resuming after its lock was replaced** detects the token mismatch and does not write;
**ownership-checked release cannot delete another process's lock**; interrupted write (kill
between temp-write and rename → prior history intact, stray temp cleaned next run); corrupt line
skipped + counted in health line; state dir absent → created; **history absent without
`--bootstrap` → DEGRADED, not auto-seeded**.

**M3 — The guard rule + labels + routing (~3h; was 2.5h — +escalation/rerun edge tests)**
D2 window rule (W=5, MIN 2/4, label-agnostic K=3 escalation), D3 removal of `was_solid_in_prev`,
D4 routing in the shell (suspected-flake → aggregated `--type note` with escalation counters;
escalation body text). Tests: each row of the D4 table on synthetic windows; escalation fires
exactly once across nights; same-date rerun after an escalation re-derives the same verdict
(delivery dedup is V25's, asserted at the title-convention level); current-date records present
in history do not alter the window; new-benchmark timeline (D6).

**M4 — Real-history replay + CI wiring (~1h)**
The verified V9 `json_parse` history embedded as a fixture; asserts 07-25 and 07-27 classify
SUSPECTED-FLAKE (i.e. #480/#485 suppressed) and the synthetic solid→broken control still
REGRESSES. `make test-nightly-classifier` added to `make ci`. CHANGELOG entry.

Total: 2.5 + 4 + 3 + 1 = **10.5h ≈ 1.3 working days**. The original "~1 day" is no longer
honest after the round-1 contract work and the round-2 ownership-checked lock; rather than
compress the new atomicity tests to fit, the estimate moves (again, in the open). If M2 still
overruns, the cut is M4's CI wiring (slips to a follow-up commit) — never the contract tests.

---

## Acceptance criteria

- [ ] Replay of the real banked history (V9 fixture): 2026-07-25 → SUSPECTED-FLAKE and
      2026-07-27 → SUSPECTED-FLAKE — neither files a GitHub issue nor pings Discord.
- [ ] Solid→broken control: 5 solid nights then all-fail → REGRESSION **the same night** (no
      added latency for the genuine case), `--type bug` path unchanged.
- [ ] Flaky-then-died: mixed window then 3 consecutive all-fail nights → exactly ONE escalated
      REGRESSION (night 3; nights 4+ do not re-fire).
- [ ] New-benchmark timeline (D6 concession, pinned): pass n1, all-fail n2–n4 →
      INSUFFICIENT-HISTORY (named in summary with counter), SUSPECTED-FLAKE, escalated
      REGRESSION exactly once at n4 — the ≤48h added latency is documented behaviour, not an
      accident.
- [ ] Same-date rerun idempotency: classifier run twice over the same tonight-dir →
      byte-identical verdicts (including any escalation), history holds exactly one record per
      (date, bench, model, arm), and the rerun's `--type bug` send is refused by title-dedup
      before insert/GitHub sync (V25) — no second issue.
- [ ] Current-date exclusion: with tonight's records already present in history (rerun case),
      window statistics are unchanged — only `date < tonight` records count.
- [ ] Concurrent invocation: with the lock held by another process, a second `--update-history`
      waits bounded (≤60 s), then exits non-zero LOUDLY; the history file is never torn.
- [ ] Lock ownership (D1.3, round-2): a holder past the 10-min staleness threshold but **still
      alive** (`os.kill(pid, 0)` succeeds) is NOT stolen from — the waiter fails loudly at the
      60 s bound; an **old holder resuming after its lock was replaced** sees the token mismatch
      and refuses to write; **ownership-checked release cannot delete another process's lock**.
- [ ] Interrupted write: simulated crash between temp-write and rename → prior history byte-intact;
      stray temp removed on the next locked update.
- [ ] Fresh machine: history parent dir absent → created (`makedirs exist_ok`); the nightly run
      (no `--bootstrap`) reports DEGRADED and exits 0; a subsequent explicit
      `--bootstrap` run seeds from `/tmp` and exits 0.
- [ ] No auto-heal (D1.5, round-2): history deleted mid-week with `/tmp` dirs still present →
      the next nightly run does **not** re-seed itself; it degrades loudly. Only an explicit
      `--bootstrap` invocation restores history.
- [ ] SUSPECTED-FLAKE reaches controlplane as `--type note` only — grep of the routing block
      shows no `--type bug`/`--github` on that path (issue-creation gate per V7).
- [ ] History file deleted mid-week → next run's summary body leads with the degradation warning
      and names each INSUFFICIENT-HISTORY benchmark; log line present; exit code 0.
- [ ] Single-trial prior night alone can no longer produce a REGRESSION verdict (MIN_TRIALS=4
      test — the V4 weakness).
- [ ] `make test-nightly-classifier` green in `make ci`; `shellcheck tools/launchd/nightly-eval.sh`
      still clean.
- [ ] Mission doc: V8 correction recorded (the "compare like-for-like CONDITIONS" queue bullet is
      wrong and is dropped when this doc lands).

---

## Rejected alternatives

- **R1 — Sustained N-night drop as the primary rule (option a).** Requiring N consecutive all-fail
  nights before ANY page adds 24–48h latency to *every* genuine regression — including the common
  solid→broken case that same-night paging exists for. It also still pages on a slow bimodal
  (any flake whose bad phase spans N nights). D2 gets the suppression without the latency by
  making *history solidity*, not *failure persistence*, the discriminator; the sustained rule
  survives only as the K=3 escalation backstop for flaky-then-died.
- **R2 — Binomial/confidence test over trials × nights (option c).** Powerless at the actual
  trial count: with 2 trials/night and a window estimate p̂ = 0.75 (the real 07-25 value),
  P(0 passes | p̂) = 0.25² = 0.0625 > 0.05 — the test **cannot reject at α = 0.05 even once** at
  n=2; it would demand ≥3 trials to ever fire at all. It degenerates into "mixed window → cannot
  conclude", which is exactly D2 without the statistics theater, plus tunable-α bikeshed.
- **R3 — Observatory `trial_history` as the substrate.** The table exists but nothing writes it
  (V13); adopting it means new Go writer wiring in the eval-suite, schema/migration coupling to
  half-built ELO columns, and DB reads from a shell context (against the repo's
  use-the-CLI-not-raw-SQLite rule). Out of the 1-day box; JSONL can seed it later.
- **R4 — Suppress by muting Discord only, keep filing issues.** The GitHub issue IS the main
  cost (#417 optics + Gate-0 triage slots); a quieter ping with the same issue spam fixes nothing.
- **R5 — Raise nightly trials (e.g. `--trials 4`) so one night is self-sufficient.** Doubles GPU
  wall-clock every night to buy variance data the JSONL already accumulates for free across
  nights; and 0/4 on a bimodal benchmark still flips (07-27 was 0/4 pooled across arms, V9).
- **R6 — Per-benchmark allowlist of known flakes.** Hand-maintained state that rots; the window
  rule derives the same fact from data, benchmark-agnostically, and self-heals when a flake
  stabilizes (a clean 5-night window restores same-night paging automatically).
- **R7 — REGRESSION at MIN_NIGHTS=1 when the single prior night is p̂ = 1.0** (proposed by
  gemini-3-1-pro in quorum round 1 as an alternative to accepting the new-benchmark delay).
  Rejected: it re-creates the V4 single-night-certifies-solid flip-sensitivity exactly where
  evidence is thinnest — a 2-trial first night "proving" solidity is the failure mode this doc
  exists to remove. Concretely: `json_parse` was 2/2 on 2026-07-24 (V9); a benchmark *added* that
  day would have re-filed #480 verbatim on 07-25 under this exception. The bounded, visible K=3
  path (D6) is preferred; the reviewers' other branch (acknowledge the window concretely) is the
  one taken.

## Non-goals

- No change to what counts as a pass (`compile_ok && runtime_ok && stdout_ok`), to
  `INFRA_CATEGORIES` membership (V15's missing `stream_death` is noted for the mission backlog,
  not smuggled in here), to trial count, tiers, or the μRAG A/B schedule.
- No investigation of *why* `json_parse` is bimodal on this model (model-side PAR_UNEXPECTED_TOKEN
  per #485 triage; that is eval-gap territory, and per the program's routing bias it stays in the
  AILANG-fix/extension lanes, not this harness fix).
- No cloud/multi-model generalization: the nightly runs one local model; the JSONL schema carries
  `model` so a future multi-model nightly extends without migration.

## Axiom compliance

Harness/tooling change — no language-semantics surface, so the 12-axiom matrix is scored on the
process axioms only (matching the precedent set by the v0.29 detector-contract doc):
A1 Determinism +1 (classification becomes a pure function of tonight-dir + history file);
A2 Replayability +2 (the real incident history is an executable fixture);
A7 Machines First +1 (false pages train triage to ignore pages; suppression restores signal);
A11 Structured Failure +1 (two new explicit labels where "regression" over-claimed);
A12 System Boundary +1 (the detector owns a durable memory instead of leaning on `/tmp`).
No hard violations; net ≥ +2. ✅

## Quorum verification log

**Round 1 (2026-07-27): BLOCKED** by both reviewers, `gpt5-6-sol` and `gemini-3-1-pro`
(author claude-fable-5 excluded under generator ≠ judge; controller's in-session verdict was
PASS). Both accepted the design direction; both objections were to specifics. Neither was
contested — repo evidence confirmed both.

| # | Objection (reviewer) | Resolution in this revision |
|---|---|---|
| Q1 | **gpt5-6-sol**: append-only JSONL had no idempotency, locking, or atomicity contract — a same-date rerun appends duplicates, can count tonight as its own prior history, and can re-fire the K=3 escalation; concurrent/partial writes can corrupt the sole history source; no audit of existing state/locking machinery was done. | **Accepted in full.** D1 rewritten around an explicit contract (D1.1–D1.4): unique key (date, bench, model, arm) with same-key **replace** on rerun + deterministic last-wins duplicate resolution; window strictly `date < tonight` (D1.2 — a rerun can never pollute its own window); bounded mkdir lock (60 s bound, 10 min stale-steal) reusing the `rig-lock.sh`/`internal/riglock` convention (V19–V21); temp + fsync + `os.replace` writes per the `writeJSONAtomic` convention (V22). New **State & locking conflict surface (audit)** section with per-row verification (V19–V26). Exactly-once *delivery* on rerun anchored on the pre-insert title-dedup (V25) — verdicts re-derive identically, sends are refused. All six requested test classes added to M2/M3 + Acceptance criteria. **Estimate moved ~1 day → ~1.25 days (+2h), stated rather than absorbed.** One precision note (not a contest): `AILANG_NIGHTLY_EVAL_DRY_RUN` exits before the classifier (V26), so the real rerun vector is manual full invocation — covered either way. |
| Q2 | **gemini-3-1-pro**: D6 falsely claimed solid benchmarks page "exactly as today" — the MIN_NIGHTS=2/MIN_TRIALS=4 floors delay newly-added (and recently-recovered) benchmarks by up to 72h, contradicting D3/D6; and D1's "the `state/` dir already exists" was unverified (FileNotFoundError risk on fresh deploy). | **Accepted; took their branch 1** (acknowledge the window concretely) and rejected branch 2 as **R7** (the MIN_NIGHTS=1 p̂=1.0 exception re-creates V4 flip-sensitivity where evidence is thinnest — json_parse's own 07-24 night proves it, V9). D6 rewritten as a per-class contract: solid-window = same-night as today; newly-added / recently-recovered = page on the 3rd consecutive failing night (**48h added latency vs today** — their 72h counted the failing nights inclusively). The K=3 backstop is now **label-agnostic** over SUSPECTED-FLAKE + INSUFFICIENT-HISTORY (D2), which bounds the worst case their scenario exposed — tighter than the round-0 design. The delay is not silent: D5 now guarantees (and acceptance-tests) nightly naming of every suppressed benchmark with its escalation counter. On `state/`: premise **verified true on this rig** (V24, controller-checked) AND their defensive `makedirs exist_ok` fix is adopted in M2 for fresh machines — both, not either. |

**Round 2 (2026-07-27): BLOCKED** by both reviewers again — on *new, narrower* objections, not on
the round-1 resolutions (round 1's were accepted). Both again accepted the design direction; both
supplied a fully-specified `proposed_fix`. Resolved by the mission's **narrow-refinement carve-out**
(ratified iter-98): the controller applied each reviewer's fix **verbatim to their own
specification** — no controller-invented resolution, no objection overridden. Applied by the
**controller (opus)**, not the designer, because both fixes were transcription rather than
judgment; recorded here as the carve-out requires.

| # | Objection (reviewer) | Fix applied — VERBATIM to the reviewer's specification |
|---|---|---|
| Q3 | **gpt5-6-sol**: "D1.3's mtime-only stale-steal does not guarantee mutual exclusion. A live classifier paused for more than 10 minutes can have its lock stolen, resume concurrently with the new owner, and overwrite that owner's read-modify-write; the old process may also remove the replacement owner's lock unless release is ownership-checked." Their catch: staleness-at-10-min is *an unsupported safety premise, not merely an implementation choice*; `os.replace` prevents torn files but not lost updates. | **Accepted in full — this was also the controller's own named residual risk in the round-2 note, so both judges converged on it.** D1.3 rewritten to their spec: the lock dir stores **PID + a random ownership token**; steal **only when the holder PID is confirmed dead** via `os.kill(pid, 0)`; a **separately specified conservative recovery rule** covers unreadable metadata (treat as possibly-live; steal only past the staleness threshold, logged at WARN; revert to PID-liveness if metadata becomes readable); the **token is verified after acquisition, before entering the critical section**; **release removes the lock only if the token still matches**. The 60 s bounded wait is kept and a still-live holder **fails loudly**. All three of their named tests added to M2 + Acceptance (stale-but-alive not stolen; old holder resuming after replacement refuses to write; ownership-checked release cannot delete another process's lock). Estimate moved again in the open: M2 3.5h → 4h, total 1.25d → **1.3d**. |
| Q4 | **gemini-3-1-pro**: "Logical contradiction and axiom violation (No Silent Fallbacks) between D1 and D5 regarding file absence… Because the script cannot distinguish a 'first run' from a 'mid-week deletion' purely by file absence, a deleted file mid-week will trigger D1 and silently auto-heal from remaining `/tmp` directories. This completely bypasses the D5 degradation warning and breaks the 'History file deleted mid-week' acceptance test." | **Accepted in full — a genuine self-contradiction in the doc, caught against its own acceptance criterion.** Their fix applied exactly: **automatic bootstrap on file absence is REMOVED** from D1; an explicit **`--bootstrap` CLI flag** is introduced, *manual, once, at initial deployment*, and the nightly script never passes it; **without it an absent history file strictly triggers the D5 DEGRADED state**. Recorded as **D1.5**, cross-referenced from D5, and pinned by a new acceptance test (history deleted mid-week with `/tmp` dirs still present → degrades loudly, does **not** re-seed itself). The "fresh machine" acceptance row was corrected accordingly — it previously asserted the auto-seed this objection deletes. |

**Carve-out conditions checked before applying** (both must hold for every remaining objection,
else the doc parks for a human): (a) each objection carries a concrete **reviewer-authored**
`proposed_fix` — yes, both are prescriptive to the field level; (b) neither disputes the design
**direction** — Q3 is the correctness of a locking mechanism, Q4 an internal contradiction between
two of the doc's own clauses. Both are completeness/determinism defects of exactly the kind the
carve-out exists for. Trajectory supports it: round 1 objected that there was *no* state contract
at all; round 2 objects to one rule *inside* that contract. The objections are converging, not
recycling.

---

## Related documents

- [m-eval-regression-detector-contract](../../implemented/v0_29_0/m-eval-regression-detector-contract.md) — the single-night contract this extends (and whose pinning test this finally lands)
- [m-eval-stream-health-retry](../../planned/v0_29_0/m-eval-stream-health-retry.md) — infra-noise taxonomy neighbor
- `design_docs/v1-mission.md:1231` — the queue row (with the V8 correction)
- GitHub: #286, #292, #480, #485 (the four noise issues), #417 (external-optics cost)
