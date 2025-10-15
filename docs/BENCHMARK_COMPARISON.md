# AILANG vs Python: AI Code Generation Benchmark Comparison

> Generated from automated evaluation across multiple AI models

## Summary

### Key Performance Indicator

**AILANG vs Python Delta: **-39%** ⚠️ Python advantage**

### Detailed Metrics

| Metric | AILANG | Python | Delta |
|--------|--------|--------|-------|
| Success Rate | 42% (16/38) | 81% (30/37) | -39% |
| Avg Cost per Benchmark | $0.0012 | $0.0003 | 0.00 |
| Models Tested | 2 | 2 | - |
| Benchmarks | 20 | 20 | - |

## Detailed Comparison

| Benchmark | Description | AILANG Success | Python Success | AILANG Tokens (in/out) | Python Tokens (in/out) | AILANG Speed (ms) | Python Speed (ms) | AILANG Cost | Python Cost | Status |
|-----------|-------------|----------------|----------------|------------------------|------------------------|-------------------|-------------------|-------------|-------------|--------|
| adt_option | Option/Maybe monad operations (algebraic data types) | 1/2 | 2/2 | 3985/183 | 165/255 | 12ms | 44ms | $0.0015 | $0.0006 | ✅ Both passing |
| cli_args | Read file from CLI argument, process, write result (IO + FS) | 0/2 | 0/2 | 83/205 | 83/121 | 13ms | 38ms | $0.0005 | $0.0003 | ❌ Both failing |
| error_handling | Result/Either type for error propagation | 1/2 | 2/2 | 4103/397 | 282/570 | 15ms | 44ms | $0.0021 | $0.0014 | ✅ Both passing |
| fizzbuzz | Classic FizzBuzz problem (control flow) | 2/2 | 2/2 | 3957/144 | 136/74 | 18ms | 41ms | $0.0014 | $0.0002 | ✅ Both passing |
| float_eq | N/A | 0/2 | 1/2 | 45/20 | 45/19 | 13ms | 39ms | $0.0001 | $0.0001 | ⚠️ Python only (AILANG: compile_error) |
| higher_order_functions | Compose, map, filter, currying, partial application | 0/2 | 2/2 | 4053/209 | 232/112 | 10ms | 30ms | $0.0016 | $0.0003 | ⚠️ Python only (AILANG: compile_error) |
| json_parse | Parse JSON array, filter, and output (data parsing) | 0/2 | 2/2 | 98/78 | 98/74 | 13ms | 39ms | $0.0002 | $0.0002 | ⚠️ Python only (AILANG: compile_error) |
| list_comprehension | Map/filter/fold operations on lists | 0/2 | 1/2 | 4057/461 | 237/102 | 13ms | 40ms | $0.0022 | $0.0003 | ⚠️ Python only (AILANG: compile_error) |
| list_operations | List construction, pattern matching, and recursion | 0/2 | 2/2 | 4003/195 | 182/139 | 11ms | 27ms | $0.0016 | $0.0004 | ⚠️ Python only (AILANG: compile_error, logic_error) |
| nested_records | Nested record construction and field access | 2/2 | 2/2 | 4020/110 | 200/145 | 139ms | 29ms | $0.0014 | $0.0004 | ✅ Both passing |
| numeric_modulo | N/A | 0/2 | 2/2 | 39/11 | 39/12 | 13ms | 22ms | $0 | $0 | ⚠️ Python only (AILANG: compile_error) |
| pattern_matching_complex | Nested patterns, guards, exhaustiveness checking | 1/2 | 1/1 | 4106/398 | 277/335 | 16ms | 47ms | $0.002 | $0.0007 | ✅ Both passing |
| pipeline | Read stdin, transform data, write stdout (IO + lists) | 0/2 | 0/2 | 99/64 | 99/70 | 14ms | 36ms | $0.0002 | $0.0002 | ❌ Both failing |
| record_update | Record update syntax {r | field: value} | 2/2 | 1/2 | 3996/202 | 176/152 | 17ms | 37ms | $0.0016 | $0.0004 | ✅ Both passing |
| records_person | Record types with field access and updates | 2/2 | 2/2 | 3971/118 | 150/109 | 11ms | 37ms | $0.0014 | $0.0003 | ✅ Both passing |
| recursion_factorial | Recursive factorial computation | 2/2 | 2/2 | 3918/80 | 98/61 | 18ms | 30ms | $0.0013 | $0.0002 | ✅ Both passing |
| recursion_fibonacci | Recursive Fibonacci computation (compute-intensive) | 1/2 | 2/2 | 3961/97 | 140/67 | 42ms | 30ms | $0.0013 | $0.0002 | ✅ Both passing |
| simple_print | Simple print test - designed to test repair mechanism | 0/0 | 2/2 | 0/0 | 95/19 | 0ms | 28ms | $0 | $0.0001 | ⚠️ Python only (AILANG: N/A) |
| string_manipulation | String concatenation, show(), and comparisons | 2/2 | 2/2 | 4001/118 | 180/84 | 18ms | 21ms | $0.0014 | $0.0002 | ✅ Both passing |
| targeted_repair_test | Targeted test to validate repair mechanism works | 0/2 | 0/0 | 4083/33 | 0/0 | 5ms | 0ms | $0.0012 | $0 | ❌ Both failing |

## Model-by-Model Results

| Model | AILANG Pass Rate | Python Pass Rate | Advantage |
|-------|------------------|------------------|-----------|
| gemini-2-5-flash | 42% (8/19) | 77% (14/18) | +35% for Python |
| gpt5-mini | 42% (8/19) | 84% (16/19) | +42% for Python |

---

*Generated: Wed Oct 15 10:53:26 CEST 2025*
*Source: Automated AI code generation benchmarks*
