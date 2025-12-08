# M-DX-TEACHING-PROMPT-v0.4.6: Teaching Prompt Improvements

**Status**: Planned
**Version Target**: v0.4.6
**Priority**: HIGH (could fix 60%+ of eval failures)
**Effort**: 4-6 hours (prompt updates, validation, testing)
**Impact**: 68.6% → 75%+ 0-shot success rate (target)

## Problem Statement

**v0.4.5 eval analysis** reveals that **66% of 0-shot failures** (59/89) are due to models misunderstanding AILANG syntax and semantics:

- **40% WRONG_LANG**: Models generate Python/imperative code
- **26% PAR_001**: Module confusion, missing operators

**Root cause**: Current teaching prompt (v0.4.5) has gaps:
1. ❌ Doesn't explain what AILANG does NOT have (for loops, `%`, semicolons, multiple modules/file)
2. ❌ No multi-module project example
3. ❌ No JSON standard library example
4. ❌ No operator reference table
5. ❌ Effect declarations not emphasized enough

## Goals

1. **Eliminate WRONG_LANG errors** (36 failures → <10)
   - Add "What AILANG Does NOT Have" section
   - Emphasize functional style (no loops)

2. **Eliminate PAR_001 errors** (23 failures → <5)
   - Clarify one module per file
   - Add operator reference (no `%`)

3. **Improve JSON handling** (json_parse, json_encode benchmarks)
   - Add std/json example

4. **Improve effect understanding** (8 CAP_001 errors → 0)
   - Emphasize effect declarations

**Success metric**: v0.4.6 eval baseline shows ≥75% 0-shot success (up from 68.6%)

## Design

### Change 1: Add "What AILANG Does NOT Have" Section

**Location**: Near top of prompt, before syntax examples

**Content**:
```markdown
## What AILANG Does NOT Have (Important!)

AILANG is a **pure functional language**. It does NOT have:

❌ **No for/while loops** → Use recursion, `fold`, or `map` instead
❌ **No `%` modulo operator** → Use `mod(a, b)` function from std/prelude
❌ **No semicolons as statement separators** → Use block expressions `{ e1; e2 }`
❌ **No multiple modules in one file** → One module per file (file path = module name)
❌ **No mutable variables** → Use `let` bindings (immutable)
❌ **No null/undefined** → Use Option type: `Some(x)` or `None`
❌ **No exceptions** → Use Result type or explicit error values
❌ **No classes/objects** → Use records and functions

**Examples of Common Mistakes**:

```python
# ❌ WRONG (Python/imperative style):
for item in list:
    if item > 0:
        print(item)

# ✅ CORRECT (functional style):
func printPositive(xs: [int]) -> () ! {IO} =
  match xs {
    [] => (),
    x :: rest => {
      if x > 0 then println(show(x)) else ();
      printPositive(rest)
    }
  }
```

```python
# ❌ WRONG (missing operator):
let remainder = 5 % 3

# ✅ CORRECT (use mod function):
import std/prelude (mod)
let remainder = mod(5, 3)
```

```python
# ❌ WRONG (multiple modules in one file):
module myapp/data
type User = { name: string }

module myapp/storage  # ❌ Can't have second module!
func saveUser(...) = ...

# ✅ CORRECT (separate files):
File: myapp/data.ail
module myapp/data
export type User = { name: string }

File: myapp/storage.ail
module myapp/storage
import myapp/data (User)
export func saveUser(...) = ...
```
```

### Change 2: Add Multi-Module Project Example

**Location**: After module system section

**Content**:
```markdown
## Multi-Module Projects

**CRITICAL**: One module per file. The file path determines the module name.

**Example project structure**:
```
myapp/
  ├── data.ail       -- module myapp/data
  ├── storage.ail    -- module myapp/storage
  └── main.ail       -- module myapp/main
```

**File: myapp/data.ail**
```ailang
module myapp/data

export type User = { name: string, age: int }

export func validateAge(age: int) -> bool =
  age >= 0 && age <= 150
```

**File: myapp/storage.ail**
```ailang
module myapp/storage

import myapp/data (User)
import std/fs (writeFile, readFile)
import std/json (encode, decode)

export func saveUser(u: User, filename: string) -> () ! {FS} =
  writeFile(filename, encode(u))

export func loadUser(filename: string) -> User ! {FS} =
  let content = readFile(filename) in
  decode(content)
