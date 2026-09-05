# M-MOTOKO-SUITE-BOUND-DERIVATION: derive the suite's wall-clock and node-ceiling bounds from a stimulus measured in-test

**Status**: PLANNED (design, not yet started)
**Mission**: motoko · charter row **6p** (corroborated by rows 6j and 6r)
**Iteration**: 35 · designer role
**Base**: origin/dev `087fbea631a0b80556baa034b499fbdae33e76d2`
**Author**: design-doc-creator (loop), 2026-09-05
**Revision**: 3 (rev 2 = the one protocol-mandated quorum revision; rev 3 = the controller's bounded narrow-refinement carve-out applying two reviewers' verbatim fixes, §4.8). Rev 2 note follows: (the one protocol-mandated quorum revision, 2026-09-05; quorum blocked rev 1 3/3). It closes the
three objections in place: the floor check is now CONDITIONAL in the §4.2 code with both paths prototyped and a
LOUD disabled path (V28-V30, M1 AC-5/6/7); the proxy spread is a named term `P_PROXY` folded into every leg-A
margin with the arithmetic shown (§2.7, §3); and the post-measurement load exposure is bounded and published by
an end-of-suite bookend measurement (§4.2, §8, V31).
**Scope files**: `tools/eval/test_motoko_connection_probe.sh` only. The production probe
`tools/eval/motoko_connection_probe.sh` is **not touched** (see Design §4.6 for why not).
**Shell floor**: GNU bash 3.2.57 (the only CI leg that runs this suite is `launchd drivers (bash 3.2)`,
`.github/workflows/ci.yml:583`, `runs-on: macos-latest`, `timeout-minutes: 15`). No associative arrays,
no `${var^^}`, no `mapfile`/`readarray`, no `$EPOCHREALTIME`, no `date +%N`.
**Estimated size**: 3 milestones, each ≤ 1 day.
**Quorum verification log**:
| Round | Artifact | Present | Verdicts | Cost | Disposition |
|---|---|---|---|---|---|
| R1 | `m-motoko-suite-bound-derivation-2026-09-05T11-28-05Z.json` | 3/3, `.synthesis.absent_reviewers` = `[]` | `gpt5-6-sol` reject · `gemini-3-1-pro` reject · `oc-glm-5-2` reject | $0.1587 | Two objections named ONE defect (§4.2 prototype vs M1's `BOUND_FLOOR_ENFORCED` flag) — applied as written. The third was a PREMISE objection, so the controller MEASURED it instead of forwarding it (rule 3f): four stimuli degrade 1.13x–2.04x under one identical load step, so the proxy is directionally right and not tight. **UPHELD.** Designer given the measurement, not the objection. |
| R2 | `m-motoko-suite-bound-derivation-2026-09-05T11-47-05Z.json` | 2/3 as synthesised — `gpt6-astra` **ABSENT (budget)** | `gemini-3-1-pro` **pass** (flipped) · `oc-glm-5-2` reject · astra absent | $0.1015 | Absentee re-run ALONE at a raised cap (`design-review --reviewer gpt6-astra --max-cost-usd 0.40`, $0.2654) → **reject**. So the synthesis was a pass-with-a-named-hole and the degrade was hiding a real objection — the self-selecting failure, since astra dropped on budget because the doc had just grown 527 → 774 lines. Effective R2: **1 pass / 2 reject, 3/3 present**. |

**Objection surfaces per round** (rule: track the surface, not the round count): R1 = prototype/flag
consistency (×2) + stimulus-proxy premise (×1) — **two surfaces**. R2 = proxy measurability/gating
(glm) + measurement-helper error propagation (astra) — **two surfaces, both new, neither the R1
surface**. Objections are SPREAD and MOVING, with one reviewer flipping to pass; that is a maturing
doc, **not** a SPLIT signal.

**Disposition — NARROW-REFINEMENT CARVE-OUT** (ratified for this mission at iteration 29). After the
one protocol-mandated re-quorum, both surviving objections (a) carry a concrete reviewer-authored
`proposed_fix` and (b) dispute no design DIRECTION — they are completeness/no-silent-fallback
objections. So the CONTROLLER applied the reviewers' OWN TEXT verbatim (§4.8), rather than spending a
second designer revision or force-passing. This SATISFIES the objections; it is not a force-pass, and
no objection was overridden. The Fable diet is intact: ONE doc, one authoring run, one revision run.


---

## 1. Problem

`tools/eval/test_motoko_connection_probe.sh` (875 lines, 46 arms, 57 s on this host) bounds every
arm with wall-clock constants and bounds one arm with a process-tree node ceiling. All of those
constants were calibrated on one machine at one load level, and the mission has now measured the
same defect shape three times:

1. **Node ceiling vs wall clock** (row 6p, iteration 32). The probe's `descendant_pids` walk is
   bounded by two racing mechanisms: a wall clock (`TREE_DISCOVERY_SECS`, checked at probe:188) and a
   node ceiling (`MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}`, probe:126, checked at probe:196).
   Which one fires first is a property of the host's fork rate, not of the code. Iteration 32 pinned
   `PROBE_MAX_TREE_NODES=50000` on ONE arm's own `env` line (test:519) and gated the variable at suite
   scope (test:829-832). That closed the race for that arm on that machine; the constant is still a
   constant.
2. **Wall-clock lane deadlines** (row 6j addendum, iteration 33). Two arms red intermittently at load
   average 39-46 on 16 CPUs: `refusing live path refuses with the control-void message` (its lane
   tripped `exceeded 4s sampling deadline`, the `PROBE_TIMEOUT_SECS=4` at test:422) and `production
   run_lane fixture readiness failed (outer_rc=82)` (the `run_lane_ready_cap_secs=5` at test:588).
   Interleaved three-way run, base 0 reds in 17; one extra arm ahead of those two, 4 reds in 19; that
   arm moved behind them, 0 in 5.
3. **Arm cap on the CI runner** (row 6j). `descendant discovery refuses on the real wall-clock
   deadline` blew the 120 s `ARM_CAP_SECS` (test:9) on the GitHub macOS runner twice on unrelated
   commits. (The row's "~1.06 s locally" is stale; see Evidence §2.4 — the arm now takes 2.2-2.7 s
   here because a 1 s pgrep-stub delay was added after that number was taken. The >100x gap stands.)

The mission's conclusion, which this doc takes as its brief: **the durable fix is not a different
constant** but a helper, in the suite itself, that measures the relevant stimulus at run time and
derives each bound from the measurement.

Revision 1 said the resulting ratio "holds by construction on any machine". The quorum controller
measured that a single bash-fork stimulus does NOT bound the heterogeneous ops (`date`, `pgrep`,
`lsof`) tightly — their degradation factors under one identical load step differ by up to 1.8x — and
this revision re-measured it (§2.7, V32-V33). The claim this doc now makes is the narrower one the
evidence supports: **leg A holds on any machine where the proxy spread `P_PROXY` (worst real-op
slowdown ÷ stimulus slowdown) stays within the tolerated spread `X = 4.7` of the tightest bound (§3),
budgeted at `P_PROXY = 2`, measured at 1.35 here (interleaved) and 1.6 by the controller
(sequential).** What happens outside that envelope, and what happens when load steps AFTER the
measurement, is stated in §3, §4.2 and §8 rather than left inside "by construction".

This doc answers the five questions the row asks: what stimulus, what ratio (and the trade between
the two legs), how the derivation fails loudly, what happens to the arm-scoped `50000` and its gate,
and which mutation each acceptance criterion kills — including addition-shaped ones.

---

## 2. Evidence (with numbers)

All measurements below were taken on this host on 2026-09-05 (16 CPUs, `/bin/bash` 3.2.57,
darwin/arm64), at HEAD `087fbea63`, with the stubs copied byte-identically from test:284-357 into
`/tmp/bd-bench/live-bin`. Every number is reproduced verbatim in the Verification Log (§7).

### 2.1 The stimulus moves 3.5-4x with ambient load alone, on the same host, same afternoon

Fork/exec rate of the suite's own `pgrep` stub (a 7-line `#!/bin/bash` script), counted while a
background `sleep 1` is alive:

| Condition | Load average (1 min) | Stub fork rate (iter/s), 3 reps | Rows |
|---|---|---|---|
| Quiet | 2.12 | 554 · 782 · 800 | V9 |
| Quiet, 0.5 s window ×2 | 2.12 | 822 · 872 · 818 | V9 |
| 32 × `yes >/dev/null` on 16 CPUs | (2.47 at start; 1-min average lags) | 228 · 218 · 200 | V14 |

Ratio quiet/loaded: **3.5-4.0x**. Iteration 32 reported 474-653 quiet vs 181-200 at load 20.59: this
measurement CORROBORATES both the magnitude and the swing. The reference stimulus the design
actually forks (a 2-line `#!/bin/bash` script, §4.1) measures 688-787 quiet and 195-209 loaded (V21).

### 2.2 The two wall-clock arms that flake sit at roughly 2x margin, and the margin is per lane

`refusing live path` runs TWO lanes (`treatment treatment`), each with a 2 s stub sleep and a
`PROBE_TIMEOUT_SECS=4` deadline. The deadline is set per lane (probe:247) and trips when
`now > deadline` (probe:250), i.e. between 4.0 and 5.0 s of lane wall time.

| Condition | Whole-arm elapsed (2 lanes) | Per-lane (÷2) | Verdict | Rows |
|---|---|---|---|---|
| Quiet ×3 | 4.488 · 4.470 · 4.482 s | ≈2.24 s | correct (`verdict is void`) | V15 |
| 32 × `yes` | 4.960 s | ≈2.48 s | correct | V14 |

So under this host's synthetic load the lane runs at 2.5 s against a 4-5 s trip: the margin is
1.6-2.0x on the SLEEP-dominated path. At iteration 33's load 39-46 the forks per loop iteration
(`date`, `pgrep`, `lsof`, `sleep 1` — probe:248-276) each stretch and the lane crosses 4 s. That is
the wrong-verdict failure: a healthy probe reported red for a timing reason unrelated to what the
arm asserts.

### 2.3 The measurement is cheap: 1.04-1.44 s per measurement, two per suite run, 4.9% of the 57 s suite

| Method | Cost (s), reps | Rate reported | Rows |
|---|---|---|---|
| Direct: `sleep 1 &` timer + count stub forks until `kill -0` fails | 1.083 · 1.037 | 709 · 659 | V18 |
| Same, wrapped in the suite's `run_bounded` (10 s cap) | 1.390 · 1.435 | 587 · 626 | V18 |
| `date +%s` tick-aligned 1 s window, walk-shaped (date+stub per iteration) | 1.018 · **1.992** | 341 · 339 | V10 |
| 0.5 s window, direct | 0.551 · 0.581 · 0.546 | 822 · 872 · 818 | V9 |

The tick-aligned method is rejected: its cost is 1-2 s depending on where in the second it starts
(V10 shows both extremes). `run_bounded` adds ~0.35 s of poll latency (its back-off polls at 0.05,
0.25, 1.25 s) and depresses the reading ~10% because its own `date`/`sleep` forks compete — in the
CONSERVATIVE direction (a lower reading derives a larger scale). Chosen: **1 s window inside
`run_bounded`, cost 1.4 s, taken TWICE per suite run** — once at startup to derive the bounds and once
as a bookend after the last gate to publish drift (§4.2) — 2.8 s, 4.9% of 57 s. The suite's rule
(test:142-144) is that every bounded wait at startup goes through the same `run_bounded` deadline as
every arm; the bookend goes through it too.

### 2.4 The node-ceiling arm: what the walk actually visits inside its window

The discovery arm (test:516-522) in isolation, with its own env line reproduced exactly:

| Variant | Elapsed (s), reps | Wall-clock message | Node message | Nodes visited | Rows |
|---|---|---|---|---|---|
| As shipped (`PGREP_LOOP_DELAY=1`) | 2.673 · 2.201 · 2.178 | 1 | 0 | (≤3 by construction) | V12 |
| Delay removed (`=0`) | 1.358 · 1.992 | 1 | 0 | 288 · 405 | V13 |

Two consequences. (a) The row's "~1.06 s locally" is superseded: the arm now costs 2.2-2.7 s because
the 1 s per-node delay makes the 1 s discovery deadline (which trips between 1.0 and 2.0 s) visit two
to three nodes at a second each. (b) With the delay in place, the node ceiling's value is
**irrelevant to the arm's outcome anywhere in [3, ∞)**: the walk cannot reach even 10 nodes before
the wall clock. The ceiling only matters if the delay is removed or made ineffective — which the
suite's own comment (test:508-512) says is "measured UNPINNED in isolation". The derived ceiling
therefore defends the delay-less configuration, and the doc says so rather than claiming the
ceiling is what keeps the arm green today.

### 2.5 The refusal-phrasing convention and the arm-scoping gate already exist

- `instrument failure, not a verdict` appears 5 times in the suite (test:153, 157, 162, 853, 859),
  always as the tail of a `not ok - <gate>: <what was observed>; instrument failure, not a verdict`
  line followed by `exit 1` (V6).
- `PROBE_MAX_TREE_NODES=50000` is on the arm's env line at test:519 only; the suite-scope gate at
  test:829-832 refuses any ambient value (V5).
- No bound-derivation helper exists: `grep -cE 'derive_bound|measure_stimulus|calibrat'` = **0**,
  known-positive control `grep -cE 'run_bounded'` = **10** on the same file (V2).

### 2.6 The suite is green at HEAD on this host

`/bin/bash tools/eval/test_motoko_connection_probe.sh`: rc=0, 57 s, `PASS: 46 probe self-test arms
ran`, one `UNINFORMATIVE UNDER SANDBOX` (the loopback socket arm) (V16). Worktree unmodified before
and after (V22, V37).

### 2.7 The proxy spread: one bash-fork stimulus is directionally right and 1.35-1.6x loose

The quorum controller measured on this machine (2-second windows, 12 bash spinners, 1-minute load
9.42 → 10.91) that the stimulus and the real ops all degrade in the SAME direction under one load step
but by DIFFERENT factors: minimal bash script 564→444/s (1.27x), `/bin/date +%s` 480→235/s (2.04x),
`/usr/bin/pgrep -x <absent>` 76→67/s (1.13x), `/usr/bin/true` 817→622/s (1.31x). Re-measured here
with the same load step, two ways, adding the two ops the controller did not include and this suite
actually forks — the `pgrep` STUB the walk forks, and the REAL `lsof` the fixture oracle forks (test:43):

| Method | Op | Ambient | 12 spinners | Degradation | ÷ stimulus (= P) | Rows |
|---|---|---|---|---|---|---|
| Interleaved: 0.5 s windows round-robin × 4 cycles, so every op sees the same load trajectory (load 17.5→18.6) | `stimulus.sh` | 587/s | 278/s | 2.11x | 1.00 | V33 |
| 〃 | `/bin/date +%s` | 628/s | 220/s | 2.85x | **1.35** | V33 |
| 〃 | `/usr/bin/pgrep -x <absent>` | 76/s | 59/s | 1.29x | 0.61 | V33 |
| 〃 | `pgrep` stub (`live_bin`; what the walk forks) | 559/s | 261/s | 2.14x | 1.01 | V33 |
| 〃 | `/usr/bin/true` | 727/s | 339/s | 2.14x | 1.01 | V33 |
| 〃 | `/usr/sbin/lsof -a -c sleep -d cwd` (the fixture oracle) | 23/s | 15/s | 1.53x | 0.73 | V33 |
| Sequential: 2 s windows, rep 1 (load 5.0→9.5) | stimulus / date / pgrep / stub / true / lsof | 723 / 763 / 93 / 784 / 986 / 25 | 216 / 217 / 56 / 283 / 225 / 18 | 3.35 / 3.52 / 1.66 / 2.77 / 4.38 / 1.39 | 1.00 / 1.05 / 0.50 / 0.83 / **1.31** / 0.41 | V32 |
| Sequential: 2 s windows, rep 2 — **confounded**: load rose 9.5 → 20.3 INSIDE the block | same | same | 175 / 39 / 18 / 73 / 128 / 9 | 3.9 / 21 / 5.4 / 10.5 / 7.6 / 2.8 | 1.00 / **5.4** / 1.4 / 2.7 / 1.95 / 0.72 | V32 |

Read in both directions, as the controller asked:

- **The controller's finding is reproduced, and the numbers differ; both are given.** Directional
  correlation holds in every row. The spread against the stimulus is 1.35x here (interleaved),
  1.05-1.31x here (sequential rep 1), and 1.6x in the controller's table (2.04/1.27). The between-op
  spread is 2.2x here (2.85/1.29) against the controller's 1.8x. Both of us measured one machine and
  one load step; neither number is a CI runner's. This revision budgets **`P_PROXY` = 2**, above
  every uncontaminated reading from either of us, and names it in every margin (§3).
- **The confounded row is the DRIFT case, not the spread case.** Sequential rep 2 shows `date` at
  5.4x the stimulus's degradation because the load average doubled BETWEEN the stimulus window and the
  `date` window, 2-8 s apart. That is gpt5-6-sol's second sentence — a load change after the one-time
  measurement — and it is handled as its own defect (§4.2 bookend, §8), not folded into `P_PROXY`.
  It is also the shape the interleaved method exists to exclude, which is why the interleaved row is
  the one the budget is set from.
- **The slowest op sets per-path cost, and the margins are computed from measured path cost, not
  from the stimulus rate.** The real `pgrep` is 7.7x slower than the stimulus (76 vs 587/s) and the
  real `lsof` 25x slower. The suite's walk never forks the real `pgrep`: every live arm sets
  `PATH="$live_bin"` (test:361, 409, 422, 435, 518, 609, 784) and forks the stub, which is the
  stimulus's own class (P = 1.01). The real `lsof` IS forked, by the fixture oracle inside the cleanup
  and survivor paths (test:43, called from :53-70 and :720). So the stimulus rate positions `k`
  only; each bound's margin in §3 uses that path's MEASURED quiet cost `F_b`, which already contains
  whatever slow op the path forks. Where the stimulus rate enters a margin directly — the node
  ceiling's walk-shape factor `r_walk ≈ 0.5 r` (V10) — it is a direct measurement of the walk with
  the stub, and §3 says what changes if the walk ever forks the real binary.
- **Readiness is fast, and its iteration-33 failure is not fork-rate-shaped.** The `run_lane`
  fixture's time-to-ready is 0.052-0.136 s against its 5 s cap, 10-14 stub forks before the ready
  file, quiet and under 12 spinners alike (V34). So `outer_rc=82` at load 39-46 required a ≥36x stall
  of a 0.1 s path, while this host's load steps move the stimulus 2-4x. The derivation scales that cap
  with `k` like every other must-not-fire bound, but the honest statement is that a stall of that
  shape is scheduling starvation, not fork slowness; the instrument that will show it is the bookend
  drift line (§4.2), not the scale.

---

## 3. What the two legs are, and the trade this design makes

For BOTH bound families there are two legs, and they pull in opposite directions:

| Family | Leg A — violation is a WRONG VERDICT | Leg B — violation is a WRONG EXPLANATION |
|---|---|---|
| Wall-clock caps (`ARM_CAP_SECS`, lane `PROBE_TIMEOUT_SECS=4`, `run_lane_ready_cap_secs=5`, …) | Cap too TIGHT for the host: a healthy arm reds on a timing accident (iteration 33's 4-in-19) | Cap too LOOSE: a regressed tree is reported later, cap-shaped, and the suite takes longer to red |
| Node ceiling on the discovery arm | Ceiling too LOW: the node message wins the race and the arm reds asserting the wall-clock message | Ceiling too HIGH: on a regressed (shared-deadline) tree the arm dies on the ARM CAP instead of on the node message |

Iteration 32 recorded that with ONE constant the feasible interval was EMPTY: leg A needs
`ceiling > rate_max × window` where rate_max is bounded only by hardware (~25,000 iter/s, 125x the
contended rate), and leg B needs `ceiling < rate_min × cap` where rate_min is whatever the loaded
host gives (181/s × 120 s = 21,720). 50,000 > 21,720: empty.

**Measuring in-test changes the interval because it shrinks the uncertainty to the swing between
measurement and arm** — minutes apart on the same host, measured 3.5-4x (§2.1) — instead of the
swing between hardware ceiling and the worst contended host ever seen (125x). It does NOT shrink it
to zero, and revision 1 wrote as if it did. Two named terms carry what remains:

- **`P_PROXY` = 2** — the proxy spread: worst real-op slowdown ÷ stimulus slowdown under the same load
  change. Measured 1.35 (interleaved, here), 1.05-1.31 (sequential, here), 1.6 (controller); budgeted
  at 2 (§2.7).
- **`r_cal` = 700/s** — the stimulus rate on the host where the quiet costs `F_b` below were measured
  (688-787, V21). The derivation's reference `FORK_RATE_REF = 400` sits 1.75x below it.

**Leg A (wall clock), with the arithmetic.** For a must-not-fire bound `b` with base cap `C_b`, a
fixed sleep component `S_b` and a measured quiet fork-cost component `F_b`, the bound holds at scale
`k` on a host measuring `r` if

```
S_b + F_b × P_PROXY × (r_cal / r)  ≤  C_b × k          with  k = ceil(FORK_RATE_REF / r)
```

The binding case is the slowest host that still gets `k = 1`, i.e. `r = 400`, where `r_cal / r =
1.75`. Solving for the largest proxy spread each bound tolerates there gives
`X_b = (C_b − S_b) / (1.75 × F_b)`:

| Bound (must not fire) | `C_b` | `S_b` | `F_b` (quiet, measured) | `X_b` | Rows |
|---|---|---|---|---|---|
| Lane `PROBE_TIMEOUT_SECS=4` (trips at 4.0-5.0 s; 4.0 used) | 4 | 2 (stub sleep) | 0.244 s per lane = (4.488 − 4.0) / 2 | **4.7** | V15 |
| `run_lane_ready_cap_secs=5` | 5 | 0 | 0.136 s (worst of 6) | 21 | V34 |
| `cap_elapsed > 10` (test:533) | 10 | 3 (2 s fixture cap + 1 s TERM grace) | ≈0.3 s | ≈13 | V16 |
| `ARM_CAP_SECS=120` on the longest must-not-fire arm (discovery, 2.7 s) | 120 | ≈2 (two 1 s pgrep delays) | ≤0.7 s | ≥96 | V12 |
| cleanup / `run_bounded` 5 s grace waits | 5 | 0 | ≈0.05 s (one real-`lsof` oracle call ≈ 40 ms) | ≈57 | V33 |
| `run_lane_outer_cap` `+10` | 10 | 0 | ≈0.4 s (readiness + one lane's forks) | ≈14 | V34, V15 |

The tightest is the lane deadline: **X = 4.7**. At `k ≥ 2` every `X_b` grows (the sleep term is
divided by `k`), so `k = 1` at `r = 400` is the binding case for all of them. The statement this
design makes is therefore: **leg A holds on any host where the proxy spread stays within 4.7,
budgeted at 2, measured at 1.35-1.6.** With `P_PROXY = 2` the lane's margin is: at `r = 400, k = 1`,
`2 + 0.244 × 2 × 1.75 = 2.85 s` against 4.0 s (**1.4x**); at this host's quiet `r ≈ 700, k = 1`,
`2 + 0.244 × 2 = 2.49 s` (1.6x); at `r = 200, k = 2`, `2 + 0.244 × 2 × 3.5 = 3.71 s` against 8.0 s
(2.2x). Revision 1 quoted 1.6-2.0x with no proxy term; the corrected worst case is 1.4x, and it is
above 1.

**When the spread exceeds X.** A healthy lane reds on timing — the leg-A wrong verdict, shaped exactly
like iteration 33's `exceeded 4s sampling deadline` — next to a `# bound derivation:` line that shows
the `k` used. The suite cannot tell that red from a real regression; the bookend line (§4.2) can say
whether the stimulus itself moved during the run. The remedy is a named constant, not a
re-derivation: raise `FORK_RATE_REF` (which lowers the `r` at which `k` becomes 2 and buys `X`
linearly) and redo the §4.3 `SCALE_MAX` arithmetic; or, structurally, add the slowest op class as a
second stimulus. Neither is done here (§8).

- Leg A (node): `ceiling ≥ 2 s × r_walk × M` where `r_walk ≈ 0.5 r` (the walk forks `date` AND the
  stub per node: 341/s walk-shaped vs 650-800/s stub-only, V10 vs V9; the stub's spread against the
  stimulus is 1.01, §2.7). Setting `ceiling = 16 r` gives margin `16 r / (2 × 0.5 r) = 16x` against
  the rate at measurement time, `16 / P_PROXY = 8x` after the proxy-spread correction, and **2x**
  against a 4x load LIFT between measurement and arm on top of that (the bad direction: measured
  slow, arm runs fast). Measured directly: 288-405 nodes visited in the window (V13) against
  `16 × 645 = 10,320`, a 25-36x margin. If the walk ever forked the real `pgrep` (76/s), `r_walk`
  would fall to ≈0.11 r: leg A gains margin, but leg B's ≈32 s below becomes ≈150 s, past the 120 s
  arm cap, and `NODE_CEILING_FACTOR` would need re-deriving. No gate in §4.5 catches that; it is
  outside this design's model and is stated as such.
- Leg B (node): `ceiling = 16 r` is reached by an un-delayed walk in `16 r / r_walk ≈ 32 s`, well
  inside `ARM_CAP_SECS × k` (≥ 120 s), so on a regressed tree WITHOUT the pgrep delay the red is
  message-shaped. WITH the delay (as shipped) the walk does ~1 node/s and the arm dies on the cap at
  ~120k s: leg B is conceded — exactly as it is today, and independently of the ceiling's value.

**The trade, stated**: this design satisfies leg A for both families on any host whose proxy spread
stays within X = 4.7 (budgeted 2, measured 1.35-1.6) and whose load does not step by more than the
lane's tolerated slowdown INSIDE the 57 s run (§8; bounded, and published by the bookend), and
accepts a bounded degradation of leg B. Wall-clock caps scale UP with measured slowness (never down), so a
regressed tree on a slow host is reported at up to `K_MAX` = 4x the base cap. That is the mission's
standing position (satisfy the leg whose violation is a wrong verdict; accept degradation in the leg
whose violation is only a wrong explanation), and the degradation is bounded by `K_MAX` so the CI
job's 15-minute budget still holds (§4.3).

---

## 4. Design

### 4.1 One stimulus, measured twice per suite invocation (startup derives, bookend publishes drift)

**What**: the host's fork/exec rate for a minimal `#!/bin/bash` script. Every bound in this suite
guards a chain of fork/execs of exactly this class (the five stubs in `live_bin` are bash scripts;
the probe's per-node and per-sample work is `date`, `pgrep`, `lsof`, `sleep`). The stimulus is the
cost of the thing the bounds are bounding.

**How** (bash 3.2, no sub-second clock needed, no new dependency):

```bash
# Written to $tmp_dir at startup; same cost class as every stub in live_bin (V21 vs V9).
printf '#!/bin/bash\nexit 0\n' > "$tmp_dir/stimulus.sh" || instrument_failure "cannot write stimulus"
chmod +x "$tmp_dir/stimulus.sh" || instrument_failure "cannot chmod stimulus"

# CARVE-OUT REVISION (gpt6-astra, round 2, verbatim proposed_fix applied — see 4.8).
# The prior form incremented n UNCONDITIONALLY after `|| true`, so a missing,
# non-executable or failing stimulus produced a POSITIVE rate that then determined
# every scaled deadline and node ceiling. That is a silent fallback on the one input
# the whole design rests on.
measure_fork_rate() {   # $1 = window secs, $2 = stimulus path. Prints iterations completed.
  local window=$1 stim=$2 n=0 timer trc srv
  [ -f "$stim" ] && [ -x "$stim" ] || {
    echo "instrument failure, not a verdict: stimulus $stim missing or not executable" >&2
    return 71
  }
  sleep "$window" & timer=$!
  while kill -0 "$timer" 2>/dev/null; do
    if "$stim" >/dev/null 2>&1; then
      n=$((n + 1))                      # increment ONLY on a successful execution
    else
      srv=$?
      kill "$timer" 2>/dev/null; wait "$timer" 2>/dev/null
      echo "instrument failure, not a verdict: stimulus $stim exited $srv during measurement" >&2
      return 72
    fi
  done
  wait "$timer"; trc=$?
  [ "$trc" -eq 0 ] || {
    echo "instrument failure, not a verdict: measurement timer exited $trc — window unreliable" >&2
    return 73
  }
  printf '%s\n' "$n"
}
```

`kill -0` is a builtin (no fork), `sleep` accepts a decimal on both BSD and GNU (the suite already
relies on `sleep 0.05` at test:59 and test:133), and the loop exits when the timer dies. It runs
through `run_bounded` with a 10 s cap so a single stalled fork cannot hang the suite:

```bash
run_bounded "$tmp_dir/stimulus.out" "$tmp_dir/stimulus.err" 10 -- measure_fork_rate 1 "$tmp_dir/stimulus.sh"
stimulus_rc=$?
```

**Where**: immediately after the `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY` early exit (test:176-179)
and before `report_arm_cap` (test:181). That is after `run_bounded` exists, after the getconf gate
(which keeps its own unscaled 5 s cap: it is a libc lookup and "anything above about one second is
already pathological", test:144), and before the first arm that consumes `ARM_CAP_SECS`. The two
recursive self-invocations that run only the containment gate (test:814, 818) exit before this point
and do not pay the 1.4 s.

**Where, second time**: the same `run_bounded` call again after the last suite-scope gate (test:869)
and before `PASS:` (test:875) — the **bookend**. Its reading is not used to derive anything; it is
fed to `classify_drift` (§4.2) and published. It exists because the startup measurement is one
sample and gpt5-6-sol's second sentence is right: a load increase after it can leave `BOUND_SCALE=1`
while the flaky paths need 2. The bookend does not prevent that; it makes it a published line
instead of an absence (§8 states the exposure and its bound).

**Cost**: 1.39-1.44 s per measurement (V18), two per invocation, 2.8 s, 4.9% of 57 s. The recursive
arm at test:773 (`PROBE_SELFTEST_ARM_CAP_SECS=invalid`) exits at test:10-13 before it and pays
nothing.

**Override for the self-arms only**: `PROBE_SELFTEST_FORK_RATE=<value>` replaces the measurement with
`<value>` verbatim, tested with `${PROBE_SELFTEST_FORK_RATE+x}` so that set-but-empty reaches the derivation (the
probe uses the same idiom at probe:216). No validation happens before the derivation — that is the point: it must
refuse `abc`, `0` and empty itself). It follows the existing `PROBE_SELFTEST_*` naming, is consumed
only by the derivation, and is the hook the addition-shaped mutations in §5 use. When it is set, BOTH
measurements (startup and bookend) are replaced by the same value, so a forced run is deterministic
and its drift line always reads `drift=none`; the drift classifier is exercised directly (§5 M1 AC-7).

**Early exit for the recursion arms**: `PROBE_SELFTEST_DERIVATION_ONLY=1` exits 0 immediately after
the `# bound derivation:` line, the same construction as `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY`
(test:176-179), with the same leak guard before the arm section (test:802-804: `refusing to
recurse`). It is what lets a recursion arm run the derivation at a forced rate and read its stdout
without running the 57 s suite inside the suite.

### 4.2 The derivation — the CONDITIONAL form, which is the code M1 ships

```bash
FORK_RATE_REF=400      # iter/s; below this the base constants are known to be too tight (§4.3)
SCALE_MAX=4            # bounded by timeout-minutes: 15 on the launchd leg (§4.3)
NODE_CEILING_FACTOR=16 # 2 s window × 0.5 walk-shape × 16 = 16x at measurement rate, 8x after P_PROXY (§3)
# The ONE named flag. M1 ships the literal default 0 (diagnostic: floor measured, published, not
# enforced). M2 flips the literal default to 1 in the same commit that wires the first bound. The
# PROBE_SELFTEST_ override exists so the recursion arms exercise BOTH paths in BOTH milestones.
BOUND_FLOOR_ENFORCED=${PROBE_SELFTEST_BOUND_FLOOR_ENFORCED:-0}
if [[ ! "$BOUND_FLOOR_ENFORCED" =~ ^[01]$ ]]; then
  echo "not ok - PROBE_SELFTEST_BOUND_FLOOR_ENFORCED must be 0 or 1" >&2
  exit 1
fi

derive_bounds() {      # $1 = measured rate. Sets BOUND_SCALE, NODE_CEILING or refuses.
  local r=$1 floor=$(( FORK_RATE_REF / SCALE_MAX )) floor_state=enforced
  if [[ ! "$r" =~ ^[1-9][0-9]*$ ]]; then
    echo "not ok - bound derivation: fork-rate stimulus measured '${r:-<empty>}' iterations in 1s; instrument failure, not a verdict" >&2
    exit 1
  fi
  BOUND_SCALE=$(( (FORK_RATE_REF + r - 1) / r ))        # ceil(REF / r)
  (( BOUND_SCALE < 1 )) && BOUND_SCALE=1
  if (( BOUND_SCALE > SCALE_MAX )); then
    if (( BOUND_FLOOR_ENFORCED == 1 )); then
      echo "not ok - bound derivation: fork rate ${r}/s needs scale ${BOUND_SCALE} > ${SCALE_MAX} (floor ${floor}/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict" >&2
      exit 1
    fi
    # NOT silent: one named, grep-able line on every run under the floor. This is a published
    # measurement of "floor not held", not a pass.
    echo "# BOUND_FLOOR_NOT_ENFORCED: fork rate ${r}/s is under the floor ${floor}/s (needs scale ${BOUND_SCALE} > ${SCALE_MAX}); running at scale ${SCALE_MAX} because BOUND_FLOOR_ENFORCED=0; the design ratio is NOT held on this run"
    BOUND_SCALE=$SCALE_MAX
  fi
  (( BOUND_FLOOR_ENFORCED == 1 )) || floor_state=DISABLED
  NODE_CEILING=$(( r * NODE_CEILING_FACTOR ))
  echo "# bound derivation: fork_rate=${r}/s reference=${FORK_RATE_REF}/s scale=${BOUND_SCALE} arm_cap=$((ARM_CAP_BASE * BOUND_SCALE))s node_ceiling=${NODE_CEILING} floor=${floor_state}"
}

classify_drift() {     # $1 = scale used this run, $2 = bookend rate. One line; LOUD when the end rate needs a higher scale.
  local k_start=$1 r_end=$2 k_end
  if [[ ! "$r_end" =~ ^[1-9][0-9]*$ ]]; then
    echo "not ok - bound drift: end-of-suite stimulus measured '${r_end:-<empty>}' iterations in 1s; instrument failure, not a verdict" >&2
    return 1
  fi
  k_end=$(( (FORK_RATE_REF + r_end - 1) / r_end )); (( k_end < 1 )) && k_end=1
  if (( k_end > k_start )); then
    echo "# BOUND_DRIFT_DURING_RUN: end-of-suite fork rate ${r_end}/s needs scale ${k_end} but this run used scale ${k_start}; any timing-shaped red above may be a wrong verdict"
  else
    echo "# bound drift: end-of-suite fork rate ${r_end}/s scale_end=${k_end} scale_used=${k_start} drift=none"
  fi
}

bound_secs() { printf '%s\n' "$(( $1 * ${BOUND_SCALE:-1} ))"; }   # :-1 because run_bounded and the EXIT trap run before derivation, under set -u

# startup, immediately after derive_bounds:
if [[ "${PROBE_SELFTEST_DERIVATION_ONLY:-0}" == 1 ]]; then exit 0; fi
```

Prototyped verbatim under `/bin/bash` 3.2.57, BOTH paths of the flag at the same forced rates
(V28-V30): with `BOUND_FLOOR_ENFORCED=0` (M1's default) r=99 and r=50 PROCEED, print exactly one
`# BOUND_FLOOR_NOT_ENFORCED:` line each, set `BOUND_SCALE=4`, rc=0, and the diag line ends
`floor=DISABLED`; with `=1` the same r=99 and r=50 REFUSE with `not ok - bound derivation: fork rate
99/s needs scale 5 > 4 (floor 100/s); host too slow to hold the ratio inside the CI budget;
instrument failure, not a verdict`, rc=1; r=800/400/100 behave identically under both (k=1/1/4,
`floor=enforced` vs `floor=DISABLED`); `=2` is refused before any measurement (`must be 0 or 1`).
The rev-1 always-on table (V19: 800→k=1, 400→1, 399→2, 200→2, 133→4, 100→4, 0/abc/empty→refuse)
still holds and is the `=1` path. `classify_drift` prototyped over six pairs (V31): (1,800) and
(2,200) and (4,100) print `drift=none`; (1,399) and (2,150) print the loud line; (1,abc) refuses.
The `#` lines are TAP-legal, and the diag line is the instrument row 6j has never had: **one CI run
after M1 lands reports the runner's actual fork rate, and whether the floor held, in the job log.**

**Why the disabled path is not a silent fallback (glm's point, answered).** (1) It is loud: one named
line per run under the floor, on stdout, uppercase-tagged, grep-able, asserted by a self-arm on every
run (M1 AC-5), and its removal is a named mutation. (2) In M1 nothing consumes `BOUND_SCALE`: no bound,
verdict or artifact depends on it, so the disabled state changes no result — it is a measurement
being published before it is enforced. (3) The flag's default flips to 1 in the SAME commit that wires
the first bound (M2 AC-2 is `=99` refusing with no override set), so there is no milestone in which a
bound is derived from a scale the floor did not hold. (4) `floor=DISABLED` is on the diag line of every
run, not only under-floor runs, so the state is always visible.

**Scale is integer and never below 1.** A fast host gets exactly today's constants; the happy-path
suite time on a quiet host is unchanged except the 1.4 s measurement (bounds that do not fire cost
no time).

### 4.3 Why `FORK_RATE_REF = 400` and `SCALE_MAX = 4`

- `400` sits below every quiet reading on this host (554-872 stub, 688-787 stimulus.sh) and below
  iteration 32's quiet 474-653, and above every loaded reading (181-228). So k=1 exactly on the
  hosts where the constants were calibrated and are known to hold, k=2 at the load levels where
  iteration 32/33 measured them failing. That is the reference the constants were, in fact,
  calibrated against; it is now written down and applied.
- `SCALE_MAX=4` is the largest integer k for which one hung arm still reports inside the job's
  `timeout-minutes: 15` (900 s): the happy path is ≤ 57 s × k (an upper bound — most of the 57 s is
  fixed stub sleeps that do not scale) plus one arm cap of 120 k s: k=4 → 228 + 480 = 708 s < 900;
  k=5 → 285 + 600 = 885 s, no headroom. Below the floor `400/4 = 100` iter/s the suite refuses
  rather than run with a ratio it cannot hold. **The floor's default is `BOUND_FLOOR_ENFORCED=0` in
  M1 and `1` from M2** (§4.2), so that M1's diagnostic line reveals the CI runner's rate first (§5 M1
  AC-4) without a red. A run under the floor in M1 is not silent: it prints `# BOUND_FLOOR_NOT_ENFORCED:`
  (§4.2, §4.5 item 3). If the runner measures under 100/s, the row 6j hang is at least partly slowness
  and `SCALE_MAX`/the CI timeout become an explicit decision, recorded in the mission log, before the
  M2 flip.
- **`P_PROXY = 2`** is a design term, not a shell constant: it is consumed by the §3 margin
  arithmetic that justifies `FORK_RATE_REF = 400`, and by nothing in the code. Putting it in the code
  as a multiplier on `FORK_RATE_REF` would make `k = 2` on every quiet host measured so far (800/688
  = 1.16 < 2), doubling every must-not-fire cap on the happy path for no measured benefit. The
  budget is held in the reference instead: 400 is 1.75x below `r_cal`, which is where the 1.4x
  worst-case lane margin in §3 comes from.

### 4.4 Which wall-clock bounds are scaled — the census

Every wall-clock literal in the suite, classified. "Must fire" bounds are pinned stimuli that an arm
asserts DO trip; they stay literal. "Must not fire" bounds are capacity margins; they take
`bound_secs`.

| test line | Literal | Class | Decision |
|---|---|---|---|
| 9 | `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}` | must not fire | `ARM_CAP_BASE=120`; after derivation `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-$(bound_secs 120)}` (an explicit override stays verbatim; the validation arm at :771-773 is unchanged) |
| 56, 66 | `cleanup_fixture_sleeps` 5 s TERM/KILL waits | must not fire | `$(bound_secs 5)`; reads `${BOUND_SCALE:-1}` because the EXIT trap can run before derivation |
| 118 | `run_bounded` 5 s terminate grace | must not fire | `$(bound_secs 5)` — `run_bounded` is called before derivation for the getconf gate and the measurement itself, so it reads `${BOUND_SCALE:-1}` |
| 145 | `gate_cap_secs=${PROBE_GETCONF_CAP_SECS:-5}` | must not fire | **unchanged**: runs before the measurement; libc lookup, "above one second is already pathological" |
| 361, 409, 422, 435 | `PROBE_TIMEOUT_SECS=4` (stub sleeps 2) | must not fire — **flaked (iter 33)** | `PROBE_TIMEOUT_SECS="$(bound_secs 4)"` on each env line |
| 371 | `PROBE_TIMEOUT_SECS=0` | must fire (validation) | unchanged |
| 390 | `PROBE_TIMEOUT_SECS=2` (dependency gates) | neither: refuses before any lane starts | unchanged |
| 402, 404 | `PROBE_TIMEOUT_SECS=1` (stub sleeps 10/20) | must fire | unchanged |
| 440, 452 | socket arm 5 s deadlines | must not fire, but failure is `UNINFORMATIVE`, not red | unchanged (out of the wrong-verdict class) |
| 516 | `discovery_killer_lane_secs=$((ARM_CAP_SECS + 30))` | ordering pin: lane deadline > arm cap | unchanged in text; inherits the scaled cap. A new one-line gate asserts `discovery_killer_lane_secs > ARM_CAP_SECS` so a future literal cannot silently invert the ordering under k>1 |
| 524 | `cap_secs_fixture=2` (sleep 30) | must fire | unchanged |
| 533 | `cap_elapsed > 10` | must not fire | `> $(bound_secs 10)` |
| 560 | orphan cap `1` (sleep 2849) | must fire | unchanged |
| 587 | `run_lane_timeout_secs=2` (fixture sleeps 2861/2863) | must fire (the arm asserts `exceeded 2s sampling deadline`) | unchanged |
| 588 | `run_lane_ready_cap_secs=5` | must not fire — **flaked (iter 33, outer_rc=82)** | `$(bound_secs 5)` |
| 589 | `run_lane_outer_cap_secs=$(( timeout + grace + 10 ))` | must not fire (emergency containment) | `+ $(bound_secs 10)` |
| 743 | `ARM_CAP_SECS=2` (report-path subshell, sleep 30) | must fire | unchanged |
| 784 | `PROBE_TIMEOUT_SECS=60` (3-node ceiling arm) | must not fire, 6000x margin | unchanged |

After M2 the file contains exactly **5** `PROBE_TIMEOUT_SECS=<digits>` literals (371, 390, 402, 404,
784), down from 9 (V17). That count is the drift gate's expected value (§4.5).

### 4.5 Failing loudly, and the gates that make the helper LOOK

1. **Non-numeric / zero / empty measurement** → `not ok - bound derivation: fork-rate stimulus
   measured '<value>' iterations in 1s; instrument failure, not a verdict`, exit 1 (§4.2). Same
   phrasing family as test:153/157/162/853/859.
2. **Measurement hung** (`run_bounded` rc=199) or exited non-zero → `not ok - bound derivation:
   stimulus exceeded its 10s cap; instrument failure, not a verdict` / `... exited <rc>; ...`.
   Same three-way shape as the getconf gate (test:152-164).
3. **Host below the floor** (`scale > SCALE_MAX`) → with `BOUND_FLOOR_ENFORCED=1` (default from M2):
   refusal with the floor message, exit 1. With `=0` (M1's default): NOT a refusal and NOT silent —
   one `# BOUND_FLOOR_NOT_ENFORCED: fork rate <r>/s is under the floor 100/s (needs scale <k> > 4);
   running at scale 4 because BOUND_FLOOR_ENFORCED=0; the design ratio is NOT held on this run` line,
   scale clamped to `SCALE_MAX`, and `floor=DISABLED` on the diag line. Both paths prototyped
   (V28-V29); both asserted by recursion arms in both milestones (M1 AC-5/6, M2 AC-2).
4. **The instrument must respond to its input** (addition-shaped). A self-arm measures a second
   stimulus that is the same script plus `sleep 0.05`, and asserts `slowed × 4 < ambient`. Measured:
   ambient 645, slowed 7, ratio 92x (V20). A counter replaced by a constant, a loop that forks the
   wrong thing, or an added `sleep` inside the AMBIENT stimulus all collapse the ratio and red this
   arm. This is the arm that proves the derivation LOOKS, not merely fires.
5. **Literal drift gate** (addition-shaped, same construction as the refusal-branch gate at
   test:846-869): count `PROBE_TIMEOUT_SECS=[0-9]+` literals in `$0` and refuse unless it equals
   5; count `PROBE_MAX_TREE_NODES=[0-9]+` and refuse unless it equals 1 (the `=3` pin at :784).
   Anti-vacuity control in the same block: `grep -c 'bound_secs '` must be ≥ 8 and `grep -c
   'PROBE_TIMEOUT_SECS='` must be ≥ 9, else `instrument failure, not a verdict`. Adding a new arm with
   a literal capacity bound moves the count and reds the suite. The reader must then either use
   `bound_secs` or document a new must-fire pin and bump the expected count — the same discipline
   `expected_refusal_branches` already enforces.
6. **Bookend drift line** (§4.2 `classify_drift`): every full run ends with exactly one `# bound
   drift:` or `# BOUND_DRIFT_DURING_RUN:` line. The loud form fires when the end-of-suite rate would
   have derived a higher scale than the run used: it does not change any verdict (a green run stays
   green) — it labels the run so a timing-shaped red beside it can be read as a possible wrong
   verdict rather than argued about later. A hung or non-numeric bookend is an instrument failure like
   the startup one (item 1/2 phrasing).
7. **Suite-scope gates survive unchanged** (test:829-844). `NODE_CEILING` and `BOUND_SCALE` are plain
   shell variables, never exported, never named `PROBE_MAX_TREE_NODES`, so the ambient-env gate stays
   quiet on a correct tree and still reds on `PROBE_MAX_TREE_NODES=50000 /bin/bash test` (§5 M3 AC-2).

### 4.6 Fate of `PROBE_MAX_TREE_NODES=50000` (test:519) and its gate — SUBSUMED, gate SURVIVES

- The literal `50000` on the arm's env line is **replaced** by `PROBE_MAX_TREE_NODES="$NODE_CEILING"`
  on the SAME env line. It stays arm-scoped (a per-command env assignment), so the suite-scope gate
  at test:829-832 is untouched and still enforces what it enforces today: no file-global assignment,
  no ambient value. Evidence the gate is still non-vacuous after the change: M3 AC-2 runs it.
- Derived value on this host: `16 × 645 = 10,320` quiet, `16 × 200 = 3,200` loaded — smaller than
  50,000 in both cases and still 25-36x above the nodes the window can visit (V13). The constant was
  larger than it needed to be because it had to cover the hardware ceiling; the derived one covers
  the measured host plus a 4x lift.
- The node-ceiling arm at test:783-786 keeps its literal `PROBE_MAX_TREE_NODES=3`: that is a must-fire
  pin (the arm asserts `exceeded 3 nodes`), and the drift gate's expected count of 1 is that line.
- The 1 s pgrep-stub delay (test:519 `PROBE_TEST_PGREP_LOOP_DELAY=1`) is **kept and out of scope**.
  §2.4 shows it makes the ceiling irrelevant while present; removing it would restore leg B for this
  arm (message-shaped red on a regressed tree in ~32 s) but the comment at test:499-512 records that
  its companion change was proved necessary only by the CI matrix, and the same instrument would be
  needed again. Filed as a follow-up in §8, not done here.

**Why the production probe is not touched.** Every bound this row names is either a suite constant or
a value the suite passes to the probe per arm through an env var the probe already validates
(probe:128-130). The probe's own defaults (`900`, `4096`, `30`) are production values for a real
motoko run, not calibration inputs for these arms, and the refusal-branch gate (`expected_refusal_
branches=28`, test:850) would have to move if a single refusal were added there. A test-side-only fix
lands without touching the number 28.

### 4.8 Round-2 carve-out revisions — the two surviving reviewer objections, applied VERBATIM

The design quorum blocked twice. Round 2: `gemini-3-1-pro` **pass**, `oc-glm-5-2` **reject**,
`gpt6-astra` recorded **ABSENT (budget)** — so the synthesis was a pass-with-a-named-hole, and the
controller re-ran the absentee alone at a raised cap (`--max-cost-usd 0.40`, $0.2654). It
**REJECTED**, i.e. the degrade was hiding a second real objection, exactly the self-selecting failure
the rule exists to catch: astra dropped on budget because this doc had just GROWN 527 → 774 lines.

Both survivors carry a concrete reviewer-authored `proposed_fix` and neither disputes the design
DIRECTION, so the **narrow-refinement carve-out** applies (ratified for this mission at iteration 29):
the CONTROLLER applied the reviewers' own text. No controller-invented resolution, no objection
overridden.

#### 4.8.1 `gpt6-astra` — the measurement helper converted failures into successful measurements

Objection, verbatim:

> §4.1 silently converts failed stimulus executions into successful measurements: `"$stim" ... || true`
> is followed unconditionally by `n=$((n + 1))`, and the helper returns the status of `printf`. A
> missing, non-executable, or failing stimulus can therefore produce a positive rate, pass
> `derive_bounds`, and determine every scaled deadline and node ceiling. This directly violates the
> no-silent-fallback axiom and contradicts §4.5's promised refusal on measurement failure.
>
> Catch: V18–V21 verify successful stimuli only. The slowed-stimulus test does not establish that
> execution failures propagate. Likewise, `wait "$timer" ... || true` hides timer failure, potentially
> accepting a shortened measurement window as one second.

**UPHELD, and it is the sharpest objection either round produced** — it lands on the single input the
entire design rests on, and the doc's own §4.5 promised the opposite behaviour. Applied as astra
specified, in §4.1: the counter increments ONLY after a successful execution; a stimulus failure emits
a diagnostic naming the stimulus and its status, terminates and reaps the timer, and returns non-zero
(72); a missing or non-executable stimulus refuses before the window opens (71); the timer's own
`wait` status is checked and an unsuccessful timer refuses rather than printing a rate (73); and
stimulus creation and `chmod` are validated at the point of writing. All three refusals use the
suite's existing `instrument failure, not a verdict` phrasing.

**Owed to M1, from astra's own text**: self-arms and Verification Log rows for (a) a stimulus exiting
1, (b) a non-executable stimulus, (c) an unsuccessfully terminated timer — each must yield an
instrument-failure refusal **with no derived-bound line emitted** — retaining the successful and
slowed stimuli (V18–V21) as positive controls. See M1 AC-8.

#### 4.8.2 `oc-glm-5-2` — `P_PROXY` is budgeted, never measured, and nothing in the suite can see it exceeded

Objection, verbatim:

> P_PROXY is a design-critical parameter that appears in every leg-A margin calculation and determines
> whether the derived bounds actually hold, yet it is measured only on one development machine (two
> observers, same host: 1.05–1.6x) and budgeted at 2 against a tolerance of 4.7. No CI runner has been
> measured. Critically, no in-suite gate exists to detect when P_PROXY is exceeded on the CI runner:
> the floor mechanism catches absolute slowness (rate too low → scale too high → refuse), but op-class
> heterogeneity is invisible to it. The bookend drift line cannot detect this either because it
> re-measures the same bash-fork stimulus, not the real ops. On a runner where the real ops degrade
> faster than the stimulus by more than 4.7x, the design silently produces wrong verdicts — healthy
> arms red on timing — that are indistinguishable from real regressions, with no instrument in the
> suite that can tell them apart.

**UPHELD.** It is the same defect the controller's own measurement established one level up (the four
stimuli degrade by 1.13x–2.04x under one identical load step), carried to its consequence: the doc
turned that spread into a *budgeted constant* and then had no way to notice the budget being wrong.

glm's `proposed_fix` offers TWO forms and explicitly names the second as the cheaper one. **The second
is adopted, and the choice is glm's own text, not a controller invention** — the first form requires a
synthetic load step inside every suite run, which would both blow the 4.9% cost budget and perturb the
very wall-clock arms this row exists to stabilise, i.e. it would re-create row 6p's defect while fixing
it. glm's alternative, verbatim:

> If the synthetic load step is too expensive for every run, alternatively measure only the real-op
> rate alongside the stimulus at startup, publish both on the diag line, and add the P_PROXY gate as a
> M2 self-arm that runs under forced rates (`PROBE_SELFTEST_FORK_RATE`) with a mocked real-op rate, so
> the gate is exercised in CI without the load step.

Applied:

1. **A second startup measurement**, same `measure_fork_rate` helper and therefore inheriting every
   refusal in §4.8.1, over a REAL op the suite actually forks. The op is the `pgrep` stub in
   `live_bin` — chosen because §2.7 measured it as the walk's actual per-node cost and because the
   controller measured real `pgrep` at 76/s against the bash stimulus's 564/s, a 7.4x absolute gap
   that the stimulus alone never sees. Cost: one further 1.0 s window (total 3.8 s, 6.7% of 57 s);
   the bookend is NOT duplicated, so this adds one measurement, not two.
2. **Both rates are published** on the `# bound derivation:` diag line as `r=<stimulus>`
   `r_real=<real-op>` `p_obs=<ratio>`, so a CI log reader and a later `grep` both get the observed
   ratio rather than the budgeted one. This is the same "publish the measurement" discipline §4.8.1's
   floor line already follows.
3. **The gate**: when `p_obs` exceeds `P_PROXY_MAX` (= 4.7, the tightest bound's tolerance from §3),
   the suite refuses with `instrument failure, not a verdict: observed proxy spread <p_obs> exceeds
   <P_PROXY_MAX>` — the same phrasing family as the floor refusal. A wrong verdict caused by op-class
   heterogeneity is now a NAMED refusal instead of a red arm indistinguishable from a regression.
4. **The gate is exercised in CI without a load step**, per glm: an M2 self-arm forces both rates
   (`PROBE_SELFTEST_FORK_RATE` plus a new `PROBE_SELFTEST_REAL_OP_RATE` on the identical
   `${VAR+x}` idiom) and asserts the refusal fires at `p_obs > 4.7` and does not fire at `p_obs`
   just under it — a two-sided arm, so it proves the gate LOOKS rather than merely fires. See M2 AC-6.

**Honest residual, which glm's objection does not remove and this doc will not pretend it does**: this
gate measures the spread between TWO op classes at ONE moment, on the host, at startup. It does not
establish the spread for every op the suite forks, and no CI runner has been measured for any of them
— the first real number arrives from M1's published diag line. §8 carries that exposure.

### 4.7 Bash 3.2 conformance

Everything above uses: `local`, integer `$(( ))`, `[[ =~ ]]` with an unquoted ERE (already used at
test:10 and probe:128), `sleep <decimal>`, `kill -0`, `wait`, `printf`, `grep -c`. No arrays are
added beyond the existing pattern; no `declare -A`, `${var,,}`, `mapfile`, `$EPOCHREALTIME`,
`date +%N`, `coproc`. The prototype ran under `/bin/bash` 3.2.57 (V18-V20) and the `make
test-launchd-drivers` target already runs `/bin/bash -n` on the file (make/test.mk:71).

---

## 5. Milestones

Each is ≤ 1 day. Every acceptance criterion names a command a reviewer can run and the specific
mutation it catches. "Addition-shaped" mutations add code or text; "removal-shaped" ones neuter it.

### M1 — Measure, derive, report (both floor paths, loudly), bookend, and prove the instrument looks (no bound is wired yet)

Adds `stimulus.sh`, `measure_fork_rate`, `derive_bounds` in its §4.2 conditional form with the
literal default `BOUND_FLOOR_ENFORCED=${PROBE_SELFTEST_BOUND_FLOOR_ENFORCED:-0}`, `classify_drift`,
`bound_secs`, the `# bound derivation:` line, the `# BOUND_FLOOR_NOT_ENFORCED:` line, the bookend
measurement and its `# bound drift:` line, the refusals for non-numeric/zero/empty/hung measurements
and for an invalid flag value, the `PROBE_SELFTEST_FORK_RATE` and `PROBE_SELFTEST_DERIVATION_ONLY`
hooks, and eight self-arms. `BOUND_SCALE` is computed but consumed by nothing yet. **The code in §4.2
is the code this milestone ships**; there is no always-on variant.

| # | Acceptance criterion (command + expected observation) | Mutation it kills |
|---|---|---|
| AC-1 | `/bin/bash tools/eval/test_motoko_connection_probe.sh \| grep -E '^# bound derivation: fork_rate=[1-9][0-9]*/s reference=400/s scale=[1-4] arm_cap=(120\|240\|360\|480)s node_ceiling=[1-9][0-9]* floor=DISABLED$'` prints exactly one line; the same run prints exactly one line matching `^# (bound drift: .* drift=none\|BOUND_DRIFT_DURING_RUN: )`; suite rc=0 and the `PASS:` count is 46 + 8 (slowed-stimulus, three `FORK_RATE` recursions, floor-disabled, floor-enforced, invalid-flag, drift-classifier) | Removal: delete the `echo "# bound derivation…"` → grep empty. Addition: append a second `echo` of the line (copy-paste duplicate) → two lines, `wc -l` ≠ 1. Removal: delete the bookend `run_bounded … measure_fork_rate` call → no drift line |
| AC-2 | Arm `bound derivation responds to a slowed stimulus` passes: it measures `$tmp_dir/stimulus.sh` and a sibling with `sleep 0.05` and asserts `slowed * 4 < ambient` (expected ~90x here, V20) | **Addition**: insert `sleep 0.05` into `stimulus.sh`'s text → ambient ≈ slowed → arm reds. Removal: replace `n=$((n + 1))` with `n=1` → both read 1 → arm reds. Substitution: fork `/usr/bin/true` instead of `$stim` → slowed ≈ ambient → arm reds |
| AC-3 | Three recursive arms (same construction as test:771-773, each with `PROBE_SELFTEST_DERIVATION_ONLY=1`): `PROBE_SELFTEST_FORK_RATE=abc /bin/bash "$0"` reds with `fork-rate stimulus measured 'abc' iterations in 1s; instrument failure, not a verdict`; `=0` reds with `measured '0'`; `=` (empty) reds with `measured '<empty>'` | **Addition**: add a silent fallback `(( r < 1 )) && r=1` or `r=${r:-400}` after the measurement → the `=0` / empty arms pass instead of red. Removal: delete the `[[ =~ ]]` check → all three arms red on "unexpectedly succeeded" |
| AC-4 | One CI run of `launchd drivers (bash 3.2)` after merge: the job log contains the `# bound derivation:` line with `floor=DISABLED`, and EITHER `scale=[1-4]` with no `BOUND_FLOOR_NOT_ENFORCED` line (runner at or above 100/s) OR exactly one `# BOUND_FLOOR_NOT_ENFORCED:` line (runner below it). Either is the measurement; **the absence of both means the derivation did not run and M2 does not start.** Record `fork_rate`, the floor state and the bookend line in the mission log — the first measurement of the runner row 6j has ever had | Not a mutation test; it is the observation M2's floor flip depends on. If the runner is under the floor, the M2 flip is preceded by an explicit `SCALE_MAX`/CI-timeout decision (§4.3), never by silently leaving `0` |
| AC-5 | Arm `bound floor disabled is loud, not silent`: `PROBE_SELFTEST_FORK_RATE=99 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash "$0"` exits **0** and its stdout contains exactly one line equal to `# BOUND_FLOOR_NOT_ENFORCED: fork rate 99/s is under the floor 100/s (needs scale 5 > 4); running at scale 4 because BOUND_FLOOR_ENFORCED=0; the design ratio is NOT held on this run` and exactly one `# bound derivation: fork_rate=99/s … scale=4 … floor=DISABLED` line (V28 is this run's prototype) | **Removal of the loud line**: delete the `echo "# BOUND_FLOOR_NOT_ENFORCED…"` (leaving the clamp) → stdout lacks the line → arm reds. Addition: clamp silently earlier (`(( BOUND_SCALE > SCALE_MAX )) && BOUND_SCALE=$SCALE_MAX` before the `if`) → the branch is never entered, no line → arm reds. Substitution: hard-wire the flag test to `(( 1 ))` → this arm gets a refusal and rc=1 → red |
| AC-6 | Arm `bound floor enforced refuses under the floor`: `PROBE_SELFTEST_FORK_RATE=99 PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=1 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash "$0"` exits 1 with `fork rate 99/s needs scale 5 > 4 (floor 100/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict` and NO `BOUND_FLOOR_NOT_ENFORCED` line; and `=100` with the same env exits 0 with `scale=4 arm_cap=480s … floor=enforced` (V29). A third recursion, `PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=2`, reds with `must be 0 or 1` (V30) | Removal: drop the `SCALE_MAX` comparison → `=99` passes with `scale=5`. Substitution: hard-wire the flag test to `(( 0 ))` → the `=1` arm gets the loud line and rc=0 instead of a refusal → red. **Addition**: accept `yes`/`true` for the flag → the `=2` arm's expected refusal is the reviewer's check that only the two literal states exist |
| AC-7 | Arm `bound drift classifier is loud only when the end rate needs a higher scale`: in a subshell, `classify_drift 1 800` prints `# bound drift: … scale_end=1 scale_used=1 drift=none`; `classify_drift 1 399` prints `# BOUND_DRIFT_DURING_RUN: end-of-suite fork rate 399/s needs scale 2 but this run used scale 1; any timing-shaped red above may be a wrong verdict`; `classify_drift 1 abc` returns 1 with `instrument failure, not a verdict` (V31) | Removal: delete the loud branch → (1,399) prints `drift=none` → red. Substitution: reverse the comparison (`k_end < k_start`) → (1,800) prints the loud line → red. **Addition**: make the loud line also `exit 1` (turning an annotation into a verdict) → the full-suite AC-1 run cannot be asserted green on a host whose rate crosses 400 mid-run; the reviewer's check is that the classifier's rc is 0 on both non-refusal paths |

| AC-8 (carve-out, §4.8.1 — `gpt6-astra`'s verbatim fix) | Three recursion arms, each `PROBE_SELFTEST_DERIVATION_ONLY=1`, exercising the measurement helper's own failure paths, with the successful and slowed stimuli (AC-2, V18-V21) retained as the positive controls. (a) **stimulus exits 1**: the stimulus file is rewritten to `exit 1` → the run exits non-zero with `instrument failure, not a verdict: stimulus <path> exited 1 during measurement` and **no `# bound derivation:` line on stdout**. (b) **stimulus not executable**: `chmod -x` → exits non-zero with `instrument failure, not a verdict: stimulus <path> missing or not executable`, again with no derived-bound line. (c) **timer terminated unsuccessfully**: the measurement's `sleep` is killed mid-window → exits non-zero with `measurement timer exited <n> — window unreliable`, no derived-bound line. In all three the assertion is BOTH the refusal text AND the ABSENCE of a derived-bound line — a refusal that still published a bound would be the original defect wearing a diagnostic | **Addition** (the one that matters, and the shape astra's objection names): restore `\|\| true` after the stimulus invocation, i.e. re-add the unconditional `n=$((n + 1))` → arm (a) measures a positive rate from a stimulus that never succeeded, prints a derived-bound line, and exits 0 → all three arms red. Removal: delete the `[ -f ] && [ -x ]` guard → (b) enters the window and returns 0 iterations, which the `=0` refusal in AC-3 catches with the WRONG message → red by name. Substitution: replace `wait "$timer"; trc=$?` with `wait "$timer" 2>/dev/null \|\| true` → (c) accepts a shortened window and passes → red |

### M2 — Wire the wall-clock class, enforce the floor, add the literal drift gate

Applies `bound_secs` to every "must not fire" row in §4.4, flips the literal default to
`BOUND_FLOOR_ENFORCED=${PROBE_SELFTEST_BOUND_FLOOR_ENFORCED:-1}` **in the same commit as the first
`bound_secs` consumer** (no commit derives a bound from an unenforced floor), swaps which of the two
floor arms needs the explicit override (M1 AC-5 now passes `…_ENFORCED=0`, M1 AC-6 passes nothing),
adds the `PROBE_TIMEOUT_SECS` literal census gate, and the `discovery_killer_lane_secs >
ARM_CAP_SECS` ordering gate.

| # | Acceptance criterion | Mutation it kills |
|---|---|---|
| AC-1 | `PROBE_SELFTEST_FORK_RATE=200 /bin/bash tools/eval/test_motoko_connection_probe.sh`: diag line shows `scale=2 arm_cap=240s`; suite rc=0; and `grep -c 'exceeded 1s sampling deadline\|exceeded 2s sampling deadline' <stderr>` is unchanged from k=1 (must-fire pins did not scale) | Removal: change `$(bound_secs 4)` at :409 back to `4` → the literal census gate (AC-3) reds. Substitution: scale a must-fire pin (`PROBE_TIMEOUT_SECS="$(bound_secs 1)"` at :402) → under k=2 the arm expects `exceeded 1s` and reads `exceeded 2s` → red |
| AC-2 | `PROBE_SELFTEST_FORK_RATE=99 /bin/bash tools/eval/test_motoko_connection_probe.sh` (NO flag override) exits 1 with `fork rate 99/s needs scale 5 > 4 (floor 100/s); host too slow…; instrument failure, not a verdict` and no `BOUND_FLOOR_NOT_ENFORCED` line; `=100` exits 0 with `scale=4 arm_cap=480s … floor=enforced`; and `PROBE_SELFTEST_FORK_RATE=99 PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=0 PROBE_SELFTEST_DERIVATION_ONLY=1 …` still exits 0 with the loud line (the disabled path stays testable after the flip) | Removal: drop the `SCALE_MAX` comparison → `=99` passes with `scale=5`. **Addition**: raise `SCALE_MAX` to 5 without changing the CI timeout → `=99` passes; the doc's arithmetic (§4.3) is the reviewer's check that 5 does not fit in 900 s. **Addition**: an ambient `PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=0` exported in CI → the no-override `=99` recursion inherits it, gets the loud line instead of the refusal, and reds — the M2 default cannot be silently undone from the environment |
| AC-3 | Literal census gate: `grep -cE 'PROBE_TIMEOUT_SECS=[0-9]+' tools/eval/test_motoko_connection_probe.sh` prints `5` and the suite's own gate agrees; the anti-vacuity control `grep -c 'bound_secs ' …` prints ≥ 8 | **Addition**: add a new arm line containing `PROBE_TIMEOUT_SECS=4` → count 6 → `not ok - wall-clock literal drift: 6 PROBE_TIMEOUT_SECS literals, this suite is written for 5`. Removal: delete the anti-vacuity control and rename `bound_secs` → the gate must red with `instrument failure, not a verdict`, not pass on a zero |
| AC-4 | Ordering gate: `PROBE_SELFTEST_FORK_RATE=100 …` (k=4) still passes arm `descendant discovery refuses on the real wall-clock deadline`, and the lane deadline it passes is 510 (= 480 + 30) | Substitution: hardcode `discovery_killer_lane_secs=150` → under k=4 the lane deadline (150) is below the arm cap (480), the iteration-319 pin is vacuous, and the new gate reds with `lane deadline 150 is not above arm cap 480` |
| AC-5 (evidence, control-first, bounded) | Under `64 × yes >/dev/null` on this host, inside a `date +%s` loop capped at 40 min: N=10 runs with the derivation active vs N=10 with `PROBE_SELFTEST_FORK_RATE=800` (forces k=1, today's constants). Report reds in each. **The control must red at least once or the load was insufficient and the evidence is UNINFORMATIVE, not a pass.** Success = control ≥ 1 red AND derived = 0 reds | This is the row's actual claim (the ratio holds on a slower machine) measured rather than argued. It cannot be a CI gate (30+ min, load-dependent) and is recorded in the sprint report with the load average and the diag line from each run |

| AC-6 (carve-out, §4.8.2 — `oc-glm-5-2`'s verbatim alternative fix) | The observed proxy spread is measured, published and GATED, and the gate is proven two-sided under forced rates with no load step (which is precisely what glm's alternative buys). (a) A plain run's `# bound derivation:` line carries `r=<n>/s r_real=<n>/s p_obs=<d.dd>` — three fields, all present, `p_obs` non-empty and numeric. (b) `PROBE_SELFTEST_FORK_RATE=800 PROBE_SELFTEST_REAL_OP_RATE=100 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash "$0"` (p_obs = 8.0 > 4.7) exits non-zero with `instrument failure, not a verdict: observed proxy spread 8.00 exceeds 4.70` and prints NO derived-bound line. (c) `PROBE_SELFTEST_FORK_RATE=400 PROBE_SELFTEST_REAL_OP_RATE=100 …` (p_obs = 4.0 < 4.7) exits 0 and DOES print the derived-bound line. (b) and (c) together are the two-sided assertion; (c) alone is what makes it a LOOK rather than a fire | **Addition**: widen the gate to `p_obs > 47` (a fat-fingered constant, or a "temporarily relax it" edit) → arm (b) stops refusing and passes → red. Removal: delete the `p_obs` field from the diag line → (a) reds on the field count, and note (b)/(c) would still pass, which is why (a) exists separately. Substitution: reverse the comparison to `p_obs < P_PROXY_MAX` → (c) refuses and (b) passes → both red, and the pair identifies the direction. Removal: drop the second (real-op) measurement and set `r_real=$r` → `p_obs` is 1.00 on every host, (a) passes, (b) and (c) both fail their forced expectations → red, so the gate cannot be quietly hollowed into a constant |

### M3 — Derive the node ceiling on the discovery arm and prove both gates survived

Replaces the literal at test:519 with `PROBE_MAX_TREE_NODES="$NODE_CEILING"`, adds the
`PROBE_MAX_TREE_NODES=[0-9]+` literal census (expected 1).

| # | Acceptance criterion | Mutation it kills |
|---|---|---|
| AC-1 | `PROBE_SELFTEST_FORK_RATE=200 /bin/bash tools/eval/test_motoko_connection_probe.sh` diag shows `node_ceiling=3200`; the discovery arm passes with `deadline expired (wall clock)`; a manual re-run of the arm's env line with `PROBE_TEST_PGREP_LOOP_DELAY=0` and `PROBE_MAX_TREE_NODES=3200` (bounded 30 s) still emits the wall-clock message and visits < 800 nodes (marker count) | Substitution: `NODE_CEILING_FACTOR=1` → derived 200 ≤ nodes visited (288-405, V13) → the delay-less re-run emits `exceeded 200 nodes` instead: leg A violated, visible |
| AC-2 | `PROBE_MAX_TREE_NODES=50000 /bin/bash tools/eval/test_motoko_connection_probe.sh` exits 1 with `PROBE_MAX_TREE_NODES is set at suite scope` (gate at :829 still fires after the change); `grep -cE 'PROBE_MAX_TREE_NODES=[0-9]+' …` prints `1` | **Addition**: add a line `export PROBE_MAX_TREE_NODES="$NODE_CEILING"` (promotes the derived value to suite scope) → the :829 gate reds. **Addition**: add a second arm with a literal `PROBE_MAX_TREE_NODES=50000` → census reads 2 → red |
| AC-3 | `git diff --stat origin/dev -- tools/eval/motoko_connection_probe.sh` is empty and the refusal-branch gate still reports `(28)` | Any edit to the probe → refusal count moves or the diff is non-empty; the reviewer's check that the fix stayed test-side |

---

## 6. Conflict Surface

**Files this design edits**

| File | Lines (at HEAD 087fbea63) | What changes |
|---|---|---|
| `tools/eval/test_motoko_connection_probe.sh` | 9-13 | `ARM_CAP_BASE=120`; derived `ARM_CAP_SECS` after the measurement; override validation stays |
| 〃 | 56, 66, 118 | `bound_secs` on cleanup/terminate grace waits (reads `${BOUND_SCALE:-1}`) |
| 〃 | after 179 (before 181) | new: stimulus script, `measure_fork_rate`, `derive_bounds` (conditional form), `classify_drift`, `bound_secs`, the flag and its validation, diag line, loud floor line, refusals, `PROBE_SELFTEST_DERIVATION_ONLY` early exit |
| 〃 | beside 802-804 | leak guard for `PROBE_SELFTEST_DERIVATION_ONLY` (same shape as the containment-only guard) |
| 〃 | after 869 (before 870-875) | bookend measurement + `classify_drift` line, ahead of the `PASS:` line |
| 〃 | 361, 409, 422, 435 | `PROBE_TIMEOUT_SECS="$(bound_secs 4)"` |
| 〃 | 516-522 | `PROBE_MAX_TREE_NODES="$NODE_CEILING"` on the env line; ordering gate beside it |
| 〃 | 533, 588, 589 | scaled containment margins |
| 〃 | after 793 (behind every wall-clock-bounded arm, per the iteration-33 placement rule at 788-791) | new self-arms: slowed-stimulus, three `PROBE_SELFTEST_FORK_RATE` recursions, floor-disabled (loud), floor-enforced (refusal), invalid flag value, drift classifier |
| 〃 | 823-844 | unchanged, but the two gates are re-verified (M3 AC-2) |
| 〃 | after 869 | new: literal census gates (`PROBE_TIMEOUT_SECS`, `PROBE_MAX_TREE_NODES`) with anti-vacuity controls |
| `tools/eval/motoko_connection_probe.sh` | — | **untouched** (M3 AC-3 checks this) |

**Who else touches these lines**

- Row **6j** (open, out of scope here): owns the CI-runner hang on the same discovery arm (test:516-522).
  M1's diag line is deliberately the first artifact 6j can use; nothing here changes what that arm
  asserts.
- Row **6o** (iteration 34, `a5b694c85`, `479ddd14f`, `b97cbf83c`): last toucher of test:141-179 (the
  getconf gate this design inserts directly after) and of the `run_lane_fixture_arm` block
  (test:582-734) whose `run_lane_ready_cap_secs` this design scales. Landed at HEAD; no open PR.
- Row **6r** (iteration 33, `115184a2e`): established the "behind every wall-clock-bounded arm"
  placement rule at test:788-791 that the new self-arms follow.
- Planned docs still referencing this arm or these constants (`design_docs/planned/`):
  `m-motoko-discovery-arm-discriminating-refusal*.md`, `m-motoko-stub-refusal-arm*.md`,
  `m-motoko-group-kill-and-lsof-containment*.md`. All three describe work already landed at HEAD;
  none proposes a further edit to the lines above. They should move to `implemented/` in a
  housekeeping pass, not here.
- The refusal-branch count (`expected_refusal_branches=28`, test:850) is a shared invariant with
  anyone editing the probe; this design does not move it.

---

## 7. Verification Log

Host: darwin/arm64, 16 CPUs, `/bin/bash` = GNU bash 3.2.57(1)-release, 2026-09-05, worktree
`/Users/voightkampff/dev/sunholo-data/.wt-motoko-iter35-sprint`. Rows V1-V27 are revision 1 (ambient
load 2-6); rows V28-V38 are this revision, taken later the same day under a HIGHER ambient load
(1-minute average 5-18 from other sessions on the box), which is why the "ambient" columns in §2.7
sit below rev-1's "quiet" numbers. The bench scripts for the new rows are `/tmp/bd-bench/proto2.sh`
(conditional derivation + drift classifier), `spread.sh` (sequential), `spread2.sh` (interleaved),
`readiness.sh`. `$f` = `tools/eval/test_motoko_connection_probe.sh`,
`$p` = `tools/eval/motoko_connection_probe.sh`. Bench scripts lived in `/tmp/bd-bench` (outside the
worktree); the `live-bin` stubs there are byte-for-byte the heredocs at test:284-357.

| # | Command | Observed output (verbatim) |
|---|---|---|
| V1 | `git rev-parse HEAD && git status --short \| head` | `087fbea631a0b80556baa034b499fbdae33e76d2` (status empty) |
| V2 | `grep -cE 'derive_bound\|measure_stimulus\|calibrat' $f; grep -cE 'run_bounded' $f` | `0` then `10` (zero paired with a non-zero control on the same file) |
| V3 | `grep -nE 'PROBE_TIMEOUT_SECS\|MAX_TREE_NODES\|PROBE_MAX_TREE_NODES' $p` | `17:… PROBE_TIMEOUT_SECS may be` · `125:timeout_secs=${PROBE_TIMEOUT_SECS:-900}` · `126:MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}` · `128:[[ "$timeout_secs" =~ ^[1-9][0-9]*$ ]] \|\| instrument_failure "PROBE_TIMEOUT_SECS must be a positive integer"` · `129:[[ "$MAX_TREE_NODES" =~ …` · `196:    if (( visited > MAX_TREE_NODES )); then` · `197:      echo "process-tree discovery exceeded $MAX_TREE_NODES nodes" >&2` |
| V4 | `grep -nE 'ARM_CAP_SECS' $f \| head -3` | `9:ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}` · `10:if [[ ! "$ARM_CAP_SECS" =~ ^[1-9][0-9]*$ ]]; then` · `11:  echo "not ok - PROBE_SELFTEST_ARM_CAP_SECS must be a positive integer" >&2` |
| V5 | `grep -nE 'PROBE_MAX_TREE_NODES' $f` | `519:    PROBE_MAX_TREE_NODES=50000 PROBE_TEST_PGREP_LOOP=1 PROBE_TEST_PGREP_LOOP_DELAY=1 \` · `776:  "PROBE_MAX_TREE_NODES must be a positive integer" \` · `777:  env PROBE_MAX_TREE_NODES=invalid …` · `784:  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=60 PROBE_MAX_TREE_NODES=3 \` · `826:# file-global assignment or export, and an ambient PROBE_MAX_TREE_NODES in the caller's` · `829:if [[ -n "${PROBE_MAX_TREE_NODES:-}" ]]; then` · `830:  echo "not ok - PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines" >&2` |
| V6 | `grep -nE 'instrument failure, not a verdict' $f` | lines `153`, `157`, `162` (getconf gate: cap / exit code / no text), `853` (refusal-branch gate: `$probe` not a file), `859` (`refusal-branch counter matched nothing; instrument failure, not a verdict`) |
| V7 | `bash --version \| head -1; sysctl -n hw.ncpu; uptime` | `GNU bash, version 3.2.57(1)-release (arm64-apple-darwin25)` · `16` · `13:11  up 11 mins, 2 users, load averages: 5.79 8.91 8.52` |
| V8 | `sed -n '583p;589p;590p' .github/workflows/ci.yml; grep -n test_motoko_connection_probe make/test.mk` | `583:    name: launchd drivers (bash 3.2)` · `589:    runs-on: macos-latest` · `590:    timeout-minutes: 15` · `make/test.mk:68:	@/bin/bash tools/eval/test_motoko_connection_probe.sh` · `71:	@/bin/bash -n tools/eval/test_motoko_connection_probe.sh` |
| V9 | `/bin/bash /tmp/bd-bench/measure.sh` (quiet section; `measure_rate` = `sleep W &` timer + count stub forks) | `== host: 16 cpus; load averages: 2.12 4.92 6.77` · `A stub  window=1.0 rep=1 iter=554 iter/s=554 cost_s=1.081` · `rep=2 iter=782 … cost_s=1.052` · `rep=3 iter=800 … cost_s=1.081` · `A stub  window=0.5 rep=1 iter=411 iter/s=822 cost_s=.551` · `rep=2 iter=436 iter/s=872 cost_s=.581` · `rep=3 iter=409 iter/s=818 cost_s=.546` · `A tiny  window=0.5 rep=1 iter=298 iter/s=596 cost_s=.545` · `rep=2 iter=410 iter/s=820 cost_s=.542` |
| V10 | same script, tick-aligned walk-shaped method (`date +%s` + stub per iteration) | `B walk-shaped tick window=1 rep=1 iter/s=341 cost_s=1.018` · `rep=2 iter/s=339 cost_s=1.992` |
| V11 | same script, `/usr/bin/true` at 0.5 s | `A true  window=0.5 iter/s=970 cost_s=.544` |
| V12 | same script: the test:516-522 env line verbatim against `$p` (`PROBE_TIMEOUT_SECS=150 PROBE_TREE_DISCOVERY_SECS=1 PROBE_MAX_TREE_NODES=50000 PROBE_TEST_PGREP_LOOP=1 PROBE_TEST_PGREP_LOOP_DELAY=1 PROBE_TEST_DRIVER_SLEEP=150`) ×3 | `arm rep=1 rc=1 elapsed_s=2.673 msg=1` · `arm rep=2 rc=1 elapsed_s=2.201 msg=1` · `arm rep=3 rc=1 elapsed_s=2.178 msg=1` (`msg` = count of `deadline expired (wall clock)` on stderr) |
| V13 | same, `PROBE_TEST_PGREP_LOOP_DELAY=0`, with `PROBE_TEST_MARKER` counting `pgrep` invocations | `arm(delay=0) rep=1 rc=1 elapsed_s=1.358 wallclock_msg=1 node_msg=0 pgrep_calls=288` · `rep=2 rc=1 elapsed_s=1.992 wallclock_msg=1 node_msg=0 pgrep_calls=405` |
| V14 | same script, loaded section: `32 × ( exec yes >/dev/null ) &`, then 3 stub measurements and one `treatment treatment` run at `PROBE_TIMEOUT_SECS=4`; then kill | `load: load averages: 2.47 4.82 6.69` · `LOADED A stub window=1.0 rep=1 iter/s=228 cost_s=1.098` · `rep=2 iter/s=218 cost_s=1.090` · `rep=3 iter/s=200 cost_s=1.089` · `LOADED refusing-live-path arm rc=1 elapsed_s=4.960 void_msg=1 deadline_msg=0` · `load after kill: load averages: 7.94 5.91 7.05; yes survivors: 0` |
| V15 | quiet `treatment treatment` run at `PROBE_TIMEOUT_SECS=4` ×3 (the `refusing live path` arm's exact env, test:421-424) | `quiet arm rep=1 rc=1 elapsed_s=4.488 void_msg=1 deadline_msg=0` · `rep=2 … 4.470 …` · `rep=3 … 4.482 …` (load at start: `5.82 5.59 6.89`) |
| V16 | `deadline=$(( $(date +%s) + 300 )); /bin/bash $f > suite.out 2> suite.err; echo rc=$?` | `suite rc=0 elapsed=57s within_deadline=1` · `ok 45 - REAL_LSOF containment accepts a leading directory without an lsof` · `ok 46 - refusal-branch count still matches the set this suite covers (28)` · `PASS: 46 probe self-test arms ran` · `grep -c '^ok'` = `46` · one line `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; fixture arm remains authoritative`; no `not ok` |
| V17 | `grep -cE 'PROBE_TIMEOUT_SECS=[0-9]+' $f; grep -nE … ; grep -cE 'PROBE_MAX_TREE_NODES=[0-9]+' $f; grep -c bound_secs $f; grep -c run_bounded $f` | `9` (lines 361, 371, 390, 402, 404, 409, 422, 435, 784) · `2` (lines 519, 784) · `0` and `10` (control) |
| V18 | `/bin/bash /tmp/bd-bench/proto.sh` — the §4.1 helper verbatim, once inside a verbatim copy of test:88-139 `run_bounded` (cap 10) and once direct | `via run_bounded window=1 rc=0 rate=587 cost_s=1.390` · `via run_bounded window=1 rc=0 rate=626 cost_s=1.435` · `direct window=1 rate=709 cost_s=1.083` · `direct window=1 rate=659 cost_s=1.037` |
| V19 | same script, `derive` at `R_REF=400 K_MAX=4` over `800 400 399 200 133 100 99 0 abc ""` | `r=800  -> k=1 node_ceiling=12800 arm_cap=120` · `r=400  -> k=1 node_ceiling=6400 arm_cap=120` · `r=399  -> k=2 node_ceiling=6384 arm_cap=240` · `r=200  -> k=2 node_ceiling=3200 arm_cap=240` · `r=133  -> k=4 node_ceiling=2128 arm_cap=480` · `r=100  -> k=4 node_ceiling=1600 arm_cap=480` · `r=99   -> not ok - bound derivation: fork rate 99/s needs scale 5 > K_MAX=4 (floor 100/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict` · `r=0    -> not ok - bound derivation: fork-rate stimulus measured '0' iterations; instrument failure, not a verdict` · `r=abc  -> … measured 'abc' …` · `r=     -> … measured '<empty>' …` |
| V20 | same script, ambient `stimulus` vs the same script plus `sleep 0.05`, 1 s window each | `ambient=645 slowed=7 ratio=92x  slowed<ambient/4: 1` |
| V21 | `/bin/bash /tmp/bd-bench/calib.sh` — the exact §4.1 `stimulus.sh` (`#!/bin/bash` / `exit 0`), 1 s window ×3 quiet, then ×3 under 32 × `yes` | `quiet load: load averages: 2.94 4.24 6.06` · `QUIET stimulus.sh window=1 rep=1 rate=772` · `rep=2 rate=787` · `rep=3 rate=688` · `LOADED(32 yes) stimulus.sh window=1 rep=1 rate=209` · `rep=2 rate=195` · `rep=3 rate=204` · `yes survivors: 0` |
| V22 | `git status --short \| wc -l` (after every bench and the suite run, before writing this doc) | `0` |
| V23 | `sed -n '247,250p;272p' $p` (deadline semantics the per-lane margin in §2.2 relies on) | `deadline=$(( $(date +%s) + timeout_secs ))` · `while kill -0 "$pid" 2>/dev/null; do` · `now=$(date +%s)` · `if (( now > deadline )); then` · `instrument_failure "lane $lane exceeded ${timeout_secs}s sampling deadline"` |
| V24 | `sed -n '587,589p' $f` | `run_lane_timeout_secs=2` · `run_lane_ready_cap_secs=5` · `run_lane_outer_cap_secs=$(( run_lane_timeout_secs + grace_allowance + 10 ))` |
| V25 | `git log --oneline -3 -- $f; git log --oneline -3 -- $p` | `a5b694c85 test(probe): make REAL_LSOF containment a fail-loud gate, bounded (motoko row 6o, M3)` · `479ddd14f … (motoko row 6o, M2)` · `b97cbf83c … (motoko row 6o, M1)` — probe: `20cce785e fix(ci): give process-tree discovery its own deadline …` · `64ca81852 … (motoko iter-32, D4) (#1008)` · `fd1fa9e01 motoko row 6g …` |
| V26 | `grep -lE 'PROBE_MAX_TREE_NODES\|ARM_CAP_SECS\|test_motoko_connection_probe' design_docs/planned/*.md` | `m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md` · `m-motoko-discovery-arm-discriminating-refusal.md` · `m-motoko-group-kill-and-lsof-containment.md` · `…-sprint-plan.md` · `m-motoko-stub-refusal-arm.md` · `…-sprint-plan.md` |
| V27 | `grep -n 'expected_refusal_branches=' $f` | `850:expected_refusal_branches=28` |
| V28 | `/bin/bash /tmp/bd-bench/proto2.sh 800 400 100 99 50 0 abc ""` — the §4.2 `derive_bounds` verbatim, flag at its M1 default (`PROBE_SELFTEST_BOUND_FLOOR_ENFORCED` unset → 0) | `FLOOR=0 r=800 -> # bound derivation: fork_rate=800/s reference=400/s scale=1 arm_cap=120s node_ceiling=12800 floor=DISABLED [rc=0 BOUND_SCALE=1]` · `r=400 -> … scale=1 … floor=DISABLED [rc=0 BOUND_SCALE=1]` · `r=100 -> … scale=4 arm_cap=480s node_ceiling=1600 floor=DISABLED [rc=0 BOUND_SCALE=4]` · `r=99 -> # BOUND_FLOOR_NOT_ENFORCED: fork rate 99/s is under the floor 100/s (needs scale 5 > 4); running at scale 4 because BOUND_FLOOR_ENFORCED=0; the design ratio is NOT held on this run` then `# bound derivation: fork_rate=99/s reference=400/s scale=4 arm_cap=480s node_ceiling=1584 floor=DISABLED [rc=0 BOUND_SCALE=4]` · `r=50 -> # BOUND_FLOOR_NOT_ENFORCED: fork rate 50/s is under the floor 100/s (needs scale 8 > 4); running at scale 4 … [rc=0 BOUND_SCALE=4]` · `r=0 -> not ok - bound derivation: fork-rate stimulus measured '0' iterations in 1s; instrument failure, not a verdict` · `r=abc -> … measured 'abc' …` · `r= -> … measured '<empty>' …` |
| V29 | `PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=1 /bin/bash /tmp/bd-bench/proto2.sh 800 100 99 50` — same code, flag at its M2 default | `FLOOR=1 r=800 -> # bound derivation: fork_rate=800/s … scale=1 … floor=enforced [rc=0 BOUND_SCALE=1]` · `r=100 -> … scale=4 arm_cap=480s node_ceiling=1600 floor=enforced [rc=0 BOUND_SCALE=4]` · `r=99 -> not ok - bound derivation: fork rate 99/s needs scale 5 > 4 (floor 100/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict` · `r=50 -> not ok - bound derivation: fork rate 50/s needs scale 8 > 4 (floor 100/s); host too slow … instrument failure, not a verdict` (no `BOUND_FLOOR_NOT_ENFORCED` line on either refusal) |
| V30 | `PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=2 /bin/bash /tmp/bd-bench/proto2.sh 800; echo rc=$?` | `not ok - PROBE_SELFTEST_BOUND_FLOOR_ENFORCED must be 0 or 1` · `rc=1` (refused before any derivation ran; V28/V29 are the positive controls for the same script) |
| V31 | `DRIFT=1 /bin/bash /tmp/bd-bench/proto2.sh` — `classify_drift` over six (scale_used, end_rate) pairs | `k_start=1 r_end=800 -> # bound drift: end-of-suite fork rate 800/s scale_end=1 scale_used=1 drift=none [rc=0]` · `k_start=1 r_end=399 -> # BOUND_DRIFT_DURING_RUN: end-of-suite fork rate 399/s needs scale 2 but this run used scale 1; any timing-shaped red above may be a wrong verdict [rc=0]` · `k_start=2 r_end=200 -> … drift=none [rc=0]` · `k_start=2 r_end=150 -> # BOUND_DRIFT_DURING_RUN: … needs scale 3 but this run used scale 2 … [rc=0]` · `k_start=4 r_end=100 -> … drift=none [rc=0]` · `k_start=1 r_end=abc -> not ok - bound drift: end-of-suite stimulus measured 'abc' iterations in 1s; instrument failure, not a verdict [rc=1]` |
| V32 | `/bin/bash /tmp/bd-bench/spread.sh` — sequential 2 s windows per op, two reps ambient, then 12 × `( while :; do :; done ) &` spinners, two reps, then kill | `== QUIET: load averages: 4.98 6.61 6.34` · `QUIET rep=1 window=2 stimulus.sh=723/s date=763/s pgrep_real=93/s pgrep_stub=784/s true=986/s lsof_real=25/s` · `rep=2 … stimulus.sh=674/s date=814/s pgrep_real=98/s pgrep_stub=768/s true=978/s lsof_real=25/s` · `== LOADED12: load averages: 9.45 7.30 6.59` · `LOADED12 rep=1 window=2 stimulus.sh=216/s date=217/s pgrep_real=56/s pgrep_stub=283/s true=225/s lsof_real=18/s` · `rep=2 … stimulus.sh=175/s date=39/s pgrep_real=18/s pgrep_stub=73/s true=128/s lsof_real=9/s` · `spinner survivors: 0` · `== after: load averages: 20.29 10.41 7.77` (rep 2 is the confounded row in §2.7: the load average doubled inside the block) |
| V33 | `/bin/bash /tmp/bd-bench/spread2.sh` — interleaved 0.5 s windows round-robin over the six ops × 4 cycles (2 s per op), ambient then 12 spinners | `== QUIET start: load averages: 17.45 10.54 7.89` · `QUIET cycles=4 window=0.5s per-op-total=2s stimulus.sh=587/s date=628/s pgrep_real=76/s pgrep_stub=559/s true=727/s lsof_real=23/s` · `== QUIET end: load averages: 18.05 11.00 8.10` · `== LOADED12 start: load averages: 18.05 11.12 8.16` · `LOADED12 cycles=4 window=0.5s per-op-total=2s stimulus.sh=278/s date=220/s pgrep_real=59/s pgrep_stub=261/s true=339/s lsof_real=15/s` · `== LOADED12 end: load averages: 18.62 11.57 8.37` · `spinner survivors: 0` |
| V34 | `/bin/bash /tmp/bd-bench/readiness.sh` — the test:606-617 `run_lane_fixture_harness` env line verbatim against `$p` with `GRANDCHILD_SECS=1`, timing launch → ready file (`perl Time::HiRes`), ×3 ambient, ×3 under 12 spinners; `forks_before_ready` = `PROBE_TEST_MARKER` line count | `== QUIET: load averages: 9.79 10.20 8.11` · `QUIET readiness rep=1 ready_s=.109 probe_rc=1 forks_before_ready=14` · `rep=2 ready_s=.062 probe_rc=0 forks_before_ready=12` · `rep=3 ready_s=.070 probe_rc=0 forks_before_ready=12` · `== LOADED12: load averages: 9.42 10.12 8.11` · `LOADED12 readiness rep=1 ready_s=.052 probe_rc=0 forks_before_ready=12` · `rep=2 ready_s=.136 probe_rc=0 forks_before_ready=10` · `rep=3 ready_s=.102 probe_rc=0 forks_before_ready=10` · `spinner survivors: 0` |
| V35 | Control for V34: the first readiness attempt printed `ready_s=NEVER (probe exited)` ×6 because the bench's `ailang-stub` copy predated the grandchild branch — `grep -c GRANDCHILD /tmp/bd-bench/live-bin/ailang-stub` → `0`; re-copied from the heredoc (`stub heredoc lines 322-357`) → `5`; `grep -n GRANDCHILD $f \| head -3` → `335:if [[ -n "${PROBE_TEST_RUN_LANE_GRANDCHILD_CWD:-}" ]]; then` · `336:  expected_cwd=$PROBE_TEST_RUN_LANE_GRANDCHILD_CWD` · `337:  ready_file=${PROBE_TEST_RUN_LANE_GRANDCHILD_READY:?}` | as stated: `0` before, `5` after; V34 is the run after the refresh. The stale copy did not affect V9-V21 (none of them exercise the grandchild path) |
| V36 | `grep -n 'PASS:' $f; grep -n 'PATH="\$live_bin"' $f \| cut -d: -f1 \| tr '\n' ' '; grep -n PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY $f` (placement of the bookend, which arms fork only stubs, the early-exit + leak-guard pattern `DERIVATION_ONLY` copies) | `875:echo "PASS: $arms probe self-test arms ran"` · `361 409 422 435 518 609 784` · `176:if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then` · `802:if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then` · `803:  echo "not ok - PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY leaked into the arm section; refusing to recurse" >&2` · `814:… PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 … /bin/bash "$0"` · `818:…` |
| V37 | `git rev-parse HEAD; git status --short` (after every rev-2 bench, before writing this revision) | `087fbea631a0b80556baa034b499fbdae33e76d2` · `?? design_docs/planned/m-motoko-suite-bound-derivation.md` (only this doc; the suite and the probe untouched) |
| V38 | The §4.2 code block extracted VERBATIM from this file (`awk` between the first ```` ```bash ```` after `### 4.2` and its closing fence → `/tmp/bd-bench/doc42.sh`, 53 lines), `/bin/bash -n`, then sourced with `ARM_CAP_BASE=120` under `set -u` and driven at r=99 and r=100 under both flag values, plus `bound_secs 5` BEFORE any derivation and `classify_drift 1 399`, plus the early exit | `bash -n ok` · flag default: `# BOUND_FLOOR_NOT_ENFORCED: fork rate 99/s is under the floor 100/s (needs scale 5 > 4); running at scale 4 because BOUND_FLOOR_ENFORCED=0; the design ratio is NOT held on this run` · `# bound derivation: fork_rate=99/s reference=400/s scale=4 arm_cap=480s node_ceiling=1584 floor=DISABLED` · `[rc=0 scale=4 bound_secs4=16]` · r=100: `… scale=4 … floor=DISABLED [rc=0 scale=4 bound_secs4=16]` · `bound_secs before derivation (set -u): 5` · `drift: # BOUND_DRIFT_DURING_RUN: … 399/s needs scale 2 but this run used scale 1 …` — flag=1: r=99 `not ok - bound derivation: fork rate 99/s needs scale 5 > 4 (floor 100/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict` (subshell exited 1 before its echo, as designed), r=100 `… scale=4 … floor=enforced [rc=0 scale=4 bound_secs4=16]` — `PROBE_SELFTEST_DERIVATION_ONLY=1`: no output, `rc=0` |

**Premises re-verified**: helper absent (V2) ✔ · two racing bounds at probe:126/188/196 (V3) ✔ ·
arm-scoped 50000 at :519 with suite-scope gate at :829 (V5) ✔ · 3.3-3.6x load swing → measured
3.5-4.0x, corroborated (V9, V14, V21) ✔ · ARM_CAP 120 at :9 (V4) ✔ · bash 3.2 only CI leg (V8) ✔ ·
rev-1 "ratio holds by construction on any machine" → **narrowed**: holds within proxy spread X = 4.7,
measured 1.35 interleaved / 1.05-1.31 sequential here, 1.6 by the controller (V32, V33) ✔ · rev-1
"floor disabled by a flag" → **now in the code and prototyped both ways** (V28-V30) ✔ ·
"~1.06 s locally" for the discovery arm → **superseded: 2.18-2.67 s at HEAD** (V12; cause: the 1 s
pgrep delay added at test:519 after the number was taken; the >100x CI gap conclusion is unchanged).

---

## 8. Risks and what this does NOT fix

- **Does NOT fix row 6j's CI hang.** If the runner walk stalls for a reason other than fork
  slowness, scaling the cap only makes the red arrive later (bounded by `SCALE_MAX`). M1's
  diagnostic line is the first instrument that can tell those apart; it does not resolve them.
- **The runner's fork rate is unknown.** If it measures under 100/s, an enforced floor would red every
  CI run. That is why M1 ships `BOUND_FLOOR_ENFORCED=0` — loudly: `# BOUND_FLOOR_NOT_ENFORCED:` on
  every under-floor run, asserted by M1 AC-5 — until the M1 line has been read from one CI log (M1
  AC-4), and why `SCALE_MAX`/`FORK_RATE_REF` are named constants with their arithmetic in §4.3 rather
  than magic numbers. M2 flips the default in the commit that wires the first bound.
- **The proxy spread is budgeted, not bounded.** `P_PROXY = 2` covers every uncontaminated reading
  from two observers on one machine (1.05-1.6); the tightest bound tolerates 4.7 (§3). No CI runner
  has been measured. If a runner's spread exceeds 4.7 the symptom is a timing-shaped lane red beside a
  diag line with `k = 1`; the remedy is `FORK_RATE_REF` (named, with its arithmetic) or a second
  stimulus of the slowest op class. The interleaved bench (`spread2.sh`, V33) is the instrument to run
  on such a runner before either.
- **Measurement-to-arm drift — the exposure, bounded and published.** The scale is derived once, at
  startup. A load step AFTER it is covered only by the margin the bound has at the `k` chosen: for the
  lane deadline at `k = 1` and this host's quiet rate, `(4.0 − 2) / 0.244 = 8.2x` of real-op slowdown
  from the measured moment, or 4.7x at the `r = 400` edge (§3); the other bounds tolerate more. A
  step larger than that inside the 57 s run reproduces iteration 33's flake regardless of the
  derivation, and V32 rep 2 shows such a step can happen in 2-8 s (load 9.5 → 20.3). What this design
  does about it: the bookend measurement (+1.4 s, §4.2) re-measures the stimulus after the last gate
  and prints `# BOUND_DRIFT_DURING_RUN:` when the end rate would have derived a higher scale than the
  run used, so a red beside it is read as a possible wrong verdict, not argued about later. It does
  not prevent the wrong verdict. Preventing it means re-measuring before each scaled arm: 8 arms × 1.4
  s ≈ 11 s, 20% of the suite — the named extension if the drift line ever accompanies a red in CI;
  not done here. Iteration 33 measured flakes at SUSTAINED load 39-46, which the startup measurement
  does catch (k = 4 or the floor).
- **Leg B is degraded, on purpose and boundedly.** A regressed tree on a k=4 host reports its first
  cap-shaped red at 480 s rather than 120 s. The discovery arm's backstop stays conceded while the
  1 s pgrep delay is present (§2.4). Follow-up candidate, out of scope: drop the delay once the
  derived ceiling has held on the CI matrix for a run window, restoring message-shaped reds for that
  arm in ~32 s.
- **`run_bounded` depresses the reading ~10%** (V18). This is in the safe direction and is stated;
  if it ever pushes a quiet host across the 400 boundary, the symptom is k=2 on a fast machine,
  visible in the diag line, costing nothing on the happy path.
- **Windows and ubuntu legs** do not run this suite (only `launchd drivers (bash 3.2)` does,
  make/test.mk:68). On a non-Darwin host the suite already skips the `run_lane` fixture arms
  (test:26-28); the measurement and derivation run there too and use nothing Darwin-specific.
- **What the probe asserts about motoko is unchanged.** Not one line of
  `tools/eval/motoko_connection_probe.sh` moves; M3 AC-3 checks it.
