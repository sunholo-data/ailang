# Motoko Mission — make motoko the best AILANG-specific harness, and keep our evals honest about it

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md)); advanced by a scheduled
outer loop on the always-on rig.
**North star**: motoko should be the BEST harness for writing AILANG specifically — exploiting
structural advantages a generic harness on an untyped language cannot (typed-interface reads, AST
edits/queries, effect rows, contracts + Z3, exact best-of-N) — and every claim we make about it
should be measured on the tree we actually run. **The mission is done when motoko is good enough to
be an executor in the mission fleet itself** (clause 6): the harness we improve becomes a harness
that does the improving. That graduation is the honest end-state test — a harness that can land its
own sprints has demonstrated something no benchmark score argues for on its own.

**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (AILANG fix / motoko extension / core-floor fix).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration — the SAME unforked skill every mission uses (M-MISSION-PORTABILITY).
The [motoko-analyzer](../.claude/skills/motoko-analyzer/SKILL.md) skill is the **diagnostic
playbook** for "why is motoko failing" queue items (its five gates), not a competing outer loop.
**Scheduling**: launchd `dev.ailang.mission-motoko`, `StartInterval=43200` (**12h** — corrected
iteration 1 from a stale `21600`/6h; measured against the installed plist, and matching Mark's
2026-08-12 note on #663 that the cadence is halved this week while quotas are watched) — deliberately
staggered against V1 (5400s) and World (14400s), and deliberately slow: the queue is gated (see
Guardrails) and the rig's quota is shared.
**Log**: [motoko-mission-log.md](motoko-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue [#663](https://github.com/sunholo-data/ailang/issues/663) — every iteration posts its
report there as a comment; driver crashes post there too.

**The weak-model path is the METHOD, not a budget compromise (Mark, 2026-08-12).** Motoko is tuned
against weak models on purpose, and the expected result is that it becomes the best AILANG harness
**for the strongest models too**. The mechanism is a forcing function: optimising against a model
that cannot carry itself forces the *harness* to supply what the model lacks — structure,
verification, error recovery, context discipline, retry-on-the-right-signal. Those affordances are
**model-independent**. A harness tuned against a strong model can lean on the model's competence and
never grow them, and so plateaus lower on strong models than the weak-model path does.

This is a real, falsifiable claim and it is **still unmeasured**. Its test is the archived charter's
**R3** (cross-model generality study: do motoko's gains hold with strong models, and are they
AILANG-specific or general?) — carried forward here rather than left in the archive. R3 also carries
the generality split worth keeping in view: best-of-N (check + run) is **language-general** — a
portable edge on any compiler+runtime — while contracts + Z3 are **AILANG-specific**, the moat. Do
not let "we use cheap models" get recorded as a constraint we are working around; it is the design.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

The single source of truth for the values that differ per mission. The one `mission-control` skill
reads this block (and the driver env it exports from `~/.config/ailang/mission-motoko.env`) instead
of hardcoding.

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/motoko-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `motoko` (driver: `MISSION_NAME`; any name ≠ `v1` gets fully
  namespaced `~/.ailang/state/mission-motoko-*` paths — no collision with the V1 loop)
- **Executing root**: `~/.ailang-driver-pin/motoko` — the worktree `tools/launchd/lib/pin-root.sh`
  re-execs into, pinned to `origin/dev` on every fire. **This, not the clone below, is where the
  loop actually runs**, and therefore which `.claude/skills/` and `design_docs/` it reads. Confirmed
  from a live pinned session: `MISSION_WORKDIR=~/.ailang-driver-pin/motoko` while
  `AILANG_DRIVER_SRC=~/dev/sunholo-data/ailang-motoko`. `MISSION_WORKDIR` therefore names the
  *executing* root at runtime and the *source clone* only in the env file's pre-pin default — do not
  read the env file's literal as a statement about what runs.
- **Source clone**: `/Users/voightkampff/dev/sunholo-data/ailang-motoko` (env-file
  `MISSION_WORKDIR` default; `AILANG_DRIVER_SRC` at runtime) — it owns the `.git` the pin worktree
  hangs off, so it cannot be deleted. **RECONCILED 2026-08-24 (iteration 21) on Mark's one-word
  `Yes` to `D-MOTOKO-WORKDIR-1`:** it is now **0 behind / 0 ahead** of `origin/dev` with a clean
  tree, its `.claude/skills/mission-control/SKILL.md` is **3682** lines and byte-identical to the
  pin worktree's, and a session started there executes the current rulebook. Its 7 uncommitted
  files were discarded as measured-superseded (132 of 136 added lines byte-present upstream) and
  are backed up sha256-verified at `~/.ailang/backups/motoko-clone-reconcile-2026-08-24`.
  **The drift can return, and the only thing that will say so is the notice**, which re-arms
  automatically once drift falls below `AILANG_DRIVER_DRIFT_WARN` (default 25) — proved by
  extracting `mission-control.sh`'s real branch and driving it at drift 0 / 178 / 356. Before
  starting a session here, re-measure `git -C <clone> rev-list --count HEAD..origin/dev`; a
  charter sentence is a claim about the day it was written. **A SEPARATE clone from V1's**,
  deliberately. There is no cross-mission
  lock (`rig-lock.sh` guards eval jobs, not missions; the driver's overlap guard is a per-mission
  pidfile), so two missions sharing one working tree would contend on `git commit`/`push` to `dev`.
  V1 fires every 90 min, so overlap would be routine, not rare.
- **Bookkeeping issue**: `#663`, rotates weekly; live number in `~/.ailang/state/mission-motoko-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI`, `Build and Release`, `Deploy Documentation to GitHub Pages`
- **Verify profile**: `go-compiler` — this repo compiles the AILANG toolchain, so gates rebuild
  BOTH binaries (`make quick-install && make build`) and run `make test`; `~/go/bin/ailang` (PATH)
  and `bin/ailang` go stale independently (confirm `--version` == `git describe`).

**Skill sync, and why the separate checkout is NOT a skill fork.** V1 resolves `mission-control`
through `~/.claude/skills/mission-control`, a symlink into V1's checkout. This checkout has its own
git-tracked `.claude/skills/`, which takes precedence for sessions run here. That is **convergence
via git, not divergence**: both checkouts track `dev`, so a Gate-5 edit made here reaches V1 on its
next pull and vice versa. Do not "fix" this by symlinking over the tracked directory. Do keep
Gate 5's one-edit-per-iteration rule — it is what bounds the divergence window.

---

## Human Decision Ledger (authoritative current state)

This marked table—not STATUS prose or the rolling GitHub thread—is the source of truth for which
decisions are open. Validate it with `scripts/mission_decisions.sh --check`; generate human asks
with `scripts/mission_decisions.sh --open`. Rows and IDs are append-only.

<!-- decision-ledger:start -->
| ID | Status | Decision / recorded answer | Evidence |
|---|---|---|---|
| D-MOTOKO-1 | RESOLVED | Arni's ABI-settled acknowledgement is an objective gate, not an unbounded wait on a person. | Mark resolved charter D1 on 2026-08-12; the queue records the measurable predicate. |
| D-MOTOKO-ROUTE-1 | RESOLVED | Controller and Anthropic-required planner routes fall back to Codex Sol when Anthropic is unavailable; executor remains Codex Sol primary, DeepSeek v4 Flash second, and Opus last. | Fleet routing directive landed in `de0e41099` on 2026-08-15. |
| D-MOTOKO-FMT-1 | RESOLVED | **PRECONDITION of D1** (Mark, attended 2026-08-19) — the sprint TRACES motoko's resolved runtime provider first, then changes the preflight. Do not redesign around the unknown: the objection is precisely that nobody has measured which provider actually serves the ollama-declared lanes, so measure it. | Un-parks `m-motoko-fmt-remeasurement-instrument.md` (`needs-human-review`). The trace needs the `mk-ast` fork's own resolution path and/or a live motoko run holding `rig.lock` — schedule it against the GPU accordingly, and note `~/.ailang/state/launchd-hold/<label>` is the sanctioned way to keep scheduled GPU jobs off the device while it runs. The trace must DISCRIMINATE: show whether removing the unconditional `OPENROUTER_API_KEY` refusal at `internal/executor/motoko/healthcheck.go:64` deletes a real fail-fast or admits a silent OpenRouter fallback for entries declaring `provider: "ollama"`, `env_var: ""` (models.yml:1854, :1880). No reviewer disputed the instrument's DIRECTION in either round, so nothing else in the doc waits on this. **DISCHARGED 2026-08-19 (iteration 13): the trace is RUN via the fork-resolution-path arm — see `m-motoko-fmt-remeasurement-instrument.md` §12; O4 CLOSED, D1 re-shaped, live arm moved to `AC-D1-live`.** |
| D-MOTOKO-WORKDIR-1 | RESOLVED | **Yes — reconcile.** Mark answered on `#743` at `2026-08-23T18:59:43Z` with one word. Performed and verified 2026-08-24 (iteration 21): the clone is **0 behind / 0 ahead** of `origin/dev`, `git status --porcelain` is **0 lines**, `SKILL.md` is **3682** lines and byte-identical to the pin worktree copy, and all **8** worktrees survived. | Re-measured first-party before acting rather than inheriting iteration 20's numbers: ahead-commits **0** (so Gate 1 obligation 1 holds *vacuously*, stronger than the `patch-id` test it prescribes); obligation 2 fails **7 of 7** (`comm -12` = 7, positive control `CHANGELOG.md` = 1, negative control = 0); **132 of 136** added lines byte-present on `origin/dev`, the 4 absent being ledger prose origin supersedes. Residue backed up sha256-verified to `~/.ailang/backups/motoko-clone-reconcile-2026-08-24` with a firing corruption control. `git checkout -B dev origin/dev` was run **first, as prescribed**, and REFUSED rc=1 leaving the tree byte-unchanged — that refusal is recorded as the evidence the operation is protective, not `reset --hard`. |
| D-MOTOKO-WORKDIR-2 | OPEN | Grant **standing** authorization to reconcile the source clone `~/dev/sunholo-data/ailang-motoko` to `origin/dev` unattended, whenever three predicates all hold? One word: **yes** (standing) or **no** (keep asking each time). The three predicates, each a command the loop runs rather than a claim it transcribes: (1) **ahead-commits == 0**; (2) every dirty file's added lines are byte-present on `origin/dev`, or the file is byte-identical to origin's; (3) a sha256-verified backup outside the repo exists and re-verifies **after**. On any predicate failing, it still asks you. | Measured 2026-08-24 (iteration 21) **after** performing the one-shot you authorised: the clone was back to **4 behind within the hour**, because nothing pulls it — `pin-root.sh` runs `git fetch` only, `git pull` appears **0** times in both drivers. origin/dev lands **21.8 commits/day** (153 in 7d; 17 in 1d, 60 in 3d, 353 in 14d as corroborating points), so drift re-crosses `AILANG_DRIVER_DRIFT_WARN`=25 in **~1.1 days** and the doubling-dedupe then re-notifies at ~50, ~100, ~200 — i.e. roughly four asks per nine days, each resolving to the same one-word answer for the same mechanical operation. `mission-control` is explicit that a standing authorization is a human call and that until one is granted the loop must ASK, so this is the ask. Iteration 21 is the evidence it is safe: `git checkout -B dev origin/dev` **REFUSED rc=1** and left the tree byte-unchanged before anything was discarded — the refusal, not the controller's care, is what distinguishes this from the `reset --hard` Critical Principle 0 forbids. |
<!-- decision-ledger:end -->

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `motoko-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Rationale:
every iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the
scarcest model budget; the append-only history lives in the log + archive.

> The archive file currently holds the **pre-2026-08-12 charter in full** — the "beat pi" arc, the
> P0 large-context grading, and the June harness-correctness frontier. That charter was last touched
> 2026-06-24 and had gone stale in a way that matters: three of its four stated goals are now solved
> or superseded upstream (see CURRENT GOAL). It is kept because its *findings* remain valid evidence;
> it is not kept as direction.

## STATUS 2026-08-25 — ITERATION 22 COMPLETE: **ROW 6e LANDED — AND THE TWO GUARDS WRITTEN TO CLOSE IT WERE BOTH DECORATION UNTIL A JUDGE MUTATED THEM.** Pick was the queue head, row **6e**. PR [#871](https://github.com/sunholo-data/ailang/pull/871) → [`086b72184`](https://github.com/sunholo-data/ailang/commit/086b72184), evaluator **round 1 FAIL 54/100 (3 blocking) → round 2 PASS 91/100, ZERO blocking**. Gate 3b GREEN on head `ddd8f3f09` (**21** checks, 0 pending, 0 not-green, **4/4** required — build/docs-gate/lint/test — `mergeStateStatus=CLEAN`), `mergeable` read FIRST per the iteration-198 rule and `MERGEABLE` throughout. **THE ROW'S ATTRIBUTION WAS RIGHT AND ITS COUNT WAS LOW, AND ONLY THE CI LOGS COULD SAY SO.** Row 6e recorded **two** cancellations; measured across the last **100** CI runs (oldest `2026-08-22T08:42Z`) the `launchd drivers (bash 3.2)` job is **95 success / 2 failure / 3 cancelled**, and the third — `32673098414`, `23:15:56Z` — is a **push to `dev` itself**, not a PR. All three sit at **~918s**, which is `timeout-minutes: 15` firing, so these are hangs and not flakes. The arm-33 attribution was verified rather than inherited: in all three the last suite line is byte-identical (`UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer…`) and `ok 33` never appears, and the only code between that echo and arm 33 is arm 33's own invocation — control, green run `32785167377`, emits the same line and then `ok 33` **1.06s** later with all 34 arms. **THE MECHANISM IS NOT ISOLATED AND THIS SAYS SO.** It has not recurred in the ~44 runs since, on an unchanged file (`git log` over the probe, the suite, `make/test.mk` and `ci.yml` returns nothing in that path since the last cancellation; the one commit in the window, `ba2eeb4b4`, touches a different `ci.yml` step and a different make target). A synthetic near-copy of the walk **aborts bash 3.2 with SIGABRT rc=134, 5/5**, while the shipped arm passes locally in 1.06s — so the stimulus sits near a cliff and the shipped arm's margin is unmeasured; that divergence means the synthetic repro is **not** the CI mechanism and is not claimed as it. What IS established is structural and is what shipped: **the suite had no bound of its own**, so its only bound was the job timeout and a hang was a silent 15-minute cancellation carrying zero diagnostics. **EVERY ARM NOW HAS A HARD CAP** (`ARM_CAP_SECS` 120, validated, `run_bounded` backgrounds with closed stdin, TERM then KILL after 5s, real rc captured without a pipe, on expiry a named `not ok` plus both captured tails), and `descendant_pids` is bounded by **node count** as well as by the clock, with a **distinct** message so a reader can tell which bound fired — the row's own first option, since a wall-clock bound calibrated on the author's machine is the shape it names. Refusal-branch drift gate 23 → **24**; suite 34 → **39** arms. **THE EXECUTOR CAPPED BEFORE REPORTING, SO THE ENTIRE MUTATION DRILL IS FIRST-PARTY.** `codex:gpt-5.6-sol` hit the 30-minute wall (**FLAGGED**) with `.snap/M1`–`M3` complete; the work was VERIFIED, not adopted. Four mutants, each `if false && …` so it still parses, each asserted LANDED by sha256 and VALID by `bash -n` **before** its result was read, each restored byte-identical: M1 cap expiry, `ARM_CAP_SECS` validation, M2 node ceiling (run at `PROBE_TIMEOUT_SECS=60` and **not** falling through to the clock bound's message, which is what proves the arm discriminates), and the `PROBE_MAX_TREE_NODES` validation neutered **while keeping its message** so the branch-count gate could not be the killer. **A RED BANKED FOR THE WRONG REASON, CAUGHT BY THE RULE THAT EXISTS FOR IT:** the first batch run of the M1 mutant redded at **arm 30**, not at the cap arm; re-run isolated it reds at its own arm **2/2** with the unmutated suite green **3/3** in the same block. A second identical shape appeared at arm 32 (rc=1 then rc=0 on the immediate re-run). Read the exit code alone and this iteration banks a pin that does not exist. **THE JUDGE FOUND WHAT THE DRILL DID NOT, AND TWO OF THE THREE ARE DEFECTS IN MY OWN GUARDS.** Round 1 **FAIL 54/100**: **(B1)** `report_arm_cap` — the function implementing M1's headline promise — had **zero coverage**; arm 34 calls `run_bounded` directly and stops there, and neutering its `exit 1` left the suite green at 38/38. **(B2)** the cap arm asserted only `cap_rc == 199` and an upper elapsed bound, so a fixture exiting 199 of its own accord passed with **no TERM and no KILL**. **(B3)** my own PR body claimed *"fast arms are unaffected — the poll loop is not entered when the child has already exited"*, and the judge **measured it false**: the suite went **30s → 66-93s**, because the first `kill -0` runs after a `date` subshell so anything slower than a trivial binary pays a flat mandatory second. All three reproduced first-party before acting. **AND THE FIRST FIX FOR B1 PASSED FOR THE WRONG REASON — ITS OWN MUTANT IS WHAT SHOWED IT.** With `exit 1` removed, `expect_failure` falls through to its `lacked expected message` refusal and **still exits 1 with every marker present**, so rc=1-plus-markers does not discriminate; the arm now requires that fall-through message to be **ABSENT**, which is the only observable unique to `report_arm_cap` having ended the arm. Proven with a **line-range-scoped** mutation (the first attempt was an unscoped `s/^  exit 1$/` that hit the validation's `exit 1` too and redded at the wrong arm — the same lesson twice in one iteration). B3 answered by backing the poll off `0.05 → 0.2 → 1s` and by **correcting the claim in the PR body rather than leaving it standing**. Round 2 **PASS 91/100, ZERO blocking**, and the judge re-derived the round-1 table itself, reproducing two rows at **byte-identical** sha256 — the controller's evidence is corroborated, not taken on trust. Timing re-measured by the judge: pre-PR **29.89s** / flat-sleep **66.61s** / shipped **44.98-45.07s**, `make test-launchd-drivers` **70-73s**. **ONE NON-BLOCKING FINDING ROUTED TO THE QUEUE RATHER THAN INTO THIS PR** (row **6g**): `run_bounded`'s cap kills the recorded wrapper PID, not the process **group**, so a `sleep 30` fixture is reparented to `PPID 1` and survives — the suite's own "process survived" check passes because the wrapper really is dead. Pre-existing since M1 and present in production `run_lane` too, so it is a defect the doc did not introduce and it belongs on the queue, not in a growing PR. Gate 0/1 clean: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, ran from the pin root detached and clean at `b28800ddf` == `origin/dev`; running-skill check on the **RESOLVED** symlink target (it resolves to **V1's** checkout, inode `49847168` vs this pin's `49847664`) — byte-identical to origin. dev at pick time: HEAD's `test` was mid-flight, **0** not-green of 14, parent control **16**. **0** human directives on `#850` since the watermark (of **4** comments); `#743` re-read for the rotation-week catch, **0**. Ledger valid, **1 OPEN** (`D-MOTOKO-WORKDIR-2`, unanswered). Inbox **1** unread, informational. External predicates re-run as commands: Phase-0 **G1 `#154` still OPEN** with control `#175` **MERGED**, so rows 10/11/12 stay parked. No died-mid-flight traces: `#695` has no branch in this clone's worktree list and is **not attributable to this mission** — left alone per the fleet-filter rule. Weekly sweep and rotation **both not due** (`#850` created `2026-08-24T07:39:32Z`, after the Monday-07:00 **local** boundary; 4 comments < 80). Source clone re-measured at **24 behind / 0 ahead**, clean — just under the notice threshold, exactly as `D-MOTOKO-WORKDIR-2` predicted. Routing: controller `claude:claude-opus-5`; executor `codex:gpt-5.6-sol` (probe rc=0, replied `ok`) **CAPPED at 30 min, FLAGGED**; evaluator **sonnet** in **its own worktree**, two rounds — distinct provider from the executor, so generator≠judge holds there, and **FLAGGED**: the mutation drill and every round-1 fix are Anthropic-authored and Anthropic-judged, which is why the judge was pointed at them by name. No planner, no designer, no quorum — the row specifies its own scope and the parent doc's quorum is spent. Rotation pointer untouched at `claude:claude-fable-5`; Fable **unspent**. Metered **$0.00** of $5 — codex and sonnet are quota buckets. No GPU, no `rig.lock`. Gates on **darwin/arm64** only; windows and ubuntu legs unrun locally and read from Gate 3b's matrix. Gate-5: **NO skill edit** — this iteration's two frictions (an unscoped `sed` mutation killing the wrong arm; an assertion satisfied by a fall-through) are instances of rules the skill ALREADY has (3j's corollary *read WHICH test failed*, and 3i's *what else writes this value*), so they go to the log's Ruled-out rather than into the rulebook. Next: row **6f**.

## STATUS 2026-08-24 — ITERATION 21 COMPLETE: **THE RECONCILE THE SKILL CALLS A HUMAN CALL HAD ZERO AHEAD-COMMITS TO LOSE — AND THE PREDICATE THAT MADE IT ONE CANNOT TELL "YOUR WORK WOULD BE LOST" FROM "YOUR WORK IS ALREADY UPSTREAM".** Pick was **NOT the queue head**: Mark answered `D-MOTOKO-WORKDIR-1` on `#743` at `2026-08-23T18:59:43Z` with one word, **"Yes"**, 42 minutes after iteration 20 asked it — and Gate 0's contract makes an allowlisted answer to a parked item unpark it and *become* the pick, outranking row 6e. Iteration 20's drift notice fired in the same window at **178 behind** (up from 170): the doubling-dedupe working, not a second defect. **NUMBERS RE-MEASURED FIRST-PARTY, NOT INHERITED — 178 COMMITS HAD LANDED SINCE THE ROW WAS WRITTEN, AND THE RE-MEASURE CHANGED THE VERDICT'S BASIS.** Ahead-commits: **0**, so Gate 1's reconcile obligation 1 holds **vacuously** — strictly stronger than the `git patch-id --stable` comparison it prescribes, and not knowable from the row. Obligation 2 fails **7 of 7** (`comm -12` intersection = 7; positive control `CHANGELOG.md` = **1**, negative control = **0**, so the instrument fires in both directions). Supersession re-derived: **132 of 136** added lines byte-present on `origin/dev`; the **4** absent are decision-ledger prose origin carries in a superseding form (`git grep` positive and negative controls both fired). The two untracked files **split, and only measuring them showed it**: `scripts/mission_decisions.sh` is **byte-identical** to origin's, while `tools/launchd/test_mission_routing.sh` is superseded **and would actively regress** — the local copy asserts the executor fallback still carries `:floor`, which origin deliberately dropped on 2026-08-18 with the rationale in a comment above the assertion. Discarding it removes a red, not a fix. **THE PRESCRIBED COMMAND WAS RUN FIRST AND ITS REFUSAL RECORDED AS EVIDENCE.** `git checkout -B dev origin/dev` returned **rc=1** naming all 7 files and left the tree **byte-unchanged** — that refusal is precisely what distinguishes this operation from the `reset --hard` Critical Principle 0 forbids, and it is worth a measurement rather than a route-around. Then, under Mark's authorization: backup of all 7 files plus the full `git diff` patch to `~/.ailang/backups/motoko-clone-reconcile-2026-08-24`, sha256-manifested and verified byte-identical **with a corruption negative control that fired** (append a byte → verifier reds → restore → verifier greens); `git checkout origin/dev -- <5 tracked>`; `rm` of the 2 untracked; `checkout -B` retried → `Reset branch 'dev'`. **VERIFIED, NOT ASSUMED:** behind **0** / ahead **0**, HEAD == `origin/dev` == `e3ed9467f`, `git status --porcelain` **0 lines**, `SKILL.md` **3682** lines byte-identical to the pin's copy (negative control vs `CLAUDE.md` fired), charter byte-identical to the pin's, **all 8 worktrees intact** — including `.wt-motoko-iter8-fmt`, which holds this mission's only quorum artifacts in a gitignored directory — and the backup re-verified **OK on all 7 files after** the operation. **THE NOTICE RE-ARMS ITSELF AND THAT WAS PROVED, NOT ASSERTED:** `mission-control.sh` removes `$PIN_DRIFT_FILE` below `AILANG_DRIVER_DRIFT_WARN` (25); the real branch was extracted and driven three ways — drift **0** → removed ("re-armed"), drift **178** unchanged → deduped and file kept (control fires), drift **356** → EMIT. Live drift is now **0**. **THE WEEKLY SWEEP'S FIRST ANSWER WAS AN ARTIFACT OF ITS CORPUS, AND THE RULE'S OWN PRECEDENT SAYS SO.** Against motoko's four docs the orphan count is **67 of 76**; this repo's issue queue is **shared with V1**, and iteration 170's instance of this rule measured *"zero mentions across ALL FOUR mission docs"*. Re-swept across all **9** mission docs the real count is **8 of 76** (enumeration asserted against `gh issue list … | wc` = 76; positive control `#558` = **57** hits; negative control fired, literal not published). Filed as row **6f**: **2 are motoko's** — `#842`, *provider failure masked as successful empty completion*, which is this mission's own harness false-green class and sits under the *never conclude model wall* guardrail, and `#839` — and **6 are not**, handed to V1 on the cross-mission channel and **explicitly not triaged here**. All 8 internally authored, so no external-origin class applies. **ROUTING: NO SPRINT.** No designer, planner, executor or evaluator was spawned — the pick is a human-authorized ops action whose procedure Gate 1 writes out step-by-step with machine-checkable postconditions, so adding a judge would have given it nothing to judge. Rotation pointer untouched at `claude:claude-fable-5`; Fable unspent; metered **$0.00** of $5. Gate 0/1 clean: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, ran from the pin root detached and clean at `e3ed9467f` == `origin/dev`; running-skill check done on the **RESOLVED** symlink target, which resolves to **V1's** checkout and is byte-identical to origin. dev **verified green, not merely un-red**: **20** exact-SHA checks, **0** not-green, `runs_total=3` so a run exists, parent control **16**. Ledger valid at **4** rows, **0 OPEN**. Inbox: **1** unread, this loop's own drift notice. Bookkeeping thread **ROTATED** (`#743` created `2026-08-17T05:48:23Z` = 07:48 **local**, before today's Monday-07:00 **local** boundary — the timezone is load-bearing). Gate-5: **NO skill edit** — both frictions are instances of shapes the skill already names, pre-registered as instance 1 rather than spent on one datapoint. Next: row **6e**.

## STATUS 2026-08-23 — ITERATION 20 COMPLETE: **ROW 6d LANDED — AND THE REASON THE CLONE REACHED 170 COMMITS STALE UNNOTICED IS A DOCUMENTED ASSUMPTION THAT HAS GONE FALSE, NOT AN OVERSIGHT.** Pick was the queue head, row **6d**. PR [#838](https://github.com/sunholo-data/ailang/pull/838) → [`50460040c`](https://github.com/sunholo-data/ailang/commit/50460040c), evaluator **PASS 90/100, ZERO blocking**; three of five non-blocking findings answered in-iteration, two accepted as reported and said so. Gate 3b GREEN on head `15b89569d` (**21** checks, 0 pending, 0 not-green, 4/4 required, `CLEAN`). **THE ROW ASKED FOR A DECISION AND THE MEASUREMENT FOUND A MECHANISM.** `pin-root.sh` re-execs every driver into a worktree pinned to `origin/dev`, so the driver, skill and charter move together — and it leaves the SOURCE CLONE behind, while `mission-control.sh` posts to the human channel only when `PIN_STATUS=STALE`. The comment above that emit block states the reason in full: *"The shared clone being behind is not itself reportable — once drivers pin, that drift is harmless, and posting it every 90 minutes would train the channel to be ignored"*. **Half of that is true and half is measured FALSE.** Drift is harmless to the DRIVER, whose pin holds; it is not harmless to a HUMAN SESSION, because this charter named that clone the working checkout and a session started there resolves ITS `.claude/skills/`. The driver log carried the growth and nothing else did — **119 → 132 → 144 → 159 → 170** over five fires, on the SUCCESS path, in a log nobody reads. That is Critical Principle 2 aimed at the very helper written to close it: **guard the helper, miss the call site**, this loop's own named recurring shape. Fix keeps the true half — notify on crossing `AILANG_DRIVER_DRIFT_WARN` (default 25), thereafter only on a **doubling**, persisted in a per-mission state file removed below threshold so it re-arms after a reconcile. **THE CONTROLLER CAUGHT THE EXECUTOR'S NOTICE NAMING THE WRONG PATH, AND THE PATH IT NAMED IS SELF-REFUTING.** The body used `$REPO`; on the pinned pass `pin-root.sh` has already exported `MISSION_WORKDIR=<pin worktree>` and `mission-control.sh:40` derives `REPO` from it — so the notice would have told a human to reconcile a detached throwaway **whose drift is 0 by construction**. Verified from this session's own live environment: `MISSION_WORKDIR=~/.ailang-driver-pin/motoko`, `AILANG_DRIVER_SRC=~/dev/sunholo-data/ailang-motoko@170`. The pre-existing STALE body's `$REPO` is CORRECT — that arm fires only on the pre-exec pass — which the evaluator confirmed independently, so it is one arm right and one arm wrong in the same block. Suite **17 → 27** arms, awk-extracted from the real blocks; six mutants, each LANDED by sha256 and `bash -n` rc=0, red sets **produced by running them** (the previous message asserted a set nobody had run — the measured neutering set is **7** arms, not 4). **GATE 3b's OWN INSTRUMENT NEARLY MIS-ATTRIBUTED THIS:** `launchd drivers (bash 3.2)` came back **cancelled after 15m18s** against a **~68s** success on dev's own HEAD and on the other 18 of the last 20 CI runs — a red in a direction nothing in the diff predicts. Separated by rule 3d's strongest control, a re-run on a **byte-identical tree**: attempt 2 **success in 88s**. The diff touches no file the probe suite reads. The hang is in iteration 19's own arm 33 (*descendant discovery refuses on the real wall-clock deadline*), first observation, filed as row **6e** rather than declared a flake class off one instance. **PARKED FOR MARK: `D-MOTOKO-WORKDIR-1`** — the reconcile itself, which Gate 1's obligation 2 and `pin-root.sh`'s own header both make a human call. Phase-0 re-measured this iteration and still CLOSED: G1 `#154` OPEN (control `#175` MERGED), G2 rc=128 with its README control rc=0, G3 registry `latest=2.2.0`. Next: row **7**.



## Premise Verification Log

Added at iteration 0 after `gpt5-6-sol` **blocked** ratification on it: this charter makes
operationally decisive claims — isolation, routing, gating, queue order — and a reader cannot
otherwise tell a measured claim from an assumed one. Every row below was run on **2026-08-12**
against `sunholo-data/ailang@98ffaf5cf` (this checkout), `arniwesth/motoko_agent@303d8697`
(`origin/main_dst`), and `sunholo-data/ailang-packages` working tree.

**Acceptance rule (the reviewer's, adopted): iteration 0 may not ratify while any safety-, routing-,
or queue-ordering premise is UNVERIFIED.** New claims added later carry a row or the label
`UNVERIFIED — blocks ratification`.

| # | Claim | How measured | Result |
|---|---|---|---|
| V1 | No cross-mission lock exists, so a shared tree would contend | read `rig-lock.sh` header (scope = eval jobs); grep `mission-control.sh` for `flock/LOCK` | **Confirmed** — only per-mission `PIDFILE`/`BLOCKED_FILE`. V1 was mid-iteration (pid 71129, 70 min) during this bootstrap |
| V2 | Any `MISSION_NAME` ≠ `v1` gets fully namespaced state | read `mission-control.sh` lines 72-80 — every path interpolates `${MISSION_NAME}` | **Confirmed** |
| V3 | motoko's pidfile cannot collide with V1's or World's | driver dry-run printed `pidfile=…/mission-motoko.pid`; `ls ~/.ailang/state/*.pid` | **Confirmed live** — `mission-control.pid` (v1), `mission-world.pid`, `mission-motoko.pid` distinct |
| V4 | The three Gate-3b CI workflow names exist in this repo | `gh workflow list --repo sunholo-data/ailang` | **Confirmed** — `CI`, `Build and Release`, `Deploy Documentation to GitHub Pages`, all `active`. NB: a local `for f in .github/workflows/*.yaml` check printed **nothing** because zsh aborts on an unmatched glob — a rule 3a(i-d) instrument failure, caught only by a control. The API is the authority here |
| V5 | `provider:model` routes via `provider_executor`, not the Agent tool; codex is the only lane today | read the skill's cross-provider recipe (regex `^([a-z_]+):(.+)$`) | **Confirmed** |
| V6 | Role defaults resolve as stated | `MISSION_PROFILE=motoko MISSION_DRY_RUN=1` | **Confirmed live** — `designer=claude:claude-fable-5 planner=codex:gpt-5.6-sol executor=codex:gpt-5.6-sol evaluator=sonnet` |
| V7 | Executor fallback chain is codex → pi:deepseek(:floor) → opus | read `MISSION_EXECUTOR_FALLBACK` / `MISSION_PLANNER_FALLBACK` defaults | **Confirmed** |
| V8 | Fork delta is 52 ours-only / 805 theirs-only | `git rev-list --count` both directions | **Confirmed** |
| V9 | 12 of our packages pin ABI `2.2.0` | `grep -l '"sunholo/motoko_ext_abi" = "2.2.0"' */ailang.toml` | **Confirmed** — 12 |
| V10 | 14 of 18 `motoko_profile:` entries name an absent profile | `grep -oE` count over `models.yml` | **Confirmed** — 14 of 18; only `ollama` survives |
| V11 | 5 of 6 profiles absent from `main_dst` | `git ls-tree -d origin/main_dst .motoko/config/` vs local `ls` | **Confirmed (negative existence)** |
| V12 | `motoko_ext_fmt` absent from `main_dst` | `git grep -il 'motoko_ext_fmt\|ext-fmt' origin/main_dst` | **Confirmed (negative existence)** — zero hits |
| V13 | His vendored extensions diverge from our published ones under the same name | compare `compaction_ai` `ailang.toml` version + blob sizes | **Confirmed** — his `0.3.0` = 33,851 B; our published `0.3.2` = 9,454 B |
| V14 | Motoko has no reachable subscription lane | `jq .auth_mode ~/.codex/auth.json` (= `chatgpt`, OAuth token object); motoko provider block is `openai_chat` + bearer-from-env | **Confirmed** |
| V15 | Kimi/DeepSeek prices and the stale pin | live `GET https://openrouter.ai/api/v1/models` | **Confirmed 2026-08-12** — prices move; re-measure before acting on queue item 6 |
| V16 | Separate checkout is not a skill fork | `readlink ~/.claude/skills/mission-control` → V1's checkout; motoko's copy is `git ls-files`-tracked; `cmp` the two | **Confirmed** — and currently **byte-identical**, because V1 landed its Gate-5 edit and this checkout pulled it. Convergence-via-git demonstrated, not just argued |
| V17 | Mission iterations never take `rig.lock`, so a local-model executor has no GPU-lock story | `grep rig.lock tools/launchd/mission-control.sh` → line 15 states it explicitly. **Same-path control**: `grep -rl 'rig-lock.sh' tools/launchd/` → `os-rotation-filler.sh`, `nightly-lang-eval.sh`, `nightly-eval.sh` | **Confirmed (negative existence)** — the instrument finds takers, and mission-control is genuinely not among them |
| V18 | R3 (cross-model generality) has never been run | `grep -rn R3` over the archive + analysis log → 2 hits, both *planning* prose in the archived charter. **Same-path control**: `grep -c 'R1\|R2'` on the analysis log → **33**, so that log does record this class of work | **Confirmed (negative existence)** — R3 is absent from the log that would hold it |
| V22 | **A NEW checkout must be opened in Claude Code once, or every controller probe hangs** — cost this mission its entire first fire (iteration 1, 2026-08-12) | driver log: all 3 models in `MISSION_MODEL_PREFS` timed out at 120s ×2, captured output *"accept the trust dialog, or set `projects[…ailang-motoko].hasTrustDialogAccepted: true`"* → *"NO usable model in prefs … Refusing"*. Compared `~/.claude.json`: `ailang` (trust=T, onboarded=T, **works**), `ailang-world` (trust=**F**, onboarded=T, **works** — iteration 75 rc=0 the same morning), `ailang-motoko` (trust=F, onboarded=**absent**, **fails**) | **Confirmed as a defect; MECHANISM CORRECTED 2026-08-12 after iteration 1 measured the counterfactual.** The original row claimed *"the discriminator is `hasCompletedProjectOnboarding`, NOT the `hasTrustDialogAccepted` the error names"* — **that was wrong.** Mark then set trust=T and left onboarding ABSENT, and the 10:32 fire probed **`controller=claude-opus-5 via probe ok`**. So `ailang-motoko` (trust=T, onboarded=absent) WORKS, alongside `ailang-world` (trust=F, onboarded=T) which also works: **either flag suffices, and motoko simply had neither.** The error message's own advice was correct. The reasoning error is worth keeping: three config snapshots were used to pick the variable that *correlated*, with no test of the counterfactual — World licensed "trust=F can work", never "trust=T alone cannot". A cost of the separate-checkout decision that no reviewer flagged and the bootstrap guide did not list (now prerequisite 5). Presents as *"no usable model"* — indistinguishable from a quota outage unless you read the captured probe output. **NB onboarding is separately load-bearing for the #558 driver pin** — see PR #667, which gates the pin root on it |
| V19 | Verify profile `go-compiler` works **in this checkout** | `make quick-install && make build` run here 2026-08-12. **The inherit-from-V1 claim was rejected by `gemini-3-1-pro` and it was right**: `bin/ailang` is a per-working-tree artifact, so V1's measurement cannot speak for this tree | **Confirmed first-party** — both binaries build; `bin/ailang` and `~/go/bin/ailang` both report `v0.33.0-149-g4a45e993d-dirty` |
| V20 | **`make quick-install` writes a SHARED path** — discovered by running V19 | after the V19 build, `which ailang` → `~/go/bin/ailang`, `ailang --version` → `4a45e993d-dirty`, i.e. stamped from **this** checkout | **Confirmed — and it is a cross-mission side effect.** The system binary that V1's iterations and the eval rig use was replaced by a build from the motoko tree. Benign here (both on `dev`, delta is docs-only) but NOT benign in general. See Guardrails |
| V21 | An executor-lane demotion is **logged but never surfaced to the human channel** | `grep` the driver: fallbacks `log` at lines 360 (codex→fallback) and 392 (pi→opus); `gh issue comment` appears at 4 sites — no-usable-model refusal, controller model change, post-record late kill, and iteration failure. **None is the executor/planner lane demotion.** Control: the driver does call `gh` (8 hits), so the instrument works | **Confirmed (negative existence).** The gap `gpt5-6-sol` named is real and it affects **all three missions**, not just this charter. Queue item 2 |

## CURRENT GOAL

1. ~~**Iteration 0 (definition)**: ratify the bar with Mark through the design quorum.~~ **DONE
   2026-08-12** — quorum blocked twice, all four objections measured and true, three fixed in-doc
   and the fourth (V21) escalated; Mark ratified the bar and queue, and routed the V21 driver fix.
2. **Now**: work the queue through the inner loop (design-doc → sprint-plan → execute → evaluate),
   one sprint-sized item per iteration, recording routing evidence every time.

**What changed under the old charter, and why this is a rewrite rather than an edit.** The June
charter's frontier was (1) canonical prompt delivery, (2) respect context length, (3) uncaught
harness errors, (4) AILANG-native power. As of 2026-08-12: (1) is **solved**; (2) is **superseded
upstream** — Arni's affine calibration is anchor-size-insensitive where our ratio calibration was
not, which was the actual bug; (3) is **partly superseded** — `motoko-ext-empty-stop-guard`
reimplements our empty-response retry as a pure budgeted `on_solver_candidate`. Only (4) survives
intact. Meanwhile the whole tree beneath us has been rewritten (see the queue's epic).

## DST scope — what it actually covers, measured (2026-08-12; corrects an earlier over-read)

Arni, on handing the refactor over: *"Doing proper DST of extensions turned out to be exquisitely
complex. That is basically an open research project."* His own closing note
(`.agent/projects/009_motoko_dst_execution/NOTE-d28`, HEAD `b3953a9`) quantifies it. **Plan against
these numbers, not against the ambition.**

**The CORE is strongly covered** — and this is why adopting the refactor is still right:
11/11 acceptance rows across three profiles; **9 of 11 fault classes and 9 of 11 NAMED production
recovery branches reached**, so recovery paths execute under injected faults rather than merely
existing; seeded generation byte-identical at equal seed and distinct across seeds; a virtual clock;
exact-program strict replay; per-variant ledger parity (`ProviderResult 15/15`, `RunSummary 8/8`);
a blocking fixed-seed CI corpus plus a rotating day-keyed one.

**EXTENSION coverage is very nearly nil, and his tooling says so out loud rather than hiding it:**

| profile | covered hooks | extensions | substantively world-mediated |
|---|---|---|---|
| `driver_plus_compose` | 7 | 1 | **1** |
| `driver_plus_no_ops` | 32 | 4 | **0** — *"entirely of no-ops"*, 16 satisfying criterion 2 **vacuously, over an empty set of performed effects** |

**≈1 of 40 covered hooks is substantively simulated, across 15 extensions.** The note states it as
"one-of-forty" deliberately — *"the difference between reporting the demonstration and overclaiming
from it"* — and `tools/profile_definition/check_no_op_profile.py` **fails the build** if a non-zero
coverage number is stated without its vacuity qualifier. That is unusually honest engineering; treat
the numbers as trustworthy.

**What this means for THIS mission, whose entire value is in extensions.** DST gives us a
**contract layer, not a simulation layer**: `make declared_vs_performed` (hook effect rows checked
against measured behaviour by two independent producers), `conformance`, `hook_guard`,
`ext_call_inventory`, `ext_ambient_inventory`. Genuinely useful — it catches a lazily-widened effect
row during the 12-package ABI port, which is the mistake that port most invites. It will **not**
tell us whether `fmt` saves tokens, whether a compaction strategy converges, or whether μRAG helps.
Those stay rig questions and must be priced as such.

**Do not repeat the over-read.** This charter's first draft made "answer an A/B via DST instead of a
rig run" a success metric of the migration. That mistook the core's maturity for the framework's
reach. If extension-level DST is ever solved upstream it changes our economics completely — watch
for it, do not assume it.

## The bar — what "motoko is the best AILANG harness, honestly measured" means (**RATIFIED by Mark, 2026-08-12**)

- **Clause 1 — It builds and gates green from source.** The tree our evals run is rebuildable and
  passes `make check_core && make verify_extensions`. A harness we cannot rebuild is not a harness
  we can improve.
- **Clause 2 — No extension drift.** Every extension we own is published, registry-pinned, and
  ABI-current. No vendored `{path=...}` copy diverging from a published version under the same name.
- **Clause 3 — Every carried improvement is measured, and RE-measured when the tree moves.** No
  improvement survives on assumption across an architecture change. An unmeasurable improvement is
  dropped, not carried.
- **Clause 4 — Profile↔model routing is explicit and resolves.** Every `motoko_profile:` entry in
  `internal/eval_harness/models.yml` names a profile that exists. No implicit defaulting (the
  failure that once gave cloud eval models neither the AILANG-knowledge extensions nor a verify gate).
- **Clause 5 — Motoko exploits what a generic harness cannot.** Typed-interface reads, AST
  edits/queries, contracts + Z3, exact best-of-N — the moat, and the reason this mission is not
  "make a good agent loop".
- **Clause 6 (META — the loop closes) — Motoko graduates into the mission executor fleet.**
  `motoko:<model>` becomes a valid `MISSION_EXECUTOR_MODEL`, so the harness this mission improves
  becomes a harness that *does* the improving. This is the strongest available dogfood and the
  operational proof of [PROGRAM.md](PROGRAM.md)'s self-specializing thesis: a harness good enough to
  land its own sprints is good enough, in a way no benchmark score argues for on its own.

  **What graduation concretely requires** (from the landed `codex` lane, M1b — currently the *only*
  cross-provider executor):
  1. A `provider:model` spawn recipe in the shared skill's cross-provider section — `motoko:<model>`
     matched by `^([a-z_]+):(.+)$` and routed via `provider_executor`, NOT the Agent tool.
  2. A **bounded, token-cheap pre-flight probe** (Standing rule 6 — never unbounded), plus a place in
     the driver's fallback chain **that posts a loud degradation notice to the bookkeeping issue and
     names the lane, the probe's exit code, and the model actually used** — so a dead lane degrades
     rather than wedges, *and never degrades quietly*.

     **This clause was BLOCKED at iteration 0 by `gemini-3-1-pro` and the objection was correct.**
     The first draft asked for a fallback slot "so a dead lane degrades rather than wedges" with no
     alerting requirement — in a charter that, two sections earlier, cites the World mission losing
     **five iterations** to the codex lane being silently demoted to opus. That is Critical Principle
     2 (NO SILENT FALLBACKS) violated in the document that quotes the precedent. A fallback whose
     degradation is only visible in a routing-evidence row nobody reads is the same defect wearing a
     different hat: the Gate-4 row is written *after* the iteration already ran on the wrong lane.
     The signal must fire at degradation time, not at reporting time.
  3. A real-run recipe that survives what a real coding sprint needs: a write sandbox that also
     reaches build caches outside the worktree, a background spawn (the 30-min cap exceeds the
     harness's 10-min foreground `Bash` limit), and `< /dev/null`.
  4. **The false-green guards that killed the DeepSeek-Flash lane** — it went 3/3 FAILED on real
     sprints while reporting `rc=0` with an empty worktree. Assert directive delivery before
     spawning; a silent success is the failure mode to design against, not an edge case.
  5. A gate trial on real sprints — plan-faithful landing of held-out tests, not a smoke reply.

  **First target is an AILANG-source repo, NOT this one.** This mission's anchor repo is
  `sunholo-data/ailang`, a **Go** repo on the `go-compiler` verify profile — motoko has no structural
  advantage writing Go, and would be graded against `codex` precisely where its moat does not apply.
  The natural first lane is a repo on the `ailang-code` profile, where `ailang check` / `ailang test`
  / `ailang ai-check` *are* the gates: **Ailang World** is AILANG source and already runs that
  profile. Expect motoko's executor graduation to land on World before it lands here, and treat a
  Go-repo trial as the harder, later bar rather than the starting one.

  **MOTOKO HAS NO SUBSCRIPTION LANE, AND CANNOT GET ONE.** Measured 2026-08-12; recorded here so
  nobody spends an iteration trying to bridge it. Both subscription buckets the fleet currently
  runs on are **bound to a CLI client**, not reachable as an API:
  - *Anthropic* — the Claude Code OAuth path. Motoko is a different harness; it cannot present it.
  - *ChatGPT/codex* — `~/.codex/auth.json` reports `auth_mode = chatgpt` with an OAuth token object.
    That credential is bound to the codex CLI, and motoko's providers are
    `request_shape = "openai_chat"` + `auth = { type = "bearer", env = … }` — standard OpenAI chat
    to a URL, which subscription OAuth does not speak. `OPENAI_API_KEY` *is* present, so motoko can
    reach OpenAI — but **metered**, with no advantage over OpenRouter.

  So motoko's lanes are exactly two: **OpenRouter (metered)** or **local GPU ($0)**. This sharpens
  the strategy rather than weakening it — the local lane is the only executor the fleet can ever gain
  that ADDS capacity instead of spending it, which is most of clause 6's value.

  **The local lane needs a GPU-lock story that does not exist yet.** The driver is explicit that
  mission iterations *never* take `rig.lock` because they are cloud-model work (GPU-touching sprint
  steps take it per-step, inside the session). A motoko-*local* executor is GPU work for the whole
  sprint, so it would contend with the nightly evals and the OS rotation. **OpenRouter-backed
  `motoko:` lanes have no such problem and are therefore the easier first target**; the $0 local lane
  is the bigger prize and the later one.

## Guardrails (mission-specific; the skill's Standing Rules always apply on top)

- **THE PHASE-0 GATE IS REAL. Do not start the ABI 5.0 extension port until BOTH:
  [#154](https://github.com/arniwesth/motoko_agent/pull/154) is merged to `origin/main`, AND Arni
  has declared the ABI stable.** He states it "is still subject to change and the current version
  number is somewhat arbitrary." Porting 12 packages against a moving ABI means porting twice.
  **If the unblocked queue is exhausted, SAY SO and idle** — record a no-op iteration rather than
  pulling gated work forward. An idle iteration is a correct outcome here, not a failure.
- **We are GUESTS in `arniwesth/motoko_agent`.** Never push to it. PRs only, never force a draft to
  ready, and never re-open something the maintainer closed without Mark. He is hands-on.
- **Three repos, one mission.** `sunholo-data/ailang` is the anchor (evals, benchmarks, design docs,
  and the only issue queue). The motoko fork worktrees (`~/dev/mk-*`, see [MOTOKO.md](../MOTOKO.md))
  and `sunholo-data/ailang-packages` are work surfaces, not mission repos. Gate 3b CI applies to the
  anchor.
- **V1 OWNS DEV CI RED ON THE ANCHOR. YOU DO NOT — hand it over and keep your own pick.** V1 and this
  mission both run `MISSION_REPO=sunholo-data/ailang` (separate clones, one GitHub repo), so a red dev
  is visible to both — and the skill's rule that a red "hits whoever observes next" silently assumes a
  single observer. There is not one. The driver's overlap guard is per-mission by construction
  (`PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"` — each loop guards only against *itself*), and no
  cross-mission mutex exists; iterations deliberately never take `rig.lock` (GPU mutex only, driver
  line 15). **Measured 2026-08-17:** both loops preempted onto the same red and opened
  [#758](https://github.com/sunholo-data/ailang/pull/758) (this mission, iteration 9) and
  [#759](https://github.com/sunholo-data/ailang/pull/759) (V1, iteration 217) — **the same six files**,
  both adding `scripts/test_check_changelog.sh`, both moving the same 169 lines out of `CHANGELOG.md`.
  Gate 2's open-PR check cannot catch this: V1 ran it at ~18:58Z, before #758 existed at 19:05Z, so it
  is a point-in-time query aimed at a *past* iteration's abandoned work and is blind to a concurrent
  peer. **Third** instance of cross-mission contention on work rather than on a git ref — iterations 5
  and 6 both collided on `changelogs/v0.18-current.md`, the same file again here. So: **on observing a
  red dev, record it, hand it to V1 via the cross-mission channel, and proceed with your own pick. A
  red you do not own never outranks it.** ONE carve-out: if the red is *yours* — a commit or PR from
  this mission caused it, or it sits in motoko/eval-lane territory V1 has no domain knowledge for —
  you own the fix, because handing that to V1 strands it.
- **Never run `ailang fmt` across motoko sources.** It reflows whole expressions and inserts blank
  lines between imports — hundreds of lines of conflict surface against a fork we must stay
  mergeable with, for no benefit.
- **[MOTOKO.md](../MOTOKO.md) is roughly half-stale until the migration completes** (§3 packaging,
  §4 profiles, §5 the retired a2a deferral, §7 upstream delta). Verify against the tree before
  citing it; rewriting it is a success criterion of the migration doc.
- **`make quick-install` is a SHARED WRITE — treat the verify profile as touching V1 and the rig**
  (measured V20, 2026-08-12). It installs to `~/go/bin/ailang`, the binary V1's iterations and the
  eval rig resolve through `PATH`. A separate checkout isolates the *working tree*, not the installed
  toolchain. So: before any gate that runs `quick-install`, confirm this checkout is not behind
  `origin/dev`, and never run it from a tree carrying experimental compiler changes. If a gate needs
  an experimental binary, build to `bin/ailang` (`make build`) and invoke it by path — do not install
  it. A rig eval that silently ran on a mission's half-finished compiler would be indistinguishable
  from a language regression.
- **Never conclude "model wall."** Every motoko disengagement investigated on this mission so far
  has been a harness bug. Prove it with `ailang chains` / the wire bytes before claiming capacity.

## Routing policy

Uses the **shared** per-role model routing from `mission-control` (controller / designer-rotation /
planner / executor / evaluator, generator≠judge enforced). Overrides for THIS mission live in
`~/.config/ailang/mission-motoko.env`:

- **Executor default**: inherits the driver's ratified chain (`codex:gpt-5.6-sol` →
  `pi:deepseek(:floor)` → `opus`), which is already non-Anthropic-first — so a second concurrent
  loop does not double the Anthropic burn.
- **Evaluator**: inherits the shared default (differs in provider from the executor).
- No other overrides. `PATH` is set in the env file to include `/opt/homebrew/bin` — see the World
  mission's iter-18-to-22 lesson, where a PATH-less plist silently demoted the codex lane to opus
  on **every** fire for five iterations before anyone noticed.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**Current epic**: [m-motoko-dst-refactor-migration](planned/m-motoko-dst-refactor-migration.md) —
adopt Arni's phase-core/DST refactor, re-prove our improvements. The epic is gated; the items below
are ordered so the UNGATED work runs first.

1. [LANDED 2026-08-12] **Iteration 0 — ratify this charter** · all clauses · quorum x2 ($0.115),
   4/4 objections true; bar + queue **RATIFIED by Mark**
2. [LANDED 2026-08-12 for v1+motoko · **World still owed**] **Loud lane-degradation notice in the driver** · clause 6 + Critical Principle 2 · **CROSS-MISSION
   DEFECT, found by `gpt5-6-sol` at iteration 0 and measured as V21.** When the codex or pi lane
   probe fails, the driver `log`s the demotion (lines 360/392) and continues — it never posts to the
   bookkeeping issue, so the human channel sees nothing. That is precisely how World lost five
   iterations to a silently-demoted codex lane. Fix: on any lane fallback, post the failed lane, the
   probe's exit code/timeout, and the model actually used, **before execution continues**. Affects
   **v1, world and motoko** — so it needs Mark's routing call (driver is frozen core) and probably
   belongs to whichever mission owns the driver, not automatically to this one · 1 iteration
3. [LANDED 2026-08-12 · iteration 1 · **Gate 3b GREEN** — confirmed on a second bounded poll:
   16 checks present, 0 pending, 0 not-green, `test: completed/success`. The first poll timed out
   with `test` still running and was recorded as *not* green, which is the rule working]
   **Disposition all 52 fork commits** · clause 3 ·
   [m-motoko-fork-disposition.md](planned/m-motoko-fork-disposition.md), `752254d3f` —
   **14 SUPERSEDED / 16 PORT / 14 DROP / 7 UNRESOLVED**. Split into its own file: the migration doc
   is the decision, this is the ledger. Three corrections came out of it: the range is 52 commits
   but only **51** are dispositionable (`ed61097` is a content-free merge); **path existence is not
   a supersession signal** here (80 of 94 paths survive upstream as facades — `agent_loop_v2.ail`
   is 4,005 B there vs 95,868 B here), which retired the obvious instrument before it was used; and
   an independent evaluator **failed the first pass 65/100**, correctly, on 2 of 16 SUPERSEDED rows
   — the one verdict whose error is irreversible. **The 7 UNRESOLVED rows are the honest residual**,
   each naming its settling measurement; Phase 3 is not done until they are settled.
   **Ledger moved 2026-08-14 (iteration 4): R8 settled → PORT by measurement, so the counts are now
   14 SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED = 51.** First of the seven to close, and it
   closed on the row's own decision rule rather than on a re-read. The counts in this row and in
   the iteration 1–3 STATUS stamps are left as written — they are the record of what was true then;
   the live ledger is [m-motoko-fork-disposition.md](planned/m-motoko-fork-disposition.md)
4. [LANDED 2026-08-12 · iteration 2 · **D1 RESOLVED by Mark — G5 is a predicate**] **Designer pass
   on the migration doc + re-quorum ONCE** — **DONE, and R1's objections are answered**
   (`1d0e2e511`): Phase 0 is now a bounded fail-closed gate (4 conjunctive predicates with their
   commands and observed values, a 28-fire ~14d timebox, structured BLOCKED expiry, declared human
   residual) and the four Port claims carry rows V21–V24 with same-scope controls. **The one
   re-quorum is spent and it BLOCKED on two NEW consistency defects**, both measured: `gemini`'s
   Design Freeze **deadlock** and its `on_pre_step` "ten effects" claim (both FIXED — V26 shows the
   row is `! {AI, IO, Trace}` and the cited ABI commentary is one upstream *retracted*); and
   `gpt5-6-sol`'s finding that the gate starts Phase 1 on G1–G4 while the **ratified charter
   guardrail** demands Arni's declaration too. That last one needs **Mark**, so the doc is parked
   rather than force-passed. **D1 — RESOLVED by Mark 2026-08-12: Arni's ABI-settled acknowledgement is a GATE
   PREDICATE (G5), not an accepted risk. "We wait for Arni's ABI settled acknowledgement."**
   So Phase 0 is G1-G5 conjunctive: the four measured predicates AND Arni's declaration. This
   matches the ratified guardrail rather than relaxing it, and it means the gate CANNOT open on
   registry evidence alone — the 5.x-at-a-pinned-digest predicate (iteration 2's non-vacuous
   find) is necessary but not sufficient. Human residual is therefore permanent by design, not
   a gap: the timebox escalates to Mark, it does not self-open. Nothing is blocked meanwhile: Phase 0 measures CLOSED (G1/G2/G3 all FALSE).
   Also fixed here: **V25**, a defect in the designer's own output — G2 written as bare
   `origin/main` is `rc=128` from this checkout and indistinguishable from the FALSE it reports, so
   the gate could never have opened · *(historical: the quorum **BLOCKED** it at
   pick time (2026-08-12, both reviewers present, $0.064) — **both R1 objections are now ANSWERED
   and were not re-raised at R2; this paragraph is kept only as the trail.**
   Iteration 1 measured them rather than forwarding them (rule 3f), so the designer got numbers,
   not opinions: **obj 2** (`gemini-3-1-pro`: the four "Port — carry forward" claims are unverified)
   is procedurally right and **substantively refuted 4/4** — it needs verification rows, not a
   redesign. **obj 1** (`gpt5-6-sol`: Phase 0 is an unbounded wait with no machine-verifiable
   stability condition) is **upheld, and its obvious remedy is vacuous** — ABI 5.0 declares
   `[stability] level = "stable"` and so does the 2.2.0 we call unstable, so a gate on that field
   passes immediately and falsely. A real bounded condition is a design call: candidates are
   `#154` merged (objective, currently OPEN), the ABI version string unchanged for N days, or an
   explicit word from Arni — plus a re-check cadence and a defined action on expiry)*
5. [**LANDED 2026-08-18 · iteration 11 — upstream issue [#165](https://github.com/arniwesth/motoko_agent/issues/165) FILED; `#97` CLOSED as superseded**; historical tags kept below as the trail: IN-PROGRESS — case CORRECTED and BOUNDED 2026-08-13 (iteration 3); filing gated on one
   measurement, not on a human] **Output-headroom upstream issue** · clause 3 · ~~file the case
   against `main_dst` (qwen3 arithmetic + the `docx_lambda` failure) **if Arni's #97 reply invites
   it**~~. **The precondition was FALSE and is an unbounded wait.** Measured 2026-08-13: **zero**
   `arniwesth` events on PR #97 across all four surfaces (issue comments, review comments, reviews,
   timeline) since our 2026-08-11T19:39Z comment — *control fires*: `commenter:arniwesth` in that
   repo returns **34** issues/PRs, so the instrument can see him. Waiting on a third party with no
   expiry is the same defect `gpt5-6-sol` blocked Phase 0 for at iteration 1, and it was sitting in
   the queue head unnoticed one iteration after we fixed the identical shape next door.
   **What iteration 3 actually delivered — the unblocked half, which is the evidence:** the case as
   written was wrong in both directions and would have filed a defensible-looking but incorrect
   issue. Re-measured against `main_dst@6c06b08` (migration doc **V27–V29**): (a) the live ladder
   `compact_for_pre_step` targets **70%**, not the 95% we asserted, which on a 262144 window leaves
   **≥78,644** — *more* headroom than the 75k reserve `96542f8` adds, so upstream already beats our
   patch **when the ladder reaches its target**; (b) the mitigation we credited,
   `try_emergency_compaction_with_limit`, has **zero production callers** — the real refusal is the
   phase core's `seal_compacted_payload` at `exhaustion_pct() = 95`, which *does* fail loudly
   (`ContextExhausted`, `retryable: false`), so "hard-stop, not silent corruption" is confirmed at a
   different function than we named; (c) the residual is the **band between the ladder's 70% target
   and the seal's 95% permission**, and the seal's predicate is **input-only**. Net effect: the ask
   shrinks from "adopt an output reserve in a compaction extension" to **one argument at
   `session.ail:2561`** — and upstream already does exactly that move one line up for the pinned
   system prefix (`context_limit - pinned_tokens`), which is the strongest possible precedent to
   cite. **BOUNDED RULE (replaces "if Arni's reply invites it"):** if no `arniwesth` response on #97
   by **2026-08-27**, file the corrected case as a standalone upstream issue citing V27–V29 and
   carry the patch locally regardless; a reply may redirect it to a PR at any time — what expires is
   the waiting, not the offer. ~~**Remaining before filing (one unit-level assertion, NOT a rig
   run):** the disposition's **R8** — is the 70→95 band reachable in practice?~~
   **DONE 2026-08-14 (iteration 4) — R8 SETTLED → PORT. Nothing technical blocks filing now.**
   The instrument was built and run (`tools/motoko/r8_headroom_band.ail` against
   `main_dst@6c06b08`, replicating the live `session.ail:2534-2561` wiring): the ladder returns
   `structural: tier=floor keep_last=1` at **79%**, the seal sees 79% < 95% and returns **`Ok`** —
   it SENDS — leaving headroom **54,905 < the 65,536 output cap**. Both controls fire (small
   history → `PassThrough`/`Ok`; 158% → `Err(SealExhausted)`), so that `Ok` is a real permission,
   not a dead gate. **The mechanism is sharper than the case we were about to file, and it changes
   the argument.** The ladder does not "grind down" — it has **no lever at all**: `elide_walk`
   only rewrites `role=="tool"` messages, so a large **user** message (the `docx_lambda` shape, a
   pasted document) is invisible to all four tiers, which remove ~**1%** between them. So the ask
   is not "make the ladder try harder" — which is what an extension-side reserve would be — but
   "the seal is the only place that can see this". Hence **one argument at `session.ail:2561`**,
   with upstream's own `:2534` (`context_limit - pinned_tokens`) as the precedent, now confirmed
   by reading the call site first-party rather than inherited from the row.
   ~~**Remaining: the bounded wait only** — still zero `arniwesth` events on #97; 2026-08-27 stands~~
   **CLOSED 2026-08-18 (iteration 11). THE WAIT HAD ALREADY ENDED AND THE ROW DID NOT NOTICE — that
   is the finding.** `arniwesth` commented on `#97` at **2026-08-13T18:45:54Z** (control:
   `commenter:arniwesth` in that repo → **34** issues/PRs, so the instrument sees him). Iteration 3's
   *"zero events"* was measured on 08-13 and true when taken; iteration 4 wrote *"still zero"* on
   **08-14**, after the comment, and iterations 5–10 each carried the sentence forward. *Still* reads
   as a re-check and is a transcription (rule 3b(v)(b)) — and the 08-27 timebox added as the **remedy**
   for an unbounded wait is what hid the wait ending, because a timebox invites checking the clock
   rather than the predicate. **His reply grants more than the row hoped:** agreed on all four points;
   `#97` closes as superseded (nothing from that branch to revive — calibration, elision ladder and
   system-prefix pinning are all better covered by `main_dst`/`#154`); the output-headroom concern
   **is still valid**; and *"Please go ahead with the separate issue against `main_dst`"*, scoped by
   him to the effective input budget across **both** the pre-step chain and the final seal, the raw
   context limit retained for telemetry, and regression coverage for 262,144/65,536 **and** the
   unknown-/small-limit fail-open. **Filed as [arniwesth/motoko_agent#165](https://github.com/arniwesth/motoko_agent/issues/165)**,
   with every load-bearing claim re-derived first-party at `main_dst@6c06b08` — still the branch tip,
   **0** commits since V27–V29 — and the R8 probe **re-run rather than quoted** (arm A: `tier=floor
   keep_last=1` at **79%**, seal **`Ok` — sends**, headroom **54,905 < 65,536**; controls B and C both
   fire). `#97` closed with the verdict, comment count asserted **2 → 3**. One correction to our own
   record went into the issue: *"`try_emergency_compaction_with_limit` has zero production callers"* is
   false as written (one caller, `compact_step_with_limit:137`) — the substance holds, and upstream's
   own `006_compactor_strategy/PLAN-compactor-strategy.md:75` calls that function **test-only**, so the
   issue cites their line instead of repeating our overstatement. Nothing local waits on `#165`
   **CORRECTED 2026-08-18 (iteration 12) — the base moved within six hours of filing, and this row
   said it had not.** The sentence above reads *"`origin/main_dst` is **still exactly `6c06b08`**,
   **0** commits since our V27–V29 rows"*; measured at iteration 12, `git rev-list --count
   6c06b08..8110ffc` = **59** (100 files), `src/core/session.ail` among them (`+67/-13`), and
   upstream cut `arniwesth/mot-100-fix-output-headroom` in the same window. **The defect is intact
   and only the offsets moved**: the window `session.ail:2552-2561@6c06b08` is byte-identical to
   `:2606-2615@8110ffc` (sha256 `c53792eb…`; negative control, the same window shifted one line,
   `01d10306…`), so `#165`'s three `session.ail` citations shift **+54** — `2552→2606`, `2556→2610`,
   `2561→2615` — while `compaction.ail`, `compaction_structural.ail`, `phase_vocab.ail` and
   `context_usage.ail` are byte-unchanged across the range (same-call control: `session.ail` is the
   only path `git diff --stat 6c06b08 8110ffc --` returns over those five). Re-anchored upstream in
   [comment 5332596902](https://github.com/arniwesth/motoko_agent/issues/165#issuecomment-5332596902),
   delivery asserted 0 → 1 comments. **The general lesson, pre-registered as instance 1 rather than
   written into the skill on one datapoint:** every freshness rule in `mission-control` is scoped to
   a row you are about to PICK, so a LANDED row whose deliverable is an artifact published to an
   external party — pinned to that party's moving branch — is re-read by nothing. **Any future row
   that pins a claim to an upstream SHA states the SHA and the date it was measured, and is
   re-measured before the claim is quoted again**
5a. [**DIAGNOSED + FIXED 2026-08-14 · iteration 5 · PR #721** — root cause is OURS and it is one
   line. `scripts/hooks/session_start.sh` backgrounded `ailang docs embed-warmup --timeout 3m &`
   **without redirecting stdout**. A backgrounded child inherits the stdout descriptor, so a
   `$(...)`-style capture cannot see EOF until the CHILD exits — and Claude Code captures hook
   stdout. The hook was therefore held open for the warmup's whole life, bounded only by its own
   `--timeout 3m` = **180s**, against the driver's **120s** `PROBE_TIMEOUT`. Hence
   `captured output: ''` (`claude -p` is silent until it completes) and hence all three models
   failing identically — **it was never a model verdict, it is one hook stall observed six times**
   (3 models × 2 attempts). Mechanism proven in three arms: capture **8,433ms** / redirected
   **8ms** / background child removed **237ms**; one real capture measured **96,377ms** against a
   warm `embed-warmup` of ~1.2s. **The population separation was the decisive evidence and it was
   already in the logs:** v1 **47/186** refusals, motoko **6/11**, world **0/89** — world being
   the one mission with **no `.claude/settings.json`, hence no hooks** (control: world's log has
   **90** `probe ok` lines, so the zero is a measurement). **`quota-limited` has never fired once**
   fleet-wide, so the driver's "quota-limited, timed out, or errored" summary has only ever meant
   *timed out*. **Amplification measured:** the warmup is a GRANDCHILD of the killed `claude`, so
   it survives (verified `ppid=1`) — every timed-out probe leaves a GPU tenant and the next probe
   adds another, up to six per fire. **RULED OUT: GPU contention alone.** The filler held
   `rig.lock` 07:58:30→12:39:58 on 08-14 and motoko's *successful* 09:45 fire is inside that same
   window as the three refusals before it — the hypothesis fit three points and died on the fourth.
   **This corrects iteration 4's "not motoko-specific / environmental" reading**: it is common to
   v1 and motoko because both are `sunholo-data/ailang` checkouts carrying the three SessionStart
   hooks, and world is immune structurally, not by luck. Fix is a stdout redirect — NOT the
   timeout increase this item forbids — guarded by `tools/launchd/test_hook_stdout.sh` in
   `make test-launchd-drivers` (CI gate, `ci.yml:472`), mutation-verified 1s→11s RED.
   **Residual, still open and NOT closed by this fix:** the `make test-launchd-drivers` half below
   — `test_pin_root.sh` is **1 passed / 28 failed** on the rig against green in CI, re-measured
   this iteration on a pristine `origin/dev` worktree, so the one gate covering the driver scripts
   still cannot catch anything on the machine those scripts run on. Carried as **item 5b**]
   *(historical scope note, kept as the trail: inserted 2026-08-14 by iteration 4, sub-numbered to
   avoid renumbering 6–14, scoped diagnose-only. Scope was widened to the fix once the mechanism
   was known, since the remedy is a redirect and not the forbidden timeout change.)*
   **The driver's model probe hangs with EMPTY output, and it is costing this mission most of its
   fires** · loop health · **measured, not inferred.** From `/tmp/ailang-mission-motoko.log`:
   **6 refusals against 4 starts** over the loop's whole life — refusals 08-12 09:21, 08-13 00:25,
   08-14 01:19, 08-14 08:57, 09:17, 09:37; starts 08-12 10:32, 08-12 11:37, 08-13 12:27, 08-14
   09:45. That is **60% of fires refused**, and at `StartInterval=43200` (12h) each one costs a
   half-day. Only the FIRST (08-12 09:19) carried a diagnostic — the Claude Code trust dialog,
   charter V22. **Every refusal since reads `captured output: ''`** on all three models across both
   attempts: a hang, not a quota message, and the driver's "quota-limited, timed out, or errored"
   summary line flattens the distinction away.
   **Not motoko-specific** — control, same window, separate checkout and separate config:
   `/tmp/ailang-mission-control.log` shows v1 refusing with the identical empty-output signature at
   05:20, 05:36 and 08:49 on 08-14, overlapping motoko's 08:47/09:07/09:27. Two independently
   configured missions failing together is what makes this environmental rather than per-mission.
   **The symptom is now treated** (iteration 4 landed the motoko recovery job, which is why this
   iteration exists at all) **but treated by brute retry**: it kickstarted 4 times over an hour and
   only the 4th got through. Recovery masks the defect from the human channel, so this item exists
   to keep it visible. **Scope: diagnose only** — reproduce the empty-output probe under the driver's
   own invocation, establish whether it is CLI-side, auth-side or contention between concurrently
   probing missions (v1 fires every 90 min, motoko every 12h, World every 4h — they overlap by
   construction and nothing serialises them). Do NOT "fix" it by lengthening the 120s timeout until
   the mechanism is known.
   **Adjacent evidence found while gate-sweeping this iteration, folded in here rather than given
   its own row:** `make test-launchd-drivers` — which **CI does run** (`ci.yml:472`) — fails on this
   rig with **10 passed / 25 failed, rc=2**, while dev CI is **green** on the same SHA (20 checks,
   0 not-green). Baselined per rule 3e: the failure is **byte-identical on a pristine `origin/dev`
   checkout with zero local changes**, and its output never mentions the plist landed this
   iteration (0 hits), so it is neither a regression nor ours. The failure text reads the real rig
   (*"source clone … was 1 behind"*), i.e. the gate is environment-dependent and takes a different
   path in CI. That is the **inverse** of the usual class this loop guards against: not a local
   green hiding a remote red, but a remote green hiding a local red — and its consequence is that
   the one gate covering the driver scripts **cannot catch anything on the machine those scripts
   actually run on**. Worth settling in the same investigation, since both halves are about whether
   we can see our own driver's health · 1 iteration
5b. [LANDED 2026-08-15 · iteration 6 · PR #728 · recovered by iteration 7 — split out 2026-08-14 by iteration 5, which fixed 5a's probe-hang half and left this
   one genuinely open] **The one CI gate covering the driver scripts is blind on the machine those
   scripts run on** · loop health · `make test-launchd-drivers` is **green in CI** and red on the
   rig: `tools/launchd/test_pin_root.sh` returns **1 passed / 28 failed, rc=1**, re-measured this
   iteration from a **pristine `origin/dev` worktree with only unrelated files changed**, so it is
   neither a regression nor ours (iteration 4 baselined the same thing byte-identically). Its
   output reads the real rig (*"source clone … was 1 behind"*), i.e. the gate takes an
   environment-dependent path and CI's green says nothing about rig behaviour. This is the
   **inverse** of the class the loop usually guards: not a local green hiding a remote red, but a
   remote green hiding a local red. Note the new sibling `test_hook_stdout.sh` is deliberately
   environment-independent (it stubs its own slow warmup) and passes in both places, so it is a
   working control for what a rig-honest driver test looks like. Scope: make the driver gate mean
   something where the drivers run — either by fixing `test_pin_root.sh`'s environment coupling or
   by splitting the rig-dependent arms behind an explicit marker that FAILS LOUDLY rather than
   passing in CI · 1 iteration
6. [**M1 + M3 + M4 + M5 LANDED · M2's INSTRUMENT LANDED 2026-08-22 (iteration 18) BUT ITS LIVE
   VERDICT IS *VOID* — M2 IS STILL THE RESUME POINT.**
   **M2 instrument LANDED 2026-08-22 · iteration 18 · PR [#829](https://github.com/sunholo-data/ailang/pull/829)**
   (`tools/eval/motoko_connection_probe.sh` + an 8-arm hermetic self-test). The live two-lane sweep
   was RUN under `rig.lock` and came back **VOID, not FAIL**: both lanes `driver_rc=1` with peer set
   `[]` in 8m15s/8m17s, and `AC-M2-control` states that a control which does not fire makes
   `AC-M2-treatment` **VOID — the probe proved nothing**. The instrument refusing to certify on an
   absence is design doc §12.4 working as written, so this is the gate holding, not a regression.
   **The two explanations were separated by measurement, not assumed** (doc rows V36–V38): scoped
   `lsof` DOES see an ESTABLISHED peer of a child process here (unscoped same-call control: 67
   lines), and the treatment lane is `rc=0` standalone (1m53s) AND `rc=0` under a faithful
   replication of the probe's own `run_lane` shape (1m1s, 244 lsof lines) with **`127.0.0.1:11434`
   present**. So the observable is reachable and the probe AS SHIPPED is what breaks the runs it
   observes. **Mechanism NOT isolated** — it lies in what the replication did not reproduce (the
   deadline-carrying `descendant_pids`, or the two-lanes-in-one-process sequencing; the evaluator
   ruled out `classify_lsof` hermetically, since it runs after `wait` and cannot change an exit code).
   **NAMED NEXT STEP, and it is cheap: keep the driver logs.** The probe's `trap … EXIT` deletes both,
   which is why diagnosing this needed a re-run — for a lane that exits non-zero that log is the entire
   diagnostic. Evaluator **FAIL 58/100**; blocking finding B1 (an IPv6 false negative) fixed and gated
   in the same PR (row V39), blocking finding B2 carried as row **6c** · resume point: **M2**
   *(historical: M1 + M3 + M4 + M5 LANDED · ONLY M2 REMAINED, AND IT NEEDS THE RIG.)*
   **M3 + M5 LANDED 2026-08-21 · built by iteration 16, verified and landed by iteration 17 · PR [#813](https://github.com/sunholo-data/ailang/pull/813) → `b2733201a` · Gate 3b GREEN on head `aa4543ba4` (21 checks, 0 pending, 0 not-green, 4/4 required, `mergeStateStatus=CLEAN`).**
   **Iteration 16 built both milestones and then died before landing them** — killed 09:00:33 CEST by the
   driver's STALL watchdog (`idle with a descendant alive ≥2400s`, `rc=143`), leaving zero charter rows and
   zero log entries, so this row read `[NEXT]` for work that was finished and self-reviewed in an open PR.
   Found by Gate 2's died-mid-flight traces (open PR from this account · two iteration-16 worktrees ·
   `.snap/` residue), and **verified rather than adopted**: the three smoke refusal branches each red under
   their own named arm, ON-always-leads reds both the shell sequence assertion and — the load-bearing one —
   `TestFmtDriverScheduleSatisfiesOrderIntegrity` at `CheckFmtOrderIntegrity … order_integrity_lead_not_alternating`,
   which is AC-M3-4's whole claim. Restores byte-identical by sha256.
   *(historical: M1 + M4 LANDED · M2, M3, M5 were the RESUME POINT, not dropped.)*
   **M4 (D2) LANDED 2026-08-20 · iteration 15 · PR [#806](https://github.com/sunholo-data/ailang/pull/806) → `d5bcfa0c8` · Gate 3b GREEN on head `922190dd3` (21 checks, 0 pending, 0 not-green, 4/4 required, `mergeStateStatus=CLEAN`) · evaluator FAIL 80/100 with ONE BLOCKING finding — against the CONTROLLER'S OWN repair — fixed, re-verified and landed.**
   `ailang eval-censored-pairs <on-dir> <off-dir>` is a **sibling** of `eval-paired`, never an
   extension: `eval_paired.go` is byte-identical to base (control: a file we DID change differs),
   because its stdout JSON has two live callers (`nightly-eval.sh:339` microRAG, `:515` fmt — the
   plan said `:512`, a +3 drift measured at pick time). It implements §2 censored pairing, a
   one-sided exact sign test on the existing binomial helpers, the §5/§5.3 VOID rules and the §7
   decision rule. **Specification gap resolved at pick time:** the plan lists `n_eff < 24` among
   the §7 thresholds without saying what verdict it yields, and the doc's RETIRE-at-low-`n_eff`
   is conditioned on *"after two slots plus the authorized third"* — state a directory analyzer
   cannot observe. So `n_eff < 24` → INCONCLUSIVE/`insufficient-neff`; the slot-budget call stays
   with the driver and the human.
   **THE FINDING, and it is this repo's named recurring shape for the SECOND consecutive
   iteration: the guard was written and its CALL SITE loaded past it.** The command loaded arms
   via `LoadArmForPairing`, which wraps `FilterValidResults` and drops invalid rows **at load
   time** — and the `>20% of ON rows quarantined → VOID` gate counts exactly those rows, so
   through the CLI it could never fire. The analyzer's unit tests passed regardless, because they
   construct rows directly and bypass the loader. Measured, both arms, control firing: filtering
   loader **5** ON rows / **0** quarantined visible; raw loader **6** / **1**. Sharper: **AC-M4-5's
   verdict was right for the wrong reason** — the AC is cited as *"a real dataset whose correct
   verdict is known in advance"*, and the delivered code returned VOID
   `order_integrity_unpaired_block`, an artifact of the **odd row count** the silent drop left,
   where the data's actual defect is whole-arm blocking → `order_integrity_nonadjacent_arms`,
   which is what it now returns.
   **AND THE FIRST REPAIR TRADED ONE DEFECT FOR ANOTHER — the evaluator caught it, aimed at the
   controller on purpose.** `AnalyzeCensoredPairs` filtered only ON; that was invisible while the
   filtering loader dropped invalid rows from **both** arms, so switching to the raw loader let an
   OFF row invalid for any non-contamination reason (`harness_error`, `config_mismatch`) reach
   `PairArms` and the win tally. Reproduced first-party before acting: all-valid arms give
   `off_wins=1 both_pass=2`, and marking that same decisive row `Validity{Valid:false}` gives
   **identical** numbers — a non-measurement deciding a §7 verdict. Fixed with
   `partitionMeasurements` on BOTH arms before any statistic, the gates keeping the raw slices
   deliberately (executed order is a fact about the run; the quarantine RATE is defined over the
   banked set), and `off_quarantined`/`off_rows` now reported.
   **Three non-blocking findings closed:** `order_integrity_repeated_benchmark` is not merely
   untested but **unreachable** (blocks dedupe by (benchmark, arm), so a third block is always a
   duplicate key caught by `noncontiguous_block`, and a half-pair is caught by
   `nonadjacent_arms`) — DECLARED as such in code and pinned by a test that fails loudly if it
   becomes reachable, per the rule that an undeclared unreachable branch is a guard nobody
   protects; the `>20%` boundary is pinned on both sides (1 of 5 = exactly 20% does **not** void,
   1 of 4 does); and `TestCensoredVerdictMatrix` **passed with `[]` as its fixture** — zero
   subtests, one green line — now guarded.
   **Gate 3b caught a windows-only defect in a controller-written test** (base arm green on
   windows, so the red was ours): the fixture filename embedded an RFC3339 timestamp and `:` is
   illegal in a Windows filename, so the test died in its own setup.
   *(historical: **M1 (D1) LANDED 2026-08-20 · iteration 14 · PR [#794](https://github.com/sunholo-data/ailang/pull/794) → `bc0b5a8d4` · Gate 3b GREEN on the PR head `fb1ef41b5` (21 checks, 0 not-green, 4/4 required, `mergeStateStatus=CLEAN`) · evaluator PASS 84/100, zero blocking**)* —
   plan: [m-motoko-fmt-remeasurement-instrument-sprint-plan.md](planned/m-motoko-fmt-remeasurement-instrument-sprint-plan.md),
   sprint JSON records M1 and M4 `completed`/`passes=true`, M2/M3/M5 `not_started`.
   **What M1 changed and why it was not the one-liner §6 priced.** The planner refuted the design
   doc's own §12.2 preference: option (1) is unsound because `ExecutorFactory.GetExecutor` caches by
   executor NAME, so one `*MotokoExecutor` serves all **17** motoko lanes (7 ollama + 10 requiring a
   credential), and `HealthCheck` is **`sync.Once`-cached**, which would freeze a model-dependent
   verdict at whichever model canaried first — silent and order-dependent, strictly worse than the
   loud refusal it replaced. So the refusal moved to `ExecuteStreaming`, the per-task choke point,
   keyed on `e.getModel(task)` and expressed on the **resolved provider**
   (`ai.EnvVarForProvider(ai.GuessProvider(model))`), never on a literal env-var name — and strictly
   downstream of repo discovery, so §12.3 holds for free.
   **The finding worth carrying: the guard was pinned and its WIRING was not.** Neutering the call
   site left the entire package green as first delivered, because every arm called the helper
   directly and the one `Execute`-level arm drove an `ollama/…` task the guard ADMITS. `T-CALLSITE`
   repairs the row and is now MUT-4's sole killer. Honest full-package blast radii: neuter-ADMIT
   **12**, neuter-REFUSAL **3**, neuter-unresolvable-guard **1**, neuter-call-site **1**.
   **Remaining after iteration 17 — M2 ONLY:** **M2** `AC-D1-live` — one fmt-lane run reaching
   `localhost:11434` with ZERO `openrouter.ai` connections, asserted on the CONNECTION and paired
   with an OpenRouter-lane known-positive control (**needs the rig**; its circularity is broken,
   since the preflight no longer refuses the fmt lane) is the **only** milestone left, and it is the one
   that cannot run without `rig.lock`. **M3** (D1b counterbalancing) and **M5** (smoke-bank wiring)
   LANDED 2026-08-21 in `b2733201a`. M4 was taken ahead of M2/M3 deliberately: the plan declares it *parallel-safe
   with M1–M3*, it is the only remaining milestone with no rig dependency, and M3-first would have
   left a milestone with an unclosable acceptance criterion.
   **Deployment precondition still owed (doc §6, open issue #558):** merging to `origin/dev` does
   NOT put D1 on the rig — the installed plist executes `nightly-eval.sh` in place from V1's
   checkout, so the Wednesday lane does not benefit until that checkout advances. Verify by reading
   the script at the exact path in the installed plist's `ProgramArguments`, never at a working-tree
   path.
   *(historical: UN-PARKED + TRACE DONE 2026-08-19 · iteration 13 — `D-MOTOKO-FMT-1` RESOLVED **precondition**
   by Mark (attended 2026-08-19) and the trace it demanded is RUN; **O4 CLOSED by measurement** in the
   doc's new §12. Answer: *both halves of the reviewer's fear are true, of different lanes* — for the
   two fmt arms no OpenRouter routing is reachable (every firing tier resolves `ollama/…`, measured
   `GuessProvider`→`ollama`/env-var `""`, both profiles also pinning `openai_base_url=localhost:11434/v1`,
   and motoko's own `required_secret_for_model` returns `""` for `ollama/`), so the preflight is a FALSE
   fail-fast there; while for the OpenRouter lanes it is the ONLY hard stop, because motoko's own check
   is a **warning** that proceeds (`supervisor.ail:11-19,42-51`). Hence a CONDITION, never a deletion.
   **And the trace found the blocker that re-shapes D1:** the condition is *not expressible where the
   check sits* — `HealthCheck(ctx)` takes no task (`executor.go:31`) and `cfg.MotokoModel` is never set
   from models.yml (**3** non-test hits, only the hardcoded `factory.go:71` default; control
   `MotokoProfile` **11**), so an `if` at `healthcheck.go:64` would read `openrouter/anthropic/claude-haiku-4-5`
   for every lane and refuse the ollama arms exactly as today. D1 is therefore a plumbing change, not a
   one-liner — three costed options in §12.2, and the live arm moved to acceptance criterion `AC-D1-live`
   rather than claimed. **Next: this is now a normal sprint** (planner → executor), gated only by its own
   stated dependencies (migration V6/V7) for the *rig slots*, not for D1/D1b. Historical tag: DESIGNED +
   PARKED `needs-human-review` 2026-08-17 · iteration 8 · `0e1edd80c`; no reviewer disputed the
   instrument's direction in either quorum round)*]
   **fmt re-measurement instrument** · clause 3 · ~~design HOW we re-prove the −74% tokens-to-pass
   result on the new tree~~. Decides whether `motoko_ext_fmt` survives. **DST will NOT do this** —
   `fmt` is an effectful extension hook, precisely the class measured at 1-of-~40 coverage (see DST
   scope). Design a real instrument and price it honestly rather than assuming it is cheap.
   **THE ITEM'S OWN PREMISE DID NOT SURVIVE GATE 2, and that is the finding.** The −74% is real and
   correctly attributed, but: its author rates it *"direction, not proof"* (n=1/pair, p≈0.11);
   **74.7% of the saving is ONE benchmark** (all six pairs −74.2%, drop `log_file_analyzer` →
   **−47.1%**, that pair alone 3,125,933 of 4,182,882 saved tokens); that benchmark is **3/30
   lifetime, 0/10 last five nights** in the `rag_on` opencode rotation lane (open **#649**), so
   tokens-to-pass is undefined there; **one of the six pairs was quarantined by the harness itself**
   (`ab2_fmt_on/emit_exact_bytes_varied`: zero fmt-hook events, `validity=treatment_unproven`) and
   summed into the headline anyway, while the other 5 ON rows carry one `status=formatted` event
   each and all 6 OFF rows are clean; and the run was **order-confounded** — all six ON rows
   completed before the first OFF row started (16:42:37→17:00:10 then 17:01:28→17:36:49).
   **On the four pairs with proven treatment AND a passable benchmark the result is −5.7%**
   (ON cheaper 3/4), not −74.2%. So the doc replaces "re-prove −74%" with a paired **censored
   win-rate** defined when arms fail, a **counterbalanced** schedule with an order-integrity gate,
   ELO-banded selection, ~9.8 rig-hours priced off a measured 4.91 min/row anchor, and a
   pre-registered KEEP/RETIRE rule. **Confirmed live defect, blocking any run:** the Wednesday fmt
   A/B lane has banked nothing since AC5 — 08-05 and 08-12 both died at
   `internal/executor/motoko/healthcheck.go:64`, an **unconditional** `OPENROUTER_API_KEY` refusal
   with no lane condition, whose error text *"motoko routes ALL models via OpenRouter"* is false for
   both fmt arms (`provider: "ollama"`, `env_var: ""`, models.yml:1854/:1880).
   **Next**: answer D-MOTOKO-FMT-1, then this becomes a normal sprint · ≤3 days
6c. [**LANDED 2026-08-23 · iteration 19 · PR [#831](https://github.com/sunholo-data/ailang/pull/831) → [`c1950750c`](https://github.com/sunholo-data/ailang/commit/c1950750c) · evaluator PASS 93/100, ZERO blocking · Gate 3b GREEN on head `7de24f03c` (21 checks, 0 pending, 0 not-green, 4/4 required, `CLEAN`).**
   **The row understated itself twice.** (i) The self-test was not under-covering, it ran **NOWHERE**:
   `grep -rl test_motoko_connection_probe make/ .github/workflows/` = **0** files against a same-scope
   firing control (`test_fmt_ab_schedule` = 1, `make/test.mk:43`) and a repo-wide **0**. Now wired into
   `test-launchd-drivers` (`ci.yml:507`) under an explicit `/bin/bash` (the rig's is 3.2.57).
   (ii) The gap was not only coverage — it hid a **soundness defect that could certify falsely in both
   directions**. Two arms each: `instrument_failure` called inside a command substitution exits only the
   SUBSHELL (repro rc=**0**, control outside `$( )` rc=**1**), which is exactly `descendant_pids`'s
   deadline shape, leaving `pids=""`; and `lsof … -a -p ""` returns **75** lines rc=0 with empty stderr,
   the same count as the fully **unscoped** query, against a control with a real TCP-less pid at **0**
   lines rc=1. **An empty scope argument does not narrow a query, it removes the scope.** So the probe
   could have sampled every connection on the machine and then passed on another process's loopback
   peer or failed on another process's OpenRouter one — silently, behind `2>/dev/null || true`.
   Fixed, and the audit for the same shape re-derived at **0** remaining (the evaluator attacked that
   claim twice, including re-introducing the pre-fix shape verbatim, and could not break it).
   **Driver logs are now retained past the `trap`, on the REFUSING path too** — row 6's named next step.
   **34 arms** (from 8), live path driven hermetically via a stub `AILANG_BIN`; each new arm
   mutation-proven. Includes a **refusal-branch drift gate**: adding a 24th `instrument_failure` used to
   leave the suite byte-identical at `PASS: 32`, and now reds — *a removal proves a check FIRES, only an
   addition proves it LOOKS*. **STILL OPEN, and this row does not claim it: V38 is NOT isolated.** The
   two defects fixed here are live candidates, not a demonstrated mechanism; isolation needs the rig, and
   now has the logs it needs. Resume point stays **M2** on row 6] **The connection
   probe's self-test covers its four `assert_*` front doors and `classify_lsof`, and NOTHING in its
   live path** · loop health · ~**15** of ~19 refusal branches in
   `tools/eval/motoko_connection_probe.sh` are never reached by
   `tools/eval/test_motoko_connection_probe.sh`, because every self-test invocation goes through a
   `--`-flag entry point and none exercises the live run. Demonstrated by the evaluator rather than
   argued: neutering the **darwin/arm64 platform gate** (`if false && [[ … ]]`, mutant asserted LANDED
   by sha256 and VALID by `bash -n`) left the suite reporting a **byte-identical `PASS: 7`**, rc=0.
   The uncovered set includes `descendant_pids`'s deadline, `run_lane`'s timeout/kill path, the
   empty-`dig` refusal and the JSON-write failure — i.e. **exactly the territory the unisolated V38
   defect lives in**, which is why this is one row with that isolation and not a separate cleanup.
   Do them together: keep the driver logs (the `trap` deletes them), isolate V38, and add an arm per
   live-path branch. A guard is not a gate until something reds when you remove it · 1 iteration
6b. [**CLOSED 2026-08-18 · iteration 10 — motoko owns ZERO of the 15, measured; nothing to hand**]
   **Triage-lite the 15 charter-orphaned open issues** · loop health · from the **weekly
   external-issue sweep** run 2026-08-17 (iteration 8; first fire past the Monday-07:00 local
   boundary). **15 orphans of 75 enumerated open issues** — zero mentions across all eight
   charter/log/status-archive/dashboard files for both missions, printed as a per-issue table with
   firing controls (`#663`→motoko charter 4, `#617`→v1 charter 2, `#663`→`mission-dashboard.md` 1)
   and the list length asserted against `gh issue list … | wc -l` = 75:
   **#727 #708 #696 #694 #689 #688 #687 #679 #676 #672 #671 #662 #656 #646 #644**.
   Note the shape rather than the count: most are AILANG-lane (`ailang-parse`, `cli`, `email-parse`,
   `housemove2026`), i.e. **V1's** territory, and only `#708`/`#696` are mission-infra. Scope:
   ghost-discipline each repro at HEAD → verdict comment → queue-or-close, and hand the AILANG-lane
   ones to V1 via the cross-mission channel rather than adopting them here. A sweep never outranks
   an existing pick — this row sits at normal ordering · 1 iteration
   **VERDICT 2026-08-18 (iteration 10), re-derived at HEAD `c0dde65eb` rather than inherited:
   motoko owns 0 of 15, and the hand-off this row prescribes was already satisfied before it was
   written.** (a) **3 have CLOSED** since the sweep — `#727` and `#708` on 08-18, `#696` on 08-17,
   all by V1's own lane. (b) **11 of the remaining 12 are AILANG-lane by their own labels**, not by
   inference: every one carries `ailang-message` + `from:<consumer>` (`ailang-parse` — `#689` `#688`
   `#662` `#656`; `email-parse` — `#679` `#676`; `housemove2026` — `#672` `#671`; `cli` — `#694`
   `#646` `#644`), i.e. they arrive from downstream AILANG consumers through the importer, and **all
   eleven are already enumerated in V1's own charter** at `design_docs/v1-mission.md:2104-2109`.
   (c) The 12th, `#687` (`⚠ Binary may be stale` is an mtime heuristic over CWD-relative dirs), is
   mission-infra and is **V1's declared next pick** — named as such in its iteration 220 *and* 221
   reports (*"`#687` closes the mission-infra sweep lane"*). So there is no cross-mission message to
   send: the recipient charter already lists all twelve. Nothing was closed or commented by this
   mission, deliberately — a verdict comment from the non-owning loop is noise on someone else's lane.
   **Instrument note, and the reason this row's own re-run would have lied:** the per-issue table
   re-taken at HEAD shows all 15 at **≥1** mention, because *this row lists all fifteen numbers in
   the motoko charter* — every orphan is now self-mentioning and the weekly sweep can never
   re-detect them. That is the sweep WORKING (they are tracked, by this row) and not a false
   negative — but it means the row closing is what makes them invisible again, so the disposition
   had to be a measurement of who owns them, not a bare close. Table scope: 8 charter/log/archive/
   dashboard files across both missions; controls fired (`#663`→motoko charter **6**,
   `#617`→v1 charter **2**, `#770`→v1 dashboard **1**), and `${#FILES[@]}`=8 asserted against the
   8 columns printed (rule 3a(i-d) + the zsh 1-indexed-array rule)
6d. [**LANDED 2026-08-23 · iteration 20 · PR [#838](https://github.com/sunholo-data/ailang/pull/838) → [`50460040c`](https://github.com/sunholo-data/ailang/commit/50460040c) · evaluator PASS 90/100, ZERO blocking · Gate 3b GREEN on head `15b89569d` (21 checks, 0 pending, 0 not-green, 4/4 required, `CLEAN`).**
   **The row asked for a decision and the measurement found a MECHANISM, which is the durable half.**
   The Repo Profile now names the **executing root** (`~/.ailang-driver-pin/motoko`, the pin worktree)
   separately from the **source clone**, which is marked DO-NOT-RUN-HERE — the charter no longer sends a
   future session into a stale rulebook. **Why the clone reached 170 unnoticed is not an oversight but a
   documented assumption that has gone false:** `mission-control.sh` posts to the human channel only when
   the pin FAILS, and the comment above that emit block says the shared clone being behind *"is not itself
   reportable — once drivers pin, that drift is harmless"*. Harmless to the DRIVER, yes; not to a HUMAN
   SESSION. The growth was in the driver log and nowhere else — **119 → 132 → 144 → 159 → 170** over five
   fires, on the SUCCESS path. Fixed by keeping the true half of that comment: notify on crossing
   `AILANG_DRIVER_DRIFT_WARN` (25), thereafter only on a **doubling**, state re-armed below threshold.
   The notice names `AILANG_DRIVER_SRC`; a `$REPO` body — which is what the executor first wrote — names
   the pin worktree, whose drift is **0 by construction**. Suite 17 → 27 arms, six mutants with red sets
   produced by running them. **THE RESIDUE IS SUPERSEDED, MEASURED NOT INHERITED:** of 129 added lines in
   the clone's uncommitted delta, **125 are byte-present on `origin/dev`** and the 4 that are not are prose
   reflows of a decision-ledger block `origin/dev` carries in a superseding form (negative control fired).
   **STILL OPEN, and this row does not claim it:** the reconcile itself is `D-MOTOKO-WORKDIR-1`, parked for
   Mark — Gate 1's obligation 2 fails by construction and `pin-root.sh`'s own header says that first
   reconcile is human. Nothing is blocked meanwhile] **The Repo Profile's
   declared `MISSION_WORKDIR` holds a rulebook 160 commits stale, and the charter tells a future
   session to run there** · loop health · `~/dev/sunholo-data/ailang-motoko` — the checkout this
   charter names as the working checkout — is **160 commits behind `origin/dev`** with **7**
   uncommitted files dated 2026-08-15, including a modified
   `.claude/skills/mission-control/SKILL.md` (38 lines changed). The charter's own Skill-sync note
   says that copy *"takes precedence for sessions run here"*. It does **not** execute today — this
   loop runs from the `#558` pin root, and pin / `~/.claude`'s resolved target / origin are
   byte-identical three ways (`cmp` rc=0) — and the residue is **superseded, not unfinished**: both
   untracked files now exist on `origin/dev`, `scripts/mission_decisions.sh` byte-identically. So it
   is not work to adopt. It is a **loaded gun at a documented path**: any session started in
   `MISSION_WORKDIR` executes a rulebook missing every rule added since 2026-08-15. Scope: decide
   whether the pin root is now the canonical checkout and make the Repo Profile say so, or bring the
   workdir current — and either way remove the stale skill copy, since a diverged rulebook that
   nothing executes is indistinguishable from one that does until someone runs there · 1 iteration

6e. [**LANDED 2026-08-25 · iteration 22 · PR [#871](https://github.com/sunholo-data/ailang/pull/871) → [`086b72184`](https://github.com/sunholo-data/ailang/commit/086b72184) · evaluator round 1 **FAIL 54/100** (3 blocking) → round 2 **PASS 91/100, ZERO blocking** · Gate 3b GREEN on `ddd8f3f09` (21 checks, 0 pending, 0 not-green, 4/4 required, `CLEAN`).**
   **The count was low and the mechanism is still not isolated — both stated rather than smoothed.** Three cancellations, not two (the third, `32673098414`, is a push to `dev` itself), all at ~918s = `timeout-minutes: 15` firing. Arm-33 attribution verified from the three logs against a green control. It has NOT recurred in the ~44 runs since, on an unchanged file, so what shipped is the structural fix rather than a mechanism claim: every arm now carries a hard cap (`ARM_CAP_SECS`, TERM→KILL, named `not ok` + both captured tails), and `descendant_pids` is bounded by NODE COUNT as well as by the clock with a distinct message. Suite 34 → 39 arms; drift gate 23 → 24. Executor CAPPED at 30 min (FLAGGED) so the whole mutation drill is first-party.
   *(historical tag kept as the trail: NEXT — found 2026-08-23 by iteration 20's Gate 3b, **CONFIRMED the same hour by a second instance on a
   MARKDOWN-ONLY diff**] **`test_motoko_connection_probe.sh` arm 33 hangs CI for ~15m and is then cancelled;
   twice in 40 minutes, and the second time nothing executable changed**
   · loop health · The `launchd drivers (bash 3.2)` job on PR `#838` was **cancelled after 15m18s**
   against a **~68s** success on dev's own HEAD and on 18 of the last 20 CI runs; the log stops after
   `ok 32` and never emits arm 33, *descendant discovery refuses on the real wall-clock deadline*, and the
   runner then reported `Terminate orphan process: pid (bash)` / `(make)`. Attribution was NOT taken from
   the co-occurrence: the diff touches no file the suite reads (`changelogs/v0.18-current.md`,
   `tools/launchd/mission-control.sh`, `tools/launchd/test_driver_notify.sh`), and rule 3d's strongest
   control — a **re-run on a byte-identical tree** — came back **success in 88s**, so the variable is the
   environment, not the code. The codex executor independently stalled on the same suite in its sandbox
   and said so. **INSTANCE 2, 40 minutes later, settles it:** the record PR
   [#840](https://github.com/sunholo-data/ailang/pull/840) — whose five changed files are **all markdown**
   (one `SKILL.md` and four `design_docs/*.md`, i.e. **zero** executable lines) — reproduced it exactly:
   `17:38:10Z → 17:53:28Z`, **15m17s**, cancelled, log stopping after the identical `ok 32`. A markdown-only
   diff cannot break a shell suite, so code attribution is **refuted**, not merely doubted. Three
   observations in one iteration (two CI, one sandbox) against **~68s** on dev, for an arm wired into CI
   only yesterday (iteration 19). This is a confirmed defect, not a watch-item. The
   shape is rule 3m's: a wall-clock/process-tree bound calibrated on the author's machine, in a suite whose
   own arm 33 already prints `UNINFORMATIVE UNDER SANDBOX` locally. Scope: derive arm 33's bound from the
   stimulus it measures in-test rather than from a constant, or give the arm its own hard cap so a hang is
   a loud instrument failure instead of a 15-minute cancellation. **A guard that can hang is not a gate —
   it is an outage with a green history** · 1 iteration)*

6f. [NEXT — batched from the weekly external-issue sweep, 2026-08-24 iteration 21]
   **Triage-lite the 8 open issues no mission doc mentions** · loop health · The Monday sweep
   enumerated **76** open issues (count asserted against `gh issue list … | wc` = 76) and greped
   `-cE "#<n>\b"` across all **9** mission docs — motoko's four AND V1's four AND the fleet-shared
   one — because this repo's issue queue is shared and a sweep scoped to one mission's docs reports
   its neighbours' work as orphaned. Against motoko's four alone the count is **67**; against all
   nine it is **8**. Positive control `#558` = 57 hits; negative control fired (literal not
   published, per Gate 4's *a control you record is a control you spend*).
   **Ours (2):** [#842](https://github.com/sunholo-data/ailang/issues/842) — *provider failure
   masked as successful empty completion (`finish_reason=stop`, no usage block)*, filed by the
   `motoko-agent` account. This is squarely the mission's own class — a harness false-green — and
   sits directly under the charter guardrail *"never conclude model wall; every motoko
   disengagement investigated so far has been a harness bug."* An empty completion that reports
   success is exactly how a harness bug gets recorded as a model verdict.
   [#839](https://github.com/sunholo-data/ailang/issues/839) — `std/net` ignores
   `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` while the AI provider path honours them.
   **Not ours (6):** `#847` (nightly-eval sustained failure, `explicit_dataflow_ssa`), `#800`,
   `#754`, `#753`, `#752`, `#751` — eval/registry/Z3/email-parse territory the anchor's owning
   mission has the domain knowledge for. Handed to V1 on the cross-mission channel; **not triaged
   here, and this row does not claim they were.** All 8 are internally authored (no
   `github-untrusted:` class), so none is external-origin feedback.
   A sweep never outranks an existing pick — this row is positioned by normal ordering · 1 iteration

6g. [NEXT after 6f — non-blocking finding from iteration 22's round-2 evaluator, routed to the queue rather
   than into a growing PR] **`run_bounded` and `run_lane` cap the wrapper PID, not the process GROUP — a hung
   grandchild is reparented to `PPID 1` and survives** · loop health · The judge reproduced it against the
   unmutated shipped `run_bounded`: the `sleep 30` fixture's wrapper is killed and the suite's own
   "process survived" check therefore passes, while `ps -axo pid,ppid,command` shows a live `sleep 30` at
   `PPID 1`. Both `sleep`-fixture arms leak one self-expiring orphan per run. **This is a defect iteration 22
   did not introduce**: the single-PID `kill` predates M1 and the identical shape is in the PRODUCTION
   `run_lane`, which is the half that matters — the probe's lane bound can return while the driver's
   descendants keep running, and the mission's own guardrail is that a harness bug must never be recorded as
   a model verdict. So it is a queue row on its own evidence, per the rule that a pre-existing defect a
   reviewer surfaces is a row and not a revision. Scope: kill the process group (`setsid` + `kill -TERM -$pid`)
   or record and kill descendants, in BOTH call sites, and give the arms an observable that a wrapper-only
   kill cannot satisfy — the current one passes for exactly the reason it should fail · 1 iteration

7. [NEXT after 6e, 6f and 6g] **Profile restoration design** · clause 4 · 5 profiles, 14 of 18 model entries · 1 iteration
8. **Repin the stale OpenRouter motoko models** · clause 4 · measured live 2026-08-12: our
   `motoko-or-kimi-k2-6` pins `moonshotai/kimi-k2.6` ($0.95/$4.00 per M), but
   **`moonshotai/kimi-k2.7-code` dominates it on every axis** — newer, code-specialised, and cheaper
   ($0.70/$3.50), same 262k context. `moonshotai/kimi-k3` also now exists (**1M context**, $3/$15 —
   4.3× k2.7-code both ways, so a targeted large-context instrument for the `docx_reimplement` class,
   NOT a default). DeepSeek needs no change: our `deepseek-v4-flash-0731` pin at $0.08/$0.18 with 1M
   context is already the cheapest concrete option (`~…-flash-latest` is cheaper still but is a
   FLOATING alias, which would undo the `:floor` prompt-cache pinning we do deliberately).
   **Not a side edit** — a model repin moves the eval baseline, so it needs a deliberate before/after
   and a banked comparison, per the extension-fix baseline lesson · 1 iteration
9. [PARKED — needs a green tree] **R3 — cross-model generality study** · clause 5 + the north star's
   weak-model thesis · do motoko's gains hold with strong models, and are they AILANG-specific or
   general? This is the TEST of the mission's central claim and it has never been run. Carried
   forward from the archived charter. Split to measure: best-of-N is language-general (portable
   edge), contracts + Z3 are AILANG-specific (the moat)
10. [PARKED — Phase-0 gated] **Extension port to ABI 5.0** · clauses 1+2 · 12 packages, pilot on
   `test-dummy`, `compaction-ai` last
11. [PARKED — Phase-0 gated] **Registry-vs-vendored reconciliation with Arni** · clause 2 · his
   `compaction_ai` "0.3.0" is 33,851 B; our published `0.3.2` is 9,454 B — same name, lower version,
   different code
12. [PARKED — Phase-0 gated] **Re-prove and re-baseline** · clauses 3+4 · migration Phase 3
13. [PARKED — needs a green tree first] **Motoko executor-lane graduation, design** · clause 6 ·
   the `motoko:<model>` spawn recipe, bounded probe, fallback-chain placement, and the false-green
   guards. Design work can start once clause 1 holds (a motoko we can rebuild); the *trial* needs a
   real sprint. **Target World (`ailang-code` profile) first — not this Go repo.**
14. [PARKED — after the executor-lane design item] **Motoko executor-lane gate trial** · clause 6 · real sprints, plan-faithful
    landing of held-out tests. The DeepSeek-Flash precedent is the bar to clear: 3/3 real-sprint
    failures behind a clean `rc=0`, so a passing smoke proves nothing on its own.

---
**Document created**: 2026-08-12 (rewritten from the 2026-06-24 charter, archived in full at
[motoko-mission-status-archive.md](motoko-mission-status-archive.md)). Iteration 0 ratifies it via
the quorum with Mark before any sprint routes.
