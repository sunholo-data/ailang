# M-DX11: String Split Builtin

**Status:** Planned
**Target:** v0.4.7
**Priority:** P0 (High) - closes 3% Python parity gap with Gemini 3 Pro
**Estimated:** 2-3 hours
**Dependencies:** None (all dependencies exist)
**Eval Impact:** +2 benchmarks (config_file_parser, csv_to_json_converter)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | **+1** | Enables concise string parsing without custom implementations |
| Preserve Semantic Clarity | 0 | **0** | Pure function, deterministic behavior, clear semantics |
| Increase Determinism | + | **+1** | Fully deterministic (identical inputs → identical outputs) |
| Lower Token Cost | + | **+1** | AI models write `split(s, d)` in ~5 tokens vs ~50 tokens for custom impl |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Rationale:**
- **Syntactic noise ↓**: Models currently generate 20-50 line custom split implementations
- **Token cost ↓**: `split("a,b,c", ",")` vs manual recursion/accumulator pattern
- **Determinism maintained**: Pure function, no hidden state, matches Go stdlib semantics
- **Semantic clarity preserved**: Function name and behavior are self-documenting

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

AI models cannot parse delimited strings in AILANG because the stdlib lacks a `split()` function. This is the **single biggest gap** preventing Python parity with Gemini 3 Pro in eval benchmarks.

**Affected benchmarks (v0.4.6):**
- `config_file_parser` - needs to split on newlines and colons
- `csv_to_json_converter` - needs to split CSV rows

**Current error:**
```
Error: IMP010: symbol 'split' not exported by 'std/string'
```

**Impact on Python parity:**
- **Current:** AILANG 73.2% vs Python 76.9% (3.7pp gap)
- **With split():** AILANG ~78% vs Python 76.9% (**exceeds Python!**)

**Why this matters:**
- Common use case: parsing CSV, config files, logs, structured text
- AI models expect `split()` to exist (standard in Python, JavaScript, Go, etc.)
- Models are **generating correct AILANG code** but missing the function
- Only 2 benchmarks blocking parity - highest ROI fix available

## Current State

**std/string.ail (23 lines):**
- ✅ `length`, `substring`, `toUpper`, `toLower`, `trim`
- ✅ `compare`, `find`
- ✅ `stringToInt`, `stringToFloat` (added in v0.4.4)
- ❌ `split` - **MISSING**
- ❌ `join` - missing (complementary to split)
- ❌ `replace` - missing
- ❌ `startsWith`, `endsWith` - missing

**AI model expectations:**
Models consistently generate:
```ailang
import std/string (split)
let lines = split(text, "\n")
let fields = split(line, ",")
```

## Design Goals

1. **Standard semantics** - Exactly match Go `strings.Split()` (including edge cases)
2. **Pure function** - No effects, deterministic
3. **Simple API** - `split(haystack, delimiter)` returns `[string]`
4. **Empty handling** - Match Go behavior precisely, including `split("", "")` → `[]`
5. **Unicode safe** - Use Go's string handling (already Unicode-aware)

## Proposed API

### Primary Function

```ailang
-- Split string by delimiter
-- Returns [string] (list of strings) separated by delimiter
-- Empty delimiter: split into individual characters (UTF-8 codepoints)
-- Examples:
--   split("a,b,c", ",")        => ["a", "b", "c"]
--   split("hello", ",")        => ["hello"]
--   split("a,,c", ",")         => ["a", "", "c"]
--   split("", ",")             => [""]
--   split("a,b,c", "")         => ["a", ",", "b", ",", "c"]
--   split("", "")              => []  -- empty list (special case, matches Go)
export pure func split(s: string, delimiter: string) -> [string]
```

**Type signature:**
```
split : string -> string -> [string]
```

**Semantics (exactly matches Go `strings.Split`):**
- Non-empty delimiter: split at each occurrence, keep empty strings
- Empty delimiter: split into individual Unicode codepoints
- Empty string with non-empty delimiter: returns `[""]` (list with one empty string)
- **Empty string with empty delimiter: returns `[]` (empty list) - only case that returns empty list**
- Delimiter not found: returns `[original_string]` (list with one element)

