# Z3 Verification Patterns

## 1. Pure Core, Effectful Shell

The most important architectural pattern for Z3 verification: **separate pure logic from effects**.

Z3 can only verify pure functions (`! {}`). By extracting computation into pure functions and keeping IO in a thin wrapper, you maximize the code surface that gets mathematically proven correct.

### Before: Nothing is verifiable

```ailang
-- BAD: Logic mixed with effects — Z3 cannot verify ANY of this
export func main() -> () ! {IO} {
  let subtotal = readInt();
  let tax = subtotal * 8 / 100;
  let discount = if subtotal > 1000 then subtotal / 10 else 0;
  let total = subtotal - discount + tax;
  println("Total: " ++ show(total))
}
```

### After: Core logic is provably correct

```ailang
-- GOOD: Pure core — Z3 proves total >= 0 for ALL non-negative inputs
export func calculateTotal(subtotal: int) -> int ! {}
requires { subtotal >= 0 }
ensures { result >= 0 }
{
  let tax = subtotal * 8 / 100;
  let discount = if subtotal > 1000 then subtotal / 10 else 0;
  subtotal - discount + tax
}

-- Thin effectful shell — just IO plumbing
export func main() -> () ! {IO} {
  let subtotal = readInt();
  let total = calculateTotal(subtotal);
  println("Total: " ++ show(total))
}
```

Run `ailang verify file.ail` — Z3 proves `calculateTotal` is correct for all valid inputs, not just test cases.

---

## 2. Contract Syntax

```ailang
export func name(params) -> ReturnType ! {}
requires { precondition1, precondition2 }
ensures { postcondition1, postcondition2 }
{
  body
}
```

| Clause | Purpose | Notes |
|--------|---------|-------|
| `requires { ... }` | Preconditions on inputs | Comma-separated predicates |
| `ensures { ... }` | Postconditions on output | Use `result` to refer to return value |
| `forall i: lo..hi => pred` | Bounded quantifier | Use inside ensures |
| `@verify(depth: N)` | Bounded recursion depth | Annotation before func |
| `! {}` | Empty effect set | **Required** for Z3 eligibility |

### Examples

```ailang
-- Precondition + postcondition
export func clamp(x: int, lo: int, hi: int) -> int ! {}
requires { lo <= hi }
ensures { result >= lo, result <= hi }
{
  if x < lo then lo
  else if x > hi then hi
  else x
}

-- Bounded quantifier
export func allPositive(xs: [int], n: int) -> bool ! {}
ensures { forall i: 0..n => i >= 0 }
{
  true
}

-- Bounded recursion
@verify(depth: 3)
export func factorial(n: int) -> int ! {}
requires { n >= 0, n <= 3 }
ensures { result >= 1 }
{
  if n == 0 then 1 else n * factorial(n - 1)
}
```

---

## 3. The Decidable Fragment

Z3 can only verify functions within the "decidable fragment." This is checked by `IsSMTEncodable` in `internal/smt/encodable.go`.

### Z3 CAN verify functions that:

- Are **pure** (`! {}` effect annotation, no IO/FS/Net/Clock)
- Use **encodable types**: `int`, `float`, `bool`, `string`, records, enum ADTs, lists
- Use `match` on enums, ADTs, records — nested patterns such as `Some(Some(x))` verify
- Have `requires`/`ensures` contracts
- Make **cross-function calls** to other pure functions (Z3 inlines them)
- Are **non-recursive**, or recursive with a `requires` that bounds the depth within the unrolling limit (default 2; `@verify(depth: N)` or `--verify-recursive-depth N`). Otherwise the result is INCONCLUSIVE (exit 1)
- Use **bounded quantifiers** (`forall i: 0..N => predicate`)

### Z3 CANNOT verify functions that:

