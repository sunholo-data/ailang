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

---

# AILANG — HOW TO CONVERGE (build it GREEN, never big-bang)

Runs that PASS keep the file type-checking at EVERY step. Runs that FAIL write a big
file that lands 30+ errors at once and can never dig out. So grow it green:

1. **ORIENT before writing** (kills "undefined" / "not exported"). Read the real API —
   do not invent functions:
   - `ailang iface --compact <module>` — exact typed signatures of a dependency
   - `ailang docs <module>` · `ailang examples search "<concept>"`

2. **SKELETON FIRST.** Write `module …` + every `export func` SIGNATURE with a stub
   body that compiles, then check. Reach GREEN *before* adding any logic:
   ```ailang
   export func parseDoc(s: string) -> [int] ! {} = []   -- stub
   export func title(s: string) -> string ! {} = ""     -- stub
   ```

3. **PURE CORE + EFFECTFUL SHELL.** Put logic in pure funcs (`! {}`); keep `main` a thin
   `! {IO}` shell. Pure funcs are small (each edit stays green) AND the only thing the
   verifier can prove.

4. **FILL ONE FUNC AT A TIME.** Replace one stub → check → green? next. Red? fix THAT
   func before moving on — never accumulate red. Make SMALL edits; do NOT rewrite the
   whole file (rewriting working code is the #1 cause of an unrecoverable spiral).

5. **CONTRACTS = THE SPEC (your edge).** On a pure func, state the invariant; Z3 proves
   it for ALL inputs (or hands you a concrete counterexample):
   ```ailang
   export func clamp(x: int) -> int ! {}
   requires { x >= 0 }
   ensures  { result >= 0 }
   { if x > 100 then 100 else x }
   ```

**Your convergence signal is `ailang ai-check FILE`** — type-check + contract proof in
one JSON, not just `check`. Green check + verified contracts = done.
