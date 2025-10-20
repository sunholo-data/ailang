# M-DX1 Future Polish (v0.3.15+)

**Status**: Optional DX Enhancements
**Prerequisites**: M-DX1 Core Complete (October 2025) ✅
**Estimated Total Effort**: ~2.5 hours

## Context

🎉 **M-DX1 Core COMPLETE (90% done!)** - October 2025

**Accomplished:**
- ✅ All 52 builtins migrated to new spec-based registry
- ✅ Feature flag removed - new registry is default
- ✅ Development time reduced from 7.5h → 2.5h (-67%)
- ✅ Files to edit reduced from 4 → 1 (-75%)
- ✅ **All 52 builtins fully documented (100%)**
- ✅ File organization optimized (7 AI-friendly modules)
- ✅ Migration safety validator deployed
- ✅ Enhanced metadata system (11 fields)
- ✅ All tests passing (2,847 tests)

**What remains**: Optional developer-facing polish features that improve the daily workflow but are not blocking.

---

## M-DX1.6: REPL `:type` Command (~0.5 hours)

**Priority**: Medium (Nice to have)
**Complexity**: Low (infrastructure already exists)

### Goal
Add `:type <expr>` command to REPL for quick type inspection during development.

### Motivation
**Current pain point**: When developing builtins or debugging type errors, developers must:
1. Write a test file
2. Run the compiler
3. Read the error message
4. Repeat

**With `:type`**: Instant feedback in REPL.

### Tasks

**30 minutes: Implementation**
- Parse `:type <expr>` in REPL command loop
- Run type inference on expression
- Pretty-print type signature using existing pretty-printer
- Handle polymorphic types (display type variables)
- Update REPL help text

**Note**: Type inference infrastructure already exists, just needs REPL command wiring!

### Example Usage

```ailang
λ> :type _str_len
string -> int

λ> :type map
(a -> b) -> [a] -> [b]

λ> :type _net_httpRequest
string -> string -> [{name: string, value: string}] -> string
  -> <Net> Result[HttpResponse, NetError]

λ> :type let f = \x -> x + 1 in f
int -> int

λ> :type 42
int

λ> :type "hello"
string
```

### Success Criteria
- [ ] `:type` works for builtins
- [ ] `:type` works for user-defined functions
- [ ] `:type` displays polymorphic types correctly
- [ ] Effect rows are formatted clearly
- [ ] 5+ test cases covering common scenarios

### Files to Modify
- `internal/repl/repl.go` - Add command parsing (~150 LOC)
- `internal/types/pretty.go` - Type formatting (may need new file, ~100 LOC)
- `internal/repl/repl_test.go` - Test cases (~50 LOC)

---

## M-DX1.7: Enhanced CLI - List with Descriptions (~2 hours)

**Priority**: High (Most useful polish feature)
**Complexity**: Low (metadata already exists)

### Goal
Add `--verbose` and `search` modes to `ailang builtins list` to leverage the metadata we just added!

### Motivation
**Current output is minimal:**
```
$ ailang builtins list
_str_len [pure]
_str_compare [pure]
...
```

**With `--verbose`:**
```
$ ailang builtins list --verbose

_str_len [pure] - std/string
  Get the length of a string in Unicode characters
  Type: string -> int
  Since: v0.1.0

_str_compare [pure] - std/string
  Compare two strings lexicographically
  Type: string -> string -> int
  Since: v0.1.0
```

**With `search`:**
```
$ ailang builtins search "http"
_net_httpRequest [net] - std/net
  Make an HTTP request with custom headers
  Tags: http, network, request
```

### Tasks

**Hour 1: Implement `--verbose` mode**
- Add flag to `cmd/ailang/main.go`
- Format output with metadata (description, type, since, tags)
- Pretty-print type signatures
- Handle long descriptions (wrap text)

**Hour 2: Implement `search` command**
- `ailang builtins search <query>` command
- Search by: name, description, tags, category
- Fuzzy matching for typos
- Highlight matched terms in output

