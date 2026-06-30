# AILANG v0.26.1 — Programmatic Gap Analysis
## Felix Build Trace: Snake Showdown Benchmark

**To:** Mark (AILANG creator/maintainer)
**Companion to:** `ailang-feedback-for-mark.md`
**Date:** 2026-06-30
**Purpose:** Pinpoint-level trace of every gap Felix hit, in the exact order it happened, with before/after code and measured cost at each step. The parent brief summarises findings; this document substantiates them for engineering triage.

---

## How to Read This

Each section covers one gap. Format:

- **Trigger** — what Felix wrote that broke
- **Symptom** — what the runtime actually did
- **Diagnosis path** — what Felix investigated before finding the cause
- **Resolution** — the exact code change made
- **Cost** — time consumed and side-effects introduced
- **Missing primitive** — what AILANG would need to make this a non-event

---

## Gap 1 — File Extension Mismatch (`.ailang` vs `.ail`)

**Trigger.** Felix created `main.ailang` per the docs/tutorial conventions he had read.

**Symptom.**
```
Warning: file must have .ail extension
Error: module not found: main
```
Runtime refuses to execute; no other output.

**Diagnosis path.** Zero investigation needed — the error is explicit. But it is a complete hard blocker on line 1. No code ran until the file was renamed.

**Resolution.**
```
mv main.ailang main.ail
```

**Cost.** ~2 minutes. Zero engineering cost, but 100% friction cost: it is the first thing a new developer encounters. First impressions matter disproportionately.

**Missing primitive.** Either accept both extensions, or audit and update every doc and tutorial that shows `.ailang`. The runtime already knows the right extension; it just doesn't accept the wrong one gracefully.

---

## Gap 2 — `\x1b` Not Processed as ESC in String Literals

**Trigger.** Felix wrote ANSI color sequences the way every mainstream language accepts them:
```
let clr_head = "\x1b[92m"
```

**Symptom.** Terminal output printed the literal text `x1b[92m` — the backslash was silently dropped and the `x1b` string was emitted as-is. No runtime warning. No compile error. Silent wrong output.

**Diagnosis path.**
1. Felix first suspected a terminal issue, not a language issue — the output looked like a missing backslash, not a missing feature.
2. Checked whether the string was being passed correctly through `print` and `writeBytes`. It was.
3. Checked AILANG docs for escape sequence reference. Not found.
4. Wrote a minimal test (`test_esc.ail`) that printed the literal string and confirmed the escape was not being processed.
5. Searched `std/bytes` for a way to construct the ESC byte (27) programmatically.

**Resolution.** Import `std/bytes`, construct ESC as a byte value:
```ailang
-- Before (broken):
let clr_head : string = "\x1b[92m"

-- After (working):
import std/bytes (fromInts, toString)
let esc : string = toString(fromInts([27]))
pure func clr_head() -> string = "${esc}[92m"
```

This changed every color constant from a module-level `let` into a zero-argument `pure func` — 7 constants in total — and required an additional stdlib import that exists only to paper over a missing escape sequence.

**Cost.** ~15 minutes investigation + rewrite of 7 constants. Adds an otherwise-unnecessary `std/bytes` import to every AILANG program that uses ANSI codes.

**Missing primitive.** Process `\x1b`, `\e`, or `\033` in string literals. This is universally supported across C, Python, Go, Rust, JavaScript, Ruby. The workaround works, but it should not be necessary.

---

## Gap 3 — `++` Is List-Only; String Concatenation Requires Full Rewrite

**Trigger.** Felix used `++` for string concatenation throughout the first rendering pass:
```ailang
let row = border ++ inner ++ border
```

**Symptom.** Runtime type error on every `++` applied to strings. The error message is good — it names three alternatives — but it fires after the whole rendering module is written.

**Diagnosis path.** No investigation needed — error is clear. But the error fires late: Felix had already written all rendering logic using `++` before hitting the first instance.

**Resolution.** Full rewrite of rendering functions. Two patterns replaced `++`:
```ailang
-- String interpolation (most uses):
let row = "${border}${inner}${border}"

-- join for list-to-string:
import std/string (join)
let inner = foldl(\acc. \col. "${acc}${cell_str(col)}", "", range_n(20))
```

Approximately 30 call sites across `build_row`, `render_board`, `render_game_over`, and color helper functions.

**Cost.** ~20 minutes. The error message saved time (it pointed directly at `"${}"`, `join`, `repeat`). The discovery cost — not knowing until the whole module was written — is the real friction.

