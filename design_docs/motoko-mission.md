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
| D-MOTOKO-WORKDIR-2 | RESOLVED | **Yes — standing authorization granted.** Reconcile the source clone unattended only when all three recorded safety predicates hold; any failed predicate still requires a new human decision. | Mark answered `MOTOKO-WORKDIR-2: Yes` on bookkeeping issue `#850` at `2026-08-29T09:09:20Z`. Iteration 28 re-measured **0 ahead / 0 dirty / 292 behind**, so no backup payload existed; `git checkout -B dev origin/dev` advanced `e3ed9467f` → `bd0bb157d` and post-verified **0 dirty**. |
| D-MOTOKO-6N-1 | RESOLVED | **Ship the measured minimal fix for the discovery arm, or hold for a race-free construction?** The arm at `test_motoko_connection_probe.sh:449` cannot fail for the reason it names — measured: neuter the wall clock and the suite stays 41/41 green. The minimal fix (three distinct refusal messages + the arm asserting the wall-clock one) is measured to work: clean 41/41 in 50s, and with the wall-clock mutant rc=1 in 44s on the exact arm. The design quorum BLOCKED it 3/3 in both rounds on ONE surface: the fix converts a silent false PASS into a possible loud false RED if the node ceiling wins the race on the CI host. Local margin, corrected after the evaluator caught me benchmarking the wrong binary: **~6-9x**, not the ~52x I first reported. **(A) SHIP IT** — the arm becomes honest now; risk is a flake on the CI host at an unmeasured rate, visible immediately as a red on the next PR. **(B) HOLD for D4** — the evaluator's synthesis: scope `PROBE_MAX_TREE_NODES` to that one arm's `env` line, measured free on the happy path (41/41, 47.1s), which removes the race structurally rather than betting on it; costs one more iteration and a third quorum round. **(C) NEITHER** — leave the arm vacuous and de-prioritise. **Loop's recommendation: (B)** — the reviewers were right twice for the same reason, and D4 is the thing all three were circling; it is one iteration and it ends the argument instead of deferring it to a CI leg. **Default if unanswered by 2026-09-08: (B)**, taken as a normal queue pick under row 6p.  **ANSWERED — (B) HOLD for D4. Scope PROBE_MAX_TREE_NODES to that one arm's env line and remove the race structurally rather than betting on a 6-9x margin holding on a CI host nobody has measured. The reviewers blocked twice for the same reason and D4 is what all three were circling; one more iteration and a third quorum round is the right price for ending the argument instead of deferring it to a red CI leg. Take it as a normal queue pick under row 6p, as the loop proposed.** (Mark Edmondson, attended 2026-09-01, recorded directly in this ledger.)| Design doc `design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md` (PARKED). Quorum artifacts `m-motoko-discovery-arm-discriminating-refusal-2026-09-01T00-35-20Z.json` and `-2026-09-01T00-43-50Z.json`, both BLOCKED, 3/3 external present, no absentees. Evaluator PASS 78/100, 1 blocking (the wrong-binary benchmark, corrected in-doc). Filed by motoko iteration 31.  **Attended ruling 2026-09-01** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author — an attended identity, which the fleet bot does not hold and the loop may not author with. Attended ruling, matching the loop recommendation and the evaluator's D4 synthesis (measured free on the happy path: 41/41, 47.1s).|
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

## STATUS 2026-09-06 — ITERATION 36 COMPLETE: **AN ATTENDED SESSION TURNED `dev` RED IN SIX CHECKS AND LEFT; THE FIRE 37 MINUTES LATER INHERITED IT, AND FOUND A FIFTH DEFECT NO CI LOG COULD YET NAME.** Pick was NOT the queue head — Gate 1's red-outranks-the-queue rule fired, and rule 3d's parent-walk attributed all six reds to M-MISSION-LOOP-WORKBENCH Phase 1: `ab3252109` **16 checks all green** → `a9de67fe6` M1 **20 checks, windows pair red** → `6536cfb98` tip **adds `lint`, `docs-build`, `docs-gate`, `launchd drivers`**; the intra-push commits read `total=0` by construction, since only a push's tip gets a run. **IT WAS NOT AN ORPHANED ITERATION AND THE DIED-MID-FLIGHT TRACES ARE WHAT PROVED IT** — the driver log records exactly two motoko fires on 09-05 (13:01→16:21 = iteration 35; 23:39 = this one), so nothing fired at 22:39; the doc, the plan and M1–M4 were landed by an **attended session** between 22:15 and 23:02 local with **zero charter rows and zero log entries** (`grep -ci workbench` **0** in both files; control `ITERATION 35` **4** / `ITERATION 34` **5**). It was **still working while this ran**: M5/M6/M7 (`cad2ecbdb`, `096d04020`, `7a0bbd5da`) landed mid-flight, so the branch was rebased and every defect **re-verified against fresh origin before proceeding** — all four still present (1/1/1/1), CI still red the same six ways. **THE OWNING-MISSION RULE DID NOT SEND THIS TO V1**: its own exception applies twice over — mission-loop machinery is this charter's clause 6. **FIVE DEFECTS, AND THE FIFTH IS THE ONE THAT MATTERS.** M1 `lint`: a map literal at `doctor_test.go:229` overwritten by `walk()` before any read. M2 `test-windows`/`Build windows-latest`: `registry.go:121` validated the workdir with `filepath.IsAbs`, which is **platform-dependent** — on Windows it rejects `/Users/…`, so every fixture failed validation and ~20 tests died at the gate. M3 `launchd drivers (bash 3.2)`: `test-launchd-drivers` called `test-mission-registry`, which runs `go test`, inside a CI job that installs no Go by design. M4 `docs-build`/`docs-gate`: a repo-relative link escaping the docs tree. **M5 — `GOOS=windows go vet` on the REBASED tree returned `kill_unix.go:6:51: undefined: syscall.Kill`**: `kill_unix.go` had landed minutes earlier with **no build constraint and no windows counterpart**, and Go does **not** treat `_unix` as a GOOS suffix — so the package did not COMPILE on windows, and **M2 alone would have left both windows checks red while looking like it had fixed them.** The stub reports the pid **live**, because `missionBusy` reads a non-nil error as *not busy* — the unsafe direction, which would let `apply` overwrite a running mission's artifacts; it fails closed, matching the design's own "refuse on a live pidfile" decision. Mutation drill: stripping `//go:build unix` takes windows back to red, restore byte-identical by sha256. **BASELINE-PAIRED VERIFICATION ON THE SAME MACHINE, SAME COMMANDS** (rule 3e): `make lint` baseline **1 issue (ineffassign)** → treatment **0**; `go test ./internal/mission/...` `ok` both sides; `GOOS=windows go vet` baseline **`undefined: syscall.Kill`** → treatment **rc=0**, linux and darwin rc=0 too; `make test-launchd-drivers` with `go` masked off `PATH` **rc=0**, `PASS: 57 probe self-test arms ran`, control `command -v go` rc=1 so `go` really was absent. Commits were reconstructed from the executor's per-milestone snapshots and proved **byte-identical to its final tree** (`shasum -c` 5/5), `.snap/` absent from every commit, auto-close keyword scan clean with its known-bad control firing. **A PRE-EXISTING RED FIXED ITSELF UNDER ME, AND ONLY THE RE-BASELINE CAUGHT IT**: `TestLive_DoctorReproducesTheMeasuredDivergences` was **red on the pristine tree** at `f5edd569a` with failure text identical on treatment and baseline (correctly ruled out of scope); after the rebase onto `7a0bbd5da` it is **green at baseline** — M6 repaired the rig drift the gate asserts. The first reading was right when taken and wrong forty minutes later; it skips off-rig (`golden_live_test.go:330`), so CI never saw either state. **EVALUATOR PASS 85/100, ZERO BLOCKING** (sonnet, own worktree, distinct provider from the codex executor). It drilled causally rather than plausibly — **built the docs**, reverted M4's line, recovered the exact `Docusaurus found broken links` failure, restored; reintroduced M1's initialiser for exactly 1 `ineffassign`. It independently found M3's echoed comment, which the controller had already fixed as **M3a** (two instruments agreeing). **ITS STRONGEST OBJECTION WAS RIGHT AND IT WAS ABOUT A COMMENT, WHICH IS WHY IT NEARLY SURVIVED**: M2's new test claimed to "kill both mutations" and does not — reproduced first-party, reverting to `filepath.IsAbs` leaves **both arms green on darwin**, because on unix `filepath.IsAbs` **is** `strings.HasPrefix(p, "/")`, the same shipped function (`path_unix.go:35-37`), so no POSIX input can distinguish them; only `test-windows` and `Build windows-latest` catch that revert, and a contributor trusting a green local run would ship the very regression the test exists to prevent. Fixed in **M6** with the platform qualifier stated, together with the judge's second find — `test-mission-registry` orphaned by M3, now documented as a deliberate hand-run entry point rather than deleted, this repo having scar tissue about removing things that merely look unused. **ONE JUDGE FINDING NOT ACTIONED, WITH THE CONTROL THAT DECIDES IT**: the missing `CHANGELOG.md` entry — `git log --name-only f5edd569a~1..origin/dev -- changelogs/ CHANGELOG.md` is **empty**, so the seven-commit landing this fixes added none either, and the changelog is sectioned by released version (top entry `v0.35.1`, shipped); a new unreleased section for the fix-forward alone would misdescribe both. `make check-changelog` is index hygiene, not a per-PR gate, and is green. Filed as a queue row. Routing: controller `claude:claude-opus-5`; executor `codex:gpt-5.6-sol` via the cross-provider recipe (probe rc=0 with a real `tokens used` artifact, real run rc=0, non-empty worktree diff, `-o` final message **7,110 B**); evaluator **sonnet** via the Agent tool in its own worktree. **No designer and no planner ran, and that is the routing table APPLYING rather than being skipped** — both branches gate on artifacts a CI-red fix-forward does not have; `derive-planner-lane.sh` returned `opus fail-closed:no-doc`, and the planner's own pin is `codex:gpt-5.6-sol`, a `provider:model` value for which the spawn-pin hook would have denied an opus spawn. Rotation pointer untouched at `codex:gpt-6-astra`; **Fable unspent**; metered **$0.00** of $5 (no quorum, no design doc). No GPU, no `rig.lock`. Gate 0: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, **0** human directives on `#987` since the watermark, ledger valid at **6** rows **0 OPEN**, 14 unread on the canonical store and **0** of them a motoko directive. Blocked-row predicates re-run as commands: upstream `#154` still **OPEN** (control `#175` **MERGED**, negative control 404s), so rows 10/11/12 stay Phase-0 parked. Gate-5: **NO skill edit** — every friction here was a rule that already exists and FIRED. **CORRECTION (the base moved 3x mid-iteration): the attended session kept landing — M8, M9, HD ratifications, Phase 3 part 1 — so `git rebase` DROPPED M1 and M4 as *already upstream* (the human fixed the initialiser and the docs link himself; both greps now **0**, while M2/M3/M5 still read **1/1/0** and remain ours), and a SEVENTH defect arrived in the base: `TestDriverCopiesDoNotMultiply` counts driver copies across sibling checkouts that exist only on the rig, so CI sees `distinct=1` against `knownDriverCopies=2` and the FELL arm fires for an ENVIRONMENT — dev's own head `19d6b03c7` is red on `test` for exactly this, first-party. Fixed as **M7**, down-arm skips when no sibling is present (controls: from the real clone all three are VISIBLE, world DIFFERS and has no `lib/pin-root.sh`, so `distinct=2` and the ratchet still holds; from a worktree or CI `observed=0`). And the `internal/smt` red is NOT ours — alone on the pristine tree it PASSES, but under the MATCHED `go test ./...` it fails identically **9.02s vs 9.01s**, load-dependent and pre-existing; the wrong-scope control would have licensed blaming the branch. **SECOND CORRECTION — six red checks became NINE defects, in LAYERS, and one of them was mine.** M5 unblocked the windows build for the first time and the next layer was **23** `internal/mission` failures, every one reading `workdir "C:\Users\RUNNER~1\…" must be absolute` — **M2's own check refusing a genuinely absolute path**. The two fixture families disagree by construction: `registry_test.go` uses POSIX literals, the fleet fixtures use `t.TempDir()` (`C:\Users\…` on a runner), so `filepath.IsAbs` alone rejects the first and a POSIX-only check — what M2 shipped — rejects the second. **The original defect and my fix were the same mistake pointing opposite ways.** M8 accepts either arm (identical on unix, so the rig sees no change) and retargets the test at the invariant — a RELATIVE path is refused everywhere — mutation-checked on darwin, which M2's version never was. That took windows 23 → **1**: `TestApply_BacksUpWhatItReplaces`, where a hand-rolled `baseName` splitting on `"/"` alone returned a backslash path WHOLE and produced `…\bak\C:\Users\…` — a second drive letter mid-path. **M9** uses `filepath.Base`/`Join` and adds the invariant that would have caught it. **RESULT: all six originally-red checks are `completed/success` on `7ec4c178a` — `lint`, `test-windows`, `Build windows-latest`, `docs-build`, `docs-gate`, `launchd drivers (bash 3.2)`.** Three lessons, and the third is the one that costs: **(i)** a build break hides every defect behind it, so the honest expectation after fixing one is MORE red, not less; **(ii)** the base moved under this iteration three times across five pushes — two of my commits were dropped as already-upstream and two of my defects came from commits that did not exist when it started; **(iii)** I shipped M2 with a green local suite, a PASS 85/100 evaluation and my own out-of-sandbox sweep, and it was still wrong, because every one of those instruments runs on darwin where the bug is invisible — the judge SAID SO in its strongest objection and I fixed the comment rather than hearing it as a statement about the change.** Next: row **6p M2/M3**.

