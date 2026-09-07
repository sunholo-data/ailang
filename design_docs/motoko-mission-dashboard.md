# Mission Dashboard — Motoko

*Snapshot, overwritten every iteration. History lives in the charter and the log.*

**Last iteration**: 38 · 2026-09-07 · `[HARNESS]` · row 16 **LANDED** (`a329fdb4f`, Gate 3b required 4/4), row 6s **PARKED**
**Latest release**: v0.35.1 · repo `sunholo-data/ailang` (shared with V1, which OWNS dev CI red)

## In flight / next
- **`D-MOTOKO-CARVEOUT-1` is the gate on the next pick** — was Gate 2's narrow-refinement carve-out
  available at round 2, or should the doc have parked? Loop recommends **(B) OVERRULE** against its
  own decision. Default if unanswered by **2026-09-14**: (B).
- **row 6s** — `expected_arms` floor. Design, plan, implementation and a 4-mutant matrix are DONE and
  UNMERGED on `sprint/motoko-iter38-changelog-arms`; only the approval question is open.
- **row 7** — profile restoration design. BLOCKED on itself: the charter row is one line and its
  premise is not reconstructible. Needs a restated premise before it can be picked.
- **rows 10/11/12** — Phase-0 gated on Arni's ABI declaration. Re-measured 2026-09-07: **0**
  `arniwesth` comments on upstream `#165` (control: 35 elsewhere), `#154` still OPEN.

## Loop cadence + routing
- launchd `dev.ailang.mission-motoko`, `StartInterval=43200` (12h), staggered against V1 and World.
- controller `claude-opus-5` · designer **ROTATION**, last used `pi:ollama/deepseek-v4-flash:0731-cloud`
  · planner + executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
- **Only the evaluator can use the Agent tool.** The spawn-pin hook denies that path for any role
  pinned to a `provider:model` value; the other three go through their own lane recipes.

## Parked on Mark
- **D-MOTOKO-CARVEOUT-1** (OPEN, filed 2026-09-07) — the only open decision. One word: A or B.

## Quota posture
- Metered **$0.1546** last iteration of a $5/iteration ceiling (two design-quorum rounds).
- Subscription/flat-rate: ChatGPT bucket (codex), Ollama Cloud (pi deepseek), Anthropic (sonnet judge).
- Fable **unspent**; the designer rotation is on its non-Anthropic lane.

## Known drift
- Source clone `~/dev/sunholo-data/ailang-motoko` is **345 commits behind** `origin/dev` (237 at
  iteration 35). `D-MOTOKO-WORKDIR-2`'s standing authorisation covers the reconcile once its three
  predicates are re-measured. The pin worktree is at `origin/dev` by construction and is what runs.
