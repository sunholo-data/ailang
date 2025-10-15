## Example Status

### Working Examples ✅
- `adt_option.ail`
- `adt_simple.ail`
- `ai_call.ail` ← ✨ NEW (v0.3.9) - OpenAI API integration
- `arithmetic.ail`
- `block_recursion.ail` ← ✨ NEW (v0.3.0-alpha2)
- `claude_haiku_call.ail` ← ✨ NEW (v0.3.9) - Anthropic API integration
- `demos/adt_pipeline.ail` ← ✅ FIXED (M-R5 Day 1)
- `demos/hello_io.ail`
- `effects_basic.ail`
- `effects_pure.ail`
- `guards_basic.ail`
- `hello.ail`
- `micro_block_if.ail` ← ✨ NEW (v0.3.0-alpha2)
- `micro_block_seq.ail` ← ✨ NEW (v0.3.0-alpha2)
- `micro_io_echo.ail` ← ✅ FIXED (M-R5 Day 1)
- `micro_option_map.ail` ← ✅ FIXED (M-R5 Day 1)
- `micro_record_person.ail` ← ✨ NEW (v0.3.0-alpha3 M-R5 Day 3)
- `recursion_error.ail`
- `recursion_factorial.ail`
- `recursion_fibonacci.ail`
- `recursion_mutual.ail`
- `recursion_quicksort.ail`
- `showcase/01_type_inference.ail`
- `showcase/02_lambdas.ail`
- `showcase/03_type_classes.ail`
- `showcase/04_closures.ail`
- `simple.ail`
- `test_effect_annotation.ail`
- `test_effect_capability.ail`
- `test_effect_fs.ail`
- `test_effect_io.ail`
- `test_exhaustive_bool_complete.ail`
- `test_exhaustive_bool_incomplete.ail`
- `test_exhaustive_wildcard.ail`
- `test_guard_bool.ail`
- `test_guard_debug.ail`
- `test_guard_false.ail`
- `test_import_ctor.ail` ← ✅ FIXED (M-R5 Day 1)
- `test_import_func.ail` ← ✅ FIXED (M-R5 Day 1)
- `test_invocation.ail`
- `test_io_builtins.ail`
- `test_module_minimal.ail`
- `test_no_import.ail`
- `test_record_subsumption.ail` ← ✨ NEW (v0.3.0-alpha3 M-R5 Day 3)
- `test_single_guard.ail`
- `test_use_constructor.ail` ← ✅ FIXED (M-R5 Day 1)
- `test_with_import.ail`
- `type_classes_working_reference.ail`
- `v3_3/imports.ail` ← ✅ FIXED (M-R5 Day 1)
- `v3_3/imports_basic.ail` ← ✅ FIXED (M-R5 Day 1)

### Failing Examples ❌
- `demos/effects_pure.ail`
- `experimental/ai_agent_integration.ail`
- `experimental/concurrent_pipeline.ail`
- `experimental/factorial.ail`
- `experimental/quicksort.ail`
- `experimental/web_api.ail`
- `lambda_expressions.ail`
- `list_patterns.ail`
- `patterns.ail`
- `records.ail`
- `showcase/03_lists.ail`
- `test_effect_io_simple.ail`
- `typeclasses.ail`
- `v3_3/math/gcd.ail`

### Skipped Examples ⏭️
- `block_demo.ail`
- `option_demo.ail`
- `stdlib_demo.ail`
- `stdlib_demo_simple.ail`

**Summary:** 50 passed, 14 failed, 4 skipped (Total: 68)

**Recent improvements:**
- ✅ **v0.3.9 (Oct 2025)**: 2 new AI API integration examples!
  - `ai_call.ail`: OpenAI GPT-4o-mini integration with JSON encoding
  - `claude_haiku_call.ail`: Anthropic Claude Haiku integration (verified with real API)
- ✅ **M-R5 (v0.3.0-alpha3)**: 11 examples fixed/added via records & row polymorphism!
  - Day 1: 9 examples fixed (demos/adt_pipeline, micro_io_echo, micro_option_map, test_import_ctor, test_import_func, test_use_constructor, v3_3/imports, v3_3/imports_basic)
  - Day 3: 2 new examples (micro_record_person, test_record_subsumption)
- ✅ **M-R8 (v0.3.0-alpha2)**: `micro_block_*.ail`, `block_recursion.ail` (3 files) - Block expressions with recursion
- ✅ **M-R4 (v0.3.0-alpha1)**: `recursion_*.ail` (5 files) - Recursion support with RefCell indirection
