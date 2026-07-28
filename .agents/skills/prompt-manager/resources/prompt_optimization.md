# AILANG Teaching Prompt Optimization Guide

## Core Principle: Maximum Information Density

AI teaching prompts should be **concise, dense, and reference-based**. Every token counts.

### Optimization Strategies

#### 1. Reference External Documentation Instead of Duplicating

**❌ Bad (verbose duplication):**
```markdown
## Module System

AILANG has a module system that allows you to organize code into separate files.

### Import Syntax
You can import functions from other modules using the import statement.
The syntax is: import "path/to/module"

### Export Syntax
To export functions from a module, use the export keyword.
The syntax is: export functionName

### Example
Here's an example of importing:
```ailang
import "std/io"
import "std/fs"

let main = () => {
  io.println("Hello")
}
```
```

**✅ Good (concise with references):**
```markdown
## Module System

**Syntax:** `import "path/to/module"` | `export functionName`
**Stdlib:** `std/io`, `std/fs`, `std/prelude`, `std/json`, `std/net`
**Details:** See [Module System Guide](../docs/guides/modules.md)

**Example:**
```ailang
import "std/io"
let main = () => io.println("Hello")
```
```

**Reduction:** ~200 tokens → ~50 tokens (-75%)

#### 2. Use Tables for Reference Data

**❌ Bad (prose format):**
```markdown
## Available Builtins

The _str_len function takes a string and returns its length as an integer.
The type signature is: string -> int

The _str_upper function takes a string and returns an uppercase version.
The type signature is: string -> string

The _str_lower function takes a string and returns a lowercase version.
The type signature is: string -> string
```

**✅ Good (table format):**
```markdown
## String Builtins

| Function | Type | Description |
|----------|------|-------------|
| `_str_len` | `string -> int` | String length |
| `_str_upper` | `string -> string` | Uppercase |
| `_str_lower` | `string -> string` | Lowercase |

Full list: `ailang builtins list --by-module std/string`
```

**Reduction:** ~120 tokens → ~40 tokens (-67%)

#### 3. Consolidate Examples

**❌ Bad (separate examples for each feature):**
```markdown
## Lambda Example
```ailang
let double = (x) => x * 2
```

## Function Call Example
```ailang
let result = double(5)
```

## Type Annotation Example
```ailang
let add : (int, int) -> int = (a, b) => a + b
```

## Pattern Matching Example
```ailang
let factorial = (n) => match n {
  | 0 => 1
  | n => n * factorial(n - 1)
}
```
```

**✅ Good (combined comprehensive example):**
```markdown
## Core Syntax Example
```ailang
-- Lambda with type annotation
let add : (int, int) -> int = (a, b) => a + b

-- Pattern matching + recursion
let factorial = (n) => match n {
  | 0 => 1
  | n => n * factorial(n - 1)
}

-- Usage
let result = add(3, 4)  -- 7
```
```

**Reduction:** ~180 tokens → ~80 tokens (-56%)

#### 4. Link to Implementation for Complex Details

**❌ Bad (detailed explanation):**
```markdown
## Type Inference

AILANG uses the Hindley-Milner type inference algorithm with extensions for row polymorphism.
The type checker operates in several phases:
1. First, it performs constraint generation by walking the AST
2. Then it solves constraints using unification
3. Finally, it generalizes types and checks for errors

Row polymorphism allows extensible records and effects. For example, you can have a function
that works on any record with a specific field, without knowing all the other fields...
```

**✅ Good (summary + reference):**
```markdown
## Type Inference

**Algorithm:** Hindley-Milner + row polymorphism
**Implementation:** [internal/types/](../../internal/types/)
**Details:** [Type System Guide](../docs/guides/types.md)

**Key feature:** Functions work on records with partial field knowledge:
```ailang
let getName = (r : {name: string | a}) => r.name  -- Works on any record with 'name'
```
```

**Reduction:** ~250 tokens → ~60 tokens (-76%)

#### 5. Use Checklists for Common Mistakes

**❌ Bad (paragraph format):**
```markdown
## Common Mistakes

When using imports, make sure you don't forget to import the module before using it.
Also remember that you need to use the full path for stdlib modules.
Another common mistake is forgetting that imports are declarations and must come at the top level.
```

**✅ Good (checklist format):**
```markdown
## Import Checklist

- [ ] Module imported before use: `import "std/io"` before `io.println()`
- [ ] Full stdlib path: `"std/io"` not `"io"`
- [ ] Top-level only: imports outside functions
- [ ] No circular imports
```

**Reduction:** ~80 tokens → ~35 tokens (-56%)

#### 6. Quick Reference Section

**Every prompt should have a quick reference at the top:**

```markdown
# AILANG Quick Reference (v0.3.17)

**Syntax:** `let` | `match` | `=>` | `|` | `!` | `:` | `--`
**Builtins:** See `ailang builtins list`
**Limitations:** See `docs/LIMITATIONS.md`
**Examples:** See `examples/` directory (66 files)

**Critical Rules:**
- Flags before filename: `ailang run --caps IO file.ail`
- Import stdlib: `import "std/io"` not `import "io"`
- Effect annotations required: `! {IO}` for side effects
- Pattern match exhaustive or use wildcard: `| _ => default`
```

This gives AI quick context without reading the whole prompt.

## Token Budget Analysis

### Current Prompt Stats (v0.3.16)

Run this to analyze:
```bash
wc -w prompts/v0.3.16.md   # Word count (rough token estimate)
wc -l prompts/v0.3.16.md   # Line count
```