```

**File: myapp/main.ail**
```ailang
module myapp/main

import myapp/data (User, validateAge)
import myapp/storage (saveUser, loadUser)
import std/io (println)

export func main() -> () ! {IO, FS} = {
  let user = { name: "Alice", age: 30 };
  println("Saving user...");
  saveUser(user, "user.json");
  println("Loading user...");
  let loaded = loadUser("user.json");
  println("Loaded: " ++ loaded.name)
}
```

**Running**:
```bash
ailang run --entry main --caps IO,FS myapp/main.ail
```
```

### Change 3: Add JSON Handling Example

**Location**: After standard library section

**Content**:
```markdown
## JSON Handling (std/json)

**Import**: `import std/json (encode, decode)`

**Encoding (AILANG value → JSON string)**:
```ailang
import std/json (encode)

let user = { name: "Alice", age: 30 }
let json_string = encode(user)
-- Result: "{\"name\":\"Alice\",\"age\":30}"

let list_data = [1, 2, 3]
let json_array = encode(list_data)
-- Result: "[1,2,3]"
```

**Decoding (JSON string → AILANG value)**:
```ailang
import std/json (decode)

let json_str = "{\"name\":\"Alice\",\"age\":30}"
let parsed = decode(json_str)
-- Result: { name: "Alice", age: 30 }

-- Access fields:
let name = parsed.name    -- "Alice"
let age = parsed.age      -- 30
```

**Filtering JSON arrays**:
```ailang
import std/json (decode, encode)
import std/list (filter)

let json = "[{\"age\":25},{\"age\":35},{\"age\":20}]"
let people = decode(json)

-- Filter people over 30:
func isOver30(person: {age: int}) -> bool = person.age > 30

let filtered = filter(isOver30, people)
let result = encode(filtered)
-- Result: "[{\"age\":35}]"
```
```

### Change 4: Add Operator and Function Reference

**Location**: After syntax section, before examples

**Content**:
```markdown
## Operators and Functions Reference

**Arithmetic operators**:
- `+`, `-`, `*`, `/` - Standard arithmetic
- **NO `%` operator** → Use `mod(a, b)` from std/prelude
- `**` - Exponentiation

**Comparison operators**:
- `>`, `<`, `>=`, `<=`, `==`, `!=`

**Logical operators**:
- `&&` (and), `||` (or), `!` (not)

**String operators**:
- `++` - String concatenation
- Example: `"Hello" ++ " " ++ "World"` → `"Hello World"`

**List operators**:
- `::` - Cons (prepend element)
- Example: `1 :: [2, 3]` → `[1, 2, 3]`
- **NO `++` for lists** → Use `append` from std/list

**Common functions (std/prelude - auto-imported)**:
- `mod(a, b)` - Modulo (remainder)
- `show(x)` - Convert value to string
- `not(b)` - Logical NOT (alternative to `!`)

**List functions (std/list)**:
- `map(f, xs)` - Apply function to each element
- `filter(pred, xs)` - Keep elements matching predicate
- `fold(f, init, xs)` - Left fold (reduce)
- `append(xs, ys)` - Concatenate two lists
- `length(xs)` - List length
- `reverse(xs)` - Reverse list

**String functions (std/string)**:
- `length(s)` - String length
- `substring(s, start, len)` - Extract substring
- `split(s, sep)` - Split by separator
- `join(sep, xs)` - Join list of strings

**Effect-ful I/O functions (std/io)**:
- `print(s)` - Print without newline (requires `! {IO}`)
- `println(s)` - Print with newline (requires `! {IO}`)
- `readLine()` - Read line from stdin (requires `! {IO}`)

**Effect-ful File functions (std/fs)**:
- `readFile(path)` - Read file contents (requires `! {FS}`)
- `writeFile(path, content)` - Write file (requires `! {FS}`)
- `fileExists(path)` - Check if file exists (requires `! {FS}`)
```

### Change 5: Emphasize Effect Declarations

**Location**: In effects section, add prominent warning

**Content**:
```markdown
## Effect Declarations (CRITICAL!)

**⚠️ IMPORTANT**: Every function that performs I/O MUST declare effects in its type signature.

**Functions requiring `! {IO}`**:
- `print(...)`, `println(...)`
- `readLine()`
- Any function that calls these

**Functions requiring `! {FS}`**:
- `readFile(...)`, `writeFile(...)`
- `fileExists(...)`
- Any function that calls these

**Multiple effects**: `! {IO, FS}`

**Examples**:

```ailang
-- ❌ WRONG (missing effect declaration):
export func main() -> () {
  println("Hello")  -- ERROR: println requires IO effect!
}