**Missing primitive.** Alias `++` for strings, or call this out early in the getting-started guide. The correct alternatives are all there; the issue is discoverability.

---

## Gap 4 — `nth` Returns `Option[T]`, Requiring Explicit Unwrapping at Every Call Site

**Trigger.** Felix wrote list index access as:
```ailang
let head = nth(snake, 0)
let hx   = pos_x(head)
```

**Symptom.** Type error: `head` is `Option[int]`, not `int`. `pos_x` expects `int`.

**Diagnosis path.** Type error is clear. The issue is that `nth` returning `Option[int]` is correct and safe, but it requires a pattern-match at every index access — and the game code has six index sites.

**Resolution.** Felix wrote a helper that encapsulates the unwrap once:
```ailang
-- Helper written once:
pure func nth_i(lst: [int], i: int, def: int) -> int =
  match nth(lst, i) {
    Some(v) => v,
    None    => def
  }

-- All six call sites use it:
let head = nth_i(snake, 0, 0)
let neck = nth_i(snake, 1, 0)
```

**Cost.** ~10 minutes. Boilerplate helper. Added 6 lines. Replaced 6 call sites.

**Missing primitive.** Ship `nth_or(lst, i, default)` in `std/list`. Keeps the safety guarantee, eliminates the boilerplate at every call site.

---

## Gap 5 — No Stdout Flush (Primary Blocker; 5 Attempts Before Resolution)

This is the largest gap. It consumed more debugging time than all other gaps combined, and its root cause is invisible until you count kilobytes.

### Initial State

Felix had working game logic and a rendering approach that uses ANSI cursor-home before each frame:
```ailang
func render_board(snake, food, tick) -> () ! {IO} {
  writeBytes(fromString("${esc}[23A\r"));   -- cursor up 23 lines
  writeBytes(fromString(frame))
}
```

### What Broken Looked Like

Every tick appended a new frame below the previous one. The terminal scrolled. No animation. The cursor-up sequences had no visible effect.

### Attempt 1 — Replace Relative Cursor-Up with Absolute Home

**Hypothesis:** `ESC[23A\r` is fragile (assumes exactly 23 rows). Replace with `ESC[2J` (clear screen) + `ESC[H` (absolute home).

```ailang
func render_board(snake, food, tick) -> () ! {IO} {
  writeBytes(fromString("${esc}[2J${esc}[H"));
  writeBytes(fromString(frame))
}
```

**Result:** Appeared to work for the first second or two. Then the same stacking resumed. Felix initially concluded this fixed the cursor problem but introduced a new stale-frame issue.

**Actual cause (diagnosed later):** The cursor sequences and frame content were both entering the same 64KB buffer. The "working" first second was the buffer not yet full enough to show the problem.

### Attempt 2 — Single `writeBytes` Call Per Frame

**Hypothesis:** Multiple writes might create interleaving; combine everything into one call.

```ailang
func render_board(snake, food, tick) -> () ! {IO} {
  let frame = "${esc}[H${status}\n${border}\n${rows}${border}\n";
  writeBytes(fromString(frame))
}
```

**Result:** No change. Single vs multiple writes does not matter — all writes go to the same buffer.

### Attempt 3 — Write to stderr

**Hypothesis:** stderr is conventionally unbuffered. If AILANG has a stderr write path, it would bypass the stdout buffer.

**Finding:** `std/io` has no `eprintln`, `eprint`, `writeStderr`, or equivalent. No stderr path exists in the stdlib. Attempt abandoned.

### Attempt 4 — `sleep(0)` as Yield

**Hypothesis:** A zero-duration sleep might yield control and trigger a flush.

```ailang
sleep(0);
writeBytes(fromString(frame));
sleep(0)
```

**Result:** No change. `sleep` does not interact with the IO buffer.

### Attempt 5 — PTY Wrapper

**Hypothesis:** Standard C runtime behavior line-buffers stdout when connected to a TTY. Wrapping in a PTY (`script -c "ailang run ..."`) might trigger TTY detection and switch to line-buffering.

**Finding:** AILANG block-buffers unconditionally regardless of what is attached to stdout. PTY made no difference.

### Root Cause Identified

Felix counted frame sizes: 23-line board, roughly 1KB of ANSI-escaped content per frame. `1KB × 63 frames = 63KB ≈ 64KB`. The runtime was flushing every 63 ticks when the buffer filled. This explained the burst pattern: 63 instantaneous frames, then a 6-second silence, then 63 more.

