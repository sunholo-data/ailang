# internal/eval_analysis

Benchmark analysis and dashboard generation for AILANG evaluation system.

## File Organization

This package was reorganized in November 2025 to comply with the 800-line file size limit for AI-maintainability.

### Core Analysis Files

- **comparison.go** (195 lines) - Baseline comparison and regression detection
- **matrix.go** (242 lines) - Performance matrix generation and aggregation
- **loader.go** (279 lines) - Benchmark result loading and filtering
- **validate.go** (221 lines) - Result validation and health checks
- **formatter.go** (325 lines) - Human-readable output formatting

### Export Files (split from export_docusaurus.go)

The original `export_docusaurus.go` (980 lines) was split into 4 focused files:

- **dashboard_io.go** (145 lines) - Dashboard JSON I/O operations
  - `loadExistingDashboard()` - Load existing dashboard with history
  - `mergeHistory()` - Merge new results into history
  - `buildHistoryEntryFromMatrix()` - Create history entries
  - `writeJSONAtomic()` - Atomic file writes with validation

- **export_json.go** (616 lines) - JSON export for client-side rendering
  - `ExportBenchmarkJSON()` - Main JSON export function
  - Agent vs standard metrics separation
  - Per-language, per-model, per-benchmark breakdowns
  - Fair comparison metrics (agent-comparable benchmarks)

- **export_mdx.go** (208 lines) - MDX export for Docusaurus
  - `ExportDocusaurusMDX()` - Generate React-enhanced markdown
  - Model performance tables
  - Benchmark detail tables
  - Success stories and case studies

- **export_helpers.go** (32 lines) - Shared formatting utilities
  - `formatBenchmarkName()` - Convert snake_case to Title Case
  - `formatModelName()` - Shorten model names for tables

### Chain-Based Loading (v0.8.0+)

- **loader_chains.go** (174 lines) - Load results from observatory.db chains
  - `LoadResultsFromChain(chainID)` - Load all benchmark results from a chain
  - `LoadResultsFromLatestEvalChain()` - Find and load most recent eval_suite chain
  - `LoadBaselineFromChain(chainID)` - Create Baseline for comparisons
  - `stageToResult()` - Convert chain stage + eval assessment to BenchmarkResult

### Data Types

- **types.go** (343 lines) - All data structure definitions
  - `BenchmarkResult` - Single benchmark run result
  - `PerformanceMatrix` - Aggregated performance data
  - `DashboardJSON` - Dashboard structure with history
  - Language, model, and benchmark stats

### Tests

- **comparison_test.go** (295 lines) - Comparison logic tests
- **matrix_test.go** (230 lines) - Matrix generation tests
- **export_docusaurus_test.go** (285 lines) - Dashboard I/O tests
  - History preservation
  - Version deduplication
  - Atomic write validation
  - Rollback on error

## Usage

### Generate Performance Matrix (file-based)

```go
results := LoadResults("eval_results/baselines/v0.4.0")
matrix := GenerateMatrix(results, "v0.4.0")
```

### Generate Performance Matrix (chain-based - v0.8.0+)

```go
results, err := LoadResultsFromChain("e9c7501d-...")
matrix := GenerateMatrix(results, "v0.8.0")
```

### Export Dashboard JSON

```go
jsonStr, err := ExportBenchmarkJSON(matrix, history, results, "docs/static/benchmarks/latest.json")
// Automatically preserves history, validates, and writes atomically
```

### Export Docusaurus MDX

```go
mdx := ExportDocusaurusMDX(matrix, history)
os.WriteFile("docs/docs/benchmarks/performance.md", []byte(mdx), 0644)
```

### Compare Baselines

```go
baseline := LoadBaseline("eval_results/baselines/v0.4.0")
newResults := LoadResults("eval_results/baselines/v0.4.1")
report := Compare(baseline, newResults)
```

### Compare Chain-Based Baselines (v0.8.0+)

```go
baseline, _ := LoadBaselineFromChain("chain-id-1")
newResults, _ := LoadResultsFromChain("chain-id-2")
report := Compare(baseline, newResults)
```

## Design Principles

1. **History Preservation** - Dashboard JSON maintains full version history
2. **Atomic Writes** - All file writes are atomic (temp + rename)
3. **Fair Comparisons** - Agent metrics compare against same benchmark set
4. **Validation** - JSON structure validated before writing
5. **AI-Friendly** - Files kept under 800 lines for AI maintainability

## Recent Changes

**v0.8.0 (February 2026)** - Chain-based result loading
- Added `loader_chains.go` for loading results from observatory.db chains
- Agent eval results now stored as chains (one stage per benchmark)
- `LoadResultsFromChain()` returns same `[]*BenchmarkResult` type as `LoadResults()`
- Entire downstream pipeline (matrix, export, comparison) works unchanged

**v0.4.0 (November 2025)** - File split for AI-maintainability
- Split `export_docusaurus.go` (980 lines) into 4 files (145, 616, 208, 32 lines)
- All files now under 800-line limit
- All tests passing (100% compatibility maintained)
- Zero functional changes - pure refactoring

## See Also

- [M-EVAL-LOOP Design](../../design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md)
- [Evaluation Guide](../../docs/docs/guides/evaluation/README.md)
- [Dashboard JSON Schema](types.go) - See `DashboardJSON` struct
