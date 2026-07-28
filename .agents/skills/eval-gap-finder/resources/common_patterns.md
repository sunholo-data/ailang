# Common Eval Gap Patterns

This document catalogs frequently encountered gaps between Python and AILANG evaluation results, with known fixes.

## Quick Reference: Error Type → Fix

| Error Type | Common Cause | Fix |
|------------|--------------|-----|
| WRONG_LANG | Model wrote Python | Add "NOT Python" warning |
| PAR_001 | Syntax error | Add syntax example |
| type_error | Type unification failure | Add type example or design doc |
| logic_error | Wrong algorithm | Add algorithm example |
| EOF | Incomplete code | Model limitation |

---

## Pattern 1: Model Writes Python Instead of AILANG

**Symptoms:**
- Error category: `WRONG_LANG`
- Code contains `def`, `print(`, `for x in`, `lambda x:`

**Example broken code:**
```python
# Model wrote Python instead of AILANG
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
```

**Fix: Add prominent warning to prompt**

```markdown
## ⚠️ AILANG is NOT Python!

Do NOT use Python syntax in AILANG. Common mistakes:
- ❌ `def func():` → ✅ `func myFunc() -> Type = ...`
- ❌ `print(x)` → ✅ `println(x)`
- ❌ `for x in xs:` → ✅ `map(f, xs)` or recursion
- ❌ `lambda x: x+1` → ✅ `\x. x + 1`
```

---

## Pattern 2: Polymorphic ADT Type Inference

**Symptoms:**
- Error: `cannot unify type constructors: string vs int`
- Code defines generic Result/Either type
- Used with Option[int] from stdlib

**Example broken code:**
```ailang
type Result[a] = Ok(a) | Err(string)

pure func safe(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => Ok(n),        -- Fails! Option[int] vs Result[a]
    None => Err("bad")
  }
```

**Root Cause:** Type checker fails to unify polymorphic ADT with concrete Option type.

**Fix: Use monomorphic types**

```ailang
-- Works: specific type, not polymorphic
type IntResult = IntOk(int) | IntErr(string)

pure func safe(s: string) -> IntResult =
  match stringToInt(s) {
    Some(n) => IntOk(n),
    None => IntErr("bad")
  }
```

**Design Doc:** `design_docs/planned/v0_6_6/m-poly-adt-option-inference.md`

---

## Pattern 3: Missing `letrec` for Recursive Lambdas

**Symptoms:**
- Error: `undefined variable: f` in recursive lambda
- Model tried to define recursive local function

**Example broken code:**
```ailang
let f = \n. if n == 0 then 1 else n * f(n-1)  -- f not in scope!
```

**Fix: Use `letrec`**

```ailang
letrec f = \n. if n == 0 then 1 else n * f(n-1) in f(5)
```

---

## Pattern 4: String Iteration

**Symptoms:**
- Model tries `for c in str` or `str.chars()`
- Error: type mismatch or undefined function

**Example broken code:**
```ailang
-- Doesn't work
for c in "hello" { println(c) }
```

**Fix: Use `chars()` function**

```ailang
import std/string (chars)

-- chars converts string to list of single-character strings
let cs = chars("hello");  -- ["h", "e", "l", "l", "o"]
map(\c. println(c), cs)
```

---

## Pattern 5: Record Update Syntax

**Symptoms:**
- Model uses Python dict syntax: `{**old, "key": val}`
- Or doesn't know record update exists

**Example broken code:**
```ailang
let new = {old, age: 31}  -- Wrong syntax
```

**Fix: Use pipe syntax `{base | field: val}`**

```ailang
let person = {name: "Alice", age: 30};
let older = {person | age: person.age + 1};
-- Result: {name: "Alice", age: 31}
```

---

## Pattern 6: Pattern Matching on Strings

**Symptoms:**
- Model tries `match str { "a" => ... }`
- Error: strings can't be pattern matched directly

**Example broken code:**
```ailang
match input {
  "yes" => true,
  "no" => false
}
```

**Fix: Use if-then-else or guards**

```ailang
if input == "yes" then true
else if input == "no" then false
else false
```

---

## Pattern 7: List Comprehensions

**Symptoms:**
- Model writes `[x*2 for x in xs]`
- AILANG doesn't have list comprehensions

**Fix: Use map/filter**

```ailang
-- Python: [x*2 for x in xs]
map(\x. x * 2, xs)

-- Python: [x for x in xs if x > 0]
filter(\x. x > 0, xs)

-- Python: [x*2 for x in xs if x > 0]
map(\x. x * 2, filter(\x. x > 0, xs))
```

---

## Pattern 8: Mutable Variables / Loops

**Symptoms:**
- Model tries `let mut x = 0` or `x = x + 1`
- AILANG has no mutation

**Fix: Use recursion with accumulator**

```ailang
-- Sum a list (instead of for loop with accumulator)
letrec sum = \xs. \acc.
  match xs {
    [] => acc,
    [h, ...t] => sum(t)(acc + h)
  }
in sum([1,2,3])(0)
```

---

## Pattern 9: Exception Handling

**Symptoms:**
- Model writes `try/catch` or `raise`
- AILANG has no exceptions

**Fix: Use Result types**

```ailang
type IntResult = IntOk(int) | IntErr(string)

pure func divide(a: int, b: int) -> IntResult =
  if b == 0 then IntErr("division by zero")
  else IntOk(a / b)

-- Usage with pattern matching
match divide(10, 0) {
  IntOk(n) => println(show(n)),
  IntErr(msg) => println("Error: " ++ msg)
}
```

---

## Pattern 10: Import Errors

**Symptoms:**
- Error: undefined module or function
- Model guesses import paths

**Fix: Document correct import paths**

```ailang
-- Correct imports
import std/io (println, print, readLine)
import std/string (chars, stringToInt, substring, contains)
import std/list (map, filter, foldl, length, head, tail)
import std/json (decode, encode)
import std/prelude (show)
```

---

## Adding New Patterns

When you discover a new gap pattern:

1. **Document the symptoms** - What error message? What did model write?
2. **Show the broken code** - Minimal reproduction
3. **Identify root cause** - Prompt gap or language limitation?
4. **Provide the fix** - Working code example
5. **Test the fix** - Use `test_example.sh`
6. **Create design doc if needed** - For language limitations

Template:
```markdown
## Pattern N: [Name]

**Symptoms:**
- Error message or behavior

**Example broken code:**
```ailang
-- What model wrote
```

**Fix:**

```ailang
-- Working solution
```

**Design Doc (if language gap):** `design_docs/planned/vX_Y_Z/m-xxx.md`
```
