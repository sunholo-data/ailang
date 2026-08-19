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

## STATUS 2026-08-19 — ITERATION 13 COMPLETE: **MARK'S RULING UN-PARKED ITEM 6 AND THE TRACE HE MADE A PRECONDITION ANSWERS THE REVIEWER BOTH WAYS — THEN FINDS THAT THE FIX IT AUTHORISES CANNOT BE WRITTEN WHERE EVERYONE ASSUMED.** Pick was **not** the untagged queue head (item 7): `D-MOTOKO-FMT-1` was RESOLVED **precondition** by Mark in an attended session dated **2026-08-19** and recorded in `1a3ca2d5f`, and an answer to a parked item unparks it and becomes the pick. The ruling: *"the sprint TRACES motoko's resolved runtime provider first, then changes the preflight. Do not redesign around the unknown"*, with the trace required to **discriminate** — does removing the unconditional `OPENROUTER_API_KEY` refusal at `internal/executor/motoko/healthcheck.go:64` delete a real fail-fast, or admit a silent OpenRouter fallback for `provider: "ollama"`, `env_var: ""` entries? **RUN, via the `mk-ast` fork-resolution-path arm the ruling names as sufficient (`and/or`), no GPU and no `rig.lock`. THE ANSWER IS BOTH, OF DIFFERENT LANES — so the remedy is a CONDITION, never a deletion.** (a) For the two fmt arms **no OpenRouter routing is reachable**: precedence in the fork is `process.env.MODEL ?? profileAgent.model ?? "anthropic/claude-sonnet-4-6"` (`index.ts:768-771`, read first-party — and the `models.yml:1854` comment citing `index.ts:580` for this is a **stale transcription**, that line is now event-printing code), tier 1 is always set by the Go executor (`motoko.go:343`) from `agent_model_name` (`models.go:493-494`) and tier 2 is the profile's own `ollama/qwen3.5:35b-a3b-mxfp8` — **measured** `GuessProvider("ollama/qwen3.6:35b-a3b-mxfp8")=ollama` with env var `""`, matched at `config.go:55-57` *before* the generic `vendor/model → OpenRouter` rule at `:67-73`, by an in-code comment demanding exactly that ordering; both profiles additionally pin `openai_base_url=http://localhost:11434/v1`; and motoko's own `required_secret_for_model` returns `""` for the `ollama/` prefix (`supervisor.ail:21-26`). **The measurement discriminates rather than passing vacuously** — the same call returned `openrouter`/`OPENROUTER_API_KEY` for all three fallback defaults (`anthropic/claude-sonnet-4-6`, `openrouter/auto`, `openrouter/anthropic/claude-haiku-4-5`), so both arms are present. (b) For the OpenRouter-routed motoko lanes the preflight **is** a real fail-fast **and the only one**, because motoko's own check is a WARNING that proceeds: `validate_secrets` → `emit_warnings` prints `{"type":"warning"}` and `main` continues to `run_with_config` (`supervisor.ail:11-19`, `:42-51`). **THE FINDING THAT RE-SHAPES D1, and it is the part no reading of `healthcheck.go` could give you: the condition is not expressible where the check sits.** `HealthCheck(ctx context.Context) error` takes **no task** — it is the shared `executor.Executor` interface method (`executor.go:31`) — so the only model it can read is `e.model`, and `cfg.MotokoModel` is **never set from models.yml**: **3** non-test hits total, of which the only assignment is the hardcoded `"openrouter/anthropic/claude-haiku-4-5"` at `factory.go:71` (control: `MotokoProfile` → **11** hits, so the grep finds wiring where it exists). The lane's real model arrives per-task as `task.Model` (`eval_benchmark_agent.go:174,195,253,389`), consumed by `getModel(task)` at Execute time — **after** the health check has already refused. So an `if` added at `healthcheck.go:64` would evaluate the OpenRouter default for **every** lane and refuse the ollama arms exactly as today: D1 is a plumbing change, not the ~0.5-day one-liner §6 prices. Three costed options recorded (set `cfg.MotokoModel` from models.yml · move the check into `Execute` · widen the interface, 6+ implementors), plus the recommendation to express the condition on the **resolved provider** — `ai.EnvVarForProvider(ai.GuessProvider(model))` returns the needed var *or `""`* for every provider (`config.go:104-119`) — rather than hardcoding one vendor; note `internal/executor/motoko` does not import `internal/ai` today, so the boundary must be checked first. **Free from the same read:** the preflight returns *before* the `motoko --version` query that discovers `e.motokoRepo`, and `MOTOKO_REPO` is what stops the profile silently degrading to `extensions.order=[]` — the defect `motoko.go:344-364` records as **39 of 39** eval sessions with `loaded_extensions=[]` — so a conditional check must not run ahead of repo discovery, or it would drop the `fmt` extension that IS the treatment. **NO SILENT-FALLBACK PATH EXISTS ANYWHERE**: repo-wide non-test Go reads `OPENROUTER_API_KEY` at exactly **4** sites (`ai_handlers.go:276`, `exec.go:475` — both explicit `openrouter` subcommands — `config.go:115,143`, keyed by an ALREADY-resolved provider, and the motoko preflight), none re-routing on absence; control **23** `OPENAI_API_KEY` sites. **WHAT THE TRACE DOES NOT SETTLE, moved to acceptance rather than claimed:** the static arm proves no OpenRouter routing is *reachable*, not that no connection is *made*; the live arm is circular today (the fmt lane cannot run while the preflight refuses it), so it becomes `AC-D1-live` — one fmt run reaching `localhost:11434` with **zero** `openrouter.ai` connections, asserted on the connection (not on the absence of an error, which "the run never started" satisfies equally) and paired with an OpenRouter-lane known-positive control. **DELIVERED**: doc `m-motoko-fmt-remeasurement-instrument.md` un-parked, O4 CLOSED with disposition, new **§12** (the trace, the three options, the ordering note, `AC-D1-live`) and verification rows **V25–V32**. **Phase-0 predicates re-run, not transcribed — all still FALSE, rows 10/11/12 stay parked:** G1 `#154` `state=OPEN`/`mergedAt=-` (control `#161`/`#162` → `MERGED` with non-null `mergedAt`), G2 predicate rc=**128** with the mandatory `README.md` control rc=**0** → FALSE for the right reason, G3 registry `latest=2.2.0`, `versions=1.0.0,2.0.0,2.1.0,2.2.0` → no 5.x, G4 unrunnable, G5 unchanged. **UPSTREAM IS WORKING OUR ISSUE:** `arniwesth/mot-100-fix-output-headroom` moved `ffd6256 → 2f61665` since iteration 12 and now carries **7** commits ahead of `main_dst`, including `da999ac fix(compaction): reserve provider output headroom` and their PR **#166** — i.e. `#165` is being acted on, and iteration 12's re-anchor landed in time to matter. **RULED OUT:** *the `models.yml:1854` comment "`MODEL env > profile config model (index.ts:580)`" is current* — the precedence claim is TRUE (re-derived at `:768-771`) but its **citation is stale**, which is rule 3b(v)(b) caught in the repo's own comments rather than in a controller's prose; the correct behaviour was re-measured, not inherited. *The silent-OpenRouter fallback the reviewer feared is live today* — it is **real but unreachable on the wired path**: tier 3 `anthropic/claude-sonnet-4-6` does resolve to OpenRouter (measured), and it fires only if MODEL is empty AND the profile does not resolve, which the unconditional `MODEL=` at `motoko.go:343` forecloses. Recorded as a residual with its trigger named, not as a live defect. Gate 0/1 clean: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, ran from the `#558` driver pin root detached and clean at `c29c48e96` == `origin/dev`, running skill **byte-identical to origin** (`cmp` rc=0); dev **verified green, not merely un-red** — **16** exact-SHA checks, **0** not-green, `runs_total=2` so a run exists; **0** human directives on `#743` since the watermark (of 10 comments) — the ruling arrived by the attended-session channel, not the issue; ledger valid at **3** rows, **0 OPEN**. No died-mid-flight traces: no motoko worktree or PR from iteration 12 outstanding; `#695`/`#613` are V1's, as the last four iterations also measured. Weekly sweep and rotation **both not due** (`#743` created `2026-08-17T05:48:23Z`, after the Monday-07:00 **local** boundary = `05:00Z`; 10 comments < 80). Routing: controller `claude:claude-opus-5` only; **no designer, planner, executor, evaluator or quorum spawned** — a controller-run trace answering a reviewer objection has no doc to design, no plan to write and nothing to judge, and the doc's own quorum is spent (2 rounds, both reviewers present, `absent_reviewers` empty in both artifacts). Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no `rig.lock`; `make quick-install` deliberately NOT run (shared-write guardrail) — the one build step taken was a throwaway `go test ./internal/ai/` whose test file was deleted and the tree re-asserted clean. Gates on **darwin/arm64 only** (rule 3b(viii)). Gate-5: **NO skill edit** — the iteration's friction (a repo COMMENT carrying a stale `file:line` citation for a behaviour that is itself still true) is instance 1 of *"rule 3b(v)(b) applies to the codebase's own comments, not just to controller prose"*, pre-registered at the ≥2-friction bar rather than written on one datapoint.

