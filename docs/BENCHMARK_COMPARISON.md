# AILANG vs Python: AI Code Generation Benchmark Comparison

> Generated from automated evaluation across multiple AI models

## Summary

### Key Performance Indicator

**AILANG vs Python Delta: **-33%** ⚠️ Python advantage**

### Detailed Metrics

| Metric | AILANG | Python | Delta |
|--------|--------|--------|-------|
| Success Rate | 49% (28/57) | 82% (47/57) | -33% |
| Avg Cost per Benchmark | $0.0079 | $0.0018 | 0.01 |
| Models Tested | 4 | 4 | - |
| Benchmarks | 20 | 20 | - |

## Detailed Comparison

| Benchmark | Description | AILANG Success | Python Success | AILANG Tokens (in/out) | Python Tokens (in/out) | AILANG Speed (ms) | Python Speed (ms) | AILANG Cost | Python Cost | Status |
|-----------|-------------|----------------|----------------|------------------------|------------------------|-------------------|-------------------|-------------|-------------|--------|
| adt_option | Option/Maybe monad operations (algebraic data types) | 3/3 | 3/3 | 4166/197 | 170/222 | 13ms | 44ms | $0.0102 | $0.0029 | ✅ Both passing |
| cli_args | Read file from CLI argument, process, write result (IO + FS) | 0/3 | 0/3 | 85/124 | 86/123 | 17ms | 47ms | $0.0016 | $0.0015 | ❌ Both failing |
| error_handling | Result/Either type for error propagation | 0/3 | 3/3 | 4288/630 | 292/417 | 9ms | 60ms | $0.0159 | $0.0053 | ⚠️ Python only (AILANG: compile_error) |
| fizzbuzz | Classic FizzBuzz problem (control flow) | 3/3 | 3/3 | 4138/164 | 142/76 | 14ms | 40ms | $0.0097 | $0.0012 | ✅ Both passing |
| float_eq | N/A | 0/3 | 2/3 | 47/24 | 47/23 | 11ms | 37ms | $0.0004 | $0.0004 | ⚠️ Python only (AILANG: compile_error) |
| higher_order_functions | Compose, map, filter, currying, partial application | 0/3 | 3/3 | 4237/213 | 241/110 | 9ms | 29ms | $0.0105 | $0.0018 | ⚠️ Python only (AILANG: compile_error) |
| json_parse | Parse JSON array, filter, and output (data parsing) | 0/3 | 3/3 | 101/82 | 102/83 | 10ms | 44ms | $0.0012 | $0.0012 | ⚠️ Python only (AILANG: compile_error) |
| list_comprehension | Map/filter/fold operations on lists | 0/3 | 3/3 | 4239/472 | 243/118 | 9ms | 31ms | $0.0135 | $0.0018 | ⚠️ Python only (AILANG: compile_error) |
| list_operations | List construction, pattern matching, and recursion | 0/3 | 3/3 | 4184/219 | 188/193 | 16ms | 32ms | $0.0105 | $0.0025 | ⚠️ Python only (AILANG: logic_error, runtime_error) |
| nested_records | Nested record construction and field access | 3/3 | 2/3 | 4201/120 | 205/142 | 13ms | 47ms | $0.0093 | $0.002 | ✅ Both passing |
| numeric_modulo | N/A | 0/3 | 3/3 | 40/16 | 41/15 | 13ms | 29ms | $0.0003 | $0.0003 | ⚠️ Python only (AILANG: compile_error) |
| pattern_matching_complex | Nested patterns, guards, exhaustiveness checking | 3/3 | 3/3 | 4291/311 | 295/317 | 13ms | 54ms | $0.0117 | $0.0042 | ✅ Both passing |
| pipeline | Read stdin, transform data, write stdout (IO + lists) | 0/3 | 0/3 | 102/41 | 102/67 | 21ms | 38ms | $0.0007 | $0.0009 | ❌ Both failing |
| record_update | Record update syntax {r | field: value} | 2/3 | 1/3 | 4176/184 | 180/164 | 20ms | 51ms | $0.0101 | $0.0022 | ✅ Both passing |
| records_person | Record types with field access and updates | 3/3 | 3/3 | 4150/127 | 154/101 | 14ms | 57ms | $0.0093 | $0.0015 | ✅ Both passing |
| recursion_factorial | Recursive factorial computation | 3/3 | 3/3 | 4098/84 | 102/59 | 14ms | 27ms | $0.0087 | $0.0009 | ✅ Both passing |
| recursion_fibonacci | Recursive Fibonacci computation (compute-intensive) | 3/3 | 3/3 | 4144/92 | 148/69 | 49ms | 26ms | $0.0089 | $0.0011 | ✅ Both passing |
| simple_print | Simple print test - designed to test repair mechanism | 0/0 | 3/3 | 0/0 | 97/21 | 0ms | 32ms | $0 | $0.0004 | ⚠️ Python only (AILANG: N/A) |
| string_manipulation | String concatenation, show(), and comparisons | 3/3 | 3/3 | 4181/122 | 185/84 | 13ms | 24ms | $0.0093 | $0.0013 | ✅ Both passing |
| targeted_repair_test | Targeted test to validate repair mechanism works | 2/3 | 0/0 | 4268/47 | 0/0 | 24ms | 0ms | $0.0086 | $0 | 🏆 AILANG only (Python: N/A) |

## Model-by-Model Results

| Model | AILANG Pass Rate | Python Pass Rate | Advantage |
|-------|------------------|------------------|-----------|
| claude-sonnet-4-5 | 52% (10/19) | 84% (16/19) | +32% for Python |
| gemini-2-5-pro | 52% (10/19) | 78% (15/19) | +26% for Python |
| gpt5 | 42% (8/19) | 84% (16/19) | +42% for Python |

---

*Generated: Wed Oct 15 11:12:24 CEST 2025*
*Source: Automated AI code generation benchmarks*
