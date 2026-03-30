# M-DX-UTF8: UTF-8 String Operation Correctness + replace() Builtin

**Status**: Planned
**Target**: v0.9.5
**Priority**: P2 (DX — runtime correctness bug affecting multi-byte strings)
**Estimated**: 0.5 days
**Dependencies**: None
**Milestone ID**: M-DX-UTF8
**Created**: 2026-03-30
**Source**: DocParse agent message `5437739b` (find/substring byte offset mismatch)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | String ops must produce identical results regardless of byte encoding |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Correct index semantics enable local reasoning about string manipulation |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | AI agents processing UTF-8 text (emails, web pages) hit this bug silently |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +2 | `substring(s, find(s, x), ...)` must compose correctly — currently broken |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | +1 | Text from HTTP, files, emails contains UTF-8 multi-byte chars |

**Net Score: +8** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): `find()` returning byte offsets means identical AILANG code produces different logical results depending on string encoding
- [x] A10 (Composability): `substring(s, find(s, needle), length(s))` is the canonical compose pattern — currently broken for multi-byte strings

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — systemic. The codebase has **two builtin registration systems** and the active one (spec registry in `internal/builtins/`) has byte/rune mismatches in multiple string functions. The deprecated system (`internal/eval/`) had correct implementations that were never ported.

Additionally, the Go codegen layer (`registry_codegen.go`) has its own set of byte/rune bugs that are independent of the runtime.

**Affected locations (runtime):**

| Function | File | Bug |
|----------|------|-----|
| `_str_find` | `internal/builtins/string.go:251` | `strings.Index()` returns byte offset, not rune index |

**Affected locations (codegen):**

| Function | File | Bug |
|----------|------|-----|
| `_str_len` | `internal/builtins/registry_codegen.go:47` | `len(string)` returns bytes, not rune count |
| `_str_find` | `internal/builtins/registry_codegen.go:56` | `strings.Index()` returns byte offset |
| `_str_slice` | `internal/builtins/registry_codegen.go:67-73` | Uses `len(str)` for bounds + byte-based slicing |

**Correct implementations (runtime, no fix needed):**

| Function | File | Why correct |
|----------|------|-------------|
| `_str_len` | `internal/builtins/string.go:93` | Uses `utf8.RuneCountInString()` |
| `_str_slice` | `internal/builtins/string.go:309` | Converts to `[]rune` before slicing |

---

## Problem

### Bug 1: `find()` returns byte offset at runtime

```ailang
let s = "cafe\u0301 world"   // "café world" — é is 2 bytes
let pos = find(s, "world")   // Returns 7 (byte offset) instead of 6 (char index)
substring(s, pos, length(s)) // Wrong result — skips a character
```

DocParse discovered this parsing email headers with UTF-8 characters (ç, ü). The `find()`/`substring()` composition loses characters proportional to multi-byte chars before the match.

### Bug 2: Codegen emits byte-based operations

When AILANG compiles to Go, `length()`, `find()`, and `substring()` all operate on bytes instead of characters. This means compiled AILANG programs have different behavior from interpreted ones on multi-byte strings.

### Feature gap: No `replace()` builtin

String replacement is a fundamental operation needed by docparse and any text-processing AILANG program. Currently requires manual `find()`/`substring()`/`concat()` composition, which is verbose and error-prone (especially given Bug 1).

---

## Solution

### Fix 1: Runtime `_str_find` — add rune conversion

In `internal/builtins/string.go`, `strFindImpl`:

```go
func strFindImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    haystack, err := SafeAsString(args[0])
    if err != nil { return nil, fmt.Errorf("_str_find: arg 0 - %w", err) }
    needle, err := SafeAsString(args[1])
    if err != nil { return nil, fmt.Errorf("_str_find: arg 1 - %w", err) }

    byteIdx := strings.Index(haystack, needle)
    if byteIdx == -1 {
        return &eval.IntValue{Value: -1}, nil
    }
    // Convert byte offset to rune index
    runeIdx := utf8.RuneCountInString(haystack[:byteIdx])
    return &eval.IntValue{Value: runeIdx}, nil
}
```

### Fix 2: Codegen specs — use rune-aware operations