## Implementation Plan

### Phase 1: Core Implementation (Required)

**1. Add Go builtin** (`internal/builtins/string.go`)

```go
func builtinStrSplit(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("split expects 2 arguments, got %d", len(args))
    }

    // Extract string arguments (note: pointers!)
    str, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("split: first argument must be string, got %T", args[0])
    }

    delim, ok := args[1].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("split: second argument must be string, got %T", args[1])
    }

    // Use Go's strings.Split for exact standard behavior
    // This handles all edge cases including split("", "") -> []
    parts := strings.Split(str.Value, delim.Value)

    // Convert []string to [string] (ListValue with Elements slice)
    elements := make([]eval.Value, len(parts))
    for i, part := range parts {
        elements[i] = &eval.StringValue{Value: part}
    }

    return &eval.ListValue{Elements: elements}, nil
}
```

**2. Register in builtin spec** (`internal/builtins/spec.go`)

```go
{
    Name:   "_str_split",
    GoFunc: builtinStrSplit,
    Type:   types.FuncType(types.StringType,
                types.FuncType(types.StringType,
                    types.ListType(types.StringType))),
    Module: "std/string",
    Pure:   true,
},
```

**3. Export from std/string.ail**

```ailang
-- std/string.ail (add at end)

-- Split string by delimiter
export pure func split(s: string, delimiter: string) -> List[string] {
  _str_split(s, delimiter)
}
```

**4. Add comprehensive tests** (`internal/builtins/string_test.go`)

```go
func TestBuiltinStrSplit(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        delimiter string
        expected  []string
    }{
        {"basic comma", "a,b,c", ",", []string{"a", "b", "c"}},
        {"no delimiter found", "hello", ",", []string{"hello"}},
        {"empty fields", "a,,c", ",", []string{"a", "", "c"}},
        {"leading delimiter", ",b,c", ",", []string{"", "b", "c"}},
        {"trailing delimiter", "a,b,", ",", []string{"a", "b", ""}},
        {"empty string with delimiter", "", ",", []string{""}},
        {"empty string empty delimiter", "", "", []string{}},  // Special case!
        {"multi-char delimiter", "a::b::c", "::", []string{"a", "b", "c"}},
        {"empty delimiter", "abc", "", []string{"a", "b", "c"}},
        {"newlines", "line1\nline2\nline3", "\n", []string{"line1", "line2", "line3"}},
        {"tabs", "col1\tcol2\tcol3", "\t", []string{"col1", "col2", "col3"}},
        {"unicode", "café☕️", "", []string{"c", "a", "f", "é", "☕️"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := effects.NewEffContext()
            args := []eval.Value{
                &eval.StringValue{Value: tt.input},
                &eval.StringValue{Value: tt.delimiter},
            }

            result, err := builtinStrSplit(ctx, args)
            if err != nil {
                t.Fatalf("split failed: %v", err)
            }

            // Result should be a ListValue
            listVal, ok := result.(*eval.ListValue)
            if !ok {
                t.Fatalf("expected *eval.ListValue, got %T", result)
            }

            // Convert Elements to []string for comparison
            var got []string
            for _, elem := range listVal.Elements {
                strVal, ok := elem.(*eval.StringValue)
                if !ok {
                    t.Fatalf("expected *eval.StringValue in list, got %T", elem)
                }
                got = append(got, strVal.Value)
            }

            if !reflect.DeepEqual(got, tt.expected) {
                t.Errorf("split(%q, %q) = %v, want %v",
                    tt.input, tt.delimiter, got, tt.expected)
            }
        })
    }
}
```

**5. Add integration test** (`examples/string_split.ail`)

```ailang
module examples/string_split

import std/string (split)
import std/io (println)

export func main() -> () ! {IO} {
  -- Test basic split
  let csv = "Alice,30,Engineer"
  let fields = split(csv, ",")

  -- Test newline split
  let text = "line1\nline2\nline3"
  let lines = split(text, "\n")

  -- Print results
  println("CSV fields:")
  printList(fields);

  println("Lines:")
  printList(lines)
}

func printList(xs: [string]) -> () ! {IO} {
  match xs {
    [] => (),
    head :: tail => {
      println(head);
      printList(tail)
    }
  }
}
```