### Success Criteria
- [ ] `ailang builtins list --verbose` shows descriptions
- [ ] `ailang builtins search <query>` finds builtins by name/tags/description
- [ ] Output is nicely formatted and readable
- [ ] Fuzzy matching helps with typos

### Files to Modify
- `cmd/ailang/main.go` - Add `--verbose` flag and `search` command (~100 LOC)
- `internal/builtins/registry.go` - Add search/filter methods (~50 LOC)

---

## M-DX1.8: Error Diagnostics (~0.5 hours)

**Priority**: Low (Nice to have)
**Complexity**: Low

### Goal
Better error messages when builtins are not found or misused.

### Examples

**Missing builtin:**
```
Error: Undefined variable '_my_op'

Hint: No builtin named '_my_op' found.
Did you mean: _io_print, _io_println?
To see all builtins: ailang builtins list
```

**Arity mismatch:**
```
Error: Function expects 2 arguments, got 3

Hint: _str_find expects 2 arguments: (haystack, needle)
Usage: _str_find("hello world", "world")
```

### Tasks
- Detect common error patterns
- Suggest similar builtin names (typo detection)
- Show correct usage examples from metadata

### Files to Modify
- `internal/errors/hints.go` - Pattern matching (~100 LOC, new file)

---

## Timeline

**Optional polish - can be done anytime:**

**Priority 1** (~2 hours):
- M-DX1.7: Enhanced CLI with `--verbose` and `search` (2h)

**Priority 2** (~0.5 hours):
- M-DX1.6: REPL `:type` command (0.5h)

**Priority 3** (~0.5 hours):
- M-DX1.8: Error diagnostics (0.5h)

**Total: ~2.5 hours** (all optional DX polish)

---

## Success Metrics - ACHIEVED! ✅

**Development workflow:**
- Time to add builtin: 2.5h ✅ **ACHIEVED**
- Files to edit: 1 ✅ **ACHIEVED**
- Type construction: 10 LOC ✅ **ACHIEVED**
- CLI validation available: Yes ✅ **ACHIEVED** (`ailang doctor builtins`)
- All 52 builtins documented: Yes ✅ **ACHIEVED** (100%)
- REPL type checking: (M-DX1.6 - optional)
- Enhanced CLI search: (M-DX1.7 - optional)

**Code quality:**
- All builtins in new registry: 52/52 ✅ **ACHIEVED**
- Files organized & AI-friendly: Yes ✅ **ACHIEVED** (all < 600 lines)
- 100% test coverage: Yes ✅ **ACHIEVED** (2,847 tests passing)
- Migration safety: Yes ✅ **ACHIEVED** (validator prevents disasters)

**Developer experience:**
- Implementation time: -67% reduction ✅ **ACHIEVED**
- Comprehensive metadata: Yes ✅ **ACHIEVED** (11 fields)
- Examples for all builtins: Yes ✅ **ACHIEVED** (100+ examples)
- Documentation coverage: 100% ✅ **ACHIEVED**

---

## References

- **Session Summary**: [M-DX1-FINAL-SUMMARY.md](../../M-DX1-FINAL-SUMMARY.md)
- **Completion Announcement**: [M-DX1-COMPLETION-ANNOUNCEMENT.md](../../M-DX1-COMPLETION-ANNOUNCEMENT.md)
- **Main docs**: [CLAUDE.md](../../CLAUDE.md) - See "Adding Builtin Functions" section
- **Motivation**: `design_docs/planned/easier-ailang-dev.md`
- **Infrastructure**: M-DX1.1-1.5 (v0.3.9-v0.3.10)
- **Documentation**: M-DX1.11-1.14 (October 2025)

---

## Summary

🎉 **M-DX1 Core is 90% COMPLETE!**

All critical work is done:
- ✅ All 52 builtins migrated & documented
- ✅ Implementation time reduced by 67%
- ✅ Files organized for AI (<600 lines each)
- ✅ Migration safety validator deployed
- ✅ 2,847 tests passing

The remaining 10% (M-DX1.6-1.8) are **optional DX polish features** that can be added anytime but are not blocking. The core mission - making builtins "a joy for developers" - is **accomplished**! 🚀