## STATUS 2026-09-05 — ITERATION 35 COMPLETE: **THE ABSENT REVIEWER WAS THE ONE THAT FOUND THE DEFECT AT THE BOTTOM OF THE DESIGN, AND THE JUDGE FOUND ITS TWIN IN THE CODE.** Pick was row **6p**, named by iteration 34's Next — derive the suite's wall-clock and node-ceiling bounds from a stimulus measured in-test. **LANDED: PR [#1048](https://github.com/sunholo-data/ailang/pull/1048), four commits, `M1 of 3`.** M2 (wire the wall-clock class, enforce the floor, gate `p_obs`) and M3 (derive the node ceiling) are NOT in it — the executor was killed by its 30-minute cap after M1, and the honest scope is one milestone, not three. **PREMISES RE-VERIFIED AT HEAD BEFORE ANY ROUTING:** `MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}` probe:126 checked at probe:196, `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}` test:9, and **no bound-derivation helper existed** — `grep -cE 'derive_bound|measure_stimulus|calibrat'` **0** with the known-positive control `run_bounded` **10**. **THE DESIGNER LANE FAILED ON ITS FIRST REAL RUN AND THE WRAPPER SAID rc=0.** `codex:gpt-6-astra` — the rotation entry that landed THIS MORNING (`087fbea63`) — probed rc=0, then produced **zero artifact**: no `-o` final message, no worktree change, no `tokens used` summary (the probe's log has one; control), dead after ~3 minutes against a 30-minute cap. Read as an ARTIFACT rather than an exit code, per the recipe's own false-green list. Fell back to `MISSION_DESIGNER_FALLBACK` = fable, **FLAGGED**. **AND THE FALLBACK COULD NOT BE SPAWNED THE DOCUMENTED WAY:** the spawn-pin hook DENIED `Agent(model="fable")` with `deny:provider-pin — designer is pinned to codex:gpt-6-astra`. The hook enforces the DECLARED pin and has no notion of a fallback, so a role whose primary lane dies cannot be re-routed through the Agent tool at all; the fable run went through the `claude:` recipe via `claude-sub` (billing guard: tripwire CLEAN, keys stripped). **THE SAME HOOK CONTRADICTED `resolve-role-spawn.sh` FOR THE PLANNER**: the resolver returned `agent-tool opus fail-closed:planner-lane-field-missing` (and `derive-planner-lane.sh` agreed, run from the worktree that holds the doc — iteration 33's scope trap avoided), while the hook returned `planner is pinned to codex:gpt-5.6-sol`. Two enforcement mechanisms, opposite answers, and the hook wins because it sits at the tool boundary. The opus lane being unavailable is a LANE failure, not a judgment call, so the role took its configured `MISSION_PLANNER_ANTHROPIC_FALLBACK` = `codex:gpt-5.6-sol`, **FLAGGED**. **QUORUM BLOCKED BOTH ROUNDS.** R1 **3/3 reject**, `.synthesis.absent_reviewers` `[]` (cross-checked two ways, control `has("synthesis")` true), $0.1587. Two of the three named ONE defect — §4.2's prototype unconditionally enforced the floor that M1 requires be flag-disabled — applied as written. The third, `gpt5-6-sol`'s, was a PREMISE objection, so I **MEASURED it instead of forwarding it** (rule 3f): four stimuli under one identical load step degrade **bash-script 1.27× · date 2.04× · pgrep 1.13× · true 1.31×**, i.e. spread up to **1.8×**, and real `pgrep` runs at **76/s** against the stimulus's **564/s**. **UPHELD** — directionally right, not tight — and the designer got the measurement, not the objection. R2: `gemini-3-1-pro` **flipped to pass**, `oc-glm-5-2` reject, and **`gpt6-astra` recorded ABSENT (budget)** — the self-selecting degrade, since a reviewer drops on budget exactly when the doc has GROWN (527 → 774 lines). Re-run ALONE at a raised cap ($0.2654) it **REJECTED**, so the synthesis was a pass-with-a-named-hole and R2 was really **1 pass / 2 reject, 3/3 present**. **ASTRA'S OBJECTION IS THE SHARPEST EITHER ROUND PRODUCED AND IT WOULD HAVE BEEN LOST**: `measure_fork_rate` incremented its counter unconditionally after `|| true`, so a missing or failing stimulus produced a **positive rate** that would then have determined every derived bound — a silent fallback on the single input the whole design rests on, contradicting the doc's own §4.5. Both survivors carried concrete reviewer-authored `proposed_fix` text and disputed no design DIRECTION, so the **narrow-refinement carve-out** applied and the **CONTROLLER** applied their own text VERBATIM (§4.8), keeping the **Fable diet intact** at one doc / one authoring run / one revision run. Surfaces per round: R1 = prototype-flag consistency ×2 + proxy premise; R2 = proxy gating + measurement-helper error propagation — **spread and MOVING, neither R2 surface being an R1 surface**, with one reviewer flipping to pass: a maturing doc, **not** a SPLIT signal. **THE PLANNER REFUTED THE DESIGN DOC ELEVEN TIMES AND I CONFIRMED THE LOAD-BEARING ONE FIRST-PARTY.** The doc placed new arms *"after line 793"*; arms in fact run to **818** (`run_lane_fixture_arm` at 798, `REAL_LSOF` containment at 813–818), so the doc's placement would have inserted them AHEAD of bounded arms — precisely the change that took the wall-clock arms from **0-in-17** to **4-in-19** at iteration 33. The plan moves them after 821. It also caught two execution-order defects (`bound_secs` unreachable from pre-derivation EXIT paths; `ARM_CAP_SECS` derived at line 9 would call it before definition), that the startup insertion point **cannot** measure `live_bin/pgrep` because that file is created much later, that M1's `46 + 8` arithmetic ignored AC-8's three arms, and that the base is **59 s**, not 57. **THE EXECUTOR WAS KILLED BY ITS OWN 30-MINUTE CAP AND THE WRAPPER STILL REPORTED rc=0** — the recipe's false-green, second consecutive iteration, and **no snapshots were written**, so the uncommitted worktree diff was the only artifact. Completeness was ESTABLISHED rather than assumed: `bash -n` clean, zero bash-4+ constructs (control `run_bounded` **19**), `ARM_CAP_BASE=120` still feeding `ARM_CAP_SECS` unchanged and `$(bound_secs ` consumed **0** times, so M1 is genuinely ADDITIVE; the node ceiling is still literal at test:687 and the `p_obs` gate is absent, which is exactly the M1/M2/M3 split the plan specifies. Suite **rc=0, 57 ok, 0 not ok** against a base of 46. The published diag line reads `r=489/s r_real=665/s p_obs=1.36 ... scale=1 arm_cap=120s node_ceiling=7824 floor=DISABLED` with the bookend `drift=none` — and **`p_obs=1.36` independently reproduces the designer's interleaved 1.35×**, which is the design's central claim measuring itself. **THE EVALUATOR'S ONE BLOCKING FINDING IS REAL, AND IT FOUND IT WITH OUR OWN PLAN.** Evaluator **PASS 81/100** (sonnet, its own worktree at `cef2dae4b`, distinct from the codex executor and the opus controller → generator≠judge holds). `PROBE_SELFTEST_FORK_RATE` **short-circuits** the measurement rather than steering it, and was not cleared before the three AC-8 recursions, so an ambient value makes `measure_rate_or_refuse` unreachable and the injected fault is never exercised. The judge found it by running the sprint plan's **own M2 AC-1 boundary command**, which therefore already failed against shipped M1 for a reason unrelated to M2. **REPRODUCED FIRST-PARTY BEFORE BEING BELIEVED**: `PROBE_SELFTEST_FORK_RATE=200 <suite>` → **rc=1, 53 ok**, arm `bound measurement refuses a stimulus that exits nonzero`, against the no-override control **rc=0, 57 ok**. Fixed with `env -u` on that recursion (not a leak guard — the overrides are legitimate at suite scope for every other arm), and verified in BOTH directions: **rc=0, 57 ok** with the override, control unchanged at **rc=0, 57 ok**. The judge also ran the astra mutant and it **REDS** (`expected_rc=72 refusal=0 derived=1`), so that gate LOOKS. **ITS TWO NON-BLOCKING FINDINGS ARE CORROBORATION, NOT NEW DEFECTS**: a duplicated diagnostic line and an ADDED arm both leave the suite green — the second is row **6s**, filed before this sprint by iteration 33's judge, now independently reproduced. Gate 0/1: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, invalid-store control REFUSED (`unknown message store mode "not-a-real-store"`) so the inbox read is real (`store: gcp (Firestore, project ailang-multivac)`) — **30 unread, none a motoko directive, none outranking; no `mission-*` and no `github-untrusted:` senders**. **0** human directives on `#987` since the watermark `2026-08-29T09:09:20Z`, known-positive control firing (a widened `#850` read returns exactly `MOTOKO-WORKDIR-2: Yes`) and a negative control returning 0. Ledger valid at **6** rows, **0 OPEN**; no row flipped, so no attended-provenance check was owed. No rotation due (`#987` created 2026-08-31T10:45:34Z, 8 comments against a cap of 80); weekly sweep not due. Blocked-row predicates re-run **as commands**: upstream `#154` still **OPEN** (control `#175` **MERGED**, negative control 404s), so rows 10/11/12 stay Phase-0 parked. **RUNNING-SKILL CHECK ON THE RESOLVED SYMLINK: byte-identical to `origin/dev`** — `~/.claude/skills/mission-control` resolves to V1's main checkout (inode `67997727`), `cmp` rc=0; the pin copy is a different inode (`68008222`) and also matches. **dev's own head carries ONE non-green check, `SonarCloud Code Analysis: failure`, and it is INHERITED, not mine** — `failure` on all five most recent commits walked back; per the charter guardrail **V1 owns dev CI red on this anchor**: recorded, handed over, pick kept. The **source clone is 237 commits behind** (`AILANG_DRIVER_SRC`, not the pin — the pin is at origin/dev by construction), up from 121 at iteration 34; `D-MOTOKO-WORKDIR-2`'s standing authorisation still covers the reconcile when its three predicates are re-measured, and the growth rate is now the story. The fleet-shared bare `design_docs/mission-dashboard.md` was **left alone**; only the namespaced `motoko-mission-dashboard.md` was written. Routing: controller `claude:claude-opus-5`; designer **`fable`** via the `claude:` recipe after astra's lane failure; planner **`codex:gpt-5.6-sol`** after the hook denied the resolver's opus; executor **`codex:gpt-5.6-sol`**; evaluator **`sonnet`** in its own worktree. **Metered $0.5256** of the $5 ceiling (two quorum rounds + one absentee re-run). No GPU, no `rig.lock`. All local evidence is **darwin/arm64, GNU bash 3.2.57**; ubuntu and windows legs read from Gate 3b only. Auto-close scan on all four commit messages and the PR body: **0** matches with a known-bad control firing. **GATE 3b GREEN ON THE MERGE — AND THE RUNNER ANSWERED THE ONE QUESTION NOBODY COULD:** `0686d5b00`, **21** checks, **0** pending, **0** non-green, required **4/4** (`test`/`lint`/`build`/`docs-gate`), and **`launchd drivers (bash 3.2): success`** — the only leg where this suite runs. Rebase-merged as `b4bfb04e1`..`137842bfd`, four commits kept so the boundaries stay bisectable. **The derivation EXECUTED on the GitHub runner and published its own diag line: `r=318/s r_real=251/s p_obs=1.27 reference=400/s scale=2 arm_cap=240s node_ceiling=5088 floor=DISABLED`.** That is **the first measurement of the CI runner row 6j has ever had**, and it is exactly what M1 AC-4 exists to produce. Three things fall out of it, all of which M2 needed and none of which anyone could previously state: (1) the runner is genuinely slower — **318/s against this host's 489–616/s** — and derives **scale=2**, i.e. the arm cap it wants is **240s**, so **the hardcoded 120s was too tight for the runner**, which is row 6j's >100x arm-cap blowout with a number attached at last; (2) **`p_obs=1.27` on the runner**, inside the budgeted `P_PROXY=2` and far inside the 4.7 tolerance — `oc-glm-5-2`'s objection was precisely *"no CI runner has been measured"*, and it now is; (3) `floor=DISABLED` with **no** `BOUND_FLOOR_NOT_ENFORCED` line, since 318 > the 100/s floor, so **M2's floor flip is measured-safe** rather than hoped-safe. Note one instrument failure of my own, caught by its control: a grep for the new arms BY NAME in the CI log returned **0** and its known-positive control returned **0 too**, so the grep was broken rather than the arms absent (rule 3a) — the load-bearing evidence is the `PASS: 57 probe self-test arms ran` line in that job's log, present, with a shape-matched negative control at 0. Auto-close scan on all four commit messages and the PR body: **0** matches, known-bad control firing; `closingIssuesReferences` **0**; `#987` still **OPEN** at **8** comments, its pre-merge count. Gate-5: **NO skill edit** — the two spawn-pin defects are a HARNESS bug in `resolve-role-spawn.sh`/the hook, not a rulebook gap, and are filed as row **6u**. Next: row **6p M2/M3**, which are specified, unblocked, and short of nothing but executor minutes.

