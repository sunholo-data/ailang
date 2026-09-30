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
**Scheduling**: launchd `dev.ailang.mission-motoko`, driven by the **shell driver**
`tools/launchd/mission-control.sh` like every other live mission (Mark, 2026-09-30 — the restart does
NOT wait for the binary mission-control path). The repo copy of the plist
(`tools/launchd/dev.ailang.mission-motoko.plist`) sets `StartInterval=46800` (**13h**, deliberately
non-harmonic with the other loops; the installed copy is the truth — measure it, do not trust this
sentence). **PAUSED since the 2026-09-08 fleet pause and still paused after the 2026-09-30 reset**: the
kill switch `~/.ailang/state/mission-motoko.disabled` is present and only Mark removes it
(`D-MOTOKO-RESTART-1`).
**Log**: [motoko-mission-log.md](motoko-mission-log.md) — append-only, one entry per iteration; the
fork-era entries 0–39 are in [motoko-mission-log-archive.md](motoko-mission-log-archive.md) and every
iteration has a one-line row in [motoko-mission-index.md](motoko-mission-index.md).
**Human-facing reporting**: the weekly bookkeeping issue (first was [#663](https://github.com/sunholo-data/ailang/issues/663);
the live number is in `~/.ailang/state/mission-motoko-gh-issue`) — every iteration posts its report
there as a comment; driver crashes post there too.

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

**The motoko tree this mission improves (reset 2026-09-30 — [MOTOKO.md](../MOTOKO.md) is the map; read
it before touching any `~/dev/mk-*` directory).**

- **Work surface**: `~/dev/mk-main`, a worktree of the one motoko clone, on branch
  `sunholo/main-dst` of `sunholo-voight-kampff/motoko_agent` = Arni's upstream `main` (DST core,
  extension **ABI 8.0**) plus our carried commits: the `cloud` and `ollama_microrag` profiles,
  `motoko_ext_ailang_tools` (= upstream PR [arniwesth/motoko_agent#200](https://github.com/arniwesth/motoko_agent/pull/200)),
  rig-lease forwarding, a relock, and the dropped dead `enable_thinking` option. `~/go/bin/motoko`
  is a shim into `mk-main/scripts/run-agent.sh`, so this is also what every eval runs.
  `git log origin/main..sunholo/main-dst` in that worktree is the carried delta.
- **Extensions are in-repo path packages** under `mk-main/packages/`
  (`{ path = "packages/motoko-ext-*" }`). The 14 ABI 2.2 registry packages `sunholo/motoko_ext_*`
  were **unpublished 2026-09-28**
  (`ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`); do not re-publish or re-pin them.
- **Retired**: the ABI 2.2 fork (`mk-ast` worktree, branch `sunholo/eval-canonical`). Its worktrees
  were removed 2026-09-28; the branch survives on our fork as history only. Any archived row,
  decision or design doc that names `mk-ast`, `eval-canonical`, ABI 2.2/5.0 or a "Phase-0 gate" is
  describing that tree, not this one.
- **Gates for the motoko tree** (clause 1): `make check_core && make verify_extensions` inside
  `mk-main`; `make dst` needs GNU make 4 (`gmake`). The anchor repo's own gates (below) are unchanged.
- **Cloud executor image**: still carries the old fork until
  [sunholo-data/ailang#1413](https://github.com/sunholo-data/ailang/pull/1413) (pins motoko main
  `4d4917cd`) lands AND ships — it reaches test/prod only via a release + promote, never by merge alone.

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
| D-MOTOKO-WORKDIR-2 | RESOLVED | **Yes — standing authorization granted.** Reconcile the source clone unattended only when all three recorded safety predicates hold; any failed predicate still requires a new human decision. | Mark answered `MOTOKO-WORKDIR-2: Yes` on bookkeeping issue `#850` at `2026-08-29T09:09:20Z`. Iteration 28 re-measured **0 ahead / 0 dirty / 292 behind**, so no backup payload existed; `git checkout -B dev origin/dev` advanced `e3ed9467f` → `bd0bb157d` and post-verified **0 dirty**. |
| D-MOTOKO-6N-1 | RESOLVED | **Ship the measured minimal fix for the discovery arm, or hold for a race-free construction?** The arm at `test_motoko_connection_probe.sh:449` cannot fail for the reason it names — measured: neuter the wall clock and the suite stays 41/41 green. The minimal fix (three distinct refusal messages + the arm asserting the wall-clock one) is measured to work: clean 41/41 in 50s, and with the wall-clock mutant rc=1 in 44s on the exact arm. The design quorum BLOCKED it 3/3 in both rounds on ONE surface: the fix converts a silent false PASS into a possible loud false RED if the node ceiling wins the race on the CI host. Local margin, corrected after the evaluator caught me benchmarking the wrong binary: **~6-9x**, not the ~52x I first reported. **(A) SHIP IT** — the arm becomes honest now; risk is a flake on the CI host at an unmeasured rate, visible immediately as a red on the next PR. **(B) HOLD for D4** — the evaluator's synthesis: scope `PROBE_MAX_TREE_NODES` to that one arm's `env` line, measured free on the happy path (41/41, 47.1s), which removes the race structurally rather than betting on it; costs one more iteration and a third quorum round. **(C) NEITHER** — leave the arm vacuous and de-prioritise. **Loop's recommendation: (B)** — the reviewers were right twice for the same reason, and D4 is the thing all three were circling; it is one iteration and it ends the argument instead of deferring it to a CI leg. **Default if unanswered by 2026-09-08: (B)**, taken as a normal queue pick under row 6p.  **ANSWERED — (B) HOLD for D4. Scope PROBE_MAX_TREE_NODES to that one arm's env line and remove the race structurally rather than betting on a 6-9x margin holding on a CI host nobody has measured. The reviewers blocked twice for the same reason and D4 is what all three were circling; one more iteration and a third quorum round is the right price for ending the argument instead of deferring it to a red CI leg. Take it as a normal queue pick under row 6p, as the loop proposed.** (Mark Edmondson, attended 2026-09-01, recorded directly in this ledger.)| Design doc `design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md` (PARKED). Quorum artifacts `m-motoko-discovery-arm-discriminating-refusal-2026-09-01T00-35-20Z.json` and `-2026-09-01T00-43-50Z.json`, both BLOCKED, 3/3 external present, no absentees. Evaluator PASS 78/100, 1 blocking (the wrong-binary benchmark, corrected in-doc). Filed by motoko iteration 31.  **Attended ruling 2026-09-01** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author — an attended identity, which the fleet bot does not hold and the loop may not author with. Attended ruling, matching the loop recommendation and the evaluator's D4 synthesis (measured free on the happy path: 41/41, 47.1s).|
| D-MOTOKO-CARVEOUT-1 | RESOLVED | **Was Gate 2's narrow-refinement carve-out available at round 2 of `m-motoko-suite-arm-count-floor`, or should the doc have parked?** The controller judged yes and shipped; the independent evaluator judged no and FAILED the iteration 68/100 on exactly this. Both readings are defensible from the rule's text, and only you can settle which one the loop follows. **(A) UPHOLD** — a reviewer fix may be applied in PART when the unapplied part is recorded as an explicit OPEN residual (which it was, in astra's own words), and a controller MAY substitute an equivalent measurement for a reviewer's named remedy when it is honestly labelled (the PATH-shadowed Linux simulation). Then row 6s lands as-is next iteration. **(B) OVERRULE** — verbatim means verbatim: a partially-applied fix or a substituted remedy is a park, full stop. Then row 6s needs a third quorum round after the residual audit is actually run, and this iteration's routing was wrong. **Loop's recommendation: (B).** The judge read the rule more strictly than I did and the rule's own words are on its side — *"their own text — never a controller-invented resolution"* — and the cost of being wrong is asymmetric: (A) quietly widens a gate that exists to stop exactly this. **Default if unanswered by 2026-09-14: (B)**, taken as a normal queue pick under row 6s.  **ANSWERED — B: overrule the carve-out. A partially applied requested audit or controller-substituted remedy does not qualify as a verbatim narrow refinement. Perform the residual audit and run a fresh independent quorum before row 6s can land. The prior controller routing was incorrect; this does not itself reject the code.** (Mark Edmondson, attended 2026-09-07, recorded directly in this ledger.)| Evaluator report `docs/sprint-retros/motoko-iter38-evaluation-round1.md` on branch `docs/motoko-iter38-record`. Quorum artifacts `m-motoko-suite-arm-count-floor-2026-09-07T06-48-47Z.json` (R1) and `-2026-09-07T06-56-41Z.json` (R2), both BLOCKED, 3/3 external present, `absent_reviewers` `[]` cross-checked two ways. Filed by motoko iteration 38.  **Attended ruling 2026-09-07** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue. Provenance is the ATTENDED SESSION, not the commit author: this script stamps a fixed attended identity for EVERY caller (ATT_NAME/ATT_EMAIL are defaults, not derived from the invoker), and nothing in scripts/mission_decisions.sh or the mission-control skill reads the commit author of a ledger resolution (verified 2026-09-04, positive-controlled). The control is the charter rule that the UNATTENDED loop may not resolve a row on its own behalf. Mark explicitly delegated these rulings in the attended 2026-09-07 session: "please make the rulings so we are all unblocked". Codex selected and recorded this scoped ruling under that delegation. Decisions authorize the stated next gates, not fabricated execution or evaluation results.|
| D-MOTOKO-P2-1 | RESOLVED | **RULED (B) — attended 2026-09-08 (Mark)**: count only mandatory, environment-independent arms; report the live loopback check separately, outside the counter; do not adjust the expected count from the observed optional outcome. Taken ahead of the 2026-09-21 default, so row 6s unblocks on motoko's next fire rather than waiting out the clock. Rationale as the loop argued it: (B) makes the gated quantity independent of the environment instead of tracking it, and (A)'s `loopback_sampled` flag is itself a line that can be deleted — the same self-reference trap `gpt6-astra` killed in round 1, i.e. a gate that can be silenced by removing the variable that feeds it is not a gate. Note the loops are PAUSED pending binary mission control, so this takes effect whenever motoko next fires. **The arm-count gate's exact count depends on one environment-conditional arm. Which reviewer's remedy do we take?** Round 3 blocked 3/3 (all present after `gpt6-astra` was re-run alone at a raised cap, having first dropped out on `budget`). All three objections localise on **P2**: the measured Darwin count of 60 holds only while the loopback-socket arm near `test_motoko_connection_probe.sh:640` stays UNINFORMATIVE. Measured first-party in the controller's own baseline run and again by the independent judge; on a Darwin host whose `lsof` sampling succeeds the count is 61 and an exact-equality gate reds for the environment rather than for arm drift. The two remedies are opposite and I may not pick between them. **(A) gemini-3-1-pro** — model the conditional arm: set `loopback_sampled=1` where the arm runs and gate on `expected_arms=$(( 60 + ${loopback_sampled:-0} ))`, keeping strict exact-equality for every other arm. **(B) gpt6-astra** — remove the dependence: count only mandatory, environment-independent arms, report the live loopback check separately outside the counter, and *"do not adjust the expected count using the observed optional outcome"*, which is (A) named and forbidden. **Loop's recommendation: (B)**, because it makes the gated quantity independent of the environment instead of tracking it, and because (A)'s flag is itself a line that can be deleted — the same self-reference trap `gpt6-astra` killed in round 1. **Default if unanswered by 2026-09-21: (B)**, taken as a normal queue pick under row 6s. Nothing else about row 6s is in doubt; the judge passed this iteration's work 90/100 with zero blocking findings. | Round-3 quorum artifact `m-motoko-suite-arm-count-floor-2026-09-07T22-26-24Z.json` (verdict `blocked`, `absent_reviewers` `[{gpt6-astra, budget}]`, cross-checked via `.reviewers[].present`) and the astra re-run at `/tmp/astra_r3_iter39.json`; **both live under `.ailang/`, which `.gitignore:82` excludes, so they are on the rig and not in the tree** — the tracked copies of this iteration's evidence are `docs/sprint-retros/motoko-iter39-residual-audit.md` and `docs/sprint-retros/motoko-iter39-evaluation-round1.md`. Design revision `f6750002d` on `sprint/motoko-iter39-armcount-r3`. Filed by motoko iteration 39. |
| D-MOTOKO-RESET-1 | RESOLVED | **RULED — attended 2026-09-30 (Mark): reset and reconfigure the motoko mission for motoko main.** (1) The charter is reset for the `mk-main` / `sunholo/main-dst` / ABI 8.0 tree; the fork-era queue, premise log and log entries are archived verbatim, not rewritten. (2) The mission STAYS PAUSED — the kill switch `~/.ailang/state/mission-motoko.disabled` is NOT removed by the reset. (3) When it restarts, it restarts on the SHELL driver (`tools/launchd/mission-control.sh`) like every other live mission; it does not wait for the binary mission-control path. (4) The goal is unchanged: make motoko the best harness for writing AILANG, graduating to a mission executor (clause 6). (5) Headline KPI: motoko's pass rate vs pi and opencode on the GPU rotation — same model, same benchmarks, same window — plus the cloud equivalent. | Attended session 2026-09-30, recorded in this ledger under the ATTENDED LEDGER EDITS contract (Gate 0), not via the bookkeeping issue. Charter-reset commit on branch `docs/motoko-mission-reset-20260930`; archive heading `Archived 2026-09-30: fork-era charter (ABI 2.2 → motoko main migration)` in `motoko-mission-status-archive.md`. Supersedes the Phase-0 framing of `D-MOTOKO-1` (see `D-MOTOKO-RESET-3`). |
| D-MOTOKO-RESET-2 | RESOLVED | **Supersedes the `mk-ast` premise of `D-MOTOKO-FMT-1`; the ruling's PRINCIPLE stands.** `D-MOTOKO-FMT-1` made tracing motoko's resolved runtime provider a precondition, and located the trace in the `mk-ast` fork's resolution path. That fork is retired (worktrees removed 2026-09-28) and `fmt` has NOT been ported to ABI 8.0, so the trace, the preflight change it discharged and any fmt measurement taken on `mk-ast` say nothing about the tree evals now run. Carried forward: measure the runtime resolution first, never redesign around an unknown. Re-applied to the new tree as queue row 22 (port fmt into ABI 8.0 and fix `context_limit` resolving to 0), which must re-derive its premises on `mk-main`. `m-motoko-fmt-remeasurement-instrument.md` is historical until that row re-derives it. | Attended session 2026-09-30 (charter reset). Fork retirement: MOTOKO.md §1 and §9. fmt not yet ported: MOTOKO.md §9 "Not yet ported from the fork" and "To do (agreed 2026-09-28)". |
| D-MOTOKO-RESET-3 | RESOLVED | **`D-MOTOKO-1`'s gate is DISCHARGED — the Phase-0 predicate held, so the Phase-0 guardrail is dropped.** `D-MOTOKO-1` made Arni's ABI-settled acknowledgement an objective gate on the extension port. Both halves now hold: upstream PR #154 (`main dst`) merged to `main` 2026-09-25T17:17:10Z, and Arni confirms extension ABI 8.0 is stable. The port happened by adoption rather than by re-porting the 12 ABI 2.2 packages: `mk-main` builds its in-repo `packages/` and the registry packages were unpublished. Archived rows 10, 11 and 12 (the Phase-0-gated port, registry reconciliation, re-baseline) are superseded by the new queue. | `gh pr view 154 --repo arniwesth/motoko_agent` → `MERGED 2026-09-25T17:17:10Z`, measured 2026-09-30. ABI-stable confirmation and the retirement: MOTOKO.md §9 and `ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`. Attended session 2026-09-30. |
| D-MOTOKO-RESTART-1 | OPEN | **When does the motoko loop restart?** The charter is reset and the queue is ready, but the kill switch `~/.ailang/state/mission-motoko.disabled` stays until Mark removes it (`D-MOTOKO-RESET-1` clause 2). Restart means: remove the kill switch, confirm the installed plist runs the shell driver `tools/launchd/mission-control.sh` with `MISSION_PROFILE=motoko`, and let the next fire run iteration 40 against row 20. **No default** — the loop may not unpause itself, and nothing here times out into a restart. | Filed at the attended charter reset 2026-09-30. Kill switch measured present 2026-09-30 (read-only existence check). |
<!-- decision-ledger:end -->

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `motoko-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Rationale:
every iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the
scarcest model budget; the append-only history lives in the log + archive. Write the stamp in the
canonical shape `## STATUS <date> — ITERATION <n>: <title>` (`canonicalStatusRe`,
`internal/mission/normalize.go`) so rotating it into the linted archive cannot raise the
`TestMissionDocHeadingsStayCanonical` ratchet (row 17/18).

> **Reset 2026-09-30 (`D-MOTOKO-RESET-1`).** The archive now holds, newest first: the section
> **"Archived 2026-09-30: fork-era charter (ABI 2.2 → motoko main migration)"** — the fork-era Queue
> (rows 1–19), its Premise Verification Log (V1–V22) and the ITERATION 37 stamp, moved verbatim — then
> the fork-era stamps 36 → 0, then the **pre-2026-08-12 charter in full**. Both archived charters are
> kept because their *findings* remain valid evidence; neither is kept as direction.
>
> **Why two fork-era stamps (ITERATION 39, ITERATION 38) are still below.** Gate 4's structural
> invariant is `grep -c "^## STATUS 2026" design_docs/motoko-mission.md` **= 3**, and Gate 4's
> stale-copy tell greps this file for the previous iteration's stamp with a known-present control
> (`ITERATION 39` / `ITERATION 38`). A reset that left one stamp would break both on the very next
> fire. So the reset was performed AS a normal rotation — reset stamp added, the now-4th stamp
> (ITERATION 37) moved to the archive — and 39/38 will rotate out on the next two fires as usual.
> **Read them as record, not direction**: the row numbers they cite (6s, 7, 10/11/12, 16) are the
> ARCHIVED queue's. Only 6s, 6m, 6t, 9, 17/18 and 19 were carried into the queue below, each
> re-verified at HEAD.
>
> **The next fire is iteration 40, and it OVERWRITES the `ITERATION 40: PENDING` stamp below in
> place** rather than adding a fourth stamp — the iteration-0 precedent of 2026-08-12, where the
> PENDING stamp became the COMPLETE one. Nothing rotates on that fire; the count stays 3.

## STATUS 2026-09-30 — ITERATION 40: PENDING — **CHARTER RESET FOR MOTOKO MAIN BY MARK'S ATTENDED RULING; THE LOOP STAYS PAUSED; NO ITERATION HAS RUN.** `D-MOTOKO-RESET-1`: reset and reconfigure for motoko main (`~/dev/mk-main`, branch `sunholo/main-dst` = Arni's `main`, DST core, extension ABI 8.0, plus our carried commits); keep it **PAUSED** (`~/.ailang/state/mission-motoko.disabled` stays); restart on the **shell driver** `tools/launchd/mission-control.sh`, not the binary path; goal unchanged — the best harness for writing AILANG, graduating to a mission executor; headline KPI = motoko's pass rate vs pi and opencode on the GPU rotation (same model, same benchmarks, same window), plus the cloud equivalent. **What moved, verbatim:** the fork-era Queue, Premise Verification Log and ITERATION 37 stamp → `motoko-mission-status-archive.md` under "Archived 2026-09-30: fork-era charter (ABI 2.2 → motoko main migration)"; log entries 18–39 → `motoko-mission-log-archive.md` (entries 0–39 now all live there); `m-motoko-dst-refactor-migration.md` and `m-motoko-fork-disposition.md` → `design_docs/implemented/v0_48_0/`. **Ledger**: `D-MOTOKO-RESET-1/2/3` RESOLVED (the ruling; `D-MOTOKO-FMT-1`'s `mk-ast` premise superseded; `D-MOTOKO-1`'s Phase-0 gate discharged — #154 merged 2026-09-25, ABI 8.0 stable), `D-MOTOKO-RESTART-1` OPEN with no default. **Where it stands (seeded in the Premise log, P1–P14):** GPU rotation since 2026-09-29 12:00 has `motoko-local-qwen3-8-27b-microrag` at 34/41 (83%), all 7 failures 1h timeouts driven by qwen3.8's hidden reasoning, and thinking stays ON by Mark's ruling; pi/opencode are not yet comparable on that window (broken 09-29 17:07 → 09-30 10:00 by an ollama-rig provider config gap, now fixed); the ailang_tools A/B on deepseek-v4-flash is at ceiling and NOT a measured win; the cloud image moves to motoko main only when #1413 ships. **Carried from the fork-era queue after re-verification at HEAD**: 6s, 6m, 6t, 9, 17/18, 19; **archived as fixed or superseded**: 6j, 6l, 6u, 7, 8, 10, 11, 12, 13, 14 (13/14 re-filed as row 25). Next: **row 20** (cloud motoko executor on motoko main, verified end-to-end in test) once Mark answers `D-MOTOKO-RESTART-1`.

## STATUS 2026-09-07 — ITERATION 39: COMPLETE: **THE AUDIT THE RULING ORDERED WAS RUN, AND IT FOUND A READER CLASS THREE QUORUM ROUNDS HAD MISSED — THEN THE FRESH QUORUM BLOCKED ON A DEFECT NOBODY HAD NAMED EITHER.** Pick was row **6s**, resumed on `D-MOTOKO-CARVEOUT-1`'s attended ruling. **THE RULING IS ACKNOWLEDGED AND ACTIONED: (B) OVERRULE** — *"A partially applied requested audit or controller-substituted remedy does not qualify as a verbatim narrow refinement. Perform the residual audit and run a fresh independent quorum before row 6s can land."* Both were done; nothing else was attempted. **PROVENANCE FLAG, raised rather than actioned around**: `git log -S'| D-MOTOKO-CARVEOUT-1 |'` names the commit author as `Voight-Kampff (bot)`, which Gate 0(b) calls SELF-RESOLUTION. The row's own Evidence cell says it was recorded in an attended 2026-09-07 session under the ATTENDED LEDGER EDITS contract, and that `mission_answer.sh` stamps a fixed identity for every caller so the author byte carries no signal (CLAUDE.md principle 4). **It is honoured because honouring it manufactures no decision**: the recorded answer (B) is identical to both the loop's own written recommendation AND the row's stated `Default if unanswered by 2026-09-14: (B)`, and B is the conservative direction — it blocks row 6s harder rather than unblocking it. Mark should confirm the channel worked; nothing waits on him for it. **THE AUDIT — the thing iteration 38 filed as an open residual and this iteration actually ran.** Full file 1..1212 at base `878939117`, per class, with controls. Its headline is new to all three prior rounds: **the suite re-execs ITSELF 14 times as a child (`/bin/bash "$0"`) across three sub-modes**, and a child starts from `arms=0`, so an exact-equality gate would red for a reason unrelated to arm drift — it does not, ONLY because all three sub-modes exit before the tail (`exit 1` at 12-14, `exit 0` at 181-183 and 353-354; exactly **3** `exit 0` in the file, one of which — 204 — is inside a heredoc fixture body, which the judge verified independently). Now stated as **precondition P1** rather than left implicit. The rest of the inventory: **18** `$0`/`BASH_SOURCE` hits (4 census greps, 14 re-execs, 1 path alias at line 4), **0** sourced files, **25** external referencing files of which exactly **2** are machinery (`make/test.mk` runs it and `bash -n`s it; `scripts/test_check_referenced_paths.sh` asserts the PATH exists, never the contents), and **6** glob gates each measured OUT of scope — `check-file-sizes` is `find internal cmd -name "*.go"`, Go-only, so the 800-line cap cannot see a `tools/eval/*.sh`; `AUTOPUSH_SHELL_SCRIPTS` is two named hook files. **`oc-glm-5-2`'s REMEDY IS NOW ITS OWN TEXT, NOT A SUBSTITUTE.** Its remedy 1 (measure on a real Linux host) is **measured unavailable**: `docker`, `podman`, `colima`, `lima`, `nerdctl`, `orb` all absent, `docker info` fails, control `git` present so the probe can see a positive. So remedy 2 ships verbatim — `else echo 'UNVERIFIED host: arm-count gate skipped' >&2; exit 1` — and the round-2 `expected_arms=56` is GONE, because it rested on a PATH-shadowed `uname`, the controller-substituted remedy the ruling names. Choosing between two reviewer-named remedies on measured availability is not a third option. **ROUND 3 BLOCKED — AND THE ABSENT-REVIEWER RULE PAID FOR ITSELF ON ITS FIRST FIRE HERE.** The synthesis printed `blocked` with `absent_reviewers` naming **`gpt6-astra` on `budget`** — the doc had grown 276 → 520 lines, which is exactly the self-selecting trigger the rule describes, and astra is the reviewer whose objection drove the whole revision. Re-run alone at a raised cap (`design-review --reviewer gpt6-astra --max-cost-usd 0.30`, $0.16374): **REJECT**, and its objection is the sharpest of the three. Final round 3: **3/3 present, 3/3 reject**, plus the controller's own reject. **P2 — THE DEFECT THE FRESH QUORUM WAS FOR.** The measured 60 depends on the loopback-socket arm near line 640 staying UNINFORMATIVE. Confirmed first-party in the controller's own baseline run (output line 34: `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer`) and independently by the judge. On a Darwin host where that arm's `lsof` sampling succeeds the count is **61** and an exact-equality gate reds for the environment. **astra's and gemini's remedies CONTRADICT**: gemini says `expected_arms=$(( 60 + ${loopback_sampled:-0} ))`; astra says *"Do not adjust the expected count using the observed optional outcome"* and instead count only environment-independent arms. Filed as **`D-MOTOKO-P2-1`**. **THE JUDGE PASSED THE WORK 90/100, ZERO BLOCKING, AND CORRECTED THE CONTROLLER'S REASONING — WHICH IS ADOPTED.** It reproduced every audit row first-party from base rather than from the doc's numbers, verified the heredoc boundary at 204 itself, re-ran the 878939117 deletion mutant live (`PASS: 58`, zero `not ok` — the defect reproduced, not asserted), and **ran the addition and self-arm mutants the round-3 doc only SPECIFIED** (`61 != 60` and `59 != 60`, both red) — because a Test-plan row is a spec, not a measurement, and round 3 had narratively called them "proven". Its item 6: the controller's stated reason for parking — that choosing between astra and gemini would replay the forbidden carve-out — **overreaches `D-MOTOKO-CARVEOUT-1`'s scope**, which was about applying a verbatim fix WITHOUT a re-quorum, not a blanket ban on controller synthesis. **The judge is right and the correction is recorded here so it is not inherited as precedent**: the park is correct on the narrower and sturdier basis that Gate 2's default is one-revision-one-requorum-then-park, and the carve-out is independently unavailable because both P2 remedies dispute the design DIRECTION rather than completeness. **ROUTING — the operator's standing request was all four roles via the Agent tool, and it was measured rather than inherited.** The designer spawn was attempted and **DENIED first-party**: `deny:provider-pin — designer is pinned to claude:claude-fable-5-1; Agent-tool alias spawn refused` — **instance 4** for row **6u**, whose ≥3-evidence bar was already met at iteration 38. Designer then ran on the rotation entry `claude:claude-fable-5-1` through the `claude-sub` recipe (probe rc=0 `ok`, real run rc=0, 276 → 520 lines, one file). **Planner and executor DID NOT RUN, and that is the routing table applying rather than being skipped**: both gate on artifacts a quorum-blocked doc does not have — there is no approved design, so no plan is owed and nothing is owed execution. Evaluator `sonnet` via the Agent tool (alias pin, ACCEPTED), own worktree, 139,623 tok / 57 tool calls / 18m. generator≠judge holds at the model level (sonnet ≠ fable designer ≠ opus controller) and **is FLAGGED at the vendor level** — all three are Anthropic, because codex routing is blocked on a stale provider observation and the pi/minimax evaluator lane timed out at iteration 37. **COSTS**: metered **$0.27158** of the $5 ceiling ($0.10784 round-3 quorum + $0.16374 the astra re-run); everything else subscription or flat-rate. Gate 0: kill switch armed (`mission-motoko.disabled`), `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, **0** human directives on `#1078` since watermark `2026-09-07T00:00:00Z` (4 comments, none allowlisted), ledger valid at 7 rows. Gate 1: `dev` **not red** — `CI` in-flight at `ead709c31` with `test` pending, `Build and Release` success, 14 checks, 1 non-green and it is the pending one; motoko does not own this repo in any case. Blocked-row predicates re-run as commands: **0** `arniwesth` comments on upstream `#165` (control: **2** total comments, and a `commenter:arniwesth` search returns **1**, so the instrument sees him), `#154` still `open`/unmerged, negative control `#999999` 404s — rows 10/11/12 stay Phase-0 parked. **A GATE-1 INSTRUMENT DEFECT FOUND FIRST-PARTY AND FIXED AT GATE 5**: the prescribed running-skill drift check greened on all 12 files while the copy this controller was actually READING — the pin worktree's, resolved relative to CWD — was **68 lines behind origin on `gate-3-route.md` and `gate-3b-ci-green.md`**. Next: **`D-MOTOKO-P2-1`'s answer**, then row **6s** in one more revision round; row **7** still needs its premise restated.

## STATUS 2026-09-07 — ITERATION 38: COMPLETE: **THE INDEPENDENT JUDGE FAILED THE CONTROLLER, NOT THE CODE — AND IT WAS RIGHT.** Two picks: row **16** (bookkeeping, ordered next) LANDED; row **6s** DONE, MEASURED, and **PARKED `needs-human-review`** on a process question about how its design was approved. **Row 16** — PR [#1076](https://github.com/sunholo-data/ailang/pull/1076), squash-merged as [`a329fdb4f`](https://github.com/sunholo-data/ailang/commit/a329fdb4f). **Gate 3b: 21 checks, 0 pending, required 4/4 green** (`test`/`lint`/`build`/`docs-gate`); the single non-green, `launchd drivers (bash 3.2)`, is NOT a required context and is INHERITED — the control is that the identical check is `failure` on `origin/dev`'s own HEAD, and this PR is a docs-only single-file change. `UNSTABLE` is not `BLOCKED`. The charter row's premise had MOVED and the commit corrects it: the row's stated reason for deferring ("the top entry is the shipped `v0.35.1`, so a new unreleased section would misdescribe both halves") no longer holds because an `## [Unreleased]` section now exists, so the feature and its repair land in one entry as the row asked. The row's other measurement had also moved — `git log --name-only f5edd569a~1..origin/dev -- changelogs/ CHANGELOG.md` is no longer empty (V1's entries landed in between), so the predicate was re-derived as `grep -rniE "workbench" changelogs/ CHANGELOG.md` → **0**, with `git log --oneline -3 -- CHANGELOG.md` → **3** as the control proving the grep could see a positive. **The executor overruled the controller's directive and was right** (rule 3h, adjudicated by measurement not by a deviations-are-suspect prior): the directive quoted M2's "workdir absoluteness is POSIX, not host-dependent", but M8 [`a427154c8`](https://github.com/sunholo-data/ailang/commit/a427154c8) supersedes it — a POSIX-only check rejects the fixtures' own Windows temp dirs — so the entry says what M8 says. **Row 6s** — the defect was reproduced FIRST-PARTY before any routing: pristine `PASS: 59`, delete one whole arm → `PASS: 58`, **both rc=0 and fully green**, tree restored sha256-identical. Shipped shell adds an `expected_arms` drift gate whose own `pass_arm` runs BEFORE a bare `(( arms != expected_arms ))`, host-scoped 60 Darwin / 56 otherwise. Verified out-of-sandbox by the controller: pristine `PASS: 60` rc=0; **R2 addition rc=1 `suite ran 61 arms`** (the only row that proves the gate LOOKS rather than merely FIRES) and **R3 self-arm deletion rc=1 `suite ran 59 arms`**, each restored to `52ab531e…296df7`. The executor ran R1/R4 too, all red. **THE QUORUM BLOCKED 3/3 IN BOTH ROUNDS, ALL PRESENT, `absent_reviewers` `[]` cross-checked two ways.** R1's killer was `gpt6-astra`: the first draft's `arms + 1 != expected_arms` counts an ASSUMED execution, so deleting the gate's own line computes 59+1=60, PASSES, and prints 59 — the fix recreating the defect. R1 also gave `gemini-3-1-pro` (an unreachable duplicate zero-guard, TRUE) and `oc-glm-5-2` (unproven Conflict-Surface enumeration), plus the controller's own reject (an exact gate on a host-dependent count). **EVERY OBJECTION WAS MEASURED RATHER THAN FORWARDED** (rule 3f): determinism **4/4 runs at `PASS: 59`**; the non-Darwin count MEASURED at **55** (+ the gate's own arm = 56) via a PATH-shadowed `uname -s` returning `Linux`, control `/usr/bin/uname -s` → `Darwin` in the same call; the uniqueness premise re-proven with a GLOBAL `grep -nw arms` census over all 1212 lines (controls: `pass_arm` **19**, invented literal **0**) instead of the doc's 7-line `sed`; and the exhaustiveness question closed at **4** greps of `"$0"` with **3** of `"$probe"` as the cross-check. **THE CONTROLLER'S OWN ROUND-2 FINDING WAS THAT THE DOC CONTRADICTED ITSELF**: it claimed the suite "is not referenced by CI or the Makefile", which is FALSE — `make/test.mk:72` runs it inside `test-launchd-drivers` and `ci.yml:602` runs that in the `launchd drivers (bash 3.2)` job the same doc correctly cites as `runs-on: macos-latest`, so a wrong count is a repo-blocking red rather than a contributor annoyance. **ROUND 2 BLOCKED 3/3 AGAIN AND THE OBJECTIONS LOCALISED ONTO ONE SURFACE** — *is the gated quantity actually characterised?* — with nobody disputing the direction and all three carrying concrete reviewer-authored `proposed_fix` text. The controller therefore invoked Gate 2's **narrow-refinement carve-out** for a bounded SECOND revision rather than parking. **THE INDEPENDENT EVALUATOR (Agent tool, `sonnet`, against a `codex:gpt-5.6-sol` executor — distinct agent, model AND provider) FAILED IT 68/100 WITH TWO BLOCKING FINDINGS, BOTH AIMED AT THAT DECISION**, and both reproduce against the controller's own directives: `gpt6-astra`'s fix explicitly said *"Mark the prior quorum objection unresolved until this audit is recorded"* and only its cosmetic half was applied; `oc-glm-5-2` named two remedies (a real Linux host, or an explicit `UNVERIFIED` refusal in the shell) and the controller shipped a third. The carve-out's own text says *"their own text — never a controller-invented resolution"*, so **the honest reading is that this should have parked, and it is parked now** — filed as `D-MOTOKO-CARVEOUT-1` with the loop recommending **(B) OVERRULE** against its own decision. **THE JUDGE ALSO CORRECTED A CONTROLLER-SUPPLIED MEASUREMENT**, which is the check it was explicitly asked to run: "`grep -c UNINFORMATIVE` → 4, exactly the 4 skipped arms" is a COINCIDENCE — the Darwin control run already prints **1** UNINFORMATIVE line, so the delta is **3** lines covering **4** lost arms (the REAL_LSOF line covers two). The 56 stands because it was measured directly; the supporting data point did not. Its own devised mutant (`!=` → `>=`) made the addition mutant invisible, which is the design's exact-equality-over-floor argument proving itself. **THE EXECUTOR STOPPED AT A BOUNDARY RATHER THAN ADJUST A CONSTANT**, exactly as directed: acceptance criterion 5's `grep -c 'pass_arm '` returns **20**, not 19, because the gate's own explanatory COMMENT contains the literal. Controller re-derived 20 bare / **19** executable / **18** at base — the number was right and the COMMAND was wrong, and only the command changed; the reviewer-mandated comment stays. **GATE 1 FOUND `dev` RED TWO WAYS AND HANDED BOTH TO V1** per the owning-mission rule: `test` failing `make check-file-sizes` (`cmd/ailang/exec.go` 807 > 800, new at the tip from `8c41d41d4`, green at parent `92a9df7f9`) and `launchd drivers (bash 3.2)`, inherited across 6+ consecutive commits. V1 fixed the first mid-iteration ([`16f0cb741`](https://github.com/sunholo-data/ailang/commit/16f0cb741), #1074), which moved the base under this run — the row-16 branch was rebased onto it and the changelog conflict resolved by keeping BOTH sides, and the stale CI poller watching the pre-rebase head was KILLED rather than left to expire (rule d-bis). **ROW 7 WAS SKIPPED WITH A RECORDED REASON, NOT SILENTLY**: its charter row is one line and its premise ("5 profiles, 14 of 18 model entries") is not reconstructible at pick time, so routing a designer at it would be a sprint on a phantom; it needs a restated premise before it can be picked. **ROUTING — all four roles ran, and the Agent tool was usable for exactly one of them.** The operator's standing request for this fire was to spawn all four through the Agent tool; the spawn-pin hook DENIES that path for any role whose `MISSION_<ROLE>_MODEL` contains a colon, which is designer (`claude:claude-fable-5-1`), planner and executor (`codex:gpt-5.6-sol`) — so no spawn was burned on a guaranteed denial and each was routed to its own lane recipe, with the resolver's answer and the pin both recorded. Only the **evaluator** (`sonnet`, an alias pin) could go through the Agent tool, and it is the one the operator called non-negotiable. Designer was the ROTATION entry after `codex:gpt-6-astra` — `pi:ollama/deepseek-v4-flash:0731-cloud`, independent of all three quorum reviewers — run through `scripts/mission_pi_run.sh`, verdict `ok` on all three runs (395s / 81s / 121s, 1 changed file each, `agent_end` present). **A CONTROLLER ERROR WORTH RECORDING**: a `pkill -f` aimed at a superseded poller matched the launcher shell of the live designer run by its own argv and killed it; the runner survived, the `.done` marker did not, and the recovery was to poll the runner's verdict artifact instead. Kill by task id, not by a pattern that also matches the work. **COSTS**: metered **$0.1546** of the $5 ceiling (two quorum rounds, $0.0657 + $0.0889); everything else is subscription or flat-rate. Gate 0: kill switch armed (`mission-motoko.disabled`), `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, **0** human directives on `#987` since watermark `2026-09-06T00:27:48Z` (15 comments, none allowlisted), ledger valid, running skill byte-identical to `origin/dev` on the RESOLVED symlink target and on the pin copy. Blocked-row predicates re-run as commands: **0** `arniwesth` comments on upstream `#165` with control **35**, `#154` still OPEN — rows 10/11/12 stay Phase-0 parked. Source clone **345 behind**, up from 237 at iteration 35; `D-MOTOKO-WORKDIR-2`'s standing authorisation still covers it and the growth rate is the story. Gate-5: **no skill edit** — the carve-out question is a human ruling, not a rulebook gap. Next: **`D-MOTOKO-CARVEOUT-1`'s answer**, then row **6s** on its terms; row **7** needs its premise restated first.

## Premise Verification Log

Reset 2026-09-30. The fork-era log (V1–V22, measured 2026-08-12 against `mk-ast` and
`origin/main_dst@303d8697`) is archived verbatim; most of its rows describe a tree that no longer runs.
**Same acceptance rule, carried over:** a safety-, routing- or queue-ordering claim in this charter
carries a row here or the label `UNVERIFIED`. Each row names its source; "reported" means it came from
the attended 2026-09-30 session's own measurements and was not re-run by the charter author, so the
first iteration that leans on it re-measures it.

| # | Claim | Source / how measured | Result |
|---|---|---|---|
| P1 | Evals and the `motoko` shim run `~/dev/mk-main`, branch `sunholo/main-dst` = upstream `main` (DST core, ABI 8.0) + our carried commits (cloud and ollama_microrag profiles, `motoko_ext_ailang_tools`, rig-lease forwarding, relock, dropped dead `enable_thinking`) | [MOTOKO.md](../MOTOKO.md) §1, §2, §9; attended session 2026-09-30 | **Confirmed (documented)** — re-check with `git -C ~/dev/mk-main log --oneline origin/main..sunholo/main-dst` before relying on the exact delta |
| P2 | Extensions are in-repo path packages under `mk-main/packages/`; the 14 ABI 2.2 registry packages `sunholo/motoko_ext_*` were unpublished 2026-09-28 | MOTOKO.md §3; `ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md` | **Confirmed (documented)** |
| P3 | The ABI 2.2 fork is retired: `mk-ast` worktree (branch `sunholo/eval-canonical`) removed 2026-09-28; the branch survives on the fork | MOTOKO.md header and §9 | **Confirmed (documented)** |
| P4 | Upstream state: #154 (`main dst`) MERGED 2026-09-25T17:17:10Z; #191, #193, #198 MERGED 2026-09-28; #200 (`motoko_ext_ailang_tools`) OPEN; #192 is our open-questions thread; Arni confirms ABI 8.0 stable | `gh pr view <n> --repo arniwesth/motoko_agent`, run 2026-09-30 (192 is not a PR, so `pr view` cannot resolve it — it is the issue MOTOKO.md §9 cites); ABI-stable: MOTOKO.md §9 | **Confirmed live 2026-09-30** for the PR states; ABI-stable is documented, not re-asked |
| P5 | GPU rotation since 2026-09-29 12:00: `motoko-local-qwen3-8-27b-microrag` 34/41 (83%); all 7 failures are 1h timeouts on quine, legal_obligation_engine, commonmark_emphasis and gauntlet_10 | attended session 2026-09-30 | **Reported** — re-measure from the bank (`ailang eval-elo` / the banked rows' `error_category`) before citing a number in a KPI |
| P6 | The timeouts are driven by qwen3.8's hidden reasoning: 10k–25k output tokens per slow step | attended session 2026-09-30 (per-step token counts) | **Reported.** Mark's ruling: thinking stays ON — "a failure of the benchmark if whilst thinking they can't finish in time". Not a harness knob to turn |
| P7 | `agent_options.enable_thinking` was dead config in motoko — never sent on the wire — and was dropped on `sunholo/main-dst` | attended session 2026-09-30; carried commit listed in P1 | **Reported** — the class is the fleet's "declared ≠ walked" (see the provider-side Broadcast trace instrument in CLAUDE.md) |
| P8 | pi and opencode are NOT yet comparable to motoko on that window: they were broken 2026-09-29 17:07 → 2026-09-30 10:00 by an ollama-rig provider config gap, now fixed | attended session 2026-09-30 | **Reported** — the KPI row (23) must start its window after the fix and exclude the broken span, not average across it |
| P9 | `ailang_tools` A/B on deepseek-v4-flash is NOT a measured win: trial 1 on 23/23 vs off 20/23; trial 2 on 21/23 vs off 23/23 — the set is at ceiling for that model | attended session 2026-09-30 | **Reported** — this is why row 24 re-measures on a set with headroom |
| P10 | The motoko cloud executor image still carries the old fork; [#1413](https://github.com/sunholo-data/ailang/pull/1413) moves it to motoko main (pins `4d4917cd`) and reaches test/prod only via a release + promote | `gh pr view 1413 --repo sunholo-data/ailang` → OPEN, 2026-09-30; deploy path: executor images build on a release tag and reach prod by promote, so a merge alone changes nothing that runs | **Confirmed live 2026-09-30** (PR open) |
| P11 | `ailang run` is quiet by default — progress lines were polluting stdout (quine) | `gh pr view 1410 --repo sunholo-data/ailang` → MERGED 2026-09-30T09:14:19Z | **Confirmed live 2026-09-30** — needs a release before rig evals see it |
| P12 | `fmt` is not yet ported to ABI 8.0, and `context_limit` resolves to 0 in eval workspaces (`context_limit_resolved` shows `profile_key_absent` / `catalogue_absent`) | MOTOKO.md §9 "Not yet ported"; attended session 2026-09-30 for the resolver reading | **fmt: confirmed (documented). context_limit: reported** — row 22 re-measures it first |
| P13 | The mission is paused and restarts on the shell driver | kill switch `~/.ailang/state/mission-motoko.disabled` present (read-only existence check, 2026-09-30); `D-MOTOKO-RESET-1` | **Confirmed live 2026-09-30** |
| P14 | Which fork-era open rows still apply at HEAD `91eb860a2` | per row: 6s — `expected_arms` absent from `tools/eval/test_motoko_connection_probe.sh` (0 hits) while `sprint/motoko-iter39-armcount-r3` @ `f6750002d` exists on origin; 6m — `cached_tokens` appears in no `ParseChatStepResponse` test; 6t — `gate-3b-ci-green.md` still prescribes a foreground 1800s poll with no background note; 17/18 — no canonical-stamp instruction in `gate-4-record.md`; 19 — `rotate.go` still refuses a status archive with interleaved sections, and the archive still has them; 6j — `launchd drivers (bash 3.2)` **27/27 success** over the completed dev CI runs since 2026-09-29T09:17Z, and row 6p (its stated scope) LANDED 2026-09-06; 6l — `tools/launchd/lib/pin-root.sh` now refreshes the gate from the ref (`git show "$target:tools/launchd/lib/pin-root.sh"`) before deciding; 6u — the fallback half was built as D-FLEET-2 (`49f18bdbd`, 2026-09-29) and the resolver-vs-hook half is tracked in `m-resolver-hook-disagree-on-docless-pick` | **Measured 2026-09-30** — carried: 6s, 6m, 6t, 17/18, 19 (+ 9, never run); archived as fixed/superseded: 6j, 6l, 6u |

## CURRENT GOAL

**Make motoko the best harness for writing AILANG, and graduate it into a mission executor**
(clause 6) — unchanged by the 2026-09-30 reset (`D-MOTOKO-RESET-1`). What changed is the tree: the
migration epic is DONE (motoko main, ABI 8.0, in-repo extensions — see the Repo Profile), so the
mission now improves and measures the tree evals actually run instead of porting towards it.

**Headline KPI (Mark, 2026-09-30):** motoko's pass rate **vs pi and opencode on the GPU rotation** —
same local model, same benchmarks, same time window, harness defects excluded first — **plus the
cloud equivalent** (same cloud model through each harness). A motoko number with no pi/opencode
number beside it from the same window is not a KPI reading. Paired, not pooled: use
`ailang eval-paired` (discordant pairs + headroom warning) and `ailang eval-elo` (difficulty fitted
apart from harness strength), never a raw aggregate across different windows or model versions.

**Where it stands at the reset:** local motoko on qwen3.8 is at 34/41 on the rotation with every
failure a thinking-driven 1h timeout (P5/P6); pi and opencode have no comparable window yet (P8); the
one extension A/B we have is at ceiling (P9); the cloud image still runs the old fork (P10). So the
first reading of the KPI does not exist yet — building it (row 23) and getting the cloud executor
onto motoko main (row 20) come before any claim that motoko is ahead or behind.

**Status**: PAUSED. Restarts only on Mark's say-so (`D-MOTOKO-RESTART-1`), on the shell driver.

## DST scope — what it actually covers, measured (2026-08-12; corrects an earlier over-read)

> **Reset note 2026-09-30.** Measured 2026-08-12 against Arni's `main_dst` (HEAD `b3953a9`), before
> #154 merged. Kept because the core-vs-extension coverage split is still the right way to price
> DST work; re-measure against `mk-main` before quoting any number from it.

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
- **Clause 2 — No extension drift.** Extensions build from the pinned motoko main commit, with no
  drift between the rig (`mk-main`), the cloud executor image and our upstream PRs: one source per
  extension, in-repo under `mk-main/packages/`, and the same commit wherever motoko runs.
  *(Reworded at the 2026-09-30 reset, `D-MOTOKO-RESET-1`; the ratified 2026-08-12 wording —
  "published, registry-pinned, and ABI-current" — described the retired registry packages.)*
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

- **The kill switch is Mark's.** `~/.ailang/state/mission-motoko.disabled` is removed only on his
  say-so (`D-MOTOKO-RESTART-1`). An iteration never removes it, and a reset or charter edit never
  implies it.
- **We are GUESTS in `arniwesth/motoko_agent`.** Never push to it. PRs only, never force a draft to
  ready, and never re-open something the maintainer closed without Mark. He is hands-on.
- **Keep the carried delta small and upstream-bound.** Every commit on `sunholo/main-dst` that is not
  upstream is either on its way there as a PR or has a written reason to stay local (MOTOKO.md §9
  lists them). When an upstream PR merges, drop our carried copy in the same change (row 21 is the
  first instance).
- **Three repos, one mission.** `sunholo-data/ailang` is the anchor (evals, benchmarks, design docs,
  and the only issue queue). `~/dev/mk-main` (see [MOTOKO.md](../MOTOKO.md)) is the motoko work
  surface, not a mission repo; `sunholo-data/ailang-packages` no longer hosts motoko extensions
  (P2). Gate 3b CI applies to the anchor.
- **V1 OWNS DEV CI RED ON THE ANCHOR. YOU DO NOT — hand it over and keep your own pick.** V1 and this
  mission both run `MISSION_REPO=sunholo-data/ailang` (separate clones, one GitHub repo), so a red dev
  is visible to both — and the skill's rule that a red "hits whoever observes next" silently assumes a
  single observer. There is not one. The driver's overlap guard is per-mission by construction
  (`PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"` — each loop guards only against *itself*), and no
  cross-mission mutex exists; iterations deliberately never take `rig.lock` (GPU mutex only).
  **Measured 2026-08-17:** both loops preempted onto the same red and opened
  [#758](https://github.com/sunholo-data/ailang/pull/758) (this mission, iteration 9) and
  [#759](https://github.com/sunholo-data/ailang/pull/759) (V1, iteration 217) — **the same six files**.
  So: **on observing a red dev, record it, hand it to V1 via the cross-mission channel, and proceed
  with your own pick. A red you do not own never outranks it.** ONE carve-out: if the red is *yours* —
  a commit or PR from this mission caused it, or it sits in motoko/eval-lane territory V1 has no
  domain knowledge for — you own the fix, because handing that to V1 strands it.
- **Never run `ailang fmt` across motoko sources.** It reflows whole expressions and inserts blank
  lines between imports — hundreds of lines of conflict surface against an upstream we must stay
  mergeable with, for no benefit. (Porting the `fmt` *extension*, row 22, is a different thing.)
- **`make quick-install` is a SHARED WRITE — treat the verify profile as touching V1 and the rig**
  (measured V20, 2026-08-12, archived). It installs to `~/go/bin/ailang`, the binary V1's iterations
  and the eval rig resolve through `PATH`. A separate checkout isolates the *working tree*, not the
  installed toolchain. So: before any gate that runs `quick-install`, confirm this checkout is not
  behind `origin/dev`, and never run it from a tree carrying experimental compiler changes. If a gate
  needs an experimental binary, build to `bin/ailang` (`make build`) and invoke it by path — do not
  install it. Never `cp` over a live `~/go/bin/ailang` either; an in-place overwrite kills running evals.
- **Thinking stays ON for the local rotation** (Mark, 2026-09-30, P6). A benchmark a thinking model
  cannot finish in its budget is a finding about the benchmark or the budget, not a reason to turn
  reasoning off; only evals may restrict thinking, and not this one.
- **Harness defects are excluded BEFORE any KPI comparison.** A harness-crash, provider-config or
  rig-gateway failure on any of the three harnesses is a defect to fix, not a data point for or against
  motoko (P8 is the live instance). Read `error_category` first; `api_error` means "cause unknown".
- **Never conclude "model wall."** Every motoko disengagement investigated on this mission so far
  has been a harness bug. Prove it with `ailang chains` / the wire bytes before claiming capacity.

## Routing policy

Uses the **shared** per-role model routing from `mission-control` (controller / designer-rotation /
planner / executor / evaluator, generator≠judge enforced), resolved by the shell driver from its
defaults plus `~/.config/ailang/mission-motoko.env` (repo copy:
`tools/launchd/mission-env/mission-motoko.env`, which carries no role overrides today). **Measure
the live routing, do not read it off this charter**: `MISSION_PROFILE=motoko MISSION_DRY_RUN=1
tools/launchd/mission-control.sh` prints the resolved roles. The fleet's codex lanes moved to
GPT-6.1 Sol on 2026-09-30 (#1412), so any model name written into the archived charter is stale.

- **Executor**: the driver's ratified non-Anthropic-first chain, so a second concurrent loop does not
  double the Anthropic burn.
- **Evaluator**: the shared default, which must differ in provider from the executor.
- `PATH` is set in the env file to include `/opt/homebrew/bin` — see the World mission's
  iter-18-to-22 lesson, where a PATH-less plist silently demoted the codex lane to opus on **every**
  fire for five iterations before anyone noticed.
- **`motoko:<model>` is not yet an executor lane** — that is row 25 (clause 6).

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**Era**: motoko main (reset 2026-09-30). The migration epic
([m-motoko-dst-refactor-migration](implemented/v0_48_0/m-motoko-dst-refactor-migration.md)) is
DONE. New rows start at **20** so they cannot be confused with the archived fork-era rows 1–19, which
the ITERATION 38/39 stamps above still cite. Carried fork-era rows keep their old IDs; their full
original filings are in the status archive's fork-era Queue. Every row needs a done-test a command can
check — "improved" is not one.

20. [NEXT — waits only on `D-MOTOKO-RESTART-1`] **Cloud motoko executor runs motoko main, verified
    end-to-end in test** · clauses 1+6 + KPI (cloud half) ·
    [#1413](https://github.com/sunholo-data/ailang/pull/1413) moves the image to motoko main (pins
    `4d4917cd`) but an image reaches test/prod only through a release + promote (P10). **Done when**:
    #1413 is merged; a release carrying it is cut; the test environment runs that image (the deployed
    revision's image digest/tag names the release — measured, not inferred from the merge); and ONE real
    cloud task dispatched to the motoko executor in test completes with its result banked, with the
    provider-side trace (OpenRouter Broadcast `rawRequest`) confirming the model and budget that went
    on the wire. A green build or a smoke reply is not the done-test · 1–2 iterations (release cadence
    is outside this loop; park on it rather than wait)
21. **Upstream #200 (`motoko_ext_ailang_tools`) merged, then drop our carried commit** · clauses 1+2 ·
    [arniwesth/motoko_agent#200](https://github.com/arniwesth/motoko_agent/pull/200) is OPEN (P4).
    Answer review asks through PRs only (Guardrails: guests). **Done when**: `gh pr view 200 --repo
    arniwesth/motoko_agent` reports `MERGED`; `sunholo/main-dst` is rebased onto the upstream `main`
    carrying it; `git log origin/main..sunholo/main-dst` in `mk-main` no longer lists our copy of the
    extension; and `make check_core && make verify_extensions` is green on the rebased tree. Bounded wait:
    if #200 has no maintainer activity for 14 days, record it and move on — never an open-ended wait on a
    person (the fork-era `D-MOTOKO-1` lesson) · 1 iteration of work + a bounded wait
22. **Port `fmt` into ABI 8.0, and fix `context_limit` resolving to 0 in eval workspaces** · clauses 3+5 ·
    `fmt` did not come across from the fork (P12), and the step-0 resolved-config event reports
    `context_limit_resolved` as `profile_key_absent` / `catalogue_absent`, i.e. 0 — so compaction and
    budget logic run blind in every eval workspace. Likely home for fmt: inside
    `motoko_ext_ailang_tools` (MOTOKO.md §9: a `ToolProvider` can wrap native `WriteFile`/`EditFile`).
    Supersedes the fork-era fmt instrument's premise (`D-MOTOKO-RESET-2`): re-derive on `mk-main`, do
    not reuse `mk-ast` numbers. **Done when**: (a) a motoko run in an eval workspace emits a non-zero
    `context_limit` with a resolution source other than `*_absent`, for every `motoko_profile:` in
    `internal/eval_harness/models.yml` that the rotation or cloud uses, with a test pinning the
    resolution; (b) fmt runs as an ABI 8.0 extension on `mk-main`, `make verify_extensions` is green,
    and its effect is measured before/after on the rig (clause 3) — kept only if the measurement
    supports it, dropped otherwise · 2 iterations
23. **KPI instrument: paired motoko vs pi vs opencode on the GPU rotation, and the cloud equivalent** ·
    clause 3 + the headline KPI · The measurement the goal is judged by does not exist yet (P5, P8).
    Same model, same benchmarks, same time window; harness defects (crash, provider config, rig
    gateway, `api_error` of unknown cause) excluded first and listed, not silently dropped. Use
    `ailang eval-paired` for each pair and `ailang eval-elo` for harness strength net of benchmark
    difficulty. **Done when**: one committed, re-runnable command (or documented `ailang` invocation)
    produces, for a stated window starting after the 2026-09-30 10:00 fix: per-harness pass rate, the
    discordant-pair counts for motoko-vs-pi and motoko-vs-opencode, the excluded-defect list with each
    row's `error_category`, and the headroom warning — for the local qwen3.8 rotation AND one cloud model
    run through all three harnesses; and the first reading is recorded on the dashboard · 1–2 iterations
24. **Measure `ailang_tools` on a set with headroom** · clauses 3+5 · The only A/B we have is at
    ceiling on deepseek-v4-flash (on 23/23 vs off 20/23, then on 21/23 vs off 23/23 — P9), which
    cannot show a win or a loss. Re-run on local qwen3.8 (the rotation model, 83% with real failures),
    on benchmarks where at least one arm fails. **Done when**: an `ailang eval-paired` on/off result on
    local qwen3.8 over a set with headroom is banked, with at least one discordant pair or an explicit
    "no discordant pairs in N" statement, and the verdict (keep / drop / inconclusive, with the headroom
    warning's reading) is written here and on the dashboard · 1 iteration + rig time under `rig.lock`
25. **Executor-lane design + gate trial (`motoko:<model>` as a mission executor)** · clause 6 · The
    fork-era rows 13/14, previously parked on "a motoko we can rebuild" — now unblocked (clause 1 holds on
    `mk-main`). Scope as clause 6 lists it: a `provider:model` spawn recipe, a bounded token-cheap probe,
    a fallback-chain slot that degrades LOUDLY, a real-run recipe (write sandbox, background spawn,
    `< /dev/null`), and the false-green guards that killed the DeepSeek-Flash lane (3/3 real-sprint
    failures behind `rc=0`). **Target World (`ailang-code` profile) first, not this Go repo.** **Done
    when**: (design) a design doc passes `ailang design-quorum`; (trial) motoko lands ≥1 real World
    sprint with its held-out tests passing and a non-empty, plan-faithful diff, verified by an
    independent evaluator — a smoke reply or `rc=0` alone does not count · 2+ iterations

**Carried from the fork-era queue** — each re-verified at HEAD `91eb860a2` on 2026-09-30 (P14); none
depends on the retired fork.

6s. [PARKED → unblocked by `D-MOTOKO-P2-1` (B), attended 2026-09-08] **`expected_arms` drift gate for
    `tools/eval/test_motoko_connection_probe.sh`** · loop health · Design revision `f6750002d` and the
    M1 code sit unmerged on `sprint/motoko-iter39-armcount-r3`; the ruling says count only mandatory,
    environment-independent arms and report the loopback check outside the counter. **Done when**: the
    gate lands on `dev` counting only environment-independent arms, the addition and removal mutants
    each red it, and `launchd drivers (bash 3.2)` is green on the merge · 1 iteration
6m. **`cacheRead = usage.PromptTokensDetails.CachedTokens` in `ParseChatStepResponse` is pinned by
    nothing** · loop health · still true at HEAD: no `ParseChatStepResponse` test carries
    `cached_tokens` (`cache_usage_test.go` exercises `Generate`, a different path). **Done when**: an arm
    drives a body with `prompt_tokens_details.cached_tokens` through `ParseChatStepResponse` and asserts
    `CacheReadInputTokens`, plus the absent → 0 arm, and a `+ 999` mutant of the line reds it · 1 iteration
6t. **Gate 3b's prescribed 30-min CI poll cannot finish in a 10-min foreground `Bash` call** · loop
    health · `resources/gate-3b-ci-green.md` still prescribes `deadline=… + 1800` with no
    `run_in_background` note. Shared-skill edit — land it from the checkout the running skill resolves
    to. **Done when**: the gate's snippet is run in the background (or bounded under 10 min with a re-poll
    rule) and the file says why · 1 iteration
17/18. **Gate 4 must WRITE canonical STATUS stamps so rotation cannot raise the heading ratchet** ·
    loop health · (rows 17 and 18 were filed twice for one defect.) `canonicalStatusRe` is
    `^## STATUS <date> — ITERATION <n>: <title>`; `gate-4-record.md` still says nothing about the shape,
    so any mission writing `ITERATION <n> COMPLETE:` reds `TestMissionDocHeadingsStayCanonical` the
    moment the stamp rotates. This charter's rotation rule now states the shape for motoko; the shared
    fix is still owed. **Done when**: Gate 4 prescribes the canonical shape for every mission and
    `go test ./internal/mission/...` stays green across one rotation in each live mission · 1 iteration
19. **The iteration index cannot be regenerated for motoko** · loop health · `ailang mission rotate-log
    motoko --status` refuses (interleaved structural sections in the status archive), and plain
    `rotate-log motoko` needs ≥1 live log entry (`keep must be >= 1`), so `motoko-mission-index.md` is
    hand-maintained. **Done when**: an index-only regenerate mode exists (the index build reads both files
    and does not need the archive to be rotatable) and running it on this mission reproduces the current
    index byte-for-byte plus any new rows · 1 iteration
9. [PARKED — after row 23 exists] **R3 — cross-model generality study** · clause 5 + the north star's
    weak-model thesis · do motoko's gains hold with strong models, and are they AILANG-specific or
    general? Never run (fork-era V18). Row 23's cloud half is its natural first data. **Done when**: the
    paired comparison is run on one weak and one strong model through the same harnesses and the result
    is recorded against the north star's claim, whichever way it falls · 2 iterations

**Archived with the fork-era queue, not carried** (full text in the status archive): 1–6, 6b–6i, 6k,
6n–6r, 15, 16 (LANDED/CLOSED); **6j** (bash 3.2 arm hang — superseded by 6p's in-test bound derivation,
LANDED 2026-09-06, and 27/27 green since 2026-09-29; re-file if a red recurs); **6l** (pin bootstrap
trap — fixed: `pin-root.sh` refreshes the gate from the ref before deciding); **6u** (spawn-pin vs
fallbacks — fallback half built as D-FLEET-2, `49f18bdbd`; resolver-vs-hook half tracked in
`m-resolver-hook-disagree-on-docless-pick`); **7, 8** (profile restoration and OpenRouter repins —
premises measured on the fork's profiles and models; `models.yml` routing is re-checked by row 22);
**10, 11, 12** (Phase-0-gated port, registry reconciliation, re-baseline — superseded by adoption,
`D-MOTOKO-RESET-3`); **13, 14** (re-filed as row 25).

---
**Document created**: 2026-08-12 (rewritten from the 2026-06-24 charter). **Reset**: 2026-09-30 for
motoko main (`D-MOTOKO-RESET-1`); the fork-era queue, premise log and ITERATION 37 stamp are archived
verbatim in [motoko-mission-status-archive.md](motoko-mission-status-archive.md), below the heading
"Archived 2026-09-30: fork-era charter (ABI 2.2 → motoko main migration)", and the 2026-06-24 charter
is further down the same file.
