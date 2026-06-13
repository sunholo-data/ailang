# AILANG — TOP DIALECT TRAPS (read this first, every turn)

You already know the algorithm. The thing that fails here is writing it in the
**wrong dialect** — reverting to Python/JS/Haskell habits. Before you write code,
obey these. They are the exact mistakes that break AILANG solutions:

1. **No list indexing `xs[0]`.** AILANG has no `[]` subscript operator. Use
   `nth(xs, 0)` (returns `Option`) or pattern-match `match xs { [x, ...rest] => ... }`.
   `import std/list (nth, head)`.

2. **`=`-body holds ONE expression — no `let x = ...;` after `=`.** The reflex
   `func f() -> T = let x = e; rest` is the single most common failure here. Fix it
   one of two ways:
   - **brace block** (drop the `=`): `func f() -> T { let x = e; rest }` — `;` is legal here
   - **let-in** (keep the `=`): `func f() -> T = let x = e in rest`
   `;` is valid ONLY inside `{ }`. For recursion use a named top-level `func` (there is
   no top-level `let rec`).

3. **`match x { Pat => expr, ... }`** — NOT `match x with` (that's Haskell/OCaml).
   Arms use `=>` and are separated by **commas**.

4. **Strings use `"${expr}"` interpolation. No backticks `` ` ``** (JS template
   literals are a lexer error). `++` is **list-only**; for strings use
   `"${a}${b}"`, or `concat([a, b])` / `join(sep, xs)` from `std/string`.

5. **Import stdlib functions before use.** `import std/string (concat, join)`,
   `import std/list (map, filter, nth)`. An "undefined variable" error almost always
   means a **missing import**, not a missing function — check stdlib before inventing one.

6. **Module + entry are mandatory.** Start with `module benchmark/solution`, and the
   entry point is `export func main() -> () ! {IO} { ... }` — `export` is REQUIRED.

When the compiler rejects your code, it tells you the fix (e.g. "`++` is list-only,
use `${}`"). Read that message and apply it — do not retry the same dialect.
