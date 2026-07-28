# Sprint Plan: M-NIGHTLY-FLAKE-GUARD

**Design doc**: [m-nightly-flake-guard.md](m-nightly-flake-guard.md) (LANDED, quorum-cleared 2 rounds, commit `6ad39b863`)
**Sprint ID**: `M-NIGHTLY-FLAKE-GUARD`
**Branch**: `sprint/m-nightly-flake-guard`
**Worktree**: `/Users/voightkampff/dev/sunholo-data/ailang/.claude/worktrees/sprint-m-nightly-flake-guard`
**Base**: `origin/dev` @ `fedfdb4e0`
**Planner model**: `claude-opus-4-8`
**Planned**: 2026-07-28 (mission-control iteration 113)
**Estimate**: **~1.4 working days (11.5h)** — up from the doc's 1.3d; the delta is justified below (§Estimate delta), not absorbed.
**Risk**: medium (a *classifier* change with no compiler surface, but the guard is load-bearing for the mission's Gate-0 signal, and one refuted CI premise would have made the whole thing unenforced)

---

## 0. Premise re-verification (first-party, this session, at `fedfdb4e0`)

The controller handed me a labelled premise set and asked me to re-verify the load-bearing ones.
**I confirmed 6 and refuted or materially corrected 6.** Refutations are stated loudly, per the
mission's iteration-112 standard.

### CONFIRMED

| # | Premise | How verified |
|---|---|---|
| C1 | `nightly-eval.sh:244-249` globs `/tmp/nightly_eval_*_rag_on/agent`, keeps the LAST non-today match | Read `tools/launchd/nightly-eval.sh:240-254` |
| C2 | The classifier is an inline `python3` heredoc starting at `:271`, ending `:331` | Read `:271-332` |
| C3 | `was_solid_in_prev` certifies "solid" from a single trial (`return len(seen) >= 1 and all(seen)`) — **line 326 exactly as the doc claims** | Read `:315-326` |
| C4 | Only `--type bug`/`--type feature` reach GitHub. `knownGitHubTypes = {bug, feature}`; `syncToGitHub := *github \|\| knownGitHubTypes[category]`. `--type` defaults to `""`. So `--type note` and the un-typed Discord send create **no issue** | Read `cmd/ailang/messages_send.go:40-41,99-103` |
| C5 | Both sides of the comparison already use the **rag_on** arm — the queue's "compare like-for-like CONDITIONS" bullet is a phantom, correctly dropped (doc V8) | `nightly-eval.sh:233` (`RESULTS_AGENT="${RESULTS_DIR}_rag_on/agent"`) + the `*_rag_on/agent` glob at `:245`. **Do not design for it.** |
| C6 | ZERO **open** `[nightly-eval]` issues right now | `gh issue list --search '"Nightly regression" in:title' --state all` → 45 results, `OPEN: 0` |

### REFUTED / MATERIALLY CORRECTED

**R-A. "Four false alarms on the SAME benchmark" understates the problem by ~11x.**
There are **45** `[nightly-eval] Nightly regression:` issues across **24 distinct benchmarks**, and
**all 45 are closed**. `json_parse` accounts for 4 of 45 (9%). The design doc's problem statement
("4 GitHub issues from one bimodal benchmark") is *true but unrepresentative* — the real defect is a
systemic ~45-issue false-alarm stream. This does not change the fix; it strengthens the case and it
**changes the acceptance corpus** (see R-B). Command:
`gh issue list --search '"Nightly regression" in:title' --state all --limit 100 --json number,title,state`.

**R-B. Two MORE false alarms fired *after* the design doc was written — #499 `contract_bst_validate`
and #500 `list_comprehension`, both created 2026-07-28, both closed ~4 minutes later.** Plus #483
`higher_order_functions` (07-26). The doc's replay corpus (json_parse only) is therefore stale AND
too narrow. The real, live corpus with surviving `/tmp` data is **5 issues over 4 nights**
(#480, #483, #485, #499, #500). M4 uses all five.

**R-C. THE DOC'S HEADLINE ACCEPTANCE CRITERION IS NO LONGER REPRODUCIBLE FROM LIVE DATA.**
The doc's AC says *"2026-07-25 → SUSPECTED-FLAKE"*, derived from a window of
07-23 (1/2) + 07-24 (2/2). **`/tmp/nightly_eval_20260723_rag_on/agent` is now EMPTY** (0 files;
dir mtime `Jul 28 00:00`), as is `20260722`. First-party measurement of the reaper:

| dir | files emptied at | age of files at reaping |
|---|---|---|
| `nightly_eval_20260722_rag_on/agent` | Jul 27 00:00 | ~5 days |
| `nightly_eval_20260723_rag_on/agent` | Jul 28 00:00 | ~5 days (created Jul 23 05:35) |
| `nightly_eval_20260724_rag_on/agent` | still present (84 files) | 4.3 days |

So `/tmp` retains **at most 5 nights of file data**, i.e. **at most 4 *prior* nights**, swept at
midnight. Consequences, all load-bearing:
1. Re-deriving the 07-25 window today gives **1 night / 2 trials → INSUFFICIENT-HISTORY**, not
   SUSPECTED-FLAKE. Both suppress the issue, so the *outcome* the doc wants still holds — but the
   *label* in the doc's AC does not. **The AC must be restated** (done in M4 below) or the fixture
   must hardcode the doc's V9 table, which is now **unverifiable** (source data destroyed).
   I chose: hardcode the doc's V9 table as a *documented-provenance* fixture AND add the live
   re-derivation as a separate, honest AC.
2. `--bootstrap` from `/tmp` can **never** fill the W=5 window — the ceiling is 4 prior nights.
   The doc does not say this. It must appear in the migration note.
3. The doc's V14 ("only 6 nights survive") and the controller's "7 dirs today" are both about
   *directories*; the directories outlive their *contents* by ~a week. Counting dirs overstates
   available history. Measure files, not dirs.

**R-D. `make ci` IS NOT WHAT GITHUB ACTIONS RUNS. The doc's M4 CI wiring would have been vacuous.**
`.github/workflows/ci.yml` never invokes `make ci`; the `test` job runs individual targets
(`make test-parser`, `make check-file-sizes`, `make check-boundaries`, `make test-regression-guards`,
`make doctor`, …). Adding `make test-nightly-classifier` to `make ci` (`make/ci.mk:11`) alone gives
**zero enforcement** — exactly the class of vacuous acceptance criterion the controller warned about
(iteration 110). M4 therefore requires a **step in `ci.yml`'s `test` job**, and an anti-skip assertion
modelled on the existing precedent at `ci.yml:86-91` (which greps `--- PASS: $t` to prove gated tests
actually ran rather than silently skipping).

**R-E. `shellcheck` is not in the Makefile or any workflow.** The doc's AC "`shellcheck
tools/launchd/nightly-eval.sh` still clean" is a **local-only, non-enforced** check. Keep it as a
manual gate; do NOT claim it as CI coverage. (`shellcheck` IS installed at `/opt/homebrew/bin/shellcheck`.)

**R-F. The doc's exactly-once-delivery guarantee has a hole the quorum did not find.**
`InboxMessageExistsByTitle` (`internal/messaging/inbox.go:559-574`) excludes messages whose status is
`deleted` or `archived`:
```sql
WHERE to_inbox = ? AND title = ? AND status NOT IN ('deleted','archived')
```
`ailang messages ack` sets status to `read` (not archived), so the guard survives the ordinary flow —
**but any archive/delete of a nightly controlplane message re-opens the duplicate window**, and a
same-date rerun would then file a second GitHub issue. D1.2/V25 states this guard unconditionally.
Severity: low probability, non-silent (a duplicate issue is visible), but it means the
"exactly-once delivery" claim is conditional. Recorded here; **not** in scope to fix (that is a
`messages send` change, a different lane). M3's test asserts the guard at the *title-convention*
level only, and the plan says so.

### Additional first-party finding (not in the controller's set)

**F-1. The `_trial2` grouping bug is real but historical.** 7 of the 45 issues are titled
`json_parse_trial2`, `binary_tree_sum_trial2`, `explicit_state_threading_trial2` — i.e. the
`_trial\d+` stripping failed and each trial was classified as its own single-trial benchmark. All 7
are dated **2026-06-05 … 06-07**, before the 06-12 fix. This makes trial-grouping a **required M1
pinning test**, not an optional one: the extraction must not regress it.

---

## 1. The evidence that decides this sprint: a live replay of the proposed rule

I implemented D2 (W=5, MIN_NIGHTS=2, MIN_TRIALS=4, label-agnostic K=3, GAP excluded) as a throwaway
script and ran it over the **real surviving banked history** (`/tmp/nightly_eval_2026072{4..8}_rag_on/agent`,
84 files/night, 42 benchmarks, model `opencode-qwen3-5-35b-a3b-mxfp8`). Every persistent failure
(≥2 trials, all failed, ≥1 non-infra category) for every night:

| date | benchmark | TODAY's detector | PROPOSED rule | window | consec |
|---|---|---|---|---|---|
| 07-25 | json_parse | **REGRESSION → #480** | INSUFFICIENT-HISTORY | 2/2 over 1n | 1/3 |
| 07-26 | higher_order_functions | **REGRESSION → #483** | **REGRESSION (still pages, same night)** | 4/4 over 2n | 1/3 |
| 07-26 | cli_args | GAP | GAP | 0/4 over 2n | 3/3 (GAP-excluded, no escalation) |
| 07-26 | csv_to_json_converter | GAP | SUSPECTED-FLAKE | 1/4 over 2n | 2/3 |
| 07-27 | json_parse | **REGRESSION → #485** | SUSPECTED-FLAKE | 4/6 over 3n | 1/3 |
| 07-27 | csv_to_json_converter | GAP | SUSPECTED-FLAKE | 1/6 over 3n | **3/3 → ESCALATES to REGRESSION** |
| 07-28 | contract_bst_validate | **REGRESSION → #499** | SUSPECTED-FLAKE | 4/8 over 4n | 1/3 |
| 07-28 | list_comprehension | **REGRESSION → #500** | SUSPECTED-FLAKE | 7/8 over 4n | 1/3 |
| 07-28 | json_parse | GAP | SUSPECTED-FLAKE | 4/8 over 4n | 2/3 |

**Net effect over 07-25…07-28: 5 issues filed today → 2 issues under the guard (a 60% reduction),
and the guard is provably non-vacuous three ways:**

1. **It still pages same-night on a REAL solid→broken case.** #483 (`higher_order_functions`,
   07-26, p̂ = 4/4 = 1.0) fires REGRESSION under the new rule with **zero added latency**. The design
   doc only offered a *synthetic* control; there is a **real one in the banked data**. This is the
   single most important acceptance fixture in the sprint — a guard that suppresses everything is as
   broken as no guard.
2. **It files a NEW issue today's detector never filed.** `csv_to_json_converter` escalates via K=3
   on 07-27. So the guard is not purely subtractive; the escalation backstop demonstrably barks.
3. **GAP exclusion is load-bearing, and it works.** `cli_args` hits consec=3 on 07-26 but its window
   is 0/4 → GAP → correctly NOT escalated. Without the exclusion, never-passed benchmarks would page.

**The guard's sharpest cost, named plainly:** #500 `list_comprehension` had a window of **7/8 over 4
nights** (p̂ = 0.875) and is suppressed to SUSPECTED-FLAKE. That is the closest thing to a genuine
regression in the corpus, and the guard mutes it for up to 48h. This is the D6 concession made
concrete — it is not hypothetical, it happened last night. The plan surfaces it rather than hiding it.

Reproduce with the M4 replay harness (see M4 AC).

---

## 2. Required design refinement (one, small, flagged for the controller)

**The doc's "escalate exactly at the 3rd night" (`consec == K`) is fragile to a missed night.**
If the classifier crashes, the rig is down, or the send fails on precisely the Kth consecutive
failing night, the escalation is **lost forever** (night 4 has `consec == 4 != 3`), and the doc's
own worst-case guarantee — *"no benchmark that has ever passed goes unpaged past its 3rd consecutive
failing night"* — silently breaks.

**Refinement (implement this, not the literal `== K`):** escalate when

```
consec >= K  AND  no record inside the current consecutive-failure run already has class == "REGRESSION"
```

Both operands are pure functions of (tonight-dir, history-minus-tonight): the JSONL schema **already
carries a `class` field** (`{"...","class":"suspected-flake"}`), so "did we already escalate in this
run?" is derivable without new state. This still fires exactly once, is replay-deterministic, and is
robust to a missed night. It is strictly stronger than the doc's rule and does not change any
verdict in the live replay above (`csv_to_json_converter` still escalates once, at 07-27; 07-28 does
not re-fire because 07-27's record carries `class: regression`).

**Deviation flag**: this is a narrow strengthening of D2's escalation clause, in service of D2's own
stated guarantee. It does not touch the design direction, the window rule, the labels, or the
routing. Controller: if you want the doc's literal `== K` instead, say so before M3 — but then the
"never unpaged past night 3" claim must be softened in the doc.

---

## 3. Milestones

Four milestones, each independently commit-able, in dependency order. **Every acceptance criterion
below names an exact command and an exact observable, and every one can FAIL.**

---

### M1 — Extract the classifier, behavior-preserving (~2.5h, ~260 LOC + ~180 test LOC)

Create `tools/nightly_classify.py` replicating **today's exact logic** (trial dedup by
`(slot, newest-ts)`, `_trial\d+` stripping, `INFRA_CATEGORIES = {api_error, timeout, executor_error}`,
persistent-failure gate ≥2 trials/all-fail/≥1 non-infra cat, `was_solid_in_prev` one-prior-night
check). Replace the heredoc at `nightly-eval.sh:271-332` with one invocation. Create
`tools/test_nightly_classify.py` (stdlib `unittest`, no third-party deps).

Structure the module as **pure functions + thin CLI** so M2/M3 can extend without rewriting:
`parse_results_dir(path) -> {bench: [(ok, cat)]}`, `classify_bench(...)`, `main(argv)`.
stdout contract unchanged: `LABEL\tbench\t[cat1,cat2]`, one line per persistent failure, sorted.

**Acceptance criteria (each must be able to fail):**

| # | Command | Exact observable |
|---|---|---|
| M1.1 | `python3 tools/test_nightly_classify.py -v` | rc=0; output contains `--- PASS` for ≥8 named tests (fixture matrix: pass/pass, pass/fail, solid→allfail, all-infra-cats-only, single-trial-tonight-not-persistent, missing-prior-dir, empty-prior-dir, trial-grouping) |
| M1.2 | **Golden equivalence against the live heredoc.** Run the *pre-change* heredoc and the new tool over each of the 5 surviving nights and diff: `for d in /tmp/nightly_eval_2026072{4,5,6,7,8}_rag_on; do ... done` (harness script committed at `tools/testdata/nightly_classify/compare_legacy.sh`) | For all 5 nights, `diff <(legacy) <(new)` is **empty**. Any nonempty diff fails M1. This is the behavior-preservation proof. |
| M1.3 | Trial-grouping pin (F-1 regression): fixture dir with `json_parse_ailang_M_1.json` (fail) + `json_parse_trial2_ailang_M_2.json` (fail) | Output contains exactly one line, for bench `json_parse`; **no line mentions `json_parse_trial2`**. (7 real issues in June were caused by this exact break.) |
| M1.4 | `shellcheck tools/launchd/nightly-eval.sh` (local; **NOT** CI-enforced per R-E) | rc=0, no new warnings vs. the pre-change baseline captured in the commit message |
| M1.5 | `bash -n tools/launchd/nightly-eval.sh` and `AILANG_NIGHTLY_EVAL_DRY_RUN=1 tools/launchd/nightly-eval.sh` | rc=0 (dry-run still exits before the eval per doc V26) |

**Risk**: the legacy heredoc reads `$RESULTS_AGENT`/`$PREV_RESULTS` from env; the new tool takes
flags. M1.2's comparison harness must pass the same paths both ways or the diff is meaningless.

---

### M2 — Durable history + the D1 contract + loud degradation (~4h, ~180 LOC + ~220 test LOC)

Implement D1.1–D1.5 in `tools/nightly_classify.py`. Classification rule is still M1's — only the
*substrate* changes here, so a failure in M2 cannot be confused with a failure in M3.

- **D1.1** unique key `(date, bench, model, arm)`; `--update-history` = load → drop tonight's keys →
  append tonight's → atomic write-back. Same-key duplicates found on load resolve **last-in-file-wins**
  and are compacted away.
- **D1.2** window = records with `date < tonight`, where tonight's date is parsed from the
  `--tonight` dir name (`nightly_eval_(\d{8})_rag_on`), **never** wall clock.
- **D1.3** ownership-checked lock at `~/.ailang/state/nightly-eval-history.lock.d`, implemented
  **verbatim to gpt5-6-sol's round-2 `proposed_fix`**: atomic `os.mkdir`; lock dir stores PID + a
  random ownership token; steal **only** when `os.kill(pid, 0)` confirms the holder dead; conservative
  recovery for unreadable/truncated metadata (treat as possibly-live → steal only past a 10-min
  staleness threshold, log the steal + reason at WARN, revert to PID-liveness if metadata becomes
  readable); **verify token after acquisition, before the critical section**; **release only if the
  token still matches**. 60 s bounded wait, then **fail loud** (non-zero exit).
- **D1.4** write to `<history>.tmp.<pid>` in the same dir → `flush` → `os.fsync` → `os.replace`.
  Stray temps removed on the next locked update. Unparseable lines skipped **individually and
  counted** into the health line.
- **D1.5** `--bootstrap` is an explicit one-off flag. **The nightly script never passes it.**
  Absent history without `--bootstrap` ⇒ D5 DEGRADED, never an auto-seed.
- **D5** health line `history: <file> | <B> benchmarks, <N> nights, newest <date>, <S> skipped lines`
  emitted as `HEALTH\t…` on stdout and echoed into log + controlplane summary body; on
  absent/unreadable/corrupt history, first summary line is
  `⚠ history unavailable (<reason>) — regression detection DEGRADED tonight`.
- `os.makedirs(os.path.dirname(history), exist_ok=True)` (gemini-3-1-pro's round-1 fix, adopted for
  fresh machines even though V24/first-party confirms `~/.ailang/state/` exists on this rig).

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M2.1 | `python3 tools/test_nightly_classify.py -v -k History` | rc=0; `--- PASS` for: `test_same_date_rerun_replaces_records`, `test_rerun_yields_byte_identical_verdicts`, `test_preseeded_duplicate_keys_last_wins_and_compacts`, `test_tonight_records_do_not_enter_own_window` |
| M2.2 | **Concurrency, real subprocesses** (not mocks): `-k Lock` | `--- PASS` for: `test_held_lock_waiter_waits_bounded_then_exits_nonzero` (asserts rc≠0 AND elapsed ≥ bound-ε AND ≤ bound+5 s AND history file byte-identical before/after); `test_stale_but_alive_holder_is_not_stolen_from` (holder sleeps > 10 min *simulated* by back-dating the lock mtime while the PID is genuinely alive → waiter must FAIL LOUDLY, must NOT steal); `test_old_holder_resuming_after_replacement_detects_token_mismatch_and_does_not_write`; `test_ownership_checked_release_cannot_delete_another_process_lock`; `test_unreadable_lock_metadata_conservative_recovery` |
| M2.3 | **Crash safety**: `-k Atomic` | `--- PASS` for `test_kill_between_tmpwrite_and_rename_leaves_prior_history_byte_intact` (SHA-256 of the pre-crash file equals post-crash) and `test_stray_temp_removed_on_next_locked_update` |
| M2.4 | **No auto-heal (D1.5)**: in a sandboxed HOME with `/tmp/nightly_eval_*` fixtures present, delete the history file, run the classifier the way the nightly runs it (no `--bootstrap`) | stdout contains `history unavailable`; **history file still does not exist afterwards**; **zero** `REGRESSION` lines emitted; **rc=0** (a degraded history is a loud classification, not a crash) |
| M2.5 | Then run `--bootstrap` in the same sandbox | rc=0; history file now exists; `wc -l` ≥ 1 record per (night × persistent-failure bench) found in the fixture dirs; a second `--bootstrap` is idempotent (byte-identical file) |
| M2.6 | Fresh machine: `HOME=$(mktemp -d)` (no `.ailang/state/`), run the nightly path | rc=0; `~/.ailang/state/` created; DEGRADED reported; no traceback in stderr |
| M2.7 | Corrupt-line tolerance: prepend `{not json` and a truncated line to a valid history | rc=0; HEALTH line reports `2 skipped lines`; verdicts identical to the uncorrupted run; next `--update-history` drops both lines |

**Why M2.2's "stale-but-ALIVE" test is the one that matters**: it is the *only* test that can fail if
the implementer reverts to the mtime-only stale-steal that quorum round 2 rejected. If that test is
weak (e.g. mocks `os.kill`), the round-2 objection is unresolved in practice. It must use a real
child process.

---

### M3 — The guard rule + labels + routing (~3h, ~140 LOC + ~200 test LOC)

D2 window rule, D3 deletion of `was_solid_in_prev`, D4 routing in the shell.

- W=5 nights-with-data, MIN_NIGHTS=2, MIN_TRIALS=4, all CLI-tunable (`--window-nights`,
  `--min-nights`, `--min-trials`, `--escalate-after`) so the tests can drive edge cases.
- Labels: `REGRESSION` / `SUSPECTED-FLAKE` / `GAP` / `INSUFFICIENT-HISTORY`, per D2's table.
- **Escalation per §2's refinement** (`consec >= K AND no prior REGRESSION in the current run`),
  GAP windows excluded.
- Shell routing per D4: `REGRESSION` → per-bench `--type bug` (unchanged) + one Discord ping;
  `SUSPECTED-FLAKE` → **ONE aggregated** `--type note` per night carrying per-bench
  `passes/trials over N nights` **and** `failing X/3 toward escalation`; `GAP` → aggregated
  `--type note` keeping the existing "Gap-finder candidates" phrasing verbatim (eval-gap-finder is
  a text-contract consumer); `INSUFFICIENT-HISTORY` → named in the summary body with the shortfall
  and the counter. Escalated REGRESSION bodies say `escalated: 3rd consecutive all-fail night` plus
  the label escalated from.

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M3.1 | `python3 tools/test_nightly_classify.py -v -k Rule` | `--- PASS` for one test per row of D4's table on synthetic windows, incl. the boundaries: 2 nights/4 trials all-pass → REGRESSION; **2 nights/3 trials all-pass → INSUFFICIENT-HISTORY** (MIN_TRIALS boundary); 1 night/4 trials all-pass → INSUFFICIENT-HISTORY (MIN_NIGHTS boundary) |
| M3.2 | **The V4 weakness is dead**: `-k test_single_trial_prior_night_cannot_certify_solid` | A history of exactly one prior night with one passing trial + tonight all-fail ⇒ label is `INSUFFICIENT-HISTORY`, **never** `REGRESSION`. (Under the *old* rule this same fixture yields REGRESSION — assert both, so the test proves a behavior *change*, not a tautology.) |
| M3.3 | Escalation, exactly once: `-k Escalat` | 5-night synthetic: mixed window then all-fail n1,n2,n3,n4,n5 ⇒ exactly ONE `REGRESSION` line across the 5 nights, at n3; n4/n5 emit SUSPECTED-FLAKE or GAP. **Plus** the missed-night variant (§2): skip the classifier on n3, run on n4 ⇒ escalation still fires at n4 (this test FAILS under the doc's literal `consec == K`) |
| M3.4 | GAP exclusion: `-k test_never_passed_bench_does_not_escalate` | window 0/6 over 3 nights + 3 consecutive all-fail ⇒ `GAP`, no escalation. (Real analogue: `cli_args`, 07-26.) |
| M3.5 | New-benchmark timeline (D6 concession, pinned): `-k test_new_benchmark_timeline` | pass n1; all-fail n2 ⇒ INSUFFICIENT-HISTORY; n3 ⇒ SUSPECTED-FLAKE; n4 ⇒ escalated REGRESSION exactly once. Assert the INSUFFICIENT-HISTORY line **names the benchmark and carries the counter** — D5's visibility promise is what pays for the 48h delay, so it is tested, not promised |
| M3.6 | **Routing has no issue-creating path for suppressed labels**: `grep -nE -- '--type "?bug"?\|--github' tools/launchd/nightly-eval.sh` | Every match is inside the `REGRESSION` branch. Zero matches in the SUSPECTED-FLAKE, GAP, INSUFFICIENT-HISTORY, or summary branches. (This is the C4 gate: `--type note` and un-typed sends provably never reach `syncMessageToGitHub`.) |
| M3.7 | Shell smoke with a stubbed `ailang` on `PATH` that logs its argv: run the routing block against an M3 fixture producing 1 REGRESSION + 2 SUSPECTED-FLAKE + 1 GAP + 1 INSUFFICIENT-HISTORY | Captured argv shows exactly: 1 × `--type bug`, 1 × Discord send, **exactly ONE** aggregated suspected-flake send with `--type note`, 1 × GAP `--type note`, and the summary body contains both the HEALTH line and the `insufficient history:` line. Any second `--type bug` fails M3 |
| M3.8 | Title-convention pin (the conditional part of R-F): `-k test_send_titles_embed_date` | Every generated title matches `.*\(\d{4}-\d{2}-\d{2}\)$`. **Documented limitation**, asserted in the test docstring: `InboxMessageExistsByTitle` skips `archived`/`deleted` rows (`internal/messaging/inbox.go:563`), so exactly-once delivery holds only while nightly messages are not archived |

---

### M4 — Real-history replay + CI wiring (~2h, ~120 LOC) *(was 1h in the doc; see R-B/R-D)*

**M4a — the replay is the acceptance evidence.** Commit
`tools/testdata/nightly_classify/replay_2026-07.jsonl` — the real banked history for the 5 nights,
generated by a committed extractor `tools/testdata/nightly_classify/extract_history.py` run over
`/tmp/nightly_eval_2026072{4..8}_rag_on/agent` (the extractor is committed so the fixture's
provenance is reproducible *while the data still exists*; per R-C it will not be re-derivable after
~5 days). Additionally embed the doc's V9 07-23 row (`1/2, compile_error`) as an explicitly
**provenance-annotated, no-longer-verifiable** record so the doc's own 07-25 replay can be run.

**M4b — CI wiring that actually gates** (R-D). Add `test-nightly-classifier` to `make/test.mk`
running `python3 tools/test_nightly_classify.py -v`, add it to `make/ci.mk`'s `ci` target **and** add
a step to the `test` job in `.github/workflows/ci.yml` with an anti-skip assertion in the style of
`ci.yml:86-91`.

**Acceptance criteria:**

| # | Command | Exact observable |
|---|---|---|
| M4.1 | `python3 tools/test_nightly_classify.py -v -k Replay` — **the suppression half** | With the replay fixture: **07-25 json_parse (#480) not REGRESSION**, **07-27 json_parse (#485) → SUSPECTED-FLAKE (window 4/6 over 3n)**, **07-28 contract_bst_validate (#499) → SUSPECTED-FLAKE (4/8 over 4n)**, **07-28 list_comprehension (#500) → SUSPECTED-FLAKE (7/8 over 4n)**. Two variants asserted for 07-25: **with** the V9 07-23 row → `SUSPECTED-FLAKE` (the doc's literal AC); **without** it (live-derivable data only) → `INSUFFICIENT-HISTORY`. Both suppress; the test asserts *both labels explicitly* so R-C can never be quietly papered over |
| M4.2 | **the non-vacuity half — a REAL control, not synthetic** | **07-26 `higher_order_functions` (#483) → REGRESSION, on 07-26 itself** (window 4/4 over 2 nights, p̂=1.0, zero added latency). A guard that suppresses everything fails this. **Plus** the synthetic 5-solid-nights → all-fail control from the doc |
| M4.3 | **the guard barks at something new**: `-k test_replay_csv_to_json_escalates_once` | `csv_to_json_converter` escalates to REGRESSION on **07-27** (consec 3/3, window 1/6) and does **not** re-fire on 07-28 |
| M4.4 | Aggregate assertion, printed by the test: | Over 07-25…07-28 the corpus contains **5** real filed issues (#480, #483, #485, #499, #500) and the guard produces **2** REGRESSIONs (#483 same-night + the csv_to_json escalation). Test asserts `regressions == 2` and `suppressed == 4`. A change to the rule that moves either number fails visibly |
| M4.5 | `make test-nightly-classifier` | rc=0 |
| M4.6 | **CI actually runs it** (R-D): after push, `gh api repos/:owner/:repo/actions/jobs/<test-job-id>/logs \| grep -c -- '--- PASS'` for the classifier step | ≥ 20 PASS lines from `tools/test_nightly_classify.py`. Additionally, `ci.yml` contains a step whose `run:` includes `make test-nightly-classifier`, and an assertion step that fails with `::error::` if the classifier PASS lines are absent (mirroring `ci.yml:86-91`). **Adding it only to `make ci` does not satisfy M4.6.** |
| M4.7 | Deliberate-break probe (the anti-vacuity proof): on a scratch commit, invert one replay expectation and run `make test-nightly-classifier` | rc≠0 with a named failing test. Revert. Record the observed failure text in the sprint JSON `notes` |
| M4.8 | `CHANGELOG.md` `[Unreleased]` gains an entry naming the new tool, the two new labels, the `--bootstrap` flag, and the history-file path | `git diff CHANGELOG.md` non-empty; entry mentions `tools/nightly_classify.py` and `~/.ailang/state/nightly-eval-history.jsonl` |
| M4.9 | Mission doc correction (doc's last AC): the V8 "compare like-for-like CONDITIONS" bullet is already dropped from the queue row at `design_docs/v1-mission.md:1271` | `grep -c "like-for-like" design_docs/v1-mission.md` — the only surviving occurrence is inside the explicit **CORRECTION** clause. **CONFIRMED already done** by the doc-landing commit; M4 only verifies, no edit needed |

---

## 4. Migration / back-compat (hard requirement #4 — explicit answer)

**Night 1 after the change lands (JSONL does not exist, nightly never passes `--bootstrap`):**
- The classifier reports `⚠ history unavailable (file not found) — regression detection DEGRADED tonight`
  as the first line of the controlplane summary body, and logs it.
- **Every** persistent failure classifies as INSUFFICIENT-HISTORY and is named in the summary with its
  shortfall. On 07-28's real data that is ~8 benchmarks — a readable list, not an explosion.
- **Zero GitHub issues are filed that night.** rc=0. That is the intended, loud, one-night blind spot;
  it is not a silent fallback because the summary leads with the warning.
- The run **does** write tonight's records (`--update-history` runs regardless of whether history
  existed), so night 2 has 1 prior night, night 3 meets MIN_NIGHTS=2, and by night 6 the W=5 window
  is full. Same-night REGRESSION detection is restored for solid benchmarks from **night 3**.

**Can `/tmp` be seeded in?** Yes, but only partially, and this is a first-party measured ceiling
(R-C): `/tmp` retains at most **4 prior nights** of *files* (directories persist longer but are
emptied ~5 days after their files were written, swept at 00:00). So a deployment-time
`python3 tools/nightly_classify.py --bootstrap --history ~/.ailang/state/nightly-eval-history.jsonl`
seeds **≤4 nights** — enough to clear MIN_NIGHTS=2/MIN_TRIALS=4 immediately (so night 1 is not blind),
but **never** enough to fill W=5 on day one. **Recommended deployment order**: merge → rig pulls →
run `--bootstrap` manually **once, the same day**, before the next 05:00 nightly → verify with
`wc -l ~/.ailang/state/nightly-eval-history.jsonl` and the HEALTH line. Bootstrap is idempotent
(M2.5) so a repeat is harmless.

**Rollback**: `git revert` the M1 commit restores the heredoc; the JSONL is inert (nothing else reads
it) and can be left in place for a re-land.

---

## 5. Scope guard (hard requirement #6)

The doc's scope has **not** grown. Reviewed against the P2 ~1.3d box:

- **Keep all four milestones.** Nothing here is gold-plating; every element traces to a quorum
  objection that was accepted verbatim.
- **Estimate delta: 1.3d → 1.4d (+1h).** Attributed precisely: M4 1h → 2h because the doc's CI wiring
  was **vacuous** (R-D: `make ci` is not run by GitHub Actions) and the replay corpus grew from 1
  benchmark to 5 real issues (R-B). Stated in the open, not absorbed — same discipline the doc used.
- **Cut candidate if M2 overruns** (in priority order, never the contract tests):
  1. M4.7's deliberate-break probe → move to the executor's commit message as a recorded observation
     rather than a scripted step (saves ~15 min, loses a little rigor).
  2. M4a's V9-07-23 provenance row → drop the doc's-literal-AC variant and keep only the
     live-derivable INSUFFICIENT-HISTORY assertion (saves ~20 min; **cost**: the doc's own headline
     AC becomes unassertable, so this needs a doc erratum — prefer cut 1).
  3. **Never cut**: M2.2's lock-ownership tests (they are the entire round-2 quorum resolution),
     M4.2's real solid→broken control (the non-vacuity proof), M4.6's real CI wiring.
- **Explicitly NOT in scope** (confirmed against the doc's non-goals): fixing `InboxMessageExistsByTitle`'s
  archived/deleted hole (R-F — a `messages send` change, different lane); adding `stream_death` to
  `INFRA_CATEGORIES` (doc V15, backlog); investigating *why* `json_parse`/`csv_to_json_converter` are
  bimodal (eval-gap lane); multi-model or rag_off history.

---

## 6. Risks (most-likely-to-make-this-silently-wrong first)

| # | Risk | Why it's likely | Mitigation baked into the plan |
|---|---|---|---|
| **1** | **The lock tests get written against mocks, so the round-2 objection is "resolved" only on paper.** `os.kill(pid, 0)` and mtime back-dating are annoying to test for real; the path of least resistance is `unittest.mock.patch`. A mocked stale-but-alive test passes against an mtime-only implementation | This is the single most valuable objection the quorum produced, and it is the easiest to fake | M2.2 **requires real child processes** and requires the waiter's *elapsed time* to be asserted against the 60 s bound; the AC names the exact test IDs so an evaluator can grep for them |
| **2** | **CI wiring lands as `make ci` only ⇒ the whole guard is untested on every future PR** | The doc literally says "wired into `make ci`", and that is false-but-plausible (R-D) | M4.6 requires a `ci.yml` step **and** a log-grep observable, and explicitly says `make ci` alone does not satisfy it |
| **3** | **The replay fixture is regenerated from `/tmp` after the data expires and silently becomes weaker.** The 07-23 row is already gone; by ~2026-08-02 the 07-24/07-25 rows follow | Measured, in progress, right now (R-C) | M4a commits the extracted JSONL **as a fixture in git**, plus the extractor for provenance. The fixture must never be regenerated from a live `/tmp` after this sprint. M4.1 asserts both the with-07-23 and without-07-23 labels so the divergence stays visible |
| **4** | **`consec == K` ships as written and the "never unpaged past night 3" guarantee quietly breaks on any missed night** | It is what the doc says | §2 refinement + M3.3's missed-night variant, which **fails** under the literal rule |
| 5 | The aggregated SUSPECTED-FLAKE note's title collides across nights or the GAP note's "Gap-finder candidates" phrasing drifts, breaking eval-gap-finder's text contract | New title pattern + reworded body | M3.7 asserts exactly one aggregated note and M3.8 pins the date-suffixed title; the GAP body text is required to stay byte-identical |
| 6 | M1's "behavior-preserving" claim is asserted rather than measured, and a subtle trial-dedup difference changes verdicts | The heredoc has never had a test (doc V15) | M1.2 diffs old vs new over all 5 real nights; M1.3 pins the historical `_trial2` break (F-1) |
| 7 | R-F: a nightly controlplane message gets archived, the title-dedup guard stops firing, and a rerun files a duplicate issue | Low probability; `ack` only sets `read` | Documented in M3.8's docstring; **out of scope to fix**, flagged to the controller |

---

## 7. Velocity basis

Last 7 days: 283 commits, 319 files changed, +35,334/−4,947. Recent comparable harness/tooling
sprints (m-docs-gate, m-cost-per-success-kpi M4a) landed in 1–2 days each. This sprint is ~700 LOC
total (≈300 impl / ≈400 test) across Python + shell + Makefile + one workflow file — well inside a
single-executor 1.5-day box at the observed pace. Test-to-impl ratio is deliberately >1: the entire
value of this item is that the classifier stops being untested.

---

## 8. Deliverables

- `tools/nightly_classify.py` (new, ~440 LOC final)
- `tools/test_nightly_classify.py` (new, ~600 LOC)
- `tools/testdata/nightly_classify/replay_2026-07.jsonl` (new, fixture)
- `tools/testdata/nightly_classify/extract_history.py` (new, provenance)
- `tools/testdata/nightly_classify/compare_legacy.sh` (new, M1.2 harness)
- `tools/launchd/nightly-eval.sh` (modified: heredoc → invocation; D4 routing; net ≈ −40 lines)
- `make/test.mk` (+ `test-nightly-classifier`), `make/ci.mk` (+ to `ci`)
- `.github/workflows/ci.yml` (+ 2 steps in the `test` job)
- `CHANGELOG.md` (`[Unreleased]`)
- **No Go changes. No compiler surface. `make check-boundaries` unaffected.**

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-nightly-flake-guard-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-NIGHTLY-FLAKE-GUARD.json`
