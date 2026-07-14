---
sidebar_position: 40
title: One-Shot Stdlib Discovery
description: Discover the whole AILANG stdlib in one command — --all-functions, unknown-module recovery, and docs prelude — without writing throwaway probe files.
---

# One-Shot Stdlib Discovery (for AI Agents)

When an agent writes AILANG from scratch, most of its extra cost versus a familiar
language is **stdlib discovery overhead**: repeatedly running `ailang docs std/X` to
find a function name, or writing a throwaway file to `/tmp` to see whether a symbol
needs an import. These three surfaces answer those questions in one command each.

All three are rendered from the **live** compiler mechanisms (the AST, the prelude
injector, the loader's implicit-import table) — there is no hand-maintained table that
can drift out of sync with the language.

## `ailang docs --all-functions [filter]`

Dumps one deterministic, grep-able line per stdlib export — signature rendered from the
AST, so effect rows are always complete:

```
std/clock.now: (()) -> int ! {Clock} -- Returns epoch time in seconds
std/list.map: [a, b]((a) -> b, list[a]) -> list[b] -- Apply f to every element
prelude.println: string -> () ! {IO} -- Available without import in entry modules
```

Modules are sorted; exports appear in file (declaration) order. The optional trailing
positional filters case-insensitively over the whole line:

```bash
ailang docs --all-functions timestamp   # every export mentioning "timestamp"
ailang docs --all-functions | grep '! {IO}'   # every IO-effecting export
```

If a stdlib file fails to parse, the command exits non-zero and names the file — it
never silently drops a row.

## Unknown-module recovery

A mistyped stdlib module name is now recoverable from the error itself:

```
$ ailang check my.ail   # my.ail has `import std/time (now)`
Error: stdlib module not found: std/time
searched:
  ...
tip: set AILANG_STDLIB_PATH=/path/to/ailang/std or use --stdlib-path flag

did you mean: std/clock?
available: std/ai, std/array, ..., std/zip (44 modules)
```

A curated alias table (`time→clock`, `date→datetime`, `http→net`, ...) is tried first —
it catches slips that are too far for a typo heuristic — then a Levenshtein ≤2 pass over
the live module list catches near-misses like `std/lst → std/list`.

## `ailang docs prelude`

Shows exactly what is callable **without any import** in an entry module (a module with
an exported `main`): `println`, the `show` builtin, and the implicit `std/option` /
`std/result` types and constructors — with the scope rules (entry-only, lowest
precedence, silent shadowing).

```bash
ailang docs prelude
ailang docs --list        # footer points at `ailang docs prelude`
```

Library modules (no `main`) do **not** get the prelude, so they must import
`Option`/`Result` explicitly.
