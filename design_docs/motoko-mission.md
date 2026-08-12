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
- **Working checkout**: `/Users/voightkampff/dev/sunholo-data/ailang-motoko` (driver:
  `MISSION_WORKDIR`) — **a SEPARATE clone from V1's**, deliberately. There is no cross-mission
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

## STATUS 2026-08-12 — ITERATION 2 COMPLETE: **PHASE 0 IS NOW A BOUNDED FAIL-CLOSED GATE — AND THE PREDICATE AS FIRST DRAFTED COULD NEVER HAVE FLIPPED TRUE. ITEM 4 PARKED `needs-human-review` ON A DOC-vs-CHARTER CONTRADICTION.** Queue item 4 (head): designer pass on [m-motoko-dst-refactor-migration.md](planned/m-motoko-dst-refactor-migration.md) + the one re-quorum the item allows. Gate 0/1 clean — the RUNNING skill is byte-identical to `origin/dev` (`cmp` silent) and local `dev` == `origin/dev`, so no reconcile was owed; 18 checks on HEAD, 17 green, `test` in-flight, no RED to outrank the queue. Designer ran on `claude:claude-fable-5` (rotation start — the namespaced `mission-motoko-designer-rotation` was absent; probe rc=0, `claude-sub` wrapper, tripwire CLEAN), 11,159-byte directive, rc=0, **exactly one file touched, no git writes**. **R1's two objections are answered and were NOT re-raised at R2**: Phase 0 became a conjunction of four predicates each carrying its evaluating command and observed value, a **28-fire (~14d) timebox** at the charter's 12h interval, a structured BLOCKED expiry escalating to Mark, and a declared human residual; and the four "Port — carry forward" claims got rows V21–V24, each with a **same-scope** known-positive control. **The iteration's sharpest find is a defect in the designer's own output, caught only by running the predicate table verbatim instead of reading it (V25):** the draft wrote G2 as `git cat-file -e origin/main:packages/…` with **no repo**, and `origin/main` is ambiguous across this mission's three repos — from this checkout it is `rc=128 invalid object name`, because the anchor repo's default branch is `dev`. The wrong-repo error and the genuine path-absent answer **both return 128**, so an `exits 0` test cannot separate them and **G2 would have read FALSE forever, including after `#154` merges** — a gate that can never open, wearing a correct gate's clothes. Fixed by pinning `-C` and pairing the predicate with a `README.md` control that proves the ref resolves. **R2 BLOCKED on two NEW internal-consistency defects**, both measured rather than forwarded (rule 3f): `gemini-3-1-pro` found (i) the Design Freeze said "check off before *sprint-executor* starts" — a **deadlock**, since Phase 0 is precisely what the executor runs to evaluate G1–G4, silently restoring the unbounded wait R1 rejected (fixed with its verbatim wording) and (ii) the Phase 1 ordering rationale claimed `compaction-ai`'s `on_pre_step` "genuinely needs all ten effects" — **refuted and WORSE than filed** (V26): the row is `! {AI, IO, Trace}`, **three**, and the ABI commentary the sentence cited *as its authority* is one upstream has **retracted** — *"it was taken as given … over-declared by SEVEN"* — with `on_tool_handle`'s genuine ten-effect row in the same record as the firing control. `gpt5-6-sol` found the gate authorises Phase 1 on G1–G4 while the doc's own Dependencies demand more — and the contradiction reaches **outside the doc**, because the **ratified charter guardrail** requires "#154 merged **AND** Arni has declared the ABI stable", which the R1 *Declared residual* had quietly relaxed. Its fix has two branches, one needing the named decision owner, and branch A's G6 would make queue item 11 — itself Phase-0 gated — a Phase-0 predicate, a dependency loop. **Choosing is Mark's call, so item 4 is PARKED `needs-human-review` rather than force-passed**, the one-revision-one-requorum budget being spent. **Nothing is blocked by the park**: Phase 0 measures CLOSED today (G1 `#154` OPEN with a firing MERGED control, G2 `packages/` absent from `origin/main`, G3 the registry exposes only 1.0.0–2.2.0 against a non-empty 4-version control), so no sprint existed to hold. Metered **$0.096** of $5. Both reviewers present both rounds, `absent_reviewers` empty — no N−1 hole.