### Phase 2: Related Functions (Optional - Future)

These would complement `split()` but are **not required** for closing the eval gap:

```ailang
-- Join list of strings with delimiter (reverse of split)
export pure func join(xs: [string], delimiter: string) -> string

-- Replace all occurrences
export pure func replace(s: string, old: string, new: string) -> string

-- Split at first occurrence only (useful for key:value parsing)
export pure func splitOnce(s: string, delimiter: string) -> {left: string, right: string}

-- Split and trim whitespace from each part
export pure func splitAndTrim(s: string, delimiter: string) -> [string]
```

**Priority:** Low - benchmarks only need basic `split()`

## Usage Examples

### Example 1: CSV Parsing with Trim (csv_to_json_converter)

```ailang
module benchmark/solution

import std/string (split, trim)
import std/list (map)
import std/fs (readFile)
import std/json (encode, Json, JObject, JString)

func parseCSV(content: string) -> [{key: string, value: Json}] {
  let lines = split(content, "\n")
  match lines {
    [] => [],
    header :: rows => {
      let columns = split(header, ",")
      -- Note: split is "dumb" - layer trim on top when needed
      let trimmedColumns = map(trim, columns)
      map(\row. parseRow(trimmedColumns, row), rows)
    }
  }
}

func parseRow(columns: [string], row: string) -> {key: string, value: Json} {
  let rawFields = split(row, ",")
  let fields = map(trim, rawFields)  -- Compose split + trim
  -- Build JSON object from columns and fields
  {key: "row", value: buildObject(columns, fields)}
}
```

**Key pattern**: Split is dumb (preserves whitespace), so compose with `map(trim, ...)` when needed.

### Example 2: Config File Parsing with Pattern Match (config_file_parser)

```ailang
module benchmark/solution

import std/string (split, trim, stringToInt)
import std/option (Option, Some, None)
import std/fs (readFile)

type Config = {app_name: string, version: string, port: int}

func parseConfig(content: string) -> Config {
  let lines = split(content, "\n")
  parseLines(lines, defaultConfig)
}

func parseLines(lines: [string], config: Config) -> Config {
  match lines {
    [] => config,
    line :: rest => {
      let parts = split(line, ":")
      -- Fixed-arity pattern match on list literal (good LM pattern!)
      match parts {
        [key, value] => {
          let k = trim(key)
          let v = trim(value)
          let updated = if k == "app_name" then {config | app_name: v}
                        else if k == "port" then
                          match stringToInt(v) {
                            Some(p) => {config | port: p},
                            None => config
                          }
                        else config
          parseLines(rest, updated)
        },
        _ => parseLines(rest, config)
      }
    }
  }
}
```

**Key pattern**: `split → fixed-arity pattern match` is the idiom you want LMs to internalize.

### Example 3: Log Parsing

```ailang
module examples/log_parser

import std/string (split)
import std/io (println)

func parseLogLine(line: string) -> {timestamp: string, level: string, message: string} {
  let parts = split(line, " ")
  match parts {
    timestamp :: level :: rest => {
      timestamp: timestamp,
      level: level,
      message: joinSpace(rest)  -- Reverse of split
    },
    _ => {timestamp: "", level: "", message: line}
  }
}

-- Helper: join list with spaces (reverse of split)
func joinSpace(words: [string]) -> string {
  match words {
    [] => "",
    [last] => last,
    first :: rest => first ++ " " ++ joinSpace(rest)
  }
}
```

## Semantic Decisions

### Empty Delimiter Behavior

**Decision:** Follow Go `strings.Split()` - split into individual Unicode codepoints

```ailang
split("hello", "") => ["h", "e", "l", "l", "o"]
```

**Rationale:**
- Matches Go's behavior (our implementation language)
- Useful for character-level processing
- Consistent with "split at every position"

**Alternative considered:** Error on empty delimiter
- Rejected: Less useful, doesn't match standard libraries

### Empty String Behavior

**Decision:** Return `[""]` (list with one empty string)

