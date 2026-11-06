---
sidebar_position: 4
title: Training data
description: Where AILANG fine-tuning data comes from, the rules it must pass, the habits it should teach, and the coverage gaps it exposed.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Training data

The binding constraint is **task diversity, not volume**.

## Sources

| Source | Size (2026-11-05) | Teaches |
|---|---|---|
| AILANG code in our repositories (de-duplicated by content hash) | 80,415 `.ail` files scanned; 3,684 unique; ~843k lines | the language |
| Past eval results | 14,373 distinct passing programs, but only 114 distinct tasks | the language, and error-to-fix repairs |
| Agent trajectories: Claude Code, motoko, opencode and pi sessions | ~7,860 passing episodes; 23,740 per-turn samples; 3,303 of them fix turns | the agent protocol: write, check, read the error, fix |
| `examples/` | 481 files, one feature each | canonical core syntax ([D-FT-5](./decisions.md)) |

## Rules

1. **Hold out first.** 24 benchmarks (12 core, 12 stretch) are frozen before extraction. Nothing
   from them enters training: not other models' solutions, not repo copies, not near-duplicates.
2. **Re-grade with today's compiler.** Every program is re-checked with the current
   `ailang check`; only what compiles today is kept. This strips stale syntax from older code.
3. **Negatives are never targets.** Failed attempts appear only as context (error-to-fix pairs,
   recovery trajectories). Loss is computed only on the turns that lead to a verified pass.
   Failure clusters every model hits become AILANG fixes, not training data.
4. **One tool schema.** Every trajectory is converted to pi's tools (`bash`, `read`, `write`,
   `edit`), the harness the model runs in.
5. **No teaching prompt in training inputs.** At most a slim prompt in the system turn.

## Habits to instill

Decided 2026-11-06 ([D-FT-4](./decisions.md)). The fine-tune targets agent mode, and the habits
below matter more than memorised stdlib names:

1. **Write through the tool, never answer in chat.** This was base 12B's main failure.
2. **Check every write.** Write → `ailang check` → read the error → apply the fix it names → run
   → compare the output with the spec.
3. **Look up before guessing.** Use `ailang docs std/<module>`, `ailang iface`, `ailang builtins`
   or `ailang examples search <topic>`. Keep lookups targeted: every one grows the context.
4. **Trust the compiler over memory.** When remembered syntax disagrees with an error message,
   follow the message. This is the habit that lets an older fine-tune work on a newer language.
5. **Stop when verified.** Don't re-verify in a loop or keep generating after a pass.

Behaviour to keep out of the data: filesystem search loops, grepping the repository for syntax,
and re-reading the whole teaching prompt every turn.

**How the data teaches them:**
- 2,345 trajectory samples already contain a lookup call; pilot-2 up-weights them.
- **Staleness pairs** (old code + today's error + the fixed version) come almost free. About 1,400
  past programs use `++` on strings, and the current error message names the fix.

## Pilot-1 mix

2,183 training samples, 4.79M tokens (1.62M of them targets):

| Kind | Samples | Tokens |
|---|---|---|
| Agent trajectory turns (first write, fix, final write, verify) | 1,079 | 3.2M |
| Standard Q/A: task → passing program (≤10 per benchmark, 86 benchmarks) | 585 | 0.6M |
| Repo programs, with the header comment as the instruction | 519 | 1.0M |

## Coverage gaps the work exposed

Measured 2026-11-06:

- **Examples barely reached the data.** Of ~397 examples that pass `check`, 25 reached the repo pool
  and 1 reached the pilot-1 mix. The duplicate filter kept a worktree copy of each example, and
  those copies were rejected. Pilot-2 reads `examples/` from the main checkout first.
- **Stdlib functions without an example:** 96 of 496 exported functions. The largest gaps are in
  `fs` (11 of the Result-returning variants), `xml` (7), `sem` (6), `datetime` (5),
  `embedding` (5) and `json` (5). `std/iter` and `std/trace` have no example at all.
- **Thin modules:** 23 modules have only one or two examples.

Filling these gaps is a docs improvement in its own right. The fine-tune can only learn what
`examples/` shows, so the training set doubles as a check on how well the examples cover the
language.