## STATUS 2026-08-12 — ITERATION 1 COMPLETE: **THE FIRST ITERATION THAT ACTUALLY RAN. QUEUE ITEM 3 LANDED — 51 FORK COMMITS DISPOSITIONED — AND THE INDEPENDENT JUDGE BROKE 2 OF 16 `SUPERSEDED` ROWS, BOTH REAL.** Iteration 0 was bootstrap and the 09:11 fire died on V22's trust-dialog defect, so this is the loop's first end-to-end pass. Gate 1 found local `dev` **4 behind** origin with a clean tree; reconciled via `git checkout -B dev origin/dev` after measuring all four obligations with firing controls, and confirmed git wrote a **new inode** (35083371→35155023) so the in-flight driver kept its old fd. **Quorum-at-pick BLOCKED the migration doc** (both reviewers present, `absent_reviewers` empty, **$0.064**) and per rule 3f both objections were MEASURED rather than forwarded: `gemini-3-1-pro`'s "the four Port claims are unverified" is procedurally right and **substantively refuted 4/4** (it needs verification rows, not a redesign); `gpt5-6-sol`'s "Phase 0 is an unbounded wait" is **upheld — and its obvious remedy is vacuous**, because ABI 5.0 declares `[stability] level = "stable"` and so does the 2.2.0 we are pinned to and call unstable, so a gate on that field passes immediately and falsely. That is the sharpest negative result of the iteration and it is now queue item 4's starting point. The pick itself produced [m-motoko-fork-disposition.md](planned/m-motoko-fork-disposition.md) (`752254d3f`): **14 SUPERSEDED / 16 PORT / 14 DROP / 7 UNRESOLVED = 51**. Two instruments were retired before they could mislead: the range holds **52** commits but `ed61097` is a merge whose second-parent diff is empty, so 51 is the ceiling and the doc's own success criterion asked for a row that cannot exist; and **path existence is not a supersession signal here** — 80 of 94 touched paths survive upstream, but `agent_loop_v2.ail` is **4,005 B** there against **95,868 B** here, so the paths survive as facades while the substance moved, and a sweep built on that test would have been ~85% wrong. The `codex:gpt-5.6-sol` executor ran as pinned (probe rc=0, no lane degradation) and its evidence held under three first-party re-runs that reproduced exactly, file:line and denominators included — but the **sonnet** evaluator (generator≠judge: openai executor, anthropic judge) still **FAILED it 65/100** and was right twice, both in `SUPERSEDED`, the one verdict whose error is irreversible: **R16** cited `should_retry_stream_error` by name while its body is `retryable && retry_enabled && budget > 1` and `session.ail:2379` builds `code: "Internal"` with `retryable: false` **in the file the row cited**; **R34** named `phase_vocab.ail` as superseding a resolved-config broadcast, but that module is types and four candidate tokens measure **0** upstream against a firing 29-file control. Both were the failure mode the directive had named as the primary attack surface — a token match read as a behavioural equivalence — and both were reproduced by the controller before the downgrades were applied. The artifact landed as **REVISED-after-FAIL, not as a pass**, and the UNRESOLVED count went **up** under review, which is the direction to trust. Gate 3b: 13 of 14 checks green on the pushed SHA with `test` still running at the poll's bound, recorded as **not-yet-green** rather than waved through.

## STATUS 2026-08-12 — ITERATION 0 COMPLETE: **CHARTER RATIFIED BY MARK; QUORUM BLOCKED TWICE AND ALL FOUR OBJECTIONS WERE TRUE; THE V21 DRIVER DEFECT IS FIXED FOR v1+motoko.** Bootstrapped per [mission-bootstrap.md](../docs/docs/guides/mission-bootstrap.md) (separate checkout, env profile, plist, kill switch ON, dry-run isolation proven). Charter put through `ailang design-quorum` twice, both reviewers present both rounds, **metered $0.115 total**. Round 1 — `gpt5-6-sol`: no Premise Verification Log for operationally decisive claims. `gemini-3-1-pro`: clause 6.2 designs a fallback with no loud signal, **in a charter that cites the World mission losing five iterations to exactly that**. Both fixed (V1-V18 log added; 6.2 now requires a degradation-time notice). Round 2 — `gpt5-6-sol`: the *live* `codex→pi→opus` chain has none of the safeguards clause 6 demands of the future motoko lane. `gemini-3-1-pro`: V19 illegitimately inherited a **per-working-tree** build artifact claim from V1. The V19 fix (running `make quick-install && make build` here) then surfaced **V20 — `quick-install` writes the SHARED `~/go/bin/ailang`** that V1 and the eval rig resolve through, a cross-mission side effect no reviewer could have predicted. V21 measured the round-2 objection and **confirmed it: lane demotion is logged (driver 360/392) and never posted to the bookkeeping issue — a defect affecting v1, world AND motoko.** Re-quorum budget was exhausted (one re-run, per the guardrail), so V21 was escalated rather than ping-ponged. **Mark ratified the bar + queue and routed the V21 fix to this mission**, which then landed it in the driver: a degradation ledger accumulated at both probe loops and emitted ONCE, after every early exit and before the iteration starts, over the same two channels the driver's four existing report sites use. Deliberately NOT fail-closed on the post — aborting would make GitHub availability a hard dependency of every fire — but a failed post is itself loud, which is the one thing the old path never was. Tested at both halves without spending an iteration: five stubbed-channel tests over the real emit block (fires when degraded; **silent when healthy**; `gh` invoked; `gh` failure warns and does NOT abort; unset issue warns), plus a forced codex failure proving accumulation (probe rc=1 → handed to the pi lane). The seam between the two halves is now permanently testable: `MISSION_DRY_RUN=1` reports `lanes=ok` / `lanes=DEGRADED(n)…` — both arms exercised. **World is still owed the fix** and cannot take it by copy: its driver has drifted 233 lines (165 ailang-only / 68 World-only) and its fallback site is shaped differently (`_fb`), so it is handed over via the cross-mission channel rather than overwritten mid-flight.

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
   each naming its settling measurement; Phase 3 is not done until they are settled
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
5. [NEXT] **Output-headroom upstream issue** · clause 3 · file the case against `main_dst` (qwen3 arithmetic
   + the `docx_lambda` failure) if Arni's #97 reply invites it. **Iteration 1 note:** the
   disposition's **R8 is UNRESOLVED** on exactly this question and names the settling measurement,
   so this item now has a concrete instrument to cite rather than a recollection · 1 iteration
6. **fmt re-measurement instrument** · clause 3 · design HOW we re-prove the −74% tokens-to-pass
   result on the new tree. Decides whether `motoko_ext_fmt` survives. **DST will NOT do this** —
   `fmt` is an effectful extension hook, precisely the class measured at 1-of-~40 coverage (see DST
   scope). Design a real instrument and price it honestly rather than assuming it is cheap ·
   1-2 iterations
7. **Profile restoration design** · clause 4 · 5 profiles, 14 of 18 model entries · 1 iteration
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