```ailang
split("", ",") => [""]
```

**Rationale:**
- Matches Python and Go
- `length(split(s, d))` is always >= 1
- Inverse property: `join(split("", d), d) == ""`

### Empty Fields

**Decision:** Preserve empty strings between consecutive delimiters

```ailang
split("a,,c", ",") => ["a", "", "c"]
split(",b,", ",")  => ["", "b", ""]
```

**Rationale:**
- Matches standard library behavior
- Important for CSV/TSV parsing (empty fields are meaningful)
- `length(split(s, d))` equals field count

## Testing Strategy

1. **Unit tests** (Go) - Test all edge cases
   - Empty string, empty delimiter
   - Single character, multi-character delimiters
   - Leading/trailing delimiters
   - Consecutive delimiters
   - No delimiter found
   - Unicode strings

2. **Integration tests** (AILANG) - Real-world usage
   - CSV parsing example
   - Config file parsing
   - Log file parsing

3. **Eval validation** - Re-run affected benchmarks
   - `config_file_parser` (should pass)
   - `csv_to_json_converter` (should pass)
   - Full baseline with Gemini 3 Pro (target: 78%)

4. **Regression tests** - Ensure no breakage
   - Existing string functions still work
   - Other benchmarks unaffected

## Success Criteria

- [ ] All unit tests pass (10+ test cases covering edge cases)
- [ ] Function registered in builtin spec (`internal/builtins/spec.go`)
- [ ] Exported from std/string module
- [ ] Integration example runs successfully (`examples/string_split.ail`)
- [ ] **config_file_parser passes** (Gemini 3 Pro eval)
- [ ] **csv_to_json_converter passes** (Gemini 3 Pro eval)
- [ ] Gemini 3 Pro AILANG success rate reaches ~78% (from 73.2%)
- [ ] Teaching prompt updated to document `split()` with examples
- [ ] All existing tests still pass (no regressions)
- [ ] Documentation updated in README.md if needed

## Success Metrics

**Primary Goal:** Close the 3.7pp Python parity gap with Gemini 3 Pro by adding string splitting capability

**Quantitative Metrics:**
- Development time: 2-3 hours (builtin + tests + docs)
- Benchmarks fixed: +2 (config_file_parser, csv_to_json_converter)
- Success rate improvement: 73.2% → ~78% (+4.8pp)
- Test coverage: 10+ unit tests, 1 integration test

**Qualitative Metrics:**
- AI models can now parse CSV, config files, logs without custom code
- Reduces token count for string parsing tasks (~45 tokens saved per usage)
- Aligns with standard library expectations (Python, Go, JavaScript)

## Teaching Prompt Updates

Add to `prompts/vX.X.X.md` in the "Standard Library" section:

```markdown
### String Manipulation (std/string)

Available functions:
- `length(s: string) -> int` - String length
- `substring(s: string, start: int, end: int) -> string` - Extract substring
- `split(s: string, delimiter: string) -> [string]` - Split string by delimiter
- `toUpper(s: string) -> string` - Convert to uppercase
- `toLower(s: string) -> string` - Convert to lowercase
- `trim(s: string) -> string` - Remove leading/trailing whitespace
- `find(haystack: string, needle: string) -> int` - Find substring (-1 if not found)
- `compare(a: string, b: string) -> int` - Compare strings (-1/0/+1)
- `stringToInt(s: string) -> Option[int]` - Parse integer
- `stringToFloat(s: string) -> Option[float]` - Parse float

**split() behavior:**
- Empty delimiter means "split into individual characters" (UTF-8 codepoints via Go's strings.Split)
- Preserves empty fields between consecutive delimiters
- `split("", "")` returns `[]` (only case that returns empty list)

**Examples:**
```ailang
import std/string (split, trim)
import std/list (map)

let csv = "Alice,30,Engineer"
let fields = split(csv, ",")  -- ["Alice", "30", "Engineer"]

let text = "line1\nline2\nline3"
let lines = split(text, "\n")  -- ["line1", "line2", "line3"]

let empty = split("", ",")  -- [""] (one empty string)
let chars = split("hi", "")  -- ["h", "i"] (empty delimiter = split into chars)

