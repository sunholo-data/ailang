---
sidebar_position: 5
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

## Simulating the habits (pilot-2)

The habits are taught by **simulated agent episodes built from verified programs**. Every tool
result in them is real: the program is run with today's `ailang`, and an episode is kept only if
its output matches the task's expected output.
- **Format:** the task prompt comes from the eval harness itself, and the layout matches pi's
  captured traffic (pi's system prompt, the arm's prompt in the user turn, pi's tool-result
  strings). Each episode gets a fresh workspace path.
- **Solve episodes:** read the template → *sometimes* a targeted signature lookup
  (`ailang docs --all-functions std/string.split`, 1–3 lines) or `ailang examples search` →
  write → `ailang check` → run → stop.
- **Repair episodes:** write the old program → `ailang check` → the real error → edit → … →
  run → stop.
  - Source: staleness pairs, programs that passed when banked but fail today's checker.
  - Fixes come from gemma-4-31b and, for the harder ones, Opus (D-FT-7).
  - A fix is kept only if the program prints exactly what it printed when banked.
- **Thoughts:** every step carries a short thought built from facts in the episode, for example
  "The checker says: `++` is for lists only… So I'll switch the concatenation to `${...}`
  interpolation." That makes them correct by construction ([D-FT-8](./decisions.md)).

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
  and 1 reached the pilot-1 mix.
  - Cause: the extractor called `ailang check` with an **absolute path**. `ailang` derives the
    expected module name from the path it is given, so every path-named module
    (`module examples/runnable/x`) failed with MOD010.
  - Fixed 2026-11-06: paths are now relative to the repository root, and duplicate copies are
    tried main-checkout first. On the examples alone, kept files went from 27 to 466.
  - The same bug shrank the rest of the pilot-1 repo corpus, so pilot-2 starts from a re-extraction.
- **Stdlib functions without an example:** 96 of 496 exported functions. The largest gaps were in
  `fs` (11 of the Result-returning variants), `xml` (7), `sem` (6), `datetime` (5),
  `embedding` (5) and `json` (5). `std/iter` and `std/trace` had no example at all.
- **Thin modules:** 23 modules had only one or two examples.

### Examples expansion (D-FT-5, 2026-11-06)

63 new examples in `examples/runnable/stdlib/`. Each one:
- is checked by `ailang check` and pinned to its exact stdout in the manifest;
- is run in CI by `make verify-examples`;
- starts with a `-- Task:` header that doubles as its training instruction.

They split into:
- **49 stdlib examples:** strings, lists, Option/Result, math, I/O, seeded random, JSON, YAML,
  XML, HTML, map, set, regex, datetime, fs, arrays, bytes, iter, trace, crypto, JWT, compression,
  archives, audio, embeddings, simhash.
- **14 core-language examples (`lang_*`):** these target the syntax pilot-1 got wrong. They cover
  `let` bindings (there is no `x = …`), interpolation, records, ADTs with guards, tree folds,
  Option/Result, lambdas, tuples, effects, import aliases, numbers, tail recursion, polymorphism,
  and a parse-transform-print program.

**Coverage: 469 of 496 exported stdlib functions (95%), up from 400 (81%).** The remaining 27 are
mostly network, AI and DOM calls that cannot run deterministically in CI. The exceptions are
`sem`, `sharedmem`, `sharedindex` and `cognition`, which are still open.

**Writing the examples surfaced language and doc issues:**
- The teaching prompt teaches semicolons in blocks; `ailang fmt` strips them.
- The prompt lists `**` as an operator, but it does not parse.
- The prompt's `mapSlicesJoin` and JWT `verifyWithKid` examples are wrong.
- `ailang check` accepts non-exhaustive matches.
- There is no integer `max`/`min`, which models invent.
- `ailang fmt` fails its round-trip check on nested interpolation and on a `match` inside `${...}`.

Each of these teaches models a mistake, so they are routed as AILANG fixes, not training data.
The prompt mistakes were fixed the same day in teaching prompt v0.16.12 and agent prompt v0.9.1.
That pass also found the prompts claiming `print`, `readLine`, `getArgs`, `int_to_float` and
`float_to_int` work without imports. None does: the first three need an import, and the last two
do not exist at all (`intToFloat`/`floatToInt` live in `std/math`). That class of error is what
pilot-1's invented-function failures looked like. The tooling bugs are filed as issues #2144–#2146.

Filling coverage gaps is a docs improvement in its own right. The fine-tune can only learn what
`examples/` shows, so the training set doubles as a check on how well the examples cover the
language.
