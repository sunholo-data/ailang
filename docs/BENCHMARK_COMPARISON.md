# AILANG vs Python: AI Code Generation Benchmark Comparison

> Generated from automated evaluation across multiple AI models

## Summary

### Key Performance Indicator

**AILANG vs Python Delta: **-44%** ⚠️ Python advantage**

### Detailed Metrics

| Metric | AILANG | Python | Delta |
|--------|--------|--------|-------|
| Success Rate | 44% (4/9) | 88% (8/9) | -44% |
| Avg Cost per Benchmark | $0.0578 | $0.0084 | 0.05 |
| Models Tested | 1 | 1 | - |
| Benchmarks | 10 | 10 | - |

## Detailed Comparison

| Benchmark | Description | AILANG Success | Python Success | AILANG Tokens (in/out) | Python Tokens (in/out) | AILANG Speed (ms) | Python Speed (ms) | AILANG Cost | Python Cost | Status |
|-----------|-------------|----------------|----------------|------------------------|------------------------|-------------------|-------------------|-------------|-------------|--------|
| adt_option | Option/Maybe monad operations (algebraic data types) | 0/1 | 1/1 | 2586/202 | 270/167 | 16ms | 50ms | $0.0836 | $0.0131 | ⚠️ Python only (AILANG: runtime_error) |
| cli_args | Read file from CLI argument, process, write result (IO + FS) | 0/1 | 0/0 | 91/163 | 0/0 | 17ms | 0ms | $0.0076 | $0 | ❌ Both failing |
| fizzbuzz | Classic FizzBuzz problem (control flow) | 1/1 | 1/1 | 2511/188 | 195/88 | 15ms | 41ms | $0.081 | $0.0085 | ✅ Both passing |
| float_eq | N/A | 0/1 | 1/1 | 50/49 | 51/32 | 7ms | 35ms | $0.003 | $0.0025 | ⚠️ Python only (AILANG: compile_error) |
| json_parse | Parse JSON array, filter, and output (data parsing) | 0/1 | 1/1 | 108/96 | 109/80 | 15ms | 55ms | $0.0061 | $0.0057 | ⚠️ Python only (AILANG: compile_error) |
| numeric_modulo | N/A | 0/1 | 1/1 | 43/41 | 44/18 | 15ms | 42ms | $0.0025 | $0.0019 | ⚠️ Python only (AILANG: compile_error) |
| pipeline | Read stdin, transform data, write stdout (IO + lists) | 0/0 | 0/1 | 0/0 | 110/37 | 0ms | 37ms | $0 | $0.0044 | ❌ Both failing |
| records_person | Record types with field access and updates | 1/1 | 1/1 | 3639/139 | 346/135 | 15ms | 33ms | $0.1133 | $0.0144 | ✅ Both passing |
| recursion_factorial | Recursive factorial computation | 1/1 | 1/1 | 3597/90 | 304/75 | 13ms | 43ms | $0.1106 | $0.0114 | ✅ Both passing |
| recursion_fibonacci | Recursive Fibonacci computation (compute-intensive) | 1/1 | 1/1 | 3649/113 | 356/97 | 39ms | 45ms | $0.1129 | $0.0136 | ✅ Both passing |

## Model-by-Model Results

| Model | AILANG Pass Rate | Python Pass Rate | Advantage |
|-------|------------------|------------------|-----------|
| claude-sonnet-4-5-20250929 | 44% (4/9) | 88% (8/9) | +44% for Python |

---

*Generated: Wed Oct  8 15:21:44 CEST 2025*
*Source: Automated AI code generation benchmarks*