## STATUS 2026-08-18 — ITERATION 12 COMPLETE: **THE UPSTREAM ISSUE WE FILED SIX HOURS AGO WAS ALREADY ANCHORED TO A BASE THAT HAD MOVED 59 COMMITS — AND THE MAINTAINER HAD JUST CUT THE BRANCH TO WORK IT.** Pick was **not** the untagged head (item 7): Gate 1's external-predicate re-read — the rule iteration 11 added to this skill — found queue row 5's six-hour-old claim *"`origin/main_dst` is **still exactly `6c06b08`**, **0** commits since our V27–V29 rows"* **FALSE**, measured `git rev-list --count 6c06b08..8110ffc` = **59** (**100** files), with `src/core/session.ail` among them (`+67/-13`). Upstream cut `arniwesth/mot-100-fix-output-headroom` in the same window. **THE FINDING: a LANDED row's deliverable can rot against a base nobody re-reads, and every freshness rule in this file is scoped to a row you are about to PICK.** Iteration 11's rule catches a *blocked* row re-asserting a stale measurement; this is the mirror — the row reads LANDED so no gate revisits it, and its deliverable is not a local file but an artifact **published to an external party** pinned to that party's moving branch. **THE DEFECT IS INTACT — only the offsets moved, and that is measured, not hoped.** The ten-line window `session.ail:2552-2561@6c06b08` is **byte-identical** to `:2606-2615@8110ffc`, sha256 `c53792eb5b778cf32e72001006e485274b05f33d13f14cbe578c836e9e15f1dc`; **negative control** — the same window shifted by one line hashes `01d10306…`, so that is a real match and not a hash of nothing. All three `session.ail` citations in `#165` shift **+54**: `2552→2606` (the `context_limit` binding), `2556→2610` (`ext_context_limit = context_limit - pinned_tokens`), `2561→2615` (`seal_compacted_payload(…, context_limit, …)`). The asymmetry the issue is about — extensions get the reduced limit, the seal gets the raw one, five lines apart — survives verbatim. **The other cited files are byte-unchanged, with a SAME-CALL control:** `git diff --stat 6c06b08 8110ffc -- phase_vocab.ail compaction.ail compaction_structural.ail context_usage.ail session.ail` returns **`session.ail` only**, so the instrument fires (rule 3a(i-d)) and `compaction.ail:25-28`/`:30`, `compaction_structural.ail:97-113`/`:170`/`:191`, `phase_vocab.ail:145-155` all stand as written. **DELIVERED — a correction comment on [#165](https://github.com/arniwesth/motoko_agent/issues/165#issuecomment-5332596902)** carrying the old→new table, the block sha256 with its negative control, the same-call unchanged-files control, and the note that `arniwesth/mot-100-fix-output-headroom` points at `ffd6256`, an **ancestor** of `main_dst@8110ffc` with **zero** commits of its own, so branching from the tip saves a rebase. Posted `--body-file`, **delivery asserted by comment count 0 → 1**. **PHASE-0 PREDICATES RE-RUN, NOT TRANSCRIBED — all still FALSE, so 10/11/12 stay parked.** G1 `#154` `state=OPEN`, `mergedAt=-` (control: `#161`/`#162` return `MERGED` with non-null `mergedAt`) — but `updatedAt=2026-08-18T16:48:58Z`, i.e. actively moving, which is what put `main_dst` 59 ahead; G2 predicate rc=**128** with the mandatory `README.md` control rc=**0** → FALSE for the right reason; G3 registry `latest=2.2.0`, `versions=1.0.0,2.0.0,2.1.0,2.2.0` → no 5.x; G4 unrunnable; G5 unchanged. **RULED OUT:** *upstream already fixed it* — `session.ail:51` now imports `resolve_context_limit` from `src/core/context_usage`, which looks exactly like the fix, but `context_usage.ail` is **byte-unchanged** in the range and resolves the limit rather than reserving output headroom, and `:2615` still gets the raw value (the solved-upstream class, negative on measurement). *The four merge commits moved `session.ail`* — **refuted before posting**, and the draft comment had already asserted it: `git log 6c06b08..8110ffc -- src/core/session.ail` returns exactly **two** commits (`b1ad13b`, `a45f708`, both `#160`), which are the whole `+54`; rule 3b(v)(b) caught on this controller's own prose. *The fix branch has work on it* — it is an ancestor of `main_dst`, `git log main_dst..branch` empty. Gate 0/1 clean: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, ran from the `#558` driver pin root detached and clean at `9467ef183`, running skill **byte-identical to origin** (`cmp` rc=0), pin root **4 behind** `origin/dev` so all mission state was read from origin blobs and this record is written in a worktree branched from `origin/dev`; dev's last **settled** commit `9467ef183` green (**16** checks, **0** not-green) while the tip `d22681e27` is **mid-flight** (15 pending) and is recorded as *unverified, not green*; `5f9f4ba4f` `checks=0` is the documented intra-push zero, not an anomaly — and dev is V1's to own regardless. **0** human directives on `#743` since the watermark (of 8 comments), ledger valid at 3 rows with **1 OPEN** (`D-MOTOKO-FMT-1`). No died-mid-flight traces: five motoko worktrees `dirty=0`; `#695`/`#613` are V1's; the stale source clone's 7 paths unchanged since iteration 11 adjudicated them superseded. Weekly sweep and rotation **both not due** (`#743` created `2026-08-17T05:48:23Z`, after the Monday-07:00 **local** boundary; 8 comments < 80). Routing: controller `claude:claude-opus-5` only; **no designer, planner, executor, evaluator or quorum spawned** — a base-freshness re-measurement of a filed artifact has no doc to design, no plan to write and nothing to judge. Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no `rig.lock`; `make quick-install` deliberately NOT run (shared-write guardrail) — nothing here tests local AILANG behaviour. Gates on **darwin/arm64 only** (rule 3b(viii)). Gate-5: **NO skill edit** — the gap (*every freshness rule here is scoped to a row you are about to pick; none covers an artifact a LANDED row already published to a third party whose base moves*) is **instance 1**, pre-registered at the ≥2-friction bar rather than written on one datapoint; the correction goes to queue row 5 in the process lane instead.

