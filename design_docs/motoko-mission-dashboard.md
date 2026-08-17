# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 each iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md. Humans
> steer via the bookkeeping issue. **Namespaced** — `mission-dashboard.md` is V1's.

**Updated**: 2026-08-17 ~08:00 local (iteration 8)

## Where the mission is
- **Charter RATIFIED** (iter 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–4**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md),
  now 14 SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED); Phase 0 became a bounded fail-closed gate
  with **G5** (Arni's word) a permanent human predicate; **R8 SETTLED → PORT** — the ladder has *no
  lever*, so the upstream ask is one argument at `session.ail:2561`.
- **Iter 5**: the loop's own 25%-of-fires refusal was **ours** — `session_start.sh` handed its
  stdout to a background child. PR **#721**. Iter 6/7: the rig-blind driver gate fixed (PR **#728**)
  and iteration 6's lost record recovered.
- **Iter 8**: **the −74% we were about to re-prove is one benchmark and one void pair.** 74.7% of
  the saving is `log_file_analyzer` (now 0/10 over five nights, open **#649**); one of the six pairs
  was quarantined by the harness (`treatment_unproven`) and summed in anyway; and every ON row ran
  before the first OFF row, so time is aliased with treatment. **On the four honest pairs: −5.7%.**

## Next
- **D-MOTOKO-FMT-1 is the only thing gating item 6** — one word from Mark (see below).
- Otherwise **item 6b** (triage-lite the 15 charter-orphaned open issues from this week's sweep;
  most are AILANG-lane and belong to V1) is the head. Item 5 is bounded until **2026-08-27**.

## Blocked / parked
- **Item 6 PARKED** `needs-human-review` — [the instrument](planned/m-motoko-fmt-remeasurement-instrument.md)
  is written and twice-reviewed; **no reviewer disputed its direction in either round**.
- **Blocking any fmt run regardless**: the Wednesday A/B lane has banked nothing since AC5 — both
  fires died at `internal/executor/motoko/healthcheck.go:64`, an unconditional `OPENROUTER_API_KEY`
  refusal, though both arms are local ollama (`provider: "ollama"`, `env_var: ""`).
- **Phase-0 gate REAL and unmet**: G1 `#154` OPEN, G2 `packages/` absent upstream, G3 registry
  1.0.0–2.2.0 only. Items 10/11/12 parked; items 9, 13, 14 need a green tree.

## Open with Mark (see bookkeeping issue) — one decision, one word
1. **D-MOTOKO-FMT-1**: is tracing motoko's *resolved runtime provider* a **precondition** of D1, or
   does D1 need a **redesign** that leaves the preflight alone? *(precondition / redesign)*
2. Carried, unanswered since iter 1: (a) was routing item 3's *analysis* while its *design* was
   quorum-blocked too loose? (b) keep the namespaced `motoko-mission-dashboard.md`?

## Loop posture
- Cadence **12h** (`43200s`). Bookkeeping issue **rotated this iteration** (Monday boundary).
- Routing: controller opus · designer `claude:claude-fable-5` **(rotation fallback — codex is
  quota-exhausted until Aug 20; gemini unwired pending G4)** · planner/executor never spawned ·
  evaluator sonnet. **Codex lane DOWN, second consecutive fire.**
- **Metered iter 8: $0.1424** of $5 (two quorum rounds, both reviewers present both times).
- `make quick-install` **skipped deliberately** (shared-write guardrail, V20).