-- Compose split + trim for CSV with whitespace
let rawFields = split("a, b , c", ",")  -- ["a", " b ", " c"]
let cleanFields = map(trim, rawFields)  -- ["a", "b", "c"]
```

**Common patterns:**
- CSV parsing: `split(line, ",")` then `map(trim, ...)` if needed
- Line splitting: `split(text, "\n")`
- Tab-separated: `split(line, "\t")`
- Key-value pairs: `match split(line, ":") { [k,v] => ... }`

**Common mistakes:**
```ailang
-- ❌ WRONG (Python/JavaScript method style):
let fields = csv.split(",")

-- ✅ CORRECT in AILANG:
let fields = split(csv, ",")
```
```

## Migration Notes

**For AI models:**
- **Before:** `split(text, ",")` → undefined variable error
- **After:** `split(text, ",")` → works as expected
- **Import:** `import std/string (split)`

**For existing code:**
- No breaking changes (new function)
- Any custom split implementations can be removed

## Non-Goals

**Not in this feature:**
- `join()` - Reverse of split (list → string) - Deferred to future work
- `replace()` - String replacement - Separate feature
- `splitN()` - Split into at most N parts - Can add if benchmarks need it
- `splitLines()` - Smart line splitting (\n, \r\n, \r) - Not needed yet
- `splitWhitespace()` - Split on any whitespace - Not needed yet
- `rsplit()` - Split from right to left - Python-specific, not standard

**Why deferred:**
- Only `split()` is needed to close the eval gap
- Other functions can be added incrementally as benchmarks reveal need
- Keep this PR focused and reviewable

## Performance Considerations

**Time complexity:** O(n) where n is string length
**Space complexity:** O(n) for result list
**Allocations:** One allocation per substring + list structure

**Optimization notes:**
- Uses Go's `strings.Split()` (heavily optimized)
- No unnecessary string copies
- List construction is backwards (efficient for functional lists)

## Related Issues

**Complement functions (future):**
- `join()` - Reverse of split (list -> string)
- `replace()` - String replacement
- `splitOnce()` - Split at first occurrence only

**Blocked by:** None (all dependencies exist)
**Blocks:** Python parity gap closure

## Timeline

- **Design:** 30 minutes (done)
- **Implementation:** 45-60 minutes (Go builtin + registration)
- **Testing:** 30-45 minutes (unit + integration)
- **Documentation:** 15 minutes (prompt update)
- **Eval validation:** 30-60 minutes (re-run benchmarks)
- **Total:** 2.5-3.5 hours (well within estimate)

## Dependencies

- **Go stdlib:** `strings.Split()` (already available)
- **List type:** Already implemented
- **Builtin registry:** Already implemented (M-DX1)
- **std/string module:** Already exists

## Risks

**Low risk:**
- Pure function, no side effects
- Standard semantics (Go `strings.Split`)
- No breaking changes
- High test coverage

**Potential issues:**
1. **Unicode edge cases** - Mitigated by using Go's Unicode-aware strings
2. **Memory overhead for large strings** - Acceptable (standard split behavior)
3. **Empty delimiter semantics** - Documented and tested

## Future Enhancements (Out of Scope)

- **splitN:** Split into at most N parts
- **splitLines:** Smart line splitting (handles \n, \r\n, \r)
- **splitWhitespace:** Split on any whitespace (spaces, tabs, newlines)
- **rsplit:** Split from right to left
- **splitAndFilter:** Split and remove empty strings

## References

- **Go strings.Split:** https://pkg.go.dev/strings#Split
- **Python str.split:** https://docs.python.org/3/library/stdtypes.html#str.split
- **Gemini 3 Analysis:** `/tmp/gemini3_analysis.md`
- **Eval Results:** `eval_results/baselines/v0.4.6/`
- **M-DX10 (String Parsing):** `design_docs/planned/v0_4_3/m-dx10-string-parsing-builtins.md`
- **M-DX1 (Builtin System):** `design_docs/implemented/v0_3_10/m-dx1-builtin-migration.md`

---

**Document created:** 2024-11-24
**Last updated:** 2024-11-24