## STATUS 2026-09-03 — ITERATION 34 COMPLETE: **THE WORK WAS DONE, VERIFIED AND JUDGED; IT COULD NOT MERGE, AND NEITHER RED WAS OURS.** Pick was the **queue head, row 6o**, named by iteration 33's Next — the group SIGKILL escalation with no killer, and `REAL_LSOF` containment narrower than the code claimed. **BOTH PREMISES RE-VERIFIED AT HEAD BEFORE ANY ROUTING** (`kill -9 "-$pid"` probe:261 beside group `-TERM` probe:252; `command -p -v lsof` test:16 with a `case` checking only absolute-and-executable; controls firing at 2 and 0). **NOT LANDED — PR [#1034](https://github.com/sunholo-data/ailang/pull/1034), five commits, rebased onto `267a94e92`, head `db1996128`, open and resume-ready.** **Evaluator PASS 96/100, ZERO blocking** (sonnet, its own worktree at `a7b0002a3`, distinct from the codex executor and the opus controller → generator≠judge holds); it re-derived every controller number, ran all four required drills AND six bonus drills including **T12/T13/T14, which had never been run by anyone**, and confirmed the headline mutant (probe:261 alone → single-PID) now reds with the new arm the **SOLE** `not ok` where at base it SURVIVED. Local verification is complete and repeated: base **rc=0, 43 ok** → head **rc=0, 46 ok, 0 not ok** on **4** runs (62/62/58s, 59s post-rebase); green at **every milestone boundary 43 → 44 → 46**, so bisectability is measured not hoped; production probe byte-unchanged (`f0b5e024…aabc99`, control discriminates); snapshot reconstruction proven byte-identical by `shasum -c` rc=0. **GATE 3b NOT GREEN, AND THE ATTRIBUTION IS THE DELIVERABLE.** `mergeable` was read FIRST and returned `CONFLICTING/DIRTY` on a changelog collision — the boring cause, one call, before any dropped-event lever — resolved by rebase + force-push, after which all **20** checks completed. Required are `test`/`lint`/`build`/`docs-gate`; `docs-gate` passes, `build` skips, and **`test` and `lint` both fail ON DEV'S OWN HEAD at the identical steps**. Two independent causes, neither reachable by this diff (**0 Go files changed**): (1) `c8c841e24` deleted the `FMT_AB_TESTABLE_FUNCTIONS` markers from `tools/launchd/nightly-eval.sh` — red on **19+ consecutive commits**, green at `115184a2e` 63 back; V1's `#1030` covers it and is now MERGEABLE; (2) **NEW TODAY and NOT covered by #1030** — `lint` fails `make fmt` on **seven Go files** from the coordinator work, a REQUIRED check, blocking every PR in the repo. Per the charter guardrail **V1 owns dev CI red on this anchor**: recorded, handed over (delivery ASSERTED by reading the message back, not by the exit code), pick kept. Auto-merge deliberately NOT armed — it is a prediction, and this gate's discipline is that a prediction is not an observation. **THE SCOPE STATEMENT THAT COST THE ITERATION IS WIDER THAN A RED CHECK:** the `launchd drivers (bash 3.2)` target dies at the fmt_ab script **before reaching the probe suite**, so **the new arms have never executed in CI at all** (grep for their names in that job's log = **0**, control firing). That leg is not merely red, it is **BLIND** — nothing downstream of `test_fmt_ab_schedule.sh` is exercised for anyone. All evidence here is **darwin/arm64, GNU bash 3.2.57, local**; ubuntu and windows legs unrun and unreadable. That is the honest scope of a 96/100, and it is why row 6o stays OPEN. **QUORUM BLOCKED BOTH ROUNDS AND `gpt5-6-sol` WAS RECORDED ABSENT (budget) IN BOTH** — the self-selecting degrade, since a reviewer drops on budget exactly when the doc has GROWN. Re-run alone at a raised cap both times, it **REJECTED both times**, so each synthesis was hiding a real reject and the quorum was **3/3 present**, never N−1. Every premise was MEASURED, not forwarded: R1's sha256 objection **substantively REFUTED** (hash exact, control discriminates) and procedurally right; R1's "the re-exec arms are unbounded" **REFUTED BY CONSTRUCTION** (`expect_failure` routes every arm through `run_bounded`/`ARM_CAP_SECS` test:9 with a TERM→KILL escalation, and a `$0` precedent exists at test:721) — the DOC was at fault, not the design, and the real residual (**2.07×** margin on a leaked inner run against a 120s cap, on a host whose load moves a comparable stimulus 3.3–3.6×) was closed to ~**5,000×**; R2's were **attribution** (four cited boundaries — measured, **all four correct, no drift**) and **determinism** (`getconf` outside `run_bounded` — **UPHELD**: `run_bounded` is defined at test:88 and the gate sat at test:28, so it *could not* have called it). Both R2 objections carried concrete reviewer-authored `proposed_fix` text and disputed no design DIRECTION, so the **narrow-refinement carve-out** applied and the **CONTROLLER** applied the reviewers' own text VERBATIM — which is what keeps the **Fable diet intact** at one doc / one authoring run / one revision run. Surfaces tracked per round (R1 provenance + harness semantics; R2 provenance + bounded-waits): **spread and shrinking**, one reviewer passing both rounds — a maturing doc, **not** a SPLIT signal. **THE PLANNER REFUTED THE DESIGN DOC TWICE AND WAS RIGHT BOTH TIMES:** `AC11` demanded `grep '/usr/bin/getconf PATH'` = **1 hit** where the relocated gate contains it **4** times — it **would have failed a correct implementation** — and the Overview still placed the gate at test:28. **Both were defects in text I had just written under the carve-out**, reproduced first-party before fixing. **THE EXECUTOR WAS KILLED BY ITS OWN 30-MINUTE CAP AND THE WRAPPER STILL REPORTED `rc=0`** — a true statement about a dead process, the recipe's own false-green, caught by reading ARTIFACTS (no `.snap/M4`, no `-o` file) rather than the code; completeness was then ESTABLISHED, not assumed, because `.snap/M3` is byte-identical to the final test file so M4 had touched only the changelog. **THE EVALUATOR'S T6 FINDING IS AGAINST THE DOC AND IT IS CORRECT:** inverting the containment comparison does NOT refuse at startup — the loop is a DISJUNCTION, so `!=` is satisfied by 3 of this host's 4 `getconf PATH` entries and the gate is silently **defeated**, not tripped; reproduced by me and corrected in the doc before promotion. Gate 0/1: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, inbox read from `store: gcp (Firestore, project ailang-multivac)` — **16 unread, none a motoko directive, none outranking**. **0** human directives on `#987` since the watermark `2026-08-29T09:09:20Z`, with the known-positive control firing (a widened read of `#850` returns exactly the `MOTOKO-WORKDIR-2: Yes`). Ledger valid at **6** rows, **0 OPEN**; no row flipped since the watermark, so no attended-provenance check was owed. No rotation due (`#987` created 2026-08-31T10:45:34Z, 7 comments against a cap of 80); weekly sweep not due. Blocked-row predicates re-run **as commands**: upstream `#154` still **OPEN** (control `#175` **MERGED**, negative control 404s), so rows 10/11/12 stay Phase-0 parked. **RUNNING-SKILL CHECK ON THE RESOLVED SYMLINK: byte-identical to `origin/dev`** — `~/.claude/skills/mission-control` resolves to V1's main checkout (inode `64996938`), **2,854** lines, `cmp` rc=0 against origin, and this fire's pin copy (inode `66039351`) matches too; the 223-line drift iteration 33 recorded is CLOSED. The **source clone is 121 commits behind** (`AILANG_DRIVER_SRC`, not the pin) — not reconciled this iteration, and `D-MOTOKO-WORKDIR-2`'s standing authorisation still covers it when the three predicates are re-measured. The fleet-shared bare `design_docs/mission-dashboard.md` holds a sibling's content and was **left alone**; only the namespaced `motoko-mission-dashboard.md` was written. Routing: controller `claude:claude-opus-5`; designer **`fable`** (rotation entry after the pointer's deepseek — the Agent-tool `model="fable"` pin was ACCEPTED and ran to completion twice, corroborating the 2026-08-20 correction first-party; pointer advanced); planner **`opus`** (`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`, VERBATIM, no codex probe fired); executor **`codex:gpt-5.6-sol`**; evaluator **`sonnet`** in its own worktree. **Metered $0.50** of the $5 ceiling. No GPU, no `rig.lock`. Gate-5: **NO skill edit** — every friction was a rule the rulebook already carries and that FIRED. **CORRECTION, SAME ITERATION — IT LANDED.** The record above was written when the merge was blocked, and is kept because the reversal is the point. ~90 minutes after the hand-over V1 landed `b51e53f78`, which covers **both** causes I reported — **including the `lint` red its own in-flight `#1030` did not** — and three more nobody had seen. The branch was rebased onto the fixed base, re-verified (**rc=0, 46 ok**, probe hash unchanged), and **rebase-merged as `b97cbf83c`..`684ab8331`** (five commits kept, so the milestone boundaries stay bisectable). **GATE 3b GREEN: 21 checks, 0 non-green, required 4/4, and `launchd drivers (bash 3.2): success`** — with the three new arms **executing in CI** on the macOS runner (`ok 43` SIGKILL escalation, `ok 44`/`ok 45` containment, `PASS: 46 probe self-test arms ran`; control: a pre-existing arm in the same log). **So the BLIND-leg scope caveat is RETIRED, not merely improved.** Auto-close scan on all five commit messages and the PR body: **0** matches with a known-bad control firing; `closingIssuesReferences` **0**; `#987` still OPEN at **7** comments, its pre-merge count. Next: row **6p** — derive the suite's wall-clock and node-ceiling bounds from a stimulus measured in-test, so the ratio holds by construction on any machine.



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