## STATUS 2026-08-18 — ITERATION 11 COMPLETE: **THE QUEUE'S BOUNDED WAIT HAD ALREADY EXPIRED — ARNI ANSWERED #97 FIVE DAYS AGO, AND THE ROW TRACKING HIM WAS RE-ASSERTING A MEASUREMENT TAKEN BEFORE HIS REPLY.** Pick was queue item **5**, not the untagged head 7, because Gate 2's blocker re-verification found item 5's declared blocker **dead**. The row read *"Remaining: the bounded wait only — still zero `arniwesth` events on #97; 2026-08-27 stands"*; measured this iteration, `arniwesth` commented at **2026-08-13T18:45:54Z**, control firing (`commenter:arniwesth` → **34** issues/PRs). **THE FINDING: a blocker row can re-assert a stale measurement as a fresh one, and its own text cannot tell you which it is.** Iteration 3 measured the zero on 2026-08-13 and was right; iteration 4 wrote *"still zero"* on 08-14, **after** the comment, and iterations 5–10 carried it forward. The word *still* reads as a re-check and is a transcription (rule 3b(v)(b)). Sharper: the 08-27 timebox iteration 3 added as the **remedy** for an unbounded wait is what hid the wait ending, because a timebox invites checking the clock instead of the predicate. Five days of a nine-day window, on a row sitting above two ungated items. **WHAT ARNI SAID, and it is better than the row assumed:** *"Agreed on all four points"*; close `#97` as superseded (calibration, elision ladder and system-prefix pinning all covered more strongly by `main_dst`/`#154`, nothing to revive); **the output-headroom concern is still valid**; and *"Please go ahead with the separate issue against `main_dst`"* — with scope specified: the effective input budget for **both** the pre-step chain and the final seal, raw context limit retained for telemetry, regression coverage for 262,144/65,536 **and** the unknown-/small-limit fail-open. So iteration 3's original precondition, declared FALSE and replaced by the timebox, is now **satisfied on its own terms**. **DELIVERED — upstream [arniwesth/motoko_agent#165](https://github.com/arniwesth/motoko_agent/issues/165)**, every load-bearing claim re-derived first-party at `main_dst@6c06b08` rather than inherited (rule 3b(v)); the base was checked, not assumed — `origin/main_dst` is **still exactly `6c06b08`**, **0** commits since our V27–V29 rows. `session.ail:2561` passes the seal the **raw** `context_limit` while `:2556` gives extensions `context_limit - pinned_tokens` — so the seal is the one consumer still handed the unreduced number, and the precedent for the fix is one line above the call site. `phase_vocab.ail:145-155` + `compaction.ail:30` show the refusal is **input-only** at `exhaustion_pct() = 95`. The ladder cannot close the gap: `compact_for_pre_step` (`compaction_structural.ail:170`) targets `result_target_pct() = 70` (`:30`,`:16`) but every tier routes through `elide_walk` (`:97-113`), which rewrites a message **only** at `m.role == "tool"` (`:101`), so a large **user** message is invisible to all four and the hook returns the unconditional `Compacted(floor, …)` (`:191`) — a payload it has already determined does not fit, with no error. And `compaction.ail:25-28` returns **0%** when `limit == 0`, so an unknown model makes the seal compare `0 >= 95` and send unconditionally: the fail-open arm Arni named, and the reason a percentage-shaped reserve cannot inherit its own zero case. **THE R8 PROBE WAS RE-RUN, NOT QUOTED** (`tools/motoko/r8_headroom_band.ail`, fresh `~/dev/mk-r8-main-dst` at `6c06b08`; `ailang lock` rewrote the 19 dev-container absolute paths — `/workspaces` **0**, control `"path":` **19**). Every number reproduced: arm **A** ladder `tier=floor keep_last=1` at **79%**, seal **`Ok` — SENDS**, headroom **54,905 < 65,536**; controls **B** (`PassThrough`/`Ok`, 259,642) and **C** (`Err(SealExhausted)` at 157%) both fire, so that `Ok` is a real permission and not a dead gate. New on the re-run and carried into the issue: the ladder removed **2,061 of 208,980** segment tokens (~1%), and the extension saw `ext_limit = 261,824` while the seal saw the raw **262,144** — the asymmetry as one line of output rather than as an argument. **`#97` CLOSED with the verdict**, comment-then-close, comment count asserted **2 → 3**. **RULED OUT:** *the charter's "zero production callers" for `try_emergency_compaction_with_limit`* — it has exactly one caller (`compact_step_with_limit:137`), so the bare claim is false as written; the substance holds and upstream says so itself (`006_compactor_strategy/PLAN-compactor-strategy.md:75` records it **test-only**, single caller), so the issue says "off-path" citing their line rather than repeating our overstatement. Also ruled out: *an output-headroom issue already exists upstream* (all **20** enumerated, none is one; `#31`/`#49` cited as related, titles verified), and *`main_dst` moved under us* (**0** commits since `6c06b08`). Gate 0/1 clean: kill switch armed, `gh` on `sunholo-voight-kampff`, tripwire **CLEAN**, ran from the `#558` driver pin root detached and clean at `1350d308f` == `origin/dev`, running skill **byte-identical to origin** (`cmp` rc=0 with a rc=1 control), dev **verified green not merely un-red** — **16** exact-SHA checks, **0** not-green, `test: success` present, `runs_total=2` — **0** human directives on `#743` since the watermark, ledger valid at 3 rows with **1 OPEN** (`D-MOTOKO-FMT-1`, unanswered), inbox 5 unread carrying no request or bug against motoko, no died-mid-flight traces (four motoko worktrees `dirty=0`; `#695`/`#613` are V1's; the stale `ailang-motoko` checkout's 7 paths re-verified **superseded not orphaned** — 3 of 4 blobs verbatim in `de0e41099`, the 4th the decision-ledger block already in origin's charter). Weekly sweep and rotation **both not due**. Routing: controller `claude:claude-opus-5` only; **no designer, planner, executor, evaluator or quorum spawned** — an upstream issue assembled from measurements the controller took itself has no doc to design, no plan to write and nothing to judge; quorum-at-pick N/A (item 5 has no doc of its own; its case lives in already-reviewed V27–V29 and R8). Rotation pointer untouched at `claude:claude-fable-5`. Metered **$0.00** of $5. No GPU, no `rig.lock`; `make quick-install` deliberately NOT run (shared-write guardrail) — PATH binary `v0.33.1-103-g0002c9b0b-dirty`, stated because the R8 re-run used it. Gates on **darwin/arm64 only** (rule 3b(viii)). Gate-5: **one skill edit** — Gate 2, a blocker row's freshness claim.


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
6. [**UN-PARKED + TRACE DONE 2026-08-19 · iteration 13** — `D-MOTOKO-FMT-1` RESOLVED **precondition**
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
   instrument's direction in either quorum round]
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
