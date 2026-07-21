# V1 Mission — work the backlog to a v1.0.0 release

**Type**: Long-running mission (peer of [motoko-mission.md](motoko-mission.md)); advanced by a
scheduled nightly outer loop on the always-on rig, coordinated by Fable.
**North star**: Ship AILANG v1.0.0 — a release whose bar is *written down, met, and verified*,
with the backlog worked through the honed inner loop (design-doc → sprint-plan → execute → evaluate)
rather than ad-hoc sessions.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration. **Scheduling: launchd `dev.ailang.mission-control`** (CONTINUOUS since
2026-07-10 per Mark: StartInterval 2h + overlap guard = back-to-back iterations, ≤2h idle; was
22:00-nightly for the first supervised runs) behind the billing guard — API keys are stripped from the environment
(subscription-or-nothing by construction) and a cheap auth probe runs first: keychain OAuth
suffices while the rig is logged in (verified 2026-07-10); `CLAUDE_CODE_OAUTH_TOKEN` in
secrets.env is an optional belt-and-braces for post-reboot login screens. Probe failure refuses
loudly with zero spend. The Claude Code
scheduled-tasks path was TESTED AND RULED OUT for this job (2026-07-10 canary): that system is
desktop-side — tasks landed on /Users/mark (Mark's machine, not the rig) and a probe task never
dispatched even there (a June one-time task was also found a month overdue). Wrong machine +
unreliable dispatch → launchd is primary, not fallback.
**Log**: [v1-mission-log.md](v1-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue
[#329](https://github.com/sunholo-data/ailang/issues/329) — every iteration posts its morning
report there as a comment (Mark follows by email via issue subscription, no Claude login
needed); driver crashes post there too.

---

## STATUS (rotation rule — added 2026-07-14, Fable-quota diet)

Newest **3** STATUS stamps live here; older ones move to
[v1-mission-status-archive.md](v1-mission-status-archive.md). **Loop: at Gate 4, after
adding your stamp, move the now-4th stamp to the TOP of the archive file.** Rationale:
every iteration re-reads this charter — 30+ stamps were ~500 lines of history tax per
read, on the scarcest model budget. The append-only history lives in the log + archive.

## STATUS 2026-07-20 — ITERATION 65: queue pick **m-ailang-fmt-adoption EXECUTED + LANDED** — PR #415 squash `b787bb98f`; executor (opus, worktree) M1–M3, evaluator (sonnet, generator≠judge) PASS **89/100 r1**; `ailang fmt` now discoverable (teaching prompt v0.16.3) + opt-in auto-format PostToolUse hook with the Mark-approved SIGTERM→grace→SIGKILL escalation; `metered=$0.00`

iter-64 produced the ready 3-milestone plan → this iteration executed it (Gate-3 plan→execute). Preflight CLEAN (killswitch armed, billing **CLEAN** both keys empty, gh `sunholo-voight-kampff`, main tree carried 3 pre-existing dirty generated docs — left untouched, Critical Principle 0; no `MERGE_HEAD`; issue **#399**, 40 comments <80, today 07-20 00:10 local is Monday but BEFORE the 07:00 quota-reset boundary → most-recent-passed boundary is 07-13, #399 created after it → **no rotation**; no new Mark comment on #399 OR predecessor #329 since watermark `2026-07-19T07:52:58Z`). Local dev == origin/dev `5afa9a1e1`; dev CI CI/Build/Docs all `success` @ `5afa9a1e1`. Inbox: 1 `eval-suite` informational → acked. **Gate-2**: NOT landed/ghost — only plan commit `52ed0204c`, doc in `planned/`, no branch/PR; the intervening `5afa9a1e1`/K3 commits are a separate eval stream (CI green, not the fmt loop). Quorum satisfied by Mark decision (no re-quorum). **EXECUTOR (opus, pinned Agent, isolated worktree)** shipped M1 prompt v0.16.3 (append-only, hashed) → M2 `formatter.md` Adoption section + dev-workflow cross-link → M3 `make fmt-check-ail` (renamed off the pre-existing Go `fmt-check` gate; `make ci` byte-identical) + opt-in `format_ail.sh` w/ SIGKILL escalation. 5 commits `7de65dc4b`→`f1546e4a7`. **EVALUATOR (sonnet, generator≠judge)** PASS **89/100 r1** — independently re-ran hook tests (b/c/d incl. the SIGTERM-ignoring-stub reap in ~11s) + append-only proof + `make ci` byte-identical + `make test` green; 2 MINOR gaps (CHANGELOG, doc-move) = controller finalization. **Controller DISPROVED the executor's flagged "Docusaurus build failure"**: it was the skipped CI-only `docs/scripts/sync-registry.sh` gen step (generates untracked `packages/sunholo/*` + sidebar entries) + stale `.docusaurus` cache — after the CI-faithful step the build is green with the sprint edits. Controller finalized (CHANGELOG + doc→`implemented/v0_30_0/` + noise-revert `a813fe1b2`), PR #415 auto-merge squash, Gate-3b bounded-poll (2 windows, 0 failures) → merged `b787bb98f`. `metered=$0.00` (all quota-bucket; no metered lane fired; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 70.

## STATUS 2026-07-19 — ITERATION 64: queue pick **m-ailang-fmt-adoption** (fmt follow-up; phase2 gate now SATISFIED) → **routed to sprint-planner (opus); 3-milestone plan produced (teaching prompt v0.16.3 · `make fmt-check` · opt-in PostToolUse hook w/ Mark-approved SIGKILL escalation), READY FOR EXECUTOR** — no re-quorum (Mark-approved iters 60/62); exit-code contract VERIFIED matches (0/1/2/3); `metered=$0.00`

Iter-63 landed phase2 → its "⛔ Execution Gate" on adoption is now cleared (Gate-3 advances one stage: this iteration = plan; next = execute, per the iter-62→63 cadence). Preflight CLEAN (killswitch armed, billing **CLEAN** both Anthropic keys empty, gh `sunholo-voight-kampff`, tree clean, no MERGE_HEAD; issue **#399**, 37 comments <80, next-Monday boundary 07-20 is tomorrow → no rotation; **no new Mark comment** since watermark `2026-07-19T07:52:58Z`). Local dev == origin/dev `06513167e`; dev CI CI/Build/Docs all `success` @ `06513167e`. Inbox: 1 `eval-suite` informational (started, no regression) → acked. **Gate-2 reality check:** NOT landed, NOT a ghost — fresh `git fetch`; only doc-creation commits exist (`ad14dfc19`/`72996baaa`), doc still in `planned/`, no `sprint/*` branch, no adoption sprint JSON. Quorum: 3 artifacts present + Mark approved the last SIGKILL fix with **no re-quorum** → QUORUM-AT-PICK satisfied, no spawn. **PLANNER (opus, pinned Agent, planning-only)** produced `.ailang/state/sprints/sprint_M-AILANG-FMT-ADOPTION.json` + `m-ailang-fmt-adoption-sprint-plan.md` — 3 milestones, **1.25d (~130 LOC)**, LOW risk/conflict (prompts/docs/build-glue/hooks; zero compiler/runtime change): M1 teaching prompt v0.16.3 (append-only, hashed registry) · M2 docs + CLI audit (`formatter.md` adoption section) · M3 `make fmt-check` (NOT in `make ci`) + opt-in `format_ail.sh` PostToolUse hook landing the **Mark-approved SIGTERM→grace→SIGKILL** escalation (doc snippet is the pre-fix version — must NOT copy verbatim). Stale-premise flags folded in: exit-code contract **VERIFIED matches** the hook's 3-vs-2 silent/surface split (`0`=ok, `1`=`--check` drift, `2`=operational incl. 15.28% inline-interior refusal, `3`=parse) → no execution-time premise to resolve; phase2 gate sections historical. `metered=$0.00` (planner = opus/OAuth quota bucket; no metered lane fired; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 69.

## STATUS 2026-07-19 — ITERATION 63: queue pick **m-ailang-fmt-phase2 EXECUTED + LANDED** — PR #414 squash `3815ba617`; executor (opus, worktree) shipped M0–M3, evaluator (sonnet, generator≠judge) PASS **78/100 r1**; `ailang fmt` now preserves comments losslessly on ~85% of the corpus, fail-closed (never lossy) on the rest; `metered=$0.00`

The plan produced iter-62 was ready → this iteration executed it (Gate-3 advances one stage: plan→execute). Preflight CLEAN (killswitch armed, billing **CLEAN**, gh `sunholo-voight-kampff`, tree clean, no MERGE_HEAD; issue **#399**; watermark advanced to `2026-07-19T07:52:58Z` — the 3 Mark comments (02:36/06:14/07:52) were all already processed iters 58–60, no new directive). Local dev == origin/dev `f48e268e5`; dev CI CI/Build/Docs all green @ `f48e268e5`. Inbox: 4 `eval-suite` informational (partial/started, no regression) → acked. **EXECUTOR (opus, pinned Agent, isolated worktree `ailang-wt-fmt-phase2`)** shipped M0 printer child-list audit (verified 39-site inventory folded into the doc) → M1 lossless collector + token-anchored envelope + interpolation FAIL-CLOSED carve-out → M2 deterministic attachment (rules 1–5) + emission → M3 marker/corpus/property gates + `HasComments` refusal removal + exit-split (3=parse, 2=operational) + docs. 6 commits `83f7ebf23`→`b29e871c4`. **EVALUATOR (sonnet, generator≠judge vs opus executor)** PASS **78/100 r1** — ruled BOTH deviations acceptable: 15.28% (59/386) inline-interior refusal is fail-closed/never-lossy; 28 `properties[...]` round-trip bugs verified pre-existing (fail comment-free on dev too). Controller independently: rebuilt/tested the worktree, fixed 3 sprint-introduced lint issues (`fe236572c`), moved doc → `implemented/v0_30_0/`, opened PR #414 (auto-merge squash), Gate-3b bounded-polled CI green → merged `3815ba617`. **Corpus gate V22**: 386 parse-valid → 327 formatted, 0 comment-loss, 0 Phase-2 regressions, interp-refusal 0/386, idempotence 299/299. Two follow-ups queued (inline-interior reduction; properties-printer r/t). `metered=$0.00` (executor + evaluator + controller all quota-bucket; no metered lane fired; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 68.

## CURRENT GOAL

1. **Iteration 0 (definition)**: write the v1.0 bar (see "The v1.0 bar" below — draft to be
   ratified by Mark), re-score all 93+ planned docs against it into: `required-for-v1` /
   `nice-for-v1` / `post-v1`. Output: updated folder assignments + ordered queue in this doc.
2. **Then**: work the queue P0-first through the inner loop, one sprint-sized item per iteration,
   recording routing evidence every time.

## The v1.0 bar — v2, PRODUCT-SHAPED (RATIFIED 2026-07-11, Mark; supersedes the 2026-07-10 hygiene bar)

**The 1.0 claim**: ***the verified AI-orchestration language*** — an AI author gets a
verified-correct program at the lowest cost, and AI orchestration is type-checked. Derived from
[m-fable-strategy-review](planned/m-fable-strategy-review.md) (Design Freeze items 1+2 ratified
by Mark 2026-07-11: cost-per-success is the headline KPI; orchestration is the vertical. Item 3,
trace publication, stays deferred — post-v1).

**The cutoff rule**: a design doc gates v1.0 **only if it serves an open clause below.**
Everything else ships on the normal v0.2x road or is post-v1 — regardless of folder history.
The v1 hygiene bar (2026-07-10) is absorbed: its clauses are 1–2 below, both essentially done.

1. **STABLE** ✅ — the 1.x surface promise (docs/docs/reference/stability.md, iteration 5;
   tier-assignment ratification parked for Mark).
2. **SOUND** — zero P0s ✅ (all four closed, iterations 1–4); residue: m-check-strict-fallbacks,
   m-bytecode-vm-parity-bugs (both ≤2d, queued).
3. **ACCESSIBLE TO THE FLEET TIER** (strategy R1+R4): the finite, documented mid-tier footgun
   list burned down — the 3 parser/type inconsistencies fixed (match-in-HOF-lambda parse,
   polymorphic-arithmetic panic, arity call-style diagnostic), m-syntax-ai-forgiving landed
   (kills the ~32% small-model failure class), and the teaching prompt ≤1,500 lines with a
   rig-A/B showing no pass-rate loss (R3.1 measures the curve first; the deletion pass stays
   gated on replacement diagnostics landing, per m-diagnostic-coverage's deferred section).
   **Gate = this finite work.** The sonnet-class ≥ −5pts outcome is measured and published at
   release, NOT blocking (per Mark: partially vendor-dependent).
4. **ORCHESTRATION FLAGSHIP** (R6 + R7 + effect refinement): the four effect sprints (public
   docs promise); a **verified multi-step AI pipeline** as the flagship example (typed LLM calls
   + budgets + secret-flow + replay) with orchestration benchmarks promoted into the default
   rotation and README/site positioning led by it; **linear-time regex + URL-parse builtins**
   (both verified absent — an orchestration 1.0 without them is a credibility hole).
5. **COST CREDIBILITY** (R3): the dashboard headline KPI flips to **cost-per-verified-success
   vs Python, per tier**, and v1.0 ships with the measured baseline + trajectory. The ≤3×
   zero-shot / ≤1.5× agent targets are the tracked post-1.0 trajectory, NOT release gates.

## How the mission runs (each iteration — codified in the mission-control skill)

1. **OBSERVE** — read this doc's backlog + last log entry + agent inbox + eval health. Deterministic, cheap.
2. **PICK** — top open queue item per the ordering policy. **Verify against repo reality first**
   (git log + code + tests), never trust a status header — stale-status docs are how we shipped
   M-EVAL-BENCH-UI twice (2026-07-10 lesson: doc said Planned, all 4 milestones were long done).
3. **ROUTE + EXECUTE** — through the honed inner loop with the model routing policy below:
   design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator. Sprint work runs in
   an isolated git worktree (concurrent-agent safety). Max 3 evaluator rounds, then park as
   `needs-human-review` and move on.
4. **RECORD** — append a log entry (fixed template in v1-mission-log.md): what shipped, evaluator
   score, routing evidence row, ruled-out ledger additions, next.
5. **RETRO** — route observed friction into exactly one lane: **skill fix** (edit the offending
   SKILL.md — max ONE skill edit per iteration, each traced to ≥2 recorded frictions), **process
   fix** (edit this doc), or **backlog item** (new/re-prioritized design doc). Then send the
   morning report to controlplane.

## Model routing policy (evidence-updated, not vibes)

| Role | Model | Why / evidence |
|---|---|---|
| Mission controller (this loop: triage, pick, judge, retro) | **Opus** — opus-first PREFS since 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"; the long orchestration session is mechanical and was the residual Fable drain even after M1a). Fable = emergency fallback only | The 07-14 Fable revert burned the weekly bucket at 2h cadence; orchestration doesn't need the top tier |
| Design docs (create/review) | **ROTATION across top-of-line models (Mark 2026-07-17)**: `claude:claude-fable-5` (via `claude-sub`) ⇄ `codex:gpt-5.6-sol`. **gemini caveat (iter 53):** G4 clone-over-egress LIVE-LANDED, so gemini is fleet-ready — but as an in-sandbox **evaluator**, not a designer: `CapRemoteSandbox` means it cannot edit a worktree, so a designer spawn can't write the doc without the text-bridge (unwired). gemini's designer-rotation entry is therefore PARKED for Mark's fleet-role ratification (evaluator recommended). Each new-doc iteration takes the next designer in rotation; record `(designer, quorum outcome)` in the evidence row | Every design passes the QUORUM regardless of author — the quorum is the quality gate, so authorship diversity is free comparative signal on which frontier model designs best for AILANG. Fires only when a doc is created/revised |
| Sprint planning | **Opus** (claude-opus-4-8) | Plan quality determined execution success historically |
| Sprint execution | **Opus** — the default, per Mark 2026-07-10 | Sonnet execution was a false economy (needed corrections); also `dev-cycle.md` had silently pinned sonnet |
| Sprint evaluation | **Sonnet** — `$MISSION_EVALUATOR_MODEL`-PINNED sub-agent (default changed fable→sonnet 2026-07-16, Mark directive #399; see below). generator≠judge holds STRUCTURALLY (sonnet ≠ the opus executor pin) AND is now ENFORCEABLE (sonnet is an Agent-tool alias; fable was not — F1 — so the fable default re-routed to sonnet every iteration anyway: 31, 36) | Behavioral independence (fresh sub-agent, re-runs tests, adversarial probes) retained on top |

> **✅ Evaluation-independence (RESOLVED 2026-07-16, iteration 38):** the evaluator default is now
> **Sonnet** — a distinct model from the Opus executor, so generator≠judge model-diversity is
> restored (the 2026-07-11 "Opus-evaluates-Opus rubber-stamp risk" is gone) AND it is *enforceable*
> (sonnet is an Agent-tool pin; the old `fable` default was not — F1 — so it silently re-routed to
> sonnet every iteration anyway; this makes that the standing state, not a per-iteration patch).
> Behavioral value (independent test re-runs, cross-history non-vacuity, distinct-sample recounts)
> is unchanged. Fable is retired from the every-iteration evaluator slot to protect the weekly quota
> (it fires every iteration, unlike the designer which fires only on new docs).
>
> **⚠ CORRECTION (2026-07-16 evening, Mark + interactive session): "Fable quota-exhausted until
> 2026-08-01" was a MISDIAGNOSIS — OAuth Fable was available the whole time.** The tell: OAuth
> buckets reset **weekly Monday 07:00**; an "until the 1st" date is the **API key's monthly
> cycle**. Root cause: `~/.zshenv` sources `secrets.env`, so every tool shell re-exports
> `ANTHROPIC_API_KEY`; nested `claude -p` calls (the `claude:` CLI lane) therefore billed the
> METERED API — iteration 37's fable designer+evaluator runs were API-billed $, and the key's cap
> then produced the fake "Fable exhausted" error. Fixed in the skill: every nested `claude` call
> now strips the keys at the call-site (`env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN`).
> **The `claude:claude-fable-5` designer lane is AVAILABLE again — do not treat Fable as gone
> until 08-01.** Any future "quota" error naming a reset date that is not a Monday = you are on
> the API key; fix the leak, don't fall back.
| Mechanical tasks (doc moves, regen, banking) | Sonnet allowed | Only with deterministic verification; promotion beyond this requires evidence |

**Evidence rule**: every sprint's log entry records `(model, task class, evaluator round-1 score,
rounds-to-pass, corrections)`. A routing change (either direction) requires ≥3 data points and is
made in RETRO, recorded here with a dated stamp.

> **ENFORCED 2026-07-15 (m-mission-agentic-provider-routing M1):** this table is no longer prose.
> The driver exports `$MISSION_PLANNER_MODEL` / `$MISSION_EXECUTOR_MODEL` / `$MISSION_EVALUATOR_MODEL`
> and mission-control Gate 3 spawns each heavy role as a model-PINNED sub-agent (the controller
> session runs `$MODEL` only). **Before M1, every role inherited the single session model → 100%
> Fable burn** (the driver had been Fable-first since 07-14). Execution now bills the executor pin
> (Opus), not the controller (Fable); generator≠judge is restored (Fable evaluator ≠ Opus executor).
> M2 extends the evidence rows with `(provider, agent, $/quota)`; M3 A/Bs the **sprint-planner
> down-tier** (kept at Opus until ≥3 datapoints — do NOT lower it on this hypothesis alone).
> Cross-provider AGENT executors (codex/motoko/managed_agents) ride the same env once fleet Phase C
> resolves a value like `codex:gpt-5.6` in the spawn.
>
> **AMENDED 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"):** the
> controller session itself was the residual Fable drain after M1a (a ≤6h mostly-mechanical
> orchestration session on the scarcest model). Driver PREFS are now **opus-first**
> (`claude-opus-4-8,claude-fable-5`; Fable = emergency fallback only) and design-doc-creator moved
> from inline to a **`$MISSION_DESIGNER_MODEL`-pinned sub-agent (fable)**. Net Fable spend per
> iteration = ~~two bounded sub-agents: designer (only when a new doc is needed) + evaluator~~
> ONE bounded sub-agent: the **designer** only (fires only when a new doc is needed). The evaluator
> moved OFF Fable to **sonnet** in iteration 38 (below) — this also RESOLVES the iteration-36/37
> inconsistency between this clause and the "evaluator→sonnet unless ≥3 datapoints" rule (Mark's
> #399 directive settles it: not fable).
>
> **AMENDED 2026-07-16 iteration 38 (Mark directive #399: "once we have gemini via managed agents
> and openai we can use one of those instead for evaluator? so default can be gemini (if able to git
> clone the codebase etc)? otherwise sonnet-5"):** evaluator default moved **fable → sonnet**.
> gemini (managed_agents) — Mark's *preferred* default — is NOT viable as the evaluator today, on
> two independent counts VERIFIED this iteration: **(1) architectural (code-proven)** — the
> managed_agents request body carries only `Directive`+`SystemPrompt` over a server-side
> `CapRemoteSandbox` (`internal/executor/managed_agents/managed_agents.go:164`); there is no repo
> upload, so the agent cannot see the sprint's UNCOMMITTED worktree changes nor re-run local tests
> (at most it could `git clone` the *public* origin/dev, which lacks the changes) — exactly the
> "if able to git clone the codebase" gap Mark flagged; **(2) operational (live-observed)** — a
> bounded `ailang exec gemini` probe timed out (`http2 timeout awaiting response headers`), same
> class as iterations 36-37. Per Mark's own ladder this resolves to **sonnet-5**. gemini-as-evaluator
> is a queued follow-up (**m-gemini-evaluator-diff-bridge**): needs a bridge that ships the sprint
> diff + changed files into the directive text AND the Vertex backend returning reliably. NOTE:
> **codex (openai)** is a viable local distinct-provider evaluator alternative (it runs a sandboxed
> local CLI → CAN read the worktree + re-run tests; openai≠anthropic satisfies generator≠judge) —
> but Mark's stated default ladder is gemini→sonnet-5, so sonnet is the default; codex-as-evaluator
> requires the executor NOT be codex (generator≠judge) and stays opt-in.

### Right-sizing table — the (provider, agent, tier) hypothesis (M2)

Landed 2026-07-16 (m-mission-agentic-provider-routing M2). This is the *hypothesis* that the routing
evidence rows below test — updated by the ≥3-datapoint evidence rule, never by vibes. Canonical source:
[design_docs/planned/v0_30_0/m-mission-agentic-provider-routing.md](planned/v0_30_0/m-mission-agentic-provider-routing.md).

| Role | Agentic? | Needs | Tier hypothesis | Agent candidates |
|---|---|---|---|---|
| Controller (pick/judge/retro) | agent (claude-code) | orchestration judgment | **mid** | claude-code (home harness) |
| Design-doc-creator | agent (`check` in loop) | deep spec reasoning (highest leverage) | **strong** | strong claude/codex + live quorum |
| **Sprint-planner** | agent-capable | decompose a quorum-reviewed doc | **MID (down-tier)** — kept at Opus until M3's ≥3-datapoint A/B | mid codex/gemini/motoko |
| Sprint-executor | AGENT (heavy) | tool-using coding | **strong AGENT** (not just a model) | **codex / motoko / claude**; motoko may over-perform on AILANG (M1b wired codex) |
| Sprint-evaluator | AGENT (re-runs tests) | behavioral verification | **mid**, distinct provider from executor | gemini/codex ≠ executor |
| Mechanical (moves/regen) | no | deterministic | **low / local** | local-GPU (Phase D) |

> The model-routing table above (Opus-first) is the CURRENT enforced assignment; this right-sizing
> table is the tier *hypothesis* those assignments are converging toward as evidence accrues. Where
> the two differ (e.g. controller runs Opus today but the hypothesis is mid-tier), the gap is a
> deliberate, evidence-gated decision — a routing change requires the ≥3-datapoint rule.

## Rig integration — the two-tier rule

`rig.lock` (`~/.ailang/state/rig.lock.d`) is a **GPU mutex, nothing more** (Mark, 2026-07-10).

1. **Default iteration (cloud models: Fable/Opus coding, `make test`, git): NEVER touches
   rig.lock.** CPU/disk co-tenancy with the eval rotation is fine; the loop runs 24/7 without
   starving the rotation and vice versa.
2. **GPU-touching steps only** (a sprint whose acceptance includes local-model validation, wire
   diagnostics, anything driving ollama): `rig_lock_acquire wait` for **that step only** — never
   held across a whole sprint. Same discipline as `os-rotation-filler.sh`.

Hygiene: a sprint must not *accidentally* reach the GPU (the port-8080-zombie class). "Does this
step touch the GPU?" is an explicit routing question in the skill, not an accident of what a test
invokes.

## Guardrails (the loop may not…)

- **No releases** by the loop — but a rolling release cadence (Mark, 2026-07-12): the loop lands
  to `dev` continuously and never cuts a release; **Mark snapshots interim releases (v0.30.x,
  v0.31.x…) as needed**, each carrying whatever's accumulated. **v1.0.0 is a MILESTONE declared
  when all five bar clauses are satisfied — not a single big-bang release.** Implications: (1) dev
  must stay release-ready at EVERY commit — the "Dev stays GREEN" guardrail already enforces this
  and it is now load-bearing (any commit may become a release point); (2) each iteration's #329
  report should note when it CLOSES a bar clause (e.g. "clause 3 footgun burn-down: N of M
  landed"), so Mark can watch the bar fill and time the v1.0 call; (3) a version bump mid-mission
  is expected, not a stop signal — the loop already handled v0.29.0/.1/.2 landing between iterations.
- **No pushes without account check** (`gh auth status` → `sunholo-voight-kampff`).
- **No work on a dirty main worktree** — sprints run in coordinator-managed worktrees; the
  controller session itself is read-mostly + doc edits.
- **Budgeted**: hard wall-clock kill in the driver (default 6h); one backlog item per iteration.
- **Kill switch**: `touch ~/.ailang/state/mission-control.disabled` (checked in preflight) or
  `launchctl unload ~/Library/LaunchAgents/dev.ailang.mission-control.plist`.
- **Subscription billing only** (2026-07-10): the nightly bills the Claude subscription via
  `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, stored in secrets.env) — the driver strips
  `ANTHROPIC_API_KEY` and refuses to start without the token. The first kickstarted run billed
  ~13 min of API credits before this was caught; never again.
- **Escalation**: evaluator `needs-human-review`, merge conflicts, or any guardrail trip →
  `ailang messages send controlplane`, park the item, pick the next; never force through.
- **Skill edits**: max one per iteration, ≥2 recorded frictions each, called out in the morning
  report (git history is the rollback).
- **Dev stays GREEN** (2026-07-10, Mark): an item is not [LANDED] until remote CI passes on its
  merge commit (Gate 3b — local gates miss fmt-check/govulncheck/file-sizes/docs build), and a
  red dev CI outranks the queue at OBSERVE, including time-based reds from newly published vuln
  advisories.

## Backlog ordering policy

1. Open **P0s** first (list above), oldest-known-risk first.
2. **Unblockers** — items other queued items depend on (e.g. m-effect-row-poly-params blocks
   sunholo/demos).
3. **P1 by impact-per-day** (the census has estimates; prefer ≤3-day items to keep iterations
   sprint-sized).
4. **Strategic multi-week items enter only after decomposition** into sprint-sized design docs
   (a decomposition is itself a valid iteration deliverable).
5. Anything re-scored `post-v1` in iteration 0 leaves the queue.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**Required-for-v1 (the bar's critical path):**

1. [LANDED 2026-07-10] m-named-test-blocks closeout (iteration 1a; deontic criterion deferred,
   package absent locally)
2. [LANDED 2026-07-10] m-typeenv-sub-fix (iteration 1b: RESOLVED — pre-closed by adjacent
   M-TYPE-LIST-SOUND work, regression-guarded, eval 92/100, merge f59421ac8)
3. [LANDED 2026-07-10] m-feedback-triage-gate (iteration 2: full loop headless, eval 93/100
   PASS round 1, merge 40f1cdc3f, remote CI green on dev post-merge; gate logic complete +
   merged, off by default — production activation gated on the next item)
4. [LANDED 2026-07-10] m-feedback-gate-cloud-adapter (iteration 3: full loop headless, round-1
   eval FAIL → round-2 PASS 97/100, merge 842d7d501, dev CI fully green on 4c22032de; gate
   code complete incl. cloud adapters, OFF by default — production enablement is a HUMAN ops
   task: sibling-repo terraform TTL + ANTHROPIC_API_KEY secret, then DRY_RUN=1 week 1)
5. [LANDED 2026-07-10] m-diagnostic-coverage (iteration 4: M1–M3 found pre-shipped 2026-07-09
   under a stale "Planned" status; remainder sprint M-DIAG-FIXTURE-PROMOTION promoted 4 rows
   to covered — 7 CI fixtures across 6 footgun rows — eval PASS 96/100 round 1, PR #336 →
   fe807aac8, dev CI green per-workflow. DEFERRED, rationale in doc: deletion pass + rig A/B
   until deletable surface ≥ 100 lines; PARKED for human: haiku causal re-run, API-billed)
6. [LANDED 2026-07-10] m-v1-stability-promise (iteration 5: FULL loop headless round-1 clean —
   Fable design doc → Opus plan (caught 2 premise errors: 42 modules not 39; LIMITATIONS
   double-maintained + diverged, both copies fixed) → Opus execute in worktree → Fable eval
   PASS 96/100 round 1. Stability page docs/docs/reference/stability.md (3 tiers, full stdlib +
   CLI tables), both LIMITATIONS files live-verified at HEAD, 4 website vX-promises retracted,
   PR #337 → fcccd7208, dev CI green per-workflow. PARKED for human at RELEASE gate: tier-
   assignment ratification — ⚠ proposed: std/net, crypto, jwt, xml, zip, process, CLI
   watch/serve-api)
7. [LANDED 2026-07-11] m-effect-refinement **decomposition** (iteration 6: repo-verified phase
   census — P1/P2 + AI port shipped v0.15.0 under the parent's stale "Planned"; P7 CryptoRand
   never existed (m-cryptorand.md swept to implemented/ in error — header corrected); P6 routed
   OUT to M-ENTROPY. Remaining ~64h split into 4 sprint docs (below, items 9/12/13/14) with live-
   verified premises; parent doc now the umbrella. BONUS finding: the public guide's "typechecker
   rejects unknown values" is FALSE (`Rand[mode=banana]` passes check) — interim accuracy note
   shipped, enforcement is sprint 1)
8. [LANDED 2026-07-11] m-eval-frontier-tier (iteration 7: full loop headless, round-1 clean —
   Opus plan (9 discrepancies) → Opus execute (frontier tier + 8 re-tiered + prefix_line
   structural grader + 7 core→stretch demotions via 4-dim rule from banked data) → Fable eval
   PASS 96/100 round 1 w/ independent distinct-sample recount. PR #339 → 0515578ae, dev CI
   green per-workflow. PARKED for human: frontier-failure validation of the 8 (API-billed —
   each must fail ≥1 frontier model or demote back per CURATION.md §5) + 4 remaining sketches)
*(Queue re-derived 2026-07-11 from bar v2 — clause tag on every open item. NEW-DOC items start
with design-doc-creator; existing-doc items start at reality-check.)*

9. [LANDED 2026-07-11] m-effect-mode-validation (iteration 8: full loop headless, round-1 clean —
   Opus plan (2 discrepancies: bridge carries no params, scope-reduced; EFF_* codes frozen) →
   Opus execute (effectSchema + validateEffectParams at elaboration, 3 fix-carrying diagnostics
   CI-fixtured, guide truth-up: the public closed-set claim is now TRUE, prompt names the codes)
   → Fable eval PASS 96/100 round 1 w/ independent transcript re-production. PR #340 → 8faa49de9,
   dev CI green per-workflow. Unlocks effect sprints 2-4. BONUS: dev-health issue #341 filed
   (5 pre-existing example type-check failures; verify-examples not a CI gate))
10. [LANDED 2026-07-11] m-syntax-ai-forgiving (iteration 9 — the first iteration SPLIT ACROSS
    TWO scheduled runs: run A did reality-check 192a79149 + Opus plan a7bd8257c + Opus execute
    (worktree, M1–M4 64ddd6021) then died pre-evaluation; run B resumed at sprint-evaluator.
    Fable eval PASS 96/100 round 1 (FIFTH consecutive) w/ independent fuzz-gate re-run (zero
    AST diffs over 389 currently-valid corpus files), rebuilt-binary transcripts, non-vacuity
    vs v0.29.2 (PAR017/PAR020 fire on exactly the now-accepted fixtures). R1+R2 BOTH landed —
    R2 systemically patched FOUR block loops (plan's D6 knew two; if/then + \-lambda route via
    parseRecordLiteral). PR #342 → merge, dev CI green per-workflow. DEFERRED: ailang fmt →
    m-ailang-fmt.md stub (D1). PARKED for controller/human: the rig A/B compile_error Δ on
    ;-family benchmarks — the REAL success metric, GPU step, rotation held the rig)
11. [LANDED 2026-07-11] m-stdlib-regex (iteration 11: full loop headless, round-1 clean — Opus
    plan (3 de-risking findings: F1 `_str_slice`/`_str_len` are RUNE-indexed but Go `regexp`
    returns BYTE offsets → span conversion is load-bearing; F2 embed is a glob; F4 changelog
    path) → Opus execute (worktree: 6 `_regex_*` builtins in the MODERN `internal/builtins/`
    RegisterEffectBuiltin system — NOT the doc's outdated `internal/eval` path, **D-ARCH**;
    memoized RE2 cache; `std/regex.ail` + 3 examples incl. the log-orchestration clause-4 use
    case) → Opus eval PASS 97/100 round 1 w/ INDEPENDENT reproduction (backref reject, CJK
    `日本語 world` rune span [4,9) not byte [10,15), findAll). PR #343 → squash-merge 0b0ed7ea0,
    all required checks green. `std/regex` = linear-time (RE2): compile/isMatch/findFirst/findAll/
    replaceAll/split; RE2 subset (no backref/lookaround) → `compile` Err, never panics. Docs:
    LIMITATIONS + stability (Experimental) + CHANGELOG. Design → implemented/v0_30_0)
12. [LANDED 2026-07-12] m-stdlib-url-parse (iteration 13: full build loop headless — Opus executor
    (worktree: `_net_url_parse` + `_net_url_parse_query` pure builtins in the modern
    `internal/builtins/net.go`, wrapping Go `net/url`; `Url` record + wrappers in `std/net.ail`;
    26 non-vacuous tests incl. IPv6 `[::1]:80`, error-never-panics, order+dupe preservation,
    round-trip; 2 examples; docs) → independent Opus evaluator round-1 FAIL 80/100 (single BLOCKER:
    stale `builtin_types.golden` not regenerated → repo-wide `make test` red) → round-2 golden
    regen → PASS 100/100. Design → implemented/v0_30_0. PR #347 → squash-merge `a8628a40c`,
    auto-merge on green required checks. `std/net` now parses URLs: `parseUrl(s) -> Result[Url,string]`
    (Err on malformed, never panics/fallbacks — CP2; `port:string` ""=absent) + order-preserving
    percent-decoded `parseQuery` (inverse of `urlEncodeForm`). Pure `! {}`, no Net cap. Closes v1.0
    bar clause 4's URL-parse half (regex half = #11). BONUS finding: `cmd/ailang`
    `TestRunCommand_PipedStdoutFlushesPerLine` is a pre-existing flaky under parallel `make test`
    load — passes 3/3 in isolation, unrelated to this sprint; flagged for dev-health, not a gate)
13. [LANDED 2026-07-12] m-module-less-run-fail-loud (iteration 14: full build loop headless, round-1
    clean — reality-check caught the doc's **MOD011 collision** (already the module-path-collision
    code) → reassigned **MOD014**; Opus plan → Opus execute (worktree: `validateModulePath` early-
    accept replaced with a loud MOD014 error gated on `len(Funcs) > 0`, fires for both `run` AND
    `check`; the doc's 3-way `Funcs||Statements||Decls` guard was code-refuted mid-sprint — a bare-
    expr FILE does reach `validateModulePath`, so the OR would break `ailang run 1+1`; block_demo
    remediated; footgun fixture 17→18) → independent Opus evaluator PASS 100/100 round 1 w/ a
    base-origin/dev binary proving test non-vacuity + pre-existing-failure claims. PR #349 →
    merge `c2ffd1b5c`, post-merge dev CI green per-workflow. Design → implemented/v0_30_0. Module-less
    files now FAIL LOUDLY (CP2). Skill-fix: design-doc-creator error-code + mechanism verification gates)
14. [LANDED 2026-07-12] m-match-xcheck-error-quality (iteration 15: full build loop headless, round-1
    clean — Gate-1 origin-sync caught local dev 4 commits behind origin/dev (iter 14 landed via #350),
    read state from origin; reproduced the empty `Option's constructors are: ` line live at HEAD →
    **Option A** (design doc's own recommendation): a diagnostic-only `Constructor→ADT` registry
    (`moduleImports.AllCtorTypes`) built from ALL transitively-loaded ifaces via
    `modLinker.GetLoadedModules()`, plumbed via new `SetDiagnosticConstructorTypes`, consulted by
    `lookupADTConstructors` ONLY when the primary direct/local scan is empty — never enters scope
    (`types` can't import `link` → passed as a plain `map[string]string`). Opus plan → Opus execute
    (worktree, commits `3ded459cc`/`f5498ca0e`/`ecca08b3b`) → independent Opus evaluator **PASS 96/100
    round 1** w/ base-binary non-vacuity proof + scope-non-leak + format-unchanged checks; 2
    non-blocking deductions folded into the hardening commit
    (`TestSchemeImport_DiagnosticRegistryDoesNotLeakIntoScope` + collision note). PR #352 →
    squash-merge `5aaaff2ed`, required checks green (auto-merge). Design → implemented/v0_30_0.
    Foreign-ctor errors now enumerate transitively-known constructors (`None, Some` + did-you-mean).
    SonarCloud PR gate red = advisory/non-required (merge succeeded) — flagged for sonarcloud-triage)
15. [LANDED 2026-07-13] m-module-let-func-resolution (iteration 23: full build loop headless, round-1
    clean — first CI-red fix-forward (gofmt miss from `366c5bbb2` broke dev fmt-check 2 runs →
    `39171a4f9`, observed green); Opus plan (caught the design doc's WRONG test path: the #327
    40-cell matrix is `internal/pipeline/record_update_positions_test.go`, NOT `internal/types/`;
    proposed MOD007 from the reserved block) → Opus execute (worktree: M0 spike **GO** — evaluator
    binds any `core.Let`, `CheckCoreProgram` threads forward env → unified SCC over lets+funcs,
    `wrapInLets` + BOTH re-elaboration loops DELETED; module `letrec` SUPPORTED via `core.LetRec`;
    dup module-scope name → **MOD007** hard error, zero corpus collisions; hint truth pass — 0
    `known bug #327` hits, residual hint cites #366 + real workaround "declare it as a func") →
    independent **Fable** evaluator (model diversity restored — controller reverted from Opus)
    **PASS 98/100 round 1** w/ own worktrees+binaries, base-binary non-vacuity (v3/v7/v8 fail at
    `116ebcb49` → run 16/0/4 post-fix; v10 silent shadow → MOD007), adversarial probes (func→let→func
    topo chain, let↔func cycle → LetRec no crash, effectful module let rejected identically).
    PR #368 → squash-merge `fd38ec14e`, post-merge dev CI green per-workflow. Design →
    implemented/v0_30_0. Module lets now resolve module funcs uniformly (4th family member CLOSED).
    ⚠ PICK-ORDER MISS recorded: Mark's [NEXT-FIRST] below (added 13:04, pre-session) should have
    outranked this pick; Gate-2 read the queue head + prior log's Next but not the fresh directive.
    Sprint was already through eval when caught → landed; iteration 24 is HARD-PINNED to it)
**[LANDED 2026-07-13, iteration 24 — was Mark's NEXT-FIRST, ⚠ missed by iteration 23, taken
first by iteration 24 as pinned]** m-public-feedback-delivery-audit
([implemented/v0_30_0](implemented/v0_30_0/m-public-feedback-delivery-audit.md), P1): full inner
loop headless, round-1 clean — Opus plan (killed 2 feared ops steps: prod sub exists, ADC owner
on both projects; corrected "structural, not novel" → real multi-project fan-in) → Opus execute
in worktree (Defect A: `isExternalFeedbackInbox` tags `pkg:*` as `public-feedback`, allow-list
untouched; Defect B: `Daemon` N-message-sources refactor + `firestore.NewClientForProject` +
opt-in `extra_message_envs`/`--also-subscribe`, default OFF byte-identical) → Fable evaluator
**PASS 97/100 round 1** (base-binary non-vacuity both defects; 0 test deletions; conflict surface
intact). PR #378 → `4fee247a8`, post-merge dev CI green per-workflow (observed). ⚠ PARKED for
Mark: daemon reload + 2 live prod test-sends (checklist: sprint plan §Parked-for-human +
docs/docs/guides/notify-daemon.md); until reloaded, prod feedback still doesn't ping — the CODE
is landed, the OPS switch is human.

16. [LANDED 2026-07-13] iteration 25 — **R4a+R4b GHOST-CLOSE + m-lambda-open-record-pattern
    EXECUTED**: Gate-2 reality check live-probed the queue's R4 rows (the sourcing strategy
    review admitted they were never individually re-verified) → R4a `m-dx-match-hof` GHOST
    (retired `match … with` syntax was the culprit; design doc archived Not-Applicable
    2026-05-09; `\x ->` already has a teaching diagnostic) + R4b `m-poly-arith-lambda` GHOST
    (fixed v0.7.0) — guards `examples/match_hof_lambda.ail` + `poly_arith_lambda.ail`, PR #379
    → `ea8116f83`, CI green observed. Then the full inner loop on m-lambda-open-record-pattern
    (REAL at HEAD; mislabeled NEW-DOC — full design doc existed at planned/v0_29_0): Opus plan
    (refuted the doc's H3-primary via an IIFE probe) → Opus execute (found the TRUE primary
    site absent from doc+plan: `unifyRecord` rejected on field-count BEFORE consulting row
    variables; `core.RecordPattern.Rest` + `unifyOpenRecords` row-polymorphic subsumption;
    closed-pattern strictness preserved) → independent Fable evaluator **PASS 92/100 round 1**
    (own base+sprint worktrees/binaries, non-vacuity both directions, 8 adversarial probes, 0
    test deletions; found an arm-order-dependent acceptance) → hardening commit `89b75bd3f`
    (order-independence fix proven load-bearing, dead-code removal, cacheKeyVersion v2 for the
    gob-struct change). PR #380 → `47576e25d`, dev CI green per-workflow observed. Design +
    sprint plan → implemented/v0_30_0.

**[LANDED 2026-07-16 (M1a+M1b+M2) / M3 PARKED-protocol]** **m-mission-agentic-provider-routing**
([planned/v0_30_0](planned/v0_30_0/m-mission-agentic-provider-routing.md)) — mission-infra P0.
Fixed the routing-never-enforced bug (memory `project-mission-routing-table-never-enforced`).
**M1a LANDED 2026-07-15** (interactive, 8ee07ef23 + amended d545d4a9e): per-role env pins, opus-first
controller, fable designer/evaluator by inheritance. **M1b+M2 LANDED 2026-07-16 iteration 31**
(direct-on-dev main checkout, zero Go — the planner found registry/DryRun/codex executor all
pre-exist since v0.22.0): Gate-3 `provider:model`→bounded `codex exec` recipe (probe live-verified:
gpt-5.6-sol exit 0; default-env fire = no-op, codex strictly opt-in) `956fda55c` + charter
right-sizing table & provider/agent/cost evidence-row schema `8d12e8e9c`; eval PASS 87/100 round 1;
hardening `1c964aae2` — **F1: the Agent tool pins only sonnet|opus|haiku, `fable` is REJECTED**
(fable roles run by session inheritance only; alias-lane generator≠judge guard added: evaluator
never falls back to bare $MODEL, re-routes to sonnet + FLAG) + F2 `exec` orphan-kill fix.
**Open by design**: first REAL cross-provider fire (opt-in `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol`,
= the doc's M1b acceptance) · **M3** (planner down-tier A/B) PARKED with a concrete protocol in the
sprint plan until 3 quorum-reviewed docs accrue. Doc stays in planned/ until those close.

**[NEXT-FIRST, Mark 2026-07-16 — FLEET ROLLOUT ("should be awesome")]** The ratified starting
fleet is **claude (Anthropic) + codex gpt-5.6-sol (OpenAI) + managed_agents gemini (Google) +
motoko/qwen3-6 (local GPU)**. Sequenced, one per iteration:
- **(a) ~~Iteration 32 — codex LIVE-FIRE~~ DONE 2026-07-16**: FIRST real cross-provider fire landed.
  `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol` (one-shot override consumed) executed `20251013_auto_caps`
  M1 (`--caps auto`) end-to-end: Opus planner → **codex/gpt-5.6-sol executor** (OpenAI, ~4.5-min run,
  metered) → **Sonnet evaluator** (generator≠judge: openai≠anthropic; fable pin unenforceable →
  re-routed to sonnet + FLAGGED per the F1 guard) PASS 98/100 r1. PR #397 → `e542065c0`. **Recipe
  frictions found & fixed (Gate-5 skill edit): the codex real-run recipe had only ever been verified
  against the text probe** — a real coding run needs `--sandbox workspace-write` + `--add-dir` for
  GOCACHE/GOMODCACHE, cannot self-commit (worktree `.git` lives under the main checkout →
  controller finalizes the commit from the uncommitted worktree diff), and must run backgrounded
  (the 30-min cap exceeds the harness's 10-min foreground bash limit).
- **(b) ~~M1c — gemini managed_agents recipe branch~~ DONE 2026-07-16 (iteration 33, PR #398 →
  `bd89418a6`)**: the "no new plumbing" claim was REFUTED — `ailang exec gemini` (agentic) was
  unreachable (`unknown executor: gemini`; managed_agents registers under its own name, no gemini
  alias). Landed a real ~30-LOC `exec.go` fix (`resolveAgenticExecutorName`: gemini→managed_agents,
  `--api-only` untouched) + test + the Gate-3 `PROVIDER=gemini` recipe branch. **CapRemoteSandbox
  scoping**: the lane serves READ-ONLY roles (evaluator/reviewer/quorum-verifier) only — the
  server-side sandbox never writes the local worktree, so the file-editing executor role needs a
  bridge (follow-up). Sonnet eval PASS 96/100 r1. First LIVE gemini fire deferred to (c).
- **(c) [CORE LANDED 2026-07-16 iter 36 (M1-M3) — PR #400 → `0e83a1b12`; M0/M4/M5 now UNBLOCKED — (c0) plumbing landed iter 37, this is the `[← NEXT fleet step]`]** m-mission-quorum-agentic-verify+HONE
  — **M1-M3 shipped**: `agenticCaller` behind the `JSONCaller` seam (frozen verdict JSON via the coordinator
  executor layer), `ShouldEscalate` two-tier trigger + additive-optional `proposed_fix` (option (a), contract
  frozen), Tier-2 codex+claude read-only verify. 43 tests pass, verdict contract independently verified
  unchanged, evaluator PASS 91/100 r1. **M0 (gemini network probe) BLOCKED**: `ailang exec gemini` fails
  `GCP project not set` — `cmd/ailang/exec.go` never plumbs `Task.GCPProject` outside the eval harness
  (fix = item (c0)). Once (c0) lands: M0 (live gemini probe) → M4 (conditional on M0 result) → M5 (live-fire
  + doc → implemented/). Watch items carried: `agentic_caller.go:85` ctx.Background→caller-ctx before a live
  Tier-2 fire; `premiseSignals` breadth; M4 fallback must carry an explicit `VerificationDegraded` marker.
  — iteration 34's Gate-2 quorum-at-pick park is RESOLVED: Mark chose **(a) `proposed_fix` optional,
  not validated, contract frozen** (doc's HONE section stamped; the code-cited Verification-Log rows
  for the refuted sol objection added — provider_executor.go exposes ctx-cancel/Timeout/CostUSD/
  read-only-AllowedTools/WorkingDir, reuse premise HOLDS). Doc is quorum-cleared for routing: **start
  at sprint-planner** (both quorum rounds + revisions already done; do NOT re-quorum — the two rounds
  + resolved authorial decision ARE the quorum outcome). M0 = the managed-sandbox network probe (doc
  §Agentic reviewer backend). Meta-finding stands (text quorum blocked premises TRUE-in-code — the
  motivating case for this doc). Meta-finding (Gate-5): the TEXT quorum-at-pick blocked a doc whose premises are TRUE-in-code
  precisely because text reviewers can't read code — the motivating case for this very doc. Original ask:
  reviewers become tool-using agents (codex/managed_agents/
  claude-CLI, read-only worktrees) that VERIFY premises against the repo AND attach a concrete
  `proposed_fix` per objection; the AUTHOR (designer role, now true-Fable via the Gate-3
  `claude:claude-fable-5` CLI lane — driver default updated) accepts/rejects each by name.
  Single-author + adversarial-proposers, NOT co-authoring. Preconditions all satisfied (doc
  updated). Two-tier stays: text quorum always, agentic escalation when contested/high-stakes.
- **(c0) [LANDED 2026-07-16 iter 37 → implemented/v0_30_0; PR #401 → `60351087b`, eval PASS 96/100 r1]**
  m-gemini-exec-project-plumbing — `resolveGCPProjectEnv()` (`AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT`,
  coordinator precedence) + `GCPProject`/`GCPLocation` now set on the shared `executor.Task` in
  `cmd/ailang/exec.go:executeCLI`; empty location defers to executor `defaultLocation="global"`, unset
  project keeps the loud error (no silent default). **Live-verified**: env-unset → loud error preserved;
  `AILANG_CLOUD_PROJECT` set → error moved past "GCP project not set" to Vertex `HTTP 400: Resource setup
  has just started` (project REACHES the backend — the "resource setup" state is fleet (c) M0/M4 territory).
  Non-vacuous `t.Setenv` regression test. **Fleet (c)'s M0/M4 gemini reviewer lane is now UNBLOCKED** —
  next fleet step is (c) M0 (live gemini network probe) → M4 (conditional on M0) → M5 (bounded live-fire +
  doc → implemented/).
- **(c1) [LANDED 2026-07-17 iter 39 → implemented/v0_30_0; PR #405 → `ae5f0a00f`, eval PASS 96/100 r1]** m-gemini-evaluator-diff-bridge — Mark's #399
  directive ("default evaluator = gemini if able to git clone the codebase, otherwise sonnet-5")
  forced fleet (c)'s M0 live probe early. **Two findings**: (1) **M0 live gemini probe TIMED OUT**
  (`ailang exec gemini` → `http2: timeout awaiting response headers` on the Vertex
  `interactions` POST — the request reaches the backend but no response returns; same class as the
  iter-37 "Resource setup has just started"). Backend reliability is still unproven — M4/M5 stay
  blocked on it. (2) **The evaluator role needs a diff-bridge, not just the executor** (extends
  fleet (b)'s note): the managed_agents request body carries only `Directive`+`SystemPrompt`
  (`managed_agents.go:164`), so even a READ-ONLY evaluator sees NO local repo — it cannot inspect
  the sprint's uncommitted worktree changes nor re-run tests. To make gemini a real evaluator: ship
  the `git diff` + changed files INTO the directive text (mirror `managed_agents_bridge.go`), accept
  it's reasoning-only (no local test re-runs), AND land backend reliability. **BOTH DONE (iter 39)**:
  backend reliability confirmed (4/4 bounded probes SUCCESS); the diff-bridge capability shipped
  (`internal/eval_harness/gemini_evaluator_bridge.go` — `BuildDiffBundle` untracked-inclusive +
  reasoning-only directive + `GeminiVerdict` + `RunGeminiEvaluator` injectable caller seam +
  caller-enforced `VerificationDegraded`; PASS 96/100 r1). Default evaluator STAYS **sonnet** —
  capability only; a gemini-default flip needs a live diff-bridge fire + the ≥3-datapoint evidence rule.
**[GAP CLOSURE PRIORITY — Mark 2026-07-17: "I want the gaps here worked on as priority"]**
Work these BEFORE returning to the clause queue; one per iteration, cheapest-confirmation-first:
- ~~**(G1) gemini FIRST LIVE ROLE FIRE**~~ **CONFIRMED iter 43** — live `ailang design-quorum` →
  `gemini-3-1-pro` **present, verdict=reject, $0.023** (its first clean live reviewer verdict). The
  evaluator arm (`RunGeminiEvaluator`, PR #405) has no CLI seam yet, so the **quorum-reviewer seat**
  (G1's explicit OR) carried it. Reliability blocker found+fixed same iteration: gemini's THINKING
  tokens overran the `reviewMaxTokens=4096` cap → intermittent silent-truncation N-1 quorum (PR #408
  → `885725f06`: cap→16384, fail-loud on `finish_reason=length`, wired gemini `finishReason`). Log 48.
- ~~**(G2) 3-provider quorum CONFIRMATION round**~~ **CONFIRMED iter 43** — same live quorum:
  `gpt5-6-sol` (OpenAI, restored post-#407) + `gemini-3-1-pro` (Google) BOTH present + claude
  controller = 3 providers, both `reject`. The solo-gemini-veto era is over. Log 48.
- ~~**(G3) DESIGNER ROTATION live test**~~ **CONFIRMED iter 44** — `codex:gpt-5.6-sol` (rotation next
  after `claude:claude-fable-5`) authored the G4 design doc via the cross-provider `workspace-write`
  worktree recipe carrying the design-doc-creator directive (**first codex-designer fire**), then ran a
  competent objection-addressing revision. The rotation MECHANISM works end-to-end (design → quorum-gate →
  revise). Evidence row: `(designer=codex:gpt-5.6-sol, quorum=reject→revise→reject over 2 rounds × 3
  providers)` — the content reject is the quorum enforcing data-before-conclusions (unverified external
  contract), NOT a designer failure. Rotation state advanced to `codex:gpt-5.6-sol`; next new-doc iteration
  returns to `claude:claude-fable-5` (gemini joins after G4). Log 49.
- **(G4) gemini REPO-MOUNT upgrade** — **[✅ FULLY LANDED + LIVE-VERIFIED iter 53 — Mark "vertex git clone
  test granted" (#399 2026-07-18T11:59:47Z). The last INCORPORATED premise (provider `git fetch --depth 1
  <sha>` support) is now VERIFIED-LIVE: `TestLiveCloneOverEgressE2E` pinned a real non-HEAD SHA `80cbd9612…`
  through the production `Executor.Execute` path → fetch-by-SHA → exact-SHA echo → `CLONE_OK` → PASS (113.6s,
  $0.865, 527k in/8.2k out). Doc + sprint-plan MOVED to `implemented/v0_30_0/`. **Fleet role (reported to Mark,
  ratification parked):** gemini/managed_agents is now a proven in-sandbox EVALUATOR/reviewer (clone→`ailang
  check`→verdict; Google provider = valid generator≠judge) — but `CapRemoteSandbox` means it canNOT edit a
  worktree, so "gemini joins the DESIGNER rotation" needs the text-bridge and is NOT auto-wired; recommend
  gemini enter as evaluator. **→ RATIFIED by Mark 2026-07-18 (interactive, after cost review):
  gemini is ADMITTED to the fleet as the ESCALATION-TIER in-sandbox evaluator/reviewer — NOT
  every-iteration (sonnet stays the default evaluator) — with THREE mandatory cost guards:
  (1) ENVIRONMENT REUSE ("environment reuse for sure"): clone once per review target, persist the
  `env_<id>`, reuse across rounds — never re-clone per round; (2) tight directives (targeted
  `ailang check`/grep, no repo wandering); (3) two-tier discipline (text quorum first; in-sandbox
  only when a premise is contested/high-stakes). Cost basis VERIFIED against official docs
  2026-07-18: NO managed-agents premium — standard Gemini token rates only ($1.50/M in, $9/M out
  incl. thought tokens at output rate; our client math reconciles $0.865 = 0.79 in + 0.07 out);
  sandbox compute is FREE during preview. ⚠ WATCH ITEM: at GA, Google adds environment-compute
  charges — re-benchmark the escalation-tier economics when GA pricing lands. Designer rotation
  UNCHANGED (claude⇄codex) pending the text-bridge. Next iteration wires the evaluator seat +
  env-reuse.** Prior: LANDED (code) iter 52; both approved fixes (typed `RequiresEgress`/
  `CapNetworkEgress` gate + `ValidateTaskCapabilities`; bounded-execution) + iter-52 shallow-fetch-by-SHA fix;
  opus executor, sonnet evaluator 91/100. Log 57–58.]** iter-45 refuted the `repository`/`inline`
  mount model (only `gcs`+`skill_registry`; egress OFF by default; egress param "undiscovered"). iter-46
  (Mark #399 → philschmid.de/managed-agents-gh) **found the egress param and superseded the mount model**:
  it is a structured list `environment.network.allowlist:[{domain,transform}]` (not iter-45's scalar
  guesses). Re-probing OUR Vertex endpoint (probes O–R, same ADC harness): `network.allowlist:[{domain:"*"}]`
  is **accepted and provisions an egress-enabled sandbox** (Vertex allows wildcard `*` only today;
  per-domain + header-`transform` = "not supported now"). Probe **R**: an egress-only env (NO data source)
  **cloned the public ailang repo end-to-end** (`git clone` OK, `rev-parse HEAD`=`806b3b4a4`=current dev,
  file listing + `go.mod` returned). **⇒ new dominant option (d) CLONE-OVER-EGRESS:** give the executor an
  egress env + have the agent `git clone` the public repo at a SHA itself, then `ailang check`/review
  in-sandbox — no encoder/GCS/inline/mount. Small; directly delivers #399's "gemini can git clone the
  codebase" for the reviewer role. **Recommendation: (d)** (fallbacks: (a) GCS for *private* code, (b)
  shelve, (c) skill_registry). **Decision for Mark on #399:** greenlight the Phase-2 clone-over-egress
  decomposition, or shelve. Reproducible probe: `internal/executor/managed_agents/managed_agents_live_test.go`
  (`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`, CI-inert, probes O–R). Doc RESHAPED with full VERIFIED-LIVE
  contract. Log 51. **Note:** the blog is the Gemini *Developer* API surface (`ai.google.dev`, API-key) —
  a different contract from the Vertex executor; our-project Developer-API confirm is parked (the available
  `GOOGLE_API_KEY` is invalid even for generateContent).
- ~~(G5)~~ **REMOVED from the gap path (Mark 2026-07-17): the qwen3-6 lane is a NICE-TO-HAVE,
  not a gap.** See (d) below — sequenced only after the cloud fleet is fully proven (G1–G4
  done), and NOT at gap priority: after G4, the loop returns to the clause queue; (d) is picked
  on normal cheapest-impact ordering.

- **(d) Phase D — motoko + qwen3-6 local-GPU lane** (fleet doc Phase D, ~2–3d) — **NICE-TO-HAVE,
  post-cloud (Mark 2026-07-17)**: the standing role of this lane is the **local assignee for
  slow-but-free task classes** — long-running, low-urgency work with deterministic verification
  (bulk regens, wide test sweeps, corpus churn) where wall-clock doesn't matter and $0/token does.
  It is NOT a peer of the cloud lanes for interactive-cadence roles. HARD constraints unchanged:
  `rig.lock` two-tier discipline (GPU mutex per-step, never iteration-wide), the port-8080 zombie
  hazard (memory: a hung motoko holding 8080 breaks all later runs), and the same evaluator gate
  as cloud work — no quality discount for free tokens.

**[NEXT-FIRST after the authorized decision set drains — Mark 2026-07-20: FULL BACKLOG RE-TRIAGE]**
("lets do another review of the docs in planned and see which we can put into the cycle.") The
third triage pass, against July reality: fleet live, fmt shipped, v0.30.0 released, arch-boundaries
landed, quorum-at-pick in force. Sweep **ALL planned/ folders** (~114 docs: root 14 · v0_29_0 38 ·
v0_30_0 19 · v0_31_0 3 · v1_0_0 5 · v1_1_0 30 · docparse-billing 5). Rules:
- **Sequencing**: run AFTER the currently-authorized work (raw-handler M1 · reasoning-effort final
  round · fmt polish pair · strict-fallbacks) — those are decided; triage before picking anything
  NEW beyond them. May span 2–3 iterations (folder-group per iteration; oldest first: root +
  v0_29_0, then v0_30/31/v1_0_0, then v1_1_0 + docparse).
- **Per doc**: reality-check the status claim FIRST (the ghost discipline — cheap live probes for
  bug-claims, `git log --grep` for landed-claims; iteration-0/25/48 precedent: statuses LIE), then
  tag exactly one: **[GATING clause-N]** (serves an open bar clause → queue placement) ·
  **[CYCLE]** (non-gating but net-valuable now → normal v0.3x road, loop may pick when gating
  queue is blocked) · **[POST-V1]** · **[GHOST/SUPERSEDED → close with a CI-enforced guard where
  the claim was a bug]** · **[FOLD-INTO <doc>]**.
- **Controller-lane** (read+verify, no generation — iterations 45/48 pattern; no quorum during
  triage, quorum-at-pick fires when a doc is actually PICKED). Deliverable: triage table on the
  bookkeeping issue + charter queue rewrite + archive moves for ghosts/superseded. Docs promoted
  to [CYCLE] get an explicit one-line WHY (what changed since they were shelved).
- **(B) GITHUB-ISSUE TRIAGE (Mark 2026-07-20: "for v1.0.0 we should also triage all the other
  github issues — see if they have design docs, are stale or defunct superseded etc or need a new
  doc")** — the ~19 open non-thread issues (span Dec 2025–Jul 2026: CI flakies #351/#338,
  test-runner litter #328, runtime/test paths #324/#326, effect-row bug #386, the May
  motoko_explore trio #231/#226/#225, CLI asks #223/#157/#155, Z3 #215, docparse #224/#153/#143,
  Sonar #104, stapledons #18, nightly-watch #384). Use the **github-issue-triage skill**. Per
  issue: reality-check at HEAD (cheap live repro where the claim is a bug — issue bodies age like
  doc statuses), then exactly one: **FIXED → close citing the commit** · **STALE/SUPERSEDED →
  close with evidence** · **COVERED-BY an existing/queued doc → link both ways + tag** ·
  **NEEDS-NEW-DOC → note for the designer rotation (doc authored on pick, quorum applies)** ·
  **GENUINE v1-GATING → clause-tag + queue placement** · **POST-V1 → label + say so on the
  issue**. External-author issues ALWAYS get a reply (public repo — same courtesy as #417).
  Runs alongside/after the doc sweep; same 2–3-iteration budget, same evidence discipline.

**[NEXT]** clause-3 accessibility cluster (the bulk of v1.0). Loop ordering within a group:
P0/unblockers first, then cheapest impact-per-day. The DOC-READY/small diagnostics AND the
VERIFY-then-route backlog are now EXHAUSTED (module-less/xcheck/json-bool/split-arg landed iters
14–17; both VERIFY-then-route items closed as ghosts iter 18). **Iteration 25 (2026-07-13)
Gate-2 reality check found the strategy review's R4 rows were never individually re-verified:
R4a `m-dx-match-hof` and R4b `m-poly-arith-lambda` are BOTH GHOSTS** (R4a: original failure used
the retired `match … with` syntax, brace-form works in every probed position, design doc was
already archived Not-Applicable 2026-05-09; R4b: fixed v0.7.0, verified incl. one let-bound
lambda at BOTH int and float) — guard examples `match_hof_lambda.ail` + `poly_arith_lambda.ail`,
PR #379 → `ea8116f83`, dev CI green observed. Same iteration EXECUTED
**m-lambda-open-record-pattern** (REAL at HEAD — doc existed at planned/v0_29_0, so NOT NEW-DOC;
see queue item 16). The parser/type footgun row is now FULLY BURNED DOWN (m-xmod-alias-poly
landed iter 26). **Iteration 27 (2026-07-14) opened the Prelude/discovery group:
m-prelude-option-result LANDED (PASS 98/100 round 1, mission high; PR #382 → `d26215341`) +
m-prompt-option-none-idiom closed SUPERSEDED by it. **Iteration 29 (2026-07-14) EXECUTED
m-dx-examples-coverage → LANDED** (doc re-scoped through the FIRST LIVE 5-round quorum, PR #392
→ `3d451947c`, eval round-2 PASS after a one-line Windows fix `881711325`; 5 red examples
quarantined under #386, verify-examples now a REAL CI gate, docs --examples un-inert —
doc → implemented/v0_30_0). **Iteration 30 (2026-07-14) EXECUTED the last clause-3 starter
m-dx-ai-discovery → LANDED** (RESUMED from a died-mid-execution prior run [transient Anthropic
rc=1 at 16:05, pre-dating the 17:16 driver-retry fix]; doc re-scoped + quorumed by that run at
`39d671a52`, executor completed M1/M3/M4/M5, PR #393 → squash `c07c36b25`, eval round-1 PASS
93/100 + hardening `ea6069815` [arrays→array alias] + Windows guard fix `0ad27444c`. Interleave:
dev went RED mid-iteration from sibling M-STD-YAML/M-SMT merges — fixed forward direct-to-dev
`9a314772d` [yaml builtin golden + Z3-gating verify e2e] + `4caddfd23` [>800-line split of
verify.go/codegen.go]; `ailang docs --all-functions [filter]`, unknown-module did-you-mean +
module list, `ailang docs prelude`, V16 effect-row fix — doc → implemented/v0_30_0).**
**Iteration 22 (2026-07-13) front-ran R4a with a regression-derived NEW-DOC pick** (nightly
`higher_order_functions` triage → real decl-class resolver gap #366); **iteration 23 EXECUTED it
→ LANDED** (PR #368 → `fd38ec14e`, eval PASS 98/100 round 1 — queue item 15). Full inner-loop
sprints, NOT bookkeeping.
*(m-match-xcheck-error-quality LANDED iter 15; m-dx-json-bool-coercion in-repo half LANDED iter 16
[`std/json.asBoolLoose`; Phase-1 firestore fix PARKED out-of-repo]; m-dx-split-argument-warning LANDED
iter 17; m-dx-record-cons-pattern + m-dx-tapp-trecord-unification GHOSTS/verified-closed iter 18;
m-arity-style-diagnostic (R4c) LANDED iter 21 [TC_ARITY_001, PR #363 → `5b54509d1`] —
all → implemented/v0_30_0.)*

*(SCOPE EXPANDED 2026-07-12, Mark — full-v1.0 triage of the 69 non-gating docs. The clause-3
accessibility cluster, BOTH DX tooling investments, and the FULL clause-4 orchestration surface
are all IN. v1.0 = the complete "verified AI-orchestration language, accessible to mid-tier
models" — ~33 open items, ~40–55 sprint-days. Rig/cloud/motoko/post-v1 infra stays OUT. Full
triage evidence = log entry 10.)*

### Clause 3 — fleet-tier accessibility (the footgun burn-down; the thesis's core deficit)
- **Parser/type footgun fixes** (NEW-DOC, Conflict Surface mandatory): ~~m-module-let-func-resolution~~
  **[LANDED iter 23 → implemented/v0_30_0; unified SCC over lets+funcs, wrapInLets deleted, module
  letrec via core.LetRec, MOD007 dup-name, truthful hint; PR #368 → `fd38ec14e`, eval PASS 98/100
  round 1]** ·
  ~~m-dx-match-hof (R4a)~~ **[GHOST iter 25 — retired `match … with` syntax was the real culprit,
  brace-form match works in every probed position (block-body/direct/mid-block/nested-HOF/curried
  foldl); `\x ->` wrong-arrow already has a teaching diagnostic; guard
  `examples/match_hof_lambda.ail`, PR #379 → `ea8116f83`]** ·
  ~~m-poly-arith-lambda (R4b)~~ **[GHOST iter 25 — fixed v0.7.0 (m-poly-arithmetic-fix); verified
  incl. let-bound lambda at BOTH int and float; guard `examples/poly_arith_lambda.ail`, PR #379 →
  `ea8116f83`]** · ~~m-arity-style-diagnostic (R4c, 1–2d)~~ **[LANDED iter 21 →
  implemented/v0_30_0; `TC_ARITY_001` coded/directional/style-aware arity diagnostic at
  `unification_types.go`, 5 golden/regression tests, eval PASS 97/100 round 1, PR #363 →
  `5b54509d1`]** · ~~m-lambda-open-record-pattern (1d)~~ **[LANDED iter 25 → implemented/v0_30_0;
  `{name, ...}` in lambda params now infers OPEN `{name: τ | r}`; PRIMARY root cause was
  `unifyRecord`'s pre-row field-count rejection (deeper than the doc's hypotheses) + Rest erased
  at AST→Core; closed-pattern strictness preserved + arm-order-independence hardened + cacheKey
  v2; eval PASS 92/100 round 1, PR #380 → `47576e25d`, dev CI green per-workflow observed]** ·
  ~~m-xmod-alias-poly (1–2d, VERIFY-FIRST)~~ **[LANDED 2026-07-14, iter 26 →
  implemented/v0_30_0; VERIFY-FIRST probe confirmed REAL at HEAD (NOT a ghost — but the NEW-DOC
  tag was wrong, full doc existed at planned/v0_29_0); parameterized aliases now instantiate
  (`Box[int]` → `{items: [int]}`, single- + cross-module) via `expandAlias` `*TApp` branch keyed
  strictly on alias-env membership (ADTs stay nominal, proven); `TC_ALIAS_ARITY_001`; cacheKey
  v3; eval PASS 93/100 round 1 (first zero-correction pass); PR #381 → `fd1b11a47`, dev CI green
  per-workflow observed]** · **m-parser-block-let-separator** (PARKED, evidence-gated, split out
  of m-dx-expected-fail-fixes iter 40 → planned/v0_30_0): a simple-RHS `let x = e` tolerates
  eliding the statement separator before a trailing expr, but a block-RHS `let x = match{...}`
  does not — a minor parser ASI inconsistency. NOT auto-fixed (default-bias-not-core); route only
  with a measured eval failure-rate + Conflict Surface.
- **VERIFY-then-route** (ran the doc repro FIRST — both were ghosts): ~~m-dx-record-cons-pattern~~
  **[LANDED/GHOST iter 18 → implemented/v0_30_0; `{…} :: rest` type-checks; guard
  `TestListConsPatternWithRecord` + `examples/record_cons_pattern.ail`, PR #358 → `adde9e9d0`]** ·
  ~~m-dx-tapp-trecord-unification~~ **[LANDED/GHOST iter 18 → implemented/v0_30_0; `[[TableCell]]`
  extraction type-checks; guard `examples/record_list_extraction.ail`, PR #358 → `adde9e9d0`]**
- **Diagnostics** (DOC-READY / small): ~~m-module-less-run-fail-loud (MOD014)~~ **[LANDED iter 14 →
  implemented/v0_30_0]** · ~~m-match-xcheck-error-quality~~ **[LANDED iter 15 → implemented/v0_30_0]** ·
  ~~m-dx-json-bool-coercion~~ **[in-repo half LANDED iter 16 → implemented/v0_30_0 (`std/json.asBoolLoose`);
  Phase-1 firestore-package fix PARKED out-of-repo in `ailang-packages`]** ·
  ~~m-dx-split-argument-warning (1d)~~ **[LANDED iter 17 → implemented/v0_30_0; compile-time
  non-blocking reversed-`split` warning, extensible `swapTraps` table, PR #356 → `8339b6421`]**
- **Prelude / discovery**: ~~m-prelude-option-result (Some/None no-import, 1.5d)~~ **[LANDED
  2026-07-14, iter 27 → implemented/v0_30_0; Gate-2 probe confirmed REAL at HEAD (`undefined
  variable: Some`/`Err` without import); planner CORRECTED the doc's mechanism (the proposed
  `InjectPreludeValues` never existed — real fix = implicit lowest-precedence std/option +
  std/result imports at ONE loader call-site consumed by both compile and runtime, entry-modules
  only); explicit imports + local types shadow cleanly, library modules unchanged, no
  cacheKeyVersion bump (verified); 15 new tests, 0 deletions; eval PASS 98/100 round 1 (mission
  high; 20 adversarial probes incl. entry-only through real multi-module runs + PR-#381 alias-env
  non-interaction); PR #382 → `d26215341`, dev CI green per-workflow observed]** ·
  ~~m-dx-ai-discovery (2d)~~ **[LANDED iter 30 → implemented/v0_30_0; re-scoped (one-shot
  discovery: docs --all-functions, unknown-module recovery, docs prelude, V16 fix); PR #393 →
  `c07c36b25`, eval PASS 93/100 round 1]** · ~~m-dx-examples-coverage (1d)~~ **[LANDED iter 29 →
  implemented/v0_30_0; first live 5-round quorum subject; PR #392 → `3d451947c`; 5 red examples
  quarantined under #386; verify-examples now a real gate + validate_manifest --ci wired;
  docs --examples un-inert via manifest `modules` field]** ·
  ~~20251013_auto_caps (infer caps, 2d)~~ **[M1 LANDED iter 32 (kept in planned/v0_29_0 — 1 of 4
  phases): `ailang run --caps auto` infers the entrypoint's effect row + grants exactly those
  (planner refuted the doc's ~200-LOC new-package mechanism → 74-line reuse of the existing
  `iface`/`TFunc2`/`EffectRow` path); FIRST cross-provider codex live-fire (executor = OpenAI
  gpt-5.6-sol, evaluator = Sonnet PASS 98/100 r1), PR #397 → `e542065c0`, all required checks green
  observed. Deferred: `--auto-caps` flag, `AILANG_AUTO_CAPS` env, always-on preflight+exit-2,
  bench-harness integ, cap manifest]** · ~~m-dx-expected-fail-fixes (1–2d)~~ **[GHOST-CLOSED
  2026-07-17 iter 40 → implemented/v0_30_0; Gate-2 live-repro CONFIRMED largely-ghost — 0 of 4
  "bugs" needed a language fix. Bug4 effect_budgets: `@limit` enforcement WORKS at runtime
  ("budget exhausted: semantic limit=3"); the doc's repro put `--caps` AFTER the filename where
  it's ignored (flag must precede the file). Bugs1/2 (arrow-lambda, multi-`requires`) + the 2
  match_foreign files: good teaching diagnostics / intended type-rejections, not bugs. Bug3
  serve_api_webhook: non-canonical example (omitted `;`/`in` after a block-RHS `let`, deprecated
  string `++`). CLOSED with regression guards: the 3 parser-bug examples fixed to canonical syntax
  + promoted to `examples/runnable/` (now CI-gated), effect_budgets README corrected, manifest
  de-drifted (2 mispathed contracts entries repaired). Executor Opus / evaluator Sonnet PASS
  92/100 r1 (generator≠judge). Split-out: the block-RHS-`let` separator ASI inconsistency →
  new backlog `m-parser-block-let-separator` (evidence-gated, default-bias-not-core). PR #406]**
- **Prompt teaching** (batchable, ~0.5d each): ~~m-prompt-option-none-idiom~~ **[SUPERSEDED
  2026-07-14 by m-prelude-option-result's structural fix (its own doc named this band-aid as
  superseded-on-ship); prompt v0.16.2 already teaches the prelude availability; doc → archive/
  with library-module caveat noted]** · ~~m-prompt-single-file-module · m-prompt-split-list-operations ·
  m-prompt-log-file-analyzer-string-ops~~ **[CONSOLIDATED iter 47 into `m-prompt-footguns-to-diagnostics`,
  RATIFIED by Mark 2026-07-18, LANDED iter 54 (2026-07-18) → implemented/v0_30_0.** Part A (PRIMARY):
  wired dormant `MOD002` + new `PAR_MODULE_PLACEMENT` at `parseTopLevelDecl` (mirrors `reportMisplacedImport`)
  + gemini's error-recovery state-isolation fix (two late modules → `PAR_MODULE_PLACEMENT`×1 + `MOD002`×1
  genuine-dup, never a FALSE MOD002) — the ~10% multi-module footgun's opaque `PAR_NO_PREFIX_PARSE` cascade
  is now a coded teaching diagnostic. Part B: split-list-operations GHOST-closed with CI-gated
  `examples/runnable/split_map_join.ail`. Part C (primitive-field, ~2%) SEVERED → extension backlog stub
  `m-diag-primitive-field-suggestions.md`. Planner opus / executor opus / evaluator **sonnet** (generator≠judge)
  **PASS 91/100 round 1**; 3 superseded prompt docs → archive/. No re-quorum (Parts A+B unanimous both rounds).
  Log entry 59.]**
- **DX tooling** (Mark: both in → resolved 2026-07-18: M-TOOLING-DETERMINISTIC **CLOSED-SUPERSEDED**
  by Mark; fmt is the DX item): **m-ailang-fmt [LANDED 2026-07-19 (iter 56)]** — `ailang fmt [--write]
  [--check]` canonical formatter shipped, doc → [implemented/v0_30_0/m-ailang-fmt.md](implemented/v0_30_0/m-ailang-fmt.md).
  New `internal/format` package (exhaustive precedence-aware AST→source printer, no `String()` fallback),
  `cmd/ailang` fmt subcommand (stdout/`--write`/`--check`, atomic same-dir-temp+`os.Rename`, exit 0/1/2),
  opt-in lossless lexer comment scan; `internal/ast/print.go` untouched; newline-per-statement braced
  canonical form, Phase-1 fail-CLOSED on comments (exit 2, byte-identical). Author codex-rotation doc
  (quorum-complete, no re-quorum per Mark ratify). Planner opus / executor opus / evaluator **sonnet**
  (generator≠judge) **PASS 87/100 round 1**. **Controller independent verification caught + fixed a real
  defect the corpus test missed** — the explicitly-pure empty effect row `! {}` was dropped (round-trip
  failed on the doc's own V2–V5 idiom; `ast.FormatEffects` collapses nil vs non-nil-empty; no comment-free
  example uses `! {}`) → `formatEffectRow` helper at all 3 sites + regression fixtures (`0b983a8f8`); 2
  evaluator lint nits cleaned (`305a37dd6`). `metered=$0.00`. Log entry 61. ·
  **m-ailang-fmt-phase2 [LANDED 2026-07-19 (iter 63) — PR #414 squash `3815ba617`; evaluator (sonnet) PASS 78/100 r1; doc → `implemented/v0_30_0/m-ailang-fmt-phase2.md`]**
  — Executor (opus, isolated worktree) shipped M0–M3 (6 commits `83f7ebf23`→`b29e871c4` + lint fix `fe236572c`).
  **Corpus gate V22**: 386 parse-valid → 327 formatted, 0 comment-loss, 0 Phase-2 round-trip regressions, interpolation-refusal 0/386, idempotence 299/299.
  **Calibrated fail-closed boundaries** (both never-lossy, in Future Work): (1) 15.28% (59/386) inline-interior refusal
  (`let … in` chains the parser collapses → no stable idempotent boundary → exit 2, byte-identical) → **follow-up sub-sprint queued below**;
  (2) 28 pre-existing Phase-1 `properties[...]` printer round-trip bugs surfaced (verified not caused; fail comment-free r/t on dev too) → **separate item queued below**.
  Controller fixed 3 sprint-introduced lint issues + moved the doc. Was: DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; UNPARKED by Mark option (b) `c624b456d` (iter 62); planned iter 62.
  — Phase-2 lossless comment preservation, the UNBLOCK for fmt on the 94.7% commented corpus. Doc authored iter-59,
  Rev-3 iter-60 (`design_docs/planned/v0_30_0/m-ailang-fmt-phase2.md`, `d1ed2fe57`); token-anchored envelope
  (AST spans proven unusable at design time). Rev-3 FIXED the 2 R2 defects (V21: **386/393 parse-valid** via
  `ailang check`; hard-left-wall widening clause), but the re-quorum surfaced **2 NEW architecture-level objections**:
  (a) gpt5-6-sol → attacher-totality inventory unproven (no code-audit of all printer child-list boundaries —
  params/type-args/ctor-args/record-fields/annotations); (b) gemini → interpolation clamping structurally fatal
  (collapses inner-AST boundaries; would silently delete comments in `${…}`). **→ RESOLVED by Mark 2026-07-19
  (interactive, option (b) + recommendations): UNPARKED, [NEXT] route to sprint-planner, do NOT re-quorum.**
  (1) M0 of the sprint = the PRINTER CHILD-LIST CODE AUDIT — proven inventory folded into the design before
  attachment code; (2) interpolation = FAIL-CLOSED CARVE-OUT (preflight refuses files with comments inside
  `${…}` holes — silent deletion structurally impossible; full interpolation-aware attachment deferred,
  evidence-gated on measured refusal rate, expected ≈0). Doc Status stamped. ·
  **m-ailang-fmt-adoption [LANDED 2026-07-20 (iter 65) — PR #415 squash `b787bb98f`; executor (opus, worktree) M1–M3, evaluator (sonnet, generator≠judge) PASS 89/100 r1; teaching prompt v0.16.3 (append-only) + `formatter.md` Adoption section + `make fmt-check-ail` (renamed off the Go gofmt gate; `make ci` byte-identical) + opt-in `format_ail.sh` hook w/ Mark-approved SIGTERM→grace→SIGKILL escalation; doc → `implemented/v0_30_0/m-ailang-fmt-adoption.md`. Controller disproved a false "docs build failure" (skipped CI-only sync-registry.sh gen step). Was: IN-SPRINT plan iter 64; UNBLOCKED iter 63; DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; Rev-3 iter 60]**
  — discoverability + opt-in hooks. **NOTE (iter 63): with phase2's 15.28% inline-interior fail-closed refusal, a per-turn
  auto-`fmt --write` hook still no-ops on ~15% of real commented files — adoption scope should account for this (teach
  fmt in the prompt + `--check`/`--write` hooks are now viable for the ~85% lossless majority; the ~15% refuse cleanly, exit 2).**
  Rev-3 iter-60 (`…/m-ailang-fmt-adoption.md`, `d1ed2fe57`). Rev-3 FIXED the jq
  defect (`command -v jq` probe + dropped first-jq `2>/dev/null`); re-quorum accepted it but both reviewers reject
  the timeout fix (SIGTERM-then-unbounded-`wait` wedges on a signal-ignoring proc) — **1 trivial SIGKILL-escalation
  from clean**, but hard-gated behind phase2. **→ Mark 2026-07-19: SIGKILL-escalation correction APPROVED as
  written in the doc; no re-quorum; still rides behind phase2** (which is now unparked). Original scope retained below.
  — Mark #399: "Is ailang fmt discoverable by agents via prompt… run every turn after .ail writes by Motoko or a
  hook in other harnesses?" iter-58 findings: (1) `ailang fmt` is **NOT** in `ailang prompt` (embedded v0.16.2
  teaches `check`/`run`/`test`, not `fmt`) → agents don't know it exists. (2) A per-turn auto-`fmt --write` hook
  (Claude Code PostToolUse on `*.ail`, Motoko per-edit) is **near-useless pre-Phase-2** — it would exit-2/no-op on
  87.5% of real files. Scope once Phase-2 lands: teaching-prompt line + CLI discoverability + opt-in harness hooks
  (`--check` in CI, `--write` post-write). Deliberately NOT teaching fmt in the prompt yet — teaching a tool that
  refuses 87.5% of commented files would frustrate agents (no-premature-adoption). ·
  **m-ailang-fmt-inline-interior** (**[PARKED needs-human-review iter 67 — DESIGN DOC CREATED (codex:gpt-5.6-sol
  rotation designer, `design_docs/planned/v0_30_0/m-ailang-fmt-inline-interior.md`); QUORUM-AT-PICK bounded gate
  CONSUMED: R1 BLOCKED (unverified `*ast.Let` struct) → controller revision V13 → R2 re-quorum BLOCKED on a NEW
  premise (block-form `;`-lets → nested `Let.Body` unproven; "15 of 28 will silently fail"). Controller in-session
  data-check DATA-REFUTES R2 for the 28 targets: all sampled chain via `let … in` → nested `*ast.Let.Body`
  (incl. brace-body outlier `neural_semantic_search.ail` via leading-`in`); the bare-`;` Block.Exprs form does NOT
  occur in the target set. Recommends option (a) printer-local conditional multi-line let-chain; scopes to 28/59
  (15.28%→8.03%), rest explicitly deferred. See its ⛔ Quorum Record. gpt5-6-sol excluded (==designer, generator≠judge);
  reviewer gemini-3-1-pro + controller opus; metered $0.0517.]**) — **Human fork on the bookkeeping issue:**
  (1) [RECOMMENDED] route to sprint-planner (R2 data-refuted for the 28; add M0 mandate to dump+verify the surface
  `*ast.Let.Body` shape of all 28 & correct the Problem-Statement bare-`;` sentence), (2) authorize one more bounded
  quorum round to fold the AST-shape verification in, (3) keep parked. NOT urgent (current behavior safe fail-closed,
  never lossy). Log entry 72. ·
  **m-fmt-properties-printer-roundtrip** (**[LANDED 2026-07-20 (iter 69) — PR #424 squash `942931816`;
  UNPARKED by Mark #422 "Continue the AILANG fmt sprint and go to sprint planner"; planner (opus) →
  executor (opus, worktree) M1–M2 → evaluator (sonnet, generator≠judge) PASS 98/100 r1; contract-clause
  printer round-trip fix + silent-contract-deletion data-loss fix (`parser_func.go` `=`→append) + 2
  adjacent Phase-1 printer bugs the full-corpus sweep exposed (precedence-driven `;`-separation;
  `@verify(depth:)` key re-synthesis) → `preexisting-Phase1-rt-bug` gate 28→0, hardened; 30 contract
  examples reformatted; doc → `implemented/v0_30_0/`. Was: PARKED needs-human-review iter 66 (quorum R2
  data-refuted). `metered=$0.00`]** →
  [implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md](implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md),
  see its ⛔ Quorum Record. **Controller repo-wide re-check DATA-REFUTES the residual objection**: the
  only `ast.FuncDecl.Properties` consumers repo-wide are exactly the V17 sites (`internal/elaborate`
  + `internal/testing`); the `cmd/ailang/test.go` `.Properties` hits are a distinct `[]PropertyResult`
  results field, not the AST field; no accessor/interface/visitor indirection. Scope-corrected doc:
  the real defect is `requires`/`ensures` contract clauses (NOT `properties[...]` blocks) failing the
  Phase-1 printer round-trip (exit 2) on 30 corpus files, PLUS a latent silent-contract-deletion
  data-loss bug (`parser_func.go:169` `=`→append). **Human fork on #399:** (1) authorize routing to
  sprint-planner [RECOMMENDED — sole objection data-refuted, not fmt-phase2's deepening gaps],
  (2) authorize one more bounded round to fold the repo-wide audit into the Verification Log,
  (3) keep parked. ~1d impl, LOW risk/conflict, metered $0.1347 quorum (iter-66). Log entry 71. ·
  ~~M-TOOLING-DETERMINISTIC (normalize/suggest-imports/apply, 3–4d)~~ **[REALITY-CHECKED iter 48
  (2026-07-18) → PREMISE SUPERSEDED; scope-close PARKED for Mark.** The CLI trio doesn't exist, but
  its premise (single-shot fragment + LLM repair) is obsolete — `prompts/repair_prompts/` deleted,
  eval flow is agentic w/ per-edit `ailang check` feedback — and its core capability already ships as
  `normalizeProgram` (`internal/eval_harness/normalize.go`, deterministic module-wrap + std/io inject).
  Per-goal: G1 normalize=SHIPPED; G2 suggest-imports=PARTIAL/ABSORBED (std/io only; general
  symbol→import now met by implicit prelude + agentic feedback + `ailang docs`); G3 apply=obsolete
  (agents edit files directly). Regression guard `TestNormalizeProgram_MToolingMotivatingFragment`
  landed. Doc header → REALITY-CHECKED w/ per-goal table. **Mark scoped DX tooling "both in" → not
  ruled out unilaterally; awaiting his SUPERSEDED-close vs. much-smaller "expose `ailang normalize`"
  decision on #399.** Recommend prefer m-ailang-fmt for any DX budget. Log entry 53.]**
- **Prompt-diet** (GATED — unblocks once the diagnostics above land + the curve authorizes):
  m-eval-slim-prompt-self-discovery (R3.1 pass-rate-per-token curve, 2d) → prompt-deletion pass R1.2

### Clause 4 — orchestration flagship (Mark: full surface in)
- **Effect sprints** (decomposed): m-effect-replay-contracts (2/4, 3d) · m-effect-clock-net-fs-modes
  (3/4, 3d) · m-effect-scope-params (4/4, 2.5d — release-gate re-score candidate)
- **Flagship + surface**: m-v1-orchestration-flagship (verified AI-pipeline example + orchestration
  benchmarks into rotation + README/site lead, 2–3d; m-contracts-as-code-vertical folds in as the
  worked example) · m-serve-api-live-tool-registry (hot MCP tool registry, 3–4d) ·
  **m-serveapi-raw-handler-mcp** (**[DECIDED by Mark 2026-07-20 → ROUTABLE: ship M1 `@nomcp`
  standalone NOW (closes the live docparse MCP leak); M2 fake-envelope DROPPED — doc Status
  stamped, no re-quorum. Historical park:]** **[was: PARKED iter 57 — QUORUM-AT-PICK BLOCKED ×2]** →
  [planned/v0_30_0/m-serveapi-raw-handler-mcp.md](planned/v0_30_0/m-serveapi-raw-handler-mcp.md), see its ⛔ Quorum Reblock section;
  M1 `@nomcp` MCP-exclusion annotation — keep a `@route` on HTTP but off the `--mcp-http` tool surface
  (`@noexpose` can't: it also kills HTTP + is overridden by `@route`) — closes the live docparse
  `getKeyUsage`/`requestHistory` MCP leak; **M1 is CLEAN + unobjected in both rounds → independently shippable**.
  M2 `@raw`-over-MCP twice-rejected: R1 default-on = authority-widening + silent header-fabrication; R2 the
  `@mcp`-opt-in + typed-sentinel fix itself violates the frozen core (`headers`/`query` are `Json` → a non-`Json`
  sentinel type-panics at binding; a `Json` sentinel needs core `std/json` changes). **Human fork on #399:**
  (1) split+ship M1 now [RECOMMENDED], (2) pick an M2 arch — valid-`Json` provenance marker
  `{"_transport":"MCP_UNAVAILABLE"}` + require `req.method=="MCP"` branch, OR drop the fake-envelope entirely,
  (3) keep parked. Unblocks docparse quota-hardening item 5. Log entry 62. ~0.5d for M1 alone) ·
  m-agent-step-cancellation (1.5d) ·
  **m-ai-reasoning-effort** (**[AUTHORIZED by Mark 2026-07-20 → ROUTABLE: ONE more bounded
  revision+re-quorum round, scoped to the 2 named R2 objections only — doc front-matter stamped.
  Historical park:]** **[was: PARKED iter 61 — QUORUM-AT-PICK: R1 BLOCKED
  (no-silent-fallback + missing MaxTokens conflict surface) → codex-designer Rev-1 resolved both
  (fail-loud contract w/ 5 typed errors + capability gating + full Conflict Surface) → R2
  re-quorum BLOCKED on 2 NEW *narrower converging* fixes]** →
  [planned/v0_29_0/m-ai-reasoning-effort.md](planned/v0_29_0/m-ai-reasoning-effort.md), see its
  ⛔ Quorum Record. R2 objections: (1) resolver omits OpenRouter's `reasoning_max_tokens` 4th input;
  (2) Gemini rule over-reaches forcing `MaxTokens` for `B=0` "off" (breaks docparse consumer).
  Both small/concrete — NOT fmt-phase2's deepening gaps. **Human fork on #399:** (1) authorize
  ONE more bounded round [RECOMMENDED — close to green], (2) amend scope (drop `reasoning_max_tokens`
  from the typed resolver), (3) keep parked. ~14h impl (doc est), metered $0.23 (iter 61). Log entry 66.
  **REALITY-CHECK (iter 66, log entry 71): STILL PARKED — the feature did NOT land out-of-loop.** The
  iter-65 "Next" flagged commit `5afa9a1e1` ("feat(eval): reasoning_effort knob") as a possible
  out-of-loop landing. REFUTED: that commit is an EVAL-HARNESS-only OpenRouter `reasoning.effort` knob
  (`models.yml` + `internal/eval_harness/` + `openrouter/chat.go`), NOT this doc's typed
  `ai.Request.ReasoningEffort` field. Verified absent on origin/dev: `git show origin/dev:internal/ai/provider.go`
  has no `ReasoningEffort`, and the 5 sentinel errors (`ErrUnsupportedReasoningEffort`, …) do not exist.
  The v0.31.0 cross-provider feature is unbuilt; the R2 fork above still awaits Mark.)

### Clause 5 — cost credibility
- m-cost-per-success-kpi (dashboard KPI flip to cost-per-verified-success + v1.0 measured baseline, 1–2d)

### Clause 2 — soundness (near-done; no new holes found in triage)
- **m-check-strict-fallbacks** (now ~2d) **[DECIDED by Mark 2026-07-18 — option 2: post-name-resolution
  pass + curated known-empty-builder registry (catches `Ok(jo([]))`), warning-in-dev / hard-error at
  `check --package`; doc Status stamped UNPARKED → route to sprint-planner, no re-quorum. The historical
  park record follows:]** — iter-42 re-attempt (both iter-41 blockers cleared: Fable designer back; #407 quorum
  fix). Resolved the iter-41 "OPEN decision" to option (a) (syntactic surface-AST pass, hooks
  live-verified) + grounded Pattern C in the language-enforced uppercase-constructor rule — BUT a
  clean re-quorum (on a REBUILT binary; the #407 fix was NOT in the stale installed binary, so
  gpt5-6-sol had been silently unreachable) **BLOCKED** on a goal-contradicting objection:
  **the purely-syntactic pass cannot catch its own motivating incident** `None => Ok(jo([]))` —
  `jo` is a LOWERCASE function call, which the doc (and Pattern C) never flags; catching it needs
  resolved callee identity (name resolution), refuting option (a). Human decision = the architecture
  fork: (1) run after name resolution, (2) narrow the goal (literal-empties only), or (3) curated
  known-empty-builder list; + resolve the warning-vs-error/exit-1 `--package` channel. Doc has the
  full REBLOCK write-up. See log entry 47.
- **m-bytecode-vm-parity-bugs** (bytecode-VM vs eval output divergences, 1–2d) **[REAL but doc-drifted,
  iter-41 Gate-2 data]** — live `verify_bytecode_parity.go` at HEAD shows **6 DIVERGE**, not the doc's 3
  (2026-04-08 baseline): real show-dispatch (`<Closure>`/`<List>` in array_basic, pattern_sugar,
  recursion_quicksort) + a loop-dup (tar_gzip_reader) + a timing false-positive (xml_walk_perf) + an
  API case (claude_haiku_call). Needs a designer/planner re-scope against fresh data before routing.

**HUMAN-LED lanes (Mark, 2026-07-14 — the loop keeps HANDS OFF unless a sub-item is
explicitly delegated in this queue):**
- **The coordination dashboard** (`ui/` Collaboration Hub + internal/server): six build passes
  since v0.4.4, feature-complete but architecturally unfinished (simplification 1-of-6 PRs done;
  EvolutionTree 2,061 lines), unmaintained since Feb. Day-to-day coordination-watching has moved
  to `ailang chains` CLI + issue #329. Mark hand-holds; in/out decided at the release gate.
- **The Go/bytecode compilation story** (`internal/gen/*`, `internal/vm`, `internal/bytecode`):
  strategy of record = evaluator-first + Statement IR + bytecode (Tier B perf path, ~95% parity);
  Go source emission DEMOTED to diagnostic projection; emit-go-v2 PAUSED (415 symbols short, open
  design-committee question). Mark hand-holds; posture ratified at the release gate.
  (Exception already delegated: m-bytecode-vm-parity-bugs — 3 small output divergences — stays
  in the clause-2 queue.)

**v1.0 RELEASE-GATE AUDIT (one human session, Mark + controller, when the gating queue is
empty — the bundled in/out + ratification calls):**
1. Stability tier assignments (parked since iteration 5: std/net, crypto, jwt, xml, zip,
   process, CLI watch/serve-api).
2. Dashboard: in/out of 1.0 (evidence: dashboard-lineage review 2026-07-14; keeping it OUT costs
   nothing user-facing — chains CLI covers the live path; IN = commit 4–7d to finish the
   abandoned simplification).
3. Compilation posture: ratify evaluator-first for 1.0 (`--bytecode` labeled experimental).
   **PRE-DECIDED (Mark 2026-07-14): emit-go-v2 FROZEN** (contracts projection stays live) —
   formal ratification here; VERIFY the contracts codegen caveat (gen/golang/contracts is live
   via --verify-contracts — if 1.0 materials mention contract compilation, that ships).
4. Boundary split: **PRE-DECIDED (Mark 2026-07-14): m-arch-boundaries ADOPTED** — Phases 1–3
   pre-1.0 (queued, loop-executable), Phase 4 physical restructure AT this boundary (schedule it
   here), separate repos rejected. Audit confirms Phase 1–3 landed + green, then greenlights
   Phase 4 as the first v1.1 act.
5. Effect-scope-params re-score (standing flag from iteration 6).

**The v1.1 arc (spine, Mark 2026-07-14):** *"the bytecode VM grows up, proven by a game"* —
m-arch-boundaries Phase 4 (physical core/apps/tools split) → the game engine as typed effects
(`m-game-engine-effects`, [planned/v1_1_0](planned/v1_1_0/m-game-engine-effects.md)): Stapledon's
Voyage revived on `!{Render, Input, Clock}` host effects, evaluator-first, with the game's
frame-budget as the VM's standing flagship KPI. Go source codegen stays demoted (emit-go-v2
frozen; contracts projection live).

**Mission-infrastructure backlog** (improves HOW the loop runs; not a v1.0 gate):
- **m-mission-adaptive-multiprovider-routing** ([planned/v0_30_0](planned/v0_30_0/m-mission-adaptive-multiprovider-routing.md); EXPANDED 2026-07-14 per Mark — quota now the binding constraint) — the heterogeneous model FLEET. **[Phases A+B LANDED 2026-07-14, iteration 28]**: Phase A (quota-aware multi-candidate probing in the driver) landed `3bee6b6df` direct-to-dev by the interactive session + verified/hardened by the sprint; Phase B (design-doc QUORUM: gpt-5.6-sol + gemini-3-1-pro-via-Vertex-ADC + Claude controller in-session, reject-by-default, N−1 named-absence degrade, budget-capped) landed PR #383 → `1186a48e6`, eval PASS 94/100 round 1 — `ailang design-review`/`design-quorum` live, artifacts under `.ailang/state/mission-quorum/`. REMAINING (opt-in as evidence accrues): Phase C cross-provider executors (re-scoped ~1d, audit binding); Phase D local-GPU lane (~2–3d); Phase E full (provider, model)×task-class assignment (~3–4d). Quorum-on-sprint-plans deferred (hook scoped to design docs). Requested + prioritized by Mark.
- **m-arch-boundaries Phases 1–3** **[LANDED 2026-07-20 (iter 68) — PR #420 squash `ee97fada6`; evaluator (sonnet, generator≠judge vs opus executor) PASS 88/100 r1; doc → [implemented/v0_30_0/m-arch-boundaries.md](implemented/v0_30_0/m-arch-boundaries.md)]** — `scripts/check_boundaries.sh` self-testing import-boundary CI gate (Rule 1: no core→dashboard; Rule 2: no dashboard→compiler-surface except via `internal/embed`; MODULE-vs-`go.mod` drift guard; `eval` excluded from Rule 2 for the sanctioned `eval.Value` bridge type, documented) + `make check-boundaries` + CI step + `ARCHITECTURE.md`/`CLAUDE.md` boundary docs + `.github/CODEOWNERS`. **No physical restructure** — Phase 4 (`git mv` core/apps split) reserved for the v1.0→v1.1 boundary; dual-release-tracks out of scope. Planner (opus) corrected 5 stale doc premises; executor (opus) caught 2 real defects (wrong module anchor → false-pass; `server→eval` bridge import); `metered=$0.00`. **Follow-on queued: m-arch-boundaries-eval-exclusion-tighten** (evidence-gated — tighten the `eval` exclusion package→file level; only 1 file uses it today).
- **m-mission-quorum-agentic-verify** ([planned/v0_30_0](planned/v0_30_0/m-mission-quorum-agentic-verify.md), 2026-07-14; P1) — the shipped text quorum REASONS but cannot VERIFY (no repo access); this makes reviewers tool-using agents (codex/managed_agents, read-only worktree) that actually run `ailang check`/grep to confirm premises, two-tier (cheap text first → agentic escalation only when a premise is contested). Reuses the quorum contract + executor registry. Sequenced after fleet Phase C. Precondition: confirm Tier-1 has fired LIVE (no artifacts found yet). Requested by Mark.

- **m-mission-portability** — **M1 DONE (attended with Mark, 2026-07-21: driver parameterized —
  MISSION_PROFILE/NAME/REPO/DOC, v1-legacy-exact vs namespaced state, template plist; 3 dry-run
  acceptance tests incl. live-isolation proof). M2+M3 HEADLESS-GREENLIT (Mark option a) → loop-routable:**
  ([planned/v0_30_0](planned/v0_30_0/m-mission-portability.md), 2026-07-18;
  **P1 mission-infra — GATES THE AILANG WORLD MISSION LAUNCH**, Mark: "design doc this up and plan
  it in") — extract the loop into a portable template: M1 driver parameterization + per-mission
  state namespace (`MISSION_NAME/REPO/DOC` profile env; backward-compatible defaults — this
  mission's behavior unchanged), M2 skill repo/verify profiles (go-compiler vs ailang-code —
  World verifies via `ailang check/test/ai-check`, which the binary ships), M3 bootstrap kit +
  charter template + scratch-repo dry-run (no state collision with the live loop). ~1–1.5d, zero
  language surface. **Pick order: after the greenlit clause-3 trio (fmt → footguns → strict-
  fallbacks) — OR earlier if the clause queue blocks on anything.** ONE skill parameterized, never
  forked (Gate-5 retro fixes must keep benefiting all missions). Expect quorum-at-pick (doc
  authored interactively, no creation-time quorum).
- **m-eval-reasoning-model-fairness** ([planned/](planned/m-eval-reasoning-model-fairness.md);
  authored 2026-07-11, **QUEUED by Mark 2026-07-19, P1**: "why does GLM 5.2 perform worse than
  5.1? We think it may be our eval harness's fault — thinking tokens/limits with OpenRouter") —
  the doc already carries the evidence: GLM-5.2 40/56 vs 5.1 48/56 with negative token counts,
  empty `code` fields despite compile_ok, and NO reasoning request/budget (MaxTokens bounds total
  output → thinking crowds out the answer). Iteration 43 proved the same mechanism live in our
  quorum (PR #408) — apply the same remedies: reasoning-aware budgets, fail-loud on
  `finish_reason=length`, per-turn finish_reason capture, then RE-RUN the GLM pair to split
  harness-artifact from genuine regression. ~1–2d, metered-cheap (OpenRouter), no GPU. Expect
  quorum-at-pick. Eval-infra (non-gating for v1.0) but Mark-prioritized — pick after the
  greenlit clause-3 trio unless the queue blocks. **RE-RUN VERIFICATION MODELS: the GLM 5.1/5.2
  pair + Kimi K3 (top OpenRouter model, 97/109 standard — also reasoning-class).**
- **m-comments-for-ai-authors** ([planned/v0_31_0](planned/v0_31_0/m-comments-for-ai-authors.md),
  **direction RATIFIED by Mark 2026-07-20**: prompt style guidance + first-class `---`
  doc-comments + contracts/tests-as-self-documentation "as much as is reasonable" + the eval) —
  M1 prompt comment-style section (≤15 net lines, prompt-manager lane, ~0.5d) · M2 the
  comment-variant A/B (V-strip / V-keep / V-migrate on MODIFICATION tasks, haiku, N-run
  aggregates; registered hypotheses; SHARES m-eval-fmt-weakmodel-ab's variant machinery — build
  once, run both) · M3 first-class `---` doc-comments as AST nodes (v0.31; dissolves fmt
  attachment at the root for the doc position; sequence AFTER the fmt polish pair) · M4
  contracts-as-docs exemplars (rolling). First measured comment semantics for AI authors.
- **m-eval-fmt-weakmodel-ab [NEW-DOC, Mark 2026-07-20** — "fmt should be a real help for weaker
  models creating AILANG… can we do a test with a weak model to see if its making a difference?"
  + his #422 directive "test it's used by small model such as haiku"**]**: A/B agent-mode evals,
  ONE weak model (haiku first; optionally a local small model as replication), fmt PostToolUse
  hook ON vs OFF, same benchmarks/N-runs. Metrics: pass rate + compile-stuck/green-stability
  convergence (the noisy-agentic-metrics rule: N-run aggregates, never single runs) + per-turn
  fmt exit codes (was fmt actually invoked/useful). Hypothesis: canonical formatting reduces
  weak-model syntax drift. Depends: fmt+adoption (LANDED); sequence AFTER the fmt polish pair
  below lands (test the finished tool, not the interim). ~0.5d + eval time, subscription/cheap.
- **m-eval-kimi-k3-agentic** ([planned/v0_31_0](planned/v0_31_0/m-eval-kimi-k3-agentic.md),
  Mark 2026-07-19: "Kimi K3 did very well — look into using it within the suite via OpenRouter
  and Pi or motoko harness") — K3 = **97/109 (89%) standard, the strongest OpenRouter model on
  the v0.30.0 board** (beats GLM-5.2 88, K2.7-code 88, GLM-5.1 85). Onboard it AGENTICALLY:
  `motoko-or-kimi-k3` + `pi-or-kimi-k3` roster entries (K2.6 precedent, mechanical), smoke→core
  tiered runs, 4-way comparison (vs its own standard score, vs K2.6, vs GLM-5.x, motoko-vs-pi
  harness effect), routing-evidence rows; if it clears the sweet-spot bar → PROPOSE for the
  fleet's Phase-E table (admission stays a routing-policy decision). ~0.5–1d, metered-cheap,
  no GPU. **HARD-SEQUENCED AFTER m-eval-reasoning-model-fairness** — K3 is always-reasoning;
  measuring it agentically on the pre-fix harness = the broken ruler. Expect quorum-at-pick.
- **m-mission-loop-heartbeat [NEW, 2026-07-21 — born from the 18h reboot outage]**: a tiny
  SECOND launchd agent (independent of the loop it watches) that every ~2h checks: newest driver-log
  line older than ~4h AND no kill switch AND no live pidfile → send a controlplane alert + ⚠ comment
  on the bookkeeping issue + `launchctl kickstart` the mission job (recovery, not just alarm). The
  2026-07-20 reboot silenced the loop for 18h and only a human ping caught it — the loop needs a
  pulse that does not share its failure domain. ~0.5d; pairs with RunAtLoad=true (b5b9899a0: repair)
  as detect+repair. Also: the driver should DELETE a stale pidfile whose boot-time predates uptime
  (reboot invalidates PIDs — a reused PID would false-yield every fire; cleared by hand this time).
- **m-mission-cost-chains** ([planned/v0_30_0](planned/v0_30_0/m-mission-cost-chains.md), 2026-07-18;
  **P1½ — the clause-5 KPI's data substrate**, Mark: "keep an eye on these budgets… that should
  all appear in ailang chains CLI") — VERIFIED live: chains flow for eval (45 chains/50.3M tokens
  in 48h) but **cost=$0.0000 everywhere** (attribution unwired, Defect A) and the MISSION's
  activity — iterations, codex $, gemini $ (the $0.865 E2E appears nowhere), quorum cents — is
  **entirely absent** (loop never posts to `handleCreateChain`, Gap B). M1 fix cost attribution
  at source + rate-fallback (flagged estimates) · M2 Gate-4 posts one chain per iteration
  (`mission:<name>/iter-N` — portability-ready for World; fail-soft spool, never blocks the loop;
  quota-lane stages carry bucket attribution at $0) · M3 `chains stats --by-mission` budget
  rollup vs `MISSION_METERED_BUDGET_USD`. ~1.5–2d. **Sequence BEFORE m-cost-per-success-kpi**
  (a headline cost KPI over a $0.0000 tracker is not credible). Expect quorum-at-pick.
- **m-public-feedback-delivery-audit** ([planned/v0_30_0](planned/v0_30_0/m-public-feedback-delivery-audit.md), 2026-07-12; **P1**) — external user feedback (Kevin's) silently lost: ROOT-CAUSED: dev/prod env split (Mark). Public MCP writes feedback to PROD (`ailang-multivac`) — Kevin's June-30 messages are there, triaged; the rig daemon subscribes to DEV only, so external feedback never pings Discord. Fix = daemon dual-subscribes dev+prod; plus the latent pkg:*-inbox Discord-filter bug. The human-input channel that feeds the data-led loop — prioritize. Requested by Mark.

- **m-mem-budget-runtime** ([planned/v0_31_0](planned/v0_31_0/m-mem-budget-runtime.md), 2026-07-21;
  **P1 — host-safety, DOC-READY**, Mark: "make a design doc for this to insert into our mission
  loop sequence") — the 2026-07-20 rig kernel panic (watchdogd starved under swap-thrash; Jetsam:
  3 model-generated Python procs at ~80-120GB, ailang at 7.7GB) proved generated code WILL
  occasionally be a memory bomb. AILANG's protection today is incidental (no while/mutation +
  interpreter speed) — this makes it guaranteed: `--max-mem`/`AILANG_MAX_MEM` → Go soft limit +
  memguard monitor + cooperative unwind → typed `MEM001` (verified unallocated) instead of host
  death; harness banks it as a distinct error category (model signal, not rig outage). Extension
  lane, zero syntax change, `Mem`-as-effect explicitly rejected (A3/A8). Complements (does not
  replace) the harness-side RSS watchdog task covering the Python/JS/Go lanes. Verification Log
  complete incl. negative-existence rows; Design Freeze needs quorum ratify of the two frozen
  decisions (runtime-control-not-effect; default-off CLI / explicit-on harness). ~2-3d. Phase 2
  (deterministic logical meter, replayable exhaustion) split to a future `m-mem-meter-logical`.

**Not gating** (the ~30 non-gating docs (eval-infra rig/harness, cloud-infra, motoko-fork, post-v1)): ship on the normal v0.2x road or post-v1 per the
clause rule. `planned/v1_0_0/` now contains ONLY gating docs (17 non-gating docs re-bucketed to
v1_1_0 on 2026-07-11); v0_29_0 docs that appear above gate v1 via the queue, not the folder.

**Post-v1**: everything in `planned/v1_1_0/`.

## Ruled out / resolved

- **Sonnet as default executor** — ruled out 2026-07-10 (Mark: corrections needed; false economy).
  Re-entry only via the evidence rule.
- **Scheduling via cron / scheduled-tasks MCP** — ruled out; this rig's substrate is launchd
  (nightly-eval + os-rotation-filler precedents), and the coordinator has no internal timer.

## Done / superseded

*(nothing yet — mission initialized 2026-07-10)*