6f. [**LANDED 2026-08-25 · iteration 23 — both motoko-owned issues triaged: `#839` CLOSED as a version
   skew (fixed 16 days before it was filed), `#842` CONFIRMED REAL at HEAD and re-filed as row 6h.**
   Historical tag kept as the trail: NEXT — batched from the weekly external-issue sweep, 2026-08-24 iteration 21]
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
   **VERDICTS (iteration 23, first-party at `02bf43668`).** `#839` — **GHOST, closed.** The reported build
   `v0.33.0`/`ae36986` is dated 2026-08-04; the request-aware proxy transport landed 2026-08-20 in
   `e5ee6c5e5` (PR `#613`). Measured, not inferred: `git ls-tree ae36986 internal/effects/net_proxy.go` is
   **empty** with control `internal/effects/context.go` returning a blob in the same call, and
   `git merge-base --is-ancestor e5ee6c5e5 ae36986` is **false**. Closed durably per the ghost rule — the
   behaviour its decisive third repro isolates is covered by committed tests CI runs (`ci.yml:101`
   `go test ./...`): `TestNetProxyBoundary/proxy_selected_from_environment` and
   `TestNetProxyTargetValidation/proxy_hostname_remains_unresolved`, plus three more; suite rc=**0** captured
   without a pipe, negative control `[no tests to run]`. Its lone `SKIP` was **checked, not counted**
   (rule 2): `TestNetProxyEnvProxyHelper` is a subprocess helper `TestNetProxyBoundary` re-execs with
   `AILANG_M1_PROXY_HELPER=1` and asserts `--- PASS:` on, and it is the only arm running production
   `http.ProxyFromEnvironment` rather than the injected hook. `#842` — **REAL, re-filed as row 6h.**
   **The six not-ours were handed to V1 at iteration 21 and are NOT re-triaged here; this row does not claim
   they were.**