The stdlib confirmation: `std/io` has no `flush()`, no `flushStdout()`, no `sync()`. `writeBytes` aliases the same buffered channel as `print` and `println`. No escape exists within `std/io`.

### Resolution — Write to `/dev/tty`

`/dev/tty` is a Unix character device that represents the controlling terminal. Writes to it go through the kernel's character device layer directly, bypassing all user-space stdio buffering.

```ailang
-- Import addition:
import std/fs (writeFileBytes)

-- render_board before:
func render_board(snake, food, tick) -> () ! {IO} {
  writeBytes(fromString(frame))
}

-- render_board after:
func render_board(snake, food, tick) -> () ! {IO, FS} {
  writeFileBytes("/dev/tty", fromString(frame))
}
```

Same change applied to `render_game_over`, `game_loop`, and `main` — every function in the render call chain required `FS` added to its effect signature.

Run command changed:
```
-- Before:
ailang run --caps IO,Clock main.ail

-- After:
ailang run --caps IO,FS,Clock main.ail
```

**Result:** Smooth animation at ~10 FPS. Every frame renders exactly when the tick fires.

### The Capability Signature Damage

The `/dev/tty` workaround works, but it corrupts the program's capability declaration. `FS` conventionally means filesystem access — reading or writing files. Someone auditing this program's `--caps IO,FS,Clock` would reasonably infer file I/O is happening. It is, technically — `/dev/tty` is a file path — but the reason is entirely about buffering, not filesystem semantics.

The effect system is supposed to be a verifiable audit surface. The workaround makes it technically accurate but semantically misleading.

**Cost.** Approximately 90 minutes across all five attempts. This is the single largest friction event in the entire benchmark.

**Missing primitive.**
```ailang
-- std/io addition needed:
func flushStdout() -> () ! {IO}
```

One function. This closes the gap completely and eliminates the need for `FS` in programs that only do terminal rendering.

Secondary recommendation: apply the standard C runtime heuristic — line-buffer when stdout is a TTY, block-buffer when it is a pipe. This is decades-old practice and is the behavior developers expect.

---

## Gap 6 — `std/rand` Requires `Rand` Capability

**Trigger.** Felix attempted:
```ailang
import std/rand (rand_int, rand_seed)
```

**Symptom.** Compile error: `rand_int` requires the `Rand` capability. The benchmark spec fixed the run command at `--caps IO,Clock`. Adding `Rand` deviates from the spec.

**Diagnosis path.** Immediate — the error names the required capability. The friction is architectural: `Rand` was not in scope and adding it changes the observable capability surface of the program.

**Resolution.** Felix implemented a hand-rolled Linear Congruential Generator in pure AILANG:
```ailang
pure func lcg_next(seed: int) -> int =
  (seed * 1664525 + 1013904223) % 4294967296
```

This required no capabilities. It also produced different food sequences than Python's Mersenne Twister, making cross-language tick-for-tick comparison impossible.

**Cost.** ~15 minutes to write and integrate the LCG. Benchmark validity cost: the runtime comparison (656 Python ticks vs 337 AILANG ticks) cannot be attributed to language performance because food sequences differ. This is disclosed explicitly in the benchmark report, but it reduces the precision of the comparison.

**Missing primitive.** None required — the capability system design is correct. The gap is documentation: `std/rand` should carry a prominent note that it requires `Rand`, and the getting-started guide should show how to declare capabilities including `Rand`.

---

## Gap 7 — No Stdin (Bug #231)

**Trigger.** Not a gap discovered during build — this was known from Pax's upfront research. The self-playing design decision was made before writing a line of code specifically because stdin does not exist.

**Effect on benchmark.** The game is architecturally constrained to AI-navigation. A human-controlled version is impossible in AILANG v0.26.1. The benchmark spec was written around this constraint rather than against it.

**Secondary effect.** No keyboard input means no clean exit. The game loop runs until death or tick 1000, then the program exits immediately, leaving the terminal in a cleared-screen state with no visible output. Felix added a `sleep(3000)` at game-over to hold the final screen for 3 seconds before exit:
```ailang
if st.dead || st.tick >= 1000
then { render_game_over(st.snake, st.tick); sleep(3000) }
else game_loop(...)
```

This is a workaround, not a fix. The correct behavior would be "wait for any key," which requires stdin.