```go
// _str_len: use utf8.RuneCountInString instead of len()
setSpec("_str_len", &GoCodegenSpec{
    Inline:     `int64(utf8.RuneCountInString({{arg0}}.(string)))`,
    Imports:    []string{"unicode/utf8"},
    StdlibName: "length",
})

// _str_find: convert byte index to rune index
// Needs a helper function for the conversion
setSpec("_str_find", &GoCodegenSpec{
    Helper: &GoHelperSpec{
        FuncName:  "FindRune",
        Signature: "func FindRune(s interface{}, sub interface{}) interface{}",
        Body: `str := s.(string)
    byteIdx := strings.Index(str, sub.(string))
    if byteIdx == -1 { return int64(-1) }
    return int64(utf8.RuneCountInString(str[:byteIdx]))`,
    },
    Imports:    []string{"strings", "unicode/utf8"},
    StdlibName: "find",
})

// _str_slice: convert to runes before slicing
setSpec("_str_slice", &GoCodegenSpec{
    Helper: &GoHelperSpec{
        FuncName:  "Substring",
        Signature: "func Substring(s interface{}, start interface{}, end interface{}) interface{}",
        Body: `runes := []rune(s.(string))
    length := len(runes)
    st := int(toInt64(start))
    en := int(toInt64(end))
    if st < 0 { st = 0 }
    if en > length { en = length }
    if st > en { return "" }
    return string(runes[st:en])`,
    },
    StdlibName: "substring",
})
```

### Feature: `replace()` builtin

**Name**: `_str_replace` / stdlib `replace`
**Signature**: `(String, String, String) -> String`
**Semantics**: Replace all occurrences of a substring with a replacement string.

```ailang
replace("hello world", "world", "AILANG")  // "hello AILANG"
replace("aaa", "a", "bb")                  // "bbbbbb"
replace("no match", "xyz", "abc")          // "no match"
```

**Design decisions:**
- **Replace all** (not first) — matches Python `str.replace()` behavior and is the common case for text processing
- **No regex** — keep it simple; regex can be a separate future builtin
- **Pure function** — no effects needed
- **UTF-8 safe** — `strings.ReplaceAll()` in Go is already UTF-8 correct

**Runtime implementation** (`internal/builtins/string.go`):

```go
func strReplaceImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, err := SafeAsString(args[0])
    if err != nil { return nil, fmt.Errorf("_str_replace: arg 0 - %w", err) }
    old, err := SafeAsString(args[1])
    if err != nil { return nil, fmt.Errorf("_str_replace: arg 1 - %w", err) }
    newStr, err := SafeAsString(args[2])
    if err != nil { return nil, fmt.Errorf("_str_replace: arg 2 - %w", err) }
    return &eval.StringValue{Value: strings.ReplaceAll(s, old, newStr)}, nil
}
```

**Codegen spec** (`registry_codegen.go`):

```go
setSpec("_str_replace", &GoCodegenSpec{
    Inline:     `strings.ReplaceAll({{arg0}}.(string), {{arg1}}.(string), {{arg2}}.(string))`,
    Imports:    []string{"strings"},
    StdlibName: "replace",
})
```

**Stdlib binding** (`stdlib/std/string.ail`):

```ailang
export func replace(s: String, old: String, new: String) -> String =
  _str_replace(s, old, new)
```

---

## Testing Strategy

### UTF-8 fix tests

```go
// Runtime: find() returns rune index
{"find café world", `_str_find("café world", "world")`, 5},  // not 6
{"find with emoji", `_str_find("hello 🎉 world", "world")`, 8},  // not 10

// Runtime: find+substring compose correctly
{"find+substring UTF-8", `_str_slice("café world", _str_find("café world", "world"), _str_len("café world"))`, "world"},

// Codegen: same tests compiled to Go
// (run via make test-codegen or similar)
```

### replace() tests

```go
{"replace basic", `_str_replace("hello world", "world", "AILANG")`, "hello AILANG"},
{"replace all", `_str_replace("aaa", "a", "bb")`, "bbbbbb"},
{"replace no match", `_str_replace("hello", "xyz", "abc")`, "hello"},
{"replace empty needle", `_str_replace("hello", "", "x")`, "xhxexlxlxox"},
{"replace UTF-8", `_str_replace("café", "é", "e")`, "cafe"},
{"replace to empty", `_str_replace("hello world", " world", "")`, "hello"},
```

---

## Files to Change

| File | Change |
|------|--------|
| `internal/builtins/string.go` | Fix `strFindImpl`, add `registerStringReplace()` call to init |
| `internal/builtins/string_ops.go` | Add `registerStringReplace` + `strReplaceImpl` |
| `internal/builtins/registry_codegen.go` | Fix `_str_len`, `_str_find`, `_str_slice` codegen; add `_str_replace` |
| `internal/builtins/string_test.go` | Add UTF-8 tests for find, tests for replace |
| `internal/eval/builtins_string.go` | Add `_str_replace` to legacy registry (for compile-time validation) |
| `std/string.ail` | Add `replace()` stdlib binding |
| `examples/string_replace.ail` | Example file demonstrating replace |
| `changelogs/v0.9-current.md` | Document fix and new builtin |

---

## Risks

- **Performance**: Converting to `[]rune` in `_str_slice` and counting runes in `_str_find` adds O(n) overhead. For the vast majority of use cases this is negligible. If performance-critical string processing is needed, a byte-oriented API can be added later.
- **Breaking change**: Any code that accidentally relied on byte offsets from `find()` would break. This is extremely unlikely to be intentional — the bug report proves the current behavior is wrong.