- Have **effects** (`! {IO}`, `! {FS}`, etc.)
- Prove **unbounded recursion** — there is no induction; an unbounded recursive function is INCONCLUSIVE
- Pass a **function-typed parameter** to `map`/`filter`/`foldl`/`any`/`flatMap` (a lambda or a named top-level function is fine). They are unrolled to depth 3: map/filter length relations are proved for any list; other properties VERIFY when `requires` bounds the list length and are INCONCLUSIVE otherwise
- Use **bitwise** operators (`&`, `^`, `~`, `<<`, `>>`) — skipped as unencodable
- Use an **unannotated generic parameter** (`func f(x)`) — reaches Z3 as an error (#2400); annotate `x: int`
- Use **arrays** (use lists instead)
- Use unsupported string builtins: `trim`, `toUpper`, `toLower`, `split`, `chars`; or datetime builtins (`addDays`, …)

### What VERIFIED does NOT cover

- **Call-site preconditions**: a callee's `requires` is assumed, never checked where it is called
- **int64 overflow**: `int` is Z3's unbounded `Int`; runtime `int` wraps at 64 bits
- **NaN / IEEE rounding**: `float` is Z3's `Real`

### Supported operations

**Strings**: `length`, `substring`, `find`, `startsWith`, `endsWith`, `contains`, `"${a}${b}"` / `concat([..])` (string `++` is a type error)

**Lists**: `length`, `head`, `nth`, `contains`, `reverse`, `take`, `drop`, `::` (cons)

**Numeric**: `intToFloat`, `floatToInt`, `abs`

---

## 4. Contract Patterns by Domain

### Non-negativity (most common)
```ailang
export func price(qty: int, unit: int) -> int ! {}
requires { qty >= 0, unit >= 0 }
ensures { result >= 0 }
{ qty * unit }
```

### Boundedness
```ailang
export func percentage(x: int) -> int ! {}
requires { x >= 0 }
ensures { result >= 0, result <= 100 }
{ if x > 100 then 100 else x }
```

### Preservation (discount never exceeds price)
```ailang
export func applyDiscount(price: int, discount: int) -> int ! {}
requires { price >= 0, discount >= 0, discount <= price }
ensures { result >= 0 }
{ price - discount }
```

### Enum exhaustiveness
```ailang
type Priority = HIGH | MEDIUM | LOW

export func weight(p: Priority) -> int ! {}
ensures { result > 0 }
{
  match p {
    HIGH => 3,
    MEDIUM => 2,
    LOW => 1
  }
}
-- Z3 checks ALL enum variants automatically
```

### Record field invariants
```ailang
export func netAmount(inv: {subtotal: int, tax: int, discount: int}) -> int ! {}
requires { inv.subtotal >= 0, inv.tax >= 0, inv.discount >= 0, inv.discount <= inv.subtotal }
ensures { result >= 0 }
{ inv.subtotal - inv.discount + inv.tax }
```

### Cross-function chains
```ailang
-- Z3 inlines surcharge() when verifying lineTotal()
export func surcharge(p: Priority) -> int ! {}
ensures { result >= 0 }
{ match p { HIGH => 500, MEDIUM => 100, LOW => 0 } }

export func lineTotal(price: int, qty: int, p: Priority) -> int ! {}
requires { price >= 0, qty >= 0 }
ensures { result >= 0 }
{ price * qty + surcharge(p) }
```

### Tautology verification
```ailang
export func isValid(code: string) -> bool ! {}
ensures { result == (startsWith(code, "ORD-") && strLength(code) >= 8) }
{ startsWith(code, "ORD-") && strLength(code) >= 8 }
-- Z3 proves the body equals the ensures clause for ALL strings
```

---

## 5. Refactoring Recipes

### Recipe 1: Extract pure computation from effectful function

**Identify**: Function has `! {IO}` and does both IO and computation.

**Action**: Move computation into a new `! {}` function with contracts.

```ailang
-- BEFORE
export func processOrder(qty: int) -> () ! {IO} {
  let total = qty * 100 + 50;
  println("Total: " ++ show(total))
}

-- AFTER
export func orderTotal(qty: int) -> int ! {}
requires { qty >= 0 }
ensures { result >= 50 }
{ qty * 100 + 50 }

export func processOrder(qty: int) -> () ! {IO} {
  println("Total: " ++ show(orderTotal(qty)))
}
```

### Recipe 2: Separate enum logic from effects

**Identify**: `match` on enum inside an effectful function.

**Action**: Extract the match into a pure function.

```ailang
-- BEFORE
export func handlePriority(p: Priority) -> () ! {IO} {
  let msg = match p {
    HIGH => "urgent",
    MEDIUM => "normal",
    LOW => "deferred"
  };
  println(msg)
}

-- AFTER
export func priorityLabel(p: Priority) -> string ! {}
ensures { strLength(result) > 0 }
{
  match p {
    HIGH => "urgent",
    MEDIUM => "normal",
    LOW => "deferred"
  }
}

export func handlePriority(p: Priority) -> () ! {IO} {
  println(priorityLabel(p))
}
```

### Recipe 3: Decompose complex functions

**Identify**: Function with 4+ parameters and complex arithmetic.

**Action**: Break into smaller pure functions, each with targeted contracts. Z3 inlines callees via `define-fun`, proving the full chain.

### Recipe 4: Make recursion bounded

**Identify**: Recursive function you want to verify.

**Action**: Add `@verify(depth: N)` and constrain inputs in `requires`.

```ailang
@verify(depth: 4)
export func sumTo(n: int) -> int ! {}
requires { n >= 0, n <= 4 }
ensures { result >= 0 }
{ if n == 0 then 0 else n + sumTo(n - 1) }
```

---

## 6. Running `ailang verify`

```bash
ailang verify file.ail                              # Basic verification
ailang verify --verbose file.ail                    # Show generated SMT-LIB
ailang verify --json file.ail                       # Machine-readable output
ailang verify --verify-recursive-depth 5 file.ail   # Bounded recursion depth
ailang verify --strict file.ail                     # Also fail on skipped or inconclusive (partial → exit 1)
ailang verify --allow-inconclusive file.ail         # Exit 0 on inconclusive (bounded recursion)
```

### Output interpretation

| Status | Meaning |
|--------|---------|
| verified | Z3 proved the contract holds for ALL inputs |
| counterexample | Z3 found specific inputs that violate the contract |
| inconclusive | Bounded recursion found a model that may be an unrolling artifact — not proved (exit 1) |
| skipped | Function not in decidable fragment (reason shown); `requires`-only functions are skipped with `missing_postcondition` and never fail the run |
| error | Z3 solver error, `unknown`, or timeout |

Exit: 0 when the outcome is `verified`, `partial` (some proved, some skipped) or `no_obligations`; 1 for `unproved`, `inconclusive`, `counterexample`, `error`, or no Z3. Full table: `docs/docs/guides/contracts.mdx`.

When Z3 finds a counterexample, it prints the specific input values that break the contract — making bugs immediately actionable.

### Reference examples

- `examples/runnable/contracts/showcase.ail` — All features in one file
- `examples/runnable/contracts/invoice.ail` — Provably correct billing (4-deep call chain)
- `examples/runnable/contracts/access_control.ail` — Role-based authorization
- `examples/runnable/contracts/finance.ail` — Financial calculations

---

## 7. Design Checklist

### When writing a new function:

1. **Does it do IO?** Split into pure core + IO wrapper
2. **Does the core have invariants?** Add `requires`/`ensures`
3. **Is it recursive?** Add `@verify(depth: N)` and bound inputs in `requires`
4. **Does it call other functions?** Make those pure too (Z3 inlines them)
5. **Does it match on enums?** Extract to pure function — Z3 proves all branches

### When refactoring:

1. Look for **computation inside `! {IO}` functions** — extract to `! {}`
2. Look for **business rules without contracts** — add `requires`/`ensures`
3. Look for **enum combinations** — let Z3 verify all paths exhaustively
4. After refactoring, run: `ailang verify file.ail`

### The goal:

Maximize the **verifiable surface area** of your code. Every pure function with contracts is code that Z3 proves correct for ALL inputs — not just the test cases you thought of.
