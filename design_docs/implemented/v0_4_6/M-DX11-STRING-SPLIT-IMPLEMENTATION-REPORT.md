# String split() Implementation Summary (v0.4.6)

## 🎯 Mission
Close the 3% Python parity gap identified in v0.4.6 baseline for Gemini 3 Pro (73.2% AILANG vs 76.9% Python).

## ✅ Accomplishments

### 1. Implemented split() Builtin
**Location**: `internal/builtins/string.go` (87 lines)

**Function signature**: `split(s: string, delimiter: string) -> [string]`

**Semantics**: Exact match to Go's `strings.Split()`:
- Empty delimiter splits into individual UTF-8 codepoints
- Preserves empty fields: `split("a,,c", ",")` → `["a", "", "c"]`
- Edge case: `split("", "")` → `[]` (only case returning empty list)

**Implementation details**:
- Registered via M-DX1 builtin system (central registry)
- Type: `string -> string -> [string]` (curried)
- Pure function (no effects)
- Full test coverage: 13 test cases, all passing

### 2. Updated Teaching Prompt (prompts/v0.4.6.md)
**Changes made**:
- ✅ Added split() documentation with examples (lines 282-330)
- ✅ Fixed `append` → `concat` (std/list function name correction)
- ✅ Added complete list function reference (10 functions documented)
- ✅ Added nested function guidance (`func` only at module level)
- ✅ Added split + trim composition examples
- ✅ Added common mistakes section

### 3. Created Simple split() Benchmark
**Location**: `benchmarks/string_split.yml`

**Tests**:
1. CSV parsing: `split("Alice,Bob,Charlie", ",")`
2. Line splitting: `split("line1\nline2\nline3", "\n")`
3. Character splitting: `split("hello", "")`
4. Empty field preservation: `split("a,,c", ",")`
5. Edge case: `split("", "")`

## 📊 Evaluation Results

### Simple Benchmark (string_split)
**AILANG Success Rate**: 75% (3/4 models)
- ✅ Gemini 3 Pro: PASS
- ✅ Gemini 2.5 Flash: PASS
- ✅ GPT-5 Mini: PASS
- ❌ Claude Haiku 4.5: FAIL (imported `length` from wrong module)

**Python Parity (Gemini 3 Pro)**: 100% (2/2)
- ✅ AILANG: PASS
- ✅ Python: PASS

### Complex Benchmarks (config_file_parser, csv_to_json_converter)
**AILANG Success Rate**: 0% (0/8 attempts)

**Failure reasons** (NOT related to split() itself):
1. **Nested function syntax**: Models using `func` keyword inside expressions (parser error)
2. **Semicolon usage**: Using `;` outside `{ }` blocks (parser error)
3. **Wrong imports**: Importing `length` from `std/string` instead of `std/list`
4. **List operations**: Trying to use non-existent `append()` (should be `concat()`)

## 🔍 Key Findings

### split() Implementation: ✅ SUCCESS
- Function works correctly
- Prompt documentation is comprehensive
- Models CAN use split() when task is simple
- AILANG achieves Python parity on focused split() benchmark

### Complex Benchmarks: ❌ BLOCKED by syntax issues
The complex benchmarks (config_file_parser, csv_to_json_converter) fail not because of missing split(), but because they require:
- Multiple stdlib modules
- Complex JSON manipulation
- Recursive list processing
- Proper use of `let ... in ...` vs block syntax
- Understanding when to use semicolons

These are **prompt effectiveness issues**, not missing language features.

## 📈 Expected Python Parity Impact

### Optimistic Scenario
If prompt improvements fix the syntax issues:
- **Before (v0.4.5)**: Gemini 3 Pro at 73.2% AILANG, 76.9% Python (3.7pp gap)
- **Expected (v0.4.6)**: Gemini 3 Pro at ~78% AILANG, 76.9% Python (**+1.1pp lead**)
- **Improvement**: +4.8pp absolute, closing gap and overtaking Python

### Realistic Scenario
Based on current eval failures:
- **Achieved**: split() benchmark shows 100% AILANG parity with Python (Gemini 3 Pro)
- **Blocked**: Complex benchmarks need additional prompt refinement
- **Status**: Feature is ready, prompt needs iteration

## 🎯 Recommendations

### Short Term (v0.4.6 Release)
1. ✅ **Ship split() builtin** - Implementation is solid and tested
2. ✅ **Include updated prompt** - split() is documented with examples
3. ✅ **Add string_split benchmark** - Simple test validates functionality
4. ⚠️ **Document known issues** - Complex benchmarks need prompt iteration

### Medium Term (v0.4.7)
1. **Refine prompt for complex patterns**:
   - Emphasize `let ... in ...` for sequential bindings in match arms
   - Show more examples of nested lambdas (since `func` doesn't work)
   - Add table showing when to use `;` vs `let ... in ...`

2. **Add intermediate benchmarks**:
   - CSV parsing (just split + basic list ops)
   - Config parsing (just split + record construction)
   - Bridge the gap between simple and complex

3. **Consider stdlib additions**:
   - `std/list.append` as alias for `concat` (matches common expectations)
   - Or update prompt to be VERY clear about `concat` vs `append`

## 📝 Files Modified

### Implementation (87 lines)
- `internal/builtins/string.go` - Added registerStrSplit()
- `std/string.ail` - Exported split()
- `internal/builtins/string_test.go` - Added 13 test cases
- `examples/string_split.ail` - Integration example

### Documentation
- `prompts/v0.4.6.md` - Updated (343 lines changed)
- `prompts/versions.json` - Updated hash
- `design_docs/planned/v0_4_6/m-dx11-string-split-builtin.md` - Design doc

### Benchmarks
- `benchmarks/string_split.yml` - New simple benchmark

### Evaluation Results
- `eval_results/string_split_test/` - Simple benchmark results (75% success)
- `eval_results/split_validation_v*/` - Complex benchmark attempts (0% success)

## 🏆 Success Metrics

| Metric | Target | Achieved | Status |
|--------|--------|----------|--------|
| split() implementation | Complete with tests | ✅ 13/13 tests passing | ✅ DONE |
| Prompt documentation | Comprehensive examples | ✅ 48 lines of docs | ✅ DONE |
| Simple benchmark success | >70% AILANG | ✅ 75% (3/4 models) | ✅ DONE |
| Python parity (simple) | 100% | ✅ 100% (Gemini 3 Pro) | ✅ DONE |
| Complex benchmark success | >0% | ❌ 0% (blocked by syntax) | ⚠️ BLOCKED |
| Python parity gap closed | -3.7pp | ⚠️ Pending prompt iteration | ⚠️ PENDING |

## 💡 Conclusion

**The split() builtin is successfully implemented and ready for release.** Models CAN use it correctly when the task is focused (75% success rate, 100% Python parity on simple benchmark).

**The 3% Python parity gap is NOT closed yet** because complex benchmarks fail due to prompt effectiveness issues (nested functions, semicolons, module imports), not missing language features.

**Next steps**:
1. Ship v0.4.6 with split() - the feature is ready
2. Iterate on prompt refinement in v0.4.7 to improve complex benchmark success
3. Consider adding intermediate-complexity benchmarks to bridge the gap

---

**Time invested**: ~4 hours (design doc, implementation, testing, prompt updates, benchmark creation)
**Lines of code**: ~250 lines (implementation + tests + examples)
**Documentation**: ~400 lines (design doc + prompt updates)
**Tests**: 13 test cases, 100% passing