-- ✅ CORRECT:
export func main() -> () ! {IO} {
  println("Hello")
}

-- ❌ WRONG (missing FS effect):
func loadConfig() -> string ! {IO} {
  let content = readFile("config.txt")  -- ERROR: readFile requires FS!
  content
}

-- ✅ CORRECT (both IO and FS):
func loadConfig() -> string ! {IO, FS} {
  println("Loading config...");
  let content = readFile("config.txt");
  content
}
```

**Rule of thumb**: If your function uses print/println/readLine/readFile/writeFile, declare the effect!
```

## Implementation Plan

### Phase 1: Update Teaching Prompt (2-3 hours)

1. Add "What AILANG Does NOT Have" section
2. Add multi-module project example
3. Add JSON handling example
4. Add operator/function reference table
5. Emphasize effect declarations

**Files**:
- `prompts/v0.4.6.md` (new version)
- `prompts/versions.json` (register new version)

### Phase 2: Validate Prompt (1 hour)

1. Run through skill: `ailang prompt --version v0.4.6 > /tmp/prompt.md`
2. Manually review for clarity
3. Check all code examples compile
4. Verify markdown formatting

### Phase 3: Test with Models (1-2 hours)

1. Run quick eval with v0.4.6 prompt (3 dev models, 10 benchmarks):
   ```bash
   # Update models.yml to use new prompt version
   ailang eval-suite --benchmarks numeric_modulo,json_parse,multi_module_imports,\
     api_call_json,cli_args,config_file_parser,json_encode,pipeline,\
     record_update,float_eq --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash
   ```

2. Check if targeted failures improve:
   - numeric_modulo: Should succeed (no `%` confusion)
   - multi_module_imports: Should succeed (one module per file)
   - json_parse/json_encode: Should improve

3. If improvements seen, proceed to full baseline
4. If no improvement, iterate on prompt

### Phase 4: Full Eval Baseline (15-20 minutes)

```bash
make eval-baseline EVAL_VERSION=v0.4.6 FULL=true
```

**Expected results**:
- 0-shot success: 68.6% → **75%+** (target)
- WRONG_LANG errors: 36 → **<10**
- PAR_001 errors: 23 → **<5**
- Final success (with repair): 68.6% → **78%+**

### Phase 5: Update Documentation (30 minutes)

1. Update CHANGELOG.md with prompt improvements
2. Update website docs if needed
3. Commit changes

## Risks and Mitigations

**Risk 1**: Prompt becomes too long
- **Mitigation**: Remove outdated sections, consolidate examples
- **Current prompt size**: ~8,000 tokens
- **Target**: <10,000 tokens (within model context)

**Risk 2**: No improvement in eval results
- **Mitigation**: Iterative testing (Phase 3) before full baseline
- **Fallback**: Revert to v0.4.5 prompt, try alternative approaches

**Risk 3**: Models still confused by multi-module projects
- **Mitigation**: Add even more explicit example
- **Alternative**: Consider creating benchmarks that don't require multi-module

## Success Criteria

1. ✅ v0.4.6 prompt includes all 5 changes (What NOT to have, multi-module, JSON, operators, effects)
2. ✅ All code examples in prompt compile successfully
3. ✅ Quick eval (10 benchmarks) shows improvement on targeted failures
4. ✅ Full baseline shows ≥75% 0-shot success rate (up from 68.6%)
5. ✅ WRONG_LANG errors reduced to <10 (from 36)
6. ✅ PAR_001 errors reduced to <5 (from 23)

## Related Work

- **v0.4.5 analysis**: `design_docs/planned/v0_4_6/EVAL_FAILURE_ANALYSIS_v0_4_5.md`
- **Current prompt**: `prompts/v0.4.5.md`
- **Prompt versioning**: `prompts/versions.json`
- **Eval harness**: `internal/eval_harness/`

## Timeline

- **Phase 1** (Update prompt): 2-3 hours
- **Phase 2** (Validate): 1 hour
- **Phase 3** (Quick test): 1-2 hours
- **Phase 4** (Full baseline): 20 minutes
- **Phase 5** (Documentation): 30 minutes

**Total**: 5-7 hours

**Target completion**: v0.4.6 release (early December 2025)