6g. [**LANDED 2026-08-26 · iteration 24 · PR [#892](https://github.com/sunholo-data/ailang/pull/892) → [`fd1fa9e01`](https://github.com/sunholo-data/ailang/commit/fd1fa9e01) · evaluator round 1 **PASS 82/100, ZERO blocking** · Gate 3b GREEN on head `c23a8c785` (21 checks, 0 pending, 0 not-green, 4/4 required, `CLEAN`).**
   Guarded process-group kill via `set -m` in BOTH call sites; suite 39 → 40 arms. The judge reproduced all five
   controller claims and could not construct a false positive for the `group_safe` guard. **The production half is
   NOT pinned — see row 6i, filed from this iteration's own mutation drill rather than by growing the PR.**] **`run_bounded` and `run_lane` cap the wrapper PID, not the process GROUP — a hung
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

6h. [**LANDED 2026-08-28 · iteration 27 · PR [#946](https://github.com/sunholo-data/ailang/pull/946) → `75e4e12be` · evaluator round 1 **PASS 94/100, ZERO blocking** · Gate 3b GREEN on head `75e4e12be` (**21** checks, **0** pending, **0** not-green, **4/4** required — build/docs-gate/lint/test — `mergeable` read FIRST per the iteration-198 rule: `MERGEABLE`/`CLEAN`), merged as [`0e3a96585`](https://github.com/sunholo-data/ailang/commit/0e3a96585).** The guard was not missing, it was **inexpressible**: `Usage` is now `*ChatStepUsage`, so an omitted `usage` key (nil) and a present all-zero block are finally different values. Representation ONLY — the judge's own 16-case differential harness across the parent commit found byte-identical output for every input, including the `usage:null` and `usage:[]` cases the controller had named as untested. Step 2 (policy) stays deferred by design. Mutation drill anchored to the diff hunk by hunk: M2 is the **sole killer** of the no-behaviour-change arm (inverse arm `-skip` mine → rc=0), M1 is a kill-set member, M3 reds only a **pre-existing** test, and M4 — reverting to the value type — **does not build**, recorded as a compile-time arm rather than a behavioural kill. A zero-killer hunk the drill missed is filed as row **6m**.] *(original row text follows as the trail)* **A provider failure arrives as a successful empty completion, and the guard the reporter suggests is not expressible against the current type** — split out of 6f's triage on first-party evidence, per the rule that a confirmed defect a
   triage surfaces is a row of its own] **A provider failure arrives as a successful empty completion, and the
   guard the reporter suggests is not expressible against the current type** · loop health + the *never conclude
   model wall* guardrail · reported at [#842](https://github.com/sunholo-data/ailang/issues/842) by the
   `motoko_agent` account (internally authored — no `github-untrusted:` class). Confirmed at `02bf43668` by
   feeding the verbatim failing body to `openai.ParseChatStepResponse` (`internal/ai/openai/step.go:560`) with
   three controls in one run: the failing shape (`finish_reason:"stop"`, `content:null`, **no `usage` key**)
   returns **OK** — no error, `text=""`, `toolcalls=0`, `in=0 out=0 total=0`; the healthy control returns
   `"pong"` with real usage (instrument sees a positive); the legitimate-tool-call control (`content:null`,
   usage **present**) returns 1 tool call, which is what a *"null content is suspicious"* heuristic would
   false-positive on and is why keying on `usage` is right; and a fourth control — usage **present but
   all-zero** — is **byte-identical in output to the failing arm**.
   **The load-bearing find:** `ChatStepResponse.Usage` is a **value type** (`step.go:300`), so an absent
   `usage` key unmarshals to the zero struct — `absent.Usage == zeroed.Usage` is **true**. The suggested guard
   therefore cannot be written behind the present type at all.
   **Blast radius is wider than filed.** `ParseChatStepResponse` has exactly **two** production callers,
   `openrouter/step.go:162` and `openai/step.go:170` (controls: 77 hits for a common symbol, 0 for an invented
   one), and `internal/ai/ollama/step.go:345` constructs `openai.NewClient(…)` against `<endpoint>/v1` — so
   **every ollama tool-calling turn parses through it**, putting the defect on our own eval rig, where a masked
   provider failure is indistinguishable from a local model declining to act.
   **Scope, in this order:** (1) make absence **observable** — `*ChatStepUsage` or a raw-key presence check;
   pure representation, no behaviour change; (2) *then* decide the policy per provider and per
   streaming/non-streaming path. The reporter's own caveat is **upheld and is why (2) is not obvious**: the
   Anthropic and Gemini paths do not share this parser and streaming usage delivery is opt-in on some
   providers, so a uniform guard would turn a legitimate empty completion into an error — the same defect
   pointed the other way. Standing negative control for any guard: the legitimate tool call with `content:null`
   and usage present. `native_finish_reason` stays unread (gateway-specific; agreed with the reporter) ·
   1 iteration

6i. [**LANDED 2026-08-31 · iterations 29+30 · PR [#985](https://github.com/sunholo-data/ailang/pull/985) · evaluator round 1 **PASS 87/100, ZERO blocking**, two non-blocking findings BOTH reproduced first-party and filed as row **6o** · authored by iteration 29's codex executor, which died before opening the PR; iteration 30 inherited the branch, re-derived every load-bearing claim rather than transcribing it, and landed the missing design doc + sprint plan. **The row's own bar is met and the mutant is a SOLE KILLER:** reverting the production group kill to a single-PID kill takes the suite from rc=0/41 ok to rc=1 with arm 36 `production run_lane timeout kills wrapper grandchild` the ONLY `not ok` — `survivors=1 outer_cap_fired=no`, i.e. the red is the production timeout's cleanup, not the emergency containment. Before this row that same revert left the suite green 40/40. Historical tag kept as the trail: NEXT after 6h — filed by iteration 24's OWN mutation drill, per rule 3n: anchor the enumeration to the diff, not to the defect] **The production `run_lane` process-group kill is pinned by nothing — reverting the whole
   hunk leaves the suite green 40/40** · loop health · Row 6g asked for the group kill "in BOTH call sites", and
   both got it. Only one is *tested*. Mutant C, run by the controller and reproduced independently by the
   evaluator: revert `tools/eval/motoko_connection_probe.sh` entirely to its pre-fix version and
   `/bin/bash tools/eval/test_motoko_connection_probe.sh` still returns **rc=0 with 40 ok and 0 not ok** — zero
   killers. Its only gate is `bash -n`, which cannot tell a group kill from a single-PID kill. This is the half
   row 6g itself calls "the one that matters": the production lane bound runs on the GPU rig, where a surviving
   descendant is indistinguishable from a local model declining to act, which is the *never conclude model wall*
   guardrail arriving from inside our own instrument. **Scope:** give `run_lane` a behavioural harness of its own
   (the self-test's fixture-cwd + `lsof -c sleep` observable is the pattern that works, and it binds no socket, so
   unlike the rest of the suite it is satisfiable inside a sandbox), then prove it with the same two mutants that
   pin the self-test half — group→single-PID kill, and neutering `set -m`. A guard is not a gate until something
   reds when you remove it · 1 iteration

6j. [NEXT after 6h and 6i] **`launchd drivers (bash 3.2)` arm 33 hangs intermittently on the CI runner —
   iteration 22's cap now makes it a LOUD red, and the hang underneath it is still unfixed** · loop health ·
   Measured 2026-08-26 by enumerating the last **60** CI runs on `dev` and reading the
   `launchd drivers (bash 3.2)` job out of each (58 rows returned; the 2 missing are the two runs still
   in flight, so the enumeration is complete for settled runs): **55 success / 2 failure / 1 cancelled**.
   All three non-success rows are the same day and the same arm — `33001432738` 18:45Z and `32989036133`
   16:33Z both `not ok - descendant discovery refuses on the real wall-clock deadline exceeded its 120s arm
   cap`, and `32985981420` 15:43Z cancelled after **15m2s**, i.e. `timeout-minutes: 15` firing. All three
   landed on **unrelated V1 coordinator commits**, so this is not attributable to any motoko diff — it is
   the arm itself. **The cap is working exactly as row 6e designed it**: a silent 15-minute cancellation
   with zero diagnostics became a named `not ok` in ~3m15s. What the cap did NOT do is fix the hang, and
   iteration 22 said so at the time — *"the shipped arm's margin is unmeasured"*. Now it is bounded from
   one side: the arm completes in **~1.06s locally** and blows a **120s** cap on the runner, a >100x gap,
   which is rule 3m's shape exactly — a bound calibrated on the author's machine against a stimulus that
   scales with the machine. Scope: derive arm 33's bound from a stimulus measured in-test rather than from
   a wall-clock constant, or establish why the descendant walk stalls on the runner (it is the loopback
   sampling arm, which already reports `UNINFORMATIVE UNDER SANDBOX` on the step before). A per-arm
   `not ok` is not a fix; it is a legible failure · 1 iteration
   **NEW EVIDENCE 2026-09-02 (iteration 33), and it widens the row from one arm to a class.** Two MORE arms in this suite flake
   under host contention, measured locally rather than on the runner: `refusing live path refuses with the control-void message`
   (underlying `INSTRUMENT FAILURE: lane treatment exceeded 4s sampling deadline`) and `production run_lane fixture readiness
   failed (outer_rc=82)`. Rates from an interleaved three-way run at load average 39–46 on 16 CPUs, one probe, test file the only
   variable: base **0 reds in 17 runs**; the same suite with one extra arm inserted AHEAD of these two, **4 in 19**; with that arm
   placed AFTER them, **0 in 5**. So the fragility is not specific to arm 33, and it is sensitive to how much work precedes a
   bounded arm — the same rule-3m shape this row already names, now with three affected arms instead of one. The iteration-33
   evaluator reproduced one of them verbatim in its own worktree (base 5/5 clean, arm-ahead 4/5). Scope unchanged and it is row
   6p's: derive these bounds from a stimulus measured in-test.

6k. [**LANDED 2026-08-26 · iteration 25 · PR [#923](https://github.com/sunholo-data/ailang/pull/923) →
   [`ff0da7445`](https://github.com/sunholo-data/ailang/commit/ff0da7445) · evaluator round 1 **PASS 98/100,
   ZERO blocking** · Gate 3b GREEN on the merge (16 checks, 0 pending, required test/lint/build success,
   `launchd drivers (bash 3.2)` success; 1 not-green = inherited non-required SonarCloud)**]
   **The driver pin refused unconditionally, because `hasCompletedProjectOnboarding` no longer exists** ·
   loop health · Preempted the queue as a genuine regression, and this fire was the instance: the driver
   logged `DRIVER PIN FAILED` and ran the source clone **152 behind** `origin/dev`. `_pin_onboarded`
   read a retired `~/.claude.json` key — **0 of 15** live project entries carry it, control
   `hasTrustDialogAccepted` **15 of 15**, negative control **0** — so the predicate answered `false` for
   every path and all three missions simulated **REFUSE-TO-PIN**. The suite could not see it: it writes its
   own fixtures still carrying the retired key, and one arm *asserted that the only key which now exists
   must be refused*. Fixed by accepting either key, plus an anti-vacuity floor that reports **schema drift**
   (fail-closed, #558's posture unchanged) so the next rename is loud on its first fire. Judge-driven
   round 2 split the refusal three ways — unreadable · drift · not-onboarded — after the evaluator found the
   drift sentence also fired for a missing/malformed file. Suite **35 → 53** arms; four mutants pin the two
   headline hunks and the supporting `local` hunk is recorded **unpinned** rather than claimed as covered ·
   1 iteration

6l. [**NEXT after 6h and 6i — BLOCKED-BY-DESIGN on `D-MOTOKO-WORKDIR-2`; landing it alone changes nothing**]
   **The pin gate's code is loaded from the tree the pin exists to replace — a bootstrap trap** · loop health ·
   Filed by iteration 26, whose own fire is the instance. launchd's `ProgramArguments` names the **source
   clone's** `tools/launchd/mission-control.sh`, which sources the **source clone's**
   `tools/launchd/lib/pin-root.sh`. So the predicate deciding whether to run committed code is read from the
   stale tree the pin exists to bypass. Iteration 25's fix (`ff0da7445`) is correct — measured `PIN-OK`
   against the live `~/.claude.json` — and has **never executed here**, because the clone that would have to
   read it is 172 commits behind. **The trap is self-sustaining:** a stale gate refuses, the refusal keeps the
   fixed gate from running, and the drift grows without bound (152 → 172 in one day).
   **Scope, and the ordering is the whole point.** (1) Make the gate refresh itself before it decides —
   `git -C "$src" fetch origin` is already run to compute drift, so extracting `origin/dev`'s own
   `lib/pin-root.sh` (`git show` to a temp file) and re-exec'ing through it costs one command and makes the
   *newest* gate logic always the deciding one. (2) Or move the launchd entry point behind a minimal
   bootstrap that fetches first and is small enough to be stable. **Either fix is itself subject to the trap**
   — it only begins working once a tree carrying it executes — so this row is **not** a substitute for the
   clone reconcile and must not be landed as if it were. That is exactly the mistake iteration 25 made one
   level down, and re-making it is the failure mode this row exists to prevent.
   **Standing negative control for any fix here:** it must still `REFUSE-TO-PIN` for a checkout that genuinely
   carries neither onboarding key (world, today), or it is indistinguishable from deleting the gate — the
   discrimination test iteration 25 already established · 1 iteration

6m. [NEXT after 6i — filed by iteration 27's EVALUATOR, not by the controller's own drill, which is the
   point] **`cacheRead = usage.PromptTokensDetails.CachedTokens` is pinned by nothing, and the test that
   looks like its coverage reaches a different code path** · loop health · Found by the round-1 judge as a
   non-blocking finding and reproduced first-party before it was believed: mutate that line to `+ 999` and it
   LANDS (sha256), BUILDS, effect-verifies **0 → 1** on the gofmt-parsed form, and leaves
   `internal/ai/openai`, `internal/ai/openrouter` and `internal/ai/ollama` at **rc=0 with 0 `--- FAIL` lines
   in total**. It is a live line — it is what populates `ai.Response.CacheReadInputTokens` for every OpenAI,
   OpenRouter and ollama `/v1` tool-calling turn, i.e. the cache-hit accounting the cost ledger reads.
   **The mirage is the interesting half and it is rule 3i's shape exactly** — *which write does this read?*
   `cache_usage_test.go` is named for this behaviour and contains **0** references to
   `ParseChatStepResponse` and **7** to `Generate`, a path that never enters the function. So a reader
   auditing coverage by filename finds a confident yes. This was NOT introduced by iteration 27 — the read
   existed at the parent commit and only its receiver was renamed — so per the rule that a hunk with no
   killer is a finding rather than a failure, and that a pre-existing defect surfaced by a reviewer is a
   queue row rather than a revision, it was filed instead of used to widen PR #946. Same disposition
   iteration 24 gave row 6i. **Scope:** one arm driving a body carrying `prompt_tokens_details.cached_tokens`
   through `ParseChatStepResponse` and asserting `CacheReadInputTokens`, plus the negative arm (details
   absent → 0); then check whether the sibling `chat.go`/`responses.go`/`streamstep.go` cache reads have the
   same hole, since a repo with one filename-shaped coverage mirage will have grown others · 1 iteration

6n. [**LANDED 2026-09-01 · iteration 32 · Gate 3b GREEN on the PR head (21/21 checks, `mergeStateStatus=CLEAN`), including `launchd drivers (bash 3.2): success` — the leg decision (d) named as the only place the runner's behaviour is observable, and the leg that failed on V1's `#971` with this exact discovery error. Merged as [#1008](https://github.com/sunholo-data/ailang/pull/1008) → `64ca81852`. Evaluator PASS 93/100, zero blocking. UNPARKED BY ATTENDED RULING `D-MOTOKO-6N-1` (Mark, 2026-09-01, commit `878e0a5a0`), option (B): adopt D4. The arm now asserts `process-tree discovery deadline expired (wall clock)`, which only the wall-clock branch can emit, and the judge's own T1 confirms the mutant reds that arm. Historical PARK text kept below as the trail.**]
   [was: PARKED needs-human-review 2026-09-01 · iteration 31 — design written, quorum BLOCKED 3/3 in BOTH
   rounds on one surface; the row's own runner blocker is DISCHARGED. Doc:
   [m-motoko-discovery-arm-discriminating-refusal](planned/m-motoko-discovery-arm-discriminating-refusal.md).
   **The defect is confirmed and the fix is measured to work locally**: baseline 41/41; neuter the wall
   clock alone → 41/41 with arm 33 still `ok`; neuter the node ceiling alone → only arm 40 reds, arm 33
   still `ok` (so each branch independently suffices and the arm cannot discriminate); the minimal fix
   alone → 41/41 in 50s; the minimal fix **plus** the wall-clock mutant → rc=1 in 44s with arm 33 the
   failing arm on the exact message. **What blocks it is not the fix, it is a trade**: asserting the
   wall-clock message converts a silent false PASS into a possible loud false RED if the node ceiling
   ever wins the race on the CI host, and that host's rate is not measured. Round 1 rejected raising
   `MAX_TREE_NODES`, round 2 rejected NOT raising it. **Controller error recorded**: the margin offered
   against the reviewers' premise was benchmarked on **system `pgrep`** (~79 iter/s, ~52×) when arm 33
   PATH-overrides to this suite's own stub — corrected first-party to **~475–653 iter/s, 4096 iterations
   ≈ 6.3–8.6s, a ~6–9× margin**. Caught by the iteration-31 evaluator. **The evaluator also supplied the
   synthesis both rounds missed (D4)**: scope `PROBE_MAX_TREE_NODES` to arm 33's own `env` line —
   measured free on the happy path (41/41, 47.1s) — which removes the race structurally instead of
   betting on it. Recorded, not applied: the revision budget is spent and applying it would be a
   controller-invented resolution to a blocked quorum. **ONE-WORD ASK to Mark is D-MOTOKO-6N-1.**]
   **The wall-clock discovery arm cannot fail for the reason it names, and on the CI runner its branch never fires
   at all** · loop health · reported at [#975](https://github.com/sunholo-data/ailang/issues/975), filed by V1
   iteration 308's evaluator against motoko's instrument. Ghost-disciplined at motoko's own HEAD rather than
   inherited: neuter the in-loop wall-clock check in `descendant_pids` with
   `if false && (( $(date +%s) > deadline )); then` — mutant LANDED (sha256 differs), PARSES (`bash -n` rc=0),
   intended effect asserted against the system's own view (neutered sites 0→1) — and the whole suite stays
   **rc=0, 41 ok, 0 not ok**, this arm included. So the branch the arm is *named* for can be deleted outright and
   CI stays green. Cause: `sample_tree` collapses every `descendant_pids` failure into one generic
   `process-tree discovery failed` wrapper, which the arm greps, so it passes equally for the wall-clock deadline
   and for the 4096-node ceiling; the sibling node-ceiling arm is fine because it asserts its own discriminating
   message. **The same class as row 6i one arm over** — an assertion over an over-subscribed observable — and it
   is PRE-EXISTING (`git show c29ec1d00:tools/eval/test_motoko_connection_probe.sh` already carried that string).
   `#975` also reports a THIRD finding (B3) that the obvious fix reddened `launchd drivers (bash 3.2)` because on
   the runner discovery refuses with NEITHER diagnostic message — that half is unreproduced here and must be
   measured on the runner, not locally, before any fix is designed · 1 iteration

6o. [**LANDED 2026-09-03 · iteration 34 · Gate 3b GREEN — 21 checks, 0 non-green, required 4/4, `launchd drivers (bash 3.2): success`.** PR [#1034](https://github.com/sunholo-data/ailang/pull/1034) rebase-merged (`b97cbf83c`..`684ab8331`, five commits kept so the milestone boundaries stay bisectable). **The three new arms are CI-VERIFIED, not just local**: `ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild`, `ok 44`/`ok 45` for containment, `PASS: 46 probe self-test arms ran` on the macOS runner (control: a pre-existing arm in the same log). Historical tag kept as the trail, because the mid-iteration reversal is the record:  **IN-SPRINT — BUILT, JUDGED AND VERIFIED; NOT LANDED, AND THE REASON IS NOT OURS.**
   PR [#1034](https://github.com/sunholo-data/ailang/pull/1034), five commits, rebased onto `267a94e92`, head `db1996128`, OPEN.
   Evaluator **PASS 96/100, ZERO blocking** (sonnet, own worktree → generator≠judge). Suite **43 → 46 arms**; the headline
   mutant (probe:261 alone → single-PID), which SURVIVED at base, now reds with the new arm the **SOLE** `not ok`; row 6i's
   both-sites mutant still reds arm 36, so the old coverage is intact. Green at every milestone boundary **43 → 44 → 46**;
   production probe byte-unchanged (`f0b5e024…aabc99`).
   **BLOCKED ON A PREDICATE, NOT ON A DECISION — nothing is asked of Mark.** Required checks `test` and `lint` are red on
   **dev's own HEAD**, two independent causes, neither reachable by this diff (0 Go files): the `FMT_AB_TESTABLE_FUNCTIONS`
   markers deleted by `c8c841e24` (V1's #1030 covers it), and a `make fmt` red on seven Go files from the coordinator work
   (NOT covered by #1030). V1 owns dev CI on this anchor; handed over on the cross-mission channel.
   **RESUME PREDICATE — DISCHARGED IN-ITERATION.** V1 landed `b51e53f78` ("dev has been red for 24h on five defects stacked in
   one sequential job") ~90 minutes after the hand-over. It covers BOTH causes reported, including the `lint` one its own
   in-flight PR did not — and found three more nobody had seen. The branch was rebased onto the fixed base, re-verified
   (rc=0, 46 ok), and merged. **The scope caveat below is therefore RETIRED**: `make test-launchd-drivers` is rc=0 locally
   AND green in CI with all three new arms executing.
   **SCOPE AS IT STOOD BEFORE THE RESUME (kept as the trail, now RETIRED):** the `launchd drivers (bash 3.2)` target died at
   the fmt_ab script before reaching the probe suite, so at that moment these arms had never run in CI. Design + plan: `m-motoko-group-kill-and-lsof-containment.md` (+ sprint plan), quorum BLOCKED r1/r2, resolved
   under the narrow-refinement carve-out with the reviewers' verbatim fixes.
   Historical tag kept as the trail: NEXT after 6n — filed by iteration 30's EVALUATOR, both findings REPRODUCED first-party
   by the controller before being believed AND before being dismissed]
   **Only the TERM half of the production group kill is pinned; the SIGKILL escalation's group form has zero
   killers, and the survivor oracle's PATH immunity is narrower than the code claimed** · loop health ·
   **(a)** `run_lane` escalates to `kill -9 "-$pid"` when the driver survives a 5s TERM grace window. Mutating
   ONLY that site to `kill -9 "$pid"` — mutant LANDED, PARSES, intended effect asserted (group `-9` sites 1→0
   while group `-TERM` sites stay 1) — leaves the suite **rc=0, 41 ok, survivors=0**. The fixture's grandchild is
   a plain `sleep` with no SIGTERM trap, so it dies at the TERM stage and the escalation is never reached: that
   stage is dead-for-discrimination. Row 6i's mandated mutant changes BOTH sites at once, so it reds on the TERM
   half alone and the doc's acceptance bar is honestly met — this is the gap the bar itself could not see, which
   is rule (i-e)'s shape (a removal proves the check FIRES, only a differently-shaped mutant proves it LOOKS).
   **Scope:** a fixture grandchild that traps SIGTERM (or a shortened grace window in the run_lane fixture only),
   so the `-9` group form gets its own discriminating arm. **(b)** `REAL_LSOF` is resolved with `command -p -v`,
   whose POSIX standard-path guarantee **does not hold on the CI shell**: measured on GNU bash 3.2.57
   (arm64-apple-darwin25), a shadowing `lsof` ahead of `getconf PATH` in the AMBIENT environment resolves as
   `REAL_LSOF` (arm → the hijack; control, clean PATH → `/usr/sbin/lsof`; negative control, hostile dir without
   an `lsof` → `/usr/sbin/lsof`). The defence the design doc actually names — the stub PATH the suite installs
   for itself — holds; the code comment's "can never" did not, and was narrowed to what is defended in
   `1caf02e44`. **Scope:** assert the resolved `REAL_LSOF` lies inside `getconf PATH` and hard-fail on Darwin
   otherwise, which turns a measured hole into a fail-loud rather than a comment · 1 iteration

6t. [NEW — filed by iteration 34's own Gate-5 retro, from a first-party friction; NOT a skill edit, see below]
   **Gate 3b's prescribed CI poll cannot run to its own deadline in this harness, and the collision is already documented
   one recipe over** · loop health · The skill's Gate-3b snippet is `deadline=$(( $(date +%s) + 1800 ))` — a 30-minute
   bounded poll — and the natural way to run it is a foreground `Bash` call, which the tool layer **caps at 10 minutes**.
   Measured this iteration: the poll was killed at exactly `10m 0s` (exit 143) with **15 of 18** checks complete, having
   printed a clean progress trace and no verdict. It is recoverable (re-poll) but it costs a turn, and a controller who
   read the truncation as a timeout would PARK a landing that was about to go green — the exact disposition error Gate 3b
   exists to prevent. **The identical collision is already recorded in this same file** for the codex executor recipe
   (*"the wall-clock cap is 30 min but the `Bash` tool caps at 10 min"*), where the fix was to launch it with
   `run_in_background: true`; nothing carried that to Gate 3b's own snippet. That is the file's own named
   *guard the helper, miss the call site* shape, aimed at the gate that decides LANDED vs parked.
   **Why this is a queue row and not iteration 34's one skill edit:** the second instance is a *different call site of a
   fixed problem* rather than a second failure of the same rule, which is a thinner case than the ≥2-recorded-frictions bar
   wants; and the running skill resolves to **V1's main checkout**, so landing the edit from a motoko worktree would reach
   origin and NOT the copy that executes — manufacturing exactly the drift iteration 33 measured at 223 lines and this
   iteration found closed. Filed so the next instance is recognised rather than rediscovered · 1 iteration

6u. [NEW — filed by iteration 35's Gate-5 retro, from two first-party frictions in ONE iteration]
   **The spawn-pin hook enforces the DECLARED role pin and knows nothing about either fallbacks or
   `resolve-role-spawn.sh`'s own derivation — so it denies the two spawns the rulebook tells you to make** ·
   loop health · Two measured instances, both this iteration, both at the tool boundary where there is no
   appeal. **(a) It has no notion of a fallback.** `codex:gpt-6-astra` failed its first real designer run
   (rc=0, zero artifact). The rulebook's remedy is `MISSION_DESIGNER_FALLBACK` = `claude:claude-fable-5-1`,
   and the roles table's 2026-08-20 correction explicitly says a Fable role may be spawned with an
   `Agent(model="fable")` pin. That spawn was DENIED: `deny:provider-pin — designer is pinned to
   codex:gpt-6-astra; Agent-tool alias spawn refused`. The hook reads `MISSION_DESIGNER_MODEL` and stops,
   so **a role whose primary lane dies cannot be re-routed through the Agent tool at all** — the fallback
   only exists on lanes that happen to be provider recipes. **(b) It contradicts the resolver.** For the
   planner, `resolve-role-spawn.sh planner <doc>` returned `agent-tool opus fail-closed:planner-lane-field-missing`
   and `derive-planner-lane.sh` agreed (run from the worktree holding the doc, so not iteration 33's scope
   trap); the hook returned `planner is pinned to codex:gpt-5.6-sol`. Gate 3 says the resolver's line is to be
   used VERBATIM and `do not second-guess it` — but the hook is what actually adjudicates, so the instruction
   is unfollowable whenever the two disagree, and the controller must silently pick one. Both were handled by
   routing to the role's configured provider fallback and FLAGGING, which is the right disposition and is not
   a fix. **Scope:** make the hook read the SAME resolver the skill prescribes (or make the resolver read the
   hook's rule), and give it a fallback-aware mode so a probe-failed or run-failed primary can degrade without
   the degradation itself being denied. Until then, every FLAG this loop writes about a role model is a claim
   about a negotiation between two disagreeing components rather than about a routing decision.
   **Why a queue row and not iteration 35's one skill edit:** the defect is in
   `tools/launchd/resolve-role-spawn.sh` and the spawn-pin hook, i.e. CODE, not in the rulebook — the rulebook's
   instruction is correct and merely unexecutable. Filed so the next controller recognises the denial instead of
   rediscovering it · 1 iteration

6p. [**M1 OF 3 LANDED 2026-09-05 · iteration 35** — PR [#1048](https://github.com/sunholo-data/ailang/pull/1048),
   four commits. The suite now MEASURES the host fork rate in-test and derives its bounds from it, and the
   derivation is **additive**: `BOUND_SCALE`, the derived arm cap and the node ceiling are computed, validated and
   published, but consumed by nothing (`ARM_CAP_BASE=120` still feeds `ARM_CAP_SECS`; `$(bound_secs ` consumed **0**
   times — the evaluator confirmed this independently). Suite **rc=0, 57 ok** against a base of 46. The diag line
   publishes `r=`, `r_real=` and `p_obs=` every run, and **`p_obs=1.36` independently reproduces the designer's
   interleaved 1.35×** — the design's central claim measuring itself. Carries both round-2 carve-out fixes in the
   reviewers' own words: `gpt6-astra`'s (the counter incremented unconditionally after `|| true`, so a failing
   stimulus produced a positive rate that would have set every bound) and `oc-glm-5-2`'s (the proxy spread was
   budgeted and never measured; a second real-op measurement is now published). Evaluator **PASS 81/100**, its one
   blocking finding fixed in the PR.
   **M2 AND M3 REMAIN AND ARE THIS ROW'S RESIDUAL** — wire the wall-clock class, enforce the floor and gate `p_obs`
   (M2); derive the node ceiling on the discovery arm (M3). Both are fully specified in
   `m-motoko-suite-bound-derivation-sprint-plan.md` and blocked on nothing: the executor was killed by its
   30-minute cap after M1. **NEXT.** · 1 iteration.
   Historical tag kept as the trail: DESIGN HALF LANDED 2026-09-01 · iteration 32 — D4 shipped at `PROBE_MAX_TREE_NODES=50000`, scoped to arm :449's own `env` line, with a gate refusing any suite-scope value (verified non-vacuous in BOTH arms by the evaluator: the same command is rc=0/41-ok at base `48817dcdd` and rc=1 on the merged tree). **RESIDUAL, and it is a real one, kept OPEN here rather than declared closed:** the ceiling is a hardcoded constant calibrated on one machine at one load level, and iteration 32 measured the calibration input moving **3.3–3.6x from ambient load alone** — 181–200 iter/s under load average 20.59/16 CPUs against iteration 31's 474–653 quiet, same machine, byte-identical stub, confirmed against a second clock. The race leg is therefore far stronger than argued (**125x** contended, and bounded below by hardware: 25,000 iter/s is ~100x this machine's bare `/usr/bin/true` spawn rate) while the backstop leg is weaker (**2.2x–5.7x** under the 600s cap, not 5.7x) — and at the doc's own stated margins the feasible interval is **EMPTY**. The shipped value satisfies the leg whose violation is a WRONG VERDICT and accepts degradation in the leg whose violation is only a WRONG EXPLANATION (a correct red, cap-shaped rather than message-shaped); the evaluator reproduced exactly that degradation under contention. **The durable fix is not a different constant** — it is to derive the bound from a stimulus measured in-test, so the ratio holds by construction on any machine. That is the work this row still owns · 1 iteration.
   **CORROBORATED 2026-09-02 (iteration 33) from a second direction, and the class is wider than this row states.** Row 6p is
   written about ONE hardcoded constant (`PROBE_MAX_TREE_NODES`). Iteration 33 measured the same defect shape in the suite's
   *wall-clock* bounds: two further arms (`refusing live path`, a 4s sampling deadline; `production run_lane fixture readiness`)
   red intermittently at load 39–46 on 16 CPUs, and moving unrelated work out from in front of them took the observed rate from
   4-in-19 to 0-in-5. So "derive the bound from a stimulus measured in-test" is owed by at least three arms, not one, and the
   deliverable is better shaped as the suite's own bound-derivation helper than as a per-arm constant. Rates and controls are on
   row 6j · 1 iteration]
   [was: NEXT after 6o — PRE-EXISTING at HEAD, filed rather than absorbed into row 6n's doc]
   **`descendant_pids` bounds its walk by two mechanisms that race, and nothing chooses between them**
   · loop health · The wall clock fires on WALL TIME (`date +%s > deadline`, a 1-second window at
   `PROBE_TIMEOUT_SECS=1`, and note `date +%s`'s 1-second granularity makes the true window 0–2s); the
   node ceiling fires on ITERATION COUNT (`visited > MAX_TREE_NODES`, default 4096). Which wins is a
   property of the machine. Measured on the pin worktree against the suite's OWN `pgrep` stub (the
   binary the arm actually calls — NOT system `pgrep`, which is 6× slower and is the instrument the
   controller first got this wrong with): **474.9 / 652.7 / 648.6 iter/s**, so 4096 iterations take
   **6.3–8.6s** against a 1–2s window — the wall clock wins here by 6–9×, and runner contention slows
   spawns, which moves the margin the protective way. This is **not introduced** by row 6n's doc; it is
   why that doc's quorum could not settle. The evaluator's D4 (scope `PROBE_MAX_TREE_NODES` to the one
   arm's `env` line, measured free at 41/41 in 47.1s) is the candidate fix and belongs here rather than
   in 6n. **Do NOT re-open 6n's quorum to absorb this** — a pre-existing defect a reviewer surfaces is a
   queue row, not a revision · 1 iteration

6q. [**LANDED 2026-09-01 · iteration 32**, folded into `#1008`'s M3. `expected_refusal_branches` is now **27** (19 `instrument_failure "` + 5 `|| usage$` + 3 echo-shaped), with its own anti-vacuity floor and a `[[ -f "$probe" ]]` assertion so no counter in the gate can fall through to reading stdin. **The iteration-32 evaluator ran the addition-shaped mutant that the executor never reached** (its 50-min cap killed it first): a dead-but-real `instrument_failure` branch, guarded by `false &&` so it never executes, still moves the count **27 → 28** and reds the gate. So the gate reads the file's SHAPE, not runtime behaviour — it **LOOKS**, it does not merely fire, which is precisely the property this row existed to buy.]
   [was: NEXT — bookkeeping-sized, found by the iteration-31 evaluator's addition-shaped mutant]
   **The refusal-branch count gate is blind to the three refusals this whole arc is about** · loop
   health · `test_motoko_connection_probe.sh`'s `expected_refusal_branches=24` counts
   `instrument_failure "` (**19**) + `|| usage$` (**5**) and does **not** count the three
   `echo "process-tree discovery ...` refusals in `descendant_pids`. Measured by the evaluator: adding
   a synthetic echo-shaped refusal branch leaves the suite **41/41 ok** — the gate whose entire purpose
   is "only an addition proves it LOOKS" cannot see an addition in this shape. Extending it to
   19 + 5 + 3 = **27** with its own anti-vacuity floor makes the addition red (`probe has 28 refusal
   branches ... written for 27`, verified by running it). Row 6n's doc carries this change; if 6n stays
   parked, this lands on its own · 1 iteration

6r. [**LANDED 2026-09-02 · iteration 33 · PR [#1027](https://github.com/sunholo-data/ailang/pull/1027) → [`115184a2e`](https://github.com/sunholo-data/ailang/commit/115184a2e) · evaluator round 1 **PASS 76/100, 1 blocking** — and the blocking finding CHANGED THE SHIPPED CODE. Gate 3b GREEN on the merge: 14 checks, 0 pending, required 4/4, **`launchd drivers (bash 3.2): success`** on both the PR head and the merge. Two non-required reds (`test-windows`, `Build windows-latest`) measured as INHERITED — both `success` on my base `7292ec780` and its parent, `origin/dev` itself red on the identical two, cause `f3301a44c` landed after my base, and a PR's checks run on the test-merge; V1 owns it and has `sprint/iter320-home-isolation` in flight. The arm asserts `process-tree discovery deadline expired (test stub)` and BOTH the strip and the reword mutant red it by name; the append-shaped mutant is vacuous by construction under `grep -Fq` and is documented as such. **The judge's finding, and the iteration's real result:** placed after the caller-message arm the new arm sits ahead of every wall-clock-bounded arm and correlates with them tipping over under contention — measured three ways, one probe, interleaved, five rounds: base **5/5**, arm-at-26 **4/5**, arm-at-42 **5/5**; pooled with the judge's own A/B, base **0 reds in 17**, arm-at-26 **4 in 19**, arm-at-42 **0 in 5**. M3 moved it behind the bounded arms, so the mechanism is unreachable by construction rather than merely unlikely, and the arm is still the sole killer. Residual filed on **6j**/**6p**; the judge's non-blocking coverage finding is row **6s**. Historical NEXT text kept below as the trail.**]
   **The `(test stub)` refusal message is pinned by a static grep and by nothing behavioural** · loop health · Reverting just the
   `(test stub)` suffix at `motoko_connection_probe.sh:182` — leaving `:187`'s `(wall clock)` intact — leaves the suite **41/41 ok,
   rc=0**, measured by the evaluator on a scratch copy. The only thing holding that hunk is the sprint plan's `grep -Fc` acceptance
   check, i.e. a claim about the file's text rather than about behaviour. This is **not a defect the sprint introduced and not one it
   hid** — the design doc says so itself ("the stub's message is never directly asserted") — but it is a hunk with no killer, which
   this loop's own rule says is a finding rather than a failure. **Scope:** either give `PROBE_TEST_DESCENDANT_FAILURE=1` an arm that
   asserts its own discriminating message (the symmetric treatment the other two branches now get), or declare the branch
   behaviourally unpinned **in the code**, so the gap is named where a reader meets it rather than only in a design doc · 1 iteration
6s. [NEXT — filed by the iteration-33 EVALUATOR as a non-blocking finding, reproduced first-party by the controller]
   **Nothing in CI would notice a self-test ARM silently disappearing — the suite counts refusal branches in the probe but never
   asserts its own arm count** · loop health · The refusal-branch drift gate (`test_motoko_connection_probe.sh`,
   `expected_refusal_branches=28`) exists precisely so a new PRODUCTION refusal branch cannot ship unpinned, and the iteration-32
   evaluator proved it LOOKS rather than merely fires. There is no symmetric gate on the TEST side: the only arm-count assertion
   anywhere is `arms == 0` (fatal), so deleting any single arm — including the one row 6r just landed — leaves the suite rc=0 and
   green, and the loss is invisible. Measured 2026-09-02: base 42 ok rc=0, HEAD 43 ok rc=0, and a plain run is green either way;
   only a deliberate mutation drill distinguishes them. This is the same guard-the-helper-miss-the-call-site shape the probe gate
   already closed one level down, aimed at the suite that guards it. **Scope:** an `expected_arms` floor beside
   `expected_refusal_branches`, with the same anti-vacuity treatment (a non-numeric or zero count is `instrument failure, not a
   verdict`) and an ADDITION-shaped mutant proving it LOOKS — a removal only proves it fires. **Cost to weigh before taking it:**
   an exact arm-count gate reds on every legitimate arm addition, a maintenance tax the branch-count gate already charges and that
   this loop has judged worth paying once · 1 iteration
7. [NEXT after 6h and 6i] **Profile restoration design** · clause 4 · 5 profiles, 14 of 18 model entries · 1 iteration
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

15. [LANDED 2026-09-06 · iteration 36 · **Gate 3b pending at time of writing — see the STATUS stamp
    for the settled verdict**] **Unbreak `dev` after the attended Phase 1 workbench landing** ·
    clause 6 · PR [#1055](https://github.com/sunholo-data/ailang/pull/1055), seven commits.
    M-MISSION-LOOP-WORKBENCH Phase 1 was landed **attended**, directly to `dev`, between 22:15 and
    23:02 local on 2026-09-05, with **no charter row and no log entry** — this row is that missing
    record as much as it is the fix. It left six checks red: `lint` (a dead initialiser), the
    windows pair (`filepath.IsAbs` is platform-dependent, so ~20 tests failed validation), `docs-build`
    /`docs-gate` (a repo-relative link escaping the docs tree) and `launchd drivers (bash 3.2)` (a Go
    test wired into a job that installs no Go). A **fifth** defect was found only by measurement, not
    by any CI log: `kill_unix.go` shipped with no build constraint and no windows counterpart, and Go
    does not treat `_unix` as a GOOS suffix — the package did not COMPILE on windows, so fixing the
    validation alone would have left both windows checks red while appearing to fix them.
    Evaluator PASS **85/100**, zero blocking; its strongest objection (M2's test comment overclaimed
    its mutation coverage — the revert is invisible on darwin) was reproduced and fixed in M6.
16. [NEXT — bookkeeping, ≤1 iteration] **Changelog debt for the workbench Phase 1 arc** · clause 6 ·
    `coding-standards.md` requires a `CHANGELOG.md` entry per change; the seven-commit Phase 1
    landing added none, and neither did iteration 36's fix-forward — measured, `git log --name-only
    f5edd569a~1..origin/dev -- changelogs/ CHANGELOG.md` is **empty**. Raised by iteration 36's
    evaluator and deferred deliberately, not dropped: the changelog is sectioned by RELEASED version
    (top entry `v0.35.1`, already shipped), so a new unreleased section for the fix alone would
    misdescribe both halves. Write ONE entry covering the feature and its fix when Phase 1 is
    written up. Note `make check-changelog` is index hygiene only — it is green and will not catch
    this, so nothing else will surface it.

---
**Document created**: 2026-08-12 (rewritten from the 2026-06-24 charter, archived in full at
[motoko-mission-status-archive.md](motoko-mission-status-archive.md)). Iteration 0 ratifies it via
the quorum with Mark before any sprint routes.