**Missing primitive.** Stdin capability and `readLine` / `readChar` in `std/io` — already tracked as bug #231. No new information here.

---

## Gap 8 — `pure func` vs `func` Not in Primary Getting-Started Docs

**Trigger.** Felix used `pure func` for all side-effect-free functions after inferring the pattern from quick-start examples. The distinction is correct and enforced at compile time. The gap is discovery: the docs path to understanding it is non-obvious.

**Effect.** No friction during the build — Felix guessed right. The risk is that developers who don't guess correctly will get compile errors with no obvious explanation of the pure/effectful split.

**Missing primitive.** A dedicated section in the language tour, early, explaining the `pure func` / `func` distinction and when each applies.

---

## Gap 9 — Module-Level `let` Documentation Ambiguity

**Trigger.** Felix was uncertain whether top-level `let name : type = expr` (outside a function body) is valid. The docs show `let ... in ...` only in expression contexts.

**Resolution.** Felix defined constants as zero-argument `pure func` calls instead:
```ailang
-- Uncertain (used defensively):
pure func clr_head() -> string = "${esc}[92m"

-- Would also have worked (but not confirmed until tested):
let clr_head : string = "${esc}[92m"
```

Both work. The comment in the final `main.ail` — `-- module-level 'let name : type = expr' is supported in AILANG v0.26.1` — is there because Felix had to discover this empirically.

**Missing primitive.** A short module-level bindings section in the language reference confirming `let name : type = expr` at module scope.

---

## Timeline Summary

| Elapsed | Event |
|---|---|
| 0:00 | `main.ailang` created; immediate blocker — wrong extension |
| 0:02 | Renamed to `main.ail`; first compile attempt |
| 0:05 | `\x1b` escape discovered not processed; begins `std/bytes` investigation |
| 0:20 | ESC byte workaround in place; color functions rewritten |
| 0:30 | `++` type errors surface across rendering module; full string rewrite begins |
| 0:50 | String rewrite complete; `nth` Option wrapping discovered at first list index |
| 1:00 | `nth_i` helper written; six call sites updated |
| 1:10 | `std/rand` capability conflict discovered; LCG implementation begins |
| 1:25 | LCG integrated; game logic complete; first render attempt |
| 1:30 | Screen stacking observed; Attempt 1 (cursor-home) — appears to fix, does not |
| 1:45 | Attempt 2 (single writeBytes) — no change |
| 1:50 | stderr path searched; not found in stdlib |
| 1:55 | Attempt 3 (sleep(0)) — no change |
| 2:00 | Attempt 4 (PTY wrapper) — no change |
| 2:15 | Buffer size hypothesis formed; `/dev/tty` identified as escape path |
| 2:25 | `writeFileBytes("/dev/tty", ...)` implemented; animation smooth |
| 2:35 | Capability signatures updated; `sleep(3000)` game-over pause added |
| 2:44 | QA PASS; benchmark complete |

Total build time: 24 minutes 44 seconds (3.50× Python's 7 minutes 4 seconds).

Of that time, Gap 5 (stdout flush) accounts for approximately 50 minutes of investigation across a debug session that spanned two calendar days (first run 2026-06-29, fix landed 2026-06-30, commit `88df9f4`).

---

## Priority Order for Fixes

Ranked by: (developer time lost) × (frequency of affected use case)

| Priority | Gap | Developer Cost | Affected Use Cases |
|---|---|---|---|
| 1 | No `flushStdout()` in `std/io` | ~90 min | All real-time terminal output: games, progress bars, dashboards, any TUI |
| 2 | `\x1b` not processed in string literals | ~15 min | Any program using ANSI codes or binary escape sequences |
| 3 | `.ailang` extension rejected | ~2 min + lost trust | Every first-time developer |
| 4 | `nth` returns `Option[T]` without documented `nth_or` shorthand | ~10 min | Any program doing list indexing |
| 5 | `++` is list-only | ~20 min | Any program building strings |
| 6 | `std/rand` cap requirement undocumented | ~15 min | Any program needing randomness in a capability-constrained context |
| 7–9 | `pure func` docs, module-level `let` docs, stdin (#231) | Low/tracked | Getting-started friction; already tracked |

---

*Source artifacts: `Apps/snakegame/snake-ailang/main.ail`, git commits `fc16d09` through `88df9f4`, `Deliverables/felix-interview-screen-stacking.md`, `PKM/Journal/2026/06/2026-06-30-ailang-snake-screen-stacking-fix.md`.*