**Target metrics:**
- Total tokens: <4000 (currently ~8000+)
- Lines: <200 (currently 500+)
- Examples: 5-10 comprehensive (currently 20+ scattered)

### Optimization Priority Areas

1. **Highest ROI:** Builtin documentation (use tables + link to `ailang builtins`)
2. **High ROI:** Module/import examples (consolidate into one example)
3. **Medium ROI:** Type system explanations (link to docs)
4. **Low ROI:** Historical notes (remove or move to changelog)

## Automated Optimization Tools

### Word Count by Section

```bash
# Count tokens per section (rough estimate)
awk '/^## / {if (section) print section, words; section=$0; words=0; next} {words+=NF} END {print section, words}' prompts/v0.3.16.md | sort -k2 -nr
```

### Find Redundant Content

```bash
# Find duplicate examples
grep -n "```ailang" prompts/v0.3.16.md | wc -l

# Find sections that could be tables
grep -A 5 "function.*takes" prompts/v0.3.16.md
```

### Link Candidates

```bash
# Find sections that should link to docs
grep -n "## " prompts/v0.3.16.md | while read line; do
  section=$(echo "$line" | cut -d: -f2)
  echo "$section -> Check if docs exist: docs/docs/guides/"
done
```

## Before/After Metrics

When optimizing a prompt, track:

```markdown
## Optimization Report

**Version:** v0.3.17 (optimized from v0.3.16)

**Before:**
- Total tokens: ~8200
- Total lines: 520
- Code examples: 24
- Tables: 2

**After:**
- Total tokens: ~3800 (-54%)
- Total lines: 180 (-65%)
- Code examples: 8 (consolidated)
- Tables: 12

**Changes:**
- Moved builtin docs to tables + referenced `ailang builtins list`
- Consolidated 24 scattered examples into 8 comprehensive ones
- Linked type system details to docs/guides/types.md
- Moved module system details to docs/guides/modules.md
- Created quick reference section at top

**Validation:**
- Ran eval baseline: v0.3.16 vs v0.3.17
- Success rate unchanged (prompt still effective)
- AI response time improved (less token processing)
```

## Progressive Disclosure Strategy

**Main prompt should have:**
1. Quick reference (1 screen)
2. Core syntax (2 screens)
3. Critical limitations (1 screen)
4. 5-8 comprehensive examples (2 screens)
5. Links to detailed docs (1 screen)

**Separate docs should have:**
- Complete builtin reference (`ailang builtins list`)
- Type system deep dive (docs/guides/types.md)
- Module system details (docs/guides/modules.md)
- Architecture notes (design_docs/)
- Historical context (CHANGELOG.md)

**Total main prompt: ~6-7 screens (~4000 tokens)**

## Validation Checklist

After optimizing a prompt:

- [ ] Run `wc -w prompts/vX.Y.Z.md` - target <4000 words
- [ ] Check all external links resolve: `grep -o "docs/.*\.md" prompts/vX.Y.Z.md | xargs -I{} test -f {} || echo "Missing: {}"`
- [ ] Verify examples work: Test in REPL
- [ ] Run eval baseline to ensure success rate maintained
- [ ] Update hash: `.claude/skills/prompt-manager/scripts/update_hash.sh vX.Y.Z`
- [ ] Document optimization metrics in prompt header

## Anti-Patterns to Avoid

### 1. Explaining Why (Move to Design Docs)

**❌ Remove:** "We chose Hindley-Milner because it provides a good balance..."
**✅ Keep:** "Algorithm: Hindley-Milner. Details: docs/guides/types.md"

### 2. Historical Context (Move to Changelog)

**❌ Remove:** "In v0.2.0 we added effects, then in v0.3.0 we improved them..."
**✅ Keep:** "Effects: `! {IO, FS}`. Since: v0.2.0. See: CHANGELOG.md"

### 3. Implementation Details (Link to Code)

**❌ Remove:** "The parser uses recursive descent with Pratt parsing for expressions..."
**✅ Keep:** "Parser: internal/parser/. Syntax: BNF in docs/grammar.md"

### 4. Verbose Examples (Show, Don't Tell)

**❌ Remove:** "First we define a function, then we call it, and then we see the result..."
**✅ Keep:**
```ailang
let double = (x) => x * 2
double(5)  -- Result: 10
```

### 5. Apologetic Limitations (Be Direct)

**❌ Remove:** "Unfortunately we don't yet support custom type classes, but we hope to..."
**✅ Keep:** "❌ Custom type classes (roadmap: v0.4.0)"

## Template for Optimized Prompt

```markdown
# AILANG vX.Y.Z Teaching Prompt

## Quick Reference
[1 screen - syntax, commands, critical rules]

## Core Language Features
[2 screens - tables + minimal examples]

## Effects & Capabilities
[1 screen - effect syntax, stdlib effects]

## Critical Limitations
[1 screen - what DOESN'T work]

## Comprehensive Examples
[2 screens - 5-8 examples covering common patterns]

## External Resources
- Complete builtin list: `ailang builtins list`
- Type system guide: [docs/guides/types.md](...)
- Module system: [docs/guides/modules.md](...)
- Known issues: [docs/LIMITATIONS.md](...)
- Examples: [examples/](../../examples/) (66 files)

## Checklist for AI Code Generation
[1 screen - common mistakes, imports, syntax gotchas]

---
**Tokens:** ~3800 | **Optimized:** [date] | **Baseline:** v0.3.16→v0.3.17
```

**Target: 6-7 screens, ~4000 tokens, maximum information density**
